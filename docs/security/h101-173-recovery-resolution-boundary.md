# H101-173 — Recovery-resolution security boundary

Points 1–4 below gate any later restart/unblock card (recorded against H101-158, the admin-override CallerScope widening). No unblock code may ship until the gated card satisfies them, and I require seeing that card's design before code. This ruling covers no code and no CI gate; it states the boundary the unblock card must satisfy.

Source: Stanley's H101-169 ruling (RecoveryRequired blocks new starts, no unblock path authorized), h101-128-phase3-supervision.md line 73 ("Exact resolution UI/proof policy needs security review before enabling restart"), and my own verification below.

Verified before ruling:
- `CallerScope` today has exactly two fields — `AgentID`, `IsHumanReviewer` (`internal/core/task/engine.go:31-34`). No process-admin capability field exists yet.
- h101-128-phase3-supervision.md:35 — "a principal may start its own registered agent only with a host-granted managed-run capability for that profile; human admin may manage registered agents."

## 1. Authority — the IsHumanReviewer category error

`IsHumanReviewer` is a task-review flag, not a process-admin capability. Reusing it to authorize recovery resolution is a category error: task review and process-lifecycle control are different powers, and nothing in the codebase today grants the second.

Blocking prerequisite: `CallerScope` needs a distinct host-granted capability field (e.g. `CanResolveRecovery`), issued the same way h101-128:35 already describes for admin start-authority. That field must exist and be checked before any unblock code ships. This is not a UI/proof detail — it is a prerequisite to any unblock path existing at all.

## 2. Confirmation and evidence

The admin must attest to a specific, externally-verified fact: what they personally checked (PID/process-list inspection, an explicit kill they issued, a monitoring tool, etc.) and when. A bare confirm click with no evidence field is a reviewer-flag shortcut, and shortcuts of that kind are already excluded by h101-128:73 ("No default auto-clear").

## 3. Recording — separate record type

The attestation must be stored as its own distinct record type (e.g. `AdministrativeResolution`), never merged into or rendered as an observed-exit/Finish event. Mixing admin belief with system-observed fact would let a false admin claim later read as machine-verified death. Keep the separation in both storage and display treatment — an administrative resolution must never appear merged into the observed lifecycle status.

## 4. Audit trail

A durable, immutable record of: who (the `AgentID` holding the resolution capability), when, which run/agent was released, and the evidence description given under point 2. The record must commit before the unblock effect takes place, using the same commit-before-effect pattern already used for the Start barriers — so a crash between attestation and unblock cannot silently apply a stale or partial resolution. The record must be queryable independent of ordinary task/run history, for the same reason mailbox envelopes cannot create sessions: an administrative override record must not be confusable with a normal domain event.

## 5. Same-user file tampering — exclusion holds, with one condition

Agrees with Stanley's exclusion; no new position — consistent with the exclusion already standing since H101-131/H101-135. One condition carries forward: the resolution action itself must go through the sole host's serialized authenticated command path, the same barrier used for Start, never a raw file/state edit outside it. That is an application of the existing architecture, not a new mechanism.

## 6. Negative case — shipping H101-170 with no unblock path

Shipping H101-170 with no unblock path at all is acceptable as an interim state, and is the strictest safe direction: a false block costs availability only, not integrity, confidentiality, or safety, and creates no new risk that needs handling now. This does not gate H101-170. What it gates is any later card that adds restart/unblock — that card must satisfy points 1–4 above, reviewed at the design stage, before code.
