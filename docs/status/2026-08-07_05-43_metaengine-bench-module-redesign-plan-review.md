# Status Report: metaengine/bench Module Redesign — Plan Review

**Date:** 2026-08-07 05:43
**Session goal:** Create a comprehensive plan for restructuring the metaengine benchmark module to allow DuckDB columnar extraction benchmarking (M4.2)
**Outcome:** Plan was produced but contains CRITICAL ERRORS that must be fixed before execution

---

## a) FULLY DONE

### Architecture analysis and approach selection

- Presented 3 architectural approaches for breaking the DuckDB dep-boundary problem
- User selected Approach 1 (`metaengine/bench/` — dedicated cross-engine benchmark module, following `stack/bench/` precedent)
- This decision is FINAL and correct

### Research completed

- Read all 9 "promise" benchmark files in `metaengine/` (bench_promise, bench_fanout, bench_readmix, bench_enginepool, bench_planner, bench_storm, bench_layout, bench_materialize, bench_parity)
- Read DuckDB engine patterns: `duckdbengine/layout_bench_test.go`, `calibration_bench_test.go`, `engine.go`
- Read all relevant go.mod files: metaengine core, duckdbengine, pebbleengine, stack/bench
- Verified git tags: `metaengine/v4.6.0`, `metaengine/duckdbengine/v4.0.0` exist; sqliteengine and pebbleengine have NO tags
- Read `go.work` structure (80+ modules, only 1 non-local replace)
- Read api-stability modules list and bench-matrix.sh

### Plan produced

- 30-task table with priority scoring, effort estimates, dependency chains, and 8 execution phases
- The plan's STRUCTURE is sound: foundation → M4.2 core → DuckDB integration → migration → cleanup → Pebble → infrastructure → verification

---

## b) PARTIALLY DONE

### Plan table — structurally correct but factually flawed

The table EXISTS and has the right shape, but contains multiple errors (see section d).

### Task identification — incomplete

The plan lists 10 files to migrate from `metaengine/` but there are actually **11 bench files** (missed `bench_filter_test.go`, 209 lines). Additionally, 4 deprecated stub files (`calibration_bench_test.go`, `json_tax_bench_test.go`, `layout_bench_test.go`, `planner_bench_test.go`) need cleanup — these were completely missed.

---

## c) NOT STARTED

- **Zero code has been written.** No `go.mod` created, no files migrated, no benchmarks written.
- **No TODO list created via `todos` tool** — the user explicitly asked for this.
- **No execution has begun** — this was a planning-only session.

---

## d) TOTALLY FUCKED UP

### BUG 1: `sqlite_helpers_test.go` MUST NOT be deleted

The plan's Task 21 says "Delete 10 old bench files from `metaengine/` — 9 `bench_*_test.go` + `sqlite_helpers_test.go`." **THIS IS WRONG.** Six non-bench test files depend on `newSQLiteEngine()` and `newSQLiteEngineForPath()` from `sqlite_helpers_test.go`:

- `cost_assignment_test.go` (2 calls)
- `hardening_test.go` (4 calls)
- `pushdown_test.go` (1 call)
- `regression_test.go` (1 call)
- `reify_test.go` (defines `newSQLiteEngineForStd` which wraps it)
- `restart_test.go` (4 calls)

**Deleting `sqlite_helpers_test.go` would break 6 test files and break the build.** The new `metaengine/bench/` module needs its OWN copy of the SQLite factory. The original stays in `metaengine/`.

### BUG 2: Missed `bench_filter_test.go` entirely

`bench_filter_test.go` (209 lines) is a substantial benchmark file with its own types (`benchItemResult`, `benchListInput`), its own query declaration (`benchFilterQuery`), and `setupBenchStore` helper. It was completely absent from the 30-task plan. It needs to be migrated as a 11th file.

### BUG 3: Circular dependency in phase ordering

Task T9 ("extend parity with DuckDB") is in Phase 3 but depends on T13 ("migrate bench_parity_test.go") which is in Phase 4. You can't extend a file that hasn't been migrated yet. The phases need reordering: migration must happen BEFORE DuckDB integration.

### BUG 4: Tag situation not addressed in the plan

`metaengine/sqliteengine/v4` and `metaengine/pebbleengine/v4` have **ZERO published git tags**. The new `metaengine/bench/go.mod` will require them. In workspace mode (`go.work`), this works via the `use` directive. But for standalone builds (`GOWORK=off`), `go mod tidy` will try to fetch from VCS and hit the GOPRIVATE auth failure documented in AGENTS.md. The plan needs explicit `replace` directives for ALL untagged local modules.

### BUG 5: Score formula is fabricated

The priority score `(Impact x Customer Value) / max(Effort, 1)` looks authoritative but is completely invented. The numbers are subjective ratings multiplied together. No validation. No calibration. This gives false confidence in the ordering.

