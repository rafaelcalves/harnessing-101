# H101-226 — Native membership observation and dependency decision

Architecture recommendation, 2026-09-22. Owner decision required before adding a
dependency. No code, module, gate or acceptance changes made.

## Decision

**Recommend a pinned `golang.org/x/sys/unix` dependency for Darwin's native
process-group membership observation. Keep H101-222's predicate unchanged.**
Use its Darwin `SysctlKinfoProcSlice` wrapper for `kern.proc.pgrp` with the retained
group identifier. Linux may retain its adapter-local `/proc` implementation;
neither platform exports native process records into core.

Proposed pin: **`golang.org/x/sys v0.44.0`**, the locally inspected source version,
subject to verified acquisition and dependency review below. This is not a claim
that the extracted cache is authenticated or that native behavior has passed.
Its module declares Go 1.25.0 and no module requirements; this repository declares
Go 1.27.1. Its licence is BSD-3-Clause, with notice requirements to preserve.

[ADR 0001's dependency policy](../adr/0001-language-and-runtime.md#dependencies-and-admission-of-another-platform)
already treats zero dependencies as a preference and prefers reviewed OS bindings
over hand-written low-level calls. This recommendation applies that policy; it
does not silently authorize installation or require a new runtime language.

## What this buys, and what it does not

The inspected Darwin wrapper uses libSystem syscall trampolines and returns typed
`KinfoProc` records, including buffer-size checks and retry after table growth.
It avoids project-owned cgo or copied assembly/ABI definitions. Build and verify
with `CGO_ENABLED=0`; internal generated `cgo_import_dynamic` directives must not
be confused with requiring a C toolchain for this application.

This supplies an observation mechanism, **not automatic proof of absence**. The
adapter still must distinguish the retained leader from other members, preserve
its lifetime binding, account for membership races, and treat errors, incomplete
results or lost binding as `other_members_uncertain`. Linux process-file reads
have the same proof obligation; readable `/proc` does not itself establish a
race-safe negative result. Repeated empty scans or pipe EOF cannot simply be
renamed confirmation.

Native evidence must establish that the chosen inspection supports the predicate
on each qualified target. If it cannot, retain the conservative outcome and report
the unresolved capability; adding the module does not waive H101-222 or authorize
speculative reap. Core and queries remain free of process-table mechanisms.

## Why not permanent Darwin uncertainty

Always returning uncertainty is safe for this predicate, but is a material runtime
limit, not merely missing coverage. Naturally completed runs retain leader slots
and ownership records; automatic run completion/resource reclamation cannot be
claimed. Complete output may still be readable because capture is separate.
Stop remains available while ownership is retained, but users may need explicit
Stop even after apparent natural completion. Bounded retention then requires
admission backpressure or another explicit lifecycle policy; silently accumulating
zombies/records is not acceptable.

H101-222 permits uncertainty for unavailable evidence. It does not approve that
as the permanent normal behavior of an in-scope platform. If the owner declines
the dependency, record this as a product capability deferral with resource/admission
limits and Kelly's acceptance disposition before shipping it. Do not silently
substitute Linux-only successful finalization for the two-target commitment.

## Alternatives and admission conditions

A new cgo build path is unnecessary for the inspected binding. Shelling out to
`ps` adds executable/output-format dependencies and does not establish stronger
absence evidence. Copying libSystem trampolines into this repository merely moves
ABI and maintenance responsibility here. None is preferred to the narrow binding;
this is a maintenance decision, not a claim of platform impossibility.

Before admission, god should obtain the owner's dependency scope decision, then:

- Acquire the exact pin through the normal verified module workflow and commit
  genuine `go.mod`/`go.sum` entries. Do not disable checksum verification or invent
  metadata from the partial cache. Build-time acquisition is explicit; runtime
  network access remains prohibited.
- Keep imports in the process adapter's platform files. Preserve existing import
  containment and no-network gates; run them on both target dependency graphs.
  The inspected scripts do not ban every external module, but that is not a
  substitute for dependency approval.
- Record licence notices, resolved transitive/build requirements and maintenance
  responsibility with the process-adapter owner; pin upgrades require review.
- Validate native membership, uncertainty, survivor and natural-completion cases
  on Darwin and Linux CI. Cross-compilation is not native evidence. Verify clean
  builds without cgo and normal distribution without an added C toolchain.

Kevin's report and the local cached wrapper were inspected; his failed raw-syscall
experiment was not rerun. No dependency was fetched or installed by this ruling.
