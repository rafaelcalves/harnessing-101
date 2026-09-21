package assembly_test

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/rafaelcalves/harnessing-101/internal/adapters/mailbox"
	"github.com/rafaelcalves/harnessing-101/internal/api"
	"github.com/rafaelcalves/harnessing-101/internal/assembly"
	"github.com/rafaelcalves/harnessing-101/internal/core/domain"
)

// TestWithSession_FullMailboxPath_SendPublishProcessAck is Kelly's H101-83
// gap: no test chained SendMessage (real engine) -> publish -> process ->
// ack through the REAL assembled product path (assembly.WithSession) and
// the REAL FileMailbox — every prior test used either fakeRecorder or
// called the Deliverer directly. This uses only the same entry point
// cmd/harnessing itself calls.
func TestWithSession_FullMailboxPath_SendPublishProcessAck(t *testing.T) {
	root := t.TempDir()
	var stderr bytes.Buffer

	// 0. H101-23's ripple: SendMessage now checks both SenderAgentID and
	// RecipientAgentID against the registry, so both must be registered
	// before step 1 can send between them.
	for _, agent := range []struct{ caller, agentID string }{{"engineer", "engineer"}, {"reviewer", "reviewer"}} {
		code := assembly.WithSession(&stderr, root, "ws1", nil, domain.AgentID(agent.caller), "register", func(ctx context.Context, s api.FrontendSession) int {
			_, err := s.RegisterAgent(ctx, api.RegisterAgentRequest{
				RequestID: domain.RequestID("reg-" + agent.agentID), AgentID: domain.AgentID(agent.agentID), DisplayName: agent.agentID,
			})
			if err != nil {
				t.Fatalf("RegisterAgent(%s): %v", agent.agentID, err)
			}
			return 0
		})
		if code != 0 {
			t.Fatalf("register session exit code = %d, stderr = %q", code, stderr.String())
		}
	}

	// 1. engineer sends a message to reviewer. WithSession's own
	// post-command mailbox drive (H101-85) runs immediately after this
	// returns, in the SAME call: by the time this returns, DeliverPending
	// has already published the envelope and IngestPending has already
	// processed+archived it, since nothing else in this single-process
	// CLI could have done either step first.
	code := assembly.WithSession(&stderr, root, "ws1", nil, "engineer", "send", func(ctx context.Context, s api.FrontendSession) int {
		_, err := s.SendMessage(ctx, api.SendMessageRequest{
			RequestID: "r1", MessageID: "m1",
			SenderAgentID: "engineer", RecipientAgentID: "reviewer",
			Kind: domain.MessageRequest, Body: "please look",
		})
		if err != nil {
			t.Fatalf("SendMessage: %v", err)
		}
		return 0
	})
	if code != 0 {
		t.Fatalf("send session exit code = %d, stderr = %q", code, stderr.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("unexpected stderr from the send session: %q", stderr.String())
	}

	// 2. Confirm the delivery facts landed on the real Message record —
	// queued (from SendMessage itself), published, and processed, all
	// through the real engine + real FileMailbox, no fakes anywhere in
	// this chain.
	var msg domain.Message
	code = assembly.WithSession(&stderr, root, "ws1", nil, "reviewer", "inspect", func(ctx context.Context, s api.FrontendSession) int {
		var err error
		msg, err = s.GetMessage(ctx, "m1")
		if err != nil {
			t.Fatalf("GetMessage: %v", err)
		}
		return 0
	})
	if code != 0 {
		t.Fatalf("inspect session exit code = %d, stderr = %q", code, stderr.String())
	}
	if msg.QueuedAt == nil || msg.PublishedAt == nil || msg.ProcessedAt == nil {
		t.Fatalf("delivery facts = %+v, want queued+published+processed all set", msg)
	}
	if msg.AcknowledgedAt != nil {
		t.Fatalf("AcknowledgedAt already set before any ack was written: %+v", msg)
	}

	// 3. Simulate the file-protocol acknowledgement leg: an external
	// process acting as reviewer writes a real ack control record
	// directly into the same mailbox tree WithSession itself drives —
	// exactly what a separate reviewer-side agent process would do
	// after reading its inbox, and exactly the boundary this product
	// has no CLI command to cross by direct call.
	mail, err := mailbox.Open(root + "/mailbox")
	if err != nil {
		t.Fatalf("mailbox.Open: %v", err)
	}
	if err := mail.WriteAck("reviewer", domain.MessageAcknowledgement{
		SchemaVersion: 1, WorkspaceID: "ws1", ControlRecordID: "c1",
		OriginalMessageID: "m1", RecipientAgentID: "reviewer",
	}); err != nil {
		t.Fatalf("WriteAck: %v", err)
	}

	// 4. The NEXT session for reviewer (any command at all — this one
	// does nothing itself) ingests that ack purely as WithSession's own
	// post-command drive, through the real AcknowledgeMessage checks.
	code = assembly.WithSession(&stderr, root, "ws1", nil, "reviewer", "noop", func(ctx context.Context, s api.FrontendSession) int {
		return 0
	})
	if code != 0 {
		t.Fatalf("noop session exit code = %d, stderr = %q", code, stderr.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("unexpected stderr from the ack-ingesting session: %q", stderr.String())
	}

	code = assembly.WithSession(&stderr, root, "ws1", nil, "reviewer", "inspect2", func(ctx context.Context, s api.FrontendSession) int {
		var err error
		msg, err = s.GetMessage(ctx, "m1")
		if err != nil {
			t.Fatalf("GetMessage: %v", err)
		}
		return 0
	})
	if code != 0 {
		t.Fatalf("inspect2 session exit code = %d, stderr = %q", code, stderr.String())
	}
	if msg.AcknowledgedAt == nil || msg.AcknowledgedBy != "reviewer" {
		t.Fatalf("final message state = %+v, want AcknowledgedBy=reviewer with AcknowledgedAt set", msg)
	}

	// The ack control record must have been archived, not left pending,
	// once ingestion succeeded.
	remaining, err := mail.ScanAcks("reviewer")
	if err != nil {
		t.Fatalf("ScanAcks: %v", err)
	}
	if len(remaining) != 0 {
		t.Fatalf("remaining acks = %+v, want the ingested record archived", remaining)
	}
}

// TestWithSession_MailboxDeliveryFailureDoesNotFailTheCommand: a mailbox
// hiccup (an unwritable mailbox root) must surface on stderr but not
// turn an already-successful command's own result into a failure —
// H101-85's session.go comment states this; this proves it.
func TestWithSession_MailboxDeliveryFailureDoesNotFailTheCommand(t *testing.T) {
	root := t.TempDir()
	var stderr bytes.Buffer

	// First call succeeds normally and creates <root>/mailbox.
	code := assembly.WithSession(&stderr, root, "ws1", nil, "engineer", "create", func(ctx context.Context, s api.FrontendSession) int {
		return 0
	})
	if code != 0 {
		t.Fatalf("first session exit code = %d, stderr = %q", code, stderr.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("unexpected stderr from the first session: %q", stderr.String())
	}

	mailboxRoot := filepath.Join(root, "mailbox")
	if err := os.Chmod(mailboxRoot, 0o000); err != nil {
		t.Fatalf("chmod: %v", err)
	}
	defer func() { _ = os.Chmod(mailboxRoot, 0o755) }()

	stderr.Reset()
	code = assembly.WithSession(&stderr, root, "ws1", nil, "engineer", "create2", func(ctx context.Context, s api.FrontendSession) int {
		if _, err := s.CreateTask(ctx, api.CreateTaskRequest{RequestID: "r2", TaskID: "t1", Title: "x"}); err != nil {
			t.Fatalf("CreateTask: %v", err)
		}
		return 0
	})
	if code != 0 {
		t.Fatalf("session exit code = %d with an unwritable mailbox root, want the command's own success (0) preserved; stderr = %q", code, stderr.String())
	}
	if !strings.Contains(stderr.String(), "mailbox delivery") {
		t.Fatalf("stderr = %q, want the mailbox delivery failure surfaced even though the command itself succeeded", stderr.String())
	}
}
