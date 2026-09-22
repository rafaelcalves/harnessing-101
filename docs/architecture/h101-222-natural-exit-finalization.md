# H101-222 — Natural exit must preserve Stop authority

Architecture ruling, 2026-09-22. Read Kevin's reproduced-bug report
`2026-09-22T13-48-05-446Z-8bdb31` and current supervisor/wait helpers. No code or
acceptance-bar edits; Stop's existing security floor is unchanged.

## Decision and layer

**Keep autonomous lifecycle observation in the process adapter. Separate capture
completion from leader reaping and group completion.** Do not move process
inspection or finalization policy into GetRun/run-output merely to avoid the
background race. A query-triggered adapter check could be an optimization later,
but is neither required nor a substitute for correct ownership predicates.

The defect is reaping on leader termination alone, not the existence of a
background observer. Checking group existence after reaping is too late. Checking
for an empty group before reaping using the existing `kill(-pgid, 0)` is also
insufficient: the deliberately retained zombie may itself keep that probe positive.
Neither proposed timing change fixes that predicate.

## Required lifecycle shape

Use one adapter-owned, serialized lifecycle record for natural observation and
Stop, retaining the run/attempt binding from [H101-215](h101-215-stop-identity-binding.md).

1. Observe leader termination without consuming status. Record that fact, but
   retain the leader, signaling authority and reachable run record while other
   group members remain or their absence is unknown. Mode B must remain stoppable.
2. A spontaneous natural-reap path requires positive evidence that no group member
   **other than the retained terminated leader** remains. This needs an adapter
   observation capable of distinguishing those cases; the current boolean group
   probe cannot. Pipe EOF and leader exit are not substitutes: a surviving worker
   may close or redirect both output descriptors.
3. That observation must account for membership changing during inspection; a
   partial enumeration or unreadable process is uncertainty, not an empty group.
   Platform inspection stays behind the process adapter and needs native evidence
   on both targets. Do not introduce a core process-table reader or weaken the
   supported-group contract. If reliable absence cannot be established, retain
   the binding for Stop rather than reap speculatively.
4. Natural finalization and Stop claim the same serialized record. Once Stop has
   claimed it, the natural observer must not steal its leader or finish its work
   on its behalf. If natural finalization safely wins, close signal authority
   before reaping, retain the resulting outcome for concurrent callers, and
   confirm group disappearance before reporting run completion.
5. Stop keeps H101-215's bounded signal sequence, authority-close-before-reap and
   prohibition on post-release signals. A closed record means only that signaling
   is closed; concurrent callers must await the actual terminal result, not infer
   successful group termination from `closed` alone. Do not forget the record
   before that result is available to its waiters.

This is a substantive adapter fix. A stronger pre-reap membership observation may
be needed for autonomous resource cleanup. If unavailable, expose that limitation
and use bounded observation with honest uncertainty; do not silently spin forever,
leak unbounded records, or automatically kill a healthy descendant to simplify
natural completion. Retention limits and host teardown must preserve the same
no-signal-after-release rule.

## Output completion and timing

Own stdout/stderr drain completion and append failures explicitly, independently
of the consuming wait. `Wait4` avoids the inherited-pipe wait but does not prove
that copying or persistence finished. `Finish(Complete)` requires both channels
drained and all captured bytes durably appended; failure or bounded incomplete
drain must remain honest. A live worker holding a pipe open prevents that proof.
A live worker that closed both pipes need not prevent **capture** completion, but
still prevents **run** completion and unsafe release of Stop's binding.

Thus item 5 may continue to publish complete capture asynchronously, without any
query causing a reap. No query-dependent observation-timing change is selected.
Kelly retains item 5 acceptance, including the independent full-output evidence;
a test passing because it polls does not justify changing the lifecycle contract.

This applies H101-215, including its required separation of reaping from drain;
it changes neither the ownership model nor startup acceptance. Evidence should
cover the mode-B survivor both holding and closing inherited pipes, natural-exit
versus Stop races, a self-contained graceful capture, and concurrent callers
receiving the same final outcome. Existing security and QA gates remain in force.
