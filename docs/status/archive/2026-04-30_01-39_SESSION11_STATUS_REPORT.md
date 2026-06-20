# Session 11 — Deep Fixes & Architecture Hardening

**Date:** 2026-04-30 01:39 CEST
**Session:** 11 (continuation from 2026-04-29 ~22:00)
**Commits:** 10 new commits since last status report
**Branch:** master (pushed to origin)

---

## a) FULLY DONE (Completed in This Session)

### 1. Delete stale `go.work.example`

- **Commit:** `52f6424`
- Removed file referencing deleted `xtypes` module (was orphaned since session 9)

### 2. Fix `MarkChangesAsCommitted` slice reuse

- **Commit:** `83726e9`
- **File:** `core/aggregate/aggregate.go:72`
- Changed `a.changes = make([]event.Event, 0)` to `a.changes = a.changes[:0]`
- **Impact:** Eliminates per-save garbage allocation (GC pressure reduction)

### 3. Add `ApplySnapshot` to `Root` interface

- **Commit:** `83726e9`
- **Files:** `core/aggregate/aggregate.go`, `core/aggregate/repository.go`, +6 test files
- `Root` now requires `ApplySnapshot(state []byte) error`
- `EventSourcedRepository.Load` calls `root.ApplySnapshot(snapshot.State)` after `SetVersion`
- `order.ApplySnapshot` deserializes JSON `{"status": "..."}` — makes snapshot tests meaningful
- All test aggregates implement no-op `ApplySnapshot`
- **Impact:** Snapshot support is now real, not a ghost system. Previously the repository set version from snapshot but never restored state.

### 4. Repository functional options refactor

- **Commit:** `d660b7f`
- **File:** `core/aggregate/repository.go`
- Replaced 3 constructors (`NewRepository`, `NewRepositoryWithSnapshot`, `NewRepositoryWithOutbox`) with:
  - `NewRepository(store, bus, opts ...RepositoryOption)`
  - `WithSnapshotStore(store)` option
  - `WithOutbox(outbox)` option
- **Impact:** Any combination of snapshot + outbox now possible (previously snapshot+outbox together was impossible)
- Extracted `loadEvents` helper to fix `nestif` complexity (was 5 levels deep)
- Added `id` import for typed `AggregateID` parameter

### 5. Cache middleware chain at registration

- **Commit:** `5fdf0f4`
- **Files:** `core/pkg/dispatcher/dispatcher.go`, `base.go`, `command/dispatcher.go`, `query/dispatcher.go`, `dispatcher_test.go`
- `Register` now accepts `wrap func(M, H) H` and stores the wrapped handler directly
- `Dispatch` becomes a direct lookup (no wrap parameter needed)
- **Impact:** Eliminates per-dispatch middleware chain recomputation. Every dispatch is now O(1) handler lookup.
- **BREAKING:** `Dispatcher.Register` and `Dispatcher.Dispatch` signatures changed. All test callers updated.

### 6. Test outbox failure after store success

- **Commit:** `9e126f6`
- **File:** `core/aggregate/outbox_test.go`
- Added `failingOutbox` mock and test verifying:
  - `Save` returns error when `outbox.Append` fails after `store.Save` succeeds
  - `MarkChangesAsCommitted` is NOT called (events remain uncommitted)
  - Store still contains the events (they were persisted before outbox failure)
- **Impact:** Documents and verifies the critical invariant: outbox failure must not lose events from store.

### 7. Remove acked entries from memory outbox

- **Commit:** `e85f37b`
- **Files:** `memory/outbox.go`, `memory/outbox_test.go`
- Removed `acked` field from `outboxEntry` struct
- `Ack` now filters acked entries out of the slice entirely (builds filtered slice)
- `PollPending` simplified (no acked check needed)
- Added `TestMemoryOutboxStore_Ack_RemovesEntryFromSlice` verifying acked entries truly removed
- **Impact:** Frees memory, cleaner internal state. Previously acked entries accumulated forever.

### 8. Error grammar consistency

- **Commit:** `7dca50e`
- **File:** `core/command/dispatcher.go:47`
- Changed `"register handler"` → `"registering handler"` (matches query dispatcher pattern)
- **Impact:** Consistent error message grammar across all dispatchers.

