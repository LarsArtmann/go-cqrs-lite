# Session Status: Meta-Engine Build — On API Proven, Gaps Identified

**Date:** 2026-07-23 22:27
**Session focus:** Aligned metaengine API with Event-Query Model design doc, proved all 5 ADT types work end-to-end, then compared against `meta-engine-project-definition.md` to identify the gap between the working prototype and the research vision.

---

## Session Journey (Chronological)

1. **Inherited broken build from prior session** — 4 compilation errors from per-query architecture rewrite. `encoded.go` referenced removed `s.models`. `planner.go` referenced removed `queryRuntime` fields.

2. **Plan + status committed** — Wrote comprehensive build plan at `docs/planning/2026-07-23_22-17_meta-engine-build-plan.md` with Pareto breakdown and mermaid execution graph. Committed and pushed.

3. **Fixed build in ~5 min** — Deleted dead `readmodel.go` (67 lines). Fixed `encoded.go` to iterate `s.queries`. Fixed `planner.go` to use `config QueryConfig` + `keyType` instead of removed fields.

4. **Rewrote all tests for On API** — Two test files (729 lines total) covering all 5 ADT types (Map, Set, Counter, Graph, SortedMap), 3 regression tests (sorted pagination, concurrent FoldUpdate, stable order), ApplyEncoded, plan diagnostics, and On classification (7 handler shapes).

5. **Fixed 2 test failures** — Set queries returned nil because `QueryKeyType()` only checked `FoldInsert`, not `FoldSet`. Graph queries returned empty because `extractKeyValueByType` needs a fallback when there's no insert/set fold. Both fixed.

6. **All 15 tests pass with -race** — Build clean, vet clean, 15 tests green.

7. **User asked to compare against `meta-engine-project-definition.md`** — Read the full project definition. Identified that what we built is Phase 1 (declaration API) of a 5-phase, 11-week research project. The planner is a greedy stub, not the formal cost-based optimizer the project definition envisions.

---

## a) FULLY DONE

| #   | What                                     | Files                                       | Evidence                                                                        |
| --- | ---------------------------------------- | ------------------------------------------- | ------------------------------------------------------------------------------- |
| 1   | **Build compiles + vets clean**          | All `.go` files                             | `go build ./...` + `go vet ./...` pass                                          |
| 2   | **All 15 tests pass with -race**         | `metaengine_test.go`, `correctness_test.go` | 1.012s, 0 race conditions                                                       |
| 3   | **On API with all 7 handler patterns**   | `fold.go`                                   | `TestOnClassification` verifies: insert, update, set, count, edge, remove, skip |
| 4   | **Remove[V](<>) sentinel**               | `fold.go`                                   | Type-safe, key auto-derived by type matching                                    |
| 5   | **Skip sentinel**                        | `types.go`, `fold.go`                       | `Skip struct{}`, classified as FoldSkip                                         |
| 6   | **Auto key derivation**                  | `fold.go`                                   | Scans event struct for fields matching insert fold's key type K                 |
| 7   | **Per-query projections**                | `planner.go`, `store.go`                    | Each query owns folds/ADT/collection. No ReadModel, no model dedup.             |
| 8   | **FilterOn typed closures**              | `query.go`                                  | `FilterOn(func(r R) T { return r.Field })` — no strings                         |
| 9   | **SortOn typed closures**                | `query.go`                                  | `SortOn(func(r R) T { return r.Field })` — no strings                           |
| 10  | **Pagination from input struct**         | `reflect.go`                                | Detects `Limit int` + `After *Cursor` fields by type                            |
| 11  | **Collection results by field shape**    | `reflect.go`                                | Detects `[]T` + `*Cursor` fields, no `Page[T]` wrapper                          |
| 12  | **Map ADT end-to-end**                   | Tests                                       | FindUser: insert + update (suspend) + remove (delete)                           |
| 13  | **Set ADT end-to-end**                   | Tests                                       | CheckEmail: membership test                                                     |
| 14  | **Counter ADT end-to-end**               | Tests                                       | CountByStatus: active/suspended/deleted transitions                             |
| 15  | **Graph ADT end-to-end**                 | Tests                                       | FriendsOf: 2-hop traversal                                                      |
| 16  | **SortedMap (filtered scan) end-to-end** | Tests                                       | ListByStatus: FilterOn + SortOn + pagination + HasMore + Next cursor            |
| 17  | **Regression: sorted pagination**        | Tests                                       | 10 items, limit=5, verifies correct 5 by sort order                             |
| 18  | **Regression: concurrent FoldUpdate**    | Tests                                       | 100 concurrent +1 increments, expects Total=100 (MapUpdater atomicity)          |
| 19  | **Regression: stable order**             | Tests                                       | 10 items with equal scores, 5 repeated scans, all identical order               |
| 20  | **ApplyEncoded (JSON payloads)**         | `encoded.go`                                | Decode JSON → fold's event type → Apply                                         |
| 21  | **Plan diagnostics**                     | `planner.go`                                | Degradation warnings (graph on scan engine), write amplification warnings       |
| 22  | **README updated**                       | `README.md`                                 | Three-role model, On API, all 5 examples, FilterOn/SortOn, pagination           |
| 23  | **nix fmt applied**                      | All files                                   | 2 files reformatted                                                             |
| 24  | **Committed + pushed**                   | git                                         | Commits `40b6eaa6` through `046d0a4d`                                           |

