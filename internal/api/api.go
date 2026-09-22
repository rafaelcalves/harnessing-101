// Package api contains the neutral frontend contract. It imports only
// context and passive domain records; it never reaches host, task, storage,
// or an outbound port.
package api

import (
	"context"

	"github.com/rafaelcalves/harnessing-101/internal/core/domain"
)

type RegisterAgentRequest struct {
	RequestID   domain.RequestID
	AgentID     domain.AgentID
	DisplayName string
	ProfileID   string
}

type UpdateAgentRequest struct {
	RequestID   domain.RequestID
	AgentID     domain.AgentID
	DisplayName *string
	ProfileID   *string
}

type CreateTaskRequest struct {
	RequestID  domain.RequestID
	TaskID     domain.TaskID
	Title      string
	AssigneeID domain.AgentID
}

type TransitionTaskRequest struct {
	RequestID  domain.RequestID
	TaskID     domain.TaskID
	FromStatus domain.TaskStatus
	ToStatus   domain.TaskStatus
	Reason     string
}

type ReportTaskResultRequest struct {
	RequestID            domain.RequestID
	TaskID               domain.TaskID
	ResultID             domain.ResultID
	ExpectedTaskRevision uint64
	Summary              string
	Artifacts            []string
}

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

type AcknowledgeMessageRequest struct {
	RequestID domain.RequestID
	MessageID domain.MessageID
}

type MessageDeliveryRequest struct {
	RequestID domain.RequestID
	MessageID domain.MessageID
}

// StartRunRequest's TaskID/PeerAgentID are participation context (R2),
// threaded to the supervisor as domain.RunParticipationContext — never
// treated as authorization. ToolExecutable/ToolArgv, when set, are the
// per-run invocation the caller asks to launch (mirroring
// H101-152's runner CLI contract, where a trusted local script resolves
// a descriptor's context slots before ever calling this product
// surface); the engine accepts them only after confirming
// ToolExecutable matches the profile's own approved executable — the
// profile still fixes WHICH BINARY may run, never an ad hoc one (threat
// model rule 5). Leaving both empty uses the approved profile's Args
// verbatim, which is what Layer A's participation fixture and this
// card's native tests do.
type StartRunRequest struct {
	RequestID      domain.RequestID
	RunID          string
	AgentID        domain.AgentID
	ProfileID      string
	TaskID         *domain.TaskID
	PeerAgentID    *domain.AgentID
	ToolExecutable string
	ToolArgv       []string
}

// ApproveProfileRequest is the host-established human decision that
// makes a ProfileID eligible for StartRun (threat-model rule 5): it
// fixes the executable, args, and environment a caller may later ask to
// run under this ProfileID, verbatim. There is no edit path — approving
// an already-approved ProfileID again is Conflict, not a revision bump.
type ApproveProfileRequest struct {
	RequestID domain.RequestID
	ProfileID string
	Spec      domain.ExecutionSpec
}

type StopRunRequest struct {
	RequestID domain.RequestID
	RunID     string
	Reason    string
}

type SetRunBudgetRequest struct {
	RequestID          domain.RequestID
	RunID              string
	ElapsedTimeLimit   *uint64
	ReportedTokenLimit *uint64
}

// FrontendSession is the caller-bound interface a presentation adapter may
// hold. It has no lifecycle, mailbox, storage, or caller-identity authority.
type FrontendSession interface {
	RegisterAgent(context.Context, RegisterAgentRequest) (domain.Receipt, error)
	UpdateAgent(context.Context, UpdateAgentRequest) (domain.Receipt, error)
	CreateTask(context.Context, CreateTaskRequest) (domain.Receipt, error)
	TransitionTask(context.Context, TransitionTaskRequest) (domain.Receipt, error)
	ReportTaskResult(context.Context, ReportTaskResultRequest) (domain.Receipt, error)
	AcceptTaskResult(context.Context, AcceptTaskResultRequest) (domain.Receipt, error)
	RejectTaskResult(context.Context, RejectTaskResultRequest) (domain.Receipt, error)
	SendMessage(context.Context, SendMessageRequest) (domain.Receipt, error)
	AcknowledgeMessage(context.Context, AcknowledgeMessageRequest) (domain.Receipt, error)
	ApproveProfile(context.Context, ApproveProfileRequest) (domain.Receipt, error)
	StartRun(context.Context, StartRunRequest) (domain.Receipt, error)
	StopRun(context.Context, StopRunRequest) (domain.Receipt, error)
	SetRunBudget(context.Context, SetRunBudgetRequest) (domain.Receipt, error)
	GetAgent(context.Context, domain.AgentID) (domain.Agent, error)
	GetTask(context.Context, domain.TaskID) (domain.Task, error)
	GetMessage(context.Context, domain.MessageID) (domain.Message, error)
	GetRun(context.Context, domain.RunID) (domain.Run, error)
	GetOperation(context.Context, domain.OperationID) (domain.Operation, error)
	GetSnapshot(context.Context) (domain.Snapshot, error)
	ResolveRequest(context.Context, domain.RequestID) (domain.Receipt, error)
	Subscribe(context.Context, string) (<-chan domain.Event, error)
}
