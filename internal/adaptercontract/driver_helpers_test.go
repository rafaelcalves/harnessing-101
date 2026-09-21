package adaptercontract_test

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/rafaelcalves/harnessing-101/internal/core/domain"
	"github.com/rafaelcalves/harnessing-101/internal/throwawayadapter"
)

func invokeCLI(ctx context.Context, d *cliDriver, env WorkspaceEnv, caller domain.AgentID, subcommand string, extra ...string) Result {
	args := joinCLI([]string{subcommand}, extra...)
	return d.Invoke(ctx, env, Call{Caller: caller, CLIArgs: args})
}

func queryCLIMessages(ctx context.Context, d *cliDriver, env WorkspaceEnv, recipient string) Result {
	return invokeCLI(ctx, d, env, "", "messages", "-recipient", recipient)
}

func queryCLIMessage(ctx context.Context, d *cliDriver, env WorkspaceEnv, messageID string) Result {
	return invokeCLI(ctx, d, env, "", "message", messageID)
}

func decodeThrowawayOK(t *testing.T, result Result) json.RawMessage {
	t.Helper()
	var resp throwawayadapter.Response
	if err := json.Unmarshal([]byte(result.Stdout), &resp); err != nil {
		t.Fatalf("decode throwaway response: %v raw=%s", err, result.Stdout)
	}
	if !resp.OK {
		t.Fatalf("throwaway response not ok: code=%q detail=%q", result.Code, result.Detail)
	}
	raw, err := json.Marshal(resp.Result)
	if err != nil {
		t.Fatalf("marshal throwaway result: %v", err)
	}
	return raw
}

func throwawaySnapshot(t *testing.T, ctx context.Context, d *throwawayDriver, env WorkspaceEnv, caller domain.AgentID) (domain.Snapshot, Result) {
	result := d.Invoke(ctx, env, Call{
		Caller:             caller,
		ThrowawayOperation: "snapshot",
	})
	if result.ExitCode != 0 || !result.OK {
		return domain.Snapshot{}, result
	}
	var snap domain.Snapshot
	if err := json.Unmarshal(decodeThrowawayOK(t, result), &snap); err != nil {
		t.Fatalf("unmarshal snapshot: %v", err)
	}
	return snap, result
}

func throwawayMessage(t *testing.T, ctx context.Context, d *throwawayDriver, env WorkspaceEnv, caller domain.AgentID, messageID domain.MessageID) (domain.Message, Result) {
	payload, _ := json.Marshal(map[string]string{"messageID": string(messageID)})
	result := d.Invoke(ctx, env, Call{
		Caller:             caller,
		ThrowawayOperation: "message",
		ThrowawayPayload:   payload,
	})
	if result.ExitCode != 0 || !result.OK {
		return domain.Message{}, result
	}
	var msg domain.Message
	if err := json.Unmarshal(decodeThrowawayOK(t, result), &msg); err != nil {
		t.Fatalf("unmarshal message: %v", err)
	}
	return msg, result
}

func assertNoValidationBadge(t *testing.T, output string) {
	t.Helper()
	lower := strings.ToLower(output)
	for _, forbidden := range []string{"verified sender", "validated", "identity confirmed", "✓"} {
		if strings.Contains(lower, forbidden) {
			t.Fatalf("output suggests validation badge %q: %s", forbidden, output)
		}
	}
}
