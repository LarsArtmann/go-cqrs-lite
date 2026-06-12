# Features

> Honest, verified inventory of what go-cqrs-lite actually does — not what it plans to do.

**Last audited:** 2026-06-12 (v2.3.0 release) · **Module count:** 28 (22 library + 2 examples + 1 integration + 2 cmd + turso/indexing sub-package) · **Go version:** 1.26.3

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

> `import "github.com/larsartmann/go-cqrs-lite/command"`

| Feature                  | Detail                                                                                           | Status |
| ------------------------ | ------------------------------------------------------------------------------------------------ | ------ |
| Command dispatch         | `Dispatcher.Dispatch(ctx, cmd)` routes to registered handler                                     | ✅     |
| Handler registration     | `Dispatcher.Register(cmdType, handler)` with duplicate guard                                     | ✅     |
| Middleware chain         | `Dispatcher.Use(middleware...)` — applied at registration time, reverse order                    | ✅     |
| Lifecycle                | `Dispatcher.Close()` — rejects all ops after close                                               | ✅     |
| Validation               | `New()` rejects empty type and zero aggregateID                                                  | ✅     |
| TypedHandler[T]          | `RegisterTyped[T](d, type, handler)` — type-safe handler receiving `T` not `Command`             | ✅     |
| Command metadata         | `Metadata` struct (alias of `event.Metadata`) with CorrelationID, CausationID, UserID, RequestID | ✅     |
| Metadata options         | `WithCorrelationID`, `WithCausationID`, `WithUserID`, `WithRequestID`                            | ✅     |
| Persisted command        | `PersistedCommand` struct with ID, Type, AggregateRef, ReceivedAt, Payload, Metadata             | ✅     |
| Command store interfaces | `CommandSink`, `CommandSource`, `Store` (Sink+Source) — persisted command log                    | ✅     |

**Sentinel errors:** `ErrHandlerNotFound`, `ErrDispatcherClosed`, `ErrEmptyCommandType`, `ErrNilAggregateID`, `ErrTypeAssertion`, `ErrEmptyAggregateType`, `ErrDuplicateCommand`, `ErrCommandNotFound`, `ErrStoreClosed`

### Query Dispatcher ✅ FULLY_FUNCTIONAL

> `import "github.com/larsartmann/go-cqrs-lite/query"`

| Feature              | Detail                                                                             | Status |
| -------------------- | ---------------------------------------------------------------------------------- | ------ |
| Query dispatch       | `Dispatcher.Dispatch(ctx, query)` returns `(any, error)`                           | ✅     |
| Typed dispatch       | `DispatchTyped[T](ctx, dispatcher, query)` — generic type-safe result extraction   | ✅     |
| Handler registration | Same pattern as command — duplicate guard, lifecycle                               | ✅     |
| Middleware chain     | Same pattern as command                                                            | ✅     |
| Pagination           | `Pagination` struct with `Page`, `PageSize`, `Offset()`, `Validate()`              | ✅     |
| Paginated results    | `PaginatedResult[T]` with `HasNext()`, `HasPrev()`, computed `TotalPages`          | ✅     |
| TypedHandler[Q, R]   | `RegisterTyped[Q, R]` — type-safe handler receiving `Q` and returning `(R, error)` | ✅     |

**Defaults:** Page 1, PageSize 20, max 100.
**Sentinel errors:** `ErrHandlerNotFound`, `ErrDispatcherClosed`, `ErrEmptyQueryType`, `ErrTypeAssertion`

---

### Event System ✅ FULLY_FUNCTIONAL

> `import "github.com/larsartmann/go-cqrs-lite/event"`

