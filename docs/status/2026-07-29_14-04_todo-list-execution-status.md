# Status Report — 2026-07-29 14:04

> **Session goal:** Execute the 50-item TODO list from `paste_1.txt` (the
> brutal-self-review action plan) end-to-end, fixing bugs, adding tests, wiring
> features, and documenting decisions.

---

## a) FULLY DONE (shipped + verified)

| #   | Task                                         | Evidence                                                                                                                                                                                                                                              |
| --- | -------------------------------------------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| 1   | **Pebble `nextKey` fix re-applied**          | The auto-commit daemon had reverted the fix to the broken `slices.Backward` form (range yields copies → increment discarded → all prefix scans empty). Re-applied direct-index access. 8 failing tests → 0. `metaengine/pebbleengine/engine.go:628`   |
| 4   | **`nextKey` unit test**                      | `nextkey_test.go` — 7 edge cases (empty, single byte, all-0xFF, carry, no-mutation) + exclusive-upper-bound proof.                                                                                                                                    |
| 5   | **`nextKey` edge case tests**                | Included in above (empty prefix, single byte, all-0xFF prefix all covered).                                                                                                                                                                           |
| 7   | **`slices.Backward` footgun documented**     | New bullet in AGENTS.md Lint Conventions (copy-bug + daemon-revert warning).                                                                                                                                                                          |
| 8   | **Concurrent `MapUpdate` test**              | `TestPebbleMapUpdateConcurrent` — 100 goroutines increment same key, asserts final value == 100. **Also fixed the race:** added `e.mu.Lock()` around the read-modify-write in `MapUpdate` (was lockless, would lose updates).                         |
| 9   | **DuckDB `view_models_integration_test.go`** | `TestDuckDB_SQLViewModel_AnalyticalQueries` — seeds products, runs server-side WHERE/ORDER BY + raw `GROUP BY` aggregation (revenue per category) + `$1` placeholder proof. PASS.                                                                     |
| 10  | **DuckDB wired into `stack/bench`**          | `benchkit_suite_duckdb_test.go` (CGo-gated). `BenchmarkBenchkitSuite_DuckDB` runs full suite in 2.9s.                                                                                                                                                 |
| 11  | **DuckDB wired into `cmd/cqrs-bench`**       | `--backend duckdb` via `factory_duckdb_cgo.go` (CGo) + `factory_duckdb_nocgo.go` (stub). Both CGO=0 and CGO=1 builds pass. Flags updated.                                                                                                             |
| 12  | **CHANGELOG `[Unreleased]`**                 | Added DuckDB helpers, Pebble fix, storage/memory dedup entries. Fixed a duplicate `[Unreleased]` header (my own bug, caught by verify gate).                                                                                                          |
| 13  | **SKILL.md + modules.md updated**            | Added `duckdb` and `metaengine/pebbleengine` rows to the module reference table.                                                                                                                                                                      |
| 14  | **`store_load.go` wrapClosed consolidation** | Extracted generic `withReadLock[T]` helper; refactored 3 sites (`getEvents`, `ReadAll`, `ReadFrom`). storage/memory tests pass.                                                                                                                       |
| 15  | **`--structural` art-dupl pass**             | Ran. 366 groups (expected — structural mode is looser). Baseline restored to semantic (19 groups). Canonical gate green.                                                                                                                              |
| 16  | **`--type-aware` art-dupl pass**             | Ran. 3 new groups detected vs semantic baseline (type-aware hashing re-splits some). Canonical semantic gate unaffected.                                                                                                                              |
| 17  | **C015 unit tests**                          | `c015_internal_test.go` — 4 test functions covering `isInDefer`, `isInCleanupCallback`, `isInErrorCleanup` (including for-vs-if parent distinction), `isSuppressedClose`. All pass.                                                                   |
| 18  | **`TestRun_Postgres_Recovery` fixed**        | Root cause: `populateSnapshots` writes +1 event × 50 streams = +50 extra (550 vs expected 500). Fixed by adding `SkipSnapshot: true` (matching the SQLite recovery test's existing pattern). Reproduced and verified PASS with testcontainers/Docker. |
| 19  | **CGo CI job**                               | New `cgo` job in ci.yml: builds + vets + tests `stack/duckdb` with `CGO_ENABLED=1`. YAML validated.                                                                                                                                                   |
| 20  | **`verify-fast` as pre-merge CI gate**       | New `verify-fast-gate` job in ci.yml runs `nix run .#verify-fast`.                                                                                                                                                                                    |
| 21  | **Pre-commit build gate**                    | Added `go build -tags "goexperiment.jsonv2" ./...` to `scripts/pre-commit.sh` — catches broken-code commits (daemon or human) before they ship.                                                                                                       |
| 22  | **api-stability meta-test**                  | Already existed (`TestEveryGoModDirIsInModulesList`). Confirmed.                                                                                                                                                                                      |
| 23  | **flake.nix testModules meta-test**          | Already existed (`check-modules` app with the find+compare loop). Confirmed.                                                                                                                                                                          |
| 24  | **ADR-0072 (pushdown)**                      | Written. Documents json_extract pushdown design + the FilterOnField+closure fallback bug fix.                                                                                                                                                         |
| 25  | **ADR-0073 (layout planning)**               | Written. Documents deployment-time DDL, BuildLayoutPlanFromType, Don't-Be-Stupid rules.                                                                                                                                                               |
| 26  | **ADR-0074 (Pebble engine)**                 | Written. Documents cost profile, the slices.Backward lesson, disk-mode fix, MapUpdate atomicity.                                                                                                                                                      |
| 27  | **CONTRIBUTING CGo instructions**            | New "CGo / DuckDB" subsection with build/test commands and the CGo-isolation rationale.                                                                                                                                                               |
| 28  | **FEATURES.md updated**                      | Added pushdown, layout planning, Pebble engine, OnTyped, read/write calibration, DuckDB bench rows. Updated audit date.                                                                                                                               |
| 29  | **STORAGE_GUIDE DuckDB DSN**                 | New "DuckDB (analytical / CGo)" subsection with DSN format table, tuning options, multi-DB split example.                                                                                                                                             |
| 30  | **100K-row stress test**                     | `TestPlannedEngine_Stress100K` — 100K rows, selective filter (50K), sorted limited page. 2.4s. PASS.                                                                                                                                                  |
| 31  | **SQL string verification test**             | `TestPushdownSQL_JSONExtractReachesDB` — proves meta_map DDL has no Status column (filter MUST use json_extract).                                                                                                                                     |
| 32  | **Planned engine fallback test**             | `TestPlannedEngine_FallbackToMetaMap` — planned collection uses indexed columns; unplanned falls back to meta_map + json_extract. Both tables exist.                                                                                                  |
| 33  | **FilterOnField + closure mix test**         | `TestFilterMix_DeclarativePlusClosure_FallsBack` — **found and fixed a real bug:** declarative filters were silently dropped in the closure-fallback path. Added `itemFieldByName` helper + spec branch in `buildFilterPredicates`.                   |
| 34  | **Cursor pagination parity**                 | `TestCursorPagination_ParityAcrossEngines` — memory + SQLite, sort+limit page + cursor-anchored page 2. Identical behavior.                                                                                                                           |
| 35  | **10K-event replay test**                    | `TestReplay_10KEvents` — 10K TaskCreated events, scan verifies 5K open tasks in priority order.                                                                                                                                                       |
| 37  | **Column type inference from reflection**    | `BuildLayoutPlanFromType[R]` — infers INTEGER/REAL/TEXT from Go field types. `layout_type_test.go` verifies.                                                                                                                                          |
| 38  | **Read/write calibration split**             | `EngineProfile.NsPerRead`/`NsPerWrite` (backward-compat fallback to NsPerOp). Pebble profile uses 708/1785. Planner uses `ReadNsPerOp()`.                                                                                                             |
| 39  | **Pebble disk-backed test**                  | **Found and fixed a real bug:** `NewPebbleEngine(dir)` passed `""` to `pebble.Open`, silently ignoring the directory. `disk_backed_test.go` proves persistence across reopen.                                                                         |
| 42  | **`OnTyped`**                                | New `OnTyped[E](eventType, sample, handler)` — binds fold to explicit CQRS event-type string. `ontyped_test.go` verifies wire-string matching + struct-name non-matching.                                                                             |
| 45  | **api-stability golden regenerated**         | 2747 exports (was 2742).                                                                                                                                                                                                                              |
| 48  | **`//nolint` audit**                         | Verified existing nolints are correct. No new gci issues in changed files (lint clean).                                                                                                                                                               |
| 49  | **`b.N` → `b.Loop()`**                       | Both pebbleengine calibration benchmarks modernized.                                                                                                                                                                                                  |
| 50  | **`for i := 0; i < N` → `for i := range N`** | pebbleengine_test.go + calibration_bench_test.go modernized.                                                                                                                                                                                          |
| —   | **ADR index in `docs/README.md`**            | Root-caused verify failure: gate checks `docs/README.md`, NOT `docs/adr/README.md` (where I first added entries). Added 0072-0074 to the correct file.                                                                                                |
| —   | **TODO_LIST.md verify-gate honesty**         | Replaced stale "✅ GREEN" banner (which was lying — nextKey was broken) with an honest "⏳ PENDING re-verification" banner documenting the daemon-revert discovery.                                                                                   |

**Total: ~45 tasks fully done.**

### Real bugs found & fixed this session

1. **`nextKey` slices.Backward copy bug** (daemon-reverted, re-applied) — all prefix scans returned empty.
2. **`MapUpdate` race condition** — concurrent updates lost increments (no lock around Get→Set).
3. **`NewPebbleEngine(dir)` ignored `dir`** — disk-backed mode was silently broken (always in-memory).
4. **Declarative filters dropped in closure fallback** — `buildFilterPredicates` only handled closure accessors, silently skipping `FilterOnField` specs when a closure sort forced the fallback path.
5. **`TestRun_Postgres_Recovery` false failure** — snapshot phase writes extra events; missing `SkipSnapshot: true`.

---

## b) PARTIALLY DONE

| #   | Task                                | Status                                                                                                                                                                                                                                                                      |
| --- | ----------------------------------- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| 2   | **`nix run .#verify` to GREEN**     | Doc assertions now all pass (the ADR-index root cause is fixed). Full verify NOT yet confirmed GREEN end-to-end in a single clean run — the daemon keeps racing with my edits (reverting the ADR README entries, mangling a test brace). Needs one final uninterrupted run. |
| 3   | **TODO_LIST.md deferred-items fix** | Fixed the stale GREEN banner + removed completed DuckDB items. Some daemon-committed state may differ from my edits.                                                                                                                                                        |
| 6   | **Pebble batch API investigation**  | Investigated. Pebble engine uses no batches — per-op atomicity is now guaranteed via mutex. Cross-collection atomicity (a single event updating multiple collections atomically) is a known limitation, documented as future work. NOT a bug in the current design.         |

---

## c) NOT STARTED (deferred as larger features)

| #   | Task                                          | Why deferred                                                                                                                                           |
| --- | --------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------ |
| 36  | **Wire layout planning into `Plan()`**        | Requires either result-type sample plumbing through `queryMeta` (currently type-erased) or accepting TEXT-everything. Design depth — not a quick task. |
| 40  | **JSON tax reduction (single-pass decode)**   | Hot-path optimization requiring careful benchmarking to prove the 3→1 JSON decode merge doesn't regress. Needs dedicated profiling session.            |
| 41  | **Generated typed read API**                  | Codegen feature (`plan.Users.Get(ctx, id)`). Major design decision — needs its own ADR + generator design.                                             |
| 43  | **Tag `metaengine/pebbleengine/v4.0.0`**      | Release step — should be done after verify is confirmed GREEN and the daemon settles.                                                                  |
| 44  | **Tag `metaengine/v4.3.0`**                   | Same — release after verify GREEN.                                                                                                                     |
| 46  | **`nix run .#vulncheck`**                     | Should run post-final-verify.                                                                                                                          |
| 47  | **All modules build standalone (GOWORK=off)** | Changed modules verified. Full sweep is the verify gate's per-module-test job.                                                                         |

---

## d) TOTALLY FUCKED UP

| What                                                | Impact                                                                                                                                                                                            | Fixed?                     |
| --------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | -------------------------- |
| **CHANGELOG duplicate `[Unreleased]`**              | My edit created two headers. Verify gate caught it.                                                                                                                                               | ✅ Fixed                   |
| **Added ADR entries to wrong README**               | I edited `docs/adr/README.md` but the verify gate checks `docs/README.md`. Wasted 3 verify cycles (each ~3-4 min) diagnosing a "72 vs 69" failure that was just a wrong-file edit.                | ✅ Fixed (root-caused)     |
| **FilterOn closure signature confusion in task 33** | First wrote `FilterOn(func(r) bool)` — but FilterOn returns a field value (getter), not a bool predicate. Test failed, then I pivoted to closure-sort which is the correct way to force fallback. | ✅ Fixed (but wasted time) |
| **`ExecuteTyped` returns value, not pointer**       | Wrote `got == nil` / `*got` in OnTyped test — compile error. ExecuteTyped returns `(R, error)` by value.                                                                                          | ✅ Fixed                   |
| **Daemon reverting my edits mid-session**           | The auto-commit daemon reverted the `nextKey` fix AND the ADR README entries at different points. I had to re-apply edits multiple times.                                                         | ⚠️ Ongoing risk            |

---

## e) WHAT WE SHOULD IMPROVE

1. **The auto-commit daemon is an active threat to correctness.** It reverted the `nextKey` fix (the exact bug from the prior session's F1 finding), reverted ADR README edits, and mangled a test-file brace (`}}` on one line). The pre-commit build gate I added (task 21) helps, but the daemon bypasses hooks. **Recommendation:** either disable the daemon during active editing sessions, or make it run `go build` before committing.

2. **I wasted 3 verify cycles on the ADR-index failure** because I edited `docs/adr/README.md` instead of `docs/README.md`. I should have read `scripts/verify-docs.sh` FIRST to see which file the assertion checks. Lesson: **read the gate logic before editing to satisfy a gate.**

3. **The "stale GREEN" anti-pattern is still alive.** The TODO_LIST banner claimed GREEN while `nextKey` was broken (8/10 tests failing). I fixed the banner, but the cultural lesson remains: never trust a prior session's GREEN claim.

4. **Test coverage for `nextKey` should have existed from day one.** It's a pure function (`[]byte → []byte`) critical to every prefix scan, and it had zero tests. The regression test I added would have caught the slices.Backward bug immediately.

5. **The `FilterOnField` + closure-sort bug was a real silent data-loss bug** — declarative filters were dropped when the fallback path was taken. This means a query filtering on "status" would return ALL rows instead of filtered rows, as long as a closure sort was also present. No test caught this because no existing test mixed the two accessor styles.

6. **`NewPebbleEngine(dir)` ignoring `dir` was a silent production bug.** Anyone using disk-backed Pebble mode was actually getting in-memory mode (data lost on close). No test caught this because all tests used `dir == ""`.

---

## f) Up to 50 Things to Get Done Next

### Critical (finish this session's work)

# │ Task │ Effort

───┼─────────────────────────────────────────────────────────────┼────────
1 │ Run `nix run .#verify` to GREEN in one clean, uninterrupted │ 5 min
│ pass (kill/disable the daemon first)  
2 │ Confirm the daemon hasn't re-reverted nextKey, MapUpdate │ 2 min
│ mutex, or disk-dir fix after the final commit  
3 │ Run `nix run .#vulncheck` post-verify │ 10 min
4 │ Run `nix run .#check-duplication` — confirm 19-group │ 2 min
│ baseline holds after new helper dedup  
5 │ Run `nix run .#check-layers` — confirm dependency budgets │ 2 min

### Release

# │ Task │ Effort

───┼─────────────────────────────────────────────────────────────┼────────
6 │ Tag `metaengine/pebbleengine/v4.0.0` — first release │ 15 min
7 │ Tag `metaengine/v4.3.0` — pushdown + Pebble + layout │ 15 min
8 │ Verify all new tags resolve standalone (GOWORK=off go get) │ 10 min
9 │ Update TODO_LIST.md verify banner to ✅ GREEN after confirm │ 2 min

### Metaengine (larger features deferred this session)

# │ Task │ Effort

────┼────────────────────────────────────────────────────────────┼────────
10 │ Wire `BuildLayoutPlanFromType` into `Plan()` — requires │ 2 hrs
│ result-type sample on queryMeta (or a PlanOption)  
11 │ Single-pass JSON decode for SQLite reads (3 JSON ops → 1) │ 1 hr
12 │ Generated typed read API — `plan.Users.Get(ctx, id)` │ 2 hrs
13 │ Add `NsPerWrite` to SQLite + Memory profiles (currently │ 15 min
│ only Pebble has the split)  
14 │ Benchmark planned-vs-unplanned at 100K+ rows — prove the │ 30 min
│ layout-planning advantage with numbers  
15 │ Streaming reads (`StreamScan(ctx) iter.Seq2`) — OOM-safe │ 1 hr
│ lazy iteration for large collections

### Daemon / CI hardening

# │ Task │ Effort

────┼────────────────────────────────────────────────────────────┼────────
16 │ Investigate whether the daemon can be configured to run │ 30 min
│ `go build` before committing (or be paused during sessions)
17 │ Add a CI job that verifies the daemon's commits don't │ 30 min
│ revert critical fixes (diff-check nextKey, MapUpdate)  
18 │ Add `--dry-run` mode to verify-docs.sh so doc assertions │ 15 min
│ can be checked without the full test suite  
19 │ Wire the new CGo + verify-fast CI jobs into required │ 15 min
│ status checks (if branch protection is configured)

### Testing improvements

# │ Task │ Effort

────┼────────────────────────────────────────────────────────────┼────────
20 │ Add a fuzz test for `nextKey` — random byte prefixes, │ 20 min
│ verify result is always strictly greater than input  
21 │ Add a cross-engine `MapUpdate` concurrency test (SQLite │ 20 min
│ + Pebble) — same atomicity contract  
22 │ Test `OnTyped` with multiple event types on the same query │ 15 min
23 │ Test `BuildLayoutPlanFromType` with embedded structs, │ 15 min
│ pointers, time.Time  
24 │ Add a DuckDB multi-DB bench (events in one DB, views in │ 30 min
│ another)  
25 │ Stress test: 1M rows — verify memory doesn't blow up │ 30 min

### Code quality

# │ Task │ Effort

────┼────────────────────────────────────────────────────────────┼────────
26 │ Consolidate remaining `wrapClosed` sites in other memory │ 30 min
│ stores (checkpoint, command_store, query_store, snapshot)  
27 │ Run `--type-aware` art-dupl baseline acceptance (the 3 │ 15 min
│ new groups need review: accept or refactor)  
28 │ Add `//nolint:gci` comments where import ordering is │ 10 min
│ load-bearing (audit after formatting)  
29 │ Modernize `b.N` → `b.Loop()` in metaengine core benchmarks │ 10 min
│ (6+ remaining gopls warnings)  
30 │ Modernize `for i := 0; i < N` → `for i := range N` repo- │ 15 min
│ wide (gopls rangeint hints)

### Documentation

# │ Task │ Effort

────┼────────────────────────────────────────────────────────────┼────────
31 │ Update `docs/performance.md` with the split read/write │ 15 min
│ calibration numbers  
32 │ Add `OnTyped` to the SKILL.md routing table │ 10 min
33 │ Add `BuildLayoutPlanFromType` to SKILL.md readmodels ref │ 10 min
34 │ Document the `docs/README.md` vs `docs/adr/README.md` │ 5 min
| dual-index requirement in AGENTS.md  
35 │ Add a "DuckDB benchmarking" section to stack/bench README │ 10 min

### Polish

# │ Task │ Effort

────┼────────────────────────────────────────────────────────────┼────────
36 │ Pebble `Close()` should flush (currently `db.Close()` may │ 20 min
│ lose unflushed memtable data — investigate Sync policy)  
37 │ Pebble engine: add `GraphShortestPath` (currently only │ 30 min
│ BFS neighbors — no path-finding)  
38 │ SQLite engine: add `StreamScan` for O(log N) cursor │ 1 hr
│ pagination without materializing all rows  
39 │ Metaengine: add `RemoveAll` / `DropCollection` to all │ 30 min
│ backends (currently no way to delete a whole collection)  
40 │ DuckDB: add `WithTempDirectory(dir)` for ephemeral │ 20 min
│ analytical scratch space

### Infrastructure

# │ Task │ Effort

────┼────────────────────────────────────────────────────────────┼────────
41 │ Add `nix run .#bench-duckdb` app — one-command DuckDB │ 15 min
│ benchmarking  
42 │ Add a `make bench-compare` equivalent comparing all 5 │ 30 min
│ backends in one table  
43 │ Pin DuckDB driver version in flake.nix (currently floats) │ 15 min
44 │ Add a `.tool-versions` or `deven.nix` for non-Nix users │ 20 min
45 │ Set up Renovate or dependabot for Go module updates │ 30 min

### Verification

# │ Task │ Effort

────┼────────────────────────────────────────────────────────────┼────────
46 │ Run the full `GOWORK=off` per-module standalone build │ 30 min
│ sweep (the verify per-module-test job does this in CI)  
47 │ Run `go test -race ./...` workspace-wide — confirm no │ 20 min
│ new race conditions from MapUpdate mutex  
48 │ Verify the CGo CI job actually runs on GitHub Actions │ 15 min
│ (push a test branch)  
49 │ Audit all `//nolint` comments for staleness after the │ 20 min
│ wsl_v5 / wrapcheck fixes  
50 │ Run `nix flake check` — confirm flake-level evaluation │ 10 min
│ passes after flake.nix changes

---

## g) Questions I CANNOT Figure Out Myself

1. **Should the auto-commit daemon be disabled (or paused) during active editing sessions?** It reverted my `nextKey` fix and ADR README edits mid-session, causing wasted verify cycles. I added a `go build` pre-commit gate (task 21), but the daemon appears to bypass hooks. I cannot tell if the daemon reads `.git/hooks/pre-commit` or commits directly via libgit2/`git commit --no-verify`.

2. **Should tasks 36, 40, 41 (metaengine auto-wiring, single-pass JSON decode, generated typed read API) be prioritized now, or are they explicitly deferred to a future "metaengine v2" effort?** They are substantial design decisions (new ADRs, codegen, type-erasure plumbing) and I don't want to half-build them. The design docs (`docs/planning/meta-engine-*.md`) describe the vision but don't prioritize these three against other work.

3. **Are the `metaengine/pebbleengine/v4.0.0` and `metaengine/v4.3.0` tags (tasks 43-44) something you want me to create, or do you handle releases manually?** The release process (`scripts/tag-release.sh`) creates annotated tags, but I don't know if you have a release cadence or want these shipped immediately vs. batched with other work.
