# H101-171 — Register an existing agent session for participation

2026-09-22. **Proposed specification; no construction approval or compatibility claim.** Source review at `f347f30`. This document specifies the owner's reshaped H101-163 request; it does not implement it, amend quality gates, or authorize ADR 0006 orchestration. Proposed operation names below are application contracts, not shipped CLI commands.

**H101-178 revision (2026-09-22):** incorporates Angela’s H101-177 product rulings, relayed in dispatch `2026-09-22T10-28-27-008Z-ae3b54`: cooperative check-in is the first registration product; a descriptive participation profile replaces the original execution-profile prerequisite. These product decisions do not authorize construction. H101-174 delivery scope remains open.

## Owner intent and the unresolved delivery scope

Owner wording, relayed in dispatch `2026-09-22T10-19-34-917Z-347d88`:

> I want each worker/agent session to be linked to a profile on our side identifying which agent is what. for example. if the user has they're own open session with claude code, they could register that claude code session and it would be receiving messages based on the message protocol we have

**Interpretation:** support registration of an already-open conversation and link it to the product's agent/profile identity. God is right about the direction of attachment. “Announces itself,” however, must not mean an arbitrary caller can claim an agent identity without user pairing. The user authorizes the binding; the session redeems it.

The answer selects participation rather than blanket tool deferral. It **does not say** that managed launch is removed, nor explicitly require both entry modes for Claude Code in the current delivery. Existing managed-start work and criteria remain in place until explicitly changed. Architecturally, both modes can use one participation contract, but neither mode's evidence proves the other.

**Owner decision still needed:** attachment first with managed-launch evidence explicitly deferred/resequenced, or both attachment and managed launch in the current scope? This is a delivery/evidence choice, not a reason to build two protocols. It has been raised to god. [Item 1 D13 and Layer B](../quality/phase3-exit-criteria.md) require managed start; an external session cannot satisfy them by writing a manifest. [Kelly’s H101-172 ruling](../quality/h101-172-layerb-registration-ruling.md) confirms registration falls outside item 1 Layer B and recommends separate registration acceptance. This specification grants no waiver or deadline.

## Four separate identities

| Identity | Meaning in this proposal |
| --- | --- |
| AgentID | Durable participant/address in the existing registry. Tasks and mailbox messages continue to use it. |
| ParticipationProfileID + revision | Lightweight, user-selected description of the intended tool and participant configuration. Neither execution approval nor proof about the open program/provider. |
| ParticipationSessionID | New opaque identity for one paired external conversation/connector, with a binding generation. Several short-lived CLI calls can use this same participation session. |
| RunID / OperationID | Existing product-managed execution/command identities. External registration creates neither and does not adopt the process. |

The current [Agent](../../internal/core/domain/records.go) has `ProfileID`; the current `Profile` is an approved **execution profile**. Retain those existing meanings. Introduce a distinct `ParticipationProfile` concept and typed `participationProfileID`/`participationProfileRevision` references in the proposed session binding; do not reuse `Agent.ProfileID` or infer that its value identifies this new record. The durable agent, descriptive profile and particular paired session remain three different records.

**Representation:** a participation profile has an opaque ID, immutable revision, display name, user-selected tool label and nonsecret intended participant configuration (such as role or working preferences), plus recorded author/time. These are descriptions, not executable launch arguments, credentials, verified provider settings or permissions. Profile text cannot grant a capability. User-authorized profile creation/revision is distinct from execution approval; an explicit setup flow can create this description without recording launch consent. An agent, descriptor or pairing request cannot create or change it implicitly.

The durable participation binding contains workspace, AgentID, the exact participation-profile revision, external-session origin, claimed tool/conversation label, generation, lifecycle state and provenance. Store no provider credential or conversation transcript in either record. A tool-supplied conversation ID is optional, unverified metadata, never a primary key or authorization input.

**Pairing prerequisite:** an existing agent and an explicitly selected participation-profile revision, followed by explicit user authorization for workspace participation. An approved execution profile and a populated `Agent.ProfileID` are **not** prerequisites, and matching their identifiers is not required. One participation profile may describe several agents; the exclusive attachment slot stays per agent. Creating a new profile revision leaves existing bindings on their frozen revision. Applying a different revision to a bound session requires explicit user reconfirmation and a new binding generation; no silent rebind or authority expansion follows from a metadata edit.

