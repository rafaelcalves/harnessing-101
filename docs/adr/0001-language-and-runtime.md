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

**INFERENCE.** Favor the standard library and dependencies that do not require a native C toolchain for the initial coordinator. Distribute explicit builds for approved operating-system/architecture pairs. Do not imply one binary runs everywhere. The initial platform deferral is resolved by the H101-50 amendment below. Exact toolchain selection remains in go.mod/build configuration; signing and release transport remain release-engineering decisions, not reasons to defer the platform scope.

**INFERENCE.** Run one controller per workspace with exclusive write ownership. The core is a library; a small host assembles its adapters and owns its lifetime. Phase 2 can prove replacement with two in-process front ends and shared contract tests. A future JavaScript GUI can use a versioned local standard-input/output bridge without adding network listeners or changing domain logic. Building that bridge is not required in this card.

## Consequences and checks

**INFERENCE.** The strongest objection is onboarding: TypeScript might match the actual contributor pool better, and a Go backend plus JavaScript GUI creates two toolchains. We accept this potential cost because no GUI framework or contributor survey is established, while distribution and process orchestration are explicit requirements. Shared UI-language types are not worth binding the core to an interface runtime.

**INFERENCE.** Runtime locality and build-time dependency acquisition are separate. The installed coordinator must not fetch updates, dependencies, agents, or telemetry. Contributors may need explicit downloads to prepare a build; document an offline build route if reproducible offline builds become a requirement. Agent executables are user-provided, with visible opt-in for configured provider access; starting a child does not establish that the child is network-confined.

**INFERENCE.** Before Phase 3 acceptance, test process-tree termination, output saturation, start/stop races, and controller crashes on every supported platform. Before Phase 4 acceptance, install the release artifact on clean machines without Go installed and observe runtime egress. These are proposed acceptance checks, not tests performed. Do not defer discovery of an impossible packaging constraint until release day: agree the target matrix during Phase 0 and attempt a packaging spike in an approved implementation phase.

## Revisit triggers

**UNKNOWN.** Interactive terminal requirements, contributor language experience, and distribution/signing constraints remain unsettled. Supported platform scope is now fixed by the H101-50 amendment below; its admission gates distinguish a support commitment from verified release compatibility. Revisit this ADR if a required agent or GUI integration imposes native dependencies that erase Go's distribution benefit; if measured onboarding friction dominates maintenance cost; or if an actual minimal Node package satisfies all target installations and materially simplifies an approved GUI. Record evidence and a superseding ADR rather than rewriting the rationale retrospectively.

## ADR process

**INFERENCE — proposed convention.** Store sequential, zero-padded records as `docs/adr/NNNN-short-title.md`; the architect assigns the next unused number. Use Proposed, Accepted, Rejected, and Superseded states. Require an ADR for changes to public contracts, persistence/recovery, runtime/language, process ownership, security/egress, or supported distribution platforms; routine implementation choices need none. Each record includes context, alternatives, decision, consequences, evidence, and revisit triggers. Phase approval accepts the initial decisions; later material changes follow the same owner review. Preserve accepted history and link both directions when a new ADR supersedes an old one.

## H101-50 amendment — platform scope and dependency policy

**INFERENCE — architecture ruling, 2026-09-19.** The architect sets the following support scope for Phase 2 and the initial release under H101-50. This replaces the platform UNKNOWN above, not the accepted Go decision. It records a delegated architecture ruling; it does not claim a new owner approval or that release verification has already passed.

| Target | Support decision | Filesystem / evidence scope |
| --- | --- | --- |
| macOS on Apple Silicon (`darwin/arm64`) | In scope; required milestone/release target | Native local storage; APFS is the reference filesystem. Record the exact macOS build used for each acceptance run. |
| Linux on x86-64 (`linux/amd64`) | In scope; required milestone/release target | Native local storage; Ubuntu with ext4 is the reference environment. Record the exact distribution, kernel and runner image used for each acceptance run. |
| Windows, any architecture | Out of scope; workspace operation unavailable | Current locking returns Unsupported. No degraded coordination mode or Windows release artifact is promised. |
| Intel macOS, Linux ARM64, BSDs, other OS/architecture pairs | Out of scope | Successful compilation or sharing the non-Windows implementation does not establish support. |

**INFERENCE — limits and rationale.** Two targets match the existing development/CI direction and keep native verification achievable. This is not a promise for every macOS version, every Linux distribution, or future OS releases. Supported runtime versions are the exact tested environments recorded in each milestone/release's evidence manifest; untested versions are unqualified, not silently covered by a “version X or newer” claim. A new reference OS version must pass the same native tests before replacing an existing baseline. Network shares, cloud-synchronised workspace directories, and unverified mounted/translated filesystems are excluded; WSL or emulation does not automatically count as either supported target. This closes the OS/architecture decision without inventing minimum OS compatibility from a build tag.

