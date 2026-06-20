# Session 72 — Comprehensive Status & Master TODO Plan

**Date:** 2026-05-19 20:51
**Branch:** master (clean, no unpushed commits)
**Test Status:** 23/24 packages pass (1 known example/todo integration test failure)
**Lint:** core=0, memory=0, catalog=75 pre-existing

---

## Executive Summary

go-cqrs-lite is a multi-module Go CQRS/ES library with 12 modules, 240 commits since May 1, and 42,514 total LOC. Sessions 70-71 delivered the zero-cost catalog API and storage dialect deduplication — two of the highest-impact architectural improvements. The library is approaching v0.1.0-alpha readiness.

**Key metrics:**

- 22/22 library packages pass (example/todo has 1 known integration test failure)
- Production LOC: 12,273 | Test LOC: 26,194 | Example LOC: 3,066 | Total: 42,514
- 240 commits since May 1 (19 days = ~13 commits/day average)
- 75 pre-existing lint issues (all in catalog module)
- Overall test coverage: ~91.6%
- 4 production files exceed 250-line limit
- 43+ test files exceed 250-line limit

---

## A) FULLY DONE (Sessions 70-71)

| #   | Deliverable                                                                | Impact                                                | Files Changed                                                                     |
| --- | -------------------------------------------------------------------------- | ----------------------------------------------------- | --------------------------------------------------------------------------------- |
| 1   | Zero-cost catalog API — `catalog.Command[T]()`, `Event[T]()`, `Query[T]()` | HIGH — eliminates fake-instance construction for docs | `catalog/build.go`, `message_config.go`, `auto_name.go`, deleted 6 adapters files |
| 2   | OutboxPublisher split brain fix — `publisherState` enum                    | HIGH — single source of truth for lifecycle           | `core/event/outbox_publisher.go`                                                  |
| 3   | Aggregate version/changes split brain fix — `SetVersion()` clears changes  | MEDIUM — prevents version/changes drift               | `core/aggregate/aggregate.go`                                                     |
| 4   | BaseDispatcher removal — deleted 44 lines of pure delegation               | MEDIUM — simpler mental model                         | deleted `core/pkg/dispatcher/base.go`                                             |
| 5   | Storage dialect deduplication — `Dialect` interface, unified stores        | HIGH — removed 5 SQLite files, ~500 lines duplication | `storage/dialect.go`, all store files                                             |
| 6   | Event helper tests — `PublishChanges`, `SaveSnapshot`, `ShouldSnapshot`    | MEDIUM — coverage for shared helpers                  | `core/event/publish_helper_test.go`, `snapshot_helper_test.go`                    |
| 7   | Lint fixes — exhaustive switch, nilnil nolint                              | LOW — clean build                                     | `core/event/outbox_publisher.go`                                                  |
| 8   | Golden file updates for YAML indentation drift                             | LOW — test stability                                  | `catalog/testdata/golden/*`                                                       |
| 9   | Docserver embedded static assets (Scalar + AsyncAPI React)                 | MEDIUM — standalone API docs server                   | `catalog/docserver/static/*`, `embed.go`                                          |
| 10  | OpenAPI 3.0 exporter                                                       | HIGH — REST API documentation                         | `catalog/openapi/exporter.go`                                                     |
| 11  | `reflect.TypeFor[T]()` replacing `reflect.TypeOf((*T)(nil)).Elem()`        | LOW — Go 1.22+ idiom                                  | Multiple catalog files                                                            |

---

## B) PARTIALLY DONE

| Task                     | What's Done                                               | What's Left                                                                                                                                                                                                                          |
| ------------------------ | --------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| Catalog lint (75 issues) | Core + memory at 0 issues                                 | 75 catalog issues: exhaustruct(15), wsl_v5(12), varnamelen(12), noctx(9), staticcheck(6), gocritic(4), nlreturn(4), errchkjson(3), wrapcheck(3), goconst(2), gochecknoglobals(1), modernize(1), revive(1), noinlineerr(1), unused(1) |
| Test coverage            | Most modules >90%                                         | catalog/adapters 66.7%, catalog/docserver 83.5%, catalog/openapi 83.9%, storage 86.9%                                                                                                                                                |
| File size compliance     | Zero production files over 250 lines (before recent adds) | 4 production files now exceed: `catalog/openapi/exporter.go`(311), `storage/event_store.go`(305), `catalog/docserver/docserver.go`(258), `storage/outbox.go`(255)                                                                    |

