# Comprehensive Status Report — go-cqrs-lite

**Date:** 2026-04-30 02:41 CEST  
**Branch:** master (1 commit ahead of origin)  
**Session:** #12 — Post-Integration Module Completion  
**Reporter:** Crush AI Assistant  

---

## a) FULLY DONE

### Module Architecture & Structure
| Item | Status | Detail |
|------|--------|--------|
| Multi-module monorepo | **COMPLETE** | 7 modules: `core`, `memory`, `catalog`, `middleware`, `testhelpers`, `integration` |
| `go.work` workspace | **COMPLETE** | All modules wired, tracked in VCS |
| Circular dependency fix | **COMPLETE** | `core/go.mod` has zero deps on `memory`/`testhelpers`. `integration/` holds cross-module tests. |
| `core` independent publishability | **COMPLETE** | `cd core && GOWORK=off go test ./...` passes with no replace directives |
| Per-module `go.mod` isolation | **COMPLETE** | Each module declares only its actual dependencies |

### Code Quality
| Item | Status | Detail |
|------|--------|--------|
| Lint-clean all modules | **COMPLETE** | 0 issues across core, memory, catalog, middleware, integration |
| Zero `TODO`/`FIXME`/`HACK` in codebase | **COMPLETE** | `grep` confirms zero occurrences |
| File size limits | **COMPLETE** | All production files ≤ 250 lines |
| Format clean | **COMPLETE** | `nix fmt` produces zero changes |
| Dead code removal | **COMPLETE** | `evtest` package, `xtypes` module, `internal/testhelpers` shim, stale replace directives all removed |
| Function size limits | **COMPLETE** | All functions ≤ 30 lines |

### Core Module (`core/`)
| Package | Non-Test Files | Test Files | Coverage | Status |
|---------|---------------|------------|----------|--------|
| `core/command` | 4 | 2 | **95.0%** | Unit tests in-package. Integration tests in `integration/command/` |
| `core/query` | 3 | 1 | **91.0%** | Unit tests in-package. BDD tests moved to `integration/query/` |
| `core/event` | 5 | 4 | **100.0%** | All paths covered |
| `core/aggregate` | 2 | 1 | **21.1%** | Only `aggregate.go` (Core struct) tested. `repository.go` has zero unit tests |
| `core/pkg/dispatcher` | 2 | 1 | **100.0%** | Complete coverage |
| `core/pkg/id` | 2 | 3 | **97.1%** | Encoding edge cases covered |
| **core total** | **33** | **13** | varies | 2,275 LoC non-test, 3,609 LoC test |

### Memory Module (`memory/`)
| Package | Non-Test Files | Test Files | Coverage | Status |
|---------|---------------|------------|----------|--------|
| `memory` | 5 | 4 | **98.9%** | `LoadAtVersion` (92.3%) and `Ack` (92.3%) have minor gaps |
| **memory total** | **5** | **4** | 98.9% | 548 LoC non-test, 1,152 LoC test |

### Catalog Module (`catalog/`)
| Package | Non-Test Files | Test Files | Coverage | Status |
|---------|---------------|------------|----------|--------|
| `catalog` | 5 | 5 | **94.3%** | `collectionSchema` and `goTypeToJSON` branches |
| `catalog/adapters` | 2 | 3 | **98.8%** | Near-complete |
| `catalog/asyncapi` | 3 | 3 | **97.6%** | `SchemaToAny` error path |
| `catalog/eventcatalog` | 4 | 3 | **95.5%** | `writeSchema` nil path, `writePackageJSON` error path |
| `catalog/internal/cattest` | 2 | 0 | 0.0% | Test helpers (no test files expected) |
| **catalog total** | **18** | **14** | varies | 2,159 LoC non-test, 3,801 LoC test |

### Middleware Module (`middleware/`)
| Package | Non-Test Files | Test Files | Coverage | Status |
|---------|---------------|------------|----------|--------|
| `middleware` | 8 | 10 | **100.0%** | All 8 source files fully covered |
| **middleware total** | **8** | **10** | 100.0% | 577 LoC non-test, 1,320 LoC test |

