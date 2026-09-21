// Package mailbox is the local-filesystem implementation of
// ports.Mailbox (outbound port 6): one directory tree per workspace,
// one subdirectory per agent, envelopes and acknowledgement control
// records as individual JSON files. See boundaries.md's mailbox and
// file-acknowledgement sections for the contract this implements.
//
// This is the first component whose whole job is reading files another
// agent's process may have written directly (boundaries.md's threat
// model: workspace file content is untrusted input once it reaches
// another agent). Every read here treats file content as a claim to
// validate, never a fact to trust — see ScanInbox and ScanAcks.
package mailbox

import (
	"context"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/rafaelcalves/harnessing-101/internal/core/domain"
	"github.com/rafaelcalves/harnessing-101/internal/core/ports"
)

var _ ports.Mailbox = (*FileMailbox)(nil)

const (
	inboxDirName       = "inbox"
	archiveDirName     = "archive"
	acksDirName        = "acks"
	acksArchiveDirName = "acks-archive"
	tmpSuffix          = ".tmp"

	// maxEnvelopeBytes bounds a single envelope or acknowledgement file.
	// boundaries.md requires file adapters to "impose configured size
	// bounds" before presenting content to the core; this is that bound.
	// One workspace-scoped constant rather than a per-call parameter,
	// matching this port's other adapter (FileStore) having no runtime
	// size configuration either — revisit if a real deployment needs a
	// different limit per workspace.
	maxEnvelopeBytes = 256 * 1024
)

// FileMailbox is a single-workspace-root mailbox: <root>/<agentID>/inbox,
// <root>/<agentID>/archive, <root>/<agentID>/acks,
// <root>/<agentID>/acks-archive. It holds no lock of its own — unlike
// FileStore, concurrent mailbox directories for different agents do not
// contend, and this adapter does not claim single-writer semantics for
// one agent's directory; ScanInbox/ScanAcks tolerate concurrent writers
// dropping in new files, which is exactly what boundaries.md's "external
// agents may write complete inbox envelopes through the documented
// protocol" describes.
type FileMailbox struct {
	root string
}

// Open resolves root once. It does not create per-agent directories
// eagerly — an agent with nothing ever sent to or from it should not
// need a directory to exist, and ScanInbox/ScanAcks against a
// non-existent directory is treated as "nothing pending," not IOFailure.
func Open(root string) (*FileMailbox, error) {
	abs, err := filepath.Abs(root)
	if err != nil {
		return nil, &domain.Error{Code: domain.ErrIOFailure, Detail: err.Error()}
	}
	if err := os.MkdirAll(abs, 0o755); err != nil {
		return nil, &domain.Error{Code: domain.ErrIOFailure, Detail: err.Error()}
	}
	return &FileMailbox{root: abs}, nil
}

// ScanInbox is outbound port 6's read side. cursor is accepted, per the
// port signature, but not yet used to skip already-seen files: every
// call returns every envelope currently in the agent's inbox directory.
// This is deliberate, not an oversight — boundaries.md explicitly
// tolerates repeats ("Scan can repeat entries... Duplicate scans are
// harmless through persisted deduplication"), and every consumer of
// this method (the Deliverer below, and RecordMessagePublished/
// AcknowledgeMessage upstream) is already idempotent by message ID.
// Revisit if a large inbox makes a full directory scan too expensive —
// nothing about this design forecloses using cursor for that later.
//
// Untrusted input handling: a file that is oversized, not valid JSON,
// or fails schema/consistency checks is SKIPPED, not returned and not
// deleted — "invalid envelopes yield typed validation failures and
// remain available for diagnosis" (boundaries.md port 6). One bad file
// must not block delivery of every other envelope in the same
// directory, and an operator can still inspect it directly on disk;
// this method does not build a second reporting channel for it.
func (m *FileMailbox) ScanInbox(ctx context.Context, agentID domain.AgentID, cursor string) ([]domain.Envelope, error) {
	_ = ctx
	_ = cursor
	return m.scanEnvelopes(agentID, inboxDirName)
}

func (m *FileMailbox) scanEnvelopes(agentID domain.AgentID, dirName string) ([]domain.Envelope, error) {
	dir, err := m.agentDir(agentID, dirName)
	if err != nil {
		return nil, err
	}
	names, err := readDirNames(dir)
	if err != nil {
		return nil, err
	}

	envelopes := make([]domain.Envelope, 0, len(names))
	for _, name := range names {
		env, ok, err := m.readEnvelope(dir, name)
		if err != nil {
			return nil, err
		}
		if !ok {
			continue // malformed or oversized: skipped, left in place.
		}
		envelopes = append(envelopes, env)
	}
	sort.Slice(envelopes, func(i, j int) bool { return envelopes[i].CreatedAt.Before(envelopes[j].CreatedAt) })
	return envelopes, nil
}

