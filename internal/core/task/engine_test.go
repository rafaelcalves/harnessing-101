package task_test

import (
	"context"
	"errors"
	"testing"

	"github.com/rafaelcalves/harnessing-101/internal/adapters/clock"
	"github.com/rafaelcalves/harnessing-101/internal/adapters/idsource"
	"github.com/rafaelcalves/harnessing-101/internal/adapters/statestore"
	"github.com/rafaelcalves/harnessing-101/internal/core/domain"
	"github.com/rafaelcalves/harnessing-101/internal/core/task"
)

const (
	workspaceID = domain.WorkspaceID("ws-1")
	engineerID  = domain.AgentID("engineer")
	reviewerID  = domain.AgentID("reviewer")
)

func newEngine(t *testing.T, dir string) (*task.Engine, func()) {
	t.Helper()
	store, err := statestore.Open(dir)
	if err != nil {
		t.Fatalf("Open(%q): %v", dir, err)
	}
	e := task.NewEngine(store, clock.NewSystem(), idsource.Random{}, workspaceID)
	return e, func() { _ = store.Close() }
}

func caller(id domain.AgentID, isHuman bool) task.CallerScope {
	return task.CallerScope{AgentID: id, IsHumanReviewer: isHuman}
}

func mustErrorCode(t *testing.T, err error, want domain.ErrorCode) {
	t.Helper()
	if err == nil {
		t.Fatalf("expected error with code %s, got nil", want)
	}
	var derr *domain.Error
	if !errors.As(err, &derr) {
		t.Fatalf("expected *domain.Error with code %s, got %T: %v", want, err, err)
	}
	if derr.Code != want {
		t.Fatalf("expected code %s, got %s (%v)", want, derr.Code, derr)
	}
}

func createDoingTask(t *testing.T, ctx context.Context, e *task.Engine, taskID domain.TaskID) {
	t.Helper()
	if _, err := e.RegisterAgent(ctx, caller(engineerID, false), task.RegisterAgentRequest{
		RequestID: domain.RequestID("reg-" + string(taskID)), AgentID: engineerID, DisplayName: "Engineer",
	}); err != nil {
		t.Fatalf("RegisterAgent: %v", err)
	}
	if _, err := e.CreateTask(ctx, caller(engineerID, false), task.CreateTaskRequest{
		RequestID:  domain.RequestID("req-create-" + string(taskID)),
		TaskID:     taskID,
		Title:      "do the thing",
		AssigneeID: engineerID,
	}); err != nil {
		t.Fatalf("CreateTask: %v", err)
	}
	if _, err := e.TransitionTask(ctx, caller(engineerID, false), task.TransitionTaskRequest{
		RequestID:  domain.RequestID("req-start-" + string(taskID)),
		TaskID:     taskID,
		FromStatus: domain.TaskTodo,
		ToStatus:   domain.TaskDoing,
	}); err != nil {
		t.Fatalf("TransitionTask Todo->Doing: %v", err)
	}
}

// Happy path: the full report/accept cycle actually reaches Done, and
// only through the dedicated commands.
func TestReportAndAcceptCycle(t *testing.T) {
	ctx := context.Background()
	e, closeStore := newEngine(t, t.TempDir())
	defer closeStore()

	taskID := domain.TaskID("t-happy")
	createDoingTask(t, ctx, e, taskID)

	if _, err := e.ReportTaskResult(ctx, caller(engineerID, false), task.ReportTaskResultRequest{
		RequestID:            "req-report-1",
		TaskID:               taskID,
		ResultID:             "res-1",
		ExpectedTaskRevision: 2,
		Summary:              "fixed it",
		Artifacts:            []string{"artifacts/patch.diff"},
	}); err != nil {
		t.Fatalf("ReportTaskResult: %v", err)
	}

	got, err := e.GetTask(ctx, taskID)
	if err != nil {
		t.Fatalf("GetTask: %v", err)
	}
	if got.Status != domain.TaskAwaitingReview {
		t.Fatalf("status after report = %s, want AwaitingReview", got.Status)
	}
	if got.Provenance.IdentityVerification != domain.IdentityUnverified {
		t.Fatalf("task provenance IdentityVerification = %s, want %s", got.Provenance.IdentityVerification, domain.IdentityUnverified)
	}

	if _, err := e.AcceptTaskResult(ctx, caller(reviewerID, true), task.AcceptTaskResultRequest{
		RequestID:            "req-accept-1",
		TaskID:               taskID,
		ResultID:             "res-1",
		ExpectedTaskRevision: 3,
	}); err != nil {
		t.Fatalf("AcceptTaskResult: %v", err)
	}

	got, err = e.GetTask(ctx, taskID)
	if err != nil {
		t.Fatalf("GetTask: %v", err)
	}
	if got.Status != domain.TaskDone {
		t.Fatalf("status after accept = %s, want Done", got.Status)
	}
}

