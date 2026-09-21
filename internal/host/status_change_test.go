package host_test

import (
	"context"
	"testing"

	"github.com/rafaelcalves/harnessing-101/internal/core/domain"
	"github.com/rafaelcalves/harnessing-101/internal/core/task"
	"github.com/rafaelcalves/harnessing-101/internal/host"
)

// TestCapabilities_GetTask_LastStatusChangeIsDetached exercises the
// H101-70 field through the actual composition boundary (cloneTask),
// not just the store's own fresh-decode behavior: mutating a returned
// StatusChange, including its FromStatus pointer, must never reach what
// a later GetTask call returns.
func TestCapabilities_GetTask_LastStatusChangeIsDetached(t *testing.T) {
	caps, err := host.Open(t.TempDir(), "ws-status", nil)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = caps.Close() }()
	ctx := context.Background()

	if _, err := caps.RegisterAgent(ctx, "engineer", task.RegisterAgentRequest{
		RequestID: "reg-r1", AgentID: "engineer", DisplayName: "Engineer",
	}); err != nil {
		t.Fatalf("RegisterAgent: %v", err)
	}
	if _, err := caps.CreateTask(ctx, "engineer", task.CreateTaskRequest{
		RequestID: "r1", TaskID: "t1", Title: "x", AssigneeID: "engineer",
	}); err != nil {
		t.Fatalf("CreateTask: %v", err)
	}
	if _, err := caps.TransitionTask(ctx, "engineer", task.TransitionTaskRequest{
		RequestID: "r2", TaskID: "t1", FromStatus: domain.TaskTodo, ToStatus: domain.TaskDoing,
	}); err != nil {
		t.Fatalf("TransitionTask: %v", err)
	}

	got, err := caps.GetTask(ctx, "t1")
	if err != nil {
		t.Fatalf("GetTask: %v", err)
	}
	if got.LastStatusChange == nil || got.LastStatusChange.FromStatus == nil {
		t.Fatalf("expected a populated LastStatusChange with a FromStatus, got %+v", got.LastStatusChange)
	}
	*got.LastStatusChange.FromStatus = "tampered"
	got.LastStatusChange.ToStatus = "tampered"

	reread, err := caps.GetTask(ctx, "t1")
	if err != nil {
		t.Fatalf("GetTask (reread): %v", err)
	}
	if reread.LastStatusChange.ToStatus == "tampered" || *reread.LastStatusChange.FromStatus == "tampered" {
		t.Fatalf("mutating a returned StatusChange reached the store: %+v", reread.LastStatusChange)
	}
}
