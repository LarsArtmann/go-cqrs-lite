# Benchmarking Tool Design Report

> How to build an easy-to-use benchmarking tool for go-cqrs-lite that lets
> consumers and deployers measure performance across all supported backends,
> deployment sizes, and workload profiles — for both synthetic and production
> data.

**Date:** 2026-07-23
**Status:** ~~Proposal~~ **Implemented** (2026-07-23) as `benchkit/` module + `cmd/cqrs-bench/` CLI. Phases 1 (core types, runner, profiles, reports), 3 (projection), 4 (comparison + report), and 5 (CLI) are done. Phase 2 (durability) has partial progress (disk measurement via `DiskPath`/`DiskSizer` interface, Pebble `Disk.DatabaseBytes` tested). Phases 6 (production replay) and 7 (`benchtest.RunSuite`) remain. 55 tests pass with `-race`. See [implementation status](../status/2026-07-23_17-10_benchkit-implementation-status.md), [bugfix session](../status/2026-07-24_05-59_benchkit-bugfix-session-status.md), [critical fixes](../status/2026-07-24_15-13_benchkit-critical-fixes-status.md), and [completeness session](../status/2026-07-24_16-45_benchkit-completeness-session-status.md) for progressive status.

---

## 1. Executive Summary

The library has five storage presets (memory, SQLite, Pebble, Postgres, Turso),
a deployer-first Bundle abstraction, and a contract test suite that verifies
behavioral parity across backends. What's missing is the **performance
equivalent of contracttest**: a tool that runs the same workload against any
backend and produces a structured metrics report.

This report proposes **`benchkit`** — a new module that mirrors the
`contracttest.Factory` pattern for benchmarks. A deployer provides a Bundle
factory, the consumer optionally provides domain types, and `benchkit` runs a
suite of realistic workloads while collecting latency percentiles, throughput,
memory, CPU, storage footprint, and projection lag.

The tool serves two audiences:

| Audience     | Question answered                                         |
| ------------ | --------------------------------------------------------- |
| **Deployer** | "Which backend should I choose for my expected workload?" |
| **Consumer** | "How does my decider/projection perform under load?"      |

Both get the same metrics, the same report format, and the same one-line API.

---

## 2. Current State

### What exists today

| Component         | Location                               | What it does                                                                                      |
| ----------------- | -------------------------------------- | ------------------------------------------------------------------------------------------------- |
| Micro-benchmarks  | `stack/bench/`                         | Proves zero-overhead for Bundle field access (ns/op + allocs/op)                                  |
| Scale benchmarks  | `integration/realistic_bench*_test.go` | `//go:build scale` gated. E-commerce model, memory store only. Measures full pipeline throughput. |
| Benchmark results | `docs/benchmarks/README.md`            | Historical results for memory + SQLite (manual, not reproducible)                                 |
| Contract tests    | `stack/contracttest/`                  | `RunSuite(t, factory)` — behavioral parity across backends                                        |
| Pebble metrics    | `stack/pebble/Bundle.Metrics()`        | LSM-tree health (block cache hit rate, etc.)                                                      |
| Projection lag    | `projectionhost.Host`                  | `LagDuration()`, `LagPerProjection()`, `Status()`                                                 |

### What's missing

1. **No cross-backend comparison** — scale benchmarks only test memory store
2. **No latency percentiles** — only total time + throughput, no P50/P95/P99
3. **No storage footprint measurement** — no on-disk size reporting
4. **No memory/CPU profiling** — only allocs/op from `testing.B`
5. **No durability measurement** — no "time to disk" vs "time to DB"
6. **No structured output** — results are in `testing.B` format, not comparable JSON
7. **No production data replay** — no way to benchmark against a real event dump
8. **No workload profiles** — every benchmark hardcodes its own scale parameters

---

## 3. Design Principles

The tool follows the library's existing philosophy:

1. **Factory pattern** — Same as `contracttest.Factory`: `func() (*stack.Bundle, error)`. The tool never imports a backend directly.

2. **Deployer-first** — The deployer provides the Bundle factory. The tool doesn't know or care which backend is used. Switching backends is a one-line change.

3. **Library, not framework** — `benchkit` is importable library code. Consumers embed it in their test suite or CLI. The optional `cmd/cqrs-bench` CLI is a convenience, not a requirement.

4. **Composition over configuration** — Workloads are composable. A consumer can mix built-in operations with their own domain-specific operations.

5. **Minimal dependencies** — Percentile tracking is implemented in-package (simple sorted-slice for small N, optional HDR histogram for large N). No heavyweight dependencies.

6. **Structured output** — Results are serializable (JSON). Reports are generated from results, not baked into the runner. This enables CI integration, trend tracking, and comparison.

---

## 4. Architecture

### Module structure

