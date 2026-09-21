package adaptercontract_test

import (
	"testing"

	"github.com/rafaelcalves/harnessing-101/internal/adaptercontract/expected"
	"github.com/rafaelcalves/harnessing-101/internal/core/domain"
)

func drivers(t *testing.T) []Driver {
	return []Driver{
		newCLIDriver(t),
		newThrowawayDriver(),
	}
}

func newEnv(t *testing.T) WorkspaceEnv {
	return WorkspaceEnv{
		Root:        t.TempDir(),
		WorkspaceID: domain.WorkspaceID(expected.WorkspaceID),
		Reviewers:   expected.Reviewers(),
	}
}
