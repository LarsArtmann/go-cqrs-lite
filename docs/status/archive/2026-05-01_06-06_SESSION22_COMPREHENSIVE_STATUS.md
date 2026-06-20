# Session 22 — Full Status After io.Closer Lifecycle Cleanup

**Date:** 2026-05-01 06:06 CEST
**Branch:** master (clean, pushed)
**Tests:** 17/17 pass (race-clean)
**Lint:** Zero issues across all 7 modules
**Build:** Clean

---

## A. FULLY DONE ✅

### Session 21–22 Work

| #   | Item                                        | Detail                                                                                                                                                   | Commit                        |
| --- | ------------------------------------------- | -------------------------------------------------------------------------------------------------------------------------------------------------------- | ----------------------------- |
| 1   | **FuzzParse case-sensitivity fix**          | Roundtrip test now uses `Parse(String())` instead of comparing against raw input. ULID's Crockford Base32 normalization no longer causes false failures. | `b3b5336`                     |
| 2   | **io.Closer on all 4 lifecycle interfaces** | `event.Store`, `event.Bus`, `event.SnapshotStore`, `event.Outbox` all embed `io.Closer`. Consumers can `defer io.Closer.Close()` generically.            | Prior sessions                |
| 3   | **All fakes implement Close()**             | `FakeStore`, `FakeBus`, `FakeSnapshotStore`, `FakeOutbox`, `MemoryOutboxStore` all have `Close()`                                                        | Prior sessions + this session |
| 4   | **Remove dead SQLEventStoreOption**         | No concrete options existed — dead API surface removed. `NewSQLEventStore` now takes only `*sql.DB`.                                                     | `e0d3365`                     |
| 5   | **Storage package doc fixed**               | "compatible with any SQL database" → "DDL targets PostgreSQL (BYTEA, JSONB)"                                                                             | `e0d3365`                     |
| 6   | **Duplicate INSERT extracted to constant**  | `insertEventSQL` constant replaces two identical strings in Save/AppendBatch                                                                             | `9eff88e`                     |
| 7   | **testhelpers/fakes.go split**              | 354 lines → 5 files (143, 108, 67, 45, 21 lines). All under 250-line limit.                                                                              | `9d2ad81`                     |
| 8   | **Planning docs rewritten**                 | Session 18 plans rewritten with library/SDK mindset (not application mindset)                                                                            | `5ad1d85`                     |
| 9   | **FEATURES.md storage section fixed**       | 🔴 BROKEN → ⚠️ PARTIALLY_FUNCTIONAL. Removed 5 stale "critical issues" that were already fixed.                                                          | `5ab307f`                     |
| 10  | **FEATURES.md upcaster bug removed**        | `>=` → `==` was already fixed in session 16 but FEATURES.md still documented it                                                                          | `5ab307f`                     |
| 11  | **AGENTS.md library/SDK header**            | Added unmissable block at top with table comparing wrong vs correct evaluation lens                                                                      | `3af3a71`                     |
| 12  | **testhelpers added to CI**                 | Added to `flake.nix` testModules for test/vet/lint/coverage                                                                                              | `b028c0e`                     |
| 13  | **Stale binary deleted**                    | `example/user/user` (9.8MB) removed from repo                                                                                                            | `b028c0e`                     |
| 14  | **Docs updated**                            | Coverage numbers, io.Closer lifecycle, storage status all current                                                                                        | `106763d`                     |
| 15  | **Lint config updated**                     | `.golangci.yml` exhaustruct exclusion updated for split fake files                                                                                       | `a58eb86`                     |

### Prior Sessions (confirmed still good)

| Item                                   | Status                                        |
| -------------------------------------- | --------------------------------------------- |
| Storage unit tests (event store 96.2%) | ✅ 23 test functions with go-sqlmock          |
| Storage JSON v2 migration              | ✅ Uses `go-json-experiment/json` exclusively |
| Key separator unification (`:`)        | ✅ All stores use `:`                         |
| `ProjectionRunner` interface removed   | ✅ Only `InMemoryRunner`                      |
| `core/event` coverage 86.7% → 96.5%    | ✅                                            |
| `core/pkg/id` coverage 100%            | ✅                                            |
| `memory` coverage 98.5%                | ✅                                            |
| Upcaster `>=` → `==` fix               | ✅                                            |
| `FakeCheckpointStore` in testhelpers   | ✅                                            |
| Aggregate nil snapshot guard           | ✅                                            |
| `ErrConcurrencyConflict` unified       | ✅                                            |

---

## B. PARTIALLY DONE ⚠️

