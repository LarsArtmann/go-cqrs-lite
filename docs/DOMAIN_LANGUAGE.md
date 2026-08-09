# Domain Language

A **Ubiquitous Language** for `go-cqrs-lite` — shared across library consumers, contributors, and AI.
Inspired by Domain-Driven Design (DDD).

Every term below should mean the **same thing** to everyone who reads it.
If a word means something different to a consumer than to an implementer, it is defined here.

> **Import convention:** All modules use the `/v4` import path suffix (e.g. `github.com/larsartmann/go-cqrs-lite/event/v4`). The `Context` column uses abbreviated package names (`event.`, `command.`, `storage.`) for readability — consumers must append `/v4` when importing.

> **How to use this file:**
>
> - Keep terms concise — one clear sentence per definition
> - Update when new domain concepts emerge
> - Use these terms consistently in code, docs, and conversations
> - When in doubt about a word's meaning, check here first

---

## Core Concepts

### Event Sourcing

| Term                 | Definition                                                                      | Context                                                                                                      |
| -------------------- | ------------------------------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------ |
| **Event**            | Immutable record of something that happened in the domain                       | `event.Event = *ImmutableEvent` — the single concrete implementation (not an interface)                      |
| **ImmutableEvent**   | The concrete event struct: ID, type, stream, version, payload, metadata         | `event.New()` (typed payload) or `event.NewEvent()` (raw bytes)                                              |
| **StreamRef**        | `{Type, ID}` — canonical identity of a stream instance                          | `id.NewStreamRef(type, id)` — passed to all Store methods. (`AggregateRef` is a deprecated alias)            |
| **StreamType**       | String category for a stream (e.g. `"User"`, `"Order"`)                         | `type StreamType string`. (`AggregateType` is a deprecated alias)                                            |
| **Stream**           | Ordered, append-only sequence of events for a single entity, ordered by Version | A stream is the fundamental unit of identity in event sourcing — what was previously called an "aggregate"   |
| **Version**          | Monotonically increasing position of an event within its stream (1-indexed)     | `type Version uint64` — used for optimistic concurrency                                                      |
| **Event Store**      | Append-only persistence layer for event streams                                 | `event.Store` (composite of `EventSink` + `EventSource`)                                                     |
| **Journal**          | Global append-only log of all events across all streams                         | `event.Journal.ReadAll()` — cross-stream reads                                                               |
| **SeekableJournal**  | Journal with position-based reading (by EventID)                                | `event.SeekableJournal.ReadFrom(afterEventID, limit)` — efficient projection catch-up                        |
| **BackwardsSource**  | Read-side that loads events in reverse (newest-first)                           | `event.BackwardsSource.LoadBackwards(ref)`                                                                   |
| **Snapshot**         | Point-in-time capture of stream state at a specific version                     | Avoids replaying the entire stream on every load                                                             |
| **Snapshot Store**   | Persistence layer for stream snapshots                                          | `snapshot.SnapshotStore` (composite of `SnapshotSink` + `SnapshotSource`)                                    |
| **Projection**       | Consumer-side contract for building a read model from events                    | `projection.Projection` — `Name()`, `Handle()`, `EventTypes()`                                               |
| **Checkpoint**       | Last-processed event position for a specific projection                         | `event.CheckpointStore` — enables resume after restart                                                       |
| **Tombstone**        | Soft-delete marker on event metadata (3 statuses)                               | `Active`, `Tombstoned`, `Undetermined` — detected via `event.DetectTombstone()`                              |
| **Rebirth**          | Undo of a tombstone — marks a stream as live again                              | `event.MarkRebirth(evt)` — sets rebirth metadata                                                             |
| **ProcessingMode**   | Context-scoped flag: `ModeLive` vs `ModeReplay`                                 | `event.WithProcessingMode(ctx, ModeReplay)` — lets handlers skip side-effects during catch-up                |
| **Metadata**         | Typed envelope on every event: tracing, causation, tombstone, custom fields     | `event.Metadata` struct — `Tracing`, `Causation`, `Tombstone`, `Custom map[MetadataKey]string`               |
| **Tracing**          | Embedded metadata fields: CorrelationID, CausationID, UserID, RequestID         | `event.Tracing` struct — promoted into `Metadata`, JSON-serializable                                         |
| **Causation**        | Links an event to the command that caused it (type + ID)                        | `event.Causation{CommandType, CommandID}` — set via `event.WithCommandCausality(ctx, type, id)`              |
| **ContextEnricher**  | Function that extracts metadata from context and stamps it onto new events      | `event.ContextEnricher` — `decider.Repository` applies it automatically on Save                              |
| **SnapshotStrategy** | Policy deciding when to persist a snapshot                                      | `snapshot.SnapshotStrategy` — `ShouldSnapshot(type, version) bool`; impl: `snapshot.EveryNEvents(n)`         |
| **Load Coalescing**  | Concurrent `Load` calls for the same stream coalesce into one store query       | `decider.Repository` uses `singleflight.Group` — transparent, disable via `WithLoadCoalescing[State](false)` |

### CQRS

| Term                 | Definition                                                          | Context                                                                                  |
| -------------------- | ------------------------------------------------------------------- | ---------------------------------------------------------------------------------------- |
| **Command**          | Intent to mutate state — dispatched to exactly one handler          | `command.Command` interface; `command.BasicCommand` (in-memory impl)                     |
| **PersistedCommand** | Serialized command record for durable audit trail                   | `command.NewPersistedCommand(type, ref, payload)` — stored via `CommandSink`             |
| **Query**            | Request for read-only data — returns a result, never mutates        | `query.Query` interface; `query.BasicQuery` (in-memory impl)                             |
| **PersistedQuery**   | Serialized query record for read-side audit                         | `query.NewPersistedQuery(type, payload)` — stored via `QuerySink`                        |
| **Dispatcher**       | Routes commands/queries to registered handlers by type string       | `command.Dispatcher`, `query.Dispatcher` — built on generic `dispatcher.Dispatcher[H,M]` |
| **Handler**          | Function that processes a command or query                          | `command.Handler = func(ctx, Command) error`; `query.Handler` returns `(any, error)`     |
| **TypedHandler**     | Compile-time type-safe handler — no manual type assertion           | `command.TypedHandler[T]`, `query.TypedHandler[Q,R]`                                     |
| **RegisterTyped**    | Registers a typed handler, wrapping it for the type-erased dispatch | `command.RegisterTyped[T]()`, `query.RegisterTyped[Q,R]()`                               |
| **DispatchTyped**    | Dispatches a query and asserts the result to type `T`               | `query.DispatchTyped[T](ctx, dispatcher, query)`                                         |
| **Decider**          | Pure-function stream: fold state from events                        | `decider.Decider[State]` — `Initial` + `Apply(state, evt) (State, error)`                |
| **TypedDecider**     | Decider with command type bound at compile time                     | `decider.TypedDecider[State, Cmd]` — carries `Decide` as a struct field                  |
| **Repository**       | Loads stream state, executes decider, saves and publishes events    | `decider.Repository[State]` — composes Store + Bus + optional Snapshot                   |
| **Bus** (event)      | Message bus for publishing/subscribing to events                    | `event.Bus` (Publisher + Subscriber + `Use`/`UsePublish` middleware)                     |
| **Bus** (command)    | Message bus for command pub/sub (queue-style dispatch)              | `command.Bus` (Publisher + Subscriber + `Use` middleware)                                |
| **MemoryBus**        | In-memory, synchronous command bus implementation                   | `command.NewMemoryBus()` — `Subscribe`, `SubscribeAll`, `Publish`, `Use`                 |
| **Deriver**          | Pure function transforming an event into zero or more commands      | `deriver.Deriver` — building block for saga/process-manager patterns                     |
| **Idempotent**       | Deriver combinator that stamps deterministic command IDs            | `deriver.Deriver.Idempotent()` — deduplication via idempotency store                     |

### Identity

| Term               | Definition                                                              | Context                                                                                                             |
| ------------------ | ----------------------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------- |
| **Branded ID**     | ULID-backed, phantom-typed identifier — compile-time type safety        | `id.Of[T] = cbid.ID[T, ulid.ULID]` — prevents mixing IDs of different types                                         |
| **StreamID**       | Branded **string-backed** ID for streams (the only non-ULID ID)         | `id.StreamID = cbid.ID[StreamMarker, string]` — accepts any non-empty string. (`AggregateID` is a deprecated alias) |
| **EventID**        | ULID-branded ID for events (time-sortable)                              | `id.EventID = id.Of[EventMarker]` — `id.NewEventID()`                                                               |
| **CorrelationID**  | ULID-branded ID linking related operations across a command→event chain | `id.CorrelationID = id.Of[CorrelationMarker]`                                                                       |
| **CausationID**    | ULID-branded ID tracking which event/command caused this one            | `id.CausationID = id.Of[CausationMarker]`                                                                           |
| **CommandID**      | ULID-branded ID for commands                                            | `id.CommandID = id.Of[CommandMarker]` — can be deterministically derived                                            |
| **Marker**         | Phantom struct type used as a type parameter for branding               | `StreamMarker`, `EventMarker`, `CorrelationMarker`, `CommandMarker`, etc.                                           |
| **DeriveStreamID** | Deterministic SHA-256-derived stream ID for idempotent workflows        | `id.DeriveStreamID(namespace, keys...)`                                                                             |

### Error Taxonomy

All errors are classified into a 6-family taxonomy:

