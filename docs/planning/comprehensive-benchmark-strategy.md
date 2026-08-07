# Comprehensive Benchmark Strategy: Full-Pipeline Matrix

**Date:** 2026-08-07
**Status:** PLANNING — awaiting "BUILD" approval

---

## 1. What We Already Do Well

### Infrastructure (strong)

| Component | Coverage | Quality |
|-----------|----------|---------|
| **benchkit** — 14-phase pipeline runner | write, batch-write, read, versioned-read, read-model, projection, checkpoint, mixed-workload, journey, query, snapshot, metaengine, raw-sink, durability, recovery | Mature: latency percentiles, throughput, GC, memory, disk footprint, JSON serialization, comparison, soak tests |
| **8 named profiles** | dev (500), small (10K), medium (500K), large (10M), stress (5M), write-heavy, read-heavy, analytical | Good: covers tiny→production scale, write/read mix variants |
| **9 backends in stack/bench** | Memory, SQLite, SQLite-CGo, Pebble, BBolt, DuckDB, Turso, Postgres, MySQL | Good: all major backends have suite benchmarks |
| **cmd/cqrs-bench CLI** | run, compare, sweep subcommands | Mature: text/json/markdown/benchstat output, worker sweep |
| **bench-all.sh** | auto-discovers all 41 modules with benchmarks | Good: runs everything, exit-code FAIL detection |
| **CI** | Quick (push), Go-bench (push), Nightly (memory+SQLite+Pebble) | Partial: only 3 of 9 backends in nightly |

### Per-module microbenchmarks (~230+ functions, ~72 files)

| Module | What's measured |
|--------|----------------|
| event | NewEvent, payload decode, bus publish, clone, tombstone detection |
| codec | JSON/CBOR/Raw encode/decode, transcode, buffer encoder, compact vs canonical |
| command | New, dispatch, metadata, typed store |
| query | dispatch, pagination, typed handler |
| decider | Execute, cache (hot/cold), singleflight coalescing |
| storage | SQL save/load, memory save/load/concurrent, pebble, turso |
| snapshot | EveryNEvents, ReadPressure, load/store |
| signing | HMAC sign/verify, Ed25519 sign/verify |
| encryption | AES-256-GCM, XChaCha20-Poly1305, codec wrapper |
| middleware | logging, recovery, validation, retry, circuit breaker, OTel bundle |
| kv | MemStore set/get/has/delete/batch/iterator |
| dedup | ring buffer add/evict/has/has-miss |
| catalog | registry build, schema export, AsyncAPI/EventCatalog/OpenAPI |
| metaengine | filter scan, layout planning, calibration per engine (Pebble, DuckDB, PG, Dgraph, Badger) |
| integration | scale (10K×100), realistic (Order domain), concurrent, signing, snapshot, listing, query |

---

## 2. What We Can Improve

| Area | Current State | Improvement Needed |
|------|---------------|-------------------|
| **Profile in suite benchmarks** | All use ProfileDev (500 events) | Add ProfileSmall (10K) variant for more realistic numbers |
| **Payload size in suite benchmarks** | All use single 128B | Add mixed-size distribution (64B, 256B, 1KB, 10KB) |
| **Middleware isolation** | Each middleware measured alone | Chain 3-5 middleware together (production config) |
| **Nightly CI backend coverage** | Only Memory, SQLite, Pebble | Add BBolt, DuckDB, Turso, Postgres |
| **Regression detection** | Script exists, not wired to CI | Wire benchstat into CI per-PR gate |
| **Codec in pipeline** | Codec benchmarks are isolated | Measure JSON vs CBOR in a full write+read cycle |
| **Concurrency sweep** | Hardcoded per benchmark | Systematic sweep: 1, 2, 4, 8, 16, 32 workers |

---

## 3. What We're NOT Doing (Critical Gaps)

### Gap 1: No Full-Pipeline Benchmark (THE #1 GAP)
**The complete path** — command → decider.Execute → decide → event.NewEvent → EventSink.Save → EventBus.Publish → projection.Handle → kv.Set → query.DispatchTyped — is **never measured as one benchmark**.

Two halves exist separately:
- `BenchmarkCommandPath_Memory`: command → decider → save → publish (STOPS HERE)
- benchkit `journeyPhase`: save → projection → query (STARTS HERE, skips command/decider)

**Impact:** Consumers cannot answer "what's the end-to-end latency of my CQRS system?"

### Gap 2: No Realistic Multi-Domain Data
Only **one** realistic domain type exists (Order in `integration/`). Real projects have User, Task, Cart, Invoice, Notification coexisting in the same event store. The benchkit `BenchPayload` is synthetic padding.

### Gap 3: No Mixed-Size Distribution in Actual Benchmarks
`NewMixedGenerator` exists in benchkit but **zero suite benchmarks use it**. All use single 128B. Real events range from 64B status changes to 10KB embedded collections.