| Feature               | Detail                                                                                                                                                                                                                                                                                                                     | Status |
| --------------------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ------ |
| Event creation        | `NewEvent()` with auto-generated `EventID` (ULID) and `time.Now()` timestamp                                                                                                                                                                                                                                               | ✅     |
| Auto-marshal creation | `New()` — creates event from `any` payload (auto-json for structs/maps)                                                                                                                                                                                                                                                    | ✅     |
| Batch creation        | `NewEvents()` — batch event creation with auto-incrementing versions                                                                                                                                                                                                                                                       | ✅     |
| 19 functional options | `WithEventID`, `WithOccurredAt`, `WithMetadata`, `WithCorrelationID`, `WithCausationID`, `WithUserID`, `WithRequestID`, `WithSource`, `WithIPAddress`, `WithUserAgent`, `WithCustom`, `WithSchemaVersion`, `WithEncoding`, `WithCodec`, `WithClock`, `WithClientID`, `WithClientOccurredAt`, `WithDeadline`, `FromContext` | ✅     |
| Metadata              | `Metadata` struct: `CorrelationID`, `CausationID`, `UserID`, `RequestID`, `Source`, `IPAddress`, `UserAgent`, `Custom`                                                                                                                                                                                                     | ✅     |
| Context enricher      | `ContextEnricher` extracts options from `context.Context`; `CompositeEnricher` chains multiple                                                                                                                                                                                                                             | ✅     |
| Defensive copies      | `Payload()` and `Metadata()` return copies — callers can't mutate internals                                                                                                                                                                                                                                                | ✅     |
| Event.Clone()         | Deep copy of `ImmutableEvent`                                                                                                                                                                                                                                                                                              | ✅     |
| Typed values          | `Source`, `IPAddress`, `UserAgent`, `Version`, `SchemaVersion` — all parsed and validated                                                                                                                                                                                                                                  | ✅     |
| Version arithmetic    | `Version.Add`, `Sub`, `Mod`, `Cmp`, `IsPositive` — phantom type math                                                                                                                                                                                                                                                       | ✅     |
| Event Bus interface   | `Bus` (with `io.Closer`): `Publish`, `Subscribe`, `SubscribeAll`, `Use`, `UsePublish`                                                                                                                                                                                                                                      | ✅     |
| PublishMiddleware     | `Bus.UsePublish(mw)` — middleware for publish path                                                                                                                                                                                                                                                                         | ✅     |
| PublisherFunc adapter | `PublisherFunc` — function adapter for `Publisher`                                                                                                                                                                                                                                                                         | ✅     |
| Event Store interface | `Store = EventSink + EventSource` (with `io.Closer`): `Save` (optimistic concurrency), `AppendBatch`, `Load`, `LoadFromVersion`, `LoadToVersion`, `LoadToTimestamp`                                                                                                                                                        | ✅     |
| ISP split             | `EventSink` (write) + `EventSource` (read) — fine-grained dependency injection                                                                                                                                                                                                                                             | ✅     |
| Journal               | `ReadAll()` returns all events ordered by `occurred_at ASC` — for projection replay                                                                                                                                                                                                                                        | ✅     |
| SeekableJournal       | `ReadFrom(ctx, afterEventID, limit)` — efficient projection catch-up                                                                                                                                                                                                                                                       | ✅     |
| BackwardsSource       | `LoadBackwards(ctx, aggRef)` — loads events in reverse version order                                                                                                                                                                                                                                                       | ✅     |
| TombstoneStatus       | `Active`, `Tombstoned`, `Undetermined` — tri-state enum for soft-delete; `DetectTombstone`, `MarkTombstone`, `MarkRebirth`                                                                                                                                                                                                 | ✅     |
| Time-travel queries   | `LoadToVersion` and `LoadToTimestamp` — read aggregate state at a point in time                                                                                                                                                                                                                                            | ✅     |
| Projection interface  | `Projection`: `Name`, `Handle(ctx, Event)`, `EventTypes()` (nil = all)                                                                                                                                                                                                                                                     | ✅     |
| Context replay marker | `WithReplay(ctx, true)` — marks context as replay; handlers can distinguish                                                                                                                                                                                                                                                | ✅     |
| DecodePayload[T]      | `DecodePayload[T](evt, codec)` — type-safe payload deserialization                                                                                                                                                                                                                                                         | ✅     |
| DecodePayloads[T]     | `DecodePayloads[T](events, codec)` — batch payload deserialization                                                                                                                                                                                                                                                         | ✅     |
| PayloadReadOnly       | Zero-copy read access for internal paths (signing, pebble, storage, middleware)                                                                                                                                                                                                                                            | ✅     |
| Stream loading        | `StreamLoader` interface + `EventStream` cursor + `StoreStreamAdapter`                                                                                                                                                                                                                                                     | ✅     |
| Reactive streams      | `EventBus = ro.Subject[Event]`, `NewEventBus`, `NewReplayEventBus`, `NewBehaviorEventBus`, `FilterEventType`, `FilterEventTypes`, `ReplayFilter`, `HandlerToObserver`                                                                                                                                                      | ✅     |
| Slice helpers         | `SliceFromVersion`, `SliceToVersion`, `FilterByTimestamp` — in-memory event slicing                                                                                                                                                                                                                                        | ✅     |
| Checkpoint            | `Checkpoint` struct + `CheckpointSink/Source/Store` interfaces for projection positioning                                                                                                                                                                                                                                  | ✅     |
| Clock injection       | `Clock` type + `WithClock` option for deterministic testing                                                                                                                                                                                                                                                                | ✅     |
| Error taxonomy        | 5-family: Rejection / Conflict / Transient / Infrastructure / Corruption; 12 helper funcs (`New*`, `Wrap*`, `Classify`, `IsRetryable`); 16 sentinel errors                                                                                                                                                                 | ✅     |
| Event reconstruction  | `ReconstructEventFromFields` — shared deserialization for all store implementations                                                                                                                                                                                                                                        | ✅     |
| JSON metadata         | `MarshalMetadataJSON`, `UnmarshalMetadataJSON` — DB-safe metadata serialization                                                                                                                                                                                                                                            | ✅     |

