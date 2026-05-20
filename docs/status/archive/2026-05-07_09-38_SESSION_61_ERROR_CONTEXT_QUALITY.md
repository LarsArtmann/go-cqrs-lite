# go-cqrs-lite — Comprehensive Status Report

**Date:** 2026-05-07 09:38
**Branch:** master
**Commits since May 1:** 165
**Sessions:** 61

---

## Executive Summary

The project is in **excellent health**. Session 61 focused on error context quality:

- **22/22 test packages pass** (zero failures)
- **0 lint issues across all 8 linted modules**
- **branching-flow context score: 95.4 → 96.5/100** (+1.1)
- **29 → 16 error paths** with missing context (45% reduction)
- **11 → 6 HIGH severity** issues remaining (5 fixed, 6 are false positives)
- **41 → 22 MEDIUM severity** issues remaining (19 fixed, 22 are diminishing returns)

12 files changed across 4 modules — all error messages now include domain-identifying context (aggregateType, aggregateID, serviceID, etc.) that was previously lost.

---

## A) FULLY DONE ✅

### Architecture & Core Types

| Item                            | Status      | Detail                                                                                               |
| ------------------------------- | ----------- | ---------------------------------------------------------------------------------------------------- |
| Event sourcing with branded IDs | ✅ Complete | `id.Of[T]` type alias to `go-branded-id`, all interfaces return branded types                        |
| ISP on event.Bus                | ✅ Complete | `Publisher` + `Subscriber` sub-interfaces; repos accept `Publisher`, projections accept `Subscriber` |
| Error taxonomy (5 families)     | ✅ Complete | Rejection, Conflict, Transient, Corruption, Infrastructure — 48 sentinels classified                 |
| Extensible classification       | ✅ Complete | `RegisterClassification(sentinel, family)` — external packages register via `init()`                 |
| Decider pattern                 | ✅ Complete | Pure-function aggregate: `Decider[State]`, `Repository[State]`, `Execute`, `Load`                    |
| Aggregate pattern (OO)          | ✅ Complete | `Root`, `Repository`, `EventSourcedRepository` — kept for backward compat                            |
| Snapshot strategy               | ✅ Complete | Shared `event.SnapshotStrategy`, `EveryNEvents`, `ShouldSnapshot`, `SaveSnapshot`                    |
| Shared repository helpers       | ✅ Complete | `event.PublishChanges()`, `event.SaveSnapshot()` — eliminated duplication in aggregate/decider       |
| Command/Query dispatchers       | ✅ Complete | Generic `Dispatcher[H, M]` with middleware chain, catalog entries                                    |
| TypedHandler[T] for queries     | ✅ Complete | `query.TypedHandler[T]`, `query.RegisterTyped[T]()` — compile-time type safety                       |
| Branded types (ULID-backed)     | ✅ Complete | `AggregateID`, `EventID`, `UserID`, `CorrelationID`, `ClientID`, `RequestID`                         |
| Offline-first metadata          | ✅ Complete | `WithClientID`, `WithClientOccurredAt` options on events                                             |
| Context-aware operations        | ✅ Complete | All handlers accept `context.Context`                                                                |

### Storage Backends

| Item                                    | Status      | Detail                                                                      |
| --------------------------------------- | ----------- | --------------------------------------------------------------------------- |
| PostgreSQL event store                  | ✅ Complete | `SQLEventStore` — Save, Load, LoadFromVersion, AppendBatch, Delete, LoadAll |
| SQLite event store                      | ✅ Complete | `SQLiteEventStore` — same interface, `?` placeholders                       |
| Pebble KV event store                   | ✅ Complete | `CQRSAdapter` — embedded key-value store for edge/offline                   |
| PostgreSQL snapshot store               | ✅ Complete | `SQLSnapshotStore`                                                          |
| SQLite snapshot store                   | ✅ Complete | `SQLiteSnapshotStore`                                                       |
| PostgreSQL outbox                       | ✅ Complete | `SQLOutbox` with batch chunking                                             |
| SQLite outbox                           | ✅ Complete | `SQLiteOutbox`                                                              |
| PostgreSQL checkpoint store             | ✅ Complete | `SQLCheckpointStore`                                                        |
| SQLite checkpoint store                 | ✅ Complete | `SQLiteCheckpointStore`                                                     |
| TransactionalStore (atomic save+outbox) | ✅ Complete | Both PG and SQLite implementations                                          |
| Turso connector                         | ✅ Complete | `OpenTursoSync`, `TursoSyncDB` with Push/Pull/Checkpoint/Stats              |
| Turso memory DB guard                   | ✅ Complete | `ErrTursoMemorySync` sentinel prevents in-memory sync                       |

