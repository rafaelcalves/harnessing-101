# Threat model — the local-only guarantee

Status: **Phase 0, no code exists.** Evidence convention per the project plan: CODE-PATH
FACT, OBSERVED STATE (dated), SECONDARY SOURCE, INFERENCE, UNKNOWN. Almost everything
below is INFERENCE reasoning about a design in [boundaries.md](../architecture/boundaries.md)
and [ADR 0001](../adr/0001-language-and-runtime.md); it is labeled as such rather than
dressed as fact.

## 1. Assets, trust boundaries, in/out of scope

**INFERENCE — assets.** The workspace on disk (task ledger, agent registry, message
envelopes, run output journals) is the primary asset: the durable, human-inspectable
record of who did what. Second, provider credentials and any secret an agent process
needs — these belong to the user, not the core, but `ProcessSupervisor` is the mechanism
by which they reach a child's environment. Third, the workspace lock (one exclusive
writer) — losing it to a second process is a correctness and confidentiality problem, not
just a bug. Fourth, the user's filesystem outside the workspace root, which the product
must never touch as a side effect.

**INFERENCE — trust boundaries.** (a) Between the controller/core and each supervised
agent process — the agent is untrusted output-producer and untrusted input-consumer at
once. (b) Between the core and the workspace directory on disk — file content, including
message envelopes, is attacker-controllable if anything else on the machine can write
there. (c) Between the core and any other local process/user on the same machine that can
read or write the workspace path. (d) Between the core and whatever an agent process
itself decides to dial out to, which the product does not and should not control beyond
visibility.