```
benchkit/                     # New top-level module (like scenario/, testutil/)
├── go.mod                    # Depends on: stack, event, kv, decider, projectionhost
├── benchkit.go               # Core types: Config, Result, Scenario
├── workload.go               # Workload definition + built-in workloads
├── generator.go              # Synthetic data generation (template + replay)
├── metrics.go                # Latency collector, memory/CPU/storage probes
├── runner.go                 # Scenario runner: executes workload, collects metrics
├── report.go                 # JSON, markdown table, comparison report
├── profiles.go               # Named deployment profiles (dev/small/medium/large/stress)
├── benchtest/                # Test helpers (mirrors contracttest/)
│   └── benchtest.go          # RunSuite(b, factory) — cross-backend benchmark suite
└── cmd/cqrs-bench/           # Optional CLI (separate go.mod, like cmd/cqrs-lint/)
    ├── go.mod
    └── main.go               # CLI: run profiles, compare backends, output reports
```

### Dependency graph

```
benchkit → stack → event, command, query, kv, snapshot
                   event → id, metadata, codec
         → projectionhost → event, projection
         → decider → event

cmd/cqrs-bench → benchkit + all stack presets (memory, sqlite, pebble, postgres, turso)
```

`benchkit` itself depends only on `stack` and `projectionhost` — it never
imports a backend driver. The CLI tool imports presets to provide out-of-the-box
backend factories, but the library does not.

---

## 5. Core Types

### Config

```go
// Config defines a benchmark run.
type Config struct {
    // Profile controls the scale: number of aggregates, events per aggregate,
    // concurrency level, read/write ratio. Use a named profile or customize.
    Profile Profile

    // PayloadSize controls the synthetic payload byte size per event.
    // Default: 256 bytes (realistic for JSON domain events).
    PayloadSize int

    // Codec controls payload encoding. Default: codec.JSONCodec{}.
    // Set to codec.CBORCodec{} to benchmark CBOR performance.
    Codec codec.Codec

    // Duration caps the wall-clock time. 0 = run to completion.
    Duration time.Duration

    // Warmup runs N operations before measurement begins (JIT cache warming).
    Warmup int

    // Concurrency controls parallel writers/readers.
    // 0 = use Profile.Concurrency.
    Concurrency int

    // SnapshotEvery enables snapshotting every N events per aggregate.
    // 0 = no snapshots.
    SnapshotEvery int

    // Projections registers projections to run during the benchmark,
    // enabling projection-lag measurement.
    Projections []projection.Projection

    // Seed controls the deterministic random data generator.
    // Same seed = same data across runs. Default: 1.
    Seed int64
}
```

### Result

```go
// Result is the output of a single benchmark run against one backend.
type Result struct {
    // Identification
    Backend    string        `json:"backend"`     // "memory", "sqlite", "pebble", etc.
    Profile    string        `json:"profile"`     // "dev", "small", "medium", etc.
    Timestamp  time.Time     `json:"timestamp"`
    Duration   time.Duration `json:"duration"`

    // Workload
    Aggregates     int `json:"aggregates"`
    EventsPerAgg   int `json:"eventsPerAggregate"`
    TotalEvents    int `json:"totalEvents"`
    PayloadBytes   int `json:"payloadBytesPerEvent"`

    // Write metrics
    WriteLatency LatencyStats `json:"writeLatency"`    // Save() latency
    WriteThroughput float64   `json:"writeThroughput"` // events/sec

    // Read metrics
    LoadLatency   LatencyStats `json:"loadLatency"`    // Load() latency
    ReadAllTime   time.Duration `json:"readAllTime"`   // ReadAll() wall time
    ReadFromTime  time.Duration `json:"readFromTime"`  // ReadFrom() wall time

    // Read model metrics
    ReadModelGet LatencyStats `json:"readModelGet"`    // kv.TypedStore.Get()
    ReadModelSet LatencyStats `json:"readModelSet"`    // kv.TypedStore.Set()

    // Projection metrics (if projections configured)
    ProjectionLag    time.Duration `json:"projectionLag"`    // max lag across projections
    ProjectionEvents int64         `json:"projectionEvents"` // events processed

    // Resource metrics
    Memory ResourceStats `json:"memory"`
    CPU    ResourceStats `json:"cpu"`
    Disk   DiskStats     `json:"disk"`

    // Codec
    Codec string `json:"codec"` // "json", "cbor"
}

// LatencyStats holds percentile latency data.
type LatencyStats struct {
    P50  time.Duration `json:"p50"`
    P75  time.Duration `json:"p75"`
    P90  time.Duration `json:"p90"`
    P95  time.Duration `json:"p95"`
    P99  time.Duration `json:"p99"`
    P100 time.Duration `json:"p100"` // max
    Mean time.Duration `json:"mean"`
}

// ResourceStats holds resource usage snapshots.
type ResourceStats struct {
    Before uint64 `json:"before"` // baseline before workload
    After  uint64 `json:"after"`  // peak during/after workload
    Delta  uint64 `json:"delta"`  // After - Before
}

// DiskStats holds storage footprint data.
type DiskStats struct {
    DatabaseBytes int64 `json:"databaseBytes"` // on-disk DB size
    EventBytes    int64 `json:"eventBytes"`    // raw event payload bytes
    OverheadBytes int64 `json:"overheadBytes"` // DatabaseBytes - EventBytes
    OverheadPct   float64 `json:"overheadPct"` // overhead as % of DatabaseBytes
}
```

### Workload

