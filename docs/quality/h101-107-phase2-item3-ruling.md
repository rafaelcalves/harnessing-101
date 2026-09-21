# H101-107 — Phase 2 exit item 3 ruling (2026-09-21)

Kelly QA ruling on whether the cross-adapter contract suite (`internal/adaptercontract/`,
`e565b82`) is sufficient evidence to discharge Phase 2 exit **item 3** (swappability /
ADR 0003 UI-01..08 on both adapters).

**Independent review cited:** Stanley H101-91 stage 2 —
[`docs/architecture/h101-91-stage2-expectation-traceability.md`](../architecture/h101-91-stage2-expectation-traceability.md)
(`c77783f`). Ruling there: **partially traceable**; seven gaps **G1–G7** named. This
document does not re-derive his traceability work.

**Contract sources:** ADR 0003, `boundaries.md`, H101-53 assertion table in
[`definition-of-done.md`](definition-of-done.md).

---

## Verdict

**SATISFIED WITH LIMIT** — the suite is structurally complete and running green on
both adapters in CI (linux + darwin-arm64, `e565b82`). It discharges item 3's
**architecture intent**: shared black-box scenarios, independent expected states,
crossover continuation, and both real drivers. Stanley's seven gaps are real limits
on what the passing run proves; one gap (**G5**) is carded for correction before
item 3 evidence is sound on harness-only UI-06/07 paths.

Item 3 does **not** discharge item 2 (crash restart) or H101-104 (cross-target ack
survival). Out of scope here.

---

## Gap dispositions

| Gap | Stanley summary | Disposition | Reason |
| --- | --- | --- | --- |
| **G1** | Rendering treated as canonical state — exact CLI labels, stderr placement, blanket validation-cue substring ban | **(c) Accepted with limit** | Semantic requirements (delivery facts, unverified provenance, ADR 0002 disclosure) are traceable and tested on both paths where applicable. Exact punctuation, stderr stream, and substring deny-list are **CLI-local regression guards** under ADR 0003 presentation honesty — not shared cross-adapter canonical state. Throwaway UI-05 asserts domain fields, not CLI labels. **Limit:** item 3 credits presentation **meaning**, not exact CLI formatting or the deny-list as architecture. |
| **G2** | Exact event `SubjectIDs` lists and singleton cardinality | **(c) Accepted with limit** | Spec requires post-cursor observation of the committed fixture fact (B:21,38; ADR3:46), not exclusive single-subject cardinality. **Limit:** UI-06 observer tests prove the fixture task subject appears after the cursor; they do **not** prove no additional subjects may be present. |
| **G3** | Literal cursor encodings (`"0"`) and expiry on every restart | **(c) Accepted with limit** | Stale-cursor-after-restart test targets ADR3:80 / H101-70 when retained history is unavailable (fresh process, empty replay floor) — not expiry on every restart when history is retained. Literal `"0"` is bootstrap at empty workspace, not mandated cursor encoding. **Limit:** item 3 credits the **named restart stale-cursor scenario**, not universal cursor syntax or unconditional post-restart expiry. |
| **G4** | Strictly increasing raw revisions vs at-least-once delivery | **(c) Accepted with limit** | Helper is a regression guard for the H101-71 commit-ordering bug class (ADR3:80, B:48). Architecture allows at-least-once delivery and multiple events per revision. **Limit:** monotonic raw revisions in this helper are **ordering evidence**, not proof of duplicate-free or one-event-per-revision raw streams. |
| **G5** | Assigned-task creation with unregistered `engineer` in harness-only paths | **(b) Affects item 3 — carded** | `ContractHarness` UI-06/07 tests create tasks for `engineer` without `RegisterAgent`, while B:24 requires unknown IDs to fail NotFound. Adapter-driven scenarios (UI-01..05, crossover) register first. **Card:** register `engineer` in harness setup or stop asserting successful create on unregistered assignee. Until fixed, item 3 **does not** credit harness-only observer/detach paths as evidence of assignee/registry rules. |
| **G6** | Mandatory nonempty `Detail`; unsupported-vs-invalid precedence | **(c) Accepted with limit** | `Unsupported` on Phase 3 ops is traceable (ADR3:48, B:40). Nonempty detail is supplementary per B:13, not mandatory. Malformed-payload precedence for unavailable ops is unsettled in spec. **Limit:** item 3 credits **Unsupported availability** on throwaway Phase 3 surface; nonempty detail and request-shape precedence are **not** architectural discharge. |
| **G7** | Task-revision preconditions 4,5,6,7 with no stated arithmetic | **(c) Accepted with limit** | Workspace revision 14 at crossover end cut is sourced (B:109, H101-94; `AMBIGUITIES.md` A3 resolved). Per-task `-expected-revision` inputs are fixture choreography, not independently derived task-revision rules (stage-one gap, unchanged). **Limit:** cross-adapter comparison uses **workspace end state and equivalent committed cut**, not individual task-revision literals as architecture. |

---

## What item 3 discharge includes (as it stands)

| Requirement (H101-53 / ADR 0003) | Evidence |
| --- | --- |
| UI-01..04 on both adapters | Green; stage 1 + Stanley stage 1 traceability (`bd6f366`) |
| UI-05..08 on both adapters | Green; semantics partially traceable per Stanley stage 2 |
| Second adapter (throwaway), no shared dispatch | `throwawayadapter` independent JSON handler |
| Crossover on shared workspace | `TestCrossAdapter_Continuation` |
| CI on in-scope targets | `go test ./...` in linux + darwin-arm64 jobs (`e565b82`) |
| Independent expected states | `expected/spec.go` + harness snapshot for domain cuts |

## What item 3 discharge does not include

- Stanley's **coverage limits** (CLI discover supplies known task ID; some observer/detachment via harness not both adapters' production paths) — see his §Limits.
- **G5** harness paths until carded fix lands.
- **G1** exact CLI rendering as universal contract.
- Item 2 crash restart, H101-104 cross-target ack, A2 OutcomeUncertain fault injection (open per `AMBIGUITIES.md`).

---

## Open card from this ruling

| ID | Owner | Action |
| --- | --- | --- |
| **G5-fix** (dispatch as god sees fit) | Kelly / suite maintainer | Register `engineer` in `openContractHarness` UI-06/07 setup, or fail create when assignee not registered — align harness with B:24 |

No test edits in this ruling commit; card only.

---

## Summary for god

| Question | Answer |
| --- | --- |
| Is the suite sufficient evidence for item 3, as it stands, with seven gaps? | **Yes, with limits** — satisfied with limit |
| Gaps blocking full satisfaction without qualification? | **G5** carded; others accepted with documented limits |
| Cites Stanley? | `h101-91-stage2-expectation-traceability.md` (`c77783f`) |

Authored by Kelly (QA).
