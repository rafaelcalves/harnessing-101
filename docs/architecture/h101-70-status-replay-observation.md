# H101-70 — Status provenance, commit replay, and observation scope

Architecture rulings, 2026-09-21. Documentation only; no code or tests changed. Read against the local working tree, not an independently verified commit hash.

## 1. Persist latest status provenance now

**INFERENCE — ruling.** Implement `Task.LastStatusChange` as an optional record with taskRevision, optional fromStatus, toStatus, and the existing Provenance value. Set it on creation and every actual transition, including report/accept/reject/reopen, in the same transaction as the task. The actor is the host-bound ClaimedAgentID; RecordedAt comes from the core clock; IdentityVerification remains unverified. This records who reported the change, not authenticated authorship or observed process activity.

**INFERENCE.** Keep creation provenance unchanged. Retries, rejected commands, and no-op operations do not advance latest-status provenance. Missing legacy records remain explicitly unavailable. Both task lookup and snapshots must detach the new nested fields. Implementation details and legacy handling are now specified in [boundaries.md](boundaries.md#h101-70--latest-status-provenance-2026-09-21).

**INFERENCE.** Choose the latest record, not a new unbounded history, for this requirement. Full audit history has separate retention, migration and query implications; §2 needs last reporter/time. This choice deliberately does not preserve every intermediate transition. If complete history becomes a requirement, record it prospectively and do not claim old history can be reconstructed. Future ownership changes must define their own provenance; no reassignment command is added here.

## 2. Keep replayed original events; correct the contract wording

**CODE-PATH FACT.** `ports/outbound.go` says identical-request replay returns a receipt with “no new events”; FileStore returns the recorded events. `Engine.commit` assumes only fresh events reach publication, while `events.go` separately deduplicates by revision. The descriptions do not agree.

**INFERENCE — ruling.** Keep the established return behavior. Specify: “Replay creates no new revision, mutation, or event identity. After required confirmation it returns the original receipt and original event records; those records are replay data, not newly committed notifications.” Correct both the port and Engine comments in the implementation card. Do not discard stored event records just to satisfy the ambiguous wording.

**INFERENCE.** Publication must be centralized, ordered and idempotent; every consumer of Commit's event return must distinguish returned history from new publication. The current high-water-mark deduplication is sufficient only if commit/publication order is guaranteed. A stable event ID supports deduplication, but no dedup scheme alone restores an omitted event. No new Replay flag or port signature is required by this ruling; consumer-contract tests must cover the distinction.

## 3. In-process observation is sufficient; current implementation is not yet proven sufficient

**INFERENCE — ruling.** ADR 0003 requires cross-adapter observation, not independent writer processes. Its concurrent scenario is two sessions on the same authoritative host. A local bridge could later connect another process to that host without making a durable log mandatory. Bounded replay with explicit CursorExpired/resnapshot is allowed; durable full-history replay is not required. The clarification is appended to [ADR 0003](../adr/0003-ui-adapter-contract.md).

**CODE-PATH FACT — two gaps to test/fix before claiming UI-06.** `newEventBus` initializes lastPublished and trimmedUpTo to zero; Subscribe checks only parsing and the trimmed floor. Reopening does not seed that floor from storage or identify an epoch. A previously issued numeric cursor can therefore be accepted against an empty buffer instead of rejected. The existing expiry tests cover malformed and evicted cursors, not this restart path.

**CODE-PATH FACT / INFERENCE.** `Engine.commit` calls FileStore.Commit and then publishes after that call returns. The store serializes commits, but there is no engine-wide serialization covering both operations. Two callers can commit revisions R and R+1, publish R+1 first, and then have R dropped by `events.go`'s `<= lastPublished` check. The single-writer lock does not prevent goroutine interleaving within that writer. A failed/uncertain commit later confirmed also needs honest publication or explicit stream invalidation; newer revisions must not conceal a gap.

**INFERENCE.** These are local stream correctness obligations, not grounds to build a cross-process log. Preserve ordered commit publication, establish restart cursor validity, and test two sessions plus resnapshot recovery. If one commit can contain multiple events, checkpoint only the complete revision or use a cursor position within it; otherwise reconnecting mid-commit can lose remaining events. No implementation acceptance is implied by approving an in-memory design.
