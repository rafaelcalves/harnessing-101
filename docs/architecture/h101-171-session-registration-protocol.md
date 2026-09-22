# H101-171 — Register an existing agent session for participation

2026-09-22. **Proposed specification; no construction approval or compatibility claim.** Source review at `f347f30`. This document specifies the owner's reshaped H101-163 request; it does not implement it, amend quality gates, or authorize ADR 0006 orchestration. Proposed operation names below are application contracts, not shipped CLI commands.

## Owner intent and the unresolved delivery scope

Owner wording, relayed in dispatch `2026-09-22T10-19-34-917Z-347d88`:

> I want each worker/agent session to be linked to a profile on our side identifying which agent is what. for example. if the user has they're own open session with claude code, they could register that claude code session and it would be receiving messages based on the message protocol we have

**Interpretation:** support registration of an already-open conversation and link it to the product's agent/profile identity. God is right about the direction of attachment. “Announces itself,” however, must not mean an arbitrary caller can claim an agent identity without user pairing. The user authorizes the binding; the session redeems it.

The answer selects participation rather than blanket tool deferral. It **does not say** that managed launch is removed, nor explicitly require both entry modes for Claude Code in the current delivery. Existing managed-start work and criteria remain in place until explicitly changed. Architecturally, both modes can use one participation contract, but neither mode's evidence proves the other.

**Owner decision still needed:** attachment first with managed-launch evidence explicitly deferred/resequenced, or both attachment and managed launch in the current scope? This is a delivery/evidence choice, not a reason to build two protocols. It has been raised to god. [Item 1 D13 and Layer B](../quality/phase3-exit-criteria.md) require managed start; an external session cannot satisfy them by writing a manifest. Kelly's parallel H101-172 ruling owns that criterion disposition. This specification grants no waiver or deadline.

## Four separate identities

| Identity | Meaning in this proposal |
| --- | --- |
| AgentID | Durable participant/address in the existing registry. Tasks and mailbox messages continue to use it. |
| Profile binding | User-selected existing profile reference plus a frozen revision or content digest. Describes the intended tool/configuration, not an observed fact about the open tool. |
| ParticipationSessionID | New opaque identity for one paired external conversation/connector, with a binding generation. Several short-lived CLI calls can use this same participation session. |
| RunID / OperationID | Existing product-managed execution/command identities. External registration creates neither and does not adopt the process. |

The current [Agent](../../internal/core/domain/records.go) has `ProfileID`; the current `Profile` is an approved execution specification, not a conversation record. Keep session lifecycle out of both records. Add a separate durable participation binding containing workspace, agent, frozen profile reference, external-session origin, claimed tool/conversation label, generation, lifecycle state and provenance. Store no provider credential or conversation transcript in it. A tool-supplied conversation ID is optional, unverified metadata, never a primary key or authorization input.

**Proposed first scope:** pair to an existing agent and existing profile selected by the user; do not create either implicitly. Require the selected profile to match the agent's configured profile reference. If that reference is missing, setup must resolve it explicitly before pairing. A later agent/profile edit never silently rebinds a live session: invalidate its binding and require user reconfirmation. One profile may describe several agents; the exclusive attachment slot is per agent, not per tool executable or profile.

**Product question:** if the owner means a lightweight identity card by “profile,” requiring today's execution-approval Profile for a manually opened tool is unnecessary setup friction. That would need a separate descriptive profile model or an explicit change to the existing model. Do not fabricate a launch approval merely to register an external session. The first scope above is a proposal pending that interpretation, not an assertion that the owner chose the current record type.

## Pairing and what it proves

A continuing host is required for this first design; it already holds the writer lock and mailbox delivery loop. Reuse its local file transport, but **not today's unrestricted attach request**: [AttachPayload](../../internal/adapters/transport/protocol.go) currently supplies only `callerAgentID`. It has no external-conversation pairing or freshness proof.

