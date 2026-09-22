package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
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
	outputMode := os.Getenv("HARNESSING_FIXTURE_OUTPUT_MODE")
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
	if syncPath := os.Getenv("HARNESSING_FIXTURE_SYNC_FILE"); syncPath != "" {
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
