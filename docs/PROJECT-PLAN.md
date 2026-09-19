# Harnessing 101 — project plan

Status: **APPROVED by the project owner on 2026-09-19** — the phases and the stack decision.
Phase 0 is executing. No product code exists yet.

Authority: for the stack, the authoritative record is
[ADR 0001](adr/0001-language-and-runtime.md), not this document. Where this plan and an
accepted ADR disagree, the ADR wins.

## What this is

A local-first application for running and coordinating several AI agents. Agents are
addressed and driven from a command line, and they organise themselves as a "hive"
through plain files on the user's own machine.

Reference and inspiration: Munder Difflin (https://munderdiffl.in/). Harnessing 101 is a
corporate-friendly, serious take on the same idea. **It is not a copy** — no code, no
assets, no wording taken from it. Where behaviour is genuinely similar, that is because
the problem is the same, and it will be reimplemented from first principles.

Repository: https://github.com/rafaelcalves/harnessing-101 (currently PRIVATE and EMPTY).

## Non-negotiable constraints

These come from the project owner and are not open for the team to relitigate:

1. **Runs completely locally.** The product must not reach anything outside the user's
   machine by itself. **The approved public wording is narrower than that sentence and is
   the one that ships** (owner-approved 2026-09-19): "Harnessing 101 runs completely
   locally and makes no external connections itself; agents you configure may send content
   they can access to external services, and Harnessing 101 does not confine those agents
   or guarantee that your data stays on this machine." Do not restore the shorter claim in
   the README or anywhere user-facing — it is not true, for the reasons in
   [the threat model](security/threat-model.md), sections 4 and 5. No telemetry, no phone-home, no hosted service, no implicit network
   calls. If a user configures an agent that itself calls a model provider, that is the
   user's own configured egress, not ours — and it must be visible and opt-in.
2. **Decoupled architecture.** The user interface is a replaceable adapter. Swapping the
   CLI for a GUI must not require touching the core. This is a structural requirement
   from day one, not a later refactor.
3. **Open-source ready.** Licence, contribution guide, code of conduct, issue and pull
   request templates, and a clean history from the first commit — prepared before we
   need contributors, not after.
4. **Commits use the owner's personal identity:** `rafael.ca.dev@gmail.com`. Repository-local
   git config, never the global work identity. This is verified in Phase 0 and enforced,
   not remembered.
5. **All work stays inside `~/Projects/Harnessing101`.** Documentation lives here too.

## Stack — DECIDED: Go

**Go for the core and the first command line interface.** Accepted in
[ADR 0001](adr/0001-language-and-runtime.md) on 2026-09-19, with the ports-and-adapters
core preserved and ten ports defined in [boundaries.md](architecture/boundaries.md).
A later graphical interface reaches the Go core through a versioned local standard
input/output bridge, with no network listener — which keeps the local-only constraint
intact rather than trading it for convenience.

**Historical note, kept deliberately.** I recommended TypeScript on Node and named its
weakness in the same breath: single-binary distribution and process supervision are easier
in a compiled language. The architect was asked to attack that recommendation rather than
ratify it, and he overruled it. No product code existed, so the change cost nothing — which
was the reason for sequencing the stack decision before Phase 1. The full argument, the
alternatives considered, and what would make us revisit it are in the ADR.

## Phases

Each phase ends with a milestone that reviews **the work and the practices**, produces a
written summary, and **stops for human approval** before the next phase starts.

### Phase 0 — Foundations and practices (no product code)
Repository hygiene, licence choice, contribution and conduct documents, architecture
decision record process, the stack decision with its argument recorded, the local-only
threat model baseline, continuous integration, and the commit-identity guard.
Exit, checkable rather than aspirational: a licence file is present; `CONTRIBUTING.md`
documents build, test and the architecture-decision-record process; continuous integration
runs build, test, vet and a linter on pull requests and is green; the commit-identity guard
has been observed rejecting a wrong-author commit; and every accepted decision is recorded
in an ADR. The earlier wording, "a repository a stranger could contribute to", was
unmeasurable — flagged in the Phase 0 audit and replaced.

### Cross-cutting: content provenance (added 2026-09-19, from the Phase 0 threat model)

The local-only rule permits one channel it cannot police: an agent using **its own
already-authorised** model-provider connection. The security review's top finding is that
adversarial content planted in a task or message file can travel to a second agent and
leave through that agent's legitimate egress — so **no network test on our own process
catches it**. Our code not calling out is necessary and not sufficient.

Consequence for the design, not deferred to Phase 4: workspace file content is **untrusted
input** when it reaches another agent, and the core must carry enough provenance to say
where a piece of content came from. This is a Phase 1 core concern because retrofitting
provenance onto a message format already in use is expensive. See
[the threat model](security/threat-model.md), section 4.

### Phase 1 — Core domain, headless
The hive as a library: agent registry, the on-disk message protocol, the task ledger,
and the file conventions. Zero user interface, zero network, tested in isolation.
Exit: the core drives a multi-agent scenario end to end with no UI present.

### Phase 2 — CLI adapter
The first user interface, built strictly through the core's ports. Includes the proof
that matters: a second, throwaway adapter driving the same core, demonstrating the UI is
genuinely replaceable rather than nominally so.
Exit: `harnessing` command usable for real work; swappability demonstrated, not claimed.

### Phase 3 — Agent process control
Starting, supervising and stopping local agent processes. Budgets, a circuit breaker,
crash recovery, and what happens when an agent misbehaves.
Exit: agents can be run, constrained and stopped safely from the CLI.

### Phase 4 — Hardening and first release
New-joiner documentation, packaging and distribution, and verification of the local-only
guarantee by observation rather than assertion. The security review has made that
observation concrete and automatable: run the controller alone under a network-denying
harness across the full lifecycle and assert zero outbound attempts from the controller
process; then run it with one agent profile against a decoy sink and confirm any egress
traces to the child process rather than the controller, and follows a recorded approval
event.

**What that test does NOT prove, stated here because I overclaimed it once already.** It
covers the controller's own egress, which is zero in every phase, and — only for agents the
product itself starts — that observed egress traces to an approved profile. It says nothing
about a manually started agent. In Phases 1 and 2 every agent is manually started, so for
those phases **nothing enforces the boundary**: the product has no process to gate and no
approval event, and it relies entirely on the user running commands they trust. See
[the threat model](security/threat-model.md), section 5.
Exit: a tagged release an outside user can install and run.

## How the team works on this

- Cards for this project are prefixed **`H101-`** to keep them distinct from the
  unrelated `HIVE-` work on the other floor.
- The architect pressure-tests decisions; the product owner owns scope and user
  experience; quality assurance owns acceptance criteria; the security expert reviews the
  local-only guarantee; engineers implement; the documentation owners keep the repository
  contributor-ready. Practices themselves are reviewed at every milestone, not just code.
- Evidence convention, carried over because it earned its place: claims are labelled
  CODE-PATH FACT, OBSERVED STATE (dated), SECONDARY SOURCE, INFERENCE, or UNKNOWN.
  UNKNOWN is an acceptable answer. A confident guess is not.

## Open questions for the owner

1. The repository is **private**. Open-sourcing means making it public — your call, and
   the timing is yours. Phase 0 prepares for it either way.
2. **Licence.** The team will bring a recommendation with reasoning in Phase 0. If you
   already have a preference, say so and we will not spend the cycles.
3. **Name collision and branding.** "Harnessing 101" and any logo or wording must not
   read as Munder Difflin's. Flagging it now so it is a decision, not an accident.
