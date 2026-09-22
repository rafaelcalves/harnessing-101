# H101-224 — Natural exit finalization acceptance spec (2026-09-22)

Kelly QA. **Bar only** — blocks implementation dispatch on Stanley's shape.

**Authority:** [`h101-222-natural-exit-finalization.md`](../architecture/h101-222-natural-exit-finalization.md)
(`d4c9e4f`); H101-215/H101-217 Stop binding; H101-209 mode B; H101-223 item 5 standing
(`6c62a3e` — **not disturbed**).

**Engineering owner:** Kevin (`fixture_participate.go` + adapter). Claudio not required for
new modes if Kevin extends the worker child only.

---

## What this card proves

> Autonomous natural-exit handling **does not** reap the retained leader or release Stop
> authority while a same-group worker may still live — and **capture complete** is not the
> same thing as **run complete**.

Stanley's predicate: positive evidence that **no group member other than the retained
terminated leader remains** before a natural reap. Boolean `groupExists` / `kill(-pgid,0)`
**cannot** be the sole evidence (the zombie leader keeps it positive).

---

## Vocabulary (load-bearing)

| Term | Meaning |
| --- | --- |
| **Capture complete** | Both channels drained and durably appended; `Capture status: complete` on `run-output` (item 5) |
| **Run complete** | Owned group gone; durable run `Exited` (item 2) |
| **Retained leader** | Unreaped terminated leader per H101-217 |
| **Other-member absence** | Adapter observation that no live same-group member exists **except** the retained leader |

Pipe state is **independent** of run completion (h101-222): a worker **holding** inherited
pipes blocks capture complete; a worker **closing** pipes may allow capture complete while
the run stays `Running` and Stop binding remains.

---

## Ordering honesty — “no other members”

CLI black-box tests **cannot** prove “we did not reap at a lucky moment” during a
membership race. **Do not write a Layer A row that pretends to.**

Layer B **must** record membership decisions on the adapter lifecycle recorder (same class
as H101-217):

| Event | Meaning |
| --- | --- |
| `leader_terminated_observed` | Leader exit seen without reap |
| `other_members_absent_confirmed` | Positive evidence: no live member besides retained leader |
| `other_members_uncertain` | Partial/unreadable enumeration — **no** natural reap |
| `natural_reap_refused_uncertain` | Binding retained for Stop |
| `stop_claimed` | Stop owns serialized record |
| `authority_closed` | Before reap (H101-217) |
| `child_reaped` | Final consume |

**Layer A proves behaviour; Layer B proves the predicate was not `groupExists` alone.**

When `other_members_uncertain`, adapter **retains** Stop authority — no speculative reap,
no silent success.

---

## Fixture requirements — **one new worker behaviour, no new top-level mode**

Reuse **mode B** `parent_exits_worker_survives` (H101-213 + H101-221). Add worker env only:

| `HARNESSING_FIXTURE_WORKER_PIPE` | Worker child behaviour |
| --- | --- |
| **`hold`** (default) | Existing `worker_block` — keeps inherited stdout/stderr (and stdin) open |
| **`close`** | **New (Kevin).** After appending `worker_ready\n`, worker **closes** inherited stdin/stdout/stderr, then blocks |

Parent sequence unchanged: spawn worker → sync + `worker_ready\n` → block on
`PARENT_RELEASE_FILE` → exit 0 with worker alive.

**Count:** **0** new `HARNESSING_FIXTURE_OUTPUT_MODE` values; **1** new worker pipe mode
(`close`).

---

## Evidence layers

| Layer | Package | Proves |
| --- | --- | --- |
| **A — Integration** | `cmd/harnessing/` | Capture vs run separation; mode B pipe hold/close; item 5 regression |
| **B — Native** | `internal/adapters/process/` | Membership predicate, natural↔Stop serialization, concurrent outcome |

Both ADR targets (R8).

---

## Layer A — Integration rows (serve-owned, mode B)

Same serve owner as H101-209. All rows require H101-221 `worker_ready\n` before
worker-alive claims.

### A1 — Pipe **hold** (capture blocked, run stoppable)

1. Mode B + `WORKER_PIPE=hold`.
2. After parent release: `Running`; leader zombie; worker alive + `worker_ready\n`.
3. **≤5s** poll of attached `run-output` (20ms interval): **`Capture status: complete` must NOT appear**.
4. `stop-run` → **0**; run `Exited`; worker pid probe fails ≤30s.

