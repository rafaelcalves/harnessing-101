# H101-184 — Item 4 D2 acceptance spec (2026-09-22)

Kelly QA. **Bar only** — blocks D2 engineering dispatch. Closes item 4 row **D2**
only. D4 remains after D2 per H101-180 order.

**Authority:** Stanley `h101-128-phase3-supervision.md` lines 53, 69–71 (already
committed architecture — **no new Stanley card required** for the behaviour below).

---

## Design answer (god's question)

When persisted state is **`Running`** but the supervised child is **gone** and
ownership/outcome **cannot** be re-established on reopen:

| Outcome | Required? |
| --- | --- |
| Remain `Running` | **Forbidden** — item 4 D2 defect |
| `Exited` without established terminal outcome | **Forbidden** — h101-128: report uncertainty, not `Exited` |
| **`RecoveryRequired`** | **Required** — conservative `Recover` → Unknown → durable `RecoveryRequired` on reopen reconcile |

Live supervision while the controller stays up is **out of scope** for this card
(H101-170 non-claim). D2 is proved on **next `harnessing` open** after the child is
dead, same class as E2/D6.

---

## What D2 means here

> Run stuck `Running` though child is dead

The user must not query `harnessing run` and see **`State:      Running`** when the
fixture child has been **SIGKILLed** and the conservative adapter cannot establish
ownership. The read path is again **`harnessing run`** → `printRun` (D6 discipline
carries forward — no engine/host/store reads in the test).

---

## Passing assertions (named)

After the condition producer (below), a **fresh** `harnessing run` subprocess:

| # | Assertion |
| --- | --- |
| B1 | Exit code **0** |
| B2 | stdout contains **`Run <runID>`** for the affected run |
| B3 | stdout contains **`State:      RecoveryRequired`** |
| B4 | stdout does **not** contain **`State:      Running`** |
| B5′ | (Precondition, same test) `HARNESSING_FIXTURE_SYNC_FILE` contains
**`participated\npid=<N>`** before controller SIGKILL; child killed via that
`pid=` before reopen — proves stuck-**Running** window, not E2
`start-called` / stuck-**Starting** (see H101-186: written B5 was
unsatisfiable with today's lock lifecycle) |

**Not sufficient:**

- Engine `GetRun`, `host.Open`, parsing `runs/` JSON
- Only checking events without `harnessing run`

---

## Condition producer (reuse before inventing)

**Do not** reuse the E2 `block_after_marker` window — that dies before `Running`.

**Preferred (one card):**

1. Same setup as `TestCLI_StartRun_LayerAParticipationFixture`: real `harnessing`
   binary, register, approve profile with `__fixture-participate` +
   `context-file`, **`harnessing start-run`** in a subprocess with
   `HARNESSING_FIXTURE_SYNC_FILE` set.
2. Wait until sync file contains **`participated\npid=<N>`** (extend normal
   participate path to write this — **not** `block_after_marker` /
   `start-called`). Proves Running-commit window per H101-186 B5′.
3. **SIGKILL** the fixture child using **`pid=`** from the sync file.
4. **SIGKILL** the `start-run` controller subprocess (unclean, no `Close`) —
   while still alive after step 2; leaves durable `Running` with dead child.
5. Fresh `harnessing run` → assert B1–B4.

**Product work expected (one card):** extend `ReconcileStuckRuns` (or equivalent
single `host.Open` reconcile pass) to **`RunRunning`** when `Recover` returns
`ErrRecoveryRequired` — h101-128 line 69 already requires reconciling **nonterminal**
runs; H101-170 only implemented the `Starting` slice.

**One card** (H101-185): Running reconcile + CLI test above. H101-187
(`4be5966`) affirms conservative reopen reconcile; B5′ per H101-186. Reconcile
must run only on **fresh `host.Open` after controller death** — never on a
continuing host's own supervised runs (h101-187 §1).

---

## Decision table (read before implementation)

| # | Observation | Defect class | Fix |
| --- | --- | --- |
| S1 | Test uses `engine.GetRun`, `host.Open`, or store file read for verdict | **Test** | D6 class |
| S2 | Test never proved Running-commit window before kill (B5′ missing) | **Test** | Not D2 — wrong precondition / E2 again |
| S3 | Child killed before `Running` committed | **Test** | E2 window, not D2 |
| S4 | `harnessing run` still `State:      Running` after dead child + reopen | **Product** | **D2** |
| S5 | `harnessing run` shows `Exited` without established terminal outcome | **Product** | h101-128 honesty |
| S6 | Crash after graceful `Close` only | **Test** | D5 class |
| S7 | Pass on one in-scope CI target via `t.Skip` | **Test** | R8 / ADR 0001 |
| S8 | D2 claimed while test red/skipped | **Evidence** | R9 class |
| S9 | Reconcile exercised on continuing host's own run, not post-crash fresh open | **Test** | h101-187 §1 collision |

---

## R8 (cross-target)

Same rule as H101-183: both ADR targets when ubuntu + macos-14 CI jobs green.
H101-183 recorded CI SUCCESS on `21c782e` for E2 — same test package pattern applies.

---

## Scope

**One card** if Running reconcile + CLI test above suffices.

**Two cards** if architecture review shows Running reconcile is a new design decision
(Kelly does not expect this given h101-128 line 69).

Does **not** discharge item 4, D3 partial upgrade, D4, or live supervision.

Authored by Kelly (QA), H101-184.
