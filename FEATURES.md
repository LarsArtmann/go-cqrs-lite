# Features

> Honest, verified inventory of what go-cqrs-lite actually does — not what it plans to do.

**Last audited:** 2026-06-08 (updated) · **Module count:** 30 (22 library + 6 examples + 2 cmd) · **Go version:** 1.26.3

## Status Legend

|| Status | Meaning |
|| ----------------------- | ---------------------------------------------------------- |
|| ✅ FULLY_FUNCTIONAL | Tested, production-quality, no known issues |
|| ⚠️ PARTIALLY_FUNCTIONAL | Works for happy paths but has gaps or known bugs |
|| 🔴 BROKEN | Compiles but has correctness issues |
|| 🧪 TESTING_ONLY | Works but is explicitly designed for tests, not production |
|| 🧪 EXPERIMENTAL | New/reactive features, API may change |
|| 📐 PLANNED | Mentioned in docs/planning but no code exists |
|| 💡 DEMO | Example code, not a reusable module |
|| 🔧 TOOL | Code generation or developer tooling |

---

## Core CQRS

### Command Dispatcher ✅ FULLY_FUNCTIONAL

> `import "github.com/larsartmann/go-cqrs-lite/command"`

|| Feature | Detail | Status |
|| -------------------- | ------------------------------------------------------------------------------------ | ------ |
|| Command dispatch | `Dispatcher.Dispatch(ctx, cmd)` routes to registered handler | ✅ |
|| Handler registration | `Dispatcher.Register(cmdType, handler)` with duplicate guard | ✅ |
|| Middleware chain | `Dispatcher.Use(middleware...)` — applied at registration time, reverse order | ✅ |
|| Lifecycle | `Dispatcher.Close()` — rejects all ops after close | ✅ |
|| Validation | `New()` rejects empty type and zero aggregateID | ✅ |
|| MustNew panic helper | `MustNew()` for test convenience | ✅ |
|| TypedHandler[T] | `RegisterTyped[T](d, type, handler)` — type-safe handler receiving `T` not `Command` | ✅ |
|| Command metadata | `Metadata` struct with CorrelationID, CausationID, UserID, RequestID | ✅ |
|| Metadata options | `WithCorrelationID`, `WithCausationID`, `WithUserID`, `WithRequestID` | ✅ |
|| Catalog introspection| `CatalogDispatcher` embedded — auto-import entries from live dispatchers | ✅ |

**Sentinel errors:** `ErrHandlerNotFound`, `ErrDispatcherClosed`, `ErrEmptyCommandType`, `ErrNilAggregateID`, `ErrTypeAssertion`

### Query Dispatcher ✅ FULLY_FUNCTIONAL

> `import "github.com/larsartmann/go-cqrs-lite/query"`

|| Feature | Detail | Status |
|| -------------------- | -------------------------------------------------------------------------------- | ------ |
|| Query dispatch | `Dispatcher.Dispatch(ctx, query)` returns `(any, error)` | ✅ |
|| Typed dispatch | `DispatchTyped[T](ctx, dispatcher, query)` — generic type-safe result extraction | ✅ |
|| Handler registration | Same pattern as command — duplicate guard, lifecycle | ✅ |
|| Middleware chain | Same pattern as command | ✅ |
|| Pagination | `Pagination` struct with `Page`, `PageSize`, `Offset()`, `Validate()` | ✅ |
|| Paginated results | `PaginatedResult[T]` with `HasNext()`, `HasPrev()`, computed `TotalPages` | ✅ |
|| TypedHandler[T] | `RegisterTyped[T]` — type-safe handler returning `(T, error)` | ✅ |
|| Catalog introspection| `CatalogDispatcher` embedded — same pattern as command | ✅ |

**Defaults:** Page 1, PageSize 20, max 100.
**Sentinel errors:** `ErrQueryNotSupported`, `ErrDispatcherClosed`, `ErrEmptyQueryType`

---

### Event System ✅ FULLY_FUNCTIONAL

> `import "github.com/larsartmann/go-cqrs-lite/event"`

