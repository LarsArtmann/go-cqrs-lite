# Features

> Honest, verified inventory of what go-cqrs-lite actually does — not what it plans to do.

## Status Legend

| Status                  | Meaning                                                    |
| ----------------------- | ---------------------------------------------------------- |
| ✅ FULLY_FUNCTIONAL     | Tested, production-quality, no known issues                |
| ⚠️ PARTIALLY_FUNCTIONAL | Works for happy paths but has gaps or known bugs           |
| 🔴 BROKEN               | Compiles but has correctness issues                        |
| 🧪 TESTING_ONLY         | Works but is explicitly designed for tests, not production |
| 🧪 EXPERIMENTAL         | New/reactive features, API may change                      |
| 📐 PLANNED              | Mentioned in docs/planning but no code exists              |
| 💡 DEMO                 | Example code, not a reusable module                        |
| 🔧 TOOL                 | Code generation or developer tooling                       |

---

## Core CQRS

### Command Dispatcher ✅ FULLY_FUNCTIONAL

> `import "github.com/larsartmann/go-cqrs-lite/command/v4"`

| Feature                  | Detail                                                                                                                                             | Status |
| ------------------------ | -------------------------------------------------------------------------------------------------------------------------------------------------- | ------ |
| Command dispatch         | `Dispatcher.Dispatch(ctx, cmd)` routes to registered handler                                                                                       | ✅     |
| Handler registration     | `Dispatcher.Register(cmdType, handler)` with duplicate guard                                                                                       | ✅     |
| Middleware chain         | `Dispatcher.Use(middleware...)` — applied at **dispatch time** (can be added before or after Register)                                             | ✅     |
| Lifecycle                | `Dispatcher.Close()` — rejects all ops after close                                                                                                 | ✅     |
| Validation               | `New()` rejects empty type and zero streamID                                                                                                       | ✅     |
| TypedHandler[T]          | `RegisterTyped[T](d, type, handler)` — type-safe handler receiving `T` not `Command`                                                               | ✅     |
| Command metadata         | Own `Metadata` struct (embeds `Tracing`, no longer aliases `event.Metadata` — ADR-0031) with CorrelationID, CausationID, UserID, RequestID, Custom | ✅     |
| Metadata options         | `WithCorrelationID`, `WithCausationID`, `WithUserID`, `WithRequestID`                                                                              | ✅     |
| Persisted command        | `PersistedCommand` struct with ID, Type, StreamRef, ReceivedAt, Payload, Metadata                                                                  | ✅     |
| Command store interfaces | `CommandSink`, `CommandSource`, `Store` (Sink+Source) — persisted command log                                                                      | ✅     |
| CommandJournal           | `ReadAll(ctx)` — global command log ordered by ReceivedAt; audit trail of every command                                                            | ✅     |
| SeekableCommandJournal   | `ReadFrom(ctx, afterCommandID, limit)` — position-based command replay with ULID checkpoints                                                       | ✅     |
| Command Bus              | `Bus`: `Publish`, `Subscribe`, `SubscribeAll`, `Use` — command pub/sub (concrete impls keep `Close()`)                                             | ✅     |
| Publisher / Subscriber   | ISP split: `Publisher.Publish(ctx, cmds...)`, `Subscriber.Subscribe(type, handler)`                                                                | ✅     |
| PublishMiddleware        | `PublishMiddleware` wraps the publish path for cross-cutting concerns (signing, tracing)                                                           | ✅     |

### Query Dispatcher ✅ FULLY_FUNCTIONAL

> `import "github.com/larsartmann/go-cqrs-lite/query/v4"`

| Feature                | Detail                                                                             | Status |
| ---------------------- | ---------------------------------------------------------------------------------- | ------ |
| Query dispatch         | `Dispatcher.Dispatch(ctx, query)` returns `(any, error)`                           | ✅     |
| Typed dispatch         | `DispatchTyped[T](ctx, dispatcher, query)` — generic type-safe result extraction   | ✅     |
| Handler registration   | Same pattern as command — duplicate guard, lifecycle                               | ✅     |
| Middleware chain       | Same pattern as command                                                            | ✅     |
| Pagination             | `Pagination` struct with `Page`, `PageSize`, `Offset()`, `Validate()`              | ✅     |
| Paginated results      | `PaginatedResult[T]` with `HasNext()`, `HasPrev()`, computed `TotalPages`          | ✅     |
| TypedHandler[Q, R]     | `RegisterTyped[Q, R]` — type-safe handler receiving `Q` and returning `(R, error)` | ✅     |
| PersistedQuery         | Stored query with full audit metadata (ID, Type, ReceivedAt, Payload, Metadata)    | ✅     |
| Query store interfaces | `QuerySink`, `QuerySource`, `QueryStore` (Sink+Source) — persisted query log       | ✅     |
| QueryJournal           | `ReadAllQueries(ctx)` — global query log for audit ("who queried what and when?")  | ✅     |
| SeekableQueryJournal   | `ReadQueriesFrom(ctx, afterRequestID, limit)` — position-based query replay        | ✅     |

**Defaults:** Page 1, PageSize 20, max 100.
**Sentinel errors:** `ErrHandlerNotFound`, `ErrDispatcherClosed`, `ErrEmptyQueryType`, `ErrTypeAssertion`

---

### Event System ✅ FULLY_FUNCTIONAL

> `import "github.com/larsartmann/go-cqrs-lite/event/v4"`

| Feature               | Detail                                                                                                                                                                                                                                                                                                                                      | Status |
| --------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ------ |
| Event creation        | `NewEvent()` with auto-generated `EventID` (ULID) and `time.Now()` timestamp                                                                                                                                                                                                                                                                | ✅     |
| Auto-marshal creation | `New()` — creates event from `any` payload (auto-json for structs/maps)                                                                                                                                                                                                                                                                     | ✅     |
| Batch creation        | `NewEvents()` — batch event creation with auto-incrementing versions                                                                                                                                                                                                                                                                        | ✅     |
| 20 functional options | `WithEventID`, `WithOccurredAt`, `WithMetadata`, `WithCorrelationID`, `WithCausationID`, `WithUserID`, `WithRequestID`, `WithSource`, `WithIPAddress`, `WithUserAgent`, `WithCustom`, `WithSchemaVersion`, `WithCausation`, `WithEncoding`, `WithCodec`, `WithClock`, `WithClientID`, `WithClientOccurredAt`, `WithDeadline`, `FromContext` | ✅     |
| Metadata              | `Metadata` struct: embeds `Tracing` (CorrelationID, CausationID, UserID, RequestID), `Source`, `IPAddress`, `UserAgent`, `Tombstone` (*TombstoneMark), `Causation` (*Causation), `Custom` (map[MetadataKey]string) — ADR-0031 typed split                                                                                                   | ✅     |
| MetadataKey           | Typed custom metadata key (`MetadataKey` string type) with constants: `ClientID`, `ClientOccurredAt`, `CommandType`, `CommandID` — replaces raw string keys                                                                                                                                                                                 | ✅     |
| Metadata.Merge        | `Metadata.Merge(other)` overlays non-zero fields from other onto a copy — used by `WithMetadata` and cross-module metadata composition                                                                                                                                                                                                      | ✅     |
| Tracing struct        | Embedded `Tracing{CorrelationID, CausationID, UserID, RequestID}` — shared by event, command, and query Metadata (ADR-0031)                                                                                                                                                                                                                 | ✅     |
| Context enricher      | `ContextEnricher` extracts options from `context.Context`; `CompositeEnricher` chains multiple                                                                                                                                                                                                                                              | ✅     |
| Defensive copies      | `Payload()` and `Metadata()` return copies — callers can't mutate internals                                                                                                                                                                                                                                                                 | ✅     |
| Event.Clone()         | Deep copy of `ImmutableEvent`                                                                                                                                                                                                                                                                                                               | ✅     |
| Typed values          | `Source`, `IPAddress`, `UserAgent`, `Version`, `SchemaVersion` — all parsed and validated                                                                                                                                                                                                                                                   | ✅     |
| Version arithmetic    | `Version.Add`, `Sub`, `Mod`, `Cmp`, `IsPositive` — phantom type math                                                                                                                                                                                                                                                                        | ✅     |
| Event Bus interface   | `Bus`: `Publish`, `Subscribe`, `SubscribeAll`, `Use`, `UsePublish` (concrete impls keep `Close()` — ADR-0010)                                                                                                                                                                                                                               | ✅     |
| PublishMiddleware     | `Bus.UsePublish(mw)` — middleware for publish path                                                                                                                                                                                                                                                                                          | ✅     |
| PublisherFunc adapter | `PublisherFunc` — function adapter for `Publisher`                                                                                                                                                                                                                                                                                          | ✅     |
| Event Store interface | `Store = EventSink + EventSource`: `Save` (optimistic concurrency), `AppendBatch`, `Load`, `LoadFromVersion`, `LoadToVersion`, `LoadToTimestamp` (concrete impls keep `Close()` — ADR-0010)                                                                                                                                                 | ✅     |
| ISP split             | `EventSink` (write) + `EventSource` (read) — fine-grained dependency injection                                                                                                                                                                                                                                                              | ✅     |
| Journal               | `ReadAll()` returns all events ordered by `occurred_at ASC` — for projection replay                                                                                                                                                                                                                                                         | ✅     |
| SeekableJournal       | `ReadFrom(ctx, afterEventID, limit)` — efficient projection catch-up                                                                                                                                                                                                                                                                        | ✅     |
| BackwardsSource       | `LoadBackwards(ctx, streamRef)` — loads events in reverse version order                                                                                                                                                                                                                                                                     | ✅     |
| TombstoneStatus       | `Active`, `Tombstoned`, `Undetermined` — tri-state enum for soft-delete; `DetectTombstone`, `MarkTombstone`, `MarkRebirth`                                                                                                                                                                                                                  | ✅     |
| Time-travel queries   | `LoadToVersion` and `LoadToTimestamp` — read stream state at a point in time                                                                                                                                                                                                                                                                | ✅     |
| Projection interface  | `Projection`: `Name`, `Handle(ctx, Event)`, `EventTypes()` (nil = all)                                                                                                                                                                                                                                                                      | ✅     |
| Context replay marker | `WithProcessingMode(ctx, ModeReplay)` — marks context as replay; handlers can distinguish                                                                                                                                                                                                                                                   | ✅     |
| DecodePayload[T]      | `DecodePayload[T](evt, codec)` — type-safe payload deserialization                                                                                                                                                                                                                                                                          | ✅     |
| DecodePayloads[T]     | `DecodePayloads[T](events, codec)` — batch payload deserialization                                                                                                                                                                                                                                                                          | ✅     |
| PayloadReadOnly       | Zero-copy read access for internal paths (signing, pebble, storage, middleware)                                                                                                                                                                                                                                                             | ✅     |
| Stream loading        | `StreamingSource`/`StreamingJournal` — cursor-based event reads without materializing full slices (SQL, Pebble, Memory backends)                                                                                                                                                                                                            | ✅     |
| Slice helpers         | `SliceFromVersion`, `SliceToVersion`, `FilterByTimestamp` — in-memory event slicing                                                                                                                                                                                                                                                         | ✅     |
| Command causality     | `WithCommandCausality(ctx, type, id)` + `CommandCausalityEnricher` — auto-tag events with the command that caused them                                                                                                                                                                                                                      | ✅     |
| Checkpoint            | `Checkpoint` struct + `CheckpointSink/Source/Store` interfaces for projection positioning                                                                                                                                                                                                                                                   | ✅     |
| Clock injection       | `Clock` type + `WithClock` option for deterministic testing                                                                                                                                                                                                                                                                                 | ✅     |
| Error taxonomy        | 6-family: Rejection / Conflict / Transient / Infrastructure / Orchestration / Corruption; 14 helper funcs (`New*`, `Wrap*`, `Wrap*f`, `Classify`, `IsRetryable`); 16 sentinel errors                                                                                                                                                        | ✅     |
| Event reconstruction  | `ReconstructEventFromFields` — shared deserialization for all store implementations                                                                                                                                                                                                                                                         | ✅     |
| JSON metadata         | `MarshalMetadataJSON`, `UnmarshalMetadataJSON` — DB-safe metadata serialization                                                                                                                                                                                                                                                             | ✅     |

### Decider (Pure-Function Event Sourcing) ✅ FULLY_FUNCTIONAL

> `import "github.com/larsartmann/go-cqrs-lite/decider/v4"`

| Feature              | Detail                                                                                                 | Status |
| -------------------- | ------------------------------------------------------------------------------------------------------ | ------ |
| Decider[State]       | `{Initial State; Fold func(State, Event) (State, error)}` — pure-function event-sourcing pattern       | ✅     |
| Repository[State]    | `NewRepository[State](store, publisher, decider, opts...)` — manages stream lifecycle                  | ✅     |
| Execute              | `Repository.Execute(ctx, streamID, streamType, decide)` — load → decide → save → publish               | ✅     |
| Load                 | `Repository.Load(ctx, streamID, streamType)` — returns `(State, Version, error)`                       | ✅     |
| LoadAtVersion        | `Repository.LoadAtVersion(ctx, streamID, streamType, maxVersion)` — time-travel to version             | ✅     |
| LoadAtTime           | `Repository.LoadAtTime(ctx, streamID, streamType, maxTime)` — time-travel to timestamp                 | ✅     |
| Snapshot integration | `WithSnapshotStore` + `WithSnapshotStrategy` + `WithCodec` — automatic snapshot optimization           | ✅     |
| Hot-state cache      | `WithStateCache` + `NewStateCache[State](capacity)` — LRU cache, incremental loads (7.4x faster)       | ✅     |
| Load coalescing      | `WithLoadCoalescing` — singleflight dedup of concurrent Loads for same stream                          | ✅     |
| Context enrichment   | `WithEnricher` — injects metadata from context into events                                             | ✅     |
| OTel tracing         | OpenTelemetry spans for load/save/execute operations (opt-in)                                          | ✅     |
| WaitForVersion       | `WaitForVersion(ctx, store, streamID, version, opts)` — read-your-writes helper, polls LoadFromVersion | ✅     |

