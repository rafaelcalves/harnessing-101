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

// TestCLI_RunningDeadChildReconcilesToRecoveryRequired is H101-185's D2
// proof. A real start-run controller reaches the normal participate path,
// whose sync artifact proves the child passed startup and Running was the
// next durable observation. The child and still-live controller are then
// killed, and a fresh shipped `harnessing run` shows RecoveryRequired. The
// test never opens the host or reads engine/store state directly.
func TestCLI_RunningDeadChildReconcilesToRecoveryRequired(t *testing.T) {
	if runtime.GOOS != "darwin" && runtime.GOOS != "linux" {
		t.Skip("workspace operation is Unsupported on this platform")
	}

	binary := filepath.Join(t.TempDir(), "harnessing")
	build := exec.Command("go", "build", "-o", binary, ".")
	if output, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build harnessing binary: %v\n%s", err, output)
	}

	dir := t.TempDir()
	const wsID = "ws-cli-running-dead-child"
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

	syncPath := filepath.Join(t.TempDir(), "running.sync")
	start := exec.Command(binary, "start-run", "-workspace", dir, "-workspace-id", wsID, "-run", "run-1", "-agent", "agent-a", "-profile-id", "fixture-profile")
	start.Env = append(os.Environ(), "HARNESSING_FIXTURE_SYNC_FILE="+syncPath)
	var stdout, stderr bytes.Buffer
	start.Stdout = &stdout
	start.Stderr = &stderr
	if err := start.Start(); err != nil {
		t.Fatalf("start-run subprocess: %v", err)
	}

	fixturePID := waitForFixturePID(t, syncPath)
	// B5′a: normal participation, not the block_after_marker/Starting path.
	data, err := os.ReadFile(syncPath)
	if err != nil || !strings.HasPrefix(string(data), "participated\npid=") {
		t.Fatalf("sync marker = %q, want participated\\npid=<N>", string(data))
	}
	// B5′c: kill the participating child before the fresh reopen.
	if err := syscall.Kill(fixturePID, syscall.SIGKILL); err != nil {
		t.Fatalf("SIGKILL fixture child: %v", err)
	}
	// B5′b: the controller is still in its startup/observation call here;
	// killing it releases the workspace lock without a graceful Close.
	if err := start.Process.Kill(); err != nil {
		t.Fatalf("SIGKILL start-run controller: %v stdout=%q stderr=%q", err, stdout.String(), stderr.String())
	}
	if err := start.Wait(); err == nil {
		t.Fatal("SIGKILLed start-run controller exited successfully")
	}

	after := run("run", "-workspace", dir, "-workspace-id", wsID, "run-1")
	if after.code != 0 {
		t.Fatalf("harnessing run after dead child: exit=%d stdout=%s stderr=%s", after.code, after.stdout, after.stderr)
	}
	if !strings.Contains(after.stdout, "Run run-1") {
		t.Fatalf("run output omitted affected run identity: %q", after.stdout)
	}
	if !strings.Contains(after.stdout, "State:      RecoveryRequired") {
		t.Fatalf("run output omitted RecoveryRequired: %q", after.stdout)
	}
	if strings.Contains(after.stdout, "State:      Running") {
		t.Fatalf("run output still reports Running after dead child + reopen: %q", after.stdout)
	}
}

func waitForFixturePID(t *testing.T, syncPath string) int {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		data, err := os.ReadFile(syncPath)
		if err == nil {
			for _, line := range strings.Split(string(data), "\n") {
				if strings.HasPrefix(line, "pid=") {
					pid, parseErr := strconv.Atoi(strings.TrimPrefix(line, "pid="))
					if parseErr != nil || pid <= 0 {
						t.Fatalf("invalid fixture pid in sync file %q", string(data))
					}
					return pid
				}
			}
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatalf("fixture did not write pid sync marker: %s", syncPath)
	return 0
}