### BUG 6: "27 tasks" header but 30 rows

The header says "27 tasks" but the table has rows numbered 1-30. Sloppy copy-paste error.

### BUG 7: No `go mod tidy` steps

After creating the new module and after deleting old files, `go mod tidy` is REQUIRED in both `metaengine/` and `metaengine/bench/`. This is completely absent from the plan. Without it, `go.sum` will be stale and the build will fail.

### BUG 8: No `go.work.sum` update

Adding a module to `go.work` changes `go.work.sum`. This needs regeneration. Missing from the plan.

### BUG 9: 4 deprecated stub files not identified

These 4 files are empty placeholder stubs left over from an earlier extraction:

- `calibration_bench_test.go` (4 lines — "moved to sqliteengine package")
- `json_tax_bench_test.go` (4 lines — same)
- `layout_bench_test.go` (4 lines — same)
- `planner_bench_test.go` (3 lines — same)

They should be deleted as part of cleanup, but the plan never identified them.

### BUG 10: Never used the `todos` tool

The user explicitly said "Split the TODOs into small tasks max 12min each." I showed a table but never called the `todos` tool to create actual trackable items.

---

## e) WHAT WE SHOULD IMPROVE

### Planning quality

1. **Always verify file inventories before planning migration** — I read 9 files but missed the 10th (`bench_filter_test.go`) and 4 stubs. A simple `ls -1 *_test.go` at the start would have caught this.
2. **Always check cross-references before planning deletions** — The `sqlite_helpers_test.go` deletion would have broken the build. A `grep -l` for its functions would have caught this in 2 seconds.
3. **Never invent scoring formulas** — present tasks with impact/effort ratings but don't fabricate precision.
4. **Always include `go mod tidy` and checksum regeneration** in any module-creation plan.
5. **Always check tag availability** before listing modules as go.mod dependencies.
6. **Use the `todos` tool** when the user explicitly asks for TODOs.

### Plan structure

7. **Migration must precede extension** — you can't extend files that haven't been migrated yet. The phase ordering was backwards.
8. **Separate "copy helpers" from "delete originals"** — these are different risk levels and should be in different phases with a verification gate between them.
9. **Identify deprecated/dead files** as a separate cleanup category — don't lump them with active migrations.

---

## f) Up to 50 Things We Should Get Done Next

### Phase 1: Foundation (must be first)

1. Create `metaengine/bench/go.mod` with module path `metaengine/bench/v4`
2. Add `replace` directives for ALL local modules (metaengine, sqliteengine, duckdbengine, pebbleengine)
3. Add `./metaengine/bench` to `go.work` `use` block (alphabetical position)
4. Run `go mod tidy` in `metaengine/bench/`
5. Regenerate `go.work.sum`

### Phase 2: Shared fixtures and helpers

6. Create `metaengine/bench/fixtures_test.go` — copy ALL domain types (OrderCreated through OrderPaid, plus benchItemResult/benchListInput from bench_filter_test.go)
7. Create `metaengine/bench/helpers_test.go` — copy generatePromiseEvents, seedPromiseStore, planPromiseStore, mustApply, measureLatency, setupBenchStore
8. Create `metaengine/bench/sqlite_factory_test.go` — copy newSQLiteEngine/newSQLiteEngineForPath (original stays in metaengine/)
9. Verify fixtures compile: `cd metaengine/bench && go vet ./...`

### Phase 3: Migrate existing benchmarks (copy, don't move yet)

10. Copy + adapt `bench_promise_test.go` (package → bench_test, fix imports)
11. Copy + adapt `bench_fanout_test.go`
12. Copy + adapt `bench_readmix_test.go`
13. Copy + adapt `bench_enginepool_test.go`
14. Copy + adapt `bench_planner_test.go`
15. Copy + adapt `bench_storm_test.go`
16. Copy + adapt `bench_layout_test.go`
17. Copy + adapt `bench_materialize_test.go`
18. Copy + adapt `bench_parity_test.go`
19. Copy + adapt `bench_filter_test.go` (THE ONE I MISSED)
20. Verify all migrated benchmarks compile: `go build -tags "goexperiment.jsonv2" ./...`
21. Smoke-test all migrated benchmarks: `go test -run='^$' -bench=. -benchtime=1x ./...`

### Phase 4: Delete old bench files from metaengine/

22. Delete the 10 bench files from `metaengine/` (9 promise files + bench_filter_test.go)
23. **DO NOT DELETE `sqlite_helpers_test.go`** — 6 non-bench test files depend on it
24. Delete the 4 deprecated stub files (calibration_bench_test.go, json_tax_bench_test.go, layout_bench_test.go, planner_bench_test.go)
25. Run `go mod tidy` in `metaengine/` (may remove unused deps after bench file deletion)
26. Verify `metaengine/` still vets clean: `cd metaengine && go vet -tags "goexperiment.jsonv2" ./...`
27. Verify `metaengine/` tests still pass: `cd metaengine && go test -tags "goexperiment.jsonv2" ./...`

