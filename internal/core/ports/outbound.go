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
// Mutate. A match with an identical PayloadFingerprint returns the
// recorded receipt with no new events; a match with a different
// fingerprint fails Conflict; a request ID is scoped to its caller, so
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
