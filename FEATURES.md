# Features

> Honest, verified inventory of what go-cqrs-lite actually does — not what it plans to do.

**Last audited:** 2026-07-05 (error taxonomy sweep, deriver module, DOMAIN_LANGUAGE.md rebuild, dead code removal, ROADMAP/TODO reconciliation) · **Module count:** 47 `go.mod` files (all wired into `go.work`) · **Go version:** 1.26.3

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

> `import "github.com/larsartmann/go-cqrs-lite/command/v3"`

| Feature                  | Detail                                                                                                                                             | Status |
| ------------------------ | -------------------------------------------------------------------------------------------------------------------------------------------------- | ------ |
| Command dispatch         | `Dispatcher.Dispatch(ctx, cmd)` routes to registered handler                                                                                       | ✅     |
| Handler registration     | `Dispatcher.Register(cmdType, handler)` with duplicate guard                                                                                       | ✅     |
| Middleware chain         | `Dispatcher.Use(middleware...)` — applied at registration time, reverse order                                                                      | ✅     |
| Lifecycle                | `Dispatcher.Close()` — rejects all ops after close                                                                                                 | ✅     |
| Validation               | `New()` rejects empty type and zero aggregateID                                                                                                    | ✅     |
| TypedHandler[T]          | `RegisterTyped[T](d, type, handler)` — type-safe handler receiving `T` not `Command`                                                               | ✅     |
| Command metadata         | Own `Metadata` struct (embeds `Tracing`, no longer aliases `event.Metadata` — ADR-0031) with CorrelationID, CausationID, UserID, RequestID, Custom | ✅     |
| Metadata options         | `WithCorrelationID`, `WithCausationID`, `WithUserID`, `WithRequestID`                                                                              | ✅     |
| Persisted command        | `PersistedCommand` struct with ID, Type, AggregateRef, ReceivedAt, Payload, Metadata                                                               | ✅     |
| Command store interfaces | `CommandSink`, `CommandSource`, `Store` (Sink+Source) — persisted command log                                                                      | ✅     |
| CommandJournal           | `ReadAll(ctx)` — global command log ordered by ReceivedAt; audit trail of every command                                                            | ✅     |
| SeekableCommandJournal   | `ReadFrom(ctx, afterCommandID, limit)` — position-based command replay with ULID checkpoints                                                       | ✅     |
| Command Bus              | `Bus`: `Publish`, `Subscribe`, `SubscribeAll`, `Use` — command pub/sub (concrete impls keep `Close()`)                                             | ✅     |
| Publisher / Subscriber   | ISP split: `Publisher.Publish(ctx, cmds...)`, `Subscriber.Subscribe(type, handler)`                                                                | ✅     |
| PublishMiddleware        | `PublishMiddleware` wraps the publish path for cross-cutting concerns (signing, tracing)                                                           | ✅     |

### Query Dispatcher ✅ FULLY_FUNCTIONAL

> `import "github.com/larsartmann/go-cqrs-lite/query/v3"`

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

> `import "github.com/larsartmann/go-cqrs-lite/event/v3"`

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
| BackwardsSource       | `LoadBackwards(ctx, aggRef)` — loads events in reverse version order                                                                                                                                                                                                                                                                        | ✅     |
| TombstoneStatus       | `Active`, `Tombstoned`, `Undetermined` — tri-state enum for soft-delete; `DetectTombstone`, `MarkTombstone`, `MarkRebirth`                                                                                                                                                                                                                  | ✅     |
| Time-travel queries   | `LoadToVersion` and `LoadToTimestamp` — read aggregate state at a point in time                                                                                                                                                                                                                                                             | ✅     |
| Projection interface  | `Projection`: `Name`, `Handle(ctx, Event)`, `EventTypes()` (nil = all)                                                                                                                                                                                                                                                                      | ✅     |
| Context replay marker | `WithReplay(ctx, true)` — marks context as replay; handlers can distinguish                                                                                                                                                                                                                                                                 | ✅     |
| DecodePayload[T]      | `DecodePayload[T](evt, codec)` — type-safe payload deserialization                                                                                                                                                                                                                                                                          | ✅     |
| DecodePayloads[T]     | `DecodePayloads[T](events, codec)` — batch payload deserialization                                                                                                                                                                                                                                                                          | ✅     |
| PayloadReadOnly       | Zero-copy read access for internal paths (signing, pebble, storage, middleware)                                                                                                                                                                                                                                                             | ✅     |
| Stream loading        | `StreamingSource`/`StreamingJournal` — cursor-based event reads without materializing full slices (SQL, Pebble, Memory backends)                                                                                                                                                                                                            | ✅     |
| Slice helpers         | `SliceFromVersion`, `SliceToVersion`, `FilterByTimestamp` — in-memory event slicing                                                                                                                                                                                                                                                         | ✅     |
| Command causality     | `WithCommandCausality(ctx, type, id)` + `CommandCausalityEnricher` — auto-tag events with the command that caused them                                                                                                                                                                                                                      | ✅     |
| Checkpoint            | `Checkpoint` struct + `CheckpointSink/Source/Store` interfaces for projection positioning                                                                                                                                                                                                                                                   | ✅     |
| Clock injection       | `Clock` type + `WithClock` option for deterministic testing                                                                                                                                                                                                                                                                                 | ✅     |
| Error taxonomy        | 5-family: Rejection / Conflict / Transient / Infrastructure / Corruption; 12 helper funcs (`New*`, `Wrap*`, `Classify`, `IsRetryable`); 16 sentinel errors                                                                                                                                                                                  | ✅     |
| Event reconstruction  | `ReconstructEventFromFields` — shared deserialization for all store implementations                                                                                                                                                                                                                                                         | ✅     |
| JSON metadata         | `MarshalMetadataJSON`, `UnmarshalMetadataJSON` — DB-safe metadata serialization                                                                                                                                                                                                                                                             | ✅     |

### Decider (Pure-Function Aggregate) ✅ FULLY_FUNCTIONAL

> `import "github.com/larsartmann/go-cqrs-lite/decider/v3"`

