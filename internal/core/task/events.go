package task

import (
	"context"
	"strconv"
	"sync"

	"github.com/rafaelcalves/harnessing-101/internal/core/domain"
)

// eventBufferCapacity bounds how many committed events one Engine
// instance retains in memory for replay. It is a process-lifetime
// buffer, not a durable log: a workspace reopened in a new process starts
// with an empty buffer and no history to resume from, which is why a
// cursor from a prior process must come back as CursorExpired rather than
// silently served as "nothing happened since." See boundaries on this
// card (H101-61): the engine and the events surface, not the store — a
// durable, cross-process event log is future work, not something this
// slice claims.
const eventBufferCapacity = 256

// subscriberQueueCapacity must be at least eventBufferCapacity so that
// seeding a new subscriber's backlog (bounded by the buffer's own size)
// can never block while eventBus.mu is held.
const subscriberQueueCapacity = eventBufferCapacity

// eventBus is Engine's in-process implementation of inbound port 3
// (ports.StateEvents): a bounded, ordered buffer of committed events plus
// live fan-out to subscribers, each filtered by its own afterCursor so a
// resumed observer never receives an event it has already been shown.
//
// THE NO-LOST-COMMITS ARGUMENT this depends on: publish and subscribe
// both take the same mutex for their entire respective critical
// sections. So for any event E and any subscribe call S requesting
// afterCursor < E's revision, either E's publish fully happened before S
// (E is already in the buffer, and S's backlog seed includes it), or it
// happened after S acquired the lock to register (S is already a live
// subscriber by the time E's publish runs, and receives it there). There
// is no interleaving in which E lands in neither place. This holds
// regardless of what revision a concurrent GetSnapshot's Load happened to
// observe — it does not need to race publish for correctness, because
// "after cursor N" excludes exactly what a snapshot showing revision N
// already reflects and includes everything published after, live or not.
type eventBus struct {
	mu            sync.Mutex
	seeded        bool
	buffer        []domain.Event
	trimmedUpTo   uint64
	lastPublished uint64
	subscribers   map[*eventSubscriber]struct{}
}

type eventSubscriber struct {
	ch    chan domain.Event
	after uint64
}

func newEventBus() *eventBus {
	return &eventBus{subscribers: make(map[*eventSubscriber]struct{})}
}

// ensureFloor seeds trimmedUpTo from the workspace's actual revision,
// once, on this bus's first real use (H101-71, Stanley's first
// correctness gap). trimmedUpTo already means "a cursor at or below this
// cannot be safely resumed from" for buffer eviction; a fresh process's
// retained history starts at exactly the restored revision too — this
// process has observed nothing before it — so the SAME field and the
// SAME CursorExpired check in subscribe cover both an evicted-by-eviction
// cursor and a stale-by-restart one, without a separate epoch concept.
// Without this, a fresh eventBus's default trimmedUpTo of zero would let
// a numeric cursor cached from BEFORE this process started — e.g. by a
// UI across a host restart — parse fine and be silently accepted, even
// though this process has no record of what happened between that old
// revision and now.
//
// This is lazy rather than done at NewEngine time deliberately: seeding
// eagerly at construction would call store.Load before Open otherwise
// touches the store, so a workspace unreadable for any reason would fail
// at construction instead of at the first real command — a behavior
// change to cmd/harnessing's already-established IOFailure-surfacing
// contract that this card must not make.
func (b *eventBus) ensureFloor(ctx context.Context, load func(context.Context) (uint64, error)) error {
	b.mu.Lock()
	if b.seeded {
		b.mu.Unlock()
		return nil
	}
	b.mu.Unlock()

	revision, err := load(ctx)
	if err != nil {
		return err
	}

	b.mu.Lock()
	defer b.mu.Unlock()
	if !b.seeded {
		b.trimmedUpTo = revision
		b.seeded = true
	}
	return nil
}

