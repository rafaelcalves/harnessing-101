# Phase 3 exit criteria (2026-09-21)

Kelly QA. Written at **phase start** (H101-126), not mid-phase — same pattern as
Phase 2 exit (H101-39). Owner approved opening Phase 3 on 2026-09-21 (H101-114).

**Scope source:** [`docs/product/definition.md`](../product/definition.md) §3 refusal
list — background process supervision, restart guarantees, and hard budget enforcement
were explicitly refused before Phase 3. Phase 3 is where product-managed
**start**, **stop**, **supervise**, and **budget** become real. Phase 2 exit item 7
(`StartRun`/`StopRun`/`SetRunBudget` return `Unsupported`) is **discharged** by this
phase; Phase 2 itself is closed and not reopened here.

**Contract sources:** [`boundaries.md`](../architecture/boundaries.md) (commands,
`ProcessSupervisor`, `OutputJournal`, supervision/recovery inferences),
[`threat-model.md`](../security/threat-model.md) rule 5 and §5,
[ADR 0002](../adr/0002-manual-agent-exposure-in-phases-1-2.md),
[ADR 0003](../adr/0003-ui-adapter-contract.md),
[Creed's Phase 2 exit boundary review](../security/h101-phase2-exit-boundary-review.md)
(H101-115 carry-forwards).

---

## Inherited facts (not re-litigated)

Phase 3 **inherits** the following from Phase 2 exit. They stay green; this phase does
not repeat their proof unless a Phase 3 change regresses them:

| Inherited | Authority |
| --- | --- |
| Containment allowlist (store + presentation imports) | Phase 2 item 4 |
| Zero network on `cmd/harnessing` / CLI tree | Phase 2 item 8 |
| Mailbox sole-writer | Phase 2 item 5 |
| Phase 2 adaptercontract UI-01–08 on both adapters | Phase 2 item 3 (SATISFIED WITH LIMIT) |
| Supported targets `linux/amd64` + `darwin/arm64` | ADR 0001 H101-50 |
| Windows out-of-scope exclusion smoke | H101-103 — compile-only guard; **not** in-scope partial satisfaction |

**Platform rule for Phase 3 items below:** each in-scope target owes **native** evidence
on `linux/amd64` (ubuntu CI) and `darwin/arm64` (macos-14 CI). Ubuntu-only proof does
not discharge macOS. Windows workspace commands remain `Unsupported`; `version`/`help`
unchanged — same asymmetric pattern as Phase 2 item 2.

**Phase 2 item 3 limits (G1–G4, G6–G7):** restated for any **new** Phase 3
adaptercontract scenarios in item 7 — presentation **meaning** is credited, not exact CLI
label punctuation, stderr placement, or unsettled malformed-payload precedence (see
[`h101-107-phase2-item3-ruling.md`](h101-107-phase2-item3-ruling.md)). G5 (unregistered
assignee) is **closed** and does not recur.

---

## Creed H101-115 carry-forwards — where they land

| # | Constraint (Creed) | Exit representation | Checkable? |
| --- | --- | --- | --- |
| CF1 | Registry membership on sender/recipient is **not** authentication — `StartRun`/budget must not key off a message's claimed `SenderAgentID` | Item 1 decision table **D3**; named negative test | **CI** |
| CF2 | Manual-start exposure does **not** close because Phase 3 adds process control; ADR 0002 revisit trigger | Item 8 disclosure text + provenance unchanged on pre-Phase-3 records | **Disclosure** (+ CI spot-checks on metadata) |
| CF3 | `ProcessSupervisor.Start` approval gate gates **future** starts only — not content already in workspace or sender fields already trusted downstream | Items 1 **D4** and 8 **D2** | **CI** + **disclosure** |

CF2's user-understanding half ("whether users tolerate the limitation") remains
**UNKNOWN** per ADR 0002 — not a CI gate. CF2 is **not dropped**; it is an honest
disclosure obligation with checkable doc/provenance assertions.

---

## Phase 3 is not accepted until all ten items pass

| # | Criterion | CI-checkable? |
| --- | --- | --- |
| 1 | Approved-profile `StartRun` | **Yes** |
| 2 | `StopRun` and owned process-tree termination | **Yes** |
| 3 | Hard budget enforcement (elapsed + token paths) | **Yes** |
| 4 | Controller crash recovery and `RecoveryRequired` | **Yes** |
| 5 | Real run output journal (not fake stream) | **Yes** |
| 6 | Background hosting survives UI detach | **Yes** |
| 7 | Phase 3 adapter contract on both adapters | **Yes** (with inherited G-limits) |
| 8 | Trust-boundary honesty (Creed CF2/CF3) | **Disclosure** + partial CI |
| 9 | Runnable local startup guide | **Acceptance procedure** (not automated CI) |
| 10 | Registered external agent session (attachment-first) | **Yes** (Layer A) + manifest (Layer B) |

**Count: 10 items** — **amended H101-196 (2026-09-22)** after owner H101-174
attachment-first choice. Seven core process-control items remain CI-checkable on
`main`. Item 8 is primarily disclosure with named CI spot-checks. Item 9 is a
cold-read acceptance procedure. Item 10 is **required** for Claude Code
participation under attachment-first; it does **not** discharge item 1 managed
`StartRun` (see item 10 spec).

**Budget enforcement shape:** **one exit item** (item 3) with **two named failure-mode
rows** (elapsed-time stop vs reported-token stop). Both must actually **stop** a run;
splitting into two exit items would let one pass while the other is stubbed. The decision
table keeps them coupled.

---

### Item 1 — Approved-profile `StartRun`

**Amended H101-147 (2026-09-21)** — managed start must launch a participating agent
tool, not a generic process. See [amendment](#h101-147--item-1-managed-agent-tool-evidence-2026-09-21).

**Amended H101-158 (2026-09-22)** — initial managed-run caller scope and one-active-run
per agent, per Stanley H101-128. See [amendment](#h101-158--item-1-caller-scope-and-concurrency-2026-09-22).

`StartRun` through the shipped `harnessing` command (subprocess, same surface as Phase 2
item 1) must:

- Persist start intent before invoking `ProcessSupervisor.Start` (boundaries supervision
  inference).
- **Reject** a run whose `profileID` lacks a prior recorded user-approval event (threat
  model rule 5) with stable `Denied` or documented equivalent — not silent success, not
  panic.
- **Accept** a run with an approved profile and reach `Running` (or documented
  `Starting`→`Running` progression observable through `GetRun` / state events).
- Authorize through **host-established caller scope**, never from a message envelope's
  claimed `SenderAgentID` alone.

#### Decision table (read before any run)

| # | Observation | Defect class | Fix |
| --- | --- | --- | --- |
| D1 | `StartRun` succeeds with unapproved `profileID` | **Product** | Approval gate missing or bypassed |
| D2 | Approved `StartRun` never reaches observable `Running` | **Product** | Supervision wiring or state machine |
| D3 | Fixture sends `StartRun` authorized only because a **message** claimed sender matches run agent (no caller-scope path) | **Product** | **CF1 violation** — registry ≠ auth |
| D4 | Pre-approval workspace message shows `IdentityVerification` upgraded to verified after first successful `StartRun` | **Product** | **CF3 violation** — retroactive validation |
| D5 | `StartRun` receipt missing `operationID` when async work continues | **Product** | boundaries command-completion rule |
| D6 | Test calls `host`/`Capabilities` directly instead of `harnessing start-run` (or CLI equivalent) | **Test** | Not product-surface proof |
| D7 | Pass on one CI target only via `t.Skip` | **Test** | Both ADR targets must run natively |

---

## H101-147 — Item 1 managed agent-tool evidence (2026-09-21)

**Product ruling (Angela, owner H101-142):** profiles invoke the user's
**already-installed agentic command-line interface (CLI)** — e.g. Claude Code, Codex,
Cursor Agent. The user installs and authenticates; the product uses them. Generic
approved-process→`Running` proof is **withdrawn** — it could pass while starting
nothing a user could coordinate with.

**Dispatch-ordering invariant:** unchanged. Stanley H101-135 numbered sequence
(commit intent → dispatch-attempted marker → `StartRun` once → separate outcome
commit; crash between marker and Start stays ambiguous/`RecoveryRequired`). This
amendment does not relax it.

### Required evidence (replaces bare `Running` proof)

| # | Requirement | Checkable? |
| --- | --- | --- |
| R1 | **Selected tool** — `StartRun` launches the executable named by an **approved profile revision** for the bound agent, not an ad hoc argv | **CI** |
| R2 | **Participation context** — the start path supplies task and workspace context the tool needs to **participate** (paths, identifiers, or documented protocol handoff), not merely spawn a process | **CI** (fixture) + **manifest** (real tool) |
| R3 | **Honest failure modes** — missing binary, authentication-required, unsupported invocation mode, and spawn failure each return stable, actionable errors — not `Running`, not silent success | **CI** (fixture negatives) + **manifest** (real-tool auth/network) |
| R4 | **Approval / caller scope** — all pre-H101-147 gates preserved: profile approval event, host caller scope, no message-`SenderAgentID` authorization (CF1), no retroactive provenance upgrade (CF3) | **CI** |
| R5 | **Installed ≠ approved** — detecting an installed tool on `PATH` does **not** imply workspace approval; unapproved profile still `Denied` | **CI** negative |

### Two-layer proof model

**Layer A — CI (both ADR targets, every merge):** Use a **participation fixture**
shipped in-repo: a minimal approved profile pointing at a test binary/script that
(1) is launched through the full shipped `harnessing start-run` path under
continuing-host mode, (2) receives injected task/workspace context via the
profile's documented mechanism, (3) emits an observable participation signal
(reads context, writes a protocol-compliant file or exit marker). Prove
`Starting`→`Running`, context present, and R3 negative cases with the fixture.
**A `sleep`/`cat` profile cannot discharge Layer A.**

**Layer B — real agentic CLI (per target, before Phase 3 exit):** **Amended
H101-153** — requires a committed **runner script** (owner's demo/dev script) that
produces a **manifest as output**, plus one **compatibility descriptor** per
named tool (or deferral entry). See
[`h101-153-phase3-item1-layerb-refinements.md`](h101-153-phase3-item1-layerb-refinements.md).
At least **one** owner-named agentic CLI per ADR target (`linux/amd64`,
`darwin/arm64`) must have runner-produced manifest, **or** owner-documented
deferral in descriptor format. Hand-written manifests do not count. Manifest must
show: managed start through the product (not manual shell), authenticated tool
state, context supplied, and honest handling when auth or launch fails.

**CI cannot require live provider network.** Claudio H101-141: installed Claude
Code blocked at `api.anthropic.com:443` in the hive environment — a clean
reachability failure, not a product defect. Layer B is **manifest/disclosure** on
CI; Layer A is **automated CI**. Do not fake Layer B in CI with network calls.

**Three named tools** (Claude Code, Codex, Cursor Agent) are **compatibility
targets**, not one proof for all. Each needs Layer B evidence or explicit owner
deferral before release claims that tool.

### Amendment to item 1 decision table (add rows)

| # | Observation | Defect class | Fix |
| --- | --- | --- | --- |
| D8 | `Running` but launched process received no task/workspace context | **Product** | R2 — launch ≠ participate |
| D9 | Layer A uses non-participating fixture (`sleep`, echo-only) | **Test** | Participation fixture required |
| D10 | Missing binary / auth-required / unsupported mode → `Running` or silent success | **Product** | R3 |
| D11 | Installed tool on `PATH` starts without profile approval | **Product** | R5 — installed ≠ approved |
| D12 | Spawn before dispatch-attempted marker durable, or duplicate Start on replay | **Product** | H101-135 violation |
| D13 | Layer B claimed from manual shell start or spike without managed `StartRun` | **Evidence** | Manifest must use product path |
| D14 | Layer B manifest hand-written or edited without runner output | **Evidence** | H101-153 — runner produces manifest |
| D15 | No committed documented runner for Layer B | **Evidence** | H101-153 — script required |
| D16 | New agentic CLI would need hardcoded start-path branch | **Product** | H101-153 — descriptor extensibility |
| D17 | Runner substitutes for Layer A participation fixture in CI | **Test** | Fixture only in CI |
| D18 | Agent already has an active run (`Starting` or `Running`) and a second `StartRun` for that agent succeeds | **Product** | H101-128 — reject second active run for same agent (`Conflict`) |
| D19 | `StartRun` for agent B succeeds when host-bound caller is agent A (cross-principal start without a host-granted managed-run capability for that profile) | **Product** | H101-128 — initial policy: principal starts **own** registered agent only |

### H101-158 — Item 1 caller scope and concurrency (2026-09-22)

**Authority:** god conflict ruling on Creed vs Kelly (2026-09-22), citing Stanley
[`h101-128-phase3-supervision.md`](../architecture/h101-128-phase3-supervision.md)
Approval and caller authority — items 1 and 8, and Start, stop and operation
completion — items 1 and 2.

**Ruling:** **Add rows only** — original item 1 text and prior amendments (H101-147,
H101-153) are **retained**. This amendment **tightens** item 1; it does not waive any
existing row.

| H101-128 source | New row |
| --- | --- |
| Start table: “Reject a second active run for the same agent initially (`Conflict`)” | **D18** |
| Approval section: “a principal may start its own registered agent only with a host-granted managed-run capability for that profile” | **D19** (self-scope half only) |

**Explicitly out of item 1:** “human admin may manage registered agents” from the same
H101-128 paragraph. That requires a new `CallerScope` process-admin capability
(`IsHumanReviewer` is task-review authority only). Card separately; ship restrictive
(self-scope) first, widen explicitly later (same pattern as ADR 0006 grants).

**D5 note:** H101-128’s “receipt/operationID before any spawn” is forwarded to Stanley
for the existing D5 row. No change to D5 in this amendment pending his ruling.

**Standing at `20a6f8a` (Kelly re-verdict 2026-09-22):** Item 1 remains **NOT
SATISFIED**. Layer A CI and product rows below are discharged on both ADR targets
via `make build test vet lint fmt` green locally (darwin/arm64) and the same
suite on ubuntu + macos-14 CI. **Sole remaining item 1 blocker:** Layer B for
Claude Code — no runner-produced manifest on record; descriptor `deferred: false`.
Codex and Cursor Agent have owner deferrals in committed descriptors (not
blockers). Owner decision H101-163 (participation protocol vs Claude deferral) is
the path to close that slot; Kelly does not rule it here.

| Row | Verdict | Evidence (quote tails, not summaries) |
| --- | --- | --- |
| D5 | **SATISFIED** | `TestStartRun_ReceiptCarriesOperationIDWithQueryableProgress`, `TestStartRun_OperationIDIsStableAcrossReceiptReplay` (`20a6f8a`); `TestCLI_StartRun_ReceiptOperationIDIsQueryable` (`20a6f8a`) — all PASS |
| D18 | **SATISFIED** | `TestStartRun_SecondActiveRunForSameAgentIsConflict`, `TestStartRun_NewRunAllowedAfterPriorExit` (`5048489`) — PASS; engine-only, no dedicated CLI subprocess row |
| D19 | **SATISFIED** | `TestStartRun_CallerScopeNotRequestFields` rewritten for self-scope (`5048489`) — PASS; engine-only |
| R3/D10 CI negatives | **SATISFIED** | `TestCLI_StartRun_MissingToolIsHonestNotRunning`, `TestCLI_StartRun_AuthRequiredFixtureEndsExitedNotRunning`, `TestCLI_StartRun_UnsupportedContextTransportEndsExitedNotRunning` (`5048489`); engine `TestStartRun_SupervisorFailureRecordsExitedHonestly` — PASS |
| Layer B Claude | **NOT SATISFIED** | GETTING-STARTED §9: network policy blocked before runner wrote a report; no `cycle_completed` manifest. D14/D15 runner exists; evidence file absent |

### H101-172 — Registered external session vs item 1 Layer B (2026-09-22)

**Ruling:** owner H101-163 registration shape **does not discharge** item 1 Layer
B. Pre-existing user-started sessions are **outside** D13–D15 managed-`StartRun`
evidence (D13 forbids claiming Layer B from manual/spike start). **Recommend new
exit item** for registered-session participation; **do not amend** item 1 Layer B
to absorb registration. Claude Layer B still closable via managed-`StartRun`
runner manifest or descriptor deferral only. Evidence bar for registration:
provisional pending Stanley H101-171. Full ruling:
[`h101-172-layerb-registration-ruling.md`](h101-172-layerb-registration-ruling.md).

### H101-196 — Attachment-first gate disposition (2026-09-22)

**Authority:** owner H101-174 — attachment-first (**A**); managed launch (**B**) on
near-term backlog.

| Question | Verdict |
| --- | --- |
| Item 1 Layer B Claude under attachment-first | **Closes by owner DEFERRAL** (H101-153), **not** registration, **not** manifest — pending committed descriptor update |
| New exit item | **Item 10 written** — [`h101-196-item10-registered-session-acceptance-spec.md`](h101-196-item10-registered-session-acceptance-spec.md) |
| Backlog B vs plain deferral | **Same deferral gate**; **extra** `deferral.backlog` metadata + item 10 obligation for Claude — not identical to Codex/Cursor indefinite deferrals |

**Layer B Claude after deferral commit:** **SATISFIED BY OWNER DEFERRAL** — **not**
demonstrated managed-start compatibility. Item 1 overall re-verdict when descriptor
lands. **Item 10 NOT SATISFIED** until implementation.

**Honest ceiling:** Phase 3 may exit without Claude Code managed `StartRun` proof;
attachment proved via item 10 only. Manual attachment cannot be relabelled
managed-start evidence (D13 unchanged).

Full ruling:
[`h101-196-attachment-first-gate-disposition.md`](h101-196-attachment-first-gate-disposition.md).

### H101-153 Layer B refinements (2026-09-21)

**Refinement 1:** runner script **required**; manifest is runner output only.
**Refinement 2:** descriptor-per-tool schema **required**; extensibility proved
by schema+loader (not a fourth tool). Descriptor field schema waits for Stanley
ruling on Claudio's proposal; principle fixed now. Full ruling:
[`h101-153-phase3-item1-layerb-refinements.md`](h101-153-phase3-item1-layerb-refinements.md).

### Estimate implication

**Material.** Kevin's 3–5 agent-day estimate assumed dispatch-ordering as the hard
part against the **old** generic-process criterion. H101-147 adds: participation
fixture + profile context wiring + failure taxonomy + Layer B manifest procedure.
**Owner should not hold the stale 3–5 figure** — plan **~5–8 agent-days** for item
1 alone, with dispatch-ordering still the deepest invariant work.

### Item 9 coupling

When the Phase 3 guide supplement lands (H101-138 limit), it must document managed
`StartRun` with a **named participation fixture** walkthrough and state which
agentic CLIs have Layer B manifests vs owner deferral. **H101-196:** Claude Code
must be listed as **deferred managed-start** with the **item 10 registration**
walkthrough as the participation path — not Layer B manifest evidence.

Full amendment authority: Angela H101-146 / owner H101-142. Ruling:
[`h101-147-phase3-item1-amendment.md`](h101-147-phase3-item1-amendment.md).

---

### Item 2 — `StopRun` and owned process-tree termination

**Amended H101-130 (2026-09-21)** — tree scope bounded; see [amendment](#h101-130--item-2-tree-scope-2026-09-21).

`StopRun` through the shipped command surface must:

- Transition run to `Stopping` then `Exited` only after **observed** child exit (boundaries:
  kill request alone is not an `Exited` event).
- On supported platforms, apply graceful termination then configured **forced termination
  of the owned process tree** where the adapter documents support.
- Leave persisted run state consistent after stop completes; duplicate `StopRun` is
  idempotent or stably rejected, not a second orphan tree.

#### Decision table

| # | Observation | Defect class | Fix |
| --- | --- | --- | --- |
| D1 | `Exited` recorded while test child still alive (PID probe) | **Product** | Premature terminal state |
| D2 | Child survives stop (orphan process) on platform claiming tree kill | **Product** | Supervision/termination policy |
| D3 | `StopRun` on unknown `runID` returns `NotFound`, state unchanged | **Product** | Error taxonomy |
| D4 | Stop only kills parent, leaves child worker running | **Product** | Tree termination |
| D5 | Test asserts exit from mock without spawning real child | **Test** | Insufficient for item 2 — mock OK only for adaptercontract **error-path** rows, not this item's termination proof |

---

### Item 3 — Hard budget enforcement

**Amended H101-130 (2026-09-21)** — elapsed deadline evidence model; see [amendment](#h101-130--item-3-elapsed-budget-evidence-2026-09-21).

**One item, two stop paths.** The product must **actually stop** a run when a configured
limit is exceeded — the first phase where "enforce" means termination, not disclosure.

| Path | Requirement |
| --- | --- |
| **Elapsed time** | `SetRunBudget` with positive elapsed limit; run **terminates** (observable `Exited` or `Stopping`→`Exited`) before or at limit under test clock; not merely a warning event |
| **Reported tokens** | When the execution spec/agent path supports token reporting, exceeding reported-token limit **terminates** the run; when unsupported, `SetRunBudget` token fields return `Unsupported` — not silently ignored |

Budget decisions stay in core policy (boundaries: budget decisions inside core); adapters
surface observations only.

#### Decision table

| # | Observation | Defect class | Fix |
| --- | --- | --- | --- |
| D1 | Elapsed limit exceeded; run still `Running` | **Product** | Hard stop missing — **item 3 blocker** |
| D2 | Token limit exceeded with reporting enabled; run continues | **Product** | Token budget not enforced |
| D3 | Token limit set on agent that cannot report tokens; returns success | **Product** | Must be `Unsupported` or explicit denial |
| D4 | Budget breach emits warning event but process keeps running | **Product** | Warning ≠ enforcement |
| D5 | `SetRunBudget` after run already `Exited` mutates terminal state | **Product** | State machine integrity |
| D6 | Test uses wall clock without injectable/test clock for elapsed path | **Test** | Flaky or unbounded — use controlled clock per boundaries port 9 |
| D7 | Budget keyed off message `SenderAgentID` instead of run/agent binding | **Product** | **CF1 violation** (same class as item 1 D3) |

---

### Item 4 — Controller crash recovery and `RecoveryRequired`

After unclean controller exit during an active or recently active run (`kill -9` on host
holder, no graceful `Close`):

- Next `harnessing` invocation reopens workspace and surfaces run state honestly.
- `Recover` establishes process identity/ownership or returns `Unknown`; reused PID alone
  is insufficient (boundaries supervision inference).
- Core marks `RecoveryRequired` and **refuses automatic duplicate spawn** when identity
  cannot be established.
- Output gaps after crash are **reported**, not fabricated.

Extends Phase 2 item 2 crash pattern from task/message cycle to **run supervision**.
Cross-target native evidence on both ADR targets (same CI jobs as Phase 2).

#### Decision table

| # | Observation | Defect class | Fix |
| --- | --- | --- | --- |
| D1 | After crash, silent second child for same `runID` | **Product** | Duplicate spawn |
| D2 | Run stuck `Running` though child is dead | **Product** | Recovery/detection |
| D3 | `RecoveryRequired` never surfaced when identity unknown | **Product** | State honesty |
| D4 | Fabricated output bytes filling gap after crash | **Product** | OutputJournal integrity |
| D5 | Crash test kills after `Close` | **Test** | Not unclean exit |
| D6 | Assert run state only via `host.GetSnapshot`, not CLI/`harnessing` query | **Test** | Same class as H101-104 D8 |

### H101-180 — Item 4 standing at `2b76d80` (2026-09-22)

**Verdict: NOT SATISFIED.** H101-170 minimal slice **PARTIALLY ACCEPTED** — authorised
increment only. Four new tests PASS under `-race` (`make build test vet lint fmt` green).

| Stanley E# / row | Verdict |
| --- | --- |
| E1 reconcile before admit | **SATISFIED** |
| E2 `RecoveryRequired` through product (CLI) | **SATISFIED** at `21c782e` (H101-183) |
| E3 same-agent refuse, no child | **SATISFIED** |
| E4 persist across reopen | **SATISFIED** |
| E5 unrelated agent eligible | **SATISFIED** |
| D1 duplicate spawn | **Partial** (agent-scoped, not same-`runID`) |
| D2 stuck `Running`, dead child | **Open** |
| D3 surface `RecoveryRequired` | **Partial** (stuck-`Starting` at reopen) |
| D4 output gaps | **Open** |
| D5 unclean crash test | **SATISFIED** (native SIGKILL window) |
| D6 CLI not host query | **SATISFIED** at `21c782e` (H101-183) |

Kevin's non-claims (no live supervision, adoption, termination, output continuity, item 4
acceptance) are **correctly scoped**. Defensive Succeeded-operation guard:
**acceptable with recorded limit** — invariant in H101-180 ruling; vacuous same-run test
correctly avoided. Full ruling:
[`h101-180-phase3-item4-minimal-slice-ruling.md`](h101-180-phase3-item4-minimal-slice-ruling.md).

### H101-182 — Item 4 E2/D6 acceptance spec (2026-09-22)

H101-181 blocked until this spec lands. **`harnessing run`** subprocess must show
`State:      RecoveryRequired` after the H101-161 crash window (same semantics as
`crash_reconcile_native_test.go`; crash via `harnessing start-run` + fixture sync
line + SIGKILL). Decision table R1–R9. Full spec:
[`h101-182-item4-e2-d6-acceptance-spec.md`](h101-182-item4-e2-d6-acceptance-spec.md).

### H101-183 — Item 4 E2/D6 discharged (2026-09-22)

**E2 SATISFIED. D6 SATISFIED** at `21c782e` (`TestCLI_RecoveryRequiredAfterRealStartRunCrash`,
H101-182 A1–A5, R3–R7). **R8:** both ADR targets — CI SUCCESS on `21c782e` observed
(H101-183 upgrade). **Item 4 still NOT SATISFIED** (D2, D4 open). Ruling:
[`h101-183-phase3-item4-e2-d6-discharge-ruling.md`](h101-183-phase3-item4-e2-d6-discharge-ruling.md).

### H101-184 — Item 4 D2 acceptance spec (2026-09-22)

**Running + dead child → `RecoveryRequired` on next `harnessing run`**, not stuck
`Running` (h101-128 lines 69–71). Reuse participation fixture + PID sync +
controller SIGKILL; **not** `block_after_marker` (E2 window). Full spec:
[`h101-184-item4-d2-acceptance-spec.md`](h101-184-item4-d2-acceptance-spec.md).

### H101-186 — D2 B5 precondition ruling (2026-09-22)

Claudio stop on H101-185 **correct**: written B5 unsatisfiable with today's lock
lifecycle. **B5′** substitute: `participated\npid=` sync before controller
SIGKILL. H101-187 (`4be5966`) affirms conservative Running → `RecoveryRequired`
on reopen; **H101-185 unblocked** (one card). Ruling:
[`h101-186-item4-d2-b5-ruling.md`](h101-186-item4-d2-b5-ruling.md). Architecture:
[`h101-187-running-recovery-ownership.md`](../architecture/h101-187-running-recovery-ownership.md).

### H101-188 — Item 4 D2 discharged (2026-09-22)

**D2 SATISFIED** at `b764691` (`TestCLI_RunningDeadChildReconcilesToRecoveryRequired`,
H101-184 B1–B4 + B5′, S2–S6 + S9). Layer A + reconcile expectation updates **sound**
(H101-187 behaviour, not regression). **R8:** both ADR targets — CI SUCCESS on
`b764691`. **Item 4 still NOT SATISFIED** (D4 open). Ruling:
[`h101-188-phase3-item4-d2-discharge-ruling.md`](h101-188-phase3-item4-d2-discharge-ruling.md).

### H101-189 — Item 4 D4 acceptance spec (2026-09-22)

**Output-gap honesty** — behaviour decided in h101-128 lines 73/79; minimal journal
+ output read slice. Producer: **D2 participate + pid sync** + fixture stdout lines.
Full spec: [`h101-189-item4-d4-acceptance-spec.md`](h101-189-item4-d4-acceptance-spec.md).
Architecture gate: [`h101-190-d4-journal-boundary.md`](../architecture/h101-190-d4-journal-boundary.md).

### H101-192 — Item 4 D4 discharged; item 4 exit (2026-09-22)

**D4 SATISFIED** at `b7d1131`. **PHASE 3 ITEM 4: SATISFIED WITH LIMIT** — D1
evidence-scope limit only (see H101-193 D1 note: not a latent defect). Ruling:
[`h101-192-phase3-item4-d4-discharge-ruling.md`](h101-192-phase3-item4-d4-discharge-ruling.md).

### H101-193 — Item 5 acceptance spec (2026-09-22)

Item 5 engineering blocked until amended spec commits. Beyond D4: graceful **complete**
capture, stdout+stderr channels, offset resume, UI-08 separation, real-not-fake CLI
proof. **Amended H101-199:** serve-owned producer with QA-defined
`serve_release_both_channels` + `HARNESSING_FIXTURE_RELEASE_FILE` (Stanley H101-195
`a61187f`). Withdrawn: one-shot `emit_both_channels_then_exit`. Full spec:
[`h101-193-item5-acceptance-spec.md`](h101-193-item5-acceptance-spec.md).

---

### Item 5 — Real run output journal

Replace Phase 2's fake `ProcessSupervisor`/stream doubles for **verdict** purposes (ADR
0003: fake stream tests are adapter-behavior only, not supervision proof). Phase 3 exit
requires:

- `OutputJournal` persistence with offset-ordered chunks (boundaries port 8).
- Run output readable through the **product command surface** (`harnessing` run-output or
  documented equivalent) **separate from** domain `StateEvents` (UI-08).
- Stream cancellation detaches reader only; stopping run requires `StopRun` (UI-08).

#### Decision table

| # | Observation | Defect class | Fix |
| --- | --- | --- | --- |
| D1 | Output only visible in test mock, not CLI/session | **Product** | Not shipped |
| D2 | Domain event stream contains raw stdout bytes | **Product** | UI-08 separation violated |
| D3 | Missing chunks after graceful run complete | **Product** | Journal durability |
| D4 | Reader cancel stops the run | **Product** | UI-08 detach semantics |
| D5 | Phase 3 exit claimed using Phase 2 fake-stream tests only | **Test/evidence** | Wrong evidence class |

---

### Item 6 — Background hosting survives UI detach

**Ruled H101-143 (2026-09-21):** **NOT SATISFIED** at `18af1ec` — hosting foundation
only (`harnessing serve` + attach/detach); run half blocked on item 1. Mailbox pump
omission is correct scoping, not an item 6 gap. See
[`h101-143-phase3-item6-foundation-ruling.md`](h101-143-phase3-item6-foundation-ruling.md).

Boundaries hosting inference: closing one UI does not stop runs managed by a continuing
host; explicit host shutdown requests termination.

Provable via subprocess test:

1. Start run through CLI in **hold** subprocess (or documented background host mode).
2. Detach/exit a second CLI session (or close observer) without stop command.
3. Run remains observable as `Running`; output continues or resumable.
4. Explicit shutdown/stop still terminates.

#### Decision table

| # | Observation | Defect class | Fix |
| --- | --- | --- | --- |
| D1 | Closing observer CLI kills child run | **Product** | UI detach ≠ stop |
| D2 | No way to observe run after starting CLI exits | **Product** | Background hosting missing |
| D3 | Second workspace writer opened while host lives | **Product** | Lock/ownership breach |
| D4 | Test runs in-process without subprocess | **Test** | Detach semantics not proven |
| D5 | Foundation hosting proved (serve/attach/lock) but no `StartRun` / no `Running` observability | **Progress only** | H101-143 — not item 6 discharge; extend test when item 1 lands |

---

## H101-143 — Item 6 foundation slice (2026-09-21)

**Verdict: NOT SATISFIED.** `18af1ec` lands hosting foundation (`serve`, transport,
attach/detach, lock); does not discharge item 6 until item 1 exists and full
four-step proof passes. Mailbox pump: correct scoping for slice, not item 6 gap;
card under serve completeness separately. Full ruling:
[`h101-143-phase3-item6-foundation-ruling.md`](h101-143-phase3-item6-foundation-ruling.md).

---

### Item 7 — Phase 3 adapter contract (both adapters)

Extend `internal/adaptercontract/` with **Phase 3 scenarios** (stable IDs **UI-09+** to be
mapped by Stanley in ADR 0003 amendment — criteria name behaviors, not implementation).
Minimum scenarios both adapters must pass in CI:

| ID | Behavior |
| --- | --- |
| UI-09 | Approved `StartRun` succeeds through real driver; unapproved profile stably rejected |
| UI-10 | `StopRun` reaches observed terminal run state; no fake-supervisor verdict for this row |
| UI-11 | Budget elapsed path stops run (may share fixture clock with item 3) |
| UI-12 | Run output readable through adapter observation path, detached from domain events |
| UI-13 | Phase 2 **UI-08** rows inverted: `StartRun`/`StopRun`/`SetRunBudget` return **success or documented in-progress**, not `Unsupported` |

Phase 2 UI-01–08 suite stays green. Cross-adapter continuation includes a run started on
adapter A observable on B after rebind.

**Inherited G-limits (H101-107):** item 7 verdict may be **SATISFIED WITH LIMIT** if
G1–G4 or G6–G7 gaps remain on **new** UI-09+ rows only; limits must be named per G-row,
not silently waived.

#### Decision table

| # | Observation | Defect class | Fix |
| --- | --- | --- | --- |
| D1 | CLI passes UI-09–13, throwaway fails | **Product** | Swappability broken |
| D2 | Suite calls `host` commands, not adapter drivers | **Test** | ADR 0003 violation |
| D3 | UI-08 `Unsupported` still required for Phase 3 ops | **Product** | Phase 2/3 boundary error |
| D4 | Only one adapter run | **Test** | Both required |

---

### Item 8 — Trust-boundary honesty (disclosure + spot checks)

**Primary: disclosure.** Phase 3 documentation and in-product copy must state plainly:

1. **CF2 — Manual-start limit persists:** Product-managed start approval applies only to
   runs the product starts after approval. Agents the user started manually before Phase 3
   (or outside `StartRun`) remain unobserved and unvalidated. ADR 0002 revisit trigger
   unchanged — process control is **not** proof that workspace content is safe or that
   historical sender fields were authenticated.
2. **CF3 — No retroactive trust:** Profile approval does not upgrade provenance on
   messages, tasks, or results that existed before approval.
3. **CF1 — Wording:** Never describe registry membership or the adaptercontract
   claimed-engineer fixture as "sender identity handled safely" (Creed boundary review
   item 2 verdict). Impersonation-shaped input is accepted **by design** for routing; it
   is not an authentication control.
4. **Item 2 tree limit (H101-130):** Supported profiles are supervised process groups,
   not sandbox confinement. Escaping/daemonizing profiles are unsupported; deliberate
   escape outside profile contract is outside product scope (CF2 family).

**CI spot-checks (partial):**

- Workspace fixture with pre-approval `SendMessage` record: after first successful
  `StartRun`, `GetMessage` / `harnessing message` still shows `IdentityVerification`
  unverified (or equivalent absent verified label).
- No validation badge/checkmark on manually sourced records (extends Phase 2 UI-05 /
  H101-16 restraint into Phase 3 views).
- Updated disclosure line for Phase 3 scope (replaces or supplements Phase 2
  "does not start…" sentence where product now **does** start approved profiles — must
  not imply all agents are product-managed).

#### Decision table

| # | Observation | Defect class | Fix |
| --- | --- | --- | --- |
| D1 | README claims "all agents are validated at start" | **Docs** | **CF2 violation** |
| D2 | Pre-approval message shows verified after `StartRun` | **Product** | **CF3 violation** |
| D3 | Green checkmark on unverified manual content | **Product/UI** | Validation cue |
| D4 | Marketing text cites adaptercontract fixture as security proof | **Docs** | Creed wording violation |

**Not CI-checkable:** whether the owner understands CF2 in practice (ADR 0002 UNKNOWN).

---

### Item 9 — Runnable local startup guide

**Ruled H101-138 (2026-09-21):** **SATISFIED WITH LIMIT** — coordination guide
accepted at `6d12c48`; Phase 3 process-control supplement pending. See
[`h101-138-phase3-item9-guide-ruling.md`](h101-138-phase3-item9-guide-ruling.md).

**Acceptance procedure, not CI.** Satisfied when:

1. A committed guide (path TBD by docs owner — e.g. `docs/GETTING-STARTED.md` or
   README section) exists on `main`.
2. Guide covers at minimum: clone → build/install → create workspace → register agents →
   complete §2 coordination cycle → **Phase 3**: approve profile → `StartRun` → observe
   output → `StopRun` → budget demonstration.
3. Kelly (or delegate) records a **cold-read acceptance**: an agent follows only the
   written guide on a clean tree and reaches item 1's walkthrough plus one Phase 3 start/stop
   without asking the floor. Failures are guide gaps, not waived criteria.

Converges with owner H101-114 request and `definition.md` §5 ("Documentation starts with
a runnable local workflow"). Ryan's startup-guide card is **implementation**; this item is
the **exit gate** on that deliverable.

#### Decision table

| # | Observation | Defect class | Fix |
| --- | --- | --- | --- |
| D1 | Guide missing Phase 3 steps | **Docs** | Incomplete |
| D2 | Cold-read required human on floor | **Docs** | Gap per H101-120 class |
| D3 | Guide references uncommitted or internal-only paths | **Docs** | Not reproducible |
| D4 | Phase 3 exit claimed without cold-read record | **Process** | Item 9 not discharged |
| D5 | Coordination guide accepted but Phase 3 process-control section not yet runnable | **Expected limit** | H101-138 — supplement + second cold-read when items 1–7 ship |
| D6 | Guide implies Claude Code managed-start was demonstrated when only item 10 attachment applies | **Docs** | H101-196 — deferral + item 10 path |

---

### Item 10 — Registered external agent session (attachment-first)

**Added H101-196 (2026-09-22).** Owner H101-174 chose attachment-first; managed
launch for Claude Code is backlog **B**, not dropped. This item proves **linked
user-started sessions** participate through the product mailbox. It does **not**
discharge item 1 Layer B, item 2–4 managed-run obligations, or item 5 journal.

**Authority:** Stanley
[`h101-171-session-registration-protocol.md`](../architecture/h101-171-session-registration-protocol.md);
acceptance bar:
[`h101-196-item10-registered-session-acceptance-spec.md`](h101-196-item10-registered-session-acceptance-spec.md).

A user-started tool session is **linked** to a registered agent and participation-
profile revision through a **shipped product command** (not shell convention).
After linking:

- Messages to that agent are **deliverable and observable** through product queries
- The session **acknowledges** through an explicit product command — not passive bridge
- Evidence carries `evidenceClass: registered_external_session`,
  `managedStartRunUsed: false`, `preExistingSession: true`

**Two-layer proof:** Layer A = in-repo registration fixture on both ADR targets
(entrypoint **not** `start-run`). Layer B = committed registration runner manifest
(same anti-fraud rules as item 1 H101-153). **First product:** cooperative
check-in (H101-177).

#### Decision table

| # | Observation | Defect class | Fix |
| --- | --- | --- | --- |
| D1 | Registration without shipped CLI | **Evidence** | Product command required |
| D2 | Delivery/ack via `GetSnapshot` bypass | **Test** | CLI/query only |
| D3 | Passive bridge sets `AcknowledgedAt` | **Product** | Explicit worker ack |
| D4 | Manifest omits `managedStartRunUsed: false` | **Evidence** | Hard guard |
| D5 | Item 10 pass used to close item 1 Layer B | **Process** | H101-196 forbidden |
| D6 | Disclosure omits user-paired / unverified-tool limit | **Docs** | Item 8 CF2 cross-ref |
| D9–D19 | Pre-pair MUST disclosure, three recovery actions, never-infer, actionable conflicts | **Product/Test** | H101-203, H101-206 |

**Amended H101-203 / H101-206:** pre-pair disclosure **MUST** (R7); three recovery
actions RECONNECT/REPLACE/DISCONNECT with positive observables (R9–R14); reconnect≠replace
(R10), failed recovery≠free slot (R12), registration removal≠agent retirement (R11).
No timeout-takeover. Full bar:
[`h101-196-item10-registered-session-acceptance-spec.md`](h101-196-item10-registered-session-acceptance-spec.md).

**Standing:** **NOT SATISFIED** — bar only.

---

## H101-138 — Item 9 guide acceptance (2026-09-21)

**Verdict: SATISFIED WITH LIMIT.**

Coordination-cycle guide and H101-137 cold-read **discharged** the owner's
runnable-workflow ask and definition.md §5 for the current product. **Limit:**
Phase 3 `StartRun`/`StopRun`/budget walkthrough not in guide or cold-read until
process control ships; full item 9 / Phase 3 exit requires supplement + second
cold-read.

H101-125 checklist **A16 retired**; **A16'** = empty-inbox `messages` exits 0,
no H101-122 regression. Amend checklist before next formal cold-read; not a
blocker to this ruling. Full ruling:
[`h101-138-phase3-item9-guide-ruling.md`](h101-138-phase3-item9-guide-ruling.md).

---

## H101-130 — Item 3 elapsed budget evidence (2026-09-21)

**Ruling:** Stanley (`h101-128-phase3-supervision.md`) is correct — item 3's original
elapsed wording ("terminates … before or at limit under test clock") demanded a
**zero-latency operating-system exit guarantee** that a controlled clock cannot prove.
That wording is **withdrawn**, not weakened in intent.

**Intent preserved:** hard budget must **actually stop** a product-started run, with
evidence the stop was **budget-caused** and **not** a natural exit. Warning-only or
timer-without-termination remains item 3 D4 (blocker). The one-item budget decision
(elapsed + token in a single item) **stands** — this amendment changes **how elapsed
evidence is structured**, not whether both paths are required.

**Replacement — elapsed path.** When monotonic elapsed time reaches the configured limit
under an injectable/test clock, all four must hold:

| # | Evidence piece | What it proves |
| --- | --- | --- |
| E1 | Durable `Stopping` (or equivalent) with **recorded budget cause** (trigger type, limit, budget revision) committed at or immediately after the limit crossing under controlled clock | Policy fired — distinguishable from natural exit |
| E2 | Termination dispatch enqueued/attempted without waiting for natural child exit | Hard stop initiated, not passive observation |
| E3 | Bounded escalation/observation timeout (Kelly-authored constant in Phase 3 fixture spec before first acceptance run; wall-clock cap ≤30s in CI) | Real-time latency is bounded, not infinite |
| E4 | Native test on each ADR target: owned process group gone (PID/group probe) within E3 | Actual stop happened — not clock theatre |

**Not acceptable:** freezing the test clock during an OS wait and calling that a deadline
guarantee (new decision-table row D8). A naturally short-lived test child cannot
discharge the elapsed path.

**Token path unchanged** in structure: reported cumulative usage `>= limit` triggers the
same termination path as `StopRun`. Overshoot from reporting/termination latency is a
**documented limit** (item 8 disclosure), not a waiver of D2. Telemetry-loss stop is a
separate honest outcome, not silent zero-counting.

**Four-piece shape does not reopen stub risk:** elapsed and token remain coupled in item
3; E1–E4 apply to elapsed only. Token path still requires a real reporting execution
profile and independent native termination proof.

#### Amendment to item 3 decision table (add rows)

| # | Observation | Defect class | Fix |
| --- | --- | --- | --- |
| D8 | Test advances injectable clock but freezes it during OS wait, then claims deadline proof | **Test** | E1 proves ordering; E3+E4 prove latency — do not conflate |
| D9 | `Exited` without budget-cause record while child was still running past limit+observation bound | **Product** | E1/E3 failure |
| D10 | Budget trigger recorded but no termination dispatch before natural exit | **Product** | E2 failure — warning/timer only |

---

## H101-130 — Item 2 tree scope (2026-09-21)

**Ruling:** **SATISFIED WITH LIMIT** when implemented per Stanley's bounded claim.
"Owned process tree" means the **supervised process group** for **supported execution
profiles** on each ADR target — not containment of an arbitrary deliberately escaping
descendant.

**What the product claims (in scope):**

- Approved profiles used with `StartRun` keep descendants in the process group the
  supervisor establishes; profiles that daemonize, double-fork, or detach into an
  unowned session are **unsupported** and rejected **before spawn** (not after promising
  tree termination).
- `StopRun` and budget enforcement use graceful then forced **group** termination on
  supported profiles.
- Native tests include a **parent + worker** fixture, including **parent exits before
  worker** — parent exit alone is insufficient for `Exited` (Stanley item 2 table).

**What the product does not claim (limit, not gap):**

- Confinement of a **deliberately escaping** program that breaks profile contract (same
  family as Creed CF2 manual-start exposure: outside the product's start gate). This is
  **not** a Phase 3 exit blocker; it is an **item 8 disclosure** obligation: supported
  profiles are supervised groups, not a sandbox against hostile code with the user's
  filesystem permissions.

**Creed CF2 consistency:** CF2 says manual-start exposure does not close. An escaping
descendant on an **unsupported or violated profile** is analogous — the product did not
undertake to observe or stop it. An escaping descendant on a **supported profile that
promised group membership** is a **product defect** (item 2 D2/D4), not a CF2 limit.

**Verdict shape:** item 2 may exit **SATISFIED WITH LIMIT** naming this tree-scope
boundary explicitly in item 8 disclosure and supported-profile documentation. It is not
deferred to Phase 4.

#### Amendment to item 2 decision table (add rows)

| # | Observation | Defect class | Fix |
| --- | --- | --- | --- |
| D6 | Supported-profile parent+worker: worker survives after `StopRun` on either ADR target | **Product** | Group termination failure |
| D7 | Profile allows daemonize/detach but `StartRun` promised tree kill | **Product** | Must reject at spawn (unsupported profile) |
| D8 | Docs claim "all descendant processes" without supported-profile boundary | **Docs** | Item 8 / CF2-family overclaim |

---

## Explicitly not Phase 3 exit

(Owner/product/architecture — not quality gates here unless a card promotes them.)

- Agent retirement / task reassignment (Angela product calls; H101-38/H101-46).
- Full egress observation harness (Phase 4 per project plan).
- Windows feature support (remains excluded; H101-103 smoke only).
- Provenance/sender field compile-time guard (H101-117 — architect track).
- Whether "Harnessing 101" is the final product name (`definition.md` UNKNOWN).
- User comprehension of manual-start risk beyond honest disclosure (ADR 0002 UNKNOWN).

---

## Milestone handoff

Phase 3 exit review uses the same milestone procedure as Phase 2
([`definition-of-done.md`](definition-of-done.md) §Milestone review procedure). Kelly
maps each item to CI job names and quoted verify tails at acceptance time — same
discipline as H101-90 (quote tails, not summaries).

Authored by Kelly (QA), H101-126 / H101-115.