| Family             | Meaning                                                | Constructor                          |
| ------------------ | ------------------------------------------------------ | ------------------------------------ |
| **Rejection**      | Business rule violation (4xx equivalent)               | `errorfamily.NewRejection(...)`      |
| **Conflict**       | Optimistic concurrency or duplicate (409 equivalent)   | `errorfamily.NewConflict(...)`       |
| **Transient**      | Retryable infrastructure failure (503 equivalent)      | `errorfamily.NewTransient(...)`      |
| **Infrastructure** | Non-retryable infrastructure failure (500 equivalent)  | `errorfamily.NewInfrastructure(...)` |
| **Corruption**     | Data integrity violation — human intervention required | `errorfamily.NewCorruption(...)`     |
| **Orchestration**  | Saga/workflow coordination failure (compound errors)   | `errorfamily.NewOrchestration(...)`  |

### Record (Structural Foundation)

Both Commands (intent, pre-decision) and Events (facts, post-decision) are **Records** — append-only, immutable entries in streams. The Record type is the shared structural base extracted in [ADR-0111](adr/0111-record-type-extraction.md), enabling the metaengine to treat events and commands uniformly.

| Term                 | Definition                                                                         | Context                                                                                                          |
| -------------------- | ---------------------------------------------------------------------------------- | ---------------------------------------------------------------------------------------------------------------- |
| **Record**           | The shared structural base for Commands and Events: Type, Payload, StreamID, Version, MetaData | `record.Record` struct — append-only, immutable; the metaengine consumes Records directly via `OnRecord` folds   |
| **StreamRef** (record) | `"StreamType/EntityID"` string — the primary key for event-sourced aggregates     | `record.StreamRef` type (`record.NewStreamRef(type, id)`) — distinct from `id.StreamRef` (branded ID) but same concept |
| **CommonMetadata**   | Metadata shared by all records: correlation, causation, actor, three timestamps   | `record.CommonMetadata` struct — replaces parallel metadata hierarchies in event/ and command/                  |
| **ActorID**          | Who or what produced a record: user ID, service name, cron job, or `"system"`     | `record.CommonMetadata.ActorID` — provenance tracking for audit and authorization                                |
| **ClientCreatedAt**  | Client's clock at creation — may lie (clock skew, offline tampering)              | `record.CommonMetadata.ClientCreatedAt` — for offline-first conflict resolution and UX                           |
| **ServerReceivedAt** | Server clock when the record arrived — trustworthy for server-side ordering       | `record.CommonMetadata.ServerReceivedAt` — set before store.Save                                                 |
| **ServerStoredAt**   | Database acknowledgment timestamp — for audit trails and EC reconciliation         | `record.CommonMetadata.ServerStoredAt` — what the DB reported, not necessarily what it did internally            |
| **SchemaVersion**    | Payload schema version, set once at creation, never changed — enables upcasting   | `record.CommonMetadata.SchemaVersion` — different versions of the same event type coexist in the stream          |

> **Record vs Event vs Command:** A Record is the structural shape. An Event is a Record of a fact (immutable truth). A Command is a Record of intent (may be rejected). The distinction is semantic, not structural — both share the same fields.

---

## Storage

| Term               | Definition                                                                           | Context                                                                                                                                             |
| ------------------ | ------------------------------------------------------------------------------------ | --------------------------------------------------------------------------------------------------------------------------------------------------- |
| **Dialect**        | SQL dialect abstraction for portable store implementations                           | `sql.Dialect` interface in `storage/sql` — `SQLiteDialect{}`, `PostgresDialect{}` (re-exported as `storage.SQLiteDialect`)                          |
| **SQLEventStore**  | SQL-backed implementation of `event.Store` + `Journal` + `SeekableJournal`           | `storage.NewSQLiteEventStore(db)`, `storage.NewSQLEventStore(db)`, `storage.NewSQLEventStoreWithDialect(db, dialect)`                               |
| **SQLBackend**     | Facade exposing all SQL stores sharing one `*sql.DB` (lazy, goroutine-safe)          | `storage.NewSQLiteBackend(db)`, `storage.NewSQLBackend(db)` — exposes EventStore, CommandStore, QueryStore, SnapshotStore, CheckpointStore, KVStore |
| **Pebble Store**   | Embedded KV event store (no SQL dependency)                                          | `pebble.NewStore(db, logger)` (package: `storage/pebble`)                                                                                           |
| **Pebble Backend** | Facade exposing all Pebble stores sharing one `*pebble.DB` via disjoint key prefixes | `pebble.Open(dir, opts, logger)` (owns DB) or `pebble.NewBackend(db, logger)` (borrows DB)                                                          |
| **Turso**          | Embedded Turso Database connector with sync support                                  | `turso.Open(dbPath)`, `turso.OpenInMemory()`, `turso.OpenSync()` (package: `storage/turso`)                                                         |
| **MemoryStore**    | In-memory implementations for testing                                                | `memory.NewMemoryStore()`, `memory.NewMemorySnapshotStore()` (package: `storage/memory`)                                                            |

---

## Stack Bundles

The primary consumer entry point. A **Bundle** wires all stores, buses, and journals into one struct — one constructor call per backend.

| Term               | Definition                                                       | Context                                                                                                          |
| ------------------ | ---------------------------------------------------------------- | ---------------------------------------------------------------------------------------------------------------- |
| **Bundle**         | Pre-wired facade: all stores + buses + journals in one struct    | `stack.Bundle` — `EventSink`, `EventSource`, `Journal`, `Publisher`, `CommandSink`, `QuerySink`, `KVStore`, etc. |
| **Stack Preset**   | One-call constructor that builds a Bundle for a specific backend | `sqlite.New(dsn)`, `memory.New()`, `pebble.New(dir)`, `postgres.New(dsn)`, `turso.New(dbPath)`                   |
| **ReadModel**      | One-call typed read-model store from a Bundle                    | `stack.ReadModel[T,K](bundle, nil)` — nil uses default codec (CBOR)                                              |
| **NewMaterialize** | One-call tombstone-aware projection builder from a Bundle        | `stack.NewMaterialize[T,K](bundle, nil, keyFunc)` — nil uses default codec                                       |
| **SQLViewModel**   | One-call SQL view store from a SQLite/Postgres Bundle            | `sqlite.SQLViewModel[V,K](bundle, mapper)`                                                                       |

> **When to use Stack vs raw modules:** Stack presets are the recommended entry point for new consumers. Use raw modules (`storage.NewSQLiteEventStore`, `pebble.NewStore`, etc.) when you need fine-grained control over which stores to create.

---

## Read Models

The library provides three projection tiers, chosen by read-pattern shape:

| Tier            | Builder                        | Store shape                 | Best for                                                             |
| --------------- | ------------------------------ | --------------------------- | -------------------------------------------------------------------- |
| **Document/KV** | `stack.Materialize[V,K]`       | One record per key          | Single-entity views (user profile, todo item)                        |
| **Relational**  | `storage.RelationalProjection` | Multiple related SQL tables | Multi-table atomic writes (messages + attachments + junction tables) |
| **Graph**       | `graph.GraphProjection`        | Nodes + edges               | Variable-depth traversal, adjacency, path-finding                    |

### KV Layer

| Term             | Definition                                                            | Context                                                                                           |
| ---------------- | --------------------------------------------------------------------- | ------------------------------------------------------------------------------------------------- |
| **KV Store**     | Byte-key/byte-value store abstraction with atomic batches             | `kv.Store` (Reader + Writer + Closer) — implemented by MemStore, Pebble, SQL backends             |
| **TypedStore**   | Typed read-model store over a `kv.Store`, serialized via codec        | `kv.NewTypedStore[T,K](backend)` — one per read-model type                                        |
| **Cache**        | Write-through LRU cache over a TypedStore                             | `kv.NewCache[T,K](store)` — TinyLFU eviction                                                      |
| **ViewStore**    | Typed read-model interface decoupling Materialize from storage        | `kv.ViewStore[V,K]` — `Get`, `Set`, `Delete`, `Scan`; both TypedStore and SQLViewStore satisfy it |
| **SQLViewStore** | `kv.ViewStore` backed by a dedicated SQL table with queryable columns | `storage.NewSQLiteViewStore[V,K](db, mapper)` — enables server-side WHERE/ORDER BY/LIMIT          |

### Projection Builders

| Term                     | Definition                                                              | Context                                                                             |
| ------------------------ | ----------------------------------------------------------------------- | ----------------------------------------------------------------------------------- |
| **Materialize**          | Document-tier projection: one record per key, tombstone-aware lifecycle | `stack.Materialize[V,K]` — `OnCreate`/`OnUpdate`/`OnTombstone`/`OnRebirth` handlers |
| **RelationalProjection** | Multi-table projection: atomic per-event writes across related tables   | `storage.NewRelationalProjection(name, schema, db, dialect, handler, types)`        |
| **RelationalStore**      | Dialect-agnostic read-side companion for relational projections         | `storage.NewRelationalStore(schema, db, dialect)` — Count, Query with conditions    |
| **GraphProjection**      | Graph-tier projection: merges events into nodes and edges               | `graph.NewGraphProjection(name, driver, handler, types)`                            |
| **GraphSchema**          | Closed-world validation for graph writes (opt-in)                       | `graph.Schema` — validates node labels, edge types at the sink boundary             |

### Projection Lifecycle