| Feature              | Detail                                                                                       | Status |
| -------------------- | -------------------------------------------------------------------------------------------- | ------ |
| Decider[State]       | `{Initial State; Fold func(State, Event) (State, error)}` — pure-function aggregate pattern  | ✅     |
| Repository[State]    | `NewRepository[State](store, publisher, decider, opts...)` — manages aggregate lifecycle     | ✅     |
| Execute              | `Repository.Execute(ctx, aggID, aggType, decide)` — load → decide → save → publish           | ✅     |
| Load                 | `Repository.Load(ctx, aggID, aggType)` — returns `(State, Version, error)`                   | ✅     |
| LoadAtVersion        | `Repository.LoadAtVersion(ctx, aggID, aggType, maxVersion)` — time-travel to version         | ✅     |
| LoadAtTime           | `Repository.LoadAtTime(ctx, aggID, aggType, maxTime)` — time-travel to timestamp             | ✅     |
| Snapshot integration | `WithSnapshotStore` + `WithSnapshotStrategy` + `WithCodec` — automatic snapshot optimization | ✅     |
| Context enrichment   | `WithEnricher` — injects metadata from context into events                                   | ✅     |
| OTel tracing         | OpenTelemetry spans for load/save/execute operations (opt-in)                                | ✅     |

**Sentinel errors:** `ErrNilStore`, `ErrNilPublisher`, `ErrNilFold`, `ErrLoadFailed`, `ErrFoldFailed`, `ErrSaveFailed`, `ErrIncompleteSnapshotConfig`

### Branded IDs ✅ FULLY_FUNCTIONAL

> `import "github.com/larsartmann/go-cqrs-lite/id/v3"`

| Feature                | Detail                                                                                                                                                        | Status |
| ---------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------- | ------ |
| Generic branded type   | `id.Of[T]` — phantom type parameter for compile-time safety                                                                                                   | ✅     |
| ULID-backed            | Binary-sortable, time-ordered, 16-byte binary form                                                                                                            | ✅     |
| 8 built-in types       | `AggregateID`, `EventID`, `CorrelationID`, `CausationID`, `RequestID`, `UserID`, `ClientID`, `CommandID`                                                      | ✅     |
| Custom branded types   | `type OrderID = id.Of[OrderMarker]` — users can create their own                                                                                              | ✅     |
| Exported markers       | All 8 phantom markers exported (`Aggregate`, `User`, `Correlation`, `Request`, `Causation`, `Client`, `Command`, `Event`) for downstream `BrandNamer` tooling | ✅     |
| All serialization      | JSON (incl. `null`), binary, text, SQL `Scan`/`Value`                                                                                                         | ✅     |
| Convenience funcs      | `New[T]()`, `Parse[T]()`, `ULID[T]()`, `FromPtr[T]()`, `CompareIDs[T]()`                                                                                      | ✅     |
| AggregateID derivation | `DeriveAggregateID()` — deterministic ID from namespace + key                                                                                                 | ✅     |
| Timestamp extraction   | `ULID(id)` extracts embedded timestamp                                                                                                                        | ✅     |

### Generic Dispatcher ✅ FULLY_FUNCTIONAL

> `import "github.com/larsartmann/go-cqrs-lite/dispatcher/v3"`

| Feature             | Detail                                                         | Status |
| ------------------- | -------------------------------------------------------------- | ------ |
| Generic Dispatcher  | `Dispatcher[H, M]` — type-safe handler + middleware dispatcher | ✅     |
| LifecycleMixin      | `Lifecycle` — `Close()` prevents all ops; thread-safe          | ✅     |
| Middleware ordering | Reverse-order middleware application at registration time      | ✅     |
| Duplicate guard     | `ErrHandlerAlreadyRegistered` — prevents double-registration   | ✅     |

### Idempotency ✅ FULLY_FUNCTIONAL

> `import "github.com/larsartmann/go-cqrs-lite/idempotency/v3"`

| Feature             | Detail                                                                                     | Status |
| ------------------- | ------------------------------------------------------------------------------------------ | ------ |
| Store interface     | `Store`: `Seen`, `Record`, `CheckAndRecord` — dedup opaque keys (command idempotency keys) | ✅     |
| MemoryStore         | `MemoryStore` — in-memory TTL store with background sweep + lazy deletion                  | ✅     |
| Atomic dedup        | `CheckAndRecord` — single-lock check+record prevents the TOCTOU race (exactly one winner)  | ✅     |
| TTL expiration      | Keys expire after a configurable duration; removed by sweeper and lazily on read           | ✅     |
| ErrDuplicate        | Conflict sentinel returned when a key is already recorded (maps to HTTP 409)               | ✅     |
| Dispatch middleware | `CommandIdempotency` — `command.Middleware` that deduplicates by `Command.ID()`            | ✅     |
| KeyExtractor        | `KeyExtractor` func type + `CommandIDKey` default extractor                                | ✅     |

**Sentinel errors:** `ErrDuplicate` (Conflict)

### Dead-Letter Queue ✅ FULLY_FUNCTIONAL

> `import "github.com/larsartmann/go-cqrs-lite/middleware/v3"`

| Feature           | Detail                                                                             | Status |
| ----------------- | ---------------------------------------------------------------------------------- | ------ |
| DeadLetterStore   | `DeadLetterStore` interface — `Store`, `List`, `Replay`, `Purge`                   | ✅     |
| MemoryDLQStore    | In-memory `DeadLetterStore` for dev/test                                           | ✅     |
| SQLDLQStore       | SQL-backed `DeadLetterStore` (Postgres, SQLite)                                    | ✅     |
| DeadLetterWrapper | Wraps a projection/event handler — captures poison messages, advances checkpoint   | ✅     |
| Error metadata    | Each dead-letter entry captures event, error, handler name, timestamp, retry count | ✅     |

---

## Reliability Infrastructure ✅ FULLY_FUNCTIONAL

The "reliability trio" (sans the deferred transactional outbox): idempotency +
DLQ (above) + managed projection host + scheduled deadlines.

### Managed Projection Host ✅ FULLY_FUNCTIONAL

> `import "github.com/larsartmann/go-cqrs-lite/projectionhost/v3"`

