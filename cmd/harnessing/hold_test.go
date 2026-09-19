package main

import (
	"bufio"
	"bytes"
	"os"
	"os/exec"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/rafaelcalves/harnessing-101/internal/host"
)

// holdHelperEnv, when set, makes this test binary re-exec itself as a
// standalone `harnessing hold` process instead of running go test's own
// test functions — the same re-exec-self-as-helper pattern used by
// internal/adapters/statestore/lock_recovery_test.go and by os/exec's
// own test suite. holdHelperArgsEnv carries the argv to pass to run(),
// joined with a unit separator (not NUL: os/exec rejects a NUL byte
// anywhere in an environment value) so temp-dir paths round-trip safely
// regardless of spaces.
const (
	holdHelperEnv     = "HARNESSING_HOLD_HELPER"
	holdHelperArgsEnv = "HARNESSING_HOLD_ARGS"
	holdArgsSep       = "\x1f"
)

func TestMain(m *testing.M) {
	if os.Getenv(holdHelperEnv) == "1" {
		args := strings.Split(os.Getenv(holdHelperArgsEnv), holdArgsSep)
		os.Exit(run(args, os.Stdout, os.Stderr))
	}
	os.Exit(m.Run())
}

func startHoldHelper(t *testing.T, args []string) (*exec.Cmd, *bufio.Reader) {
	t.Helper()
	cmd := exec.Command(os.Args[0], "-test.run=^$")
	cmd.Env = append(os.Environ(),
		holdHelperEnv+"=1",
		holdHelperArgsEnv+"="+strings.Join(args, holdArgsSep),
	)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatalf("StdoutPipe: %v", err)
	}
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Start(); err != nil {
		t.Fatalf("start hold helper: %v", err)
	}
	t.Cleanup(func() { _ = cmd.Process.Kill() })
	return cmd, bufio.NewReader(stdout)
}

// TestRun_HoldBlocksConcurrentOpenAndReleasesOnKill is the CLI-level
// counterpart to H101-20's adapter-level recovery test: it proves the
// tool a real user runs, not just the library underneath it, produces a
// live process that (a) genuinely holds the workspace and (b) releases
// it when killed. This is exactly the fixture Claudio's H101-49
// cycle-kill-reopen test needs: before this card, no `harnessing`
// invocation stayed alive long enough to kill.
func TestRun_HoldBlocksConcurrentOpenAndReleasesOnKill(t *testing.T) {
	dir := t.TempDir()
	cmd, out := startHoldHelper(t, []string{"hold", "-workspace", dir, "-workspace-id", "ws-hold"})

	line, err := out.ReadString('\n')
	if err != nil || !strings.Contains(line, "holding lock") {
		t.Fatalf("hold did not report holding the lock (line=%q, err=%v)", line, err)
	}

	if _, err := host.Open(dir, "ws-hold", nil); err == nil {
		t.Fatal("host.Open succeeded while `harnessing hold` is alive")
	}

	if err := cmd.Process.Kill(); err != nil {
		t.Fatalf("kill hold helper: %v", err)
	}
	_ = cmd.Wait()

	deadline := time.Now().Add(5 * time.Second)
	for {
		caps, err := host.Open(dir, "ws-hold", nil)
		if err == nil {
			_ = caps.Close()
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("workspace still locked after `harnessing hold` was killed: %v", err)
		}
		time.Sleep(20 * time.Millisecond)
	}
}

// TestRun_HoldExitsCleanlyOnSIGTERM checks the graceful path: a normal
// stop signal, not a crash, still closes the workspace and exits 0.
func TestRun_HoldExitsCleanlyOnSIGTERM(t *testing.T) {
	dir := t.TempDir()
	cmd, out := startHoldHelper(t, []string{"hold", "-workspace", dir, "-workspace-id", "ws-hold-term"})

	line, err := out.ReadString('\n')
	if err != nil || !strings.Contains(line, "holding lock") {
		t.Fatalf("hold did not report holding the lock (line=%q, err=%v)", line, err)
	}

	if err := cmd.Process.Signal(syscall.SIGTERM); err != nil {
		t.Fatalf("signal SIGTERM: %v", err)
	}

	closedLine, err := out.ReadString('\n')
	if err != nil || !strings.Contains(closedLine, "closed") {
		t.Fatalf("hold did not report a clean close after SIGTERM (line=%q, err=%v)", closedLine, err)
	}
	if err := cmd.Wait(); err != nil {
		t.Fatalf("hold exited non-zero after SIGTERM: %v", err)
	}

	caps, err := host.Open(dir, "ws-hold-term", nil)
	if err != nil {
		t.Fatalf("workspace still locked after a graceful SIGTERM close: %v", err)
	}
	_ = caps.Close()
}
