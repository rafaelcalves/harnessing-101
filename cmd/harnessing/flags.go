package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"

	"github.com/rafaelcalves/harnessing-101/internal/core/domain"
	"github.com/rafaelcalves/harnessing-101/internal/host"
)

// agentIDList collects a repeatable -reviewer flag into an ordered list
// of domain.AgentID, so the operator can name zero or more reviewers on
// one command line. This is the only source host.Open ever sees for
// human-review authority — no env var, no config file, no default.
type agentIDList []domain.AgentID

func (l *agentIDList) String() string { return fmt.Sprint(*l) }

func (l *agentIDList) Set(value string) error {
	if value == "" {
		return fmt.Errorf("-reviewer must not be empty")
	}
	*l = append(*l, domain.AgentID(value))
	return nil
}

// stringList collects a repeatable flag into an ordered []string, used
// for -artifact.
type stringList []string

func (l *stringList) String() string { return fmt.Sprint(*l) }

func (l *stringList) Set(value string) error {
	*l = append(*l, value)
	return nil
}

// optionalString distinguishes "flag not given" from "flag given as
// empty": Get returns nil unless Set was actually called. UpdateAgent's
// patch semantics need this — leaving a field alone must not look like
// clearing it to empty.
type optionalString struct {
	value *string
}

func (o *optionalString) String() string {
	if o.value == nil {
		return ""
	}
	return *o.value
}

func (o *optionalString) Set(value string) error {
	o.value = &value
	return nil
}

func (o *optionalString) Get() *string { return o.value }

// workspaceFlags is the trio every command that touches a workspace
// needs: root, workspace ID, and the reviewer set. Adding them to a
// FlagSet is one call so no command can accidentally omit -reviewer's
// wiring or reach for a shortcut source instead.
type workspaceFlags struct {
	root      *string
	id        *string
	reviewers agentIDList
}

func addWorkspaceFlags(fs *flag.FlagSet) *workspaceFlags {
	wf := &workspaceFlags{}
	wf.root = fs.String("workspace", "", "path to the workspace root (required)")
	wf.id = fs.String("workspace-id", "default", "workspace identifier stored alongside state")
	fs.Var(&wf.reviewers, "reviewer", "agent ID with human-review authority for this invocation (repeatable)")
	return wf
}

// withCapabilities opens the workspace named by wf, runs fn, and always
// closes it before returning — on every exit path, including a failure
// inside fn. cmdName is only used to prefix stderr messages.
func withCapabilities(stderr io.Writer, wf *workspaceFlags, cmdName string, fn func(ctx context.Context, caps host.Capabilities) int) int {
	if *wf.root == "" {
		_, _ = fmt.Fprintf(stderr, "harnessing %s: -workspace is required\n", cmdName)
		return 1
	}

	caps, err := host.Open(*wf.root, domain.WorkspaceID(*wf.id), wf.reviewers)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "harnessing %s: %s\n", cmdName, describeError(err))
		return 1
	}
	closeFailed := false
	defer func() {
		if closeErr := caps.Close(); closeErr != nil {
			_, _ = fmt.Fprintf(stderr, "harnessing %s: workspace did not close cleanly: %s\n", cmdName, describeError(closeErr))
			closeFailed = true
		}
	}()

	code := fn(context.Background(), caps)
	if closeFailed && code == 0 {
		return 1
	}
	return code
}

// describeError renders a domain.Error with its stable code, so a script
// scraping stderr sees "NotFound"/"Denied"/"Conflict"/etc. rather than a
// bare human-readable sentence it would have to parse. Every command
// funnels its errors through this — write commands are where Denied and
// Conflict actually become reachable, and this is the one place that
// rendering is produced.
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

// printReceipt is the one success-rendering used by every write command:
// factual, no celebration, per definition.md's tone section.
func printReceipt(stdout io.Writer, cmdName string, receipt domain.Receipt) {
	_, _ = fmt.Fprintf(stdout, "harnessing %s: OK (request %s, workspace revision %d)\n", cmdName, receipt.RequestID, receipt.CommittedRevision)
}
