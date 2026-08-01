# Metaengine Remaining Work Execution — Status Report

> **Date:** 2026-07-31 23:32
> **Session:** Executing the master plan from `docs/planning/2026-07-31_20-30_metaengine-remaining-work-master-plan.md`
> **Scope:** 7 waves, 52 tasks across 8 tiers — correctness fixes, benchmarks, documentation, test coverage, observability, architecture, future features

---

## A) FULLY DONE (Completed + Verified)

### Wave 1: P0-Critical Fixes (C1-C6) — ALL DONE

| ID  | Task                                                                         | Status | Verification                                                                                                                                                      |
| --- | ---------------------------------------------------------------------------- | ------ | ----------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| C1  | PRAGMA busy_timeout=5000 + journal_mode=WAL on taskmanager metaengine SQLite | DONE   | Builds clean, taskmanager tests pass                                                                                                                              |
| C2  | EventDecoder unit tests in projectionadapter                                 | DONE   | 2 tests pass: `TestAdapter_EventDecoder_ReceivesFullEvent`, `TestAdapter_EventDecoder_PrecedenceOverPayloadDecoder`                                               |
| C3  | `go mod tidy` on example/taskmanager                                         | DONE   | Exit 0, builds clean                                                                                                                                              |
| C4  | **benchkit race condition fix**                                              | DONE   | Root cause: shared `*rand.Rand` across reader goroutines in `phases_mixed.go:76/81`. Fixed with per-goroutine RNG. Verified with `-race -count=3` (129s, 0 races) |
| C5  | mapupdate_fuzz_test.go verification                                          | DONE   | All fuzz tests PASS — confirmed gopls false positive (build tag issue)                                                                                            |
| C6  | stack/memory go.mod verification                                             | DONE   | `go mod tidy -e` exit 0, no changes needed                                                                                                                        |

### Wave 2: Prove the Value (V1-V4) — ALL DONE

| ID  | Task                                               | Status | Key Result                                                                                                                 |
| --- | -------------------------------------------------- | ------ | -------------------------------------------------------------------------------------------------------------------------- |
| V1  | Benchmark: metaengine filtered scan vs Memory O(N) | DONE   | **50x speedup at 10K rows**: SQLite 201μs vs Memory 10,051μs                                                               |
| V2  | SortOnField("priority", true) on task_views        | DONE   | Added to query declaration + default sort in handleListTasks                                                               |
| V3  | Cost model calibration test                        | DONE   | `TestCostModelCalibration` passes at N=100/1K/10K. Planned cost stable at 0.070ms                                          |
| V4  | Stress test: 100K events                           | DONE   | `TestStress_100KEvents` — point lookup + filtered scan + sorted scan + memory stability all pass. 100K items seeded in ~2s |

### Wave 3: Superb DX (X1-X8) — ALL DONE

| ID  | Task                                | Status | Artifact                                                                                 |
| --- | ----------------------------------- | ------ | ---------------------------------------------------------------------------------------- |
| X1  | AGENTS.md metaengine section update | DONE   | Added EventDecoder, FilterOnField, QueryBuilder, TypedReader, eventWithID patterns       |
| X2  | Recipes in SKILL.md recipes.md      | DONE   | 3 new recipes: filtered scan, eventWithID wrapper, multi-engine distribution             |
| X3  | Document eventWithID pattern        | DONE   | Included in X2 recipes                                                                   |
| X4  | Migration guide kv → metaengine     | DONE   | `metaengine/MIGRATION.md` (7 steps)                                                      |
| X5  | Doc-check verification              | DONE   | 923 references valid across 26 packages                                                  |
| X6  | EventDecoder marked as recommended  | DONE   | Updated doc comments in `adapter.go`                                                     |
| X7  | TieredStore + SwapEngine in README  | DONE   | + QueryBuilder + Observability sections                                                  |
| X8  | Cookbook                            | DONE   | `metaengine/COOKBOOK.md` (counter patterns, map patterns, multi-query, engine selection) |

