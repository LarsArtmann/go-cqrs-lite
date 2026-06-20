# ULID Migration — Comprehensive Status Report

**Date:** 2026-04-25 09:52  
**Author:** Crush (AI Assistant)  
**Session:** Resume from interrupted session

---

## A) FULLY DONE ✅

### 1. Core ID System Rewrite (`core/pkg/id/id.go`)

- **`type Of[T any] = cbid.ID[T, string]`** — type alias to `go-composable-business-types/id.ID[T, string]`
- ULID generation via `oklog/ulid/v2` with `rand.Reader` (thread-safe, no shared entropy)
- Thin wrappers preserved: `New[T]()`, `NewWithPrefix[T]()`, `Parse[T]()`, `MustParse[T]()`, `ULID()` helper
- Removed 194 lines of custom UUID code, replaced with 28 lines delegating to cbid
- All encoding methods (JSON, SQL, Binary, Text, Gob) inherited from cbid — zero custom encoding code

### 2. All Production Code Updated

| File                                    | Change                                                        |
| --------------------------------------- | ------------------------------------------------------------- |
| `core/pkg/id/id.go`                     | Complete rewrite (28 additions, 194 deletions)                |
| `core/event/event.go`                   | `.IsEmpty()` → `.IsZero()`, removed `""` metadata assignments |
| `core/event/internal/evtest/helpers.go` | Replaced `google/uuid` with `id.New[struct{}]()`              |
| `xtypes/event.go`                       | `.IsEmpty()` → `.IsZero()`                                    |

### 3. All Test Files Updated (9 files)

| File                                    | Changes                                              |
| --------------------------------------- | ---------------------------------------------------- |
| `core/pkg/id/id_test.go`                | Complete rewrite for new API (183 lines changed)     |
| `core/pkg/id/fuzz_test.go`              | Seed corpus updated to ULID format                   |
| `core/aggregate/aggregate_test.go`      | 9 edits wrapping string literals with `MustParse*()` |
| `core/command/command_test.go`          | String literals wrapped with `MustParse*()`          |
| `core/event/event_test.go`              | 6 edits: zero values, casts, comparisons             |
| `core/event/event_sourcing_bdd_test.go` | CorrelationID/CausationID/UserID/RequestID wrapping  |
| `xtypes/xtypes_test.go`                 | `.IsEmpty()` → `.IsZero()`                           |
| `memory/bus_test.go`                    | Full rewrite, all string literals wrapped            |
| `memory/store_test.go`                  | Full rewrite, extracted `aggID` variables for reuse  |

### 4. All Module Dependencies Updated

| Module               | Changes                                                               |
| -------------------- | --------------------------------------------------------------------- |
| `core/go.mod`        | Added `go-composable-business-types`, `oklog/ulid/v2`, local replace  |
| `xtypes/go.mod`      | Added replace for `go-composable-business-types`                      |
| `memory/go.mod`      | Added replace for `go-composable-business-types` (converted to block) |
| `catalog/go.mod`     | Added replace for `go-composable-business-types`                      |
| `middleware/go.mod`  | Added replace for `go-composable-business-types`                      |
| All 5 `go.sum` files | Updated checksums                                                     |

### 5. All Tests Passing — Verified Fresh

| Module              | Packages | Coverage | Status  |
| ------------------- | -------- | -------- | ------- |
| core/aggregate      | 1        | 78.3%    | ✅ PASS |
| core/command        | 1        | 84.4%    | ✅ PASS |
| core/event          | 1        | 89.0%    | ✅ PASS |
| core/pkg/dispatcher | 1        | 77.4%    | ✅ PASS |
| core/pkg/id         | 1        | 63.6%    | ✅ PASS |
| core/query          | 1        | 91.5%    | ✅ PASS |
| xtypes              | 1        | —        | ✅ PASS |
| memory              | 1        | —        | ✅ PASS |
| catalog             | 4        | —        | ✅ PASS |
| middleware          | 1        | —        | ✅ PASS |

### 6. All Builds Passing

All 5 modules compile cleanly: `go build ./...` succeeds everywhere.

---

## B) PARTIALLY DONE ⚠️

### 1. ID Test Coverage: 63.6% (target: 80%+)

The `core/pkg/id` package has the lowest coverage in the project. The new thin wrapper over cbid means:

- cbid's own methods are tested upstream, but our wrappers (`New`, `Parse`, `MustParse`, `NewWithPrefix`, `ULID`) need more coverage
- Missing: error paths in `Parse`, edge cases in `MustParse`, prefix formatting

### 2. Race Detector Not Run

All tests pass but `-race` flag has not been verified across all modules. The previous session discovered that `ulid.MonotonicEntropy` is NOT thread-safe (caused `slice bounds out of range` panics). We switched to `rand.Reader` but haven't verified with `-race`.

### 3. Lint Not Run

`golangci-lint run` has not been executed post-migration. There may be new warnings.

---

## C) NOT STARTED ❌

### 1. Root-Level Package Migration

The root-level packages (`aggregate/`, `command/`, `event/`, `pkg/id/`) outside `core/` may still have stale UUID code. These appear to be either:

- Re-exports from `core/`
- Older duplicates
- Need investigation

### 2. Examples Module

`examples/` directory not checked or updated for the new ID API.

### 3. Publishing `go-composable-business-types`

Currently using local `replace` directives because `go-composable-business-types v0.0.0` isn't published. All downstream consumers would need the replace directive too. Publishing would eliminate this friction.

### 4. go.work Integration

`GOWORK=off` is needed for all commands because `/Users/larsartmann/go.work` exists but doesn't include go-cqrs-lite. Either:

- Add go-cqrs-lite to go.work, or
- Add a project-level go.work file