### Integration Module (`integration/`)
| Package | Non-Test Files | Test Files | Test Functions | Status |
|---------|---------------|------------|----------------|--------|
| `integration/aggregate` | 0 | 7 | ~25 | BDD (Ginkgo) + table-driven + benchmarks |
| `integration/command` | 0 | 2 | ~8 | Command dispatch integration tests |
| `integration/event` | 0 | 3 | ~15 | Event sourcing BDD + benchmarks |
| `integration/query` | 0 | 3 | ~12 | Query dispatch + BDD |
| **integration total** | **0** | **15** | **~45** | 3,034 LoC test-only |

### Testhelpers Module (`testhelpers/`)
| Package | Non-Test Files | Test Files | Status |
|---------|---------------|------------|--------|
| `testhelpers` | 1 | 0 | Shared Noop*, Failing*, Panic*, Callback*, Middleware, AppendEventsHandler |

### Interfaces & Middleware
| Interface/Feature | Status | File |
|-------------------|--------|------|
| `event.Store` | **COMPLETE** | `core/event/event.go` |
| `event.Bus` | **COMPLETE** | `core/event/event.go` |
| `event.SnapshotStore` | **COMPLETE** | `core/event/event.go` |
| `aggregate.Root` | **COMPLETE** | `core/aggregate/aggregate.go` |
| `aggregate.Repository` | **COMPLETE** | `core/aggregate/repository.go` |
| `command.Dispatcher` | **COMPLETE** | `core/command/dispatcher.go` |
| `query.Dispatcher` | **COMPLETE** | `core/query/dispatcher.go` |
| `CommandLogging` | **COMPLETE** | `middleware/logging.go` |
| `EventLogging` | **COMPLETE** | `middleware/logging.go` |
| `QueryLogging` | **COMPLETE** | `middleware/logging.go` |
| `CommandMetrics` | **COMPLETE** | `middleware/metrics.go` |
| `EventMetrics` | **COMPLETE** | `middleware/metrics.go` |
| `QueryMetrics` | **COMPLETE** | `middleware/metrics.go` |
| `CommandRecovery` | **COMPLETE** | `middleware/recovery.go` |
| `EventRecovery` | **COMPLETE** | `middleware/recovery.go` |
| `QueryRecovery` | **COMPLETE** | `middleware/recovery.go` |
| `CommandRetry` | **COMPLETE** | `middleware/retry.go` |
| `EventRetry` | **COMPLETE** | `middleware/retry.go` |
| `QueryRetry` | **COMPLETE** | `middleware/retry.go` |
| `CommandValidation` | **COMPLETE** | `middleware/validation.go` |
| `EventValidation` | **COMPLETE** | `middleware/validation.go` |
| `QueryValidation` | **COMPLETE** | `middleware/validation.go` |
| `CommandTracing` (OTel) | **COMPLETE** | `middleware/tracing.go` |
| `EventTracing` (OTel) | **COMPLETE** | `middleware/tracing.go` |
| `QueryTracing` (OTel) | **COMPLETE** | `middleware/tracing.go` |
| `SlogAdapter` | **COMPLETE** | `middleware/slog.go` |
| `MemoryOutboxStore` | **COMPLETE** | `memory/outbox.go` |
| `ApplySnapshot` (Root interface) | **COMPLETE** | `core/aggregate/aggregate.go` |
| Functional options for `EventSourcedRepository` | **COMPLETE** | `core/aggregate/repository.go` |
| Cached middleware chain | **COMPLETE** | `core/pkg/dispatcher/dispatcher.go` |
| Branded IDs (`id.Of[T]`) | **COMPLETE** | `core/pkg/id/` |
| ULID-based IDs | **COMPLETE** | `core/pkg/id/` |
| Catalog system (AsyncAPI + EventCatalog) | **COMPLETE** | `catalog/` |
| Golden tests (AsyncAPI/EventCatalog output) | **COMPLETE** | `catalog/asyncapi/`, `catalog/eventcatalog/` |
| Benchmarks (query dispatch, aggregate ops) | **COMPLETE** | `integration/*/benchmark_test.go` |

### CI / Build Infrastructure
| Item | Status |
|------|--------|
| Nix flake (`flake.nix`) | **COMPLETE** — test, test-race, coverage, build, vet, lint, clean |
| `go.work` in VCS | **COMPLETE** |
| GitHub Actions CI | **COMPLETE** — single `ci.yml` |
| `CONTRIBUTING.md` | **COMPLETE** |

---

## b) PARTIALLY DONE

