package main

import (
	"context"
	"flag"
	"fmt"
	"io"

	"github.com/rafaelcalves/harnessing-101/internal/core/domain"
	"github.com/rafaelcalves/harnessing-101/internal/host"
)

// runTask implements `harnessing task`: a read-only view of one task,
// through host.Capabilities.GetTask only.
func runTask(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("task", flag.ContinueOnError)
	fs.SetOutput(stderr)
	wf := addWorkspaceFlags(fs)

	if err := fs.Parse(args); err != nil {
		return 1
	}
	if fs.NArg() != 1 {
		_, _ = fmt.Fprintln(stderr, "harnessing task: exactly one taskID argument is required")
		return 1
	}
	taskID := domain.TaskID(fs.Arg(0))

	return withCapabilities(stderr, wf, "task", func(ctx context.Context, caps host.Capabilities) int {
		t, err := caps.GetTask(ctx, taskID)
		if err != nil {
			_, _ = fmt.Fprintln(stderr, "harnessing task: "+describeError(err))
			return 1
		}
		printTask(stdout, t)
		return 0
	})
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
