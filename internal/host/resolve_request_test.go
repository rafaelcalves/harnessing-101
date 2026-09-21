package host_test

import (
	"context"
	"errors"
	"testing"

	"github.com/rafaelcalves/harnessing-101/internal/core/domain"
	"github.com/rafaelcalves/harnessing-101/internal/core/task"
	"github.com/rafaelcalves/harnessing-101/internal/host"
)

// TestCapabilities_ResolveRequest exercises H101-64's promotion through
// Capabilities directly: Confirmed after a normal commit, Absent for an
// unknown request ID, and cross-caller isolation (another agent's
// request ID is indistinguishable from Absent — ledger keying scopes
// the lookup, per the port's own doc comment).
func TestCapabilities_ResolveRequest(t *testing.T) {
	caps, err := host.Open(t.TempDir(), "ws-resolve", nil)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = caps.Close() }()
	ctx := context.Background()

	receipt, err := caps.CreateTask(ctx, "engineer", task.CreateTaskRequest{
		RequestID: "r1", TaskID: "t1", Title: "x",
	})
	if err != nil {
		t.Fatalf("CreateTask: %v", err)
	}

	resolved, err := caps.ResolveRequest(ctx, "engineer", "r1")
	if err != nil {
		t.Fatalf("ResolveRequest: %v", err)
	}
	if resolved != receipt {
		t.Fatalf("resolved receipt = %+v, want the original %+v", resolved, receipt)
	}

	t.Run("unknown request ID is Absent (NotFound)", func(t *testing.T) {
		_, err := caps.ResolveRequest(ctx, "engineer", "never-existed")
		var derr *domain.Error
		if !errors.As(err, &derr) || derr.Code != domain.ErrNotFound {
			t.Fatalf("ResolveRequest(unknown) = %v, want *domain.Error{Code: NotFound}", err)
		}
	})

	t.Run("another caller's request ID is also Absent, not exposed", func(t *testing.T) {
		_, err := caps.ResolveRequest(ctx, "someone-else", "r1")
		var derr *domain.Error
		if !errors.As(err, &derr) || derr.Code != domain.ErrNotFound {
			t.Fatalf("cross-caller ResolveRequest = %v, want NotFound (not leaked)", err)
		}
	})
}

// TestFrontendSession_ResolveRequestIsBoundNotParameterized proves the
// H101-64 qualification in code: a session bound to one principal can
// only ever resolve ITS OWN requests. There is no parameter on
// FrontendSession.ResolveRequest a caller could use to ask about
// someone else's — the identity comes from BindFrontendSession, once,
// and nowhere else.
func TestFrontendSession_ResolveRequestIsBoundNotParameterized(t *testing.T) {
	caps, err := host.Open(t.TempDir(), "ws-session-resolve", nil)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = caps.Close() }()
	ctx := context.Background()

	engineer := host.BindFrontendSession(caps, "engineer")
	other := host.BindFrontendSession(caps, "someone-else")

	receipt, err := engineer.CreateTask(ctx, task.CreateTaskRequest{RequestID: "r1", TaskID: "t1", Title: "x"})
	if err != nil {
		t.Fatalf("CreateTask: %v", err)
	}

	resolved, err := engineer.ResolveRequest(ctx, "r1")
	if err != nil {
		t.Fatalf("engineer.ResolveRequest(own request): %v", err)
	}
	if resolved != receipt {
		t.Fatalf("resolved = %+v, want %+v", resolved, receipt)
	}

	// The other session, bound to a different principal, cannot reach
	// it — same request ID, different session identity, Absent.
	_, err = other.ResolveRequest(ctx, "r1")
	var derr *domain.Error
	if !errors.As(err, &derr) || derr.Code != domain.ErrNotFound {
		t.Fatalf("other session ResolveRequest = %v, want NotFound", err)
	}
}
