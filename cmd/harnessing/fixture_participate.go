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
	_, _ = fmt.Fprintln(stdout, "harnessing fixture: participated")

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
