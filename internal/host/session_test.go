package host_test

import (
	"context"
	"testing"

	"github.com/rafaelcalves/harnessing-101/internal/core/domain"
	"github.com/rafaelcalves/harnessing-101/internal/core/task"
	"github.com/rafaelcalves/harnessing-101/internal/host"
)

func TestFrontendSession_EmptySnapshotIsDiscoverable(t *testing.T) {
	caps, err := host.Open(t.TempDir(), "empty", nil)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = caps.Close() }()

	snapshot, err := host.BindFrontendSession(caps, "operator").GetSnapshot(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if snapshot.WorkspaceID != "empty" || snapshot.Cursor == "" {
		t.Fatalf("snapshot identity = workspace %q, cursor %q", snapshot.WorkspaceID, snapshot.Cursor)
	}
	if snapshot.Agents == nil || snapshot.Tasks == nil || snapshot.TaskResults == nil || snapshot.Messages == nil {
		t.Fatalf("empty collections must be present: %+v", snapshot)
	}
}

func TestFrontendSession_BindsCallerAndDeepDetachesSnapshot(t *testing.T) {
	caps, err := host.Open(t.TempDir(), "session", []domain.AgentID{"reviewer"})
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = caps.Close() }()

	ctx := context.Background()
	reviewer := host.BindFrontendSession(caps, "reviewer")
	engineer := host.BindFrontendSession(caps, "engineer")
	if _, err := reviewer.RegisterAgent(ctx, task.RegisterAgentRequest{RequestID: "register-engineer", AgentID: "engineer", DisplayName: "Engineer"}); err != nil {
		t.Fatal(err)
	}
	if _, err := reviewer.CreateTask(ctx, task.CreateTaskRequest{RequestID: "create-task", TaskID: "task-1", Title: "Inspect", AssigneeID: "engineer"}); err != nil {
		t.Fatal(err)
	}
	if _, err := engineer.TransitionTask(ctx, task.TransitionTaskRequest{RequestID: "start-task", TaskID: "task-1", FromStatus: domain.TaskTodo, ToStatus: domain.TaskDoing}); err != nil {
		t.Fatal(err)
	}
	if _, err := engineer.ReportTaskResult(ctx, task.ReportTaskResultRequest{RequestID: "report", TaskID: "task-1", ResultID: "result-1", ExpectedTaskRevision: 2, Summary: "ready", Artifacts: []string{"report.md"}}); err != nil {
		t.Fatal(err)
	}
	if _, err := engineer.SendMessage(ctx, task.SendMessageRequest{RequestID: "send", MessageID: "message-1", SenderAgentID: "engineer", RecipientAgentID: "reviewer", Kind: domain.MessageResult, Body: "ready", TaskID: func() *domain.TaskID { id := domain.TaskID("task-1"); return &id }()}); err != nil {
		t.Fatal(err)
	}

	snapshot, err := reviewer.GetSnapshot(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if snapshot.Revision == 0 || snapshot.Cursor == "" || len(snapshot.Agents) != 1 || len(snapshot.Tasks) != 1 || len(snapshot.TaskResults) != 1 || len(snapshot.Messages) != 1 {
		t.Fatalf("incomplete snapshot: %+v", snapshot)
	}
	// Probe every nested mutable field exposed by the snapshot.
	snapshot.Agents[0].LastUpdatedProvenance = &domain.Provenance{ClaimedAgentID: "mutated"}
	snapshot.Tasks[0].CurrentResultID = func() *domain.ResultID { id := domain.ResultID("mutated"); return &id }()
	snapshot.TaskResults[0].Artifacts[0] = "mutated"
	snapshot.Messages[0].TaskID = func() *domain.TaskID { id := domain.TaskID("mutated"); return &id }()
	snapshot.Messages[0].QueuedAt = nil

	fresh, err := reviewer.GetSnapshot(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if fresh.Tasks[0].CurrentResultID == nil || *fresh.Tasks[0].CurrentResultID != "result-1" || fresh.TaskResults[0].Artifacts[0] != "report.md" || fresh.Messages[0].TaskID == nil || *fresh.Messages[0].TaskID != "task-1" || fresh.Messages[0].QueuedAt == nil {
		t.Fatalf("snapshot mutation leaked into workspace: %+v", fresh)
	}

	if _, err := engineer.AcceptTaskResult(ctx, task.AcceptTaskResultRequest{RequestID: "accept-by-engineer", TaskID: "task-1", ResultID: "result-1", ExpectedTaskRevision: 3}); err == nil {
		t.Fatal("caller-bound engineer session accepted a result")
	}
}
