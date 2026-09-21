# H101-128 — Phase 3 supervision architecture

2026-09-21. **Proposed implementation contract; design only.** Phase 3 is owner-approved according to the dispatch. Read Kelly's [nine exit criteria](../quality/phase3-exit-criteria.md) first; they remain the acceptance authority. This document does not declare any item satisfied. [ADR 0005](../adr/0005-single-owner-supervision.md) records the hosting/effect decision; the ADR 0003 appendix maps UI-09–13.

## Decisions needing resolution before implementation

**Elapsed deadline wording is not implementable literally.** Item 3 asks for observed real-process termination “before or at” the configured limit. A deadline notification, signal and operating-system exit observation cannot have guaranteed zero latency. Proposed interpretation, sent to god for Kelly's ruling: the core commits/enqueues termination at the deadline under a controlled clock, then native tests independently prove actual exit within a configured escalation/observation bound. Never freeze a test clock during a long wait and call that a real-time deadline guarantee. The criterion needs an explicit ruling; this document does not amend it.

**Owned tree means a supported process group, not arbitrary escaped descendants.** Proposed profiles must keep children in the group established by the supervisor and must not daemonize or detach descendants. Native tests must include a parent and worker, including a parent that exits before its worker. A group alone is not confinement against a deliberately escaping program. If item 2 means containment of all arbitrary descendants on both targets, this proposal cannot discharge it; god/Kelly must confirm the bounded support claim before implementation. Unsupported profiles fail before spawn, not after promising tree termination.

**ADR 0002 still requires its Phase 3 entry review.** Phase approval does not by itself document renewal of that ADR's risk acceptance. Security/product/owner must record the entry disposition before expanding its accepted workflow; no architectural choice below authenticates old content or confines provider egress.

## One continuing host, multiple detachable clients — items 6 and 9

Introduce explicit `harnessing serve` (foreground host suitable for a dedicated terminal). It alone holds the workspace lock, owns the engine/effect loop and all supervised processes, runs the existing mailbox Deliverer, and writes canonical state/journals. CLI sessions and the independent scripted adapter attach to that owner; a CLI's exit releases its session only. No implicit daemon per command and no UI-owned run context.

Use a local **file request/response transport**, not a socket, to preserve the inherited zero-network dependency constraint. A trusted transport adapter implements the restricted api.FrontendSession proxy; presentation never reads state.json, constructs CallerScope or receives a store. Clients publish bounded complete request envelopes into per-session intake using temporary-file/rename; only the host consumes them and performs domain mutation. Responses, polls for events/output, cancellation and host-generation identifiers are framed/versioned. Intake is not a second domain ledger or workspace writer. Domain receipts remain authoritative; abandoned transport files are bounded and cleaned without erasing domain request history.

The host bootstraps a session under its configured principal policy and binds a host-side principal/capability to an opaque session handle. Ordinary command envelopes carry that handle, not reviewer flags or caller overrides. Missing/invalid sessions are denied; read-only observer sessions cannot mutate. Restrict directory/file access, reject unsafe paths/symlinks and oversize input, and scope handles to a host generation. This is same-user local application authority, not protection against malicious code with the same filesystem permissions. Existing CLI caller selection is an input to trusted host policy, never proof of identity. Message envelopes cannot create sessions or request process control.

If a live host exists, all workspace commands connect; do not fall back to opening another store when transport is unavailable—return Busy/unavailable. Before server startup, a one-shot host may still support coordination commands, advertising supervision unavailable; StartRun requires continuing-host mode. Explicit shutdown is a host lifecycle operation available only through host-admin authority, separate from FrontendSession.Close/Detach. It stops owned runs, waits within the declared bounds and reports failures before releasing ownership; a failed shutdown is not a clean terminal state.

## Core policy, ports and records

Keep the ten architectural roles. Extend existing interfaces; replace placeholder `any` observations with passive typed records. Core owns authorization, run/operation transitions, budget decisions, recovery decisions and commit/effect sequencing. ProcessSupervisor owns OS process/group identities and normalized observations; OutputJournal owns byte persistence; Clock supplies monotonic deadlines and an injectable timer/wakeup operation, beyond today's MonotonicNow accessor. File transport, executable resolution, descriptors and platform signals stay in adapters.

Persist Runs, Operations, immutable ProfileApprovals and pending effect/attempt records in the existing atomic state generation. GetSnapshot enumerates runs/operations and nonsecret approval references; add GetRun, GetOperation, GetCapabilities, ReadOutput and FollowOutput to the neutral api/session. Deep detachment, observation cursors, request identity and structured errors apply unchanged. A Run includes agent binding, approved profile revision, initiating scope/provenance, state, start/stop operation references, budget settings/revision, latest usage evidence, stop cause and observed exit/capture outcomes. Never expose raw OS handles as authority.

