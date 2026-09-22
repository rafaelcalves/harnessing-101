# H101-236 — UI-08 capability-disabled fixture acceptance (2026-09-22)

Kelly QA. **Test acceptance** for H101-235 F1–F5 after Stanley's constructor
ruling (god relay `2026-09-22T18-20-30-000Z-kel07`). Stanley owns constructors;
Kelly owns whether the fixture satisfies the bar.

**Authority:** [`h101-235-ui08-capability-disabled-fixture-disposition.md`](h101-235-ui08-capability-disabled-fixture-disposition.md);
Stanley — `OpenWithoutSupervisor` / `WithSessionWithoutSupervisor` accepted with
mode-contract constraints.

---

## Verdict

**ACCEPT WITH LIMIT** — Kevin's `TestUI08_CapabilityDisabledHosting_StopRunUnsupported`
satisfies F1–F5 as written. The exact-detail pin is **accepted** as the
regression discriminator for gate 3; **no stronger discriminator required**.

---

## F-row acceptance

| Row | Verdict | Evidence |
| --- | --- | --- |
| **F1** | **SATISFIED** | `host.OpenWithoutSupervisor` → `open(..., withSupervisor=false)` → `engine.supervisor == nil` |
| **F2** | **SATISFIED** | `assembly.WithSessionWithoutSupervisor` → real `FrontendSession` → throwaway forward (no adapter-local refusal) |
| **F3** | **SATISFIED** | Valid `stop-run` JSON via `adapter.Handle` |
| **F4** | **SATISFIED WITH LIMIT** | `ErrUnsupported` + detail pinned to engine path string — see limits below |
| **F5** | **SATISFIED** | Throwaway-only; CLI out of scope per original disposition |

**Constructor comments:** must state **supported internal composition mode with
supervision absent** (Stanley's contract), not QA-only framing — **acceptance
conditional on Kevin's comment fix** before land; does not block this ruling.

---

## Exact-detail pin — disposition (Stanley's narrowing)

| Question | Ruling |
| --- | --- |
| Does `wantDetail == "no process supervisor is configured for this host"` satisfy F4 / gate 3? | **Yes, with limit** |
| What it proves | Catches adapter-local `Unsupported` with **different** detail (Kevin's injection — the regression H101-235 gate 3 names) |
| What it does **not** prove | Generic call-origin proof — an adapter could copy the same string, or refuse only on enabled hosts |
| Public error-text contract? | **No** — test-local constant only; failure message cites `engine.go` provenance; not a stability guarantee for callers |
| Item 7 / UI-13 credit? | **None** — enabled-host `stop-run` success and crossover remain separate rows |

**No different discriminator required.** White-box hooks or duplicate-path
instrumentation would over-engineer a UI-08 negative whose job is honest
advertisement on capability-disabled hosting, not forward-path forensics.
Stanley's limit is recorded as an **exclusion**, not an open gap.

---

## Gate (unchanged from H101-235)

| # | Requirement | Status at review |
| --- | --- | --- |
| G1 | `go test ./internal/adaptercontract/ -run TestUI08` green | God verified `-race -count=5` on held tree |
| G2 | Blanket row must not re-add `stop-run` | Satisfied in `ui08_test.go` |
| G3 | Capability-disabled test fails on adapter injection | Kevin proved: different-detail injection passed weak check, fails exact pin |

---

## Decision table (additions)

| # | Observation | Defect class | Fix |
| --- | --- | --- | --- |
| U5 | Exact detail promoted to public API contract | **Process** | Keep test-local; no doc freeze |
| U6 | Test claims enabled-host forward proof | **Test** | Out of scope — UI-13 |
| U7 | Adapter copies engine detail string without forward | **Product** | Not caught by this row — acceptable limit; enabled-host rows cover real Stop |
| U8 | Constructor comments say "QA fixture only" | **Docs** | Stanley constraint — fix before commit |

---

Authored by Kelly (QA), H101-236 acceptance.
