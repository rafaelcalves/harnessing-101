// Package transport is H101-136's minimal item-6 file request/response
// transport: the mechanism a continuing `harnessing serve` host and a
// separate, later-attaching CLI process use to exchange one command at a
// time without a network dependency (ADR 0005, H101-135's rationale for
// why files rather than a socket). It carries requests, never a second
// state store: every response here is either a passive echo of what
// internal/core/task's Engine already decided, or a domain.Error it
// already returned — this package invents no new authority and performs
// no domain mutation of its own.
//
// Scope for this card: framing, the request/response file protocol, and
// attach/detach. It deliberately does NOT attempt to carry a live
// Subscribe event stream or run output over this transport yet — see
// Client.Subscribe's doc comment. That is output-shaped work item 5
// owns, not this card's minimal attach/detach claim.
package transport

import (
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/rafaelcalves/harnessing-101/internal/core/domain"
)

// Envelope is one client request: an operation name (matching
// api.FrontendSession's method names, snake-free to match Go identifiers
// directly — "RegisterAgent", "GetSnapshot", and so on) plus its
// undecoded JSON payload. The host decodes Payload only once it knows
// which typed request the Operation calls for.
type Envelope struct {
	Operation string          `json:"operation"`
	Payload   json.RawMessage `json:"payload,omitempty"`
}

// Response is one host answer. Result carries the raw success value
// (a domain.Receipt, a domain.Snapshot, ...); Error carries a domain.Error's
// fields verbatim, including ADR 0004's uncertain-outcome fields, so a
// client reconstructs the exact error a direct host.Capabilities caller
// would have seen — this transport must not lose information at the
// file boundary.
type Response struct {
	OK     bool            `json:"ok"`
	Result json.RawMessage `json:"result,omitempty"`
	Error  *ErrorInfo      `json:"error,omitempty"`
}

// ErrorInfo mirrors domain.Error's fields. It is defined independently
// of internal/throwawayadapter's own identically-shaped type rather than
// importing it — the two adapters proving swappability must not share a
// dispatch layer (ADR 0003), and that independence is worth a few
// duplicated lines.
type ErrorInfo struct {
	Code             string  `json:"code"`
	Detail           string  `json:"detail,omitempty"`
	RequestID        string  `json:"requestID,omitempty"`
	WorkspaceID      string  `json:"workspaceID,omitempty"`
	Effect           string  `json:"effect,omitempty"`
	Confirmation     string  `json:"confirmation,omitempty"`
	ObservedRevision *uint64 `json:"observedRevision,omitempty"`
	Uncertain        bool    `json:"uncertain,omitempty"`
}

// ToDomainError reconstructs a *domain.Error from the wire fields, the
// client side's mirror of the host's own errorInfo(...) encoding.
func (e *ErrorInfo) ToDomainError() *domain.Error {
	if e == nil {
		return nil
	}
	return &domain.Error{
		Code:             domain.ErrorCode(e.Code),
		Detail:           e.Detail,
		RequestID:        domain.RequestID(e.RequestID),
		WorkspaceID:      domain.WorkspaceID(e.WorkspaceID),
		Effect:           domain.Effect(e.Effect),
		Confirmation:     domain.ConfirmationKind(e.Confirmation),
		ObservedRevision: e.ObservedRevision,
	}
}

func errorInfo(err error) *ErrorInfo {
	var derr *domain.Error
	if de, ok := err.(*domain.Error); ok {
		derr = de
	} else {
		return &ErrorInfo{Code: string(domain.ErrIOFailure), Detail: err.Error()}
	}
	return &ErrorInfo{
		Code: string(derr.Code), Detail: derr.Detail,
		RequestID: string(derr.RequestID), WorkspaceID: string(derr.WorkspaceID),
		Effect: string(derr.Effect), Confirmation: string(derr.Confirmation),
		ObservedRevision: derr.ObservedRevision,
		Uncertain:        derr.Code == domain.ErrOutcomeUncertain,
	}
}

