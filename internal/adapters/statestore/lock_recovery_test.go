//go:build !windows

package statestore

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"testing"
	"time"

	"github.com/rafaelcalves/harnessing-101/internal/core/domain"
)

// lockHelperEnv, when set to "1", makes this same test binary act as a
// standalone process that holds the workspace lock and blocks until
// killed — the standard Go trick for exercising cross-process behaviour
// without a separate compiled fixture (see os/exec's own tests). The
// parent test re-execs itself with this set via exec.Command(os.Args[0],
// ...) and a matching -test.run filter, so no other test function runs
// inside the helper invocation.
const lockHelperEnv = "STATESTORE_LOCK_HELPER"

func TestMain(m *testing.M) {
	if os.Getenv(lockHelperEnv) == "1" {
		runLockHelper(os.Args[len(os.Args)-1])
		return
	}
	os.Exit(m.Run())
}

// runLockHelper opens dir, prints "locked" once it holds the lock (the
// parent waits on this line as its synchronization signal), then blocks
// forever. The parent kills this process; it never exits on its own.
func runLockHelper(dir string) {
	store, err := Open(dir)
	if err != nil {
		fmt.Println("helper: open failed: " + err.Error())
		os.Exit(1)
	}
	_ = store // keep the lock file open; do not Close.
	fmt.Println("locked")
	select {}
}

// TestOpen_RecoversAfterOwnerCrash is H101-20's proof: a workspace
// killed without Close is not locked out forever, and — just as
// important — a workspace whose owner is still alive is correctly
// refused the whole time, with no timeout involved either way.
func TestOpen_RecoversAfterOwnerCrash(t *testing.T) {
	dir := t.TempDir()

	// -test.run matching nothing skips every real test function; TestMain
	// still runs first regardless (it is not itself test-selected) and
	// intercepts before m.Run() would be called.
	helper := exec.Command(os.Args[0], "-test.run=^$", dir)
	helper.Env = append(os.Environ(), lockHelperEnv+"=1")
	stdout, err := helper.StdoutPipe()
	if err != nil {
		t.Fatalf("StdoutPipe: %v", err)
	}
	if err := helper.Start(); err != nil {
		t.Fatalf("start helper: %v", err)
	}
	t.Cleanup(func() { _ = helper.Process.Kill() })

	line, err := bufio.NewReader(stdout).ReadString('\n')
	if err != nil || line != "locked\n" {
		t.Fatalf("helper did not report locked (line=%q, err=%v)", line, err)
	}

	// While the helper is alive, a second Open must fail Busy — no
	// timeout, no grace period, just the kernel's real answer.
	if _, err := Open(dir); err == nil {
		t.Fatal("Open succeeded while the owning process is still alive")
	} else if derr, ok := err.(*domain.Error); !ok || derr.Code != domain.ErrBusy {
		t.Fatalf("Open while alive = %v, want *domain.Error{Code: Busy}", err)
	}

	// Simulate a crash: no Close, just SIGKILL.
	if err := helper.Process.Kill(); err != nil {
		t.Fatalf("kill helper: %v", err)
	}
	_ = helper.Wait()

	// Recovery must not depend on wall-clock elapsed time; poll briefly
	// only to let the kernel finish tearing down the killed process's
	// file table, not to "wait out" any lock timeout (there is none).
	deadline := time.Now().Add(5 * time.Second)
	var store *FileStore
	for {
		store, err = Open(dir)
		if err == nil {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("Open still failing %s after the owner was killed: %v", time.Since(deadline), err)
		}
		time.Sleep(20 * time.Millisecond)
	}
	defer func() { _ = store.Close() }()
}