### Catalog & Documentation

| Item                    | Status      | Detail                                                              |
| ----------------------- | ----------- | ------------------------------------------------------------------- |
| AsyncAPI 3.0 export     | ✅ Complete | YAML + JSON, `go-faster/yaml`, string constants extracted           |
| EventCatalog MDX export | ✅ Complete | Services, schemas, frontmatter                                      |
| D2 diagram export       | ✅ Complete | Color-coded nodes, cross-service connections                        |
| Schema reflection       | ✅ Complete | `SchemaFromType[T]()` via reflect, reads `json`/`doc`/`format` tags |
| Registry pattern        | ✅ Complete | Thread-safe, `AddService`, `AddCommand`, `AddEvent`, `AddQuery`     |
| Catalog adapters        | ✅ Complete | `CatalogBuilder`, `FromDispatcher` extraction                       |
| `catalog.MessageID()`   | ✅ Complete | Canonical message ID extraction from catalog messages               |

### Middleware

| Item                         | Status      | Detail                                                                                         |
| ---------------------------- | ----------- | ---------------------------------------------------------------------------------------------- |
| Command logging              | ✅ Complete | `slog`-based, respects context                                                                 |
| Command retry                | ✅ Complete | Exponential backoff, uses `event.IsRetryable()`                                                |
| Command recovery             | ✅ Complete | Panic recovery with `ErrPanicRecovered` sentinel                                               |
| Command validation           | ✅ Complete | `ErrValidationFailed` sentinel                                                                 |
| Command metrics              | ✅ Complete | OpenTelemetry integration                                                                      |
| Event validation             | ✅ Complete | API symmetry with command                                                                      |
| Query validation             | ✅ Complete | Same pattern                                                                                   |
| All middleware use sentinels | ✅ Complete | `middleware.ErrValidationFailed`, `ErrRetryExhausted`, `ErrRetryCanceled`, `ErrPanicRecovered` |

### Error Context Quality (Session 61)

| File                                  | Fix                                                          | Severity      |
| ------------------------------------- | ------------------------------------------------------------ | ------------- |
| `catalog/adapters/builder.go`         | Added `serviceID` to `ErrDomainNotFound`                     | HIGH          |
| `catalog/registry.go`                 | Added `serviceID` to `ErrDomainNotFound`                     | HIGH          |
| `catalog/eventcatalog/exporter.go`    | Added `svcID`/`kind` to 3 error paths (mkdir, write, schema) | HIGH          |
| `catalog/eventcatalog/writer.go`      | Added `schemaDir` to schema dir error                        | MEDIUM        |
| `catalog/internal/cattest/catalog.go` | Added `eventType` to catalog core error                      | MEDIUM        |
| `storage/helpers.go`                  | Added `aggType`/`version` to parse/reconstruction errors     | HIGH + MEDIUM |
| `storage/snapshot.go`                 | Added `aggregateType`/`aggregateID` to snapshot errors       | HIGH + MEDIUM |
| `storage/sqlite_helpers.go`           | Added `aggregateType`/`aggregateID` to snapshot scan         | MEDIUM        |
| `storage/outbox.go`                   | Added `limit` to poll pending error                          | MEDIUM        |
| `storage/sqlite_outbox.go`            | Added `limit` to poll pending error                          | MEDIUM        |
| `storage/pebble_event_store.go`       | Added `count`/`logMsg` to commit error                       | MEDIUM        |
| `storage/turso_connector.go`          | Added `remoteURL` to sync errors                             | MEDIUM        |

