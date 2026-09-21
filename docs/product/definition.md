# Harnessing 101 — product definition

**Status — 2026-09-21:** Phases 1 and 2 have exited. The command-line coordination workflow is implemented and runnable; see [Getting started](../GETTING-STARTED.md) and the [Phase 2 exit ruling, including its recorded limits](../quality/h101-112-phase2-item2-discharge-ruling.md). This does not establish real agent-tool compatibility or validated user benefit.

Phase 3 is owner-approved and in progress, not accepted as complete; its [exit criteria](../quality/phase3-exit-criteria.md) distinguish the required capabilities from completed work. Other proposals in this document are not implementation or construction approval: unresolved scope changes still require owner review, and the graphical interface remains deferred.

Source: [project plan](../PROJECT-PLAN.md). This document combines product decisions, recommendations and open questions. These are not validated user findings; user benefit and real agent-tool compatibility remain unvalidated.

## 1. Core user and job

The initial user is an individual software engineer coordinating two or three local agent sessions on one bounded project: for example, one investigates a failure, another checks the proposed fix, and the engineer decides what to accept. They already know how to start their chosen agent tools. They need to know who owns each task, what evidence came back, and which decision requires them.

The core job is to assign work across agent sessions and recover an understandable record of ownership, handoffs, blockers, and results without copying context between terminals.

Several terminals already provide parallel execution. Harnessing 101 earns its place only if it reduces coordination effort: a handoff reaches a named recipient, remains available across sessions, and links to the task and resulting artifact. The user can leave and return without reconstructing progress from terminal scrollback. More agents, conversational personalities, or a count of completed messages are not themselves value.

**UNKNOWN:** whether this user has enough recurring coordination work to justify another tool. Settle through an owner-led trial of the same bounded workflow with terminals alone and with Harnessing 101. Record manual context transfers, time spent reconstructing status, and missed or duplicated work. If coordination does not improve, narrow or stop the product rather than adding a dashboard.

## 2. Minimum useful product

The minimum is a local, inspectable coordination workspace for two manually started agents and one human. It supports one complete cycle: assign, hand off, report a blocker, resolve it, and review a result.

The user can create a workspace in a chosen directory, register named agents, and see how to connect their existing sessions. They can create a task with one accountable owner, send a task-linked message, inspect pending messages, and see whether a message was recorded or acknowledged. Acknowledgement must not be presented as proof that work happened.

One status view answers: what is assigned, what is reported in progress, what is blocked, and what has a result ready for review. Every status identifies its reporter and last update; old reports must not imply a live process. A result includes a concise explanation and a local artifact reference. “Reported complete” remains distinct from human acceptance. After closing and reopening the interface, the user can recover these records without resending work.

Agents can propose assignments and exchange messages through documented local files. Human intervention stays possible: correct ownership, answer a blocker, or reject a result. The minimum does not depend on agents autonomously decomposing a broad objective.

Phase 1 proves this cycle without a user interface. Phase 2 makes the same cycle usable from the command line and demonstrates a replaceable interface. Its milestone is a real two-agent task completed with durable handoffs, including a restart of the interface and a human decision.

The plan postpones process control until Phase 3. Therefore Phase 2 cannot promise to launch, stop, supervise, or enforce spending limits on agents. It must say plainly that users start and stop their sessions themselves. **UNKNOWN:** which agent tool can follow the file protocol reliably enough for this trial. Settle with one named tool and a successful walkthrough before claiming compatibility. If manual setup erases the coordination benefit, revisit phase boundaries at owner review.

## 3. What it refuses to do

- Hosted coordination, shared web dashboards, team synchronization, or remote access.
- Telemetry, automatic update checks, implicit downloads, or product-initiated external calls.
- Bundled model access or silently configured provider connections. Any agent egress is explicitly user-configured and visible; local coordination does not mean local inference.
- A general chat client, integrated development environment, or replacement for existing agent tools.
- Guaranteed agent correctness, automatic acceptance of results, or autonomous publication and deployment.
- Automatic merging or safe concurrent editing of the same files. The initial workflow uses separate artifacts and human integration.
- Universal agent compatibility, workflow marketplaces, or enterprise administration in the minimum product.
- Background process supervision, restart guarantees, or hard budget enforcement before Phase 3.

