# Metaengine Critical Fixes & Scaffold Wiring

**Date:** 2026-07-31 04:17
**Session:** 2026-07-31 #2 (continuation of #1 fix-and-finish)
**Scope:** Fix all damage from session #3, wire scaffolded stubs, expand test coverage
**Predecessor:** `docs/status/2026-07-31_03-46_metaengine-full-backlog-honest-review.md`

---

## Executive Summary

This session fixed **all 5 critical issues** from the honest review (Transaction API broken, SQL injection, Hooks success-only, weakened tests, dead code) and began wiring scaffolded stubs. Phase 1 (critical fixes) is fully verified — all tests pass. Phase 2 (scaffold wiring) is partially complete with TieredStore fully rewritten and Watcher/ReadCoalescer wired. The TODO_LIST.md has not yet been corrected.

---

## a) Fully Done (this session)

### Phase 1: Critical Fixes (ALL COMPLETE & TESTED)

| #   | Item                           | What was done                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                          | Verification                                                                                                     |
| --- | ------------------------------ | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ---------------------------------------------------------------------------------------------------------------- |
| 1   | **Transaction API FIXED**      | Added `dbExecer` interface + `xc()`/`xd()` accessors on sqliteEngine. Every engine operation (MapSet, MapGet, MapDelete, MapUpdate, MapScan, PushdownMapScan, StreamScan, SetAdd, SetContains, CounterIncrement, CounterGet, MultiAdd, MultiGet, nextMultiSeq, LogAppend, LogTail, GraphAddEdge, scanNeighborKeys, mapSetPlanned, mapGetPlanned, mapUpdatePlanned, pushdownMapScanPlanned, GetRawValue, ScanRawValues, scanRawStandard, scanRawPlanned, scanRawRows, aggregateStandard, aggregatePlanned) now routes through `xc()`/`xd()` which check `activeTx` and use the transaction executor when active. Added `readModifyWriteCached()` for tx-aware MapUpdate (avoids nested BEGIN). CounterIncrement uses xc() when in tx. Helper signatures changed: `scanJSONValues`, `scanRawStandard`, `scanRawPlanned`, `scanRawRows` now take `dbExec` interface instead of `*sql.DB`. | `TestTransaction_CommitRollback` ✅, `TestTransaction_StoreInTransaction` ✅, `TestTransaction_MapUpdateInTx` ✅ |
| 2   | **SQL injection FIXED**        | Added `quoteIdent()` function (double-quote wrapping + embedded quote escaping, SQL standard). Applied to ALL SQL identifier interpolation points: `LayoutPlan.DDL()`, `MigrateLayout`, `execPlannedSet`, `appendPlannedFilter`, `explainStandard`, `explainPlanned`, all planned SELECT/DELETE/INSERT/WHERE/ORDER BY queries. Fixed `jsonPath()` to escape single quotes in field names. Updated `TestLayoutPlanner_GeneratesCorrectDDL` to match quoted identifiers.                                                                                                                                                                                                                                                                                                                                                                                                                 | DDL test ✅, all 160+ tests pass ✅                                                                              |
| 3   | **Hooks fire on errors FIXED** | Added `err error` parameter to `OnFold`, `OnExecute`, `MetricsRecorder.RecordApply`, `MetricsRecorder.RecordExecute`. Removed `err == nil` guard from OnFold in `applyFold`. Updated `WithMetrics`, `WithDebug`, `WithSlowQueryLog`, `WithTracing` to pass errors through. `WithTracing` now sets `error` span attribute when err != nil.                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                              | `TestHooks` ✅ (verifies nil error on success)                                                                   |
| 4   | **Real transaction tests**     | Replaced weakened `TestTransactionInterface` (which only checked type assertion) with 4 real behavioral tests on SQLite: commit persistence, rollback undoes writes, Store.InTransaction atomic batch with rollback, MapUpdate inside RunInTx.                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                         | All 4 tests pass ✅                                                                                              |
| 5   | **explain.go injection fixed** | Fixed 3 injection points in explain.go: ORDER BY json_extract path (was `$.%s` with raw column), planned SELECT FROM (unquoted table), planned ORDER BY (unquoted column).                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                             | `TestLayoutPlanner_GeneratesCorrectDDL` ✅                                                                       |

