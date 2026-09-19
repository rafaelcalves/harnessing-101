// Package host is the headless composition boundary (H101-19 obligation
// 1). It creates the state store and the task engine privately and hands
// out only Capabilities: one method per boundaries.md command, three
// queries, and Close. Nothing here returns a *statestore.FileStore, a
// ports.StateStore, a ports.CommitRequest builder, or any object whose
// extra methods reach persistence — the concrete type behind Capabilities
// is unexported, and docs/architecture/import-allowlist.txt is what makes
// that durable: any other production package that imports statestore or
// ports fails CI, regardless of what this file's API surface looks like.
//
// The other half of this boundary is authority, not persistence. Kelly's
// H101-27 finding: engine.CallerScope.IsHumanReviewer is a plain bool, so
// any code holding one can grant itself human-review authority. That bool
// still exists — H101-25 ruled the engine does not change — but no
// Capabilities method takes it as a parameter. Instead, Open is given a
// fixed reviewers set once, by whatever trusted code assembles the
// workspace (host configuration, not a Task/Message/Agent field, and not
// derived from anything an agent-authored payload carries), and every
// command call derives IsHumanReviewer by checking the caller's AgentID
// against that set. This directly answers the card's question: "who is
// calling" is the AgentID parameter every method already takes, and it
// cannot become "a reviewer" no matter what an ingress payload contains,
// because the check never reads the payload — only the fixed set fixed
// at construction. What this does not do, and does not claim to: stop
// trusted Go code that already holds a Capabilities value, or that
// constructs the reviewers set itself, from lying — that is exactly the
// "same-process code" / "compromised composition package" limit
// threat-model.md §7 names as explicitly out of scope. Obligation 2's
// import allowlist is what constrains which packages may even construct
// a Capabilities value or a reviewers set in the first place; this file
// only makes the boundary express *that* correctly once it exists.
package host

import (
	"context"

	"github.com/rafaelcalves/harnessing-101/internal/adapters/clock"
	"github.com/rafaelcalves/harnessing-101/internal/adapters/idsource"
	"github.com/rafaelcalves/harnessing-101/internal/adapters/statestore"
	"github.com/rafaelcalves/harnessing-101/internal/core/domain"
	"github.com/rafaelcalves/harnessing-101/internal/core/task"
)

// Capabilities is the entire surface this package returns: the version 1
// command set from boundaries.md line 19 (minus StartRun/StopRun/
// SetRunBudget, which are Phase 3 and out of this card's scope), three
// queries, and Close. No method takes an authority flag; see the package
// doc comment for how "is this caller a reviewer" is decided instead.
type Capabilities interface {
	RegisterAgent(ctx context.Context, callerAgentID domain.AgentID, req task.RegisterAgentRequest) (domain.Receipt, error)
	UpdateAgent(ctx context.Context, callerAgentID domain.AgentID, req task.UpdateAgentRequest) (domain.Receipt, error)
	CreateTask(ctx context.Context, callerAgentID domain.AgentID, req task.CreateTaskRequest) (domain.Receipt, error)
	TransitionTask(ctx context.Context, callerAgentID domain.AgentID, req task.TransitionTaskRequest) (domain.Receipt, error)
	ReportTaskResult(ctx context.Context, callerAgentID domain.AgentID, req task.ReportTaskResultRequest) (domain.Receipt, error)
	AcceptTaskResult(ctx context.Context, callerAgentID domain.AgentID, req task.AcceptTaskResultRequest) (domain.Receipt, error)
	RejectTaskResult(ctx context.Context, callerAgentID domain.AgentID, req task.RejectTaskResultRequest) (domain.Receipt, error)
	SendMessage(ctx context.Context, callerAgentID domain.AgentID, req task.SendMessageRequest) (domain.Receipt, error)
	AcknowledgeMessage(ctx context.Context, callerAgentID domain.AgentID, req task.AcknowledgeMessageRequest) (domain.Receipt, error)

	// Delivery facts the mailbox adapter would record. The engine itself
	// treats these as unauthenticated host facts (CallerScope{} today,
	// per engine.go's recordDeliveryFact) — composition passes them
	// through unchanged rather than inventing an authority model the
	// engine doesn't have.
	RecordMessagePublished(ctx context.Context, req task.MessageDeliveryRequest) (domain.Receipt, error)
	RecordMessageProcessed(ctx context.Context, req task.MessageDeliveryRequest) (domain.Receipt, error)

	// Queries return detached values: mutating the returned struct never
	// changes what a later query or a workspace reopen observes.
	GetTask(ctx context.Context, taskID domain.TaskID) (domain.Task, error)
	GetMessage(ctx context.Context, messageID domain.MessageID) (domain.Message, error)
	GetAgent(ctx context.Context, agentID domain.AgentID) (domain.Agent, error)

	// Close releases the workspace lock. It does not delete state.
	Close() error
}

