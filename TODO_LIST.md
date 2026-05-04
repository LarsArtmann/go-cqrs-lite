# TODO List

**Audited:** 2026-05-04 · **Session 50**
**Sources:** Sessions 48-50 execution (ISP, dedup, lint, coverage, benchmarks, design docs)

---

## 🔴 CRITICAL (Blocks Production Use)

None — all critical items resolved.

---

## 🟡 HIGH Priority (Library Quality)

- [ ] **Consolidate `CatalogMeta`** — duplicated in `event`, `command`, `query` packages
  - `event.CatalogMeta` has extra `AggregateType` field — not identical to command/query
  - `command.CatalogMeta` and `query.CatalogMeta` are identical (3 fields)
  - Options: shared base package, or accept as intentional per-package types
  - LOW effort, LOW impact — consider accepting duplication

- [ ] **Implement `query.TypedHandler[T]`** — design done, implementation pending
  - Design doc: `docs/planning/QUERY_HANDLER_GENERICS.md`
  - `DispatchTyped[T]` is the workaround but interface still uses `any`
  - Non-breaking migration via `RegisterTyped[T]` + `TypedHandler[T]` interface

- [ ] **Implement `TransactionalStore`** — design done, implementation pending
  - Design doc: `docs/planning/OUTBOX_TRANSACTION_API.md`
  - Atomic save+outbox in single database transaction
  - Non-breaking via optional interface + type assertion

---

## 🟢 LOW Priority (Nice-to-Have)

- [ ] **Add remaining benchmarks** — 33 benchmarks exist in 10 files; missing from `middleware`, `core/event`
- [ ] **Implement Saga/Process Manager** — design done, implementation pending
  - Design doc: `docs/planning/SAGA_DESIGN.md` (4-phase plan, 18h estimate)
- [ ] **Tag `v0.1.0-alpha`** — first public release after verification
- [ ] **PostgreSQL integration tests for storage** — unit tests use go-sqlmock only
- [ ] **Create CONTRIBUTING.md** — architecture guidelines for contributors

---

## ✅ COMPLETED (Session 50)

- [x] **Fix TODO_LIST.md false benchmark claim** — "zero benchmarks" → "26 benchmarks" → now 33
- [x] **Fix FEATURES.md stale coverage numbers** — 9 numbers corrected, ISP Publisher row added, decider added to matrix
- [x] **Fix CHANGELOG.md duplicate sections** — merged duplicate `### Changed` under `[Unreleased]`
- [x] **Add decider benchmarks** — 4 benchmarks in `core/decider/benchmark_test.go`
- [x] **Add projection benchmarks** — 3 benchmarks in `projection/benchmark_test.go`
- [x] **Design outbox transaction co-participation API** — `docs/planning/OUTBOX_TRANSACTION_API.md`
- [x] **Design query.Handler generics migration** — `docs/planning/QUERY_HANDLER_GENERICS.md`
- [x] **Review SAGA_DESIGN.md** — answered open questions, added 4-phase implementation plan
- [x] **Investigate go.mod ginkgo/gomega warnings** — already direct deps, gopls false positive

## ✅ COMPLETED (Session 48)

- [x] **Extract SnapshotStrategy to core/event** — deduplicated from aggregate/decider
- [x] **ISP: Publisher/Subscriber sub-interfaces** — activated in aggregate, decider, projection, outbox
- [x] **Error classification registration** — aggregate, projection, storage sentinels registered
- [x] **Fix all lint issues** — zero issues across all modules
- [x] **Test coverage gaps** — memory 99.1%, aggregate 95.3%, storage 93.6%
- [x] **Fix root go.mod module path** — `LarsArtmann` → `larsartmann`
- [x] **Extract shared PublishChanges + SaveSnapshot** — to core/event/publish_helper.go
