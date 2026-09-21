# H101-143 — Phase 3 exit item 6 foundation ruling (2026-09-21)

Kelly QA ruling on `18af1ec` (`harnessing serve`, file transport,
`TestRun_ServeAttachRoundTripThenDetach`). Kevin did not claim item 6
discharged; god verified test passes and make targets clean.

---

## Verdict

**Item 6 does not move.** **NOT SATISFIED** — no partial exit credit.

What landed is the **hosting foundation slice** only. Useful, correctly scoped,
insufficient for item 6 discharge.

---

## Does item 6 move?

**No.**

Item 6 requires all four steps in
[`phase3-exit-criteria.md`](phase3-exit-criteria.md) item 6:

1. Start run through CLI in continuing-host mode
2. Detach second session without stop
3. **Run remains observable as `Running`; output continues or resumable**
4. Explicit shutdown/stop still terminates

`18af1ec` proves step 2's **host lifetime** half and lock ownership (serve
survives two attach/detach cycles; `host.Open` Busy while alive; lock released
after SIGTERM). It does **not** start a run — `StartRun` is item 1, not built.
Same shape as H101-138 item 9: cannot discharge a criterion whose product
surface does not exist yet.

**Foundation evidence (H101-136 class, not item 6 verdict):**

| Proven at `18af1ec` | Item 6 step |
| --- | --- |
| Real `harnessing serve` subprocess | Prerequisite for step 1 host mode |
| Lock held; second `host.Open` rejected while serve lives | D3 negative control |
| Two sequential transport sessions; first write visible to second | Host owns state across detach |
| Detach does not end host | UI detach ≠ host shutdown |
| SIGTERM ends host and releases lock | Partial step 4 (host shutdown, not run stop) |

---

## Mailbox pump — gap or correct scoping?

**Correct scoping for this slice. Not an item 6 gap.**

Item 6 observability is about **run state and run output** (steps 3–4), not
mailbox background delivery. Decision table D2 ("no way to observe **run** after
starting CLI exits") targets `GetRun` / output resumption — not
`RecordMessagePublished` while detached.

Task/agent state observation through a second attach (as H101-136's test does via
`GetAgent`) is **supporting evidence** that the transport path works; it does
not substitute for run observability.

Stanley's continuing-host design (`h101-128-phase3-supervision.md`) expects the
serve process to run the mailbox `Deliverer` eventually — for **coordination
continuity** in background mode (Phase 2 mailbox obligation in a continuing
host). That is a **separate follow-on card** when wiring full serve lifecycle,
not item 6 discharge and not a reason to widen H101-136.

**Do not card the pump as blocking item 6 acceptance.** Card it when implementing
full serve host completeness if coordination-in-background is required before
Phase 3 exit — track under serve/host wiring, not item 6.

---

## Re-ruling when item 1 lands?

**No criteria rewrite needed.** Item 6 text already states the full proof.

When `StartRun` ships:

1. **Extend** acceptance test (or add sibling) to run the full four-step proof on
   both ADR targets — real run, detach, `Running` via shipped CLI/`GetRun`,
   output resumable (item 5 coupling), explicit stop/shutdown.
2. H101-136 foundation test **remains** as regression guard for attach/detach/lock;
   it does not discharge item 6 alone.

Until that test passes natively on linux + darwin, item 6 stays **NOT SATISFIED**.

---

## Phase 3 exit standing (item 6 only)

| Status | Meaning |
| --- | --- |
| **NOT SATISFIED** | No exit credit |
| **Foundation in progress** | `18af1ec` / H101-136 — hosting slice landed |
| **Blocked on** | Item 1 (`StartRun`) for run half of proof |

Authored by Kelly (QA).
