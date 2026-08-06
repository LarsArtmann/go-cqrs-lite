# Status Report: Full Benchmark Suite + Resident Memory Metric

**Date:** 2026-08-06 23:24 CEST  
**Session scope:** Benchmark tools and linter (user-scoped)  
**Prior session context:** `docs/status/2026-08-06_22-45_benchmarks-linter-green-metaengine-daemon.md`

---

## a) FULLY DONE

### 1. Full Benchmark Suite Run — 302 benchmarks across 25+ modules

Ran the complete `go test -bench=.` suite across the entire repo in 4 parallel groups. **301 PASS, 1 FAIL (fixed).**

| Group | Modules | Benchmarks | Wall Time | Result |
|-------|---------|------------|-----------|--------|
| Core Domain | event, command, query, decider, dispatcher, id, codec, dedup, schema, snapshot, catalog, metadata | 118 | ~312s | ALL PASS |
| Infrastructure | storage/memory, pebble, bbolt, view, storage, signing, encryption, middleware, transport/http, watermill, listing | 83 | ~143s | 1 FAIL (fixed) |
| Metaengine | core, pebbleengine, duckdbengine, pgengine, projectionadapter | 55 | ~313s | ALL PASS |
| Stack/Integration/Tooling | stack/bench, integration, benchkit, cqrs-bench, cqrs-lint | 46 | ~82s | ALL PASS |

**Headline numbers from the full run:**
- **Throughput king:** Memory event bus at **35.5M events/sec**
- **Zero-allocation champion:** `PayloadReadOnly` at **0.78ns/op, 0 allocs**
- **Fastest codec:** `RawCodec` at **17ns encode / 28ns decode** (vs JSON 280/376ns, CBOR 210/404ns)
- **Heaviest single bench:** `BenchmarkBenchkitSuite_DuckDB` at **8.6s** (full 13-phase end-to-end)
- **Scale test:** 10K streams x 100 events = **1M events in 758ms** (1.3M ev/s sustained)

### 2. Backend Comparison via cqrs-bench CLI (dev profile)

Ran all 4 embedded backends head-to-head:

| Backend | Raw Sink | Write | Cold Read P50 | DB Size | Write Amp |
|---------|----------|-------|---------------|---------|-----------|
| memory | **2.2M ev/s** | 139k ev/s | 630ns | — | — |
| pebble | 122k ev/s | 64k ev/s | 20us | 2.0 MiB | 16x |
| bbolt | 41k ev/s | 36k ev/s | 20us | 8.0 MiB | 66x |
| sqlite | 22k ev/s | 17k ev/s | 33us | 5.4 MiB | 45x |

### 3. Fixed BenchmarkPebbleStore_Save100

**Bug:** Each iteration created a new stream via `id.NewStreamID()`, so expected version must always be `0`. But the test passed `event.Version(i*100)`, causing a version conflict on every iteration after the first.

**Fix:** Changed `event.Version(i*100)` → `event.Version(0)`, cleaned up unused loop variable `i` → `range b.N`.

**File:** `storage/pebble/benchmark_test.go:69`

**After fix:** 269us/op, 144KB/op, 1942 allocs/op — passing.

### 4. Added Resident Memory Metric to benchkit

**Problem:** The old `Memory.Delta` (peak - baseline) was misleading. On small workloads, GC cleans up write-phase temporaries before the sampler catches the peak, so Delta shows `0 B` — useless. Peak heap (~2-4 MiB) is dominated by Go runtime overhead and is nearly identical across backends — also useless for comparison.

**Solution:** A new `Memory.Resident` field that measures the **actual RAM cost of stored data**. After all benchmark phases complete, force `runtime.GC()` twice, then snapshot `HeapAlloc`. The delta from baseline is the settled heap footprint — the metric for capacity planning.

**Files changed (4 files):**

| File | Change |
|------|--------|
| `benchkit/result.go:278` | Added `Resident uint64` to `ResourceStats` struct with doc comment |
| `benchkit/runner_concurrent.go:19-28` | Force double GC, measure settled heap, compute resident delta |
| `benchkit/report.go:336` | Text report: `RAM: X resident (post-GC data footprint)` replaces misleading `Delta: X` |
| `benchkit/report_comparison.go` | Comparison text table + Markdown: new `RAM` column between `CoV%` and `Heap` |
| `cmd/cqrs-bench/render.go` | Styled table/CSV/TSV/markdown: new `RAM` column + `RAM Resident` row in single-run summary |

