# H101-153 — Phase 3 item 1 Layer B refinements (2026-09-21)

Kelly QA ruling on owner instructions (H101-153). Amends H101-147 Layer B edges only.
Layer A (participation fixture in CI) **unchanged** — not softened.

Owner independently asked for the same two-layer split: CI mock + real script in
demo/dev. One evidence model; no parallel scheme.

---

## Refinement 1 — Is the script required evidence?

**Yes.** Layer B requires **both**:

| Artefact | Role |
| --- | --- |
| **Runner script** (committed, documented) | Repeatable procedure any human runs in a provider-reachable environment; invokes managed `StartRun` through the product |
| **Manifest output** | **Written by the runner**, not hand-authored — timestamp, commit hash, target, descriptor used, commands, stdout/stderr tails, pass/fail |

A hand-written manifest does **not** discharge Layer B. It cannot be re-verified
after a code change; the script can.

**What the criterion does not mandate:** a particular language (shell, Python, etc.)
or a particular test framework. It mandates a **committed runner** whose documented
invocation produces the manifest.

**Layer A unchanged:** the in-repo participation fixture remains the only CI half.
The runner script is **not** a substitute for the fixture and does not run in
network-blocked CI.

**Environment (H101-141):** the runner cannot succeed in the hive until the owner
enables provider reachability. That blocks **producing** Layer B evidence here,
not the criterion. Layer A still required every merge.

---

## Refinement 2 — Extensible descriptor format

**Yes — extensibility is exit evidence, proved without a fourth tool.**

| Required | Checkable how |
| --- | --- |
| **Descriptor-per-tool** — each compatibility target (Claude Code, Codex, Cursor Agent) has one committed descriptor, **or** one committed **deferral record** in the same format | Review + directory layout |
| **Schema contract** — adding a fourth CLI is documented as "new descriptor file + runner invocation"; no new product code path required for tool identity alone | Stanley rules schema after Claudio bottom-up proposal; Kelly accepts schema as exit artefact |
| **No hardcoded trio** — start/profile resolution does not branch on tool name strings for the three owner-named CLIs; descriptors are loaded data | CI grep/review gate (implementation card defines exact check) |

**Not required:** implementing or certifying a fourth agentic CLI before Phase 3 exit.
Extensibility is **schema + loader behaviour**, not tool count.

**Hold (narrow):** Kelly does **not** hold the whole ruling for Claudio's proposal.
**Hold only the descriptor field list** until Stanley rules the format. Principle
above is fixed now; field names wait for `h101-153` follow-on when proposal lands.

---

## Decision table additions (Layer B)

| # | Observation | Defect class | Fix |
| --- | --- | --- | --- |
| D14 | Layer B manifest hand-written or edited without runner output | **Evidence** | Runner must produce manifest |
| D15 | No committed runner documented for Layer B | **Evidence** | Script required |
| D16 | Fourth tool would require new `switch`/hardcoded branch in start path | **Product** | Descriptor extensibility violated |
| D17 | Runner used as Layer A CI substitute | **Test** | Fixture only in CI |

---

## Three tools + deferrals

Unchanged from H101-147: three named tools are separate targets. Each descriptor
slot is **tool manifest** (runner output) **or** **owner deferral** (same
descriptor/deferral format, explicit `deferred: true` + reason). Missing slot =
item 1 not discharged for that tool.

Authored by Kelly (QA).
