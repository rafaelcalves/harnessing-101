# H101-119 — preserving caller and sender as separate claims

2026-09-21. Architecture ruling on the shipped Phase 2 contract. Source inspection only; no implementation, test execution, or Phase 3 authorization. Read Creed's Phase 2 exit review and memory, H101-109, the current boundaries, domain/engine code, contract assertions, and continuous integration (CI) configuration. Line references identify the working tree inspected for this ruling.

## Verdict

**The existing fixture is useful but insufficient to protect both stored identities: add explicit behavioral assertions to the existing contract suite.** This is a small regression-coverage correction for the shipped contract, not a reason to introduce new identity types or an assignment-analysis gate. Kevin implements the checks; Kelly assesses their evidence. This ruling does not reopen Phase 2 acceptance or assert a live exploit/new production defect.

The invariant is source fidelity, not inequality: `Message.SenderAgentID` preserves the submitted routing claim, while `Message.Provenance.ClaimedAgentID` preserves the host-bound caller claim. Equal values remain legal. Registry membership establishes neither authorship nor verified identity; both claims remain unverified at this maturity. In particular, “host-bound” does not mean cryptographically authenticated.

## What is actually protected today

- `boundaries.md:38`, H101-109's membership/authorship ruling (`h101-109-agent-reference-validation.md:19`), and `records.go:109–120,185–204` establish the separate meanings. The current reviewer amendment is at boundaries line 34; it is not the routing-claim paragraph.
- `engine.go:600–604,619–630` checks registry membership, copies sender from the request, and constructs provenance from caller scope. The implementation inspected preserves the distinction.
- `internal/adaptercontract/ui04_test.go:153–195` submits a registered third-party sender through both adapters and requires success. That would catch a newly imposed sender-equals-caller rejection.
- **The final assertion is narrower than the input fixture suggests.** `assert_test.go:119–144` checks workspace revision, agent count, and task state; it does not assert either message identity. UI-05 checks labels and the unverified marker (`ui05_test.go:24–70`), not both identity values. The core provenance assertion (`internal/core/task/message_test.go:35–65`) uses the same sender and caller, so it cannot distinguish their sources. These observations do not support a claim that the cited fixture catches silent coercion in both directions.

Creed is right that acceptance of differing claims is a genuine compatibility property and not impersonation resistance. Narrow his stronger “durably records / no coercion” description to what explicit value assertions demonstrate; do not infer that guarantee from successful submission alone. No deliberately modified implementation was run in this review.

## Proportionate guard for Kevin

Extend the existing shared contract scenario, with independent expected literals and a clear failure name, rather than creating a new checker framework:

1. Use three distinct registered identities: bound caller A, submitted sender B, recipient C. Require send success through each supported adapter.
2. Locate the message by its explicit ID and assert sender B, provenance caller A, recipient C, and `IdentityUnverified`. Check canonical queried state and each adapter's exposed message values; CLI checks must associate values with their respective fields, not merely search for both strings somewhere in output.
3. Close and reopen the workspace, then assert the same values through the other adapter's read surface. Reuse existing lifecycle/crossover helpers. Preserve a case where caller equals sender so the test does not invent a mandatory-inequality rule.
4. Demonstrate sensitivity during implementation review: rejecting unequal identities, overwriting sender with caller, and overwriting provenance with sender must each fail the focused check. These are temporary local mutations, never shipped; report the evidence. No permanent mutation-testing dependency is required.

Normal CI already runs `go test ./...` on Linux and macOS (`.github/workflows/ci.yml:29–30,77–78`), so the guard belongs there. Its scope is regression detection on exercised paths, not proof that no downstream consumer ever trusts sender data or that hostile contributors cannot edit tests.

## Why not a static or type-level mechanism now

The import gate explicitly checks package containment, not resolved symbols (`scripts/check-import-allowlist.sh:4–6,75–100`). That structural property has a bounded dependency graph. A rule forbidding assignments between two fields misses copying through locals, conversions, constructors, and serializers; it also misses an equality-based rejection that assigns neither field. It cannot honestly prove this behavioral invariant.

Distinct named types can make accidental direct assignment harder, but explicit conversions still compile and equality enforcement can still be introduced. Their broader API/serialization impact is not justified solely by this regression gap. A runtime inequality guard would itself violate the contract whenever the caller legitimately sends under its own ID.

Revisit the mechanism before introducing authenticated principals, any authorization decision based on message sender, a new ingestion path that establishes caller scope, or a persisted identity/schema migration. At that point require an explicit identity/authority design and path-specific tests; consider private constructors and separate principal/claim types where they enforce that design. Repeated coercion regressions despite the behavioral checks are also a trigger. A new presentation adapter must satisfy this same contract immediately. None of these triggers grants permission to merge existing stored claims silently.