| Term                  | Definition                                                               | Context                                                                      |
| --------------------- | ------------------------------------------------------------------------ | ---------------------------------------------------------------------------- |
| **ProjectionHost**    | Managed host: one goroutine per projection, crash-restart with backoff   | `projectionhost.New(journal, checkpointStore)` — `Register`, `Start`, `Stop` |
| **DeadLetterStore**   | Captures poison messages that exhaust retries for later replay           | `projectionhost.DeadLetterStore` — `Store`, `List`, `Delete`, `Purge`        |
| **CatchUpSubscriber** | Replay+live handoff: historical events then seamless live transition     | `watermill.NewCatchUpSubscriber(journal, liveSub, cpStore, logger)`          |
| **StreamListing**     | Read model for listing streams with cursor pagination + tombstone status | `listing.InMemoryStreamReader` (from Journal) or `storage.SQLStreamReader`   |

---

## Metaengine

A **cost-based storage planner** (CBO) for event-sourced projections. Given a set of query declarations and available engines, the planner assigns each query to the cheapest engine and emits cost diagnostics. The core axiom: _every query pattern CAN be served by every storage engine — the question is never "can we?" but "at what cost?"_ The metaengine is the strategic future of this project ([design docs](planning/meta-engine-design.md), ADRs [0061](adr/0061-metaengine-sqlite-engine.md)–[0117](adr/0117-command-lifecycle-as-events.md)).

> **Relationship to Read Models:** The three projection tiers above (Document/KV, Relational, Graph) are _manual_ — the consumer hand-writes each projection. The metaengine _automates_ this: it infers the ADT from fold return types, picks the optimal engine, generates DDL, and routes queries — 80% auto-generated, 100% auto-routed ([ADR-0116](adr/0116-layered-auto-projection.md)).

### Core Concepts

| Term                   | Definition                                                                                  | Context                                                                                                          |
| ---------------------- | ------------------------------------------------------------------------------------------- | ---------------------------------------------------------------------------------------------------------------- |
| **Engine**             | A storage backend with a cost profile; the unit the optimizer ranks and assigns queries to  | `metaengine.Engine` interface — `Profile() EngineProfile` + `Closer`; impls: Memory, SQLite, Pebble, DuckDB, PG |
| **EngineProfile**      | Declares what an engine can do (ADTs + Complexity), how fast (calibrated ns/op), layout, persistence, replication | `metaengine.EngineProfile` struct — the engine's "datasheet"                                            |
| **Store**              | The running, planned runtime: holds engines, registered queries, the plan, and event log   | `metaengine.Store` — created by `Plan()`, queried via `ExecuteTyped`                                              |
| **Plan / PlanResult**  | The optimizer's output: one `QueryAssignment` per query + diagnostics + layout plans        | `metaengine.PlanResult` — produced by `metaengine.Plan(engines, queries...)`                                     |
| **QueryAssignment**    | The per-query plan decision: engine, ADT, read pattern, complexity, cost, diagnostics       | `metaengine.QueryAssignment` struct — one per declared query                                                     |
| **Collection**         | A planned query's materialized projection — the named, queryable view an engine serves      | `metaengine.CollectionInfo` — introspect via `Store.Collections()`                                               |
| **QueryDecl**          | A fully-analyzed query declaration: name, folds, inferred ADT, read pattern, config         | `metaengine.QueryDecl[Q,R]` — created via `metaengine.Query[Q,R](name, folds...)`                                |
| **QueryConfig**        | Declarative per-query tuning: Volume, LatencyBudget, TTL, filter/sort fields, columnarLayout | `metaengine.QueryConfig` — set via `QueryOption` funcs like `metaengine.Volume(n)`                               |

### ADTs (Abstract Data Types)

An **ADT** is the logical data structure the planner infers from a query's fold return types. It determines which engines can serve the query and at what complexity. Inference happens automatically via fold classification.

| ADT              | Operations                          | Key Trait                    | Constant                          |
| ---------------- | ----------------------------------- | ---------------------------- | --------------------------------- |
| **Map**          | Get/Set/Delete by key               | Keys unique                  | `metaengine.ADTMap`               |
| **Sorted Map**   | Map ops + Range/Filter/OrderBy      | Ordered                      | `metaengine.ADTSortedMap`         |
| **Multimap**     | Add(key,val)/GetAll(key)            | One key → many values        | `metaengine.ADTMultimap`          |
| **Counter**      | Increment/Get                       | Numeric aggregation          | `metaengine.ADTCounter`           |
| **Set**          | Add/Contains/Members                | Values unique                | `metaengine.ADTSet`               |
| **Log**          | Append/ReadFrom (ordered)           | Append-only sequence         | `metaengine.ADTLog`               |
| **Stream Log**   | Stream-keyed append (ES primitive)  | Per-stream ordering          | `metaengine.ADTStreamLog`         |
| **Graph**        | AddEdge/Neighbors/Traverse          | Edge traversal               | `metaengine.ADTGraph`             |
| **Vector**       | Insert embedding / k-NN search      | Similarity (cosine/euclidean) | `metaengine.ADTVector`           |
| **Search**       | Insert document / full-text query   | Inverted index (TF-IDF/BM25) | `metaengine.ADTSearch`            |
| **Spatial**      | Insert point / range proximity      | Geo distance (haversine)     | `metaengine.ADTSpatial`           |

Each ADT maps to an optional **Backend interface** an engine implements (ISP): `MapBackend`, `SetBackend`, `CounterBackend`, `MultimapBackend`, `LogBackend`, `StreamLogBackend`, `GraphBackend`, `VectorBackend`, `SearchBackend`, `SpatialBackend`. An engine that lacks the native backend for an ADT may still serve it via **degraded** brute-force fallback (with a `DEGRADED` diagnostic).

> **Typed fold inputs:** Vector and Search ADTs have dedicated input types: `metaengine.Embedding` (ID + `[]float32` values) for k-NN similarity folds, and `metaengine.IndexedText` (ID + content string) for full-text search folds.

### Fold DSL (Event → Projection Mapping)

A **Fold** is a single typed event-to-projection update rule: "when event E happens, update the projection like this." Folds are the write path. The planner inspects fold return types to infer the ADT.

| Term           | Definition                                                                       | Context                                                                                |
| -------------- | -------------------------------------------------------------------------------- | -------------------------------------------------------------------------------------- |
| **Fold**       | A sealed interface representing one event-to-projection update rule              | `metaengine.Fold` — created only via `On`/`OnTyped`/`OnRecord`                         |
| **On**         | Constructor binding an event type to a handler closure; classifies the fold kind | `metaengine.On[E](sample, handler)` — infers ADT from handler return type             |
| **OnTyped**    | Like `On` but binds to an explicit event-type string (for external schemas)      | `metaengine.OnTyped[E](eventType, sample, handler)`                                    |
| **OnRecord**   | Creates a Record-aware fold: handler receives the full `record.Record` context   | `metaengine.OnRecord[E](sample, handler)` — access StreamID, Version, MetaData         |
| **FoldKind**   | Classifies what a fold does: insert, update, remove, count, edge, set, skip, etc. | `metaengine.FoldKind` — `FoldInsert`, `FoldUpdate`, `FoldCount`, `FoldEdge`, etc.    |
| **Delta**      | Counter update: `map[string]int64` of key → delta                                | `metaengine.Delta` — return type for Counter ADT folds                                 |
| **Edge**       | Graph edge (From, To)                                                             | `metaengine.Edge` struct — return type for Graph ADT folds                             |
| **MultiEntry** | Sentinel return type classifying a fold as a multimap insert (Key + Value)       | `metaengine.MultiEntry` struct                                                         |
| **Skip**       | Sentinel signaling an event does not apply to this projection (no-op)            | `metaengine.Skip` struct — return `metaengine.Skip{}` to ignore                        |
| **Remove**     | Constructor for a delete-by-key fold                                              | `metaengine.Remove[V]()`                                                               |
| **Poison**     | A collection refuses reads after a fold panic; error stored until store recreate | `Store.IsPoisoned(collection)` — quarantine mechanism                                  |
| **AutoInsert** | Reflection-based fold: inserts a new record from event fields, auto-stamping Record metadata | `metaengine.AutoInsert[E, R](sample, eventType)` — no hand-written handler needed |
| **AutoUpdate** | Reflection-based fold: updates an existing record's fields from event            | `metaengine.AutoUpdate[E, R](sample, eventType)` — field-by-field merge via reflection |
| **AutoDelete** | Reflection-based fold: marks a record for deletion by key                        | `metaengine.AutoDelete[E](sample, eventType)`                                          |
| **AutoCRUD**   | Combines AutoInsert + AutoUpdate + AutoDelete into one declaration               | `metaengine.AutoCRUD[C, U, D, R](create, update, delete, result)` — full lifecycle    |

### Cost Model

The cost model estimates how expensive serving a query on a given engine is, using Big-O complexity classes and calibrated nanoseconds-per-operation.