---

## C) NOT STARTED

| Task                                                              | Est.  | Module                             |
| ----------------------------------------------------------------- | ----- | ---------------------------------- |
| Delete deprecated `Catalogable`/`CatalogCore`/`CatalogMeta` types | 1h    | core/{command,event,query}         |
| Update adapters_test.go to stop using deprecated types            | 30min | catalog/adapters                   |
| Split 4 production files over 250 lines                           | 1h    | catalog, storage                   |
| Split large test files (11 files >400 lines)                      | 3h    | core, storage, catalog, projection |
| Version/SchemaVersion → uint migration                            | 1.5h  | core/event, storage                |
| SubscriptionScope enum for wildcard subscriptions                 | 1h    | core/event, projection             |
| MemoryBus handler storage consolidation                           | 20min | memory                             |
| Error wrapping consistency sweep                                  | 30min | storage                            |
| PostgreSQL integration tests for storage                          | 2h    | storage                            |
| Saga/Process Manager implementation                               | 18h   | new module                         |
| CONTRIBUTING.md                                                   | 1h    | docs                               |
| Tag v0.1.0-alpha                                                  | 30min | release                            |

---

## D) TOTALLY FUCKED UP

| Issue                                      | Detail                                                                                           | Status                              |
| ------------------------------------------ | ------------------------------------------------------------------------------------------------ | ----------------------------------- |
| example/todo integration test failure      | `TestUpdateTodo_InvalidID` expects HTTP 400 but gets 500 — error handling gap in API handler     | BROKEN — needs fix                  |
| Golden file fragility                      | `go-faster/yaml` produces different indentation across runs; golden tests fail until regenerated | WORKAROUND — regenerate when needed |
| catalog/adapters coverage dropped to 66.7% | After zero-cost API migration, some adapter paths may be dead code                               | NEEDS INVESTIGATION                 |
| Sync module uses stretchr/testify          | All other modules use onsi/ginkgo+gomega; `sync/` uses testify — inconsistent test framework     | LOW — works but inconsistent        |

---

## E) WHAT WE SHOULD IMPROVE

1. **Catalog lint (75 issues)** — the only module with lint debt. 2-3h focused effort → zero.
2. **4 production files exceed 250-line limit** — `openapi/exporter.go`(311), `storage/event_store.go`(305), `docserver/docserver.go`(258), `storage/outbox.go`(255).
3. **example/todo integration test broken** — `TestUpdateTodo_InvalidID` returns 500 instead of 400. Error classification not wired to HTTP status codes.
4. **AGENTS.md stale** — missing `sync/` module, `catalog/openapi/`, `catalog/docserver/`, `example/todo/`, storage `Dialect`. Session notes are comprehensive but module overview is outdated.
5. **TODO_LIST.md stale** — last audited Session 54 (May 4). Many items completed since then.
6. **FEATURES.md stale** — last audited May 3. Missing openapi, docserver, sync, dialect, todo example.
7. **Test framework inconsistency** — `sync/` uses testify; rest uses ginkgo/gomega.
8. **`query.Handler` returns `any`** — runtime type erasure, no compile-time safety. `DispatchTyped` workaround exists but is a separate call site.

---

## F) Master TODO Plan — All Tasks Sorted by Impact/Effort

> Each task ≤12 min. Tasks grouped into execution phases.
> **Legend:** P0=critical, P1=high, P2=medium, P3=low. Impact: H/M/L. Effort in minutes.

### Phase 1: Zero Lint (75 → 0) — P1, HIGH impact, ~2.5h