| Item | What's Done | What's Missing | Severity |
|------|-------------|----------------|----------|
| **core/aggregate test coverage** | `aggregate.go` (Core struct) tested at 100% | `repository.go` (`EventSourcedRepository`) has **zero unit tests** in core/aggregate. Coverage is 21.1%. Integration tests in `integration/aggregate/` cover it via MemoryStore/MemoryBus, but core module alone cannot test its own repository. | **HIGH** |
| **Outbox pattern** | `MemoryOutboxStore` interface + implementation | Background publisher (polling + publish) not implemented. Only the store half exists. | MEDIUM |
| **Event upcasting** | Not started | No `Upcaster` interface, no registry, no migration path for event schema versions | MEDIUM |
| **SQL-backed storage** | Not started | No `storage/` module. All persistence is in-memory only. | CRITICAL |
| **Watermill pub/sub integration** | Not started | No `watermill/` module. `event.Bus` only has in-memory implementation. | HIGH |
| **Projection/read models** | Not started | No `projection/` module. The "Q" in CQRS is manual. | HIGH |
| **Snapshot strategies** | `SnapshotStore` interface exists | No strategy interface, no automatic snapshotting in repository | LOW |
| **Codec abstraction** | `event.Codec` not in core | JSON encoding is hardcoded. No pluggable encoding (protobuf, msgpack) | MEDIUM |
| **Examples** | `example/` modules removed in session 9 | No working examples for consumers to copy. `example/user/` was simplified but not restored. | MEDIUM |
| **Module versioning / tags** | No tags | All modules at v0.0.0. No stable releases tagged. | MEDIUM |

---

## c) NOT STARTED

### Phase 5: Storage Module (`storage/`)
- `storage/go.mod` — module creation
- PostgreSQL schema (events table, outbox table, indexes)
- sqlc config + generated queries
- `event.Store` adapter (`storage/eventstore.go`)
- Transactional outbox implementation
- Schema migration helpers
- SQLite support
- MySQL support
- Integration tests with testcontainers

### Phase 6: Watermill Module (`watermill/`)
- `watermill/go.mod` — module creation
- `event.Bus` adapter via Watermill Publisher
- Backend config helpers (Redis, NATS, Kafka)
- Unit tests with in-memory bus

### Phase 7: Projection Module (`projection/`)
- `projection/go.mod` — module creation
- `Runner` (subscribe + dispatch to projection handlers)
- `CheckpointStore` interface + SQL implementation
- `Projector` builder API (`projector.On("user.created", handler)`)
- Integration tests

### Phase 8: Snapshot Module (`snapshot/`)
- `snapshot/go.mod` — module creation
- SQL-backed `SnapshotStore`
- Snapshot strategies (every N events, time-based)
- Wire snapshot strategy into `EventSourcedRepository.Save()`

### Documentation
- Storage module design doc
- Saga / process manager design doc
- Event upcasting design doc
- README update with new module structure
- GitHub Pages `index.html` with go-import tags
- `CONTRIBUTING.md` update for new modules

### Release
- Git tags for all modules (`core/v1.0.0`, `memory/v1.0.0`, etc.)
- CI matrix for new modules

---

## d) TOTALLY FUCKED UP

| Item | Severity | Why It's Fucked | What Would Fix It |
|------|----------|-----------------|-------------------|
| **`core/aggregate` coverage: 21.1%** | 🔴 **CRITICAL** | `repository.go` (EventSourcedRepository with Load, Save, loadEvents, etc.) has **zero unit tests** after moving integration tests to `integration/aggregate/`. The only test file (`aggregate_test.go`) tests `aggregate.Core` (RecordEvent, Commit, UncommittedEvents) but not the repository at all. | Add unit tests for `EventSourcedRepository` that use minimal mock implementations of `event.Store` and `event.Bus` — **without** importing `memory` module. Or create a `core/internal/teststore` package with minimal test doubles. |
| **`core/command` coverage: 95.0%** | 🟡 MEDIUM | Only `dispatcher.go` and `command.go` are tested in-package. The 5% gap is likely error paths in `New`/`MustNew`. Not critical but noticeable. | Add 2–3 error path tests to `core/command/command_test.go` |
| **`core/query` coverage: 91.0%** | 🟡 MEDIUM | `pagination.go` is well-covered; `query.go` and `dispatcher.go` have gaps likely in error paths. | Add error path tests for `New`/`MustNew` and closed dispatcher edge cases |
| **Coverage report aggregates `integration/` into `core/`** | 🟡 MEDIUM | `nix run .#coverage` only runs `core`, `memory`, `catalog`, `middleware`, `integration` modules. The aggregate coverage is 85.9% but this masks that `core/aggregate` is 21.1% in isolation. Developers running `cd core && go test -cover` will see misleadingly low numbers. | Document that `core/aggregate` requires integration tests for full coverage, or add mock-based unit tests to core. |