// workspace is the unexported concrete type behind Capabilities. Nothing
// outside this file can name it, assert to it, or reach its store field.
type workspace struct {
	store       *statestore.FileStore
	engine      *task.Engine
	workspaceID domain.WorkspaceID
	reviewers   map[domain.AgentID]bool
}

// Open creates the store and engine for one workspace and returns only
// Capabilities. reviewerAgentIDs is the fixed, host-supplied set of
// AgentIDs with human-review authority for this workspace's lifetime —
// it comes from whoever calls Open (trusted assembly code), never from
// workspace content. A second Open against the same root fails Busy, per
// FileStore's single-writer lock.
func Open(root string, workspaceID domain.WorkspaceID, reviewerAgentIDs []domain.AgentID) (Capabilities, error) {
	store, err := statestore.Open(root)
	if err != nil {
		return nil, err
	}
	engine := task.NewEngine(store, clock.NewSystem(), idsource.Random{}, workspaceID)

	reviewers := make(map[domain.AgentID]bool, len(reviewerAgentIDs))
	for _, id := range reviewerAgentIDs {
		reviewers[id] = true
	}
	return &workspace{store: store, engine: engine, workspaceID: workspaceID, reviewers: reviewers}, nil
}

// caller builds the CallerScope the engine sees. IsHumanReviewer comes
// only from the fixed reviewers set Open was given — never a parameter
// on this or any exported method, and never anything read from req.
func (w *workspace) caller(agentID domain.AgentID) task.CallerScope {
	return task.CallerScope{AgentID: agentID, IsHumanReviewer: w.reviewers[agentID]}
}

func (w *workspace) RegisterAgent(ctx context.Context, callerAgentID domain.AgentID, req task.RegisterAgentRequest) (domain.Receipt, error) {
	return w.engine.RegisterAgent(ctx, w.caller(callerAgentID), req)
}

func (w *workspace) UpdateAgent(ctx context.Context, callerAgentID domain.AgentID, req task.UpdateAgentRequest) (domain.Receipt, error) {
	return w.engine.UpdateAgent(ctx, w.caller(callerAgentID), req)
}

func (w *workspace) CreateTask(ctx context.Context, callerAgentID domain.AgentID, req task.CreateTaskRequest) (domain.Receipt, error) {
	return w.engine.CreateTask(ctx, w.caller(callerAgentID), req)
}

func (w *workspace) TransitionTask(ctx context.Context, callerAgentID domain.AgentID, req task.TransitionTaskRequest) (domain.Receipt, error) {
	return w.engine.TransitionTask(ctx, w.caller(callerAgentID), req)
}

func (w *workspace) ReportTaskResult(ctx context.Context, callerAgentID domain.AgentID, req task.ReportTaskResultRequest) (domain.Receipt, error) {
	return w.engine.ReportTaskResult(ctx, w.caller(callerAgentID), req)
}

