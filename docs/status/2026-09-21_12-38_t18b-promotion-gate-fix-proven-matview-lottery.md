# Status Report #3: T18b — Promotion Shipped, Gate Fix Proven, Matview Lottery Quantified

**Date:** 2026-09-21 12:38 CEST (10:38 UTC)
**Scope:** continuation of the T18b session (`docs/status/2026-09-20_22-02_…`, `docs/status/2026-09-21_10-30_…`) — covers the owner rulings executed since report #2 (tooling promotion, chained cost pass) and the overnight verifier outcome. No unrelated work researched.
**Host:** shared 32-core; **load 99.6/36/22 at writing** — the heaviest storm yet; the chained cost pass (pid 1016121) is waiting it out (6h deadline, expires ~17:26 CEST).

---

## a) FULLY DONE

| # | What | Evidence |
|---|------|----------|
| A1 | **Carried: 1.27.1 baseline re-pin + gate fixes** — baseline (`a91e7cd90`), `--save` guard + `--force-save`, p99 headline demotion all committed and intact after daemon absorption | Verified in committed `scripts/benchmark-regression.sh:107,131` this morning |
| A2 | **Q2 ruling executed: quiet-window tooling PROMOTED** — `scripts/quiet-window-run.sh` (load1/load5 < `--ceiling` waiter, bounded `--attempts`/`--no-retry` retries, `--deadline`, `--log`) | Script + 6/6 green `--self-test` (quiet+pass, never-quiet deadline exit 3, retry-then-pass with both attempts logged, always-fail exhausted exit 1, usage exit 2) |
| A3 | **Flake app wired** — `nix run .#quiet-window-run -- CMD` mirrors the `load-sweep` app pattern; `nix eval` resolves the store path | flake.nix (daemon-committed); live `nix run … -- --self-test` PASS |
| A4 | **Q1 fix production-validated** — the 08:46 UTC verification run's noise gate **PASSED** with the demoted headline at load 1.87/4.84 (`write_p99_ns` advisory at CoV 22.5%). First decision-grade run since the 1.27.1 cutover | `/tmp/t18b-verify.log` attempt 1 |
| A5 | **Bounded-attempts semantics executed correctly end-to-end** — the decision-grade run flagged `MIN/matview +33%` (11062 vs 8319 ns/op); verifier exited 5, baseline stands, flag recorded for triage — exactly the codified acceptance | Verifier log; TODO rows 34/40 updated with the final state |
| A6 | **Q3 ruling armed: matview cost pass chained** — measures the matview gate suite at (100x/5, 1000x/5, 100x/9, 1000x/9): wall-clock + MIN/matview medians per config; detached, survives until its deadline, waits for the quiet window | `/tmp/matview-cost.sh` (pid 1016121), log `/tmp/matview-cost.log` |
| A7 | **Bookkeeping current** — TODO rows 34/40 carry the 2026-09-21 updates (production validation, expected-noise classification for MIN/matview flags, promotion DONE, ceiling-policy open); status report #2 got a dated addendum | TODO_LIST (daemon-committed 221553bf9 et al.) |

## b) PARTIALLY DONE

