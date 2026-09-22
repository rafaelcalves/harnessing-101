package process_test

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/rafaelcalves/harnessing-101/internal/adapters/process"
	"github.com/rafaelcalves/harnessing-101/internal/core/domain"
)

// mustErrorCode asserts err is a *domain.Error with the given Code.
func mustErrorCode(t *testing.T, err error, want domain.ErrorCode) {
	t.Helper()
	derr, ok := err.(*domain.Error)
	if !ok {
		t.Fatalf("error = %T %v, want *domain.Error with code %s", err, err, want)
	}
	if derr.Code != want {
		t.Fatalf("error code = %s, want %s (detail: %s)", derr.Code, want, derr.Detail)
	}
}

// TestSupervisor_MissingExecutableIsClassified is this package's own
// direct proof of the classification cmd/harnessing's CLI-level test
// only observes end to end — a unit test against the adapter itself,
// not through the whole command surface.
func TestSupervisor_MissingExecutableIsClassified(t *testing.T) {
	sup := &process.Supervisor{}
	err := sup.Start(context.Background(), "run-1",
		domain.ExecutionSpec{Args: []string{"/definitely/not/a/real/executable-xyz"}},
		domain.RunParticipationContext{})
	mustErrorCode(t, err, domain.ErrMissingTool)
}

// TestSupervisor_EmptyArgsIsInvalidArgument proves a spec with no
// executable at all is rejected before ever touching os/exec.
func TestSupervisor_EmptyArgsIsInvalidArgument(t *testing.T) {
	sup := &process.Supervisor{}
	err := sup.Start(context.Background(), "run-1", domain.ExecutionSpec{}, domain.RunParticipationContext{})
	mustErrorCode(t, err, domain.ErrInvalidArgument)
}

// TestSupervisor_UnsupportedContextTransportIsExplicit is R3: an
// unrecognized ContextTransport value must fail loudly, never silently
// skip participation context and still report success.
func TestSupervisor_UnsupportedContextTransportIsExplicit(t *testing.T) {
	sup := &process.Supervisor{}
	err := sup.Start(context.Background(), "run-1",
		domain.ExecutionSpec{Args: []string{"true"}, ContextTransport: "stdin"},
		domain.RunParticipationContext{})
	mustErrorCode(t, err, domain.ErrUnsupported)
}

// TestSupervisor_FastExitIsSpawnFailed proves a process that exits
// immediately (within StartupWindow) is reported honestly rather than
// as a healthy Running participant — the fixture-less half of D9/D10.
func TestSupervisor_FastExitIsSpawnFailed(t *testing.T) {
	orig := process.StartupWindow
	process.StartupWindow = 100 * time.Millisecond
	defer func() { process.StartupWindow = orig }()

	sup := &process.Supervisor{}
	// /bin/true (or /usr/bin/true) exits 0 immediately — a real binary
	// that still cannot count as "participating."
	err := sup.Start(context.Background(), "run-1",
		domain.ExecutionSpec{Args: []string{"true"}}, domain.RunParticipationContext{})
	mustErrorCode(t, err, domain.ErrSpawnFailed)
}

// TestSupervisor_LongLivedProcessReturnsNil proves a process still
// running past StartupWindow is treated as a healthy start, not killed
// or misreported.
func TestSupervisor_LongLivedProcessReturnsNil(t *testing.T) {
	orig := process.StartupWindow
	process.StartupWindow = 100 * time.Millisecond
	defer func() { process.StartupWindow = orig }()

	sup := &process.Supervisor{}
	err := sup.Start(context.Background(), "run-1",
		domain.ExecutionSpec{Args: []string{"sleep", "2"}}, domain.RunParticipationContext{})
	if err != nil {
		t.Fatalf("Start: %v, want nil (process still running past the startup window)", err)
	}
}

// TestSupervisor_ContextFileTransportWritesParticipationContext is R2's
// own unit-level proof: the exact fields a launched tool receives via
// context-file, verified directly rather than only through the CLI's
// end-to-end fixture test.
func TestSupervisor_ContextFileTransportWritesParticipationContext(t *testing.T) {
	orig := process.StartupWindow
	process.StartupWindow = 200 * time.Millisecond
	defer func() { process.StartupWindow = orig }()

	dir := t.TempDir()
	sup := &process.Supervisor{ContextDir: dir}
	taskID := domain.TaskID("task-1")
	peer := domain.AgentID("agent-b")

	err := sup.Start(context.Background(), "run-1",
		domain.ExecutionSpec{Args: []string{"sleep", "2"}, ContextTransport: "context-file"},
		domain.RunParticipationContext{
			WorkspaceID: "ws-1", RunID: "run-1", AgentID: "agent-a", TaskID: &taskID, PeerAgentID: &peer,
		})
	if err != nil {
		t.Fatalf("Start: %v", err)
	}

	path := filepath.Join(dir, "run-1.context.json")
	data, readErr := os.ReadFile(path)
	if readErr != nil {
		t.Fatalf("reading context file: %v", readErr)
	}
	var got map[string]any
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("context file is not valid JSON: %v", err)
	}
	if got["workspaceId"] != "ws-1" || got["runId"] != "run-1" || got["agentId"] != "agent-a" || got["taskId"] != "task-1" || got["peerAgentId"] != "agent-b" {
		t.Fatalf("context file = %v, missing or wrong expected fields", got)
	}
}
