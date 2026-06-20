# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/).

## [Unreleased]

### Added

- **Streaming event reads** — `StreamingSource`/`StreamingJournal` now implemented on all three stores: `SQLEventStore` (cursor-based via `*sql.Rows`), Pebble `EventStore` (iterator-based with limit + skip), `MemoryStore` (SliceIterator-wrapped). Consumers can type-assert to streaming interfaces uniformly across backends.
- **DistributedRunner** (`projection/`) — Wraps a projection Runner with `LeaderElection` gating. Waits for leadership, runs replay+live, periodically checks `IsLeader` (default 5s), stops gracefully on loss, resigns on exit. `WithLeadershipCheckInterval` option for tuning detection latency.
- **cqrs-gen event handler generation** (`cmd/cqrs-gen/`) — `-type=event` generates typed projection handler registration functions via `projection.On[T]()`. Supports both `//cqrs:event` comment markers and `cqrs:"event:Type"` struct tags.
- **Postgres LISTEN/NOTIFY event bus** (`storage/`) — `PostgresBus` implements `event.Bus` using `SELECT pg_notify()` with lightweight JSON reference payloads (under 8KB). `NotificationListener` interface abstracts driver-specific LISTEN; the bus calls `Listen(channel)` itself so consumers don't need to pre-arm. Listener re-fetches full events from store with retry for visibility-gap handling. Uses `LoadByEventID` (indexed O(1) lookup) when the store implements `EventByIDLoader`. **Wired into `stack/postgres` preset** via `WithDistributedBus(listener)` option.
- **PgxListener** (`stack/postgres/`) — `PgxListener` implements `storage.NotificationListener` using `pgxpool`. Dedicated single-connection pool for LISTEN; channel-name allow-list defends against SQL injection. `NewPgxListener(pool)` wraps an existing pool; `NewPgxListenerFromDSN(ctx, dsn)` creates an owned single-conn pool.
- **PostgresBus otel spans** — `pg_bus.publish` (SpanKindInternal) and `pg_bus.handle_notification` (SpanKindConsumer) spans for distributed tracing of NOTIFY round-trips.
- **Real-Postgres integration tests** (`stack/postgres/`) — Three `-tags=integration` tests covering the full LISTEN/NOTIFY round-trip, channel validation, and preset wiring. Run in CI's `postgres-integration` job.
- **Documentation site content** — `docs/index.md` landing page with value proposition, quick start, module overview, presets comparison table.
- **PgxListener auto-reconnect** (`stack/postgres/`) — On connection loss, the listener automatically re-acquires a connection and re-issues LISTEN with exponential backoff (default: 10 attempts, 1s→30s). Configurable via `WithReconnect(maxAttempts)`, `WithReconnectBackoff(initial, max)`, `WithoutReconnect()`. A dropped connection no longer silently kills event delivery.
- **PgxListener deadlock regression test** — `TestPgxListener_CloseDoesNotDeadlock` asserts Close() returns within 2s when the receive loop is running, preventing regression of the critical cancelFn fix.
- **Property-based channel-name validation** — `rapid` property tests (3 properties × 100 inputs) covering valid identifiers, digit-first rejection, and no-panic-on-arbitrary-input.

### Changed

