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
