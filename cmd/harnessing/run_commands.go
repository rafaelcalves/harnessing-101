package main

import (
	"context"
	"flag"
	"fmt"
	"io"

	"github.com/rafaelcalves/harnessing-101/internal/adapters/idsource"
	"github.com/rafaelcalves/harnessing-101/internal/api"
	"github.com/rafaelcalves/harnessing-101/internal/core/domain"
)

// runStartRun implements `harnessing start-run`: Phase 3 item 1's
// managed start surface (H101-144). -caller defaults to -agent when
// omitted — a run started for one's own agent identity, the same
// self-scope precedent RegisterAgent already uses — and -request-id is
// auto-generated when omitted, matching H101-152's runner CLI contract,
// which calls this command without either flag.
//
// -tool-executable/-tool-argv-json, when given, must name the SAME
// executable the -profile-id was approved for (the engine checks this);
// per-run argv may otherwise vary — task/run identifiers differ every
// invocation by necessity. Omitting them launches the approved
// profile's own Args verbatim, which is what this card's native tests
// and the Layer A participation fixture use.
func runStartRun(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("start-run", flag.ContinueOnError)
	fs.SetOutput(stderr)
	wf := addWorkspaceFlags(fs)
	caller := fs.String("caller", "", "agent ID invoking this command (defaults to -agent if omitted)")
	requestID := fs.String("request-id", "", "idempotency key for this command (auto-generated if omitted)")
	runID := fs.String("run", "", "run identifier (required)")
	agentID := fs.String("agent", "", "agent ID this run is for (required)")
	profileID := fs.String("profile-id", "", "approved profile identifier (required)")
	taskID := fs.String("task", "", "task identifier this run participates in (optional)")
	peerID := fs.String("peer", "", "peer agent identifier for participation context (optional)")
	executable := fs.String("tool-executable", "", "executable to launch for this run (optional; defaults to the approved profile's own executable)")
	var toolArgvJSON optionalString
	fs.Var(&toolArgvJSON, "tool-argv-json", "JSON array of arguments for -tool-executable")

	if err := fs.Parse(args); err != nil {
		return 1
	}
	if missing := requireFlags(
		flagValue{"-run", *runID}, flagValue{"-agent", *agentID}, flagValue{"-profile-id", *profileID},
	); missing != "" {
		_, _ = fmt.Fprintf(stderr, "harnessing start-run: %s is required\n", missing)
		return 1
	}
	argv, err := parseArgvJSON(toolArgvJSON.Get())
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "harnessing start-run: -tool-argv-json: %s\n", err)
		return 1
	}
	if *executable != "" && len(argv) == 0 {
		_, _ = fmt.Fprintln(stderr, "harnessing start-run: -tool-argv-json is required when -tool-executable is set")
		return 1
	}

	effectiveCaller := *caller
	if effectiveCaller == "" {
		effectiveCaller = *agentID
	}
	effectiveRequestID := *requestID
	if effectiveRequestID == "" {
		id, err := idsource.Random{}.NewID(context.Background())
		if err != nil {
			_, _ = fmt.Fprintf(stderr, "harnessing start-run: %s\n", err)
			return 1
		}
		effectiveRequestID = id
	}

	var taskPtr *domain.TaskID
	if *taskID != "" {
		t := domain.TaskID(*taskID)
		taskPtr = &t
	}
	var peerPtr *domain.AgentID
	if *peerID != "" {
		p := domain.AgentID(*peerID)
		peerPtr = &p
	}

	return withSession(stderr, wf, domain.AgentID(effectiveCaller), "start-run", func(ctx context.Context, session api.FrontendSession) int {
		receipt, err := session.StartRun(ctx, api.StartRunRequest{
			RequestID:      domain.RequestID(effectiveRequestID),
			RunID:          *runID,
			AgentID:        domain.AgentID(*agentID),
			ProfileID:      *profileID,
			TaskID:         taskPtr,
			PeerAgentID:    peerPtr,
			ToolExecutable: *executable,
			ToolArgv:       argv,
		})
		if err != nil {
			printCommandError(stderr, "start-run", effectiveRequestID, err)
			return 1
		}
		printReceipt(stdout, "start-run", receipt)
		return 0
	})
}

