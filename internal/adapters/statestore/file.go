// Package statestore is the local-filesystem implementation of
// ports.StateStore (outbound port 5). One JSON file under the workspace
// root holds the whole snapshot plus the command-replay receipt ledger;
// commits are atomic via write-temp-then-rename.
package statestore

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/rafaelcalves/harnessing-101/internal/core/domain"
	"github.com/rafaelcalves/harnessing-101/internal/core/ports"
)

const (
	stateFileName = "state.json"
	tmpFileName   = "state.json.tmp"
	lockFileName  = ".lock"
)

// FileStore is a single-process, single-writer StateStore rooted at one
// directory. Creed's rule 3 ("all writes stay under the resolved
// workspace root, and reject .. and symlink escapes") is enforced by
// resolving the root once at open time and by safeJoin rejecting any
// non-constant name that would not resolve back under it — there is no
// caller-supplied path component in this slice (StateStore's filenames
// are fixed constants), but the guard exists so a later port (Mailbox)
// cannot introduce an escape by accident.
type FileStore struct {
	mu       sync.Mutex
	root     string
	lockPath string
	lockFile *os.File
}

// Open resolves root, creates it if missing, and takes the workspace
// lock. A second Open against the same root fails Busy — boundaries.md
// "one host holds an exclusive operating-system-backed workspace lock; a
// second writer fails Busy." The lock is released by Close.
func Open(root string) (*FileStore, error) {
	resolved, err := resolveRoot(root)
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(resolved, 0o755); err != nil {
		return nil, fmt.Errorf("create workspace root: %w", err)
	}
	// Re-resolve after MkdirAll so a root that did not exist yet still
	// gets symlink-checked once it does.
	resolved, err = resolveRoot(root)
	if err != nil {
		return nil, err
	}

	lockPath, err := safeJoin(resolved, lockFileName)
	if err != nil {
		return nil, err
	}
	lockFile, err := os.OpenFile(lockPath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o644)
	if err != nil {
		if os.IsExist(err) {
			return nil, &domain.Error{Code: domain.ErrBusy, Detail: "workspace root is already locked by another store"}
		}
		return nil, fmt.Errorf("acquire workspace lock: %w", err)
	}

	return &FileStore{root: resolved, lockPath: lockPath, lockFile: lockFile}, nil
}

// Close releases the workspace lock. It does not delete persisted state.
func (s *FileStore) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.lockFile == nil {
		return nil
	}
	closeErr := s.lockFile.Close()
	removeErr := os.Remove(s.lockPath)
	s.lockFile = nil
	if closeErr != nil {
		return closeErr
	}
	return removeErr
}

type receiptRecord struct {
	Receipt            domain.Receipt
	Events             []domain.Event
	PayloadFingerprint string
}

type persistedState struct {
	Snapshot domain.Snapshot          `json:"snapshot"`
	Receipts map[string]receiptRecord `json:"receipts"`
}

func receiptKey(callerAgentID domain.AgentID, requestID domain.RequestID) string {
	// Request IDs are scoped to the host-established caller identity
	// (boundaries.md "retries and conflicts") — replaying someone else's
	// request ID must not return their receipt.
	return string(callerAgentID) + "\x00" + string(requestID)
}

func (s *FileStore) Load(ctx context.Context, workspaceID domain.WorkspaceID) (domain.Snapshot, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	state, err := s.readLocked()
	if err != nil {
		return domain.Snapshot{}, err
	}
	if state.Snapshot.WorkspaceID == "" {
		return domain.Snapshot{WorkspaceID: workspaceID}, nil
	}
	return state.Snapshot, nil
}

func (s *FileStore) Commit(ctx context.Context, workspaceID domain.WorkspaceID, req ports.CommitRequest) (domain.Receipt, []domain.Event, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	state, err := s.readLocked()
	if err != nil {
		return domain.Receipt{}, nil, err
	}
	if state.Receipts == nil {
		state.Receipts = make(map[string]receiptRecord)
	}

	key := receiptKey(req.CallerAgentID, req.RequestID)
	if existing, found := state.Receipts[key]; found {
		if existing.PayloadFingerprint == req.PayloadFingerprint {
			return existing.Receipt, existing.Events, nil
		}
		return domain.Receipt{}, nil, &domain.Error{Code: domain.ErrConflict, Detail: "request ID already used with a different payload"}
	}

	working := cloneSnapshot(state.Snapshot)
	if working.WorkspaceID == "" {
		working.WorkspaceID = workspaceID
	}

	events, err := req.Mutate(&working)
	if err != nil {
		// Commands that fail validation commit nothing: no snapshot write,
		// no receipt, so a retry re-validates against real state.
		return domain.Receipt{}, nil, err
	}

	working.Revision++
	for i := range events {
		events[i].WorkspaceRevision = working.Revision
	}

	receipt := domain.Receipt{RequestID: req.RequestID, CommittedRevision: working.Revision}
	state.Snapshot = working
	state.Receipts[key] = receiptRecord{Receipt: receipt, Events: events, PayloadFingerprint: req.PayloadFingerprint}

	if err := s.writeLocked(state); err != nil {
		return domain.Receipt{}, nil, err
	}
	return receipt, events, nil
}

