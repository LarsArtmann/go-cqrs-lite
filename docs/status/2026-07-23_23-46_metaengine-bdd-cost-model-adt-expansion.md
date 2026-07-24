# Session Status: Meta-Engine BDD Suite, Cost Model, and ADT Expansion

**Date:** 2026-07-23 23:46
**Session focus:** Loaded BDD skill, wrote comprehensive Ginkgo spec suite, implemented formal cost model, cursor serialization, write amplification budget, and Multimap/Log ADTs. Discovered and fixed 3 bugs along the way.

---

## Session Journey (Chronological)

1. **Loaded skills + read all code** — Loaded `bdd-testing`, `how-to-golang`, `data-model-review` skills. Read all 11 existing `.go` files (types, fold, query, planner, store, engine, reflect, encoded, 2 test files, README). Read full design doc and prior status report (242 lines). Read BDD syntax reference.

2. **P0 Code Quality Cleanup** — Removed `FieldPath` type, removed vestigial `queryMeta` methods (`QueryFilters`, `QuerySortField`), cleaned `QueryAssignment` struct, deduplicated `detectPagination` (removed 20-line duplicate block), removed dead code (`isTimestampType`, `detectSortField`, `getFieldValue`), unexported `MemoryEngine` → `memoryEngine`, changed `NewMemoryEngine()` to return `Engine` interface. Removed unused `"strings"` import from `query.go`.

3. **Added Ginkgo/Gomega deps** — `go get github.com/onsi/ginkgo/v2 github.com/onsi/gomega@latest`. Added 7 transitive deps to go.mod.

4. **Wrote 7-file BDD spec suite** — `metaengine_suite_test.go` (bootstrap), `fixtures_test.go` (task management domain for all 7 ADTs), `on_test.go` (9 On classification specs), `planner_test.go` (15 plan/diagnostic/profile specs), `execution_test.go` (22 apply+execute+error specs), `pagination_test.go` (8 pagination+stability specs), `cost_test.go` (18 cost model+budget specs), `cursor_test.go` (9 cursor serialization specs), `adt_test.go` (9 Multimap+Log specs).

5. **Implemented formal cost model** (`cost.go`) — `CostEstimate` with `Complexity`, `Volume`, `EstimatedOps`, `EstimatedLatencyMs`. `estimateCost()` computes ops from complexity × volume with nsPerOp baseline. `ScaleThreshold` tables per ADT. `effectiveReadComplexity()` adjusts O(1) hash maps to O(N) for filtered scans. Planner now ranks engines by estimated latency (not just complexity rank).

6. **Integrated cost model into planner** — `planQuery()` now uses `estimateCost()` per engine, sorts by estimated latency, populates `CostEstimate` on `QueryAssignment`, checks latency budget, checks scale thresholds, emits diagnostics for both.

7. **Implemented write amplification budget** — `WithWriteAmplificationBudget(n)` plan option. `checkWriteAmplification` now accepts a budget parameter. Default budget constant `DefaultWriteAmplificationBudget = 3`.

8. **Implemented cursor serialization** (`cursor.go`) — `Cursor.String()` → URL-safe base64 (JSON wrapped). `ParseCursor(s)` → round-trip safe. Empty string = nil cursor.

9. **Fixed 3 bugs during cursor integration:**
   - **Bug #1:** `buildSortFunc` panicked when comparing result items against a deserialized cursor value (different Go types). Fixed with type check: if the value matches the closure's parameter type, call the closure; otherwise use the raw value directly.
   - **Bug #2:** `compareValue` couldn't compare `int` (from result item sort key) vs `float64` (from JSON-deserialized cursor). Added `tryNumericCompare()` + `toFloat64()` fallback that converts all numeric types to float64 for cross-type comparison.
   - **Bug #3:** `reconstructCollection` stored the full result item as cursor value instead of the sort key. Fixed by threading `sortKeyFn` through `ExecuteTyped` → `reconstructCollection`.

10. **Implemented Multimap and Log ADTs** — Added `MultiEntry{Key, Value}` and `Append{Value}` sentinels. New `FoldMultiInsert` and `FoldAppend` fold kinds. `MultimapBackend` interface (MultiAdd/MultiGet). `LogBackend` interface (LogAppend/LogTail). `memoryEngine` implements both. Updated ADT classification, read pattern inference, cost thresholds. Added 7 BDD specs covering both ADTs.

11. **Final verification** — Build clean, vet clean, 89 BDD specs + 11 original tests pass with `-race` in 1.035s. Coverage: 81.8%.