**Sentinel errors:** `ErrNilStore`, `ErrNilPublisher`, `ErrNilFold`, `ErrLoadFailed`, `ErrFoldFailed`, `ErrSaveFailed`, `ErrIncompleteSnapshotConfig`

### Branded IDs ✅ FULLY_FUNCTIONAL

> `import "github.com/larsartmann/go-cqrs-lite/id/v4"`

| Feature              | Detail                                                                                                                                                     | Status |
| -------------------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------- | ------ |
| Generic branded type | `id.Of[T]` — phantom type parameter for compile-time safety                                                                                                | ✅     |
| ULID-backed          | Binary-sortable, time-ordered, 16-byte binary form                                                                                                         | ✅     |
| 8 built-in types     | `StreamID`, `EventID`, `CorrelationID`, `CausationID`, `RequestID`, `UserID`, `ClientID`, `CommandID`                                                      | ✅     |
| Custom branded types | `type OrderID = id.Of[OrderMarker]` — users can create their own                                                                                           | ✅     |
| Exported markers     | All 8 phantom markers exported (`Stream`, `User`, `Correlation`, `Request`, `Causation`, `Client`, `Command`, `Event`) for downstream `BrandNamer` tooling | ✅     |
| All serialization    | JSON (incl. `null`), binary, text, SQL `Scan`/`Value`                                                                                                      | ✅     |
| Convenience funcs    | `New[T]()`, `Parse[T]()`, `ULID[T]()`, `FromPtr[T]()`, `CompareIDs[T]()`                                                                                   | ✅     |
| StreamID derivation  | `DeriveStreamID()` — deterministic ID from namespace + key                                                                                                 | ✅     |
| Timestamp extraction | `ULID(id)` extracts embedded timestamp                                                                                                                     | ✅     |

### Generic Dispatcher ✅ FULLY_FUNCTIONAL

> `import "github.com/larsartmann/go-cqrs-lite/dispatcher/v4"`

| Feature             | Detail                                                         | Status |
| ------------------- | -------------------------------------------------------------- | ------ |
| Generic Dispatcher  | `Dispatcher[H, M]` — type-safe handler + middleware dispatcher | ✅     |
| LifecycleMixin      | `Lifecycle` — `Close()` prevents all ops; thread-safe          | ✅     |
| Middleware ordering | Middleware applied at dispatch time (any order vs Register)    | ✅     |
| Duplicate guard     | `ErrHandlerAlreadyRegistered` — prevents double-registration   | ✅     |

### Idempotency ✅ FULLY_FUNCTIONAL

> `import "github.com/larsartmann/go-cqrs-lite/idempotency/v4"`

| Feature         | Detail                                                                                                          | Status |
| --------------- | --------------------------------------------------------------------------------------------------------------- | ------ |
| Store interface | `Store`: `Seen`, `Record`, `CheckAndRecord` — dedup opaque keys (command idempotency keys)                      | ✅     |
| MemoryStore     | `MemoryStore` — in-memory TTL store with background sweep + lazy deletion                                       | ✅     |
| Atomic dedup    | `CheckAndRecord` — single-lock check+record prevents the TOCTOU race (exactly one winner)                       | ✅     |
| TTL expiration  | Keys expire after a configurable duration; removed by sweeper and lazily on read                                | ✅     |
| ErrDuplicate    | Conflict sentinel returned when a key is already recorded (maps to HTTP 409)                                    | ✅     |
| Generic factory | `middleware.NewIdempotency[M]` — generic dedup factory for any message type                                     | ✅     |
| Command dedup   | `middleware.CommandIdempotency(store, ttl, keyFn)` — defaults to `cmd.ID().String()` when keyFn is nil          | ✅     |
| Event dedup     | `middleware.EventIdempotency(store, ttl, keyFn)` — defaults to `evt.ID().String()` when keyFn is nil            | ✅     |
| Query dedup     | `middleware.QueryIdempotency(store, ttl, keyFn)` — panics if keyFn is nil (queries have no built-in identity)   | ✅     |
| KVStore         | `idempotency/kvstore` — persistent dedup backed by any `kv.Store`                                               | ✅     |
| SQLStore        | `idempotency/sqlstore` — `NewSQLiteStore` / `NewPostgresStore` with `INSERT ON CONFLICT DO NOTHING` + TTL sweep | ✅     |

**Sentinel errors:** `ErrDuplicate` (Conflict)

### Dead-Letter Queue ✅ FULLY_FUNCTIONAL

> Dispatch-side: `import "github.com/larsartmann/go-cqrs-lite/middleware/v4"`
> Projection-side: `import "github.com/larsartmann/go-cqrs-lite/projectionhost/v4"`

Two intentionally separate dead-letter systems (ADR-0043): dispatch-side captures
commands/events/queries that exhausted retries in the middleware pipeline;
projection-side captures events that poisoned a projection handler.

| Feature                              | Detail                                                                                                             | Status |
| ------------------------------------ | ------------------------------------------------------------------------------------------------------------------ | ------ |
| **Dispatch-side (middleware)**       |                                                                                                                    |
| DeadLetterHandler                    | `DeadLetterHandler` func type — wired via `RetryConfig.OnDeadLetter`                                               | ✅     |
| MemoryDeadLetterStore                | In-memory store: `Handle`, `Entries`, `Count`, `Clear`                                                             | ✅     |
| SQLDeadLetterStore                   | SQL-backed (Postgres + SQLite): `Handle`, `Entries`, `Count`, `Clear` with auto-migrating schema                   | ✅     |
| DeadLetterEntry                      | Captures Kind, Type, StreamID, Error, ErrorCode, ErrorFamily, Attempts, FailedAt                                   | ✅     |
| **Projection-side (projectionhost)** |                                                                                                                    |
| DeadLetterStore                      | `DeadLetterStore` interface — `Store`, `List`, `Delete`, `Purge`                                                   | ✅     |
| DeadLetterStoreAdmin                 | Optional interface: `Count`, `ListPaged`, `PurgeBefore` — production management (pagination, time-bounded cleanup) | ✅     |
| MemoryDeadLetterStore                | In-memory `DeadLetterStore` + `DeadLetterStoreAdmin` for dev/test                                                  | ✅     |
| SQLiteDeadLetterStore                | SQLite-backed `DeadLetterStore` + `DeadLetterStoreAdmin` — persists across restarts                                | ✅     |
| Poison capture                       | Exceeded retry threshold → entry stored, checkpoint advances (no stream blockage)                                  | ✅     |

---

## Metaengine 🧪 EXPERIMENTAL

> `import "github.com/larsartmann/go-cqrs-lite/metaengine/v4"`

Cost-based storage planner for event-sourced data. Derives projections, indexes,
and engine assignments from two primitives: **Events** (mutations) and
**Queries** (read intent). The fold return type IS the ADT declaration — the
developer never declares "I need a Map" or "I need a Counter."

| Feature                     | Detail                                                                                                          | Status |
| --------------------------- | --------------------------------------------------------------------------------------------------------------- | ------ |
| Fold sealed interface       | 12 concrete unexported fold types replace 11-field `any` god-struct. Zero nil-panic risk (ADR-0081)             | 🧪     |
| Unified `On[E]()` fold      | Reflection-based handler classification: 12 fold types (insert, update, set, count, edge, remove, skip, etc.)   | 🧪     |
| ADT inference               | 10 ADTs: Map, Set, Counter, Graph, SortedMap, Multimap, Log, Scan, Vector, Search, Spatial (ADR-0085)           | 🧪     |
| Typed FilterOn / SortOn     | `FilterOn(func(r R) T { ... })` — typed closures, no field name strings                                         | 🧪     |
| Pagination from input       | Detected from domain input struct fields (`Limit int`, `After *Cursor`)                                         | 🧪     |
| Cursor serialization        | Base64-encoded URL-safe cursors for HTTP transport                                                              | 🧪     |
| Cost model                  | `CostEstimate` with Volume-based estimation, `LatencyBudget` enforcement, scale threshold tables                | 🧪     |
| Write amplification budget  | Tracks events exceeding write amplification limit                                                               | 🧪     |
| MemoryEngine                | In-memory backend implementing all backend interfaces including Vector/Search/Spatial                           | 🧪     |
| SQLite engine               | `SQLiteEngine` wrapping `storage/view.SQLViewStore` — first production backend (ADR-0061)                       | 🧪     |
| Pebble engine               | `metaengine/pebbleengine` — LSM point reads (~7x faster than SQLite); separate module (ADR-0074)                | 🧪     |
| DuckDB engine               | `metaengine/duckdbengine` — MapBackend, CounterBackend, PushdownScan; CGo required (ADR-0086)                   | 🧪     |
| Postgres engine             | `metaengine/pgengine` — MapBackend, CounterBackend, ScanBackend, PushdownScan (JSONB), LayoutPlanner (ADR-0087) | 🧪     |
| Planner                     | Cost-based optimizer: assigns engines to queries, produces `PlanResult` with diagnostics                        | 🧪     |
| Rule pipeline               | `PlanRule` interface + `RulePipeline`. 4 composable rules: schemaRule, layoutRule, writeAmpRule (ADR-0083)      | 🧪     |
| Materialize-vs-replay       | `ReplayCost`/`MaterializeCost`/`ShouldMaterialize` — ES-specific cost formula for projection decisions          | 🧪     |
| StorageLayout + cost matrix | `Layout{Row, Columnar, LSM, KV}`, `(ADT × Layout) → Complexity` mapping, `EngineProfile.Layouts`, `RuleTrace`   | 🧪     |
| SerializablePlan            | JSON-serializable `PlanResult` for diff/pin/round-trip testing                                                  | 🧪     |
| VersionedStorage            | Temporal queries (`ExecuteAsOf`) on Memory engine. Version chains + binary search                               | 🧪     |
| Store                       | `Plan(engines, queries...)` returns `*Store` for Apply/Execute; `ApplyEncoded` for JSON payloads                | 🧪     |
| Collection results          | Reconstructs typed result collections by field shape from scan output                                           | 🧪     |
| ScanResult explicit HasMore | `ScanResult{Items []any; HasMore bool}` — explicit contract across all 5 engines                                | 🧪     |
| Context.Context             | All backend interfaces accept `context.Context`                                                                 | 🧪     |
| Compile-time assertions     | Interface conformance verified at compile time for all backends                                                 | 🧪     |
| Zero dependencies           | Core `metaengine/v4` has zero production deps; adapter module is separate                                       | 🧪     |
| Projection adapter          | `metaengine/projectionadapter` implements `projection.Projection` for `projectionhost.Host` (ADR-0062)          | 🧪     |
| Cost calibration            | `EngineProfile.NsPerRead`/`NsPerWrite` — split read vs write cost (Memory=500ns, SQLite=7000ns)                 | 🧪     |
| Store.EventTypes()          | Returns sorted unique event types from registered queries — enables adapter event routing                       | 🧪     |
| `ExecuteTyped[Q,R]`         | Cross-engine JSON reification: a query runs on any engine, results reified via JSON round-trip (ADR-0066)       | 🧪     |
| Tx-atomic MapUpdate         | SQLite `MapUpdate` wraps read-modify-write in one tx — no lost updates across concurrent calls (ADR-0067)       | 🧪     |
| Multimap seq-seed           | Lazy `sync.Once` seeding from `MAX(seq)` on first use — safe restart without sequence collisions (ADR-0068)     | 🧪     |
| Fold-classify               | `classifyFold` inspects fold return types to assign ADT patterns — shared across engines for consistency        | 🧪     |
| Cross-engine meta-test      | 150 specs run identical Apply → ExecuteTyped sequences on Memory + SQLite, asserting identical typed results    | 🧪     |
| End-to-end verification     | Signature + ciphertext verification integrated across Memory and SQLite engines                                 | 🧪     |
| SQL pushdown                | `PushdownScan` + `FilterOnField`/`SortOnField` push WHERE/ORDER BY/LIMIT into SQL (SQLite/Pg/DuckDB)            | 🧪     |
| Layout planning             | `LayoutPlan`/`BuildLayoutPlanFromType[R]` generate indexed-column DDL for declared query fields (ADR-0073)      | 🧪     |
| Pebble LayoutPlanner        | Secondary index with O(matches) prefix scan (108x speedup). Range filters via index bounds                      | 🧪     |
| Pebble sort index           | `'o'` prefix key structure for sort fields. 1,233x speedup (8,145µs → 6.6µs)                                    | 🧪     |
| Raw value readers           | `RawValueReader`/`RawScanReader` skip JSON decode for filter/sort/cursor paths (single-pass decode)             | 🧪     |
| SSE event delivery          | `ServeSSE` with Last-Event-ID reconnection, backpressure, `dedup.Ring`, byte-budgeted replay                    | 🧪     |
| PrefetchCache               | Cursor-encoded auto-population cache for paginated reads; thread-safe (`sync.RWMutex`)                          | 🧪     |
| Watcher                     | Reactive change notifications with per-key filtering                                                            | 🧪     |
| Transaction API             | Fully threaded `*sql.Tx` through engine operations (atomic multi-collection updates)                            | 🧪     |
| ADT test harness            | `adttest.RunMatrix` — cross-engine parity tests for all 10 ADTs. Reflect-based capability auto-detect           | 🧪     |
| Property-based parity       | `pgregory.net/rapid` generates random op sequences, verifies Memory and SQLite agree on every operation         | 🧪     |
| Aggregate pushdown          | `AggregateReader` interface — SQL COUNT/SUM/MIN/MAX/AVG pushdown via the engine                                 | 🧪     |
| Error sentinels             | Exported `ErrNotFound`, `ErrAmbiguousKey`, `ErrUnsupportedADT`, `ErrLayoutConflict` wired into execution paths  | 🧪     |
| `OnTyped(eventType, ...)`   | Bind a fold to an explicit CQRS event-type string (decouples from the Go struct name)                           | 🧪     |
| Enum validation             | All 6 enum families (ADT, StorageLayout, FilterOp, CursorKind, etc.) have `Valid()` + registries                | 🧪     |
| Store composition           | Store decomposed from 17→13 fields: `poisonTracker`, `idempotencyTracker`, `workloadMeter`, `subscriberHub`     | 🧪     |
| Vector ADT                  | k-NN similarity search (cosine/euclidean/dot). Memory-only (brute-force). ADR-0085                              | 🧪     |
| Search ADT                  | Full-text search (TF-IDF inverted index). Memory-only (brute-force). ADR-0085                                   | 🧪     |
| Spatial ADT                 | Geo range queries (haversine distance). Memory-only (brute-force). ADR-0085                                     | 🧪     |

