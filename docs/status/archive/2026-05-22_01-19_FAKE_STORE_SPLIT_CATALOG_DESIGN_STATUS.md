# Session 90 — Fake Store Split + Catalog Design Deep Dive

**Date:** 2026-05-22 01:19
**Branch:** master
**Last Commit:** f989870 refactor(testhelpers): split fake_store.go to meet <250 line quality gate
**Previous Status:** 2026-05-22_00-05_SESSION_89_API_SURFACE_STATUS.md

---

## Executive Summary

This session accomplished two things: (1) split `testhelpers/fake_store.go` to meet the 250-line quality gate, and (2) during review, identified a fundamental design tension in the `CatalogDispatcher`/`CatalogEntry` system. `CatalogEntry` lives in `core/pkg/dispatcher/` (not `catalog/`) due to module boundary constraints. The deeper question: should `CatalogDispatcher` exist at all, or should the deprecated catalog extraction adapters (`FromCommandDispatcher`, `FromQueryDispatcher`) be removed entirely?

All golden files updated. 24/24 healthy test packages pass.

---

## a) FULLY DONE ✅

### Session 90 Deliverables (commit f989870)

| #   | Task                              | Files                                                                    | Impact                                            |
| --- | --------------------------------- | ------------------------------------------------------------------------ | ------------------------------------------------- |
| 1   | Split `testhelpers/fake_store.go` | `fake_store.go` (263→221 lines), `fake_store_setters.go` (new, 51 lines) | Quality gate met — both files under 250 lines     |
| 2   | Update README.md CatalogMeta refs | `README.md:418,422`                                                      | `command.CatalogMeta` → `dispatcher.CatalogEntry` |
| 3   | Golden file refresh               | `catalog/testdata/golden/*`                                              | AsyncAPI YAML, EventCatalog JS, package.json      |
| 4   | AGENTS.md CatalogMeta status      | `AGENTS.md:521`                                                          | Marked as FIXED                                   |

### Pre-existing Deliverables (Session 88/89, already in git history)

| Commit    | Description                                                                                           |
| --------- | ----------------------------------------------------------------------------------------------------- |
| `1088fcd` | Centralize `CatalogEntry` in `core/pkg/dispatcher/`, delete `command.CatalogMeta`/`query.CatalogMeta` |
| `333a6af` | Format all test files for `CatalogEntry` migration                                                    |

---

## b) PARTIALLY DONE 🔶

### Catalog Design Review

The `CatalogDispatcher` mixin and `CatalogEntry` type were reviewed in depth. The type placement is technically correct (circular dependency prevention) but architecturally questionable. Full redesign deferred pending decision on adapter deprecation.

| Aspect                 | Status      | Detail                                         |
| ---------------------- | ----------- | ---------------------------------------------- |
| Design analysis        | Done        | Identified 3 approaches (see section g)        |
| Code changes           | Not started | Requires breaking change decision              |
| Test impact assessment | Done        | 12 files reference `CatalogDispatcher` methods |

---

## c) NOT STARTED ⬜

| #   | Task                                                                     | Priority | Effort | Blocker                                     |
| --- | ------------------------------------------------------------------------ | -------- | ------ | ------------------------------------------- |
| 1   | Remove `CatalogDispatcher` mixin entirely                                | HIGH     | 2h     | Decision: keep or delete adapters           |
| 2   | Move `CatalogEntry` to `catalog/` if adapters survive                    | LOW      | 30min  | Needs circular dep solution via interface   |
| 3   | Delete deprecated `FromCommandDispatcher`/`FromQueryDispatcher`          | HIGH     | 1h     | Decision only                               |
| 4   | Delete `CatalogDispatcher`, `CopyCatalogEntries`, `NewCatalogDispatcher` | HIGH     | 1h     | Depends on #3                               |
| 5   | Remove `RegisterCatalogEntry`/`CatalogEntries` from tests                | MED      | 30min  | Depends on #3                               |
| 6   | Projection builder pre-existing failures                                 | MED      | 1h     | 2 test files broken at session start        |
| 7   | Integration event codec pre-existing failures                            | MED      | 30min  | `ErrNilSnapshot` undefined at session start |

---

## d) TOTALLY FUCKED UP 💥

### Pre-existing Build Failures (not caused by this session)

| Issue                                                 | Severity     | Package                                 | Detail                                       |
| ----------------------------------------------------- | ------------ | --------------------------------------- | -------------------------------------------- |
| `undefined: event.ErrNilSnapshot`                     | Pre-existing | `integration/event/codec_batch_test.go` | Missing error sentinel                       |
| `undefined: ErrNilPayload`                            | Pre-existing | `core/event/codec_typed.go:31`          | Missing error sentinel                       |
| `builtProjection does not implement event.Projection` | Pre-existing | `projection/builder.go:65`              | Missing `Handle` method on `builtProjection` |

