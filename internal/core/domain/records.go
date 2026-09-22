package domain

import "time"

// Enumerations from boundaries.md. Phase 0 declares the vocabulary only; no
// transition table is enforced yet.
type (
	TaskStatus     string
	MessageKind    string
	RunState       string
	OperationState string
	CaptureStatus  string
)

const (
	CaptureInterrupted CaptureStatus = "interrupted"
	CaptureUnknown     CaptureStatus = "unknown"
	CaptureComplete    CaptureStatus = "complete"
)

// OutputChunk is a durably appended stdout/stderr segment. Offset advances
// only after the journal persists the segment; absent tail bytes are never
// represented by a synthetic count or range.
type OutputChunk struct {
	Offset     uint64
	Channel    string
	Bytes      []byte
	CapturedAt time.Time
}

// RunOutput is the product read model for the minimal output-journal slice.
type RunOutput struct {
	RunID         RunID
	Chunks        []OutputChunk
	CaptureStatus CaptureStatus
}

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
//
// SenderAgentID, like Provenance.ClaimedAgentID, is a claimed identity, not
// an authenticated one: SendMessage checks it against the agent registry
// (membership) but never against the caller scope that submitted the
// command (authorship). It is not renamed to a Claimed* form here because
// this is a stored wire-format field shared with Envelope and read by every
// adapter; the same caveat applies to RecipientAgentID, which additionally
// gains no special claim status from being checked — a registered ID is
// still just an address, not proof anyone at that address agreed to
// anything.
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
	Profiles    []Profile
	Runs        []Run
	Operations  []Operation
}

// ExecutionSpec describes a local process an adapter may start. Resolving
// paths, environment, descriptors, signals, and process groups is adapter
// work; the core never inspects a process ID. See boundaries.md,
// "supervision and recovery".
//
// ContextTransport is Phase 3 item 1's R2 (participation context)
// mechanism: how task/workspace identifiers reach the launched tool.
// Only "context-file" is implemented by internal/adapters/process today
// (R2's "documented protocol handoff"); any other transport value a
// profile names is honored as a stable, explicit Unsupported at Start
// rather than silently ignored (R3) — see process.Supervisor.Start's
// doc comment. Empty ContextTransport means the profile carries no
// participation context at all (a plain spawn — never used for a real
// managed-agent-tool claim, only for fixtures that do not need one).
type ExecutionSpec struct {
	ProfileID        string
	Args             []string
	WorkingDirectory string
	EnvironmentRefs  []string
	ContextTransport string
}

// Operation is boundaries.md's command-completion record (line 40:
// "StartRun/StopRun return operation IDs, with progress and terminal
// outcomes obtained through Queries/StateEvents"): a durable record of
// one accepted intent's own progress, separate from the Run it acts on.
// Operation completion and run exit are different facts (Stanley,
// H101-158's D5 ruling) — an operation reaching Succeeded means the
// dispatch was carried out and observed, not that the run's whole
// lifetime is over; Exited is the Run's own terminal state, tracked
// independently.
type Operation struct {
	ID          OperationID
	RunID       RunID
	Kind        string
	State       OperationState
	CreatedAt   time.Time
	CompletedAt *time.Time
	Outcome     string
}

// RunParticipationContext is what StartRun hands the supervisor about
// THIS run specifically — distinct from ExecutionSpec, which is the
// PROFILE's own fixed, approved shape. Only the identifiers named here
// ever reach a launched tool via ContextTransport; there is no path for
// arbitrary request-supplied text to ride along as "context."
type RunParticipationContext struct {
	WorkspaceID WorkspaceID
	RunID       RunID
	AgentID     AgentID
	TaskID      *TaskID
	PeerAgentID *AgentID
}

// Profile is a host-recorded approval of one local execution profile
// (threat-model rule 5): ProcessSupervisor.Start — and this product's
// StartRun — rejects any spec whose ProfileID lacks a prior recorded
// approval event here. Approval fixes the executable, args, and
// environment; StartRun never accepts ad hoc argv. There is no update
// path in this slice: approving an already-approved ProfileID is
// Conflict, not a silent revision bump (CF3 — no retroactive change to
// what was approved).
type Profile struct {
	ProfileID  string
	Spec       ExecutionSpec
	Provenance Provenance
}

// Run is one StartRun/StopRun lifecycle record (boundaries.md
// "supervision and recovery"; H101-135/H101-128's dispatch-ordering
// invariant). DispatchAttemptedAt is committed durably BEFORE
// ProcessSupervisor.Start is ever invoked for this run, and confirmed
// durable before that call — the marker H101-135 requires. A run
// observed with DispatchAttemptedAt set but StartedAt still nil, and no
// recorded exit, is the invariant's named ambiguous state: this product
// never automatically redispatches it; only explicit recovery (item 4)
// resolves it later. ExitReason, when set, uses one of the H101-152
// classification names (missing_tool, authentication_required,
// network_egress_refused, unsupported_invocation, spawn_failed) or
// "stopped" — never a raw error string a caller would have to parse to
// find the taxonomy.
type Run struct {
	ID                  RunID
	AgentID             AgentID
	ProfileID           string
	State               RunState
	Revision            uint64
	Provenance          Provenance
	DispatchAttemptedAt *time.Time
	StartedAt           *time.Time
	ExitedAt            *time.Time
	ExitReason          string
}
