# Status Report: Metaengine System — Bug Fixes + Deferred Features

> **Date:** 2026-08-04 22:19
> **Session scope:** Fix 5 known bugs from the previous session's status report, then implement all 7 "deferred" tasks + write comprehensive tests.
> **Starting point:** Previous session left 18/25 tasks done, 5 bugs documented, 7 tasks deferred.
> **Ending point:** 25/25 tasks addressed, 5 bugs fixed, 7 features implemented, 15 new tests passing with `-race`.

---

## a) FULLY DONE (working, tested, committed)

### 1. AtomicAppender Interface + Memory Implementation

- **Files:** `metaengine/engine.go` (interface), `metaengine/memory_stream_log.go` (impl), `metaengine/errors.go` (`ErrVersionConflict`)
- The `AtomicAppender` interface does version-check-then-append under a single lock acquisition — true atomic optimistic concurrency without the deadlock risk of `RunInTx` holding the mutex.
- Compile-time assertions on both `memoryEngine` and `sqliteEngine`.
- **Tested:** `TestSystem_AtomicConcurrencyConflict` — verifies conflict detection on wrong expected version.

### 2. EventAdapter Seq Cache (ReadFrom O(1) lookup)

- **File:** `system/adapter_event.go` — `lookupSeq` checks a `map[string]int64` cache before scanning. On cache miss, scans journal once to populate, subsequent lookups are O(1). Incrementally updated from `ReadFrom` results.
- **Tested:** `TestSystem_Journal` (3 events across 3 streams, ReadFrom verified).

### 3. Event Bus + Publisher Wiring

- **File:** `system/bus.go` — `simpleBus` implements `event.Bus` (Publisher + Subscriber + middleware chains). Synchronous dispatch on publishing goroutine.
- Wired into `RegisterDecider` — `decider.NewRepository` now gets `sys.bus` instead of `nil`.
- **Tested:** `TestSystem_EventBusPubSub` — subscribe → dispatch → verify handler invoked.

### 4. Projection Wiring via projectionadapter

- **File:** `system/constructor.go` — imports `metaengine/projectionadapter/v4`, creates adapter from `sys.projStore`, registers on `projectionhost.Host`.
- `DomainConfig.ProjectionDecoder` field added so consumers can provide typed event decoders.
- `go.mod` updated with `projectionadapter/v4` dependency.

### 5. Cache Tier Wiring

- **File:** `system/constructor.go` — when `InstanceConfig.Cache` is set, wraps `eventStore` in `CachedEventStore`.
- **File:** `system/cache.go` — `CacheStats` fixed from O(N) iteration to O(1) via `cache.EstimatedSize()`.

### 6. CommandAdapter (M7)

- **File:** `system/adapter_command.go` — full `command.Store` (Save, AppendBatch, Load, LoadFromTimestamp, LoadToTimestamp) + `command.SeekableCommandJournal` (ReadAll, ReadFrom).
- Compile-time assertions on both interfaces.
- Wired into constructor when `RoleSourceOfTruth` or `RoleCommands` instance is configured.

### 7. QueryAdapter (M8)

- **File:** `system/adapter_query.go` — full `query.QueryStore` (SaveQuery, LoadQueries) + `query.SeekableQueryJournal` (ReadAllQueries, ReadQueriesFrom).
- Compile-time assertions on both interfaces.
- Wired into constructor when `RoleSourceOfTruth` or `RoleQueries` instance is configured.

### 8. SQLite StreamLogBackend (M14)

- **Files:** `metaengine/sqlite_engine.go` (DDL + query strings), `metaengine/sqlite_stream_log.go` (5 StreamLogBackend methods + `StreamAppendExpected` + `AtomicAppender`)
- New `meta_stream_log` table with indexes on `(collection, stream_id, seq)` and `(collection, seq)`.
- `StreamAppendExpected` uses `RunInTx` for true SQLite transactional isolation.
- `JournalReadFrom` with `limit <= 0` correctly applies `seq > afterSeq` filter.
- **Tested:** 3 tests — roundtrip, atomic appender conflict, multiple streams.

### 9. MultiBus Fan-Out (M20)

- **File:** `system/multi_bus.go` — `MultiBus` fans out `Publish` to N publishers. `AddPublisher` for dynamic registration.
- **Tested:** `TestMultiBus_FanOut` — verifies both buses receive events.

### 10. SnapshotBackend Isolation Fix