- **SSE moved to transport/http/** (`transport/http/`) — SSE broker moved from `middleware/` to new `transport/http/` module (ADR-0025). SSE wire format rewritten with proper `SSEEvent` struct, spec-correct multi-line `data:` handling, `SSEEventID` branded type, and 15s heartbeat to prevent proxy timeouts. Healthcheck, metrics_http, and pprof handlers deleted (generic utilities, zero CQRS deps, zero consumers).
- **Ghost streaming interfaces removed** — Consolidated the old `StreamLoader`/`EventStream` types (bool-based `Next()` + `Err()`) into the shipped `EventIterator` interface (standard Go `io.EOF` pattern). Dead code that never compiled against the real interface is gone.
- **WASM compilation** — All 7 core modules (id, codec, dispatcher, event, command, query, decider) now compile to `GOOS=js GOARCH=wasm`. Moved `NewCQRSViews()` behind `//go:build !js` to exclude the OTel SDK's `os/user` dependency.
- **notifyPayload type model** — Replaced 5 stringly-typed fields with branded domain types (`id.EventID`, `event.Type`, `event.AggregateType`, `id.AggregateID`, `event.Version`). Eliminates the manual `String()`→`Parse` roundtrip on the receive side.
- **pgx upgraded v5.7.1 → v5.10.0** — Patches critical memory-safety vulnerability (CVE) and SQL-injection via placeholder confusion.
- **API surface** — 1806 → 1852 exports.

## [2.7.0] - 2026-06-19

The **Bundle composition layer**: consumers stop deciding on infrastructure. A deployer picks a backend via one preset call; the application imports only `readmodel` and `stack` and never touches a storage driver. 8 new modules (~5,500 lines), persistent read models for every preset, a shared contract suite, and a zero-lint release gate.

### Added

- **Bundle composition root** (`stack/v2`) — `Bundle` with ISP-honest fields (EventSink/EventSource/Journal kept separate, not a fat Store), `Option = func(*Bundle)`, pointer-deduplicated `Close()`, and rollback-on-validation. Repository/ReadModel helpers are top-level generic functions (`stack.Repository[State]`, `stack.ReadModel[T,K]`) since Go forbids generic methods.
- **Bundle presets** — `stack/memory`, `stack/sqlite` (modernc, WAL, auto-migrate), `stack/pebble` (single PebbleDB for all stores via disjoint key prefixes), `stack/postgres` (pgx, auto-migrate). Each wires event store+bus, command/query/snapshot/checkpoint stores, and a read-model backend in one call.
- **Typed read-model store** (`readmodel/v2`) — `Store[T any, K fmt.Stringer]` over `kv.Store` with codec + key prefixing; `Backend` is an alias for `kv.Store`, so `kv.MemStore`, `pebble.KVAdapter`, and the new SQL KV store all satisfy it.
- **Read-model cache decorator** (`readmodel/cache/v2`) — Otter-backed `CachedStore[T,K]` (TinyLFU admission) with capacity + TTL, write-through.
- **Typed stores** — `snapshot.TypedSnapshot[State]` + `TypedStore` (closes the `[]byte` hole on snapshot state); `command.TypedCommandStore[P]` (with `AppendBatch`); `query.TypedQueryStore[P]`. Encode/decode happens once at the adapter boundary.
- **Pebble gaps closed** (`pebble/v2`) — `CommandStore`, `QueryStore`, and `ReadModels()` accessor on `Backend`; EventStore.Close() is now a no-op so the Backend owns the DB lifecycle (fixes a double-close).
- **SQL-backed kv.Store** (`storage/v2`) — `SQLKVStore` implements `kv.Store` over a `cqrs_kv` table (Get/Set/Has/Delete/streaming-Iterator/transactional-Batch), exposed via `SQLBackend.KVStore()`. SQLite and Postgres presets now **persist read models across restarts** instead of using `kv.MemStore`. Verified by an E2E reopen test.
- **Shared contract test suite** (`stack/contracttest`) — `RunSuite(t, factory)` runs 5 behavioural checks; 4 presets × 5 = 20 contract assertions.
- **Zero-overhead benchmarks** (`stack/bench/v2`) — proves Bundle field access is a direct struct read (~0.20 ns/op).
- **godoc example** (`stack/memory`) — `ExampleNew` renders the canonical Bundle entry point on pkg.go.dev.

### Changed

- **Dialect interface** (`storage/v2/sql`) — gained `KVSchema()` for the `cqrs_kv` table (BLOB for SQLite, BYTEA for Postgres). The only implementations are the in-package `PostgresDialect`/`SQLiteDialect`; upsert uses `ON CONFLICT(key) DO UPDATE … excluded.value`, identical across dialects.
- **Lint app resilience** (`flake.nix`) — `nix run .#lint` now reports every failing module instead of aborting on the first (it ran under `errexit`).
- **API surface** — 1351 → 1784 exports; golden file regenerated and the checker's module list expanded to 33 consumer-facing modules.
- **Example rewrite** (`example/todo`) — uses the pebble Bundle preset + `readmodel.Store`; dead `storage/` package deleted (7 files).

### Fixed

- **Postgres preset tests ran in CI** — the `postgres-integration` job set `DATABASE_URL` (read by `storage` tests) but not `POSTGRES_TEST_DSN` (read by `stack/postgres` tests), so preset tests were silently skipped despite a running container. Now sets both and runs the preset suite.
- **Zero lint violations** — 39 violations shipped under `--no--verify` in the first pass are now 0 across all 34 modules (readmodel, pebble, snapshot, stack presets cleaned up).
- **Workspace** — `go work sync` applied; dependency budgets reconciled (`DEP_BUDGET[storage]` 11→12 for the new kv dep).

### Infrastructure

- CI matrix, `flake.nix`, `check-module-layers.sh`, and `.golangci.yml` updated for the 8 new modules (otter + pgx added to depguard).

## [2.6.0] - 2026-06-19

27 commits since v2.5.0. Two new modules (schema validator, prometheus exporter), projection replay/live split, replay→live dedup pipeline, OTel correlation enricher, bounded dedup, streaming event reads, exported ID marker types, cqrs-gen struct tags, and leader election interface.

`pebble.DeleteEventsBefore` (added in v2.5.0, the immediately prior release ~24h earlier) is removed: it contradicted event-sourcing immutability and no consumer could depend on it between releases. No other existing API removed or renamed.

### Added

- **Schema registry validator** (`schema/v2`) — `Validator` with `RegisterType[T]()`, `RegisterTypeWithValidator[T]()`, strict/lenient modes, custom codec support. Returns `Rejection` errors on invalid payloads. ADR-0017 accepted
- **Prometheus metrics exporter** (`prometheus/v2`) — New module wrapping OTel Prometheus exporter. `Setup()` creates a `MeterProvider` backed by a Prometheus registry and an HTTP handler for `/metrics`. `WithRegistry()`, `WithHandlerOptions()`, `MustSetup()`
- **Bounded dedup** (`event/v2`, `projection/v2`) — `DistinctByEventIDBounded(cap)` with FIFO ring eviction for bounded memory in 24/7 projections. `DistinctByEventIDBoundedWith(cap, seen)` seeded variant. `WithDedupCapacity(n)` Runner option
- **Streaming event reads** (`event/v2`) — `EventIterator` interface for one-at-a-time event reading without materializing slices. `StreamingSource` and `StreamingJournal` opt-in interfaces. `SliceIterator` adapts pre-loaded slices
- **cqrs-gen struct tag scanning** (`cmd/cqrs-gen`) — Supports `cqrs:"command:CreateUser"` struct tags on `_ struct{}` fields in addition to `//cqrs:command CreateUser` comment markers. Comment markers take precedence
- **LeaderElection interface** (`projection/v2`) — `LeaderElection` interface + `AlwaysLeader` default for distributed projection coordination per ADR-0018. Consumers implement coordination (Redis, etcd, k8s); library provides interface and default
- **Projection replay/live split** (`projection/v2`) — `Runner.RunReplay(ctx)` replays historical events synchronously and returns once the read model is caught up (read-your-writes); `Runner.RunLive(ctx)` then tails live events in the background. `Run` remains as a convenience wrapper calling both. Eliminates `time.Sleep`-based catch-up hacks in consumers. Adds `ErrReplayRequired` when `RunLive` is called before `RunReplay`
- **Replay→live dedup pipeline** (`event/v2`, `projection/v2`) — Closes the duplicate-processing gap at the replay→live boundary. New `event.SubscriberToObservable` adapts callback-based `Subscriber` to `ro.Observable[Event]`; `event.DistinctByEventIDWith(seen)` seeds the dedup set with IDs from journal replay. The Runner's live path now builds `live → DistinctByEventIDWith(replayIDs) → handler`, suppressing overlap-window duplicates
- **OTel correlation enricher** (`middleware/v2`) — `OTelCorrelationEnricher` bridges OTel baggage correlation IDs into event metadata via `event.WithCustom`. Composes with `CommandCausalityEnricher` via `CompositeEnricher`. New `OTelCorrelationIDFromEvent` extractor and `MetadataKeyOTelCorrelationID` constant
- **Exported ID marker types** (`id/v2`) — All 8 phantom marker types are now exported (`AggregateMarker`, `UserMarker`, `CorrelationMarker`, `RequestMarker`, `CausationMarker`, `ClientMarker`, `CommandMarker`, `EventMarker`), enabling downstream `go-branded-id` `BrandNamer` integration and other type-parameterized tooling against the root module's ID types

### Removed

- **Pebble `DeleteEventsBefore`** (`pebble/v2`) — Removed. Events are immutable truth; automatic event deletion contradicts event sourcing principles. Introduced in v2.5.0 (immediately prior release) and removed before any consumer could adopt it. The `Flush()` method remains for durability control

## [2.5.0] - 2026-06-18

70 commits since v2.4.0. Pebble backup/retention/consistent reads, OpenTelemetry baggage correlation + metric views + propagator, load coalescing via singleflight, HKDF multi-tenant key derivation, CBOR streaming, reactive event dedup operators, Watermill middleware wrappers, and turso race fixes. No breaking API changes.

### Added

- **Pebble backup and consistent reads** (`pebble/`) — `PebbleBackend.Checkpoint(dir)` for point-in-time DB snapshots and `NewSnapshot()` for consistent read views via Pebble snapshots
- **OTel baggage correlation IDs** (`otel/`) — `WithCorrelationID(ctx, id)` and `CorrelationIDFromContext(ctx)` propagate correlation IDs across distributed service boundaries via W3C baggage
- **OTel TextMapPropagator** (`otel/`) — `NewTextMapPropagator()` implements W3C trace context + baggage propagation for inject/extract across transports
- **OTel CQRS metric views** (`otel/`) — `NewCQRSViews()` configures customized histogram boundaries (`CQRSHistogramBoundaries`) for CQRS latency ranges; `ServiceResourceAttributes()` for service identification; `CounterAddWithAttributes()` and `AddSpanEvent()` helpers for rate metrics and span events
- **Decider load coalescing via singleflight** (`decider/`) — `Repository[State]` now coalesces concurrent `Load` calls for the same aggregate into one `store.Load` query. Events are immutable (`*ImmutableEvent`), so sharing the loaded slice is safe. Disable via `WithLoadCoalescing[State](false)`
- **HKDF key derivation** (`encryption/`) — `DeriveKey(masterKey, info, length)` derives per-tenant/subscope keys via HKDF-SHA256, enabling multi-tenant encryption without separate master keys
- **SQLite foreign keys helper** (`storage/`) — `SQLiteEnableForeignKeys(ctx, db)` enables `PRAGMA foreign_keys=ON` for opt-in referential integrity
- **Codec BufferEncoder interface** (`codec/`) — `BufferEncoder` extension enables zero-allocation encoding directly into a caller-provided `*bytes.Buffer` via `EncodeToBuffer(payload, buf)`, bypassing intermediate allocations
- **Event stream deduplication operators** (`event/`) — `DistinctByEventID()` suppresses duplicate event IDs; `DistinctByAggregateID()` keeps only the first event per aggregate. Composable via `ro.Pipe1`
- **Watermill middleware wrappers** (`watermill/`) — `CorrelationIDMiddleware()` and `NewRetryMiddleware(config)` for Watermill routers, plus Router integration support
- **CBOR streaming and compact codec docs** (`codec/`) — `CBORCompactCodec` documentation (struct fields as positional array, ~35% smaller payloads); `Diagnose()` for human-readable CBOR debugging
- **Testutil seed control** (`testutil/`) — seed control helper and rapid testing generator patterns for reproducible randomized tests

### Changed

- **Dependency upgrades** — `go-error-family` v0.3.0 → v0.4.0; `go-branded-id` v0.3.0 → v0.3.1 across all consuming modules
- **API surface growth** — 1266 → 1289 exports (29 new public symbols), golden file updated
- **Testutil ghost API removal** (`testutil/`) — removed non-functional `EventSlice` and `SeedFromEnv` exports (dead code that never worked; technically a public surface reduction but no behavioral impact)

### Fixed

- **Turso CheckpointScheduler race** (`turso/indexing/`) — `Stop()` now drains the checkpoint goroutine via a `done` channel before returning, preventing goroutine leaks and races on repeated Start/Stop cycles
- **Turso parallel test flakiness** (`turso/`) — eliminated flaky parallel test failures by isolating state and increasing checkpoint test timing margins
- **Decider singleflight error passthrough** (`decider/`) — singleflight errors now pass through verbatim instead of being wrapped with `fmt.Errorf`, preserving error classification (Rejection/Conflict/etc.) via `errors.Is`
- **OTel NewCQRSViews wildcard** (`otel/`) — corrected view instrument name wildcard matching so all CQRS histograms receive custom boundaries
- **Production dependency budget accuracy** (`scripts/check-module-layers.sh`) — test-only packages (gomega, ginkgo, rapid) now excluded from the production dep count, reflecting true direct dependency budgets

### Infrastructure

- **Watermill Router integration test** — end-to-end test for CorrelationID + Retry middleware through a real Watermill Router

## [2.4.0] - 2026-06-17

15 performance optimizations across 7 modules. No public API changes, no disk format changes, no breaking behavior. Verified with 5-run benchmark averages (allocation deltas are deterministic and reliable; ns/op has ±15% variance), tests + race detector + lint.

### Performance

- **Pebble double serialization eliminated** (`pebble/`) — events serialized once, `batch.Set` called for both event and journal keys. Halves CPU and disk bytes per write
- **Event lazy metadata map initialization** (`event/`) — `NewMetadata()` returns zero-value struct instead of always allocating a map. Eliminates 1 heap allocation per event when no custom metadata is set
- **Projection handler Lookup zero-allocation** (`projection/`) — `lookupSlices()` returns pre-built handler slices directly instead of allocating a combined slice per event. Only benefits `projection.Builder`-created projections
- **Projection Runner event type caching** (`projection/`) — Runner caches `p.EventTypes()` once at `Register()` time, eliminating 10.5M per-event clone allocations (100K events × 100 projections) in the scale benchmark. This is the real fix for the projection allocation hotspot — the original T3/T4 `*builtProjection` type assertion was dead code for `event.NewProjection()` users. Also pre-allocates the candidates slice in `dispatchToProjections`
- **SQL template strings cached per dialect** (`storage/`) — INSERT SQL built once at `SQLEventStore` construction, eliminating `fmt.Sprintf` per call
- **MemoryStore Load double-copy eliminated** (`memory/`) — removed redundant `slices.Clone` wrapper on already-fresh slice from `getEvents()`
- **SSE vestigial goroutine removed** (`middleware/`) — removed useless `go func() { <-ctx.Done() }()` goroutine leak. Consolidated 3× `fmt.Fprintf` into single write
- **Event Merge EnsureCustom hoisted** (`event/`) — `EnsureCustom` called once before the Merge loop instead of per-iteration nil-check
- **Event FilterByTimestamp pre-sized** (`event/`) — result slice initialized with `make([]Event, 0, len(events))` to eliminate nil-slice append growth pattern
- **SQL ScanSlice pre-allocated** (`storage/`) — initial capacity hint of 64 reduces log₂(N) slice growth copies during large Loads
- **CircuitBreaker atomic state machine** (`middleware/`) — replaced `sync.Mutex` + `int` fields with `atomic.Int32`. Happy path (circuit closed) is now lock-free: single `state.Load()` check
- **MemoryBus middleware pre-computation** (`memory/`) — middleware chains pre-computed at `Use()`/`UsePublish()` registration time. `Publish()` reads cached chain under RLock — zero per-publish closure allocation
- **Pebble ReadFrom key-based skip** (`pebble/`) — during cursor skip phase, parse event ID from journal key via `journalKeyEventID()` instead of CBOR-deserializing every skipped event
- **SQL multi-VALUES INSERT batching** (`storage/`) — single `INSERT INTO events ... VALUES (..), (..), (..)` statement replaces N individual INSERTs. SQLite 999-parameter limit handled via automatic chunking (99 events/batch)

### Added

- **Reactive CommandBus and QueryBus** (`command/`, `query/`) — `NewCommandBus`, `NewQueryBus`, `FilterCommandType`, `FilterQueryType`, `HandlerToObserver`, plus replay/behavior variants. Mirrors the existing reactive event API for command and query streams
- **PebbleBackend facade** (`pebble/`) — `Open()` and `NewBackend()` provide a single shared-DB entry point for Pebble-backed EventStore, SnapshotStore, and CheckpointStore, with clear ownership semantics
- **SQLBackend lifecycle facade** (`storage/`) — `SnapshotStore()`, `CheckpointStore()`, and `Close()` methods complete the SQL backend full-stack facade
- **KV module** (`kv/`) — Layer-0 in-memory key-value store abstraction (`MemStore`) with snapshot iteration and atomic batch commit
- **`command.Compose` and `query.Compose`** — re-export `go-error-family.Compose` for classified multi-error composition in command and query modules
- **Integration tests** (`integration/`) — end-to-end tests for pebble-backed projection Runner (replay + live) and decider Repository with Pebble SnapshotStore
- **Pebble KV Store adapter** (`pebble/`) — `NewKVStore()` wraps `*pebble.DB` as `kv.Store`, making pebble the first real consumer of the kv/ abstraction. Supports owned and borrowed DB lifecycle, prefix-bounded iteration, atomic batch commit, and `ErrNotFound`/`ErrClosed` error mapping
- **Built-in pprof endpoints** (`middleware/`) — `ProfilingHandler()` and `RegisterProfiling()` expose Go runtime profiling (heap, goroutine, CPU, allocs, block, mutex) via standard `/debug/pprof/` paths
- **Pebble benchmarks** (`pebble/`) — 4 benchmarks (Save100, SaveLoad100, Save1, LoadEmpty) for performance regression tracking
- **KV contract tests** (`pebble/`) — 10-test contract suite run against both PebbleAdapter and MemStore, proving semantic equivalence
- **Compose tests** (`command/`, `query/`) — 5 tests each for `Compose` error composition (nil, single, multiple, classified, mixed)
- **PostgreSQL CI** (`.github/workflows/ci.yml`) — `postgres-integration` job with PostgreSQL 16 service container wired to storage integration tests

### Fixed

- **Turso error classification** (`storage/sql/query_engine.go`) — `QueryRows` no longer re-wraps classified errors as Infrastructure, preserving Rejection semantics for `LoadNonExistent`
- **Module layer budgets** (`scripts/check-module-layers.sh`) — budgets updated to reflect actual direct dependencies: codec 2, pebble 8, storage 11, turso 10, integration 19
- **Turso lint hygiene** (`turso/indexing/advisor_data.go`) — cleared 3 pre-existing `gochecknoglobals` findings on static advisor data tables

### Infrastructure

- **CI replace-directives check** — `scripts/check-replace-directives.sh` now runs in GitHub Actions to verify every module `replace` directive matches `go.work`
- **`cmd/api-stability` in CI matrix** — per-module-test job now tests the API stability checker in isolation

## [2.3.0] - 2026-06-12

231 commits since v2.2.0. Lint hygiene, coverage improvements, CBOR codec, encryption module, phantom types, and release readiness.

### Added

- **CBOR codec** (`codec/`) — `CBORCodec` with deterministic canonical encoding, sorted map keys, `DecMode` option
- **Pebble CBOR envelope** (`pebble/serialization.go`) — events serialized as CBOR with JSON backward compatibility layer
- **Encryption module** (`encryption/`) — XChaCha20-Poly1305, AES-256-GCM, `Algorithm` enum, `KeyID` phantom type, `KeyResolver` interface, composable `NewCodec` wrapper, `EncryptMiddleware`/`DecryptMiddleware`
- **Command store interfaces** (`command/`) — `CommandSink`, `CommandSource`, `Store` (Sink+Source) for persisted command logs
- **SQL CommandStore** (`storage/`) — `SQLCommandStore` with Save, AppendBatch, Load, LoadFromTimestamp, LoadToTimestamp
- **SQL Backend facade** (`storage/`) — `SQLBackend` returning EventStore, SnapshotStore, CheckpointStore, CommandStore
- **Phantom types** across library modules — `DbPath`, `RemoteURL`, `AuthToken` (turso); `KeyID` (encryption); `Algorithm` (encryption); `DisplayID` (catalog); type-safe domain IDs in examples
- **Event binary blob helpers** (`event/`) — `AttachBlob`, `ExtractBlob`, `HasBlob` for signing/encryption
- **`command.TypedHandler[Q, R]`** with `RegisterTyped[Q, R]` — type-safe command handler
- **`event.DecodePayloads[T]()`** — batch payload deserialization
- **Listing table schema** (`storage/`) — DDL + repository for aggregate status persistence
- **ADR-0008 through ADR-0015** — 8 new architecture decision records (TypedHandler, immutability, OTel re-exports, error taxonomy, CBOR, encryption, saga, config)
- **ADR index** (`docs/adr/README.md`) — complete index of all 15 ADRs with titles, dates, status
- **Comprehensive fuzz testing** — fuzz tests in codec, encryption, signing/multisig, integration
- **Property-based tests** — `pgregory.net/rapid` in command, query, event, decider, id modules
- **go-snaps snapshot tests** — catalog, integration, projection golden test coverage
- **Benchmark infrastructure** — realistic scale benchmarks, fuzz benchmarks, multisig concurrent benchmarks
- **gosec security scanning** in CI with SARIF upload
- **Module layer check** — `.go-arch-lint.yml` architecture rules enforced in CI
- **17 scale benchmarks** across modules (10K–1M events)
- **`pkg/config/`** — YAML config loader with env-specific overlays
- **`pkg/gracefulshutdown/`** — signal-aware shutdown with timeout and hook support
- **Docker packaging** for `example/user/` (multi-stage Dockerfile + docker-compose.yml)
- **SSE broker** (`middleware/sse.go`) — server-sent events over event bus
- **Health check middleware** (`middleware/healthcheck.go`) — `/health`, `/health/live`, `/health/ready`
- **Metrics HTTP handler** (`middleware/metrics_http.go`) — request count, error rate, avg response time
- **EventCatalog docserver** (`catalog/docserver/`) — embedded SPA with AsyncAPI + Scalar rendering
- **`integration/simulation/`** — event sequence generator + decider stress tests
- **Encryption integration** — end-to-end encrypt→sign→verify→decrypt round-trip tests
- **Test coverage:** storage/sql 37.4%→89.2%, otel 73.0%→97.3%, turso 26.8%→39.0%

### Changed

- **Pebble: migrated event envelope from JSON to CBOR encoding** — deterministic, compact binary format
- **Pebble: sharded mutex pool** (FNV-1a hash, 256 shards) replaces unbounded `sync.Map` — bounded memory, zero allocations
- **storage/sql: extracted generic `LoadWithSpan[T]` + `QueryRows[T]`** — eliminated event/command store load duplication
- **storage/sql: context-aware SQL methods** throughout — `BeginTx`, `ExecContext`, `QueryRowContext` (no more `noctx` lint)
- **storage/sql: `ClosableBase` extracted** — deduplicated store lifecycle boilerplate
- **OTel abstraction** — modules import `otel/` re-exports instead of `go.opentelemetry.io` directly (decider, storage, middleware, projection)
- **Error wrapping** — replaced `fmt.Errorf` wrapping classified errors with `WrapRejection`/`WrapCorruption` across memory, pebble, storage, listing
- **`command/command.go`** — added `Type.IsZero()`, `ParseType()`, `MustParseType()` to match `event.Type` API
- **`query/query.go`** — added `Type.IsZero()`, `ParseType()`, `MustParseType()` to match `event.Type` API
- **`event/types.go`** — `SchemaVersion.Cmp` now uses `cmp.Compare` (matches `Version.Cmp`)
- **`event/errors.go`** — doc comments on all 30 exported error symbols
- **`event/Clone()`** — deep-copies `eventOptions` pointer to prevent shared mutation
- **`event: Map/ScanState/Tap` reactive wrappers removed** (unused, no consumers)
- **`event: StreamKey` free function removed** (unused)
- **All 120 `//nolint` suppressions** now have documented `// reason` justifications
- **0 lint issues** across all 27 modules — first zero-lint release
- **`golang.org/x/exp`** bumped across all workspace modules
- **`storage/AggregateProjection`** uses `Dialect.Placeholder()` (Postgres-compatible)
- **`listing/AggregateRef` renamed to `AggregateListing`** with JSON tags
- **`catalog: ErrorExporter` deprecated** as type alias to `Exporter[error]`
- **`catalog: asyncapi.Info` and `openapi.Info` consolidated** into shared `DocumentInfo`
- **`snapshot: json tags`** added to `Snapshot` struct
- **Dissolved `core/` module** — all sub-packages are flat peer-level modules (v2.0.0, maintained in v2.3.0)
- **`event.Snapshot*` types moved to `snapshot/` package** — all consumers updated
- **`dispatcher/Lifecycle` field unexported** with method delegation added

### Fixed

- **SSE broker send-on-closed-channel race** — `handleEvent`/`RemoveClient` synchronization
- **SSE broker constructor** — `NewSSEBroker` now returns `(*SSEBroker, error)` instead of nil on error
- **Circuit breaker nil `IsFailure` guard** — defaults to `event.IsRetryable`
- **Circuit breaker error taxonomy** — `ErrCircuitBreakerOpen` uses error taxonomy instead of bare `errors.New`
- **Projection Runner double-wrapping classified errors** in `opError`
- **Projection Runner fresh done channel** per `Run` invocation
- **Projection Runner `Close()`** now waits for `Run` to complete
- **Clone shared opts pointer** — deep-copy `eventOptions` prevents shared mutation
- **Retry middleware** — `ErrRetryCanceled` sentinel actually used on context cancellation
- **Pebble `NewStore(nil, ...)` panics** with clear message instead of nil pointer dereference
- **Pebble `countEvents` uses `iter.Last()`** instead of full scan
- **Pebble `MarshalMetadataJSON` error** — handled instead of discarded
- **Decider `slog.WarnContext` fallback** for snapshot failures (previously OTel-only)
- **Multiple lint issues** — nlreturn, varnameld, noctx, errcheck, unconvert, nolintlint
- **`event.NewMetadata`** now initializes `Custom` map
- **`dispatcher/Lifecycle`** field unexported, added method delegation
- **`event: renamed `WithNewCodec`→`WithCodec`** (kept deprecated alias)
- **Config loader path traversal** — `filepath.Clean` sanitizes paths (gosec G304)
- **Graceful shutdown select guards** on errCh sends to prevent panic

### Performance

- **`catalog.SchemaFromType` cached by `reflect.Type`** — 553ns→8ns, 15→0 allocs
- **`event.New()` lazy-initializes metadata map** — 3→2 allocs per event
- **`event.New()` moves clock/newCodec/deadline to `eventOptions` pointer** — 48B saved per event
- **`event.PayloadReadOnly()` zero-copy** for internal paths (signing, pebble, storage, middleware)
- **`event.DecodePayload` bypasses `Payload()` clone** for zero-copy decoding
- **`listing` caches sorted aggregate index** — 25× faster listing
- **`memory` replaces O(n log n) `collectAllSorted`** with append-only global log
- **`signing.canonicalPayload()` eliminates alloc overhead**

### Security

- **gosec scanning** in CI with SARIF upload
- **Module layer check** enforced in CI
- **Config loader path traversal fix** (G304)
- **Constant-time ciphertext comparison** in encryption module

### Removed

- **`storage/options.go`** — deleted `NewSQLEventStoreWithOptions`, `WithOwnership`, `SQLEventStoreOption` (zero external consumers)
- **`storage/doc.go`** — removed 5 unused re-exports
- **`pebble/config.go`** — deleted entire config abstraction layer (`Backend`, `Config`, `NewConfig`, etc.)
- **`pebble/example_test.go`** — tested only deleted config API
- **`pebble/errors.go`** — removed `ErrPebbleProviderRequired`
- **`turso/errors.go`** — removed `ErrTursoMemorySync` backward-compat alias
- **All `MustParse`/`MustParseType` panic wrappers** removed from command, query, event test code
- **Deprecated backward-compat aliases** from `pebble/` module
- **Dead code and unused APIs** across multiple modules
- **`command/errors.go`** — removed unused `WrapTransient` re-export
- **`event/go.mod`** — removed `query/v2` direct dependency
- **`snapshot/go.mod`** — removed `memory/v2` dependency

## [2.2.0] - 2026-06-08

81 commits since v2.1.0. Operational readiness, testing rigor, and developer experience release.

### Added

- **Health check middleware** (`middleware/`) — `/health`, `/health/live`, `/health/ready` endpoints
- **Metrics HTTP handler** (`middleware/`) — request count, error rate, avg response time
- **SSE broker** (`middleware/`) — server-sent events over event bus with subscription management
- **Config loader** (`pkg/config/`) — YAML config with env-specific overlays
- **Graceful shutdown** (`pkg/gracefulshutdown/`) — signal-aware shutdown with timeout and hook support
- **Docker packaging** (`example/user/`) — multi-stage Dockerfile + docker-compose.yml
- **Production server example** (`example/user/server.go`) — operational endpoints demonstrating health, metrics, graceful shutdown
- **Property-based tests** (`decider/`, `event/`, `id/`) — `pgregory.net/rapid` for deterministic decide, version monotonicity, ULID validity
- **Snapshot tests** (`integration/`) — `go-snaps` for event JSON serialization, catalog exports
- **Simulation framework** (`integration/simulation/`) — event sequence generator + decider stress tests
- **Benchmark baseline** (`benchmark-baseline.txt`) — saved from all benchmarks for regression detection
- **Module READMEs** — 9 modules with usage and API surface documentation
- **Package doc.go** — 7 library modules with usage examples for pkg.go.dev
- **example_test.go** coverage — storage, otel, projection, watermill, schema, signing, snapshot, listing, pebble, turso, codec, dispatcher
- **docserver** (`catalog/docserver/`) — embedded EventCatalog SPA server with AsyncAPI + Scalar rendering

### Changed

- **Standardized flake configuration** — dev shell, test apps, benchmark apps unified
- **Command store split** — `storage/command_store.go` (387L → 3 focused files)
- **Snapshot errors extracted** — `snapshot/errors.go` with all sentinel errors
- **Projection replay refactored** — `loadReplayEvents` extracted (65L → 37L + 28L)
- **Dependencies bumped** — `golang.org/x/exp` across all workspace modules
- **Lint issues resolved** — all catalog, infrastructure, and pre-commit hook failures fixed

### Fixed

- **Catalog ToPascal byte underflow** — unicode boundary bug in case conversion
- **Duplicate package godoc** — removed from non-doc.go files in event, middleware, dispatcher
- **Broken example_test.go** — repaired in projection, schema, signing, watermill

### Security

- **gosec scanning** — Go security scanner integrated in CI with SARIF upload
- **Module layer check** — `.go-arch-lint.yml` architecture rules enforced in CI

## [2.1.0] - 2026-06-03

62 commits since v2.0.0. Performance-focused release with production bug fixes, new query types, and comprehensive benchmarking.

### Added

- `query.TypedHandler[Q Query, R any]` — typed query parameter + typed result via `RegisterTyped[Q, R]`
- `listing.CacheInvalidationMiddleware(reader)` — auto-invalidates `InMemoryAggregateReader` cache after publish
- `listing.CacheInvalidator` interface — decouples middleware from concrete reader type
- 17 scale benchmarks across event, memory, listing, storage, pebble, turso, watermill, and codec modules
- 6 new benchmark suites with `b.ReportAllocs` for allocation tracking
- `nix run .#bench` app and `benchstat-compare` script for regression detection
- Turso CRUD integration tests for event/snapshot/checkpoint stores
- Realistic scale benchmarks behind `-tags=scale` in integration module
- ADR-0008 for `TypedHandler[Q Query, R any]` dual type parameter signature
- `docs/STORAGE_GUIDE.md` — performance comparison across PostgreSQL/SQLite/Pebble/Turso backends

### Changed

- `MemoryStore` deduplicated event storage — single `globalLog` + `streamIndex` map of indices replaces per-stream event copies (2× memory reduction)
- `event.New()` inlined codec extraction — removed `findCodecOption` helper, fast path for empty opts avoids probe allocation
- `MemoryStore.ReadFrom` uses cursor-based pagination instead of linear scan
- `schema.VersionedStore` load methods deduplicated into shared `loadAndUpcast` helper
- Error wrapping migrated to `event.Wrap*` taxonomy across storage, watermill, command, query, schema, and listing
- Deprecated backward-compat aliases removed from `pebble/` module
- Dead code removed + Go idioms modernized across multiple modules
- `event.Metadata()` documented as returning a defensive copy

### Performance

- `catalog.SchemaFromType` cached by `reflect.Type` — 553ns→8ns, 15→0 allocs
- `event.New()` lazy-initializes metadata map — 3→2 allocs per event
- `event.New()` moves clock/newCodec/deadline to `eventOptions` pointer — 48B saved per event
- `event.Payload()` removes defensive clone — 1 fewer alloc per access
- `event.New()` skips redundant payload copy — 1 fewer alloc
- `event.New()` stamps encoding directly — 1 fewer alloc
- `signing.canonicalPayload()` eliminates alloc overhead
- `listing` caches sorted aggregate index — 25× faster listing
- `memory` replaces O(n log n) `collectAllSorted` with append-only global log

### Fixed

- HealthCheck OOM on large event stores
- `SQLAggregateReader` Postgres compatibility
- `SubscriberAdapter` race condition
- Pebble `Close` not releasing resources
- `Version.Sub` panic on zero value
- `codec.Raw` passthrough encoding
- `GetID` rename consistency
- `ToAny` error propagation
- `HasSignature` false negatives
- `errgroup` error propagation
- `projection.Runner` missing `ErrAlreadyRunning` guard
- `storage` closed state tracking, snapshot SQL filter, `createTable` context
- `subscribeLive` handler guard for nil handlers
- `eventtest.FakeStore` ReadFrom test for sorted ReadAll output

### Removed

- Deprecated backward-compat aliases from `pebble/` module
- Dead code and unused APIs across multiple modules

## [2.0.0] - 2026-06-01

### Added

- `schema/` module — Upcaster, UpcasterRegistry, VersionedSource for schema evolution (extracted from event/)
- `snapshot/` module — Snapshot, SnapshotStore, SnapshotStrategy, helpers, error sentinels (extracted from event/)
- `samber/ro` integration in `event/reactive.go` — EventBus, NewReplayEventBus, NewBehaviorEventBus, FilterEventType/Types, ReplayFilter, HandlerToObserver/WithContext, Map, ScanState, Tap, Observable type alias
- `samber/ro` integration in `command/reactive.go` — CommandBus, FilterCommandType, Observable type alias
- `samber/ro` integration in `query/reactive.go` — QueryBus, FilterQueryType, Observable type alias
- `event/reactive.go` uses context-aware `ro.NewObserverWithContext` API — handler errors terminate the observer via `ErrorWithContext`
- `projection/runner.go` replay uses direct loop filters (`filterByEventTypes`, `filterFromCheckpoint`) instead of ro.Pipe1/ro.Collect overhead — projection no longer depends on `samber/ro`
- `listing/` module added to flake.nix testModules
- `otel/`, `pebble/`, `turso/`, `codec/` modules added to flake.nix testModules

### Changed

- **Dissolved `core/` module** — All 8 sub-packages (event, command, query, decider, id, dispatcher, schema, snapshot) are now flat peer-level modules. Import paths changed from `go-cqrs-lite/core/{pkg}` to `go-cqrs-lite/{pkg}`.
- `event.Snapshot*` types moved to `snapshot/` package — all consumers updated (decider, memory, storage, testhelpers)
- `event.ErrSnapshotNotFound` / `event.ErrSnapshotStoreClosed` moved to `snapshot/store.go`
- `memory/snapshot.go` uses `snappkg` alias to avoid local variable shadowing
- Removed duplicate `EventHandler` type from `event/reactive.go` (identical to `Handler`)
- AGENTS.md fully rewritten with new monorepo structure, dependency graph, key patterns
- Removed self-referencing replace directives (`module => ./`) from 6 go.mod files

### Removed

- `command/reactive.go` — temporarily deleted (restored in this release)
- `event/reactive.go` — restored with context-aware ro API (NewObserverWithContext + ErrorWithContext)
- `core/` directory — all sub-packages promoted to workspace root
- `event.Context() context.Context` — Go anti-pattern removed; use `Event.Deadline()` instead
- `event/context.go` — `deadlineCtx` type deleted (only used by removed `Context()`)

### Fixed

- `flake.nix` now includes all library modules in testModules
- `go.work.sum` stale references cleaned via `go work sync`

### Added

- `event.DecodePayloads[T]()` batch decode helper for processing multiple events at once
- `middleware.WithLogger(*slog.Logger)` option for retry, recovery, and validation middleware
- `storage/tables.go` — 5 table name constants replacing inline SQL strings
- `dispatcher.LifecycleMixin` embedded in `memory/checkpoint` and `memory/outbox`
- Concurrent access tests for MemoryBus, MemoryStore, MemoryOutbox, MemoryCheckpoint, MemorySnapshot
- `CONTEXT.md` — Domain glossary (aggregate, decider, event, fold, projection, saga)
- `docs/adr/` — ADR-0001 (Decider), ADR-0002 (Error taxonomy), ADR-0003 (Multi-module monorepo)
- `docs/ARCHITECTURE_PATTERNS.md` — Time-travel API, state-is-disposable, determinism, versioned events
- `docs/STORAGE_GUIDE.md` — PostgreSQL/SQLite/Pebble/Turso backends, event store operations

### Changed

- `AGENTS.md` trimmed from 384→121 lines (all essential info preserved)
- TODO_LIST.md reconciled: 40+ stale items verified as already done

### Fixed

- `storage/sql_base.go` bare `%w` wrapping → direct sentinel error return
- LSP hints: `sync.WaitGroup.Go` simplification, `fmt.Appendf` replacing `[]byte(fmt.Sprintf(...))`
- `projection/filterEvents` optimized from O(n×k) to O(n+k) via typeSet map

## [1.0.0] - 2026-05-26

### Added

- **saga** — Saga / Process Manager with compensation, retry, and timeout support
- **watermill** — Watermill message bus adapter with metadata-based event serialization
- **stream loading** — Memory-efficient `EventStream` + `StreamLoader` iterator pattern
- **event versioning** — `VersionedStore` with registered `Upcaster`s for transparent legacy event upcasting
- Full CQRS pipeline integration test (Command → Decider → Store → Bus → Projection → Query → Stream)
- Watermill metadata protocol: 15 metadata keys preserving all event fields

### Changed

- Eventcatalog coverage: 85.7% → 92.8%
- Saga coverage: 70.5% → 93.8%
- Watermill coverage: 28.6% → 89.6%
- `go.work` expanded to 13 modules

### Fixed

- Watermill `toEvent` used broken `json.Unmarshal` into `ImmutableEvent` — replaced with metadata reconstruction

## [0.2.0] - 2026-04-05

### Added

- **Event catalog system** (`catalog/`): Three-layer architecture with reflection-based schema generation, custom YAML marshaler, AsyncAPI and EventCatalog exporters
- **SnapshotStrategy** (`core/event`): Canonical interface and `EveryNEvents(n)` extracted to `core/event/snapshot_strategy.go`
- **Publisher/Subscriber ISP** (`core/event`): Sub-interfaces extracted from `event.Bus` for Interface Segregation
- **Error classification** via `event.RegisterClassification()` in `init()` for aggregate, projection, storage sentinels
- **PublishChanges / SaveSnapshot** (`core/event`): Shared functions eliminating duplication in aggregate/decider repositories
- **Strong ID migration**: 62 bare `string`/`int` violations replaced with named types (`OperationID`, `NodeID`, `ServiceID`, `DomainID`, etc.)
- **Dialect tests** (`storage`): 15 tests for PostgresDialect, SQLiteDialect, `placeholders()`
- **OpenAPI coverage tests** (`catalog/openapi`)
- **Performance benchmarks**: 43 benchmarks across 12 files
- **Design documents**: Outbox transaction API, query handler generics, saga design

### Changed

- **ISP activation**: Repositories accept `Publisher`, projections accept `Subscriber` (backward-compatible)
- Root go.mod module path: `github.com/LarsArtmann/go-cqrs-lite` (consistent casing)
- Zero lint issues across all 8 linted modules (was 50+)
- File splits: all files under 250 lines
- `outboxEvent` fields: `Version`/`SchemaVersion` changed from bare `int` to strong types
- `gomodguard` → `gomodguard_v2`

### Fixed

- All linter issues resolved: exhaustruct, gosec G201, tagliatelle, wrapcheck, noinlineerr, prealloc, goconst, fatcontext
- `FakeSnapshotStore.Save` now records snapshots for verification (was no-op)
- Dispatcher lifecycle: `Register()` and `Dispatch()` on closed dispatcher return errors correctly

## [0.1.0] - 2026-01-01

### Added

- Initial release with core CQRS infrastructure (command, event, query dispatchers)
- Event sourcing with `Store`, `Bus`, `SnapshotStore` interfaces
- In-memory implementations (`memory/` module)
- Branded IDs via `go-branded-id`
- Middleware: logging, retry, recovery, validation
- Test helpers for fakes and mocks
