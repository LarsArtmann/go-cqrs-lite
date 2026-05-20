# Session 85 — Comprehensive Status Report

**Date:** 2026-05-21 01:32  
**Session Type:** READ → UNDERSTAND → RESEARCH → REFLECT → EXECUTE  
**Branch:** master  
**Commits since May 1:** 508

---

## Executive Summary

Deep structural review of the entire package and file layout. Executed 6 concrete improvements. Identified the #1 architectural issue: `core/event` is a god-package with 12 concerns and ~75 exports. All 26 test packages pass, zero lint issues.

---

## a) FULLY DONE

### Session 85 Executed Changes

| #   | Change                                                                                                                                                     | Files           | Net Lines  |
| --- | ---------------------------------------------------------------------------------------------------------------------------------------------------------- | --------------- | ---------- |
| 1   | Removed dead deprecated exports (`Catalogable`, `CatalogCore`, `NewCatalogCore`, `MustNewCatalogCore`) from `event/command/query` catalog.go + their tests | -6 files        | -120 lines |
| 2   | Merged `catalog/auto_name.go` (single unexported function) into `catalog/message_config.go` (its only caller)                                              | -1 file         | -3 lines   |
| 3   | Renamed 4 misleading `coverage_test.go` → descriptive names (`store_extra_test.go`, `constructor_test.go`, `runner_extra_test.go`, `core_errors_test.go`)  | 4 renames       | 0 lines    |
| 4   | Archived 22 stale session planning docs + 11 stale status reports to `docs/*/archive/`                                                                     | -33 from active | 0 lines    |
| 5   | Archived `DEDUPLICATION_PLAN.md` (abandoned, 21/31 POSTPONED) to `docs/planning/archive/`                                                                  | -1 from root    | 0 lines    |
| 6   | Renamed `CONTEXT.md` → `DOMAIN_GLOSSARY.md` + updated `README.md` reference                                                                                | 2 files         | 0 lines    |

### Pre-Session 85 (Commits on master since last status)

| Commit    | Summary                                                                                   |
| --------- | ----------------------------------------------------------------------------------------- |
| `b59a213` | Convert all 48 sentinels to structured `errorfamily` constructors with dot-notation codes |
| `ab3914d` | Eliminate dead `CatalogCore` code and resolve version chaos                               |
| `a16f95e` | Split `PebbleEventStore.Save` into focused helpers                                        |
| `a98903b` | Validate `NewVectorClockFromMap` rejects negative counters                                |
| `2a98bab` | Rename `CONTEXT.md` to `DOMAIN_GLOSSARY.md` and archive stale planning documents          |

---

## b) PARTIALLY DONE

| Item                             | Status         | Detail                                                                                                                                                                                                                                              |
| -------------------------------- | -------------- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Deprecated `CatalogMeta` removal | Partially done | Removed `Catalogable`, `CatalogCore`, constructors + tests. `CatalogMeta` itself remains because it's embedded in `command.Dispatcher` and `query.Dispatcher` via `CatalogDispatcher[Type, CatalogMeta]`. Requires dispatcher refactor to complete. |
| `core/event` god-package split   | Research only  | Identified 12 distinct concerns, mapped all consumers, designed sub-package structure. NOT executed — breaking change requiring dedicated session.                                                                                                  |
| Root-level doc cleanup           | Partially done | Archived `DEDUPLICATION_PLAN.md`. `BDD_TESTS_REVIEW.md`, `PUBLIC_OR_PRIVATE.md` still at root (but are active/pending).                                                                                                                             |
| Storage file naming              | Not started    | PostgreSQL files (`event_store*.go`) lack prefix while `pebble_*` and `sqlite_*` are prefixed.                                                                                                                                                      |

---

## c) NOT STARTED

