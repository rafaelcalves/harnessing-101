# H101-209 — Item 2 acceptance spec (2026-09-22)

Kelly QA. **Bar only** — blocks item 2 engineering dispatch.

**Authority:** `phase3-exit-criteria.md` item 2; H101-130 tree-scope amendment;
Stanley [`h101-128-phase3-supervision.md`](../architecture/h101-128-phase3-supervision.md)
lines 9, 47–53 (group supervision, exit observation); Creed
[`h101-173-recovery-resolution-boundary.md`](../security/h101-173-recovery-resolution-boundary.md)
(no unblock/restart path in this card).

**Stanley behaviour card:** **not required** — h101-128 already settles StopRun,
group termination, and parent-exit insufficiency. Card Stanley only if implementation
discovers a gap h101-128 does not cover.

---

## What item 2 means here

> `StopRun` and owned process-tree termination

Through the shipped `harnessing stop-run` command, an active supervised run reaches
honest **`Exited`** only after the **owned process group** is gone — not on kill
request alone, not on parent exit while a group worker lives (H101-130 / h101-128).

**H101-173 boundary:** this card proves **termination**, not recovery resolution,
admin unblock, or restart. No `RecoveryRequired` clearance path belongs here.

---

## Producer reuse (five existing — all insufficient)

| Existing mode | Why not item 2 |
| --- | --- |
| `prefix_then_block` | D4 crash / interrupted capture only |
| `block_after_marker` | Single process; no worker tree |
| `serve_release_both_channels` | Graceful natural exit (item 5) |
| Normal participate + sync | Single process; no descendant to kill |
| `HARNESSING_FIXTURE_SIMULATE` | R3 negatives only |

**New producer required:** two fixture modes below. Item 2 cannot reuse item 5's
graceful-exit or item 4's crash producers.

---

## Condition producer (Kelly ruling)

