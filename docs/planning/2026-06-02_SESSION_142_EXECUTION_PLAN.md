# Execution Plan — Session 142+

**Created:** 2026-06-02 20:45
**Scope:** ALL open TODOs + performance improvements + quality items
**Rule:** Every task ≤ 12 minutes. Sorted by importance → impact → effort → customer-value.

---

## Sorting Criteria

| Rank | Factor | Weight |
|---|---|---|
| 1 | Customer-facing correctness & safety | Highest |
| 2 | Performance (measurable throughput improvement) | High |
| 3 | Test coverage & reliability | High |
| 4 | Developer experience & tooling | Medium |
| 5 | Code hygiene & consistency | Medium |
| 6 | Documentation & examples | Lower |
| 7 | Future/speculative | Lowest |

---

## PLAN — 67 Tasks

### P0: Hygiene Fixes (trivial, do first to clear the deck)

| # | Task | Module | Est. | Impact | Source |
|---|---|---|---|---|---|
| 1 | `go mod tidy` projection/ (fix x/sync missing) | projection | 1m | Fixes LSP error | Session 141 |
| 2 | `go mod tidy` example/projection/ | example | 1m | Fixes LSP error | Session 141 |
| 3 | `go mod tidy` example/saga-pattern/ | example | 1m | Fixes LSP error | Session 141 |
| 4 | `go mod tidy` turso/ (id/v2 + snapshot/v2 indirect → direct) | turso | 1m | Fixes lint warning | Session 141 |
| 5 | Fix `integration/go.mod` listing/v2 indirect → direct | integration | 1m | Fixes lint warning | Session 141 |
| 6 | Remove pebble backward-compat aliases (config.go:59-69) | pebble | 5m | Removes dead code | TODO line 416 |
| 7 | Fix LSP hints in scale_benchmark_test.go (atomic.Int64, forvar, min) | integration | 3m | Code quality | LSP |

### P1: Performance — Allocation Reduction (highest customer impact)

| # | Task | Module | Est. | Impact | Source |
|---|---|---|---|---|---|
| 8 | Design `ImmutableEvent.Reset()` method that zeros all 14 fields | event | 10m | Required for pool | Session 141 |
| 9 | Write test: assert all fields zeroed after Reset() | event | 8m | Safety gate | Session 141 |
| 10 | Add `sync.Pool` for `ImmutableEvent` with Reset() in NewEvent | event | 10m | 30-50% fewer allocs under load | Session 141 |
| 11 | Benchmark: re-run event/NewEvent with pool, compare before/after | event | 5m | Verify improvement | Session 141 |
| 12 | Eliminate `probeCodec` allocation — iterate opts directly for codec | event | 8m | 1 fewer alloc per event.New() | Session 141 |
| 13 | Fix `canonicalPayload` — use pre-sized `strings.Builder` | signing | 8m | 8 fewer allocs per sign/verify | Session 141 |
| 14 | Add pooled `[]byte` buffer for canonicalPayload alternative path | signing | 10m | Further reduce allocs under sustained signing | Session 141 |
| 15 | Add `WithNoCopy()` option for trusted payload paths (e.g., loaded from DB) | event | 10m | Eliminate defensive copy per event | Session 141 |
| 16 | Benchmark: measure alloc reduction from WithNoCopy on load path | event | 5m | Verify improvement | Session 141 |

### P2: Benchmark Coverage (missing hot paths)

| # | Task | Module | Est. | Impact | Source |
|---|---|---|---|---|---|
| 17 | Add benchmarks for `command.New()`, `MustNew()` | command | 8m | Every command request | Session 141 |
| 18 | Add benchmarks for `command.Dispatcher.Register()`, typed dispatch | command | 10m | Handler registration path | Session 141 |
| 19 | Add benchmarks for `query.New()`, `MustNew()`, `DispatchTyped()` | query | 8m | Every query request | Session 141 |
| 20 | Add benchmarks for `query.NewPaginatedResult()`, pagination | query | 8m | Pagination hot path | Session 141 |
| 21 | Add benchmarks for `schema.NewUpcaster()`, `VersionedStore.Load()` | schema | 10m | Every event load with schema evolution | Session 141 |
| 22 | Add benchmarks for `snapshot.EveryNEvents()`, `ShouldSnapshot()` | snapshot | 8m | Evaluated on every aggregate save | Session 141 |
| 23 | Add benchmarks for `memory.MemoryStore` Save/Load/ReadAll | memory | 10m | Baseline for all in-memory perf | Session 141 |
| 24 | Add benchmark for `memory.MemoryBus` Publish with subscribers | memory | 8m | Bus baseline | Session 141 |
| 25 | Add benchmark for `dispatcher.NewDispatcher()` + generic dispatch | dispatcher | 8m | Foundation for cmd/query/event | Session 141 |