### 5. Changelog / Release Notes

No CHANGELOG.md entry or release tag for this breaking change.

### 6. Benchmark Comparison

No benchmarks run comparing old UUID vs new ULID performance. The old `core/pkg/id/benchmark_test.go` exists but hasn't been run against both versions.

---

## D) TOTALLY FUCKED UP 💥

### Nothing catastrophic.

The migration went remarkably smoothly for a breaking API change touching 24 files across 5 modules. The only close call was:

- **MonotonicEntropy thread-safety bug**: Shared `*ulid.MonotonicEntropy` panicked in parallel tests with `slice bounds out of range`. Fixed immediately by switching to `rand.Reader`. This was caught during development, not in production.

---

## E) WHAT WE SHOULD IMPROVE 🔧

### 1. Test Coverage for ID Package

63.6% is the lowest in the project. The new thin wrappers need proper test coverage including error paths.

### 2. Race Detector Discipline

Every significant change should be verified with `-race`. This should be a CI gate.

### 3. Root-Level Package Clarity

The relationship between `core/aggregate` and root-level `aggregate/` is unclear. This should be documented or consolidated.

### 4. Replace Directive Strategy

Every module that transitively depends on `go-composable-business-types` needs a `replace` directive. This is fragile. Publishing the dependency or using `go.work` would fix this.

### 5. Breaking Change Documentation

The migration changed the public API surface (`.IsEmpty()` → `.IsZero()`, string casts no longer work). A migration guide would help consumers.

### 6. Consistent Zero-Value Idioms

Tests mix `id.AggregateID{}` and `var x id.AggregateID` for zero values. Pick one and standardize.

---

## F) Top 25 Things to Do Next

| #   | Priority | Task                                                                | Est. Effort |
| --- | -------- | ------------------------------------------------------------------- | ----------- |
| 1   | 🔴 HIGH  | Run `go test -race ./...` across all modules                        | 5 min       |
| 2   | 🔴 HIGH  | Run `golangci-lint run` across all modules                          | 5 min       |
| 3   | 🔴 HIGH  | Increase `core/pkg/id` test coverage to 80%+                        | 30 min      |
| 4   | 🔴 HIGH  | Investigate root-level packages (`aggregate/`, `command/`, etc.)    | 15 min      |
| 5   | 🟡 MED   | Add `-race` to CI / test commands in AGENTS.md                      | 5 min       |
| 6   | 🟡 MED   | Run benchmarks: UUID v4 vs ULID performance comparison              | 15 min      |
| 7   | 🟡 MED   | Audit `examples/` for stale UUID code                               | 15 min      |
| 8   | 🟡 MED   | Write migration guide (BREAKING CHANGE docs for consumers)          | 30 min      |
| 9   | 🟡 MED   | Standardize zero-value idiom in tests (`id.X{}` vs `var`)           | 10 min      |
| 10  | 🟡 MED   | Add ULID validation to `Parse[T]()` (26 chars, Crockford Base32)    | 15 min      |
| 11  | 🟡 MED   | Publish `go-composable-business-types` to remove replace directives | 30 min      |
| 12  | 🟡 MED   | Add `go.work` or `go.work.use` at project root                      | 10 min      |
| 13  | 🟡 MED   | Update AGENTS.md with ULID/IsZero/MustParse patterns                | 10 min      |
| 14  | 🟢 LOW   | Add CHANGELOG.md entry for v0.x breaking change                     | 10 min      |
| 15  | 🟢 LOW   | Add `testutil` helpers for common `MustParse*()` patterns           | 15 min      |
| 16  | 🟢 LOW   | Add integration test spanning core → memory → xtypes                | 30 min      |
| 17  | 🟢 LOW   | Document `go-composable-business-types` API surface used            | 10 min      |
| 18  | 🟢 LOW   | Add example_test.go for `core/pkg/id` showing new patterns          | 15 min      |
| 19  | 🟢 LOW   | Verify SQL NULL behavior with actual database driver                | 30 min      |
| 20  | 🟢 LOW   | Add fuzz targets for `Parse()` and `MustParse()`                    | 15 min      |
| 21  | 🟢 LOW   | Consider `fmt.Stringer` benchmark (hot path in logging)             | 10 min      |
| 22  | 🟢 LOW   | Add `.IsZero()` check to `event.NewEvent()` for all ID fields       | 10 min      |
| 23  | 🟢 LOW   | Update catalog schema generation for struct-based IDs               | 20 min      |
| 24  | 🟢 LOW   | Clean up `docs/status/` — 26 files accumulated                      | 10 min      |
| 25  | 🟢 LOW   | Git tag for the ULID migration milestone                            | 2 min       |

---

## G) Top #1 Question I Cannot Figure Out Myself 🤔

**What is the intended relationship between root-level packages (`aggregate/`, `command/`, `event/`, `pkg/id/`) and their `core/` counterparts?**

- Are they re-exports? (If so, they need updating)
- Are they deprecated? (If so, they should be removed or marked)
- Are they independent implementations? (If so, they also need ULID migration)
- Are they part of a planned go.work multi-module setup?

This affects whether the migration is truly complete or has a significant gap.

---

## Summary Statistics

| Metric                   | Value                                         |
| ------------------------ | --------------------------------------------- |
| Files modified           | 24                                            |
| Lines added              | +445                                          |
| Lines removed            | -492                                          |
| Net change               | -47 lines (smaller codebase!)                 |
| Production files changed | 4                                             |
| Test files changed       | 9                                             |
| Config files changed     | 11                                            |
| Modules affected         | 5 (core, xtypes, memory, catalog, middleware) |
| Tests passing            | 12 packages across 5 modules                  |
| Build status             | ✅ ALL GREEN                                  |
