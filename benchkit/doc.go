// Package benchkit provides a factory-driven benchmarking suite for
// [stack.Bundle] presets and [system.System] deployments — the performance
// equivalent of [stack/contracttest].
//
// A deployer provides a [Factory] (a function that returns a fresh
// *stack.Bundle) or a [SystemFactory] (via [FactoryFromSystem]/[AdaptSystem],
// which run the same suite against a *system.System deployment; unsupported
// capability phases are skipped with recorded warnings), and benchkit runs a
// suite of realistic write, read, read-model, and projection workloads while
// collecting latency percentiles, throughput, memory deltas, and storage
// footprint.
//
// # Quick start
//
//	result, err := benchkit.Run(ctx, benchkit.Config{
//	    Profile:     benchkit.ProfileDev,
//	    PayloadSize: 256,
//	}, func() (*stack.Bundle, error) {
//	    return sqlite.New(filepath.Join(dir, "bench.db"))
//	})
//	if err != nil { ... }
//	benchkit.PrintReport(os.Stdout, result)
//
// # Cross-backend comparison
//
//	results, err := benchkit.Compare(ctx, config, map[string]benchkit.Factory{
//	    "memory": func() (*stack.Bundle, error) { return memory.New() },
//	    "sqlite": func() (*stack.Bundle, error) { return sqlite.New(":memory:") },
//	})
//	benchkit.PrintComparison(os.Stdout, results)
//
// The tool never imports a backend driver. Switching backends is a one-line
// factory change, mirroring the library's deployer-first philosophy.
//
// # Metric boundaries
//
// benchkit measures distinct performance boundaries. Each metric names exactly
// what it times so results are never misinterpreted:
//
//   - RawSinkLatency / RawSinkThroughput — pre-built events timed against
//     EventSink.Save only. Event generation, payload encoding, ID creation,
//     and metadata construction happen BEFORE timing begins. This isolates
//     pure backend write capacity.
//
//   - WriteLatency / WriteThroughput — generated events timed including
//     generation + encoding + Save. This is the practical ingest cost.
//
//   - LoadLatency — stream load (EventSource.Load) latency percentiles.
//
//   - ReadAllTime / ReadFromTime — journal scan wall-clock time.
//
//   - ReadModelSet / ReadModelGet — raw kv.Store Set/Get latency.
//
//   - ProjectionLag / ProjectionEvents — projection host catch-up metrics.
//
//   - RecoveryTime / RecoveredEvents — crash-recovery replay time.
//
//   - JourneyLatency / JourneyProjectionLatency / JourneyQueryLatency —
//     end-to-end publish→projection→query round trip (M14). Each sample writes
//     one event to a fresh stream, synchronously projects it, and dispatches a
//     typed query. JourneySamples counts the round trips measured.
//
//   - QueryHitLatency / QueryMissLatency / QueryPaginatedLatency —
//     query.Dispatcher overhead for a cache hit (registered handler reads real
//     data), a miss (unregistered type → handler-not-found), and a paginated
//     result construction (M15). QueryCorrectnessErrors counts mismatches.
//
//   - SnapshotColdLatency / SnapshotLoadLatency / CacheMissLatency /
//     CacheHitLatency — decider Load under cold replay (full fold), snapshot
//     load (EveryNEvents), and state-cache hit/miss (M16).
//     SnapshotCorrectnessErrors counts state/version mismatches.
//
//   - MetaEngineApplyLatency / MetaEngineQueryLatency /
//     MetaEngineApplyThroughput — Counter ADT workload: planner write
//     overhead + ExecuteTyped read latency + sustained write throughput.
//     MetaEngineScanLatency: filtered collection scan (TypedReader.Scan with
//     WHERE clause). MetaEnginePointReadLatency: single-item point lookup
//     (TypedReader.Get). MetaEngineApplyConcurrent: concurrent write
//     throughput (contention test). MetaEngineScanResults: items returned by
//     scan (correctness check).
//     MetaEngineSQLiteScanLatency / MetaEngineSQLitePointReadLatency /
//     MetaEngineSQLiteApplyThroughput — same Map ADT workload run against the
//     SQLite engine (NewSQLiteEngine). Gives a direct Memory-vs-SQLite
//     comparison: Memory shows planner overhead with zero I/O, SQLite shows
//     the cost of SQL execution + json_extract pushdown.
//
//   - ColdReadLatency — first read-pass latency (OS page cache cold).
//     LoadLatency aggregates all passes (cold + warm). On SQLite/Pebble,
//     ColdReadLatency P50 may be 10x higher than warm LoadLatency P50.
//
//   - GCCount / GCMaxPause / GCTotalPause / GCMeanPause — garbage collection
//     pause metrics. GC pauses are the dominant cause of P99 latency spikes.
//     These reveal whether tail latency originates from the backend or Go's GC.
//
//   - AllocCount / AllocBytes — total heap allocations during the benchmark.
//     High alloc rates correlate with GC pressure and latency variance.
//
//   - AllocsPerOp / BytesPerOp — derived per-event allocation rates. Eliminates
//     manual division: directly comparable across profiles and backends.
//
//   - GCPercent — percentage of wall-clock time spent in GC pauses. >5% means
//     GC is a significant performance factor.
//
//   - TailRatio — LoadLatency.P99 / LoadLatency.P50. A ratio >3 means tail
//     latency is unpredictable: P50 looks fine but P99 users see 3x worse.
//
//   - WriteTailRatio — WriteLatency.P99 / WriteLatency.P50. Same concept as
//     TailRatio but for the write path. High ratios matter for ingestion pipelines.
//
//   - IntegrityErrors — count of events that failed read-back verification.
//     Zero means all written events round-trip correctly. Non-zero indicates
//     silent data corruption — a fast backend with integrity errors is worse
//     than a slow one without.
//
//   - Disk.WriteAmplification — ratio of on-disk bytes to logical event bytes.
//     1.0 = zero overhead; 3.0 = 3x write amplification. The key metric for
//     comparing storage efficiency across backends (LSM vs row-store vs columnar).
//
//   - RepeatStdDev / RepeatCoV / RepeatIsReliable — statistical reliability
//     of repeat-run results. CoV < 10% means results are trustworthy for
//     cross-backend comparison. When RepeatIsReliable=false, increase Repeat.
//     These cover WRITE THROUGHPUT only; MetricVariation extends the same
//     dispersion analysis to every measured metric.
//
//   - MetricVariation — per-metric cross-run dispersion (mean, population
//     StdDev, CoV, per-run Samples) for every measured metric, populated by
//     [RunRepeated] when Config.Repeat > 1. A metric whose CoV exceeds
//     [VariationThreshold] moved too much between runs for its median to be
//     decision-grade; [NoisyMetricNames] lists them worst-first.
//
//   - Environment.CPUModel / Environment.TotalRAMBytes / Environment.LoadAvg1 —
//     machine metadata for honest cross-machine comparisons. Different CPUs
//     and RAM amounts produce dramatically different latency numbers.
//     LoadAvg1 samples the 1-minute load average at run start: above NumCPU
//     the run records a warning, because its latencies then include scheduler
//     wait rather than pure backend cost.
//
// Every Result includes Environment metadata (GoVersion, NumCPU, GOMAXPROCS,
// GOOS, GOARCH) and the actual Workers count so comparisons across machines
// and configurations are honest.
//
// # Percentile semantics
//
// LatencyStats percentiles come from [LatencyCollector]: P50–P99 are
// nearest-rank estimates over a bounded reservoir sample, so they stay O(1)
// memory for arbitrarily large runs. Two statistics are exact even after the
// reservoir saturates: Mean (accumulated running sum) and P100 — the collector
// tracks the true maximum on every Record, so a single multi-second stall in a
// 10M-event run appears in the tail report instead of being evicted from the
// reservoir. Treat P100 as "worst observed", not as a stable percentile: it is
// dominated by the single worst scheduling hiccup.
//
// # Repeats and statistical rigor
//
// A single run answers "roughly how fast". Drawing conclusions — regression
// gates, backend choices, optimization wins — needs dispersion data:
//
//	repeated, err := benchkit.RunRepeated(ctx, config, factory)
//	if err != nil { ... }
//	if !repeated.Reliable() {
//		log.Warnf("noisy metrics, re-run before comparing: %v", repeated.NoisyMetrics())
//	}
//	benchkit.WriteBenchstatRepeated(os.Stdout, repeated)
//
// RunRepeated returns every run plus the median (annotated with Repeat* and
// MetricVariation). WriteBenchstatRepeated emits one benchstat sample per run,
// which is the sample count benchstat needs to report a confidence interval:
// a single-sample file can only be compared as point estimates.
// RepeatedResult.WriteRepeatedJSON (or --format manifest --include-runs)
// serializes the full per-run record instead.
//
// The latency bounds are exact at BOTH ends: P100 (worst) and Min (fastest)
// are tracked per Record and survive reservoir eviction. P50-P99 default to
// nearest-rank; Config.InterpolatedPercentiles switches them to linear
// interpolation for small-n runs where nearest-rank P99 is just the maximum.
// Load is sampled at start (Environment.LoadAvg1) and end (LoadAvg1End), so a
// run the machine polluted mid-flight is self-describing; the
// oversubscription line is Config.LoadWarnThreshold (default 1.0 x CPUs).
//
// # Soak testing
//
// RunSoak repeats the workload for a fixed duration, forcing GC between
// iterations, to detect memory leaks and performance degradation (M19):
//
//	soakResult, err := benchkit.RunSoak(ctx, benchkit.SoakConfig{
//	    Duration:       5 * time.Minute,
//	    ReportInterval: 10 * time.Second,
//	    ProgressWriter: os.Stderr,
//	    Config:         config,
//	}, factory)
//	benchkit.PrintSoakReport(os.Stdout, soakResult)
//
// The result reports HeapGrowthBytes, HeapLeakRate (bytes/iteration),
// ThroughputDriftPct, WriteP99DriftPct, and per-phase P99 drift metrics
// (JourneyP99DriftPct, QueryHitP99DriftPct, CacheHitP99DriftPct — zero when
// the corresponding phase is skipped). ThroughputCoV and WriteP99CoV add
// cross-iteration dispersion next to the endpoint drift: a soak whose
// iterations alternate fast/slow can drift ~0% while being garbage, and the
// CoV is what exposes it. GCMaxPauseDriftPct tracks GC pause
// degradation across iterations (positive = worsening GC behavior).
// AllocGrowthPct tracks allocation growth (positive = allocation leak).
// Use a small profile (ProfileDev) for fast iterations and more data points.
// CLI: `cqrs-bench run --soak 5m`.
//
// # Toolchain requirement
//
// This package imports encoding/json/v2: it requires Go 1.27+ (where the
// package graduated). No build tag or GOEXPERIMENT is needed.
//
//	go build ./benchkit/...
//	go test ./benchkit/...
package benchkit
