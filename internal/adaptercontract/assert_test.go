package adaptercontract_test

import (
	"encoding/json"
	"regexp"
	"strings"
	"testing"

	"github.com/rafaelcalves/harnessing-101/internal/adaptercontract/expected"
	"github.com/rafaelcalves/harnessing-101/internal/core/domain"
)

var cliReceiptLine = regexp.MustCompile(`harnessing \w+: OK \(request (\S+), workspace revision (\d+)\)`)

func parseCLIReceipt(stdout string) *domain.Receipt {
	m := cliReceiptLine.FindStringSubmatch(stdout)
	if m == nil {
		return nil
	}
	var rev uint64
	_, _ = parseUint(m[2], &rev)
	return &domain.Receipt{RequestID: domain.RequestID(m[1]), CommittedRevision: rev}
}

func parseUint(s string, dst *uint64) (bool, error) {
	var v uint64
	for _, c := range s {
		if c < '0' || c > '9' {
			return false, nil
		}
		v = v*10 + uint64(c-'0')
	}
	*dst = v
	return true, nil
}

func receiptFromResult(t *testing.T, r Result) *domain.Receipt {
	t.Helper()
	if r.Receipt != nil {
		return r.Receipt
	}
	if rec := parseCLIReceipt(r.Stdout); rec != nil {
		return rec
	}
	var resp struct {
		OK     bool            `json:"ok"`
		Result json.RawMessage `json:"result"`
	}
	if err := json.Unmarshal([]byte(r.Stdout), &resp); err == nil && resp.OK && len(resp.Result) > 0 {
		return parseThrowawayReceipt(t, resp.Result)
	}
	return nil
}

func assertReceiptEqual(t *testing.T, got *domain.Receipt, want expected.Receipt) {
	t.Helper()
	if got == nil {
		t.Fatal("no receipt parsed from adapter result")
	}
	if got.RequestID != want.RequestID {
		t.Fatalf("receipt request ID = %q, want %q", got.RequestID, want.RequestID)
	}
	if got.CommittedRevision != want.Revision {
		t.Fatalf("receipt revision = %d, want %d", got.CommittedRevision, want.Revision)
	}
}

func assertErrorCode(t *testing.T, r Result, wantCode string) {
	t.Helper()
	if r.ExitCode == 0 && r.OK {
		t.Fatalf("expected failure with code %q, got success", wantCode)
	}
	code := r.Code
	if code == "" {
		// CLI renders "Code: detail" in stderr after describeError.
		for _, line := range strings.Split(r.Stderr, "\n") {
			if idx := strings.Index(line, ":"); idx > 0 {
				prefix := strings.TrimSpace(line[strings.LastIndex(line, "harnessing ")+len("harnessing "):])
				if parts := strings.SplitN(prefix, ":", 2); len(parts) == 2 {
					code = strings.TrimSpace(parts[1])
					if strings.Contains(code, ":") {
						code = strings.SplitN(code, ":", 2)[0]
					}
					break
				}
			}
			if strings.Contains(line, wantCode+":") || strings.HasPrefix(strings.TrimSpace(line), wantCode) {
				return
			}
		}
		if strings.Contains(r.Stderr, wantCode+":") || strings.Contains(r.Stderr, wantCode) {
			return
		}
	}
	if code != wantCode && !strings.Contains(r.Stderr, wantCode) {
		t.Fatalf("error code = %q stderr=%q detail=%q, want %q", code, r.Stderr, r.Detail, wantCode)
	}
}

func assertSnapshotUnchanged(t *testing.T, before, after domain.Snapshot) {
	t.Helper()
	br, ba, bt, bm := snapshotDigest(before)
	ar, aa, at, am := snapshotDigest(after)
	if br != ar || ba != aa || bt != at || bm != am {
		t.Fatalf("workspace mutated: before rev=%d agents=%d tasks=%d msgs=%d; after rev=%d agents=%d tasks=%d msgs=%d",
			br, ba, bt, bm, ar, aa, at, am)
	}
}

func findTask(snap domain.Snapshot, id domain.TaskID) *domain.Task {
	for i := range snap.Tasks {
		if snap.Tasks[i].ID == id {
			return &snap.Tasks[i]
		}
	}
	return nil
}

func findMessageInSnapshot(snap domain.Snapshot, id domain.MessageID) *domain.Message {
	for i := range snap.Messages {
		if snap.Messages[i].MessageID == id {
			return &snap.Messages[i]
		}
	}
	return nil
}

// assertMessageIdentitySeparate is H101-119's guard (Stanley): the
// differing-sender fixture (ui04_test.go's sendMessage helper) proves a
// submitted SenderAgentID may differ from the caller without rejection,
// but nothing previously asserted the two are actually PRESERVED as
// distinct values — SenderAgentID could be silently coerced into
// Provenance.ClaimedAgentID, or the reverse, and this suite would stay
// green. Takes a domain.Message directly so the same check applies to
// both the canonical harness snapshot and throwaway's own typed message
// response (both are domain.Message; only the CLI's text rendering needs
// a separate parser, see assertCLIMessageIdentity below).
func assertMessageIdentitySeparate(t *testing.T, msg domain.Message, wantSender, wantCaller, wantRecipient domain.AgentID) {
	t.Helper()
	if msg.SenderAgentID != wantSender {
		t.Fatalf("message SenderAgentID = %q, want %q (submitted sender must not be coerced into the caller)", msg.SenderAgentID, wantSender)
	}
	if msg.Provenance.ClaimedAgentID != wantCaller {
		t.Fatalf("message Provenance.ClaimedAgentID = %q, want %q (bound caller must not be coerced into the submitted sender)", msg.Provenance.ClaimedAgentID, wantCaller)
	}
	if msg.RecipientAgentID != wantRecipient {
		t.Fatalf("message RecipientAgentID = %q, want %q", msg.RecipientAgentID, wantRecipient)
	}
	if msg.Provenance.IdentityVerification != domain.IdentityUnverified {
		t.Fatalf("message Provenance.IdentityVerification = %q, want %q", msg.Provenance.IdentityVerification, domain.IdentityUnverified)
	}
}