### Decider (Pure-Function Aggregate) ✅ FULLY_FUNCTIONAL

> `import "github.com/larsartmann/go-cqrs-lite/decider"`

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

> `import "github.com/larsartmann/go-cqrs-lite/id"`

| Feature                | Detail                                                                                                   | Status |
| ---------------------- | -------------------------------------------------------------------------------------------------------- | ------ |
| Generic branded type   | `id.Of[T]` — phantom type parameter for compile-time safety                                              | ✅     |
| ULID-backed            | Binary-sortable, time-ordered, 16-byte binary form                                                       | ✅     |
| 8 built-in types       | `AggregateID`, `EventID`, `CorrelationID`, `CausationID`, `RequestID`, `UserID`, `ClientID`, `CommandID` | ✅     |
| Custom branded types   | `type OrderID = id.Of[OrderMarker]` — users can create their own                                         | ✅     |
| All serialization      | JSON (incl. `null`), binary, text, SQL `Scan`/`Value`                                                    | ✅     |
| Convenience funcs      | `New[T]()`, `Parse[T]()`, `ULID[T]()`, `FromPtr[T]()`, `CompareIDs[T]()`                                 | ✅     |
| AggregateID derivation | `DeriveAggregateID()` — deterministic ID from namespace + key                                            | ✅     |
| Timestamp extraction   | `ULID(id)` extracts embedded timestamp                                                                   | ✅     |

### Generic Dispatcher ✅ FULLY_FUNCTIONAL

> `import "github.com/larsartmann/go-cqrs-lite/dispatcher"`

| Feature             | Detail                                                         | Status |
| ------------------- | -------------------------------------------------------------- | ------ |
| Generic Dispatcher  | `Dispatcher[H, M]` — type-safe handler + middleware dispatcher | ✅     |
| LifecycleMixin      | `Lifecycle` — `Close()` prevents all ops; thread-safe          | ✅     |
| Middleware ordering | Reverse-order middleware application at registration time      | ✅     |
| Duplicate guard     | `ErrHandlerAlreadyRegistered` — prevents double-registration   | ✅     |

---

## Schema Evolution ✅ FULLY_FUNCTIONAL

> `import "github.com/larsartmann/go-cqrs-lite/schema"`

| Feature              | Detail                                                                               | Status |
| -------------------- | ------------------------------------------------------------------------------------ | ------ |
| Upcaster             | `Upcaster` interface — transforms old schema versions to newer on load               | ✅     |
| Upcaster constructor | `NewUpcaster(eventType, fromVersion, fn)` — version-gated transform                  | ✅     |
| Cycle detection      | Registry detects schema version revisits during upcast chain                         | ✅     |
| VersionedStore       | `VersionedStore` wraps any `event.Store` — transparent upcasting on all read methods | ✅     |
| Full load API        | `Load`, `LoadFromVersion`, `LoadToVersion`, `LoadToTimestamp` — all with upcasting   | ✅     |

---

## Snapshot ✅ FULLY_FUNCTIONAL

> `import "github.com/larsartmann/go-cqrs-lite/snapshot"`

| Feature          | Detail                                                          | Status |
| ---------------- | --------------------------------------------------------------- | ------ |
| Snapshot type    | `Snapshot` struct with AggregateRef, Version, State, SavedAt    | ✅     |
| Store interfaces | `SnapshotSink`, `SnapshotSource`, `SnapshotStore` (Sink+Source) | ✅     |
| Strategy         | `SnapshotStrategy` interface + `EveryNEvents(n)` built-in       | ✅     |
| Helper functions | `ShouldSnapshot`, `SaveSnapshot` — decider integration helpers  | ✅     |

