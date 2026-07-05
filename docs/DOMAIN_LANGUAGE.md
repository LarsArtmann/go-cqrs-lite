# Domain Language

A **Ubiquitous Language** for `go-cqrs-lite` — shared across library consumers, contributors, and AI.
Inspired by Domain-Driven Design (DDD).

Every term below should mean the **same thing** to everyone who reads it.
If a word means something different to a consumer than to an implementer, it is defined here.

> **Import convention:** All modules use the `/v3` import path suffix (e.g. `github.com/larsartmann/go-cqrs-lite/event/v3`). The `Context` column uses abbreviated package names (`event.`, `command.`, `storage.`) for readability — consumers must append `/v3` when importing.

> **How to use this file:**
>
> - Keep terms concise — one clear sentence per definition
> - Update when new domain concepts emerge
> - Use these terms consistently in code, docs, and conversations
> - When in doubt about a word's meaning, check here first

---

## Core Concepts

### Event Sourcing

| Term                 | Definition                                                                   | Context                                                                                                      |
| -------------------- | ---------------------------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------ |
| **Event**            | Immutable record of something that happened in the domain                    | `event.Event = *ImmutableEvent` — the single concrete implementation (not an interface)                      |
| **ImmutableEvent**   | The concrete event struct: ID, type, aggregate, version, payload, metadata   | `event.New()` (typed payload) or `event.NewEvent()` (raw bytes)                                              |
| **Aggregate**        | Cluster of domain objects treated as a single unit of consistency            | Has a unique identity (`AggregateRef`) and an event stream; state is app-defined                             |
| **AggregateRef**     | `{Type, ID}` — canonical identity of an aggregate instance                   | `event.NewAggregateRef(type, id)` — passed to all Store methods                                              |
| **AggregateType**    | String category for an aggregate (e.g. `"User"`, `"Order"`)                  | `type AggregateType string`                                                                                  |
| **Stream**           | Ordered sequence of events for a single aggregate, ordered by Version        | `Load()` returns the full stream; `LoadFromVersion()` returns a suffix                                       |
| **Version**          | Monotonically increasing position of an event within its stream (1-indexed)  | `type Version uint64` — used for optimistic concurrency                                                      |
| **Event Store**      | Append-only persistence layer for event streams                              | `event.Store` (composite of `EventSink` + `EventSource`)                                                     |
| **Journal**          | Global append-only log of all events across all aggregates                   | `event.Journal.ReadAll()` — cross-aggregate reads                                                            |
| **SeekableJournal**  | Journal with position-based reading (by EventID)                             | `event.SeekableJournal.ReadFrom(afterEventID, limit)` — efficient projection catch-up                        |
| **BackwardsSource**  | Read-side that loads events in reverse (newest-first)                        | `event.BackwardsSource.LoadBackwards(ref)`                                                                   |
| **Snapshot**         | Point-in-time capture of aggregate state at a specific version               | Avoids replaying the entire stream on every load                                                             |
| **Snapshot Store**   | Persistence layer for aggregate snapshots                                    | `snapshot.SnapshotStore` (composite of `SnapshotSink` + `SnapshotSource`)                                    |
| **Projection**       | Consumer-side contract for building a read model from events                 | `projection.Projection` — `Name()`, `Handle()`, `EventTypes()`                                               |
| **Checkpoint**       | Last-processed event position for a specific projection                      | `event.CheckpointStore` — enables resume after restart                                                       |
| **Tombstone**        | Soft-delete marker on event metadata (3 statuses)                            | `Active`, `Tombstoned`, `Undetermined` — detected via `event.DetectTombstone()`                              |
| **Rebirth**          | Undo of a tombstone — marks an aggregate as live again                       | `event.MarkRebirth(evt)` — sets rebirth metadata                                                             |
| **ProcessingMode**   | Context-scoped flag: `ModeLive` vs `ModeReplay`                              | `event.WithProcessingMode(ctx, ModeReplay)` — lets handlers skip side-effects during catch-up                |
| **Metadata**         | Typed envelope on every event: tracing, causation, tombstone, custom fields  | `event.Metadata` struct — `Tracing`, `Causation`, `Tombstone`, `Custom map[MetadataKey]string`               |
| **Tracing**          | Embedded metadata fields: CorrelationID, CausationID, UserID, RequestID      | `event.Tracing` struct — promoted into `Metadata`, JSON-serializable                                         |
| **Causation**        | Links an event to the command that caused it (type + ID)                     | `event.Causation{CommandType, CommandID}` — set via `event.WithCommandCausality(ctx, type, id)`              |
| **ContextEnricher**  | Function that extracts metadata from context and stamps it onto new events   | `event.ContextEnricher` — `decider.Repository` applies it automatically on Save                              |
| **SnapshotStrategy** | Policy deciding when to persist a snapshot                                   | `snapshot.SnapshotStrategy` — `ShouldSnapshot(type, version) bool`; impl: `snapshot.EveryNEvents(n)`         |
| **Load Coalescing**  | Concurrent `Load` calls for the same aggregate coalesce into one store query | `decider.Repository` uses `singleflight.Group` — transparent, disable via `WithLoadCoalescing[State](false)` |

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
| **Decider**          | Pure-function aggregate: fold state from events                     | `decider.Decider[State]` — `Initial` + `Apply(state, evt) (State, error)`                |
| **TypedDecider**     | Decider with command type bound at compile time                     | `decider.TypedDecider[State, Cmd]` — carries `Decide` as a struct field                  |
| **Repository**       | Loads aggregate state, executes decider, saves and publishes events | `decider.Repository[State]` — composes Store + Bus + optional Snapshot                   |
| **Bus** (event)      | Message bus for publishing/subscribing to events                    | `event.Bus` (Publisher + Subscriber + `Use`/`UsePublish` middleware)                     |
| **Bus** (command)    | Message bus for command pub/sub (queue-style dispatch)              | `command.Bus` (Publisher + Subscriber + `Use` middleware)                                |
| **MemoryBus**        | In-memory, synchronous command bus implementation                   | `command.NewMemoryBus()` — `Subscribe`, `SubscribeAll`, `Publish`, `Use`                 |
| **Deriver**          | Pure function transforming an event into zero or more commands      | `deriver.Deriver` — building block for saga/process-manager patterns                     |
| **Idempotent**       | Deriver combinator that stamps deterministic command IDs            | `deriver.Deriver.Idempotent()` — deduplication via idempotency store                     |