---

## a) FULLY DONE

| #   | What                                      | Files                                          | Evidence                                                                     |
| --- | ----------------------------------------- | ---------------------------------------------- | ---------------------------------------------------------------------------- |
| 1   | **Build compiles + vets clean**           | All `.go` files                                | `go build ./...` + `go vet ./...` pass                                       |
| 2   | **89 BDD specs pass with -race**          | 7 spec files                                   | 0.016s, 0 race conditions                                                    |
| 3   | **All 11 original tests preserved**       | `metaengine_test.go`, `correctness_test.go`    | All pass, no regressions                                                     |
| 4   | **MemoryEngine unexported**               | `engine.go`                                    | `memoryEngine` struct, `NewMemoryEngine()` returns `Engine` interface        |
| 5   | **FieldPath type removed**                | `types.go`                                     | Vestigial type deleted                                                       |
| 6   | **Vestigial queryMeta methods removed**   | `query.go`, `planner.go`                       | `QueryFilters`, `QuerySortField` deleted; `QueryAssignment.Filters` deleted  |
| 7   | **detectPagination deduplicated**         | `reflect.go`                                   | Single-pass field iteration, no duplicate logic                              |
| 8   | **Dead code removed**                     | `reflect.go`                                   | `isTimestampType`, `detectSortField`, `getFieldValue` deleted                |
| 9   | **Formal CostEstimate**                   | `cost.go`                                      | Ops + latency from complexity × volume, `WithinBudget()`, `String()`         |
| 10  | **Volume-based cost estimation**          | `cost.go`, `planner.go`                        | `Volume(n)` populates cost estimate, `estimateCost()` uses it                |
| 11  | **LatencyBudget enforcement**             | `planner.go`                                   | `WithLatencyBudget(ms)` → WARN diagnostic when exceeded                      |
| 12  | **Scale threshold tables**                | `cost.go`                                      | Per-ADT optimal ranges, `checkScaleThreshold()` emits warnings               |
| 13  | **effectiveReadComplexity**               | `cost.go`                                      | Hash map scan = O(N), not O(1)                                               |
| 14  | **Write amplification budget**            | `planner.go`                                   | `WithWriteAmplificationBudget(n)`, `DefaultWriteAmplificationBudget = 3`     |
| 15  | **Cursor serialization**                  | `cursor.go`                                    | `String()` → base64, `ParseCursor()` round-trip safe                         |
| 16  | **Cross-type numeric comparison**         | `engine.go`                                    | `tryNumericCompare()`, `toFloat64()` for int vs float64 from JSON cursors    |
| 17  | **Sort key in cursor**                    | `store.go`, `reflect.go`                       | `sortKeyFn` threaded through `ExecuteTyped` → `reconstructCollection`        |
| 18  | **Multimap ADT**                          | `types.go`, `fold.go`, `engine.go`, `store.go` | `MultiEntry`, `FoldMultiInsert`, `MultimapBackend`, MultiAdd/MultiGet        |
| 19  | **Log ADT**                               | `types.go`, `fold.go`, `engine.go`, `store.go` | `Append`, `FoldAppend`, `LogBackend`, LogAppend/LogTail                      |
| 20  | **ADT classification for 7 types**        | `fold.go`                                      | Map, Set, Counter, Graph, SortedMap, Multimap, Log                           |
| 21  | **Read pattern inference for 7 patterns** | `query.go`                                     | Point, Membership, FilteredScan, Aggregate, Traversal, MultiLookup, LogTail  |
| 22  | **BDD: On classification (9 specs)**      | `on_test.go`                                   | 7 handler shapes + panic tests + EventTypeName                               |
| 23  | **BDD: Planner (15 specs)**               | `planner_test.go`                              | Plan, errors, ADT inference, pagination detection, report string, profiles   |
| 24  | **BDD: Execution (22 specs)**             | `execution_test.go`                            | All 7 ADTs end-to-end, concurrent FoldUpdate, ApplyEncoded, error handling   |
| 25  | **BDD: Pagination (8 specs)**             | `pagination_test.go`                           | First page, last page, multi-page traversal, sort stability                  |
| 26  | **BDD: Cost model (18 specs)**            | `cost_test.go`                                 | WithinBudget, cost estimation, latency warnings, scale thresholds, WA budget |
| 27  | **BDD: Cursor serialization (9 specs)**   | `cursor_test.go`                               | String, ParseCursor, round-trip, HTTP boundary pagination                    |
| 28  | **BDD: Multimap + Log (9 specs)**         | `adt_test.go`                                  | MultiEntry classification, multimap reads, log reads, ADT classification     |
| 29  | **README updated**                        | `README.md`                                    | 7 ADTs, cursor serialization, cost model, write amplification budget         |
| 30  | **nix fmt applied**                       | All files                                      | All files formatted                                                          |
| 31  | **Coverage: 81.8%**                       | Module                                         | `go test -cover`                                                             |

