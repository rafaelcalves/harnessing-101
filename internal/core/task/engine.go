// Package task is the headless task-lifecycle slice of the core: agents,
// tasks, and the Todo/Doing/Blocked/AwaitingReview/Done state machine from
// docs/architecture/boundaries.md. It imports no CLI, GUI, filesystem
// layout, or network client — only ports and domain. Authorization comes
// from the CallerScope the host supplies to every call; nothing here
// trusts a field an agent could have written itself.
package task

import (
	"context"
	"fmt"
	"time"

	"github.com/rafaelcalves/harnessing-101/internal/core/domain"
	"github.com/rafaelcalves/harnessing-101/internal/core/ports"
)

// CallerScope is host-established authorization for one call: which agent
// is calling, and whether the host's policy grants that caller human
// reviewer authority. Nothing here comes from a command payload — an
// agent field can claim anything, and boundaries.md is explicit that
// review authority "comes from host policy, never a reviewerID or
// isHuman field supplied by an agent." A CLI adapter (Phase 2) builds this
// from the operator running the process; this Phase 1 engine takes it as
// given from whoever wires it, which in this slice's tests is the test
// acting as the host.
type CallerScope struct {
	AgentID         domain.AgentID
	IsHumanReviewer bool
}

// Engine implements the task-lifecycle commands against a StateStore. It
// holds no state of its own; every call loads-mutates-commits atomically
// through the store.
type Engine struct {
	store       ports.StateStore
	clock       ports.Clock
	ids         ports.IDSource
	workspaceID domain.WorkspaceID
}

func NewEngine(store ports.StateStore, clock ports.Clock, ids ports.IDSource, workspaceID domain.WorkspaceID) *Engine {
	return &Engine{store: store, clock: clock, ids: ids, workspaceID: workspaceID}
}

// CreateTaskRequest is version 1's CreateTask payload (boundaries.md
// "version 1 command payloads"), plus the request ID envelope field
// every command carries.
type CreateTaskRequest struct {
	RequestID  domain.RequestID
	TaskID     domain.TaskID
	Title      string
	AssigneeID domain.AgentID
}

func (e *Engine) CreateTask(ctx context.Context, caller CallerScope, req CreateTaskRequest) (domain.Receipt, error) {
	if req.Title == "" {
		return domain.Receipt{}, &domain.Error{Code: domain.ErrInvalidArgument, Detail: "title must not be empty"}
	}
	if req.TaskID == "" {
		return domain.Receipt{}, &domain.Error{Code: domain.ErrInvalidArgument, Detail: "taskID must not be empty"}
	}

	fp := fingerprint("CreateTask", req.TaskID, req.Title, req.AssigneeID)
	return e.commit(ctx, caller, req.RequestID, fp, func(snap *domain.Snapshot) ([]domain.Event, error) {
		if _, _, found := findTask(snap, req.TaskID); found {
			return nil, &domain.Error{Code: domain.ErrConflict, Detail: "taskID already exists"}
		}

		now := e.clock.WallNow()
		newTask := domain.Task{
			ID:         req.TaskID,
			Title:      req.Title,
			AssigneeID: req.AssigneeID,
			Status:     domain.TaskTodo,
			Revision:   1,
			Provenance: e.claimedProvenance(caller, now),
		}
		snap.Tasks = append(snap.Tasks, newTask)

		ev, err := e.event(ctx, "TaskCreated", now, string(req.TaskID))
		if err != nil {
			return nil, err
		}
		return []domain.Event{ev}, nil
	})
}

// TransitionTaskRequest is the generic transition command. Only
// Todo->Doing, Doing->Blocked, Blocked->Doing, and an authorized human's
// Done->Todo reopen are permitted here; AwaitingReview and Done are only
// reachable through the dedicated report/decision commands below.
type TransitionTaskRequest struct {
	RequestID  domain.RequestID
	TaskID     domain.TaskID
	FromStatus domain.TaskStatus
	ToStatus   domain.TaskStatus
	Reason     string
}

var genericTransitions = map[domain.TaskStatus]map[domain.TaskStatus]bool{
	domain.TaskTodo:    {domain.TaskDoing: true},
	domain.TaskDoing:   {domain.TaskBlocked: true},
	domain.TaskBlocked: {domain.TaskDoing: true},
	domain.TaskDone:    {domain.TaskTodo: true},
}