```go
// Workload defines what operations a benchmark run performs.
// The built-in workloads cover the standard CQRS pipeline.
// Consumers can implement custom workloads for domain-specific benchmarks.
type Workload struct {
    // Setup is called once before the benchmark. Use it to create
    // the decider, projections, read-model stores, etc.
    Setup func(ctx context.Context, bundle *stack.Bundle) error

    // WriteOps generates write operations. Each call returns the next
    // event to write. Returns nil to signal completion.
    WriteOps func(ctx context.Context, aggIndex int) ([]event.Event, error)

    // ReadOps generates read operations. Each call returns the aggregate
    // reference to load. Returns empty ref to signal completion.
    ReadOps func(ctx context.Context, aggIndex int) (id.AggregateRef, error)

    // Teardown is called once after the benchmark for cleanup.
    Teardown func(ctx context.Context, bundle *stack.Bundle) error
}
```

---

## 6. Metrics Catalog

Every metric the tool collects, what it measures, and how:

### Latency metrics

| Metric               | What                                       | How measured                                  | Percentiles                   |
| -------------------- | ------------------------------------------ | --------------------------------------------- | ----------------------------- |
| **Write latency**    | `EventSink.Save()` call duration           | `time.Since()` around Save, recorded per-call | P50/P75/P90/P95/P99/P100/Mean |
| **Load latency**     | `EventSource.Load()` call duration         | `time.Since()` around Load, recorded per-call | P50/P75/P90/P95/P99/P100/Mean |
| **ReadFrom latency** | `SeekableJournal.ReadFrom()` call duration | `time.Since()` around ReadFrom                | P50/P75/P90/P95/P99/P100/Mean |
| **Read model get**   | `kv.TypedStore.Get()` call duration        | `time.Since()` around Get                     | P50/P75/P90/P95/P99/P100/Mean |
| **Read model set**   | `kv.TypedStore.Set()` call duration        | `time.Since()` around Set                     | P50/P75/P90/P95/P99/P100/Mean |
| **Query dispatch**   | `query.DispatchTyped()` call duration      | `time.Since()` around dispatch                | P50/P75/P90/P95/P99/P100/Mean |

### Throughput metrics

| Metric                    | What                                | How measured                            |
| ------------------------- | ----------------------------------- | --------------------------------------- |
| **Write throughput**      | Sustained events/sec                | `totalEvents / wallTime`                |
| **Read throughput**       | Sustained loads/sec                 | `totalLoads / wallTime`                 |
| **Projection throughput** | Events processed/sec by projections | `projectionEvents / projectionWallTime` |

### Durability metrics

| Metric                  | What                                                                        | How measured                                                                                                                                     |
| ----------------------- | --------------------------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------ |
| **Time to DB**          | Wall time for Save() to return (includes DB call but may not include fsync) | Direct: time.Save()                                                                                                                              |
| **Time to disk**        | Wall time for write to be durable on disk                                   | Indirect: after Save(), immediately Load() from a fresh connection and measure total roundtrip. Compare with/without WAL to estimate fsync cost. |
| **WAL checkpoint time** | SQLite WAL checkpoint duration                                              | `PRAGMA wal_checkpoint(TRUNCATE)` timing                                                                                                         |

The "time to disk" metric is nuanced. True fsync latency requires either:

- OS-level tracing (strace/eBPF) — out of scope for a library tool
- Backend-specific queries (SQLite `PRAGMA synchronous`, Pebble `Flush()`)
- Comparison methodology: run the same workload with `synchronous=FULL` vs `synchronous=OFF` and measure the delta

The tool uses the **comparison methodology**: it reports Save() latency as "time to DB" and optionally runs a second pass with forced durability (`Pebble.Flush()` or SQLite `PRAGMA wal_checkpoint(FULL)`) to measure "time to disk". The difference is the fsync cost.

### Projection metrics

| Metric                    | What                                               | How measured                             |
| ------------------------- | -------------------------------------------------- | ---------------------------------------- |
| **Projection lag**        | Time between newest event and last processed event | `projectionhost.Host.LagDuration()`      |
| **Per-projection lag**    | Lag for each registered projection                 | `projectionhost.Host.LagPerProjection()` |
| **Projection throughput** | Events processed/sec                               | `WorkerState.Processed / wallTime`       |
| **Projection errors**     | Total handler errors                               | `WorkerState.Errors`                     |
| **Projection restarts**   | Worker crash-restart count                         | `WorkerState.Restarts`                   |

### Resource metrics

| Metric                   | What                          | How measured                                                         |
| ------------------------ | ----------------------------- | -------------------------------------------------------------------- |
| **Heap allocation**      | Go heap bytes in use          | `runtime.MemStats.HeapAlloc` (before + after)                        |
| **Total allocation**     | Cumulative bytes allocated    | `runtime.MemStats.TotalAlloc` (delta)                                |
| **GC pauses**            | Total GC pause time           | `runtime.MemStats.PauseTotalNs` (delta)                              |
| **GC count**             | Number of GC cycles           | `runtime.MemStats.NumGC` (delta)                                     |
| **RSS (process memory)** | Resident set size             | `/proc/self/status` VmRSS (Linux), `os.Process` (portable fallback)  |
| **CPU user time**        | User-mode CPU time consumed   | `os.Process` times via `process.RuntimeProfile` or `/proc/self/stat` |
| **CPU sys time**         | Kernel-mode CPU time consumed | Same source as user time                                             |

