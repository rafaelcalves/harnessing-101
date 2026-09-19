package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"

	"github.com/rafaelcalves/harnessing-101/internal/core/domain"
	"github.com/rafaelcalves/harnessing-101/internal/host"
)

// agentIDList collects a repeatable -reviewer flag into an ordered list
// of domain.AgentID, so the operator can name zero or more reviewers on
// one command line.
type agentIDList []domain.AgentID

func (l *agentIDList) String() string {
	return fmt.Sprint(*l)
}

func (l *agentIDList) Set(value string) error {
	if value == "" {
		return fmt.Errorf("-reviewer must not be empty")
	}
	*l = append(*l, domain.AgentID(value))
	return nil
}

// runTask implements `harnessing task`: a read-only view of one task,
// through host.Capabilities.GetTask only. It opens the workspace with
// the operator-supplied root, workspace ID, and reviewer set — none of
// which come from anything already written into the workspace — and
// always closes it before returning, on every exit path.
func runTask(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("task", flag.ContinueOnError)
	fs.SetOutput(stderr)
	workspaceRoot := fs.String("workspace", "", "path to the workspace root (required)")
	workspaceID := fs.String("workspace-id", "default", "workspace identifier stored alongside state")
	var reviewers agentIDList
	fs.Var(&reviewers, "reviewer", "agent ID with human-review authority (repeatable)")

	if err := fs.Parse(args); err != nil {
		// flag already printed its own message to stderr.
		return 1
	}
	if *workspaceRoot == "" {
		_, _ = fmt.Fprintln(stderr, "harnessing task: -workspace is required")
		return 1
	}
	if fs.NArg() != 1 {
		_, _ = fmt.Fprintln(stderr, "harnessing task: exactly one taskID argument is required")
		return 1
	}
	taskID := domain.TaskID(fs.Arg(0))

	caps, err := host.Open(*workspaceRoot, domain.WorkspaceID(*workspaceID), reviewers)
	if err != nil {
		_, _ = fmt.Fprintln(stderr, "harnessing task: "+describeError(err))
		return 1
	}
	closeFailed := false
	defer func() {
		if closeErr := caps.Close(); closeErr != nil {
			_, _ = fmt.Fprintln(stderr, "harnessing task: workspace did not close cleanly: "+describeError(closeErr))
			closeFailed = true
		}
	}()

	t, err := caps.GetTask(context.Background(), taskID)
	if err != nil {
		_, _ = fmt.Fprintln(stderr, "harnessing task: "+describeError(err))
		return 1
	}

	printTask(stdout, t)
	if closeFailed {
		return 1
	}
	return 0
}

func printTask(w io.Writer, t domain.Task) {
	_, _ = fmt.Fprintf(w, "Task %s\n", t.ID)
	_, _ = fmt.Fprintf(w, "  Title:      %s\n", t.Title)
	_, _ = fmt.Fprintf(w, "  Status:     %s\n", t.Status)
	_, _ = fmt.Fprintf(w, "  Assignee:   %s\n", t.AssigneeID)
	_, _ = fmt.Fprintf(w, "  Revision:   %d\n", t.Revision)
	if t.CurrentResultID != nil {
		_, _ = fmt.Fprintf(w, "  ResultID:   %s\n", *t.CurrentResultID)
	} else {
		_, _ = fmt.Fprintln(w, "  ResultID:   (none)")
	}
	_, _ = fmt.Fprintf(w, "  Created by: %s (claimed sender, %s, %s)\n",
		t.Provenance.ClaimedAgentID, t.Provenance.EntryMechanism, t.Provenance.IdentityVerification)
}

// describeError renders a domain.Error with its stable code, so a script
// scraping stderr sees "NotFound"/"Denied"/etc. rather than a bare
// human-readable sentence it would have to parse.
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