|| Feature | Detail | Status |
|| --------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ | ------ |
|| Event creation | `NewEvent()` with auto-generated `EventID` (ULID) and `time.Now()` timestamp | ✅ |
|| Auto-marshal creation | `New()` — creates event from `any` payload (auto-json for structs/maps) | ✅ |
|| Batch creation | `NewEvents()` / `MustNewEvents()` — batch event creation with auto-incrementing versions | ✅ |
|| 19 functional options | `WithEventID`, `WithOccurredAt`, `WithMetadata`, `WithCorrelationID`, `WithCausationID`, `WithUserID`, `WithRequestID`, `WithSource`, `WithIPAddress`, `WithUserAgent`, `WithCustom`, `WithSchemaVersion`, `WithEncoding`, `WithNewCodec`, `WithClock`, `WithClientID`, `WithClientOccurredAt`, `WithDeadline`, `FromContext` | ✅ |
|| Metadata | `Metadata` struct: `CorrelationID`, `CausationID`, `UserID`, `RequestID`, `Source`, `IPAddress`, `UserAgent`, `Custom` | ✅ |
|| Context enricher | `ContextEnricher` extracts options from `context.Context`; `CompositeEnricher` chains multiple | ✅ |
|| Defensive copies | `Payload()` and `Metadata()` return copies — callers can't mutate internals | ✅ |
|| Event.Clone() | Deep copy of `ImmutableEvent` | ✅ |
|| Typed values | `Source`, `IPAddress`, `UserAgent`, `Version`, `SchemaVersion` — all parsed and validated | ✅ |
|| Version arithmetic | `Version.Add`, `Sub`, `Mod`, `Cmp`, `IsPositive` — phantom type math | ✅ |
|| Event Bus interface | `Bus` (with `io.Closer`): `Publish`, `Subscribe`, `SubscribeAll`, `Use`, `UsePublish` | ✅ |
|| PublishMiddleware | `Bus.UsePublish(mw)` — middleware for publish path | ✅ |
|| PublisherFunc adapter | `PublisherFunc` — function adapter for `Publisher` | ✅ |
|| Event Store interface | `Store = EventSink + EventSource` (with `io.Closer`): `Save` (optimistic concurrency), `AppendBatch`, `Load`, `LoadFromVersion`, `LoadToVersion`, `LoadToTimestamp` | ✅ |
|| ISP split | `EventSink` (write) + `EventSource` (read) — fine-grained dependency injection | ✅ |
|| Journal | `ReadAll()` returns all events ordered by `occurred_at ASC` — for projection replay | ✅ |
|| SeekableJournal | `ReadFrom(ctx, afterEventID, limit)` — efficient projection catch-up | ✅ |
|| BackwardsSource | `LoadBackwards(ctx, aggType, aggID)` — loads events in reverse version order (`BackwardsLoader` alias kept) | ✅ |
|| BackwardsSource | `LoadBackwards(ctx, aggType, aggID)` — loads events in reverse version order (`BackwardsLoader` alias kept) | ✅ |
|| TombstoneStatus | `Active`, `Tombstoned`, `Undetermined` — tri-state enum for soft-delete; `DetectTombstone`, `MarkTombstone`, `MarkRebirth` | ✅ |
|| Time-travel queries | `LoadToVersion` and `LoadToTimestamp` — read aggregate state at a point in time | ✅ |
|| Projection interface | `Projection`: `Name`, `Handle(ctx, Event)`, `EventTypes()` (nil = all) | ✅ |
|| BatchProjection | Optional interface extending `Projection` with `HandleBatch` for throughput | ✅ |
|| Context replay marker | `WithReplay(ctx, true)` — marks context as replay; handlers can distinguish | ✅ |
|| JSON Codec | `JSONcodec` using `encoding/json` | ✅ |
|| DecodePayload[T] | `DecodePayload[T](evt, codec)` — type-safe payload deserialization | ✅ |
|| DecodePayloads[T] | `DecodePayloads[T](events, codec)` — batch payload deserialization | ✅ |
|| Upcaster system | `Upcaster` interface + `UpcasterRegistry` + `NewUpcaster` + `NewVersionedStore` — schema migration on load | ✅ |
|| Cycle detection | UpcasterRegistry detects schema version revisits during upcast chain | ✅ |
|| Clock injection | `Clock` type + `WithClock` option for deterministic testing | ✅ |
|| Error taxonomy | 5-family: Rejection / Conflict / Transient / Infrastructure / Corruption; 13 helper funcs (`New*`, `Wrap*`, `Classify`, `IsRetryable`); 16 sentinel errors | ✅ |

**Coverage:** 89.1%

### Decider (Pure-Function Aggregate) ✅ FULLY_FUNCTIONAL

> `import "github.com/larsartmann/go-cqrs-lite/decider"`

|| Feature | Detail | Status |
|| ---------------------- | ----------------------------------------------------------------------------------------------- | ------ |
|| Decider[State] | `{Initial State; Fold func(State, Event) (State, error)}` — pure-function aggregate pattern | ✅ |
|| Repository[State] | `NewRepository[State](store, publisher, decider, opts...)` — manages aggregate lifecycle | ✅ |
|| Execute | `Repository.Execute(ctx, aggID, aggType, decide)` — load → decide → save → publish | ✅ |
|| Load | `Repository.Load(ctx, aggID, aggType)` — returns `(State, Version, error)` | ✅ |

|| LoadAtVersion | `Repository.LoadAtVersion(ctx, aggID, aggType, maxVersion)` — time-travel to version | ✅ |
|| LoadAtTime | `Repository.LoadAtTime(ctx, aggID, aggType, maxTime)` — time-travel to timestamp | ✅ |
|| Crash recovery | `SeekableJournal.ReadFrom` + `CheckpointStore` — tail events from checkpoint, republish on restart | ✅ |
|| Context enrichment | `WithEnricher` — injects metadata from context into events | ✅ |

**Sentinel errors:** `ErrNilStore`, `ErrNilBus`, `ErrNilFold`, `ErrLoadFailed`, `ErrFoldFailed`, `ErrSaveFailed`

### Aggregate Root (Traditional) ✅ FULLY_FUNCTIONAL

> `import "github.com/larsartmann/go-cqrs-lite/aggregate"`