### 9. `AddServiceToDomain` returns error in builder

- **Commit:** `ab5dfad`
- **Files:** `catalog/adapters/builder.go`, `catalog/adapters/adapters_dispatcher_test.go`
- `CatalogBuilder.AddServiceToDomain` now returns `error` (was void, silently no-opped)
- Matches `Registry.AddServiceToDomain` behavior
- Tests updated to assert error on nonexistent domain
- **Impact:** Behavioral divergence between builder and registry eliminated.

### 10. Registry godoc additions

- **Commit:** `f114285`
- **File:** `catalog/registry.go`
- Added godoc comments to all 10 exported methods: `Registry`, `NewRegistry`, `AddService`, `AddCommand`, `AddEvent`, `AddQuery`, `AddDomain`, `AddServiceToDomain`, `AddChannel`, `Build`

### 11. Comprehensive execution plan created

- **File:** `docs/planning/2026-04-30_01-20_COMPREHENSIVE_EXECUTION_PLAN.md`
- 76 tasks across 8 tiers, sorted by impact/effort/customer-value score
- Estimated total: ~11 hours of work
- 4-week recommended schedule

---

## b) PARTIALLY DONE (In Progress or Needs More Work)

### Multi-module monorepo migration

- **Phases 0–4:** DONE (core, memory, catalog, middleware extracted)
- **Phase 5 (Storage):** NOT STARTED — PostgreSQL event store planned
- **Phase 6 (Watermill):** NOT STARTED — pub/sub abstraction planned
- **Phase 7 (Projection):** NOT STARTED — read models planned
- **Phase 8 (Snapshot):** PARTIALLY DONE — snapshot store interface exists, but SQL-backed implementation not started
- **Phase 9 (Test utilities):** DONE — `testhelpers/` module exists
- **Phase 10 (Tags):** NOT STARTED — waiting for stability

### Coverage campaign

- **core/command:** 100.0% — DONE
- **core/query:** 100.0% — DONE
- **core/pkg/dispatcher:** 100.0% — DONE
- **memory:** 98.9% — MOSTLY DONE (2 branches uncovered)
- **middleware:** 99.3% — MOSTLY DONE (1 branch: `DefaultRetryConfig.IsRetryable`)
- **catalog/adapters:** 98.8% — MOSTLY DONE (1 branch)
- **core/event:** 98.2% — MOSTLY DONE (2 branches: `ensureMetadata` when metadata exists, `Metadata()` nil path)
- **core/pkg/id:** 97.1% — MOSTLY DONE (binary encoding error paths)
- **catalog/asyncapi:** 97.6% — MOSTLY DONE (1 branch: `SchemaToAny` marshal error)
- **catalog/eventcatalog:** 95.5% — MOSTLY DONE (3 branches)
- **core/aggregate:** 94.4% — MOSTLY DONE (2 branches: `loadEvents` snapshot error, `Save` bus publish error)
- **catalog:** 94.3% — MOSTLY DONE (3 branches: `collectionSchema`, `goTypeToJSON`, `propertyFromReflect`)

### Known Issues (from AGENTS.md)

| Issue                                                    | Severity | Status                                                   |
| -------------------------------------------------------- | -------- | -------------------------------------------------------- |
| `MemoryBus.Publish` holds RLock during handler execution | LOW      | Acknowledged, acceptable for test utility                |
| `MemorySnapshotStore` deep copy of `State []byte`        | LOW      | Acknowledged — `Load` returns shallow copy               |
| `toDotAddress` number handling                           | LOW      | Acknowledged — "Get3DView" → "get.3.d.view"              |
| No `EventRetry` tests                                    | LOW      | Acknowledged — shares same retry logic as `CommandRetry` |

---

## c) NOT STARTED (From Execution Plan)

### Tier 1: Coverage quick wins (19 tasks)

- All 19 tasks identified but not executed
- Includes: dead `evtest` removal, `ensureMetadata` test, `Metadata()` nil test, `loadEvents` error test, etc.

### Tier 2: Architecture foundation (9 tasks)

