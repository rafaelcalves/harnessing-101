//go:build windows

package statestore

import (
	"errors"
	"os"
	"testing"

	"github.com/rafaelcalves/harnessing-101/internal/core/domain"
)

// H101-101: Kelly's exit check is that an unsupported platform refuses
// cleanly rather than failing by accident. This is the unit half of
// that proof: acquireLock and releaseLock, on windows, must answer
// Unsupported — not a syscall panic, not a silent success, not some
// other error code a caller would have to guess at.
//
// This file only builds and runs under GOOS=windows; see
// docs/quality/h101-101-windows-exclusion.md for how that is verified
// without a windows runner in this project's CI today.
func TestAcquireLock_Windows_ReturnsUnsupported(t *testing.T) {
	f, err := os.CreateTemp(t.TempDir(), "lock")
	if err != nil {
		t.Fatalf("CreateTemp: %v", err)
	}
	defer f.Close()

	err = acquireLock(f)
	var derr *domain.Error
	if !errors.As(err, &derr) {
		t.Fatalf("acquireLock error = %T(%v), want *domain.Error", err, err)
	}
	if derr.Code != domain.ErrUnsupported {
		t.Fatalf("acquireLock code = %s, want %s", derr.Code, domain.ErrUnsupported)
	}
}

func TestReleaseLock_Windows_ReturnsUnsupported(t *testing.T) {
	f, err := os.CreateTemp(t.TempDir(), "lock")
	if err != nil {
		t.Fatalf("CreateTemp: %v", err)
	}
	defer f.Close()

	err = releaseLock(f)
	var derr *domain.Error
	if !errors.As(err, &derr) {
		t.Fatalf("releaseLock error = %T(%v), want *domain.Error", err, err)
	}
	if derr.Code != domain.ErrUnsupported {
		t.Fatalf("releaseLock code = %s, want %s", derr.Code, domain.ErrUnsupported)
	}
}
