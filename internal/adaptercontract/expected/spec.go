// Package expected holds scenario literals derived from ADR 0003, boundaries.md,
// and domain rules — compiled before any adapter runs. Nothing here reads
// adapter output.
//
// Where the spec is silent, assumptions are written up in ../AMBIGUITIES.md
// (not in literals) so H101-91 can review them without inferring from tests.
package expected

import (
	"github.com/rafaelcalves/harnessing-101/internal/core/domain"
)

// Workspace and principal IDs used across UI-01..04 scenarios.
const (
	WorkspaceID = "ws-adaptercontract"
	ReviewerID  = "reviewer1"
	EngineerID  = "engineer"
	AnalystID   = "analyst"
	// ClaimedSenderID is UI-04's message-cycle third identity (H101-119):
	// registered separately from the caller (EngineerID) and the
	// recipient (AnalystID) so a submitted SenderAgentID that differs
	// from the caller has an independently expected, named value to
	// compare against, rather than a bare literal at each call site.
	ClaimedSenderID = "claimed-engineer"
	TaskID          = "t1"
	MessageID       = "m1"
	ResultID1       = "res1"
	ResultID2       = "res2"
)

// Reviewers returns the human-review authority set for contract workspaces.
func Reviewers() []domain.AgentID {
	return []domain.AgentID{domain.AgentID(ReviewerID)}
}

// Receipt records an independently expected successful commit outcome.
// Revision follows boundaries.md H101-94: start at 0; each atomic commit
// group advances once (not per field/event).
type Receipt struct {
	RequestID domain.RequestID
	Revision  uint64
}

// UI01ValidRegister is the expected outcome after one successful register
// on an empty workspace (ADR UI-01: IDs and body preserved).
var UI01ValidRegister = struct {
	Receipt Receipt
	Agent   domain.AgentID
	Name    string
}{
	Receipt: Receipt{RequestID: "ui01-r1", Revision: 1},
	Agent:   domain.AgentID(EngineerID),
	Name:    "Engineer",
}

// UI03IdempotentRegister is the receipt for the first successful register in
// the idempotency scenario (UI-03).
var UI03IdempotentRegister = Receipt{RequestID: "ui03-r1", Revision: 1}

// UI04CycleEnd is the expected domain state at the equivalent committed cut
// after the full UI-04 cycle (boundaries.md H101-94 "UI-04 final cut",
// amended per H101-109/H101-23): 13 user-request groups + 2 delivery-record
// groups = revision 15. H101-109 requires SendMessage's SenderAgentID to be
// a registered agent, so the cycle's claimed sender ("claimed-engineer",
// deliberately distinct from -caller to prove the claim is independent of
// authorship) is now a third registered agent, one more commit than the
// original H101-94 cut counted.
var UI04CycleEnd = struct {
	WorkspaceRevision uint64
	TaskStatus        domain.TaskStatus
	TaskTitle         string
	TaskAssignee      domain.AgentID
	CurrentResultID   domain.ResultID
	AgentCount        int
}{
	WorkspaceRevision: 15,
	TaskStatus:        domain.TaskDone,
	TaskTitle:         "Investigate",
	TaskAssignee:      domain.AgentID(EngineerID),
	CurrentResultID:   domain.ResultID(ResultID2),
	AgentCount:        3, // engineer, analyst, and claimed-engineer; reviewer is authority not an agent record
}

// Stable domain error codes adapters must surface (ADR UI-02).
const (
	CodeInvalidArgument = string(domain.ErrInvalidArgument)
	CodeDenied          = string(domain.ErrDenied)
	CodeConflict        = string(domain.ErrConflict)
	CodeNotFound        = string(domain.ErrNotFound)
)
