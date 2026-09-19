package task_test

import (
	"context"
	"testing"

	"github.com/rafaelcalves/harnessing-101/internal/core/domain"
	"github.com/rafaelcalves/harnessing-101/internal/core/task"
)

func TestRegisterAgent_SelfAndHuman(t *testing.T) {
	ctx := context.Background()
	e, closeStore := newEngine(t, t.TempDir())
	defer closeStore()

	// Self-registration is allowed.
	if _, err := e.RegisterAgent(ctx, caller(engineerID, false), task.RegisterAgentRequest{
		RequestID: "req-1", AgentID: engineerID, DisplayName: "Engineer",
	}); err != nil {
		t.Fatalf("self RegisterAgent: %v", err)
	}

	// A host-authorized human may register a different agent.
	if _, err := e.RegisterAgent(ctx, caller(reviewerID, true), task.RegisterAgentRequest{
		RequestID: "req-2", AgentID: "reviewer-2", DisplayName: "Second Reviewer",
	}); err != nil {
		t.Fatalf("human RegisterAgent for another ID: %v", err)
	}

	got, err := e.GetAgent(ctx, engineerID)
	if err != nil {
		t.Fatalf("GetAgent: %v", err)
	}
	if got.Provenance.IdentityVerification != domain.IdentityUnverified {
		t.Fatalf("Agent.Provenance.IdentityVerification = %s, want %s", got.Provenance.IdentityVerification, domain.IdentityUnverified)
	}
	if got.LastUpdatedProvenance != nil {
		t.Fatalf("fresh registration must not have LastUpdatedProvenance set: %+v", got.LastUpdatedProvenance)
	}
}

func TestRegisterAgent_OtherAgentDenied(t *testing.T) {
	ctx := context.Background()
	e, closeStore := newEngine(t, t.TempDir())
	defer closeStore()

	// A non-human agent may not register a DIFFERENT agentID.
	_, err := e.RegisterAgent(ctx, caller(engineerID, false), task.RegisterAgentRequest{
		RequestID: "req-1", AgentID: "someone-else", DisplayName: "Someone Else",
	})
	mustErrorCode(t, err, domain.ErrDenied)

	if _, err := e.GetAgent(ctx, "someone-else"); err == nil {
		t.Fatal("agent should not have been created by a denied call")
	}
}

func TestRegisterAgent_ExistingIDIsConflict(t *testing.T) {
	ctx := context.Background()
	e, closeStore := newEngine(t, t.TempDir())
	defer closeStore()

	if _, err := e.RegisterAgent(ctx, caller(engineerID, false), task.RegisterAgentRequest{
		RequestID: "req-1", AgentID: engineerID, DisplayName: "Engineer",
	}); err != nil {
		t.Fatalf("first RegisterAgent: %v", err)
	}

	// Same payload, but a genuinely new request ID: still Conflict.
	// Idempotency is the store's job for a REPLAYED request ID, not a
	// second domain-level rule about matching payloads.
	_, err := e.RegisterAgent(ctx, caller(engineerID, false), task.RegisterAgentRequest{
		RequestID: "req-2", AgentID: engineerID, DisplayName: "Engineer",
	})
	mustErrorCode(t, err, domain.ErrConflict)
}

func TestRegisterAgent_SameRequestIDReplays(t *testing.T) {
	ctx := context.Background()
	e, closeStore := newEngine(t, t.TempDir())
	defer closeStore()

	req := task.RegisterAgentRequest{RequestID: "req-1", AgentID: engineerID, DisplayName: "Engineer"}
	r1, err := e.RegisterAgent(ctx, caller(engineerID, false), req)
	if err != nil {
		t.Fatalf("first RegisterAgent: %v", err)
	}
	r2, err := e.RegisterAgent(ctx, caller(engineerID, false), req)
	if err != nil {
		t.Fatalf("replayed RegisterAgent: %v", err)
	}
	if r1.CommittedRevision != r2.CommittedRevision {
		t.Fatalf("replay committed a new revision: first=%d second=%d", r1.CommittedRevision, r2.CommittedRevision)
	}
}

