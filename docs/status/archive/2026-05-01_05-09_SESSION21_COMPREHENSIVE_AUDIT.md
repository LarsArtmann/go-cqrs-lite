# Session 21 — Comprehensive Audit & Execution Plan

**Date:** 2026-05-01 05:09 CEST
**Branch:** master (1 commit ahead of origin)
**Scope:** Full codebase audit with library/SDK-first mindset

---

## A. FULLY DONE ✅

| Item                                          | Evidence                                                            | Session |
| --------------------------------------------- | ------------------------------------------------------------------- | ------- |
| Storage module unit tests (79.8% coverage)    | `storage/event_store_test.go` (489 lines, 13 tests with go-sqlmock) | 18-20   |
| Storage JSON v1→v2 migration                  | `storage/helpers.go` uses `go-json-experiment/json` exclusively     | 18-20   |
| Key separator unification (`:` everywhere)    | FakeStore, MemoryStore, MemorySnapshotStore all use `:`             | 18      |
| Storage in CI (flake.nix testModules)         | `"storage"` in testModules array                                    | 18      |
| Storage helpers extracted (file < 250 lines)  | `event_store.go` (250) + `helpers.go` (122)                         | 18-20   |
| `ProjectionRunner` interface removed          | Only `InMemoryRunner` remains                                       | 18-20   |
| Coverage recovery: `core/event` 86.7% → 96.3% | Handle/subscribesTo/CatalogCore/ProjectionFunc tests added          | 18-20   |
| Coverage: `core/pkg/id` 92.9% → 100%          | Ptr/FromPtr tests added                                             | 18-20   |
| Coverage: `memory` 94.9% → 99.0%              | CheckpointStore tests added                                         | 18-20   |
| `FakeCheckpointStore` in testhelpers          | Shared fake for projection tests                                    | 18-20   |
| Aggregate nil snapshot guard                  | Repository checks for nil snapshot state                            | 20      |
| ErrConcurrencyConflict unified                | Storage uses `event.ErrVersionConflict` alias                       | 20      |
| Example catalog extraction                    | `example/user/catalog.go` separated from `main.go`                  | 20      |
| example/user in CI build                      | `flake.nix` build app includes `examplePaths`                       | 20      |
| Aggregate options extracted                   | `options.go` with `SnapshotStrategy`, `EveryNEvents`, `With*`       | 20      |
| Golden files regenerated                      | asyncapi JSON/YAML, eventcatalog config/package                     | 20      |
| UpcasterRegistry `>=` → `==` fix              | `upcaster_registry.go:53` uses exact version match                  | 16      |

---

## B. PARTIALLY DONE ⚠️

| Item                      | Status                     | What's Left                                                                                                                                                                  |
| ------------------------- | -------------------------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| **FEATURES.md**           | Partially updated          | Storage still marked 🔴 BROKEN with stale issues (metadata discarded, zero tests, no Close) — ALL FIXED. Upcaster `>=` bug listed but already fixed. Coverage numbers stale. |
| **AGENTS.md**             | Partially updated          | Coverage table still shows old numbers. Module list missing `storage`. "Known Issues" updated with library context but some items already fixed.                             |
| **Storage test coverage** | 79.8%                      | Missing: error paths in scanEvents (bad ULID parse, bad metadata unmarshal, rows.Err()), Close() double-close, concurrent access                                             |
| **testhelpers in CI**     | Missing from `testModules` | Has `go.mod`, has code (342 lines), but not in flake.nix test/lint/vet/coverage                                                                                              |
| **Golden file commits**   | Uncommitted changes        | Golden test output differs from committed files (formatting changes)                                                                                                         |

---

## C. NOT STARTED 📋

| Item                                                                                      | Impact  | Effort |
| ----------------------------------------------------------------------------------------- | ------- | ------ |
| Remove stale binary `example/user/user` (9.8MB)                                           | LOW     | 1 min  |
| Consolidate `CatalogBuilder` on top of `Registry`                                         | HIGH    | 90 min |
| Storage: add `SQLEventStoreOption` config (table name, schema prefix) or remove dead type | MEDIUM  | 20 min |
| Storage: document that `Close()` owns the `*sql.DB` lifecycle                             | LOW     | 5 min  |
| Storage: Postgres-specific DDL comment ("compatible with any SQL" is misleading)          | LOW     | 5 min  |
| example/user: add smoke test (`TestExampleRuns`)                                          | MEDIUM  | 45 min |
| `testhelpers/fakes.go` at 342 lines — exceeds 250 line limit                              | MEDIUM  | 20 min |
| `catalog/internal/cattest/helpers.go` at 330 lines — exceeds 250 line limit               | LOW     | 20 min |
| Tagged releases (all modules at v0.0.0)                                                   | LOW     | 60 min |
| Watermill module                                                                          | PLANNED | —      |
| SQL SnapshotStore                                                                         | PLANNED | —      |

