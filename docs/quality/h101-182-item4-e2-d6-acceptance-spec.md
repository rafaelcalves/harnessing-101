# H101-182 — Item 4 E2/D6 acceptance spec (2026-09-22)

Kelly QA. **Spec only** — unblocks H101-181. Closes Stanley H101-169 evidence **E2**
and item 4 row **D6** only. Does **not** cover D2 (Running with dead child) or D4
(output gaps); those remain later per H101-180 order.

**Authority:** H101-180 partial acceptance at `2b76d80`; engine slice already proves
reconcile + `GetRun` → `RecoveryRequired`. This spec is what a **user** must see.

---

## What E2 means here

> Durable `RecoveryRequired` visible **through the product**

Engine-level `GetRun` after `ReconcileStuckRuns` is **already satisfied** (H101-170).
E2 discharges only when a **fresh `harnessing` subprocess** — no `host.Open`, no
`engine.GetRun` in the test — observes the reconciled state through the **shipped
read path** the user has.

**The user read path for runs:** `harnessing run` → `FrontendSession.GetRun` →
`printRun` (`cmd/harnessing/run_commands.go`).

---

## Passing stdout shape (named assertions)

Test MUST build a real `harnessing` binary and invoke **`harnessing run`** as a
subprocess (same class as `TestCLI_StartRun_LayerAParticipationFixture`).

After the crash window (below) and a **second** CLI open (new process, same
workspace directory):

| # | Assertion | Rationale |
| --- | --- | --- |
| A1 | Subprocess exit code **0** | Read-only query succeeds; `RecoveryRequired` is honest state, not an error return |
| A2 | stdout contains **`State:      RecoveryRequired`** exactly | `printRun` format; domain `RunRecoveryRequired` string; same spacing pattern as existing `State:      Running` / `State:      Exited` assertions in `start_run_test.go` |
| A3 | stdout contains **`Run <runID>`** for the crashed run | Proves query is scoped to the right run, not a stale default |
| A4 | stdout does **not** contain **`State:      Starting`** for that run after A2 passes | Starting-after-reopen without reconcile is the H101-161 failure mode |
| A5 | (Regression guard, same test or sibling) second **`harnessing start-run`** for the **same agent** (new run/request/profile IDs): exit **non-zero**, stderr contains **`Conflict`**, no child spawned | User-visible refusal; pairs with E2 — state is visible *and* acted on |

**Not sufficient for E2/D6:**

- Calling `engine.GetRun`, `host.Open`, or `Capabilities.GetRun` in the test
- Parsing workspace JSON under `runs/` directly
- Asserting only on events/journal without `harnessing run`

---

## Crash producer (reuse H101-170 window)

The crash MUST reproduce **H101-161's occurrence**: controller dies **after** the
dispatch-attempted marker is durable, **before** the StartRun observation commit
(Start never returns → run stays `Starting` in store until reconcile).

**Authoritative engine reference:** `TestReconcileStuckRuns_NativeCrashBetweenMarkerAndObservation`
(`internal/core/task/crash_reconcile_native_test.go` at `2b76d80`):
SIGKILL on a real subprocess **strictly after** it prints `start-called` (sync
proof marker committed), **strictly before** Start returns.

### For the CLI proof — same window, product subprocess

The E2/D6 test MUST use the **same semantic window**, reached through **`harnessing
start-run`**, not engine injection:

1. **Preferred (single card):** extend the participation fixture (or approved
   profile executable) with a simulate mode — e.g.
   `HARNESSING_FIXTURE_SIMULATE=block_after_marker` — that prints a **single
   sync line to stdout** (convention: `start-called\n`, matching the engine test's
   proof line) then blocks until killed. Parent test:
   - spawns `harnessing start-run` subprocess with that profile;
   - waits for sync line on child stdout;
   - **SIGKILL** the `harnessing` subprocess (unclean, no `Close`);
   - spawns fresh `harnessing run` → asserts A1–A4.

2. **Do not** reuse the engine test's `HARNESSING_RECONCILE_HELPER` re-exec path
   as the *only* crash evidence — that proves reconcile without the CLI surface
   (D6 violation). It **may** be shared library code if the CLI test still drives
   crash via `harnessing start-run`.

3. **Do not** invent a weaker crash (kill before marker, kill after graceful
   `Close`, or panic inside test without a real CLI holder).

**Platform:** native on **both** ADR targets (`linux/amd64`, `darwin/arm64`); same
`runtime.GOOS` skip pattern as other CLI subprocess tests. No `t.Skip` on one CI
target.

---

## Decision table (read before implementation)

| # | Observation | Defect class | Fix |
| --- | --- | --- | --- |
| R1 | Test asserts run state via `engine.GetRun`, `host.Open`, or store file read | **Test** | **D6** — not product surface |
| R2 | Test asserts via `GetSnapshot` | **Test** | H101-104 D8 class |
| R3 | `harnessing run` exit 0 but `State:      Starting` after crash + reopen | **Product** | Reconcile not run on `host.Open`, or not durable |
| R4 | `harnessing run` shows `Running` or `Exited` instead of `RecoveryRequired` | **Product** | State honesty / wrong reconcile |
| R5 | `State:` line missing or wrong label spacing | **Product** | CLI render regression |
| R6 | Crash subprocess killed before dispatch marker durable | **Test** | Wrong window — does not prove H101-161 class |
| R7 | Crash after graceful workspace `Close` | **Test** | Item 4 D5 — not unclean exit |
| R8 | Pass on one CI target only via `t.Skip` | **Test** | Both ADR targets required |
| R9 | E2 claimed from engine-only evidence while this test is red/skipped | **Evidence** | E2 not discharged |

---

## Scope sizing (for dispatch)

**One engineering card** if the fixture simulate mode (`block_after_marker` + sync
line) is sufficient to block a real `harnessing start-run` child at the marker
window. Expected touch: `cmd/harnessing/*_test.go` (new `TestCLI_*`), small fixture
hook in `internal/adapters/process` or existing `__fixture-participate` path, no
new exit criterion.

**Split to two cards** only if implementing the sync line requires a new product
command or changes `StartRun` observability beyond fixture/env — stop and report
rather than expanding H101-181 silently.

---

## What this does not discharge

Unchanged from H101-180: item 4 **NOT SATISFIED**; D2, D4, full E1–E5 matrix
beyond E2, operation-query path, unblock/resolution UI (H101-173), CLI proof for
`harnessing operation` on `OperationRecoveryRequired` (optional follow-on, not
required for H101-181).

Authored by Kelly (QA), H101-182.