### Identity

| Term                  | Definition                                                              | Context                                                                            |
| --------------------- | ----------------------------------------------------------------------- | ---------------------------------------------------------------------------------- |
| **Branded ID**        | ULID-backed, phantom-typed identifier — compile-time type safety        | `id.Of[T] = cbid.ID[T, ulid.ULID]` — prevents mixing IDs of different types        |
| **AggregateID**       | Branded **string-backed** ID for aggregates (the only non-ULID ID)      | `id.AggregateID = cbid.ID[AggregateMarker, string]` — accepts any non-empty string |
| **EventID**           | ULID-branded ID for events (time-sortable)                              | `id.EventID = id.Of[EventMarker]` — `id.NewEventID()`                              |
| **CorrelationID**     | ULID-branded ID linking related operations across a command→event chain | `id.CorrelationID = id.Of[CorrelationMarker]`                                      |
| **CausationID**       | ULID-branded ID tracking which event/command caused this one            | `id.CausationID = id.Of[CausationMarker]`                                          |
| **CommandID**         | ULID-branded ID for commands                                            | `id.CommandID = id.Of[CommandMarker]` — can be deterministically derived           |
| **Marker**            | Phantom struct type used as a type parameter for branding               | `AggregateMarker`, `EventMarker`, `CorrelationMarker`, `CommandMarker`, etc.       |
| **DeriveAggregateID** | Deterministic SHA-256-derived aggregate ID for idempotent workflows     | `id.DeriveAggregateID(namespace, keys...)`                                         |

