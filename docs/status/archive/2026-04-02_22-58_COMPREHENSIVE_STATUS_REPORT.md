# Comprehensive Status Report — 2026-04-02 22:58

**Branch:** `master` | **Commit:** `507c97f` (docs) + uncommitted `RecordEvent` rename | **Pushed:** yes (committed), no (rename) | **Tests:** PASS (verified at 21:17) | **Build:** PASS (verified with `GOCACHE=$(mktemp -d)`)

---

## a) FULLY DONE (committed, verified, pushed)

### P0 Bug Fixes

1. **Store interface type mismatch** — `event.Store` now uses `id.AggregateID` + `event.Version` instead of `string`/`int`, matching `MemoryStore`. Added `var _ Store = (*MemoryStore)(nil)` compile-time check. (`64ffdba`)

2. **LoadFromHistory bug** — Previously only incremented version counter without calling `Apply`, meaning aggregate state was never reconstructed. Now accepts a `Root` parameter and calls `root.Apply(evt)` for each event. Updated `aggregate/aggregate.go`, `aggregate/aggregate_test.go`, `example/user/handlers.go`, `xtypes/aggregate.go`, `xtypes/xtypes_test.go`. (`0e5ec14`)

3. **Data race on Handlers map** — `Dispatcher.Handlers` (exported `map[string]H`) replaced with unexported `handlers map[string]H` + `handlersMu sync.RWMutex`. Added `GetHandler(t string) (H, bool)` method. Updated callers: `command/dispatcher.go`, `query/dispatcher.go`, `internal/dispatcher/dispatcher_test.go`. (`412a9a3`)

4. **MemoryStore thread safety** — Added `sync.RWMutex` to `MemoryStore` with proper RLock/Lock around all map operations (Save, Load, LoadFromVersion, Delete). (`412a9a3`)

### P1 Features & Refactoring

5. **Catalog system** — Added `catalog/`, `catalog/asyncapi/`, `catalog/eventcatalog/`, `catalog/yaml/` with full test coverage.

6. **Example user CQRS application** — `example/user/` with full event sourcing flow (CreateUser, ChangeEmail commands).

7. **CI workflows** — Fixed Go 1.26, CHANGELOG updated, TODO_LIST cleaned up, documentation updated.

8. **`fmt.Appendf` usage, tagged switches, asyncapi options pattern, YAML field ordering** — Various refactoring per gopls hints.

### Test Coverage (verified at 21:17, all 11 packages PASS)

| Package                 | Coverage |
| ----------------------- | -------- |
| `aggregate/`            | 92.3%    |
| `catalog/`              | 91.9%    |
| `catalog/asyncapi/`     | 92.6%    |
| `catalog/eventcatalog/` | 86.4%    |
| `catalog/yaml/`         | 79.8%    |
| `command/`              | 90.5%    |
| `event/`                | 93.2%    |
| `internal/dispatcher/`  | 100.0%   |
| `pkg/id/`               | 88.0%    |
| `query/`                | 92.6%    |
| `xtypes/`               | 95.6%    |

---

## b) PARTIALLY DONE

### 1. `Core.ApplyEvent` → `Core.RecordEvent` rename (UNCOMMITTED)

**Status:** Code changes complete in 5 files, build verified PASS, test compilation in progress (blocked by Go cache).

**Files modified:**

- `aggregate/aggregate.go` — Method renamed `ApplyEvent` → `RecordEvent`
- `aggregate/aggregate_test.go` — All test references updated (10 occurrences)
- `xtypes/aggregate.go` — `TypedAggregate.ApplyEvent` → `TypedAggregate.RecordEvent`, calls `core.RecordEvent`
- `xtypes/xtypes_test.go` — Test references updated (2 occurrences)
- `example/user/aggregate.go` — `u.ApplyEvent(ctx, evt)` → `u.RecordEvent(ctx, evt)` (2 occurrences)

**Rationale:** Disambiguates from `Root.Apply(event.Event) error` which rebuilds aggregate state from history. `RecordEvent` records a new event to the uncommitted changes list — fundamentally different purpose.

**Remaining:** Commit + verify tests pass.

### 2. Race detector verification

**Status:** The data race fix and MemoryStore thread safety changes are committed and pushed. Tests pass normally. Race detector test (`-race` flag) has NOT been successfully completed due to Go build cache corruption. The fix itself is correct (uses `sync.RWMutex` properly), but formal `-race` verification is blocked.

---

## c) NOT STARTED

### P1 — Design Improvements

