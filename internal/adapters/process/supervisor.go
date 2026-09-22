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
	"os"
	"os/exec"
	"path/filepath"
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

	var combined bytes.Buffer
	cmd.Stdout = &combined
	cmd.Stderr = &combined

	if err := cmd.Start(); err != nil {
		return &domain.Error{Code: domain.ErrSpawnFailed, Detail: err.Error()}
	}

	done := make(chan error, 1)
	go func() { done <- cmd.Wait() }()

	select {
	case waitErr := <-done:
		// Fast exit within the startup window: known, not ambiguous.
		// R3's authentication/network/unsupported-mode classification
		// belongs to whatever wrote the descriptor's diagnostic hints
		// (Layer B's runner inspects this product's own relayed
		// output for that); this adapter's own honest classification
		// for a fast, unclassified exit is ErrSpawnFailed.
		if waitErr != nil {
			return &domain.Error{Code: domain.ErrSpawnFailed, Detail: "process exited during startup: " + truncate(combined.String())}
		}
		return &domain.Error{Code: domain.ErrSpawnFailed, Detail: "process exited immediately with no participation signal: " + truncate(combined.String())}
	case <-time.After(StartupWindow):
		// Still running past the bounded window: a healthy long-lived
		// participant. Leave it running, unowned by this call.
		return nil
	}
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

// Capabilities, Observe, Stop, and Recover are items 2/4/5's scope —
// honest Unsupported here, not a stub that pretends to observe or stop
// anything this card never built.
func (s *Supervisor) Capabilities(ctx context.Context) (any, error) {
	return nil, &domain.Error{Code: domain.ErrUnsupported, Detail: "Capabilities is not implemented in this Phase 3 slice"}
}

func (s *Supervisor) Observe(ctx context.Context, runID domain.RunID) (<-chan any, error) {
	return nil, &domain.Error{Code: domain.ErrUnsupported, Detail: "Observe is not implemented in this Phase 3 slice"}
}

func (s *Supervisor) Stop(ctx context.Context, runID domain.RunID, grace time.Duration) error {
	return &domain.Error{Code: domain.ErrUnsupported, Detail: "Stop is not implemented in this Phase 3 slice"}
}

func (s *Supervisor) Recover(ctx context.Context, runID domain.RunID) error {
	return &domain.Error{Code: domain.ErrUnsupported, Detail: "Recover is not implemented in this Phase 3 slice"}
}

var _ ports.ProcessSupervisor = (*Supervisor)(nil)