// H101-96: boundaries.md (75166b3) requires inbound creation IDs to be
// explicit and non-empty, and the core must reject rather than allocate
// one implicitly. The command-line adapter always supplied ResultID, so
// this hole was invisible until a second, machine-facing front door
// passed an empty value straight through.
func TestReportTaskResult_EmptyResultIDRejected(t *testing.T) {
	ctx := context.Background()
	e, closeStore := newEngine(t, t.TempDir())
	defer closeStore()

	taskID := domain.TaskID("t-empty-result-id")
	createDoingTask(t, ctx, e, taskID)

	_, err := e.ReportTaskResult(ctx, caller(engineerID, false), task.ReportTaskResultRequest{
		RequestID:            "req-report-empty",
		TaskID:               taskID,
		ResultID:             "",
		ExpectedTaskRevision: 2,
		Summary:              "fixed it",
		Artifacts:            []string{"artifacts/patch.diff"},
	})
	mustErrorCode(t, err, domain.ErrInvalidArgument)

	got, err := e.GetTask(ctx, taskID)
	if err != nil {
		t.Fatalf("GetTask: %v", err)
	}
	if got.Status != domain.TaskDoing {
		t.Fatalf("status after rejected report = %s, want unchanged Doing", got.Status)
	}
}

// H101-108 / Stanley's G5: boundaries.md B:24 says unknown IDs fail
// NotFound. CreateTask's AssigneeID is a command-payload agent ID like
// any other, so an assignee that was never registered must fail the
// same way — this was unenforced in the engine itself, not just in a
// harness fixture that forgot to call RegisterAgent first.
func TestCreateTask_UnregisteredAssigneeNotFound(t *testing.T) {
	ctx := context.Background()
	e, closeStore := newEngine(t, t.TempDir())
	defer closeStore()

	_, err := e.CreateTask(ctx, caller(engineerID, false), task.CreateTaskRequest{
		RequestID:  "req-create-unregistered",
		TaskID:     "t-unregistered-assignee",
		Title:      "assign to nobody",
		AssigneeID: "never-registered",
	})
	mustErrorCode(t, err, domain.ErrNotFound)

	if _, err := e.GetTask(ctx, "t-unregistered-assignee"); err == nil {
		t.Fatal("GetTask found a task that a rejected CreateTask must not have committed")
	}
}

// The product rule the whole review round was about: there is no path
// from Doing to Done that skips acceptance.
func TestDirectDoingToDoneRejected(t *testing.T) {
	ctx := context.Background()
	e, closeStore := newEngine(t, t.TempDir())
	defer closeStore()

	taskID := domain.TaskID("t-skip")
	createDoingTask(t, ctx, e, taskID)

	_, err := e.TransitionTask(ctx, caller(engineerID, false), task.TransitionTaskRequest{
		RequestID:  "req-skip",
		TaskID:     taskID,
		FromStatus: domain.TaskDoing,
		ToStatus:   domain.TaskDone,
	})
	mustErrorCode(t, err, domain.ErrInvalidArgument)

	got, err := e.GetTask(ctx, taskID)
	if err != nil {
		t.Fatalf("GetTask: %v", err)
	}
	if got.Status != domain.TaskDoing {
		t.Fatalf("status = %s, want Doing (transition must not have applied)", got.Status)
	}
}

