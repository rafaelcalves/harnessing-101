# ADR 0002 — Accept manual-agent exposure in Phases 1–2 with disclosure only

Status: **Accepted**. Decision date: **2026-09-19**. Decision-maker: **project owner**. Recorded by: Stanley, Architect, under H101-13.

## Context

**SECONDARY SOURCE — decision provenance.** The owner chose to accept the Phases 1–2 exposure with disclosure only, rather than move process control earlier. Michael (orchestrator) relayed that explicit decision in H101-13, message `god-2026-09-19-h101-adr2`, dated 2026-09-19. This records the owner's risk acceptance, not a decision made by the architect or security reviewer. It is separate from the owner's earlier product-based choice to keep process control in Phase 3.

**SECONDARY SOURCE — exposure.** [Threat model §5](../security/threat-model.md#5-manual-start-exposure-in-phases-1-2-h101-11) establishes the design gap: agents in Phases 1–2 are started manually, outside the controller. The product has no process-start gate or enforced approval event for them. It cannot observe, attribute, or restrict those processes. Hostile task/message content can influence an agent to disclose readable data through its own external connection; the controller making no external connections does not prevent this path. This is a design finding, not evidence of an observed attack.

**H101-228 evidence clarification (Creed, H101-227):** the shipped
`check-no-network.sh` guard rejects literal production imports of `net` and
`net/http`; it does not prove the controller cannot connect. Raw socket syscalls
and an `os/exec` network child remain invisible without those imports.
Syscall-capable dependencies require manual syscall-behavior review on admission
and every version bump, not pin-and-forget, semantic-version, or automatic-update
trust.

**SECONDARY SOURCE — intended scope.** The [product definition](../product/definition.md) describes a bounded workflow with two manually started agents and one human. [ADR 0001](0001-language-and-runtime.md) and the [project plan](../PROJECT-PLAN.md) place product-managed process control in Phase 3. Origin fields in writable files do not authenticate their claimed author.

## Options considered

**INFERENCE — retain sequencing with disclosure.** Preserve the planned headless core and command-line coordination milestones. Make the lack of process controls visible and retain unverified origin metadata. This preserves the small trial while leaving the exfiltration path uncontrolled by the product.

**INFERENCE — move process control earlier.** Introduce product-managed starts and their approval gate before the minimum workflow. This changes phase scope and adds engineering work. It would create a place to enforce start policy, but would not itself sanitize content or confine an agent's filesystem/network access. It is not a complete solution to the underlying risk.

## Decision

**SECONDARY SOURCE — accepted owner decision.** Accept the manual-agent exposure for **Phases 1–2 only**, with disclosure rather than technical enforcement. Keep process control in Phase 3. Retain the requirement that the controller itself makes no implicit external calls. Acceptance of this exposure does not waive the controller's filesystem, command-validation, or record-integrity obligations.

**SECONDARY SOURCE — required disclosure measures.** The accepted mitigation from the [security finding](../security/threat-model.md#5-manual-start-exposure-in-phases-1-2-h101-11) is:

- Persist origin metadata from Phase 1: claimed sender, entry mechanism, recorded timestamp, and explicitly unverified identity. Carry that context with content delivered to agents and displayed to humans.
- Make this statement unconditionally visible during Phase 2 use, not just at onboarding: “Harnessing 101 does not start, observe, or restrict any agent process in this phase. Nothing here confirms which program produced this content.”
- Do not attach checkmarks, “verified” badges, reassuring green indicators, or equivalent validation cues to manually sourced content. A recorded acknowledgement or human acceptance remains a workflow fact, not proof of safe content or verified authorship.

## Consequences for Phase 2's shippable state

**INFERENCE.** A Phase 2 delivery can satisfy the agreed scope while leaving manually started agents unobserved and unrestricted. It ships inspectable coordination records, origin disclosures, and human workflow decisions; it does **not** ship agent confinement, validated process identity, enforced provider approval, or a guarantee that workspace data stays on the machine. Those limitations must be part of the ordinary interface and walkthrough, not hidden in release notes. This decision is not itself authorization to ship or advance a phase; milestone approval still applies.

**INFERENCE.** Acceptance checks must verify persisted origin metadata, the unconditional disclosure, and absence of validation-looking cues. Observing no controller egress supports only the controller claim. A future test tracing product-started children to approved profiles cannot establish anything about independently started processes. Disclosure makes the limitation understandable; it does not reduce an agent's technical ability to exfiltrate data.

**UNKNOWN.** Whether users understand and tolerate this limitation in the bounded trial, and whether hostile content will exploit it, remain unmeasured. No probability or residual-risk rating is established by this ADR.

## Revisit triggers

**INFERENCE — acceptance boundary. Reopen this decision before any use, release claim, or workflow requires trusted agent identity, controlled data disclosure, or agent-process enforcement, or expands beyond the bounded human-supervised manual-agent trial.** Disclosure-only acceptance cannot justify that changed requirement. Also reopen if the disclosure cannot be delivered as specified, users demonstrably mistake records for validation, or a credible incident shows the assumed trial exposure is inadequate.

**INFERENCE — review action.** Security and product should present the changed condition to the owner before proceeding with the affected expansion. The owner must explicitly renew the risk acceptance, narrow the workflow, or require controls/resequencing; the team cannot silently extend this ADR. Review it at Phase 3 entry: product-managed start approval does not retroactively validate manual agents or eliminate content-driven exfiltration. Record any replacement in a superseding ADR, preserving this decision's history.

## Architect's assessment

**INFERENCE.** I do not disagree with this bounded owner decision. It keeps an explicit limitation visible while testing the coordination product. I would disagree with using it as permanent acceptance for broader deployment or treating Phase 3 process supervision as proof that the underlying exfiltration risk has been solved.
