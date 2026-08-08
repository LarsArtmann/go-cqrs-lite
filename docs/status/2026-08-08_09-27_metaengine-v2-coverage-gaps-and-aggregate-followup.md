# Status Report: Session 3 — Metaengine v2 Coverage Gaps + Aggregate Pushdown Follow-up

**Date:** 2026-08-08 09:27 CEST
**Session Start:** ~08:34 CEST (resumed from session 2 status report)
**Commits This Session:** 16 (13 mine + 3 daemon-interleaved)
**Working Tree:** Clean (auto-commit daemon committed everything)

---

## a) FULLY DONE (12/12 TODO items)

### Metaengine v2 — Test Coverage Gaps (5/5)

1. **DuckDB race regression test** — `metaengine/duckdbengine/race_regression_cgo_test.go`
   - `TestDuckDB_RaceRegression_LayoutPlanConcurrentAccess`: 30 goroutines (10 ApplyLayoutPlan writers + 10 ExplainAggregateQuery readers + 10 MapSet readers) × 50 iterations
   - Verified under `-race`: PASS (0.10s)
   - Build tags: `goexperiment.jsonv2 cgo`

2. **`lookupPlan` shallow-copy semantics documented** — `metaengine/duckdbengine/engine.go:218`
   - Doc comment explains: struct copy returned, but slice fields (Columns, Indexes) share underlying array
   - All 5 callers are read-only today (MapSet, MapGet, MapDelete, explainScanQuery, ExplainAggregateQuery)

3. **DuckDB `t.Parallel()` audit** — all 54 test functions audited
   - Only `TestDuckDB_ExplainAggregateQuery` has `//nolint:tparallel` (correct: subtests share mutable engine)
   - Race regression test deliberately serial (no subtests, no nolint needed)
   - Every other test creates its own engine and calls `t.Parallel()` safely
   - **Result: no changes needed, codebase is consistent**

4. **Coverage baselines verified** — `nix run .#check-coverage` passes
   - Metaengine: 80.8% actual vs 81.0% baseline (-0.2%, within ±2.0% tolerance)
   - All other modules unchanged
   - No baseline updates needed

5. **QUIC convergence under `-parallel 4`** — `metaengine/irohengine/quic`
   - 3x pass with `-parallel 4 -count=3` and 30s timeout (0.03s each)
   - No flakiness detected

### Aggregate Pushdown — Follow-up Items (7/7)

6. **DecodeFloat extraction** — `metaengine/scan.go`
   - `metaengine.DecodeFloat`: shared scan-value-to-float64 normalizer (nil, float64, float32, int64, int, *big.Int, []byte)
   - `metaengine.DecodeFloatResults`: shared MultiAggregate result builder (takes raws + specs + errPrefix)
   - Eliminated 3-way duplication across DuckDB (`decodeFloat`), SQLite (`sqliteDecodeFloat`), Postgres (`pgDecodeFloat`)
   - Duplication gate: 0 new clones (baseline 67)
   - NOTE: The auto-commit daemon committed the initial DecodeFloat extraction as `a380e1ed1` and the DecodeFloatResults follow-up as `74cacfce1`

7. **PG functional aggregate tests** — `metaengine/pgengine/aggregations_test.go`
   - 7 test functions covering all 5 aggregate interfaces + empty collection + explain
   - `TestPostgres_Aggregate`: COUNT, SUM, MIN (negative), MAX, AVG, filtered COUNT
   - `TestPostgres_GroupedAggregate`: grouped count + grouped sum by status
   - `TestPostgres_MultiAggregate`: count + sum + min + max in one pass
   - `TestPostgres_MultiGroupedAggregate`: count + sum + avg per group with full row comparison
   - `TestPostgres_DistinctValues`: 2 distinct status values
   - `TestPostgres_Aggregate_EmptyCollection`: empty count + empty multi-aggregate
   - `TestPostgres_ExplainAggregateQuery`: non-empty SQL check
   - All pass via testcontainers (postgres:16-alpine, ~9s per run)

8. **DuckDB planned-path empty-collection test** — `metaengine/duckdbengine/aggregations_cgo_test.go`
   - `TestDuckDB_Aggregate_EmptyPlannedCollection`: all 5 interfaces on an empty planned table
   - Tests: Aggregate (COUNT + SUM), GroupedAggregate, MultiAggregate (count + total), MultiGroupedAggregate, DistinctValues

