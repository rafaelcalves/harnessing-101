# H101-101: proving windows refuses cleanly, without a windows runner

Kelly's exit check: on windows, a workspace operation answers
`Unsupported`, while `version` and `help` still work. This is a Phase 2
exit check, not windows support — the point is that an unsupported
platform fails in a stated way rather than by accident.

## What exists and where

- `internal/adapters/statestore/lock_windows.go` — the windows build of
  `acquireLock`/`releaseLock`, both returning `domain.ErrUnsupported`.
  Pre-existing; this card did not change it.
- `internal/adapters/statestore/lock_windows_test.go` (new,
  `//go:build windows`) — unit proof that those two functions return
  `ErrUnsupported`, not a panic or a different code.
- `internal/assembly/windows_unsupported_test.go` (new,
  `//go:build windows`) — product-path proof: `assembly.WithSession`,
  the only place that opens a workspace, returns a non-zero exit code
  and stderr naming `Unsupported` on windows, and the caller's function
  body never runs. `version`/`help` are not covered here because they
  never call `WithSession` at all (see `cmd/harnessing/run.go`) — there
  is nothing platform-specific to prove about a code path windows
  cannot reach.

## Why these are build-tag-guarded tests, not a windows CI job

Both tests only compile into the `windows` build of their package.
Three ways were considered:

1. **A real windows CI runner.** This is the only way to actually
   *execute* these tests and observe a pass/fail from a genuine windows
   process. It costs money (a paid runner class) and is a decision for
   a human, not something to add unilaterally to an existing account's
   CI bill. Not done here; flagged instead.
2. **Build-tag-guarded tests plus a compile-only check in today's
   (ubuntu-only) CI.** Added: `GOOS=windows GOARCH=amd64 go build ./...`
   builds the entire product for windows, and `go vet` under the same
   `GOOS`/`GOARCH` type-checks the two new windows-only test files.
   Both ran clean locally as of this card. This proves the code is
   *shaped* to do the right thing — it type-checks, the branch exists,
   the error code is spelled correctly — but it does not prove the
   `windows` build actually *behaves* that way at runtime, because
   `go vet` does not execute test bodies and this repository's CI has
   no way to run a windows binary.
3. **Faking a runtime pass locally** (e.g. asserting the same logic
   under a non-windows build, or mocking `GOOS`) was rejected: the
   whole point of the exit check is that the windows-tagged code path
   itself behaves correctly, and nothing on this machine can execute
   that path.

Given the choice actually offered — a compile-time/build-tag check
that "genuinely proves the refusal" versus paying for a windows
runner — the honest position is that build-tag tests plus the new
compile-only CI steps prove *shape* (it will not panic on typos, it
references the right error code, it type-checks against the current
`domain`/`api` packages), while genuinely proving *behavior* still
requires either a windows runner or a human running the built
`.exe` on a windows machine at least once. That is out of this card's
scope and is reported to `god` as a decision needing the human, per
the card's own instruction, rather than claimed as done here.
