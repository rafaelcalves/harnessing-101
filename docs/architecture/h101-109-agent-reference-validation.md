# H101-109 — agent-reference validation

2026-09-21. Source review of local `8da96de` and the engine diff in `1fcec3a`. The requested pull failed because the sandbox cannot write `.git/FETCH_HEAD`; local HEAD already equals the requested revision. Reviewed Kevin's memory, engine command/query paths and the added regression test. No execution/CI claim is made here; no engine or test changes. Kelly owns whether item 3's G5 limit closes. Other G findings are outside this review.

## Ruling on CreateTask

**Correct placement, shape, code, and optionality.** `internal/core/task/engine.go:123–133` checks a nonempty AssigneeID against the snapshot inside the commit mutation, before constructing the task. This is domain reference validation shared by every adapter. Checking the same serialized snapshot that receives the task avoids a separate preflight lookup against potentially different state. An adapter-only check would leave other callers able to bypass the rule. The existing duplicate-task Conflict check remains intact, and same-request receipt replay retains the established idempotency behavior.

`NotFound` is correct: boundaries.md's version-1 payload contract (line 24 at review) says unknown IDs fail NotFound. That includes references to existing entities, such as AssigneeID, not just the command's primary subject. A syntactically valid ID naming no registered agent is neither malformed input (`InvalidArgument`) nor a demonstrated authorization failure (`Denied`). This ruling does not prescribe a new precedence when multiple independent errors coexist.

Empty AssigneeID means no assignment, as permitted by the same paragraph's **optional assigneeID**. The conditional lookup preserves legal unassigned tasks; it must not resolve the empty value as an agent. Registration proves registry membership only, not authenticated identity or authority to act as that agent.

The added `TestCreateTask_UnregisteredAssigneeNotFound` checks NotFound and absence of the rejected task. I inspected it, but did not run it or re-certify the reported green CI. The change does not repair previously persisted dangling assignments; any cleanup would be a separate scoped decision.

## Other agent-ID references

**Yes, another command shares the defect: SendMessage.** At `engine.go:584–632`, sender and recipient are required to be nonempty, but neither is looked up in the registry before both are persisted. The mutation checks TaskID and ReplyToMessageID only. Under boundaries.md's same unknown-ID rule, both SenderAgentID and RecipientAgentID require existence checks inside this mutation and NotFound on a missing record. Suggested correction scope: independently exercise unknown sender and unknown recipient with the other endpoint registered, then validate both references before queuing. No correction made here.

Sender identity remains a routing claim under boundaries.md's event/mailbox contract and ADR 0003's authority rules. A registry lookup does **not** authenticate the sender and does not imply `SenderAgentID == caller.AgentID`; introducing that equality would be a different authorization change. Nor may recipient validation silently create an agent or treat a host reviewer principal as registered. A reviewer wishing to be an addressable message endpoint needs an explicit agent record, although reviewing a result does not require one.

| Remaining path | Finding |
| --- | --- |
| RegisterAgent (`463–498`) | Creates the record; checks nonempty ID and rejects duplicates. Requiring prior existence would prevent registration. |
| UpdateAgent (`501–537`), GetAgent (`541–550`) | Existing registry lookup and NotFound; no equivalent missing check. |
| ReportTaskResult (`230–264`) | Compares the bound actor with the stored task assignee. It introduces no new assignee reference; CreateTask now establishes existence for new assigned tasks. Legacy dangling tasks are not retroactively repaired by that fact. |
| AcknowledgeMessage (`640–660`) | Checks the stored recipient against the bound caller, not the registry. It introduces no new routing endpoint, but can act on a dangling recipient admitted by SendMessage or old data. The immediate defect is at message creation; this is not proof that old messages are valid. |
| Accept/Reject/Reopen, provenance/status records, receipt scope/ResolveRequest | IDs identify host-bound actors, review authority, audit claims or request namespaces, not new registry references. H101-94 explicitly permits reviewer authority without an agent record. Do not add blanket registry checks here. |
| Delivery recorders | Resolve an existing message and update its facts; no new agent endpoint is supplied. |

The engine has no implemented agent deletion/reassignment command or Phase 3 run command introducing another live agent reference in this reviewed version. This is a bounded source review of `internal/core/task`, not certification of file ingress, arbitrary stored-state repair, or future process-control validation.
