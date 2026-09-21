package adaptercontract_test

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/rafaelcalves/harnessing-101/internal/adaptercontract/expected"
	"github.com/rafaelcalves/harnessing-101/internal/core/domain"
)

// UI-01: malformed input rejected without mutation; valid command preserves IDs.
func TestUI01_MalformedInputNoMutation(t *testing.T) {
	ctx := context.Background()
	for _, driver := range drivers(t) {
		t.Run(driver.Name(), func(t *testing.T) {
			env := newEnv(t)
			before := harnessSnapshot(t, env)

			var result Result
			switch driver.Name() {
			case "cli":
				result = driver.Invoke(ctx, env, Call{
					CLIArgs: []string{"register", "-caller", expected.EngineerID, "-request-id", "bad", "-agent", "x", "-display-name", "X"},
				})
			case "throwaway":
				result = driver.Invoke(ctx, env, Call{
					Caller:             domain.AgentID(expected.EngineerID),
					ThrowawayOperation: "register",
					ThrowawayPayload:   []byte(`{}`),
				})
			}
			if result.ExitCode == 0 || result.OK {
				t.Fatalf("malformed input succeeded: %+v", result)
			}
			if driver.Name() == "throwaway" {
				assertErrorCode(t, result, expected.CodeInvalidArgument)
			}
			after := harnessSnapshot(t, env)
			assertSnapshotUnchanged(t, before, after)
		})
	}
}

func TestUI01_ValidRegisterPreservesIDs(t *testing.T) {
	ctx := context.Background()
	want := expected.UI01ValidRegister
	for _, driver := range drivers(t) {
		t.Run(driver.Name(), func(t *testing.T) {
			env := newEnv(t)
			var result Result
			switch driver.Name() {
			case "cli":
				result = driver.Invoke(ctx, env, Call{
					Caller: domain.AgentID(expected.EngineerID),
					CLIArgs: []string{
						"register",
						"-caller", expected.EngineerID,
						"-request-id", string(want.Receipt.RequestID),
						"-agent", string(want.Agent),
						"-display-name", want.Name,
					},
				})
			case "throwaway":
				payload, _ := json.Marshal(map[string]string{
					"RequestID":   string(want.Receipt.RequestID),
					"AgentID":     string(want.Agent),
					"DisplayName": want.Name,
				})
				result = driver.Invoke(ctx, env, Call{
					Caller:             domain.AgentID(expected.EngineerID),
					ThrowawayOperation: "register",
					ThrowawayPayload:   payload,
				})
			}
			if result.ExitCode != 0 || !result.OK {
				t.Fatalf("register failed: exit=%d stderr=%q stdout=%q code=%q", result.ExitCode, result.Stderr, result.Stdout, result.Code)
			}
			assertReceiptEqual(t, receiptFromResult(t, result), want.Receipt)

			snap := harnessSnapshot(t, env)
			if len(snap.Agents) != 1 {
				t.Fatalf("agent count = %d, want 1", len(snap.Agents))
			}
			if snap.Agents[0].ID != want.Agent || snap.Agents[0].DisplayName != want.Name {
				t.Fatalf("agent = %+v, want id=%s name=%s", snap.Agents[0], want.Agent, want.Name)
			}
		})
	}
}
