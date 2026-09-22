# H101-235 — UI-08 fixture disposition after throwaway `stop-run` forward (2026-09-22)

Kelly QA. **Fixture/gate disposition** — Stanley's StopRun ruling (carded to Kevin as H101-234) assigned this to Kelly, not
Stanley. God relay `2026-09-22T16-15-00-000Z-kel04`.

**Authority:** Stanley — throwaway `stop-run` must decode `api.StopRunRequest` and
forward to `session.StopRun`; Phase 2 surface is **not** permanently outside Phase 3
`StopRun`. ADR 0003 H101-128 extension (line 124): Phase 2 blanket `Unsupported`
fixture becomes explicit **capability-disabled hosting**, not a requirement on
supervision-enabled product.

**Stanley's constraint to carry:** do not preserve an obsolete `Unsupported`
assertion by suppressing enabled forward behaviour.

**Does not:** implement the forward (Kevin H101-234); item 2 verdict; UI-13 rows.

---

## Verdict

**Amend UI-08.** Remove `stop-run` from the supervision-enabled blanket
`Unsupported` row. Add a **capability-disabled hosting** negative for `stop-run`
only. Keep `set-run-budget` on the existing row until item 3 lands.

---

## What changes

| Surface | Before | After |
| --- | --- | --- |
| `TestUI08_Phase3OperationsUnsupported` | `stop-run` + `set-run-budget` → `Unsupported` on default (supervision-enabled) host | **`set-run-budget` only** — still `Unsupported` via `frontendSession` stub until item 3 |
| `stop-run` on default host | Expected `Unsupported` | **Removed from this test** — forward reaches `Engine.StopRun`; honest codes are `NotFound` / `Denied` / `Conflict` / success, **not** blanket `Unsupported` |
| New row | — | **`TestUI08_CapabilityDisabledHosting_StopRunUnsupported`** |

---

## Capability-disabled hosting fixture (new — QA-owned)

**Purpose:** preserve UI-08's honest “advertise unavailable supervision” semantics
without blocking Phase 3 forward on the real host.

| # | Requirement |
| --- | --- |
| F1 | Open a workspace host **without** `ProcessSupervisor` wired (`engine.supervisor == nil` — same condition as `engine.go` `StopRun` unsupported path: `"no process supervisor is configured for this host"`) |
| F2 | Bind throwaway adapter through the **same** `assembly.WithSession` / `FrontendSession` path as other UI rows — no adapter-local refusal |
| F3 | Invoke `stop-run` with valid JSON payload (`RequestID`, `RunID`) |
| F4 | Assert `ErrUnsupported` with **non-empty detail** on throwaway surface |
| F5 | **Both adapters** not required for this row — throwaway-only negative is sufficient; CLI `stop-run` on capability-disabled host is a separate composition concern and is **not** UI-08 scope |

**Explicit non-goal:** do **not** reintroduce throwaway-local `Unsupported` for
`stop-run` on supervision-enabled hosts to keep this test green.

---

## What stays unchanged

| Item | Reason |
| --- | --- |
| `set-run-budget` in `TestUI08_Phase3OperationsUnsupported` | Still `unsupportedPhase3` on `frontendSession` until item 3 budget bar |
| `TestUI08_DetachedViewMutationProbe` | Detachment semantics unchanged |
| StartRun off UI-08 | H101-144 — UI-13 / item 1 owns enabled start |
| UI-13 (future item 7) | Enabled-host `StopRun` success / receipt path — **not** this amendment |

---

## Gate

After Kevin's H101-234 forward lands:

1. `go test ./internal/adaptercontract/ -run TestUI08 -count=1` green
2. Amended `TestUI08_Phase3OperationsUnsupported` must **fail** if `stop-run` is
   re-added without capability-disabled fixture
3. New capability-disabled test must **fail** if throwaway suppresses forward on
   enabled host (regression guard per Stanley)

---

## Decision table

| # | Observation | Defect class | Fix |
| --- | --- | --- | --- |
| U1 | UI-08 expects `stop-run` `Unsupported` on supervision-enabled host | **Test** | Remove from blanket row; add F1–F4 |
| U2 | Throwaway manufactures `Unsupported` instead of forwarding | **Product** | Stanley H101-233 — forward |
| U3 | Capability-disabled negative missing after forward | **Test** | F1–F4 |
| U4 | Enabled `stop-run` success claimed from UI-08 | **Test/evidence** | UI-13 / item 7 — wrong row |

---

Authored by Kelly (QA), H101-235.
