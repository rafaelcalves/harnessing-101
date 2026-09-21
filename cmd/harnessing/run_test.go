package main

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/rafaelcalves/harnessing-101/internal/core/domain"
	"github.com/rafaelcalves/harnessing-101/internal/core/task"
	"github.com/rafaelcalves/harnessing-101/internal/host"
)

func TestRun_Version(t *testing.T) {
	var out, errOut bytes.Buffer
	code := run([]string{"version"}, &out, &errOut)
	if code != 0 {
		t.Fatalf("exit code = %d, want 0; stderr=%s", code, errOut.String())
	}
	if !strings.Contains(out.String(), "harnessing ") {
		t.Fatalf("stdout = %q, want it to contain %q", out.String(), "harnessing ")
	}
}

func TestRun_DisclosureIsUnconditionalAndKeepsStdoutScriptable(t *testing.T) {
	var out, errOut bytes.Buffer
	code := run([]string{"version"}, &out, &errOut)
	if code != 0 {
		t.Fatalf("exit code = %d, want 0; stderr=%s", code, errOut.String())
	}
	if out.String() != "harnessing dev\n" {
		t.Fatalf("stdout = %q, want stable version output without disclosure", out.String())
	}
	if !strings.HasPrefix(errOut.String(), disclosure+"\n") {
		t.Fatalf("stderr = %q, want disclosure before command output", errOut.String())
	}
	if !strings.Contains(errOut.String(), "does not start, observe, or restrict any agent process in this phase") {
		t.Fatalf("stderr omitted the unconditional process disclosure: %q", errOut.String())
	}
}

func TestRun_DisclosurePrecedesFreshWorkspaceCommand(t *testing.T) {
	var out, errOut bytes.Buffer
	code := run([]string{"task", "-workspace", t.TempDir(), "missing"}, &out, &errOut)
	if code == 0 {
		t.Fatal("exit code = 0 for missing task, want non-zero")
	}
	if !strings.HasPrefix(errOut.String(), disclosure+"\n") {
		t.Fatalf("stderr = %q, want disclosure before task handling", errOut.String())
	}
}

func TestRun_UnknownCommand(t *testing.T) {
	var out, errOut bytes.Buffer
	code := run([]string{"bogus"}, &out, &errOut)
	if code == 0 {
		t.Fatal("exit code = 0 for an unknown command, want non-zero")
	}
	if !strings.Contains(errOut.String(), "unknown command") {
		t.Fatalf("stderr = %q, want it to mention the unknown command", errOut.String())
	}
}

func TestRun_TaskMissingWorkspaceFlag(t *testing.T) {
	var out, errOut bytes.Buffer
	code := run([]string{"task", "t-1"}, &out, &errOut)
	if code == 0 {
		t.Fatal("exit code = 0 with no -workspace, want non-zero")
	}
	if !strings.Contains(errOut.String(), "-workspace is required") {
		t.Fatalf("stderr = %q, want it to say -workspace is required", errOut.String())
	}
}

// seedTask opens a workspace directly through host.Capabilities (the only
// path this test uses to write) and creates one task, then closes it so
// the CLI's own host.Open can take the lock next.
func seedTask(t *testing.T, dir string, taskID domain.TaskID) {
	t.Helper()
	caps, err := host.Open(dir, "ws-cli", nil)
	if err != nil {
		t.Fatalf("host.Open: %v", err)
	}
	defer func() { _ = caps.Close() }()

	ctx := context.Background()
	if _, err := caps.RegisterAgent(ctx, "engineer", task.RegisterAgentRequest{
		RequestID: "req-seed-register", AgentID: "engineer", DisplayName: "Engineer",
	}); err != nil {
		t.Fatalf("RegisterAgent: %v", err)
	}
	if _, err := caps.CreateTask(ctx, "engineer", task.CreateTaskRequest{
		RequestID: "req-seed", TaskID: taskID, Title: "seeded task", AssigneeID: "engineer",
	}); err != nil {
		t.Fatalf("CreateTask: %v", err)
	}
}

func TestRun_TaskDisplaysSeededTask(t *testing.T) {
	dir := t.TempDir()
	seedTask(t, dir, "t-1")

	var out, errOut bytes.Buffer
	code := run([]string{"task", "-workspace", dir, "-workspace-id", "ws-cli", "t-1"}, &out, &errOut)
	if code != 0 {
		t.Fatalf("exit code = %d, want 0; stderr=%s", code, errOut.String())
	}
	if !strings.Contains(out.String(), "Task t-1") {
		t.Fatalf("stdout missing task header: %q", out.String())
	}
	if !strings.Contains(out.String(), "seeded task") {
		t.Fatalf("stdout missing title: %q", out.String())
	}
	if !strings.Contains(out.String(), "Todo") {
		t.Fatalf("stdout missing status: %q", out.String())
	}
}

func TestRun_TaskUnknownIDIsLegibleFailure(t *testing.T) {
	dir := t.TempDir()
	seedTask(t, dir, "t-1")

	var out, errOut bytes.Buffer
	code := run([]string{"task", "-workspace", dir, "-workspace-id", "ws-cli", "does-not-exist"}, &out, &errOut)
	if code == 0 {
		t.Fatal("exit code = 0 for an unknown task, want non-zero")
	}
	if !strings.Contains(errOut.String(), "NotFound") {
		t.Fatalf("stderr = %q, want it to name the NotFound code", errOut.String())
	}
}

// The reviewer set must come only from the -reviewer flag, never from
// workspace content: this task command never even parses one, but the
// end-to-end point is that host.Open only ever sees what this file
// passes to it, and this file only ever reads that from flags.
func TestRun_TaskAcceptsRepeatedReviewerFlag(t *testing.T) {
	dir := t.TempDir()
	seedTask(t, dir, "t-1")

	var out, errOut bytes.Buffer
	code := run([]string{
		"task", "-workspace", dir, "-workspace-id", "ws-cli",
		"-reviewer", "alice", "-reviewer", "bob",
		"t-1",
	}, &out, &errOut)
	if code != 0 {
		t.Fatalf("exit code = %d, want 0; stderr=%s", code, errOut.String())
	}
}

func TestRun_TaskClosesWorkspaceEvenOnQueryFailure(t *testing.T) {
	dir := t.TempDir()
	seedTask(t, dir, "t-1")

	var out, errOut bytes.Buffer
	code := run([]string{"task", "-workspace", dir, "-workspace-id", "ws-cli", "missing"}, &out, &errOut)
	if code == 0 {
		t.Fatal("exit code = 0 for a missing task, want non-zero")
	}

	// If the previous run left the lock held, this Open fails Busy.
	caps, err := host.Open(dir, "ws-cli", nil)
	if err != nil {
		t.Fatalf("workspace still locked after a failed query: %v", err)
	}
	_ = caps.Close()
}
