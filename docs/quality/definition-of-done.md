# Harnessing 101 — definition of done and phase criteria

Status: **Proposed**, Phase 0. Owner: Kelly (QA). Applies to all `H101-` cards.

---

## Definition of done

A card is **done** only when every item below is satisfied. A completion report from the assignee is input to review; it is not acceptance.

### Universal (documentation and code)

1. **Traceability** — Acceptance criteria on the card are explicit, and each criterion maps to a requirement in `docs/PROJECT-PLAN.md`, `docs/product/definition.md`, an ADR, or an owner decision recorded on the card.
2. **Evidence labels** — Every factual claim carries one label: CODE-PATH FACT, OBSERVED STATE (dated), SECONDARY SOURCE, INFERENCE, or UNKNOWN. UNKNOWN is acceptable; an unlabelled guess is not.
3. **Honest verification** — Any check that compares expected vs actual state must **fail** (non-zero exit, failing test, or explicit FAIL in the report) when they differ. Printing a mismatch and reporting success is a **failure of the card**, not evidence.
4. **Scope discipline** — Work stays inside `~/Projects/Harnessing101`. The card does not deliver capabilities reserved for a later phase unless the card explicitly revises phase boundaries with owner approval.
5. **Status accuracy** — Document headers (`Status`, `DRAFT`, `Accepted`, etc.) reflect reality. A superseding decision updates or links from older artefacts; stale recommendations left in place are a finding.
6. **Cross-artefact consistency** — No unresolved contradiction with a locked ADR, the product definition, or non-negotiable constraints in the project plan. Conflicts are reported, not silently patched in a downstream file.
7. **QA sign-off** — Kelly (or delegate) has reviewed against this checklist and recorded accept, accept-with-findings, or reject. Assignee "done" ≠ accepted.

### Documentation cards (additional)

8. **Resolvable references** — Linked paths exist or are marked TODO with a blocking card ID.
9. **Checkable exits** — Phase-exit language is restated as observable criteria (see below), or flagged as too vague.
10. **Terminology** — Uses project vocabulary (`workspace`, `task`, `message`, `agent`) consistently, or documents a deliberate alias.

### Code cards (additional)

11. **Tests** — Behaviour is covered by automated tests; a failing assertion blocked merge. No test that asserts success when the assertion failed.
12. **Hexagonal boundaries** — Core code imports no CLI, GUI, filesystem layout, network client, or process handle. Adapters implement ports only.
13. **Local-only** — No implicit network, telemetry, or phone-home in product code paths; egress is user-configured and visible.
14. **Commit identity** — Commits use `rafael.ca.dev@gmail.com` (repository-local config); verified by the Phase 0 guard or CI.

---

## Phase exit criteria

Derived from `docs/PROJECT-PLAN.md` phase descriptions. Where the plan's one-liner is not yet checkable, noted.

| Phase | Checkable exit | Gap |
| --- | --- | --- |
| **0** | `CONTRIBUTING.md`, licence, conduct, templates, CI, commit-identity guard, accepted stack ADR, threat-model baseline; plan updated to match ADR | "Stranger could contribute" is subjective; Kevin/Ryan artefacts in flight |
| **1** | See **Phase 1 exit (revised 2026-09-19)** below — Stanley H101-25 adds composition boundary, CI allowlist, and assembly authorization tests before phase acceptance | Was: port APIs + unit tests only; H101-19 containment moved in from Phase 2 |
| **2** | See **Phase 2 exit (revised 2026-09-19)** below — CLI §2 walkthrough, restart-after-crash, shared adapter contract ([ADR 0003](../adr/0003-ui-adapter-contract.md) UI-01–08 / H101-40), mailbox (H101-22), disclosure (H101-16), stale-lock (H101-20), Phase 3 `Unsupported`, inherited containment | Was: one sentence; obligations H101-16/20/22 were undated in the table |
| **3** | Start/stop/budget/crash-recovery on supported platforms; process-tree termination; misbehaviour → `RecoveryRequired` | Supported: `darwin/arm64`, `linux/amd64` per ADR 0001 H101-50 (`bcba765`); native evidence per target |
| **4** | Clean-machine install, observed egress audit, new-joiner docs, version tag | Needs written observation protocol |

**Weakest phase exit:** Phase 0 — "a repository a stranger could contribute to" has no objective threshold, and the plan still recommends TypeScript while ADR 0001 accepts Go (see audit).

### Phase 1 exit (revised 2026-09-19, per H101-25)

Phase 1 is **not accepted** until all of the following pass in CI on `main`:

1. **Product cycle** — ≥2 registered agents, one human, full task/message cycle (assign, hand off, report result, human accept/reject, acknowledge) through the **headless composition** entry point, with durable state surviving workspace reopen.
2. **Assembly authorization** — negative tests through the returned capabilities only: direct Doing→Done rejected; agent `AcceptTaskResult` rejected; wrong-recipient `AcknowledgeMessage` rejected; persisted state unchanged after each denial; legitimate accept and ack survive reopen; agent ingress cannot set human-review authority (obligation 3).
3. **Containment** — production packages outside an approved allowlist do not import `internal/adapters/statestore` or construct `ports.CommitRequest`; CI allowlist check green, including a **forbidden fixture** that must fail the check (obligation 2).
4. **Composition surface** — shipped entry point returns only command/query capabilities plus shutdown; no `FileStore`, `StateStore`, `CommitRequest` builder, or recoverable concrete store (obligation 1).
5. **Unit tests** — engine/statestore unit tests may still construct engines directly for domain rules; they do not substitute for items 1–4.
6. **Zero network** — unchanged; core module has no network imports.

### Phase 2 exit (revised 2026-09-19, per H101-39)

Phase 2 is **not accepted** until all of the following pass in CI on `main`. Items inherit Phase 1 limits recorded in [`docs/PHASE-1-MILESTONE.md`](../PHASE-1-MILESTONE.md): the import allowlist gate stays green; no unapproved package imports the store; the forbidden fixture must fail the check (exit code **1** once H101-34 lands); presentation adapters reach persistence only through reviewed assembly (`host.Open` → caller-bound **FrontendSession**, per [ADR 0003](../adr/0003-ui-adapter-contract.md)), not direct `StateStore` / `Capabilities` injection.

**Obligation placement (H101-16, H101-22, H101-20):** all three are **Phase 2 exit**, not Phase 3. Phase 3 is process supervision; mailbox ordering, first-run disclosure, and workspace lock recovery are prerequisites for a usable CLI and the §2 restart milestone, not for starting agents.

| # | Criterion | Checkable today? |
| --- | --- | --- |
| 1 | CLI product cycle (§2 walkthrough) | **Partially** — linux/amd64 CI + darwin/arm64 manifest (`02b1407`, [manifest](darwin-arm64-milestone-manifest-2026-09-21.md)); platform-agnostic gaps if any remain per latest review |
| 2 | Restart after crash | **Partially** — linux + darwin native crash-reopen + lock per manifest; cross-target handoff ack proof still host-side on both |
| 3 | Swappability (ADR 0003 UI-01–08) | **No** — `throwawayadapter` exists (`7dc7809`); no `adaptercontract` suite; UI-01–08 not run on both adapters |
| 4 | Containment preserved | **Satisfied** — store half (`24a2022`); presentation imports `api` only; `internal/assembly` sole `host` importer (`9a6b464`) |
| 5 | Mailbox adapter (H101-22) | **Satisfied** — wired in `assembly.WithSession` (`6bae581`); full-path integration test; sole-writer by call-graph + containment (limit: methods remain on `host.Capabilities`) |
| 6 | First-run disclosure (H101-16) | **Satisfied** — every invocation, stderr, ADR 0002 sentence, CI tests (`1eda92b`) |
| 7 | Phase 3 ops `Unsupported` | **Satisfied** — `StartRun`/`StopRun`/`SetRunBudget` on `api.FrontendSession` return `Unsupported`; named test per op (`20e0caa`); no CLI subcommands (allowed) |
| 8 | Zero network (CLI tree) | **Satisfied** — `go list -deps ./cmd/harnessing` finds no `net`/`net/http` (`24a2022`) |
| 9 | Test layering | **N/A** — policy; enforced by review |

1. **CLI product cycle (§2 walkthrough)** — A CI-runnable script or `go test` drives the **minimum useful product** in [`docs/product/definition.md`](../product/definition.md) §2 exclusively through the shipped `harnessing` command (subprocess, not in-process `host` import from the test package): create/open workspace; register ≥2 named agents; create a task with one accountable owner; task-linked handoff with explicit acknowledgement; report a blocker and resolve it; report a result; human reject then accept a replacement result; status queries show assigned / in-progress / blocked / awaiting-review with reporter identity; close the CLI; reopen; assert durable records without resending work. **This item is what “usable for real work” means** — not a subjective judgment.

2. **Restart after crash** — After the cycle in item 1, simulate an unclean host exit (`kill -9` or equivalent) without `Close`; a subsequent `harnessing` invocation must reopen the same workspace and recover the persisted cycle state. **Supported targets** (ADR 0001 H101-50, `bcba765`): `linux/amd64` and `darwin/arm64` on native local storage only. Each in-scope target owes **native** adapter-cycle and crash-reopen evidence recorded in the milestone manifest (exact OS/build, commands, outputs) — **Ubuntu-only CI does not discharge macOS.** **H101-20** satisfies the lock-recovery prerequisite per target when that target's native crash-reopen proof passes (see H101-55). **Windows exclusion smoke:** workspace commands return `Unsupported`; `version`/`help` still run — asserts defined exclusion, not support. Graceful reopen alone does **not** discharge this item (ADR 0003 UI-07). Inherited from Phase 1 limits; not deferrable to Phase 3.

3. **Swappability (shared adapter contract)** — [ADR 0003](../adr/0003-ui-adapter-contract.md) (H101-40) defines behavioral scenarios **UI-01 through UI-08**; this item requires both adapters to pass them in CI per the **H101-53 assertion table** below, not reproduce the architecture here. The shared black-box suite lives in `internal/adaptercontract/contract_test.go`; each adapter enters through its **real** input driver (CLI argv/stdin/structured output; throwaway independent handler — no CLI imports, no shared dispatch). Observations come through the caller-bound **FrontendSession** surface (not raw `Capabilities` injection). Suite includes per-adapter isolated workspaces, cross-adapter continuation (A writes, B reads/acks/reviews), and concurrent observer visibility. Compare each adapter against **independently specified expected states**, not adapter-to-adapter equality alone (ADR 0003). **Item 3 does not discharge item 2** — UI-07 covers graceful reopen only; crash restart stays item 2. **Fake run-output stream tests do not count** toward replaceability; actual supervision/output stays Phase 3. Core must import neither adapter.

4. **Containment preserved** — `scripts/check-import-allowlist.sh` green on `main`; store-import allowlist unchanged (only `internal/host` constructs persistence). **Extend** checks so presentation packages cannot import `host`, `core/task`, storage, or outbound ports (ADR 0003 — no storage allowlist exception for UI). Forbidden fixture fails the check. H101-34: fixture step must require exit code **1**, not any non-zero.

5. **Mailbox adapter (H101-22)** — File-protocol mailbox is the sole writer of `RecordMessagePublished` and `RecordMessageProcessed`; publish precedes process; skipping publish or reversing order is rejected with persisted state unchanged. `MessageAcknowledgement` control records and `AcknowledgeMessage` share the same recipient and idempotency rules. Integration test covers at least one full send → publish → process → ack path through the adapter.

6. **First-run disclosure (H101-16)** — Automated test: every `harnessing` invocation emits the unconditional ADR 0002 disclosure line (“does not start, observe, or restrict any agent process…”) before command handling (stderr, so stdout stays scriptable for item 1). **“Unconditional”** means visible on every run, not a persisted once-per-workspace marker. No validation-looking cues on manually sourced content. README clone-and-run disclosure remains (regression check against existing text).

7. **Phase 3 operations rejected** — `StartRun`, `StopRun`, and `SetRunBudget` (or their CLI equivalents) return **`Unsupported`** with a stable, documented error — not absent handlers that panic, not silent success. If Phase 2 ships without registering those subcommands, CI asserts they are unreachable as success paths. Process control remains Phase 3 scope.

8. **Zero network** — `go list -deps` on `cmd/harnessing` and packages under `internal/adapters/cli/` (or equivalent) shows no `net` / `net/http` imports, in addition to the Phase 1 core check.

9. **Test layering** — Engine and statestore unit tests remain for domain rules. The ADR 0003 / `adaptercontract` suite (UI-01–08) and item 1's subprocess CLI proof are both required; neither substitutes for the other.

**Explicitly not Phase 2 exit** (owner/product, not quality gates here): which external agent tool is named for the file-protocol trial (`definition.md` §2 UNKNOWN); agent retirement; task reassignment command; full symbol-level `CommitRequest` detection (still not checkable today per Phase 1).

---

## Milestone review procedure

Lightweight, fixed order. Output: one **milestone summary** (≤2 pages) for the owner.

| Step | Who | What |
| --- | --- | --- |
| 1 | Assignee | Completion report: criteria met, evidence labels, verification commands and outcomes, known gaps. |
| 2 | Kelly (QA) | DoD checklist pass; phase-exit criteria; audit for print-and-pass verifications. |
| 3 | Stanley (Architect) | Port boundaries, ADR compliance, stack/phase scope. |
| 4 | Security | Local-only and egress posture for anything touching process or network configuration. |
| 5 | Angela (Product) | Minimum-useful-product alignment; no scope smuggled in. |
| 6 | God | Consolidate findings; **stop for owner approval** before next phase. |

Owner receives: milestone summary, QA accept/reject, open findings list, and explicit go/no-go ask. Practices reviewed, not only code — e.g. were evidence labels used, did anyone confuse report with acceptance.

---

## Phase 0 audit (2026-09-19)

Audited: `docs/PROJECT-PLAN.md`, `docs/product/definition.md`, `docs/adr/0001-language-and-runtime.md`, `docs/architecture/boundaries.md`. Not judged: repository scaffold, OSS templates, CI, threat model (in flight per dispatch).

### Findings (6 inconsistencies)

1. **Stack recommendation vs accepted ADR** — Plan still recommends TypeScript/Node and asks the architect to attack it (`PROJECT-PLAN.md` lines 39–50). ADR 0001 accepts Go and overrules that recommendation (`0001-language-and-runtime.md` line 33). The plan should be updated or explicitly marked superseded on stack.

2. **Approval state mismatch** — Plan header: "DRAFT, awaiting human approval" (`PROJECT-PLAN.md` line 3). ADR 0001: "Accepted, approved by the project owner" (`0001-language-and-runtime.md` line 3). Readers cannot tell which artefact is authoritative for go-ahead.

3. **Evidence convention not applied in product definition** — Plan requires labelled claims (`PROJECT-PLAN.md` lines 93–95). `definition.md` uses inline **UNKNOWN** but does not label other recommendations as INFERENCE or SECONDARY SOURCE — weakens auditability.

4. **"Reported complete" vs task `Done`** — Product requires human acceptance distinct from reported completion (`definition.md` lines 21, 25). Boundaries allow `Doing→Done` without a separate acceptance command or state (`boundaries.md` line 22). Ports may not express the product rule without clarification.

5. **Acknowledgement semantics** — Product requires visibility of message acknowledgement, with acknowledgement ≠ work done (`definition.md` lines 19–20). Boundaries use queued/published/processed receipt states (`boundaries.md` line 47) but never "acknowledged." Terminology gap risks a UI that mislabels publish as ack.

6. **Workspace vs hive** — Product keeps "hive" as protocol metaphor (`definition.md` line 50). Boundaries and ports use **workspace** throughout (`boundaries.md` lines 9, 17+). Not fatal, but public docs need one primary term.

### Alignment

Phase 2/3 process-control split, local-only guarantee, and replaceable UI are consistent across all four files. Ten ports can express the §2 minimum cycle with manual agent start.

### Not yet judgeable

No `CONTRIBUTING.md`, CI, or commit-identity guard in tree. Agent file-protocol compatibility remains **UNKNOWN** (`definition.md` line 27).

### Verdict

Directionally coherent, but **not consistent on stack and approval status**. Resolve acceptance/acknowledgement semantics before Phase 1 code.

---

## H101-7 acceptance review (2026-09-19)

**Card:** Stanley's semantic corrections to `docs/architecture/boundaries.md`. **Verdict: ACCEPT.**

Reviewed against the 14 DoD criteria. H101-7 is a documentation card; criteria 11–14 (code) are N/A. Criteria 1–10 satisfied for this card's scope.

### DoD checklist (documentation)

| # | Result | Note |
| --- | --- | --- |
| 1 Traceability | Pass | Maps to `definition.md` §2 and audit findings 4–5; rationale at `boundaries.md` lines 79–83 |
| 2 Evidence labels | Pass | INFERENCE and SECONDARY SOURCE used throughout |
| 3 Honest verification | N/A | No executable verification on this card; Phase 2 scenario correctly labelled proposed (`boundaries.md` line 75) |
| 4 Scope discipline | Pass | Semantics only; ten ports unchanged |
| 5 Status accuracy | Pass | Header still "Proposed, Phase 0; no implementation" |
| 6 Cross-artefact consistency | Pass | Resolves product-vs-ports gaps without new contradictions. Audit findings 1–3 (plan/ADR stack, evidence labels in definition.md) remain open elsewhere |
| 7 QA sign-off | Pass | This section |
| 8 Resolvable references | Pass | Links to `definition.md` and this file resolve |
| 9 Checkable exits | Pass | Phase 2 scenario expanded with concrete assertions (see below) |
| 10 Terminology | Pass | Hive/workspace defined at line 9 per owner ruling |

### Finding 4 — reported complete vs human acceptance: **CLOSED**

