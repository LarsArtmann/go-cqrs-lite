# TODO List

**Audited:** 2026-05-19 · **Session 74**
**Sources:** Code Quality Scan, Features Audit, BDD Testing Analysis, Full Code Review, Architecture Improvement, Architecture Review, Go Modularize, Architecture Visualization

---

## 🔴 CRITICAL (Correctness Bugs)

- [ ] **Pebble Store: No optimistic concurrency check in Save** — `storage/pebble_event_store.go:48`
  - `Save` validates versions but doesn't check existing stream length matches `expectedVersion`
  - Concurrent writes silently overwrite each other (unlike `MemoryStore.Save` which checks)
  - Files: `storage/pebble_event_store.go:48-102`

- [ ] **Retry middleware timer leak** — `middleware/retry.go:104`
  - `time.NewTimer` created per retry but not stopped in the normal path
  - Add `defer timer.Stop()` before the select statement
  - Files: `middleware/retry.go:86-122`

- [ ] **Aggregate snapshot with nil state** — `core/aggregate/load_helpers.go:93-122`
  - `trySnapshot` creates snapshot with nil state when `codec` is nil
  - Should skip snapshot save entirely when codec is nil
  - Files: `core/aggregate/load_helpers.go`, `core/decider/load.go`

- [ ] **testhelpers v1.1.0 version mismatch** — published tag incompatible with current core
  - Published `testhelpers v1.1.0` uses `int` for version, core now requires `event.Version`
  - `core/event`, `core/aggregate`, `core/decider` fail to build in isolation (GOWORK=off)
  - Fix: Bump testhelpers to v1.2.0 with `event.Version` params

- [ ] **example/todo build broken** — stale storage/core API references
  - `SaveWithOutbox` signature changed (added `Outbox` param)
  - `event.SchemaVersion` undefined in storage helpers
  - `event.Version` type mismatch in outbox
  - Files: `example/todo/`, `storage/`

---

## 🟠 HIGH Priority (Data Safety & Observability)

- [ ] **Pebble deserialization silently discards 4 ID parsing errors** — `storage/pebble_serialization.go:76-88`
  - Corrupt correlation/causation/user/request IDs become zero-value with no warning
  - Add log warnings for corrupt metadata

- [ ] **Pebble iterateEvents silently skips corrupt events** — `storage/pebble_event_store.go:120-123`
  - Produces incomplete results with no error return
  - Should return error or at minimum count/log skipped events

- [ ] **OutboxPublisher.publishPending swallows all errors** — `core/event/outbox_publisher.go:221`
  - `_ = p.pollPublishAck(ctx)` — zero observability for publish failures
  - Add slog.Warn for failed publishes

- [ ] **sync.NewLWWResolver allows nil TimestampFunc → panic** — `sync/conflict.go:40`
  - Add nil check with `panic("sync: TimestampFunc must not be nil")`

- [ ] **catalog.SchemaFromType panics on interface types** — `catalog/schema.go:25-29`
  - `reflect.TypeOf(zero)` returns nil for interface types → panic
  - Add nil type check before processing

- [ ] **Decider Execute has dual %w wrapping** — `core/decider/decider.go:113`
  - `fmt.Errorf("...: %w: %w", ErrSaveFailed, err)` — first error unreachable via `errors.As`
  - Use nested `fmt.Errorf` or `errors.Join`

- [ ] **projection.Runner.filterEvents is O(n)** — `projection/runner.go:211-237`
  - Scans entire event stream linearly to find checkpoint position
  - Add position-based loading to `GlobalLoader` interface

- [ ] **core depends on memory + testhelpers in production go.mod** — test deps listed as direct requires
  - Anyone importing core transitively gets ginkgo/gomega
  - Move to test-only requires or separate go.mod strategy

---

## 🟡 MEDIUM Priority (Architecture & Quality)

- [ ] **Unify aggregate/decider repository logic** — ~200 lines duplicated
  - Snapshot loading, save+publish, outbox branching duplicated in both packages
  - Extract shared `Repository[State]` or shared helpers in `core/event`
  - Files: `core/aggregate/load_helpers.go`, `core/decider/decider.go`, `core/decider/load.go`

- [ ] **Add catalog.Exporter interface + WalkMessages helper** — catalog exporters duplicate iteration
  - All 4 exporters follow identical service→message loop
  - Extract shared `WalkMessages(cat, fn)` in `catalog` package
  - Files: `catalog/asyncapi/builder.go`, `catalog/openapi/exporter.go`, `catalog/d2/exporter.go`, `catalog/eventcatalog/exporter.go`

- [ ] **Remove all replace directives from go.mod files** — use go.work only
  - 8 modules have both replace directives AND go.work entries
  - go.work makes replaces redundant — remove for clarity
  - Files: all go.mod files with replace blocks

- [ ] **Standardize version references across go.mod files** — inconsistent v0.0.0 vs v1.1.0
  - `catalog/go.mod`: `core v0.0.0` while `middleware/go.mod`: `core v1.1.0`
  - Standardize to v0.0.0 for development or consistent semver

