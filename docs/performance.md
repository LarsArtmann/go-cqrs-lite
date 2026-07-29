# Performance

Benchmark data for go-cqrs-lite modules. All numbers measured on AMD Ryzen AI MAX+ 395, 96 GB RAM, Go 1.26.x unless otherwise noted.

> **Reproducing:** Run `nix run .#build && go test -tags "goexperiment.jsonv2" -bench=. -benchmem ./...` per module, or use `cqrs-bench` for end-to-end workload profiles (see below).

---

## Core Throughput

End-to-end scale benchmarks (10K aggregates, 100 events each, concurrent):

| Operation                                  | Throughput              | Source                                |
| ------------------------------------------ | ----------------------- | ------------------------------------- |
| Event publish (MemoryBus)                  | **43.7M events/sec**    | `integration/scale_benchmark_test.go` |
| Command dispatch (100 handlers)            | **6.5M commands/sec**   | `integration/scale_benchmark_test.go` |
| Query dispatch (1K handlers)               | **8.0M queries/sec**    | `integration/scale_benchmark_test.go` |
| Event save (memory store)                  | **2.6M events/sec**     | `integration/scale_benchmark_test.go` |
| Full pipeline (cmd→event→projection→query) | **160K aggregates/sec** | `integration/scale_benchmark_test.go` |

---

## Micro-Benchmarks

| Module     | Operation      |  ns/op | B/op | allocs/op |
| ---------- | -------------- | -----: | ---: | --------: |
| event      | NewEvent       |    201 |  384 |         3 |
| event      | DecodePayload  |    419 |  560 |        10 |
| id         | New (ULID)     |    100 |   16 |         1 |
| id         | Parse          |     17 |    0 |         0 |
| command    | New            |     50 |  208 |         2 |
| query      | New            |    0.6 |    0 |         0 |
| dispatcher | Dispatch       |     24 |    0 |         0 |
| memory     | Store Save     |    583 |  736 |         9 |
| memory     | Bus Publish    |     66 |   48 |         3 |
| signing    | HMAC Sign      |    662 |  864 |        12 |
| signing    | Ed25519 Sign   | 13,486 |  416 |         7 |
| signing    | Ed25519 Verify | 30,369 |  352 |         6 |

Full benchmark output: `benchmarks/2026-06-02_20-18-40.md`

---

## Storage Backend Comparison

Benchkit results (small profile: 1K aggregates × 10 events = 10K total):

| Backend       | Throughput           | Notes                         |
| ------------- | -------------------- | ----------------------------- |
| Memory        | ~227K events/sec     | CPU-bound below 256B payloads |
| Pebble + CBOR | **80.4K events/sec** | 6.5× faster than SQLite       |
| SQLite        | 12.2K events/sec     | WAL mode + busy_timeout       |

### Event-size scaling (memory backend)

| Payload Size | Throughput | Write P99 | Heap      |
| ------------ | ---------- | --------- | --------- |
| 64 B         | 227.5K/s   | 3.3µs     | 1.3 MB    |
| 256 B        | 189.9K/s   | 2.5µs     | 1.3 MB    |
| 1 KB         | ~170K/s    | 3-4µs     | 1.3-17 MB |
| 4 KB         | 72.9K/s    | 4.9µs     | 39 MB     |
| 16 KB        | 32.8K/s    | 7.4µs     | 208 MB    |

Throughput is flat below ~256B (CPU-bound), declines linearly above 1KB (bandwidth-bound).

### Event-size scaling (Pebble backend)

| Payload Size | Throughput | Write P99 | Disk    |
| ------------ | ---------- | --------- | ------- |
| 64 B         | 68.3K/s    | 315µs     | 9.6 MB  |
| 256 B        | 68.0K/s    | 268µs     | 11.1 MB |
| 1 KB         | ~50K/s     | 527µs     | 20.9 MB |
| 4 KB         | 20.4K/s    | 799µs     | 29.1 MB |
| 16 KB        | 6.3K/s     | 2.83ms    | 40.9 MB |

Full data: `docs/status/2026-07-24_19-30_event-size-scaling-benchmark.md`

### SQLite event store

| Operation    |  ns/op |   B/op | allocs/op |
| ------------ | -----: | -----: | --------: |
| Save (batch) | 41,042 |  4,080 |        92 |
| Load         | 48,505 | 20,233 |       554 |

SQLite WAL mode with `synchronous=NORMAL` gives **3-10× better write throughput** without durability loss.

---

## Codec Comparison

