# Darwin/arm64 milestone evidence manifest (H101-55 / H101-98)

**Recorded:** 2026-09-21  
**Recorder:** Kelly QA (kelly-qa-mu802q7c)  
**Purpose:** Disclosed native evidence for ADR 0001 in-scope target `darwin/arm64` per H101-55 — not an undisclosed laptop run.

This manifest is **weaker than CI**: one machine, one date, manual execution. It discharges the “recorded milestone manifest” path until a **macOS CI runner** re-runs the same tests on every change.

**Redaction (H101-100):** The machine hostname in `uname -a` and the operator home directory in the module root path were replaced with placeholders before public commit — they carry no evidentiary weight for platform qualification and this repository is public.

---

## Environment

| Field | Value |
| --- | --- |
| **Commit** | `02b140768176991c9128772589cbbed7ec2d9691` (clean tree; no local modifications at run time) |
| **Date (UTC)** | 2026-09-21 |
| **Product** | macOS 27.0 (Build 26A428) |
| **Kernel** | Darwin 27.0.0; `Darwin <redacted-hostname> 27.0.0 Darwin Kernel Version 27.0.0: Tue Aug 11 21:06:24 PDT 2026; root:xnu-13432.1.9~1/RELEASE_ARM64_T6030 arm64` |
| **Go toolchain** | `go version go1.27.1 darwin/arm64` |
| **Module root** | `<repository-root>` (commands run from module root; paths below are relative) |

### `sw_vers`

```
ProductName:		macOS
ProductVersion:		27.0
BuildVersion:		26A428
```

### `uname -a`

```
Darwin <redacted-hostname> 27.0.0 Darwin Kernel Version 27.0.0: Tue Aug 11 21:06:24 PDT 2026; root:xnu-13432.1.9~1/RELEASE_ARM64_T6030 arm64
```

---

## Commands and outputs

### 1. Item 1 — §2 subprocess product walkthrough

**Command:**

```bash
go test -count=1 -v ./cmd/harnessing -run TestCLI_Phase2ProductWalkthroughSubprocess
```

**Output:**

```
=== RUN   TestCLI_Phase2ProductWalkthroughSubprocess
--- PASS: TestCLI_Phase2ProductWalkthroughSubprocess (1.89s)
PASS
ok  	github.com/rafaelcalves/harnessing-101/cmd/harnessing	2.175s
```

**What it proves:** Full Phase 2 §2 minimum product cycle through compiled `harnessing` subprocess only (register, task, handoff, blocker, report, reject, accept, reopen, status queries).

---

### 2. Item 2 — crash reopen with persisted product state

**Command:**

```bash
go test -count=1 -v ./cmd/harnessing -run TestRun_Phase2CycleRecoversAfterKilledCLI
```

**Output:**

```
=== RUN   TestRun_Phase2CycleRecoversAfterKilledCLI
--- PASS: TestRun_Phase2CycleRecoversAfterKilledCLI (0.13s)
PASS
ok  	github.com/rafaelcalves/harnessing-101/cmd/harnessing	2.175s
```

**What it proves:** CLI-driven workspace with `AwaitingReview` task + `res1` survives `SIGKILL` of a `hold` subprocess; subsequent `harnessing task` recovers state without resending work.

---

### 3. Item 2 prerequisite — lock recovery after owner crash (H101-20)

**Command:**

```bash
go test -count=1 -v ./internal/adapters/statestore/ -run 'Recover|SecondOpen'
```

**Output:**

```
=== RUN   TestOpen_SecondOpenIsBusy
--- PASS: TestOpen_SecondOpenIsBusy (0.00s)
=== RUN   TestOpen_RecoversAfterOwnerCrash
--- PASS: TestOpen_RecoversAfterOwnerCrash (0.01s)
PASS
ok  	github.com/rafaelcalves/harnessing-101/internal/adapters/statestore	0.173s
```

**What it proves:** `flock` lock recovery after simulated owner crash on this darwin/arm64 host (H101-20).

---

## Kelly ruling — what this manifest discharges

| Exit item | Darwin/arm64 effect | Notes |
| --- | --- | --- |
| **1** CLI §2 walkthrough | **Darwin half satisfied** for native evidence | Linux/amd64 already covered by CI. Platform-agnostic gaps (if any remain in DoD) are unchanged. |
| **2** Restart after crash | **Darwin native crash-reopen + lock prerequisite satisfied** | Product-state crash test and H101-20 lock tests pass on this host. |
| **2** (overall) | **Still partially satisfied** | Cross-target gaps remain: e.g. handoff ack survival proof still host-side on **both** targets per prior rulings — this manifest does not close those. |
| **3** Adapter contract | **Not discharged** | `adaptercontract` must also run natively on darwin (this manifest did not include it; stage 1 was verified locally but not recorded here — add in a follow-up manifest entry or macOS CI). |

### Strength of evidence

- **This manifest:** one disclosed run on one `darwin/arm64` machine at one commit.
- **Stronger replacement:** macOS CI runner (e.g. `macos-latest` or pinned image) executing the same three command blocks on every PR to `main`, with runner image ID recorded alongside `ubuntu-latest` per ADR 0001.

---

## Related follow-ups (status at H101-100)

1. **macOS CI runner** — raised to human (spend/account decision).
2. **Windows exclusion smoke** — carded to Kevin as H101-101 (Phase 2 exit check, not support).

---

## H101-102 — macOS CI runner landed (2026-09-21)

The human approved a `macos-14` (Apple silicon, arm64) job in
`.github/workflows/ci.yml` (`darwin-arm64`, commit `f6075fa`),
alongside the existing `ubuntu-latest` job, on every push and pull
request. This is the "stronger replacement" this manifest names above.

**What the runner now proves, on every change, that this manifest could
only prove once:**

- The exact three commands recorded above — item 1's subprocess
  walkthrough, item 2's crash-reopen, and item 2's H101-20 lock
  prerequisite — run unmodified, natively, on `darwin/arm64`, on every
  push/PR, not on one machine on one date.
- `go build`, the full `go test ./...`, `go vet`, and the gofmt guard
  all run natively on `darwin/arm64` as well, catching a
  darwin-only build break or test failure this manifest's narrower
  scope would have missed.

**What it still does not prove:**

- **Item 3 (adapter contract) is still not discharged on darwin** —
  the new job does not add a darwin-specific `adaptercontract` step
  beyond what `go test ./...` already runs; that gap is unchanged from
  the table above and is not this card's scope to close.
- **Cross-target handoff-acknowledgement survival** (item 2, overall)
  remains open on both targets per prior rulings; this card was
  explicitly told not to attempt it (a separate card follows once the
  runner exists).
- **This manifest is not deleted or superseded by this commit alone** —
  that is Kelly's call, not something decided here. It stays as the
  disclosed record of the one run that first satisfied H101-55 before
  a runner existed.
- **The runner's own first real execution inside GitHub's macOS fleet
  was not directly observed when this section was written** — the
  environment used to write this change had an invalid GitHub
  credential (`gh auth status` failed), so every command above was
  verified by native local reproduction on a `darwin/arm64` machine
  (macOS 27.0, `go1.27.1 darwin/arm64`) rather than by watching the
  Actions run itself. See the commit message on `f6075fa` for the
  exact output watched. That distinction — local native reproduction
  of every step vs. a confirmed green run inside GitHub Actions
  itself — is reported to `god` as still open, not folded into this
  manifest's own evidence.
