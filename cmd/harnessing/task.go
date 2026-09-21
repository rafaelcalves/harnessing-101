package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"time"

	"github.com/rafaelcalves/harnessing-101/internal/api"
	"github.com/rafaelcalves/harnessing-101/internal/core/domain"
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

	return withSession(stderr, wf, "", "task", func(ctx context.Context, session api.FrontendSession) int {
		snapshot, err := session.GetSnapshot(ctx)
		if err != nil {
			_, _ = fmt.Fprintln(stderr, "harnessing task: "+describeError(err))
			return 1
		}
		var task *domain.Task
		for i := range snapshot.Tasks {
			if snapshot.Tasks[i].ID == taskID {
				task = &snapshot.Tasks[i]
				break
			}
		}
		if task == nil {
			_, _ = fmt.Fprintln(stderr, "harnessing task: "+describeError(&domain.Error{Code: domain.ErrNotFound, Detail: "task not found"}))
			return 1
		}
		printTask(stdout, *task)
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
	if t.LastStatusChange != nil {
		_, _ = fmt.Fprintf(w, "  Reporter:   %s (claimed reporter, %s)\n", t.LastStatusChange.Provenance.ClaimedAgentID, t.LastStatusChange.Provenance.IdentityVerification)
		_, _ = fmt.Fprintf(w, "  Last update: %s\n", t.LastStatusChange.Provenance.RecordedAt.Format(time.RFC3339Nano))
	} else {
		_, _ = fmt.Fprintln(w, "  Reporter:   (unavailable; status-update provenance is not recorded)")
		_, _ = fmt.Fprintln(w, "  Last update: (unavailable; status-update timestamp is not recorded)")
	}
}
