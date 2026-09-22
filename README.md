# Harnessing 101

Harnessing 101 is an early-stage, local-first tool for coordinating AI agent work through inspectable files. It is not yet a released or complete agent runtime.

## Current disclosure

**Agents you configure may send data they can access to external services. Harnessing 101 itself does not confine those agents or guarantee that your data stays on this machine; it runs locally and makes no external connections of its own.**

**The product itself does not yet start, observe, or restrict agents; you start agents by hand in the current phases, and content received from another agent is unverified.**

The continuous-integration check behind the product-side network claim is an
import-graph guard: it rejects production dependencies on `net` and `net/http`.
As Creed's H101-227 security review notes, it cannot detect raw socket syscalls
or a child network tool launched through `os/exec`. A syscall-capable dependency
therefore requires manual syscall-behavior review on admission and on every
version bump; a pinned version is not approved for automatic updates.

To use the product from a clean clone, follow the runnable
[Getting started guide](docs/GETTING-STARTED.md). It covers the complete first
task, message, acknowledgement, result, and human-review cycle.

If a workspace will not open, follow [Workspace lock troubleshooting](docs/TROUBLESHOOTING.md) before changing or deleting any file.

## Understanding command receipts

A successful command prints a receipt such as:

```text
harnessing create: OK (request req-123, workspace revision 7)
```

Workspace revision identifies a recorded version of your workspace; it advances when commands or background delivery record changes, so it is not a count of your commands.

See [Receipt revision troubleshooting](docs/TROUBLESHOOTING.md#a-receipt-shows-an-older-workspace-revision) if a receipt's revision is lower than the workspace's current revision.

## Build and test

Install Go 1.27.1 or later, then run from the repository root:

```sh
make build
make test
make vet
make lint
make fmt
```

See [Development](docs/DEVELOPMENT.md) for prerequisites and repository setup, and [Contributing](CONTRIBUTING.md) before preparing a change.

## Project documents

The project plan and architecture decisions are the source of truth for current scope and constraints:

- [Project plan](docs/PROJECT-PLAN.md)
- [Product definition](docs/product/definition.md)
- [Language and runtime decision](docs/adr/0001-language-and-runtime.md)
- [Manual agent exposure decision](docs/adr/0002-manual-agent-exposure-in-phases-1-2.md)
- [Architecture boundaries](docs/architecture/boundaries.md)
- [Threat model](docs/security/threat-model.md)
- [Troubleshooting](docs/TROUBLESHOOTING.md)
- [Security policy](SECURITY.md)

These documents may change as phases are reviewed. Follow them rather than relying on summaries copied elsewhere.
