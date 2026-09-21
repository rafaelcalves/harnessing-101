package adaptercontract_test

import (
	"bytes"
	"context"
	"testing"

	"github.com/rafaelcalves/harnessing-101/internal/api"
	"github.com/rafaelcalves/harnessing-101/internal/assembly"
	"github.com/rafaelcalves/harnessing-101/internal/core/domain"
)

// harnessSnapshot reads workspace state through the same assembly →
// FrontendSession path adapters use. It does not import host — item 4's
// "assembly sole host importer" should not gain a silent second importer in
// this package (H101-92).
func harnessSnapshot(t *testing.T, env WorkspaceEnv) domain.Snapshot {
	t.Helper()
	var stderr bytes.Buffer
	var snap domain.Snapshot
	var snapErr error
	code := assembly.WithSession(&stderr, env.Root, env.WorkspaceID, env.Reviewers, "", "snapshot", func(ctx context.Context, session api.FrontendSession) int {
		snap, snapErr = session.GetSnapshot(ctx)
		if snapErr != nil {
			return 1
		}
		return 0
	})
	if code != 0 {
		t.Fatalf("harness snapshot: exit=%d err=%v stderr=%q", code, snapErr, stderr.String())
	}
	return snap
}

func snapshotDigest(snap domain.Snapshot) (revision uint64, agents, tasks, messages int) {
	return snap.Revision, len(snap.Agents), len(snap.Tasks), len(snap.Messages)
}
