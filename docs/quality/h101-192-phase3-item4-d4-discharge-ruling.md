# H101-192 — Item 4 D4 discharge and item 4 exit ruling (2026-09-22)

Kelly QA acceptance on Claudio's H101-191 at `b7d1131`.

**Verified independently:** `make build test vet lint fmt` green (darwin/arm64) on
retry; `TestCLI_RunOutputCrashPrefixIsHonest` PASS ×3 under `-race`. No `engine`,
`host.Open`, or `statestore` references in `d4_test.go`. (First local `make test`
hit a flaky `TestOpen_RecoversAfterOwnerCrash`; passed on immediate retry — not
attributed to this commit.)

God's hold on stale verification report noted; Kelly re-ran final tree locally.

---

## (1) Is D4 satisfied?

**Yes — SATISFIED at `b7d1131`** on in-scope targets (local). Row D4: crash output
prefix honest — exact persisted bytes, interrupted capture, no fabricated tail or
invented byte counts.

---

## (2) Is the coverage claim accurate?

**Yes.** Not overstated.

| Claim | Verdict | Evidence |
| --- | --- | --- |
| C1 exit 0 | **Present** | `d4_test.go` run-output subprocess |
| C2 exact `fixture-out-1\n` | **Present** | positive contains |
| C3 interrupted/unknown, not complete | **Present** | `Capture status: interrupted` or `unknown`; no `complete` path in test output format for crash case |
| C4 no `fixture-out-2\n` | **Present** | negative contains |
| C5 no byte count/range claim | **Present** | no `Bytes:` / `Range:` |
| G2 prefix durability | **Covered** | C2 |
| G3 fabrication | **Covered** | C4 — named literal |
| G4 complete-after-crash | **Covered** | C3 |
| G5 invented counts | **Covered** | C5 |
| G6 zero-chunk fabrication | **Avoided** | producer always emits prefix |
| G9 continuing-host reconcile | **Avoided** | post-crash fresh `run-output` subprocess |
| G1/G7/G8/G10/G11 | **Avoided** or CI-only per claim | accurate |

**R8:** CI-only limit until ubuntu + macos-14 green on `b7d1131`.

---

## (3) Does the minimal journal respect Stanley's H101-190 boundary?

**Yes.**

| Boundary | Verdict |
| --- | --- |
| Shared OutputJournal foundation, not second ledger | **Yes** — `internal/adapters/journal/journal.go`; supervisor `journalWriter` |
| Shipped `harnessing run-output` read path | **Yes** — `run_output.go` |
| No fixture-only persisted copy bypass | **Yes** — stdout streamed via supervisor into journal |
| Item 5 **NOT** discharged | **Yes** — no graceful complete drain, follow/cancel, retention proofs; commit message and H101-190 acceptance consequence align |
| `Finish(Interrupted)` on RecoveryRequired reopen | **Yes** — `host.Open` |

---

## (4) Is Phase 3 item 4 satisfied?

**Yes — SATISFIED WITH LIMIT at `b7d1131`.**

### Row standing (amended)

| Row | Verdict |
| --- | --- |
| E1–E5 | **SATISFIED** (unchanged from H101-183/H101-188 chain) |
| D1 duplicate spawn | **Partial — LIMIT** | Agent-scoped refuse proved; same-`runID` duplicate not in scope |
| D2 stuck `Running`, dead child | **SATISFIED** `b764691` |
| D3 surface `RecoveryRequired` | **SATISFIED** | Starting-at-reopen (H101-183) + Running-at-reopen (H101-188) |
| D4 output gaps | **SATISFIED** `b7d1131` |
| D5 unclean crash | **SATISFIED** |
| D6 CLI not host query | **SATISFIED** |

### What item 4 now guarantees

After unclean controller exit during run supervision: next `harnessing` open
reconciles honestly (`RecoveryRequired` through CLI for Starting and Running
dead-child cases); same-agent automatic respawn refused; persisted run-output
prefix readable through `harnessing run-output` without fabricated tail bytes or
false complete capture.

### What item 4 does **not** guarantee

- Same-`runID` duplicate-spawn block (D1 limit)
- Live supervision, PID adoption, tree termination, budgets through crash
- Full output journal (item 5): graceful complete capture, follow/cancel,
  retention/CursorExpired, reader-cancel semantics
- Admin resolution / restart UI (H101-173)
- Windows / out-of-scope targets

---

## (5) Next Phase 3 item

**Item 5** — shares this journal foundation. Criteria decision table exists in
`phase3-exit-criteria.md`; **no Kelly acceptance spec yet** (bar-before-card
discipline applies before engineering dispatch).

---

Authored by Kelly (QA), H101-192.
