package task

import (
	"context"
	"testing"
	"time"

	"github.com/rafaelcalves/harnessing-101/internal/adapters/clock"
	"github.com/rafaelcalves/harnessing-101/internal/adapters/idsource"
	"github.com/rafaelcalves/harnessing-101/internal/adapters/statestore"
	"github.com/rafaelcalves/harnessing-101/internal/core/domain"
)

// fakeSupervisor is a white-box test double for ports.ProcessSupervisor:
// exactly the interface StartRun's dispatch-ordering invariant needs to
// prove, with none of internal/adapters/process's real spawning — those
// native, subprocess-driven scenarios live in cmd/harnessing's own test
// suite (start_run_test.go, Layer A's participation fixture), which is
// what actually launches a process. This file proves the ENGINE's own
// ordering guarantee in isolation from that.
type fakeSupervisor struct {
	startErr error
	calls    int
}

func (f *fakeSupervisor) Capabilities(context.Context) (any, error) { return nil, nil }
func (f *fakeSupervisor) Start(context.Context, domain.RunID, domain.ExecutionSpec, domain.RunParticipationContext) error {
	f.calls++
	return f.startErr
}
func (f *fakeSupervisor) Observe(context.Context, domain.RunID) (<-chan any, error) { return nil, nil }
func (f *fakeSupervisor) Stop(context.Context, domain.RunID, time.Duration) error   { return nil }
func (f *fakeSupervisor) Recover(context.Context, domain.RunID) error               { return nil }

// startRunMustErrorCode is this file's own copy of engine_test.go's
// mustErrorCode: that helper lives in package task_test (black-box) and
// is not reachable from here, a white-box package task file — needed
// so afterDispatchMarkerBeforeStart's test seam is in scope.
func startRunMustErrorCode(t *testing.T, err error, want domain.ErrorCode) {
	t.Helper()
	derr, ok := err.(*domain.Error)
	if !ok {
		t.Fatalf("error = %T %v, want *domain.Error with code %s", err, err, want)
	}
	if derr.Code != want {
		t.Fatalf("error code = %s, want %s (detail: %s)", derr.Code, want, derr.Detail)
	}
}

func newStartRunEngine(t *testing.T, sup *fakeSupervisor) *Engine {
	t.Helper()
	store, err := statestore.Open(t.TempDir())
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() { _ = store.Close() })
	e := NewEngine(store, clock.NewSystem(), idsource.Random{}, "ws-start-run")
	e.SetProcessSupervisor(sup)
	return e
}

func mustRegisterAndApprove(t *testing.T, ctx context.Context, e *Engine, agentID domain.AgentID, profileID string) {
	t.Helper()
	if _, err := e.RegisterAgent(ctx, CallerScope{AgentID: agentID}, RegisterAgentRequest{
		RequestID: "reg-" + domain.RequestID(agentID), AgentID: agentID, DisplayName: string(agentID),
	}); err != nil {
		t.Fatalf("RegisterAgent: %v", err)
	}
	if _, err := e.ApproveProfile(ctx, CallerScope{AgentID: "human1", IsHumanReviewer: true}, ApproveProfileRequest{
		RequestID: "approve-" + domain.RequestID(profileID), ProfileID: profileID,
		Spec: domain.ExecutionSpec{Args: []string{"/bin/true"}},
	}); err != nil {
		t.Fatalf("ApproveProfile: %v", err)
	}
}

// TestStartRun_UnapprovedProfileDenied is decision-table D1/R5: no
// profile approval event exists, so StartRun must reject before ever
// touching the supervisor — an installed executable is not an approved
// profile.
func TestStartRun_UnapprovedProfileDenied(t *testing.T) {
	sup := &fakeSupervisor{}
	e := newStartRunEngine(t, sup)
	ctx := context.Background()
	if _, err := e.RegisterAgent(ctx, CallerScope{AgentID: "agent-a"}, RegisterAgentRequest{
		RequestID: "reg-1", AgentID: "agent-a", DisplayName: "agent-a",
	}); err != nil {
		t.Fatalf("RegisterAgent: %v", err)
	}

	_, err := e.StartRun(ctx, CallerScope{AgentID: "agent-a"}, StartRunRequest{
		RequestID: "r1", RunID: "run-1", AgentID: "agent-a", ProfileID: "never-approved",
	})
	startRunMustErrorCode(t, err, domain.ErrDenied)
	if sup.calls != 0 {
		t.Fatalf("supervisor.Start called %d times for an unapproved profile; want 0", sup.calls)
	}
}

