package main

import (
	"context"
	"flag"
	"fmt"
	"io"

	"github.com/rafaelcalves/harnessing-101/internal/core/domain"
	"github.com/rafaelcalves/harnessing-101/internal/core/task"
	"github.com/rafaelcalves/harnessing-101/internal/host"
)

// runSend implements `harnessing send`. -sender defaults to -caller when
// omitted: the ordinary case is sending as yourself. Setting -sender to
// something else is allowed and sends unchanged — the engine does not
// check caller against sender for this command, and this file does not
// invent that check either; SenderAgentID is a routing claim, not an
// authenticated identity, on this path exactly as boundaries.md and the
// H101-24 review already documented.
func runSend(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("send", flag.ContinueOnError)
	fs.SetOutput(stderr)
	wf := addWorkspaceFlags(fs)
	caller := fs.String("caller", "", "agent ID invoking this command (required)")
	requestID := fs.String("request-id", "", "idempotency key for this command (required)")
	messageID := fs.String("message", "", "message ID (required)")
	var sender optionalString
	fs.Var(&sender, "sender", "claimed sender agent ID (defaults to -caller)")
	recipient := fs.String("recipient", "", "recipient agent ID (required)")
	kind := fs.String("kind", "", "Request, Inform, or Result (required)")
	body := fs.String("body", "", "message body (required)")
	var taskID, replyTo optionalString
	fs.Var(&taskID, "task", "optional related task ID")
	fs.Var(&replyTo, "reply-to", "optional message ID this replies to")

	if err := fs.Parse(args); err != nil {
		return 1
	}
	if missing := requireFlags(
		flagValue{"-caller", *caller}, flagValue{"-request-id", *requestID}, flagValue{"-message", *messageID},
		flagValue{"-recipient", *recipient}, flagValue{"-kind", *kind}, flagValue{"-body", *body},
	); missing != "" {
		_, _ = fmt.Fprintf(stderr, "harnessing send: %s is required\n", missing)
		return 1
	}

	senderID := *caller
	if s := sender.Get(); s != nil {
		senderID = *s
	}

	return withCapabilities(stderr, wf, "send", func(ctx context.Context, caps host.Capabilities) int {
		receipt, err := caps.SendMessage(ctx, domain.AgentID(*caller), task.SendMessageRequest{
			RequestID:        domain.RequestID(*requestID),
			MessageID:        domain.MessageID(*messageID),
			SenderAgentID:    domain.AgentID(senderID),
			RecipientAgentID: domain.AgentID(*recipient),
			Kind:             domain.MessageKind(*kind),
			Body:             *body,
			TaskID:           optionalTaskID(taskID.Get()),
			ReplyToMessageID: optionalMessageID(replyTo.Get()),
		})
		if err != nil {
			_, _ = fmt.Fprintln(stderr, "harnessing send: "+describeError(err))
			return 1
		}
		printReceipt(stdout, "send", receipt)
		return 0
	})
}

func runAck(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("ack", flag.ContinueOnError)
	fs.SetOutput(stderr)
	wf := addWorkspaceFlags(fs)
	caller := fs.String("caller", "", "agent ID invoking this command; must be the message's recipient (required)")
	requestID := fs.String("request-id", "", "idempotency key for this command (required)")
	messageID := fs.String("message", "", "message ID to acknowledge (required)")

	if err := fs.Parse(args); err != nil {
		return 1
	}
	if missing := requireFlags(
		flagValue{"-caller", *caller}, flagValue{"-request-id", *requestID}, flagValue{"-message", *messageID},
	); missing != "" {
		_, _ = fmt.Fprintf(stderr, "harnessing ack: %s is required\n", missing)
		return 1
	}

	return withCapabilities(stderr, wf, "ack", func(ctx context.Context, caps host.Capabilities) int {
		receipt, err := caps.AcknowledgeMessage(ctx, domain.AgentID(*caller), task.AcknowledgeMessageRequest{
			RequestID: domain.RequestID(*requestID),
			MessageID: domain.MessageID(*messageID),
		})
		if err != nil {
			_, _ = fmt.Fprintln(stderr, "harnessing ack: "+describeError(err))
			return 1
		}
		printReceipt(stdout, "ack", receipt)
		return 0
	})
}

func optionalTaskID(s *string) *domain.TaskID {
	if s == nil {
		return nil
	}
	id := domain.TaskID(*s)
	return &id
}

func optionalMessageID(s *string) *domain.MessageID {
	if s == nil {
		return nil
	}
	id := domain.MessageID(*s)
	return &id
}
