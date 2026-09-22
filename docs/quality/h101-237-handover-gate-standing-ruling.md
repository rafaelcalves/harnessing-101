# H101-237 — Handover gate standing after flake-family hypothesis (2026-09-22)

Kelly QA. Answers whether H101-232's package `-race -count=5` handover gate
changes given Kevin's hypothesis that three whole-suite-only failures share a
**wall-clock contention** shape (fork/exec, signal, fs-metadata deadlines), not
shared mutable state (`-race` silent).

**Authority:** [`h101-232-ordering-flake-ruling.md`](h101-232-ordering-flake-ruling.md);
god relay `2026-09-22T19-05-00-000Z-kel09`.

**Does not:** item 2 verdict (waits on CI); assign H101-237 fix owner; reverse
H101-232 amendment.

---

## Verdict

**GATE STAYS AS RULED.** No new obligation on the handover gate. Kevin's
hypothesis is **accepted as a limit on what the gate proves**, not as grounds to
reopen it or add full-suite `-count=2` / `-p 1` whole-repo passes.

---

## What the gate proves (unchanged)

| Gate | Proves | Does not prove |
| --- | --- | --- |
| `go test ./internal/adapters/process/ -race -count=5` per ADR target | H101-217 N1/N2/N5 + H101-224 membership rows on **native** runner under **package** load | Whole-repo parallel load stability |
| `term_ignored\n` + subsequence assertion (H101-232 amend) | Escalation path is deterministic under package load (20/20 post-fix) | Immunity to all future wall-clock flakes |

Removing full-suite `-count=2` was intentional: it mixed **separate cards**
(H101-225 detach, crash-prefix, recovery crash) with lifecycle row discharge.

---

## Kevin's hypothesis — row by row

| Claim | Ruling |
| --- | --- |
| Three failures share short fixed real-time windows around OS primitives | **Accepted as hypothesis** — plausible; god's `-p 1` discriminating run is the right next evidence (inform H101-237, not this gate) |
| Not shared-mutable-state races (`-race` silent) | **Accepted** — different failure class from H101-232's original recorder concern |
| Package-only passes by **excluding contention**, not fixing timing assumption | **Accepted as limit** — does not invalidate the gate's purpose |
| Next test may "work alone, lose under load" | **Accepted as exclusion** — tracked under H101-237; not item 2 handover gate scope |

**Does not reopen H101-232 root-cause ruling.** Package-only `-count=20` reproduced
the escalation failure **before** the `term_ignored` fix — that was trap-install
brittleness, not whole-suite contention. Post-fix 20/20 closes **that** defect
class for that row.

---

## Why the gate is not weakened

The gate was never claimed to prove whole-suite stability. It proves **named
lifecycle rows** on the **designated native channel** (per H101-220). Cheapening
would be adding full-suite `-count=2` back without a row that requires it.

**Item 2 verdict** still uses **CI green on both `ubuntu-latest` and
`macos-14`** (H101-220, Creed H101-227) — that is the integration gate in the
real runner environment, not local `go test ./...` at default parallelism.

| If… | Then… |
| --- | --- |
| CI green both targets | Item 2 verdict proceeds on CI evidence |
| CI flakes on wall-clock class | Verdict **blocked** — fix or limit on that row; H101-237 informed |
| God's `-p 1` run clean, default parallel flaky | Supports contention hypothesis for **H101-237**; does not change handover gate |

---

## What H101-237 owns (not the handover gate)

- Flake family pattern: fixed short deadline + real OS primitive + whole-suite load
- Kevin's heuristic: fourth flake → look for same shape first
- Future bar work (item 3+) may add suite-level policy if rows need it — **not now**

**Explicitly not required on handover:** full-repo `-count=2`, `-p 1` whole suite,
or longer package `-count` than 5.

---

Authored by Kelly (QA), H101-237 gate standing.
