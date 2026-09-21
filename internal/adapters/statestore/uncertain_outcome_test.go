package statestore

import (
	"context"
	"errors"
	"testing"

	"github.com/rafaelcalves/harnessing-101/internal/core/domain"
	"github.com/rafaelcalves/harnessing-101/internal/core/ports"
)

// withFsyncFault replaces fsyncDirFunc with one that always fails for
// the duration of fn, then restores it. This is ADR 0004's "inject a
// post-application confirmation fault" check, done deterministically
// rather than by racing real disk I/O.
func withFsyncFault(t *testing.T, fn func()) {
	t.Helper()
	original := fsyncDirFunc
	fsyncDirFunc = func(string) error { return errors.New("injected fsync failure") }
	defer func() { fsyncDirFunc = original }()
	fn()
}

// mustUncertain asserts err is an OutcomeUncertain *domain.Error and
// returns it for further field checks. Several call sites below discard
// that return value with `_ = mustUncertain(...)` — that is not a
// blanket-discarded error: the assertion itself already ran and already
// fails the test loudly via t.Fatalf if err is anything other than
// exactly OutcomeUncertain. The discard is only of the extra handle
// those call sites have no further use for (they don't need
// Effect/Confirmation/ObservedRevision beyond what mustUncertain already
// checked), never of an unexamined error.
func mustUncertain(t *testing.T, err error) *domain.Error {
	t.Helper()
	var derr *domain.Error
	if !errors.As(err, &derr) {
		t.Fatalf("expected *domain.Error, got %T: %v", err, err)
	}
	if derr.Code != domain.ErrOutcomeUncertain {
		t.Fatalf("expected Code=OutcomeUncertain, got %s (%v)", derr.Code, derr)
	}
	return derr
}

func noopMutate(id string) func(*domain.Snapshot) ([]domain.Event, error) {
	return func(snap *domain.Snapshot) ([]domain.Event, error) {
		snap.Agents = append(snap.Agents, domain.Agent{ID: domain.AgentID(id)})
		return []domain.Event{{ID: domain.EventID("ev-" + id), Kind: "Test"}}, nil
	}
}

// TestCommit_FreshCommitUncertainOnFsyncFailure is the ADR's "assert the
// structured Applied/Durability result" check, on stable fields — not
// on Detail's wording, which the test deliberately changes underneath
// to prove that.
func TestCommit_FreshCommitUncertainOnFsyncFailure(t *testing.T) {
	dir := t.TempDir()
	store, err := Open(dir)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer func() { _ = store.Close() }()

	var derr *domain.Error
	withFsyncFault(t, func() {
		_, _, err = store.Commit(context.Background(), "ws", ports.CommitRequest{
			CallerAgentID: "a", RequestID: "r1", PayloadFingerprint: "fp1", Mutate: noopMutate("a1"),
		})
		derr = mustUncertain(t, err)
	})

	if derr.Effect != domain.EffectApplied {
		t.Fatalf("Effect = %s, want Applied", derr.Effect)
	}
	if derr.Confirmation != domain.ConfirmationDurability {
		t.Fatalf("Confirmation = %s, want Durability", derr.Confirmation)
	}
	if derr.RequestID != "r1" {
		t.Fatalf("RequestID = %q, want %q", derr.RequestID, "r1")
	}
	if derr.WorkspaceID != "ws" {
		t.Fatalf("WorkspaceID = %q, want %q", derr.WorkspaceID, "ws")
	}
	if derr.ObservedRevision == nil || *derr.ObservedRevision != 1 {
		t.Fatalf("ObservedRevision = %v, want 1", derr.ObservedRevision)
	}

	// Change the diagnostic text: behavior must not depend on it.
	derr.Detail = "some completely different wording"
	if derr.Code != domain.ErrOutcomeUncertain || derr.Effect != domain.EffectApplied {
		t.Fatal("structured fields must survive independently of Detail")
	}
}

