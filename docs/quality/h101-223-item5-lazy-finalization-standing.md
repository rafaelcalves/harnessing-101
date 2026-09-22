# H101-223 — Item 5 lazy finalization standing (2026-09-22)

Kelly QA. Authority: H101-207 discharge at `e1a2e4b`; god
`2026-09-22T13-49-25-000Z-kelly`; Kevin H101-222 bug context; Stanley design ruling
pending.

---

## Q1 — Does item 5 remain SATISFIED WITH LIMIT?

**Yes — unchanged.**

The accepted bar never required eager background finalization “a few milliseconds after
exit.” H101-193 already says **“Wait boundedly for terminal capture before asserting I2.”**
H101-207 I2 is **“`Capture status: complete` after bounded wait.”** The owner sentence
guarantees durable complete capture **readable through `harnessing run-output`** — not
spontaneous completion without a reader.

Lazy reap/finalize on a run-state or `run-output` query is **compatible** with that
acceptance. The existing LIMIT rows (reader cancel / `FollowOutput` stub) are unchanged.
No limit wording amendment required.

**Clarifying note (not a new limit):** if Stanley adopts lazy finalization, `complete`
may appear only after a query path runs — that was always how `e1a2e4b` was proved.

---

## Q2 — Is incidental polling acceptable evidence?

**No — but the current evidence is not incidental.**

| Layer | What exists |
| --- | --- |
| **Bar** | H101-193: bounded wait before I2 (already written) |
| **Test** | `item5_test.go` lines 119–134: explicit `run-output` poll loop, **5s** deadline, **20ms** interval, fail if never complete |

That is acceptable evidence **because the poll is the asserted mechanism**, not a hidden
pass condition. Same lesson as H101-221: the bar must name what the test depends on.

**H101-223 bar tighten (amend H101-193 I2):** I2 now reads:

> `Capture status: complete` on a **fresh attached `harnessing run-output` query** within
> a **≤5s** bounded poll (≤20ms interval) after fixture natural exit.

Stanley may rule lazy finalization on shape; Kelly does **not** need to re-verdict item 5
after implementation if I2 still passes under this explicit poll contract. Re-verdict only
if a query never reaches `complete` within the bound, or if completion requires a path
other than `run-output`/`run` observability the bar does not name.

---

## Stanley input

Item 5 standing **does not block** lazy-on-query finalization. Creed/Kevin item 2 binding
work is separate.

---

Authored by Kelly (QA), H101-223.
