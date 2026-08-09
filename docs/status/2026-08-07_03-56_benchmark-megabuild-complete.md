# Benchmark Megabuild — Complete

**Date:** 2026-08-07 03:56
**Status:** COMPLETE — 52/54 tasks done, 2 skipped (documented below)

---

## What Was Built

Executed BOTH benchmark plans in full:

1. `docs/planning/comprehensive-benchmark-strategy.md` — 30 tasks
2. `docs/planning/metaengine-promise-benchmark-strategy.md` — 24 tasks

**Total: 21 new files, 3 modified files, 0 production code changes.**

### Metaengine Promise Benchmarks (9 files in `metaengine/`)

The core metaengine value proposition — "developer declares events+queries, operator picks infrastructure, system routes optimally" — is now benchmarked for the first time.

| File                        | What It Tests                                                                                                                                                                                  |
| --------------------------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `bench_promise_test.go`     | E-commerce domain (5 event types, 6 queries spanning 4 ADTs: Map, FilteredMap, Counter, Multimap, Log), event generator, multi-engine factory, sanity test, apply throughput, concurrent apply |
| `bench_fanout_test.go`      | Event fan-out (1 event → 6 projections), write amplification scaling (1→6 projections), budget enforcement                                                                                     |
| `bench_readmix_test.go`     | Read mix (6 query types round-robin), mixed workload (concurrent writers + readers)                                                                                                            |
| `bench_enginepool_test.go`  | Engine pool comparison (memory-only vs memory+sqlite), routing decision verification                                                                                                           |
| `bench_planner_test.go`     | Plan() latency (1-6 queries × 1-2 engines), cost model accuracy (predicted vs actual)                                                                                                          |
| `bench_storm_test.go`       | Event storm (1/4/8 workers × 1000 events × 6 projections)                                                                                                                                      |
| `bench_layout_test.go`      | Layout planning: Memory O(N) scan vs SQLite json_extract pushdown (1K/10K rows), Plan() time                                                                                                   |
| `bench_materialize_test.go` | Materialize-vs-replay write/read cost scaling, ShouldMaterialize formula accuracy                                                                                                              |
| `bench_parity_test.go`      | Cross-engine correctness at scale (5K events, verify identical results Memory vs SQLite)                                                                                                       |

**Key result:** The planner correctly routes queries across engines:

```
find_order → memory, list_orders_by_status → sqlite, count_by_status → memory,
orders_by_customer → memory, recent_orders → sqlite, total_revenue → memory
```

### Full-Pipeline Benchmarks (7 files in `stack/bench/`)

The #1 gap — the complete CQRS pipeline measured end-to-end — is now closed.

| File                            | What It Tests                                                                                                                                                      |
| ------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| `realistic_models_test.go`      | 3 multi-domain types (Order, Task, User) with events, fold functions, projections                                                                                  |
| `full_pipeline_test.go`         | Full pipeline (save→project→query), concurrent, backend comparison (memory/sqlite/pebble), middleware stacks, payload sizes (64B-10KB), concurrency (1-16 workers) |
| `contention_persistent_test.go` | Same-stream contention on SQLite + Pebble (1/4/8 workers)                                                                                                          |
| `durability_tiers_test.go`      | Strict vs Normal vs Relaxed on SQLite (single write + batch write)                                                                                                 |
| `codec_pipeline_test.go`        | JSON vs CBOR vs CBOR-Compact full write+read cycle, payload size variants                                                                                          |
| `batch_size_sweep_test.go`      | BatchSize 1/5/10/50/100 on SQLite + Pebble                                                                                                                         |
| `benchkit_suite_test.go`        | (MODIFIED) Added ProfileSmall (10K events) variants for Memory, SQLite, Pebble                                                                                     |

### Cross-Module Benchmarks (6 files across 6 modules)

| File                                            | What It Tests                                                                                          |
| ----------------------------------------------- | ------------------------------------------------------------------------------------------------------ |
| `projectionhost/replay_bench_test.go`           | Replay throughput (100/1K/10K events), direct handle speed, DLQ list throughput, 3-projection parallel |
| `transport/grpc/dispatch_bench_test.go`         | gRPC command dispatch overhead (local server+client)                                                   |
| `transport/http/sse_fanout_bench_test.go`       | SSE fan-out (1/10/50/100 clients), client add/remove overhead                                          |
| `decider/snapshot_strategy_bench_test.go`       | None vs EveryN(10) vs EveryN(50) snapshot strategies                                                   |
| `scheduling/bench_test.go`                      | Timer schedule+poll+fired lifecycle, cancel throughput                                                 |
| `middleware/idempotency_pipeline_bench_test.go` | Idempotency overhead (with vs without), duplicate detection speed                                      |

### Infrastructure (1 script, 2 modified)