The "last loop every consumer rewrites", as a library module. Composes any
`event.SeekableJournal` + `event.CheckpointStore` + `projection.Projection`s
into a managed lifecycle.

| Feature                   | Detail                                                                                      | Status |
| ------------------------- | ------------------------------------------------------------------------------------------- | ------ |
| Host                      | `Host` — manages projection workers, lifecycle, and health                                  | ✅     |
| Per-projection goroutines | Each registered projection runs independently in its own goroutine                          | ✅     |
| Crash auto-restart        | Workers restart on panic/error with exponential backoff (configurable initial/max)          | ✅     |
| Checkpoint persistence    | Survives restarts — reads resume from the last committed checkpoint (no event loss)         | ✅     |
| Dead-letter queue         | `DeadLetterStore` / `MemoryDeadLetterStore` — poison messages captured, checkpoint advances | ✅     |
| Health / liveness         | `Status()` reports per-worker state + processed/errors/restarts counters                    | ✅     |
| Graceful drain            | `Stop()` waits for in-flight events (30s timeout)                                           | ✅     |
| RegisterAndWait           | Convenience: register + start + block until ctx cancelled                                   | ✅     |

Worker states: `idle`, `running`, `backoff`, `draining`, `stopped`, `failed`.
Reads directly from `event.SeekableJournal` — no message-bus dependency. For
live (push) delivery alongside replay, pair with `watermill/CatchUpSubscriber`.

### Scheduling (Durable Deadlines) ✅ FULLY_FUNCTIONAL

> `import "github.com/larsartmann/go-cqrs-lite/scheduling/v3"`

Classic ES need — "cancel the order 30 minutes after creation if still unpaid" —
as a library primitive.

| Feature             | Detail                                                                           | Status |
| ------------------- | -------------------------------------------------------------------------------- | ------ |
| TimerStore          | `TimerStore` interface — `Schedule`, `Due`, `MarkFired`, `Cancel`                | ✅     |
| MemoryTimerStore    | In-memory `TimerStore` for development and testing                               | ✅     |
| Scheduler           | Polls `Due()`, dispatches via callback, `MarkFired()`; retries failed dispatches | ✅     |
| Idempotent schedule | Re-scheduling the same `TimerID` is a no-op (safe on command retry)              | ✅     |
| Cancel              | Remove a timer before it fires (e.g. order paid → cancel timeout)                | ✅     |
| Configurable        | `WithPollInterval`, `WithMaxRetries`, `WithLogger`                               | ✅     |

---

## Schema Evolution ✅ FULLY_FUNCTIONAL

> `import "github.com/larsartmann/go-cqrs-lite/schema/v3"`

| Feature              | Detail                                                                               | Status |
| -------------------- | ------------------------------------------------------------------------------------ | ------ |
| Upcaster             | `Upcaster` interface — transforms old schema versions to newer on load               | ✅     |
| Upcaster constructor | `NewUpcaster(eventType, fromVersion, fn)` — version-gated transform                  | ✅     |
| Cycle detection      | Registry detects schema version revisits during upcast chain                         | ✅     |
| VersionedStore       | `VersionedStore` wraps any `event.Store` — transparent upcasting on all read methods | ✅     |
| Full load API        | `Load`, `LoadFromVersion`, `LoadToVersion`, `LoadToTimestamp` — all with upcasting   | ✅     |
| Schema validator     | `Validator` with `RegisterType[T]()`, strict/lenient modes, custom codecs (ADR-0017) | ✅     |
| Custom validators    | `RegisterTypeWithValidator[T](v, type, fn)` — business-rule validation after decode  | ✅     |

---

## Snapshot ✅ FULLY_FUNCTIONAL

> `import "github.com/larsartmann/go-cqrs-lite/snapshot/v3"`

| Feature          | Detail                                                          | Status |
| ---------------- | --------------------------------------------------------------- | ------ |
| Snapshot type    | `Snapshot` struct with AggregateRef, Version, State, SavedAt    | ✅     |
| Store interfaces | `SnapshotSink`, `SnapshotSource`, `SnapshotStore` (Sink+Source) | ✅     |
| Strategy         | `SnapshotStrategy` interface + `EveryNEvents(n)` built-in       | ✅     |
| Helper functions | `ShouldSnapshot`, `SaveSnapshot` — decider integration helpers  | ✅     |

---

## Payload Codec ✅ FULLY_FUNCTIONAL

> `import "github.com/larsartmann/go-cqrs-lite/codec/v3"`

| Feature            | Detail                                                                      | Status |
| ------------------ | --------------------------------------------------------------------------- | ------ |
| Codec interface    | `Codec` — `Encoding()`, `Encode(v)`, `Decode(data, v)`                      | ✅     |
| JSON codec         | `JSONCodec` — standard JSON encoding                                        | ✅     |
| CBOR codec         | `CBORCodec` — deterministic canonical CBOR with sorted map keys             | ✅     |
| CBOR compact codec | `CBORCompactCodec` — ~35% smaller via `toarray` positional mode             | ✅     |
| Raw passthrough    | `RawCodec` — `[]byte` pass-through (no encoding)                            | ✅     |
| BufferEncoder      | Optional `BufferEncoder` interface — zero-alloc encoding into caller buffer | ✅     |
| CBOR diagnostic    | `Diagnose(data)` — human-readable CBOR output for debugging                 | ✅     |
| Encoding constants | `EncodingJSON`, `EncodingCBOR`, `EncodingRaw`                               | ✅     |

---

## In-Memory Implementations 🧪 TESTING_ONLY

> `import "github.com/larsartmann/go-cqrs-lite/storage/memory/v3"`

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

> `import "github.com/larsartmann/go-cqrs-lite/middleware/v3"`

All **9 concerns** are provided for all 3 message types (command, event, query) — **27 domain-specific middleware factories** + generic `Middleware[M]` for custom message types.

### Generic Middleware ✅

| Feature               | Detail                                                                        | Status |
| --------------------- | ----------------------------------------------------------------------------- | ------ |
| Generic Handler[M]    | `Handler[M]` = `func(context.Context, M) error` — works with any message type | ✅     |
| Generic Middleware[M] | `Middleware[M]` = `func(Handler[M]) Handler[M]`                               | ✅     |
| MessageAdapter[M]     | `MessageAdapter[M]` — converts between message types                          | ✅     |
| Domain adapters       | `CommandAdapter`, `EventAdapter`, `QueryAdapter` — pre-built adapters         | ✅     |

