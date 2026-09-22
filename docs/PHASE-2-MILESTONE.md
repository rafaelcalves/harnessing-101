# Phase 2 milestone — CLI adapter

Status: **exit criteria discharged, awaiting owner approval.** Prepared 2026-09-21 at
commit `bfa5f1b` (Kelly's phase-exit ruling). The project plan requires each phase to end
with a review of the work and the practices, a written summary, and a stop for human
approval before the next phase starts. This is that summary. **Phase 3 has not started**
and is not implied by this document — it records what Phase 2 discharged and asks the
owner to approve opening the next phase, the same stop Phase 1 required.

## What Phase 2 promised

"Phase 1 proves this cycle without a user interface. Phase 2 makes the same cycle usable
from the command line and demonstrates a replaceable interface. Its milestone is a real
two-agent task completed with durable handoffs, including a restart of the interface and
a human decision." (`docs/product/definition.md` §2)

## Exit criteria and their evidence

Kelly rewrote the exit criterion into nine CI-checkable items at the start of the phase,
the same practice Phase 1 used. All nine are dispositioned; the verdicts are quality's,
not the orchestrator's, and each is recorded in full in
[`docs/quality/definition-of-done.md`](quality/definition-of-done.md). The final
phase-exit ruling is
[`docs/quality/h101-112-phase2-item2-discharge-ruling.md`](quality/h101-112-phase2-item2-discharge-ruling.md)
(`bfa5f1b`), re-verified by Kelly against `17af7af` rather than taken on report.

| Item | Status | Discharged by |
| --- | --- | --- |
| 1 Product cycle through a spawned binary, both ADR 0001 targets | **Satisfied** | linux/amd64 native CI (ongoing); darwin/arm64 disclosed manifest (`02b1407`, H101-98) superseded by the native `darwin-arm64` macOS CI runner (`f6075fa`, H101-102/H101-105), observed green on run `35598625946` |
| 2 Restart after crash, cross-target acknowledgement survival | **Satisfied, unconditionally on ADR 0001 targets** | `17af7af` (H101-112, `TestRun_Phase2CycleRecoversAfterKilledCLI` amended per the H101-104 spec); ruled at `bfa5f1b`; CI run `35614603811` green on both `build-test-lint` (ubuntu) and `darwin-arm64` (macos-14) |
| 3 Swappability, one shared suite, independent expectations | **Satisfied with limit** | `e565b82` (suite structurally complete on both adapters); ruled at `1b28910` (H101-107); see [Limits](#limits-recorded-rather-than-resolved) below — this is not rounded up to plain "satisfied" |
| 4 Containment: presentation imports `api` only; `assembly` is the sole `host` importer | **Satisfied** | `8ba1112` + `9a6b464` (H101-79 acceptance) |
| 5 Mailbox adapter (H101-22): sole writer of publish/process facts | **Satisfied** | `6bae581`; sole-writer proven by call-graph plus containment — limit recorded, not waived: `host.Capabilities` still exposes the record methods (type-level enforcement was never required by any ruling) |
| 6 First-run disclosure, unconditional on every invocation | **Satisfied** | `1eda92b` |
| 7 Phase 3 operations (`StartRun`/`StopRun`/`SetRunBudget`) return `Unsupported` | **Satisfied** | `20e0caa` (H101-81/H101-80 acceptance) |
| 8 Zero network | **Satisfied** | `548e3a3` (CI-guarded `go list -deps` check, not just a manual habit) |
| 9 Test layering (engine unit tests do not substitute for the adapter-contract/subprocess proof, and vice versa) | **Not applicable** | Policy check, not a gap: engine unit tests, the `adaptercontract` suite, and the §2 subprocess walkthrough are all present and distinct, per the H101-112 ruling |

**Evidence limit (Creed, H101-227):** item 8 proves only that the production
dependency graph lacks literal `net` and `net/http` imports. It does not detect
raw socket syscalls or an `os/exec` network child. Any syscall-capable dependency
needs manual syscall-behavior review when admitted and on every version bump;
pinning, semantic-version trust, and automated updates do not substitute for it.

## What was built

A **shipped `harnessing` CLI** (`cmd/harnessing`) driving the full product cycle through
one composition root (`internal/assembly`) that is the sole importer of `internal/host`;
a **second, independent adapter** (`internal/throwawayadapter`) proving the interface
boundary is real, not aspirational; a **shared black-box contract suite**
(`internal/adaptercontract`, UI-01 through UI-08) running both adapters against
independently specified expected states, including crossover continuation on one shared
workspace; a **file-protocol mailbox adapter** (`internal/adapters/mailbox`) that is the
sole writer of publish/process delivery facts, wired into every CLI invocation via
`assembly.WithSession`; **native CI on both ADR 0001 in-scope targets**
(`ubuntu-latest` and `macos-14`), retiring the disclosed-manifest evidence that first
carried darwin/arm64 through the phase; and a **compile-time exclusion proof** for
Windows (H101-103) — not support, a stated and verified refusal.

## Limits, recorded rather than resolved

These are honest gaps, not oversights, exactly as Phase 1 recorded its own.

- **Item 3's six standing limits (G1, G2, G3, G4, G6, G7)** — from Stanley's independent
  traceability review (`c77783f`, H101-91 stage 2), dispositioned by Kelly at `1b28910`
  (H101-107) and unchanged since:
  - **G1** — item 3 credits presentation **meaning** (delivery facts, unverified
    provenance, ADR 0002 disclosure), not exact CLI label punctuation, stderr stream
    choice, or the validation-cue substring deny-list as universal contract.
  - **G2** — UI-06 observer tests prove the fixture task's subject appears after the
    cursor; they do not prove no additional subject may be present.
  - **G3** — item 3 credits the named restart stale-cursor scenario, not universal
    cursor encoding syntax or unconditional expiry on every restart.
  - **G4** — monotonic raw revisions in the concurrency helper are ordering evidence,
    not proof of a duplicate-free or one-event-per-revision raw stream.
  - **G6** — item 3 credits `Unsupported` availability on the Phase 3 surface; a
    mandatory nonempty `Detail` and malformed-payload precedence are not architectural
    discharge.
  - **G7** — cross-adapter comparison uses the workspace end state at an equivalent
    committed cut, not individually derived task-revision literals as architecture.
  - **G5** (unregistered assignee succeeding in harness-only paths) is **closed**, at
    `8da96de` (H101-108): the engine itself validates `AssigneeID` against the registry
    inside `CreateTask`, discharged on evidence grounds per `de76f73` (H101-110). This
    is the one gap of seven that moved from limit to closed; the other six stand.
  - **Verdict stands at "satisfied with limit."** It is not "satisfied."
- **Windows exclusion (H101-103) is the only remaining limit on item 2**, and it is a
  separate, out-of-scope platform assertion — **not** partial satisfaction on either
  in-scope target. Item 2 is unconditionally satisfied on `linux/amd64` and
  `darwin/arm64`; Windows workspace operations return a stable `Unsupported`, proved by
  a compile-time guard (`GOOS=windows` build plus `go vet` of windows-tagged tests, no
  windows runner exists), which is proportionate for a platform ADR 0001 never brought
  into scope.
- **Item 5's sole-writer property is proven by call graph plus containment, not by
  removing the capability.** `host.Capabilities` still exposes
  `RecordMessagePublished`/`RecordMessageProcessed`; nothing in the current product path
  calls them except the mailbox `Deliverer`, and grepping the repository confirms it, but
  the type system does not prevent a future caller from doing so. No ruling required
  removing those methods, so this is recorded as the limit it is, not solved.

## Non-exit follow-ups that do not block this milestone

Per `docs/quality/definition-of-done.md` §"Explicitly not Phase 2 exit," none of the
following gate this milestone. They are recorded here so the next phase inherits them
deliberately rather than silently.

- **H101-30** — Creed's standing constraint on quoting the disclosure text: a partial
  quote of ADR 0002's opening clause alone (egress, without the immediately-qualifying
  non-confinement and no-data-stay-guarantee language) would mislead. It is not a defect
  in the in-product text; it fires the first time "inspectable trust" reaches a
  reader-facing document quoting it selectively.
- **H101-78** — Kevin's own honest maybe on running `go mod tidy`: recorded at the
  strength it was given (uncertain, not a ruled gap), worth checking only if a dependency
  is ever added to `go.mod`.
- **H101-38 and H101-46** — the two held product calls (whether task reassignment is
  wanted; whether the first-run disclosure should lead with the threat or the promise).
  Currently unassigned; orphaned when their prior owner left the floor. Neither blocks
  anything per the DoD's own item 6 ruling, which explicitly separates lead-sentence
  ordering from acceptance criteria.

## Where sources agree and where this document does not referee

`docs/quality/definition-of-done.md` is an append-only log: earlier entries in that file
show items 1–5 and 7 as "not satisfied" or "partially satisfied" at the time they were
written, superseded by later entries in the same file and by the two standalone rulings
this document cites (`h101-107-phase2-item3-ruling.md`, `h101-112-phase2-item2-discharge-ruling.md`).
That is the log doing its job, not a disagreement between authorities — the most recent,
most specific ruling for each item is the one quoted in the table above. No source
reviewed for this document contradicts another as of the same date; if one is found to,
it is reported to `god` rather than resolved here.

## What is being asked

Approval to close Phase 2 and open **Phase 3**, whose scope this document does not define
and whose opening this document does not claim has happened. Phase 2 carries forward the
recorded limits above (item 3's six standing gaps, the item 5 type-level enforcement gap,
the Windows compile-only proof) and the two unassigned product calls (H101-38, H101-46)
as inherited, not resolved, context for whoever scopes the next phase.