func (e *Engine) TransitionTask(ctx context.Context, caller CallerScope, req TransitionTaskRequest) (domain.Receipt, error) {
	allowedTargets, fromKnown := genericTransitions[req.FromStatus]
	if !fromKnown || !allowedTargets[req.ToStatus] {
		return domain.Receipt{}, &domain.Error{
			Code:   domain.ErrInvalidArgument,
			Detail: fmt.Sprintf("transition %s->%s is not permitted by TransitionTask; AwaitingReview and Done are reached only through ReportTaskResult/AcceptTaskResult", req.FromStatus, req.ToStatus),
		}
	}
	if req.ToStatus == domain.TaskBlocked && req.Reason == "" {
		return domain.Receipt{}, &domain.Error{Code: domain.ErrInvalidArgument, Detail: "reason is required to move to Blocked"}
	}
	reopening := req.FromStatus == domain.TaskDone && req.ToStatus == domain.TaskTodo
	if reopening {
		if req.Reason == "" {
			return domain.Receipt{}, &domain.Error{Code: domain.ErrInvalidArgument, Detail: "reason is required to reopen a Done task"}
		}
		if !caller.IsHumanReviewer {
			return domain.Receipt{}, &domain.Error{Code: domain.ErrDenied, Detail: "reopening a Done task requires an authorized human reviewer"}
		}
	}

	fp := fingerprint("TransitionTask", req.TaskID, req.FromStatus, req.ToStatus, req.Reason)
	return e.commit(ctx, caller, req.RequestID, fp, func(snap *domain.Snapshot) ([]domain.Event, error) {
		idx, existing, found := findTask(snap, req.TaskID)
		if !found {
			return nil, &domain.Error{Code: domain.ErrNotFound, Detail: "task not found"}
		}
		if existing.Status != req.FromStatus {
			return nil, &domain.Error{Code: domain.ErrConflict, Detail: "task is no longer in fromStatus"}
		}

		snap.Tasks[idx].Status = req.ToStatus
		snap.Tasks[idx].Revision++
		if reopening {
			snap.Tasks[idx].CurrentResultID = nil
		}

		now := e.clock.WallNow()
		ev, err := e.event(ctx, "TaskTransitioned", now, string(req.TaskID))
		if err != nil {
			return nil, err
		}
		return []domain.Event{ev}, nil
	})
}

// ReportTaskResultRequest binds to the task revision the caller last saw,
// per boundaries.md's completion contract.
type ReportTaskResultRequest struct {
	RequestID            domain.RequestID
	TaskID               domain.TaskID
	ResultID             domain.ResultID
	ExpectedTaskRevision uint64
	Summary              string
	Artifacts            []string
}

func (e *Engine) ReportTaskResult(ctx context.Context, caller CallerScope, req ReportTaskResultRequest) (domain.Receipt, error) {
	if req.Summary == "" {
		return domain.Receipt{}, &domain.Error{Code: domain.ErrInvalidArgument, Detail: "summary must not be empty"}
	}
	if len(req.Artifacts) == 0 {
		return domain.Receipt{}, &domain.Error{Code: domain.ErrInvalidArgument, Detail: "at least one artifact reference is required"}
	}

	fp := fingerprint("ReportTaskResult", req.TaskID, req.ResultID, req.ExpectedTaskRevision, req.Summary, req.Artifacts)
	return e.commit(ctx, caller, req.RequestID, fp, func(snap *domain.Snapshot) ([]domain.Event, error) {
		idx, existing, found := findTask(snap, req.TaskID)
		if !found {
			return nil, &domain.Error{Code: domain.ErrNotFound, Detail: "task not found"}
		}
		if existing.AssigneeID != caller.AgentID {
			return nil, &domain.Error{Code: domain.ErrDenied, Detail: "only the current assignee may report a result"}
		}
		if existing.Revision != req.ExpectedTaskRevision {
			return nil, &domain.Error{Code: domain.ErrConflict, Detail: "expectedTaskRevision is stale"}
		}
		if existing.Status != domain.TaskDoing {
			return nil, &domain.Error{Code: domain.ErrInvalidArgument, Detail: "task is not Doing"}
		}
		if _, _, found := findResult(snap, req.ResultID); found {
			return nil, &domain.Error{Code: domain.ErrConflict, Detail: "resultID already exists"}
		}

		now := e.clock.WallNow()
		result := domain.TaskResult{
			ResultID:   req.ResultID,
			TaskID:     req.TaskID,
			Summary:    req.Summary,
			Artifacts:  append([]string(nil), req.Artifacts...),
			Provenance: e.claimedProvenance(caller, now),
			Decision:   domain.TaskResultPending,
		}
		snap.TaskResults = append(snap.TaskResults, result)

		resultID := req.ResultID
		snap.Tasks[idx].CurrentResultID = &resultID
		snap.Tasks[idx].Status = domain.TaskAwaitingReview
		snap.Tasks[idx].Revision++

		ev, err := e.event(ctx, "TaskResultReported", now, string(req.TaskID), string(req.ResultID))
		if err != nil {
			return nil, err
		}
		return []domain.Event{ev}, nil
	})
}