## b) PARTIALLY DONE

| #   | What                                    | Current State                                                                                   | What's Missing                                                                                                      |
| --- | --------------------------------------- | ----------------------------------------------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------- |
| 1   | **Cost model**                          | `estimateCost` uses nsPerOp baseline constant. Volume drives latency. Scale thresholds per ADT. | No empirical calibration. nsPerOp=100 is arbitrary. No benchmark suite to validate estimates. No ILP formulation.   |
| 2   | **Scale-dependent structure selection** | Scale threshold tables emit WARN diagnostics when volume exceeds optimal range.                 | Does NOT actually switch structures (e.g., hash → B-tree) based on volume. Just warns. No automatic downgrading.    |
| 3   | **Write amplification**                 | Budget enforcement + diagnostic warnings.                                                       | No auto-denormalization detection. No cross-projection optimization suggestions.                                    |
| 4   | **Engine profiles**                     | Memory + SQLite profiles defined. SQLite is a stub profile only (no implementation).            | No real SQLite engine. No Pebble engine. No plugin registration pattern.                                            |
| 5   | **Cursor serialization**                | JSON+base64 encoding works. Sort keys survive round-trip.                                       | No versioning (no migration path if cursor format changes). No encryption for opaque cursors.                       |
| 6   | **Multimap ADT**                        | MultiAdd/MultiGet implemented. Classified correctly.                                            | No MultiRemove (delete a value from a key). No MultiCount. No range queries within a key.                           |
| 7   | **Log ADT**                             | LogAppend/LogTail implemented. Ordered.                                                         | No LogTruncate (cap log length). No LogRange (read between two positions). No LogSize.                              |
| 8   | **BDD test suite**                      | 89 specs across 7 files. All public API covered.                                                | No property-based testing. No fuzzing. No concurrent execution specs for multimap/log. Coverage gaps in edge cases. |
| 9   | **Comparison engine**                   | `compareValue` handles 12 numeric types + strings + time.Time + cross-type numeric fallback.    | No comparison for nested structs, slices, or maps. Fallback is `fmt.Sprintf` string comparison (fragile).           |

## c) NOT STARTED

| #   | What                                                                                | From                              |
| --- | ----------------------------------------------------------------------------------- | --------------------------------- |
| 1   | **Engine plugin registration** (`Register()`/`Open(cfg)` database/sql pattern)      | Project definition Phase 4        |
| 2   | **Real SQLite engine** (wrapping `storage/view/SQLViewStore`)                       | Project definition Phase 4        |
| 3   | **Real Pebble engine** (wrapping `storage/pebble/`)                                 | Project definition Phase 4        |
| 4   | **SQL pushdown for FilterOn closures**                                              | Open Decision Q2                  |
| 5   | **DDL generation** (CREATE TABLE, CREATE INDEX from fold types)                     | Project definition Phase 2        |
| 6   | **Generated typed read API** (`store.Users.Get(id)` instead of `ExecuteTyped[Q,R]`) | Project definition Phase 3        |
| 7   | **Streaming** (`iter.Seq2[T, error]` for unbounded results)                         | Project definition Phase 3        |
| 8   | **Query expression tree** (`query.Or`, `query.Eq`, etc.) — only AND via FilterOn    | Project definition Problem 6      |
| 9   | **Hot-reload** (re-plannable planner, background replay, atomic cutover)            | event-query-model.md Section 14   |
| 10  | **Auto-denormalization** (cross-projection query detection)                         | Project definition Phase 2        |
| 11  | **Metadata parameter in folds** (`func(e, md Metadata)`)                            | event-query-model.md Section 8    |
| 12  | **Integration with `event.Event`**                                                  | Blocked by go.sum checksum issues |
| 13  | **Integration with `projection.Projection` / `projectionhost.Host`**                | Adapter documented but not wired  |
| 14  | **Wire into `stack.Bundle`**                                                        | Not started                       |
| 15  | **Update AGENTS.md module list** with metaengine                                    | Not done this session             |
| 16  | **Property-based testing** with `rapid`                                             | Not started                       |
| 17  | **Benchmark suite** (planner-chosen vs hand-tuned)                                  | Not started                       |
| 18  | **D2/Mermaid plan visualizer**                                                      | Not started                       |
| 19  | **Formal model paper** (ILP formulation, tractability argument)                     | Research phase                    |
| 20  | **MultiRemove / MultiCount for Multimap**                                           | Not started                       |
| 21  | **LogTruncate / LogRange / LogSize for Log ADT**                                    | Not started                       |
| 22  | **Metadata propagation in cursor** (tenant, auth context in cursor)                 | Not started                       |
| 23  | **Update `go.work` to include metaengine module**                                   | Not verified                      |

