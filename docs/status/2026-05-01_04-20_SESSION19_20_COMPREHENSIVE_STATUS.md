# Session 19-20: Comprehensive Status Report

**Date:** 2026-05-01 04:20 CEST  
**Author:** Session 19 (Crush) + Session 20 (concurrent Crush)  
**Commits since session 17:** 18  
**Total Go files:** 142 (65 test files)  
**Total Go LOC:** 22,585  
**Total tests passing:** 486  

---

## A) FULLY DONE ✅

These features are complete, tested, documented, and in CI.

### Core Module (`core/`)
| Feature | Status | Coverage | Notes |
|---------|--------|----------|-------|
| Command dispatch | ✅ | 100.0% | Dispatcher, Handler, Middleware, Catalog |
| Query dispatch | ✅ | 100.0% | Pagination, PaginatedResult, DispatchTyped |
| Event types | ✅ | 96.3% | NewEvent, Builder, Metadata, Options |
| Aggregate roots | ✅ | 95.6% | Root, Repository, EventSourcedRepository |
| Branded IDs | ✅ | 100.0% | id.Of[T], ULID-backed, all encoding |
| Generic dispatcher | ✅ | 100.0% | LifecycleMixin, CheckClosed, Register guards |
| Projection interface | ✅ | 96.3% | ProjectionFunc, subscribesTo |
| InMemoryRunner | ✅ | 96.3% | Handle, Register, checkpoint tracking |
| UpcasterRegistry | ✅ | Tested | `==` version comparison (fixed from `>=`) |
| Snapshot strategy | ✅ | Tested | EveryNEvents, Strategy interface |
| Codec interface | ✅ | Tested | DecodePayload[T] |
| ContextEnricher | ✅ | Tested | CompositeEnricher |
| CheckpointStore | ✅ | Interface | Interface defined, memory impl exists |

### Memory Module (`memory/`)
| Feature | Status | Coverage | Notes |
|---------|--------|----------|-------|
| MemoryStore | ✅ | 99.0% | Save, Load, LoadFromVersion, Delete, defensive copies |
| MemoryBus | ✅ | 100.0% | Publish, Subscribe, SubscribeAll, lifecycle |
| MemorySnapshotStore | ✅ | 100.0% | Save, Load, Delete, deep copy |
| MemoryCheckpointStore | ✅ | 99.0% | Load, Save, overwrite |
| MemoryOutboxStore | ✅ | 100.0% | Append, PollPending |

### Catalog Module (`catalog/`)
| Feature | Status | Coverage | Notes |
|---------|--------|----------|-------|
| Registry | ✅ | 94.4% | Thread-safe, Build() → Catalog |
| SchemaFromType[T] | ✅ | Tested | Reflection-based, reads json/doc/format tags |
| MessageID extraction | ✅ | Tested | Unified in catalog.MessageID() |
| AsyncAPI 3.0 export | ✅ | 97.9% | YAML + JSON, golden tests |
| EventCatalog export | ✅ | 95.5% | MDX files, golden tests |
| CatalogBuilder adapter | ✅ | 98.8% | Generic AddCommandFromType[T], AddEventFromType[T] |
| FromCommandDispatcher | ✅ | Tested | Extracts catalog from live dispatcher |

### Middleware Module (`middleware/`)
| Feature | Status | Coverage | Notes |
|---------|--------|----------|-------|
| CommandLogging | ✅ | 99.4% | Structured logging |
| EventLogging | ✅ | 99.4% | Structured logging |
| CommandRetry | ✅ | 99.4% | Exponential backoff, ctx cancellation |
| EventRetry | ✅ | 99.4% | Exponential backoff |
| CommandRecovery | ✅ | 99.4% | Panic recovery |
| EventRecovery | ✅ | 99.4% | Panic recovery |
| CommandValidation | ✅ | 99.4% | ValidateFunc |
| EventValidation | ✅ | 99.4% | ValidateFunc |
| QueryValidation | ✅ | 99.4% | ValidateFunc |
| CommandMetrics | ✅ | 99.4% | MetricsRecorder interface |
| EventMetrics | ✅ | 99.4% | MetricsRecorder interface |
| QueryMetrics | ✅ | 99.4% | MetricsRecorder interface |

### Infrastructure
| Feature | Status | Notes |
|---------|--------|-------|
| Multi-module workspace | ✅ | 8 modules in go.work |
| CI (flake.nix) | ✅ | test, lint, build, vet, coverage for all 6 test modules |
| example/user build | ✅ | Built as part of `nix run .#build` |
| Dev shell | ✅ | Go 1.26, golangci-lint, gofumpt, golines, trash-cli |
| Lint config | ✅ | Zero issues across all modules |
| CONTRIBUTING.md | ✅ | Multi-module workflow documented |