**Coverage:** 86.1% (verified `go test -cover ./...` 2026-07-27). 174 BDD specs + 150 cross-engine
meta specs + 12 ADT harness self-tests. The metaengine went through 10+ hardening
sessions (2026-07-30 to 2026-08-02): transaction API fix, SQL injection fix,
hooks-on-error, ReadCoalescer wiring, Watcher with per-key filtering,
PrefetchCache with cursor-encoded auto-population, SSE adapter with
Last-Event-ID reconnection, ContractSuite expanded to all 10 ADTs, Pebble
LayoutPlanner (108x speedup), Pebble sort index (1,233x speedup),
RawValueReader/RawScanReader (single-pass decode), rule pipeline extraction,
materialize-vs-replay cost model, StorageLayout + cost matrix, SerializablePlan,
VersionedStorage temporal queries, Fold sealed interface refactor, 5-engine
cross-engine parity, Vector/Search/Spatial ADTs, pgengine + duckdbengine.
API surface: 3162 exports.

Remaining: wire dead code from data model refactor (branded units, ApplyError),
DuckDB LayoutPlanner, Postgres GIN indexes, SSE consolidation ADR, Vector/
Search/Spatial engine backends. See [TODO_LIST.md](TODO_LIST.md).

---

## Benchmarking Toolkit 🧪 EXPERIMENTAL

> `import "github.com/larsartmann/go-cqrs-lite/benchkit/v4"`

Factory-driven benchmarking suite for measuring CQRS performance across
backends, deployment sizes, and workload profiles. Mirrors the contracttest
pattern: same workload, any backend, structured metrics report.

| Feature                 | Detail                                                                                                         | Status |
| ----------------------- | -------------------------------------------------------------------------------------------------------------- | ------ |
| Core types              | `Config`, `Result`, `LatencyStats`, `ResourceStats`, `DiskStats`, `Factory`, `Environment`                     | 🧪     |
| LatencyCollector        | Sorted-slice + reservoir sampling (10K cap), thread-safe                                                       | 🧪     |
| Resource sampling       | Peak heap via 100ms polling goroutine, baseline/after deltas                                                   | 🧪     |
| Synthetic generator     | Seeded PCG, deterministic, configurable payload size, codec-aware padding                                      | 🧪     |
| Mixed payload sizes     | `NewMixedGenerator(seed, sizes, codec)` — uniform-random per-event sizing                                      | 🧪     |
| 7 named profiles        | Dev, Small, Medium, Large, Stress, WriteHeavy, ReadHeavy                                                       | 🧪     |
| 9-phase runner          | setup → warmup → write → read → readmodel → projection → durability → rawsink → teardown                       | 🧪     |
| Raw sink phase          | Pre-built events timed against `EventSink.Save` only — isolates pure backend write capacity                    | 🧪     |
| Environment metadata    | `GoVersion`, `NumCPU`, `GOMAXPROCS`, `GOOS`, `GOARCH` recorded in every `Result`                               | 🧪     |
| Schema versioning       | `Result.SchemaVersion` for JSON schema stability tracking                                                      | 🧪     |
| Median fix              | `runRepeated` sorts results by throughput before picking median (was insertion-order bug)                      | 🧪     |
| Concurrent workers      | Channel-based, cancel-on-error, WaitGroup                                                                      | 🧪     |
| `Run()` API             | Single-backend benchmark, returns `*Result`                                                                    | 🧪     |
| `Compare()` API         | Multi-backend comparison, handles factory failures gracefully                                                  | 🧪     |
| DiskSizer               | `Bundle.DiskSize()` via `stack.WithDiskSize()`, implemented by Pebble preset                                   | 🧪     |
| CPU measurement         | `syscall.Getrusage` (Unix), stub on non-Unix — microsecond resolution                                          | 🧪     |
| Projection phase        | Polls until all events processed, reports lag + events                                                         | 🧪     |
| Reports                 | Text, JSON (v2), Markdown, benchstat, manifest — latency percentiles, throughput, memory, disk, env            | 🧪     |
| Scaling sweeps          | `WorkerSweep`, `BatchSizeSweep`, `StreamLengthSweep`, `GOMAXPROCSSweep` — systematic parameter exploration     | 🧪     |
| benchstat output        | `WriteBenchstat` — benchstat-compatible lines for statistical comparison                                       | 🧪     |
| Suite manifest          | `WriteManifest` — config + environment + result as JSON for reproducibility                                    | 🧪     |
| JSON schema check       | `ExpectedJSONFields` + `VerifyJSONFields` — guards against silent schema changes                               | 🧪     |
| ReadRatio               | Configurable read/write mix for WriteHeavy and ReadHeavy profiles                                              | 🧪     |
| Durability phase        | `Config.Recovery` — close bundle, reopen via factory, reload streams (`RecoveryTime`, `RecoveredEvents`)       | 🧪     |
| Replay phase            | `Config.ReplayOnly` — skip writes, discover streams from journal, benchmark reads + projections                | 🧪     |
| `benchtest.RunSuite`    | `RunSuite(b, config, factory)` wraps benchkit into Go `testing.B` (`b.ReportMetric`); wired into `stack/bench` | 🧪     |
| Analytical profile      | `ProfileAnalytical` (10K streams, 90% reads, 5x journal scans) + `Profile.JournalScans`                        | 🧪     |
| Postgres backend        | `postgres` backend in `cqrs-bench`; benchkit tests skip without `POSTGRES_TEST_DSN`                            | 🧪     |
| kv projection handler   | Projection phase exercises a real `kv.Store` (Get+Set per event); atomic counter fallback                      | 🧪     |
| Statistical reliability | `RepeatStdDev`/`RepeatCoV`/`RepeatMean`/`RepeatIsReliable` — cross-run variance (ADR-0090)                     | 🧪     |
| GC pause metrics        | `GCMaxPause` — maximum GC pause during benchmark run                                                           | 🧪     |
| Allocation metrics      | `AllocsPerOp`, `BytesPerOp` — derived per-operation allocation tracking                                        | 🧪     |
| Data integrity          | `IntegrityErrors` — verifies event round-trip after benchmark run                                              | 🧪     |
| Write amplification     | `Disk.WriteAmplification` — ratio of bytes written to logical payload size                                     | 🧪     |
| Cold/warm read          | `ColdReadLatency` — first-read latency (no cache) vs steady-state                                              | 🧪     |
| Tail ratio              | `TailRatio` (P99/P50) — latency distribution tail metric                                                       | 🧪     |
| Environment enrichment  | `CPUModel`, `TotalRAMBytes` — hardware metadata for reproducibility                                            | 🧪     |
| Soak test drift         | `SoakResult.GCMaxPauseDriftPct`, `AllocGrowthPct` — memory boundedness over sustained load                     | 🧪     |
| Metaengine benchmark    | Memory + SQLite engines. Counter + Map ADTs. Correctness assertions prevent empty-store silent failure         | 🧪     |
| Mixed workload          | `BenchmarkMixedWorkload_ReadsDuringWrites` — concurrent read/write contention profiling                        | 🧪     |

**Coverage:** 88 benchkit + 12 CLI test functions (`-race`). Includes raw sink phase,
scaling sweeps, benchstat output, suite manifest, schema verification, environment
metadata, schema versioning, durability/recovery, replay, `benchtest.RunSuite`,
analytical profile, Postgres backend, median selection tests, evidence-grade
metrics (GC pause, write amplification, tail ratio, allocation tracking), soak
test drift, metaengine benchmark (Memory + SQLite), and mixed workload phase.
Run-to-run variance is ~20-25% on the memory backend (use `--repeat N` for median reporting).
See [backend comparison](docs/benchmarks/2026-07-31_backend-comparison.md)
and [evidence metrics ADR](docs/adr/0090-benchkit-evidence-metrics.md).

### cqrs-bench CLI 🔧

> `go run github.com/larsartmann/go-cqrs-lite/cmd/cqrs-bench`

| Feature   | Detail                                                                     | Status |
| --------- | -------------------------------------------------------------------------- | ------ |
| `run`     | Benchmark a single backend with a named workload profile                   | 🔧     |
| `compare` | Compare multiple backends side-by-side                                     | 🔧     |
| `sweep`   | Scaling sweep: vary workers, batch size, stream length, or GOMAXPROCS      | 🔧     |
| Profiles  | `--profile {dev\|small\|medium\|large\|stress\|writeheavy\|readheavy}`     | 🔧     |
| Output    | `--format {text\|json\|markdown\|benchstat\|manifest}`                     | 🔧     |
| Codec     | `--codec {json\|cbor}`                                                     | 🔧     |
| Payload   | `--payload-size N` or `--payload-sizes 64,256,4096` (mixed)                | 🔧     |
| Warmup    | `--warmup N`                                                               | 🔧     |
| Repeat    | `--repeat N` — median of N runs with min/max spread (sorted by throughput) | 🔧     |
| Raw sink  | `--skip-raw-sink` — skip prebuilt-event Save-only phase                    | 🔧     |
| Profiling | `--cpuprofile file` and `--memprofile file` — pprof output                 | 🔧     |
| Version   | `--version` via `runtime/debug.ReadBuildInfo()`                            | 🔧     |

---

## Flight Recorder 🧪 EXPERIMENTAL

> `import "github.com/larsartmann/go-cqrs-lite/flightrecorder/v4"`

Go 1.25 `runtime/trace` capture on slow/error/always triggers. Zero-dependency
module (stdlib only). One active recorder per process (ADR-0089).

| Feature             | Detail                                                                                          | Status |
| ------------------- | ----------------------------------------------------------------------------------------------- | ------ |
| `Recorder`          | `New`/`Start`/`Stop`/`Enabled`/`Snapshot`/`SnapshotToFile`/`SnapshotIf`/`Reset`. Once-semantics | 🧪     |
| Trigger functions   | `OnLatency(d)`, `OnError()`, `OnErrorOrLatency(d)`, `OnAlways()`, `OnAny(...)`, `OnAll(...)`    | 🧪     |
| Options             | `WithMinAge`, `WithMaxBytes`, `WithWriter`, `WithFile`                                          | 🧪     |
| Thread-safe         | `sync.Mutex` — safe for concurrent trigger checks                                               | 🧪     |
| `io.Closer`         | `Recorder.Close()` stops recording AND closes file writers                                      | 🧪     |
| `ErrAlreadyEnabled` | Only 1 active recorder per process (double `Start` returns error)                               | 🧪     |
| Command middleware  | `middleware.CommandFlightRecorder(recorder, trigger)` — captures on slow/error dispatch         | 🧪     |
| Event middleware    | `middleware.EventFlightRecorder(recorder, trigger)` — captures on slow/error publish            | 🧪     |
| Query middleware    | `middleware.QueryFlightRecorder(recorder, trigger)` — captures on slow/error dispatch           | 🧪     |
| Decider integration | `decider.WithFlightRecorder[State](recorder, trigger)` — captures on slow/error Execute         | 🧪     |
| Projection host     | `projectionhost.WithFlightRecorder(recorder, trigger)` — captures on terminal worker failure    | 🧪     |
| Stack bundle        | `stack.WithFlightRecorder(recorder)` — lifecycle management + discovery via Bundle              | 🧪     |

**Coverage:** 92.5%. 35 tests, `-race` clean. Analyze with `go tool trace`.
API surface: 29 symbols.

---

## Reliability Infrastructure ✅ FULLY_FUNCTIONAL

The "reliability trio" (sans the deferred transactional outbox): idempotency +
DLQ (above) + managed projection host + scheduled deadlines.

### Managed Projection Host ✅ FULLY_FUNCTIONAL

> `import "github.com/larsartmann/go-cqrs-lite/projectionhost/v4"`

The "last loop every consumer rewrites", as a library module. Composes any
`event.SeekableJournal` + `event.CheckpointStore` + `projection.Projection`s
into a managed lifecycle.