---

## Payload Codec ✅ FULLY_FUNCTIONAL

> `import "github.com/larsartmann/go-cqrs-lite/codec"`

| Feature            | Detail                                                          | Status |
| ------------------ | --------------------------------------------------------------- | ------ |
| Codec interface    | `Codec` — `Encoding()`, `Encode(v)`, `Decode(data, v)`          | ✅     |
| JSON codec         | `JSONCodec` — standard JSON encoding                            | ✅     |
| CBOR codec         | `CBORCodec` — deterministic canonical CBOR with sorted map keys | ✅     |
| Raw passthrough    | `RawCodec` — `[]byte` pass-through (no encoding)                | ✅     |
| Encoding constants | `EncodingJSON`, `EncodingCBOR`, `EncodingRaw`                   | ✅     |

---

## In-Memory Implementations 🧪 TESTING_ONLY

> `import "github.com/larsartmann/go-cqrs-lite/memory"`

| Component             | Detail                                                                                                   | Status |
| --------------------- | -------------------------------------------------------------------------------------------------------- | ------ |
| MemoryStore           | `event.Store` + `Journal` + `SeekableJournal` + `BackwardsSource` + `StreamLoader` with defensive copies | 🧪     |
| MemoryBus             | `event.Bus` with typed `Subscribe` + `SubscribeAll` + handler/publish middleware                         | 🧪     |
| MemorySnapshotStore   | `snapshot.SnapshotStore` with deep-copy snapshots, version-aware `LoadAtVersion`                         | 🧪     |
| MemoryCheckpointStore | `event.CheckpointStore` for projection checkpointing                                                     | 🧪     |
| MemoryCommandStore    | `command.Store` for persisted command log                                                                | 🧪     |

**Intended use:** Testing and development only. All implementations are thread-safe (`sync.RWMutex`), support `Close()` lifecycle, and return defensive copies.

---

## Middleware Suite ✅ FULLY_FUNCTIONAL

> `import "github.com/larsartmann/go-cqrs-lite/middleware"`

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

### HTTP Metrics ✅

| Feature           | Detail                                                     | Status |
| ----------------- | ---------------------------------------------------------- | ------ |
| MetricsCollector  | In-memory request metrics with `RecordRequest`, `Snapshot` | ✅     |
| MetricsHandler    | `net/http` handler exposing JSON metrics endpoint          | ✅     |
| MetricsMiddleware | HTTP middleware recording request count and latency        | ✅     |

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

### Trace Logging ✅

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

### Health Check ✅

| Feature             | Detail                                                              | Status |
| ------------------- | ------------------------------------------------------------------- | ------ |
| HealthChecker       | Interface for health-checkable components                           | ✅     |
| HealthCheckResponse | `Status` (Pass/Fail/Warn) + `Checks` map with per-component details | ✅     |
| HealthCheckHandler  | `net/http` handler returning JSON health status                     | ✅     |

### SSE Broker ✅

| Feature     | Detail                                                                             | Status |
| ----------- | ---------------------------------------------------------------------------------- | ------ |
| SSEBroker   | Server-Sent Events broker with `AddClient`, `RemoveClient`, `ClientCount`, `Close` | ✅     |
| SSEHandler  | `net/http` handler for SSE connections with client ID extraction                   | ✅     |
| Thread-safe | Concurrent client management with proper channel lifecycle                         | ✅     |

---

## Event Signing ✅ FULLY_FUNCTIONAL

> `import "github.com/larsartmann/go-cqrs-lite/signing"`

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

> `import "github.com/larsartmann/go-cqrs-lite/encryption"`

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

> `import "github.com/larsartmann/go-cqrs-lite/catalog"`

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

> `import "github.com/larsartmann/go-cqrs-lite/catalog/asyncapi"`

| Feature             | Detail                                                                                                      | Status |
| ------------------- | ----------------------------------------------------------------------------------------------------------- | ------ |
| Document generation | `Exporter.Export(catalog)` produces full AsyncAPI 3.0 `Document`                                            | ✅     |
| YAML output         | `Document.MarshalYAML()` — uses `go-faster/yaml`                                                            | ✅     |
| JSON output         | `Document.MarshalJSON()` — type-alias trick to avoid recursion                                              | ✅     |
| Server config       | `WithServer(name, host, protocol)` option (defaults: kafka, localhost:9092)                                 | ✅     |
| Channel mapping     | Commands → `receive`, Events with `Sends` → `send`, Events with `Receives` → `receive`, Queries → `receive` | ✅     |