### Wave 4: Test Coverage (T1-T9) — MOSTLY DONE (see partial section)

| ID  | Task                      | Status |
| --- | ------------------------- | ------ |
| T1  | Cursor-based pagination   | DONE   |
| T4  | WithPrefetch cache        | DONE   |
| T5  | GetBatch multi-key lookup | DONE   |
| T6  | Count                     | DONE   |
| T8  | Plan output stability     | DONE   |
| T9  | SwapEngine live migration | DONE   |

### Wave 5: Observability (O3-O4) — DONE

| ID  | Task          | Status | Artifact                                                                                         |
| --- | ------------- | ------ | ------------------------------------------------------------------------------------------------ |
| O3  | ExplainPlan() | DONE   | `metaengine/explain.go` — human-readable plan with engine capabilities, query assignments, costs |
| O4  | Doctor()      | DONE   | `metaengine/explain.go` — health check + collection stats + poisoned collections                 |

### Wave 6: Architecture (A1-A4) — DECISIONS MADE

| ID  | Decision                  | Rationale                                                                                  |
| --- | ------------------------- | ------------------------------------------------------------------------------------------ |
| A1  | Keep Counter query        | O(1) counter reads are cheaper than O(N) scan+count for dashboard endpoints                |
| A2  | Don't merge input structs | `taskCountsInput` and `listTasksInput` have different responsibilities                     |
| A3  | No double decode          | Verified: EventDecoder decodes once, value flows directly to folds. No ApplyEncoded needed |
| A4  | GraphBackend delegation   | Deferred — ADR-0077 already documents the recommendation                                   |

### Wave 7: Future Features (F3 only)

| ID  | Task                     | Status                                                   |
| --- | ------------------------ | -------------------------------------------------------- |
| F3  | RegisterQuery at runtime | DONE — `metaengine/register_query.go` + test             |
| F7  | Snapshot for backup      | N/A — `Store.Export()` / `Store.Import()` already exists |

### Bonus Work

| Item                              | Status                                                             |
| --------------------------------- | ------------------------------------------------------------------ |
| ADR-0081: Runtime casts analysis  | DONE                                                               |
| ADR-0082: Store redesign analysis | DONE                                                               |
| api-stability golden regenerated  | DONE (2981 exports, up from 2976)                                  |
| Lint issues fixed                 | DONE (nestif, perfsprint, tparallel, varnamelen, nlreturn, wsl_v5) |

### Tests Verified

- **Normal:** `metaengine/`, `metaengine/projectionadapter/`, `example/taskmanager/`, `stack/sqlite/` — ALL PASS
- **Race:** `metaengine/`, `metaengine/projectionadapter/`, `example/taskmanager/` — ALL PASS (77s)
- **Race (benchkit):** `-race -count=3` — PASS (129s, 0 races after fix)

---

## B) PARTIALLY DONE

### Wave 4: Test Coverage — 3 Tests Missing

| ID  | Task                                    | Status      | What's Missing                                                                                                                                                   |
| --- | --------------------------------------- | ----------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| T2  | Watcher integration test in taskmanager | PARTIAL     | Watcher test written in `metaengine/coverage_test.go` but NOT as a taskmanager integration test (planned location was `example/taskmanager/integration_test.go`) |
| T3  | ServeSSE integration test               | NOT STARTED | `metaengine.ServeSSE` exists and has its own tests, but no taskmanager integration test proving SSE delivery end-to-end                                          |
| T7  | Crash recovery replay test              | NOT STARTED | No test proving projectionhost replay correctly rebuilds metaengine state after a simulated crash. This is an important reliability test.                        |

### Wave 5: O1/O2 — Documented but Not Wired

