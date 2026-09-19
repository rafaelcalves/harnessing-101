package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"
	"syscall"

	"github.com/rafaelcalves/harnessing-101/internal/core/domain"
	"github.com/rafaelcalves/harnessing-101/internal/host"
)

// runHold implements `harnessing hold`: opens the workspace and keeps it
// open — holding the exclusive lock — until interrupted, then closes it
// and exits 0.
//
// Why this command exists (point 3 of H101-51): every other command
// opens, does one thing, and closes within the same process invocation,
// so there is never a live `harnessing` process to test crash/kill
// behaviour against — Claudio's finding, and the reason exit item 2's
// cycle-kill-reopen test was blocked. `hold` is a real, minimal, honest
// answer to "what does a long-running invocation look like here": a
// user (or a test) starts it, it holds the workspace open the way any
// long session would, and killing it — SIGINT/SIGTERM for a graceful
// stop, SIGKILL to simulate a crash — is exactly the scenario H101-20's
// flock-based recovery exists to handle. It prints one line once the
// lock is held, so a script has an unambiguous synchronization point
// instead of guessing with a sleep.
func runHold(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("hold", flag.ContinueOnError)
	fs.SetOutput(stderr)
	wf := addWorkspaceFlags(fs)

	if err := fs.Parse(args); err != nil {
		return 1
	}
	if *wf.root == "" {
		_, _ = fmt.Fprintln(stderr, "harnessing hold: -workspace is required")
		return 1
	}

	caps, err := host.Open(*wf.root, domain.WorkspaceID(*wf.id), wf.reviewers)
	if err != nil {
		_, _ = fmt.Fprintln(stderr, "harnessing hold: "+describeError(err))
		return 1
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	_, _ = fmt.Fprintf(stdout, "harnessing hold: workspace open, holding lock (pid %d)\n", os.Getpid())
	<-ctx.Done()

	if closeErr := caps.Close(); closeErr != nil {
		_, _ = fmt.Fprintln(stderr, "harnessing hold: workspace did not close cleanly: "+describeError(closeErr))
		return 1
	}
	_, _ = fmt.Fprintln(stdout, "harnessing hold: closed")
	return 0
}
