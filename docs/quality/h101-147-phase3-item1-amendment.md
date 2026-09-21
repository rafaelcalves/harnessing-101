# H101-147 — Phase 3 exit item 1 amendment (2026-09-21)

Kelly QA amendment before item 1 construction. Authority: Angela product ruling
(H101-146), owner decision (H101-142 agentic-CLI profile model). Original item 1
text retained in [`phase3-exit-criteria.md`](phase3-exit-criteria.md); this
document is the dated amendment.

---

## Problem

Angela: item 1's generic approved-process→`Running` proof could **pass without a
usable agent tool** — built, green CI, exit satisfied, user still cannot
coordinate. Owner priority: profiles invoke **already-installed, user-authenticated
agentic CLIs** (Claude Code, Codex, Cursor Agent); product does not bundle or
authenticate providers.

---

## Amended required evidence

Replaces bare "`Running` after approved profile" as the item 1 bar.

1. **Selected installed tool** — `StartRun` launches the executable configured in
   an approved profile revision for the bound agent.
2. **Participation, not launch** — supplies task/workspace context the tool needs
   to participate in coordination (documented handoff), not merely exec a process.
3. **Honest failures** — missing binary, authentication-required, unsupported mode,
   and spawn failure surface stable actionable errors.
4. **Preserved gates** — profile approval, caller scope, H101-135 dispatch
   ordering, CF1/CF3 unchanged.
5. **Installed ≠ approved** — presence on `PATH` does not grant workspace approval.

---

## CI-checkable vs manifest

| Layer | What | Where |
| --- | --- | --- |
| **A — CI** | Participation **fixture** (in-repo test tool); full shipped path; context injection; R3 negatives; dispatch-ordering barriers; approval/caller negatives | `go test` on both ADR targets every merge |
| **B — manifest** | At least one owner-named agentic CLI per ADR target, managed through product, with auth/context proof — **or** owner deferral for that tool/target | Native manifest; not automated in network-blocked CI |

Claudio H101-141: real Claude Code blocked at provider network in hive CI
environment. Layer B cannot be CI-mandatory where provider reachability is absent;
it is **disclosure/manifest**, not a waiver of the participation requirement.

---

## What does not count

- Generic `sleep`/`cat` reaching `Running`
- Manual shell start of an agentic CLI (spike class)
- Assuming install implies approval
- Weakening dispatch-ordering to accommodate tool startup latency

---

## Estimate

Kevin's **3–5 agent-days** (old criterion, dispatch-ordering hard part) is
**stale**. H101-147 adds participation fixture, profile context, failure taxonomy,
manifest procedure. Plan **~5–8 agent-days** for item 1; tell owner the number
moved.

---

## UI-09 / item 7 coupling

Item 7 UI-09 must assert Layer A participation fixture path on both adapters, not
generic process start.

Authored by Kelly (QA).
