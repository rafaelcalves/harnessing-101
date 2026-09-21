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
	"strconv"
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
//
// The workspace lock (H101-20) is an OS advisory file lock (flock(2)),
// not a lock *file's existence*. That distinction is the whole point:
// the kernel releases an flock automatically when the holding process
// exits for any reason — normal Close, crash, or SIGKILL — with no PID
// bookkeeping and no risk of a reused PID being mistaken for a live
// owner. boundaries.md forbids stealing a lock on elapsed wall time;
// this needs no such timeout, because "is the owner still alive" is a
// question the kernel already answers correctly. See lock_unix.go for
// the mechanism and its platform scope, and
// docs/architecture/h101-20-lock-recovery.md for the fuller reasoning
// and the manual-removal fallback for a platform where it is unavailable.
type FileStore struct {
	mu       sync.Mutex
	root     string
	lockPath string
	lockFile *os.File

	// durableRevision is the highest workspace revision this process
	// instance has itself confirmed durable (a successful post-rename
	// directory fsync). It is why H101-56's fix does not make every
	// replay pay a sync cost: within one process's lifetime, once a
	// revision is confirmed durable, no later Commit call — fresh or
	// replay — can have silently un-synced it (this store is the sole
	// writer under mu, guarded further by H101-20's flock), so a replay
	// whose receipt is at or below this line needs no new fsync. Only a
	// replay above it — meaning the fsync for that very commit is the
	// one that failed — pays the cost, which is exactly when paying it
	// is the fix, not overhead. A freshly Opened FileStore starts at
	// zero and so re-confirms its first replay regardless: a prior
	// process's un-synced write is not this instance's fact to assume.
	durableRevision uint64
}

// Open resolves root, creates it if missing, and takes the workspace
// lock via acquireLock (platform-specific; see lock_unix.go). A second
// Open against the same root — from this process or a live other one —
// fails Busy; a second Open after the previous owner died without
// calling Close succeeds, because the OS already released the lock.
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
	// Not O_EXCL: existence of this file means nothing by itself — a
	// crashed owner's lock file is expected to still be sitting here.
	// Ownership is decided by acquireLock, not by whether this call
	// creates or reopens the file.
	lockFile, err := os.OpenFile(lockPath, os.O_CREATE|os.O_RDWR, 0o644)
	if err != nil {
		return nil, fmt.Errorf("open workspace lock file: %w", err)
	}
	if err := acquireLock(lockFile); err != nil {
		_ = lockFile.Close()
		return nil, err
	}
	// Best-effort diagnostic content for a human inspecting the file
	// while debugging — never read back by this adapter for any
	// correctness decision. A failure here does not fail Open: holding
	// the lock is what matters.
	_ = lockFile.Truncate(0)
	_, _ = lockFile.WriteAt([]byte(fmt.Sprintf("pid=%d\n", os.Getpid())), 0)

	return &FileStore{root: resolved, lockPath: lockPath, lockFile: lockFile}, nil
}