// readEnvelope returns ok=false (never an error) for any file this
// adapter judges malformed rather than absent — size, JSON, schema
// version, or required-field checks all fail the same way, because the
// caller's response to "not a valid envelope" is identical regardless
// of which check caught it: skip, don't delete, don't crash the scan.
func (m *FileMailbox) readEnvelope(dir, name string) (domain.Envelope, bool, error) {
	data, ok, err := readBounded(filepath.Join(dir, name))
	if err != nil {
		return domain.Envelope{}, false, err
	}
	if !ok {
		return domain.Envelope{}, false, nil
	}

	var env domain.Envelope
	if err := json.Unmarshal(data, &env); err != nil {
		return domain.Envelope{}, false, nil
	}
	if !validEnvelope(env) {
		return domain.Envelope{}, false, nil
	}
	return env, true, nil
}

func validEnvelope(env domain.Envelope) bool {
	if env.SchemaVersion != 1 {
		return false
	}
	if env.MessageID == "" || env.SenderAgentID == "" || env.RecipientAgentID == "" {
		return false
	}
	switch env.Kind {
	case domain.MessageRequest, domain.MessageInform, domain.MessageResult:
	default:
		return false
	}
	return true
}

// Publish is outbound port 6's write side: idempotent by message ID —
// publishing the same MessageID twice with the same envelope content is
// a harmless no-op; a DIFFERENT envelope under an already-published
// MessageID is rejected rather than silently overwritten, since a
// recipient may already have scanned and be acting on the first one.
// Readers only ever see a complete envelope: written to a temp file in
// the same directory, then renamed into place, matching FileStore's own
// write-temp-then-rename pattern for exactly the same reason — a reader
// mid-write must never observe a partial file.
func (m *FileMailbox) Publish(ctx context.Context, messageID domain.MessageID, envelope domain.Envelope) error {
	_ = ctx
	if err := validID(string(messageID)); err != nil {
		return err
	}
	if envelope.MessageID != messageID {
		return &domain.Error{Code: domain.ErrInvalidArgument, Detail: "envelope.MessageID does not match the published messageID"}
	}
	if envelope.SchemaVersion == 0 {
		envelope.SchemaVersion = 1
	}
	if !validEnvelope(envelope) {
		return &domain.Error{Code: domain.ErrInvalidArgument, Detail: "envelope fails validation"}
	}

	dir, err := m.agentDir(envelope.RecipientAgentID, inboxDirName)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return &domain.Error{Code: domain.ErrIOFailure, Detail: err.Error()}
	}
	finalPath := filepath.Join(dir, string(messageID)+".json")

	existing, ok, err := readBounded(finalPath)
	if err != nil {
		return err
	}
	if ok {
		var prior domain.Envelope
		if json.Unmarshal(existing, &prior) == nil && prior == envelope {
			return nil // identical republish: harmless no-op.
		}
		return &domain.Error{Code: domain.ErrConflict, Detail: "messageID already published with a different envelope"}
	}

	data, err := json.MarshalIndent(envelope, "", "  ")
	if err != nil {
		return &domain.Error{Code: domain.ErrIOFailure, Detail: err.Error()}
	}
	return writeAtomic(dir, finalPath, data)
}

// Archive is outbound port 6's completion side. Archiving moves the
// envelope file into an archive subdirectory rather than deleting it —
// boundaries.md says the message body/history remains queryable after
// archival (through the state store's own Message record, not this
// file), but nothing requires this adapter to destroy its own copy, and
// keeping it costs nothing this phase while giving an operator a real
// trail. Archiving an already-archived or never-published message is a
// no-op, not an error — Archive is scheduled by the Deliverer, not
// requested by an external caller, so there is no caller-facing replay
// concern here the way there is for Commands.Execute.
func (m *FileMailbox) Archive(ctx context.Context, messageID domain.MessageID) error {
	_ = ctx
	return m.archiveAcrossAgents(string(messageID), inboxDirName, archiveDirName)
}

// archiveAcrossAgents is factored out because Archive's signature (port
// 6) is keyed only by messageID, not by which agent's inbox holds it —
// this adapter must find it. ArchiveAck (below, adapter-only, not part
// of the port) takes the agent explicitly instead, since the caller
// already knows it from ScanAcks.
func (m *FileMailbox) archiveAcrossAgents(id, fromDir, toDir string) error {
	if err := validID(id); err != nil {
		return err
	}
	agentDirs, err := readSubdirNames(m.root)
	if err != nil {
		return err
	}
	for _, agentName := range agentDirs {
		from := filepath.Join(m.root, agentName, fromDir, id+".json")
		if _, err := os.Stat(from); err != nil {
			continue
		}
		to := filepath.Join(m.root, agentName, toDir, id+".json")
		if err := os.MkdirAll(filepath.Dir(to), 0o755); err != nil {
			return &domain.Error{Code: domain.ErrIOFailure, Detail: err.Error()}
		}
		if err := os.Rename(from, to); err != nil {
			return &domain.Error{Code: domain.ErrIOFailure, Detail: err.Error()}
		}
		return nil
	}
	return nil // already archived, or never published: a harmless no-op.
}