// TestStartRun_ApprovedReachesRunning is D2: an approved profile's
// StartRun reaches observable Running through GetRun, with the marker
// committed before (and exactly one) Start call.
func TestStartRun_ApprovedReachesRunning(t *testing.T) {
	sup := &fakeSupervisor{}
	e := newStartRunEngine(t, sup)
	ctx := context.Background()
	mustRegisterAndApprove(t, ctx, e, "agent-a", "profile-a")

	receipt, err := e.StartRun(ctx, CallerScope{AgentID: "agent-a"}, StartRunRequest{
		RequestID: "r1", RunID: "run-1", AgentID: "agent-a", ProfileID: "profile-a",
	})
	if err != nil {
		t.Fatalf("StartRun: %v", err)
	}
	if receipt.RequestID != "r1" {
		t.Fatalf("receipt.RequestID = %q, want r1", receipt.RequestID)
	}
	if sup.calls != 1 {
		t.Fatalf("supervisor.Start called %d times; want exactly 1", sup.calls)
	}
	run, err := e.GetRun(ctx, "run-1")
	if err != nil {
		t.Fatalf("GetRun: %v", err)
	}
	if run.State != domain.RunRunning {
		t.Fatalf("run.State = %q, want Running", run.State)
	}
	if run.DispatchAttemptedAt == nil || run.StartedAt == nil {
		t.Fatalf("run = %+v, want both DispatchAttemptedAt and StartedAt set", run)
	}
}

// TestStartRun_CallerScopeNotRequestFields is CF1 (D3): authorization
// comes from the CallerScope the host established, never from a field
// on the request payload. StartRunRequest carries no sender-claim field
// at all — proving this means proving the request's OWN AgentID/
// ProfileID are select-what-to-check values, not authority, by showing
// two different callers get the identical outcome for the identical
// request.
func TestStartRun_CallerScopeNotRequestFields(t *testing.T) {
	sup := &fakeSupervisor{}
	e := newStartRunEngine(t, sup)
	ctx := context.Background()
	mustRegisterAndApprove(t, ctx, e, "agent-a", "profile-a")

	for i, callerID := range []domain.AgentID{"someone-else", "another-caller"} {
		runID := "run-" + string(rune('a'+i))
		_, err := e.StartRun(ctx, CallerScope{AgentID: callerID}, StartRunRequest{
			RequestID: domain.RequestID("r-" + string(rune('a'+i))), RunID: runID, AgentID: "agent-a", ProfileID: "profile-a",
		})
		if err != nil {
			t.Fatalf("StartRun as caller %s: %v", callerID, err)
		}
	}
	if sup.calls != 2 {
		t.Fatalf("supervisor.Start called %d times; want 2 (caller identity never gated this)", sup.calls)
	}
}

// TestStartRun_SupervisorFailureRecordsExitedHonestly is R3/D10: a
// classified Start failure is known information, not ambiguous — the
// run is recorded Exited with that reason, and StartRun itself returns
// the classified error rather than Running or silent success.
func TestStartRun_SupervisorFailureRecordsExitedHonestly(t *testing.T) {
	sup := &fakeSupervisor{startErr: &domain.Error{Code: domain.ErrMissingTool, Detail: "not on PATH"}}
	e := newStartRunEngine(t, sup)
	ctx := context.Background()
	mustRegisterAndApprove(t, ctx, e, "agent-a", "profile-a")

	_, err := e.StartRun(ctx, CallerScope{AgentID: "agent-a"}, StartRunRequest{
		RequestID: "r1", RunID: "run-1", AgentID: "agent-a", ProfileID: "profile-a",
	})
	startRunMustErrorCode(t, err, domain.ErrMissingTool)

	run, getErr := e.GetRun(ctx, "run-1")
	if getErr != nil {
		t.Fatalf("GetRun: %v", getErr)
	}
	if run.State != domain.RunExited {
		t.Fatalf("run.State = %q, want Exited", run.State)
	}
	if run.ExitReason != string(domain.ErrMissingTool) {
		t.Fatalf("run.ExitReason = %q, want %q", run.ExitReason, domain.ErrMissingTool)
	}
}

