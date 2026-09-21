# Development

This is a local-first Go project. Nothing here is expected to reach the
network at build or run time except downloading the Go toolchain itself and,
for contributors who want it, `golangci-lint`.

## Prerequisites

- Go 1.27.1 or later (`go version`). Install from https://go.dev/dl/ or via
  a package manager, e.g. `brew install go` on macOS.
- (Optional, for `make lint`) `golangci-lint` — https://golangci-lint.run,
  e.g. `brew install golangci-lint` on macOS.
- `git`.

No other dependency is required. `go.mod` currently declares zero external
modules.

## Install the commit-identity guard (do this first)

Commits in this repository must be authored as `rafael.ca.dev@gmail.com`,
never a work or default machine identity. Git hooks are not cloned with the
repository, so this is a manual, one-time step per clone:

```sh
git config core.hooksPath githooks
```

That points git at the `githooks/` directory checked into this repo, which
contains a `pre-commit` hook that rejects a commit made under any other
author email. If you also need to set your identity for this repo only:

```sh
git config user.email rafael.ca.dev@gmail.com
```

## Build

```sh
make build   # or: go build ./...
```

## Test

```sh
make test    # or: go test ./...
```

## Lint and vet

```sh
make vet     # go vet ./...
make lint    # golangci-lint run ./...
make fmt     # gofmt -l . — lists any unformatted file; empty output is clean
```

## Layout

```
cmd/harnessing/        CLI adapter entry point (Phase 2+; version stub only today)
internal/core/domain/  Core vocabulary: opaque IDs, record shapes
internal/core/ports/   The ten inbound/outbound contracts (docs/architecture/boundaries.md)
internal/adapters/     One package per outbound port implementation (empty skeletons today)
internal/host/         Wires adapters to the core (empty skeleton today)
```

The core (`internal/core/...`) never imports an adapter. See
`docs/architecture/boundaries.md` for what must stay outside it.

## Continuous integration

`.github/workflows/ci.yml` runs build, test, vet, and lint on every pull
request and on push to `main`. It is a required check before merge.

## Troubleshooting

If a workspace will not open, follow [Workspace lock troubleshooting](TROUBLESHOOTING.md). Do not delete a workspace `.lock` file based on its age.
