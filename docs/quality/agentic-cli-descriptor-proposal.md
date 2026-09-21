# Agentic CLI descriptor proposal

Status: **Proposed, H101-152.** This is a manifest shape for Layer B real-tool
evidence under H101-147. It is not an accepted product contract yet.

The runner is intentionally descriptor-driven: adding a fourth agentic CLI should
add a descriptor, not a new branch in the runner. A descriptor says only what the
runner needs to discover and classify a tool; it does not store credentials,
provider account details, or a command that authenticates.

## Shape derived from the runner

Each descriptor contains:

- `schemaVersion`, `tool`, and `executable`: stable display name and executable
  lookup on `PATH`;
- `version.args` and `successExitCodes`: a local version check and its accepted
  exit codes;
- `auth.detectionMode`, `requiredPatterns`, `networkRefusedPatterns`, and
  `notes`: classification of the first real invocation without trying to log in;
- `invocation.managedStart.args` and `contextMode`: the product-managed start
  command and how the approved profile supplies workspace/task context;
- `participation.requiredSignals`, `completionSignals`, and `failureSignals`:
  evidence that the tool participated, completed the bounded cycle, and did not
  merely start;
- `humanInterventions` and `friction`: manifest fields required by Angela's
  evidence record.

Template values available to `managedStart.args` are `{workspace}`,
`{workspace_id}`, `{task_id}`, `{agent_id}`, `{peer_id}`, `{profile_id}`, and
`{run_id}`. The product remains responsible for profile approval, caller scope,
dispatch ordering, and durable records. The runner never hand-writes those
records and never turns `PATH` presence into approval. The required `--report`
path is written by the runner for every classified outcome; it includes the
descriptor SHA-256, owner-supplied product revision/commit, exact version output,
command, identifiers, and outcome.

## Authentication and failure policy

The descriptors use runtime-output classification because a portable,
non-authenticating status command is not established for all three tools. The
runner may detect “missing tool” and version failure locally. It classifies
authentication-required and provider-network refusal from the managed invocation
output, then stops. It never retries through another endpoint, opens a login flow,
supplies a key, or edits a provider profile.

`product_surface_unavailable` is an additional honest precondition result while
the shipped CLI lacks managed `start-run`; it prevents a manual shell launch from
being misreported as Layer B evidence. The runner also refuses to emit
`cycle_completed` unless both participation and completion signals are present.
Once the managed surface exists, the required five outcomes are: missing tool,
authentication required, network egress refused, started without participation,
and cycle completed.

## Named descriptors

- `scripts/agentic-cli-manifests/claude-code.json` — executable and version shape
  checked locally; managed invocation and participation are not yet tested.
- `scripts/agentic-cli-manifests/codex.json` — not tested here.
- `scripts/agentic-cli-manifests/cursor-agent.json` — not tested here.

The descriptors are proposals, not compatibility claims. A Layer B manifest must
record the exact tool version, target, approved profile revision, context supplied,
human interventions, task/message identifiers, and friction from an actual run.
