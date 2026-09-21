package adaptercontract_test

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/rafaelcalves/harnessing-101/internal/adaptercontract/expected"
	"github.com/rafaelcalves/harnessing-101/internal/core/domain"
)

// UI-02: receipt distinct from completed work; stable error codes on failure paths.
func TestUI02_ReceiptDistinctFromTaskStatus(t *testing.T) {
	ctx := context.Background()
	for _, driver := range drivers(t) {
		t.Run(driver.Name(), func(t *testing.T) {
			env := newEnv(t)
			seedMinimalTask(t, ctx, driver, env)

			var reg Result
			switch driver.Name() {
			case "cli":
				reg = driver.Invoke(ctx, env, Call{
					Caller: domain.AgentID(expected.EngineerID),
					CLIArgs: []string{
						"transition",
						"-caller", expected.EngineerID,
						"-request-id", "ui02-r4",
						"-task", expected.TaskID,
						"-from", "Todo", "-to", "Doing",
					},
				})
			case "throwaway":
				payload, _ := json.Marshal(map[string]interface{}{
					"RequestID":  "ui02-r4",
					"TaskID":     expected.TaskID,
					"FromStatus": "Todo",
					"ToStatus":   "Doing",
				})
				reg = driver.Invoke(ctx, env, Call{
					Caller:             domain.AgentID(expected.EngineerID),
					ThrowawayOperation: "transition",
					ThrowawayPayload:   payload,
				})
			}
			if reg.ExitCode != 0 {
				t.Fatalf("transition failed: %q %q", reg.Stderr, reg.Stdout)
			}
			if !strings.Contains(reg.Stdout, "OK (request ui02-r4") && driver.Name() == "cli" {
				t.Fatalf("stdout missing receipt line: %q", reg.Stdout)
			}

			query := driver.QueryTask(ctx, env, domain.TaskID(expected.TaskID))
			if query.ExitCode != 0 {
				t.Fatalf("task query failed: %q", query.Stderr)
			}
			combined := query.Stdout + query.Stderr
			if strings.Contains(combined, "OK (request") {
				t.Fatal("task status query must not render a mutation receipt")
			}
			if !strings.Contains(combined, "Doing") {
				t.Fatalf("task status missing Doing: %q", combined)
			}
		})
	}
}

func TestUI02_DeniedAndConflictStableCodes(t *testing.T) {
	ctx := context.Background()
	for _, driver := range drivers(t) {
		t.Run(driver.Name()+"_denied", func(t *testing.T) {
			env := newEnv(t)
			seedReportedTask(t, ctx, driver, env)
			var result Result
			switch driver.Name() {
			case "cli":
				result = driver.Invoke(ctx, env, Call{
					Caller: domain.AgentID(expected.EngineerID),
					CLIArgs: []string{
						"accept",
						"-caller", expected.EngineerID,
						"-request-id", "ui02-denied",
						"-task", expected.TaskID,
						"-result", expected.ResultID1,
						"-expected-revision", "3",
					},
				})
			case "throwaway":
				payload, _ := json.Marshal(map[string]interface{}{
					"RequestID":            "ui02-denied",
					"TaskID":               expected.TaskID,
					"ResultID":             expected.ResultID1,
					"ExpectedTaskRevision": 3,
				})
				result = driver.Invoke(ctx, env, Call{
					Caller:             domain.AgentID(expected.EngineerID),
					ThrowawayOperation: "accept",
					ThrowawayPayload:   payload,
				})
			}
			assertErrorCode(t, result, expected.CodeDenied)
		})

		t.Run(driver.Name()+"_conflict", func(t *testing.T) {
			env := newEnv(t)
			seedMinimalTask(t, ctx, driver, env)
			var result Result
			switch driver.Name() {
			case "cli":
				result = driver.Invoke(ctx, env, Call{
					Caller: domain.AgentID(expected.EngineerID),
					CLIArgs: []string{
						"transition",
						"-caller", expected.EngineerID,
						"-request-id", "ui02-conflict",
						"-task", expected.TaskID,
						"-from", "Doing", "-to", "Blocked", "-reason", "stale",
					},
				})
			case "throwaway":
				payload, _ := json.Marshal(map[string]interface{}{
					"RequestID":  "ui02-conflict",
					"TaskID":     expected.TaskID,
					"FromStatus": "Doing",
					"ToStatus":   "Blocked",
					"Reason":     "stale",
				})
				result = driver.Invoke(ctx, env, Call{
					Caller:             domain.AgentID(expected.EngineerID),
					ThrowawayOperation: "transition",
					ThrowawayPayload:   payload,
				})
			}
			assertErrorCode(t, result, expected.CodeConflict)
		})
	}
}

func seedMinimalTask(t *testing.T, ctx context.Context, driver Driver, env WorkspaceEnv) {
	registerAgent(t, ctx, driver, env, expected.EngineerID, "ui02-seed-r1", "Engineer")
	createTask(t, ctx, driver, env, "ui02-seed-r2", expected.TaskID, "Probe", expected.EngineerID)
}

func seedReportedTask(t *testing.T, ctx context.Context, driver Driver, env WorkspaceEnv) {
	seedMinimalTask(t, ctx, driver, env)
	transitionTask(t, ctx, driver, env, "ui02-seed-r3", expected.TaskID, "Todo", "Doing")
	reportResult(t, ctx, driver, env, "ui02-seed-r4", expected.TaskID, expected.ResultID1, 2, "done", []string{"a.txt"})
}