## d) TOTALLY FUCKED UP

| #   | What                                                 | Severity | Details                                                                                                                                                                                                                                                                                                            |
| --- | ---------------------------------------------------- | -------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| 1   | **`SatisfyAny` + `MatchError` hack in cost_test.go** | MEDIUM   | First attempt at cost_test.go used `ContainElement(SatisfyAny(MatchError("not a match"), WithTransform(...)))` — a hack to work around Gomega matcher type constraints. Rewrote with a simple `hasDiagnostic()` helper function. The intermediate commit was committed by an auto-hook with a meaningless message. |
| 2   | **`DescribeTable` parameter type mismatch**          | LOW      | First cost_test.go used `DescribeTable` with `(float64, int64, bool)` entries but Ginkgo's table DSL requires all entries to match the function signature exactly. The `int64` budget parameter was the issue. Rewrote as individual `It` blocks.                                                                  |
| 3   | **First attempt used `time` import unused**          | LOW      | `on_test.go` had `import "time"` and `var _ time.Duration` as a "compile-time check" — completely unnecessary. Removed.                                                                                                                                                                                            |
| 4   | **Auto-commit hooks fired during edits**             | MEDIUM   | Commits `971f4b2c`, `4d77630a`, `70eac417`, `4654b589`, `c0c3ccf5` were auto-committed with generic messages like "refactor(metaengine): restructure query planning" while mid-edit. These make the git history noisy. The README.md is currently uncommitted (modified after last auto-commit).                   |
| 5   | **Three bugs found and fixed mid-implementation**    | HIGH     | Bug #1 (sort closure panic on cursor values), Bug #2 (cross-type numeric comparison), Bug #3 (cursor stored full item not sort key). These were caught by the BDD specs — proving the TDD approach works. But they indicate the cursor pagination path was underdesigned in the prior session.                     |
| 6   | **File size violations**                             | LOW      | 6 files exceed the 350-line CI limit: `engine.go` (597), `store.go` (457), `fold.go` (425), `metaengine_test.go` (404), `reflect.go` (360), `planner.go` (352). The module is not yet wired into CI, but when it is, these will fail.                                                                              |

## e) WHAT WE SHOULD IMPROVE

### Architecture

1. **Split `engine.go` (597 lines)** — It holds 7 backend interfaces + the entire `memoryEngine` implementation + `compareValue` + `tryNumericCompare` + `toFloat64`. Split into: `engine.go` (interfaces), `memory_engine.go` (implementation), `compare.go` (comparison helpers).

2. **Split `store.go` (457 lines)** — It holds `Store`, `queryRuntime`, `Apply`, `applyFold`, `Execute`, `executeQuery`, `executeFilteredScan`, `buildFilterPredicates`, `buildSortFunc`, `ExecuteTyped`, `sortKeyFn`. Split into: `store.go` (Store + Apply), `execute.go` (Execute + executeQuery + ExecuteTyped), `filter.go` (filter predicates + sort func).

3. **Cursor format versioning** — The current `Cursor.String()` produces unversioned base64. If the format changes, old cursors break. Add a version byte prefix: `v1:<base64>`.

4. **`Plan` variadic signature is weakly typed** — `Plan(engines, args ...any)` accepts `any` for both queries and plan options. A consumer could pass `"hello"` and get a runtime error. Consider a `PlanOption` type or a builder pattern.

5. **Graph key extraction fallback is fragile** — `extractFirstDomainField` grabs the first non-meta field from the input struct. This works for `FriendsOf{ID UserID, Depth int}` but breaks on structs with multiple domain fields.

6. **`compareValue` fallback is string comparison** — When types don't match and aren't numeric, it falls back to `fmt.Sprintf`. This is nondeterministic across locales for some types.

