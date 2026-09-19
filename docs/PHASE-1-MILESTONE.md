# Phase 1 milestone — core domain, headless

Status: **exit criteria discharged, awaiting owner approval.** Prepared 2026-09-19 at
commit `6df933f`. The project plan requires each phase to end with a review of the work
**and the practices**, a written summary, and a stop for human approval before the next
phase starts. This is that summary.

## What Phase 1 promised

"The hive as a library: agent registry, the on-disk message protocol, the task ledger, and
the file conventions. Zero user interface, zero network, tested in isolation."

## Exit criteria and their evidence

The exit criterion at the start of Phase 1 was one sentence — "the core drives a
multi-agent scenario end to end with no UI present". It was **rewritten during the phase**
into six items that pass in CI on `main`, because a one-sentence criterion gets argued
about and a six-item one gets checked. All six are discharged; the verdicts are quality's,
not the orchestrator's, and each is recorded in full in
[`docs/quality/definition-of-done.md`](quality/definition-of-done.md).

| Item | Status | Discharged by |
| --- | --- | --- |
| 1 Product cycle, 2+ agents and 1 human, through the composition entry point, surviving reopen | Satisfied | `fc18dde` |
| 2 Assembly authorization negatives through the returned capabilities only | Satisfied | `8dc828f` |
| 3 Containment: no unapproved package imports the store; forbidden fixture must fail the check | Satisfied | `a34f51d`, `ea4425c` |
| 4 Composition surface exposes only capabilities plus shutdown | Satisfied | `8dc828f` |
| 5 Unit tests do not substitute for the assembly proof | Satisfied | unchanged core tests |
| 6 Zero network | Satisfied | `go list -deps`, no `net` import |

## What was built

Three core command sets, each QA-accepted separately: the **task ledger** with acceptance
that cannot be reached except through a human reviewer (`681d663`); **messages** whose four
delivery facts — queued, published, processed, acknowledged — are each written by exactly
one code path so none can be inferred from another (`3ff78eb`); and the **agent registry**
(`2841635`). Above them, a **headless composition boundary** (`8dc828f`) and a **CI import
gate** (`a34f51d`, `ea4425c`).

## The one decision that changed the phase

A host could bypass the core's rules by calling `StateStore.Commit` with its own mutation.
Quality found it twice — once on task acceptance, once on message acknowledgement — and it
was logged against Phase 2 host assembly. That was wrong, and the reason is worth
recording: **the bypass gained a new surface every time the core grew a command**, so its
cost rose with every slice while the fix waited. Raised with the architect, whose ruling
(`f68367f`) kept the port shape — a replacement snapshot authors the same false state, so
narrowing the payload buys nothing — and found **a second route nobody had seen**:
`CallerScope.IsHumanReviewer` was a host-supplied boolean, so privileged code could claim
review authority without touching `Commit` at all. Containment moved into Phase 1.

Had we fixed only the route we found, we would have closed one door, left the other open,
and believed the problem solved.

The engineer then solved it better than it was specified. **No exported method takes an
authority flag.** `host.Open` receives a fixed reviewer set from whoever assembles the
workspace, commands take only a caller agent ID, and authority is decided by set
membership — outside the call, with no path from any agent-authored payload. The claim
cannot be made, so it does not need checking.

## Limits, recorded rather than resolved

These are honest gaps, not oversights. Each is a tracked card.

- **`CommitRequest` construction inside an already-approved package is not statically
  detected.** The stronger check the architect asked for — resolved symbols and method
  values — needs an AST checker or a new dependency, and quality ruled it **not checkable
  today** in a zero-dependency repository. The weaker package-import rule was named as
  weaker, and CI proves it fires against a deliberately forbidden fixture.
- **The forbidden-fixture CI step accepts any non-zero exit**, including the exit code that
  means "your allowlist is broken". It is safe only because the containment step runs first
  and fails on the same misconfiguration — so that guarantee currently rests on step order
  rather than on the assertion. Fix queued.
- **The composition surface has no reassignment command.** `AssigneeID` is written only at
  `CreateTask`, so a task cannot be moved between agents. Found by the exit proof, which is
  what exit proofs are for. Quality ruled "hand off" discharges on a task-linked message
  plus acknowledgement; whether reassignment is wanted is a product question, open.
- **No agent can be retired.** `Agent` has no status field where `Task` has one, so the
  roster only grows. With the product owner now.
- **No stale-lock recovery.** A host killed without `Close` leaves the workspace locked.
  Deferred deliberately, and it **blocks the Phase 2 restart-after-crash milestone** until
  either recovery exists or manual removal is documented.
- **Trust is inspectable, not absolute.** The named trusted set is a real boundary against
  an agent claiming authority it was not given. It is **no claim of resistance to malicious
  same-process code, hostile file writers, or forged same-user identity.** That sentence
  travels with the claim wherever the claim goes.

## The practices, which the plan asks about separately

What worked, stated as practice rather than praise:

- **Commits are the orchestrator's, staged by name.** Seven times a document review sat in
  the working tree beside half-finished core code, twice from two different agents at once.
  Nothing was ever mixed, and no `git add -A` published unfinished code inside a review
  commit.
- **Reviewers re-ran the checks instead of trusting the relay**, unprompted, including
  checking the cited code against the document describing it. Two stale findings were
  caught this way — including one of mine.
- **A guard is not accepted until it has been watched failing.** The commit-identity guard
  was accepted in Phase 0 only after it rejected a wrong-author commit; the import gate was
  accepted only after the orchestrator broke the allowlist by hand and saw it exit non-zero
  with the missing package named.
- **The exit proof was deliberately not written by the author of the boundary it
  exercises.** It found a missing command within the hour.
- **"Zero divergences" is treated as an absence of a result, not a result.** A clean slice
  and an unrecognised divergence produce the same report; only a second reader tells them
  apart. One slice declared zero and had three.

What to change in Phase 2:

- A finding that recurs on a new surface as the system grows is a **design question**, and
  the phase it was first filed against is probably wrong. That cost a re-sequencing here.
- Adding a new exit code to a guard silently weakens every caller that tested only for
  "non-zero". Two guards now share that shape.
- Token velocity and being stuck look identical from outside. The breaker fired twice on
  legitimate work; the agent's own account was the tiebreaker both times.

## What is being asked

Approval to close Phase 1 and open **Phase 2 — the CLI adapter**, whose exit is a usable
`harnessing` command plus a second throwaway adapter proving the interface is genuinely
replaceable. Phase 2 carries three obligations already dated: the in-product first-run
disclosure, the mailbox causal-ordering rules, and stale-lock recovery before the
restart-after-crash milestone.