// publish appends one commit's events (already stamped with their
// resulting WorkspaceRevision) to the retained buffer and fans them out
// to every live subscriber whose own cursor they are after. A subscriber
// that cannot keep up is disconnected explicitly rather than left to
// stall this call — publish runs on the same path as Commit, and this
// bus must never make a slow observer capable of blocking a write.
func (b *eventBus) publish(events []domain.Event) {
	if len(events) == 0 {
		return
	}
	b.mu.Lock()
	defer b.mu.Unlock()

	// A replayed commit reproduces the SAME WorkspaceRevision as its
	// original attempt — revision only advances on a fresh Mutate,
	// never on a replay (StateStore.Commit's own invariant). FileStore
	// deliberately hands the ORIGINAL event records back on a replay
	// too, after re-confirming durability (H101-70 ruling: those are
	// replay data, not a new publication — the store keeps returning
	// them, the wording that used to claim otherwise was wrong, not
	// the behavior). This is the one place that draws the line for a
	// subscriber: dedup by revision, since a replay's revision is
	// never higher than its original commit's. This is what makes
	// "deduplicate replays" (UI-06) hold regardless of whether a given
	// StateStore implementation replays its original events or not.
	if events[0].WorkspaceRevision <= b.lastPublished {
		return
	}
	b.lastPublished = events[0].WorkspaceRevision

	for _, ev := range events {
		b.buffer = append(b.buffer, cloneEvent(ev))
	}
	for len(b.buffer) > eventBufferCapacity {
		evicted := b.buffer[0]
		b.buffer = b.buffer[1:]
		if evicted.WorkspaceRevision > b.trimmedUpTo {
			b.trimmedUpTo = evicted.WorkspaceRevision
		}
	}

	for sub := range b.subscribers {
		for _, ev := range events {
			if ev.WorkspaceRevision <= sub.after {
				continue
			}
			select {
			case sub.ch <- cloneEvent(ev):
			default:
				// Explicit resumable disconnect (UI-06/ADR 0003 stream
				// semantics), not a blocked send: this subscriber's
				// queue is full, so it stops receiving now rather than
				// stalling this publish. Its channel close is the
				// signal to reconnect with the last cursor it actually
				// processed.
				close(sub.ch)
				delete(b.subscribers, sub)
			}
			if _, stillSubscribed := b.subscribers[sub]; !stillSubscribed {
				break
			}
		}
	}
}

// subscribe returns a channel delivering every retained-or-future event
// with WorkspaceRevision > afterCursor, in commit order, until ctx is
// done. An afterCursor this bus can no longer resume from (unparseable,
// or older than what the buffer has already evicted) is CursorExpired —
// ADR 0003's UI-06 clause this exists to satisfy: "an expired cursor
// treated as an empty one is the failure this requirement exists to
// prevent," so this returns an error instead of silently starting empty.
func (b *eventBus) subscribe(ctx context.Context, afterCursor string) (<-chan domain.Event, error) {
	var after uint64
	if afterCursor != "" {
		v, err := strconv.ParseUint(afterCursor, 10, 64)
		if err != nil {
			return nil, &domain.Error{Code: domain.ErrCursorExpired, Detail: "cursor is not a form this process can resume from"}
		}
		after = v
	}

	b.mu.Lock()
	if after < b.trimmedUpTo {
		b.mu.Unlock()
		return nil, &domain.Error{Code: domain.ErrCursorExpired, Detail: "cursor is older than this process's retained event history"}
	}

	sub := &eventSubscriber{ch: make(chan domain.Event, subscriberQueueCapacity), after: after}
	for _, ev := range b.buffer {
		if ev.WorkspaceRevision > after {
			sub.ch <- cloneEvent(ev)
		}
	}
	b.subscribers[sub] = struct{}{}
	b.mu.Unlock()

	go func() {
		<-ctx.Done()
		b.mu.Lock()
		if _, ok := b.subscribers[sub]; ok {
			delete(b.subscribers, sub)
			close(sub.ch)
		}
		b.mu.Unlock()
	}()

	return sub.ch, nil
}

// cloneEvent detaches SubjectIDs so a subscriber mutating a received
// event cannot reach the buffer's own copy or another subscriber's. Payload
// is left as-is: its shape is per-Kind and undefined at this phase (see
// domain.Event's doc comment), and nothing in this engine sets it yet.
func cloneEvent(ev domain.Event) domain.Event {
	out := ev
	out.SubjectIDs = append([]string{}, ev.SubjectIDs...)
	return out
}