### P3: Benchmark Quality (consistency & rigor)

| # | Task | Module | Est. | Impact | Source |
|---|---|---|---|---|---|
| 26 | Migrate `signing/` 5 benchmarks from `for i:=0; i<b.N` to `b.Loop()` | signing | 5m | Consistency | Session 141 |
| 27 | Add `b.ReportAllocs()` to event/ benchmarks (6 functions) | event | 3m | Self-documenting alloc tracking | Session 141 |
| 28 | Add `b.ReportAllocs()` to id/ benchmarks (6 functions) | id | 3m | Self-documenting alloc tracking | Session 141 |
| 29 | Add `b.ReportAllocs()` to decider/ benchmarks (4 functions) | decider | 3m | Self-documenting alloc tracking | Session 141 |
| 30 | Add `b.ReportAllocs()` to signing/ benchmarks (5 functions) | signing | 3m | Self-documenting alloc tracking | Session 141 |
| 31 | Add `b.ReportAllocs()` to listing/ benchmarks (4 functions) | listing | 3m | Self-documenting alloc tracking | Session 141 |
| 32 | Add `b.ReportAllocs()` to projection/ benchmarks (3 functions) | projection | 2m | Self-documenting alloc tracking | Session 141 |
| 33 | Add `b.ReportAllocs()` to catalog/ benchmarks (5 functions) | catalog | 3m | Self-documenting alloc tracking | Session 141 |
| 34 | Add `b.ReportAllocs()` to middleware/ benchmarks (4 functions) | middleware | 2m | Self-documenting alloc tracking | Session 141 |
| 35 | Add `b.ReportAllocs()` to storage/ benchmarks (3 functions) | storage | 2m | Self-documenting alloc tracking | Session 141 |
| 36 | Add `b.ReportAllocs()` to pebble/ benchmarks (2 functions) | pebble | 2m | Self-documenting alloc tracking | Session 141 |
| 37 | Add `b.ReportAllocs()` to integration benchmarks (6 functions) | integration | 3m | Self-documenting alloc tracking | Session 141 |
| 38 | Fix `BenchmarkSQLEventStore_Save` — debug and fix sqlmock expectations | storage | 10m | Fix failing benchmark | Session 141 |
| 39 | Evaluate: drop sqlmock benchmarks vs fix them (real vs mock) | storage | 5m | Honest numbers | Session 141 |

### P4: Benchmark Infrastructure

| # | Task | Module | Est. | Impact | Source |
|---|---|---|---|---|---|
| 40 | Write `scripts/benchstat-compare.sh` — run -count=10 + benchstat diff | scripts | 10m | Regression detection | Session 141 |
| 41 | Add `nix run .#bench` command to flake.nix (runs all benchmarks) | flake.nix | 10m | Developer ergonomics | Session 141 |
| 42 | Add CI job: benchstat comparison on PR (post threshold check) | .github | 12m | Regression gate | TODO line 262 |
| 43 | Benchmark storage backends: write SQLite vs Pebble comparison test | storage/pebble | 10m | Backend selection data | TODO line 225 |
| 44 | Run storage backend comparison, document results | docs | 8m | Decision support | TODO line 225 |

### P5: SIMD Optimization (single high-value target)

| # | Task | Module | Est. | Impact | Source |
|---|---|---|---|---|---|
| 45 | Prototype: SIMD timestamp scan in pebble/LoadToTimestamp using archsimd | pebble | 12m | 4-6× faster full scan | Session 141 |
| 46 | Benchmark: compare SIMD vs scalar pebble LoadToTimestamp | pebble | 5m | Verify improvement | Session 141 |
| 47 | Gate SIMD path behind `GOEXPERIMENT=simd` build tag + runtime CPU check | pebble | 10m | Portability safety | Session 141 |

### P6: Test Coverage Gaps

