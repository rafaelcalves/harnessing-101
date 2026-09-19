package statestore

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/rafaelcalves/harnessing-101/internal/core/domain"
	"github.com/rafaelcalves/harnessing-101/internal/core/ports"
)

// White-box: safeJoin is the guard for Creed's rule 3 ("reject .. and
// symlink escapes"). It has no live caller-supplied input in this slice
// (StateStore only ever joins its own constant filenames), but the rule
// must hold once something like Mailbox starts feeding it names.
func TestSafeJoinRejectsEscape(t *testing.T) {
	root := t.TempDir()

	if _, err := safeJoin(root, "../outside.json"); err == nil {
		t.Fatal("safeJoin accepted a .. escape")
	}
	if _, err := safeJoin(root, "/etc/passwd"); err == nil {
		t.Fatal("safeJoin accepted an absolute path")
	}
	if _, err := safeJoin(root, "a/../../outside.json"); err == nil {
		t.Fatal("safeJoin accepted a nested .. escape")
	}

	got, err := safeJoin(root, "state.json")
	if err != nil {
		t.Fatalf("safeJoin rejected a legitimate name: %v", err)
	}
	want := filepath.Join(root, "state.json")
	if got != want {
		t.Fatalf("safeJoin(%q) = %q, want %q", "state.json", got, want)
	}
}

// A symlinked workspace root must not change which real directory ends
// up holding the state file: Open resolves it once, and every write
// lands under that resolved path.
func TestOpen_ResolvesSymlinkedRoot(t *testing.T) {
	realDir := filepath.Join(t.TempDir(), "real")
	if err := os.MkdirAll(realDir, 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	linkDir := filepath.Join(t.TempDir(), "link")
	if err := os.Symlink(realDir, linkDir); err != nil {
		t.Skipf("symlinks unsupported here: %v", err)
	}

	store, err := Open(linkDir)
	if err != nil {
		t.Fatalf("Open(symlinked root): %v", err)
	}
	defer func() { _ = store.Close() }()

	ctx := context.Background()
	_, _, err = store.Commit(ctx, domain.WorkspaceID("ws"), ports.CommitRequest{
		CallerAgentID:      "a",
		RequestID:          "r1",
		PayloadFingerprint: "fp",
		Mutate: func(snap *domain.Snapshot) ([]domain.Event, error) {
			snap.Agents = append(snap.Agents, domain.Agent{ID: "a", DisplayName: "A"})
			return nil, nil
		},
	})
	if err != nil {
		t.Fatalf("Commit: %v", err)
	}

	if _, err := os.Stat(filepath.Join(realDir, stateFileName)); err != nil {
		t.Fatalf("state file was not written under the resolved real root: %v", err)
	}
}

func TestOpen_SecondOpenIsBusy(t *testing.T) {
	dir := t.TempDir()
	first, err := Open(dir)
	if err != nil {
		t.Fatalf("first Open: %v", err)
	}
	defer func() { _ = first.Close() }()

	_, err = Open(dir)
	if err == nil {
		t.Fatal("second Open over the same root succeeded; want Busy")
	}
	derr, ok := err.(*domain.Error)
	if !ok || derr.Code != domain.ErrBusy {
		t.Fatalf("second Open error = %v, want *domain.Error{Code: Busy}", err)
	}
}

func TestCommit_DifferentPayloadSameRequestIDIsConflict(t *testing.T) {
	dir := t.TempDir()
	store, err := Open(dir)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer func() { _ = store.Close() }()

	ctx := context.Background()
	noop := func(snap *domain.Snapshot) ([]domain.Event, error) { return nil, nil }

	if _, _, err := store.Commit(ctx, "ws", ports.CommitRequest{
		CallerAgentID: "a", RequestID: "r1", PayloadFingerprint: "fp-1", Mutate: noop,
	}); err != nil {
		t.Fatalf("first commit: %v", err)
	}

	_, _, err = store.Commit(ctx, "ws", ports.CommitRequest{
		CallerAgentID: "a", RequestID: "r1", PayloadFingerprint: "fp-2", Mutate: noop,
	})
	derr, ok := err.(*domain.Error)
	if !ok || derr.Code != domain.ErrConflict {
		t.Fatalf("replay with different payload = %v, want *domain.Error{Code: Conflict}", err)
	}
}
