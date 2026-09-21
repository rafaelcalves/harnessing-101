# ADR 0006 — Orchestration skill over a host-enforced delegated CLI session

Status: **Proposed, planned phase only**, 2026-09-21. Stanley, H101-148. Construction is not approved; Phase 3 remains prior work.

## Decision

Prove the first agent-orchestrator workflow with one documented skill invoking a structured, restricted CLI session attached to the existing continuing host. Require neither a Model Context Protocol (MCP) server nor both interfaces. Reuse neutral API requests, durable receipts and the existing transport; add host/core delegation enforcement rather than trusting skill instructions.

The user selects an already-installed/authenticated tool and delegates one workspace/objective explicitly. The session is frozen to one registered AgentID, immutable workspace and delegation; changing workspace requires revocation and fresh human issuance, not a payload-selected identity. Default coordination access carries no process authority. Separate human grants may authorize bounded start/stop of other registered workers under immutable approved profile revisions: this is an explicit later-phase widening of Phase 3's own-agent policy. Human acceptance/rejection, decision answers, profile approval, scope extension and increased limits remain nondelegable.

## Rationale and alternatives

A skill plus CLI exercises the existing product path and keeps another protocol out of the first compatibility proof. A skill alone cannot enforce anything; ordinary unrestricted CLI access is also insufficient. A future MCP adapter could improve tool discovery and typed interactions, but must reuse the same caller-bound policy and demonstrate compatibility independently. Tool-list filtering or prompt wording cannot replace checks at every host ingress.

The host can enforce API authority, quotas, explicit unresolved-decision waiting and revocation ordering. It cannot guarantee honest natural-language summaries, recognition of every dilemma, confinement of independently launched programs, or resistance to same-user theft of human session handles and direct file tampering. Preserve those limits explicitly; do not claim authenticated human identity or secret isolation.

## Consequences and review

[H101-148's architecture](../architecture/h101-148-agent-orchestrator-integration.md) defines grants, evidence, revocation, the own-message restriction for delegated roles and the preserved ordinary sender/provenance contract. It requires future human-decision records and policy checks, not just documentation. Workspace-wide reads follow the existing model; objective-level confidentiality is not promised and is the main design uncertainty.

Revisit the adapter choice only if the selected installed tool cannot use the documented CLI flow or a separately approved integration need justifies MCP. The owner's installed-tool priority before Phase 4 does not authorize or schedule this whole phase. No graphical interface, terminal embedding, tool authentication, new provider service or implementation is included. Creed completed a clean second pass on 2026-09-21, confirming all four invariants and reporting no new findings (review message `creed-2026-09-21-h101-148-secondpass-reply`); construction approval and runtime verification are not implied. Revocation gates new authority without stopping already active processes; explicit human stop remains separate.
