## 5. Module Reference (quick lookup)

### Core (Layer 0–3)

| Module       | Import          | One-liner                                                                                                                                                                                                                                                                |
| ------------ | --------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| `id`         | `id/v4`         | Branded IDs: `id.Of[T]` = `cbid.ID[T, ulid.ULID]`. All 8 markers exported (`StreamMarker`, `EventMarker`, `CommandMarker`, …) for `BrandNamer` integration. Custom via `id.Of[struct{}]`.                                                                             |
| `dispatcher` | `dispatcher/v4` | Generic `Dispatcher[H, M]` with `LifecycleMixin`. Base for command/query dispatchers.                                                                                                                                                                                    |
| `codec`      | `codec/v4`      | Payload encoding: `JSONCodec{}`, `CBORCodec{}` (deterministic), `RawCodec{}`, `ForEncoding(enc)`, `AutoDetect(data)`, `Size(v)`.                                                                                                                                         |
| `event`      | `event/v4`      | `Event`, `Store` (=`EventSink`+`EventSource`), `Bus`, `Journal`, `SeekableJournal`, `NewEvent`, `NewEvents`, `DecodePayload[T]`, `DecodePayloadAuto[T]`, `DefaultCodec`, 5-family errors, tombstone (`TombstoneMark`), causality (`Causation`), `Tracing`, `Checkpoint`. |
| `command`    | `command/v4`    | `Dispatcher`, `Handler`, `RegisterTyped`, `BasicCommand`, `PersistedCommand`, `CommandSink`/`Source`, `CommandBus` (pub/sub).                                                                                                                                            |
| `query`      | `query/v4`      | `Dispatcher`, `TypedHandler[Q,R]`, `RegisterTyped`, `PaginatedResult[T]`, `PersistedQuery`, `QuerySink`/`Source`.                                                                                                                                                        |
| `decider`    | `decider/v4`    | `Decider[State]{Initial, Apply}`, `Repository[State]` (`Execute`, `Load`, `LoadAtVersion`), `NewStateCache`, snapshot integration.                                                                                                                                       |

### Read models (Layer 4–5)

| Module    | Import       | One-liner                                                                                                              |
| --------- | ------------ | ---------------------------------------------------------------------------------------------------------------------- |
| `kv`      | `kv/v4`      | `ViewStore[V,K]` interface, `TypedStore[V,K]`, `Cache[V,K]`, `MemStore`. Foundation for all read models.               |
| `stack`   | `stack/v4`   | `Materialize[V,K]` (deployer-first projection builder), `Bundle`, presets. Accepts any `kv.ViewStore`.                 |
| `listing` | `listing/v4` | `StreamListing`, `StreamStatus` (Active/Tombstoned/Undetermined), `StatusMiddleware`, `InMemoryStreamReader`. |
| `query`   | `query/v4`   | (see Core) — query the read model.                                                                                     |

### Storage (Layer 5)

| Module     | Import              | One-liner                                                                                                                                                                                                                                                                       |
| ---------- | ------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `memory`   | `storage/memory/v4` | `MemoryStore`, `MemorySnapshotStore`, `MemoryCheckpointStore`, `MemoryCommandStore`, `MemoryQueryStore`. Tests & dev. (`MemoryBus`/`MemoryCommandBus` deprecated — use `watermill.EventBus`)                                                                                    |
| `storage`  | `storage/v4`        | `SQLEventStore`, `SQLSnapshotStore`, `SQLCheckpointStore`, `SQLCommandStore`, `SQLQueryStore`, `SQLKVStore`, **`SQLViewStore`** (column-mapped views). PG/SQLite. `NewSQLiteBackend`/`NewSQLBackend` facade. `sql/` sub-package: `RunInTx`, `IsDuplicateKeyError`, `ScanSlice`. |
| `pebble`   | `storage/pebble/v4` | `EventStore`, `SnapshotStore`, `CheckpointStore`, `NewKVStore`. CBOR envelope. Shared DB via disjoint key prefixes. `Open()` facade.                                                                                                                                            |
| `turso`    | `storage/turso/v4`  | Turso connector, embedded sync, `indexing/` sub-package for index management. Delegates to `storage`.                                                                                                                                                                           |
| `kv`       | `kv/v4`             | `Store` (Reader+Writer+Closer), `MemStore`, `Iterator`, `Batch`, `TypedStore[T,K]`, `Cache[T,K]` (Otter LRU).                                                                                                                                                                   |
| `snapshot` | `snapshot/v4`       | `Snapshot`, `SnapshotSink`/`Source`/`Store`, `SnapshotStrategy`, `EveryNEvents(n)`, `NewReadPressure(loads)`.                                                                                                                                                                   |
| `schema`   | `schema/v4`         | `Upcaster`, `VersionedStore`, `VersionedSeekableJournal`, `Validator`, `RegisterType[T]()`. Schema evolution on read.                                                                                                                                                           |

