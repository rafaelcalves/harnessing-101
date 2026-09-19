# H101-20 — Workspace lock recovery

Status: **Implemented**, 2026-09-19. Author: Kevin. Evidence convention per
docs/PROJECT-PLAN.md: CODE-PATH FACT, OBSERVED STATE (dated), SECONDARY SOURCE,
INFERENCE, UNKNOWN.

## The gap, restated

`FileStore` (`internal/adapters/statestore/file.go`) refuses a second `Open`
against a locked workspace root with `Busy`. Until this card, the lock was
just a file created with `O_EXCL`: its *existence* was the lock. A host
killed without calling `Close` — a crash, a `kill -9`, a CLI user hitting
Ctrl-C hard enough — left that file behind with nothing to remove it, so
every subsequent `Open` failed `Busy` forever. `boundaries.md` forbids
recovering a lock by elapsed wall time ("do not steal a lock based on
elapsed wall time"), so a timeout was never an option.

## Decision: recovery exists, via flock(2)

**CODE-PATH FACT.** `acquireLock`/`releaseLock` (`internal/adapters/
statestore/lock_unix.go`) now take an exclusive, non-blocking OS advisory
lock — `syscall.Flock(fd, LOCK_EX|LOCK_NB)` — on the same lock file, instead
of relying on the file's existence. `Open` still creates the file
(`O_CREATE`, no longer `O_EXCL`) but ownership is decided entirely by whether
`acquireLock` succeeds.

**Why this resolves the gap rather than working around it:** an `flock` is
tracked by the kernel against the *open file description*, not against any
data this adapter writes. The kernel releases it the moment every file
descriptor referring to it is closed — which happens automatically when a
process exits, for *any* reason, including `SIGKILL` and an unclean crash.
There is no PID to compare, so there is no PID-reuse failure mode (the
classic hazard with PID-file-based recovery: a dead owner's PID gets
reassigned to an unrelated live process, and a naive "does this PID exist"
check reports a false live owner). There is no timeout, so `boundaries.md`'s
rule is not bent, only satisfied by a mechanism that does not need one: "is
the owner still alive" is a question the kernel already answers correctly,
continuously, for free.

**OBSERVED STATE (2026-09-19, darwin/arm64, local machine).** Verified
directly before relying on it, not assumed from documentation:

1. `syscall.Flock`/`LOCK_EX`/`LOCK_NB` exist and behave as documented on
   `darwin/arm64` — confirmed by compiling and running a throwaway program.
2. Two file descriptors on the same lock file *within one process* correctly
   conflict (second `LOCK_EX|LOCK_NB` returns `EWOULDBLOCK`) — this is what
   `TestOpen_SecondOpenIsBusy` actually exercises, and it still passes.
3. Two *separate processes*: the second's `flock` attempt fails while the
   first is alive, and succeeds immediately after the first is killed with
   `SIGKILL` (no `Close`, simulating a crash) — confirmed with a holder
   process and a prober process, not inferred from platform documentation.
4. `GOOS=windows go build ./...` fails on `syscall.Flock`/`LOCK_EX`/
   `LOCK_NB` being undefined — confirmed, not assumed.

`internal/adapters/statestore/lock_recovery_test.go`'s
`TestOpen_RecoversAfterOwnerCrash` automates point 3 using the standard
Go re-exec-self-as-helper pattern (the same technique `os/exec`'s own tests
use): it spawns the test binary as a subprocess that opens and holds the
lock, confirms a second `Open` fails `Busy` while that subprocess is alive,
`SIGKILL`s it, and confirms the next `Open` succeeds — polling briefly only
to let the kernel finish tearing down the killed process, never waiting out
a timeout, because there is none to wait out.

## Platform scope, and the honest limit

**INFERENCE.** `lock_unix.go` carries `//go:build !windows`; a companion
`lock_windows.go` returns `Unsupported` outright rather than compiling a
best-effort or untested substitute. Windows has an equivalent kernel
primitive (`LockFileEx`), but it is not in Go's standard `syscall` package
the way `flock` is on Unix, using it would need `golang.org/x/sys/windows`
(a dependency this module does not currently have and this card should not
add unilaterally), and — most importantly — nobody has run it here to
verify its crash-release behaviour. ADR 0001 already lists the supported
platform matrix as unsettled; this card does not settle it. An honest
`Unsupported` is the right answer under the project's own evidence
convention: **UNKNOWN is acceptable; a confident, unverified guess is not.**
Revisit this file, not `boundaries.md`, if Windows support is ever
committed to.

**Scope note carried over from the fsync durability fix (H101-12 follow-
up):** this depends on the workspace root being a local filesystem, which
`threat-model.md` already declares — "unverified network filesystems are
unsupported rather than assumed equivalent to local disks." `flock` over
NFS is a known trouble spot (unreliable without a correctly configured lock
daemon on both client and server); this card does not add detection for
that case, because the project already does not claim to support it.

## If the mechanism is ever unavailable: the manual fallback

Even with recovery working on the platforms above, a fallback still belongs
on record, because the Windows path returns `Unsupported`, not a working
lock, and because a future reader deserves the reasoning rather than a bare
"just delete the file":

- The lock file is `<workspace-root>/.lock`. Its content (a `pid=<n>` line)
  is a **diagnostic hint written for a human, never read back by the
  adapter for any decision** — do not trust it to identify the process by
  itself; check the actual process table (`ps`, Task Manager) for that PID
  first, independent of the file's content.
- Deleting `.lock` while its process is genuinely dead is safe: the next
  `Open` recreates it. Deleting it while the owner is **still alive** does
  **not** stop that owner — it still holds its kernel-tracked lock on its
  own open file descriptor — but it does let a *second* process's `Open`
  proceed, because on the platforms where recovery works, deletion is
  irrelevant to the kernel-level lock anyway (a new `Open` calls `O_CREATE`
  and re-acquires cleanly regardless of whether the old file is still
  there); on a hypothetical platform with no working recovery, manual
  deletion is the only lever, and it is the dangerous one. **This is the
  exact failure mode the card asked to avoid making worse: two hosts
  writing one workspace.** Anyone about to remove `.lock` by hand should
  first confirm, by process ID, that no process on the machine is the one
  that created it — not assume it from the file's age.

This paragraph, or a short version of it, is a candidate for Ryan's
user-facing docs (`docs/DEVELOPMENT.md` or a future troubleshooting page);
it is recorded here first because it is reasoning about a mechanism, not
yet a polished instruction to an end user.

## What this does not change

No change to `ports.StateStore`'s signature, `Commit`'s transaction shape,
the JSON file format, or `TestOpen_SecondOpenIsBusy`'s existing assertion
(it still holds, for the reason in point 2 above). `TestOpen_ResolvesSymlinkedRoot`
and the fsync-durability behaviour from the prior card are untouched.

## The CLI question, answered directly

**Does the CLI make this worse than the library did?** Yes, materially, and
that is why this card exists now rather than staying in the backlog: a
library caller who leaks a lock is a programmer who can attach a debugger,
read the source, and understand `Busy` as "something didn't call Close."
A CLI user who hits Ctrl-C, or whose terminal is killed, or whose laptop
sleeps mid-command, has none of that — they get an opaque `Busy` error on
their *next* attempt with no visible connection to what they did five
minutes ago, and no built-in way back in short of finding this document.
Before this card, that user's only recourse was deleting a file they had
no principled way to reason about — exactly the "delete it and hope" outcome
the fallback section above still describes as dangerous when it's the *only*
option. The fix earns its place because the CLI turns "a host was killed
without Close" from the exotic case into the ordinary one.
