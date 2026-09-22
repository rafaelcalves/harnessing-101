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

// TestCLI_RunOutputCrashPrefixIsHonest is H101-191's D4 proof. A normal
// participating child emits one known prefix, blocks before the named tail,
// and is killed with its controller. A fresh shipped run-output command reads
// the exact prefix and interrupted status without invented bytes or counts.
func TestCLI_RunOutputCrashPrefixIsHonest(t *testing.T) {
	if runtime.GOOS != "darwin" && runtime.GOOS != "linux" {
		t.Skip("workspace operation is Unsupported on this platform")
	}
	binary := filepath.Join(t.TempDir(), "harnessing")
	build := exec.Command("go", "build", "-o", binary, ".")
	if output, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build harnessing binary: %v\n%s", err, output)
	}
	dir := t.TempDir()
	const wsID = "ws-cli-output-prefix"
	run := func(args ...string) spawnedCLIResult { t.Helper(); return buildAndRunCLI(t, binary, args...) }
	succeed := func(args ...string) {
		t.Helper()
		result := run(args...)
		if result.code != 0 {
			t.Fatalf("%v: exit=%d stdout=%s stderr=%s", args, result.code, result.stdout, result.stderr)
		}
	}
	succeed("register", "-workspace", dir, "-workspace-id", wsID, "-caller", "agent-a", "-request-id", "reg-a", "-agent", "agent-a", "-display-name", "Agent A")
	succeed("approve-profile", "-workspace", dir, "-workspace-id", wsID, "-reviewer", "human1", "-caller", "human1", "-request-id", "approve-a", "-profile-id", "fixture-profile", "-tool-executable", binary, "-tool-argv-json", `["__fixture-participate"]`, "-context-transport", "context-file")

	syncPath := filepath.Join(t.TempDir(), "output.sync")
	start := exec.Command(binary, "start-run", "-workspace", dir, "-workspace-id", wsID, "-run", "run-1", "-agent", "agent-a", "-profile-id", "fixture-profile")
	start.Env = append(os.Environ(), "HARNESSING_FIXTURE_OUTPUT_MODE=prefix_then_block", "HARNESSING_FIXTURE_SYNC_FILE="+syncPath)
	var stdout, stderr bytes.Buffer
	start.Stdout = &stdout
	start.Stderr = &stderr
	if err := start.Start(); err != nil {
		t.Fatalf("start-run subprocess: %v", err)
	}
	fixturePID := waitForD4Fixture(t, syncPath)
	// The sync write follows the fixture's prefix write; allow the journal
	// writer's fsync to complete before killing the producer.
	time.Sleep(100 * time.Millisecond)
	if err := syscall.Kill(fixturePID, syscall.SIGKILL); err != nil {
		t.Fatalf("SIGKILL fixture child: %v", err)
	}
	if err := start.Process.Kill(); err != nil {
		t.Fatalf("SIGKILL start-run controller: %v stdout=%q stderr=%q", err, stdout.String(), stderr.String())
	}
	if err := start.Wait(); err == nil {
		t.Fatal("SIGKILLed start-run controller exited successfully")
	}

	output := run("run-output", "-workspace", dir, "-workspace-id", wsID, "run-1")
	if output.code != 0 {
		t.Fatalf("run-output: exit=%d stdout=%s stderr=%s", output.code, output.stdout, output.stderr)
	}
	if !strings.Contains(output.stdout, "fixture-out-1\n") {
		t.Fatalf("run-output omitted exact persisted prefix: %q", output.stdout)
	}
	if strings.Contains(output.stdout, "fixture-out-2\n") {
		t.Fatalf("run-output fabricated never-emitted tail: %q", output.stdout)
	}
	if !strings.Contains(output.stdout, "Capture status: interrupted") && !strings.Contains(output.stdout, "Capture status: unknown") {
		t.Fatalf("run-output omitted interrupted/unknown status: %q", output.stdout)
	}
	if strings.Contains(output.stdout, "Bytes:") || strings.Contains(output.stdout, "Range:") {
		t.Fatalf("run-output claimed a byte count/range: %q", output.stdout)
	}
}

func waitForD4Fixture(t *testing.T, syncPath string) int {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		data, err := os.ReadFile(syncPath)
		if err == nil && strings.Contains(string(data), "prefix-ready") {
			for _, line := range strings.Split(string(data), "\n") {
				if strings.HasPrefix(line, "pid=") {
					pid, parseErr := strconv.Atoi(strings.TrimPrefix(line, "pid="))
					if parseErr != nil || pid <= 0 {
						t.Fatalf("invalid fixture pid in %q", string(data))
					}
					return pid
				}
			}
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatalf("fixture did not write prefix sync marker: %s", syncPath)
	return 0
}
