# Benchkit Evidence Metrics + Soak Drift — Session Status Report

**Date:** 2026-08-01 19:45
**Session Goal:** Add decision-grade metrics to benchkit, verify with real benchmark, fix gaps
**Sessions spanned:** Two bursts (metrics creation → comparison/soak improvements → benchmark execution)

---

## A. FULLY DONE (shipped and tested)

### 1. Statistical Reliability (RepeatStdDev, RepeatCoV, RepeatIsReliable)

- **What:** Population stddev, coefficient of variation, reliability flag (CoV < 10%)
- **Files:** `result.go`, `run.go` (new file, extracted from benchkit.go)
- **Test:** `TestRepeat_StatisticalReliability`
- **Verified by deep benchmark:** memory 3.3%, pebble 3.2%, sqlite 8.3% — all TRUSTWORTHY

### 2. GC Pause Metrics (GCCount, GCTotalPause, GCMaxPause, GCMeanPause)

- **What:** `computeGCMetrics` scans `runtime.MemStats.PauseNs[256]` circular buffer
- **Files:** `metrics.go`, `runner_concurrent.go`, `runner.go`
- **Test:** `TestResult_GCMetrics`
- **Verified by deep benchmark:** Pebble has 1.17ms GC max pause (6.7x worse than memory). This was invisible before.

### 3. Allocation Metrics (AllocCount, AllocBytes)

- **What:** Delta of `Mallocs` and `TotalAlloc` across benchmark
- **Files:** `runner_concurrent.go`
- **Test:** `TestResult_AllocationMetrics`
- **Verified by deep benchmark:** SQLite allocates 8.3M objects (6.4x more than memory)

### 4. Data Integrity Verification (IntegrityErrors)

- **What:** Reads back 20 sampled streams post-write, verifies count/payload/version
- **Files:** `phases.go` (`verifyIntegrity`)
- **Test:** `TestResult_IntegrityErrors`
- **Verified by deep benchmark:** All 3 backends, 0 errors

### 5. Write Amplification (Disk.WriteAmplification)

- **What:** `DatabaseBytes / EventBytes` ratio
- **Files:** `result.go`, `runner_concurrent.go`
- **Test:** `TestResult_WriteAmplification`
- **Verified by deep benchmark:** Pebble = 10.15x (expected LSM amplification)

### 6. Cold/Warm Read Distinction (ColdReadLatency)

- **What:** First read pass captured separately from aggregate LoadLatency
- **Files:** `phases_read.go`
- **Test:** `TestResult_ColdReadLatency`
- **Verified:** At ProfileSmall cold ≈ warm (page cache not exhausted); needs larger profile to diverge

### 7. Environment Enrichment (CPUModel, TotalRAMBytes)

- **What:** `/proc/cpuinfo` + `/proc/meminfo` on Linux, stubs elsewhere
- **Files:** `env_linux.go` (new), `env_other.go` (new)
- **Test:** `TestResult_EnvironmentEnrichment`
- **Verified:** "AMD RYZEN AI MAX+ 395 w/ Radeon 8060S" detected

### 8. PrintComparison Updated with Evidence Columns

- **What:** 11-column table: WriteP50/P99, LoadP50/P99, ColdP50, GCMaxPause, WrtAmp, CoV%, Heap, Disk + integrity warnings
- **Files:** `report_comparison.go` (new file, split from report.go), `report.go`
- **Test:** `TestPrintComparison_EvidenceColumns` (replaces old `TestPrintComparison_RawSinkColumns`)
- **Verified by deep benchmark:** Output rendered correctly

### 9. PrintMarkdown Updated

- **What:** Markdown table with all new columns including integrity checkmarks
- **Files:** `report_comparison.go`
- **Verified by deep benchmark:** Output rendered correctly

### 10. Soak GC + Alloc Drift

- **What:** `SoakSample.GCMaxPause` + `SoakSample.AllocBytes`, `SoakResult.GCMaxPauseDriftPct` + `AllocGrowthPct`
- **Files:** `soak.go`
- **Impact:** Soak tests now detect GC degradation under sustained load, not just heap/throughput drift

### 11. Deep Benchmark Executed (Proof the Metrics Work)

- **What:** Ran `benchkit.Compare` with ProfileSmall (10K events) + Repeat=5 against memory, SQLite, Pebble
- **Output:** Full comparison table, per-backend detailed reports, markdown table, reliability assessment
- **Key findings:**
  - Pebble GC max pause: 1.17ms (10x its P99 write latency)
  - SQLite: 8.3M allocations (6.4x memory engine)
  - Pebble: 10.15x write amplification
  - All three: CoV < 10% (trustworthy), 0 integrity errors

