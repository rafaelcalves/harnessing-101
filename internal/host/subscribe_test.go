package host_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/rafaelcalves/harnessing-101/internal/core/domain"
	"github.com/rafaelcalves/harnessing-101/internal/core/task"
	"github.com/rafaelcalves/harnessing-101/internal/host"
)

// TestCapabilities_SubscribeFromSnapshotCursor is UI-06 exercised through
// the composition boundary rather than the bare engine: GetSnapshot's own
// cursor must be usable as Subscribe's afterCursor with nothing missing.
func TestCapabilities_SubscribeFromSnapshotCursor(t *testing.T) {
	caps, err := host.Open(t.TempDir(), "ws-events", nil)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = caps.Close() }()
	ctx := context.Background()

	if _, err := caps.CreateTask(ctx, "engineer", task.CreateTaskRequest{
		RequestID: "r1", TaskID: "t1", Title: "before",
	}); err != nil {
		t.Fatalf("CreateTask t1: %v", err)
	}
	snap, err := caps.GetSnapshot(ctx)
	if err != nil {
		t.Fatalf("GetSnapshot: %v", err)
	}

	subCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	events, err := caps.Subscribe(subCtx, snap.Cursor)
	if err != nil {
		t.Fatalf("Subscribe(%q): %v", snap.Cursor, err)
	}

	if _, err := caps.CreateTask(ctx, "engineer", task.CreateTaskRequest{
		RequestID: "r2", TaskID: "t2", Title: "after",
	}); err != nil {
		t.Fatalf("CreateTask t2: %v", err)
	}

	select {
	case ev := <-events:
		if len(ev.SubjectIDs) != 1 || ev.SubjectIDs[0] != "t2" {
			t.Fatalf("event subjects = %v, want [t2]", ev.SubjectIDs)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for the post-cursor event")
	}
}

// TestFrontendSession_SubscribeIsWiredThroughCapabilities proves a bound
// session reaches the same event stream as Capabilities — Subscribe is a
// workspace fact, not caller-scoped, unlike ResolveRequest.
func TestFrontendSession_SubscribeIsWiredThroughCapabilities(t *testing.T) {
	caps, err := host.Open(t.TempDir(), "ws-session-events", nil)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = caps.Close() }()
	ctx := context.Background()

	session := host.BindFrontendSession(caps, "engineer")
	events, err := session.Subscribe(ctx, "0")
	if err != nil {
		t.Fatalf("Subscribe: %v", err)
	}

	if _, err := session.CreateTask(ctx, task.CreateTaskRequest{RequestID: "r1", TaskID: "t1", Title: "x"}); err != nil {
		t.Fatalf("CreateTask: %v", err)
	}

	select {
	case ev := <-events:
		if ev.Kind != "TaskCreated" {
			t.Fatalf("event kind = %q, want TaskCreated", ev.Kind)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for the event")
	}
}

// TestCapabilities_SubscribeCursorExpired confirms the composition
// boundary surfaces ErrCursorExpired unchanged rather than translating it
// into something else (e.g. an empty stream).
func TestCapabilities_SubscribeCursorExpired(t *testing.T) {
	caps, err := host.Open(t.TempDir(), "ws-expired", nil)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = caps.Close() }()

	_, err = caps.Subscribe(context.Background(), "not-a-cursor")
	if err == nil {
		t.Fatal("expected an error for an unresumable cursor")
	}
	var derr *domain.Error
	if !errors.As(err, &derr) || derr.Code != domain.ErrCursorExpired {
		t.Fatalf("Subscribe(bad cursor) = %v, want *domain.Error{Code: CursorExpired}", err)
	}
}
