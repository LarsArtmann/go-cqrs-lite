# TODO List

**Audited:** 2026-05-21 · **Session 84**
**Sources:** Full Architecture Review, Naming Review, Codebase Deep Analysis

---

## 🔴 CRITICAL (Correctness Bugs)

- [x] ~~**Pebble Store: optimistic concurrency check**~~ — Fixed in Session 77
- [x] ~~**Retry middleware timer leak**~~ — Already fixed (defer timer.Stop present)
- [x] ~~**Aggregate snapshot with nil codec**~~ — Fixed in Session 78
- [x] ~~**example/todo build broken**~~ — Fixed in Session 82

---

## 🟠 HIGH Priority (Architecture & API Quality)

- [ ] **Formally deprecate `aggregate` package** — Add deprecation notice; ADR-0001 recommends `decider`. Package has 70% structural overlap with `decider` (Session 84 finding).
- [ ] **Remove deprecated catalog API (21 exports)** — `Catalogable`, `CatalogMeta`, `CatalogCore`, `MustNewCatalogCore` in `event`, `command`, `query` packages. Also `catalog/adapters.CatalogBuilder` and `MessageIDString`. All superseded by zero-cost `catalog.Command[T]()` API.
- [ ] **Extract `middleware/tracing` to separate module** — `tracing.go` depends on `go.opentelemetry.io/otel`, forcing the transitive dependency on all middleware consumers. Should be `middleware/tracing/` sub-module.
- [ ] **Unify `ErrNilBus` sentinels** — 3 independent sentinels (intentional per-package context)
- [ ] **Decider Execute dual %w wrapping** — `core/decider/decider.go:113`. Works in Go 1.20+ (multi-unwrap). Low priority.

---

## 🟡 MEDIUM Priority (Consistency & Naming)

- [ ] **Remove replace directives from go.mod files** — BLOCKED: required until modules are tagged
- [ ] **Standardize version references across go.mod files** — inconsistent v0.0.0 vs v1.1.0
- [ ] **Fix inconsistent `memory/` constructor naming** — `NewMemoryBus()` vs `NewCheckpointStore()` (no "Memory" prefix). Should be `NewMemoryCheckpointStore()`.
- [ ] **Rename `sync` package** — Shadows stdlib `sync`, requiring import aliases. Consider `syncx` or `crdt`.
- [ ] **Consolidate HandlerRegistry + MemoryBus dispatch** — Both independently implement type-specific + wildcard handler dispatch pattern. Should share.
- [ ] **Share error classification setup** — Every package calls `event.RegisterClassification()` in `init()`. Hidden global side effects with no conflict detection. Consider explicit setup.

---

## 🟢 LOW Priority (Nice-to-Have)

- [ ] **Add BDD tests for catalog, storage, sync**
- [ ] **Implement Saga/Process Manager** — design done, implementation pending
- [ ] **PostgreSQL integration tests for storage** — unit tests use go-sqlmock only
- [ ] **Add `event.Context` propagation to time.Now() calls**
- [ ] **Consolidate `Core` naming** — 4 packages export `Core` struct (`event`, `command`, `query`, `aggregate`). Consider domain-specific names.
- [ ] **Move `sync` to its own repository** — Zero imports from any other module. Fully isolated.
- [ ] **Rename `Of[T]` to something more descriptive** — `ID[T]` or `Branded[T]`

---

## ✅ COMPLETED (Session 84)

- [x] **Architecture review** — Full analysis: module depth, coupling, scalability, composability. Report at `docs/architecture-understanding/2026-05-21_00-20-SESSION_84_ARCHITECTURE_REVIEW.md`
- [x] **Architecture diagrams** — Current + ideal state D2 diagrams at `docs/architecture-understanding/`
- [x] **Naming review** — Automated detection + manual review. Fixed CQRSAdapter, Mixin suffixes, helpers.go files.
- [x] **Rename `CQRSAdapter` → `PebbleEventStore`** — Terrible name → honest name. `NewCQRSAdapter` → `NewPebbleStore`. Updated across storage, tests, example/todo.
- [x] **Rename `LifecycleMixin` → `Lifecycle`** — Idiomatic Go naming. Updated in dispatcher, memory (bus/store/snapshot).
- [x] **Rename `SyncContextMixin` → `SyncContext`** — Updated in sync package.
- [x] **Rename `PebbleMixin` → `PebbleBase`** — Updated in example/todo.
- [x] **Rename `helpers.go` files** — `storage/helpers.go` → `storage/event_reconstruction.go`, `memory/helpers.go` → `memory/keys.go`, `catalog/asyncapi/helpers.go` → `catalog/asyncapi/serde.go`
- [x] **Unify event serialization (3→1)** — `outbox_helpers.go` and `pebble_serialization.go` now both call shared `reconstructEvent()` from `event_reconstruction.go`. Eliminated ~60 lines of duplication.
- [x] **Refresh stale golden test files** — asyncapi.yaml, eventcatalog-config.js, package.json
- [x] **Zero lint across all 8 modules**
- [x] **All 24 test packages pass**

## ✅ COMPLETED (Session 82)

- [x] **example/todo build fix** — replace directives for core/memory/testhelpers, MarshalJSON recursion fix, test assertion fix
- [x] **Pebble iterateEvents corrupt event handling** — returns error instead of silently skipping
- [x] **Zero lint across all 8 modules**
- [x] **DDL on Dialect interface**
- [x] **VectorClock.Cmp() with ClockOrder enum**
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

## ✅ COMPLETED (Earlier Sessions)

- [x] **query.TypedHandler[T]** — Session 54
- [x] **TransactionalStore** — Session 55-56
- [x] **Middleware sentinel errors** — Session 54
- [x] **Replace cockroachdb/errors with stdlib** — Session 54
- [x] **Replace go-json-experiment/json with encoding/json** — Session 54
- [x] **Extract SnapshotStrategy to core/event** — Session 48
- [x] **ISP: Publisher/Subscriber sub-interfaces** — Session 48
- [x] **Error classification registration** — Session 48
- [x] **Extract shared PublishChanges + SaveSnapshot** — Session 48
- [x] **43 benchmarks across 12 files** — Session 50
