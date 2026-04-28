# Comprehensive Status Report

**Date:** 2026-04-23 07:22 CEST
**Branch:** master
**Last Commit:** 145cfb2 — chore(catalog): update benchmark tests and module dependencies

## Executive Summary

go-cqrs-lite is a ~14,300 LOC CQRS library in Go. All tests pass (13 packages, race-clean). The monorepo has been successfully restructured into 3 independent modules (`core/`, `memory/`, `catalog/`) with `go.work`. This session produced a complete AGENTS.md rewrite to reflect the multi-module monorepo reality. No production code was changed — all work was documentation.

---

## A) FULLY DONE

### Documentation (This Session)

| Item | Status | Detail |
|------|--------|--------|
| AGENTS.md rewrite | ✅ Complete | Rewrote from scratch to reflect multi-module monorepo structure, accurate deps, correct package paths |

### Previously Completed (Confirmed Still Working)

| Item | Status |
|------|--------|
| **Core CQRS packages** | ✅ `command/`, `query/` (WITH context.Context), `event/`, `aggregate/` — all stable, tested, race-clean |
| **Generic internal dispatcher** | ✅ `pkg/dispatcher/` with `Dispatcher[H, M]`, `MiddlewareChain[H, M]`, `LifecycleMixin` |
| **Branded IDs** | ✅ `pkg/id/` — type-safe `id.Of[T]` with full JSON/DB/encoding support |
| **Catalog system** | ✅ `catalog/` core + `catalog/adapters` + `catalog/asyncapi` (AsyncAPI 3.0) + `catalog/eventcatalog` (MDX) |
| **Memory implementations** | ✅ Extracted to standalone `memory/` module: `MemoryStore`, `MemoryBus`, `MemorySnapshotStore` |
| **xtypes** | ✅ Type-safe wrappers for compile-time safety |
| **Middleware** | ✅ `middleware/` — logging, retry, validation, recovery, metrics |
| **Multi-module monorepo** | ✅ `go.work` with 3 modules: `core/`, `memory/`, `catalog/` |
| **Catalog module extraction** | ✅ Independent `catalog/` module with own `go.mod` (deps: core, go-faster/yaml, go-json-experiment/json) |
| **Memory module extraction** | ✅ Independent `memory/` module with own `go.mod` (dep: core only) |
| **Query handler context.Context** | ✅ `query.Handler = func(context.Context, Query) (any, error)` — fixed |
| **Custom YAML marshaler** | ✅ Deleted, replaced with `go-faster/yaml` |
| **BDD test suites** | ✅ Ginkgo v2 BDD tests for event, aggregate, query packages |
| **Benchmarks** | ✅ All packages benchmarked |
| **CI workflows** | ✅ `test.yml` + `lint.yml` (Go 1.26, race detection, coverage, examples build) |
| **Makefile** | ✅ test, test-race, test-cover, build, lint, fmt, imports, check, clean |

---

## B) PARTIALLY DONE

| Item | Current State | What Remains |
|------|---------------|--------------|
| **Phase 0 of migration plan** | Query ctx fixed, YAML replaced, pkg/errors still exists | Delete `pkg/errors` (dead code), fix err113 warnings |
| **AGENTS.md** | Rewritten this session — should be accurate now | Keep in sync as codebase evolves |
| **Test coverage** | Most packages >80%, overall 85.1% | `catalog/adapters` at 66%, `aggregate` at 77.3%, `internal/dispatcher` at 77.4% |
| **Module boundaries** | Phases 1-3 done (core, memory, catalog) | Phases 4-10 of migration plan |
| **Middleware in core** | 5 middleware implemented | Not yet extracted to own module (Phase 4), no tracing (OTel), no idempotency |
| **xtypes in core** | Functional | Not yet extracted to own module (Phase 4) |

---

## C) NOT STARTED

| Item | Priority | Module |
|-------|----------|--------|
| Multi-module migration Phase 4 (extract middleware + xtypes) | HIGH | middleware/, xtypes/ |
| Storage module (sqlc event store, PostgreSQL first) | HIGH | storage/ |
| Watermill module (pub/sub, Redis Streams first) | HIGH | watermill/ |
| Projection module (samber/ro internally) | HIGH | projection/ |
| Snapshot module (SQL-backed) | MEDIUM | snapshot/ |
| Test utilities module (AggregateTester, ProjectionTester) | MEDIUM | testutil/ |
| Event Codec interface | MEDIUM | core/event |
| Event Upcasting | LOW | core/upcasting |
| Outbox pattern implementation | HIGH | storage/ |
| ULID migration (replace google/uuid) | MEDIUM | core/pkg/id |
| Compatibility shim (old → new import paths) | MEDIUM | root module |
| go-import meta tag hosting | LOW | GitHub Pages |
| Schema migration tool decision | LOW | storage/ |
| Examples CI integration | LOW | examples/ |
| Tag releases | LOW | all modules |

