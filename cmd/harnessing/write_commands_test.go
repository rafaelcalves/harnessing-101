package main

import (
	"bytes"
	"context"
	"os"
	"strings"
	"testing"

	"github.com/rafaelcalves/harnessing-101/internal/host"
)

// TestRun_FullCycleThroughWriteCommands drives register -> create ->
// transition -> report -> accept entirely through run(), the same
// dispatch a real invocation uses, and checks the task lands Done. This
// is the shape Claudio's H101-49 cycle test needs to exist at all.
func TestRun_FullCycleThroughWriteCommands(t *testing.T) {
	dir := t.TempDir()
	const wsID = "ws-cycle"

	steps := [][]string{
		{"register", "-workspace", dir, "-workspace-id", wsID, "-reviewer", "reviewer1",
			"-caller", "engineer", "-request-id", "r1", "-agent", "engineer", "-display-name", "Engineer"},
		{"create", "-workspace", dir, "-workspace-id", wsID, "-reviewer", "reviewer1",
			"-caller", "engineer", "-request-id", "r2", "-task", "t1", "-title", "do it", "-assignee", "engineer"},
		{"transition", "-workspace", dir, "-workspace-id", wsID, "-reviewer", "reviewer1",
			"-caller", "engineer", "-request-id", "r3", "-task", "t1", "-from", "Todo", "-to", "Doing"},
		{"report", "-workspace", dir, "-workspace-id", wsID, "-reviewer", "reviewer1",
			"-caller", "engineer", "-request-id", "r4", "-task", "t1", "-result", "res1",
			"-expected-revision", "2", "-summary", "done", "-artifact", "a.txt"},
		{"accept", "-workspace", dir, "-workspace-id", wsID, "-reviewer", "reviewer1",
			"-caller", "reviewer1", "-request-id", "r5", "-task", "t1", "-result", "res1", "-expected-revision", "3"},
	}
	for _, args := range steps {
		var out, errOut bytes.Buffer
		if code := run(args, &out, &errOut); code != 0 {
			t.Fatalf("%v: exit code = %d; stdout=%s stderr=%s", args, code, out.String(), errOut.String())
		}
	}

	var out, errOut bytes.Buffer
	if code := run([]string{"task", "-workspace", dir, "-workspace-id", wsID, "t1"}, &out, &errOut); code != 0 {
		t.Fatalf("final task query: exit=%d stderr=%s", code, errOut.String())
	}
	if !strings.Contains(out.String(), "Status:     Done") {
		t.Fatalf("task did not reach Done through the CLI: %q", out.String())
	}
}

// TestRun_AcceptByNonReviewerDeniedAndRendered proves Kelly's rendering
// requirement where it actually becomes reachable: a write command's
// Denied path.
func TestRun_AcceptByNonReviewerDeniedAndRendered(t *testing.T) {
	dir := t.TempDir()
	const wsID = "ws-denied"

	run([]string{"register", "-workspace", dir, "-workspace-id", wsID, "-caller", "engineer", "-request-id", "r1", "-agent", "engineer", "-display-name", "Engineer"}, &bytes.Buffer{}, &bytes.Buffer{})
	run([]string{"create", "-workspace", dir, "-workspace-id", wsID, "-caller", "engineer", "-request-id", "r2", "-task", "t1", "-title", "x", "-assignee", "engineer"}, &bytes.Buffer{}, &bytes.Buffer{})
	run([]string{"transition", "-workspace", dir, "-workspace-id", wsID, "-caller", "engineer", "-request-id", "r3", "-task", "t1", "-from", "Todo", "-to", "Doing"}, &bytes.Buffer{}, &bytes.Buffer{})
	run([]string{"report", "-workspace", dir, "-workspace-id", wsID, "-caller", "engineer", "-request-id", "r4", "-task", "t1", "-result", "res1", "-expected-revision", "2", "-summary", "done", "-artifact", "a.txt"}, &bytes.Buffer{}, &bytes.Buffer{})

	var out, errOut bytes.Buffer
	// engineer is never listed as a -reviewer anywhere in this workspace's
	// history: no command here has any way to grant that authority.
	code := run([]string{"accept", "-workspace", dir, "-workspace-id", wsID,
		"-caller", "engineer", "-request-id", "r5", "-task", "t1", "-result", "res1", "-expected-revision", "3"}, &out, &errOut)
	if code == 0 {
		t.Fatalf("accept by a non-reviewer succeeded: stdout=%s", out.String())
	}
	if !strings.Contains(errOut.String(), "Denied:") {
		t.Fatalf("stderr = %q, want the \"Denied: ...\" rendering", errOut.String())
	}
}