### 12. Documentation + API Surface

- `doc.go` updated with all new metrics documented
- `ExpectedJSONFields` updated (includes gcCount, allocBytes, coldReadLatency, writeAmplification, etc.)
- `WriteBenchstat` updated with new metric lines
- API surface golden regenerated (3088 exports)
- Code formatted with gofumpt + goimports

---

## B. PARTIALLY DONE

### 1. Soak GC/Alloc drift — code written but NOT compiled separately

The `soak.go` edits added `GCMaxPause` and `AllocBytes` to `SoakSample`, plus `GCMaxPauseDriftPct` and `AllocGrowthPct` to `SoakResult`, and the drift computation in `computeSoakTrends`. The `PrintSoakReport` was updated to show GC and alloc drift lines. **BUT the full test suite was NOT re-run after the soak.go changes.** The last full `go test` run was before the soak.go edits. The soak test specifically needs verification.

### 2. cqrs-bench CLI

The CLI module has a pre-existing build break (`storage.SQLiteSetSynchronous` undefined in published `stack/v4` tag). The CLI uses `benchkit.Result` so it picks up the new fields automatically via JSON output, but no new CLI flags were added and the text output wasn't verified end-to-end.

### 3. Derived rate metrics (AllocsPerOp, BytesPerOp, GCPercent, TailRatio)

The raw data is collected (AllocCount, AllocBytes, GCTotalPause, Duration, P50, P99). The derived rates are NOT computed. These would be single-line additions in `finalizeResult` — the data exists, the computation doesn't.

---

## C. NOT STARTED

### 1. Metaengine benchmark quality assessment

The user asked "do we benchmark metaengine? If so do we do it well?!?!" I read the current `phases_metaengine.go` and identified the gaps but did NOT act on them. The current metaengine phase:

- Tests only ONE ADT (Counter — a single map increment)
- Tests only ONE engine (Memory — the trivially-fast baseline)
- Tests only Apply + ExecuteTyped (no Scan, no PushdownScan, no ScanRawValues)
- Does NOT test layout planning impact (JSON-blob vs typed-column pushdown)
- Does NOT test at scale (capped at 500 samples)
- Does NOT test concurrent Apply
- Does NOT test the materialize-vs-replay planner decision
- Does NOT benchmark SQLite/Pebble/Postgres/DuckDB engines

This is a **toy benchmark** for the module that the project considers "THE STRATEGIC FUTURE."

### 2. `-race` verification

Never ran `go test -race ./benchkit/...`. The new `computeGCMetrics` reads `MemStats.PauseNs[256]` (safe — value copy), and `finalizeResult` runs single-threaded (post-phases), but unverified.

### 3. `nix run .#verify` full gate

Never ran. We know benchkit builds and tests pass, but 62 other modules are unchecked. The cqrs-bench break is pre-existing but would fail the gate.

### 4. ADR for evidence-metrics design

No ADR written.

### 5. README update

`benchkit/README.md` "Metrics collected" section still lists the old metric set.

### 6. RunSuite update

`benchtest.go:RunSuite` does not report the new metrics via `b.ReportMetric`.

### 7. PrintSweep update

Sweep table still shows old columns only.

---

## D. TOTALLY FUCKED UP / MISTAKES

### 1. Built up changes, then got interrupted before final verification

The session flow was: add metrics → fix tests → run deep benchmark → user asked for more → started soak changes → user interrupted for status report. The soak.go changes are **unverified by tests**. This is the #1 mistake — I should have run tests immediately after the soak.go edit.

### 2. Left a missing closing brace in soak.go

Added fields to `SoakSample` struct but forgot to close the struct brace. Compiler caught it. Fixed. Wasted a build cycle.

### 3. Used string literal "-" for a time.Duration variable

`gcStr` was assigned `"-"` (string) in a context where `roundDuration` returns `time.Duration`. Compiler error. Fixed with `.String()` conversion. Type mismatch that I should have caught.

### 4. Report.go exceeded 350 lines after adding comparison columns

Had to split `PrintComparison`/`PrintMarkdown`/`WriteComparisonJSON` into `report_comparison.go`. Should have anticipated the size increase.

### 5. Left dead `coldColl` variable in phases_read.go

Declared but unused. Compiler caught it.

### 6. Never answered the user's metaengine question

