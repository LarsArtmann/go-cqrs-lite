# Session 44 — Comprehensive Status Report

**Date:** 2026-05-03 05:57  
**Branch:** `master`  
**Commits since Session 42:** 6  
**Test suites:** 21 packages, ALL PASS  
**Total LOC:** 30,681 Go  
**Lint:** 46 pre-existing issues (0 new from this session)

---

## A) FULLY DONE ✓

### Session 44 Deliverables

| Item | Status | Commit | Detail |
|------|--------|--------|--------|
| **Publisher/Subscriber sub-interfaces** | ✅ DONE | `a46b4c9` | `event.Publisher` + `event.Subscriber` composed by `event.Bus`. Non-breaking ISP improvement. |
| **Extensible error classification** | ✅ DONE | `bf5f731` | `RegisterClassification()` lets command/query/aggregate register sentinels without circular deps. `Classify()` checks registered map. |
| **Projection logger injection** | ✅ DONE | `6c3fcf7` | `WithLogger(*slog.Logger)` option replaces global `slog.Default()`. DI over globals. |
| **Lint compliance** | ✅ DONE | `81a0a27` | Resolved `gochecknoglobals` (inline struct init) and `gochecknoinits` (nolint directives). Fixed `wsl_v5` in test. |
| **errors.As → errors.AsType** | ✅ DONE | `3b02c3a` | Modernized example/user to use Go 1.26 `errors.AsType`. |

### Session 43 Deliverables (committed at session start)

| Item | Status | Commit | Detail |
|------|--------|--------|--------|
| Panic recovery in HandleParallel | ✅ DONE | `623609a` | `core/event/runner.go` goroutine recovers panics |
| Panic recovery in OutboxPublisher | ✅ DONE | `623609a` | `core/event/outbox_publisher.go` goroutine recovers panics |
| Test assertion fixes | ✅ DONE | `623609a` | Added assertions to `TestCoreDoesNotImplementRootDirectly`, `TestOutboxPublisher_PublishNow_ContextCanceled` |
| Data race fixes (FakeOutbox, FakeStore) | ✅ DONE | `623609a` | `sync.RWMutex` with `RLock()`, `defer` unlock throughout |
| Decider refactor (292→243 lines) | ✅ DONE | `623609a` | Extracted `loadFromSnapshot` to `options.go` |
| Decider coverage 77.4%→94.3% | ✅ DONE | `623609a` | 8 new tests for snapshot error paths |
| Projection test quality | ✅ DONE | `623609a` | Replaced 9× `time.Sleep` with channel-based `subscribeSignalBus` |
| Memory concurrent access tests | ✅ DONE | `623609a` | `MemoryStore` and `MemoryBus` `-race` tests |

### Evergreen (All Sessions)

- Error taxonomy (5 families) — `core/event/errors.go`
- **Extensible classification** — `RegisterClassification()` + `init()` in command/query
- `id.ClientID` branded type
- `IdempotencyKey()` on Command interface
- Projection retry with `event.IsRetryable()`
- All PostgreSQL stores (event, snapshot, checkpoint, outbox)
- `core/decider` package — functional aggregate pattern
- Multi-module monorepo with 9+1 modules
- Catalog system (AsyncAPI, D2, EventCatalog exporters)
- **Publisher/Subscriber sub-interfaces** — ISP-compliant `event.Bus`
- **Injected logger** — projection.Runner uses `*slog.Logger` via `WithLogger`
- 21 test packages, 0 failures

---

## B) PARTIALLY DONE ⚠️

| Item | Status | What's Done | What's Missing |
|------|--------|-------------|----------------|
| **Function size compliance** | ⚠️ Partial | Session 43 extracted worst offenders | 43 functions still exceed 30-line max (catalog exporters, validateEventParams at 51) |
| **CatalogMeta consolidation** | ⚠️ SKIPPED | Investigated | `event.CatalogMeta` has extra `AggregateType` field — not identical, no clean shared location |

---

## C) NOT STARTED ○

| Item | Priority | Notes |
|------|----------|-------|
| `query.Handler` returns `any` | HIGH | Breaking change — `DispatchTyped[T]` is the workaround |
| Transaction co-participation (outbox + events) | HIGH | `SQLOutbox.Append` runs in separate tx from `SQLEventStore.Save` |
| Event signing / integrity verification | LOW | No HMAC or checksum on stored events |
| Saga / Process Manager | PLANNED | `docs/planning/SAGA_DESIGN.md` exists |
| Watermill module | PLANNED | Pub/sub adapter for Kafka, NATS |
| Tagged releases | PLANNED | All modules at `v0.0.0` |