### Cross-cutting (Layer 4–5)

| Module           | Import              | One-liner                                                                                                                                      |
| ---------------- | ------------------- | ---------------------------------------------------------------------------------------------------------------------------------------------- |
| `signing`        | `signing/v4`        | `NewHMAC`, `NewEd25519`, `multisig`, `SignMiddleware`/`VerifyMiddleware`. Tamper-proof streams.                                                |
| `encryption`     | `encryption/v4`     | `NewXChaCha20Poly1305`, `NewAES256GCM`, `Codec` wrapper, `EncryptMiddleware`/`DecryptMiddleware`, `StaticKeyResolver`.                         |
| `middleware`     | `middleware/v4`     | `Logging`, `Retry`, `Recovery`, `Validation`, `Metrics`, `CircuitBreaker`, `EventTracing`, `CommandMetrics`, etc. For command + event + query. |
| `transport/http` | `transport/http/v4` | `NewSSEBroker`, `SSEHandler`. Bridges `event.Bus` to Server-Sent Events HTTP clients.                                                          |
| `otel`           | `otel/v4`           | `Tracer`, `Meter`, `Spans`, `Attributes`. Re-exports — import this, not go.opentelemetry.io.                                                   |
| `catalog`        | `catalog/v4`        | `Registry`, `SchemaFromType[T]()`, exporters: `asyncapi`, `d2`, `eventcatalog`, `openapi`.                                                     |
| `watermill`      | `watermill/v4`      | `EventBus` (GoChannel-backed, replaces `memory.MemoryBus`), `CatchUpSubscriber`, `EventPublisher`, `MessageToEvent`. ADR-0028.                 |

### Reliability & Testing (Layer 1–3)

| Module           | Import              | One-liner                                                                                                                                                                                                                                           |
| ---------------- | ------------------- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `idempotency`    | `idempotency/v4`    | `Store`, `MemoryStore`, `KVStore` (any `kv.Store`+`ConditionalWriter`), `ErrDuplicate`. Middleware in `middleware/`.                                                                                                                                |
| `idempotency/kvstore` | `idempotency/kvstore/v4` | `KVStore`, `KVBackend`. KV-backed idempotency (optional subpackage, pulls `kv/`). |
| `idempotency/sqlstore` | `idempotency/sqlstore/v4` | `NewSQLiteStore`, `NewPostgresStore`. SQL-backed idempotency (`INSERT ON CONFLICT`, TTL sweep). |
| `retry`     | `retry/v4`     | Zero-dep retry with exponential backoff+jitter: `Do`, `Config`, `ErrExhausted`. Standalone — no CQRS/OTel deps. |
| `dedup`     | `dedup/v4`     | `Ring`, `DefaultCapacity`. Bounded O(1) fixed-capacity ID dedup for stream boundaries. |
| `scheduling`     | `scheduling/v4`     | `TimerStore`, `MemoryTimerStore`, `Scheduler` (poll + retry). Idempotent durable deadlines ("cancel order after 30 min").                                                                                                                           |
| `projection`     | `projection/v4`     | `Projection`, `NewProjection`. Consumer-side projection interface extracted from `event/`.                                                                                                                                                          |
| `projectionhost` | `projectionhost/v4` | `Host`, `WorkerState`, `DeadLetterStore`, `SQLiteDeadLetterStore`, `DeadLetterStoreAdmin` (Count/ListPaged/PurgeBefore), `MemoryDeadLetterStore`, `RegisterAndWait`, `ReplayDeadLetters`. Managed lifecycle: crash-restart, checkpoint, poison DLQ. |
| `scenario`       | `scenario/v4`       | Fluent BDD: `Given/When/Then`, `ThenError`, `ThenState`, `GivenProjection/ThenNoError`. Test deciders + projections.                                                                                                                                |

