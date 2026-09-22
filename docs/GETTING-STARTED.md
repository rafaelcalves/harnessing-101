# Getting started with Harnessing 101

This guide takes you from a clean clone to one completed task with two agent
identities and one human reviewer. You will assign work, record a handoff,
acknowledge it, record and resolve a blocker, reject and replace a result, and
accept the replacement.

The manual workflow was run against commit `f2d99fb` on 2026-09-21. The Layer B
commands were rechecked on 2026-09-22 after the managed-start surface landed.
The observed outcomes and their limits are recorded below.

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

Windows workspace commands deliberately return `Unsupported`; only commands
that do not open a workspace, such as `help` and `version`, may run there. No
support claim is made for other operating systems, processor architectures, or
network and cloud-synchronised filesystems. This is a platform boundary, not a
lock-file problem or a weaker fallback mode.

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

## 7. Report, reject, rework, and accept the result

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
`ResultID: result-1`.

### Recover from a stale task revision

The following rejection deliberately uses stale task revision 4 to show the
failure and recovery path. It must fail and must not change the task:

```sh
$H reject -workspace "$WORKSPACE" -workspace-id demo \
  -reviewer owner -caller owner -request-id req-reject-result \
  -task task-1 -result result-1 -expected-revision 4 \
  -reason "Add the missing recovery steps."
EXIT_CODE=$?
printf 'exit status: %s\n' "$EXIT_CODE"
```

Expected command result, after the repeated disclosure:

```text
harnessing reject: Conflict: expectedTaskRevision is stale
exit status: 1
```

On this `Conflict`, do not guess the new revision and do not switch to the
workspace revision. Re-read the task:

```sh
$H task -workspace "$WORKSPACE" -workspace-id demo task-1
```

It remains `AwaitingReview` at task `Revision: 5` with `ResultID: result-1`.
A `Conflict` did not commit, so retry the **same intended operation** with the
same request ID and the freshly read task revision. If you change the intended
operation or its payload, use a new request ID instead.

### Reject and replace the result

Retry the rejection with the current task revision:

```sh
$H reject -workspace "$WORKSPACE" -workspace-id demo \
  -reviewer owner -caller owner -request-id req-reject-result \
  -task task-1 -result result-1 -expected-revision 5 \
  -reason "Add the missing recovery steps."
```

Expected receipt:

```text
harnessing reject: OK (request req-reject-result, workspace revision 12)
```

Re-read the task:

```sh
$H task -workspace "$WORKSPACE" -workspace-id demo task-1
```

Rejection returns it to `Status: Doing` at task `Revision: 6`; it is not
complete. `result-1` remains in history as the rejected result. `agent-one` now
reports a replacement with a new result ID:

```sh
$H report -workspace "$WORKSPACE" -workspace-id demo \
  -caller agent-one -request-id req-report-rework \
  -task task-1 -result result-2 -expected-revision 6 \
  -summary "Startup guide reviewed with recovery steps." \
  -artifact docs/GETTING-STARTED.md
```

Expected receipt:

```text
harnessing report: OK (request req-report-rework, workspace revision 13)
```

Read the task once more:

```sh
$H task -workspace "$WORKSPACE" -workspace-id demo task-1
```

It should show `Status: AwaitingReview`, `Revision: 7`, and `ResultID: result-2`.
Accept that exact replacement:

```sh
$H accept -workspace "$WORKSPACE" -workspace-id demo \
  -reviewer owner -caller owner -request-id req-accept-result \
  -task task-1 -result result-2 -expected-revision 7
```

Expected receipt:

```text
harnessing accept: OK (request req-accept-result, workspace revision 14)
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
  Revision:   8
  ResultID:   result-2
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

## 9. Run the Layer B owner preflight

Layer B is the real-tool trial: the product, rather than a shell script, must
start an approved agent tool and record authoritative participation and task
completion. The current product cannot complete that trial yet. This preflight
records exactly where it stops without treating tool output as proof of work.

Before running it, install and authenticate Claude Code yourself. Harnessing 101
and this runner never install a tool, sign in, change credentials, or change
provider configuration.

From the repository root, after building `./bin/harnessing` and completing the
workspace setup and human assignment above, resolve the installed executable:

```sh
CLAUDE_BIN="$(command -v claude)"
test -x "$CLAUDE_BIN"
claude --version
```

### Approve the exact execution profile

The profile identifier passed to Layer B is not a label that creates itself. An
authorized human reviewer must record the executable, static arguments, working
directory, and participation-context transport before `start-run` may use it:

```sh
$H approve-profile -workspace "$WORKSPACE" -workspace-id demo \
  -reviewer owner -caller owner -request-id req-approve-claude-profile \
  -profile-id approved-claude-profile \
  -tool-executable "$CLAUDE_BIN" \
  -tool-argv-json '["-p"]' \
  -working-dir "$WORKSPACE" \
  -context-transport context-file
```

Following this guide exactly, the receipt is:

```text
harnessing approve-profile: OK (request req-approve-claude-profile, workspace revision 15)
```

The `-reviewer owner` flag is required: profile approval is a human decision.
`context-file` tells the process supervisor to write the run's workspace, task,
agent, peer, and run identifiers to a run-scoped JSON file and pass its path in
`HARNESSING_CONTEXT_FILE`. Approval is immutable for this profile ID; changing
the executable or approved shape requires a different profile ID.

### Attempt the managed run

Now run the Layer B driver:

```sh
python3 scripts/run-agentic-cli-cycle.py \
  --descriptor scripts/agentic-cli-manifests/claude-code.json \
  --harnessing ./bin/harnessing \
  --workspace "$WORKSPACE" \
  --workspace-id demo \
  --task-id task-1 \
  --agent-id agent-one \
  --peer-id agent-two \
  --profile-id approved-claude-profile \
  --run-id run-1 \
  --product-revision "$(git rev-parse --short HEAD)" \
  --report "$WORKSPACE/claude-code-layer-b.json"
