# TODO List

**Audited:** 2026-05-20 · **Session 82**
**Sources:** Session 78-79 Execution Plan, Session 82 Implementation, Full Codebase Analysis

---

## 🔴 CRITICAL (Correctness Bugs)

- [x] ~~**Pebble Store: optimistic concurrency check**~~ — Fixed in Session 77
- [x] ~~**Retry middleware timer leak**~~ — Already fixed (defer timer.Stop present)
- [x] ~~**Aggregate snapshot with nil codec**~~ — Fixed in Session 78
- [x] ~~**example/todo build broken**~~ — Fixed in Session 82 (added replace directives for core/memory/testhelpers, fixed MarshalJSON infinite recursion in queries + commands, fixed test assertion)

---

## 🟠 HIGH Priority (Data Safety & Observability)

- [x] ~~**Pebble corrupt ID warnings**~~ — Added slog.Warn in Session 78
- [x] ~~**Pebble iterateEvents silently skips corrupt events**~~ — Fixed in Session 82 (returns error instead of silently continuing)
- [x] ~~**OutboxPublisher.publishPending swallows all errors**~~ — Already logs with slog.Warn (verified in Session 82)
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
- [x] ~~**Move schema DDL onto Dialect interface**~~ — Session 82: EventSchema/SnapshotSchema/CheckpointSchema/OutboxSchema now on Dialect; free functions delegate
- [ ] **Unify ErrNilBus sentinels** — 3 independent sentinels (intentional per-package context)
- [x] ~~**Add clock injection to NewEvent**~~ — Already injectable via `WithOccurredAt(time.Time)` option
- [x] ~~**Split storage helper files**~~ — Session 82: extracted `deserializeMetadata` from `deserializeEvent`

---

## 🟢 LOW Priority (Nice-to-Have)

- [x] ~~**Consolidate CatalogMeta**~~ — Intentional per-package context (event has extra AggregateType field)
- [ ] **Add BDD tests for catalog, storage, sync**
- [x] ~~**VectorClock.Compare returns enum**~~ — Session 82: added `Cmp()` returning `ClockOrder` enum (Before/After/Concurrent/Equal); `Compare()` deprecated
- [ ] **Implement Saga/Process Manager** — design done, implementation pending
- [ ] **PostgreSQL integration tests for storage** — unit tests use go-sqlmock only
- [x] ~~**Standardize logger injection across all modules**~~ — Already standardized (constructors accept \*slog.Logger, optional via functional options)
- [ ] **Add `event.Context` propagation to time.Now() calls**

---

## ✅ COMPLETED (Session 82)

- [x] **example/todo build fix** — replace directives for core/memory/testhelpers, MarshalJSON recursion fix, test assertion fix
- [x] **Pebble iterateEvents corrupt event handling** — returns error instead of silently skipping
- [x] **Zero lint across all 8 modules** — fixed embeddedstructfieldcheck, godot, unparam, exhaustruct, godoclint, varnamelen, errchkjson, prealloc, noinlineerr
- [x] **DDL on Dialect interface** — EventSchema/SnapshotSchema/CheckpointSchema/OutboxSchema on Dialect
- [x] **Pebble deserializeEvent split** — extracted `deserializeMetadata` helper
- [x] **VectorClock.Cmp() with ClockOrder enum** — Before/After/Concurrent/Equal with String() method
- [x] **All tests pass (27 main + 7 example/todo packages)**

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