### Honest Assessment: The 21.1% Problem

This is **not acceptable** for a module claiming production readiness. Here's the breakdown:

```
core/aggregate/
├── aggregate.go       (2413 bytes) — 100% covered by aggregate_test.go
├── repository.go      (4874 bytes) —  0% covered by anything in core/
```

`repository.go` contains:
- `EventSourcedRepository` struct
- `NewEventSourcedRepository()` constructor with functional options
- `Load()` — loads snapshot + replays events, handles version check
- `Save()` — saves uncommitted events, handles no-op, version conflict
- `loadEvents()` — helper for event replay
- `WithHistoryLoader()`, `WithSnapshotStore()`, `WithBus()` options

**All of this code is currently untested from the core module's perspective.**

The integration tests in `integration/aggregate/repository_test.go` cover it, but:
1. They require `memory` module → circular dependency if moved back
2. They don't run when you `cd core && go test ./...`
3. A consumer `go get`-ing core has no assurance the repository works

**This is the single biggest quality gap in the entire codebase.**

---

## e) WHAT WE SHOULD IMPROVE

### Immediate (This Session)
1. **Fix `core/aggregate` coverage crisis** — Create `core/internal/teststore/` with minimal `event.Store`/`event.Bus`/`event.SnapshotStore` test doubles. Write 15–20 unit tests for `EventSourcedRepository` without importing `memory`.
2. **Restore `core/command` to 100%** — 2–3 error path tests for `MustNew` panic and `New` validation.
3. **Restore `core/query` to 100%** — Error path tests for `New`/`MustNew` and closed dispatcher.

### Short-Term (Next 2–3 Sessions)
4. **Add `Codec` interface** — Pluggable event encoding (JSON, protobuf, msgpack). Currently hardcoded JSON.
5. **Add `Upcaster` interface** — Event schema versioning. V1 → V2 migration without data loss.
6. **Create `storage/` module** — PostgreSQL event store via sqlc. This is the most requested feature.
7. **Add `Projection` interface + `CheckpointStore`** — The missing "Q" in CQRS. Read models from event streams.
8. **Write design docs** — Storage, Saga, Upcasting. Each ≤ 1 hour of focused writing.
9. **Add working example** — A minimal `example/user/` that demonstrates the full flow.
10. **Tag v1.0.0 releases** — At least `core/v1.0.0` and `memory/v1.0.0`.

### Medium-Term (Next Month)
11. **Watermill module** — Redis/NATS/Kafka pub/sub for `event.Bus`.
12. **SQLite support in storage** — For embedded/testing use cases.
13. **Snapshot strategies** — Every N events, time-based, conditional.
14. **Background outbox publisher** — Poll outbox + publish to bus automatically.
15. **Performance benchmarks** — End-to-end throughput benchmarks (commands/sec, events/sec).
16. **Go 1.26 JSON v2 full migration** — Currently using `go-json-experiment/json`.
17. **Fuzz testing expansion** — More fuzz targets for event creation, ID parsing.
18. **GitHub Pages docs** — Go-import meta tags for subdirectory modules.
19. **Multi-engine CI matrix** — Test PostgreSQL, SQLite, MySQL in CI.
20. **Saga / process manager design** — Choreography vs orchestration research.

### Architectural Improvements
21. **Extract shared lifecycle pattern** — `Close()` + `CheckClosed` is duplicated in `memory/bus.go`, `memory/store.go`, `memory/snapshot.go`.
22. **Extract shared middleware chain pattern** — `Use()` + `wrap()` is duplicated across `memory/bus.go`, `command/dispatcher.go`, `query/dispatcher.go`.
23. **Event metadata type safety** — `Metadata` is `map[string]string`. Consider typed metadata with schema validation.
24. **Command/Query catalog consistency** — `command.Core` has catalog metadata; `query.Core` does too. Ensure parity.
25. **Aggregate root generic type** — `aggregate.Core` could be `Core[T any]` where `T` is the aggregate type, enabling type-safe `Repository[T]`.

