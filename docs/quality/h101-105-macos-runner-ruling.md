# H101-105 — macOS CI runner ruling (2026-09-21)

Kelly QA ruling on Kevin's H101-102 delivery (`f6075fa` workflow, `638bbcc`
manifest appendix). God verified the workflow file (triggers, pinned action SHAs, Go version,
no hostname or operator path in output) — not the test runs. Workflow
unpushed at ruling time.

Contract: [H101-55](definition-of-done.md#h101-55--platform-ruling-vs-item-2-2026-09-19),
[darwin manifest](darwin-arm64-milestone-manifest-2026-09-21.md).

---

## Question 1 — Does the runner discharge darwin/arm64 re-proof on every change?

**Verdict: SATISFIED** — upgraded 2026-09-21 after observed green
`darwin-arm64` job (GHA run `35598625946`, commit `2534afb`, `macos-14`).

### What H101-55 asked for

Darwin/arm64 native evidence on every change — either a **macOS CI runner**
or a **recorded milestone manifest**. The manifest discharged the one-shot
path; the runner is the ongoing path.

### What landed

- `darwin-arm64` job on `macos-14` (Apple silicon, arm64), every push/PR.
- Three test filters are **copy-exact** from the manifest — subprocess §2
  walkthrough, crash-reopen, H101-20 lock recovery.
- Also: native `go build`, full `go test ./...`, `go vet`, gofmt on darwin.
- Platform recording step prints `sw_vers`, kernel arch, Go version — no
  hostname or operator paths (H101-100).

### What was observed

| Evidence | Status |
| --- | --- |
| Workflow shape and commands | **Present** in repo (`f6075fa`, local) |
| Commands pass on native darwin/arm64 | **Yes** — Kevin reproduced every job step locally and watched each pass; god did **not** re-run them, so this rests on Kevin's single disclosed report |
| First green run inside GitHub Actions | **Observed** — run `35598625946` on `2534afb` (`macos-14`); `darwin-arm64` job green (god session) |

### Reasoning

- **Structural delivery:** satisfied. The job matches H101-55's runner
  requirement: same contract as the manifest, automated on every change,
  correct architecture label.
- **"On every change" discharge:** **satisfied** as of run `35598625946`.
  The open condition from the initial ruling is closed.

### Out of scope (not a defect)

- Item 3 `adaptercontract` darwin gap — unchanged; H101-104 ack test — not
  Kevin's card.

---

## Question 2 — Does the runner supersede the manual manifest?

**Verdict: Both stand — split roles.** Runner **supersedes manifest as live
standing evidence** once Question 1's open condition closes. Manifest is
**not deleted.**

| Role | Manifest | Runner |
| --- | --- | --- |
| **Historical** | Permanent: first disclosed native run that satisfied H101-55 before a runner existed (`02b1407`, redacted H101-100) | N/A |
| **Standing evidence (now)** | Archive / baseline only | **Authoritative** for darwin items 1–2 on every change |

### Reasoning

- Deleting the manifest would erase the audit trail of *how* darwin was first
  discharged and why redaction was applied.
- Superseding **functionally** is correct: ongoing proof should come from CI,
  not re-reading a one-date document.
- Kevin was right not to weaken the manifest; Kelly's call is **retain +
  supersede in role**, not replace in file.

---

## Summary for god

| Question | Verdict |
| --- | --- |
| 1. Runner discharges darwin re-proof on every change? | **Satisfied** — run `35598625946` green on `2534afb` |
| 2. Runner vs manifest? | **Both stand** — manifest historical; runner is live standing evidence |
