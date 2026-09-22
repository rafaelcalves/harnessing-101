# H101-207 — Phase 3 item 5 discharge ruling (2026-09-22)

Kelly QA. Authority: H101-193/H101-199 bar; Kevin H101-194 at `e1a2e4b`; god
verification relay `2026-09-22T12-21-14-610Z-0e98c2`.

---

## (1) Is Phase 3 item 5 satisfied?

**Yes — SATISFIED WITH LIMIT at `e1a2e4b`.**

---

## (2) Three god-flagged review points

### Participated-line suppression (I5 arithmetic)

**Legitimate — does not weaken I5.**

In `serve_release_both_channels` only, the fixture suppresses the stdout line
`harnessing fixture: participated`. **Startup survival** is still proved via
`HARNESSING_FIXTURE_SYNC_FILE` containing `participated\npid=<N>\n` before release
(B5′ class — same ordering H101-199 requires). That line is item 1 Layer A
participation evidence in other modes; item 5 does not require it on stdout.

Suppression isolates the two **named literals** as the only fixture-authored
channel content (besides the harnessing disclosure banner the test accounts for in
offset math). I5 proves **offset-ordered resume** through `run-output -after-offset`
at a real journal chunk boundary — not presence of the participate banner.

**Recorded note (not a product limit):** I5 boundary uses known 19-byte literal
lengths + disclosure length; stdout/stderr append order between the two literals
is not assumed — test accepts either literal alone after the boundary.

### Bounded `events` batch vs live stream (I6)

**Bounded batch satisfies I6.**

I6 is a **UI-08 negative**: raw run-output literals must not appear on the shipped
**state-event read path**. `harnessing events` with a bounded Subscribe drain proves
that separation on the **actual product surface**. I6 does **not** require live
event streaming — that is the separate **follow/cancel** obligation (item 5 D4),
explicitly recorded as a LIMIT below.

### Reader cancel LIMIT

**Yes — this is the limit Kelly pre-authorised in H101-193.**

`FollowOutput` cancel without stopping the run is **not** proved; journal `Follow`
remains stub. **Cost:** Phase 3 exit does not claim detached readers can cancel
follow without `StopRun`; full UI-08 reader-lifecycle matrix deferred. **Does not**
block I1–I6 discharge.

---

## (3) I-row coverage at `e1a2e4b`

| Row | Verdict | Evidence |
| --- | --- | --- |
| I1 | **SATISFIED** | `TestCLI_ServeOwnedGracefulCaptureCompletesBothChannels` — `run-output` exit 0 |
| I2 | **SATISFIED** | `Capture status: complete` after bounded `run-output` poll (H101-223: query-triggered, not eager-ms) |
| I3 | **SATISFIED** | Exact `fixture-out-stdout\n` on stdout channel |
| I4 | **SATISFIED** | Exact `fixture-out-stderr\n` on stderr channel |
| I5 | **SATISFIED** | Second attached `run-output -after-offset` returns only later chunk (not both literals) |
| I6 | **SATISFIED** | Attached `events` output contains neither literal |
| D5 real-not-fake | **SATISFIED** | Real binary; `serve` owner; `start-run` + `run-output` + `events` CLI only — no `fakeSupervisor`, no `host.Open` in test |
| H101-195 fast-exit | **Preserved** | `TestSupervisor_FastExitIsSpawnFailed` PASS unmodified |
| R8 cross-target | **Pending CI** | Native test runs `linux` + `darwin`; Kelly re-verdict on ubuntu + macos-14 green at this commit |

Named test: `TestCLI_ServeOwnedGracefulCaptureCompletesBothChannels` (`e1a2e4b`).
Kelly local: `make build test vet lint fmt` green; test PASS ×2 under `-race`.

---

## (4) Owner sentence — what item 5 guarantees

After a **real supervised child** exits gracefully under a **continuing `harnessing
serve` host**, both stdout and stderr are **durably captured** with
`Capture status: complete`, readable through **`harnessing run-output`** on a fresh
attached reader, with **honest offset resume**, and **raw output bytes do not
appear** on the shipped **`harnessing events`** path.

## What item 5 does **not** guarantee

- Reader **cancel** without stopping the run (`FollowOutput`/journal `Follow` stub —
  item 5 LIMIT)
- Retention / `CursorExpired` / long-horizon follow
- `StopRun` tree termination (item 2)
- Crash-prefix / interrupted capture (item 4 — already discharged separately)
- Item 6 background-hosting / detach survival
- `-byte-limit` behaviour (shipped but not in I-rows; no discharge claim)

---

Authored by Kelly (QA), H101-207.