> `import "github.com/larsartmann/go-cqrs-lite/catalog/eventcatalog"`

| Feature        | Detail                                                                        | Status |
| -------------- | ----------------------------------------------------------------------------- | ------ |
| MDX generation | Services, commands, events, queries — all with YAML frontmatter               | ✅     |
| Schema files   | `schema.json` per message (only when schema is non-nil)                       | ✅     |
| Domain pages   | Domain frontmatter with service associations                                  | ✅     |
| Config files   | `eventcatalog.config.js`, `package.json` with `@eventcatalog/core` dependency | ✅     |
| LLM summary    | `llms.txt` — plain-text catalog summary for LLM consumption                   | ✅     |

> `import "github.com/larsartmann/go-cqrs-lite/catalog/d2"`

| Feature             | Detail                                                      | Status |
| ------------------- | ----------------------------------------------------------- | ------ |
| D2 text export      | `Exporter.Export(cat)` produces D2 diagram syntax           | ✅     |
| Service nodes       | Color-coded rectangles per service with command/event/query | ✅     |
| Cross-service flows | Animated arrows between publishers and receivers            | ✅     |
| Domain grouping     | Domain labels with dashed "contains" links to services      | ✅     |
| Schema tooltips     | Field names and types shown on hover                        | ✅     |

> `import "github.com/larsartmann/go-cqrs-lite/catalog/openapi"`

| Feature             | Detail                                                            | Status |
| ------------------- | ----------------------------------------------------------------- | ------ |
| Document generation | `Exporter.Export(catalog)` produces full OpenAPI 3.0.3 `Document` | ✅     |
| JSON output         | `Document` serializes to JSON                                     | ✅     |
| Schema generation   | Auto-generates JSON Schema from catalog types                     | ✅     |
| Base path support   | `WithBasePath(path)` option for API path prefix                   | ✅     |
| Description option  | `WithDescription(desc)` for document metadata                     | ✅     |

> `import "github.com/larsartmann/go-cqrs-lite/catalog/docserver"`

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

> `import "github.com/larsartmann/go-cqrs-lite/storage"`

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

> `import "github.com/larsartmann/go-cqrs-lite/pebble"`

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

### Turso Database Connector

> `import "github.com/larsartmann/go-cqrs-lite/turso"`

| Feature                 | Detail                                                                                                       | Status |
| ----------------------- | ------------------------------------------------------------------------------------------------------------ | ------ |
| Local DB                | `Open(dbPath)`, `OpenInMemory()` — embedded LibSQL                                                           | ✅     |
| Schema init             | `InitSchema(ctx, db)` — delegates to `storage.SQLiteInitSchema`                                              | ✅     |
| Convenience stores      | `NewEventStore`, `NewSnapshotStore`, `NewCheckpointStore` — thin wrappers over storage/                      | ✅     |
| Remote sync             | `OpenSync(ctx, dbPath, remoteURL, authToken)` — `SyncDB` with `Push`, `Pull`, `Checkpoint`, `Stats`, `Close` | ✅     |
| Phantom types           | `DbPath`, `RemoteURL`, `AuthToken` — compile-time type safety                                                | ✅     |
| Backward-compat aliases | `OpenTurso`, `NewTursoEventStore`, etc. — deprecated aliases preserved                                       | ✅     |
| Indexed schema init     | `InitSchemaWithIndexes`, `InitSchemaWithIndexesAndOptimizations` — tables + indexes + pragmas                | ✅     |
| Index convenience       | `NewIndexAdvisor`, `NewAutoIndexer`, `ApplyCQRSIndexes`, `ApplyTursoOptimizations`                           | ✅     |

### Turso Indexing (sub-package) ✅ FULLY_FUNCTIONAL

> `import "github.com/larsartmann/go-cqrs-lite/turso/indexing"`

