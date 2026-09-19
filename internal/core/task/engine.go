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
	"unicode/utf8"

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

// SendMessageRequest is the version 1 message command. SenderAgentID is the
// message's claimed sender; Provenance separately records the host-scoped
// caller and remains Unverified in this phase.
type SendMessageRequest struct {
	RequestID        domain.RequestID
	MessageID        domain.MessageID
	SenderAgentID    domain.AgentID
	RecipientAgentID domain.AgentID
	Kind             domain.MessageKind
	Body             string
	TaskID           *domain.TaskID
	ReplyToMessageID *domain.MessageID
}

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
type AcknowledgeMessageRequest struct {
	RequestID domain.RequestID
	MessageID domain.MessageID
}

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

// MessageDeliveryRequest identifies a delivery fact recorded by the host.
type MessageDeliveryRequest struct {
	RequestID domain.RequestID
	MessageID domain.MessageID
}

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
