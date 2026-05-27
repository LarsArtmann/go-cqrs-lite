# Session 95 — Comprehensive Status Report

**Date:** 2026-05-23 04:30  
**Session Focus:** Code deduplication sweep (art-dupl)  
**Branch:** master  
**Commits Since Last Report:** 8 (Sessions 93–95)

---

## Executive Summary

All 10 production modules + 2 example modules passing. **Zero lint. Zero test failures.** Clone count reduced from 151 → 141 groups. Production code deduplication: catalog copy helpers, `execDDL`, `CheckVersionConflict`, decider `loadFromStore` delegation. Test deduplication: shared `storeTestConfig` + reusable test suite for storage backends (Pebble/SQLite/Turso).

---

## a) FULLY DONE ✅

### This Session (95)

| What                                     | Files Changed                                                      | Impact                                                                                                                                                                     |
| ---------------------------------------- | ------------------------------------------------------------------ | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Catalog `copy*Ptr` delegation            | `catalog/registry.go`                                              | 30 lines eliminated, `copyServicePtr`/`copyDomainPtr`/`copyChannelPtr` now delegate to existing helpers                                                                    |
| `execDDL` extraction                     | `storage/sqlite_helpers.go`                                        | `SQLiteInitSchema` + `PostgresInitSchema` share DDL execution loop                                                                                                         |
| `CheckVersionConflict` helper            | `core/event/types.go`, `memory/store.go`, `storage/pebble_save.go` | Shared optimistic concurrency check across 2 stores                                                                                                                        |
| Decider `loadFromStore` → `loadByEvents` | `core/decider/load.go`                                             | 20 lines eliminated, single implementation of load-fold logic                                                                                                              |
| Storage shared test suite                | `storage/store_testsuite_test.go` + 4 test files                   | ~200 lines eliminated, shared `testEventStore_SaveAndLoad`, `ConcurrencyConflict`, `AppendBatch`, `LoadFromVersion`, `Delete`, `MetadataRoundtrip`, `testOutbox_Roundtrip` |
| Golden file refresh                      | `catalog/asyncapi/testdata/`, `catalog/eventcatalog/testdata/`     | Updated for registry.go changes                                                                                                                                            |

### Prior Sessions (93–94)

| What                                           | Impact                                   |
| ---------------------------------------------- | ---------------------------------------- |
| Zero lint across ALL 10 modules                | 0 golangci-lint issues                   |
| Decider dual-wrap fix                          | Single `%w` wrapping in `decider.go`     |
| Registry deterministic Build                   | Sorted map iteration via `slices.Sorted` |
| Testhelpers 10→80.3% coverage                  | FailingQuery, Panic\*, FakeStore setters |
| Catalog quality sweep (Session 93)             | Pointer escapes, deep copy, validation   |
| Docserver mustStaticFS extraction (Session 94) | Panic-pattern helper                     |
| Caseutil 100%, schemautil improved             | Edge-case tests                          |

### Ongoing Complete Items

- **All tests green** — 27 packages, 0 failures
- **Zero lint** — 9 production modules + test modules
- **Coverage** — See table below

---

## b) PARTIALLY DONE 🔧

| Item                  | Status                                                                                        | What's Left                                                                                                                             |
| --------------------- | --------------------------------------------------------------------------------------------- | --------------------------------------------------------------------------------------------------------------------------------------- |
| Code deduplication    | 151→141 clone groups (reduced by 10)                                                          | Remaining 141 are mostly: Go interface implementations (unavoidable), single-line assertions, 2-clone groups below extraction threshold |
| Storage test coverage | 89.2%                                                                                         | Error paths in Pebble deserialization, some dialect code                                                                                |
| `core/event` coverage | 86.1% (dropped from 90.9% due to `CheckVersionConflict` not being tested directly)            | Need test for new `CheckVersionConflict` function                                                                                       |
| Testhelpers coverage  | 80.3%                                                                                         | Some FakeBus/FakeSnapshot methods still uncovered                                                                                       |
| AGENTS.md accuracy    | Coverage table updated, but missing `sync/` details, `catalog/openapi/`, `catalog/docserver/` | Needs full refresh                                                                                                                      |

---

## c) NOT STARTED 📋

### HIGH Priority