// AcceptTaskResultRequest and RejectTaskResultRequest bind to both the
// task revision and the exact current result ID: either being stale is
// Conflict, per boundaries.md's completion contract.
type AcceptTaskResultRequest struct {
	RequestID            domain.RequestID
	TaskID               domain.TaskID
	ResultID             domain.ResultID
	ExpectedTaskRevision uint64
	ReviewNote           string
}

type RejectTaskResultRequest struct {
	RequestID            domain.RequestID
	TaskID               domain.TaskID
	ResultID             domain.ResultID
	ExpectedTaskRevision uint64
	Reason               string
}

func (e *Engine) AcceptTaskResult(ctx context.Context, caller CallerScope, req AcceptTaskResultRequest) (domain.Receipt, error) {
	if !caller.IsHumanReviewer {
		return domain.Receipt{}, &domain.Error{Code: domain.ErrDenied, Detail: "accepting a result requires an authorized human reviewer"}
	}

	fp := fingerprint("AcceptTaskResult", req.TaskID, req.ResultID, req.ExpectedTaskRevision, req.ReviewNote)
	return e.commit(ctx, caller, req.RequestID, fp, func(snap *domain.Snapshot) ([]domain.Event, error) {
		taskIdx, _, resultIdx, _, err := loadPendingDecision(snap, req.TaskID, req.ResultID, req.ExpectedTaskRevision)
		if err != nil {
			return nil, err
		}

		now := e.clock.WallNow()
		snap.TaskResults[resultIdx].Decision = domain.TaskResultAccepted
		snap.TaskResults[resultIdx].DecisionProvenance = e.claimedProvenance(caller, now)

		snap.Tasks[taskIdx].Status = domain.TaskDone
		snap.Tasks[taskIdx].Revision++

		ev, err := e.event(ctx, "TaskResultAccepted", now, string(req.TaskID), string(req.ResultID))
		if err != nil {
			return nil, err
		}
		return []domain.Event{ev}, nil
	})
}

func (e *Engine) RejectTaskResult(ctx context.Context, caller CallerScope, req RejectTaskResultRequest) (domain.Receipt, error) {
	if !caller.IsHumanReviewer {
		return domain.Receipt{}, &domain.Error{Code: domain.ErrDenied, Detail: "rejecting a result requires an authorized human reviewer"}
	}
	if req.Reason == "" {
		return domain.Receipt{}, &domain.Error{Code: domain.ErrInvalidArgument, Detail: "reason is required to reject a result"}
	}

	fp := fingerprint("RejectTaskResult", req.TaskID, req.ResultID, req.ExpectedTaskRevision, req.Reason)
	return e.commit(ctx, caller, req.RequestID, fp, func(snap *domain.Snapshot) ([]domain.Event, error) {
		taskIdx, _, resultIdx, _, err := loadPendingDecision(snap, req.TaskID, req.ResultID, req.ExpectedTaskRevision)
		if err != nil {
			return nil, err
		}

		now := e.clock.WallNow()
		snap.TaskResults[resultIdx].Decision = domain.TaskResultRejected
		snap.TaskResults[resultIdx].DecisionReason = req.Reason
		snap.TaskResults[resultIdx].DecisionProvenance = e.claimedProvenance(caller, now)

		snap.Tasks[taskIdx].Status = domain.TaskDoing
		snap.Tasks[taskIdx].Revision++

		ev, err := e.event(ctx, "TaskResultRejected", now, string(req.TaskID), string(req.ResultID))
		if err != nil {
			return nil, err
		}
		return []domain.Event{ev}, nil
	})
}