---

## D) WHAT WE SHOULD IMPROVE

### Architecture

1. **Transaction co-participation** — #1 architectural gap. Outbox + event store must share a transaction.

2. **Classify() coverage expansion** — Currently covers event, command, query sentinels. Still missing: aggregate (`ErrAggregateNotFound`, etc.), projection (`ErrDuplicateProjection`, etc.), storage errors. These have circular dependency constraints but could register via `init()` like command/query.

3. **Function size enforcement** — 43 functions >30 lines. Catalog exporters are the bulk. Consider accepting 40-line max for rendering functions.

### Test Quality

4. **Projection coverage at 88.8%** — Below 95% target. Missing error paths in retry logic and checkpoint handling.

5. **Integration tests for outbox** — `SQLOutbox` has sqlmock tests but no integration test exercising the full write→poll→publish→ack cycle.

6. **Storage coverage at 92.0%** — `scanOutboxEntries` at 75%, `reconstructOutboxEvent` at 76.9%.

### Developer Experience

7. **`query.Handler` returns `any`** — Violates "no any" rule. Breaking change needed.

8. **No benchmarks** — No performance benchmarks for outbox or decider operations.

---

## E) TOP 10 THINGS TO DO NEXT

Ranked by impact × effort (Pareto):

### CRITICAL

1. **Fix outbox transaction co-participation** — `SQLOutbox.Append` must run inside `SQLEventStore.Save`'s transaction.

### HIGH

2. **Register aggregate + projection + storage sentinels** — Extend `RegisterClassification` pattern to remaining packages.

3. **Increase projection coverage to 95%+** — Add retry error path tests, checkpoint error tests.

4. **Increase storage coverage to 95%+** — Add `scanOutboxEntries` error paths, `reconstructOutboxEvent` edge cases.

### MEDIUM

5. **Refactor catalog exporter long functions** — Extract sub-functions from asyncapi.Export (62L), d2.writeCrossServiceConnections (60L).

6. **Add outbox integration test** — Full cycle: Append → PollPending → Publish → Ack.

7. **Add benchmarks** — Decider operations, outbox throughput, event store save.

8. **Add `CHANGELOG.md`** — Track changes per session for release notes.

9. **Tag `v0.1.0-alpha`** — All core modules are stable enough for early adopters.

10. **Refactor `validateEventParams`** (51 lines) — Split 4 validation blocks.

---

## F) TOP #1 QUESTION

**How should outbox transaction co-participation work?**

This was identified in Session 42 and remains unresolved. Three options:

1. **`SQLEventStore` accepts `Outbox` reference** — `Save()` begins tx, inserts events, calls `outbox.Append(ctx, tx, events)`, commits. Requires `Outbox.Append` to accept `*sql.Tx`.

2. **Accept external `*sql.Tx`** — Both get `SaveWithTx`/`AppendWithTx` methods. Consumer manages the transaction.

3. **Outbox wraps event store** — `OutboxEventStore` composes both, manages single tx internally.

This is an interface-breaking decision that requires consumer input.

---

## Test Coverage Summary

| Package | Coverage | Target | Status |
|---------|----------|--------|--------|
| `core/command` | 100.0% | >95% | ✅ |
| `core/query` | 100.0% | >95% | ✅ |
| `core/pkg/dispatcher` | 100.0% | >95% | ✅ |
| `core/pkg/id` | 100.0% | >95% | ✅ |
| `middleware` | 100.0% | >95% | ✅ |
| `core/event` | 98.0% | >95% | ✅ |
| `catalog/d2` | 97.6% | >95% | ✅ |
| `catalog/asyncapi` | 95.9% | >95% | ✅ |
| `catalog/eventcatalog` | 95.6% | >95% | ✅ |
| `catalog/adapters` | 95.5% | >95% | ✅ |
| `core/decider` | 94.3% | >95% | ⚠️ |
| `catalog` | 94.4% | >95% | ⚠️ |
| `core/aggregate` | 93.2% | >95% | ⚠️ |
| `storage` | 92.0% | >95% | ⚠️ |
| `memory` | 91.9% | >95% | ⚠️ |
| `projection` | 88.8% | >95% | ⚠️ |

---

_Generated: 2026-05-03 05:57 — Session 44_