func (w *workspace) AcceptTaskResult(ctx context.Context, callerAgentID domain.AgentID, req task.AcceptTaskResultRequest) (domain.Receipt, error) {
	return w.engine.AcceptTaskResult(ctx, w.caller(callerAgentID), req)
}

func (w *workspace) RejectTaskResult(ctx context.Context, callerAgentID domain.AgentID, req task.RejectTaskResultRequest) (domain.Receipt, error) {
	return w.engine.RejectTaskResult(ctx, w.caller(callerAgentID), req)
}

func (w *workspace) SendMessage(ctx context.Context, callerAgentID domain.AgentID, req task.SendMessageRequest) (domain.Receipt, error) {
	return w.engine.SendMessage(ctx, w.caller(callerAgentID), req)
}

func (w *workspace) AcknowledgeMessage(ctx context.Context, callerAgentID domain.AgentID, req task.AcknowledgeMessageRequest) (domain.Receipt, error) {
	return w.engine.AcknowledgeMessage(ctx, w.caller(callerAgentID), req)
}

func (w *workspace) RecordMessagePublished(ctx context.Context, req task.MessageDeliveryRequest) (domain.Receipt, error) {
	return w.engine.RecordMessagePublished(ctx, req)
}

func (w *workspace) RecordMessageProcessed(ctx context.Context, req task.MessageDeliveryRequest) (domain.Receipt, error) {
	return w.engine.RecordMessageProcessed(ctx, req)
}

func (w *workspace) GetTask(ctx context.Context, taskID domain.TaskID) (domain.Task, error) {
	t, err := w.engine.GetTask(ctx, taskID)
	if err != nil {
		return domain.Task{}, err
	}
	return cloneTask(t), nil
}

func (w *workspace) GetMessage(ctx context.Context, messageID domain.MessageID) (domain.Message, error) {
	m, err := w.engine.GetMessage(ctx, messageID)
	if err != nil {
		return domain.Message{}, err
	}
	return cloneMessage(m), nil
}

func (w *workspace) GetAgent(ctx context.Context, agentID domain.AgentID) (domain.Agent, error) {
	a, err := w.engine.GetAgent(ctx, agentID)
	if err != nil {
		return domain.Agent{}, err
	}
	return cloneAgent(a), nil
}

func (w *workspace) Close() error {
	return w.store.Close()
}

// cloneTask/cloneMessage/cloneAgent deep-copy every pointer field so a
// caller mutating a query result cannot reach anything the store holds.
// The file adapter happens to unmarshal fresh values from disk on every
// Load today, which would make this incidentally true anyway — but that
// is the store's implementation detail, not a guarantee this boundary
// should depend on. Detachment is enforced here, once, regardless of
// what sits behind Capabilities.
func cloneTask(t domain.Task) domain.Task {
	out := t
	if t.CurrentResultID != nil {
		id := *t.CurrentResultID
		out.CurrentResultID = &id
	}
	return out
}

func cloneMessage(m domain.Message) domain.Message {
	out := m
	if m.TaskID != nil {
		id := *m.TaskID
		out.TaskID = &id
	}
	if m.ReplyToMessageID != nil {
		id := *m.ReplyToMessageID
		out.ReplyToMessageID = &id
	}
	if m.QueuedAt != nil {
		v := *m.QueuedAt
		out.QueuedAt = &v
	}
	if m.PublishedAt != nil {
		v := *m.PublishedAt
		out.PublishedAt = &v
	}
	if m.ProcessedAt != nil {
		v := *m.ProcessedAt
		out.ProcessedAt = &v
	}
	if m.AcknowledgedAt != nil {
		v := *m.AcknowledgedAt
		out.AcknowledgedAt = &v
	}
	return out
}

func cloneAgent(a domain.Agent) domain.Agent {
	out := a
	if a.LastUpdatedProvenance != nil {
		p := *a.LastUpdatedProvenance
		out.LastUpdatedProvenance = &p
	}
	return out
}
