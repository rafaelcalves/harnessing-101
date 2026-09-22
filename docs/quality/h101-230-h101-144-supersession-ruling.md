# H101-230 — H101-144 supersession ruling (2026-09-22)

Kelly QA. God `2026-09-22T14-23-09-000Z-kelly`. Authority: item 1 verdict
`f5c9560` / [`h101-211-phase3-item1-discharge-ruling.md`](h101-211-phase3-item1-discharge-ruling.md);
H101-144 task notes; h101-128 H101-135 sequence (line 118 disclaimer).

---

## Answer

**(1) Discharged — close H101-144 as superseded by the item 1 verdict.**

---

## Why

H101-144 was Kevin's **implementation umbrella** for approved-profile `StartRun`, not a
separate acceptance bar. The work landed in the commit chain Kelly already closed at
`c70ac28` / `f5c9560`:

| H101-144 obligation | Where it closed |
| --- | --- |
| Approved-profile `StartRun` + Layer A fixture | `TestCLI_StartRun_LayerAParticipationFixture`; R3/D10 negatives (`5048489`+) |
| Dispatch-ordering invariant (D12) | `TestStartRun_DispatchMarkerAmbiguous_NoAutoRedispatch`; engine `afterDispatchMarkerBeforeStart` seam |
| `RunDispatchAttempted` journal | H101-157 / `TestStartRun_DispatchAttemptedEventReachesTheJournal` (`98729c9`) |
| Native crash in marker→Start window | `TestReconcileStuckRuns_NativeCrashBetweenMarkerAndObservation` — built on H101-170 card, same H101-135 window Kelly accepted for item 4 E3 |
| Layer B named tools | Owner deferrals at `c70ac28` (H101-211) |

Kevin's revert note stands: Layer A and dispatch-ordering were **discharged before**
H101-144 was re-blocked on Layer B / H101-163; H101-196/H101-211 closed Layer B by
deferral.

---

## What does **not** reopen H101-144

h101-128 line 118 lists additional implementation-evidence windows (pre-marker crash,
post-spawn-before-`Started`, marker-confirmation failure, repeated dispatcher delivery).
That paragraph **explicitly adds no acceptance claim** and does not override criterion
dispositions. Kelly never wrote item 1 rows for each window. Closing H101-144 does
**not** drop accepted work — it was never a parallel bar.

If god wants those extra windows as **named item 1 rows**, that is a **fresh card** with
Kelly rows — not H101-144 residue.

---

Authored by Kelly (QA), H101-230.
