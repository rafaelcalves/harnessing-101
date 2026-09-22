//go:build !windows

package task

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/rafaelcalves/harnessing-101/internal/adapters/clock"
	"github.com/rafaelcalves/harnessing-101/internal/adapters/idsource"
	"github.com/rafaelcalves/harnessing-101/internal/adapters/statestore"
	"github.com/rafaelcalves/harnessing-101/internal/core/domain"
)

// reconcileHelperEnv/reconcileHelperDirEnv make this same test binary
// re-exec itself as a standalone helper process — the same trick
// internal/adapters/statestore/lock_recovery_test.go and
// cmd/harnessing/hold_test.go already use for real cross-process crash
// evidence, applied here to StartRun's own dispatch-ordering window.
const (
	reconcileHelperEnv    = "HARNESSING_RECONCILE_HELPER"
	reconcileHelperDirEnv = "HARNESSING_RECONCILE_DIR"
)

func TestMain(m *testing.M) {
	if os.Getenv(reconcileHelperEnv) == "1" {
		runReconcileCrashHelper(os.Getenv(reconcileHelperDirEnv))
		return
	}
	os.Exit(m.Run())
}

// blockingAfterMarkerSupervisor.Start prints a sync line the parent
// waits on — proof the dispatch-attempted marker already committed,
// since engine.go's StartRun only calls Start after that commit — then
// blocks forever. The parent's SIGKILL lands here, reproducing
// H101-161's real occurrence (a controller dying after the marker,
// before the observation commit) at a real OS process boundary.
type blockingAfterMarkerSupervisor struct{}

func (blockingAfterMarkerSupervisor) Capabilities(context.Context) (any, error) { return nil, nil }

func (blockingAfterMarkerSupervisor) Start(context.Context, domain.RunID, domain.ExecutionSpec, domain.RunParticipationContext) error {
	fmt.Println("start-called")
	select {}
}

func (blockingAfterMarkerSupervisor) Observe(context.Context, domain.RunID) (<-chan any, error) {
	return nil, nil
}

func (blockingAfterMarkerSupervisor) Stop(context.Context, domain.RunID, time.Duration) error {
	return nil
}

func (blockingAfterMarkerSupervisor) Recover(context.Context, domain.RunID) error { return nil }

func runReconcileCrashHelper(dir string) {
	store, err := statestore.Open(dir)
	if err != nil {
		fmt.Println("helper: open failed: " + err.Error())
		os.Exit(1)
	}
	e := NewEngine(store, clock.NewSystem(), idsource.Random{}, "ws-crash-reconcile")
	e.SetProcessSupervisor(blockingAfterMarkerSupervisor{})
	ctx := context.Background()
	if _, err := e.RegisterAgent(ctx, CallerScope{AgentID: "agent-a"}, RegisterAgentRequest{
		RequestID: "reg-a", AgentID: "agent-a", DisplayName: "agent-a",
	}); err != nil {
		fmt.Println("helper: register failed: " + err.Error())
		os.Exit(1)
	}
	if _, err := e.ApproveProfile(ctx, CallerScope{AgentID: "human1", IsHumanReviewer: true}, ApproveProfileRequest{
		RequestID: "approve-a", ProfileID: "profile-a", Spec: domain.ExecutionSpec{Args: []string{"/bin/true"}},
	}); err != nil {
		fmt.Println("helper: approve failed: " + err.Error())
		os.Exit(1)
	}
	_, _ = e.StartRun(ctx, CallerScope{AgentID: "agent-a"}, StartRunRequest{
		RequestID: "r1", RunID: "run-1", AgentID: "agent-a", ProfileID: "profile-a",
	})
	// unreachable: Start (called from inside StartRun) blocks forever.
}