## b) PARTIALLY DONE

| #   | What                               | Current State                                                      | What's Missing                                                                                                                                                               |
| --- | ---------------------------------- | ------------------------------------------------------------------ | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| 1   | **Planner (greedy stub)**          | Picks cheapest-complexity engine per query                         | No formal cost model. No scale thresholds. No hardware budget constraints. No Volume/LatencyBudget enforcement.                                                              |
| 2   | **Engine interfaces**              | 7 backends: Map, MapUpdater, Scan, Set, Counter, Graph, Engine     | `MemoryEngine` is public — should be test infrastructure. No plugin registration pattern (`Register()`/`Open(cfg)`).                                                         |
| 3   | **FilterOn/SortOn**                | Works at runtime via closure invocation                            | Can't translate closures to SQL WHERE/ORDER BY (Go can't inspect closure bodies). Open design decision for SQL pushdown.                                                     |
| 4   | **Write amplification diagnostic** | Counts per-event-type projection updates, warns if >3              | No auto-denormalization detection (the project definition envisions it)                                                                                                      |
| 5   | **Key extraction**                 | By type matching for fold update/remove + query input point lookup | `extractFirstDomainField` fallback is fragile for graph traversal (just takes first non-meta field)                                                                          |
| 6   | **Code quality**                   | Build + vet clean                                                  | `detectPagination` has duplicate logic. `queryMeta` interface has vestigial methods (`QueryFilters`, `QuerySortField`). `engine.go` has `reflect` import that may be unused. |

## c) NOT STARTED

| #   | What                                                                                      | From Source                              |
| --- | ----------------------------------------------------------------------------------------- | ---------------------------------------- |
| 1   | **Formal cost model** `Cost(query, view, engine, volume)`                                 | Project definition Phase 2               |
| 2   | **Scale-dependent structure selection** (Bloom vs hash at N>100K, B-tree vs sorted slice) | Project definition Phase 2               |
| 3   | **Auto-denormalization** (cross-engine query detection)                                   | Project definition Phase 2               |
| 4   | **Index DDL generation** (CREATE TABLE, CREATE INDEX from fold types)                     | Project definition Phase 2               |
| 5   | **Generated typed read API** (`store.Users.Get(id)` instead of `ExecuteTyped[Q,R]`)       | Project definition Phase 3               |
| 6   | **Streaming** (`iter.Seq2[T, error]` for unbounded results)                               | Project definition Phase 3 / Decision 2  |
| 7   | **Query expression tree** (`query.Or(query.Eq(...), ...)`) — only AND via FilterOn        | Project definition Problem 6             |
| 8   | **Engine plugin registration** (database/sql `Register()` pattern)                        | Project definition Phase 4               |
| 9   | **Real SQLite engine** (wrapping `storage/view/SQLViewStore`)                             | Project definition Phase 4               |
| 10  | **Real Pebble engine** (wrapping `storage/pebble/`)                                       | Project definition Phase 4               |
| 11  | **Benchmark suite** (planner-chosen vs hand-tuned)                                        | Project definition Phase 5               |
| 12  | **Hot-reload** (re-plannable planner, background replay, atomic cutover)                  | event-query-model.md Section 14          |
| 13  | **Cursor serialization** (`String()` / `ParseCursor` for HTTP)                            | Needed for real pagination               |
| 14  | **Metadata parameter in folds** (`func(e, md Metadata)`)                                  | event-query-model.md Section 8           |
| 15  | **Multimap + Log ADTs**                                                                   | Project definition: 7 ADTs, we have 5    |
| 16  | **Formal model paper**                                                                    | Project definition Phase 5               |
| 17  | **D2/Mermaid plan visualizer**                                                            | Prior session item                       |
| 18  | **Property-based testing** with `rapid`                                                   | Prior session item                       |
| 19  | **Integration with `event.Event`**                                                        | Blocked by go.sum checksum issues        |
| 20  | **Integration with `projection.Projection` / `projectionhost.Host`**                      | Adapter pattern documented but not wired |