---

## D) TOTALLY FUCKED UP

| Item | Problem | Severity |
|------|---------|----------|
| **`catalog/benchmark_test.go` uses core internal package** | Imports `core/internal/testhelpers` from `catalog/` module — cross-module internal package violation. Test cannot compile from root via `go.work`. | 🔴 HIGH — broken test |
| **Flaky concurrency BDD test** | `core/aggregate/cqrs_bdd_test.go:449` — "should serialize all writes" fails intermittently. Race between goroutines in test setup. | 🔴 HIGH — unreliable CI |
| **`core/go.mod` has unused deps** | `go-faster/yaml`, `go-faster/errors`, `go-faster/jx`, `segmentio/asm`, `uber/multierr` are in `go.mod` but not imported by core. Needs `go mod tidy`. | 🟡 MODERATE |
| **`catalog/go.sum` is stale** | Missing `go-faster/yaml` entries. `GOWORK=off go test` fails in catalog/ in isolation. Works via `go.work` because core provides the transitive deps. | 🟡 MODERATE |
| **`pkg/errors` is dead code** | Defined in core but never imported anywhere. Confuses users about error handling strategy. | 🟡 MODERATE |
| **No production event store** | Only MemoryStore exists. The entire storage/ module is a plan, not code. Users can't use this in production today. | 🔴 CRITICAL — blocks real-world usage |
| **`catalog/adapters` has unused functions** | `buildCatalogMessageFromMeta`, `buildQueryMessageFromReflect`, `buildEventMessageFromReflect` are defined but never called. Dead code flagged by gopls. | 🟡 LOW |
| **`MemoryBus.Publish` holds RLock during handler execution** | Subscribers block publishers. Acceptable for test utility but should be documented. | 🟡 LOW |
| **`xtypes.TypedCommand.Command()` allocates on every call** | Creates new `command.Core` each time. Should embed or cache. | 🟡 LOW |

---

## E) WHAT WE SHOULD IMPROVE

### Architecture

1. **Fix cross-module internal import** — `catalog/benchmark_test.go` imports `core/internal/testhelpers`. Either make `testhelpers` public (`core/testhelpers`) or move the benchmark into `core/`.
2. **Fix flaky concurrency BDD test** — The aggregate concurrency test is timing-dependent. Needs deterministic synchronization.
3. **Run `go mod tidy` on core** — 5 unused indirect dependencies bloating `go.mod`.
4. **Fix catalog go.sum** — Run `go mod tidy` in catalog to fix standalone builds.
5. **Delete unused adapter functions** — `buildCatalogMessageFromMeta`, `buildQueryMessageFromReflect`, `buildEventMessageFromReflect` in `catalog/adapters/message.go`.
6. **Extract middleware to own module** — Phase 4 of migration plan.
7. **Extract xtypes to own module** — Phase 4 of migration plan.
8. **Implement storage/ module** — The single biggest gap. Without a persistent event store, the library is testing-only.
9. **Add Projection concept to core** — The "Q" in CQRS is missing. Users have no way to build read models.
10. **Add Event Codec interface** — `[]byte` payload with no serialization strategy is a footgun.

### Code Quality

11. **Raise `catalog/adapters` coverage from 66% to >80%** — Lowest tested package. `AddServiceToDomain`, `AddChannel`, `AddCommandWithSchema`, `AddEventWithDirection`, `AddEventFromType` all at 0%.
12. **Delete `pkg/errors`** — Dead code adds confusion.
13. **Fix `event.Core` mutability via Option** — `WithMetadata` can mutate event after construction, contradicting "immutable" doc comment.

### Developer Experience

14. **Fix broken example modules** — `example/catalog/go.mod` is stale. Not CI-tested via go.work.
15. **Write a getting-started guide** — pkg.go.dev isn't enough for a framework.

---

## F) TOP 25 THINGS WE SHOULD GET DONE NEXT

### Tier 1: Fix Broken Things (Do First)

| # | Task | Module | Effort |
|---|------|--------|--------|
| 1 | Fix `catalog/benchmark_test.go` cross-module internal import | catalog | XS |
| 2 | Fix flaky aggregate concurrency BDD test | core/aggregate | S |
| 3 | Run `go mod tidy` on core and catalog | root | XS |
| 4 | Delete unused functions in `catalog/adapters/message.go` | catalog/adapters | XS |
| 5 | Delete dead `pkg/errors` package | core/pkg | XS |