// TestReconcileStuckRuns_NativeCrashBetweenMarkerAndObservation is
// H101-170's native crash-window evidence: a REAL process holding the
// workspace lock is SIGKILLed strictly after it printed proof the
// dispatch-attempted marker committed and strictly before it could
// ever commit an observation — Start never returns, so Step 4 never
// runs. This is H101-161's actual occurrence, reproduced
// deterministically rather than depending on real-world network-policy
// timing. It also proves the ruling's required evidence list: durable
// RecoveryRequired visible through GetRun, a new attempt for the same
// agent refused with no child regardless of new identifiers,
// persistence across a SECOND reopen, and an unrelated agent left
// eligible.
func TestReconcileStuckRuns_NativeCrashBetweenMarkerAndObservation(t *testing.T) {
	dir := t.TempDir()

	helper := exec.Command(os.Args[0], "-test.run=^$")
	helper.Env = append(os.Environ(), reconcileHelperEnv+"=1", reconcileHelperDirEnv+"="+dir)
	stdout, err := helper.StdoutPipe()
	if err != nil {
		t.Fatalf("StdoutPipe: %v", err)
	}
	if err := helper.Start(); err != nil {
		t.Fatalf("start helper: %v", err)
	}
	t.Cleanup(func() { _ = helper.Process.Kill() })

	line, err := bufio.NewReader(stdout).ReadString('\n')
	if err != nil || !strings.Contains(line, "start-called") {
		t.Fatalf("helper did not confirm Start was called (line=%q, err=%v)", line, err)
	}

	if err := helper.Process.Kill(); err != nil {
		t.Fatalf("kill helper: %v", err)
	}
	_ = helper.Wait()

	// Reopen #1 through the real store — the kernel needs a moment to
	// finish descriptor teardown after SIGKILL; there is no lock
	// timeout to wait out, only this small window.
	var store *statestore.FileStore
	deadline := time.Now().Add(5 * time.Second)
	for {
		store, err = statestore.Open(dir)
		if err == nil {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("could not reopen after helper kill: %v", err)
		}
		time.Sleep(20 * time.Millisecond)
	}
	sup := &fakeSupervisor{}
	e := NewEngine(store, clock.NewSystem(), idsource.Random{}, "ws-crash-reconcile")
	e.SetProcessSupervisor(sup)
	ctx := context.Background()

	run, err := e.GetRun(ctx, "run-1")
	if err != nil || run.State != domain.RunStarting {
		t.Fatalf("precondition after real kill: run-1 = %+v, err = %v; want State Starting", run, err)
	}

	// This is what host.Open does automatically on every fresh Open;
	// called explicitly here since this test drives the engine
	// directly (host-level wiring is covered separately in
	// internal/host).
	if err := e.ReconcileStuckRuns(ctx); err != nil {
		t.Fatalf("ReconcileStuckRuns: %v", err)
	}

	run, err = e.GetRun(ctx, "run-1")
	if err != nil {
		t.Fatalf("GetRun after reconcile: %v", err)
	}
	if run.State != domain.RunRecoveryRequired {
		t.Fatalf("run-1.State = %q after native crash + reconcile, want RecoveryRequired", run.State)
	}

	// Evidence: a new attempt for the SAME agent is refused with no
	// child, whatever new runID/profile/requestID it uses.
	_, err = e.StartRun(ctx, CallerScope{AgentID: "agent-a"}, StartRunRequest{
		RequestID: "r2-new", RunID: "run-2-new", AgentID: "agent-a", ProfileID: "profile-b-new",
	})
	startRunMustErrorCode(t, err, domain.ErrConflict)
	if sup.calls != 0 {
		t.Fatalf("supervisor.Start called %d times for agent-a after RecoveryRequired; want 0", sup.calls)
	}

	if err := store.Close(); err != nil {
		t.Fatalf("close after reconcile: %v", err)
	}

	// Reopen #2 — persistence across ANOTHER reopen, with nothing left
	// Starting to reconcile this time (idempotent).
	store2, err := statestore.Open(dir)
	if err != nil {
		t.Fatalf("second reopen: %v", err)
	}
	defer func() { _ = store2.Close() }()
	e2 := NewEngine(store2, clock.NewSystem(), idsource.Random{}, "ws-crash-reconcile")
	e2.SetProcessSupervisor(&fakeSupervisor{})
	if err := e2.ReconcileStuckRuns(ctx); err != nil {
		t.Fatalf("ReconcileStuckRuns on second reopen: %v", err)
	}
	run, err = e2.GetRun(ctx, "run-1")
	if err != nil || run.State != domain.RunRecoveryRequired {
		t.Fatalf("run-1 after second reopen = %+v, err = %v; want it to stay RecoveryRequired", run, err)
	}

	// An unrelated, eligible agent is not permanently blocked by
	// agent-a's RecoveryRequired record. profile-a is already approved
	// (by the helper process) — only register the new agent.
	if _, err := e2.RegisterAgent(ctx, CallerScope{AgentID: "agent-b"}, RegisterAgentRequest{
		RequestID: "reg-b", AgentID: "agent-b", DisplayName: "agent-b",
	}); err != nil {
		t.Fatalf("RegisterAgent agent-b: %v", err)
	}
	if _, err := e2.StartRun(ctx, CallerScope{AgentID: "agent-b"}, StartRunRequest{
		RequestID: "r3", RunID: "run-3", AgentID: "agent-b", ProfileID: "profile-a",
	}); err != nil {
		t.Fatalf("StartRun for unrelated agent-b: %v", err)
	}
}
