# H101-187 — Running recovery and ownership evidence

Architecture ruling, 2026-09-22. Scope: interpretation only; no implementation or gate change.

## 1. Conservative recovery is correct when ownership is lost

An inherited `Running` record must become `RecoveryRequired` when the new host cannot establish ownership and current outcome. This remains correct if the child survived. `Running` records a past successful launch; `RecoveryRequired` records present uncertainty, not observed death. Preserve the successful start operation and its historical observations. Do not fabricate `Exited`, kill the child, or release the agent's active-run exclusion.

The resulting agent block is a real availability cost, explicitly allowed by [H101-128's crash-recovery contract](h101-128-phase3-supervision.md#crash-recovery--item-4). Distinguishing live from dead is not required for this conservative slice. This does not establish that either native platform is incapable of stronger recovery; today's adapter lacks that proof.

“Every nonterminal run on reopen” needs qualification: definitely undispatched intents have a separate cancellation path, and a client attaching to a continuing host must not reconcile that host's own actively supervised runs as abandoned. Reconciliation belongs to acquisition of host ownership. With one-shot hosts and no transferable ownership proof, even a normal host departure can leave a live child uncertain on the next open. That limitation is accepted for this slice; it is not evidence of continuous supervision or budget enforcement.

## 2. Stronger evidence is permitted, not required for B5

The prohibition is adoption or signaling by process identifier (PID) alone. A product-started child is not exempt: identifiers can be reused, and persisted lineage does not establish present ownership.

A stronger recovery design must bind fresh evidence to the exact execution attempt, distinguish that process incarnation from an unrelated or reused identifier, and establish the continuing authority and mechanism to supervise the owned process group. Recorded incarnation identity plus a liveness check can contribute evidence; neither a raw PID check nor identity/liveness alone establishes the whole contract. A host-generation token helps reject stale claims but does not prove a child is alive or controllable. A surviving trusted supervisor with a verified attempt-bound handoff is another possible direction, not a mandated implementation.

Only the facts actually proved may be restored. Observing a live direct child alone cannot promise group ownership, safe termination, or restored budgets. Missing, stale, mismatched, or interrupted proof must remain conservative. A dead direct child alone also cannot establish group disappearance or a known exit outcome.

Any stronger proposal needs native evidence on both supported targets covering:

- A surviving original execution is correctly identified and supervision actually resumes.
- A dead child, surviving descendants, reused/unrelated identifiers, and wrong attempt or generation cannot produce a false ownership claim or unsafe signal.
- Ownership evidence lost during a crash, persistence failure, or recovery race falls back to uncertainty without duplicate spawn.
- Claimed stop and budget behavior works after recovery; unsupported guarantees remain explicit.

This ruling grants no administrative unblock path. [H101-173](../security/h101-173-recovery-resolution-boundary.md) still governs resolution and restart.

## 3. Dispatch ordering does not change

Durable start intent, operation and receipt precede a durable attempted-dispatch marker; both barriers precede the single spawn attempt. A stronger design may add evidence collection and persistence but cannot weaken that ordering. Successful launch evidence follows actual launch observation. A crash after dispatch but before identity evidence is durable remains ambiguous: never respawn to fill that gap.

## Consequence for D2 evidence

[H101-184](../quality/h101-184-item4-d2-acceptance-spec.md) requires a historical `Running` sighting before testing recovery. That precondition and the fresh host's conservative result are separate facts. The lock/query obstacle does not justify retaining unsupported `Running`, changing dispatch ordering, or treating `Starting` evidence as `Running` evidence. Kelly's H101-186 owns the admissible evidence ruling on today's product. Stronger recovery is not required merely to make B5 observable; any needed observation surface or lifecycle change must be explicit.
