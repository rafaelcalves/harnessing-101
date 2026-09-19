package domain

import "time"

// Enumerations from boundaries.md. Phase 0 declares the vocabulary only; no
// transition table is enforced yet.
type (
	TaskStatus     string
	MessageKind    string
	RunState       string
	OperationState string
)

const (
	TaskTodo    TaskStatus = "Todo"
	TaskDoing   TaskStatus = "Doing"
	TaskBlocked TaskStatus = "Blocked"
	TaskDone    TaskStatus = "Done"
)

const (
	MessageRequest MessageKind = "Request"
	MessageInform  MessageKind = "Inform"
	MessageResult  MessageKind = "Result"
)

const (
	RunStarting         RunState = "Starting"
	RunRunning          RunState = "Running"
	RunStopping         RunState = "Stopping"
	RunExited           RunState = "Exited"
	RunRecoveryRequired RunState = "RecoveryRequired"
)

const (
	OperationPending          OperationState = "Pending"
	OperationRunning          OperationState = "Running"
	OperationSucceeded        OperationState = "Succeeded"
	OperationFailed           OperationState = "Failed"
	OperationRecoveryRequired OperationState = "RecoveryRequired"
)

// Agent is a registered participant in a workspace.
type Agent struct {
	ID          AgentID
	DisplayName string
	ProfileID   string
}

// Task is a unit of accountable work.
type Task struct {
	ID         TaskID
	Title      string
	AssigneeID AgentID
	Status     TaskStatus
}

// Envelope is a mailbox message as adapters exchange it. See boundaries.md,
// "event and mailbox records".
type Envelope struct {
	SchemaVersion    int
	WorkspaceID      WorkspaceID
	MessageID        MessageID
	SenderAgentID    AgentID
	RecipientAgentID AgentID
	Kind             MessageKind
	Body             string
	CreatedAt        time.Time
	ReplyToMessageID *MessageID
}

// Receipt is returned by a successful Commands.Execute.
type Receipt struct {
	RequestID         RequestID
	CommittedRevision uint64
	OperationID       *OperationID
}

// Event is one ordered, committed state transition. Payload shape is
// per-Kind and undefined at this phase.
type Event struct {
	ID                EventID
	Kind              string
	SchemaVersion     int
	WorkspaceRevision uint64
	Timestamp         time.Time
	SubjectIDs        []string
	Payload           any
}

// Snapshot is a consistent, versioned read of one workspace.
type Snapshot struct {
	WorkspaceID WorkspaceID
	Revision    uint64
	Cursor      string
	Agents      []Agent
	Tasks       []Task
}

// ExecutionSpec describes a local process an adapter may start. Resolving
// paths, environment, descriptors, signals, and process groups is adapter
// work; the core never inspects a process ID. See boundaries.md,
// "supervision and recovery".
type ExecutionSpec struct {
	ProfileID        string
	Args             []string
	WorkingDirectory string
	EnvironmentRefs  []string
}
