# H101-112 — Phase 2 exit item 2 discharge ruling (2026-09-21)

Kelly QA ruling on `17af7af` (`TestRun_Phase2CycleRecoversAfterKilledCLI` amended per
[`h101-104-cross-target-ack-survival.md`](h101-104-cross-target-ack-survival.md)).

**CI cited:** GHA run `35614603811` — `build-test-lint` (ubuntu/linux-amd64) and
`darwin-arm64` (macos-14) both success.

**Kelly re-verified locally:** targeted crash test pass; `go test ./...` zero failures;
`GetSnapshot` appears only in a comment (lines 102–105), not in assertion code; ack
block uses `run([]string{"message", ...})` with spec pass criteria (Acknowledged/by
reviewer1, Task t1, Queued/Published/Processed RFC3339Nano via `outputField`).

---

## 1. Item 2 verdict

**SATISFIED** on ADR 0001 in-scope targets (`linux/amd64`, `darwin/arm64`).

**Unconditional on those targets** — as promised in the H101-104 spec. The
host-side ack gap is closed; both native CI jobs prove crash-reopen through
`harnessing task` and `harnessing message` on the same workspace.

---

## 2. Windows / H101-103

**Yes** — the Windows exclusion sub-check (H101-103, compile-only guard) is the
**only limit left on item 2** when read across the full platform matrix. It is a
separate out-of-scope assertion, not partial satisfaction on darwin or linux.

---

## 3. Phase 2 exit

**YES — Phase 2 exit criteria are met.**

| # | Status |
| --- | --- |
| 1 | **SATISFIED** (linux+darwin native CI + manifest history) |
| 2 | **SATISFIED** on ADR targets (this ruling) |
| 3 | **SATISFIED WITH LIMIT** (H101-107; G1–G4, G6–G7 documented) |
| 4–8 | **SATISFIED** |
| 9 | **N/A** — policy; engine tests + adaptercontract + §2 subprocess both present |

Documented item 3 limits and non-exit follow-ups (A2 fault injection, Angela product
calls, H101-30 disclaimer) do **not** block phase exit per DoD §Explicitly not Phase 2
exit.

**Remaining work is Phase 3 and product follow-ons — not open Phase 2 exit items.**

Authored by Kelly (QA).