|| Feature | Detail | Status |
|| ---------------------- | ----------------------------------------------------------------------------------------------- | ------ |
|| Aggregate root base | `Core` provides `ID`, `Type`, `Version`, `RecordEvent`, `UncommittedChanges`, `LoadFromHistory` | ✅ |
|| Event sourcing | `EventSourcedRepository` — Save (persist + publish) and Load (replay from store) | ✅ |
|| Optimistic concurrency | `Save` passes `expectedVersion` to `Store.Save` | ✅ |
|| Snapshot strategy | `event.SnapshotStrategy` interface + `EveryNEvents(n)` — shared with decider | ✅ |
|| Snapshot codec | `WithCodec` option for custom snapshot serialization | ✅ |
|| Crash recovery | `SeekableJournal.ReadFrom` + `CheckpointStore` — tail events from checkpoint, republish on restart
|| ISP Publisher | Repository accepts `event.Publisher` (not full `Bus`) — backward-compatible | ✅ |
|| Defensive copies | `UncommittedChanges()` returns a copy; `MarkChangesAsCommitted()` reuses backing array | ✅ |

> **Note:** The `aggregate` package is formally deprecated in favor of `decider`. It remains fully functional.

**Coverage:** 95.9%

### Branded IDs ✅ FULLY_FUNCTIONAL

> `import "github.com/larsartmann/go-cqrs-lite/id"`

|| Feature | Detail | Status |
|| -------------------- | ------------------------------------------------------------------------------------------- | ------ |
|| Generic branded type | `id.Of[T]` — phantom type parameter for compile-time safety | ✅ |
|| ULID-backed | Binary-sortable, time-ordered, 16-byte binary form | ✅ |
|| 8 built-in types | `AggregateID`, `EventID`, `CorrelationID`, `CausationID`, `RequestID`, `UserID`, `ClientID`, `CommandID` | ✅ |
|| Custom branded types | `type OrderID = id.Of[OrderMarker]` — users can create their own | ✅ |
|| All serialization | JSON (incl. `null`), binary, text, SQL `Scan`/`Value` | ✅ |
|| Convenience funcs | `New[T]()`, `Parse[T]()`, `MustParse[T]()`, `Ptr()`, `FromPtr()` | ✅ |
|| Comparison | `Equal`, `Compare` (lexicographic), `IsZero`, `Or` (default value) | ✅ |
|| fmt.Formatter | `%s`, `%v`, `%#v`, `%q` | ✅ |
|| Timestamp extraction | `ULID(id)` extracts embedded timestamp | ✅ |

**Coverage:** 97.8%

### Generic Dispatcher ✅ FULLY_FUNCTIONAL

> `import "github.com/larsartmann/go-cqrs-lite/dispatcher"`

|| Feature | Detail | Status |
|| -------------------- | --------------------------------------------------------------------------------------------- | ------ |
|| Generic Dispatcher | `Dispatcher[H, M]` — type-safe handler + middleware dispatcher | ✅ |
|| LifecycleMixin | `Lifecycle` — `Close()` prevents all ops; thread-safe | ✅ |
|| CatalogDispatcher | `CatalogDispatcher[KT, VT]` — embeddable catalog introspection for dispatchers | ✅ |
|| Middleware ordering | Reverse-order middleware application at registration time | ✅ |

**Coverage:** 100.0%

---

## In-Memory Implementations 🧪 TESTING_ONLY

> `import "github.com/larsartmann/go-cqrs-lite/memory"`

|| Component | Detail | Status |
|| --------------------- | ---------------------------------------------------------------------------------------------------------- | ------ |
|| MemoryStore | `event.Store` + `Journal` + `SeekableJournal` + `BackwardsSource` + `StreamLoader` with defensive copies | 🧪 |
|| MemoryBus | `event.Bus` with typed `Subscribe` + `SubscribeAll` + handler/publish middleware | 🧪 |
|| MemorySnapshotStore | `event.SnapshotStore` with deep-copy snapshots, version-aware `LoadAtVersion` | 🧪 |
|| MemoryCheckpointStore | `event.CheckpointStore` for projection checkpointing | 🧪 |

**Intended use:** Testing and development only. All implementations are thread-safe (`sync.RWMutex`), support `Close()` lifecycle, and return defensive copies.

**Coverage:** 99.6%

---

## Middleware Suite ✅ FULLY_FUNCTIONAL

> `import "github.com/larsartmann/go-cqrs-lite/middleware"`

All **8 concerns** are provided for all 3 message types (command, event, query) — **24 middleware factories** total.

### Logging ✅

|| Factory | Logs |
|| ------------------------ | ------------------------------------------------------------------------------------- |
|| `CommandLogging(logger)` | `"command dispatching"` / `"succeeded"` / `"failed"` with type, aggregateID, duration |
|| `EventLogging(logger)` | Same pattern for events |
|| `QueryLogging(logger)` | Same pattern for queries |

Accepts `*slog.Logger`.

### Metrics ✅

|| Factory | Records |
|| -------------------------- | -------------------------------------------------------------------- |
|| `CommandMetrics(recorder)` | `"command_success"` / `"command_error"` with duration and type label |
|| `EventMetrics(recorder)` | Same for events |
|| `QueryMetrics(recorder)` | Same for queries |

Accepts any `MetricsRecorder` interface (`Observe`).

### Recovery ✅

|| Factory | Behavior |
|| ------------------- | ------------------------------------------------ |
|| `CommandRecovery()` | Recovers panics → returns error with stack trace |
|| `EventRecovery()` | Same |
|| `QueryRecovery()` | Same (uses named returns for result + err) |

### Retry ✅

