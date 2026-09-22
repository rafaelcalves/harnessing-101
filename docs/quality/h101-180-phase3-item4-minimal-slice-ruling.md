# H101-180 — Item 4 minimal slice acceptance (2026-09-22)

Kelly QA acceptance ruling on Kevin's H101-170 slice at `2b76d80`. Stanley
H101-169 assigned acceptance; Kevin explicitly does **not** claim item 4.

**Verified:** `make build test vet lint fmt` green (darwin/arm64); four new
tests PASS under `-race`:
`TestReconcileStuckRuns_TransitionsStuckStartingToRecoveryRequired`,
`TestReconcileStuckRuns_LeavesAnUnrelatedRunsOperationAlone`,
`TestReconcileStuckRuns_NativeCrashBetweenMarkerAndObservation`,
`TestSupervisor_RecoverIsAlwaysStructuredUnknown`.

**Verdict: PARTIAL ACCEPTANCE of H101-170.** Slice is **accepted** as the
authorised minimal crash-reconciliation increment. **Item 4 is NOT SATISFIED.**

---

## (1) What discharges vs what stays open

### Stanley H101-169 five required evidence items

| # | Requirement | Verdict | Named evidence |
| --- | --- | --- | --- |
| E1 | Reconciliation before new-start admission | **SATISFIED** | `host.Open` → `engine.ReconcileStuckRuns` once (`internal/host/host.go`); `findActiveRunForAgent` blocks before intent commit (`engine.go`) |
| E2 | Durable `RecoveryRequired` visible **through the product** | **NOT SATISFIED** | No CLI subprocess test; `GetRun` on engine only. `harnessing run` after real crash not asserted (item 4 D6 class) |
| E3 | New run/request for same agent refused, no child | **SATISFIED** | `TestReconcileStuckRuns_TransitionsStuckStartingToRecoveryRequired` (Conflict, `sup.calls == 0`); native test same |
| E4 | Block persists across another reopen | **SATISFIED** | `TestReconcileStuckRuns_NativeCrashBetweenMarkerAndObservation` second `statestore.Open` |
| E5 | Unrelated eligible agent not permanently blocked | **SATISFIED** | Both reconcile tests: agent-b `StartRun` succeeds after agent-a blocked |

### Item 4 decision table (phase3-exit-criteria.md)

| Row | Verdict | Notes |
| --- | --- | --- |
| D1 | **Partial** | Same-agent duplicate spawn blocked (`Conflict`, no `supervisor.Start`); not same-`runID` redispatch proof |
| D2 | **Open** | Slice reconciles **Starting** only, not **Running** with dead child |
| D3 | **Partial** | `RecoveryRequired` surfaced for stuck-**Starting** at reopen (`2b76d80`); not full unknown-identity matrix |
| D4 | **Open** | OutputJournal untouched |
| D5 | **Satisfied** | Native SIGKILL after marker proof, before observation (`crash_reconcile_native_test.go`) |
| D6 | **Open** | No `harnessing run` / CLI query proof |

### Item 4 narrative bullets

| Bullet | Verdict |
| --- | --- |
| Next invocation surfaces run state honestly | **Partial** — native reopen + engine `GetRun`; not CLI |
| `Recover` → identity or `Unknown` | **Satisfied** — `TestSupervisor_RecoverIsAlwaysStructuredUnknown`; structured `ErrRecoveryRequired`, no PID probe |
| `RecoveryRequired` + refuse duplicate spawn | **Partial** — Starting-at-reopen path; no unblock; agent-scoped block |
| Output gaps reported, not fabricated | **Open** |

---

## (2) Kevin's non-claims — correctly scoped?

**Yes.** The slice does not establish live supervision, process adoption, tree
termination, output continuity, or item 4 discharge. Kelly finds **no over-claim**
in the stated non-claims.

**Does not quietly establish less:** the D18/`findActiveRunForAgent` correction
(adding `RecoveryRequired` and `Stopping` to the per-agent block) is real product
behaviour shipped in this slice. It is **item 1 cross-impact**, correctly bundled
per H101-169 (line 71 stacks on line 43). Not item 4 discharge; regression guard
for item 1 D18 when `RecoveryRequired` becomes reachable.

**Does not quietly establish more:** no CLI surface proof (E2/D6 gap above).

---

## (3) Defensive guard on already-Succeeded operations

Kevin reported `reconcileStuckRun`'s guard on a still-open start operation cannot
be load-tested on the **same** run because Steps 2/4 always move run state and
start-operation state together today. He renamed the test to
`TestReconcileStuckRuns_LeavesAnUnrelatedRunsOperationAlone` — asserts the
**non-vacuous** cross-run claim.

**Ruling: ACCEPTABLE WITH RECORDED LIMIT.**

- Keep the defensive guard in code.
- **Record the invariant** in this ruling and criteria standing (Kelly acceptance
  only — no code edit): while Steps 2/4 move run and start-operation state in
  lockstep, the same-run Succeeded guard is unreachable; a future change that
  splits them **must** add a load-bearing test before merge.
- Do **not** delete the guard to chase a vacuous test.

---

## Item 4 standing

At `2b76d80`: **NOT SATISFIED.** H101-170 minimal slice **accepted**; next gap
for item 4 progress is **CLI-visible `RecoveryRequired` after real crash** (E2 +
D6), then Running-with-dead-child (D2), then output-gap honesty (D4).

Authored by Kelly (QA), H101-180.
