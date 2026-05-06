# Features

> Honest, verified inventory of what go-cqrs-lite actually does — not what it plans to do.

**Last audited:** 2026-05-03 · **Module count:** 9 · **Go version:** 1.26

## Status Legend

| Status                  | Meaning                                                    |
| ----------------------- | ---------------------------------------------------------- |
| ✅ FULLY_FUNCTIONAL     | Tested, production-quality, no known issues                |
| ⚠️ PARTIALLY_FUNCTIONAL | Works for happy paths but has gaps or known bugs           |
| 🔴 BROKEN               | Compiles but has correctness issues                        |
| 🧪 TESTING_ONLY         | Works but is explicitly designed for tests, not production |
| 📐 PLANNED              | Mentioned in docs/planning but no code exists              |
| 💡 DEMO                 | Example code, not a reusable module                        |

---

## Core CQRS

### Command Dispatcher ✅ FULLY_FUNCTIONAL

> `import "github.com/larsartmann/go-cqrs-lite/core/command"`

| Feature              | Detail                                                                        | Status |
| -------------------- | ----------------------------------------------------------------------------- | ------ |
| Command dispatch     | `Dispatcher.Dispatch(ctx, cmd)` routes to registered handler                  | ✅     |
| Handler registration | `Dispatcher.Register(cmdType, handler)` with duplicate guard                  | ✅     |
| Middleware chain     | `Dispatcher.Use(middleware...)` — applied at registration time, reverse order | ✅     |
| Lifecycle            | `Dispatcher.Close()` — rejects all ops after close                            | ✅     |
| Validation           | `New()` rejects empty type and zero aggregateID                               | ✅     |
| MustNew panic helper | `MustNew()` for test convenience                                              | ✅     |
| Catalog metadata     | `Catalogable` interface + `CatalogCore` embed for auto-documentation          | ✅     |

**Coverage:** 100.0%

---

### Query Dispatcher ✅ FULLY_FUNCTIONAL

> `import "github.com/larsartmann/go-cqrs-lite/core/query"`

| Feature              | Detail                                                                           | Status |
| -------------------- | -------------------------------------------------------------------------------- | ------ |
| Query dispatch       | `Dispatcher.Dispatch(ctx, query)` returns `(any, error)`                         | ✅     |
| Typed dispatch       | `DispatchTyped[T](ctx, dispatcher, query)` — generic type-safe result extraction | ✅     |
| Handler registration | Same pattern as command — duplicate guard, lifecycle                             | ✅     |
| Middleware chain     | Same pattern as command                                                          | ✅     |
| Pagination           | `Pagination` struct with `Page`, `PageSize`, `Offset()`, `Validate()`            | ✅     |
| Paginated results    | `PaginatedResult[T]` with `HasNext()`, `HasPrev()`, computed `TotalPages`        | ✅     |
| Catalog metadata     | `Catalogable` interface + `CatalogCore`                                          | ✅     |

**Defaults:** Page 1, PageSize 20, max 100. **Coverage:** 100.0%

---

### Event System ✅ FULLY_FUNCTIONAL

> `import "github.com/larsartmann/go-cqrs-lite/core/event"`

| Feature               | Detail                                                                                                                                                                               | Status |
| --------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ | ------ |
| Event creation        | `NewEvent()` with auto-generated `EventID` (ULID) and `time.Now()` timestamp                                                                                                         | ✅     |
| Event builder         | `Builder` fluent API: `WithPayload`, `WithCorrelationID`, `WithCausationID`, `WithUserID`, `Build`, `MustBuild`                                                                      | ✅     |
| Metadata              | `Metadata` struct: `CorrelationID`, `CausationID`, `UserID`, `RequestID`, `Source`, `IPAddress`, `UserAgent`, `Custom`                                                               | ✅     |
| 12 functional options | `WithEventID`, `WithOccurredAt`, `WithMetadata`, `WithCorrelationID`, `WithCausationID`, `WithUserID`, `WithRequestID`, `WithSource`, `WithIPAddress`, `WithUserAgent`, `WithCustom` | ✅     |
| Context enricher      | `ContextEnricher` extracts options from `context.Context`; `CompositeEnricher` chains multiple                                                                                       | ✅     |
| Defensive copies      | `Payload()` and `Metadata()` return copies — callers can't mutate internals                                                                                                          | ✅     |
| Typed values          | `Source`, `IPAddress`, `UserAgent`, `Version` — all parsed and validated                                                                                                             | ✅     |
| Event Bus interface   | `Bus` (with `io.Closer`): `Publish`, `Subscribe`, `SubscribeAll`                                                                                                                     | ✅     |
| Event Store interface | `Store` (with `io.Closer`): `Save` (optimistic concurrency), `AppendBatch`, `Load`, `LoadFromVersion`, `Delete`                                                                      | ✅     |
| JSON Codec            | `JSONCodec` using `go-json-experiment/json` (JSON v2)                                                                                                                                | ✅     |
| DecodePayload[T]      | `DecodePayload[T](evt, codec)` — type-safe payload deserialization                                                                                                                   | ✅     |
| Catalog metadata      | `Catalogable` interface + `CatalogCore`                                                                                                                                              | ✅     |

