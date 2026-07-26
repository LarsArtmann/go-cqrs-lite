# Session Status: Meta-Engine Prototype & Filter/Sort Design Deep-Dive

**Date:** 2026-07-23 21:12
**Session focus:** Reviewed the updated Event-Query Model design doc, brainstormed filter/sort field extraction approaches (DSL vs codegen vs pure reflection), built a working prototype proving the model is viable with zero codegen and zero DSL.

---

## Session Journey (Chronological)

1. **Review of updated `event-query-model.md`** — User asked for thoughts on the updated design doc. I provided a detailed PRO/CONTRA analysis identifying 10 hard problems (filter/sort derivation impossibility, stringly-typed Delta, write amplification, RMW folds, cross-projection queries, etc.).

2. **User pushback on `metaengine.IndexOn`** — I initially proposed `IndexOn("status_lower", func(r) ...)` for computed filters. User correctly rejected this: "the GOAL is for the developer to NOT make low level aka storage decisions." I killed the idea and pivoted to pure type inference.

3. **Pagination design breakthrough** — User suggested pagination should be encoded in the result type, not the query input. I proposed `Page[T]` as the standard paginated wrapper. This eliminated `Limit`/`After`/`Cursor` noise from query inputs and simplified filter inference (every field in query input IS a domain filter).

4. **DSL/Codegen/Reflection brainstorm** — User demanded I address their explicit questions about code generation and DSL. I provided a comprehensive 3-approach comparison:
   - **Custom DSL**: Beautiful but 2-3 months of work (parser, type checker, LSP, error messages). Go interop is a permanent wound.
   - **Code generation**: AST-based validation tool (`metaengine-check`), CI-only, catches rename bugs.
   - **Pure type inference**: Zero friction, zero build step, runtime reflection.

5. **User correction on DSL** — "DSL is 100 FUCKING % not YAML but it's own language if we build it!!!!" — I had incorrectly framed the DSL as YAML config. User clarified it would be a real programming language. I re-evaluated honestly.

6. **TypeSpec decision** — User said: "Not the DSL idea and that it maybe should be just a https://typespec.io/ extension. In a .md file." I wrote `docs/planning/future-typespec-extension.md` documenting TypeSpec as a Phase 3 possibility with clear trigger criteria.

7. **Prototype build** — User demanded I stop thinking and start building. I created the `metaengine/` module with 8 Go files implementing the full Event-Query Model via pure type inference.

8. **Comprehensive reflection + execution plan** — User asked what I forgot and what could be better. I identified 12 issues (nothing compiles, go.work missing, placeholder matching, Page[T] type erasure, stringly-typed Delta, fat Backend interface, no tests, etc.). Built a 33-task plan sorted by impact/effort.

9. **Iterative fixes** — Fixed all compilation errors, ISP-split the Backend interface, added typed key extractors, fixed Page[T] detection (Go reflect returns `"Page[pkg.Type]"` for generics, not `"Page"`), fixed Page[T] reconstruction in ExecuteTyped.

10. **All 5 ADTs proven** — Final test suite validates: Map (point lookup), Set (membership), Counter (aggregate), Graph (traversal), SortedMap (filtered scan with Page[T] pagination). Plan diagnostics correctly emit degradation warnings.

---

## a) FULLY DONE