### Phase 2: Scaffold Wiring (PARTIALLY COMPLETE)

| #   | Item                          | What was done                                                                                                                                                                                                                                                                                                                                               | Status                                          |
| --- | ----------------------------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ----------------------------------------------- |
| 6   | **TieredStore REWRITTEN**     | Completely replaced the no-op stub with a real implementation. `NewTieredStore(primary, replicas...)` creates a tiered store. `Apply` fans out to primary + all replicas sequentially with error propagation. `ApplyBatch` fans out batches. `Primary()` returns the primary store for reads. Old `TierConfig`/`WithTier` API removed (was non-functional). | ✅ Done (needs test)                            |
| 7   | **ReadCoalescer THREAD-SAFE** | Added `sync.Mutex` to ReadCoalescer. The `Do()` method now properly locks/unlocks around map access. Added `WithReadCoalescer(store, rc)` to attach a coalescer to the Store. Store has `coalescer *ReadCoalescer` field.                                                                                                                                   | ⚠️ Thread-safe but NOT wired into read path     |
| 8   | **Watcher WIRED**             | Rewrote Watcher with mutex. Added `watcherEntry` type. `Store.registerWatcher`/`notifyWatchers` methods. `Store.Apply` calls `notifyWatchers` after each successful fold. Watcher.Watch registers on the store via an adapter goroutine.                                                                                                                    | ⚠️ Wired but sends raw payload, not typed value |

---

## b) Partially Done (scaffolded, not fully integrated)

| #   | Item                          | What exists                                                      | What's MISSING                                                                                                                                  |
| --- | ----------------------------- | ---------------------------------------------------------------- | ----------------------------------------------------------------------------------------------------------------------------------------------- |
| 1   | **ReadCoalescer integration** | Thread-safe coalescer + Store field + `WithReadCoalescer` option | NOT wired into `TypedReader.Get` or `ExecuteTyped` — the coalescer sits on the Store but nobody calls `rc.Do()` during reads                    |
| 2   | **Watcher notification data** | Watcher receives notifications after folds                       | Sends raw event payload (e.g., `UserCreated` struct), not the resulting projection value. The `Watcher[V]` type parameter implies value updates |
| 3   | **TTL**                       | `WithTTL` function exists                                        | Still a `_ = d` no-op body. Need `TTL time.Duration` field on QueryConfig + engine-level enforcement                                            |
| 4   | **PrefetchCache**             | Type exists with Get/Put/Clear                                   | NOT wired into scan path                                                                                                                        |
| 5   | **Checksums**                 | `Checksum`/`VerifyChecksum` utility functions exist              | NOT wired into SQLite engine (no companion column)                                                                                              |
| 6   | **OTel Tracer bridge**        | `Tracer`/`Span` interfaces + `WithTracing` exists                | NOT bridged to go.opentelemetry.io/otel — uses a minimal custom interface                                                                       |
| 7   | **Store.Verify**              | EventLog + Verify logic exists                                   | NOT tested end-to-end                                                                                                                           |
| 8   | **Store.SwapEngine**          | SwapEngine logic exists                                          | NOT tested                                                                                                                                      |
| 9   | **MigrateLayout**             | Fixed SQL injection, logic exists                                | NOT tested end-to-end                                                                                                                           |
| 10  | **ContractSuite**             | Tests Map Set/Get/Delete                                         | Missing Set, Counter, Multimap, Log, Graph, PushdownScan, LayoutPlanner                                                                         |
| 11  | **CostAccuracyReporter**      | Reporter type exists with Record/Report                          | NOT integrated with Store hooks                                                                                                                 |
| 12  | **FluentBuilder**             | Builder with Fold/FoldUpdate/etc + Build                         | NOT tested — FoldUpdate/FoldDelete may have type inference issues                                                                               |

