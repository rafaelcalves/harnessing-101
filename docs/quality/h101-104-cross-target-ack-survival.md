# H101-104 — cross-target handoff-acknowledgement survival (2026-09-21)

Kelly QA spec for the last open gap on Phase 2 exit **item 2**. Kevin implements;
Kelly does not edit source in this card.

**Contract sources:** item 2 in [`definition-of-done.md`](definition-of-done.md),
ADR 0001 H101-50 (`linux/amd64`, `darwin/arm64`), H101-55 platform ruling,
[`h101-105-macos-runner-ruling.md`](h101-105-macos-runner-ruling.md) (darwin native
CI now authoritative), existing crash test
`cmd/harnessing/phase2_crash_reopen_test.go` (`TestRun_Phase2CycleRecoversAfterKilledCLI`).

**Gap this closes:** crash-reopen already proves task/result state through the shipped
`harnessing` command and proves acknowledged-handoff durability only through an
in-process `host.GetSnapshot` query. Item 2 requires the same product surface the §2
walkthrough uses (`harnessing message`) on **each** in-scope native target.

---

## Decision table (name before any run)

Read this table first. A failing CI run maps to **one** row; do not pick the cheaper
branch by default (criteria-owner lesson from H101-110).

| # | Observation after `SIGKILL` holder + CLI reopen | Defect class | Fix |
| --- | --- | --- | --- |
| D1 | `harnessing task t1` loses `AwaitingReview` / `ResultID` | **Product** | Persistence or CLI task render — same class as today's test lines 91–93 |
| D2 | `harnessing message m1` exits non-zero (`NotFound`, `IOFailure`, etc.) while task state from D1 is correct | **Product** | Message record or `GetMessage` path did not survive crash |
| D3 | `message m1` exits 0 but `Acknowledged: (absent)` (or missing `by <recipient>`) | **Product** | Ack facts not persisted or not rendered — **this is the H101-104 hole**; do not patch with `host.GetSnapshot` |
| D4 | `message m1` shows ack but `Task: (absent)` or wrong task ID | **Product** | Task link on handoff lost |
| D5 | Delivery facts show `(absent)` for Queued/Published/Processed after crash | **Product** | Delivery-fact persistence/render regression (walkthrough already locks these as present) |
| D6 | Ack recipient or timestamp wrong (`by` ≠ pre-crash acker; timestamp not parseable RFC3339Nano) | **Product** | Corrupt or partial ack record |
| D7 | `host.Open` succeeds while holder alive (Busy negative control) | **Product** if lock broken; **Test** if kill/reap race or wrong workspace | Re-read H101-20 ruling: kernel teardown poll is honest; flaky kill → fix test timing only |
| D8 | Test asserts ack only via `host.GetSnapshot` / `Capabilities` | **Test** | Violates this spec — replace with `harnessing message` per §Pass criteria |
| D9 | Test never sends `ack` before kill, or kills after `Close` | **Test** | Not exercising acknowledged-handoff survival |
| D10 | Test passes on one CI target only because of `t.Skip` / platform probe misuse | **Test** | Both targets must run the same contract natively |
| D11 | `host.GetSnapshot` shows ack but CLI `message` does not (D3) | **Product** | CLI/session/query bug — **not** a test workaround |

**Default rule:** if D3 or D11 fires, the product lost ack visibility on the shipped
command surface. **Fix the product.** Removing the host assertion without adding the
CLI assertion is **not** a pass.

---

## Two targets

| Target | CI job | Evidence role |
| --- | --- | --- |
| `linux/amd64` | `build-test-lint` on `ubuntu-latest` (`go test ./...`) | Native crash+ack proof |
| `darwin/arm64` | `darwin-arm64` on `macos-14` (named step, manifest command unchanged) | Native crash+ack proof |

**Not in scope:** cross-machine workspace handoff (linux workspace opened on darwin),
Windows (exclusion smoke already ruled H101-103), NFS/network filesystems (unsupported
per boundaries).

---

## What must survive