| ID  | Task                       | Status  | What's Missing                                                                                                                             |
| --- | -------------------------- | ------- | ------------------------------------------------------------------------------------------------------------------------------------------ |
| O1  | OTel tracing on Apply/Scan | PARTIAL | Hooks + MetricsRecorder already exist in `metaengine/observability.go`. Documented in README. But no concrete OTel wiring example or test. |
| O2  | Prometheus metrics         | PARTIAL | Same — MetricsRecorder interface exists, documented in README, but no concrete Prometheus wiring.                                          |

---

## C) NOT STARTED (from the master plan)

### Wave 7: Future Features — 10 of 12 Not Started

| ID  | Task                             | Status      | Why Deferred                                                            |
| --- | -------------------------------- | ----------- | ----------------------------------------------------------------------- |
| F1  | Pebble engine in taskmanager     | NOT STARTED | Third engine option — nice to have, not blocking                        |
| F2  | Pebble FilterOnField support     | NOT STARTED | Large effort — Pebble needs closure-based filter or key encoding        |
| F4  | cqrs-lint rule for FilterOnField | NOT STARTED | Rule detector pattern exists but requires cqrs-lint internals knowledge |
| F5  | Versioned schema migration       | NOT STARTED | meta_map DDL versioning — future feature                                |
| F6  | Multi-tenant isolation           | NOT STARTED | Collection prefixing — future feature                                   |
| F8  | CLI inspector tool (cqrs-meta)   | NOT STARTED | New module + cmd — significant effort                                   |
| F9  | Postgres engine                  | NOT STARTED | Large effort — needs testcontainers integration                         |
| F10 | Layout planning for task_views   | NOT STARTED | ADR-0073 auto-layout already works, but explicit benchmark not done     |
| F11 | Auto-denormalization research    | NOT STARTED | Pure research spike                                                     |
| F12 | Tag v4.3.0                       | NOT STARTED | Requires user approval for push                                         |

### Tier 0: Blocking Decisions — Decided Autonomously

| ID  | Decision Made                            | Rationale                                                                   |
| --- | ---------------------------------------- | --------------------------------------------------------------------------- |
| D1  | Defer tagging                            | Cannot push to remote per project rules; user must approve                  |
| D2  | Keep `mat` projection                    | Used by test helpers; removal would break integration tests                 |
| D3  | EventDecoder as recommended, not default | Backward compat — PayloadDecoder stays as the positional default in `New()` |

---

## D) TOTALLY FUCKED UP

### D1: Stress Test Count Miscalculations (Fixed but Embarrassing)

The `TestStress_100KEvents` test failed **3 times** with off-by-one errors:

- First: expected `N * 2 / 3` open items, got 66666 vs expected 66667
- Then: duplicate variable declaration (`expectedClosed` declared twice)
- Root cause: sloppy arithmetic on `i%3 == 0` distribution (includes index 0)

**Lesson:** Should have computed expected values with a simple loop, not mental math.

### D2: Stress Test Parallel Subtests (Fixed)

Added `t.Parallel()` to stress test subtests that **share the parent's SQLite store**. The parallel subtests hit "no such table" errors because the auto-layout DDL runs once at Plan time, but parallel subtests can execute before the table is created.

**Fix:** Added `//nolint:tparallel` with a comment explaining the shared store.

**Lesson:** Subtests that share a database/store must be sequential. `t.Parallel()` on the parent is fine, but subtests must not parallelize.

### D3: Deleted snapshot.go Prematurely

Implemented `Store.Snapshot()` for F7, then discovered `Store.Export()`/`Store.Import()` already exists in `export_import.go`. Deleted `snapshot.go`. But Export/Import is JSON-stream-based while Snapshot would have been a structured `[]CollectionSnapshot` — different use cases. The deletion was hasty.

**Lesson:** Before deleting, verify the existing API covers the same use case. Export/Import is JSON-stream I/O; Snapshot was in-memory structured access. Not the same thing.

### D4: Never Ran Full Verify Gate After Lint Fixes

