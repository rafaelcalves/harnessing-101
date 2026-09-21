// Package expected holds scenario literals derived from ADR 0003, boundaries.md,
// and domain rules — compiled before any adapter runs. Nothing here reads
// adapter output.
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
	TaskID      = "t1"
	MessageID   = "m1"
	ResultID1   = "res1"
	ResultID2   = "res2"
)

// Reviewers returns the human-review authority set for contract workspaces.
func Reviewers() []domain.AgentID {
	return []domain.AgentID{domain.AgentID(ReviewerID)}
}

// Receipt records an independently expected successful commit outcome.
// Revision follows the domain rule: each successful command advances the
// workspace revision by one, starting from 0 on an empty workspace.
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

// UI04CycleEnd is the expected domain state after the full UI-04 interaction
// cycle (ADR UI-04 + product definition §2 minimum cycle).
var UI04CycleEnd = struct {
	TaskStatus      domain.TaskStatus
	TaskTitle       string
	TaskAssignee    domain.AgentID
	CurrentResultID domain.ResultID
	AgentCount      int
}{
	TaskStatus:      domain.TaskDone,
	TaskTitle:       "Investigate",
	TaskAssignee:    domain.AgentID(EngineerID),
	CurrentResultID: domain.ResultID(ResultID2),
	AgentCount:      2, // engineer and analyst; reviewer is authority not an agent record
}

// Stable domain error codes adapters must surface (ADR UI-02).
const (
	CodeInvalidArgument = string(domain.ErrInvalidArgument)
	CodeDenied          = string(domain.ErrDenied)
	CodeConflict        = string(domain.ErrConflict)
	CodeNotFound        = string(domain.ErrNotFound)
)
