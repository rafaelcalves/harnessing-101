package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"
	"syscall"

	"github.com/rafaelcalves/harnessing-101/internal/adapters/idsource"
	"github.com/rafaelcalves/harnessing-101/internal/assembly"
	"github.com/rafaelcalves/harnessing-101/internal/core/domain"
)

// runServe implements `harnessing serve`: the continuing host of H101-136
// (Phase 3 exit item 6's minimal slice). It opens the workspace once and
// holds it until interrupted, servicing attach/detach and every
// api.FrontendSession call through the file transport instead of a
// single one-shot session — see internal/assembly.Serve and
// internal/adapters/transport for the mechanics.
func runServe(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("serve", flag.ContinueOnError)
	fs.SetOutput(stderr)
	wf := addWorkspaceFlags(fs)
	if err := fs.Parse(args); err != nil {
		return 1
	}

	generation, err := idsource.Random{}.NewID(context.Background())
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "harnessing serve: %s\n", err)
		return 1
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	code := assembly.ServeWithReady(ctx, stderr, *wf.root, domain.WorkspaceID(*wf.id), []domain.AgentID(wf.reviewers), generation, func() {
		_, _ = fmt.Fprintf(stdout, "harnessing serve: workspace open, listening (pid %d, generation %s)\n", os.Getpid(), generation)
	})
	_, _ = fmt.Fprintln(stdout, "harnessing serve: closed")
	return code
}
