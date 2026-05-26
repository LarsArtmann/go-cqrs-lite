# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/).

## [Unreleased]

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

- **SnapshotStrategy** (`core/event`): Canonical `SnapshotStrategy` interface and `EveryNEvents(n)` extracted from aggregate/decider to `core/event/snapshot_strategy.go`. Backward-compatible type aliases in `core/aggregate` and `core/decider`.
- **ShouldSnapshot helper** (`core/event`): Shared `ShouldSnapshot()` function in `core/event/snapshot_helper.go`.
- **Publisher/Subscriber ISP** (`core/event`): `event.Publisher` and `event.Subscriber` sub-interfaces extracted from `event.Bus`. Repositories accept `Publisher`, projections accept `Subscriber`.
- **Error classification registration** (`core/aggregate`, `projection`, `storage`): Sentinel errors registered via `event.RegisterClassification()` in `init()`.
- **ErrProjectionPanicked** (`core/event`): Sentinel error for panicking projection handlers.
- **Cross-module classification tests** (`integration/event`): Tests verifying aggregate, projection, and storage sentinels classify correctly.
- **PublishChanges** (`core/event`): Shared function for outbox/bus event dispatch. Eliminates duplication in aggregate and decider repositories.
- **SaveSnapshot** (`core/event`): Shared function for snapshot persistence with pre-encoded state.
- **Coverage tests**: New test files for memory (99.1%), projection (92.5%), storage (93.6%), aggregate (95.8%).
- **Performance benchmarks** (Session 50): `core/decider` (4 benchmarks), `projection` (3 benchmarks), `middleware` (4 benchmarks), `core/event` (6 benchmarks). Total: 43 benchmarks across 12 files.
- **Design documents** (Session 50): Outbox transaction co-participation API (`docs/planning/OUTBOX_TRANSACTION_API.md`), query handler generics migration (`docs/planning/QUERY_HANDLER_GENERICS.md`), saga design answers and implementation plan (`docs/planning/SAGA_DESIGN.md`).
- **Comprehensive execution plan** (Session 68): 86-task plan in `docs/planning/2026-05-18_15-10-COMPREHENSIVE-EXECUTION-PLAN.md`.
- **Strong ID migration** (Session 76): 62 bare `string`/`int` violations replaced with named types across `sync` and `catalog` modules.
  - `sync`: `OperationID`, `NodeID`, `SyncMessageType` named types for operation identity, vector clock keys, and message classification.
  - `catalog`: `ServiceID`, `DomainID`, `MessageID`, `ChannelID` named types for catalog entry identity.
  - All types have `String()`, `IsZero()`, `Parse*()`, `MustParse*()` methods with compile-time `fmt.Stringer` checks.
  - `catalog.MessageID()` function renamed to `GetID()` (returns typed `MessageID`). `MessageIDString()` deprecated.
  - API boundary pattern: public methods accept bare `string`, convert internally to typed IDs.
- **Dialect tests** (`storage`): 15 tests for PostgresDialect, SQLiteDialect, `placeholders()`.
- **OpenAPI coverage tests** (`catalog/openapi`): WithBasePath, nil schema, empty catalog, schemaToAny(nil), toKebab edge cases.

### Changed

- **ISP activation**: `aggregate.Repository` accepts `event.Publisher` (not `event.Bus`). `decider.Repository` accepts `event.Publisher`. `projection.Runner` accepts `event.Subscriber`. `event.OutboxPublisher` accepts `event.Publisher`. All backward-compatible — `event.Bus` satisfies both sub-interfaces.
- **Root go.mod module path**: `github.com/LarsArtmann/go-cqrs-lite` → `github.com/larsartmann/go-cqrs-lite` (consistent with sub-modules).
- **Zero lint issues** across all 8 linted modules (was 50+).
- **File splits** (Session 73): `catalog/openapi/exporter.go` → `convert.go`, `storage/event_store.go` → `event_store_scan.go`, `storage/outbox.go` → `outbox_helpers.go`, `catalog/docserver/docserver.go` → `builders.go`, `catalog/adapters/adapters_test.go` → `dispatcher_test.go` + `export_test.go`.
- **outboxEvent type safety**: `Version` and `SchemaVersion` fields changed from bare `int` to `event.Version`/`event.SchemaVersion`.
- **Golden test refresh**: Updated asyncapi.yaml, eventcatalog-config.js, package.json to match current output.
- **Zero catalog lint**: All gci and golines issues resolved.
- **SnapshotStrategy deduplication**: Removed 22-line duplicate from aggregate and decider; replaced with type aliases.
- `FakeSnapshotStore.Save` now records snapshots for verification (was no-op)
- Updated `dispatcher.Typed` documentation to clarify string-backed named types require explicit `string()` conversion
- **storage: PebbleBackendMemory** no longer pulls in `memory` module. Returns `ErrPebbleProviderRequired` — users must provide `WithPebbleProvider`.
- **File splits**: `storage/helpers.go`, `storage/pebble_event_store.go`, `catalog/asyncapi/exporter.go`, `catalog/registry.go` all under 250 lines.
- **Linter**: `gomodguard` → `gomodguard_v2` (deprecated linter replaced).

### Fixed