**Must defend against (INFERENCE, this is the product's job):**
- A malicious or compromised agent process making the *core* originate network traffic,
  escape the workspace root, or corrupt another agent's task/message state.
- A hostile or tampered workspace directory: symlinks pointing outside the workspace,
  crafted message envelopes, path traversal in any user- or agent-supplied ID or path field.
- File/message content treated as instruction rather than data inside the core (injection
  into *core logic* — a task title or message body that changes what command executes,
  not just what is displayed).

**Explicitly out of scope (INFERENCE — an honest boundary, not a promise we can't keep):**
- Confining what a *user-configured* agent profile does once started. If the user points
  an agent at a model provider or gives it shell access, that egress is the user's;
  sandboxing arbitrary agent binaries (seccomp, containers, network namespaces) is a
  separate, unbuilt security design, not something ADR 0001 or boundaries.md commits to.
- Multi-user / multi-tenant isolation on a shared machine (OS permissions are the existing
  boundary; the product adds none of its own).
- A machine already compromised at the OS level (a root-level attacker can read anything
  regardless of what this product does).
- Supply-chain integrity of the Go toolchain or dependencies — real risk, but a
  build/release-process concern, not this threat model's.

## 2. Making the local-only guarantee testable

**INFERENCE — the line that must not blur.** "Our code calling out" is a defect; "an
agent the user configured calling out" is the user's own egress. The precise test: did
the *controller/core process* itself open a socket, or did a *child process the user
started with a user-supplied execution profile* open one? The core opening any socket
(other than a future explicit, documented local pipe/bridge for a GUI adapter) is always
a violation, unconditionally. A child agent process opening a socket is never a violation
of this guarantee by itself — but it must be observable and it must have required the
user's opt-in profile configuration to happen, never a default.

**INFERENCE — the concrete, automatable observation (proposed for Phase 4):**
1. Run the controller binary alone (no agent started) under a network-denying harness
   (Linux network namespace with no route, or a syscall filter on `connect`/`socket`, or a
   deny-and-log firewall rule) across the full lifecycle: install, first run, workspace
   create, task/message commands, shutdown. Assert **zero** outbound connection attempts.
   This alone proves the core's own egress claim, independent of agents.
2. Separately, run the controller with one agent profile started, pointed at a decoy
   DNS/HTTP sink the harness controls. Assert observed egress traces to the *child
   process*, not the controller, and only occurs after a recorded profile-approval event
   preceding any connection.
3. Repeat step 1 with no agents ever registered and no credentials present, and confirm
   the full local cycle still succeeds — proving locality doesn't silently depend on
   outbound reachability.
4. Make this a CI gate on every release build, not a one-time Phase 4 check, so a future
   dependency can't reintroduce a phone-home path unnoticed.

**UNKNOWN.** Which sandboxing primitive (network namespace vs. syscall filter vs. firewall
log) is available and reliable on every platform in the release matrix has not been
decided; that matrix itself is still open per ADR 0001.

## 3. Five non-negotiable, checkable rules

1. **No network client import in the core module.** Core package and transitive
   dependencies may not import `net`, `net/http`, or any third-party networking package.
   Checkable by `go list -deps` in CI — a diff of the dependency list, no judgment call.
2. **No telemetry, update-check, or crash-report call anywhere in the codebase.** Any
   outbound call outside the `ProcessSupervisor` adapter's documented child-spawn path is
   banned. Reviewer checks one yes/no: does this PR add a way to leave the process other
   than starting a user-configured child?
3. **All filesystem writes stay under the workspace root the host was given.** Every write
   goes through `StateStore`/`Mailbox`/`OutputJournal`, each resolving and rejecting `..`,
   absolute escapes, and symlink escapes before any write. Checkable: a test harness
   asserting no write-mode file descriptor is ever opened outside the resolved root.
4. **Secrets never appear in an event, snapshot, log, or command payload.** Any field
   reaching `StateStore`/`StateEvents` is checked in CI against a deny-list of
   secret-shaped field/env-var names; a new schema field matching it fails the build.
5. **Every `ProcessSupervisor.Start` requires a pre-registered, user-approved execution
   profile — no ad hoc argv, no shell interpolation, no implicit inherited environment.**
   `Start` rejects any spec whose profile ID lacks a prior recorded approval event; the
   adapter never invokes `/bin/sh -c`.

## 4. What worries me that the plan doesn't mention

**INFERENCE — my top concern.** The plan treats "agent egress is the user's own" as a
clean line, but message envelopes and task/message *content* are exactly the kind of data
that a compromised or careless agent can use to attack the *next* agent or the human, not
the network. Nothing in boundaries.md addresses content-level injection: a task title or
message body crafted so that when a second agent reads it (as instructions, since these
tools are LLM-driven) it manipulates that agent into writing malicious files into the workspace, running
destructive commands, or exfiltrating secrets through its *own* legitimate,
user-approved provider connection — which would not trip any network test above, because
that channel is explicitly permitted. This is the most likely way this product gets
someone hurt: not the core phoning home, but one agent using its already-authorized
model-provider connection as a covert exfiltration path for data another agent or the
human placed in the shared workspace, triggered by adversarial content sitting in a task
or message file. The plan has no mention of content provenance, no note that agents
should treat workspace file content as untrusted input, and no requirement that
credential material stay out of anything an agent can read from the workspace. I'd add:
agents must never get direct read access to credential material (the core holds none;
the ProcessSupervisor adapter should pass credentials via environment only to the process
that needs them, never write them into a workspace-visible file), and the docs should say
plainly that a malicious task/message body can manipulate a downstream agent — a risk the
network guarantee does not eliminate.

## 5. Manual-start exposure in Phases 1-2 (H101-11)

**Context.** Rule 5 ("`ProcessSupervisor.Start` requires a pre-registered, user-approved
profile") only binds agents the *product* starts. Process control is Phase 3. The
Phase 1-2 minimum useful product is explicitly two manually started agents plus one
human, per [definition.md](../product/definition.md). Rule 5 does not bind that
population at all, in either phase.

**1. What actually protects a workspace in Phases 1-2, if anything.** **INFERENCE, and
stated bluntly because a soft answer here is worse than a blunt one: nothing enforces it.
The controller has no process, no start gate, and no approval event to withhold — it
relies entirely on the user to only run agent commands they trust, by hand, outside the
product's view.** The product's only contribution in those phases is *record-keeping*: a
message envelope's writable sender field, a task's declared owner, and whatever the human
chose to type — none of it authenticated, none of it enforced. This is exactly the gap
Angela named: writable sender fields do not authenticate authorship, configured
destinations are not an enforced allowlist, and approval does not sanitize content. In
Phases 1-2 there isn't even a profile-approval event to point to; the exfiltration path
from section 4 (an agent's own connection, triggered by adversarial workspace content) is
live from day one, with zero product-side mitigation beyond what the human notices.

**2. Whether this changes the Phase 4 test.** No — but its scope claim must be stated
precisely. **INFERENCE.** The section 2 test proves: the controller itself never
originates egress (unconditionally, all phases), and — only for product-started
children — that observed egress traces to a specific child process following a recorded
approval event. A manually started agent in Phase 1-2 is not the controller's child, has
no approval event, and is invisible to `ProcessSupervisor` entirely; the test cannot
observe it, attribute its traffic, or prove anything about its behavior. Do not describe
the Phase 4 test as validating "agent egress is controlled" in Phases 1-2 — it validates
only that the controller's own egress is zero. That is a true and useful claim; it is not
the same claim as "agent egress is visible," which requires Phase 3 process control to be
true at all.

**3. Whether this is a reason to reconsider phase order.** **This is my finding, offered
to the owner, not a request to relitigate sequencing.** There is a security argument, and
it is narrow: in Phases 1-2 the product cannot observe, attribute, or gate *any* agent
process, manually started or not — the entire exfiltration path in section 4 is live with
no product-side control the whole time process control sits in Phase 3. That is a real
window, not a hypothetical one, and it is open for two full phases. Whether that risk is
acceptable given the phase's stated scope (a two-agent, human-supervised trial, small
blast radius, no claim of enforcement made to the user) is a product-risk-tolerance
call, not a security-only one — I am not asserting the window makes Phases 1-2 unsafe to
ship, only that it exists and is total, not partial. If the owner reads the honest
mitigation below and decides that is not enough for what gets shipped or advertised in
Phase 2, that is the trigger to reconsider order, not this section.

**4. Cheapest honest mitigation for Phases 1-2.** **INFERENCE.** Record what was observed,
never claim what was controlled. Concretely and cheaply, before Phase 3 exists:
- Every message/task envelope already carries the origin metadata Angela specified
  (claimed sender, entry mechanism, timestamp, verification status = unverified). Ship
  that from Phase 1, not later — it is the only honest signal available.
- Add one line, unconditionally visible, not a one-time onboarding screen: "Harnessing
  101 does not start, observe, or restrict any agent process in this phase. Nothing here
  confirms which program produced this content." Do not let a compact origin display
  read as a safety indicator.
- Do not implement any UI element that could be mistaken for validation (a checkmark, a
  "verified" badge, a green status) attached to manually reported content in Phases 1-2 —
  definition.md already asks for this ("do not show a 'safe' badge"); this section adds
  that the same restraint applies to any manually-sourced record, not only
  provider-approved ones.
- This is disclosure, not defense. It costs a docs line and a UI restraint, not an
  engineering project, and it is the ceiling of what Phases 1-2 can honestly claim.

## 6. Public-repository exposure review (H101-15)

**Context.** `rafaelcalves/harnessing-101` went public before Phase 0 planned it to. I
independently re-checked the claims handed to me rather than trusting them on relay:
`git log --format='%ae' | sort -u` returns exactly one address,
`rafael.ca.dev@gmail.com`; `git grep` across the current tree for `wolt.com` or the
owner's work address returns nothing; `git log --all -p` searched for the same terms plus
secret-shaped words (`secret`, `token`, `password`, key-header strings) across full
history, not just the current tree, and every hit is prose in the threat model,
SECURITY.md, ADR 0002, issue/PR templates, or the commit-identity guard describing how to
handle secrets — never a value. I also read the CI workflow, pre-commit hook, and
`.gitignore`: pinned actions by hash, no embedded token, no path or credential specific to
this machine.

**Verdict: confirmed, nothing found here.** No file or history entry should be removed on
confidentiality grounds. This is an OBSERVED STATE for this repository at this commit
(630c917), not a standing guarantee — re-run the same check before any future push if the
history changes.

**2. Disclosure risk of publishing the threat model and ADR 0002 at this detail.**
**INFERENCE — verdict: publish as-is, do not soften.** The exfiltration path described
(an agent using its own already-approved connection, triggered by adversarial workspace
content) is a structural property of "an LLM agent with tool access reads shared files,"
not a secret about this codebase's implementation. It is close to the general prompt-injection
risk already public in the wider industry's LLM-agent security discussion; naming it
precisely here gives a reader no attack they could not already reason out from the product
description alone, and there is no working exploit, no credential, and no code in either
document. What would be a mistake is the opposite move: softening or removing the
detail would look like the team noticed the exposure only after going public and reacted
by hiding it — worse for trust than the honest version, and it would not reduce the actual
risk, which lives in the architecture, not in these two documents. I would change one
thing, not for confidentiality but for correctness: ADR 0002 and threat model §5 currently
reason about "the bounded trial" and "users" in a way that assumed a small, likely
private-in-practice audience; that assumption is now false and needs correcting, which is
finding 3 below.

**3. Does going public earlier than planned change Phase 1-2 advice.** **Yes, one change,
INFERENCE.** The disclosure mitigation in section 5 (unconditional in-product line, origin
metadata, no validation-looking UI) was written assuming a reader who had likely seen the
surrounding docs — an owner-run trial. A public repository means the actual first contact
for many readers is a random clone-and-run, with no guarantee they read PROJECT-PLAN.md,
ADR 0002, or this file first. The in-product disclosure line therefore carries more weight
than it did when this section was written, not less: it is now plausibly the *only*
disclosure some users see. Recommendation, still cheap: keep it in-product exactly as
specified, and additionally surface it in the top-level README / first-run output, not
only inside the coordination views a user might not open before running two agents. I do
not see a reason to change the phase order or the ADR 0002 acceptance itself on account of
going public early — the underlying exfiltration risk and its scope are unchanged by who
can read about it; only how many people the in-product disclosure now needs to reach has
changed. I deliberately left the phase-order question, ADR 0002's acceptance, and the rest
of section 5 untouched — going public does not reopen either.
