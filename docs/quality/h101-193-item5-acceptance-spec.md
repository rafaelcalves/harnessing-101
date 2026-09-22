# H101-193 — Item 5 acceptance spec (2026-09-22)

Kelly QA. **Bar only** — blocks item 5 engineering dispatch.

**Amended H101-199 (2026-09-22):** producer replaced per Stanley
[`h101-195-graceful-capture-producer.md`](../architecture/h101-195-graceful-capture-producer.md)
(`a61187f`). Fast-exit classification **unchanged**. Serve-owned capture; QA-defined
release below. Kevin held until this amendment commits.

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

After the **serve-owned graceful-complete** producer (below), assert I1–I6 using
**fresh CLI reader subprocesses** that **attach to the same serve owner** (same
workspace lock / session transport target). A fresh reader is **not** a fresh
competing host open. Wait boundedly for terminal capture before asserting I2.

| # | Assertion |
| --- | --- |
| I1 | `harnessing run-output` (reader attached to **same owner**) exit code **0** |
| I2 | `Capture status: complete` (exact `printRun`-class spacing on status line) |
| I3 | stdout channel contains **exact** named literal `fixture-out-stdout\n` |
| I4 | stderr channel contains **exact** named literal `fixture-out-stderr\n` |
| I5 | Second `harnessing run-output` reader on the **same owner** with **`-after-offset`** past the first chunk returns **only** later bytes (offset-ordered resume per boundaries port 8) |
| I6 | `harnessing` state-event read path (subscribe/events command as shipped), reader on the **same owner**, does **not** contain either named literal — UI-08 separation (item 5 D2) |

**Real-not-fake evidence (item 5 D5):** test builds real `harnessing` binary;
**`harnessing serve`** owns the workspace; **`harnessing start-run`** is a client
that completes the **start operation** (OK receipt) while the child remains under
the continuing host; output read only through **`harnessing run-output`** readers
on that owner. No `fakeSupervisor`, no engine `ReadOutput`, no journal file parse,
no `host.Open` bypass in test.

**Not sufficient:**

- `d4_test.go` / H101-189 crash proof alone
- `internal/core/task` tests with `fakeSupervisor` as **verdict**
- Asserting capture without named child-emitted literals

---

## Condition producer (Kelly ruling — amended H101-199)

**Serve-owned graceful capture.** Supersedes withdrawn one-shot
`emit_both_channels_then_exit` (StartupWindow fast-exit = `ErrSpawnFailed`; capture
goroutines die with one-shot process — H101-195).

### QA-defined release (Kelly-owned — not engineer discretion)

Fixture mode: `HARNESSING_FIXTURE_OUTPUT_MODE=serve_release_both_channels`.

| Mechanism | Rule |
| --- | --- |
| `HARNESSING_FIXTURE_SYNC_FILE` | **Required.** Fixture writes `participated\npid=<N>\n` **before** blocking — proves startup survived (same marker class as item 4 B5′). |
| `HARNESSING_FIXTURE_RELEASE_FILE` | **Required.** Fixture **blocks** until this path exists and contains **exact** bytes `release\n` (7 bytes). Poll interval ≤50ms; **wall-clock fail** if unreleased after **30s** (Kelly CI constant). |
| Post-release emission | On valid release only: write `fixture-out-stdout\n` to stdout, `fixture-out-stderr\n` to stderr, **exit 0**. |
| Ordering | No `HARNESSING_FIXTURE_SLEEP_MS` for release ordering. No fixed sleep race against `StartupWindow`. Release file is the sole gate. |

Test writes `release\n` to `HARNESSING_FIXTURE_RELEASE_FILE` **after** the start
client receives OK receipt and **before** bounded wait for capture complete.

### Producer sequence (numbered)

1. Start real **`harnessing serve`**; wait until workspace ownership is live.
2. Register + approve `__fixture-participate` + `context-file` through serve-attached CLI clients.
3. Set fixture env: `serve_release_both_channels`, `SYNC_FILE`, `RELEASE_FILE` paths.
4. **`harnessing start-run`** (client) → **OK receipt** (start **operation** complete). Client may exit. Child remains supervised by **same owner**.
5. Test writes `release\n` to `RELEASE_FILE`.
6. Fixture emits both literals and exits naturally; host **drains**, durably appends, records `Finish(Complete)` when justified.
7. **Fresh reader CLI subprocesses** attach to **same serve owner** → I1–I6.
8. **Teardown serve** only after evidence collection.

**Preserved (H101-195):** no exit-zero exception, context-file exception, sibling-marker
check, fixture-content inspection in supervisor, or fixture-aware fast-exit promotion.
`TestSupervisor_FastExitIsSpawnFailed` with `true` must still fail.

**Do not use:**

| Producer | Why |
| --- | --- |
| One-shot CLI `start-run` without continuing serve | Capture lifetime mismatch (H101-195) |
| `emit_both_channels_then_exit` (withdrawn) | Fast-exit inside StartupWindow |
| `prefix_then_block` | D4 crash — **already discharged**; interrupted only |
| `block_after_marker` / `start-called` | E2 Starting window |
| `host.Open` / direct journal read in test | Bypasses shipped reader path (H101-195) |
| Shutting down owner to bypass lock | Not equivalent producer |

**Product work expected:** serve session transport carries `run-output` and state-event
reads for attached readers; `Finish(Complete)` after drain on natural child exit;
CLI `-after-offset` on `run-output`. Shared session/transport plumbing is legitimate
implementation work and **does not** automatically discharge item 6 (H101-195).

**Non-discharge lines (carry forward):**

- Capture `complete` does **not** prove task completion or process-group disappearance.
- Successful start receipt does **not** prove capture complete or child exit.

---

## Beyond D4 — explicit item 5 obligations

| Obligation | h101-128 / boundaries | Item 5 proof |
| --- | --- | --- |
| Graceful complete capture | line 79 drain + `Finish(Complete)` | I2 |
| Offset resume | port 8 ReadAfter | I5 |
| Channel identity | port 4 stdout/stderr | I3, I4 |
| RunOutput ≠ StateEvents | UI-08 | I6 |
| Real supervision path | ADR 0003 | serve owner + CLI start-run + CLI readers |

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
