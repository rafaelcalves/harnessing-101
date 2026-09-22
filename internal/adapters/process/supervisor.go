// Package process is the Phase 3 item 1 implementation of
// ports.ProcessSupervisor (outbound port 7): the adapter that actually
// spawns a local executable named by an approved ExecutionSpec. It
// resolves paths, environment, and I/O; the core never inspects a
// process ID (boundaries.md "supervision and recovery"). This slice
// implements Start only — Observe/Stop/Recover belong to items 2/4/5
// and return Unsupported here, honestly, rather than a stub that lies.
package process

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"syscall"
	"time"

	"github.com/rafaelcalves/harnessing-101/internal/core/domain"
	"github.com/rafaelcalves/harnessing-101/internal/core/ports"
)

// StartupWindow bounds how long Start waits, after spawning, to decide
// whether the launched tool is a healthy long-lived participant or one
// that failed fast (R3: missing auth, unsupported invocation, spawn
// failure). It is a package variable, not a constant, so a test can
// shrink it — Kelly's own bounded-observation-timeout precedent (item 3
// E3) for exactly this reason: real-time latency in a test must be
// bounded, not tuned by trial and error against a fixed constant.
var StartupWindow = 500 * time.Millisecond

// Supervisor is the concrete adapter. contextDir is where per-run
// context-file artifacts are written (R2); it is typically the
// workspace root's own run-scoped subdirectory, supplied by whoever
// constructs this (host composition), never invented here from a
// caller-supplied path.
type Supervisor struct {
	ContextDir string
	Journal    ports.OutputJournal
	// LifecycleObserver is a test-only ordered-event hook (H101-217's
	// "injectable hook, build-tag test helper, or equivalent"). Nil in
	// production. It never influences behavior -- only records it --
	// so a test can assert Creed's ordering points without changing
	// what actually happens.
	LifecycleObserver func(runID domain.RunID, event string)
	mu                sync.Mutex
	processes         map[domain.RunID]*runRecord
}

// runRecord is the one adapter-owned lifecycle record h101-215
// requires: the exclusive owner of signaling authority and of the
// retained, unreaped leader for one run. Every signal, every
// authority-close, and the one final reap are serialized through
// mu -- never through the package-level Supervisor.mu, which only
// protects the processes map itself.
type runRecord struct {
	mu          sync.Mutex
	pid         int
	stopClaimed bool
	// closed is irrevocable: once true, no further signal may be sent
	// to pid or its group under this record, by any caller, ever.
	closed bool
	reaped bool
	// done and result are H101-222 point 5 / H101-224 B7: closed alone
	// is not success. done is closed exactly once, by whichever path
	// (Stop's forced sequence or natural finalization) reaches the
	// run's actual terminal outcome; result is only meaningful to read
	// after done is observed closed -- the close itself is the
	// synchronization point (Go memory model: a receive that observes
	// a channel closed happens after every write that preceded the
	// close), so a concurrent waiter never needs rec.mu to read it.
	done   chan struct{}
	result error
	// terminatedObserved guards leader_terminated_observed so it fires
	// exactly once regardless of which of Stop or the natural observer
	// detects the leader's exit first -- both genuinely observe it,
	// racing each other, and a test recorder should see one
	// deterministic event rather than a coin flip on which goroutine's
	// poll iteration happened to land first.
	terminatedObserved bool
}

// observeLeaderTerminatedOnce records leader_terminated_observed the
// first time either Stop or the natural path detects the leader's
// exit, and is a silent no-op on every later call.
func (s *Supervisor) observeLeaderTerminatedOnce(runID domain.RunID, rec *runRecord) {
	rec.mu.Lock()
	already := rec.terminatedObserved
	rec.terminatedObserved = true
	rec.mu.Unlock()
	if !already {
		s.observe(runID, "leader_terminated_observed")
	}
}

func (s *Supervisor) observe(runID domain.RunID, event string) {
	if s.LifecycleObserver != nil {
		s.LifecycleObserver(runID, event)
	}
}

