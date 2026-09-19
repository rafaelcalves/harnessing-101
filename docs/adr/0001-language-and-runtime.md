# ADR 0001 — Go for the local orchestration core and first CLI

Status: **Accepted**, approved by the project owner on 2026-09-19. Date: 2026-09-19. Owner: Stanley, Architect. Recorded by Michael (orchestrator); the decision and its reasoning are Stanley's.

Evidence convention: **INFERENCE** marks recommendations and design requirements, not implemented behavior. **SECONDARY SOURCE** marks documentation-backed facts. **UNKNOWN** marks untested assumptions. No product code exists in this decision.

## Context

**SECONDARY SOURCE — project requirements.** [PROJECT-PLAN.md](../PROJECT-PLAN.md) requires local execution without implicit network calls, a replaceable command-line interface (CLI), contributor readiness, child-process supervision, and end-user distribution. The project plan remains draft. This architecture decision record (ADR) proposes a stack; it does not authorize the next phase.

**INFERENCE.** The binding product is a local coordinator with file persistence and process lifecycle responsibilities. A graphical user interface (GUI) is a future adapter, not the present deployment unit. Choosing a language chiefly to share code with a hypothetical GUI puts an optional convenience ahead of two immediate obligations: installing and supervising the coordinator.

## Options considered

**INFERENCE — comparison.** These are architectural judgments, not benchmark results or measured contributor preferences.

| Criterion | TypeScript / Node | Go |
| --- | --- | --- |
| Replaceable interface | Good with explicit ports; sharing types with a JavaScript GUI is convenient, but does not enforce separation. | Good with explicit ports; a non-Go GUI needs a thin local protocol adapter. This is an explicit cost. |
| Fully local runtime | Achievable; reject implicit telemetry, update checks, downloads, and network dependencies. | Achievable under the same rules. A compiled executable is not a network sandbox. |
| Contributor productivity | Advantage for contributors already familiar with TypeScript; runtime, transpilation, and dependency conventions must be documented. | A smaller initial toolchain and standard formatting/testing conventions; contributors unfamiliar with Go still face a learning cost. |
| Child processes | Capable asynchronous process APIs; process trees, terminal sessions, cancellation, and recovery remain platform concerns. | Capable process APIs and explicit concurrency; the same platform concerns remain. Go does not solve supervision automatically. |
| User distribution | Installing Node is an extra user prerequisite unless the runtime is bundled. Bundling is feasible, but must be tested against actual dependencies and targets. | Favors distributing a per-platform executable without a separately installed language runtime. Native dependencies and platform signing can complicate this. |

**SECONDARY SOURCE.** Node documents both [child-process control](https://nodejs.org/api/child_process.html) and [single-executable applications](https://nodejs.org/api/single-executable-applications.html). Claiming Node cannot ship a single executable would be false. Its packaging documentation includes module, native-addon, signing, and cross-platform caveats. Those are release-engineering work, not disqualifications.

**SECONDARY SOURCE.** Go's [os/exec documentation](https://pkg.go.dev/os/exec) supplies process start, wait, cancellation, and pipe primitives; it does not invoke a shell by default. Go's [build environment documentation](https://go.dev/doc/install/source#environment) identifies operating-system and architecture targets. These primitives support the proposal; they are not evidence that this application already works on every target.

**INFERENCE — Rust alternative.** Rust is a credible compiled option, especially if tight resource bounds or an existing Rust contributor base becomes binding. Neither is established here. Its additional implementation and learning demands are not justified by a demonstrated need in this file-and-process coordinator. This is a project-fit judgment, not a general ranking of languages.

## Decision

**INFERENCE — verdict. Choose Go; overrule the TypeScript/Node recommendation for the core and initial CLI.** Preserve the proposed ports-and-adapters architecture. Keep UI commands, rendering, transport, filesystem operations, and operating-system process handles outside the core. The ten contracts in [boundaries.md](../architecture/boundaries.md) govern both the initial CLI and later adapters.

**INFERENCE.** Favor the standard library and dependencies that do not require a native C toolchain for the initial coordinator. Distribute explicit builds for approved operating-system/architecture pairs. Do not imply one binary runs everywhere. Defer exact toolchain version, supported-platform matrix, signing, and release transport to documented decisions before release work; no version is pinned by this ADR.

**INFERENCE.** Run one controller per workspace with exclusive write ownership. The core is a library; a small host assembles its adapters and owns its lifetime. Phase 2 can prove replacement with two in-process front ends and shared contract tests. A future JavaScript GUI can use a versioned local standard-input/output bridge without adding network listeners or changing domain logic. Building that bridge is not required in this card.

## Consequences and checks

**INFERENCE.** The strongest objection is onboarding: TypeScript might match the actual contributor pool better, and a Go backend plus JavaScript GUI creates two toolchains. We accept this potential cost because no GUI framework or contributor survey is established, while distribution and process orchestration are explicit requirements. Shared UI-language types are not worth binding the core to an interface runtime.

**INFERENCE.** Runtime locality and build-time dependency acquisition are separate. The installed coordinator must not fetch updates, dependencies, agents, or telemetry. Contributors may need explicit downloads to prepare a build; document an offline build route if reproducible offline builds become a requirement. Agent executables are user-provided, with visible opt-in for configured provider access; starting a child does not establish that the child is network-confined.

**INFERENCE.** Before Phase 3 acceptance, test process-tree termination, output saturation, start/stop races, and controller crashes on every supported platform. Before Phase 4 acceptance, install the release artifact on clean machines without Go installed and observe runtime egress. These are proposed acceptance checks, not tests performed. Do not defer discovery of an impossible packaging constraint until release day: agree the target matrix during Phase 0 and attempt a packaging spike in an approved implementation phase.

## Revisit triggers

**UNKNOWN.** Supported platforms, interactive terminal requirements, contributor language experience, and distribution/signing constraints are unsettled. Revisit this ADR if a required agent or GUI integration imposes native dependencies that erase Go's distribution benefit; if measured onboarding friction dominates maintenance cost; or if an actual minimal Node package satisfies all target installations and materially simplifies an approved GUI. Record evidence and a superseding ADR rather than rewriting the rationale retrospectively.

## ADR process

**INFERENCE — proposed convention.** Store sequential, zero-padded records as `docs/adr/NNNN-short-title.md`; the architect assigns the next unused number. Use Proposed, Accepted, Rejected, and Superseded states. Require an ADR for changes to public contracts, persistence/recovery, runtime/language, process ownership, security/egress, or supported distribution platforms; routine implementation choices need none. Each record includes context, alternatives, decision, consequences, evidence, and revisit triggers. Phase approval accepts the initial decisions; later material changes follow the same owner review. Preserve accepted history and link both directions when a new ADR supersedes an old one.