**Coverage:** 93.6%

---

### Aggregate & Repository ✅ FULLY_FUNCTIONAL

> `import "github.com/larsartmann/go-cqrs-lite/core/aggregate"`

| Feature                | Detail                                                                                          | Status |
| ---------------------- | ----------------------------------------------------------------------------------------------- | ------ |
| Aggregate root base    | `Core` provides `ID`, `Type`, `Version`, `RecordEvent`, `UncommittedChanges`, `LoadFromHistory` | ✅     |
| Event sourcing         | `EventSourcedRepository` — Save (persist + publish) and Load (replay from store)                | ✅     |
| Optimistic concurrency | `Save` passes `expectedVersion` to `Store.Save`                                                 | ✅     |
| Snapshot support       | `WithSnapshotStore` option — loads from snapshot then replays remaining events                  | ✅     |
| Snapshot strategy      | `event.SnapshotStrategy` interface + `EveryNEvents(n)` — shared with decider                    | ✅     |
| Snapshot codec         | `WithCodec` option for custom snapshot serialization                                            | ✅     |
| Transactional outbox   | `WithOutbox` option — events go to outbox instead of direct bus publish                         | ✅     |
| ISP Publisher          | Repository accepts `event.Publisher` (not full `Bus`) — backward-compatible                     | ✅     |
| Defensive copies       | `UncommittedChanges()` returns a copy; `MarkChangesAsCommitted()` reuses backing array          | ✅     |

**Coverage:** 95.5%

---

### Branded IDs ✅ FULLY_FUNCTIONAL

> `import "github.com/larsartmann/go-cqrs-lite/core/pkg/id"`

| Feature              | Detail                                                                          | Status |
| -------------------- | ------------------------------------------------------------------------------- | ------ |
| Generic branded type | `id.Of[T]` — phantom type parameter for compile-time safety                     | ✅     |
| ULID-backed          | Binary-sortable, time-ordered, 16-byte binary form                              | ✅     |
| 6 built-in types     | `AggregateID`, `EventID`, `CorrelationID`, `CausationID`, `RequestID`, `UserID` | ✅     |
| Custom branded types | `type OrderID = id.Of[OrderMarker]` — users can create their own                | ✅     |
| All serialization    | JSON (incl. `null`), binary, text, SQL `Scan`/`Value`                           | ✅     |
| Convenience funcs    | `New[T]()`, `Parse[T]()`, `MustParse[T]()`, `Ptr()`, `FromPtr()`                | ✅     |
| Comparison           | `Equal`, `Compare` (lexicographic), `IsZero`, `Or` (default value)              | ✅     |
| fmt.Formatter        | `%s`, `%v`, `%#v`, `%q`                                                         | ✅     |
| Timestamp extraction | `ULID(id)` extracts embedded timestamp                                          | ✅     |

**Coverage:** 100.0%

---

### Projections ⚠️ PARTIALLY_FUNCTIONAL

> `import "github.com/larsartmann/go-cqrs-lite/core/event"`

