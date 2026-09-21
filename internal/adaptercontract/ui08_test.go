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

func TestUI08_Phase3OperationsUnsupported(t *testing.T) {
	ctx := context.Background()
	ops := []struct {
		name string
		op   string
		body map[string]string
	}{
		{name: "StartRun", op: "start-run", body: map[string]string{"RequestID": "ui08-sr", "RunID": "run1"}},
		{name: "StopRun", op: "stop-run", body: map[string]string{"RequestID": "ui08-st", "RunID": "run1"}},
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