// finalize records rec's one terminal outcome and wakes every current
// and future waiter. Only the winner of the stopClaimed/closed gating
// upstream ever reaches this call for a given rec, so it is safe to
// write result and close done without additional locking here.
func finalize(rec *runRecord, err error) error {
	rec.result = err
	close(rec.done)
	return err
}

// contextFile is R2's "documented protocol handoff" payload: the only
// identifiers a launched tool receives via ContextTransport
// "context-file". Field names are stable and intentionally narrow —
// there is no path for arbitrary request text to ride along as
// "context."
type contextFile struct {
	SchemaVersion int                `json:"schemaVersion"`
	WorkspaceID   domain.WorkspaceID `json:"workspaceId"`
	RunID         domain.RunID       `json:"runId"`
	AgentID       domain.AgentID     `json:"agentId"`
	TaskID        *domain.TaskID     `json:"taskId,omitempty"`
	PeerAgentID   *domain.AgentID    `json:"peerAgentId,omitempty"`
}

// Start spawns spec.Args[0] with the remaining elements as arguments.
// It classifies failure into the R3 taxonomy rather than returning a
// bare os/exec error: LookPath failure is ErrMissingTool; an
// unsupported ContextTransport value is ErrUnsupported (never silently
// ignored); any other spawn-time failure is ErrSpawnFailed. Once
// spawned, Start waits up to StartupWindow to see whether the process
// exits fast (capturing its combined output for the caller's Detail —
// the same signal a Layer B runner's diagnostic hints inspect) or is
// still running, in which case Start returns nil and the process
// continues independently: Start does not block for the tool's whole
// lifetime, and this package does not yet track or own that process
// beyond this call (items 2/4 add ownership/termination).
func (s *Supervisor) Start(ctx context.Context, runID domain.RunID, spec domain.ExecutionSpec, participation domain.RunParticipationContext) error {
	if len(spec.Args) == 0 {
		return &domain.Error{Code: domain.ErrInvalidArgument, Detail: "execution spec has no executable"}
	}
	executable, argv := spec.Args[0], spec.Args[1:]

	if _, err := exec.LookPath(executable); err != nil {
		return &domain.Error{Code: domain.ErrMissingTool, Detail: executable + " is not on PATH or not executable"}
	}

	cmd := exec.CommandContext(context.Background(), executable, argv...) //nolint:gocritic // detached from ctx deliberately: Start's own caller-cancellation must not kill an already-spawned tool.
	// Every managed child starts a new process group. Stop addresses this
	// recorded, adapter-owned group rather than a caller-supplied PID.
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	if spec.WorkingDirectory != "" {
		cmd.Dir = spec.WorkingDirectory
	}
	cmd.Env = os.Environ()

	switch spec.ContextTransport {
	case "":
		// No participation context requested — a plain spawn.
	case "context-file":
		path, err := s.writeContextFile(runID, participation)
		if err != nil {
			return &domain.Error{Code: domain.ErrSpawnFailed, Detail: "could not write participation context file: " + err.Error()}
		}
		cmd.Env = append(cmd.Env, "HARNESSING_CONTEXT_FILE="+path)
	default:
		// R3: an unsupported transport value is a stable, explicit
		// Unsupported — never a silent no-op that pretends R2 was
		// satisfied.
		return &domain.Error{Code: domain.ErrUnsupported, Detail: "context transport " + spec.ContextTransport + " is not implemented"}
	}

	// H101-222's capture/run separation starts here: cmd.Stdout/Stderr
	// are OUR OWN pipe write-ends (plain *os.File), not a bare
	// io.Writer. os/exec special-cases an *os.File target -- it spawns
	// no internal copy goroutine and never touches it in Wait() -- so
	// draining is entirely ours to observe, independent of reaping
	// (reapLeader, a bare syscall.Wait4) or of the leader's own group
	// membership. Capture completes purely from pipe EOF; run
	// completion is a separate fact decided by observeNaturalExit/Stop.
	stdoutR, stdoutW, err := os.Pipe()
	if err != nil {
		return &domain.Error{Code: domain.ErrSpawnFailed, Detail: "creating stdout pipe: " + err.Error()}
	}
	stderrR, stderrW, err := os.Pipe()
	if err != nil {
		_ = stdoutR.Close()
		_ = stdoutW.Close()
		return &domain.Error{Code: domain.ErrSpawnFailed, Detail: "creating stderr pipe: " + err.Error()}
	}
	cmd.Stdout = stdoutW
	cmd.Stderr = stderrW

	if err := cmd.Start(); err != nil {
		_ = stdoutR.Close()
		_ = stdoutW.Close()
		_ = stderrR.Close()
		_ = stderrW.Close()
		return &domain.Error{Code: domain.ErrSpawnFailed, Detail: err.Error()}
	}
	// Our own copies of the write ends must close now: the child (and
	// any descendant that inherited them) holds the real copies. If we
	// kept ours open, our own read-side drain below would never see
	// EOF even after every child-held copy closed.
	_ = stdoutW.Close()
	_ = stderrW.Close()

	rec := &runRecord{pid: cmd.Process.Pid, done: make(chan struct{})}
	s.mu.Lock()
	if s.processes == nil {
		s.processes = make(map[domain.RunID]*runRecord)
	}
	s.processes[runID] = rec
	s.mu.Unlock()

	var combined bytes.Buffer
	combinedMu := &sync.Mutex{}
	s.drainAndCapture(ctx, runID, &combined, combinedMu, stdoutR, stderrR)

	// h101-215's trap: the leader is retained UNREAPED from the
	// moment it is spawned. There is no unconditional cmd.Wait() here
	// -- that call reaps unconditionally, which would defeat the
	// whole identity-binding mechanism before Stop ever runs. Instead
	// every exit observation below goes through waitidPeek (WNOWAIT),
	// which never consumes the child's status.
	deadline := time.Now().Add(StartupWindow)
	for {
		exited, peekErr := waitidPeek(rec.pid)
		if peekErr != nil {
			// The retain-unreaped binding cannot be observed on this
			// target. Honest, unclassified failure -- never a silent
			// assumption that the process is fine.
			s.forget(runID, rec)
			return &domain.Error{Code: domain.ErrSpawnFailed, Detail: "observing process exit: " + peekErr.Error()}
		}
		if exited {
			// Fast exit within the startup window: known, not
			// ambiguous. R3's authentication/network/unsupported-mode
			// classification belongs to whatever wrote the
			// descriptor's diagnostic hints (Layer B's runner
			// inspects this product's own relayed output for that);
			// this adapter's own honest classification for a fast,
			// unclassified exit is ErrSpawnFailed. The leader is
			// reaped here, synchronously, because a run that never
			// leaves Starting never becomes Stop-eligible -- no
			// signaling window is ever opened for it, so retaining it
			// further would only leak a zombie.
			ws, reapErr := reapLeader(rec.pid)
			s.forget(runID, rec)
			combinedMu.Lock()
			combinedSnapshot := combined.String()
			combinedMu.Unlock()
			detail := "process exited during startup: " + truncate(combinedSnapshot)
			if reapErr == nil && ws.ExitStatus() == 0 && !ws.Signaled() {
				detail = "process exited immediately with no participation signal: " + truncate(combinedSnapshot)
			}
			return &domain.Error{Code: domain.ErrSpawnFailed, Detail: detail}
		}
		if time.Now().After(deadline) {
			break
		}
		time.Sleep(pollInterval)
	}

	// Still running past the bounded window: a healthy long-lived
	// participant. Leave it running, retained-unreaped for Stop's
	// benefit — but hand its eventual natural exit to a background
	// observer so it is not left as a zombie forever if no Stop is
	// ever called (H101-222/H101-224). That observer decides RUN
	// completion only; it no longer touches capture at all -- capture
	// (drainAndCapture, started above) is driven purely by pipe EOF and
	// publishes Complete/Interrupted independently of whether the
	// leader has been reaped or the group is empty (H101-195's serve-
	// ownership of graceful capture: this drain only ever gets to see
	// EOF while the PROCESS running Start outlives the descendants
	// holding those pipes long enough to observe it -- true for a
	// continuing `harnessing serve` host, never true for a one-shot CLI
	// command). A mode-B worker holding its inherited pipes open keeps
	// capture incomplete even after the leader exits; a worker that
	// closes them lets capture complete while the run, and Stop's
	// binding, both remain exactly as retained as before.
	go s.observeNaturalExit(runID, rec)
	return nil
}

