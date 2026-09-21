package main

import (
	"context"
	"flag"
	"fmt"
	"io"

	"github.com/rafaelcalves/harnessing-101/internal/api"
	"github.com/rafaelcalves/harnessing-101/internal/core/domain"
)

// runRegister implements `harnessing register`: Capabilities.RegisterAgent.
// -caller is who is invoking the command; -agent is who is being
// registered. They may be the same (self-registration) or different (an
// operator listed in -reviewer registering someone else) — the engine,
// not this file, decides whether that is allowed.
func runRegister(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("register", flag.ContinueOnError)
	fs.SetOutput(stderr)
	wf := addWorkspaceFlags(fs)
	caller := fs.String("caller", "", "agent ID invoking this command (required)")
	requestID := fs.String("request-id", "", "idempotency key for this command (required)")
	agentID := fs.String("agent", "", "agent ID to register (required)")
	displayName := fs.String("display-name", "", "display name for the agent (required)")
	profileID := fs.String("profile-id", "", "optional profile identifier")

	if err := fs.Parse(args); err != nil {
		return 1
	}
	if missing := requireFlags(
		flagValue{"-caller", *caller}, flagValue{"-request-id", *requestID},
		flagValue{"-agent", *agentID}, flagValue{"-display-name", *displayName},
	); missing != "" {
		_, _ = fmt.Fprintf(stderr, "harnessing register: %s is required\n", missing)
		return 1
	}

	return withSession(stderr, wf, domain.AgentID(*caller), "register", func(ctx context.Context, session api.FrontendSession) int {
		receipt, err := session.RegisterAgent(ctx, api.RegisterAgentRequest{
			RequestID:   domain.RequestID(*requestID),
			AgentID:     domain.AgentID(*agentID),
			DisplayName: *displayName,
			ProfileID:   *profileID,
		})
		if err != nil {
			printCommandError(stderr, "register", *requestID, err)
			return 1
		}
		printReceipt(stdout, "register", receipt)
		return 0
	})
}

// runUpdate implements `harnessing update`: Capabilities.UpdateAgent.
// -display-name and -profile-id are an explicit patch — omitting a flag
// leaves that field unchanged, it does not clear it. At least one of
// them must be given; the engine enforces that and this file does not
// duplicate the check.
func runUpdate(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("update", flag.ContinueOnError)
	fs.SetOutput(stderr)
	wf := addWorkspaceFlags(fs)
	caller := fs.String("caller", "", "agent ID invoking this command (required)")
	requestID := fs.String("request-id", "", "idempotency key for this command (required)")
	agentID := fs.String("agent", "", "agent ID to update (required)")
	var displayName, profileID optionalString
	fs.Var(&displayName, "display-name", "new display name (omit to leave unchanged)")
	fs.Var(&profileID, "profile-id", "new profile identifier (omit to leave unchanged)")

	if err := fs.Parse(args); err != nil {
		return 1
	}
	if missing := requireFlags(
		flagValue{"-caller", *caller}, flagValue{"-request-id", *requestID}, flagValue{"-agent", *agentID},
	); missing != "" {
		_, _ = fmt.Fprintf(stderr, "harnessing update: %s is required\n", missing)
		return 1
	}

	return withSession(stderr, wf, domain.AgentID(*caller), "update", func(ctx context.Context, session api.FrontendSession) int {
		receipt, err := session.UpdateAgent(ctx, api.UpdateAgentRequest{
			RequestID:   domain.RequestID(*requestID),
			AgentID:     domain.AgentID(*agentID),
			DisplayName: displayName.Get(),
			ProfileID:   profileID.Get(),
		})
		if err != nil {
			printCommandError(stderr, "update", *requestID, err)
			return 1
		}
		printReceipt(stdout, "update", receipt)
		return 0
	})
}

// flagValue pairs a flag's displayed name with its parsed value, for
// requireFlags.
type flagValue struct {
	name  string
	value string
}

// requireFlags returns the name of the first empty required flag, in the
// order given, or "" if all are set.
func requireFlags(flags ...flagValue) string {
	for _, f := range flags {
		if f.value == "" {
			return f.name
		}
	}
	return ""
}