// GetTask is a read, not a command: no receipt, no idempotency ledger.
func (e *Engine) GetTask(ctx context.Context, taskID domain.TaskID) (domain.Task, error) {
	snap, err := e.store.Load(ctx, e.workspaceID)
	if err != nil {
		return domain.Task{}, err
	}
	_, task, found := findTask(&snap, taskID)
	if !found {
		return domain.Task{}, &domain.Error{Code: domain.ErrNotFound, Detail: "task not found"}
	}
	return task, nil
}

// loadPendingDecision applies the shared precondition checks for Accept
// and Reject: task exists, revision matches, the referenced result is
// exactly the task's current result (a superseded or unknown result ID
// is Conflict, never silently ignored), the task is AwaitingReview, and
// that result has not already been decided.
func loadPendingDecision(snap *domain.Snapshot, taskID domain.TaskID, resultID domain.ResultID, expectedTaskRevision uint64) (taskIdx int, task domain.Task, resultIdx int, result domain.TaskResult, err error) {
	taskIdx, task, found := findTask(snap, taskID)
	if !found {
		return 0, domain.Task{}, 0, domain.TaskResult{}, &domain.Error{Code: domain.ErrNotFound, Detail: "task not found"}
	}
	if task.Revision != expectedTaskRevision {
		return 0, domain.Task{}, 0, domain.TaskResult{}, &domain.Error{Code: domain.ErrConflict, Detail: "expectedTaskRevision is stale"}
	}
	if task.CurrentResultID == nil || *task.CurrentResultID != resultID {
		return 0, domain.Task{}, 0, domain.TaskResult{}, &domain.Error{Code: domain.ErrConflict, Detail: "resultID is not the task's current result"}
	}
	if task.Status != domain.TaskAwaitingReview {
		return 0, domain.Task{}, 0, domain.TaskResult{}, &domain.Error{Code: domain.ErrInvalidArgument, Detail: "task is not AwaitingReview"}
	}

	resultIdx, result, found = findResult(snap, resultID)
	if !found {
		return 0, domain.Task{}, 0, domain.TaskResult{}, &domain.Error{Code: domain.ErrNotFound, Detail: "result not found"}
	}
	if result.Decision != domain.TaskResultPending {
		return 0, domain.Task{}, 0, domain.TaskResult{}, &domain.Error{Code: domain.ErrConflict, Detail: "result already decided"}
	}

	return taskIdx, task, resultIdx, result, nil
}

func findTask(snap *domain.Snapshot, id domain.TaskID) (int, domain.Task, bool) {
	for i, t := range snap.Tasks {
		if t.ID == id {
			return i, t, true
		}
	}
	return 0, domain.Task{}, false
}

func findResult(snap *domain.Snapshot, id domain.ResultID) (int, domain.TaskResult, bool) {
	for i, r := range snap.TaskResults {
		if r.ResultID == id {
			return i, r, true
		}
	}
	return 0, domain.TaskResult{}, false
}

// claimedProvenance builds the Provenance every task/result record
// carries. IdentityVerification is always Unverified in this phase: see
// the type's doc comment on why that is not a gap to "fix" later without
// a real verification mechanism.
func (e *Engine) claimedProvenance(caller CallerScope, now time.Time) domain.Provenance {
	return domain.Provenance{
		ClaimedAgentID:       caller.AgentID,
		EntryMechanism:       domain.EntryMechanismCommand,
		RecordedAt:           now,
		IdentityVerification: domain.IdentityUnverified,
	}
}

func (e *Engine) event(ctx context.Context, kind string, at time.Time, subjects ...string) (domain.Event, error) {
	id, err := e.ids.NewID(ctx)
	if err != nil {
		return domain.Event{}, err
	}
	return domain.Event{
		ID:            domain.EventID(id),
		Kind:          kind,
		SchemaVersion: 1,
		Timestamp:     at,
		SubjectIDs:    subjects,
	}, nil
}

func (e *Engine) commit(ctx context.Context, caller CallerScope, requestID domain.RequestID, fp string, mutate func(*domain.Snapshot) ([]domain.Event, error)) (domain.Receipt, error) {
	receipt, _, err := e.store.Commit(ctx, e.workspaceID, ports.CommitRequest{
		CallerAgentID:      caller.AgentID,
		RequestID:          requestID,
		PayloadFingerprint: fp,
		Mutate:             mutate,
	})
	return receipt, err
}

func fingerprint(parts ...any) string {
	return fmt.Sprintf("%v", parts)
}