9. **Cross-engine planned-table parity test** — `metaengine/bench/aggregate_parity_cgo_test.go`
   - `TestAggregateParity_PlannedTable_DuckDB_vs_SQLite`: verifies identical aggregate results on planned tables
   - Tests: Count, Sum, Min, Max, Avg, GroupedCount/open, GroupedSum/closed
   - Added `newPlannedSQLiteEngine` helper in `sqlite_factory_test.go` (uses `NewPlannedSQLiteEngine` constructor)
   - Discovered: SQLite engine does NOT implement `LayoutPlanApplier` — plans must be registered at construction time via `NewPlannedSQLiteEngine`

10. **ReadPattern in SerializablePlan** — `metaengine/serializable.go` + `metaengine/plan_diff.go`
    - `SerializableQuery.ReadPattern` field added (populated from `QueryAssignment.ReadPattern`)
    - `QueryChange` struct now includes `OldReadPattern` + `NewReadPattern`
    - `PlanDiff` detects read-pattern changes (e.g., point_lookup → aggregate)

11. **Aggregate diagnostics in Doctor()** — `metaengine/explain.go`
    - New `--- Aggregate Pushdown ---` section in `Doctor()` output
    - `aggregateCapabilities` helper: checks 5 interfaces, returns "pushdown: scalar, grouped, multi, multi-grouped, distinct"
    - Collections without pushdown support show "none"

12. **ADR-0120** — `docs/adr/0120-aggregate-pushdown-architecture.md`
    - Documents: 5-interface design, cross-engine parity strategy, DecodeFloat extraction
    - Explains why aggregation goes to the engine instead of Go-side accumulation
    - Includes performance numbers (GROUP BY 4.4x faster, MultiAggregate 2.1x faster at 100K rows)

---

## b) PARTIALLY DONE

Nothing — all items are fully complete.

---

## c) NOT STARTED

Nothing from the assigned TODO list. All 12 items were addressed.

---

## d) TOTALLY FUCKED UP

1. **Auto-commit daemon interleaved unrelated work into my session** — The daemon committed 3 unrelated changes during my session:
   - `25ff10791` — SQLite WAL concurrency tests (unrelated to metaengine)
   - `3b4d48207` — System pebbleengine + watermill integration (unrelated)
   - `ab6f01b8e` — flake.nix DuckDB/Turso VM health tests (unrelated)
   These are mixed into the git history alongside my metaengine work, making `git bisect` harder.

2. **Empty commit message on `797d9ce45`** — The daemon committed my ADR-0120 + parity test + sqlite factory with a completely empty commit message. This violates every git convention and makes history scanning harder.

3. **Daemon commit `6bf856f2d` claims to have "refactored engine aggregations"** — This commit message says it touched `metaengine/duckdbengine/engine.go` and aggregation files. I verified my changes survived intact (the total diff is correct), but the daemon's commit message conflates system lifecycle test consolidation with metaengine aggregation refactoring, making it unclear what actually changed.

4. **`system/integration_lifecycle_test.go:214` typecheck error** — Pre-existing (confirmed by stashing my changes and re-running). The file is untracked from another session and has a `StreamID` vs `StreamRef` type mismatch. The daemon's system lifecycle commits may have introduced or worsened this. Not investigated deeply because it's outside my task scope.

---

## e) WHAT WE SHOULD IMPROVE

### Things I Forgot

1. **No direct unit test for `DecodeFloat` / `DecodeFloatResults`** — These are now shared core metaengine functions with zero dedicated tests. They're exercised indirectly through aggregate tests across 3 engines, but a direct unit test covering all type branches (nil, float64, float32, int64, int, *big.Int, []byte, unknown) would be stronger proof.

2. **No test verifying Doctor() renders the aggregate pushdown section** — The Doctor tests (`TestDoctor_*`) only use the Memory engine (which has no aggregate pushdown). I never verified the `--- Aggregate Pushdown ---` section actually appears in output when an aggregate-capable engine is registered. The code is correct (the helper checks all 5 interfaces), but there's no test assertion for it.

3. **PG `ExplainAggregateQuery` test is shallow** — Only checks `sqlStr != ""`. Should verify the SQL contains expected keywords (`SUM`, `GROUP BY`, `$1`, etc.) like the DuckDB equivalent does.

4. **PG `DistinctValues` test doesn't verify actual values** — Only checks `len(vals) == 2`. Should verify the actual distinct values are "open" and "closed".

5. **Didn't run the full DuckDB test suite under `-race`** — Only ran the race regression test in isolation under `-race`. The full DuckDB suite (95s without -race) would take ~5min with -race. Would catch any remaining races in other paths.

