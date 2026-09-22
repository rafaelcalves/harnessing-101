# H101-196 — Item 10 acceptance spec: registered external agent session (2026-09-22)

Kelly QA. **Bar only** — blocks item 10 engineering dispatch.

**Amended H101-203 (2026-09-22):** Creed H101-197 clean review — pre-pair disclosure
**MUST** (testable); stuck-slot observability + documented user recovery **required**;
availability cost of no timeout-takeover **not** a construction gate. Never-infer
vocabulary added.

**Amended H101-206 (2026-09-22):** Angela H101-204 — three distinct recovery actions
(RECONNECT / REPLACE / DISCONNECT) with positive observables; reconnect≠replace and
failed-recovery≠free-slot guards testable; connection-attention ≠ task Blocked;
actionable conflict errors; staleness cites **last connector contact**.

**Authority:** owner H101-163/H101-174; Stanley
[`h101-171-session-registration-protocol.md`](../architecture/h101-171-session-registration-protocol.md);
Kelly H101-172/H101-196; Angela H101-177/H101-204 (cooperative check-in, recovery
actions — user-facing names, not CLI command names); Creed H101-197.

**Single item.** Managed launch for Claude Code remains backlog **B** (H101-174);
this item is the **attachment-first** proof. It does **not** discharge item 1 Layer B,
item 2 tree kill, item 3 budget, or item 4 recovery.

---

## What item 10 means

A **user-started** agentic CLI session is **linked** to a registered agent and
participation-profile revision through a **shipped product command**, then
**participates** through the product mailbox: delivery observable, acknowledgement
explicit, no managed `RunID` manufactured.

**Preserved:** `managedStartRunUsed: false` on all registration evidence.
Registration ≠ `StartRun` (D13).

---

## Two-layer proof model

### Layer A — CI (both ADR targets, every merge)

In-repo **registration fixture** (not `start-run`):

| # | Requirement |
| --- | --- |
| A1 | Stand-in external session opened **before** registration command |
| A2 | Registration through **shipped CLI** (subprocess, same surface discipline as item 1 D6) |
| A3 | `ParticipationSessionID` (or documented equivalent) committed; `preExistingSession: true` in test log |
| A4 | Injected mailbox message **delivered** to bound agent; **not** acknowledged by passive bridge alone |
| A5 | Fixture issues **explicit** acknowledgement through product command; `AcknowledgedAt` set |
| A6 | `harnessing` query path shows delivery + ack — **no** `host.GetSnapshot` bypass (item 4 D6 class) |
| A7 | Negative: expired/wrong invitation → `Denied`/`Conflict`; no silent bind |
| A8 | Negative: second session for occupied agent → `Conflict` |
| A9 | Native on `linux/amd64` and `darwin/arm64` — no `t.Skip` discharge |

**Named test home (engineer):** `TestCLI_RegisterParticipation_LayerAFixture` or
equivalent in `cmd/harnessing/` — registration entrypoint, **not**
`TestCLI_StartRun_LayerAParticipationFixture`.

### Layer B — registration runner (manifest / disclosure, not CI network)

Committed **registration runner** (sibling to `scripts/run-agentic-cli-cycle.py`)
observes a **real** pre-opened supported tool session after deliberate user pairing.

**Manifest required fields:**

| Field | Requirement |
| --- | --- |
| `evidenceClass` | `registered_external_session` |
| `managedStartRunUsed` | `false` |
| `preExistingSession` | `true` |
| `toolId` | e.g. `claude-code` |
| `registrationCommand` | Exact shipped CLI argv |
| `participationProfileId` / `agentId` | Match workspace after pairing |
| `deliveryObserved` | Product-delivered message per protocol |
| `ackObserved` | Explicit ack via product query |
| `outcome` | `registration_cycle_completed` or typed honest failure |

**Does not count:** hand-edited manifest; reusing StartRun `cycle_completed` schema;
`GetSnapshot` bypass; proving only `PATH` presence.

**First product scope (H101-177):** cooperative **check-in** — worker inspects
pending messages at turn boundaries; no promise of idle wake.

---

## Passing assertions (Layer A — named)

After registration fixture run via fresh CLI subprocesses:

| # | Assertion |
| --- | --- |
| R1 | Registration command exit **0** with stable success receipt |
| R2 | `harnessing messages` (or shipped list command) shows injected message for bound `agentId` |
| R3 | Before ack: message **not** `AcknowledgedAt` |
| R4 | After fixture ack command: message **is** `AcknowledgedAt` |
| R5 | No `RunID` created for registration path (query run list / documented negative) |
| R6 | `managedStartRunUsed` guard: no `StartRun` receipt in registration test log |
| R7 | **Pre-pair disclosure (MUST):** before pairing redemption succeeds, shipped pairing preview/invitation surface (CLI or documented equivalent) emits **all** required disclosure lines below — not aspirational |
| R8 | **Stuck-slot observability:** when registration is `Stale` or `Disconnected`, shipped query shows that state, names the blocking slot, and cites **`last connector contact`** (exact substring) — **not** last-agent-activity wording |
| R9 | **Three recovery actions** (Angela H101-204 — distinct product meanings, each with shipped CLI mapped in guide; user-facing labels below are criteria vocabulary, not command names) |
| R10 | **RECONNECT ≠ REPLACE (positive):** see [R9 scenarios](#r9--three-recovery-actions-angela-h101-204) — same `ParticipationSessionID` after reconnect; new ID only after explicit replace |
| R11 | **Registration removal ≠ agent retirement (positive):** after REPLACE or DISCONNECT, `agentId` still listed; assigned tasks/messages unchanged; agent record not deleted |
| R12 | **Failed recovery ≠ free slot (positive):** failed recovery exits non-zero; query still `Stale`/`Disconnected` with slot occupied; error names remaining condition **and** lists `RECONNECT` / `REPLACE` / `DISCONNECT` choices |
| R13 | **Connection attention ≠ task Blocked:** task stays `Doing` (or prior non-Blocked state) when registration becomes `Stale` — connection loss alone does not flip task to `Blocked` |
| R14 | **Actionable slot conflict:** conflicting pair or managed-start attempt returns stable error naming occupied registration state **and** lists all three recovery choices — not bare `slot occupied` |

### R7 — required pre-pair disclosure lines (MUST)

Security-load-bearing copy is **not** `should` (Creed H101-46 class). Criteria bind even
if h101-171 line 101 still says `should`; **criteria route sufficient** — recommend
Stanley align spec text to `must` on next touch, not a construction blocker.

Passing test asserts these **exact substrings** on the pre-pair surface (stdout, stderr,
or named JSON field — engineer documents which):

| Token | Meaning |
| --- | --- |
| `user-paired` | Pairing is user-authorized, not verified tool identity |
| `tool identity unverified` | No cryptographic tool attestation |
| `not product-managed start` | Registration ≠ `StartRun` / process control |
| `worker capability` | Grant is narrow participation, not admin |
| `no reviewer authority` | Worker cannot accept/reject as human |
| `no delegation` | Worker cannot widen authority |

**Not sufficient:** disclosure only in external docs; post-pair-only copy; UI hide without
CLI-testable surface.

### R8 — stuck slot observability

Creed H101-197 availability tradeoff **not** gated (no timeout takeover). User must
**see** stuck state (R8) — never silent indefinite block. Staleness reason must cite
connector contact only (Angela H101-204).

### R9 — three recovery actions (Angela H101-204)

Three user-facing actions with **different effects** — tests must prove the distinction,
not collapse them into one "replace/revoke".

| Action | Meaning | Passing positive observables (Layer A fixture) |
| --- | --- | --- |
| **RECONNECT** | Restore **same** conversation through required revalidation | After `Stale`: reconnect succeeds (exit 0); query shows `Connected`; **`ParticipationSessionID` unchanged**; pending message still deliverable; fixture ack with **same** binding succeeds |
| **REPLACE** | Bind a **different** conversation; old session loses workspace-mutation permission; does **not** stop old external program; work/messages/history stay with **agent** | Replace succeeds; query shows **new** `ParticipationSessionID` ≠ prior; **prior binding cannot authorize mutations** (stable `Denied`); **agent + tasks + messages unchanged**; fixture records old external PID **still running** (test marker file) |
| **DISCONNECT** | Close registration **without** replacement; releases reservation; does **not** stop external tool; does **not** clear separate managed-run recovery restriction | Disconnect succeeds; registration `Closed`; slot admits new pairing; **agent still listed**; external PID marker **still running**; if managed run `RecoveryRequired` on same agent, `StartRun` still blocked per H101-171 dual-admission |

**No timeout, auto-release, or takeover** in criteria or tests (Angela + Creed preserved).

#### R10 — reconnect must not silently mean replace

Negative guard needs **positive** observables (item 5 lesson):

1. **Reconnect path (R10a):** `ParticipationSessionID` **equals** pre-stale value `S1`.
2. **Replace path (R10b):** `ParticipationSessionID` **not equal** `S1`; old binding denied.

Pass requires **both** scenarios in CI (may be subtests). Reconnect that mints a new
session ID without user choosing replace → **D14**.

#### R11 — removing registration must not retire agent

After REPLACE or DISCONNECT: `agentId` query returns same record; at least one
pre-existing task/message row unchanged. Agent delete or hide → **D15**.

#### R12 — failed recovery must not imply slot is free

Induce failed reconnect (invalid credential / wrong generation fixture). Assert **all**:

| Observable | Requirement |
| --- | --- |
| Exit code | **Non-zero** |
| Registration query | Still `Stale` or `Disconnected`; slot **occupied** |
| Error surface | Names remaining condition + contains all three tokens: `RECONNECT`, `REPLACE`, `DISCONNECT` |
| Follow-on conflict | Fresh pair attempt still `Conflict` with actionable error (R14), **not** admitted as if slot were free |

#### R13 — connection attention ≠ task Blocked

Fixture: task in `Doing`, registration goes `Stale`. Task query still `Doing` (not
`Blocked`) until a **separate** user blocker action. Auto-Blocked on staleness alone → **D17**.

#### R14 — actionable slot conflict

On occupied-slot pair or managed-start conflict, error body (CLI stderr or named JSON
field) contains: registration state label + all three recovery choice tokens. Bare
`slot occupied` without choices → **D18**.

Layer B adds manifest file + runner stdout tails per H101-153 anti-fraud rules.

---

## Decision table

| # | Observation | Defect class | Fix |
| --- | --- | --- | --- |
| D1 | Registration claimed from manual shell convention without shipped CLI | **Evidence** | Product command required |
| D2 | Delivery proved via host snapshot / file peek bypass | **Test** | CLI/query surface only |
| D3 | Passive bridge or poll sets `AcknowledgedAt` | **Product** | Explicit worker ack only (H101-171) |
| D4 | Manifest `managedStartRunUsed: true` or omits field | **Evidence** | Hard negative guard |
| D5 | Layer A uses `start-run` entrypoint | **Test** | Registration fixture only |
| D6 | Registration manifest hand-written | **Evidence** | Runner output only (H101-153 D14) |
| D7 | Item 10 pass used to discharge item 1 Layer B | **Process** | H101-196 forbidden reading |
| D8 | Disclosure omits user-paired / tool-unverified limit | **Docs** | Item 8 CF2 cross-ref |
| D9 | Pre-pair disclosure missing any R7 token or appears only after pairing | **Product** | Creed H101-203 — security-load-bearing MUST |
| D10 | `Stale`/`Disconnected` slot blocks admission but query shows Connected or silent | **Product** | R8 observability |
| D11 | Recovery collapses RECONNECT/REPLACE/DISCONNECT into one undifferentiated action | **Product** | R9 — Angela H101-204 |
| D12 | Test or product uses timeout-based slot takeover | **Product** | H101-171 forbidden shortcut |
| D14 | Reconnect mints new `ParticipationSessionID` without explicit replace | **Product** | R10 — silent replace |
| D15 | REPLACE/DISCONNECT deletes or hides agent record | **Product** | R11 |
| D16 | Failed recovery returns success or query shows slot free / `Closed` without user disconnect | **Product** | R12 |
| D17 | Registration `Stale` alone flips task to `Blocked` | **Product** | R13 — Angela H101-204 |
| D18 | Slot conflict error is bare `slot occupied` without recovery choices | **Product** | R14 |
| D19 | Staleness reason cites last agent activity / model attention | **Product** | R8 — must be last connector contact |

---

## Never-infer vocabulary (Creed H101-197 — confirmed)

Item 10 evidence and product copy must **never** infer:

| Never infer | From |
| --- | --- |
| Tool identity | Pairing credential alone |
| Liveness or model attention | Connected session / heartbeat |
| Participation or ack | Live connector, queued delivery, successful poll |
| Death or auto-release | Heartbeat staleness |
| Managed-run resolution | Registration alone |

Tests that treat any column right as proof of the column left are **defect class Test**
(new row **D13**).

| D13 | Evidence conflates pairing credential with verified tool identity, or poll/delivery with ack/participation | **Test** | Never-infer table |

---

## Item 8 coupling (required disclosure)

Registered external sessions are **not** product-managed starts. Copy must state:

- User-paired; tool identity **unverified** (H101-171 proof limit)
- Same-user credential theft boundary (ADR 0002 family)
- Registration does not approve execution profile or imply `StartRun` consent

---

## Out of scope (explicit)

| Not item 10 | Where |
| --- | --- |
| Managed `StartRun` manifest | Item 1 Layer B (deferred for Claude — backlog B) |
| Process tree stop | Item 2 |
| Budget enforcement | Item 3 |
| Crash recovery / `RecoveryRequired` | Item 4 |
| Output journal graceful complete | Item 5 |
| Automatic idle wake | Future material requirement change (H101-171) |

---

## Standing

**NOT SATISFIED** — bar only. Dispatch blocked until god commits this spec and
Stanley/security gates clear construction per H101-171.

Authored by Kelly (QA), H101-196. Amended H101-203, H101-206.
