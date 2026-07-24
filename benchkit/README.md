# benchkit

> Factory-driven benchmarking suite for go-cqrs-lite Bundle presets — the
> performance equivalent of `stack/contracttest`.

A deployer provides a `Factory` (a function returning a fresh
`*stack.Bundle`), and benchkit runs realistic write, read, read-model, and
projection workloads while collecting latency percentiles, throughput,
memory deltas, and storage footprint.

## Quick start

```go
import "github.com/larsartmann/go-cqrs-lite/benchkit/v4"

result, err := benchkit.Run(ctx, benchkit.Config{
    Profile:     benchkit.ProfileDev,
    PayloadSize: 256,
}, func() (*stack.Bundle, error) {
    return sqlite.New(filepath.Join(dir, "bench.db"))
})
if err != nil { ... }

benchkit.PrintReport(os.Stdout, result)
```

## Cross-backend comparison

```go
results, err := benchkit.Compare(ctx, config, map[string]benchkit.Factory{
    "memory": func() (*stack.Bundle, error) { return memory.New() },
    "sqlite": func() (*stack.Bundle, error) { return sqlite.New(":memory:") },
    "pebble": func() (*stack.Bundle, error) { return pebble.New(t.TempDir()) },
})

benchkit.PrintComparison(os.Stdout, results)
```

Output:

```
Backend Comparison
================================================================================
Backend       Write P50    Write P99     Load P50     Load P99    Heap MB    Disk MB
------------------------------------------------------------------------------------
memory            160ns        3.1µs        100ns       16.3µs     1.2 MB        0 B
sqlite           42.8µs      143.3µs       32.6µs      260.8µs     2.5 MB        0 B
```

## CLI

```bash
# Run a single backend
cqrs-bench run --backend sqlite --dsn ":memory:" --profile dev

# Compare all local backends
cqrs-bench compare --profile small --format markdown

# JSON output for CI/trend tracking
cqrs-bench run --backend pebble --dir /tmp/bench --profile small --format json

# CBOR codec
cqrs-bench run --backend pebble --dir /tmp/bench --profile small --codec cbor

# Mixed payload sizes (models real event-size distributions)
cqrs-bench run --backend pebble --profile small --payload-sizes 64,256,1024,4096

# Multi-sample averaging (median of N runs, reduces 20% variance)
cqrs-bench run --backend memory --profile small --repeat 5

# Crash-recovery benchmark (close, reopen, reload all streams)
cqrs-bench run --backend sqlite --dsn /tmp/bench.db --profile small --recovery

# Production replay (benchmark reads + projections on existing data)
cqrs-bench run --backend sqlite --dsn /tmp/existing.db --profile dev --replay

# Postgres backend (requires --dsn)
cqrs-bench run --backend postgres --dsn "postgres://user:pass@localhost:5432/bench?sslmode=disable" --profile dev

# Version
cqrs-bench --version
```

Build: `GOEXPERIMENT=jsonv2 go build -tags "goexperiment.jsonv2" ./cmd/cqrs-bench/...`

See [ADR-0060](../docs/adr/0060-benchkit-design-decisions.md) for design rationale.

## Named profiles

| Profile      | Aggregates | Events/Agg | Total | Concurrent |
| ------------ | ---------: | ---------: | ----: | ---------: |
| `Dev`        |        100 |          5 |   500 |          1 |
| `Small`      |      1,000 |         10 |   10K |          4 |
| `Medium`     |     10,000 |         50 |  500K |         16 |
| `Large`      |    100,000 |        100 |   10M |         32 |
| `Stress`     |     10,000 |        500 |    5M |         64 |
| `WriteHeavy` |     10,000 |        100 |    1M |         32 |
| `ReadHeavy`  |     10,000 |        100 |    1M |         32 |
| `Analytical` |     10,000 |         10 |  100K |         16 |

`Analytical` uses `ReadRatio=0.9` (9 read passes) and `JournalScans=5`
(repeated full-journal scans) to model OLAP-style dashboard/aggregation workloads.

## Metrics collected

- **Write latency**: P50/P75/P90/P95/P99/P100/Mean around `EventSink.Save()`
- **Read latency**: P50-P100 around `EventSource.Load()`
- **Journal scans**: `ReadAll()` and `ReadFrom()` wall time (multi-pass when `Profile.JournalScans > 1`)
- **Read model**: kv.Store `Set()` and `Get()` latency percentiles
- **Projection**: lag, events processed — uses a real kv.Store-backed counting projection (Get+Set per event)
- **Recovery**: close+reopen+reload time and recovered events (when `Config.Recovery` is set)
- **Throughput**: events/sec sustained during write phase
- **Memory**: peak heap allocation via runtime.MemStats
- **Storage**: on-disk database size (when DiskPath configured)
- **CPU**: process user+sys time via `syscall.Getrusage` (Unix), stub on non-Unix