## 4. Command line to graphical interface

The command line is useful for precise assignment, scripted inspection, and reading one task. A graphical interface would help when users need to scan several agents, follow dependencies, compare proposed results, or locate the sequence of messages behind a blocker. Those are information-navigation needs, not reasons to build an animated office.

Both interfaces must expose the same task ownership, message acknowledgement, evidence, timestamps, and acceptance decisions. Switching interfaces must preserve identifiers and history. A graphical view must not invent status from activity animations; the command line must not hide essential decisions in transient prompts. A local graphical interface is a possible future adapter, not a Phase 2 deliverable or permission to introduce hosting.

## 5. Tone and positioning

Use factual output: “Task T-12 assigned to reviewer.” “Result recorded; awaiting acceptance.” “Agent status unknown; last report 14:32.” Errors name the failed action, retained state, and next step. Avoid celebration, theatrical authority, and claims of safety unsupported by enforcement.

Documentation starts with a runnable local workflow, explains what data is written where, and states responsibility for configured agent connections. Use “workspace,” “agent,” “task,” and “message”; keep “hive” as a protocol metaphor, not an office role-play.

Avoid sitcom names, character avatars, office-floor visuals, and borrowed slogans. **UNKNOWN:** whether “Harnessing 101” is available and whether it sounds like a course rather than a working tool. Owner-approved naming and originality review should settle that before release. No reference implementation, assets, or copy were inspected for this definition.

## Content provenance and the honest local-only promise (H101-8)

### The promise

**INFERENCE — proposed README promise, contingent on implementation and verification:** “Harnessing 101 runs completely locally and makes no external connections itself; agents you configure may send content they can access to external services, and Harnessing 101 does not confine those agents or guarantee that your data stays on this machine.”

**SECONDARY SOURCE — design basis:** [Threat model, sections 2–4](../security/threat-model.md) separates controller traffic from agent traffic and identifies malicious workspace content as a way to influence an agent through its already-approved connection. This is a design review, not an observed incident. **CODE-PATH FACT:** none established for this proposed experience; there is no verified enforcement to advertise here.

**INFERENCE — meaning for users:** local records and a controller that never connects externally are the promise. Protection against misleading instructions in messages, unrestricted agent file access, or disclosure through an approved provider is not. Provider approval is not approval of every later message, and origin information is not proof that content is safe. Do not describe the workspace as a confidentiality boundary or show a “safe” badge beside an approved profile.

### What users see, and when

**INFERENCE — normal reading:** every task or message view should show a compact origin line: claimed author, how it entered the workspace, recorded time, task reference, and verification status. For example: “Claims sender: reviewer · imported file · identity unverified · task T-12.” Separately show any controller-recorded submission identity, with the limits of that mechanism. A sender field in a writable file is not authenticated authorship. A quoted instruction claiming to be from the user remains content from that message, not a user decision.

**INFERENCE — history:** the detail view should retain the original message reference, known forwarding links, and the exact content version to which acknowledgement or acceptance applied. Summaries should identify their author and source messages; missing links should say “origin incomplete.” Preserve what the controller actually recorded, without pretending to reconstruct an agent's reasoning or unknown sources. Keep credentials out of these records. Origin context should accompany the content delivered to another agent as well as the human display, explicitly identifying it as untrusted input; that label cannot ensure an agent obeys it.

**INFERENCE — interruptions:** do not prompt for every inter-agent message. Interrupt before the product enables a new execution profile or a changed provider, executable, credential reference, workspace access declaration, or tool capability. Present the change and require renewed approval. Hold product-mediated use of a profile whose approval is missing or no longer matches. If provenance required for a handoff is missing or inconsistent, hold that handoff for inspection, with options to reject or explicitly continue as unverified; approval must not relabel its origin as verified. Ordinary origin metadata and acknowledged-message history belong in inspectable records, not repeated warnings.