|| Factory | Behavior |
|| ---------------------- | ----------------------------------------------------------- |
|| `CommandRetry(config)` | Exponential backoff with jitter, context-aware cancellation |
|| `EventRetry(config)` | Same |
|| `QueryRetry(config)` | Same |

**RetryConfig:** `MaxAttempts`, `InitialDelay`, `MaxDelay`, `Multiplier`, `IsRetryable` predicate. Jitter uses `math/rand/v2`.

### Tracing ✅

|| Factory | Span |
|| ------------------------ | ------------------------------------------------------------------- |
|| `CommandTracing(tracer)` | `"command.handle"`, SpanKindServer, attributes: `cqrs.command.type` |
|| `EventTracing(tracer)` | `"event.handle"`, SpanKindConsumer, attributes: `cqrs.event.type` |
|| `QueryTracing(tracer)` | `"query.handle"`, SpanKindServer, attributes: `cqrs.query.type` |

OpenTelemetry via `go.opentelemetry.io/otel/trace`. Caller provides the `Tracer`.

### Validation ✅

|| Factory | Behavior |
|| ------------------------------ | --------------------------------------------------------- |
|| `CommandValidation(validator)` | Calls validator before handler; returns descriptive error |
|| `EventValidation(validator)` | Same |
|| `QueryValidation(validator)` | Same |

### Circuit Breaker ✅

|| Factory | Behavior |
|| ---------------------------------- | --------------------------------------------------------------------------------------------- |
|| `CommandCircuitBreaker(config)` | Three-state machine: Closed → Open (threshold) → Half-Open (timeout) → Closed (successes) |
|| `EventCircuitBreaker(config)` | Same |
|| `QueryCircuitBreaker(config)` | Same |

**CircuitBreakerConfig:** `FailureThreshold` (default 5), `SuccessThreshold` (default 3), `Timeout` (default 30s), `IsFailure` predicate. Rejected requests return `ErrCircuitBreakerOpen` wrapped as transient.

**Coverage:** 100.0%

---

## Event Signing ✅ FULLY_FUNCTIONAL

> `import "github.com/larsartmann/go-cqrs-lite/signing"`

### Single-Signature Mode

|| Feature | Detail | Status |
|| ------------------- | ------------------------------------------------------------------------------------ | ------ |
|| HMAC-SHA256 signer | `NewHMAC(key)` — `SignerVerifier` (sign + verify with same key) | ✅ |
|| Ed25519 signer | `NewEd25519(privateKey)` — `Signer` (sign-only, separate verifier) | ✅ |
|| Ed25519 verifier | `NewEd25519Verifier(publicKey)` — `Verifier` (verify-only) | ✅ |
|| Key pair generation | `GenerateEd25519KeyPair()` — convenience function | ✅ |
|| Canonical encoding | Deterministic length-prefixed format v1 (id, type, aggID, version, SHA-256 of payload) | ✅ |
|| Event helpers | `AttachSignature`, `ExtractSignature`, `HasSignature` — stored in event metadata | ✅ |
|| SignMiddleware | `event.PublishMiddleware` — auto-signs every published event | ✅ |
|| VerifyMiddleware | `event.Middleware` — verifies signatures if present; allows unsigned through | ✅ |
|| RequireSignature | `event.Middleware` — rejects unsigned events; verifies present signatures | ✅ |

### Multi-Signature Mode (`signing/multisig` sub-package)

|| Feature | Detail | Status |
|| ------------------- | ----------------------------------------------------------------------------------------------- | ------ |
|| MultiSigner | Actor-based multi-party signing with heterogeneous algorithms (e.g., device Ed25519 + server HMAC) | ✅ |
|| SignatureEntry | Per-actor entry with `Actor`, `Algorithm`, `Sig`, `SignedAt`; re-signing replaces prior entry | ✅ |
|| MultiSignature | Collection of entries with `Count()`, `HasActor()`, `Get()`, `Actors()` | ✅ |
|| MultiSignMiddleware | Appends actor's signature to multi-sig collection | ✅ |
|| MultiVerifyMiddleware | Verifies specific actor's multi-sig; allows missing through | ✅ |
|| RequireMultiSig | Rejects events missing signatures from ALL required actors; verifies all | ✅ |
|| VerifyAll | Bulk verification using actor→verifier map | ✅ |
|| VerifierMap | Convenience builder from `MultiSigner` slice to `map[Actor]Verifier` | ✅ |

**Key material always defensively copied. No external crypto dependencies beyond Go stdlib.**
**Coverage:** ~95%

---

## Auto-Documentation

### Catalog System ✅ FULLY_FUNCTIONAL

> `import "github.com/larsartmann/go-cqrs-lite/catalog"`

|| Feature | Detail | Status |
|| ------------------------ | --------------------------------------------------------------------------------------------------------------- | ------ |
|| Registry | Thread-safe builder: `AddService`, `AddCommand`, `AddEvent`, `AddQuery`, `AddDomain`, `AddChannel`, `AddFlow`, `AddDataStore`, `AddTeam`, `AddUser`, `Build()` | ✅ |
|| Schema reflection | `SchemaFromType[T]()` — auto-generates JSON Schema from Go structs via `reflect` | ✅ |
|| Struct tag support | `json` (name, omitempty), `doc`/`description`, `format`, `default`, `enum`, `nullable`, `deprecated`, `pattern` | ✅ |
|| Immutable catalog | `Build()` returns deep-copied, immutable `*Catalog` | ✅ |
|| Validation | `Catalog.Validate()` returns `[]Violation` — checks titles, duplicates, references | ✅ |
|| Exporter interface | `Exporter[T]` + `ErrorExporter` — pluggable output format | ✅ |
|| WalkMessages | `WalkMessages(cat, fn)` — iterate all messages across all services | ✅ |
|| Rich resource model | Services, Domains, Channels, DataStores, Flows, Teams, Users — with badges, owners, repositories, specifications | ✅ |
|| Dispatcher introspection | `FromCommandDispatcher`, `FromQueryDispatcher` — auto-import entries from live dispatchers | ✅ |

