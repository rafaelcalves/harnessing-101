package mailbox_test

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/rafaelcalves/harnessing-101/internal/adapters/mailbox"
	"github.com/rafaelcalves/harnessing-101/internal/core/domain"
)

// mustOpen returns the adapter AND the root directory it was opened
// against, since FileMailbox exports no way to recover its root — tests
// that need to drop a raw file directly (simulating an external write)
// use the root they themselves gave Open, not a reflection trick.
func mustOpen(t *testing.T) (*mailbox.FileMailbox, string) {
	t.Helper()
	root := t.TempDir()
	m, err := mailbox.Open(root)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	return m, root
}

func sampleEnvelope() domain.Envelope {
	return domain.Envelope{
		SchemaVersion:    1,
		WorkspaceID:      "ws1",
		MessageID:        "m1",
		SenderAgentID:    "engineer",
		RecipientAgentID: "reviewer",
		Kind:             domain.MessageRequest,
		Body:             "please look",
		CreatedAt:        time.Now().UTC().Truncate(time.Second),
	}
}

func TestPublishThenScanInbox_RoundTrips(t *testing.T) {
	m, _ := mustOpen(t)
	ctx := context.Background()
	env := sampleEnvelope()

	if err := m.Publish(ctx, env.MessageID, env); err != nil {
		t.Fatalf("Publish: %v", err)
	}
	got, err := m.ScanInbox(ctx, "reviewer", "")
	if err != nil {
		t.Fatalf("ScanInbox: %v", err)
	}
	if len(got) != 1 || got[0] != env {
		t.Fatalf("ScanInbox = %+v, want [%+v]", got, env)
	}

	// A different recipient's inbox must not see it.
	other, err := m.ScanInbox(ctx, "someone-else", "")
	if err != nil {
		t.Fatalf("ScanInbox (other): %v", err)
	}
	if len(other) != 0 {
		t.Fatalf("other recipient's inbox = %+v, want empty", other)
	}
}

func TestPublish_IdempotentBySameEnvelope_ConflictOnDifferent(t *testing.T) {
	m, _ := mustOpen(t)
	ctx := context.Background()
	env := sampleEnvelope()

	if err := m.Publish(ctx, env.MessageID, env); err != nil {
		t.Fatalf("first Publish: %v", err)
	}
	if err := m.Publish(ctx, env.MessageID, env); err != nil {
		t.Fatalf("identical republish should be a no-op, got: %v", err)
	}

	changed := env
	changed.Body = "different body"
	err := m.Publish(ctx, env.MessageID, changed)
	var derr *domain.Error
	if !errors.As(err, &derr) || derr.Code != domain.ErrConflict {
		t.Fatalf("Publish (different envelope, same ID) = %v, want Conflict", err)
	}
}

func TestScanInbox_MalformedFileIsSkippedNotDeleted(t *testing.T) {
	m, root := mustOpen(t)
	ctx := context.Background()
	env := sampleEnvelope()
	if err := m.Publish(ctx, env.MessageID, env); err != nil {
		t.Fatalf("Publish: %v", err)
	}

	// Drop a malformed file directly, simulating an untrusted external
	// write (boundaries.md: "external agents may write complete inbox
	// envelopes... workspace file content is untrusted input").
	badPath := filepath.Join(inboxDirFor(root, "reviewer"), "bad.json")
	if err := os.WriteFile(badPath, []byte("{not json"), 0o644); err != nil {
		t.Fatalf("write bad file: %v", err)
	}

	got, err := m.ScanInbox(ctx, "reviewer", "")
	if err != nil {
		t.Fatalf("ScanInbox must not fail because of one bad file: %v", err)
	}
	if len(got) != 1 || got[0] != env {
		t.Fatalf("ScanInbox = %+v, want only the one valid envelope", got)
	}
	if _, err := os.Stat(badPath); err != nil {
		t.Fatalf("malformed file must remain available for diagnosis, but stat failed: %v", err)
	}
}

func TestScanInbox_OversizedFileIsSkipped(t *testing.T) {
	m, root := mustOpen(t)
	ctx := context.Background()
	env := sampleEnvelope()
	if err := m.Publish(ctx, env.MessageID, env); err != nil {
		t.Fatalf("Publish: %v", err)
	}

	oversizedPath := filepath.Join(inboxDirFor(root, "reviewer"), "huge.json")
	huge := make([]byte, 300*1024)
	if err := os.WriteFile(oversizedPath, huge, 0o644); err != nil {
		t.Fatalf("write oversized file: %v", err)
	}

	got, err := m.ScanInbox(ctx, "reviewer", "")
	if err != nil {
		t.Fatalf("ScanInbox: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("ScanInbox = %+v, want only the one valid envelope (oversized skipped)", got)
	}
}

func TestScanInbox_SchemaAndFieldValidation(t *testing.T) {
	m, root := mustOpen(t)
	ctx := context.Background()

	cases := []domain.Envelope{
		{SchemaVersion: 2, MessageID: "m1", SenderAgentID: "a", RecipientAgentID: "reviewer", Kind: domain.MessageRequest},
		{SchemaVersion: 1, MessageID: "", SenderAgentID: "a", RecipientAgentID: "reviewer", Kind: domain.MessageRequest},
		{SchemaVersion: 1, MessageID: "m3", SenderAgentID: "a", RecipientAgentID: "reviewer", Kind: "NotAKind"},
	}
	dir := inboxDirFor(root, "reviewer")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	for i, c := range cases {
		data, err := json.Marshal(c)
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, "raw"+string(rune('a'+i))+".json"), data, 0o644); err != nil {
			t.Fatal(err)
		}
	}

	got, err := m.ScanInbox(ctx, "reviewer", "")
	if err != nil {
		t.Fatalf("ScanInbox: %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("ScanInbox = %+v, want none of these invalid envelopes surfaced", got)
	}
}

