package adaptercontract_test

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/rafaelcalves/harnessing-101/internal/adaptercontract/expected"
	"github.com/rafaelcalves/harnessing-101/internal/core/domain"
)

// UI-03: idempotent retry by request ID; changed payload conflicts.
func TestUI03_IdempotentRetrySameReceipt(t *testing.T) {
	ctx := context.Background()
	want := expected.UI03IdempotentRegister
	for _, driver := range drivers(t) {
		t.Run(driver.Name(), func(t *testing.T) {
			env := newEnv(t)
			first := registerAgent(t, ctx, driver, env, expected.EngineerID, string(want.RequestID), "Engineer")
			second := registerAgent(t, ctx, driver, env, expected.EngineerID, string(want.RequestID), "Engineer")
			assertReceiptEqual(t, receiptFromResult(t, first), want)
			assertReceiptEqual(t, receiptFromResult(t, second), want)

			snap := harnessSnapshot(t, env)
			if snap.Revision != want.Revision {
				t.Fatalf("idempotent retry advanced revision to %d, want %d", snap.Revision, want.Revision)
			}
			if len(snap.Agents) != 1 {
				t.Fatalf("agent count = %d after retry, want 1", len(snap.Agents))
			}
		})
	}
}

func TestUI03_ChangedPayloadSameRequestIDConflicts(t *testing.T) {
	ctx := context.Background()
	for _, driver := range drivers(t) {
		t.Run(driver.Name(), func(t *testing.T) {
			env := newEnv(t)
			_ = registerAgent(t, ctx, driver, env, expected.EngineerID, "ui03-r1", "Engineer")

			var conflict Result
			switch driver.Name() {
			case "cli":
				conflict = driver.Invoke(ctx, env, Call{
					Caller: domain.AgentID(expected.EngineerID),
					CLIArgs: []string{
						"register",
						"-caller", expected.EngineerID,
						"-request-id", "ui03-r1",
						"-agent", expected.EngineerID,
						"-display-name", "Different Name",
					},
				})
			case "throwaway":
				payload, _ := json.Marshal(map[string]string{
					"RequestID":   "ui03-r1",
					"AgentID":     expected.EngineerID,
					"DisplayName": "Different Name",
				})
				conflict = driver.Invoke(ctx, env, Call{
					Caller:             domain.AgentID(expected.EngineerID),
					ThrowawayOperation: "register",
					ThrowawayPayload:   payload,
				})
			}
			assertErrorCode(t, conflict, expected.CodeConflict)
		})
	}
}
