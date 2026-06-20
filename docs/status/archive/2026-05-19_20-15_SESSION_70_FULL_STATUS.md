# Session 70 Full Status Report

**Date:** 2026-05-19 20:15  
**Session Focus:** Zero-Cost Catalog API (Phase 1) + Core Quality Fixes (Phase 2 partial) + BaseDispatcher Removal  
**Commits This Session:** 5  
**Commits Ahead of Origin:** 5  
**Test Status:** All 22 packages pass

---

## Executive Summary

Delivered the zero-cost catalog API that makes documentation generation "free" for consumers — schemas, names, and directions are auto-derived from existing Go struct types via reflection and generics. Eliminated the need for consumers to write a second fake CQRS system just to feed the catalog builder.

Also fixed two split-brain conditions in core types and removed the useless `BaseDispatcher` abstraction.

---

## Detailed Work Log

### A) FULLY DONE

#### 1. Zero-Cost Catalog API (Phase 1 — Complete)

| File                                        | Action                                                        | Lines           |
| ------------------------------------------- | ------------------------------------------------------------- | --------------- |
| `catalog/build.go`                          | NEW — `Builder` with `AddService()`, `AddDomain()`, `Build()` | +67             |
| `catalog/message_config.go`                 | NEW — Generic `Command[T]()`, `Event[T]()`, `Query[T]()`      | +116            |
| `catalog/auto_name.go`                      | NEW — `camelCaseToHuman()` name derivation                    | +48             |
| `catalog/build_test.go`                     | NEW — 9 tests for Builder API                                 | +209            |
| `catalog/auto_name_test.go`                 | NEW — 10 tests for name algorithm                             | +36             |
| `catalog/adapters/builder.go`               | REFACTORED — wraps `catalog.Builder`                          | -36 net         |
| `catalog/adapters/from_query_dispatcher.go` | REFACTORED — uses registry directly                           | -20 net         |
| `core/command/catalog.go`                   | DEPRECATED — `Catalogable`, `CatalogCore`, `CatalogMeta`      | +3 doc comments |
| `core/event/catalog.go`                     | DEPRECATED — same                                             | +3 doc comments |
| `core/query/catalog.go`                     | DEPRECATED — same                                             | +3 doc comments |
| **DELETED**                                 | `adapters/command.go`, `event.go`, `query.go`, `message.go`   | -189            |
| **DELETED**                                 | `adapters_fromtype_test.go`, `adapters_dispatcher_test.go`    | -422            |
| **DELETED**                                 | `core/pkg/catalogmeta/catalogmeta.go`                         | -12             |
| `example/user/catalog.go`                   | REWRITTEN — `mustNewCatalogEvent` → `catalog.Event[T]()`      | -25 net         |

**Consumer API before:**

```go
// 30+ lines, fake aggregate IDs, dummy payloads
builder.AddEvent("svc", mustNewCatalogEvent(string(evtType), "Name", "Summary"))
```

**Consumer API after:**

```go
builder.AddService("svc", "Name", "1.0.0", "Summary",
    catalog.Event[UserCreatedPayload](string(evtType), catalog.Sends,
        catalog.Name("User Created"),
        catalog.Summary("Fired when..."),
    ),
)
```

#### 2. OutboxPublisher Split Brain Fix (Phase 2)

- Replaced `closed bool` + `cancel context.CancelFunc` with `publisherState` enum
- Three explicit states: `publisherIdle`, `publisherRunning`, `publisherClosed`
- Single source of truth eliminates drift between dual representations

#### 3. Aggregate Version/Changes Split Brain Fix (Phase 2)

- `SetVersion()` now clears `changes` slice to prevent version/changes inconsistency
- Documented the clearing behavior in godoc

#### 4. BaseDispatcher Abstraction Removal (Phase 2)

- **DELETED:** `core/pkg/dispatcher/base.go` (44 lines of pure delegation)
- **DELETED:** 121 lines of BaseDispatcher tests
- `command.Dispatcher` and `query.Dispatcher` now embed `*dispatcher.Dispatcher` directly
- Net -67 lines, zero behavioral change, simpler mental model

#### 5. Golden File Updates

- Updated `catalog/testdata/golden/asyncapi.yaml`, `eventcatalog-config.js`, `package.json`
- Fixed go-faster/yaml indentation drift

---

### B) PARTIALLY DONE

#### CatalogMeta Consolidation

