package adaptercontract_test

import (
	"context"

	"github.com/rafaelcalves/harnessing-101/internal/core/domain"
)

// Driver is the black-box entry point for one frontend adapter.
type Driver interface {
	Name() string
	Invoke(ctx context.Context, env WorkspaceEnv, call Call) Result
	QueryTask(ctx context.Context, env WorkspaceEnv, taskID domain.TaskID) Result
}

// WorkspaceEnv is an isolated on-disk workspace both adapters share the shape of.
type WorkspaceEnv struct {
	Root        string
	WorkspaceID domain.WorkspaceID
	Reviewers   []domain.AgentID
}

// Call is one adapter interaction. Exactly one of CLIArgs or ThrowawayEnvelope
// is set depending on the driver.
type Call struct {
	Caller             domain.AgentID
	CLIArgs            []string
	ThrowawayOperation string
	ThrowawayPayload   []byte
}

// Result is the normalized observation from one adapter invocation.
type Result struct {
	ExitCode int
	Stdout   string
	Stderr   string
	OK       bool
	Code     string
	Detail   string
	Receipt  *domain.Receipt
}