// TestRun_TransitionConflictIsRendered exercises the "Conflict" half of
// Kelly's rendering requirement: a stale -from value.
func TestRun_TransitionConflictIsRendered(t *testing.T) {
	dir := t.TempDir()
	const wsID = "ws-conflict"

	run([]string{"create", "-workspace", dir, "-workspace-id", wsID, "-caller", "engineer", "-request-id", "r1", "-task", "t1", "-title", "x"}, &bytes.Buffer{}, &bytes.Buffer{})

	var out, errOut bytes.Buffer
	// The task is still Todo; claiming -from Doing is stale.
	code := run([]string{"transition", "-workspace", dir, "-workspace-id", wsID,
		"-caller", "engineer", "-request-id", "r2", "-task", "t1", "-from", "Doing", "-to", "Blocked", "-reason", "x"}, &out, &errOut)
	if code == 0 {
		t.Fatalf("transition from a stale status succeeded: stdout=%s", out.String())
	}
	if !strings.Contains(errOut.String(), "Conflict:") {
		t.Fatalf("stderr = %q, want the \"Conflict: ...\" rendering", errOut.String())
	}
}

func TestRun_SendAndAckCycle(t *testing.T) {
	dir := t.TempDir()
	const wsID = "ws-msg"

	// H101-23's ripple: SendMessage now checks both SenderAgentID and
	// RecipientAgentID against the registry, so both must be registered
	// before the send below.
	if code := run([]string{"register", "-workspace", dir, "-workspace-id", wsID,
		"-caller", "engineer", "-request-id", "reg-engineer", "-agent", "engineer", "-display-name", "Engineer"}, &bytes.Buffer{}, &bytes.Buffer{}); code != 0 {
		t.Fatal("register engineer failed")
	}
	if code := run([]string{"register", "-workspace", dir, "-workspace-id", wsID,
		"-caller", "reviewer1", "-request-id", "reg-reviewer1", "-agent", "reviewer1", "-display-name", "Reviewer"}, &bytes.Buffer{}, &bytes.Buffer{}); code != 0 {
		t.Fatal("register reviewer1 failed")
	}

	var out, errOut bytes.Buffer
	if code := run([]string{"send", "-workspace", dir, "-workspace-id", wsID,
		"-caller", "engineer", "-request-id", "r1", "-message", "m1",
		"-recipient", "reviewer1", "-kind", "Inform", "-body", "status update"}, &out, &errOut); code != 0 {
		t.Fatalf("send: exit=%d stderr=%s", code, errOut.String())
	}

	out.Reset()
	errOut.Reset()
	if code := run([]string{"ack", "-workspace", dir, "-workspace-id", wsID,
		"-caller", "reviewer1", "-request-id", "r2", "-message", "m1"}, &out, &errOut); code != 0 {
		t.Fatalf("ack: exit=%d stderr=%s", code, errOut.String())
	}

	// Wrong recipient: engineer never received m1.
	out.Reset()
	errOut.Reset()
	code := run([]string{"ack", "-workspace", dir, "-workspace-id", wsID,
		"-caller", "engineer", "-request-id", "r3", "-message", "m1"}, &out, &errOut)
	if code == 0 {
		t.Fatal("wrong-recipient ack succeeded")
	}
	if !strings.Contains(errOut.String(), "Denied:") {
		t.Fatalf("stderr = %q, want \"Denied: ...\"", errOut.String())
	}
}

