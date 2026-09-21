package adaptercontract_test

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/rafaelcalves/harnessing-101/internal/adaptercontract/expected"
	"github.com/rafaelcalves/harnessing-101/internal/api"
	"github.com/rafaelcalves/harnessing-101/internal/core/domain"
)

// UI-06: discover without injected IDs, observe from cursor, restart cursor expiry,
// concurrent commit ordering, detachment probe.
func TestUI06_DiscoverWithoutInjectedIDs(t *testing.T) {
	ctx := context.Background()
	for _, driver := range drivers(t) {
		t.Run(driver.Name(), func(t *testing.T) {
			env := newEnv(t)
			registerAgent(t, ctx, driver, env, expected.EngineerID, "ui06-disc-r1", "Engineer")
			createTask(t, ctx, driver, env, "ui06-disc-r2", expected.TaskID, "Discover me", expected.EngineerID)

			switch driver.Name() {
			case "cli":
				cli := driver.(*cliDriver)
				result := invokeCLI(ctx, cli, env, domain.AgentID(expected.EngineerID), "task", expected.TaskID)
				if result.ExitCode != 0 {
					t.Fatalf("task query failed: %q", result.Stderr)
				}
				if !strings.Contains(result.Stdout, "Discover me") {
					t.Fatalf("task discovery failed: %q", result.Stdout)
				}
			case "throwaway":
				td := driver.(*throwawayDriver)
				snap, result := throwawaySnapshot(t, ctx, td, env, domain.AgentID(expected.EngineerID))
				if result.ExitCode != 0 {
					t.Fatalf("snapshot failed: %q", result.Stderr)
				}
				if snap.Cursor == "" {
					t.Fatal("snapshot missing cursor for subscription")
				}
				found := false
				for _, task := range snap.Tasks {
					if task.ID == domain.TaskID(expected.TaskID) {
						found = true
						break
					}
				}
				if !found {
					t.Fatalf("snapshot did not enumerate task %s", expected.TaskID)
				}
			}
		})
	}
}

func TestUI06_ObserveFromSnapshotCursor(t *testing.T) {
	env := newEnv(t)
	h := openContractHarness(t, env)
	defer func() { _ = h.Close() }()
	ctx := context.Background()
	observer := h.Bind(domain.AgentID(expected.AnalystID))
	snap, err := observer.GetSnapshot(ctx)
	if err != nil {
		t.Fatalf("GetSnapshot: %v", err)
	}
	events, err := observer.Subscribe(ctx, snap.Cursor)
	if err != nil {
		t.Fatalf("Subscribe: %v", err)
	}

	writer := h.Bind(domain.AgentID(expected.EngineerID))
	if _, err := writer.CreateTask(ctx, api.CreateTaskRequest{
		RequestID: "ui06-obs-r1", TaskID: "ui06-t2", Title: "After cursor", AssigneeID: domain.AgentID(expected.EngineerID),
	}); err != nil {
		t.Fatalf("CreateTask: %v", err)
	}

	select {
	case ev := <-events:
		if len(ev.SubjectIDs) != 1 || ev.SubjectIDs[0] != "ui06-t2" {
			t.Fatalf("event subjects = %v, want [ui06-t2]", ev.SubjectIDs)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("timed out waiting for post-cursor event")
	}
}

func TestUI06_StaleCursorAfterRestart(t *testing.T) {
	env := newEnv(t)
	ctx := context.Background()

	first := openContractHarness(t, env)
	snap, err := first.Bind(domain.AgentID(expected.EngineerID)).GetSnapshot(ctx)
	if err != nil {
		t.Fatalf("GetSnapshot: %v", err)
	}
	if _, err := first.Bind(domain.AgentID(expected.EngineerID)).CreateTask(ctx, api.CreateTaskRequest{
		RequestID: "ui06-restart-r2", TaskID: "ui06-t2", Title: "gap", AssigneeID: domain.AgentID(expected.EngineerID),
	}); err != nil {
		t.Fatalf("CreateTask: %v", err)
	}
	staleCursor := snap.Cursor
	if err := first.Close(); err != nil {
		t.Fatalf("Close first harness: %v", err)
	}

	harnessSubscribeCursorExpired(t, env, staleCursor)

	second := openContractHarness(t, env)
	defer func() { _ = second.Close() }()
	fresh, err := second.Bind(domain.AgentID(expected.EngineerID)).GetSnapshot(ctx)
	if err != nil {
		t.Fatalf("GetSnapshot after restart: %v", err)
	}
	events, err := second.Bind(domain.AgentID(expected.EngineerID)).Subscribe(ctx, fresh.Cursor)
	if err != nil {
		t.Fatalf("Subscribe fresh cursor: %v", err)
	}
	if _, err := second.Bind(domain.AgentID(expected.EngineerID)).CreateTask(ctx, api.CreateTaskRequest{
		RequestID: "ui06-restart-r3", TaskID: "ui06-t3", Title: "after restart", AssigneeID: domain.AgentID(expected.EngineerID),
	}); err != nil {
		t.Fatalf("CreateTask after restart: %v", err)
	}
	select {
	case <-events:
	case <-time.After(3 * time.Second):
		t.Fatal("fresh cursor after restart did not deliver events")
	}
}

func TestUI06_ConcurrentCommitsPublishInOrder(t *testing.T) {
	env := newEnv(t)
	h := openContractHarness(t, env)
	defer func() { _ = h.Close() }()
	ctx := context.Background()
	if _, err := h.Bind(domain.AgentID(expected.EngineerID)).RegisterAgent(ctx, api.RegisterAgentRequest{
		RequestID: "ui06-conc-r1", AgentID: domain.AgentID(expected.EngineerID), DisplayName: "Engineer",
	}); err != nil {
		t.Fatalf("RegisterAgent: %v", err)
	}
	harnessConcurrentCommitsInOrder(t, h, 20)
}

func TestUI06_DetachmentProbe(t *testing.T) {
	env := newEnv(t)
	registerAgent(t, context.Background(), drivers(t)[0], env, expected.EngineerID, "ui06-det-r1", "Engineer")
	createTask(t, context.Background(), drivers(t)[0], env, "ui06-det-r2", expected.TaskID, "Detach", expected.EngineerID)
	harnessDetachedSnapshotProbe(t, env)
}