// WriteAck, ScanAcks, and ArchiveAck are the file-protocol
// acknowledgement side (boundaries.md "file acknowledgement and
// archival"). They are deliberately NOT on ports.Mailbox — the doc
// there says the control record "may arrive through the same existing
// Mailbox port; no additional port is needed," which this reads as
// permission to keep it a port-shaped adapter capability rather than a
// mandate to force a structurally different record through Publish's
// Envelope-typed parameter. Only code holding a concrete *FileMailbox
// (the Deliverer, constructed by trusted assembly) can reach these —
// nothing holding only the ports.Mailbox interface can.
func (m *FileMailbox) WriteAck(recipientAgentID domain.AgentID, ack domain.MessageAcknowledgement) error {
	if err := validID(string(recipientAgentID)); err != nil {
		return err
	}
	if err := validID(ack.ControlRecordID); err != nil {
		return err
	}
	if ack.RecipientAgentID != recipientAgentID {
		return &domain.Error{Code: domain.ErrInvalidArgument, Detail: "ack.RecipientAgentID does not match the writing agent"}
	}
	if ack.SchemaVersion == 0 {
		ack.SchemaVersion = 1
	}
	if ack.OriginalMessageID == "" {
		return &domain.Error{Code: domain.ErrInvalidArgument, Detail: "originalMessageID is required"}
	}

	dir, err := m.agentDir(recipientAgentID, acksDirName)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return &domain.Error{Code: domain.ErrIOFailure, Detail: err.Error()}
	}
	finalPath := filepath.Join(dir, ack.ControlRecordID+".json")
	data, err := json.MarshalIndent(ack, "", "  ")
	if err != nil {
		return &domain.Error{Code: domain.ErrIOFailure, Detail: err.Error()}
	}
	return writeAtomic(dir, finalPath, data)
}

// ScanAcks reads recipientAgentID's own pending acknowledgement control
// records. recipientAgentID is the host-validated submitting scope —
// which agent's directory this call was told to read — established by
// the caller, never by a field inside any file found there;
// validAck below cross-checks the claimed RecipientAgentID against it
// and skips (does not trust) a mismatch, the same "supplies the scope
// separately from the claimed field" rule boundaries.md states for this
// exact record.
func (m *FileMailbox) ScanAcks(recipientAgentID domain.AgentID) ([]domain.MessageAcknowledgement, error) {
	dir, err := m.agentDir(recipientAgentID, acksDirName)
	if err != nil {
		return nil, err
	}
	names, err := readDirNames(dir)
	if err != nil {
		return nil, err
	}

	acks := make([]domain.MessageAcknowledgement, 0, len(names))
	for _, name := range names {
		data, ok, err := readBounded(filepath.Join(dir, name))
		if err != nil {
			return nil, err
		}
		if !ok {
			continue
		}
		var ack domain.MessageAcknowledgement
		if json.Unmarshal(data, &ack) != nil {
			continue
		}
		if ack.SchemaVersion != 1 || ack.ControlRecordID == "" || ack.OriginalMessageID == "" {
			continue
		}
		if ack.RecipientAgentID != recipientAgentID {
			continue // claimed field disagrees with the trusted scope: skip, don't trust.
		}
		acks = append(acks, ack)
	}
	return acks, nil
}

// ArchiveAck removes one processed acknowledgement control record from
// recipientAgentID's pending set, by moving it to an archive
// subdirectory — same rationale as Archive. Idempotent: archiving an
// already-archived or unknown control record is a no-op.
func (m *FileMailbox) ArchiveAck(recipientAgentID domain.AgentID, controlRecordID string) error {
	if err := validID(string(recipientAgentID)); err != nil {
		return err
	}
	if err := validID(controlRecordID); err != nil {
		return err
	}
	from := filepath.Join(m.root, string(recipientAgentID), acksDirName, controlRecordID+".json")
	if _, err := os.Stat(from); err != nil {
		return nil
	}
	to := filepath.Join(m.root, string(recipientAgentID), acksArchiveDirName, controlRecordID+".json")
	if err := os.MkdirAll(filepath.Dir(to), 0o755); err != nil {
		return &domain.Error{Code: domain.ErrIOFailure, Detail: err.Error()}
	}
	if err := os.Rename(from, to); err != nil {
		return &domain.Error{Code: domain.ErrIOFailure, Detail: err.Error()}
	}
	return nil
}

