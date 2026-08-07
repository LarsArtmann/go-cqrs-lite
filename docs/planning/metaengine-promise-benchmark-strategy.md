# Metaengine Promise Benchmark Strategy

**Date:** 2026-08-07
**Status:** PLANNING

---

## The Core Problem

The metaengine's entire reason for existence is:

> **Developer** declares Events + Queries. **Operator** picks infrastructure. The **system** routes each query to the optimal engine, applies the best layout, and serves everything efficiently.

**This is NEVER benchmarked.** What we benchmark instead:

| What we DO benchmark                     | What we DON'T benchmark                                        |
| ---------------------------------------- | -------------------------------------------------------------- |
| Individual MapSet on Memory              | 6 queries of 4 different ADTs fanning out from one event       |
| Individual MapGet on SQLite              | Planner routing 6 queries to 3 engines in one Plan()           |
| FilteredScan Memory vs SQLite (1 query)  | Write amplification: 1 event → 5 projection writes             |
| Counter increment on Pebble              | Event storm: 1000 mixed events → 6 projections concurrently    |
| Calibration constants per engine         | Planner decision latency: Plan() with 10 queries × 5 engines   |
| Cost model is "reasonable" (loose check) | Cost model accuracy: predicted vs actual latency per query     |
| Layout scan vs unplanned (1 query)       | Layout planning payoff across a realistic multi-query scenario |

Every dimension exists in isolation. **Nobody has ever combined them.**

---

## What the Metaengine Promise Actually Requires

### The "Dream Scenario" (currently untested)

```
Developer declares:
  find_order          (Map ADT, point lookup)
  list_orders_status  (FilteredMap, FilterOnField "status")
  count_by_status     (Counter ADT, aggregate)
  orders_by_customer  (Multimap ADT, secondary index)
  recent_orders       (Log ADT, time-ordered)
  order_totals        (Counter ADT, sum)

Operator provides:
  Memory engine   (fast point lookups, O(1))
  SQLite engine   (filtered scan pushdown, O(logN))
  Pebble engine   (LSM point reads, high throughput)

System does:
  1. Plan() routes each of 6 queries to the best engine
  2. OrderCreated event fans out to ALL 6 projections
  3. All 6 queries return correct results
  4. Latency matches the cost model's prediction
```

**This scenario has ZERO benchmarks.** That is the gap.

---

## The Plan: 24 Tasks (≤12 min each)

### Workstream M1: The Multi-Query Domain Model

The shared domain types + query declarations used by all subsequent benchmarks.

| #    | Task                                                                                                                                                                                   | Impact   | Effort | Time |
| ---- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | -------- | ------ | ---- |
| M1.1 | Define e-commerce domain events: OrderCreated, ItemAdded, OrderShipped, OrderCancelled, OrderPaid (realistic fields, 256B-1KB payloads)                                                | CRITICAL | 10m    | 10m  |
| M1.2 | Declare 6 metaengine queries spanning 4 ADTs: find_order (Map), list_by_status (FilteredMap), count_by_status (Counter), by_customer (Multimap), recent (Log), total_revenue (Counter) | CRITICAL | 12m    | 12m  |
| M1.3 | Build event generator: produces realistic mixed event stream (60% OrderCreated, 20% ItemAdded, 10% each Ship/Cancel/Paid) with deterministic seed                                      | HIGH     | 10m    | 10m  |
| M1.4 | Build multi-engine factory helper: returns []Engine for configurable engine pools (memory-only, memory+sqlite, memory+sqlite+pebble)                                                   | HIGH     | 8m     | 8m   |

### Workstream M2: The Core Promise Benchmarks

These benchmarks test combinations — the actual metaengine value proposition.

