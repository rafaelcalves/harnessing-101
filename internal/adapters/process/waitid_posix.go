package process

import (
	"context"
	"errors"
	"sync/atomic"
	"syscall"
	"time"
	"unsafe"
)

// idTypePID is idtype_t's P_PID value from <sys/wait.h>. On both ADR
// targets -- Linux (include/uapi/linux/wait.h idtype_t) and Darwin
// (bsd/sys/wait.h idtype_t) -- the enum order is P_ALL=0, P_PID=1,
// P_PGID=2[, ...], stable across every supported kernel/SDK release.
// The stdlib syscall package exposes no portable named constant for
// it, so it is pinned here rather than discovered at runtime.
const idTypePID = 1

// waitidInfo mirrors only the leading, stable fields of siginfo_t
// that waitid(2) populates for a SIGCHLD-shaped wait: si_signo,
// si_errno, si_code, si_pid at byte offsets 0/4/8/12 on both ADR
// targets. Confirmed against the Go runtime's own Darwin siginfo
// struct (runtime/defs_darwin_{amd64,arm64}.go) and the Linux kernel
// ABI's padded siginfo_t (golang.org/x/sys/unix.Siginfo uses the same
// leading layout). The trailing padding is sized to the larger
// (Linux, 128-byte) struct so a kernel write on either target never
// overruns this buffer; the extra space on Darwin is simply unused.
type waitidInfo struct {
	Signo int32
	Errno int32
	Code  int32
	Pid   int32
	_     [112]byte
}

// pollInterval bounds how often the non-reaping observation loops
// below re-check exit status. It is deliberately short: none of these
// loops block the kernel (WNOHANG), so a short interval only costs a
// few extra syscalls, never a longer wall-clock wait.
const pollInterval = 10 * time.Millisecond

// waitidPeekFn is an atomically-swappable indirection, not a plain
// function var, so an internal (white-box) test can inject a
// binding-establishment failure -- proving the N6 hard-refusal path,
// Stop must return RecoveryRequired, never a best-effort signal to
// the bare pgid -- from one goroutine while Start's own background
// natural-exit observer is concurrently calling waitidPeek in
// another; a plain package-level func var reassignment there would
// be a real data race, not just a test artifact.
var waitidPeekFn atomic.Value

func init() {
	waitidPeekFn.Store(waitidPeekFunc(rawWaitidPeek))
}

type waitidPeekFunc func(pid int) (bool, error)

func waitidPeek(pid int) (bool, error) {
	fn := waitidPeekFn.Load().(waitidPeekFunc)
	return fn(pid)
}

// rawWaitidPeek asks whether pid -- always one of this adapter's own
// retained, unreaped children -- has an exit status available WITHOUT
// consuming it (WNOWAIT) and without blocking if it does not
// (WNOHANG). This is the non-reaping observation h101-215 requires:
// callers may poll it as many times, from as many goroutines, as they
// like, and the child is never touched. A zero Pid in the populated
// info means WNOHANG found no newly-exited child matching pid yet (a
// documented waitid(2) quirk, not an error); ECHILD means pid is not
// one of our unreaped children at all (already reaped, or never was).
func rawWaitidPeek(pid int) (exited bool, err error) {
	var info waitidInfo
	_, _, errno := syscall.Syscall6(syscall.SYS_WAITID,
		uintptr(idTypePID), uintptr(pid), uintptr(unsafe.Pointer(&info)), //nolint:gosec // waitid(2) writes into this buffer; size is intentionally the larger of the two ADR targets' siginfo_t.
		uintptr(syscall.WEXITED|syscall.WNOWAIT|syscall.WNOHANG), 0, 0)
	if errno != 0 {
		return false, errno
	}
	return info.Pid == int32(pid), nil
}

// reapLeader performs the one, final, consuming wait -- deliberately
// a bare syscall.Wait4 on pid alone, never cmd.Wait(): the latter also
// blocks on its internal stdout/stderr copy goroutines reaching pipe
// EOF, which a surviving mode-B worker (inheriting those same pipe
// fds from the leader at fork) can hold open indefinitely after the
// leader itself has already exited. This call reaps ONLY the process,
// decoupled from any I/O this adapter no longer needs to wait on.
func reapLeader(pid int) (syscall.WaitStatus, error) {
	var ws syscall.WaitStatus
	_, err := syscall.Wait4(pid, &ws, 0, nil)
	return ws, err
}

// waitLeaderExited bounds how long Stop will wait, via non-reaping
// observation, for the retained leader itself to show an exit status
// after a forced signal -- never forever, and never inferred from the
// group probe (which a retained zombie keeps positive on its own).
func waitLeaderExited(pid int, bound time.Duration) bool {
	deadline := time.Now().Add(bound)
	for {
		exited, err := waitidPeek(pid)
		if err == syscall.ECHILD {
			// The background natural-exit observer (Start's own
			// goroutine) won the race and already reaped this leader
			// under the run's own serialization -- that is still
			// termination, observed a different way, not a failure.
			return true
		}
		if err != nil {
			return false
		}
		if exited {
			return true
		}
		if time.Now().After(deadline) {
			return false
		}
		time.Sleep(pollInterval)
	}
}

