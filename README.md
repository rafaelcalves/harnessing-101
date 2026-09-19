# Harnessing 101

Harnessing 101 is an early-stage, local-first tool for coordinating AI agent work through inspectable files. It is not yet a released or complete agent runtime.

## Current disclosure

**Harnessing 101 runs completely locally and makes no external connections itself; agents you configure may send content they can access to external services, and Harnessing 101 does not confine those agents or guarantee that your data stays on this machine.**

**The product itself does not yet start, observe, or restrict agents; you start agents by hand in the current phases, and content received from another agent is unverified.**

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
- [Security policy](SECURITY.md)

These documents may change as phases are reviewed. Follow them rather than relying on summaries copied elsewhere.