| #    | Task                                                                                                                                                                                           | Impact   | Effort | Time |
| ---- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | -------- | ------ | ---- |
| M2.1 | **BenchmarkMultiQuery_EventFanOut**: seed N events, each event fans out to 6 projections. Measure events/sec and per-projection write latency. Compare single-engine vs multi-engine.          | CRITICAL | 12m    | 12m  |
| M2.2 | **BenchmarkMultiQuery_ReadMix**: after seeding 10K events, execute all 6 query types in round-robin. Measure combined read throughput + per-query latency.                                     | CRITICAL | 12m    | 12m  |
| M2.3 | **BenchmarkMultiQuery_MixedWorkload**: concurrent writers + readers. Writers Apply events, readers ExecuteTyped queries. Measure read latency under write load.                                | HIGH     | 10m    | 10m  |
| M2.4 | **BenchmarkMultiQuery_EnginePoolComparison**: same 6 queries, run against 3 engine pools (memory-only, mem+sqlite, mem+sqlite+pebble). Compare routing decisions + throughput.                 | CRITICAL | 12m    | 12m  |
| M2.5 | **BenchmarkPlanner_PlanLatency**: measure Plan() call time itself with 1, 3, 6, 10, 20 queries × 1, 2, 3, 5 engines. The planner must be fast or it's useless at startup.                      | HIGH     | 10m    | 10m  |
| M2.6 | **BenchmarkPlanner_CostModelAccuracy**: for each of 6 queries, compare predicted latency (Cost.EstimatedLatencyMs) vs actual measured latency after seeding 10K events. Report accuracy ratio. | HIGH     | 12m    | 12m  |

### Workstream M3: Write Amplification & Event Storm

| #    | Task                                                                                                                                                                          | Impact | Effort | Time |
| ---- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ------ | ------ | ---- |
| M3.1 | **BenchmarkWriteAmplification_Scaling**: measure Apply throughput as projection count scales 1→2→4→6→10. One event type triggers N projection writes. Shows the fan-out cost. | HIGH   | 10m    | 10m  |
| M3.2 | **BenchmarkEventStorm_Concurrent**: 8 goroutines each Apply 1000 mixed events to a 6-query store. Measure total throughput + contention behavior.                             | HIGH   | 10m    | 10m  |
| M3.3 | **BenchmarkWriteAmplification_BudgetEnforcement**: with WithWriteAmplificationBudget(3), verify the diagnostic fires and measure whether it affects throughput.               | MEDIUM | 8m     | 8m   |

### Workstream M4: Layout Planning Payoff

| #    | Task                                                                                                                                                                                                 | Impact | Effort | Time |
| ---- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ------ | ------ | ---- |
| M4.1 | **BenchmarkLayoutPlanning_UnplannedVsPlanned**: 10K rows, FilterOnField+SortOnField. Compare unplanned (json_extract) vs planned (extracted columns + index). Measure scan latency + Apply overhead. | HIGH   | 12m    | 12m  |
| M4.2 | **BenchmarkLayoutPlanning_ColumnarExtraction**: WithColumnarLayout on DuckDB (if CGo available, skip otherwise). Measure vectorized aggregation speedup vs row-based.                                | MEDIUM | 10m    | 10m  |
| M4.3 | **BenchmarkLayoutPlanning_BuildTime**: measure how long BuildLayoutPlan + ApplyLayoutPlan takes. This is a one-time cost at Plan() — must be fast.                                                   | MEDIUM | 8m     | 8m   |

### Workstream M5: Materialize-vs-Replay Validation

| #    | Task                                                                                                                                                                                  | Impact | Effort | Time |
| ---- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ------ | ------ | ---- |
| M5.1 | **BenchmarkMaterializeVsReplay_ReadHeavy**: 100K events, 90% reads. Compare materialized projection (O(1) read) vs replay-on-read (O(N) fold). Measure the crossover point.           | HIGH   | 12m    | 12m  |
| M5.2 | **BenchmarkMaterializeVsReplay_WriteHeavy**: 100K events, 10% reads. Materialization should be wasteful here. Measure the cost of maintaining an unused projection.                   | MEDIUM | 10m    | 10m  |
| M5.3 | **BenchmarkMaterializeVsReplay_PredictionAccuracy**: with WithWorkloadStats, compare ShouldMaterialize() recommendation vs actual measured crossover. Does the formula match reality? | HIGH   | 10m    | 10m  |

### Workstream M6: Cross-Engine Correctness at Scale

| #    | Task                                                                                                                                                                                                                                       | Impact | Effort | Time |
| ---- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ | ------ | ------ | ---- |
| M6.1 | **BenchmarkCrossEngine_ParityAtScale**: seed 10K events through Memory store, execute all 6 queries, record results. Seed same events through SQLite store, execute same queries, verify identical results. Measure parity check overhead. | HIGH   | 12m    | 12m  |
| M6.2 | **BenchmarkCrossEngine_SwapUnderLoad**: apply events to Memory store, swap to SQLite mid-stream, verify results consistent. Measure swap latency.                                                                                          | MEDIUM | 10m    | 10m  |

