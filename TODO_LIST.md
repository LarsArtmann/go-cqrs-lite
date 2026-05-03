# TODO List

**Audited:** 2026-05-03 · **Session 43**
**Sources:** Session 43 execution (bug fixes + test quality)

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

## 🟡 MEDIUM Priority (Developer Experience)

None — all medium items resolved.

---

## 🟢 LOW Priority (Polish)

- [ ] **Event signing / integrity verification** — No HMAC or checksum on stored events

---

## 📐 PLANNED (No Code Exists)

- [ ] **Saga / Process Manager** — `docs/planning/SAGA_DESIGN.md` exists (orchestration pattern)
  - Needs saga.Core, saga.Step, saga.Instance, saga.Store, saga.Runner

- [ ] **Watermill module** — `docs/planning/archive/2026-04-23_WATERMILL_PRO_CONTRA.md` evaluated
  - Pub/sub adapter for Kafka, NATS, etc. — never started

- [ ] **Tagged releases** — All modules at `v0.0.0`
  - Tag `v0.1.0-alpha` when modules stabilize

---

## ✅ COMPLETED (Session 43)

- [x] **Fix `HandleParallel` panic recovery** — goroutine now recovers panics instead of deadlocking
- [x] **Fix `OutboxPublisher.run()` panic recovery** — background goroutine recovers panics
- [x] **Fix `TestCoreDoesNotImplementRootDirectly`** — added proper assertions
- [x] **Fix `TestOutboxPublisher_PublishNow_ContextCanceled`** — added error assertion
- [x] **Fix `FakeOutbox.PollPending`** — `Lock()` → `RLock()` + changed to `sync.RWMutex`
- [x] **Fix `FakeStore.Save` data race** — `saveFn` read under `RLock`
- [x] **Fix `FakeStore`/`FakeOutbox` defer unlocks** — all methods use `defer` unlock
- [x] **Split `core/decider/decider.go`** — 292→243 lines (extracted `loadFromSnapshot`)
- [x] **Increase `core/decider` coverage** — 77.4%→94.3% (8 new tests for error paths)
- [x] **Replace `time.Sleep` in projection tests** — 9× sleep replaced with channel-based sync
- [x] **Add concurrent access tests for memory** — MemoryStore + MemoryBus tested with `-race`

---

## ✅ COMPLETED (Session 42)

- [x] **PostgreSQL outbox store** — `storage/outbox.go` implementing `event.Outbox`
  - `SQLOutbox` with `Append`, `PollPending`, `Ack`, `Close`
  - `OutboxSchema()` DDL for `outbox` table with partial index on pending status
  - Events serialized as JSONB (round-trip tested)
  - 10 tests with go-sqlmock (Append, PollPending, Ack, round-trip, errors, nil checks)
- [x] **Refactor long functions** — reduced from 16 to 8 functions exceeding 30-line max
  - `storage/helpers.go:scanEvents` 76→22 lines (extracted `scanEvent`)
  - `core/event/runner.go:HandleParallel` 66→5 lines (extracted 3 methods)
  - `storage/event_store.go:Save` 53→~30 lines (extracted `checkVersion`)
  - `storage/event_store.go:AppendBatch` 49→~20 lines (reused `insertEvents`)
  - `storage/snapshot.go` Load/LoadAtVersion refactored (shared `scanSnapshot`)
- [x] **Add snapshot support to `decider.Repository`** — feature parity with aggregate
  - `WithSnapshotStore`, `WithCodec`, `WithSnapshotStrategy` options
  - `loadFromSnapshot`, `saveSnapshot`, `shouldSnapshot` methods
  - Non-fatal snapshot errors (consistent with aggregate behavior)
- [x] **Increase `core/decider` test coverage** — added 4 new tests
  - `TestExecute_WithSnapshot` — snapshot save on strategy trigger
  - `TestLoad_WithSnapshot` — state reconstruction from snapshot + remaining events
  - `TestExecute_Concurrent` — 5 goroutines executing concurrently
  - `TestExecute_ContextCancellation` — canceled context propagated via `ctxCheckStore`
- [x] **Create `CONTEXT.md`** — domain glossary with 20 terms
- [x] **Create `docs/adr/`** — 3 ADRs (decider pattern, error taxonomy, multi-module monorepo)
- [x] **Archive stale docs** — moved 15 planning + 15 status files to `archive/` subdirectories
- [x] **Refresh golden test files** — 3 catalog golden files updated

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
- [x] **PostgreSQL outbox store** — `SQLOutbox` in `storage/outbox.go`

---

_Last updated: 2026-05-03 (Session 43)_
