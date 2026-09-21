package ports

import (
	"context"
	"time"

	"github.com/rafaelcalves/harnessing-101/internal/core/domain"
)

// StateStore is outbound port 5.
//
// Commit is atomic and single-writer. The adapter owns the on-disk
// transaction, the workspace revision counter, and command-replay
// idempotency (boundaries.md "retries and conflicts"): it looks up
// (CallerAgentID, RequestID) in a durable receipt ledger before running
// Mutate. A match with an identical PayloadFingerprint creates no new
// revision, mutation, or event identity: Mutate does not run again, and
// after required confirmation this returns the ORIGINAL receipt and the
// ORIGINAL event records from when they were first committed (H101-70
// ruling, docs/architecture/h101-70-status-replay-observation.md §2) —
// those returned records are replay data, not a new publication, and
// every consumer of this return value must tell the two apart rather
// than treat a replay's events as freshly committed. A match with a
// different fingerprint fails Conflict; a request ID is scoped to its
// caller, so
// replaying someone else's request ID never returns their receipt. Task-
// level optimistic concurrency (an expectedTaskRevision, the current
// result ID, reviewer authority) is a domain rule Mutate enforces against
// the snapshot it is given, not a StateStore-level check. On any error
// from Mutate, or a mismatched fingerprint, nothing is persisted and no
// receipt is recorded — boundaries.md "commands that fail validation
// commit nothing."
//
// This diverges deliberately from boundaries.md's literal
// Commit(expectedRevision, changes, events, receipt, pendingEffects)
// signature: that document leaves "changes" and "pendingEffects"
// unspecified, and a mutate closure lets the core express one atomic,
// multi-record change (task + result + decision together) without the
// port inventing a generic diff format ahead of need. Replay(afterCursor)
// is dropped from this port for this slice: no StateEvents subscriber
// exists yet to consume it, and boundaries.md does not fix its shape
// either. Revisit both if a later phase needs them.
type StateStore interface {
	Load(ctx context.Context, workspaceID domain.WorkspaceID) (domain.Snapshot, error)
	Commit(ctx context.Context, workspaceID domain.WorkspaceID, req CommitRequest) (domain.Receipt, []domain.Event, error)

	// ResolveRequest is ADR 0004's caller-bound resolution operation
	// (docs/adr/0004-uncertain-command-outcomes.md, appended ruling
	// docs/architecture/h101-25-state-store-authority.md /
	// ADR-0004-append `267885e`: adding a method to an existing port is
	// not a breach of "ten ports" — that property is the count and role
	// of the ten, four inbound and six outbound, not a frozen method
	// set). It performs no domain mutation, allocates no new request
	// ID, and does not advance the workspace revision; it may still
	// perform confirmation I/O. A matching receipt with confirmed
	// durability returns Confirmed (the original receipt, unchanged);
	// a still-failing confirmation returns OutcomeUncertain with the
	// best observed evidence; no receipt for (callerAgentID, requestID)
	// returns NotFound — Absent, meaning "no record found now," never
	// "this never happened."
	//
	// callerAgentID here is a scoping key into a caller-partitioned
	// ledger (receiptKey), not an identity check: ledger keying scopes
	// a lookup, it does not authenticate the caller. Nothing in this
	// port binds callerAgentID to a real session — that binding is
	// internal/host's job (FrontendSession, bound once at
	// BindFrontendSession, never re-supplied per call). A caller of
	// this port method directly, with an arbitrary callerAgentID, gets
	// exactly that ID's records — which is why only reviewed
	// composition code may hold a StateStore value at all (the import
	// allowlist), not a substitute for that review.
	ResolveRequest(ctx context.Context, workspaceID domain.WorkspaceID, callerAgentID domain.AgentID, requestID domain.RequestID) (domain.Receipt, error)
}

// CommitRequest is one attempted mutation against a workspace.
type CommitRequest struct {
	CallerAgentID domain.AgentID
	RequestID     domain.RequestID
	// PayloadFingerprint identifies the logical command payload. Replaying
	// the same (CallerAgentID, RequestID) with a different fingerprint is
	// Conflict, per boundaries.md "a different payload returns Conflict".
	PayloadFingerprint string
	// Mutate receives a deep copy of the current snapshot and returns the
	// events a successful change produces. It must not mutate anything
	// reachable outside that copy, and must return a *domain.Error with a
	// stable Code (InvalidArgument, NotFound, Conflict, Denied) on any
	// domain-rule failure.
	Mutate func(snapshot *domain.Snapshot) ([]domain.Event, error)
}

// Mailbox is outbound port 6.
type Mailbox interface {
	ScanInbox(ctx context.Context, agentID domain.AgentID, cursor string) ([]domain.Envelope, error)
	Publish(ctx context.Context, messageID domain.MessageID, envelope domain.Envelope) error
	Archive(ctx context.Context, messageID domain.MessageID) error
}

// ProcessSupervisor is outbound port 7.
type ProcessSupervisor interface {
	Capabilities(ctx context.Context) (any, error)
	Start(ctx context.Context, runID domain.RunID, spec domain.ExecutionSpec) error
	Observe(ctx context.Context, runID domain.RunID) (<-chan any, error)
	Stop(ctx context.Context, runID domain.RunID, grace time.Duration) error
	Recover(ctx context.Context, runID domain.RunID) error
}

// OutputJournal is outbound port 8.
type OutputJournal interface {
	Append(ctx context.Context, runID domain.RunID, channel string, bytes []byte, captureTime time.Time) (uint64, error)
	ReadAfter(ctx context.Context, runID domain.RunID, offset uint64, limit int) (any, error)
	Follow(ctx context.Context, runID domain.RunID, offset uint64) (<-chan any, error)
	Finish(ctx context.Context, runID domain.RunID, captureOutcome any) error
}

// Clock is outbound port 9. Tests supply a controlled implementation;
// persisted wall time never substitutes for a continuous monotonic clock
// across a restart.
type Clock interface {
	WallNow() time.Time
	MonotonicNow() time.Duration
}

// IDSource is outbound port 10. IDs are workspace-unique and opaque; they
// encode no path, PID, display name, or timestamp.
type IDSource interface {
	NewID(ctx context.Context) (string, error)
}
