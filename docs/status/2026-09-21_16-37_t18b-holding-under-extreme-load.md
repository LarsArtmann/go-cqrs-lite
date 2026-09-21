# Status Report #6: T18b — Holding Under Extreme Load (863); Chain Intact, Zero Execution This Round

**Date:** 2026-09-21 16:37 CEST (14:37 UTC)
**Scope:** continuation of the T18b session (reports #1–#5 + addenda). This round executed **nothing** — by design it is a state check, and the honest headline is the machine state itself. No unrelated work researched.

---

## The headline: load 863.32 / 782.28 / 462.62 on 32 cores

At report time the shared host is under the most extreme load of the entire session (prior peak this week: 207 during a compile storm, 99 earlier today). At this level:

- No benchmark or calibration work is possible — quiet windows (load1/load5 < 5) are not a near-term prospect.
- All three detached stages are **alive and correctly sleeping** (60s polls, no CPU cost): closure-completion (pid 1566584), campaign queue (pid 1528750), root-cause campaign (pid 1593577).
- The completion watcher's 6h deadline (started ~15:45 CEST, expires ~21:45 CEST) will likely **lapse during this storm** — its designed behavior is to exit 3 having written nothing. Re-arm is one command: `setsid nohup bash /var/tmp/t18b/closure-completion.sh </dev/null >/dev/null 2>&1 &` (scripts preserved in `/var/tmp/t18b/`).
- Baseline `a91e7cd90` still untouched; gate entry `100x::9` + CI parity committed; nothing has been mutated since the last report.

## a) FULLY DONE (cumulative, verified unchanged this round)

A1 1.27.1 baseline re-pin with provenance (`a91e7cd90`) · A2 p99 headline demotion (production-validated decision-grade run) · A3 `--save` refusal + `--force-save` · A4 KNOWN-UNSTABLE suppression (MIN/matview, expiry 2026-10-21) · A5 per-suite `benchtime::count` + CI parity applied (matview leg → `100x/9`) · A6 baseline archival on supersede (fixture-verified) · A7 tooling promotion (`quiet-window-run` 6/6 self-test, `nightly-bench`, flake apps) · A8 nightly timer declared the SystemNix way (eval-verified, formatted, **uncommitted for owner**) · A9 weekly load-sweep leg in nightly (Sundays) · A10 campaign queue + root-cause matrix armed and chained · A11 shellcheck clean on all six touched scripts · A12 four status reports + three addenda; TODO rows 34/40/41 carry the complete decision log.

## b) PARTIALLY DONE

| #  | What works                                                                                                 | What remains                                                                                                   | Blocker                                                          |
| -- | ---------------------------------------------------------------------------------------------------------- | -------------------------------------------------------------------------------------------------------------- | ---------------------------------------------------------------- |
| B1 | Autonomous closure chain (armed, guarded, GOTOOLCHAIN-fixed)                                               | The actual quiet-window execution: re-pin → verify → green                                                     | Load 863; watcher deadline may lapse tonight (re-arm documented) |
| B2 | Campaign queue (SearchQuery count=5 + dgraph calibration capture) — wiring smoke-tested                    | Runs only after B1                                                                                             | Same                                                             |
| B3 | Root-cause matrix (cold/warm, taskset, GOMAXPROCS=1, freq sampling, conditional perf stat)                 | Runs only after B2                                                                                             | Same                                                             |
| B4 | Nightly automation                                                                                         | Owner-side timer activation via SystemNix deploy (sudo)                                                        | Owner                                                            |
| B5 | Hygiene batch — 4 of ~10 S-items executed last round (gotchas caveat, AGENTS rows, plan addendum, row ~81) | go-env.sh helper; check-release-scripts wiring; stale-reference sweep; watchdog markers; evidence preservation | Time, not blockers — listed in (f)                               |

## c) NOT STARTED

C1 T19–T21 (v5-gated) · C2 SearchQuery + dgraph doc/constant edits (numbers must land first — B1/B2) · C3 W1 siblings (`api-stability` TestEvery, composed `#verify` re-record) · C4 MIN/matview bimodality root cause (queued as campaign 3, blocked behind B1/B2) · C5 other-session `scripts/vm-mysql.sh` (untouched) · C6 SystemNix timer deploy (owner).

## d) TOTALLY FUCKED UP

**D1 — This round delivered zero execution, and the round before it under-delivered its own ruling.** The "execute hygiene now" ruling landed 4 of ~10 S-sized items before the report interrupted; the remainder have sat listed across three reports. Severity: process debt, not correctness — but a ruling of the owner ("execute now") that ships 40% is a miss, and this report would rather carry it in (d) than bury it in (b). The items are f#4–#8 here, each ≤30 min, none blocked.

**D2 — The session's entire outcome still hangs on detached bash scripts outliving a hostile machine.** Load 863 is exactly the class of event (like last night's reboot) that the architecture cannot survive: if the OOM killer or an admin restarts sessions, three armed stages die and the recovery is manual re-arming from report prose. The durable designs (systemd user units — which now exist for nightly; results-file polling; marker files) have been _listed_ since report #4 and only partially applied (nightly is a real unit; the matview chain is still raw bash). Honest classification: the arc's remaining fragility is chosen-by-inertia, not unknown.