### Error Taxonomy

All errors are classified into a 5-family taxonomy:

| Family             | Meaning                                                | Constructor                    |
| ------------------ | ------------------------------------------------------ | ------------------------------ |
| **Rejection**      | Business rule violation (4xx equivalent)               | `event.NewRejection(...)`      |
| **Conflict**       | Optimistic concurrency or duplicate (409 equivalent)   | `event.NewConflict(...)`       |
| **Transient**      | Retryable infrastructure failure (503 equivalent)      | `event.NewTransient(...)`      |
| **Infrastructure** | Non-retryable infrastructure failure (500 equivalent)  | `event.NewInfrastructure(...)` |
| **Corruption**     | Data integrity violation — human intervention required | `event.NewCorruption(...)`     |

---

## Storage

| Term               | Definition                                                                           | Context                                                                                                                                             |
| ------------------ | ------------------------------------------------------------------------------------ | --------------------------------------------------------------------------------------------------------------------------------------------------- |
| **Dialect**        | SQL dialect abstraction for portable store implementations                           | `sql.Dialect` interface in `storage/sql` — `SQLiteDialect{}`, `PostgresDialect{}` (re-exported as `storage.SQLiteDialect`)                          |
| **SQLEventStore**  | SQL-backed implementation of `event.Store` + `Journal` + `SeekableJournal`           | `storage.NewSQLiteEventStore(db)`, `storage.NewSQLEventStore(db)`, `storage.NewSQLEventStoreWithDialect(db, dialect)`                               |
| **SQLBackend**     | Facade exposing all SQL stores sharing one `*sql.DB` (lazy, goroutine-safe)          | `storage.NewSQLiteBackend(db)`, `storage.NewSQLBackend(db)` — exposes EventStore, CommandStore, QueryStore, SnapshotStore, CheckpointStore, KVStore |
| **Pebble Store**   | Embedded KV event store (no SQL dependency)                                          | `pebble.NewStore(db, logger)` (package: `storage/pebble`)                                                                                           |
| **Pebble Backend** | Facade exposing all Pebble stores sharing one `*pebble.DB` via disjoint key prefixes | `pebble.Open(dir, opts, logger)` (owns DB) or `pebble.NewBackend(db, logger)` (borrows DB)                                                          |
| **Turso**          | Embedded LibSQL connector with sync support                                          | `turso.Open(dbPath)`, `turso.OpenInMemory()`, `turso.OpenSync()` (package: `storage/turso`)                                                         |
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

| Term                  | Definition                                                                  | Context                                                                          |
| --------------------- | --------------------------------------------------------------------------- | -------------------------------------------------------------------------------- |
| **ProjectionHost**    | Managed host: one goroutine per projection, crash-restart with backoff      | `projectionhost.New(journal, checkpointStore)` — `Register`, `Start`, `Stop`     |
| **DeadLetterStore**   | Captures poison messages that exhaust retries for later replay              | `projectionhost.DeadLetterStore` — `Store`, `List`, `Delete`, `Purge`            |
| **CatchUpSubscriber** | Replay+live handoff: historical events then seamless live transition        | `watermill.NewCatchUpSubscriber(journal, liveSub, cpStore, logger)`              |
| **AggregateListing**  | Read model for listing aggregates with cursor pagination + tombstone status | `listing.InMemoryAggregateReader` (from Journal) or `storage.SQLAggregateReader` |

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