// attachOperation and detachOperation are reserved operation names: the
// only two an intake file may name before a session is known to the
// host (attach) and the one that ends it (detach). No api.FrontendSession
// method is named either of these, so there is no collision to guard.
const (
	attachOperation = "attach"
	detachOperation = "detach"
)

// AttachPayload is attach's own request payload: the caller identity the
// host should bind this session to, exactly as CLI flag -caller does for
// a one-shot WithSession call today. It confers no authority by itself —
// host policy (the reviewer set host.Open was given) decides what that
// caller may do, same as every other entry point.
type AttachPayload struct {
	CallerAgentID domain.AgentID `json:"callerAgentID"`
}

// serveDir, hostMarkerPath, and the intake/response layout below are the
// transport's whole file surface: one advertisement file, one directory
// per attached session, holding only request and response files. bounded
// intake, restricted to one workspace-relative subtree — no different in
// kind from the mailbox adapter's own per-agent directories.
const serveDirName = "serve"
const hostMarkerName = "host.json"
const sessionsDirName = "sessions"
const intakeDirName = "intake"
const responsesDirName = "responses"

// maxEnvelopeBytes bounds one request or response file, matching the
// mailbox adapter's own untrusted-input discipline (H101-22) — this
// transport reads files a session partner wrote, and a size bound is
// cheap insurance against a runaway or malicious writer regardless of
// how unlikely that is for a same-user local process.
const maxEnvelopeBytes = 256 * 1024

func serveDir(root string) string       { return filepath.Join(root, serveDirName) }
func hostMarkerPath(root string) string { return filepath.Join(serveDir(root), hostMarkerName) }
func sessionsDir(root string) string    { return filepath.Join(serveDir(root), sessionsDirName) }
func sessionDir(root, id string) string { return filepath.Join(sessionsDir(root), id) }
func intakeDir(root, id string) string  { return filepath.Join(sessionDir(root, id), intakeDirName) }
func responsesDir(root, id string) string {
	return filepath.Join(sessionDir(root, id), responsesDirName)
}
func requestPath(root, id, requestID string) string {
	return filepath.Join(intakeDir(root, id), requestID+".json")
}
func responsePath(root, id, requestID string) string {
	return filepath.Join(responsesDir(root, id), requestID+".json")
}

func removeMarker(root string) error {
	if err := os.Remove(hostMarkerPath(root)); err != nil && !os.IsNotExist(err) {
		return &domain.Error{Code: domain.ErrIOFailure, Detail: err.Error()}
	}
	return nil
}

// HostMarker is host.json's content: a live host's own advertisement.
// Generation exists so a later card can detect a stale marker left by a
// killed host (H101-135's staleness handling is explicitly deferred —
// this card's minimal scope only writes and removes this file around
// its own clean lifetime; it does not yet validate someone else's).
type HostMarker struct {
	Generation string `json:"generation"`
	PID        int    `json:"pid"`
}

// validID rejects anything that cannot be one safe path segment, the
// same three checks the mailbox adapter's own validID makes (H101-22) —
// duplicated rather than imported, for the same swappability-independence
// reason ErrorInfo is duplicated rather than shared.
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

func writeAtomic(dir, finalPath string, data []byte) error {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return &domain.Error{Code: domain.ErrIOFailure, Detail: err.Error()}
	}
	tmp, err := os.CreateTemp(dir, "*.tmp")
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

// readBounded reads a file up to maxEnvelopeBytes+1: ok=false for a
// missing file, a non-regular file, or one over the bound — the same
// three-way distinction the mailbox adapter's own readBounded makes
// (H101-22), so a poller reading a not-yet-fully-written file sees
// "not ready" rather than a spurious error.
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
		return nil, false, nil
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

func readDirFileNames(dir string) ([]string, error) {
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
		names = append(names, strings.TrimSuffix(e.Name(), ".json"))
	}
	return names, nil
}

// readSubdirNames is readDirFileNames' counterpart for a directory whose
// entries are themselves directories — sessionsDir, where each entry is
// one session ID, not a request file.
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
		if !e.IsDir() {
			continue
		}
		names = append(names, e.Name())
	}
	return names, nil
}
