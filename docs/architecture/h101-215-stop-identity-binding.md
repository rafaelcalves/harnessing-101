# H101-215 — Stop identity binding on both targets

Architecture decision, 2026-09-22. Stop only; no code or QA-bar changes. Applies
Creed's H101-214 floor against signaling a reused process-group number.

## Selected mechanism

**Retain the spawned group leader's child lifetime until all possible group
signals are finished. Observe exit without reaping; serialize signaling and final
reaping through one adapter-owned lifecycle record.** Use this mechanism on both
darwin/arm64 and linux/amd64, with target-specific syscall plumbing for
`waitid(..., WNOWAIT)` where required.

This is a lifetime binding, not a later lookup of a number. POSIX prevents reuse
of a process ID during its process lifetime and defines a process to include a
zombie; retaining the child prevents its identifier from becoming an unrelated
new group leader's identifier. This is the design inference from the
[POSIX identity lifetime rules](https://pubs.opengroup.org/onlinepubs/9799919799/basedefs/V1_chap04.html#tag_04_17)
and [process definitions](https://pubs.opengroup.org/onlinepubs/9799919799/basedefs/V1_chap03.html).
`WNOWAIT` leaves observed child status available for later consumption.
[POSIX waitid contract](https://www.man7.org/linux/man-pages/man3/waitid.3p.html).

Local Darwin SDK `sys/wait.h` declares both `WNOWAIT` and `waitid`; this establishes
an available interface, not native qualification of our implementation. Both
supported targets still require native evidence.

## Lifetime and layering contract

The process adapter owns the direct-child relationship, group creation, private
run/attempt-bound handle, exit observation and one serialized signal/reap state
machine. Core receives typed outcomes and owns run/operation transitions. No PID,
start-time tuple or OS handle becomes caller authority or a new architectural port.

- Establish the group and ownership record from the actual spawn. Only its
  exclusive lifecycle owner may wait/reap or signal it. No auto-reap policy,
  competing waiter or current unconditional `cmd.Wait()` may consume the child
  while a group signal remains possible. Pipe draining must be managed separately;
  this is substantive adapter work, not an extra check before `Kill`.
- Before **each** SIGTERM/SIGKILL, verify under that same serialization that the
  run/attempt still owns the unreaped child and signaling authority remains active.
  Keep that protection through the syscall. Parent termination does not revoke
  the binding while its child lifetime is retained.
- Finish graceful/forced signaling while the binding is retained. A retained
  zombie may keep a group probe positive; never wait forever for that probe to
  become false before allowing final reaping. A bounded grace period may reach
  forced signaling even when the retained leader is already dead.
- Irrevocably close signaling authority before consuming the final child status.
  Then reap and confirm group disappearance for a clean stop. Any remaining group,
  permission failure or inconclusive observation gives uncertainty; **never send
  another signal to the old number after releasing the binding**. A newly reused
  number can cause a conservative false block, never an unrelated kill.

Concurrent Stop calls share this lifecycle; a stale timer/callback cannot signal
a released record. Host death loses the binding: persisted numeric identity cannot
restore authority. Existing conservative recovery remains in force.

## Why not start-time lookup

Linux `/proc` start-time and Darwin `sysctl`/`kinfo_proc` are platform-specific
identity observations. They can corroborate evidence, but are not selected as kill
authority: a check followed by a numeric kill leaves a check-to-signal race, and a
reaped leader cannot validate its surviving worker by its own start time. Choosing
a Darwin equivalent alone does not solve either issue. No handshake or startup
classification relaxation is introduced.

If either target cannot retain or validate this lifetime binding, do not signal:
return existing `RecoveryRequired`, retain exclusion, and report the limitation.
No new domain error or silent weaker fallback. Independently established group
disappearance may support completion; mismatch or lost ownership alone may not.

## Evidence and scope

Native tests on both targets must cover parent exit with surviving worker,
concurrent Stop/reap, fast-exit cleanup, denied/unknown observations, and stale
callbacks after binding release. Prove no signal reaches an unrelated process;
prove actual group disappearance before Exited. Distinguish parent termination
from reaping in fixture observations. Kelly owns any producer clarification.

This applies H101-128's existing no-unowned-signal rule and preserves H101-212's
startup rejection. It selects new adapter mechanics, not a weaker safety promise.
Creed must confirm this stronger alternative satisfies his floor before code;
this document does not claim runtime verification or waive his review.