### Logging ✅

| Factory                  | Logs                                                                                  |
| ------------------------ | ------------------------------------------------------------------------------------- |
| `CommandLogging(logger)` | `"command dispatching"` / `"succeeded"` / `"failed"` with type, aggregateID, duration |
| `EventLogging(logger)`   | Same pattern for events                                                               |
| `QueryLogging(logger)`   | Same pattern for queries                                                              |

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

| Factory                       | Span                                                                |
| ----------------------------- | ------------------------------------------------------------------- | --- |
| `CommandTracing(tracer)`      | `"command.handle"`, SpanKindServer, attributes: `cqrs.command.type` |
| `EventTracing(tracer)`        | `"event.handle"`, SpanKindConsumer, attributes: `cqrs.event.type`   |
| `QueryTracing(tracer)`        | `"query.handle"`, SpanKindServer, attributes: `cqrs.query.type`     |
| `EventPublishTracing(tracer)` | `"event.publish"`, SpanKindProducer                                 | ✅  |

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

> `import "github.com/larsartmann/go-cqrs-lite/transport/http/v3"`

| Feature     | Detail                                                                             | Status |
| ----------- | ---------------------------------------------------------------------------------- | ------ |
| SSEBroker   | Server-Sent Events broker with `AddClient`, `RemoveClient`, `ClientCount`, `Close` | ✅     |
| SSEHandler  | `net/http` handler for SSE connections with client ID extraction                   | ✅     |
| Thread-safe | Concurrent client management with proper channel lifecycle                         | ✅     |

### gRPC Transport ✅ FULLY_FUNCTIONAL

> `import "github.com/larsartmann/go-cqrs-lite/transport/grpc/v3"`

Remote gRPC transport for command & query dispatch (ADR-0025). Bridges gRPC clients to local dispatchers.

| Feature           | Detail                                                                  | Status |
| ----------------- | ----------------------------------------------------------------------- | ------ |
| CommandService    | `RegisterCommandService(srv, dispatcher)` — serves commands over gRPC   | ✅     |
| QueryService      | `RegisterQueryService(srv, dispatcher)` — serves queries over gRPC      | ✅     |
| CommandClient     | `NewCommandClient(conn)` — remote `command.Dispatcher` over a gRPC conn | ✅     |
| QueryClient       | `NewQueryClient(conn)` — remote `query.Dispatcher` over a gRPC conn     | ✅     |
| Protobuf contract | Generated `.proto` types in `transport/grpc/proto`                      | ✅     |