func TestAcceptByNonHumanDenied(t *testing.T) {
	ctx := context.Background()
	e, closeStore := newEngine(t, t.TempDir())
	defer closeStore()

	taskID := domain.TaskID("t-nonhuman")
	createDoingTask(t, ctx, e, taskID)
	if _, err := e.ReportTaskResult(ctx, caller(engineerID, false), task.ReportTaskResultRequest{
		RequestID: "req-report", TaskID: taskID, ResultID: "res-1", ExpectedTaskRevision: 2,
		Summary: "done", Artifacts: []string{"a.txt"},
	}); err != nil {
		t.Fatalf("ReportTaskResult: %v", err)
	}

	// The assignee agent itself is not a human reviewer.
	_, err := e.AcceptTaskResult(ctx, caller(engineerID, false), task.AcceptTaskResultRequest{
		RequestID: "req-accept", TaskID: taskID, ResultID: "res-1", ExpectedTaskRevision: 3,
	})
	mustErrorCode(t, err, domain.ErrDenied)

	got, err := e.GetTask(ctx, taskID)
	if err != nil {
		t.Fatalf("GetTask: %v", err)
	}
	if got.Status != domain.TaskAwaitingReview {
		t.Fatalf("status = %s, want AwaitingReview (denied accept must not have applied)", got.Status)
	}
}

func TestStaleResultAcceptanceConflict(t *testing.T) {
	ctx := context.Background()
	e, closeStore := newEngine(t, t.TempDir())
	defer closeStore()

	taskID := domain.TaskID("t-stale")
	createDoingTask(t, ctx, e, taskID)
	if _, err := e.ReportTaskResult(ctx, caller(engineerID, false), task.ReportTaskResultRequest{
		RequestID: "req-report", TaskID: taskID, ResultID: "res-1", ExpectedTaskRevision: 2,
		Summary: "done", Artifacts: []string{"a.txt"},
	}); err != nil {
		t.Fatalf("ReportTaskResult: %v", err)
	}

	t.Run("stale task revision", func(t *testing.T) {
		_, err := e.AcceptTaskResult(ctx, caller(reviewerID, true), task.AcceptTaskResultRequest{
			RequestID: "req-accept-stale-rev", TaskID: taskID, ResultID: "res-1", ExpectedTaskRevision: 999,
		})
		mustErrorCode(t, err, domain.ErrConflict)
	})

	t.Run("superseded result after rework", func(t *testing.T) {
		// Reject, then report a fresh result under a new ID: res-1 is now
		// history, not the task's current result.
		if _, err := e.RejectTaskResult(ctx, caller(reviewerID, true), task.RejectTaskResultRequest{
			RequestID: "req-reject", TaskID: taskID, ResultID: "res-1", ExpectedTaskRevision: 3, Reason: "needs more work",
		}); err != nil {
			t.Fatalf("RejectTaskResult: %v", err)
		}
		if _, err := e.ReportTaskResult(ctx, caller(engineerID, false), task.ReportTaskResultRequest{
			RequestID: "req-report-2", TaskID: taskID, ResultID: "res-2", ExpectedTaskRevision: 4,
			Summary: "actually done", Artifacts: []string{"b.txt"},
		}); err != nil {
			t.Fatalf("ReportTaskResult rework: %v", err)
		}

		_, err := e.AcceptTaskResult(ctx, caller(reviewerID, true), task.AcceptTaskResultRequest{
			RequestID: "req-accept-superseded", TaskID: taskID, ResultID: "res-1", ExpectedTaskRevision: 5,
		})
		mustErrorCode(t, err, domain.ErrConflict)
	})
}