### Testing & Quality

| Item                              | Status       | Detail                                                                                           |
| --------------------------------- | ------------ | ------------------------------------------------------------------------------------------------ |
| Test-to-production ratio          | ✅ 2.0:1     | ~24,547 test LOC / ~12,150 production LOC                                                        |
| Zero lint (all 8 modules)         | ✅ Complete  | 0 issues across core, memory, catalog, middleware, storage, projection, integration, testhelpers |
| Zero TODO/FIXME                   | ✅ Complete  | None in production code                                                                          |
| All exported symbols documented   | ✅ Complete  | Godoc on all exported types/functions                                                            |
| Compile-time interface checks     | ✅ 37 checks | `var _ Interface = (*Impl)(nil)` pattern                                                         |
| Benchmarks                        | ✅ 43 total  | Across 10 files, 7 modules                                                                       |
| Integration tests                 | ✅ Complete  | BDD (Ginkgo) for event, aggregate, command, query                                                |
| Cross-module classification tests | ✅ Complete  | `integration/event/classify_test.go`                                                             |
| Concurrent access tests           | ✅ Complete  | MemoryStore (10 goroutines × 50 ops), MemoryBus (5 × 20 events) with `-race`                     |

### Error Sentinels (48 total)

| Package      | Count | Key Sentinels                                                                                                   |
| ------------ | ----- | --------------------------------------------------------------------------------------------------------------- |
| `event`      | 17    | `ErrVersionConflict`, `ErrAggregateNotFound`, `ErrStoreClosed`, `ErrProjectionPanicked`                         |
| `command`    | 5     | `ErrHandlerNotFound`, `ErrDispatcherClosed`, `ErrEmptyCommandType`                                              |
| `query`      | 4     | `ErrHandlerNotFound`, `ErrDispatcherClosed`, `ErrEmptyQueryType`                                                |
| `aggregate`  | 4     | `ErrNilStore`, `ErrNilBus`, `ErrSnapshotNotFound`                                                               |
| `decider`    | 6     | `ErrNilStore`, `ErrNilBus`, `ErrNilFold`, `ErrLoadFailed`, `ErrSaveFailed`                                      |
| `projection` | 3     | `ErrDuplicateProjection`, `ErrNilCheckpointStore`                                                               |
| `middleware` | 4     | `ErrValidationFailed`, `ErrRetryExhausted`, `ErrPanicRecovered`                                                 |
| `storage`    | 9     | `ErrNilDB`, `ErrAggregateTypeMismatch`, `ErrVersionMismatch`, `ErrPebbleProviderRequired`, `ErrTursoMemorySync` |

### Dependencies

| Item                            | Status      | Detail                                          |
| ------------------------------- | ----------- | ----------------------------------------------- |
| cockroachdb/errors removed      | ✅ Complete | Replaced with `fmt.Errorf` + `%w`               |
| go-json-experiment/json removed | ✅ Complete | Replaced with `encoding/json`                   |
| go-faster/yaml retained         | ✅ Complete | Zero-transitive-dep YAML library                |
| go-branded-id adopted           | ✅ Complete | Type alias `id.Of[T]` = `cbid.ID[T, ulid.ULID]` |
| `slices.Backward` adopted       | ✅ Complete | Idiomatic Go 1.21+ reverse iteration            |
| `math/rand/v2` for jitter       | ✅ Complete | Replaced `crypto/rand` in middleware            |

---

## B) PARTIALLY DONE ⚠️

### branching-flow Context Score (96.5/100 — was 95.4)

6 remaining HIGH issues are **false positives**: `scanEvent` in `storage/helpers.go:77` and `storage/sqlite_helpers.go:56` — variables (`idStr`, `eventType`, `aggType`, `aggIDStr`, `version`) aren't populated when `rows.Scan` fails, so adding them to the error message would log zero values. The `scan event row` error already provides enough context (the SQL error itself describes the scan failure).

22 remaining MEDIUM issues are **diminishing returns**: redundant context in nested calls, security-excluded `authToken`, `opError` format string variable, `version` in `NewEvent` validation (already wrapped with sentinel + field names).

