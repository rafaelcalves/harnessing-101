# H101-220 — Linux-native Stop binding: CI sufficiency ruling (2026-09-22)

Kelly QA. Answers whether **ubuntu CI (`linux/amd64`)** can discharge the
linux-native leg of Stop identity binding and natural-exit membership evidence
(H101-217 R8, H101-224 R8), so Creed's H101-227 falsifiable condition has a
real evidence class — not a design-doc promise.

**Authority:** [`h101-217-stop-identity-binding-acceptance-spec.md`](h101-217-stop-identity-binding-acceptance-spec.md);
[`h101-224-natural-exit-finalization-acceptance-spec.md`](h101-224-natural-exit-finalization-acceptance-spec.md);
[`h101-226-native-membership-dependency.md`](../architecture/h101-226-native-membership-dependency.md);
[`phase3-exit-criteria.md`](phase3-exit-criteria.md) platform rule (ubuntu = native
`linux/amd64` evidence).

**Does not:** item 2 full verdict; Kevin's ordering-flake report; acquire or pin
`x/sys` (Darwin leg).

---

## Verdict

**SUFFICIENT WITH LIMIT** — ubuntu CI is the correct and sufficient discharge
channel for the **linux/amd64** native leg. Kevin's lack of a local linux runner
is **not** an evidence-class gap. A hand-written manifest or ad hoc linux box is
**not** required for this leg.

---

## Guarantees CI can establish (linux/amd64)

When the H101-218 handover commit is **green on `ubuntu-latest`** with the rows
below, the linux-native leg is **discharged**:

| Evidence class | Rows | What green CI proves |
| --- | --- | --- |
| Stop binding — integration | H101-217 T1–T6 (mode B) | End-to-end Stop under real ownership; zombie-before-reap; worker survives parent exit; `stop-run` → `Exited` |
| Stop binding — native | H101-217 N1–N6 + decoy | Per-signal recheck, authority-before-reap ordering, stale-callback refusal, bounded forced path, `RecoveryRequired` on binding failure; unrelated pid untouched |
| Natural exit — native | H101-224 B1–B3 | No `other_members_absent_confirmed` while worker lives; positive absence before reap; uncertain membership → `natural_reap_refused_uncertain`, no speculative reap, Stop still succeeds |
| Platform zombie table | H101-217 §Platform-native | `/proc/<pid>/status` `State: Z` for TERMINATION; ESRCH/absent for REAPING — applied on the runner, not cross-compiled |

**Creed H101-227 falsifiable condition (linux half):** satisfied when CI shows
every **coded** membership failure or injection resolves to
`other_members_uncertain` / `natural_reap_refused_uncertain` with binding retained
— never `other_members_absent_confirmed` on leader-alone, never `child_reaped` on
the uncertain-only path. A green run is falsifiable evidence; a design doc is not.

---

## Non-guarantees (limits — Stanley H101-226, carried)

These are **exclusions**, not open gaps blocking the sufficiency ruling:

1. **Observation, not proof.** Linux `/proc` enumeration supports the predicate; it
   does not globally prove absence under all kernel races. Readable `/proc` ≠
   race-safe negative.
2. **Repeated empty scans are not confirmation.** The adapter must not treat scan
   emptiness alone as `other_members_absent_confirmed` when evidence is partial.
3. **Capability ≠ discharge.** Adding `membership_linux.go` or passing compile does
   not waive the CI rows above.
4. **Cross-compilation is not native evidence.** `GOOS=linux` tests run on darwin do
   not discharge this leg.

Wrongful reap on a membership race **voids** Creed's approval — that is a product
failure found by CI or review, not a reason to demand a second evidence channel
beyond native ubuntu CI.

---

## What would be insufficient (named for clarity)

| Approach | Why insufficient |
| --- | --- |
| Design doc or code review only | Creed condition is falsifiable against runs, not prose |
| Darwin-only local runs | Does not discharge `linux/amd64` |
| Cross-compiled `GOOS=linux` on macOS | Not native per ADR platform rule |
| Layer A integration only | H101-217 B1 — ordering and membership predicate need Layer B |
| Package-scoped green only | Full-suite obligation remains for handover commit (ordering flake is Kevin's separate verdict, not a CI-channel defect) |

---

## Relationship to item 2 verdict

H101-220 settles **evidence class** for the linux leg. Item 2 discharge still
requires Kelly acceptance **after** the handover commit is green on **both**
`ubuntu-latest` and `macos-14` with all H101-217 + H101-224 + H101-209 rows,
and after Kevin's report on `TestSupervisor_StopOrderingEscalationAndDecoy` under
full-suite `-count=2`.

---

## Bar queue (OUTPUT 2 — god request)

Order for remaining Phase 3 bars after item 2 verdict:

| Order | Item | Prerequisite |
| --- | --- | --- |
| 1 | **Item 3** — hard budget (elapsed + token) | Item 2 verdict; item 1 Layer A satisfied |
| 2 | **Item 6** — background hosting / UI detach | Item 1 Layer A satisfied; item 3 bar informs shared fixture/clock patterns |
| 3 | **Item 8** — trust-boundary disclosure (CF1–CF3 + item 2 tree limit) | Item 2 tree-scope text final (H101-130); item 1/10 standing fixed for CF2 wording |
| 4 | **Item 7** — adaptercontract UI-09+ | **Stanley UI map** (ADR 0003 amendment) carded first; items 3/6 bars inform UI-11/UI-12 rows |

H101-231 deliberately deferred per god — not in this sequence.

---

Authored by Kelly (QA), H101-220.
