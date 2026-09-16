# Status: Benchmark Statistical Rigor (benchkit + cqrs-bench) — 2026-09-16 02:09

> Session scope: "How could we improve our benchmarks and benchmark SDKs?" —
> analysis + implementation + verification. This report covers only this
> session's work and what it surfaced. Auto-commit daemon absorbed all edits
> into `chore: auto-commit` commits (expected).

## Verdict

Shipped 5 improvements, all verified green. One SDK doc (benchkit/README.md)
was initially forgotten and fixed during this review. Two PRE-EXISTING red
gates (not caused by this session) remain open and are the top CI blockers.

---

## a) FULLY DONE (implemented + verified this session)

1. **P100 exactness** — `LatencyCollector` tracks the true max on every
   `Record`; `LatencyStats.P100` is the exact worst observed latency instead
   of the largest value that survived reservoir sampling (a stall in a
   >10K-sample run was previously evicted and invisible).
   Regression test `TestLatencyCollector_TailSpikeSurvivesReservoir` proves
   the old implementation fails. (benchkit/metrics.go, metrics_test.go)
2. **`RunRepeated` + `RepeatedResult`** — multi-run benchmarks now return
   EVERY run plus the median (annotated). `Reliable()` / `NoisyMetrics()`
   verdicts. `Run` keeps its signature (back-compat). (benchkit/repeat.go)
3. **Per-metric cross-run variation** — `Result.MetricVariation` (mean,
   StdDev, CoV, per-run samples) for ~34 metrics, not just write throughput;
   `VariationThreshold`, `NoisyMetricNames`. Metrics recorded by <2 runs are
   dropped (no fake CoV-0). Text report prints a `Variation:` section
   (empirical first run: only 7/34 metrics stable at dev profile — previously
   invisible noise); table gets a `Variation` row. (benchkit/repeat.go,
   report_variation.go; cmd/cqrs-bench/render_variation.go)
4. **Real benchstat samples** — `WriteBenchstatRepeated` emits one line per
   run per metric when `--repeat N --format benchstat` (verified: 34 metrics
   × 3 samples). Previously exactly ONE line per metric regardless of repeat,
   so `benchstat old.txt new.txt` could never report confidence intervals.
   Also fixed mislabeled units (run totals no longer carry `/op`) and added
   `write_max_ns`/`load_max_ns`. (benchkit/artifacts.go, cmd/cqrs-bench)
5. **Load provenance** — `Environment.LoadAvg1` recorded at run start
   (Linux `/proc/loadavg`); oversubscribed machine (load > CPU count) records
   a warning naming the pollution. Noisy runs are now self-describing.
   (benchkit/environment.go, env_linux.go, env_other.go, runner.go)
6. **Dedup** — `ambientLoadFactor` (test) now reuses `detectLoadAvg1()`;
   killed the clone this session initially introduced (caught by
   `#check-duplication`). (benchkit/load_aware_test.go)
7. **Docs shipped**: cmd/cqrs-bench/README.md (stale backends/formats/flags
   refreshed + new "Statistical rigor" section), docs/benchmarks/README.md
   (historical-record banner + how-to-run-today), SKILL.md benchmarking
   section, benchkit/doc.go (percentile semantics + repeats section),
   benchkit/README.md (metrics list + new "Repeats and statistical rigor"
   section — initially forgotten, fixed 02:05), FEATURES.md rows,
   CHANGELOG [Unreleased] section, gotchas-module-management.md (replace
   rationale).
8. **Release plumbing** — API golden regenerated (+8 exports:
   `VariationThreshold`, `NoisyMetricNames`, `RunRepeated`,
   `WriteBenchstatRepeated`, `MetricVariation`, `RepeatedResult`, +methods);
   cmd/cqrs-bench carries sanctioned relative sibling
   `replace benchkit/v4 => ../../benchkit` (pre-tag symbols; gate-checked).

### Verification matrix (all green)