## d) TOTALLY FUCKED UP

| #   | What                                                       | Severity | Details                                                                                                                                                                                                                                             |
| --- | ---------------------------------------------------------- | -------- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| 1   | **Lost the plot mid-session**                              | HIGH     | Spent significant effort on `MemoryEngine.MapScan` internals (filter predicate plumbing, sort comparator wiring) when the meta-engine is a PLANNER, not a database. User corrected: "did you lose the plot?" and "Why MemoryEngine?"                |
| 2   | **`FilterOn(field string)` — stringly-typed anti-pattern** | HIGH     | First implementation took a string field name. Design doc explicitly rejects this. User caught immediately. Fixed to typed closures.                                                                                                                |
| 3   | **Hardcoded domain types in mental model**                 | MEDIUM   | When discussing `ListByStatus{Status string}`, I was thinking in terms of User/Status instead of generic Event→Fold→Query→Result. User: "You know this is kinda a runtime engine? You plan it not to hardcode User or Status right?"                |
| 4   | **Committed broken code multiple times**                   | MEDIUM   | Commits `593764cc` through `c4a92d66` contained code that didn't compile. The build was broken across the entire per-query architecture rewrite until I fixed it in Phase 1 of the build plan. Same mistake documented in the prior session status. |
| 5   | **Auto-commits during writing**                            | LOW      | BuildFlow or hooks auto-committed file changes with generic messages like "refactor(metaengine): improve query reflection and storage engine architecture" while I was still mid-edit. These commits are noisy and the messages are meaningless.    |

## e) WHAT WE SHOULD IMPROVE

### Architecture

1. **Make MemoryEngine unexported** — It's test infrastructure, not public API. An operator never imports it. Should be `memoryEngine` or in a `metaengine/metest` subpackage.
2. **Clean up `queryMeta` interface** — Has vestigial methods (`QueryFilters`, `QuerySortField`, `QueryIsPaginated`) from the old FieldPath-based approach. With typed closures, filters/sort live in `QueryConfig` and are accessed via `QueryConfig()`, not individual getters.
3. **Clean up `detectPagination`** — Has duplicate logic (reflectFields check + reflect.Type check) and a dead `_ = limitType` line.
4. **Remove unused `reflect` import from `engine.go`** — `matchesFilters` was deleted but the import may linger. Verify after build.
5. **Consider the Graph key extraction problem** — `extractFirstDomainField` is a fragile fallback. Graph queries have no insert fold (only Edge folds), so keyType is nil. The fallback grabs the first non-meta field. This works for `FriendsOf{ID UserID, Depth int}` but could break with complex inputs.

### Planner

6. **Volume/LatencyBudget are accepted but ignored** — The planner takes `Volume(n)` and `WithLatencyBudget(ms)` but never uses them for engine selection. Either implement or remove (YAGNI).
7. **No index planning** — The project definition envisions the planner generating DDL (CREATE TABLE, CREATE INDEX). Currently the planner just assigns engines and reports complexity.
8. **No scale-dependent selection** — No threshold tables (Bloom at N>100K, sorted slice at N<10K). The MemoryEngine always uses hash maps regardless of cardinality.

### Design Decisions to Resolve

9. **FilterOn closure → SQL pushdown** — Go reflection can't inspect closure bodies. Memory engine uses runtime invocation. SQL engine needs `WHERE status = ?`. Options: code generation, named field descriptors, or accept that memory and SQL engines use different mechanisms. This is Open Decision 1 in the design doc.
10. **Engine interface shape** — Current: operation-level (`MapSet/MapGet/MapDelete`). Alternative: capability-level (`Supports map[ADT]Complexity`). Which is the right abstraction for real engines?
11. **Generated typed read API vs `ExecuteTyped[Q,R]`** — The project definition envisions `store.Users.Get(id)`. Currently `ExecuteTyped[FindUser, FindUserResult](ctx, store, FindUser{ID: id})`. Code generation vs runtime generics?

### Process

12. **Don't write 8 files without compiling** — This was done in the prior session AND this session. Must compile after every file change.
13. **Disable or configure auto-commit hooks** — Generic auto-commits with messages like "refactor(metaengine): improve query reflection" are noise. They make the git history unreadable.

