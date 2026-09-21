// Package assembly is the trusted CLI composition root. It is deliberately
// small: it opens the workspace, fixes reviewer and caller policy, binds a
// FrontendSession, and owns Close. Presentation code receives only api.
package assembly

import (
	"context"
	"errors"
	"fmt"
	"io"
	"path/filepath"

	"github.com/rafaelcalves/harnessing-101/internal/adapters/mailbox"
	"github.com/rafaelcalves/harnessing-101/internal/api"
	"github.com/rafaelcalves/harnessing-101/internal/core/domain"
	"github.com/rafaelcalves/harnessing-101/internal/host"
)

// mailboxSubdir is where this workspace's file-protocol mailbox lives,
// relative to the same root WithSession is given — one mailbox tree per
// workspace root, sibling to state.json, matching FileStore's own
// single-root convention rather than inventing a second configured path.
const mailboxSubdir = "mailbox"

func WithSession(stderr io.Writer, root string, workspaceID domain.WorkspaceID, reviewers []domain.AgentID, caller domain.AgentID, cmdName string, fn func(context.Context, api.FrontendSession) int) int {
	if root == "" {
		_, _ = fmt.Fprintf(stderr, "harnessing %s: -workspace is required\n", cmdName)
		return 1
	}
	caps, err := host.Open(root, workspaceID, reviewers)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "harnessing %s: %s\n", cmdName, describeError(err))
		return 1
	}
	ctx := context.Background()
	session := host.BindFrontendSession(caps, caller)
	code := fn(ctx, session)

	// H101-85: this is THE product-path wiring for the mailbox. It is
	// deliberately the ONLY place in this codebase that hands caps to a
	// mailbox.Deliverer — no other line in assembly, and nothing in
	// cmd/harnessing (which only ever holds api.FrontendSession, whose
	// method set has no RecordMessagePublished/RecordMessageProcessed at
	// all — see api.FrontendSession), calls those two methods or drives
	// file-protocol acknowledgement. That is what makes Deliverer the
	// SOLE writer in the actual product path rather than merely the
	// sole writer nothing currently contradicts: grep this repository
	// for RecordMessagePublished/RecordMessageProcessed outside this
	// file and internal/adapters/mailbox itself, and there is nothing
	// to find. A mailbox hiccup here is reported but does not turn this
	// command's own already-decided result into a failure — delivery
	// is background work that retries from persisted facts on the next
	// invocation (every step Deliverer takes is idempotent by message
	// or control-record ID), not part of what this command promised.
	if err := driveMailbox(ctx, root, caps, caller); err != nil {
		_, _ = fmt.Fprintf(stderr, "harnessing %s: mailbox delivery: %s\n", cmdName, describeError(err))
	}

	if closeErr := caps.Close(); closeErr != nil {
		_, _ = fmt.Fprintf(stderr, "harnessing %s: workspace did not close cleanly: %s\n", cmdName, describeError(closeErr))
		if code == 0 {
			return 1
		}
	}
	return code
}

// driveMailbox runs one pass of the file-protocol mailbox: publish
// whatever the engine has queued, process whatever this pass finds
// already published in a recipient's inbox, and ingest whatever
// acknowledgement control records are waiting in the CALLING agent's
// own acks directory. There is no background daemon in this product —
// every CLI invocation is the only chance any of this has to run, so
// each one pumps it once rather than leaving it for a process that
// does not exist.
//
// An empty caller (H101-122) means this invocation is an observer, not
// scoped to any one agent's identity — task, message, messages, and hold
// all call WithSession this way. There is no such agent's acks directory
// to ingest for "no one": skip IngestAcks rather than call it with an ID
// the mailbox adapter will always reject as empty. This is not a
// narrower error tolerance, it is recognizing the call has nothing to do.
func driveMailbox(ctx context.Context, root string, caps host.Capabilities, caller domain.AgentID) error {
	mail, err := mailbox.Open(filepath.Join(root, mailboxSubdir))
	if err != nil {
		return err
	}
	deliverer := mailbox.NewDeliverer(mail, caps)
	if err := deliverer.DeliverPending(ctx); err != nil {
		return err
	}
	if err := deliverer.IngestPending(ctx); err != nil {
		return err
	}
	if caller == "" {
		return nil
	}
	return deliverer.IngestAcks(ctx, caller)
}

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