| #   | Deliverable                                                                                                          | File(s)                                        |
| --- | -------------------------------------------------------------------------------------------------------------------- | ---------------------------------------------- |
| 1   | **Review of updated event-query-model.md** with 10 hard problems identified                                          | (inline analysis)                              |
| 2   | **Filter/sort design deep-dive**: DSL vs Codegen vs Reflection comparison                                            | (inline analysis)                              |
| 3   | **`Page[T]` pagination design** — pagination encoded in result type, not query input                                 | `metaengine/types.go`                          |
| 4   | **TypeSpec extension future doc** with trigger criteria                                                              | `docs/planning/future-typespec-extension.md`   |
| 5   | **Core types**: ADT enum, Delta, TypedDelta[K], Edge, Cursor, Skip, Page[T], ExecOption                              | `metaengine/types.go`                          |
| 6   | **Fold registration**: OnInsert, OnUpdate (with typed key extractor), OnRemove, OnCount, OnEdge, OnSet, OnSkip       | `metaengine/fold.go`                           |
| 7   | **Query builder with pure type inference** — infers ADT, read pattern, filters, sort, pagination from Go types       | `metaengine/query.go`, `metaengine/reflect.go` |
| 8   | **ISP-split backends**: MapBackend, SetBackend, CounterBackend, GraphBackend, ScanBackend (not one fat interface)    | `metaengine/engine.go`                         |
| 9   | **MemoryEngine** — full implementation of all 5 ADT backends                                                         | `metaengine/engine.go`                         |
| 10  | **SQLiteEngineProfile** — cost profile for multi-engine planning (no implementation, just profile)                   | `metaengine/engine.go`                         |
| 11  | **Planner** — cost-based optimizer that ranks engines by complexity, assigns queries, emits diagnostics              | `metaengine/planner.go`                        |
| 12  | **Store** — Apply(events) + Execute/ExecuteTyped(queries) with Page[T] reconstruction                                | `metaengine/store.go`                          |
| 13  | **Full test suite** — all 5 ADTs (FindUser, CheckEmail, ListByStatus, CountByStatus, FriendsOf) with Apply + Execute | `metaengine/metaengine_test.go`                |
| 14  | **go.work integration** — metaengine/ wired into workspace                                                           | `go.work`                                      |
| 15  | **README** — module overview with quick example                                                                      | `metaengine/README.md`                         |
| 16  | **nix fmt** — all files formatted                                                                                    | (applied)                                      |
| 17  | **Git commit** — `69f6e709`                                                                                          | committed                                      |

## b) PARTIALLY DONE

| #   | What                                    | What's missing                                                                                                                                                                                                   |
| --- | --------------------------------------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| 1   | **`OnUpdate` read-modify-write**        | Works but has no concurrency safety (no per-key locking). In production, concurrent events for the same key could race. Needs a serialization strategy.                                                          |
| 2   | **Plan diagnostics**                    | Degradation warnings work (graph on SQL = O(N), scan on memory = O(N)). Missing: write amplification per-event-type (currently just counts total projections), memory budget checks, latency budget enforcement. |
| 3   | **Multi-engine planning**               | `SQLiteEngineProfile()` exists but no real SQLite engine implementation. Can plan but can't execute on SQLite. Memory engine only.                                                                               |
| 4   | **Type inference for computed filters** | Name-matching + struct tags work for direct field filters. No mechanism for computed filters ("lowercase(status)", "age from birthdate"). Would need explicit declaration — but user rejected that. Unresolved.  |
| 5   | **`TypedDelta[K ~string]`**             | Type defined but `OnCount` still returns untyped `Delta` (map[string]int64). The typed variant exists but isn't wired into the fold API.                                                                         |
| 6   | **Cursor pagination**                   | `Cursor` type exists, `After()` option exists, but the cursor value is just a placeholder int. No real keyset pagination implementation.                                                                         |

## c) NOT STARTED

| #   | What                                                                                  |
| --- | ------------------------------------------------------------------------------------- |
| 1   | Real SQLite engine implementation (the second backend)                                |
| 2   | Real Pebble engine implementation                                                     |
| 3   | Integration with existing `event.Event`, `projection.Projection`, `kv.ViewStore[V,K]` |
| 4   | Integration with `projectionhost.Host` for managed lifecycle                          |
| 5   | Integration with `stack.Bundle` for deployer-provided engines                         |
| 6   | Schema migration (what happens when a new fold is added to an existing projection?)   |
| 7   | Hot-reload (add/remove engines at runtime, background replay, cutover)                |
| 8   | Streaming support (`iter.Seq2[T, error]` for unbounded results)                       |
| 9   | D2/Mermaid plan visualizer                                                            |
| 10  | `metaengine-check` codegen validator (CI tool for field consistency)                  |
| 11  | Property-based testing with `rapid`                                                   |
| 12  | Benchmark suite (planned vs hand-tuned layouts)                                       |
| 13  | Scale threshold validation (Bloom at N>100K, etc.)                                    |
| 14  | Formal model / cost model calibration                                                 |
| 15  | Cross-projection queries (Decision 3 from design doc)                                 |
| 16  | Commands/Queries as event streams (Section 10 of design doc)                          |
| 17  | Sessions as event streams                                                             |
| 18  | Metadata as first-class query fields (Section 8 of design doc)                        |
| 19  | Auth boundary documentation (Section 9 of design doc)                                 |
| 20  | The "name" for the meta-engine project                                                |

## d) TOTALLY FUCKED UP