| #   | Task                                                                      | File(s)                                     | Effort | Impact |
| --- | ------------------------------------------------------------------------- | ------------------------------------------- | ------ | ------ |
| 1   | Fix exhaustruct: add missing `Examples` field to `Message` struct literal | `catalog/message_config.go`                 | 5min   | M      |
| 2   | Fix exhaustruct: add missing fields in asyncapi `Document` literal        | `catalog/asyncapi/exporter.go`              | 8min   | M      |
| 3   | Fix exhaustruct: add missing fields in asyncapi builder                   | `catalog/asyncapi/builder.go`               | 8min   | M      |
| 4   | Fix exhaustruct: add missing fields in openapi exporter                   | `catalog/openapi/exporter.go`               | 10min  | M      |
| 5   | Fix exhaustruct: remaining 11 struct literals in catalog                  | `catalog/*.go`                              | 12min  | M      |
| 6   | Fix wsl_v5: add blank lines before `w.Header().Set()` calls               | `catalog/docserver/docserver.go`            | 10min  | L      |
| 7   | Fix wsl_v5: add blank lines in openapi exporter before conditions         | `catalog/openapi/exporter.go`               | 5min   | L      |
| 8   | Fix varnamelen: rename short vars (`w`→`writer`, `ds`→`data`, etc.)       | `catalog/docserver/docserver.go`            | 10min  | L      |
| 9   | Fix varnamelen: rename short vars in openapi                              | `catalog/openapi/exporter.go`               | 10min  | L      |
| 10  | Fix varnamelen: remaining 10 short variable names                         | `catalog/asyncapi/*.go`, `catalog/types.go` | 12min  | L      |
| 11  | Fix noctx: add `context.Background()` to `httptest.NewRequest` calls      | `catalog/docserver/docserver_test.go`       | 10min  | L      |
| 12  | Fix noctx: remaining 6 test `NewRequest` calls                            | `catalog/docserver/docserver_test.go`       | 8min   | L      |
| 13  | Fix staticcheck: replace deprecated `CatalogMeta` usage in tests          | `catalog/adapters/adapters_test.go`         | 10min  | M      |
| 14  | Fix staticcheck: remaining 5 deprecated usage warnings                    | `catalog/internal/cattest/*.go`             | 10min  | M      |
| 15  | Fix gocritic: `if-else` → `switch` in `camelCaseToHuman`                  | `catalog/auto_name.go`                      | 5min   | L      |
| 16  | Fix gocritic: `HasSuffix`+`TrimSuffix` → `strings.CutSuffix`              | `catalog/auto_name.go`                      | 5min   | L      |
| 17  | Fix gocritic: remaining 2 gocritic issues                                 | `catalog/*.go`                              | 8min   | L      |
| 18  | Fix nlreturn: add blank lines before return statements                    | `catalog/asyncapi/types.go`                 | 8min   | L      |
| 19  | Fix errchkjson: silence or fix 3 JSON marshal warnings                    | `catalog/asyncapi/types.go`                 | 8min   | L      |
| 20  | Fix wrapcheck: wrap 3 errors from external calls                          | `catalog/asyncapi/*.go`                     | 8min   | L      |
| 21  | Fix goconst: extract repeated string literals                             | `catalog/asyncapi/*.go`                     | 5min   | L      |
| 22  | Fix gochecknoglobals: inline struct init for classifier                   | `catalog/asyncapi/types.go`                 | 5min   | L      |
| 23  | Fix modernize: use `min`/`max` builtins                                   | `catalog/*.go`                              | 5min   | L      |
| 24  | Fix revive: add missing doc comments                                      | `catalog/*.go`                              | 5min   | L      |
| 25  | Fix noinlineerr: inline error creation                                    | `catalog/*.go`                              | 5min   | L      |
| 26  | Fix unused: remove unused variable/import                                 | `catalog/*.go`                              | 5min   | L      |
| 27  | Verify catalog lint = 0                                                   | `nix run .#lint`                            | 2min   | H      |

### Phase 2: File Size Compliance — P1, HIGH impact, ~1.5h