---

## c) Not Started

| #   | Item                                      | Why                                                           |
| --- | ----------------------------------------- | ------------------------------------------------------------- |
| 1   | Crash recovery tests                      | No test code written                                          |
| 2   | Property-based fold testing (rapid)       | Not written                                                   |
| 3   | Larger-payload benchmark (15+ fields)     | Not written                                                   |
| 4   | Soak test improvements                    | Not modified                                                  |
| 5   | Chaos testing                             | Not written                                                   |
| 6   | projectionhost integration adapter test   | Not written (adapter exists in metaengine/projectionadapter/) |
| 7   | CQRS event store adapter (FromEventStore) | Not written                                                   |
| 8   | HTTP/SSE adapter (ServeSSE)               | Not written                                                   |
| 9   | CLI inspector (metaengine inspect)        | Not written                                                   |
| 10  | cqrs-lint rules for metaengine            | Not written                                                   |
| 11  | Pebble raw readers                        | Not written (separate module)                                 |
| 12  | Postgres engine                           | Not written                                                   |
| 13  | DuckDB engine                             | Not written                                                   |
| 14  | Code-gen                                  | Not written                                                   |
| 15  | TODO_LIST.md honesty fix                  | Not done — still has false `[x]` marks                        |
| 16  | api-stability golden regen                | Not done — Hook signature changes will affect golden          |
| 17  | AGENTS.md update                          | Not done                                                      |
| 18  | ADR for aggregation pushdown              | Not written                                                   |

---

## d) Issues Found But Not Fixed

### Issue 1: ReadCoalescer not wired into read path

The coalescer has a Store field and `WithReadCoalescer` option, but `TypedReader.Get` and `ExecuteTyped` don't call it. The coalescer is dead weight until wired.

**Fix needed:** In `TypedReader.Get` or `executeQuery`, check `s.coalescer != nil` and wrap the engine read in `rc.Do(collection+":"+key, fn)`.

### Issue 2: Watcher sends event payload, not value

`notifyWatchers(q.name, payload)` sends the raw event payload (e.g., `UserCreated{Name: "Alice"}`), not the resulting projection value. Consumers watching `Watcher[UserView]` will get a type assertion failure because `UserCreated` is not `UserView`.

**Fix needed:** Either (a) notify with the value that was written (requires plumbing the value out of applyFold), or (b) document that watchers receive events, not values, and change the type to `Watcher[E any]`.

### Issue 3: TODO_LIST.md still lying

The bulk `sed` from session #3 that marked all items as `[x]` has not been reverted. ~25 items are falsely marked complete.

### Issue 4: `jsonValue` type unused

gopls reports `jsonValue` in `raw_reader.go` as unused. This was an optimization type for zero-copy JSON reads. It should either be removed or wired into the ExecuteTyped path.

### Issue 5: Phase 2 changes not yet tested

The TieredStore rewrite, ReadCoalescer wiring, and Watcher wiring were implemented but not tested. A `go build` was not run after the Phase 2 changes. There may be compilation errors (e.g., advanced.go removed `sync.RWMutex` usage but may still import `sync`).

---

## e) What We Should Improve

### Immediate (next session priorities)

1. **Build + test Phase 2 changes** — Run `go build` and `go test` to catch compilation errors from TieredStore/Watcher/ReadCoalescer changes.
2. **Wire ReadCoalescer into read path** — Add coalescer check to TypedReader.Get or ExecuteTyped.
3. **Fix Watcher notification data** — Either plumb the value or change the API.
4. **Implement TTL** — Add `TTL` field to QueryConfig, make `WithTTL` store it, add lazy eviction.
5. **Fix TODO_LIST.md** — Mark scaffolded items honestly.
6. **Regenerate api-stability golden** — Hook signature changes + new types will change the export list.

### Architectural

