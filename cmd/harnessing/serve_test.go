package main

import (
	"context"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/rafaelcalves/harnessing-101/internal/api"
	"github.com/rafaelcalves/harnessing-101/internal/assembly"
	"github.com/rafaelcalves/harnessing-101/internal/host"
)

// TestRun_ServeAttachRoundTripThenDetach is H101-136's own native proof:
// a real `harnessing serve` subprocess (not an in-process fake) stays up
// across two separate attach/detach cycles, answers a real
// api.FrontendSession call through the file transport for each, and
// still holds the workspace lock between them — the thing a one-shot
// CLI invocation could never do before this card.
func TestRun_ServeAttachRoundTripThenDetach(t *testing.T) {
	dir := t.TempDir()
	cmd, out := startHoldHelper(t, []string{"serve", "-workspace", dir, "-workspace-id", "ws-serve"})

	line, err := out.ReadString('\n')
	if err != nil || !strings.Contains(line, "listening") {
		t.Fatalf("serve did not report listening (line=%q, err=%v)", line, err)
	}

	// A live serve host must hold the workspace lock — no second
	// direct opener while it's up (same expectation `hold` proves).
	if _, err := host.Open(dir, "ws-serve", nil); err == nil {
		t.Fatal("host.Open succeeded while `harnessing serve` is alive")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// First attach: register an agent through the transport, entirely
	// via api.FrontendSession — no direct store/host access from this
	// side either.
	client1, err := assembly.Attach(ctx, dir, "engineer")
	if err != nil {
		t.Fatalf("first Attach: %v", err)
	}
	if _, err := client1.RegisterAgent(ctx, api.RegisterAgentRequest{
		RequestID: "r-attach-1", AgentID: "engineer", DisplayName: "Engineer",
	}); err != nil {
		t.Fatalf("RegisterAgent over transport: %v", err)
	}
	if err := client1.Detach(ctx); err != nil {
		t.Fatalf("first Detach: %v", err)
	}

	// Second, LATER attach must still reach the SAME continuing host and
	// observe what the first session committed — proving the host, not
	// any one client, owns the workspace across attach/detach cycles.
	client2, err := assembly.Attach(ctx, dir, "observer")
	if err != nil {
		t.Fatalf("second Attach: %v", err)
	}
	agent, err := client2.GetAgent(ctx, "engineer")
	if err != nil {
		t.Fatalf("GetAgent over transport on second session: %v", err)
	}
	if agent.DisplayName != "Engineer" {
		t.Fatalf("agent.DisplayName = %q, want %q (first session's write did not survive to the second attach)", agent.DisplayName, "Engineer")
	}
	if err := client2.Detach(ctx); err != nil {
		t.Fatalf("second Detach: %v", err)
	}

	// Serve is still up after both clients detached — ordinary detach
	// never ends host lifetime (ADR 0005).
	if _, err := host.Open(dir, "ws-serve", nil); err == nil {
		t.Fatal("host.Open succeeded while `harnessing serve` is still alive after both clients detached")
	}

	if err := cmd.Process.Signal(syscall.SIGTERM); err != nil {
		t.Fatalf("signal SIGTERM: %v", err)
	}
	closedLine, err := out.ReadString('\n')
	if err != nil || !strings.Contains(closedLine, "closed") {
		t.Fatalf("serve did not report a clean close after SIGTERM (line=%q, err=%v)", closedLine, err)
	}
	if err := cmd.Wait(); err != nil {
		t.Fatalf("serve exited non-zero after SIGTERM: %v", err)
	}

	caps, err := host.Open(dir, "ws-serve", nil)
	if err != nil {
		t.Fatalf("workspace still locked after serve's graceful SIGTERM close: %v", err)
	}
	_ = caps.Close()
}
