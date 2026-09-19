// Package ports declares the ten contracts between the core and its
// adapters: four inbound (what any UI can call) and six outbound (what the
// core requires of its environment). See docs/architecture/boundaries.md.
//
// Phase 0: signatures only, no implementation, no behaviour. Payload and
// command types are placeholders until Phase 1 defines the command set.
package ports

import (
	"context"

	"github.com/rafaelcalves/harnessing-101/internal/core/domain"
)

// Command is a versioned inbound instruction. Kind and Payload are
// undefined at this phase; Phase 1 fixes the command set from
// boundaries.md's "version 1 command payloads".
type Command struct {
	WorkspaceID      domain.WorkspaceID
	RequestID        domain.RequestID
	Kind             string
	Payload          any
	ExpectedRevision *uint64
}

// Commands is inbound port 1.
type Commands interface {
	Execute(ctx context.Context, cmd Command) (domain.Receipt, error)
}

// Queries is inbound port 2.
type Queries interface {
	GetSnapshot(ctx context.Context, workspaceID domain.WorkspaceID) (domain.Snapshot, error)
	GetAgent(ctx context.Context, id domain.AgentID) (domain.Agent, error)
	GetTask(ctx context.Context, id domain.TaskID) (domain.Task, error)
	GetRun(ctx context.Context, id domain.RunID) (any, error)
	GetOperation(ctx context.Context, id domain.OperationID) (any, error)
	GetCapabilities(ctx context.Context, workspaceID domain.WorkspaceID) (any, error)
}

// StateEvents is inbound port 3.
type StateEvents interface {
	Subscribe(ctx context.Context, workspaceID domain.WorkspaceID, afterCursor string) (<-chan domain.Event, error)
}

// RunOutput is inbound port 4.
type RunOutput interface {
	ReadOutput(ctx context.Context, runID domain.RunID, afterOffset uint64, byteLimit int) (any, error)
	FollowOutput(ctx context.Context, runID domain.RunID, afterOffset uint64) (<-chan any, error)
}