**Coverage:** ~94%

> `import "github.com/larsartmann/go-cqrs-lite/catalog/asyncapi"`

|| Feature | Detail | Status |
|| ------------------- | ----------------------------------------------------------------------------------------------------------- | ------ |
|| Document generation | `Exporter.Export(catalog)` produces full AsyncAPI 3.0 `Document` | ✅ |
|| YAML output | `Document.MarshalYAML()` — uses `go-faster/yaml` | ✅ |
|| JSON output | `Document.MarshalJSON()` — type-alias trick to avoid recursion | ✅ |
|| Server config | `WithServer(name, host, protocol)` option (defaults: kafka, localhost:9092) | ✅ |
|| Channel mapping | Commands → `receive`, Events with `Sends` → `send`, Events with `Receives` → `receive`, Queries → `receive` | ✅ |
|| Examples | `toExamples()` converts `json.RawMessage` to AsyncAPI examples | ✅ |

**Coverage:** 93.7%

> `import "github.com/larsartmann/go-cqrs-lite/catalog/eventcatalog"`

|| Feature | Detail | Status |
|| -------------- | ----------------------------------------------------------------------------- | ------ |
|| MDX generation | Services, commands, events, queries — all with YAML frontmatter | ✅ |
|| Schema files | `schema.json` per message (only when schema is non-nil) | ✅ |
|| Domain pages | Domain frontmatter with service associations | ✅ |
|| Config files | `eventcatalog.config.js`, `package.json` with `@eventcatalog/core` dependency | ✅ |
|| LLM summary | `llms.txt` — plain-text catalog summary for LLM consumption | ✅ |

**Coverage:** 91.3%

> `import "github.com/larsartmann/go-cqrs-lite/catalog/d2"`

|| Feature | Detail | Status |
|| ------------------- | ----------------------------------------------------------- | ------ |
|| D2 text export | `Exporter.Export(cat)` produces D2 diagram syntax | ✅ |
|| Service nodes | Color-coded rectangles per service with command/event/query | ✅ |
|| Cross-service flows | Animated arrows between publishers and receivers | ✅ |
|| Domain grouping | Domain labels with dashed "contains" links to services | ✅ |
|| Schema tooltips | Field names and types shown on hover | ✅ |
|| Options | `WithDescription`, `WithDirection` for layout customization | ✅ |

**Coverage:** 95.0%

> `import "github.com/larsartmann/go-cqrs-lite/catalog/openapi"`

|| Feature | Detail | Status |
|| ------------------- | ---------------------------------------------------------------- | ------ |
|| Document generation | `Exporter.Export(catalog)` produces full OpenAPI 3.0.3 `Document` | ✅ |
|| JSON output | `Document` serializes to JSON | ✅ |
|| Schema generation | Auto-generates JSON Schema from catalog types | ✅ |
|| Base path support | `WithBasePath(path)` option for API path prefix | ✅ |
|| Description option | `WithDescription(desc)` for document metadata | ✅ |

**Coverage:** 94.4%

> `import "github.com/larsartmann/go-cqrs-lite/catalog/docserver"`

|| Feature | Detail | Status |
|| ------------------- | -------------------------------------------------------------------- | ------ |
|| HTTP handlers | Framework-agnostic `net/http` handlers for serving docs | ✅ |
|| OpenAPI rendering | Scalar UI for interactive API documentation | ✅ |
|| AsyncAPI rendering | AsyncAPI React for event documentation | ✅ |
|| Raw spec serving | JSON/YAML endpoints for both OpenAPI and AsyncAPI | ✅ |
|| Catalog provider | `CatalogProvider` func — generates fresh catalog on each request | ✅ |
|| Embedded assets | HTML/JS/CSS embedded via `embed.FS` — zero external file dependencies | ✅ |

**Coverage:** 91.0%

---

## SQL & Key-Value Event Store ✅ FULLY_FUNCTIONAL

> `import "github.com/larsartmann/go-cqrs-lite/storage"`

### SQL Stores (PostgreSQL / SQLite / Turso)