| Feature                   | Detail                                                                 | Status |
| ------------------------- | ---------------------------------------------------------------------- | ------ |
| Projection interface      | `Projection`: `Name`, `Handle(ctx, Event)`, `EventTypes()` (nil = all) | ✅     |
| ProjectionFunc            | Convenience adapter from a function                                    | ✅     |
| CheckpointStore interface | `Load`/`Save` last processed event ID per projection                   | ✅     |
| InMemoryRunner            | Single-process runner with per-projection checkpointing, thread-safe   | ✅     |
| HandleParallel            | Concurrent dispatch to all matching projections via goroutines         | ✅     |
| Event type filtering      | Runner filters events by `Projection.EventTypes()`                     | ✅     |
| OutboxPublisher           | Background goroutine polls outbox and publishes to bus                 | ✅     |
| PublishNow                | Synchronous poll-publish-ack for testing or manual triggering          | ✅     |

**Gaps in InMemoryRunner:**

- No retry or dead-letter mechanism
- No background polling (push-model only via `Handle`)

**Coverage:** Tested via unit + integration tests

---

### Event Upcasting ✅ FULLY_FUNCTIONAL

> `import "github.com/larsartmann/go-cqrs-lite/core/event"`

| Feature                 | Detail                                                                                    | Status |
| ----------------------- | ----------------------------------------------------------------------------------------- | ------ |
| Upcaster interface      | `SourceType()`, `SourceVersion()`, `Upcast(Event) (*Core, error)`                         | ✅     |
| UpcasterFunc            | Convenience adapter from a function                                                       | ✅     |
| UpcasterRegistry        | Thread-safe, sorted by source version, chains upcasters sequentially with cycle detection | ✅️     |
| Version-sorted chaining | Upcasters sorted by `SourceVersion()` ascending                                           | ✅     |

| Cycle detection | Defense-in-depth: detects schema version revisits during upcast chain | ✅ |

**Coverage:** Tested

---

## In-Memory Implementations

### Memory Store, Bus, Snapshot, Outbox, Checkpoint 🧪 TESTING_ONLY

> `import "github.com/larsartmann/go-cqrs-lite/memory"`

| Component             | Detail                                                                        | Status |
| --------------------- | ----------------------------------------------------------------------------- | ------ |
| MemoryStore           | `event.Store` implementation with optimistic concurrency, defensive copies    | 🧪     |
| MemoryBus             | `event.Bus` with typed `Subscribe` + `SubscribeAll` + middleware              | 🧪     |
| MemorySnapshotStore   | `event.SnapshotStore` with deep-copy snapshots, version-aware `LoadAtVersion` | 🧪     |
| MemoryOutboxStore     | `event.Outbox` with append/poll/ack, auto-incrementing IDs                    | 🧪     |
| MemoryCheckpointStore | `event.CheckpointStore` for projection checkpointing                          | 🧪     |

**Intended use:** Testing and development only. All implementations are thread-safe (`sync.RWMutex`), support `Close()` lifecycle, and return defensive copies. Not designed for production workloads.

**Coverage:** 99.1%

---

## Middleware Suite ✅ FULLY_FUNCTIONAL

> `import "github.com/larsartmann/go-cqrs-lite/middleware"`

All 6 concerns are provided for all 3 message types (command, event, query) — **18 middleware factories** total.

### Logging ✅

| Factory                  | Logs                                                                                  |
| ------------------------ | ------------------------------------------------------------------------------------- |
| `CommandLogging(logger)` | `"command dispatching"` / `"succeeded"` / `"failed"` with type, aggregateID, duration |
| `EventLogging(logger)`   | Same pattern for events                                                               |
| `QueryLogging(logger)`   | Same pattern for queries                                                              |

Accepts any `Logger` interface (`Info`, `Error`). `SlogAdapter` adapts `*slog.Logger`.

### Metrics ✅

| Factory                    | Records                                                              |
| -------------------------- | -------------------------------------------------------------------- |
| `CommandMetrics(recorder)` | `"command_success"` / `"command_error"` with duration and type label |
| `EventMetrics(recorder)`   | Same for events                                                      |
| `QueryMetrics(recorder)`   | Same for queries                                                     |

Accepts any `MetricsRecorder` interface (`Observe`).

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

**Config:** `MaxAttempts`, `InitialDelay`, `MaxDelay`, `Multiplier`, `IsRetryable` predicate. Jitter uses `crypto/rand`.

