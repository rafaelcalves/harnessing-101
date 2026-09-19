# H101-25 — Keep transactional mutation private; enforce access in Phase 1

Status: **Architecture ruling for review**, 2026-09-19. No implementation or port-signature change. Sources below are the local working-tree files inspected for this review; the dispatch identifies baseline `9ca3cb8`, but I did not independently verify the checkout revision.

## Ruling

**INFERENCE. The callback is not the security hole; exposing a privileged persistence capability to command callers is. Keep the current StateStore transaction shape, but move the access-boundary portion of H101-19 into Phase 1, before phase acceptance.** Phase 2 must verify both interfaces preserve that boundary. This is neither “rewrite Commit now” nor “trust all host code and defer the problem.” Ten ports remain unchanged; no new ADR is needed for a signature change because none is proposed.

## What the code establishes

**CODE-PATH FACT.** `internal/core/ports/outbound.go` declares a privileged persistence port, not an authorization service: CommitRequest accepts a mutation over a snapshot and explicitly assigns task-rule validation to that mutation. `internal/core/task/engine.go:568` constructs the request in its private `commit` helper. The existing core already authors its changes; the adapter is not inventing them.

**CODE-PATH FACT.** `internal/adapters/statestore/file.go:123–166` checks replay, clones state, invokes `req.Mutate`, assigns the workspace revision, and persists state plus receipt. It has no acceptance/recipient-policy validation. Engine acceptance requires human-review authority at `engine.go:237`; acknowledgement checks the recipient before assigning acknowledgement at `:390–396`. A custom store mutation can bypass either command. Separately, CallerScope exposes an `IsHumanReviewer` boolean at `:28–30`: arbitrary privileged host code can also lie about authority without touching Commit.

**SECONDARY SOURCE.** [Quality's H101-17 and H101-18 reviews](../quality/definition-of-done.md) correctly locate the bypass outside the engine and accept the callback's atomicity rationale. What I change is the timing of prevention, not those findings about the accepted slices.

## Why a narrower payload is insufficient

**INFERENCE.** A host able to submit a replacement snapshot, changeset, or event can author the same false Done/acknowledgement state. An opaque core-created transaction can reduce accidental construction, but cannot make a malicious persistence adapter or same-process host trustworthy: either can write storage directly. Putting all domain authorization in FileStore would duplicate core rules across adapters and reverse the intended boundary.

**INFERENCE.** Retain the callback to preserve validation and multi-record mutation under one serialized transaction. This is an infrastructure capability usable by reviewed core code only. It is not a public command extension hook. Replacing it later for transactional efficiency or a different backend is a separate design question; do not claim that change authenticates writes.

## Checkable Phase 1 obligations

**INFERENCE.** Add a minimal headless composition boundary now, without building the CLI. It creates the store and engines privately and returns only command/query capabilities plus lifecycle shutdown. It must not return a FileStore, StateStore, CommitRequest builder, mutable store-backed state, or a concrete object whose extra methods expose persistence. A narrower interface over an exposed concrete store is insufficient if callers can recover that store. Query results must be detached values.

**INFERENCE.** Enforce an explicit production dependency/call allowlist in continuous integration: only approved core transaction helpers may construct mutation requests or invoke Commit; storage implements it, and the small composition package may construct/inject the store but not author domain mutations. UI, ingress, and unrelated host packages cannot import concrete storage or use outbound persistence APIs. Check resolved symbols and method-value references, not just the text `.Commit(`. Keep low-level storage contract tests as an explicit test-only exception; they legitimately exercise arbitrary mutations. Fail the check on a deliberately forbidden fixture so the rule is demonstrated, not merely documented.

**INFERENCE.** Add headless assembly tests through the returned capabilities: direct Done transitions and agent acceptance fail; wrong-recipient acknowledgement fails; persisted state remains unchanged; legitimate acceptance/acknowledgement survives reopening. Caller authority must be bound by the reviewed host policy boundary, never decoded from agent payloads. Test that agent ingress cannot select human-review authority. These checks constrain shipped code paths; they are not a sandbox against arbitrary Go code, hostile file writers, or forged same-user identity. Authentication strength remains unverified unless separately established.

## Cost, Phase 2, and trust limits

**INFERENCE.** Existing engine command bodies, transaction atomicity, persistence format, and file-store contract tests need no rewrite. Work is a small composition/facade surface, capability/dependency checks, and assembly tests; tests that currently build engines directly may remain unit tests, but cannot substitute for that assembly proof. The incoming registry slice uses the same boundary rather than requiring a new storage design. Exact effort is unestimated; no implementation was attempted.

**INFERENCE.** Phase 2 adds CLI and second-adapter integration tests against the same restricted surface. Keep H101-22's mailbox ordering obligation untouched. The remaining trusted components are explicitly the core, composition/identity-policy code, and persistence adapter. Their review and enforced dependency boundaries make trust inspectable, not absolute.

**SECONDARY SOURCE / INFERENCE.** [ADR 0002](../adr/0002-manual-agent-exposure-in-phases-1-2.md) accepts exposure from manually started agents; it expressly does not waive command-validation or record-integrity obligations. It is not permission for shipped host routes to bypass the core. Revisit this ruling if hostile in-process plugins or protection against same-user storage tampering become requirements: those need a stronger isolation/authentication design, not a different Commit payload.
