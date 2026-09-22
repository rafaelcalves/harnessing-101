package task

import (
	"context"
	"testing"

	"github.com/rafaelcalves/harnessing-101/internal/core/domain"
)

// TestReconcileStuckRuns_TransitionsStuckStartingToRecoveryRequired is
// H101-170's engine-level proof of the minimal item 4 slice: a run left
// Starting by a controller that crashed between the dispatch-attempted
// marker and the observation commit (H101-135's ambiguous window, the
// real occurrence behind H101-161) is durably reconciled to
// RecoveryRequired — never silently left Starting, never guessed into
// Running. Reuses the exact in-process crash simulation
// TestStartRun_DispatchMarkerAmbiguous_NoAutoRedispatch already
// established (H101-135/D12) to produce the stuck run; this file's own
// job starts where that one stops — turning "stuck" into "reconciled."
func TestReconcileStuckRuns_TransitionsStuckStartingToRecoveryRequired(t *testing.T) {
	sup := &fakeSupervisor{}
	e := newStartRunEngine(t, sup)
	ctx := context.Background()
	mustRegisterAndApprove(t, ctx, e, "agent-a", "profile-a")

	orig := afterDispatchMarkerBeforeStart
	afterDispatchMarkerBeforeStart = func() { panic("simulated crash between marker and Start") }
	func() {
		defer func() { _ = recover() }()
		_, _ = e.StartRun(ctx, CallerScope{AgentID: "agent-a"}, StartRunRequest{
			RequestID: "r1", RunID: "run-1", AgentID: "agent-a", ProfileID: "profile-a",
		})
	}()
	afterDispatchMarkerBeforeStart = orig

	run, err := e.GetRun(ctx, "run-1")
	if err != nil || run.State != domain.RunStarting {
		t.Fatalf("precondition: run-1 = %+v, err = %v; want State Starting before reconciliation", run, err)
	}

	if err := e.ReconcileStuckRuns(ctx); err != nil {
		t.Fatalf("ReconcileStuckRuns: %v", err)
	}

	run, err = e.GetRun(ctx, "run-1")
	if err != nil {
		t.Fatalf("GetRun after reconcile: %v", err)
	}
	if run.State != domain.RunRecoveryRequired {
		t.Fatalf("run-1.State = %q after reconcile, want RecoveryRequired", run.State)
	}

	op, err := e.GetOperation(ctx, domain.OperationID("run-1-start"))
	if err != nil {
		t.Fatalf("GetOperation: %v", err)
	}
	if op.State != domain.OperationRecoveryRequired {
		t.Fatalf("start operation.State = %q after reconcile, want RecoveryRequired", op.State)
	}

	// R3/D10-style evidence #3: a new run/request/profile for the SAME
	// agent is refused with no child, no evasion by changing any of
	// those identifiers.
	_, err = e.StartRun(ctx, CallerScope{AgentID: "agent-a"}, StartRunRequest{
		RequestID: "r2-new-request", RunID: "run-2-new-runid", AgentID: "agent-a", ProfileID: "profile-b-new-profile",
	})
	startRunMustErrorCode(t, err, domain.ErrConflict)
	if sup.calls != 0 {
		t.Fatalf("supervisor.Start called %d times for agent-a after RecoveryRequired; want 0", sup.calls)
	}

	// Evidence #5: an unrelated eligible agent is NOT permanently
	// blocked by agent-a's RecoveryRequired record.
	mustRegisterAndApprove(t, ctx, e, "agent-b", "profile-a")
	if _, err := e.StartRun(ctx, CallerScope{AgentID: "agent-b"}, StartRunRequest{
		RequestID: "r3", RunID: "run-3", AgentID: "agent-b", ProfileID: "profile-a",
	}); err != nil {
		t.Fatalf("StartRun for unrelated agent-b: %v", err)
	}
	if sup.calls != 1 {
		t.Fatalf("supervisor.Start called %d times total; want exactly 1 (only agent-b's run)", sup.calls)
	}
}

// TestReconcileStuckRuns_LeavesAnUnrelatedRunsOperationAlone proves
// ReconcileStuckRuns's per-run commit only ever touches the ONE run
// (and its start operation) it found genuinely Starting — an unrelated
// agent's already-Succeeded run/operation survives a reconcile pass
// untouched. This does NOT exercise reconcileStuckRun's own
// found-but-not-Pending/Running guard on the SAME run's operation: that
// guard is defensive against a combination (Run.State == Starting
// while its own start Operation is already terminal) that cannot
// currently arise, because Step 2/Step 4 always move a run's state and
// its start operation's state together — never independently. Recorded
// here rather than left implicit: the guard's own test would be
// vacuous, so this asserts the weaker, real claim instead.
func TestReconcileStuckRuns_LeavesAnUnrelatedRunsOperationAlone(t *testing.T) {
	sup := &fakeSupervisor{}
	e := newStartRunEngine(t, sup)
	ctx := context.Background()
	mustRegisterAndApprove(t, ctx, e, "agent-done", "profile-a")

	if _, err := e.StartRun(ctx, CallerScope{AgentID: "agent-done"}, StartRunRequest{
		RequestID: "r1", RunID: "run-done", AgentID: "agent-done", ProfileID: "profile-a",
	}); err != nil {
		t.Fatalf("StartRun agent-done: %v", err)
	}

	// A second, unrelated agent gets stuck Starting via the same
	// crash-simulation technique.
	mustRegisterAndApprove(t, ctx, e, "agent-stuck", "profile-a")
	orig := afterDispatchMarkerBeforeStart
	afterDispatchMarkerBeforeStart = func() { panic("simulated crash between marker and Start") }
	func() {
		defer func() { _ = recover() }()
		_, _ = e.StartRun(ctx, CallerScope{AgentID: "agent-stuck"}, StartRunRequest{
			RequestID: "r2", RunID: "run-stuck", AgentID: "agent-stuck", ProfileID: "profile-a",
		})
	}()
	afterDispatchMarkerBeforeStart = orig

	if err := e.ReconcileStuckRuns(ctx); err != nil {
		t.Fatalf("ReconcileStuckRuns: %v", err)
	}

	doneRun, err := e.GetRun(ctx, "run-done")
	if err != nil {
		t.Fatalf("GetRun run-done: %v", err)
	}
	if doneRun.State != domain.RunRunning {
		t.Fatalf("run-done.State = %q after an unrelated reconcile pass, want it to stay Running", doneRun.State)
	}
	doneOp, err := e.GetOperation(ctx, domain.OperationID("run-done-start"))
	if err != nil {
		t.Fatalf("GetOperation run-done: %v", err)
	}
	if doneOp.State != domain.OperationSucceeded {
		t.Fatalf("run-done's start operation.State = %q after an unrelated reconcile pass, want it to stay Succeeded", doneOp.State)
	}

	stuckOp, err := e.GetOperation(ctx, domain.OperationID("run-stuck-start"))
	if err != nil {
		t.Fatalf("GetOperation run-stuck: %v", err)
	}
	if stuckOp.State != domain.OperationRecoveryRequired {
		t.Fatalf("run-stuck's start operation.State = %q, want RecoveryRequired", stuckOp.State)
	}
}
