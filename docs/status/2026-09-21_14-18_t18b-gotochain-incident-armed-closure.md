# Status Report #4: T18b — GOTOOLCHAIN Incident, Guarded Re-Arm, Autonomy Armed

**Date:** 2026-09-21 14:18 CEST (12:18 UTC)
**Scope:** continuation of the T18b session (reports #1–#3) — covers the post-ruling execution: the failed cost pass, its root cause, the guards added, and the re-armed autonomous chain. No unrelated work researched.
**Host:** shared 32-core; load 34/37 at report time (another storm); cost pass + closure re-armed and waiting.

---

## a) FULLY DONE

| # | What | Evidence |
|---|------|----------|
| A1 | **All prior deliverables intact** — 1.27.1 baseline (`a91e7cd90`, still last commit touching it), `--save` guard, p99 demotion, known-unstable suppression, per-suite `benchtime::count` GATE_SETS support, `quiet-window-run.sh` + self-test, `nightly-bench.sh` + units + flake apps | Committed tree verified this morning; fixtures green |
| A2 | **Q3 known-flaky annotation shipped and proven** — `UNSTABLE-KNOWN` suppression (MIN/matview until 2026-10-21) fixture-verified in all three directions: listed-flag → exit 0 (quiet-night green), unlisted +66% → REGRESSION exit 1, all-suppressed → exit 0 | Session fixtures against the real script |
| A3 | **Q2 nightly wired** — `scripts/nightly-bench.sh` (quiet-window-run → gate; never self-re-pins), systemd user units + README (`scripts/nightly/`), flake app `nightly-bench`; `bash -n` + `nix eval` + shellcheck clean | Timer install remains a single owner command (see B3) |
| A4 | **The twice-forgotten lesson finally executed** — shellcheck run on ALL touched scripts via `nix run nixpkgs#shellcheck`; found + fixed SC2034 in `quiet-window-run.sh`; gate script clean | Clean re-run + self-test still green |
| A5 | **Incident caught and contained (D1)** — garbage cost-pass results detected on this report's routine state check; the closure pipeline was killed BEFORE it executed its revert based on zero-median data; gate entry `::100x::9` intact, baseline untouched | `pkill` + log tail in session; `git log` baseline unchanged |
| A6 | **Root cause found and fixed; chain re-armed with guards** — `GOTOOLCHAIN=local` (inherited shell env) vs go.work's `>= 1.27.1`: every bare `go test` in my ad-hoc scripts died instantly (`wall=0s`, 0 samples). Both scripts now `export GOTOOLCHAIN=auto`; closure gained a plausibility guard (implausible/zero median → candidate skipped, never confirmed, never mutates) | `echo $GOTOOLCHAIN` → `local`; suite file error message; `bash -n` both; new pids 2033248 (cost) → 2045814 (closure) |

## b) PARTIALLY DONE

| # | What works | What remains | Blocker |
|---|-----------|--------------|---------|
| B1 | **Matview closure chain (Q1 rule)** — armed with the fix: cost pass measures the 4 configs, closure tests candidates cheapest-first (100x/9 → 1000x/5 → 1000x/9) against the accepted rule (±10% across 3 quiet runs, ≤60s), then widens → re-pins (noise-clean) → verifies autonomously; on no-qualify reverts the entry and reports | Has not produced numbers yet: the first attempt burned its quiet window on the env failure (load was 1.71/4.59 — a *perfect* window — at 11:15 UTC); re-armed chain now waits out load 34/37. 6h budgets per phase | Quiet window + my own earlier bug (now fixed) |
| B2 | **Local green verification of the 1.27.1 baseline** — gate machinery fully sound; the chain closes it autonomously once widening is confirmed | Pending B1 | Same |
| B3 | **Nightly automation** — everything shipped except the final `systemctl --user enable --now go-cqrs-nightly-bench.timer` | That command is harness-banned for me (`systemctl`); it is documented verbatim in `scripts/nightly/README.md` | Owner runs one command |
| B4 | **Doc wiring for the new tooling** — scripts self-document; TODO rows carry state | AGENTS.md Quick Reference row; stale-reference sweep (#f-carry); gotchas-testing note | Queued, not picked up |

## c) NOT STARTED (carried)

C1 T19–T21 (v5-gated) · C2 SearchQuery re-run + dgraph re-anchor via the promoted tool (row 306c/d) · C3 `api-stability` TestEvery record + composed `#verify` re-record (W1) · C4 plan-doc addenda (2026-09-18 SUPERB T18b row; TODO row ~81 pointer) · C5 MIN/matview bimodality *root cause* (independent of the widening stopgap) · C6 other-session `scripts/vm-mysql.sh` (untouched, not mine).

## d) TOTALLY FUCKED UP

**D1 — The cost pass burned a perfect quiet window on a silent env failure, and the closure nearly mutated the gate on garbage.** Severity: high (this round's defining failure). Chain of events: the 11:15 UTC window was the best of the entire session (load 1.71/4.59); the cost pass entered it and all four configs failed in `wall=0s` — `go.work requires go >= 1.27.1 (running go 1.26.7; GOTOOLCHAIN=local)` — because my ad-hoc scripts inherited `GOTOOLCHAIN=local` while every repo script explicitly sets `GOTOOLCHAIN=auto`. The documented rule ("cache env chain on every go command", gowork-modes.md) was skipped for scripts I classified as "mere session tooling". Worse: the closure then woke, read `MIN/matview median: 0 ns; samples: 0`, and began "confirming" a zero median — on schedule to exhaust candidates and **revert the `::100x::9` gate entry based on garbage input**. I killed it mid-flight on this report's routine state check (A5). Three real defects, all fixed: (1) missing `GOTOOLCHAIN=auto` export; (2) no plausibility guard on measured data (median=0 accepted); (3) destructive action (revert) reachable from unvalidated input — guards must precede mutations, always.

**D2 — I armed a four-config measurement chain without smoke-testing it.** One 5-second manual run of a single `go test` invocation (the exact command the cost script wraps) would have exposed the env failure before arming anything. "Test the test" was in my own improvement list from report #2 (pre-flight everything) — applied to flake evals and awk fixtures, not to the newest scripts' runtime env. The failure mode (detached script, silent stderr-to-file, no watchdog) meant nobody would have noticed until the deadline.

**D3 — The pid-chained design is brittle.** The closure waits on a hard-coded pid; a reboot or pid reuse desyncs the chain silently. It worked this time (re-arm re-chained), but polling for the *results file* (mtime/content) instead of a pid is the durable pattern. Listed as f#20.

## e) WHAT WE SHOULD IMPROVE

1. **The env chain is load-bearing for every `go` invocation, including "temporary" scripts.** Concrete fix candidate: a tiny `scripts/go-env.sh` (`export GOTOOLCHAIN=auto` + cache vars) that repo scripts and session tooling both source — the knowledge currently lives in per-script discipline, and discipline just failed twice.
2. **Guards before mutations, plausibility before confirmations** — any automation that reverts/re-pins/writes must first assert its inputs are physically plausible (nonzero medians, sane sample counts, expected file shape).
3. **Smoke-test the exact wrapped command before arming a chain** — 5 seconds of manual execution beats hours of silent failure.
4. **Watchdogs for detached work**: a completion-check ("results file non-empty within N minutes of window entry") with a loud marker file beats discovering failure at the next report.
5. **Report-time state checks are load-bearing** — this incident was caught *because* the status-report ritual forces a full state read. Keep the ritual even when "nothing changed".

## f) NEXT TASKS (up to 50; ★ = already in TODO_LIST; harvest candidates)

| # | Task | Impact | Effort | Category |
|---|------|--------|--------|----------|
| 1 | ★ Let the re-armed chain land: cost numbers → rule check → widen/re-pin/verify (autonomous) | Critical | S | Verification |
| 2 | ★ `scripts/go-env.sh` helper (GOTOOLCHAIN=auto + cache chain) sourced by gate scripts + documented for session tooling | High | S | Quality |
| 3 | Install the nightly timer (one owner command; README has it) | High | S | Feature |
| 4 | `shellcheck` gate: add both benchmark scripts (and go-env.sh) to a standing lint path | Medium | S | Quality |
| 5 | Stale-reference sweep: old headline list / unconditional `--save` / matview benchtime mentions (README, `cmd/cqrs-bench/README.md`, docs/benchmarks, workflows) | High | S | Documentation |
| 6 | AGENTS.md Quick Reference rows: `#quiet-window-run`, `#nightly-bench` | Medium | S | Documentation |
| 7 | ★ Reclassify remaining MIN/matview "regression" language in watcher logs/docs (expected sampling noise until widening) | Low | S | Cleanup |
| 8 | Preserve `/tmp` gate evidence into `docs/benchmarks/` (pipeline, verify, cost, closure logs) | Medium | S | Cleanup |
| 9 | SearchQuery count=5 re-run via `nix run .#quiet-window-run -- …` (row 306c) | High | S | Verification |
| 10 | Dgraph constants re-anchor campaign (row 306d) | High | M | Quality |
| 11 | `api-stability` TestEvery green record (W1) | High | S | Verification |
| 12 | Composed `#verify` re-record (gate script changed twice since last green) | Medium | S | Verification |
| 13 | Benchkit-side variance-aware stability probe (two-axes verdicts: machine-loud vs sampler-unstable) | Medium | M | Feature |
| 14 | Root-cause MIN/matview bimodality (perf/freq governor/alignment) — independent of widening | High | M | Bug |
| 15 | Investigate `SUM_VIA_GROUPED/baseline` +34.5% one-off | Low | S | Bug |
| 16 | Deep-quiet window probe/logger (loadavg sampling → best campaign windows) | Medium | M | Feature |
| 17 | Auto-embed noise verdict + CoV + `GOVERSION` into `--save` headers | High | S | Quality |
| 18 | Mutation-tested fixture: noise-fail + `--save` must refuse | Medium | S | Quality |
| 19 | Add `quiet-window-run --self-test` to `check-release-scripts` CI coverage | Medium | S | Quality |
| 20 | Replace pid-chaining with results-file polling in chained automation | Medium | S | Quality |
| 21 | Confirm CI's baseline artifact carries the 5 new entries | Medium | S | Quality |
| 22 | Sweep other gates for fragile p99/max thresholds (~100-iteration estimators) | Medium | M | Quality |
| 23 | `api-stability` README-claims ownership check (W1 remainder) | Medium | S | Verification |
| 24 | Reconcile the overnight `gotchas-testing.md` +1 line (449416da2) with the quiet-window caveat | Medium | S | Documentation |
| 25 | Document the closure re-arm procedure in TODO (reboot insurance, incl. GOTOOLCHAIN fix) | Low | S | Documentation |
| 26 | Case-study appendix in `docs/benchmarks/calibration-2026-08-30.md`: the full storm/reboot/p99/bimodal/GOTOOLCHAIN series | Medium | S | Documentation |
| 27 | Plan-doc addendum: 2026-09-18 SUPERB T18b row → DONE pointer | Low | S | Documentation |
| 28 | Annotate TODO row ~81 pointer ("remaining bench re-pin is T18b/T14") as resolved | Low | S | Documentation |
| 29 | Audit other pre-1.27 baselines for the same staleness class | Medium | S | Quality |
| 30 | Nightly gate: add load-sweep leg on a weekly cadence? (→ Q3-adjacent policy) | Low | S | Decision |
| 31 | `--explain <bench>` per-sample diagnostics in the gate | Medium | M | Feature |
| 32 | `--json` evidence output for the promoted tool | Low | S | Feature |
| 33 | DirectSQL A/B benches gate-set decision (dep-budget review first) | Low | S | Decision |
| 34 | Refresh stale load claims in TODO rows (now up to 99) | Low | S | Cleanup |
| 35 | Reconcile `gotchas-testing.md` quiet-window caveat with 449416da2's +1 line — one authoritative note | Medium | S | Documentation |

**Harvest note:** ★ in TODO_LIST; #1–#6 are the actionable core; the rest route through docs-health HARVEST on instruction.

## g) QUESTIONS I CANNOT ANSWER MYSELF

**Q1 — Widening fallback:** if the cost pass shows even count=9/1000x fails the ±10% stability rule (i.e., the bimodality is not sampling-count-limited), which fallback do you want: (a) drop MIN/matview from the gate set entirely (simplest; loses that serving-path signal), (b) move it to a nightly non-gating suite (signal without gate risk), or (c) extend the KNOWN_UNSTABLE expiry and keep investigating root cause? The accepted rule covers "widen if it works" — not this branch.

**Q2 — Baseline history policy:** should superseded baselines be preserved as a dated series (`docs/benchmarks/baselines/<date>.txt`) for long-term perf archaeology, or is single-file supersede (current) fine and the git history IS the archive?

**Q3 — CI matview parity:** CI's matview leg runs `-benchtime=10x -count=5` while local will run the widened config — medians across the two are not comparable, so CI drift detection for matview is approximate. Bump CI's matview leg to match (slower CI job, like-for-like drift detection), or keep CI cheap and accept local-only matview gating?

---

*State at writing: cost pass pid 2033248 + closure pid 2045814 (chained, GOTOOLCHAIN=auto, plausibility-guarded, 6h budgets); stale zero-results purged; gate entry `::100x::9` intact; baseline `a91e7cd90` untouched; `scripts/vm-mysql.sh` still another session's uncommitted work. No manual commit per harness contract. Waiting for instructions.*

---

## ADDENDUM 2026-09-21 ~14:30 CEST — three more rulings executed; chain hardened + relaunched

Q1 fallback = **nightly-only suite**; Q2 = **versioned baseline history**; Q3 = **CI parity**. All encoded into the re-armed closure:

| Ruling | Implementation |
|--------|----------------|
| Q2 versioned history | `benchmark-regression.sh --save` now auto-archives the superseded baseline to `docs/benchmarks/baselines/benchmark-baseline-<UTC>.txt` before overwrite. Fixture-verified (archive contains old content, new baseline written, exit 0) |
| Q3 CI parity | On qualify, the closure updates the benchmarks.yml matview stanza (`-benchtime/-count`) to the chosen config via a targeted awk rewrite (only the stanza following the matview bench regex) |
| Q1 fallback | On no-qualify: matview suite removed from GATE_SETS, `scripts/nightly/matview-stability.sh` created (1000x/9, non-gating, `GOTOOLCHAIN=auto`), auto-wired into `nightly-bench.sh`; syntax-checked before the closure declares success |

**Incident closure (D1):** the killed closure had NOT mutated anything (revert was pre-guard); stale zero-results purged; both scripts re-armed with `GOTOOLCHAIN=auto` + plausibility guards (new pids: cost 3244128 → closure 3258300). The `GOTOOLCHAIN=local` env incident is now documented in TODO row 40 as a fresh instance of the env-chain rule.

**Remaining for closure:** one real quiet window (the only blocker all session); everything after that is autonomous per the accepted rule + fallback.
