package task_test

import (
	"context"
	"strconv"
	"testing"
	"time"

	"github.com/rafaelcalves/harnessing-101/internal/core/domain"
	"github.com/rafaelcalves/harnessing-101/internal/core/task"
)

// TestSubscribe_NoLostCommitsFromSnapshotCursor is UI-06's central
// requirement (H101-61): the cursor GetSnapshot hands back must be
// usable as afterCursor with nothing in the gap. Commit one task before
// subscribing and one after; both must arrive, in order, and the
// pre-snapshot commit must NOT be redelivered.
func TestSubscribe_NoLostCommitsFromSnapshotCursor(t *testing.T) {
	e, cleanup := newEngine(t, t.TempDir())
	defer cleanup()
	ctx := context.Background()

	if _, err := e.CreateTask(ctx, caller(engineerID, false), task.CreateTaskRequest{
		RequestID: "r1", TaskID: "t1", Title: "before",
	}); err != nil {
		t.Fatalf("CreateTask t1: %v", err)
	}

	snap, err := e.GetSnapshot(ctx)
	if err != nil {
		t.Fatalf("GetSnapshot: %v", err)
	}

	subCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	events, err := e.Subscribe(subCtx, snap.Cursor)
	if err != nil {
		t.Fatalf("Subscribe(%q): %v", snap.Cursor, err)
	}

	if _, err := e.CreateTask(ctx, caller(engineerID, false), task.CreateTaskRequest{
		RequestID: "r2", TaskID: "t2", Title: "after",
	}); err != nil {
		t.Fatalf("CreateTask t2: %v", err)
	}

	select {
	case ev := <-events:
		if ev.Kind != "TaskCreated" {
			t.Fatalf("event kind = %q, want TaskCreated", ev.Kind)
		}
		if len(ev.SubjectIDs) != 1 || ev.SubjectIDs[0] != "t2" {
			t.Fatalf("event subjects = %v, want [t2] (t1's event, from before the cursor, must not be redelivered)", ev.SubjectIDs)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for the post-cursor event")
	}
}

// TestSubscribe_ReplayDoesNotDuplicate proves the store-level replay
// contract (identical request ID + payload returns the original receipt
// with no new events) also means a subscriber never sees a retried
// command's event twice.
func TestSubscribe_ReplayDoesNotDuplicate(t *testing.T) {
	e, cleanup := newEngine(t, t.TempDir())
	defer cleanup()
	ctx := context.Background()

	req := task.CreateTaskRequest{RequestID: "r1", TaskID: "t1", Title: "x"}
	if _, err := e.CreateTask(ctx, caller(engineerID, false), req); err != nil {
		t.Fatalf("CreateTask (first): %v", err)
	}

	events, err := e.Subscribe(ctx, "0")
	if err != nil {
		t.Fatalf("Subscribe: %v", err)
	}
	select {
	case <-events:
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for the first event")
	}

	// Identical retry: same request ID, same payload.
	if _, err := e.CreateTask(ctx, caller(engineerID, false), req); err != nil {
		t.Fatalf("CreateTask (replay): %v", err)
	}

	select {
	case ev, ok := <-events:
		if ok {
			t.Fatalf("replay must not publish a second event, got %+v", ev)
		}
	case <-time.After(300 * time.Millisecond):
		// No second event arrived — correct, replay produced none.
	}
}

// TestSubscribe_CursorExpiredIsHonest: a cursor this process never
// produced (or has since evicted from its bounded buffer) must come back
// CursorExpired, never silently served as empty — the ADR 0003 failure
// mode this exists to prevent.
func TestSubscribe_CursorExpiredIsHonest(t *testing.T) {
	e, cleanup := newEngine(t, t.TempDir())
	defer cleanup()
	ctx := context.Background()

	t.Run("unparseable cursor", func(t *testing.T) {
		_, err := e.Subscribe(ctx, "not-a-revision")
		mustErrorCode(t, err, domain.ErrCursorExpired)
	})

	t.Run("evicted cursor", func(t *testing.T) {
		for i := 0; i < 300; i++ {
			id := domain.TaskID("t" + strconv.Itoa(i))
			if _, err := e.CreateTask(ctx, caller(engineerID, false), task.CreateTaskRequest{
				RequestID: domain.RequestID("r" + strconv.Itoa(i)), TaskID: id, Title: "x",
			}); err != nil {
				t.Fatalf("CreateTask %d: %v", i, err)
			}
		}
		// Revision 1 was long evicted by the bounded buffer (capacity
		// 256, 300 commits made) by the time this asks for it.
		_, err := e.Subscribe(ctx, "1")
		mustErrorCode(t, err, domain.ErrCursorExpired)
	})
}

// TestSubscribe_UnsubscribesOnContextCancel proves a cancelled
// subscription's channel is closed rather than leaked — UI-07/UI-08's
// "detaching an observer" must be a real disconnect.
func TestSubscribe_UnsubscribesOnContextCancel(t *testing.T) {
	e, cleanup := newEngine(t, t.TempDir())
	defer cleanup()
	ctx := context.Background()
	subCtx, cancel := context.WithCancel(ctx)

	events, err := e.Subscribe(subCtx, "0")
	if err != nil {
		t.Fatalf("Subscribe: %v", err)
	}
	cancel()

	select {
	case _, ok := <-events:
		if ok {
			t.Fatal("expected the channel to close after context cancellation, got a value instead")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for the channel to close after cancel")
	}
}