// drainAndCapture owns stdout/stderr drain completion and append
// failures explicitly, entirely independent of reaping (H101-222):
// each pipe read-end is drained by its own goroutine until EOF or a
// real read error, and only once BOTH finish does this report
// Capture complete (clean EOF, every append durable) or Interrupted
// (a genuine drain/append failure) via Journal.Finish. No leader exit
// status is consulted -- capture completion and run completion are
// now two separate facts, on purpose.
func (s *Supervisor) drainAndCapture(ctx context.Context, runID domain.RunID, combined *bytes.Buffer, combinedMu *sync.Mutex, stdoutR, stderrR *os.File) {
	var wg sync.WaitGroup
	var errMu sync.Mutex
	var drainErr error
	copyOne := func(r *os.File, channel string) {
		defer wg.Done()
		defer func() { _ = r.Close() }()
		w := journalWriter{ctx: ctx, journal: s.Journal, runID: runID, channel: channel}
		// bytes.Buffer is not safe for concurrent use, and the stdout
		// and stderr goroutines both write into the same combined
		// buffer (plus Start's own fast-exit branch reads it) -- every
		// write to it must go through the shared lock, not just the
		// call that builds the io.Writer chain.
		dst := lockedMultiWriter{mu: combinedMu, w: io.MultiWriter(combined, w)}
		if _, err := io.Copy(dst, r); err != nil {
			errMu.Lock()
			if drainErr == nil {
				drainErr = err
			}
			errMu.Unlock()
		}
	}
	wg.Add(2)
	go copyOne(stdoutR, "stdout")
	go copyOne(stderrR, "stderr")
	go func() {
		wg.Wait()
		if s.Journal == nil {
			return
		}
		errMu.Lock()
		failed := drainErr != nil
		errMu.Unlock()
		status := domain.CaptureComplete
		if failed {
			status = domain.CaptureInterrupted
		}
		_ = s.Journal.Finish(context.Background(), runID, status)
	}()
}

