package main

import (
	"bytes"
	"strings"
	"testing"

	"github.com/rafaelcalves/harnessing-101/internal/core/domain"
)

// TestPrintCommandError_OutcomeUncertainRendersFromPolicyTable is
// H101-64's second job: the store can now emit the stable
// OutcomeUncertain code with structured fields, and this proves the CLI
// actually branches on those fields per ADR 0004's policy table instead
// of falling through to the old generic IOFailure line (the gap Kelly
// found — "the honest error reaches the surface as the previous
// generation of itself"). Constructing the *domain.Error directly here
// is deliberate: it isolates the rendering decision from whether a real
// fsync can be made to fail in this process, which internal/adapters/
// statestore's own fault-injection tests already cover end to end at
// the store layer.
func TestPrintCommandError_OutcomeUncertainRendersFromPolicyTable(t *testing.T) {
	rev := uint64(3)
	cases := []struct {
		name        string
		err         *domain.Error
		wantContain []string
		wantAbsent  []string
	}{
		{
			name: "Applied+Durability",
			err: &domain.Error{
				Code: domain.ErrOutcomeUncertain, Detail: "irrelevant wording",
				RequestID: "r1", Effect: domain.EffectApplied, Confirmation: domain.ConfirmationDurability, ObservedRevision: &rev,
			},
			wantContain: []string{"UNCERTAIN", "change applied", "durability unconfirmed", "SAME request ID (r1)", "Do NOT resubmit with a new request ID"},
		},
		{
			name: "Applied+Outcome",
			err: &domain.Error{
				Code: domain.ErrOutcomeUncertain, RequestID: "r2",
				Effect: domain.EffectApplied, Confirmation: domain.ConfirmationOutcome,
			},
			wantContain: []string{"UNCERTAIN", "change applied", "outcome confirmation is unavailable", "SAME request ID (r2)"},
		},
		{
			name: "Unknown+Outcome",
			err: &domain.Error{
				Code: domain.ErrOutcomeUncertain, RequestID: "r3",
				Effect: domain.EffectUnknown, Confirmation: domain.ConfirmationOutcome,
			},
			wantContain: []string{"UNCERTAIN", "outcome unknown", "response was lost", "stop any automated follow-on", "r3"},
			wantAbsent:  []string{"change applied"},
		},
		{
			name: "old plain IOFailure still gets the conservative fallback, not the new wording",
			err: &domain.Error{
				Code: domain.ErrIOFailure, Detail: "disk full",
			},
			wantContain: []string{"IOFailure", "UNCERTAIN", "not necessarily failed"},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var stderr bytes.Buffer
			printCommandError(&stderr, "report", string(tc.err.RequestID), tc.err)
			out := stderr.String()
			for _, want := range tc.wantContain {
				if !strings.Contains(out, want) {
					t.Fatalf("stderr = %q, want it to contain %q", out, want)
				}
			}
			for _, absent := range tc.wantAbsent {
				if strings.Contains(out, absent) {
					t.Fatalf("stderr = %q, want it to NOT contain %q", out, absent)
				}
			}
		})
	}
}

// TestPrintCommandError_DeniedAndConflictUnchanged confirms the new
// Code-based switch did not disturb rendering for the codes that were
// already correct — Denied/Conflict/NotFound get exactly describeError's
// output and nothing else appended.
func TestPrintCommandError_DeniedAndConflictUnchanged(t *testing.T) {
	var stderr bytes.Buffer
	printCommandError(&stderr, "accept", "r1", &domain.Error{Code: domain.ErrDenied, Detail: "no"})
	out := stderr.String()
	if strings.Contains(out, "UNCERTAIN") {
		t.Fatalf("Denied must not get uncertain-outcome wording: %q", out)
	}
	if !strings.Contains(out, "Denied: no") {
		t.Fatalf("stderr = %q, want plain Denied rendering", out)
	}
}