1. **User prepares pairing.** A trusted local administration action selects workspace, AgentID and profile binding, checks the attachment slot, and creates a short-lived, single-use invitation. It returns a nonsecret invitation ID and a protected local credential reference. This is not inferred from a sender field, PID, profile name or tool claim.
2. **The open session joins.** The user deliberately instructs that conversation to invoke the local connector using the supplied invitation. The connector redeems it and answers a fresh host challenge. Invitation expiry and challenge freshness use host-monotonic deadlines; restarting the host invalidates unfinished invitations. Credential material belongs to the protected control channel, never task/message bodies, model instructions, domain payloads, snapshots, logs or manifests. Pass a local reference rather than embedding a bearer secret in a prompt or command argument.
3. **Host commits the binding.** Atomically consume the invitation and establish ParticipationSessionID, generation, agent/profile binding and initial connected observation. Bind an explicitly limited worker capability. Only then return registration success. A failed or uncertain commit gives no authority to mutate; recover the same registration request through the pairing protocol rather than minting another session.
4. **Subsequent calls prove possession.** Each call carries the bound control credential and current generation, with transport replay protection. Neither payload-selected caller/profile nor an old generation can change the acting identity. Domain mutations retain their separate stable RequestIDs and ordinary receipt deduplication.

**Proof limit:** this proves that the endpoint holds the user-provided pairing capability and responded to the host. It does not cryptographically establish that the program is Claude Code, that it is the intended visible conversation, or that it is still thinking. A human sees the pairing ID returned in that conversation and confirms the intended association. Product labels must say “user-paired; tool identity unverified,” not “verified Claude session.” Exact provider-conversation attestation would require a separately investigated tool-specific mechanism; no such proof exists in the inspected code.