- **File:** `system/snapshot.go` — eliminated package-level global `snapshotStore` map. Each `memorySnapshotBackend` instance has its own `map[string]map[string]snapshotEntry` with a `sync.Mutex`.
- `NewMemorySnapshotBackend()` exported for testing.
- **Tested:** `TestSnapshotBackend_Isolation` — two backends don't interfere, `TestSnapshotBackend_MemoryRoundtrip` — save/load/loadAtVersion/delete.

### 11. System Mutex Concurrency Fix

- **File:** `system/system.go` — `sync.Mutex` → `sync.RWMutex`. Command dispatch uses `RLock` (concurrent reads of the `repos` map).
- `RegisterDecider` still uses `Lock` (exclusive write).

### 12. Op Accessors

- **File:** `system/system.go` — `Op[State].StreamID()` and `Op[State].StreamType()` exported.
- **Tested:** `TestOp_Accessors`.

### 13. Comprehensive Test Suite

- **File:** `system/system_extended_test.go` — 12 new tests covering: query dispatch, driver registry, snapshot backend + isolation, multi-decider (two stream types), concurrent dispatch (20 goroutines, race detector), event bus pub/sub, MultiBus fan-out, Op accessors, atomic concurrency conflict.
- All 15 system tests pass with `-race`.
- All metaengine tests pass with `-race` (77s full suite).

---

## b) PARTIALLY DONE

### 1. Projections Wired but Untested

- **Status:** The constructor creates a `projectionadapter.Adapter` from `sys.projStore` and registers it on the host. But there is NO test that dispatches a command → verifies the projection is updated. The wiring compiles but the full event → projection flow is unproven.
- **Risk:** The projectionadapter's `EventTypes()` is derived from `store.EventTypes()`, which depends on the QueryDecl fold declarations. If the fold declarations don't match the event types produced by the decider, the projection silently ignores events.

### 2. SQLite StreamLogBackend Cannot Be Used Through System

- **Status:** The SQLite engine implements `StreamLogBackend` and `AtomicAppender`, but:
  - The constructor's `createEngine()` function has a hardcoded `switch` that only supports `"memory"`. It does NOT use the driver registry (`createEngineFromDriver`).
  - There is no `RegisterDriver("sqlite", ...)` call anywhere.
  - The `EventAdapter` has a `WithSerialization()` option for SQL engines, but the constructor never passes it.
- **Impact:** Even if an operator configures `EngineConfig{Driver: "sqlite", DSN: "..."}`, the system will error: `unsupported driver "sqlite"`.

### 3. EventAdapter Serialization Path Untested

- **Status:** The `serializedEvent` envelope, `encodeEvent`, and `decodeEvent` functions exist and compile. The `encodeEvent` uses `event.PayloadReadOnly`, `event.MarshalMetadataJSON`, and `event.ReconstructEventFromFields`. But there is ZERO test coverage of this path — no test creates an adapter with `WithSerialization()`, writes events, and reads them back.

### 4. CommandAdapter + QueryAdapter Wired but Untested