### Phase 5: M4.2 Core — DuckDB columnar extraction (THE WHOLE POINT)

28. Create `metaengine/bench/engine_duckdb_cgo_test.go` (`//go:build cgo`) — newDuckDBEngine() factory
29. Create `metaengine/bench/bench_duckdb_columnar_cgo_test.go` — 3-way comparison benchmark:
    - `WithColumnarLayout()` native columns vs `FilterOnField` json_extract pushdown vs Memory O(N) scan
    - At 1K/10K/100K rows
    - This is THE deliverable that was blocked by the dep boundary
30. Create `metaengine/bench/bench_duckdb_correctness_cgo_test.go` — verify WithColumnarLayout produces identical results to Memory engine
31. Smoke-test M4.2: `go test -tags "cgo goexperiment.jsonv2" -run='^$' -bench='BenchmarkDuckDB_Columnar' -benchtime=1x ./...`

### Phase 6: Extend existing benchmarks with DuckDB

32. Add DuckDB store to parity test (Memory vs SQLite vs DuckDB at 5K events)
33. Add `memorySQLiteDuckDBPool()` to engine pool comparison benchmark
34. Create DuckDB layout planning benchmark (extends bench_layout_test.go with DuckDB LayoutPlanner)
35. Add DuckDB engine to planner PlanLatency benchmark (3-engine Plan() cost)
36. Verify all DuckDB benchmarks: `go test -tags "cgo goexperiment.jsonv2" -run='^$' -bench=. -benchtime=1x ./...`

### Phase 7: Pebble integration (optional, lower priority)

37. Create `metaengine/bench/engine_pebble_test.go` — newPebbleEngine() factory
38. Add Pebble to parity test (4-engine comparison)
39. Add Pebble to engine pool comparison
40. Verify Pebble benchmarks: `go test -tags "goexperiment.jsonv2" -run='^$' -bench='Pebble' -benchtime=1x ./...`

### Phase 8: Infrastructure

41. Add `"metaengine/bench"` to `cmd/api-stability/main.go` modules slice
42. Run api-stability golden regen if needed
43. Update `scripts/bench-matrix.sh` — add `metaengine/bench/...` target with DuckDB columnar benchmarks
44. Update `scripts/bench-all.sh` — add metaengine/bench to module list
45. Update `.github/workflows/benchmarks.yml` — add `metaengine/bench/**` to trigger paths
46. Update `AGENTS.md` — add `metaengine/bench` to Modules list and Test command

### Phase 9: Full verification

47. Run `go build -tags "goexperiment.jsonv2" ./...` across workspace
48. Run `go vet -tags "goexperiment.jsonv2" ./...` for metaengine + metaengine/bench
49. Run `nix fmt` on all new files
50. Run full smoke-test of ALL benchmarks at `-benchtime=1x`

---

## g) Questions I CANNOT Answer Myself

### Q1: Should `metaengine/bench/` be a versioned module (tagged) or stay untagged?

`stack/bench/` is tagged as `stack/bench/v4`. But `metaengine/bench/` is a test-only module that no consumer would ever import. Tagging it adds release overhead (scripts/tag-release.sh, version-sequence management). Not tagging it means `GOWORK=off` consumer builds that somehow reference it will fail — but since it's test-only, no consumer should ever reference it.

**My recommendation:** Tag it for consistency with `stack/bench/`, but I cannot decide this alone because it affects the release process.

### Q2: Should the 4 deprecated stub files be deleted in THIS change or separately?

The stubs (`calibration_bench_test.go`, `json_tax_bench_test.go`, `layout_bench_test.go`, `planner_bench_test.go`) are dead code from a prior extraction. Deleting them is obviously correct, but they're unrelated to the M4.2 work. Should I bundle the cleanup or keep the PR focused?

**Context I cannot resolve:** I don't know if the user prefers focused PRs (M4.2 only) or opportunistic cleanup (delete dead code when you touch the area).

### Q3: Should `bench_filter_test.go`'s types (`benchItemResult`, `benchListInput`) be unified with the promise domain types?

`bench_filter_test.go` defines its OWN domain types (`benchItemResult` with ID/Status/Priority) separate from the promise domain (`OrderView` with ID/CustomerID/Status/TotalCents/Items). Both are test fixtures for benchmarking. Migrating them as-is means two parallel domain models in the same module. Unifying them means rewriting the filter benchmark to use OrderView. I cannot decide this because it changes the benchmark semantics — `benchItemResult.Priority` has no equivalent in `OrderView`.