**Results at dev profile (500 events):**
- memory: **896 KiB resident** (data lives in Go heap)
- sqlite/pebble/bbolt: **0 B resident** (data flushed to disk, GC reclaimed Go objects)

**Results at small profile (10K events):**
- memory: **20 MiB resident** (linear scaling with event count)
- pebble: **44 KiB resident** (minimal Go heap — just LSM caches)
- sqlite/bbolt: **0 B resident** (pure disk)

### 5. Daemon Already Committed Our Work

The auto-commit daemon committed both fixes within minutes:
- `64cc6b715` — Pebble benchmark fix
- `afb41634a` — Resident memory metric (benchkit + render + report)

No uncommitted changes remain from this session.

---

## b) PARTIALLY DONE

### Nothing partially done from this session's scope

Everything we touched is either fully done and committed, or was already done in the prior session.

---

## c) NOT STARTED

### Out of scope (daemon's domain — metaengine v2 refactor)
- Metaengine Phase 2: Record type extraction (daemon actively working — commits `a4e8a9d9b`, `6efdaeabd`)
- Metaengine Phase 3: graphadapter module (daemon added — commit `09c4b7fe8`)
- badgerengine module (daemon added — commit `df23a4b58`)
- `nix run .#verify` clean run — still blocked by daemon race condition
- API stability golden regeneration after daemon's new modules
- `nix fmt` on final state

### Not started from this session's observations
- See section (e) for the full improvement list

---

## d) TOTALLY FUCKED UP

### 1. The resident memory metric has a measurement noise problem on disk backends

The double-GC approach shows **0 B** for sqlite/bbolt and **44 KiB** for pebble. The 0 B result is technically correct (data is on disk), but it's **uninformative** — the user sees "0 B" and wonders if the measurement is broken. The real issue: after GC, all the Go-side objects (event structs, payload byte slices) are collected, leaving literally nothing in heap for disk-backed stores. This is correct but not useful for comparison.

**Better would be:** Show `Memory.Resident` as the Go heap data footprint, and separately note in the report that disk backends store data outside the Go heap (so resident heap ≈ 0 is expected, not a bug). Or: show a combined "effective footprint" that sums `Memory.Resident + Disk.DatabaseBytes` for a true apples-to-apples comparison.

### 2. I didn't verify the Markdown comparison output renders correctly

I edited the Markdown table format in `report_comparison.go` to add a `RAM` column, but I only tested the text format and the styled table format. The Markdown separator row and header alignment might be off. This is a "should have tested" gap.

### 3. The comparison table is getting too wide

Adding RAM as a 14th column pushes the text comparison table to 140 characters wide. On standard terminal widths (80-120 chars), this wraps and becomes unreadable. The table was already wide at 13 columns; 14 is worse. This is a UX regression I introduced without flagging.

### 4. The cqrs-bench compare output shows daemon's new OnRecord work

The daemon's commit `03733594a refactor(bench): thread context through factory creation and document OnRecord` and the projectionadapter changes in `afb41634a` mean the benchmark factory now has Record-related context. I didn't review or verify this code — it's the daemon's work mixed into our benchmark infrastructure. This could cause issues if the Record API is unstable.

---

## e) WHAT WE SHOULD IMPROVE

### Benchmark Infrastructure

1. **Add a "Total Cost" column to comparison tables** — sum of `Memory.Resident + Disk.DatabaseBytes` for a true apples-to-apples capacity comparison. Currently you have to mentally add RAM + Disk.
2. **Add `--profile custom` flag** to cqrs-bench for user-specified stream/event/goroutine counts — right now you're limited to the 7 hardcoded profiles.
3. **Add `-count=N` flag to cqrs-bench** — the benchkit library supports repeat runs with CoV analysis, but the CLI doesn't expose it. CoV < 5% is the trustworthiness threshold.
4. **Add benchmark output comparison (benchstat mode)** — the `--format benchstat` exists but there's no `cqrs-bench diff` command to compare two runs side-by-side with statistical significance.
5. **Run benchmarks with `-race` flag as a separate profile** — race detector catches data races that correctness tests miss, but it 5-10x inflates timings, so it should be a separate mode, not the default.
6. **Add a `--filter` flag to cqrs-bench** to run only specific phases (e.g., `--filter write,read` to skip projection/snapshot/metaengine). Currently you always run all 13 phases.
7. **The Pebble benchmark Save100 is the only benchmark testing batch saves** — add Save1000, Save10000 to stress batch insert paths.
8. **Add concurrent write benchmarks for pebble/bbolt/sqlite** — the memory backend has contention benchmarks, but disk backends don't. Disk backends have very different contention profiles (single-writer for bbolt, MVCC for sqlite).
9. **Add a benchmark for the tombstone/listing path** — `listing/` has benchmarks but no backend comparison. Tombstone detection is O(n) on every Load.
10. **DuckDB benchmarks take 216 seconds** — that's 40% of the total benchmark time for one module. Consider splitting or marking the calibration benchmarks as `long` so they can be skipped.