A same-user process can steal credentials, alter files or invoke trusted local-user tooling. Opaque handles do not solve that boundary. [ADR 0002](../adr/0002-manual-agent-exposure-in-phases-1-2.md) and [H101-135](h101-128-phase3-supervision.md#h101-135--transport-rationale-and-exposure-scope-2026-09-21) remain relevant limits, not automatic risk acceptance for this new workflow. Before construction, security review must settle the pairing/control-secret lifecycle, bootstrap authority and the interaction with legacy trusted CLI entry points. This document does not claim hostile same-user isolation.

## Collisions, reconnection and stale registrations

Registration states are Connected, Stale, Disconnected and Closed; retain a closure reason such as voluntary detach, user revocation or replacement. An invitation is not yet a registration. A credential-bound `UnregisterParticipationSession` closes only its own binding, invalidates its generation and releases the attachment slot for a subsequent user-authorized pairing. It does not stop the external program or alter assigned work. Stale/Disconnected registrations retain their slot until revalidated or explicitly closed/replaced; timeout alone cannot authorize takeover.

| Situation | Required behavior |
| --- | --- |
| Unknown agent/profile or mismatched binding | NotFound or Conflict respectively; no implicit registry/profile mutation. Invalid invitation or credential is Denied without revealing another binding. |
| Same registration request repeated | Recover the same committed binding, never allocate a second session. The connector retains an ephemeral recovery credential until confirmation; losing all proof requires explicit user replacement. Raw credentials are not stored in domain receipts. |
| Second session for an occupied agent | Conflict. User must choose another agent or explicitly replace the binding; identical tool/profile labels do not justify takeover. |
| Connector reconnects during current host lifetime | Reauthenticate the same binding, rotate transport credentials/generation as needed and resume inspection. Recheck profile/policy and resolve earlier uncertain requests before new mutations. |
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

**Registration does not wake an arbitrary terminal conversation.** First propose cooperative pull: the user teaches the open session to call `CheckIn`/pending-message inspection at turn boundaries and before reporting, or requests a check-in explicitly. A connector may wait for new messages, but cannot claim the model consumed them. Continuous automatic wake requires a documented tool hook/integration that actually delivers a new turn to that same conversation. No terminal keystroke injection or imagined universal hook is specified.

**Owner clarification:** is cooperative check-in sufficient, or does “receiving messages” require automatic wake while the session is idle? If automatic wake is required, selected-tool hook feasibility is a prerequisite, not a transparent implementation detail. Instruction-only polling cannot be accepted as that behavior.

## Worker participation and authority

Proposed `CheckIn` returns the bound identities/profile reference, current assigned tasks and revisions, pending incoming messages with provenance, and explicit connection/uncertainty status. Use bounded structured responses and cursors; never silently truncate. On expired event cursors, reload authoritative task/message state. A connector protocol document teaches the worker the sequence below; it is not an enforcement mechanism.

| Expected worker action | Existing product meaning | Required new binding behavior |
| --- | --- | --- |
| Query its assignment | Task/query records identify current owner and status | Derive actor from pairing; no self-assignment or new-task authority is implied. |
| Start/resume or block assigned work | Task transitions record reported progress and blocker reason | Check current assignee and revision for this worker path. Merely attaching does not set Doing. |
| Read and acknowledge messages | Recipient acknowledges explicit receipt | Only its own recipient identity; no bridge-generated acknowledgement. |
| Contact a peer | Send a task-linked message; peer is a registry address | Worker sends as itself; profile or session labels grant no authority over the peer. |
| Report evidence | ReportTaskResult records attributed summary/artifacts and AwaitingReview | Current-assignee/revision checks retained; no automatic acceptance. |
| Rework after rejection or wait | Human decision determines Doing or Done | Agent cannot accept/reject, answer as a human, or interpret silence as approval. |

These are proposed restrictions for the new worker entry path, not silent changes to ordinary caller/sender compatibility. Do not rewrite historical claims. The worker capability exposes no profile approval, reviewer configuration, identity switching, process administration, reassignment, delegation or human-review commands. Existing core checks that do not enforce these worker-specific predicates require explicit implementation; frontend command hiding alone is insufficient. Reads are limited by the selected worker interface, but the workspace still represents one user's bounded project; this is no tenant-confidentiality claim or ADR 0006 `DelegatedSession` implementation.

The product can enforce allowed commands, bindings, revisions and recorded review. It cannot force an idle model to poll, assess whether it understood a message, guarantee useful peer collaboration, detect every dilemma or validate prose/artifact correctness. It also cannot stop an external tool using shell or provider access independently. Registration should display those limits before the user pairs the session.

## Layering, evidence and construction prerequisites

New registration records and lifecycle/admission/authorization policy belong in core; application request/query types in the neutral API; invitation issuance and binding in trusted host assembly; credential exchange, polling and tool hooks in adapters. Preserve the existing StateStore/Commands/Queries/Mailbox roles. A per-tool descriptor can identify a supported attachment mode and instruction/hook version separately from launch arguments; that schema extension is proposed work, not permission to treat the current launch descriptor as attach-capable.

A registration evidence runner must observe an independently opened tool, deliberate user pairing, the bound profile, a real incoming message and explicit acknowledgement, peer exchange, assigned-task result, human rejection/rework/acceptance, and disconnect/reconnect behavior. It must collect authoritative product records and actual interventions. It must not simulate the worker's commands itself and call that tool participation, or manufacture RunID/managed-start evidence. Record tool/version as observed or claimed, with the distinction intact.

Before acceptance, test invalid/expired invitation and replay, wrong-agent/profile claims, simultaneous joins, old-generation queued writes, lost registration/mutation responses, stale heartbeat, host restart, profile changes and user replacement. Prove that reads/bridge delivery do not acknowledge; pending work survives death; accepted receipts survive credential rotation; and disconnected sessions cannot resume authority silently. A real supported-tool check-in/wake demonstration is separate from deterministic fixture proof. Kelly fixes criteria before construction and retains both native platform requirements where applicable.

**Outstanding decisions, not construction instructions:** owner resolves managed-launch coexistence and idle-message expectations; product clarifies whether “profile” means today's execution Profile; security reviews pairing authority and residual exposure; Kelly rules registration evidence versus item 1. The present specification is complete as a proposed contract with those explicit open decisions. No code, gate changes, tool launches, credentials or provider calls were made for it.

Sources: [blueprint and earlier option A](blueprint.md), [functional feature status](functional-features.md), [core boundaries](boundaries.md), [H101-128](h101-128-phase3-supervision.md), [ADR 0002](../adr/0002-manual-agent-exposure-in-phases-1-2.md), [ADR 0006 and its H101-150 amendment](../adr/0006-orchestrator-skill-and-delegated-session.md), [Phase 3 criteria](../quality/phase3-exit-criteria.md), current API/registry/transport source, and the quoted owner dispatch. No prior acceptance is extended by this proposal.
