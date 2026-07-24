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

# Version
cqrs-bench --version
```

Build: `GOEXPERIMENT=jsonv2 go build -tags "goexperiment.jsonv2" ./cmd/cqrs-bench/...`

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

## Metrics collected

- **Write latency**: P50/P75/P90/P95/P99/P100/Mean around `EventSink.Save()`
- **Read latency**: P50-P100 around `EventSource.Load()`
- **Journal scans**: `ReadAll()` and `ReadFrom()` wall time
- **Read model**: kv.Store `Set()` and `Get()` latency percentiles
- **Projection**: lag, events processed (when SeekableJournal available)
- **Throughput**: events/sec sustained during write phase
- **Memory**: peak heap allocation via runtime.MemStats
- **Storage**: on-disk database size (when DiskPath configured)
- **CPU**: process user+sys time (Linux `/proc/self/stat`)

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
