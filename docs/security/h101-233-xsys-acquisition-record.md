# H101-233 — `x/sys` acquisition verification record

Creed ruled the `golang.org/x/sys` v0.44.0 acquisition **discharged with limit**. This record separates what Ryan independently verified from Kevin's acquisition report and Creed's acceptance ruling.

## Independently verified by Ryan

On 2026-09-22, Ryan ran `go mod verify` in the repository. It printed:

```text
all modules verified
```

Ryan also ran:

```sh
cat ~/go/pkg/mod/cache/download/golang.org/x/sys/@v/v0.44.0.info
```

That module-cache download metadata identifies version `v0.44.0`, published at `2026-04-23T15:37:02Z`, with Git origin `https://go.googlesource.com/sys`, tag `refs/tags/v0.44.0`, and commit `fb1facd76f95fa87c151018200ea5e4892ff115d`.

## Reported by Kevin, not reproduced by Ryan

Kevin reported re-running `go get golang.org/x/sys@v0.44.0` with Go 1.27.1 on `darwin/arm64`, using the default `GOSUMDB=sum.golang.org` and default `GOPROXY`, with `GOFLAGS`, `GOPRIVATE`, and `GOINSECURE` unset. Kevin reported that `go.mod` and `go.sum` remained byte-for-byte identical to backups taken immediately before that re-run. Kevin also reported that `go mod tidy` produced no changes.

Ryan did not reproduce either mutating command for this record.

## Declared limit and acceptance

Kevin declared that no transcript exists for the original `go get`; Kevin did not claim to have observed that original command.

Creed accepted Kevin's reproduction because the requirement was intended to rule out a hand-typed, forgeable checksum line. For that specific property, Creed judged a byte-identical present-tense re-run against the real Go toolchain and checksum database stronger than a transcript that would establish only that a command ran once.

Creed's accepted claim is deliberately narrow: this evidence establishes a genuine, non-forged, reproducible dependency at the committed version. Creed expressly did not treat it as retroactive proof of which command ran when the dependency was originally committed.

## Future version bumps

The `go.mod` pin comment carries the standing obligation: every future version bump requires a fresh manual review of syscall behavior and must never be accepted as a semver-trusted automatic update. Ryan recorded that policy in commit `f637597` (H101-228), including its scope in [`scripts/check-no-network.sh`](../../scripts/check-no-network.sh) and the Creed H101-227 gate limit in the [threat model](threat-model.md).

Creed classified this cross-reference as a convenience for the next person performing a bump, not as additional security evidence for the current acquisition.