### SQL Dialect Duplication (6 file pairs, ~500 lines duplicated)

- `event_store.go` ↔ `sqlite_event_store.go` (~95% identical, only `$1` vs `?`)
- `snapshot.go` ↔ `sqlite_snapshot.go` (~90% identical, time format differs)
- `outbox.go` ↔ `sqlite_outbox.go` (~85% identical)
- `checkpoint.go` ↔ `sqlite_checkpoint.go` (~90% identical)
- `transactional_store.go` ↔ `sqlite_transactional_store.go` (~95% identical)
- `helpers.go` ↔ `sqlite_helpers.go` (~70% identical)

**Status:** Deduplication planned but not started. A `Dialect` interface would consolidate to a single implementation. The `dupl` linter is suppressed via `.golangci.yml` for `storage/`.

---

## C) NOT STARTED ❌

### Architecture

| Item                                | Impact | Effort | Notes                                                                |
| ----------------------------------- | ------ | ------ | -------------------------------------------------------------------- |
| SQL dialect abstraction             | HIGH   | MEDIUM | Would eliminate ~500 lines of duplication                            |
| `io.Closer` removal from interfaces | MEDIUM | LOW    | Breaking change, deferred                                            |
| `CatalogMeta` consolidation         | LOW    | LOW    | `event.CatalogMeta` has extra `AggregateType` field                  |
| Shared `opError` helper             | LOW    | LOW    | Two different signatures in aggregate vs decider                     |
| Query handler type safety           | HIGH   | HIGH   | 24 bare `any` in params/returns; `TypedHandler[T]` is the workaround |

### Documentation & Process

| Item                                          | Impact | Effort | Notes                                |
| --------------------------------------------- | ------ | ------ | ------------------------------------ |
| `docs/planning/SAGA_DESIGN.md` implementation | HIGH   | HIGH   | 4-phase plan exists, no code written |
| `docs/planning/QUERY_HANDLER_GENERICS.md`     | MEDIUM | MEDIUM | TypedHandler migration plan          |
| `docs/planning/OUTBOX_TRANSACTION_API.md`     | MEDIUM | MEDIUM | TransactionalStore design doc        |
| API stability / versioning strategy           | MEDIUM | LOW    | No semver guarantees yet             |
| CONTRIBUTING.md                               | LOW    | LOW    | No contribution guide                |
| Generated API reference (GoDoc)               | LOW    | LOW    | No hosted godoc                      |

### Features

| Item                                | Impact | Effort | Notes                                          |
| ----------------------------------- | ------ | ------ | ---------------------------------------------- |
| Saga/process manager support        | HIGH   | HIGH   | Design doc exists                              |
| Event upcasting (versioned schemas) | MEDIUM | MEDIUM | `UpcasterRegistry` exists with cycle detection |
| Watermill adapter                   | MEDIUM | MEDIUM | Listed as "planned" in catalog                 |
| NATS/Kafka bus implementations      | MEDIUM | HIGH   | Would need `event.Subscriber` impl             |
| Projection replayer CLI             | LOW    | MEDIUM | Could use `event.GlobalLoader`                 |
| Event signing/verification          | LOW    | MEDIUM | Listed in non-goals                            |

---

## D) TOTALLY FUCKED UP 💥

### Nothing is critically broken.

All tests pass, all lint is clean, all modules build. The codebase is in the best shape it has ever been.

### Known Issues (LOW severity)

| Issue                                                    | Severity | Detail                                                                 |
| -------------------------------------------------------- | -------- | ---------------------------------------------------------------------- |
| `MemoryBus.Publish` holds RLock during handler execution | LOW      | Subscribers block publishers; acceptable for test utility              |
| `query.Handler` returns `any`                            | LOW      | Violates "no any" rule; `DispatchTyped[T]` is the workaround           |
| `CatalogMeta` duplicated across 3 packages               | LOW      | Nearly identical but `event.CatalogMeta` has extra field               |
| `Root.LoadEvents` vs `Core.LoadFromHistory` mismatch     | LOW      | Every aggregate must implement `LoadEvents` and delegate               |
| `core/go.mod` circular dependency                        | LOW      | `memory`+`testhelpers` as test-only deps, resolved locally via replace |
| 5 production files over 250 lines                        | LOW      | Largest: `pebble_event_store.go` (445 lines)                           |
| ~26 functions over 30 lines                              | LOW      | Worst: `writeCrossServiceConnections` (59), `Export` asyncapi (55)     |