- `Codec` interface (JSON, protobuf, msgpack)
- `Upcaster` interface (event versioning V1→V2)
- `Projection` interface (the missing "Q" in CQRS)
- `CheckpointStore` interface
- `SnapshotStrategy` interface
- Extract shared `Close()` pattern
- Extract shared `Use()` middleware pattern

### Tier 3: Storage module (14 tasks)

- `storage/` directory + `go.mod`
- PostgreSQL schema (events + outbox tables)
- sqlc configuration + code generation
- `event.Store` SQL adapter
- Transactional outbox implementation
- Integration tests with testcontainers
- SQLite/MySQL support

### Tier 4: Watermill module (5 tasks)

- `watermill/` directory + `go.mod`
- `event.Bus` implementation via Watermill
- Backend config helpers (Redis, NATS, Kafka)
- Unit tests

### Tier 5: Projection module (6 tasks)

- `projection/` directory + `go.mod`
- `Runner` (subscribe + dispatch)
- `Checkpoint` SQL store
- `Projector` builder API
- Integration tests

### Tier 6: Snapshot module (6 tasks)

- SQL-backed `SnapshotStore`
- Snapshot strategies (every N events, time-based)
- Wire strategy into repository `Save`
- Integration tests

### Tier 7: Known issue fixes (5 tasks)

- Fix `MemorySnapshotStore` deep copy
- Fix `toDotAddress` number handling
- Document `MemoryBus.Publish` RLock behavior
- Add `EventRetry` direct tests
- Add BDD test for aggregate error paths

### Tier 8: Documentation + Release (12 tasks)

- Update AGENTS.md
- Update README with new module structure
- Write storage module design doc
- Git tags: `core/v1.0.0`, `memory/v1.0.0`, `catalog/v1.0.0`, `middleware/v1.0.0`
- GitHub Pages with go-import meta tags
- CI matrix for new modules
- Example modules in go.work + CI

---

## d) TOTALLY FUCKED UP (Critical Problems)

### None.

The codebase is in its best state ever:

- **Zero lint issues** across all 4 active modules (core, memory, catalog, middleware)
- **All tests pass** (`go test ./...` green)
- **Formatting clean** (`nix flake check` passes)
- **No broken imports** (except LSP false positive on deleted `xtypes`)
- **No known crashes or data loss bugs**

**However:** The `core/event/internal/evtest` package is completely dead (0% coverage, zero imports). It should be deleted.

---

## e) WHAT WE SHOULD IMPROVE (Priority Order)

### 1. Start the storage module (Phase 5)

This is the #1 blocker. Without a real SQL-backed event store, this library is just a toy. The multi-module architecture, interfaces, and memory implementations are all solid — but we need PostgreSQL persistence to be production-viable.

### 2. Delete dead `evtest` package

`core/event/internal/evtest/` — 0% coverage, not imported anywhere, internal only. Dead weight.

### 3. Close coverage gaps (Tier 1)

19 small tasks, ~2.5 hours total. Most are <10 minutes each. The aggregate effect gets us to ~99%+ across all modules.

### 4. Add `Codec` interface

Right now event payloads are raw `[]byte`. Users must manually serialize/deserialize. A pluggable `Codec` interface (JSON, protobuf, msgpack) would make the library significantly more usable.

### 5. Add `Projection` interface

The missing "Q" in CQRS. We have command dispatch, event sourcing, and catalog generation — but no read model infrastructure. This is the next major architectural gap.

### 6. Fix `MemorySnapshotStore` deep copy

`Load` returns a shallow copy of `State []byte`. If the caller mutates it, they corrupt the store's internal state. Needs defensive copy.

### 7. Tag stable releases

Users can't `go get` with confidence without semver tags. `core/v1.0.0` should be tagged now — the API is stable.

### 8. Write storage module design doc

Before writing code, document the schema, sqlc config, build tags, and outbox pattern. This prevents rework.

### 9. Add GitHub Pages with go-import tags

Go 1.25 subdirectory module resolution requires `go-import` meta tags. Without them, `go get github.com/larsartmann/go-cqrs-lite/core` won't work.

### 10. Re-enable example modules

Examples were removed in session 9 (81+ LSP false positives). They should be restored as proper CI-tested modules in `go.work`.

---

