package main

import (
	"context"
	"flag"
	"fmt"
	"io"

	"github.com/rafaelcalves/harnessing-101/internal/core/domain"
	"github.com/rafaelcalves/harnessing-101/internal/host"
)

// runMessages lists the pending handoffs visible in a workspace. It uses the
// complete snapshot query so a recipient does not need to know a message ID
// before inspecting what is waiting for acknowledgement.
func runMessages(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("messages", flag.ContinueOnError)
	fs.SetOutput(stderr)
	wf := addWorkspaceFlags(fs)
	recipient := fs.String("recipient", "", "only show messages addressed to this agent ID")

	if err := fs.Parse(args); err != nil {
		return 1
	}
	return withCapabilities(stderr, wf, "messages", func(ctx context.Context, caps host.Capabilities) int {
		snapshot, err := caps.GetSnapshot(ctx)
		if err != nil {
			_, _ = fmt.Fprintln(stderr, "harnessing messages: "+describeError(err))
			return 1
		}
		pending := make([]domain.Message, 0, len(snapshot.Messages))
		for _, message := range snapshot.Messages {
			if message.AcknowledgedAt != nil || (*recipient != "" && string(message.RecipientAgentID) != *recipient) {
				continue
			}
			pending = append(pending, message)
		}
		_, _ = fmt.Fprintf(stdout, "Pending messages: %d\n", len(pending))
		for _, message := range pending {
			printMessage(stdout, message)
		}
		return 0
	})
}
