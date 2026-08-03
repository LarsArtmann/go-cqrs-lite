# Status Report — Metaengine Quality Paydown: PG Testcontainers, ScanBackend Verification, VersionedStorage Hardening

**Date:** 2026-08-01 16:45
**Session scope:** Execute remaining tasks from the quality paydown plan (`docs/planning/2026-08-01_15-08_SUPERB-METAENGINE-QUALITY-PAYDOWN.md`): L3.3-L3.5 (PG testcontainers + ScanBackend tests), L4.1-L4.4 (batch counter, property-based versioning, ExecuteAsOf integration, final gate).
**Branch:** master
**Predecessor:** `docs/status/2026-08-01_15-07_tier4-fixup-quality-gap-closure.md`

---

## Executive Summary

This session closed the **verification gap** — the most critical quality debt from the prior plan. Postgres engine tests had NEVER run against a real database; now they do, via testcontainers-go with per-test isolation. DuckDB and Postgres ScanBackends are verified against real databases. A property-based test (100 iterations via rapid) proves VersionedStorage correctness. A full Plan→Apply→ExecuteAsOf integration test caught and fixed a real bug (MapUpdate was not recording versions). The auto-commit daemon concurrently added PushdownScan + LayoutPlanner + adttest matrix to both engines, which this session also debugged and lint-fixed.

**`nix run .#verify`: ✅ ALL CHECKS PASSED** (build + vet + test + race + lint 0 issues + doc-check 1169 refs + API surface 3086 exports)

---

## A. FULLY DONE (verified, tested, gate-green)

### A1. Postgres Testcontainer Setup (L3.3)

- Created `metaengine/pgengine/testcontainer_test.go` following the `stack/postgres/testcontainer_test.go` pattern:
  - `TestMain` starts a shared `postgres:16-alpine` container; priority: `POSTGRES_TEST_DSN` env (CI) → testcontainers (local) → skip
  - `pgDSN(t)` creates a per-test fresh database (`test_1`, `test_2`, ...) via `CREATE DATABASE` for parallel-test isolation
  - `replaceDBInDSN` swaps the DB name in URL-format DSNs
  - `t.Cleanup` drops the per-test database with `DROP DATABASE ... WITH (FORCE)`
- Added `testcontainers-go` + `testcontainers-go/modules/postgres` to `pgengine/go.mod`
- **All 6 existing tests now run against real Postgres** — previously they always skipped
- **Status:** ✅ Complete — first time pgengine has EVER been tested against a real database

### A2. Postgres ScanBackend Test (L3.5)

- Added `TestPostgresEngine_ScanBackend` to `engine_test.go` — 4 sub-cases:
  1. No filter, no sort → returns all rows
  2. Filter by category → returns matching subset
  3. Sort by price descending + limit 3 → ordered results
  4. Keyset pagination (cursor = price 2.0) → returns items below cursor
- **Verified against real Postgres:** JSONB decode, Go-side filter/sort, keyset pagination all correct
- **Status:** ✅ Complete

### A3. DuckDB ScanBackend Test (L3.4)

- Added `TestDuckDBEngine_ScanBackend` to `engine_cgo_test.go` — same 4 sub-cases as Postgres
- **Verified with CGo:** DuckDB JSON decode, Go-side filter/sort, keyset pagination all correct
- All 5 DuckDB tests pass (Map, Counter, Profile, MetaenginePlan, ScanBackend)
- **Status:** ✅ Complete

### A4. Batch Postgres CounterIncrement (L4.1)

- Rewrote `CounterIncrement` from N individual `ExecContext` calls (one per delta key) to a single multi-row `INSERT ... VALUES ($1,$2,$3), ($1,$4,$5), ... ON CONFLICT DO UPDATE`
- Keys are sorted deterministically for reproducible placeholder ordering
- Empty-delta fast-path: `if len(deltas) == 0 { return nil }`
- **Verified against real Postgres:** `TestPostgresEngine_CounterBackend` passes (open: 3+2=5, closed: 1)
- **Status:** ✅ Complete — N round-trips → 1 round-trip

### A5. Property-Based VersionedStorage Test (L4.2)

