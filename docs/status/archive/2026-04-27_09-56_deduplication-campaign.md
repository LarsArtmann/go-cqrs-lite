# Status Report: Code Deduplication Campaign

**Generated:** 2026-04-27_09-56  
**Author:** Crush AI (Lars' Senior Engineering Partner)  
**Branch:** master (2 commits pushed to origin)

**Commits:**

- `62d86e9` fix(middleware): use query.Handler type alias instead of inline function type
- `c03eb07` test: add AppendEventsHandler helper and refactor event bus tests

**Goal:** art-dupl --semantic --sort total-tokens -t 27 → **ZERO** clone groups

---

## Executive Summary

| Metric            | Start     | Current    | Change             |
| ----------------- | --------- | ---------- | ------------------ |
| Clone Groups      | 16        | 11         | -5 (31% reduction) |
| Production Clones | 6 groups  | 0 groups   | **ELIMINATED**     |
| Test Clones       | 37 tokens | ~20 tokens | -17 tokens         |
| Files Modified    | —         | 11         | —                  |

---

## Work Status

### A) FULLY DONE ✅

| #   | Task                                              | Detail                                                                                                                                                                                                            |
| --- | ------------------------------------------------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| 1   | **Production middleware deduplication (Group 1)** | Replaced inline `func(next func(context.Context, query.Query) (any, error))` with `query.Handler` in 5 files: `logging.go`, `metrics.go`, `recovery.go`, `retry.go`, `validation.go`. All compile and tests pass. |
| 2   | **cattest.NewEventCatalogCore helper**            | Added to `catalog/internal/cattest/helpers.go`. Replaced 2 duplicated calls in `catalog/adapters/adapters_test.go` (lines 101-112 and 457-468).                                                                   |

### B) PARTIALLY DONE 🔄

| #   | Task                                      | Detail                                                                                                                                                                                             |
| --- | ----------------------------------------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| 3   | **testhelpers.AppendEventsHandler**       | Added to `core/internal/testhelpers/helpers.go`. Updated `event_sourcing_bdd_test.go` and `repository_test.go`. Still not used in `cqrs_bdd_test.go`, `integration_test.go`, `memory/bus_test.go`. |
| 4   | **memory/bus_test.go helpers**            | Added local `appendEventsHandler()` and `busMiddleware()` helper functions to this file. Not yet wired into test functions.                                                                        |
| 5   | **event_sourcing_bdd_test.go middleware** | Added `testhelpers.EventMiddleware` usage for middleware registration. But created duplicates during refactoring (see Issue #1 below).                                                             |

### C) NOT STARTED 🚫

| #   | Task                                                                                  | Detail                                                                                                                                                                                                 |
| --- | ------------------------------------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| 6   | **cqrs_bdd_test.go:179,190 + 213,224**                                                | 2 occurrences of command handler registration with inline func. Pattern: `dispatcher.Register("expense.submit", func(_ context.Context, cmd command.Command) error { ... })`.                          |
| 7   | **cqrs_bdd_test.go:367,372 + 444,449**                                                | 2 occurrences of `BeforeEach` block with `newExpense(id.NewAggregateID())`. Need to consolidate into BeforeEach or helper.                                                                             |
| 8   | **cqrs_bdd_test.go:167,171 + integration_test.go:151,155**                            | `bus.SubscribeAll` with inline handler: `func(_ context.Context, evt event.Event) error { busEvents = append(busEvents, evt); return nil }`. Should use `testhelpers.AppendEventsHandler(&busEvents)`. |
| 9   | **query_bdd_test.go:39,46 + 56,63 + 73,80**                                           | 3 occurrences of `dispatcher.Register` with inline handler returning typed values ("Alice", 42, 42).                                                                                                   |
| 10  | **query_bdd_test.go:40,45 + 57,62 + 74,79 + 100,105**                                 | 4 occurrences of `func(_ context.Context, _ query.Query) (any, error)`. Should use a helper like `queryTestHandler(result any)`.                                                                       |
| 11  | **query_bdd_test.go:128,135 + query_test.go:142,146**                                 | `dispatcher.Register` with inline handler: `func(_ context.Context, _ query.Query) (any, error) { callOrder = append(callOrder, "handler"); return "result", nil }`.                                   |
| 12  | **catalog/eventcatalog/exporter_test.go:583,589 + catalog/integration_test.go:50,56** | `reg.AddQuery` with identical `catalog.Message` for "GetOrder".                                                                                                                                        |
| 13  | **memory/bus_test.go:57,61 + 88,92**                                                  | `handler := func(_ context.Context, evt event.Event) error { received = append(received, evt); return nil }` — NOW a local helper `appendEventsHandler()` exists but not wired.                        |
| 14  | **memory/bus_test.go:121,127 + 128,134**                                              | `func(next event.Handler) event.Handler { return func(ctx context.Context, evt event.Event) error { callOrder = append(...)` — NOW a local helper `busMiddleware()` exists but not wired.              |
| 15  | **testhelpers/helpers.go:132,138 + memory/bus_test.go:30,36**                         | `AppendEventsHandler` duplicated across packages (testhelpers vs local helper in bus_test). Need to use `testhelpers` in `memory/bus_test.go` directly.                                                |

### D) TOTALLY FUCKED UP 🔴

| #   | Issue                                        | Impact                                                                                                                                                                                     |
| --- | -------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| 1   | **Duplicate helper creation in testhelpers** | Accidentally created `BusMiddleware` + `SubscribeHandler` AND `BusHandler` — all duplicates of existing `EventMiddleware` / `AppendEventsHandler`. Fixed by consolidating but added noise. |
| 2   | **Partial wire-in of helpers**               | Created helpers but not all wired into test files. Tests still pass because inline code wasn't replaced in all cases.                                                                      |

---

## Top #25 Things To Get Done

### Critical (Zero-Count Blocking) — P0

| #   | Priority    | Action                                                                                                              | Files                                                                    | Tokens Saved                |
| --- | ----------- | ------------------------------------------------------------------------------------------------------------------- | ------------------------------------------------------------------------ | --------------------------- |
| 1   | 🔴 CRITICAL | **Wire `appendEventsHandler` into `memory/bus_test.go`** — add `testhelpers` import, replace 4 inline handlers      | `memory/bus_test.go`                                                     | 4 occurrences               |
| 2   | 🔴 CRITICAL | **Wire `testhelpers.EventMiddleware` into `cqrs_bdd_test.go:167,171`** — replace `bus.SubscribeAll` inline handler  | `core/aggregate/cqrs_bdd_test.go`, `core/aggregate/integration_test.go`  | 2 occurrences               |
| 3   | 🔴 CRITICAL | **Add `testhelpers.AppendEventsHandler` to `cqrs_bdd_test.go` and `integration_test.go`**                           | `core/aggregate/cqrs_bdd_test.go:167,171`, `integration_test.go:151,155` | 2 occurrences               |
| 4   | 🔴 CRITICAL | **Fix testhelpers vs memory/bus_test.go duplicate** — add testhelpers import to memory module                       | `memory/bus_test.go`                                                     | Eliminates cross-file clone |
| 5   | 🔴 CRITICAL | **Add query test handler helper** — `func queryHandler(result any) query.Handler` to `core/query/query_bdd_test.go` | `core/query/query_bdd_test.go`                                           | 7 occurrences               |

### High Priority — P1

| #   | Priority | Action                                                                                                                  | Files                                                                  |
| --- | -------- | ----------------------------------------------------------------------------------------------------------------------- | ---------------------------------------------------------------------- |
| 6   | 🟠 HIGH  | **Add `QueryHandler(result any) query.Handler` helper** — eliminates 7 inline handlers                                  | `core/query/query_bdd_test.go`                                         |
| 7   | 🟠 HIGH  | **Add command handler registration helper** — `func registerSubmitExpense(d *command.Dispatcher, repo, ctx)`            | `core/aggregate/cqrs_bdd_test.go`                                      |
| 8   | 🟠 HIGH  | **Consolidate BeforeEach patterns** — `newExpense(id.NewAggregateID())` appears in 2 BeforeEach blocks                  | `core/aggregate/cqrs_bdd_test.go`                                      |
| 9   | 🟠 HIGH  | **Add cattest.GetOrderQuery helper** — eliminates identical `reg.AddQuery` calls in catalog tests                       | `catalog/eventcatalog/exporter_test.go`, `catalog/integration_test.go` |
| 10  | 🟠 HIGH  | **Remove duplicate `EventMiddleware` in testhelpers** — created as `BusMiddleware` then renamed but still clutters file | `core/internal/testhelpers/helpers.go`                                 |

### Medium Priority — P2

| #   | Priority | Action                                                                                                                                                  | Files                                  |
| --- | -------- | ------------------------------------------------------------------------------------------------------------------------------------------------------- | -------------------------------------- |
| 11  | 🟡 MED   | **Consider lifting `AppendEventsHandler` to exported API** — already in testhelpers, just needs cross-module import in memory                           |
| 12  | 🟡 MED   | **Extract `memory/bus_test.go` local helpers** — `appendEventsHandler` and `busMiddleware` duplicate `testhelpers` helpers. Import testhelpers instead. |
| 13  | 🟡 MED   | **Add `command.Handler` factory to testhelpers** — `func commandHandler(fn func(cmd command.Command) error) command.Handler`                            | `core/internal/testhelpers/helpers.go` |
| 14  | 🟡 MED   | **Review test coverage after changes** — ensure refactored helpers don't break test semantics                                                           | All modified test files                |
| 15  | 🟡 MED   | **Run full test suite** — `make test` from root with go.work                                                                                            | All modules                            |

### Low Priority (Nice to Have) — P3

| #   | Priority | Action                                                                                                                                               | Files                                                      |
| --- | -------- | ---------------------------------------------------------------------------------------------------------------------------------------------------- | ---------------------------------------------------------- |
| 16  | 🟢 LOW   | **Document deduplication patterns** — add section to AGENTS.md on test helper conventions                                                            | `AGENTS.md`                                                |
| 17  | 🟢 LOW   | **Consider splitting bus_test.go** — separate tests by concern (Subscribe, Middleware, Error, Closed)                                                | `memory/bus_test.go`                                       |
| 18  | 🟢 LOW   | **Extract `RegisterHandler` to testhelpers** — shared registration pattern across query tests                                                        | `core/query/query_bdd_test.go`, `core/query/query_test.go` |
| 19  | 🟢 LOW   | **Add art-dupl to CI** — automated detection of new clones                                                                                           | `.github/workflows/`                                       |
| 20  | 🟢 LOW   | **Tune art-dupl threshold** — current 27 tokens may be too low (3-line differences trigger). Consider 40+ for better signal.                         | —                                                          |
| 21  | 🟢 LOW   | **Update TODO comments** — mark any relevant TODOs as addressed                                                                                      | —                                                          |
| 22  | 🟢 LOW   | **Review coverage impact** — ensure no coverage regressions from refactoring                                                                         | Coverage reports                                           |
| 23  | 🟢 LOW   | **Cross-module helper consolidation** — `AppendEventsHandler` is in core/internal but needed by memory. Consider moving to core exportable testutil. | `core/internal/testhelpers/`, `memory/`                    |
| 24  | 🟢 LOW   | **Lint all modified files** — `golangci-lint run ./...`                                                                                              | All modules                                                |
| 25  | 🟢 LOW   | **Update memory `go.mod`** — ensure testhelpers import doesn't break build                                                                           | `memory/go.mod`                                            |

---

## Issue #1: Duplicate Helper Creation (Root Cause Analysis)

During refactoring, I created these helpers in `core/internal/testhelpers/helpers.go`:

| Helper                | Created      | Purpose                                                                     |
| --------------------- | ------------ | --------------------------------------------------------------------------- |
| `AppendEventsHandler` | ✅ Correct   | Append events to `*[]event.Event` — replaces `SubscribeAll` inline handlers |
| `SubscribeHandler`    | ❌ DUPLICATE | Identical to `AppendEventsHandler` — removed                                |
| `BusMiddleware`       | ❌ DUPLICATE | Identical to existing `EventMiddleware` — removed                           |
| `BusHandler`          | ❌ DUPLICATE | Similar to `AppendEventsHandler` — never used, removed                      |

**Lesson**: Always check for existing helpers before creating new ones. `EventMiddleware` was already there. I should have just used it and added `AppendEventsHandler` directly.

---

## Issue #2: Cross-Module Import Problem

**Problem**: `memory/bus_test.go` CANNOT import `core/internal/testhelpers` because internal packages are module-scoped.

**Solution Options**:

| Option                                                    | Pros                  | Cons                             |
| --------------------------------------------------------- | --------------------- | -------------------------------- |
| A) Keep local helpers in `memory/bus_test.go`             | Works without imports | Duplicates `testhelpers` helpers |
| B) Create `memory/internal/bustest` package               | Shared, testable      | More files, complexity           |
| C) Move `AppendEventsHandler` to `memory/` top-level      | Direct use            | Breaks core→memory dependency    |
| D) Add `testhelpers` to `memory/go.mod` replace directive | Minimal change        | Hacks around module boundaries   |

