# Comprehensive Status Report — go-cqrs-lite

> **Date:** 2026-06-03 11:41  
> **Session:** 144 (Performance Optimization + Housekeeping)  
> **Branch:** master  
> **Commits Ahead of Origin:** 0 (all pushed)

---

## Executive Summary

| Metric                 | Value                                                |
| ---------------------- | ---------------------------------------------------- |
| **Total Modules**      | 30 (22 library + 6 examples + 1 cmd + 1 integration) |
| **Source Files**       | 259 `.go` files                                      |
| **Test Files**         | 237 `_test.go` files                                 |
| **Benchmark Files**    | 24 `benchmark_test.go` / `bench_test.go`             |
| **Overall Coverage**   | **83.5%**                                            |
| **Packages with 100%** | decider, dispatcher, catalog/caseutil                |
| **Lowest Coverage**    | turso (28.6%), eventtest (18.4%)                     |
| **Race Detector**      | ✅ Zero races across all 32 packages                 |
| **Build Status**       | ✅ Clean (all packages compile)                      |

---

## a) FULLY DONE — This Session

### Performance Optimizations (3 major)

| #   | Change                                                                                                              | Files                                                                | Impact                                                              |
| --- | ------------------------------------------------------------------------------------------------------------------- | -------------------------------------------------------------------- | ------------------------------------------------------------------- |
| 1   | **MemoryStore global log** — replaced O(n log n) `collectAllSorted()` with append-only `globalLog` + `eventIDIndex` | `memory/store.go`, `memory/store_load.go`                            | ReadAll: 98μs→3.3μs (-96%), 12→1 alloc. ReadFrom: O(1) index lookup |
| 2   | **Listing projection cache** — `InMemoryAggregateReader` caches sorted `[]AggregateStatus`, rebuilds on first call  | `listing/in_memory.go`                                               | List: 840ms→33ms (-96%), 809MB→165MB, 1M→10K allocs                 |
| 3   | **ImmutableEvent slim-down** — moved `clock`/`newCodec`/`deadline` (48B) to `*eventOptions` pointer                 | `event/event.go`, `event_construct.go`, `event_new.go`, `options.go` | 336B→304B per event (-10%)                                          |

### Benchmarks Added (4 modules, 10 benchmarks)

| Module       | Benchmarks                                                      | Baselines               |
| ------------ | --------------------------------------------------------------- | ----------------------- |
| `codec/`     | JSON Encode/Decode, Raw Encode/Decode                           | 249ns/519ns/14ns/23ns   |
| `watermill/` | eventToMessage, messageToEvent, Publish, buildMetadata          | 442ns/528ns/593ns/122ns |
| `turso/`     | Save, Load (100 events)                                         | 83μs/1.08ms             |
| `memory/`    | ReadAll Scale (100→100K), Save Concurrent, ReadWrite Concurrent | Linear O(n) confirmed   |

### Housekeeping (6 items)

| #   | Task                                                      | Status  |
| --- | --------------------------------------------------------- | ------- |
| 1   | ADR numbering fix: 0007→0009                              | ✅ Done |
| 2   | `integration/go.mod` tidy (codec/v2 + snapshot/v2 direct) | ✅ Done |
| 3   | Remove deprecated `otel.TraceIDLogger`                    | ✅ Done |
| 4   | Remove deprecated `query.ErrQueryNotSupported`            | ✅ Done |
| 5   | Rewrite `CONTRIBUTING.md` for actual project              | ✅ Done |
| 6   | Update `docs/benchmarks/README.md` with all results       | ✅ Done |

### Tests Added (1 module, 14 tests)

| Module             | Tests                                                                  | Coverage Before | Coverage After |
| ------------------ | ---------------------------------------------------------------------- | --------------- | -------------- |
| `event/eventtest/` | 14 tests (Save, Load, ReadAll, ReadFrom, Close, overrides, concurrent) | 0.0% (no tests) | 18.4%          |

### Documentation (3 files)

