# Troubleshooting

## A workspace will not open

Record the exact error before changing any file. The recovery depends on whether the product reports `Busy`, `Unsupported`, or another error, and on the operating system and filesystem holding the workspace.

### Supported macOS and Linux targets: open the workspace again

Workspace operation is supported on macOS on Apple Silicon (`darwin/arm64`) and Linux on x86-64 (`linux/amd64`), using native local storage. Workspace ownership on those targets uses a kernel-managed advisory lock. If the owning process crashes or is killed, the kernel releases that lock automatically; no cleanup command or waiting period is required. Open the workspace again.

If the next attempt still reports `Busy`, treat that as evidence that another live process has the workspace open. Find and close that process, or use a different workspace. **Do not delete `<workspace-root>/.lock` to recover from a crash on either supported target.** Its age does not prove that the lock is stale.

The reference environments are APFS on macOS and Ubuntu with ext4 on Linux. This is not a promise for every macOS version or Linux distribution. Network shares, cloud-synchronised directories, and unverified mounted or translated filesystems are excluded. Stop every process using the workspace before moving a copy to native local storage and trying again.

### Windows: workspace operation is unavailable

On Windows, opening the state store returns `Unsupported` with `workspace locking is not implemented on windows`. Help and version output may still run, but anything that opens a coordination workspace is unavailable. There is no fallback to unprotected or weaker writes: the product refuses to open the workspace.

This is an explicit scope decision, not a stale lock, a repairable lock file, or a promise with an implied date. Deleting `.lock` cannot make workspace operation available. Zero external dependencies is a preference rather than a prohibition; admitting Windows or any other target requires a separate architecture decision and native evidence. See [ADR 0001](adr/0001-language-and-runtime.md#dependencies-and-admission-of-another-platform) for those gates.

### What `pid=<n>` means

The `.lock` file may contain one line such as `pid=12345`. It is a diagnostic hint for a person. The product never reads that line back to grant ownership, reject ownership, recover a lock, or make any other decision.

The line is not proof that the named process still exists or still owns the workspace. Process identifiers can be reused, and file age proves nothing. Check the machine's actual process table when investigating: for example, `ps` on either supported target or Activity Monitor on macOS.

### Last-resort manual removal

> **Danger: deleting `.lock` while a workspace host is alive can allow two processes to write the same workspace. That can corrupt or misattribute state. Never skip the process check.**

This procedure is **not ordinary crash recovery** on a supported target; reopen the workspace first because the kernel releases a dead owner's lock. It is **not a Windows or excluded-platform workaround**. Consider it only in a genuinely stuck case on a named supported target, after every host has been stopped and the exact error or a maintainer's diagnosis establishes that the leftover file itself is blocking access.

1. Stop every known Harnessing 101 process that points at the workspace.
2. Read `<workspace-root>/.lock` only to obtain its diagnostic `pid=<n>` value.
3. Look up that process identifier in the machine's process table. Do not use the lock file's age as evidence.
4. If that process is present, or if you cannot prove the owning process is gone, **stop**. Do not remove the file.
5. Only after the process table confirms that the owner is gone, remove `<workspace-root>/.lock` and retry the workspace-open action.

If the error remains, preserve the exact message and report the operating system, filesystem type, workspace location, and process-table check. Do not repeat file deletion.

## A receipt shows an older workspace revision

A successful command receipt identifies the workspace version committed by that request. Background message delivery can record another change immediately afterward, so the receipt's revision may already be older than the workspace's current revision when you read it. This is expected and does not mean that the command was lost. A workspace revision records all workspace changes; it is not a count of your commands.

Replaying the same request describes the original request and returns its original receipt. It does not perform a new action or report the workspace's present revision.

## Technical background

For the mechanism, platform boundary, and evidence behind this procedure, read [Workspace lock recovery](architecture/h101-20-lock-recovery.md). That document is architecture reasoning; the steps above are the user procedure.