// lockedMultiWriter serializes writes into a shared *bytes.Buffer
// (not itself concurrency-safe) from two independent drain goroutines
// (stdout/stderr) that may otherwise race on it.
type lockedMultiWriter struct {
	mu *sync.Mutex
	w  io.Writer
}

func (l lockedMultiWriter) Write(p []byte) (int, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.w.Write(p)
}

// observeNaturalExit is H101-222/H101-224's natural-completion path: a
// run that finishes with nobody ever calling Stop. Reaping on leader
// termination ALONE was the defect this replaces (H101-222): a mode-B
// worker surviving the leader would then be orphaned, unsignalable,
// forever. This now only reaps once it has POSITIVE evidence -- not a
// boolean group probe, which a retained zombie leader keeps positive
// on its own -- that no OTHER member remains. Where that evidence is
// unavailable, it retains the binding and exposes the uncertainty
// rather than reaping speculatively (point 3). It shares rec's single
// serialized record with Stop and steps aside the instant Stop claims
// it (point 4): the loser of that race does no work at all.
func (s *Supervisor) observeNaturalExit(runID domain.RunID, rec *runRecord) {
	for {
		exited, err := waitidPeek(rec.pid)
		if err != nil {
			return
		}
		if exited {
			break
		}
		time.Sleep(pollInterval)
	}
	s.observeLeaderTerminatedOnce(runID, rec)

	for {
		rec.mu.Lock()
		claimed := rec.closed || rec.stopClaimed
		rec.mu.Unlock()
		if claimed {
			return
		}

		confirmed, uncertain := confirmOtherMembersAbsent(rec.pid, rec.pid)
		if uncertain {
			s.observe(runID, "other_members_uncertain")
			s.observe(runID, "natural_reap_refused_uncertain")
			return
		}
		if confirmed {
			s.observe(runID, "other_members_absent_confirmed")
			break
		}
		// A live other member (the mode-B survivor) still exists --
		// not uncertain, just not yet safe. Keep checking; Stop may
		// claim the record on any later iteration.
		time.Sleep(membershipRecheckInterval)
	}

	rec.mu.Lock()
	if rec.closed || rec.stopClaimed {
		rec.mu.Unlock()
		return
	}
	rec.closed = true
	s.observe(runID, "authority_closed")
	_, reapErr := reapLeader(rec.pid)
	rec.reaped = true
	rec.mu.Unlock()
	s.observe(runID, "child_reaped")
	s.forget(runID, rec)

	// Confirm group disappearance before reporting run completion
	// (point 4). At this point the leader is already reaped, so a
	// positive probe means a REAL remaining member, not the retained
	// zombie's own artifact.
	if !waitGroupGone(rec.pid, 30*time.Second) {
		_ = finalize(rec, &domain.Error{Code: domain.ErrRecoveryRequired, Detail: "process group did not fully disappear after natural termination"})
		return
	}
	if reapErr != nil {
		_ = finalize(rec, &domain.Error{Code: domain.ErrRecoveryRequired, Detail: "reaping leader after natural termination: " + reapErr.Error()})
		return
	}
	_ = finalize(rec, nil)
}