1. **`aggregate.Repository` interface** — Generic repository backed by Store+Bus. The example already has a concrete `Repository` struct in `example/user/handlers.go` that can serve as the design reference.

2. **Add `context.Context` to `query.Handler`** — Currently `type Handler = func(Query) (any, error)`, inconsistent with command/event handler patterns which accept `context.Context`.

### P2 — Consistency

3. **Standardize error wrapping** — Replace `fmt.Errorf` in `event/event.go`, `event/types.go`, `xtypes/event.go` with `cockroachdb/errors` for consistent stack traces.

4. **Remove duplicated `Lifecycle` methods** — `internal/dispatcher/dispatcher.go:42-69` re-declares Close/IsClosed/CheckClosed with identical implementations to `LifecycleMixin`. Should just embed `LifecycleMixin` without re-declaring.

5. **Fix `internal/dispatcher.Register` error semantics** — Uses `ErrHandlerNotFound` when closed, semantically wrong. Should use `ErrDispatcherClosed`.

### P3 — New Features

6. **Integration test** — Full CQRS roundtrip in `example/user/` or `integration_test/`.

7. **Benchmarks** — For ID generation, dispatcher throughput, event store operations.

8. **`middleware/logging.go`** — Structured logging middleware for command/query dispatchers.

9. **`middleware/recovery.go`** — Panic recovery middleware.

10. **`middleware/tracing.go`** — OpenTelemetry-compatible tracing middleware.

---

## d) TOTALLY FUCKED UP

### Go Build Cache Corruption (ongoing, environment issue)

**Symptoms:** `package io/fs is not in std`, `package runtime is not in std`, `no such file or directory` for cache files, `can't create $WORK/...`.

**Root cause:** Nix-installed Go 1.26.0 on macOS. The Go build cache (`~/Library/Caches/go-build/`) gets corrupted during test runs, especially when running multiple Go commands concurrently. The stdlib itself can't be found after corruption.

**Workaround:** `rm -rf ~/Library/Caches/go-build` before each run, OR use `GOCACHE=$(mktemp -d)` for isolated runs. The latter is slower (fresh build every time) but reliable.

**Impact:** Every `go test` or `go build` command takes 30-90 seconds instead of 2-5 seconds. Race detector tests have never completed successfully in this environment. This is NOT a code issue — it's purely environmental.

**Permanent fix needed:** Either upgrade to Go 1.26.1+ (go.work requires it per other project errors), or switch Go installation method.

---

## e) WHAT WE SHOULD IMPROVE (Architectural Findings)

### Critical Design Issues

1. **Interfaces return `string` instead of typed IDs** — `Root.ID()`, `Event.AggregateID()`, `Event.ID()`, `Command.AggregateID()` all return `string`, defeating the branded type system. Forces lossy `string→ID→string` roundtrips everywhere (e.g., `id.MustParseAggregateID(u.ID())` in example code). This is the single biggest type safety gap.

2. **`Command.AggregateID()` forced on all commands** — Create commands don't have an aggregate yet. Cross-aggregate commands touch multiple aggregates. System-level commands have no aggregate. The `query.Query` interface correctly only requires `Type()` — command should follow the same pattern.

3. **`xtypes.TypedAggregate` duplicates state** — Stores `aggregateID` and `aggregateType` redundantly alongside embedded `core *aggregate.Core`. The core already holds both values. Should delegate to core instead of duplicating.

4. **`xtypes.TypedCommand.AggregateID()` returns `id.AggregateID`** but `command.Command.AggregateID()` returns `string` — `TypedCommand` does NOT implement `command.Command` interface. Type system is broken across the boundary.

5. **Duplicated validation** in `xtypes.EventBuilder.Build()` and `event.NewEvent()` — Both validate aggregateID, aggregateType, version. Should validate once.

### Moderate Design Issues

6. **`query.Handler` uses type alias (`=`)** — `type Handler = func(Query) (any, error)` means you can't add methods or use it as a named type. Command handler uses type definition (`type Handler func(...)`), which is better.

7. **Mixed `fmt.Errorf` / `cockroachdb/errors`** — Some errors have stack traces (cockroachdb), some don't (fmt). Should standardize on cockroachdb/errors for all error creation.

8. **`Lifecycle` duplicates `LifecycleMixin`** — Same three methods redeclared with identical implementations. Should just embed without redeclaring.

9. **`internal/dispatcher.Register` uses wrong error** — Uses `ErrHandlerNotFound` when dispatcher is closed. Semantically misleading.

