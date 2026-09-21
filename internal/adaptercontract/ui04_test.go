package adaptercontract_test

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/rafaelcalves/harnessing-101/internal/adaptercontract/expected"
	"github.com/rafaelcalves/harnessing-101/internal/core/domain"
)

// UI-04: full accountable-work cycle with identical domain outcomes.
func TestUI04_FullCycleDomainOutcomes(t *testing.T) {
	ctx := context.Background()
	for _, driver := range drivers(t) {
		t.Run(driver.Name(), func(t *testing.T) {
			env := newEnv(t)
			runUI04Cycle(t, ctx, driver, env)
			snap := harnessSnapshot(t, env)
			assertUI04EndState(t, snap)

			// H101-119 (Stanley): assert the three message identities as
			// distinct VALUES, not just that the send succeeded. Canonical
			// read first (works for either adapter), then this driver's
			// own read surface so both adapters are checked, not just the
			// shared harness path.
			canonicalMsg := findMessageInSnapshot(snap, domain.MessageID(expected.MessageID))
			if canonicalMsg == nil {
				t.Fatal("message m1 missing from canonical snapshot")
			}
			assertMessageIdentitySeparate(t, *canonicalMsg, domain.AgentID(expected.ClaimedSenderID), domain.AgentID(expected.EngineerID), domain.AgentID(expected.AnalystID))
			switch driver.Name() {
			case "cli":
				cli := driver.(*cliDriver)
				detail := queryCLIMessage(ctx, cli, env, expected.MessageID)
				if detail.ExitCode != 0 {
					t.Fatalf("message query failed: %q", detail.Stderr)
				}
				assertCLIMessageIdentity(t, detail.Stdout, domain.AgentID(expected.ClaimedSenderID), domain.AgentID(expected.EngineerID), domain.AgentID(expected.AnalystID))
			case "throwaway":
				td := driver.(*throwawayDriver)
				msg, result := throwawayMessage(t, ctx, td, env, domain.AgentID(expected.AnalystID), domain.MessageID(expected.MessageID))
				if result.ExitCode != 0 {
					t.Fatalf("message query failed: %q", result.Stderr)
				}
				assertMessageIdentitySeparate(t, msg, domain.AgentID(expected.ClaimedSenderID), domain.AgentID(expected.EngineerID), domain.AgentID(expected.AnalystID))
			}

			// AwaitingReview must not be presented as Done mid-cycle.
			mid := newEnv(t)
			runUI04CycleUntilAwaitingReview(t, ctx, driver, mid)
			q := driver.QueryTask(ctx, mid, domain.TaskID(expected.TaskID))
			combined := q.Stdout + q.Stderr
			if strings.Contains(combined, "Done") {
				t.Fatalf("AwaitingReview rendered as Done: %q", combined)
			}
			if !strings.Contains(combined, "AwaitingReview") {
				t.Fatalf("missing AwaitingReview status: %q", combined)
			}
		})
	}
}

func runUI04Cycle(t *testing.T, ctx context.Context, driver Driver, env WorkspaceEnv) {
	registerAgent(t, ctx, driver, env, expected.EngineerID, "r1", "Engineer")
	registerAgent(t, ctx, driver, env, expected.AnalystID, "r2", "Analyst")
	createTask(t, ctx, driver, env, "r3", expected.TaskID, "Investigate", expected.EngineerID)
	transitionTask(t, ctx, driver, env, "r4", expected.TaskID, "Todo", "Doing")
	sendMessage(t, ctx, driver, env, "r5", expected.MessageID, expected.AnalystID, "Please investigate", expected.TaskID)
	ackMessage(t, ctx, driver, env, "r6", expected.MessageID, expected.AnalystID)
	transitionTask(t, ctx, driver, env, "r7", expected.TaskID, "Doing", "Blocked", "Need evidence")
	transitionTask(t, ctx, driver, env, "r8", expected.TaskID, "Blocked", "Doing", "")
	reportResult(t, ctx, driver, env, "r9", expected.TaskID, expected.ResultID1, 4, "Evidence attached", []string{"report.md"})
	deniedAccept(t, ctx, driver, env, "r10-denied", expected.TaskID, expected.ResultID1, 5, expected.EngineerID)
	rejectResult(t, ctx, driver, env, "r11", expected.TaskID, expected.ResultID1, 5, "Add evidence")
	reportResult(t, ctx, driver, env, "r12", expected.TaskID, expected.ResultID2, 6, "Evidence complete", []string{"final.md"})
	acceptResult(t, ctx, driver, env, "r13", expected.TaskID, expected.ResultID2, 7, expected.ReviewerID)
}