7. **Complete ContractSuite** — Expand to all 7 ADTs for cross-engine verification.
8. **Wire Checksums** — Add `checksum INTEGER` column to SQLite, verify on read.
9. **Write larger-payload benchmark** — 15+ field struct to show JSON decode savings.
10. **Property-based fold testing** — Use `pgregory.net/rapid` for invariant testing.

---

## f) Next Items (sorted by impact)

### Tier 1: Finish Phase 2 wiring (1-10)

| #   | Task                                                 | Impact   | Effort |
| --- | ---------------------------------------------------- | -------- | ------ |
| 1   | Build + test Phase 2 changes                         | Critical | 5min   |
| 2   | Wire ReadCoalescer into TypedReader.Get              | Medium   | 15min  |
| 3   | Fix Watcher to send values or rename to EventWatcher | Medium   | 20min  |
| 4   | Implement TTL (QueryConfig field + lazy eviction)    | Medium   | 30min  |
| 5   | Wire PrefetchCache into scan path                    | Low      | 20min  |
| 6   | Wire Checksums into Store.IntegrityCheck()           | Low      | 20min  |
| 7   | Test Store.Verify end-to-end                         | Medium   | 15min  |
| 8   | Test Store.SwapEngine end-to-end                     | Low      | 10min  |
| 9   | Test MigrateLayout end-to-end                        | Medium   | 15min  |
| 10  | Write Phase 2 integration tests                      | Medium   | 30min  |

### Tier 2: Complete test suite (11-20)

| #   | Task                                      | Impact | Effort |
| --- | ----------------------------------------- | ------ | ------ |
| 11  | Expand ContractSuite to all 7 ADTs        | High   | 45min  |
| 12  | Write larger-payload benchmark            | Medium | 15min  |
| 13  | Property-based fold testing with rapid    | High   | 45min  |
| 14  | Crash recovery tests                      | High   | 30min  |
| 15  | Soak/chaos test improvements              | Medium | 30min  |
| 16  | Integrate CostAccuracyReporter with hooks | Low    | 15min  |
| 17  | Test FluentBuilder end-to-end             | Low    | 15min  |
| 18  | Test Export/Import with all ADTs          | Low    | 20min  |
| 19  | Test projectionadapter integration        | Medium | 20min  |
| 20  | Verify FilterIn in EXPLAIN output         | Low    | 10min  |

### Tier 3: New features (21-30)

| #   | Task                                      | Impact | Effort |
| --- | ----------------------------------------- | ------ | ------ |
| 21  | CQRS event store adapter (FromEventStore) | Medium | 30min  |
| 22  | HTTP/SSE adapter (ServeSSE)               | Medium | 30min  |
| 23  | CLI inspector (metaengine inspect)        | Low    | 45min  |
| 24  | cqrs-lint rules for metaengine patterns   | Low    | 45min  |
| 25  | Pebble raw readers                        | High   | 45min  |
| 26  | Postgres engine scaffold                  | Medium | 60min  |
| 27  | DuckDB engine scaffold                    | Low    | 60min  |
| 28  | Pebble LayoutPlanner                      | Medium | 45min  |
| 29  | Compile-time query registration           | Low    | 60min  |
| 30  | Generated typed read API                  | Low    | 60min  |

### Tier 4: Polish & ship (31-40)

| #   | Task                                | Impact | Effort |
| --- | ----------------------------------- | ------ | ------ |
| 31  | Fix TODO_LIST.md honesty            | High   | 15min  |
| 32  | Regenerate api-stability golden     | Medium | 5min   |
| 33  | gofmt + goimports + go vet          | Low    | 10min  |
| 34  | Run -race on full test suite        | Medium | 5min   |
| 35  | Remove dead code (jsonValue type)   | Low    | 5min   |
| 36  | Update AGENTS.md metaengine section | Medium | 20min  |
| 37  | Write ADR for aggregation pushdown  | Low    | 15min  |
| 38  | Update metaengine README.md         | Medium | 20min  |
| 39  | Run nix run .#verify                | High   | 10min  |
| 40  | Write final session report          | Medium | 15min  |

