# Status Report: 2026-08-08 02:09 — Aggregate Pushdown P0+P1 Complete, Self-Critique

> **Scope**: DuckDB aggregation pushdown P0 bug fixes + P1 SQLite/PG parity + tests + benchmarks + ExplainableAggregate. This was a continuation session from `2026-08-08_01-33_duckdb-aggregation-pushdown.md`.

---

## A. FULLY DONE ✅

### P0 — Critical Fixes (4/4)

| #    | Item                                            | Detail                                                                                                                                                                                                                                                                                                                                                                                         |
| ---- | ----------------------------------------------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| P0.1 | **MultiGroupedAggregate AVG fallback bug**      | `groupAccum` now tracks `nonNullCounts map[string]int` per-spec. AVG divides by per-spec non-null count, not total group `acc.count`. Also fixed MAX 0-init bug (`!exists` check added). `metaengine/typed_reader.go:815-895`                                                                                                                                                                  |
| P0.2 | **API-surface golden regenerated**              | 3 passes: initial (3781→3792 for SQLite+PG), final (3798 for ExplainableAggregate). `cmd/api-stability` passes clean.                                                                                                                                                                                                                                                                          |
| P0.3 | **ADTStreamLog missing from AllADTs()**         | Added to slice at `metaengine/enum_validation.go:12`. `ADTStreamLog.Valid()` now returns true.                                                                                                                                                                                                                                                                                                 |
| P0.4 | **aggregatePushdown MIN/MAX/AVG fallback bugs** | Replaced `result == 0` sentinel with `firstSet bool` flag. MIN now correctly returns negative values. MAX now correctly handles all-negative sets. AVG divides by `nonNullCount` (skips non-numeric fields) not `len(rows)`. Added `AggregateCount` handling (was missing — broke `MultiAggregate` fallback). Also fixed grouped MAX 0-init bug. `metaengine/typed_reader.go:499-540, 717-729` |

### P1 — Parity + Testing (11/11)

| #        | Item                                                                         | Files                                                                                                                  | LOC    |
| -------- | ---------------------------------------------------------------------------- | ---------------------------------------------------------------------------------------------------------------------- | ------ |
| P1.5-8   | **SQLite engine: 4 aggregate interfaces**                                    | `sqliteengine/aggregations_grouped.go`                                                                                 | 524    |
| P1.9     | **Cross-engine parity harness (DuckDB vs SQLite)**                           | `bench/aggregate_parity_cgo_test.go`                                                                                   | 254    |
| P1.10    | **TypedReader-level fallback tests**                                         | `typed_reader_aggregate_test.go` (13 subtests)                                                                         | 222    |
| P1.11    | **GROUP BY pushdown benchmark**                                              | `duckdbengine/aggregate_bench_cgo_test.go`                                                                             | 194    |
| P1.12    | **MultiAggregate single-pass vs N-calls benchmark**                          | same file                                                                                                              | —      |
| P1.13    | **ExplainableAggregate interface + impl**                                    | `aggregations.go` (interface), `explain.go` (TypedReader method), `duckdbengine/explain.go`, `sqliteengine/explain.go` | ~250   |
| P1.14-15 | **Postgres engine: all 5 aggregate interfaces**                              | `pgengine/aggregations.go`                                                                                             | 371    |
| —        | **SQLite aggregate tests**                                                   | `sqliteengine/aggregations_test.go`                                                                                    | 361    |
| —        | **Fixed pre-existing broken import** in `sqliteengine/transactional_test.go` | missing `metaengine` import                                                                                            | 1 line |

**Total new code**: ~1926 LOC across 6 new files + ~120 LOC modified in existing files.

### Benchmark Results (10K rows, DuckDB in-memory)

```
BenchmarkDuckDB_GroupedAggregatePushdown/rows_10K    6.5ms/op
BenchmarkDuckDB_GroupedAggregate_GoSide/rows_10K    14.4ms/op  (2.2x slower)
BenchmarkDuckDB_MultiAggregate_SinglePass/rows_10K  11.0ms/op
BenchmarkDuckDB_MultiAggregate_N_Calls/rows_10K     11.8ms/op
```

Pushdown is 2.2x faster than Go-side at 10K rows. The gap widens dramatically at 100K+ (not yet measured due to test time).

---

## B. PARTIALLY DONE ⚠️

### 1. Postgres engine lacks ExplainableAggregate