| Feature                  | Detail                                                                                          | Status |
| ------------------------ | ----------------------------------------------------------------------------------------------- | ------ |
| EXPLAIN-based Advisor    | `Advisor` analyzes queries via `EXPLAIN QUERY PLAN` — detects missing indexes, full table scans | ✅     |
| AutoIndexer              | `AutoIndexer` creates/drops indexes automatically — lifecycle with Enable/Disable/Apply         | ✅     |
| CQRS index presets       | `RecommendedCQRSIndexes()` — pre-built IndexSet for event store tables                          | ✅     |
| Per-table Policy         | `Policy` type — exclude tables, mark critical, skip auto-creation per table                     | ✅     |
| Priority classification  | `Priority` enum (Critical/Recommended/Optional) on Recommendations                              | ✅     |
| Dry-run mode             | `WithDryRun()` — prints DDL without executing                                                   | ✅     |
| OTel tracing             | All major operations traced via OpenTelemetry spans                                             | ✅     |
| Index usage stats        | `Stats(ctx, db)` — queries `sqlite_stat1` for index hit counts                                  | ✅     |
| Unused index detection   | `UnusedIndexes(ctx, db)` — finds indexes with zero hits                                         | ✅     |
| WAL checkpoint scheduler | `CheckpointScheduler` — background WAL auto-checkpoint                                          | ✅     |
| Optimization pragmas     | `DefaultOptimizations`, `ApplyOptimizations`, `ApplyWAL` — cache size, memory map, ANALYZE      | ✅     |
| Functional options       | `WithExcludedTables`, `WithAutoAnalyze` — configurable behavior                                 | ✅     |
| Benchmarks               | Indexed vs unindexed `ReadFrom` benchmarks proving value                                        | ✅     |

---

## Projection Runner ✅ FULLY_FUNCTIONAL

> `import "github.com/larsartmann/go-cqrs-lite/projection"`

| Feature                   | Detail                                                                                        | Status |
| ------------------------- | --------------------------------------------------------------------------------------------- | ------ |
| Runner                    | `NewRunner(journal, subscriber, checkpoint, opts...)` — replay → live                         | ✅     |
| Builder + On[T]()         | `NewBuilder(name)` + `On[T](builder, eventType, handler)` — type-safe                         | ✅     |
| HandlerRegistry           | Thread-safe `On(eventType, handler)`, `OnAll(handler)`, `Lookup`                              | ✅     |
| Checkpoint per projection | `CurrentCheckpoint(ctx, name)` — read last-processed event ID                                 | ✅     |
| Reset/rebuild             | `Runner.Reset(ctx, name)` — zero-out checkpoint → full replay on next Run                     | ✅     |
| Event type filtering      | Runner filters events by `Projection.EventTypes()`                                            | ✅     |
| SeekableJournal detection | Auto-detects `SeekableJournal` for position-based replay                                      | ✅     |
| Live-only mode            | Pass `nil` journal to skip replay entirely                                                    | ✅     |
| Retry with backoff        | `WithRetry(count, delay)` — exponential backoff, only if `IsRetryable`                        | ✅     |
| Dead letter queue         | `WithDeadLetterHandler(func)` — callback after retries exhausted                              | ✅     |
| Parallel processing       | `WithParallelism(n)` — semaphore-bounded goroutine pool                                       | ✅     |
| Replay context marking    | `event.WithReplay(ctx, true)` during replay; handlers can distinguish                         | ✅     |
| Close lifecycle           | `Runner.Close()` — cancel internal context, graceful shutdown                                 | ✅     |
| Duplicate name guard      | `Register()` rejects projections with same `Name()`                                           | ✅     |
| Health check              | `Runner.HealthCheck(ctx)`, `Runner.DetailedHealthCheck(ctx)` — projection-level health status | ✅     |

**Sentinel errors:** `ErrNilHandler`, `ErrNilSubscriber`, `ErrNilCheckpoint`, `ErrNoProjections`, `ErrDuplicateProjection`, `ErrAlreadyRunning`

---

## Stream Read Model ✅ FULLY_FUNCTIONAL

> `import "github.com/larsartmann/go-cqrs-lite/listing"`

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

> `import "github.com/larsartmann/go-cqrs-lite/otel"`

