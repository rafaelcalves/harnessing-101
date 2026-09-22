# H101-152 real-tool Layer B runner

Status: **version-2 contract revision, pending managed start surface**.

The runner is `scripts/run-agentic-cli-cycle.py`. It accepts one version-2
descriptor and writes the selected tool, exact version output, operating
system, product revision, workspace/task/agent identifiers, approved profile
identifier, run identifier, planned interventions, and observed diagnostics in
separate JSON sections. It classifies these
outcomes without installing, authenticating, retrying, or changing provider
configuration:

- `missing_tool` (exit 20)
- `authentication_required` (exit 21)
- `network_egress_refused` (exit 22)
- `started_no_participation` (exit 23)
- `product_surface_unavailable` (exit 24)
- `unsupported_invocation` (exit 25)
- `spawn_failed` (exit 26)
- `cycle_completed` (exit 0)

`cycle_completed` is no longer inferred from tool output. A manual shell launch
cannot satisfy the Layer B contract; authoritative product records must establish
participation and completion.
The runner therefore exits `product_surface_unavailable` against the current
shipped CLI, whose help has no managed `start-run` command.

## Owner command

From the repository root, after setup and human assignment:

```sh
python3 scripts/run-agentic-cli-cycle.py \
  --descriptor scripts/agentic-cli-manifests/claude-code.json \
  --harnessing /path/to/harnessing \
  --workspace /path/to/demo-workspace \
  --workspace-id demo-workspace \
  --task-id task-a \
  --agent-id agent-a \
  --peer-id agent-b \
  --profile-id approved-claude-profile \
  --run-id run-1 \
  --product-revision packaged-revision \
  --report /tmp/h101-152-layer-b.json
```

The three descriptors are `claude-code.json`, `codex.json`, and
`cursor-agent.json`. Claude Code's local version preflight was observed as
`2.1.278 (Claude Code)`. Codex and Cursor Agent are explicit owner-deferral
descriptors; their invocations were not tested.

The descriptor shape is documented in
`docs/quality/agentic-cli-descriptor-proposal.md` and
`scripts/agentic-cli-manifest.schema.json`.
