package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"

	"github.com/rafaelcalves/harnessing-101/internal/api"
	"github.com/rafaelcalves/harnessing-101/internal/core/domain"
)

// runApproveProfile implements `harnessing approve-profile`: the
// human-reviewer decision (threat-model rule 5) that makes a profileID
// eligible for `harnessing start-run`. -tool-executable and
// -tool-argv-json fix exactly what that profileID may ever launch;
// there is no later edit path in this slice.
func runApproveProfile(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("approve-profile", flag.ContinueOnError)
	fs.SetOutput(stderr)
	wf := addWorkspaceFlags(fs)
	caller := fs.String("caller", "", "human-reviewer agent ID approving this profile (required)")
	requestID := fs.String("request-id", "", "idempotency key for this command (required)")
	profileID := fs.String("profile-id", "", "profile identifier to approve (required)")
	executable := fs.String("tool-executable", "", "approved executable path or PATH lookup name (required)")
	var toolArgvJSON optionalString
	fs.Var(&toolArgvJSON, "tool-argv-json", "JSON array of the approved static arguments (optional; defaults to none)")
	workingDir := fs.String("working-dir", "", "working directory for the launched process (optional)")
	contextTransport := fs.String("context-transport", "", `participation-context transport for this profile: "" or "context-file"`)

	if err := fs.Parse(args); err != nil {
		return 1
	}
	if missing := requireFlags(
		flagValue{"-caller", *caller}, flagValue{"-request-id", *requestID},
		flagValue{"-profile-id", *profileID}, flagValue{"-tool-executable", *executable},
	); missing != "" {
		_, _ = fmt.Fprintf(stderr, "harnessing approve-profile: %s is required\n", missing)
		return 1
	}
	argv, err := parseArgvJSON(toolArgvJSON.Get())
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "harnessing approve-profile: -tool-argv-json: %s\n", err)
		return 1
	}

	return withSession(stderr, wf, domain.AgentID(*caller), "approve-profile", func(ctx context.Context, session api.FrontendSession) int {
		receipt, err := session.ApproveProfile(ctx, api.ApproveProfileRequest{
			RequestID: domain.RequestID(*requestID),
			ProfileID: *profileID,
			Spec: domain.ExecutionSpec{
				Args:             append([]string{*executable}, argv...),
				WorkingDirectory: *workingDir,
				ContextTransport: *contextTransport,
			},
		})
		if err != nil {
			printCommandError(stderr, "approve-profile", *requestID, err)
			return 1
		}
		printReceipt(stdout, "approve-profile", receipt)
		return 0
	})
}

// parseArgvJSON decodes an optional JSON string-array flag value. A nil
// pointer (flag never set) yields an empty, non-error slice — "no extra
// args" is a normal, common case, not something a caller must spell out.
func parseArgvJSON(raw *string) ([]string, error) {
	if raw == nil || *raw == "" {
		return nil, nil
	}
	var argv []string
	if err := json.Unmarshal([]byte(*raw), &argv); err != nil {
		return nil, err
	}
	return argv, nil
}