---

## D. TOTALLY FUCKED UP 💥

| Item                               | Detail                                                                                                                                                                                  |
| ---------------------------------- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| **FuzzParse case-sensitivity**     | `FuzzParse/e88fe8276d5dfaba` fails: roundtrip mismatch. ULID case-folding issue in `id.Parse`. Pre-existing, known since session 16. Not fixed.                                         |
| **Uncommitted doc flood**          | 16 modified doc files + AGENTS.md + FEATURES.md + BDD_TESTS_REVIEW.md uncommitted from multiple sessions. Commits should be clean per session but they accumulated.                     |
| **FEATURES.md lies about storage** | Still says "Metadata silently discarded", "Zero tests", "No Close" — ALL FIXED in sessions 18-20. Consumers reading this will think the module is broken when it's actually functional. |

---

## E. WHAT WE SHOULD IMPROVE 🔧

### Type Model Improvements

1. **`event.Store` interface lacks `Close()`** — `SQLEventStore` has `Close()` but it's not part of the interface. Consumers can't generically close stores. Consider adding to interface or documenting that implementors may implement `io.Closer`.

2. **`event.Metadata` has no JSON struct tags** — Relies on `go-json-experiment/json` auto-mapping, which works but is fragile. If field names change, storage serialization breaks silently. Should have explicit `json:"correlationId"` etc. tags (v2 compatible).

3. **`event.Version` is an interface, not a concrete type** — `Version.Int()` is the only method. Could be a simple `int` type alias with methods. The interface adds indirection for no gain — only one implementation exists (`eventVersion` in `event.go`). For a library, a concrete type would be clearer for consumers.

4. **`SQLEventStoreOption` is declared but has zero concrete options** — Dead type. Either add options (table name, logger, schema prefix) or remove it. Having a dead option type in the public API is confusing.

5. **Duplicate insert query string** in `event_store.go` — Save and AppendBatch have identical `INSERT INTO events...` strings. Extract to a constant.

### Architecture Improvements

6. **CatalogBuilder vs Registry split brain** — Two accumulators for the same `Catalog` type. `CatalogBuilder` reimplements all of Registry's logic minus thread safety. Should be unified: `CatalogBuilder` wraps `Registry` internally.

7. **`testhelpers/fakes.go` at 342 lines** — Five fake implementations in one file. Split by concern: `fake_store.go`, `fake_bus.go`, `fake_snapshot.go`, `fake_outbox.go`, `fake_checkpoint.go`.

8. **`memory/helpers.go` exports `streamKey` but testhelpers hardcodes the separator** — Should import and share the function, or define a shared constant.

### Library Improvements (Consider Established Libraries)

9. **`go-sqlmock` for storage tests** ✅ Already done — good choice.

10. **Consider `jmoiron/sqlx` or `volatiletech/sqlboiler`** for storage\*\* — Current raw SQL is correct but verbose. For a library, raw `database/sql` is actually the RIGHT choice (no extra deps, maximum consumer flexibility). Keep as-is.

11. **Consider `oklog/ulid` for event ordering verification** — Already using it ✅.

12. **No structured logging dependency** — Middleware accepts a `Logger` interface. This is correct for a library. Keep as-is.

---

## F. Top #25 Things to Get Done Next

Sorted by (impact × urgency) / effort:

| #   | Task                                                                                        | Impact | Effort | Status   |
| --- | ------------------------------------------------------------------------------------------- | ------ | ------ | -------- |
| 1   | **Fix FEATURES.md: update storage from 🔴 to ⚠️ PARTIALLY_FUNCTIONAL, remove stale issues** | HIGH   | 10min  | DO NOW   |
| 2   | **Fix FEATURES.md: remove stale upcaster `>=` bug (already fixed)**                         | HIGH   | 2min   | DO NOW   |
| 3   | **Add `testhelpers` to flake.nix testModules**                                              | HIGH   | 2min   | DO NOW   |
| 4   | **Delete stale binary `example/user/user`**                                                 | LOW    | 1min   | DO NOW   |
| 5   | **Commit all uncommitted doc changes**                                                      | HIGH   | 5min   | DO NOW   |
| 6   | **Update AGENTS.md coverage table with current numbers**                                    | MEDIUM | 5min   | DO NOW   |
| 7   | **Add explicit JSON tags to `event.Metadata`**                                              | HIGH   | 15min  | NEXT     |
| 8   | **Extract duplicate INSERT query to constant in storage**                                   | LOW    | 5min   | NEXT     |
| 9   | **Add storage error path tests (bad parse, bad metadata, rows.Err())**                      | HIGH   | 30min  | NEXT     |
| 10  | **Split `testhelpers/fakes.go` into per-fake files**                                        | MEDIUM | 20min  | NEXT     |
| 11  | **Remove or implement `SQLEventStoreOption`**                                               | MEDIUM | 15min  | NEXT     |
| 12  | **Document `Close()` DB ownership in storage godoc**                                        | MEDIUM | 5min   | NEXT     |
| 13  | **Fix Postgres-specific DDL comment**                                                       | LOW    | 5min   | NEXT     |
| 14  | **Consolidate `CatalogBuilder` on top of `Registry`**                                       | HIGH   | 90min  | LATER    |
| 15  | **Add `example/user` smoke test**                                                           | MEDIUM | 45min  | LATER    |
| 16  | **Fix FuzzParse case-sensitivity**                                                          | MEDIUM | 30min  | LATER    |
| 17  | **Split `cattest/helpers.go` under 250 lines**                                              | LOW    | 20min  | LATER    |
| 18  | **Add `event.Version` as concrete type (breaking change)**                                  | HIGH   | 60min  | DEFERRED |
| 19  | **Consider `io.Closer` on `event.Store` (breaking change)**                                 | HIGH   | 30min  | DEFERRED |
| 20  | **Tagged releases for all modules**                                                         | LOW    | 60min  | DEFERRED |
| 21  | **Watermill pub/sub module**                                                                | HIGH   | 4hr    | PLANNED  |
| 22  | **SQL SnapshotStore**                                                                       | MEDIUM | 2hr    | PLANNED  |
| 23  | **SQL CheckpointStore**                                                                     | MEDIUM | 1hr    | PLANNED  |
| 24  | **Outbox background publisher**                                                             | MEDIUM | 2hr    | PLANNED  |
| 25  | **Saga/Process Manager**                                                                    | HIGH   | 8hr    | PLANNED  |

---

## G. Top #1 Question I Cannot Figure Out Myself

**Should `event.Store` interface get a `Close()` method (or compose `io.Closer`)?**

Arguments FOR:

- `SQLEventStore` has `Close()` that releases the `*sql.DB`
- `MemoryStore`, `MemoryBus`, `MemorySnapshotStore` all have `Close()` via lifecycle
- Consumers managing store lifecycles generically need this

Arguments AGAINST:

- Breaking change for any external implementors of `event.Store`
- Not all stores need closing (e.g., a store wrapping a shared connection pool)
- `io.Closer` composition means consumers must type-assert to close

**This is a design decision that affects the public API contract.** I cannot make it alone because it's a breaking change for consumers.

---

## Current Coverage (Live)

| Package                | Coverage        | Status                  |
| ---------------------- | --------------- | ----------------------- |
| `core/command`         | 100.0%          | ✅                      |
| `core/query`           | 100.0%          | ✅                      |
| `core/event`           | 96.3%           | ✅                      |
| `core/aggregate`       | 95.9%           | ✅                      |
| `core/pkg/id`          | 100.0%          | ✅ (FuzzParse case bug) |
| `core/pkg/dispatcher`  | 100.0%          | ✅                      |
| `memory`               | 99.0%           | ✅                      |
| `catalog`              | 94.4%           | ✅                      |
| `catalog/adapters`     | 98.8%           | ✅                      |
| `catalog/asyncapi`     | 97.9%           | ✅                      |
| `catalog/eventcatalog` | 95.5%           | ✅                      |
| `middleware`           | 99.4%           | ✅                      |
| `storage`              | 79.8%           | ⚠️                      |
| `testhelpers`          | 0.0% (no tests) | ⚠️                      |

---

## Test Failures

| Test                         | Package       | Issue                                               |
| ---------------------------- | ------------- | --------------------------------------------------- |
| `FuzzParse/e88fe8276d5dfaba` | `core/pkg/id` | ULID case-folding roundtrip mismatch. Pre-existing. |

---

## Uncommitted Changes Summary

20 modified files (mostly docs + formatting). 1 untracked fuzz corpus file. 1 stale binary.