func runUI04CycleUntilAwaitingReview(t *testing.T, ctx context.Context, driver Driver, env WorkspaceEnv) {
	registerAgent(t, ctx, driver, env, expected.EngineerID, "r1", "Engineer")
	createTask(t, ctx, driver, env, "r3", expected.TaskID, "Investigate", expected.EngineerID)
	transitionTask(t, ctx, driver, env, "r4", expected.TaskID, "Todo", "Doing")
	reportResult(t, ctx, driver, env, "r9", expected.TaskID, expected.ResultID1, 2, "Evidence attached", []string{"report.md"})
}

func registerAgent(t *testing.T, ctx context.Context, driver Driver, env WorkspaceEnv, agentID, requestID, name string) Result {
	var result Result
	switch driver.Name() {
	case "cli":
		result = driver.Invoke(ctx, env, Call{
			Caller: domain.AgentID(agentID),
			CLIArgs: []string{
				"register",
				"-caller", agentID,
				"-request-id", requestID,
				"-agent", agentID,
				"-display-name", name,
			},
		})
	case "throwaway":
		payload, _ := json.Marshal(map[string]string{
			"RequestID": requestID, "AgentID": agentID, "DisplayName": name,
		})
		result = driver.Invoke(ctx, env, Call{
			Caller:             domain.AgentID(agentID),
			ThrowawayOperation: "register",
			ThrowawayPayload:   payload,
		})
	}
	if result.ExitCode != 0 {
		t.Fatalf("register %s: exit=%d stderr=%q", agentID, result.ExitCode, result.Stderr)
	}
	return result
}

func createTask(t *testing.T, ctx context.Context, driver Driver, env WorkspaceEnv, requestID, taskID, title, assignee string) {
	var result Result
	switch driver.Name() {
	case "cli":
		result = driver.Invoke(ctx, env, Call{
			Caller: domain.AgentID(expected.EngineerID),
			CLIArgs: []string{
				"create",
				"-caller", expected.EngineerID,
				"-request-id", requestID,
				"-task", taskID,
				"-title", title,
				"-assignee", assignee,
			},
		})
	case "throwaway":
		payload, _ := json.Marshal(map[string]string{
			"RequestID": requestID, "TaskID": taskID, "Title": title, "AssigneeID": assignee,
		})
		result = driver.Invoke(ctx, env, Call{
			Caller:             domain.AgentID(expected.EngineerID),
			ThrowawayOperation: "create",
			ThrowawayPayload:   payload,
		})
	}
	if result.ExitCode != 0 {
		t.Fatalf("create %s: %q", taskID, result.Stderr)
	}
}

func transitionTask(t *testing.T, ctx context.Context, driver Driver, env WorkspaceEnv, requestID, taskID, from, to string, reason ...string) {
	reasonText := ""
	if len(reason) > 0 {
		reasonText = reason[0]
	}
	var result Result
	switch driver.Name() {
	case "cli":
		args := []string{
			"transition",
			"-caller", expected.EngineerID,
			"-request-id", requestID,
			"-task", taskID,
			"-from", from, "-to", to,
		}
		if reasonText != "" {
			args = append(args, "-reason", reasonText)
		}
		result = driver.Invoke(ctx, env, Call{Caller: domain.AgentID(expected.EngineerID), CLIArgs: args})
	case "throwaway":
		payload, _ := json.Marshal(map[string]interface{}{
			"RequestID": requestID, "TaskID": taskID, "FromStatus": from, "ToStatus": to, "Reason": reasonText,
		})
		result = driver.Invoke(ctx, env, Call{
			Caller: domain.AgentID(expected.EngineerID), ThrowawayOperation: "transition", ThrowawayPayload: payload,
		})
	}
	if result.ExitCode != 0 {
		t.Fatalf("transition %s->%s: %q", from, to, result.Stderr)
	}
}

