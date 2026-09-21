package mailbox_test

import (
	"context"
	"testing"
	"time"

	"github.com/rafaelcalves/harnessing-101/internal/adapters/mailbox"
	"github.com/rafaelcalves/harnessing-101/internal/api"
	"github.com/rafaelcalves/harnessing-101/internal/core/domain"
)

// fakeRecorder is a minimal, in-memory stand-in for host.Capabilities'
// relevant slice — enough to drive Deliverer's causal-order logic
// without needing a real engine/store. It intentionally has NO
// authority checks of its own; that is the engine's job, exercised
// separately in internal/core/task's own tests. This fake's purpose is
// to prove what the Deliverer itself does and does not call, and in
// what order.
type fakeRecorder struct {
	messages  []domain.Message
	published []domain.MessageID
	processed []domain.MessageID
	acked     []struct {
		caller domain.AgentID
		msgID  domain.MessageID
	}
	denyAckFor domain.MessageID
}

func (f *fakeRecorder) GetSnapshot(ctx context.Context) (domain.Snapshot, error) {
	return domain.Snapshot{Messages: append([]domain.Message{}, f.messages...)}, nil
}

func (f *fakeRecorder) RecordMessagePublished(ctx context.Context, req api.MessageDeliveryRequest) (domain.Receipt, error) {
	for i := range f.messages {
		if f.messages[i].MessageID == req.MessageID {
			now := time.Now()
			f.messages[i].PublishedAt = &now
		}
	}
	f.published = append(f.published, req.MessageID)
	return domain.Receipt{RequestID: req.RequestID}, nil
}

func (f *fakeRecorder) RecordMessageProcessed(ctx context.Context, req api.MessageDeliveryRequest) (domain.Receipt, error) {
	for i := range f.messages {
		if f.messages[i].MessageID == req.MessageID {
			now := time.Now()
			f.messages[i].ProcessedAt = &now
		}
	}
	f.processed = append(f.processed, req.MessageID)
	return domain.Receipt{RequestID: req.RequestID}, nil
}

func (f *fakeRecorder) AcknowledgeMessage(ctx context.Context, callerAgentID domain.AgentID, req api.AcknowledgeMessageRequest) (domain.Receipt, error) {
	if req.MessageID == f.denyAckFor {
		return domain.Receipt{}, &domain.Error{Code: domain.ErrDenied, Detail: "not the recipient"}
	}
	f.acked = append(f.acked, struct {
		caller domain.AgentID
		msgID  domain.MessageID
	}{callerAgentID, req.MessageID})
	return domain.Receipt{RequestID: req.RequestID}, nil
}

func TestDeliverer_DeliverPending_OnlyQueuedNotYetPublished(t *testing.T) {
	m, _ := mustOpen(t)
	rec := &fakeRecorder{
		messages: []domain.Message{
			{WorkspaceID: "ws1", MessageID: "m1", SenderAgentID: "engineer", RecipientAgentID: "reviewer", Kind: domain.MessageRequest, QueuedAt: tPtr()},
			{WorkspaceID: "ws1", MessageID: "m2", SenderAgentID: "engineer", RecipientAgentID: "reviewer", Kind: domain.MessageRequest}, // never queued
		},
	}
	d := mailbox.NewDeliverer(m, rec)
	ctx := context.Background()

	if err := d.DeliverPending(ctx); err != nil {
		t.Fatalf("DeliverPending: %v", err)
	}
	if len(rec.published) != 1 || rec.published[0] != "m1" {
		t.Fatalf("published = %v, want exactly [m1]", rec.published)
	}
	got, err := m.ScanInbox(ctx, "reviewer", "")
	if err != nil {
		t.Fatalf("ScanInbox: %v", err)
	}
	if len(got) != 1 || got[0].MessageID != "m1" {
		t.Fatalf("ScanInbox = %+v, want only m1's envelope on disk", got)
	}

	// Running it again must not re-publish m1 (its PublishedAt is now set).
	if err := d.DeliverPending(ctx); err != nil {
		t.Fatalf("DeliverPending (second run): %v", err)
	}
	if len(rec.published) != 1 {
		t.Fatalf("published = %v after a second run, want still just [m1]", rec.published)
	}
}

