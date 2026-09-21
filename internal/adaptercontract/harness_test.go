package adaptercontract_test

import (
	"context"
	"testing"

	"github.com/rafaelcalves/harnessing-101/internal/core/domain"
	"github.com/rafaelcalves/harnessing-101/internal/host"
)

func harnessSnapshot(t *testing.T, env WorkspaceEnv) domain.Snapshot {
	t.Helper()
	caps, err := host.Open(env.Root, env.WorkspaceID, env.Reviewers)
	if err != nil {
		t.Fatalf("host.Open: %v", err)
	}
	defer caps.Close()
	snap, err := caps.GetSnapshot(context.Background())
	if err != nil {
		t.Fatalf("GetSnapshot: %v", err)
	}
	return snap
}

func snapshotDigest(snap domain.Snapshot) (revision uint64, agents, tasks, messages int) {
	return snap.Revision, len(snap.Agents), len(snap.Tasks), len(snap.Messages)
}