func sendMessage(t *testing.T, ctx context.Context, driver Driver, env WorkspaceEnv, requestID, messageID, recipient, body, taskID string) {
	var result Result
	switch driver.Name() {
	case "cli":
		// H101-23/H101-109: SendMessage now checks SenderAgentID against
		// the registry, so this deliberately-not-the-caller claimed
		// sender (proving the claim is independent of -caller) must
		// itself be registered — Kelly's H101-111 ruling (a): register
		// it as a third agent rather than reuse an already-registered
		// one, which would collapse sender into caller/recipient and
		// erase the independence proof.
		registerAgent(t, ctx, driver, env, expected.ClaimedSenderID, requestID+"-register-claimed-sender", "Claimed Engineer")
		result = driver.Invoke(ctx, env, Call{
			Caller: domain.AgentID(expected.EngineerID),
			CLIArgs: []string{
				"send",
				"-caller", expected.EngineerID,
				"-request-id", requestID,
				"-message", messageID,
				"-recipient", recipient,
				"-kind", "Request",
				"-body", body,
				"-sender", expected.ClaimedSenderID,
				"-task", taskID,
			},
		})
	case "throwaway":
		// Cross-adapter parity per H101-111: throwaway's claimed sender
		// now matches the CLI path's (expected.ClaimedSenderID) instead
		// of equaling -caller, so both adapters exercise the same
		// independent-claim property, not just the CLI one.
		registerAgent(t, ctx, driver, env, expected.ClaimedSenderID, requestID+"-register-claimed-sender", "Claimed Engineer")
		payload, _ := json.Marshal(map[string]interface{}{
			"RequestID": requestID, "MessageID": messageID,
			"SenderAgentID": expected.ClaimedSenderID, "RecipientAgentID": recipient,
			"Kind": "Request", "Body": body, "TaskID": taskID,
		})
		result = driver.Invoke(ctx, env, Call{
			Caller: domain.AgentID(expected.EngineerID), ThrowawayOperation: "send", ThrowawayPayload: payload,
		})
	}
	if result.ExitCode != 0 {
		t.Fatalf("send %s: %q", messageID, result.Stderr)
	}
}

func ackMessage(t *testing.T, ctx context.Context, driver Driver, env WorkspaceEnv, requestID, messageID, caller string) {
	var result Result
	switch driver.Name() {
	case "cli":
		result = driver.Invoke(ctx, env, Call{
			Caller: domain.AgentID(caller),
			CLIArgs: []string{
				"ack",
				"-caller", caller,
				"-request-id", requestID,
				"-message", messageID,
			},
		})
	case "throwaway":
		payload, _ := json.Marshal(map[string]string{"RequestID": requestID, "MessageID": messageID})
		result = driver.Invoke(ctx, env, Call{
			Caller: domain.AgentID(caller), ThrowawayOperation: "ack", ThrowawayPayload: payload,
		})
	}
	if result.ExitCode != 0 {
		t.Fatalf("ack %s: %q", messageID, result.Stderr)
	}
}