**UNKNOWN — detection:** what identity and tamper checks can reliably support those interruptions is unsettled. Architecture and security review must define the verification mechanism and its limits before the interface offers a verified-origin label. Do not promise detection of malicious prose or interception of network calls made independently by an agent.

### Approving an agent's external access

**INFERENCE — approval summary:** show the local program and fixed invocation, intended task, declared readable locations and tools, configured provider and destination, credential reference names without values, and whether any restriction is actually enforced. Explain: “This agent may send prompts, messages, and files it can read to [configured provider]. These destinations are declared configuration, not an enforced network allowlist.” Unknown destinations or access scope must remain visibly unknown; do not substitute reassuring defaults.

**INFERENCE — decision:** offer approve this specific profile revision, edit, or cancel. Record the approving user action and revision locally. An agent-authored message cannot grant or expand that approval. Users should be able to withdraw approval for future product-managed starts; withdrawal does not recall transmitted data or stop an independently running process. Actual provider retention and onward handling are **UNKNOWN** until the user reviews the chosen provider's applicable terms and configuration; profile approval must not invent that assurance.

### Cost to the minimum useful product

**INFERENCE — do not defer provenance past Phase 1.** Persist origin, verification limitations, source references, and approval/acceptance bindings in the first usable record format. Exercise them in the headless handoff scenario, including unverified file input. Phase 2 adds compact displays and decisions. Rich history visualization can wait; the facts it would display cannot.

**INFERENCE — real cost:** this adds setup and review friction to the two-agent trial. Use one reusable approval per unchanged profile, not per message. Manually started agents remain outside the product's start gate: registering a profile documents consent but cannot prevent the user launching another command or changing that process. Enforcing approved profiles at product-managed start belongs to Phase 3. The Phase 2 walkthrough must disclose this gap rather than imply existing sessions were validated. Measure this extra effort in the terminals-versus-coordination trial; provenance is necessary context, not evidence that the product is useful or that the injection risk is solved.

## Agent retirement: stop offering new work, preserve the record (H101-29)

**INFERENCE — product decision:** the minimum useful product needs a human-controlled way to stop offering new work to an agent while retaining its identity and history. Include this by the Phase 2 usable-work milestone; it need not block Phase 1's current containment work. Decide the behavior now so architecture can accommodate it deliberately. An append-only historical roster is acceptable; a roster where every historical participant remains a valid destination indefinitely is not.

**CODE-PATH FACT — narrow baseline:** [records.go](../../internal/core/domain/records.go) currently gives an agent an identity, display name, profile reference, and provenance, but no retirement information. Tasks and messages retain agent references. **SECONDARY SOURCE:** [boundaries.md](../architecture/boundaries.md) describes registration and updates without a retirement operation. This is a review of the record definition and documented contract, not a test of the engine.

### Why it belongs in the minimum

**INFERENCE:** a two-agent workflow can outlive either manually started session. Once the user decides a participant will not receive further work, keeping it available for assignment creates avoidable dead-end handoffs. That undermines the product's central promise of understandable ownership and durable coordination, even with a small roster. Merely hiding a name would leave direct addressing and agent-generated assignments able to repeat the mistake.

**INFERENCE — language:** describe the action as “Retire from new work.” It means a human has removed this participant from future assignments and new message delivery, not that a process has exited or that existing work is complete. A closed terminal, silence, or old activity timestamp must not automatically retire an agent. Use process observations separately when those exist; do not label the remaining roster “running.”

### What happens to the work

**INFERENCE — history:** completed tasks, accepted results, sent messages, and recorded acknowledgements keep their original attribution. Historical views identify the participant as retired without suggesting it was retired when the earlier event happened. Do not erase or transfer authorship to a replacement, reuse the identity for a different participant, or rewrite past acceptance.