// cliMessageFieldValue extracts the value on one labeled line of
// `harnessing message` output, associating a value with its own field
// rather than searching the whole output for a bare substring — Stanley's
// explicit requirement, since a substring search cannot tell "Sender: X"
// apart from "Recorded by: X" if X ever appears in both.
func cliMessageFieldValue(t *testing.T, output, label string) string {
	t.Helper()
	idx := strings.Index(output, label)
	if idx == -1 {
		t.Fatalf("message output missing field %q: %s", label, output)
	}
	line := output[idx+len(label):]
	if nl := strings.IndexByte(line, '\n'); nl != -1 {
		line = line[:nl]
	}
	line = strings.TrimSpace(line)
	if paren := strings.Index(line, " ("); paren != -1 {
		line = line[:paren]
	}
	return line
}

func assertCLIMessageIdentity(t *testing.T, output string, wantSender, wantCaller, wantRecipient domain.AgentID) {
	t.Helper()
	if got := cliMessageFieldValue(t, output, "  Sender:"); got != string(wantSender) {
		t.Fatalf("CLI message Sender field = %q, want %q: %s", got, wantSender, output)
	}
	if got := cliMessageFieldValue(t, output, "  Recipient:"); got != string(wantRecipient) {
		t.Fatalf("CLI message Recipient field = %q, want %q: %s", got, wantRecipient, output)
	}
	if got := cliMessageFieldValue(t, output, "  Recorded by:"); got != string(wantCaller) {
		t.Fatalf("CLI message Recorded by field = %q, want %q: %s", got, wantCaller, output)
	}
	if !strings.Contains(output, "unverified") {
		t.Fatalf("message output must mark identity unverified: %s", output)
	}
}

func assertUI04EndState(t *testing.T, snap domain.Snapshot) {
	t.Helper()
	want := expected.UI04CycleEnd
	if snap.Revision != want.WorkspaceRevision {
		t.Fatalf("workspace revision = %d, want %d (boundaries H101-94 UI-04 final cut)", snap.Revision, want.WorkspaceRevision)
	}
	if len(snap.Agents) != want.AgentCount {
		t.Fatalf("agent count = %d, want %d", len(snap.Agents), want.AgentCount)
	}
	task := findTask(snap, domain.TaskID(expected.TaskID))
	if task == nil {
		t.Fatal("task t1 missing from snapshot")
	}
	if task.Status != want.TaskStatus {
		t.Fatalf("task status = %q, want %q (AwaitingReview must not render as Done)", task.Status, want.TaskStatus)
	}
	if task.Title != want.TaskTitle {
		t.Fatalf("task title = %q, want %q", task.Title, want.TaskTitle)
	}
	if task.AssigneeID != want.TaskAssignee {
		t.Fatalf("task assignee = %q, want %q", task.AssigneeID, want.TaskAssignee)
	}
	if task.CurrentResultID == nil || *task.CurrentResultID != want.CurrentResultID {
		t.Fatalf("task result = %v, want %q", task.CurrentResultID, want.CurrentResultID)
	}
}

func assertDisclosure(t *testing.T, stderr string) {
	t.Helper()
	if !stderrHasDisclosure(stderr) {
		t.Fatalf("stderr missing ADR 0002 disclosure: %q", stderr)
	}
}

func assertMessageDeliveryFacts(t *testing.T, output string, wantAck bool) {
	t.Helper()
	for _, field := range []string{"Queued:", "Published:", "Processed:"} {
		if !strings.Contains(output, field) {
			t.Fatalf("message output missing delivery fact %q: %s", field, output)
		}
	}
	if wantAck {
		if strings.Contains(output, "Acknowledged: (absent)") {
			t.Fatalf("expected acknowledged message, got absent: %s", output)
		}
		if !strings.Contains(output, "Acknowledged:") {
			t.Fatalf("message output missing acknowledgement: %s", output)
		}
	} else if !strings.Contains(output, "Acknowledged: (absent)") {
		t.Fatalf("pending message must show absent acknowledgement: %s", output)
	}
	if !strings.Contains(output, "Recorded by:") {
		t.Fatalf("message output missing provenance Recorded by: %s", output)
	}
	if !strings.Contains(output, "unverified") {
		t.Fatalf("message output must mark identity unverified: %s", output)
	}
}

func parseThrowawayReceipt(t *testing.T, raw json.RawMessage) *domain.Receipt {
	t.Helper()
	var rec domain.Receipt
	if err := json.Unmarshal(raw, &rec); err != nil {
		t.Fatalf("unmarshal receipt: %v raw=%s", err, string(raw))
	}
	return &rec
}