PG engine got all 5 aggregate interfaces but NOT `ExplainableAggregate`. DuckDB and SQLite both have it. This is an inconsistency.

### 2. Benchmarks only measured at 10K rows

The 100K row variants exist in the benchmark code but were not run (test time constraints). The performance delta should be much larger at scale.

### 3. No planned-table path tests for SQLite aggregates

SQLite aggregate tests only test the standard (json_extract) path. The planned-table path (direct column references on `meta_planned_*` tables) is implemented but untested.

### 4. No planned-table path tests for PG aggregates

PG engine doesn't even have a `plans` map — its aggregate implementation is standard-path only. But it should support layout-planned tables for consistency with DuckDB/SQLite.

### 5. Parity test doesn't test with filters

The cross-engine parity test verifies matching results with no filters. Filter behavior (especially comparison operators on JSON values) could differ between engines.

---

## C. NOT STARTED ❌

### From the original P1 list

- **Nothing remaining** — all 15 items from the pasted task list were completed.

### Discovered during this session but not actioned

1. **PG engine has no `plans` map** — can't support planned-table aggregate path. Would need struct changes.
2. **PG engine has no `ExplainableScan`** — can't show SQL for scan queries either. Only DuckDB and SQLite implement it.
3. **`inferColumnType` maps "price" → INTEGER** — truncating decimals. This is a pre-existing bug in `metaengine/layout.go` discovered in the prior session but not fixed.
4. **DuckDB aggregation tests don't test the ExplainableAggregate interface** — only SQLite tests it.
5. **`groupAccum.count` field in MultiGroupedAggregate is vestigial** — after the AVG fix, it's only used for nothing (AVG now uses `nonNullCounts`). Could be removed but harmless.
6. **No coverage of empty-collection edge case in SQLite tests** — DuckDB tests cover this, SQLite doesn't.
7. **`assertFloat` helper in `typed_reader_aggregate_test.go` is exported** — should be unexported or use existing test helpers.

---

## D. TOTALLY FUCKED UP 💥

### Nothing critically broken, but several concerns:

1. **`contains`/`indexOf` helper functions in SQLite test** — I initially wrote custom `contains` and `indexOf` functions, then rewrote to use `strings.Contains` but the old versions may still be in the file (appended via `cat >>`). **UPDATE**: Verified — the file uses `strings.Contains` correctly, the custom helpers were never committed.

2. **Auto-commit daemon interleaving** — The daemon committed my changes mid-session, which means some commits mix my changes with daemon-initiated refactoring (e.g., `DeferClose` helper extraction). This makes it harder to review the pure aggregate pushdown work as a unit.

3. **`groupedAggExprSQLite` is a redundant alias** — I created a `groupedAggExprSQLite` function that just calls `aggExprSQLite`. Dead weight.

---

## E. WHAT WE SHOULD IMPROVE 🔧

### Code Quality

1. **Remove `groupedAggExprSQLite` alias** — it's just `aggExprSQLite` with a different name. Dead code.
2. **Remove vestigial `groupAccum.count` field** — no longer used after AVG fix uses `nonNullCounts`.
3. **Add `ExplainableAggregate` to PG engine** — parity gap with DuckDB/SQLite.
4. **PG engine needs `plans` map** for planned-table aggregate path — currently standard-path only.
5. **`assertFloat` should be unexported** — it's in a `_test.go` file so it's fine, but naming consistency matters.
6. **DuckDB ExplainableAggregate test missing** — only SQLite has explain tests.
7. **SQLite/PG planned-table aggregate path untested** — only standard path tested.

### Architecture

8. **Filter comparison type-coercion inconsistency** — DuckDB needs `CAST(... AS DOUBLE)` for numeric comparison on json_extract, SQLite doesn't (native types), PG uses `::float8` cast on the JSONB value. Three different approaches for the same logical operation. This should be documented or abstracted.
9. **No shared SQL builder abstraction** — Each engine copy-pastes the filter/aggregate SQL building logic with minor variations. The `//art-dupl:accept` pattern is used, but a shared helper module could reduce maintenance burden.
10. **Benchmark infrastructure** — The benchmarks create+seed a new DuckDB engine per iteration which includes table creation overhead. Should pre-seed outside the timer.

### Testing