Trusted observation ingestion is a restricted host-to-core capability, inaccessible to presentation or mailbox input. Observations carry run/attempt identity plus sequence; deduplicate and reject stale generations. Commit callbacks must perform no process or output side effect: repeated mutation evaluation or receipt replay cannot spawn or signal.

## Approval and caller authority — items 1 and 8

Human-admin approval is an explicit host-policy operation, recorded as a durable event before any start. It covers an immutable executable profile revision: resolved executable identity, argument template, working-directory policy, environment *references*, declared provider/egress intent, I/O mode, termination support and token-reporting protocol. Changed executable/configuration invalidates approval for new starts; a name alone is not approval. Recheck the resolved profile before dispatch; failure leaves an observable failed start without running an unapproved replacement. Local same-user executable replacement remains outside hostile-filesystem guarantees.

For the initial policy, a principal may start its own registered agent only with a host-granted managed-run capability for that profile; human admin may manage registered agents. Stop/budget require the run's bound owner with corresponding host permission, or human admin. Membership supplies reference integrity, not authority. StartRun agent/profile binding is frozen for that run; later UpdateAgent never retargets it. SenderAgentID is never an input to these checks. Budget increases/removal require human-admin authority; an owner with budget permission may tighten. Core-triggered enforcement uses an internal capability, not a forged user command.

Neither approval nor successful spawn changes existing Message/Task/Result provenance. New managed-run observations identify the product's observation mechanism, not verified content authorship. Keep manual-agent/non-confinement disclosure unconditionally visible, replacing only the obsolete claim that the product never starts anything. Do not claim agent credentials, environment contents or provider endpoints are inaccessible to the child. Secret values stay out of state/events/transport diagnostics; arbitrary child output can itself contain secrets and is not promised redacted.

## Start, stop and operation completion — items 1 and 2

| Trigger | Durable state and effect |
| --- | --- |
| Authorized StartRun | Commit Starting + Pending start operation + stable attempt/dispatch intent and receipt/operationID before any spawn. Reject a second active run for the same agent initially (Conflict). |
| Dispatch | Confirm intent durability; durably mark dispatch attempted, then call Start once within the host lifetime. A persisted attempted marker is not evidence of successful spawn. |
| Started observation | Commit Running and successful start operation with observed timing; start success means launch observed, not task completion. |
| Definitive spawn failure | Exited with spawn-failure outcome and failed operation; distinguish no child created from unknown outcome. |
| StopRun | Unknown run: NotFound, unchanged state (item 2 D3 is the expected negative result, despite its table label). Active run: commit Stopping + stop operation/intent and reason, then request termination. |
| Exit observation | Exited only after owned process group termination is established; persist outcome and complete waiting stop operation. Parent exit alone is insufficient. |
| Unknown ownership/effect | RecoveryRequired for run and affected operation; no automatic spawn or unsafe signal. |

Stop while Starting cancels a definitely undispatched intent without creating a child; if dispatch has begun, serialize start/stop and terminate any successfully created group. Same request replay returns its original operation; repeated stop under a new ID joins existing stop work, or returns a completed no-op for an already exited run, without duplicate signaling workers. SetRunBudget on Exited is rejected without mutation. Stop/budget racing natural exit uses serialized facts; record natural exit honestly, never manufacture a budget-caused exit from an after-the-fact timer.

Adapter creates a dedicated process group for each supported run, uses graceful then forced group termination, and observes both the direct child and remaining owned group. Record attempts and outcome separately. PID/group numbers are private adapter data, never sufficient after restart. Never signal unrelated/manual processes or a reused identifier. If ownership/group disappearance cannot be established, report uncertainty, not Exited. Forced-signal failure/timeouts surface failure/RecoveryRequired; no success based solely on sending a signal. Go's process-group setup and Wait semantics are documented in [syscall](https://pkg.go.dev/syscall?GOOS=darwin) and [os/exec](https://pkg.go.dev/os/exec); native tests remain necessary for our stronger group-lifecycle contract.

## Hard budgets — item 3, both paths required

Elapsed limits are positive integer milliseconds of monotonic time since observed process start; updates change the total allowed duration, not the timer origin. No supplied field means no change; both omitted is InvalidArgument; zero is invalid. Initial profile limits can apply before start. Tightening below already consumed time/tokens immediately schedules enforcement. A limit must never be advertised as installed before core validation/commit succeeds.

Core arms the Clock deadline, rechecks budget revision/state on wakeup, records budget trigger evidence and Stopping, then calls the same termination path used by StopRun. Stale timers do nothing. Elapsed path fires at `elapsed >= limit`; token path fires at reported cumulative usage `>= limit` (at least as strict as exceeding). Persist trigger type, limit/revision, observation, termination dispatch and observed exit separately. Test children run until stopped; a naturally short-lived program cannot prove enforcement.

