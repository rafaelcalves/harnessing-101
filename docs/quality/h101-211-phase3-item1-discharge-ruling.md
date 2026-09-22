# H101-211 — Phase 3 item 1 discharge ruling (2026-09-22)

Kelly QA. Authority: H101-196 disposition; Kevin H101-201 at `c70ac28`; god relay
`2026-09-22T12-29-26-168Z-5e535d`.

---

## (1) Does Layer B Claude close by deferral?

**Yes — SATISFIED BY OWNER DEFERRAL at `c70ac28`.**

Verified committed `scripts/agentic-cli-manifests/claude-code.json`:

| H101-196 requirement | Present |
| --- | --- |
| `deferred: true` | Yes |
| `toolSpec` absent (schema rule) | Yes |
| `deferral.reason` names owner H101-174 decision | Yes — `managed StartRun deferred by owner H101-174` |
| `deferral.reference` H101-174 attachment-first | Yes |
| `deferral.backlog` with `managed_start_run`, `nearTerm`, `reopensWhen` | Yes |

**Not** demonstrated managed-start compatibility. **Not** registration evidence.
Codex and Cursor were already deferred; all three named tools are now
manifest-or-deferral complete (all deferral).

---

## (2) Is Phase 3 item 1 satisfied?

**Yes — SATISFIED WITH LIMIT at `c70ac28`.**

### Row standing (amended)

| Row | Verdict | Evidence |
| --- | --- | --- |
| Layer A (participation fixture) | **SATISFIED** | `TestCLI_StartRun_LayerAParticipationFixture` + R3/D10 negatives (`5048489`+); CI green both ADR targets at `20a6f8a` chain |
| Layer B Claude | **SATISFIED BY OWNER DEFERRAL** | `claude-code.json` `c70ac28`; runner `owner_deferred` exit 19 with backlog in report (Kevin verify) |
| Layer B Codex | **SATISFIED BY OWNER DEFERRAL** | `codex.json` committed deferral |
| Layer B Cursor | **SATISFIED BY OWNER DEFERRAL** | `cursor-agent.json` committed deferral |
| D5 | **SATISFIED** | unchanged `20a6f8a` |
| D18/D19 | **SATISFIED** | unchanged `5048489` |
| R3/D10 | **SATISFIED** | unchanged `5048489` |

Kelly local: `make build test vet lint fmt` green at `c70ac28`.

### The limit (honest ceiling — H101-196)

All three owner-named agentic CLIs close Layer B by **deferral**, not runner-produced
manifest. Phase 3 item 1 proves **managed `StartRun` through the product** with the
in-repo **participation fixture** (Layer A). It does **not** prove live Claude Code,
Codex, or Cursor managed-start compatibility. Claude participation under
attachment-first is **item 10**, not item 1.

---

## (3) Owner sentence — what item 1 guarantees

An **approved profile** can **`StartRun` through the shipped `harnessing` command**,
reach observable **`Running`** with **participation context** (Layer A fixture), honor
**caller scope**, **approval**, **operationID**, and **honest failure modes** — on both
ADR targets in CI.

## What item 1 does **not** guarantee

- Demonstrated managed `StartRun` through **real** Claude Code, Codex, or Cursor
  (all three Layer B slots are **owner deferrals** at `c70ac28`)
- Claude Code **attachment / registration** (item 10; backlog **B** for managed launch)
- Layer B **manifest** evidence for any named tool (deferred, not absent gate)
- Human-admin cross-agent `StartRun` (explicitly out of scope per H101-158)
- Item 8 disclosure text or item 9 guide supplement (tracked separately; not item 1
  blockers)

---

## Docs flags (Kevin — not item 1 blockers)

| Issue | Kelly ruling |
| --- | --- |
| `blueprint.md` line 79 cites Claude `toolSpec` invocation as current fact | **Stale** after `c70ac28`. **Do not card now** — wait for **item 9 Phase 3 guide supplement** (H101-196 coupling) or Ryan docs pass; not gating item 1 verdict |
| Item 8 / item 9 disclosure updates | Already tracked in H101-196; discharge with items 8/9, not retroactive item 1 block |

Kevin leaving Layer B criteria row for Kelly re-verdict: **correct instinct** — criteria
owner owns discharge rows.

---

Authored by Kelly (QA), H101-211.
