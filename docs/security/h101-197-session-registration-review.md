# H101-197 — Session-registration pairing security review

This discharges Stanley's registration spec's security-review construction prerequisite (docs/architecture/h101-171-session-registration-protocol.md) for **attachment-first scope only**, as amended at 94260c9. H101-198's deferred coexistence and exact-run managed bootstrap need their own review before activation — Stanley wrote that requirement into line 81 of the amended spec, this ruling does not extend to it.

No gate fires. Reviewed against the amended admission rules (lines 74, 76, 77, 57, 81 of the amended spec).

## Evidence-versus-authority match

Honest. The spec states its own limit plainly: "user-paired; tool identity unverified," never "verified Claude session" — no cryptographic identity claim. What that credential authorizes is proportionate to how weak it is: mailbox receipt and worker-capability task participation only, explicitly excluding process/execution control, profile administration, reviewer authority and delegation. Execution authority still requires the separate, stronger-evidence StartRun path. A weak credential getting a narrow grant is the right shape — the spec does not use pairing to buy authority pairing's evidence can't support.

## What must never be inferred from a registered session

Confirmed as complete and correct, not amended:

- Tool identity, from a pairing credential.
- Liveness or attention ("still thinking"), from a connected session.
- Participation, from a live connector, queued delivery, or a successful poll.
- Death, from heartbeat staleness — staleness must never auto-release the attachment slot.
- That a run's uncertainty is cleared, from registration existing — registration cannot serve as evidence that prior execution ended.

## Takeover and denial finding

No takeover path. One-attachment-per-agent and second-session-for-an-occupied-agent are both explicit `Conflict`, never a silent takeover. Replacement and revocation both require an explicit user action; nothing external can trigger either. The dual-admission rule (amended lines 74, 76, 77) closes the exposure that removing the earlier implicit case could have created: both directions are enforced (no new attachment while a managed run is Starting/Running/Stopping/RecoveryRequired; no new managed start while an attachment is Connected/Stale/Disconnected), rechecked at redemption, serialized against shared state so concurrent requests can't both win, and explicitly refuses a timeout-based takeover shortcut (line 57) or a managed-bootstrap exception.

Abandoned Stale or Disconnected registrations do create a real availability cost: they can block both re-pairing and managed start for that agent indefinitely until a user explicitly replaces the binding. This is disclosed plainly in the spec ("this conservative blocking can cost availability; timeout-based takeover is not an acceptable shortcut") and is an operational and product tradeoff for Kelly and Angela, not a security finding — a false or stuck block costs availability only, never integrity or confidentiality, so it does not gate construction.

## H101-195 confirmation

Recorded separately from the H101-197 review above. My gate condition on the fast-exit classification (docs/architecture/h101-195-graceful-capture-producer.md) was "if a relaxation is proposed." None was proposed. The ruling bars four exception paths — exit-zero, context-file, sibling-marker, and fixture-content inspection — from ever promoting fast exit to successful startup, not just the one relaxation I gated against. `TestSupervisor_FastExitIsSpawnFailed` keeps failing `/bin/true` unconditionally. No design review is owed; the gate is closed.