- [ ] **Move schema DDL onto Dialect interface** — currently free functions
  - `Schema()`, `SQLiteSchema()`, `OutboxSchema()`, etc. should be Dialect methods
  - Cleaner extensibility for new SQL backends
  - Files: `storage/event_store.go`, `storage/snapshot.go`, `storage/outbox.go`

- [ ] **Unify error sentinels across aggregate/decider/projection** — same concept, 3 sentinels
  - `ErrNilBus` defined independently in aggregate, decider, projection
  - Define canonical sentinels in `core/event/errors.go`
  - Files: `core/aggregate/errors.go`, `core/decider/errors.go`, `projection/errors.go`

- [ ] **Add clock injection to NewEvent** — `time.Now()` not injectable
  - Accept `func() time.Time` via option for deterministic testing
  - Files: `core/event/event.go:226`

- [ ] **Fix event_test.go:396 golines lint** — only remaining lint issue
  - Single line too long in test file
  - Files: `core/event/event_test.go:396`

- [ ] **Split storage helper files** — `storage/pebble_serialization.go:47` `deserializeEvent` is 71 lines
  - Extract `deserializeMetadata`, `deserializeOptions` helpers
  - Files: `storage/pebble_serialization.go`

- [ ] **Delete deprecated catalog.go** — `core/event/catalog.go` is deprecated but still ships
  - Remove or mark as deprecated with clear migration path

- [ ] **projection.Runner.Register missing duplicate check** — unlike event.InMemoryRunner
  - Duplicate projection name processes events twice, causes checkpoint thrashing
  - Files: `projection/runner.go:66-74`

---

## 🟢 LOW Priority (Nice-to-Have)

- [ ] **Consolidate CatalogMeta** — duplicated in `event`, `command`, `query` packages
  - `event.CatalogMeta` has extra `AggregateType` field — not identical to command/query
  - Consider accepting duplication as intentional per-package types

- [ ] **Add BDD tests for catalog, storage, sync** — no ginkgo BDD coverage
  - Catalog E2E: register → build → export flow
  - Storage lifecycle: save → load → delete roundtrip
  - Sync: vector clock ordering + conflict resolution

- [ ] **VectorClock.Compare returns 0 for both equal and concurrent** — caller can't distinguish
  - Consider returning enum: Before, After, Equal, Concurrent
  - Files: `sync/vectorclock.go:43-74`

- [ ] **Implement Saga/Process Manager** — design done, implementation pending
  - Design doc: `docs/planning/SAGA_DESIGN.md` (4-phase plan, 18h estimate)

- [ ] **Tag `v0.1.0-alpha`** — first public release after verification

- [ ] **PostgreSQL integration tests for storage** — unit tests use go-sqlmock only

- [ ] **Create CONTRIBUTING.md** — architecture guidelines for contributors

- [ ] **Standardize logger injection across all modules** — inconsistent logging
  - `projection.Runner` accepts `WithLogger`, others use `slog.Default()`
  - Define `Logger` interface or standardize on `*slog.Logger` option

- [ ] **Add `event.Context` propagation to time.Now() calls** — outbox created_at not injectable
  - `outbox.Append` uses `time.Now()` — no way to control timestamp for testing

---

## ✅ COMPLETED (Previous Sessions)

- [x] **Implement `query.TypedHandler[T]`** — Session 54
- [x] **Implement `TransactionalStore`** — Session 55-56
- [x] **Middleware sentinel errors** — Session 54
- [x] **Replace `cockroachdb/errors` with stdlib** — Session 54
- [x] **Replace `go-json-experiment/json` with `encoding/json`** — Session 54
- [x] **Extract SnapshotStrategy to core/event** — Session 48
- [x] **ISP: Publisher/Subscriber sub-interfaces** — Session 48
- [x] **Error classification registration** — Session 48
- [x] **Fix all lint issues** — Session 48 (1 remaining as of Session 74)
- [x] **Test coverage gaps** — Session 48
- [x] **Extract shared PublishChanges + SaveSnapshot** — Session 48
- [x] **Add remaining benchmarks** — 43 benchmarks across 12 files (Session 50)
- [x] **Fix TODO_LIST.md false benchmark claim** — Session 50
- [x] **Fix FEATURES.md stale coverage numbers** — Session 50
- [x] **Design outbox transaction co-participation API** — Session 50
- [x] **Design query.Handler generics migration** — Session 50
- [x] **Review SAGA_DESIGN.md** — Session 50

---

## Files Read This Session

- [x] All 11 go.mod files
- [x] All production .go files in core/, memory/, catalog/, middleware/, projection/, storage/, sync/
- [x] All test files in core/event, core/aggregate, core/decider, projection, storage, sync
- [x] FEATURES.md
- [x] TODO_LIST.md
- [x] docs/planning/ (all files)
- [x] docs/status/ (all files)
- [x] docs/quality/ (created this session)
- [x] docs/architecture-understanding/ (created this session)
