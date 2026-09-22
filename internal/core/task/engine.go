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
	"strconv"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/rafaelcalves/harnessing-101/internal/api"
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
// holds no state of its own beyond the in-process event buffer; every
// command loads-mutates-commits atomically through the store.
//
// commitMu (H101-71, Stanley's second correctness gap) serializes the
// WHOLE store.Commit-then-publish sequence across concurrent callers of
// this Engine. FileStore's own lock only serializes store.Commit calls
// against each other; it says nothing about the order in which their
// CALLERS go on to publish afterward. Without this, goroutine A committing
// revision R and goroutine B committing R+1 can have B's publish (for
// R+1) run before A's (for R) — A returned from store.Commit and was
// simply not yet scheduled to reach events.publish — and eventBus's
// high-water dedup then silently drops R as "already superseded." This
// mutex makes publish order match commit order by construction: whichever
// goroutine's commit-then-publish sequence starts first finishes first,
// because no other goroutine's sequence can interleave with it.
type Engine struct {
	store       ports.StateStore
	clock       ports.Clock
	ids         ports.IDSource
	workspaceID domain.WorkspaceID
	events      *eventBus
	commitMu    sync.Mutex
	supervisor  ports.ProcessSupervisor
}

// SetProcessSupervisor wires this engine's Phase 3 ProcessSupervisor.
// StartRun without one configured returns Unsupported — an honest
// "not configured for this host" rather than a nil-pointer panic.
func (e *Engine) SetProcessSupervisor(supervisor ports.ProcessSupervisor) {
	e.supervisor = supervisor
}

// afterStoreCommitBeforePublish is a package-level test seam (same
// pattern as statestore's fsyncDirFunc): a no-op in production, swapped
// by a white-box test to deterministically pause one goroutine's commit
// right in the gap Stanley named — after its store.Commit returned,
// before its publish — while a second goroutine attempts a concurrent
// commit, to prove commitMu actually blocks the second one rather than
// relying on a race that might not happen to fire.
var afterStoreCommitBeforePublish = func() {}

// NewEngine does not itself seed the event buffer's replay floor: doing
// that here would call store.Load before Open has otherwise touched the
// store, and a workspace that is unreadable for any reason (corrupt
// state, in cmd/harnessing's own IOFailure-rendering test) would then
// fail at construction instead of at the first real command — a
// behavior change to an already-established error-surfacing contract
// this card must not touch. The floor (H101-71, Stanley's first
// correctness gap) is instead seeded lazily, once, on first real use —
// see eventBus.ensureFloor and its callers in commit and Subscribe.
func NewEngine(store ports.StateStore, clock ports.Clock, ids ports.IDSource, workspaceID domain.WorkspaceID) *Engine {
	return &Engine{store: store, clock: clock, ids: ids, workspaceID: workspaceID, events: newEventBus()}
}

// Subscribe is inbound port 3 (ports.StateEvents), promoted onto Engine
// (H101-61): observe committed events from a cursor — typically
// GetSnapshot's own Cursor — without losing anything committed after it.
// See eventBus's doc comment for the no-lost-commits argument this rests
// on, and note its actual limit: retention is this process's lifetime
// only, so a cursor from a process that is no longer running comes back
// CursorExpired, not silently empty.
func (e *Engine) Subscribe(ctx context.Context, afterCursor string) (<-chan domain.Event, error) {
	if err := e.events.ensureFloor(ctx, e.currentRevision); err != nil {
		return nil, err
	}
	return e.events.subscribe(ctx, afterCursor)
}

// currentRevision is the eventBus's floor-seeding callback: the
// workspace's revision as of right now, per the store. Subscribe needs
// this to answer honestly; a failure here is reported directly rather
// than defaulting to a wrong floor.
func (e *Engine) currentRevision(ctx context.Context) (uint64, error) {
	snap, err := e.store.Load(ctx, e.workspaceID)
	if err != nil {
		return 0, err
	}
	return snap.Revision, nil
}

// CreateTaskRequest is version 1's CreateTask payload (boundaries.md
// "version 1 command payloads"), plus the request ID envelope field
// every command carries.
type CreateTaskRequest = api.CreateTaskRequest

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
		if req.AssigneeID != "" {
			if _, _, found := findAgent(snap, req.AssigneeID); !found {
				return nil, &domain.Error{Code: domain.ErrNotFound, Detail: "assignee is not a registered agent"}
			}
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
		newTask.LastStatusChange = e.statusChange(caller, now, newTask.Revision, nil, domain.TaskTodo)
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
type TransitionTaskRequest = api.TransitionTaskRequest

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
		from := req.FromStatus
		snap.Tasks[idx].LastStatusChange = e.statusChange(caller, now, snap.Tasks[idx].Revision, &from, req.ToStatus)
		ev, err := e.event(ctx, "TaskTransitioned", now, string(req.TaskID))
		if err != nil {
			return nil, err
		}
		return []domain.Event{ev}, nil
	})
}