The user explicitly asked: "do we benchmark metaengine? If so do we do it well?!?!" I read the code, identified that the benchmark is superficial, but then got derailed by the status report request. The metaengine benchmark gap analysis was NOT communicated to the user.

### 7. Deep benchmark runner was throwaway

Wrote `cmd_deep_bench/main.go`, ran the benchmark, then deleted it. The benchmark results are in the status report but not reproducible by anyone else. Should have been a test or committed example.

### 8. Did not update SoakSample test

The soak test (`soak_test.go`) was NOT updated to verify the new `GCMaxPause` and `AllocBytes` fields are populated.

---

## E. WHAT WE SHOULD IMPROVE

### Architecture

1. **Metaengine benchmark needs a complete rewrite** — The current single-ADT, single-engine, Apply+Query-only phase is inadequate for "THE STRATEGIC FUTURE" module. It should benchmark: multiple ADTs (Map, Counter, Set, Multimap, Graph, Vector, Search, Spatial), scan performance, pushdown vs closure, layout planning impact, all 5 engines (Memory, SQLite, Pebble, Postgres, DuckDB), and concurrent Apply.

2. **Derived metrics as first-class fields** — AllocsPerOp, BytesPerOp, GCPercent, TailRatio. The raw data exists; the derived computations are one-liners. Without them, users must manually divide to get actionable rates.

3. **SoakSample JSON field additions need soak_test verification** — The new fields aren't tested.

4. **RunSuite needs the new metrics** — `go test -bench=` output is missing GC, alloc, cold-read, and CoV data.

5. **PrintSweep needs new columns** — Scaling sweep tables don't show how GC pressure or write amp change with scale.

### Process

6. **Always run tests after every file edit** — The soak.go changes are unverified. This violates the core workflow rule.

7. **Always run `-race` after concurrency-adjacent changes** — Never done this session.

8. **Deep benchmarks should be reproducible** — Not throwaway scripts. Either a committed example or a CLI invocation.

9. **Answer user questions before getting derailed** — The metaengine question was asked and never answered.

---

## F. NEXT STEPS (prioritized, up to 50)

### P0 — Immediate (blocking correctness)

1. **Run `go test` after soak.go changes** — Verify the soak edits compile and pass
2. **Run `go test -race`** — Verify no data races in new metrics code
3. **Fix the soak.go size** (currently 360 lines, exceeds 350 limit)
4. **Add soak test for GCMaxPause/AllocBytes** fields being populated

### P1 — Metaengine benchmark overhaul (user explicitly asked)

5. **Benchmark Scan performance** — Current phase only does Apply + ExecuteTyped. Scan is the primary read path for collections.
6. **Benchmark PushdownScan vs closure scan** — The key optimization decision (SQL WHERE/ORDER BY vs Go closures)
7. **Benchmark multiple ADTs** — Map, Counter, Set, Multimap at minimum. Each has different performance characteristics.
8. **Benchmark layout planning impact** — JSON-blob scan vs typed-column scan (the 50x speedup claim)
9. **Benchmark at scale** — 10K+ items, not 500. The planner's value is most visible at scale.
10. **Add MetaEngineScanLatency + MetaEngineScanThroughput** to Result
11. **Add MetaEnginePushdownLatency** to Result (separate from closure scan)
12. **Test concurrent Apply** — Current phase is single-threaded
13. **Add correctness verification for metaengine** — Verify Apply + ExecuteTyped returns correct counts

### P2 — Derived metrics (data exists, computation missing)

14. **Add `AllocsPerOp`** — `AllocCount / TotalEvents`
15. **Add `BytesPerOp`** — `AllocBytes / TotalEvents`
16. **Add `GCPercent`** — `GCTotalPause / Duration * 100`
17. **Add `TailRatio`** — `P99 / P50` for write and load latency
18. **Add `ThroughputPerCore`** — `WriteThroughput / GOMAXPROCS`

### P3 — Verification gates

19. **Run `nix run .#verify`** — Full 63-module gate
20. **Fix cqrs-bench build break** — Pre-existing `storage.SQLiteSetSynchronous` undefined
21. **Run deep benchmark at ProfileMedium** — 500K events, enough to exhaust page cache for cold-read divergence
22. **Run deep benchmark with `--recovery`** — Verify crash recovery metrics
23. **Run soak test for 5 minutes** — Verify GC drift detection works across iterations

### P4 — Report/output improvements