| Feature            | Detail                                                                                                                                                                                       | Status |
| ------------------ | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ------ |
| Type aliases       | `Tracer`, `Span`, `SpanKind`, `KeyValue`, `Meter`, `Float64Histogram` — re-exported from OTel                                                                                                | ✅     |
| Tracer factory     | `NewTracer(component)` — creates OTel Tracer with standard instrumentation name                                                                                                              | ✅     |
| Meter factory      | `NewMeter(component)` — creates OTel Meter                                                                                                                                                   | ✅     |
| Span helpers       | `StartSpan`, `RecordError`, `EndWithError`, `AggregateAttrs`, `CommandAttrs`, `EventAttrs`, `QueryAttrs`                                                                                     | ✅     |
| Context helpers    | `SpanFromContext`, `TraceIDFromContext`, `SpanIDFromContext`                                                                                                                                 | ✅     |
| Attribute helpers  | `AttrString`, `AttrInt`, `AttrInt64`, `WithAttributes`, `WithSpanKind`                                                                                                                       | ✅     |
| Metric helpers     | `MetricWithAttributes`, `MetricWithDescription`, `MetricWithUnit`                                                                                                                            | ✅     |
| Logging helpers    | `ComponentLogger`, `ContextLogger` — structured logging with trace correlation                                                                                                               | ✅     |
| Standard constants | `AttrMessageKind`, `AttrCommandType`, `AttrEventType`, `AttrQueryType`, `AttrAggregateType`, `AttrAggregateID`, `AttrAggregateVersion`, `AttrEventCount`, `AttrProjectionName`, `AttrStatus` | ✅     |

---

## Watermill Adapter ✅ FULLY_FUNCTIONAL

> `import "github.com/larsartmann/go-cqrs-lite/watermill"`

| Feature             | Detail                                                                                                   | Status |
| ------------------- | -------------------------------------------------------------------------------------------------------- | ------ |
| Metadata protocol   | Bidirectional `event.Event` ↔ Watermill `message.Message` via 15+ metadata keys                          | ✅     |
| PublisherAdapter    | `NewPublisherAdapter(publisher)` — wraps `event.Publisher` as `message.Publisher`                        | ✅     |
| SubscriberAdapter   | `NewSubscriberAdapter(bus)` — wraps `event.Bus` as `message.Subscriber`, feeds `<-chan *message.Message` | ✅     |
| Full event fidelity | 15 metadata keys preserve ID, type, aggregate, version, schema version, all metadata fields              | ✅     |
| Custom metadata     | `custom.*` prefix preserves all custom metadata entries                                                  | ✅     |

---

## Test Infrastructure 🧪 TESTING_ONLY

### eventtest 🧪

> `import "github.com/larsartmann/go-cqrs-lite/event/v2/eventtest"`

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

---

## Tools 🔧

### cqrs-gen Code Generator 🔧

> `go run github.com/larsartmann/go-cqrs-lite/cmd/cqrs-gen`

| Feature             | Detail                                                                       | Status |
| ------------------- | ---------------------------------------------------------------------------- | ------ |
| AST-based scanning  | Parses Go source for `//cqrs:command <Name>` / `//cqrs:query <Name>` markers | ✅     |
| Typed handler gen   | Generates `Register<StructName>Handler` functions using `RegisterTyped[T]`   | ✅     |
| CLI flags           | `-type` (command/query), `-output` (file), `-pkg` (package name)             | ✅     |
| Recursive directory | Walks directories, skips `_test.go`, extracts markers from doc comments      | ✅     |

### API Stability Checker 🔧

> `go run github.com/larsartmann/go-cqrs-lite/cmd/api-stability`

| Feature                | Detail                                                      | Status |
| ---------------------- | ----------------------------------------------------------- | ------ |
| Module scanning        | Parses all exported symbols from 17 library modules         | ✅     |
| Golden file comparison | Compares current API surface against `docs/api_surface.txt` | ✅     |
| Update mode            | `-update` flag regenerates golden file                      | ✅     |
| Diff reporting         | Reports REMOVED/NEW exports — CI gate for breaking changes  | ✅     |

---

## Integration Tests ✅

> `import "github.com/larsartmann/go-cqrs-lite/integration"`

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

| Example         | Detail                                                                                        |
| --------------- | --------------------------------------------------------------------------------------------- |
| `example/todo/` | Full CQRS app: HTTP API, decider, projections, queries, Pebble + memory storage, saga pattern |
| `example/user/` | Event sourcing lifecycle: create, replay, mutate, signing, SSE, catalog, Docker packaging     |

**Not reference applications.** These demonstrate library usage patterns.

---

## Known Code Quality Issues

Found during code reviews. See `docs/planning/` for details.

| Issue                                                      | Severity | Module              |
| ---------------------------------------------------------- | -------- | ------------------- |
| command re-exports event types (module boundary violation) | HIGH     | command             |
| Reactive extensions not wired into dispatchers             | LOW      | event/command/query |
| Pre-existing golden test drift (codec, middleware)         | LOW      | codec, middleware   |