| #   | Task                                                                        | File(s)              | Effort | Impact |
| --- | --------------------------------------------------------------------------- | -------------------- | ------ | ------ |
| 28  | Split `catalog/openapi/exporter.go` (311→<250) — extract schema builder     | `catalog/openapi/`   | 12min  | H      |
| 29  | Split `storage/event_store.go` (305→<250) — extract scan/load helpers       | `storage/`           | 12min  | H      |
| 30  | Split `catalog/docserver/docserver.go` (258→<250) — extract handler routing | `catalog/docserver/` | 12min  | H      |
| 31  | Split `storage/outbox.go` (255→<250) — extract poll/publish helpers         | `storage/`           | 10min  | H      |
| 32  | Verify all production files ≤250 lines                                      | `find` + `wc -l`     | 2min   | H      |

### Phase 3: Fix Broken Tests — P0, CRITICAL, ~30min

| #   | Task                                                                      | File(s)                 | Effort | Impact |
| --- | ------------------------------------------------------------------------- | ----------------------- | ------ | ------ |
| 33  | Fix `TestUpdateTodo_InvalidID` — return 400 for invalid ID instead of 500 | `example/todo/cmd/api/` | 12min  | H      |
| 34  | Run full test suite to verify 24/24 pass                                  | `go test ./...`         | 5min   | H      |

### Phase 4: Coverage Gaps — P1, MEDIUM impact, ~2h

| #   | Task                                                  | File(s)                   | Effort | Impact |
| --- | ----------------------------------------------------- | ------------------------- | ------ | ------ |
| 35  | Investigate catalog/adapters 66.7% — find dead paths  | `catalog/adapters/*.go`   | 10min  | M      |
| 36  | Add tests for uncovered adapter paths                 | `catalog/adapters/`       | 10min  | M      |
| 37  | Add storage dialect-specific unit tests (SQLite path) | `storage/dialect_test.go` | 12min  | M      |
| 38  | Add tests for `catalog/docserver` (83.5%→90%+)        | `catalog/docserver/`      | 12min  | M      |
| 39  | Add tests for `catalog/openapi` exporter (83.9%→90%+) | `catalog/openapi/`        | 12min  | M      |
| 40  | Add tests for `storage` error paths (86.9%→90%+)      | `storage/`                | 12min  | M      |
| 41  | Add tests for `core/decider` (92.7%→95%+)             | `core/decider/`           | 10min  | M      |
| 42  | Run coverage report to verify improvements            | `go test -coverprofile`   | 5min   | M      |

### Phase 5: Documentation Update — P1, HIGH impact, ~1.5h

| #   | Task                                                           | File(s)        | Effort | Impact |
| --- | -------------------------------------------------------------- | -------------- | ------ | ------ |
| 43  | Update AGENTS.md: add `sync/` module overview                  | `AGENTS.md`    | 10min  | H      |
| 44  | Update AGENTS.md: add `catalog/openapi/` to Package Overview   | `AGENTS.md`    | 8min   | H      |
| 45  | Update AGENTS.md: add `catalog/docserver/` to Package Overview | `AGENTS.md`    | 8min   | H      |
| 46  | Update AGENTS.md: add storage `Dialect` architecture           | `AGENTS.md`    | 10min  | H      |
| 47  | Update AGENTS.md: add `example/todo/` description              | `AGENTS.md`    | 8min   | H      |
| 48  | Update AGENTS.md: refresh coverage numbers                     | `AGENTS.md`    | 5min   | M      |
| 49  | Update AGENTS.md: refresh known issues                         | `AGENTS.md`    | 8min   | M      |
| 50  | Update TODO_LIST.md to current status                          | `TODO_LIST.md` | 12min  | M      |
| 51  | Update FEATURES.md: add openapi, docserver, sync, dialect      | `FEATURES.md`  | 12min  | H      |
| 52  | Update FEATURES.md: refresh coverage numbers                   | `FEATURES.md`  | 5min   | M      |

### Phase 6: Cleanup & Deprecation — P2, MEDIUM impact, ~2h

