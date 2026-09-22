package main

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"testing"
	"time"
)

// TestCLI_ServeOwnedGracefulCaptureCompletesBothChannels is H101-193/
// H101-199's item 5 evidence: a real supervised child, participating
// and later releasing under a continuing `harnessing serve` host (not
// the withdrawn one-shot producer H101-195 ruled out — see
// docs/architecture/h101-195-graceful-capture-producer.md), is durably
// captured on both channels and read back only through the shipped
// `harnessing run-output`/`events` commands attached to that SAME
// owner. I1-I6 are asserted in the order Kelly's h101-193 spec names
// them.
func TestCLI_ServeOwnedGracefulCaptureCompletesBothChannels(t *testing.T) {
	if runtime.GOOS != "darwin" && runtime.GOOS != "linux" {
		t.Skip("workspace operation is Unsupported on this platform")
	}
	binary := filepath.Join(t.TempDir(), "harnessing")
	build := exec.Command("go", "build", "-o", binary, ".")
	if output, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build harnessing binary: %v\n%s", err, output)
	}
	dir := t.TempDir()
	const wsID = "ws-item5-serve"

	run := func(args ...string) spawnedCLIResult { t.Helper(); return buildAndRunCLI(t, binary, args...) }
	succeed := func(args ...string) spawnedCLIResult {
		t.Helper()
		result := run(args...)
		if result.code != 0 {
			t.Fatalf("%v: exit=%d stdout=%s stderr=%s", args, result.code, result.stdout, result.stderr)
		}
		return result
	}

	// 1. Start the real serve host and wait for it to report the
	// workspace lock and transport are actually live. human1 must be a
	// reviewer of the SERVE HOST itself (fixed at its own launch,
	// never per-attached-client) so a later attached ApproveProfile
	// call can carry human-review authority.
	// Fixture env vars belong to the SERVE process's own environment,
	// not the attached start-run client's: Supervisor.Start spawns the
	// fixture as a child of serve (`cmd.Env = os.Environ()` in
	// internal/adapters/process/supervisor.go), so serve is the
	// process whose environment the child actually inherits — a
	// one-shot attached client's env never reaches it.
	syncPath := filepath.Join(t.TempDir(), "item5.sync")
	releasePath := filepath.Join(t.TempDir(), "item5.release")
	serveCmd := exec.Command(binary, "serve", "-workspace", dir, "-workspace-id", wsID, "-reviewer", "human1")
	serveCmd.Env = append(os.Environ(),
		"HARNESSING_FIXTURE_OUTPUT_MODE=serve_release_both_channels",
		"HARNESSING_FIXTURE_SYNC_FILE="+syncPath,
		"HARNESSING_FIXTURE_RELEASE_FILE="+releasePath,
	)
	stdoutPipe, err := serveCmd.StdoutPipe()
	if err != nil {
		t.Fatalf("StdoutPipe: %v", err)
	}
	var serveStderr bytes.Buffer
	serveCmd.Stderr = &serveStderr
	if err := serveCmd.Start(); err != nil {
		t.Fatalf("start serve: %v", err)
	}
	t.Cleanup(func() { _ = serveCmd.Process.Kill() })
	line, err := bufio.NewReader(stdoutPipe).ReadString('\n')
	if err != nil {
		t.Fatalf("reading serve stdout: %v", err)
	}
	if !strings.Contains(line, "listening") {
		t.Fatalf("serve did not report listening (line=%q stderr=%s)", line, serveStderr.String())
	}

	// 2. Register + approve `__fixture-participate` + context-file
	// through serve-attached CLI clients — never a fresh competing
	// workspace open.
	succeed("register", "-workspace", dir, "-attach",
		"-caller", "agent-a", "-request-id", "reg-a", "-agent", "agent-a", "-display-name", "Agent A")
	succeed("approve-profile", "-workspace", dir, "-attach",
		"-caller", "human1", "-request-id", "approve-a", "-profile-id", "fixture-profile",
		"-tool-executable", binary, "-tool-argv-json", `["__fixture-participate"]`, "-context-transport", "context-file")

	// 4. `harnessing start-run` (attached client) -> OK receipt (start
	// operation complete). The child remains supervised by the SAME
	// serve owner after this client process exits.
	startCmd := exec.Command(binary, "start-run", "-workspace", dir, "-attach",
		"-run", "run-1", "-agent", "agent-a", "-profile-id", "fixture-profile")
	var startOut, startErrBuf bytes.Buffer
	startCmd.Stdout = &startOut
	startCmd.Stderr = &startErrBuf
	if err := startCmd.Run(); err != nil {
		t.Fatalf("start-run: %v stdout=%s stderr=%s", err, startOut.String(), startErrBuf.String())
	}
	if !strings.Contains(startOut.String(), "OK") {
		t.Fatalf("start-run did not report an OK receipt: %s", startOut.String())
	}

	// Wait for the fixture's participation sync — proves startup
	// survived (B5' evidence class) before this test ever releases it.
	waitForFile(t, syncPath, "participated", 5*time.Second)

	// 5. Release: the test controls ordering explicitly, after the
	// start client's OK receipt and before the bounded wait below.
	if err := os.WriteFile(releasePath, []byte("release\n"), 0o600); err != nil {
		t.Fatalf("write release file: %v", err)
	}

	// 7. Fresh reader CLI subprocess attaches to the SAME serve owner.
	// Wait boundedly for terminal capture before asserting I2 — the
	// host drains and records Finish(Complete) asynchronously to the
	// start client's own exit (H101-195).
	var output spawnedCLIResult
	deadline := time.Now().Add(5 * time.Second)
	for {
		output = run("run-output", "-workspace", dir, "-attach", "run-1")
		if output.code == 0 && strings.Contains(output.stdout, "Capture status: complete") {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("run-output never reported complete capture: exit=%d stdout=%q stderr=%q", output.code, output.stdout, output.stderr)
		}
		time.Sleep(20 * time.Millisecond)
	}

	// I1: exit code 0 (already required by the loop above's break
	// condition; asserted again explicitly for the record).
	if output.code != 0 {
		t.Fatalf("run-output exit code = %d, want 0", output.code)
	}
	// I2 already satisfied by the wait loop's own break condition.
	// I3/I4: exact named literals, both channels.
	if !strings.Contains(output.stdout, "fixture-out-stdout\n") {
		t.Fatalf("run-output missing the exact stdout literal: %q", output.stdout)
	}
	if !strings.Contains(output.stdout, "fixture-out-stderr\n") {
		t.Fatalf("run-output missing the exact stderr literal: %q", output.stdout)
	}

	// I5: offset-ordered resume. The child's very first stderr write is
	// always the disclosure banner every `harnessing` command prints
	// (run.go's own `disclosure` const, same package here) — one
	// journal chunk of known exact length. Both named literals are
	// exactly 19 bytes ("fixture-out-stdout\n" / "fixture-out-stderr\n"),
	// so disclosureBytes+19 always lands exactly on the boundary
	// between the SECOND and THIRD chunk regardless of which literal
	// the journal happened to append second (stdout/stderr copy order
	// is not otherwise guaranteed).
	const literalLen = len("fixture-out-stdout\n")
	disclosureBytes := len(disclosure) + 1 // Fprintln's own trailing newline
	afterOffset := disclosureBytes + literalLen
	resumed := run("run-output", "-workspace", dir, "-attach", "-after-offset", fmt.Sprint(afterOffset), "run-1")
	if resumed.code != 0 {
		t.Fatalf("run-output -after-offset: exit=%d stderr=%s", resumed.code, resumed.stderr)
	}
	hasStdoutLiteral := strings.Contains(resumed.stdout, "fixture-out-stdout\n")
	hasStderrLiteral := strings.Contains(resumed.stdout, "fixture-out-stderr\n")
	if hasStdoutLiteral && hasStderrLiteral {
		t.Fatalf("run-output -after-offset returned BOTH chunks; want only the one after the boundary: %q", resumed.stdout)
	}
	if !hasStdoutLiteral && !hasStderrLiteral {
		t.Fatalf("run-output -after-offset returned no later bytes at all: %q", resumed.stdout)
	}

	// I6: the shipped state-event read path never carries either
	// literal — UI-08 separation, proven on a fresh attached reader,
	// not by inspecting engine internals.
	events := run("events", "-workspace", dir, "-attach")
	if events.code != 0 {
		t.Fatalf("events: exit=%d stderr=%s", events.code, events.stderr)
	}
	if strings.Contains(events.stdout, "fixture-out-stdout") || strings.Contains(events.stdout, "fixture-out-stderr") {
		t.Fatalf("state-event read path leaked raw output bytes: %q", events.stdout)
	}

	// 8. Teardown only after evidence collection.
	if err := serveCmd.Process.Signal(syscall.SIGTERM); err != nil {
		t.Fatalf("signal serve SIGTERM: %v", err)
	}
	_ = serveCmd.Wait()
}

func waitForFile(t *testing.T, path, wantSubstring string, timeout time.Duration) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for {
		data, err := os.ReadFile(path)
		if err == nil && strings.Contains(string(data), wantSubstring) {
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("%s did not appear containing %q within %s", path, wantSubstring, timeout)
		}
		time.Sleep(10 * time.Millisecond)
	}
}