---

## Not Yet Implemented 📐 PLANNED

Features mentioned in project docs/planning but with **no production code yet**:

| Feature                      | Description                                                                                               |
| ---------------------------- | --------------------------------------------------------------------------------------------------------- |
| Reactive CommandBus/QueryBus | `ro.Subject[Command]` / `ro.Subject[Query]` reactive streams (mentioned in AGENTS.md but not implemented) |
| Schema registry              | JSON Schema middleware for event validation                                                               |
| PostgreSQL integration tests | testcontainers-based real PG testing                                                                      |
| Documentation site           | Docusaurus/MkDocs/Hugo site                                                                               |

---

## Module Maturity Matrix

| Module                 | Import Path                 | Maturity        |
| ---------------------- | --------------------------- | --------------- |
| `event`                | `…/event/v2`                | ✅ Production   |
| `event/eventtest`      | `…/event/v2/eventtest`      | 🧪 Test helper  |
| `command`              | `…/command/v2`              | ✅ Production   |
| `query`                | `…/query/v2`                | ✅ Production   |
| `decider`              | `…/decider/v2`              | ✅ Production   |
| `id`                   | `…/id/v2`                   | ✅ Production   |
| `dispatcher`           | `…/dispatcher/v2`           | ✅ Production   |
| `schema`               | `…/schema/v2`               | ✅ Production   |
| `snapshot`             | `…/snapshot/v2`             | ✅ Production   |
| `codec`                | `…/codec/v2`                | ✅ Production   |
| `memory`               | `…/memory/v2`               | 🧪 Test utility |
| `catalog`              | `…/catalog/v2`              | ✅ Production   |
| `catalog/asyncapi`     | `…/catalog/v2/asyncapi`     | ✅ Production   |
| `catalog/d2`           | `…/catalog/v2/d2`           | ✅ Production   |
| `catalog/openapi`      | `…/catalog/v2/openapi`      | ✅ Production   |
| `catalog/eventcatalog` | `…/catalog/v2/eventcatalog` | ✅ Production   |
| `catalog/docserver`    | `…/catalog/v2/docserver`    | ✅ Production   |
| `catalog/schema`       | `…/catalog/v2/schema`       | ✅ Production   |
| `middleware`           | `…/middleware/v2`           | ✅ Production   |
| `integration`          | `…/integration/v2`          | ✅ Test suite   |
| `projection`           | `…/projection/v2`           | ✅ Production   |
| `signing`              | `…/signing/v2`              | ✅ Production   |
| `signing/multisig`     | `…/signing/v2/multisig`     | ✅ Production   |
| `encryption`           | `…/encryption/v2`           | ✅ Production   |
| `storage`              | `…/storage/v2`              | ✅ Production   |
| `storage/sql`          | `…/storage/v2/sql`          | 🧪 Shared infra |
| `watermill`            | `…/watermill/v2`            | ✅ Production   |
| `listing`              | `…/listing/v2`              | ✅ Production   |
| `otel`                 | `…/otel/v2`                 | ✅ Production   |
| `pebble`               | `…/pebble/v2`               | ✅ Production   |
| `turso`                | `…/turso/v2`                | ✅ Production   |
| `turso/indexing`       | `…/turso/v2/indexing`       | ✅ Production   |
| `cmd/cqrs-gen`         | `…/cmd/cqrs-gen/v2`         | 🔧 Tool         |
| `cmd/api-stability`    | `…/cmd/api-stability/v2`    | 🔧 Tool         |
| `example/user`         | `…/example/user`            | 💡 Demo         |
| `example/todo`         | `…/example/todo`            | 💡 Demo         |

---

## Architecture Guarantees

| Guarantee              | Detail                                                                           |
| ---------------------- | -------------------------------------------------------------------------------- |
| Zero lint issues       | Clean golangci-lint across all modules                                           |
| Race-free              | `go test -race` passes across all modules                                        |
| Multi-module isolation | Each module has independent `go.mod`, no circular dependencies                   |
| Interface-first        | All core types are interfaces — provide your own implementations                 |
| Library, not framework | Import what you need, compose your own stack                                     |
| Context-aware          | All handlers accept `context.Context`                                            |
| Errors as values       | No panics in production code, explicit error returns, sentinel errors + wrapping |
| Defensive copies       | All public accessors return copies — callers cannot mutate internals             |
| Tombstone over delete  | Soft-delete via metadata — no `Delete` on Store                                  |

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