// waitLeaderExitedWithinGrace bounds Stop's own grace period as an
// early-exit-aware poll rather than a blind sleep: if the leader
// already died (typically to the SIGTERM Stop just sent) before grace
// elapses, this returns true immediately so Stop can skip a
// needless SIGKILL, rather than always waiting out the full duration.
func waitLeaderExitedWithinGrace(ctx context.Context, pid int, grace time.Duration) bool {
	deadline := time.Now().Add(grace)
	for {
		exited, err := waitidPeek(pid)
		if err == nil && exited {
			return true
		}
		if time.Now().After(deadline) {
			return false
		}
		select {
		case <-ctx.Done():
			return false
		case <-time.After(pollInterval):
		}
	}
}

// waitGroupGone is ONLY used after the leader has already been closed
// and reaped: at that point a positive probe is real signal (some
// other group member is still alive), not the always-positive
// artifact a retained zombie leader would otherwise produce.
func waitGroupGone(pgid int, bound time.Duration) bool {
	deadline := time.Now().Add(bound)
	for groupExists(pgid) {
		if time.Now().After(deadline) {
			return false
		}
		time.Sleep(pollInterval)
	}
	return true
}

func groupExists(pgid int) bool {
	err := syscall.Kill(-pgid, 0)
	return err == nil || err == syscall.EPERM
}

// isBenignGroupSignalError is for the SIGTERM/SIGKILL send calls in
// Stop, before the leader is reaped -- NOT for groupExists above,
// which runs only after reaping and needs the opposite EPERM
// interpretation. Confirmed empirically on darwin/arm64: kill(-pgid,
// sig) on a process group whose ONLY member is our own retained,
// terminated-but-unreaped leader returns EPERM, not ESRCH and not
// success. At this point in Stop the leader may already be dead (a
// fast-dying leader, or one that already died to an earlier signal in
// this same call) while still correctly retained-unreaped, so this is
// an expected, benign outcome -- there is nothing left to signal --
// not a sign the binding or permission was actually lost. Tolerating
// it here does not weaken safety: if the pgid number had genuinely
// been taken over by an unrelated live process instead, Stop's own
// POST-REAP waitGroupGone check still sees that real occupant and
// correctly reports RecoveryRequired rather than success.
func isBenignGroupSignalError(err error) bool {
	return err == syscall.ESRCH || err == syscall.EPERM
}

// membershipRecheckInterval separates the two scans confirmOtherMembersAbsent
// requires. It is deliberately short (a fast retest), not a substitute
// for a long observation window -- the goal is catching an obviously
// racing fork, not eliminating every theoretical one.
const membershipRecheckInterval = 50 * time.Millisecond

// otherMembersAbsentFn is an atomically-swappable indirection over the
// platform-specific otherMembersAbsent, for the same reason
// waitidPeekFn is one: an internal test needs to inject an unreadable/
// uncertain membership scan without faking an actual kernel failure,
// from one goroutine while production code may be calling the real
// one concurrently.
var otherMembersAbsentFn atomic.Value

func init() {
	otherMembersAbsentFn.Store(membershipCheckFunc(otherMembersAbsent))
}

type membershipCheckFunc func(pgid, excludePID int) (bool, error)

// confirmOtherMembersAbsent is H101-222's membership-race-aware
// predicate: TWO scans, separated by membershipRecheckInterval, must
// BOTH report no member other than excludePID before this reports
// confirmed==true. This does not eliminate every race (no single or
// double point-in-time scan can prove a negative against an
// adversarially-timed fork) -- it is real, bounded evidence against
// the specific race H101-222 names, never a single boolean probe's
// lucky snapshot. Any read/parse error on either scan is uncertain,
// never treated as absence.
// InjectMembershipUncertainty and RestoreMembershipCheck are the
// external-test seam over otherMembersAbsentFn: membership_test.go
// lives in package process_test (black-box, matching this package's
// own convention of testing the Supervisor as a caller would), so it
// cannot swap the unexported package var directly the way
// lifecycle_internal_test.go swaps waitidPeekFn. Both are methods on
// *Supervisor purely to give an external test a reachable name; the
// swapped state is package-wide, same as waitidPeekFn.
func (s *Supervisor) InjectMembershipUncertainty() error {
	otherMembersAbsentFn.Store(membershipCheckFunc(func(pgid, excludePID int) (bool, error) {
		return false, errors.New("injected: membership scan uncertain")
	}))
	return nil
}

func (s *Supervisor) RestoreMembershipCheck() {
	otherMembersAbsentFn.Store(membershipCheckFunc(otherMembersAbsent))
}

func confirmOtherMembersAbsent(pgid, excludePID int) (confirmed, uncertain bool) {
	fn := otherMembersAbsentFn.Load().(membershipCheckFunc)
	absent1, err1 := fn(pgid, excludePID)
	if err1 != nil {
		return false, true
	}
	if !absent1 {
		return false, false
	}
	time.Sleep(membershipRecheckInterval)
	absent2, err2 := fn(pgid, excludePID)
	if err2 != nil {
		return false, true
	}
	return absent2, false
}
