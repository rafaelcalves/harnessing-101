package adaptercontract_test

import (
	"bytes"
	"context"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/rafaelcalves/harnessing-101/internal/core/domain"
)

func moduleRoot(t *testing.T) string {
	t.Helper()
	out, err := exec.Command("go", "list", "-m", "-f", "{{.Dir}}").Output()
	if err != nil {
		t.Fatalf("go list module root: %v", err)
	}
	return strings.TrimSpace(string(out))
}

type cliDriver struct {
	binary string
}

func newCLIDriver(t *testing.T) *cliDriver {
	t.Helper()
	if runtime.GOOS != "darwin" && runtime.GOOS != "linux" {
		t.Skip("CLI contract tests require linux or darwin")
	}
	binary := filepath.Join(t.TempDir(), "harnessing-contract")
	build := exec.Command("go", "build", "-o", binary, "./cmd/harnessing")
	build.Dir = moduleRoot(t)
	if output, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build harnessing: %v\n%s", err, output)
	}
	return &cliDriver{binary: binary}
}

func (d *cliDriver) Name() string { return "cli" }

func (d *cliDriver) Invoke(ctx context.Context, env WorkspaceEnv, call Call) Result {
	if len(call.CLIArgs) == 0 {
		return Result{ExitCode: 1, Stderr: "missing CLI command"}
	}
	args := joinCLI(cliWorkspaceArgs(env), call.CLIArgs...)
	return d.run(ctx, args)
}

func (d *cliDriver) QueryTask(ctx context.Context, env WorkspaceEnv, taskID domain.TaskID) Result {
	return d.Invoke(ctx, env, Call{CLIArgs: []string{"task", string(taskID)}}) // caller empty for read-only task query
}

func (d *cliDriver) run(ctx context.Context, args []string) Result {
	cmd := exec.CommandContext(ctx, d.binary, args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	code := 0
	if err != nil {
		code = 1
		if exitErr, ok := err.(*exec.ExitError); ok {
			code = exitErr.ExitCode()
		}
	}
	out := stdout.String()
	rec := parseCLIReceipt(out)
	return Result{
		ExitCode: code,
		Stdout:   out,
		Stderr:   stderr.String(),
		OK:       code == 0,
		Receipt:  rec,
	}
}

func cliWorkspaceArgs(env WorkspaceEnv) []string {
	args := []string{
		"-workspace", env.Root,
		"-workspace-id", string(env.WorkspaceID),
	}
	for _, r := range env.Reviewers {
		args = append(args, "-reviewer", string(r))
	}
	return args
}

func joinCLI(base []string, extra ...string) []string {
	out := make([]string, 0, len(base)+len(extra))
	out = append(out, base...)
	out = append(out, extra...)
	return out
}

func stderrHasDisclosure(stderr string) bool {
	return strings.Contains(stderr, "does not start, observe, or restrict any agent process")
}