| Term                     | Definition                                                                          | Context                                                                                            |
| ------------------------ | ----------------------------------------------------------------------------------- | -------------------------------------------------------------------------------------------------- |
| **Complexity**           | A Big-O complexity class for cost estimation                                        | `metaengine.Complexity` — `ComplexityO1`, `ComplexityOLogN`, `ComplexityON`, `ComplexityONLogN`, `ComplexityODegree` |
| **CostEstimate**         | The estimated cost of serving one query: complexity, volume, ops, latency (ms)      | `metaengine.CostEstimate` — `WithinBudget()` checks against LatencyBudget                           |
| **NsPerOp**              | Calibrated nanoseconds-per-operation from benchmark calibration                     | `EngineProfile.NsPerOp`, `NsPerRead`, `NsPerWrite` — real measurements, not theoretical            |
| **ReadCosts**            | Per-read-pattern calibrated costs (point lookup vs scan vs aggregate)               | `metaengine.ReadCosts` — engines span 4000× across operations                                      |
| **ScaleThreshold**       | The optimal cardinality range for a data structure; planner warns when exceeded     | `metaengine.ScaleThreshold` — e.g. hash map warns past 10M entries                                 |
| **Volume**               | The expected number of items in a projection, used as N in cost formulas            | `metaengine.Volume(n)` QueryOption                                                                  |
| **LatencyBudget**        | Target latency ceiling for engine selection; planner flags when unmet               | `metaengine.WithLatencyBudget(ms)` QueryOption                                                      |
| **WriteAmplification**   | The cost of read optimization: each projection an event updates increases write cost | `metaengine.DefaultWriteAmplificationBudget` (3) — planner warns when exceeded                     |

### Storage Layouts

A **StorageLayout** describes the physical storage structure — it lets the planner reason about _why_ one engine beats another for a given access pattern.

| Layout         | Optimal for                          | Used by                          | Constant                    |
| -------------- | ------------------------------------ | -------------------------------- | --------------------------- |
| **Row**        | Point lookups, single-record updates | SQLite (B-Tree), Memory          | `metaengine.LayoutRow`      |
| **Columnar**   | Aggregations, field-subset scans     | DuckDB                           | `metaengine.LayoutColumnar` |
| **LSM**        | Write-heavy with point reads         | Pebble, Badger                   | `metaengine.LayoutLSM`      |
| **KV**         | Simple point lookups                 | Memory (hash map), generic KV    | `metaengine.LayoutKV`       |

**Layout Planning (Level 2 optimization — within-engine):**

| Term                  | Definition                                                                  | Context                                                                                  |
| --------------------- | --------------------------------------------------------------------------- | ---------------------------------------------------------------------------------------- |
| **LayoutPlan**        | A planned table schema: extracts JSON fields into indexed SQL columns       | `metaengine.LayoutPlan` — replaces `json_extract()` scans with indexed column reads      |
| **PlannedColumn**     | An extracted column (JSON field name → SQL type)                            | `metaengine.PlannedColumn` struct                                                        |
| **LayoutPlanner**     | Engine capability: can create optimized table layouts for filter/sort fields | `metaengine.LayoutPlanner` interface — implemented by SQLite, DuckDB, Postgres          |
| **LayoutPlanApplier** | Extension receiving fully-built LayoutPlan with reflection-derived types    | `metaengine.LayoutPlanApplier` — DuckDB columnar-native extraction                        |
| **WithColumnarLayout** | QueryOption requesting full columnar extraction of ALL exported fields     | `metaengine.WithColumnarLayout()` — vectorized GROUP BY/SUM/AVG on DuckDB                |

### Read Patterns

A **ReadPattern** describes how a query reads its projection — distinct from the ADT's data-structure complexity. The cost model adjusts complexity per read pattern (a hash map O(1) for point lookup still scans O(N) for filtered scans).

| Read Pattern          | Meaning                              | Constant                              |
| --------------------- | ------------------------------------ | ------------------------------------- |
| **Point Lookup**      | Single key → value                   | `metaengine.ReadPointLookup`          |
| **Membership**        | Key exists in set?                   | `metaengine.ReadMembership`           |
| **Multi-Lookup**      | Batch key → values                   | `metaengine.ReadMultiLookup`          |
| **Filtered Scan**     | WHERE predicate over collection      | `metaengine.ReadFilteredScan`         |
| **Aggregate**         | COUNT/SUM/MIN/MAX/AVG                | `metaengine.ReadAggregate`            |
| **Traversal**         | Graph neighbor traversal             | `metaengine.ReadTraversal`            |
| **Scan**              | Full collection scan                 | `metaengine.ReadScan`                 |
| **Log Tail**          | Read latest N entries from append log | `metaengine.ReadLogTail`             |
| **Vector Search**     | k-NN similarity search               | `metaengine.ReadVectorSearch`         |
| **Full-Text Search**  | TF-IDF/BM25 ranked query             | `metaengine.ReadFullTextSearch`       |
| **Spatial Range**     | Geo proximity query (haversine)      | `metaengine.ReadSpatialRange`         |

### Filter, Sort & Pagination

| Term             | Definition                                                                        | Context                                                                                          |
| ---------------- | --------------------------------------------------------------------------------- | ------------------------------------------------------------------------------------------------ |
| **FilterSpec**   | Declarative filter pushdown-able to SQL (Column + Op + Value → `json_extract` WHERE) | `metaengine.FilterSpec` struct                                                              |
| **SortSpec**     | Declarative sort pushdown-able to SQL (Column + Desc → ORDER BY)                  | `metaengine.SortSpec` struct                                                                     |
| **FilterOnField** | Declare a filter by declarative field name (SQL pushdown)                        | `metaengine.FilterOnField[R](field, op)` — op: `metaengine.FilterEq`, `FilterLt`, etc.          |
| **SortOnField**  | Declare sort by declarative field name (SQL pushdown)                             | `metaengine.SortOnField[R](field, desc)`                                                         |
| **FilterOn**     | Declare a filter via typed closure (Go-side filtering)                            | `metaengine.FilterOn[R,T](accessor)` — for non-SQL engines                                      |
| **Cursor**       | Position marker for keyset (offset-free) pagination; encodes to URL-safe base64   | `metaengine.Cursor` struct — `ParseCursor(s)`                                                    |
| **TypedReader**  | Typed read access to a collection without constructing a query input struct       | `metaengine.NewReader[V](store, coll)` — Get/Scan/Count/Sum                                     |
| **QueryBuilder** | Fluent, chainable API for building scans (Where/OrderBy/Limit/Cursor/Execute)    | `metaengine.NewQueryBuilder[V](reader)`                                                          |

### PlanRule Pipeline

Rules run **after** engine assignment — they enrich the `PlanResult` (diagnostics, layout plans) but never override engine selection. This makes plans debuggable: "why was this query assigned here?"

| Term              | Definition                                                                   | Context                                                                                  |
| ----------------- | ---------------------------------------------------------------------------- | ---------------------------------------------------------------------------------------- |
| **PlanRule**      | A single composable post-assignment planning decision                        | `metaengine.PlanRule` interface — appends diagnostics or layout plans                    |
| **RulePipeline**  | Applies a sequence of PlanRules in order; aborts on first error              | `metaengine.RulePipeline` — `NewRulePipeline(rules...)`                                  |
| **RuleTraceEntry** | Records one rule's decision and reason, making EXPLAIN output debuggable    | `metaengine.RuleTraceEntry` struct                                                       |
| **Diagnostic**    | A plan-time message at one of four severity levels                          | `metaengine.Diagnostic` struct — Level + Query + Message                                 |

**Diagnostic levels** (escalating severity):

| Level        | Meaning                                                                              | Constant                  |
| ------------ | ------------------------------------------------------------------------------------ | ------------------------- |
| **SCREAM**   | Configuration that will cause permanent data loss — the store refuses to start       | `metaengine.DiagLevelScream`  |
| **DEGRADED** | Engine serves the ADT via brute-force fallback (works but slow)                      | `metaengine.DiagLevelDegraded` |
| **WARN**     | Suboptimal but not data-threatening configuration                                    | `metaengine.DiagLevelWarn`    |
| **INFO**     | Advisory note (e.g., materialize-vs-replay recommendation)                          | `metaengine.DiagLevelInfo`    |

### Persistence Model

**Persistence** (DDIA Ch1: survivability) answers one binary question: _if the process exits, is the data gone?_

| Term           | Definition                                                                     | Constant                          |
| -------------- | ------------------------------------------------------------------------------ | --------------------------------- |
| **Volatile**   | Data lives in process RAM and is lost on exit (the safe zero-value default)    | `metaengine.PersistenceVolatile`  |
| **Persistent** | Data survives process exit via disk file or remote server                      | `metaengine.PersistencePersistent` |

The zero value is `PersistenceVolatile` — forgetting to set it causes a WARN (no silent data loss). SQLite/Pebble/DuckDB set it dynamically: volatile for in-memory constructors, persistent for file/DB constructors.

### Replication Model

**Replication** (DDIA Ch5) declares how an engine's data propagates across process boundaries. All CQRS read models are eventually consistent; the only strongly-consistent operation is the event store's optimistic-concurrency append.

| Term               | Definition                                                                  | Constant                              |
| ------------------ | --------------------------------------------------------------------------- | ------------------------------------- |
| **None**           | Data stays in this process (zero-value default for all current engines)     | `metaengine.ReplicationNone`          |
| **Single-Leader**  | Writes to one leader, propagate to followers async (Postgres streaming)     | `metaengine.ReplicationSingleLeader`  |
| **Multi-Leader**   | Multiple nodes accept writes, reconcile via consensus (CockroachDB)         | `metaengine.ReplicationMultiLeader`   |
| **Leaderless**     | Any node accepts writes, converge via CRDT merge (Iroh, Dynamo)             | `metaengine.ReplicationLeaderless`    |
| **Replication Lag** | Expected delay between write on one node and visibility on another (freshness, NOT latency) | `EngineProfile.ReplicationLag`  |
| **Network RTT**    | Round-trip time to reach the engine's data (additive fixed latency, 0 for in-process) | `EngineProfile.NetworkRTT`      |