| Feature                   | Detail                                                                                                                 | Status |
| ------------------------- | ---------------------------------------------------------------------------------------------------------------------- | ------ |
| Host                      | `Host` — manages projection workers, lifecycle, and health                                                             | ✅     |
| Per-projection goroutines | Each registered projection runs independently in its own goroutine                                                     | ✅     |
| Crash auto-restart        | Workers restart on panic/error with exponential backoff (configurable initial/max)                                     | ✅     |
| Checkpoint persistence    | Survives restarts — reads resume from the last committed checkpoint (no event loss)                                    | ✅     |
| Dead-letter queue         | `DeadLetterStore` / `MemoryDeadLetterStore` / `SQLiteDeadLetterStore` — poison messages captured, checkpoint advances  | ✅     |
| DLQ admin                 | `DeadLetterStoreAdmin` (optional): `Count`, `ListPaged`, `PurgeBefore` — pagination, depth metrics, time-bounded purge | ✅     |
| Health / liveness         | `Status()` reports per-worker state + processed/errors/restarts counters                                               | ✅     |
| Graceful drain            | `Stop()` waits for in-flight events (30s timeout)                                                                      | ✅     |
| RegisterAndWait           | Convenience: register + start + block until ctx cancelled                                                              | ✅     |
| Lag monitoring            | `LagDuration()` + `LagPerProjection()` — wall-clock lag for dashboards/alerting                                        | ✅     |
| CheckStaleness            | `WithMaxStaleness(d)` — reject reads whose projection lag exceeds threshold                                            | ✅     |

Worker states: `idle`, `running`, `live`, `backoff`, `draining`, `stopped`, `failed`.
Reads directly from `event.SeekableJournal` — no message-bus dependency. For
live (push) delivery alongside replay, pair with `watermill/CatchUpSubscriber`.

### Scheduling (Durable Deadlines) ✅ FULLY_FUNCTIONAL

> `import "github.com/larsartmann/go-cqrs-lite/scheduling/v4"`

Classic ES need — "cancel the order 30 minutes after creation if still unpaid" —
as a library primitive.

| Feature             | Detail                                                                           | Status |
| ------------------- | -------------------------------------------------------------------------------- | ------ |
| TimerStore          | `TimerStore` interface — `Schedule`, `Due`, `MarkFired`, `Cancel`                | ✅     |
| MemoryTimerStore    | In-memory `TimerStore` for development and testing                               | ✅     |
| SQLTimerStore       | `storage.SQLTimerStore[T]` — persistent `TimerStore` backed by `*sql.DB`         | ✅     |
| Scheduler           | Polls `Due()`, dispatches via callback, `MarkFired()`; retries failed dispatches | ✅     |
| Idempotent schedule | Re-scheduling the same `TimerID` is a no-op (safe on command retry)              | ✅     |
| Cancel              | Remove a timer before it fires (e.g. order paid → cancel timeout)                | ✅     |
| Configurable        | `WithPollInterval`, `WithMaxRetries`, `WithLogger`                               | ✅     |

---

## Schema Evolution ✅ FULLY_FUNCTIONAL

> `import "github.com/larsartmann/go-cqrs-lite/schema/v4"`

| Feature                  | Detail                                                                                                    | Status |
| ------------------------ | --------------------------------------------------------------------------------------------------------- | ------ |
| Upcaster                 | `Upcaster` interface — transforms old schema versions to newer on load                                    | ✅     |
| Upcaster constructor     | `NewUpcaster(eventType, fromVersion, fn)` — version-gated transform                                       | ✅     |
| Cycle detection          | Registry detects schema version revisits during upcast chain                                              | ✅     |
| VersionedStore           | `VersionedStore` wraps any `event.Store` — transparent upcasting on all read methods                      | ✅     |
| VersionedSeekableJournal | `VersionedSeekableJournal` wraps `SeekableJournal` — upcasting for projection host (`ReadAll`/`ReadFrom`) | ✅     |
| Full load API            | `Load`, `LoadFromVersion`, `LoadToVersion`, `LoadToTimestamp` — all with upcasting                        | ✅     |
| Schema validator         | `Validator` with `RegisterType[T]()`, strict/lenient modes, custom codecs (ADR-0017)                      | ✅     |
| Custom validators        | `RegisterTypeWithValidator[T](v, type, fn)` — business-rule validation after decode                       | ✅     |

---

## Snapshot ✅ FULLY_FUNCTIONAL

> `import "github.com/larsartmann/go-cqrs-lite/snapshot/v4"`

| Feature          | Detail                                                                              | Status |
| ---------------- | ----------------------------------------------------------------------------------- | ------ |
| Snapshot type    | `Snapshot` struct with StreamID, StreamType, Version, State, CreatedAt              | ✅     |
| Store interfaces | `SnapshotSink`, `SnapshotSource`, `SnapshotStore` (Sink+Source)                     | ✅     |
| Strategy         | `SnapshotStrategy` interface + `EveryNEvents(n)` built-in                           | ✅     |
| Read-pressure    | `ReadPressure` strategy — snapshots based on load frequency                         | ✅     |
| Aggregate-aware  | `AggregateAwareStrategy` + `ReadTracker` optional interfaces                        | ✅     |
| Helper functions | `ShouldSnapshot`, `ShouldSnapshotFor`, `SaveSnapshot` — decider integration helpers | ✅     |

---

## Payload Codec ✅ FULLY_FUNCTIONAL

> `import "github.com/larsartmann/go-cqrs-lite/codec/v4"`

| Feature               | Detail                                                                                  | Status |
| --------------------- | --------------------------------------------------------------------------------------- | ------ |
| Codec interface       | `Codec` — `Encoding()`, `Encode(v)`, `Decode(data, v)`                                  | ✅     |
| JSON codec            | `JSONCodec` — standard JSON encoding                                                    | ✅     |
| CBOR codec            | `CBORCodec` — deterministic canonical CBOR with sorted map keys                         | ✅     |
| CBOR compact codec    | `CBORCompactCodec` — ~35% smaller via `toarray` positional mode                         | ✅     |
| Raw passthrough       | `RawCodec` — `[]byte` pass-through (no encoding)                                        | ✅     |
| BufferEncoder         | Optional `BufferEncoder` interface — zero-alloc encoding into caller buffer             | ✅     |
| CBOR diagnostic       | `Diagnose(data)` — human-readable CBOR output for debugging                             | ✅     |
| Cross-codec transcode | `TranscodeToJSON(payload, enc)` — schema-free CBOR→JSON bridge for browser/REST output  | ✅     |
| Encoding constants    | `EncodingJSON`, `EncodingCBOR`, `EncodingRaw`                                           | ✅     |
| Envelope wrapping     | `WrapEncode`/`UnwrapDecode` — self-describing blind stores (ADR-0044)                   | ✅     |
| CBOR default          | All codec defaults flipped to CBOR (ADR-0053) — backward-compat via envelopes           | ✅     |
| Timezone-safe types   | `Instant`, `WallTime`, `Date` — prevent CBOR timezone loss in event payloads (ADR-0056) | ✅     |
| C013 lint rule        | Detects `time.Time` fields in event payloads, suggests timezone-safe alternatives       | ✅     |

---

## In-Memory Implementations 🧪 TESTING_ONLY

> `import "github.com/larsartmann/go-cqrs-lite/storage/memory/v4"`

| Component             | Detail                                                                                                   | Status |
| --------------------- | -------------------------------------------------------------------------------------------------------- | ------ |
| MemoryStore           | `event.Store` + `Journal` + `SeekableJournal` + `BackwardsSource` + `StreamLoader` with defensive copies | 🧪     |
| MemoryBus             | `event.Bus` with typed `Subscribe` + `SubscribeAll` + handler/publish middleware                         | 🧪     |
| MemorySnapshotStore   | `snapshot.SnapshotStore` with deep-copy snapshots, version-aware `LoadAtVersion`                         | 🧪     |
| MemoryCheckpointStore | `event.CheckpointStore` for projection checkpointing                                                     | 🧪     |
| MemoryCommandStore    | `command.Store` + `CommandJournal` + `SeekableCommandJournal` for persisted command log                  | 🧪     |
| MemoryCommandBus      | `command.Bus` with typed `Subscribe` + `SubscribeAll` + middleware chain                                 | 🧪     |
| MemoryQueryStore      | `query.QueryStore` + `QueryJournal` + `SeekableQueryJournal` for persisted query audit log               | 🧪     |

**Intended use:** Testing and development only. All implementations are thread-safe (`sync.RWMutex`), support `Close()` lifecycle, and return defensive copies.

---

## Middleware Suite ✅ FULLY_FUNCTIONAL

> `import "github.com/larsartmann/go-cqrs-lite/middleware/v4"`

All **9 concerns** are provided for all 3 message types (command, event, query) — **27 domain-specific middleware factories** + generic `Middleware[M]` for custom message types.

### Generic Middleware ✅

| Feature               | Detail                                                                        | Status |
| --------------------- | ----------------------------------------------------------------------------- | ------ |
| Generic Handler[M]    | `Handler[M]` = `func(context.Context, M) error` — works with any message type | ✅     |
| Generic Middleware[M] | `Middleware[M]` = `func(Handler[M]) Handler[M]`                               | ✅     |
| MessageAdapter[M]     | `MessageAdapter[M]` — converts between message types                          | ✅     |
| Domain adapters       | `CommandAdapter`, `EventAdapter`, `QueryAdapter` — pre-built adapters         | ✅     |

### Logging ✅

| Factory                  | Logs                                                                               |
| ------------------------ | ---------------------------------------------------------------------------------- |
| `CommandLogging(logger)` | `"command dispatching"` / `"succeeded"` / `"failed"` with type, streamID, duration |
| `EventLogging(logger)`   | Same pattern for events                                                            |
| `QueryLogging(logger)`   | Same pattern for queries                                                           |

Accepts `*slog.Logger`.

### Metrics ✅

| Factory                    | Records                                                              |
| -------------------------- | -------------------------------------------------------------------- |
| `CommandMetrics(recorder)` | `"command_success"` / `"command_error"` with duration and type label |
| `EventMetrics(recorder)`   | Same for events                                                      |
| `QueryMetrics(recorder)`   | Same for queries                                                     |

Accepts any `MetricsRecorder` interface (`Observe`).

### OTel Metrics ✅

| Factory                         | Detail                                                           |
| ------------------------------- | ---------------------------------------------------------------- |
| `NewOTelMetricsRecorder(meter)` | Creates `OTelMetricsRecorder` using OpenTelemetry `metric.Meter` |
| `CommandOTelMetrics(meter)`     | Pre-wired OTel metrics for commands                              |
| `EventOTelMetrics(meter)`       | Pre-wired OTel metrics for events                                |
| `QueryOTelMetrics(meter)`       | Pre-wired OTel metrics for queries                               |
| `CommandTypedMetrics(rec)`      | Typed attributes via `attribute.KeyValue` (no string labels)     |
| `EventTypedMetrics(rec)`        | Same for events                                                  |
| `QueryTypedMetrics(rec)`        | Same for queries                                                 |

### OTel Bundle ✅

| Feature                  | Detail                                                  |
| ------------------------ | ------------------------------------------------------- |
| `NewOTelBundle(tr, met)` | One-call tracing + metrics for all message kinds        |
| `.Command()`             | Spread into `cmdDisp.Use(...)`                          |
| `.Event()`               | Spread into `bus.Use(...)`                              |
| `.Query()`               | Spread into `qryDisp.Use(...)`                          |
| `.Publish()`             | Spread into `bus.UsePublish(...)`                       |
| `.CorrelationEnricher()` | Decider enricher bridging OTel baggage → event metadata |
| `WithMetricsDisabled()`  | Tracing-only mode (nil meter allowed)                   |

### HTTP Metrics ❌ REMOVED

Deleted — generic utility with no CQRS dependencies. Use `prometheus/` module (OTel→Prometheus bridge) for metrics exposition.

### Recovery ✅

| Factory             | Behavior                                         |
| ------------------- | ------------------------------------------------ |
| `CommandRecovery()` | Recovers panics → returns error with stack trace |
| `EventRecovery()`   | Same                                             |
| `QueryRecovery()`   | Same (uses named returns for result + err)       |

### Retry ✅

| Factory                | Behavior                                                    |
| ---------------------- | ----------------------------------------------------------- |
| `CommandRetry(config)` | Exponential backoff with jitter, context-aware cancellation |
| `EventRetry(config)`   | Same                                                        |
| `QueryRetry(config)`   | Same                                                        |

**RetryConfig:** `MaxAttempts`, `InitialDelay`, `MaxDelay`, `Multiplier`, `IsRetryable` predicate. Jitter uses `math/rand/v2`.

### Tracing ✅

| Factory | Span |
| ----------------------------- | ------------------------------------------------------------------- | --- |
| `CommandTracing(tracer)` | `"command.handle"`, SpanKindServer, attributes: `cqrs.command.type` |
| `EventTracing(tracer)` | `"event.handle"`, SpanKindConsumer, attributes: `cqrs.event.type` |
| `QueryTracing(tracer)` | `"query.handle"`, SpanKindServer, attributes: `cqrs.query.type` |
| `EventPublishTracing(tracer)` | `"event.publish"`, SpanKindProducer | ✅ |

OpenTelemetry via `go.opentelemetry.io/otel/trace`. Caller provides the `Tracer`.

Retry middleware emits `retry.attempt.N` child spans per attempt. See `docs/SPAN_NAMING.md` for the full span naming convention.

### Trace Logging ✅

### Enrichers ✅

| Factory                      | Purpose                                                              |
| ---------------------------- | -------------------------------------------------------------------- |
| `OTelCorrelationEnricher`    | Bridges OTel baggage → event custom metadata (`otel.correlation_id`) |
| `OTelCorrelationIDFromEvent` | Extracts the stored OTel correlation ID from an event                |