1. **Fix query.Handler returns `any`** — Generic `TypedHandler[T]` returning `(T, error)` (breaking change, design doc exists)
2. **Publish go-composable-business-types** — #1 blocker for external adoption
3. **Add global TransactionID branded type** — Breaking change, deferred to v2
4. **io.Closer removal from core interfaces** — Breaking change
5. **Add catalog diff/breaking-change detection tool**
6. **Add high-level test utilities** — AggregateTester, ProjectionTester, BusTester

### MEDIUM Priority

7. Fix Pebble Store concurrent write overwrite
8. Fix outbox transaction co-participation (SQLOutbox.Append + SQLEventStore.Save separate transactions)
9. Fix OutboxPublisher split-brain (cancel stays non-nil after Close)
10. Fix aggregate snapshot with nil state when codec is nil
11. Add slog.Warn for corrupt IDs in Pebble deserialization
12. Move example/todo to own repository
13. Design ADR for outbox transaction co-participation
14. Add PostgreSQL integration tests with testcontainers
15. Add context cancellation to SQLOutbox
16. Extract storage table name constants
17. Add command metadata (CorrelationID, CausationID, UserID, RequestID, Custom)
18. Split core/event god-package into sub-packages
19. Build catch-up projection runner
20. Add retry/dead-letter mechanism for projections

---

## d) TOTALLY FUCKED UP 💥

| Item                               | Problem                                                                                                          | Severity                                                      |
| ---------------------------------- | ---------------------------------------------------------------------------------------------------------------- | ------------------------------------------------------------- |
| `core/event` coverage dropped      | 90.9% → 86.1% — new `CheckVersionConflict` function has no direct test                                           | LOW — function is tested indirectly via memory/storage stores |
| `sync/` module is a ghost          | Zero external consumers, shadows stdlib name, questionable value                                                 | MEDIUM — consider deleting or renaming                        |
| 141 clone groups remain            | Most are inherent to Go (interface impls, method signatures), but some test boilerplate could still be extracted | LOW                                                           |
| `example/todo` external dependency | CQRS-HTMX dep creates build fragility                                                                            | MEDIUM                                                        |

---

## e) WHAT WE SHOULD IMPROVE 🔍

1. **Test `CheckVersionConflict` directly** — New helper in `core/event/types.go` needs its own test
2. **Event package coverage recovery** — 90.9% → 86.1% needs investigation and test additions
3. **AGENTS.md full refresh** — Missing sync/, openapi, docserver, example/todo, storage Dialect
4. **Push git tags** — 8 tags LOCAL ONLY, blocks external consumers
5. **CI coverage gate** — No minimum coverage enforcement in CI
6. **Delete or formalize `sync/` module** — Ghost module with no consumers
7. **Test deduplication for middleware** — retry_test.go, retry_event_test.go, retry_query_test.go have identical test structures (identified but not yet extracted)
8. **Test deduplication for decider** — `newSnapshotTestRepo` helper identified but not extracted (10 occurrences, ~100 lines)
9. **Coverage gate at 80%** — Some packages dip below (core/event 86%, schemautil 84%)
10. **Documentation of time-travel API** — LoadToVersion/LoadToTimestamp not documented

---

## f) Top #25 Things We Should Get Done Next

Sorted by impact × effort × customer-value:

| #   | Task                                                                    | Impact | Effort        | Category     |
| --- | ----------------------------------------------------------------------- | ------ | ------------- | ------------ |
| 1   | Test `event.CheckVersionConflict` directly (new helper)                 | HIGH   | LOW (12min)   | Quality      |
| 2   | Recover `core/event` coverage to 90%+                                   | HIGH   | LOW (30min)   | Quality      |
| 3   | AGENTS.md full refresh — add sync/, openapi, docserver, storage Dialect | HIGH   | MED (60min)   | Docs         |
| 4   | Extract decider `newSnapshotTestRepo` helper (10 occurrences)           | MED    | LOW (20min)   | Dedup        |
| 5   | Extract middleware retry test helper (12 identical tests)               | MED    | MED (45min)   | Dedup        |
| 6   | Push 8 local git tags to remote                                         | HIGH   | LOW (5min)    | Release      |
| 7   | Bump testhelpers to v1.2.0 (fix event.Version breaking change)          | HIGH   | MED (30min)   | Release      |
| 8   | Delete or formalize `sync/` module                                      | MED    | LOW (15min)   | Cleanup      |
| 9   | Add CI coverage gate (80% minimum)                                      | MED    | LOW (15min)   | CI           |
| 10  | Fix Pebble Store concurrent write overwrite                             | HIGH   | MED (45min)   | Bug          |
| 11  | Fix outbox transaction co-participation ADR                             | HIGH   | MED (60min)   | Design       |
| 12  | Add catalog diff/breaking-change detection                              | MED    | MED (60min)   | Feature      |
| 13  | Trim AGENTS.md from 827→<400 lines                                      | MED    | MED (45min)   | Docs         |
| 14  | Remove replace directives from go.mod files                             | MED    | LOW (20min)   | Cleanup      |
| 15  | Add GOWORK=off CI matrix job                                            | MED    | LOW (15min)   | CI           |
| 16  | Fix OutboxPublisher split-brain (cancel non-nil after Close)            | MED    | LOW (20min)   | Bug          |
| 17  | Add slog.Warn for corrupt Pebble IDs                                    | LOW    | LOW (10min)   | Quality      |
| 18  | Document time-travel API (LoadToVersion/LoadToTimestamp)                | MED    | LOW (20min)   | Docs         |
| 19  | Move example/todo to own repository                                     | MED    | MED (30min)   | Cleanup      |
| 20  | Split core/event god-package into sub-packages                          | HIGH   | HIGH (120min) | Architecture |
| 21  | Fix query.Handler returns `any` → TypedHandler[T] (breaking)            | HIGH   | HIGH (120min) | API          |
| 22  | Add high-level test utilities (AggregateTester, ProjectionTester)       | MED    | HIGH (90min)  | Feature      |
| 23  | Add PostgreSQL integration tests with testcontainers                    | MED    | HIGH (90min)  | Quality      |
| 24  | Build catch-up projection runner                                        | MED    | HIGH (120min) | Feature      |
| 25  | Publish go-composable-business-types as Go module                       | HIGH   | HIGH (120min) | Release      |

---

## Coverage Summary

| Package                       | Coverage | Trend |
| ----------------------------- | -------- | ----- |
| `core/query`                  | 100.0%   | →     |
| `core/pkg/dispatcher`         | 100.0%   | →     |
| `middleware`                  | 100.0%   | →     |
| `catalog/adapters`            | 100.0%   | →     |
| `catalog/internal/caseutil`   | 100.0%   | →     |
| `memory`                      | 99.6%    | →     |
| `core/pkg/id`                 | 98.1%    | →     |
| `catalog`                     | 96.8%    | ↑     |
| `core/aggregate`              | 95.9%    | →     |
| `catalog/d2`                  | 95.0%    | →     |
| `core/command`                | 94.7%    | →     |
| `projection`                  | 94.4%    | →     |
| `catalog/openapi`             | 94.4%    | →     |
| `catalog/asyncapi`            | 93.7%    | →     |
| `core/decider`                | 93.6%    | ↓     |
| `catalog/eventcatalog`        | 91.3%    | →     |
| `catalog/docserver`           | 90.1%    | ↑     |
| `sync`                        | 90.2%    | →     |
| `storage`                     | 89.2%    | ↑     |
| `core/event`                  | 86.1%    | ↓     |
| `catalog/internal/schemautil` | 84.2%    | →     |
| `testhelpers`                 | 80.3%    | ↑     |

---

## Code Metrics

| Metric              | Value                     |
| ------------------- | ------------------------- |
| Production Go files | 181                       |
| Test Go files       | 137                       |
| Production LOC      | 16,092                    |
| Test LOC            | 32,830                    |
| Total LOC           | 48,922                    |
| Clone groups        | 141                       |
| Lint issues         | 0                         |
| Test failures       | 0                         |
| Modules             | 10 production + 2 example |

---

## Build & CI Status

- **Build:** ✅ Passing (`nix run .#build`)
- **Test:** ✅ All 27 packages green
- **Lint:** ✅ Zero issues across all 10 modules
- **Format:** ✅ `nix fmt` clean
- **Flake check:** ✅ Passing
- **CI:** GitHub Actions configured (`ci.yml`)

---

## g) Top #1 Question I Cannot Figure Out Myself

**What is the strategic intent for `sync/`?** It has zero internal consumers, shadows the stdlib `sync` package name, and has no integration with any other module. The TODO list mentions CRDT primitives, but there's no concrete plan. Should we:

1. **Delete it entirely** — it's dead weight for a CQRS library?
2. **Rename it** to `crdt/` or `distributed/` and flesh it out?
3. **Keep it as-is** — it's a standalone utility consumers can import?

This decision impacts whether we invest more time in it or remove it to reduce the module surface area.