func runStopRun(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("stop-run", flag.ContinueOnError)
	fs.SetOutput(stderr)
	wf := addWorkspaceFlags(fs)
	caller := fs.String("caller", "", "agent ID invoking this command")
	requestID := fs.String("request-id", "", "idempotency key for this command")
	reason := fs.String("reason", "stopped", "reason for termination")
	if err := fs.Parse(args); err != nil {
		return 1
	}
	if fs.NArg() != 1 {
		_, _ = fmt.Fprintln(stderr, "harnessing stop-run: exactly one runID argument is required")
		return 1
	}
	runID := fs.Arg(0)
	effectiveCaller := *caller
	if effectiveCaller == "" {
		effectiveCaller = ""
	}
	effectiveRequestID := *requestID
	if effectiveRequestID == "" {
		id, err := idsource.Random{}.NewID(context.Background())
		if err != nil {
			_, _ = fmt.Fprintln(stderr, err)
			return 1
		}
		effectiveRequestID = id
	}
	return withSession(stderr, wf, domain.AgentID(effectiveCaller), "stop-run", func(ctx context.Context, session api.FrontendSession) int {
		receipt, err := session.StopRun(ctx, api.StopRunRequest{RequestID: domain.RequestID(effectiveRequestID), RunID: runID, Reason: *reason})
		if err != nil {
			printCommandError(stderr, "stop-run", effectiveRequestID, err)
			return 1
		}
		printReceipt(stdout, "stop-run", receipt)
		return 0
	})
}

// runRunQuery implements `harnessing run`: a read-only view of one run,
// through FrontendSession.GetRun — the observation R2/D2 require be
// reachable through the product surface, not just an internal state
// dump.
func runRunQuery(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("run", flag.ContinueOnError)
	fs.SetOutput(stderr)
	wf := addWorkspaceFlags(fs)

	if err := fs.Parse(args); err != nil {
		return 1
	}
	if fs.NArg() != 1 {
		_, _ = fmt.Fprintln(stderr, "harnessing run: exactly one runID argument is required")
		return 1
	}
	runID := domain.RunID(fs.Arg(0))

	return withSession(stderr, wf, "", "run", func(ctx context.Context, session api.FrontendSession) int {
		run, err := session.GetRun(ctx, runID)
		if err != nil {
			_, _ = fmt.Fprintln(stderr, "harnessing run: "+describeError(err))
			return 1
		}
		printRun(stdout, run)
		return 0
	})
}

func printRun(w io.Writer, r domain.Run) {
	_, _ = fmt.Fprintf(w, "Run %s\n", r.ID)
	_, _ = fmt.Fprintf(w, "  Agent:      %s\n", r.AgentID)
	_, _ = fmt.Fprintf(w, "  ProfileID:  %s\n", r.ProfileID)
	_, _ = fmt.Fprintf(w, "  State:      %s\n", r.State)
	if r.ExitReason != "" {
		_, _ = fmt.Fprintf(w, "  ExitReason: %s\n", r.ExitReason)
	}
}

// runOperationQuery implements `harnessing operation`: boundaries.md
// line 40's read path for a StartRun/StopRun receipt's OperationID —
// progress and terminal outcome, separate from the Run's own state.
func runOperationQuery(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("operation", flag.ContinueOnError)
	fs.SetOutput(stderr)
	wf := addWorkspaceFlags(fs)

	if err := fs.Parse(args); err != nil {
		return 1
	}
	if fs.NArg() != 1 {
		_, _ = fmt.Fprintln(stderr, "harnessing operation: exactly one operationID argument is required")
		return 1
	}
	operationID := domain.OperationID(fs.Arg(0))

	return withSession(stderr, wf, "", "operation", func(ctx context.Context, session api.FrontendSession) int {
		op, err := session.GetOperation(ctx, operationID)
		if err != nil {
			_, _ = fmt.Fprintln(stderr, "harnessing operation: "+describeError(err))
			return 1
		}
		printOperation(stdout, op)
		return 0
	})
}

func printOperation(w io.Writer, op domain.Operation) {
	_, _ = fmt.Fprintf(w, "Operation %s\n", op.ID)
	_, _ = fmt.Fprintf(w, "  RunID:   %s\n", op.RunID)
	_, _ = fmt.Fprintf(w, "  Kind:    %s\n", op.Kind)
	_, _ = fmt.Fprintf(w, "  State:   %s\n", op.State)
	if op.Outcome != "" {
		_, _ = fmt.Fprintf(w, "  Outcome: %s\n", op.Outcome)
	}
}
