package main

import (
	"context"
	"flag"
	"fmt"
	"io"

	"github.com/rafaelcalves/harnessing-101/internal/api"
	"github.com/rafaelcalves/harnessing-101/internal/core/domain"
)

func runRunOutputQuery(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("run-output", flag.ContinueOnError)
	fs.SetOutput(stderr)
	wf := addWorkspaceFlags(fs)
	if err := fs.Parse(args); err != nil {
		return 1
	}
	if fs.NArg() != 1 {
		_, _ = fmt.Fprintln(stderr, "harnessing run-output: exactly one runID argument is required")
		return 1
	}
	runID := domain.RunID(fs.Arg(0))
	return withSession(stderr, wf, "", "run-output", func(ctx context.Context, session api.FrontendSession) int {
		reader, ok := session.(interface {
			ReadOutput(context.Context, domain.RunID, uint64, int) (domain.RunOutput, error)
		})
		if !ok {
			_, _ = fmt.Fprintln(stderr, "harnessing run-output: output read is unsupported on this session")
			return 1
		}
		output, err := reader.ReadOutput(ctx, runID, 0, 0)
		if err != nil {
			_, _ = fmt.Fprintln(stderr, "harnessing run-output: "+describeError(err))
			return 1
		}
		_, _ = fmt.Fprintf(stdout, "RunOutput %s\nCapture status: %s\n", output.RunID, output.CaptureStatus)
		for _, chunk := range output.Chunks {
			_, _ = fmt.Fprintf(stdout, "Channel: %s\n", chunk.Channel)
			_, _ = stdout.Write(chunk.Bytes)
		}
		return 0
	})
}
