## 5. Module Reference (quick lookup)

### Core (Layer 0–3)

| Module       | Import          | One-liner                                                                                                                                                                                                                                                                |
| ------------ | --------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| `id`         | `id/v3`         | Branded IDs: `id.Of[T]` = `cbid.ID[T, ulid.ULID]`. All 8 markers exported (`AggregateMarker`, `EventMarker`, `CommandMarker`, …) for `BrandNamer` integration. Custom via `id.Of[struct{}]`.                                                                             |
| `dispatcher` | `dispatcher/v3` | Generic `Dispatcher[H, M]` with `LifecycleMixin`. Base for command/query dispatchers.                                                                                                                                                                                    |
| `codec`      | `codec/v3`      | Payload encoding: `JSONCodec{}`, `CBORCodec{}` (deterministic), `RawCodec{}`, `ForEncoding(enc)`, `AutoDetect(data)`, `Size(v)`.                                                                                                                                         |
| `event`      | `event/v3`      | `Event`, `Store` (=`EventSink`+`EventSource`), `Bus`, `Journal`, `SeekableJournal`, `NewEvent`, `NewEvents`, `DecodePayload[T]`, `DecodePayloadAuto[T]`, `DefaultCodec`, 5-family errors, tombstone (`TombstoneMark`), causality (`Causation`), `Tracing`, `Checkpoint`. |
| `command`    | `command/v3`    | `Dispatcher`, `Handler`, `RegisterTyped`, `BasicCommand`, `PersistedCommand`, `CommandSink`/`Source`, `CommandBus` (pub/sub).                                                                                                                                            |
| `query`      | `query/v3`      | `Dispatcher`, `TypedHandler[Q,R]`, `RegisterTyped`, `PaginatedResult[T]`, `PersistedQuery`, `QuerySink`/`Source`.                                                                                                                                                        |
| `decider`    | `decider/v3`    | `Decider[State]{Initial, Apply}`, `Repository[State]` (`Execute`, `Load`, `LoadAtVersion`), `NewStateCache`, snapshot integration.                                                                                                                                       |

### Read models (Layer 4–5)

| Module    | Import       | One-liner                                                                                                              |
| --------- | ------------ | ---------------------------------------------------------------------------------------------------------------------- |
| `kv`      | `kv/v3`      | `ViewStore[V,K]` interface, `TypedStore[V,K]`, `Cache[V,K]`, `MemStore`. Foundation for all read models.               |
| `stack`   | `stack/v3`   | `Materialize[V,K]` (deployer-first projection builder), `Bundle`, presets. Accepts any `kv.ViewStore`.                 |
| `listing` | `listing/v3` | `AggregateListing`, `AggregateStatus` (Active/Tombstoned/Undetermined), `StatusMiddleware`, `InMemoryAggregateReader`. |
| `query`   | `query/v3`   | (see Core) — query the read model.                                                                                     |

### Storage (Layer 5)

| Module     | Import              | One-liner                                                                                                                                                                                                                                                                       |
| ---------- | ------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `memory`   | `storage/memory/v3` | `MemoryStore`, `MemorySnapshotStore`, `MemoryCheckpointStore`, `MemoryCommandStore`, `MemoryQueryStore`. Tests & dev. (`MemoryBus`/`MemoryCommandBus` deprecated — use `watermill.EventBus`)                                                                                    |
| `storage`  | `storage/v3`        | `SQLEventStore`, `SQLSnapshotStore`, `SQLCheckpointStore`, `SQLCommandStore`, `SQLQueryStore`, `SQLKVStore`, **`SQLViewStore`** (column-mapped views). PG/SQLite. `NewSQLiteBackend`/`NewSQLBackend` facade. `sql/` sub-package: `RunInTx`, `IsDuplicateKeyError`, `ScanSlice`. |
| `pebble`   | `storage/pebble/v3` | `EventStore`, `SnapshotStore`, `CheckpointStore`, `NewKVStore`. CBOR envelope. Shared DB via disjoint key prefixes. `Open()` facade.                                                                                                                                            |
| `turso`    | `storage/turso/v3`  | Turso connector, embedded sync, `indexing/` sub-package for index management. Delegates to `storage`.                                                                                                                                                                           |
| `kv`       | `kv/v3`             | `Store` (Reader+Writer+Closer), `MemStore`, `Iterator`, `Batch`, `TypedStore[T,K]`, `Cache[T,K]` (Otter LRU).                                                                                                                                                                   |
| `snapshot` | `snapshot/v3`       | `Snapshot`, `SnapshotSink`/`Source`/`Store`, `SnapshotStrategy`, `EveryNEvents(n)`, `NewReadPressure(loads)`.                                                                                                                                                                   |
| `schema`   | `schema/v3`         | `Upcaster`, `VersionedStore`, `VersionedSeekableJournal`, `Validator`, `RegisterType[T]()`. Schema evolution on read.                                                                                                                                                           |

### Cross-cutting (Layer 4–5)