| Gate | Result |
| --- | --- |
| benchkit full suite (workspace mode) | ok (53.5s) |
| benchkit suite GOWORK=off (published pins) | ok (38.8s) — closed during this review |
| benchkit race subset (new tests) | ok (51.7s) |
| cmd/cqrs-bench suite GOWORK=off (with replace) | ok (33.2s) |
| golangci-lint benchkit + cqrs-bench | 0 new issues |
| file-size ratchet (`#check-file-size`) | PASS (render.go shrank 580→461) |
| replace-directives gate | PASS |
| changelog-symbols gate (86 citations) | PASS |
| api-stability golden regen + `TestEvery` | PASS |
| doc-check (SKILL + references + AGENTS) | exit 0, 1142 refs valid |
| `nix fmt` idempotent | PASS |
| benchmark regression gate (local baseline, median, 25%) | PASS — 0 regressions, 3 improvements, 13 stable |
| duplication gate | my clone eliminated; gate still RED from pre-existing queue/* work (see d) |

---

## b) PARTIALLY DONE

1. **`compare` command** — per-backend median results DO carry
   `MetricVariation` when `--repeat N` is passed, and the comparison table's
   CoV column still shows throughput CoV only; no per-backend noisy-metric
   count column, no Variation section in markdown compare output.
2. **JSON/manifest output** — `Result.MetricVariation` serializes (new
   optional field), but the per-run `Runs` are NOT serialized by any CLI
   format (only `WriteBenchstatRepeated` consumes them). A `--format json`
   consumer cannot recover raw per-run data.
3. **FEATURES.md coverage line** — still says "88 benchkit + 12 CLI test
   functions"; this session added 11 benchkit + 1 CLI test functions. Count
   not refreshed (cosmetic drift).
4. **Legacy `Repeat*` fields kept** — `RepeatMin/Max/Samples/Mean/StdDev/
   CoV/IsReliable` now duplicate what `MetricVariation["write_throughput"]`
   carries. Kept deliberately for back-compat; will need a v5 decision.

## c) NOT STARTED (identified, deferred)

1. Nightly CI multi-sample benchstat artifacts (benchmarks.yml quick job now
   emits 3 samples/metric for free, but nothing feeds them to benchstat).
2. `benchstat` A/B-by-revision convenience (e.g. `scripts/bench-ab.sh
   <old-rev> <new-rev>`).
3. Per-metric MIN tracking (fast-path documentation).
4. `LoadAvg` at run END (drift indicator) and configurable noise threshold.
5. Soak-mode integration with variation (drift + cross-iteration CoV).
6. Tag wave for benchkit (drops the cqrs-bench sibling replace; needs push).

## d) TOTALLY FUCKED UP (honest accounting)

1. **Introduced a clone, then fixed it** — wrote `detectLoadAvg1` without
   grepping for existing `/proc/loadavg` readers; `load_aware_test.go` already
   had one. `#check-duplication` caught it (env_linux.go vs load_aware_test.go);
   refactored the test to reuse the production reader. Lesson: search for
   existing readers before adding a second one.
2. **Edited baselined files past their ratchet** — grew render.go 580→585 and
   runner.go past its 478 baseline before checking budgets; both gates failed
   mid-session. Fixes (extract run_render.go / environment.go) were correct
   improvements, but the budgets should have been checked BEFORE editing.
3. **Two tool-call fumbles** — a multiedit mangled `metrics_test.go`
   (`{	t.Parallel()` join) and a doc.go edit duplicated the `package` clause
   mid-file (syntax error). Both caught immediately by view/gofmt and fixed;
   ~2 wasted round trips each. Cause: constructing edits without verifying
   unique-anchor context.
4. **One build break** — `writeResult` re-plumb dropped the `err` declaration
   (`undefined: err`); one extra build cycle.
5. **Initial verification gap (closed during this review)** — benchkit was
   only tested in workspace mode until 02:07; the GOWORK=off leg (CI Module
   matrix shape) had not been run. Now green.
6. **Forgot benchkit/README.md** — the SDK's primary doc was not in the
   original doc pass (only CLI README, SKILL, doc.go, FEATURES). Caught during
   this self-review and fixed.

## e) WHAT WE SHOULD IMPROVE (systemic, from this session's observations)

1. **Pre-existing red gates make real regressions invisible** —
   `#check-duplication` has been red since ~10:42 2026-09-15 (19 new clone
   groups, all `queue/{postgres,sqlite,conformance}`) and lint flags
   `benchkit/soak_test.go:249` gocyclo 24>20. A permanently red gate is worse
   than no gate. Whoever lands module-scale work must re-pin/annotate in the
   same change.
2. **Benchstat was advertised but structurally unable to work** — the docs
   promised `benchstat old.txt new.txt` for ~2 months while the format could
   only emit single samples. "Does the documented workflow actually function?"
   should be a doc-check-class assertion (a tiny self-test running benchstat
   semantics over `--repeat 2` output would have caught it).
3. **Reported stats should always state what they DON'T cover** — the old
   "CoV" line was truthful for throughput and silent about everything else.
   The new `Variation:` section fixes benchkit; the same honesty rule should
   apply to every metrics surface (soak drift, calibration tables).
4. **Line-budget pre-flight** — before editing any file in
   `scripts/file-size-baseline.txt`, check `wc -l` against its baseline entry;
   extraction-first beats ratchet-failure-then-extract.
5. **Auto-commit daemon hides authorship** — this session's work is spread
   over ~10 `chore: auto-commit` commits; the CHANGELOG entry is the only
   reviewable narrative. Commit at phase boundaries when history matters.

## f) NEXT — up to 50 tasks, priority order

**Blockers / debt (do first)**
1. Resolve the 19 `queue/*` clone groups (annotate intentional dialect twins
   with `//art-dupl:accept` or consolidate) and re-pin
   `.art-dupl-baseline.json` — `#check-duplication` green again.
2. Fix `benchkit/soak_test.go:249` gocyclo (split the round-trip test) — lint
   green again.
3. Tag wave: cut benchkit (statistical-rigor APIs), bump cmd/cqrs-bench pin,
   strip its sibling replace (see gotchas-module-management.md entry).
4. Refresh FEATURES.md coverage line (benchkit/CLI test-function counts).
5. Root `nix run .#verify` full pass post-queue-fix (was skipped this session:
   exclusivity + time; targeted gates all ran).

**benchkit SDK**
6. `compare` table: per-backend noisy-metric count column + Variation footer.
7. Markdown compare output: variation summary section.
8. Serialize per-run data: `--format manifest` gains `runs[]` (opt-in flag to
   avoid size blowup).
9. `RepeatedResult` JSON writer (mirror of WriteBenchstatRepeated).
10. Track per-metric MIN (fast path) in LatencyCollector.
11. Record `LoadAvg1` at run end too; report load drift.
12. Configurable oversubscription threshold (Config knob).
13. Soak × variation: cross-iteration CoV next to drift metrics.
14. Reservoir size configurability per-phase (P99 fidelity at 10M+ events).
15. Percentile interpolation option for small-n runs (nearest-rank P50 is
   coarse below ~20 samples).
16. `tail_ratio` semantics for write_max_ns (true-max/P50 ratio).
17. Export `resultMetrics()` names as a public constant list (stable benchstat
    metric names for downstream tooling).
18. `RunSuite` (testing.B) variant that uses RunRepeated + b.ReportMetric per
    metric CoV.
19. Zero-value audit: a phase that records Count=0 but non-zero throughput
    (or inverse) should warn.

**CLI (cqrs-bench)**
20. `benchstat-diff` subcommand: run two revisions (worktrees), emit benchstat
    comparison table.
21. `--repeat` default guidance: warn when benchstat format used with repeat
    < 6 (benchstat wants ≥6 samples for CIs).
22. `--format csv`: add variation columns (CoV per key metric).
23. Sweep output: CoV column across the sweep's internal repeats.
24. `--strict` should also fail on NOISY headline metrics (opt-in flag).
25. Progress output: show per-repeat progress (currently per-phase only).
26. Table output: show `Load1` in env row when > 0.
27. `--warmup` docs: state that warmup uses a separate bundle (README gap).
28. `list-phases`: include which metrics each phase feeds.

**CI / gates**
29. Nightly job: capture `--repeat 10 --format benchstat` artifacts and run
    benchstat against previous nightly; post delta summary.
30. Regression gate: add a second gate set entry for a sqlite backend path
    (currently memory + turso matview only).
31. Add `LoadAvg1 > threshold` abort to benchmark-regression.sh (reuse
    calibration-gate semantics) so local runs refuse to compare on loud
    machines.
32. `check-bench-gate`: assert gate set entries still exist as benchmarks
    (guard against silent benchmark renames breaking the allowlist regex).
33. CI lint leg currently misses gocyclo in tests? (soak_test shipped red) —
    investigate version skew between local and CI golangci-lint.

**Docs / skill**
34. recipes.md: add a "statistical rigor" recipe block (RunRepeated +
    benchstat) — then classify it in recipes_catalog (compile harness).
35. faq.md: "why is my P100 1000x P99" entry (exact-max semantics).
36. readmodels.md/core.md: cross-link variation section where CoV mentioned.
37. AGENTS.md benchkit one-liner: mention RunRepeated/MetricVariation.
38. docs/benchmarks/: capture a fresh backend-comparison with repeats
    (current one is 2026-07-31, pre-variation).

**Queue module (from pre-existing red gate, not this session)**
39. Audit queue/postgres vs queue/sqlite 19 clones: shared core extraction or
   accept-annotations.
40. queue/conformance: consolidate the 4 `openEnv(t)` clone groups into a
   helper.

**Hygiene**
41. Baseline `benchmarks/benchmark-baseline.txt` is stale relative to today's
    improvements (3 improvements >5%) — re-pin on a quiet window after
    calibration-gate PASS (protocol: titled header).
42. Consider adding `P100` to benchstat gate metrics (tail regression
    detection) once enough samples exist.
43. `docs/status/README.md`: index this report.
44. Sweep: benchkit has `infertypeargs` hints (pre-existing) — one-line fixes.

## g) Questions I cannot answer myself

1. **Queue-module clones (task 1/39-40):** that module's 19 clone groups are
   from another session's auto-committed work. Do you want me to fix/annotate
   + re-pin the baseline now (touching code I didn't write), or should the
   owning session/backlog item handle it?
2. **Tag wave timing (task 3):** the cqrs-bench sibling replace and the
   untagged benchkit API are deliberate pre-release state. When do you want
   the next tag wave (I will not push tags without your go-ahead)?
3. **Prioritization:** of the NEXT list, is the A/B-by-revision benchstat
   workflow (tasks 20-21, 29) what you want next — or deeper per-metric
   gating in CI (tasks 30-32)?

---

*Session artifacts: benchkit/{repeat,repeat_test,environment,report_variation}.go
(new), metrics/artifacts/result/run/benchkit/runner/report/doc/env_linux/
env_other/load_aware_test.go, cmd/cqrs-bench/{main,output,render,run_render,
render_variation,main_test}.go + README + go.mod (replace), docs/api_surface.txt,
docs/benchmarks/README.md, docs/agents/gotchas-module-management.md, SKILL.md,
FEATURES.md, CHANGELOG.md, benchkit/README.md.*
