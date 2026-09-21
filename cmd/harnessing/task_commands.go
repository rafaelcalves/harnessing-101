package main

import (
	"context"
	"flag"
	"fmt"
	"io"

	"github.com/rafaelcalves/harnessing-101/internal/api"
	"github.com/rafaelcalves/harnessing-101/internal/core/domain"
)

func runCreate(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("create", flag.ContinueOnError)
	fs.SetOutput(stderr)
	wf := addWorkspaceFlags(fs)
	caller := fs.String("caller", "", "agent ID invoking this command (required)")
	requestID := fs.String("request-id", "", "idempotency key for this command (required)")
	taskID := fs.String("task", "", "task ID to create (required)")
	title := fs.String("title", "", "task title (required)")
	assignee := fs.String("assignee", "", "optional assignee agent ID")

	if err := fs.Parse(args); err != nil {
		return 1
	}
	if missing := requireFlags(
		flagValue{"-caller", *caller}, flagValue{"-request-id", *requestID},
		flagValue{"-task", *taskID}, flagValue{"-title", *title},
	); missing != "" {
		_, _ = fmt.Fprintf(stderr, "harnessing create: %s is required\n", missing)
		return 1
	}

	return withSession(stderr, wf, domain.AgentID(*caller), "create", func(ctx context.Context, session api.FrontendSession) int {
		receipt, err := session.CreateTask(ctx, api.CreateTaskRequest{
			RequestID:  domain.RequestID(*requestID),
			TaskID:     domain.TaskID(*taskID),
			Title:      *title,
			AssigneeID: domain.AgentID(*assignee),
		})
		if err != nil {
			printCommandError(stderr, "create", *requestID, err)
			return 1
		}
		printReceipt(stdout, "create", receipt)
		return 0
	})
}

func runTransition(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("transition", flag.ContinueOnError)
	fs.SetOutput(stderr)
	wf := addWorkspaceFlags(fs)
	caller := fs.String("caller", "", "agent ID invoking this command (required)")
	requestID := fs.String("request-id", "", "idempotency key for this command (required)")
	taskID := fs.String("task", "", "task ID to transition (required)")
	from := fs.String("from", "", "current status (required)")
	to := fs.String("to", "", "target status (required)")
	reason := fs.String("reason", "", "reason (required to move to Blocked or to reopen Done)")

	if err := fs.Parse(args); err != nil {
		return 1
	}
	if missing := requireFlags(
		flagValue{"-caller", *caller}, flagValue{"-request-id", *requestID},
		flagValue{"-task", *taskID}, flagValue{"-from", *from}, flagValue{"-to", *to},
	); missing != "" {
		_, _ = fmt.Fprintf(stderr, "harnessing transition: %s is required\n", missing)
		return 1
	}

	return withSession(stderr, wf, domain.AgentID(*caller), "transition", func(ctx context.Context, session api.FrontendSession) int {
		receipt, err := session.TransitionTask(ctx, api.TransitionTaskRequest{
			RequestID:  domain.RequestID(*requestID),
			TaskID:     domain.TaskID(*taskID),
			FromStatus: domain.TaskStatus(*from),
			ToStatus:   domain.TaskStatus(*to),
			Reason:     *reason,
		})
		if err != nil {
			printCommandError(stderr, "transition", *requestID, err)
			return 1
		}
		printReceipt(stdout, "transition", receipt)
		return 0
	})
}

