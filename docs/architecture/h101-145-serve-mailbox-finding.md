# H101-145 — serve mailbox completeness finding

**Finding: real serve-completeness gap, but not a Phase 3 item 6 acceptance
gap.**

`harnessing serve` opens the workspace and runs the file transport host in
`internal/assembly/serve.go`; it does not open the mailbox or drive a
`mailbox.Deliverer`. The one-shot path in `internal/assembly/session.go` calls
`driveMailbox` once after each command. A client attached to a long-running
serve host therefore has no host-side pass that publishes queued envelopes,
records processing, or ingests acknowledgement files after the client
detaches.

This is a product completeness gap against the continuing-host design in
`docs/architecture/h101-128-phase3-supervision.md`, which says the sole host
“runs the existing mailbox Deliverer.” It is not a defect against H101-143's
item 6 foundation slice: Kelly explicitly ruled mailbox background delivery
correctly scoped out of that item's run-state/output observability criterion
in `docs/quality/h101-143-phase3-item6-foundation-ruling.md`. It is also not
dependent on `StartRun`; mailbox coordination can be exercised independently
through an attached session. StartRun sequencing can remain with H101-144.

## Smallest change

Give the continuing host a serialized periodic mailbox pass. The composition
root should construct the mailbox and `Deliverer`; the transport host should
invoke an assembly-provided tick while it owns the workspace, so mailbox
recording cannot race transport mutations. The pass should retry on later
ticks and must not turn one transient mailbox error into host shutdown. It
should run `DeliverPending` and `IngestPending`, then ingest pending acks for
the registered recipient identities. Keep one-shot `WithSession` pumping as a
fallback for commands used without `serve`.

## Test that fails today

Add a native subprocess test that starts `harnessing serve`, attaches a real
client, sends a message, detaches that client, waits for a host tick, then
attaches an observer and reads the message. It should require `PublishedAt` and
`ProcessedAt` after the sender has detached. On the current implementation the
message remains queued because `Serve` never invokes `Deliverer`; the test
fails before any `StartRun` behavior is involved.

This finding does not move item 6 or change its acceptance table. It identifies
serve completeness work to sequence separately from H101-144's live StartRun
implementation.