// closeAndReap is h101-215's "close signaling authority BEFORE
// consuming the final child status, both under the same
// serialization." Stop's own forced-signal path is the only caller
// left: the fast-exit branch in Start and observeNaturalExit each
// close/reap inline now, since each has its own distinct evidence
// (never opening a signaling window, and confirmed other-member
// absence, respectively) that this shared helper doesn't need to
// re-derive. didReap is false if the natural path (or a stale
// concurrent call) already closed rec first.
func (s *Supervisor) closeAndReap(rec *runRecord) (ws syscall.WaitStatus, reapErr error, didReap bool) {
	rec.mu.Lock()
	defer rec.mu.Unlock()
	if rec.closed {
		return ws, nil, false
	}
	rec.closed = true
	ws, reapErr = reapLeader(rec.pid)
	rec.reaped = true
	return ws, reapErr, true
}

type journalWriter struct {
	ctx     context.Context
	journal ports.OutputJournal
	runID   domain.RunID
	channel string
}

func (w journalWriter) Write(p []byte) (int, error) {
	if w.journal == nil {
		return len(p), nil
	}
	if _, err := w.journal.Append(w.ctx, w.runID, w.channel, append([]byte(nil), p...), time.Now()); err != nil {
		return 0, err
	}
	return len(p), nil
}

func (s *Supervisor) writeContextFile(runID domain.RunID, participation domain.RunParticipationContext) (string, error) {
	dir := s.ContextDir
	if dir == "" {
		dir = os.TempDir()
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	payload := contextFile{
		SchemaVersion: 1,
		WorkspaceID:   participation.WorkspaceID,
		RunID:         runID,
		AgentID:       participation.AgentID,
		TaskID:        participation.TaskID,
		PeerAgentID:   participation.PeerAgentID,
	}
	data, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}
	path := filepath.Join(dir, string(runID)+".context.json")
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return "", err
	}
	return path, nil
}

const maxDetailBytes = 4096

func truncate(s string) string {
	if len(s) <= maxDetailBytes {
		return s
	}
	return s[len(s)-maxDetailBytes:]
}

