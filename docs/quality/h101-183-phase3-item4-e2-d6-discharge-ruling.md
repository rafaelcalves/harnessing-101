# H101-183 — Item 4 E2/D6 discharge ruling (2026-09-22)

Kelly QA acceptance on Claudio's H101-181 at `21c782e`.

**Verified:** `make build test vet lint fmt` green (darwin/arm64);
`TestCLI_RecoveryRequiredAfterRealStartRunCrash` PASS ×3 under `-race`.
No `engine`, `host.Open`, or `statestore` references in `e2_d6_test.go`.

---

## (1) E2 and D6

| Row | Verdict |
| --- | --- |
| **E2** (`RecoveryRequired` visible through product) | **SATISFIED** at `21c782e` on in-scope targets |
| **D6** (CLI not host/engine/store) | **SATISFIED** |

Fresh `harnessing run` subprocess after SIGKILL on real `harnessing start-run`
controller post `start-called` sync proves the user read path.

---

## (2) R-row coverage claim

**Accurate.** Not overstated.

| Row | Disposition | Evidence |
| --- | --- | --- |
| R1 | Avoided | No engine/host/store reads in test |
| R2 | Avoided | No `GetSnapshot` |
| R3 | Covered | A4: no `State:      Starting` after reconcile |
| R4 | Covered | A2: `State:      RecoveryRequired` |
| R5 | Covered | A2 exact `printRun` spacing |
| R6 | Covered | Waits for `start-called` sync artifact before SIGKILL |
| R7 | Covered | SIGKILL on `start-run` controller; no graceful `Close` |
| R8 | See below | |
| R9 | Avoided | Test exists and PASS; E2 not claimed from engine-only path |

H101-182 assertions A1–A5: all present in `e2_d6_test.go`.

**Sync file note:** fixture uses `HARNESSING_FIXTURE_SYNC_FILE` because the
supervisor captures child stdout — same H101-161 window, not a weaker crash.
Acceptable adaptation; documented in `fixture_participate.go`.

---

## (3) R8 cross-target limit

**SATISFIED on both ADR targets** when CI `build-test-lint` (ubuntu) and
`darwin-arm64` (macos-14) jobs are green — same pattern as Layer A CLI tests.
Test skips only Windows (H101-103 out-of-scope), not an in-scope target.

Local verification this session: **darwin/arm64 only**. Kelly has not observed
a CI run number for `21c782e`; discharge on linux/amd64 rests on the standard
CI job pattern, not a manifest disclosure.

---

## Item 4 standing (partial)

E2 and D6 **discharged**. Item 4 **still NOT SATISFIED** — D2, D4, and
remaining rows open per H101-180 order.

---

## Next card

**Write D2 bar first**, then dispatch — same sequencing as item 1 and E2/D6.
Do not dispatch Running-with-dead-child implementation until criteria owner
fixes the acceptance spec.

Authored by Kelly (QA), H101-183.