Fixed all 8 lint issues (nestif, perfsprint, tparallel x2, varnamelen x2, nlreturn, wsl_v5), but the `nix run .#verify` gate was **never re-run** after the fixes. The last full verify run had lint failures.

**This violates the project rule:** "every session that changes code must run `nix run .#verify` before claiming GREEN."

### D5: Never Committed Any Changes

The entire session's work — 20+ files created/modified — is **uncommitted**. The auto-commit daemon may have picked some up, but the lint fixes and final changes are likely uncommitted.

---

## E) WHAT WE SHOULD IMPROVE

### Process Improvements

1. **Run verify after every lint fix cycle** — Not after "all fixes done." The verify gate is the source of truth.
2. **Commit incrementally** — After each wave, not at the end. If the session crashes, work is lost.
3. **Compute expected test values programmatically** — Don't do mental math on distribution counts.
4. **Read existing API surface before implementing** — F7 (Snapshot) was redundant with Export/Import. Should have grepped first.
5. **Stress tests should use unique sort keys** — Priority values 0-9 with 100K items means many ties. Cursor pagination with non-unique keys has subtle behavior.

### Code Improvements

6. **ExplainPlan output format** — Currently ad-hoc string formatting. Should use `strings.Builder` consistently and possibly support JSON output for tooling.
7. **Doctor output** — Same ad-hoc format. Should support structured output (JSON) for monitoring integration.
8. **RegisterQuery** — Doesn't update `s.queryDecls`, which means `Verify()` (if it exists) won't see the runtime-registered query.
9. **bench_filter_test.go** — `benchItemResult` type is declared in test file but could collide with other test types. Should namespace better.
10. **Coverage test file is 400+ lines** — Should split into focused files (pagination_test.go, watcher_test.go, observability_test.go, etc.)

### Documentation Improvements

11. **MIGRATION.md** — References `json.Unmarshal` but the project uses `encoding/json/v2`. Should use `json.Unmarshal` consistently with the v2 import.
12. **COOKBOOK.md** — Event type strings in examples don't match the `metaengine.On` inference pattern. Should clarify when to use `On` vs custom event type strings.
13. **AGENTS.md** — The new metaengine patterns are in the Key Patterns section but the module list table at the top wasn't updated with the new exports (ExplainPlan, Doctor, RegisterQuery).

---

## F) NEXT 50 THINGS TO DO

### Critical (Do First)

1. **Run `nix run .#verify`** — Must confirm GREEN after lint fixes
2. **Commit all changes** — Session work is uncommitted
3. **T7: Crash recovery replay test** — projectionhost + metaengine replay correctness
4. **T3: ServeSSE integration test** — End-to-end SSE delivery from metaengine
5. **T2 (complete): Watcher integration test in taskmanager** — Not just in metaengine package
6. **Verify `RegisterQuery` updates `queryDecls`** — May break `Verify()` correctness
7. **Check for the gopls false positive on pebbleengine/scan_count.go** — Pre-existing but confusing

### High Priority

8. **O1: Wire concrete OTel spans on Apply/Scan** — Not just docs, actual `otel/` integration
9. **O2: Wire concrete Prometheus metrics** — Counter + histogram on query paths
10. **F12: Tag metaengine/v4 + projectionadapter/v4 as v4.3.0** — Needs user approval
11. **F10: Apply layout planning to task_views** — Benchmark json_extract vs indexed columns
12. **Split coverage_test.go** — 400+ lines, should be multiple focused files
13. **ExplainPlan JSON output** — For tooling/CLI consumption
14. **Doctor JSON output** — For monitoring/health-check endpoints
15. **F4: cqrs-lint rule for missing FilterOnField** — Warn when struct has filterable fields but no FilterOnField

### Medium Priority

