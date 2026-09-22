# H101-209 — Item 2 acceptance spec (2026-09-22)

Kelly QA. **Bar only** — blocks item 2 engineering dispatch.

**Amended H101-213 (2026-09-22):** mode B producer ordering per Stanley
[`h101-212-parent-exit-startup-boundary.md`](../architecture/h101-212-parent-exit-startup-boundary.md)
(`262c863`). Parent exits **after** successful start + `Running` observation, not before
startup completes. QA-defined parent-release sync; parent-gone/worker-alive proof; N2
fast-parent/idle-worker negative. Fast-exit classification unchanged (Creed floor).

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

**Amended H101-213.** Required native subtest per H101-130 / h101-128 line 9 (parent +
worker, including parent exits **before worker** — **not** before startup completes).
Authority: h101-212.

**Forbidden ordering (Stanley H101-212):** parent exits during `StartupWindow` while a
same-group worker survives and `Start` accepts the run via group-existence alone.
Fixture release is **test coordination only** — never a marker the production
supervisor inspects to accept a start.

#### QA-defined parent release (Kelly-owned)

| Mechanism | Rule |
| --- | --- |
| `HARNESSING_FIXTURE_PARENT_RELEASE_FILE` | **Required** for mode B. Parent **blocks** after writing sync until this path contains **exact** bytes `parent-release\n` (14 bytes). Poll ≤50ms; fail after **30s**. |
| Ordering | Parent release happens **only after** steps 4–5 below — never before OK receipt + `Running` observation. |

#### Producer sequence (numbered — same serve owner throughout)

1. Serve env: `parent_exits_worker_survives`, `SYNC_FILE`, `PARENT_RELEASE_FILE`.
2. Fixture spawns same-group worker; writes sync:

   ```
   participated\npid=<parent>\nworker_pid=<worker>\n
   ```

3. Parent **blocks** (does **not** exit yet).
4. Attached `start-run` → **OK receipt** (start operation complete under normal
   fast-exit rules — no group-exists startup exception).
5. Attached `harnessing run` shows **`State:      Running`**.
6. Test writes `parent-release\n` to `PARENT_RELEASE_FILE`.
7. Parent exits **0**; worker remains alive.
8. **Parent-gone / worker-alive proof** (positive artefact, same serve owner):

   | Probe | After step 7 |
   | --- | --- |
   | Parent `pid` from sync | **Fails** |
   | `worker_pid` from sync | **Succeeds** |

9. Assert S1–S3 (below), then `stop-run` → worker gone → `Exited`.

**`Exited` must not** be recorded at step 7 or before `stop-run` clears the worker.

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

### Mode B sub-assertions (same test file, same serve owner, required)

Observed **after** producer steps 1–7 and **before** `stop-run`:

| # | Assertion |
| --- | --- |
| S0 | Before parent release (step 5): both parent and worker PID probes **succeed** |
| S1 | After parent exit (step 7): parent probe **fails**, worker probe **still succeeds** |
| S2 | `harnessing run` on same owner shows **`Running`** (or `Stopping`) — **not** `Exited` on parent exit alone |
| S3 | After `stop-run`, worker probe fails and run shows **`Exited`** |

### Negatives (same card)

| # | Assertion |
| --- | --- |
| N1 | `stop-run` on unknown `runID` → stable **`NotFound`**; persisted state unchanged (item 2 D3) |
| N2 | **Fast-parent / idle-worker (H101-212):** fixture mode `fast_parent_idle_worker` spawns same-group idle worker (`worker_block` child) and parent exits **during startup** without participate blocking — `start-run` **fails** (not `Running`); `TestSupervisor_FastExitIsSpawnFailed` with `true` **still passes** unchanged. Test **cleans up** surviving worker (bounded teardown) |

**N2 producer:** `HARNESSING_FIXTURE_OUTPUT_MODE=fast_parent_idle_worker` — fork worker,
exit parent immediately; **no** `PARENT_RELEASE_FILE`, no participate sync wait. This is
the regression complement to mode B's positive case; it must **not** reuse mode B's
post-startup parent release path.

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
| J11 | `Start` succeeds because parent exited during startup leaving idle same-group worker | **Product** | H101-212 — fast-exit / groupExists exception forbidden |
| J12 | Mode B parent exits before OK receipt or before `Running` observation | **Test** | H101-213 ordering violation |
| J13 | N2 fast-parent case reaches `Running` or leaves worker without cleanup | **Test** | N2 |

**Preserved (Creed floor / Stanley H101-212):** no weaker fast-exit observables; no
supervisor inspection of fixture release markers for start acceptance; H101-173 no
unblock/restart in this card.

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

Authored by Kelly (QA), H101-209. Amended H101-213.
