# TODO List

**Audited:** 2026-05-20 · **Session 79**
**Sources:** Session 78 Execution Plan, Session 79 Implementation, Full Codebase Analysis

---

## 🔴 CRITICAL (Correctness Bugs)

- [x] ~~**Pebble Store: optimistic concurrency check**~~ — Fixed in Session 77 (already existed)
- [x] ~~**Retry middleware timer leak**~~ — Already fixed (defer timer.Stop present)
- [x] ~~**Aggregate snapshot with nil codec**~~ — Fixed in Session 78
- [ ] **example/todo build broken** — stale storage/core API references
  - `SaveWithOutbox` signature changed, `event.SchemaVersion` undefined, `event.Version` type mismatch
  - Files: `example/todo/`

---

## 🟠 HIGH Priority (Data Safety & Observability)

- [x] ~~**Pebble corrupt ID warnings**~~ — Added slog.Warn in Session 78
- [ ] **Pebble iterateEvents silently skips corrupt events** — `storage/pebble_event_store.go`
  - Produces incomplete results with no error return
- [ ] **OutboxPublisher.publishPending swallows all errors** — `core/event/outbox_publisher.go`
  - `_ = p.pollPublishAck(ctx)` — zero observability for publish failures
- [x] ~~**sync.NewLWWResolver nil TimestampFunc guard**~~ — Already has nil check + panic
- [x] ~~**projection.Runner.Register duplicate check**~~ — Added in Session 78
- [ ] **Decider Execute dual %w wrapping** — `core/decider/decider.go:113`
  - Works in Go 1.20+ (multi-unwrap). Low priority.
- [x] ~~**projection.Runner.filterEvents O(n)**~~ — Added PositionalLoader interface in Session 79
- [x] ~~**core depends on memory + testhelpers in production go.mod**~~ — Fixed (test deps are indirect)

---

## 🟡 MEDIUM Priority (Architecture & Quality)

- [x] ~~**Unify aggregate/decider repository logic**~~ — Shared PublishChanges + SaveSnapshot in core/event (Session 48)
- [x] ~~**Add WalkMessages helper**~~ — Added `catalog.WalkMessages` in Session 79
- [ ] **Remove replace directives from go.mod files** — BLOCKED: required until modules are tagged
- [ ] **Standardize version references across go.mod files** — inconsistent v0.0.0 vs v1.1.0
- [ ] **Move schema DDL onto Dialect interface** — currently free functions
- [ ] **Unify ErrNilBus sentinels** — 3 independent sentinels (intentional per-package context)
- [ ] **Add clock injection to NewEvent** — `time.Now()` not injectable
- [ ] **Split storage helper files** — `pebble_serialization.go` deserializeEvent is 71 lines

---

## 🟢 LOW Priority (Nice-to-Have)

- [ ] **Consolidate CatalogMeta** — duplicated in event/command/query packages
- [ ] **Add BDD tests for catalog, storage, sync**
- [ ] **VectorClock.Compare returns enum** — Before/After/Equal/Concurrent
- [ ] **Implement Saga/Process Manager** — design done, implementation pending
- [ ] **PostgreSQL integration tests for storage** — unit tests use go-sqlmock only
- [ ] **Standardize logger injection across all modules**
- [ ] **Add `event.Context` propagation to time.Now() calls**

---

## ✅ COMPLETED (Sessions 77-79)

- [x] **command.TypedHandler[T] + RegisterTyped** — Session 78
- [x] **event.NewEvents / MustNewEvents** — Session 78
- [x] **event.DecodePayloads[T]** — Session 78
- [x] **event.NewTypedProjection[T]** — Session 78
- [x] **Duplicate projection check** — Session 78
- [x] **Pebble corrupt ID warnings** — Session 78
- [x] **event.Store: LoadToVersion, LoadToTimestamp** — Session 79
- [x] **event.PositionalLoader interface** — Session 79
- [x] **catalog.WalkMessages helper** — Session 79
- [x] **Getting-started README section** — Session 78
- [x] **docs/MIGRATION.md** — Session 79
- [x] **CONTRIBUTING.md** — Session 79
- [x] **example/user updated to TypedHandler + DecodePayload** — Session 79
- [x] **All lint issues fixed (0 issues across 8 modules)** — Session 79
- [x] **All tests pass (24/24 packages)** — Session 79

## ✅ COMPLETED (Earlier Sessions)

- [x] **query.TypedHandler[T]** — Session 54
- [x] **TransactionalStore** — Session 55-56
- [x] **Middleware sentinel errors** — Session 54
- [x] **Replace cockroachdb/errors with stdlib** — Session 54
- [x] **Replace go-json-experiment/json with encoding/json** — Session 54
- [x] **Extract SnapshotStrategy to core/event** — Session 48
- [x] **ISP: Publisher/Subscriber sub-interfaces** — Session 48
- [x] **Error classification registration** — Session 48
- [x] **Test coverage gaps** — Session 48
- [x] **Extract shared PublishChanges + SaveSnapshot** — Session 48
- [x] **43 benchmarks across 12 files** — Session 50
