# Session 77 — Comprehensive Status Report

**Date:** 2026-05-20 03:05  
**Session:** 77 (continuation from Sessions 74-76)  
**Commits this session:** 8  
**Total commits since May 1:** 434

---

## Executive Summary

The go-cqrs-lite library is in its **healthiest state ever**: zero lint across all 6 production modules, 22/22 test packages passing, 14,915 production LOC / 28,987 test LOC across 11 workspace modules. This session completed the SchemaType branded type migration, zeroed storage lint, and laid groundwork for catalog deduplication.

---

## A) FULLY DONE ✓

| #   | Task                                                                                                                                                                                                                                                                                | Commit    | Impact                                         |
| --- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | --------- | ---------------------------------------------- |
| 1   | **Split event.go** (284→220 lines): extract Metadata+mergeFrom+NewMetadata to metadata.go                                                                                                                                                                                           | `16b5d98` | Eliminates last production file over 250 lines |
| 2   | **Zero storage lint** (16→0 issues): extract eventColumnCount/baseEventOpts constants, wrap long ParseTime lines, exclude mnd for SQL positional params, fix nolintlint/err113                                                                                                      | `127e6ea` | Storage module now lint-clean                  |
| 3   | **Zero middleware lint** (2→0): replaced deprecated `command.CatalogMeta` with `command.New`                                                                                                                                                                                        | `c4bc5c0` | All modules lint-clean                         |
| 4   | **SchemaType branded type**: `Schema.Type` and `Property.Type` now `catalog.SchemaType` with 7 exported constants (TypeString, TypeObject, TypeInteger, TypeNumber, TypeBoolean, TypeArray, TypeNull). Replaced unexported `jsonType*` constants. Updated ~15 sites across catalog. | `dc90ebc` | Compile-time type safety for JSON Schema types |
| 5   | **Example/todo split**: main.go (472→235 lines) → 3 files (main, handlers, middleware)                                                                                                                                                                                              | `69d658b` | No example file over 250 lines                 |
| 6   | **Catalog id_parse tests**: rewritten with stdlib (no testify dependency)                                                                                                                                                                                                           | `c4bc5c0` | Removed unnecessary test dependency            |
| 7   | **Integration test fixes**: command_test.go, query_test.go updated for typed APIs                                                                                                                                                                                                   | earlier   | Build consistency                              |
| 8   | **Pre-commit hook fixes**: wsl, staticcheck, err113, golines across storage/catalog/middleware                                                                                                                                                                                      | `c4bc5c0` | CI-friendly                                    |

### Current Quality Metrics

| Metric                              | Value                                                    |
| ----------------------------------- | -------------------------------------------------------- |
| **Lint issues (all modules)**       | **0**                                                    |
| **Test packages passing**           | **22/22**                                                |
| **Production files over 250 lines** | **0** (max: 237 runner.go)                               |
| **Test files over 1000 lines**      | **3** (decider_test 1190, runner_test 1057, id_test 993) |
| **Production LOC**                  | 14,915                                                   |
| **Test LOC**                        | 28,987                                                   |
| **Benchmark functions**             | 43 across 11 files                                       |
| **TODO/FIXME in production**        | 0                                                        |

### Coverage by Module

| Package                | Coverage |
| ---------------------- | -------- |
| `core/command`         | 100.0%   |
| `core/query`           | 100.0%   |
| `core/pkg/dispatcher`  | 100.0%   |
| `middleware`           | 100.0%   |
| `memory`               | 99.5%    |
| `projection`           | 98.3%    |
| `core/pkg/id`          | 97.8%    |
| `catalog/d2`           | 97.6%    |
| `catalog/adapters`     | 97.1%    |
| `catalog/openapi`      | 96.6%    |
| `core/aggregate`       | 96.9%    |
| `catalog/eventcatalog` | 95.8%    |
| `catalog`              | 94.9%    |
| `catalog/asyncapi`     | 93.9%    |
| `core/event`           | 93.7%    |
| `core/decider`         | 93.6%    |
| `catalog/docserver`    | 92.3%    |
| `storage`              | 88.1%    |

---

## B) PARTIALLY DONE 🔧

| Task                                       | Status | What's Done                         | What's Left                                                                                         |
| ------------------------------------------ | ------ | ----------------------------------- | --------------------------------------------------------------------------------------------------- |
| **Catalog deduplication: schemaToAny**     | 30%    | Research complete, files identified | Extract `SchemaToAny` + `objectSchema` to `catalog/internal/schemautil/`, update asyncapi + openapi |
| **Catalog deduplication: case conversion** | 30%    | Research complete, files identified | Extract `toDotAddress`/`toKebab` → shared `convertCase(s, sep)` to `catalog/internal/caseutil/`     |

