# H101-188 — Item 4 D2 discharge ruling (2026-09-22)

Kelly QA acceptance on Claudio's H101-185 at `b764691`.

**Verified:** `make build test vet lint fmt` green (darwin/arm64);
`TestCLI_RunningDeadChildReconcilesToRecoveryRequired` PASS ×3 under `-race`.
No `engine`, `host.Open`, or `statestore` references in `d2_test.go`.

---

## (1) Is D2 satisfied?

**Yes — SATISFIED at `b764691`** on in-scope targets (local). Row D2:
stuck `Running` with dead child → `RecoveryRequired` on fresh `harnessing run`,
not stuck `Running`.

---

## (2) Is Claudio's coverage claim accurate?

**Yes.** Not overstated.

| Claim | Verdict | Evidence |
| --- | --- | --- |
| B1 exit 0 | **Present** | `d2_test.go` after `code != 0` check |
| B2 `Run run-1` | **Present** | stdout contains affected run identity |
| B3 `State:      RecoveryRequired` | **Present** | exact `printRun` spacing |
| B4 not `Running` | **Present** | negative assertion |
| B5′a `participated\npid=` before controller kill | **Present** | lines 62–65 before SIGKILLs |
| B5′b controller SIGKILL while alive | **Present** | after participated, `Wait` ≠ success |
| B5′c child killed by recorded pid before reopen | **Present** | child SIGKILL before controller kill |
| S2 wrong precondition | **Avoided** | B5′ enforced |
| S3 E2 window | **Avoided** | not `start-called` |
| S4 stuck Running product defect | **Covered** | B4 |
| S5 fake Exited | **Avoided** | B3 path |
| S6 graceful Close only | **Avoided** | controller SIGKILL |
| S9 continuing-host reconcile | **Avoided** | post-crash fresh `harnessing run` subprocess |

**S7 / R8:** recorded as **CI-only limit** — not claimed discharged here.
Upgrade to satisfied on both ADR targets when ubuntu + macos-14 jobs green on
`b764691`, same rule as H101-183.

---

## (3) Are the two updated expectations sound?

**Yes — both correct, not regression laundering.**

### Layer A CLI (`start_run_test.go`)

Old assertion: `harnessing run` → `State:      Running` after successful
`start-run`. **Wrong after H101-187:** a fresh `harnessing run` is a new
`host.Open`; inherited `Running` conservatively reconciles to
`RecoveryRequired`. The update matches Stanley's ruled behaviour.

**Nothing unprotected:** successful launch is still proved by `start-run` OK
receipt + `.participated` marker (R2). The test never claimed same-session
continuing-host supervision — only product-surface reads.

### Reconcile unit (`reconcile_test.go`)

Renamed to `TestReconcileStuckRuns_PreservesRunningOperationOutcome`. Old test
expected inherited `Running` to **survive** reconcile — that protected pre-H101-185
behaviour that H101-187 explicitly retired. New test asserts the **real** claim:
`Running` → `RecoveryRequired` while **Succeeded** start operation facts stay
Succeeded (`engine.go` Pending/Running-only operation rewrite). Sound.

---

## (4) Criteria standing amendment

| Row | Was | Now |
| --- | --- | --- |
| **D2** stuck `Running`, dead child | Open | **SATISFIED** at `b764691` |
| **D3** surface `RecoveryRequired` | Partial (Starting-at-reopen) | **Partial** (+ Running-at-reopen reopen path; full matrix still open) |
| D4 output gaps | Open | Open |
| **Item 4** | NOT SATISFIED | **NOT SATISFIED** (D4 open) |

E1–E5, D1 partial, D5, D6 unchanged from H101-183 standing.

---

## (5) D4 next step

**Write the D4 bar first** — same sequencing as E2/D6 and D2. Do not dispatch
engineering until Kelly publishes an acceptance spec. D4 is output-gap honesty
after crash; scope and verdict path need the same bar-before-card discipline.

---

Authored by Kelly (QA), H101-188.
