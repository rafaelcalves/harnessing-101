# H101-193 — Item 5 acceptance spec (2026-09-22)

Kelly QA. **Bar only** — blocks item 5 engineering dispatch.

**Authority:** `phase3-exit-criteria.md` item 5; h101-128 lines 75–79; boundaries ports
4 and 8; H101-190 (shared foundation — D4 does **not** discharge item 5).

---

## What D4 already proved (do not re-claim for item 5)

| Discharged at `b7d1131` (H101-189/H101-192) | **Not** item 5 |
| --- | --- |
| Crash-prefix read via `harnessing run-output` | Graceful **complete** capture |
| `Capture status: interrupted` after mid-stream crash | `Capture status: complete` after natural exit |
| Single-chunk prefix honesty (G3 fabrication class) | Full stdout **and** stderr durability |
| `prefix_then_block` crash producer | Graceful-complete producer |
| Minimal append + `Finish(Interrupted)` on reopen | `Finish(Complete)` after drain on graceful exit |

Item 5 requires **new evidence** for every row below.

---

## What item 5 means here

> Real run output journal (not fake stream)

Output from a **real supervised child** is durably captured in `OutputJournal`,
read through the **shipped** `harnessing run-output` command, **separate from**
domain `StateEvents` (UI-08). Phase 2 fake-stream / `fakeSupervisor` verdicts
do **not** count (item 5 D5).

**Behaviour largely decided** in h101-128 lines 75–79 and boundaries port 8 —
this spec names observables; no new Stanley behaviour card expected.

---

## Passing assertions (named)

After the **graceful-complete** producer (below), via **fresh** CLI subprocesses
only:

| # | Assertion |
| --- | --- |
| I1 | `harnessing run-output` exit code **0** |
| I2 | `Capture status: complete` (exact `printRun`-class spacing on status line) |
| I3 | stdout channel contains **exact** named literal `fixture-out-stdout\n` |
| I4 | stderr channel contains **exact** named literal `fixture-out-stderr\n` |
| I5 | Second `harnessing run-output` with **`-after-offset`** past the first chunk returns **only** later bytes (offset-ordered resume per boundaries port 8) |
| I6 | `harnessing` state-event read path (subscribe/events command as shipped) does **not** contain either named literal — UI-08 separation (item 5 D2) |

**Real-not-fake evidence (item 5 D5):** test builds real `harnessing` binary,
drives **`harnessing start-run`** to completion (OK receipt), reads output only
through **`harnessing run-output`**. No `fakeSupervisor`, no engine
`ReadOutput`, no journal file parse in test.

**Not sufficient:**

- `d4_test.go` / H101-189 crash proof alone
- `internal/core/task` tests with `fakeSupervisor` as **verdict**
- Asserting capture without named child-emitted literals

---

## Condition producer (Kelly ruling)

**Use normal participate path with a new fixture mode — `emit_both_channels_then_exit`:**

1. Real binary; register + approve `__fixture-participate` + `context-file`.
2. `HARNESSING_FIXTURE_OUTPUT_MODE=emit_both_channels_then_exit` (writes
   `participated\npid=<N>` to sync if set, emits stdout + stderr literals, **exits 0**).
3. **`harnessing start-run`** runs to **completion** (OK receipt) — graceful path.
4. Fresh `harnessing run-output` → I1–I4.
5. Fresh `harnessing run-output -after-offset <N>` → I5.

**Do not use:**

| Producer | Why |
| --- | --- |
| `prefix_then_block` | D4 crash — **already discharged**; proves interrupted only |
| `block_after_marker` / `start-called` | E2 Starting window |
| Native `HARNESSING_RECONCILE_HELPER` alone | No CLI start-run + journal path (D5) |

**Product work expected:** `Finish(Complete)` when child exits gracefully and
start-run observation commits; CLI `-after-offset` (and optional `-byte-limit`)
on `run-output`; supervisor drain or equivalent so complete status is honest.

---

## Beyond D4 — explicit item 5 obligations

| Obligation | h101-128 / boundaries | Item 5 proof |
| --- | --- | --- |
| Graceful complete capture | line 79 drain + `Finish(Complete)` | I2 |
| Offset resume | port 8 ReadAfter | I5 |
| Channel identity | port 4 stdout/stderr | I3, I4 |
| RunOutput ≠ StateEvents | UI-08 | I6 |
| Real supervision path | ADR 0003 | I1–I5 via CLI start-run |

**Explicitly out of this card unless implementer stops:**

- Reader **cancel** without stopping run (item 5 D4 / UI-08) — requires working
  `FollowOutput` + cancel; journal `Follow` is stub today. **Record as item 5
  LIMIT** if not shipped; do not fake.
- Retention / `CursorExpired` — later slice
- `StopRun` tree termination — item 2

---

## Decision table

| # | Observation | Defect class | Fix |
| --- | --- | --- | --- |
| J1 | Verdict uses `fakeSupervisor` or engine `ReadOutput` only | **Test** | D5 |
| J2 | Item 5 claimed from `d4_test.go` alone | **Evidence** | D4 ≠ item 5 |
| J3 | `Capture status: complete` missing after graceful exit | **Product** | Drain/Finish |
| J4 | Named stdout/stderr literals missing | **Product** | Journal capture |
| J5 | `-after-offset` returns already-read prefix bytes | **Product** | Offset ordering |
| J6 | Raw output bytes in state-event stream | **Product** | D2 / UI-08 |
| J7 | Crash/interrupted proof used as item 5 verdict | **Test** | Wrong producer |
| J8 | Pass on one CI target via `t.Skip` | **Test** | R8 / ADR 0001 |
| J9 | Item 5 claimed while test red/skipped | **Evidence** | R9 |

---

## R8 (cross-target)

Both ADR targets when ubuntu + macos-14 CI green on item 5 proof commit.

---

## Scope

**One card** if graceful complete + both channels + offset CLI + I1–I6 suffices.

**Two cards** only if `FollowOutput`/cancel (item 5 D4) cannot be deferred as a
documented limit without architectural conflict — **stop and report**.

Does **not** discharge item 2, 4, 6, or full h101-128 retention/follow matrix.

Authored by Kelly (QA), H101-193.