### Tracing ✅

| Factory                  | Span                                                                |
| ------------------------ | ------------------------------------------------------------------- |
| `CommandTracing(tracer)` | `"command.handle"`, SpanKindServer, attributes: `cqrs.command.type` |
| `EventTracing(tracer)`   | `"event.handle"`, SpanKindConsumer, attributes: `cqrs.event.type`   |
| `QueryTracing(tracer)`   | `"query.handle"`, SpanKindServer, attributes: `cqrs.query.type`     |

OpenTelemetry via `go.opentelemetry.io/otel/trace`. Caller provides the `Tracer`.

### Validation ✅

| Factory                        | Behavior                                                  |
| ------------------------------ | --------------------------------------------------------- |
| `CommandValidation(validator)` | Calls validator before handler; returns descriptive error |
| `EventValidation(validator)`   | Same                                                      |
| `QueryValidation(validator)`   | Same                                                      |

**Coverage:** 100.0%

---

## Auto-Documentation

### Catalog System ✅ FULLY_FUNCTIONAL

> `import "github.com/larsartmann/go-cqrs-lite/catalog"`

| Feature                  | Detail                                                                                                          | Status |
| ------------------------ | --------------------------------------------------------------------------------------------------------------- | ------ |
| Registry                 | Thread-safe builder: `AddService`, `AddCommand`, `AddEvent`, `AddQuery`, `AddDomain`, `AddChannel`, `Build()`   | ✅     |
| Schema reflection        | `SchemaFromType[T]()` — auto-generates JSON Schema from Go structs via `reflect`                                | ✅     |
| Struct tag support       | `json` (name, omitempty), `doc`/`description`, `format`, `default`, `enum`, `nullable`, `deprecated`, `pattern` | ✅     |
| Catalog adapters         | `CatalogBuilder` fluent API with `AddCommandFromType[T]`, `AddEventFromType[T]`, `AddQueryFromType[T]`          | ✅     |
| Dispatcher introspection | `FromCommandDispatcher`, `FromQueryDispatcher` — auto-import entries from live dispatchers                      | ✅     |
| Immutable catalog        | `Build()` returns deep-copied, immutable `*Catalog`                                                             | ✅     |

**Coverage:** 94.4%

---

### AsyncAPI 3.0 Export ✅ FULLY_FUNCTIONAL

> `import "github.com/larsartmann/go-cqrs-lite/catalog/asyncapi"`

| Feature             | Detail                                                                                                      | Status |
| ------------------- | ----------------------------------------------------------------------------------------------------------- | ------ |
| Document generation | `Exporter.Export(catalog)` produces full AsyncAPI 3.0 `Document`                                            | ✅     |
| YAML output         | `Document.MarshalYAML()` — uses `go-faster/yaml`                                                            | ✅     |
| JSON output         | `Document.MarshalJSON()` — type-alias trick to avoid recursion                                              | ✅     |
| Server config       | `WithServer(name, host, protocol)` option (defaults: kafka, localhost:9092)                                 | ✅     |
| Channel mapping     | Commands → `receive`, Events with `Sends` → `send`, Events with `Receives` → `receive`, Queries → `receive` | ✅     |
| Examples            | `toExamples()` converts `json.RawMessage` to AsyncAPI examples                                              | ✅     |

**Coverage:** 95.9%

---

### EventCatalog Export ✅ FULLY_FUNCTIONAL

> `import "github.com/larsartmann/go-cqrs-lite/catalog/eventcatalog"`

| Feature        | Detail                                                                        | Status |
| -------------- | ----------------------------------------------------------------------------- | ------ |
| MDX generation | Services, commands, events, queries — all with YAML frontmatter               | ✅     |
| Schema files   | `schema.json` per message (only when schema is non-nil)                       | ✅     |
| Domain pages   | Domain frontmatter with service associations                                  | ✅     |
| Config files   | `eventcatalog.config.js`, `package.json` with `@eventcatalog/core` dependency | ✅     |
| LLM summary    | `llms.txt` — plain-text catalog summary for LLM consumption                   | ✅     |

**Coverage:** 95.6%

---

### D2 Diagram Export ✅ FULLY_FUNCTIONAL

