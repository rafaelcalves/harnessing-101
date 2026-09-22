package adaptercontract_test

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/rafaelcalves/harnessing-101/internal/adaptercontract/expected"
	"github.com/rafaelcalves/harnessing-101/internal/core/domain"
)

// UI-08: detached views; Phase 3 ops return Unsupported with detail (throwaway surface).
func TestUI08_DetachedViewMutationProbe(t *testing.T) {
	env := newEnv(t)
	registerAgent(t, context.Background(), drivers(t)[0], env, expected.EngineerID, "ui08-r1", "Engineer")
	createTask(t, context.Background(), drivers(t)[0], env, "ui08-r2", expected.TaskID, "Immutable", expected.EngineerID)
	harnessDetachedSnapshotProbe(t, env)
}

// TestUI08_Phase3OperationsUnsupported covers SetRunBudget only (item
// 3). StartRun moved off this row in H101-144 (Phase 3 item 1); StopRun
// moved off it the same way in H101-233 (Stanley): the throwaway
// adapter now forwards stop-run through the caller-bound session
// instead of manufacturing its own refusal, so its actual contract is
// whatever Engine.StopRun/the real supervisor returns, not a fixed
// Unsupported — see internal/adapters/process's TestSupervisor_Stop*
// and internal/core/task/engine.go's StopRun for that contract.
func TestUI08_Phase3OperationsUnsupported(t *testing.T) {
	ctx := context.Background()
	ops := []struct {
		name string
		op   string
		body map[string]string
	}{
		{name: "SetRunBudget", op: "set-run-budget", body: map[string]string{"RequestID": "ui08-bu", "RunID": "run1", "Budget": "1"}},
	}
	for _, tc := range ops {
		t.Run(tc.name, func(t *testing.T) {
			env := newEnv(t)
			payload, _ := json.Marshal(tc.body)
			result := newThrowawayDriver().Invoke(ctx, env, Call{
				Caller:             domain.AgentID(expected.EngineerID),
				ThrowawayOperation: tc.op,
				ThrowawayPayload:   payload,
			})
			assertErrorCode(t, result, string(domain.ErrUnsupported))
			if result.Detail == "" {
				t.Fatalf("%s: unsupported error missing detail", tc.name)
			}
		})
	}
}
