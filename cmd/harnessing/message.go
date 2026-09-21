package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"time"

	"github.com/rafaelcalves/harnessing-101/internal/core/domain"
	"github.com/rafaelcalves/harnessing-101/internal/host"
)

// runMessage implements `harnessing message`: a read-only view of one
// durable handoff. Delivery facts stay separate so an absent publication or
// processing observation cannot be mistaken for a completed step.
func runMessage(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("message", flag.ContinueOnError)
	fs.SetOutput(stderr)
	wf := addWorkspaceFlags(fs)

	if err := fs.Parse(args); err != nil {
		return 1
	}
	if fs.NArg() != 1 {
		_, _ = fmt.Fprintln(stderr, "harnessing message: exactly one messageID argument is required")
		return 1
	}
	messageID := domain.MessageID(fs.Arg(0))

	return withCapabilities(stderr, wf, "message", func(ctx context.Context, caps host.Capabilities) int {
		message, err := caps.GetMessage(ctx, messageID)
		if err != nil {
			_, _ = fmt.Fprintln(stderr, "harnessing message: "+describeError(err))
			return 1
		}
		printMessage(stdout, message)
		return 0
	})
}

func printMessage(w io.Writer, message domain.Message) {
	_, _ = fmt.Fprintf(w, "Message %s\n", message.MessageID)
	_, _ = fmt.Fprintf(w, "  Sender:      %s (claimed routing claim; identity unverified)\n", message.SenderAgentID)
	_, _ = fmt.Fprintf(w, "  Recipient:   %s\n", message.RecipientAgentID)
	_, _ = fmt.Fprintf(w, "  Kind:        %s\n", message.Kind)
	_, _ = fmt.Fprintf(w, "  Body:        %s\n", message.Body)
	if message.TaskID != nil {
		_, _ = fmt.Fprintf(w, "  Task:        %s\n", *message.TaskID)
	} else {
		_, _ = fmt.Fprintln(w, "  Task:        (absent)")
	}
	_, _ = fmt.Fprintf(w, "  Queued:      %s\n", messageTime(message.QueuedAt))
	_, _ = fmt.Fprintf(w, "  Published:   %s\n", messageTime(message.PublishedAt))
	_, _ = fmt.Fprintf(w, "  Processed:   %s\n", messageTime(message.ProcessedAt))
	if message.AcknowledgedAt == nil {
		_, _ = fmt.Fprintln(w, "  Acknowledged: (absent)")
	} else {
		_, _ = fmt.Fprintf(w, "  Acknowledged: %s by %s\n", messageTime(message.AcknowledgedAt), message.AcknowledgedBy)
	}
	_, _ = fmt.Fprintf(w, "  Recorded by: %s (claimed sender, %s, %s)\n",
		message.Provenance.ClaimedAgentID, message.Provenance.EntryMechanism, message.Provenance.IdentityVerification)
}

func messageTime(value *time.Time) string {
	if value == nil {
		return "(absent)"
	}
	return value.Format(time.RFC3339Nano)
}