> `import "github.com/larsartmann/go-cqrs-lite/catalog/d2"`

| Feature             | Detail                                                      | Status |
| ------------------- | ----------------------------------------------------------- | ------ |
| D2 text export      | `Exporter.Export(cat)` produces D2 diagram syntax           | ✅     |
| Service nodes       | Color-coded rectangles per service with command/event/query | ✅     |
| Cross-service flows | Animated arrows between publishers and receivers            | ✅     |
| Domain grouping     | Domain labels with dashed "contains" links to services      | ✅     |
| Schema tooltips     | Field names and types shown on hover                        | ✅     |
| Options             | `WithDescription`, `WithDirection` for layout customization | ✅     |

**Coverage:** 97.6%

---

## SQL Event Store ✅ FULLY_FUNCTIONAL

> `import "github.com/larsartmann/go-cqrs-lite/storage"`

| Feature                         | Detail                                                               | Status |
| ------------------------------- | -------------------------------------------------------------------- | ------ |
| PostgreSQL event store          | `SQLEventStore` implements `event.Store`                             | ✅     |
| SQLite event store              | `SQLiteEventStore` — `?` placeholders, `BLOB`/`TEXT` DDL             | ✅     |
| Turso connector (local)         | `OpenTurso(path)` — returns `*sql.DB` for local Turso database       | ✅     |
| Turso connector (sync)          | `OpenTursoSync(ctx, path, url, token)` — `*sql.DB` + Push/Pull       | ✅     |
| Turso in-memory                 | `OpenTursoInMemory()` — `:memory:` for testing                       | ✅     |
| Schema DDL                      | `Schema()` PostgreSQL, `SQLiteSchema()` for SQLite/Turso             | ✅     |
| Optimistic concurrency          | `Save` checks version in transaction                                 | ✅     |
| AppendBatch                     | Appends without concurrency check                                    | ✅     |
| Load / LoadFromVersion / Delete | All implemented for both engines                                     | ✅     |
| Metadata persistence            | Full roundtrip: correlation IDs, user IDs, custom metadata           | ✅     |
| SQL SnapshotStore               | PostgreSQL + SQLite variants, upsert, version-aware load, delete     | ✅     |
| SQL CheckpointStore             | PostgreSQL + SQLite variants, upsert, `sql.ErrNoRows` handling       | ✅     |
| SQL Outbox                      | PostgreSQL + SQLite variants, append/poll/ack                        | ✅     |
| TransactionalStore              | Atomic save + outbox append, both engines                            | ✅     |
| Close lifecycle                 | No-op `Close()` — does not close `*sql.DB`; caller owns DB lifecycle | ✅     |

**Remaining gaps:**

| Issue                           | Severity  | Detail                                                             |
| ------------------------------- | --------- | ------------------------------------------------------------------ |
| No PostgreSQL integration tests | ⚠️ MEDIUM | Unit tests use go-sqlmock only; no real PostgreSQL verification    |
| `SQLEventStoreOption` unused    | ⚠️ LOW    | Type does not exist — consider adding table name or logger options |

**Coverage:** 94.8% (SQL event store, checkpoint store, snapshot store with go-sqlmock)

---

## Test Helpers 🧪 TESTING_ONLY

> `import "github.com/larsartmann/go-cqrs-lite/testhelpers"`

| Helper                                                         | Purpose                                              |
| -------------------------------------------------------------- | ---------------------------------------------------- |
| `FakeStore`                                                    | Configurable `event.Store` with overridable `SaveFn` |
| `FakeBus`                                                      | Records published events, injectable `PublishErr`    |
| `FakeSnapshotStore`                                            | Configurable snapshot load/save with error injection |
| `FakeOutbox`                                                   | Simple outbox with auto-generated IDs                |
| `AppendEventsHandler`                                          | Handler that collects events into a slice            |
| `NoopCommandHandler` / `NoopEventHandler` / `NoopQueryHandler` | No-op handlers                                       |
| `FailingCommandHandler` / `FailingEventHandler`                | Handlers that always error                           |
| `PanicCommandHandler` / `PanicEventHandler`                    | Handlers that panic                                  |
| `CallbackCommandHandler` / `CallbackEventHandler`              | Sets a bool flag                                     |
| `CommandMiddleware` / `EventMiddleware`                        | Call-order tracking middleware                       |
| `TestMetrics`                                                  | `MetricsRecorder` that records names and durations   |
| `NewTestEvent`                                                 | Creates a minimal test event                         |