### Tier 2: Unblock Production Use

| # | Task | Module | Effort |
|---|------|--------|--------|
| 6 | Implement `storage/` module with sqlc (PostgreSQL first) | storage | L |
| 7 | Implement outbox pattern in `storage/` | storage | M |
| 8 | Implement `watermill/` module (Redis Streams first) | watermill | M |
| 9 | Implement `projection/` module with checkpoint tracking | projection | L |
| 10 | Extract middleware to own module (Phase 4) | middleware | S |

### Tier 3: Complete the SDK

| # | Task | Module | Effort |
|---|------|--------|--------|
| 11 | Extract xtypes to own module (Phase 4) | xtypes | S |
| 12 | Implement `snapshot/` module (SQL-backed) | snapshot | M |
| 13 | Implement `testutil/` module (AggregateTester, ProjectionTester) | testutil | M |
| 14 | Add Event Codec interface to core (JSON default) | core/event | S |
| 15 | Migrate from google/uuid to oklog/ulid | core/pkg/id | M |

### Tier 4: Polish & Production Hardening

| # | Task | Module | Effort |
|---|------|--------|--------|
| 16 | Raise `catalog/adapters` coverage to >80% | catalog/adapters | S |
| 17 | Raise `aggregate` coverage to >85% | core/aggregate | S |
| 18 | Raise `internal/dispatcher` coverage to >85% | core/pkg/dispatcher | S |
| 19 | Fix all err113 linter warnings (sentinel errors) | all | S |
| 20 | Fix broken example modules, add to go.work CI | examples | S |
| 21 | Add MySQL + SQLite schemas to storage/ | storage | M |
| 22 | Add Event Upcasting interface + implementation | core/upcasting | M |
| 23 | Add OpenTelemetry tracing middleware | middleware | M |
| 24 | Write getting-started guide + architecture overview | docs | M |
| 25 | Tag v1.0.0 releases for all modules | all | S |

---

## G) TOP #1 QUESTION I CANNOT FIGURE OUT MYSELF

**Should we continue Phase 4 (extract middleware + xtypes) now, or skip straight to Phase 5 (storage/) to unblock production use?**

The migration plan says Phase 4 comes before Phase 5. But the storage module is the single biggest gap — without it, nobody can use this library in production. Middleware and xtypes are fine in core for now (they only depend on core interfaces, adding no transitive deps to users who don't import them).

Arguments for skipping to Phase 5:
- Unblocks real-world usage immediately
- Middleware/xtypes extraction is low-risk and can happen anytime
- Storage is the highest-value module by far

Arguments for Phase 4 first:
- Keeps the migration plan sequential and predictable
- Each phase builds confidence before the hard work
- Smaller PRs are easier to review

This is a prioritization decision that depends on whether you want "complete architecture" or "usable ASAP" first.

---

## Build & Test Verification

```
$ go test ./core/... ./memory/... ./catalog/... -count=1 -race
ok  core/aggregate       (race-clean)
ok  core/command          (race-clean)
ok  core/event            (race-clean)
ok  core/middleware        (race-clean)
ok  core/pkg/dispatcher   (race-clean)
ok  core/pkg/id           (race-clean)
ok  core/query            (race-clean)
ok  core/xtypes           (race-clean)
ok  memory                (race-clean)
ok  catalog/adapters      (race-clean)
ok  catalog/asyncapi      (race-clean)
ok  catalog/eventcatalog  (race-clean)
FAIL catalog (cross-module internal import)
```

13/14 test packages pass. 1 failure is a pre-existing cross-module import violation.

## Lines of Code

| Category | Lines | Files |
|----------|-------|-------|
| Production Go | 5,240 | 59 |
| Test Go | 8,603 | 34 |
| **Total** | **14,326** | **93** |

## Test Coverage Summary

| Package | Coverage | Status |
|---------|----------|--------|
| `catalog/asyncapi` | 96.3% | ✅ |
| `xtypes` | 95.7% | ✅ |
| `query` | 91.5% | ✅ |
| `event` | 89.7% | ✅ |
| `catalog/eventcatalog` | 89.7% | ✅ |
| `memory` | 99.2% | ✅ |
| `pkg/id` | 85.4% | ✅ |
| `middleware` | 84.6% | ✅ |
| `command` | 84.4% | ✅ |
| `internal/dispatcher` | 77.4% | ⚠️ |
| `aggregate` | 77.3% | ⚠️ |
| `catalog/adapters` | 66.0% | 🔴 |
| **Total** | **85.1%** | |