// agentDir resolves <root>/<agentID>/<sub>, rejecting any agentID that
// cannot be a safe single path segment. agentID here may be a routing
// claim from envelope content (Publish's recipient) or a host-supplied
// scope (ScanInbox/ScanAcks's parameter) — this adapter enforces path
// safety on it either way, regardless of which trust level called it,
// because a filename is a filename.
func (m *FileMailbox) agentDir(agentID domain.AgentID, sub string) (string, error) {
	if err := validID(string(agentID)); err != nil {
		return "", err
	}
	return filepath.Join(m.root, string(agentID), sub), nil
}

// validID rejects anything that cannot be one safe path segment: empty,
// containing a path separator, or a "." / ".." traversal token. IDs in
// this codebase are opaque per boundaries.md port 10 ("no path, PID,
// display name, or timestamp") — this only checks the property this
// adapter actually depends on, not that origin promise.
func validID(id string) error {
	if id == "" {
		return &domain.Error{Code: domain.ErrInvalidArgument, Detail: "id must not be empty"}
	}
	if id == "." || id == ".." {
		return &domain.Error{Code: domain.ErrInvalidArgument, Detail: "id must not be a path traversal token: " + id}
	}
	if strings.ContainsAny(id, "/\\") {
		return &domain.Error{Code: domain.ErrInvalidArgument, Detail: "id must not contain a path separator: " + id}
	}
	return nil
}

// readBounded reads a file up to maxEnvelopeBytes+1: ok=false for a
// missing file or one over the bound, distinguished from a real error
// (permission, a directory where a file was expected) so callers can
// treat "not a usable envelope" uniformly without conflating it with an
// I/O failure worth surfacing.
func readBounded(path string) ([]byte, bool, error) {
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, false, nil
		}
		return nil, false, &domain.Error{Code: domain.ErrIOFailure, Detail: err.Error()}
	}
	defer func() { _ = f.Close() }()

	info, err := f.Stat()
	if err != nil {
		return nil, false, &domain.Error{Code: domain.ErrIOFailure, Detail: err.Error()}
	}
	if !info.Mode().IsRegular() {
		return nil, false, nil // a directory or special file where an envelope was expected.
	}
	if info.Size() > maxEnvelopeBytes {
		return nil, false, nil
	}

	data := make([]byte, info.Size())
	if _, err := io.ReadFull(f, data); err != nil {
		return nil, false, &domain.Error{Code: domain.ErrIOFailure, Detail: err.Error()}
	}
	return data, true, nil
}

// readSubdirNames lists immediate subdirectory names — used to find
// which agent directories exist under root, as opposed to readDirNames'
// job of listing .json files within ONE agent's directory. Archive is
// keyed only by messageID (port 6's signature), so finding it means
// checking each agent's directory in turn.
func readSubdirNames(dir string) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, &domain.Error{Code: domain.ErrIOFailure, Detail: err.Error()}
	}
	names := make([]string, 0, len(entries))
	for _, e := range entries {
		if e.IsDir() {
			names = append(names, e.Name())
		}
	}
	return names, nil
}

func readDirNames(dir string) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, &domain.Error{Code: domain.ErrIOFailure, Detail: err.Error()}
	}
	names := make([]string, 0, len(entries))
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		names = append(names, e.Name())
	}
	return names, nil
}

// writeAtomic is Publish's/WriteAck's shared write-temp-then-rename step,
// so a concurrent reader (ScanInbox/ScanAcks, possibly another process)
// never observes a partially written file.
func writeAtomic(dir, finalPath string, data []byte) error {
	tmp, err := os.CreateTemp(dir, "*"+tmpSuffix)
	if err != nil {
		return &domain.Error{Code: domain.ErrIOFailure, Detail: err.Error()}
	}
	tmpPath := tmp.Name()
	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		_ = os.Remove(tmpPath)
		return &domain.Error{Code: domain.ErrIOFailure, Detail: err.Error()}
	}
	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		_ = os.Remove(tmpPath)
		return &domain.Error{Code: domain.ErrIOFailure, Detail: err.Error()}
	}
	if err := tmp.Close(); err != nil {
		_ = os.Remove(tmpPath)
		return &domain.Error{Code: domain.ErrIOFailure, Detail: err.Error()}
	}
	if err := os.Rename(tmpPath, finalPath); err != nil {
		_ = os.Remove(tmpPath)
		return &domain.Error{Code: domain.ErrIOFailure, Detail: err.Error()}
	}
	return nil
}