| #   | Task                                                                      | File(s)                     | Effort | Impact |
| --- | ------------------------------------------------------------------------- | --------------------------- | ------ | ------ |
| 53  | Delete deprecated `Catalogable` from `core/command/catalog.go`            | `core/command/`             | 8min   | M      |
| 54  | Delete deprecated `Catalogable` from `core/event/catalog.go`              | `core/event/`               | 8min   | M      |
| 55  | Delete deprecated `Catalogable` from `core/query/catalog.go`              | `core/query/`               | 8min   | M      |
| 56  | Update `catalog/adapters/adapters_test.go` — remove deprecated type usage | `catalog/adapters/`         | 12min  | M      |
| 57  | Remove deprecated usage from `catalog/internal/cattest/`                  | `catalog/internal/cattest/` | 10min  | M      |
| 58  | Verify example/user compiles without deprecated types                     | `example/user/`             | 5min   | M      |
| 59  | Verify example/todo compiles without deprecated types                     | `example/todo/`             | 5min   | M      |
| 60  | Run full test suite after deprecation removal                             | `go test ./...`             | 5min   | H      |

### Phase 7: Type Safety — P2, MEDIUM impact, ~2h

| #   | Task                                                     | File(s)                | Effort | Impact |
| --- | -------------------------------------------------------- | ---------------------- | ------ | ------ |
| 61  | Change `event.Version` backing type `int` → `uint`       | `core/event/types.go`  | 10min  | M      |
| 62  | Update `Version` callers in core/                        | `core/**/*.go`         | 12min  | M      |
| 63  | Update `Version` callers in storage/                     | `storage/*.go`         | 10min  | M      |
| 64  | Update `Version` callers in memory/                      | `memory/*.go`          | 8min   | M      |
| 65  | Update `Version` callers in examples/                    | `example/**/*.go`      | 8min   | M      |
| 66  | Change `event.SchemaVersion` backing type `int` → `uint` | `core/event/types.go`  | 5min   | M      |
| 67  | Update `SchemaVersion` callers                           | Multiple               | 8min   | M      |
| 68  | Define `SubscriptionScope` enum type                     | `core/event/types.go`  | 8min   | M      |
| 69  | Update `SubscribesTo` to use `SubscriptionScope`         | `core/event/`          | 10min  | M      |
| 70  | Update `projection.Runner` for `SubscriptionScope`       | `projection/runner.go` | 10min  | M      |
| 71  | Run full test suite after type changes                   | `go test ./...`        | 5min   | H      |

### Phase 8: Infrastructure Improvements — P2, MEDIUM impact, ~2h

| #   | Task                                                          | File(s)                     | Effort | Impact |
| --- | ------------------------------------------------------------- | --------------------------- | ------ | ------ |
| 72  | Consolidate MemoryBus handlers — single map with sentinel key | `memory/bus.go`             | 12min  | M      |
| 73  | Standardize storage error wrapping patterns                   | `storage/*.go`              | 12min  | M      |
| 74  | Replace remaining dynamic errors with sentinels               | Multiple                    | 12min  | M      |
| 75  | Migrate `sync/` tests from testify to ginkgo/gomega           | `sync/*_test.go`            | 12min  | L      |
| 76  | Add benchmark for dialect-based stores                        | `storage/benchmark_test.go` | 12min  | L      |
| 77  | Add integration test for SQLite dialect transactional store   | `storage/`                  | 12min  | M      |

### Phase 9: Release Preparation — P3, LOW impact, ~3h

| #   | Task                                         | File(s)           | Effort | Impact |
| --- | -------------------------------------------- | ----------------- | ------ | ------ |
| 78  | Write CHANGELOG.md entry for v0.1.0-alpha    | `CHANGELOG.md`    | 12min  | M      |
| 79  | Write CONTRIBUTING.md                        | `CONTRIBUTING.md` | 12min  | L      |
| 80  | Verify `go mod tidy` in all 12 modules       | All `go.mod`      | 12min  | L      |
| 81  | Run `nix flake check` for full CI validation | `flake.nix`       | 5min   | L      |
| 82  | Tag v0.1.0-alpha                             | Git tag           | 2min   | H      |
| 83  | Write release notes                          | GitHub release    | 12min  | M      |

---