## f) Up to 50 Things to Do Next

### P0 — Code Quality (quick wins, must do)

| #   | Task                                                                              | Effort | Impact |
| --- | --------------------------------------------------------------------------------- | ------ | ------ |
| 1   | Make `MemoryEngine` unexported                                                    | 15min  | HIGH   |
| 2   | Clean up `queryMeta` interface — remove vestigial methods                         | 10min  | MEDIUM |
| 3   | Clean up `detectPagination` — remove duplicate logic                              | 8min   | LOW    |
| 4   | Remove unused `reflect` import from `engine.go`                                   | 5min   | LOW    |
| 5   | Remove unused `FieldPath`/`matchFilterFields`/`detectSortField` from `reflect.go` | 10min  | MEDIUM |
| 6   | Add `-cover` to test run, identify uncovered branches                             | 5min   | MEDIUM |

### P1 — Planner Foundation

| #   | Task                                                           | Effort | Impact   |
| --- | -------------------------------------------------------------- | ------ | -------- |
| 7   | Define `Cost(query, view, engine, volume)` cost function       | 1h+    | CRITICAL |
| 8   | Implement Volume-based engine selection (use the hint!)        | 30min  | HIGH     |
| 9   | Implement LatencyBudget enforcement                            | 30min  | HIGH     |
| 10  | Add scale threshold tables as Go data (Bloom at N>100K, etc.)  | 2h     | HIGH     |
| 11  | Implement scale-dependent structure selection                  | 1h     | HIGH     |
| 12  | Add write amplification budget (reject plans exceeding budget) | 30min  | MEDIUM   |

### P2 — Engine Design

| #   | Task                                                                 | Effort         | Impact   |
| --- | -------------------------------------------------------------------- | -------------- | -------- |
| 13  | Design engine plugin registration pattern (`Register()`/`Open(cfg)`) | 1h             | HIGH     |
| 14  | Decide: operation-level vs capability-level engine interfaces        | 30min thinking | CRITICAL |
| 15  | Design how SQL engine receives filter/sort natively                  | 1h+            | HIGH     |
| 16  | Design DDL generation from fold types (CREATE TABLE, CREATE INDEX)   | 2h             | HIGH     |
| 17  | Implement SQLite engine wrapping `storage/view/SQLViewStore`         | 2h+            | HIGH     |
| 18  | Implement Pebble engine wrapping `storage/pebble/`                   | 1h+            | MEDIUM   |

### P3 — Read API

| #   | Task                                                           | Effort      | Impact |
| --- | -------------------------------------------------------------- | ----------- | ------ |
| 19  | Design generated typed read API (`store.Users.Get(id)`)        | 1h thinking | HIGH   |
| 20  | Implement streaming (`iter.Seq2[T, error]`)                    | 30min       | MEDIUM |
| 21  | Implement query expression tree (`query.Or`, `query.Eq`, etc.) | 2h          | MEDIUM |
| 22  | Implement cursor serialization (`String()` / `ParseCursor`)    | 20min       | HIGH   |

### P4 — Missing ADTs

| #   | Task                                           | Effort | Impact |
| --- | ---------------------------------------------- | ------ | ------ |
| 23  | Implement Multimap ADT (one key → many values) | 1h     | MEDIUM |
| 24  | Implement Log ADT (append-only, time-ordered)  | 30min  | MEDIUM |

### P5 — Denormalization

| #   | Task                                      | Effort | Impact |
| --- | ----------------------------------------- | ------ | ------ |
| 25  | Detect cross-projection query patterns    | 1h     | MEDIUM |
| 26  | Auto-generate denormalization projections | 2h     | MEDIUM |
| 27  | Write-amplification tradeoff analysis     | 1h     | LOW    |

### P6 — Integration

| #   | Task                                               | Effort | Impact |
| --- | -------------------------------------------------- | ------ | ------ |
| 28  | Resolve go.sum checksum issues for `event/` import | 30min  | HIGH   |
| 29  | Add `Apply(event.Event)` method                    | 10min  | HIGH   |
| 30  | Add `projection.Projection` implementation         | 15min  | HIGH   |
| 31  | Wire into `projectionhost.Host`                    | 30min  | HIGH   |
| 32  | Wire into `stack.Bundle`                           | 30min  | MEDIUM |
| 33  | Update AGENTS.md module list with metaengine       | 5min   | MEDIUM |

### P7 — Testing & Validation

