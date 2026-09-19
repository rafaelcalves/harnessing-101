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
| **3** | Start/stop/budget/crash-recovery on supported platforms; process-tree termination; misbehaviour → `RecoveryRequired` | Platform matrix **UNKNOWN** (`boundaries.md` line 65) |
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
| 1 | CLI product cycle (§2 walkthrough) | **No** — needs `cmd/harnessing` and a CI subprocess test |
| 2 | Restart after crash | **No** — needs H101-20 recovery (or documented manual procedure exercised in test) plus CLI reopen |
| 3 | Swappability (ADR 0003 UI-01–08) | **No** — needs `internal/adaptercontract` suite, FrontendSession, throwaway adapter |
| 4 | Containment preserved | **Partially** — store half satisfied (`24a2022`); UI import allowlist (no host from presentation) not yet enforced |
| 5 | Mailbox adapter (H101-22) | **No** — needs file mailbox wired to delivery-fact recorders |
| 6 | First-run disclosure (H101-16) | **Partially** — README disclosure present; in-product first-run line needs CLI test |
| 7 | Phase 3 ops `Unsupported` | **No** — needs Capabilities/CLI surface and explicit `Unsupported` assertions |
| 8 | Zero network (CLI tree) | **Satisfied** — `go list -deps ./cmd/harnessing` finds no `net`/`net/http` (`24a2022`) |
| 9 | Test layering | **N/A** — policy; enforced by review |

1. **CLI product cycle (§2 walkthrough)** — A CI-runnable script or `go test` drives the **minimum useful product** in [`docs/product/definition.md`](../product/definition.md) §2 exclusively through the shipped `harnessing` command (subprocess, not in-process `host` import from the test package): create/open workspace; register ≥2 named agents; create a task with one accountable owner; task-linked handoff with explicit acknowledgement; report a blocker and resolve it; report a result; human reject then accept a replacement result; status queries show assigned / in-progress / blocked / awaiting-review with reporter identity; close the CLI; reopen; assert durable records without resending work. **This item is what “usable for real work” means** — not a subjective judgment.

2. **Restart after crash** — After the cycle in item 1, simulate an unclean host exit (`kill -9` or equivalent) without `Close`; a subsequent `harnessing` invocation must reopen the same workspace and recover the persisted cycle state. **Blocked until H101-20** delivers stale-lock recovery **or** a documented, tested manual `.lock` removal procedure is exercised in CI. Graceful reopen alone does **not** discharge this item (ADR 0003 UI-07). Inherited from Phase 1 limits; not deferrable to Phase 3.

3. **Swappability (shared adapter contract)** — [ADR 0003](../adr/0003-ui-adapter-contract.md) (H101-40) defines behavioral scenarios **UI-01 through UI-08**; this item requires both adapters to pass them in CI, not reproduce the architecture here. The shared black-box suite lives in `internal/adaptercontract/contract_test.go`; each adapter enters through its **real** input driver (CLI argv/stdin/structured output; throwaway independent handler — no CLI imports, no shared dispatch). Observations come through the caller-bound **FrontendSession** surface (not raw `Capabilities` injection). Suite includes per-adapter isolated workspaces, cross-adapter continuation (A writes, B reads/acks/reviews), and concurrent observer visibility. Compare normalized domain outcomes per ADR 0003 — not adapter rendering. **Fake run-output stream tests do not count** toward replaceability; actual supervision/output stays Phase 3. Core must import neither adapter.

4. **Containment preserved** — `scripts/check-import-allowlist.sh` green on `main`; store-import allowlist unchanged (only `internal/host` constructs persistence). **Extend** checks so presentation packages cannot import `host`, `core/task`, storage, or outbound ports (ADR 0003 — no storage allowlist exception for UI). Forbidden fixture fails the check. H101-34: fixture step must require exit code **1**, not any non-zero.

5. **Mailbox adapter (H101-22)** — File-protocol mailbox is the sole writer of `RecordMessagePublished` and `RecordMessageProcessed`; publish precedes process; skipping publish or reversing order is rejected with persisted state unchanged. `MessageAcknowledgement` control records and `AcknowledgeMessage` share the same recipient and idempotency rules. Integration test covers at least one full send → publish → process → ack path through the adapter.

6. **First-run disclosure (H101-16)** — Automated test: on first `harnessing` use in a fresh workspace, the unconditional ADR 0002 disclosure line (“does not start, observe, or restrict any agent process…”) is emitted before interactive coordination commands are accepted. No validation-looking cues on manually sourced content. README clone-and-run disclosure remains (regression check against existing text).

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