func TestDuplicateDecisionCreatesNoDuplicateFacts(t *testing.T) {
	ctx := context.Background()
	e, closeStore := newEngine(t, t.TempDir())
	defer closeStore()

	taskID := domain.TaskID("t-dup")
	createDoingTask(t, ctx, e, taskID)
	if _, err := e.ReportTaskResult(ctx, caller(engineerID, false), task.ReportTaskResultRequest{
		RequestID: "req-report", TaskID: taskID, ResultID: "res-1", ExpectedTaskRevision: 2,
		Summary: "done", Artifacts: []string{"a.txt"},
	}); err != nil {
		t.Fatalf("ReportTaskResult: %v", err)
	}

	acceptReq := task.AcceptTaskResultRequest{
		RequestID: "req-accept-once", TaskID: taskID, ResultID: "res-1", ExpectedTaskRevision: 3,
	}
	r1, err := e.AcceptTaskResult(ctx, caller(reviewerID, true), acceptReq)
	if err != nil {
		t.Fatalf("first AcceptTaskResult: %v", err)
	}
	afterFirstAccept, err := e.GetTask(ctx, taskID)
	if err != nil {
		t.Fatalf("GetTask (after first accept): %v", err)
	}

	t.Run("same request ID replays, no new fact", func(t *testing.T) {
		r2, err := e.AcceptTaskResult(ctx, caller(reviewerID, true), acceptReq)
		if err != nil {
			t.Fatalf("replayed AcceptTaskResult: %v", err)
		}
		if r2.CommittedRevision != r1.CommittedRevision {
			t.Fatalf("replay committed a new revision: first=%d second=%d", r1.CommittedRevision, r2.CommittedRevision)
		}
		got, err := e.GetTask(ctx, taskID)
		if err != nil {
			t.Fatalf("GetTask: %v", err)
		}
		if got.Status != domain.TaskDone || got.Revision != afterFirstAccept.Revision {
			t.Fatalf("task state changed by replay: status=%s revision=%d, want status=%s revision=%d unchanged from before the replay", got.Status, got.Revision, domain.TaskDone, afterFirstAccept.Revision)
		}
	})

	t.Run("new request ID against an already-decided result is Conflict, not a second accept", func(t *testing.T) {
		_, err := e.AcceptTaskResult(ctx, caller(reviewerID, true), task.AcceptTaskResultRequest{
			RequestID: "req-accept-again", TaskID: taskID, ResultID: "res-1", ExpectedTaskRevision: 3,
		})
		mustErrorCode(t, err, domain.ErrConflict)
	})
}

// The card's fifth required test: AwaitingReview must survive an actual
// process restart, not just an in-memory struct. This closes the store,
// reopens the same directory, and checks the state read back.
func TestAwaitingReviewSurvivesRestart(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	taskID := domain.TaskID("t-restart")

	func() {
		e, closeStore := newEngine(t, dir)
		defer closeStore()
		createDoingTask(t, ctx, e, taskID)
		if _, err := e.ReportTaskResult(ctx, caller(engineerID, false), task.ReportTaskResultRequest{
			RequestID: "req-report", TaskID: taskID, ResultID: "res-1", ExpectedTaskRevision: 2,
			Summary: "done", Artifacts: []string{"a.txt"},
		}); err != nil {
			t.Fatalf("ReportTaskResult: %v", err)
		}
	}()

	// Simulate a restart: a brand-new Engine and FileStore over the same
	// directory, as if the process had exited and been relaunched.
	e2, closeStore2 := newEngine(t, dir)
	defer closeStore2()

	got, err := e2.GetTask(ctx, taskID)
	if err != nil {
		t.Fatalf("GetTask after restart: %v", err)
	}
	if got.Status != domain.TaskAwaitingReview {
		t.Fatalf("status after restart = %s, want AwaitingReview (must not have become Done)", got.Status)
	}
	if got.CurrentResultID == nil || *got.CurrentResultID != domain.ResultID("res-1") {
		t.Fatalf("current result ID lost across restart: %v", got.CurrentResultID)
	}
}