| Factory                       | Behavior                                          |
| ----------------------------- | ------------------------------------------------- |
| `CommandTraceLogging(logger)` | Logs trace ID + span ID alongside command details |
| `EventTraceLogging(logger)`   | Same for events                                   |
| `QueryTraceLogging(logger)`   | Same for queries                                  |

### Validation ✅

| Factory                        | Behavior                                                  |
| ------------------------------ | --------------------------------------------------------- |
| `CommandValidation(validator)` | Calls validator before handler; returns descriptive error |
| `EventValidation(validator)`   | Same                                                      |
| `QueryValidation(validator)`   | Same                                                      |

### Circuit Breaker ✅

| Factory                         | Behavior                                                                                  |
| ------------------------------- | ----------------------------------------------------------------------------------------- |
| `CommandCircuitBreaker(config)` | Three-state machine: Closed → Open (threshold) → Half-Open (timeout) → Closed (successes) |
| `EventCircuitBreaker(config)`   | Same                                                                                      |
| `QueryCircuitBreaker(config)`   | Same                                                                                      |

**CircuitBreakerConfig:** `FailureThreshold` (default 5), `SuccessThreshold` (default 3), `Timeout` (default 30s), `IsFailure` predicate. Rejected requests return `ErrCircuitBreakerOpen` wrapped as transient.

### Health Check ❌ REMOVED

Deleted — generic utility with no CQRS dependencies and zero consumers.

### SSE Broker ✅ (moved to transport/http/)

> `import "github.com/larsartmann/go-cqrs-lite/transport/http/v4"`

| Feature              | Detail                                                                                                                              | Status |
| -------------------- | ----------------------------------------------------------------------------------------------------------------------------------- | ------ |
| SSEBroker            | Server-Sent Events broker with `AddClient`, `RemoveClient`, `ClientCount`, `Close`                                                  | ✅     |
| SSEHandler           | `net/http` handler for SSE connections with client ID extraction                                                                    | ✅     |
| Thread-safe          | Concurrent client management with proper channel lifecycle                                                                          | ✅     |
| Last-Event-ID replay | Reconnect via `WithReconnectJournal(journal, limit)` — replays missed events before live                                            | ✅     |
| Replay timeout       | `WithReplayTimeout(d)` — stops replay after duration, sends advisory event, switches to live                                        | ✅     |
| Byte-budget replay   | `WithReplayByteBudget(bytes)` — stops replay when payload bytes exceed budget                                                       | ⚠️     |
| Retry field          | `WithRetryInterval(d)` — writes `retry:` field on connect for browser auto-reconnect                                                | ✅     |
| Event filtering      | `WithEventFilter(fn)` — broker-level predicate that drops events before fanout                                                      | ✅     |
| Auth middleware      | `SSEAuthMiddleware(next, tokenFunc)` — reference bearer-token auth implementation                                                   | ✅     |
| Backfill endpoint    | `BackfillHandler(broker)` — REST endpoint returning missed events as JSON array                                                     | ✅     |
| Payload transform    | `WithPayloadTransform(fn)` + ready-made `CBORToJSONTransform` adapter — wire-format transcoding (CBOR→JSON) on SSE + backfill paths | ✅     |
| Per-client stats     | `Stats() []ClientStats` — per-client buffered event depth for debugging slow consumers                                              | ✅     |
| Graceful close       | `CloseWithGrace(d)` — drains in-flight events before closing client channels                                                        | ✅     |
| Dedup ring           | Bounded `dedup.Ring` (1024 entries) for replay→live deduplication — O(1), memory-bounded                                            | ✅     |
| Replay metrics       | `ReplayMetrics` struct + OTel instruments (duration histogram, event counter, incomplete counter)                                   | ✅     |

### gRPC Transport ✅ FULLY_FUNCTIONAL

> `import "github.com/larsartmann/go-cqrs-lite/transport/grpc/v4"`

Remote gRPC transport for command & query dispatch (ADR-0025). Bridges gRPC clients to local dispatchers.

| Feature           | Detail                                                                  | Status |
| ----------------- | ----------------------------------------------------------------------- | ------ |
| CommandService    | `RegisterCommandService(srv, dispatcher)` — serves commands over gRPC   | ✅     |
| QueryService      | `RegisterQueryService(srv, dispatcher)` — serves queries over gRPC      | ✅     |
| CommandClient     | `NewCommandClient(conn)` — remote `command.Dispatcher` over a gRPC conn | ✅     |
| QueryClient       | `NewQueryClient(conn)` — remote `query.Dispatcher` over a gRPC conn     | ✅     |
| Protobuf contract | Generated `.proto` types in `transport/grpc/proto`                      | ✅     |