### Testing

7. **No property-based testing** — The cost model, cursor round-trip, and pagination logic are perfect candidates for property-based testing with `pgregory.net/rapid`. "For any volume V and complexity C, WithinBudget(estimateCost(C,V)) is deterministic."

8. **No concurrent multimap/log specs** — We tested concurrent FoldUpdate (100 goroutines) but not concurrent MultiAdd or LogAppend.

9. **Coverage gaps in edge cases** — 81.8% means ~18% of code paths are untested. Likely: error branches in `applyFold` (unsupported engine backend), cursor ParseCursor invalid JSON path, `executeQuery` default case.

10. **No integration tests with the real event-sourcing pipeline** — ApplyEncoded works but isn't tested with actual `event.Event` types (blocked by go.sum).

### Planner

11. **Cost model is uncalibrated** — `nsPerOp = 100.0` is an arbitrary constant. Real in-memory hash lookups are ~10ns, SQL B-tree lookups are ~100µs. The estimates are off by 3 orders of magnitude.

12. **No actual structure switching** — Scale thresholds warn but don't act. At volume > 10M, the planner should prefer SQLite over memory for Map ADTs.

13. **Engine ranking is single-dimensional** — Only estimated latency. No memory footprint, no write cost, no startup time.

### Process

14. **README is uncommitted** — The README was modified after the last auto-commit. It needs to be committed.

15. **AGENTS.md module list not updated** — `metaengine` is missing from the monorepo module list in `AGENTS.md`.

16. **`go.work` may not include the metaengine module** — Need to verify `go.work` includes `metaengine/`.

17. **Auto-commit hooks are actively harmful** — They committed 5 intermediate states with meaningless messages. The git history is unreadable. These hooks should be disabled or configured to only fire on explicit saves, not on every file write.

---

## f) Up to 50 Things to Do Next

### P0 — Immediate (must do before next feature)

| #   | Task                                                                    | Effort |
| --- | ----------------------------------------------------------------------- | ------ |
| 1   | Commit the uncommitted README.md                                        | 2min   |
| 2   | Split `engine.go` (597 lines) into 3 files under 350 lines              | 30min  |
| 3   | Split `store.go` (457 lines) into 2-3 files under 350 lines             | 30min  |
| 4   | Split `fold.go` (425 lines) — extract call helpers + ADT classification | 20min  |
| 5   | Split `reflect.go` (360 lines) — extract collection reconstruction      | 20min  |
| 6   | Split `planner.go` (352 lines) — extract diagnostics                    | 15min  |
| 7   | Add metaengine to `go.work` if missing                                  | 5min   |
| 8   | Update AGENTS.md module list with metaengine                            | 5min   |

### P1 — Cost Model Calibration

| #   | Task                                                                           | Effort |
| --- | ------------------------------------------------------------------------------ | ------ |
| 9   | Write benchmarks for memory engine operations (MapGet, SetContains, etc.)      | 1h     |
| 10  | Calibrate `nsPerOp` constant from benchmark results                            | 30min  |
| 11  | Add memory footprint estimation to CostEstimate                                | 1h     |
| 12  | Add write cost estimation (not just read cost)                                 | 1h     |
| 13  | Add `CostEstimate.WithinMemoryBudget(bytes int64)` method                      | 30min  |
| 14  | Property-based test: for all volumes, cost estimate is monotonic in complexity | 45min  |

### P2 — Engine Design

| #   | Task                                                                 | Effort |
| --- | -------------------------------------------------------------------- | ------ |
| 15  | Design engine plugin registration pattern (`Register()`/`Open(cfg)`) | 1h     |
| 16  | Design how SQL engine receives filter/sort natively (Q2)             | 1h+    |
| 17  | Implement real SQLite engine wrapping `storage/view/SQLViewStore`    | 2h+    |
| 18  | Implement real Pebble engine wrapping `storage/pebble/`              | 1h+    |
| 19  | DDL generation from fold types (CREATE TABLE, CREATE INDEX)          | 2h     |
| 20  | Multi-engine plan test (memory + SQLite profile side by side)        | 30min  |

### P3 — Read API

| #   | Task                                                              | Effort |
| --- | ----------------------------------------------------------------- | ------ |
| 21  | Implement streaming (`iter.Seq2[T, error]`) for unbounded results | 30min  |
| 22  | Implement query expression tree (`query.Or`, `query.Eq`, etc.)    | 2h     |
| 23  | Design generated typed read API (`store.Users.Get(id)`)           | 1h     |
| 24  | Cursor versioning (prefix byte for migration safety)              | 20min  |