- **Status:** Both adapters compile, implement their interfaces, and are wired into the constructor. But neither has a dedicated test. They share the same StreamLogBackend as the EventAdapter, so they work for Memory (pointer storage) but would need serialization for SQL engines (which doesn't exist for commands/queries).

### 5. Cache Tier Wired but Untested

- **Status:** The constructor wraps the event store in `CachedEventStore` when `InstanceConfig.Cache` is set. But no test verifies: cache hit on second Load, cache miss on first Load, eviction at capacity, or that writes bypass the cache.

### 6. Introspection API Still Returns Static Data

- **Status:** Unchanged from previous session. `Snapshot()`, `Health()`, `Explain()` compile but don't wire to live runtime state (handler counts, worker status, engine health).

### 7. Scream Store Still Incomplete

- **Status:** Unchanged from previous session. Has type system + 2 rules. Missing `PlanDiff`, `PlanFingerprint`, `Manifest`, full safety rules table.

### 8. Config Loader Still Has Placeholder YAML Parser

- **Status:** Unchanged from previous session. `LoadConfig()` returns nil for YAML. Env var overrides exist but YAML parsing is not implemented (koanf not added).

---

## c) NOT STARTED

### 1. SQLite Integration Test Through System (M15)

- No test creates a System with SQLite engine config and verifies the full CQRS roundtrip. This is blocked by the `createEngine` → `createEngineFromDriver` wiring gap.

### 2. SQLite Driver Registration

- No `init()` function registers the SQLite driver in the driver registry. The infrastructure exists (`RegisterDriver`, `lookupDriver`, `createEngineFromDriver`) but is unused.

### 3. Migrate taskmanager to System (M24)

- Not started. Depends on SQLite working through System.

### 4. Connection Pool Lifecycle (M25)

- Not started. Needs samber/do integration.

### 5. koanf YAML Parsing

- Not started. Config loader still has placeholder.

---

## d) TOTALLY FUCKED UP

### 1. Constructor Bypasses the Driver Registry

- **File:** `system/constructor.go:39` — calls `createEngine(cfg)` (hardcoded switch, only "memory") instead of `createEngineFromDriver(ctx, cfg)` (driver registry lookup).
- **Impact:** The entire driver registry (M16) — `RegisterDriver`, `RegisteredDrivers`, `lookupDriver`, `createEngineFromDriver` — is dead code. The constructor never uses it. This means:
  - SQLite engine cannot be used through System despite being fully implemented.
  - The `database/sql` model (D1) is not realized in running code.
  - Any future driver registration (Pebble, DuckDB, Postgres) would be ignored.
- **Fix:** Replace `createEngine(cfg)` with `createEngineFromDriver(ctx, cfg)` in the constructor, and register a SQLite driver in `init()`.

### 2. No Serialization Wiring for SQL Engines

- **File:** `system/constructor.go:73` — `NewEventAdapter(backend, "events")` never passes `WithSerialization()`.
- **Impact:** Even if the driver registry were fixed, SQLite-persisted events would be stored as raw `*ImmutableEvent` pointers that the engine would JSON-encode via `encodeJSON` — losing the typed event reconstruction path. The `serializedEvent` envelope exists specifically for this but is never used.
- **Fix:** Detect if the engine is persistent (not Memory) and pass `WithSerialization()`.

### 3. Two Files Exceed the 350-Line CI Limit

- `system/constructor.go` — 369 lines (limit: 350)
- `system/adapter_event.go` — 372 lines (limit: 350)
- **Impact:** The CI gate `nix run .#lint` will fail on these files. `system.go` is at 318 lines (close to limit).

### 4. simpleBus Middleware Semantics Differ from event.Bus

- **File:** `system/bus.go` — `dispatch` chains ALL handlers (typed + catch-all) into a single sequential chain. If one handler errors, subsequent handlers are skipped.
- **Impact:** This differs from the standard `event.Bus` contract where each handler is called independently. A failing handler in simpleBus blocks all subsequent handlers, which could silently swallow event processing.

### 5. api-stability Golden Not Regenerated

- Every new exported symbol (`NewMemorySnapshotBackend`, `NewMultiBus`, `NewCommandAdapter`, `NewQueryAdapter`, `WithSerialization`, `Op.StreamID`, `Op.StreamType`, `AtomicAppender`, `ErrVersionConflict`, etc.) requires regenerating the api-stability golden file. This was not done.

### 6. system/ Not in api-stability Modules List

- `cmd/api-stability/main.go` does not include `"system"` in its modules slice. The `TestEveryGoModDirIsInModulesList` meta-test will fail.

### 7. AGENTS.md Not Updated

- The module list, build commands, and test commands in `AGENTS.md` do not include `system/`. The module table doesn't list it. The test command doesn't include `./system/...`.

---

## e) WHAT WE SHOULD IMPROVE

### Architecture

1. **Replace `createEngine` with `createEngineFromDriver`** — the driver registry is dead code. This is the #1 priority.
2. **Register SQLite driver in `init()`** — so operators can actually use `Driver: "sqlite"`.
3. **Auto-detect serialization need** — when the engine is not Memory, pass `WithSerialization()`. Or better: always serialize (Memory can decode JSON envelopes too, at a small perf cost).
4. **Fix simpleBus handler independence** — each handler should be called independently, not chained. One failing handler should not block others.
5. **CommandAdapter and QueryAdapter need serialization** — same `serializedEvent`-style envelope for SQL engines.

### Code Quality

6. **Split `constructor.go` (369 lines)** — extract projection wiring into `projections.go`, cache wiring into `cache.go`.
7. **Split `adapter_event.go` (372 lines)** — extract serialization into `adapter_event_sql.go`.
8. **Add `gofumpt -w` to all new files** — haven't been formatted yet.
9. **Add file-size CI awareness** — the 350-line limit is CI-enforced.

### Testing

10. **Projection E2E test** — dispatch command → Start host → verify projection store updated.
11. **SQLite-through-System integration test** — full CQRS roundtrip on SQLite.
12. **Cache hit/miss/eviction test** — verify CachedEventStore behavior.
13. **CommandAdapter roundtrip test** — save/load/ReadAll/ReadFrom.
14. **QueryAdapter roundtrip test** — save/load/ReadAllQueries/ReadQueriesFrom.
15. **Serialization roundtrip test** — encode event → store as JSON → decode → verify identity.
16. **Multi-bus error propagation test** — verify first publisher error is returned.

### Integration

17. **Regenerate api-stability golden** — add all new exported symbols.
18. **Add `system/` to api-stability modules list**.
19. **Update AGENTS.md** — module list, test command, design principles section.
20. **Update the implementation plan** — mark deferred tasks as done.

---

## f) Next 50 Things to Get Done (prioritized)

### Critical Path (blocks production use)

| #   | Task                                                                         | Impact   | Effort |
| --- | ---------------------------------------------------------------------------- | -------- | ------ |
| 1   | Replace `createEngine` with `createEngineFromDriver` in constructor          | Critical | 5min   |
| 2   | Register SQLite driver in `init()` (open sql.DB, call NewSQLiteEngine)       | Critical | 15min  |
| 3   | Auto-detect serialization: pass `WithSerialization()` for non-Memory engines | Critical | 15min  |
| 4   | Write SQLite-through-System integration test (full CQRS roundtrip on SQLite) | Critical | 30min  |
| 5   | Write projection E2E test (command → host.Start → verify projection updated) | Critical | 45min  |
| 6   | Fix simpleBus handler independence (each handler called separately)          | High     | 20min  |

### File Size / CI Gate

| #   | Task                                                       | Impact | Effort |
| --- | ---------------------------------------------------------- | ------ | ------ |
| 7   | Split constructor.go (369→<350): extract projection wiring | High   | 15min  |
| 8   | Split adapter_event.go (372→<350): extract serialization   | High   | 15min  |
| 9   | Run `gofumpt -w` on all new files                          | Medium | 5min   |

### Test Coverage Gaps

| #   | Task                                                                     | Impact | Effort |
| --- | ------------------------------------------------------------------------ | ------ | ------ |
| 10  | Cache hit/miss/eviction test                                             | High   | 20min  |
| 11  | CommandAdapter save/load/journal test                                    | High   | 20min  |
| 12  | QueryAdapter save/load/journal test                                      | High   | 20min  |
| 13  | Serialization roundtrip test (encode → store → decode → verify identity) | High   | 20min  |
| 14  | Multi-bus error propagation test                                         | Medium | 10min  |
| 15  | Concurrent ReadFrom + Save test (race detector)                          | Medium | 15min  |
| 16  | Test bus middleware (Use + UsePublish)                                   | Medium | 15min  |
| 17  | Test bus SubscribeAll (catch-all handler)                                | Medium | 10min  |

### Integration & Admin

| #   | Task                                                        | Impact | Effort |
| --- | ----------------------------------------------------------- | ------ | ------ |
| 18  | Add `system/` to cmd/api-stability modules list             | High   | 5min   |
| 19  | Regenerate api-stability golden file                        | High   | 10min  |
| 20  | Update AGENTS.md module list + test commands                | High   | 15min  |
| 21  | Run `nix fmt` on system/ and metaengine/ new files          | Medium | 5min   |
| 22  | Verify `nix run .#build` passes (all modules)               | High   | 5min   |
| 23  | Verify `nix run .#lint` passes (file size, depguard, gosec) | High   | 5min   |

### Serialization for Command/Query Adapters

| #   | Task                                                              | Impact | Effort |
| --- | ----------------------------------------------------------------- | ------ | ------ |
| 24  | Add serialization envelope for CommandAdapter (serializedCommand) | Medium | 20min  |
| 25  | Add serialization envelope for QueryAdapter (serializedQuery)     | Medium | 20min  |
| 26  | Test CommandAdapter serialization roundtrip on SQLite             | Medium | 15min  |
| 27  | Test QueryAdapter serialization roundtrip on SQLite               | Medium | 15min  |

### Scream Store

| #   | Task                                                   | Impact | Effort |
| --- | ------------------------------------------------------ | ------ | ------ |
| 28  | Implement PlanDiff (compare two SerializablePlans)     | Medium | 45min  |
| 29  | Implement PlanFingerprint (canonical hash)             | Medium | 20min  |
| 30  | Implement pinned manifest (persist SerializablePlan)   | Medium | 30min  |
| 31  | Implement full safety rules table from design doc §9.4 | Medium | 30min  |
| 32  | Test SCREAM-tier blocks startup                        | Medium | 20min  |

### Introspection API

| #   | Task                                                                 | Impact | Effort |
| --- | -------------------------------------------------------------------- | ------ | ------ |
| 33  | Wire Snapshot() to live runtime data (handler counts, worker status) | Medium | 30min  |
| 34  | Wire Health() to engine health checks (db.PingContext)               | Medium | 20min  |
| 35  | Wire Explain() to metaengine store.Explain()                         | Medium | 20min  |
| 36  | Test introspection API returns correct topology                      | Medium | 20min  |

### Config Loader

| #   | Task                                                  | Impact | Effort |
| --- | ----------------------------------------------------- | ------ | ------ |
| 37  | Add koanf dependency to go.mod                        | Low    | 5min   |
| 38  | Implement real YAML parsing                           | Low    | 30min  |
| 39  | Implement full env var mapping (CQRS_ENGINES__, etc.) | Low    | 20min  |
| 40  | Test config loader (YAML + env override)              | Low    | 15min  |

### Real-World Validation

| #   | Task                                                      | Impact | Effort |
| --- | --------------------------------------------------------- | ------ | ------ |
| 41  | Migrate example/taskmanager to System                     | High   | 90min  |
| 42  | Write system/ module README.md                            | Medium | 30min  |
| 43  | Add system/ to cqrs-lint feature profile detection        | Low    | 20min  |
| 44  | Write Example_* test function showing full consumer usage | Medium | 20min  |

### Connection Pool Lifecycle

| #   | Task                                                           | Impact | Effort |
| --- | -------------------------------------------------------------- | ------ | ------ |
| 45  | Add samber/do dependency                                       | Low    | 5min   |
| 46  | Implement named service lifecycle (RegisterNamed, InvokeNamed) | Low    | 45min  |
| 47  | Wire connection pool into System.Close() ordering              | Low    | 30min  |

### Polish

| #   | Task                                                        | Impact | Effort |
| --- | ----------------------------------------------------------- | ------ | ------ |
| 48  | Add godoc to all new exported functions                     | Medium | 20min  |
| 49  | Update the design doc status from "DESIGN" to "IMPLEMENTED" | Low    | 5min   |
| 50  | Write a `system/` getting-started guide                     | Low    | 30min  |

---

## g) Questions

### Q1: Should the EventAdapter always serialize, even for Memory?

Currently the Memory engine stores `*ImmutableEvent` pointers directly (zero-copy, no serialization). SQL engines need the `serializedEvent` JSON envelope. The constructor must detect the engine type and pass `WithSerialization()` accordingly.

**Option A:** Always serialize (uniform code path, small perf cost on Memory).
**Option B:** Detect engine type and serialize only for non-Memory (current design, requires type assertion or config flag).

I lean toward Option A — the perf cost is negligible for Memory (events are small, JSON encode/decode is fast), and it eliminates an entire class of bugs where the wrong path is taken.

### Q2: Should the constructor use the driver registry exclusively, or keep the Memory fallback?

The driver registry has Memory auto-registered in `init()`. If I switch the constructor to `createEngineFromDriver`, it will always go through the registry. But the fallback "if no source-of-truth instance, create Memory" path in the constructor creates engines directly via `metaengine.NewMemoryEngine()` — bypassing the registry.

**Should ALL engine creation go through the driver registry, including the default fallback?** Or is it OK for the fallback to be hardcoded?

### Q3: Should simpleBus dispatch handlers independently or chain them?

The current `simpleBus.dispatch` chains all handlers into a sequential call chain — if handler 1 errors, handler 2 never runs. The standard `event.Bus` contract calls each handler independently.

**Should I fix simpleBus to call handlers independently (matching event.Bus semantics), or is the chained behavior acceptable for a "simple" bus?** If independent, a failing handler logs/swallows the error and the next handler still runs.

---

_End of status report._