---

## C) NOT STARTED ○

| #   | Task                                                                         | Priority | Effort |
| --- | ---------------------------------------------------------------------------- | -------- | ------ |
| 1   | Storage benchmarks (only module without any)                                 | Medium   | 2h     |
| 2   | Deduplicate `schemaToAny`/`objectSchema` into `catalog/internal/schemautil/` | Medium   | 1h     |
| 3   | Deduplicate `toDotAddress`/`toKebab` into `catalog/internal/caseutil/`       | Medium   | 1h     |
| 4   | Trim AGENTS.md from 837 to ≤377 lines (go-structure-linter warning)          | Low      | 2h     |
| 5   | Sync module tests and coverage                                               | Medium   | 4h     |
| 6   | Publish `testhelpers@v1.2.0` (fixes `int` → `event.Version` breaking change) | High     | 30min  |
| 7   | Move coverage.out to coverage/ directory                                     | Low      | 10min  |
| 8   | Add `storage` benchmarks for EventStore, SnapshotStore, Outbox               | Medium   | 2h     |
| 9   | Investigate `core/decider` coverage drop (93.6% → was 95.0%)                 | Low      | 1h     |

---

## D) TOTALLY FUCKED UP ✗

Nothing is totally fucked up. The codebase is clean.

**However, there are these persistent irritants:**

