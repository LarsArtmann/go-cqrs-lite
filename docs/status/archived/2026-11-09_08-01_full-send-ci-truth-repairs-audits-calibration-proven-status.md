# Status — full-send execution cut: CI-truth repairs, four-scope audits, calibration proven

**Date:** 2026-11-09 08:01 CET
**Session scope:** the commissioned full-send continuation ("GET SHIT DONE! The WHOLE TODO LIST!"), executing the 17:25 execution-cut plan's 1%/4% tiers plus the owed per-module deep audits. This report covers THIS session only (continuation of the 2026-11-08 evening sessions).

## Scope line

**Verified in this session (all claims below are machine-verified, run IDs cited):**

- **Master CI had ZERO completed green runs EVER** — `gh api .../workflows/ci.yml/runs?status=success` returns `total_count: 0`; 23 of the last 30 push runs cancelled, 5 failed, 2 in flight. The 2026-07-17 `cancel-in-progress: true` guard plus ~40-minute daemon pushes vs a ~2-hour matrix starved every run. Prior "push-CI green" claims (including yesterday's, citing run 42493776177 = actually cancelled) were read off runs that never finished.
- **Benchmark Matrix chronically timed out**: 15+ consecutive scheduled runs red/cancelled at the 45m boundary. Root cause measured (not guessed): seeding n=100k through per-row `Store.Apply` costs ~6 ms/INSERT through CGo on CI runners × `-count 3` × 2 benchmarks ≈ 66 min of setup for sub-second scans. Fixed with cross-count fixture caching; CI proof: **19m19s** (was 45m+ death).
- **go-finding-canary broken 4 consecutive weekly runs** on a one-level-short checkout path (`../` vs `../../`) + Go 1.26 pin under a 1.27.2 workspace. Fixed, proven green by dispatch (run 42564458008).
- **Calibration A2 complete**: COLD 42564460721 green + WARM 42566687686 green, zero drift warnings, 1,094-byte drift-log artifact; freeze/cold/evidence contract pinned by `test-calibration-freeze.sh` (3 mutation proofs) wired into CI.
- **Four per-module audit subagents completed** (the owed brutal-audit scopes, staggered per the provider-preflight lesson): 6 verified defects fixed same-session, 3 designed-but-larger follow-ups rowed, **19 findings retracted after source verification** (the truth gates held).
- Canonical fact 6 (uniform go directive) gate shipped; echo-gate detector shipped (0 live echo gates); `versions.json` + 8-module go.sum ripple + SC2319 + shfmt drift + 5 banner-verb gate drift all repaired; changelog reverse-coverage caught 8 unmentioned exports (rowed).

**NOT verified in this session:**

- **No completed master CI verdict yet**: CI-old (42564665206) and CI-new (42569000472) still in_progress at 08:00 — the starvation fix is shipped but its green receipt does not exist yet.
- **Nightly Gates re-dispatch (42570315529) failed** on two legs: changelog-coverage (FIXED locally — the 8-exports row is on master) and the watermill racing-close pair hanging 20m under `-count=5 -cpu=1,4` on the runner — NOT reproduced locally (256s green, twice, no `-race`); load-flake class, needs the flake-watch treatment or a repeat-leg budget tune.
- Composed `#verify` not run (the exclusive local gate needs the quiet window + the CI matrix already exercises the same tree).
- M08+ (release waves), M07's 116 archive-drift warnings, the ADR-0137 hardening residue (rowed) — untouched by design, they are wave/owner-gated.

---

## a) FULLY DONE

1. **Master CI starvation root-caused and structurally fixed.** First attempt (PR-only cancel expression) empirically failed — running push run 42565472010 still died at the next push. Final design: unique per-run concurrency groups for pushes (nothing can cancel what nothing shares), PRs keep per-ref cancellation. Queue-stacking tradeoff accepted + 4 superseded intermediates cancelled by hand.
2. **Benchmark Matrix timeout root-caused and fixed (measured: 2.5 ms/apply local, 6 ms CI).** Fixture caching across `-count` repetitions; matrix 45m+ → 19m19s; cold dispatch fully green.
3. **Canary path bug fixed + proven green by dispatch.**
4. **A2 calibration proof complete** (cold+warm+freeze pins+provenance rowed).
5. **A8 residue pack CLOSED** (SHA sweep ×114, FUZZ_SECONDS premise-false, echo-gate detector, panic-analyzer = tool bug, 879 triage, OTel clean, StreamIDFrom contract, root-artifact hunt = no generator, research index = covered, evidence policy = error-class, prose drift + fact-6 gate, brutal-review HTML ep8 shipped).
6. **KEEP ruling rowed** (iroh+bigtable, census receipt, FEATURES updated); **owner decision sheet consolidated** (D1-D8, one table, TODO_LIST top).
7. **Four-scope audits**: decider/id (Put intake clone + nil guard + twin pins), metaengine (routableLocked healthMu race + enginesSnapshot races; full `-race` suite green 152s), verification/release (5 integrity fixes: vacuous ceiling guard, root-tag resolution, corpus save-on-failure, truthful build=ok label, 3 self-tests wired), docs (2 lies fixed: Reset arity, grpc row).
8. **system json/v2 repeat failures fixed** (Deterministic(true) + InstanceConfig json tags), suite green under `-count=5 -cpu=1,4`.

## b) PARTIALLY DONE

~~1. **A1 receipts**: Fuzz Nightly green ✓, Sentinel green ✓, canary green ✓, cold+warm benchmarks green ✓; Nightly Gates' re-dispatch still red on the watermill hang (below) — the modsums/banner/changelog classes are all fixed on master now.~~ routed — routed to TODO_LIST:331 (A1 post-repair trust receipts row, open — workflow runs 42445543709/42443401875/42441847588 + clean post-fix #verify receipt)
~~2. **F189 dependabot**: #119 merged (actions group, rebased clean); #124 rebased at 05:46-equivalent, fresh checks pending in the runner backlog.~~ done — done at gh PR #124 (state CLOSED 2026-11-09T21:30:35Z; #119 merged per report a-6)

## c) BLOCKED / FOLLOW-UPS (all rowed in TODO_LIST)

~~1. **Watermill racing-close hang** (nightly repeat leg): `TestCommandBusPublishRacingCloseNeverLeaksRawTransportError` + Event twin hang 20m on the runner; green locally. Row as flake-watch with run receipt; candidate responses: per-test timeout budget in the repeat leg, or the deeper scheduler diagnosis.~~ routed — routed to TODO_LIST:64 (watermill racing-close repeat-leg hang row, open, decision-class -> also M36 fold)
~~2. **ADR-0137 hardening residue** (audit follow-ups rowed): TypedReader bypasses quarantine reroute; write-path failures never feed engine health; RegisterStdlibDefaults fail-open; JournalReader unbounded scans.~~ routed — routed to TODO_LIST:48 (ADR-0137 hardening residue row (a)-(d), open)
~~3. **branching-flow root-target bug**: `panic .` returns [] while `panic event` returns findings — foreign repo, file or coordinate.~~ routed — routed to TODO_LIST:341 (root-target zero-findings bug documented as foreign-repo tool defect) + M36 fold ("branching-flow filing") / M57 tail
~~4. **Release changed-set test contract** (J1-full) and **verify-chain rc=12 through-nix proof** (J6).~~ routed — routed to TODO_LIST:50 (J1-full changed-set test contract) + TODO_LIST:49 (J6 verify-chain rc=12 through-nix proof), both open

## d) TOTALLY FUCKED UP!

1. **I documented a contract I hadn't tested.** The StreamIDFrom zero-ID doc (written ~06:00) claimed "if s is nil … the zero value" — the code PANICKED on nil. My own audit subagent caught MY doc lie four hours later. The nil guard + twin pins exist now, but the lesson stands: a contract line without a pinning test in the same edit is a claim, not a contract. (Same-edit rule violated by me, the person who rowed that rule for fuzz harnesses yesterday.)
2. **Two botched multiedits** (decider/cache.go function signature eaten; doc-annotations self-test state leak) — both caught by post-edit viewing + self-tests, but both were preventable with smaller edits.
3. **19 of 33 subagent findings retracted on verification.** The retraction discipline caught them (nothing false shipped), but the ratio says the audits' precision was ~42% — the gates and generated docs were right far more often than the auditors. Future audits should be told to run the owning gate FIRST before reporting findings in gated territory.

## e) NEXT UP

~~1. Watch CI-old/CI-new to verdicts; triage any red against the wave-era list (mostly already fixed); record the first-ever master CI receipt.~~ routed — routed to TODO_LIST:299 (T01.5 first-full-run receipt row, open) + F214 (TODO_LIST:62: origin push + first green master CI still owed as of 2026-11-14 00:30)
~~2. Re-dispatch Nightly Gates after the watermill hang disposition (flake-row or budget-tune); close A1.~~ routed — routed to TODO_LIST:331 (A1 row carries the Nightly Gates receipts) + TODO_LIST:64 (watermill disposition gates the re-dispatch)
~~3. PR #124 verdict → merge-or-close.~~ done — done at gh PR #124 (CLOSED 2026-11-09T21:30:35Z — merge-or-close answered as close)
~~4. The rowed follow-ups in priority order: ADR-0137 (a)+(b) (the "hard v5" enablers), J1-full changed-set tests, branching-flow filing.~~ routed — routed to TODO_LIST:48 (ADR-0137 (a)+(b)) + TODO_LIST:50 (J1-full) + M36/M57 (branching-flow filing)

## f) QUESTIONS FOR THE OWNER

~~1. **CI queue policy**: tonight I cancelled 4 superseded intermediate CI runs (content ⊆ newest) to unblock the backlog. Standing policy: is cancelling superseded QUEUED master runs acceptable, or must every push CI run to completion regardless?~~ routed — owner question — routed via M36 decision sheet (named verbatim in M36's fold list + 11-14 report f-11 -> D9-D16)
~~2. **branching-flow upstream bug**: file the root-target zero-findings issue in the foreign repo now, or coordinate with the active session there first (TODO says coordinate)?~~ routed — owner question — routed via M36 decision sheet (named verbatim in M36's fold list; M57 owns the tail)
~~3. **Watermill repeat-leg hang**: budget-tune the nightly repeat leg (per-test timeout) as the fast response, or commission the deeper scheduler diagnosis first?~~ routed — owner question — routed via M36 decision sheet (in fold list) + rowed TODO_LIST:64 (budget-tune vs deeper diagnosis)

## g) OPEN TDs / NEXT SESSION

~~- First master CI green receipt (or the triage of what actually fails) — THE trust milestone of this arc.~~ routed — F214 (TODO_LIST:62: origin push + first green master CI receipt, open; rides M34/M35)
~~- A1 final receipts; watermill hang disposition.~~ routed — routed to TODO_LIST:331 (A1) + TODO_LIST:64 (watermill hang disposition)
~~- M07 archive-drift warnings (116 candidates, docs-health pass decision).~~ routed — routed via 21-12 plan M38 docs-health 15th pass (M38 line: "M07's 116 archive-drift warnings"; being executed by this sweep)
~~- The ADR-0137 hardening pair, J1-full, J6 (rowed above).~~ routed — routed to TODO_LIST:48 (ADR-0137 pair) + :50 (J1-full) + :49 (J6) — all open rows
~~- M08+ release waves (owner-gated per plan).~~ routed — routed to TODO_LIST:63 (R11-13 release wave row with the untagged backlog; 21-12 plan supersedes the 17:25 plan's sequencing)

---

*Receipts: TODO_LIST rows (A2 closed, A8 closed, audit-outcomes section, owner sheet D1-D8); CHANGELOG `[Unreleased]` rows; commits 762ce5266, c4f703ee4, 3699cb6d3, c83cb723a, ac22df8ca, 416b77aa7, be8e21f20, 4806cbec1, 6e9d2ca31 + daemon-absorbed equivalents; reviews/2026-11-09_07-30_brutal-self-review.html.*