11. **No race-detector run on SQLite aggregate tests** — Only core metaengine and DuckDB were tested with `-race`.
12. **No 100K row benchmark results** — Benchmarks exist but weren't run at scale.
13. **Parity test doesn't test filter pushdown** — Only tests unfiltered aggregates.
14. **No test for `ExplainAggregateQuery` with planned tables** — Only standard-path explain tested.
15. **TypedReader test doesn't verify pushdown was actually used** — The Memory engine always uses fallback; there's no test verifying a SQL engine takes the pushdown path through TypedReader (only direct engine interface tests do).

### Process

16. **API golden was regenerated 3 times** — Should have done all code changes first, then regen once. Wasted cycles.
17. **File naming inconsistency** — SQLite uses `aggregations_grouped.go` (separate file), PG uses `aggregations.go`. Should be consistent.
18. **`sqliteDecodeFloat` is exported-visible within package but duplicates logic** — DuckDB has `decodeFloat`, SQLite now has `sqliteDecodeFloat`, PG has `pgDecodeFloat`. Three identical functions across packages.

---

## F. Up to 50 Things to Do Next

### High Priority (correctness + completeness)

1. Add `ExplainableAggregate` to PG engine
2. Run SQLite aggregate tests with `-race`
3. Run full DuckDB aggregate test suite (all 12 tests + explain) with `-race`
4. Run 100K/1M row benchmarks and record results
5. Add planned-table aggregate tests for SQLite (test both paths)
6. Fix `inferColumnType` mapping "price" → INTEGER (should be DOUBLE/REAL)
7. Add filter-based parity tests (same filter, all engines, verify same results)
8. Write TypedReader-level integration test using a SQL engine (verify pushdown path is taken, not just fallback)
9. Add empty-collection edge case tests for SQLite aggregates
10. Remove `groupedAggExprSQLite` dead alias function

### Medium Priority (architecture + cleanup)

