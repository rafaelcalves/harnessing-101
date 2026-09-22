# H101-154 — agentic CLI descriptor contract ruling

**Verdict: retain descriptor-driven extensibility; revise the field contract before product consumption.** The current shape is useful runner scaffolding, not yet a sufficient approved-profile tool contract. Split product-consumed tool facts from evidence-runner configuration now. This is a ruling, not a schema/runner implementation or compatibility certification. Kelly's H101-147/H101-153 gates stand unchanged; no fourth-tool demonstration is required.

## What the proposal actually establishes

Reviewed the schema, three descriptors, proposal, H101-152 report, runner source, Kelly's amendments and Claudio/Kelly memory. The dispatch identifies the bundle as `449e059`; no checkout, .git write, tool launch or test run was performed here.

- `shutil.which` and the version subprocess (`run-agentic-cli-cycle.py:114–128`) establish local discovery/version evidence, not approval or provider authentication.
- Every `invocation.managedStart.args` is the same **product** `start-run` command. None specifies the selected tool's own argument vector or how that executable receives participation context. The approved profile may eventually supply these, but today's descriptor cannot itself tell the second consumer how three tools differ.
- The runner checks required top-level keys (`81–85`), not the JSON Schema contract. `auth.detectionMode`, `contextMode` and `failureSignals` are declared but do not govern distinct behaviors in the inspected path. Thus a declared mode is not yet a demonstrated capability.
- Authentication/network classification is substring matching over combined product output (`154–162`); `requiredPatterns` actually means authentication-required hints, not positive authentication evidence. Words such as “credentials,” a provider hostname or “timed out” alone are ambiguous.
- Participation/completion are output substring checks (`165–175`). Empty arrays are schema-valid and make the corresponding check vacuous; incidental echoed words can also match. A successful asynchronous StartRun response cannot by itself prove the later coordination cycle.
- Schema lacks H101-153's explicit deferral variant. Optional `humanInterventions`/`friction` mix planned procedure with observed run evidence; their presence in a descriptor cannot certify actions happened.

The observed `product_surface_unavailable` remains an honest prerequisite failure. It is not invalidated by these findings, and it does not validate a future positive path. This ruling names contract changes; Claudio owns runner adaptation, not a redesign commissioned here.

## Canonical location and consumer split

Put the normative schemas at **`schemas/agentic-cli/tool-descriptor.schema.json`** and **`schemas/agentic-cli/evidence-scenario.schema.json`**, with a documented field contract and examples under `docs/architecture/`. This is a language-neutral executable-configuration contract, not a Go domain request: do not place JSON files in `internal/api` merely because the product reads them. Runtime loader/types belong to the execution-profile adapter; core receives a validated immutable execution specification and independently approved policy.

Keep one file per tool in the existing descriptor directory initially, containing a product `toolSpec` section and an optional `evidence` section referencing a common scenario. The two schemas define those sections; the top-level schema also supports deferral. Product must read/validate `toolSpec` without executing evidence configuration; runner consumes the same section plus its evidence reference. Do not maintain separate conflicting executable/invocation definitions. Moving directory layout again when packaging is an implementation choice, not a second contract.

Use **schemaVersion 2** for this changed meaning so old proposal files fail explicitly in the new consumer instead of being silently reinterpreted. Migrate the three proposals together. Unsupported major versions, unknown behavioral fields and unsupported mode kinds fail before executing even a probe. An optional namespaced metadata object may allow nonbehavioral annotations; no ignored typo in an invocation field.

## Required version-2 shape

| Section | Minimum contract |
| --- | --- |
| Identity | Stable machine `toolId`, display name, schemaVersion, immutable descriptor revision; toolId is data, never a dispatch branch. Record exact installed version separately from claimed supported/tested versions. |
| Availability/deferral | Explicit `deferred` discriminator. True requires reason and a reference to the recorded owner deferral; no executable invocation is required or attempted. False requires toolSpec. A descriptor author cannot grant the owner deferral merely by setting a boolean. |
| `toolSpec.executable` | Lookup name (or explicitly configured local path); consumer resolves once, records the resolved identity/version and binds it to approval. No installation, path search script or fallback download. Missing binary is a stable precondition failure. |
| `toolSpec.versionProbe` | Direct argument array, nonempty unique accepted exit codes, bounded execution/output; records observed output/version. Probe behavior is separately authorized local execution, not proof that arbitrary descriptor commands are safe. |
| `toolSpec.invocation` | Direct **tool** argument vector with declared context binding and I/O mode. No `harnessing start-run` wrapper, shell command string, hidden tool-name branch or embedded executable code. |
| `toolSpec.context` | Enumerated implemented transport: argument value(s), stdin, or a host-created context file. Define UTF-8 encoding, template/format version, required fields and file lifetime; reject missing required context. Task/workspace/agent/run IDs come from trusted host bindings. |
| `toolSpec.authentication` | Explicit non-authenticating detection strategy: runtime diagnostic hints, optional documented local status probe, or owner-attested/unknown status. No login command, credentials, browser flow or endpoint retry. |
| `toolSpec.diagnostics` | Typed hints for authentication-required, unsupported invocation and provider-connectivity failure, with declared source (exit status/structured diagnostic/stderr) and matching semantics. Preserve unknown/ambiguous outcomes. |
| `evidence` | Common scenario/version reference plus optional tool-specific observation hints; expected intervention checklist is labelled planned. No fields that can weaken the fixed scenario or authorize execution. |

