// Contract harness for adaptercontract multi-session scenarios (UI-06/07).
// adaptercontract must not import host; assembly remains the sole importer.
package assembly

import (
	"context"

	"github.com/rafaelcalves/harnessing-101/internal/api"
	"github.com/rafaelcalves/harnessing-101/internal/core/domain"
	"github.com/rafaelcalves/harnessing-101/internal/host"
)

// ContractHarness is a long-lived workspace used when the contract suite
// needs two FrontendSession bindings or a Subscribe that outlives one
// adapter invocation. Ordinary adapter drivers still enter through CLI or
// throwaway one-shot sessions; this is for observation and concurrency only.
type ContractHarness struct {
	Root        string
	WorkspaceID domain.WorkspaceID
	Reviewers   []domain.AgentID
	caps        host.Capabilities
}

func OpenContractHarness(root string, workspaceID domain.WorkspaceID, reviewers []domain.AgentID) (*ContractHarness, error) {
	caps, err := host.Open(root, workspaceID, reviewers)
	if err != nil {
		return nil, err
	}
	return &ContractHarness{
		Root:        root,
		WorkspaceID: workspaceID,
		Reviewers:   reviewers,
		caps:        caps,
	}, nil
}

func (h *ContractHarness) Bind(caller domain.AgentID) api.FrontendSession {
	return host.BindFrontendSession(h.caps, caller)
}

func (h *ContractHarness) DriveMailbox(ctx context.Context, caller domain.AgentID) error {
	return driveMailbox(ctx, h.Root, h.caps, caller)
}

func (h *ContractHarness) Close() error {
	return h.caps.Close()
}