### Tooling (Layer 6)

| Module              | Import               | One-liner                                                                                                    |
| ------------------- | -------------------- | ------------------------------------------------------------------------------------------------------------ |
| `testutil`          | `testutil/v4`        | `MustNewCmd(tb, ...)`, `NoopCommandHandler`. Shared test helpers (zero panics).                              |
| `id/idtest`         | `id/v4/idtest`       | `ParseStreamID(tb, s)`, `ParseEventID(tb, s)`. Branded-ID test helpers — `tb.Fatalf` on error, no panics. |
| `query/querytest`   | `query/v4/querytest` | `New(tb, queryType)`. Construct valid test queries — `tb.Fatalf` on error.                                   |
| `event/eventtest`   | `event/v4/eventtest` | `FakeStore`, `FakeBus`, `AssertGolden`. Event test doubles and golden test helpers.                          |
| `cmd/cqrs-gen`      | (go install)         | Code generator: typed handler registration from `//cqrs:command` / `//cqrs:query` markers.                   |
| `cmd/doc-check`     | (go run)             | Doc verifier: scans Markdown for Go code references, checks symbols exist.                                   |
| `cmd/api-stability` | (go install)         | API surface checker: compares exports against `docs/api_surface.txt` golden file.                            |
| `cmd/cqrs-bench`    | (go build)           | Benchmarking CLI: synthetic event workloads against memory/sqlite/pebble. `cqrs-bench run --backend sqlite --profile small`. |
| `benchkit`    | `benchkit/v4`        | Factory-driven benchmarking suite: `Run`/`Compare`, latency percentiles, throughput, memory. Mirrors contracttest pattern. |
| `transport/grpc`    | `transport/grpc/v4`  | `RegisterCommandService`, `RegisterQueryService`, `NewCommandClient`, `NewQueryClient`. gRPC transport.      |

### Reactive & Advanced Read Models (Layer 2–5)

| Module       | Import          | One-liner                                                                                                                                                  |
| ------------ | --------------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `deriver`    | `deriver/v4`    | Event→command derivation. Pure functions, composable (`.Then` fan-out, `.Filter`), wires into `bus.SubscribeAll`. Saga alternative. ADR-0040.              |
| `graph`      | `graph/v4`      | Third projection tier: nodes + edges for traversal-heavy read models. `GraphProjection`, `MemoryDriver`, `Schema` validation, native Cypher/Gremlin reads. |
| `prometheus` | `prometheus/v4` | OTel→Prometheus bridge: `Setup()` → MeterProvider + `/metrics` handler. Exposes all CQRS instruments. `WithRegistry()` for custom registries.              |
| `metaengine` | `metaengine/v4` | Cost-based storage planner: `Engine`, `Store`, `Plan`, `Query[Q,R]`, `Fold`, 7 ADTs (Map/Set/Counter/Graph/Multimap/Log/Scan), `NewSQLiteEngine`, `NewMemoryEngine`. Picks the cheapest backend per query. ADR-0047. |
| `metaengine/projectionadapter` | `metaengine/projectionadapter/v4` | Wraps a `metaengine.Store` as a `projection.Projection`. Bridges cost-planned stores into `projectionhost`. ADR-0062. |