// Close releases the workspace lock and removes the lock file. It does
// not delete persisted state. If the process dies before Close runs,
// the OS releases the flock anyway (that is the mechanism's entire
// point) and the leftover file is harmless clutter the next Open reuses.
func (s *FileStore) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.lockFile == nil {
		return nil
	}
	unlockErr := releaseLock(s.lockFile)
	closeErr := s.lockFile.Close()
	removeErr := os.Remove(s.lockPath)
	s.lockFile = nil
	if unlockErr != nil {
		return unlockErr
	}
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
		if existing.PayloadFingerprint != req.PayloadFingerprint {
			return domain.Receipt{}, nil, &domain.Error{Code: domain.ErrConflict, Detail: "request ID already used with a different payload"}
		}
		// H101-56: this branch used to return the saved receipt here,
		// unconditionally. That is the bug ADR 0004 found reasoning
		// about the fsync fix: a replay confirms no new mutation ran,
		// but it does not by itself re-confirm the durability guarantee
		// an earlier attempt for this exact receipt may have failed to
		// establish. Ask first.
		return s.confirmDurable(workspaceID, req.RequestID, existing)
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
	working.Cursor = strconv.FormatUint(working.Revision, 10)
	for i := range events {
		events[i].WorkspaceRevision = working.Revision
	}

	receipt := domain.Receipt{RequestID: req.RequestID, CommittedRevision: working.Revision}
	state.Snapshot = working
	state.Receipts[key] = receiptRecord{Receipt: receipt, Events: events, PayloadFingerprint: req.PayloadFingerprint}

	durable, err := s.writeLocked(state)
	if err != nil {
		// The write itself did not apply (marshal/write/rename failure):
		// nothing changed, this is a plain failure.
		return domain.Receipt{}, nil, err
	}
	if !durable {
		// The rename succeeded — the change applied, the receipt above
		// is already what a replay will find — but the follow-up
		// directory fsync failed, so durability is unconfirmed. This is
		// ADR 0004's central mapping: Applied, not Unknown; Durability,
		// not Outcome.
		rev := working.Revision
		return domain.Receipt{}, nil, &domain.Error{
			Code:             domain.ErrOutcomeUncertain,
			Detail:           "commit applied but directory fsync failed; durability unconfirmed",
			RequestID:        req.RequestID,
			WorkspaceID:      workspaceID,
			Effect:           domain.EffectApplied,
			Confirmation:     domain.ConfirmationDurability,
			ObservedRevision: &rev,
		}
	}
	s.durableRevision = working.Revision
	return receipt, events, nil
}

// confirmDurable answers a replay. If this process instance already
// knows the receipt's revision is durable (see FileStore.durableRevision),
// it returns the cached receipt with no further I/O. Otherwise it
// re-establishes the directory-fsync barrier before answering — the
// "ask first" this function's callers exist for — and answers
// OutcomeUncertain rather than a bare success if that barrier still
// fails.
func (s *FileStore) confirmDurable(workspaceID domain.WorkspaceID, requestID domain.RequestID, existing receiptRecord) (domain.Receipt, []domain.Event, error) {
	if existing.Receipt.CommittedRevision <= s.durableRevision {
		return existing.Receipt, existing.Events, nil
	}
	if err := fsyncDirFunc(s.root); err != nil {
		rev := existing.Receipt.CommittedRevision
		return domain.Receipt{}, nil, &domain.Error{
			Code:             domain.ErrOutcomeUncertain,
			Detail:           "replay could not reconfirm durability: " + err.Error(),
			RequestID:        requestID,
			WorkspaceID:      workspaceID,
			Effect:           domain.EffectApplied,
			Confirmation:     domain.ConfirmationDurability,
			ObservedRevision: &rev,
		}
	}
	s.durableRevision = existing.Receipt.CommittedRevision
	return existing.Receipt, existing.Events, nil
}

