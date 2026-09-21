# H101-148 — Agent-orchestrator integration

2026-09-21. **Proposed design for a later phase, not approved for construction.** Angela's H101-146 product ruling and god's H101-148 dispatch supply the scope. Phase 3 items 1–7 come first. The owner's before-Phase-4 priority applies to installed-tool execution, not a deadline or construction approval for this orchestration phase. No graphical interface, embedded terminal, new model service or implicit installation/authentication is included.

**Lead principle: skill instructions and tool hiding are not enforcement; host policy is.**

## Adapter choice

Choose **one documented orchestration skill using a restricted, structured CLI session** backed by the continuing Phase 3 host. Do not require Model Context Protocol (MCP) as well. The skill teaches inspection, task decomposition, worker assignment, own-message handling, blockers, human questions and evidence summaries. The installed tool invokes ordinary local commands; a trusted CLI proxy translates them into the same neutral session API and Phase 3 file transport. It cannot bypass core authorization by choosing another command spelling.

A delegated CLI session is a new policy-bearing entry mode, not permission to pass arbitrary `-caller` or reviewer configuration to today's CLI. Structured versioned responses must preserve domain IDs, revision, receipts, operation status and ADR 0004 uncertainty. Provide snapshot/query, bounded polling and request resolution so interruption never requires guessing whether a command succeeded. The skill is an instruction document, not a daemon, credential store or authorization mechanism. [ADR 0006](../adr/0006-orchestrator-skill-and-delegated-session.md) records this choice.

Compatibility is established by a real workflow using an explicitly selected installed/authenticated tool and its supported invocation mode. No claim about every listed tool follows from one success. If that selected tool cannot invoke this CLI workflow, report a compatibility gap; do not silently introduce MCP or a provider service. Additional tool compatibility is separately evidenced.

## Delegation is durable host policy

A human-admin session creates a delegation containing: immutable delegation ID, workspace ID, bound orchestrator principal, bounded objective text, allowed task scope/operations, eligible registered worker IDs, quantitative limits, validity/revocation state and a policy revision. An agent session cannot create, broaden, replace or reactivate this record. Registration alone grants no delegation. Each delegated session is frozen for its lifetime to exactly one registered AgentID, one immutable workspace ID and one delegation identity; no command may select another acting identity. Claimed-identity fields never establish the authority used for delegation/grant/quota checks (target IDs only select objects to check against that bound authority). Workspace rebinding is prohibited: revoke the old delegation and issue a new one through a fresh human decision and new session; no in-place scope edit or inherited handle is valid for another workspace. Objective text records intent; the host cannot determine whether arbitrary natural-language work actually advances it.

Coordination and process authority are separate grants. Default coordination access includes bounded workspace inspection, task creation/initial assignment to allowed eligible workers, permitted task progress changes, own-message send/acknowledgement, human-decision requests and evidence summaries. It grants no StartRun, StopRun or budget administration. Reading a workspace reveals its content to the installed tool; neither the skill nor this design confines that tool's provider connection.

A human may separately grant start/stop for specific worker IDs and **already-approved immutable profile revisions**, with allowed run scope, cumulative start count, concurrent-run ceiling, per-run elapsed/token ceilings and aggregate reported-usage limits. This is an **explicit widening** of H101-128's initial own-agent-only policy, confined to this later phase. An agent may tighten granted run limits, never raise/remove the human ceilings. No wildcard grant inferred from membership, profile name, ownership of a task, sender claim or successful prior start. Stopping a worker does not imply authority over every historical run: require the grant's explicit run selection, initially runs started under this delegation.

Enforce policy inside the core for every public operation, with host-established session binding. Re-evaluate delegation revision, revocation, object scope and quota in the mutation that records the command. Reserve capacity atomically with accepted task/run intent; parallel requests cannot oversubscribe it. Same-request replay creates no new charge; new IDs do not evade cumulative counts. Failed spawn may release concurrent capacity on definitive evidence, not erase the cumulative authorized-start charge. Unknown execution retains its reservation until explicit resolution. Counter reset, grant replacement and extensions are human decisions, never an agent's retry technique.

