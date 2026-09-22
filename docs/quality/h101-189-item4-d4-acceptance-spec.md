# H101-189 — Item 4 D4 acceptance spec (2026-09-22)

Kelly QA. **Bar only** — blocks D4 engineering dispatch. Closes item 4 row **D4**
only. Completing D4 is the last open row before item 4 satisfaction (subject to
E1–E5 / D1–D3 / D5–D6 standing unchanged).

**Authority:** Stanley `h101-128-phase3-supervision.md` lines 73, 79 and
`boundaries.md` port 4 (RunOutput terminal status). **Behaviour is already
decided** — this spec names observables and the minimal slice; it does not invent
product policy.

---

## Design answer (god's question)

When a run crashes mid-capture with **some** output already persisted:

| Requirement | Source |
| --- | --- |
| Persisted prefix bytes remain **readable and exact** | h101-128 line 79 — "complete persisted chunks" |
| Missing tail is **reported as a gap**, not invented | h101-128 line 79 — "report any unknown gap **without invented bytes or byte counts**" |
| Terminal capture is **interrupted/unknown**, not **complete** | h101-128 line 73 — "interrupted/unknown terminal capture marker"; boundaries port 4 |
| Observations never captured stay **absent** | h101-128 line 73 — "absent observations remain absent" |
| **Forbidden:** synthesizing stdout/stderr to fill the gap | Item 4 D4 defect class |
| **Forbidden:** implying a complete capture when the controller died mid-stream | boundaries port 4 |

**Distinguish gap from "no output yet":**

| Situation | Required honest surface |
| --- | --- |
| Zero chunks ever persisted | Empty/absent read — **not** fabricated bytes |
| Prefix persisted, crash before tail | Prefix exact + **interrupted/unknown** terminal status + **no** invented tail |

**No new Stanley card for behaviour.** If dispatch surfaces a dispute that item 4
may not introduce **any** `OutputJournal` work before item 5's full acceptance,
**stop** — that is scope architecture (see Scope), not criteria invention.

---

## What D4 means here

> Fabricated output bytes filling gap after crash

The user must not read run output after an unclean controller exit and receive
bytes, byte counts, or a **complete** capture claim for data that was never
durably appended. The read path must be a **shipped product command** (item 5 names
`harnessing run-output` or documented equivalent — use whatever lands; D6 class:
no `engine`, `host.Open`, or store file reads in the test).

---

## Passing assertions (named)

After the condition producer (below), a **fresh** CLI subprocess on the output
read path:

| # | Assertion |
| --- | --- |
| C1 | Exit code **0** |
| C2 | Response contains the **exact** persisted prefix bytes the fixture emitted before kill (named literal in test, e.g. `fixture-out-1\n`) |
| C3 | Terminal/capture status is **interrupted** or **unknown** — **not** **complete** |
| C4 | Response does **not** contain bytes the fixture never emitted (include at least one named negative literal, e.g. `fixture-out-2\n`, emitted only after the kill window) |
| C5 | Response does **not** claim a byte count or range covering unpersisted tail data |

**Not sufficient:**

- Engine `GetRun`, journal adapter unit tests only, or parsing workspace files
- Asserting only on `harnessing run` state without the output read path
- Phase 2 fake-stream doubles as verdict evidence (item 5 D5 class)

---

## Condition producer (Kelly ruling — reuse, do not let engineer pick)

**Use the D2 participate + pid sync producer** (`d2_test.go` class):

1. Real `harnessing` binary; register + approve `__fixture-participate` +
   `context-file`; `HARNESSING_FIXTURE_SYNC_FILE` set.
2. **`harnessing start-run`** subprocess; wait for **`participated\npid=<N>`**.
3. Fixture emits **known stdout line 1** (persisted), then blocks before **line 2**
   (never emitted before kill). Extend `fixture_participate.go` — not a third
   crash harness.
4. **SIGKILL** fixture child by recorded `pid=` **after** line 1 is durable in
   journal (test waits on sync or journal-ready hook — product must expose one).
5. **SIGKILL** `start-run` controller while still alive (unclean; S9 class).
6. Fresh output-read subprocess → assert C1–C5.

**Do not use:**

| Producer | Why |
| --- | --- |
| `block_after_marker` / `start-called` | E2/Starting window — no Running, no meaningful capture slice |
| Native `HARNESSING_RECONCILE_HELPER` re-exec **alone** | Engine path; D6 violation without CLI output read |
| Graceful `Close` only | D5 class |

**Product work expected (minimal item 4 slice):** `OutputJournal` append for
supervised child stdout during Running (at least one chunk); `Finish(Interrupted)`
or equivalent honest terminal on crash reopen; shipped read command returning
chunks + terminal status per boundaries port 4. **Does not** discharge item 5
(graceful complete drain, follow/cancel semantics, retention limits).

---

## Decision table (read before implementation)

| # | Observation | Defect class | Fix |
| --- | --- | --- | --- |
| G1 | Test reads journal via engine/store file, not product command | **Test** | D6 class |
| G2 | Prefix bytes wrong or missing after crash | **Product** | Durability |
| G3 | Tail bytes present though never emitted | **Product** | **D4 fabrication** |
| G4 | Terminal status **complete** after mid-stream crash | **Product** | **D4 honesty** |
| G5 | Byte count/range implies unpersisted data | **Product** | **D4** — h101-128 line 79 |
| G6 | Zero persisted chunks but response contains invented bytes | **Product** | "absent remain absent" |
| G7 | Crash after graceful `Close` only | **Test** | D5 class |
| G8 | Wrong crash window (`start-called` / Starting only) | **Test** | E2 class, not D4 |
| G9 | Reconcile on continuing host's own run | **Test** | h101-187 §1 / S9 |
| G10 | Pass on one in-scope CI target via `t.Skip` | **Test** | R8 / ADR 0001 |
| G11 | D4 or item 4 claimed while test red/skipped | **Evidence** | R9 class |

---

## R8 (cross-target)

Same rule as H101-183/H101-188: both ADR targets when ubuntu + macos-14 CI jobs
green on the D4 proof commit.

---

## Scope

**One card (H101-189 engineering)** if minimal journal append + interrupted
finish + output read command + fixture line emission + CLI test above suffices.

**Two cards** only if:

1. Stanley rules the journal adapter must not land under item 4 at all (item 5
   gate only) — **stop and report**; item 4 cannot close without a ruling; or
2. Implementer finds read command and journal adapter cannot ship together without
   an architectural conflict — **stop per one-card-or-two rule**.

**Does not discharge:** item 5, item 4 rows already closed, live supervision,
full follow/cancel/retention matrix.

**Pre-dispatch check:** `internal/adapters/journal/` is placeholder-only today.
This bar **requires** first real journal code — expected increment, same class as
H101-170's minimal reconcile slice, citing h101-128 acceptance map (items 4–5
crash journal evidence).

Authored by Kelly (QA), H101-189.