func TestDeliverer_IngestPending_RefusesProcessedBeforePublished(t *testing.T) {
	m, root := mustOpen(t)
	// Write an envelope DIRECTLY to disk (simulating an external write,
	// or a race where the file exists before the engine's own
	// PublishedAt fact is committed) without ever calling
	// RecordMessagePublished for it.
	env := domain.Envelope{SchemaVersion: 1, MessageID: "m1", SenderAgentID: "engineer", RecipientAgentID: "reviewer", Kind: domain.MessageRequest}
	if err := m.Publish(context.Background(), env.MessageID, env); err != nil {
		t.Fatalf("Publish: %v", err)
	}
	_ = root

	rec := &fakeRecorder{
		messages: []domain.Message{
			{WorkspaceID: "ws1", MessageID: "m1", SenderAgentID: "engineer", RecipientAgentID: "reviewer", Kind: domain.MessageRequest, QueuedAt: tPtr()},
			// PublishedAt deliberately nil: the engine-side fact was never recorded.
		},
	}
	d := mailbox.NewDeliverer(m, rec)

	if err := d.IngestPending(context.Background()); err != nil {
		t.Fatalf("IngestPending: %v", err)
	}
	if len(rec.processed) != 0 {
		t.Fatalf("processed = %v, want none — PublishedAt was never set for m1", rec.processed)
	}
	got, err := m.ScanInbox(context.Background(), "reviewer", "")
	if err != nil {
		t.Fatalf("ScanInbox: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("ScanInbox = %+v, want the envelope still pending (not archived)", got)
	}
}

func TestDeliverer_IngestPending_ProcessesAndArchivesOncePublished(t *testing.T) {
	m, _ := mustOpen(t)
	env := domain.Envelope{SchemaVersion: 1, MessageID: "m1", SenderAgentID: "engineer", RecipientAgentID: "reviewer", Kind: domain.MessageRequest}
	if err := m.Publish(context.Background(), env.MessageID, env); err != nil {
		t.Fatalf("Publish: %v", err)
	}
	rec := &fakeRecorder{
		messages: []domain.Message{
			{WorkspaceID: "ws1", MessageID: "m1", SenderAgentID: "engineer", RecipientAgentID: "reviewer", Kind: domain.MessageRequest, QueuedAt: tPtr(), PublishedAt: tPtr()},
		},
	}
	d := mailbox.NewDeliverer(m, rec)

	if err := d.IngestPending(context.Background()); err != nil {
		t.Fatalf("IngestPending: %v", err)
	}
	if len(rec.processed) != 1 || rec.processed[0] != "m1" {
		t.Fatalf("processed = %v, want exactly [m1]", rec.processed)
	}
	got, err := m.ScanInbox(context.Background(), "reviewer", "")
	if err != nil {
		t.Fatalf("ScanInbox: %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("ScanInbox = %+v, want empty — Archive should follow RecordMessageProcessed", got)
	}
}

func TestDeliverer_IngestAcks_GoesThroughAcknowledgeMessageAndQuarantinesDenied(t *testing.T) {
	m, _ := mustOpen(t)
	if err := m.WriteAck("reviewer", domain.MessageAcknowledgement{
		SchemaVersion: 1, ControlRecordID: "c1", OriginalMessageID: "m1", RecipientAgentID: "reviewer",
	}); err != nil {
		t.Fatalf("WriteAck (c1): %v", err)
	}
	if err := m.WriteAck("reviewer", domain.MessageAcknowledgement{
		SchemaVersion: 1, ControlRecordID: "c2", OriginalMessageID: "m2", RecipientAgentID: "reviewer",
	}); err != nil {
		t.Fatalf("WriteAck (c2): %v", err)
	}

	rec := &fakeRecorder{denyAckFor: "m2"}
	d := mailbox.NewDeliverer(m, rec)

	if err := d.IngestAcks(context.Background(), "reviewer"); err != nil {
		t.Fatalf("IngestAcks: %v", err)
	}
	if len(rec.acked) != 1 || rec.acked[0].msgID != "m1" || rec.acked[0].caller != "reviewer" {
		t.Fatalf("acked = %+v, want exactly one ack for m1 by reviewer", rec.acked)
	}

	remaining, err := m.ScanAcks("reviewer")
	if err != nil {
		t.Fatalf("ScanAcks: %v", err)
	}
	if len(remaining) != 1 || remaining[0].ControlRecordID != "c2" {
		t.Fatalf("remaining acks = %+v, want the denied c2 record quarantined (left in place), c1 archived", remaining)
	}
}

func tPtr() *time.Time {
	now := time.Now()
	return &now
}