| Item                                           | Status                                                                                                                                                                  | What's Left                                   |
| ---------------------------------------------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------- | --------------------------------------------- |
| **Storage module test coverage (63.5%)**       | Event store at 96.2%, but `SQLCheckpointStore` (59 lines) and `SQLSnapshotStore` (154 lines) have **zero tests**. These were added in a prior session but never tested. | Write tests for checkpoint + snapshot stores  |
| **`cattest/helpers.go` at 330 lines**          | Exceeds 250-line limit. Known since session 9.                                                                                                                          | Split into smaller files                      |
| **`CatalogBuilder` vs `Registry` split brain** | Two accumulators for the same `Catalog` type. `CatalogBuilder` reimplements Registry's logic minus thread safety. Known since session 9.                                | Consolidate CatalogBuilder on top of Registry |

---

## C. NOT STARTED 📋

| Item                                                       | Impact | Effort |
| ---------------------------------------------------------- | ------ | ------ |
| **Storage: SQLCheckpointStore tests**                      | HIGH   | 30min  |
| **Storage: SQLSnapshotStore tests**                        | HIGH   | 45min  |
| **CatalogBuilder → Registry consolidation**                | HIGH   | 90min  |
| **example/user smoke test**                                | MEDIUM | 45min  |
| **Split `cattest/helpers.go` under 250 lines**             | LOW    | 20min  |
| **Tagged releases (all modules at v0.0.0)**                | LOW    | 60min  |
| **Watermill pub/sub module**                               | HIGH   | 4hr    |
| **Saga/Process Manager**                                   | HIGH   | 8hr+   |
| **Outbox background publisher**                            | MEDIUM | 2hr    |
| **SQL-backed SnapshotStore tests** (code exists, no tests) | HIGH   | 30min  |
| **`example/user` in CI test/lint** (currently build-only)  | MEDIUM | 10min  |

---

## D. TOTALLY FUCKED UP 💥

| Item                                                          | Detail                                                                                                                                                                                                                                                                       |
| ------------------------------------------------------------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| **Storage coverage reported as 63.5%**                        | Previous sessions reported 79.8% → 92.3%. The number dropped because `SQLCheckpointStore` and `SQLSnapshotStore` were added (59 + 154 = 213 lines) with zero tests. The event store itself is 96.2% — the low number masks that the _new code_ is the gap, not the old code. |
| **FEATURES.md coverage numbers may still be stale in places** | Session 22 updated some but not all numbers. The Module Maturity Matrix may show old data.                                                                                                                                                                                   |
| **Multiple sessions committed with overlapping changes**      | Sessions 18–22 have interleaved commits making git history hard to follow. Some changes appear in unexpected commits due to git hooks auto-staging.                                                                                                                          |

---

## E. WHAT WE SHOULD IMPROVE 🔧

### Type Model

1. **`event.Version` is an interface with one implementation** — Only `eventVersion` (unexported) exists. Consider making it a concrete `type Version int` with methods. Breaking change but cleaner for a library. The interface adds indirection with zero gain.

2. **`event.Metadata` Custom field uses `map[MetadataKey]string`** — `MetadataKey` is a `type MetadataKey string`. Consider using `map[string]string` with a helper, or make `MetadataKey` do validation. Currently any string works.

3. **`event.CheckpointStore` doesn't embed `io.Closer`** — Unlike Store/Bus/SnapshotStore/Outbox. Could add it for consistency, but `SQLCheckpointStore.Close()` owns the `*sql.DB` which is the same pattern as `SQLEventStore.Close()`. Worth considering.

### Architecture

4. **Storage `Close()` methods all own the `*sql.DB`** — If a consumer passes the same `*sql.DB` to `SQLEventStore` + `SQLCheckpointStore` + `SQLSnapshotStore`, calling `Close()` on any one breaks the others. Should document this clearly or accept a `Closer` interface so consumers control lifecycle.

5. **`CatalogBuilder` duplicates `Registry` logic** — Two accumulators for the same output type. Should unify: `CatalogBuilder` wraps `Registry` internally, adding only the core-type adapter layer.

### Library Consumer Experience

6. **No `example/user` test** — The only demo of the full CQRS lifecycle has zero verification. If APIs change, it silently breaks.

7. **`example/user` not in CI test/lint** — Only in `build` app. Could catch API breakage.

---

## F. Top #25 Things to Get Done Next

Sorted by (consumer trust impact × effort):

