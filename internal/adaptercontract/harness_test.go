package adaptercontract_test

import (
	"bytes"
	"context"
	"errors"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/rafaelcalves/harnessing-101/internal/adaptercontract/expected"
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

func openContractHarness(t *testing.T, env WorkspaceEnv) *assembly.ContractHarness {
	t.Helper()
	h, err := assembly.OpenContractHarness(env.Root, env.WorkspaceID, env.Reviewers)
	if err != nil {
		t.Fatalf("OpenContractHarness: %v", err)
	}
	return h
}

func harnessSubscribeCursorExpired(t *testing.T, env WorkspaceEnv, staleCursor string) {
	t.Helper()
	h := openContractHarness(t, env)
	defer func() { _ = h.Close() }()
	_, err := h.Bind(domain.AgentID(expected.EngineerID)).Subscribe(context.Background(), staleCursor)
	var derr *domain.Error
	if !errors.As(err, &derr) || derr.Code != domain.ErrCursorExpired {
		t.Fatalf("Subscribe(%q) after reopen = %v, want CursorExpired", staleCursor, err)
	}
}

func harnessConcurrentCommitsInOrder(t *testing.T, h *assembly.ContractHarness, n int) {
	t.Helper()
	ctx := context.Background()
	session := h.Bind(domain.AgentID(expected.EngineerID))
	events, err := session.Subscribe(ctx, "0")
	if err != nil {
		t.Fatalf("Subscribe: %v", err)
	}
	var wg sync.WaitGroup
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			id := domain.TaskID("ui06-t" + strconv.Itoa(i))
			_, err := session.CreateTask(ctx, api.CreateTaskRequest{
				RequestID:  domain.RequestID("ui06-r" + strconv.Itoa(i)),
				TaskID:     id,
				Title:      "concurrent",
				AssigneeID: domain.AgentID(expected.EngineerID),
			})
			if err != nil {
				t.Errorf("CreateTask %d: %v", i, err)
			}
		}(i)
	}
	wg.Wait()

	seen := make(map[string]bool, n)
	var lastRevision uint64
	for len(seen) < n {
		select {
		case ev := <-events:
			if ev.WorkspaceRevision <= lastRevision {
				t.Fatalf("event revision %d at or below last %d", ev.WorkspaceRevision, lastRevision)
			}
			lastRevision = ev.WorkspaceRevision
			if len(ev.SubjectIDs) == 1 {
				seen[ev.SubjectIDs[0]] = true
			}
		case <-time.After(5 * time.Second):
			t.Fatalf("timed out with %d/%d events", len(seen), n)
		}
	}
}

func harnessDetachedSnapshotProbe(t *testing.T, env WorkspaceEnv) {
	t.Helper()
	h := openContractHarness(t, env)
	defer func() { _ = h.Close() }()
	ctx := context.Background()
	session := h.Bind(domain.AgentID(expected.EngineerID))
	original, err := session.GetSnapshot(ctx)
	if err != nil {
		t.Fatalf("GetSnapshot: %v", err)
	}
	if len(original.Tasks) == 0 {
		t.Fatal("detachment probe needs at least one task")
	}
	local := original
	local.Tasks[0].Title = "mutated-local-view"
	local.Messages = append(local.Messages, domain.Message{MessageID: "phantom"})
	after, err := session.GetSnapshot(ctx)
	if err != nil {
		t.Fatalf("GetSnapshot after local mutation: %v", err)
	}
	assertSnapshotUnchanged(t, original, after)
}