6. **Didn't run `nix fmt` proactively** — The verify gate caught gci import ordering issues in `duckdbengine/aggregations.go` and `pgengine/aggregations.go` after the DecodeFloat extraction. Running `nix fmt` before verify would have caught this in 1 second instead of wasting a 4-minute verify cycle.

7. **Didn't update AGENTS.md** — The metaengine section in AGENTS.md documents many exports but doesn't mention `DecodeFloat`/`DecodeFloatResults`. Also, `ReadPattern` is now serialized in `SerializableQuery` but the serialization docs don't mention it.

### Process Improvements

8. **Run `nix fmt` after multi-file refactors** — When touching imports across 3+ modules, always `nix fmt` before verify. The gci linter is strict about section ordering (standard → default → prefix).

9. **Write direct tests for shared helpers** — When extracting a helper used by 3 engines, write a unit test for the helper itself. Indirect coverage through 3 engines is good but not as clear as a focused test.

10. **Assert new Doctor() sections in tests** — When adding a new section to Doctor output, add a test that asserts the section header appears with a real engine that triggers it.

---

## f) Up to 50 Things We Should Get Done Next

### Correctness & Hardening (P0)

1. Write direct unit tests for `metaengine.DecodeFloat` (all 7 type branches + error case)
2. Write direct unit tests for `metaengine.DecodeFloatResults` (empty specs, nil raws, mismatched lengths)
3. Add Doctor() test with an aggregate-capable engine (SQLite) asserting `--- Aggregate Pushdown ---` section renders
4. Fix `system/integration_lifecycle_test.go:214` StreamID vs StreamRef typecheck error
5. Run full DuckDB test suite under `-race` (not just the regression test)
6. Add `TestPostgres_ExplainAggregateQuery` SQL content assertions (SUM keyword, $1 placeholder, etc.)
7. Verify `TestPostgres_DistinctValues` checks actual values ("open", "closed"), not just count

### Test Coverage Expansion (P1)

8. Add PG planned-path aggregate tests (PG has no `LayoutPlanApplier` — may need different approach)
9. Add aggregate tests with NULL values in columns (edge case: NULL handling differs by engine)
10. Add aggregate tests with large datasets (10K+ rows) to verify performance claims in ADR-0120
11. Add `MultiGroupedAggregate` parity test between DuckDB and SQLite (currently only scalar + grouped tested)
12. Add `DistinctValues` cross-engine parity test (currently only count checked)
13. Add filter + planned-table aggregate test (filters + planned columns simultaneously)
14. Add negative-value aggregate parity test (prices include -5, verify MIN returns -5 on all engines)

### Architecture & Design (P1)

15. Add `LayoutPlanApplier` support to SQLite engine (currently only DuckDB supports post-construction plan registration)
16. Consider deep-copying `LayoutPlan` in `lookupPlan` instead of documenting the shallow-copy constraint
17. Add aggregate pushdown cost to `SerializablePlan` (ns_per_aggregate is in ReadCosts but not surfaced per-query)
18. Document `DecodeFloatResults` limitation: cannot be used for `MultiGroupedAggregate` (different return type)
19. Consider a shared `DecodeGroupedFloatResults` for the grouped paths (3 engines have similar grouped scan loops)

### Documentation (P2)

20. Update AGENTS.md metaengine section with `DecodeFloat`/`DecodeFloatResults` exports
21. Update AGENTS.md with `ReadPattern` in `SerializableQuery` serialization docs
22. Update SKILL.md references if aggregate pushdown patterns changed
23. Add aggregate pushdown section to `docs/architecture-understanding/FOUR-TIER-MODEL.md`
24. Document the SQLite `NewPlannedSQLiteEngine` vs DuckDB `ApplyLayoutPlan` asymmetry in ADR-0120

### CI & Tooling (P2)

25. Add `-race` flag to the DuckDB CI job (currently only runs without -race due to time)
26. Consider splitting the DuckDB test suite into fast/slow groups for CI parallelism
27. Add `nix fmt` as a pre-commit hook (or pre-verify step) to catch gci issues early
28. Add coverage tracking for `metaengine/pgengine` and `metaengine/duckdbengine` (currently only core metaengine tracked)
29. Consider tagging `metaengine/v4.7.0` with the new `DecodeFloat`/`DecodeFloatResults`/`ReadPattern` exports
30. Run `cmd/doc-check` on ADR-0120 to verify all Go import paths are valid

### Performance (P3)