| File                                                         | What                                        |
| ------------------------------------------------------------ | ------------------------------------------- |
| `docs/planning/2026-06-03_02-25_PERFORMANCE-OPTIMIZATION.md` | Pareto analysis plan (8 tasks, 52 subtasks) |
| `docs/planning/2026-06-03_02-25_EXECUTION-PLAN.md`           | Session continuation plan (25 tasks)        |
| `docs/benchmarks/README.md`                                  | Updated with T1-T4 results + new baselines  |

---

## b) PARTIALLY DONE

| Item                   | What's Done                   | What's Missing                              | Why Blocked                              |
| ---------------------- | ----------------------------- | ------------------------------------------- | ---------------------------------------- |
| **Pebble Journal**     | `Store` interface implemented | `Journal`/`SeekableJournal` not implemented | Needs design decision on iteration order |
| **cqrs-gen tests**     | Core logic tested (89.9%)     | CLI entry point untested                    | Requires mock filesystem                 |
| **Turso coverage**     | Basic connector tests (28.6%) | SyncDB Push/Pull/Checkpoint untested        | Requires remote Turso server             |
| **api-stability tool** | Tool exists, works            | Zero tests                                  | Low priority — tool is self-validating   |
| **eventtest coverage** | FakeStore tested (18.4%)      | VersionQueryFn, other helpers untested      | Time constraint                          |

---

## c) NOT STARTED

### From Original Plan (Phase C + D)

| #   | Task                                                                                       | Impact                        | Effort |
| --- | ------------------------------------------------------------------------------------------ | ----------------------------- | ------ |
| C1  | MemoryStore deduplication: store events ONLY in globalLog, use index for per-stream lookup | **2× memory reduction**       | 20min  |
| C2  | Listing cache auto-invalidation via atomic counter                                         | Correctness + performance     | 10min  |
| C3  | `findCodecOption` elimination — cache default codec                                        | 1 alloc per `New()`           | 10min  |
| C4  | `sync.Pool` for `ImmutableEvent` construction                                              | Reduce GC pressure            | 15min  |
| C5  | api-stability tests                                                                        | Untested guard tool           | 10min  |
| C6  | `Option` as interface (v3)                                                                 | Eliminates func closure alloc | 20min  |
| D1  | Reactive `AggregateReader` (subscribe to bus)                                              | Real-time listing             | 15min  |
| D2  | Compact `Metadata` representation                                                          | Reduce 152B per event         | 20min  |
| D3  | Evaluate faster JSON codec (`goccy/go-json`)                                               | Potential 2-3× JSON speedup   | 15min  |

### From TODO_LIST.md (Still Open)

| #   | Item                                                  | Source              |
| --- | ----------------------------------------------------- | ------------------- |
| 1   | `pebble/config.go` backward-compat aliases (20 lines) | Session 140 review  |
| 2   | `query.TypedHandler[T]` takes `Query` not `T`         | Session 140 review  |
| 3   | `ROADMAP.md` creation                                 | Documentation gap   |
| 4   | `CHANGELOG.md` update                                 | 2 days behind       |
| 5   | Outbox Pattern design doc                             | FEATURES.md planned |
| 6   | Schema Registry design doc                            | FEATURES.md planned |

---

## d) TOTALLY FUCKED UP

| #   | Issue                                | Severity | Root Cause                                                                                                     |
| --- | ------------------------------------ | -------- | -------------------------------------------------------------------------------------------------------------- |
| 1   | **BuildFlow pre-commit hook broken** | HIGH     | `build_mode: full` runs heavy checks, missing `scripts`/`docs` excludes, no actual git hook plumbing installed |
| 2   | **ADR 0005 missing**                 | LOW      | Gap in sequence — skipped during creation                                                                      |
| 3   | **turso coverage at 28.6%**          | MEDIUM   | SyncDB methods (Push/Pull/Checkpoint) untested — require remote server                                         |

**Note:** Item #1 was intentionally deferred per user request. All other items are acceptable technical debt.

---

## e) WHAT WE SHOULD IMPROVE

### Immediate (This Week)

1. **MemoryStore deduplication (C1)** — Events stored in both `events` map AND `globalLog`. That's 2× memory for every event. The `events` map is only needed for per-stream `Load()`. We could derive stream events from `globalLog` via a `(type, id) → []int` index. **2× memory reduction**.