## f) Top #25 Things We Should Get Done Next

| #   | Task                                                   | Effort | Impact   | Tier |
| --- | ------------------------------------------------------ | ------ | -------- | ---- |
| 1   | **Delete dead `evtest` package**                       | 5min   | HIGH     | 1    |
| 2   | **Test `ensureMetadata` when metadata already exists** | 5min   | MEDIUM   | 1    |
| 3   | **Test `Metadata()` nil metadata path**                | 5min   | MEDIUM   | 1    |
| 4   | **Test `loadEvents` snapshot error branch**            | 8min   | MEDIUM   | 1    |
| 5   | **Test `DefaultRetryConfig.IsRetryable`**              | 5min   | LOW      | 1    |
| 6   | **Test `collectionSchema` uncovered branches**         | 8min   | MEDIUM   | 1    |
| 7   | **Test `goTypeToJSON` uncovered branches**             | 10min  | MEDIUM   | 1    |
| 8   | **Create `storage/` directory + `go.mod`**             | 5min   | CRITICAL | 3    |
| 9   | **Add `storage/` to `go.work`**                        | 2min   | CRITICAL | 3    |
| 10  | **Write PostgreSQL event store schema**                | 10min  | CRITICAL | 3    |
| 11  | **Write sqlc queries (Save, Load, LoadFromVersion)**   | 12min  | CRITICAL | 3    |
| 12  | **Create `sqlc.yaml` config**                          | 10min  | HIGH     | 3    |
| 13  | **Run `sqlc generate`**                                | 5min   | HIGH     | 3    |
| 14  | **Implement `event.Store` SQL adapter**                | 12min  | CRITICAL | 3    |
| 15  | **Implement transactional outbox**                     | 12min  | CRITICAL | 3    |
| 16  | **Add `Codec` interface to core**                      | 10min  | HIGH     | 2    |
| 17  | **Add `Upcaster` interface to core**                   | 10min  | HIGH     | 2    |
| 18  | **Add `Projection` interface to core**                 | 12min  | HIGH     | 2    |
| 19  | **Add `CheckpointStore` interface**                    | 8min   | HIGH     | 2    |
| 20  | **Create `watermill/` directory + `go.mod`**           | 5min   | HIGH     | 4    |
| 21  | **Implement `event.Bus` via Watermill**                | 12min  | HIGH     | 4    |
| 22  | **Fix `MemorySnapshotStore` deep copy**                | 8min   | LOW      | 7    |
| 23  | **Tag `core/v1.0.0`**                                  | 3min   | HIGH     | 8    |
| 24  | **Tag `memory/v1.0.0`**                                | 3min   | HIGH     | 8    |
| 25  | **Write storage module design doc**                    | 10min  | MEDIUM   | 8    |

---

## g) Top #1 Question I Cannot Figure Out Myself

### How should the `storage/` module handle database connection lifecycle?

The `storage/` module needs a PostgreSQL connection (or connection pool). But the `core/event.Store` interface says nothing about connection management:

```go
type Store interface {
    Save(ctx context.Context, aggregateType AggregateType, aggregateID AggregateID, events []Event, expectedVersion Version) error
    Load(ctx context.Context, aggregateType AggregateType, aggregateID AggregateID) ([]Event, error)
    // ...
}
```

**The question:** Should the `storage` module:

1. **Accept `*sql.DB` in the constructor** (caller manages connection lifecycle)?
   - Pros: Flexible, works with existing connection pools, testable with `sqlmock`
   - Cons: Caller must set up `pgx/v5`, connection string, pool config

2. **Accept a connection string and manage its own pool internally**?
   - Pros: Self-contained, simpler API (`NewEventStore("postgres://...")`)
   - Cons: Hidden resource management, harder to test, caller can't share pool with other code

3. **Use an interface like `Querier` from sqlc**?
   - Pros: Testable with any `*sql.DB` or mock, clean separation
   - Cons: Adds a dependency on sqlc-generated types to the public API

**Why this matters:** This decision affects every user of the storage module. Get it wrong and we force awkward workarounds or leaky abstractions. Get it right and the storage module is a drop-in replacement for `memory.NewMemoryStore()`.

**What I've tried:**