| #   | Task                                             | Effort | Impact |
| --- | ------------------------------------------------ | ------ | ------ |
| 34  | Property-based testing with `rapid`              | 30min  | MEDIUM |
| 35  | Multi-engine plan test (memory + SQLite profile) | 15min  | MEDIUM |
| 36  | Test all `compareValue` numeric branches         | 10min  | LOW    |
| 37  | Benchmark suite (planned vs hand-tuned)          | 2h     | MEDIUM |
| 38  | Scale threshold empirical validation             | 3h     | LOW    |

### P8 — Advanced

| #   | Task                                | Effort | Impact |
| --- | ----------------------------------- | ------ | ------ |
| 39  | Hot-reload (re-plannable planner)   | 2h+    | FUTURE |
| 40  | Background replay + atomic cutover  | 2h+    | FUTURE |
| 41  | Metadata parameter in fold handlers | 45min  | HIGH   |
| 42  | Domain result types in examples     | 15min  | LOW    |
| 43  | D2/Mermaid plan visualizer          | 30min  | LOW    |

### P9 — Documentation & Research

| #   | Task                                                          | Effort | Impact   |
| --- | ------------------------------------------------------------- | ------ | -------- |
| 44  | Write formal model (ILP formulation, tractability argument)   | 1 week | RESEARCH |
| 45  | Document engine interface contract                            | 15min  | MEDIUM   |
| 46  | Document three-role model clearly                             | 15min  | MEDIUM   |
| 47  | Consolidate 7 overlapping planning docs into narrative        | 1h     | MEDIUM   |
| 48  | Write case study (taskmanager on SQLite-only vs multi-engine) | 2h     | LOW      |
| 49  | Fix broken LaTeX in `meta-engine-project-definition.md`       | 15min  | LOW      |
| 50  | Write the meta-engine name decision                           | 5min   | LOW      |

---

## g) Questions I CANNOT Answer Myself

**Q1: ~~Should the meta-engine stay as a monorepo module or become a separate project?~~**
**ANSWERED:** Monorepo submodule for now. Will extract to a separate project when we reach a first stable version. The `meta-engine-project-definition.md` separate-project plan remains the long-term goal but is deferred until stability.

**Q2: How should FilterOn typed closures translate to SQL pushdown?**
`FilterOn(func(r FindUserResult) string { return r.Status })` works at runtime for in-memory engines (call closure on each item). But for SQL: we need `WHERE status = ?`. Go reflection cannot extract "Status" from a closure body. Options: (a) accept memory uses closures, SQL uses a different declaration mechanism, (b) code generation to extract field paths at build time, (c) the `Field()` named-descriptor approach from the design doc's Decision 1, (d) something else. This blocks building a real SQL engine.

**Q3: Should the planner output executable handlers or just a diagnostic plan?**
Currently `Plan()` returns a `*Store` that holds runtime state (engines, queries, byInputType map) AND a diagnostic `PlanResult`. The project definition envisions the planner generating projection handlers (event → engine writes) and typed read API (store.Users.Get(id)). Is the current `Store.Apply()` + `ExecuteTyped[Q,R]` the right shape, or should the planner output code/handlers that get wired into `projectionhost.Host`?

---

## Resolution (2026-07-26)

All three open questions have been resolved:

- **Q1:** Confirmed as monorepo submodule. `metaengine/v4.1.1` tagged and pushed.
- **Q2 (FilterOn SQL pushdown):** Phase 1 keeps in-memory closures + `PushdownScan`
  interface seam. Phase 2 declarative `FilterSpec`/`SortSpec` deferred until a
  production consumer needs SQL filter/sort pushdown (ADR-0063).
- **Q3 (planner output shape):** `Store.Apply()` + `ExecuteTyped[Q,R]` IS the right
  shape. The `metaengine/projectionadapter/` module wraps the Store as a
  `projection.Projection` for `projectionhost.Host` integration (ADR-0062).

**Stale claims in this report now resolved:**
- "Planner is greedy stub (no formal cost model)" → cost model calibrated with
  `EngineProfile.NsPerOp` (Memory=500ns, SQLite=7000ns, ADR-0061).
- "No real SQLite/Pebble engines" → `SQLiteEngine` shipped, wrapping
  `storage/view.SQLViewStore` (ADR-0061).
- "No cursor serialization" → base64-encoded URL-safe cursors implemented.
- "6 files exceed 350-line CI limit" → all split, lint clean (143 → 0 issues).
- "go.sum checksum issues" → resolved by the 2026-07-25 release tag fix (32
  missing tags created and pushed).