Checked against `definition.md` lines 19–21 and 25 (not Stanley's summary).

| Product rule | Port expression |
| --- | --- |
| "result ready for review" (`definition.md` line 21) | `AwaitingReview` state (`boundaries.md` line 24) |
| "Reported complete" ≠ acceptance (lines 21, 25) | `ReportTaskResult` → AwaitingReview; only `AcceptTaskResult` → Done (`lines 26–28`) |
| Generic bypass removed | `TransitionTask` permits only Todo→Doing, Doing→Blocked, Blocked→Doing, Done→Todo reopen; transitions into AwaitingReview or Done fail InvalidArgument (`line 24`) |
| Human can reject (`definition.md` line 23) | `RejectTaskResult` → Doing with reason (`line 28`) |
| Survives restart (`definition.md` line 21) | AwaitingReview persists until decision, including across restart (`line 26`) |
| Reporter on status (`definition.md` line 21) | Actor identities and timestamps on all task updates (`line 30`) |

No challenge. The dedicated commands enforce what the product requires.

### Finding 5 — acknowledgement semantics: **CLOSED**

Checked against `definition.md` lines 19–20.

| Product rule | Port expression |
| --- | --- |
| See whether message recorded or acknowledged (line 19) | Four separate facts — queued, published, processed, acknowledged — exposed by `GetMessage` and snapshots (`boundaries.md` lines 20, 57) |
| Acknowledgement ≠ work done (line 20) | `AcknowledgeMessage` "proves neither comprehension nor work completion" (`line 32`); task results are separate (`line 57`) |
| Recipient-scoped | Only caller scoped to recipient may acknowledge; wrong recipient → Denied (`lines 32, 75`) |
| Not inferred from publication | Absent acknowledgement labelled "not acknowledged"; ingestion ≠ acknowledgement (`lines 57–58`) |
| File-protocol path | `MessageAcknowledgement` control record through same core checks (`lines 59, 75`) |

No challenge.

### Finding 6 — hive vs workspace: **CLOSED**

Owner ruling: hive = agent team; workspace = on-disk directory.

`boundaries.md` line 9 matches: "A **hive** is the team of agents organising itself; a **workspace** is the on-disk directory the tool owns and writes." Grep of the file shows **hive** appears only at lines 9 and 81 (definitions/rationale). All operational references use **workspace** for storage and command scope. No swapped usage found.

### Phase 2 acceptance scenario (`boundaries.md` line 75)

**Checkable?** Yes, with one implementation-time dependency.

The scenario lists discrete, assertable outcomes: wrong-recipient acknowledgement → Denied; agent acceptance → Denied; direct Doing→Done rejected; stale-result acceptance → Conflict; duplicate acknowledgements/decisions create no duplicate facts; AwaitingReview survives restart without becoming Done; reject/re-accept cycle; command and file-control acknowledgement paths; both adapters produce identical state; Phase 3 ops return Unsupported.

**Would it catch a real regression?** Yes, for the failure modes this card fixed: bypassing dedicated accept/ack commands, conflating publish/ingest with acknowledgement, auto-promoting reported results to Done on restart or replay. The assertions are behavioural, not cosmetic.

**Gap (not a reject):** "identical normalized state/events" needs a defined comparison schema at implementation time. Host-policy test harness for "authorized human" must be specified in the test plan. Without those, two adapters could pass while formatting differs — but the domain-rule regressions would still be caught.

### Phase 0 milestone — still outstanding

Not judged on this card (per dispatch):

- Kevin scaffold report not yet received (Ryan confirmed `go build`, `test`, `vet`, `gofmt` on Go 1.27.1 independently)
- `CONTRIBUTING.md`, licence, conduct, issue/PR templates
- CI pipeline and commit-identity guard
- Threat-model baseline
- `PROJECT-PLAN.md` still recommends TypeScript while ADR 0001 accepts Go (audit finding 1)
- Agent file-protocol walkthrough remains **UNKNOWN** (`definition.md` line 27)

---

## H101-12 and H101-14 acceptance review (2026-09-19)

Reviewed commits `681d663` (Phase 1 core slice), `c44d5d9` (directory fsync amendment), `7cdfbd5` and `5601803` (OSS landing + README). Re-ran `go test -count=1 ./...` locally — all pass.

### H101-12 — Phase 1 task lifecycle slice

**Verdict: ACCEPT**

#### DoD checklist (code)

| # | Result | Note |
| --- | --- | --- |
| 1 Traceability | Pass | Implements `boundaries.md` completion/acceptance contract; card-declared divergences documented in `ports/outbound.go` lines 27–35 |
| 2 Evidence labels | N/A | Code card |
| 3 Honest verification | Pass | Tests assert failure codes and unchanged state on denied transitions; restart test reads back from disk |
| 4 Scope discipline | Pass | Task engine + FileStore only; messages/mailbox/process not claimed |
| 5 Status accuracy | Pass | No false "implemented" claims in code |
| 6 Cross-artefact consistency | Pass | Matches accepted `boundaries.md` semantics for report/accept/reject |
| 7 QA sign-off | Pass | This section |
| 11 Tests | Pass | Six engine tests + four statestore tests; names match behaviours exercised |
| 12 Hexagonal boundaries | Pass | `internal/core/task` imports only `domain` and `ports`; no `net`, filesystem, or CLI |
| 13 Local-only | Pass | No network imports in core (`grep` clean) |
| 14 Commit identity | Pass | Commits authored `rafael.ca.dev@gmail.com`; `githooks/pre-commit` enforces |

#### 1. Is acceptance unbypassable?

**No unbypassed code path from Doing to Done found.**

`TaskDone` is assigned in exactly one place: `AcceptTaskResult` (`engine.go` line 251), which requires `caller.IsHumanReviewer` (line 236) and `loadPendingDecision` enforcing `AwaitingReview` (lines 322–323). `TransitionTask` cannot reach Done from Doing: `genericTransitions` (`engine.go` lines 101–106) permits only Todo→Doing, Doing→Blocked, Blocked→Doing, and Done→Todo reopen; `AwaitingReview` and `Done` as targets fail `InvalidArgument` (lines 109–114). `TestDirectDoingToDoneRejected` covers the named concern; structurally, no other `TransitionTask` path sets `TaskDone`.

**Untested but structurally blocked:** `AwaitingReview→Done` via `TransitionTask` (no test by name; would fail for same map reason). **Not a bypass.**

**Out of slice scope:** a caller could invoke `StateStore.Commit` with a custom `Mutate` that sets `Done` directly (`ports/outbound.go` line 54). That bypasses the engine, not the state machine. Phase 2 host wiring must route commands through `Engine`, not raw `Mutate`.

#### 2. Provenance

Checked against `definition.md` lines 66, 82 and `threat-model.md` lines 138–141.

| Field | Assessment |
| --- | --- |
| `ClaimedAgentID` | Correct — "claimed" in name and doc (`records.go` lines 86–88) |
| `IdentityVerification` = `unverified` only | Correct — doc states no stronger value until a mechanism exists (`records.go` lines 64–70) |
| `EntryMechanism` = `command` | Honest for this slice; file ingress not built yet |

**Minor naming note (not a reject):** `IdentityVerification` as a field name could read like a completed check; the value `unverified` and doc comment mitigate this. UI copy should follow Angela's example ("identity unverified"), not the field name.

No type or field implies authentication occurred.

#### 3. Five declared divergences

| Divergence | Ruling |
| --- | --- |
| (a) `Mutate` closure vs literal `Commit` signature | **Accept** — documented in `outbound.go` lines 27–31; enables atomic task+result updates |
| (b) `Replay` dropped | **Accept** — no subscriber yet; comment commits to revisit |
| (c) `RegisterAgent` not implemented; unregistered `AssigneeID` | **Accept for this slice** — openly declared; full product cycle needs registration in a follow-on card |
| (d) Path-escape guard, no caller-supplied paths yet | **Accept** — `safeJoin` tested (`file_test.go` lines 17–37); constants only in this slice |
| (e) No stale-lock recovery | **Correctly deferred** — does not block H101-12. `boundaries.md` forbids lock stealing on elapsed time; recovery is unbuilt. **Operational impact:** kill -9 without `Close` leaves workspace unusable until manual `.lock` removal. Phase 2 milestone claiming "restart the interface" after crash must not pass until lock recovery exists or manual recovery is documented |

#### 4. Quiet reinterpretation of accepted semantics?

No material reinterpretation found. Dedicated events (`TaskResultReported`/`Accepted`/`Rejected`) used instead of generic `TaskTransitioned` for report/decision (`engine.go` lines 208, 254, 285) — matches `boundaries.md` line 34. Validation failures commit nothing (`file.go` lines 148–152).

#### Amendment — `c44d5d9` directory fsync after rename

**Verdict: ACCEPT** (H101-12 verdict unchanged).

`writeLocked` now calls `fsyncDir` on the workspace root after `os.Rename` (`file.go` lines 226–240). Reasoning is sound: rename atomicity ≠ directory-entry durability; without directory fsync, `TestAwaitingReviewSurvivesRestart` can pass while power-loss would silently lose the commit.

**1. Test worth having?** **No — not economically testable in CI.** Proving this branch needs a crash simulation or a filesystem that refuses `fsync` on a directory. A unit test that only mocks `fsyncDir` to return an error would document the branch but not prove real durability. **Accept as reviewed code**; revisit if a fault-injection hook is added later.

**2. IOFailure after rename — caller problem?** **Manageable; one gap in error taxonomy.**

Checked `Commit` (`file.go` lines 160–166) and `Engine.commit` (`engine.go` lines 382–389):

| Scenario | On-disk state | API return | Safe retry? |
| --- | --- | --- | --- |
| `writeLocked` fails before rename | Unchanged | `IOFailure`, empty receipt | Yes — same request ID re-executes mutate |
| `fsyncDir` fails after rename | **New state persisted** (including receipt in `state.json`) | `IOFailure` with detail "commit applied but durability unconfirmed" (`file.go` line 239), **empty receipt** | **Yes** — retry with **same** request ID hits receipt replay (`file.go` lines 136–138) and returns the recorded receipt |

The engine does not retry and does not treat `IOFailure` as rollback (`engine.go` line 389 passes error through). A caller that retries with a **new** request ID after `fsyncDir` failure may hit `Conflict`/`InvalidArgument` (state already advanced) — confusing but not a silent double-accept.

**Error taxonomy gap:** `boundaries.md` line 13 lists `IOFailure` and `RecoveryRequired` but not "applied but durability unconfirmed." The detail string carries the meaning; no dedicated code exists. **Finding, not a reject** — consider `RecoveryRequired` or a new stable code in a follow-on card. Misleading if a Phase 2 CLI maps all `IOFailure` to "nothing happened."

**H101-12 verdict stands: ACCEPT.**

---

### H101-14 — open-source landing and README

**Verdict: ACCEPT**

#### DoD checklist (documentation)

| # | Result | Note |
| --- | --- | --- |
| 1–7 | Pass | OSS set matches Phase 0 exit requirements for contributor readiness |
| 8 Resolvable references | Pass | All `README.md` links resolve (lines 23–35) |
| 9 Checkable exits | Pass | CONTRIBUTING commands match CI (`.github/workflows/ci.yml` lines 19–28) |
| 10 Terminology | Pass | Consistent with project vocabulary |

#### CONTRIBUTING accuracy

Verified against repository as committed:

- Go 1.27.1 in `go.mod` — matches CONTRIBUTING line 9
- Zero external module dependencies — matches
- `LICENSE` (Apache-2.0), `CODE_OF_CONDUCT.md`, `SECURITY.md`, issue/PR templates — present
- CI runs build, test, vet, golangci-lint — matches CONTRIBUTING line 41
- Commit-identity hook in `githooks/pre-commit` — matches CONTRIBUTING lines 27–28

CONTRIBUTING is true of the repository as it stands.

#### README disclosure (Creed's requirements)

`README.md` lines 7–9 state: (1) product runs locally with no external connections itself; user-configured agents may egress; product does not confine agents. (2) product does not start/observe/restrict agents in current phases; inter-agent content is unverified. This matches `threat-model.md` §5 and the local-only line in the project plan. Adequate for a clone-and-run reader.

#### Link integrity

No broken links in `README.md`. No references to removed paths (e.g. `docs/oss-draft/`).

**Note outside H101-14 scope:** `docs/security/threat-model.md` line 3 still says "no code exists" — stale header, not introduced by this card.

---

### Phase 0 milestone — updated outstanding items

**Now present:** licence, CONTRIBUTING, conduct, templates, CI, commit-identity guard, README disclosure, Go scaffold with passing tests.

**Still outstanding:** threat-model header refresh; `PROJECT-PLAN.md` stack alignment with ADR 0001 (audit finding 1); agent file-protocol walkthrough (`definition.md` line 27); stale-lock recovery before Phase 2 "restart interface" milestone.

---

## H101-18 acceptance review (2026-09-19)

Reviewed commit `3ff78eb` (Phase 1 slice 2 — messages with recipient-scoped acknowledgement). Re-ran `go test -count=1 ./internal/core/task/ -run 'TestSend|TestAck'` — pass.

**Verdict: ACCEPTED WITH FOLLOW-UP**

### DoD checklist (code)

| # | Result | Note |
| --- | --- | --- |
| 1 Traceability | Pass | Implements `boundaries.md` acknowledgement command and four delivery facts (`lines 32, 57`) |
| 3 Honest verification | Pass | Tests assert `ErrDenied` on wrong ack caller; restart test reads from disk |
| 4 Scope discipline | Pass | Mailbox filesystem I/O deferred; explicit host recorders documented |
| 6 Cross-artefact consistency | Pass with follow-up | Core semantics match; host-integration gaps deferred to Phase 2 (see below) |
| 7 QA sign-off | Pass | This section |
| 11 Tests | Pass | Three named tests prove what they claim (see per-test notes) |
| 12 Hexagonal boundaries | Pass | No new adapter imports in core |
| 13 Local-only | Pass | No network imports added |
| 14 Commit identity | Pass | `3ff78eb` authored `rafael.ca.dev@gmail.com` |

### 1. Four delivery facts — never inferred from one another

**Pass in code.**

| Fact | Set only by | Code |
| --- | --- | --- |
| Queued | `SendMessage` | `engine.go` lines 349–361 — sets `QueuedAt` only |
| Published | `RecordMessagePublished` | `engine.go` lines 414–421 — sets `PublishedAt` only |
| Processed | `RecordMessageProcessed` | `engine.go` lines 426–432 — sets `ProcessedAt` only |
| Acknowledged | `AcknowledgeMessage` | `engine.go` lines 395–396 — sets `AcknowledgedBy`/`AcknowledgedAt` only |

Zero-value defaults are nil pointers and empty `AcknowledgedBy` (`records.go` lines 151–155). No code copies an earlier timestamp into a later field. `GetMessage` returns all fields without synthesis (`engine.go` lines 456–468).

`TestSendMessageAndRecipientAcknowledgement` (`message_test.go` lines 31–42) asserts each stage independently — the test does what its name claims.

**Ack before publish is allowed** (`TestAcknowledgedMessageSurvivesRestart` acknowledges without calling publish/process). Matches `boundaries.md` line 57.

### 2. Recipient scoping — bypass routes

**No bypass through `AcknowledgeMessage` API.**

Recipient check: stored `RecipientAgentID` must equal `caller.AgentID` (`engine.go` lines 387–388). Wrong caller → `Denied` (`message_test.go` lines 48–51). `TestSendMessageAndRecipientAcknowledgement` proves this.

**Same class as H101-19 (host bypass):** a host calling `StateStore.Commit` with a custom `Mutate` that sets `AcknowledgedAt` bypasses recipient checks — identical to the task-lifecycle bypass noted in H101-17. Out of slice scope; **H101-19** remains the obligation.

**Recipient field on send:** `RecipientAgentID` is required non-empty (`engine.go` line 322) and persisted; acknowledgement compares against the stored value, not a request field. No ack route through unvalidated recipient — the recipient is whatever `SendMessage` committed.

**Undeclared gap (not an ack bypass):** `SenderAgentID` in the request is not checked against `caller.AgentID` (`engine.go` lines 353–354). Per `boundaries.md` line 34, envelope agent IDs are routing claims; `Provenance` records the host-scoped caller separately. Acceptable if documented; should not be read as authenticated authorship.

### 3. "Zero divergences" — unverified; findings

Claudio declared zero divergences. **Cold read finds at least three** that should have been declared:

| # | Divergence | Ruling |
| --- | --- | --- |
| 1 | `RecordMessagePublished` / `RecordMessageProcessed` are host-callable engine methods, not `Commands` port operations (`boundaries.md` line 19 lists `SendMessage` and `AcknowledgeMessage` only) | **Accept for Phase 1** — pragmatic stub until Mailbox adapter exists; must be declared |
| 2 | Delivery recorders use `CallerScope{}` (`engine.go` line 438), so receipt replay is scoped to empty agent ID, not the host identity | **Minor** — follow-up when host assembly lands |
| 3 | No ordering guard: host may call `RecordMessageProcessed` before `RecordMessagePublished`, or skip publish entirely | **Accept for Phase 1** — Phase 2 mailbox obligation (see §4) |

Not a reject — Kevin's five were structural; these are integration stubs. **Declaring zero was incorrect.**

### 4. Host-driven delivery recorders (mailbox stub)

**Acceptable for Phase 1; Phase 2 obligation.**

The engine intentionally exposes `RecordMessagePublished` / `RecordMessageProcessed` for the host to drive (`engine.go` lines 411–434) because Mailbox is a Phase 2 adapter stub. Facts can be set out of order or skipped by a careless host today.

**This parallels stale-lock recovery (H101-17):** does not block accepting this slice's core semantics, but **Phase 2 mailbox integration must not pass milestone review** until:

- publish/process are driven only by the Mailbox adapter in causal order (queued → published → processed per `boundaries.md` line 57);
- file-protocol `MessageAcknowledgement` ingress shares the same `AcknowledgeMessage` checks (`boundaries.md` lines 59–60).

Acknowledgement before publication remains valid per boundaries; skipping publication while showing "processed" is not.

### Test assessment

| Test | Proves |
| --- | --- |
| `TestSendMessageAndRecipientAcknowledgement` | Four facts stay separate; wrong recipient denied; duplicate ack preserves timestamp |
| `TestAcknowledgedMessageSurvivesRestart` | Ack + queued facts survive process restart |
| `TestSendMessageUnknownReferencesNotFound` | Unknown `taskID` / `replyToMessageID` → `NotFound` |

**Not economically required now:** fault-injection for delivery-fact ordering violations (host misbehaviour); H101-19 mutate bypass (host assembly card).

### Suggested follow-up cards

- **H101-19** (existing): host must route all mutations through `Engine`, never raw `Mutate` for domain rules.
- **Phase 2 mailbox card:** wire `RecordMessage*` from adapter only; enforce publish-before-process; implement file `MessageAcknowledgement` ingress.
- **RegisterAgent validation:** recipient/sender IDs still unregistered (continuation from H101-12 divergence c).

---

## H101-27 — Stanley H101-25 obligations: checkability and Phase 1 exit (2026-09-19)

Reviewed `docs/architecture/h101-25-state-store-authority.md` (`f68367f`). Judged checkability only; did not review security-honesty (H101-26) or H101-22.

**Phase 1 exit changes: yes.** See **Phase 1 exit (revised 2026-09-19)** above. H101-19 containment moves from Phase 2 into Phase 1 acceptance. Phase 1 is not done when engine unit tests pass alone.

### Obligation 1 — private headless composition

**Verdict: CHECKABLE WITH A STATED TEST**

Stanley's wording is directionally right but needs one concrete enforcement mechanism beyond prose.

| Check | What proves it |
| --- | --- |
| Surface shape | `go build ./...` with composition package exporting only a capability interface + `Close`/`Shutdown`; integration test lists methods on returned type — no `Commit`, `Open`, or store types |
| Import containment | CI allowlist (obligation 2): ingress/CLI packages must not import `statestore` or `ports` outbound persistence |
| Detached query results | Assembly test mutates returned task/message structs after `GetTask`/`GetMessage` and asserts store state unchanged on reload |
| **Store recovery** | **Import allowlist + unexported concrete type**, not a one-off type assertion test |

**Recovery question (god ask):** A type assertion back to `*FileStore` is easy to write and easy to forget. The durable check is: (a) composition returns an interface defined in the composition package; (b) concrete `*workspace` (or similar) is **unexported**; (c) CI fails if any non-allowlisted package imports `statestore`. A `testdata/forbidden/recover_store_test.go` that imports `statestore` and calls `Open` demonstrates the allowlist fires. That catches the real failure mode (new package reaching persistence), not a forgotten assertion in one test file.

A narrower interface over an **exported** concrete store remains insufficient even with a passing assertion test today.

### Obligation 2 — CI dependency/call allowlist (resolved symbols)

**Verdict: NOT CHECKABLE AS WRITTEN today. CHECKABLE WITH A STATED TEST using a weaker rule.**

**God ask — can we build resolved-symbol + method-value checking with Go tooling and zero module dependencies?**

| Layer | Feasible today? | Concrete check |
| --- | --- | --- |
| **Package import allowlist** | **Yes** | Shell + `go list -f '{{.ImportPath}} {{.Imports}}' ./...`; fail if any package outside `{statestore, core/task, core/ports, headless/host, …}` imports `…/statestore` or outbound `ports` mutation paths |
| **Text grep for `.Commit(`** | Yes but **insufficient** — Stanley is correct; method values and wrappers evade it |
| **Resolved symbols + method-value references** | **Not without new tooling** | Needs a custom checker using `go/parser` + `go/ast` (stdlib-only, ~100–200 lines) **or** a dev dependency on `golang.org/x/tools/go/analysis`. Not present in repo today; aspirational until implemented |
| **Forbidden fixture** | **Yes** | `testdata/forbidden/` (or similar) with a package that imports `statestore` and calls `Commit`; CI script runs checker and **requires non-zero exit** on that path, zero exit on production tree |
| **Test-only exception** | **Yes** | Allowlist explicitly includes `internal/adapters/statestore` and `*_test.go` in statestore package only |

**Weaker check that is real now:** package import allowlist + forbidden fixture that must fail. Promote to full AST call analysis when the stdlib checker lands; do not claim Phase 1 exit on grep alone.

This is the same class of fix as replacing "stranger could contribute" with a file checklist: name the weaker measurable rule until the stronger one exists.

### Obligation 3 — negative assembly authorization tests

**Verdict: CHECKABLE AS WRITTEN**

Standard integration tests through the composition entry point, once obligation 1 exists:

| Scenario | Expected | State after denial |
| --- | --- | --- |
| Direct Doing→Done via generic transition | `InvalidArgument` | Unchanged |
| Agent `AcceptTaskResult` | `Denied` | `AwaitingReview` |
| Wrong-recipient `AcknowledgeMessage` | `Denied` | No ack facts |
| Legitimate accept + ack | Success | Survives workspace reopen |
| Agent ingress selects `IsHumanReviewer: true` | **Denied** at host policy boundary before engine | Unchanged |

Concrete proof: `internal/headless` (or `host`) assembly test package opens a temp workspace via composition API only, runs each scenario, reloads store through composition queries. Maps directly to existing engine behaviour already proven in unit tests; the new requirement is the **entry path**, not new domain rules.

### Cost claim — rewrite required?

**Mostly true for production code; false for "no test work."**

| Stanley claim | Verdict |
| --- | --- |
| Engine command bodies unchanged | **True** — authorization stays in `engine.go`; composition wraps, does not rewrite |
| File format + transaction semantics unchanged | **True** — `Mutate` closure shape stands per H101-25 |
| Statestore contract tests unchanged | **True** — remain test-only exception |
| Tests that build engines directly unchanged | **True** — `newEngine` in `engine_test.go` can remain as unit tests |
| No assembly work | **False** — new composition package, assembly tests, and CI checker are **additive** obligations; unit tests cannot substitute |

Kevin's registry slice can use the same composition boundary without a storage redesign, as Stanley states.

### Summary table

| Obligation | Verdict | Concrete check |
| --- | --- | --- |
| 1 — private composition | CHECKABLE WITH A STATED TEST | Capability-only API; unexported concrete; import allowlist; detached-query assembly test |
| 2 — CI allowlist | NOT CHECKABLE AS WRITTEN (full symbol resolution); **CHECKABLE WITH A STATED TEST** (import allowlist + forbidden fixture) | `go list` import gate now; stdlib AST checker later; fixture must fail |
| 3 — assembly auth negatives | CHECKABLE AS WRITTEN | Integration tests through composition only |

---

## H101-24 acceptance review (2026-09-19)

Reviewed commit `2841635` (Phase 1 slice 3 — agent registry). Re-ran `go test -count=1 ./internal/core/task/ -run 'Agent|Register|Update'` — nine tests pass.

**Verdict: ACCEPTED WITH FOLLOW-UP**

### DoD checklist (code)

| # | Result | Note |
| --- | --- | --- |
| 1 Traceability | Pass | Implements `boundaries.md` line 24 RegisterAgent/UpdateAgent |
| 3 Honest verification | Pass | Denied paths assert state unchanged (`agent_test.go` lines 53–55, 156–158) |
| 6 Cross-artefact consistency | Pass with follow-up | Registry semantics match; `CreateTask` still accepts unregistered assignee (see below) |
| 11 Tests | Pass | Nine named tests prove claimed behaviours |
| 12–14 | Pass | No new boundary violations; commit identity correct |

### Kevin's five decisions — verified

| # | Claim | Verdict | Evidence |
| --- | --- | --- | --- |
| 1 | Existing agent ID → unconditional `Conflict`; idempotency via store replay only | **Pass** | `engine.go` lines 355–361 always `Conflict` if ID exists in mutate. Store replays same `(CallerAgentID, RequestID)` + fingerprint **before** mutate (`file.go` lines 136–138). `TestRegisterAgent_ExistingIDIsConflict` (new request ID) + `TestRegisterAgent_SameRequestIDReplays` |
| 2 | Agent ID structurally unpatchable | **Pass today** | `UpdateAgentRequest` has no ID field (`engine.go` lines 322–327); mutate only touches `DisplayName`/`ProfileID` (lines 401–406). `TestUpdateAgent_PatchesOnlyGivenFields` checks ID unchanged after patch |
| 3 | `canWriteAgentRecord` = self or human reviewer | **Pass** | `engine.go` lines 338–340; denied in `TestRegisterAgent_OtherAgentDenied`, `TestUpdateAgent_OtherAgentDenied`; allowed in human tests |
| 4 | `Provenance` + `LastUpdatedProvenance`, `Unverified` throughout | **Pass** | `records.go` lines 44–54; `agent_test.go` lines 34–38, 128–130 |
| 5 | No deregistration | **Pass** | No command implemented; matches boundaries command set |

### `IsHumanReviewer` — acceptable given H101-25 containment?

**Yes for this slice. No registry-specific mechanism beyond general containment.**

Reusing `caller.IsHumanReviewer` is the right primitive (`engine.go` lines 329–337). A forged human flag lets a caller update **any** agent record (`canWriteAgentRecord` is global, not per-target-role). The roster is a higher-value target than a single task — god's concern is valid.

This is **not** fixed inside the registry slice. It is the same host-policy hole Stanley named in H101-25. **H101-27 Phase 1 exit obligation 3** must include assembly proof that **agent ingress cannot receive `IsHumanReviewer: true`**, covering registry paths (`RegisterAgent`/`UpdateAgent` for another agent's ID) as well as accept/ack.

No fourth authority mechanism in the engine is needed or desirable.

### Decision 1 — store replay narrower than assumed?

**No.** Replay short-circuits before `Mutate` with matching fingerprint. A replayed successful register returns the original receipt without re-hitting "already registered." Different payload + same request ID → `Conflict` at store (`file.go` line 140), not tested by name but follows established statestore contract.

### Decision 2 — test that fails if ID becomes patchable?

**No such test exists.** Structural unpatchability via missing struct field is true today; `TestUpdateAgent_PatchesOnlyGivenFields` only checks ID unchanged after a legitimate display-name patch. Adding `NewAgentID *string` to `UpdateAgentRequest` would not fail any test until someone also added assignment code.

**Follow-up (low priority):** optional compile-time guard is unnecessary; a code-review convention plus the struct comment is sufficient for Phase 1. Not a reject.

### "Zero divergences" — unverified; one found

| Gap | Note |
| --- | --- |
| **`CreateTask` does not require registered `AssigneeID`** | `engine.go` lines 72–75 still accept any assignee. Registry exists but tasks are not wired to it. Undeclared; continuation of H101-12 divergence (c). Not introduced by this slice, but zero divergences claim is **incorrect** if counted against the full Phase 1 surface |

Engine-wide `StateStore.Commit` bypass remains logged (H101-19 / H101-25).

### Test assessment

All nine tests prove what their names claim. No restart-durability test for registry — not required by card; agents persist via same `state.json` path as tasks.

### Follow-up cards

- **H101-25 assembly** (existing): agent ingress cannot set `IsHumanReviewer`; include roster hijack scenario (`UpdateAgent` on another agent's ID with forged human flag denied at host).
- **CreateTask assignee validation** (existing): require `AssigneeID` registered before task create, or document intentional deferral.

---

## H101-19b acceptance review (2026-09-19)

Reviewed commit `a34f51d` (CI import containment — H101-27 obligation 2 weaker rule). Re-ran checker: tree exits 0; `./testdata/forbidden` exits 1 with both violations named.

**Verdict: ACCEPTED WITH FOLLOW-UP**

**Phase 1 exit item 3 (Containment): PARTIALLY SATISFIED** — import gate + forbidden fixture are in place; allowlist hygiene and `CommitRequest` construction inside permitted packages remain open.

### DoD checklist

| # | Result | Note |
| --- | --- | --- |
| 3 Honest verification | Pass | CI requires fixture to **fail** (`.github/workflows/ci.yml` lines 27–32); not print-and-pass |
| 11 Tests | Pass | Forbidden fixture is the demonstrated negative case |
| 14 Commit identity | Pass | `a34f51d` authored correctly |

### What this card delivers (matches H101-27 weaker rule)

| Requirement | Status |
| --- | --- |
| `go list` package import allowlist | **Done** — `scripts/check-import-allowlist.sh` + `docs/architecture/import-allowlist.txt` |
| Forbidden fixture must fail check | **Done** — `testdata/forbidden/reach_persistence.go` |
| Documented as import containment, not symbol analysis | **Done** — allowlist header lines 3–6; script lines 4–6 |
| Wired in CI | **Done** — workflow steps after vet |

Claudio's declared limit (cannot detect `CommitRequest` construction inside an already-permitted package) is **explicit and honest** — not hidden.

### God question 1 — `internal/host` vs `internal/headless`

**Exactly one** production package outside `statestore` itself should construct/inject the store. **Not both.**

| Package | At `a34f51d` | Ruling |
| --- | --- | --- |
| `internal/host` | Exists (`doc.go` stub; no store import yet) | **Keep** — matches `boundaries.md` hosting section and existing package name |
| `internal/headless` | **Does not exist** | **Remove from allowlist** until Kevin lands composition under that name, **or** replace `host` with `headless` if that is the chosen name — never both |

Pre-authorizing two doors where the design intends one boundary is load-bearing drift. Justification must live in `import-allowlist.txt` next to the single entry, e.g. "sole composition root; constructs FileStore privately; does not expose persistence."

If Kevin wires composition in `internal/host`, delete the `headless` lines. If he chooses `internal/headless`, delete `host` when that package lands.

### God question 2 — `CommitRequest` inside permitted packages

**Within the weaker H101-27 rule: acceptable.** Item 3 as written mentions both import containment and `CommitRequest` construction. This script covers **imports only**.

| Layer | Covered by H101-19b? |
| --- | --- |
| New package imports `statestore` | **Yes** |
| Allowed package builds `CommitRequest` with hostile `Mutate` | **No** — documented limit |

That gap is why obligation 3 (assembly authorization negatives) and future AST analysis remain on the Phase 1 exit. **Does not reject H101-19b** — it was never promised to discharge the full item 3 wording alone.

### God question 3 — allowlist entry for non-existent package

**Worth a check.** A phantom entry fails silently — nothing imports, nothing warns, but a second door appears approved in writing.

**Weaker self-correction:** Kevin hits CI failure if he imports store from an unlisted package name. That does **not** catch the opposite failure (allowlist lists a package he never creates, leaving a documented door unused).

**Follow-up:** add to `check-import-allowlist.sh`: every non-comment allowlist path except `statestore` itself must resolve via `go list` (or directory exists under `internal/`). ~10 lines.

### Declared divergence — verified

One divergence, stated upfront: no `CommitRequest` call-site detection inside permitted packages. **Accurate.** Treated as accepted limitation per H101-27, not a hidden zero.

### Follow-up before calling item 3 satisfied

1. Collapse allowlist to **one** store-constructor package with inline justification.
2. Optional: allowlist paths must exist in tree.
3. H101-19 obligations 1 and 3 (composition + assembly tests) — still required for item 3 completion.

---

## H101-19 obligations 1 and 3 acceptance review (2026-09-19)

Reviewed commit `8dc828f` (composition boundary + assembly tests). Allowlist hygiene from `ea4425c` verified. Seven assembly tests pass (`go test -count=1 ./internal/host/`).

**Verdict: ACCEPTED WITH FOLLOW-UP**

### Phase 1 exit items 1–6

| Item | Status | Evidence |
| --- | --- | --- |
| **1 Product cycle** | **UNSATISFIED** | H101-31 (end-to-end cycle through composition) still in flight. This card delivers the boundary, not the full ≥2-agent walkthrough |
| **2 Assembly authorization** | **SATISFIED** | Seven tests through `host.Capabilities` only (`host_test.go`): Doing→Done rejected; agent accept denied; wrong-recipient ack denied; state unchanged on denials; accept+ack survive reopen; agent ingress cannot forge human authority (with positive control for real reviewer) |
| **3 Containment** | **SATISFIED** | Import allowlist + forbidden fixture (`ea4425c`); single `internal/host` door with inline justification. `CommitRequest` inside `host` still not statically proven — accepted H101-27 limitation. **H101-34 CI assertion weakness does not block discharge** (see below) |
| **4 Composition surface** | **SATISFIED** | `Capabilities` interface only; unexported `workspace` (`host.go` lines 78–85); no `IsHumanReviewer` on any export; queries cloned (`cloneTask`/`cloneMessage`/`cloneAgent`); `TestAssembly_QueryResultsAreDetached` + `TestAssembly_CapabilitiesExposeNoPersistenceMethods` |
| **5 Unit tests** | **SATISFIED** | `internal/core/task` unit tests unchanged and supplementary |
| **6 Zero network** | **SATISFIED** | Per god verification on `internal/host` deps |

**Items 2 and 4 discharge with `8dc828f`.** Item 3 discharges with `ea4425c` + `a34f51d` + this card's composition proof. Item 1 waits on H101-31.

### Design — reviewer set at `Open`, not per call

**Pass.** `host.Open(root, workspaceID, reviewerAgentIDs)` fixes reviewers at assembly (`host.go` lines 93–104). `caller()` derives `IsHumanReviewer` from set membership only (`lines 107–112`). No exported method accepts an authority flag. This closes the H101-25 boolean bypass for agent ingress paths without changing the engine.

`TestAssembly_AgentIngressCannotSelectHumanAuthority` proves denial is membership-based (forged summary in payload + engineer denied; same call succeeds for `reviewerID`).

### H101-34 — forbidden-fixture step accepts any non-zero exit

**Valid finding. Does not block item 3 discharge today.**

`.github/workflows/ci.yml` lines 28–32: forbidden step passes when the checker exits non-zero. Exit **2** (allowlist config error from `check-import-allowlist.sh` lines 56–58) satisfies that condition without proving the fixture was caught as an import violation (exit **1**).

**Why item 3 still discharges:** the **containment step runs first** (line 25–26) and also exits 2 on phantom allowlist entries — CI fails before the forbidden step runs. Guarantee rests on step order, not on the forbidden assertion.

**Follow-up (H101-34):** require exit code **1** specifically on the forbidden fixture run, e.g. capture `$?` and fail unless exactly 1. Belt-and-suspenders; not a reject of H101-19b or this card.

### Reflection name denylist on `Capabilities`

`TestAssembly_CapabilitiesExposeNoPersistenceMethods` rejects methods named `Commit`/`Recover`/`Load`/`Replay` (`host_test.go` lines 297–308).

**Adequate as the stated test for obligation 1**, paired with import allowlist (the durable enforcement). **Fails open** if someone adds `Persist()` or exposes store via a differently named wrapper — the name denylist alone would not catch it.

Acceptable per H101-27 ("stated test + import gate"); not a substitute for future AST analysis. Unexported concrete type prevents type assertion to `*FileStore` (test lines 310–315).

### Report count error

Commit subject says "eight assembly tests"; **seven exist and pass**. Code is right; report is wrong. Minor documentation hygiene, not a reject.

### `RecordMessagePublished`/`Processed` without caller

Kevin exposes them on `Capabilities` with no caller parameter (`host.go` lines 65–66, 150–156), matching engine `CallerScope{}` treatment.

**Agree: fix belongs in H101-22 (mailbox adapter)**, not here. Phase 2 mailbox must enforce causal ordering and bind publication/process recording to adapter identity. Host pass-through is correct for Phase 1.

### Follow-up cards

- **H101-34**: forbidden-fixture CI step must assert exit code 1, not any non-zero.
- **H101-22** (existing): mailbox authority for delivery-fact recorders.

---

## H101-31 acceptance review (2026-09-19)

Reviewed commit `fc18dde` (`TestPhase1_ProductCycleThroughComposition`). Test passes; file imports only `domain`, `task`, `host` — no `statestore`.

**Verdict: ACCEPTED**

**Phase 1 exit item 1: SATISFIED**

**All six Phase 1 exit items are now discharged.** Phase 1 is exit-ready pending milestone review (items 1–6); H101-34 remains belt-and-suspenders follow-up, not a blocker per prior ruling.

### Does "hand off" discharge on a task-linked message?

**Yes.** Item 1 does not mean reassignment.

| Source | What "hand off" means |
| --- | --- |
| Item 1 wording (this doc) | Comma-separated step between **assign** and **report result** — coordination between agents, not a synonym for changing `AssigneeID` |
| `definition.md` line 11 | "A handoff reaches a **named recipient**, remains available across sessions, and **links to the task**" — message semantics |
| `definition.md` line 17 | Minimum cycle: assign, **hand off**, blocker, review — handoff is distinct from assign |
| Command surface | `AssigneeID` set only at `CreateTask` (`engine.go` lines 72–75); no reassignment command exists |

The test's task-linked `SendMessage` to `analystID` with `TaskID` set, followed by recipient `AcknowledgeMessage`, is the handoff Angela's product definition describes. **Reassignment would be a separate capability** ("correct ownership," `definition.md` line 23) — a `boundaries.md`/product gap if required later, **not** a reason to fail item 1 today.

Claudio's declared assumption is **correct**, not a workaround.

### Cycle coverage

| Item 1 step | Test evidence (`phase1_cycle_test.go`) |
| --- | --- |
| Assign | `CreateTask` with `AssigneeID` (lines 44–48) |
| Hand off | Task-linked message + ack (lines 57–67) |
| Report result | `ReportTaskResult` ×2 (lines 69–74, 89–94) |
| Human reject | `RejectTaskResult` (lines 75–80) |
| Human accept | `AcceptTaskResult` on `cycle-result-2` (lines 95–99) |
| Acknowledge | Included in handoff (lines 63–67) |
| ≥2 registered agents + human | Engineer, analyst registered; reviewer is human authority set at `Open` (lines 25, 32–41) |
| Composition only | No store/engine imports (lines 12–14) |
| Reopen | `Close` + `host.Open` (lines 100–110) |

**Not in item 1:** blocker cycle (`definition.md` line 17 includes it; item 1 list does not). Not a fail.

### God question 1 — reject branch proof

**Sufficient.** After reject, test asserts `Doing` (lines 81–87), re-reports under new `cycle-result-2`, accepts that ID, reopen asserts `Done` with `cycle-result-2` (lines 121–122). That proves reject returned the task for rework and the second result was accepted — not a no-op visit.

**Could be stronger:** assert rejected result remains in history with `Rejected` decision — not exposed on `GetTask` today. Not required to discharge item 1.

### God question 2 — reopen gaps

**Adequate for item 1.** Re-verified after reopen: both agent registrations, task `Done` + correct `CurrentResultID`, handoff `TaskID` link, acknowledgement by analyst.

**Not re-checked (acceptable gaps):** message body text, publication/process delivery facts (test never records them; ack-before-publish is valid per boundaries), assignee field on task, provenance fields. None are required by item 1 wording.

### God question 3 — "no other gap" claim

**Treat as unverified; no second gap blocks item 1.**

| Potential gap | Blocks item 1? |
| --- | --- |
| No reassignment command | **No** — handoff is message-based (above) |
| `CreateTask` without registered assignee | **No** — test registers agents first; known H101-12 follow-up |
| No `RecordMessagePublished`/`Processed` in cycle | **No** — not required for handoff proof |
| Blocker path omitted | **No** — not in item 1 list |

No evidence Claudio worked around an unrecognised second gap.

---

## H101-41 acceptance review (2026-09-19)

Reviewed commit `24a2022` (CLI skeleton — read-only `harnessing task`). Re-ran `go test -count=1 ./cmd/harnessing/...` — seven tests pass. Re-verified allowlist (tree 0, fixture 1) and `go list -deps ./cmd/harnessing` (no `net`).

**Verdict: ACCEPTED WITH FOLLOW-UP** — appropriate first slice; does not discharge Phase 2 exit.

### Phase 2 exit item deltas

| # | Status after `24a2022` | Note |
| --- | --- | --- |
| 1 | **Unsatisfied** | Read-only `task` query only; no §2 walkthrough, no subprocess cycle test |
| 2 | **Unsatisfied** | No crash-restart path |
| 3 | **Unsatisfied** | No `adaptercontract` suite, no second adapter; see enumeration gap below |
| 4 | **Partially satisfied** | Store-import half **yes**; UI import allowlist half **no** |
| 5 | **Unsatisfied** | No mailbox adapter |
| 6 | **Unsatisfied** | No first-run disclosure in CLI |
| 7 | **Unsatisfied** | Phase 3 ops not exposed or asserted `Unsupported` |
| 8 | **Satisfied** | See god Q1 below |
| 9 | **N/A** | Policy; unit tests do not claim to substitute for items 1 or 3 |

### God question 1 — item 8, zero network

**Discharges item 8.** The item means `go list -deps` on the CLI package tree finds no `net` / `net/http` — not a runtime egress harness (that is Phase 4). `cmd/harnessing` is the current equivalent of `internal/adapters/cli/`; no stronger requirement is stated. Optional follow-up: add an explicit CI step so the check cannot regress silently (Phase 1 core used the same manual pattern).

### God question 2 — item 4, containment

**Store-import half: discharged for this slice.** Allowlist unchanged; `cmd/harnessing` imports only `internal/host` and `internal/core/domain`; no `statestore`, no `ports`, no allowlist entry needed. Forbidden fixture still exits **1** (CI asserts exactly 1).

**UI import allowlist half: not discharged.** Item 4 also requires presentation packages not import `host` / `core/task` (ADR 0003 direction). This card correctly imports `host` as an interim thin wrapper — that is the right pattern until `FrontendSession` lands, but the extended gate is still open.

### God question 3 — stable errors and item 1

**Right shape; not yet required to accept this slice.** Item 1 does not name an error format today, but a subprocess walkthrough will need stable machine-readable denial/conflict codes. `describeError` rendering `"<ErrorCode>: <detail>"` with `NotFound` asserted in `TestRun_TaskUnknownIDIsLegibleFailure` is the correct foundation. **Recommend:** when item 1's walkthrough test is written, assert at least one `Denied` and one `Conflict` path render the code prefix — do not add to item 1 wording until then.

### God question 4 — zero divergences

**Not a clean bill; no undeclared divergence found.** Kevin volunteered the real gap: **`Capabilities` has no list/enumeration query** — workspace overview and status views cannot be built from get-by-ID alone. Forwarded to Stanley for H101-40 / ADR 0003; **item 3 risk:** two adapters that only display records whose IDs are already known could pass a thin shared suite while demonstrating little. Contract must require snapshot/list semantics (ADR UI-06) before item 3 can discharge.

**Scope boundaries (not divergences):** no mutation commands, no mailbox, no disclosure — correctly deferred.

### Design calls verified

| Call | Result |
| --- | --- |
| Authority from CLI flags only (`-workspace`, `-workspace-id`, `-reviewer`) | **Pass** — `runTask` passes only flag values to `host.Open`; no workspace-content authority |
| No env-var reviewer default | **Pass** — deliberate; agree with god's rating |
| `Close` on every exit path | **Pass** — `TestRun_TaskClosesWorkspaceEvenOnQueryFailure` positive control |

### Follow-up (non-blocking for this card)

- Enumeration / snapshot query for item 3 (Stanley H101-40 amendment).
- UI import allowlist when `FrontendSession` splits presentation from `host`.
- Item 1 walkthrough subprocess test with stable error assertions on denial paths.

---

## H101-45 / H101-16 acceptance review (2026-09-19)

Reviewed commit `1eda92b` (unconditional CLI disclosure). Re-ran `go test -count=1 ./cmd/harnessing/ -run Disclosure` — pass.

**Verdict: ACCEPTED**

**Phase 2 exit item 6: SATISFIED**

### Item 6 checklist

| Requirement | Result |
| --- | --- |
| Automated test | **Pass** — `TestRun_DisclosureIsUnconditionalAndKeepsStdoutScriptable`, `TestRun_DisclosurePrecedesFreshWorkspaceCommand` |
| ADR 0002 sentence present | **Pass** — exact wording in `disclosure` constant line 3 (`run.go`) |
| Before coordination commands | **Pass** — printed at top of `run()` before switch; stronger than item 6 minimum |
| No validation-looking cues | **Pass** — plain text only |
| README regression | **Pass** — README local-only paragraph unchanged |

### God question — does “unconditional” mean every invocation?

**Yes.** ADR 0002 requires the line “unconditionally visible during Phase 2 use, not just at onboarding.” A persisted once-per-workspace marker would **not** satisfy that. Kevin’s every-invocation reading is the correct and **stronger** interpretation. Item 6 wording updated above to say so explicitly (was ambiguous “first use in a fresh workspace”).

### God question — stderr before command handling?

**Correct placement.** Item 1’s subprocess walkthrough must parse stdout for task/status output; disclosure on stderr keeps pipelines scriptable. Tests assert stdout is clean (`harnessing dev\n` only). Item 1’s future walkthrough test should parse stdout and may ignore stderr or assert disclosure as a separate prefix check — either works.

### God finding 1 — opens with “runs completely locally”

**Not a criteria failure; route to Angela for voice.** Item 6 does not govern lead-sentence ordering. Read in full, the opening clause is immediately qualified (egress, non-confinement, no data-stay guarantee) — same sentence as README/`definition.md`. Creed’s H101-30 excerpt hazard applies to **any** use of that phrase: a partial quote of the first clause alone would mislead. That is a standing constraint on quoting, not proof the in-product text is dishonest.

**Judgement call:** whether reassurance-then-qualify reads better than threat-first for clone-and-run users is **product/customer experience**, not acceptance criteria. I do not block item 6 on it. If Angela wants threat-first ordering, that is a copy edit, not a phase-exit miss.

### God finding 2 — lines 2 and 3 overlap

**Agree, minor.** Line 2 paraphrases; line 3 is ADR 0002 verbatim. Redundant for a careful reader; defensible while ADR exactness is required. **Follow-up copy tighten** for Angela or Claudio — not a blocker.

### Design call verified

Every invocation + stderr-first: **correct** for public clone-and-run and scriptable stdout. No persisted marker.

---

## H101-47 / H101-20 acceptance review (2026-09-19)

Reviewed commit `d2f4266` (flock lock recovery). Re-ran `go test -count=1 ./internal/adapters/statestore/ -run 'Recover|SecondOpen'` — pass. Read `docs/architecture/h101-20-lock-recovery.md`.

**Verdict: ACCEPTED WITH FOLLOW-UP**

**Phase 2 exit item 2: PARTIALLY SATISFIED**

### What item 2 has two parts

| Part | Status after `d2f4266` |
| --- | --- |
| Lock recovery prerequisite (H101-20) | **Satisfied on linux/darwin** — flock, subprocess SIGKILL test, `SecondOpenIsBusy` still passes |
| Full item 2 proof (item 1 cycle → kill CLI → `harnessing` reopen) | **Unsatisfied** — no harnessing subprocess crash test yet |

H101-20 was the **blocker** named in item 2; it is now cleared on exercised platforms. Item 2 itself is not fully discharged until item 1's walkthrough exists and a crash-reopen test runs through `harnessing`.

### God question — Windows `Unsupported` vs item 2

**Item 2 discharges on platforms we actually exercise, not on an undefined "all supported platforms" set.**

- **linux** (CI) and **darwin** (local dev): lock recovery is implemented and crash-tested → H101-20 part of item 2 **yes**.
- **Windows**: honest `Unsupported` → item 2 **no** on Windows until lock support lands with verified crash-release, or Windows is explicitly excluded from Phase 2 scope in writing.

ADR 0001's platform matrix is still **UNKNOWN**; this card does not settle it. Phase 2 exit should not be blocked on Windows while the matrix is unsettled, but item 2 must record Windows as not yet satisfied. **Owner decision** if Phase 2 claims cross-platform CLI before Phase 3.

### God question 1 — is the crash test honest?

**Yes.** The poll waits for kernel process teardown after `SIGKILL`, not a lock timeout — there is none. The test also asserts `Busy` while the helper is alive (lines 73–79), which is the negative control. Five-second deadline with 20 ms sleep is proportionate; failure message would catch a real regression.

### God question 2 — NFS / network filesystem

**"We do not claim to support it" is enough for item 2.** `boundaries.md` and `threat-model.md` already say unverified network filesystems are unsupported. `flock` over NFS is unreliable without correct lock-daemon configuration; this card correctly adds no silent detection. **Follow-up (non-blocking):** if clone-and-run users on NFS hit opaque `Busy`, that is an operational hazard outside claimed scope — not an item 2 miss today.

### God question 3 — one declared divergence (no PID fallback)

**Accepted.** PID/start-time check would reintroduce PID-reuse hazard flock avoids. No undeclared divergence found.

### Mechanism vs boundaries.md

**Satisfies** the ban on stealing locks by elapsed time — recovery is kernel descriptor lifetime, not wall-clock theft. Documented manual `.lock` removal remains as fallback in `h101-20-lock-recovery.md`.

### Follow-up

- Harnessing subprocess crash test once item 1 lands.
- Windows lock implementation + crash verification, or explicit Phase 2 platform exclusion.
- Optional: CI runs `lock_recovery_test.go` on linux only (`//go:build !windows` already excludes Windows test file).

---

## H101-53 — ADR 0003 CI assertions for item 3 (2026-09-19)

Reviewed committed ADR 0003 (`9055124`). Stanley owns contract shape; **Kelly owns executable CI assertions** mapped here. Do not implement the suite in this card.

### (a) Item 4 split vs committed ADR — still holds

Re-checked against `9055124`, not the earlier draft from memory. **The split survives unchanged:**

| Half | Committed ADR says | Status |
| --- | --- | --- |
| **Store-import** | Only reviewed assembly (`host`) constructs persistence; no storage allowlist exception for UI (ADR §Prohibitions) | **Satisfied** on current tree (`24a2022` — `cmd/harnessing` imports `host` only, not `statestore`/`ports`) |
| **UI import allowlist** | Presentation packages must not import `host`, `core/task`, storage, or outbound ports; only assembly imports `host` (ADR §Prohibitions). CI gate via `check-import-allowlist.sh` H101-43 rule; forbidden fixture exit **1**. | **Partially** — gate enforced (`306c6fa`); `cmd/harnessing` interim exception remains until `FrontendSession` migration removes direct `host`/`Capabilities` use |

`FrontendSession`, `GetSnapshot` enumeration (UI-06), and the adaptercontract suite location are all **present in the committed ADR** — Stanley did not drop the design H101-42 referenced. (ADR header still says "Proposed for review"; technical content is what item 3/4 reference.)

### (b) What CI must assert for item 3 to discharge

Both adapters run the **same** `internal/adaptercontract` scenarios through **real** drivers. Each scenario asserts against **fixture-embedded expected domain state** compiled before either adapter runs, **and** cross-adapter equality on normalized observations (ADR §canonical observations). Suite dependency set must be checked (no `statestore`/`ports` imports in contract tests).

| ID | CI must assert (both adapters) | Checkable today? |
| --- | --- | --- |
| **UI-01** | Malformed input → non-zero exit, stable error code, **no** workspace mutation (reopen or snapshot unchanged). Valid command → exact IDs/revisions/body preserved vs fixture. Display activity alone never implies ack/accept/authority. | **No** — needs write commands + FrontendSession + suite |
| **UI-02** | Success paths render receipt distinct from completed work (revision/result state). Errors use stable `domain.ErrorCode` prefix; never map Denied/Conflict/IOFailure to success. **IOFailure uncertainty:** see gap below. | **Partially** — read-path `NotFound` shape proven (`24a2022`); write + IOFailure paths need surface |
| **UI-03** | Same `request-id` retry → identical receipt; changed payload → Conflict. No silent new-ID retry after uncertain mutation. **Generated entity IDs** surfaced on create so interrupted caller can recover (stdout or documented field). | **No** — write commands (H101-51); caller `-request-id` design exists uncommitted |
| **UI-04** | Full cycle: register ≥2 agents, task, blocker, linked message + ack, report, reject, accept — identical domain outcomes. `AwaitingReview` ≠ `Done`. Wrong principal → Denied, state unchanged. | **No** — needs session surface + mutations via CLI/throwaway |
| **UI-05** | Provenance fields visible; four message delivery facts including **absent** facts; ADR 0002 disclosure present; no validation badges on manual content. | **Partially** — disclosure on every run (`1eda92b`); delivery facts + provenance rendering need mailbox + write path |
| **UI-06** | **No injected entity IDs** for first overview: discover fixture via `GetSnapshot`, then observe from returned cursor through concurrent change (other adapter or helper). **`CursorExpired` → reload** — must include **host restart with stale cursor** (empty replay floor), not only buffer eviction. **Commit/publication ordering** — must include **concurrent commits where R+1 can publish before R** (goroutine interleaving within single writer), not only publish-vs-subscribe race. Deduplicate replays; snapshot refresh not presented as full event history. Detachment/mutation probe per ADR §detachment proof. Two sessions on same host (ADR H101-70 — durable cross-process log not required). | **No** — implementation gaps per H101-70 (`2e72fd7`); suite not built |
| **UI-07** | Detach/replace UI does not close other session or cancel work. **Graceful** reopen retains records + request-id replay. | **No** — needs session detach + multi-session test |
| **UI-08** | Returned view values detached (mutation probe on nested data does not persist). **Phase 3 ops on `FrontendSession`:** `StartRun`, `StopRun`, `SetRunBudget` each return stable `Unsupported` with non-empty detail (H101-80). Run output stream separate from domain events (fake stream tests excluded from item 3 verdict). | **Partially** — Phase 3 `Unsupported` surface on session (`20e0caa`); detachment probe still needed |

**Cross-cutting (all scenarios):** throwaway adapter does not import CLI or share dispatch; crossover scenario (A writes, B continues) on **shared** workspace; concurrent observer sees external commits; `adaptercontract` imports only allowed packages.

### (c) God question 1 — UI-02 and `IOFailure` "applied but uncertain"

**`boundaries.md` cannot express this as a separate stable code today.** Listed codes are `IOFailure` and `RecoveryRequired` only (`boundaries.md` line 13; `domain/errors.go`). The store **does** return `IOFailure` with detail `"commit applied but durability unconfirmed: …"` (`file.go` line 270) — meaning lives in **Detail**, not `ErrorCode`.

**Gap:** UI-02 contractually requires adapters to show uncertainty, not "nothing happened." Adapters **can** satisfy this by rendering `IOFailure` plus detail substring and retaining the caller's request ID — but that is **fragile** (detail is supplementary per boundaries). **Recommendation for implementation card:** either (1) contract test asserts stderr/stdout contains `IOFailure` and `durability unconfirmed` on injected fsync fault, **or** (2) add a dedicated stable code (e.g. extend with `DurabilityUncertain` or map to `RecoveryRequired`) in a scoped domain change. **Item 3 can discharge with (1); (2) is cleaner long-term.**

### (c) God question 2 — UI-03 generated IDs on CLI as shipped

**Not applicable on committed `9055124`** — only read-only `task` exists. **Requirement for H101-51:** write commands must require caller `-request-id` (already designed uncommitted) and **print created entity IDs** (task ID, message ID, etc.) on success, not only receipt request ID. `printReceipt` showing request ID is necessary but not sufficient for "generated IDs surfaced."

### (c) God question 3 — how CI catches "independently expected outcomes"

**Load-bearing rule:** expected state is a **fixture struct literal** (or checked-in golden domain object) built from the scenario definition **before** either adapter runs. Per adapter: `assertNormalizedEqual(observed, expectedFixture)`. **Secondary:** `assertNormalizedEqual(cliObserved, throwawayObserved)` — never the only check.

**CI catches cheating if:**
1. Each scenario file imports `expected` from a `testdata/adaptercontract/<scenario>/want.go` (or inline literal) with no reference to adapter output.
2. Code review / optional lint: forbid assigning `expected = cliResult` in contract tests.
3. Negative regression test: a deliberately broken "echo adapter B copies A's parsed output" must fail the fixture assertion even if A==B.

### (c) God question 4 — UI-07 vs item 2

**Confirmed: item 3 does not discharge item 2.** ADR UI-07 line 47 explicitly separates crash restart ("separate inherited recovery obligation") from graceful reopen. The adaptercontract suite may prove UI-07 graceful paths only. **Item 2** remains the dedicated `harnessing` subprocess `SIGKILL` test after item 1's full cycle (partially satisfied via H101-20 lock mechanism only).

---

## H101-55 — platform ruling vs item 2 (2026-09-19)

Reviewed ADR 0001 H101-50 amendment (`bcba765`). **Amends H101-47 platform language** — local dev on a laptop is not milestone discharge without a recorded evidence manifest.

### Does the lock half of item 2 still discharge on darwin?

**No — not for milestone/exit purposes as H101-47 recorded it.** Revised per target:

| Target | Lock half (H101-20) evidence today | Discharges lock prerequisite? |
| --- | --- | --- |
| **linux/amd64** | `TestOpen_RecoversAfterOwnerCrash` runs in CI on `ubuntu-latest` (`d2f4266`) | **Yes** — native CI evidence (record exact runner image in manifest per ADR) |
| **darwin/arm64** | `h101-20-lock-recovery.md` OBSERVED STATE (manual darwin/arm64, 2026-09-19); tests pass when run locally | **Not yet** — engineering confidence only until **macOS CI runner** **or** milestone manifest records commit, exact macOS build, commands, and outputs |
| **Windows** | `Unsupported` on workspace open (`lock_windows.go`) | **N/A** — excluded; not a partial discharge |

**Answer to god:** either **card a macOS runner**, or **require a written milestone evidence manifest** for darwin-native runs. I will not treat undisclosed laptop runs as discharge.

**Full item 2** (item 1 cycle → `SIGKILL` → `harnessing` reopen) remains **unsatisfied on both targets** — unchanged.

### God question 1 — items 8 and 3 vs per-milestone environment recording

| Item | Ubuntu-only CI adequate? | Why |
| --- | --- | --- |
| **8** (zero network, `go list -deps`) | **Yes** | OS-agnostic import graph; same result on any `GOOS` that compiles |
| **3** (adaptercontract UI-01–08) | **No** | ADR 0001: both in-scope targets owe **native** adapter-cycle evidence; suite must run on `linux/amd64` CI **and** `darwin/arm64` (runner or recorded manifest) |
| **2** (crash reopen) | **No** | Same native requirement; lock-only proof on one OS does not discharge the other |

**Per-milestone recording** applies to **platform-behaviour claims** (lock, crash, adapter cycle), not to static dependency checks.

### God question 2 — Windows exclusion assertion

**Worth one Phase 2 exit check — not item 2 discharge.** Windows is excluded from coordination; regressions to silent partial support are a real risk.

**Add (lightweight):** on `GOOS=windows` build/test or a Windows CI job: `harnessing version` and `help` succeed; any workspace-open path returns **`Unsupported`** with stable code — not panic, not success, not unprotected writes. This asserts **defined exclusion**, not support. Does not block phase exit on Windows features.

### God question 3 — zero dependencies vs H101-27 obligation 2 AST checker

**Revisit warranted; Phase 1 item 3 discharge unchanged.**

ADR 0001 now records zero external dependencies as **preference, not constraint**. The stronger obligation-2 check (resolved symbols / `CommitRequest` construction) is therefore **not technically forbidden** — it is **unpaid for**.

| Path | Status |
| --- | --- |
| Import allowlist + forbidden fixture (current) | Still discharges Phase 1 item 3 — **done** |
| Stdlib-only `go/ast` checker (~100–200 lines) | **Feasible without new modules** — always was; preference change does not affect this path |
| `golang.org/x/tools/go/analysis` | **Now arguable** with explicit ADR on pin, licence, maintenance — no longer vetoed by zero-dep alone |

**Recommendation:** keep weaker gate for Phase 2 unless owner funds stronger checker. Preference change opens the door; it does not require upgrading containment before phase exit.

### CI / wording changes required

1. **macOS runner** (or formal milestone manifest process for darwin-native evidence) — **card it** for items 2 and 3.
2. **Pin and record** `ubuntu-latest` image ID in milestone manifest (ADR requirement).
3. **Windows exclusion smoke test** — add to Phase 2 exit checklist (new sub-bullet under item 2 or item 7).
4. **H101-47** platform paragraphs superseded by this section for discharge claims.

---

## H101-57 acceptance review (2026-09-19)

Three slices on `a9d3d07`: `d906a9c` (session surface), `3659e6a` (write commands + hold), `a9d3d07` (IOFailure uncertainty rendering). Re-ran named tests — pass. Did not credit uncommitted `phase2_crash_reopen_test.go` (H101-49 in flight).

### Verdicts

| Slice | Verdict |
| --- | --- |
| `d906a9c` Claudio — FrontendSession + GetSnapshot | **ACCEPTED WITH FOLLOW-UP** |
| `3659e6a` Kevin — write commands + hold | **ACCEPTED WITH FOLLOW-UP** |
| `a9d3d07` Kevin — UI-02 IOFailure rendering | **ACCEPTED WITH FOLLOW-UP** |

### Phase 2 exit item deltas

| # | Moves? | After these three commits |
| --- | --- | --- |
| 1 | **No** | Write path exists; **not discharged** — see Q1 |
| 2 | **Slight** | `hold` subprocess kill/recovery (`hold_test.go`) advances CLI-level crash fixture; full item-2 proof still needs post-§2-cycle subprocess kill (H101-49) + darwin manifest |
| 3 | **Slight** | Session surface lands; **not discharged** — no `adaptercontract` suite, no second adapter, UI-06 incomplete |
| 4 | **No** | CLI still imports `host` directly; UI allowlist gate still open |
| 5–7 | **No** | Unchanged |
| 6, 8 | **No** | Already satisfied |
| 9 | **N/A** | `run()` tests are valid scaffolding; do not substitute for item 1 subprocess proof |

### God question 1 — does `TestRun_FullCycleThroughWriteCommands` discharge item 1?

**No.** Item 1 names a **subprocess** of the shipped `harnessing` binary, explicitly **not** in-process `run()` from the test package.

`TestRun_FullCycleThroughWriteCommands` is valuable — it proves the dispatch path works — but it is **scaffold**, not discharge. Same for error-rendering tests (`Denied:`, `Conflict:`).

**Also missing from any committed test vs item 1 / §2:** ≥2 registered agents; blocker report and resolve; human **reject then accept replacement**; close and **reopen** without resending; status view with reporter identity for all states. `TestRun_SendAndAckCycle` covers messaging separately but is not in the full-cycle test.

**What item 1 needs next:** one `exec.Command` subprocess test (or script in CI) driving the full §2 walkthrough through argv only.

### God question 2 — Claudio's cursor divergence vs UI-06

**UI-06 is not satisfiable yet.** Revision token without subscription/replay cannot observe from cursor, deduplicate replays, or reload on `CursorExpired`.

**This is a missing card, not a scope dodge.** Claudio correctly declared the divergence. `GetSnapshot` + empty collections + detachment satisfy the **enumeration/discovery** half of UI-06; **event observation** is a separate delivery (subscription/replay plumbing per ADR 0003 §stream semantics). Card it before item 3 can discharge UI-06.

### God question 3 — IOFailure test via corrupt state vs fsync

**Honest for the rendering contract; not proof of the fsync path.**

All `IOFailure` responses share one render path (`flags.go`); the test correctly exercises that path at the **Code** level. Corrupt-state IOFailure is not the same failure mode as post-rename fsync uncertainty — but the **UI obligation** is "never render IOFailure as plain failure / nothing happened," which the test does prove.

**Follow-up:** when H101-56 (replay/durability) lands, add fsync-fault injection or accept ADR 0004 `OutcomeUncertain` and test that code instead. ADR 0004 (`H101-54`) proposes replacing detail-substring branching — align rendering test when taxonomy changes.

### God question 4 — `-sender` vs `-caller`

**Criteria should assert unverified sender, not block the divergence.**

Kevin's separate `-sender` flag is **correct surface honesty** (H101-23). Add to item 1 walkthrough / UI-05 assertions when send is in the subprocess test: output must show sender as **claimed routing data**, never as verified identity; provenance fields visible; no validation badge. No change needed to accept `3659e6a`.

### God question 5 — no automatic retry

**Absence is sufficient for this slice.** No write command retries on failure — true by construction today. **Assert in item 3 contract suite** when built: failed mutation leaves stderr without a new auto-generated `-request-id` and without success line. Light assertion, not a blocker for these cards.

### Declared divergences

| Divergence | Ruling |
| --- | --- |
| Cursor = revision token only (no subscription) | **Accepted** — missing card for replay half of UI-06 |
| Session reuses `core/task` request records vs neutral `internal/api` | **Accepted with follow-up** — matches ADR direction; neutral package still open for item 4 UI allowlist |
| `-sender` separate from `-caller` | **Accepted** — product/security surface, not a defect |

### H101-57 amendment — `f242b2b` crash-reopen test (inform, 2026-09-19)

Reviewed `f242b2b` (`TestRun_Phase2CycleRecoversAfterKilledCLI`). Re-ran — pass on darwin (local); runs on linux CI when not `Unsupported`.

**Verdict: ACCEPTED WITH FOLLOW-UP** — advances item 2 on linux; does not discharge item 1.

#### `run()` vs spawned binary — one ruling for both items

Item 1's anchor says *subprocess, not in-process `host` import from the test package.* That forbids the **test package driving mutations through `host.Open`**, not every use of `run()`.

| Mechanism | What it proves | Enough for milestone discharge? |
| --- | --- | --- |
| `run()` in `package main` tests | Same dispatch as `main` for single-shot commands | **Scaffold only** for item 1; **acceptable** for per-invocation steps in item 2 |
| `exec.Command` subprocess (`hold` helper) | Real process, real `SIGKILL`, real lock lifetime | **Required** for the crash half of item 2 |
| `host.Open` / `GetSnapshot` from test package | Composition queries outside CLI | **Not** product walkthrough evidence — see H101-58 |

**Unified answer:** `run()` does **not** discharge item 1. It does **not** fully discharge item 2 either, but item 2's crash path is satisfied on **linux/amd64** when a **real subprocess** is killed and a subsequent command reopens with durable task/result state. Claudio's asymmetry (subprocess `hold`, `run()` for driving) is **enough for item 2's crash proof**; it is **not** enough to move item 1 on the same reasoning.

**Item 1 still needs:** a spawned-binary (or CI script invoking built `harnessing`) end-to-end §2 walkthrough — not `run()` alone.

#### Item 2 delta after `f242b2b`

| Part | linux/amd64 | darwin/arm64 |
| --- | --- | --- |
| Lock prerequisite (H101-20) | **Satisfied** (CI) | **Not discharged** (manifest or macOS runner) |
| Crash-reopen with product state | **Satisfied** — `hold` subprocess `SIGKILL`, `task` shows `AwaitingReview` + `res1` | Same test must run natively with recorded manifest |
| Acknowledged handoff survived | **Only via `host.GetSnapshot` in test** — not through CLI | Same gap |

**Item 2 overall: still PARTIALLY SATISFIED.** Task/result crash recovery on linux is proven; handoff durability is proven outside the product surface; darwin pending.

#### H101-58 — message query gap (god carded, not dispatched)

**Item 1 cannot honestly discharge while a walkthrough step is only verifiable from outside the product.** `send` and `ack` exist; no `message` (or snapshot) query on the CLI. The crash test acknowledges a handoff then proves it via `host.GetSnapshot` — valid engineering proof, **invalid product walkthrough proof**.

This is the third exit-proof gap (after reassignment, after long-running invocation). **H101-58 must land before item 1 discharges** if the §2 walkthrough includes handoff visibility. Item 2's handoff-survival claim should likewise move to CLI-visible evidence when H101-58 lands.

**No change to H101-57's other four answers** (UI-06 cursor, IOFailure render, `-sender`, no auto-retry).

---

## H101-60 — `f242b2b` formal ruling and H101-58 (2026-09-19)

**Context:** H101-57 (`b644535`) skipped `f242b2b` as uncommitted due to delivery timing; amendment `2026-09-19T10-03-30-865Z-26fa6f` covered it. This card makes the ruling explicit for the record.

### `f242b2b` — verdict

**ACCEPTED WITH FOLLOW-UP** (unchanged from amendment). Re-ran `TestRun_Phase2CycleRecoversAfterKilledCLI` — pass.

### Which parts of item 2 does `f242b2b` move?

Beyond the killed-`hold` fixture already credited in `3659e6a` / `hold_test.go`:

| Item 2 part | Moved by `f242b2b`? | Evidence |
| --- | --- | --- |
| Lock released after real CLI process death | **Yes** (extends `hold_test`) | Subprocess `hold` + `SIGKILL`; not empty-workspace fixture |
| **Busy while holder alive** (negative control) | **Yes** — new | Second open fails `Busy` during live `hold` |
| Durable **task + result** after crash-reopen | **Yes on linux/amd64** | `harnessing task` shows `AwaitingReview` + `res1` |
| Durable **acknowledged handoff** after crash-reopen | **Partial** | Proven via `host.GetSnapshot` only — not through CLI |
| Full item-1-scale cycle before crash | **No** | Missing blocker, reject, ≥2 agents |
| **darwin/arm64** native discharge | **No** | CI is ubuntu; manifest or macOS runner still required |
| **Windows** | **N/A** | Skips when `Unsupported` |

**Item 2 stays PARTIALLY SATISFIED.** `f242b2b` is the meaningful product-state crash-reopen proof on linux; it does not complete item 2.

### H101-58 — can item 1 discharge without a message query?

**No. H101-58 is exit-blocking for item 1.**

`definition.md` §2 requires the user to **inspect pending messages** and **see whether a message was recorded or acknowledged** (line 19). Item 1 anchors to that walkthrough. A step performed through `ack` but verified only via `host.GetSnapshot` in a test is **not** a product walkthrough — it is composition proof smuggled in through the test harness.

| | Ruling |
| --- | --- |
| Exit-blocking? | **Yes** for item 1 |
| Convenience? | **No** — §2 names acknowledgement visibility as minimum useful product |
| Item 2 impact | Handoff **survival after crash** can be credited partially without H101-58 (task/result via `task` command); handoff **visibility** in walkthrough cannot |
| H101-59 | Real `exec.Command` walkthrough must include CLI-visible ack check once H101-58 lands |

Claudio walking §2 under H101-59 will hit this wall on the handoff step — expected, not a surprise failure.

---

## H101-63 / H101-56 acceptance review (2026-09-21)

Reviewed commit `45cc821` (replay durability + `OutcomeUncertain` taxonomy). Re-ran `go test -count=1 ./internal/adapters/statestore/ -run 'Uncertain|Resolve'` — three new tests pass.

**Verdict: ACCEPTED WITH FOLLOW-UP**

### Phase 2 exit item deltas

**Nothing discharges directly** — this is store correctness, not a walkthrough or adapter-contract deliverable. **UI-02 assertability improves** at the domain/store layer (see below).

### `durableRevision` assumption — sceptical read

**Holds for every path in this commit.** Kevin's reasoning is sound under the project's stated invariants:

| Question | Finding |
| --- | --- |
| Sole writer under `mu` + flock? | **Yes.** All `state.json` writes go through `writeLocked` inside `Commit` while `mu` is held. Second `Open` in the same process fails `Busy` at flock — no second writer instance. |
| Writes outside `Commit`? | **None** in production code. Lock-file PID text is diagnostic (`WriteAt`, never read for correctness). Manual file tampering is corrupt-read / `IOFailure`, not a silent un-sync. |
| Can a confirmed revision be invalidated later in-process? | **Not today.** `durableRevision` only increases on successful directory fsync after apply or `confirmDurable`. No code path truncates or rewrites state without going through the same barrier logic. |
| Fresh `Open` starts at zero? | **Correct.** Prior process's unconfirmed fsync is not this instance's assumption to make. |

**Residual risk (documented, not a reject):** a **future** code path that writes `state.json` without updating `durableRevision` would break the optimization — same class of bug as bypassing `Commit`. The struct comment states the invariant; code review must preserve it.

### UI-02 / H101-53 — can contract stop asserting on detail substring?

**Yes, for `OutcomeUncertain` at the store layer.** Tests assert `Code`, `Effect`, `Confirmation`, `RequestID`, `WorkspaceID`, `ObservedRevision` — and deliberately mutate `Detail` to prove independence.

**Follow-up (not blocking this card):** CLI `printCommandError` still keys the UNCERTAIN second line off `IOFailure`, not `OutcomeUncertain` (`flags.go`). Host/engine must propagate the new code before adapter-contract UI-02 assertions cover the full CLI path. Update H101-53 contract table when that lands.

### `ResolveRequest` — `NotFound` for cross-caller vs Absent

**Sound.** ADR 0004 requires authorization before lookup; another principal's receipt must not be exposed. Mapping both **absent** and **not yours** to `NotFound` is deliberate data minimization — same pattern as not leaking existence. A caller who owns the request uses their own `callerAgentID`; they never need to distinguish "wrong principal" from "no record" at the store layer.

**Does not lose a required caller distinction** for the designed API: wrong-principal **commands** still return `Denied` at the engine; this is only the **resolution lookup** surface.

### `Unknown` + `Outcome` — ADR prose only

**Domain fields already exist** (`EffectUnknown`, `ConfirmationOutcome`). When a producer implements lost-response uncertainty, contract assertions should use those fields — **not** detail substrings.

**Strengthens H101-62 / Stanley's case:** item 3 UI-02 should require `Effect`/`Confirmation` assertions for the transport boundary case once implemented. Not required to accept H101-56; flag for taxonomy follow-up.

### Follow-up (non-blocking)

- Propagate `OutcomeUncertain` through host → CLI; retire IOFailure-based UNCERTAIN rendering when migrated.
- Put `ResolveRequest` on the port when Stanley rules (already deferred).
- Engine/host migration tests per ADR 0004 consequences section.

---

## H101-65 / H101-58 acceptance review (2026-09-21)

Reviewed commit `737d453` (`harnessing message <id>`, walkthrough unskipped). Re-ran `go test -count=1 ./cmd/harnessing/ -run TestCLI_Phase2ProductWalkthroughSubprocess` — pass (1.48s, real `go build` binary, every step `exec.Command`).

**Verdict: ACCEPTED**

**H101-58 is no longer exit-blocking.** Acknowledged handoff is now visible through `harnessing message` on the product surface, not only via `host.GetSnapshot` in tests.

### Item 1 — does it discharge?

**Yes on linux/amd64** — the subprocess walkthrough criterion that item 1 anchors to is met where CI runs the test.

**Not flat across both ADR 0001 targets.** Per H101-55, **darwin/arm64** still needs native milestone evidence (manifest or macOS runner) before quoting item 1 as fully discharged on every supported platform. Same class of caveat as item 2.

**Item 1 overall: PARTIALLY SATISFIED** until darwin native proof is recorded.

### What the walkthrough covers (item 1 checklist)

| Item 1 requirement | Covered? | Evidence |
| --- | --- | --- |
| Subprocess `harnessing` only | **Yes** | `buildAndRunCLI` / `exec.Command`; no test-package `host` import |
| Create/open workspace | **Yes** | `-workspace` on every invocation; implicit open |
| ≥2 named agents | **Yes** | `engineer`, `analyst` registered |
| Task with accountable owner | **Yes** | `t1` assignee `engineer` |
| Task-linked handoff + ack | **Yes** | `send` + `ack`; `message m1` asserts ack + four facts |
| Blocker report and resolve | **Yes** | `Blocked` → `Doing` |
| Report, human reject, accept replacement | **Yes** | `res1` rejected, `res2` accepted; `Denied:` prefix asserted |
| Status queries (assigned / in-progress / blocked / awaiting-review) | **Yes** | `status()` asserts Todo, Doing, Blocked, AwaitingReview, Done |
| Reporter identity on status | **No** | `task` shows Assignee + `Created by` provenance only — not who reported Blocked/AwaitingReview |
| Close CLI; reopen without resending | **Yes** | Each command is an independent process; final `task` in fresh invocation |
| Durable records | **Yes** | `ResultID: res2` after reopen |

### §2 pass (god Q1) — reachable steps Claudio may have written off

Compared to [`definition.md`](../product/definition.md) §2, not only the test's original shape:

| §2 element | In walkthrough? | Notes |
| --- | --- | --- |
| Complete assign → handoff → blocker → review cycle | **Yes** | Core cycle |
| See whether message was recorded or acknowledged | **Yes** | `message` after ack |
| Inspect **pending** messages (plural / before ack) | **No** | No list command; no pre-ack query |
| One status view with reporter + last update per state | **Partial** | Repeated `task`, not aggregate view; reporter/last-update gaps |
| See how to connect existing sessions | **No** | Onboarding/docs — not in item 1 checklist |
| Restart of the interface | **Yes** | Per-invocation close; not SIGKILL (item 2) |
| Named external agent tool | **N/A** | Explicitly not Phase 2 exit |

**Follow-up (non-blocking for item 1 discharge):** pending-message inspection and per-status reporter/last-update on `task` output align §2 prose with what item 1 already names ("reporter identity").

### God Q2 — `(absent)` for missing timestamps

**Sufficient for UI-05.** A visible `(absent)` token states the fact has not been observed — it cannot be mistaken for a zero time that implies occurrence. Stronger prose ("has not occurred") is optional polish, not a discharge requirement.

### God Q3 — unverified-sender prose is load-bearing

**Valid concern; follow-up recommended, not blocking.**

The test asserts a long parenthetical on the `Sender:` line. The `Recorded by:` line already exposes `IdentityVerification` as a separate field (`identity unverified`). **Prefer asserting on that stable field** (same pattern as H101-63 UI-02 field assertions) rather than the `Sender:` parenthetical alone. Rewording the human label should not break the contract.

### God Q4 — UI-02 at CLI surface

**Not closed.** H101-64 must propagate `OutcomeUncertain` through host → CLI before UI-02 is credited at the product surface. This commit does not change that.

### Item 2 delta from H101-58 landing

Handoff **visibility** in a product walkthrough is now CLI-provable. Item 2's **crash** test (`f242b2b`) still proves acknowledged-handoff survival via `host.GetSnapshot` — update that test to use `harnessing message` when convenient; not required to accept H101-58.

### Nine exit items — owner-ready standing (2026-09-21)

| # | Standing |
| --- | --- |
| 1 | **Partially satisfied** — linux/amd64 subprocess §2 walkthrough passes in CI; darwin/arm64 native evidence pending (H101-55); minor §2 follow-ups |
| 2 | **Partially satisfied** — linux crash-reopen with task/result state; handoff CLI-visible but crash proof still host-side for ack; darwin pending |
| 3 | **Not satisfied** — no `adaptercontract` suite + second adapter; UI-06 subscription gap (H101-61) |
| 4 | **Partially satisfied** — store-import yes; UI import allowlist not enforced |
| 5 | **Not satisfied** — mailbox H101-22 not wired to delivery-fact recorders |
| 6 | **Satisfied** — first-run disclosure (`1eda92b`) |
| 7 | **Not satisfied** — Phase 3 ops `Unsupported` surface |
| 8 | **Satisfied** — zero network on CLI tree |
| 9 | **N/A** — test layering policy |

---

## H101-67 — amendment to H101-65 item 1 discharge (2026-09-21)

**Context:** God asked Claudio to re-read [`definition.md`](../product/definition.md) §2 after H101-58 landed. Claudio reports two gaps I ruled non-blocking in H101-65 are **required by §2**. God asks whether item 1's linux/amd64 discharge still stands.

**Pick: option 2 — they are required; item 1 un-discharges until they exist.**

### Ruling

**Item 1's linux/amd64 discharge is retracted.** H101-65 was wrong to call pending-message inspection and reporter/last-update **non-blocking follow-up**. They are the same shape as H101-58: §2 names a capability item 1 anchors to; the walkthrough cannot exercise it through the shipped `harnessing` command; passing the test while skipping the step is passing over a hole.

| Gap | §2 / item 1 source | CLI today | Same class as H101-58? |
| --- | --- | --- | --- |
| Inspect **pending** messages | §2 line 19 | No list or pre-ack pending query; `message <id>` only after ack | **Yes** |
| Reporter + last update per status | §2 line 21; item 1 line 79 ("with reporter identity") | `task` shows Assignee + `Created by` only — not who reported Blocked/AwaitingReview or when | **Yes** |
| Connect existing sessions | §2 line 19 | Documentation/onboarding only — no product command expected | **No** — Claudio agrees; not exit-blocking |

**Nothing distinguishes the first two from H101-58** except that acknowledgement visibility is now fixed. The remaining gaps are structural the same way: product surface cannot take the §2 step.

**Item 1 standing: NOT SATISFIED on any platform** (including linux/amd64). Subprocess walkthrough progress is real and credited; **discharge waits on H101-68** plus walkthrough assertions, then darwin native evidence per H101-55.

H101-58 acceptance (`737d453`) stands — message query is no longer the blocker; these two are.

### Informational — landed since H101-65 (no discharge impact)

| Change | Commit | Effect on prior gaps |
| --- | --- | --- |
| `OutcomeUncertain` reaches CLI | `94105b8` | Closes H101-65 Q4 / H101-63 follow-up — CLI branches on code then `Effect`/`Confirmation` policy table; `Detail` not read for UNCERTAIN line |
| `ResolveRequest` bound on `FrontendSession` | (same card) | Stanley's caller-binding qualification is now behaviour-tested — two sessions, same request ID, different answers; not an item-1/3 discharge by itself |

### Nine exit items — owner-ready standing (amended 2026-09-21)

| # | Standing |
| --- | --- |
| 1 | **Not satisfied** — subprocess §2 walkthrough proves most of cycle; exit-blocked on pending-message inspection + reporter/last-update (H101-68); darwin pending after that |
| 2 | **Partially satisfied** — linux crash-reopen with task/result state; handoff CLI-visible; crash ack proof still host-side; darwin pending |
| 3 | **Not satisfied** — no `adaptercontract` suite + second adapter; UI-06 subscription gap (H101-61) |
| 4 | **Partially satisfied** — store-import yes; UI import allowlist not enforced |
| 5 | **Not satisfied** — mailbox H101-22 not wired to delivery-fact recorders |
| 6 | **Satisfied** — first-run disclosure (`1eda92b`) |
| 7 | **Not satisfied** — Phase 3 ops `Unsupported` surface |
| 8 | **Satisfied** — zero network on CLI tree |
| 9 | **N/A** — test layering policy |

---

## H101-69 / H101-68 acceptance review (2026-09-21)

Reviewed commit `3d118cc` (`harnessing messages`, `task` reporter/last-update, walkthrough assertions). Re-ran `go test -count=1 ./cmd/harnessing/ -run TestCLI_Phase2ProductWalkthroughSubprocess` — pass (1.27s).

**Verdict: ACCEPTED WITH FOLLOW-UP**

H101-68 closes the **two surface gaps** that blocked item 1 in H101-67. It reveals a **fourth gap at domain depth** that still blocks discharge.

### Item 1 — does linux/amd64 discharge now?

**No. Still NOT SATISFIED on any platform** (including linux/amd64). Do not report item 1 discharged.

### What H101-68 fixed (the two retracted gaps)

| Gap (H101-67) | Fixed? | Evidence |
| --- | --- | --- |
| Inspect pending messages | **Yes** | `harnessing messages` via `GetSnapshot`; walkthrough asserts pending count 1 before ack, 0 after |
| Reporter + last update | **Partial** | `task` prints `Reporter:` / `Last update:`; walkthrough asserts on final `Done` state with `identity unverified` |

### The fourth gap — transition provenance (domain, not CLI)

For **Doing** or **Blocked** with no current result, `task` prints:

```text
Reporter:   (unavailable; status-update provenance is not recorded)
Last update: (unavailable; status-update timestamp is not recorded)
```

Checked `internal/core/task/engine.go`: `TransitionTask` records the transition but **does not persist caller actor or timestamp** on the task record. Claudio's explicit unavailable rendering is the right call — not substituting creator or task time — but it means the product **cannot answer §2's question** for in-progress and blocked states.

### Does §2 require reporter/last-update for every status, or only where a result exists?

**Every status category §2 names — not only where a result exists.**

[`definition.md`](../product/definition.md) §2 line 21 lists four things one status view must answer: assigned, **reported in progress**, **blocked**, and result ready for review. The next sentence: **"Every status identifies its reporter and last update."** That applies to all four categories, not only `AwaitingReview`.

Item 1 line 79 repeats this: status queries must show assigned / in-progress / blocked / awaiting-review **with reporter identity**. The walkthrough exercises `Blocked` and `Doing` but `status()` only asserts the `Status:` line — it never asserts reporter during those states because the data does not exist.

| Status | Reporter/last-update today | Satisfies §2? |
| --- | --- | --- |
| Todo (assigned) | Creator provenance | **Yes** |
| Doing (in progress) | Unavailable | **No** |
| Blocked | Unavailable | **No** |
| AwaitingReview / Done | Result provenance | **Yes** |

**Same shape as H101-58**, one layer deeper: honest "(unavailable)" is not discharge — §2 requires the identity and time, not a disclaimer that they were not recorded.

**Exit-blocking work:** persist transition actor + timestamp in the engine/domain (records change — route design to Stanley per `boundaries.md`; implement via Kevin). Then expose on `task`, assert in walkthrough for at least one `Doing` and one `Blocked` step.

H101-68 acceptance stands for what it delivered; it does not discharge item 1.

### God Q1 — `messages` via snapshot vs UI-06

**Satisfies the enumeration/discovery half of UI-06 for messages; does not discharge UI-06 or item 3.**

| UI-06 part | `messages` via `GetSnapshot` | H101-61 (Kevin) |
| --- | --- | --- |
| Discover records without prior entity IDs | **Yes** for pending messages — recipient need not know message ID | N/A |
| Complete detached snapshot | Uses `GetSnapshot` (same port UI-06 names) | — |
| Observe from cursor without losing commits | **No** — no subscription/replay | Subscription + replay |
| Deduplicate replays / `CursorExpired` reload | **No** | H101-61 scope |

**They meet in the middle, not as duplicates:** snapshot enumeration is the right CLI route for "inspect pending messages" (item 1 / §2); subscription/replay is still required before item 3 can credit UI-06 fully. No seam that blocks item 1 once transition provenance lands.

### God Q2 — darwin (H101-55)

Unchanged. Even after transition provenance, **linux/amd64 discharge is separate from darwin/arm64** — native manifest or macOS runner required before quoting item 1 flat across both targets.

### Nine exit items — owner-ready standing (2026-09-21, post-H101-68)

| # | Standing |
| --- | --- |
| 1 | **Not satisfied** — walkthrough passes; surface §2 gaps closed (H101-68); **exit-blocked on transition provenance** (domain); darwin pending after that |
| 2 | **Partially satisfied** — linux crash-reopen with task/result state; handoff CLI-visible; crash ack proof still host-side; darwin pending |
| 3 | **Not satisfied** — no `adaptercontract` suite + second adapter; UI-06 subscription/replay gap (H101-61) |
| 4 | **Partially satisfied** — store-import yes; UI import allowlist not enforced |
| 5 | **Not satisfied** — mailbox H101-22 not wired to delivery-fact recorders |
| 6 | **Satisfied** — first-run disclosure (`1eda92b`) |
| 7 | **Not satisfied** — Phase 3 ops `Unsupported` surface |
| 8 | **Satisfied** — zero network on CLI tree |
| 9 | **N/A** — test layering policy |

---

## H101-70 inform — UI-06 assessment correction (2026-09-21)

**Context:** God informs Stanley's ruling `2e72fd7` contests H101-69's UI-06 assessment. Kevin's subscription/replay (`49d9d84`) passes its tests, but Stanley found two source-level gaps: (1) fresh event bus accepts stale cursor after restart instead of `CursorExpired`; (2) store commit and event publish are not serialized — R+1 can publish before R and R is dropped by high-water dedup. Same-host multi-session observation is sufficient (no durable cross-process log). H101-71 with Kevin for status record (item 1), port-comment fix, and stream gaps.

### God question — must item 3 / UI-06 assertions name restart and concurrent-commit explicitly?

**Yes.** Same class as substring fragility and H101-58-style holes: a check that is true about what it measures and silent about what it does not.

| Gap | What passing tests covered | What they did not |
| --- | --- | --- |
| Stale cursor after restart | Buffer eviction → `CursorExpired` | Fresh `EventBus` with no replay floor accepts old numeric cursor against empty buffer |
| Publication ordering | Publish vs subscribe race | Goroutine interleaving: `Commit` returns then `publish` — R+1 can publish before R, R dropped by `<= lastPublished` |

Without named scenarios in the H101-53 assertion table (updated above), a future implementation can pass the same suite and retain the same holes. **Item 3 cannot credit UI-06 until the contract suite exercises both cases** — per ADR 0003 §stream semantics as amended in H101-70.

### H101-69 correction

H101-69 said enumeration-via-snapshot satisfies UI-06 discovery half and subscription/replay completes the remainder. **Revised:** discovery half stands for item 1 (`messages`); **UI-06 as a whole is not proven** by `49d9d84` until restart-cursor and commit/publication-order obligations are fixed and contract-tested. Item 3 unchanged: **not satisfied**.

### Informational (no Kelly action on code)

| Stanley ruling (`2e72fd7`) | Effect |
| --- | --- |
| `Task.LastStatusChange` — latest transition provenance | Unblocks item 1 domain gap (H101-69); H101-71 with Kevin |
| Replay returns original records; comments corrected | Consumer must not treat replay data as new publication — contract tests must assert distinction |
| Same-host multi-session observation sufficient | Closes architecture ceiling Kevin flagged; does not prove current implementation |

H101-55 darwin manifest: still open, unchanged.

---

## H101-73 / H101-43 acceptance review (2026-09-21)

Reviewed commit `306c6fa` (presentation import gate). Re-ran `./scripts/check-import-allowlist.sh` — tree exit 0; `./testdata/forbidden` exit 1 naming statestore, ports, and core/task violations.

**Verdict: ACCEPTED WITH FOLLOW-UP**

### Item 4 UI half — does it discharge?

**No. UI half stays open.** Pick **reading 2** (the exception is the hole item 4 names), with the qualification structure from reading 3.

| Layer | Status |
| --- | --- |
| **Checker mechanism** | **Satisfied** — production packages importing `host` or `core/task` are rejected unless allowlisted; forbidden fixture fails with exit **1**; gate is CI-enforced |
| **Containment at the production CLI** | **Not satisfied** — `cmd/harnessing` is the interim presentation adapter and remains allowlisted; package-level check cannot distinguish `Capabilities` from `FrontendSession` inside that package |

Item 4 asks presentation packages not reach the core around the session boundary. **The one package where that violation would occur today is exactly the exception.** Enforcing everywhere else is necessary and real progress; it does not discharge the half.

**Item 4 overall: PARTIALLY SATISFIED** (unchanged label, updated substance — store half yes; UI gate yes; CLI exception no).

### God Q1 — is a declared-and-documented limit enough for containment?

**Necessary, not sufficient for discharge.**

Inline reason in `import-allowlist.txt` (H101-33 lesson applied) makes the exception **auditable** — good, required. For **containment discharge**, documentation does not substitute for the boundary holding where presentation meets host. A documented hole is a tracked migration debt, not a satisfied criterion.

### God Q2 — auditable exception

**Confirmed.** Reason inline where the allowlist reader already looks — H101-33 correction applied without being asked.

### Follow-up — card CLI `FrontendSession` migration

**Yes. Card it.** `cmd/harnessing` must migrate from direct `host`/`Capabilities` to `FrontendSession`, then remove the allowlist exception.

**Not a rename.** `FrontendSession` deliberately exposes no `Close` and no mailbox delivery facts — composition/assembly must own workspace lifecycle and any paths the session surface omits. Whoever migrates must decide where lifecycle lives before promising the card.

### H101-55 (prerequisite noted)

Darwin/arm64 native manifest still open. No impact on item 4 ruling.

### H101-72 (prerequisite noted)

Already answered (`503fb21` / H101-70 inform). UI-06 assertions now name restart and interleaving cases.

### Nine exit items — item 4 line for owner

| # | Standing |
| --- | --- |
| 4 | **Partially satisfied** — store-import yes; UI import gate enforced except interim `cmd/harnessing` exception pending `FrontendSession` migration |

---

## H101-55 reaffirmation (2026-09-21, prerequisite to H101-76)

**Unchanged.** Darwin/arm64 native milestone evidence still not recorded — no macOS CI runner, no written manifest with commit, exact macOS build, commands, and outputs. Undisclosed laptop runs do not discharge.

Linux/amd64 CI evidence stands for tests that run there. **Item 1 linux discharge is separate from darwin** — even when item 1 clears on linux, quote platform scope explicitly per H101-55.

---

## H101-76 / H101-71 acceptance review (2026-09-21)

Reviewed commit `3a4d320` (`LastStatusChange`, restart cursor floor, publish-in-commit-order). Re-ran named tests with `-race` — pass. Walkthrough subprocess test — pass.

**Verdict: ACCEPTED WITH FOLLOW-UP**

### Item 1 — does it discharge?

**No. Still NOT SATISFIED on any platform** (including linux/amd64).

| Layer | Status |
| --- | --- |
| Domain `LastStatusChange` | **Satisfied** — five real transition points including reject/reopen; replay and denied attempts do not advance it (`TestLastStatusChange_ReplayAndDenialDoNotAdvanceIt`) |
| Composition detachment | **Satisfied** — `TestCapabilities_GetTask_LastStatusChangeIsDetached` through `host.cloneTask` (correct test placement) |
| **CLI product surface** | **Not satisfied** — `cmd/harnessing/task.go` still prints `(unavailable; status-update provenance is not recorded)` for Doing/Blocked; does not read `LastStatusChange` |
| Walkthrough | **Not satisfied** — `status()` asserts `Status:` only; never asserts reporter during Blocked/Doing |

Same shape as H101-58: domain fix without product walkthrough proof does not discharge item 1. **Follow-up:** wire `LastStatusChange` into `printTask`; assert reporter + last-update (with `identity unverified`) on at least one Blocked step in `TestCLI_Phase2ProductWalkthroughSubprocess`.

**Darwin (H101-55):** unchanged — linux progress does not flatten platform scope.

### Item 3 — what moves?

| UI-06 engine scenario (H101-70) | Status |
| --- | --- |
| Stale cursor after restart → `CursorExpired` | **Satisfied** — `TestSubscribe_RestartCursorIsExpired`; overcorrection guarded by `TestSubscribe_RestartThenFreshSnapshotCursorWorks` |
| Concurrent commits publish in order | **Satisfied** — publication inside commit critical section; `TestEngine_ConcurrentCommitsPublishInOrder` (50 goroutines, `-race`) |

**Item 3 overall: still NOT SATISFIED.** Engine implementation gaps for the two named scenarios are closed; **adaptercontract suite + second adapter remain required** for discharge. Item 3 moves from "UI-06 not proven at engine" to "UI-06 engine obligations met; contract suite outstanding."

Kevin's corrected invariant (verbatim for board): *publish and subscribe share one mutex's critical section AND all commits publish in commit order, so no committed event can be missed by a subscriber whose cursor is at or after that event's predecessor.*

### Multi-event commit checkpointing — acceptable disposition?

**Yes, for Phase 2 — with a recorded future obligation.**

Kevin's finding is real at the type level (`Mutate` may return multiple events) but **unreachable through any current engine command** (every closure returns exactly one event, confirmed by inspection). Disposition:

| | Ruling |
| --- | --- |
| Fix required now? | **No** — no path to reach the failure |
| Proof required now? | **Yes** — `TestEventBus_MultiEventCommitIsDeliveredWhole` documents the property holds by construction (whole `events` slice under one publish lock) |
| Future obligation | If a command ever returns multiple events per commit, **whole-commit delivery must be demonstrated** before crediting that path — criterion carries the scenario without requiring an unverifiable fix today |

Same principle god applied to Kevin: a criterion naming a scenario nobody has observed failing must not gain a fix nobody can verify. **Held-by-construction + documented invariant is acceptable**; item 3 does not need the path made reachable solely to prove it.

### gofmt CI step (god finding)

**Recommended follow-up, not exit-blocking.** Same shape as zero-network before it became CI: agents run locally, habit not guard. Worth a lightweight `gofmt -l` (or `gofmt -s -w` check) CI step — **not** a Phase 2 exit item unless owner wants hygiene gates enumerated.

### Nine exit items — owner-ready standing (2026-09-21)

| # | Standing |
| --- | --- |
| 1 | **Not satisfied** — domain `LastStatusChange` yes; CLI/walkthrough reporter on Doing/Blocked still open; darwin pending |
| 2 | **Partially satisfied** — linux crash-reopen; darwin pending |
| 3 | **Not satisfied** — UI-06 engine scenarios met; adaptercontract + second adapter still needed |
| 4 | **Partially satisfied** — UI gate except `cmd/harnessing` exception |
| 5 | **Not satisfied** — mailbox H101-22 |
| 6 | **Satisfied** |
| 7 | **Not satisfied** |
| 8 | **Satisfied** |
| 9 | **N/A** |

---

## H101-55 reaffirmation (2026-09-21, prerequisite to H101-79)

**Unchanged.** No darwin/arm64 native milestone manifest recorded. Linux/amd64 CI discharge for item 1 is **separate** from darwin — do not quote item 1 flat across both targets.

---

## H101-79 acceptance review (2026-09-21)

Reviewed `8ba1112` (concurrent-commit seam proof) and `9a6b464` (`internal/api` migration + `LastStatusChange` on CLI). Re-ran import gate (tree 0, forbidden 1), subprocess walkthrough with `-race` (pass), `TestCommitMu_BlocksSecondCommitInTheNamedGap` (pass).

**Verdict: ACCEPTED** (both commits)

### Item 4 UI half — discharged?

**Yes. Item 4 is SATISFIED** (both halves).

The `cmd/harnessing` exception **was** the hole (H101-73). It is gone. `cmd/harnessing` imports only `api` and `assembly` — verified no `host` or `core/task` import.

**Assembly-only `host` import is different in kind, not the same objection moved one package over.**

| | `cmd/harnessing` exception (rejected) | `internal/assembly` exception (accepted) |
| --- | --- | --- |
| Role | Presentation acting as composition | **Trusted composition root** — ADR's intended sole `host` importer |
| What it held | `Capabilities` directly | Open/Close/lifecycle + `FrontendSession` binding |
| Could a careless UI bypass the session? | **Yes** — same package | **No** — presentation reaches only `api.FrontendSession` |

Item 4 text: presentation must not import `host`; only assembly imports `host`. That is what the gate now enforces. The allowlist entry documents the **architectural role**, not a workaround for a presentation package.

**Follow-up (non-blocking):** a second adapter will need its own assembly entry on the allowlist — not a regression, an expected extension.

### Item 1 — discharged?

**Yes on linux/amd64.** **Partially satisfied overall** (darwin pending per H101-55).

| Check | Status |
| --- | --- |
| Subprocess §2 walkthrough | **Pass** — `TestCLI_Phase2ProductWalkthroughSubprocess` with `-race` |
| Pending messages | **Yes** — `messages` before/after ack |
| Reporter on Doing/Blocked | **Yes** — `printTask` uses `LastStatusChange`; `status()` asserts unverified reporter + last update on every status including Blocked |
| `FrontendSession` surface | **Yes** — commands use `withSession` / `api.FrontendSession` via `assembly` |

Do not report item 1 discharged flat across darwin/arm64 until native manifest or macOS runner.

### Item 3 — second named scenario discharged?

**Yes — the concurrent publish-ordering scenario is discharged** at the engine/implementation level.

| Proof | Test |
| --- | --- |
| `commitMu` blocks second commit in the named gap | `TestCommitMu_BlocksSecondCommitInTheNamedGap` — deterministic seam (CAS, not `sync.Once`; Kevin caught the Once false positive) |
| Why ordering matters | `TestEventBus_OutOfOrderPublishDropsTheEarlierRevision` — out-of-order publish drops R |
| Stress supplement | `TestEngine_ConcurrentCommitsPublishInOrder` — 50 goroutines under `-race` |

**Item 3 overall: still NOT SATISFIED** — adaptercontract suite + second adapter outstanding. Next implementer inherits a **met** criterion for this scenario, not merely a named one.

### Informational

- `check-gofmt.sh` now in CI — closes H101-76 gofmt follow-up recommendation.
- God's sequencing note (Kevin amendment held during Claudio migration): acknowledged; separate commits verified.

### Nine exit items — owner-ready standing (2026-09-21)

| # | Standing |
| --- | --- |
| 1 | **Partially satisfied** — **linux/amd64 yes** (subprocess §2 walkthrough); darwin pending |
| 2 | **Partially satisfied** — linux crash-reopen; darwin pending |
| 3 | **Not satisfied** — UI-06 engine scenarios met; adaptercontract + second adapter needed |
| 4 | **Satisfied** |
| 5 | **Not satisfied** — mailbox H101-22 |
| 6 | **Satisfied** |
| 7 | **Not satisfied** |
| 8 | **Satisfied** |
| 9 | **N/A** |

---

## H101-55 reaffirmation (2026-09-21, prerequisite to H101-81)

**Unchanged.** Darwin/arm64 native manifest still not recorded. No impact on item 7.

---

## H101-81 / H101-80 acceptance review (2026-09-21)

Reviewed commit `20e0caa` (Phase 3 ops on `api.FrontendSession`). Re-ran `TestFrontendSession_Phase3OperationsAreUnsupported` — pass (named subtest per operation).

**Verdict: ACCEPTED**

### Item 7 — discharged?

**Yes. Item 7 is SATISFIED.**

| Item 7 requirement | Status |
| --- | --- |
| `StartRun`, `StopRun`, `SetRunBudget` named in source | **Yes** — `boundaries.md` line 19 command inventory |
| Return stable `Unsupported`, not panic/silent success | **Yes** — `domain.ErrUnsupported` + detail naming operation and Phase 2 |
| CI assertion | **Yes** — `TestFrontendSession_Phase3OperationsAreUnsupported` (one subtest per operation) |
| CLI subcommands | **Not registered** — allowed per item 7: *"If Phase 2 ships without registering those subcommands, CI asserts they are unreachable as success paths"* |

### `boundaries.md` completeness claim

**Confirmed for the command inventory.** `boundaries.md` line 19 lists exactly three Phase 3 **Execute** commands: `StartRun`, `StopRun`, `SetRunBudget`. Run-output and process-control **surfaces** (states, events, `GetRun`, streams) are descriptive/query/event paths — not separate Phase 3 commands Claudio omitted. Claudio's cold read is accurate.

### Item 3 scope — did it enlarge?

**Yes, concretely — but correctly, not newly invented.**

| | Before H101-80 | After H101-80 |
| --- | --- | --- |
| UI-08 / item 7 obligation | Phase 3 ops must return consistent `Unsupported` | Same — now **three named methods** on `FrontendSession` |
| Adaptercontract suite | Must assert UI-08 | Must call **`StartRun`, `StopRun`, `SetRunBudget`** and assert `Unsupported` on **both** adapters |
| Throwaway adapter | Implements `FrontendSession` | Must include three stubs (or delegate to host binding) |

**This is item 3 getting more correct, not bigger by accident.** ADR 0003 UI-08 already required advertising unsupported operations consistently. H101-80 makes the contract surface explicit so the suite cannot pass while leaving Phase 3 behaviour undefined.

**Dispatch guidance:** include a **UI-08 Phase 3 subsection** in `adaptercontract` with all three operations named — same principle as H101-70's named restart/interleaving scenarios.

Item 3 overall: **still NOT SATISFIED** (suite + second adapter outstanding). Scope for the suite card is now **+3 named assertions**, not a new exit item.

### Claudio's `api` change — architect routing needed?

**No.** Putting Phase 3 stubs on `api.FrontendSession` is within engineer authority:

- `boundaries.md` already names the three commands
- Presentation depends on neutral `api`; host owns Phase 2 disposition
- Stanley's role split permits adding methods to the presentation port when boundaries names them

Not a stealth architecture change — implementation of an existing boundary. **No Stanley reroute required.**

### Nine exit items — owner-ready standing (2026-09-21, post-H101-80)

| # | Standing |
| --- | --- |
| 1 | **Partially satisfied** — linux/amd64 yes; darwin pending |
| 2 | **Partially satisfied** |
| 3 | **Not satisfied** — suite must add UI-08 Phase 3 named assertions; second adapter needed |
| 4 | **Satisfied** |
| 5 | **Not satisfied** |
| 6 | **Satisfied** |
| 7 | **Satisfied** |
| 8 | **Satisfied** |
| 9 | **N/A** |

---

## H101-83 acceptance review (2026-09-21)

Reviewed `7247dc3` (Kevin: `internal/adapters/mailbox`) and `7dc7809` (Claudio: `internal/throwawayadapter`). Re-ran mailbox tests (15) and throwawayadapter tests — pass. Did not re-run full `go test ./...` (god verified).

**Verdict: ACCEPTED WITH FOLLOW-UP** (both commits)

### 1. Item 5 — SATISFIED or not?

**NOT SATISFIED.**

| Item 5 requirement | Status |
| --- | --- |
| File mailbox adapter | **Yes** — `FileMailbox` implements `ports.Mailbox` (`ScanInbox`, `Publish`, `Archive`) |
| Sole writer of `RecordMessagePublished` / `RecordMessageProcessed` | **Partial** — `Deliverer` is the designed sole driver; **host still exposes both methods on `Capabilities`**; **not wired in `assembly`** — product path does not invoke `Deliverer` yet |
| Publish precedes process; reverse order rejected | **Yes** — `Deliverer` enforces from snapshot facts; `TestDeliverer_IngestPending_RefusesProcessedBeforePublished` |
| `MessageAcknowledgement` + `AcknowledgeMessage` same rules | **Yes** — `IngestAcks` calls `AcknowledgeMessage`; file ack round-trip tested |
| **Full send → publish → process → ack through adapter** | **No** — deliverer tests use `fakeRecorder` with pre-seeded messages; **no test chains `SendMessage` (engine) → `DeliverPending` → `IngestPending` → ack through real engine + mailbox** |

Follow-up before discharge: wire `Deliverer` in assembly/host; one integration test through real engine covering the full path.

### 2. Item 3 — how much discharged, what remains?

**Moved materially; still NOT SATISFIED.**

| Piece | Before | After `7dc7809` |
| --- | --- | --- |
| Second adapter package | **Missing** | **`internal/throwawayadapter`** — JSON driver over `api.FrontendSession` only; import gate clean |
| `adaptercontract` suite | Missing | **Still missing** — no `internal/adaptercontract/` |
| UI-01–08 on both adapters | No | **No** — throwawayadapter has fake-session unit tests only (expected gap until suite) |
| Cross-adapter continuation | No | No |
| Concurrent observer | No | No |

**Rough standing:** prerequisite **second adapter exists** (~one third of item 3's structural work). **~0% of the contract suite discharged.** Dispatch the suite card with both adapters named: CLI (`cmd/harnessing` via `assembly`) and `throwawayadapter`.

### 3. Divergence A — `domain.Envelope.TaskID` vs boundaries.md

**Match.** `boundaries.md` line 34: envelope carries "optional task ID" among version, workspace ID, message ID, sender/recipient IDs, kind, body, creation time, optional reply-to ID. `domain.Envelope` has all nine fields with `TaskID *TaskID` and `ReplyToMessageID *MessageID`. **Spec gap closed.**

### 4. Divergence B — `MessageAcknowledgement` field list

**Exact match.** `boundaries.md` line 59: schema version, workspace ID, unique control-record ID, original message ID, recipient agent ID. `domain.MessageAcknowledgement`: `SchemaVersion`, `WorkspaceID`, `ControlRecordID`, `OriginalMessageID`, `RecipientAgentID`. **No extra or missing fields.**

### 5. Divergence D — `ScanInbox` cursor ignored; port doc tolerance?

**Not a defect.** Read `internal/core/ports/outbound.go` `Mailbox` interface — **no doc comment** on cursor or duplicate-scan tolerance. **`boundaries.md` port 6 (line 49)** says: *"Scan can repeat entries"*; line 57: *"Duplicate scans are harmless through persisted deduplication."* Kevin's `FileMailbox` comment cites boundaries correctly. Port doc is thin; **boundaries is authoritative**. Ignoring cursor while returning full inbox is **consistent with spec**; optimization deferred is acceptable.

### 6. Divergence F — named test per Denied and NotFound?

**Partial — Denied yes, NotFound no.**

| Disposition | Named test? |
| --- | --- |
| `Denied` on ack ingest | **Yes** — `TestDeliverer_IngestAcks_GoesThroughAcknowledgeMessageAndQuarantinesDenied` |
| `NotFound` on ack ingest | **No** — `IngestAcks` handles `ErrNotFound` in code (line 161) but **no named subtest** |

Same class as item 7 naming rule. **Follow-up:** add `TestDeliverer_IngestAcks_QuarantinesNotFound` (or subtest) before treating ack quarantine coverage complete.

### Divergence C (Stanley's) — blocks item 5?

**Does not block this verdict.** `WriteAck`/`ScanAcks` on concrete `FileMailbox` vs `ports.Mailbox` is an architecture/port-shape question for Stanley. Behaviour is tested (`TestAck_WriteScanArchiveRoundTripAndScopeMismatch`, scope-mismatch skip). Item 5 can be ruled on adapter behaviour; port placement is separate.

### Nine exit items — owner-ready standing (2026-09-21)

| # | Standing |
| --- | --- |
| 1 | **Partially satisfied** — linux/amd64 yes; darwin pending |
| 2 | **Partially satisfied** |
| 3 | **Not satisfied** — second adapter exists; contract suite + UI-01–08 on both outstanding |
| 4 | **Satisfied** |
| 5 | **Not satisfied** — mailbox adapter built; assembly wiring + full-path integration test outstanding |
| 6 | **Satisfied** |
| 7 | **Satisfied** |
| 8 | **Satisfied** |
| 9 | **N/A** |

---

## H101-88 acceptance review (2026-09-21)

Reviewed commit `6bae581` (Kevin H101-85: assembly mailbox wiring + integration tests). Re-ruling item 5 only — closes the three follow-ups from [H101-83](#h101-83-acceptance-review-2026-09-21). God verified full suite + race green; code not re-verified here per god H101-89.

**Verdict: ACCEPTED** — item 5 **SATISFIED** with documented limit on sole-writer enforcement.

### 1. Item 5 — the three H101-83 gaps

H101-83 left item 5 **NOT SATISFIED** on three concrete gaps. Each is now closed on `6bae581`:

| Gap (from H101-83) | What closed it | Evidence |
| --- | --- | --- |
| **Assembly wiring** — `Deliverer` existed but product path never invoked it | `assembly.WithSession` calls `driveMailbox` after every command: `DeliverPending` → `IngestPending` → `IngestAcks` | `internal/assembly/session.go`; `TestWithSession_MailboxDeliveryFailureDoesNotFailTheCommand` — mailbox I/O failure logs to stderr and does **not** fail the user command |
| **Full send → publish → process → ack** — deliverer tests used `fakeRecorder` with pre-seeded messages; no real engine chain | `TestWithSession_FullMailboxPath_SendPublishProcessAck` — real `Engine` + `FileStore` + `FileMailbox` through `WithSession`; external ack via `WriteAck` + second session ingests it | `internal/assembly/mailbox_integration_test.go` |
| **Sole writer** — `host.Capabilities` still exposed record methods; no proof only mailbox drove them in product path | Call-graph + containment argument (see §2 below) | Only `assembly/driveMailbox` passes `host.Capabilities` to `mailbox.Deliverer`; `cmd/harnessing` holds `api.FrontendSession` only; import allowlist restricts `host` to `internal/assembly` |

**H101-83 follow-up discharged separately:** `TestDeliverer_IngestAcks_QuarantinesNotFound` (`deliverer_test.go:206`, mirrors Denied at `:169`) — god confirmed; not re-verified here.

**Item 5 ruling: SATISFIED.** All three gaps closed for the Phase 2 product path.

### 2. Sole-writer proof — why call-graph + containment is sufficient here

Item 5's criterion (line 87) is behavioural: *"File-protocol mailbox is the sole writer of `RecordMessagePublished` and `RecordMessageProcessed`."* It does **not** require that those methods be absent from any type — only that the **file-protocol mailbox adapter** is the sole driver of those facts in the shipped product path.

**What the proof covers:**

1. **Containment** — presentation (`cmd/harnessing`) cannot import `host` or call record methods; it sees `api.FrontendSession` only (item 4, H101-73).
2. **Single wiring site** — within `internal/assembly`, only `driveMailbox` constructs a `Deliverer` with `host.Capabilities` as recorder; every command path goes through `WithSession`.
3. **Adapter discipline** — `mailbox.Deliverer` is the only production caller of `RecordMessagePublished` / `RecordMessageProcessed` on that capabilities handle; publish-before-process ordering is enforced inside `Deliverer` (unchanged from H101-83).

**What would break the proof (regression risks a future reader must watch):**

- A new `internal/assembly` code path that holds `host.Capabilities` and calls record methods outside `driveMailbox`.
- Relaxing the import allowlist so presentation or a third package imports `host` and reaches `Capabilities`.
- A second adapter or integration path that injects `Capabilities` directly instead of going through `WithSession` → `driveMailbox`.

These are **reviewable** failures — same class as H101-73's package-level vs type-level `FrontendSession` check. The allowlist + assembly-only wiring is the Phase 2 enforcement mechanism; it is not cryptographic.

**Limit recorded:** `host.Capabilities` still exposes `RecordMessagePublished` and `RecordMessageProcessed`. A careless assembly change could call them without going through `Deliverer`. That is **not** waived — it is **accepted as residual risk** for Phase 2 because the criterion asks for sole-writer behaviour in the product path, not compile-time erasure of the methods.

### 3. When type-level prevention becomes required (Phase 3 obligation)

Type-level removal of record methods from `host.Capabilities` (or narrowing the recorder interface to what `Deliverer` alone implements) is **not** a Phase 2 exit blocker. It becomes a **Phase 3 obligation** if any of the following land:

| Trigger | Why type-level matters then |
| --- | --- |
| **Second message-fact writer** — Phase 3 process supervision or run-output paths that must record message lifecycle facts independently of the file mailbox | Two legitimate writers cannot be disambiguated by "only assembly calls `driveMailbox`" alone; the type system must separate mailbox facts from supervision facts |
| **Third-party or out-of-tree assembly** — an integration that composes `host.Open` + `Capabilities` without using `WithSession` | Call-graph proof is repo-local; external composers need compile-time denial |
| **Repeated sole-writer regressions** — a review catches a second assembly path calling record methods | Hardening stops being optional hygiene and becomes exit criteria for the next phase |

Until one of those triggers fires, documenting the call-graph limit in this review is the correct artefact for phase sign-off.

### Out of scope (per god H101-88)

- Contract suite (`internal/adaptercontract/`) — still absent; item 3 unchanged.
- Walkthrough assertion change — ruled correct by god; not reopened.

### Nine exit items — owner-ready standing (2026-09-21, post-H101-88)

| # | Standing |
| --- | --- |
| 1 | **Partially satisfied** — linux/amd64 yes; darwin pending (H101-55) |
| 2 | **Partially satisfied** |
| 3 | **Not satisfied** — second adapter exists; contract suite + UI-01–08 on both outstanding |
| 4 | **Satisfied** |
| 5 | **Satisfied** — `6bae581`: assembly wiring + full-path test; sole-writer by call-graph + containment (limit: methods remain on `host.Capabilities`; see §2–§3) |
| 6 | **Satisfied** |
| 7 | **Satisfied** |
| 8 | **Satisfied** |
| 9 | **N/A** |

---

## H101-90 — adaptercontract suite stage 1 (UI-01..04) (2026-09-21)

**Verdict:** STAGE 1 COMPLETE — UI-01 through UI-04 run green on **both** adapters (`go test ./internal/adaptercontract/`). **Paused** for god commit before UI-05..08. Stanley independence review: H101-91.

**Package:** `internal/adaptercontract/` — `expected/spec.go` (spec-derived literals only) + `AMBIGUITIES.md` + CLI subprocess driver + throwaway JSON driver + harness reads via `assembly.WithSession` → `GetSnapshot` (no `host` import; H101-92).

### Stage 1 coverage

| Scenario | Tests | CLI | Throwaway | Fixture source |
| --- | --- | --- | --- | --- |
| **UI-01** | `TestUI01_MalformedInputNoMutation`, `TestUI01_ValidRegisterPreservesIDs` | pass | pass | ADR UI-01; empty-workspace revision rule |
| **UI-02** | `TestUI02_ReceiptDistinctFromTaskStatus`, `TestUI02_DeniedAndConflictStableCodes` | pass | pass | ADR UI-02; `domain.ErrDenied` / `ErrConflict` |
| **UI-03** | `TestUI03_IdempotentRetrySameReceipt`, `TestUI03_ChangedPayloadSameRequestIDConflicts` | pass | pass | ADR UI-03; one-commit-per-successful-command rule |
| **UI-04** | `TestUI04_FullCycleDomainOutcomes` | pass | pass | ADR UI-04 + product definition §2 cycle; terminal task `Done` + `res2` |

### Spec ambiguities (substance in `internal/adaptercontract/AMBIGUITIES.md`)

Full writeups live in the repo for Stanley H101-91. Summary:

| ID | Spec says | Spec fails to say | Literal assumes instead |
| --- | --- | --- | --- |
| **A1** (UI-03) | Generated IDs surfaced for interrupted callers; request-id idempotency | Whether caller-supplied entity IDs count as “generated”; what surface = “surfaced” | Request ID + revision on receipt = recoverable identity; entity IDs pre-held by caller |
| **A2** (UI-02) | IOFailure uncertainty messaging; stable codes | Whether stage 1 needs fsync fault injection; OutcomeUncertain vs IOFailure assert target | Stage 1 = Denied/Conflict + receipt/status split only; IOFailure deferred |
| **A3** (UI-04) | **Resolved** — boundaries H101-94 (`d372518`): 12 user groups + 2 delivery groups = revision 14 | — | `UI04CycleEnd.WorkspaceRevision = 14` at equivalent committed cut |

### Adapter divergences (stage 1)

**None.** Both adapters pass the same independently specified domain assertions. Rendering differs (CLI stderr receipt line vs throwaway JSON `Response`); semantic checks use harness snapshot and stable error codes.

### Stage 2 scope — COMPLETE (2026-09-21)

UI-05..08 on **both** adapters; crossover continuation; multi-session observation via `assembly.OpenContractHarness` (flock-safe; adaptercontract still does not import `host`).

| Scenario | Tests | Notes |
| --- | --- | --- |
| **UI-05** | `TestUI05_ProvenanceDeliveryFactsAndDisclosure` | CLI `messages`/`message`; throwaway snapshot/message; disclosure + delivery facts + unverified provenance |
| **UI-06** | `TestUI06_*` (5 tests) | Discovery without injected IDs; cursor observe; restart `CursorExpired`; concurrent commit ordering; detachment probe |
| **UI-07** | `TestUI07_*` (3 tests) | Graceful reopen; request-id replay; detach does not cancel other session work |
| **UI-08** | `TestUI08_*` (2 tests) | Detachment probe; throwaway `start-run`/`stop-run`/`set-run-budget` → `Unsupported` |
| **Cross-cutting** | `TestCrossAdapter_Continuation` | CLI writes, throwaway continues to UI-04 end state |

Verify: `go test ./internal/adaptercontract/` — all green. Lint: `golangci-lint run ./internal/adaptercontract/...` — 0 issues.

---

## H101-92 — ambiguities on disk + host import (2026-09-21)

**Verdict:** ADDRESSED — ambiguities written to `internal/adaptercontract/AMBIGUITIES.md`; `host` import removed from suite.

### God Q1 — ambiguities not recorded

**Fixed.** `AMBIGUITIES.md` in the adaptercontract package (linked from `expected/spec.go` doc comment). DoD table above cross-references. Substance was only in outbox message for `9d2cce5`; now in repo for Stanley.

### God Q2 — why `host`, and allowlist exception?

**Why it was there:** `harnessSnapshot` called `host.Open` + `GetSnapshot` for independent mutation checks (ADR 0003 §suite: host as composition entry). **Unnecessary:** `assembly.WithSession` → `api.FrontendSession.GetSnapshot` observes the same committed state without importing `host`.

**Allowlist exception:** **No.** Do not add a second `host` importer entry like statestore’s test exception. The suite should not import `host`; harness now uses assembly only. Import set: `api`, `assembly`, `domain`, `throwawayadapter`, `expected` (+ stdlib).

**Stage 2:** proceeds; H101-55 darwin manifest can run in parallel — not sequencing ahead unless you redirect.

---

## H101-98 — darwin/arm64 milestone manifest (2026-09-21)

**Verdict:** MANIFEST RECORDED — [darwin-arm64-milestone-manifest-2026-09-21.md](darwin-arm64-milestone-manifest-2026-09-21.md) at commit `02b140768176991c9128772589cbbed7ec2d9691` (clean tree).

### What it discharges (darwin/arm64 only)

| Item | Effect |
| --- | --- |
| **1** darwin half | **Satisfied** — `TestCLI_Phase2ProductWalkthroughSubprocess` pass on disclosed host |
| **2** darwin native | **Satisfied** — `TestRun_Phase2CycleRecoversAfterKilledCLI` + H101-20 lock tests pass on disclosed host |
| **2** overall | **Still partially satisfied** — cross-target gaps unchanged (handoff ack crash proof host-side on both targets) |
| **3** | **Not discharged** — adaptercontract native darwin run not in this manifest |

### Evidence strength

Single-machine disclosed manifest — weaker than CI. **Replacement:** macOS CI runner re-running the same three test blocks on every PR, with runner image ID recorded.

### God follow-ups (H101-55, still want)

1. **macOS CI runner** — raise **now** (manifest done; runner replaces manual re-runs).
2. **Windows exclusion smoke** — carded H101-101; ruled H101-103 below.

---

## H101-103 — Windows exclusion smoke (`2545f42`) (2026-09-21)

**Verdict: SATISFIED WITH LIMIT** — Kevin's compile-only guard discharges the exit check Kelly proposed.

**What Kelly asked for:** Phase 2 exit assertion that Windows is **excluded by design** — workspace operations return stable `Unsupported`; `version`/`help` still work. Not Windows support.

**What landed (`2545f42`):** `lock_windows_test.go` + `windows_unsupported_test.go` (`//go:build windows`); CI `GOOS=windows` whole-product `go build` + `go vet` of windows-tagged tests. Documented in [`h101-101-windows-exclusion.md`](h101-101-windows-exclusion.md).

**Ruling — workspace path:** **Satisfied.** The substantive claim is that the **product path** refuses workspace work cleanly (`assembly.WithSession` → non-zero exit, `Unsupported` on stderr, session body never runs). Compile-time proof is proportionate for an **out-of-scope** platform: ADR 0001 excludes Windows; the check asserts defined refusal, not feature correctness. In-scope targets (darwin) still owe **executed** evidence — asymmetric by design, not budget overriding criteria.

**Ruling — `version`/`help` not named in tests:** **Acceptable.** They never call `WithSession` (`run.go`); no Windows-specific branch exists on that path. Whole-product `GOOS=windows` build of `cmd/harnessing` is sufficient structural proof they compile; a dedicated runtime test would add no exclusion evidence Kelly's check was written to capture.

**Limit recorded:** Proves **shape**, not Windows **runtime** behaviour (`go vet` does not execute test bodies; no Windows runner). Acceptable for this exclusion check. Would **not** discharge an in-scope platform. Stronger proof (paid runner or one manual `.exe` run) is optional hygiene, not required to mark this exit sub-check satisfied.

**God budget note:** Disproportionate to pay for Windows execution to prove correct refusal — aligns with this ruling; does not extend to darwin/macOS runner.

---

## H101-105 — macOS runner (`f6075fa`) (2026-09-21)

Full ruling: [`h101-105-macos-runner-ruling.md`](h101-105-macos-runner-ruling.md).

| Question | Verdict |
| --- | --- |
| Runner discharges darwin re-proof on every change? | **Satisfied** — GHA run `35598625946` green on `2534afb` |
| Runner vs manual manifest? | **Both stand** — manifest historical; runner is live standing evidence |

---

## H101-90 stage 2 + H101-105 upgrade — standing (2026-09-21)

| # | Standing |
| --- | --- |
| 1 | **Partially satisfied** — linux + darwin CI runner green (`35598625946`); manifest retained as historical |
| 2 | **Partially satisfied** — darwin native crash+lock in CI; cross-target ack gap (H101-104) open |
| 3 | **Satisfied with limit** — see H101-107; Stanley `c77783f` G1–G7 dispositioned; G5 carded |
| 4–8 | Unchanged |
| 9 | **N/A** |

---

## H101-107 — Phase 2 item 3 ruling (2026-09-21)

Full ruling: [`h101-107-phase2-item3-ruling.md`](h101-107-phase2-item3-ruling.md).

**Verdict: SATISFIED WITH LIMIT** — suite structurally complete, green on both adapters
in CI (`e565b82`), cites Stanley H101-91 stage 2 (`docs/architecture/h101-91-stage2-expectation-traceability.md`, `c77783f`).

| Gap | Disposition |
| --- | --- |
| G1 rendering as canonical state | **(c)** CLI-local limit |
| G2 event subject cardinality | **(c)** fixture subject presence only |
| G3 cursor encoding / restart expiry | **(c)** named H101-70 scenario only |
| G4 strict revision growth | **(c)** ordering guard, not dedup proof |
| G5 unregistered assignee in harness | **(b)** carded — fix before harness paths credit registry |
| G6 detail / precedence | **(c)** Unsupported availability only |
| G7 task-revision preconditions | **(c)** workspace cut, not task-revision arithmetic |