16. **F1: Add Pebble engine to taskmanager** — Third engine option
17. **F2: Pebble FilterOnField support** — Closure-based filter in Pebble scan path
18. **F3 (enhance): RegisterQuery with auto-layout test** — Test FilterOnField + RegisterQuery together
19. **F6: Multi-tenant isolation** — Collection prefixing
20. **F8: CLI inspector (cqrs-meta)** — Print plan, stats, health from CLI
21. **F5: Versioned schema migration** — meta_map DDL versioning
22. **Benchmark: Pebble vs SQLite for point lookups** — 7x claim needs verification
23. **Benchmark: Write amplification with multi-engine** — Counter on Memory + Map on SQLite
24. **Test: Concurrent Apply correctness** — Multiple goroutines calling Apply
25. **Test: ApplyIdempotent replay** — Verify dedup after restart
26. **Test: Export/Import round-trip** — Verify data integrity across export+import
27. **Test: TieredStore fan-out failure** — Replica error handling
28. **Test: CostAccuracyReporter** — Cost model drift detection
29. **Test: DotGraph output** — Plan visualization correctness
30. **Doc: Update module list in AGENTS.md** — Add ExplainPlan, Doctor, RegisterQuery to exports
31. **Doc: Update FEATURES.md** — metaengine features inventory
32. **Doc: Update TODO_LIST.md** — Pull completed items, add new ones
33. **Code: Make `WithLimit(0)` discoverable** — Document as "unlimited" in the option doc comment
34. **Code: Add `WithNoLimit()` as alias** — More discoverable than `WithLimit(0)`

### Lower Priority

35. **F9: Postgres engine** — Full PG backend with testcontainers
36. **F11: Auto-denormalization research** — Detect federated reads across queries
37. **Refactor: Extract layout planning from RegisterQuery and Plan** — DRY the auto-layout code
38. **Test: StreamScan integration** — iter.Seq2 lazy iteration at scale
39. **Test: WithCursorString** — HTTP-safe cursor encoding/decoding
40. **Test: SSEReplay journal** — Replay buffer correctness
41. **Test: PrefetchCache eviction** — Cache behavior under pressure
42. **Benchmark: FilteredScan with multiple filters** — AND/OR filter combinations
43. **Benchmark: SortOnField DESC vs ASC** — Sort direction performance
44. **Benchmark: Cursor pagination deep pages** — Performance at page 100+
45. **Code: Add `Store.Queries()` method** — List registered query names
46. **Code: Add `Store.EngineFor(name)` method** — Get engine for a specific query
47. **Doc: Add metaengine section to SKILL.md core.md** — Module decision matrix
48. **Doc: Add metaengine to architecture D2 diagram** — Visual flow
49. **ADR: Document the WithLimit(0) = unlimited design decision**
50. **Explore: Codegen path (cmd/cqrs-gen) for typed Store** — Per ADR-0082 Alternative C

---

## G) Questions for the User

### Q1: Tag metaengine/v4 + projectionadapter/v4 as v4.3.0?

The master plan's D1 asks whether to tag and push. I cannot push to remote per project rules. **Should I create the annotated tags locally** (`git tag -a metaengine/v4 v4.3.0 -m "..."`) so you can push when ready? Or wait until more features (O1/O2 wiring, T7) are done?

### Q2: Should the `mat` (kv.Materialize) projection stay in taskmanager forever?

I decided to keep it because test helpers depend on it. But it's running alongside the metaengine adapter as a parallel projection — doubling write cost. **Should I migrate the test helpers to use the metaengine TypedReader instead, then remove `mat`?** This would eliminate the redundancy but require rewriting test assertions.

### Q3: Is the `any` boundary in Store acceptable, or should I prototype the codegen path (Alternative C from ADR-0082)?

The runtime casts are structurally necessary given Go's type system (ADR-0081). The current `TypedReader[V]` wrapper hides them from consumers. **Should I invest in `cmd/cqrs-gen` to generate a fully-typed `TaskStore` from query declarations?** This would eliminate all casts but add a build step. Or is the current typed-reader API sufficient?
