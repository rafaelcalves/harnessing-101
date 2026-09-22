package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"time"

	"github.com/rafaelcalves/harnessing-101/internal/api"
)

// runEventsQuery implements `harnessing events`: H101-193's shipped
// state-event read path — the command item 5's I6 negative reads
// instead of run-output, so the separation between domain StateEvents
// and raw run output (UI-08) is proven against the actual product
// surface, not by inspecting engine internals. It is a single bounded
// read (see api.FrontendSession.Subscribe / transport.Client.Subscribe
// for why), not a live follow — sufficient for a negative check;
// continuous delivery is not this command's claim.
func runEventsQuery(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("events", flag.ContinueOnError)
	fs.SetOutput(stderr)
	wf := addWorkspaceFlags(fs)
	afterCursor := fs.String("after-cursor", "", "resume after this cursor (empty replays from the retained floor)")
	if err := fs.Parse(args); err != nil {
		return 1
	}
	return withSession(stderr, wf, "", "events", func(ctx context.Context, session api.FrontendSession) int {
		// Bounded regardless of session kind: an attached transport
		// session already only ever answers one batch (its own
		// server-side drain window), but a direct, one-shot
		// host.Open session's Subscribe is the real live event bus —
		// without this deadline it would never close its channel and
		// this command would hang forever.
		subCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
		defer cancel()
		ch, err := session.Subscribe(subCtx, *afterCursor)
		if err != nil {
			_, _ = fmt.Fprintln(stderr, "harnessing events: "+describeError(err))
			return 1
		}
		for ev := range ch {
			_, _ = fmt.Fprintf(stdout, "Event revision=%d kind=%s subjects=%v payload=%v\n",
				ev.WorkspaceRevision, ev.Kind, ev.SubjectIDs, ev.Payload)
		}
		return 0
	})
}