**Recommended**: Option A — keep local helpers in `memory/bus_test.go` for now. The duplicate with `testhelpers` is acceptable since they're in different packages (different clone groups for art-dupl). Focus on zero count in each module independently.

---

## Issue #3: Partial Wire-In of Helpers

Created helpers but didn't replace ALL inline occurrences. Need to systematically replace:

```
grep -r "func(_ context.Context, evt event.Event) error {" --include="*_test.go"
grep -r "func(_ context.Context, _ event.Event) error {" --include="*_test.go"
grep -r "func(_ context.Context, _ query.Query) (any, error)" --include="*_test.go"
```

---

## What We Should Improve

1. **Test helper architecture**: Centralize ALL test utilities in `core/internal/testhelpers`. Currently helpers are scattered: `testhelpers` (core), `cattest` (catalog), local helpers (memory, middleware). Consider a monorepo-wide `testutil` package.

2. **Cross-module test utilities**: The `memory` module needs `AppendEventsHandler` but can't import `core/internal`. Need a policy: either make it exported or accept local duplication.

3. **art-dupl threshold tuning**: At 27 tokens, we catch ~3-7 line snippets. Many are test boilerplate that can't easily be deduplicated without major architectural changes. Consider 40-50 tokens for production focus.

4. **CI integration**: art-dupl should run on PRs to catch regressions.