|| Feature | Detail | Status |
|| ------------------------------- | --------------------------------------------------------------------------------- | ------ |
|| PostgreSQL event store | `NewSQLEventStore(db)` implements `event.Store` | ✅ |
|| SQLite event store | `NewSQLiteEventStore(db)` — `?` placeholders, `BLOB`/`TEXT` DDL | ✅ |
|| Turso convenience constructors | `NewTursoEventStore`, `NewTursoSnapshotStore`, `NewTursoOutbox`, `NewTursoCheckpointStore`, `NewTursoBackend`, `NewTursoTransactionalStore` | ✅ |
|| Schema DDL | `Schema()` PostgreSQL, `SQLiteSchema()` for SQLite/Turso | ✅ |
|| Per-table DDL | `SnapshotSchema`, `CheckpointSchema` + SQLite variants | ✅ |
|| Optimistic concurrency | `Save` checks version in transaction | ✅ |
|| AppendBatch | Appends without concurrency check | ✅ |
|| Full load API | `Load`, `LoadFromVersion`, `LoadToVersion`, `LoadToTimestamp`, `Delete` | ✅ |
|| LoadBackwards | Implements `event.BackwardsSource` — newest-first | ✅ |
|| Time-travel SQL queries | `LoadToVersion`, `LoadToTimestamp` with composite timestamp index | ✅ |
|| Journal / SeekableJournal | `ReadAll()`, `ReadFrom(afterEventID, limit)` | ✅ |
|| Stream loading | `LoadStream()` returns cursor-based `sqlEventStream` — memory-efficient iteration | ✅ |
|| Metadata persistence | Full roundtrip: correlation IDs, user IDs, custom metadata | ✅ |
|| SQL SnapshotStore | PostgreSQL + SQLite variants, upsert, version-aware load, delete | ✅ |
|| SQL CheckpointStore | PostgreSQL + SQLite variants, upsert, `sql.ErrNoRows` handling | ✅ |
|| TursoSyncDB | `OpenTursoSync` returns `*TursoSyncDB` with `Push`, `Pull`, `Checkpoint`, `Stats`, `Close` | ✅ |
|| DB helpers | `OpenSQLite`, `OpenSQLiteInMemory`, `SQLiteInitSchema`, `SQLiteEnableWAL`, `ConfigureSQLitePool`, `ConfigureTursoPool`, `PostgresInitSchema` | ✅ |
|| Dialect abstraction | `Dialect` interface with `Placeholder`, `FormatTime`, `ScanTimeDest`, `ParseTime`, 5 schema methods | ✅ |
|| SQLEventStore options | `SQLEventStoreOption` with `WithOwnership()` | ✅ |
|| Close lifecycle | No-op `Close()` — does not close `*sql.DB`; caller owns DB lifecycle | ✅ |

### Pebble Key-Value Store

|| Feature | Detail | Status |
|| -------------------------- | -------------------------------------------------------------------------------- | ------ |
|| PebbleEventStore | `NewPebbleStore(db, logger)` implements `event.Store` using CockroachDB Pebble | ✅ |
|| Config-based construction | `NewPebbleConfig` + `NewPebbleEventStore(cfg, logger)` with `WithPebbleBackend`, `WithPebbleProvider` | ✅ |
|| Full load API | `Load`, `LoadFromVersion`, `LoadToVersion`, `LoadToTimestamp`, `Delete` | ✅ |
|| In-memory backend | `PebbleBackendMemory` — for testing without disk | ✅ |

**Remaining gaps:**

|| Issue | Severity | Detail |
|| ------------------------------- | --------- | ------------------------------------------------------------------ |
|| No Journal/SeekableJournal | ⚠️ MEDIUM | Only implements event.Store, not Journal or SeekableJournal (unlike Memory/SQL) |
| No PostgreSQL integration tests | ⚠️ MEDIUM | Unit tests use go-sqlmock only; no real PostgreSQL verification |
| Backend field unused at runtime | ⚠️ LOW | Backend type/constants have no runtime effect |

**Coverage:** 89.6%

---

## Projection Runner ✅ FULLY_FUNCTIONAL

> `import "github.com/larsartmann/go-cqrs-lite/projection"`

|| Feature | Detail | Status |
|| ------------------------- | ---------------------------------------------------------------------- | ------ |
|| Runner | `NewRunner(journal, subscriber, checkpoint, opts...)` — replay → live | ✅ |
|| Builder + On[T]() | `NewBuilder(name)` + `On[T](builder, eventType, handler)` — type-safe | ✅ |
|| HandlerRegistry | Thread-safe `On(eventType, handler)`, `OnAll(handler)`, `Lookup` | ✅ |
|| Checkpoint per projection | `CurrentCheckpoint(ctx, name)` — read last-processed event ID | ✅ |
|| Reset/rebuild | `Runner.Reset(ctx, name)` — zero-out checkpoint → full replay on next Run | ✅ |
|| Event type filtering | Runner filters events by `Projection.EventTypes()` | ✅ |
|| SeekableJournal detection | Auto-detects `SeekableJournal` for position-based replay | ✅ |
|| Live-only mode | Pass `nil` journal to skip replay entirely | ✅ |
|| Retry with backoff | `WithRetry(count, delay)` — exponential backoff, only if `IsRetryable` | ✅ |
|| Dead letter queue | `WithDeadLetterHandler(func)` — callback after retries exhausted | ✅ |
|| Replay context marking | `event.WithReplay(ctx, true)` during replay; handlers can distinguish | ✅ |
|| Close lifecycle | `Runner.Close()` — cancel internal context, graceful shutdown | ✅ |
|| Duplicate name guard | `Register()` rejects projections with same `Name()` | ✅ |

**Sentinel errors:** `ErrNilHandler`, `ErrNilBus`, `ErrNilCheckpoint`, `ErrNoProjections`, `ErrDuplicateProjection`

**Coverage:** ~95%

---

## Test Helpers 🧪 TESTING_ONLY

> `import "github.com/larsartmann/go-cqrs-lite/testhelpers"`