- Attempted to extract `catalogmeta.Meta` base struct
- Reverted because embedding broke struct literal syntax for existing tests
- **Decision:** Keep 3× `CatalogMeta` structs with deprecation notices instead
- **Impact:** Low — structs are deprecated anyway with zero-cost API

#### Missing Unit Tests

- Attempted to add `builder_test.go`, `options_test.go`, `enricher_test.go`
- Discovered tests already exist in `event_test.go` for options
- `builder_test.go` and `enricher_test.go` were created but not committed (stale file issue)
- **Status:** Coverage gaps remain in builder, enricher, snapshot_helper, publish_helper

---

### C) NOT STARTED

| Task Group                      | Tasks                             | Est. Time                                    |
| ------------------------------- | --------------------------------- | -------------------------------------------- |
| Storage Dialect Deduplication   | 13 tasks (design + 6 store pairs) | ~3h                                          |
| File Size Compliance            | 6 files to split                  | ~1h                                          |
| Version/SchemaVersion → uint    | 9 tasks (type + propagation)      | ~1.5h                                        |
| SubscriptionScope Enum          | 3 tasks                           | ~30min                                       |
| MemoryBus Handler Consolidation | 2 tasks                           | ~20min                                       |
| Error Wrapping Standardization  | 2 tasks                           | ~30min                                       |
| DispatchTyped as Method         | 1 task                            | BLOCKED (Go doesn't support generic methods) |

---

### D) TOTALLY FUCKED UP (And Fixed)

| Issue                           | What Happened                                          | How Fixed                                                      |
| ------------------------------- | ------------------------------------------------------ | -------------------------------------------------------------- |
| Duplicate test names            | `options_test.go` tests clashed with `event_test.go`   | Deleted `options_test.go`                                      |
| Broken registry helper tests    | Referenced non-exported `copyService` etc.             | Deleted `registry_helpers_test.go`                             |
| Broken projection options tests | Referenced `fakeGlobalLoader` from another test file   | Deleted `options_test.go`                                      |
| Go generic method syntax        | Tried `func (d *Dispatcher) DispatchTyped[T any](...)` | Reverted to free function — Go doesn't support generic methods |

---

### E) WHAT WE SHOULD IMPROVE

1. **Test coverage gaps:** `event.Builder` (0% direct), `event.EnrichEvent` (0% direct), `event.PublishChanges`/`SaveSnapshot` (0% direct), `projection/options.go` (0% direct)
2. **Storage duplication:** 5 pairs of near-identical files (SQL vs SQLite) — 50%+ of storage/ is copy-paste
3. **File sizes:** 7 files exceed 250-line limit (test files dominate)
4. **Golden file fragility:** `go-faster/yaml` produces different indentation on some runs
5. **AGENTS.md too long:** 813 lines, go-structure-linter flagged it
6. **Query.Handler returns `any`:** Runtime type erasure — no compile-time safety

---

### F) Top #25 Things To Get Done Next

| #   | Priority | Task                                                                         | Est.  | Module                 |
| --- | -------- | ---------------------------------------------------------------------------- | ----- | ---------------------- |
| 1   | **P0**   | Design `storage.Dialect` interface                                           | 10min | storage                |
| 2   | **P0**   | Refactor `SQLEventStore` + `SQLiteEventStore` → single store with dialect    | 20min | storage                |
| 3   | **P0**   | Refactor `SQLSnapshotStore` + `SQLiteSnapshotStore` → single store           | 20min | storage                |
| 4   | **P0**   | Refactor `SQLCheckpointStore` + `SQLiteCheckpointStore` → single store       | 15min | storage                |
| 5   | **P0**   | Refactor `SQLOutbox` + `SQLiteOutbox` → single store                         | 20min | storage                |
| 6   | **P0**   | Refactor `SQLTransactionalStore` + `SQLiteTransactionalStore` → single store | 15min | storage                |
| 7   | **P0**   | Update storage tests for dialect-based stores                                | 20min | storage                |
| 8   | **P1**   | Add `builder_test.go` for `event.Builder`                                    | 10min | core/event             |
| 9   | **P1**   | Add `enricher_test.go` for `CompositeEnricher`/`EnrichEvent`                 | 10min | core/event             |
| 10  | **P1**   | Add `publish_helper_test.go` for `PublishChanges`/`SaveSnapshot`             | 10min | core/event             |
| 11  | **P1**   | Add `snapshot_helper_test.go` for `ShouldSnapshot`                           | 8min  | core/event             |
| 12  | **P1**   | Add `projection/options_test.go` with proper fake types                      | 10min | projection             |
| 13  | **P1**   | Split `core/decider/decider_test.go` (1146 lines)                            | 10min | core/decider           |
| 14  | **P1**   | Split `projection/runner_test.go` (1057 lines)                               | 10min | projection             |
| 15  | **P1**   | Split `core/aggregate/repository_test.go` (875 lines)                        | 10min | core/aggregate         |
| 16  | **P2**   | Change `event.Version` → `uint`                                              | 15min | core/event             |
| 17  | **P2**   | Change `event.SchemaVersion` → `uint`                                        | 10min | core/event             |
| 18  | **P2**   | Propagate `uint` versions to storage/memory/example                          | 30min | multiple               |
| 19  | **P2**   | Define `SubscriptionScope` enum                                              | 8min  | core/event             |
| 20  | **P2**   | Update `SubscribesTo` + `projection.Runner` for enum                         | 15min | core/event, projection |
| 21  | **P3**   | Consolidate MemoryBus `handlers` + `allHandlers` maps                        | 15min | memory                 |
| 22  | **P3**   | Standardize storage error wrapping                                           | 15min | storage                |
| 23  | **P3**   | Fix dynamic errors with sentinels across modules                             | 15min | multiple               |
| 24  | **P3**   | Split `catalog/schema_test.go` (604 lines)                                   | 10min | catalog                |
| 25  | **P3**   | Split `core/event/runner_test.go` (439 lines)                                | 10min | core/event             |

---

### G) Top #1 Question I Cannot Figure Out Myself

> **Should `query.Handler` remain returning `any`, or should we break the API and make it fully generic?**
>
> The current design:
>
> ```go
> type Handler = func(context.Context, Query) (any, error)
> func DispatchTyped[T any](ctx context.Context, d *Dispatcher, query Query) (T, error)
> ```
>
> The alternative (breaking):
>
> ```go
> type Handler[T any] = func(context.Context, Query) (T, error)
> type Dispatcher[T any] struct { ... }
> ```
>
> The breaking change would eliminate runtime type assertions entirely but affects EVERY consumer. The `DispatchTyped` escape hatch works but is a separate call site. Is the tradeoff worth it for a v1.0 library?

---

## Test Results

```
ok  	github.com/larsartmann/go-cqrs-lite/core/aggregate
ok  	github.com/larsartmann/go-cqrs-lite/core/command
ok  	github.com/larsartmann/go-cqrs-lite/core/decider
ok  	github.com/larsartmann/go-cqrs-lite/core/event
ok  	github.com/larsartmann/go-cqrs-lite/core/pkg/dispatcher
ok  	github.com/larsartmann/go-cqrs-lite/core/pkg/id
ok  	github.com/larsartmann/go-cqrs-lite/core/query
ok  	github.com/larsartmann/go-cqrs-lite/memory
ok  	github.com/larsartmann/go-cqrs-lite/catalog
ok  	github.com/larsartmann/go-cqrs-lite/catalog/adapters
ok  	github.com/larsartmann/go-cqrs-lite/catalog/asyncapi
ok  	github.com/larsartmann/go-cqrs-lite/catalog/d2
ok  	github.com/larsartmann/go-cqrs-lite/catalog/docserver
ok  	github.com/larsartmann/go-cqrs-lite/catalog/eventcatalog
ok  	github.com/larsartmann/go-cqrs-lite/catalog/openapi
ok  	github.com/larsartmann/go-cqrs-lite/middleware
ok  	github.com/larsartmann/go-cqrs-lite/integration/aggregate
ok  	github.com/larsartmann/go-cqrs-lite/integration/command
ok  	github.com/larsartmann/go-cqrs-lite/integration/event
ok  	github.com/larsartmann/go-cqrs-lite/integration/query
ok  	github.com/larsartmann/go-cqrs-lite/projection
ok  	github.com/larsartmann/go-cqrs-lite/storage
```

**22/22 packages pass.** Zero compilation errors. Zero lint issues in production code.

---

## Commit Log (This Session)

| Commit    | Message                                                                                  |
| --------- | ---------------------------------------------------------------------------------------- |
| `85b67c3` | fix(catalog): update golden files for go-faster/yaml indentation                         |
| `0d812cf` | feat(catalog): complete zero-cost API migration with example rewrite                     |
| `4d97b44` | fix(core): eliminate split brain conditions in OutboxPublisher and Aggregate             |
| `4fe3607` | docs(status): add Session 70 status report                                               |
| `2f008b5` | refactor(core): delete BaseDispatcher abstraction, inline into command/query dispatchers |