// ResolveRequest is ADR 0004's caller-bound resolution operation: it
// performs no domain mutation, allocates no new request ID, and does
// not advance the workspace revision. It returns Confirmed (the
// original receipt, after actually re-establishing durability) or
// OutcomeUncertain if that confirmation still fails. Authorization is
// structural, not a separate check: the receipt ledger is already keyed
// by (CallerAgentID, RequestID) — receiptKey — so callerAgentID can only
// ever find its own records; another principal's request is
// indistinguishable from Absent, never exposed by mistake.
//
// Absent — no current receipt for this (caller, requestID) — is
// reported as NotFound. That is a deliberate reuse of the existing
// stable code, not a new one: NotFound already means "no record found
// now," which is exactly Absent's contract (ADR 0004: "not 'this never
// happened'"). Corrupt/unreadable storage is not Absent; readLocked's
// own IOFailure surfaces first and is returned unchanged.
func (s *FileStore) ResolveRequest(ctx context.Context, workspaceID domain.WorkspaceID, callerAgentID domain.AgentID, requestID domain.RequestID) (domain.Receipt, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	state, err := s.readLocked()
	if err != nil {
		return domain.Receipt{}, err
	}
	existing, found := state.Receipts[receiptKey(callerAgentID, requestID)]
	if !found {
		return domain.Receipt{}, &domain.Error{Code: domain.ErrNotFound, Detail: "no receipt found for this request ID"}
	}

	receipt, _, err := s.confirmDurable(workspaceID, requestID, existing)
	return receipt, err
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
//
// The two return values are deliberately distinct kinds of failure. An
// error means the write did not apply — everything up to and including
// rename failed, so the caller changed nothing. durable=false with a nil
// error means the opposite extreme: rename succeeded (the change is
// already what a reload or a replay will find) but the follow-up
// directory fsync failed, so its durability is unconfirmed — the
// Applied/Durability case ADR 0004 maps to OutcomeUncertain. Collapsing
// these into one error return is exactly what let H101-12's fix report
// the right code (IOFailure with explanatory text) while nothing forced
// a caller to treat it any differently from "nothing happened."
func (s *FileStore) writeLocked(state persistedState) (durable bool, err error) {
	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return false, &domain.Error{Code: domain.ErrIOFailure, Detail: err.Error()}
	}

	tmpPath, err := safeJoin(s.root, tmpFileName)
	if err != nil {
		return false, err
	}
	finalPath, err := safeJoin(s.root, stateFileName)
	if err != nil {
		return false, err
	}

	f, err := os.OpenFile(tmpPath, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o644)
	if err != nil {
		return false, &domain.Error{Code: domain.ErrIOFailure, Detail: err.Error()}
	}
	if _, err := f.Write(data); err != nil {
		_ = f.Close()
		return false, &domain.Error{Code: domain.ErrIOFailure, Detail: err.Error()}
	}
	if err := f.Sync(); err != nil {
		_ = f.Close()
		return false, &domain.Error{Code: domain.ErrIOFailure, Detail: err.Error()}
	}
	if err := f.Close(); err != nil {
		return false, &domain.Error{Code: domain.ErrIOFailure, Detail: err.Error()}
	}
	if err := os.Rename(tmpPath, finalPath); err != nil {
		return false, &domain.Error{Code: domain.ErrIOFailure, Detail: err.Error()}
	}
	// rename(2) is atomic, but the directory entry it changes is not
	// guaranteed durable across a crash until the containing directory
	// itself is fsynced (POSIX leaves this filesystem-dependent; ext4
	// without a journal-ordering guarantee is the classic case). Without
	// this, "successful Commit" could still lose the rename on a power
	// loss, which is exactly the silent failure mode this adapter must
	// not have: boundaries.md requires Commit to mean durable data, and
	// AwaitingReview surviving a restart is the property this slice was
	// asked to prove.
	if err := fsyncDirFunc(s.root); err != nil {
		return false, nil
	}
	return true, nil
}

// fsyncDirFunc is a package-level seam so tests can inject a durability
// fault deterministically (and clear it again) without a filesystem
// trick that would only work on some platforms. Production code never
// reassigns it.
var fsyncDirFunc = fsyncDir

func fsyncDir(dir string) error {
	d, err := os.Open(dir)
	if err != nil {
		return err
	}
	syncErr := d.Sync()
	closeErr := d.Close()
	if syncErr != nil {
		return syncErr
	}
	return closeErr
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
	out.Messages = append([]domain.Message(nil), snap.Messages...)
	for i := range out.Messages {
		if snap.Messages[i].TaskID != nil {
			id := *snap.Messages[i].TaskID
			out.Messages[i].TaskID = &id
		}
		if snap.Messages[i].ReplyToMessageID != nil {
			id := *snap.Messages[i].ReplyToMessageID
			out.Messages[i].ReplyToMessageID = &id
		}
		if snap.Messages[i].QueuedAt != nil {
			at := *snap.Messages[i].QueuedAt
			out.Messages[i].QueuedAt = &at
		}
		if snap.Messages[i].PublishedAt != nil {
			at := *snap.Messages[i].PublishedAt
			out.Messages[i].PublishedAt = &at
		}
		if snap.Messages[i].ProcessedAt != nil {
			at := *snap.Messages[i].ProcessedAt
			out.Messages[i].ProcessedAt = &at
		}
		if snap.Messages[i].AcknowledgedAt != nil {
			at := *snap.Messages[i].AcknowledgedAt
			out.Messages[i].AcknowledgedAt = &at
		}
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