### Resident Memory Metric (what we just built)

11. **The double `runtime.GC()` is aggressive** — it forces two full GC cycles, which adds ~2-5ms to every benchmark run. For the dev profile (24ms total), that's 10-20% overhead. Consider measuring once at the end of all phases rather than per-run.
12. **Resident memory doesn't account for Go runtime overhead** — the baseline includes Go runtime structures, so subtracting it removes some legitimate overhead. The resident number is "data-only" which is what we want, but document this clearly.
13. **Add a `--memory-profile` flag** that writes a `pprof` heap profile alongside the benchmark — so users can drill into what's consuming RAM.
14. **Show bytes/event for resident memory** — "896 KiB for 500 events" is clear, but "1.8 KiB/event" is more comparable across profiles.

### cqrs-bench CLI

15. **Add `cqrs-bench list-profiles` command** — currently profiles are documented in `--help` but there's no way to see the exact stream/event/goroutine counts without reading source.
16. **Add `cqrs-bench list-backends` command** — same issue; backends are in `--help` but not queryable.
17. **The compare command doesn't show which backend "wins" each metric** — the `comparisonWinnerSummary` function exists in render.go but it's not wired to all output formats.
18. **Add CSV/TSV export of comparison results** — the format exists but compare mode only outputs text/markdown/table.
19. **Add `--warmup` flag** — currently the first benchmark run includes JIT/cache-warming overhead. A warmup run that's discarded would give cleaner numbers.
20. **The metaengine phase is always skipped** — the warning "bundle has no MetaEngine" appears for every backend. Either wire it up or suppress the warning when no backend supports it.

### Benchmark Correctness

21. **The `makeBenchEvents` helper creates events with nil payloads** — real events have 256B+ payloads. The benchmark underestimates serialization cost.
22. **BenchmarkPebbleStore_SaveLoad100 pre-saves then only benchmarks Load** — the Save100 benchmarks Save, the SaveLoad100 benchmarks Load after Save. But there's no benchmark that measures the full Save+Load round-trip latency.
23. **The `openBenchStore` helper uses `slog.LevelError`** — this suppresses all logging, which is correct for benchmarks, but it means we never catch log-level error messages that might indicate problems.
24. **Benchmarks don't test CBOR codec path** — all use JSON. The codec benchmarks show CBOR is 25% faster on encode, but the storage benchmarks don't exercise it.

### General

25. **Regenerate api-stability golden** — the daemon added new modules (badgerengine, graphadapter, record) that need to be in the api-stability modules list.
26. **Run `nix fmt`** on the final file state — the daemon may have reformatted our edits.
27. **The `Memory.Delta` field is now redundant** — we replaced it with `Resident` in the display, but it's still computed and serialized in JSON. Consider deprecating it.
28. **Add resident memory to the JSON output schema documentation** — if there's a schema doc, the new field needs documenting.
29. **Test that the Markdown comparison output is valid** — I changed column count but didn't verify the Markdown renders.
30. **The comparison table column width (140 chars) needs responsive handling** — truncate or abbreviate column names for narrow terminals.

---

## f) Up to 50 Things We Should Get Done Next

### Benchmark Suite Improvements (1-15)
1. Add "Total Cost" column (Resident + Disk) to comparison tables
2. Add `--profile custom` flag for user-specified workload parameters
3. Add `-count=N` flag to cqrs-bench for repeat/CoV analysis
4. Add `cqrs-bench diff` command for statistical comparison of two runs
5. Add `--filter` flag to run specific benchmark phases only
6. Add concurrent write benchmarks for disk backends (pebble, bbolt, sqlite)
7. Add Save1000/Save10000 batch benchmarks for all backends
8. Add `-race` benchmark mode as a separate profile
9. Split DuckDB calibration benchmarks into a `long` sub-suite
10. Add benchmark for tombstone/listing detection path across backends
11. Add `--warmup` flag for cache-warming discard run
12. Add bytes/event normalization for all resource metrics
13. Add `--memory-profile` flag for pprof heap output
14. Add `cqrs-bench list-profiles` and `list-backends` commands
15. Wire `comparisonWinnerSummary` to all output formats