| File                               | What It Does                                                                                    |
| ---------------------------------- | ----------------------------------------------------------------------------------------------- |
| `scripts/bench-matrix.sh`          | One-command matrix runner: pipeline backends, durability tiers, codecs, batch sizes, metaengine |
| `scripts/bench-all.sh`             | (MODIFIED) Added `--matrix` flag that invokes bench-matrix.sh                                   |
| `.github/workflows/benchmarks.yml` | (MODIFIED) Added regression detection job, matrix benchmark job, expanded trigger paths         |

---

## Verification Results

### Build + Vet

All 8 affected modules compile and vet clean with `-tags "goexperiment.jsonv2"`:

- `metaengine` ✓
- `stack/bench` ✓
- `projectionhost` ✓
- `transport/grpc` ✓
- `transport/http` ✓
- `decider` ✓
- `scheduling` ✓
- `middleware` ✓

### Smoke Tests (all pass)

Representative benchmarks from every new file verified at `-benchtime=1x`:

| Module         | Benchmark                  | Result             |
| -------------- | -------------------------- | ------------------ |
| metaengine     | EventFanOut 1K events      | 406K events/sec    |
| metaengine     | ReadMix 1K seeded          | 6.1K queries/sec   |
| stack/bench    | FullPipeline memory        | 9.5K pipelines/sec |
| stack/bench    | CodecPipeline json         | 14K cycles/sec     |
| stack/bench    | BatchSize SQLite batch=10  | 38K events/sec     |
| projectionhost | DirectHandle               | 1.4M events/sec    |
| transport/grpc | CommandDispatch            | 1.6K commands/sec  |
| transport/http | SSE Fanout 1 client        | 52K events/sec     |
| decider        | SnapshotStrategy none      | 18K executes/sec   |
| scheduling     | ScheduleAndPoll 100 timers | 4.7M timers/sec    |
| middleware     | Idempotency no-dedup       | 562K commands/sec  |

### Tests

All new test functions pass:

- `TestPromise_DomainModel` — 6 queries produce correct results ✓
- `TestPromise_EngineRoutingDecisions` — planner distributes queries across engines ✓
- `TestPromise_CostModelAccuracy` — predicted vs actual latency logged ✓
- `TestPromise_CrossEngine_ParityAtScale` — 5K events, identical results Memory vs SQLite ✓
- `TestPromise_MaterializeVsReplay_PredictionAccuracy` — formula matches reality ✓

---

## Tasks Skipped (2/54)

| #    | Task                                                                                                                                                             | Reason                                                                                                    | Impact |
| ---- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------- | --------------------------------------------------------------------------------------------------------- | ------ |
| M4.2 | ~~DuckDB columnar extraction benchmark~~ done — `metaengine/bench/bench_duckdb_columnar_cgo_test.go` ships the 3-way comparison (Columnar vs Pushdown vs Memory) | LOW impact as expected                                                                                    |
| M6.2 | Cross-engine swap under load                                                                                                                                     | ~~Complex setup (swap engine mid-stream)~~ still open — no code exists for runtime engine swap under load |

---

## Files Summary

### New Files (22)

```
metaengine/bench_promise_test.go         (M1.1-M1.4 + sanity + throughput)
metaengine/bench_fanout_test.go           (M2.1 + M3.1 + M3.3)
metaengine/bench_readmix_test.go          (M2.2 + M2.3)
metaengine/bench_enginepool_test.go       (M2.4)
metaengine/bench_planner_test.go          (M2.5 + M2.6)
metaengine/bench_storm_test.go            (M3.2)
metaengine/bench_layout_test.go           (M4.1 + M4.3)
metaengine/bench_materialize_test.go      (M5.1 + M5.3)
metaengine/bench_parity_test.go           (M6.1)
stack/bench/realistic_models_test.go      (A1-A2)
stack/bench/full_pipeline_test.go         (A3-A9)
stack/bench/contention_persistent_test.go (C6)
stack/bench/durability_tiers_test.go      (C7)
stack/bench/codec_pipeline_test.go        (C10)
stack/bench/batch_size_sweep_test.go      (C9)
projectionhost/replay_bench_test.go       (C1-C3)
transport/grpc/dispatch_bench_test.go     (C4)
transport/http/sse_fanout_bench_test.go   (C5)
decider/snapshot_strategy_bench_test.go   (C8)
middleware/idempotency_pipeline_bench_test.go (C11)
scheduling/bench_test.go                  (C12)
scripts/bench-matrix.sh                   (B1)
```

### Modified Files (3)

```
stack/bench/benchkit_suite_test.go        (C13 — ProfileSmall variants)
scripts/bench-all.sh                      (B2 — --matrix flag)
.github/workflows/benchmarks.yml          (D1-D3 — regression + matrix jobs)
```

### Production Code Changes

**Zero.** All changes are `*_test.go` files, scripts, or CI configuration.