### Materialize vs Replay

The planner can recommend whether a projection should be **materialized** (persisted, maintained incrementally) or **replayed** on-demand (fold the stream per query). This is the ES-specific killer feature.

| Term                  | Definition                                                          | Context                                              |
| --------------------- | ------------------------------------------------------------------- | ---------------------------------------------------- |
| **WorkloadStats**     | Observed workload: write rate, read rate, avg stream length         | `metaengine.WorkloadStats` struct                    |
| **ReplayCost**        | Cost of replaying a stream per query: `read_rate × stream_len × fold_cost` | `metaengine.ReplayCost(stats)`                 |
| **MaterializeCost**   | Cost of maintaining a materialized projection                       | `metaengine.MaterializeCost(stats)`                  |
| **ShouldMaterialize** | Returns true when materialization cost < replay cost               | `metaengine.ShouldMaterialize(stats)` — advisory     |

### Temporal Reads (As-Of)

| Term                  | Definition                                                          | Context                                              |
| --------------------- | ------------------------------------------------------------------- | ---------------------------------------------------- |
| **VersionedStorage**  | Engine capability for temporal point lookups: "value of K at time T?" | `metaengine.VersionedStorage` interface — Memory engine |
| **AsOfSignal**        | Marker type in a query input that triggers temporal routing         | `metaengine.AsOfSignal` struct                       |

### Plan Operations

| Term                  | Definition                                                          | Context                                                                                  |
| --------------------- | ------------------------------------------------------------------- | ---------------------------------------------------------------------------------------- |
| **SerializablePlan**  | A plan serialized to JSON for drift detection across deploys        | `metaengine.SerializablePlan` — `Serialize()`, `DeserializePlan(data)`                  |
| **PlanDiff**          | Compares two plans: added/removed/changed engines, queries, layouts | `metaengine.PlanDiff(prev, current)` → `PlanDiffResult`                                  |
| **Manifest**          | A plan fingerprint saved to disk for CI drift detection             | `metaengine.NewManifest(plan)`, `SaveManifest(path)`, `LoadManifest(path)`               |
| **DryRun**            | Plan option that returns PlanResult without modifying engine state  | `metaengine.WithDryRun()` — no DDL, no pinning                                          |
| **ExplainPlan**       | Human-readable plan explanation: engine per query + diagnostics      | `Store.ExplainPlan()` — string output for debugging                                     |
| **Doctor**            | Health report: per-collection engine, replication, persistence      | `Store.Doctor(ctx)` — string output for operations                                      |

### Engine Capabilities (Optional Interfaces)

Engines implement optional capability interfaces (ISP) — the planner checks at runtime which are present:

| Term                | Definition                                                                  | Context                                                |
| ------------------- | --------------------------------------------------------------------------- | ------------------------------------------------------ |
| **PushdownScan**    | Pushes filter/sort/limit into the engine's query language (SQL WHERE/ORDER BY) | `metaengine.PushdownScan` interface — SQLite, DuckDB, PG |
| **StreamingScan**   | OOM-safe lazy iteration via `iter.Seq2`                                     | `metaengine.StreamingScan` interface                   |
| **RawValueReader**  | Reads raw JSON bytes without decoding to `any` (avoids double-decode tax)   | `metaengine.RawValueReader` interface                  |
| **AtomicAppender**  | Atomic version-check-then-append (optimistic concurrency on streams)        | `metaengine.AtomicAppender` interface                  |
| **SnapshotBackend** | Engine capability for storing decider snapshots                             | `metaengine.SnapshotBackend` interface                 |
| **MapUpdater**      | Atomic read-modify-write for Map ADT (transactional)                        | `metaengine.MapUpdater` interface                      |
| **AggregateReader** | SQL-level aggregation avoiding loading all rows into Go                     | `metaengine.AggregateReader` interface                 |
| **GroupedAggregateReader** | SQL-level GROUP BY aggregation (vectorized on columnar engines)      | `metaengine.GroupedAggregateReader` interface           |
| **MultiAggregateReader** | Multiple aggregate queries in one call (batch optimization)            | `metaengine.MultiAggregateReader` interface             |

### Hot Operations

| Term            | Definition                                                          | Context                                              |
| --------------- | ------------------------------------------------------------------- | ---------------------------------------------------- |
| **TieredStore** | Wraps a primary Store with replicas; writes fan out, reads use primary | `metaengine.NewTieredStore(primary, replicas...)` |
| **Watcher**     | Subscribe to real-time value updates on a collection (push notifications) | `metaengine.NewWatcher[V](store, coll)`         |
| **SwapEngine**  | Replaces an engine at runtime, reassigning queries — zero-downtime upgrades | `Store.SwapEngine(name, newEngine)`             |
| **PrefetchCache** | Caches scan results beyond requested limit for next page           | `metaengine.PrefetchCache` — eliminates redundant round-trips |
| **MapUpdateTyped** | Typed atomic read-modify-write for a single Map key (transactional) | `metaengine.MapUpdateTyped[V](store, coll, key, fn)` — fn receives `*V`, returns updated value |

### Engines

Concrete `Engine` implementations, each in its own subpackage:

| Engine     | Type                      | Module                    |
| ---------- | ------------------------- | ------------------------- |
| **Memory** | In-process hash map; volatile; O(1) point lookup | `metaengine.NewMemoryEngine()` |
| **SQLite** | B-Tree; persistent; O(logN); SQL pushdown | `metaengine/sqliteengine/`       |
| **Pebble** | LSM tree; persistent; write-optimized     | `metaengine/pebbleengine/`       |
| **Badger** | LSM tree; persistent                      | `metaengine/badgerengine/`       |
| **DuckDB** | Columnar OLAP; persistent; O(1) aggregation (CGo) | `metaengine/duckdbengine/` |
| **Postgres** | Client-server OLTP; persistent; streaming replication | `metaengine/pgengine/`   |
| **Dgraph** | Graph-native; persistent                   | `metaengine/dgraphengine/`       |
| **Iroh**   | Leaderless CRDT replication (eventual convergence) | `metaengine/irohengine/`    |

---

## Messaging & Transport

| Term                 | Definition                                                                                              | Context                                                                         |
| -------------------- | ------------------------------------------------------------------------------------------------------- | ------------------------------------------------------------------------------- |
| **EventBus**         | `event.Bus` backed by Watermill GoChannel or injected broker                                            | `watermill.NewEventBus()` — single-process default; Kafka/NATS/Redis injectable |
| **CommandBus**       | `command.Bus` backed by Watermill GoChannel                                                             | `watermill.NewCommandBus()` — command distribution over any broker              |
| **EventPublisher**   | Wraps a Watermill `message.Publisher` as `event.Publisher`                                              | `watermill.NewEventPublisher(wmPublisher, topic)` — injects W3C trace context   |
| **CommandPublisher** | Wraps a Watermill `message.Publisher` as `command.Publisher`                                            | `watermill.NewCommandPublisher(wmPublisher, topic)`                             |
| **SSE Broker**       | Bridges `event.Bus` to HTTP clients via Server-Sent Events                                              | `http.NewSSEBroker(bus)` — `WithReconnectJournal` enables Last-Event-ID replay  |
| **gRPC Transport**   | Remote command/query/event dispatch over gRPC                                                           | `grpc.RegisterCommandService(srv, dispatcher)`, `grpc.RegisterQueryService()`   |
| **Saga**             | _Not a module._ Multi-step orchestration emerges from `bus.SubscribeAll` + command dispatch + `deriver` | See `example/taskmanager/` for the pattern                                      |

---

## Security

| Term               | Definition                                                              | Context                                                                                     |
| ------------------ | ----------------------------------------------------------------------- | ------------------------------------------------------------------------------------------- |
| **Signing**        | Tamper-proof event streams via HMAC-SHA256 or Ed25519 signatures        | `signing.NewHMAC(key)`, `signing.NewEd25519(privKey)` — `SignMiddleware`/`VerifyMiddleware` |
| **Encryption**     | Confidential event payloads via XChaCha20-Poly1305 or AES-256-GCM       | `encryption.NewXChaCha20Poly1305(key)` — `EncryptMiddleware`/`DecryptMiddleware`            |
| **Codec**          | Payload serialization abstraction (JSON, CBOR, Raw)                     | `codec.JSONCodec{}`, `codec.CBORCodec{}`, `codec.CBORCompactCodec{}`, `codec.RawCodec{}`    |
| **Encoding stamp** | Each event records its codec (`Encoding()`) for self-describing streams | `event.DecodePayloadAuto[T]` dispatches by stamp — mixed JSON+CBOR streams decode correctly |

> **Codec default asymmetry:** Events are self-describing (each stamps its `Encoding()`), so mixed streams decode correctly. Blind stores (KV, snapshots, command/query payloads) have no encoding stamp — changing their default codec would silently break existing data.

---

## Cross-Cutting