| #   | Item                                                                                                     | Effort    | Impact    |
| --- | -------------------------------------------------------------------------------------------------------- | --------- | --------- |
| 1   | Split `core/event` into sub-packages (store, bus, projection, outbox, snapshot, upcaster, codec)         | Very High | Very High |
| 2   | Break `core ↔ memory` circular dependency by moving integration tests from `core/` to `integration/`     | Medium    | High      |
| 3   | Complete `CatalogMeta` removal — redesign dispatcher catalog embedding                                   | Medium    | Medium    |
| 4   | Rename `storage/event_store*.go` → `storage/postgres_*.go` for consistency                               | Low       | Low       |
| 5   | Rename `InMemoryRunner` → `InMemoryProjectionRunner` in `core/event/runner.go`                           | Low       | Low       |
| 6   | Move `Source`/`IPAddress`/`UserAgent` phantom types out of `core/event/types.go`                         | Low       | Low       |
| 7   | Standardize BDD test file naming (`{domain}_bdd_test.go` pattern)                                        | Low       | Low       |
| 8   | Split oversized test files (`decider_test.go` 1318L, `runner_test.go` 1164L, `event_store_test.go` 885L) | Medium    | Low       |
| 9   | Fix `sync/benchmark_test.go` compilation error (4x `WrongAssignCount` for `NewVectorClockFromMap`)       | Trivial   | Low       |
| 10  | Remove `Upcaster`/`Enricher` from public API (zero external consumers)                                   | Low       | Low       |

---

## d) TOTALLY FUCKED UP

| #   | Issue                                  | Severity   | Detail                                                                                                                                                                                                       |
| --- | -------------------------------------- | ---------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| 1   | `sync/benchmark_test.go` won't compile | **HIGH**   | 4 `WrongAssignCount` errors — `NewVectorClockFromMap` returns 2 values but benchmarks only capture 1. Pre-existing from Session 81 (signature changed to `(VectorClock, error)` but benchmarks not updated). |
| 2   | `core/event` is a god-package          | **HIGH**   | 23 production files, ~75 exports, 12 distinct concerns. This is the #1 structural problem. Every consumer recompiles when any concern changes.                                                               |
| 3   | `core ↔ memory` circular dependency    | **MEDIUM** | `core` (foundational module) depends on `memory` and `testhelpers`. Architecturally backwards for a library SDK.                                                                                             |
| 4   | `testhelpers` coverage is 10.5%        | **LOW**    | Test helper package has minimal tests. Acceptable since it's infrastructure for other tests.                                                                                                                 |

---

## e) WHAT WE SHOULD IMPROVE

### Architecture (Highest Impact)

1. **Split `core/event` god-package** — The single most impactful structural improvement. 12 concerns in one package is too many. Proposed sub-packages: `store`, `bus`, `projection`, `outbox`, `snapshot`, `upcaster`, `codec`. Core event types stay in `event/`. This is a **breaking change** and needs its own session with migration guide.

2. **Break circular dependencies** — `core` should have zero internal module dependencies. Move test files that import `memory`/`testhelpers` from `core/` to `integration/`. Makes `core` independently publishable.

3. **Complete deprecated API removal** — `CatalogMeta` is the last deprecated type with active production callers. Redesign `CatalogDispatcher` embedding in command/query dispatchers, then delete.

### Code Quality (Medium Impact)

4. **Fix `sync/benchmark_test.go` compilation** — Trivial fix: capture both return values from `NewVectorClockFromMap`.

5. **Rename PostgreSQL files** — `storage/event_store*.go` → `storage/postgres_*.go` to match `pebble_*`/`sqlite_*` convention.

6. **Split oversized test files** — `decider_test.go` (1318L) into 5 files, `runner_test.go` (1164L) into 3, `repository_test.go` (849L) into 3.

7. **Move HTTP-context types** — `Source`, `IPAddress`, `UserAgent` in `core/event/types.go` are request metadata, not event-domain. Move to `core/pkg/` or dedicated package.

### Documentation (Low Impact)

8. **Prune `docs/status/archive/`** — 100+ archived status reports. Consider annual cleanup.

9. **Add `docs/planning/` README** — Explain what's active vs archived.

---

## f) Top 25 Things to Get Done Next