### Resident Memory Metric Polish (16-20)
16. Document that disk backends showing 0 B resident is expected behavior
17. Suppress or remove the now-redundant `Memory.Delta` field
18. Verify Markdown comparison output renders correctly with the new column
19. Consider abbreviating column headers for narrow terminals (RAM vs Resident)
20. Add resident memory to benchkit test assertions (verify it's non-zero for memory backend)

### Verify Gate & Daemon Coordination (21-25)
21. Run `nix run .#verify` once daemon stabilizes
22. Regenerate api-stability golden for new daemon modules (badgerengine, graphadapter, record)
23. Add new daemon modules to api-stability modules list if not already there
24. Run `nix fmt` on final file state
25. Check if the daemon's Record API changes affect benchmark factory correctness

### Pebble Benchmark Specifics (26-30)
26. Add a Save+Load round-trip benchmark (not just Save-only or Load-only)
27. Add benchmarks with realistic 256B+ payloads (current uses nil)
28. Add a benchmark for Pebble checkpoint/snapshot operations
29. Add a benchmark for Pebble KV store adapter
30. Add concurrent read benchmarks for Pebble (currently only concurrent for memory)

### cqrs-bench CLI Features (31-35)
31. Add `--codec cbor` to compare mode (currently JSON-only in compare)
32. Add CSV/TSV export for comparison results
33. Add `--quiet` flag to suppress per-phase progress lines in compare mode
34. Add JSON schema validation for `--format manifest` output
35. Add `cqrs-bench version --verbose` showing build info, Go version, CGo status

### Benchmark Coverage Gaps (36-40)
36. Add benchmarks for signing + encryption middleware overhead (signing has benchmarks, but not cross-backend)
37. Add benchmarks for projection host throughput (events/sec through projection pipeline)
38. Add benchmarks for scheduling/timer store operations
39. Add benchmarks for graph projection (MergeNode/MergeEdge throughput)
40. Add benchmarks for relational projection (multi-table upsert throughput)

### Metaengine Benchmarks (41-45)
41. Add Iroh transport benchmarks (InProcess vs Loopback vs QUIC throughput)
42. Add cross-engine comparison benchmarks (same ADT, different engines, side-by-side)
43. Add metaengine persistence benchmarks (volatile vs persistent, startup cost)
44. Add metaengine replication cost benchmarks (RTT impact on latency)
45. Add DuckDB columnar vs SQLite row-store comparison at scale (100K+ rows)

### Documentation & Observability (46-50)
46. Document the benchmark methodology (warmup, GC impact, CoV thresholds) in a BENCHMARKING.md
47. Add a benchmark regression dashboard (track key metrics over commits)
48. Document the backend selection decision matrix (when to use pebble vs sqlite vs bbolt)
49. Add `cqrs-bench doctor` command showing detected CPU/memory/CGo capabilities
50. Create a benchmark CI gate that fails when throughput drops > 20% from baseline

---

## g) Questions for the User

### Q1: Should the comparison table replace `Heap` (peak) entirely with `RAM` (resident), or keep both columns?

The table is now 14 columns wide (140 chars). Peak heap is noisy and nearly identical across backends, but it catches memory leaks. Resident is the capacity-planning metric. Keeping both is most informative but makes the table unreadably wide. Dropping peak heap loses leak detection. **I cannot decide this tradeoff for you** — it depends on whether you use the comparison table for capacity planning (drop peak) or regression detection (keep peak).

### Q2: Should I add the "Total Cost" column (Resident RAM + Disk bytes) or is that over-engineering?

The "Total Cost" column would give a single number: "memory backend costs 20 MiB total (all RAM), sqlite costs 5.4 MiB total (all disk), pebble costs 2.0 MiB total (all disk + 44 KiB cache)." This makes apples-to-apples comparison trivial. But it conflates two fundamentally different resources (RAM vs disk) that have very different cost characteristics (RAM is 100x more expensive per GB than SSD). **I cannot decide whether merging them into one number is helpful or misleading** — it depends on your deployment context (cloud where RAM is expensive, or homelab where disk is the bottleneck).

### Q3: The daemon committed our work before we could commit it ourselves — is this the intended workflow?

The auto-commit daemon picked up our uncommitted changes and committed them with its own message (`afb41634a`). This means we lost control of the commit message and timing. The daemon's message is good, but the principle is concerning: if I'm making a deliberate change and the daemon commits it mid-edit, the commit could capture a half-finished state. **Should I commit immediately after each change to beat the daemon, or is the daemon's behavior intentional and I should let it handle commits?**