| Term                    | Definition                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                               | Context                                                                                                                                                                                                                                                 |
| ----------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| **Idempotent Receiver** | Deduplication for safe retries under at-least-once delivery (DDIA pattern). Contract: `Record(k, ttl)` is a **no-op on an existing key** (the TTL is never extended); `CheckAndRecord(k, ttl)` atomically claims a key and returns `ErrDuplicate` on conflict (an expired key is reclaimed in the same statement); `Seen(k)` reports membership and **lazily deletes** expired entries. Three implementations share this contract: `idempotency.MemoryStore` (single-process, optional background sweep), `idempotency/kvstore.Store` (any `kv.Store` + `ConditionalWriter`), `idempotency/sqlstore.Store` (multi-process SQLite/Postgres). The contract is enforced across all three by `idempotency/kvstore.TestStore_Record_MatchesMemoryStoreContract` and `TestStore_CheckAndRecord_Concurrent_AllImplementations`. | `idempotency.Store` — `Record`, `CheckAndRecord`, `Seen`; `idempotency.ErrDuplicate`; `dedup.Ring` — bounded O(1) dedup at stream boundaries; middleware: `middleware.CommandIdempotency`, `middleware.EventIdempotency`, `middleware.QueryIdempotency` |
| **Timer**               | Durable deadline: "cancel order after 30 min unpaid"                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                     | `scheduling.TimerStore[P]` — `Schedule` (idempotent), `Due`, `MarkFired`, `Cancel`                                                                                                                                                                      |
| **Scheduler**           | Polls a TimerStore and dispatches due timers with retry                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                  | `scheduling.New(store, dispatchFunc)` — exponential backoff, max retries                                                                                                                                                                                |
| **Catalog**             | Auto-generates AsyncAPI, OpenAPI, D2, and EventCatalog docs                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                              | `catalog.Registry` + `catalog.SchemaFromType[T]()` + per-format exporters                                                                                                                                                                               |
| **Middleware**          | Cross-cutting concerns: Logging, Retry, Recovery, Validation, Metrics, Tracing, Circuit Breaker, Idempotency                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                             | `middleware/` — 3 variants each (Command*/Event*/Query\*)                                                                                                                                                                                               |
| **OTel**                | OpenTelemetry tracing + metrics re-exports                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                               | `otel.Setup()` — one-call provider; `otel.NewTracer()`, `otel.NewMeter()`                                                                                                                                                                               |
| **Prometheus**          | OTel→Prometheus metrics bridge with `/metrics` HTTP handler                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                              | `prometheus.Setup()`                                                                                                                                                                                                                                    |
| **Schema Evolution**    | Forward/backward-compatible event evolution via upcasting on read (DDIA Ch. 4)                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                           | `schema.Upcaster`, `schema.VersionedStore`, `schema.VersionedSeekableJournal` — events are immutable; old versions transformed at load time                                                                                                             |
| **Validator**           | Runtime type registration for event payload validation                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                   | `schema.Validator` — `RegisterType[T]()`                                                                                                                                                                                                                |
| **Circuit Breaker**     | Stops cascading failures by opening after N consecutive errors; half-open probe recovery                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                 | `middleware.NewCircuitBreaker[M]()`, `middleware.CircuitBreakerConfig`, `middleware.ErrCircuitBreakerOpen`; variants: `CommandCircuitBreaker`, `EventCircuitBreaker`, `QueryCircuitBreaker`                                                             |
| **Retry**               | Zero-dep retry with exponential backoff + jitter (up to 50%) for transient failures                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                      | `retry.Do(ctx, config, fn)`, `retry.Config`, `retry.Backoff`, `retry.ErrExhausted`, `retry.ErrCanceled`; middleware: `middleware.NewRetry[M]()`                                                                                                         |
| **Dedup Ring**          | Fixed-capacity O(1) ring buffer for deduplicating event IDs at replay→live boundaries                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                    | `dedup.Ring`, `dedup.NewRing(capacity)`, `dedup.DefaultCapacity` (1024) — used by `projectionhost` and `watermill.CatchUpSubscriber`                                                                                                                    |
| **Projection Lag**      | Time duration between newest event in journal and last-processed event per projection                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                    | `projectionhost.LagDuration()` (max across workers), `projectionhost.LagPerProjection()` (per-worker map) — for dashboards and health checks                                                                                                            |
| **Heartbeat**           | Keep-alive comment frames on SSE connections to prevent proxy idle timeouts                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                              | `transport/http.DefaultSSEHeartbeat` (15s), `transport/http.WriteSSEHeartbeat(w)` — SSE spec comment frames                                                                                                                                             |
| **Backfill**            | REST endpoint for fetching missed events by position (complement to SSE reconnection)                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                    | `transport/http.BackfillHandler(broker)` — `GET /events/backfill?after=<id>&limit=500` — applies broker's `WithPayloadTransform`                                                                                                                        |
| **BufferEncoder**       | Optional codec interface for zero-allocation encoding into a caller-provided buffer                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                      | `codec.BufferEncoder` — `EncodeToBuffer(payload, buf)` — implemented by `JSONCodec`, `CBORCodec`, `CBORCompactCodec`                                                                                                                                    |
| **Materialized View**   | Read model derived from the event log; rebuildable from events (DDIA "derived data")                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                     | `stack.Materialize[V,K]` (KV/document), `storage.RelationalProjection` (SQL), `graph.GraphProjection` (graph) — all implement `projection.Projection`                                                                                                   |
| **High-Water Mark**     | DDIA term for the maximum safely-processed position in a stream; this library calls it "Checkpoint"                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                      | `event.CheckpointStore` — the library's checkpoint IS a high-water mark; per-projection, resumable after restart                                                                                                                                        |