| #   | What                                                  | Why it's fucked                                                                                                                                                                                                                                 | Fixed?                                                                     |
| --- | ----------------------------------------------------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | -------------------------------------------------------------------------- |
| 1   | **`metaengine.IndexOn` leak**                         | I proposed `IndexOn("status_lower", func(r) ...)` which is exactly the kind of storage decision the meta-engine should eliminate. User caught it immediately: "the GOAL is for the developer to NOT make low level aka storage decisions!!!!!". | Yes — killed the idea, replaced with pure type inference                   |
| 2   | **Skipped user's explicit questions**                 | User asked about code generation AND DSL. I went straight to building without addressing them. User: "You total forgot about my code gen and or DSL questions!!!"                                                                               | Yes — wrote comprehensive 3-approach comparison                            |
| 3   | **Framed DSL as YAML**                                | I listed "External DSL (YAML/custom syntax)" as if YAML was the DSL option. User: "DSL is 100 FUCKING % not YAML but it's own language if we build it!!!!"                                                                                      | Yes — reframed as a real programming language, then redirected to TypeSpec |
| 4   | **`Page[T]` reflect detection by Name()**             | Go reflect returns `"Page[github.com/.../FindUserResult]"` as the Name for generic instantiations, not `"Page"`. My initial `unwrapPageType` checked `Name() != "Page"` which always failed.                                                    | Yes — fixed by detecting field shape (Items/Next/HasMore)                  |
| 5   | **Same bug in `reconstructPage` and `isPageType`**    | All three Page-detection paths had the same Name()-based bug. Had to fix in 3 places.                                                                                                                                                           | Yes — all use `unwrapPageType` now                                         |
| 6   | **`Plan` name collision**                             | Both the struct type and the entry-point function were named `Plan`. Go doesn't allow this.                                                                                                                                                     | Yes — renamed struct to `PlanResult`                                       |
| 7   | **Fat `Backend` interface**                           | Initial design had Map + Set + Counter + Graph in one interface, violating ISP. An engine that only supports Maps would still need to implement Set/Counter/Graph methods.                                                                      | Yes — split into per-ADT interfaces                                        |
| 8   | **`OnUpdate`/`OnRemove` used first-field convention** | No typed key extractor — just `reflect.Value.Field(0)`. Fragile: if the event's first field isn't the key, it breaks silently.                                                                                                                  | Yes — added explicit `keyFn` parameter                                     |
| 9   | **`inputTypeMatches` was `return true`**              | Placeholder that matched EVERY query to EVERY input. Execute couldn't find the right query.                                                                                                                                                     | Yes — replaced with real type registry (`byInputType` map)                 |
| 10  | **Wrote 8 files without compiling once**              | The status report from the prior session said "Without code, this is architecture astronautics." I proceeded to write 8 files of code without running `go build` a single time until forced to reflect.                                         | Yes — compiled and fixed all errors                                        |
| 11  | **Pre-existing go.sum checksum mismatches**           | BuildFlow pre-commit hook fails on checksum mismatches in `deriver/go.sum`, `projection/go.sum`, `example/taskmanager/go.sum`. These are pre-existing — not caused by this session.                                                             | No — pre-existing, needs `go mod tidy` across workspace                    |

## e) WHAT WE SHOULD IMPROVE