---

## E) WHAT WE SHOULD IMPROVE

### 1. Storage Module Coverage (~85% → 95%+)

The Pebble store has good coverage but Turso connector and some error paths in SQL stores are undertested. Focus on:

- Pebble delete/load edge cases
- Turso sync error paths
- SQL store reconnection/error scenarios

### 2. SQL Dialect Abstraction

The 6 file pairs in storage are the single largest source of duplication in the codebase. A `Dialect` interface with `Placeholder(n)`, `FormatTime(t)`, and `TimeScan()` would consolidate everything.

### 3. Function Length (26 functions over 30 lines)

The worst offenders are in catalog export logic (`writeCrossServiceConnections` at 59 lines, `Export` at 55). These could be decomposed into smaller helpers.

### 4. File Size (5 files over 250 lines)

`pebble_event_store.go` at 445 lines is the most urgent. Could be split into `pebble_event_store.go`, `pebble_serialization.go`, `pebble_iterators.go`.

### 5. Query Handler Type Safety

24 bare `any` usages propagate through the entire middleware chain. The `TypedHandler[T]` workaround exists but the core `Handler` type still returns `(any, error)`.

### 6. `core/go.mod` Circular Dependency

`core` depends on `memory` and `testhelpers` for tests only, creating a module cycle. Options: separate test module, or move test-only deps behind build tags.

### 7. Cross-Package Sentinel Consolidation

`ErrNilStore` exists in both `aggregate` and `decider` with different messages. `ErrHandlerNotFound` exists in `command`, `query`, and `dispatcher`. Could re-export from a canonical location.

---

## F) Top #25 Things We Should Get Done Next

### Tier 1: HIGH Impact, LOW Effort (Do First)

| #   | Task                                                | Impact      | Effort | Why                                        |
| --- | --------------------------------------------------- | ----------- | ------ | ------------------------------------------ |
| 1   | Split `pebble_event_store.go` (445→3 files)         | Quality     | 1h     | Largest file in codebase, hard to navigate |
| 2   | Storage test coverage 85%→95%                       | Reliability | 2h     | Lowest coverage module                     |
| 3   | Decompose `writeCrossServiceConnections` (59 lines) | Quality     | 30m    | Longest function in codebase               |
| 4   | Decompose `Export` in asyncapi (55 lines)           | Quality     | 30m    | Second longest function                    |
| 5   | Add `example/user/` integration test                | Trust       | 1h     | Example has no automated test              |
| 6   | Update `FEATURES.md` with storage sentinels         | Docs        | 15m    | 9 new sentinels not documented             |

### Tier 2: HIGH Impact, MEDIUM Effort (Do Soon)

| #   | Task                                                       | Impact       | Effort | Why                                         |
| --- | ---------------------------------------------------------- | ------------ | ------ | ------------------------------------------- |
| 7   | SQL dialect abstraction (eliminate ~500 lines duplication) | Architecture | 4h     | Biggest source of dupl in codebase          |
| 8   | Split `core/decider/decider.go` (265→under 250)            | Quality      | 30m    | One of 5 files over limit                   |
| 9   | Split `core/aggregate/repository.go` (279→under 250)       | Quality      | 30m    | Already has extracted helpers, still over   |
| 10  | Split `catalog/asyncapi/exporter.go` (258→under 250)       | Quality      | 30m    | Export function is the bottleneck           |
| 11  | Split `storage/sqlite_event_store.go` (285→under 250)      | Quality      | 30m    | Post-dialect-abstraction would resolve      |
| 12  | Add `storage/dialect.go` with `Dialect` interface          | Architecture | 2h     | Foundation for SQL dedup                    |
| 13  | `core/go.mod` circular dependency fix                      | Health       | 1h     | Test-only deps create module cycle          |
| 14  | Cross-package sentinel consolidation                       | Quality      | 2h     | `ErrNilStore` × 2, `ErrHandlerNotFound` × 3 |

