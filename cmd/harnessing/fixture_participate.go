package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"time"
)

// runFixtureParticipate is Phase 3 item 1's Layer A CI participation
// fixture (H101-147: "a minimal approved profile pointing at a test
// binary/script that ... receives injected task/workspace context via
// the profile's documented mechanism, [and] emits an observable
// participation signal"). It is not a sleep/cat stand-in (D9): it
// actually reads HARNESSING_CONTEXT_FILE, the same context-file
// ProcessSupervisor.Start writes for any real profile using that
// transport, and writes a sibling ".participated" marker recording what
// it read — the observable signal a native test asserts on directly,
// no different in kind from what a real agentic CLI's own protocol
// handoff would leave behind.
//
// HARNESSING_FIXTURE_SIMULATE, when set, makes this fixture exit fast
// with a specific classification instead of participating — the R3
// negative-path half of the same fixture (D10): "missing_tool" is
// exercised by pointing a profile at a nonexistent executable instead
// (this fixture is never reached), but "auth_required" here proves a
// fast, participating-looking process that still fails honestly is not
// reported as Running.
func runFixtureParticipate(args []string, stdout, stderr io.Writer) int {
	if sim := os.Getenv("HARNESSING_FIXTURE_SIMULATE"); sim != "" {
		if sim == "block_after_marker" {
			// Supervisor captures child stdout internally, so the native CLI
			// crash proof uses this explicit external sync artifact while
			// retaining the same start-called stdout convention.
			_, _ = fmt.Fprintln(stdout, "start-called")
			syncPath := os.Getenv("HARNESSING_FIXTURE_SYNC_FILE")
			if syncPath == "" {
				_, _ = fmt.Fprintln(stderr, "harnessing fixture: HARNESSING_FIXTURE_SYNC_FILE is not set")
				return 1
			}
			if err := os.WriteFile(syncPath, []byte(fmt.Sprintf("start-called\npid=%d\n", os.Getpid())), 0o600); err != nil {
				_, _ = fmt.Fprintln(stderr, "harnessing fixture: writing sync file: "+err.Error())
				return 1
			}
			for {
				time.Sleep(time.Hour)
			}
		}
		_, _ = fmt.Fprintln(stderr, "harnessing fixture: simulated failure: "+sim)
		return 1
	}
	outputMode := os.Getenv("HARNESSING_FIXTURE_OUTPUT_MODE")
	if outputMode == "worker_block" {
		// H101-221: the parent's own worker_pid marker is written
		// immediately after fork, before this process has done
		// anything -- a routing hint only, never proof of a live
		// worker. This exact 13-byte attestation, appended by the
		// worker ITSELF only once it has actually reached its
		// blocking state, is what a test may treat as worker-alive
		// evidence, always together with a succeeding PID probe,
		// never either alone.
		if err := appendWorkerReady(); err != nil {
			_, _ = fmt.Fprintln(stderr, "harnessing fixture: appending worker_ready: "+err.Error())
			return 1
		}
		// H101-224: the cheapest possible shape for Stanley's
		// pipe-holding versus pipe-closing distinction. "hold" (the
		// default) is the existing behavior -- keep the inherited
		// stdin/stdout/stderr open, so the adapter's copy goroutines
		// never see EOF while this worker lives. "close" additionally
		// closes all three inherited descriptors here, so those same
		// copy goroutines DO see EOF (capture may complete) even
		// though this process, and the run, are still alive.
		if os.Getenv("HARNESSING_FIXTURE_WORKER_PIPE") == "close" {
			_ = os.Stdin.Close()
			_ = os.Stdout.Close()
			_ = os.Stderr.Close()
		}
		for {
			time.Sleep(time.Hour)
		}
	}
	if outputMode == "fast_parent_idle_worker" {
		worker, err := startFixtureWorker()
		if err != nil {
			_, _ = fmt.Fprintln(stderr, "harnessing fixture: spawning worker: "+err.Error())
			return 1
		}
		if syncPath := os.Getenv("HARNESSING_FIXTURE_SYNC_FILE"); syncPath != "" {
			marker := fmt.Sprintf("participated\npid=%d\nworker_pid=%d\n", os.Getpid(), worker.Pid)
			if err := os.WriteFile(syncPath, []byte(marker), 0o600); err != nil {
				_, _ = fmt.Fprintln(stderr, "harnessing fixture: writing sync file: "+err.Error())
				return 1
			}
		}
		return 0
	}

	contextPath := os.Getenv("HARNESSING_CONTEXT_FILE")
	if contextPath == "" {
		_, _ = fmt.Fprintln(stderr, "harnessing fixture: HARNESSING_CONTEXT_FILE is not set; nothing to participate with")
		return 1
	}
	data, err := os.ReadFile(contextPath)
	if err != nil {
		_, _ = fmt.Fprintln(stderr, "harnessing fixture: reading context file: "+err.Error())
		return 1
	}
	var parsed map[string]any
	if err := json.Unmarshal(data, &parsed); err != nil {
		_, _ = fmt.Fprintln(stderr, "harnessing fixture: context file is not valid JSON: "+err.Error())
		return 1
	}
	if err := os.WriteFile(contextPath+".participated", data, 0o644); err != nil {
		_, _ = fmt.Fprintln(stderr, "harnessing fixture: writing participation marker: "+err.Error())
		return 1
	}
	if outputMode != "serve_release_both_channels" {
		// Suppressed for serve_release_both_channels only: item 5's I3/
		// I4/I5 evidence needs the two named channel literals to be the
		// ONLY content this fixture puts on stdout/stderr (besides the
		// disclosure banner every command prints), so an offset
		// computed from known constant lengths lands exactly on a real
		// chunk boundary rather than guessing around an extra line.
		_, _ = fmt.Fprintln(stdout, "harnessing fixture: participated")
	}
	if outputMode == "prefix_then_block" {
		_, _ = fmt.Fprintln(stdout, "fixture-out-1")
	}
	if outputMode == "spawn_worker_then_block" || outputMode == "parent_exits_worker_survives" {
		worker, err := startFixtureWorker()
		if err != nil {
			_, _ = fmt.Fprintln(stderr, "harnessing fixture: spawning worker: "+err.Error())
			return 1
		}
		if syncPath := os.Getenv("HARNESSING_FIXTURE_SYNC_FILE"); syncPath != "" {
			marker := fmt.Sprintf("participated\npid=%d\nworker_pid=%d\n", os.Getpid(), worker.Pid)
			if err := os.WriteFile(syncPath, []byte(marker), 0o600); err != nil {
				_, _ = fmt.Fprintln(stderr, "harnessing fixture: writing sync file: "+err.Error())
				return 1
			}
		}
		if outputMode == "parent_exits_worker_survives" {
			return waitForParentRelease(stderr)
		}
	}
	if syncPath := os.Getenv("HARNESSING_FIXTURE_SYNC_FILE"); syncPath != "" && outputMode != "spawn_worker_then_block" && outputMode != "parent_exits_worker_survives" {
		marker := fmt.Sprintf("participated\npid=%d\n", os.Getpid())
		if outputMode == "prefix_then_block" {
			marker += "prefix-ready\n"
		}
		if err := os.WriteFile(syncPath, []byte(marker), 0o600); err != nil {
			_, _ = fmt.Fprintln(stderr, "harnessing fixture: writing sync file: "+err.Error())
			return 1
		}
	}
	if outputMode == "prefix_then_block" {
		for {
			time.Sleep(time.Hour)
		}
	}

	if outputMode == "serve_release_both_channels" {
		return runFixtureServeRelease(stdout, stderr)
	}

	sleep := 3 * time.Second
	if raw := os.Getenv("HARNESSING_FIXTURE_SLEEP_MS"); raw != "" {
		var ms int
		if _, err := fmt.Sscanf(raw, "%d", &ms); err == nil {
			sleep = time.Duration(ms) * time.Millisecond
		}
	}
	time.Sleep(sleep)
	return 0
}