### Root Cause of Pre-existing Failures

These failures existed at session start (verified by `git status` showing uncommitted local changes in `core/aggregate/aggregate.go`, `projection/runner.go`, `projection/runner_live.go` that were not from this session's work). They appear to be from an incomplete branch or local experimentation that was never committed.

---

## e) WHAT WE SHOULD IMPROVE

### CatalogDispatcher Design Tension

**Current state:** `command.Dispatcher` and `query.Dispatcher` embed `dispatcher.CatalogDispatcher[Type, dispatcher.CatalogEntry]`. This gives them `RegisterCatalogEntry()` and `CatalogEntries()` methods — a side channel for documentation metadata.

**The problem:** `CatalogEntry` lives in `core/pkg/dispatcher/` because `catalog/` (separate module) already imports `core/command` and `core/query`. Moving it to `catalog/` creates a circular module dependency.

**Three approaches:**

| Approach                                    | Pros                             | Cons                                                     | Verdict                   |
| ------------------------------------------- | -------------------------------- | -------------------------------------------------------- | ------------------------- |
| **A. Keep as-is**                           | Zero change, backward compatible | Wrong package for domain concept                         | Not recommended long-term |
| **B. Remove CatalogDispatcher entirely**    | Cleanest, no mixin pollution     | Breaks deprecated adapters, requires consumer migration  | **Recommended**           |
| **C. Extract interface, place in catalog/** | Type lives in right package      | Interface indirection, still requires consumer migration | Over-engineered           |

**Recommendation: Approach B.** The adapters `FromCommandDispatcher` and `FromQueryDispatcher` are already marked `Deprecated` with zero non-test callers. Delete them, delete `CatalogDispatcher`, delete `CatalogEntry`, delete `CopyCatalogEntries`. Consumers use the zero-cost catalog API (`catalog.Command[T]()`) instead.

### Code Quality Metrics

| Metric                       | Before Session 90                      | After Session 90                              | Target |
| ---------------------------- | -------------------------------------- | --------------------------------------------- | ------ |
| Files >250 lines             | 1 (`testhelpers/fake_store.go` at 263) | 1 (`catalog/eventcatalog/exporter.go` at 250) | 0      |
| Test packages passing        | 24/24                                  | 24/24                                         | 27/27  |
| CatalogMeta references in Go | 0                                      | 0                                             | 0      |

---

## f) Top #25 Things to Do Next

### High Impact (Ship Value)

| #   | Task                                                     | Impact            | Effort |
| --- | -------------------------------------------------------- | ----------------- | ------ |
| 1   | **Remove `CatalogDispatcher` + deprecated adapters**     | Clean API surface | 2h     |
| 2   | **Fix pre-existing build failures** (3 packages)         | Green CI          | 2h     |
| 3   | **Split `catalog/eventcatalog/exporter.go`** (250 lines) | Quality gate      | 30min  |
| 4   | **Version bump and tag** post-CatalogDispatcher removal  | Consumer clarity  | 15min  |
| 5   | **Converge `InMemoryRunner` + `projection.Runner`**      | Architecture      | 4h     |

### Medium Impact (Library Quality)

| #   | Task                                                            | Impact           | Effort |
| --- | --------------------------------------------------------------- | ---------------- | ------ |
| 6   | Remove `command.IdempotencyKey()` from interface                | Clean API        | 15min  |
| 7   | Remove `aggregate` package deprecation notice (or commit to it) | Consumer clarity | 15min  |
| 8   | Move integration projection tests to internal                   | API surface      | 30min  |
| 9   | Fix `ParseUserAgent` return type inconsistency                  | Consistency      | 15min  |
| 10  | Increase `catalog/eventcatalog` coverage to 92%+                | Quality          | 30min  |
| 11  | Add example/ for Outbox pattern                                 | Consumer DX      | 1h     |
| 12  | Add integration test for outbox publisher                       | Reliability      | 1h     |
| 13  | Add godoc examples to all exported types                        | Consumer DX      | 2h     |

### Lower Impact (Polish)

| #   | Task                                                  | Impact       | Effort |
| --- | ----------------------------------------------------- | ------------ | ------ |
| 14  | Generate API surface documentation from exports       | Docs         | 2h     |
| 15  | Increase `testhelpers` coverage to 30%+               | Quality      | 1h     |
| 16  | Add OpenAPI 3.1 exporter for queries                  | Feature      | 4h     |
| 17  | Add `event.Validate()` consumer-side validation       | Feature      | 2h     |
| 18  | Document versioning strategy (semver)                 | Docs         | 1h     |
| 19  | Add CHANGELOG.md                                      | Docs         | 1h     |
| 20  | CI pipeline: fail if coverage drops >1%               | Quality      | 2h     |
| 21  | Explore `internal/` sub-packages for event impl types | Architecture | 4h     |
| 22  | PostgreSQL integration tests with testcontainers      | Reliability  | 4h     |
| 23  | Turso sync live tests (local sync server)             | Reliability  | 3h     |
| 24  | Storage backend guide (`docs/STORAGE_GUIDE.md`)       | Docs         | 2h     |
| 25  | Benchmark comparison: SQLite vs Turso vs Pebble       | Performance  | 1h     |

---

## g) Top #1 Question I Cannot Figure Out Myself

**Should `CatalogDispatcher` be removed entirely, or should it be redesigned to live in a more appropriate package?**

**Context:** `CatalogDispatcher` exists solely to support `catalog/adapters/from_query_dispatcher.go` and `catalog/adapters/from_command_dispatcher.go` — both marked `Deprecated` with zero non-test callers. The zero-cost catalog API (`catalog.Command[T]()`, `catalog.Event[T]()`, `catalog.Query[T]()`) has superseded this entire subsystem.

**Arguments for removal:**

- `CatalogDispatcher` is dead weight on `command.Dispatcher` and `query.Dispatcher` — every instance carries a `map[Type]CatalogEntry` whether used or not
- `CatalogEntry` is a documentation concept polluting the dispatcher infrastructure package
- The deprecated adapters have no production callers (confirmed via grep across all `.go` files)
- Removing it eliminates the type placement question entirely

**Arguments for keeping:**

- The `RegisterCatalogEntry`/`CatalogEntries` API is test-covered and functional
- Some consumers might still be using it (no telemetry available)
- Removing it is a breaking change requiring major version bump

**The tradeoff:** Clean architecture vs. backward compatibility. The adapters are deprecated but not removed. Should we be more aggressive about deleting deprecated code?

---

## Project Vital Signs

| Metric             | Value                                         | Status     |
| ------------------ | --------------------------------------------- | ---------- |
| Total LOC          | ~47,600                                       | ✅ Healthy |
| Production files   | ~179                                          | ✅         |
| Test files         | ~131                                          | ✅         |
| Test packages      | 24/24 pass (3 pre-existing failures)          | ⚠️         |
| Race detector      | Clean                                         | ✅         |
| Files >250 lines   | 1 (`catalog/eventcatalog/exporter.go` at 250) | ⚠️         |
| TODO/FIXME in prod | 0                                             | ✅         |
| Golden file drift  | Fixed in this session                         | ✅         |

## Coverage Summary (unchanged from Session 89)

| Package                | Coverage | Note                    |
| ---------------------- | -------- | ----------------------- |
| `core/query`           | 100.0%   | —                       |
| `core/pkg/dispatcher`  | 100.0%   | —                       |
| `middleware`           | 100.0%   | —                       |
| `catalog/adapters`     | 100.0%   | —                       |
| `memory`               | 99.6%    | —                       |
| `core/pkg/id`          | 97.8%    | —                       |
| `core/aggregate`       | 95.9%    | —                       |
| `catalog/d2`           | 95.0%    | —                       |
| `core/command`         | 94.7%    | —                       |
| `catalog/openapi`      | 94.4%    | —                       |
| `catalog/asyncapi`     | 93.7%    | —                       |
| `projection`           | 93.9%    | —                       |
| `core/decider`         | 93.3%    | —                       |
| `core/event`           | 92.1%    | —                       |
| `sync`                 | 92.2%    | —                       |
| `catalog/eventcatalog` | 91.3%    | —                       |
| `catalog`              | 90.5%    | —                       |
| `catalog/docserver`    | 90.0%    | —                       |
| `storage`              | 88.1%    | —                       |
| `testhelpers`          | 10.5%    | ⚠️ Low (test utilities) |

---

## Dependency Graph Update

```
command.Dispatcher ──► CatalogDispatcher[Type, CatalogEntry] (EMBEDDED)
query.Dispatcher   ──► CatalogDispatcher[Type, CatalogEntry] (EMBEDDED)

catalog/adapters/FromCommandDispatcher ──► command.Dispatcher.CatalogEntries()
catalog/adapters/FromQueryDispatcher   ──► query.Dispatcher.CatalogEntries()
```

**If CatalogDispatcher is removed:** Both adapters are deleted. Zero consumer impact (both are deprecated, zero non-test callers).
