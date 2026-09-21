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

## Phase 3 is not accepted until all nine items pass

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

**Count: 9 items.** Seven are CI-checkable on `main`. Item 8 is primarily disclosure with
named CI spot-checks. Item 9 is satisfied by a documented cold-read acceptance procedure
(owner-requested; converges with `definition.md` §5 and H101-120 finding).

**Budget enforcement shape:** **one exit item** (item 3) with **two named failure-mode
rows** (elapsed-time stop vs reported-token stop). Both must actually **stop** a run;
splitting into two exit items would let one pass while the other is stubbed. The decision
table keeps them coupled.

---

### Item 1 — Approved-profile `StartRun`

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

### Item 2 — `StopRun` and owned process-tree termination

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