- Added `TestMemoryEngine_VersionedStorage_Property` to `memory_versioned_test.go`
- Uses `pgregory.net/rapid` to generate 100 random sequences of MapSet/MapDelete across 3 keys
- Maintains a reference model (`map[string]int64`) alongside the engine
- After each operation, captures a timestamped snapshot
- Verifies `MapGetAsOf(t)` at every recorded timestamp for every key matches the reference model
- Also verifies "before first write" returns `ErrNotFound` for all keys
- **Result:** `OK, passed 100 tests`
- **Status:** ✅ Complete

### A6. ExecuteAsOf Integration Test (L4.3)

- Added `TestStore_ExecuteAsOf_Integration` — full pipeline test: `Plan → Apply → ExecuteAsOf`
- Uses branded `UserID` type, 3 fold handlers (insert, update, remove)
- Writes user at t1, updates name at t2, deletes at t3
- Verifies temporal reads at each timestamp return the correct version
- Verifies "before creation" returns `ErrNotFound`
- **Status:** ✅ Complete

### A7. MapUpdate Version Recording Bug Fix (discovered during L4.3)

- **Bug:** `memoryEngine.MapUpdate` did not call `recordVersion`, so FoldUpdate changes were invisible to `ExecuteAsOf`
- **Root cause:** When opt-in versioning was added (prior session), `MapSet` and `MapDelete` got `recordVersion` calls, but `MapUpdate` was missed
- **Impact:** Any projection using FoldUpdate (the `func(e, prev) V` pattern) would have broken temporal reads — `ExecuteAsOf` would return stale data after any update
- **Fix:** Added `recordVersion(col, fmt.Sprint(key), newVal)` to `MapUpdate` when `m.versions != nil`
- **Status:** ✅ Fixed — the integration test would have failed without this fix

### A8. Daemon Pushdown Code Lint Fixes

- The auto-commit daemon concurrently added `PushdownScan` + `LayoutPlanner` to both pgengine and duckdbengine
- This code arrived with 4 lint issues that this session fixed:
  1. `ineffassign` in both `pushdown.go` files — `ph` counter incremented after last use. Replaced with `len(args)+1` dynamic placeholder calculation
  2. `gci` import ordering in 3 files — fixed by `nix fmt`
  3. `prealloc` in property test — changed `var timeline []stateSnapshot` to `make([]stateSnapshot, 0, 25)`
  4. `staticcheck S1016` — changed `UserView{ID: e.ID, ...}` to `UserView(e)` (struct conversion)
- **Status:** ✅ All fixed, 0 lint issues

### A9. API Stability Golden Regenerated

- 3083 → 3086 exports (+3: `PushdownMapScan` on both engines, `ApplyLayout` on pgengine)
- `cmd/api-stability` golden regenerated and verified
- **Status:** ✅ Complete

### A10. `nix run .#verify` — GREEN

- Build: ✅ all 63 modules
- Vet: ✅ clean
- Test: ✅ all pass (including `-race`)
- Lint: ✅ 0 issues across all modules
- Doc-check: ✅ 1169 references valid across 41 packages
- API surface: ✅ 3086 exports stable
- **Status:** ✅ **GREEN**

---

## B. PARTIALLY DONE (shipped but with caveats)

### B1. adttest Cross-Engine Matrix (daemon-contributed)

- **Done:** The daemon added `adt_matrix_test.go` (pgengine) and `adt_matrix_cgo_test.go` (duckdbengine) that run `adttest.RunMatrix` against both engines. Map, Counter, and SortedMap pass with memory-engine parity. 7 unimplemented ADTs auto-skip.
- **Missing:** Vector, Search, Spatial, Set, Graph, Log, Multimap backends are not implemented on either SQL engine. They auto-skip via reflection (the matrix detects which backend interfaces an engine implements).
- **Note:** This was daemon work, not this session's work, but this session verified it passes.

### B2. DuckDB PushdownScan (daemon-contributed)

- **Done:** `duckdbengine/pushdown.go` implements `PushdownMapScan` using DuckDB's `json_extract(value, '$.field')` for WHERE/ORDER BY pushdown.
- **Missing:** DuckDB does NOT implement `LayoutPlanner` (no `ApplyLayout`). DuckDB's columnar advantage is minimal when data is stored as JSON blobs in a VARCHAR column. True columnar storage would require a different table schema.