**Host:** continuing **`harnessing serve`** (same ownership lesson as H101-199 item 5).
Register + approve `__fixture-participate` + `context-file`. **`start-run`** and
**stop-run`** are **attached CLI clients** on the same owner. Fixture env vars belong
on the **serve** process environment (child inherits serve env).

### Mode A — `spawn_worker_then_block` (primary tree kill)

1. `HARNESSING_FIXTURE_OUTPUT_MODE=spawn_worker_then_block`
2. `HARNESSING_FIXTURE_SYNC_FILE` **required**
3. After participate marker, fixture **spawns a real worker child** in the
   supervisor-established group (engineer chooses mechanism; must stay in owned group
   per supported profile contract).
4. Writes sync file **before** blocking:

   ```
   participated\npid=<parent>\nworker_pid=<worker>\n
   ```

5. Parent blocks until group receives termination (SIGTERM from StopRun path).

### Mode B — `parent_exits_worker_survives` (Stanley parent-exit case)

Required native subtest per H101-130 / h101-128 line 9 (parent + worker, including
parent exits before worker).

1. `HARNESSING_FIXTURE_OUTPUT_MODE=parent_exits_worker_survives`
2. Same sync file shape with `worker_pid=<worker>`
3. Parent writes sync, **exits 0** while worker remains alive in the group
4. **`Exited` must not** be recorded until worker is gone — proved in assertions S1–S3

### Proving absence (god's hardest negative)

**Not** proved by "looking at nothing." **Positive artefact pattern** (same class as
item 10 REPLACE PID marker):

| Phase | Observable |
| --- | --- |
| Before `stop-run` | Sync file contains `worker_pid=<N>`; **PID probe succeeds** for worker (and parent in mode A) |
| After `stop-run` + bounded wait | **PID probe fails** for `worker_pid` within **30s** wall clock (Kelly E3-class bound, same order of magnitude as item 3) |
| Terminal state | `harnessing run` (attached reader) shows **`Exited`** only after worker probe fails |

Engineer documents probe mechanism (`kill -0`, `os.FindProcess` + signal 0, or
platform equivalent). Test must fail if worker still alive at bound.

---

## Passing assertions — Mode A (named)

After producer A, via fresh attached CLI subprocesses on the **same serve owner**:

| # | Assertion |
| --- | --- |
| I1 | `start-run` exit **0** with OK receipt; run observable **`Running`** before stop |
| I2 | Sync file contains **`worker_pid=<N>`** before `stop-run` |
| I3 | PID probe **succeeds** for `worker_pid` immediately before `stop-run` |
| I4 | `stop-run` exit **0** (or documented in-progress receipt per h101-128) |
| I5 | `harnessing run` shows **`State:      Exited`** (not stuck `Running`/`Stopping`) |
| I6 | PID probe **fails** for `worker_pid` within **30s** after `stop-run` |
| I7 | PID probe **fails** for parent `pid` within same window (mode A — group gone) |

**Real-not-fake (item 2 D5):** real `harnessing` binary; real worker child; CLI
`start-run` + `stop-run` + `run` only — no `fakeSupervisor`, no engine-only verdict.

### Mode B sub-assertions (same test file, required)

| # | Assertion |
| --- | --- |
| S1 | After parent exit, worker PID probe **still succeeds** before `stop-run` |
| S2 | `stop-run` still required — run not `Exited` on parent exit alone |
| S3 | After `stop-run`, worker probe fails and run shows **`Exited`** |

### Negative (same card)

| # | Assertion |
| --- | --- |
| N1 | `stop-run` on unknown `runID` → stable **`NotFound`**; persisted state unchanged (item 2 D3) |

---

## Decision table

| # | Observation | Defect class | Fix |
| --- | --- | --- | --- |
| J1 | Verdict uses `fakeSupervisor` or engine stop only | **Test** | D5 |
| J2 | `Exited` while worker PID probe still succeeds | **Product** | D1/D2/D6 |
| J3 | Worker survives after `stop-run` on supported profile | **Product** | D2/D4/D6 |
| J4 | `Exited` recorded when parent alone exited (mode B) | **Product** | H101-130 parent-exit rule |
| J5 | `stop-run` unknown run does not return `NotFound` | **Product** | D3 |
| J6 | Kill request returns OK but run stays `Running` with live worker | **Product** | D1 |
| J7 | Test uses mock child without real spawned worker | **Test** | D5 |
| J8 | Item 2 proof includes admin unblock / restart | **Process** | H101-173 forbidden |
| J9 | Pass on one ADR target via `t.Skip` | **Test** | R8 / ADR 0001 |
| J10 | Escaping/daemon profile promised tree kill without spawn reject | **Product** | D7 |

---

## R8 (cross-target)

Both ADR targets when ubuntu + macos-14 CI green on item 2 proof commit.
Mode A **and** mode B subtests on **each** target.

---

## Scope — one card or two

**One card** if:

- Both fixture modes in `fixture_participate.go`
- Mode A + mode B subtests + N1 in `cmd/harnessing/`
- Serve attach + real group termination on both ADR targets

**Two cards** only if mode B requires adapter behaviour h101-128 does not already
require for parent-exit-before-worker — **stop and report** (Kelly does not expect
this; h101-128 line 48 already forbids parent-exit-alone `Exited`).

**Does not discharge:**

- Item 3 budget termination path (may share termination machinery; not evidence)
- Item 4 crash recovery / `RecoveryRequired` (H101-173 unblock separate)
- Item 5 output journal
- Item 6 detach hosting
- Item 8 disclosure (tree-scope limit text remains item 8 obligation)
- Deliberately escaping / unsupported-profile descendants (H101-130 limit — item 8
  disclosure, not item 2 defect)

**Verdict shape:** item 2 may exit **SATISFIED WITH LIMIT** naming H101-130 tree-scope
boundary in item 8 — same as criteria already states.

---

Authored by Kelly (QA), H101-209.