| Module           | Import              | One-liner                                                                                                                                      |
| ---------------- | ------------------- | ---------------------------------------------------------------------------------------------------------------------------------------------- |
| `signing`        | `signing/v3`        | `NewHMAC`, `NewEd25519`, `multisig`, `SignMiddleware`/`VerifyMiddleware`. Tamper-proof streams.                                                |
| `encryption`     | `encryption/v3`     | `NewXChaCha20Poly1305`, `NewAES256GCM`, `Codec` wrapper, `EncryptMiddleware`/`DecryptMiddleware`, `StaticKeyResolver`.                         |
| `middleware`     | `middleware/v3`     | `Logging`, `Retry`, `Recovery`, `Validation`, `Metrics`, `CircuitBreaker`, `EventTracing`, `CommandMetrics`, etc. For command + event + query. |
| `transport/http` | `transport/http/v3` | `NewSSEBroker`, `SSEHandler`. Bridges `event.Bus` to Server-Sent Events HTTP clients.                                                          |
| `otel`           | `otel/v3`           | `Tracer`, `Meter`, `Spans`, `Attributes`. Re-exports — import this, not go.opentelemetry.io.                                                   |
| `catalog`        | `catalog/v3`        | `Registry`, `SchemaFromType[T]()`, exporters: `asyncapi`, `d2`, `eventcatalog`, `openapi`.                                                     |
| `watermill`      | `watermill/v3`      | `EventBus` (GoChannel-backed, replaces `memory.MemoryBus`), `CatchUpSubscriber`, `EventPublisher`, `MessageToEvent`. ADR-0028.                 |

### Reliability & Testing (Layer 1–3)

| Module           | Import              | One-liner                                                                                                                                                                                                                                           |
| ---------------- | ------------------- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `idempotency`    | `idempotency/v3`    | `Store`, `MemoryStore`, `KVStore` (any `kv.Store`+`ConditionalWriter`), `ErrDuplicate`. Middleware in `middleware/`.                                                                                                                                |
| `scheduling`     | `scheduling/v3`     | `TimerStore`, `MemoryTimerStore`, `Scheduler` (poll + retry). Idempotent durable deadlines ("cancel order after 30 min").                                                                                                                           |
| `projection`     | `projection/v3`     | `Projection`, `NewProjection`. Consumer-side projection interface extracted from `event/`.                                                                                                                                                          |
| `projectionhost` | `projectionhost/v3` | `Host`, `WorkerState`, `DeadLetterStore`, `SQLiteDeadLetterStore`, `DeadLetterStoreAdmin` (Count/ListPaged/PurgeBefore), `MemoryDeadLetterStore`, `RegisterAndWait`, `ReplayDeadLetters`. Managed lifecycle: crash-restart, checkpoint, poison DLQ. |
| `scenario`       | `scenario/v3`       | Fluent BDD: `Given/When/Then`, `ThenError`, `ThenState`, `GivenProjection/ThenNoError`. Test deciders + projections.                                                                                                                                |

### Tooling (Layer 6)

| Module              | Import               | One-liner                                                                                                    |
| ------------------- | -------------------- | ------------------------------------------------------------------------------------------------------------ |
| `testutil`          | `testutil/v3`        | `MustNewCmd(tb, ...)`, `NoopCommandHandler`. Shared test helpers (zero panics).                              |
| `id/idtest`         | `id/v3/idtest`       | `ParseAggregateID(tb, s)`, `ParseEventID(tb, s)`. Branded-ID test helpers — `tb.Fatalf` on error, no panics. |
| `query/querytest`   | `query/v3/querytest` | `New(tb, queryType)`. Construct valid test queries — `tb.Fatalf` on error.                                   |
| `event/eventtest`   | `event/v3/eventtest` | `FakeStore`, `FakeBus`, `AssertGolden`. Event test doubles and golden test helpers.                          |
| `cmd/cqrs-gen`      | (go install)         | Code generator: typed handler registration from `//cqrs:command` / `//cqrs:query` markers.                   |
| `cmd/doc-check`     | (go run)             | Doc verifier: scans Markdown for Go code references, checks symbols exist.                                   |
| `cmd/api-stability` | (go install)         | API surface checker: compares exports against `docs/api_surface.txt` golden file.                            |
| `transport/grpc`    | `transport/grpc/v3`  | `RegisterCommandService`, `RegisterQueryService`, `NewCommandClient`, `NewQueryClient`. gRPC transport.      |

### Reactive & Advanced Read Models (Layer 2–5)

| Module       | Import          | One-liner                                                                                                                                                  |
| ------------ | --------------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `deriver`    | `deriver/v3`    | Event→command derivation. Pure functions, composable (`.Then` fan-out, `.Filter`), wires into `bus.SubscribeAll`. Saga alternative. ADR-0040.              |
| `graph`      | `graph/v3`      | Third projection tier: nodes + edges for traversal-heavy read models. `GraphProjection`, `MemoryDriver`, `Schema` validation, native Cypher/Gremlin reads. |
| `prometheus` | `prometheus/v3` | OTel→Prometheus bridge: `Setup()` → MeterProvider + `/metrics` handler. Exposes all CQRS instruments. `WithRegistry()` for custom registries.              |
