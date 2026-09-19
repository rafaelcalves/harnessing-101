package host_test

import (
	"context"
	"testing"

	"github.com/rafaelcalves/harnessing-101/internal/core/domain"
	"github.com/rafaelcalves/harnessing-101/internal/core/task"
	"github.com/rafaelcalves/harnessing-101/internal/host"
)

// TestPhase1_ProductCycleThroughComposition proves the Phase 1 product cycle
// through the public composition boundary only. It deliberately does not
// import or construct an engine, store, or persistence mutation.
func TestPhase1_ProductCycleThroughComposition(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	const workspaceID = domain.WorkspaceID("phase1-cycle")
	const (
		engineerID domain.AgentID = "cycle-engineer"
		analystID  domain.AgentID = "cycle-analyst"
		reviewerID domain.AgentID = "cycle-reviewer"
	)

	caps, err := host.Open(dir, workspaceID, []domain.AgentID{reviewerID})
	if err != nil {
		t.Fatalf("host.Open: %v", err)
	}

	// The human reviewer establishes a roster of two agents through the same
	// returned capabilities that the rest of the cycle uses.
	if _, err := caps.RegisterAgent(ctx, reviewerID, task.RegisterAgentRequest{
		RequestID: "cycle-register-engineer", AgentID: engineerID, DisplayName: "Engineer",
	}); err != nil {
		t.Fatalf("register engineer: %v", err)
	}
	if _, err := caps.RegisterAgent(ctx, reviewerID, task.RegisterAgentRequest{
		RequestID: "cycle-register-analyst", AgentID: analystID, DisplayName: "Analyst",
	}); err != nil {
		t.Fatalf("register analyst: %v", err)
	}

	taskID := domain.TaskID("cycle-task")
	if _, err := caps.CreateTask(ctx, reviewerID, task.CreateTaskRequest{
		RequestID: "cycle-create-task", TaskID: taskID, Title: "Investigate the failure", AssigneeID: engineerID,
	}); err != nil {
		t.Fatalf("create task: %v", err)
	}
	if _, err := caps.TransitionTask(ctx, engineerID, task.TransitionTaskRequest{
		RequestID: "cycle-start-task", TaskID: taskID, FromStatus: domain.TaskTodo, ToStatus: domain.TaskDoing,
	}); err != nil {
		t.Fatalf("start task: %v", err)
	}

	// A task-linked message is the handoff. Its recipient acknowledges it
	// through Capabilities, which is distinct from publication or processing.
	if _, err := caps.SendMessage(ctx, engineerID, task.SendMessageRequest{
		RequestID: "cycle-send-handoff", MessageID: "cycle-handoff", SenderAgentID: engineerID,
		RecipientAgentID: analystID, Kind: domain.MessageRequest, Body: "Please inspect the failure.", TaskID: &taskID,
	}); err != nil {
		t.Fatalf("send handoff: %v", err)
	}
	if _, err := caps.AcknowledgeMessage(ctx, analystID, task.AcknowledgeMessageRequest{
		RequestID: "cycle-ack-handoff", MessageID: "cycle-handoff",
	}); err != nil {
		t.Fatalf("acknowledge handoff: %v", err)
	}

	if _, err := caps.ReportTaskResult(ctx, engineerID, task.ReportTaskResultRequest{
		RequestID: "cycle-report-1", TaskID: taskID, ResultID: "cycle-result-1", ExpectedTaskRevision: 2,
		Summary: "The first explanation needs review.", Artifacts: []string{"artifacts/first-report.md"},
	}); err != nil {
		t.Fatalf("report first result: %v", err)
	}
	if _, err := caps.RejectTaskResult(ctx, reviewerID, task.RejectTaskResultRequest{
		RequestID: "cycle-reject-1", TaskID: taskID, ResultID: "cycle-result-1", ExpectedTaskRevision: 3,
		Reason: "Add the missing reproduction evidence.",
	}); err != nil {
		t.Fatalf("reject first result: %v", err)
	}
	got, err := caps.GetTask(ctx, taskID)
	if err != nil {
		t.Fatalf("get task after rejection: %v", err)
	}
	if got.Status != domain.TaskDoing {
		t.Fatalf("status after rejection = %s, want Doing", got.Status)
	}

	if _, err := caps.ReportTaskResult(ctx, engineerID, task.ReportTaskResultRequest{
		RequestID: "cycle-report-2", TaskID: taskID, ResultID: "cycle-result-2", ExpectedTaskRevision: 4,
		Summary: "Reproduced and fixed with evidence.", Artifacts: []string{"artifacts/final-report.md"},
	}); err != nil {
		t.Fatalf("report corrected result: %v", err)
	}
	if _, err := caps.AcceptTaskResult(ctx, reviewerID, task.AcceptTaskResultRequest{
		RequestID: "cycle-accept-2", TaskID: taskID, ResultID: "cycle-result-2", ExpectedTaskRevision: 5,
	}); err != nil {
		t.Fatalf("accept corrected result: %v", err)
	}
	if err := caps.Close(); err != nil {
		t.Fatalf("close before reopen: %v", err)
	}

	// Reopen through the composition entry point and verify the task, roster,
	// task-linked handoff, and recipient acknowledgement all survived.
	caps, err = host.Open(dir, workspaceID, []domain.AgentID{reviewerID})
	if err != nil {
		t.Fatalf("reopen host: %v", err)
	}
	defer func() { _ = caps.Close() }()

	for _, id := range []domain.AgentID{engineerID, analystID} {
		if _, err := caps.GetAgent(ctx, id); err != nil {
			t.Fatalf("registered agent %q did not survive reopen: %v", id, err)
		}
	}
	got, err = caps.GetTask(ctx, taskID)
	if err != nil {
		t.Fatalf("get task after reopen: %v", err)
	}
	if got.Status != domain.TaskDone || got.CurrentResultID == nil || *got.CurrentResultID != domain.ResultID("cycle-result-2") {
		t.Fatalf("task after reopen = %+v, want Done with cycle-result-2", got)
	}
	message, err := caps.GetMessage(ctx, "cycle-handoff")
	if err != nil {
		t.Fatalf("get handoff after reopen: %v", err)
	}
	if message.TaskID == nil || *message.TaskID != taskID {
		t.Fatalf("handoff task link after reopen = %v, want %q", message.TaskID, taskID)
	}
	if message.AcknowledgedBy != analystID || message.AcknowledgedAt == nil {
		t.Fatalf("handoff acknowledgement after reopen = by=%q at=%v", message.AcknowledgedBy, message.AcknowledgedAt)
	}
}