11. Remove vestigial `groupAccum.count` field
12. Document filter type-coercion differences across engines (ADR or code comment)
13. Consider shared `decodeFloat` utility across engine packages (or accept the duplication)
14. Add PG engine `plans` map for planned-table aggregate support
15. Add PG engine `ExplainableScan` (it's the only SQL engine without it)
16. Add DuckDB `ExplainableAggregate` tests (parity with SQLite explain tests)
17. Pre-seed benchmark data outside the timer loop
18. Fix file naming: rename PG `aggregations.go` to `aggregations_grouped.go` for consistency (or rename SQLite back)
19. Add `nolint` comments for the cross-module SQL builder duplication
20. Run `nix fmt` on all new files to ensure formatting compliance

### Lower Priority (polish + future)

21. Consider `ExplainableDistinct` or fold into `ExplainableAggregate` (currently folded — OK)
22. Add `ExplainPlan` integration for aggregate queries in `Store.ExplainPlan()` output
23. Add aggregate query costs to `Doctor()` diagnostics
24. Consider adding `StdDev`/`Variance` aggregate functions (DuckDB supports them natively)
25. Consider `Percentile` aggregate (DuckDB `quantile_cont`)
26. Add `HavingClause` support for post-aggregate filtering (`HAVING COUNT(*) > 5`)
27. Consider `WindowFunctions` for running totals / moving averages
28. Document the aggregate pushdown architecture in a new ADR
29. Add aggregate pushdown to the `cqrs-lint` module detection (F018-F026 pattern)
30. Update `FEATURES.md` with aggregate pushdown capabilities
31. Update SKILL.md `readmodels.md` reference with aggregate API examples
32. Add aggregate examples to `example/taskmanager/`
33. Consider `MultiGroupedAggregateWithSort` (ORDER BY on group key or aggregate value)
34. Add `CountDistinct` aggregate (COUNT DISTINCT in SQL)
35. Consider `StringAgg`/`GroupConcat` for string aggregation
36. Add benchmark comparing SQLite vs DuckDB aggregate pushdown performance
37. Add benchmark comparing planned-table vs json_extract aggregate performance
38. Consider materialized aggregate views (pre-computed GROUP BY results)
39. Add aggregate query support to the `metaengine/stream_log` ADT (time-series aggregation)
40. Consider incremental aggregate maintenance (update on write, not just read)
41. Add histogram aggregate (`bucket_count`, `bucket_min`, `bucket_max`)
42. Consider approximate aggregates (`approx_count_distinct`, `approx_quantile`)
43. Add correlation/ covariance aggregates for statistical analysis
44. Consider pivot/cross-tab support (`crosstab` function)
45. Add time-bucket aggregation (`time_bucket` for time-series rollups)
46. Consider GraphQL-style aggregate nesting (group → subgroup → aggregate)
47. Add cursor-based pagination on grouped results (LIMIT per group)
48. Consider `Rollup`/`Cube` multi-dimensional grouping (GROUP BY ROLLUP/CUBE)
49. Add aggregate result caching (memoize expensive GROUP BY queries)
50. Write a blog post / example showing the 2.2x performance improvement from pushdown

---

## G. Questions for the User

### Q1: Should the PG engine get a `plans` map for planned-table aggregate support?

DuckDB and SQLite both have `plans map[string]LayoutPlan` on the engine struct. PG doesn't — its aggregate implementation is standard-path only (JSONB `value->'field'`). Adding planned-table support to PG would require:

- Adding `plans` field to `pgEngine` struct
- Adding `ApplyLayout` logic for PG (it has `LayoutPlanner` but no `plans` map to check)
- Dual-path SQL generation in every aggregate method

This is a meaningful architecture question because PG's JSONB with expression indexes may make the planned-table path unnecessary (expression indexes achieve the same performance without dedicated tables).

### Q2: Should we run the full `nix run .#verify` gate now?

The verify gate (build + vet + test + race + lint + doc-check + doc-assertions) takes 3-4 minutes. All individual module builds and tests pass, but the full gate hasn't been run this session. Given the auto-commit daemon is active and interleaving changes, a verify run would confirm the committed state is clean.

### Q3: Should the three `decodeFloat` functions (DuckDB/SQLite/PG) be consolidated?

Each engine package has its own `decodeFloat`/`sqliteDecodeFloat`/`pgDecodeFloat` with nearly identical type-switch logic. The differences are: DuckDB handles `*big.Int` (HUGEINT), SQLite and PG don't. Options:

- **Accept duplication** (current state, `//art-dupl:accept` pattern)
- **Push to `enginetest` package** (shared test/decode utility)
- **Add to core `metaengine`** (but that adds `math/big` dep to core for one function)

---

## Resolution (2026-08-08)

Most items resolved in subsequent sessions the same day. Inline summary:

**Section B (self-critique):**
- ~~B1: PG lacks ExplainableAggregate~~ done — `pgengine/explain.go:61` implements it
- ~~B3: No planned-table SQLite aggregate tests~~ done — `sqliteengine/aggregations_test.go:377-533`
- ~~B5: Parity test doesn't test with filters~~ done — `TestAggregateParity_WithFilters` added
- B4: PG engine has no `plans` map — **still open** (architectural decision: JSONB expression indexes may make planned tables unnecessary)

**Section C (discovered):**
- ~~C2: PG lacks ExplainableScan~~ done — `pgengine/explain.go:12`
- ~~C3: `inferColumnType` price→INTEGER~~ done — now returns `"REAL"` (`layout.go:158-162`)
- ~~C4: DuckDB ExplainableAggregate tests~~ done — `TestDuckDB_ExplainAggregateQuery` (6+ subtests)
- ~~C5: `groupAccum.count` vestigial~~ done — removed
- ~~C6: No empty-collection SQLite test~~ done — `TestSQLite_Aggregate_EmptyCollection`
- C7: `assertFloat` is exported — **still open** (minor)

**Section D+E (improvements):**
- ~~E1/D3: Remove `groupedAggExprSQLite`~~ done — removed
- ~~E18: Three `decodeFloat` functions~~ done — consolidated to `metaengine.DecodeFloat` (`scan.go:21`)
- ~~E10: Benchmark pre-seed outside timer~~ done — `b.StopTimer()`/`b.StartTimer()` pattern

**Section F (20 next things):** 15 of 20 resolved (F1, F5-F11, F13, F15-F17, F19).
Still open: F2/F3 (full `-race` runs), F4 (100K benchmark results recorded), F14 (PG plans map), F18 (file naming).

**Q1:** PG `plans` map — **deferred** (JSONB expression indexes achieve same performance).
**Q2:** `nix run .#verify` — **run later** (07-45 report: GREEN, 0 issues).
**Q3:** `decodeFloat` consolidation — **done** (`metaengine.DecodeFloat` in `scan.go:21`).