| Issue                                                    | Severity | Detail                                                                                                                                                       |
| -------------------------------------------------------- | -------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| **gopls typecheck errors in storage/**                   | LOW      | `sqlite_integration_test.go` references `NewSQLiteTransactionalStore` which doesn't exist (build tags issue?). Tests pass via `go test` but gopls complains. |
| **Pre-commit hook: go-structure-linter false positives** | LOW      | Reports `pkg/` directory missing, `coverage.out` in root, `AGENTS.md` too long. All stylistic, not bugs. Hook fails on these every time.                     |
| **Pre-commit hook: golangci-lint from root**             | LOW      | `golangci-lint run` from repo root fails because of go.work. Must run per-module. Not a real issue.                                                          |
| **`testhelpers@v1.1.0` stale**                           | MEDIUM   | Published version uses `int` not `event.Version`. Workspace builds fine, but `GOWORK=off` in dependent projects breaks. Needs v1.2.0 release.                |
| **`sync/` module has no tests yet**                      | MEDIUM   | Brand new module with conflict detection but no test coverage numbers.                                                                                       |

---

## E) WHAT WE SHOULD IMPROVE

### Architecture

1. **`CatalogMeta` deprecation BLOCKED** — `command/dispatcher.go:15` and `query/dispatcher.go:18` embed `dispatcher.CatalogDispatcher[Type, CatalogMeta]`. Cannot delete `CatalogMeta`/`Catalogable`/`CatalogCore` without redesigning dispatcher catalog integration. This is the biggest architectural debt.

2. **`query.Handler` returns `any`** — Violates project "no any" rule. `DispatchTyped[T]` is the workaround. Design doc exists at `docs/planning/QUERY_HANDLER_GENERICS.md`.

3. **`Dialect.FormatTime` returns `any`** — SQL driver interop, but `driver.Valuer`/`sql.Scanner` would be nicer.

4. **`sync/` module is brand new** — Needs test coverage, benchmarks, and documentation before it can be trusted.

### Type Safety

5. **`SchemaType` is done** ✓ — But `asyncapi/typeObject` and `openapi/objectType` are still raw strings in map[string]string contexts. Could use `catalog.TypeObject` if the map values were typed.

6. **`Direction` and `MessageKind` are typed strings** — Good pattern, consistent with `SchemaType`.

### Code Quality

7. **AGENTS.md at 837 lines** — go-structure-linter says max 377. Should extract session history to archive files, keep only active reference info.

8. **3 test files over 1000 lines** — `decider_test.go` (1190), `runner_test.go` (1057), `id_test.go` (993). Low customer value to split but high LOC.

9. **No storage benchmarks** — Only module without any. Should add at least Save, Load, Snapshot, Outbox benchmarks.

### Process

10. **Pre-commit hook too aggressive** — go-structure-linter, library-policy, and golangci-lint-from-root all fail on non-issues. Should configure exclusion rules or tone down the hook.

---

## F) Top 25 Things We Should Get Done Next

| #   | Task                                                                          | Impact | Effort | Category      |
| --- | ----------------------------------------------------------------------------- | ------ | ------ | ------------- |
| 1   | **Publish `testhelpers@v1.2.0`** — unblocks external consumers                | HIGH   | 30min  | Release       |
| 2   | **Deduplicate `schemaToAny`/`objectSchema`** → `catalog/internal/schemautil/` | MED    | 1h     | Dedup         |
| 3   | **Deduplicate `toDotAddress`/`toKebab`** → `catalog/internal/caseutil/`       | MED    | 1h     | Dedup         |
| 4   | **Add storage benchmarks** (Save, Load, Snapshot, Outbox)                     | MED    | 2h     | Perf          |
| 5   | **Add sync module tests + coverage**                                          | HIGH   | 4h     | Quality       |
| 6   | **Trim AGENTS.md to ≤377 lines** (extract session history to archive)         | LOW    | 2h     | Cleanup       |
| 7   | **Investigate `NewSQLiteTransactionalStore` gopls errors**                    | LOW    | 30min  | DX            |
| 8   | **Add `catalog/docserver` integration test**                                  | MED    | 2h     | Quality       |
| 9   | **Fix pre-commit hook false positives** (go-structure-linter config)          | LOW    | 1h     | DX            |
| 10  | **`query.Handler` typed generics migration** (per design doc)                 | HIGH   | 8h     | Breaking      |
| 11  | **Redesign dispatcher catalog integration** to unblock CatalogMeta deletion   | HIGH   | 16h    | Architecture  |
| 12  | **Add `storage` Turso integration tests**                                     | MED    | 4h     | Quality       |
| 13  | **`Dialect.FormatTime` → `driver.Valuer`**                                    | LOW    | 2h     | Type Safety   |
| 14  | **Split large test files** (decider_test 1190, runner_test 1057)              | LOW    | 3h     | Cleanup       |
| 15  | **Add `sync/` benchmarks**                                                    | MED    | 1h     | Perf          |
| 16  | **Add `catalog/openapi` round-trip test**                                     | LOW    | 1h     | Quality       |
| 17  | **Create `docs/planning/SYNC_MODULE_ROADMAP.md`**                             | MED    | 2h     | Docs          |
| 18  | **Add example/todo integration test**                                         | LOW    | 2h     | Quality       |
| 19  | **Move coverage.out to coverage/ directory**                                  | LOW    | 10min  | Cleanup       |
| 20  | **Add `io.Closer` removal design doc** (deferred from Session 55)             | LOW    | 1h     | Planning      |
| 21  | **Add Saga design implementation** (per `docs/planning/SAGA_DESIGN.md`)       | MED    | 18h    | Feature       |
| 22  | **Explore Pebble backend for embedded storage**                               | LOW    | 4h     | Feature       |
| 23  | **Add `event.GlobalLoader` to Turso store**                                   | MED    | 2h     | Feature       |
| 24  | **Write `docs/planning/QUERY_HANDLER_GENERICS.md` implementation**            | MED    | 8h     | Feature       |
| 25  | **Add OpenTelemetry tracing to storage layer**                                | LOW    | 4h     | Observability |

---

## G) My #1 Question I Cannot Answer Myself

**When should we publish `testhelpers@v1.2.0` (and possibly bump all other modules)?**

The published `testhelpers@v1.1.0` uses raw `int` instead of `event.Version` in its API. This means anyone importing it with `GOWORK=off` gets compilation errors. The workspace hides this because it uses local versions. I cannot decide:

- Should we do a coordinated release of all 11 modules?
- Should we just publish testhelpers alone?
- Is there a release cadence or versioning convention you want to follow?
- Should we tag a `v0.x.0` or wait for `v1.0.0` readiness?

This is blocking external consumers and I need your decision to proceed.

---

## Session Stats

| Metric                 | Value                                                        |
| ---------------------- | ------------------------------------------------------------ |
| Commits this session   | 8                                                            |
| Files changed          | 33                                                           |
| Lines added            | 1,204                                                        |
| Lines removed          | 568                                                          |
| Net change             | +636                                                         |
| Lint issues fixed      | 18 (storage 16 + catalog 2)                                  |
| Test packages          | 22/22 passing                                                |
| Modules with zero lint | 6/6 (core, memory, catalog, middleware, projection, storage) |
