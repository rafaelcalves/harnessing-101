# H101-212 — Parent exit during startup

Architecture ruling, 2026-09-22. Reviewed the uncommitted `supervisor.go` diff,
[H101-209](../quality/h101-209-item2-acceptance-spec.md) at `ad382c9`, and
[H101-195](h101-195-graceful-capture-producer.md). Scope: the new fast-exit branch;
no Stop implementation verdict or code change.

## Decision

**The branch is an unauthorized exception to the accepted startup-window rule.**
Returning nil from the fast-exit case solely because `groupExists` is true changes
startup acceptance. Recording the process group does not put that decision outside
the rule. A launcher can fork an idle worker and exit, and this branch accepts it;
it also runs before checking `waitErr`, so even an unsuccessful parent exit can
be promoted to successful startup.

Ownership evidence answers which execution may be supervised. It does not prove
successful startup or meaningful participation. Conversely, the existing startup
window itself does not prove meaningful participation; that evidence remains
separate. Adding only a sleeper test cannot make this policy change acceptable.
Keep the fast-exit rejection, including the existing `true` case, unchanged.

## Required producer ordering

H101-209 mode B requires the parent to exit **before its worker**, not before
startup completes. Those are different orderings. The producer must:

1. Start the real participation fixture and its same-group worker under the
   continuing serve owner; keep the parent alive through successful startup.
2. Obtain the successful start receipt and observe Running through the attached
   product CLI before releasing the parent to exit. QA must specify bounded
   synchronization, not a sleep tuned against StartupWindow.
3. Establish that the parent has exited while the worker remains alive, then
   prove S1–S3 through the same owner: no premature Exited, StopRun still needed,
   and terminal Exited only after the worker is gone.

Kelly should explicitly amend/clarify mode B's producer ordering before rework;
the required parent-before-worker case and S1–S3 remain intact. Fixture release
is test coordination, never a marker inspected by the production supervisor to
accept a start.

## Keep lifecycle facts separate

After a valid start, parent exit alone must still not complete the run while an
owned worker remains. That existing group-lifecycle obligation does not authorize
a new startup-success path. Likewise, a rejected fast start with surviving workers
must not be treated as proof that the group disappeared or that replacement spawn
is safe. Preserve honest ownership/cleanup uncertainty; this ruling does not
approve a restart or review Stop's implementation.

For regression evidence, add the direct negative case: a fast-exiting launcher
leaves an idle same-group worker, yet Start does not return success. Test cleanup
must account for that worker. This complements, rather than replaces, the positive
mode B case after successful startup. Creed retains the security ruling; god
retains the commit/redispatch decision.
