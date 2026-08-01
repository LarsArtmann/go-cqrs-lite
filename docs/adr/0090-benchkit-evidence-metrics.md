# ADR-0090: Benchkit Evidence-Grade Metrics

## Status

Accepted

## Context

Benchkit originally measured throughput and P50/P99 latency — the minimum
needed to compare backends. However, two real-world failure modes proved these
metrics insufficient for making trustworthy backend selection decisions:

1. **The metaengine empty-store bug** (session 2026-08-01): Event type strings
   used `"MeBenchItemCreated"` (uppercase) but `metaengine.On()` registered folds
   under the Go struct name (`meBenchItemCreated`, lowercase). Metaengine
   silently skips non-matching event types. The benchmark measured **empty
   stores** for multiple sessions — Apply was a no-op, ExecuteTyped returned
   nil. The throughput numbers looked plausible (~500ns/op) because they
   measured the planner's fold-lookup overhead, not actual writes.

2. **Unexplained P99 spikes**: A backend with excellent P50 (50µs) but terrible
   P99 (5ms) looked identical to one with consistent P99 (100µs) when only
   throughput was reported. The question "is this a GC problem or a backend
   problem?" was unanswerable.

### Root Cause Analysis

The empty-store bug existed because benchmarks had **no correctness assertions**
— nobody verified that Apply actually wrote data. A benchmark that doesn't
verify its results is theater. The fix is not just correcting the event type
strings, but adding structural guards so this class of bug cannot recur.

## Decision

Add evidence-grade metrics to benchkit that make benchmark results
**decision-grade** — trustworthy enough to drive backend selection without
manual verification.

### 1. Correctness Assertions (defense against the empty-store class of bug)

Every metaengine benchmark phase now asserts that operations produced
non-trivial results:

- Counter workload: `ExecuteTyped` must return a non-empty map after Apply
- Map workload: `TypedReader.Get` must find inserted items
- SQLite workload: same correctness check against the SQLite engine

If Apply is silently no-op'd (event type mismatch, fold registration error,
engine not supporting the ADT), the benchmark **fails loudly** instead of
reporting meaningless numbers.

### 2. GC and Allocation Metrics

- **GCCount / GCMaxPause / GCTotalPause / GCMeanPause**: Captured via
  `runtime.ReadMemStats` delta from baseline. Reveals whether tail latency
  originates from the backend or Go's GC.
- **AllocCount / AllocBytes**: Total heap allocations during the benchmark.
  High alloc rates correlate with GC pressure and latency variance.
- **GCPercent**: Percentage of wall-clock time spent in GC pauses.
  > 5% means GC is a significant performance factor.

### 3. Derived Rate Metrics

- **AllocsPerOp** = AllocCount / TotalEvents — directly predicts GC pressure
- **BytesPerOp** = AllocBytes / TotalEvents — per-event allocation footprint
- **TailRatio** = LoadLatency.P99 / LoadLatency.P50 — ratio >3 means
  unpredictable tail latency
- **WriteTailRatio** = WriteLatency.P99 / WriteLatency.P50 — same for write path

### 4. Statistical Reliability

- **RepeatCoV** (coefficient of variation) across repeat runs
- **RepeatIsReliable** gate: CoV < 10% means results are trustworthy

### 5. Multi-Engine Metaengine Benchmark

The Map ADT workload now runs against both Memory and SQLite engines,
giving a direct comparison:

- Memory: planner + fold overhead with zero I/O
- SQLite: SQL query execution + json_extract pushdown cost

This exercises the PushdownScan path (WHERE status=... pushed to SQL),
which is the planner's primary value proposition.

### 6. Soak Test Drift Metrics

- **GCMaxPauseDriftPct**: GC pause degradation across iterations
- **AllocGrowthPct**: Allocation growth (leak detection)

## Consequences

- Benchmark test time increased (~10s slower) due to SQLite engine workload
- Every Result now carries 8+ additional fields (JSON schema grows)
- Correctness assertions catch event-type mismatches at test time, preventing
  the silent-empty-store class of bug from recurring
- PrintComparison and PrintSweep now show evidence-grade columns (TailRatio,
  AllocsPerOp) instead of throughput-only columns