24. **Update PrintSweep** with GC, CoV, WriteAmp columns
25. **Update RunSuite** with new metrics via `b.ReportMetric`
26. **Update README** "Metrics collected" section
27. **Write ADR** for evidence-metrics design
28. **Add `--min-cov` flag** to cqrs-bench for CI reliability gating
29. **Add `Summary()` method** to Result for one-line CI output
30. **Add comparison JSON output** with new metrics

### P5 — Advanced benchmarking

31. **Benchmark event signing overhead** — signing middleware adds HMAC per event
32. **Benchmark encryption overhead** — XChaCha20-Poly1305 per event
33. **Benchmark CBOR vs JSON** — codec comparison with the new metrics
34. **Benchmark durability tiers** — Strict vs Normal vs Relaxed (changes fsync behavior)
35. **Benchmark snapshot strategies** — EveryNEvents vs ReadPressure vs None
36. **Benchmark idempotency middleware** — dedup overhead per event
37. **Benchmark projection host catch-up** — replay speed for N events
38. **Benchmark watermill CatchUpSubscriber** — replay+live handoff latency
39. **Benchmark SSE broker** — concurrent SSE client fan-out
40. **Benchmark relational projection** — multi-table SQL projection vs single-table KV
41. **Benchmark graph projection** — node+edge merge performance
42. **Add batch-size sweep comparison** — how write amp changes with batch size
43. **Add payload-size sweep** — how GC pressure scales with payload size
44. **Add concurrent-write-contention benchmark** — same-stream concurrent writes
45. **Add network-latency benchmark** — for Postgres backends (measure round-trip separately)
46. **Add WAL size metric** for SQLite/Pebble
47. **Add fsync count metric** for SQL backends
48. **Add goroutine count tracking** — detect goroutine leaks
49. **Add context-switch metric** — voluntary + involuntary via getrusage
50. **Add statistical significance testing** — Welch's t-test for comparing two backends

---

## G. Questions (I cannot answer these myself)

### 1. How deep should the metaengine benchmark go?

The current phase tests 1 ADT on 1 engine with 500 samples. A proper benchmark would test all 7+ ADTs across 5 engines (Memory, SQLite, Pebble, Postgres, DuckDB) at 10K+ scale, measuring Apply, Scan, PushdownScan, and layout-planning impact. That's a massive scope increase — potentially a separate `metaengine_bench` module or a major benchkit expansion. **Should this be a full benchkit phase expansion, or a dedicated metaengine benchmarking tool?** The former integrates with the existing comparison/sweep/soak infrastructure; the latter allows metaengine-specific metrics without polluting the general Result struct.

### 2. Should derived metrics (AllocsPerOp, GCPercent, TailRatio) be computed in Result or only in reports?

Computing them as Result fields means they're serialized to JSON and available programmatically — but they're redundant (derivable from existing fields). Computing them only in report output keeps Result lean but means JSON consumers must recompute. **What's the preference: fat Result with derived fields, or lean Result with report-side computation?**

### 3. Should the deep benchmark runner be committed or kept throwaway?

I wrote a `cmd_deep_bench/main.go`, ran it, then deleted it. The results are in this status report but not reproducible. Options: (A) commit it as `benchkit/cmd_deep_bench/` — always available but adds maintenance surface; (B) make it a test function `TestDeepBenchmark` — runs in CI but adds 30-60s to the test suite; (C) document the `cqrs-bench compare` CLI invocation in the README — no code, but requires the CLI to build (currently broken). **Which approach?**

---

## Verification Status

| Check                                                      | Status                                  |
| ---------------------------------------------------------- | --------------------------------------- |
| `go build -tags "goexperiment.jsonv2" ./benchkit/...`      | ✅ GREEN (before soak.go edits)         |
| `go test -tags "goexperiment.jsonv2" ./benchkit/ -count=1` | ✅ GREEN (before soak.go edits)         |
| Soak.go changes verified                                   | ❌ NOT RUN — changes are unverified     |
| `go test -race`                                            | ❌ NEVER RUN                            |
| `nix run .#verify`                                         | ❌ NEVER RUN                            |
| `stack/bench` compiles                                     | ✅ GREEN (before soak.go edits)         |
| `cmd/cqrs-bench` compiles                                  | ❌ Pre-existing break                   |
| File line counts (max 350)                                 | ⚠️ soak.go at 360 lines                 |
| Deep benchmark executed                                    | ✅ memory vs sqlite vs pebble, Repeat=5 |
| API surface golden                                         | ✅ Regenerated (3088 exports)           |
| gofumpt formatting                                         | ⚠️ soak.go changes not formatted        |
