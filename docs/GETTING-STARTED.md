# Getting started with Harnessing 101

This guide takes you from a clean clone to one completed task with two agent
identities and one human reviewer. You will assign work, record a handoff,
acknowledge it, record and resolve a blocker, report a result, and accept it.

The workflow was run against commit `7401b52` on 2026-09-21. Every documented
step completed with the output shown below.

## What this guide does—and does not—connect

This is the canonical first-run workflow: **you run the Harnessing 101 command-line
interface (CLI) yourself** and use `agent-one` and `agent-two` as local identities.
No external agent tool has been validated as the canonical file-protocol client.
Registering an identity does not connect, authenticate, start, observe, stop, or
restrict an agent process.

If you involve real agent sessions, start and stop them yourself and give each
session only the relevant commands. Harnessing 101 records what a command claims;
it does not prove which person or program issued it. Treat agent-supplied content
as unverified.

## 1. Clone and build

Workspace operation is supported on macOS Apple Silicon (`darwin/arm64`) and
Linux x86-64 (`linux/amd64`) on native local storage. Install Go 1.27.1 or later,
then run:

```sh
git clone https://github.com/rafaelcalves/harnessing-101.git
cd harnessing-101
mkdir -p bin
go build -o ./bin/harnessing ./cmd/harnessing
./bin/harnessing version
```

Expected final line:

```text
harnessing dev
```

The unreleased local build reports version `dev`. Every invocation also prints
the three-line local-only and unverified-content disclosure to standard error.
The examples below omit that repeated disclosure but show the command result.

Create a workspace outside the source clone, then reuse these two shell
variables for the rest of the guide:

```sh
H="./bin/harnessing"
mkdir -p "$HOME/harnessing-workspaces"
WORKSPACE="$(mktemp -d "$HOME/harnessing-workspaces/getting-started.XXXXXX")"
printf 'Workspace: %s\n' "$WORKSPACE"
```

`mktemp` gives this run a new empty directory. Keeping workspace data outside
the clone prevents task and message content from appearing as untracked Git
files and later being published by an accidental `git add -A`. For real work,
choose another durable path outside a source repository. Do not reuse a
populated path for this guide: its request identifiers are idempotency keys, and
replaying one returns its original receipt.

## 2. Understand the three identities

- `agent-one` owns and reports the task.
- `agent-two` receives and acknowledges the handoff message.
- `owner` is the human reviewer. The repeated `-reviewer owner` flag grants
  review authority for that invocation; it does **not** register an agent.

Registration makes an identity available for task assignment and message
routing. Reviewer authority comes only from the command line when the workspace
opens. Registering `owner` would not grant review authority, and `owner` does not
need a registry record unless it will also be a task assignee or message endpoint.

## 3. Register the two agent identities

```sh
$H register -workspace "$WORKSPACE" -workspace-id demo \
  -reviewer owner -caller owner -request-id req-register-agent-one \
  -agent agent-one -display-name "Agent One"

$H register -workspace "$WORKSPACE" -workspace-id demo \
  -reviewer owner -caller owner -request-id req-register-agent-two \
  -agent agent-two -display-name "Agent Two"
```

Expected receipts:

```text
harnessing register: OK (request req-register-agent-one, workspace revision 1)
harnessing register: OK (request req-register-agent-two, workspace revision 2)
```

## 4. Create, assign, and start a task

The `-assignee` flag assigns the task when it is created.

```sh
$H create -workspace "$WORKSPACE" -workspace-id demo \
  -reviewer owner -caller owner -request-id req-create-task \
  -task task-1 -title "Review the startup guide" -assignee agent-one

$H transition -workspace "$WORKSPACE" -workspace-id demo \
  -caller agent-one -request-id req-start-task \
  -task task-1 -from Todo -to Doing
```

Expected receipts:

```text
harnessing create: OK (request req-create-task, workspace revision 3)
harnessing transition: OK (request req-start-task, workspace revision 4)
```

## 5. Send, inspect, and acknowledge a handoff

Send a task-linked request from `agent-one` to `agent-two`:

```sh
$H send -workspace "$WORKSPACE" -workspace-id demo \
  -caller agent-one -request-id req-send-handoff \
  -message msg-1 -sender agent-one -recipient agent-two -kind Request \
  -body "Please review the startup guide." -task task-1
```

Expected receipt:

```text
harnessing send: OK (request req-send-handoff, workspace revision 5)
```

Inspect pending messages for the recipient:

```sh
$H messages -workspace "$WORKSPACE" -workspace-id demo -recipient agent-two
```

The timestamps vary, but the expected record is:

```text
Pending messages: 1
Message msg-1
  Sender:      agent-one (claimed routing claim; identity unverified)
  Recipient:   agent-two
  Kind:        Request
  Body:        Please review the startup guide.
  Task:        task-1
  Queued:      <timestamp>
  Published:   <timestamp>
  Processed:   <timestamp>
  Acknowledged: (absent)
  Recorded by: agent-one (claimed sender, command, unverified)
```

Record `agent-two`'s acknowledgement:

```sh
$H ack -workspace "$WORKSPACE" -workspace-id demo \
  -caller agent-two -request-id req-ack-handoff -message msg-1
```

Expected receipt:

```text
harnessing ack: OK (request req-ack-handoff, workspace revision 8)
```

The revision advances from 5 to 8 because publishing and processing the message
are recorded workspace changes. Workspace revision is not a command count.
Acknowledgement proves only that it was recorded; it does not prove that the
requested work happened.

## 6. Record and resolve a blocker

```sh
$H transition -workspace "$WORKSPACE" -workspace-id demo \
  -caller agent-one -request-id req-block-task \
  -task task-1 -from Doing -to Blocked \
  -reason "Waiting for Agent Two review."

$H transition -workspace "$WORKSPACE" -workspace-id demo \
  -caller agent-one -request-id req-resume-task \
  -task task-1 -from Blocked -to Doing
```

Expected receipts:

```text
harnessing transition: OK (request req-block-task, workspace revision 9)
harnessing transition: OK (request req-resume-task, workspace revision 10)
```

Read the task before reporting a result:

```sh
$H task -workspace "$WORKSPACE" -workspace-id demo task-1
```

The important fields are:

```text
Task task-1
  Status:     Doing
  Assignee:   agent-one
  Revision:   4
  ResultID:   (none)
  Reporter:   agent-one (claimed reporter, unverified)
```

## 7. Report and accept the result

`-expected-revision` is the **task revision** shown by `harnessing task`, not the
workspace revision in a write receipt. Message delivery can advance the workspace
revision without changing the task revision.

```sh
$H report -workspace "$WORKSPACE" -workspace-id demo \
  -caller agent-one -request-id req-report-result \
  -task task-1 -result result-1 -expected-revision 4 \
  -summary "Startup guide reviewed." -artifact docs/GETTING-STARTED.md
```

Expected receipt:

```text
harnessing report: OK (request req-report-result, workspace revision 11)
```

Read the task again. It should show `Status: AwaitingReview`, `Revision: 5`, and
`ResultID: result-1`. Use that new task revision to accept the result:

```sh
$H accept -workspace "$WORKSPACE" -workspace-id demo \
  -reviewer owner -caller owner -request-id req-accept-result \
  -task task-1 -result result-1 -expected-revision 5
```

Expected receipt:

```text
harnessing accept: OK (request req-accept-result, workspace revision 12)
```

The `-reviewer owner` flag is load-bearing: `-caller owner` alone does not grant
human-review authority.

## 8. Reopen and verify the durable result

Each command opens and closes the workspace, so this final read is already a
reopen from disk:

```sh
$H task -workspace "$WORKSPACE" -workspace-id demo task-1
$H message -workspace "$WORKSPACE" -workspace-id demo msg-1
```

The task's important final fields are:

```text
Task task-1
  Status:     Done
  Assignee:   agent-one
  Revision:   6
  ResultID:   result-1
  Reporter:   owner (claimed reporter, unverified)
```

The message should show:

```text
Message msg-1
  Sender:      agent-one (claimed routing claim; identity unverified)
  Recipient:   agent-two
  Acknowledged: <timestamp> by agent-two
  Recorded by: agent-one (claimed sender, command, unverified)
```

## What is written on disk

Everything is below the directory passed to `-workspace`:

```text
getting-started.<random>/
├── state.json
└── mailbox/
    └── agent-two/
        ├── inbox/
        └── archive/
            └── msg-1.json
```

- `state.json` holds the workspace snapshot and request receipts used for
  idempotent replay. It includes tasks, results, registered identities, messages,
  delivery facts, and acknowledgements. Use the CLI rather than editing it.
- `mailbox/<recipient>/inbox/` is the local delivery queue. A processed envelope
  moves to `archive/`; it can move too quickly to observe in `inbox/` during this
  walkthrough.
- `.lock` can exist while a process has the workspace open. Do not delete it based
  on its age. Follow [workspace lock troubleshooting](TROUBLESHOOTING.md) if an
  open reports `Busy`.
- `bin/harnessing` is the local executable you built. It is outside the workspace
  and ignored by Git.

Harnessing 101 makes no product-initiated network connection. A real agent tool
may connect to its configured provider and may send content it can read; you are
responsible for that tool, its credentials, permissions, and destinations.

## Deliberate limits, not setup bugs

- `SenderAgentID` is an unverified routing claim by design. It may differ from
  `-caller`; neither field proves authorship.
- Identity verification remains `Unverified` in this phase. Do not interpret a
  display name, sender field, or registration record as authentication.
- Process operations `StartRun`, `StopRun`, and `SetRunBudget` are intentionally
  `Unsupported` in the current product surface, and the CLI exposes no commands
  for them. Phase 3 may implement them later; this guide does not promise that
  work or a delivery date.
- Nothing keeps running between CLI commands. Harnessing 101 does not supervise
  your agent sessions; you remain responsible for starting and stopping them.

For command syntax, run `./bin/harnessing help`. For workspace errors, see
[Troubleshooting](TROUBLESHOOTING.md).
