package transport

import (
	"context"
	"encoding/json"
	"os"
	"sort"
	"time"

	"github.com/rafaelcalves/harnessing-101/internal/api"
	"github.com/rafaelcalves/harnessing-101/internal/core/domain"
)

// DefaultPollInterval is how often Host scans for new sessions and new
// request files. Minimal-card choice, not tuned; a later card may make
// it configurable or event-driven.
const DefaultPollInterval = 100 * time.Millisecond

// Host runs the continuing serve process's whole file-transport side: it
// advertises itself via host.json, discovers session directories clients
// create, binds each one exactly once through Bind, dispatches every
// intake envelope that session writes, and drops the session once a
// detach request arrives. It holds no domain authority itself — Bind is
// the only place authority is granted, and that call is assembly's, not
// this package's.
type Host struct {
	Root       string
	Generation string
	Bind       func(ctx context.Context, attach AttachPayload) (api.FrontendSession, error)
	// Tick is an optional assembly-owned effect pass. Host invokes it from
	// the same serialized poll loop as session dispatch, so callers cannot
	// race a host-owned state effect with transport mutation.
	Tick         func(ctx context.Context) error
	Unbind       func(sessionID string)
	PollInterval time.Duration
}

// Run advertises the host and services sessions until ctx is cancelled.
// It removes its own marker on the way out; it does not attempt to
// detect or clean up a marker left by some other, earlier host (H101-135
// staleness handling is explicitly out of this card's scope).
func (h *Host) Run(ctx context.Context) error {
	interval := h.PollInterval
	if interval <= 0 {
		interval = DefaultPollInterval
	}
	marker, err := json.Marshal(HostMarker{Generation: h.Generation, PID: os.Getpid()})
	if err != nil {
		return &domain.Error{Code: domain.ErrIOFailure, Detail: err.Error()}
	}
	if err := writeAtomic(serveDir(h.Root), hostMarkerPath(h.Root), marker); err != nil {
		return err
	}
	defer func() { _ = removeMarker(h.Root) }()

	bound := map[string]api.FrontendSession{}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			if err := h.tick(ctx, bound); err != nil {
				return err
			}
		}
	}
}

func (h *Host) tick(ctx context.Context, bound map[string]api.FrontendSession) error {
	ids, err := readSubdirNames(sessionsDir(h.Root))
	if err != nil {
		return err
	}
	// deterministic order keeps behavior reproducible across runs/tests
	sort.Strings(ids)
	for _, id := range ids {
		if _, ok := bound[id]; !ok {
			session, done, err := h.tryBind(ctx, id)
			if err != nil || !done {
				continue
			}
			bound[id] = session
		}
		if h.serviceSession(ctx, id, bound) {
			delete(bound, id)
			if h.Unbind != nil {
				h.Unbind(id)
			}
		}
	}
	if h.Tick != nil {
		if err := h.Tick(ctx); err != nil {
			return err
		}
	}
	return nil
}

// tryBind looks for the session's one attach request and, once found,
// calls Bind and answers it. done is false while the client hasn't
// written its attach request yet — not an error, just "not ready".
func (h *Host) tryBind(ctx context.Context, id string) (api.FrontendSession, bool, error) {
	reqPath := requestPath(h.Root, id, attachOperation)
	data, ok, err := readBounded(reqPath)
	if err != nil || !ok {
		return nil, false, err
	}
	var envelope Envelope
	if err := json.Unmarshal(data, &envelope); err != nil {
		return nil, false, nil
	}
	var attach AttachPayload
	if err := json.Unmarshal(envelope.Payload, &attach); err != nil {
		return nil, false, nil
	}
	session, err := h.Bind(ctx, attach)
	if err != nil {
		resp := Response{Error: errorInfo(err)}
		encoded, _ := json.Marshal(resp)
		_ = writeAtomic(responsesDir(h.Root, id), responsePath(h.Root, id, attachOperation), encoded)
		return nil, false, nil
	}
	resp := Response{OK: true}
	encoded, merr := json.Marshal(resp)
	if merr != nil {
		return nil, false, &domain.Error{Code: domain.ErrIOFailure, Detail: merr.Error()}
	}
	if err := writeAtomic(responsesDir(h.Root, id), responsePath(h.Root, id, attachOperation), encoded); err != nil {
		return nil, false, err
	}
	return session, true, nil
}

// serviceSession dispatches every pending request in id's intake except
// attach (already handled by tryBind). It reports true once a detach
// request has been answered, telling tick to drop the session.
func (h *Host) serviceSession(ctx context.Context, id string, bound map[string]api.FrontendSession) bool {
	session := bound[id]
	names, err := readDirFileNames(intakeDir(h.Root, id))
	if err != nil {
		return false
	}
	sort.Strings(names)
	for _, requestID := range names {
		if requestID == attachOperation {
			continue
		}
		if _, already, _ := readBounded(responsePath(h.Root, id, requestID)); already {
			continue
		}
		data, ok, err := readBounded(requestPath(h.Root, id, requestID))
		if err != nil || !ok {
			continue
		}
		var envelope Envelope
		if err := json.Unmarshal(data, &envelope); err != nil {
			continue
		}
		if envelope.Operation == detachOperation {
			resp := Response{OK: true}
			encoded, _ := json.Marshal(resp)
			_ = writeAtomic(responsesDir(h.Root, id), responsePath(h.Root, id, requestID), encoded)
			return true
		}
		resp := Dispatch(ctx, session, envelope)
		encoded, err := json.Marshal(resp)
		if err != nil {
			continue
		}
		_ = writeAtomic(responsesDir(h.Root, id), responsePath(h.Root, id, requestID), encoded)
	}
	return false
}