**CODE-PATH FACT.** go.mod declares Go 1.27.1 and has no external module requirements. CI currently selects `ubuntu-latest`; it does not name a fixed runner image or add a macOS job. `lock_unix.go` selects `!windows` and calls flock; that broad build condition is not our support matrix. `lock_windows.go` returns Unsupported. FileStore.Open calls acquireLock before returning a usable store, so Windows cannot currently open a working coordination workspace. It may create the directory/lock file before failing; Unsupported does not mean a zero-side-effect installation probe.

**OBSERVED STATE (2026-09-19).** The review machine reports macOS 27.0, build 26A428, arm64 through sw_vers/uname. This identifies the current machine, not a minimum supported version or a completed release qualification. **SECONDARY SOURCE.** [H101-20](../architecture/h101-20-lock-recovery.md) records local darwin/arm64 crash-release evidence; [quality's review](../quality/definition-of-done.md) reports Linux/macOS lock recovery. I did not rerun platform tests or inspect a Linux runner during this documentation card.

**INFERENCE — required evidence, not optional coverage.** Phase 2 exit must have the real adapter cycle and crash/reopen evidence on both in-scope targets. An Ubuntu-only CI result does not discharge macOS; a local macOS run is acceptable milestone evidence if its exact commit, system, commands and outputs are recorded. Before release, establish repeatable native coverage on both, capture runner versions rather than relying on an undocumented moving `latest`, and validate clean-machine installation without Go installed. Missing evidence blocks the relevant support claim; it does not reopen Windows scope or excuse the other target's failures.

### What Unsupported means

**INFERENCE.** For Windows or any platform missing mandatory exclusive locking, fail workspace opening with Unsupported and explain that coordination is unavailable on that platform. Do not fall back to no lock, stale-file deletion, read/write with weaker exclusivity, or apparent successful commands. Help/version output may still run. For other excluded targets there is no promise that a binary even builds; if a distributed binary encounters missing mandatory platform capability, it must fail explicitly. Optional Phase 3 operations returning Unsupported on a working macOS/Linux workspace are different: Phase 2 coordination remains supported, while those named operations are unavailable. Neither case means “same guarantee, best effort.”

### Dependencies and admission of another platform

**INFERENCE — dependency ruling.** Zero external dependencies is an intentional preference for a small maintenance/distribution surface, not a non-negotiable product constraint. Preserve the no-implicit-egress and reproducible-build requirements, not a numerical dependency count. A narrowly scoped, reviewed OS binding is preferable to hand-written low-level calls merely to preserve zero. Do not add a module without an explicit decision covering need, version pin/checksums, licence, transitive/build requirements, maintenance ownership and runtime behavior. No dependency is added or approved by this amendment.

**SECONDARY SOURCE.** The documented [golang.org/x/sys/windows package](https://pkg.go.dev/golang.org/x/sys/windows) provides Windows bindings, including LockFileEx. [Microsoft's LockFileEx contract](https://learn.microsoft.com/en-us/windows/win32/api/fileapi/nf-fileapi-lockfileex) says termination releases locks, but release timing depends on available resources. Therefore native crash-release tests must use bounded observation, not assume immediate availability from the API name alone.

**INFERENCE — Windows recommendation.** If Windows becomes a product priority, recommend a separately reviewed, pinned x/sys/windows dependency for the existing lock adapter rather than treating zero dependencies as a veto. Dependency approval alone is insufficient. Admission requires all of the following, recorded in a new amendment/ADR before adding Windows to public claims:

1. Name the OS versions, CPU architecture, local filesystem and maintainer; provide a native runner for recurring coverage. Cross-compilation alone is insufficient.
2. Demonstrate live-owner exclusion across processes, graceful release, forced-process-death release without manual deletion, and rejection of concurrent writers. Translate contention and unsupported conditions into the existing error contract.
3. Verify persistence replacement/durability and path behavior on that filesystem, then recover committed state and replay receipts through the actual UI after a crash. Passing only the lock test is insufficient.
4. Pass the shared adapter/authorization/provenance contract and controller-locality checks, plus clean-machine installation. Add Phase 3 process-tree/output/budget checks when those capabilities ship; do not infer them from workspace support.
5. Record the dependency decision and native evidence, then explicitly amend this matrix. Apply the same gates to Linux ARM64 or Intel macOS rather than assuming CPU portability proves runtime behavior.

**INFERENCE — consequence and revisit trigger.** Windows exclusion is now an explicit scope decision, not an unresolved blocker or an invitation to ship degraded locking. Revisit when an owner-backed need and a maintained native verification path justify expansion, or an in-scope target fails a required guarantee. No lock redesign, dependency installation, CI edit, or platform test was performed for this ruling.
