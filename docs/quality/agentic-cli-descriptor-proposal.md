# Agentic CLI descriptor contract v2

Status: **H101-154 revision, proposed for product consumption.** Normative
schemas are under `schemas/agentic-cli/`.

Each descriptor has one of two explicit forms:

- `deferred: false` with a `toolSpec` describing the selected executable,
  bounded version probe, direct tool argument vector, context transport,
  authentication detection strategy, and typed diagnostic hints.
- `deferred: true` with a reason and owner-deferral reference. No executable is
  looked up or invoked.

The descriptor is tool data, not policy. It cannot grant profile approval,
caller authority, credentials, retries, budgets, process containment, or cycle
completion. Product owns the managed start command and authoritative task,
message, result, review, and restart records. The runner writes a separate
`evidence` manifest containing observed executable identity/version, descriptor
digest, product revision, command, identifiers, bounded diagnostics, and the
outcome. Planned human interventions remain under `planned`; they are never
copied into observed evidence as if they happened.

Invocation arguments use `{literal: ...}` or `{contextSlot: ...}` elements.
There is no unrestricted formatting, shell expression, embedded executable, or
tool-name branch. Adding a supported tool is a new descriptor file. A genuinely
new capability requires a versioned contract change rather than a special case.

The common scenario is `h101-147-full-cycle` version `1`. Tool output can
diagnose authentication, connectivity, or unsupported invocation, but cannot
prove participation or completion. Completion requires authoritative product
records associated with the expected workspace, task, agents, and run.

Claude Code currently has a proposed direct `claude -p` invocation shape; its
managed product path remains unavailable in the shipped CLI. Codex and Cursor
Agent are explicit owner deferrals because no documented, tested context-bound
invocation was available for this work. No provider calls, authentication,
installation, or configuration changes are part of this contract.
