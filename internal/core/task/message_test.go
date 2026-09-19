package task_test

import (
	"context"
	"testing"

	"github.com/rafaelcalves/harnessing-101/internal/core/domain"
	"github.com/rafaelcalves/harnessing-101/internal/core/task"
)

func TestSendMessageAndRecipientAcknowledgement(t *testing.T) {
	ctx := context.Background()
	e, closeStore := newEngine(t, t.TempDir())
	defer closeStore()

	_, err := e.SendMessage(ctx, caller(engineerID, false), task.SendMessageRequest{
		RequestID: "send-1", MessageID: "message-1", SenderAgentID: engineerID,
		RecipientAgentID: reviewerID, Kind: domain.MessageRequest, Body: "please review",
	})
	if err != nil {
		t.Fatalf("SendMessage: %v", err)
	}

	if _, err := e.RecordMessagePublished(ctx, task.MessageDeliveryRequest{RequestID: "publish-1", MessageID: "message-1"}); err != nil {
		t.Fatalf("RecordMessagePublished: %v", err)
	}
	message, err := e.GetMessage(ctx, "message-1")
	if err != nil {
		t.Fatalf("GetMessage: %v", err)
	}
	if message.QueuedAt == nil || message.PublishedAt == nil || message.ProcessedAt != nil || message.AcknowledgedAt != nil {
		t.Fatalf("delivery facts collapsed or inferred: %+v", message)
	}
	if _, err := e.RecordMessageProcessed(ctx, task.MessageDeliveryRequest{RequestID: "process-1", MessageID: "message-1"}); err != nil {
		t.Fatalf("RecordMessageProcessed: %v", err)
	}
	message, err = e.GetMessage(ctx, "message-1")
	if err != nil {
		t.Fatalf("GetMessage after processing: %v", err)
	}
	if message.QueuedAt == nil || message.PublishedAt == nil || message.ProcessedAt == nil || message.AcknowledgedAt != nil {
		t.Fatalf("processed and acknowledged facts were conflated: %+v", message)
	}
	if message.Provenance.ClaimedAgentID != engineerID || message.Provenance.IdentityVerification != domain.IdentityUnverified {
		t.Fatalf("unexpected provenance: %+v", message.Provenance)
	}

	_, err = e.AcknowledgeMessage(ctx, caller(engineerID, false), task.AcknowledgeMessageRequest{
		RequestID: "ack-sender", MessageID: "message-1",
	})
	mustErrorCode(t, err, domain.ErrDenied)

	_, err = e.AcknowledgeMessage(ctx, caller(reviewerID, false), task.AcknowledgeMessageRequest{
		RequestID: "ack-recipient", MessageID: "message-1",
	})
	if err != nil {
		t.Fatalf("recipient AcknowledgeMessage: %v", err)
	}
	message, err = e.GetMessage(ctx, "message-1")
	if err != nil {
		t.Fatalf("GetMessage after ack: %v", err)
	}
	if message.AcknowledgedBy != reviewerID || message.AcknowledgedAt == nil {
		t.Fatalf("acknowledgement not recorded: %+v", message)
	}
	ackAt := *message.AcknowledgedAt

	_, err = e.AcknowledgeMessage(ctx, caller(reviewerID, false), task.AcknowledgeMessageRequest{
		RequestID: "ack-recipient-replay", MessageID: "message-1",
	})
	if err != nil {
		t.Fatalf("duplicate acknowledgement: %v", err)
	}
	message, err = e.GetMessage(ctx, "message-1")
	if err != nil {
		t.Fatalf("GetMessage after duplicate ack: %v", err)
	}
	if !message.AcknowledgedAt.Equal(ackAt) {
		t.Fatalf("duplicate acknowledgement changed timestamp: got %v want %v", *message.AcknowledgedAt, ackAt)
	}
}

func TestAcknowledgedMessageSurvivesRestart(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()

	func() {
		e, closeStore := newEngine(t, dir)
		defer closeStore()
		if _, err := e.SendMessage(ctx, caller(engineerID, false), task.SendMessageRequest{
			RequestID: "send-restart", MessageID: "message-restart", SenderAgentID: engineerID,
			RecipientAgentID: reviewerID, Kind: domain.MessageInform, Body: "persist this",
		}); err != nil {
			t.Fatalf("SendMessage: %v", err)
		}
		if _, err := e.AcknowledgeMessage(ctx, caller(reviewerID, false), task.AcknowledgeMessageRequest{
			RequestID: "ack-restart", MessageID: "message-restart",
		}); err != nil {
			t.Fatalf("AcknowledgeMessage: %v", err)
		}
	}()

	e, closeStore := newEngine(t, dir)
	defer closeStore()
	message, err := e.GetMessage(ctx, "message-restart")
	if err != nil {
		t.Fatalf("GetMessage after restart: %v", err)
	}
	if message.AcknowledgedBy != reviewerID || message.AcknowledgedAt == nil || message.QueuedAt == nil {
		t.Fatalf("message facts lost across restart: %+v", message)
	}
}

func TestSendMessageUnknownReferencesNotFound(t *testing.T) {
	ctx := context.Background()
	e, closeStore := newEngine(t, t.TempDir())
	defer closeStore()

	unknownTask := domain.TaskID("missing-task")
	_, err := e.SendMessage(ctx, caller(engineerID, false), task.SendMessageRequest{
		RequestID: "send-missing-task", MessageID: "message-missing-task", SenderAgentID: engineerID,
		RecipientAgentID: reviewerID, Kind: domain.MessageResult, Body: "result", TaskID: &unknownTask,
	})
	mustErrorCode(t, err, domain.ErrNotFound)

	unknownMessage := domain.MessageID("missing-message")
	_, err = e.SendMessage(ctx, caller(engineerID, false), task.SendMessageRequest{
		RequestID: "send-missing-reply", MessageID: "message-missing-reply", SenderAgentID: engineerID,
		RecipientAgentID: reviewerID, Kind: domain.MessageInform, Body: "reply", ReplyToMessageID: &unknownMessage,
	})
	mustErrorCode(t, err, domain.ErrNotFound)
}
