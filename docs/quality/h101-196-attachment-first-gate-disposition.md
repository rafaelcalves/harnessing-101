# H101-196 — Attachment-first gate disposition (2026-09-22)

Kelly QA. **Criteria only** — uncommitted pending god commit.

**Authority:** owner H101-174 answer (god relay `2026-09-22T11-57-29-614Z-d91d26`):
attachment-first (**A**); managed launch (**B**) on near-term backlog, not dropped.
H101-172, H101-171 (Stanley, committed), H101-153 deferral mechanism.

**Preserved line (Kelly + Stanley):** manual attachment **cannot** be relabelled
managed-start evidence. Owner choosing the cheaper path does **not** soften D13.

---

## (1) Item 1 Layer B — Claude Code slot

**Verdict: closes by OWNER DEFERRAL — not stay open, not manifest, not registration.**

Under attachment-first, Claude Code managed `StartRun` is **out of Phase 3 scope**.
The Layer B slot discharges the same way Codex and Cursor Agent already do:
`deferred: true` in the committed descriptor (H101-153). Registration work (item 10)
is a **separate exit item**; it does **not** close or substitute for Layer B.

**Item 1 Layer B row after deferral is committed:** **SATISFIED BY OWNER DEFERRAL**
(all three named tools then manifest-or-deferral complete). **Item 1 overall** may
become **SATISFIED** if no other rows remain open — Kelly re-verdicts on commit,
not on this ruling alone.

### What must be recorded (deferral ≠ demonstrated compatibility)

| Location | Required content |
| --- | --- |
| `scripts/agentic-cli-manifests/claude-code.json` | `deferred: true`; `deferral.reason` states **managed StartRun deferred by owner H101-174**, not "tool unavailable" |
| Same descriptor | `deferral.reference`: `H101-174 attachment-first (A); managed launch (B) on backlog` |
| Same descriptor | `deferral.backlog`: `{ "track": "managed_start_run", "nearTerm": true, "reopensWhen": "runner-produced manifest per H101-153" }` — **new optional field**; distinguishes from Codex/Cursor indefinite deferrals |
| `phase3-exit-criteria.md` | H101-196 standing row: Layer B Claude **SATISFIED BY DEFERRAL** — **not** demonstrated managed-start compatibility |
| Item 8 disclosure | Phase 3 does **not** claim Claude Code was started through product-managed `StartRun`; user-paired attachment is item 10 scope (CF2 family) |
| Item 9 guide supplement | Claude listed as **deferred managed-start**; participation walkthrough points to **item 10** registration path, not Layer B manifest |

**Forbidden readings after deferral:**

- Item 10 pass ⇒ Layer B manifest equivalent
- Registration manifest with `evidenceClass: registered_external_session` ⇒ item 1 discharge
- Backlog **B** ⇒ managed launch "mostly done" for exit purposes

---

## (2) New exit item — registered external session

**Verdict: write it now.** H101-171 removed the blocker; Stanley named pairing,
mailbox delivery, cooperative check-in (H101-177), and proof limits. Kelly freezes
acceptance in a **single** item (not a pair with managed launch).

**Deliverable:** [`h101-196-item10-registered-session-acceptance-spec.md`](h101-196-item10-registered-session-acceptance-spec.md).

**Phase 3 exit table:** becomes **10 items** — item 10 inserted after item 9.
Attachment-first delivery **requires** item 10 for Claude Code participation;
managed-launch backlog **B** does not substitute.

**Still provisional at implementation time (not blockers to writing criteria):**

- Exact shipped CLI verb for pairing (H101-171 proposes shapes; engineer names argv)
- Layer B registration runner after first green Layer A fixture
- Security review gate before construction (H101-171) — blocks **code**, not this bar

---

## (3) Backlog B vs plain deferral

**Not recorded identically.** Same **gate mechanism** (H101-153 `deferred: true`
closes the Layer B slot); **different metadata and Phase 3 obligations.**

| Aspect | Codex / Cursor plain deferral | Claude deferral + backlog B |
| --- | --- | --- |
| Layer B slot | Closed by deferral | Closed by deferral (same) |
| Descriptor | `deferral.reason` = no invocation evidence | `deferral.reason` = owner deferred managed launch; `deferral.backlog` present |
| Phase 3 participation path | None required (tool not in scope) | **Item 10 required** for Claude attachment |
| Exit claim | "Managed start not demonstrated for this tool" | Same **plus** "cooperative registration demonstrated via item 10" |
| Post-Phase-3 | Replace deferral with manifest when owner supplies evidence | Backlog **B** tracked; manifest replaces deferral when runner produces H101-153 output |

Backlog **B** does **not** add rows to item 1, soften D13, or let attachment
evidence stand in for a future managed-start manifest.

---

## Honest ceiling (carry to owner)

Phase 3 can exit with:

- Item 1 Layer B for Claude closed by **deferral** (not manifest)
- Item 10 proving **user-paired cooperative check-in** for Claude Code
- **No** demonstrated Claude Code managed `StartRun` in Phase 3

That is a **real hole in the managed-start compatibility claim** for Claude Code
specifically — not a product defect, an **explicit scope choice**. Release wording
must say attachment/participation is proved; managed process control for Claude is
**deferred to backlog B**, same class as Codex/Cursor deferral for compatibility
**demonstration**, but with item 10 additionally proving the attachment path.

---

## H101-195 (inform — no new ruling)

Kelly input already with Stanley: one-shot `emit_both_channels_then_exit` blocked
by StartupWindow (`ErrSpawnFailed`); prefer **serve-based** producer if Stanley
rules serve; I1–I6 hold on fresh `run-output` + events subprocesses. Creed gated
fast-exit relaxation. If Stanley rules serve, amend H101-193 producer only.

---

Authored by Kelly (QA), H101-196.
