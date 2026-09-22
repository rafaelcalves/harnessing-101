# H101-190 — D4 and item 5 journal boundary

Architecture ruling, 2026-09-22. Ruling only; no code or acceptance-bar changes.

## Decision

**D4 may introduce the minimal real OutputJournal slice specified by H101-189 before full item 5 acceptance. Item 4 need not wait for full item 5.** That slice is shared journal foundation, not a different kind of output persistence exempt from item 5's architecture.

The proposed alternatives conflate implementation ownership with acceptance. Durable run-output persistence belongs to the OutputJournal role, but that does not reserve all journal implementation until item 5 is accepted. [H101-128](h101-128-phase3-supervision.md) assigns byte persistence to OutputJournal, and its acceptance map explicitly assigns “Crash reconciliation + journal” to **4–5**. Its item 4 recovery contract requires readable historical output; its item 5 section supplies the shared persistence semantics.

[H101-189](../quality/h101-189-item4-d4-acceptance-spec.md) does not impose a blanket prohibition on early journal work. It explicitly expects first real journal code and makes an item-5-only ruling a conditional stop. This ruling does **not** trigger that stop.

## Required boundary

The minimal path must use the existing OutputJournal responsibility and expose RunOutput through the shipped command/session surface. Preserve [boundaries ports 4 and 8](boundaries.md): complete persisted chunks, offsets allocated only after persistence, channel identity, capture metadata, and honest capture outcome. Raw stdout must remain separate from domain StateEvents. No second output ledger, fixture-specific persisted copy, or direct presentation read of storage may substitute for that path.

For D4, the proof remains exactly H101-189's crash case: known durable prefix, exact fresh-command read, interrupted/unknown capture, no invented tail bytes, and no count/range covering unpersisted data. A readiness signal must follow actual durable append; fixture emission alone proves neither capture nor persistence. Marking capture interrupted establishes neither process death nor permission to restart. Do not reconcile a continuing host's own supervised run as abandoned.

A minimal implementation may leave capabilities unfinished, but must expose that honestly. It cannot silently discard output, fake complete capture, or advertise working follow/retention behavior it does not provide. Existing safety and failure semantics still apply to the behavior it ships; narrow evidence is not an exemption from those contracts.

## Acceptance consequence

Passing D4 proves crash-prefix honesty. It does not prove graceful stdout/stderr drain and complete capture, reader cancellation without stopping the run, or the remaining journal failure/retention behavior. [Item 5's independent acceptance](../quality/phase3-exit-criteria.md#item-5--real-run-output-journal) remains required. Shared code and relevant evidence may be reused; no duplicate journal is needed, and no automatic item 5 credit follows from closing D4.

Violations include claiming item 5 from D4 alone, treating the prefix implementation as outside OutputJournal rules, or replacing the real capture path with a test-only stream. God may dispatch the minimal shared foundation for D4 under the existing bar; Kelly retains the acceptance verdicts. No ordering change requiring full item 5 first is warranted.