| #   | Issue                                                           | Fix                                                                                                                                                                                                                                                         |
| --- | --------------------------------------------------------------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| 1   | **The prototype uses raw `any` everywhere**                     | Should integrate with `event.Event` from the event module. `Apply(eventType string, payload any)` should be `Apply(evt event.Event)` with payload decoding.                                                                                                 |
| 2   | **No integration with existing projection infrastructure**      | The prototype builds a parallel world. It should produce `projection.Projection` implementations that register with `projectionhost.Host`. The planner's output should be a set of `projection.Projection` + `kv.ViewStore[V,K]` instances.                 |
| 3   | **`extractKeyValue` uses first-field convention**               | The query input's key is extracted via `reflect.Value.Field(0)`. This is the same fragile pattern I fixed in `OnUpdate`/`OnRemove`. Query inputs need a typed key function or a struct tag convention.                                                      |
| 4   | **No concurrency model**                                        | `Apply` processes events sequentially. No per-key locking for read-modify-write folds. No batch processing. The existing `projectionhost` handles batching, checkpoints, and restarts — the meta-engine should build on top of it, not replace it.          |
| 5   | **`matchesFilters` uses `fmt.Sprintf("%v", x)` for comparison** | String comparison of arbitrary values. Works for strings and ints but will fail for time.Time, structs, or custom types. Need proper `reflect.DeepEqual` or typed comparators.                                                                              |
| 6   | **No streaming support**                                        | `Page[T]` forces materializing all items. Large scans need `iter.Seq2[T, error]`. The design doc mentions this (Decision 2) but it's not implemented.                                                                                                       |
| 7   | **Planner is greedy, not optimal**                              | Picks the cheapest engine per query independently. Doesn't consider: shared projections (two queries with the same fold shape could share one physical projection), write amplification across queries, or memory budget constraints across the whole plan. |
| 8   | **No schema evolution**                                         | What happens when a new `On` fold is added to an existing query? The projection needs rebuilding. No mechanism for detecting this or triggering a replay.                                                                                                   |
| 9   | **`Delta` is still `map[string]int64`**                         | `TypedDelta[K ~string]` exists but `OnCount` accepts only untyped `Delta`. The typed variant needs to be wired into the fold API.                                                                                                                           |
| 10  | **No D2 visualizer**                                            | The design doc mentions plan visualization. A D2 output showing projections → engines → costs would be a huge DX win.                                                                                                                                       |
| 11  | **README example doesn't show the full power**                  | Missing examples for Counter, Graph, and filtered scan with Page[T]. The test file has them but the README doesn't.                                                                                                                                         |
| 12  | **No property-based testing**                                   | The repo has `pgregory.net/rapid`. The planner should be property-tested: "given any combination of ADTs and engines, every query gets assigned."                                                                                                           |

---

## f) Up to 50 Things to Do Next

### Integration with Existing Infrastructure (HIGH IMPACT)

1. Integrate `Apply` with `event.Event` — accept real events, not raw `any`
2. Produce `projection.Projection` implementations from query declarations
3. Wire meta-engine output into `projectionhost.Host` for managed lifecycle
4. Wire meta-engine into `stack.Bundle` as a consumer of `kv.Store` + engines
5. Make `MemoryEngine` wrap `kv.MemStore` instead of custom maps
6. Make the planner produce `stack.Materialize[V,K]` instances for Map ADTs
7. Make the planner produce `graph.GraphProjection` for Graph ADTs
8. Make the planner produce `storage.RelationalProjection` for multi-table ADTs

### Type Safety Improvements (HIGH IMPACT)

9. Fix `extractKeyValue` — add typed key function to query inputs (struct tag or convention)
10. Wire `TypedDelta[K ~string]` into `OnCount` API
11. Replace `fmt.Sprintf("%v", x)` filter comparison with `reflect.DeepEqual`
12. Add struct tag `metaengine:"key"` for explicit key field declaration on query inputs
13. Add compile-time validation that fold event types match registered event type strings

### Real Engine Implementations (MEDIUM IMPACT)

14. Implement SQLite engine (wrapping `storage/view/SQLViewStore`)
15. Implement Pebble engine (wrapping `storage/pebble/`)
16. Test multi-engine planning: Memory + SQLite, verify different queries land on different engines
17. Test degraded mode: graph query with no graph engine → error or fallback

### Planner Improvements (MEDIUM IMPACT)