func TestRun_UpdateAgentPatchesOnlyGivenField(t *testing.T) {
	dir := t.TempDir()
	const wsID = "ws-update"

	run([]string{"register", "-workspace", dir, "-workspace-id", wsID,
		"-caller", "engineer", "-request-id", "r1", "-agent", "engineer",
		"-display-name", "Engineer", "-profile-id", "profile-a"}, &bytes.Buffer{}, &bytes.Buffer{})

	var out, errOut bytes.Buffer
	code := run([]string{"update", "-workspace", dir, "-workspace-id", wsID,
		"-caller", "engineer", "-request-id", "r2", "-agent", "engineer", "-display-name", "Senior Engineer"}, &out, &errOut)
	if code != 0 {
		t.Fatalf("update: exit=%d stderr=%s", code, errOut.String())
	}

	caps, err := host.Open(dir, wsID, nil)
	if err != nil {
		t.Fatalf("host.Open: %v", err)
	}
	defer func() { _ = caps.Close() }()
	agent, err := caps.GetAgent(context.Background(), "engineer")
	if err != nil {
		t.Fatalf("GetAgent: %v", err)
	}
	if agent.DisplayName != "Senior Engineer" {
		t.Fatalf("DisplayName = %q, want %q", agent.DisplayName, "Senior Engineer")
	}
	if agent.ProfileID != "profile-a" {
		t.Fatalf("ProfileID = %q, want it unchanged at %q", agent.ProfileID, "profile-a")
	}
}

// TestRun_CreateMissingTitleIsRejected checks flag validation renders a
// legible message and touches no state.
func TestRun_CreateMissingTitleIsRejected(t *testing.T) {
	dir := t.TempDir()
	var out, errOut bytes.Buffer
	code := run([]string{"create", "-workspace", dir, "-workspace-id", "ws-x",
		"-caller", "engineer", "-request-id", "r1", "-task", "t1"}, &out, &errOut)
	if code == 0 {
		t.Fatal("create without -title succeeded")
	}
	if !strings.Contains(errOut.String(), "-title is required") {
		t.Fatalf("stderr = %q, want it to name -title", errOut.String())
	}
}

// TestRun_IOFailureRendersAsUncertainNotFailed is the H101-51 amendment
// (UI-02): a write command's IOFailure must not read as a plain
// "failed" — the underlying Commit can return exactly this code after a
// write that already applied but whose durability fsync failed
// (internal/adapters/statestore's writeLocked). There is no separate
// error code distinguishing that from "nothing happened" (Kelly's
// finding), so every IOFailure is rendered as uncertain, and the
// request ID is surfaced so the caller can check state before deciding
// whether to resubmit — not told to just retry.
func TestRun_IOFailureRendersAsUncertainNotFailed(t *testing.T) {
	dir := t.TempDir()
	const wsID = "ws-iofail"

	// Corrupt the persisted state file so the next Commit's read fails
	// IOFailure. This does not reproduce the exact fsync-after-rename
	// case, but it is the same error code with the same ambiguity Kelly
	// found: nothing in the code distinguishes them today, so both must
	// render the same way.
	if code := run([]string{"create", "-workspace", dir, "-workspace-id", wsID,
		"-caller", "engineer", "-request-id", "r1", "-task", "t1", "-title", "x"}, &bytes.Buffer{}, &bytes.Buffer{}); code != 0 {
		t.Fatalf("seed create failed unexpectedly")
	}
	statePath := dir + "/state.json"
	if err := os.WriteFile(statePath, []byte("{not valid json"), 0o644); err != nil {
		t.Fatalf("corrupt state file: %v", err)
	}

	var out, errOut bytes.Buffer
	code := run([]string{"create", "-workspace", dir, "-workspace-id", wsID,
		"-caller", "engineer", "-request-id", "r2-uncertain", "-task", "t2", "-title", "y"}, &out, &errOut)
	if code == 0 {
		t.Fatal("create against a corrupt state file succeeded")
	}
	if !strings.Contains(errOut.String(), "IOFailure") {
		t.Fatalf("stderr = %q, want it to name IOFailure", errOut.String())
	}
	if !strings.Contains(errOut.String(), "UNCERTAIN") {
		t.Fatalf("stderr = %q, want the UNCERTAIN warning, not a plain failure", errOut.String())
	}
	if !strings.Contains(errOut.String(), "r2-uncertain") {
		t.Fatalf("stderr = %q, want the request ID surfaced for recovery", errOut.String())
	}
}