### Storage metrics

| Metric               | What                            | How measured                                                                      |
| -------------------- | ------------------------------- | --------------------------------------------------------------------------------- |
| **Database size**    | On-disk DB footprint            | `filepath.Walk` summing file sizes (SQLite, Pebble). In-memory backends report 0. |
| **Raw event bytes**  | Sum of all event payload bytes  | `sum(len(evt.Payload()))` across all events                                       |
| **Storage overhead** | DB size minus raw payload bytes | `DatabaseBytes - EventBytes`                                                      |
| **Overhead ratio**   | Overhead as % of DB size        | `OverheadBytes / DatabaseBytes * 100`                                             |
| **Events per MB**    | Storage efficiency              | `TotalEvents / (DatabaseBytes / 1MB)`                                             |

### Backend-specific metrics

| Backend      | Extra metrics available                                                                           |
| ------------ | ------------------------------------------------------------------------------------------------- |
| **Pebble**   | `Bundle.Metrics()` — block cache hit rate, LSM-tree levels, compaction count, write amplification |
| **SQLite**   | `PRAGMA page_count`, `PRAGMA page_size`, WAL file size, `PRAGMA journal_mode`                     |
| **Postgres** | Connection pool stats, `pg_database_size()`, `pg_stat_user_tables`                                |
| **Memory**   | In-memory event count, map size estimation                                                        |

---

## 7. Workload Profiles

Named profiles representing realistic deployment sizes. Each profile is a
starting point — consumers override individual fields as needed.

```go
// Profile defines the scale parameters for a benchmark run.
type Profile struct {
    Name          string
    Aggregates    int           // number of distinct aggregate IDs
    EventsPerAgg  int           // events written per aggregate
    Concurrency   int           // parallel goroutines
    ReadRatio     float64       // 0.0 = write-only, 1.0 = read-only
    BatchSize     int           // events per Save() call (1 = single, N = batch)
}
```

| Profile      | Aggregates | Events/Agg | Total Events | Concurrent | Read/Write | Batch |
| ------------ | ---------: | ---------: | -----------: | ---------: | ---------: | ----: |
| `Dev`        |        100 |          5 |          500 |          1 |        0.2 |     1 |
| `Small`      |      1,000 |         10 |          10K |          4 |        0.3 |     1 |
| `Medium`     |     10,000 |         50 |         500K |         16 |        0.4 |     5 |
| `Large`      |    100,000 |        100 |          10M |         32 |        0.5 |    10 |
| `Stress`     |     10,000 |        500 |           5M |         64 |        0.2 |     1 |
| `WriteHeavy` |     10,000 |        100 |           1M |         32 |        0.1 |     1 |
| `ReadHeavy`  |     10,000 |        100 |           1M |         32 |        0.8 |     1 |

Profiles are **named** so results are comparable across runs and backends. A
consumer runs `benchkit.ProfileMedium` against SQLite and Pebble and gets an
apples-to-apples comparison.

---

## 8. Data Generation

### Synthetic data

Built-in payload generator produces realistic JSON/CBOR payloads:

```go
// Generator produces synthetic event payloads for benchmarking.
type Generator struct {
    rng *rand.Rand
    size int
}

// Payload returns a deterministic random byte slice of the configured size.
// The payload is valid JSON with realistic field names and types.
func (g *Generator) Payload() []byte
```

The default generator produces payloads like:

```json
{
  "id": "01HX...",
  "name": "Order-4287",
  "value": 129.99,
  "items": 3,
  "tags": ["priority", "express"],
  "metadata": { "source": "web", "session": "abc123" }
}
```

Payload size is configurable (64B to 16KB). The generator uses a seeded RNG so
runs are reproducible.

### Production data replay

For benchmarking against real workloads, the tool can replay events from a
journal dump:

```go
// ReplaySource reads events from a production dump file (JSON Lines format).
// Each line is a JSON event: {"type":"...","aggregateId":"...","payload":{...}}
type ReplaySource struct {
    path string
}

// Events returns a channel of events read from the dump file.
func (r *ReplaySource) Events(ctx context.Context) (<-chan event.Event, error)
```

Usage:

```bash
# Export from production:
cqrs-bench export --dsn "./prod.db" --output prod-dump.jsonl

# Replay against any backend:
cqrs-bench replay --backend sqlite --dsn ":memory:" --input prod-dump.jsonl
```

---

## 9. API Design

### Library API (for consumers embedding in test suites)

```go
import "github.com/larsartmann/go-cqrs-lite/benchkit/v4"

func BenchmarkMyBackend(b *testing.B) {
    factory := func() (*stack.Bundle, error) {
        return sqlite.New(filepath.Join(b.TempDir(), "bench.db"))
    }

    result, err := benchkit.Run(b.Context(), benchkit.Config{
        Profile:    benchkit.ProfileMedium,
        PayloadSize: 256,
    }, factory)
    if err != nil {
        b.Fatal(err)
    }

    benchkit.PrintReport(b.Logf, result)
}
```