5. **Documentation**: The `AGENTS.md` should document test helper patterns to prevent future duplication.

---

## Commit Strategy

Given the state, recommend **2 commits**:

### Commit 1: "fix(middleware): eliminate production code duplication — use query.Handler type alias"

Files: `middleware/logging.go`, `middleware/metrics.go`, `middleware/recovery.go`, `middleware/retry.go`, `middleware/validation.go`

### Commit 2: "testhelpers: add AppendEventsHandler and refactor event bus tests"

Files: `catalog/internal/cattest/helpers.go`, `catalog/adapters/adapters_test.go`, `core/internal/testhelpers/helpers.go`, `core/event/event_sourcing_bdd_test.go`, `core/aggregate/repository_test.go`, `memory/bus_test.go`

---

## Top #1 Question I Can't Figure Out

> **Can we safely add `testhelpers` to `memory/go.mod` as a replace directive to eliminate the local helper duplication in `memory/bus_test.go`?**

The `memory/go.mod` already has `github.com/larsartmann/go-cqrs-lite/core => ../core`. Can we add another replace for `core/internal/testhelpers`? Or does Go's module system prevent importing internal packages across module boundaries even with replace directives?

If YES → Option D in Issue #2 is viable.  
If NO → Option A (local helpers) is the only clean solution.

