package task

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/rafaelcalves/harnessing-101/internal/adapters/clock"
	"github.com/rafaelcalves/harnessing-101/internal/adapters/idsource"
	"github.com/rafaelcalves/harnessing-101/internal/adapters/statestore"
	"github.com/rafaelcalves/harnessing-101/internal/core/domain"
)

// TestCommitMu_BlocksSecondCommitInTheNamedGap is the H101-71 amendment's
// deterministic proof for Kelly's second named criterion: "an
// interleaving where revision R+1 publishes before R must not drop R."
// A stress test alone cannot prove this — it only shows the race did not
// fire, which proves nothing about whether it could. This uses the
// afterStoreCommitBeforePublish seam to put goroutine A INSIDE the exact
// gap Stanley named (after its own store.Commit returned, before its own
// publish) and then attempts a second commit from goroutine B. If
// commitMu is doing its job, B cannot even begin its own store.Commit
// call — let alone publish out of turn — until A's whole sequence,
// seam included, finishes. This is white-box (package task, not
// task_test) because the seam is intentionally unexported: nothing
// outside this package should ever be able to reach into commit
// ordering, in production or otherwise.
func TestCommitMu_BlocksSecondCommitInTheNamedGap(t *testing.T) {
	store, err := statestore.Open(t.TempDir())
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer func() { _ = store.Close() }()
	e := NewEngine(store, clock.NewSystem(), idsource.Random{}, "ws-seam")
	ctx := context.Background()

	entered := make(chan struct{})
	release := make(chan struct{})
	var triggered atomic.Bool
	orig := afterStoreCommitBeforePublish
	afterStoreCommitBeforePublish = func() {
		// Only the FIRST goroutine to reach this seam pauses; deliberately
		// a non-blocking CompareAndSwap, not sync.Once — Once's Do blocks
		// a SECOND concurrent caller until the first's function returns,
		// which would make this seam accidentally serialize B all by
		// itself and prove nothing about commitMu. CompareAndSwap lets a
		// second, unsynchronized caller (the failure mode this test
		// exists to catch) sail straight through instead.
		if triggered.CompareAndSwap(false, true) {
			close(entered)
			<-release
		}
	}
	defer func() { afterStoreCommitBeforePublish = orig }()

	doneA := make(chan error, 1)
	go func() {
		_, err := e.CreateTask(ctx, CallerScope{AgentID: "engineer"}, CreateTaskRequest{
			RequestID: "rA", TaskID: "tA", Title: "x",
		})
		doneA <- err
	}()

	select {
	case <-entered:
	case <-time.After(2 * time.Second):
		t.Fatal("goroutine A never reached the seam")
	}

	doneB := make(chan error, 1)
	go func() {
		_, err := e.CreateTask(ctx, CallerScope{AgentID: "engineer"}, CreateTaskRequest{
			RequestID: "rB", TaskID: "tB", Title: "y",
		})
		doneB <- err
	}()

	select {
	case <-doneB:
		t.Fatal("goroutine B's commit completed while A was still between store.Commit and publish — commitMu did not block it")
	case <-time.After(200 * time.Millisecond):
		// Expected: B is still blocked trying to acquire commitMu (or,
		// with the fix removed, would already be racing A directly —
		// see the sibling test that runs this same seam without the
		// lock held).
	}

	close(release)

	if err := <-doneA; err != nil {
		t.Fatalf("goroutine A: %v", err)
	}
	if err := <-doneB; err != nil {
		t.Fatalf("goroutine B: %v", err)
	}
}

// TestEventBus_OutOfOrderPublishDropsTheEarlierRevision is the other
// half of the same proof: it shows, directly and deterministically, WHY
// the ordering commitMu enforces matters — feed the bus revision 2
// before revision 1 (no goroutines, no races, just the two publish
// calls in the "bad" order) and confirm revision 1 really is dropped.
// This is the failure Stanley named, reproduced on demand rather than
// argued about; TestCommitMu_BlocksSecondCommitInTheNamedGap is what
// proves Engine never actually feeds the bus in this order.
func TestEventBus_OutOfOrderPublishDropsTheEarlierRevision(t *testing.T) {
	bus := newEventBus()
	sub, err := bus.subscribe(context.Background(), "0")
	if err != nil {
		t.Fatalf("subscribe: %v", err)
	}

	bus.publish([]domain.Event{{ID: "ev-2", Kind: "Test", WorkspaceRevision: 2}})
	bus.publish([]domain.Event{{ID: "ev-1", Kind: "Test", WorkspaceRevision: 1}})

	select {
	case ev := <-sub:
		if ev.ID != "ev-2" {
			t.Fatalf("first delivered event = %q, want ev-2", ev.ID)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for ev-2")
	}
	select {
	case ev, ok := <-sub:
		if ok {
			t.Fatalf("ev-1 must have been dropped by the out-of-order dedup, but got %+v", ev)
		}
	case <-time.After(200 * time.Millisecond):
		// Expected: nothing else arrives — ev-1 was silently dropped,
		// exactly the failure this bus's revision-ordering contract
		// depends on its caller (Engine, via commitMu) to prevent.
	}
}