### Gap 4: No Middleware Chain Benchmarks
Each middleware benchmarked in isolation. Production runs **logging + recovery + retry + OTel tracing + OTel metrics** simultaneously. The compounded overhead is unknown.

### Gap 5: No Multi-Backend Comparison Table
Each backend benchmarked separately. No comparison runs the **same profile, same codec, same payload** across all backends and produces a single comparison table. (cqrs-bench `compare` exists but isn't used in benchmarks.)

### Gap 6: Zero Benchmarks in Critical Modules

| Module | Status | What's Missing |
|--------|--------|---------------|
| `projectionhost/` | **ZERO** `func Benchmark` | Replay speed, DLQ throughput, multi-projection parallel |
| `transport/grpc/` | **ZERO** `func Benchmark` | Remote dispatch overhead |
| `scheduling/` | **ZERO** `func Benchmark` | Timer schedule/poll/dispatch throughput |
| `scenario/` | **ZERO** `func Benchmark` | BDD DSL overhead |

### Gap 7: No Configuration Matrix
No benchmark exercises the combination matrix: backend × codec × middleware × payload-size × concurrency × durability-tier. Each axis varies independently but they're never combined.

### Gap 8: No Durability Tier Comparison
Strict vs Normal vs Relaxed exists as a feature but is **never benchmarked**. Consumers can't see the fsync cost tradeoff.

### Gap 9: No Batch Size Sweep
BatchSize 1 vs 5 vs 10 vs 50 — how does write batch size affect throughput? Untested as a sweep.

### Gap 10: No Snapshot Strategy Comparison
EveryN(10) vs EveryN(50) vs ReadPressure(50) vs None — never compared in one benchmark.

### Gap 11: No Contention on Persistent Backends
Same-stream contention only measured on Memory. SQLite/Postgres/BBolt serialization behavior under contention is unknown.

---

## 4. The Comprehensive Plan

### Workstream A: Realistic Full-Pipeline Benchmark (THE CORE DELIVERABLE)

The benchmark that tests "EVERYTHING" — exercises the complete CQRS pipeline with realistic data across all configurations.

| # | Task | Impact | Effort | Customer Value |
|---|------|--------|--------|----------------|
| A1 | Define 3 multi-domain model types (Order, Task, User) with create/update/delete events + state + fold + decide | CRITICAL | 10m | HIGH — realistic data shapes |
| A2 | Build realistic workload generator: mixed domain types + mixed payload sizes (64B/256B/1KB/10KB distribution) | CRITICAL | 12m | HIGH — real-world data |
| A3 | Implement full-pipeline benchmark: command→decider→save→publish→projection→readmodel→query (memory backend, single-goroutine) | CRITICAL | 12m | CRITICAL — answers "how fast is my system?" |
| A4 | Add concurrent full-pipeline benchmark: N goroutines running the complete pipeline, report throughput + latency | HIGH | 10m | HIGH — real production load |
| A5 | Parameterize pipeline by backend factory: memory, sqlite, sqlite-cgo, pebble, bbolt sub-benchmarks | HIGH | 10m | HIGH — backend comparison |
| A6 | Add codec variant sub-benchmarks: pipeline/json, pipeline/cbor | HIGH | 8m | HIGH — codec decision support |
| A7 | Add middleware stack variant sub-benchmarks: none, logging-only, tracing-only, full-stack (logging+recovery+retry+OTel) | HIGH | 10m | HIGH — production overhead visibility |
| A8 | Add payload size variant sub-benchmarks: 128B, 1KB, 10KB, mixed-distribution | MEDIUM | 8m | MEDIUM — sizing guidance |
| A9 | Add concurrency variant sub-benchmarks: workers=1, 4, 8, 16 | MEDIUM | 8m | MEDIUM — scaling guidance |

### Workstream B: Configuration Matrix Runner

| # | Task | Impact | Effort | Customer Value |
|---|------|--------|--------|----------------|
| B1 | Create `scripts/bench-matrix.sh`: runs full-pipeline benchmark across all valid backend×codec×middleware×size combos, outputs JSON | HIGH | 12m | HIGH — one-command comprehensive comparison |
| B2 | Add `--matrix` flag to bench-all.sh that invokes the matrix runner | MEDIUM | 8m | MEDIUM — discoverability |

### Workstream C: Gap-Filling Benchmarks

| # | Task | Impact | Effort | Customer Value |
|---|------|--------|--------|----------------|
| C1 | projectionhost replay benchmark: seed N events, measure projection catch-up throughput (events/sec) | HIGH | 12m | HIGH — projection replay is a critical operation |
| C2 | projectionhost DLQ throughput benchmark: insert/list/delete entries at scale | MEDIUM | 10m | MEDIUM — DLQ is production infrastructure |
| C3 | projectionhost multi-projection parallel benchmark: 3 projections, measure combined throughput | MEDIUM | 10m | MEDIUM — multi-projection is common |
| C4 | gRPC transport benchmark: local server+client, measure command+query remote dispatch overhead | HIGH | 12m | HIGH — distributed systems decision |
| C5 | SSE real-fanout benchmark: HTTP server + N real SSE client connections, measure fan-out latency | MEDIUM | 12m | MEDIUM — real-time delivery decision |
| C6 | Contention benchmark on persistent backends: same-stream contention on SQLite, Pebble, BBolt | HIGH | 10m | HIGH — contention ceiling on real storage |
| C7 | Durability tier comparison benchmark: SQLite Strict vs Normal vs Relaxed | HIGH | 10m | HIGH — fsync cost tradeoff visibility |
| C8 | Snapshot strategy comparison benchmark: None vs EveryN(10) vs EveryN(50) vs ReadPressure(50) | MEDIUM | 10m | MEDIUM — snapshot tuning guidance |
| C9 | Batch size sweep benchmark: BatchSize 1, 5, 10, 50 on SQLite + Pebble | MEDIUM | 8m | MEDIUM — write tuning guidance |
| C10 | Codec pipeline comparison: full write+read+decode cycle with JSON vs CBOR vs CBOR-Compact | HIGH | 10m | HIGH — codec decision with real I/O |
| C11 | Idempotency middleware overhead benchmark: with vs without on command path | MEDIUM | 8m | MEDIUM — dedup cost visibility |
| C12 | Scheduling/timer benchmark: schedule + poll + dispatch + mark fired throughput | LOW | 8m | LOW — niche feature |
| C13 | Add ProfileSmall (10K events) variants to existing stack/bench suite benchmarks | MEDIUM | 8m | MEDIUM — more realistic than 500-event ProfileDev |

### Workstream D: CI & Infrastructure

| # | Task | Impact | Effort | Customer Value |
|---|------|--------|--------|----------------|
| D1 | Wire benchstat regression detection into CI: per-PR benchmark comparison against baseline | HIGH | 10m | HIGH — catch performance regressions automatically |
| D2 | Expand nightly CI to all embedded backends: add BBolt, DuckDB, Turso | MEDIUM | 10m | MEDIUM — broader nightly coverage |
| D3 | Add matrix benchmark job to nightly CI (invokes bench-matrix.sh) | MEDIUM | 8m | MEDIUM — comprehensive nightly comparison |
| D4 | Update benchmark baseline after all new benchmarks are added | MEDIUM | 8m | MEDIUM — regression detection foundation |

---

## 5. Priority-Sorted Execution Order (ALL 30 Tasks)

Sorted by: Impact DESC, Customer Value DESC, Effort ASC

| Priority | # | Task | Impact | Effort | Est. Time |
|----------|---|------|--------|--------|-----------|
| 1 | A1 | Define 3 multi-domain model types (Order, Task, User) | CRITICAL | 10m | 10m |
| 2 | A2 | Build realistic workload generator (mixed domains + mixed sizes) | CRITICAL | 12m | 12m |
| 3 | A3 | Implement full-pipeline benchmark (command→query, memory) | CRITICAL | 12m | 12m |
| 4 | C6 | Contention benchmark on persistent backends | HIGH | 10m | 10m |
| 5 | C10 | Codec pipeline comparison (JSON vs CBOR full cycle) | HIGH | 10m | 10m |
| 6 | A4 | Concurrent full-pipeline benchmark | HIGH | 10m | 10m |
| 7 | A5 | Parameterize pipeline by backend | HIGH | 10m | 10m |
| 8 | C7 | Durability tier comparison (Strict vs Normal vs Relaxed) | HIGH | 10m | 10m |
| 9 | C1 | projectionhost replay benchmark | HIGH | 12m | 12m |
| 10 | C4 | gRPC transport dispatch benchmark | HIGH | 12m | 12m |
| 11 | A6 | Codec variant sub-benchmarks (pipeline/json, pipeline/cbor) | HIGH | 8m | 8m |
| 12 | A7 | Middleware stack variant sub-benchmarks | HIGH | 10m | 10m |
| 13 | D1 | Wire benchstat regression into CI | HIGH | 10m | 10m |
| 14 | A8 | Payload size variant sub-benchmarks | MEDIUM | 8m | 8m |
| 15 | A9 | Concurrency variant sub-benchmarks | MEDIUM | 8m | 8m |
| 16 | B1 | Create bench-matrix.sh configuration matrix runner | HIGH | 12m | 12m |
| 17 | C8 | Snapshot strategy comparison benchmark | MEDIUM | 10m | 10m |
| 18 | C9 | Batch size sweep benchmark | MEDIUM | 8m | 8m |
| 19 | C2 | projectionhost DLQ throughput benchmark | MEDIUM | 10m | 10m |
| 20 | C3 | projectionhost multi-projection parallel benchmark | MEDIUM | 10m | 10m |
| 21 | C13 | Add ProfileSmall variants to existing suite benchmarks | MEDIUM | 8m | 8m |
| 22 | D2 | Expand nightly CI to all embedded backends | MEDIUM | 10m | 10m |
| 23 | C5 | SSE real-fanout benchmark | MEDIUM | 12m | 12m |
| 24 | C11 | Idempotency middleware overhead benchmark | MEDIUM | 8m | 8m |
| 25 | B2 | Add --matrix flag to bench-all.sh | MEDIUM | 8m | 8m |
| 26 | D3 | Add matrix benchmark job to nightly CI | MEDIUM | 8m | 8m |
| 27 | C12 | Scheduling/timer benchmark | LOW | 8m | 8m |
| 28 | D4 | Update benchmark baseline | MEDIUM | 8m | 8m |
| 29 | A2b | Verify all new benchmarks skip cleanly when deps unavailable | MEDIUM | 8m | 8m |
| 30 | D5 | Write final comprehensive status report | — | 10m | 10m |

**Total estimated time: ~280 min (~4.7 hours)**

---

## 6. File Plan

### New Files to Create

| File | Workstream | Content |
|------|-----------|---------|
| `stack/bench/realistic_models_test.go` | A1 | 3 domain types: Order, Task, User (events, state, fold, decide) |
| `stack/bench/realistic_generator_test.go` | A2 | Mixed-domain + mixed-size workload generator |
| `stack/bench/full_pipeline_test.go` | A3-A9 | Full-pipeline benchmark with all variant sub-benchmarks |
| `stack/bench/contention_persistent_test.go` | C6 | Same-stream contention on SQLite/Pebble/BBolt |
| `stack/bench/durability_tiers_test.go` | C7 | Strict vs Normal vs Relaxed |
| `stack/bench/codec_pipeline_test.go` | C10 | JSON vs CBOR full write+read cycle |
| `stack/bench/batch_size_sweep_test.go` | C9 | BatchSize 1/5/10/50 sweep |
| `projectionhost/replay_bench_test.go` | C1-C3 | Replay speed + DLQ + multi-projection |
| `transport/grpc/dispatch_bench_test.go` | C4 | Remote command+query dispatch |
| `transport/http/sse_fanout_bench_test.go` | C5 | Real SSE broker fan-out |
| `decider/snapshot_strategy_bench_test.go` | C8 | Snapshot strategy comparison |
| `middleware/idempotency_pipeline_bench_test.go` | C11 | Idempotency overhead in pipeline |
| `scheduling/bench_test.go` | C12 | Timer throughput |
| `scripts/bench-matrix.sh` | B1 | Configuration matrix runner |

### Files to Modify

| File | Workstream | Change |
|------|-----------|--------|
| `scripts/bench-all.sh` | B2 | Add `--matrix` flag |
| `.github/workflows/benchmarks.yml` | D1-D3 | Wire regression, expand nightly, add matrix job |
| `stack/bench/benchkit_suite_test.go` | C13 | Add ProfileSmall variants |
| `benchmark-baseline.txt` | D4 | Regenerate with all new benchmarks |

---

## 7. Design Decisions

### Why 3 domain types (Order, Task, User)?
Real projects have bounded contexts. An e-commerce system has Orders, a SaaS has Tasks/Todos, every system has Users. Measuring all three together reveals:
- Cross-domain event serialization overhead
- Mixed payload size effects (Order=large, User=small)
- Multiple stream types in the same store

### Why mixed-size distribution (64B/256B/1KB/10KB)?
Real event streams have size variance:
- 64B: status change (`{"status":"active"}`)
- 256B: typical domain event (OrderCreated)
- 1KB: event with embedded metadata (UserRegistered)
- 10KB: event with embedded collection (CartUpdated with 50 items)

The `NewMixedGenerator` already supports this — we just need to use it.

### Why full-pipeline as ONE benchmark?
Consumers ask: "If I send a command, how long until I can query the result?" Only a full-pipeline benchmark answers this. The benchkit journey phase is close but skips the command/decider layer.

### Why sub-benchmarks (b.Run) for variants?
Go's `testing.B` sub-benchmarks produce structured output that benchstat can parse. Each variant is a separate `b.Run("backend=memory/codec=json/size=128B", ...)` case, enabling direct comparison.

### Why not extend benchkit instead of stack/bench?
benchkit's `BenchPayload` is locked to one type. Changing it would break the public API. The full-pipeline benchmark belongs in `stack/bench` where it can use custom domain types. benchkit remains the storage-level benchmark; the pipeline benchmark is the system-level benchmark.
