package process

import (
	"context"
	"sync"
	"syscall"
	"testing"
	"time"

	"github.com/rafaelcalves/harnessing-101/internal/core/domain"
)

// TestSupervisor_ConcurrentStopSendsOneSignalSet is N4's "concurrent"
// half: two Stop calls racing on the same run must not each send their
// own SIGTERM/SIGKILL -- only the first claimant signals; the second
// only waits and observes the same outcome.
func TestSupervisor_ConcurrentStopSendsOneSignalSet(t *testing.T) {
	orig := StartupWindow
	StartupWindow = 50 * time.Millisecond
	defer func() { StartupWindow = orig }()

	var mu sync.Mutex
	counts := map[string]int{}
	sup := &Supervisor{LifecycleObserver: func(_ domain.RunID, event string) {
		mu.Lock()
		counts[event]++
		mu.Unlock()
	}}

	if err := sup.Start(context.Background(), "run-concurrent",
		domain.ExecutionSpec{Args: []string{"sleep", "30"}}, domain.RunParticipationContext{}); err != nil {
		t.Fatalf("Start: %v", err)
	}

	var wg sync.WaitGroup
	errs := make([]error, 2)
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			errs[i] = sup.Stop(context.Background(), "run-concurrent", 2*time.Second)
		}(i)
	}
	wg.Wait()

	for i, err := range errs {
		if err != nil {
			t.Fatalf("concurrent Stop[%d]: %v, want both to observe a clean termination", i, err)
		}
	}

	mu.Lock()
	defer mu.Unlock()
	if counts["sigterm_sent"] != 1 {
		t.Fatalf("sigterm_sent recorded %d times, want exactly 1 (one caller signals, the other only waits)", counts["sigterm_sent"])
	}
	if errs[0] != errs[1] {
		t.Fatalf("concurrent callers observed different results: %v vs %v -- H101-224 B6 requires the SAME terminal outcome", errs[0], errs[1])
	}
}

// TestSupervisor_ConcurrentCallersObserveSameTerminalFailure is B6/B7's
// failure half: a waiter must report the run's ACTUAL terminal
// outcome, including a failure, never infer success merely because
// rec.closed became true. Faults the observation mechanism (same seam
// as the hard-refusal test) so the claiming call's own Stop fails,
// then asserts the concurrent waiter gets the identical *domain.Error,
// not a bare nil.
func TestSupervisor_ConcurrentCallersObserveSameTerminalFailure(t *testing.T) {
	orig := StartupWindow
	StartupWindow = 50 * time.Millisecond
	defer func() { StartupWindow = orig }()

	origPeek := waitidPeekFn.Load()
	defer waitidPeekFn.Store(origPeek)

	sup := &Supervisor{}
	if err := sup.Start(context.Background(), "run-concurrent-fail",
		domain.ExecutionSpec{Args: []string{"sleep", "30"}}, domain.RunParticipationContext{}); err != nil {
		t.Fatalf("Start: %v", err)
	}
	waitidPeekFn.Store(waitidPeekFunc(func(pid int) (bool, error) { return false, syscall.EINVAL }))

	var wg sync.WaitGroup
	errs := make([]error, 2)
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			errs[i] = sup.Stop(context.Background(), "run-concurrent-fail", 20*time.Millisecond)
		}(i)
	}
	wg.Wait()

	mustErrorCode(t, errs[0], domain.ErrRecoveryRequired)
	mustErrorCode(t, errs[1], domain.ErrRecoveryRequired)
	d0, d1 := errs[0].(*domain.Error), errs[1].(*domain.Error)
	if d0.Detail != d1.Detail {
		t.Fatalf("concurrent callers observed different failure details: %q vs %q, want the identical recorded outcome", d0.Detail, d1.Detail)
	}

	waitidPeekFn.Store(origPeek)
}

// TestSupervisor_StopHardRefusalWhenBindingCannotBeObserved is N6: if
// the non-reaping observation mechanism itself cannot be trusted, Stop
// must refuse with RecoveryRequired -- never fall back to a
// best-effort signal against the bare pgid it already sent, and never
// report success it did not actually observe. This injects the fault
// through the package's own waitidPeek seam rather than needing an
// actually-broken kernel.
func TestSupervisor_StopHardRefusalWhenBindingCannotBeObserved(t *testing.T) {
	orig := StartupWindow
	StartupWindow = 50 * time.Millisecond
	defer func() { StartupWindow = orig }()

	origPeek := waitidPeekFn.Load()
	defer waitidPeekFn.Store(origPeek)

	sup := &Supervisor{}
	if err := sup.Start(context.Background(), "run-unobservable",
		domain.ExecutionSpec{Args: []string{"sleep", "30"}}, domain.RunParticipationContext{}); err != nil {
		t.Fatalf("Start: %v", err)
	}

	// From here on, this adapter can no longer observe exit without
	// reaping -- simulate a target where the binding is unusable.
	waitidPeekFn.Store(waitidPeekFunc(func(pid int) (bool, error) { return false, syscall.EINVAL }))

	err := sup.Stop(context.Background(), "run-unobservable", 20*time.Millisecond)
	mustErrorCode(t, err, domain.ErrRecoveryRequired)

	// SIGKILL was still sent (only the OBSERVATION of its effect was
	// faulted): restoring the real peek lets Start's own still-running
	// natural-exit observer goroutine reap the now-dead leader on its
	// next poll, instead of this test leaking it.
	waitidPeekFn.Store(origPeek)
}

// mustErrorCode asserts err is a *domain.Error with the given Code --
// duplicated from supervisor_test.go's process_test copy since this
// file is white-box (package process) and cannot import that
// external test package.
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
