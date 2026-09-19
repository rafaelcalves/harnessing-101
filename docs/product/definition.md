# Harnessing 101 — product definition

Status: proposed Phase 0 scope; owner review required. Nothing described here is a shipped capability. Source: [project plan](../PROJECT-PLAN.md). Product choices below are recommendations, not validated user findings.

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