- Looking at how `pgx/v5` handles connection pooling — it has `pgxpool.Pool` which is `*sql.DB`-compatible
- Looking at how other Go libraries handle this (sqlc examples use `*sql.DB` in constructors)
- The sqlc-generated code uses `interface { DBTX }` which works with both `*sql.DB` and `*sql.Tx`

**What I need from you:** Guidance on the connection lifecycle strategy. This is a fundamental API design decision that will be hard to change later without breaking users.

---

## Metrics

| Metric                 | Value                                        |
| ---------------------- | -------------------------------------------- |
| Total Go files         | 122                                          |
| Test files             | 56                                           |
| Lines of Go code       | ~17,927                                      |
| Modules                | 4 active (core, memory, catalog, middleware) |
| Test modules           | 1 (testhelpers)                              |
| Zero-coverage packages | 2 (`internal/evtest`, `internal/cattest`)    |
| Lint issues            | **0**                                        |
| Test failures          | **0**                                        |
| Formatting issues      | **0**                                        |

## Coverage Summary

| Package                | Coverage  | Delta vs Session 10 |
| ---------------------- | --------- | ------------------- |
| `core/command`         | 100.0%    | —                   |
| `core/query`           | 100.0%    | —                   |
| `core/pkg/dispatcher`  | 100.0%    | —                   |
| `middleware`           | 99.3%     | —                   |
| `catalog/adapters`     | 98.8%     | —                   |
| `core/event`           | 98.2%     | —                   |
| `memory`               | 98.9%     | —                   |
| `core/pkg/id`          | 97.1%     | —                   |
| `catalog/asyncapi`     | 97.6%     | —                   |
| `catalog/eventcatalog` | 95.5%     | —                   |
| `core/aggregate`       | 94.4%     | —                   |
| `catalog`              | 94.3%     | —                   |
| **Total**              | **85.9%** | —                   |

## Files Changed (This Session)

| File                                                             | Status                                             |
| ---------------------------------------------------------------- | -------------------------------------------------- |
| `go.work.example`                                                | DELETED                                            |
| `core/aggregate/aggregate.go`                                    | Modified (ApplySnapshot, slice reuse)              |
| `core/aggregate/repository.go`                                   | Modified (functional options, loadEvents)          |
| `core/aggregate/aggregate_test.go`                               | Modified (ApplySnapshot no-op)                     |
| `core/aggregate/benchmark_test.go`                               | Modified (ApplySnapshot)                           |
| `core/aggregate/repository_test.go`                              | Modified (ApplySnapshot JSON)                      |
| `core/aggregate/cqrs_bdd_test.go`                                | Modified (ApplySnapshot no-op)                     |
| `core/aggregate/integration_test.go`                             | Modified (ApplySnapshot no-op)                     |
| `core/aggregate/snapshot_test.go`                                | Modified (constructor calls, snapshot assertions)  |
| `core/aggregate/outbox_test.go`                                  | Modified (outbox failure test, failingOutbox mock) |
| `core/pkg/dispatcher/dispatcher.go`                              | Modified (cached middleware at Register)           |
| `core/pkg/dispatcher/base.go`                                    | Modified (Register/Dispatch signatures)            |
| `core/pkg/dispatcher/dispatcher_test.go`                         | Modified (updated callers)                         |
| `core/command/dispatcher.go`                                     | Modified (Register wrap func, grammar fix)         |
| `core/query/dispatcher.go`                                       | Modified (Register wrap func)                      |
| `memory/outbox.go`                                               | Modified (remove acked entries, filter slice)      |
| `memory/outbox_test.go`                                          | Modified (removal test, wsl fixes)                 |
| `catalog/adapters/builder.go`                                    | Modified (AddServiceToDomain returns error)        |
| `catalog/adapters/adapters_dispatcher_test.go`                   | Modified (error assertions)                        |
| `catalog/registry.go`                                            | Modified (godoc on all exported methods)           |
| `docs/planning/2026-04-30_01-20_COMPREHENSIVE_EXECUTION_PLAN.md` | NEW (76 tasks, 8 tiers)                            |

---

_Generated 2026-04-30 01:39 CEST | 10 commits | 0 issues | 0 failures_