### P4 — ADT Enhancements

| #   | Task                                                        | Effort |
| --- | ----------------------------------------------------------- | ------ |
| 25  | Implement `MultiRemove` (delete value from key in multimap) | 30min  |
| 26  | Implement `MultiCount` (count values per key)               | 20min  |
| 27  | Implement `LogTruncate` (cap log length)                    | 20min  |
| 28  | Implement `LogRange` (read between two positions)           | 30min  |
| 29  | Implement `LogSize` (total entries)                         | 15min  |

### P5 — Integration

| #   | Task                                                       | Effort |
| --- | ---------------------------------------------------------- | ------ |
| 30  | Resolve go.sum checksum issues for `event/` import         | 30min  |
| 31  | Add `Apply(event.Event)` method                            | 10min  |
| 32  | Add `projection.Projection` implementation                 | 15min  |
| 33  | Wire into `projectionhost.Host`                            | 30min  |
| 34  | Wire into `stack.Bundle`                                   | 30min  |
| 35  | Add metaengine to CI pipeline (`.github/workflows/ci.yml`) | 15min  |

### P6 — Testing

| #   | Task                                                                                  | Effort |
| --- | ------------------------------------------------------------------------------------- | ------ |
| 36  | Property-based testing with `rapid` (cursor round-trip, cost monotonicity)            | 1h     |
| 37  | Concurrent MultiAdd + LogAppend specs                                                 | 30min  |
| 38  | Error branch coverage (unsupported engine backend, invalid cursor, unknown fold kind) | 45min  |
| 39  | Benchmark suite (planner-chosen vs hand-tuned)                                        | 2h     |
| 40  | Scale threshold empirical validation                                                  | 3h     |

### P7 — Documentation

| #   | Task                                                                  | Effort |
| --- | --------------------------------------------------------------------- | ------ |
| 41  | Document engine interface contract (what each backend must implement) | 15min  |
| 42  | Document three-role model in a separate design doc                    | 30min  |
| 43  | Consolidate overlapping planning docs into narrative                  | 1h     |
| 44  | D2/Mermaid plan visualizer                                            | 30min  |
| 45  | Write case study (taskmanager on memory-only vs multi-engine)         | 2h     |

### P8 — Advanced

| #   | Task                                                         | Effort |
| --- | ------------------------------------------------------------ | ------ |
| 46  | Hot-reload (re-plannable planner)                            | 2h+    |
| 47  | Background replay + atomic cutover                           | 2h+    |
| 48  | Metadata parameter in fold handlers (`func(e, md Metadata)`) | 45min  |
| 49  | Auto-denormalization detection (cross-projection patterns)   | 1h     |
| 50  | Formal model paper (ILP formulation, tractability argument)  | 1 week |

---

## g) Questions I CANNOT Answer Myself

**Q1: How should FilterOn typed closures translate to SQL pushdown?**
`FilterOn(func(r FindUserResult) string { return r.Status })` works at runtime for in-memory engines (call closure on each item). But for SQL: we need `WHERE status = ?`. Go reflection cannot extract "Status" from a closure body. Options: (a) accept memory uses closures, SQL uses a different declaration mechanism, (b) code generation to extract field paths at build time, (c) the `Field()` named-descriptor approach from the design doc's Decision 1, (d) something else. This blocks building a real SQL engine.

**Q2: Should the planner output executable handlers wired into projectionhost.Host, or keep the current Store.Apply() + ExecuteTyped[Q,R] shape?**
Currently `Plan()` returns a `*Store` that holds runtime state AND diagnostic `PlanResult`. The project definition envisions the planner generating projection handlers (event → engine writes) and typed read API (`store.Users.Get(id)`). Is the current `Store.Apply()` + `ExecuteTyped[Q,R]` the right shape, or should the planner output code/handlers that get wired into `projectionhost.Host`? This affects the entire integration story.

**Q3: Should the metaengine module stay zero-dependency (stdlib only) or is it acceptable to depend on `event/` from the monorepo?**
Currently the module has zero production deps and one test dep (ginkgo/gomega). Integrating with `event.Event` and `projection.Projection` requires importing `event/`. But `event/` has a go.sum checksum issue that blocks the import. Should we fix the checksum issue and add the dependency, or keep the module standalone and use adapter types?