**INFERENCE — unfinished tasks:** show affected tasks before retirement and keep them prominent afterward as needing human attention. Preserve their last recorded ownership and progress; identify that the assignee is retired and a handoff may be needed. Do not silently unassign, reassign, complete, reject, or restart anything. The human can explicitly transfer remaining responsibility while preserving prior results and attribution. An already-submitted result remains reviewable without its author being available; rejection or reopening exposes the need for a new owner rather than implying the retired participant will resume.

**INFERENCE — messages:** exclude retired participants from normal assignment and recipient choices, and refuse new work assignments and new messages addressed to them with a clear explanation. Retain pending messages and their actual delivery facts. Show which were unacknowledged at retirement; do not mark them read, cancel them silently, or forward them to a replacement. The user decides whether to send a new, linked handoff elsewhere. Already-published files cannot be recalled from an independently running agent.

**INFERENCE — late activity:** preserve valid acknowledgements of existing messages and valid results for work still assigned to that participant, subject to the usual authority and review checks. They do not reactivate the participant or imply new work was authorized. Once ownership has transferred, a late submission must not replace the current owner's result. Surface it as late material requiring inspection rather than silently adopting it. Retirement is a routing decision, not a security boundary against a process that retains file access.

### Cost, limits, and acceptance

**INFERENCE:** provide one concise retirement outcome showing unfinished tasks and unacknowledged messages, plus persistent labels on those records. Routine history needs no repeated warning. A useful acceptance scenario retires an agent with one accepted task, one unfinished task, and one unacknowledged message: history remains attributable, new routing is refused, and unresolved work stays visible across reopening the interface.

**INFERENCE — non-goals:** this decision does not add process stopping, inactivity detection, credential revocation, automatic replacement, automatic task recovery, deletion of history, or temporary pause/resume scheduling. Restoring a retired participant to new work can wait; a replacement starts with its own identity. It specifies no command names, data model, or lifecycle enumeration.

**UNKNOWN:** how often users replace participants within one workspace, and whether “retire” communicates the intended distinction. Settle in the two-agent walkthrough by asking the user to replace one participant and recover its unfinished work. Confusion may change the wording; it should not make silent reassignment or misleading process status acceptable.

## Workspace revision on command receipts (H101-93)

**INFERENCE — product ruling:** workspace revision means the version of committed workspace state, not the number of commands the user issued. Background recording of message delivery changes that state too. It is not a count of every real-world event: process activity, attempted commands, and work outside the recorded state are not measured by this number. Users should not use revision differences as a productivity or completed-work counter.

**SECONDARY SOURCE:** [adapter-contract ambiguity A3](../../internal/adaptercontract/AMBIGUITIES.md#a3--ui-04-workspace-revision-vs-mailbox-background-commits) reports a cycle of 12 user commands ending at revision 14 because two mailbox facts were recorded. That is consistent with this product meaning; the cycle was not rerun for this ruling.

**INFERENCE — receipt decision:** keep the number and the existing label “workspace revision.” It provides a useful state reference alongside the request identifier without claiming task completion. On a command receipt it identifies the version committed by that request, not a guarantee that this remains the latest workspace version when the user reads it. Background work may already have advanced the workspace. A replayed receipt still describes the original request; it must not imply a new user action occurred.

**INFERENCE — user-facing explanation:** “Workspace revision identifies a recorded version of your workspace; it advances when commands or background delivery record changes, so it is not a count of your commands.”

**CODE-PATH FACT:** [the receipt formatter](../../cmd/harnessing/flags.go) already prints the request identifier and `workspace revision` using the receipt's committed revision. **INFERENCE — documentation consequence:** no change to the CLI receipt line is requested. Put the explanation beside the first receipt example in the README and explain background advances and replayed receipts in troubleshooting guidance. Route those edits to their current documentation owner. This ruling does not decide cross-adapter assertions, exact revision totals, or implementation changes.
