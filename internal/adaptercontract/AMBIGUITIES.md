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

- `UI03IdempotentRegister` and `TestUI03_*` treat **request ID + committed
  workspace revision** on the receipt as the recoverable identity for retry
  (idempotent replay returns the same receipt; workspace agent count stays 1).
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
- **No IOFailure / OutcomeUncertain literal** — deferred to stage 2 or a
  dedicated injected-fault helper; no expected string copied from CLI stderr.
- **Risk if wrong:** adapters could regress uncertainty messaging while stage 1
  stays green; H101-53 already flags this as follow-up, not stage-1 blocker.

---

## A3 — UI-04 workspace revision vs mailbox background commits

**What the spec says**

- ADR 0003 UI-04: full interaction cycle with **identical domain outcomes**
  (registry, task, blocker, message, ack, report, reject, accept).
- Product definition §2 and item 1 walkthrough: same cycle, durable records.
- Domain rule (boundaries): each successful user command advances workspace
  revision by one.
- H101-85 / item 5: `assembly.WithSession` runs `driveMailbox` after every
  command; mailbox may call `RecordMessagePublished` / `RecordMessageProcessed`.

**What the spec fails to say**

- Whether “domain outcome” includes the **numeric workspace revision** after a
  cycle that includes `send` + `ack`, when mailbox commits are not user commands
  but do advance revision.
- How adaptercontract should count commits when comparing to a naive
  “one revision per user command” tally (12 user writes in UI-04 cycle).

**What `expected/` assumes instead**

- `UI04CycleEnd` asserts **task domain fields only**: status `Done`, title,
  assignee, `CurrentResultID` = `res2`, agent count 2 (engineer + analyst).
- **Workspace revision is intentionally omitted** from the fixture. A naive
  count of user commands alone implies revision 12; observed revision after
  full cycle with mailbox wiring is **14** (+2 from mailbox facts after send/ack
  path). Asserting 12 would bake in an unwritten spec rule or require reading
  adapter/engine output to learn the offset.
- **Risk if wrong:** item 3 could require revision equality as part of
  cross-adapter comparison; if so, spec must state whether mailbox commits count
  and what the expected total is.