> Note: `transport/grpc` is a first-class member of `go.work`. The genproto conflict (cockroachdb/errors monolithic `google.golang.org/genproto` vs grpc-go's split `genproto/googleapis/rpc`) is resolved via a workspace-level `replace` directive pinning genproto to a version where the `googleapis/rpc` packages are split out.

### Profiling ❌ REMOVED

Deleted — trivial `net/http/pprof` re-export. Use `import _ "net/http/pprof"` directly.

---

## Event Signing ✅ FULLY_FUNCTIONAL

> `import "github.com/larsartmann/go-cqrs-lite/signing/v4"`

### Single-Signature Mode

| Feature             | Detail                                                                                 | Status |
| ------------------- | -------------------------------------------------------------------------------------- | ------ |
| HMAC-SHA256 signer  | `NewHMAC(key)` — `SignerVerifier` (sign + verify with same key)                        | ✅     |
| Ed25519 signer      | `NewEd25519(privateKey)` — `Signer` (sign-only, separate verifier)                     | ✅     |
| Ed25519 verifier    | `NewEd25519Verifier(publicKey)` — `Verifier` (verify-only)                             | ✅     |
| Key pair generation | `GenerateEd25519KeyPair()` — convenience function                                      | ✅     |
| Canonical encoding  | Deterministic length-prefixed format v1 (id, type, aggID, version, SHA-256 of payload) | ✅     |
| Event helpers       | `AttachSignature`, `ExtractSignature`, `HasSignature` — stored in event metadata       | ✅     |
| CloneEvent          | Deep copy of event for signing (zero-copy payload + fresh metadata)                    | ✅     |
| SignMiddleware      | `event.PublishMiddleware` — auto-signs every published event                           | ✅     |
| VerifyMiddleware    | `event.Middleware` — verifies signatures if present; allows unsigned through           | ✅     |
| RequireSignature    | `event.Middleware` — rejects unsigned events; verifies present signatures              | ✅     |

### Multi-Signature Mode (`signing/multisig` sub-package)

| Feature               | Detail                                                                                             | Status |
| --------------------- | -------------------------------------------------------------------------------------------------- | ------ |
| MultiSigner           | Actor-based multi-party signing with heterogeneous algorithms (e.g., device Ed25519 + server HMAC) | ✅     |
| SignatureEntry        | Per-actor entry with `Actor`, `Algorithm`, `Sig`, `SignedAt`; re-signing replaces prior entry      | ✅     |
| MultiSignature        | Collection of entries with `Count()`, `HasActor()`, `Get()`, `Actors()`                            | ✅     |
| MultiSignMiddleware   | Appends actor's signature to multi-sig collection                                                  | ✅     |
| MultiVerifyMiddleware | Verifies specific actor's multi-sig; allows missing through                                        | ✅     |
| RequireMultiSig       | Rejects events missing signatures from ALL required actors; verifies all                           | ✅     |
| VerifyAll             | Bulk verification using actor→verifier map                                                         | ✅     |
| VerifierMap           | Convenience builder from `MultiSigner` slice to `map[Actor]Verifier`                               | ✅     |

**Key material always defensively copied. No external crypto dependencies beyond Go stdlib.**

---

## Event Encryption ✅ FULLY_FUNCTIONAL

> `import "github.com/larsartmann/go-cqrs-lite/encryption/v4"`

| Feature            | Detail                                                                              | Status |
| ------------------ | ----------------------------------------------------------------------------------- | ------ |
| XChaCha20-Poly1305 | `NewXChaCha20Poly1305(key)` — `EncrypterDecrypter` with 32-byte key                 | ✅     |
| AES-256-GCM        | `NewAES256GCM(key)` — `EncrypterDecrypter` with 32-byte key                         | ✅     |
| Algorithm type     | `Algorithm` strong type (`XChaCha20Poly1305`, `AES256GCM`)                          | ✅     |
| KeyID type         | `KeyID` for key rotation tracking                                                   | ✅     |
| KeyResolver        | `KeyResolver` interface for multi-key decryption                                    | ✅     |
| Ciphertext         | `Ciphertext` type with `IsZero`, `Equal`, `Bytes`, JSON serialization               | ✅     |
| Codec wrapper      | `NewCodec(codec, encrypter)` — composable encrypt+encode pipeline                   | ✅     |
| Event helpers      | `AttachEncryption`, `ExtractCiphertext`, `HasEncryption` — stored in event metadata | ✅     |
| ExtractAlgorithm   | `ExtractAlgorithm(evt)` — reads encryption algorithm from event metadata            | ✅     |
| ExtractKeyID       | `ExtractKeyID(evt)` — reads key ID from event metadata                              | ✅     |
| EncryptMiddleware  | `event.PublishMiddleware` — auto-encrypts event payloads                            | ✅     |
| DecryptMiddleware  | `event.Middleware` — auto-decrypts event payloads                                   | ✅     |

**No external crypto dependencies beyond Go stdlib (`golang.org/x/crypto`).**

---

## Auto-Documentation

### Catalog System ✅ FULLY_FUNCTIONAL

> `import "github.com/larsartmann/go-cqrs-lite/catalog/v4"`

| Feature             | Detail                                                                                                                                                         | Status |
| ------------------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------- | ------ |
| Registry            | Thread-safe builder: `AddService`, `AddCommand`, `AddEvent`, `AddQuery`, `AddDomain`, `AddChannel`, `AddFlow`, `AddDataStore`, `AddTeam`, `AddUser`, `Build()` | ✅     |
| Builder             | `NewBuilder` fluent API: `ConfigureService`, `ConfigureDomain`, `ConfigureChannel`, `AddDataStore`, `AddFlow`, `AddTeam`, `AddUser`                            | ✅     |
| Schema reflection   | `SchemaFromType[T]()` — auto-generates JSON Schema from Go structs via `reflect`                                                                               | ✅     |
| Struct tag support  | `json` (name, omitempty), `doc`/`description`, `format`, `default`, `enum`, `nullable`, `deprecated`, `pattern`                                                | ✅     |
| Immutable catalog   | `Build()` returns deep-copied, immutable `*Catalog`                                                                                                            | ✅     |
| Validation          | `Catalog.Validate()` returns `[]Violation` — checks titles, duplicates, references                                                                             | ✅     |
| Exporter interface  | `Exporter[T]` + `ErrorExporter` — pluggable output format                                                                                                      | ✅     |
| WalkMessages        | `WalkMessages(cat, fn)` — iterate all messages across all services                                                                                             | ✅     |
| Rich resource model | Services, Domains, Channels, DataStores, Flows, Teams, Users — with badges, owners, repositories, specifications                                               | ✅     |
| Message config      | `Command[T]()`, `Event[T]()`, `Query[T]()` — generic message registration with options                                                                         | ✅     |
| Channel config      | `ChannelAddress`, `ChannelProtocols`, `ChannelMessages`, `ChannelDeliveryGuarantee`, `ChannelRoutes`, `ChannelOwners`, `ChannelBadges`                         | ✅     |
| Domain config       | `DomainSends`, `DomainReceives`, `DomainEntities`, `DomainBadges`, `DomainOwners`, `DomainAttachments`                                                         | ✅     |
| Service config      | `ServiceBadges`, `ServiceRepository`, `ServiceWritesTo`, `ServiceReadsFrom`, `ServiceEntities`, `ServiceSpecifications`, `ServiceAttachments`, `ServiceOwners` | ✅     |
| ID parsing          | `ParseServiceID`, `ParseDomainID`, `ParseMessageID`, `ParseChannelID`                                                                                          | ✅     |

> `import "github.com/larsartmann/go-cqrs-lite/catalog/v4/asyncapi"`

| Feature             | Detail                                                                                                      | Status |
| ------------------- | ----------------------------------------------------------------------------------------------------------- | ------ |
| Document generation | `Exporter.Export(catalog)` produces full AsyncAPI 3.0 `Document`                                            | ✅     |
| YAML output         | `Document.MarshalYAML()` — uses `go-faster/yaml`                                                            | ✅     |
| JSON output         | `Document.MarshalJSON()` — type-alias trick to avoid recursion                                              | ✅     |
| Server config       | `WithServer(name, host, protocol)` option (defaults: kafka, localhost:9092)                                 | ✅     |
| Channel mapping     | Commands → `receive`, Events with `Sends` → `send`, Events with `Receives` → `receive`, Queries → `receive` | ✅     |

> `import "github.com/larsartmann/go-cqrs-lite/catalog/v4/eventcatalog"`

| Feature        | Detail                                                                        | Status |
| -------------- | ----------------------------------------------------------------------------- | ------ |
| MDX generation | Services, commands, events, queries — all with YAML frontmatter               | ✅     |
| Schema files   | `schema.json` per message (only when schema is non-nil)                       | ✅     |
| Domain pages   | Domain frontmatter with service associations                                  | ✅     |
| Config files   | `eventcatalog.config.js`, `package.json` with `@eventcatalog/core` dependency | ✅     |
| LLM summary    | `llms.txt` — plain-text catalog summary for LLM consumption                   | ✅     |

> `import "github.com/larsartmann/go-cqrs-lite/catalog/v4/d2"`

| Feature             | Detail                                                      | Status |
| ------------------- | ----------------------------------------------------------- | ------ |
| D2 text export      | `Exporter.Export(cat)` produces D2 diagram syntax           | ✅     |
| Service nodes       | Color-coded rectangles per service with command/event/query | ✅     |
| Cross-service flows | Animated arrows between publishers and receivers            | ✅     |
| Domain grouping     | Domain labels with dashed "contains" links to services      | ✅     |
| Schema tooltips     | Field names and types shown on hover                        | ✅     |

> `import "github.com/larsartmann/go-cqrs-lite/catalog/v4/openapi"`

| Feature             | Detail                                                            | Status |
| ------------------- | ----------------------------------------------------------------- | ------ |
| Document generation | `Exporter.Export(catalog)` produces full OpenAPI 3.0.3 `Document` | ✅     |
| JSON output         | `Document` serializes to JSON                                     | ✅     |
| Schema generation   | Auto-generates JSON Schema from catalog types                     | ✅     |
| Base path support   | `WithBasePath(path)` option for API path prefix                   | ✅     |
| Description option  | `WithDescription(desc)` for document metadata                     | ✅     |

> `import "github.com/larsartmann/go-cqrs-lite/catalog/v4/docserver"`

| Feature            | Detail                                                                | Status |
| ------------------ | --------------------------------------------------------------------- | ------ |
| HTTP handlers      | Framework-agnostic `net/http` handlers for serving docs               | ✅     |
| OpenAPI rendering  | Scalar UI for interactive API documentation                           | ✅     |
| AsyncAPI rendering | AsyncAPI React for event documentation                                | ✅     |
| Raw spec serving   | JSON/YAML endpoints for both OpenAPI and AsyncAPI                     | ✅     |
| Catalog provider   | `CatalogProvider` func — generates fresh catalog on each request      | ✅     |
| Embedded assets    | HTML/JS/CSS embedded via `embed.FS` — zero external file dependencies | ✅     |

---

## SQL & Key-Value Event Stores ✅ FULLY_FUNCTIONAL

### SQL Stores (PostgreSQL / SQLite / MySQL)

> `import "github.com/larsartmann/go-cqrs-lite/storage/v4"`

| Feature                    | Detail                                                                                                                                       | Status |
| -------------------------- | -------------------------------------------------------------------------------------------------------------------------------------------- | ------ |
| PostgreSQL event store     | `NewSQLEventStore(db)` implements `event.Store`                                                                                              | ✅     |
| SQLite event store         | `NewSQLiteEventStore(db)` — `?` placeholders, `BLOB`/`TEXT` DDL                                                                              | ✅     |
| MySQL/MariaDB event store  | `NewMySQLBackend(db)` — `LONGBLOB`/`JSON` DDL, `ON DUPLICATE KEY UPDATE` upsert                                                              | ✅     |
| Custom dialect             | `NewSQLEventStoreWithDialect(db, d)` — pluggable SQL backend                                                                                 | ✅     |
| Schema DDL                 | `Schema()` PostgreSQL, `SQLiteSchema()` for SQLite/Turso                                                                                     | ✅     |
| Per-table DDL              | `SnapshotSchema`, `CheckpointSchema` + SQLite variants                                                                                       | ✅     |
| Optimistic concurrency     | `Save` checks version in transaction                                                                                                         | ✅     |
| AppendBatch                | Appends without concurrency check                                                                                                            | ✅     |
| Full load API              | `Load`, `LoadFromVersion`, `LoadToVersion`, `LoadToTimestamp`                                                                                | ✅     |
| LoadBackwards              | Implements `event.BackwardsSource` — newest-first                                                                                            | ✅     |
| Time-travel SQL queries    | `LoadToVersion`, `LoadToTimestamp` with composite timestamp index                                                                            | ✅     |
| Journal / SeekableJournal  | `ReadAll()`, `ReadFrom(afterEventID, limit)`                                                                                                 | ✅     |
| Stream loading             | `LoadStream()` returns cursor-based `sqlEventStream` — memory-efficient iteration                                                            | ✅     |
| Metadata persistence       | Full roundtrip: correlation IDs, user IDs, custom metadata                                                                                   | ✅     |
| SQL SnapshotStore          | PostgreSQL + SQLite variants, upsert, version-aware load, delete                                                                             | ✅     |
| SQL CheckpointStore        | PostgreSQL + SQLite variants, upsert, `sql.ErrNoRows` handling                                                                               | ✅     |
| SQL CommandStore           | `SQLCommandStore` implements `command.Store` — Save, AppendBatch, Load, LoadFromTimestamp, LoadToTimestamp                                   | ✅     |
| SQL Backend                | `SQLBackend` facade returning `EventStore()`, `SnapshotStore()`, `CheckpointStore()`, `CommandStore()`                                       | ✅     |
| StreamProjection           | Maintains SQL read-model tables from event streams with tombstone detection                                                                  | ✅     |
| Incremental rollups        | `ProjectionSink.Increment(ctx, table, key, counterCol, delta)` — atomic counter via `ON CONFLICT DO UPDATE` (ADR-0033)                       | ✅     |
| RelationalProjection.Reset | `Reset(ctx)` implements `projectionhost.Resettable` — wipes all tables for zero-based replay                                                 | ✅     |
| SQLStreamReader            | `listing.StreamReader` implementation reading from projection tables                                                                         | ✅     |
| DB helpers                 | `OpenSQLite`, `OpenSQLiteInMemory`, `SQLiteInitSchema`, `SQLiteEnableWAL`, `ConfigureSQLitePool`, `ConfigureTursoPool`, `PostgresInitSchema` | ✅     |
| Dialect abstraction        | `Dialect` interface with `Placeholder`, `FormatTime`, `ScanTimeDest`, `ParseTime`, 5 schema methods, 4 upsert/quoting methods (ADR-0080)     | ✅     |
| SQL sub-package            | `storage/sql` — `DBHandle`, `OwnedDBHandle`, generic `LoadWithSpan[T]`, `QueryRows[T]`, `ScanSlice[T]`, `ReconstructEvent`                   | ✅     |
| Eventstore sub-package     | `storage/eventstore` — `SQLEventStore`, `SQLSnapshotStore`, `SQLCheckpointStore` (re-exported via aliases in `storage/`)                     | ✅     |
| Readmodel sub-package      | `storage/readmodel` — `SQLKVStore` (re-exported via aliases in `storage/`)                                                                   | ✅     |
| Close lifecycle            | No-op `Close()` — does not close `*sql.DB`; caller owns DB lifecycle                                                                         | ✅     |
| HealthCheck (all stores)   | `OwnedDBHandle.HealthCheck(ctx)` — inherited by all SQL stores via embedding. Pings DB, checks closed state                                  | ✅     |

### Pebble Key-Value Store

> `import "github.com/larsartmann/go-cqrs-lite/storage/pebble/v4"`

| Feature                | Detail                                                                                 | Status |
| ---------------------- | -------------------------------------------------------------------------------------- | ------ |
| EventStore             | `NewStore(db, logger)` implements `event.Store` + `Journal` + `SeekableJournal`        | ✅     |
| CBOR envelope          | Events serialized as CBOR with JSON backward compatibility layer                       | ✅     |
| Per-stream locking     | Sharded mutex pool (FNV-1a hash, 256 shards) — optimistic concurrency without sync.Map | ✅     |
| Optimistic concurrency | `Save` checks version before commit                                                    | ✅     |
| AppendBatch            | Appends without concurrency check                                                      | ✅     |
| Full load API          | `Load`, `LoadFromVersion`, `LoadToVersion`, `LoadToTimestamp`                          | ✅     |
| Journal                | `ReadAll()` — global event stream ordered by occurred_at                               | ✅     |
| SeekableJournal        | `ReadFrom(afterEventID, limit)` — position-based replay                                | ✅     |
| Async writes           | `WithAsyncWrites()` option — disables Pebble sync on commit for throughput             | ✅     |
| Nil-safe logging       | All log operations nil-safe — no panics on nil logger                                  | ✅     |
| SnapshotStore          | `NewSnapshotStore(db, logger)` — CBOR envelope, ignores older versions on Save         | ✅     |
| Snapshot LoadAtVersion | "At or before" semantics — returns snapshot when stored version ≤ requested            | ✅     |
| CheckpointStore        | `NewCheckpointStore(db, logger)` — CBOR envelope, returns zero checkpoint if missing   | ✅     |
| Shared DB              | Event + Snapshot + Checkpoint stores share one `*pebble.DB` via disjoint key prefixes  | ✅     |

### Turso Database Connector

> `import "github.com/larsartmann/go-cqrs-lite/storage/turso/v4"`

| Feature                 | Detail                                                                                                                                                         | Status |
| ----------------------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------- | ------ |
| Local DB                | `Open(dbPath)`, `OpenInMemory()` — embedded Turso Database                                                                                                     | ✅     |
| Schema init             | `InitSchema(ctx, db)` — delegates to `storage.SQLiteInitSchema`                                                                                                | ✅     |
| Backend facade          | `NewBackend(db)` — all 5 stores (event, command, query, snapshot, checkpoint) sharing one `*sql.DB`, goroutine-safe lazy init                                  | ✅     |
| Convenience stores      | `NewEventStore`, `NewCommandStore`, `NewQueryStore`, `NewSnapshotStore`, `NewCheckpointStore` — thin wrappers over storage/                                    | ✅     |
| Remote sync             | `OpenSync(ctx, dbPath, remoteURL, authToken)` — `SyncDB` with `Push`, `Pull`, `Checkpoint`, `Stats`, `HealthCheck`, `Close`                                    | ✅     |
| Advanced sync config    | `OpenSyncWithConfig(ctx, ..., opts)` — `WithSyncClientName`, `WithSyncLongPollTimeout`, `WithSyncBusyTimeout`, `WithSyncBootstrapIfEmpty`, `WithSyncNamespace` | ✅     |
| Phantom types           | `DbPath`, `RemoteURL`, `AuthToken` — compile-time type safety                                                                                                  | ✅     |
| Pool configuration      | `ConfigurePool(db)` — caps `MaxOpenConns` at 1 for embedded Turso Database (required to avoid "database is locked")                                            | ✅     |
| Backward-compat aliases | `OpenTurso`, `NewTursoEventStore`, `NewTursoCommandStore`, `NewTursoQueryStore`, etc. — deprecated aliases preserved                                           | ✅     |
| Indexed schema init     | `InitSchemaWithIndexes`, `InitSchemaWithIndexesAndOptimizations` — tables + indexes + pragmas                                                                  | ✅     |
| Index convenience       | `NewIndexAdvisor`, `NewAutoIndexer`, `ApplyCQRSIndexes`, `ApplyTursoOptimizations`                                                                             | ✅     |

### Turso Indexing (sub-package) ✅ FULLY_FUNCTIONAL

> `import "github.com/larsartmann/go-cqrs-lite/storage/turso/v4/indexing"`

| Feature                  | Detail                                                                                          | Status |
| ------------------------ | ----------------------------------------------------------------------------------------------- | ------ |
| EXPLAIN-based Advisor    | `Advisor` analyzes queries via `EXPLAIN QUERY PLAN` — detects missing indexes, full table scans | ✅     |
| AutoIndexer              | `AutoIndexer` creates/drops indexes automatically — lifecycle with Enable/Disable/Apply         | ✅     |
| CQRS index presets       | `RecommendedCQRSIndexes()` — pre-built IndexSet for event store tables                          | ✅     |
| Per-table Policy         | `Policy` type — exclude tables, mark critical, skip auto-creation per table                     | ✅     |
| Priority classification  | `Priority` enum (Critical/Recommended/Optional) on Recommendations                              | ✅     |
| Dry-run mode             | `WithDryRun()` — collects DDL via `LastDDL()` without executing                                 | ✅     |
| OTel tracing             | All major operations traced via OpenTelemetry spans                                             | ✅     |
| Index usage stats        | `Stats(ctx, db)` — queries `sqlite_stat1` for index hit counts                                  | ✅     |
| Unused index detection   | `UnusedIndexes(ctx, db)` — finds indexes with zero hits                                         | ✅     |
| WAL checkpoint scheduler | `CheckpointScheduler` — background WAL auto-checkpoint                                          | ✅     |
| Optimization pragmas     | `DefaultOptimizations`, `ApplyOptimizations`, `ApplyWAL` — cache size, memory map, ANALYZE      | ✅     |
| Functional options       | `WithExcludedTables`, `WithAutoAnalyze` — configurable behavior                                 | ✅     |
| Benchmarks               | Indexed vs unindexed `ReadFrom` benchmarks proving value                                        | ✅     |

### DuckDB Analytical Store

> `import "github.com/larsartmann/go-cqrs-lite/stack/duckdb/v4"`

| Feature                 | Detail                                                                                           | Status |
| ----------------------- | ------------------------------------------------------------------------------------------------ | ------ |
| DuckDB dialect          | `storage/sql.DuckDBDialect` — Postgres-compatible placeholders, BLOB metadata, native timestamps | ✅     |
| Stack preset            | `duckdb.New(dsn)` — full `stack.Bundle` with event store, read models, in-process bus            | ✅     |
| CGo isolation           | `//go:build cgo` tags — pure-Go modules unaffected, consumers opt-in via import                  | ✅     |
| Multi-database topology | `duckdb.WithDSN(sqlopt.WithEventDB(...))` — split events/queries/views across DB files           | ✅     |
| Analytical read models  | `SQLViewModel[V,K]` — columnar scans, GROUP BY, window functions on event-sourced data           | ✅     |
| Performance tuning      | `WithThreads(n)`, `WithMemoryLimit("1GB")` — DuckDB worker thread + memory caps                  | ✅     |
| Contract test suite     | Full `contracttest.RunSuite` — event/command/read-model roundtrips, close idempotency            | ✅     |
| Benchmarkable backend   | `stack/bench` `BenchmarkBenchkitSuite_DuckDB` (CGo-gated) + `cmd/cqrs-bench --backend duckdb`    | ✅     |

---

## Stream Read Model ✅ FULLY_FUNCTIONAL

> `import "github.com/larsartmann/go-cqrs-lite/listing/v4"`

| Feature                     | Detail                                                                                               | Status |
| --------------------------- | ---------------------------------------------------------------------------------------------------- | ------ |
| StreamReader                | Interface: `List(ctx, ListOptions) → Page[StreamStatus]` — cursor-based stream listing               | ✅     |
| ListBuilder                 | Fluent API: `listing.NewListBuilder(reader).OfType("User").After(cursor).Limit(50).IncludeDeleted()` | ✅     |
| InMemoryStreamReader        | Reads from `event.Journal.ReadAll()` — single-pass, no persistence                                   | ✅     |
| TombstonePolicy             | `Exclude` (default), `Include`, `Only` — controls visibility of soft-deleted streams                 | ✅     |
| Page[T]                     | Cursor-based pagination with `HasMore` — no expensive TotalCount                                     | ✅     |
| StreamListing               | Lightweight identity: ID, Type, Version, EventCount, LastEventAt                                     | ✅     |
| StreamStatus                | Pairs StreamListing with computed TombstoneStatus                                                    | ✅     |
| StatusMiddleware            | Event bus middleware that publishes stream status changes                                            | ✅     |
| CacheInvalidationMiddleware | Returns `event.PublishMiddleware` that invalidates reader cache                                      | ✅     |
| ListRefsFromStatus          | Helper that strips status from page                                                                  | ✅     |

---

## OpenTelemetry Helpers ✅ FULLY_FUNCTIONAL

> `import "github.com/larsartmann/go-cqrs-lite/otel/v4"`

| Feature            | Detail                                                                                                                                                                              | Status |
| ------------------ | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ------ |
| Type aliases       | `Tracer`, `Span`, `SpanKind`, `KeyValue`, `Meter`, `Float64Histogram`, `Int64Counter` — re-exported from OTel                                                                       | ✅     |
| Tracer factory     | `NewTracer(component)` — creates OTel Tracer with standard instrumentation name                                                                                                     | ✅     |
| Meter factory      | `NewMeter(component)` — creates OTel Meter                                                                                                                                          | ✅     |
| Span helpers       | `StartSpan`, `RecordError`, `EndWithError`, `AddSpanEvent`, `StreamAttrs`, `CommandAttrs`, `EventAttrs`, `QueryAttrs`                                                               | ✅     |
| Context helpers    | `SpanFromContext`, `TraceIDFromContext`, `SpanIDFromContext`                                                                                                                        | ✅     |
| Attribute helpers  | `AttrString`, `AttrInt`, `AttrInt64`, `WithAttributes`, `WithSpanKind`, `ServiceResourceAttributes`                                                                                 | ✅     |
| Metric helpers     | `MetricWithAttributes`, `MetricWithDescription`, `MetricWithUnit`, `CounterAddWithAttributes`, `AddOption`                                                                          | ✅     |
| Metric views       | `NewCQRSViews()`, `CQRSHistogramBoundaries` — OTel SDK views with optimized CQRS latency buckets for all `cqrs.*` instruments                                                       | ✅     |
| Correlation        | `WithCorrelationID`, `CorrelationIDFromContext` — baggage-based correlation propagation                                                                                             | ✅     |
| W3C propagation    | `NewTextMapPropagator()` — W3C trace context + baggage propagator                                                                                                                   | ✅     |
| Logging helpers    | `ComponentLogger`, `ContextLogger` — structured logging with trace correlation                                                                                                      | ✅     |
| Standard constants | `AttrMessageKind`, `AttrCommandType`, `AttrEventType`, `AttrQueryType`, `AttrStreamType`, `AttrStreamID`, `AttrStreamVersion`, `AttrEventCount`, `AttrProjectionName`, `AttrStatus` | ✅     |

---

## Watermill Adapter ✅ FULLY_FUNCTIONAL

> `import "github.com/larsartmann/go-cqrs-lite/watermill/v4"`

| Feature              | Detail                                                                                                   | Status |
| -------------------- | -------------------------------------------------------------------------------------------------------- | ------ |
| Event protocol       | Bidirectional `event.Event` ↔ Watermill `message.Message` via 15+ metadata keys                          | ✅     |
| PublisherAdapter     | `NewPublisherAdapter(publisher)` — wraps `event.Publisher` as `message.Publisher`                        | ✅     |
| SubscriberAdapter    | `NewSubscriberAdapter(bus)` — wraps `event.Bus` as `message.Subscriber`, feeds `<-chan *message.Message` | ✅     |
| Full event fidelity  | 15 metadata keys preserve ID, type, stream, version, schema version, all metadata fields                 | ✅     |
| **Command protocol** | Bidirectional `command.Command` ↔ Watermill `message.Message` (type, stream, tracing, custom metadata)   | ✅     |
| **CommandBus**       | `NewCommandBus()` — full `command.Bus` backed by Watermill GoChannel + `WithCommandBackend` for brokers  | ✅     |
| **CommandPublisher** | `NewCommandPublisher(pub, topic)` — wraps `message.Publisher` as `command.Publisher`                     | ✅     |
| Custom metadata      | `custom.*` prefix preserves all custom metadata entries                                                  | ✅     |
| Correlation ID MW    | `CorrelationIDMiddleware()` — injects correlation ID into message metadata                               | ✅     |
| Retry middleware     | `NewRetryMiddleware(config)` + `DefaultRetryConfig()` — retry with backoff for handler errors            | ✅     |

---

## Prometheus Metrics Exporter ✅ FULLY_FUNCTIONAL

> `import "github.com/larsartmann/go-cqrs-lite/prometheus/v4"`

|| Feature | Detail | Status |
|| ------------------- | ------------------------------------------------------------------------------------------------- | ------ |
|| OTel→Prom bridge | `Setup()` — creates MeterProvider + HTTP handler backed by Prometheus registry | ✅ |
|| Custom registry | `WithRegistry(r)` — use a custom Prometheus registry | ✅ |
|| Handler options | `WithHandlerOptions(opts)` — configure promhttp.HandlerOpts | ✅ |
| Custom views | `WithViews(views...)` — apply OTel metric views (e.g. `cqrsotel.NewCQRSViews()`) for histogram boundaries | ✅ |

---

## Test Infrastructure 🧪 TESTING_ONLY

### eventtest 🧪

> `import "github.com/larsartmann/go-cqrs-lite/event/v4/eventtest"`

| Feature           | Detail                                                                                                                                                              | Status |
| ----------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ------ |
| FakeStore         | Full `Store + Journal + SeekableJournal` implementation with fluent override setters                                                                                | 🧪     |
| FakeBus           | Full `Bus` implementation with fluent override setters                                                                                                              | 🧪     |
| FakeSnapshotStore | `SnapshotStore` implementation with `SetSnapshot`, `SetLoadError`, `SetSaveError`                                                                                   | 🧪     |
| Event factories   | `NewTestEvent`, `NewEventOpts`, `QuickEvent`, `QuickEventOpts`, `MakeEvent`, `TamperEvent`                                                                          | 🧪     |
| Timeline fixtures | `MakeTimelineEvents`, `MakeThreeTimelineEvents`, `MakeLoadToTimestampFixtures`                                                                                      | 🧪     |
| Assertions        | `AssertGolden`, `AssertCallOrder`, `AssertEventType`, `AssertEventVersion`, and 10+ more                                                                            | 🧪     |
| Store test suite  | `TestStoreSaveAndLoad`, `TestStoreConcurrencyConflict`, `TestStoreAppendBatch`, `TestStoreLoadFromVersion`, `TestStoreMetadataRoundtrip` — reusable suite functions | 🧪     |
| Handler factories | `AppendEventsHandler`, `NoopEventHandler`, `FailingEventHandler`, `PanicEventHandler`, `CallbackEventHandler`                                                       | 🧪     |

### Scenario Testing DSL ✅ FULLY_FUNCTIONAL

> `import "github.com/larsartmann/go-cqrs-lite/scenario/v4"`

Fluent BDD harness for deciders and projections — no store or bus needed, just pure functions.

| Feature                 | Detail                                                                            | Status |
| ----------------------- | --------------------------------------------------------------------------------- | ------ |
| Decider Given/When/Then | `Given[Cmd,State](t, apply, initial, events...).When(cmd, decide).Then(types...)` | ✅     |
| ThenError               | Asserts the decide function returns an error wrapping the target                  | ✅     |
| ThenState               | Folds produced events and asserts the resulting state                             | ✅     |
| Projection Given        | `GivenProjection(t, proj, events...)` feeds events to a projection                | ✅     |
| ThenNoError / ThenError | Assert the projection handled all events without/with error                       | ✅     |

---

## Tools 🔧

### cqrs-gen Code Generator 🔧

> `go run github.com/larsartmann/go-cqrs-lite/cmd/cqrs-gen/v4`

| Feature             | Detail                                                                       | Status |
| ------------------- | ---------------------------------------------------------------------------- | ------ |
| AST-based scanning  | Parses Go source for `//cqrs:command <Name>` / `//cqrs:query <Name>` markers | ✅     |
| Typed handler gen   | Generates `Register<StructName>Handler` functions using `RegisterTyped[T]`   | ✅     |
| CLI flags           | `-type` (command/query), `-output` (file), `-pkg` (package name)             | ✅     |
| Recursive directory | Walks directories, skips `_test.go`, extracts markers from doc comments      | ✅     |

### API Stability Checker 🔧

> `go run github.com/larsartmann/go-cqrs-lite/cmd/api-stability/v4`

| Feature                | Detail                                                      | Status |
| ---------------------- | ----------------------------------------------------------- | ------ |
| Module scanning        | Parses all exported symbols from consumer-facing modules    | ✅     |
| Golden file comparison | Compares current API surface against `docs/api_surface.txt` | ✅     |
| Update mode            | `-update` flag regenerates golden file                      | ✅     |
| Diff reporting         | Reports REMOVED/NEW exports — CI gate for breaking changes  | ✅     |

### doc-check 🔧

> `go run github.com/larsartmann/go-cqrs-lite/cmd/doc-check/v4`

| Feature             | Detail                                                                  | Status |
| ------------------- | ----------------------------------------------------------------------- | ------ |
| Markdown scanning   | Scans `.md` files for Go code blocks                                    | ✅     |
| Symbol verification | Verifies Go import paths & qualified symbol refs actually exist in code | ✅     |
| Ghost detection     | Flags references to renamed/deleted symbols (docs-freshness gate)       | ✅     |

### cqrs-lint Domain-Aware Linter 🔧

> `go run github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/v4`

| Feature | Detail | Status |
| ------- | ------ | ------ |

| Output formats | Text, JSON, SARIF (GitHub Code Scanning), Markdown | ✅ |
| Health score | 0-100 score with severity-weighted breakdown | ✅ |
| Auto-fix | `--fix` / `--dry-run` with CQRSFixProvider (BeforeCode/AfterCode matching) | ✅ |
| Suppression comments | `//cqrs-lint:ignore(rule-id) reason` (space after `//` supported, comma-separated IDs supported) | ✅ |
| Block-level suppression | `//cqrs-lint:ignore-start` / `//cqrs-lint:ignore-end` for range suppression (ADR-0088) | ✅ |
| Stale suppression detection | `DetectStaleSuppressions` flags suppression comments where no finding fires | ✅ |
| CLI features | `--only C001,C002`, `--exclude`, `--color`, `--verbose`, `--health-score`, `init`, `--min-confidence` | ✅ |
| Config file | `.cqrs-lint.json` via cmdguard; presets (local-cli, production, library, read-only) | ✅ |
| Config presets | `local-cli`, `production`, `library`, `read-only` — sugar over feature flags | ✅ |
| Feature profile system | Auto-detects which go-cqrs-lite modules a consumer uses (store, command-flow, server, soft-delete, tracing, snapshot) and adapts rules | ✅ |
| Self-lint mode | `IsLibrarySelfLint()` auto-skips 29 consumer-coaching rules when linting the library source | ✅ |
| Import-alias resolution | `QualifierToImportPath` + `ImportQualifierMap` — rules work with aliased imports | ✅ |
| Monorepo support | Multi-module scanning via go.mod discovery | ✅ |
| Source snippets | Most detectors emit source-line context for SARIF/IDE integration | ✅ |
| `doctor` subcommand | Prints the detected feature profile for the target project | ✅ |
| F-series adoption coaching | 21 rules (F001–F021) that proactively coach consumers toward unused features | ✅ |
| T-series testing quality | 8 rules (T001–T008) detecting missing test helpers, parallel coverage gaps, snapshot store misuse | ✅ |
| E-series architecture | 17 rules (E001–E017) detecting consumer design issues (preset bypass, missing HTTP, signing disabled, etc.) | ✅ |
| 181 total rules | Correctness (36), API (31), boilerplate (28), adoption (21), architecture (17), consistency (16), performance (9), security (9), testing (8), version (6) | ✅ |
| A033 branded-ID roundtrip | Flags code that converts branded `id.Of[T]` to `string` and back (breaks type safety) | ✅ |
| C037 codec mismatch | Detects snapshot/event codec mismatches (CBOR events + JSON snapshots = deserialization failure) | ✅ |

---

## Integration Tests ✅

> `import "github.com/larsartmann/go-cqrs-lite/integration/v4"`

| Feature              | Detail                                                                         | Status |
| -------------------- | ------------------------------------------------------------------------------ | ------ |
| Full flow E2E        | Command dispatch → decider → store → bus → projection → query → stream loading | ✅     |
| Chaos testing        | Error propagation, panic recovery, retry logic, context cancellation           | ✅     |
| Cross-module BDD     | Event, command, query, signing, encryption integration via Ginkgo v2           | ✅     |
| Simulation framework | `EventGenerator` — single/multi-stream event generation for testing            | ✅     |
| Benchmarking         | 17 scale benchmarks (10K-1M events), realistic pipeline/concurrent benchmarks  | ✅     |
| OTel integration     | End-to-end OpenTelemetry tracing verification                                  | ✅     |

---

## Examples 💡 DEMO

| Example                      | Detail                                                                                                                                                                                            |
| ---------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `example/taskmanager/`       | Flagship full HTTP service: event sourcing, CQRS, KV + tombstone projections, SSE streaming, ProjectionHost with DLQ, signing, snapshot strategy, deriver (event→command), idempotency middleware |
| `example/getting-started/`   | Minimal 80-line example showing the core pipeline: counter stream, command dispatch, event store, bus, projection                                                                                 |
| `example/readme-quickstart/` | Compile-verified README Quick Start example — tests every API pattern from the main README                                                                                                        |

**Not reference applications.** These demonstrate library usage patterns.

---

## Not Yet Implemented 📐 PLANNED

Features mentioned in project docs/planning but with **no production code yet**:

| Feature                       | Description                                                       |
| ----------------------------- | ----------------------------------------------------------------- |
| PostgreSQL testcontainers     | ✅ DONE — testcontainers-go adopted (v0.43.0, postgres:16-alpine) |
| Documentation site            | Docusaurus/MkDocs/Hugo site                                       |
| Transport adapters            | gRPC ✅, NATS/ValKey (ADR-0025 accepted, no code)                 |
| Distributed projection runner | Leader election, multi-node coordination — raw idea               |

---

## Module Maturity Matrix

> 64 independently importable modules in `go.work` (64 `go.mod` files incl. root workspace + nested eventtest). Sub-packages (catalog/asyncapi, catalog/d2, catalog/openapi, catalog/eventcatalog, catalog/docserver, catalog/schema, storage/turso/indexing, signing/multisig, storage/eventstore, storage/readmodel) share their parent's `go.mod`.

| Module                         | Import Path                         | Maturity                                                                                                                                                                      |
| ------------------------------ | ----------------------------------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `event`                        | `…/event/v4`                        | ✅ Production                                                                                                                                                                 |
| `event/eventtest`              | `…/event/v4/eventtest`              | 🧪 Test helper                                                                                                                                                                |
| `command`                      | `…/command/v4`                      | ✅ Production                                                                                                                                                                 |
| `query`                        | `…/query/v4`                        | ✅ Production                                                                                                                                                                 |
| `decider`                      | `…/decider/v4`                      | ✅ Production                                                                                                                                                                 |
| `id`                           | `…/id/v4`                           | ✅ Production                                                                                                                                                                 |
| `dispatcher`                   | `…/dispatcher/v4`                   | ✅ Production                                                                                                                                                                 |
| `schema`                       | `…/schema/v4`                       | ✅ Production                                                                                                                                                                 |
| `snapshot`                     | `…/snapshot/v4`                     | ✅ Production                                                                                                                                                                 |
| `codec`                        | `…/codec/v4`                        | ✅ Production                                                                                                                                                                 |
| `kv`                           | `…/kv/v4`                           | ✅ Production                                                                                                                                                                 |
| `metadata`                     | `…/metadata/v4`                     | ✅ Production                                                                                                                                                                 |
| `dedup`                        | `…/dedup/v4`                        | ✅ Production                                                                                                                                                                 |
| `storage/memory`               | `…/storage/memory/v4`               | 🧪 Test utility                                                                                                                                                               |
| `catalog`                      | `…/catalog/v4`                      | ✅ Production                                                                                                                                                                 |
| `middleware`                   | `…/middleware/v4`                   | ✅ Production                                                                                                                                                                 |
| `integration`                  | `…/integration/v4`                  | ✅ Test suite                                                                                                                                                                 |
| `signing`                      | `…/signing/v4`                      | ✅ Production                                                                                                                                                                 |
| `encryption`                   | `…/encryption/v4`                   | ✅ Production                                                                                                                                                                 |
| `storage`                      | `…/storage/v4`                      | ✅ Production                                                                                                                                                                 |
| `storage/sql`                  | `…/storage/v4/sql`                  | 🧪 Shared infra                                                                                                                                                               |
| `watermill`                    | `…/watermill/v4`                    | ✅ Production                                                                                                                                                                 |
| `listing`                      | `…/listing/v4`                      | ✅ Production                                                                                                                                                                 |
| `otel`                         | `…/otel/v4`                         | ✅ Production                                                                                                                                                                 |
| `storage/pebble`               | `…/storage/pebble/v4`               | ✅ Production                                                                                                                                                                 |
| `storage/turso`                | `…/storage/turso/v4`                | ✅ Production                                                                                                                                                                 |
| `transport/http`               | `…/transport/http/v4`               | ✅ Production                                                                                                                                                                 |
| `transport/grpc`               | `…/transport/grpc/v4`               | ✅ Production                                                                                                                                                                 |
| `prometheus`                   | `…/prometheus/v4`                   | ✅ Production                                                                                                                                                                 |
| `testutil`                     | `…/testutil/v4`                     | 🧪 Test utility (`NewCmd`, `RaceEnabled` build-tag helper)                                                                                                                    |
| `cmd/cqrs-gen`                 | `…/cmd/cqrs-gen/v4`                 | 🔧 Tool                                                                                                                                                                       |
| `cmd/api-stability`            | `…/cmd/api-stability/v4`            | 🔧 Tool                                                                                                                                                                       |
| `cmd/doc-check`                | `…/cmd/doc-check/v4`                | 🔧 Tool                                                                                                                                                                       |
| `stack`                        | `…/stack/v4`                        | ✅ Production                                                                                                                                                                 |
| `stack/memory`                 | `…/stack/memory/v4`                 | ✅ Production                                                                                                                                                                 |
| `stack/sqlite`                 | `…/stack/sqlite/v4`                 | ✅ Production                                                                                                                                                                 |
| `stack/pebble`                 | `…/stack/pebble/v4`                 | ✅ Production                                                                                                                                                                 |
| `stack/postgres`               | `…/stack/postgres/v4`               | ⚠️ Partial (0% test coverage locally — skips without `POSTGRES_TEST_DSN`)                                                                                                     |
| `stack/turso`                  | `…/stack/turso/v4`                  | ✅ Production                                                                                                                                                                 |
| `stack/duckdb`                 | `…/stack/duckdb/v4`                 | ✅ Production (analytical OLAP, CGo required — ADR-0071)                                                                                                                      |
| `stack/mysql`                  | `…/stack/mysql/v4`                  | ⚠️ Partial (testcontainer privilege fix fragile; MySQL 8.0 contract tests pass, MariaDB untested)                                                                             |
| `stack/bench`                  | `…/stack/bench/v4`                  | 🧪 Benchmarks                                                                                                                                                                 |
| `stack/contracttest`           | `…/stack/contracttest/v4`           | 🧪 Test suite (RunSuite for cross-backend contract verification)                                                                                                              |
| `stack/sqlopt`                 | `…/stack/sqlopt/v4`                 | ✅ Production (shared SQL options: durability tiers, multi-DB topology)                                                                                                       |
| `deriver`                      | `…/deriver/v4`                      | ✅ Production                                                                                                                                                                 |
| `graph`                        | `…/graph/v4`                        | ✅ Production                                                                                                                                                                 |
| `idempotency`                  | `…/idempotency/v4`                  | ✅ Production                                                                                                                                                                 |
| `idempotency/kvstore`          | `…/idempotency/kvstore/v4`          | ✅ Production (KV-backed idempotency)                                                                                                                                         |
| `idempotency/sqlstore`         | `…/idempotency/sqlstore/v4`         | ✅ Production (SQL-backed: SQLite + Postgres, `INSERT ON CONFLICT` + TTL)                                                                                                     |
| `retry`                        | `…/retry/v4`                        | ✅ Production (zero-dep retry w/ backoff+jitter)                                                                                                                              |
| `projection`                   | `…/projection/v4`                   | ✅ Production                                                                                                                                                                 |
| `projectionhost`               | `…/projectionhost/v4`               | ✅ Production                                                                                                                                                                 |
| `scenario`                     | `…/scenario/v4`                     | ✅ Production                                                                                                                                                                 |
| `scheduling`                   | `…/scheduling/v4`                   | ✅ Production                                                                                                                                                                 |
| `example/taskmanager`          | `…/example/taskmanager`             | 💡 Demo                                                                                                                                                                       |
| `example/getting-started`      | `…/example/getting-started`         | 💡 Demo                                                                                                                                                                       |
| `example/readme-quickstart`    | `…/example/readme-quickstart`       | 💡 Demo                                                                                                                                                                       |
| `metaengine`                   | `…/metaengine/v4`                   | 🧪 Experimental (5 engines, 10 ADTs, rule pipeline, materialize-vs-replay, StorageLayout, SerializablePlan)                                                                   |
| `metaengine/projectionadapter` | `…/metaengine/projectionadapter/v4` | 🧪 Experimental (projection.Projection adapter for projectionhost)                                                                                                            |
| `metaengine/pebbleengine`      | `…/metaengine/pebbleengine/v4`      | 🧪 Experimental (Pebble LSM engine: MapBackend, ScanBackend, LayoutPlanner, sort index)                                                                                       |
| `metaengine/duckdbengine`      | `…/metaengine/duckdbengine/v4`      | 🧪 Experimental (DuckDB columnar engine: MapBackend, CounterBackend, PushdownScan. CGo)                                                                                       |
| `metaengine/pgengine`          | `…/metaengine/pgengine/v4`          | 🧪 Experimental (Postgres engine: MapBackend, CounterBackend, ScanBackend, PushdownScan, LayoutPlanner. Pure Go)                                                              |
| `metaengine/adttest`           | `…/metaengine/adttest/v4`           | 🧪 Test harness (RunMatrix cross-engine parity for 10 ADTs)                                                                                                                   |
| `flightrecorder`               | `…/flightrecorder/v4`               | 🧪 Experimental (Go 1.25 runtime/trace capture. Zero-dep. ADR-0089)                                                                                                           |
| `benchkit`                     | `…/benchkit/v4`                     | 🧪 Experimental (functional, 88 tests, `--repeat N` available)                                                                                                                |
| `cmd/cqrs-bench`               | `…/cmd/cqrs-bench`                  | 🔧 Tool                                                                                                                                                                       |
| `cmd/cqrs-lint`                | `…/cmd/cqrs-lint`                   | 🔧 Tool (181-rule domain-aware linter: correctness 36, API 31, boilerplate 28, adoption 21, architecture 17, consistency 16, performance 9, security 9, testing 8, version 6) |

---

## Architecture Guarantees

| Guarantee              | Detail                                                                                                                                                                                                                 |
| ---------------------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Lint posture           | `nix run .#lint` passes with 0 issues across all modules. `nix run .#verify` is GREEN: build + vet + test + race + lint + api-stability (3162 exports, with `TestEveryGoModDirIsInModulesList` meta-test) + doc-check. |
| Race-free              | `go test -race` passes across all modules                                                                                                                                                                              |
| Multi-module isolation | Each module has independent `go.mod`, no circular dependencies                                                                                                                                                         |
| Strong types           | `event.Event` is a concrete type alias (`= *ImmutableEvent`); core store/bus are interfaces for DI                                                                                                                     |
| Library, not framework | Import what you need, compose your own stack                                                                                                                                                                           |
| Context-aware          | All handlers accept `context.Context`                                                                                                                                                                                  |
| Errors as values       | Zero panics in production code, explicit error returns, classified sentinels via error-family taxonomy                                                                                                                 |
| Defensive copies       | All public accessors return copies — callers cannot mutate internals                                                                                                                                                   |
| Tombstone over delete  | Soft-delete via metadata — no `Delete` on Store                                                                                                                                                                        |

---

## Performance Benchmarks

> Session 142 (2026-06-02) — AMD RYZEN AI MAX+ 395, 96GB RAM, Go 1.26.3

| Module         | Benchmark      |  ns/op |   B/op | allocs/op |
| -------------- | -------------- | -----: | -----: | --------: |
| event          | NewEvent       |    201 |    384 |         3 |
| event          | DecodePayload  |    419 |    560 |        10 |
| id             | New            |    100 |     16 |         1 |
| id             | Parse          |     17 |      0 |         0 |
| command        | New            |     50 |    208 |         2 |
| query          | New            |    0.6 |      0 |         0 |
| dispatcher     | Dispatch       |     24 |      0 |         0 |
| memory         | Store Save     |    583 |    736 |         9 |
| memory         | Bus Publish    |     66 |     48 |         3 |
| signing        | HMAC Sign      |    662 |    864 |        12 |
| signing        | HMAC Verify    |    666 |    864 |        12 |
| signing        | Ed25519 Sign   | 13,486 |    416 |         7 |
| signing        | Ed25519 Verify | 30,369 |    352 |         6 |
| storage/SQLite | Save           | 41,042 |  4,080 |        92 |
| storage/SQLite | Load           | 48,505 | 20,233 |       554 |

Full results: `benchmarks/2026-06-02_20-18-40.md` · Regression pipeline: `scripts/benchstat-compare.sh`