### Testhelpers
| Feature | Status | Notes |
|---------|--------|-------|
| FakeStore | ✅ | Thread-safe event store fake |
| FakeBus | ✅ | Event bus fake with collected events |
| FakeSnapshotStore | ✅ | Snapshot store fake |
| FakeCheckpointStore | ✅ | No-op checkpoint store |
| FakeOutbox | ✅ | Outbox store fake |
| All handler/middleware helpers | ✅ | 15+ helpers |

---

## B) PARTIALLY DONE ⚠️

### Storage Module (`storage/`) — 79.8% coverage
| Feature | Status | Coverage Gap | Notes |
|---------|--------|--------------|-------|
| SQLEventStore.Save | ⚠️ | Missing begin-tx error path | Happy path + concurrency conflict tested |
| SQLEventStore.AppendBatch | ⚠️ | Missing begin-tx error path | Transactional happy path tested |
| SQLEventStore.Load | ⚠️ | Missing query error path | Happy path + not-found tested |
| SQLEventStore.LoadFromVersion | ⚠️ | Missing query error path | Happy path tested |
| SQLEventStore.Delete | ⚠️ | Missing exec error path | Happy path tested |
| SQLEventStore.Close | ⚠️ | Complete |  |
| scanEvents | ⚠️ | Missing parse/unmarshal error paths | Metadata roundtrip tested |
| Schema() DDL | ❌ | Untested | Function exists, no test |
| SQL SnapshotStore | ❌ | Not started | Interface exists in core |
| SQL CheckpointStore | ❌ | Not started | Interface exists in core |

### example/user — Compiles but no tests
| Feature | Status | Notes |
|---------|--------|-------|
| User aggregate | ⚠️ | Full lifecycle demo, no test files |
| Catalog generation | ⚠️ | Generates EventCatalog output, not verified |

### Catalog Schema type
| Feature | Status | Notes |
|---------|--------|-------|
| Format field | ❌ | `format` struct tag read but not stored in Property |
| Description field | ❌ | `doc`/`description` tags partially read |

---

## C) NOT STARTED ❌

| Feature | Priority | Issue | Notes |
|---------|----------|-------|-------|
| Watermill module | MEDIUM | #11 | event.Bus with Kafka/NATS/AMQP |
| SQL SnapshotStore | MEDIUM | #12 | PostgreSQL-backed snapshots |
| SQL CheckpointStore | MEDIUM | #13 | PostgreSQL-backed checkpoints |
| v0.1.0 release tag | LOW | #14 | Need README update + go doc check |
| CatalogBuilder/Registry dedup | LOW | #6 | Split brain: duplicated accumulation logic |
| example/user tests | MEDIUM | #8 | Zero test files |
| NewEvent refactor | LOW | #16 | 66-line function, should be 5-10 |
| README installation docs | LOW | — | Needed for v0.1.0 |
| Process Manager / Saga | — | — | Fantasy feature, not planned |
| Circuit breaker middleware | — | — | No consumer needs it |

---

## D) TOTALLY FUCKED UP 💥

### 1. Storage was a ghost system (sessions 14-17)
We spent **5 commits fixing bugs** in a module that had **zero tests, zero consumers, zero CI**. The module compiled and had correct logic, but:
- Not in `flake.nix` test matrix
- No `go-sqlmock` dependency
- `integration/go.mod` did not depend on `storage`
- Nobody would have noticed if it silently broke

**Status:** Fixed in sessions 18-19. Now in CI, 79.8% coverage, 12 tests.

### 2. JSON v1/v2 split brain
`storage/event_store.go` used `encoding/json` (v1) while everything else used `go-json-experiment/json` (v2). Compatible by accident — adding any v2-only tag would silently break metadata serialization.

**Status:** Fixed in session 18. Storage now uses go-json-experiment/json consistently.

### 3. Coverage reporting was misleading
AGENTS.md reported "99.1% core/event" but this was from an earlier measurement. After moving projection tests to `integration/`, coverage dropped to 86.7%. We shipped this without investigating.

**Status:** Fixed in session 19. Now 96.3% with direct Handle/subscribesTo tests.

### 4. We polished docs instead of writing tests
Sessions 16-17 produced 2 status reports, updated 3 doc files, regenerated golden files — but added **zero storage tests**. The most critical module (data persistence) had the least verification.

**Status:** Fixed in session 19. 12 storage tests added.

### 5. UpcasterRegistry `>=` instead of `==` (session 17)
Events were re-upcasted by ALL upcasters with sourceVersion ≤ N instead of exactly matching N. This meant a chain of upcasters v1→v2→v3 would apply v2→v3 transformation multiple times.

**Status:** Fixed in session 17. Edge case tests added.