A user may explicitly record/display a relationship to a separate execution profile for the same tool. Such a relationship carries no launch authorization, pairing authorization, credential or identity assurance in either direction. Approving an execution profile does not pair an external conversation; creating or selecting a participation profile does not approve `StartRun`. This split creates no protocol conflict: pairing binds a description and user-granted participation capability, while managed launch independently binds its approved execution configuration.

## Pairing and what it proves

A continuing host is required for this first design; it already holds the writer lock and mailbox delivery loop. Reuse its local file transport, but **not today's unrestricted attach request**: [AttachPayload](../../internal/adapters/transport/protocol.go) currently supplies only `callerAgentID`. It has no external-conversation pairing or freshness proof.

1. **User prepares pairing.** A trusted local administration action selects workspace, AgentID and participation-profile revision, checks the attachment slot, and creates a short-lived, single-use invitation. It returns a nonsecret invitation ID and a protected local credential reference. This is not inferred from a sender field, PID, profile name or tool claim.
2. **The open session joins.** The user deliberately instructs that conversation to invoke the local connector using the supplied invitation. The connector redeems it and answers a fresh host challenge. Invitation expiry and challenge freshness use host-monotonic deadlines; restarting the host invalidates unfinished invitations. Credential material belongs to the protected control channel, never task/message bodies, model instructions, domain payloads, snapshots, logs or manifests. Pass a local reference rather than embedding a bearer secret in a prompt or command argument.
3. **Host commits the binding.** Atomically consume the invitation and establish ParticipationSessionID, generation, agent/participation-profile binding and initial connected observation. Bind an explicitly limited worker capability. Only then return registration success. A failed or uncertain commit gives no authority to mutate; recover the same registration request through the pairing protocol rather than minting another session.
4. **Subsequent calls prove possession.** Each call carries the bound control credential and current generation, with transport replay protection. Neither payload-selected caller/participation profile nor an old generation can change the acting identity. Domain mutations retain their separate stable RequestIDs and ordinary receipt deduplication.

**Proof limit:** this proves that the endpoint holds the user-provided pairing capability and responded to the host. It does not cryptographically establish that the program is Claude Code, that it is the intended visible conversation, or that it is still thinking. A human sees the pairing ID returned in that conversation and confirms the intended association. Product labels must say “user-paired; tool identity unverified,” not “verified Claude session.” Exact provider-conversation attestation would require a separately investigated tool-specific mechanism; no such proof exists in the inspected code.