After an unclean exit (`SIGKILL` on the process holding the workspace lock, no
`Close`), a subsequent `harnessing` invocation on the **same** workspace must show:

1. **Task cycle state** (already asserted): task `t1` at `AwaitingReview` with
   `ResultID: res1`.
2. **Acknowledged task-linked handoff** on message `m1` (the gap):
   - Recipient acknowledged before kill (`harnessing ack` as today).
   - After reopen, `harnessing message m1` shows acknowledgement **and** task link
     **and** delivery facts — same product surface as
     `TestCLI_Phase2ProductWalkthroughSubprocess` post-reopen block (lines 148–179
     of `phase2_subprocess_walkthrough_test.go`), adapted to this fixture's IDs
     (`reviewer1`, `m1`, `t1`).

**What counts as acknowledgement surviving:** CLI stdout from `harnessing message m1`
contains:

- `Acknowledged:` line with `by reviewer1` (not `(absent)`).
- `Task:        t1`.
- `Queued:`, `Published:`, `Processed:` each with a parseable RFC3339Nano timestamp
  (not `(absent)`).

**What does not count:**

- `host.GetSnapshot`, `Capabilities`, or any in-test `host` import used **only** to
  assert ack survival (Busy negative control and platform probe remain allowed per
  existing test).
- `harnessing messages` pending list alone (ack clears pending — use `message` for
  durable ack facts).
- Substring match on stderr disclosure lines.

---

## Implementation shape (Kevin)

**Amend** `TestRun_Phase2CycleRecoversAfterKilledCLI` — do not add a parallel test
unless the amend is unreadable.

1. Keep existing setup: register, create, transition, send, **ack**, report,
   `hold` subprocess, Busy negative control, `SIGKILL`, reopen poll on `task`.
2. **Replace** the `host.GetSnapshot` ack block (current lines 95–109) with:
   `run([]string{"message", "-workspace", dir, "-workspace-id", wsID, "m1"}, ...)`
   and assert per §Pass criteria.
3. Run unchanged on both CI targets (no new workflow steps required if the test name
   stays the same — darwin job already filters it; ubuntu runs it via `go test ./...`).

**Verify (quote tails):**

```bash
go test -count=1 -v ./cmd/harnessing -run TestRun_Phase2CycleRecoversAfterKilledCLI
go test ./...
```

---

## What a failing run looks like

| Symptom | Likely row | User-visible signal |
| --- | --- | --- |
| Task OK, `message m1` shows `Acknowledged: (absent)` | D3 / D11 | Crash wiped ack or CLI hides it — **item 2 stays partial** |
| `message m1`: `harnessing message: NotFound` | D2 | Message index lost |
| Task reverted to `Doing`, result gone | D1 | Pre-existing crash gap regressed |
| Pass locally, skip on CI target | D10 | Platform gate wrong |
| Green only because host assert kept | D8 | Spec not implemented |

---

## Item 2 verdict after this test passes

| Question | Answer |
| --- | --- |
| Discharges cross-target ack gap? | **Yes** — both in-scope targets prove ack through `harnessing message` after crash |
| Item 2 overall | **SATISFIED** on ADR 0001 targets (`linux/amd64`, `darwin/arm64`) |
| Unconditional? | **Yes** for in-scope targets. Windows remains **satisfied with limit** via H101-103 (compile-only exclusion) — separate sub-check, not a limit on this test |
| Still partial? | **No**, once green on both native CI jobs — no manifest-only ack proof remains |

Item 2 does **not** need "satisfied with limit" solely because crash driving uses
`run()` + subprocess `hold` (already ruled acceptable per H101-57) or because exact
CLI label punctuation is walkthrough-locked (presentation honesty, not a new limit
for item 2).

---

## Boundaries

- Spec only — no engine, CLI, or workflow edits in Kelly's session.
- Does not discharge item 3, H101-55 manifest retention, or A2 fault injection.
- Does not require a second adapter or mailbox-only path.

Authored by Kelly (QA).