For tokens, ship at least one real reporting execution path, not only Unsupported responses or a mock. An approved profile selects a documented reporting wrapper/protocol on a dedicated inherited local descriptor; the supervisor tags observations with the run/attempt rather than trusting an ID in message text. Use monotonic cumulative counters and sequence deduplication; reject reset/negative/overflow/malformed records. Ordinary stdout text and mailbox sender claims are never usage authority. If reporting is unsupported, reject token fields atomically with Unsupported while leaving all old budget fields unchanged.

“Hard reported-token budget” means real termination based on reported usage, not a guarantee on unreported provider spend or instantaneous stop at the exact token. A bounded reporting heartbeat is required when a token limit is active; a lost/stalled/malformed channel triggers a distinct telemetry-failure stop rather than quietly counting zero. A lying provider/child can underreport; this mechanism does not authenticate usage. Reporting latency and termination latency permit overshoot and must be disclosed.

Do not let failed state persistence keep an owned process running indefinitely after a detected breach: suppress new effects/starts, attempt bounded emergency termination using the already-established ownership handle, and expose controller failure. On recovery retain uncertainty about missing audit facts, never fabricate a durable trigger record. The normal path records intent before effects; this narrowly scoped fail-stop exception cannot authorize spawn or an unowned kill.

## Crash recovery — item 4

New host acquires the real workspace lock and reconciles nonterminal runs before accepting new managed starts. Never replay persisted Start as though it were an idempotent external effect. Definitely undispatched starts may be cancelled; an attempted/ambiguous start goes through Recover. A stable run ID or receipt does not prove a child exists or that another spawn is safe.

The conservative Phase 3 adapter may return Unknown after host death whenever it cannot establish ownership; mark RecoveryRequired, block restart of that run/agent, and show manual resolution instructions through CLI. No adoption or killing by PID alone, and no automatic duplicate spawn under a new request ID. A future stronger platform identity mechanism may improve recovery, but is not required to manufacture certainty now. Never reconstruct continuous monotonic elapsed time from wall-clock timestamps. Budgets cannot be promised enforced through host death with this design; surviving uncertain processes require explicit human inspection. That limit accompanies the crash/recovery behavior, not a hidden continuation of Running.

Allow human-admin resolution only after explicit confirmation that prior execution has been dealt with; record an administrative resolution as such, not an observed natural exit or verified process death. Exact resolution UI/proof policy needs security review before enabling restart. No default auto-clear. Historical output remains readable with an interrupted/unknown terminal capture marker; absent observations remain absent.

## Output and failure isolation — item 5

Capture stdout/stderr concurrently with bounded buffers. A single per-run journal orders chunks by capture, allocates global byte offsets only after durable append, tags channels, and distinguishes readable complete frames from torn tails. Bytes across channels have capture order, not claimed causal order. Byte reads may split UTF-8; renderers handle partial characters and untrusted terminal controls. RunOutput is separate from state events and carries no authority.

On graceful completion drain both channels before Finish(Complete); process exit and capture completion are separate facts. Descendants retaining descriptors must not hang shutdown indefinitely: bounded drain then Finish(Interrupted), with honest incomplete capture. Storage exhaustion/write failure reports capture failure and triggers core stop; never drop output silently or block budget enforcement behind a slow reader. Readers resume by offset; retention loss reports CursorExpired/earliest offset. Reader cancellation drops only subscription resources. After crash recover complete persisted chunks, report any unknown gap without invented bytes or byte counts.

## Delivery sequence and acceptance map

| Slice | Exit coverage and required evidence |
| --- | --- |
| Neutral records/session + host file transport + serve | 6; real host subprocess, two detachable clients, sole writer and explicit shutdown. |
| Durable approvals/intents + real supervisor | 1–2; shipped CLI start/stop, denial/forged-message negatives, parent+worker termination on both targets. |
| Budget policy + Clock wakeups + reporter | 3; deterministic policy clock plus real elapsed and reported-token stopping; unsupported reporting negative. |
| Crash reconciliation + journal | 4–5; kill host without Close, CLI observes RecoveryRequired/no duplicate, output replay/gap and cancellation evidence. |
| Both frontend paths | 7; UI-09–13 and crossover, retain UI-01–08 invariants and documented limits. |
| Disclosure and guide | 8–9; unchanged old provenance, revised scoped disclosure and independent cold-read guide run. |

Kevin chooses package/file names, typed observation representation, file transport framing/poll intervals, journal framing, and native adapter mechanics within these constraints. Before tests run, Kelly fixes numerical escalation/observation bounds and independent fixture expectations; both native targets owe actual evidence. No Windows expansion, external network transport, terminal/PTY support, arbitrary daemon-tree containment, automatic restart, task retirement/reassignment, trusted sender identity, or provider-metering guarantee is decided here. No code or tests changed/run, and no .git writes.
