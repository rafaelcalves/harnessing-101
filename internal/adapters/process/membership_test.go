package process_test

import (
	"context"
	"os/exec"
	"syscall"
	"testing"
	"time"

	"github.com/rafaelcalves/harnessing-101/internal/adapters/process"
	"github.com/rafaelcalves/harnessing-101/internal/core/domain"
)

// startWorkerInGroup spawns a real, independently-tracked process into
// pgid so a native test can shape a mode-B survivor without going
// through the fixture binary at all. The test owns killing it.
func startWorkerInGroup(t *testing.T, pgid int) *exec.Cmd {
	t.Helper()
	worker := exec.Command("sh", "-c", "sleep 30")
	worker.SysProcAttr = &syscall.SysProcAttr{Setpgid: true, Pgid: pgid}
	if err := worker.Start(); err != nil {
		t.Fatalf("starting worker into pgid %d: %v", pgid, err)
	}
	return worker
}

// TestSupervisor_NaturalCompletionWaitsForSurvivorThenReaps is B1, B2
// and B5 in one native, mode-B-shaped scenario: the leader exits on
// its own (no Stop ever called) while a same-group worker survives.
// Reaping must wait for POSITIVE other-member-absence evidence, never
// a boolean group probe (which the retained zombie leader alone keeps
// positive) -- and once the worker is actually gone, natural
// finalization reaps safely, closing authority before consuming the
// final status.
func TestSupervisor_NaturalCompletionWaitsForSurvivorThenReaps(t *testing.T) {
	orig := process.StartupWindow
	process.StartupWindow = 50 * time.Millisecond
	defer func() { process.StartupWindow = orig }()

	rec := &recorder{}
	sup := &process.Supervisor{LifecycleObserver: rec.hook}

	err := sup.Start(context.Background(), "run-natural-survivor",
		domain.ExecutionSpec{Args: []string{"sh", "-c", "sleep 0.15"}}, domain.RunParticipationContext{})
	if err != nil {
		t.Fatalf("Start: %v", err)
	}

	leaderInfo, lerr := sup.RunInfo("run-natural-survivor")
	if lerr != nil {
		t.Fatalf("RunInfo: %v", lerr)
	}
	worker := startWorkerInGroup(t, leaderInfo.Pid)
	defer func() { _ = worker.Process.Kill(); _, _ = worker.Process.Wait() }()

	// B1: leader_terminated_observed must appear (the leader dies on
	// its own, ~150ms after Start), and other_members_absent_confirmed
	// must NOT appear while the worker is still alive.
	deadline := time.Now().Add(3 * time.Second)
	sawTerminated := false
	for time.Now().Before(deadline) {
		got := rec.snapshot()
		for _, ev := range got {
			if ev == "leader_terminated_observed" {
				sawTerminated = true
			}
			if ev == "other_members_absent_confirmed" {
				t.Fatalf("other_members_absent_confirmed fired while the worker (pid %d) is still alive: %v", worker.Process.Pid, got)
			}
		}
		if sawTerminated {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	if !sawTerminated {
		t.Fatalf("leader_terminated_observed never appeared: %v", rec.snapshot())
	}
	if err := syscall.Kill(worker.Process.Pid, 0); err != nil {
		t.Fatalf("worker no longer alive while asserting B1: %v", err)
	}

	// Now let the worker go -- B2/B5: only once it is truly gone should
	// natural finalization confirm absence and reap, authority closed
	// strictly before the reap.
	_ = worker.Process.Kill()
	_, _ = worker.Process.Wait()

	deadline = time.Now().Add(5 * time.Second)
	for {
		got := rec.snapshot()
		if len(got) > 0 && got[len(got)-1] == "child_reaped" {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("natural finalization never reached child_reaped: %v", got)
		}
		time.Sleep(20 * time.Millisecond)
	}

	got := rec.snapshot()
	idx := map[string]int{}
	for i, ev := range got {
		if _, seen := idx[ev]; !seen {
			idx[ev] = i
		}
	}
	confirmedAt, ok := idx["other_members_absent_confirmed"]
	if !ok {
		t.Fatalf("other_members_absent_confirmed never fired: %v", got)
	}
	closedAt, ok := idx["authority_closed"]
	if !ok || closedAt < confirmedAt {
		t.Fatalf("authority_closed did not follow other_members_absent_confirmed: %v", got)
	}
	reapedAt, ok := idx["child_reaped"]
	if !ok || reapedAt < closedAt {
		t.Fatalf("child_reaped did not follow authority_closed: %v", got)
	}
}

// TestSupervisor_NaturalCompletionRefusesOnUncertainMembership is B3:
// when the membership observation itself cannot be trusted, the
// natural path must retain the binding and expose the uncertainty --
// never a speculative reap -- and Stop must still be able to
// terminate the run afterward.
func TestSupervisor_NaturalCompletionRefusesOnUncertainMembership(t *testing.T) {
	orig := process.StartupWindow
	process.StartupWindow = 50 * time.Millisecond
	defer func() { process.StartupWindow = orig }()

	rec := &recorder{}
	sup := &process.Supervisor{LifecycleObserver: rec.hook}

	if err := sup.InjectMembershipUncertainty(); err != nil {
		t.Fatalf("InjectMembershipUncertainty: %v", err)
	}
	defer sup.RestoreMembershipCheck()

	err := sup.Start(context.Background(), "run-natural-uncertain",
		domain.ExecutionSpec{Args: []string{"sh", "-c", "sleep 0.1"}}, domain.RunParticipationContext{})
	if err != nil {
		t.Fatalf("Start: %v", err)
	}

	deadline := time.Now().Add(3 * time.Second)
	for {
		got := rec.snapshot()
		var sawRefused bool
		for _, ev := range got {
			if ev == "child_reaped" {
				t.Fatalf("child_reaped fired on the uncertain path alone: %v", got)
			}
			if ev == "natural_reap_refused_uncertain" {
				sawRefused = true
			}
		}
		if sawRefused {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("natural_reap_refused_uncertain never appeared: %v", got)
		}
		time.Sleep(20 * time.Millisecond)
	}

	sup.RestoreMembershipCheck()

	if err := sup.Stop(context.Background(), "run-natural-uncertain", 200*time.Millisecond); err != nil {
		t.Fatalf("Stop after uncertain natural refusal: %v, want it to still succeed", err)
	}
}
