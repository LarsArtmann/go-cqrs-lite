# Status: Benchkit Statistical-Rigor Tail — Completion (compare/serialization, per-metric CI gating, SDK polish) — 2026-09-19 15:37

> **RESOLVED-BY-ROUTING (2026-09-19 docs-health 8th pass):** struck items above = verified shipped (CHANGELOG 2026-09-19 ADR-0142/ADR-0143 entries; TODO_LIST `[x]` rows). Open remainder tracked in TODO_LIST "Metaengine Universal Storage Substrate": T18b load-sweep + benchmark re-baseline (quiet-window gated), T19–T21 (v5-gated), tag waves, claim-metrics parity owner decision. ARCHIVED.

> Session scope: execute the three open chunks of the benchkit statistical-rigor
> tail harvested from [`2026-09-16_02-09_benchmark-statistical-rigor.md`](2026-09-16_02-09_benchmark-statistical-rigor.md)
> §b/§f into TODO_LIST: (1) `compare` + serialization tail, (2) the
> decision-gated Benchstat CI workflow, (3) the SDK polish batch. This report
> covers only this session's work and what it surfaced. The auto-commit daemon
> absorbed the edits (expected).

## Verdict

All three TODO_LIST chunks are implemented, every export documented, and every
gate this session could run locally is green. The owner Q3 decision (Benchstat
CI workflow) is RESOLVED: per-metric CI gating chosen and shipped, with the
losing option (A/B-by-revision benchstat) documented as a possible future
convenience, not a gate. Two things remain deliberately open: a
decision-grade (quiet-machine) fresh capture, and the pre-existing benchkit
tag wave (which this session's +9 exports make more urgent, not less).

---

## a) FULLY DONE (implemented + verified this session)

### Chunk 1 — `compare` + serialization tail

1. **Compare table Noisy column** — `benchkit.Result.NoisyMetricCount()`
   (benchkit/repeat.go) feeds a new `Noisy` column in both the SDK text
   comparison (`PrintComparison`, benchkit/report_comparison.go) and the CLI
   go-output table / markdown / csv / tsv comparison
   (cmd/cqrs-bench/render.go `buildComparisonTable`, `fmtNoisyCount`).
   Single-run backends show `-`: no dispersion data is not a stability
   verdict. render_test.go updated (14 → 15 headers) and extended
   (`TestFmtNoisyCount`).
2. **Variation footer / summary in comparison output** —
   `benchkit.PrintComparisonVariation(w, results)` (benchkit/report_variation.go):
   one line per repeated backend, `n/m stable` + noisy metrics named with
   CoV, worst-first. Wired into `PrintComparison` (text footer) and both CLI
   compare paths (`formatTable` after the winner summary, `formatMarkdown`
   after the table).
3. **Markdown compare variation summary** — `benchkit.PrintMarkdown` gained
   the Noisy column and a `**Variation**` bullet section; the CLI markdown
   path renders the same via `PrintComparisonVariation`.
4. **Manifest `runs[]` (opt-in)** — `benchkit.SuiteManifest.Runs` +
   `NewManifestRepeated` + `WriteManifestRepeated` (benchkit/manifest.go);
   CLI `run --format manifest --include-runs` opts in (flags.go → main.go →
   output.go → run_render.go). Median-only default unchanged.
5. **`RepeatedResult.WriteRepeatedJSON`** — JSON mirror of
   `WriteBenchstatRepeated`: median + every run + dispersion, round-trip
   tested. Manifest block moved out of artifacts.go into manifest.go because
   artifacts.go was at the 350-line ratchet (349).

### Chunk 2 — Benchstat CI workflow decision (owner Q3) — RESOLVED

6. **Decision: per-metric CI gating** over A/B-by-revision benchstat.
   Rationale (recorded in CHANGELOG + TODO_LIST): the median gate's known
   weakness is a loud machine, not a missing A/B workflow — benchstat samples
   already work since 2026-09-16; a shell wrapper would automate a workflow
   that is already 90% manual-runnable, while the gate's false-positive risk
   was real and unfixed.
7. **Load gate** in scripts/benchmark-regression.sh — load1 AND load5 vs the
   CPU count (calibration-gate v2 burst-draining rule), ceiling
   `--max-load` (default `nproc`), escape `--skip-load-gate`, fault-injection
   hook `BENCH_GATE_LOADAVG_FILE` mirroring `CALIB_GATE_LOADAVG_FILE`.
   Aborts BEFORE any benchmarking; skipped in `--current` compare-only mode.
8. **Noise gate** — `noise_gate()` parses `cqrs-bench --format json`
   `metricVariation` (jq): a HEADLINE metric (default `write_throughput
   write_p50_ns write_p99_ns load_p50_ns`, `--noise-headline` overridable)
   whose CoV reaches `--noise-threshold` (default 10 =
   `benchkit.VariationThreshold`) FAILS the gate; non-headline noise warns.
   Unparseable JSON and repeat<2 results fail loudly with actionable
   remediation text. `--noise-only` runs just this gate.
9. **Sqlite gate-set entry rides along** — `stack/bench::
   BenchmarkBenchkitSuite_SQLite$` (anchored, dev profile like the memory
   entry) added to GATE_SETS and to the CI benchmarks.yml "Run gate
   benchmarks" step; CI also gained a "Noise gate" step before the median
   compare.
10. **Fixture tests** — 12 new cases in scripts/test-benchmark-regression.sh
    (noisy headline fails, non-headline warns, stable passes, unparseable
    fails, no-metricVariation fails, threshold override both ways, load-gate
    abort/override/quiet, noise-only combinations). `check-bench-gate` flake
    app gained `pkgs.jq`. **Mutation-tested twice**: corrupting the jq
    selector failed 4 tests; disabling the load-gate awk rule failed 2;
    restored green each time.
11. **Live end-to-end verification** — `./scripts/benchmark-regression.sh
    --noise-only` built cqrs-bench, ran sqlite dev × 5, load gate PASSED
    (load had dropped to ~11.8 < 32), noise gate correctly **FAILED** on
    `write_p99_ns CoV=20.9%` — this host really is too loud for
    decision-grade medians and the new gate proves it mechanically.

### Chunk 3 — SDK polish batch

12. **Per-metric MIN tracking** — `LatencyCollector` tracks the exact min per
    `Record` (reservoir eviction can lose neither end); new
    `LatencyStats.Min` field (additive JSON). `Mean/Min` now express
    scheduler+contention overhead.
13. **LoadAvg at run END + drift** — `Environment.LoadAvg1End` sampled in
    `finalizeResult`; `recordLoadEnd` warns when a quiet-start run BECAME
    oversubscribed mid-run (the pollution a start-only check cannot see).
14. **Configurable oversubscription threshold** — `Config.LoadWarnThreshold`
    (load-per-CPU ratio, default 1.0); `oversubscribed()` shared by start
    warning and end-drift warning.
15. **Percentile interpolation (opt-in)** —
    `WithInterpolatedPercentiles()`/`NewLatencyCollectorWithOptions`;
    `Config.InterpolatedPercentiles` + CLI `--interpolated-percentiles`
    threaded to all 31 collector call sites in 12 phase files via
    `r.newCollector()`. Default nearest-rank unchanged (baseline
    compatibility preserved); P50 lands identically, P90/P99 read between
    samples instead of collapsing onto the max at n=5.
16. **`benchkit.MetricNames()`** — the full stable benchstat metric-name
    universe in phase-grouped report order, fresh copy per call; resultMetrics
    refactored over a shared `allMetrics()` single source of truth.
17. **Soak × variation** — `SoakResult.ThroughputCoV`/`WriteP99CoV` via the
    existing `dispersion()`, printed next to endpoint drift in the soak
    report; exposes bimodal soaks that first→last drift hides.
18. **Zero-value audit** — `runner.auditZeroValues()` warns when a phase
    reports throughput without samples or samples without throughput
    (write, raw sink, batch write, metaengine-apply pairs).

### Chunk 3c — fresh capture

19. **`docs/benchmarks/2026-09-19_backend-comparison-variation.md`** — first
    capture in the variation format (memory/pebble/sqlite, small × 5). The
    machine was oversubscribed during capture; the header states so loudly,
    and the Variation section self-demonstrates (5/34 stable on memory,
    write_throughput CoV 11.7% = NOISY headline). Indexed from
    docs/benchmarks/README.md and FEATURES as the format example superseding
    the pre-variation 2026-07-31 shape.

### Gates, goldens, docs (all green)

20. **Verification matrix (this session)**:

| Gate | Result |
| --- | --- |
| benchkit full suite, workspace mode | ok (39.7s) |
| benchkit full suite, GOWORK=off | ok (48.3s) |
| cmd/cqrs-bench suite, GOWORK=off (sibling replace) | ok (20.2s) |
| benchkit race subset (new tests) | ok (5.9s) |
| New SDK tests (14 in benchkit/variation_rigor_test.go + CLI TestFmtNoisyCount) | all PASS |
| golangci-lint benchkit + cqrs-bench | 0 issues (1 self-introduced `varnamelen` fixed) |
| `nix fmt` (treefmt) | idempotent |
| file-size ratchet (`#check-file-size`) | PASS — no new offenders, no growth |
| duplication gate (`#check-duplication`) | PASS — 0 new clones |
| api-stability golden regen (+9 exports) + `TestEvery` | PASS |
| changelog-symbols gate (152 citations) | PASS |
| `check-bench-gate` via nix app (24 fixture cases) | PASS |
| doc-check (SKILL + references + AGENTS) | exit 0, 1183 refs valid |
| Live noise-gate run | correctly FAILED on noisy headline (see #11) |

21. **Docs shipped**: CHANGELOG `[Unreleased]` dated section (symbols
    gate-verified); benchkit/README.md rigor section (serialization, Min,
    interpolation, load-end, threshold, MetricNames, comparison variation) +
    soak CoV bullets; benchkit/doc.go both sections; cmd/cqrs-bench/README.md
    (two new flags + rigor bullets + CI-gate pointer); SKILL.md benchmarking
    paragraph; FEATURES.md (7 new rows, coverage line 151+43 → 164+44,
    capture link); docs/benchmarks/README.md index line; TODO_LIST.md — all
    three chunks checked off with evidence and the decision rationale.

## b) PARTIALLY DONE

1. **Decision-grade fresh capture** — the capture exists and is honest, but
   it was taken at load ~34-40/32 CPUs, so its deltas are flagged
   non-decision-grade by its own header. The quiet-window capture (after
   `scripts/calibration-gate.sh` PASS) remains open — ironically the new
   load/noise gates are exactly the tool to time it.
2. **Baseline staleness (pre-existing hygiene #41)** — untouched. This
   session's live noise run adds evidence the host is loud; the gate can now
   REFUSE to compare on such a window, which changes the re-pin protocol
   slightly (calibration PASS → gate PASS → `--save`).
3. **Min surfaced only in JSON/SDK** — `LatencyStats.Min` is tracked,
   serialized, and tested but not rendered in PrintReport latency lines or
   the CLI summary table (display churn deferred; report.go sits at its
   401-line baseline with 2 lines of headroom).
4. **`resultMetrics()` names as "public constants"** — delivered as
   `MetricNames()` (function returning a fresh copy, phase-grouped order)
   rather than individual exported constants; a const-per-name API was
   judged worse ergonomics, but the letter of the old TODO item differed.
5. **AGENTS.md benchkit one-liner (old status doc task #37)** — SKILL.md was
   updated; AGENTS.md itself was not touched (its benchkit mentions live in
   the Quick Reference table, which predates RunRepeated/MetricVariation).
6. **CI noise-gate build mode** — the gate's live path builds cqrs-bench
   from REPO_ROOT in workspace mode with `GOTOOLCHAIN=auto`; verified live
   locally and fixture-tested, but the CI-side invocation (fresh runner,
   preinstalled Go resolution) has not executed yet — first push to master
   is the real test.

## c) NOT STARTED (still open; attributed)

From the same 2026-09-16 tail (deliberately out of scope this session):
1. `benchstat-diff` subcommand / `scripts/bench-ab.sh` A/B-by-revision — the
   LOSING option; documented as future convenience only.
2. Nightly benchstat artifacts + delta summary (benchstat-by-revision
   companion).
3. `check-bench-gate`: assert gate-set entries still exist as benchmarks
   (silent-rename guard) — MORE valuable now that the set grew to 4 entries
   + a noise benchmark.
4. `--strict` fail on NOISY headline metrics (CLI opt-in).
5. CSV variation columns; sweep CoV column; per-repeat progress; table
   `Load1` env row; `--warmup` docs; `list-phases` metric mapping.
6. Reservoir size per-phase; `tail_ratio` semantics for `write_max_ns`;
   `RunSuite` RunRepeated variant (b.ReportMetric CoV).
7. Docs/skill tail: recipes.md statistical-rigor recipe block (+
   recipes_catalog classification), faq.md "why is my P100 1000x P99",
   readmodels/core cross-links.
8. Pre-existing: benchkit tag wave (now +9 more untagged exports), root
   `nix run .#verify` full pass, CI golangci version-skew investigation
   (#33), P100-in-benchstat-gate-metrics (#42), benchkit `infertypeargs`
   hints (#44).

## d) TOTALLY FUCKED UP (honest accounting)

1. **Five self-inflicted compile/lint breaks, all caught, ~one cycle each**:
   (a) the artifacts.go MetricNames refactor initially left `return out`
   dangling (`undefined: out`) — a hand-assembled two-part multiedit;
   (b) a stray `data, err := WriteManifestRepeated(...)` with the wrong
   return arity inside the new test file; (c) a literal `\n` escape bug in
   the markdown Variation Fprintf; (d) an accidental newline-join that
   corrupted `TestFmtCoVDash` in render_test.go from a careless
   whitespace-only edit; (e) `varnamelen` rejected `lo` — the repo's lint
   config was not consulted before declaring the SDK chunk done. Root cause
   across all five: composing large edits without verifying anchors, arities,
   and escapes first.
2. **Wasted a 6-minute baseline diagnostic run** — the first pre-change
   benchkit suite ran with `tail -2`, which hid WHICH tests failed; the
   failure turned out to be host-load flake (load 54/32 at the time), but I
   re-ran the whole suite before checking `/proc/loadavg` and before
   capturing failure names. Checking load first is free; I did it second.
3. **Nearly repeated the file-size lesson from the 02-09 session** — I added
   ~35 lines to artifacts.go before measuring it (349/350 cap); only a
   post-hoc `wc -l` caught it and forced the manifest.go extraction. Status
   doc lesson #4 ("line-budget pre-flight") was read and still almost
   repeated.
4. **First draft of the bash gate shipped three real bugs** (literal `\`+
   newline inside an echo string; a `--noise-only --current` combination
   that could exit 0 with NO noise source; a trap referencing an empty
   variable) — all caught by my own post-edit review pass, none by the
   fixture tests (which then got written to pin exactly those paths). The
   review pass worked; it should not have been needed.
5. **The session fought the host all day** — load 34-59/32 for most of it.
   Consequences: one false baseline FAIL + re-diagnosis cycle, slower suites
   (346s → 48s across the day for the same suite), and a knowingly-loud
   capture. The statistical-rigor tooling this session built is precisely
   what should schedule this work; I ran it by eyeball instead.
6. **Foreign working-tree diffs appeared mid-session** (~10 files:
   stack/{postgres,turso,duckdb}/preset.go, storage/pebble/*,
   metaengine/memory_engine.go, claiming/rich.go, cmd/cqrs-lint testdata —
   Go 1.27 embedded-literal restructuring). I verified they compile and left
   them alone per the never-revert-others'-changes rule, but I did not
   determine their origin (concurrent session vs auto-tooling). See
   question g1.

## e) WHAT WE SHOULD IMPROVE (systemic)

1. **Pre-flight checklist for edits**: file line-budget (`wc -l` vs
   scripts/file-size-baseline.txt) AND a lint-config glance for new
   identifiers — both failures this session were known lesson classes.
2. **First test run captures everything**: full log to a file, failure names
   extracted, host load checked — never `tail -2` a gate-relevant run.
   Pipeline masking (filters hiding failures) already has a repo lesson; the
   sibling rule is "don't truncate the diagnostic you're diagnosing with".
3. **Prefer `lsp_replace_symbol` for whole-function refactors**: the one
   build break came from hand-splitting a function across two multiedit
   hunks. Symbol-boundary tools exist precisely for this.
4. **Gate scripts deserve the same self-review pass as Go code**: flag-combination
   matrix (which flags co-occur), format-string escapes, trap/variable
   initialization. The fixture tests now pin these, but the ordering should
   be review → tests, not review after writing.
5. **Schedule verification behind calibration-gate**: on this shared host,
   `scripts/calibration-gate.sh` PASS should gate full-suite runs and
   captures the same way it gates baseline re-pins. The 346s-vs-48s suite
   delta is the cost of not doing this.
6. **Changelog-symbols gate works extremely well** — writing the CHANGELOG
   section with `pkg.Symbol` citations BEFORE finishing docs caught nothing
   this time, but the workflow (symbols first, prose second) is the right
   shape; keep it.

## f) NEXT — up to 50 tasks, priority order (this tail + what the session surfaced)

**Blockers / decisions (do first)**

1. Rule on the foreign embedded-literal diffs (g1): author? keep/commit or
   revert? They gate any clean `git status` for the benchkit work.
2. Benchkit tag wave (pre-existing #3, now +9 exports deeper): cut benchkit,
   bump cmd/cqrs-bench pin, strip the sibling replace. Push gated on owner
   go-ahead (g3-adjacent).
3. Quiet-window fresh capture: watch for `calibration-gate.sh` PASS, then
   re-run the compare command from the capture header; supersede the loud
   capture (keep it as the noisy-format example).
4. Root `nix run .#verify` full pass on the combined tree (pre-existing #5;
   my session ran targeted gates only, per #verify exclusivity + host load).
5. First CI run of the noise gate + sqlite gate leg (benchmarks.yml) — verify
   on a real runner: jq presence, Go 1.27 resolution, 15-min timeout fit.
6. Add the gate-set-rename guard (#32): `check-bench-gate` asserts each
   GATE_SETS regex still matches ≥1 `func Benchmark` and the noise benchmark
   still exists — silent renames must fail the fixture gate, not the CI job.
7. Baseline re-pin protocol update: fold the new load gate into the re-pin
   runbook (calibration PASS → gate PASS → `--save`), then re-pin the stale
   baseline (pre-existing #41).

**benchkit SDK (small, high-value)**

8. Render `Min` in PrintReport latency lines + CLI summary table (decide the
   report.go 2-line-headroom problem: extraction first).
9. `--strict` also fails on NOISY headline metrics (opt-in flag, #24).
10. `list-phases`: which metrics each phase feeds (#28).
11. Table env row shows `Load1` when > 0 (#26).
12. `--warmup` docs: separate-bundle semantics in README flag table (#27).
13. `--format csv`: variation columns (CoV per key metric) (#22).
14. Sweep output: CoV column across internal repeats (#23).
15. Progress: per-repeat progress for `--repeat N` (#25).
16. Reservoir size per-phase (P99 fidelity at 10M+ events) (#14).
17. `tail_ratio` semantics for `write_max_ns` (#16).
18. `RunSuite` variant using RunRepeated + b.ReportMetric per-metric CoV
    (#18).
19. Consider exporting individual metric-name constants IF a consumer asks
    (revisit the MetricNames()-as-function decision with a real use case).

**CI / gates**

20. Nightly benchstat artifacts + delta summary (#29) — the companion of the
    losing option; independent of it as pure artifact capture.
21. `scripts/bench-ab.sh` A/B-by-revision convenience (#20/#2) — only if the
    owner still wants it after the gating decision (g3).
22. Add `LoadAvg1` drift check (not just start) to the gate's noise
    benchmark — the SDK now records LoadAvg1End; the gate could parse it.
23. Investigate CI golangci-lint version skew (#33, pre-existing).
24. Consider `P100` in benchstat gate metrics once sample counts justify
    (#42).
~~25. benchkit `infertypeargs` one-liners (#44).~~ done 2026-09-19 — benchkit lint 0

**Docs / skill**

26. recipes.md statistical-rigor recipe block + recipes_catalog classification
    (#34 — the compile harness makes this M-sized, not S).
27. faq.md: "why is my P100 1000x P99" exact-max entry (#35).
28. readmodels.md/core.md cross-links where CoV is mentioned (#36).
29. AGENTS.md Quick Reference benchkit one-liner: mention
    RunRepeated/MetricVariation/noise gate (#37).
30. docs/benchmarks/: once a quiet capture lands, annotate the loud capture
    as superseded (docs-health ANNOTATE pass).

**Watch / verify (not code)**

31. Observe the first two CI regression-gate runs for noise-gate flakiness on
    shared runners; tune `--noise-threshold`/headline set with evidence
    before anyone loosens it.
32. Track whether the sqlite gate-set entry shifts CI job duration
    materially (15-min timeout budget).
33. Harvest section (f) into TODO_LIST per docs-health HARVEST routing
    (items 8-30 candidate rows; 1-7 stay as session-frontier items).

## g) Questions I cannot answer myself

1. **The ~10 foreign working-tree diffs** (stack/{postgres,turso,duckdb}/
   preset.go, storage/pebble/{store,checkpoint,command_store,query_store,
   snapshot}.go, metaengine/memory_engine.go, claiming/rich.go,
   cmd/cqrs-lint testdata — Go 1.27 embedded-literal restructuring I did not
   author, plus untracked `metaengine/memory_reset.go` and two 15-0x status
   files): are these a concurrent session's / your intentional work that I
   should leave uncommitted to the daemon, or debris to revert? They
   compile; I changed nothing about them.
2. **Capture policy**: keep the loud 2026-09-19 capture permanently as the
   "what noisy looks like" example alongside a future quiet capture, or
   supersede/replace it once a calibration-PASS window capture lands?
3. **The losing Benchstat-CI option**: with per-metric gating now shipped,
   do you still want `scripts/bench-ab.sh` / `benchstat-diff` (tasks #20-21)
   built this quarter, or is the manual `benchstat old.txt new.txt` workflow
   (documented, working) the permanent end state?

---

_Session artifacts: benchkit/{manifest.go, variation_rigor_test.go (new);
artifacts, metrics, repeat, report_variation, report_comparison, environment,
runner_concurrent, soak, soak_report, result, benchkit, doc, phases×12,
README} , cmd/cqrs-bench/{flags, main, output, run_render, render,
render_test, README}, scripts/{benchmark-regression.sh,
test-benchmark-regression.sh}, flake.nix, .github/workflows/benchmarks.yml,
docs/{api_surface.txt, benchmarks/2026-09-19_backend-comparison-variation.md
(new), benchmarks/README.md}, CHANGELOG, FEATURES, TODO_LIST, SKILL.md._