func runReport(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("report", flag.ContinueOnError)
	fs.SetOutput(stderr)
	wf := addWorkspaceFlags(fs)
	caller := fs.String("caller", "", "agent ID invoking this command; must be the task's current assignee (required)")
	requestID := fs.String("request-id", "", "idempotency key for this command (required)")
	taskID := fs.String("task", "", "task ID (required)")
	resultID := fs.String("result", "", "new result ID (required)")
	expectedRevision := fs.Uint64("expected-revision", 0, "task revision this report is based on (required)")
	summary := fs.String("summary", "", "result summary (required)")
	var artifacts stringList
	fs.Var(&artifacts, "artifact", "workspace-relative artifact reference (repeatable; at least one required)")

	if err := fs.Parse(args); err != nil {
		return 1
	}
	if missing := requireFlags(
		flagValue{"-caller", *caller}, flagValue{"-request-id", *requestID}, flagValue{"-task", *taskID},
		flagValue{"-result", *resultID}, flagValue{"-summary", *summary},
	); missing != "" {
		_, _ = fmt.Fprintf(stderr, "harnessing report: %s is required\n", missing)
		return 1
	}

	return withSession(stderr, wf, domain.AgentID(*caller), "report", func(ctx context.Context, session api.FrontendSession) int {
		receipt, err := session.ReportTaskResult(ctx, api.ReportTaskResultRequest{
			RequestID:            domain.RequestID(*requestID),
			TaskID:               domain.TaskID(*taskID),
			ResultID:             domain.ResultID(*resultID),
			ExpectedTaskRevision: *expectedRevision,
			Summary:              *summary,
			Artifacts:            artifacts,
		})
		if err != nil {
			printCommandError(stderr, "report", *requestID, err)
			return 1
		}
		printReceipt(stdout, "report", receipt)
		return 0
	})
}

func runAccept(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("accept", flag.ContinueOnError)
	fs.SetOutput(stderr)
	wf := addWorkspaceFlags(fs)
	caller := fs.String("caller", "", "agent ID invoking this command; must be in -reviewer to succeed (required)")
	requestID := fs.String("request-id", "", "idempotency key for this command (required)")
	taskID := fs.String("task", "", "task ID (required)")
	resultID := fs.String("result", "", "result ID being accepted (required)")
	expectedRevision := fs.Uint64("expected-revision", 0, "task revision this decision is based on (required)")
	reviewNote := fs.String("review-note", "", "optional review note")

	if err := fs.Parse(args); err != nil {
		return 1
	}
	if missing := requireFlags(
		flagValue{"-caller", *caller}, flagValue{"-request-id", *requestID},
		flagValue{"-task", *taskID}, flagValue{"-result", *resultID},
	); missing != "" {
		_, _ = fmt.Fprintf(stderr, "harnessing accept: %s is required\n", missing)
		return 1
	}

	return withSession(stderr, wf, domain.AgentID(*caller), "accept", func(ctx context.Context, session api.FrontendSession) int {
		receipt, err := session.AcceptTaskResult(ctx, api.AcceptTaskResultRequest{
			RequestID:            domain.RequestID(*requestID),
			TaskID:               domain.TaskID(*taskID),
			ResultID:             domain.ResultID(*resultID),
			ExpectedTaskRevision: *expectedRevision,
			ReviewNote:           *reviewNote,
		})
		if err != nil {
			printCommandError(stderr, "accept", *requestID, err)
			return 1
		}
		printReceipt(stdout, "accept", receipt)
		return 0
	})
}

func runReject(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("reject", flag.ContinueOnError)
	fs.SetOutput(stderr)
	wf := addWorkspaceFlags(fs)
	caller := fs.String("caller", "", "agent ID invoking this command; must be in -reviewer to succeed (required)")
	requestID := fs.String("request-id", "", "idempotency key for this command (required)")
	taskID := fs.String("task", "", "task ID (required)")
	resultID := fs.String("result", "", "result ID being rejected (required)")
	expectedRevision := fs.Uint64("expected-revision", 0, "task revision this decision is based on (required)")
	reason := fs.String("reason", "", "reason for rejection (required)")

	if err := fs.Parse(args); err != nil {
		return 1
	}
	if missing := requireFlags(
		flagValue{"-caller", *caller}, flagValue{"-request-id", *requestID}, flagValue{"-task", *taskID},
		flagValue{"-result", *resultID}, flagValue{"-reason", *reason},
	); missing != "" {
		_, _ = fmt.Fprintf(stderr, "harnessing reject: %s is required\n", missing)
		return 1
	}

	return withSession(stderr, wf, domain.AgentID(*caller), "reject", func(ctx context.Context, session api.FrontendSession) int {
		receipt, err := session.RejectTaskResult(ctx, api.RejectTaskResultRequest{
			RequestID:            domain.RequestID(*requestID),
			TaskID:               domain.TaskID(*taskID),
			ResultID:             domain.ResultID(*resultID),
			ExpectedTaskRevision: *expectedRevision,
			Reason:               *reason,
		})
		if err != nil {
			printCommandError(stderr, "reject", *requestID, err)
			return 1
		}
		printReceipt(stdout, "reject", receipt)
		return 0
	})
}
