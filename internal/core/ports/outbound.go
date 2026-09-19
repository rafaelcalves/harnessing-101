package ports

import (
	"context"
	"time"

	"github.com/rafaelcalves/harnessing-101/internal/core/domain"
)

// StateStore is outbound port 5.
type StateStore interface {
	Load(ctx context.Context, workspaceID domain.WorkspaceID) (domain.Snapshot, error)
	Commit(ctx context.Context, expectedRevision uint64, changes any, events []domain.Event, receipt domain.Receipt, pendingEffects any) error
	Replay(ctx context.Context, afterCursor string, limit int) ([]domain.Event, error)
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
