# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/).

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
