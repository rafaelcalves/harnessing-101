module github.com/rafaelcalves/harnessing-101

go 1.27.1

// golang.org/x/sys v0.44.0 -- this repo's first and only third-party
// dependency. Owner-approved per docs/architecture/h101-226-native-membership-dependency.md
// (Stanley's recommendation, H101-226); full reasoning, admission
// conditions and alternatives considered live there, not here.
// Used ONLY by internal/adapters/process, on darwin only, for
// H101-222/H101-224's process-group membership predicate: its
// SysctlKinfoProcSlice wrapper for "kern.proc.pgrp" against the
// retained leader's pgid. Linux uses an adapter-local /proc scan
// instead -- no x/sys import on that target. ADR 0001 treats zero
// dependencies as a preference, not a ban, and prefers a reviewed OS
// binding over hand-written low-level syscalls for exactly this case.
// Any future version bump requires the manual syscall-behavior review
// recorded in scripts/check-no-network.sh and docs/security/threat-model.md
// ("Gate limit (Creed, H101-227)") -- never a semver-trusted auto-update.
require golang.org/x/sys v0.44.0
