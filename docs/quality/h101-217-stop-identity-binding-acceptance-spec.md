# H101-217 — Stop identity binding acceptance spec (2026-09-22)

Kelly QA. **Bar only** — blocks H101-218 engineering dispatch (Kevin).

**Authority:** Stanley [`h101-215-stop-identity-binding.md`](../architecture/h101-215-stop-identity-binding.md)
(`2de10b4`); Creed H101-216 floor lifted (no residual floor); h101-128 line 53
(no kill by reused identifier); H101-173 (no unblock/restart in this card).

**Does not re-verdict item 2.** Claudio's H101-213 Start/fixture rework is verified and
held; it commits **together with** this binding. Item 2 discharge still uses
[`h101-209-item2-acceptance-spec.md`](h101-209-item2-acceptance-spec.md) once both land.

---

## What this card proves

> Stop's kill authority binds to the **actual spawned group leader**, not a bare process-group
> number that may have been reused.

Mechanism (h101-215): retain the leader **unreaped** (`waitid` + `WNOWAIT`), observe exit
without consuming status, serialize SIGTERM/SIGKILL and final reap through one adapter-owned
record, close signal authority **before** reap, never signal the old pgid after release.

---

## Vocabulary — termination vs reaping (Kelly-owned)

These words are **load-bearing** in assertions below. Do not conflate them.

| Term | Meaning | Positive observable |
| --- | --- | --- |
| **TERMINATION** | Leader has finished user code; kernel may still hold an **unreaped zombie** at `leader_pid` | `leader_pid` probe succeeds **and** process state is zombie (platform table below) |
| **REAPING** | Adapter consumed the leader's exit status; pid slot released | `leader_pid` probe fails with **ESRCH** (or platform equivalent) |

**Fixture rule:** a test that only checks `kill(pid,0)==0` proves **existence**, not
**TERMINATION-vs-running**. Creed point 4: a positive **group probe** during Stop must never
be read as "worker still running" — tests must use **per-pid state**, not group liveness alone.

---

## Platform-native zombie evidence (required — differs by target)

Both ADR targets require native evidence. Mechanism is the same; **observation** differs.

| Target | Zombie (TERMINATION, unreaped) | Reaped (REAPING complete) |
| --- | --- | --- |
| **linux/amd64** | `kill(leader_pid,0)==0` **and** `/proc/<leader_pid>/status` contains `State:\tZ` (or `State: Z`) | `kill(leader_pid,0)` returns **ESRCH**, or status file absent |
| **darwin/arm64** | `kill(leader_pid,0)==0` **and** `ps -p <leader_pid> -o state=` output trimmed is **`Z`** or **`Z+`** | `kill(leader_pid,0)` returns **ESRCH**, or `ps` reports process not found |

Engineer documents the helper used; test must **fail** if leader is still **running** when
the bar expects TERMINATION, and **fail** if leader is still zombie when the bar expects
REAPING.

---

## Evidence layers (two required)

| Layer | Package / surface | Proves |
| --- | --- | --- |
| **A — Integration** | `cmd/harnessing/` serve-owned CLI tests | End-to-end Stop under real ownership; TERMINATION→REAPING through product; mode B from H101-209 |
| **B — Native adapter** | `internal/adapters/process/` | Creed load-bearing ordering, per-signal checks, stale-callback refusal, binding-failure path |

Layer A alone is **insufficient** for points 1–3 and 5 (see §Ordering honesty).

---

## Condition producer — integration (reuse H101-209 mode B)

**Primary producer:** `parent_exits_worker_survives` per H101-213 (serve-owned,
`PARENT_RELEASE_FILE`, same serve owner throughout).

After producer step 7 (parent released, worker alive):

| # | Assertion |
| --- | --- |
| T1 | `leader_pid` (sync `pid=`) is **zombie** per platform table — **TERMINATION without REAPING** |
| T2 | `worker_pid` probe succeeds — worker **running** |
| T3 | `harnessing run` shows **`Running`** (not `Exited`) — parent TERMINATION alone insufficient |
| T4 | `stop-run` exit **0** (or documented in-progress receipt) |
| T5 | Within **30s** after `stop-run`: `worker_pid` probe fails; `leader_pid` probe fails — **REAPING** |
| T6 | `harnessing run` shows **`State:      Exited`** |

**Creed point 4 (integration row):** between T3 and T5, Stop must complete even though
T1 shows leader already **zombie** (group probe may stay positive). Test must **not**
block on group disappearance before SIGKILL; bounded forced path must finish within the same
**30s** wall clock as T5.

**Start-path trap (Creed H101-216):** T1 is also proof that `Start` did **not** silently
reap the leader via an unconditional `cmd.Wait()` goroutine — if T1 fails because leader
pid is gone before `stop-run`, the binding mechanism is defeated.

---

## Native adapter requirements (Layer B)

### Lifecycle recorder (ordering honesty)