> **Metaengine** terms (Engine, ADT, Fold, Cost Model, Layout, PlanRule, etc.) are defined in the [Metaengine](#metaengine) section above.

---

## Deployment Scope

This library is designed for **single-process applications** (embedded SQLite, Pebble) and **multi-process deployments** (Postgres with `LISTEN/NOTIFY`, Watermill with Kafka/NATS/Redis). It is **not** a distributed event-sourcing system: no multi-server replication, no leader election, no consensus protocols, no 2PC. Multi-process means multiple processes share a database; multi-server means a geographically distributed cluster with replication and failover. The library handles the former; the deployment infrastructure handles the latter.

| Deployment shape             | Supported | How                                                                                                                 |
| ---------------------------- | --------- | ------------------------------------------------------------------------------------------------------------------- |
| **Single-process embedded**  | Yes       | `stack/sqlite`, `stack/pebble`, `stack/turso` — one process, one file                                               |
| **Single-process + broker**  | Yes       | Any preset + `watermill.WithBackend(kafkaPub, kafkaSub)`                                                            |
| **Multi-process shared DB**  | Yes       | `stack/postgres` + `WithDistributedBus(listener)` — `LISTEN/NOTIFY` for cross-process pub/sub                       |
| **Multi-server distributed** | No        | No replication, consensus, or leader election. Use external coordination (etcd, Kubernetes) at the deployment layer |

---

## Consistency Guarantees

The library provides explicit guarantees on the write side and eventual consistency on the read side. Consumers must implement read-your-writes and bounded staleness themselves using the provided primitives.

| Guarantee                               | Provided?                    | Mechanism                                                                                                                                                         |
| --------------------------------------- | ---------------------------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| **Optimistic concurrency** (write side) | Yes — per stream             | `event.EventSink.Save(ctx, ref, events, expectedVersion)` — rejects with `ErrConcurrencyConflict` on version mismatch                                             |
| **Linearizable writes** (per stream)    | Yes                          | Single-writer per stream via expectedVersion; atomic save in SQL transaction or Pebble batch                                                                      |
| **Eventual consistency** (read side)    | Yes — per projection         | Projections lag behind the event log; `projectionhost.LagDuration()` and `LagPerProjection()` track lag                                                           |
| **Read-your-writes**                    | No — consumer must implement | After a command succeeds, the read model may not yet reflect it. Consumer can poll `LagDuration()` or use the command's returned events for optimistic UI updates |
| **Bounded staleness**                   | No — consumer must implement | No built-in rejection of stale reads. Consumer can check `LagDuration()` before querying and reject if lag exceeds a threshold                                    |
| **Monotonic reads**                     | No — not guaranteed          | If two projections run at different speeds, reads from different projections may see inconsistent snapshots                                                       |
| **Consistent prefix reads**             | Yes — per stream             | Events within a single stream are ordered by version; cross-stream order is eventual                                                                              |

---

## Tooling & Testing

| Term              | Definition                                                           | Context                                                                                                           |
| ----------------- | -------------------------------------------------------------------- | ----------------------------------------------------------------------------------------------------------------- |
| **cqrs-gen**      | Code generator: typed handler registration from annotated Go structs | `cmd/cqrs-gen` — generates `RegisterTyped` boilerplate from `//cqrs:command`, `//cqrs:query`, `//cqrs:event` tags |
| **api-stability** | CI tool: compares exported API surface against a golden file         | `cmd/api-stability` — catches breaking changes before release                                                     |
| **doc-check**     | CI tool: verifies Go import paths + qualified symbols in docs        | `cmd/doc-check` — validates SKILL.md, AGENTS.md, and skill references by default                                  |
| **eventtest**     | Test helpers: FakeStore, FakeBus, FakeSnapshotStore, assertions      | `event/v4/eventtest` — `AssertGolden`, event factories                                                            |
| **querytest**     | Test helper: `New(tb, queryType)` for query construction             | `query/querytest` — `tb.Fatalf` on error, no panics                                                               |
| **idtest**        | Test helpers: `Parse*(tb, s)` for branded ID parsing                 | `id/idtest` — `tb.Fatalf` on error, no panics                                                                     |
| **testutil**      | Shared cross-module test helpers                                     | `testutil` — `NewCmd(tb, ...)`                                                                                    |
| **Scenario**      | Fluent BDD test DSL for deciders and projections                     | `scenario.Given/When/Then`, `scenario.GivenProjection/ThenNoError`                                                |

---

## Interface Hierarchy

```
event.Store = EventSink + EventSource
  EventSink:       Save(ctx, ref, events, expectedVersion), AppendBatch(ctx, ref, events)
  EventSource:     Load(ctx, ref), LoadFromVersion(ctx, ref, version), LoadToVersion(ctx, ref, maxVersion), LoadToTimestamp(ctx, ref, maxTime)
  Journal:         ReadAll(ctx)
  SeekableJournal: ReadFrom(ctx, afterEventID, limit)
  BackwardsSource: LoadBackwards(ctx, ref)

snapshot.SnapshotStore = SnapshotSink + SnapshotSource
  SnapshotSink:   Save(ctx, snapshot), Delete(ctx, ref)
  SnapshotSource: Load(ctx, ref), LoadAtVersion(ctx, ref, version)

event.CheckpointStore = CheckpointSink + CheckpointSource
  CheckpointSink:   Save(ctx, projectionName, checkpoint)
  CheckpointSource: Load(ctx, projectionName)

event.Bus = Publisher + Subscriber
  Publisher:  Publish(ctx, events...)
  Subscriber: Subscribe(eventType, handler), SubscribeAll(handler)
  + Use(Middleware...), UsePublish(PublishMiddleware...)

command.Bus = Publisher + Subscriber
  Publisher:  Publish(ctx, cmds...)
  Subscriber: Subscribe(cmdType, handler), SubscribeAll(handler)
  + Use(Middleware...)

command.Store = CommandSink + CommandSource
  CommandSink:          Save(ctx, ref, cmd)
  CommandSource:        Load(ctx, ref)
  CommandJournal:       ReadAll(ctx)
  SeekableCommandJournal: ReadFrom(ctx, afterCmdID, limit)

query.Store = QuerySink + QuerySource
  QuerySink:            SaveQuery(ctx, pq)
  QuerySource:          LoadQueries(ctx, after)
  QueryJournal:         ReadAllQueries(ctx)
  SeekableQueryJournal: ReadQueriesFrom(ctx, afterReqID, limit)

projection.Projection
  Name() string, Handle(ctx, evt) error, EventTypes() []event.Type

kv.Store = Reader + Writer + Closer
  Reader: Get(key), Has(key), NewIterator(prefix)
  Writer: Set(key, value), Delete(key), Batch()

kv.ViewStore[V,K]
  Get(ctx, key), Set(ctx, key, val), Delete(ctx, key), Scan(ctx, prefix)
  Optional: ViewQuerier[V].Query(ctx, ViewQuery), ViewCounter[V].Count(ctx, ViewQuery)

metaengine.Engine = Profile() EngineProfile + Closer
  Optional ADT backends (ISP — engines implement what they support):
    MapBackend:        MapSet/MapGet/MapDelete
    MapUpdater:        MapUpdate (atomic read-modify-write)
    ScanBackend:       MapScan (filter+sort in Go)
    PushdownScan:      PushdownMapScan (SQL WHERE/ORDER BY pushdown)
    StreamingScan:     StreamScan (iter.Seq2 — OOM-safe)
    SetBackend:        SetAdd/SetContains
    CounterBackend:    CounterIncrement/CounterGet
    MultimapBackend:   MultiAdd/MultiGet
    LogBackend:        LogAppend/LogTail
    StreamLogBackend:  StreamAppend/StreamRead/StreamVersion
    GraphBackend:      GraphAddEdge/GraphNeighbors
    VectorBackend:     VectorInsert/VectorSearch
    SearchBackend:     SearchInsert/SearchQuery
    SpatialBackend:    SpatialInsert/SpatialRange
  Optional capabilities:
    LayoutPlanner:     CreateLayout(collection, plan)
    LayoutPlanApplier: ApplyLayout(collection, plan) — columnar-native
    RawValueReader:    MapGetRaw (skip decode)
    AtomicAppender:    AppendWithVersion (optimistic concurrency)
    SnapshotBackend:   SaveSnapshot/LoadSnapshot
    AggregateReader:   Aggregate(ctx, coll, specs) — SQL-level
    VersionedStorage:  GetAsOf(ctx, coll, key, timestamp) — temporal
  Lifecycle:
    PlanRule:          Apply(ctx, PlanContext, result) error
    RulePipeline:      Run(ctx, PlanContext, result) error
```

---

## Anti-Patterns (Terms We Avoid)

| Instead of                 | We say                         | Why                                                                                                                                       |
| -------------------------- | ------------------------------ | ----------------------------------------------------------------------------------------------------------------------------------------- |
| "Database"                 | "Store" or "Event Store"       | CQRS separates write/read; "database" implies a single thing                                                                              |
| "Entity"                   | "Stream"                       | A stream is the consistency boundary (what DDD calls an aggregate); "entity" is too vague                                                 |
| "CRUD"                     | "Command + Event + Projection" | No updates or deletes — only append                                                                                                       |
| "Delete"                   | "Tombstone"                    | Event streams are append-only; soft-delete via metadata, never removal                                                                    |
| "State" (mutable)          | "Folded state"                 | State is always reconstructed from events via `Apply`, never directly mutated                                                             |
| "Aggregate Root" (OO)      | "Decider" (pure functions)     | ADR-0001: 9-method OO interface couples domain to infrastructure; pure `Decider[State]` + `Apply` separates them                          |
| "Update" / "Patch"         | "Event" (append-only)          | No mutation of past events; new events supersede old state via fold                                                                       |
| "Log Compaction"           | "Snapshot"                     | Compaction destroys the audit trail; snapshots avoid replay cost without losing data (DDIA)                                               |
| "2PC" / "Two-Phase Commit" | "Derived data" (projections)   | 2PC is blocking and fragile; projections derive independently from the log (DDIA, ADR-0016)                                               |
| "Outbox"                   | "Journal as outbox"            | ADR-0016: the event journal IS the outbox; `CatchUpSubscriber` replays and publishes. No separate outbox table needed                     |
| "Replication"              | "Storage backend concern"      | The library does not replicate; Postgres/Pebble replication handles this at the storage layer                                             |
| "Leader Election"          | "Deployment concern"           | No Raft/Paxos; optimistic concurrency per stream is the application-level fencing; deployment infra (K8s, etcd) handles node coordination |
| "Fencing Token"            | "ExpectedVersion"              | Application-level fencing via optimistic concurrency; a stale instance's write fails the version check                                    |
| "God Aggregate"            | "Small Decider + Deriver"      | Large aggregates violate SRP; split into small deciders + derivers for event→command derivation                                           |
| "Enforced Transport"       | "Transport helpers"            | The library provides SSE, gRPC, REST helpers but does not force a protocol; consumers choose (Service Design Patterns)                    |
| "Data Lakehouse"           | "Read models"                  | This is an application-level CQRS library, not an analytics platform; projections are operational read models, not analytical datasets    |
| "Manual query routing"     | "Cost-based planner"           | The metaengine assigns queries to engines by estimated cost — never hand-pick an engine per query                                          |
| "Hand-written DDL"         | "LayoutPlan auto-generation"   | The planner generates DDL from declared filter/sort fields — never write table schemas manually when a LayoutPlanner engine is available  |

---

## Patterns NOT in the Library

These concepts are intentionally absent as dedicated modules. They emerge from composition.

| Pattern                      | How it emerges                                                                 | Why no module                                                                                                                                                     |
| ---------------------------- | ------------------------------------------------------------------------------ | ----------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| **Saga / Process Manager**   | `bus.SubscribeAll` + `command.Dispatcher` + `deriver.Deriver`                  | Multi-step orchestration is domain-specific; a generic saga module imposes the wrong abstraction. See `example/taskmanager/` for a real implementation.           |
| **Domain Entity**            | App-defined inside the consumer's decider `Apply` function                     | The library models stream identity (`StreamRef`), not stream state — state shape is the consumer's domain decision.                                               |
| **Message Broker**           | Injected via Watermill adapter (`watermill.NewEventBus` with Kafka/NATS/Redis) | The library is transport-agnostic. The GoChannel default is for single-process; brokers are consumer choices.                                                     |
| **Outbox**                   | Event journal + `CatchUpSubscriber` + `EventPublisher`                         | ADR-0016 declined: the journal IS the outbox. A projection reads the journal and publishes events, making the pattern composable without a dedicated table.       |
| **Distributed Consensus**    | Optimistic concurrency per stream (`expectedVersion`)                          | No Raft/Paxos: the library provides single-writer-per-stream semantics. Multi-node coordination (leader election, quorum) is a deployment concern.                |
| **Log Compaction**           | `snapshot.SnapshotStore` with strategies                                       | Compaction destroys events — incompatible with event sourcing. Snapshots avoid replay cost without data loss. See `docs/research/time-travel-options.md`.         |
| **Stream Processing Engine** | `projectionhost.Host` (simple, correct) + `CatchUpSubscriber`                  | Windowing, watermarking, stream joins are over-engineering for application-level CQRS. Consumers needing Kafka-scale streaming use Kafka + the Watermill adapter. |
| **Fencing Tokens**           | `expectedVersion` (optimistic concurrency)                                     | Deployment-level fencing (K8s leases, etcd locks) is outside scope. Application-level fencing via version check is sufficient for single-writer-per-stream.       |
| **Data Lakehouse / Fabric**  | N/A — projections are operational read models                                  | This is an application-level CQRS library, not an analytics platform. Warehouse/lakehouse/fabric solve a different problem (analytics at organizational scale).   |

---

## Verification

The code block below is scanned by `cmd/doc-check` to verify every symbol
referenced in this document still exists in the codebase. Do not remove.

```go
package domain_language_verification

import (
	// Event Sourcing
	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/projection/v4"
	"github.com/larsartmann/go-cqrs-lite/snapshot/v4"

	// Error taxonomy
	errorfamily "github.com/larsartmann/go-error-family"

	// Record (structural foundation)
	"github.com/larsartmann/go-cqrs-lite/record/v4"

	// CQRS
	"github.com/larsartmann/go-cqrs-lite/command/v4"
	"github.com/larsartmann/go-cqrs-lite/decider/v4"
	"github.com/larsartmann/go-cqrs-lite/deriver/v4"
	"github.com/larsartmann/go-cqrs-lite/dispatcher/v4"
	"github.com/larsartmann/go-cqrs-lite/query/v4"

	// Identity
	"github.com/larsartmann/go-cqrs-lite/id/v4"

	// Storage
	"github.com/larsartmann/go-cqrs-lite/storage/v4"
	"github.com/larsartmann/go-cqrs-lite/storage/v4/sql"
	"github.com/larsartmann/go-cqrs-lite/storage/memory/v4"
	"github.com/larsartmann/go-cqrs-lite/storage/pebble/v4"
	"github.com/larsartmann/go-cqrs-lite/storage/turso/v4"

	// Read Models
	"github.com/larsartmann/go-cqrs-lite/graph/v4"
	"github.com/larsartmann/go-cqrs-lite/kv/v4"
	"github.com/larsartmann/go-cqrs-lite/listing/v4"
	"github.com/larsartmann/go-cqrs-lite/stack/v4"

	// Metaengine
	"github.com/larsartmann/go-cqrs-lite/metaengine/v4"

	// Messaging & Transport
	http "github.com/larsartmann/go-cqrs-lite/transport/http/v4"
	grpc "github.com/larsartmann/go-cqrs-lite/transport/grpc/v4"
	"github.com/larsartmann/go-cqrs-lite/watermill/v4"

	// Security
	"github.com/larsartmann/go-cqrs-lite/codec/v4"
	"github.com/larsartmann/go-cqrs-lite/encryption/v4"
	"github.com/larsartmann/go-cqrs-lite/signing/v4"

	// Cross-Cutting
	"github.com/larsartmann/go-cqrs-lite/dedup/v4"
	"github.com/larsartmann/go-cqrs-lite/idempotency/v4"
	"github.com/larsartmann/go-cqrs-lite/middleware/v4"
	"github.com/larsartmann/go-cqrs-lite/otel/v4"
	"github.com/larsartmann/go-cqrs-lite/prometheus/v4"
	"github.com/larsartmann/go-cqrs-lite/retry/v4"
	"github.com/larsartmann/go-cqrs-lite/scheduling/v4"
	"github.com/larsartmann/go-cqrs-lite/schema/v4"
	"github.com/larsartmann/go-cqrs-lite/scenario/v4"
	"github.com/larsartmann/go-cqrs-lite/projectionhost/v4"

	// Catalog + Tooling
	"github.com/larsartmann/go-cqrs-lite/catalog/v4"
	"github.com/larsartmann/go-cqrs-lite/event/v4/eventtest"
	"github.com/larsartmann/go-cqrs-lite/id/v4/idtest"
	"github.com/larsartmann/go-cqrs-lite/query/v4/querytest"
	"github.com/larsartmann/go-cqrs-lite/testutil/v4"
)

var _ = []any{
	// Event Sourcing
	event.New,
	event.NewEvent,
	id.NewStreamRef,
	event.DetectTombstone,
	event.MarkTombstone,
	event.MarkRebirth,
	event.WithProcessingMode,
	event.ProcessingModeFrom,
	event.DecodePayloadAuto,
	event.NewMetadata,
	errorfamily.NewRejection,
	errorfamily.NewConflict,
	errorfamily.NewTransient,
	errorfamily.NewInfrastructure,
	errorfamily.NewCorruption,
	errorfamily.NewOrchestration,
	record.Record{},
	record.NewStreamRef,
	record.CommonMetadata{},
	projection.NewProjection,
	snapshot.NewTypedStore,
	snapshot.EveryNEvents,
	sql.SQLiteDialect{},
	sql.PostgresDialect{},

	// CQRS
	command.New,
	command.NewPersistedCommand,
	command.NewDispatcher,
	command.NewMemoryBus,
	command.RegisterTyped,
	query.New,
	query.NewPersistedQuery,
	query.NewDispatcher,
	query.RegisterTyped,
	query.DispatchTyped,
	decider.NewRepository,
	deriver.Noop,

	// Identity
	id.New,
	id.NewStreamID,
	id.NewEventID,
	id.NewCorrelationID,
	id.NewCausationID,
	id.NewCommandID,
	id.DeriveStreamID,
	id.DeriveCommandID,

	// Storage
	storage.NewSQLiteEventStore,
	storage.NewSQLEventStore,
	storage.NewSQLEventStoreWithDialect,
	storage.NewSQLiteBackend,
	storage.NewSQLBackend,
	storage.NewSQLiteViewStore,
	storage.NewRelationalProjection,
	storage.NewRelationalStore,
	memory.NewMemoryStore,
	memory.NewMemorySnapshotStore,
	memory.NewMemoryCheckpointStore,
	pebble.NewStore,
	pebble.Open,
	pebble.NewBackend,
	turso.Open,
	turso.OpenInMemory,
	turso.OpenSync,

	// Read Models
	kv.NewMemStore,
	kv.NewTypedStore[any, id.StreamID],
	kv.NewCache[any, id.StreamID],
	stack.NewMaterialize[any, id.StreamID],
	stack.ReadModel[any, id.StreamID],
	graph.NewGraphProjection,
	graph.NewMemoryDriver,

	// Metaengine
	metaengine.Plan,
	metaengine.NewMemoryEngine,
	metaengine.ExecuteTyped[any, any],
	metaengine.Query[any, any],
	metaengine.On[any],
	metaengine.OnTyped[any],
	metaengine.OnRecord[any],
	metaengine.NewReader[any],
	metaengine.NewQueryBuilder[any],
	metaengine.NewWatcher[any],
	metaengine.NewTieredStore,
	metaengine.ShouldMaterialize,
	metaengine.ReplayCost,
	metaengine.MaterializeCost,
	metaengine.Serialize,
	metaengine.DeserializePlan,
	metaengine.PlanDiff,
	metaengine.NewManifest,
	metaengine.WithDryRun,
	metaengine.WithColumnarLayout,
	metaengine.FilterOnField[any],
	metaengine.SortOnField[any],
	metaengine.ADTMap,
	metaengine.ADTCounter,
	metaengine.ADTGraph,
	metaengine.ADTVector,
	metaengine.ComplexityO1,
	metaengine.LayoutRow,
	metaengine.LayoutColumnar,
	metaengine.PersistenceVolatile,
	metaengine.PersistencePersistent,
	metaengine.ReplicationNone,
	metaengine.ReplicationLeaderless,
	metaengine.DiagLevelScream,
	metaengine.DiagLevelWarn,
	metaengine.DefaultWriteAmplificationBudget,
	metaengine.Delta{},
	metaengine.Edge{},
	metaengine.Skip{},
	metaengine.VersionedStorage(nil),
	metaengine.EngineProfile{},
	metaengine.Store{},

	// Projection Lifecycle
	projectionhost.New,
	listing.NewInMemoryStreamReader,
	watermill.NewEventBus,
	watermill.NewCommandBus,
	watermill.NewCatchUpSubscriber,
	watermill.NewEventPublisher,
	watermill.NewCommandPublisher,

	// Transport
	http.NewSSEBroker,
	grpc.NewCommandClient,
	grpc.NewQueryClient,
	grpc.RegisterCommandService,
	grpc.RegisterQueryService,

	// Security
	signing.NewHMAC,
	signing.NewEd25519,
	encryption.NewXChaCha20Poly1305,
	encryption.NewAES256GCM,
	encryption.DeriveKey,
	codec.JSONCodec{},
	codec.CBORCodec{},
	codec.CBORCompactCodec{},
	codec.RawCodec{},

	// Cross-Cutting
	idempotency.NewMemoryStore,
	idempotency.ErrDuplicate,
	middleware.CommandIdempotency,
	middleware.EventIdempotency,
	middleware.QueryIdempotency,
	middleware.CommandCircuitBreaker,
	retry.Do,
	retry.DefaultConfig,
	dedup.NewRing,
	dedup.DefaultCapacity,
	scheduling.NewMemoryTimerStore,
	scheduling.New,
	schema.NewVersionedStore,
	schema.NewVersionedSeekableJournal,
	schema.NewValidator,
	scenario.Given,
	otel.Setup,
	otel.NewTracer,
	otel.NewMeter,
	prometheus.Setup,

	// Transport (Backfill, Heartbeat)
	http.BackfillHandler,
	http.WriteSSEHeartbeat,
	http.DefaultSSEHeartbeat,

	// Catalog + Tooling
	catalog.NewRegistry,
	catalog.SchemaFromType[any],
	eventtest.AssertGolden,
	querytest.New,
	idtest.ParseEventID,
	testutil.NewCmd,
}
```