| # | What works | What remains | Blocker |
|---|-----------|--------------|---------|
| B1 | **Local green verification** — the gate itself is now sound (A4/A5); the only failing comparison is the matview sampling artifact | A green run requires either (i) the widening decision (numbers from the cost pass) or (ii) a re-pin under widened config. My earlier "REAL decision-grade regression" phrasing in the watcher log/TODO **overstates** it: the noise gate certifies machine quietness, not sampling adequacy for a bimodal benchmark — no library code changed between baseline save and verify, and the same variant has now flagged 4+ times across runs while its samples span 7.9–15µs (bimodal modes ~8.3k vs ~11–13k) | Cost-pass numbers → your widening call |
| B2 | **Matview cost pass** — armed and correct | Has not run: load hit 99 this morning; the quiet window has not reopened. Deadline expires ~17:26 CEST; if it lapses, re-arm is one command (script survives in this report's instructions) | Shared-host load |
| B3 | **Tooling documentation** — the script documents itself; `--self-test` is in-repo | No AGENTS.md Quick Reference row; not yet added to `check-release-scripts` CI coverage; no stale-reference sweep for gate docs mentioning the old save semantics | In-session scope; listed in (f) |

## c) NOT STARTED (carried open; deliberately untouched)

| # | Item | Why | Wanted? |
|---|------|-----|---------|
| C1 | T19–T21 universal-`Engine` fold + duplicate SQL stack deletion + release train | v5-gated (ADR-0142 §2) | Yes, at v5 |
| C2 | Row 306: SearchQuery count=5 re-run; dgraph constants re-anchor | Quiet-window campaigns — now unusually cheap to run via the promoted tool (`nix run .#quiet-window-run -- …`) | Yes |
| C3 | Nightly wiring for the promoted tooling | Your call — "can follow" per the promote-now ruling (→ Q2) | Open |
| C4 | Host benchmark-ceiling policy | You answered "IDK" on 2026-09-21; strict `<5` remains until ruled | Open |
| C5 | W1 siblings (`api-stability` TestEvery record, composed `#verify`/`#verify-ci` re-record), plan-doc addenda (2026-09-18 SUPERB T18b row; TODO row ~81 pointer), stale-reference sweep, `shellcheck` on both scripts, evidence preservation under `docs/benchmarks/` | Queued f-items from reports #1/#2, not yet picked up | Yes |
| C6 | Another session's uncommitted `scripts/vm-mysql.sh` edit | Not mine | n/a |

## d) TOTALLY FUCKED UP

**D1 — I repeated the exact gap I had just written down.** Report #2's section (e) #1 said "lint every script edit, immediately" after I shipped the gate changes without a shellcheck pass — and then I shipped `quiet-window-run.sh` the same way (bash -n + self-test, **no shellcheck**). Severity: low (self-test covers behavior; but an unlinted script can red the next `#verify`). This is now a twice-made mistake; it goes in (d), not (e), because repeating a self-documented lesson is a discipline failure, not an oversight. Fix queued as f#2.

**D2 — The cost pass repeats the reboot-vulnerable design.** It lives in `/tmp`, detached, with results in `/tmp` — the same shape that lost last night's v2 watcher and all evidence to the overnight reboot. I documented that exact lesson (report #2, e#2 "assume reboot") and then chained a new `/tmp`-based watcher anyway, because the measurement needed no repo changes. If the host reboots before ~17:26 CEST, the pass dies silently and the numbers wait another day. Mitigation: the re-arm command is trivial (`bash /tmp/matview-cost.sh` after recreating it from this report) — but "trivial to redo" is not "designed to survive".

**D3 — My verdict language can mislead triage.** The verifier log and an earlier TODO phrasing call the MIN/matview flag a "REAL decision-grade regression". The run IS decision-grade per the gate's semantics (machine quiet), but the benchmark itself violates the gate's sampling assumptions (bimodal 5-sample median), so "regression" is the wrong word for a future reader triaging ghosts. Corrected classification now recorded in TODO rows 34/40 ("expected noise until the widening decision"); the language lesson is (e) #3. No wrong action was taken on the mislabel — the baseline was never reverted — but the words were wrong for ~2 hours.

## e) WHAT WE SHOULD IMPROVE

1. **Shellcheck is part of "script done"** — same breath as `bash -n` and the self-test, every time, before arming anything. Twice-burned.
2. **Anything that must survive the session gets a durable home**: armed watchers' scripts + logs belong in `/var/tmp` or `docs/benchmarks/` (or the TODO row carries the inline re-arm script). `/tmp` is for throwaway only — and I keep using it for not-throwaway.
3. **Gate verdicts need two axes**: "machine loud" (noise gate) vs "benchmark unstable" (sampling adequacy). A single PASS/FAIL conflates them; the MIN/matview saga is the proof. Concrete: teach the verifier/log vocabulary to say "decision-grade run, per-benchmark sampling flag" vs "noise-rejected run", and consider a benchkit-side `BenchmarkStability` probe (variance-aware) instead of inferring from compare failures.
4. **New tools get wired into the standing gates in the same session**: `check-release-scripts` should run `quiet-window-run.sh --self-test` in CI; AGENTS.md Quick Reference should list it. Otherwise the next session rediscovers it by accident.
5. **Reference sweeps remain undone** (carried from report #2): one grep for save-semantics/headline mentions in README/docs/workflows.

## f) NEXT TASKS (up to 50; ★ = already in TODO_LIST; harvest candidates)

| # | Task | Impact | Effort | Category |
|---|------|--------|--------|----------|
| 1 | ★ Land matview cost-pass numbers → your widening ruling (pre-commit a decision rule? → Q1) | Critical | S | Decision |
| 2 | `shellcheck scripts/quiet-window-run.sh scripts/benchmark-regression.sh`; fix findings | High | S | Quality |
| 3 | Add `quiet-window-run.sh --self-test` to `scripts/check-release-scripts.sh` coverage | High | S | Quality |
| 4 | AGENTS.md Quick Reference row for `nix run .#quiet-window-run` | Medium | S | Documentation |
| 5 | Stale-reference sweep: old headline list / unconditional `--save` mentions (README, `cmd/cqrs-bench/README.md`, docs/benchmarks, workflows) | High | S | Documentation |
| 6 | ★ Reclassify MIN/matview verdicts everywhere as "expected sampling noise until widening" (done in TODO; sweep watcher logs + any docs) | Medium | S | Cleanup |
| 7 | ★ Matview widening implementation once ruled (benchtime/count in `benchmark-regression.sh` GATE_SETS or bench files) + re-pin under the widened config | Critical | M | Quality |
| 8 | Preserve `/tmp` gate evidence (pipeline, verify, cost results) into `docs/benchmarks/` before reboot/eviction | Medium | S | Cleanup |
| 9 | SearchQuery count=5 quiet re-run via `nix run .#quiet-window-run -- …` (row 306c) | High | S | Verification |
| 10 | Dgraph constants re-anchor campaign via the tool (row 306d) | High | M | Quality |
| 11 | `api-stability` TestEvery green record (W1) | High | S | Verification |
| 12 | Composed `#verify` re-record (baseline + two script changes since the last green) | Medium | S | Verification |
| 13 | Case-study appendix in `docs/benchmarks/calibration-2026-08-30.md`: storm/reboot/p99/bimodal series as the canonical gate-rationale record | Medium | S | Documentation |
| 14 | Benchkit-side variance-aware stability probe (the two-axes idea, e#3) | Medium | M | Feature |
| 15 | Nightly wiring decision + implementation for the promoted tool (→ Q2) | Medium | M | Feature |
| 16 | Deep-quiet window probe/logger (week-long loadavg sampling → best re-pin windows) | Medium | M | Feature |
| 17 | Auto-embed noise verdict + CoV table + `GOVERSION` into `--save` headers | High | S | Quality |
| 18 | Regression fixture: noise-fail + `--save` must refuse (mutation-tested golden) | Medium | S | Quality |
| 19 | Investigate `SUM_VIA_GROUPED/baseline` one-off +34.5% outlier | Low | S | Bug |
| 20 | Investigate MIN/matview bimodality root cause (perf/freq governor/alignment) — independent of the widening stopgap | High | M | Bug |
| 21 | Confirm CI's baseline artifact carries the 5 new entries | Medium | S | Quality |
| 22 | Sweep other local gates for fragile p99/max thresholds on ~100-iteration estimators | Medium | M | Quality |
| 23 | Reconcile the overnight `gotchas-testing.md` +1 line (449416da2) with the quiet-window caveat — one authoritative note | Medium | S | Documentation |
| 24 | Plan-doc addendum: 2026-09-18 SUPERB T18b row → DONE pointer | Low | S | Documentation |
| 25 | Annotate TODO row ~81 pointer ("remaining bench re-pin is T18b/T14") as resolved | Low | S | Documentation |
| 26 | Audit other pre-1.27 baselines (`audit-tag-baseline.txt`, calibration entries) for staleness | Medium | S | Quality |
| 27 | DirectSQL A/B benches gate-set decision (dep-budget review first) | Low | S | Decision |
| 28 | `--explain <bench>` diagnostics in the gate (per-sample history; would have exposed the bimodality instantly) | Medium | M | Feature |
| 29 | Evaluate `--count` bump cost for µs-scale gate benches globally (overlaps #7's numbers) | Medium | M | Quality |
| 30 | Owner-bundle: benchmark-box vs sanctioned higher ceilings for non-headline work (carried; you answered IDK) | High | S | Decision |
| 31 | Verify daemon commits absorbed the promotion trio (script + flake + TODO) cleanly — spot-check next auto-commit | Low | S | Cleanup |
| 32 | Refresh stale load claims in TODO rows ("load at 28-74 all day" → now up to 99) | Low | S | Cleanup |
| 33 | Consider `--json` evidence output for the promoted tool (machine-readable phase/exit records) | Low | S | Feature |
| 34 | Document the re-arm procedure for the cost pass in the TODO row (reboot insurance) | Low | S | Documentation |
| 35 | After green lands: close the T18b TODO row 34 caveat formally with the verifying run's hash | Medium | S | Verification |

**Harvest note:** ★ already in TODO_LIST; the rest route through docs-health HARVEST on instruction. The actionable core: #1→#7 (matview closure), #2–#5 (hygiene), #9–#12 (trust floor).

## g) QUESTIONS I CANNOT ANSWER MYSELF

Tried: the widening cost is measurable (queued); CI/nightly wiring is technically decidable by me; the remaining blockers are your priorities and one policy pre-commitment.

**Q1 — Pre-commit the widening decision rule:** when the cost numbers land, may I act autonomously under a rule you set now? Proposal for concreteness: *widen to the cheapest config whose MIN/matview median-of-N stays within ±10% across ≥3 consecutive quiet runs, provided the matview suite stays under 60s wall-clock.* Yes/no/edit — this converts the next quiet window into a fully autonomous closure.

**Q2 — Nightly automation on this shared host:** wanted or not? The promoted tool makes nightly `#verify`-style sweeps and baseline-freshness runs practical, but nightly load on a contended host is exactly what the calibration protocol guards against — your host, your policy.

**Q3 — Gate semantics for known-unstable benchmarks:** until (or instead of) widening, do you want an explicit known-flaky annotation mechanism in `benchmark-regression.sh` (e.g. `MIN/matview:sample-limited` lines that downgrade a flag to advisory with an expiry), or strictly binary pass/fail with human triage (current)? The binary gate will keep "failing" on quiet nights until the widening lands.

---

*State at writing: cost pass detached (pid 1016121, expires ~17:26 CEST; re-arm: recreate `/tmp/matview-cost.sh` — its logic is 30 lines, described in A6); v3 verifier DONE (exit 5, flag recorded); my changes daemon-committed (221553bf9 et al.); `scripts/vm-mysql.sh` still another session's uncommitted work (untouched). No manual commit per harness contract. Waiting for instructions.*

---

## ADDENDUM 2026-09-21 ~12:55 CEST — three new rulings executed

Q1 = **accept the autonomous widening rule** (cheapest config with MIN/matview median-of-N within ±10% across ≥3 consecutive quiet runs, suite ≤60s); Q2 = **nightly yes**; Q3 = **flaky annotation**.

| Ruling | Shipped | Verification |
|--------|---------|--------------|
| Q3 known-flaky | `KNOWN_UNSTABLE` table in `benchmark-regression.sh` (`NAME\|reason\|expiry`); listed+unexpired >threshold flags print `UNSTABLE-KNOWN … [suppressed]` and never fail the gate; expiry (MIN/matview → **2026-10-21**) auto-restores strictness; per-suite `benchtime::count` opts added to GATE_SETS entries (needed by the widening anyway); matview entry pre-widened to `100x::9` as the cheapest candidate | Fixtures: listed+flagged → suppressed, exit 0; unlisted +66% → REGRESSION, exit 1; all-suppressed → exit 0. `bash -n` + shellcheck CLEAN (both gate scripts; fixed SC2034 in quiet-window-run.sh — the twice-forgotten lesson finally executed) |
| Q2 nightly | `scripts/nightly-bench.sh` (quiet-window-run → gate, logs `/var/tmp/cqrs-nightly/<date>.log`), systemd user units `scripts/nightly/go-cqrs-nightly-bench.{service,timer}` (03:00, Persistent) + README with the install line, flake app `nightly-bench` | `bash -n`, `nix eval` app path, shellcheck clean. **Timer install is yours** (systemctl is harness-banned): `cp scripts/nightly/go-cqrs-nightly-bench.{service,timer} ~/.config/systemd/user/ && systemctl --user daemon-reload && systemctl --user enable --now go-cqrs-nightly-bench.timer` |
| Q1 autonomous closure | `/tmp/matview-closure.sh` (detached, pid 624990, chained after the cost pass): candidates cheapest-first (100x/9 → 1000x/5 → 1000x/9); qualifies iff cost-pass wall ≤60s AND MIN/matview median-of-N within ±10% across 3 consecutive quiet runs; on qualify → widen entry, decision-grade re-pin (noise-clean save enforced), verify, close T18b; on no-qualify → revert entry to defaults + report numbers | `bash -n`; armed and waiting on the cost pass (which waits on the storm — load was 99 at 12:30) |

TODO rows 34/40/41 updated with the rulings. Remaining open after this: the cost pass + closure need one real quiet window (load 99 → the current blocker); ceiling policy still undecided; CI's matview leg runs its own explicit `-benchtime=10x -count=5` so it is unaffected by the local widening.