// ReportTaskResultRequest binds to the task revision the caller last saw,
// per boundaries.md's completion contract.
type ReportTaskResultRequest = api.ReportTaskResultRequest

func (e *Engine) ReportTaskResult(ctx context.Context, caller CallerScope, req ReportTaskResultRequest) (domain.Receipt, error) {
	if req.ResultID == "" {
		return domain.Receipt{}, &domain.Error{Code: domain.ErrInvalidArgument, Detail: "resultID must not be empty"}
	}
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
		fromStatus := existing.Status
		snap.Tasks[idx].Status = domain.TaskAwaitingReview
		snap.Tasks[idx].Revision++
		snap.Tasks[idx].LastStatusChange = e.statusChange(caller, now, snap.Tasks[idx].Revision, &fromStatus, domain.TaskAwaitingReview)

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
type AcceptTaskResultRequest = api.AcceptTaskResultRequest

type RejectTaskResultRequest = api.RejectTaskResultRequest

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

		fromStatus := domain.TaskAwaitingReview
		snap.Tasks[taskIdx].Status = domain.TaskDone
		snap.Tasks[taskIdx].Revision++
		snap.Tasks[taskIdx].LastStatusChange = e.statusChange(caller, now, snap.Tasks[taskIdx].Revision, &fromStatus, domain.TaskDone)

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

		fromStatus := domain.TaskAwaitingReview
		snap.Tasks[taskIdx].Status = domain.TaskDoing
		snap.Tasks[taskIdx].Revision++
		snap.Tasks[taskIdx].LastStatusChange = e.statusChange(caller, now, snap.Tasks[taskIdx].Revision, &fromStatus, domain.TaskDoing)

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

// GetSnapshot returns one detached, workspace-wide view. The store's Load is
// the consistency boundary; cloning here protects nested slices and pointers.
func (e *Engine) GetSnapshot(ctx context.Context) (domain.Snapshot, error) {
	snap, err := e.store.Load(ctx, e.workspaceID)
	if err != nil {
		return domain.Snapshot{}, err
	}
	if snap.WorkspaceID == "" {
		snap.WorkspaceID = e.workspaceID
	}
	if snap.Cursor == "" {
		snap.Cursor = strconv.FormatUint(snap.Revision, 10)
	}
	return cloneSnapshot(snap), nil
}

// cloneStatusChange deep-copies H101-70's latest-status record, including
// its own FromStatus pointer, so a query result can never let a caller
// reach — or corrupt — the stored record via a shared pointer.
func cloneStatusChange(sc *domain.StatusChange) *domain.StatusChange {
	if sc == nil {
		return nil
	}
	out := *sc
	if sc.FromStatus != nil {
		from := *sc.FromStatus
		out.FromStatus = &from
	}
	return &out
}

func cloneSnapshot(snap domain.Snapshot) domain.Snapshot {
	out := snap
	out.Agents = append([]domain.Agent{}, snap.Agents...)
	for i := range out.Agents {
		if snap.Agents[i].LastUpdatedProvenance != nil {
			p := *snap.Agents[i].LastUpdatedProvenance
			out.Agents[i].LastUpdatedProvenance = &p
		}
	}
	out.Tasks = append([]domain.Task{}, snap.Tasks...)
	for i := range out.Tasks {
		if snap.Tasks[i].CurrentResultID != nil {
			id := *snap.Tasks[i].CurrentResultID
			out.Tasks[i].CurrentResultID = &id
		}
		out.Tasks[i].LastStatusChange = cloneStatusChange(snap.Tasks[i].LastStatusChange)
	}
	out.TaskResults = append([]domain.TaskResult{}, snap.TaskResults...)
	for i := range out.TaskResults {
		out.TaskResults[i].Artifacts = append([]string{}, snap.TaskResults[i].Artifacts...)
	}
	out.Messages = append([]domain.Message{}, snap.Messages...)
	for i := range out.Messages {
		if snap.Messages[i].TaskID != nil {
			id := *snap.Messages[i].TaskID
			out.Messages[i].TaskID = &id
		}
		if snap.Messages[i].ReplyToMessageID != nil {
			id := *snap.Messages[i].ReplyToMessageID
			out.Messages[i].ReplyToMessageID = &id
		}
		if snap.Messages[i].QueuedAt != nil {
			v := *snap.Messages[i].QueuedAt
			out.Messages[i].QueuedAt = &v
		}
		if snap.Messages[i].PublishedAt != nil {
			v := *snap.Messages[i].PublishedAt
			out.Messages[i].PublishedAt = &v
		}
		if snap.Messages[i].ProcessedAt != nil {
			v := *snap.Messages[i].ProcessedAt
			out.Messages[i].ProcessedAt = &v
		}
		if snap.Messages[i].AcknowledgedAt != nil {
			v := *snap.Messages[i].AcknowledgedAt
			out.Messages[i].AcknowledgedAt = &v
		}
	}
	out.Profiles = append([]domain.Profile{}, snap.Profiles...)
	for i := range out.Profiles {
		out.Profiles[i].Spec.Args = append([]string{}, snap.Profiles[i].Spec.Args...)
		out.Profiles[i].Spec.EnvironmentRefs = append([]string{}, snap.Profiles[i].Spec.EnvironmentRefs...)
	}
	out.Runs = append([]domain.Run{}, snap.Runs...)
	for i := range out.Runs {
		if snap.Runs[i].DispatchAttemptedAt != nil {
			v := *snap.Runs[i].DispatchAttemptedAt
			out.Runs[i].DispatchAttemptedAt = &v
		}
		if snap.Runs[i].StartedAt != nil {
			v := *snap.Runs[i].StartedAt
			out.Runs[i].StartedAt = &v
		}
		if snap.Runs[i].ExitedAt != nil {
			v := *snap.Runs[i].ExitedAt
			out.Runs[i].ExitedAt = &v
		}
	}
	return out
}

// RegisterAgentRequest is version 1's RegisterAgent payload (boundaries.md
// line 24): agentID and displayName are required, profileID is optional.
type RegisterAgentRequest = api.RegisterAgentRequest

// UpdateAgentRequest is "an explicit patch of those mutable fields"
// (boundaries.md line 24): DisplayName/ProfileID are nil-means-unchanged
// pointers. AgentID identifies which record to patch; there is no field
// anywhere in this struct that could write a new ID onto an existing
// Agent — the immutable field is unpatchable by construction, not by
// convention.
type UpdateAgentRequest = api.UpdateAgentRequest

// canWriteAgentRecord decides RegisterAgent/UpdateAgent authorization the
// same way for both: self (an agent registering or updating itself) or a
// host-authorized human reviewer (the operator doing initial roster
// setup, per definition.md "the user can ... register named agents").
// Any other caller acting on a different agent's record is Denied — the
// same class of rule as AcknowledgeMessage's recipient-only check, per
// Kelly's H101-21 comparison. This reuses the two authorization
// primitives the engine already has (self-scope, human-reviewer
// authority) rather than inventing a third.
func canWriteAgentRecord(caller CallerScope, target domain.AgentID) bool {
	return caller.IsHumanReviewer || caller.AgentID == target
}

func (e *Engine) RegisterAgent(ctx context.Context, caller CallerScope, req RegisterAgentRequest) (domain.Receipt, error) {
	if req.AgentID == "" {
		return domain.Receipt{}, &domain.Error{Code: domain.ErrInvalidArgument, Detail: "agentID must not be empty"}
	}
	if req.DisplayName == "" {
		return domain.Receipt{}, &domain.Error{Code: domain.ErrInvalidArgument, Detail: "displayName must not be empty"}
	}
	if !canWriteAgentRecord(caller, req.AgentID) {
		return domain.Receipt{}, &domain.Error{Code: domain.ErrDenied, Detail: "only the agent itself or an authorized human reviewer may register this agentID"}
	}

	fp := fingerprint("RegisterAgent", req.AgentID, req.DisplayName, req.ProfileID)
	return e.commit(ctx, caller, req.RequestID, fp, func(snap *domain.Snapshot) ([]domain.Event, error) {
		if _, _, found := findAgent(snap, req.AgentID); found {
			// Unconditional, matching CreateTask's precedent for an existing
			// ID: idempotency is the store's job (same request ID, same
			// payload replays the receipt); a second independent request
			// against an ID that already exists is always Conflict, not a
			// second implicit rule about matching payloads.
			return nil, &domain.Error{Code: domain.ErrConflict, Detail: "agentID already registered"}
		}

		now := e.clock.WallNow()
		snap.Agents = append(snap.Agents, domain.Agent{
			ID:          req.AgentID,
			DisplayName: req.DisplayName,
			ProfileID:   req.ProfileID,
			Provenance:  e.claimedProvenance(caller, now),
		})

		ev, err := e.event(ctx, "AgentRegistered", now, string(req.AgentID))
		if err != nil {
			return nil, err
		}
		return []domain.Event{ev}, nil
	})
}

func (e *Engine) UpdateAgent(ctx context.Context, caller CallerScope, req UpdateAgentRequest) (domain.Receipt, error) {
	if req.AgentID == "" {
		return domain.Receipt{}, &domain.Error{Code: domain.ErrInvalidArgument, Detail: "agentID must not be empty"}
	}
	if req.DisplayName == nil && req.ProfileID == nil {
		return domain.Receipt{}, &domain.Error{Code: domain.ErrInvalidArgument, Detail: "patch must set at least one of displayName or profileID"}
	}
	if req.DisplayName != nil && *req.DisplayName == "" {
		return domain.Receipt{}, &domain.Error{Code: domain.ErrInvalidArgument, Detail: "displayName, if patched, must not be empty"}
	}
	if !canWriteAgentRecord(caller, req.AgentID) {
		return domain.Receipt{}, &domain.Error{Code: domain.ErrDenied, Detail: "only the agent itself or an authorized human reviewer may update this agentID"}
	}

	fp := fingerprint("UpdateAgent", req.AgentID, req.DisplayName, req.ProfileID)
	return e.commit(ctx, caller, req.RequestID, fp, func(snap *domain.Snapshot) ([]domain.Event, error) {
		idx, _, found := findAgent(snap, req.AgentID)
		if !found {
			return nil, &domain.Error{Code: domain.ErrNotFound, Detail: "agent not found"}
		}

		if req.DisplayName != nil {
			snap.Agents[idx].DisplayName = *req.DisplayName
		}
		if req.ProfileID != nil {
			snap.Agents[idx].ProfileID = *req.ProfileID
		}
		now := e.clock.WallNow()
		provenance := e.claimedProvenance(caller, now)
		snap.Agents[idx].LastUpdatedProvenance = &provenance

		ev, err := e.event(ctx, "AgentUpdated", now, string(req.AgentID))
		if err != nil {
			return nil, err
		}
		return []domain.Event{ev}, nil
	})
}

// GetAgent is a read, not a command: no receipt, no idempotency ledger.
func (e *Engine) GetAgent(ctx context.Context, agentID domain.AgentID) (domain.Agent, error) {
	snap, err := e.store.Load(ctx, e.workspaceID)
	if err != nil {
		return domain.Agent{}, err
	}
	_, agent, found := findAgent(&snap, agentID)
	if !found {
		return domain.Agent{}, &domain.Error{Code: domain.ErrNotFound, Detail: "agent not found"}
	}
	return agent, nil
}

// ResolveRequest is ADR 0004's caller-bound resolution operation,
// promoted from the store onto the engine (H101-64): it looks up a
// previously submitted command by (callerAgentID, requestID), performs
// no mutation, allocates no new request ID, and does not advance the
// workspace revision. It takes a plain domain.AgentID rather than a full
// CallerScope on purpose — this is not an authority decision (no
// IsHumanReviewer check applies to reading your own receipt), so it
// does not carry a field this operation has no use for. callerAgentID
// scopes which receipts are reachable; it is not, by itself, proof of
// who is asking — see ports.StateStore.ResolveRequest's doc comment.
// Absent (no receipt found now) and "never applied" are different
// claims; this returns NotFound for the former and never asserts the
// latter.
func (e *Engine) ResolveRequest(ctx context.Context, callerAgentID domain.AgentID, requestID domain.RequestID) (domain.Receipt, error) {
	return e.store.ResolveRequest(ctx, e.workspaceID, callerAgentID, requestID)
}

func findAgent(snap *domain.Snapshot, id domain.AgentID) (int, domain.Agent, bool) {
	for i, a := range snap.Agents {
		if a.ID == id {
			return i, a, true
		}
	}
	return 0, domain.Agent{}, false
}

// SendMessageRequest is the version 1 message command. SenderAgentID is the
// message's claimed sender; Provenance separately records the host-scoped
// caller and remains Unverified in this phase.
type SendMessageRequest = api.SendMessageRequest

func (e *Engine) SendMessage(ctx context.Context, caller CallerScope, req SendMessageRequest) (domain.Receipt, error) {
	if req.MessageID == "" || req.SenderAgentID == "" || req.RecipientAgentID == "" {
		return domain.Receipt{}, &domain.Error{Code: domain.ErrInvalidArgument, Detail: "messageID, senderAgentID, and recipientAgentID are required"}
	}
	if !validMessageKind(req.Kind) {
		return domain.Receipt{}, &domain.Error{Code: domain.ErrInvalidArgument, Detail: "kind must be Request, Inform, or Result"}
	}
	if !utf8.ValidString(req.Body) {
		return domain.Receipt{}, &domain.Error{Code: domain.ErrInvalidArgument, Detail: "body must be valid UTF-8"}
	}

	fp := fingerprint("SendMessage", req.MessageID, req.SenderAgentID, req.RecipientAgentID, req.Kind, req.Body, req.TaskID, req.ReplyToMessageID)
	return e.commit(ctx, caller, req.RequestID, fp, func(snap *domain.Snapshot) ([]domain.Event, error) {
		if _, _, found := findMessage(snap, req.MessageID); found {
			return nil, &domain.Error{Code: domain.ErrConflict, Detail: "messageID already exists"}
		}
		if _, _, found := findAgent(snap, req.SenderAgentID); !found {
			return nil, &domain.Error{Code: domain.ErrNotFound, Detail: "sender is not a registered agent"}
		}
		if _, _, found := findAgent(snap, req.RecipientAgentID); !found {
			return nil, &domain.Error{Code: domain.ErrNotFound, Detail: "recipient is not a registered agent"}
		}
		if req.TaskID != nil {
			if _, _, found := findTask(snap, *req.TaskID); !found {
				return nil, &domain.Error{Code: domain.ErrNotFound, Detail: "task not found"}
			}
		}
		if req.ReplyToMessageID != nil {
			if _, _, found := findMessage(snap, *req.ReplyToMessageID); !found {
				return nil, &domain.Error{Code: domain.ErrNotFound, Detail: "reply-to message not found"}
			}
		}

		now := e.clock.WallNow()
		queuedAt := now
		message := domain.Message{
			WorkspaceID:      e.workspaceID,
			MessageID:        req.MessageID,
			SenderAgentID:    req.SenderAgentID,
			RecipientAgentID: req.RecipientAgentID,
			Kind:             req.Kind,
			Body:             req.Body,
			CreatedAt:        now,
			TaskID:           cloneID(req.TaskID),
			ReplyToMessageID: cloneID(req.ReplyToMessageID),
			Provenance:       e.claimedProvenance(caller, now),
			QueuedAt:         &queuedAt,
		}
		snap.Messages = append(snap.Messages, message)
		ev, err := e.event(ctx, "MessageQueued", now, string(req.MessageID))
		if err != nil {
			return nil, err
		}
		return []domain.Event{ev}, nil
	})
}

// AcknowledgeMessageRequest records the addressed recipient's explicit
// acknowledgement. Repeating it is a successful durable no-op with no second
// acknowledgement event.
type AcknowledgeMessageRequest = api.AcknowledgeMessageRequest

func (e *Engine) AcknowledgeMessage(ctx context.Context, caller CallerScope, req AcknowledgeMessageRequest) (domain.Receipt, error) {
	fp := fingerprint("AcknowledgeMessage", req.MessageID)
	return e.commit(ctx, caller, req.RequestID, fp, func(snap *domain.Snapshot) ([]domain.Event, error) {
		idx, message, found := findMessage(snap, req.MessageID)
		if !found {
			return nil, &domain.Error{Code: domain.ErrNotFound, Detail: "message not found"}
		}
		if message.RecipientAgentID != caller.AgentID {
			return nil, &domain.Error{Code: domain.ErrDenied, Detail: "only the message recipient may acknowledge it"}
		}
		if message.AcknowledgedAt != nil {
			return nil, nil
		}

		now := e.clock.WallNow()
		snap.Messages[idx].AcknowledgedBy = caller.AgentID
		snap.Messages[idx].AcknowledgedAt = &now
		ev, err := e.event(ctx, "MessageAcknowledged", now, string(req.MessageID))
		if err != nil {
			return nil, err
		}
		return []domain.Event{ev}, nil
	})
}

// ApproveProfileRequest is the host-established human decision that
// makes a profileID eligible for StartRun.
type ApproveProfileRequest = api.ApproveProfileRequest

func (e *Engine) ApproveProfile(ctx context.Context, caller CallerScope, req ApproveProfileRequest) (domain.Receipt, error) {
	if !caller.IsHumanReviewer {
		return domain.Receipt{}, &domain.Error{Code: domain.ErrDenied, Detail: "approving a profile requires an authorized human reviewer"}
	}
	if req.ProfileID == "" {
		return domain.Receipt{}, &domain.Error{Code: domain.ErrInvalidArgument, Detail: "profileID must not be empty"}
	}
	if len(req.Spec.Args) == 0 {
		return domain.Receipt{}, &domain.Error{Code: domain.ErrInvalidArgument, Detail: "spec.Args must name at least the executable"}
	}

	fp := fingerprint("ApproveProfile", req.ProfileID, req.Spec)
	return e.commit(ctx, caller, req.RequestID, fp, func(snap *domain.Snapshot) ([]domain.Event, error) {
		if _, found := findProfile(snap, req.ProfileID); found {
			return nil, &domain.Error{Code: domain.ErrConflict, Detail: "profileID already approved"}
		}
		now := e.clock.WallNow()
		spec := req.Spec
		spec.ProfileID = req.ProfileID
		snap.Profiles = append(snap.Profiles, domain.Profile{
			ProfileID:  req.ProfileID,
			Spec:       spec,
			Provenance: e.claimedProvenance(caller, now),
		})
		ev, err := e.event(ctx, "ProfileApproved", now, req.ProfileID)
		if err != nil {
			return nil, err
		}
		return []domain.Event{ev}, nil
	})
}

// afterDispatchMarkerBeforeStart is a package-level test seam (same
// pattern as afterStoreCommitBeforePublish): a no-op in production,
// swapped by a white-box test to simulate this whole process crashing
// in exactly the window H101-135/H101-128 name ambiguous — after the
// dispatch-attempted marker committed durably, before
// ProcessSupervisor.Start is ever called. A test that panics here,
// recovers, then opens a FRESH Engine against the same store proves
// StartRun refuses to auto-redispatch that run, rather than relying on
// a race that might never fire.
var afterDispatchMarkerBeforeStart = func() {}

// StartRunRequest is version 1's StartRun payload.
type StartRunRequest = api.StartRunRequest

// StartRun implements H101-135/H101-128's numbered dispatch-ordering
// invariant as three separate, durably-confirmed commits around exactly
// one ProcessSupervisor.Start call:
//
//  1. persist start intent (this run, Starting) — durable, receipted;
//  2. the single dispatcher (this call, and only this call, ever calls
//     Start for a given run) rechecks state/profile and commits the
//     DISPATCH-ATTEMPTED MARKER before invoking the supervisor,
//     confirming that marker's durability too;
//  3. call Start exactly once, outside any commit callback;
//  4. commit the observation separately from the marker and from the
//     Start call itself.
//
// A crash between step 2 and step 3 leaves the run marker-committed but
// unobserved — the invariant's named ambiguous state. This method
// never auto-redispatches a run already in that state; only explicit
// recovery (item 4, not this card) resolves it.
func (e *Engine) StartRun(ctx context.Context, caller CallerScope, req StartRunRequest) (domain.Receipt, error) {
	if req.RunID == "" || req.AgentID == "" || req.ProfileID == "" {
		return domain.Receipt{}, &domain.Error{Code: domain.ErrInvalidArgument, Detail: "runID, agentID, and profileID are required"}
	}
	if e.supervisor == nil {
		return domain.Receipt{}, &domain.Error{Code: domain.ErrUnsupported, Detail: "no process supervisor is configured for this host"}
	}
	if req.ToolExecutable != "" && len(req.ToolArgv) == 0 {
		// Nothing actually requires this pairing structurally, but an
		// executable with no argv at all is never what a real agentic
		// CLI invocation looks like — catch the likely-wrong call shape
		// early rather than silently launching a bare binary.
		return domain.Receipt{}, &domain.Error{Code: domain.ErrInvalidArgument, Detail: "toolExecutable requires a non-empty toolArgv"}
	}

	// Step 1 — persist start intent. CF1: authorization is `caller`
	// (host-established scope), never req.AgentID/req.ProfileID
	// themselves — those only select which agent/profile to check.
	fpA := fingerprint("StartRun", req.RunID, req.AgentID, req.ProfileID, req.ToolExecutable, req.ToolArgv)
	receipt, err := e.commit(ctx, caller, req.RequestID, fpA, func(snap *domain.Snapshot) ([]domain.Event, error) {
		if _, _, found := findRun(snap, domain.RunID(req.RunID)); found {
			return nil, &domain.Error{Code: domain.ErrConflict, Detail: "runID already exists"}
		}
		profile, found := findProfile(snap, req.ProfileID)
		if !found {
			// D1/R5: an installed executable on PATH is not an approved
			// profile. This IS the approval gate threat-model rule 5
			// requires — Denied, not silent success.
			return nil, &domain.Error{Code: domain.ErrDenied, Detail: "profileID has no recorded approval event"}
		}
		if req.ToolExecutable != "" && (len(profile.Spec.Args) == 0 || req.ToolExecutable != profile.Spec.Args[0]) {
			return nil, &domain.Error{Code: domain.ErrDenied, Detail: "toolExecutable does not match the profile's approved executable"}
		}
		if _, _, found := findAgent(snap, req.AgentID); !found {
			return nil, &domain.Error{Code: domain.ErrNotFound, Detail: "agent is not registered"}
		}
		now := e.clock.WallNow()
		snap.Runs = append(snap.Runs, domain.Run{
			ID:         domain.RunID(req.RunID),
			AgentID:    req.AgentID,
			ProfileID:  req.ProfileID,
			State:      domain.RunStarting,
			Revision:   1,
			Provenance: e.claimedProvenance(caller, now),
		})
		ev, err := e.event(ctx, "RunStarting", now, req.RunID)
		if err != nil {
			return nil, err
		}
		return []domain.Event{ev}, nil
	})
	if err != nil {
		return receipt, err
	}

	// Step 2 — the dispatch-attempted marker, committed before Start is
	// ever called. This commit is host-internal bookkeeping, not a
	// second caller-facing replay slot: it uses a fresh internal
	// request ID under CallerScope{}, the same pattern
	// recordDeliveryFact already uses for host-authored facts.
	var spec domain.ExecutionSpec
	markerErr := func() error {
		markerID, err := e.ids.NewID(ctx)
		if err != nil {
			return err
		}
		fpB := fingerprint("StartRunMarker", req.RunID)
		_, err = e.commit(ctx, CallerScope{}, domain.RequestID(markerID), fpB, func(snap *domain.Snapshot) ([]domain.Event, error) {
			idx, run, found := findRun(snap, domain.RunID(req.RunID))
			if !found {
				return nil, &domain.Error{Code: domain.ErrConflict, Detail: "run vanished between intent and dispatch"}
			}
			if run.DispatchAttemptedAt != nil {
				return nil, &domain.Error{Code: domain.ErrConflict, Detail: "dispatch already attempted for this run; ambiguous outcome, will not auto-redispatch (see item 4 recovery)"}
			}
			if run.State != domain.RunStarting {
				return nil, &domain.Error{Code: domain.ErrConflict, Detail: "run is no longer Starting"}
			}
			profile, found := findProfile(snap, run.ProfileID)
			if !found {
				return nil, &domain.Error{Code: domain.ErrDenied, Detail: "profileID approval no longer present"}
			}
			spec = profile.Spec
			if req.ToolExecutable != "" {
				spec.Args = append([]string{req.ToolExecutable}, req.ToolArgv...)
			}

			now := e.clock.WallNow()
			snap.Runs[idx].DispatchAttemptedAt = &now
			ev, err := e.event(ctx, "RunDispatchAttempted", now, req.RunID)
			if err != nil {
				return nil, err
			}
			return []domain.Event{ev}, nil
		})
		return err
	}()
	if markerErr != nil {
		return domain.Receipt{}, markerErr
	}

	afterDispatchMarkerBeforeStart()

	// Step 3 — exactly one Start call, outside any commit callback
	// (boundaries: "keep process effects outside Commit callbacks").
	participation := domain.RunParticipationContext{
		WorkspaceID: e.workspaceID,
		RunID:       domain.RunID(req.RunID),
		AgentID:     req.AgentID,
		TaskID:      req.TaskID,
		PeerAgentID: req.PeerAgentID,
	}
	startErr := e.supervisor.Start(ctx, domain.RunID(req.RunID), spec, participation)

	// Step 4 — observation, committed separately from the marker and
	// from the Start call. A Start error is known information, not
	// ambiguous: record it honestly (R3) rather than leaving the run
	// Starting forever.
	observeErr := func() error {
		outcomeID, err := e.ids.NewID(ctx)
		if err != nil {
			return err
		}
		fpC := fingerprint("StartRunObservation", req.RunID, startErr != nil)
		_, err = e.commit(ctx, CallerScope{}, domain.RequestID(outcomeID), fpC, func(snap *domain.Snapshot) ([]domain.Event, error) {
			idx, _, found := findRun(snap, domain.RunID(req.RunID))
			if !found {
				return nil, &domain.Error{Code: domain.ErrConflict, Detail: "run vanished before observation"}
			}
			now := e.clock.WallNow()
			if startErr != nil {
				snap.Runs[idx].State = domain.RunExited
				snap.Runs[idx].ExitedAt = &now
				snap.Runs[idx].ExitReason = startErrorCode(startErr)
				ev, err := e.event(ctx, "RunExited", now, req.RunID)
				if err != nil {
					return nil, err
				}
				return []domain.Event{ev}, nil
			}
			snap.Runs[idx].State = domain.RunRunning
			snap.Runs[idx].StartedAt = &now
			ev, err := e.event(ctx, "RunStarted", now, req.RunID)
			if err != nil {
				return nil, err
			}
			return []domain.Event{ev}, nil
		})
		return err
	}()
	if observeErr != nil {
		return domain.Receipt{}, observeErr
	}
	if startErr != nil {
		return domain.Receipt{}, startErr
	}
	return receipt, nil
}

func startErrorCode(err error) string {
	var derr *domain.Error
	if de, ok := err.(*domain.Error); ok {
		derr = de
		return string(derr.Code)
	}
	return string(domain.ErrSpawnFailed)
}

// GetRun is a read, not a command: no receipt, no idempotency ledger.
func (e *Engine) GetRun(ctx context.Context, runID domain.RunID) (domain.Run, error) {
	snap, err := e.store.Load(ctx, e.workspaceID)
	if err != nil {
		return domain.Run{}, err
	}
	_, run, found := findRun(&snap, runID)
	if !found {
		return domain.Run{}, &domain.Error{Code: domain.ErrNotFound, Detail: "run not found"}
	}
	return run, nil
}

func findProfile(snap *domain.Snapshot, id string) (domain.Profile, bool) {
	for _, p := range snap.Profiles {
		if p.ProfileID == id {
			return p, true
		}
	}
	return domain.Profile{}, false
}

func findRun(snap *domain.Snapshot, id domain.RunID) (int, domain.Run, bool) {
	for i, r := range snap.Runs {
		if r.ID == id {
			return i, r, true
		}
	}
	return 0, domain.Run{}, false
}

// MessageDeliveryRequest identifies a delivery fact recorded by the host.
type MessageDeliveryRequest = api.MessageDeliveryRequest

// RecordMessagePublished records the mailbox adapter making a complete
// envelope available. It is deliberately separate from queued and
// acknowledged: publication does not mean the recipient saw or accepted it.
func (e *Engine) RecordMessagePublished(ctx context.Context, req MessageDeliveryRequest) (domain.Receipt, error) {
	return e.recordDeliveryFact(ctx, req, "MessagePublished", func(message *domain.Message, now time.Time) bool {
		if message.PublishedAt != nil {
			return false
		}
		message.PublishedAt = &now
		return true
	})
}

// RecordMessageProcessed records controller ingestion separately from
// publication and recipient acknowledgement.
func (e *Engine) RecordMessageProcessed(ctx context.Context, req MessageDeliveryRequest) (domain.Receipt, error) {
	return e.recordDeliveryFact(ctx, req, "MessageProcessed", func(message *domain.Message, now time.Time) bool {
		if message.ProcessedAt != nil {
			return false
		}
		message.ProcessedAt = &now
		return true
	})
}

func (e *Engine) recordDeliveryFact(ctx context.Context, req MessageDeliveryRequest, eventKind string, apply func(*domain.Message, time.Time) bool) (domain.Receipt, error) {
	fp := fingerprint(eventKind, req.MessageID)
	return e.commit(ctx, CallerScope{}, req.RequestID, fp, func(snap *domain.Snapshot) ([]domain.Event, error) {
		idx, message, found := findMessage(snap, req.MessageID)
		if !found {
			return nil, &domain.Error{Code: domain.ErrNotFound, Detail: "message not found"}
		}
		now := e.clock.WallNow()
		if !apply(&message, now) {
			return nil, nil
		}
		snap.Messages[idx] = message
		ev, err := e.event(ctx, eventKind, now, string(req.MessageID))
		if err != nil {
			return nil, err
		}
		return []domain.Event{ev}, nil
	})
}

// GetMessage returns all four delivery facts without collapsing them into a
// status. The host/mailbox integration records publication and processing via
// the explicit delivery-fact methods above.
func (e *Engine) GetMessage(ctx context.Context, messageID domain.MessageID) (domain.Message, error) {
	snap, err := e.store.Load(ctx, e.workspaceID)
	if err != nil {
		return domain.Message{}, err
	}
	_, message, found := findMessage(&snap, messageID)
	if !found {
		return domain.Message{}, &domain.Error{Code: domain.ErrNotFound, Detail: "message not found"}
	}
	return message, nil
}

func validMessageKind(kind domain.MessageKind) bool {
	return kind == domain.MessageRequest || kind == domain.MessageInform || kind == domain.MessageResult
}

func findMessage(snap *domain.Snapshot, id domain.MessageID) (int, domain.Message, bool) {
	for i, message := range snap.Messages {
		if message.MessageID == id {
			return i, message, true
		}
	}
	return 0, domain.Message{}, false
}

func cloneID[T ~string](id *T) *T {
	if id == nil {
		return nil
	}
	copy := *id
	return &copy
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
// statusChange builds one H101-70 latest-status record. from is nil only
// for creation; every real transition after that passes the status the
// task was actually leaving. Callers set this on the SAME task struct
// whose Revision they just advanced, inside the same Mutate closure, so
// it lands in the same commit as the change it describes.
func (e *Engine) statusChange(caller CallerScope, now time.Time, taskRevision uint64, from *domain.TaskStatus, to domain.TaskStatus) *domain.StatusChange {
	return &domain.StatusChange{
		TaskRevision: taskRevision,
		FromStatus:   from,
		ToStatus:     to,
		Provenance:   e.claimedProvenance(caller, now),
	}
}

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
	// commitMu (see Engine's doc comment) holds this whole sequence —
	// store.Commit through publish — so no other goroutine's commit can
	// publish out of order in between.
	e.commitMu.Lock()
	defer e.commitMu.Unlock()

	// Best-effort floor seeding: if this is the first real use of this
	// bus, try to establish the restart floor before this commit's own
	// event might get published. A failure here is deliberately
	// swallowed — store.Commit below performs its own Load and will
	// surface the identical failure through the normal command error
	// path (e.g. cmd/harnessing's IOFailure rendering); reporting the
	// same read failure a second time here would only be noise.
	_ = e.events.ensureFloor(ctx, e.currentRevision)

	receipt, events, err := e.store.Commit(ctx, e.workspaceID, ports.CommitRequest{
		CallerAgentID:      caller.AgentID,
		RequestID:          requestID,
		PayloadFingerprint: fp,
		Mutate:             mutate,
	})
	if err != nil {
		return receipt, err
	}
	// Test-only seam (H101-71 amendment: Kelly's criterion needs a
	// deterministic proof, not a race that might not happen). No-op in
	// production; commit_seam_test.go (package task, white-box) swaps
	// it to prove commitMu actually blocks a second commit from
	// reaching store.Commit at all while this goroutine is between its
	// own store.Commit returning and its own publish — the exact gap
	// Stanley named.
	afterStoreCommitBeforePublish()
	// A replay of an already-recorded request creates no new revision,
	// mutation, or event identity — but StateStore.Commit still hands
	// this call the ORIGINAL event records after confirming durability
	// (H101-70 ruling: that is replay data, not a new publication, and
	// this call is not the one that gets to assume it never sees it).
	// events.publish is what actually keeps a replay from being
	// delivered as though newly committed: it dedups by revision, and
	// a replay's WorkspaceRevision is never higher than the original
	// commit's, because revision only advances on a fresh Mutate.
	if len(events) > 0 {
		stamped := make([]domain.Event, len(events))
		for i, ev := range events {
			ev.WorkspaceRevision = receipt.CommittedRevision
			stamped[i] = ev
		}
		e.events.publish(stamped)
	}
	return receipt, nil
}

func fingerprint(parts ...any) string {
	return fmt.Sprintf("%v", parts)
}
