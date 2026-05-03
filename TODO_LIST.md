# TODO List

**Audited:** 2026-05-03 · **Session 48**
**Sources:** Phases 1-7 execution (ISP, dedup, lint, coverage, shared helpers)

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

- [ ] **Fix `query.Handler` returns `any`** — known issue since Session 1
  - `DispatchTyped[T]` is the workaround but interface still uses `any`
  - Breaking change — needs migration plan

---

## 🟢 LOW Priority (Nice-to-Have)

- [ ] **Add missing benchmarks** — 26 benchmarks exist in 7 files (integration/*, core/pkg/id, catalog); missing from `core/decider`, `projection`, `middleware`
- [ ] **Design outbox transaction co-participation API** — currently Save + Append are separate calls
- [ ] **Design `query.Handler` generics migration** — breaking change plan
- [ ] **Tag `v0.1.0-alpha`** — first public release after all phases complete

---

## ✅ COMPLETED (Session 48)

- [x] **Extract SnapshotStrategy to core/event** — deduplicated from aggregate/decider
- [x] **ISP: Publisher/Subscriber sub-interfaces** — activated in aggregate, decider, projection, outbox
- [x] **Error classification registration** — aggregate, projection, storage sentinels registered
- [x] **Fix all lint issues** — zero issues across all modules
- [x] **Test coverage gaps** — memory 99.1%, aggregate 95.8%, storage 93.6%
- [x] **Fix root go.mod module path** — `LarsArtmann` → `larsartmann`
- [x] **Extract shared PublishChanges + SaveSnapshot** — to core/event/publish_helper.go
