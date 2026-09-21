# H101-107 — Phase 2 exit item 3 ruling (2026-09-21)

Kelly QA ruling on whether the cross-adapter contract suite (`internal/adaptercontract/`,
`e565b82`) is sufficient evidence to discharge Phase 2 exit **item 3** (swappability /
ADR 0003 UI-01..08 on both adapters).

**Independent review cited:** Stanley H101-91 stage 2 —
[`docs/architecture/h101-91-stage2-expectation-traceability.md`](../architecture/h101-91-stage2-expectation-traceability.md)
(`c77783f`). Ruling there: **partially traceable**; seven gaps **G1–G7** named. This
document does not re-derive his traceability work.

**G5 resolution cited:** Kevin H101-108 — `1fcec3a` (engine enforces B:24 on
CreateTask) + `8da96de` (remaining test fixtures). CI green run `35604137543`.
Architectural shape pending Stanley H101-109; G5 closure here is on **evidence**
grounds only (see §G5 amendment).

**Contract sources:** ADR 0003, `boundaries.md`, H101-53 assertion table in
[`definition-of-done.md`](definition-of-done.md).

---

## Verdict

**SATISFIED WITH LIMIT** — the suite is structurally complete and running green on
both adapters in CI (linux + darwin-arm64, `e565b82`). It discharges item 3's
**architecture intent**: shared black-box scenarios, independent expected states,
crossover continuation, and both real drivers. Stanley's seven gaps are real limits
on what the passing run proves. **G5 is closed** as of `8da96de`; six limits
(G1–G4, G6–G7) remain.

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
| **G5** | Assigned-task creation with unregistered `engineer` in harness-only paths | **(b) Resolved — limit closed** | Kevin found CreateTask never looked up AssigneeID — B:24 was unenforced in the engine for every adapter, not a harness-only quirk (`1fcec3a`). Harness UI-06/07 paths now register `engineer` before create; `cmd/harnessing` and `internal/host` fixtures aligned (`8da96de`). `TestCreateTask_UnregisteredAssigneeNotFound` proves NotFound and no commit. Harness-only observer/detach paths **may** be credited for assignee/registry rules again. **Pending H101-109:** architectural shape (NotFound code, empty-assignee carve-out, other agent-ID references) — would reopen G5 only if Stanley overturns the fix. |
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
- **G1** exact CLI rendering as universal contract.
- Item 2 crash restart, H101-104 cross-target ack, A2 OutcomeUncertain fault injection (open per `AMBIGUITIES.md`).

---

## G5 amendment (H101-110, 2026-09-21)

**Limit closed.** Verified at `8da96de`:

- `engine.go:127–130` — non-empty `AssigneeID` looked up via `findAgent`; NotFound if absent.
- `engine_test.go:161–177` — `TestCreateTask_UnregisteredAssigneeNotFound` fails first, then passes.
- `ui06_test.go`, `ui07_test.go` — `RegisterAgent` before every `CreateTask` on harness paths.
- `go test ./...` green locally.

**Item 3 verdict unchanged:** still **SATISFIED WITH LIMIT** — G5 was the only gap that blocked
harness-path evidence; the other six documented limits stand. No upgrade to unconditional
satisfaction.

**H101-109 dependency:** this amendment does not pre-judge Stanley's architecture ruling. G5
closed on QA evidence that B:24 is now enforced and the harness no longer succeeds on
unregistered assignees. If H101-109 overturns the engine change, G5 reopens.

---

## Criteria-owner lesson (H101-110)

H101-107 offered G5 as two branches: register `engineer` in the harness, **or** make CreateTask
reject an unregistered assignee. That framing hid the diagnosis inside the remedy.

The cheap branch — register in the harness — would have made the suite green while leaving B:24
unenforced in the engine. Every adapter would still accept unknown assignees; the contract suite
would have stopped surfacing a product bug and matched wrong behaviour instead.

**Lesson:** when a gap is offered as "fix the test or fix the product," name which branch is a
**test correction** and which is a **product defect**, and do not leave both equally weighted.
The criteria owner should state the expected finding before the implementer chooses the cheaper
path. Kevin was correctly told not to assume the harness fix; reading `engine.go` showed the
engine never checked — branch (b) was the real answer, and branch (a) was only valid as
harness alignment **after** the engine enforced B:24.

---

## Open cards from this ruling

None. G5 closed under H101-108.

---

## Summary for god

| Question | Answer |
| --- | --- |
| Is the suite sufficient evidence for item 3, as it stands, with seven gaps? | **Yes, with limits** — satisfied with limit |
| Gaps blocking full satisfaction without qualification? | **G5 closed** (`8da96de`); G1–G4, G6–G7 accepted with documented limits |
| Cites Stanley? | `h101-91-stage2-expectation-traceability.md` (`c77783f`) |

Authored by Kelly (QA).