// TestStartRun_DispatchMarkerAmbiguous_NoAutoRedispatch is H101-135's
// named invariant, D12: once the dispatch-attempted marker is durably
// committed, this engine never calls Start a second time for that run
// no matter how it is asked again — the ambiguous window belongs to
// item 4's recovery, not to a retry silently papering over it. The seam
// simulates the whole process crashing between the marker commit and
// the Start call by panicking there and recovering in this test, the
// same technique commit_seam_test.go uses for its own named gap.
func TestStartRun_DispatchMarkerAmbiguous_NoAutoRedispatch(t *testing.T) {
	sup := &fakeSupervisor{}
	e := newStartRunEngine(t, sup)
	ctx := context.Background()
	mustRegisterAndApprove(t, ctx, e, "agent-a", "profile-a")

	orig := afterDispatchMarkerBeforeStart
	afterDispatchMarkerBeforeStart = func() { panic("simulated crash between marker and Start") }
	defer func() { afterDispatchMarkerBeforeStart = orig }()

	func() {
		defer func() { _ = recover() }()
		_, _ = e.StartRun(ctx, CallerScope{AgentID: "agent-a"}, StartRunRequest{
			RequestID: "r1", RunID: "run-1", AgentID: "agent-a", ProfileID: "profile-a",
		})
	}()
	if sup.calls != 0 {
		t.Fatalf("supervisor.Start called %d times before the simulated crash; want 0", sup.calls)
	}
	run, err := e.GetRun(ctx, "run-1")
	if err != nil {
		t.Fatalf("GetRun after simulated crash: %v", err)
	}
	if run.DispatchAttemptedAt == nil {
		t.Fatalf("run.DispatchAttemptedAt is nil; the marker commit itself should have survived the panic")
	}
	if run.StartedAt != nil {
		t.Fatalf("run.StartedAt is set; Start was never actually called")
	}

	afterDispatchMarkerBeforeStart = orig
	_, err = e.StartRun(ctx, CallerScope{AgentID: "agent-a"}, StartRunRequest{
		RequestID: "r2-different-request-id", RunID: "run-1", AgentID: "agent-a", ProfileID: "profile-a",
	})
	startRunMustErrorCode(t, err, domain.ErrConflict)
	if sup.calls != 0 {
		t.Fatalf("supervisor.Start called %d times on retry; want 0 — this engine must never auto-redispatch an ambiguous run", sup.calls)
	}
}

// TestStartRun_DoesNotUpgradePreApprovalMessageProvenance is CF3/D4: a
// message recorded before any profile approval must still read
// Unverified after a later successful StartRun — profile approval
// authorizes a future process launch, it does not retroactively
// validate anything already in the workspace.
func TestStartRun_DoesNotUpgradePreApprovalMessageProvenance(t *testing.T) {
	sup := &fakeSupervisor{}
	e := newStartRunEngine(t, sup)
	ctx := context.Background()

	if _, err := e.RegisterAgent(ctx, CallerScope{AgentID: "agent-a"}, RegisterAgentRequest{RequestID: "reg-a", AgentID: "agent-a", DisplayName: "A"}); err != nil {
		t.Fatalf("RegisterAgent a: %v", err)
	}
	if _, err := e.RegisterAgent(ctx, CallerScope{AgentID: "agent-b"}, RegisterAgentRequest{RequestID: "reg-b", AgentID: "agent-b", DisplayName: "B"}); err != nil {
		t.Fatalf("RegisterAgent b: %v", err)
	}
	if _, err := e.SendMessage(ctx, CallerScope{AgentID: "agent-a"}, SendMessageRequest{
		RequestID: "send-1", MessageID: "msg-1", SenderAgentID: "agent-a", RecipientAgentID: "agent-b",
		Kind: domain.MessageInform, Body: "pre-approval message",
	}); err != nil {
		t.Fatalf("SendMessage: %v", err)
	}

	mustRegisterAndApprove(t, ctx, e, "agent-c", "profile-a")
	if _, err := e.StartRun(ctx, CallerScope{AgentID: "agent-c"}, StartRunRequest{
		RequestID: "start-1", RunID: "run-1", AgentID: "agent-c", ProfileID: "profile-a",
	}); err != nil {
		t.Fatalf("StartRun: %v", err)
	}

	msg, err := e.GetMessage(ctx, "msg-1")
	if err != nil {
		t.Fatalf("GetMessage: %v", err)
	}
	if msg.Provenance.IdentityVerification != domain.IdentityUnverified {
		t.Fatalf("pre-approval message IdentityVerification = %q after a successful StartRun; want it to remain %q (CF3)", msg.Provenance.IdentityVerification, domain.IdentityUnverified)
	}
}