func reportResult(t *testing.T, ctx context.Context, driver Driver, env WorkspaceEnv, requestID, taskID, resultID string, expectedRev int, summary string, artifacts []string) {
	var result Result
	switch driver.Name() {
	case "cli":
		args := []string{
			"report",
			"-caller", expected.EngineerID,
			"-request-id", requestID,
			"-task", taskID,
			"-result", resultID,
			"-expected-revision", itoa(expectedRev),
			"-summary", summary,
		}
		for _, a := range artifacts {
			args = append(args, "-artifact", a)
		}
		result = driver.Invoke(ctx, env, Call{Caller: domain.AgentID(expected.EngineerID), CLIArgs: args})
	case "throwaway":
		payload, _ := json.Marshal(map[string]interface{}{
			"RequestID": requestID, "TaskID": taskID, "ResultID": resultID,
			"ExpectedTaskRevision": uint64(expectedRev), "Summary": summary, "Artifacts": artifacts,
		})
		result = driver.Invoke(ctx, env, Call{
			Caller: domain.AgentID(expected.EngineerID), ThrowawayOperation: "report", ThrowawayPayload: payload,
		})
	}
	if result.ExitCode != 0 {
		t.Fatalf("report %s: %q", resultID, result.Stderr)
	}
}

func deniedAccept(t *testing.T, ctx context.Context, driver Driver, env WorkspaceEnv, requestID, taskID, resultID string, expectedRev int, caller string) {
	var result Result
	switch driver.Name() {
	case "cli":
		result = driver.Invoke(ctx, env, Call{
			Caller: domain.AgentID(caller),
			CLIArgs: []string{
				"accept",
				"-caller", caller,
				"-request-id", requestID,
				"-task", taskID,
				"-result", resultID,
				"-expected-revision", itoa(expectedRev),
			},
		})
	case "throwaway":
		payload, _ := json.Marshal(map[string]interface{}{
			"RequestID": requestID, "TaskID": taskID, "ResultID": resultID, "ExpectedTaskRevision": uint64(expectedRev),
		})
		result = driver.Invoke(ctx, env, Call{
			Caller: domain.AgentID(caller), ThrowawayOperation: "accept", ThrowawayPayload: payload,
		})
	}
	assertErrorCode(t, result, expected.CodeDenied)
}

func rejectResult(t *testing.T, ctx context.Context, driver Driver, env WorkspaceEnv, requestID, taskID, resultID string, expectedRev int, reason string) {
	var result Result
	switch driver.Name() {
	case "cli":
		result = driver.Invoke(ctx, env, Call{
			Caller: domain.AgentID(expected.ReviewerID),
			CLIArgs: []string{
				"reject",
				"-caller", expected.ReviewerID,
				"-request-id", requestID,
				"-task", taskID,
				"-result", resultID,
				"-expected-revision", itoa(expectedRev),
				"-reason", reason,
			},
		})
	case "throwaway":
		payload, _ := json.Marshal(map[string]interface{}{
			"RequestID": requestID, "TaskID": taskID, "ResultID": resultID,
			"ExpectedTaskRevision": uint64(expectedRev), "Reason": reason,
		})
		result = driver.Invoke(ctx, env, Call{
			Caller: domain.AgentID(expected.ReviewerID), ThrowawayOperation: "reject", ThrowawayPayload: payload,
		})
	}
	if result.ExitCode != 0 {
		t.Fatalf("reject: %q", result.Stderr)
	}
}

func acceptResult(t *testing.T, ctx context.Context, driver Driver, env WorkspaceEnv, requestID, taskID, resultID string, expectedRev int, caller string) {
	var result Result
	switch driver.Name() {
	case "cli":
		result = driver.Invoke(ctx, env, Call{
			Caller: domain.AgentID(caller),
			CLIArgs: []string{
				"accept",
				"-caller", caller,
				"-request-id", requestID,
				"-task", taskID,
				"-result", resultID,
				"-expected-revision", itoa(expectedRev),
			},
		})
	case "throwaway":
		payload, _ := json.Marshal(map[string]interface{}{
			"RequestID": requestID, "TaskID": taskID, "ResultID": resultID, "ExpectedTaskRevision": uint64(expectedRev),
		})
		result = driver.Invoke(ctx, env, Call{
			Caller: domain.AgentID(caller), ThrowawayOperation: "accept", ThrowawayPayload: payload,
		})
	}
	if result.ExitCode != 0 {
		t.Fatalf("accept: %q", result.Stderr)
	}
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var digits []byte
	for v := n; v > 0; v /= 10 {
		digits = append([]byte{byte('0' + v%10)}, digits...)
	}
	return string(digits)
}
