package task_test

import (
	"context"
	"testing"

	"github.com/rafaelcalves/harnessing-101/internal/core/domain"
	"github.com/rafaelcalves/harnessing-101/internal/core/task"
)

// TestLastStatusChange_SetOnCreationAndEveryRealTransition is H101-70's
// core requirement: the latest-status record exists from creation and is
// replaced atomically on every actual status change, carrying the bound
// caller's provenance and the task revision it belongs to.
func TestLastStatusChange_SetOnCreationAndEveryRealTransition(t *testing.T) {
	ctx := context.Background()
	e, closeStore := newEngine(t, t.TempDir())
	defer closeStore()

	if _, err := e.RegisterAgent(ctx, caller(engineerID, false), task.RegisterAgentRequest{
		RequestID: "reg-r1", AgentID: engineerID, DisplayName: "Engineer",
	}); err != nil {
		t.Fatalf("RegisterAgent: %v", err)
	}

	if _, err := e.CreateTask(ctx, caller(engineerID, false), task.CreateTaskRequest{
		RequestID: "r1", TaskID: "t1", Title: "x", AssigneeID: engineerID,
	}); err != nil {
		t.Fatalf("CreateTask: %v", err)
	}
	got, err := e.GetTask(ctx, "t1")
	if err != nil {
		t.Fatalf("GetTask: %v", err)
	}
	if got.LastStatusChange == nil {
		t.Fatal("creation must set LastStatusChange, got nil")
	}
	if got.LastStatusChange.FromStatus != nil {
		t.Fatalf("creation's FromStatus = %v, want nil (no prior status)", *got.LastStatusChange.FromStatus)
	}
	if got.LastStatusChange.ToStatus != domain.TaskTodo || got.LastStatusChange.TaskRevision != 1 {
		t.Fatalf("creation record = %+v, want ToStatus Todo, TaskRevision 1", got.LastStatusChange)
	}
	if got.LastStatusChange.Provenance.ClaimedAgentID != engineerID {
		t.Fatalf("creation record actor = %q, want %q", got.LastStatusChange.Provenance.ClaimedAgentID, engineerID)
	}

	if _, err := e.TransitionTask(ctx, caller(engineerID, false), task.TransitionTaskRequest{
		RequestID: "r2", TaskID: "t1", FromStatus: domain.TaskTodo, ToStatus: domain.TaskDoing,
	}); err != nil {
		t.Fatalf("TransitionTask: %v", err)
	}
	got, err = e.GetTask(ctx, "t1")
	if err != nil {
		t.Fatalf("GetTask: %v", err)
	}
	if got.LastStatusChange == nil || got.LastStatusChange.FromStatus == nil ||
		*got.LastStatusChange.FromStatus != domain.TaskTodo || got.LastStatusChange.ToStatus != domain.TaskDoing {
		t.Fatalf("after transition, record = %+v, want Todo->Doing", got.LastStatusChange)
	}
	if got.LastStatusChange.TaskRevision != got.Revision {
		t.Fatalf("record's TaskRevision %d != task's own Revision %d — must stay in lockstep", got.LastStatusChange.TaskRevision, got.Revision)
	}
}

