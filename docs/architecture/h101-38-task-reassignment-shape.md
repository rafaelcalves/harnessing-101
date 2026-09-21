# H101-38 — explicit transfer of unfinished task responsibility

2026-09-21. **Proposed design only; implementation is not authorized or scheduled.** Angela's product decision wants recovery/rebalancing on the existing task without losing its history. This document completes that design request; it neither reopens Phase 2 acceptance nor opens Phase 3. The orchestrator owns scheduling and integration.

## Command and authority

Propose `ReassignTask(requestID, taskID, expectedTaskRevision, expectedAssigneeID, newAssigneeID, reason)` on the existing Commands/FrontendSession surface. The host-bound human-review capability authorizes this administrative action, as it already authorizes reopening; ordinary agent scopes cannot transfer themselves or others. No payload field grants human authority. This reuses application-level authority, not authenticated identity or protection against same-user file writes.

Validate inside the same StateStore mutation that changes the task: task exists, caller is authorized, expected revision and old assignee match, target exists and is eligible, and status permits transfer. Missing references fail NotFound; stale revision/owner fails Conflict; unauthorized caller or an explicitly retired target fails Denied; empty target/reason or Done status fails InvalidArgument. Independent simultaneous failures need no new global precedence. An empty expected old assignee is legal for assigning previously unassigned work; explicit unassignment is outside this operation.

“Eligible” means registered and not retired from new work. It does not mean running, recently active, willing, suitably skilled, or verified. Today's Agent has no retirement state: registry membership alone cannot implement the full retirement policy. A future build must explicitly integrate the authoritative retirement decision; this design neither invents a runtime observation nor claims that lifecycle mechanism already exists. Check retirement and assignment against a consistent committed state, including races with retirement.

A request naming the current eligible assignee, with current preconditions, is a successful business no-op: no ownership record, task revision change, or ownership event. Its new request receipt still follows the existing workspace revision rule. Identical caller-scoped request replay returns the original receipt without applying another transfer; changed payload under that request ID conflicts. ADR 0004 governs uncertain outcomes and resolution.

## Permitted states

| Current status | Reassignment outcome |
| --- | --- |
| Todo | Change assignee; retain Todo. |
| Doing | Change assignee; retain the last reported Doing state, with transfer shown separately. This is not evidence the new owner started. |
| Blocked | Change assignee; retain Blocked and all retained blocker evidence. Clearing a blocker requires a separate explicit transition. |
| AwaitingReview | Change responsibility; retain AwaitingReview, CurrentResultID, submitted result, original reporter, and pending decision. |
| Done | Refuse; a human must explicitly reopen first, then transfer against the new revision. Keep prior acceptance and results. |

A pending result remains reviewable after transfer even if its author is retired or unavailable. Accepting that exact result may complete the task now assigned to someone else; acceptance credits the original reporter, not the replacement. Rejection records its own human decision and returns the task to Doing under the current assignee, without implying a process resumed. Replacing a pending result requires explicit rejection followed by a new owner's report; transfer itself neither rejects nor supersedes it.

## Atomic record and preservation

Each actual transfer updates AssigneeID, increments task revision once, and appends an immutable ownership-change record with old/new assignee, resulting task revision, request association, reason, and host-bound actor/time provenance. Emit `TaskReassigned` in the same commit group; workspace revision advances once. Expose the ownership records through detached query/snapshot values and both frontends, including after reopen. Persist them with task state; the in-memory event stream is not a durable history store.

Retain every existing result, review decision, contribution, message, acknowledgement and creator attribution. New ownership records preserve transfers prospectively; legacy ownership history remains unknown, never inferred from the creator. Storage layout, migration version and bounded/paginated history transport need implementation design before shipping; silent truncation or overwriting all but the latest transfer would not satisfy this contract.

Do not rewrite `Task.LastStatusChange` on an ownership-only change. H101-70 currently tells UIs to display it only when its revision matches the task; reassignment makes that equality obsolete. The build must deliberately amend that rule: the record remains the last actual status change at its original revision, may precede the current revision, and must have matching status. Never relabel its timestamp/actor as a new owner's progress report. Preserve available blocker evidence; current generic TransitionTask does not durably retain its Reason in the Task record, so do not claim recovery of lost historical reasons. A build must retain blocker reasons prospectively if it promises to display them after transfer.

## Late former-owner activity and races

The transfer commit is the ordering boundary. New result submissions by an actor who is no longer the assignee fail Denied, leaving task and current result untouched. Current-owner requests using a stale task revision fail Conflict. Review decisions use the exact pending result and current revision; a decision prepared before transfer must refresh. If a report commits first, transfer sees AwaitingReview; if transfer commits first, the former owner's new report cannot become current.

**Required companion change:** ordinary Todo→Doing, Doing→Blocked and Blocked→Doing transitions must require the current assignee or the host-authorized human, plus expectedTaskRevision. Today `engine.go:166–211` checks fromStatus but not owner or expected revision. Leaving that unchanged would let a former owner alter progress after transfer. Keep human-only reopening. Revision checks also reject an old command when ownership goes A→B→A; checking the assignee alone does not. This is an explicit command-contract change for a future build, not a claim about current enforcement.

A retry of an already committed request is historical receipt recovery, not new activity. Preserve the recorded receipt without duplicating effects, subject to established caller-scope authorization; do not retroactively erase a result or receipt because ownership later changed. Implementation must distinguish that replay from a new former-owner request and test both.

Late material must remain inspectable without disguising rejection as success. A denied ReportTaskResult commits nothing; its frontend preserves the supplied material for an explicit follow-up. Propose a separate `RecordTaskContribution` path, using a new request ID, that records task-linked material with its original claimant/time, observed task revision, and a late-former-owner classification established from persisted ownership records. It returns its own receipt and never sets CurrentResultID, changes status, or grants acceptance eligibility. Historical owners may contribute even when retired; this is existing-work material, not reactivation or new assignment. The current owner/human can inspect it and explicitly produce a new result with attribution. No automatic conversion or resubmission follows a failed or uncertain report. Exact contribution schema and frontend interaction remain implementation design, but dropping the inspectable-material path would require a product decision before building.

Existing message recipients retain their acknowledgement rights after task transfer; messages are neither rerouted nor acknowledged by changing AssigneeID. Ordinary messages remain subject to membership/routing rules and confer no task authority. No process is stopped, started, recalled or fenced, and no already-published file is withdrawn.

## Build evidence and non-decisions

Before shipping, demonstrate every status row; unauthorized/retired/unknown-target refusal; transfer versus report/decision races in both orders; former-owner status/result refusal; A→B→A stale-request refusal; historical replay; late contribution isolation; and preserved ownership/result/acceptance evidence across reopen and both adapters. Test equal-owner no-op separately from a real transfer.

This design decides no schedule, new phase, process control, automatic retirement/transfer, identity authentication, deletion, unassignment, batch transfer, new acceptance policy, or full reconstruction of past status history. No new architectural port is required; the existing command/query/event/store roles expand. No code or tests changed or run.

Sources inspected: Angela's memory and product ruling supplied in the dispatch; `docs/product/definition.md:98–106`; `docs/architecture/boundaries.md:28–38,44,95–105`; H101-109; `internal/core/task/engine.go` creation, transition, report and decision paths; `internal/core/domain/records.go`. Findings about current behavior are source inspection, not runtime evidence.