// TestCommit_ReplayNeverRerunsMutationAndStaysUncertainUntilFaultClears
// is the ADR's core sequence: the write applied once (one event, one
// receipt); a same-ID replay while the fault persists must not return a
// bare success (that is H101-56's bug) and must not run Mutate again;
// once the fault clears, the SAME replay confirms the ORIGINAL receipt
// with no new events and no new revision.
func TestCommit_ReplayNeverRerunsMutationAndStaysUncertainUntilFaultClears(t *testing.T) {
	dir := t.TempDir()
	store, err := Open(dir)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer func() { _ = store.Close() }()

	mutateCount := 0
	countingMutate := func(snap *domain.Snapshot) ([]domain.Event, error) {
		mutateCount++
		snap.Agents = append(snap.Agents, domain.Agent{ID: "a1"})
		return []domain.Event{{ID: "ev-1", Kind: "Test"}}, nil
	}

	var firstErr error
	withFsyncFault(t, func() {
		_, _, firstErr = store.Commit(context.Background(), "ws", ports.CommitRequest{
			CallerAgentID: "a", RequestID: "r1", PayloadFingerprint: "fp1", Mutate: countingMutate,
		})
	})
	_ = mustUncertain(t, firstErr)
	if mutateCount != 1 {
		t.Fatalf("mutateCount after first attempt = %d, want 1", mutateCount)
	}

	// Replay while the fault is STILL present: must stay uncertain, and
	// must not run Mutate again (H101-56's whole point — replay must
	// not silently upgrade an unconfirmed write to success, and it also
	// must not pretend to redo work that already applied).
	var secondErr error
	withFsyncFault(t, func() {
		_, _, secondErr = store.Commit(context.Background(), "ws", ports.CommitRequest{
			CallerAgentID: "a", RequestID: "r1", PayloadFingerprint: "fp1", Mutate: countingMutate,
		})
	})
	_ = mustUncertain(t, secondErr)
	if mutateCount != 1 {
		t.Fatalf("mutateCount after replay-while-uncertain = %d, want still 1 (no rerun)", mutateCount)
	}

	// Fault clears: the SAME replay now confirms the ORIGINAL receipt —
	// no new event, no new revision.
	receipt, events, err := store.Commit(context.Background(), "ws", ports.CommitRequest{
		CallerAgentID: "a", RequestID: "r1", PayloadFingerprint: "fp1", Mutate: countingMutate,
	})
	if err != nil {
		t.Fatalf("Commit after fault cleared: %v", err)
	}
	if mutateCount != 1 {
		t.Fatalf("mutateCount after confirmed replay = %d, want still 1", mutateCount)
	}
	if receipt.CommittedRevision != 1 {
		t.Fatalf("CommittedRevision = %d, want 1 (no new revision)", receipt.CommittedRevision)
	}
	if len(events) != 1 || events[0].ID != "ev-1" {
		t.Fatalf("events = %v, want exactly the original ev-1", events)
	}
}

// TestFileStore_ResolveRequest exercises Absent, uncertain-until-cleared,
// and Confirmed-after-recovery through the dedicated resolution
// operation (not a replayed Commit): it performs no mutation and must
// not advance the workspace revision even when it succeeds.
func TestFileStore_ResolveRequest(t *testing.T) {
	dir := t.TempDir()
	store, err := Open(dir)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer func() { _ = store.Close() }()
	ctx := context.Background()

	t.Run("absent is NotFound, not a crash", func(t *testing.T) {
		_, err := store.ResolveRequest(ctx, "ws", "a", "never-existed")
		var derr *domain.Error
		if !errors.As(err, &derr) || derr.Code != domain.ErrNotFound {
			t.Fatalf("ResolveRequest(absent) = %v, want *domain.Error{Code: NotFound}", err)
		}
	})

	// Seed WITH the fault active, so the commit itself lands uncertain
	// and durableRevision is never advanced for it — otherwise a later
	// ResolveRequest would correctly short-circuit to success from this
	// process's own in-memory durableRevision (see FileStore's doc
	// comment) without needing the fault to matter at all, which would
	// not exercise the case this subtest is for.
	withFsyncFault(t, func() {
		_, _, err = store.Commit(ctx, "ws", ports.CommitRequest{
			CallerAgentID: "a", RequestID: "r1", PayloadFingerprint: "fp1", Mutate: noopMutate("a1"),
		})
	})
	_ = mustUncertain(t, err)

	t.Run("another principal's request is indistinguishable from absent", func(t *testing.T) {
		_, err := store.ResolveRequest(ctx, "ws", "someone-else", "r1")
		var derr *domain.Error
		if !errors.As(err, &derr) || derr.Code != domain.ErrNotFound {
			t.Fatalf("cross-caller ResolveRequest = %v, want NotFound", err)
		}
	})

	t.Run("uncertain until the fault clears, then Confirmed with the original receipt", func(t *testing.T) {
		var uncertainErr error
		withFsyncFault(t, func() {
			_, uncertainErr = store.ResolveRequest(ctx, "ws", "a", "r1")
		})
		_ = mustUncertain(t, uncertainErr)

		receipt, err := store.ResolveRequest(ctx, "ws", "a", "r1")
		if err != nil {
			t.Fatalf("ResolveRequest after fault cleared: %v", err)
		}
		if receipt.RequestID != "r1" || receipt.CommittedRevision != 1 {
			t.Fatalf("resolved receipt = %+v, want the original (request r1, revision 1)", receipt)
		}
	})
}