31. Benchmark `DecodeFloat` vs the old per-engine implementations (verify no regression from function call overhead)
32. Benchmark `DecodeFloatResults` vs inline loops (verify no regression from the extraction)
33. Add aggregate pushdown benchmarks for Postgres (currently only DuckDB + SQLite benchmarked)
34. Add `MultiGroupedAggregate` benchmarks (the most complex aggregate path)

### Cleanup (P3)

35. Remove the empty commit `797d9ce45` from history (requires interactive rebase — ask user first)
36. Consolidate the daemon's interleaved commits via rebase (separate metaengine from system/storage changes)
37. Audit all metaengine tests for missing `-race` coverage (not just DuckDB)
38. Verify `aggregateCapabilities` handles engines implementing subsets (e.g., only AggregateReader but not DistinctReader)
39. Add `//nolint:tparallel` audit for SQLite and PG engine tests (only DuckDB was audited)
40. Consider adding `ExplainableAggregate` to the Doctor output (show generated SQL per collection)

### Strategic (P4)

41. Evaluate whether `DecodeFloat` should handle `decimal.Decimal` types (for future decimal column support)
42. Consider a `Calibratable` aggregate benchmark (auto-detect which aggregate interfaces an engine supports and benchmark them)
43. Explore adding `WindowedReader` interface for window functions (ROW_NUMBER, RANK, DENSE_RANK)
44. Consider `PivotedAggregateReader` for CROSS TAB style pivots (rotate group keys to columns)
45. Add plan-pinning support for aggregate queries (pin the generated SQL via `Manifest`)
46. Consider exposing aggregate pushdown via `TypedReader` methods (currently only via interface type assertion)
47. Evaluate whether the Memory engine should implement `AggregateReader` (Go-side accumulation as fallback)
48. Add a metaengine integration test that runs the full Plan → Store → Execute → Aggregate lifecycle
49. Consider adding `ExplainAggregateQuery` output to `SerializablePlan` (for plan auditing)
50. Write a migration guide for consumers adopting aggregate pushdown (when to use it, when to avoid it)

---

## g) Questions (3)

### Q1: Should `LayoutPlanApplier` be added to the SQLite engine?

The DuckDB engine supports `ApplyLayoutPlan` (post-construction plan registration via the `LayoutPlanApplier` interface). The SQLite engine does NOT — it only supports plans at construction time via `NewPlannedSQLiteEngine`. This asymmetry made the cross-engine parity test harder to write (required different setup code per engine). Should I add `LayoutPlanApplier` support to SQLite for consistency, or is the constructor-time approach intentional?

### Q2: Should I tag `metaengine/v4.7.0` now?

This session added 3 new exported symbols to the metaengine core: `DecodeFloat`, `DecodeFloatResults`, and the `ReadPattern` field on `SerializableQuery`. The last tag was `metaengine/v4.6.0`. Should I tag `v4.7.0` now, or batch these with more changes first? The prior session left 14 untagged metaengine-related commits.

### Q3: Should the auto-commit daemon's interleaved commits be cleaned up?

The daemon committed 3 unrelated changes (SQLite WAL tests, system pebbleengine integration, flake.nix VM tests) mixed into my metaengine work. There's also a commit with an empty message (`797d9ce45`). Should I interactive-rebase to separate concerns, or leave the history as-is since the working tree is clean and all tests pass?

---

## Verification Summary

| Gate | Status | Notes |
|------|--------|-------|
| Build (`go vet`) | GREEN | All modules compile with `goexperiment.jsonv2` |
| DuckDB build (CGo) | GREEN | Compiles with `cgo` tag |
| Tests (metaengine) | GREEN | 7.1s |
| Tests (duckdbengine) | GREEN | 94.9s (all tests pass) |
| Tests (sqliteengine) | GREEN | 0.7s |
| Tests (pgengine) | GREEN | 15.5s (testcontainers) |
| Tests (bench) | GREEN | 0.05s (aggregate parity) |
| Race (duckdbengine) | GREEN | Regression test passes under `-race` |
| Race (quic) | GREEN | 3x pass under `-parallel 4` |
| Lint | GREEN | 0 issues on all touched modules |
| Duplication | GREEN | 0 new clones (baseline 67) |
| Coverage | GREEN | Within ±2.0% tolerance |
| API stability | GREEN | 3809 exports verified |
| Doc-check | GREEN | 1237 references valid |
| Pre-existing: system typecheck | RED | `integration_lifecycle_test.go:214` — untracked file, not my change |
