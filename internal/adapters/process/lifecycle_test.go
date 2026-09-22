package process_test

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"syscall"
	"testing"
	"time"

	"github.com/rafaelcalves/harnessing-101/internal/adapters/process"
	"github.com/rafaelcalves/harnessing-101/internal/core/domain"
)

// recorder is Kelly's "test-only ordered lifecycle recorder" (H101-217
// §Lifecycle recorder): the ONLY way Layer B can distinguish "authority
// closed before reap" from "reaped first and nothing bad happened this
// run" -- a CLI black-box test structurally cannot tell those apart.
type recorder struct {
	mu     sync.Mutex
	events []string
}

func (r *recorder) hook(_ domain.RunID, event string) {
	r.mu.Lock()
	r.events = append(r.events, event)
	r.mu.Unlock()
}

func (r *recorder) snapshot() []string {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]string(nil), r.events...)
}

// TestSupervisor_StopOrderingEscalationAndDecoy is N1 (per-signal
// recheck), N2 (close before reap), N5 (bounded grace + forced path
// when the leader ignores SIGTERM), and the decoy negative (h101-128
// "never signal an unrelated process") -- all in one native scenario,
// since they share the same one Stop call.
func TestSupervisor_StopOrderingEscalationAndDecoy(t *testing.T) {
	orig := process.StartupWindow
	process.StartupWindow = 50 * time.Millisecond
	defer func() { process.StartupWindow = orig }()

	rec := &recorder{}
	sup := &process.Supervisor{LifecycleObserver: rec.hook}

	// Decoy: an unrelated same-user process that is its own group
	// leader. Stop must never reach it.
	decoy := exec.Command("sleep", "5")
	decoy.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	if err := decoy.Start(); err != nil {
		t.Fatalf("starting decoy: %v", err)
	}
	defer func() { _ = decoy.Process.Kill(); _, _ = decoy.Process.Wait() }()

	// Leader ignores SIGTERM entirely, forcing Stop's forced (SIGKILL)
	// path -- only SIGKILL can end it. `exec sleep 100` replaces the
	// shell's own process image, but POSIX exec semantics preserve a
	// SIG_IGN disposition across exec (only caught/default ones
	// reset), so the resulting process is deterministically
	// SIGTERM-proof -- unlike a shell loop respawning short-lived
	// children, which is flaky: the loop's own foreground child can
	// die to the SAME group SIGTERM independently of the shell's
	// trap, sometimes ending the whole leader on SIGTERM alone.
	//
	// H101-232 (Kelly): `trap ""` alone is not enough -- 3/20 package-only
	// runs showed SIGTERM arriving before the trap installed, so Stop
	// sometimes took the graceful path and never proved escalation at
	// all. The leader now attests immunity itself, same class as
	// H101-221's `worker_ready\n`: it writes an exact `term_ignored\n`
	// to a sync file ONLY after the trap is installed and immediately
	// before exec-ing into the SIGTERM-proof sleep, and Stop is not
	// called until that attestation is observed.
	attestFile := filepath.Join(t.TempDir(), "term-ignored")
	leaderScript := fmt.Sprintf(`trap "" TERM; printf 'term_ignored\n' > %q; exec sleep 100`, attestFile)
	err := sup.Start(context.Background(), "run-escalate",
		domain.ExecutionSpec{Args: []string{"sh", "-c", leaderScript}},
		domain.RunParticipationContext{})
	if err != nil {
		t.Fatalf("Start: %v", err)
	}

	deadline := time.Now().Add(30 * time.Second)
	for {
		data, readErr := os.ReadFile(attestFile)
		if readErr == nil && string(data) == "term_ignored\n" {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("leader never attested SIGTERM-immunity within 30s (last read: data=%q err=%v)", data, readErr)
		}
		time.Sleep(50 * time.Millisecond)
	}

	stopErr := sup.Stop(context.Background(), "run-escalate", 150*time.Millisecond)
	if stopErr != nil {
		t.Fatalf("Stop: %v", stopErr)
	}

	if err := syscall.Kill(decoy.Process.Pid, 0); err != nil {
		t.Fatalf("decoy pid %d no longer reachable after Stop: %v (an unrelated process was signaled)", decoy.Process.Pid, err)
	}

	// H101-232 assertion shape: an exact DeepEqual on the full event
	// list is wrong for a legitimately dual-path design (N2's
	// authority-closed-before-reap holds regardless of which path won
	// the race to observe termination). Escalation is now guaranteed by
	// the attestation above, so pre_sigkill_check/sigkill_sent are
	// mandatory on THIS scenario -- their absence is a wrong-path
	// failure, not tolerated slack. N1/N2 are checked as subsequence
	// pairs, not a single total order.
	got := rec.snapshot()
	index := func(event string) int {
		for i, ev := range got {
			if ev == event {
				return i
			}
		}
		return -1
	}
	before := func(first, second string) {
		fi, si := index(first), index(second)
		if fi < 0 {
			t.Fatalf("%q never appeared: %v", first, got)
		}
		if si < 0 {
			t.Fatalf("%q never appeared: %v", second, got)
		}
		if fi >= si {
			t.Fatalf("%q (at %d) did not come before %q (at %d): %v", first, fi, second, si, got)
		}
	}
	if index("pre_sigkill_check") < 0 || index("sigkill_sent") < 0 {
		t.Fatalf("escalation scenario recorded no SIGKILL events -- leader took the graceful path despite attestation: %v", got)
	}
	before("pre_sigterm_check", "sigterm_sent") // N1
	before("pre_sigkill_check", "sigkill_sent") // N1
	before("authority_closed", "child_reaped")  // N2
	if len(got) == 0 || got[0] != "stop_claimed" {
		t.Fatalf("event order = %v, want it to start with stop_claimed", got)
	}
}

// TestSupervisor_StopGracefulPathSkipsForcedSignal proves the OTHER
// shape: a leader that honors SIGTERM never sees a SIGKILL, and a
// second Stop call after the run has already terminated is a stale
// callback that produces no new signal events (N3, N4) -- the run is
// simply already gone.
func TestSupervisor_StopGracefulPathSkipsForcedSignal(t *testing.T) {
	orig := process.StartupWindow
	process.StartupWindow = 50 * time.Millisecond
	defer func() { process.StartupWindow = orig }()

	rec := &recorder{}
	sup := &process.Supervisor{LifecycleObserver: rec.hook}

	err := sup.Start(context.Background(), "run-graceful",
		domain.ExecutionSpec{Args: []string{"sleep", "30"}}, domain.RunParticipationContext{})
	if err != nil {
		t.Fatalf("Start: %v", err)
	}

	if err := sup.Stop(context.Background(), "run-graceful", 2*time.Second); err != nil {
		t.Fatalf("Stop: %v", err)
	}

	got := rec.snapshot()
	for _, forbidden := range []string{"pre_sigkill_check", "sigkill_sent"} {
		for _, ev := range got {
			if ev == forbidden {
				t.Fatalf("graceful SIGTERM path recorded %q, want it never escalated: %v", forbidden, got)
			}
		}
	}
	if len(got) == 0 || got[0] != "stop_claimed" {
		t.Fatalf("event order = %v, want it to start with stop_claimed", got)
	}

	before := len(rec.snapshot())
	staleErr := sup.Stop(context.Background(), "run-graceful", 2*time.Second)
	if staleErr == nil {
		t.Fatalf("second Stop on an already-terminated run returned nil, want a refusal (process already terminated)")
	}
	if got := rec.snapshot(); len(got) != before {
		t.Fatalf("stale second Stop call recorded new events %v, want none", got[before:])
	}
}