---

## Priority-Sorted Execution Order

| Priority | #    | Task                                                               | Impact   | Time |
| -------- | ---- | ------------------------------------------------------------------ | -------- | ---- |
| 1        | M1.1 | Define e-commerce domain events                                    | CRITICAL | 10m  |
| 2        | M1.2 | Declare 6 queries spanning 4 ADTs                                  | CRITICAL | 12m  |
| 3        | M1.3 | Build event generator (mixed types, deterministic)                 | HIGH     | 10m  |
| 4        | M1.4 | Multi-engine factory helper                                        | HIGH     | 8m   |
| 5        | M2.1 | Event fan-out benchmark (1 event → 6 projections)                  | CRITICAL | 12m  |
| 6        | M2.2 | Read mix benchmark (6 query types round-robin)                     | CRITICAL | 12m  |
| 7        | M2.4 | Engine pool comparison (memory vs mem+sqlite vs mem+sqlite+pebble) | CRITICAL | 12m  |
| 8        | M3.1 | Write amplification scaling (1→10 projections)                     | HIGH     | 10m  |
| 9        | M4.1 | Layout planning: unplanned vs planned (10K rows)                   | HIGH     | 12m  |
| 10       | M5.1 | Materialize-vs-replay: read-heavy (100K events)                    | HIGH     | 12m  |
| 11       | M2.5 | Planner Plan() latency (1-20 queries × 1-5 engines)                | HIGH     | 10m  |
| 12       | M2.6 | Cost model accuracy (predicted vs actual latency)                  | HIGH     | 12m  |
| 13       | M2.3 | Mixed workload (concurrent writers + readers)                      | HIGH     | 10m  |
| 14       | M3.2 | Event storm (8 goroutines × 1000 events × 6 projections)           | HIGH     | 10m  |
| 15       | M6.1 | Cross-engine parity at scale (10K events)                          | HIGH     | 12m  |
| 16       | M5.3 | Materialize-vs-replay prediction accuracy                          | HIGH     | 10m  |
| 17       | M4.3 | Layout plan build time                                             | MEDIUM   | 8m   |
| 18       | M3.3 | Write amplification budget enforcement                             | MEDIUM   | 8m   |
| 19       | M5.2 | Materialize-vs-replay: write-heavy                                 | MEDIUM   | 10m  |
| 20       | M4.2 | Columnar extraction (DuckDB, skip if no CGo)                       | MEDIUM   | 10m  |
| 21       | M6.2 | Cross-engine swap under load                                       | MEDIUM   | 10m  |
| 22       | M2.7 | Verify all benchmarks skip cleanly when engines unavailable        | MEDIUM   | 8m   |
| 23       | M2.8 | Run nix fmt + build + vet on all new files                         | HIGH     | 8m   |
| 24       | M2.9 | Write final status report with results table                       | —        | 10m  |

**Total: ~250 min (~4.2 hours)**

---

## File Plan

### New Files

| File                                   | Content                                                |
| -------------------------------------- | ------------------------------------------------------ |
| `metaengine/bench_promise_test.go`     | M1.1-M1.4: domain model, queries, generator, factory   |
| `metaengine/bench_fanout_test.go`      | M2.1: event fan-out, M3.1: write amplification scaling |
| `metaengine/bench_readmix_test.go`     | M2.2: read mix, M2.3: mixed workload                   |
| `metaengine/bench_enginepool_test.go`  | M2.4: engine pool comparison                           |
| `metaengine/bench_planner_test.go`     | M2.5: Plan() latency, M2.6: cost model accuracy        |
| `metaengine/bench_storm_test.go`       | M3.2: event storm, M3.3: budget enforcement            |
| `metaengine/bench_layout_test.go`      | M4.1-M4.3: layout planning payoff                      |
| `metaengine/bench_materialize_test.go` | M5.1-M5.3: materialize-vs-replay                       |
| `metaengine/bench_parity_test.go`      | M6.1-M6.2: cross-engine correctness at scale           |

### Files NOT Modified

The plan creates only new `*_test.go` files. Zero production code changes. Zero API changes.