## G) Top #1 Question I Cannot Figure Out Myself

> **Should we split large test files (>250 lines) even though the convention appears to exempt them?**
>
> 43+ test files exceed 250 lines. The largest is `core/decider/decider_test.go` at 1,146 lines. Every session audit says "Zero **production** files exceed 250 lines" — implying test files are OK. But the AGENTS.md says "Max 250 lines per file" without qualification. Splitting test files is ~3h of low-value work that doesn't change coverage or correctness.

---

## Module Coverage Summary

| Module               | Coverage | Status |
| -------------------- | -------- | ------ |
| core/command         | 100.0%   | ✅     |
| core/query           | 100.0%   | ✅     |
| core/pkg/dispatcher  | 100.0%   | ✅     |
| middleware           | 100.0%   | ✅     |
| memory               | 99.5%    | ✅     |
| projection           | 98.3%    | ✅     |
| core/pkg/id          | 97.8%    | ✅     |
| catalog/d2           | 97.6%    | ✅     |
| core/aggregate       | 96.9%    | ✅     |
| core/event           | 96.3%    | ✅     |
| catalog              | 95.3%    | ✅     |
| catalog/eventcatalog | 95.7%    | ✅     |
| catalog/asyncapi     | 93.9%    | ✅     |
| core/decider         | 92.7%    | ✅     |
| sync                 | 92.2%    | ✅     |
| catalog/adapters     | 66.7%    | ⚠️     |
| catalog/docserver    | 83.5%    | ⚠️     |
| catalog/openapi      | 83.9%    | ⚠️     |
| storage              | 86.9%    | ⚠️     |

---

## Module Lint Summary

| Module     | Issues | Notes                                                                                                                                                                                                             |
| ---------- | ------ | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| core       | 0      | ✅ Clean                                                                                                                                                                                                          |
| memory     | 0      | ✅ Clean                                                                                                                                                                                                          |
| middleware | —      | Not linted separately (core covers it)                                                                                                                                                                            |
| projection | —      | Not linted separately                                                                                                                                                                                             |
| storage    | —      | Not linted separately                                                                                                                                                                                             |
| catalog    | 75     | exhaustruct(15), varnamelen(12), wsl_v5(12), noctx(9), staticcheck(6), gocritic(4), nlreturn(4), errchkjson(3), wrapcheck(3), goconst(2), gochecknoglobals(1), modernize(1), revive(1), noinlineerr(1), unused(1) |

---

## Commit Log (Today — Sessions 70-71)

| Commit    | Message                                                                                                             |
| --------- | ------------------------------------------------------------------------------------------------------------------- |
| `dac8564` | feat(storage): complete dialect deduplication + add event helper tests                                              |
| `ac45b47` | chore: format multi-line function calls, fix lint issues, and update golden files                                   |
| `7350e02` | feat(docserver): update embedded static assets for Scalar + AsyncAPI React                                          |
| `095fa4c` | feat(docserver): embed Scalar + AsyncAPI React UI libraries in binary                                               |
| `d82ed39` | refactor(catalog): replace reflect.TypeOf((\*T)(nil)).Elem() with reflect.TypeFor[T]()                              |
| `23d8d3b` | docs(status): add Session 70 comprehensive full status report                                                       |
| `2f008b5` | refactor(core): delete BaseDispatcher abstraction, inline into command/query dispatchers                            |
| `4fe3607` | docs(status): add Session 70 status report                                                                          |
| `4d97b44` | fix(core): eliminate split brain conditions in OutboxPublisher and Aggregate                                        |
| `0d812cf` | feat(catalog): complete zero-cost API migration with example rewrite                                                |
| `15d694f` | feat(catalog): zero-cost catalog API with auto-derived schemas and names                                            |
| `087a496` | feat(catalog): zero-cost catalog API with auto-derived schemas and names                                            |
| `5805b02` | refactor(core/event, catalog/openapi, catalog/docserver): align struct field tags and embed shared catalog metadata |
| `56339fe` | feat(catalog): add OpenAPI 3.0 exporter and stdlib docserver for REST/API docs                                      |