| Codec                  | Encode   | Decode         | Payload Size    | Notes                 |
| ---------------------- | -------- | -------------- | --------------- | --------------------- |
| JSON                   | Baseline | Baseline       | Baseline        | Human-readable        |
| CBOR                   | ~Same    | **66% faster** | **19% smaller** | Binary, deterministic |
| CBOR Compact (toarray) | ~Same    | **72% faster** | **43% smaller** | Positional arrays     |

CBOR-to-JSON transcoding (for browser/SSE delivery): 7.2µs/op for nested payloads, 208µs/op for 100 SSE clients fan-out.

---

## Key Optimizations

| Optimization                       | Impact                                                   | Where                    |
| ---------------------------------- | -------------------------------------------------------- | ------------------------ |
| Decider hot-state cache (LRU)      | **7.4× faster Load** (2090→283 ns/op, 500-event streams) | `decider.WithStateCache` |
| Listing sorted-index cache         | **25× faster listing**                                   | `listing` package        |
| Projection event-type caching      | −10.5M allocs, −12% time                                 | `projectionhost`         |
| Event lazy metadata map            | −1 alloc per event                                       | `event.New()`            |
| MemoryStore Load double-copy       | Halved allocs                                            | `storage/memory`         |
| MemoryBus pre-computed middleware  | Zero per-publish closure alloc                           | `event.Bus`              |
| CircuitBreaker lock-free fast path | `atomic.Int32` state                                     | `middleware`             |
| Pebble key-based skip              | Avoids CBOR-deserializing skipped events                 | `storage/pebble`         |
| SQL multi-VALUES INSERT            | 99 events/batch                                          | `storage/eventstore`     |
| Bundle field access                | **~0.20 ns/op** (zero-overhead)                          | `stack` presets          |

---

## Soak Testing

The `cqrs-bench --soak` flag and `benchkit.Soak` function run sustained workloads (default 30s) tracking:

- **Heap growth** — memory leaks (10s memory soak: 0 bytes growth over 452 iterations)
- **Throughput drift** — performance degradation over time
- **P99 latency drift** — tail latency inflation

Soak tests use relaxed thresholds under `-race` (3× multiplier).

---

## Benchmarking with cqrs-bench

```bash
# Quick dev benchmark (500 events, 1 goroutine)
cqrs-bench run --backend memory --profile dev

# Compare backends
cqrs-bench compare --backends memory,sqlite,pebble

# Production-scale (500K events, 16 goroutines)
cqrs-bench run --backend sqlite --profile medium --format json

# 5-minute soak test
cqrs-bench run --backend pebble --profile small --soak 5m

# Repeat for stable results (median of 5)
cqrs-bench run --backend memory --profile small --repeat 5
```

### Workload Profiles

| Profile     | Aggregates | Events/Agg | Total | Concurrency | Read Ratio |
| ----------- | ---------- | ---------- | ----- | ----------- | ---------- |
| dev         | 100        | 5          | 500   | 1           | 0.2        |
| small       | 1,000      | 10         | 10K   | 4           | 0.3        |
| medium      | 10,000     | 50         | 500K  | 16          | 0.4        |
| large       | 100,000    | 100        | 10M   | 32          | 0.5        |
| stress      | 10,000     | 500        | 5M    | 64          | 0.2        |
| write-heavy | 10,000     | 100        | 1M    | 32          | 0.1        |
| read-heavy  | 10,000     | 100        | 1M    | 32          | 0.8        |

---

## Metaengine Cost Model

The metaengine planner uses calibrated cost constants to select the optimal engine per query:

| Operation | Memory Engine | SQLite Engine | Pebble Engine |
| --------- | ------------- | ------------- | ------------- |
| MapSet    | 466 ns        | 6,548 ns      | 1,785 ns      |
| MapGet    | 21 ns         | 4,960 ns      | 708 ns        |

Planner constants: `MemoryNsPerOp=500`, `SQLiteNsPerOp=7000`, `PebbleNsPerOp=1200`.
The planner selects Memory for O(1) point lookups and SQLite for persistent/complex queries.

Pebble is **7x faster** than SQLite on point reads and **3.7x faster** on writes (LSM vs B-tree).

### Layout Planning: Naive vs Planned (10K rows)

| Pattern        | Naive (json_extract) | Planned (indexed) | Speedup  |
| -------------- | -------------------- | ----------------- | -------- |
| FilterByStatus | ~91,500 ns           | ~45,500 ns        | **2.0x** |
| FilterAndSort  | ~17,050,000 ns       | ~1,700,000 ns     | **10x**  |
| PointLookup    | ~15,200 ns           | ~11,400 ns        | 1.3x     |

The core hypothesis is validated: deployment-time layout optimization produces measurably
better query performance. The planned engine extracts declared filter/sort fields into
indexed columns at DDL time, replacing `json_extract()` with direct column references.