### 6. Storage silently discarded event IDs and timestamps
`scanEvents` used `event.NewEvent()` which auto-generates new IDs/timestamps. Original event IDs and timestamps from the database were lost.

**Status:** Fixed in session 15. `WithEventID` and `WithOccurredAt` options added.

---

## E) WHAT WE SHOULD IMPROVE 🔄

### Process Issues
1. **Test before you fix.** We fixed 5 storage bugs before writing a single test. Should have been: write tests → discover bugs → fix bugs.
2. **Ghost system detection.** Any new module should be added to CI immediately. Consider a pre-commit hook that checks `go.work` modules against `flake.nix` testModules.
3. **Coverage regression alerts.** When coverage drops from 99% to 86%, we should fail CI, not just note it in a doc.
4. **Per-session issue hygiene.** Every session should create/update GitHub issues, not just write status reports.

### Code Issues
5. **testhelpers/fakes.go is 342 lines.** Exceeds 250-line convention. Should split into fakes_store.go, fakes_bus.go, fakes_snapshot.go, etc.
6. **catalog/internal/cattest/helpers.go is 330 lines.** Should split into helpers.go + assertions.go (started in session 8 but regressed).
7. **core/pkg/id/id_test.go is 947 lines.** Should split by concern: parse, encoding, comparison, format, ptr.
8. **NewEvent is 66 lines.** Should extract validation + option application helpers.
9. **CatalogBuilder duplicates Registry.** Two implementations of the same accumulation pattern (issue #6).

### Architecture Issues
10. **No error path testing in storage.** All tests are happy-path. Need DB failure scenarios.
11. **example/user has zero tests.** The only consumer of catalog builder API. Breakage goes unnoticed.
12. **No merged coverage report.** Per-module coverage hides cross-module gaps.
13. **No benchmark regression tracking.** Benchmarks exist but aren't compared between commits.

---

## F) TOP 25 NEXT ACTIONS (by impact/effort)

| # | Action | Impact | Effort | Issue | Customer Value |
|---|--------|--------|--------|-------|----------------|
| 1 | Storage error path tests (begin/query/parse failures) | HIGH | 30min | #10 | Data integrity |
| 2 | Add Schema() DDL test | MEDIUM | 5min | #10 | SQL correctness |
| 3 | example/user smoke test | MEDIUM | 30min | #8 | Demo doesn't break |
| 4 | Split testhelpers/fakes.go (342→4×85) | MEDIUM | 20min | #7 | Maintainability |
| 5 | Split id_test.go (947→5×190) | LOW | 15min | — | Maintainability |
| 6 | Split cattest/helpers.go (330→2×165) | LOW | 10min | — | Maintainability |
| 7 | Storage integration test (testcontainers?) | HIGH | 90min | — | Real DB verification |
| 8 | Watermill module: event.Bus impl | HIGH | 120min | #11 | Production pub/sub |
| 9 | SQL CheckpointStore | MEDIUM | 60min | #13 | Production projections |
| 10 | SQL SnapshotStore | MEDIUM | 60min | #12 | Production snapshots |
| 11 | CatalogBuilder/Registry dedup | MEDIUM | 45min | #6 | Eliminate split brain |
| 12 | Add Format/Description to Schema Property | LOW | 30min | #15 | Richer documentation |
| 13 | Refactor NewEvent (66→5 lines) | LOW | 20min | #16 | Maintainability |
| 14 | Coverage regression CI gate (fail if <95%) | MEDIUM | 15min | — | Catch regressions |
| 15 | go.work ↔ flake.nix sync check (pre-commit) | LOW | 15min | — | Prevent ghost modules |
| 16 | Merged coverage report across modules | MEDIUM | 30min | — | Accurate total coverage |
| 17 | README installation instructions | LOW | 20min | — | Consumer onboarding |
| 18 | go doc render check for public APIs | LOW | 15min | — | API documentation |
| 19 | Tag v0.1.0 releases per module | LOW | 30min | #14 | Go module discoverability |
| 20 | Benchmark regression tracking | LOW | 60min | — | Performance visibility |
| 21 | Add ErrConcurrencyConflict sentinel | LOW | 10min | — | Type-safe error handling |
| 22 | Storage: test AppendBatch with single event | LOW | 5min | — | Edge case |
| 23 | Add EmptyEventTypes → subscribesToAll doc | LOW | 5min | — | API clarity |
| 24 | Consider sqlc for storage queries | LOW | 60min | — | Type-safe SQL |
| 25 | Add CONTRIBUTING.md testing section for storage | LOW | 5min | — | Contributor onboarding |

---

## G) TOP #1 BLOCKING QUESTION 🤔

**Should `storage/` use `sqlc` (https://sqlc.dev/) for type-safe SQL generation?**

Right now, `storage/event_store.go` has hand-written SQL with string literals and manual `rows.Scan()`. This is error-prone:
- Column order must match between SELECT and Scan
- Adding a column requires updating 4+ query strings
- No compile-time SQL validation

`sqlc` generates Go code from SQL files, giving us:
- Compile-time column name checking
- Automatic scan struct generation
- Query validation against a PostgreSQL schema

**Trade-off:** Adds `sqlc` as a build dependency. The storage module would need a `sqlc.yaml` config and `queries.sql` file. Generated code adds ~200 lines but eliminates an entire class of bugs.

**Why I can't decide:** This changes the build toolchain (needs sqlc binary) and project structure. It's an architectural decision that affects all consumers who might want to write their own storage queries.

---

## Raw Metrics

| Metric | Value |
|--------|-------|
| Total test packages | 17 (passing) |
| Total tests | 486 (passing) |
| Total Go files | 142 |
| Total test files | 65 |
| Total Go LOC | 22,585 |
| Lint issues | 0 |
| Race conditions | 0 |
| Coverage (weighted avg) | ~95.8% |
| Lowest coverage | storage 79.8% |
| Highest coverage | core/command, core/query, core/pkg/dispatcher, core/pkg/id 100% |
| Open GitHub issues | 9 |
| Closed GitHub issues | 6 |
| Workspace modules | 8 (core, memory, catalog, middleware, storage, testhelpers, integration, example/user) |
| Uncommitted work | 1 compiled binary (`user`) — should be gitignored |
| Commits ahead of origin | 1 (`ee47a3c`) |

## File Size Violations (>250 lines)

| File | Lines | Action |
|------|-------|--------|
| testhelpers/fakes.go | 342 | Split by fake type |
| catalog/internal/cattest/helpers.go | 330 | Split helpers + assertions |
| storage/event_store.go | 249 | ✅ Under limit |
| core/pkg/id/id_test.go | 947 | Split by concern |

## Coverage Trend

| Package | Session 14 | Session 17 | Session 19 | Δ |
|---------|-----------|-----------|-----------|---|
| core/command | 100.0% | 100.0% | 100.0% | — |
| core/query | 100.0% | 100.0% | 100.0% | — |
| core/pkg/dispatcher | 100.0% | 100.0% | 100.0% | — |
| core/pkg/id | 97.1% | 97.1% | **100.0%** | +2.9 |
| core/event | 99.1% | 86.7% | **96.3%** | +9.6 |
| core/aggregate | 95.7% | 95.7% | 95.6% | -0.1 |
| middleware | 100.0% | 100.0% | 99.4% | -0.6 |
| memory | 98.9% | 94.9% | **99.0%** | +4.1 |
| catalog | 94.4% | 94.4% | 94.4% | — |
| catalog/adapters | 98.8% | 98.8% | 98.8% | — |
| catalog/asyncapi | 97.6% | 97.6% | 97.9% | +0.3 |
| catalog/eventcatalog | 95.5% | 95.5% | 95.5% | — |
| storage | 0% | 0% | **79.8%** | +79.8 |

## Session Commit History (sessions 16-20)

| Commit | Session | Description |
|--------|---------|-------------|
| `4324713` | 16 | Cleanup — docs, examples, projection tests, fixes |
| `1ce2672` | 16 | Fix toDotAddress for acronyms and numbers |
| `dc37350` | 17 | AsyncAPI key collision, lint cleanup, test alignment |
| `ffcf31c` | 17 | Session 17 bug fix sprint report |
| `0e28862` | 17 | Session 17 comprehensive status report |
| `e1cd8ae` | 18 | Session 18 honest audit and execution plan |
| `7820b04` | 18 | JSON v2 migration, CI fix |
| `d80664e` | 18 | FakeStore key separator fix |
| `ea8907d` | 19 | Storage unit tests with go-sqlmock (0% → 79.6%) |
| `eb85233` | 19 | InMemoryRunner Handle/subscribesTo tests (86.7% → 96.3%) |
| `0056ce2` | 19 | Ptr/FromPtr (100%) + CheckpointStore (99%) tests |
| `05ad9f4` | 19 | FakeCheckpointStore in testhelpers |
| `b8f4ef3` | 19 | CatalogCore unit tests |
| `91bebef` | 19 | ProjectionFunc.Handle direct tests |
| `75f2d90` | 19 | Remove dead ProjectionRunner interface |
| `1763348` | 19 | Extract aggregate options and snapshot strategy |
| `0ab5883` | 19 | example/user build in CI |
| `8cd3441` | 19 | Update AGENTS.md |
| `143c27e` | 19 | Extract catalog generation from example |
| `c3e90e7` | 19 | Extract storage helpers and schema |
| `ee47a3c` | 20 | Prior session uncommitted — lint, formatting, fixes |
