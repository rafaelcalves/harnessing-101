//go:build windows

package statestore

import (
	"os"

	"github.com/rafaelcalves/harnessing-101/internal/core/domain"
)

// Windows has an equivalent kernel-tracked advisory lock (LockFileEx),
// but it is not exposed by Go's syscall package the way flock(2) is on
// Unix, and no supported-platform decision (ADR 0001: "unsettled") has
// ever included Windows. Rather than guess at an untested
// golang.org/x/sys/windows-based implementation with no way to verify
// its crash-recovery behaviour on this floor, this returns Unsupported
// outright: an honest UNKNOWN is a better answer than a confident one
// nobody has run. See docs/architecture/h101-20-lock-recovery.md.
func acquireLock(f *os.File) error {
	return &domain.Error{Code: domain.ErrUnsupported, Detail: "workspace locking is not implemented on windows"}
}

func releaseLock(f *os.File) error {
	return &domain.Error{Code: domain.ErrUnsupported, Detail: "workspace locking is not implemented on windows"}
}