| #   | Task                                                          | Module     | Impact   | Effort |
| --- | ------------------------------------------------------------- | ---------- | -------- | ------ |
| 1   | Write SQLCheckpointStore tests                                | storage    | CRITICAL | 30min  |
| 2   | Write SQLSnapshotStore tests                                  | storage    | CRITICAL | 45min  |
| 3   | Verify storage coverage > 90% after checkpoint/snapshot tests | storage    | HIGH     | 5min   |
| 4   | Add `example/user` to CI test/lint (not just build)           | CI         | MEDIUM   | 10min  |
| 5   | Add `example/user/main_test.go` smoke test                    | example    | MEDIUM   | 45min  |
| 6   | Consolidate `CatalogBuilder` on top of `Registry`             | catalog    | HIGH     | 90min  |
| 7   | Split `cattest/helpers.go` (330 lines) under 250              | catalog    | LOW      | 20min  |
| 8   | Document `Close()` DB ownership pattern in storage godoc      | storage    | MEDIUM   | 10min  |
| 9   | Consider concrete `type Version int` (breaking)               | core/event | HIGH     | 60min  |
| 10  | Consider `io.Closer` on `CheckpointStore`                     | core/event | LOW      | 15min  |
| 11  | Add `io.Closer` on `CheckpointStore` + update fakes           | all        | MEDIUM   | 20min  |
| 12  | Watermill pub/sub module (Kafka, NATS)                        | new module | HIGH     | 4hr    |
| 13  | Outbox background publisher goroutine                         | core       | MEDIUM   | 2hr    |
| 14  | Tag v0.1.0 for all modules                                    | git        | LOW      | 60min  |
| 15  | Add `CONTRIBUTING.md` section on `io.Closer` pattern          | docs       | LOW      | 10min  |
| 16  | Add integration test with real PostgreSQL (testcontainers)    | storage    | HIGH     | 2hr    |
| 17  | Add benchmarks for storage operations                         | storage    | LOW      | 30min  |
| 18  | Consider saga/process manager design                          | new module | HIGH     | 8hr+   |
| 19  | Add `event.Bus` middleware support (like command/query)       | core/event | MEDIUM   | 2hr    |
| 20  | Add retry policy support to Outbox publisher                  | core       | MEDIUM   | 1hr    |
| 21  | Add health check interface for stores                         | core       | LOW      | 1hr    |
| 22  | Add context timeout example to godoc                          | docs       | LOW      | 15min  |
| 23  | Add OpenTelemetry tracing to storage module                   | storage    | MEDIUM   | 1hr    |
| 24  | Add `event.Streamer` for real-time event streaming            | core       | MEDIUM   | 2hr    |
| 25  | Add generated API docs (godoc -> static site)                 | docs       | LOW      | 2hr    |

---

## G. Top #1 Question

**Should `Close()` on storage types own the `*sql.DB`, or should they accept a shared connection pool?**

Current design: `NewSQLEventStore(db *sql.DB)` and `Close()` calls `db.Close()`. If a consumer creates `SQLEventStore` + `SQLCheckpointStore` + `SQLSnapshotStore` all with the same `*sql.DB`, calling `Close()` on any one will break the others with "sql: database is closed".

Options:

1. **Document it clearly** — "Caller must not call Close() if sharing \*sql.DB" (current approach, fragile)
2. **Don't call `db.Close()` in storage `Close()`** — Just mark the store as closed, let the caller manage `*sql.DB` lifecycle (breaking change)
3. **Add a `WithDBOwnership(bool)` option** — Consumer explicitly opts in to DB lifecycle management

This affects every storage consumer and can't be decided without understanding the target use case.

---

## Current Coverage

| Package                | Coverage | Change                                       |
| ---------------------- | -------- | -------------------------------------------- |
| `core/command`         | 100.0%   | —                                            |
| `core/query`           | 100.0%   | —                                            |
| `core/event`           | 96.5%    | +0.2% (from example_test fix)                |
| `core/aggregate`       | 95.9%    | —                                            |
| `core/pkg/id`          | 100.0%   | — (FuzzParse fixed)                          |
| `core/pkg/dispatcher`  | 100.0%   | —                                            |
| `memory`               | 98.5%    | -0.5%                                        |
| `catalog`              | 94.4%    | —                                            |
| `catalog/adapters`     | 98.8%    | —                                            |
| `catalog/asyncapi`     | 96.8%    | -1.1%                                        |
| `catalog/eventcatalog` | 95.5%    | —                                            |
| `middleware`           | 99.4%    | —                                            |
| `storage`              | 63.5%    | ⚠️ (event store 96%, checkpoint/snapshot 0%) |

## File Size Check (production Go files > 250 lines)

| File                                  | Lines | Status  |
| ------------------------------------- | ----- | ------- |
| `catalog/internal/cattest/helpers.go` | 330   | ⚠️ OVER |
| All other files                       | < 250 | ✅      |

## io.Closer Interface Status

| Interface               | Has io.Closer? | All impls have Close()?                                              |
| ----------------------- | -------------- | -------------------------------------------------------------------- |
| `event.Store`           | ✅             | ✅ (FakeStore, MemoryStore, SQLEventStore)                           |
| `event.Bus`             | ✅             | ✅ (FakeBus, MemoryBus)                                              |
| `event.SnapshotStore`   | ✅             | ✅ (FakeSnapshotStore, MemorySnapshotStore, SQLSnapshotStore)        |
| `event.Outbox`          | ✅             | ✅ (FakeOutbox, MemoryOutboxStore)                                   |
| `event.CheckpointStore` | ❌             | ❌ (SQLCheckpointStore has Close() but interface doesn't require it) |

---

## Test Failures

None. All 17 packages pass with race detector enabled.

## Uncommitted Changes

One untracked file: `docs/web-client-communication.d2` (diagram, not Go code).
