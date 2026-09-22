package main

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"
)

// TestCLI_RecoveryRequiredAfterRealStartRunCrash is H101-181's E2/D6 proof:
// a real start-run subprocess is killed after its fixture confirms the
// dispatch-attempted window, then a fresh shipped `harnessing run` command
// shows RecoveryRequired. The test deliberately never opens the host or reads
// the engine/store state itself.
func TestCLI_RecoveryRequiredAfterRealStartRunCrash(t *testing.T) {
	if runtime.GOOS != "darwin" && runtime.GOOS != "linux" {
		t.Skip("workspace operation is Unsupported on this platform")
	}

	binary := filepath.Join(t.TempDir(), "harnessing")
	build := exec.Command("go", "build", "-o", binary, ".")
	if output, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build harnessing binary: %v\n%s", err, output)
	}

	dir := t.TempDir()
	const wsID = "ws-cli-recovery-required"

	run := func(args ...string) spawnedCLIResult {
		t.Helper()
		return buildAndRunCLI(t, binary, args...)
	}
	succeed := func(args ...string) {
		t.Helper()
		result := run(args...)
		if result.code != 0 {
			t.Fatalf("%v: exit=%d stdout=%s stderr=%s", args, result.code, result.stdout, result.stderr)
		}
	}

	succeed("register", "-workspace", dir, "-workspace-id", wsID, "-caller", "agent-a", "-request-id", "reg-a", "-agent", "agent-a", "-display-name", "Agent A")
	succeed("approve-profile", "-workspace", dir, "-workspace-id", wsID, "-reviewer", "human1", "-caller", "human1", "-request-id", "approve-a", "-profile-id", "fixture-profile", "-tool-executable", binary, "-tool-argv-json", `["__fixture-participate"]`, "-context-transport", "context-file")

	syncPath := filepath.Join(t.TempDir(), "start-called.sync")
	start := exec.Command(binary, "start-run", "-workspace", dir, "-workspace-id", wsID, "-run", "run-1", "-agent", "agent-a", "-profile-id", "fixture-profile")
	start.Env = append(os.Environ(), "HARNESSING_FIXTURE_SIMULATE=block_after_marker", "HARNESSING_FIXTURE_SYNC_FILE="+syncPath)
	var startOut, startErr bytes.Buffer
	start.Stdout = &startOut
	start.Stderr = &startErr
	if err := start.Start(); err != nil {
		t.Fatalf("start-run subprocess: %v", err)
	}

	var fixturePID int
	deadline := time.Now().Add(5 * time.Second)
	for fixturePID == 0 && time.Now().Before(deadline) {
		data, err := os.ReadFile(syncPath)
		if err == nil && strings.HasPrefix(string(data), "start-called\n") {
			for _, line := range strings.Split(string(data), "\n") {
				if strings.HasPrefix(line, "pid=") {
					fixturePID, err = strconv.Atoi(strings.TrimPrefix(line, "pid="))
					if err != nil || fixturePID <= 0 {
						t.Fatalf("invalid fixture pid in sync file %q", string(data))
					}
				}
			}
		}
		if fixturePID == 0 {
			time.Sleep(5 * time.Millisecond)
		}
	}
	if fixturePID == 0 {
		t.Fatalf("fixture did not confirm start-called window; stdout=%q stderr=%q", startOut.String(), startErr.String())
	}
	if start.ProcessState != nil {
		t.Fatalf("start-run exited before crash window: state=%v stdout=%q stderr=%q", start.ProcessState, startOut.String(), startErr.String())
	}

	// Kill the actual CLI controller, not an engine helper. This leaves the
	// durable Starting run marker in the same unclean window as H101-161.
	if err := start.Process.Kill(); err != nil {
		t.Fatalf("SIGKILL start-run controller: %v", err)
	}
	if err := start.Wait(); err == nil {
		t.Fatal("SIGKILLed start-run controller exited successfully")
	}
	defer func() { _ = syscall.Kill(fixturePID, syscall.SIGKILL) }()

	runResult := run("run", "-workspace", dir, "-workspace-id", wsID, "run-1")
	if runResult.code != 0 {
		t.Fatalf("harnessing run after crash: exit=%d stdout=%s stderr=%s", runResult.code, runResult.stdout, runResult.stderr)
	}
	if !strings.Contains(runResult.stdout, "Run run-1") {
		t.Fatalf("run output omitted crashed run identity: %q", runResult.stdout)
	}
	if !strings.Contains(runResult.stdout, "State:      RecoveryRequired") {
		t.Fatalf("run output omitted RecoveryRequired: %q", runResult.stdout)
	}
	if strings.Contains(runResult.stdout, "State:      Starting") {
		t.Fatalf("run output still reports Starting after crash reconciliation: %q", runResult.stdout)
	}

	second := run("start-run", "-workspace", dir, "-workspace-id", wsID, "-run", "run-2", "-agent", "agent-a", "-profile-id", "fixture-profile")
	if second.code == 0 || !strings.Contains(second.stderr, "Conflict") {
		t.Fatalf("second start-run = exit %d stderr %q; want non-zero Conflict with no child spawn", second.code, second.stderr)
	}
}
