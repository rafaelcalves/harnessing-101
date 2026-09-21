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

func parseThrowawayReceipt(t *testing.T, raw json.RawMessage) *domain.Receipt {
	t.Helper()
	var rec domain.Receipt
	if err := json.Unmarshal(raw, &rec); err != nil {
		t.Fatalf("unmarshal receipt: %v raw=%s", err, string(raw))
	}
	return &rec
}