### Cross-backend comparison API

```go
results, err := benchkit.Compare(b.Context(), benchkit.Config{
    Profile: benchkit.ProfileMedium,
}, map[string]benchkit.Factory{
    "memory":  func() (*stack.Bundle, error) { return memory.New() },
    "sqlite":  func() (*stack.Bundle, error) { return sqlite.New(":memory:") },
    "pebble":  func() (*stack.Bundle, error) { return pebble.New(t.TempDir()) },
})

benchkit.PrintComparison(os.Stdout, results)
```

Output:

```
┌──────────┬───────────┬───────────┬───────────┬───────────┬─────────┐
│ Backend  │ Write P50 │ Write P99 │ Load P50  │ Load P99  │ Disk MB │
├──────────┼───────────┼───────────┼───────────┼───────────┼─────────┤
│ memory   │     1.2μs │     4.8μs │   480ns   │   2.1μs   │     0   │
│ sqlite   │    42μs   │   185μs   │   48μs    │  210μs    │    12   │
│ pebble   │    18μs   │    72μs   │   12μs    │   55μs    │     8   │
└──────────┴───────────┴───────────┴───────────┴───────────┴─────────┘
```

### benchtest suite (mirrors contracttest)

```go
// RunSuite runs the full benchmark suite against the given factory.
// Each preset registers this in its own benchmark file:
func TestBenchmarkSuite(b *testing.B) {
    benchtest.RunSuite(b, func() (*stack.Bundle, error) {
        return sqlite.New(filepath.Join(b.TempDir(), "bench.db"))
    })
}
```

### CLI API (cmd/cqrs-bench)

```bash
# Run a single backend + profile
cqrs-bench run --backend sqlite --dsn ":memory:" --profile medium

# Compare all local backends
cqrs-bench compare --profile medium --output report.json

# Compare with a specific codec
cqrs-bench compare --profile medium --codec cbor

# Replay production data
cqrs-bench replay --backend pebble --dir /tmp/bench --input prod-dump.jsonl

# Generate markdown report from previous results
cqrs-bench report --input results/*.json --format markdown > comparison.md
```

CLI flags follow the `cmdguard` struct-tag pattern used by `cqrs-lint`:

```go
type RunConfig struct {
    cmdguard.Config
    Backend  string `flag:"backend" default:"memory" help:"Backend: memory, sqlite, pebble, postgres, turso"`
    DSN      string `flag:"dsn" default:":memory:" help:"Database connection string"`
    Profile  string `flag:"profile" default:"small" help:"Workload profile: dev, small, medium, large, stress"`
    Codec    string `flag:"codec" default:"json" help:"Payload codec: json, cbor"`
    Duration string `flag:"duration" default:"0s" help:"Max wall time (0=unlimited)"`
    Output   string `flag:"output" default:"-" help:"Output file (-=stdout)"`
    Format   string `flag:"format" default:"text" help:"Output format: text, json, markdown"`
}
```

---

## 10. Latency Collection Strategy

### Percentile tracker

For latency percentiles, the tool uses a **simple, dependency-free approach**:

```go
type LatencyCollector struct {
    samples []time.Duration // unsorted, appended in order
}

func (lc *LatencyCollector) Record(d time.Duration) {
    lc.samples = append(lc.samples, d)
}

func (lc *LatencyCollector) Stats() LatencyStats {
    slices.Sort(lc.samples)
    n := len(lc.samples)
    return LatencyStats{
        P50:  lc.samples[n*50/100],
        P75:  lc.samples[n*75/100],
        P90:  lc.samples[n*90/100],
        P95:  lc.samples[n*95/100],
        P99:  lc.samples[n*99/100],
        P100: lc.samples[n-1],
        Mean: mean(lc.samples),
    }
}
```

For large N (>100K samples), the collector switches to a fixed-size reservoir
sample (10K entries) to bound memory. This gives accurate P99+ with O(1) memory.

**Why not HDR histogram?** HDR-Histogram is excellent but adds a dependency for
a feature that's only needed for >1M samples. The reservoir approach covers
99% of cases with zero deps. HDR can be added as an optional integration later.

### Memory sampling

Memory is sampled at three points:

1. **Before** the workload (after Bundle setup)
2. **Peak** during the workload (sampled every 100ms via goroutine)
3. **After** the workload (after GC)

```go
func sampleMemory() uint64 {
    var m runtime.MemStats
    runtime.ReadMemStats(&m)
    return m.HeapAlloc
}
```

### Storage measurement

After the workload completes, the tool measures on-disk footprint:

```go
func measureDisk(bundle *stack.Bundle) int64 {
    // Backend-specific size reporting via optional interface
    if sizer, ok := bundle.(interface{ DiskSize() int64 }); ok {
        return sizer.DiskSize()
    }
    // Fallback: walk the directory (SQLite/Pebble use file-based storage)
    // In-memory backends return 0
    return 0
}
```

Each preset Bundle can optionally implement `DiskSize() int64`:

- SQLite: `PRAGMA page_count * page_size` + WAL file size
- Pebble: `filepath.Walk` on the data directory
- Memory: 0 (or estimated in-memory size)
- Postgres: `pg_database_size(current_database())`

---

## 11. Runner Architecture

The runner executes a workload in phases, collecting metrics at each step:

```
┌─────────────────────────────────────────────────────────────┐
│                      benchkit.Run                            │
├─────────────────────────────────────────────────────────────┤
│                                                             │
│  1. SETUP                                                   │
│     factory() → *stack.Bundle                               │
│     measure baseline memory/CPU                             │
│                                                             │
│  2. WARMUP (optional)                                       │
│     run N write+load cycles (not measured)                  │
│                                                             │
│  3. WRITE PHASE                                             │
│     for each aggregate (concurrent):                        │
│       generate events → Save() → record write latency       │
│     record write throughput                                 │
│                                                             │
│  4. READ PHASE                                              │
│     for each aggregate (concurrent):                        │
│       Load() → record load latency                          │
│     ReadAll() → record journal scan time                    │
│     ReadFrom() → record seekable journal time               │
│                                                             │
│  5. READ MODEL PHASE                                        │
│     for each key: Set() → record set latency                │
│     for each key: Get() → record get latency                │
│                                                             │
│  6. PROJECTION PHASE (if projections configured)             │
│     start projectionhost → process events                   │
│     record projection lag, throughput, errors               │
│                                                             │
│  7. DURABILITY CHECK                                        │
│     for file backends: measure disk size                    │
│     for SQLite: PRAGMA wal_checkpoint, measure              │
│     for Pebble: Flush(), measure                            │
│                                                             │
│  8. TEARDOWN                                                │
│     measure peak memory                                     │
│     GC + measure after-heap                                 │
│     bundle.Close()                                          │
│                                                             │
│  9. RESULT ASSEMBLY                                         │
│     compute percentiles, throughput, ratios                 │
│     return Result                                           │
│                                                             │
└─────────────────────────────────────────────────────────────┘
```

### Concurrency model

The write and read phases use a worker pool:

```go
func (r *Runner) runConcurrent(
    ctx context.Context,
    totalOps int,
    concurrency int,
    op func(ctx context.Context, index int) error,
) error {
    sem := make(chan struct{}, concurrency)
    var wg sync.WaitGroup
    errCh := make(chan error, 1)

    for i := range totalOps {
        select {
        case sem <- struct{}{}:
        case <-ctx.Done():
            return ctx.Err()
        }

        wg.Add(1)

        go func(idx int) {
            defer wg.Done()
            defer func() { <-sem }()

            if err := op(ctx, idx); err != nil {
                select {
                case errCh <- err:
                default:
                }
            }
        }(i)
    }

    wg.Wait()
    select {
    case err := <-errCh:
        return err
    default:
        return nil
    }
}
```

Each goroutine records latencies into a thread-local collector, which are merged
after the phase completes.

---

## 12. Report Generation

### Text format (default)

```
Benchmark: sqlite | profile=medium | codec=json
═══════════════════════════════════════════════════════════
Workload: 10,000 aggregates × 50 events = 500,000 events
Payload:  256 bytes/event | Batch: 5 | Concurrent: 16

Write Performance:
  Throughput:  42,318 events/sec
  Latency:     P50=38μs  P95=82μs  P99=145μs  Max=312μs

Read Performance:
  Load:        P50=41μs  P95=89μs  P99=160μs  Max=298μs
  ReadAll:     1.2s (500K events)
  ReadFrom:    8ms (1K events from position)

Read Model:
  Get:         P50=22μs  P95=48μs  P99=72μs
  Set:         P50=31μs  P95=67μs  P99=95μs

Resources:
  Heap:        142MB (delta from baseline)
  Allocs:      2.1M total
  GC pauses:   3 cycles, 4.2ms total
  CPU:         8.4s user, 1.2s sys

Storage:
  Database:    18.4 MB
  Raw events:  128.0 MB (500K × 256B)
  Overhead:    -109.6 MB (CBOR compression: 86% of raw)
  Efficiency:  27,174 events/MB
```

### JSON format (for CI/trend tracking)

```json
{
  "backend": "sqlite",
  "profile": "medium",
  "timestamp": "2026-07-23T14:30:00Z",
  "duration": "12.4s",
  "writeLatency": { "p50": "38µs", "p95": "82µs", "p99": "145µs" },
  "writeThroughput": 42318.5,
  "loadLatency": { "p50": "41µs", "p95": "89µs", "p99": "160µs" },
  "memory": { "heapDelta": 142000000 },
  "disk": { "databaseBytes": 18400000 }
}
```

### Comparison format (markdown table)

```markdown
| Backend  | Write P50 | Write P99 | Load P50 | Load P99 | Throughput | Heap MB | Disk MB |
| -------- | --------: | --------: | -------: | -------: | ---------: | ------: | ------: |
| memory   |     1.2μs |     4.8μs |    480ns |    2.1μs |     450K/s |     142 |       0 |
| sqlite   |      38μs |     145μs |     41μs |    160μs |      42K/s |     142 |      18 |
| pebble   |      18μs |      72μs |     12μs |     55μs |     120K/s |      89 |       8 |
| postgres |      52μs |     210μs |     48μs |    195μs |      35K/s |     156 |     N/A |
```

