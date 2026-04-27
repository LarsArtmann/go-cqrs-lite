# Status Report: Deduplication Campaign — Round 3 (WIP)

**Generated:** 2026-04-27_12-10  
**Author:** Crush AI (Lars' Senior Engineering Partner)  
**Branch:** master (up to date with origin, UNCOMMITTED changes in working tree)  
**Goal:** `art-dupl --semantic --sort total-tokens -t 27` → **ZERO** clone groups

---

## Executive Summary

| Metric | Start (R1) | After R1 | After R2 | Current (R3 WIP) | Change |
|--------|-----------|----------|----------|-------------------|--------|
| Clone Groups | 16 | 11 | 5 | **3** | -13 (81% reduction) |
| Production Clones | 6 groups | 0 | 0 | **0** | ELIMINATED |
| Cross-Module Test Clones | 2 groups | 2 | 2 | **0** | ELIMINATED |
| Same-File Test Clones | 8 groups | 5 | 3 | **3** | Remaining |

**Current state: 3 clone groups remain, all in `core/query/query_bdd_test.go`**

---

## A) FULLY DONE ✅

### Round 1 (commits 62d86e9, c03eb07)
- Middleware production clones: `query.Handler` type alias in 5 files (6 groups → 0)
- `cattest.NewEventCatalogCore` helper
- `testhelpers.AppendEventsHandler` initial helper

### Round 2 (commits e9edcae, 926d9ca, d15dfb3)
- `memory/bus_test.go`: local helpers wired in
- `catalog/internal/cattest/helpers.go`: `AddQuerySimple` helper
- `core/aggregate/cqrs_bdd_test.go`: `setupCQRSComponents()` + `registerSubmitExpenseHandler()` helpers
- `core/aggregate/integration_test.go`: testhelpers wiring
- `go-composable-business-types` v0.1.0 consistent across all modules
- Status report committed

### Round 3 (UNCOMMITTED — in working tree)

**Created `testhelpers/` shared module** — a new repo-root module that any module can import without Go internal boundary restrictions:

| File | Change |
|------|--------|
| `testhelpers/go.mod` | NEW: module depends only on `core` |
| `testhelpers/helpers.go` | NEW: 111 lines — shared `AppendEventsHandler`, `TestMetrics`, `NoopCommandHandler`, `NoopEventHandler`, `FailingCommandHandler`, `FailingEventHandler`, `PanicCommandHandler`, `PanicEventHandler`, `CallbackCommandHandler`, `CommandMiddleware`, `EventMiddleware` |
| `core/internal/testhelpers/helpers.go` | REWRITTEN: 138 → 35 lines. Re-exports from shared testhelpers via type alias + var delegation. Only `AssertCallOrder` remains local (uses `testing.T`). |
| `memory/bus_test.go` | Removed local `appendEventsHandler`, now uses `testhelpers.AppendEventsHandler` |
| `memory/go.mod` | Added `testhelpers` dependency + replace directive |
| `middleware/middleware_test.go` | Replaced local `testMetrics`, `noopCommandHandler`, `failingCommandHandler`, `panicCommandHandler`, `noopEventHandler`, `failingEventHandler`, `panicEventHandler`, `callbackCommandHandler` with `testhelpers.*`. Only `testLogger` stays local (implements `middleware.Logger` interface — import cycle if shared). |
| `middleware/go.mod` | Added `testhelpers` dependency + replace directive |
| `core/go.mod` | Added `testhelpers` dependency + replace directive |

**Net: -203 lines, +45 lines across 6 modified files + 1 new module (111 lines)**

### Tests: ALL 13 PACKAGES PASS ✅

```
ok  core/aggregate     ok  core/command     ok  core/event
ok  core/pkg/dispatcher ok  core/pkg/id     ok  core/query
ok  memory             ok  catalog          ok  catalog/adapters
ok  catalog/asyncapi   ok  catalog/eventcatalog
ok  middleware          ok  xtypes
```

---

## B) PARTIALLY DONE 🔧

### query_bdd_test.go deduplication (Groups 1+2+3)

**3 clone groups remain**, all in `core/query/query_bdd_test.go`:

| Group | Lines | Pattern | Can Eliminate? |
|-------|-------|---------|----------------|
| #1 | 39-46, 56-63, 73-80 | `Expect(dispatcher.Register("X", func(...) (any, error) { return Y, nil })).To(Succeed())` | **Hard** — different query types and return values each time. Would need a `registerQueryHandler` helper that accepts `query.Type` + `any` return value. |
| #2 | 40-45, 57-62, 74-79, 100-105 | `dispatcher.Register("X", func(...) (any, error) { return Y, nil })` (inner part of #1) | **Subset of #1** — fixing #1 fixes #2 |
| #3 | 128-135 ↔ 142-146 | Handler with `callOrder = append(callOrder, "handler")` between `query_bdd_test.go` and `query_test.go` | **Yes** — extract a `queryHandler(result any, callOrder *[]string)` helper |

---

## C) NOT STARTED ❌

1. **query_bdd_test.go helpers** — extract `registerQueryHandler` to eliminate Groups 1+2+3
2. **`go.work` update** — add `./testhelpers` to `go.work` use directives
3. **AGENTS.md update** — document new `testhelpers` module, module dependency graph changes
4. **Remove old status reports** — `docs/status/2026-04-27_deduplication-campaign-final.md` is now outdated
5. **Lint check** — `make lint` hasn't been run since these changes

---

## D) TOTALLY FUCKED UP 💥

### `testhelpers/go.mod` has unnecessary replace directive

The `testhelpers/go.mod` has `replace github.com/larsartmann/go-cqrs-lite/middleware => ../middleware` which was added during debugging of the `TestLogger` import cycle issue. **This replace directive is unnecessary** — `testhelpers` doesn't import `middleware` anymore (the `TestLogger` was moved back to `middleware_test.go` local). The `go mod tidy` pulled in `middleware` as a dependency when `TestLogger` was in `testhelpers/helpers.go` and created this stale replace. It should be removed.

### `testhelpers` pulled in `middleware` as a dependency during go.mod tidy

The `testhelpers/go.mod` currently has `github.com/larsartmann/go-cqrs-lite/middleware v0.0.0-20260427092005-d15dfb3f6606` in its require block. This was from when `TestLogger` was trying to implement `middleware.Logger`. After removing `TestLogger` from `testhelpers`, `go mod tidy` may or may not clean this up. **Needs verification.**

---

## E) WHAT WE SHOULD IMPROVE 🔍

1. **`testhelpers/go.mod` cleanup** — Remove stale `middleware` replace directive and dependency
2. **`go.work` update** — Add `./testhelpers` to `use()` block
3. **`core/internal/testhelpers` should only re-export** — Currently it's a thin wrapper; this is correct but adds a layer. Acceptable for backward compatibility with existing callers.
4. **`query_bdd_test.go` test readability** — Adding a helper may reduce BDD readability. Each test should tell a story. The inline handlers ARE the story. Consider raising threshold instead.
5. **Module dependency graph** is now: `testhelpers → core → memory`. `middleware → core + testhelpers`. `memory → core + testhelpers`. This is clean — no cycles.
6. **The `testLogger` in `middleware_test.go`** — Cannot be shared because it implements `middleware.Logger`, and importing `middleware` from `testhelpers` creates a cycle. This is an **acceptable limitation**.
7. **Consistent go.mod structure** — All modules now need `replace github.com/larsartmann/go-cqrs-lite/testhelpers => ../testhelpers`. This is boilerplate but necessary in a multi-module monorepo without published versions.

---

## F) TOP 25 THINGS TO DO NEXT (sorted by impact × effort)

### High Impact, Low Effort (do first)

| # | Task | Impact | Effort |
|---|------|--------|--------|
| 1 | Commit current R3 changes (shared testhelpers module) | Eliminates 2 cross-module groups | DONE, just commit |
| 2 | Clean up `testhelpers/go.mod` — remove stale middleware replace + dependency | Clean module graph | 2 min |
| 3 | Add `./testhelpers` to `go.work` | Correct workspace config | 1 min |
| 4 | Run `make lint` — verify no lint errors | Quality gate | 2 min |
| 5 | Extract `queryHandler(result, callOrder)` in `query_bdd_test.go` | Eliminates Group #3 | 5 min |

### Medium Impact, Low Effort

| # | Task | Impact | Effort |
|---|------|--------|--------|
| 6 | Update `AGENTS.md` — document testhelpers module | Documentation | 5 min |
| 7 | Remove outdated `docs/status/` reports | Housekeeping | 1 min |
| 8 | Verify `go mod tidy` in all modules is clean | Module health | 3 min |
| 9 | Consider raising art-dupl threshold to 30 for test files | Remaining groups at 30: 2 | Decision |
| 10 | Add `TestLogger` to shared `testhelpers` WITHOUT middleware dependency | Would need to define `Logger` interface in `core` instead of `middleware` | Architectural |

### Medium Impact, Medium Effort

| # | Task | Impact | Effort |
|---|------|--------|--------|
| 11 | Move `middleware.Logger` interface to `core` | Enables sharing TestLogger everywhere | Refactoring |
| 12 | Move `middleware.MetricsRecorder` interface to `core` | Enables sharing TestMetrics as interface impl | Refactoring |
| 13 | `query_bdd_test.go`: extract `registerConstHandler(qType, result)` | Eliminates Groups #1+#2 | Readability tradeoff |
| 14 | `core/internal/testhelpers`: remove entirely, update all imports to use shared `testhelpers` directly | Removes indirection layer | Large search-and-replace |
| 15 | Add `TestValidator` to shared testhelpers for validation middleware tests | Test consistency | 5 min |

### Lower Impact, Higher Effort

| # | Task | Impact | Effort |
|---|------|--------|--------|
| 16 | Create `Makefile` target for art-dupl with threshold | CI integration | 5 min |
| 17 | Add art-dupl to CI pipeline (GitHub Actions) | Continuous quality | 15 min |
| 18 | Publish `go-composable-business-types` v0.1.0 to proxy | Removes all replace directives | External |
| 19 | Tag all modules as v0.1.0 | Proper versioning | External |
| 20 | Storage module (Phase 5 from migration plan) | New functionality | Large |
| 21 | Watermill module (Phase 6) | New functionality | Large |
| 22 | Projection module (Phase 7) | New functionality | Large |
| 23 | Snapshot module (Phase 8) | New functionality | Large |
| 24 | Test utilities module (Phase 9) | May supersede testhelpers | Large |
| 25 | Tag releases (Phase 10) | Release | External |

---

## G) TOP #1 QUESTION ❓

**Should `query_bdd_test.go` groups #1 and #2 be "fixed" with a helper, or should we raise the art-dupl threshold to 30 for a zero-group result?**

The 3 remaining groups are all structural patterns in BDD tests where:
- The **code shape** is identical: `dispatcher.Register("X", func(...) (any, error) { return Y, nil })`
- The **values differ** each time: "Alice" vs 42 vs "" 
- Each test tells a **self-contained story** — extracting a helper would hurt readability
- At threshold 30, only 2 groups remain (Group #3 from `query_bdd_test.go` ↔ `query_test.go`)
- At threshold 40, **0 groups remain**

The right answer depends on whether you want zero groups as a **principle** or whether BDD test readability matters more. A helper like `registerConstHandler(qType string, result any)` would work but makes each test less self-documenting.
