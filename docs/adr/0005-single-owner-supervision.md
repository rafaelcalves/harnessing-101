# ADR 0005 — Single-owner supervision and conservative process recovery

Status: **Proposed for review**, 2026-09-21. Stanley, H101-128. Phase 3 is authorized; this ADR is not an implementation or acceptance claim.

## Context

The shipped one-shot CLI composition opens the workspace, executes an interaction and closes its host. Phase 3 needs ongoing process supervision and budgets after an observer exits, while preserving one canonical state writer, no controller networking, caller-bound authority and the ten architectural roles. Starting an OS process cannot share the StateStore transaction. A crash between launch and recording its observation cannot be resolved by assuming the operation either never happened or completed successfully.

## Decision

Use an explicit continuing `harnessing serve` host holding the workspace lock. CLI and scripted-adapter sessions attach through a bounded local file transport implemented outside presentation/core. Transport files convey requests, not a second state store. A session is bound by host policy; message sender claims and payload reviewer flags confer no authority. Only host-admin shutdown terminates host lifetime; ordinary UI detach never does.

Persist approved immutable profile references, start intent, operation and effect-attempt identity before dispatch. Dispatch only after durability confirmation. Core owns policy and transitions; ProcessSupervisor owns OS effects/observations, OutputJournal owns persisted output, Clock owns monotonic wakeups. Keep process effects outside Commit callbacks. RunOutput remains separate from StateEvents.

On controller restart, reconcile rather than replay uncertain process starts. If ownership cannot be established, record RecoveryRequired, forbid automatic duplicate spawn and require explicit resolution. A matching request receipt is not proof that an external process started or stopped. This extends ADR 0004's external-effect distinction, not an exactly-once claim.

Use process-group supervision for supported, non-detaching execution profiles, subject to the explicit item-2 scope ruling requested in [the architecture document](../architecture/h101-128-phase3-supervision.md). Core budget enforcement drives the same observed termination path as user stop. An active host's hard budget is not a promise of instantaneous OS exit, trustworthy unreported token use, or enforcement surviving host death. The item-3 deadline wording needs Kelly's explicit disposition before implementation.

## Alternatives and consequences

Per-command child ownership would couple UI exit to accepted work and cannot meet background hosting. A TCP or Unix-socket daemon would broaden the inherited no-network dependency/transport boundary unnecessarily. A per-run guardian could retain stronger ownership and budget observation across controller death, but adds a second protocol and lifecycle; conservative RecoveryRequired is sufficient for the stated recovery obligation. Revisit a guardian if uninterrupted enforcement across controller crashes becomes required.

File transport adds framing, bounded intake, session permissions, response-loss and stale-host-generation handling. It is not an authentication boundary against a hostile same-user writer. Review its assembly authority and import containment, and preserve ADR 0004 errors/request recovery through the proxy. Native tests on both supported platforms must prove actual process and output effects; mocks only prove adapter/policy behavior.

This ADR does not supersede ADR 0002's owner risk acceptance: its required Phase 3 entry review remains explicit, as do manual-agent exposure and unverified historical content. It does not authorize task reassignment/retirement, Windows support, automatic restart or arbitrary escaped-process containment.

## Follow-up

God/Kelly resolve deadline/tree-scope wording; security/product/owner record the ADR 0002 entry disposition. Kevin implements the accepted slices; Kelly owns nine-item exit evidence. [H101-128](../architecture/h101-128-phase3-supervision.md) defines authority, budgets, observations, recovery, output and the build sequence. No code/tests changed or run.
