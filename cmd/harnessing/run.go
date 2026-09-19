package main

import (
	"fmt"
	"io"
)

const usage = `harnessing: a local-first, headless-core workspace tool

Usage:
  harnessing version
  harnessing task -workspace <dir> [-workspace-id <id>] [-reviewer <agentID>]... <taskID>

Every command other than version opens a workspace through
internal/host.Capabilities; there is no other path to the store. The
reviewer set is supplied here, by you, on the command line — never read
from anything already in the workspace.
`

// disclosure is intentionally printed on every invocation. A workspace file
// cannot reliably tell us whether this executable is being run for the first
// time, and a one-time marker would make a later first-contact user miss the
// risk. It goes to stderr so scripts can continue to parse stdout unchanged.
const disclosure = `Harnessing 101 runs completely locally and makes no external connections itself; agents you configure may send content they can access to external services, and Harnessing 101 does not confine those agents or guarantee that your data stays on this machine.
The product itself does not yet start, observe, or restrict agents; you start agents by hand in the current phases, and content received from another agent is unverified.
Harnessing 101 does not start, observe, or restrict any agent process in this phase. Nothing here confirms which program produced this content.`

// run is main's testable body: no os.Exit, no direct os.Args/os.Stdout
// reference, so tests can assert on exit codes and captured output.
func run(args []string, stdout, stderr io.Writer) int {
	_, _ = fmt.Fprintln(stderr, disclosure)
	if len(args) == 0 {
		_, _ = fmt.Fprint(stderr, usage)
		return 1
	}

	switch args[0] {
	case "version", "--version":
		_, _ = fmt.Fprintln(stdout, "harnessing "+version)
		return 0
	case "task":
		return runTask(args[1:], stdout, stderr)
	case "help", "-h", "--help":
		_, _ = fmt.Fprint(stdout, usage)
		return 0
	default:
		_, _ = fmt.Fprintf(stderr, "harnessing: unknown command %q\n\n", args[0])
		_, _ = fmt.Fprint(stderr, usage)
		return 1
	}
}