### Tier 3: MEDIUM Impact, MEDIUM Effort (Plan For)

| #   | Task                                                | Impact    | Effort | Why                                           |
| --- | --------------------------------------------------- | --------- | ------ | --------------------------------------------- |
| 15  | `io.Closer` removal from core interfaces            | API       | 2h     | Breaking change, needs design                 |
| 16  | `CatalogMeta` consolidation across 3 packages       | Quality   | 1h     | Nearly identical types                        |
| 17  | Shared `opError` helper (aggregate + decider)       | DRY       | 30m    | Two different signatures doing the same thing |
| 18  | Add `CONTRIBUTING.md`                               | Community | 1h     | No contribution guide exists                  |
| 19  | Benchmark storage backends (PG vs SQLite vs Pebble) | Perf      | 2h     | No perf data for storage                      |
| 20  | Add Turso integration test (requires Turso account) | Coverage  | 2h     | Turso connector only tested with :memory:     |

### Tier 4: HIGH Impact, HIGH Effort (Strategic)

| #   | Task                                | Impact        | Effort | Why                                       |
| --- | ----------------------------------- | ------------- | ------ | ----------------------------------------- |
| 21  | Saga/process manager implementation | Feature       | 18h    | Design doc exists (`SAGA_DESIGN.md`)      |
| 22  | Query handler generics migration    | Type Safety   | 8h     | Eliminate 24 bare `any` usages            |
| 23  | Watermill adapter (Kafka/NATS)      | Extensibility | 8h     | Listed as planned                         |
| 24  | API stability guarantees + semver   | Governance    | 4h     | Library consumers need stability promises |
| 25  | Event signing/verification          | Security      | 8h     | Listed in roadmap non-goals, but valuable |

---

## G) Top #1 Question I Cannot Figure Out Myself

**Should the remaining 6 HIGH branching-flow issues in `scanEvent` be fixed or accepted as false positives?**

The `scanEvent` function in `storage/helpers.go:77` and `storage/sqlite_helpers.go:56` declares local variables (`idStr`, `eventType`, `aggType`, `aggIDStr`, `version`) but `rows.Scan` populates them. When Scan fails, these are zero/empty values. Adding them to the error message would show misleading context like `idStr="", eventType=""`.

**Options:**

1. **Accept as false positive** — The `scan event row` error + the underlying SQL error is sufficient for debugging
2. **Restructure to scan first, then error with context** — Use a multi-step scan approach where partial values are captured before the error
3. **Add a debug-level log** — Log whatever was partially scanned before the error

---

## Session 61 Activity Log

### Session 61: Branching-Flow Error Context Improvement

- Ran `branching-flow context .` — scored 95.4/100 with 29 error paths
- Fixed 13 error messages across 12 files to include domain context variables
- Re-ran analysis — scored 96.5/100 with 16 error paths (45% reduction)
- Zero lint, 22/22 test packages pass

---

## Project Health Scorecard

| Dimension                     | Score                    | Trend         |
| ----------------------------- | ------------------------ | ------------- |
| Test coverage                 | ~91% avg                 | ↗️ Stable     |
| Lint issues                   | **0**                    | ✅ Stable     |
| Dead code                     | **0**                    | ✅ Stable     |
| TODO/FIXME                    | **0**                    | ✅ Stable     |
| File size compliance          | 95% (5/124 over 250)     | → Flat        |
| Function length compliance    | ~80% (~26 over 30 lines) | → Flat        |
| Sentinels with classification | 100% (48/48)             | ✅ Stable     |
| Compile-time interface checks | 37                       | ✅ Stable     |
| Benchmarks                    | 43                       | ✅ Stable     |
| Error context quality         | 96.5/100                 | ↗️ (was 95.4) |
| Dependency health             | Excellent                | ✅ Stable     |
