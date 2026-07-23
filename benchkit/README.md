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
```

Build: `cd cmd/cqrs-bench && GOWORK=off go build -tags "goexperiment.jsonv2" .`

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
