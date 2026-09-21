// Package assembly is the trusted CLI composition root. It is deliberately
// small: it opens the workspace, fixes reviewer and caller policy, binds a
// FrontendSession, and owns Close. Presentation code receives only api.
package assembly

import (
	"context"
	"errors"
	"fmt"
	"io"

	"github.com/rafaelcalves/harnessing-101/internal/api"
	"github.com/rafaelcalves/harnessing-101/internal/core/domain"
	"github.com/rafaelcalves/harnessing-101/internal/host"
)

func WithSession(stderr io.Writer, root string, workspaceID domain.WorkspaceID, reviewers []domain.AgentID, caller domain.AgentID, cmdName string, fn func(context.Context, api.FrontendSession) int) int {
	if root == "" {
		_, _ = fmt.Fprintf(stderr, "harnessing %s: -workspace is required\n", cmdName)
		return 1
	}
	caps, err := host.Open(root, workspaceID, reviewers)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "harnessing %s: %s\n", cmdName, describeError(err))
		return 1
	}
	session := host.BindFrontendSession(caps, caller)
	code := fn(context.Background(), session)
	if closeErr := caps.Close(); closeErr != nil {
		_, _ = fmt.Fprintf(stderr, "harnessing %s: workspace did not close cleanly: %s\n", cmdName, describeError(closeErr))
		if code == 0 {
			return 1
		}
	}
	return code
}

func describeError(err error) string {
	var derr *domain.Error
	if errors.As(err, &derr) {
		if derr.Detail == "" {
			return string(derr.Code)
		}
		return fmt.Sprintf("%s: %s", derr.Code, derr.Detail)
	}
	return err.Error()
}