2. **Listing cache auto-invalidation (C2)** — Current cache never invalidates automatically. If someone calls `InvalidateCache()` manually it's fine, but a forgotten call = stale data. An atomic counter incremented on every `Save()` / `AppendBatch()` would let the reader detect changes and rebuild.

3. **`findCodecOption` probe (C3)** — Still allocates `&ImmutableEvent{}` on every `New()` call. The probe is only needed when `WithNewCodec` is used (rare). Caching the default codec and only probing when options are non-empty would eliminate this alloc for 99% of calls.

4. **Update CHANGELOG.md** — 2 days behind, missing 8 commits of work.

### Short-Term (This Month)

5. **Pebble Journal implementation** — Only implements `Store`. `Journal`/`SeekableJournal` would unlock projection replay on Pebble backend.

6. **`Option` as interface (v3)** — Current `Option func(*ImmutableEvent)` requires heap allocation for every option. An interface-based design would allow zero-allocation options for common cases (WithEncoding, WithSchemaVersion).

7. **Reactive `AggregateReader`** — Subscribe to event bus, auto-update cache. Eliminates need for `InvalidateCache()` entirely.

8. **Compact `Metadata`** — 152 bytes per event is heavy. `CorrelationID`/`CausationID`/`UserID`/`RequestID` are all 24B branded IDs. Most events don't set them. A pointer-based or bitfield approach could halve this.

9. **Faster JSON codec evaluation** — `goccy/go-json` or `bytedance/sonic` could give 2-3× speedup on JSON encode/decode. Worth benchmarking.

10. **`sync.Pool` for hot paths** — `ImmutableEvent` construction, `[]byte` buffers for payload, `Metadata` structs. All could benefit from object pooling.

### Medium-Term (Next Quarter)

11. **Outbox Pattern implementation** — Documented in CONTEXT.md but no code exists. Critical for reliable at-least-once publishing.

12. **Schema Registry** — JSON Schema middleware for event validation. Needs design decisions on versioning.

13. **PostgreSQL integration tests** — Currently only SQLite in-memory tests. Testcontainers would validate real PostgreSQL behavior.

14. **Distributed consensus** — Raft/CRDT overlay for multi-node event stores.

15. **Documentation site** — Docusaurus/MkDocs for consumer-facing docs.

---

## f) Top #25 Things To Get Done Next

Sorted by **impact ÷ effort × urgency**:

| Rank | Task                                            | Phase | Est   | Impact       |
| ---- | ----------------------------------------------- | ----- | ----- | ------------ |
| 1    | MemoryStore deduplication (2× memory reduction) | C1    | 20min | **CRITICAL** |
| 2    | Listing cache auto-invalidation                 | C2    | 10min | HIGH         |
| 3    | `findCodecOption` elimination                   | C3    | 10min | MEDIUM       |
| 4    | Update CHANGELOG.md                             | A10   | 10min | LOW          |
| 5    | Pebble Journal implementation                   | —     | 30min | HIGH         |
| 6    | `sync.Pool` for ImmutableEvent                  | C4    | 15min | MEDIUM       |
| 7    | api-stability tests                             | C5    | 10min | LOW          |
| 8    | `Option` as interface (v3 design)               | C6    | 20min | MEDIUM       |
| 9    | Compact Metadata representation                 | D2    | 20min | MEDIUM       |
| 10   | Faster JSON codec evaluation                    | D3    | 15min | MEDIUM       |
| 11   | Reactive AggregateReader                        | D1    | 15min | MEDIUM       |
| 12   | pebble/config.go alias cleanup                  | —     | 5min  | LOW          |
| 13   | query.TypedHandler[T] signature fix             | —     | 10min | LOW          |
| 14   | ROADMAP.md creation                             | —     | 15min | LOW          |
| 15   | Outbox Pattern design doc                       | —     | 20min | HIGH         |
| 16   | Schema Registry design doc                      | —     | 20min | MEDIUM       |
| 17   | PostgreSQL integration tests                    | —     | 60min | HIGH         |
| 18   | BuildFlow pre-commit hook fix                   | —     | 15min | LOW          |
| 19   | cqrs-gen CLI test coverage                      | —     | 20min | LOW          |
| 20   | eventtest remaining helpers                     | —     | 15min | LOW          |
| 21   | Fuzz test expansion                             | —     | 30min | MEDIUM       |
| 22   | `-race` CI check addition                       | B2    | 5min  | MEDIUM       |
| 23   | benchstat baseline establishment                | B5    | 10min | LOW          |
| 24   | GC pressure analysis                            | —     | 20min | LOW          |
| 25   | pprof profiling harness                         | —     | 30min | LOW          |

