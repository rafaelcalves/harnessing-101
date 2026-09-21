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
	TaskTodo           TaskStatus = "Todo"
	TaskDoing          TaskStatus = "Doing"
	TaskBlocked        TaskStatus = "Blocked"
	TaskAwaitingReview TaskStatus = "AwaitingReview"
	TaskDone           TaskStatus = "Done"
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

// Agent is a registered participant in a workspace. Provenance records who
// registered it; LastUpdatedProvenance is nil until the first UpdateAgent,
// mirroring TaskResult's Provenance/DecisionProvenance split — creation and
// the most recent write are both claims, tracked separately.
type Agent struct {
	ID                    AgentID
	DisplayName           string
	ProfileID             string
	Provenance            Provenance
	LastUpdatedProvenance *Provenance
}

// Task is a unit of accountable work. Revision is task-scoped and distinct
// from the workspace revision (boundaries.md, "completion and acceptance
// contract"); commands bind to this value to detect a stale caller.
type Task struct {
	ID              TaskID
	Title           string
	AssigneeID      AgentID
	Status          TaskStatus
	Revision        uint64
	CurrentResultID *ResultID
	Provenance      Provenance

	// LastStatusChange is the latest-status evidence boundaries.md's
	// H101-70 appendix specifies: who reported the most recent actual
	// status change and when, not a full transition history. It is set
	// on creation and replaced atomically on every real status change
	// (generic transition, report, accept, reject); a retry, a denied
	// command, or any no-op leaves it untouched. Nil means unavailable
	// — a task record written before this field existed — never
	// "nothing happened."
	LastStatusChange *StatusChange
}

// StatusChange is one task's most recent actual status transition.
// FromStatus is nil for the creation record (there is no prior status);
// every later record has one. Provenance uses the same shape as every
// other claimed-actor record in this codebase — it names who reported
// the change, not authenticated authorship or observed process activity.
type StatusChange struct {
	TaskRevision uint64
	FromStatus   *TaskStatus
	ToStatus     TaskStatus
	Provenance   Provenance
}

// IdentityVerification names how much a claimed identity was checked. The
// only value that exists in this phase is Unverified: even a host-supplied
// caller scope is application-level authorization, not cryptographic
// authentication, and a written sender field never authenticates its
// author (docs/security/threat-model.md §4-5). No stronger value is
// defined until a verification mechanism actually exists — do not add one
// speculatively.
type IdentityVerification string

const IdentityUnverified IdentityVerification = "unverified"

// EntryMechanism records how a record entered the workspace. "command" is
// the only value this slice produces: every record here comes from a
// direct, in-process command call, not a file-based mailbox ingress.
type EntryMechanism string

const EntryMechanismCommand EntryMechanism = "command"

// Provenance is attached to every record whose content or identity claim
// did not come from a verified source. It exists from Phase 1 per
// docs/product/definition.md "do not defer provenance past Phase 1":
// retrofitting it onto a record format already in use is expensive.
// ClaimedAgentID is named "claimed" deliberately — it is the actor
// identity as asserted by the caller scope the host supplied, and it is
// never treated as, or labelled as, checked identity.
type Provenance struct {
	ClaimedAgentID       AgentID
	EntryMechanism       EntryMechanism
	RecordedAt           time.Time
	IdentityVerification IdentityVerification
}

// TaskResult is an immutable report against one task. Once created it is
// never mutated; rework produces a new ResultID (boundaries.md line 28).
// Decision fields are set exactly once, by AcceptTaskResult or
// RejectTaskResult, and never change afterwards.
type TaskResult struct {
	ResultID   ResultID
	TaskID     TaskID
	Summary    string
	Artifacts  []string
	Provenance Provenance

	Decision           TaskResultDecision
	DecisionReason     string
	DecisionProvenance Provenance
}

// TaskResultDecision is empty ("") while a result is awaiting review.
type TaskResultDecision string

const (
	TaskResultPending  TaskResultDecision = ""
	TaskResultAccepted TaskResultDecision = "Accepted"
	TaskResultRejected TaskResultDecision = "Rejected"
)

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
	TaskID           *TaskID
	ReplyToMessageID *MessageID
}

// MessageAcknowledgement is the mailbox's second file-protocol record
// (boundaries.md "file acknowledgement and archival"): a protocol
// control record, not a Request/Inform/Result message, and never itself
// requests an acknowledgement. ControlRecordID is the record's own
// identity for deduplication — distinct from MessageID, the message it
// acknowledges. RecipientAgentID is the claimed acknowledging agent; the
// adapter must establish that scope from where the record was found
// (which agent's own area it came from), never trust this field alone
// to grant recipient authority.
type MessageAcknowledgement struct {
	SchemaVersion     int
	WorkspaceID       WorkspaceID
	ControlRecordID   string
	OriginalMessageID MessageID
	RecipientAgentID  AgentID
}

// Message is the durable record for one handoff. Delivery facts are separate
// timestamps: a missing later timestamp is not inferred from an earlier one.
// Acknowledgement is explicit recipient action and does not mean the work is
// complete.
type Message struct {
	WorkspaceID      WorkspaceID
	MessageID        MessageID
	SenderAgentID    AgentID
	RecipientAgentID AgentID
	Kind             MessageKind
	Body             string
	CreatedAt        time.Time
	TaskID           *TaskID
	ReplyToMessageID *MessageID
	Provenance       Provenance

	QueuedAt       *time.Time
	PublishedAt    *time.Time
	ProcessedAt    *time.Time
	AcknowledgedBy AgentID
	AcknowledgedAt *time.Time
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
	TaskResults []TaskResult
	Messages    []Message
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
