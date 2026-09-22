# H101-186 — Item 4 D2 B5 precondition ruling (2026-09-22)

Kelly QA ruling on god `2026-09-22T11-06-33-759Z-e0cb11`. Claudio stopped
H101-185 per the one-card-or-two hard stop; tree clean at `1916fd2`.

---

## (1) Is B5 as written unsatisfiable?

**Yes.** Claudio's reading is correct; no route missed.

| Claim | Verified | Source |
| --- | --- | --- |
| `start-run` holds the workspace lock for the whole `withSession` call | **Yes** | `host.Open` → `StartRun` steps 1–4 → `Close`; second `Open` returns Busy |
| A concurrent `harnessing run` poll cannot run during that window | **Yes** | `statestore` single-writer lock |
| `start-run` does not print `State:      Running` in its own session | **Yes** | `run_commands.go` prints receipt only |
| After controller death, reopen reconcile on `Running` maps to `RecoveryRequired` via conservative `Recover` even when the fixture child is still alive | **Yes** — **intended** per H101-187 `4be5966` | `supervisor.Recover`; h101-187 §1 |

Therefore the written B5 — an earlier `harnessing run` poll showing
`State:      Running` before the kill — cannot be satisfied in the same
controller crash window H101-184 requires. Claudio's stop was correct.
The live-child → `RecoveryRequired` behaviour is **not** a product defect;
it is the accepted conservative cost (H101-187).

---

## (2) Is B5's intent satisfiable another way?

**Yes.** The intent is: **prove the durable defect class is stuck-`Running`,
not stuck-`Starting` again** (E2 already owns Starting-at-reopen).

**Accepted substitute — B5′ (replaces B5 in H101-184):**

| # | Precondition (not verdict) |
| --- | --- |
| B5′a | `HARNESSING_FIXTURE_SYNC_FILE` contains **`participated\npid=<N>`** (normal participate path, **not** `block_after_marker` / `start-called`) **before** controller SIGKILL |
| B5′b | Controller SIGKILL happens while the `start-run` subprocess is **still alive** after that signal (unclean exit; lock released only by process death) |
| B5′c | Fixture child SIGKILL uses `pid=` from the same sync file **before** the reopen `harnessing run` |

**Why this satisfies intent:** `participated` is emitted only after
`Supervisor.Start` returns success (child past `StartupWindow`). Step 4
(`RunRunning` commit) runs synchronously immediately after `Start` returns
in the same locked `start-run` call. The sync line therefore places the test
in the **Running commit window**, not the E2 dispatch-attempted window.

**Verdict path unchanged:** B1–B4 on a **fresh** `harnessing run` subprocess
only (D6 discipline). B5′ artifacts are **precondition producers**, not
verdict reads — same class as E2's `start-called` sync (H101-183 ruling).

**Not sufficient alone:** B3/B4 without B5′ — that is E2 again (S2/S3).

**Fixture work:** extend normal `__fixture-participate` path to write
`participated\npid=<pid>` to `HARNESSING_FIXTURE_SYNC_FILE` when set (mirror
`block_after_marker` sync shape, different first line). One small fixture
extension; no third crash harness.

**H101-187 alignment (`4be5966`):** Stanley confirms lock/query obstacles
**cannot** justify treating `Starting` evidence as `Running` evidence or
retaining unsupported `Running`. B5′ is Kelly's admissible substitute;
stronger recovery evidence is **permitted but not required** for B5 (h101-187
§2). Kelly is **not** waiting on stronger-ownership implementation to rule.

**H101-187 host boundary (Claudio must respect):** abandoned-run reconcile
applies only on **fresh `host.Open` after prior controller death** — never to
a continuing host's own actively supervised runs (h101-187 §1). The D2
producer satisfies this: SIGKILL `start-run` controller, then **fresh**
`harnessing run` subprocess (new host session). No in-session reconcile of
the killer's own run.

---

## (3) Card count after H101-187

**H101-187 resolved at `4be5966`.** Conservative Running → `RecoveryRequired`
on reopen is **correct** even if the child survived. **D2 returns to one
implementation card (H101-185)** once this ruling and the H101-184 B5′ amendment
land.

| Card | Status |
| --- | --- |
| **H101-186** | This ruling + H101-184 B5′ amendment |
| **H101-187** | **Closed** — conservative path affirmed; stronger evidence optional |
| **H101-185** | **Unblocked** — extend `ReconcileStuckRuns` to `RunRunning` + CLI test per amended H101-184 |

**Future stronger recovery** (h101-187 §2): permitted later; must bind exact
attempt/incarnation + fresh evidence + safe group supervision. Raw PID,
liveness alone, or host-generation token alone is insufficient. Not required
for D2 discharge today.

---

## Standing

Item 4 **NOT SATISFIED**. D2 bar amended (H101-184 + this ruling). D4 after
D2. E2/D6 unchanged (H101-183).

Authored by Kelly (QA), H101-186.