Task limits should include total created and active tasks; process limits include cumulative starts and simultaneous runs, not only a per-run timer that a restart resets. Aggregate reported tokens remain reported usage, not audited provider billing. Unknown/missing telemetry follows Phase 3's failure-stop policy. Exact units, ceilings and reporting support are explicit grant fields and must be fixed before acceptance tests, not hidden defaults.

## Enforced behavior versus instructions

All enforcement below is **required future code on supported host paths**, not a claim about today's implementation or a defense against hostile same-user file access.

| Rule | Host enforcement | What instructions cannot guarantee |
| --- | --- | --- |
| Human acceptance/rejection survives | Never issue human-review capability to delegated sessions; reject Accept/Reject and attempts to forge reviewer fields. Only authorized human decisions change review state. | An agent can recommend acceptance, lie in prose or misdescribe evidence; text is not a decision. |
| No human present | AwaitingReview stays until a human decision. A persisted open human-decision request blocks dependent progress until a human response. No timeout or silence creates an answer. | Host cannot detect every semantic dilemma or force the agent to recognize that it should ask. |
| Task/worker scope | Check allowed operations and registered, eligible target workers atomically. Coordination grants neither process control nor retrospective reassignment. | Suitability of a worker, task quality and faithfulness to the objective remain judgment. |
| Own messages only | Bind delegated sender to its principal and acknowledgement to the addressed recipient; reject attempts to acknowledge for workers. | Message content remains unverified and can contain prompt injection or false claims. |
| Process and budget authority | Separate cross-worker grants, immutable profile revision, reservations, ceilings and usage enforcement. Reject profile approval/edit, limit increases and ungranted stop/start. | Cannot stop the installed tool launching processes outside the host or spending through its own provider session. |
| No scope expansion | No delegated grant administration, human session minting, tool authentication, profile administration or stronger subdelegation. Initially prohibit all subdelegation. | A skill prohibition cannot prevent shell/file actions available outside the supported API. |
| Revocation | Durable policy revision invalidates future mutations and undispatched effects; agent cannot revoke the revocation. | Previously returned data cannot be recalled; existing processes are not automatically stopped. |

Task result reporting retains current-assignee checks: an orchestrator reports results only for work assigned to itself, or submits a separate attributed summary/recommendation. It cannot impersonate a worker's report. “Request rework” means send a recommendation or ask a human to reject a pending result; it is not permission to invoke RejectTaskResult or bypass AwaitingReview. Generic status transitions must not become a back door to Done, out of pending human review, or around a decision blocker.

## Human decisions need records, not message interpretation

Introduce a durable HumanDecisionRequest with task/objective association, question, requesting principal/provenance, open/resolved status and revision. Creation and the corresponding task blocker commit atomically. A separate human-authorized answer records the responder, time and answer against the open request; agent-provided `answeredBy`, quoted messages and UI text never populate those fields. Answering does not itself accept a result, start a process or clear unrelated blockers. An explicit subsequent permitted transition consumes the resolved prerequisite and retains the answer history.

The skill must surface undecidable work and poll for responses. Once a request exists, the host prevents delegated clearing/cancelling of that human-decision blocker or bypassing it through another task transition. But the host cannot prove the agent asked every necessary question or that a fabricated prose answer is truthful. Distinguish enforced open-request waiting from that instructional obligation. Human decision requests and replies must remain visible/queryable even while no agent session is connected.

## Identity invariant and all-entry-point checks

H101-109/H101-119 remain governing: stored SenderAgentID is a claim, provenance records the bound submitting scope, and registry membership does not authenticate either. Do not globally force these fields equal or rewrite historical data. The delegated role's **own-message-only** restriction is an explicit additional authorization predicate for this new role; ordinary previously authorized differing-sender interactions retain their contract. Test both populations so the new rule cannot erase the old invariant.

Mailbox/message-kind traffic cannot create, extend or rebind a delegation or session. No message, attachment, result text or tool output can supply process authority, answer a human question, change a grant or replace the session principal. A valid sender registration does not change this. Verify negatives through CLI input and any future adapters, not merely by hiding tools from a skill menu.