**God's hardest row:** CLI black-box tests **cannot** distinguish "authority closed before
reap" from "reaped first and nothing bad happened this run." **Do not write a Layer A row
that pretends to.**

Layer B **must** use a **test-only ordered lifecycle recorder** (injectable hook, build-tag
test helper, or equivalent) that records at minimum:

```
pre_sigterm_check → sigterm_sent → pre_sigkill_check → sigkill_sent → authority_closed → child_reaped
```

(Names illustrative; order is binding.)

| # | Native assertion |
| --- | --- |
| N1 | **Per-signal recheck (Creed 1):** distinct `pre_sigterm_check` and `pre_sigkill_check` events; each precedes its signal event |
| N2 | **Close before reap (Creed 2):** `authority_closed` appears **before** `child_reaped` in the ordered log |
| N3 | **No post-release signals (Creed 3):** no `sigterm_sent` or `sigkill_sent` after `authority_closed` |
| N4 | **Stale callback (Creed 3):** concurrent or sequential second `Stop` after first completes produces **no** signal events; decoy pid (below) untouched |
| N5 | **Bounded grace + zombie probe (Creed 4):** with leader already TERMINATED (zombie) and worker alive, forced path fires within configured grace + **30s** observation bound — does not wait forever for group probe false |
| N6 | **Hard refusal (Creed 5):** when binding cannot be established on an ADR target, `Stop` returns **`RecoveryRequired`** — never a best-effort kill to the recorded pgid alone |

### Decoy process (unrelated-kill negative)

Native test only. Before exercising Stop:

1. Record `decoy_pid` of an unrelated same-user process that is its **own group leader**
   (e.g. `sleep` in its own session, or test helper).
2. Run Stop on the managed fixture run to completion.
3. **`decoy_pid` probe still succeeds** after Stop — no SIGTERM/SIGKILL reached the decoy.

This is the positive artefact for h101-128 "never signal unrelated process."

### Additional native scenarios (h101-215 §Evidence)

| Scenario | Requirement |
| --- | --- |
| Parent exit + surviving worker | Mode B integration path (T1–T6) |
| Concurrent `Stop` | N4 |
| Fast-exit cleanup | `TestSupervisor_FastExitIsSpawnFailed` with `true` **unchanged** |
| Denied / unknown observation | N6 — `RecoveryRequired`, not silent kill |
| Stale timer/callback after release | N3 + N4 |

---

## Creed five load-bearing points — mapping

| Creed point | Layer | Row |
| --- | --- | --- |
| 1. Recheck before **each** signal | B | N1 |
| 2. Close authority before reap, same serialization | B | N2 (not provable at Layer A) |
| 3. No stale/queued signal after release | B | N3, N4 |
| 4. Bounded grace when zombie keeps group probe positive | A + B | T4–T6 + N5 |
| 5. `RecoveryRequired` fallback, no best-effort kill | B | N6 |

---

## Decision table

| # | Observation | Defect class | Fix |
| --- | --- | --- | --- |
| B1 | Layer A only; no native lifecycle recorder tests | **Test** | §Layer B |
| B2 | Leader pid gone before `stop-run` in mode B (Start reaped early) | **Product** | Creed trap / H101-218 |
| B3 | `stop-run` succeeds but decoy pid killed | **Product** | Identity binding |
| B4 | Signal events after `authority_closed` in recorder | **Product** | Creed 2–3 |
| B5 | Only one `pre_sig*_check` for both signals | **Product** | Creed 1 |
| B6 | Hangs with zombie leader + live worker past grace + 30s | **Product** | Creed 4 |
| B7 | Binding failure still sends SIGKILL to bare pgid | **Product** | Creed 5 |
| B8 | Test uses group probe alone to infer worker running | **Test** | §Vocabulary |
| B9 | Zombie/reap assertions missing on either ADR target | **Test** | R8 |
| B10 | Layer A row claims ordering proof without recorder | **Test** | §Ordering honesty |

---

## R8 (cross-target)

Both ADR targets when ubuntu + macos-14 CI green on H101-218 proof commit:

- Integration T1–T6 (mode B) on **each** target
- Native N1–N6 + decoy negative on **each** target
- Platform zombie table applied per target (not one helper that only works on Linux)

---

## Scope — one card

**One card (H101-218)** if:

- Serialized lifecycle in `internal/adapters/process/`
- `Start` path no longer unconditionally reaps the leader (`cmd.Wait` trap fixed)
- Layer A + Layer B tests as above
- Claudio H101-213 fixture/Start rework in the **same commit series**

**Stop and report** if Darwin cannot implement `waitid`/`WNOWAIT` binding — must hit N6
(`RecoveryRequired`), not ship Linux-only kill.

**Does not discharge:**

- Item 2 full verdict (separate Kelly pass after commit)
- Item 4 crash recovery matrix
- Item 3 budget enforcement

---

Authored by Kelly (QA), H101-217.