---

## 13. Integration with Existing Infrastructure

### Nix flake integration

Add benchmark targets to `flake.nix`:

```nix
apps = {
  # Existing
  build = ...;
  test = ...;
  lint = ...;

  # New
  bench = {
    type = "app";
    program = "${pkgs.writeShellScriptBin "bench" ''
      cd ${self}
      go run ./cmd/cqrs-bench run --backend "$1" --profile "$2" "''${@:3}"
    ''}/bin/bench";
  };

  bench-compare = {
    type = "app";
    program = "${pkgs.writeShellScriptBin "bench-compare" ''
      cd ${self}
      go run ./cmd/cqrs-bench compare --profile "$1" --output "docs/benchmarks/$(date +%Y-%m-%d)_comparison.json"
    ''}/bin/bench-compare";
  };
};
```

Usage: `nix run .#bench -- sqlite medium` or `nix run .#bench-compare -- large`

### CI integration

A GitHub Actions workflow runs benchmarks on PRs that touch storage code:

```yaml
- name: Benchmark comparison
  run: |
    nix run .#bench-compare -- small
    # Compare against baseline, fail if regression > 20%
    cqrs-bench compare --baseline .github/benchmarks/baseline.json --threshold 20
```

### docs/benchmarks integration

The JSON output feeds directly into an auto-generated `docs/benchmarks/`
directory with dated results, enabling trend tracking:

```
docs/benchmarks/
├── README.md                          # human-readable summary
├── 2026-07-23_medium_comparison.json  # machine-readable raw data
├── 2026-07-23_large_comparison.json
└── trend/                             # historical trend data
```

---

## 14. Implementation Plan

### Phase 1: Core module (MVP)

**Goal:** Run a synthetic write+read workload against any Bundle, collect
latency percentiles, output text report.

| Step | What                                           | Files                   |
| ---- | ---------------------------------------------- | ----------------------- |
| 1.1  | Create `benchkit/` module with `go.mod`        | `benchkit/go.mod`       |
| 1.2  | Core types: `Config`, `Result`, `LatencyStats` | `benchkit/benchkit.go`  |
| 1.3  | Latency collector (sorted-slice percentile)    | `benchkit/metrics.go`   |
| 1.4  | Memory + CPU sampling                          | `benchkit/metrics.go`   |
| 1.5  | Synthetic payload generator                    | `benchkit/generator.go` |
| 1.6  | Named profiles                                 | `benchkit/profiles.go`  |
| 1.7  | Runner: write + read phases                    | `benchkit/runner.go`    |
| 1.8  | Text report generator                          | `benchkit/report.go`    |
| 1.9  | Factory type + Run function                    | `benchkit/benchkit.go`  |
| 1.10 | Tests against memory + sqlite                  | `benchkit/*_test.go`    |

**Deliverable:** `benchkit.Run(ctx, config, factory)` produces a Result with
write/read latency percentiles, throughput, and memory deltas.

### Phase 2: Storage + durability metrics

| Step | What                                                   |
| ---- | ------------------------------------------------------ |
| 2.1  | Disk size measurement (filepath.Walk)                  |
| 2.2  | SQLite-specific metrics (PRAGMA page_count, WAL size)  |
| 2.3  | Pebble-specific metrics (Bundle.Metrics integration)   |
| 2.4  | Durability comparison methodology (Save vs Save+Flush) |
| 2.5  | Storage overhead calculation                           |

**Deliverable:** Disk metrics in Result. Storage efficiency reporting.

### Phase 3: Projection benchmarks

| Step | What                                                      |
| ---- | --------------------------------------------------------- |
| 3.1  | Projection phase in runner                                |
| 3.2  | projectionhost.Host integration (lag, throughput, errors) |
| 3.3  | Read model benchmark phase (kv.TypedStore Get/Set)        |
| 3.4  | Journal scan benchmark (ReadAll, ReadFrom)                |

**Deliverable:** Projection lag, read model latency, journal scan time in Result.

### Phase 4: Cross-backend comparison + report

| Step | What                                       |
| ---- | ------------------------------------------ |
| 4.1  | `Compare(ctx, config, factories)` function |
| 4.2  | Comparison table report generator          |
| 4.3  | JSON output format                         |
| 4.4  | Markdown table output format               |

**Deliverable:** One-call backend comparison with side-by-side metrics.

### Phase 5: CLI tool

| Step | What                                             |
| ---- | ------------------------------------------------ |
| 5.1  | `cmd/cqrs-bench/` module with cmdguard config    |
| 5.2  | `run` subcommand (single backend)                |
| 5.3  | `compare` subcommand (all local backends)        |
| 5.4  | `report` subcommand (generate reports from JSON) |
| 5.5  | `replay` subcommand (production data replay)     |
| 5.6  | Nix flake integration                            |

**Deliverable:** `cqrs-bench run --backend sqlite --profile medium` CLI.

### Phase 6: Production data replay