### A2 — Pipe **close** (capture may complete, run not)

1. Mode B + `WORKER_PIPE=close`.
2. After parent release: `Running`; leader zombie; worker alive + `worker_ready\n`.
3. **≤5s** `run-output` poll: **`Capture status: complete` appears** (H101-223 contract).
4. `harnessing run` still **`Running`** (not `Exited`) — capture complete ≠ run complete.
5. `stop-run` → `Exited`; worker gone ≤30s.

### A3 — Item 5 regression (unchanged contract)

`TestCLI_ServeOwnedGracefulCaptureCompletesBothChannels` **PASS** at `6c62a3e` semantics
(H101-223 I2 poll). This card does **not** amend item 5.

### A4 — Natural vs Stop race (integration smoke)

1. Mode B + `WORKER_PIPE=hold`.
2. After parent release + `Running`, concurrently: attached `stop-run` **and** repeated
   `harnessing run` queries (≤5s).
3. **Exactly one** terminal run outcome: `Exited` after stop succeeds; **no** stuck
   `Stopping` without resolution; **no** double-orphan worker (worker probe fails after bound).

(Full serialization proof is Layer B — A4 is smoke only.)

---

## Layer B — Native adapter rows

Extend H101-217 lifecycle recorder. Required on **each** ADR target.

| # | Row |
| --- | --- |
| B1 | Mode-B-shaped harness (leader exits, worker survives): `leader_terminated_observed` before any `child_reaped`; **no** `other_members_absent_confirmed` while worker pid alive |
| B2 | Worker gone (or stopped): `other_members_absent_confirmed` **before** `authority_closed` → `child_reaped` on natural path **or** Stop path |
| B3 | Injected unreadable membership (`other_members_uncertain`): `natural_reap_refused_uncertain`; Stop still succeeds later; **no** `child_reaped` on uncertain path alone |
| B4 | **Stop claims first:** `stop_claimed` before natural path reaps; natural observer does **not** steal record (no `child_reaped` without Stop sequence or confirmed absence) |
| B5 | **Natural wins safely:** when `other_members_absent_confirmed`, `authority_closed` before `child_reaped`; retained leader only |
| B6 | **Concurrent callers:** two `Stop` or `Stop`+waiter — same terminal error/success; one signal set (extends H101-217 N4) |
| B7 | `closed` alone is **not** success: waiter must observe actual terminal result after authority close |

**Forbidden:** natural reap justified only by `groupExists(pgid)==true`.

---

## Creed / Stanley mapping

| h101-222 requirement | Row |
| --- | --- |
| Observe leader termination without reap | B1, A1/A2 zombie leader |
| Positive other-member absence before natural reap | B2, B3 refusal |
| Natural ↔ Stop same serialized record | B4, B5, B6 |
| Capture ≠ run completion | A1 vs A2 |
| Item 5 not disturbed | A3 |
| No speculative reap on uncertainty | B3 |

---

## Decision table

| # | Observation | Defect class |
| --- | --- | --- |
| F1 | `Capture status: complete` while pipe-hold worker alive (A1) | **Product** |
| F2 | Run `Exited` on parent/leader exit alone with pipe-close worker still alive (A2) | **Product** |
| F3 | Natural reap with only `groupExists` true | **Product** |
| F4 | `child_reaped` after `other_members_uncertain` without Stop | **Product** |
| F5 | Natural observer reaps after `stop_claimed` | **Product** |
| F6 | Layer A row claims membership proof without Layer B recorder | **Test** |
| F7 | Item 5 I2 poll contract broken | **Product** — blocks this card |
| F8 | A1/A2 without `worker_ready\n` | **Test** — H101-221 |

---

## Scope — one card

**One card** with Kevin if:

- Worker `close` pipe mode in `fixture_participate.go`
- Adapter membership observation + h101-222 lifecycle serialization
- Layer A A1–A4 + Layer B B1–B7
- H101-217 rows preserved

**Does not discharge:** item 2 full verdict (still Kelly pass after tree green); H101-220
(CI linux-native leg — answer with item 2 verdict).

---

Authored by Kelly (QA), H101-224.
