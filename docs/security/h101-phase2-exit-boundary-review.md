# Phase 2 exit boundary review (H101-?)

Author: Creed, security. Reviewed at Phase 2 exit, main `bfa5f1b`. Read-only review;
no code touched. Evidence convention: CODE-PATH FACT / SECONDARY SOURCE / INFERENCE /
UNKNOWN, per this floor's standing convention.

## Scope read

**CODE-PATH FACT** — verified directly, not taken on relay:
- `internal/core/task/engine.go:584-633` `SendMessage`: checks `SenderAgentID` and
  `RecipientAgentID` against `findAgent` (registry membership), NotFound on miss. No
  check against `caller.AgentID`. Doc comment at `:579-581` states this explicitly.
- `internal/core/domain/records.go:154-197` `Message` struct doc comment: confirms
  `SenderAgentID` is "a claimed identity, not an authenticated one... checked against
  the agent registry (membership) but never against the caller scope... (authorship)."
  `Provenance` (`:109-120`) carries `ClaimedAgentID` (= actual caller) and
  `IdentityVerification`, permanently `Unverified` this phase (`:92-99`).
- `internal/adaptercontract/ui04_test.go:159-193`: both CLI and throwaway drivers
  register a third agent `claimed-engineer`, then send with `-caller
  expected.EngineerID` (a different, already-registered identity) but
  `-sender claimed-engineer`. The command succeeds. This is a real, executed
  divergence between caller and claimed sender, not a comment describing intended
  behavior.
- `docs/architecture/boundaries.md:111` (H101-109 amendment, dated 2026-09-21): rules
  SendMessage must check both IDs against the registry; also revises the UI-04 final
  workspace-revision cut from 14 to 15 to account for the extra registration this
  creates. Consistent with the code.
- `docs/architecture/h101-109-agent-reference-validation.md`: source review confirming
  the same gap, and enumerating every other agent-ID reference path (RegisterAgent,
  UpdateAgent, ReportTaskResult, AcknowledgeMessage, Accept/Reject/Reopen) with the same
  membership-not-authorship framing throughout.
- `docs/quality/h101-112-phase2-item2-discharge-ruling.md`: Kelly's Phase 2 exit ruling,
  all 9 items satisfied or satisfied-with-documented-limit; lists "H101-30 disclaimer" as
  an explicitly non-blocking open follow-up, not something resolved.
- No Phase 2 milestone document exists yet in the committed tree (checked: no file
  newer than the exit ruling commit, no `*milestone*`/`*phase-2-exit*` file matching a
  Phase 2 summary). I could not review a document that does not yet exist in the
  repository I can read — that is UNKNOWN, not clean.

## Threats live at this boundary, ranked

1. **Sender-claim impersonation through the new CLI surface, using a registered
   identity that is not the caller's own.** This is the same exfiltration/injection
   family named in threat-model.md §4-5, now reachable through a real external
   interface (Phase 2's stated purpose) instead of only headless/theoretical. Anyone
   who can invoke the CLI as *any* registered caller can address a message as coming
   from *any other* registered agent, and every downstream reader (human or agent) sees
   a `SenderAgentID` field that looks like an address. **Live, unchanged by Phase 2,
   now exposed through a real adapter.**
2. **Registry-membership checks add real but narrow value, and it is easy to overstate
   what they buy.** They stop typos and dangling references (a nonexistent ID can't be
   used), which is genuine data-integrity value. They do essentially nothing against
   the threat in #1: agent IDs are meant to be enumerable (GetSnapshot/GetAgent are
   coordination primitives, not secrets), so knowing a valid ID to impersonate costs an
   attacker nothing. Do not let "the sender is validated" appear anywhere as a security
   claim — it is a data-integrity claim only.
3. **Provenance/`SenderAgentID` divergence is exactly the kind of two-field state a
   future reader or adapter can quietly collapse.** Nothing today merges them, and the
   `Message` doc comment is unusually explicit about why not to — but this is a design
   invariant held by documentation and one regression test (`ui04_test.go`), not by a
   compile-time or runtime guard that would fail loudly if someone "fixed" it later by
   asserting `SenderAgentID == caller.AgentID`. Lower severity than #1 (nothing is
   exploitable today because of it) but worth naming as a durability risk.
4. **Manual-start exposure from threat-model.md §5 is unchanged and still fully open** —
   Phase 2 shipped the CLI, not process control; nothing here narrows that window.
   Restating, not new: ranked lowest only because it is already tracked and Phase 3 is
   the agreed place to address it, not because it is less real.

## Verdict on item 1 — is the SendMessage gap bounded, or documented-but-unbounded?

