# Status Report #2: T18b — Gate Hardening, Rulings Implemented, Verification Watch

**Date:** 2026-09-21 10:30 CEST (08:30 UTC)
**Scope:** continuation of the 2026-09-20 T18b session (`docs/status/2026-09-20_22-02_t18b-load-sweep-bench-baseline-repin.md`) — covers the owner rulings, the `benchmark-regression.sh` hardening, overnight events (host reboot), and current watch state. No other work researched.
**Host:** shared 32-core; current load 46/17/7 (morning burst draining); **the host rebooted overnight** — all `/tmp` evidence and the v2 verifier died with it.

---

## a) FULLY DONE

| # | What | Evidence |
|---|------|----------|
| A1 | **T13 load-sweep + T14 baseline re-pin (carried from report #1)** — 1.27.1 baseline committed with provenance + 5 new gate entries; 0 regressions vs the 2026-09-11 baseline | Commit `a91e7cd90`; still the last commit touching `benchmarks/benchmark-baseline.txt` (verified today) |
| A2 | **Q1 ruling implemented: `write_p99_ns` demoted from the noise headline** — default now `write_throughput write_p50_ns load_p50_ns`, with the full evidence rationale in the usage block | `scripts/benchmark-regression.sh:107` committed; rationale cites tonight's CoV series (p99 11.5–54%, max 22.9–53.9%, stable trio <10% in every run — note: this also refuted the report #1 "p50+max pair" option, since max is worse than p99) |
| A3 | **Q2 ruling implemented: `--save` refuses after a noise-gate failure** — new `--force-save` override; refusal message names the incident that motivated it | `scripts/benchmark-regression.sh:131` + save-block guard; committed |
| A4 | **Fixture suite for the guard: 5/5 green** — p99-noisy→saves (advisory); p50-noisy→refuse+exit1+no file; `--force-save`→file written (run still exits 1, honest); compare-only+save→saves (CI path intact); `bash -n` clean | Session fixture runs against planted `metricVariation` JSONs |
| A5 | **CI compatibility confirmed before the change** — CI never uses `--save` (artifact refresh is a plain `cp`; compare step is `--baseline/--current`), so the guard cannot block CI | `.github/workflows/benchmarks.yml:205-242` |
| A6 | **Q3 ruling codified: bounded-attempts re-pin acceptance** — verifier retries ≤3×; every-attempt-noise-rejected → save stands with rejections recorded; decision-grade regression → keep baseline + flag for triage | Verifier v2/v3 semantics; TODO row 40 records the ruling |
| A7 | **TODO row 40 → RESOLVED** with the three rulings, fixture evidence, and the open matview remainder; status report #1 got a dated addendum | TODO_LIST (daemon-committed 449416da2); report addendum |
| A8 | **Overnight integrity check** — hardened script survived the daemon commit intact (headline line + `--force-save` parse verified in the committed file); baseline untouched since `a91e7cd90` | grep of committed file + `git log --follow` today |

## b) PARTIALLY DONE

| # | What works | What remains | Blocker |
|---|-----------|--------------|---------|
| B1 | **Local green verification vs the 1.27.1 baseline** — the tooling is correct post-fix and the path to green is clear (last night's 4 rejections were all the now-demoted p99 metric) | **Still never ran green.** v2 watcher (2h deadline) was killed by the overnight reboot before a window opened; v3 re-armed today (6h deadline) and is waiting out a morning burst (load 46/17) | Shared-host load; reboot wiped the watcher |
| B2 | **Evidence preservation** — decisive excerpts (CoV series, compare summaries) are embedded in report #1 + its addendum | Raw attempt logs died with `/tmp`; the promoted tooling (#f7) + docs/benchmarks preservation (#f18) would make this durable | Not yet promoted (waiting on instruction) |
| B3 | **Script hardening QA** — `bash -n` + 5 fixtures done | No `shellcheck`/repo lint pass on the modified script yet; no sweep for stale references to the old headline list/save behavior (e.g. `cmd/cqrs-bench/README.md` mentions the script) | In-session oversight, listed in (e) |

## c) NOT STARTED (known, deliberately not touched)

| # | Item | Why | Wanted? |
|---|------|-----|---------|
| C1 | T19–T21 universal-`Engine` fold + duplicate SQL stack deletion + release train | v5-gated (ADR-0142 §2); reaffirmed | Yes, at v5 |
| C2 | Row 306: SearchQuery count=5 re-run (c) + dgraph constants campaign (d) | Separate quiet-window campaigns; the window economics are the open question (→ Q1) | Yes |
| C3 | Promote quiet-window pipeline → standing `scripts/quiet-window-run.sh` (+ `--self-test`, nightly wiring) | Session-local so far; promotion is queued behind your instruction (→ Q2) | Yes |
| C4 | Matview bimodality investigation/widening (MIN 7.9–15µs samples; 25% median-of-5 flags random variants on clean runs) | Tradeoff decision pending (→ Q3) | Yes |
| C5 | Plan-doc addenda: 2026-09-18 SUPERB T18b row; TODO row ~81 pointer annotation | Listed in report #1 (f#23/#24); not executed | Yes, S each |
| C6 | W1 siblings: `api-stability` TestEvery record; composed `#verify`/`#verify-ci` re-record | Owned by other rows/sessions; not mine tonight | Yes |
| C7 | Another session's in-flight `scripts/vm-mysql.sh` modification (uncommitted) | **Not mine — explicitly left alone** | n/a |

## d) TOTALLY FUCKED UP

**D1 — The v2 verifier design assumed the host stays up: it didn't.** Severity: medium (time lost, evidence lost). The 2h deadline was calibrated to "the storm passes tonight"; instead the host rebooted overnight, killing the watcher AND vaporizing every `/tmp` log (pipeline log, attempt logs, fixtures, the pre-run baseline backup). The verification never ran. Mitigations now in place: v3 deadline raised to 6h; decisive evidence excerpts were already embedded in report #1 (the only reason last night's data survives at all). Lesson recorded in (e) #2.

**D2 — I shipped the gate fix one verification-cycle later than the evidence justified.** Severity: process, low-medium. After the SECOND noise rejection (~17:59 UTC) the pattern was unambiguous — every rejection was `write_p99_ns`, everything else stable, flags moving between random matview variants. I ran 2 more attempts before asking. Counterpoint: changing gate semantics without the owner was not mine to do unprompted, and the extra runs strengthened the evidence (deep-quiet + repeat 7 refutations are what make the demotion defensible). But the asking could have happened ~1h earlier.

**D3 — No lint/shellcheck pass on the modified script.** Severity: low. `bash -n` + fixtures prove behavior, but the repo lints scripts (shellcheck class); an unlinted edit can fail the next `#verify`. Not yet run — queued as (e) #1 / f#3.

**D4 — Report #1's f-list items I flagged but did not execute** (plan-doc addenda, row ~81 annotation, log preservation, GOVERSION-into-header). Honest classification: they were queued behind "wait for instructions", but they should be tracked as owned follow-ups, not just report prose. Now they are: this report's section f.

## e) WHAT WE SHOULD IMPROVE

1. **Lint every script edit, immediately**: `shellcheck scripts/benchmark-regression.sh` (+ `nix run .#check-release-scripts` for gate scripts) belongs in the same breath as `bash -n`. Impact: prevents a surprise red in the next composed `#verify`.
2. **Assume reboot**: long watchers need (a) deadlines ≥ the longest observed storm (hours, not minutes), (b) state/logs written somewhere durable (`docs/benchmarks/` or `/var/tmp`), (c) a one-line re-arm command recorded in the TODO row. Impact: tonight's lost evidence + dead watcher were both avoidable.
3. **Ask at the second anomaly, not the fourth**: when a gate rejects twice on the SAME metric while everything else passes, that is a policy question for the owner — surface it immediately with the data. Impact: saves hours of bounded-value re-runs.
4. **Embed machine-readable evidence in artifacts**: the save header should carry the noise verdict + per-metric CoV table + `GOVERSION` automatically (report #1 e#4, still open — f#6). Impact: baselines self-audit; no /tmp dependence.
5. **Reference sweeps after behavior changes**: one grep for the changed contract (`--save` semantics, headline list) across README/docs/workflows before calling it done. Impact: kills doc drift at birth.
6. **Cross-session awareness**: check `git log` for same-day gate/tooling commits before editing shared scripts (W3 rulings landed overnight adjacent to my files). Impact: avoids split-brain edits.

## f) NEXT TASKS (up to 50; ★ = already reflected in TODO_LIST; harvest candidates)

| # | Task | Impact | Effort | Category |
|---|------|--------|--------|----------|
| 1 | Land the green local verification vs the 1.27.1 baseline (v3 watcher is armed; record evidence on rows 34/40) | Critical | S | Verification |
| 2 | `shellcheck` + repo lint on the hardened `benchmark-regression.sh`; fix anything surfaced | High | S | Quality |
| 3 | Sweep for stale references to the old headline list / unconditional `--save` (README, `cmd/cqrs-bench/README.md`, docs/benchmarks, workflows) | High | S | Documentation |
| 4 | ★ Promote quiet-window tooling → `scripts/quiet-window-run.sh` with `--ceiling/--deadline/--noise-repeat/--skip-load-sweep` + `--self-test` (planted loadavg fixtures) | High | M | Feature |
| 5 | ★ Matview bimodality: investigate MIN/matview 7.9–15µs modes (perf/freq-governor/alignment), then widen gate benchtime/count if warranted | High | M | Bug |
| 6 | Auto-embed noise verdict + CoV table + `GOVERSION` into the `--save` header | High | S | Quality |
| 7 | Add regression fixture: noise-fail + `--save` must refuse (golden per repo convention; mutation-tested) | High | S | Quality |
| 8 | SearchQuery count=5 quiet re-run (row 306c) | High | S | Verification |
| 9 | Dgraph constants re-anchor campaign (row 306d) | High | M | Quality |
| 10 | Case-study appendix in `docs/benchmarks/calibration-2026-08-30.md`: the reboot/storm/p99 series as the canonical "why the gates exist" record | Medium | S | Documentation |
| 11 | `api-stability` TestEvery green record (W1 remainder) | High | S | Verification |
| 12 | Composed `#verify` re-record post-baseline + post-script-hardening | Medium | S | Verification |
| 13 | `docs/agents/gotchas-testing.md`: quiet window ≠ decision-grade; check the noise verdict, not just load (the +1 line from 449416da2 may already cover part — reconcile) | Medium | S | Documentation |
| 14 | Investigate `SUM_VIA_GROUPED/baseline` one-off +34.5% (outlier samples) | Medium | S | Bug |
| 15 | Confirm CI's baseline artifact carries the 5 new entries (else informational "new in current" forever) | Medium | S | Quality |
| 16 | Sweep all local gates for fragile p99/max-style thresholds on ~100-iteration estimators | Medium | M | Quality |
| 17 | Deep-quiet window probe/logger (week-long loadavg sampling → best re-pin windows) | Medium | M | Feature |
| 18 | Preserve benchmark evidence under `docs/benchmarks/`, not `/tmp` (standing convention) | Medium | S | Cleanup |
| 19 | DirectSQL A/B benches in gate set decision (dep-budget review first) | Medium | S | Decision |
| 20 | Nightly baseline-freshness job (nightly-gates W2) | Medium | M | Feature |
| 21 | Plan-doc addendum: 2026-09-18 SUPERB T18b row → DONE pointer | Low | S | Documentation |
| 22 | Annotate TODO row ~81 pointer ("remaining bench re-pin is T18b/T14") as resolved | Low | S | Documentation |
| 23 | Audit other baselines predating 1.27 (`audit-tag-baseline.txt`, calibration entries) for the same staleness class | Medium | S | Quality |
| 24 | Reconcile the overnight `gotchas-testing.md` +1 line (449416da2) with my pending #13 note — one authoritative caveat, no split brain | Medium | S | Documentation |
| 25 | Verify tonight's daemon commits absorb TODO/report/script cleanly (spot-check `git show` on the next auto-commit) | Low | S | Cleanup |
| 26 | Add `--explain <bench>` diagnostics to the gate (print per-sample history for one benchmark; would have made the bimodality obvious instantly) | Medium | M | Feature |
| 27 | Codify the bounded-attempts acceptance into the promoted tooling's default semantics + docs | Medium | S | Documentation |
| 28 | Decide claimkit DirectSQL benches' place in CI's per-metric noise story (they emit no benchkit metrics today) | Low | S | Decision |
| 29 | Keep `/tmp/t18b_verify.sh` + v3 log preserved via #18 once the green lands | Low | S | Cleanup |
| 30 | Owner-bundle item: whether shared-host concurrency (→ Q1) warrants a dedicated benchmark box or sanctioned higher ceilings for non-headline campaigns | High | S | Decision |

**Harvest note:** ★ = already in TODO_LIST. The rest are candidates for docs-health HARVEST on instruction; #1–#7 are the actionable core.

## g) QUESTIONS I CANNOT ANSWER MYSELF

Tried: read CI workflow + calibration docs for ceiling policy (no guidance beyond "quiet machine"); checked whether promotion targets exist (none — W2 row is prose); the matview tradeoff has no data on how much runtime widening costs (measurable, but the acceptability of that cost is yours).

**Q1 — Shared-host concurrency planning:** how many parallel agent sessions are *normal* for this box going forward? Last night: storms to load 71, a reboot, and this morning another burst — while the calibration protocol requires load1+load5 < 5. If 2-3 concurrent sessions are the standing reality, future campaigns (dgraph re-anchor, SearchQuery, monthly re-pins) need either a sanctioned higher ceiling for non-headline work, scheduled windows, or a dedicated quiet machine — your call on which.

**Q2 — Promotion timing:** promote the quiet-window tooling into `scripts/quiet-window-run.sh` + nightly-gates now (S/M, immediately reusable by every session), or leave it session-local until the W2 "quiet-window verify tooling" row gets its own design pass? Related: is scheduled nightly automation on this shared host even wanted?

**Q3 — Matview stability economics:** for the bimodal `MIN/matview` (and µs-scale gate benches generally): accept a slower gate (wider benchtime/count — roughly doubles that suite's runtime) for stable medians, or keep count=5/100x and accept occasional manual `--force-save` re-pins when the bimodal draw flags a variant? I can measure the exact runtime cost first if you want numbers before deciding.

---

*State at writing: v3 verifier detached (pid 295475, 6h deadline, log `/tmp/t18b-verify.log`); `scripts/vm-mysql.sh` carries another session's uncommitted edit (untouched); my hardening + TODO/report changes are daemon-committed (latest touching mine: `449416da2`). No manual commit per harness contract. Waiting for instructions.*

---

## ADDENDUM 2026-09-21 ~10:45 CEST — rulings executed; noise-gate fix production-validated

Answers to report #2 section (g): Q1 = **undecided** (ceiling policy stays strict `<5` until ruled — recorded in TODO); Q2 = **promote now**; Q3 = **measure first**. Execution:

| Ruling | Shipped | Verification |
|--------|---------|--------------|
| Q2 promote-now | `scripts/quiet-window-run.sh` (window waiter: load1/load5 < `--ceiling`, bounded `--attempts`/`--no-retry` retries, `--deadline`, `--log`) + `--self-test` (6/6 green: quiet+pass, never-quiet deadline exit 3, retry-then-pass with both attempts logged, always-fail exhausted exit 1, usage exit 2) + flake app `nix run .#quiet-window-run -- CMD` | `bash -n`; self-test via direct + flake invocation; `nix eval` app path resolves |
| Q3 measure-first | Chained cost pass `/tmp/matview-cost.sh` (detached): waits for the verifier, then the quiet window, then measures the matview suite at (100x/5, 1000x/5, 100x/9, 1000x/9) — wall-clock + MIN/matview medians per config → `/tmp/matview-cost-results.txt` | Script syntax-checked; armed (pid 1016121) |
| Q1 undecided | No ceiling changes; recorded as open in TODO | — |

**Overnight verifier outcome (v3, 08:46 UTC):** first post-fix decision-grade run — **noise gate PASSED** with the demoted headline (`write_p99_ns` advisory at CoV 22.5%), validating the Q1 fix in production. The run flagged `MIN/matview +33%` (11062 vs 8319 ns/op) → per the bounded-attempts ruling: baseline stands, flagged for triage. Assessment: the known bimodal lottery (same variant flagged 3+ times across all of last night's runs; samples 7.9–15µs; the baseline saved the low mode) — implausible as a real regression since no library code changed between save and verify; the cost pass's count=9 medians will settle it with numbers. TODO rows 34/40 updated with the final state; tooling promotion recorded DONE with open halves (nightly wiring, ceiling policy).
