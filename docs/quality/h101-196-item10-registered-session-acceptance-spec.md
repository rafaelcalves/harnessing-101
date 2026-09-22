# H101-196 — Item 10 acceptance spec: registered external agent session (2026-09-22)

Kelly QA. **Bar only** — blocks item 10 engineering dispatch.

**Authority:** owner H101-163/H101-174; Stanley
[`h101-171-session-registration-protocol.md`](../architecture/h101-171-session-registration-protocol.md);
Kelly H101-172/H101-196; Angela H101-177 (cooperative check-in first product,
participation profile).

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

Authored by Kelly (QA), H101-196.
