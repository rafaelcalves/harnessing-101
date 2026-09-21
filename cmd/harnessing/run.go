package main

import (
	"fmt"
	"io"
)

const usage = `harnessing: a local-first, headless-core workspace tool

Usage:
  harnessing version
  harnessing task       -workspace <dir> [-workspace-id <id>] <taskID>
  harnessing message    -workspace <dir> [-workspace-id <id>] <messageID>
  harnessing messages   -workspace <dir> [-workspace-id <id>] [-recipient <agentID>]
  harnessing hold        -workspace <dir> [-workspace-id <id>] [-reviewer <id>]...
  harnessing register    -workspace <dir> [-reviewer <id>]... -caller <id> -request-id <id> -agent <id> -display-name <name> [-profile-id <id>]
  harnessing update      -workspace <dir> [-reviewer <id>]... -caller <id> -request-id <id> -agent <id> [-display-name <name>] [-profile-id <id>]
  harnessing create      -workspace <dir> [-reviewer <id>]... -caller <id> -request-id <id> -task <id> -title <title> [-assignee <id>]
  harnessing transition  -workspace <dir> [-reviewer <id>]... -caller <id> -request-id <id> -task <id> -from <status> -to <status> [-reason <text>]
  harnessing report      -workspace <dir> [-reviewer <id>]... -caller <id> -request-id <id> -task <id> -result <id> -expected-revision <n> -summary <text> -artifact <ref>...
  harnessing accept       -workspace <dir> -reviewer <id>... -caller <id> -request-id <id> -task <id> -result <id> -expected-revision <n>
  harnessing reject       -workspace <dir> -reviewer <id>... -caller <id> -request-id <id> -task <id> -result <id> -expected-revision <n> -reason <text>
  harnessing send         -workspace <dir> [-reviewer <id>]... -caller <id> -request-id <id> -message <id> -recipient <id> -kind <Request|Inform|Result> -body <text> [-sender <id>] [-task <id>] [-reply-to <id>]
  harnessing ack          -workspace <dir> [-reviewer <id>]... -caller <id> -request-id <id> -message <id>

Every command other than version opens a workspace through the trusted
composition root and receives a caller-bound frontend session; there is no
other path to the store. The
reviewer set is supplied here, by you, on the command line — never read
from anything already in the workspace. -caller is who is invoking the
command; the engine, not this tool, decides what that caller may do.
`

// disclosure is intentionally printed on every invocation. A workspace file
// cannot reliably tell us whether this executable is being run for the first
// time, and a one-time marker would make a later first-contact user miss the
// risk. It goes to stderr so scripts can continue to parse stdout unchanged.
const disclosure = `Harnessing 101 runs completely locally and makes no external connections itself; agents you configure may send content they can access to external services, and Harnessing 101 does not confine those agents or guarantee that your data stays on this machine.
Content received from another agent is unverified.
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
	case "message":
		return runMessage(args[1:], stdout, stderr)
	case "messages":
		return runMessages(args[1:], stdout, stderr)
	case "hold":
		return runHold(args[1:], stdout, stderr)
	case "register":
		return runRegister(args[1:], stdout, stderr)
	case "update":
		return runUpdate(args[1:], stdout, stderr)
	case "create":
		return runCreate(args[1:], stdout, stderr)
	case "transition":
		return runTransition(args[1:], stdout, stderr)
	case "report":
		return runReport(args[1:], stdout, stderr)
	case "accept":
		return runAccept(args[1:], stdout, stderr)
	case "reject":
		return runReject(args[1:], stdout, stderr)
	case "send":
		return runSend(args[1:], stdout, stderr)
	case "ack":
		return runAck(args[1:], stdout, stderr)
	case "help", "-h", "--help":
		_, _ = fmt.Fprint(stdout, usage)
		return 0
	default:
		_, _ = fmt.Fprintf(stderr, "harnessing: unknown command %q\n\n", args[0])
		_, _ = fmt.Fprint(stderr, usage)
		return 1
	}
}