### Nice-to-Have Improvements

10. **No `aggregate.Repository` interface** — Users must implement the load/save/publish pattern manually. The example shows the pattern; should be in the library.

11. **No generic typed dispatcher** — `DispatchTyped[T]` exists for queries but not for commands. Commands always return `error`, but a typed variant for type-safe payload extraction would be useful.

12. **Missing middleware utilities** — Logging, recovery, tracing, metrics are common cross-cutting concerns that every CQRS app needs.

---

## f) Top 25 Things to Do Next (sorted by impact/work ratio)

| #   | Task                                                   | Impact | Work             | Ratio |
| --- | ------------------------------------------------------ | ------ | ---------------- | ----- |
| 1   | Commit `RecordEvent` rename                            | High   | Zero             | ∞     |
| 2   | Remove duplicated `Lifecycle` methods                  | Medium | Tiny             | ★★★★★ |
| 3   | Fix `Register()` error semantics (ErrDispatcherClosed) | Medium | Tiny             | ★★★★★ |
| 4   | Standardize errors: `event/event.go`                   | Medium | Small            | ★★★★  |
| 5   | Standardize errors: `event/types.go`                   | Medium | Small            | ★★★★  |
| 6   | Standardize errors: `xtypes/event.go`                  | Medium | Small            | ★★★★  |
| 7   | Fix `query.Handler` to accept `context.Context`        | High   | Small            | ★★★★  |
| 8   | Fix `query.Handler` type alias → type definition       | High   | Tiny             | ★★★★  |
| 9   | Add `aggregate.Repository` interface                   | High   | Medium           | ★★★   |
| 10  | Remove duplicated state in `TypedAggregate`            | Medium | Small            | ★★★   |
| 11  | Fix `TypedCommand` to implement `command.Command`      | Medium | Small            | ★★★   |
| 12  | Deduplicate validation in `EventBuilder.Build()`       | Low    | Small            | ★★    |
| 13  | Add integration test (full CQRS roundtrip)             | High   | Medium           | ★★    |
| 14  | Add benchmarks (ID, dispatcher, store)                 | Medium | Medium           | ★★    |
| 15  | Add `middleware/logging.go`                            | Medium | Small            | ★★    |
| 16  | Add `middleware/recovery.go`                           | Medium | Small            | ★★    |
| 17  | Make `Command.AggregateID()` optional                  | High   | Large            | ★     |
| 18  | Make `Root.ID()` return `id.AggregateID`               | High   | Large (breaking) | ★     |
| 19  | Make `Event.AggregateID()` return typed ID             | High   | Large (breaking) | ★     |
| 20  | Add `middleware/tracing.go`                            | Low    | Medium           | ★     |
| 21  | Add `middleware/metrics.go`                            | Low    | Medium           | ★     |
| 22  | Add typed command dispatcher helper                    | Low    | Small            | ★     |
| 23  | Improve `catalog/yaml` coverage (79.8% → 90%+)         | Low    | Small            | ★     |
| 24  | Improve `catalog/eventcatalog` coverage (86.4% → 90%+) | Low    | Small            | ★     |
| 25  | Add Go doc examples (`Example*` test functions)        | Low    | Medium           | ★     |

---

## g) Top #1 Question I Cannot Figure Out Myself

**Should we make the breaking API changes now (items #17-19)?**

Making `Root.ID()`, `Event.AggregateID()`, and `Command.AggregateID()` return typed IDs instead of `string` would be a **breaking change** for every consumer. The library is at v0.x so breaking changes are acceptable per semver, but:

- The `Event` interface is used everywhere internally (Store, Bus, aggregates, tests)
- Changing `AggregateID()` from `string` to `id.AggregateID` cascades through every file
- The `command.Command.AggregateID()` forced-on-all-commands issue suggests we should make it optional at the same time

**My recommendation:** Do it now while the library is young and has few consumers. The branded type system is one of the library's key selling points — having the core interfaces return `string` undermines it completely. But I need your go-ahead because this is a large coordinated change.

---

## Environment

- **Go:** 1.26.0 (Nix), cache corruption issue
- **Build workaround:** `GOCACHE=$(mktemp -d)` for reliable builds
- **Test command:** `GOWORK=off go test ./... -count=1 -timeout 120s`
- **Race command:** `GOWORK=off go test ./... -count=1 -timeout 120s -race` (not yet verified)
- **Project stats:** 39 prod files (3,222 lines), 17 test files (4,663 lines), 12 packages
