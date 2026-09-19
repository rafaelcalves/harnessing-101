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
| **1** | ≥2 agents, full task/message cycle, restart durability, port APIs + fake adapters, zero network, tests green | Plan omits agent count; product §2 implies two agents |
| **2** | CLI completes product §2 cycle; throwaway adapter passes contract tests; Phase 3 ops return `Unsupported` | "Usable for real work" — anchor to §2 walkthrough |
| **3** | Start/stop/budget/crash-recovery on supported platforms; process-tree termination; misbehaviour → `RecoveryRequired` | Platform matrix **UNKNOWN** (`boundaries.md` line 65) |
| **4** | Clean-machine install, observed egress audit, new-joiner docs, version tag | Needs written observation protocol |

**Weakest phase exit:** Phase 0 — "a repository a stranger could contribute to" has no objective threshold, and the plan still recommends TypeScript while ADR 0001 accepts Go (see audit).

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