Use structured argument elements (`literal` or a named `contextSlot`) rather than unrestricted formatting evaluation. Slots expand as data in one argument, never a shell expression; unknown slots fail. Required capability kinds must already exist generically in the loader. Adding another tool using supported kinds is a new descriptor file plus runner invocation; adding a genuinely new invocation capability is an explicit versioned feature, never a special-case branch keyed by tool name. Do not invent flags for the three tools: fill them from documented/tested invocation forms, or declare unsupported/deferred honestly.

`profile-context` alone is not an injection mechanism: it must resolve to a concrete binding above. Retain `unsupported` as a diagnostic/capability result, not a runnable context mode. Interactive-terminal requirements must be declared unsupported until that capability is approved and built; descriptors cannot enable terminal embedding by assertion.

Bound every command/probe by host-owned ceilings. A descriptor may state compatibility requirements or a recommended timeout, but cannot raise host limits. Only implemented/verified reporting and termination capabilities are advertised; a data-file claim does not enable trusted token measurement or process-tree containment.

## Detection without authentication

Presence, version, login state, invocation compatibility, provider reachability and participation are separate observations. A successful local version command proves none of the others. No matching failure pattern means **unknown**, not authenticated. An owner check is an attributed attestation, not a product-verified session. A local status probe is optional, must be documented not to authenticate/change credentials, and its exact invocation needs user-approved execution policy; an editable descriptor is not proof of side-effect freedom.

Prefer stable structured diagnostic codes when a tool exposes them. Runtime-output hints remain best-effort diagnostics, with source and matched rule recorded and sensitive output bounded/redacted under the existing artefact rules. A provider hostname or generic timeout alone cannot establish “network egress refused”; ambiguous evidence stays unknown/diagnostic-unconfirmed rather than forcing one of the five required demonstrated outcomes. Hints must never grant approval or override caller authority. No descriptor claims authentication success for a different executable than the one actually launched by the approved profile.

## Participation and evidence are not tool policy

The common Layer B scenario belongs to the quality contract, not each tool descriptor. Its success requires authoritative product records associated with the expected workspace/task/agents/run: participation and handoff/acknowledgement, blocker/human response, result/reject/rework/acceptance and retained history as required by that scenario. Tool-output hints can help locate evidence or explain failure; they cannot declare acceptance, manufacture human interventions or replace the records. An asynchronous receipt is start-intent evidence, not cycle completion.

Require nonempty scenario assertions; schema-valid empty pattern lists cannot mean success. Classification distinguishes observed Running without required participation, confirmed spawn failure, and uncertain start/response loss. A timeout alone proves neither child creation nor its absence. Follow ADR 0004 and GetRun/GetOperation before labelling those cases; do not retry new IDs or manually launch the tool to obtain evidence.

Runner output keeps observations separate from expected steps: actual commands/timestamps, product revision and whether independently observed or supplied, exact executable/version, descriptor digest, approved profile revision, context identifiers, bounded diagnostic tails and observed interventions. Record planned-but-unperformed interventions as such. Descriptor hash records what was used; it is not proof of compatibility or authority. The product/runner must reconcile the discovered executable with the profile's resolved executable to avoid reporting tool A while launching B.

## Policy boundary and approval binding

Descriptor facts explain how a tool can be invoked. Host/core still decides whether it may run: profile approval, principal/agent binding, worker grants, environment references, working-directory policy, permitted invocation mode, budgets, termination, retries, and human review. No auth pattern, success code, tool ID, declared capability or evidence setting can decide those policies.

Resolve descriptor + approved configuration into an immutable execution specification. Approval binds its canonical execution-affecting digest (descriptor identity/revision, executable identity, arguments/context mechanism, environment references, mode and effect-capability requirements). Changing that material requires new approval before dispatch. Evidence-only annotations need not invalidate execution approval; retain the whole-file digest separately for audit. Recheck the approved resolved specification at dispatch under H101-128, without putting OS resolution in core or trusting a mutable file after approval.

Strict structural validation and semantic validation are required in **both** consumers, with shared valid/invalid conformance cases; JSON parsing alone is insufficient. Semantic checks cover allowed slots/modes, contradictory deferral fields, empty success-code sets, missing context and unsupported capabilities. Validate before local probes or start. The schema itself is not a security sandbox for executable data.

## Disposition

Claudio revises the schema/descriptors and their documentation/runner interpretation; Kevin consumes the resulting product toolSpec through approved-profile resolution. Kelly owns acceptance against unchanged H101-147/H101-153: descriptor-per-tool or explicit owner deferral, no tool-name switches, Layer A participation fixture plus separate real Layer B runner evidence. No fourth-tool trial, provider authentication, runner replacement or gate amendment is requested.

Most important correction: **describe the actual installed-tool invocation once, keep the product start command in the consumer, and keep cycle success and permission policy out of editable tool facts.** Current three files remain proposals until revised and validated; only the documented local Claude discovery/version and unavailable product surface have evidence in the supplied report. No code/schema/runner files changed by this review.