**INFERENCE.** Bounded in the narrow sense that matters for honesty: the code, the
architecture ruling, and the domain doc comments all say the identical thing, and I
verified that alignment directly rather than trusting any one of them. There is no gap
between what is claimed and what runs. **It is not bounded in the sense of reducing the
underlying risk** — the registry check is a data-integrity control, not an
authentication control, and the threat named in ranked item 1 is exactly as exploitable
after this change as before it. Calling this "safely bounded" without qualification
would overclaim; calling it "merely documented" undersells the real (if narrow) value of
rejecting dangling references. My phrasing for the record: **the gap is honestly and
precisely documented, and the documentation is not covering for a control that isn't
there — but nothing about Phase 2 narrows the actual exposure.**

## Verdict on item 2 — does the adaptercontract proof establish a safety property?

**INFERENCE, yes, with a specific and limited scope.** What it proves: the system
accepts and durably records a `SenderAgentID` that differs from the actual caller
identity, on both adapters, and does not silently coerce, reject, or reinterpret that
divergence. That is a safety-relevant regression guard — it is the thing that would need
to keep passing to catch a future change that quietly started enforcing
`SenderAgentID == caller.AgentID` (which would be an undocumented authorization change,
per Kelly's H101-111 note that this must be a third agent, not caller or recipient
reused) or that started silently trusting the sender field somewhere. It is not a
control that resists impersonation — it demonstrates that impersonation-shaped input is
accepted, precisely because that is what the design says should happen. Do not describe
it as proving "sender identity is handled safely" in any user-facing sense; the accurate
description is "proves sender is independent of caller, which is the documented and
accepted design, not a security boundary against a hostile caller."

## H101-30 — does the signing condition fire now?

**Not yet, on the evidence I can read, and this needs a recheck before the Phase 2
milestone doc lands.** I searched the full committed tree for the phrase and found
exactly one place it appears: `threat-model.md` §7, which I wrote, and which carries the
disclaimer in the same paragraph by construction. `definition-of-done.md:882` references
H101-30 by name as a general standing constraint on quoting *any* qualified claim
(applied there to the "runs completely locally" sentence, correctly, with the
qualification intact) — that is the condition being actively enforced, not violated. The
Phase 2 milestone document god mentions is not yet in the repository I have access to,
so I cannot confirm or deny anything about it; that is UNKNOWN, not a clean bill. **Flag,
not a finding: whoever finalizes that milestone doc should run the same grep
(`"inspectable"` / `"trust becomes"`) before commit, and attach the disclaimer sentence
wherever the phrase is quoted, per my own H101-26 condition.**

## What Phase 3 must not be allowed to assume

1. **That registry membership on `SenderAgentID`/`RecipientAgentID` is any form of
   authentication.** Phase 3 process control must not treat a message's claimed sender
   as a trustworthy signal for anything privilege-bearing (e.g., do not let a
   `StartRun`/budget decision key off a message's claimed sender without going through
   the same host-caller-scope path everything else uses).
2. **That the manually-started-agent exposure (threat-model.md §5) closes just because
   Phase 3 adds product-managed process control.** ADR 0002's own revisit trigger says
   this explicitly: product-managed start approval does not retroactively validate
   agents started outside it, and Phase 1-2 workspaces/histories carry forward.
3. **That the `ProcessSupervisor.Start` approval gate (my original rule 5) is a
   solved problem just because it now has somewhere to attach in Phase 3.** The gate
   stops the product from starting an unapproved profile; it says nothing about content
   already sitting in a workspace, or a sender field already trusted by some downstream
   consumer that predates the gate.

## Clean-review statement

No new defect found beyond what is already tracked. Everything reviewed here matches
its own documentation exactly, which is itself the main finding worth recording: Phase 2
did not quietly widen or narrow the SendMessage exposure named in Phase 0 — it shipped
the first real surface onto an unchanged, honestly-labeled gap. That is a real result,
not an absence of one.

---

## Correction (2026-09-21, after H101-119)

The reviewer revised this review's own item 2 verdict after the architecture
ruling in `h101-119-provenance-sender-durability.md` (`ac0dda2`) found the
fixture protects less than this review credited. In the reviewer's words:

> Original wording: "a regression guard proving the system won't silently start
> enforcing SenderAgentID==caller." That's wrong as stated — it only proves the
> system won't LOUDLY start enforcing it, because a rejection on differing
> sender/caller would fail the fixture.

> Corrected wording for the record: "The adaptercontract fixture proves the
> command does not REJECT a sender that differs from the caller. It does not
> prove the two identities are stored, returned, and reopened as distinct
> values — a silent coercion of one into the other would pass this fixture
> undetected. Kelly/Stanley's remedy (explicit caller/sender assertions across
> query, reopen, and three local mutation checks, ac0dda2) is what actually
> closes that gap; the original fixture alone did not."

The risk was ranked correctly as a durability concern held by one test rather
than a hard guard; what was undersold was how thin that one test was.