// TestLastStatusChange_ReplayAndDenialDoNotAdvanceIt: a retry of an
// already-recorded request, and a denied command, must not touch the
// latest-status record — only an ACTUAL status change may.
func TestLastStatusChange_ReplayAndDenialDoNotAdvanceIt(t *testing.T) {
	ctx := context.Background()
	e, closeStore := newEngine(t, t.TempDir())
	defer closeStore()

	if _, err := e.RegisterAgent(ctx, caller(engineerID, false), task.RegisterAgentRequest{
		RequestID: "reg-r1", AgentID: engineerID, DisplayName: "Engineer",
	}); err != nil {
		t.Fatalf("RegisterAgent: %v", err)
	}

	if _, err := e.CreateTask(ctx, caller(engineerID, false), task.CreateTaskRequest{
		RequestID: "r1", TaskID: "t1", Title: "x", AssigneeID: engineerID,
	}); err != nil {
		t.Fatalf("CreateTask: %v", err)
	}
	req := task.TransitionTaskRequest{RequestID: "r2", TaskID: "t1", FromStatus: domain.TaskTodo, ToStatus: domain.TaskDoing}
	if _, err := e.TransitionTask(ctx, caller(engineerID, false), req); err != nil {
		t.Fatalf("TransitionTask: %v", err)
	}
	before, err := e.GetTask(ctx, "t1")
	if err != nil {
		t.Fatalf("GetTask: %v", err)
	}

	// Identical replay: same request ID, same payload.
	if _, err := e.TransitionTask(ctx, caller(engineerID, false), req); err != nil {
		t.Fatalf("TransitionTask (replay): %v", err)
	}
	after, err := e.GetTask(ctx, "t1")
	if err != nil {
		t.Fatalf("GetTask: %v", err)
	}
	// Compare by value, not by struct equality: GetTask deep-clones
	// FromStatus on every call, so before/after hold different pointer
	// addresses even when unchanged — a plain != would always report
	// "different" regardless of the actual values.
	if before.LastStatusChange.TaskRevision != after.LastStatusChange.TaskRevision ||
		before.LastStatusChange.ToStatus != after.LastStatusChange.ToStatus ||
		before.LastStatusChange.Provenance != after.LastStatusChange.Provenance ||
		*before.LastStatusChange.FromStatus != *after.LastStatusChange.FromStatus {
		t.Fatalf("replay changed the status record: before %+v, after %+v", *before.LastStatusChange, *after.LastStatusChange)
	}

	// A denied reopen attempt (no human reviewer authority) must also
	// leave it untouched.
	_, err = e.TransitionTask(ctx, caller(engineerID, false), task.TransitionTaskRequest{
		RequestID: "r3", TaskID: "t1", FromStatus: domain.TaskDoing, ToStatus: domain.TaskBlocked, Reason: "stuck",
	})
	if err != nil {
		t.Fatalf("Doing->Blocked: %v", err)
	}
	blocked, err := e.GetTask(ctx, "t1")
	if err != nil {
		t.Fatalf("GetTask: %v", err)
	}
	if blocked.LastStatusChange.ToStatus != domain.TaskBlocked {
		t.Fatalf("expected the Blocked transition itself to advance the record, got %+v", blocked.LastStatusChange)
	}
}

// TestLastStatusChange_DetachedFromQueryResult proves a caller mutating
// a returned StatusChange (or its FromStatus pointer) cannot reach the
// stored record.
func TestLastStatusChange_DetachedFromQueryResult(t *testing.T) {
	ctx := context.Background()
	e, closeStore := newEngine(t, t.TempDir())
	defer closeStore()

	if _, err := e.RegisterAgent(ctx, caller(engineerID, false), task.RegisterAgentRequest{
		RequestID: "reg-r1", AgentID: engineerID, DisplayName: "Engineer",
	}); err != nil {
		t.Fatalf("RegisterAgent: %v", err)
	}

	if _, err := e.CreateTask(ctx, caller(engineerID, false), task.CreateTaskRequest{
		RequestID: "r1", TaskID: "t1", Title: "x", AssigneeID: engineerID,
	}); err != nil {
		t.Fatalf("CreateTask: %v", err)
	}
	if _, err := e.TransitionTask(ctx, caller(engineerID, false), task.TransitionTaskRequest{
		RequestID: "r2", TaskID: "t1", FromStatus: domain.TaskTodo, ToStatus: domain.TaskDoing,
	}); err != nil {
		t.Fatalf("TransitionTask: %v", err)
	}

	got, err := e.GetTask(ctx, "t1")
	if err != nil {
		t.Fatalf("GetTask: %v", err)
	}
	*got.LastStatusChange.FromStatus = "tampered"
	got.LastStatusChange.ToStatus = "tampered"

	reread, err := e.GetTask(ctx, "t1")
	if err != nil {
		t.Fatalf("GetTask (reread): %v", err)
	}
	if reread.LastStatusChange.ToStatus == "tampered" || *reread.LastStatusChange.FromStatus == "tampered" {
		t.Fatalf("mutating a returned StatusChange reached stored state: %+v", reread.LastStatusChange)
	}
}
