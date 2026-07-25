# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/).

## [Unreleased]

### Added

- **Journey phase** (`Config.SkipJourney`, M14): end-to-end publish→projection→query round-trip latency benchmark. For each sample, writes a single event to a fresh stream, synchronously projects it into the read model, and dispatches a typed query — measuring the full path a user request takes. Records `JourneyLatency`, `JourneyProjectionLatency`, `JourneyQueryLatency`, and `JourneySamples`. Skipped automatically when the bundle lacks EventSink + ReadModels. Capped at 200 samples.
- **Query dispatch phase** (`Config.SkipQuery`, M15): benchmarks `query.Dispatcher` overhead across hit (registered handler reads real data), miss (unregistered type → handler-not-found), and paginated (`PaginatedResult` construction) paths. Records `QueryHitLatency`, `QueryMissLatency`, `QueryPaginatedLatency`, and `QueryCorrectnessErrors`. Skipped automatically when ReadModels is absent. Capped at 500 dispatches.
- **Snapshot/cache hit-rate phase** (`Config.SkipSnapshot`, M16): measures decider `Load` performance under cold replay (full event fold), snapshot load (`EveryNEvents(1)`), and state-cache hit/miss strategies, with correctness assertions verifying state/version equality across all strategies. Records `SnapshotColdLatency`, `SnapshotLoadLatency`, `CacheMissLatency`, `CacheHitLatency`, and `SnapshotCorrectnessErrors`. Skipped automatically when EventSink is not an `event.Store`. Capped at 50 streams.
- **Soak test mode** (`RunSoak` / `--soak`, M19): sustains the benchmark workload for a fixed duration, forcing GC between iterations, to detect memory leaks and performance degradation. Each iteration runs a full benchmark with a fresh Bundle. Computes drift metrics: `HeapGrowthBytes`, `HeapLeakRate`, `ThroughputDriftPct`, `WriteP99DriftPct`, plus per-phase P99 drift (`JourneyP99DriftPct`, `QueryHitP99DriftPct`, `CacheHitP99DriftPct`). CLI flag: `cqrs-bench run --soak 5m` (progress to stderr, result to stdout).
- **CLI flags**: `--skip-journey`, `--skip-query`, `--skip-snapshot` for `cmd/cqrs-bench run`, `sweep`, and `compare`.

### Changed

- `WriteBenchstat` now emits journey, query dispatch, and snapshot/cache latency lines for benchstat trend tracking.
- `ExpectedJSONFields` updated with 16 additional Result fields (rawSinkLatency, readModel, projection, journey, query, snapshot, disk) for schema-stability enforcement.

### Fixed

- `Config.Codec` interface field now round-trips through JSON via a `CodecName` string field (`json:"-"` on Codec, `MarshalJSON`/`UnmarshalJSON` resolve via `codec.ForEncoding`). Previously the Codec field silently became nil after JSON unmarshal.
- Soak loop no longer records partial iterations with zero events (context deadline boundary fix).

### Deprecated

### Removed

### Security

## [4.1.0] - 2026-07-25

### Added

- **Recovery phase** (`Config.Recovery`): crash-recovery benchmark — closes the bundle mid-run, reopens via factory, measures replay time and recovered event count. For persistent backends (SQLite, Pebble), all events survive. For memory backends, documents zero recovery.
- **Replay-only mode** (`Config.ReplayOnly`): benchmarks existing event stores without writing. Discovers streams via batched `SeekableJournal.ReadFrom` (1000-event pages) to avoid OOM on large stores.
- **`benchtest.RunSuite`**: preset integration for `testing.B` — one-call benchmark suite with `b.ReportMetric` for custom metrics (throughput, latency p99, recovery time).
- **`ProfileAnalytical`**: OLAP-style workload profile (wide/shallow streams, 5 journal scans, 0.9 read ratio) for comparing cross-stream scan performance.
- **`Profile.JournalScans`** field: controls how many times `runJournalScans` iterates the journal (defaults to 1).
- **Real kv.Store projection handler**: `newKVCountingProjection` exercises real projection I/O (Get + Set per event) instead of a no-op counter, with graceful `kv.ErrNotFound` handling for first-seen keys.
- **Postgres benchmark backend**: `cmd/cqrs-bench --backend=postgres` with `POSTGRES_TEST_DSN` support (skips when unset).
- **CLI flags**: `--recovery` and `--replay` for `cmd/cqrs-bench`.

### Fixed

- **`readModelPhase` race condition**: `ctx.Err()` check between Set and Get phases prevents `kv.ErrNotFound` failures when Duration timeout interrupts mid-phase.
- **`recoveryPhase` context cancellation**: uses `context.WithoutCancel(parent)` so Load calls succeed even after the benchmark context expires.
- **Phase ctx.Err guards**: all phases (read, readModel, projection, write) now check `ctx.Err()` for graceful early-exit on Duration timeout, preventing wasted work and hung tests.
- **`discoverStreams` OOM risk**: replaced `Journal.ReadAll()` (loads all events) with batched `SeekableJournal.ReadFrom` (1000-event pages).

### Changed

- `newCountingProjection` and `newKVCountingProjection` no longer take a dead `aggIDs` parameter.
- `discoverStreams` refactored into `discoverFromSeekable` + `discoverFromJournal` + `collectStreamIDs` for clarity and reduced nesting.

## [4.0.0] - 2026-07-01

### Added

- Initial stable release: factory-driven benchmarking suite with latency percentiles, throughput, memory tracking, CPU monitoring, and event-size scaling analysis.
