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

All errors are classified into a 5-family taxonomy:

| Family             | Meaning                                                | Constructor                    |
| ------------------ | ------------------------------------------------------ | ------------------------------ |
| **Rejection**      | Business rule violation (4xx equivalent)               | `errorfamily.NewRejection(...)`      |
| **Conflict**       | Optimistic concurrency or duplicate (409 equivalent)   | `errorfamily.NewConflict(...)`       |
| **Transient**      | Retryable infrastructure failure (503 equivalent)      | `errorfamily.NewTransient(...)`      |
| **Infrastructure** | Non-retryable infrastructure failure (500 equivalent)  | `errorfamily.NewInfrastructure(...)` |
| **Corruption**     | Data integrity violation — human intervention required | `errorfamily.NewCorruption(...)`     |

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

| Term                    | Definition                                                                                                   | Context                                                                                                                                                                                                                                         |
| ----------------------- | ------------------------------------------------------------------------------------------------------------ | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| **Idempotent Receiver** | Deduplication for safe retries under at-least-once delivery (DDIA pattern)                                   | `idempotency.Store` — `CheckAndRecord` is atomic; `idempotency.ErrDuplicate`; `dedup.Ring` — bounded O(1) dedup at stream boundaries; middleware: `middleware.CommandIdempotency`, `middleware.EventIdempotency`, `middleware.QueryIdempotency` |
| **Timer**               | Durable deadline: "cancel order after 30 min unpaid"                                                         | `scheduling.TimerStore[P]` — `Schedule` (idempotent), `Due`, `MarkFired`, `Cancel`                                                                                                                                                              |
| **Scheduler**           | Polls a TimerStore and dispatches due timers with retry                                                      | `scheduling.New(store, dispatchFunc)` — exponential backoff, max retries                                                                                                                                                                        |
| **Catalog**             | Auto-generates AsyncAPI, OpenAPI, D2, and EventCatalog docs                                                  | `catalog.Registry` + `catalog.SchemaFromType[T]()` + per-format exporters                                                                                                                                                                       |
| **Middleware**          | Cross-cutting concerns: Logging, Retry, Recovery, Validation, Metrics, Tracing, Circuit Breaker, Idempotency | `middleware/` — 3 variants each (Command*/Event*/Query\*)                                                                                                                                                                                       |
| **OTel**                | OpenTelemetry tracing + metrics re-exports                                                                   | `otel.Setup()` — one-call provider; `otel.NewTracer()`, `otel.NewMeter()`                                                                                                                                                                       |
| **Prometheus**          | OTel→Prometheus metrics bridge with `/metrics` HTTP handler                                                  | `prometheus.Setup()`                                                                                                                                                                                                                            |
| **Schema Evolution**    | Forward/backward-compatible event evolution via upcasting on read (DDIA Ch. 4)                               | `schema.Upcaster`, `schema.VersionedStore`, `schema.VersionedSeekableJournal` — events are immutable; old versions transformed at load time                                                                                                     |
| **Validator**           | Runtime type registration for event payload validation                                                       | `schema.Validator` — `RegisterType[T]()`                                                                                                                                                                                                        |
| **Circuit Breaker**     | Stops cascading failures by opening after N consecutive errors; half-open probe recovery                     | `middleware.NewCircuitBreaker[M]()`, `middleware.CircuitBreakerConfig`, `middleware.ErrCircuitBreakerOpen`; variants: `CommandCircuitBreaker`, `EventCircuitBreaker`, `QueryCircuitBreaker`                                                     |
| **Retry**               | Zero-dep retry with exponential backoff + jitter (up to 50%) for transient failures                          | `retry.Do(ctx, config, fn)`, `retry.Config`, `retry.Backoff`, `retry.ErrExhausted`, `retry.ErrCanceled`; middleware: `middleware.NewRetry[M]()`                                                                                                 |
| **Dedup Ring**          | Fixed-capacity O(1) ring buffer for deduplicating event IDs at replay→live boundaries                        | `dedup.Ring`, `dedup.NewRing(capacity)`, `dedup.DefaultCapacity` (1024) — used by `projectionhost` and `watermill.CatchUpSubscriber`                                                                                                            |
| **Projection Lag**      | Time duration between newest event in journal and last-processed event per projection                        | `projectionhost.LagDuration()` (max across workers), `projectionhost.LagPerProjection()` (per-worker map) — for dashboards and health checks                                                                                                    |
| **Heartbeat**           | Keep-alive comment frames on SSE connections to prevent proxy idle timeouts                                  | `transport/http.DefaultSSEHeartbeat` (15s), `transport/http.WriteSSEHeartbeat(w)` — SSE spec comment frames                                                                                                                                     |
| **Backfill**            | REST endpoint for fetching missed events by position (complement to SSE reconnection)                        | `transport/http.BackfillHandler(broker)` — `GET /events/backfill?after=<id>&limit=500` — applies broker's `WithPayloadTransform`                                                                                                                |
| **BufferEncoder**       | Optional codec interface for zero-allocation encoding into a caller-provided buffer                          | `codec.BufferEncoder` — `EncodeToBuffer(payload, buf)` — implemented by `JSONCodec`, `CBORCodec`, `CBORCompactCodec`                                                                                                                            |
| **Materialized View**   | Read model derived from the event log; rebuildable from events (DDIA "derived data")                         | `stack.Materialize[V,K]` (KV/document), `storage.RelationalProjection` (SQL), `graph.GraphProjection` (graph) — all implement `projection.Projection`                                                                                           |
| **High-Water Mark**     | DDIA term for the maximum safely-processed position in a stream; this library calls it "Checkpoint"          | `event.CheckpointStore` — the library's checkpoint IS a high-water mark; per-projection, resumable after restart                                                                                                                                |

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
```

---

## Anti-Patterns (Terms We Avoid)

| Instead of                 | We say                         | Why                                                                                                                                       |
| -------------------------- | ------------------------------ | ----------------------------------------------------------------------------------------------------------------------------------------- |
| "Database"                 | "Store" or "Event Store"       | CQRS separates write/read; "database" implies a single thing                                                                              |
| "Entity"                   | "Aggregate"                    | DDD aggregate is the consistency boundary; entity is too vague                                                                            |
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

---

## Patterns NOT in the Library

These concepts are intentionally absent as dedicated modules. They emerge from composition.

| Pattern                      | How it emerges                                                                 | Why no module                                                                                                                                                     |
| ---------------------------- | ------------------------------------------------------------------------------ | ----------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| **Saga / Process Manager**   | `bus.SubscribeAll` + `command.Dispatcher` + `deriver.Deriver`                  | Multi-step orchestration is domain-specific; a generic saga module imposes the wrong abstraction. See `example/taskmanager/` for a real implementation.           |
| **Domain Entity**            | App-defined inside the consumer's decider `Apply` function                     | The library models aggregate identity (`AggregateRef`), not aggregate state — state shape is the consumer's domain decision.                                      |
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
	event.NewAggregateRef,
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
	id.NewAggregateID,
	id.NewEventID,
	id.NewCorrelationID,
	id.NewCausationID,
	id.NewCommandID,
	id.DeriveAggregateID,
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
	kv.NewTypedStore[any, id.AggregateID],
	kv.NewCache[any, id.AggregateID],
	stack.NewMaterialize[any, id.AggregateID],
	stack.ReadModel[any, id.AggregateID],
	graph.NewGraphProjection,
	graph.NewMemoryDriver,

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
