# Benchmark Interpretation Guide

This guide explains how to interpret go-cqrs-lite benchmark results, what
each metric measures, and how to use the tools for comparative analysis.

## Metrics Reference

### Write Latency / Throughput

**WriteLatency** measures the full event creation pipeline:
event generation → payload encoding → `EventSink.Save`.

**WriteThroughput** is `TotalEvents / WriteDuration`, expressed as events/sec.

This is the **end-to-end write-path metric** — it includes all overhead
(generator, codec, stream ID creation). Use this for capacity planning.

### Raw Sink Latency / Throughput

**RawSinkLatency** measures only `EventSink.Save` with pre-built events.

Events are generated, encoded, and assigned stream IDs BEFORE timing
begins. The timer wraps only the `Save` call.

**RawSinkThroughput** is the throughput of pure backend write capacity.

### Why Both?

The gap between WriteThroughput and RawSinkThroughput reveals the overhead
of event generation and encoding. For memory backends, generation dominates
(50-70% of wall time). For persistent backends, Save dominates and the gap
narrows.

### Read Latency

**LoadLatency** measures `EventSource.Load` — loading all events for one
stream. P50/P95/P99 percentiles are reported.

**ReadAllTime** measures `Journal.ReadAll` — scanning the entire journal
across all streams. Only available when the backend implements `Journal`.

**ReadFromTime** measures `SeekableJournal.ReadFrom` — position-based
reading from a checkpoint.

### Resource Metrics

**Memory.After** — peak heap memory at the end of the run (via runtime.MemStats).

**Memory.Delta** — net memory growth (peak minus baseline).

**CPU.Delta** — total CPU time consumed (via syscall.Getrusage).

**Disk.DatabaseBytes** — on-disk database size (via filesystem walk of DiskPath).

## Running Benchmarks

### Single Backend

```bash
# Quick smoke test (500 events, ~1 second)
cqrs-bench run --backend memory --profile dev

# Full benchmark with median of 5 runs
cqrs-bench run --backend sqlite --dsn bench.db --profile small --repeat 5

# Raw sink comparison
cqrs-bench run --backend pebble --dir /tmp/bench --profile small --format json
```

### Multi-Backend Comparison

```bash
# Compare all in-process backends
cqrs-bench compare --profile small --format markdown

# Compare specific backends
cqrs-bench compare --backends memory,sqlite --profile small --format json
```

### Scaling Sweeps

```bash
# Worker scaling
cqrs-bench sweep --param workers --values 1,2,4,8 --backend memory --profile dev

# Batch size scaling
cqrs-bench sweep --param batchSize --values 1,5,10,20 --backend sqlite --profile small

# Stream length scaling
cqrs-bench sweep --param streamLength --values 5,10,50,100 --backend memory --profile dev
```

### benchstat Comparison

```bash
# Capture baseline
cqrs-bench run --backend memory --profile small --format benchstat > old.txt

# ... make changes ...

# Capture new results
cqrs-bench run --backend memory --profile small --format benchstat > new.txt

# Compare statistically
benchstat old.txt new.txt
```

### Profiling

```bash
# CPU profile
cqrs-bench run --backend memory --profile medium --cpuprofile cpu.prof
go tool pprof cpu.prof

# Heap profile
cqrs-bench run --backend memory --profile medium --memprofile heap.prof
go tool pprof heap.prof
```

## Go Benchmarks (stack/bench)

In addition to the CLI tool, `stack/bench/` contains Go benchmarks that
exercise specific CQRS paths through the standard `testing.B` framework.

```bash
# Run all benchmarks
cd stack/bench && go test -bench=. -benchmem

# Specific benchmark
cd stack/bench && go test -bench=BenchmarkCommandPath_Memory -benchmem

# Contention scaling
cd stack/bench && go test -bench=BenchmarkContention -benchmem
```

| Benchmark                                   | What It Measures                                        |
| ------------------------------------------- | ------------------------------------------------------- |
| `BenchmarkBundle_EventSave`                 | Raw EventSink.Save (same as BenchmarkDirect)            |
| `BenchmarkBundle_EventLoad`                 | EventSource.Load per stream                             |
| `BenchmarkBundle_ReadModelGet`              | kv.Store.Get via Bundle.ReadModels                      |
| `BenchmarkBundle_ReadModelSet`              | kv.Store.Set via Bundle.ReadModels                      |
| `BenchmarkCommandPath_Memory`               | Full decider.Execute pipeline (decide → save → publish) |
| `BenchmarkCommandPath_Concurrent`           | 8-worker concurrent command throughput                  |
| `BenchmarkContention_SameStream`            | Same-stream sequential write throughput                 |
| `BenchmarkContention_SameStream_Concurrent` | Same-stream write contention (1/2/4/8 workers)          |

## Interpretation Tips

### Variance

Single-run throughput on the memory backend has ~20-25% variance due to
GC timing and goroutine scheduling. Always use `--repeat 5` (or more) for
results you intend to compare or publish.

### Warmup

The first run is often slower due to cold caches, page faults, and goroutine
startup. Use `--warmup 100` to write 100 events before measurement begins.
Warmup uses a separate store, so measurement data is not polluted.

### Profile Selection

| Profile     | Events          | Use Case                         |
| ----------- | --------------- | -------------------------------- |
| dev         | 500             | CI smoke test                    |
| small       | 10K             | Development feedback loop        |
| medium      | 500K            | Backend comparison (default)     |
| large       | 10M             | Pre-production capacity planning |
| stress      | 5M              | Write ceiling discovery          |
| write-heavy | 1M (90% writes) | Write-path optimization          |
| read-heavy  | 1M (80% reads)  | Read-path optimization           |
| analytical  | 100K (5x scans) | Journal scan / OLAP workload     |

## Regression Policy

### Thresholds

A regression is defined as:

1. **Throughput drop > 15%** on the same backend/profile/hardware
2. **P99 latency increase > 30%** on the same backend/profile/hardware
3. **Memory increase > 20%** (peak heap) on the same backend/profile/hardware

### Baseline Management

Store baselines as benchstat artifacts in the repository or CI artifacts:

```bash
# Capture baseline after a release
cqrs-bench run --backend memory --profile small --repeat 5 --format benchstat \
  > baselines/memory-small.txt

# Compare against baseline after changes
cqrs-bench run --backend memory --profile small --repeat 5 --format benchstat \
  > current.txt
benchstat baselines/memory-small.txt current.txt
```

### When to Investigate

- Any regression beyond the thresholds above
- New allocations appearing in profiles that weren't there before
- P99/P50 ratio increasing (longer tail latency)
- Disk overhead percentage increasing (less efficient storage)

## Common Pitfalls

1. **Comparing across hardware** — Always compare on the same machine.
   Environment metadata (GoVersion, NumCPU, GOMAXPROCS) is recorded in
   every Result for this reason.

2. **Ignoring GC pauses** — A single GC pause can inflate P99 by 10x.
   Use `--repeat 5` and look at the median, not the max.

3. **Using WriteThroughput for backend comparison** — WriteThroughput
   includes generation overhead. Use RawSinkThroughput for backend
   comparison; it isolates pure write capacity.

4. **Not using warmup** — Cold-start benchmarks are unreliable. Always
   warm up with at least 100 events.

5. **Trusting single runs** — Always repeat. The memory backend has
   ~20% variance between runs.