| # | Task | Module | Est. | Impact | Source |
|---|---|---|---|---|---|
| 48 | Add listing SQL reader tests (SQLAggregateReader) | listing | 10m | 0% → covered | TODO line 281 |
| 49 | Add BDD tests for `event.Version` (Parse, Add, Sub, Mod, Cmp, Increment) | event | 10m | Core type coverage | TODO line 273 |
| 50 | Add BDD tests for `event.SchemaVersion` (Parse, Increment, Decrement) | event | 8m | Core type coverage | TODO line 273 |
| 51 | Add BDD tests for `query.Pagination` (NewPagination, Offset, Validate) | query | 8m | Core type coverage | TODO line 273 |
| 52 | Increase projection coverage to 95%+ (currently 95.3% — add edge cases) | projection | 10m | Coverage target | TODO line 189 |
| 53 | Add fuzz test for `event.NewEvent()` (random types, versions, payloads) | event | 10m | Robustness | TODO line 274 |
| 54 | Add fuzz test for `id.Parse[T]()` (random strings, lengths, chars) | id | 8m | Robustness | TODO line 274 |
| 55 | Add fuzz test for `catalog.SchemaFromType[T]()` (random structs) | catalog | 10m | Robustness | TODO line 274 |
| 56 | Add fuzz test for `event.DecodePayload[T]()` (random bytes) | event | 8m | Robustness | TODO line 274 |

### P7: Code Quality & Architecture

| # | Task | Module | Est. | Impact | Source |
|---|---|---|---|---|---|
| 57 | Evaluate fake_store.go (273L) vs MemoryStore — decide: delete or test | event/eventtest | 10m | Removes duplication | TODO line 421 |
| 58 | Evaluate query `TypedHandler[T]` taking `Query` not `T` — document decision | query | 8m | API honesty | TODO line 419 |
| 59 | Add gofumpt + goimports to pre-commit hook config | .pre-commit | 10m | Format enforcement | TODO line 263 |
| 60 | Add 350-line test file limit check to pre-commit hook | .pre-commit | 8m | Size enforcement | TODO line 282 |
| 61 | Parallelize CI: one job per module (matrix strategy) | .github | 12m | 5-10× faster CI | TODO line 201 |

### P8: Documentation & Examples

| # | Task | Module | Est. | Impact | Source |
|---|---|---|---|---|---|
| 62 | Rewrite example/user/ — add command, decider, projection, query, signing demo | example/user | 12m | Onboarding | TODO line 258 |
| 63 | Add catalog registration to example/user/ | example/user | 8m | Full-stack demo | TODO line 258 |
| 64 | Add smoke test for rewritten example/user/ | example/user | 8m | CI coverage | TODO line 258 |
| 65 | Document E2E throughput benchmark methodology in docs/ | docs | 8m | Knowledge capture | TODO line 275 |
| 66 | Update TODO_LIST.md — mark go mod tidy items as DONE | TODO_LIST | 3m | Housekeeping | Session 141 |
| 67 | Update FEATURES.md — add scale benchmark results | FEATURES | 5m | Honest inventory | Session 141 |

---

## Summary by Category

| Category | Tasks | Total Est. | Avg/task |
|---|---|---|---|
| P0: Hygiene | 7 | 17m | 2.4m |
| P1: Performance (allocs) | 9 | 74m | 8.2m |
| P2: Benchmark coverage | 9 | 80m | 8.9m |
| P3: Benchmark quality | 14 | 51m | 3.6m |
| P4: Benchmark infra | 5 | 52m | 10.4m |
| P5: SIMD | 3 | 27m | 9.0m |
| P6: Test coverage | 9 | 82m | 9.1m |
| P7: Code quality | 5 | 48m | 9.6m |
| P8: Documentation | 6 | 44m | 7.3m |
| **TOTAL** | **67** | **475m ≈ 8h** | **7.1m** |

## Already Done This Session (before plan execution)

| # | Task | Status |
|---|---|---|
| 1 | `go mod tidy` projection/ | ✅ DONE |
| 2 | `go mod tidy` example/projection/ | ✅ DONE |
| 3 | `go mod tidy` example/saga-pattern/ | ✅ DONE |
| 4 | `go mod tidy` turso/ | ✅ DONE |

These 4 hygiene tasks are already fixed but not yet committed.