## Testing.B integration

Use `benchkit.RunSuite` to run benchmarks via Go's standard `testing.B`:

```go
func BenchmarkBenchkitSuite_Memory(b *testing.B) {
    benchkit.RunSuite(b, benchkit.Config{
        Profile: benchkit.ProfileDev,
    }, func() (*stack.Bundle, error) {
        return memory.New()
    })
}
```

Run with: `go test -bench=BenchkitSuite -benchtime=1x ./stack/bench/...`

Custom metrics are reported via `b.ReportMetric`: throughput, write/load
latency percentiles, journal scan time, projection events, recovery time.

## Design

The tool mirrors `contracttest.Factory` — the deployer provides the Bundle
factory, and the tool never imports a backend driver. Switching backends is a
one-line factory change.

Latency percentiles use a sorted-slice collector with reservoir sampling
(10K entries) for large workloads, keeping memory bounded at ~80KB per
collector regardless of workload size.

### Warmup isolation

When `Config.Warmup > 0`, the factory is called a **second time** to create a
throwaway Bundle for warmup writes. Warmup events never enter the measurement
store, so journal scans, ReadAll counts, and projection metrics reflect only
the measurement workload. The `Result.WarmupEvents` field reports how many
events were written during warmup.

### Codec-aware payload sizing

The `Generator` uses the configured codec (JSON or CBOR) to measure payload
size, not hardcoded JSON. This ensures payloads are the correct byte size
regardless of encoding. For JSON, payloads are within 2 bytes of target; for
CBOR, within 5 bytes (due to string-header boundary crossings).

### Skipping phases

```go
config := benchkit.Config{
    Profile:          benchkit.ProfileDev,
    SkipReads:        true,   // skip aggregate loads + journal scans
    SkipReadModels:   true,   // skip kv.Store Set/Get benchmark
    SkipProjections:  true,   // skip projection phase
}
```

### Config validation

`Run()` validates the config before starting: `Profile.Streams`,
`Profile.EventsPerStream`, and `Profile.BatchSize` must all be > 0.
Invalid configs return `ErrInvalidConfig` (classified as Rejection).

### Error classification

All benchkit errors use the [go-error-family](https://github.com/larsartmann/go-error-family)
5-family taxonomy: `ErrInvalidConfig` (Rejection), `ErrFactoryFailed` /
`ErrNilBundle` / `ErrIncompleteBundle` (Infrastructure), `ErrWarmupFailed`
(Transient). Phase errors are wrapped with `errorfamily.WrapTransient`.

### DiskSizer interface

Backends that report their own disk size implement `DiskSizer`:

```go
type DiskSizer interface {
    DiskSize() int64 // -1 = not available
}
```

The `durabilityPhase` checks `DiskSize() >= 0` before using the value. When not
available (-1), it falls back to walking `Config.DiskPath`. The Pebble preset
wires `backend.DiskUsage()` (computed from Metrics level sizes + WAL) via
`stack.WithDiskSize()` at construction time.

### CPU measurement

CPU time is measured via `syscall.Getrusage` (Unix) at benchmark start and end.
This provides microsecond resolution, replacing the previous `/proc/self/stat`
parsing which had 10ms tick resolution and returned `n/a` for fast benchmarks.
On non-Unix platforms (Windows, wasm), a stub returns 0.

### Multi-sample averaging (`--repeat N`)

Single-run throughput has ~20-25% variance on the memory backend (GC and OS
thread scheduling). Set `Config.Repeat > 1` to run N iterations and report the
median result with min/max throughput spread. The median result carries full
metrics; `Result.RepeatCount`, `Result.RepeatMin`, `Result.RepeatMax`, and
`Result.RepeatSamples` provide the distribution.

### Mixed payload sizes

Models real workloads where events vary from small status updates to large
events with embedded collections. `NewMixedGenerator(seed, sizes, codec)` picks
a size uniformly at random per event. CLI: `--payload-sizes 64,256,1024,4096`.
The result reports the distribution mean in `PayloadBytes` and the full
distribution in `PayloadSizes`. See
[scaling report](../docs/status/2026-07-24_19-30_event-size-scaling-benchmark.md).
