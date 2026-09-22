# H101-232 — Ordering-flake ruling on H101-217 lifecycle rows (2026-09-22)

Kelly QA. Answers whether Kevin's `-count=2` whole-suite report discharges
doubt on the three load-bearing lifecycle rows encoded in
`TestSupervisor_StopOrderingEscalationAndDecoy` (N1 per-signal recheck, N2
close-before-reap, N5 bounded grace + forced path when leader ignores SIGTERM).

**Authority:** [`h101-217-stop-identity-binding-acceptance-spec.md`](h101-217-stop-identity-binding-acceptance-spec.md);
Kevin's report (god relay `2026-09-22T15-55-00-000Z-kel03`).

**Does not:** item 2 verdict; H101-225 detach cleanup; commit the handover tree.

---

## Verdict

**Bar rows N1, N2, N5 STAND. The escalation test assertion NEEDS AMENDING.**

Kevin's environmental-timing discharge is **not accepted**. His volunteered
caveat is **correct**: the exact-order assertion can be too strict for a
legitimately dual-path design — but the failure mode is **fixture brittleness
on the forced path**, not proof that the recorder is sound under contention.

---

## What I checked

| Check | Result |
| --- | --- |
| Kevin's report (3× `go test ./... -count=2`, 2× `-race -count=2`) | Not re-run whole-suite; accepted as described |
| `go test ./internal/adapters/process/ -run TestSupervisor_StopOrderingEscalationAndDecoy -count=20` (darwin) | **3/20 failed** — flake reproduces **without** full-suite load |
| Failure shape | `got` = `[stop_claimed pre_sigterm_check sigterm_sent leader_terminated_observed authority_closed child_reaped]` — **graceful path** (no `pre_sigkill_check` / `sigkill_sent`) |
| `supervisor.go` Stop() | When leader exits during grace wait, **skipping SIGKILL is correct** (`alreadyExited` branch) |
| Package-only `-race -count=3` on `internal/adapters/process/` | Green on this run |

---

## Row-by-row

| Row | Stands? | Ruling |
| --- | --- | --- |
| **N1** — distinct `pre_sigterm_check` / `pre_sigkill_check`, each before its signal | **Yes** | Creed 1 unchanged. Forced-path proof still required; must not be asserted on a run that accidentally took the graceful path. |
| **N2** — `authority_closed` before `child_reaped` | **Yes** | Observed in both passing and failing runs. Subsequence check sufficient; exact global order not required for N2 alone. |
| **N5** — bounded grace + forced path when leader ignores SIGTERM | **Yes, bar only** | **Test implementation fails the bar today**: `trap "" TERM; exec sleep 100` is not deterministic — SIGTERM can arrive before trap installs, so the test sometimes proves graceful Stop instead of escalation. |

**N3 / N4** (no post-release signals; stale second Stop) are covered by
`TestSupervisor_StopGracefulPathSkipsForcedSignal` — **unchanged**, not part of
this flake.

---

## Kevin's verdict — partial accept

| Claim | Kelly ruling |
| --- | --- |
| Whole-suite `-count=2` failures are often unrelated tests (H101-225, crash-prefix) | **Accepted** — does not discharge this row |
| No `-race` DATA RACE on recorder | **Accepted** — supports "not a shared-state corruption bug" |
| Environment / CPU contention under doubled load | **Rejected as root cause** — flake reproduces in **package-only** `-count=20` |
| "No evidence of ordering defect" ≠ "proven sound" | **Accepted** — Kevin's honesty is right; his conclusion overreaches |
| Third possibility: assertion too strict for dual-path design | **Accepted** — but remedy is **fixture + assertion amend**, not weaker evidence |

---

## Required amendment (before handover commit)

**Do not** discharge on a dedicated full-suite `-count=2` pass alone.

**Do** amend `TestSupervisor_StopOrderingEscalationAndDecoy`:

1. **Fixture gate (new):** leader must attest SIGTERM-immunity **before** Stop is
   called — same class as H101-221 `worker_ready\n` (e.g. append `term_ignored\n`
   to a sync file only after `trap` + `exec sleep` are in effect). Poll ≤50ms;
   fail after 30s. Stop must not run until attestation present.
2. **Assertion shape:** replace exact `DeepEqual` on the full event list with:
   - **Mandatory on this scenario:** `pre_sigkill_check` and `sigkill_sent` both
     present (else **FAIL** — wrong path for an escalation row);
   - **Subsequence pairs:** `pre_sigterm_check` before `sigterm_sent`;
     `pre_sigkill_check` before `sigkill_sent`; `authority_closed` before
     `child_reaped`;
   - Decoy probe unchanged.
3. **Gate:** `go test ./internal/adapters/process/ -race -count=5` green on
   **each** ADR target at handover — package scope is sufficient for these rows;
   full-repo `-count=2` is **not** a discharge requirement for N1/N2/N5 (H101-225
   and other flakes are separate cards).

`TestSupervisor_StopGracefulPathSkipsForcedSignal` remains the graceful-path
negative for N3/N4 (no sigkill events).

---

## What a dedicated pass would need (if amendment deferred — not recommended)

Prove **10 consecutive** package-only runs of
`-run TestSupervisor_StopOrderingEscalationAndDecoy -count=5 -race` on **both**
targets with **zero** graceful-path shape (`sigkill_sent` absent). That would
still not fix the bar — it would only hide fixture brittleness. **Amendment is
the correct remedy.**

---

Authored by Kelly (QA), H101-232.
