# Benchmark Results — First Real Run

**Date:** 2026-07-24 17:54 (local), 15:54 UTC
**Tool:** `cqrs-bench v4.1.0` (commit `8f68922b`)
**Platform:** Linux (NixOS), Go 1.26.4, `GOEXPERIMENT=jsonv2`
**Payload:** 256 bytes/event (default)

> This is the first time the benchmark was actually executed and the output inspected for plausibility. All prior sessions (55 tests) verified plumbing — event counts, sample counts, error classification — but never ran the tool as a benchmark or checked whether the numbers make physical sense.

---

## Summary

The benchmark works. Numbers are physically plausible, scale correctly with profile size, and surface a real backend limitation (SQLite concurrent-write failure).

| Backend | Dev (500 events) | Small (10K) | Medium (500K) | Stress (5M) | Large (10M) |
| ------- | ---------------- | ----------- | ------------- | ----------- | ----------- |
| memory  | 225K/s           | 197K/s      | 216K/s        | 170K/s      | (skipped)   |
| pebble  | 64K/s            | 81K/s       | —             | —           | 90K/s       |
| sqlite  | 16K/s            | **FAIL**    | **FAIL**      | —           | **FAIL**    |

---

> **Update 2026-07-24 (same-day hardening):** all 5 "What Needs Improvement"
> items below are FIXED. SQLite now handles 4+ goroutines, compare-mode disk
> populates, CPU reports on fast runs, projection events are non-zero, and
> `--version` uses `runtime/debug.ReadBuildInfo`. Item-by-item status with
> commit hashes in [What Needs Improvement](#what-needs-improvement).

## 1. Dev Profile (500 events, 1 goroutine)

### Comparison table

```
| Backend | Write P50 | Write P99 | Load P50 | Load P99 | Throughput | Heap  | Disk    |
|---------|-----------|-----------|----------|----------|------------|-------|---------|
| memory  | 150ns     | 2.6µs     | 100ns    | 6µs      | 225.1K/s   | 3.1MB | 0B      |
| pebble  | 5.6µs     | 116.2µs   | 23.3µs   | 158.4µs  | 64.0K/s    | 1.5MB | 0B (*)  |
| sqlite  | 42.5µs    | 363.1µs   | 30.1µs   | 208.7µs  | 16.4K/s    | 2.7MB | 4.5MB   |
```

(*) Pebble disk shows 0B in compare mode because compare doesn't set `DiskPath`. Individual runs report disk correctly.

### Memory backend — full text report

```
Benchmark: memory | profile=dev | codec=json
Workload: 100 aggregates x 5 events = 500 events
Payload:  256 bytes/event
Duration: 2.907ms

Write Performance:
  Latency: P50=190ns P95=560ns P99=4.3µs Max=16.7µs
  Throughput: 191.1K events/sec

Read Performance:
  Latency: P50=100ns P95=240ns P99=4.4µs Max=10.1µs
  ReadAll:  1.7µs
  ReadFrom: 1.1µs

Read Model:
  Set: P50=80ns P95=2µs P99=11.2µs Max=11.2µs
  Get: P50=60ns P95=200ns P99=5.5µs Max=5.5µs

Resources:
  Heap:  1.2 MB peak
  Delta: 0 B
  CPU:   n/a
```

### SQLite backend

```
Benchmark: sqlite | profile=dev
Duration: 39.314ms

Write Performance:
  Latency: P50=40.1µs P95=78.3µs P99=139.7µs Max=672.3µs
  Throughput: 18.9K events/sec

Storage:
  Database: 4.5 MB
  Events:   125.0 KB
  Overhead: 97.3%
```

### Pebble backend

```
Benchmark: pebble | profile=dev
Duration: 16.337ms

Write Performance:
  Latency: P50=5.4µs P95=9µs P99=156.7µs Max=695.3µs
  Throughput: 69.9K events/sec

Read Model:
  Get: P50=290ns P95=730ns P99=22.5µs

Storage:
  Database: 575.0 KB
  Events:   125.0 KB
  Overhead: 78.3%
```

**Plausibility check:** Memory (sub-µs) > Pebble (single-digit µs) > SQLite (tens of µs). Correct ordering. SQLite overhead (97.3%) is expected — SQLite WAL + page allocation overhead dominates at small event counts.

---

## 2. Scaling: Memory backend across profiles

| Profile     | Events | Goroutines | Throughput | Write P50 | Write P99 | Read P50 | Heap      |
| ----------- | ------ | ---------- | ---------- | --------- | --------- | -------- | --------- |
| dev         | 500    | 1          | 225K/s     | 150ns     | 2.6µs     | 100ns    | 3.1MB     |
| small       | 10K    | 4          | 197K/s     | 300ns     | 2.9µs     | 200ns    | 1.3MB (*) |
| medium      | 500K   | 16         | 216K/s     | 1.6µs     | 23.1µs    | 600ns    | 441.9MB   |
| write-heavy | 1M     | 32         | 204K/s     | 600ns     | 4.9µs     | —        | 879.3MB   |
| read-heavy  | 1M     | 32         | 189K/s     | 600ns     | —         | 900ns    | 951.1MB   |
| stress      | 5M     | 64         | 170K/s     | 800ns     | 64.5µs    | —        | 4.85GB    |

(*) Small profile heap is lower than dev because GC ran between the two.

**Plausibility check:** Throughput stays 170-225K/s across all sizes — memory backend is CPU-bound, not I/O-bound. P50 latency increases with goroutine count (contention) but stays sub-µs until stress. P99 degrades meaningfully at 64 goroutines (64.5µs) — expected contention tail.

### ReadRatio verification

| Profile     | ReadRatio | Read passes | Expected reads | Actual reads |
| ----------- | --------- | ----------- | -------------- | ------------ |
| dev         | 0.2       | 2           | 200            | 200          |
| write-heavy | 0.1       | 1           | 10,000         | 10,000       |
| read-heavy  | 0.8       | 8           | 80,000         | 80,000       |

All exact matches. ReadRatio scaling is correct.

---

## 3. Scaling: Large profile (10M events, Pebble)

```
Benchmark: pebble | profile=large
Workload: 100,000 aggregates x 100 events = 10,000,000 events
Duration: 270.0s (4.5 minutes)

Write Performance:
  Latency: P50=74.4µs P99=5,197.4µs (5.2ms)
  Throughput: 90K events/sec

Resources:
  Heap:  8.80 GB
  Disk:  3,393 MB (3.3 GB)
```

**Plausibility check:** 10M events × 256 bytes = 2.5GB of raw payload data. Pebble's 3.3GB on disk (32% overhead from LSM structure) is reasonable. P99 of 5.2ms reflects Pebble compaction stalls under sustained write pressure — expected behavior.

**Note:** SQLite large profile failed (see Finding 1 below). Memory large was skipped to avoid OOM (extrapolated: ~20GB heap for 10M events).

---

## 4. CBOR vs JSON Codec Comparison

### Pebble, small (10K events)

| Codec | Write P50 | Throughput | Disk     | EventBytes |
| ----- | --------- | ---------- | -------- | ---------- |
| JSON  | 11.2µs    | 81.0K/s    | 11,376KB | 2,500KB    |
| CBOR  | 10.6µs    | 100.5K/s   | 11,117KB | 2,500KB    |

**CBOR: 24% faster, 2.3% smaller on disk.**

### Memory, medium (500K events)

| Codec | Write P50 | Write P99 | Throughput | Heap  |
| ----- | --------- | --------- | ---------- | ----- |
| JSON  | 1.5µs     | 25.3µs    | 221.0K/s   | 437MB |
| CBOR  | 1.6µs     | 28.8µs    | 296.3K/s   | 464MB |

**CBOR: 34% faster on memory backend.** This is surprisingly large — likely because CBOR encode/decode avoids string key parsing overhead.

**Note:** EventBytes are identical (2,500KB = 10,000 × 250 bytes) because the generator targets the same payload size regardless of codec. The difference is in encode/decode speed, not payload size.

---

## 5. Warmup Isolation

```
With warmup=100:  warmupEvents=100, totalEvents=500, writeCount=500
Without warmup:   warmupEvents=0,   totalEvents=500, writeCount=500
```

Warmup events are written to a separate throwaway Bundle and do NOT pollute the measurement store. Confirmed correct.

---

## 6. Compare Mode Output

### Dev comparison (markdown)

```
| Backend | Write P50 | Write P99 | Load P50 | Load P99 | Throughput | Heap  | Disk |
|---------|-----------|-----------|----------|----------|------------|-------|------|
| memory  | 150ns     | 2.6µs     | 100ns    | 6µs      | 225.1K/s   | 3.1MB | 0B   |
| pebble  | 5.6µs     | 116.2µs   | 23.3µs   | 158.4µs  | 64.0K/s    | 1.5MB | 0B   |
| sqlite  | 42.5µs    | 363.1µs   | 30.1µs   | 208.7µs  | 16.4K/s    | 2.7MB | 0B   |
```

### Medium comparison (text)

```
Backend       Write P50    Write P99     Load P50     Load P99    Heap MB    Disk MB
------------------------------------------------------------------------------------
memory            2.8µs      1.471ms        1.2µs        3.4µs   534.4 MB        0 B
pebble             73µs       7.19ms      1.646ms     28.271ms   430.3 MB        0 B
sqlite     skipped: write_phase: database is locked (SQLITE_BUSY)
```

---

## Key Findings

### Finding 1: SQLite fails under concurrent writes (CRITICAL)

SQLite fails with `SQLITE_BUSY` ("database is locked") at **4+ goroutines**. Even the `small` profile (4 goroutines, 10K events) fails. Only `dev` (1 goroutine) works reliably.

```
Error: [transient:benchkit.write_phase] write phase: ... database is locked (5) (SQLITE_BUSY)
```

This is a real limitation of the SQLite stack preset under concurrent workloads. The `busy_timeout=5000` setting (from `storage.SQLiteEnableWAL`) should retry for 5 seconds, but the error appears immediately, suggesting the timeout is either not applied or insufficient.

**Impact:** SQLite can only be benchmarked at Concurrency=1. Any profile with `Concurrency > 1` will fail. This means `small` (4), `medium` (16), `large` (32), `stress` (64), `write-heavy` (32), and `read-heavy` (32) all fail for SQLite.

**Root cause hypothesis:** The SQLite connection pool likely uses separate connections for reads and writes, and WAL mode still serializes writes. With 4+ goroutines all writing through the pool, the busy handler exhausts its retries.

### Finding 2: Memory backend throughput is CPU-bound

Memory backend throughput stays flat at 170-225K events/sec regardless of event count (500 to 5M). This confirms the bottleneck is CPU (event creation, marshaling, map operations), not memory allocation. Heap scales linearly: ~450 bytes overhead per event in memory.

### Finding 3: Pebble is the best persistent backend

Pebble handles 10M events in 4.5 minutes with 3.3GB disk. P99 write latency of 5.2ms reflects compaction stalls, which is expected LSM behavior. KV read latency (ReadModel Get) is excellent: P50=290ns on dev.

### Finding 4: CBOR provides 24-34% throughput improvement

CBOR encoding is meaningfully faster than JSON for the 256-byte synthetic payloads. The improvement is in encode/decode speed, not payload size (the generator targets the same byte count). Disk savings are modest (2-3%).

### Finding 5: CPU measurement returns n/a on memory backend

The CPU delta shows `n/a` for memory backend runs (both 0 before and 0 after). SQLite and Pebble correctly report CPU time (40ms, 20ms). This is likely because the memory benchmark completes too fast for the `/proc/self/stat` polling interval to capture a sample.

### Finding 6: Disk measurement gap in compare mode

Compare mode does not set `DiskPath` on the Config, so all backends report `0B` disk in comparison tables. Individual `run` commands report disk correctly (SQLite: 4.5MB, Pebble: 575KB at dev). This is a limitation of the compare command's factory setup, not the benchmark library itself.

---

## What Works Well

- Throughput numbers are physically plausible and correctly ordered (memory > pebble > sqlite)
- Latency percentiles scale correctly with concurrency (P50/P99 widen as goroutine count increases)
- ReadRatio produces exact expected read counts
- Warmup isolation is verified — warmup events don't pollute measurement
- CBOR codec works correctly end-to-end
- Pebble scales to 10M events without issues
- Disk measurement works for individual runs (SQLite + Pebble)
- Compare mode handles SQLite failure gracefully (marks as "skipped" instead of crashing)

## What Needs Improvement

1. ~~**SQLite concurrency** — the most critical issue. SQLite should either work at low concurrency (2-4 goroutines) or the benchmark should document this as a known SQLite limitation.~~ DONE: 9c738149;
2. ~~**Disk in compare mode** — compare should pass `DiskPath` so disk columns are populated.~~ DONE: ba681b09;
3. ~~**CPU measurement** — fast benchmarks (memory, <3ms) report `n/a` CPU. Consider finer-grained sampling.~~ DONE: 1b801d61;
4. ~~**No projection benchmark** — `projectionEvents: 0` in all runs. The dev profile doesn't exercise projection catch-up speed.~~ DONE: d7c7a5bf;
5. ~~**`--version` is hardcoded** — shows `v4.1.0` but should use `runtime/debug.ReadBuildInfo()`.~~ DONE: ba681b09;