| Step | What                                        |
| ---- | ------------------------------------------- |
| 6.1  | JSON Lines export format                    |
| 6.2  | ReplaySource (reads dump, generates events) |
| 6.3  | Export subcommand in CLI                    |
| 6.4  | Replay benchmark mode                       |

**Deliverable:** Benchmark against real production event streams.

### Phase 7: benchtest suite + preset integration

| Step | What                                                                        |
| ---- | --------------------------------------------------------------------------- |
| 7.1  | `benchtest.RunSuite(b, factory)`                                            |
| 7.2  | Register benchtest in each preset (memory, sqlite, pebble, postgres, turso) |
| 7.3  | CI workflow for benchmark regression detection                              |

**Deliverable:** Every preset ships with benchmark results out of the box.

---

## 15. Alignment with the Core Goal

> My Goal is for consumers of this lib should NOT decide on the implementation
> of infrastructure. They should have a simple API that allows the person
> deploying the App to decide where they want to keep their data.

The benchmarking tool reinforces this goal in three ways:

### 1. Data-driven backend selection

Today, the deployer picks a backend based on the
[Infrastructure Recommendations](../INFRASTRUCTURE_RECOMMENDATIONS.md) doc —
which is qualitative advice. With `benchkit`, the deployer runs:

```bash
cqrs-bench compare --profile medium
```

And gets **quantitative evidence** for their specific workload. The decision
becomes data-driven, not opinion-driven.

### 2. No lock-in validation

The tool proves that switching backends is truly a one-line change. If a
deployer starts with SQLite and outgrows it, they run `cqrs-bench compare` with
Postgres to verify the migration will meet their latency requirements **before**
writing any migration code.

### 3. Multi-DB split guidance

The multi-DB split (`WithEventDB`, `WithQueryDB`, `WithViewDB`) is a powerful
but complex feature. The benchmarking tool can measure the contention reduction
from splitting databases, giving deployers concrete numbers on whether the
split is worth the operational complexity:

```bash
cqrs-bench compare \
  --config single-db.yaml \
  --config multi-db.yaml \
  --profile large
```

---

## 16. Open Questions

| #   | Question                                                                                        | Recommendation                                                                                                                                                                                 |
| --- | ----------------------------------------------------------------------------------------------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| 1   | Should `benchkit` depend on `decider/` for decider benchmarks, or stay at the store level only? | Stay at store level for v1. Decider benchmarks are domain-specific and already handled by `integration/realistic_bench_test.go`. Add a `DeciderBenchmark[State]` generic type in v2 if needed. |
| 2   | HDR histogram vs sorted-slice for percentiles?                                                  | Sorted-slice with reservoir sampling for v1 (zero deps). Add optional HDR integration in v2 for >1M sample precision.                                                                          |
| 3   | Should benchmarks write to real disk or tmpfs?                                                  | Real disk by default (realistic). Add `--tmpfs` flag for CI environments that mount tmpfs. Document the difference.                                                                            |
| 4   | How to handle Postgres/Turso in `compare` (they need external servers)?                         | Skip backends whose connection fails. Print "skipped (unreachable)" in the comparison table. Document `POSTGRES_TEST_DSN` / `TURSO_TEST_DSN` env vars.                                         |
| 5   | Should the tool support distributed benchmarks (multiple processes)?                            | No — v1 is single-process. Distributed benchmarking is a deployment concern, not a library concern. The JSON output format supports aggregating results from multiple runs.                    |
| 6   | Should benchmarks run in CI or be manual?                                                       | Manual by default (gated by build tag or CLI invocation). Add an optional CI workflow that runs `Dev` profile on every PR for regression detection.                                            |
| 7   | Where to store baseline results for regression detection?                                       | Commit `docs/benchmarks/baseline.json` to the repo. Update via `cqrs-bench compare --update-baseline`. CI compares against this.                                                               |

---

## 17. What This Is NOT

- **Not a load testing framework** — no HTTP/gRPC load generation, no virtual
  users, no ramp-up/ramp-down. The tool benchmarks the storage layer, not the
  transport layer.

- **Not a continuous monitoring tool** — it produces point-in-time reports,
  not live dashboards. For live monitoring, use the existing OTel/Prometheus
  integration.

- **Not a correctness test** — that's `contracttest`. This tool assumes the
  backend is correct and measures its performance characteristics.

- **Not a benchmarking framework for arbitrary Go code** — it's specifically
  designed for the go-cqrs-lite Bundle interface and CQRS pipeline.

---

## Summary

The `benchkit` module is the performance equivalent of `contracttest`: a
factory-driven suite that runs against any backend and produces structured,
comparable results. It mirrors the library's deployer-first philosophy — the
tool never imports a backend, the deployer provides the factory, and switching
backends is a one-line change.

The implementation is phased: core latency/throughput metrics first (Phase 1-2),
projection and read-model benchmarks next (Phase 3-4), then CLI and production
replay (Phase 5-6), and finally per-preset benchmark suites (Phase 7).

The result: a deployer runs `cqrs-bench compare --profile medium` and gets a
data-driven answer to "which backend fits my workload?" — the same question that
today requires reading docs and guessing.