EXIT_CODE=$?
printf 'exit status: %s\n' "$EXIT_CODE"
```

All flags shown are required. A relative `--harnessing` path such as
`./bin/harnessing` is supported; the runner resolves it before probing the
product. The report path may name a file that does not exist, but its parent
directory must already exist.

The result now depends on the installed tool's authentication and network
environment; a completed Layer B cycle is **not** expected or claimed here. In
the environment used to verify this guide, profile approval cleared the earlier
`Denied: profileID has no recorded approval event` failure and the product
reached the real Claude process. The outer execution policy then stopped the
attempt with:

```text
Network access to "https://api.anthropic.com:443" was blocked by policy.
```

That policy stop terminated the wrapper before it wrote
`claude-code-layer-b.json` or returned a runner result and exit code. Therefore
this verification observed **no runner result and no exit code to quote**. A
subsequent read showed `run-1` persisted as `Starting`; that is not
participation, completion, or a successful Layer B trial. Preserve the output
and workspace for diagnosis rather than retrying or describing the run as
successful.

On another machine, authentication or network behavior may produce a different
typed runner result. Whatever it returns, the JSON report records the exact tool
version, operating system, product revision, descriptor digest,
workspace/task/agent/profile/run identifiers, planned interventions, and
diagnostics. Do not replace that observed result with the outcome from this
guide.

The other version-2 descriptors are explicit owner deferrals:

- `scripts/agentic-cli-manifests/codex.json` returns `owner_deferred` with exit
  19.
- `scripts/agentic-cli-manifests/cursor-agent.json` returns `owner_deferred`
  with exit 19.

Those descriptors have no approved invocation shape. The runner records the
deferral and does not look up, version-probe, authenticate, or launch Codex or
Cursor Agent. Replace the Claude descriptor in the command only if you want to
record that expected deferral; it is not a tool trial.

## What is written on disk

Everything is below the directory passed to `-workspace`:

```text
getting-started.<random>/
├── claude-code-layer-b.json       # only when the runner returns normally
├── runs/
│   └── run-1.context.json         # after managed start reaches dispatch
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
- `claude-code-layer-b.json`, when present, is the Layer B preflight report from
  step 9. It is evidence about the attempted trial, not proof that an agent
  participated or completed work. An outer policy may terminate the wrapper
  before this file is written.
- `runs/run-1.context.json` is the narrow participation context written when a
  `context-file` profile reaches dispatch. Its presence does not prove that the
  tool read it or performed work.
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

## How to read project status and limits

Use the [current Phase 2 ruling](quality/definition-of-done.md#current-phase-2-ruling-2026-09-21)
for current Phase 2 status. The dated entries below that table are an append-only
review log: they were true for the revisions reviewed but are historical, not the
current verdict. Architecture decision records and proposed documents describe
decisions or intended constraints; by themselves they are not proof that a
capability ships. The [Phase 3 exit criteria](quality/phase3-exit-criteria.md)
describe work that must pass before Phase 3 exits, not features this guide claims
are already available.

**Satisfied with limit** means accepted for the named phase purpose while the
listed exclusions remain unproved and must not be described as supported
behaviour. Phase 2 item 3 (interface swappability) has that verdict. G5 is closed;
G1–G4 and G6–G7 remain. In plain language, the verdict does **not** claim:

- exact CLI punctuation, output stream placement, or the validation substring
  deny-list as a universal interface contract (G1);
- that observer events contain exactly one subject and no additional subjects
  (G2);
- a universal cursor encoding or expiry after every restart (G3);
- duplicate-free delivery or exactly one event per workspace revision (G4);
- mandatory nonempty error detail or a settled precedence between malformed
  requests and unsupported operations (G6); or
- that the walkthrough's individual task-revision numbers are architecture
  rules rather than fixture choreography (G7).

See the [full item 3 ruling](quality/h101-107-phase2-item3-ruling.md#verdict) for
the evidence and exact boundary. These limits do not change the commands in this
guide; they prevent a passing walkthrough from being advertised as proof of more
than it tested.

## Deliberate limits, not setup bugs

- `SenderAgentID` is an unverified routing claim by design. It may differ from
  `-caller`; neither field proves authorship.
- Identity verification remains `Unverified` in this phase. Do not interpret a
  display name, sender field, or registration record as authentication.
- Harnessing 101 does not guarantee that an agent is correct, that its result is
  safe, or that an acknowledgement means the requested work happened. The human
  reviewer remains responsible for the acceptance decision.
- Local records do not make agent file access safe. An agent retains the
  operating-system permissions of the process you started, and workspace files
  and messages may contain untrusted instructions or content.
- Harnessing 101 does not confine agent processes or enforce a provider egress
  allowlist. An agent may send readable content to destinations allowed by its
  separately configured tool or provider.
- Process operations `StartRun`, `StopRun`, and `SetRunBudget` are intentionally
  `Unsupported` in the current product surface, and the CLI exposes no commands
  for them. Phase 3 may implement them later; this guide does not promise that
  work or a delivery date.
- Nothing keeps running between CLI commands. Harnessing 101 does not supervise
  your agent sessions; you remain responsible for starting and stopping them.

For command syntax, run `./bin/harnessing help`. For workspace errors, see
[Troubleshooting](TROUBLESHOOTING.md).
