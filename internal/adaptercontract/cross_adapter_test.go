package adaptercontract_test

import (
	"context"
	"testing"

	"github.com/rafaelcalves/harnessing-101/internal/adaptercontract/expected"
	"github.com/rafaelcalves/harnessing-101/internal/core/domain"
)

// Cross-cutting: adapter A writes, adapter B continues on a shared workspace.
func TestCrossAdapter_Continuation(t *testing.T) {
	ctx := context.Background()
	cli := newCLIDriver(t)
	throwaway := newThrowawayDriver()
	env := newEnv(t)

	registerAgent(t, ctx, cli, env, expected.EngineerID, "cross-r1", "Engineer")
	registerAgent(t, ctx, cli, env, expected.AnalystID, "cross-r2", "Analyst")
	createTask(t, ctx, cli, env, "cross-r3", expected.TaskID, "Investigate", expected.EngineerID)
	transitionTask(t, ctx, cli, env, "cross-r4", expected.TaskID, "Todo", "Doing")
	sendMessage(t, ctx, cli, env, "cross-r5", expected.MessageID, expected.AnalystID, "Please investigate", expected.TaskID)

	ackMessage(t, ctx, throwaway, env, "cross-r6", expected.MessageID, expected.AnalystID)
	transitionTask(t, ctx, throwaway, env, "cross-r7", expected.TaskID, "Doing", "Blocked", "Need evidence")
	transitionTask(t, ctx, throwaway, env, "cross-r8", expected.TaskID, "Blocked", "Doing", "")
	reportResult(t, ctx, throwaway, env, "cross-r9", expected.TaskID, expected.ResultID1, 4, "Evidence attached", []string{"report.md"})
	rejectResult(t, ctx, throwaway, env, "cross-r10", expected.TaskID, expected.ResultID1, 5, "Add evidence")
	reportResult(t, ctx, throwaway, env, "cross-r11", expected.TaskID, expected.ResultID2, 6, "Evidence complete", []string{"final.md"})
	acceptResult(t, ctx, throwaway, env, "cross-r12", expected.TaskID, expected.ResultID2, 7, expected.ReviewerID)

	snap := harnessSnapshot(t, env)
	assertUI04EndState(t, snap)

	// H101-119 (Stanley): the same three-identity distinctness check,
	// this time after a close-and-reopen crossing adapters — cli wrote
	// the message, and every write/read above (including this
	// harnessSnapshot call) opened and closed the workspace fresh, so
	// this is the value surviving reopen. Assert canonically, then
	// through the OTHER adapter's own read surface (throwaway; cli did
	// the writing).
	canonicalMsg := findMessageInSnapshot(snap, domain.MessageID(expected.MessageID))
	if canonicalMsg == nil {
		t.Fatal("message m1 missing from canonical snapshot after cross-adapter continuation")
	}
	assertMessageIdentitySeparate(t, *canonicalMsg, domain.AgentID(expected.ClaimedSenderID), domain.AgentID(expected.EngineerID), domain.AgentID(expected.AnalystID))

	msg, result := throwawayMessage(t, ctx, throwaway, env, domain.AgentID(expected.AnalystID), domain.MessageID(expected.MessageID))
	if result.ExitCode != 0 {
		t.Fatalf("throwaway message query after crossover failed: %q", result.Stderr)
	}
	assertMessageIdentitySeparate(t, msg, domain.AgentID(expected.ClaimedSenderID), domain.AgentID(expected.EngineerID), domain.AgentID(expected.AnalystID))
}