func TestPublish_RejectsPathEscapeInMessageID(t *testing.T) {
	m, _ := mustOpen(t)
	ctx := context.Background()
	env := sampleEnvelope()
	env.MessageID = "../escape"
	err := m.Publish(ctx, env.MessageID, env)
	var derr *domain.Error
	if !errors.As(err, &derr) || derr.Code != domain.ErrInvalidArgument {
		t.Fatalf("Publish(path-escape messageID) = %v, want InvalidArgument", err)
	}
}

func TestPublish_RejectsMismatchedEnvelopeMessageID(t *testing.T) {
	m, _ := mustOpen(t)
	ctx := context.Background()
	env := sampleEnvelope()
	err := m.Publish(ctx, "different-id", env)
	var derr *domain.Error
	if !errors.As(err, &derr) || derr.Code != domain.ErrInvalidArgument {
		t.Fatalf("Publish(mismatched messageID) = %v, want InvalidArgument", err)
	}
}

func TestArchive_MovesFileAndIsIdempotent(t *testing.T) {
	m, _ := mustOpen(t)
	ctx := context.Background()
	env := sampleEnvelope()
	if err := m.Publish(ctx, env.MessageID, env); err != nil {
		t.Fatalf("Publish: %v", err)
	}

	if err := m.Archive(ctx, env.MessageID); err != nil {
		t.Fatalf("Archive: %v", err)
	}
	got, err := m.ScanInbox(ctx, "reviewer", "")
	if err != nil {
		t.Fatalf("ScanInbox: %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("ScanInbox after Archive = %+v, want empty (archived, not pending)", got)
	}

	// Archiving again, or archiving an unknown ID, is a harmless no-op.
	if err := m.Archive(ctx, env.MessageID); err != nil {
		t.Fatalf("re-Archive: %v", err)
	}
	if err := m.Archive(ctx, "never-existed"); err != nil {
		t.Fatalf("Archive(unknown): %v", err)
	}
}

func TestAck_WriteScanArchiveRoundTripAndScopeMismatch(t *testing.T) {
	m, _ := mustOpen(t)
	ack := domain.MessageAcknowledgement{
		SchemaVersion:     1,
		WorkspaceID:       "ws1",
		ControlRecordID:   "c1",
		OriginalMessageID: "m1",
		RecipientAgentID:  "reviewer",
	}
	if err := m.WriteAck("reviewer", ack); err != nil {
		t.Fatalf("WriteAck: %v", err)
	}

	got, err := m.ScanAcks("reviewer")
	if err != nil {
		t.Fatalf("ScanAcks: %v", err)
	}
	if len(got) != 1 || got[0] != ack {
		t.Fatalf("ScanAcks = %+v, want [%+v]", got, ack)
	}

	// A recipient scanning someone ELSE's directory never sees it, and
	// WriteAck itself refuses a claimed field that disagrees with the
	// writing agent (the host-validated scope), rather than trusting
	// the record's own content.
	if err := m.WriteAck("someone-else", ack); err == nil {
		t.Fatal("WriteAck with a mismatched RecipientAgentID should be rejected")
	}

	if err := m.ArchiveAck("reviewer", ack.ControlRecordID); err != nil {
		t.Fatalf("ArchiveAck: %v", err)
	}
	got, err = m.ScanAcks("reviewer")
	if err != nil {
		t.Fatalf("ScanAcks after archive: %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("ScanAcks after archive = %+v, want empty", got)
	}
}

func TestScanAcks_ClaimedRecipientMismatchIsSkippedNotTrusted(t *testing.T) {
	m, root := mustOpen(t)
	// Simulate a hostile/corrupted file: written under "reviewer"'s own
	// directory (the trusted scope) but claiming a different recipient
	// inside its content.
	ack := domain.MessageAcknowledgement{
		SchemaVersion: 1, ControlRecordID: "c1", OriginalMessageID: "m1", RecipientAgentID: "someone-else",
	}
	data, err := json.Marshal(ack)
	if err != nil {
		t.Fatal(err)
	}
	dir := filepath.Join(root, "reviewer", "acks")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "c1.json"), data, 0o644); err != nil {
		t.Fatal(err)
	}

	got, err := m.ScanAcks("reviewer")
	if err != nil {
		t.Fatalf("ScanAcks: %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("ScanAcks = %+v, want the claimed-mismatch record skipped, not trusted", got)
	}
}

func TestOpen_RejectsAgentIDPathEscape(t *testing.T) {
	m, _ := mustOpen(t)
	ctx := context.Background()
	_, err := m.ScanInbox(ctx, domain.AgentID("../escape"), "")
	var derr *domain.Error
	if !errors.As(err, &derr) || derr.Code != domain.ErrInvalidArgument {
		t.Fatalf("ScanInbox(path-escape agentID) = %v, want InvalidArgument", err)
	}
}

func inboxDirFor(root, agentID string) string {
	return filepath.Join(root, agentID, "inbox")
}
