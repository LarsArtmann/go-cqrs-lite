# TODO List

**Audited:** 2026-05-03 · **Session 40**
**Sources:** Session 40 status report, Architecture Roadmap, Execution Plan, Session 38 deduplication plan

---

## 🔴 CRITICAL (Test Failures)

- [ ] **Fix 3 golden test failures** — Run tests with `-update`, verify diffs are cosmetic, commit
  - `catalog/asyncapi` — `TestGolden_AsyncAPIYAML`
  - `catalog/eventcatalog` — `TestGolden_EventCatalog_Config`, `TestGolden_EventCatalog_PackageJSON`
  - Root cause: golden files modified by formatting pass but never refreshed with `-update`

---

## 🔴 HIGH Priority (Blocks Production Use)

- [ ] **PostgreSQL outbox store** — `storage/postgres_outbox.go` implementing `event.Outbox`
  - `MemoryOutboxStore` exists in `memory/` (testing only)
  - `event.Outbox` interface exists in `core/event/`
  - No PostgreSQL implementation — consumers must implement their own
  - See: `docs/planning/2026-05-01_ARCHITECTURE_ROADMAP.md` Phase 1

- [ ] **Split `example/user/main.go`** — 132 lines, violates 30-line-per-function convention
  - Should be split into: `setupInfrastructure()`, `setupHandlers()`, `runDemo()`, `printUser()`
  - The example should model the library's own conventions

- [ ] **Trim `core/aggregate/repository.go`** — 258 lines, exceeds 250-line max
  - Extracted `publishChanges` helper in Session 36 (254→244), grew back to 258
  - Need another extraction or helper split

---

## 🟡 MEDIUM Priority (Library Quality)

- [ ] **Increase `core/decider` coverage** — 89.5% (below project >92% standard)
  - Missing: concurrent Execute calls, context cancellation, outbox support tests
  - See: `docs/status/2026-05-03_03-20_SESSION_40_COMPREHENSIVE_STATUS.md`

- [ ] **Add `Delete` method to `decider.Repository`** — feature parity with `aggregate.Repository`

- [ ] **Add snapshot support to `decider.Repository`** — `aggregate.Repository` has `SnapshotStrategy` + `SnapshotStore`, `decider` has neither

- [ ] **Add outbox support to `decider.Repository`** — production-grade decider needs outbox

- [ ] **Consolidate `CatalogMeta`** — identical struct in `event`, `command`, `query` packages
  - Extract to shared type in `core/event/` or create `core/catalog/` package

- [ ] **Fix `query.Handler` returns `any`** — known issue since Session 1
  - `DispatchTyped[T]` is the workaround but interface still uses `any`
  - Breaking change — needs migration plan

- [ ] **Create `CONTEXT.md`** — domain glossary defining aggregate, projection, decider, fold, decide, etc.
  - Formalizes project language for consumers

- [ ] **Create `docs/adr/`** — Architecture Decision Records
  - ADR-0001: Decider pattern over OO aggregate
  - ADR-0002: Error taxonomy design (5 families)
  - ADR-0003: Multi-module monorepo structure

---

## 🟢 LOW Priority (Polish)

- [ ] **Refactor `HandleParallel`** — 65 lines (exceeds 30-line max)
  - `projection/runner.go:HandleParallel`
  - `catalog/asyncapi/exporter.go:Export` — 54 lines
  - `catalog/eventcatalog/exporter.go:Export` — 42 lines

- [ ] **Event signing / integrity verification** — No HMAC or checksum on stored events
  - Corrupted events detectable via fold errors but not preventable

- [ ] **Context propagation through Decider** — `decider.Repository.Execute` doesn't pass context to store/bus

- [ ] **Archive stale planning docs** — 20+ obsolete files in `docs/planning/`
  - Move pre-2026-05-01 files to `docs/planning/archive/`

- [ ] **Archive stale status docs** — 25+ old files in `docs/status/`
  - Archive files older than 2 weeks

---

## 📐 PLANNED (No Code Exists)

- [ ] **Saga / Process Manager** — `docs/planning/SAGA_DESIGN.md` exists (orchestration pattern)
  - No implementation — needs saga.Core, saga.Step, saga.Instance, saga.Store, saga.Runner

- [ ] **Watermill module** — `docs/planning/2026-04-23_WATERMILL_PRO_CONTRA.md` evaluated
  - Pub/sub adapter for Kafka, NATS, etc. — never started

- [ ] **Tagged releases** — All modules at `v0.0.0`
  - Tag `v0.1.0-alpha` for core, memory, catalog, middleware modules when stabilized

---

## ✅ COMPLETED (Recently Implemented)

- [x] **Error taxonomy** — 5 families (Rejection, Conflict, Transient, Corruption, Infrastructure) in `core/event/errors.go`
- [x] **`id.ClientID` branded type** — `core/pkg/id/client_id.go`
- [x] **`event.WithClientID`, `event.WithClientOccurredAt`** — client metadata options
- [x] **`IdempotencyKey()` on Command interface** — breaking change with `command.Core` base implementation
- [x] **Projection retry with `event.IsRetryable()`** — wired in `projection/runner.go`
- [x] **`core/decider` package** — functional aggregate pattern (Session 37)
- [x] **PostgreSQL checkpoint store** — `SQLCheckpointStore` in `storage/checkpoint.go`
- [x] **PostgreSQL snapshot store** — `SQLSnapshotStore` in `storage/snapshot.go`
- [x] **PostgreSQL event store** — `SQLEventStore` in `storage/event_store.go`

---

_Last updated: 2026-05-03 (Session 40)_