// RunInfo is test-only exposure of the retained leader's pid — never
// called from product code, which the boundary comment at the top of
// this file already forbids (the core never inspects a process ID).
// It exists so a native test can shape a mode-B survivor (a real
// process placed into the leader's own pgid) without reaching into
// runRecord directly from _test.go, which is unexported.
type RunInfo struct {
	Pid int
}

func (s *Supervisor) RunInfo(runID domain.RunID) (RunInfo, error) {
	s.mu.Lock()
	rec := s.processes[runID]
	s.mu.Unlock()
	if rec == nil {
		return RunInfo{}, &domain.Error{Code: domain.ErrNotFound, Detail: "run has no owned process record"}
	}
	return RunInfo{Pid: rec.pid}, nil
}

// Capabilities, Observe, Stop, and Recover are items 2/4/5's scope —
// honest Unsupported here, not a stub that pretends to observe or stop
// anything this card never built.
func (s *Supervisor) Capabilities(ctx context.Context) (any, error) {
	return nil, &domain.Error{Code: domain.ErrUnsupported, Detail: "Capabilities is not implemented in this Phase 3 slice"}
}

func (s *Supervisor) Observe(ctx context.Context, runID domain.RunID) (<-chan any, error) {
	return nil, &domain.Error{Code: domain.ErrUnsupported, Detail: "Observe is not implemented in this Phase 3 slice"}
}

// Stop is h101-215's identity binding in force: it never signals a
// bare pgid number, only the group led by rec.pid -- a leader THIS
// call retained unreaped since Start, so that number cannot have been
// reused by an unrelated process for as long as rec.closed stays
// false. Every signal is rechecked against rec.closed immediately
// before it is sent (Creed point 1); authority is closed before the
// one reap, both under rec.mu (Creed point 2); no signal is ever sent
// after rec.closed is true (Creed point 3); a bounded grace period is
// honored even though a retained zombie can keep the group probe
// positive throughout (Creed point 4); and any failure to establish
// or validate the binding returns RecoveryRequired, never a
// best-effort signal to the bare number (Creed point 5).
func (s *Supervisor) Stop(ctx context.Context, runID domain.RunID, grace time.Duration) error {
	s.mu.Lock()
	rec := s.processes[runID]
	s.mu.Unlock()
	if rec == nil {
		return &domain.Error{Code: domain.ErrNotFound, Detail: "run has no owned process record"}
	}
	if grace <= 0 {
		grace = 2 * time.Second
	}

	rec.mu.Lock()
	if rec.closed || rec.stopClaimed {
		// Either the natural path already finalized this run, or
		// another Stop call already claimed it. Either way, this call
		// sends NO signal of its own (Creed point 3's "no stale/queued
		// signal" extends to "no redundant one either") and only
		// awaits the actual terminal result -- H101-222 point 5: a
		// closed record is not itself a successful stop.
		rec.mu.Unlock()
		return s.awaitResult(ctx, rec)
	}
	rec.stopClaimed = true
	s.observe(runID, "stop_claimed")
	s.observe(runID, "pre_sigterm_check")
	if err := syscall.Kill(-rec.pid, syscall.SIGTERM); err != nil && !isBenignGroupSignalError(err) {
		rec.mu.Unlock()
		return finalize(rec, &domain.Error{Code: domain.ErrSpawnFailed, Detail: "sending SIGTERM to process group: " + err.Error()})
	}
	s.observe(runID, "sigterm_sent")
	rec.mu.Unlock()

	// Poll for early graceful death instead of blindly sleeping the
	// whole grace period: h101-222 point 4 forbids the natural
	// observer from finishing Stop's work on its behalf once claimed
	// (it defers to Stop entirely, checked via rec.stopClaimed), so
	// Stop must notice a leader that already died to SIGTERM itself,
	// or it would needlessly escalate every graceful stop to SIGKILL.
	alreadyExited := waitLeaderExitedWithinGrace(ctx, rec.pid, grace)
	if alreadyExited {
		s.observeLeaderTerminatedOnce(runID, rec)
	}

	rec.mu.Lock()
	if rec.closed {
		// The natural path finalized this run while Stop was waiting
		// out the grace period -- its outcome is the run's one
		// terminal result; report that, not a fabricated success.
		rec.mu.Unlock()
		return s.awaitResult(ctx, rec)
	}
	if !alreadyExited {
		s.observe(runID, "pre_sigkill_check")
		if err := syscall.Kill(-rec.pid, syscall.SIGKILL); err != nil && !isBenignGroupSignalError(err) {
			rec.mu.Unlock()
			return finalize(rec, &domain.Error{Code: domain.ErrRecoveryRequired, Detail: "sending SIGKILL to process group: " + err.Error()})
		}
		s.observe(runID, "sigkill_sent")
	}
	rec.mu.Unlock()

	// Never wait forever for the group probe to go false -- a
	// retained zombie leader keeps it positive on its own. Bound the
	// wait on the LEADER's own exit status instead, which this
	// adapter can observe reliably because it is our own retained
	// child.
	if !waitLeaderExited(rec.pid, 30*time.Second) {
		return finalize(rec, &domain.Error{Code: domain.ErrRecoveryRequired, Detail: "leader did not report exit within the bounded observation window"})
	}
	s.observeLeaderTerminatedOnce(runID, rec)

	_, reapErr, didReap := s.closeAndReap(rec)
	if !didReap {
		// The natural path somehow won despite stopClaimed being set
		// (it checks that flag before claiming) -- report its already-
		// decided outcome rather than doing anything further.
		return s.awaitResult(ctx, rec)
	}
	s.observe(runID, "authority_closed")
	s.observe(runID, "child_reaped")
	s.forget(runID, rec)

	// Only after the leader itself is closed and reaped does a
	// positive group probe mean anything: any other member (e.g. a
	// surviving worker) that has not yet disappeared, not the
	// retained leader's own zombie slot artifact.
	if !waitGroupGone(rec.pid, 30*time.Second) {
		return finalize(rec, &domain.Error{Code: domain.ErrRecoveryRequired, Detail: "process group did not fully disappear after termination"})
	}
	if reapErr != nil {
		return finalize(rec, &domain.Error{Code: domain.ErrRecoveryRequired, Detail: "reaping leader after termination: " + reapErr.Error()})
	}
	return finalize(rec, nil)
}

