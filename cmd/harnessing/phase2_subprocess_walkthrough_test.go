package main

import (
	"bytes"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/rafaelcalves/harnessing-101/internal/core/domain"
)

type spawnedCLIResult struct {
	stdout string
	stderr string
	code   int
}

func buildAndRunCLI(t *testing.T, binary string, args ...string) spawnedCLIResult {
	t.Helper()
	cmd := exec.Command(binary, args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	code := 0
	if err != nil {
		code = 1
		if exitErr, ok := err.(*exec.ExitError); ok {
			code = exitErr.ExitCode()
		}
	}
	return spawnedCLIResult{stdout: stdout.String(), stderr: stderr.String(), code: code}
}

// TestCLI_Phase2ProductWalkthroughSubprocess is the executable Phase 2
// walkthrough. Every product action is an exec.Command against a compiled
// harnessing binary; no host or engine method drives the cycle.
func TestCLI_Phase2ProductWalkthroughSubprocess(t *testing.T) {
	if runtime.GOOS != "darwin" && runtime.GOOS != "linux" {
		t.Skip("workspace operation is Unsupported on this platform")
	}

	binary := filepath.Join(t.TempDir(), "harnessing")
	build := exec.Command("go", "build", "-o", binary, ".")
	if output, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build harnessing binary: %v\n%s", err, output)
	}

	dir := t.TempDir()
	const wsID = "ws-subprocess-walkthrough"
	const reviewer = "reviewer1"

	call := func(args ...string) spawnedCLIResult {
		t.Helper()
		result := buildAndRunCLI(t, binary, args...)
		if !strings.HasPrefix(result.stderr, disclosure+"\n") {
			t.Fatalf("%v: stderr omitted leading disclosure: %q", args, result.stderr)
		}
		return result
	}
	succeed := func(args ...string) string {
		t.Helper()
		result := call(args...)
		if result.code != 0 {
			t.Fatalf("%v: exit=%d stdout=%s stderr=%s", args, result.code, result.stdout, result.stderr)
		}
		return result.stdout
	}
	status := func(want string) {
		t.Helper()
		output := succeed("task", "-workspace", dir, "-workspace-id", wsID, "t1")
		if !strings.Contains(output, "Status:     "+want) {
			t.Fatalf("task status = %q, want %s", output, want)
		}
	}

	// Create a two-agent workspace and an accountable task.
	succeed("register", "-workspace", dir, "-workspace-id", wsID, "-reviewer", reviewer, "-caller", "engineer", "-request-id", "r1", "-agent", "engineer", "-display-name", "Engineer")
	succeed("register", "-workspace", dir, "-workspace-id", wsID, "-reviewer", reviewer, "-caller", "analyst", "-request-id", "r2", "-agent", "analyst", "-display-name", "Analyst")
	succeed("create", "-workspace", dir, "-workspace-id", wsID, "-reviewer", reviewer, "-caller", "engineer", "-request-id", "r3", "-task", "t1", "-title", "Investigate", "-assignee", "engineer")
	status("Todo")
	succeed("transition", "-workspace", dir, "-workspace-id", wsID, "-reviewer", reviewer, "-caller", "engineer", "-request-id", "r4", "-task", "t1", "-from", "Todo", "-to", "Doing")
	status("Doing")

	// Reusing an existing task ID is a stable Conflict, not a successful rewrite.
	conflict := call("create", "-workspace", dir, "-workspace-id", wsID, "-reviewer", reviewer, "-caller", "engineer", "-request-id", "r4-conflict", "-task", "t1", "-title", "Duplicate")
	if conflict.code == 0 || !strings.Contains(conflict.stderr, "Conflict:") {
		t.Fatalf("stale transition did not render Conflict: exit=%d stderr=%s", conflict.code, conflict.stderr)
	}

	// Handoff, blocker, resolution, and human reject/rework/accept.
	succeed("send", "-workspace", dir, "-workspace-id", wsID, "-reviewer", reviewer, "-caller", "engineer", "-request-id", "r5", "-message", "m1", "-recipient", "analyst", "-kind", "Request", "-body", "Please investigate", "-sender", "claimed-engineer", "-task", "t1")
	pending := call("messages", "-workspace", dir, "-workspace-id", wsID, "-recipient", "analyst")
	if pending.code != 0 || !strings.Contains(pending.stdout, "Pending messages: 1") || !strings.Contains(pending.stdout, "Message m1") || !strings.Contains(pending.stdout, "Acknowledged: (absent)") {
		t.Fatalf("pending message list did not expose the waiting handoff: exit=%d stdout=%s stderr=%s", pending.code, pending.stdout, pending.stderr)
	}
	succeed("ack", "-workspace", dir, "-workspace-id", wsID, "-caller", "analyst", "-request-id", "r6", "-message", "m1")
	cleared := call("messages", "-workspace", dir, "-workspace-id", wsID, "-recipient", "analyst")
	if cleared.code != 0 || !strings.Contains(cleared.stdout, "Pending messages: 0") {
		t.Fatalf("acknowledged handoff remained pending: exit=%d stdout=%s stderr=%s", cleared.code, cleared.stdout, cleared.stderr)
	}
	succeed("transition", "-workspace", dir, "-workspace-id", wsID, "-reviewer", reviewer, "-caller", "engineer", "-request-id", "r7", "-task", "t1", "-from", "Doing", "-to", "Blocked", "-reason", "Need evidence")
	status("Blocked")
	succeed("transition", "-workspace", dir, "-workspace-id", wsID, "-reviewer", reviewer, "-caller", "engineer", "-request-id", "r8", "-task", "t1", "-from", "Blocked", "-to", "Doing")
	status("Doing")
	succeed("report", "-workspace", dir, "-workspace-id", wsID, "-reviewer", reviewer, "-caller", "engineer", "-request-id", "r9", "-task", "t1", "-result", "res1", "-expected-revision", "4", "-summary", "Evidence attached", "-artifact", "report.md")
	status("AwaitingReview")

	denied := call("accept", "-workspace", dir, "-workspace-id", wsID, "-reviewer", reviewer, "-caller", "engineer", "-request-id", "r10-denied", "-task", "t1", "-result", "res1", "-expected-revision", "5")
	if denied.code == 0 || !strings.Contains(denied.stderr, "Denied:") {
		t.Fatalf("non-reviewer accept did not render Denied: exit=%d stderr=%s", denied.code, denied.stderr)
	}
	succeed("reject", "-workspace", dir, "-workspace-id", wsID, "-reviewer", reviewer, "-caller", reviewer, "-request-id", "r11", "-task", "t1", "-result", "res1", "-expected-revision", "5", "-reason", "Add evidence")
	status("Doing")
	succeed("report", "-workspace", dir, "-workspace-id", wsID, "-reviewer", reviewer, "-caller", "engineer", "-request-id", "r12", "-task", "t1", "-result", "res2", "-expected-revision", "6", "-summary", "Evidence complete", "-artifact", "final.md")
	status("AwaitingReview")
	succeed("accept", "-workspace", dir, "-workspace-id", wsID, "-reviewer", reviewer, "-caller", reviewer, "-request-id", "r13", "-task", "t1", "-result", "res2", "-expected-revision", "7")
	status("Done")

	// Every command invocation closes its workspace. This final independent
	// process is the product's close/reopen proof; no work is resent.
	final := succeed("task", "-workspace", dir, "-workspace-id", wsID, "t1")
	if !strings.Contains(final, "ResultID:   res2") {
		t.Fatalf("reopened process lost accepted replacement: %q", final)
	}
	reporter := outputField(final, "  Reporter:")
	if reporter == "" {
		t.Fatalf("reopened task omitted reporter field: %q", final)
	}
	reporterVerification := strings.TrimSuffix(reporter[strings.LastIndex(reporter, ", ")+2:], ")")
	if reporterVerification != string(domain.IdentityUnverified) {
		t.Fatalf("task reporter identity verification = %q, want %q", reporterVerification, domain.IdentityUnverified)
	}
	if outputField(final, "  Last update:") == "" {
		t.Fatalf("reopened task omitted last-update field: %q", final)
	}

	// The message query makes the handoff and its provenance visible through
	// the product surface rather than through a host-side test query.
	message := call("message", "-workspace", dir, "-workspace-id", wsID, "m1")
	if message.code != 0 {
		t.Fatalf("message query failed: exit=%d stdout=%s stderr=%s", message.code, message.stdout, message.stderr)
	}
	for _, want := range []string{
		"Sender:      claimed-engineer",
		"Queued:", "Published:   (absent)", "Processed:   (absent)",
		"Acknowledged:", "by analyst", "Task:        t1",
	} {
		if !strings.Contains(message.stdout, want) {
			t.Fatalf("message output missing %q: %s", want, message.stdout)
		}
	}
	recordedBy := outputField(message.stdout, "  Recorded by:")
	if recordedBy == "" {
		t.Fatalf("message output omitted Recorded by field: %s", message.stdout)
	}
	identityVerification := strings.TrimSuffix(recordedBy[strings.LastIndex(recordedBy, ", ")+2:], ")")
	if identityVerification != string(domain.IdentityUnverified) {
		t.Fatalf("Recorded by identity verification = %q, want %q", identityVerification, domain.IdentityUnverified)
	}
}

func outputField(output, prefix string) string {
	for _, line := range strings.Split(output, "\n") {
		if strings.HasPrefix(line, prefix) {
			return strings.TrimSpace(strings.TrimPrefix(line, prefix))
		}
	}
	return ""
}