The human admin path must be a distinct host-authorized session, not a role selected by an untrusted `--human`, `--caller` or reviewer list. Every supported ingress—including legacy CLI modes and bootstrap—must honor that separation. If an agent can mint a human session through an ordinary API, that is an implementation defect, not the same-user exception. Reading a human handle from disk, forging control files or editing state as the same OS user is the explicit residual limitation described in H101-135. No skill, opaque token or hidden command turns that into isolation; stronger assurance requires a separately approved OS identity/sandbox design.

## Revocation and effect ordering

Check grant validity again before marking a pending start dispatch-attempted, serialized with revocation. If revocation commits first, cancel undispatched work and release only reservations known safe to release. If dispatch-attempted commits first, execution may already be in flight: return that fact with the revocation outcome and affected run IDs. Do not promise revocation prevents every physical spawn after its timestamp. Retain H101-128's durable marker/unknown-outcome rule; no automatic redispatch to compensate for uncertainty.

Human revocation remains callable without the orchestrator and prevents new mutations, including budget changes, from its session. Preserve existing budget enforcement and human StopRun; revocation itself neither kills a process nor grants the revoked principal emergency authority. Read-only recovery of its own prior receipts may remain available so denial is not confused with rollback; no recovered receipt replays an effect. Revocation should return an outstanding-work view distinguishing cancelled pending intent, dispatched/active work and unknown work. Human chooses a separate explicit stop action.

## Implementation boundary and evidence

This is future core/API/host-policy work, not a skill-only packaging task. Extend existing command/query/store roles; keep presentation and the skill out of privileged assembly. Eligible-worker lifecycle checks must use the authoritative product retirement policy. H101-38 reassignment is separately unapproved: initial assignment is supported here; moving an existing owner must wait for that capability's explicit authorization/design, not be simulated by rewriting a task.

Before acceptance, run one real installed tool through inspect → decompose/assign → own-message exchange → blocker/human answer → evidence → human review. Remove the human and demonstrate persisted waiting; restart/rebind without lost requests or wider grants. Exercise direct malicious CLI payloads for acceptance/rejection, fake answers, worker impersonation, profile edits, budget escalation, unknown/retired workers, cross-workspace IDs and scope delegation. Race parallel quota consumers and revocation against dispatch; confirm active runs survive revocation unless separately stopped. Revoke an agent attempting a denied alternate CLI/bootstrap path. Preserve the old differing-sender fixture for ordinary roles and prove delegation cannot turn it into process authority.

No runtime enforcement, security acceptance, tool compatibility or successful trial is claimed here. Creed reviewed the proposed shape and found it sound subject to four explicit invariants: lifetime identity binding, in-flight revocation semantics, immutable workspace binding and the inherited control-channel limitation. Creed completed a clean second pass on both documents on 2026-09-21: all four invariants correctly incorporated, no new findings (review message `creed-2026-09-21-h101-148-secondpass-reply`). This is design-review evidence, not construction approval or runtime verification. The decision least certain is allowing workspace-wide reads while restricting mutations by delegation: it fits the existing snapshot contract, but a future objective-level confidentiality requirement would require a different query policy, not stronger skill wording.

Sources: Angela's H101-146 memory/ruling; H101-148 dispatch; H101-128 and H101-135; H101-109/H101-119; H101-38; ADRs 0002–0005. No code, tests, installed-tool calls or .git writes.


## H101-150 — delegation-scoped reads and quota strength (2026-09-21)

**This amendment supersedes the earlier workspace-wide-read choice and uniform “quota” shorthand; original text remains as history.** Agree with Creed's H101-149 ruling: normal authorized disclosure to every orchestrator is not equivalent to hostile same-user filesystem compromise. Scoped reads are required for the proposed role from its first implementation, not deferred to a future skill instruction. This is design only; construction and a before-release deadline remain unapproved.

### Distinct query capability

Give the role a concrete `DelegatedSession` exposing `GetDelegationView`, scoped task/message/result/human-decision lookups and scoped observation. It must not expose or be downcastable to the ordinary unscoped `FrontendSession.GetSnapshot`, registry enumeration or global event/output streams. Keep neutral data types in the API layer, policy in core, binding in host, and transport in its adapter. Existing unscoped views for other authorized roles retain their contract; no new architectural port is required.

The host derives delegation identity from the frozen session, never a request's asserted delegation ID. Compute visibility from authoritative task membership on one consistent state/policy revision:

- Tasks created through the delegation are atomically stamped with its identity by the host. Existing tasks enter only through an explicit human-authorized binding record. Assignment to an eligible worker, a matching objective string, or an agent-supplied reference does not add membership. Membership changes advance policy revision; delegated sessions cannot add/remove bindings themselves.
- Eligible workers appear as identity/reference records only: allowed AgentIDs and necessary eligibility facts, not their unrelated tasks, messages, profile configuration or registry metadata.
- Messages, results and human-decision requests are visible only when their authoritative task reference is in scope. Unlinked records stay invisible by default, even if the orchestrator is their sender. Delegated sends/decision requests must bind a scoped task; worker replies preserve that link. A reference does not recursively include another task or a reply's out-of-scope parent.
- Process-management grants do not imply access to a worker's other run history. Run/operation metadata and output require an authorized run binding to a scoped task and an explicit observation permission; unbound existing runs may receive an explicitly granted stop action without exposing their output. Responses reveal only the minimum authorized operation result.

Return purpose-built projected records: do not serialize a complete snapshot and ask the skill to filter it. Remove out-of-scope nested identifiers, expanded objects and links, reporting a typed withheld-reference marker where needed. Artifact references are not permission to fetch arbitrary files; any host artifact-read endpoint requires independent scope/path checks. In-scope free text or child output can itself quote unrelated content; record filtering is not semantic redaction or a data-loss-prevention guarantee.

Apply the same policy to point reads, lists/search/counts, event payloads, output, errors, receipts and transport responses. For a valid session, nonexistent and out-of-scope object lookups both return NotFound without confirming hidden existence. Wrong workspace or invalid/revoked session is denied before object lookup. Scoped observation uses an opaque delegation/policy-bound cursor, not a global event feed filtered in presentation; changes to membership invalidate the cursor and require a new scoped view. Check current policy before releasing each response/stream batch and discard queued material made inaccessible by revocation. Already delivered content cannot be recalled.

Preserve original receipt identity/revision under ADR 0004; return only the session's own authorized receipt records, never a raw ledger. Existing workspace-revision numbers in receipts can reveal aggregate activity; this amendment does not promise timing/metadata isolation. Full mutual confidentiality would require review of that metadata, shared worker output, storage and provider exposure as well as record filtering.

### Named revisit condition: multiple confidential objectives

The current product assumption is **one individual, one bounded project, one workspace trust domain**. The scoped API narrows ordinary exposure now. Before a workspace is offered for simultaneously delegated, mutually confidential objectives—separate clients, projects or tenants—reopen this design with security/product review and prove the stronger confidentiality policy before that use ships. Workspace-wide delegated reads are unacceptable for that use; the same-user exclusion cannot waive the required API policy. This trigger remains even after scoped reads exist because they alone do not establish tenant isolation. The prior clean four-invariant review is not approval of this new use case or evidence that this amendment is implemented.

### Separate quota guarantees

| Limit | Guarantee on supported host paths | Limitation |
| --- | --- | --- |
| Task/start counts and concurrent-run ceiling | **Hard host-enforced admission ceilings:** atomic reservations and host observations prevent extra admissions; retries do not reset counters. | Unknown runs retain reservations; actions outside the host remain outside this control. |
| Reported-token ceiling | **Soft with respect to actual consumption; conditional on self-reported telemetry:** core triggers real termination at the reported threshold. | Underreporting and reporting/termination delay allow actual usage or spend beyond the limit; not a hard provider-billing cap. |

Keep these as separate rows in capability descriptions, grants and acceptance summaries. Atomic accounting does not make the measurement trustworthy. Missing/malformed telemetry still invokes the declared failure-stop policy; it does not solve dishonest telemetry. Elapsed-time enforcement retains Phase 3's separately ruled timing/exit semantics.

Required future evidence: two delegations sharing a worker cannot enumerate each other's records; guessed IDs, nested references, global-event attempts and run output do not bypass scope; only explicit human task binding broadens reads; revocation/membership change blocks queued reads and expires cursors. Independently demonstrate hard admission races and reported-token termination, including honest underreporting limitations. No code/tests or .git writes.