| Term                 | Definition                                                                                      | Context                                                                            |
| -------------------- | ----------------------------------------------------------------------------------------------- | ---------------------------------------------------------------------------------- |
| **Idempotency**      | Command deduplication for at-least-once delivery                                                | `idempotency.Store` — `CheckAndRecord` is atomic; `idempotency.ErrDuplicate`       |
| **Timer**            | Durable deadline: "cancel order after 30 min unpaid"                                            | `scheduling.TimerStore[P]` — `Schedule` (idempotent), `Due`, `MarkFired`, `Cancel` |
| **Scheduler**        | Polls a TimerStore and dispatches due timers with retry                                         | `scheduling.New(store, dispatchFunc)` — exponential backoff, max retries           |
| **Catalog**          | Auto-generates AsyncAPI, OpenAPI, D2, and EventCatalog docs                                     | `catalog.Registry` + `catalog.SchemaFromType[T]()` + per-format exporters          |
| **Middleware**       | Cross-cutting concerns: Logging, Retry, Recovery, Validation, Metrics, Tracing, Circuit Breaker | `middleware/` — 3 variants each (Command*/Event*/Query\*)                          |
| **OTel**             | OpenTelemetry tracing + metrics re-exports                                                      | `otel.Setup()` — one-call provider; `otel.NewTracer()`, `otel.NewMeter()`          |
| **Prometheus**       | OTel→Prometheus metrics bridge with `/metrics` HTTP handler                                     | `prometheus.Setup()`                                                               |
| **Schema Evolution** | Upcasting: transform old event versions on load                                                 | `schema.Upcaster`, `schema.VersionedStore` — migrate payloads at read time         |
| **Validator**        | Runtime type registration for event payload validation                                          | `schema.Validator` — `RegisterType[T]()`                                           |

---

## Tooling & Testing

| Term              | Definition                                                           | Context                                                                                                           |
| ----------------- | -------------------------------------------------------------------- | ----------------------------------------------------------------------------------------------------------------- |
| **cqrs-gen**      | Code generator: typed handler registration from annotated Go structs | `cmd/cqrs-gen` — generates `RegisterTyped` boilerplate from `//cqrs:command`, `//cqrs:query`, `//cqrs:event` tags |
| **api-stability** | CI tool: compares exported API surface against a golden file         | `cmd/api-stability` — catches breaking changes before release                                                     |
| **doc-check**     | CI tool: verifies Go import paths + qualified symbols in docs        | `cmd/doc-check` — validates SKILL.md, AGENTS.md, and skill references by default                                  |
| **eventtest**     | Test helpers: FakeStore, FakeBus, FakeSnapshotStore, assertions      | `event/v3/eventtest` — `AssertGolden`, event factories                                                            |
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
```

---

## Anti-Patterns (Terms We Avoid)

| Instead of        | We say                         | Why                                                                           |
| ----------------- | ------------------------------ | ----------------------------------------------------------------------------- |
| "Database"        | "Store" or "Event Store"       | CQRS separates write/read; "database" implies a single thing                  |
| "Entity"          | "Aggregate"                    | DDD aggregate is the consistency boundary; entity is too vague                |
| "CRUD"            | "Command + Event + Projection" | No updates or deletes — only append                                           |
| "Delete"          | "Tombstone"                    | Event streams are append-only; soft-delete via metadata, never removal        |
| "State" (mutable) | "Folded state"                 | State is always reconstructed from events via `Apply`, never directly mutated |

---

## Patterns NOT in the Library

These concepts are intentionally absent as dedicated modules. They emerge from composition.

| Pattern                    | How it emerges                                                                 | Why no module                                                                                                                                           |
| -------------------------- | ------------------------------------------------------------------------------ | ------------------------------------------------------------------------------------------------------------------------------------------------------- |
| **Saga / Process Manager** | `bus.SubscribeAll` + `command.Dispatcher` + `deriver.Deriver`                  | Multi-step orchestration is domain-specific; a generic saga module imposes the wrong abstraction. See `example/taskmanager/` for a real implementation. |
| **Domain Entity**          | App-defined inside the consumer's decider `Apply` function                     | The library models aggregate identity (`AggregateRef`), not aggregate state — state shape is the consumer's domain decision.                            |
| **Message Broker**         | Injected via Watermill adapter (`watermill.NewEventBus` with Kafka/NATS/Redis) | The library is transport-agnostic. The GoChannel default is for single-process; brokers are consumer choices.                                           |
