package adaptercontract_test

import (
	"context"
	"testing"
	"time"

	"github.com/rafaelcalves/harnessing-101/internal/adaptercontract/expected"
	"github.com/rafaelcalves/harnessing-101/internal/api"
	"github.com/rafaelcalves/harnessing-101/internal/core/domain"
)

// UI-07: graceful reopen retains state; request-id replay; detach does not cancel other work.
func TestUI07_GracefulReopenRetainsState(t *testing.T) {
	ctx := context.Background()
	for _, driver := range drivers(t) {
		t.Run(driver.Name(), func(t *testing.T) {
			env := newEnv(t)
			runUI04CycleUntilAwaitingReview(t, ctx, driver, env)

			reopened := driver.QueryTask(ctx, env, domain.TaskID(expected.TaskID))
			if reopened.ExitCode != 0 {
				t.Fatalf("reopened query failed: %q", reopened.Stderr)
			}
			combined := reopened.Stdout + reopened.Stderr
			if !containsAll(combined, "AwaitingReview", expected.TaskID) {
				t.Fatalf("graceful reopen lost state: %q", combined)
			}
		})
	}
}

func TestUI07_RequestIDReplayAfterReopen(t *testing.T) {
	ctx := context.Background()
	want := expected.UI03IdempotentRegister
	for _, driver := range drivers(t) {
		t.Run(driver.Name(), func(t *testing.T) {
			env := newEnv(t)
			first := registerAgent(t, ctx, driver, env, expected.EngineerID, string(want.RequestID), "Engineer")
			assertReceiptEqual(t, receiptFromResult(t, first), want)

			second := registerAgent(t, ctx, driver, env, expected.EngineerID, string(want.RequestID), "Engineer")
			assertReceiptEqual(t, receiptFromResult(t, second), want)

			snap := harnessSnapshot(t, env)
			if len(snap.Agents) != 1 {
				t.Fatalf("replay after reopen agent count = %d, want 1", len(snap.Agents))
			}
		})
	}
}

func TestUI07_DetachDoesNotCancelOtherSession(t *testing.T) {
	env := newEnv(t)
	h := openContractHarness(t, env)
	defer func() { _ = h.Close() }()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	observer := h.Bind(domain.AgentID(expected.AnalystID))
	events, err := observer.Subscribe(ctx, "0")
	if err != nil {
		t.Fatalf("Subscribe: %v", err)
	}

	writer := h.Bind(domain.AgentID(expected.EngineerID))
	if _, err := writer.CreateTask(ctx, api.CreateTaskRequest{
		RequestID: "ui07-r1", TaskID: "ui07-t1", Title: "before detach", AssigneeID: domain.AgentID(expected.EngineerID),
	}); err != nil {
		t.Fatalf("CreateTask before detach: %v", err)
	}
	select {
	case <-events:
	case <-time.After(3 * time.Second):
		t.Fatal("observer missed first event")
	}

	cancel()
	if _, err := writer.CreateTask(context.Background(), api.CreateTaskRequest{
		RequestID: "ui07-r2", TaskID: "ui07-t2", Title: "after detach", AssigneeID: domain.AgentID(expected.EngineerID),
	}); err != nil {
		t.Fatalf("CreateTask after observer detach: %v", err)
	}

	snap, err := writer.GetSnapshot(context.Background())
	if err != nil {
		t.Fatalf("GetSnapshot after detach: %v", err)
	}
	if len(snap.Tasks) != 2 {
		t.Fatalf("detach cancelled work: task count = %d, want 2", len(snap.Tasks))
	}
}