> Note: `transport/grpc` is wired into `go.work` and covered by CI. It builds under the workspace; per-module `GOWORK=off` isolation is blocked until [cockroachdb/errors#79](https://github.com/cockroachdb/errors/issues/79) drops the monolithic `google.golang.org/genproto` (it conflicts with grpc-go's split `genproto/googleapis/rpc`).

### Profiling ❌ REMOVED

Deleted — trivial `net/http/pprof` re-export. Use `import _ "net/http/pprof"` directly.

---

## Event Signing ✅ FULLY_FUNCTIONAL

> `import "github.com/larsartmann/go-cqrs-lite/signing/v3"`

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

> `import "github.com/larsartmann/go-cqrs-lite/encryption/v3"`

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

> `import "github.com/larsartmann/go-cqrs-lite/catalog/v3"`

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

> `import "github.com/larsartmann/go-cqrs-lite/catalog/v3/asyncapi"`

| Feature             | Detail                                                                                                      | Status |
| ------------------- | ----------------------------------------------------------------------------------------------------------- | ------ |
| Document generation | `Exporter.Export(catalog)` produces full AsyncAPI 3.0 `Document`                                            | ✅     |
| YAML output         | `Document.MarshalYAML()` — uses `go-faster/yaml`                                                            | ✅     |
| JSON output         | `Document.MarshalJSON()` — type-alias trick to avoid recursion                                              | ✅     |
| Server config       | `WithServer(name, host, protocol)` option (defaults: kafka, localhost:9092)                                 | ✅     |
| Channel mapping     | Commands → `receive`, Events with `Sends` → `send`, Events with `Receives` → `receive`, Queries → `receive` | ✅     |

> `import "github.com/larsartmann/go-cqrs-lite/catalog/v3/eventcatalog"`

| Feature        | Detail                                                                        | Status |
| -------------- | ----------------------------------------------------------------------------- | ------ |
| MDX generation | Services, commands, events, queries — all with YAML frontmatter               | ✅     |
| Schema files   | `schema.json` per message (only when schema is non-nil)                       | ✅     |
| Domain pages   | Domain frontmatter with service associations                                  | ✅     |
| Config files   | `eventcatalog.config.js`, `package.json` with `@eventcatalog/core` dependency | ✅     |
| LLM summary    | `llms.txt` — plain-text catalog summary for LLM consumption                   | ✅     |

> `import "github.com/larsartmann/go-cqrs-lite/catalog/v3/d2"`

| Feature             | Detail                                                      | Status |
| ------------------- | ----------------------------------------------------------- | ------ |
| D2 text export      | `Exporter.Export(cat)` produces D2 diagram syntax           | ✅     |
| Service nodes       | Color-coded rectangles per service with command/event/query | ✅     |
| Cross-service flows | Animated arrows between publishers and receivers            | ✅     |
| Domain grouping     | Domain labels with dashed "contains" links to services      | ✅     |
| Schema tooltips     | Field names and types shown on hover                        | ✅     |

> `import "github.com/larsartmann/go-cqrs-lite/catalog/v3/openapi"`

| Feature             | Detail                                                            | Status |
| ------------------- | ----------------------------------------------------------------- | ------ |
| Document generation | `Exporter.Export(catalog)` produces full OpenAPI 3.0.3 `Document` | ✅     |
| JSON output         | `Document` serializes to JSON                                     | ✅     |
| Schema generation   | Auto-generates JSON Schema from catalog types                     | ✅     |
| Base path support   | `WithBasePath(path)` option for API path prefix                   | ✅     |
| Description option  | `WithDescription(desc)` for document metadata                     | ✅     |

> `import "github.com/larsartmann/go-cqrs-lite/catalog/v3/docserver"`

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

### SQL Stores (PostgreSQL / SQLite)

> `import "github.com/larsartmann/go-cqrs-lite/storage/v3"`

| Feature                   | Detail                                                                                                                                       | Status |
| ------------------------- | -------------------------------------------------------------------------------------------------------------------------------------------- | ------ |
| PostgreSQL event store    | `NewSQLEventStore(db)` implements `event.Store`                                                                                              | ✅     |
| SQLite event store        | `NewSQLiteEventStore(db)` — `?` placeholders, `BLOB`/`TEXT` DDL                                                                              | ✅     |
| Custom dialect            | `NewSQLEventStoreWithDialect(db, d)` — pluggable SQL backend                                                                                 | ✅     |
| Schema DDL                | `Schema()` PostgreSQL, `SQLiteSchema()` for SQLite/Turso                                                                                     | ✅     |
| Per-table DDL             | `SnapshotSchema`, `CheckpointSchema` + SQLite variants                                                                                       | ✅     |
| Optimistic concurrency    | `Save` checks version in transaction                                                                                                         | ✅     |
| AppendBatch               | Appends without concurrency check                                                                                                            | ✅     |
| Full load API             | `Load`, `LoadFromVersion`, `LoadToVersion`, `LoadToTimestamp`                                                                                | ✅     |
| LoadBackwards             | Implements `event.BackwardsSource` — newest-first                                                                                            | ✅     |
| Time-travel SQL queries   | `LoadToVersion`, `LoadToTimestamp` with composite timestamp index                                                                            | ✅     |
| Journal / SeekableJournal | `ReadAll()`, `ReadFrom(afterEventID, limit)`                                                                                                 | ✅     |
| Stream loading            | `LoadStream()` returns cursor-based `sqlEventStream` — memory-efficient iteration                                                            | ✅     |
| Metadata persistence      | Full roundtrip: correlation IDs, user IDs, custom metadata                                                                                   | ✅     |
| SQL SnapshotStore         | PostgreSQL + SQLite variants, upsert, version-aware load, delete                                                                             | ✅     |
| SQL CheckpointStore       | PostgreSQL + SQLite variants, upsert, `sql.ErrNoRows` handling                                                                               | ✅     |
| SQL CommandStore          | `SQLCommandStore` implements `command.Store` — Save, AppendBatch, Load, LoadFromTimestamp, LoadToTimestamp                                   | ✅     |
| SQL Backend               | `SQLBackend` facade returning `EventStore()`, `SnapshotStore()`, `CheckpointStore()`, `CommandStore()`                                       | ✅     |
| AggregateProjection       | Maintains SQL read-model tables from event streams with tombstone detection                                                                  | ✅     |
| SQLAggregateReader        | `listing.AggregateReader` implementation reading from projection tables                                                                      | ✅     |
| DB helpers                | `OpenSQLite`, `OpenSQLiteInMemory`, `SQLiteInitSchema`, `SQLiteEnableWAL`, `ConfigureSQLitePool`, `ConfigureTursoPool`, `PostgresInitSchema` | ✅     |
| Dialect abstraction       | `Dialect` interface with `Placeholder`, `FormatTime`, `ScanTimeDest`, `ParseTime`, 5 schema methods                                          | ✅     |
| SQL sub-package           | `storage/sql` — `DBHandle`, `OwnedDBHandle`, generic `LoadWithSpan[T]`, `QueryRows[T]`, `ScanSlice[T]`, `ReconstructEvent`                   | ✅     |
| Close lifecycle           | No-op `Close()` — does not close `*sql.DB`; caller owns DB lifecycle                                                                         | ✅     |

### Pebble Key-Value Store

> `import "github.com/larsartmann/go-cqrs-lite/storage/pebble/v3"`

| Feature                | Detail                                                                                 | Status |
| ---------------------- | -------------------------------------------------------------------------------------- | ------ |
| EventStore             | `NewStore(db, logger)` implements `event.Store` + `Journal` + `SeekableJournal`        | ✅     |
| CBOR envelope          | Events serialized as CBOR with JSON backward compatibility layer                       | ✅     |
| Per-aggregate locking  | Sharded mutex pool (FNV-1a hash, 256 shards) — optimistic concurrency without sync.Map | ✅     |
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

> `import "github.com/larsartmann/go-cqrs-lite/storage/turso/v3"`

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

> `import "github.com/larsartmann/go-cqrs-lite/storage/turso/v3/indexing"`

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

---

## Stream Read Model ✅ FULLY_FUNCTIONAL

> `import "github.com/larsartmann/go-cqrs-lite/listing/v3"`

| Feature                     | Detail                                                                                               | Status |
| --------------------------- | ---------------------------------------------------------------------------------------------------- | ------ |
| AggregateReader             | Interface: `List(ctx, ListOptions) → Page[AggregateStatus]` — cursor-based aggregate listing         | ✅     |
| ListBuilder                 | Fluent API: `listing.NewListBuilder(reader).OfType("User").After(cursor).Limit(50).IncludeDeleted()` | ✅     |
| InMemoryAggregateReader     | Reads from `event.Journal.ReadAll()` — single-pass, no persistence                                   | ✅     |
| TombstonePolicy             | `Exclude` (default), `Include`, `Only` — controls visibility of soft-deleted aggregates              | ✅     |
| Page[T]                     | Cursor-based pagination with `HasMore` — no expensive TotalCount                                     | ✅     |
| AggregateListing            | Lightweight identity: ID, Type, Version, EventCount, LastEventAt                                     | ✅     |
| AggregateStatus             | Pairs AggregateListing with computed TombstoneStatus                                                 | ✅     |
| StatusMiddleware            | Event bus middleware that publishes aggregate status changes                                         | ✅     |
| CacheInvalidationMiddleware | Returns `event.PublishMiddleware` that invalidates reader cache                                      | ✅     |
| ListRefsFromStatus          | Helper that strips status from page                                                                  | ✅     |

---

## OpenTelemetry Helpers ✅ FULLY_FUNCTIONAL

> `import "github.com/larsartmann/go-cqrs-lite/otel/v3"`

| Feature            | Detail                                                                                                                                                                                       | Status |
| ------------------ | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ------ |
| Type aliases       | `Tracer`, `Span`, `SpanKind`, `KeyValue`, `Meter`, `Float64Histogram`, `Int64Counter` — re-exported from OTel                                                                                | ✅     |
| Tracer factory     | `NewTracer(component)` — creates OTel Tracer with standard instrumentation name                                                                                                              | ✅     |
| Meter factory      | `NewMeter(component)` — creates OTel Meter                                                                                                                                                   | ✅     |
| Span helpers       | `StartSpan`, `RecordError`, `EndWithError`, `AddSpanEvent`, `AggregateAttrs`, `CommandAttrs`, `EventAttrs`, `QueryAttrs`                                                                     | ✅     |
| Context helpers    | `SpanFromContext`, `TraceIDFromContext`, `SpanIDFromContext`                                                                                                                                 | ✅     |
| Attribute helpers  | `AttrString`, `AttrInt`, `AttrInt64`, `WithAttributes`, `WithSpanKind`, `ServiceResourceAttributes`                                                                                          | ✅     |
| Metric helpers     | `MetricWithAttributes`, `MetricWithDescription`, `MetricWithUnit`, `CounterAddWithAttributes`, `AddOption`                                                                                   | ✅     |
| Metric views       | `NewCQRSViews()`, `CQRSHistogramBoundaries` — OTel SDK views with optimized CQRS latency buckets for all `cqrs.*` instruments                                                                | ✅     |
| Correlation        | `WithCorrelationID`, `CorrelationIDFromContext` — baggage-based correlation propagation                                                                                                      | ✅     |
| W3C propagation    | `NewTextMapPropagator()` — W3C trace context + baggage propagator                                                                                                                            | ✅     |
| Logging helpers    | `ComponentLogger`, `ContextLogger` — structured logging with trace correlation                                                                                                               | ✅     |
| Standard constants | `AttrMessageKind`, `AttrCommandType`, `AttrEventType`, `AttrQueryType`, `AttrAggregateType`, `AttrAggregateID`, `AttrAggregateVersion`, `AttrEventCount`, `AttrProjectionName`, `AttrStatus` | ✅     |

---

## Watermill Adapter ✅ FULLY_FUNCTIONAL

> `import "github.com/larsartmann/go-cqrs-lite/watermill/v3"`

| Feature              | Detail                                                                                                    | Status |
| -------------------- | --------------------------------------------------------------------------------------------------------- | ------ |
| Event protocol       | Bidirectional `event.Event` ↔ Watermill `message.Message` via 15+ metadata keys                           | ✅     |
| PublisherAdapter     | `NewPublisherAdapter(publisher)` — wraps `event.Publisher` as `message.Publisher`                         | ✅     |
| SubscriberAdapter    | `NewSubscriberAdapter(bus)` — wraps `event.Bus` as `message.Subscriber`, feeds `<-chan *message.Message`  | ✅     |
| Full event fidelity  | 15 metadata keys preserve ID, type, aggregate, version, schema version, all metadata fields               | ✅     |
| **Command protocol** | Bidirectional `command.Command` ↔ Watermill `message.Message` (type, aggregate, tracing, custom metadata) | ✅     |
| **CommandBus**       | `NewCommandBus()` — full `command.Bus` backed by Watermill GoChannel + `WithCommandBackend` for brokers   | ✅     |
| **CommandPublisher** | `NewCommandPublisher(pub, topic)` — wraps `message.Publisher` as `command.Publisher`                      | ✅     |
| Custom metadata      | `custom.*` prefix preserves all custom metadata entries                                                   | ✅     |
| Correlation ID MW    | `CorrelationIDMiddleware()` — injects correlation ID into message metadata                                | ✅     |
| Retry middleware     | `NewRetryMiddleware(config)` + `DefaultRetryConfig()` — retry with backoff for handler errors             | ✅     |

---

## Prometheus Metrics Exporter ✅ FULLY_FUNCTIONAL

> `import "github.com/larsartmann/go-cqrs-lite/prometheus/v3"`

|| Feature | Detail | Status |
|| ------------------- | ------------------------------------------------------------------------------------------------- | ------ |
|| OTel→Prom bridge | `Setup()` — creates MeterProvider + HTTP handler backed by Prometheus registry | ✅ |
|| Custom registry | `WithRegistry(r)` — use a custom Prometheus registry | ✅ |
|| Handler options | `WithHandlerOptions(opts)` — configure promhttp.HandlerOpts | ✅ |

---

## Test Infrastructure 🧪 TESTING_ONLY

### eventtest 🧪

> `import "github.com/larsartmann/go-cqrs-lite/event/v3/eventtest"`

| Feature           | Detail                                                                                                                                                              | Status |
| ----------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ------ |
| FakeStore         | Full `Store + Journal + SeekableJournal` implementation with fluent override setters                                                                                | 🧪     |
| FakeBus           | Full `Bus` implementation with fluent override setters                                                                                                              | 🧪     |
| FakeSnapshotStore | `SnapshotStore` implementation with `SetSnapshot`, `SetLoadError`, `SetSaveError`                                                                                   | 🧪     |
| Event factories   | `NewTestEvent`, `NewEventOpts`, `QuickEvent`, `QuickEventOpts`, `MakeEvent`, `TamperEvent`                                                                          | 🧪     |
| Timeline fixtures | `MakeTimelineEvents`, `MakeThreeTimelineEvents`, `MakeLoadToTimestampFixtures`                                                                                      | 🧪     |
| Assertions        | `AssertGolden`, `AssertCallOrder`, `AssertMetricRecord`, `AssertEventType`, `AssertEventVersion`, and 10+ more                                                      | 🧪     |
| Store test suite  | `TestStoreSaveAndLoad`, `TestStoreConcurrencyConflict`, `TestStoreAppendBatch`, `TestStoreLoadFromVersion`, `TestStoreMetadataRoundtrip` — reusable suite functions | 🧪     |
| Handler factories | `AppendEventsHandler`, `NoopEventHandler`, `FailingEventHandler`, `PanicEventHandler`, `CallbackEventHandler`                                                       | 🧪     |

### Scenario Testing DSL ✅ FULLY_FUNCTIONAL

> `import "github.com/larsartmann/go-cqrs-lite/scenario/v3"`

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

> `go run github.com/larsartmann/go-cqrs-lite/cmd/cqrs-gen/v3`

| Feature             | Detail                                                                       | Status |
| ------------------- | ---------------------------------------------------------------------------- | ------ |
| AST-based scanning  | Parses Go source for `//cqrs:command <Name>` / `//cqrs:query <Name>` markers | ✅     |
| Typed handler gen   | Generates `Register<StructName>Handler` functions using `RegisterTyped[T]`   | ✅     |
| CLI flags           | `-type` (command/query), `-output` (file), `-pkg` (package name)             | ✅     |
| Recursive directory | Walks directories, skips `_test.go`, extracts markers from doc comments      | ✅     |

### API Stability Checker 🔧

> `go run github.com/larsartmann/go-cqrs-lite/cmd/api-stability/v3`

| Feature                | Detail                                                      | Status |
| ---------------------- | ----------------------------------------------------------- | ------ |
| Module scanning        | Parses all exported symbols from consumer-facing modules    | ✅     |
| Golden file comparison | Compares current API surface against `docs/api_surface.txt` | ✅     |
| Update mode            | `-update` flag regenerates golden file                      | ✅     |
| Diff reporting         | Reports REMOVED/NEW exports — CI gate for breaking changes  | ✅     |

### doc-check 🔧

> `go run github.com/larsartmann/go-cqrs-lite/cmd/doc-check/v3`

| Feature             | Detail                                                                  | Status |
| ------------------- | ----------------------------------------------------------------------- | ------ |
| Markdown scanning   | Scans `.md` files for Go code blocks                                    | ✅     |
| Symbol verification | Verifies Go import paths & qualified symbol refs actually exist in code | ✅     |
| Ghost detection     | Flags references to renamed/deleted symbols (docs-freshness gate)       | ✅     |

---

## Integration Tests ✅

> `import "github.com/larsartmann/go-cqrs-lite/integration/v3"`

| Feature              | Detail                                                                         | Status |
| -------------------- | ------------------------------------------------------------------------------ | ------ |
| Full flow E2E        | Command dispatch → decider → store → bus → projection → query → stream loading | ✅     |
| Chaos testing        | Error propagation, panic recovery, retry logic, context cancellation           | ✅     |
| Cross-module BDD     | Event, command, query, signing, encryption integration via Ginkgo v2           | ✅     |
| Simulation framework | `EventGenerator` — single/multi-aggregate event generation for testing         | ✅     |
| Benchmarking         | 17 scale benchmarks (10K-1M events), realistic pipeline/concurrent benchmarks  | ✅     |
| OTel integration     | End-to-end OpenTelemetry tracing verification                                  | ✅     |

---

## Examples 💡 DEMO

| Example                           | Detail                                                                                        |
| --------------------------------- | --------------------------------------------------------------------------------------------- |
| `example/todo/`                   | Full CQRS app: HTTP API, decider, projections, queries, Pebble + memory storage, saga pattern |
| `example/user/`                   | Event sourcing lifecycle: create, replay, mutate, signing, SSE, catalog, Docker packaging     |
| `example/encryption/`             | Signing + encryption middleware end-to-end                                                    |
| `example/deployer-first/`         | Deployer-first stack: CatchUpSubscriber + Materialize + Watermill EventBus                    |
| `example/deployer-first-multidb/` | Multi-database isolation: separate DBs for events, queries, and views                         |

**Not reference applications.** These demonstrate library usage patterns.

---

## Known Code Quality Issues

Found during code reviews. See `docs/planning/` for details.

| Issue                                                                                                                                                                                                                                                                       | Severity   | Module              |
| --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ---------- | ------------------- |
| ~~CommandJournal/SeekableCommandJournal in MemoryCommandStore untested~~ — **RESOLVED** (`0c0cd5b3`)                                                                                                                                                                        | ~~MEDIUM~~ | memory              |
| ~~Query store interfaces (PersistedQuery, QueryStore, QueryJournal) untested~~ — **RESOLVED** (`0c0cd5b3`)                                                                                                                                                                  | ~~MEDIUM~~ | query, memory       |
| ~~Query module lacks store-specific sentinel errors~~ — **RESOLVED** (`query/errors.go`)                                                                                                                                                                                    | ~~LOW~~    | query               |
| ~~command re-exports event types (module boundary violation)~~ — **DOCUMENTED AS INTENTIONAL** (`command/aggregate_ref.go:8-10`). Commands share the same aggregate identity as events; re-exporting `AggregateType`/`AggregateRef` is convenience, not layering violation. | ~~HIGH~~   | command             |
| ~~Reactive extensions not wired into dispatchers~~ — **DELETED** with `projection/` module (ADR-0030)                                                                                                                                                                       | ~~LOW~~    | event/command/query |
| ~~Pre-existing golden test drift (codec, middleware)~~ — **RESOLVED** (`8f2d2090`). Golden tests pass; the "drift" was caused by an invalid eventtest version in stack/go.mod blocking test execution, not actual content drift.                                            | ~~LOW~~    | codec, middleware   |

---

## Not Yet Implemented 📐 PLANNED

Features mentioned in project docs/planning but with **no production code yet**:

| Feature                   | Description                                      |
| ------------------------- | ------------------------------------------------ |
| PostgreSQL testcontainers | testcontainers-based real PG testing             |
| Documentation site        | Docusaurus/MkDocs/Hugo site                      |
| Transport adapters        | gRPC ✅, NATS/Redis (ADR-0025 accepted, no code) |

---

## Module Maturity Matrix

| Module                           | Import Path                        | Maturity        |
| -------------------------------- | ---------------------------------- | --------------- |
| `event`                          | `…/event/v3`                       | ✅ Production   |
| `event/eventtest`                | `…/event/v3/eventtest`             | 🧪 Test helper  |
| `command`                        | `…/command/v3`                     | ✅ Production   |
| `query`                          | `…/query/v3`                       | ✅ Production   |
| `decider`                        | `…/decider/v3`                     | ✅ Production   |
| `id`                             | `…/id/v3`                          | ✅ Production   |
| `dispatcher`                     | `…/dispatcher/v3`                  | ✅ Production   |
| `schema`                         | `…/schema/v3`                      | ✅ Production   |
| `snapshot`                       | `…/snapshot/v3`                    | ✅ Production   |
| `codec`                          | `…/codec/v3`                       | ✅ Production   |
| `kv`                             | `…/kv/v3`                          | ✅ Production   |
| `storage/memory`                 | `…/storage/memory/v3`              | 🧪 Test utility |
| `catalog`                        | `…/catalog/v3`                     | ✅ Production   |
| `catalog/asyncapi`               | `…/catalog/v3/asyncapi`            | ✅ Production   |
| `catalog/d2`                     | `…/catalog/v3/d2`                  | ✅ Production   |
| `catalog/openapi`                | `…/catalog/v3/openapi`             | ✅ Production   |
| `catalog/eventcatalog`           | `…/catalog/v3/eventcatalog`        | ✅ Production   |
| `catalog/docserver`              | `…/catalog/v3/docserver`           | ✅ Production   |
| `catalog/schema`                 | `…/catalog/v3/schema`              | ✅ Production   |
| `middleware`                     | `…/middleware/v3`                  | ✅ Production   |
| `integration`                    | `…/integration/v3`                 | ✅ Test suite   |
| `signing`                        | `…/signing/v3`                     | ✅ Production   |
| `signing/multisig`               | `…/signing/v3/multisig`            | ✅ Production   |
| `encryption`                     | `…/encryption/v3`                  | ✅ Production   |
| `storage`                        | `…/storage/v3`                     | ✅ Production   |
| `storage/sql`                    | `…/storage/v3/sql`                 | 🧪 Shared infra |
| `watermill`                      | `…/watermill/v3`                   | ✅ Production   |
| `listing`                        | `…/listing/v3`                     | ✅ Production   |
| `otel`                           | `…/otel/v3`                        | ✅ Production   |
| `storage/pebble`                 | `…/storage/pebble/v3`              | ✅ Production   |
| `storage/turso`                  | `…/storage/turso/v3`               | ✅ Production   |
| `storage/turso/indexing`         | `…/storage/turso/v3/indexing`      | ✅ Production   |
| `transport/http`                 | `…/transport/http/v3`              | ✅ Production   |
| `transport/grpc`                 | `…/transport/grpc/v3`              | ✅ Production   |
| `prometheus`                     | `…/prometheus/v3`                  | ✅ Production   |
| `testutil`                       | `…/testutil/v3`                    | 🧪 Test utility |
| `cmd/cqrs-gen`                   | `…/cmd/cqrs-gen/v3`                | 🔧 Tool         |
| `cmd/api-stability`              | `…/cmd/api-stability/v3`           | 🔧 Tool         |
| `cmd/doc-check`                  | `…/cmd/doc-check/v3`               | 🔧 Tool         |
| `example/user`                   | `…/example/user`                   | 💡 Demo         |
| `example/todo`                   | `…/example/todo`                   | 💡 Demo         |
| `example/encryption`             | `…/example/encryption`             | 💡 Demo         |
| `example/deployer-first`         | `…/example/deployer-first`         | 💡 Demo         |
| `example/deployer-first-multidb` | `…/example/deployer-first-multidb` | 💡 Demo         |
| `stack`                          | `…/stack/v3`                       | ✅ Production   |
| `stack/memory`                   | `…/stack/memory/v3`                | ✅ Production   |
| `stack/sqlite`                   | `…/stack/sqlite/v3`                | ✅ Production   |
| `stack/pebble`                   | `…/stack/pebble/v3`                | ✅ Production   |
| `stack/postgres`                 | `…/stack/postgres/v3`              | ✅ Production   |
| `stack/turso`                    | `…/stack/turso/v3`                 | ✅ Production   |
| `stack/bench`                    | `…/stack/bench/v3`                 | 🧪 Benchmarks   |

| `deriver` | `…/deriver/v3` | ✅ Production |
| `graph` | `…/graph/v3` | ✅ Production |
| `idempotency` | `…/idempotency/v3` | ✅ Production |
| `projection` | `…/projection/v3` | ✅ Production |
| `projectionhost` | `…/projectionhost/v3` | ✅ Production |
| `scenario` | `…/scenario/v3` | ✅ Production |
| `scheduling` | `…/scheduling/v3` | ✅ Production |
| `example/graph-demo` | `…/example/graph-demo` | 💡 Demo |
| `example/deployer-first-heterogeneous` | `…/example/deployer-first-heterogeneous` | 💡 Demo |
| `example/projectionhost` | `…/example/projectionhost` | 💡 Demo |

---

## Architecture Guarantees

| Guarantee              | Detail                                                                                                                                                                                                     |
| ---------------------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Lint posture           | ~68 findings across 15/37 modules after config tuning (noinlineerr disabled, makezero set to `always:false`). Mostly wrapcheck/err113/exhaustruct — style issues, not bugs. 22/37 modules pass lint clean. |
| Race-free              | `go test -race` passes across all modules                                                                                                                                                                  |
| Multi-module isolation | Each module has independent `go.mod`, no circular dependencies                                                                                                                                             |
| Strong types           | `event.Event` is a concrete type alias (`= *ImmutableEvent`); core store/bus are interfaces for DI                                                                                                         |
| Library, not framework | Import what you need, compose your own stack                                                                                                                                                               |
| Context-aware          | All handlers accept `context.Context`                                                                                                                                                                      |
| Errors as values       | Zero panics in production code, explicit error returns, classified sentinels via error-family taxonomy                                                                                                     |
| Defensive copies       | All public accessors return copies — callers cannot mutate internals                                                                                                                                       |
| Tombstone over delete  | Soft-delete via metadata — no `Delete` on Store                                                                                                                                                            |

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
