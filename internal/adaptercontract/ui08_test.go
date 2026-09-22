package adaptercontract_test

import (
	"bytes"
	"context"
	"encoding/json"
	"testing"

	"github.com/rafaelcalves/harnessing-101/internal/adaptercontract/expected"
	"github.com/rafaelcalves/harnessing-101/internal/api"
	"github.com/rafaelcalves/harnessing-101/internal/assembly"
	"github.com/rafaelcalves/harnessing-101/internal/core/domain"
	"github.com/rafaelcalves/harnessing-101/internal/throwawayadapter"
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

// TestUI08_CapabilityDisabledHosting_StopRunUnsupported is H101-236,
// building H101-235's F1-F5 (Kelly): the honest UI-08 negative that
// TestUI08_Phase3OperationsUnsupported's stop-run removal left open.
// It is NOT the same claim as before -- this host genuinely has no
// ProcessSupervisor wired (host.OpenWithoutSupervisor, via
// assembly.WithSessionWithoutSupervisor, F1/F2: the real product
// condition engine.go's StopRun checks, not a mock standing in for
// it), so the throwaway adapter's real forward (H101-233) reaches
// Engine.StopRun's own "no process supervisor is configured for this
// host" path and Unsupported is what THAT path returns -- never a
// refusal manufactured at the adapter (F5's non-goal). A supervision-
// enabled host's stop-run is UI-13/item-7 territory, not this row.
func TestUI08_CapabilityDisabledHosting_StopRunUnsupported(t *testing.T) {
	env := newEnv(t)
	payload, err := json.Marshal(map[string]string{"RequestID": "ui08-cd-st", "RunID": "run1"})
	if err != nil {
		t.Fatalf("marshal payload: %v", err)
	}

	var out Result
	var stderr bytes.Buffer
	code := assembly.WithSessionWithoutSupervisor(&stderr, env.Root, env.WorkspaceID, env.Reviewers, domain.AgentID(expected.EngineerID), "contract", func(ctx context.Context, session api.FrontendSession) int {
		adapter := throwawayadapter.New(session)
		raw, err := json.Marshal(struct {
			Operation string          `json:"operation"`
			Payload   json.RawMessage `json:"payload"`
		}{Operation: "stop-run", Payload: payload})
		if err != nil {
			out = Result{ExitCode: 1, Code: string(domain.ErrIOFailure), Detail: err.Error()}
			return 1
		}
		respBytes, err := adapter.Handle(ctx, raw)
		if err != nil {
			out = Result{ExitCode: 1, Code: string(domain.ErrIOFailure), Detail: err.Error()}
			return 1
		}
		var resp throwawayadapter.Response
		if err := json.Unmarshal(respBytes, &resp); err != nil {
			out = Result{ExitCode: 1, Code: string(domain.ErrIOFailure), Detail: err.Error()}
			return 1
		}
		out = Result{ExitCode: boolExit(resp.OK), OK: resp.OK, Stdout: string(respBytes), Stderr: stderr.String()}
		if resp.Error != nil {
			out.Code = resp.Error.Code
			out.Detail = resp.Error.Detail
		}
		return out.ExitCode
	})
	_ = code

	assertErrorCode(t, out, string(domain.ErrUnsupported))
	// Pinned to the EXACT detail Engine.StopRun's own no-supervisor path
	// emits (internal/core/task/engine.go), not just a non-empty string
	// -- an Unsupported+detail pair alone cannot distinguish this real
	// forward from an adapter-local refusal that happens to say
	// something else. Found this gap myself: a manufactured
	// `&domain.Error{Code: ErrUnsupported, Detail: "..."}` at the
	// adapter passed the weaker check just as well, which is exactly
	// the regression H101-235's gate 3 exists to catch.
	const wantDetail = "no process supervisor is configured for this host"
	if out.Detail != wantDetail {
		t.Fatalf("capability-disabled stop-run: detail = %q, want %q (must come from Engine.StopRun's own path, not an adapter-manufactured refusal)", out.Detail, wantDetail)
	}
}