A same-user process can steal credentials, alter files or invoke trusted local-user tooling. Opaque handles do not solve that boundary. [ADR 0002](../adr/0002-manual-agent-exposure-in-phases-1-2.md) and [H101-135](h101-128-phase3-supervision.md#h101-135--transport-rationale-and-exposure-scope-2026-09-21) remain relevant limits, not automatic risk acceptance for this new workflow. Before construction, security review must settle the pairing/control-secret lifecycle, bootstrap authority and the interaction with legacy trusted CLI entry points. This document does not claim hostile same-user isolation.

## Collisions, reconnection and stale registrations

Registration states are Connected, Stale, Disconnected and Closed; retain a closure reason such as voluntary detach, user revocation or replacement. An invitation is not yet a registration. A credential-bound `UnregisterParticipationSession` closes only its own binding, invalidates its generation and releases the attachment slot for a subsequent user-authorized pairing. It does not stop the external program or alter assigned work. Stale/Disconnected registrations retain their slot until revalidated or explicitly closed/replaced; timeout alone cannot authorize takeover.

| Situation | Required behavior |
| --- | --- |
| Unknown agent/participation-profile revision or mismatched invitation binding | NotFound or Conflict respectively; no implicit registry/profile mutation. Invalid invitation or credential is Denied without revealing another binding. |
| Same registration request repeated | Recover the same committed binding, never allocate a second session. The connector retains an ephemeral recovery credential until confirmation; losing all proof requires explicit user replacement. Raw credentials are not stored in domain receipts. |
| Second session for an occupied agent | Conflict. User must choose another agent or explicitly replace the binding; identical tool/participation-profile labels do not justify takeover. |
| Connector reconnects during current host lifetime | Reauthenticate the same binding, rotate transport credentials/generation as needed and resume inspection. Recheck the frozen participation-profile binding and current authorization policy; resolve earlier uncertain requests before new mutations. |
| User replaces/revokes a binding | Commit revocation and a new generation before admitting replacement writes. Old queued calls fail authorization; already committed effects remain. Revoke neither kills the tool nor recalls delivered content. |
| Heartbeat deadline missed | Mark Stale/unknown availability and deny new worker mutations until explicit revalidation. Do not infer process death, release the agent slot automatically or manufacture acknowledgements. |
| Host restarts | Mark previously connected registrations Disconnected; invalidate old transport credentials. No wall-clock inference of continuous liveness. Require a fresh user pairing to resume the same durable binding or explicitly replace it. Old messages, results and receipts remain. |
| External session exits without unregistering | Eventually becomes Stale under the above rule. Tasks and pending messages retain their ownership and state. No automatic completion, reassignment, replayed work or managed restart. |

Use explicit positive, bounded heartbeat/lease intervals in the pairing response; implementation and QA must fix their defaults before tests, not use timing guesses. Heartbeats describe connector contact only. Record contact timestamps without presenting them as a model-attention signal. A permanently running bridge can stay alive after its conversation becomes unusable; the display must still say “connector contacted,” not “agent working.”

The first scope rejects external pairing to an agent with an unresolved product-managed execution unless a separately specified managed bootstrap binds that exact run. Registration cannot clear a run's uncertainty or serve as evidence that prior execution ended. This preserves the separation from item 4; it does not implement or redefine that recovery work. Likewise, do not admit a new managed start into an occupied external-session slot by accident. If both entry modes are selected, their shared admission rule needs explicit implementation and tests.

## Delivery keeps the existing mailbox meanings

Messages remain addressed to AgentID. Registration points a consumer at that agent's mailbox; it does not create a second message store or change the envelope schema merely to add conversation identity.

The host's mailbox loop continues the existing **queued → published → processed** recording. “Processed” means the controller ingested the envelope, not that a model read it. A session inspects pending messages through the product query surface or a connector wrapping the existing file protocol. Reading, polling, forwarding to a local buffer and successful transport delivery never set `AcknowledgedAt`.

The participating conversation explicitly invokes acknowledgement after receiving the message content. Core verifies that its bound AgentID is the recipient. The existing recipient-bound acknowledgement command/control-record semantics remain authoritative; a passive background bridge must not acknowledge for the model. Repeated delivery and acknowledgement are idempotent by message/control/request identities. Session replacement retains pending messages; it does not forward, delete or auto-ack them. Already acknowledged work can still be unfinished and is found through task queries.

Delivery is at least once, not exactly-once model execution. The connector deduplicates presentation by MessageID where possible; the worker checks task state and preserves request IDs before acting again. If a mutation response is lost, use request resolution and the same payload/ID. If authorization was revoked, a denied retry does not prove the previous mutation failed. Only the user or a freshly authorized session can inspect the retained outcome.

**Cooperative check-in is the first registration product (H101-177).** Promise a durable mailbox that the paired conversation checks when it next acts or when the user asks it to. Instructions tell the open session to call `CheckIn`/pending-message inspection at turn boundaries and before reporting. That is expected behavior, not a guarantee that the model will poll. Do not promise that an idle conversation wakes, reads or answers while the user is away.

A connector may wait for new messages, but a live connector, queued delivery or a successful poll is neither acknowledgement nor evidence of work. Acknowledgement remains an explicit action by the participating session. These are limits of this product, not unfinished first-product requirements.

Automatic wake is a separate possible capability, requiring demonstrated support in a selected tool **and an explicit product decision**. It is not authorized or scheduled here. If the owner later requires unattended responses, that is a **material requirement change, not a wording tweak**. Instruction-only polling cannot satisfy it; no terminal keystroke injection or imagined universal hook is specified.

## Worker participation and authority

Proposed `CheckIn` returns the bound identities/participation-profile revision, current assigned tasks and revisions, pending incoming messages with provenance, and explicit connection/uncertainty status. Use bounded structured responses and cursors; never silently truncate. On expired event cursors, reload authoritative task/message state. A connector protocol document teaches the worker the sequence below; it is not an enforcement mechanism.

| Expected worker action | Existing product meaning | Required new binding behavior |
| --- | --- | --- |
| Query its assignment | Task/query records identify current owner and status | Derive actor from pairing; no self-assignment or new-task authority is implied. |
| Start/resume or block assigned work | Task transitions record reported progress and blocker reason | Check current assignee and revision for this worker path. Merely attaching does not set Doing. |
| Read and acknowledge messages | Recipient acknowledges explicit receipt | Only its own recipient identity; no bridge-generated acknowledgement. |
| Contact a peer | Send a task-linked message; peer is a registry address | Worker sends as itself; profile or session labels grant no authority over the peer. |
| Report evidence | ReportTaskResult records attributed summary/artifacts and AwaitingReview | Current-assignee/revision checks retained; no automatic acceptance. |
| Rework after rejection or wait | Human decision determines Doing or Done | Agent cannot accept/reject, answer as a human, or interpret silence as approval. |

These are proposed restrictions for the new worker entry path, not silent changes to ordinary caller/sender compatibility. Do not rewrite historical claims. The worker capability exposes no participation-profile administration, execution-profile approval, reviewer configuration, identity switching, process administration, reassignment, delegation or human-review commands. Existing core checks that do not enforce these worker-specific predicates require explicit implementation; frontend command hiding alone is insufficient. Reads are limited by the selected worker interface, but the workspace still represents one user's bounded project; this is no tenant-confidentiality claim or ADR 0006 `DelegatedSession` implementation.

The product can enforce allowed commands, bindings, revisions and recorded review. It cannot force an idle model to poll, assess whether it understood a message, guarantee useful peer collaboration, detect every dilemma or validate prose/artifact correctness. It also cannot stop an external tool using shell or provider access independently. Registration should display those limits before the user pairs the session.

## Layering, evidence and construction prerequisites

New registration records and lifecycle/admission/authorization policy belong in core; application request/query types in the neutral API; invitation issuance and binding in trusted host assembly; credential exchange, polling and tool hooks in adapters. Preserve the existing StateStore/Commands/Queries/Mailbox roles. A per-tool descriptor can identify a supported attachment mode and instruction version separately from launch arguments; that schema extension is proposed work, not permission to treat the current launch descriptor as attach-capable.

A registration evidence runner must observe an independently opened tool, deliberate user pairing, the bound participation-profile revision, a real incoming message and explicit acknowledgement, peer exchange, assigned-task result, human rejection/rework/acceptance, and disconnect/reconnect behavior. It must collect authoritative product records and actual interventions. It must not simulate the worker's commands itself and call that tool participation, or manufacture RunID/managed-start evidence. Record tool/version as observed or claimed, with the distinction intact.

Before acceptance, test invalid/expired invitation and replay, wrong-agent/profile claims, simultaneous joins, old-generation queued writes, lost registration/mutation responses, stale heartbeat, host restart, participation-profile revision changes and user replacement. Verify pairing without an execution approval, and that an optional profile relationship transfers neither participation nor launch authorization. Prove that reads/bridge delivery do not acknowledge; pending work survives death; accepted receipts survive credential rotation; and disconnected sessions cannot resume authority silently. A real supported-tool cooperative check-in demonstration is separate from deterministic fixture proof. Kelly fixes criteria before construction and retains both native platform requirements where applicable.

**Remaining decision and review boundaries:** H101-174 remains with the owner: attachment-first delivery versus both attachment and managed launch. Cooperative check-in and the lightweight participation profile are now decided by H101-177; do not ask them again as unresolved prerequisites. Security still reviews pairing authority and residual exposure before construction. Kelly’s [H101-172](../quality/h101-172-layerb-registration-ruling.md) keeps registration evidence separate from managed-start Layer B; neither the profile split nor cooperative check-in changes that ruling. No code, gate changes, tool launches, credentials or provider calls were made for this specification or its H101-178 revision.

Sources: [blueprint and earlier option A](blueprint.md), [functional feature status](functional-features.md), [core boundaries](boundaries.md), [H101-128](h101-128-phase3-supervision.md), [ADR 0002](../adr/0002-manual-agent-exposure-in-phases-1-2.md), [ADR 0006 and its H101-150 amendment](../adr/0006-orchestrator-skill-and-delegated-session.md), [Phase 3 criteria](../quality/phase3-exit-criteria.md), current API/registry/transport source, and the quoted owner dispatch. No prior acceptance is extended by this proposal.
