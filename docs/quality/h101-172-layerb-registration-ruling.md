# H101-172 — Item 1 Layer B vs registered external session (2026-09-22)

Kelly QA ruling. Authority: owner H101-163 answer (verbatim in god dispatch
`2026-09-22T10-19-34-919Z-2bf048`), existing item 1 rows D13–D15 and H101-147.

Stanley H101-171 (participation protocol spec) is **not written yet**; this
ruling holds on the owner's words and the committed criterion text. Evidence
shapes below are **provisional** where they depend on Stanley's command and
transport binding.

---

## Owner intent (verbatim)

> I want each worker/agent session to be linked to a profile on our side
> identifying which agent is what. for example. if the user has they're own open
> session with claude code, they could register that claude code session and it
> would be receiving messages based on the message protocol we have

---

## (1) Does registration discharge item 1 Layer B?

**No. It falls outside item 1 Layer B — not a discharge, not a failure of the
registration idea itself.**

Item 1 is **`StartRun` through the shipped command** with Layer B manifest
evidence of **managed start through the product** (phase3-exit-criteria.md
lines 97–108, 166–167). D13 names the exclusion explicitly: Layer B claimed from
**manual shell start or spike without managed `StartRun`** is an evidence defect.

A Claude Code session the user already has open, then **registers**, is a
**pre-existing / user-started session**. It is the same trust class H101-147
listed under "What does not count": manual shell start of an agentic CLI. Linking
it to a profile does not retroactively make it a `StartRun` outcome.

**Consequence:** building registration can produce a genuinely participating
Claude Code worker and still leave **item 1 Layer B for managed `StartRun`
unsatisfied** on Claude Code (unless closed by a separate path — see below).

**Paths that still close item 1's Claude Layer B slot without registration:**

1. **Managed `StartRun` runner manifest** — existing bar; `cycle_completed` or
   honest typed failure (`authentication_required`, `network_egress_refused`,
   etc.) from `scripts/run-agentic-cli-cycle.py` output.
2. **Owner deferral in descriptor** — existing H101-153 mechanism
   (`deferred: true` in `scripts/agentic-cli-manifests/claude-code.json`), same
   as Codex/Cursor today.

Registration is **not** a third way to discharge item 1 Layer B under the
current criterion.

---

## (2) Amend item 1, or new exit item?

**Recommend: new exit item — do not amend item 1 Layer B to absorb registration.**

| Option | Verdict |
| --- | --- |
| Amend item 1 Layer B to accept registration as Claude evidence | **Reject** — erases D13, contradicts H101-147 "manual shell start does not count", and conflates product-managed start (CF2/CF3 surface) with user-started sessions (ADR 0002 exposure class) |
| New exit item with its own evidence bar | **Accept** — owner's model is a distinct capability: **registered external agent session** linked to profile + registry agent, receiving mailbox traffic through the documented protocol |
| Defer Claude in item 1 while building registration elsewhere | **Accept as interim** — honest if owner accepts bounded compatibility claim; does not deliver registration |

**Proposed new item (working title, Phase 3 item 10 or owner-priority insert):**
**Registered external agent session participates through the product mailbox.**

Scope (criteria owner draft — Stanley shapes commands):

- A user-started tool session is **linked** to a registered agent + approved
  profile through a **shipped product command** (not a shell-side convention).
- After linking, a message addressed to that agent is **deliverable and
  observable** through the product surface (`harnessing message` / `messages`),
  and the session can **acknowledge** through the same protocol the Phase 2
  cycle already proved.
- **Does not** satisfy item 1 `StartRun`, item 2 tree kill, or item 3 budget —
  those remain managed-run obligations.

Item 1 stays the **managed-start** proof. The new item is the **linked-session**
proof. Both may use Claude Code; they are not interchangeable evidence.

---

## (3) Evidence Kelly would accept for a registered session

Stated for engineers; **finalize after H101-171** names the registration command,
handoff transport, and session identity record.

### Layer B analogue (manifest / disclosure, not CI)

A committed **registration runner** (may extend
`scripts/run-agentic-cli-cycle.py` or sibling script) produces JSON manifest
output — same anti-fraud rules as item 1 (D14/D15: runner output only, not
hand-edited).

**Manifest must show, per ADR target attempted:**

| Field | Requirement |
| --- | --- |
| `evidenceClass` | `registered_external_session` (distinct from `managed_start_run`) |
| `toolId` | e.g. `claude-code` from descriptor |
| `registrationCommand` | Exact shipped CLI invocation used to link session (argv recorded) |
| `preExistingSession` | `true` — runner declares user/session existed before registration |
| `managedStartRunUsed` | `false` — explicit negative guard for D13 |
| `profileId` / `agentId` | Match workspace records after registration |
| `deliveryObserved` | Product-delivered message reached session per protocol (Stanley defines observable) |
| `ackObserved` | Acknowledgement visible via product query, not host snapshot bypass |
| `outcome` | `registration_cycle_completed` or typed honest failure |

**Does not count:**

- Registering then only proving the tool was on `PATH`
- Mailbox traffic copied manually between terminals
- `GetSnapshot` / host bypass for delivery or ack proof
- Reusing the StartRun runner's `cycle_completed` schema for a registration path

### Layer A analogue (CI, both ADR targets)

In-repo **registration fixture**: a test binary or script standing in for an
external session, registered through the shipped CLI, receives injected protocol
traffic, emits observable participation marker, acks through product commands.
Same discipline as `TestCLI_StartRun_LayerAParticipationFixture` but
**registration entrypoint, not `start-run`**.

Kelly cannot name the CI test until H101-171 fixes the registration command and
wire format.

### CF2 disclosure coupling

Any registered-session item must **cross-reference item 8**: registered external
sessions are **not** product-managed starts; manual-start exposure limits from
ADR 0002 apply to them separately from `StartRun` approval.

---

## Stanley input (H101-171, god inform 2026-09-22)

Independent alignment: **"manual attachment cannot be relabelled managed-start
evidence"** — same conclusion as D13 above. Owner wording does not remove
managed `StartRun`; both entry modes may coexist architecturally. Owner choice
**H101-174** (attachment first with managed-launch deferred vs both in scope)
needs Kelly gate disposition on whichever path ships — this ruling holds either
way: registration evidence stays **outside** item 1 Layer B.

**Honest ceiling for registered-session evidence:** a local join handshake proves
**possession of a session credential**, not cryptographic identity of the tool
process. Any future registered-session exit item must state that limit in item 8
disclosure; acceptance tests must not claim stronger proof.

---

## Blocked on Stanley (H101-171)

Before Kelly can freeze acceptance tests:

1. Registration **command name** and persisted **session identity** shape
2. How mailbox delivery reaches an **already-running** process (transport —
   file, socket, stdio attach — per H101-131 class)
3. Whether one agent may hold **both** a managed run and a registered session,
   or mutual exclusion rule

---

## Item 1 standing unchanged

At `bf8d5b9`: item 1 **NOT SATISFIED**. Layer B Claude remains the sole item 1
blocker **unless** owner chooses descriptor deferral. Registration work does not
move item 1 on its own.

Authored by Kelly (QA), H101-172.