|| Helper | Purpose |
|| -------------------------------------------------------------- | ---------------------------------------------------- |
|| `FakeStore` | Configurable `event.Store` with overridable `SaveFn` |
|| `FakeBus` | Records published events, injectable `PublishErr` |
|| `FakeSnapshotStore` | Configurable snapshot load/save with error injection |
|| `AppendEventsHandler` | Handler that collects events into a slice |
|| `NoopCommandHandler` / `NoopEventHandler` / `NoopQueryHandler` | No-op handlers |
|| `FailingCommandHandler` / `FailingEventHandler` | Handlers that always error |
|| `PanicCommandHandler` / `PanicEventHandler` | Handlers that panic |
|| `CallbackCommandHandler` / `CallbackEventHandler` | Sets a bool flag |
|| `CommandMiddleware` / `EventMiddleware` | Call-order tracking middleware |
|| `TestMetrics` | `MetricsRecorder` that records names and durations |
|| `NewTestEvent` | Creates a minimal test event |

---

## Example 💡 DEMO

> `example/user/` — `go run ./example/user`

Minimal CLI demo showing the event sourcing lifecycle:

1. Create `MemoryStore` + `MemoryBus` + `Repository`
2. Create User aggregate, record event, save
3. Load fresh User instance, replay, mutate, save again

**Not a reference application.** Uses raw byte payloads (not JSON), no command/query dispatchers, no HTTP layer, `ApplySnapshot` is a no-op.

---

## Saga Pattern — Removed

> Saga-style orchestration is demonstrated via `example/saga-pattern/`.
> No dedicated module is needed — sagas emerge from projection + command dispatch.

---

## Watermill Adapter ✅ FULLY_FUNCTIONAL

> `import "github.com/larsartmann/go-cqrs-lite/watermill"`

|| Feature | Detail | Status |
|| ------------------- | ------------------------------------------------------------------------------------------- | ------ |
|| Metadata protocol | Bidirectional `event.Event` ↔ Watermill `message.Message` via 15+ metadata keys | ✅ |
|| PublisherAdapter | `NewPublisherAdapter(publisher)` — wraps `event.Publisher` as `message.Publisher` | ✅ |
|| SubscriberAdapter | `NewSubscriberAdapter(bus)` — wraps `event.Bus` as `message.Subscriber`, feeds `<-chan *message.Message` | ✅ |
|| Full event fidelity | 15 metadata keys preserve ID, type, aggregate, version, schema version, all metadata fields | ✅ |
|| Custom metadata | `custom.*` prefix preserves all custom metadata entries | ✅ |

**Coverage:** 89.6%

---

## Stream Read Model ✅ FULLY_FUNCTIONAL

> `import "github.com/larsartmann/go-cqrs-lite/listing"`

||| Feature | Detail | Status |
||| --------------------- | --------------------------------------------------------------------------------------------------------------------------- | ------ |
||| AggregateReader | `List(ctx, ListOptions) → Page[AggregateStatus]` — cursor-based aggregate listing | ✅ |
||| ListBuilder | Fluent API: `listing.NewListBuilder(reader).OfType("User").After(cursor).Limit(50).IncludeDeleted()` | ✅ |
||| InMemoryAggregateReader | Reads from `event.Journal.ReadAll()` — single-pass, no persistence | ✅ |
||| SQLAggregateReader | Reads from projection tables with prefix validation (`^[a-z_][a-z0-9_]*$`) | ✅ |
||| AggregateProjection | Maintains SQL read-model tables from event streams with tombstone detection | ✅ |
||| StatusMiddleware | Event bus middleware that publishes aggregate status changes | ✅ |
||| TombstonePolicy | `Exclude` (default), `Include`, `Only` — controls visibility of soft-deleted aggregates | ✅ |
||| Page[T] | Cursor-based pagination with `HasMore` — no expensive TotalCount | ✅ |
||| AggregateRef | Lightweight identity: ID, Type, Version, EventCount, LastEventAt | ✅ |
||| AggregateStatus | Pairs AggregateRef with computed TombstoneStatus | ✅ |

---

## cqrs-gen Code Generator 💡 TOOL

> `go run github.com/larsartmann/go-cqrs-lite/cmd/cqrs-gen`

|| Feature | Detail | Status |
|| ------------------- | ---------------------------------------------------------------------------- | ------ |
|| AST-based scanning | Parses Go source for `//cqrs:command <Name>` / `//cqrs:query <Name>` markers | ✅ |
|| Typed handler gen | Generates `Register<StructName>Handler` functions using `RegisterTyped[T]` | ✅ |
|| CLI flags | `-type` (command/query), `-output` (file), `-pkg` (package name) | ✅ |
|| Recursive directory | Walks directories, skips `_test.go`, extracts markers from doc comments | ✅ |

**Coverage:** 70.8% (CLI main entry point not tested; all library functions covered)

---

## Known Code Quality Issues

Found during 2026-06-01 full code review. See `docs/planning/2026-06-01_CODE-QUALITY-FULL-REVIEW.md` for details.

