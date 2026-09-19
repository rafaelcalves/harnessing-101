# ADR 0004 — Structured uncertainty for mutating commands

Status: **Proposed for review**, 2026-09-19. Author: Stanley, Architect. Card: H101-54. Taxonomy and recovery contract only; no implementation change.

## Context and evidence

**CODE-PATH FACT.** `internal/core/domain/errors.go` exposes Code and Detail and explicitly says callers branch on Code, never Detail. `internal/adapters/statestore/file.go` nevertheless reports directory-sync failure after successful rename as IOFailure with the detail “commit applied but durability unconfirmed.” The snapshot and receipt are already in the replacement file at that observation point. Commit returns an empty success receipt alongside the error.

**CODE-PATH FACT.** The same file's receipt-match branch returns the saved receipt/events before calling writeLocked or any new durability barrier. Therefore same-request replay currently avoids another mutation, but receipt replay alone does not re-establish the durability guarantee that failed. These are source-reading findings, not a fault-injection test performed for this ADR.

**SECONDARY SOURCE.** [Quality's UI-02 assessment](../quality/definition-of-done.md) identifies the detail-substring dependency. [ADR 0003 UI-02/03](0003-ui-adapter-contract.md) requires honest uncertainty and preservation of request identity. Those behavioral requirements stand; the taxonomy is insufficient. [ADR 0001's platform scope](0001-language-and-runtime.md) requires confirmation mechanisms verified on both supported targets, not assumed portable from one platform.

## Options considered

**INFERENCE.** IOFailure plus text makes a semantic branch depend on wording. Mapping everything to RecoveryRequired loses the distinction between an uncertain command result and a workspace that cannot safely continue. Adding only a field to IOFailure is possible, but leaves unaware callers treating this as an ordinary storage failure. A distinct stable code with explicit evidence fields makes the required recovery path visible while preserving useful detail.

## Decision and stable error shape

**INFERENCE — ruling.** Add **OutcomeUncertain** for a mutating operation whose confirmed outcome cannot be reported. This is not a rollback signal and not a successful receipt. Its structured metadata is:

| Field | Contract |
| --- | --- |
| `requestID`, `workspaceID` | Identify the original command in the caller-bound workspace; caller identity is obtained from the session, never accepted from this error as authority. |
| `effect` | `Applied` if the handler observed the logical change take effect; `Unknown` if it cannot establish whether it took effect. Never infer Applied from a timeout alone. |
| `confirmation` | `Durability` when persistence confirmation is missing; `Outcome` when the command response/completion itself is unavailable. |
| `observedRevision` | Optional revision actually observed, not a promise that it survives a crash or a replacement for a confirmed receipt. |
| `detail` | Diagnostic prose only; no branching or conformance assertion depends on its wording. |

**INFERENCE.** Post-rename directory-sync failure maps to `OutcomeUncertain(effect=Applied, confirmation=Durability)`. A lost local command response may map to `OutcomeUncertain(effect=Unknown, confirmation=Outcome)` at the application/transport boundary. That boundary cannot manufacture a server-side confirmation. `Applied` describes an observation when the error arose, not guaranteed state after a subsequent power loss.

**INFERENCE.** Keep IOFailure for storage/transport failures that do not require an uncertain-mutation outcome, including failed reads and failures known to precede application of a new command. Never interpret IOFailure alone as proof that an earlier attempt with that request ID did not apply. Keep RecoveryRequired for cases where safe continuation needs recovery/operator intervention, such as unreadable state; it must not erase the outstanding request's uncertainty. Failures before mutation and post-application failures must be classified by their actual effect boundary, not by syscall name.

## Caller action: resolve the original request

**INFERENCE.** On OutcomeUncertain, show “change applied; durability unconfirmed” or “outcome unknown” according to the fields, retain the original request ID and exact command, and stop automated follow-on actions that assume confirmed success. Do not invent a new request ID, repeat with changed arguments, promise rollback, or issue compensation as though failure proved nothing happened. Reading a task with the expected value is useful context but does not prove which request wrote it or whether it is durable.

**INFERENCE — correction to the blanket retry premise.** Retrying is not inherently wrong. Replaying the **same** caller-scoped request ID and identical payload through a correct idempotency path is different from executing a new command. The current replay path prevents duplicate mutation while its receipt is visible, but it does not discharge the missing confirmation. Neither “just retry” nor “never retry” is a sufficient contract.

**INFERENCE — resolution operation.** Add caller-bound `ResolveRequest(requestID)` to the application query/recovery surface, not a raw-store capability. This does not execute a domain command, allocate a new request ID, or increment the workspace revision. The host resolves the request under the same serialization as commits and performs the persistence adapter's required confirmation work before returning a **Confirmed** receipt. A mere lookup returning a previously observed receipt is not Confirmed. The operation may perform confirmation I/O despite being nonmutating at the domain level.

**INFERENCE.** If confirmation still fails, return OutcomeUncertain with the best observed evidence. If the authoritative store currently has no receipt, return **Absent**, explicitly meaning “no request record found now,” not “this never happened.” Corrupt/unreadable storage is an error, not Absent. A successful confirmation preserves the original receipt and does not emit duplicate domain events. Authorization precedes lookup; another principal's request records are not exposed.

**INFERENCE.** For current core-only changes, state and receipt share one transaction: after authoritative recovery, Absent permits an explicit resubmission of the identical command with the same request ID, using normal current-state validation. For effects outside that transaction—publishing a message or starting a process—Absent alone cannot authorize repetition. Those effects require their own persisted intent, effect ID and reconciliation contract. This ADR does not replace H101-22 or promise exactly-once external effects.

## Scope beyond fsync and required follow-up

**INFERENCE.** The class includes a lost response after an otherwise successful commit, interruption after applying a write but before confirming it, and an effect whose acknowledgement was lost. A fsync-specific error name would miss that class. Each producer must define its own application and confirmation boundary; do not wrap every I/O error as uncertainty indiscriminately. Abrupt process death produces no error response, so a client without a final response must preserve an unknown outcome locally and resolve it after reconnecting.

**INFERENCE.** Implementation must carry the new code/fields through domain, host, and both adapters. Review replay and resolution together: a matching receipt cannot silently upgrade an unconfirmed write to success. The adapter may re-establish the required barriers on the authoritative file generation while serialized; the exact mechanism is an implementation decision. A later durable generation may confirm an earlier receipt it still contains, but elapsed time or a successful read cannot.

**INFERENCE.** Quality should inject a post-application confirmation fault, assert the structured Applied/Durability result, verify a repeated request never reruns mutation, and keep confirmation failing on replay/resolution until the fault is cleared. Then confirm the original receipt without new events/revisions. Separately exercise a response-lost case and Absent after recovery. Change diagnostic text in the fixture to prove behavior does not depend on a substring. These are required future checks, not tests run here.

## Consequences and revisit triggers

**INFERENCE.** This adds an error variant and a recovery operation to the evolving version 1 application contract; it does not weaken UI-02. Coordinate their implementation before marking uncertainty handling complete. Until all producers are migrated, an unclassified IOFailure from a mutating request must be presented conservatively as unconfirmed; clients must not parse detail to decide whether it rolled back. Unknown future error codes likewise never imply successful completion.

**INFERENCE.** No statestore, UI, or existing ADR was edited for this ruling. Follow-up work must reconcile boundaries.md and API declarations after acceptance. Revisit when distributed/remote effects, bounded receipt retention, or independent actors changing the same storage invalidate the current caller-scoped atomic state/receipt assumptions. This taxonomy describes what is known; it cannot manufacture durability or guarantee an operation's eventual completion.