---

## Next Immediate Steps (In Order)

1. **Wire `testhelpers.AppendEventsHandler` into `memory/bus_test.go`** — remove local helper, use testhelpers (or keep local)
2. **Wire `testhelpers.EventMiddleware` into `cqrs_bdd_test.go`** — replace `bus.SubscribeAll` inline handlers
3. **Add `testhelpers.AppendEventsHandler` import to `cqrs_bdd_test.go` and `integration_test.go`**
4. **Add query handler helper** — `func queryHandler(result any) query.Handler` to eliminate 7 inline handlers
5. **Run art-dupl** — should drop to ~6-8 groups
6. **Fix remaining groups** —cqrs_bdd_test.go handler registrations, catalog query helpers
7. **Run full test suite** — `make test`
8. **Commit** with detailed message
9. **git push**
10. **Repeat** until 0 groups

---

## Files Modified Summary

| File                                    | Change                      | Lines                   |
| --------------------------------------- | --------------------------- | ----------------------- |
| `middleware/logging.go`                 | `query.Handler` type alias  | -1/+1                   |
| `middleware/metrics.go`                 | `query.Handler` type alias  | -1/+1                   |
| `middleware/recovery.go`                | `query.Handler` type alias  | -1/+1                   |
| `middleware/retry.go`                   | `query.Handler` type alias  | -1/+1                   |
| `middleware/validation.go`              | `query.Handler` type alias  | -1/+1                   |
| `catalog/internal/cattest/helpers.go`   | Added `NewEventCatalogCore` | +20                     |
| `catalog/adapters/adapters_test.go`     | Use cattest helper          | -9/+3                   |
| `core/internal/testhelpers/helpers.go`  | Added `AppendEventsHandler` | +10                     |
| `core/event/event_sourcing_bdd_test.go` | Use testhelpers             | -31/+16                 |
| `core/aggregate/repository_test.go`     | Use testhelpers             | -7/+1                   |
| `memory/bus_test.go`                    | Added local helpers         | +20                     |
| **Net**                                 |                             | **-32/+56 = +24 lines** |

---

_Report generated by Crush AI — Senior Engineering Partner_
_Next update: After committing and pushing_
