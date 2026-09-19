//go:build !windows

package statestore

import (
	"errors"
	"os"
	"syscall"

	"github.com/rafaelcalves/harnessing-101/internal/core/domain"
)

// acquireLock takes an exclusive, non-blocking flock(2) on f. This is a
// kernel-tracked lock tied to the open file description: it is released
// automatically when every file descriptor referring to it closes,
// including when the owning process dies without running Close — no PID
// comparison, no staleness heuristic, no window where a live owner can
// be mistaken for a dead one or vice versa. Verified empirically for
// this card: a lock held by a process killed with SIGKILL is observed
// released by a second process's very next attempt (see
// docs/architecture/h101-20-lock-recovery.md).
//
// Scope: local filesystems only, matching threat-model.md's existing
// "unverified network filesystems are unsupported" line — flock over
// NFS is not reliably enforced without a correctly configured lock
// daemon, and this adapter makes no attempt to detect that case.
func acquireLock(f *os.File) error {
	err := syscall.Flock(int(f.Fd()), syscall.LOCK_EX|syscall.LOCK_NB)
	if err == nil {
		return nil
	}
	if errors.Is(err, syscall.EWOULDBLOCK) || errors.Is(err, syscall.EAGAIN) {
		return &domain.Error{Code: domain.ErrBusy, Detail: "workspace root is locked by another live process"}
	}
	return &domain.Error{Code: domain.ErrIOFailure, Detail: "acquire workspace lock: " + err.Error()}
}

func releaseLock(f *os.File) error {
	if err := syscall.Flock(int(f.Fd()), syscall.LOCK_UN); err != nil {
		return &domain.Error{Code: domain.ErrIOFailure, Detail: "release workspace lock: " + err.Error()}
	}
	return nil
}