---

## f) Top #25 Things We Should Get Done Next

| # | Task | Module | Effort | Impact | Priority |
|---|------|--------|--------|--------|----------|
| 1 | **Fix `core/aggregate` coverage (21.1% → 90%+)** | `core/aggregate` | 30min | 🔴 CRITICAL | **P0** |
| 2 | **Create `storage/` module + PostgreSQL schema** | `storage/` | 45min | 🔴 CRITICAL | **P0** |
| 3 | **Implement `event.Store` SQL adapter** | `storage/` | 30min | 🔴 CRITICAL | **P0** |
| 4 | **Add `Codec` interface to core** | `core/event` | 15min | HIGH | **P1** |
| 5 | **Add `Upcaster` interface to core** | `core/event` | 15min | HIGH | **P1** |
| 6 | **Add `Projection` interface + `CheckpointStore`** | `core/projection` | 20min | HIGH | **P1** |
| 7 | **Write storage module design doc** | `docs/planning/` | 15min | MEDIUM | **P1** |
| 8 | **Add `core/command` error path tests (95% → 100%)** | `core/command` | 10min | LOW | **P2** |
| 9 | **Add `core/query` error path tests (91% → 100%)** | `core/query` | 10min | LOW | **P2** |
| 10 | **Create `watermill/` module** | `watermill/` | 15min | HIGH | **P2** |
| 11 | **Implement `event.Bus` via Watermill** | `watermill/` | 30min | HIGH | **P2** |
| 12 | **Add working `example/user/`** | `example/` | 20min | MEDIUM | **P2** |
| 13 | **Tag `core/v1.0.0`** | Git | 5min | HIGH | **P2** |
| 14 | **Tag `memory/v1.0.0`** | Git | 5min | HIGH | **P2** |
| 15 | **Add SQLite storage support** | `storage/` | 30min | MEDIUM | **P3** |
| 16 | **Add snapshot strategies** | `core/aggregate` | 20min | MEDIUM | **P3** |
| 17 | **Implement background outbox publisher** | `storage/` or `middleware/` | 25min | MEDIUM | **P3** |
| 18 | **Add `catalog` coverage gaps** | `catalog/` | 20min | LOW | **P3** |
| 19 | **Extract shared lifecycle pattern** | `core/pkg/lifecycle/` | 15min | LOW | **P4** |
| 20 | **Extract shared middleware chain pattern** | `core/pkg/middleware/` | 15min | LOW | **P4** |
| 21 | **Write saga design doc** | `docs/planning/` | 15min | MEDIUM | **P4** |
| 22 | **Add GitHub Pages go-import tags** | `docs/index.html` | 10min | MEDIUM | **P4** |
| 23 | **Write upcasting design doc** | `docs/planning/` | 10min | MEDIUM | **P4** |
| 24 | **Add end-to-end throughput benchmarks** | `integration/` | 20min | LOW | **P5** |
| 25 | **Add `AggregateID` generic type parameter** | `core/aggregate` | 30min | MEDIUM | **P5** |

---

## g) Top #1 Question I Cannot Figure Out Myself

### "How do we test `EventSourcedRepository` in `core/aggregate` without either (a) re-creating the circular dependency by importing `memory`, or (b) adding 200+ lines of mock boilerplate that duplicates `memory` module logic?"

**The Problem:**

`EventSourcedRepository` has these dependencies:
```go
type EventSourcedRepository struct {
    store          event.Store        // for Save/Load events
    snapshotStore  event.SnapshotStore // optional, for Load
    bus            event.Bus          // optional, for Save (publish)
    historyLoader  func(ctx context.Context, id id.AggregateID) ([]event.Event, error)
}
```

To unit test it, we need implementations of `event.Store`, `event.Bus`, and `event.SnapshotStore`. The obvious choice is `memory.MemoryStore`, `memory.MemoryBus`, `memory.MemorySnapshotStore` — but importing `memory` from `core` re-creates the circular dependency we just spent hours breaking.

The alternatives:

**Option A: Mock implementations in `core/internal/teststore/`**
- Create ~150 lines of in-memory store/bus/snapshot implementations inside `core/`
- Pro: No circular dependency. Tests run with `cd core && go test`.
- Con: Duplicates `memory/` module logic. Maintenance burden. Two implementations of the same thing.

**Option B: Move `EventSourcedRepository` tests to `integration/aggregate/` only**
- Pro: Already done. `integration/aggregate/repository_test.go` covers it.
- Con: `core/aggregate` coverage is 21.1%. A consumer `go get`-ing core has no local test assurance that the repository works. This is philosophically unacceptable for a library.

**Option C: Create a `core/testing` sub-package (not module)**
- `core/testing/store.go` provides minimal test doubles
- Pro: Part of core module, no circular dependency
- Con: Increases core module's public API surface. `testing` package is importable by anyone.

**Option D: Use `testing/fstest`-style pattern — tiny in-memory structs in test files**
- Define 3 tiny structs (`fakeStore`, `fakeBus`, `fakeSnapshotStore`) directly in `core/aggregate/repository_test.go`
- Each ~20 lines, just enough for the test scenarios
- Pro: No new packages, no circular dependency, minimal duplication
- Con: Still ~60 lines of "mock" code in the test file

**My strong inclination is Option D**, but I cannot decide whether:
1. The 60 lines of mock code is acceptable duplication, OR
2. We should just accept that `core/aggregate` coverage is 21.1% and document that full coverage requires `integration/aggregate`

**What I need from you:** Should I:
- (a) Add `fakeStore`/`fakeBus`/`fakeSnapshotStore` directly in `core/aggregate/repository_test.go` (~60 lines, brings coverage to 90%+), OR
- (b) Leave it as-is and document the coverage gap in AGENTS.md, OR
- (c) Create `core/internal/teststore/` as a dedicated test double package (~150 lines, reusable across core tests)?

This decision affects our testing philosophy for all future core interfaces that need test doubles.

---

## Module Dependency Graph (Current)

```
testhelpers → core
memory      → core + testhelpers
middleware  → core + testhelpers
catalog     → core
integration → core + memory + testhelpers
core        → (no internal deps — independently publishable)
```

## File Counts

| Module | Non-Test Files | Test Files | Non-Test LoC | Test LoC | Test Functions |
|--------|---------------|------------|--------------|----------|----------------|
| core | 33 | 13 | 2,275 | 3,609 | 146 |
| memory | 5 | 4 | 548 | 1,152 | 43 |
| catalog | 18 | 14 | 2,159 | 3,801 | 116 |
| middleware | 8 | 10 | 577 | 1,320 | 46 |
| testhelpers | 1 | 0 | 142 | 0 | 0 |
| integration | 0 | 15 | 0 | 3,034 | 45 |
| **TOTAL** | **65** | **56** | **5,701** | **12,916** | **396** |

## Quality Gates (Current)

- [x] All tests pass (`nix run .#test`) — 17/17 packages ✅
- [x] No lint issues (`nix run .#lint`) — 5/5 modules, 0 issues ✅
- [x] Build compiles (`nix run .#build`) — ✅
- [x] Format clean (`nix fmt`) — ✅
- [ ] Coverage maintained or improved — ❌ `core/aggregate` dropped to 21.1%
- [x] AGENTS.md updated — ✅
- [x] Commit history clean — ✅

## Coverage Summary

| Package | Coverage | Target | Gap |
|---------|----------|--------|-----|
| `core/command` | 95.0% | 100% | -5.0% |
| `core/query` | 91.0% | 100% | -9.0% |
| `core/event` | 100.0% | 100% | ✅ |
| `core/aggregate` | 21.1% | 90% | **-68.9%** 🔴 |
| `core/pkg/dispatcher` | 100.0% | 100% | ✅ |
| `core/pkg/id` | 97.1% | 100% | -2.9% |
| `memory` | 98.9% | 100% | -1.1% |
| `catalog` | 94.3% | 95% | -0.7% |
| `catalog/adapters` | 98.8% | 100% | -1.2% |
| `catalog/asyncapi` | 97.6% | 100% | -2.4% |
| `catalog/eventcatalog` | 95.5% | 100% | -4.5% |
| `middleware` | 100.0% | 100% | ✅ |
| **TOTAL** | **85.9%** | **95%** | **-9.1%** |

---

*Report generated by Crush AI Assistant. All data from live codebase analysis.*