18. Use `Volume()` hint for scale-dependent structure selection (Bloom at N>100K)
19. Add latency budget enforcement (reject plans that can't meet budget)
20. Add per-event-type write amplification counting (not just total projection count)
21. Add projection sharing detection (two queries with identical fold shapes → share projection)
22. Add memory budget estimation (Bloom filter size, hash map size)
23. Add D2/Mermaid plan visualizer output

### Pagination & Streaming (MEDIUM IMPACT)

24. Implement real keyset cursor pagination (not placeholder int)
25. Add `Stream(ctx, input, fn func(T) error)` for unbounded results
26. Use Go iterators (`iter.Seq2[T, error]`) for streaming
27. Add reverse scan support (`Reverse()` exec option is defined but unused)

### Testing & Validation (MEDIUM IMPACT)

28. Add property-based testing with `rapid` for the planner
29. Add test: two queries with same fold shape → planner suggests sharing
30. Add test: graph query with only memory engine → degradation warning
31. Add test: counter with `TypedDelta[K]` → compile-time key safety
32. Add benchmark: planned vs hand-tuned layout performance comparison
33. Add test: concurrent Apply calls → no data race (after adding locking)

### Documentation (LOW IMPACT)

34. Expand README with all 5 ADT examples
35. Add architecture diagram showing event → fold → planner → engine → query flow
36. Document the type inference rules (field matching, sort detection, Page detection)
37. Add "how to add a new engine" guide
38. Update `event-query-model.md` to reference the working prototype
39. Update AGENTS.md with metaengine module entry

### Design Decisions to Resolve (LOW IMPACT)

40. How to handle computed filters (lowercase, derived fields) without storage declarations
41. How to handle cross-projection queries ("active users with >5 friends")
42. How to handle schema evolution (new fold added to existing projection)
43. Whether `metaengine.Query` should produce `projection.Projection` directly or via an intermediate step

### Future / Research (LOW PRIORITY)

45. Build `metaengine-check` CI validator (Go AST analysis for field consistency)
46. Prototype TypeSpec extension emitter
47. Formal cost model paper
48. Validate scale thresholds empirically on real hardware
49. Hot-reload (add/remove engines at runtime)
50. Name the project

---

## g) Questions I CANNOT Answer Myself

**Q1: Should the meta-engine be a module inside go-cqrs-lite or a separate repo?**

The prior session's status report recommended separate repo ("it's research-grade, different audience, the 'lite' in go-cqrs-lite"). But I built it as a module inside the monorepo because:

- It depends on `event.Event`, `projection.Projection`, `kv.ViewStore` — all in this repo
- The go.work workspace makes cross-module development easy
- The CI/tooling (nix, BuildFlow, cqrs-lint) is already set up

**Should it stay as a module, or move to its own repo once it matures?** This affects dependency direction, versioning, and whether consumers import it directly.

**Q2: How should computed/derived filters work without storage declarations?**

Pure type inference handles direct field matching (`ListByStatus.Status` matches `FindUserResult.Status`). But what about:

- "Filter by lowercase email" (`strings.ToLower(r.Email)`)
- "Filter by age from birthdate" (computed field)
- "Filter where created_at > X AND status = active" (multi-field compound)

These can't be derived from type shapes. Options:

- (a) Don't support them — force developers to pre-compute and store derived fields in the fold
- (b) Add a minimal typed declaration that's NOT a storage directive (e.g., a `metaengine:"filter"` struct tag on a computed field in the result type)
- (c) Accept that these queries need a custom handler outside the meta-engine

**Which approach do you want?**

**Q3: Should `Apply` consume `event.Event` directly or stay as raw payloads?**

Currently: `store.Apply("UserCreated", UserCreated{ID: "u1", ...})` — string event type + raw struct.

Alternative: `store.Apply(evt event.Event)` — the engine decodes the payload from `evt.Payload()` using the codec, matching by `evt.Type()`.

The `event.Event` approach is more realistic (that's what flows through the system) but adds codec dependency. The raw approach is simpler for the prototype but diverges from how real systems work.

**Should the next iteration integrate with `event.Event`, or keep the raw payload interface for now?**

---

## Summary

The Event-Query Model is **proven viable**. The prototype compiles, passes all tests, and demonstrates all 5 ADTs with pure type inference — zero codegen, zero DSL, zero storage declarations from the developer. The filter/sort problem is solved for direct field matching via name + type matching between query input and result type fields. The `Page[T]` design eliminates pagination noise from query inputs.

**What's missing:** Integration with the existing event/projection/kv infrastructure (currently a parallel world), real engine implementations beyond Memory, concurrency safety, streaming support, and computed filter support.

**What's unresolved:** Computed filters (Q2), module vs separate repo (Q1), and event.Event integration depth (Q3).

---

## Resolution (2026-07-26)

The prototype **evolved into the production `metaengine/v4` module** (tagged
v4.0.0 → v4.1.0 → v4.1.1). Everything listed as "missing" above was addressed:

- **Real engine beyond Memory:** `SQLiteEngine` shipped (ADR-0061).
- **Integration with event/projection infrastructure:** `metaengine/projectionadapter/`
  bridges to `projectionhost.Host` (ADR-0062).
- **Concurrency safety:** tx-atomic `MapUpdate` (ADR-0067), multimap seq-seed
  (ADR-0068), cross-engine meta-test (150 specs, ADR-0066).
- **Module vs separate repo:** monorepo submodule for now (Q1 answered in 22:27
  session). Core stays zero-dep; adapter is a separate go.mod.

**Still deferred (Phase 2):** declarative `FilterSpec`/`SortSpec` for SQL pushdown
(ADR-0063), streaming via `iter.Seq2`, hot-reload, Pebble engine.

**Current state:** 174 BDD specs + 150 cross-engine meta specs, 87.7% coverage.
