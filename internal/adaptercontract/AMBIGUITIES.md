# Adaptercontract spec ambiguities (H101-90 / H101-92)

These are places where ADR 0003, boundaries.md, or H101-53 name behaviour the
suite must assert, but the written spec does not pin down enough for a literal
in `expected/` without an explicit assumption. **Stanley (H101-91) reviews
whether each assumption is justified by his documents or smuggled from adapter
output.**

Nothing in this file reads adapter output. Each entry states: what the spec
says, what it fails to say, and what `expected/` currently assumes instead.

---

## A1 — UI-03 “generated IDs must be surfaced”

**What the spec says**

- ADR 0003 UI-03: *“Generated IDs must be surfaced so an interrupted caller can
  recover them.”*
- ADR 0003 UI-03 (same row): caller-supplied request ID on retry; changed
  payload conflicts.
- Shipped CLI (`cmd/harnessing/run.go` usage): every write command requires
  `-request-id`; entity IDs (`-task`, `-message`, `-agent`, `-result`) are also
  caller-supplied flags, not allocated by the tool.

**What the spec fails to say**

- Whether “generated IDs” includes entity IDs when the adapter never generates
  them — only the caller names them on the command line.
- What output surface counts as “surfaced” (stdout receipt line, stderr, a
  dedicated field, or durable state query).
- Whether surfacing request ID + workspace revision on the success receipt line
  (`printReceipt`) satisfies UI-03 for this CLI shape.

**What `expected/` assumes instead**

- `UI03IdempotentRegister` and `TestUI03_*` assert idempotent replay returns
  the **same receipt** (request ID and revision match) and workspace agent count
  stays 1. Retry identity is the **request ID**, not the revision number.
- **Ruling (Angela/Stanley, 2026-09-21):** receipt revision is not retry
  identity; a success echo alone cannot prove recovery of a lost response.
- Entity IDs are **not** asserted as “surfaced on success” because the spec
  does not define that requirement for caller-supplied IDs; the interrupted
  caller already holds them from argv / JSON payload.
- **Risk if wrong:** stage 1 could pass while UI-03’s rendering obligation for
  auto-generated IDs remains untested — relevant only if a future adapter
  allocates entity IDs without echoing them.

---

## A2 — UI-02 IOFailure / OutcomeUncertain uncertainty rendering

**What the spec says**

- ADR 0003 UI-02: durable receipt vs completed work distinct; stable domain
  error codes; never map Denied/Conflict/IOFailure to success.
- ADR 0003 UI-02: IOFailure may mean commit applied but durability uncertain —
  retain request ID and describe uncertainty, not “nothing happened.”
- H101-53: contract may assert `IOFailure` + detail substring on injected
  fsync fault, or wait for `OutcomeUncertain` propagation (H101-64).
- `cmd/harnessing/flags.go`: `printCommandError` branches on
  `ErrOutcomeUncertain` and conservative `ErrIOFailure` fallback.

**What the spec fails to say**

- Whether adaptercontract stage 1 must include a fault-injection scenario, or
  whether Denied/Conflict coverage is sufficient partial discharge.
- Which stable field to assert (`ErrorCode` only vs `Effect`/`Confirmation` vs
  stderr substring).
- Whether read-only commands (`task`, `message`) are in scope for IOFailure
  rendering (they use a different error path per `flags.go`).

**What `expected/` assumes instead**

- Stage 1 asserts **Denied** and **Conflict** stable codes on write paths only
  (`TestUI02_DeniedAndConflictStableCodes`); receipt-vs-status separation on
  success (`TestUI02_ReceiptDistinctFromTaskStatus`).
- **No IOFailure / OutcomeUncertain literal** yet — deferred until fault
  injection mechanism and due stage are specified.
- **Ruling (Angela/Stanley, 2026-09-21):** Denied + Conflict is **partial
  coverage, not discharge** of the uncertainty path. **ADR 0004
  `OutcomeUncertain`** governs uncertain mutations — assert **structured
  `Effect` / `Confirmation`**, not a detail substring. If stage 2 needs fault
  injection, record the gap and ask god rather than inventing a mechanism.

---

## A3 — UI-04 workspace revision (RESOLVED)

**Status:** **Resolved** — `boundaries.md` H101-94 (`d372518`, 2026-09-21).

**Spec now says**

- Workspace revision starts at **0**; each **atomic commit group** advances it
  once (not per field or event).
- Delivery publication and delivery ingestion are **separate groups** from
  `SendMessage` and from explicit acknowledgement.
- Same-ID replay, confirmation, read, and failure add no group. Denied accept
  contributes zero. Successful business no-op under a **new** request ID still
  advances revision once (repeat acknowledgement case).
- **Equivalent committed cut:** same named groups and terminal facts, mailbox
  recording complete, no unrelated requests before snapshot. Receipt revision
  may precede final snapshot revision.
- **UI-04 final cut:** 13 user-request groups + 2 delivery-record groups =
  **revision 15** (derived from declared groups, not observed output).
  **Amended per H101-109/H101-23/H101-111 (Kelly, ruling (a)):** H101-109
  requires `SendMessage`'s `SenderAgentID` to be a registered agent; the
  cycle's claimed sender (`claimed-engineer`, deliberately distinct from
  the caller to prove the claim's independence) is now a third
  registration, one more group than the original H101-94 cut counted.
  This is arithmetic following from a legitimate engine change, not a
  new architectural claim about what a "group" is.

**What `expected/` now asserts**

- `UI04CycleEnd.WorkspaceRevision = 15`, `AgentCount = 3` (engineer,
  analyst, claimed-engineer), with task terminal state at that cut.
- Cross-adapter comparison uses harness snapshot after full cycle (both adapters
  drive mailbox via assembly on each invocation). Both adapters now send as
  `claimed-engineer` rather than the throwaway path uniquely setting
  `SenderAgentID == caller` — H101-111's cross-adapter parity requirement.