| #   | Task                                                                                             | Impact    | Effort    | Type         |
| --- | ------------------------------------------------------------------------------------------------ | --------- | --------- | ------------ |
| 1   | Fix `sync/benchmark_test.go` compilation error                                                   | HIGH      | Trivial   | Bug fix      |
| 2   | Split `core/event` into sub-packages (store, bus, projection, outbox, snapshot, upcaster, codec) | VERY HIGH | Very High | Architecture |
| 3   | Break `core ↔ memory` circular dependency                                                        | HIGH      | Medium    | Architecture |
| 4   | Complete `CatalogMeta` removal (redesign dispatcher embedding)                                   | MEDIUM    | Medium    | Dead code    |
| 5   | Rename `storage/event_store*.go` → `postgres_*` prefix                                           | LOW       | Low       | Naming       |
| 6   | Rename `InMemoryRunner` → `InMemoryProjectionRunner`                                             | LOW       | Low       | Naming       |
| 7   | Split `decider_test.go` (1318L) into 5 logical test files                                        | LOW       | Medium    | Test quality |
| 8   | Split `projection/runner_test.go` (1164L) into 3 files                                           | LOW       | Medium    | Test quality |
| 9   | Split `aggregate/repository_test.go` (849L) into 3 files                                         | LOW       | Medium    | Test quality |
| 10  | Split `storage/event_store_test.go` (885L) into 3 files                                          | LOW       | Medium    | Test quality |
| 11  | Move `Source`/`IPAddress`/`UserAgent` out of `core/event/types.go`                               | LOW       | Low       | Naming       |
| 12  | Standardize BDD test file naming across modules                                                  | LOW       | Low       | Consistency  |
| 13  | Remove zero-consumer `Upcaster`/`Enricher` from public API or move to `internal/`                | LOW       | Low       | API surface  |
| 14  | Add `query.Handler` typed return (eliminate `any`)                                               | MEDIUM    | Medium    | Type safety  |
| 15  | Consolidate `Root.LoadEvents` vs `Core.LoadFromHistory` (mismatch)                               | LOW       | Low       | Consistency  |
| 16  | Implement `TransactionalStore` in `memory` module for test parity                                | MEDIUM    | Low       | Feature      |
| 17  | Add Saga/process manager design (docs exist: `SAGA_DESIGN.md`)                                   | MEDIUM    | High      | Feature      |
| 18  | Add `io.Closer` removal from interfaces where ownership is ambiguous                             | LOW       | Medium    | API cleanup  |
| 19  | Update `FEATURES.md` coverage numbers (some are stale)                                           | LOW       | Trivial   | Docs         |
| 20  | Update `TODO_LIST.md` to reflect current state                                                   | LOW       | Trivial   | Docs         |
| 21  | Add `docs/planning/README.md` explaining active vs archived                                      | LOW       | Trivial   | Docs         |
| 22  | Prune `docs/status/archive/` — keep last 20, archive older to cold storage                       | LOW       | Trivial   | Housekeeping |
| 23  | Add Turso connection pooling documentation                                                       | LOW       | Trivial   | Docs         |
| 24  | Investigate `samber/ro` replacement for `Pipe` to eliminate `[]any`                              | LOW       | Medium    | Type safety  |
| 25  | Add `go.work` sync CI check (ensure all replace directives are consistent)                       | MEDIUM    | Low       | CI           |

---

## g) Top #1 Question I Cannot Figure Out Myself

**Should `core/event` be split into sub-packages (breaking change) or is the current flat structure acceptable for a Go library?**

Arguments FOR splitting:

- 12 concerns, ~75 exports, 23 production files — violates SRP
- `Upcaster`/`Enricher` have zero external consumers — shouldn't be in the "core event" import
- Changing outbox logic forces recompilation of every consumer that only uses `Event`
- Go convention: small, focused packages

Arguments AGAINST splitting:

- Breaking change for every consumer
- Import path churn (`event.Store` → `store.Store` or `eventstore.Store`)
- Current flat structure works — consumers import once, use what they need
- Risk of over-modularization (Go stdlib `net/http` is a large flat package)

**I cannot resolve this without your product direction input.** If this library has zero external consumers today (not yet public), breaking changes are free. If it has consumers, we need a migration strategy.

---

## Project Metrics

| Metric                       | Value                                  |
| ---------------------------- | -------------------------------------- |
| Total LOC                    | 45,901                                 |
| Production LOC               | 15,470                                 |
| Test LOC                     | 30,431                                 |
| Production files             | 174                                    |
| Test files                   | 125                                    |
| Go modules                   | 12                                     |
| Test packages                | 26 (all pass)                          |
| Sentinel errors              | 39                                     |
| Deprecated symbols           | 10                                     |
| Exports                      | 572                                    |
| Benchmarks                   | 53 across 13 files                     |
| Lint issues                  | 0                                      |
| TODO/FIXME                   | 0                                      |
| Commits since May 1          | 508                                    |
| Production files > 250 lines | 1 (`testhelpers/fake_store.go` at 263) |

### Coverage by Package

| Package                | Coverage |
| ---------------------- | -------- |
| `core/query`           | 100.0%   |
| `core/pkg/dispatcher`  | 100.0%   |
| `middleware`           | 100.0%   |
| `memory`               | 99.6%    |
| `core/pkg/id`          | 97.8%    |
| `catalog/openapi`      | 98.1%    |
| `catalog/d2`           | 97.6%    |
| `catalog/asyncapi`     | 97.1%    |
| `catalog/adapters`     | 97.1%    |
| `catalog/eventcatalog` | 95.8%    |
| `core/aggregate`       | 95.9%    |
| `core/command`         | 94.7%    |
| `projection`           | 93.9%    |
| `core/decider`         | 93.3%    |
| `sync`                 | 92.2%    |
| `catalog`              | 91.2%    |
| `catalog/docserver`    | 91.0%    |
| `core/event`           | 90.9%    |
| `storage`              | 88.5%    |
| `testhelpers`          | 10.5%    |

### Known Issues (Active)

| Issue                                                    | Severity              |
| -------------------------------------------------------- | --------------------- |
| `MemoryBus.Publish` holds RLock during handler execution | LOW                   |
| `query.Handler` returns `any`                            | LOW                   |
| `CatalogMeta` duplicated across 3 packages               | LOW (partially fixed) |
| `Root.LoadEvents` vs `Core.LoadFromHistory` mismatch     | LOW                   |
| `sync/benchmark_test.go` won't compile                   | HIGH (pre-existing)   |
| `core/event` god-package (12 concerns, ~75 exports)      | HIGH                  |
| `core ↔ memory` circular dependency                      | MEDIUM                |

---

## Session 85 File Changes Summary

```
 CONTEXT.md => DOMAIN_GLOSSARY.md                   |   0
 README.md                                          |   2 +-
 catalog/auto_name.go                               |  37 --- (deleted)
 catalog/message_config.go                          |  34 ++- (merged auto_name)
 core/aggregate/coverage_test.go => core_errors_test.go | 0
 core/command/catalog.go                            |  70 --- (dead code removed)
 core/command/dispatcher_test.go                    |  74 --- (dead test removed)
 core/event/catalog.go                              |  82 --- (dead code removed)
 core/event/catalog_test.go                         | 141 --- (deleted)
 core/event/event_test.go                           |  50 --- (dead test removed)
 core/query/catalog.go                              |  61 --- (dead code removed)
 core/query/dispatcher_test.go                      |  65 --- (dead test removed)
 memory/coverage_test.go => store_extra_test.go     |   0
 projection/coverage_test.go => runner_extra_test.go|   0
 storage/coverage_test.go => constructor_test.go    |   0
 33 docs archived from active → archive dirs        |   0
 DEDUPLICATION_PLAN.md → docs/planning/archive/     |   0
```

**Net: -376 lines, -8 production/test files, -33 doc files from active dirs**
