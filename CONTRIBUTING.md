# Contributing to Harnessing 101

Harnessing 101 is a local-first tool for coordinating agent work through inspectable files. Contributions should preserve its local-only runtime, replaceable interfaces, durable records, and factual language.

## Before you start

Use an issue to establish the problem, expected behavior, and acceptance evidence before writing a substantial change. Security vulnerabilities and conduct reports must use the private routes in `SECURITY.md` and `CODE_OF_CONDUCT.md`, not public issues.

The project is written in Go 1.27.1, as declared in `go.mod`; do not assume a different local version is supported. It currently has no external Go module dependencies. A new dependency needs a concrete justification and review of its licensing, maintenance, build, and runtime behavior.

## Configure this checkout

Commits in this repository require the project's author identity. Configure it locally, never globally:

```sh
git config --local user.email "rafael.ca.dev@gmail.com"
git config core.hooksPath githooks
```

Confirm the effective settings:

```sh
git config --local --get user.email
git config --get core.hooksPath
```

The repository hook rejects commits whose effective author email is not the required project email. Do not bypass it. If the hook fails unexpectedly, report the exact output instead of committing with verification disabled.

## Build and test

Run commands from the repository root:

```sh
go build ./...
go test ./...
go vet ./...
golangci-lint run ./...
test -z "$(gofmt -l .)"
```

Continuous integration runs build, test, vet, and golangci-lint. Run `gofmt -w` on changed Go files before the final check. Add or update tests for behavior changes. A passing test suite does not replace a focused test that demonstrates the new behavior or regression.

The installed product must not make implicit network calls. New dependencies, update checks, telemetry, network listeners, or agent-provider integrations require explicit review even if tests pass. Build-time dependency acquisition and user-configured agent egress must remain distinguishable from product runtime behavior.

`scripts/check-no-network.sh` checks the production import graph for the literal
paths `net` and `net/http`. It does not inspect behavior and cannot detect raw
socket syscalls or an `os/exec` child such as `curl` when those imports are
absent. Per Creed's H101-227 security review, admitting a syscall-capable
dependency and every later version bump require the same manual review of its
syscall behavior. Do not rely on semantic versioning or an automated dependency
update to preserve that property: in Creed's words, “pin-and-forget silently
erodes exactly this review.”

## Architecture decisions

Follow the architecture decision record (ADR) process defined in [`docs/adr/0001-language-and-runtime.md`](docs/adr/0001-language-and-runtime.md). Do not restate or fork that process in an issue or pull request. Link the proposed or accepted ADR when a change affects a public contract, persistence or recovery, runtime or language, process ownership, security or egress, or supported distribution platforms.

Accepted ADRs are history. Supersede them with a new record; do not rewrite an accepted decision to make later work appear inevitable.

## Make a focused change

1. Link the change to one issue with clear acceptance criteria.
2. Keep domain behavior independent of command-line rendering, filesystem details, and process handles.
3. Preserve the local-only guarantee and make any user-configured egress visible and opt-in.
4. Add tests at the narrowest useful boundary.
5. Update user or contributor documentation when commands, files, or contracts change.
6. Run the build, test, vet, formatting, and author-identity checks before opening a pull request.

Use commit messages that state the outcome in the imperative mood, for example `Validate message recipient IDs`. Keep refactors separate from behavior changes when that makes review easier.

## What makes a good first issue

A good first issue is independently verifiable and does not require discovering product policy while implementing it. It should name the affected package or document, describe current and expected behavior, provide acceptance checks, and identify relevant tests or fixtures. It should avoid changing public contracts, persistence formats, process supervision, security boundaries, or the local-only guarantee unless an accepted ADR already defines the change.

Suitable examples include improving one validation error, adding a focused test for an existing rule, clarifying one documented command, or removing a bounded duplication without changing behavior. “Explore the architecture,” “improve reliability,” and other open-ended investigations are not good first issues.

## Pull requests

Complete the pull request template. Explain the user-visible outcome, test evidence, local-only impact, architecture impact, and recovery or compatibility risk. Keep generated output, credentials, personal workspace data, and agent transcripts out of commits.

## Licence terms for contributions

The project is licensed under Apache License 2.0. Unless you explicitly mark a submission as “Not a Contribution,” an intentionally submitted contribution is provided under Apache-2.0 without additional terms. This includes the licence's copyright and patent grants.

Preserve applicable copyright, patent, trademark, and attribution notices. Modified files distributed as part of a derivative work must carry prominent notices stating that they were changed. Identify modified files clearly in the pull request and add file-level change notices where Apache-2.0 requires them.