---

## g) Top #1 Question I Cannot Figure Out Myself

### Should we eliminate the `events` map in MemoryStore entirely?

**Context:** After T1, MemoryStore stores every event in TWO places:

1. `events map[string][]event.Event` — per-stream, used by `Load()`, `LoadFromVersion()`, etc.
2. `globalLog []event.Event` — append-only, used by `ReadAll()`, `ReadFrom()`

This is **2× memory** for every event. For 100K events, that's significant.

**Option A: Keep both** (current)

- Pros: Simple, fast `Load()` (O(1) map lookup + slice clone)
- Cons: 2× memory

**Option B: Derive per-stream from globalLog**

- Add `streamIndex map[string][]int` (stream key → indices in globalLog)
- `Load()` looks up indices, clones events from globalLog
- Pros: 1× memory, single source of truth
- Cons: More complex, `Load()` becomes O(n) in stream length (still linear, but with indirection)

**Option C: Store only in globalLog, no per-stream index**

- `Load()` scans globalLog for matching (type, id)
- Pros: Absolute minimum memory
- Cons: `Load()` becomes O(N) in total events — terrible for large stores

**My recommendation: Option B** — the memory savings justify the complexity. But I want confirmation because it changes the internal architecture of MemoryStore significantly.

**What do you think?**

---

## Appendix: Coverage by Module

| Module           | Coverage | Notes                         |
| ---------------- | -------- | ----------------------------- |
| decider          | 100.0%   | Perfect                       |
| dispatcher       | 100.0%   | Perfect                       |
| catalog/caseutil | 100.0%   | Perfect                       |
| middleware       | 98.5%    | Excellent                     |
| memory           | 99.0%    | Excellent                     |
| command          | 93.8%    | Excellent                     |
| query            | 95.5%    | Excellent                     |
| signing          | 94.1%    | Excellent                     |
| catalog          | 95.9%    | Excellent                     |
| id               | 94.5%    | Excellent                     |
| codec            | 93.3%    | Excellent                     |
| catalog/openapi  | 96.4%    | Excellent                     |
| event            | 89.4%    | Good                          |
| schema           | 89.7%    | Good                          |
| storage          | 89.3%    | Good                          |
| listing          | 91.5%    | Good                          |
| projection       | 90.5%    | Good                          |
| watermill        | 92.6%    | Good                          |
| pebble           | 88.1%    | Good                          |
| snapshot         | 92.3%    | Good                          |
| cmd/cqrs-gen     | 89.9%    | Good                          |
| turso            | 28.6%    | **Poor** — needs SyncDB tests |
| eventtest        | 18.4%    | Acceptable — test helpers     |
| integration/\*   | N/A      | Benchmark-only                |

---

## Appendix: Commit Log (This Session)

```
0dd360ec docs(planning): add execution plan for session continuation
ffef961e test(eventtest): fix FakeStore ReadFrom test for sorted ReadAll
e29ae370 test(eventtest): add comprehensive FakeStore tests
dd283ea7 docs(benchmarks): update README with T1-T4 results + new module baselines
1979e076 docs: rewrite CONTRIBUTING.md to match actual project
69a220cc chore: remove deprecated APIs, fix ADR numbering, tidy integration go.mod
c2352042 test(memory): add scaling curve and concurrent stress benchmarks
f2b03f81 test: add benchmarks for codec, watermill, and turso modules
31a8c31b perf(event): move clock/newCodec/deadline to eventOptions pointer — 48B saved
84ac73ee perf(listing): cache sorted aggregate index — 25x faster listing
b688bc57 perf(memory): replace O(n log n) collectAllSorted with append-only global log
212ffdcb docs(planning): add performance optimization plan with Pareto analysis
```

---

_Report generated by Crush on 2026-06-03 at 11:41._