**D3 — Six reports in two days for one task arc.** Each was requested and each is honest, but the documentation-to-execution ratio is now inverted: ~5 documents per executed re-pin. The consolidation task (one canonical `docs/benchmarks/` record replacing five timestamped narratives) is f#9 and should close this class.

## e) WHAT WE SHOULD IMPROVE

1. **Execute-during-wait discipline**: storm-waits are the ideal slot for the queued S-items; "report first, work later" wasted three windows' worth of wait time this session.
2. **Supervise the supervisors**: deadlines that lapse into exit-3 require a human to re-arm. A single supervising unit (or a cron line re-launching the chain idempotently) would make the whole chain survive both storms and deadlines — the design exists (nightly unit pattern), the matview chain just predates it.
3. **One canonical record per arc**: replace the five-report narrative series with a single `docs/benchmarks/2026-09-20-21_t18b-record.md` + one ADR for the gate-semantics changes (headline demotion, save guard, known-unstable table, per-suite opts). Five timestamped reports are a session log, not a reference.
4. **Consolidate the env-chain lesson into code**: `scripts/go-env.sh` sourced everywhere (carried since report #4; still open — the GOTOOLCHAIN incident will recur for the next ad-hoc script otherwise).
5. **Carried**: two-axis gate verdicts; stale-reference sweep; watchdog marker files; evidence preservation decision (currently: reports are the archive — acceptable, documented).

## f) NEXT TASKS (up to 50; ★ = already in TODO_LIST; harvest candidates)

| #  | Task                                                                                                                                                                              | Impact   | Effort | Category      |
| -- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | -------- | ------ | ------------- |
| 1  | ★ Land the closure tail (re-pin + verify → green) when a window opens; re-arm with the documented one-liner if the deadline lapses                                                | Critical | S      | Verification  |
| 2  | ★ Campaign queue fires automatically after #1 (SearchQuery + dgraph capture)                                                                                                      | Critical | S      | Verification  |
| 3  | ★ Root-cause matrix fires automatically after #2                                                                                                                                  | High     | S      | Investigation |
| 4  | ★ Convert the matview chain to a supervised design (systemd user unit à la nightly, or a cron re-arm line) so storms/deadlines cannot strand it                                   | High     | M      | Quality       |
| 5  | ★ `scripts/go-env.sh` + adopt in gate scripts                                                                                                                                     | High     | S      | Quality       |
| 6  | ★ Execute the remaining hygiene S-items: stale-reference sweep; `check-release-scripts` wiring for quiet-window-run; watchdog marker files; evidence-preservation decision record | Medium   | S      | Cleanup       |
| 7  | ★ Owner: install/activate the nightly timer via SystemNix deploy                                                                                                                  | High     | S      | Feature       |
| 8  | One canonical `docs/benchmarks/` T18b record consolidating reports #1–#6; add ADR for the four gate-semantics changes                                                             | High     | M      | Documentation |
| 9  | `api-stability` TestEvery green record (W1)                                                                                                                                       | High     | S      | Verification  |
| 10 | Composed `#verify` re-record (three script changes since last green)                                                                                                              | Medium   | S      | Verification  |
| 11 | SearchQuery/dgraph doc supersede + constant re-anchor edits once campaign numbers land (manual design pass)                                                                       | High     | M      | Quality       |
| 12 | Case-study appendix in `docs/benchmarks/calibration-2026-08-30.md` (storm/reboot/p99/bimodal/GOTOOLCHAIN/863 series)                                                              | Medium   | S      | Documentation |
| 13 | MIN/matview bimodality root cause (after matrix numbers)                                                                                                                          | High     | M      | Bug           |
| 14 | Investigate `SUM_VIA_GROUPED/baseline` +34.5% one-off                                                                                                                             | Low      | S      | Bug           |
| 15 | Benchkit variance-aware stability probe (two-axes verdicts)                                                                                                                       | Medium   | M      | Feature       |
| 16 | Stale-reference sweep (old headline list, save semantics, matview benchtime mentions)                                                                                             | High     | S      | Documentation |
| 17 | Deep-quiet window probe/logger                                                                                                                                                    | Medium   | M      | Feature       |
| 18 | Auto-embed noise verdict + CoV + `GOVERSION` in `--save` headers                                                                                                                  | High     | S      | Quality       |
| 19 | Mutation fixture: noise-fail + `--save` must refuse                                                                                                                               | Medium   | S      | Quality       |
| 20 | Results-file polling instead of pid-chaining                                                                                                                                      | Medium   | S      | Quality       |
| 21 | Confirm CI's baseline artifact carries the 5 new entries (CI parity change may also force an artifact refresh)                                                                    | Medium   | S      | Quality       |
| 22 | Sweep other gates for fragile p99/max thresholds                                                                                                                                  | Medium   | M      | Quality       |
| 23 | `--explain <bench>` per-sample diagnostics                                                                                                                                        | Medium   | M      | Feature       |
| 24 | `--json` evidence output for quiet-window-run                                                                                                                                     | Low      | S      | Feature       |
| 25 | Plan-doc addendum for the 2026-09-20_17-40 SUPERB doc's M06/F27 rows (T18b rows there are stale-DONE too)                                                                         | Low      | S      | Documentation |
| 26 | DirectSQL A/B benches decision (dep-budget review)                                                                                                                                | Low      | S      | Decision      |
| 27 | Refresh stale load claims in TODO rows (peak now 863)                                                                                                                             | Low      | S      | Cleanup       |
| 28 | Retire `/tmp` + `/var/tmp/t18b` copies after the chain lands                                                                                                                      | Low      | S      | Cleanup       |
| 29 | Owner-bundle: benchmark box vs higher ceilings for non-headline work (carried; load 863 strengthens the case)                                                                     | High     | S      | Decision      |
| 30 | Host ceiling policy ruling (carried; undecided)                                                                                                                                   | High     | S      | Decision      |
| 31 | Weekly load-sweep leg: verify it fires this Sunday and logs cleanly (first real run)                                                                                              | Medium   | S      | Verification  |
| 32 | Post-green: close TODO row 34's caveat with the verifying run's hash                                                                                                              | Medium   | S      | Verification  |
| 33 | SystemNix: commit + deploy the home.nix timer (owner)                                                                                                                             | High     | S      | Feature       |
| 34 | Consider CI-only arbitration for matview if local windows stay scarce (strategy fork, cf. report #4 Q3)                                                                           | Medium   | S      | Decision      |
| 35 | Add `nightly-bench` + `quiet-window-run` to the repo README's operations section                                                                                                  | Low      | S      | Documentation |

**Harvest note:** ★ in TODO_LIST. Critical path: #1 → #2 → #3 (all autonomous once a window opens) and #7 (owner one-liner). The rest are unblocked S/M work.

## g) QUESTIONS I CANNOT ANSWER MYSELF

**Q1 — What is producing load 863?** I have deliberately not investigated processes (out of session scope). If this is a known/expected job, the watchers simply keep waiting; if it is runaway (e.g., a stuck test loop from some session), it needs killing before anything else can run. You know what should be on this box.

**Q2 — Deadline-lapse policy:** when an armed stage lapses (exit 3, nothing written), should I auto-re-arm it indefinitely until it succeeds (self-healing, but could wait days silently), or keep one-shot semantics with manual re-arm (current)? This decides whether T18b closes tonight, this week, or on the next session.

**Q3 — At what storm frequency does local benchmark gating stop being worth it?** If 400-800 load events are now routine on this host, the honest options are: CI-only arbitration (retire local gating), a dedicated quiet machine, or sanctioned higher ceilings with the noise gate as sole arbiter. This is the strategy fork behind B1 — your host, your call.

---

_State at writing: three detached stages alive (1566584 → 1528750 → 1593577); completion deadline ~21:45 CEST (re-arm one-liner above); baseline `a91e7cd90` untouched; `docs/benchmarks/baselines/` does not exist yet (created at first re-pin); `scripts/vm-mysql.sh` still another session's uncommitted work. No manual commit per harness contract. Waiting for instructions._
