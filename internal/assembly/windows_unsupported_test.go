//go:build windows

package assembly_test

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/rafaelcalves/harnessing-101/internal/api"
	"github.com/rafaelcalves/harnessing-101/internal/assembly"
	"github.com/rafaelcalves/harnessing-101/internal/core/domain"
)

// H101-101: this is the product-path half of Kelly's exit check — not
// windows support, a proof that the refusal is deliberate and stated
// rather than an accident. Only WithSession, the trusted composition
// root, ever opens a workspace (see session.go's own comment on that);
// version/help in cmd/harnessing never call it at all, so this asserts
// the one thing that actually needs proving: a real workspace operation
// on windows answers Unsupported, cleanly, through the same path every
// other command uses — not a panic, not a silent success.
func TestWithSession_Windows_WorkspaceOperationIsUnsupported(t *testing.T) {
	root := t.TempDir()
	var stderr bytes.Buffer

	code := assembly.WithSession(&stderr, root, "ws1", nil, "engineer", "create", func(ctx context.Context, s api.FrontendSession) int {
		t.Fatal("fn must not run: host.Open should fail before the session body ever executes on windows")
		return 0
	})

	if code == 0 {
		t.Fatalf("exit code = 0, want non-zero: a workspace operation must refuse cleanly on windows, not silently succeed")
	}
	if !strings.Contains(stderr.String(), string(domain.ErrUnsupported)) {
		t.Fatalf("stderr = %q, want it to name %s", stderr.String(), domain.ErrUnsupported)
	}
}
