package assembly

import (
	"context"
	"fmt"
	"io"

	"github.com/rafaelcalves/harnessing-101/internal/adapters/idsource"
	"github.com/rafaelcalves/harnessing-101/internal/adapters/transport"
	"github.com/rafaelcalves/harnessing-101/internal/api"
	"github.com/rafaelcalves/harnessing-101/internal/core/domain"
	"github.com/rafaelcalves/harnessing-101/internal/host"
)

// Serve opens the workspace once and holds it open for transport.Host's
// whole run, binding each attaching session through host.BindFrontendSession
// exactly as WithSession binds its own one-shot session — the only
// difference is lifetime. It is item 6's minimal continuing host: no
// mailbox pump, no host-generation staleness handling, no cap above two
// attached sessions enforced here (H101-136's own stated boundary; a
// later card adds whichever of those the exit-criteria table requires).
// Serve returns when ctx is cancelled or the workspace fails to open/close.
func Serve(ctx context.Context, stderr io.Writer, root string, workspaceID domain.WorkspaceID, reviewers []domain.AgentID, generation string) int {
	if root == "" {
		_, _ = fmt.Fprintf(stderr, "harnessing serve: -workspace is required\n")
		return 1
	}
	caps, err := host.Open(root, workspaceID, reviewers)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "harnessing serve: %s\n", describeError(err))
		return 1
	}

	h := &transport.Host{
		Root:       root,
		Generation: generation,
		Bind: func(_ context.Context, attach transport.AttachPayload) (api.FrontendSession, error) {
			return host.BindFrontendSession(caps, attach.CallerAgentID), nil
		},
	}
	runErr := h.Run(ctx)

	if closeErr := caps.Close(); closeErr != nil {
		_, _ = fmt.Fprintf(stderr, "harnessing serve: workspace did not close cleanly: %s\n", describeError(closeErr))
		return 1
	}
	if runErr != nil {
		_, _ = fmt.Fprintf(stderr, "harnessing serve: %s\n", describeError(runErr))
		return 1
	}
	return 0
}

// Attach is the CLI side's entry point for joining a continuing serve
// host: it opens a Client bound to caller through the file transport
// rather than opening the workspace directly. There is no local
// host.Capabilities here at all — everything Attach's caller does goes
// through the transport Client's api.FrontendSession implementation.
func Attach(ctx context.Context, root string, caller domain.AgentID) (*transport.Client, error) {
	return transport.Attach(ctx, root, idsource.Random{}, caller)
}