| Issue                                                      | Severity | Module                   |
| ---------------------------------------------------------- | -------- | ------------------------ |
| Middleware 3x duplication (~500 lines)                     | HIGH     | middleware               |
| Three separate ErrHandlerNotFound sentinels                | HIGH     | dispatcher/command/query |
| VersionedStore exposes embedded Store (bypass upcasting)   | HIGH     | schema                   |
| command.Metadata duplicates event.Metadata (split brain)   | HIGH     | command                  |
| command re-exports event types (module boundary violation) | HIGH     | command                  |
| decider/load.go uses unclassified fmt.Errorf errors        | MEDIUM   | decider                  |
| storage/pebble error sentinels duplicated                  | MEDIUM   | storage, pebble          |
| catalog/ToAny silently swallows marshal errors             | MEDIUM   | catalog                  |
| watermill silently drops malformed ID parse errors         | MEDIUM   | watermill                |
| Reactive extensions not wired into dispatchers             | LOW      | event/command/query      |

---

## Not Yet Implemented 📐 PLANNED

Features mentioned in project docs/planning but with **no production code yet**:

| Feature                    | Description                                  | Sprint |
| -------------------------- | -------------------------------------------- | ------ |
| Outbox pattern             | Reliable at-least-once event publishing      | Future |
| Schema registry            | JSON Schema middleware for event validation  | Future |
| Playwright E2E tests       | End-to-end browser tests for `example/user/` | 5      |
| Extended snapshot coverage | `go-snaps` across remaining 11 library modules | 6    |

---

## Module Maturity Matrix

| Module                 | Import Path                      | Coverage | Maturity        |
| ---------------------- | -------------------------------- | -------- | --------------- |
| `event`                | `…/event/v2`                     | 89.4%    | ✅ Production   |
| `event/eventtest`      | `…/event/v2/eventtest`           | 18.4%    | 🧪 Test helper  |
| `command`              | `…/command/v2`                   | 80.5%    | ✅ Production   |
| `query`                | `…/query/v2`                     | 94.3%    | ✅ Production   |
| `decider`              | `…/decider/v2`                   | 100.0%   | ✅ Production   |
| `id`                   | `…/id/v2`                        | 96.4%    | ✅ Production   |
| `dispatcher`           | `…/dispatcher/v2`                | 100.0%   | ✅ Production   |
| `schema`               | `…/schema/v2`                    | 89.7%    | ✅ Production   |
| `snapshot`             | `…/snapshot/v2`                  | 92.3%    | ✅ Production   |
| `codec`                | `…/codec/v2`                     | 93.3%    | ✅ Production   |
| `memory`               | `…/memory/v2`                    | 98.2%    | 🧪 Test utility |
| `catalog`              | `…/catalog/v2`                   | 95.9%    | ✅ Production   |
| `catalog/asyncapi`     | `…/catalog/v2/asyncapi`          | 93.9%    | ✅ Production   |
| `catalog/d2`           | `…/catalog/v2/d2`                | 95.0%    | ✅ Production   |
| `catalog/openapi`      | `…/catalog/v2/openapi`           | 100.0%   | ✅ Production   |
| `catalog/eventcatalog` | `…/catalog/v2/eventcatalog`      | 92.7%    | ✅ Production   |
| `catalog/docserver`    | `…/catalog/v2/docserver`         | 90.1%    | ✅ Production   |
| `catalog/schema`       | `…/catalog/v2/schema`            | 86.0%    | ✅ Production   |
| `catalog/caseutil`     | `…/catalog/v2/internal/caseutil` | 100.0%   | ✅ Production   |
| `middleware`           | `…/middleware/v2`                | 93.5%    | ✅ Production   |
| `integration`          | `…/integration/v2`               | N/A      | ✅ Test suite   |
| `projection`           | `…/projection/v2`                | 91.2%    | ✅ Production   |
| `signing`              | `…/signing/v2`                   | 94.1%    | ✅ Production   |
| `signing/multisig`     | `…/signing/v2/multisig`          | 94.1%    | ✅ Production   |
| `storage`              | `…/storage/v2`                   | 86.8%    | ✅ Production   |
| `storage/sql`          | `…/storage/v2/sql`               | 34.7%    | 🧪 Shared infra |
| `watermill`            | `…/watermill/v2`                 | 94.3%    | ✅ Production   |
| `listing`              | `…/listing/v2`                   | 94.9%    | ✅ Production   |
| `otel`                 | `…/otel/v2`                      | 96.4%    | ✅ Production   |
| `pebble`               | `…/pebble/v2`                    | 86.7%    | ✅ Production   |
| `turso`                | `…/turso/v2`                     | 28.6%    | ✅ Production   |
| `cmd/cqrs-gen`         | `…/cmd/cqrs-gen/v2`              | 89.9%    | 🔧 Tool         |
| `cmd/api-stability`    | `…/cmd/api-stability/v2`         | N/A      | 🔧 Tool         |
| `example/user`         | `…/example/user`                 | N/A      | 💡 Demo         |

---

---

## Architecture Guarantees

|| Guarantee | Detail |
|| ---------------------- | -------------------------------------------------------------------------------- |
|| Zero lint issues | Clean golangci-lint across all modules |
|| Race-free | `go test -race` passes across all modules |
|| Multi-module isolation | Each module has independent `go.mod`, no circular dependencies |
|| Interface-first | All core types are interfaces — provide your own implementations |
|| Library, not framework | Import what you need, compose your own stack |
|| Context-aware | All handlers accept `context.Context` |
|| Errors as values | No panics in production code, explicit error returns, sentinel errors + wrapping |

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
