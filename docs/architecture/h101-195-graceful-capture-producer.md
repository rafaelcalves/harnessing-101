# H101-195 — Graceful capture producer

Architecture ruling, 2026-09-22. Shape only; no code or QA-bar edits.

## Decision

**Use a continuing `harnessing serve` host. Keep the accepted fast-exit classification unchanged.** This explicitly changes [H101-193's named producer](../quality/h101-193-item5-acceptance-spec.md); Kelly must amend that producer before implementation resumes. Its output assertions need no weakening.

The current `Supervisor.Start` rejects exit during `StartupWindow`, including exit zero; `TestSupervisor_FastExitIsSpawnFailed` deliberately covers `true`. Capture goroutines live in the process running the supervisor. A one-shot CLI cannot promise to capture a child's later output after that CLI exits. Changing a timeout does not fix that lifetime mismatch.

[H101-128](h101-128-phase3-supervision.md) already places process ownership and journal writing in a continuing host; CLI sessions attach and detach independently. Use that ownership model for this proof rather than teaching Start about fixture markers.

## Producer shape for Kelly's amendment

1. Start the real serve host and wait for readiness after workspace ownership is acquired. Register/approve and issue StartRun through shipped CLI clients attached to that host.
2. The participation fixture remains alive through successful startup. Its later emission/exit must be ordered after the successful start observation, using a bounded fixture synchronization mechanism specified by QA. Do not rely on racing a fixed sleep against the startup window.
3. The start client receives its successful receipt and may exit. That is completion of the **start operation**, not completion of the child or output capture. Keep the owning host alive.
4. Release the fixture to emit the named stdout/stderr literals and exit normally. The continuing host observes completion and drains both channels, durably appends their captured bytes, then records complete capture only when justified. Exit code zero or a successful start receipt alone is insufficient.
5. Fresh CLI subprocesses attach to that same owner for output, offset-resume and StateEvents assertions. “Fresh CLI” must not mean a fresh competing workspace owner. Wait boundedly for the terminal capture observation before asserting it. Host teardown follows the evidence collection.

This requires the shipped session/transport path to carry the needed output reads if it does not already. A direct journal read, `host.Open` in the test, or shutting down the owner merely to bypass its lock is not an equivalent producer. Extending the host path is shared implementation work; it does not automatically discharge item 6, just as shared journal code does not automatically discharge item 5 under [H101-190](h101-190-d4-journal-boundary.md).

## Preserved contracts

No exit-zero exception, context-file exception, sibling-marker check, tool-name branch or fixture-content inspection may promote fast exit to successful startup. Fixture synchronization belongs in the condition producer, not the supervisor's production acceptance decision. `true` must still fail the existing fast-exit test.

Surviving startup is also not proof of meaningful participation; existing participation evidence remains independently required. Capture completion, successful startup, task completion and verified process-group disappearance are different facts. Do not manufacture any one from another. Missing drain or persistence evidence remains interrupted/unknown or an exposed failure, never complete by assumption.

The item 5 observables remain real child output, both exact channels, durable ordered resume and StateEvents separation. Existing follow/cancel/retention limits remain for Kelly to judge; this ruling changes none of them and grants no independent engineering resume authorization. God owns redispatch after the producer amendment and any required review.