func TestUpdateAgent_PatchesOnlyGivenFields(t *testing.T) {
	ctx := context.Background()
	e, closeStore := newEngine(t, t.TempDir())
	defer closeStore()

	if _, err := e.RegisterAgent(ctx, caller(engineerID, false), task.RegisterAgentRequest{
		RequestID: "req-1", AgentID: engineerID, DisplayName: "Engineer", ProfileID: "profile-a",
	}); err != nil {
		t.Fatalf("RegisterAgent: %v", err)
	}

	newName := "Senior Engineer"
	if _, err := e.UpdateAgent(ctx, caller(engineerID, false), task.UpdateAgentRequest{
		RequestID: "req-2", AgentID: engineerID, DisplayName: &newName,
	}); err != nil {
		t.Fatalf("UpdateAgent: %v", err)
	}

	got, err := e.GetAgent(ctx, engineerID)
	if err != nil {
		t.Fatalf("GetAgent: %v", err)
	}
	if got.DisplayName != newName {
		t.Fatalf("DisplayName = %q, want %q", got.DisplayName, newName)
	}
	if got.ProfileID != "profile-a" {
		t.Fatalf("ProfileID changed to %q, want unchanged %q", got.ProfileID, "profile-a")
	}
	if got.ID != engineerID {
		t.Fatalf("ID changed to %q, want unchanged %q", got.ID, engineerID)
	}
	if got.LastUpdatedProvenance == nil {
		t.Fatal("LastUpdatedProvenance must be set after an update")
	}
}

func TestUpdateAgent_OtherAgentDenied(t *testing.T) {
	ctx := context.Background()
	e, closeStore := newEngine(t, t.TempDir())
	defer closeStore()

	if _, err := e.RegisterAgent(ctx, caller(engineerID, false), task.RegisterAgentRequest{
		RequestID: "req-1", AgentID: engineerID, DisplayName: "Engineer",
	}); err != nil {
		t.Fatalf("RegisterAgent: %v", err)
	}

	newName := "Hijacked"
	// Another non-human agent may not update someone else's record —
	// the same class of rule as a non-recipient acknowledging a message.
	_, err := e.UpdateAgent(ctx, caller(reviewerID, false), task.UpdateAgentRequest{
		RequestID: "req-2", AgentID: engineerID, DisplayName: &newName,
	})
	mustErrorCode(t, err, domain.ErrDenied)

	got, err := e.GetAgent(ctx, engineerID)
	if err != nil {
		t.Fatalf("GetAgent: %v", err)
	}
	if got.DisplayName == newName {
		t.Fatal("denied update must not have applied")
	}
}

func TestUpdateAgent_HumanReviewerMayUpdateAnyAgent(t *testing.T) {
	ctx := context.Background()
	e, closeStore := newEngine(t, t.TempDir())
	defer closeStore()

	if _, err := e.RegisterAgent(ctx, caller(engineerID, false), task.RegisterAgentRequest{
		RequestID: "req-1", AgentID: engineerID, DisplayName: "Engineer",
	}); err != nil {
		t.Fatalf("RegisterAgent: %v", err)
	}

	fixedName := "Corrected Name"
	if _, err := e.UpdateAgent(ctx, caller(reviewerID, true), task.UpdateAgentRequest{
		RequestID: "req-2", AgentID: engineerID, DisplayName: &fixedName,
	}); err != nil {
		t.Fatalf("human UpdateAgent for another agent: %v", err)
	}

	got, err := e.GetAgent(ctx, engineerID)
	if err != nil {
		t.Fatalf("GetAgent: %v", err)
	}
	if got.DisplayName != fixedName {
		t.Fatalf("DisplayName = %q, want %q", got.DisplayName, fixedName)
	}
}

func TestUpdateAgent_UnknownAgentNotFound(t *testing.T) {
	ctx := context.Background()
	e, closeStore := newEngine(t, t.TempDir())
	defer closeStore()

	newName := "Nobody"
	_, err := e.UpdateAgent(ctx, caller(engineerID, false), task.UpdateAgentRequest{
		RequestID: "req-1", AgentID: engineerID, DisplayName: &newName,
	})
	mustErrorCode(t, err, domain.ErrNotFound)
}

func TestUpdateAgent_EmptyPatchRejected(t *testing.T) {
	ctx := context.Background()
	e, closeStore := newEngine(t, t.TempDir())
	defer closeStore()

	if _, err := e.RegisterAgent(ctx, caller(engineerID, false), task.RegisterAgentRequest{
		RequestID: "req-1", AgentID: engineerID, DisplayName: "Engineer",
	}); err != nil {
		t.Fatalf("RegisterAgent: %v", err)
	}

	_, err := e.UpdateAgent(ctx, caller(engineerID, false), task.UpdateAgentRequest{
		RequestID: "req-2", AgentID: engineerID,
	})
	mustErrorCode(t, err, domain.ErrInvalidArgument)
}