---

## Example 💡 DEMO

> `example/user/` — `go run ./example/user`

Minimal CLI demo showing the event sourcing lifecycle:

1. Create `MemoryStore` + `MemoryBus` + `Repository`
2. Create User aggregate, record event, save
3. Load fresh User instance, replay, mutate, save again

**Not a reference application.** Uses raw byte payloads (not JSON), no command/query dispatchers, no HTTP layer, `ApplySnapshot` is a no-op.

---

## Not Yet Implemented 📐 PLANNED

Features mentioned in project docs/planning but with **no production code**:

| Feature                | Description                                  | Notes                                                     |
| ---------------------- | -------------------------------------------- | --------------------------------------------------------- |
| Watermill module       | Pub/sub adapter (Kafka, NATS, etc.)          | `docs/planning/2026-04-23_WATERMILL_PRO_CONTRA.md` exists |
| Saga / Process Manager | Long-running process orchestration           | `docs/planning/SAGA_DESIGN.md` exists                     |
| Tagged releases        | Semantic versioning and Go module publishing | All modules at v0.0.0                                     |

---

## Module Maturity Matrix

| Module                 | Import Path              | Code Lines | Tests       | Coverage | Maturity        |
| ---------------------- | ------------------------ | ---------- | ----------- | -------- | --------------- |
| `core/command`         | `…/core/command`         | ~250       | 10          | 100.0%   | ✅ Production   |
| `core/query`           | `…/core/query`           | ~300       | 18          | 100.0%   | ✅ Production   |
| `core/event`           | `…/core/event`           | ~1100      | 70+         | 94.4%    | ✅ Production   |
| `core/aggregate`       | `…/core/aggregate`       | ~250       | 27          | 95.5%    | ✅ Production   |
| `core/decider`         | `…/core/decider`         | ~240       | 22          | 95.0%    | ✅ Production   |
| `core/pkg/id`          | `…/core/pkg/id`          | ~400       | 30+         | 100.0%   | ✅ Production   |
| `core/pkg/dispatcher`  | `…/core/pkg/dispatcher`  | ~200       | 24          | 100.0%   | ✅ Production   |
| `memory`               | `…/memory`               | ~500       | Extensive   | 99.1%    | 🧪 Test utility |
| `catalog`              | `…/catalog`              | ~400       | Extensive   | 94.4%    | ✅ Production   |
| `catalog/asyncapi`     | `…/catalog/asyncapi`     | ~280       | Golden-file | 95.9%    | ✅ Production   |
| `catalog/d2`           | `…/catalog/d2`           | ~340       | 14          | 97.6%    | ✅ Production   |
| `catalog/eventcatalog` | `…/catalog/eventcatalog` | ~350       | Golden-file | 95.6%    | ✅ Production   |
| `middleware`           | `…/middleware`           | ~600       | Extensive   | 100.0%   | ✅ Production   |
| `testhelpers`          | `…/testhelpers`          | ~325       | N/A         | N/A      | 🧪 Test utility |
| `integration`          | `…/integration`          | 0 prod     | ~50 cases   | N/A      | ✅ Test suite   |
| `storage`              | `…/storage`              | ~614       | 31          | 94.8%    | ⚠️ Partial      |
| `example/user`         | `…/example/user`         | ~125       | 0           | N/A      | 💡 Demo         |

---

## Architecture Guarantees

| Guarantee              | Detail                                                                           |
| ---------------------- | -------------------------------------------------------------------------------- |
| Zero lint issues       | 125+ linters via `.golangci.yml`, strict config                                  |
| Race-free              | `go test -race` passes across all modules                                        |
| Multi-module isolation | Each module has independent `go.mod`, no circular dependencies                   |
| Interface-first        | All core types are interfaces — provide your own implementations                 |
| Library, not framework | Import what you need, compose your own stack                                     |
| Context-aware          | All handlers accept `context.Context`                                            |
| Errors as values       | No panics in production code, explicit error returns, sentinel errors + wrapping |