func (s *FileStore) readLocked() (persistedState, error) {
	path, err := safeJoin(s.root, stateFileName)
	if err != nil {
		return persistedState{}, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return persistedState{Receipts: make(map[string]receiptRecord)}, nil
		}
		return persistedState{}, &domain.Error{Code: domain.ErrIOFailure, Detail: err.Error()}
	}
	var state persistedState
	if err := json.Unmarshal(data, &state); err != nil {
		return persistedState{}, &domain.Error{Code: domain.ErrIOFailure, Detail: "corrupt state file: " + err.Error()}
	}
	if state.Receipts == nil {
		state.Receipts = make(map[string]receiptRecord)
	}
	return state, nil
}

// writeLocked persists atomically: write the full new state to a temp
// file under the same root, fsync it, then rename over the real file.
// rename(2) within one filesystem is atomic, so a crash mid-write leaves
// either the old or the new complete file, never a torn one.
func (s *FileStore) writeLocked(state persistedState) error {
	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return &domain.Error{Code: domain.ErrIOFailure, Detail: err.Error()}
	}

	tmpPath, err := safeJoin(s.root, tmpFileName)
	if err != nil {
		return err
	}
	finalPath, err := safeJoin(s.root, stateFileName)
	if err != nil {
		return err
	}

	f, err := os.OpenFile(tmpPath, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o644)
	if err != nil {
		return &domain.Error{Code: domain.ErrIOFailure, Detail: err.Error()}
	}
	if _, err := f.Write(data); err != nil {
		_ = f.Close()
		return &domain.Error{Code: domain.ErrIOFailure, Detail: err.Error()}
	}
	if err := f.Sync(); err != nil {
		_ = f.Close()
		return &domain.Error{Code: domain.ErrIOFailure, Detail: err.Error()}
	}
	if err := f.Close(); err != nil {
		return &domain.Error{Code: domain.ErrIOFailure, Detail: err.Error()}
	}
	if err := os.Rename(tmpPath, finalPath); err != nil {
		return &domain.Error{Code: domain.ErrIOFailure, Detail: err.Error()}
	}
	return nil
}

func cloneSnapshot(snap domain.Snapshot) domain.Snapshot {
	out := snap
	out.Agents = append([]domain.Agent(nil), snap.Agents...)
	out.Tasks = append([]domain.Task(nil), snap.Tasks...)
	for i := range out.Tasks {
		if snap.Tasks[i].CurrentResultID != nil {
			id := *snap.Tasks[i].CurrentResultID
			out.Tasks[i].CurrentResultID = &id
		}
	}
	out.TaskResults = append([]domain.TaskResult(nil), snap.TaskResults...)
	for i := range out.TaskResults {
		out.TaskResults[i].Artifacts = append([]string(nil), snap.TaskResults[i].Artifacts...)
	}
	return out
}

// resolveRoot turns a possibly relative, possibly symlinked path into the
// absolute, symlink-resolved directory writes are confined to.
func resolveRoot(root string) (string, error) {
	abs, err := filepath.Abs(root)
	if err != nil {
		return "", &domain.Error{Code: domain.ErrInvalidArgument, Detail: err.Error()}
	}
	abs = filepath.Clean(abs)
	if resolved, err := filepath.EvalSymlinks(abs); err == nil {
		return resolved, nil
	}
	// Root does not exist yet: caller creates it next, nothing to resolve.
	return abs, nil
}

// safeJoin joins a fixed adapter-chosen name onto root and refuses
// anything that could escape it: an absolute name, a ".." segment, or a
// join whose resolved parent directory is not root itself (a symlinked
// root component swapped out from under the adapter after Open).
func safeJoin(root, name string) (string, error) {
	if filepath.IsAbs(name) {
		return "", &domain.Error{Code: domain.ErrInvalidArgument, Detail: "path escape: absolute name " + name}
	}
	clean := filepath.Clean(name)
	if clean == ".." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) {
		return "", &domain.Error{Code: domain.ErrInvalidArgument, Detail: "path escape: .. in name " + name}
	}

	joined := filepath.Join(root, clean)
	if !strings.HasPrefix(joined, root+string(filepath.Separator)) && joined != root {
		return "", &domain.Error{Code: domain.ErrInvalidArgument, Detail: "path escape: resolved outside workspace root"}
	}
	return joined, nil
}
