package main

import (
	"bytes"
	"context"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/rafaelcalves/harnessing-101/internal/core/domain"
	"github.com/rafaelcalves/harnessing-101/internal/host"
)

// TestRun_Phase2CycleRecoversAfterKilledCLI proves the product-level claim
// left open by H101-49: a real CLI-driven workspace with in-progress work can
// be reopened after the CLI process holding it is killed without Close. The
// hold command supplies the long-lived command boundary; the cycle itself is
// driven through run(), the same dispatch used by main.
func TestRun_Phase2CycleRecoversAfterKilledCLI(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("workspace operation is Unsupported on Windows")
	}
	probe, err := host.Open(t.TempDir(), "platform-probe", nil)
	if err != nil {
		if derr, ok := err.(*domain.Error); ok && derr.Code == domain.ErrUnsupported {
			t.Skip("workspace operation is Unsupported on this platform")
		}
		t.Fatalf("platform workspace probe: %v", err)
	}
	_ = probe.Close()

	dir := t.TempDir()
	const wsID = "ws-crash-cycle"

	runCLI := func(args ...string) string {
		t.Helper()
		var out, errOut bytes.Buffer
		if code := run(args, &out, &errOut); code != 0 {
			t.Fatalf("%v: exit code = %d; stdout=%s stderr=%s", args, code, out.String(), errOut.String())
		}
		return out.String()
	}

	// Leave the task AwaitingReview with an acknowledged handoff and a
	// durable result: this is real product state, not an empty lock fixture.
	runCLI("register", "-workspace", dir, "-workspace-id", wsID, "-reviewer", "reviewer1", "-caller", "engineer", "-request-id", "r1", "-agent", "engineer", "-display-name", "Engineer")
	runCLI("create", "-workspace", dir, "-workspace-id", wsID, "-reviewer", "reviewer1", "-caller", "engineer", "-request-id", "r2", "-task", "t1", "-title", "Investigate", "-assignee", "engineer")
	runCLI("transition", "-workspace", dir, "-workspace-id", wsID, "-reviewer", "reviewer1", "-caller", "engineer", "-request-id", "r3", "-task", "t1", "-from", "Todo", "-to", "Doing")
	runCLI("send", "-workspace", dir, "-workspace-id", wsID, "-reviewer", "reviewer1", "-caller", "engineer", "-request-id", "r4", "-message", "m1", "-recipient", "reviewer1", "-kind", "Request", "-body", "Please review", "-task", "t1")
	runCLI("ack", "-workspace", dir, "-workspace-id", wsID, "-caller", "reviewer1", "-request-id", "r5", "-message", "m1")
	runCLI("report", "-workspace", dir, "-workspace-id", wsID, "-reviewer", "reviewer1", "-caller", "engineer", "-request-id", "r6", "-task", "t1", "-result", "res1", "-expected-revision", "2", "-summary", "Evidence attached", "-artifact", "report.md")

	hold, output := startHoldHelper(t, []string{"hold", "-workspace", dir, "-workspace-id", wsID, "-reviewer", "reviewer1"})
	line, err := output.ReadString('\n')
	if err != nil || !strings.Contains(line, "holding lock") {
		t.Fatalf("hold did not report holding the cycle workspace (line=%q, err=%v)", line, err)
	}

	// Negative control: recovery must not mean that a second process can open
	// a workspace while its owner is alive.
	if caps, err := host.Open(dir, wsID, nil); err == nil {
		_ = caps.Close()
		t.Fatal("workspace opened while the CLI holder was alive")
	} else if derr, ok := err.(*domain.Error); !ok || derr.Code != domain.ErrBusy {
		t.Fatalf("open while CLI holder alive = %v, want Busy", err)
	}

	if err := hold.Process.Kill(); err != nil {
		t.Fatalf("kill CLI holder: %v", err)
	}
	_ = hold.Wait()

	// Reopen through the product command surface. Retry only for the small
	// interval in which the kernel is finishing descriptor teardown; there is
	// no lock timeout to wait out.
	var taskOutput string
	deadline := time.Now().Add(5 * time.Second)
	for {
		var out, errOut bytes.Buffer
		code := run([]string{"task", "-workspace", dir, "-workspace-id", wsID, "t1"}, &out, &errOut)
		if code == 0 {
			taskOutput = out.String()
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("CLI could not reopen after holder kill: code=%d stderr=%s", code, errOut.String())
		}
		time.Sleep(20 * time.Millisecond)
	}
	if !strings.Contains(taskOutput, "Status:     AwaitingReview") || !strings.Contains(taskOutput, "ResultID:   res1") {
		t.Fatalf("reopened CLI lost the reported result: %q", taskOutput)
	}

	// The task command does not expose message history, so inspect the same
	// reopened workspace through the composition query solely to verify the
	// acknowledged handoff survived alongside the CLI-visible task state.
	caps, err := host.Open(dir, wsID, nil)
	if err != nil {
		t.Fatalf("host reopen after CLI query: %v", err)
	}
	defer func() { _ = caps.Close() }()
	snapshot, err := caps.GetSnapshot(context.Background())
	if err != nil {
		t.Fatalf("snapshot after CLI crash recovery: %v", err)
	}
	if len(snapshot.Messages) != 1 || snapshot.Messages[0].AcknowledgedBy != "reviewer1" || snapshot.Messages[0].AcknowledgedAt == nil {
		t.Fatalf("acknowledged handoff did not survive CLI crash recovery: %+v", snapshot.Messages)
	}
}
