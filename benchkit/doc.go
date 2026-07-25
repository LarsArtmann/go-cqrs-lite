// Package benchkit provides a factory-driven benchmarking suite for
// [stack.Bundle] presets — the performance equivalent of
// [stack/contracttest].
//
// A deployer provides a [Factory] (a function that returns a fresh
// *stack.Bundle), and benchkit runs a suite of realistic write, read,
// read-model, and projection workloads while collecting latency percentiles,
// throughput, memory deltas, and storage footprint.
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
// Every Result includes Environment metadata (GoVersion, NumCPU, GOMAXPROCS,
// GOOS, GOARCH) and the actual Workers count so comparisons across machines
// and configurations are honest.
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
// ThroughputDriftPct, and WriteP99DriftPct. Use a small profile (ProfileDev)
// for fast iterations and more data points. CLI: `cqrs-bench run --soak 5m`.
//
// # Build tag requirement
//
// This package imports encoding/json/v2 and requires the build tag
// "goexperiment.jsonv2". Use:
//
//	go build -tags "goexperiment.jsonv2" ./benchkit/...
//	go test -tags "goexperiment.jsonv2" ./benchkit/...
//
// or set GOEXPERIMENT=jsonv2 in the environment.
package benchkit