func waitForParentRelease(stderr io.Writer) int {
	releasePath := os.Getenv("HARNESSING_FIXTURE_PARENT_RELEASE_FILE")
	if releasePath == "" {
		_, _ = fmt.Fprintln(stderr, "harnessing fixture: HARNESSING_FIXTURE_PARENT_RELEASE_FILE is not set")
		return 1
	}
	deadline := time.Now().Add(30 * time.Second)
	for {
		data, err := os.ReadFile(releasePath)
		if err == nil && string(data) == "parent-release\n" {
			return 0
		}
		if time.Now().After(deadline) {
			_, _ = fmt.Fprintln(stderr, "harnessing fixture: no valid parent release within 30s")
			return 1
		}
		time.Sleep(50 * time.Millisecond)
	}
}

func startFixtureWorker() (*os.Process, error) {
	cmd := exec.Command(os.Args[0], "__fixture-participate")
	cmd.Env = append(os.Environ(), "HARNESSING_FIXTURE_OUTPUT_MODE=worker_block")
	if err := cmd.Start(); err != nil {
		return nil, err
	}
	return cmd.Process, nil
}

// workerReadyToken is H101-221's exact 13-byte worker-written
// attestation. Not the parent's own worker_pid routing hint (written
// immediately after fork, before the worker has done anything) — this
// is written by the worker process itself, only once it has actually
// reached its blocking state, and is the ONLY thing a test may treat
// as worker-alive evidence, always together with a succeeding PID
// probe, never either alone.
const workerReadyToken = "worker_ready\n"

