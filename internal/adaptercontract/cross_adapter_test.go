package adaptercontract_test

import (
	"context"
	"testing"

	"github.com/rafaelcalves/harnessing-101/internal/adaptercontract/expected"
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
}
