package main

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// TestCLI_StartRun_LayerAParticipationFixture is Phase 3 item 1's Layer
// A CI proof (H101-147): a real subprocess, launched through the
// shipped `harnessing start-run` surface under continuing-host
// semantics for state, actually spawns a PARTICIPATING fixture — not a
// sleep/cat stand-in (D9) — that reads injected task/workspace context
// (R2) and leaves an observable participation marker this test asserts
// on directly, then proves the run reaches Starting->Running (D2)
// through `harnessing run`, the product surface, not an internal store
// dump (D6).
func TestCLI_StartRun_LayerAParticipationFixture(t *testing.T) {
	if runtime.GOOS != "darwin" && runtime.GOOS != "linux" {
		t.Skip("workspace operation is Unsupported on this platform")
	}

	binary := filepath.Join(t.TempDir(), "harnessing")
	build := exec.Command("go", "build", "-o", binary, ".")
	if output, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build harnessing binary: %v\n%s", output, err)
	}

	dir := t.TempDir()
	const wsID = "ws-start-run-layer-a"

	run := func(args ...string) spawnedCLIResult {
		t.Helper()
		return buildAndRunCLI(t, binary, args...)
	}
	succeed := func(args ...string) spawnedCLIResult {
		t.Helper()
		result := run(args...)
		if result.code != 0 {
			t.Fatalf("%v: exit=%d stdout=%s stderr=%s", args, result.code, result.stdout, result.stderr)
		}
		return result
	}

	succeed("register", "-workspace", dir, "-workspace-id", wsID, "-caller", "agent-a", "-request-id", "reg-1", "-agent", "agent-a", "-display-name", "Agent A")

	succeed("approve-profile", "-workspace", dir, "-workspace-id", wsID, "-reviewer", "human1", "-caller", "human1", "-request-id", "approve-1",
		"-profile-id", "fixture-profile", "-tool-executable", binary,
		"-tool-argv-json", `["__fixture-participate"]`, "-context-transport", "context-file")

	startResult := succeed("start-run", "-workspace", dir, "-workspace-id", wsID, "-run", "run-1", "-agent", "agent-a", "-profile-id", "fixture-profile", "-task", "task-1")
	if !strings.Contains(startResult.stdout, "OK") {
		t.Fatalf("start-run stdout = %q, want an OK receipt", startResult.stdout)
	}

	runResult := succeed("run", "-workspace", dir, "-workspace-id", wsID, "run-1")
	if !strings.Contains(runResult.stdout, "State:      Running") {
		t.Fatalf("harnessing run output = %q, want State: Running", runResult.stdout)
	}

	// R2: the fixture actually received and echoed the injected context
	// — find the run's own context artifact and its ".participated"
	// sibling the fixture wrote after reading it.
	entries, err := os.ReadDir(filepath.Join(dir, "runs"))
	if err != nil {
		t.Fatalf("reading runs dir: %v", err)
	}
	var participatedPath string
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), ".context.json.participated") {
			participatedPath = filepath.Join(dir, "runs", e.Name())
		}
	}
	if participatedPath == "" {
		t.Fatalf("no participation marker found in %s; fixture did not participate", filepath.Join(dir, "runs"))
	}
	data, err := os.ReadFile(participatedPath)
	if err != nil {
		t.Fatalf("reading participation marker: %v", err)
	}
	var context map[string]any
	if err := json.Unmarshal(data, &context); err != nil {
		t.Fatalf("participation marker is not valid JSON: %v", err)
	}
	if context["taskId"] != "task-1" {
		t.Fatalf("participation marker taskId = %v, want task-1 — R2 context was not actually delivered", context["taskId"])
	}
	if context["runId"] != "run-1" || context["agentId"] != "agent-a" {
		t.Fatalf("participation marker = %v, missing expected run/agent identifiers", context)
	}
}

// TestCLI_StartRun_UnapprovedProfileDenied is D1/R5 through the real
// CLI: an installed, PATH-resolvable executable is not an approved
// profile.
func TestCLI_StartRun_UnapprovedProfileDenied(t *testing.T) {
	if runtime.GOOS != "darwin" && runtime.GOOS != "linux" {
		t.Skip("workspace operation is Unsupported on this platform")
	}
	binary := filepath.Join(t.TempDir(), "harnessing")
	build := exec.Command("go", "build", "-o", binary, ".")
	if output, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build harnessing binary: %v\n%s", output, err)
	}
	dir := t.TempDir()
	const wsID = "ws-start-run-denied"

	run := func(args ...string) spawnedCLIResult { return buildAndRunCLI(t, binary, args...) }
	if r := run("register", "-workspace", dir, "-workspace-id", wsID, "-caller", "agent-a", "-request-id", "reg-1", "-agent", "agent-a", "-display-name", "Agent A"); r.code != 0 {
		t.Fatalf("register: exit=%d stderr=%s", r.code, r.stderr)
	}

	result := run("start-run", "-workspace", dir, "-workspace-id", wsID, "-run", "run-1", "-agent", "agent-a", "-profile-id", "never-approved")
	if result.code == 0 {
		t.Fatalf("start-run against an unapproved profile succeeded; want a rejection")
	}
	if !strings.Contains(result.stderr, "Denied") {
		t.Fatalf("start-run stderr = %q, want a Denied error", result.stderr)
	}
}

// TestCLI_StartRun_MissingToolIsHonestNotRunning is D10: an approved
// profile naming a nonexistent executable fails with a stable,
// actionable error — never Running, never silent success.
func TestCLI_StartRun_MissingToolIsHonestNotRunning(t *testing.T) {
	if runtime.GOOS != "darwin" && runtime.GOOS != "linux" {
		t.Skip("workspace operation is Unsupported on this platform")
	}
	binary := filepath.Join(t.TempDir(), "harnessing")
	build := exec.Command("go", "build", "-o", binary, ".")
	if output, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build harnessing binary: %v\n%s", output, err)
	}
	dir := t.TempDir()
	const wsID = "ws-start-run-missing-tool"

	run := func(args ...string) spawnedCLIResult { return buildAndRunCLI(t, binary, args...) }
	if r := run("register", "-workspace", dir, "-workspace-id", wsID, "-caller", "agent-a", "-request-id", "reg-1", "-agent", "agent-a", "-display-name", "Agent A"); r.code != 0 {
		t.Fatalf("register: exit=%d stderr=%s", r.code, r.stderr)
	}
	if r := run("approve-profile", "-workspace", dir, "-workspace-id", wsID, "-reviewer", "human1", "-caller", "human1", "-request-id", "approve-1",
		"-profile-id", "ghost-profile", "-tool-executable", "/definitely/not/a/real/executable-xyz"); r.code != 0 {
		t.Fatalf("approve-profile: exit=%d stderr=%s", r.code, r.stderr)
	}

	result := run("start-run", "-workspace", dir, "-workspace-id", wsID, "-run", "run-1", "-agent", "agent-a", "-profile-id", "ghost-profile")
	if result.code == 0 {
		t.Fatalf("start-run against a missing executable succeeded; want a rejection")
	}
	if !strings.Contains(result.stderr, "MissingTool") {
		t.Fatalf("start-run stderr = %q, want a MissingTool error", result.stderr)
	}

	runResult := run("run", "-workspace", dir, "-workspace-id", wsID, "run-1")
	if runResult.code != 0 || !strings.Contains(runResult.stdout, "State:      Exited") {
		t.Fatalf("harnessing run output = %q (exit %d), want State: Exited — never left dangling as Starting", runResult.stdout, runResult.code)
	}
}