// appendWorkerReady waits for the PARENT's own routing-hint write to
// land before appending. The parent's marker write (os.WriteFile,
// which truncates) and this worker's own process start are two
// independent processes racing after the same fork -- appending
// before the parent's write lands would have that later truncating
// write silently erase this attestation. Polling for the parent's
// known "worker_pid=" substring first makes the ordering
// deterministic without needing a separate synchronization file.
func appendWorkerReady() error {
	syncPath := os.Getenv("HARNESSING_FIXTURE_SYNC_FILE")
	if syncPath == "" {
		return nil
	}
	deadline := time.Now().Add(30 * time.Second)
	for {
		data, err := os.ReadFile(syncPath)
		if err == nil && bytes.Contains(data, []byte("worker_pid=")) {
			break
		}
		if time.Now().After(deadline) {
			return fmt.Errorf("parent's own sync-file write never appeared within 30s")
		}
		time.Sleep(20 * time.Millisecond)
	}
	f, err := os.OpenFile(syncPath, os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	defer func() { _ = f.Close() }()
	_, err = f.WriteString(workerReadyToken)
	return err
}

// releaseToken is the exact byte sequence H101-199's QA-defined release
// mechanism requires — not "contains a release word," not case
// insensitive, not trimmed: precisely these 7 bytes, so the fixture
// cannot be released by an accidental partial write racing this poll.
const releaseToken = "release\n"

// runFixtureServeRelease is H101-193/H101-199's graceful-complete
// producer (serve_release_both_channels): having already written the
// participation marker and the sync file above (proving startup
// survived, the same B5′ evidence class item 4 uses), it now blocks on
// an explicit release rather than racing a fixed sleep against
// Supervisor's StartupWindow — Stanley's H101-195 ruling requires this
// ordering be test-controlled, not timing-controlled. Only once released
// does it emit the two named channel literals and exit 0, so the host's
// graceful-completion drain (h101-128 line 79) has real, ordered bytes
// to capture, on both channels, only after the run is already observed
// Running.
func runFixtureServeRelease(stdout, stderr io.Writer) int {
	releasePath := os.Getenv("HARNESSING_FIXTURE_RELEASE_FILE")
	if releasePath == "" {
		_, _ = fmt.Fprintln(stderr, "harnessing fixture: HARNESSING_FIXTURE_RELEASE_FILE is not set")
		return 1
	}
	deadline := time.Now().Add(30 * time.Second)
	for {
		data, err := os.ReadFile(releasePath)
		if err == nil && string(data) == releaseToken {
			break
		}
		if time.Now().After(deadline) {
			_, _ = fmt.Fprintln(stderr, "harnessing fixture: no valid release within 30s")
			return 1
		}
		time.Sleep(50 * time.Millisecond)
	}
	_, _ = fmt.Fprint(stdout, "fixture-out-stdout\n")
	_, _ = fmt.Fprint(stderr, "fixture-out-stderr\n")
	return 0
}
