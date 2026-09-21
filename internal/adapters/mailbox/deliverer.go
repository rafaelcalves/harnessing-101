package mailbox

import (
	"context"
	"errors"

	"github.com/rafaelcalves/harnessing-101/internal/api"
	"github.com/rafaelcalves/harnessing-101/internal/core/domain"
)

// Recorder is the small slice of Capabilities-shaped methods Deliverer
// needs to turn file-protocol facts into engine-recorded ones. Defined
// locally, using api's request types (the one-definition-per-record
// home Claudio's H101-74 migration established) rather than importing
// host: any concrete type whose methods match this shape structurally —
// host.Capabilities today — satisfies it without either package naming
// the other, and this package stays a leaf adapter that only imports
// ports/domain/api, matching the existing FileStore pattern.
type Recorder interface {
	RecordMessagePublished(ctx context.Context, req api.MessageDeliveryRequest) (domain.Receipt, error)
	RecordMessageProcessed(ctx context.Context, req api.MessageDeliveryRequest) (domain.Receipt, error)
	AcknowledgeMessage(ctx context.Context, callerAgentID domain.AgentID, req api.AcknowledgeMessageRequest) (domain.Receipt, error)
	GetSnapshot(ctx context.Context) (domain.Snapshot, error)
}

// Deliverer is Kelly's Phase 1 obligation (H101-22) made concrete:
// publish and process are driven ONLY here, in causal order, from facts
// the engine already committed — never independently, never out of
// order. It owns no state; every call re-derives what still needs doing
// from a fresh snapshot plus the mailbox directories.
//
// THE ORDERING GUARD THE ENGINE DOES NOT PROVIDE (one of this card's
// four decisions, made rather than assumed): recordDeliveryFact
// (internal/core/task/engine.go) has no check preventing
// RecordMessageProcessed from succeeding before RecordMessagePublished
// for the same message — it only checks the message exists. So "the
// engine's causal rules are enough once only the adapter drives them"
// is FALSE by inspection; IngestPending below enforces published-before-
// processed itself, from the snapshot's own PublishedAt fact, rather
// than trusting that nothing else could call the recorder out of order.
type Deliverer struct {
	mail     *FileMailbox
	recorder Recorder
}

func NewDeliverer(mail *FileMailbox, recorder Recorder) *Deliverer {
	return &Deliverer{mail: mail, recorder: recorder}
}

// DeliverPending finds every queued-but-not-yet-published message and
// publishes it: write the envelope, THEN record the fact. A crash
// between those two steps leaves the message re-deliverable next run —
// Publish is idempotent by message ID, and RecordMessagePublished's own
// request ID is deterministic (requestIDForFact), so replaying either
// half of this pair is always safe.
func (d *Deliverer) DeliverPending(ctx context.Context) error {
	snap, err := d.recorder.GetSnapshot(ctx)
	if err != nil {
		return err
	}
	for _, msg := range snap.Messages {
		if msg.QueuedAt == nil || msg.PublishedAt != nil {
			continue
		}
		env := domain.Envelope{
			SchemaVersion:    1,
			WorkspaceID:      msg.WorkspaceID,
			MessageID:        msg.MessageID,
			SenderAgentID:    msg.SenderAgentID,
			RecipientAgentID: msg.RecipientAgentID,
			Kind:             msg.Kind,
			Body:             msg.Body,
			CreatedAt:        msg.CreatedAt,
			TaskID:           msg.TaskID,
			ReplyToMessageID: msg.ReplyToMessageID,
		}
		if err := d.mail.Publish(ctx, msg.MessageID, env); err != nil {
			return err
		}
		if _, err := d.recorder.RecordMessagePublished(ctx, api.MessageDeliveryRequest{
			RequestID: requestIDForFact("publish", string(msg.MessageID)),
			MessageID: msg.MessageID,
		}); err != nil {
			return err
		}
	}
	return nil
}

// IngestPending scans every agent with at least one published-but-not-
// processed message and records processing for what it actually
// observes in that agent's inbox — never for a message the engine only
// THINKS should be there. A message whose own PublishedAt is still nil
// is skipped even if a file happens to already exist for it (e.g. an
// external agent wrote it directly, per boundaries.md's "external
// agents may write complete inbox envelopes"): that is exactly the
// processed-before-published case this method must refuse to create.
func (d *Deliverer) IngestPending(ctx context.Context) error {
	snap, err := d.recorder.GetSnapshot(ctx)
	if err != nil {
		return err
	}
	published := make(map[domain.MessageID]bool)
	recipients := make(map[domain.AgentID]bool)
	for _, msg := range snap.Messages {
		if msg.PublishedAt != nil && msg.ProcessedAt == nil {
			published[msg.MessageID] = true
			recipients[msg.RecipientAgentID] = true
		}
	}

	for agentID := range recipients {
		envelopes, err := d.mail.ScanInbox(ctx, agentID, "")
		if err != nil {
			return err
		}
		for _, env := range envelopes {
			if !published[env.MessageID] {
				continue // not this agent's turn yet, or already processed.
			}
			if _, err := d.recorder.RecordMessageProcessed(ctx, api.MessageDeliveryRequest{
				RequestID: requestIDForFact("process", string(env.MessageID)),
				MessageID: env.MessageID,
			}); err != nil {
				return err
			}
			if err := d.mail.Archive(ctx, env.MessageID); err != nil {
				return err
			}
		}
	}
	return nil
}

// IngestAcks drives the file-protocol acknowledgement path
// (boundaries.md "file acknowledgement and archival"): every pending
// control record in recipientAgentID's own acks directory goes through
// the SAME AcknowledgeMessage check a direct call would — this is not a
// second, weaker path to the same state. recipientAgentID is the
// caller's own host-validated scope (which agent's directory to trust);
// ScanAcks already rejected any record whose claimed RecipientAgentID
// disagreed with it, before this method ever sees the record.
//
// A record the engine's own check rejects (wrong recipient somehow
// still reaching here, an unknown message) is left in place rather than
// archived — boundaries.md requires reject/quarantine, not deletion,
// for exactly this reason, and one bad or hostile record must not stop
// every other pending acknowledgement from being ingested.
func (d *Deliverer) IngestAcks(ctx context.Context, recipientAgentID domain.AgentID) error {
	acks, err := d.mail.ScanAcks(recipientAgentID)
	if err != nil {
		return err
	}
	for _, ack := range acks {
		_, err := d.recorder.AcknowledgeMessage(ctx, recipientAgentID, api.AcknowledgeMessageRequest{
			RequestID: requestIDForFact("ack", ack.ControlRecordID),
			MessageID: ack.OriginalMessageID,
		})
		if err != nil {
			var derr *domain.Error
			if errors.As(err, &derr) && (derr.Code == domain.ErrDenied || derr.Code == domain.ErrNotFound) {
				continue // quarantined: left in place, not archived, not fatal to the rest of the scan.
			}
			return err
		}
		if err := d.mail.ArchiveAck(recipientAgentID, ack.ControlRecordID); err != nil {
			return err
		}
	}
	return nil
}

// requestIDForFact derives a stable request ID from the fact kind and
// the identity it is about, so repeating the same delivery step after a
// crash always presents the SAME (implicit caller, requestID) pair to
// the engine's replay ledger — idempotent by construction, not by
// hoping the caller remembers to reuse an ID.
func requestIDForFact(fact, id string) domain.RequestID {
	return domain.RequestID("mailbox-" + fact + "-" + id)
}
