package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"

	"github.com/rafaelcalves/harnessing-101/internal/api"
	"github.com/rafaelcalves/harnessing-101/internal/assembly"
	"github.com/rafaelcalves/harnessing-101/internal/core/domain"
)

// agentIDList collects a repeatable -reviewer flag into an ordered list
// of domain.AgentID, so the operator can name zero or more reviewers on
// one command line. This is the only source host.Open ever sees for
// human-review authority — no env var, no config file, no default.
type agentIDList []domain.AgentID

func (l *agentIDList) String() string { return fmt.Sprint(*l) }

func (l *agentIDList) Set(value string) error {
	if value == "" {
		return fmt.Errorf("-reviewer must not be empty")
	}
	*l = append(*l, domain.AgentID(value))
	return nil
}

// stringList collects a repeatable flag into an ordered []string, used
// for -artifact.
type stringList []string

func (l *stringList) String() string { return fmt.Sprint(*l) }

func (l *stringList) Set(value string) error {
	*l = append(*l, value)
	return nil
}

// optionalString distinguishes "flag not given" from "flag given as
// empty": Get returns nil unless Set was actually called. UpdateAgent's
// patch semantics need this — leaving a field alone must not look like
// clearing it to empty.
type optionalString struct {
	value *string
}

func (o *optionalString) String() string {
	if o.value == nil {
		return ""
	}
	return *o.value
}

func (o *optionalString) Set(value string) error {
	o.value = &value
	return nil
}

func (o *optionalString) Get() *string { return o.value }

// workspaceFlags is the trio every command that touches a workspace
// needs: root, workspace ID, and the reviewer set. Adding them to a
// FlagSet is one call so no command can accidentally omit -reviewer's
// wiring or reach for a shortcut source instead.
type workspaceFlags struct {
	root      *string
	id        *string
	reviewers agentIDList
}

func addWorkspaceFlags(fs *flag.FlagSet) *workspaceFlags {
	wf := &workspaceFlags{}
	wf.root = fs.String("workspace", "", "path to the workspace root (required)")
	wf.id = fs.String("workspace-id", "default", "workspace identifier stored alongside state")
	fs.Var(&wf.reviewers, "reviewer", "agent ID with human-review authority for this invocation (repeatable)")
	return wf
}

// withSession opens the workspace through the trusted composition root, binds
// the caller once, runs fn with only the neutral session, and closes the
// workspace before returning.
func withSession(stderr io.Writer, wf *workspaceFlags, caller domain.AgentID, cmdName string, fn func(ctx context.Context, session api.FrontendSession) int) int {
	return assembly.WithSession(stderr, *wf.root, domain.WorkspaceID(*wf.id), []domain.AgentID(wf.reviewers), caller, cmdName, fn)
}

// describeError renders a domain.Error with its stable code, so a script
// scraping stderr sees "NotFound"/"Denied"/"Conflict"/etc. rather than a
// bare human-readable sentence it would have to parse. Every command
// funnels its errors through this — write commands are where Denied and
// Conflict actually become reachable, and this is the one place that
// rendering is produced.
func describeError(err error) string {
	var derr *domain.Error
	if errors.As(err, &derr) {
		if derr.Detail == "" {
			return string(derr.Code)
		}
		return fmt.Sprintf("%s: %s", derr.Code, derr.Detail)
	}
	return err.Error()
}

// printReceipt is the one success-rendering used by every write command:
// factual, no celebration, per definition.md's tone section.
func printReceipt(stdout io.Writer, cmdName string, receipt domain.Receipt) {
	_, _ = fmt.Fprintf(stdout, "harnessing %s: OK (request %s, workspace revision %d)\n", cmdName, receipt.RequestID, receipt.CommittedRevision)
}

// printCommandError is the one error-rendering every WRITE command uses
// (read-only `task` does not: it has no commit whose durability could be
// uncertain).
//
// H101-64 closed a real propagation gap Kelly found: the store now
// emits the stable ErrOutcomeUncertain code with structured Effect/
// Confirmation fields (ADR 0004), but until this fix the CLI still
// rendered the OLD generic "IOFailure, might be uncertain" line for
// every mutating failure — the honest error reached the surface as the
// previous generation of itself. This now branches on Code first: an
// OutcomeUncertain error is rendered from ADR 0004's own policy table
// (Effect × Confirmation), not from a code-name string match.
//
// The plain-IOFailure branch stays for exactly the case ADR 0004's
// consequences section names: "until all producers are migrated, an
// unclassified IOFailure from a mutating request must be presented
// conservatively as unconfirmed." Today every producer in this repo
// that can fail after applying a write (the fsync case) has migrated to
// OutcomeUncertain, so this branch is defense-in-depth against a future
// producer that has not, not a live path — but removing it would be
// exactly the regression Kelly found, one call site early.
func printCommandError(stderr io.Writer, cmdName, requestID string, err error) {
	_, _ = fmt.Fprintf(stderr, "harnessing %s: %s\n", cmdName, describeError(err))

	var derr *domain.Error
	if !errors.As(err, &derr) {
		return
	}

	switch derr.Code {
	case domain.ErrOutcomeUncertain:
		_, _ = fmt.Fprintf(stderr, "harnessing %s: %s\n", cmdName, outcomeUncertainGuidance(requestID, derr))
	case domain.ErrIOFailure:
		// Conservative fallback for an as-yet-unclassified producer; see
		// the function doc comment above.
		_, _ = fmt.Fprintf(stderr,
			"harnessing %s: UNCERTAIN, not necessarily failed — the write may have committed even though its durability could not be confirmed. Do NOT resubmit with a new request ID. Check state first (e.g. `harnessing task`), then retry with the SAME request ID (%s) if you need to: an identical retry replays the original result if it already committed, and only proceeds as a fresh attempt if it did not.\n",
			cmdName, requestID)
	}
}

// outcomeUncertainGuidance implements ADR 0004's policy table directly:
// the caller-facing message is a deterministic function of Effect and
// Confirmation, never of Detail's free text and never invented ad hoc
// per call site.
func outcomeUncertainGuidance(requestID string, derr *domain.Error) string {
	switch {
	case derr.Effect == domain.EffectApplied && derr.Confirmation == domain.ConfirmationDurability:
		return fmt.Sprintf(
			"UNCERTAIN: change applied; durability unconfirmed. Do NOT resubmit with a new request ID — check state first (e.g. `harnessing task`), then retry with the SAME request ID (%s) if you still need to: it replays the original result if it already committed.",
			requestID)
	case derr.Effect == domain.EffectApplied && derr.Confirmation == domain.ConfirmationOutcome:
		return fmt.Sprintf(
			"UNCERTAIN: change applied; its outcome confirmation is unavailable. Do NOT resubmit with a new request ID — check state first, then retry with the SAME request ID (%s) if you still need to.",
			requestID)
	case derr.Confirmation == domain.ConfirmationOutcome:
		return fmt.Sprintf(
			"UNCERTAIN: outcome unknown — this command's response was lost, not necessarily its effect. Do not assume it succeeded or failed, and stop any automated follow-on that assumes confirmed success. Check state, then resolve or retry with the SAME request ID (%s), never a new one.",
			requestID)
	default:
		return fmt.Sprintf(
			"UNCERTAIN: outcome unknown for request %s. Do not assume success or failure; check state before deciding whether to retry, and if you do, retry with this SAME request ID, never a new one.",
			requestID)
	}
}
