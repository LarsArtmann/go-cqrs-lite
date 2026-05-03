# TODO List

**Audited:** 2026-05-03 · **Session 41**
**Sources:** Session 40 status report, Session 41 audit + execution

---

## 🔴 CRITICAL (Blocks Production Use)

- [ ] **PostgreSQL outbox store** — `storage/postgres_outbox.go` implementing `event.Outbox`
  - `MemoryOutboxStore` exists in `memory/` (testing only)
  - `event.Outbox` interface exists in `core/event/`
  - No PostgreSQL implementation — consumers must implement their own
  - Design challenge: outbox Append must run inside same tx as event inserts

---

## 🟡 HIGH Priority (Library Quality)

- [ ] **Increase `core/decider` coverage** — ~92% (below project >95% standard)
  - Missing: concurrent Execute calls, context cancellation
  - New Delete/outbox tests added in Session 41 but more edge cases needed

- [ ] **Refactor long functions** — 16 functions exceed 30-line max
  - `storage/helpers.go:scanEvents` — 76 lines (worst offender)
  - `core/event/runner.go:HandleParallel` — 66 lines
  - `storage/event_store.go:Save` — 53 lines
  - `core/event/event.go:validateEventParams` — 51 lines
  - `storage/event_store.go:AppendBatch` — 49 lines
  - `storage/snapshot.go:LoadAtVersion` — 43 lines
  - 10 more between 31-40 lines

- [ ] **Consolidate `CatalogMeta`** — identical struct in `event`, `command`, `query` packages
  - Extract to shared type in `core/event/` or create `core/catalog/` package

- [ ] **Fix `query.Handler` returns `any`** — known issue since Session 1
  - `DispatchTyped[T]` is the workaround but interface still uses `any`
  - Breaking change — needs migration plan

---

## 🟡 MEDIUM Priority (Developer Experience)

- [ ] **Create `CONTEXT.md`** — domain glossary defining aggregate, projection, decider, fold, decide, etc.

- [ ] **Create `docs/adr/`** — Architecture Decision Records
  - ADR-0001: Decider pattern over OO aggregate
  - ADR-0002: Error taxonomy design (5 families)
  - ADR-0003: Multi-module monorepo structure

- [ ] **Add snapshot support to `decider.Repository`** — `aggregate.Repository` has `SnapshotStrategy` + `SnapshotStore`, `decider` has neither

---

## 🟢 LOW Priority (Polish)

- [ ] **Archive stale planning docs** — 20+ obsolete files in `docs/planning/` (pre-2026-05-01)
- [ ] **Archive stale status docs** — 25+ old files in `docs/status/` (older than 2 weeks)
- [ ] **Event signing / integrity verification** — No HMAC or checksum on stored events

---

## 📐 PLANNED (No Code Exists)

- [ ] **Saga / Process Manager** — `docs/planning/SAGA_DESIGN.md` exists (orchestration pattern)
  - Needs saga.Core, saga.Step, saga.Instance, saga.Store, saga.Runner

- [ ] **Watermill module** — `docs/planning/2026-04-23_WATERMILL_PRO_CONTRA.md` evaluated
  - Pub/sub adapter for Kafka, NATS, etc. — never started

- [ ] **Tagged releases** — All modules at `v0.0.0`
  - Tag `v0.1.0-alpha` when modules stabilize

---

## ✅ COMPLETED (Session 41)

- [x] **Fix 3 golden test failures** — refreshed testdata with `-update`, all pass
- [x] **Trim `core/aggregate/repository.go`** — 258 → 245 lines (under 250 limit)
- [x] **Add option pattern to `decider.Repository`** — `RepositoryOption[State]` functional options
- [x] **Add `WithOutbox` option to decider** — same pattern as aggregate.Repository
- [x] **Add `Delete` method to `decider.Repository`** — feature parity with aggregate
- [x] **Split `example/user/main.go`** — 132-line main() → 6 focused functions ≤30 lines each

---

## ✅ COMPLETED (Previous Sessions)

- [x] **Error taxonomy** — 5 families in `core/event/errors.go`
- [x] **`id.ClientID` branded type** — `core/pkg/id/client_id.go`
- [x] **`event.WithClientID`, `event.WithClientOccurredAt`** — client metadata options
- [x] **`IdempotencyKey()` on Command interface** — breaking change with `command.Core` migration
- [x] **Projection retry with `event.IsRetryable()`** — wired in `projection/runner.go`
- [x] **`core/decider` package** — functional aggregate pattern (Session 37)
- [x] **PostgreSQL checkpoint store** — `SQLCheckpointStore` in `storage/checkpoint.go`
- [x] **PostgreSQL snapshot store** — `SQLSnapshotStore` in `storage/snapshot.go`
- [x] **PostgreSQL event store** — `SQLEventStore` in `storage/event_store.go`

---

_Last updated: 2026-05-03 (Session 41)_