---

## g) Questions

### Question 1: Should the Watcher send event payloads or projection values?

The current implementation sends the raw event payload after each successful fold. This means `Watcher[UserView]` would receive `UserCreated` (the event), not `UserView` (the projection). Options:

- **A:** Plumb the written value out of `applyFold` and send that (correct but invasive)
- **B:** Rename to `EventWatcher[E]` and document that watchers see events
- **C:** Have the watcher re-read from the engine after notification (simple but adds latency)

### Question 2: How deep should TTL go?

The current `WithTTL` is a no-op. Real TTL needs:

- **Engine-level:** Add `updated_at` column to SQLite, lazy eviction in memory engine
- **Store-level:** Track write timestamps, `SweepExpired(ctx)` method
- **Background:** Goroutine that periodically sweeps

Should we implement engine-level (correct but complex) or Store-level (simpler but coarser)?

### Question 3: What's the shipping bar for v4.1.0?

Given that Phase 1 is done and Phase 2 is partially done, what's the minimum for tagging?

- **A:** Ship now with Phase 1 fixes + honest TODO_LIST
- **B:** Finish Phase 2 wiring + Phase 3 tests first
- **C:** Complete all 50 items before tagging

---

## Technical Details

### Transaction API Architecture

```
RunInTx(fn)
  ├── e.txMu.Lock() (serialize concurrent tx)
  ├── e.db.BeginTx()
  ├── e.activeTx.Store(txC) (atomic pointer)
  ├── fn() ← all operations inside fn see activeTx
  │   ├── MapSet → e.xc() → txStmtCache.exec → tx.ExecContext
  │   ├── MapGet → e.xc() → txStmtCache.queryRow → tx.QueryRowContext
  │   ├── MapUpdate → e.txExec() != nil → readModifyWriteCached (no nested BEGIN)
  │   ├── CounterIncrement → e.txExec() != nil → xc().exec (no nested BEGIN)
  │   ├── PushdownMapScan → e.xd() → tx.QueryContext
  │   └── ... all other ops
  ├── e.activeTx.Store(nil)
  ├── fn err != nil → tx.Rollback()
  └── fn err == nil → tx.Commit()
```

### SQL Injection Fix Coverage

| Location                  | Before                                          | After                                          |
| ------------------------- | ----------------------------------------------- | ---------------------------------------------- |
| LayoutPlan.DDL()          | `CREATE TABLE %s` + raw column names            | `quoteIdent(table)` + `quoteIdent(column)`     |
| MigrateLayout             | `"ALTER TABLE " + table + " ADD COLUMN " + col` | `quoteIdent(table)` + `quoteIdent(col)`        |
| execPlannedSet            | `plan.Table` + raw column names                 | `quoteIdent(table)` + `quoteIdent(col)`        |
| appendPlannedFilter       | `f.Column` directly                             | `quoteIdent(f.Column)`                         |
| explainStandard           | `$.%s` with raw sort column                     | `jsonPath(sort.Column)` (escapes quotes)       |
| explainPlanned            | raw table + raw column                          | `quoteIdent(table)` + `quoteIdent(column)`     |
| All planned SELECT/DELETE | raw table names                                 | `quoteIdent(plan.Table)`                       |
| jsonPath                  | `"$.` + field                                   | `"$.` + `strings.ReplaceAll(field, "'", "''")` |

### Test Results (Phase 1 verified)

```
metaengine/v4: ok 1.488s (160+ tests pass)
  TestTransaction_CommitRollback ✅
  TestTransaction_StoreInTransaction ✅
  TestTransaction_MapUpdateInTx ✅
  TestTransactionInterface ✅
  TestHooks ✅ (with err parameter)
  TestLayoutPlanner_GeneratesCorrectDDL ✅ (with quoted identifiers)
  All 160 Ginkgo specs ✅
```

---

_Phase 1 is solid. Phase 2 needs build verification and read-path wiring. The transaction API finally works._
