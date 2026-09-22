# Functional-feature architecture

2026-09-22 · Baseline `eba4a60` · Read with the [system blueprint](blueprint.md).

**Status applies only to the scope named below.** BUILT AND ACCEPTED means recorded acceptance of that scope; BUILT NOT YET ACCEPTED includes partial implementations with required behavior missing; DESIGNED NOT APPROVED FOR CONSTRUCTION means a proposal, not a build instruction; NOT DESIGNED means no concrete implementation contract is established. These labels do not convert Phase 3's already-authorized but unbuilt obligations into unapproved work. Each entry cites its status authority. No tests or provider calls were run for these documents.

## Agent registry

**BUILT AND ACCEPTED — registration and metadata updates.** Sources: [accepted Phase 2 baseline](../quality/h101-112-phase2-item2-discharge-ruling.md), [reference-validation ruling](h101-109-agent-reference-validation.md), and [current records](../../internal/core/domain/records.go).

Purpose: give tasks and messages stable local addresses. Commands: `register`, `update`; application queries `GetAgent` and `GetSnapshot`. Core validates records and references; the store retains them; presentation displays them. Registration neither starts an agent nor authenticates a sender.

**DESIGNED NOT APPROVED FOR CONSTRUCTION — retirement direction.** The [product retirement proposal](../product/definition.md#agent-retirement-stop-offering-new-work-preserve-the-record-h101-29) describes removing an agent from new work while preserving history. The [current Phase 3 exclusions](../quality/phase3-exit-criteria.md#explicitly-not-phase-3-exit) keep it outside this gate. There is no retirement command or record field; the detailed engineering contract remains incomplete. Older desired milestone wording is not a current construction instruction.

## Task lifecycle and human review

**BUILT AND ACCEPTED — create, progress, block, report, reject/rework, accept and reopen.** Sources: [Phase 2 exit](../quality/h101-112-phase2-item2-discharge-ruling.md), [boundaries](boundaries.md) and [documented workflow](../GETTING-STARTED.md).

Commands: `create`, `transition`, `report`, `accept`, `reject`, `task`. Core owns the Todo → Doing/Blocked → AwaitingReview → Done rules and revision/authority checks. Reports and human decisions are separate persisted records. A report does not complete human review; accepting a report does not prove its content correct.

**DESIGNED NOT APPROVED FOR CONSTRUCTION — reassignment.** [H101-38](h101-38-task-reassignment-shape.md) proposes prospective ownership transfer preserving prior attribution and pending review, with stale-owner/revision protection. No shipped reassignment command; do not simulate it by editing state. Initial assignment is already built and is a different feature.

## Messaging and mailbox

**BUILT AND ACCEPTED — durable handoff and explicit acknowledgement.** Sources: [Phase 2 mailbox acceptance](../quality/definition-of-done.md#current-phase-2-ruling-2026-09-21); the continuing-host pump's later acceptance is recorded with provenance in [blueprint E4](blueprint.md#evidence-and-status-authority).

Commands: `send`, `messages`, `message`, `ack`; file envelopes are a second ingress. Core records queued, published, processed and acknowledged as separate facts. The mailbox adapter publishes/scans/archives; its delivery loop alone supplies delivery facts through the trusted recorder. One-shot commands pump delivery; `serve` now retries it while clients are detached. Recipient acknowledgement proves neither comprehension nor work completion. Free-text sender claims never grant start or review authority.

## Profile approval

**BUILT NOT YET ACCEPTED — part of the unfinished managed-tool feature.** Sources: [Phase 3 item 1 and H101-158](../quality/phase3-exit-criteria.md), [latest reported standing E3](blueprint.md#evidence-and-status-authority), [approval implementation](../../internal/core/task/engine.go).

Command: `approve-profile`; `start-run` consumes the recorded profile. Core requires reviewer authority for approval and refuses duplicate profile IDs. Host supplies scope; the adapter resolves and executes the program. Local installation/version detection does not grant approval or prove authentication.

Current construction covers the initial start gate, not every proposed profile administration capability. Do not infer revocation, general profile revision editing, provider isolation or a complete execution-digest enforcement mechanism from this entry. [H101-154](h101-154-agentic-cli-descriptor-contract.md) defines the intended descriptor/approval split; its requirements are not all implementation facts.

## Managed runs and supervision

**BUILT NOT YET ACCEPTED — managed start exists; full supervision is incomplete.** Sources: [Phase 3 items 1–4 with amendments](../quality/phase3-exit-criteria.md), [E3 standing](blueprint.md#evidence-and-status-authority), and [process adapter](../../internal/adapters/process/supervisor.go).

Command: `start-run`. Core checks the registered caller's own agent, approved profile and one active Starting/Running run per agent. H101-158 explicitly excludes a new human process-admin capability from item 1; task-review authority is not that capability. Intent/operation/receipt and dispatch marker precede the external effect. The adapter launches the tool and observes a bounded startup window. A child outliving that window is recorded Running, not proven to have done useful work.

`StopRun`, `SetRunBudget`, `Observe` and `Recover` are contract names, not usable shipped CLI commands here. The supervisor's stop/observe/recover paths return Unsupported; real group termination, elapsed/reported-token enforcement and crash reconciliation remain designed Phase 3 obligations. The [supervision design](h101-128-phase3-supervision.md) is approved-phase work, distinct from the unapproved orchestration proposal. No current promise of stopping a run on host shutdown, detecting every later exit or recovering ownership follows from managed start.

## Run and operation observability

**BUILT NOT YET ACCEPTED — persisted start records and queries, not complete live observability.** Sources: [D5 and items 4–5](../quality/phase3-exit-criteria.md), [E3](blueprint.md#evidence-and-status-authority), [operation/run records](../../internal/core/domain/records.go) and [CLI queries](../../cmd/harnessing/run_commands.go).

Commands: `run`, `operation`; application `GetRun`, `GetOperation`, `GetSnapshot`, `Subscribe`. Core owns the facts; presentation renders them. The start receipt now identifies an operation created before spawn. That operation can succeed while the run is still Running: start completed, child lifetime did not. A persisted Running record can become stale because ongoing observation/recovery is unfinished.

The state-event stream is separate from output. A real output journal and shipped read/follow path remain absent; [item 5](../quality/phase3-exit-criteria.md) and [the journal design](h101-128-phase3-supervision.md#output-and-failure-isolation--item-5) are requirements, not acceptance. The existing package placeholder does not count as captured, durable, resumable stdout/stderr.

## Serve mode

**BUILT NOT YET ACCEPTED — hosting foundation and mailbox pump exist; full item 6 remains unaccepted.** Sources: [H101-143](../quality/h101-143-phase3-item6-foundation-ruling.md), [E4](blueprint.md#evidence-and-status-authority), [assembly implementation](../../internal/assembly/serve.go).

Commands: `serve`, with attached clients; `hold` is the earlier host-lifetime surface. Assembly owns the continuing host and workspace lock; the file transport carries caller-bound requests; core still decides mutations. Detaching a client does not shut down the host. Full acceptance also requires a real run remaining observable with output after detach and explicit termination on stop/shutdown, on both supported native targets. Working attachment and background mailbox delivery do not prove those requirements.

## Agentic command-line interface descriptor layer

**BUILT NOT YET ACCEPTED — version-2 schemas, descriptors and runner exist; real participation remains open.** Sources: [H101-153](../quality/h101-153-phase3-item1-layerb-refinements.md), [H101-154](h101-154-agentic-cli-descriptor-contract.md), [runner report](../quality/H101-152-REAL-LAYER-B-RUNNER.md), and [E3](blueprint.md#evidence-and-status-authority).

Surface: `scripts/run-agentic-cli-cycle.py` consumes a descriptor and invokes `harnessing start-run`; it is an evidence tool, not a second supervisor. [Tool schema](../../schemas/agentic-cli/tool-descriptor.schema.json) separates invocation facts from [scenario evidence](../../schemas/agentic-cli/evidence-scenario.schema.json). Per-tool files describe executable, typed argument/context slots and diagnostic hints; core retains approval and authority. The current path resolves tool arguments in the runner and supplies them to the product; do not claim the complete proposed shared product loader is already implemented.

Claude Code has an active descriptor but no accepted successful cycle. Codex and Cursor Agent have deferral entries, not compatibility proof. A version probe proves neither authentication nor participation. Product refusal is currently reported conservatively as `product_start_failed` with diagnostics, not invented spawn failure; structured product-error output remains separate work. The runner has no successful cycle collection path yet. [Open decision H101-163](blueprint.md#open-owner-decision-h101-163-participation-protocol-or-claude-code-deferral) compares a narrow worker protocol with explicit Claude Code deferral.

## Delegated orchestration and other future surfaces

**DESIGNED NOT APPROVED FOR CONSTRUCTION — orchestration skill, delegated session, scoped views, quotas and human-decision records.** Sources: [ADR 0006](../adr/0006-orchestrator-skill-and-delegated-session.md), [H101-148/H101-150](h101-148-agent-orchestrator-integration.md). No shipped command names should be inferred from proposed `DelegatedSession` or `GetDelegationView` interfaces. Future core/host code would enforce immutable actor/workspace binding, scoped reads, grants and revocation; a skill would only teach use. Cross-worker start authority would be a deliberate later widening. No date, build instruction or acceptance is implied by review or the owner's installed-tool priority.

**NOT DESIGNED — concrete graphical interface and a Model Context Protocol adapter.** Sources: [product definition §4](../product/definition.md#4-command-line-to-graphical-interface) and [ADR 0006 alternatives](../adr/0006-orchestrator-skill-and-delegated-session.md). A replaceable frontend contract exists; neither a graphical product design nor an approved protocol adapter implementation follows from it. Hosted/team coordination remains explicitly refused, not an implied backlog item.