- **exhaustruct**: `snapshot.go` scanSnapshot now fills all Snapshot struct fields via parameter passing.
- **gosec G201**: `outbox.go` DELETE query uses parameterized placeholders (nolint with explanation).
- **tagliatelle**: `outboxEvent` JSON tags use snake_case matching DB column names (nolint).
- **wrapcheck**: `snapshot.go` wraps `sql.Row.Scan` error; `SaveSnapshot` return wrapped with `opError`.
- **noinlineerr**: 5 instances in `storage/` fixed by splitting inline error handling.
- **prealloc**: `helpers.go` preallocates options slice.
- **goconst**: Extracted `pollPendingQuery` constant in `storage/outbox.go`.
- **fatcontext**: Excluded for test files in `.golangci.yml`.
- **Codec integration** (`core/aggregate`): `WithCodec()` option on `EventSourcedRepository` for automatic snapshot serialization via `event.Codec`.
- **DecodePayload[T]** (`core/event`): Generic `DecodePayload[T any](evt Event, codec Codec) (T, error)` for type-safe event payload deserialization in handlers and projectors.
- **ContextEnricher** (`core/event`): `ContextEnricher` type and `CompositeEnricher` for extracting metadata from context and injecting into events before persistence.
- **Projection system** (`core/event`): `Projection` interface, `ProjectionFunc` convenience type, and `InMemoryRunner` that dispatches events to projections with checkpoint tracking. Can be used as `Bus.SubscribeAll` handler.
- **CheckpointStore** (`core/event` + `memory`): Interface for tracking last processed event per projection. `MemoryCheckpointStore` in memory module.
- **Upcaster system** (`core/event`): `Upcaster` interface, `UpcasterFunc` convenience type, and `UpcasterRegistry` with sorted chain application for schema evolution.
- **SQL Event Store** (`storage`): New module with `SQLEventStore` implementing `event.Store` with optimistic concurrency, transactional Save, PostgreSQL-optimized schema DDL.
- **Example app** (`example/user`): Working User aggregate demonstrating full CQRS + Event Sourcing lifecycle.
- **FakeSnapshotStore** (`testhelpers`): `Saved()` method and `SetSaveError` for testing snapshot creation and failure paths.
- **Exported fakes** (`testhelpers`): `FakeStore`, `FakeBus`, `FakeSnapshotStore`, `FakeOutbox` — reusable test doubles for all modules.

## [0.2.0] - 2026-04-05

### Added

- **Event catalog system** (`catalog/`): Three-layer architecture with reflection-based schema generation, custom YAML marshaler, AsyncAPI and EventCatalog exporters. Packages: `catalog/`, `catalog/asyncapi/`, `catalog/eventcatalog/`, `catalog/yaml/`
- **ID type methods**: `Equal`, `Compare`, `Or`, `Reset`, `GoString`, `Format` (fmt.Formatter), `MarshalBinary`/`UnmarshalBinary`, `MarshalText`/`UnmarshalText`
- **Comprehensive test suites**: `internal/dispatcher/` (0%→100%), `pkg/id/` (48%→88%), `aggregate/` (64%→100%), `event/` (75%→92.8%), `catalog/` (91.9%), `catalog/asyncapi/` (92.6%)
- **EventBuilder migration**: Moved from deleted `xtypes/` module to `core/event.Builder` with fluent API
- **EventCatalog frontmatter**: `schemaPath` in message frontmatter when schema exists; `sends`/`receives` arrays in service frontmatter
- **AsyncAPI functional options**: `WithServer()` and `WithDescription()` for configurable server name, host, and protocol
- **Integration test**: full E2E flow (Registry → Build → AsyncAPI + EventCatalog export)
- **Benchmarks**: `Registry.Build`, `SchemaFromType`, AsyncAPI `Export`/`MarshalYAML`, EventCatalog `Export`

### Changed

- **YAML marshaler**: struct fields now preserve definition order instead of alphabetical sorting (maps still sorted for determinism)
- **AsyncAPI**: `toSnakeCase` renamed to `toDotAddress` for truthful naming (produces `dot.separated`, not `snake_case`)
- JSON serialization: zero-value IDs now marshal to `null` instead of empty string; `UnmarshalJSON` supports both `null` and string values
- Dispatcher closed-check now returns `ErrHandlerNotFound` with descriptive wrapping instead of silently returning `nil`

### Fixed

- EventCatalog: `schemaPath` now correctly references `schemas/schema.json` in message frontmatter
- EventCatalog: service frontmatter now includes `sends`/`receives` message lists (required for EventCatalog rendering)
- YAML marshaler: struct field ordering preserved (was sorting alphabetically)
- CI workflows: updated Go version matrix from 1.21/1.22/1.23 to 1.26
- Dispatcher lifecycle: `Register()` and `Dispatch()` on closed dispatcher now correctly return errors (was no-op due to `CheckClosed(nil)`)
- Lint: replaced `[]byte(fmt.Sprintf(...))` with `fmt.Appendf`, used tagged switch on `msg.Kind`

### Infrastructure

- **GitHub Actions CI**: Created `.github/workflows/test.yml` and `.github/workflows/lint.yml` with Go 1.26 version matrix
- **Parallel tests**: Added `t.Parallel()` to all test functions across the codebase

## [0.1.0] - 2026-01-01

### Added

- Initial release