// awaitResult is what every caller other than the one that actually
// claims rec's termination sequence uses to learn the outcome: it
// never infers success from rec.closed alone (H101-222 point 5 / B7),
// only from the recorded terminal result once rec.done is observably
// closed.
func (s *Supervisor) awaitResult(ctx context.Context, rec *runRecord) error {
	select {
	case <-rec.done:
		return rec.result
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(35 * time.Second):
		return &domain.Error{Code: domain.ErrRecoveryRequired, Detail: "concurrent Stop did not observe termination within the bounded window"}
	}
}

func (s *Supervisor) forget(runID domain.RunID, rec *runRecord) {
	s.mu.Lock()
	if s.processes[runID] == rec {
		delete(s.processes, runID)
	}
	s.mu.Unlock()
}

// Recover is item 4's minimal slice (H101-170): an honest,
// conservative first answer. This adapter never probes a PID or
// adopts a process by reused identifier — boundaries.md is explicit
// that a reused PID alone is insufficient — so it always reports
// ownership/outcome as Unknown via the stable RecoveryRequired code.
// The core (never this adapter) durably maps that Unknown into the
// run's own RecoveryRequired state; a future stronger platform
// identity mechanism may narrow this, but is not required to
// manufacture certainty now.
func (s *Supervisor) Recover(ctx context.Context, runID domain.RunID) error {
	return &domain.Error{Code: domain.ErrRecoveryRequired, Detail: "process ownership/outcome cannot be established in this Phase 3 slice (no PID or identity adoption)"}
}

var _ ports.ProcessSupervisor = (*Supervisor)(nil)