### B3. Postgres LayoutPlanner (daemon-contributed)

- **Done:** `pgengine/pushdown.go` implements `ApplyLayout` creating partial expression indexes (`CREATE INDEX ... ON meta_map ((value->'field')) WHERE collection = '...'`).
- **Missing:** GIN indexes (for `@>` containment queries) are not implemented. The FilterSpec system doesn't yet declare containment operators.

---

## C. NOT STARTED (planned but not done)

### C1. L3.11: cqrs-lint DomainBias

- Domain-based severity calibration (`DomainBias` on `FeatureProfile`). Deferred from original plan.

### C2. L4.10: cqrs-lint Cross-Module Rules

- Snapshot codec mismatch, event type typo detection, orphaned type detection. Deferred.

### C3. L4.11: cqrs-lint New Categories (DOC/OBS/RES/DI)

- Documentation, observability, resource, and dependency-injection rule categories. Deferred.

### C4. DuckDB/Postgres Advanced Backends

- VectorBackend, SearchBackend, SpatialBackend for DuckDB/Postgres. Not started.

### C5. DuckDB LayoutPlanner

- Columnar DDL generation for DuckDB. Not started.

### C6. Postgres GIN Index Support

- `@>` containment and `?` existence operators for JSONB. Requires new FilterSpec operators.

### C7. Tag duckdbengine + pgengine

- Neither module has been tagged with a semver release. Should be done after verification (now complete).

---

## D. TOTALLY FUCKED UP (mistakes, oversights, quality gaps)

### D1. Did Not Test MapUpdate Version Recording Before Writing the Integration Test

I wrote the ExecuteAsOf integration test assuming MapUpdate already recorded versions. It didn't — the test failed on the first run. I then traced the bug, fixed `MapUpdate`, and the test passed. But the correct approach would have been to audit ALL write paths (`MapSet`, `MapDelete`, `MapUpdate`) for `recordVersion` coverage BEFORE writing the test. I discovered the bug by accident (the test caught it), not by design.

### D2. Race with the Auto-Commit Daemon

The daemon was simultaneously adding PushdownScan, LayoutPlanner, adttest matrix, and other changes to the SAME files I was editing. This caused:

- `pushdown.go` to arrive with compilation errors (missing struct fields) — the verify gate caught this
- API surface mismatches (2 new exports I didn't add) — required golden regen
- Concurrent modifications to `engine.go` and `scan.go` — I had to reconcile
  The verify gate caught all issues, but the concurrent development made the session chaotic. I should have anticipated daemon interference and either disabled it or worked in a worktree.

### D3. Did Not Add ScanBackend Tests to the adttest Matrix

The adttest matrix (daemon-contributed) tests Map, Counter, SortedMap via the backend interfaces. But ScanBackend is tested only via per-engine tests, not via the cross-engine matrix. If one engine's MapScan has a subtle divergence (e.g. different filter semantics), the matrix won't catch it. The scan tests are duplicated manually in both engine test files.

### D4. The `seedProducts` Test Helper Is Duplicated

Both `engine_test.go` and `pushdown_test.go` (daemon-contributed) in pgengine need to seed items into a collection. The pushdown test created its own `seedProducts` helper. The ScanBackend test in `engine_test.go` inlines the seeding. These should share a single helper. Minor, but it's duplication.

### D5. Property Test Timestamp Granularity

The property-based test uses `time.Sleep(time.Microsecond)` between operations to ensure distinct timestamps. This is fragile — on a loaded machine, `time.Now()` might not advance fast enough. A more robust approach would use a monotonic counter as the timestamp source instead of wall-clock time. The current approach works (100 tests pass), but it's a latent flakiness risk.

---

## E. WHAT WE SHOULD IMPROVE

1. **Audit ALL write paths when adding cross-cutting behavior** — When opt-in versioning was added, MapSet and MapDelete got `recordVersion`, but MapUpdate was missed. A systematic audit of all methods that mutate `m.data.maps` would have caught this. The pattern: any new behavior that must fire on every write should be verified against EVERY write method.

2. **Share test infrastructure across engine test files** — The `seedProducts` helper and the ScanBackend test pattern (filter, sort, limit, cursor) are identical between pgengine and duckdbengine. A shared `scantest` package (like `adttest`) would eliminate the duplication and ensure both engines are tested identically.

3. **Use monotonic timestamps in versioning tests** — Wall-clock `time.Now()` with `time.Sleep(time.Microsecond)` is fragile. The engine could accept an injectable clock interface, or tests could use a custom timestamp source.

4. **Consider disabling the auto-commit daemon during active sessions** — The daemon's concurrent modifications caused compilation errors, API surface mismatches, and required manual reconciliation. At minimum, the daemon should not modify files in modules that are being actively edited.

5. **Pushdown limit+1 semantics should be documented** — The daemon's PushdownMapScan uses `LIMIT n+1` for has-more detection, but this is inconsistent with ScanBackend.MapScan which uses `LIMIT n` exactly. The two scan paths have different limit semantics for the same `limit` parameter. This is a design split brain.

6. **Add a cross-engine pushdown parity test** — The adttest matrix tests backend interfaces (MapBackend, CounterBackend), but NOT PushdownScan. A shared pushdown test matrix would ensure both engines produce identical results for the same FilterSpec/SortSpec inputs.

7. **Tag both engine modules** — duckdbengine and pgengine have never been tagged. They now have real testcontainer/CGo verification. They should be tagged `v4.0.0` (or next appropriate version).

---

## F. Up to 50 Things to Get Done Next

### Critical (verification gaps)

1. Tag `metaengine/pgengine/v4.0.0` and `metaengine/duckdbengine/v4.0.0` (verification now complete)
2. Add CGo CI job for duckdbengine (like stack/duckdb)
3. Add Postgres service container to CI for pgengine tests (testcontainers pattern)
4. Run `nix run .#check-layers` to verify dependency budgets for both engine modules
5. Add `nix run .#check-coverage` thresholds for both engine modules

### High-value (completes the engine feature set)

6. Implement VectorBackend for DuckDB (using DuckDB's array type or VSS extension)
7. Implement SearchBackend for Postgres (using tsvector/tsquery)
8. Implement SpatialBackend for Postgres (using PostGIS or earthdistance)
9. Add GIN index support to pgengine LayoutPlanner (`@>` containment operator)
10. Implement DuckDB LayoutPlanner (columnar DDL generation)
11. Add `FilterContains` and `FilterExists` operators to FilterSpec
12. Unify limit semantics between ScanBackend.MapScan and PushdownScan
13. Extract shared `scantest` package for cross-engine ScanBackend parity tests
14. Add cross-engine pushdown parity test (Memory vs Postgres vs DuckDB)
15. Refactor `seedProducts` into shared test helper

### Testing improvements

16. Replace wall-clock timestamps in versioning tests with injectable clock
17. Add concurrent Apply + SwapEngine + VersionedStorage chaos test
18. Benchmark: Postgres MapScan vs PushdownMapScan at scale (1K, 10K, 100K rows)
19. Benchmark: DuckDB MapScan vs PushdownMapScan at scale
20. Benchmark: VersionedStorage asOf reads with long version chains (1K, 10K versions)
21. Calibration benchmarks for pgengine (vary N: 100/1K/10K/100K)
22. Calibration benchmarks for duckdbengine
23. Add property-based test for ScanBackend (random filter/sort combinations)
24. Add property-based test for PushdownScan (random FilterSpec/SortSpec)
25. Add soak test for pgengine (sustained writes over time, verify no connection leaks)
26. Test Postgres CounterIncrement batch with 100+ deltas (verify placeholder limit)
27. Test PushdownMapScan with empty filter + empty sort (should behave like MapScan)
28. Test PushdownMapScan with nil cursor + non-nil sort (should not add cursor WHERE clause)

### cqrs-lint remaining

29. L3.11: DomainBias struct on FeatureProfile with per-domain severity overrides
30. L4.10: Snapshot codec mismatch detector
31. L4.10: Event type typo detection (similar but not identical event types)
32. L4.10: Orphaned type detection (types declared but never used in any handler)
33. L4.11: DOC category — missing doc comments on exported symbols
34. L4.11: OBS category — missing observability (no OTel spans on hot paths)
35. L4.11: RES category — resource leak detection (unclosed stores, engines)
36. L4.11: DI category — dependency injection anti-patterns

### Architecture & Documentation

37. ADR for PushdownScan interface and its relationship to ScanBackend
38. ADR for LayoutPlanner and index generation patterns
39. ADR for the limit+1 has-more semantics in PushdownScan
40. Update SKILL.md recipes with PushdownScan examples
41. Update AGENTS.md module descriptions for pgengine/duckdbengine (now have pushdown)
42. Update migration guide with pushdown vs scan decision matrix
43. Add cookbook recipe: "Filtered queries with PushdownScan on Postgres"
44. Add cookbook recipe: "Analytical queries with DuckDB pushdown"
45. Update FOUR-TIER-MODEL.md with new engine capabilities

### Infrastructure

46. Add `Stats` struct and `WithStats()` option (materialize-vs-replay formula wiring)
47. Add versionedStorage to the metaengine Explain output (show temporal capability)
48. Consider extracting metaengine into its own repository
49. Add `metaengine-gen` code generator for typed Store methods
50. Add DuckDB vectorized CounterGet (`SUM...GROUP BY` pushdown)

---

## G. Questions I CANNOT Answer Myself

### 1. Should the auto-commit daemon be disabled during active editing sessions?

The daemon added PushdownScan, LayoutPlanner, adttest matrix, doc updates, and 40+ files of changes WHILE I was editing the same modules. This caused compilation errors, API surface mismatches, and required constant reconciliation. The verify gate caught everything, but the friction was significant. Should I work in a `git worktree` to isolate from the daemon, or should the daemon be paused during active sessions?

### 2. Should ScanBackend.MapScan and PushdownScan have unified limit semantics?

`MapScan` uses `LIMIT n` (exact). `PushdownScan` uses `LIMIT n+1` (has-more detection). These are inconsistent interfaces for what consumers see as the same "limit" parameter. Should they be unified? If so, which semantic is correct — should MapScan also do n+1, or should PushdownScan drop the n+1 pattern? The n+1 approach enables has-more detection without a separate COUNT query, but it changes the return count contract.

### 3. Should pgengine use `value->'field'` (JSONB) or `value->>'field'` (text) for pushdown?

The daemon chose `value->'field'` (returns JSONB, preserves numeric types for comparison). The alternative `value->>'field'` returns text, which would require casting for numeric comparisons but might be more compatible with index types. The current choice works (tests pass), but I don't know if this is the right long-term tradeoff for Postgres JSONB query patterns, especially when GIN indexes are added later.

---

## Session Metrics

| Metric                        | Value                                         |
| ----------------------------- | --------------------------------------------- |
| Tasks planned                 | 8 (L3.3-L3.5, L4.1-L4.4)                      |
| Tasks completed               | 8                                             |
| Bugs found and fixed          | 1 (MapUpdate missing recordVersion)           |
| Real DB tests added           | 7 (6 pgengine + 1 duckdb ScanBackend)         |
| Property-based tests added    | 1 (100 iterations, VersionedStorage)          |
| Integration tests added       | 1 (ExecuteAsOf full pipeline)                 |
| Lint issues fixed             | 4 (ineffassign ×2, prealloc, staticcheck)     |
| API surface exports           | 3083 → 3086                                   |
| Daemon commits during session | 5 (pushdown, matrix, docs, refactor)          |
| `nix run .#verify` runs       | 4 (1 build failure, 2 lint failures, 1 GREEN) |
| Final gate status             | ✅ GREEN                                      |

---

## Resolution (2026-08-03)

All 8 tasks (L3.3-L3.5, L4.1-L4.4) shipped. PG testcontainers, ScanBackend tests, batch CounterIncrement, property-based VersionedStorage, ExecuteAsOf integration — all in production. MapUpdate version-recording bug fixed.

**Q1 (daemon friction):** Process question — resolved by working patterns (worktree isolation documented in AGENTS.md).
**Q2 (limit semantics):** DONE — unified via `ScanResult{HasMore}` (`18-14`). The n+1 convention was replaced.
**Q3 (JSONB operators):** Design decision made — `value->'field'` (JSONB) chosen, preserves numeric types. GIN indexes still deferred (T23).
