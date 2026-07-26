# Meta-Engine Hardening Session — 2026-07-23 21:41

## Session Goal

Review the metaengine prototype, identify what was forgotten/could be improved, and execute a comprehensive hardening plan covering dead code removal, correctness bug fixes, concurrency safety, architectural improvements (ReadModel/Query separation), and regression tests.

## Commits This Session

```
a1e1c739 docs(metaengine): update README for ReadModel/Query API and new features
e6aea03c test(metaengine): add race, numeric sort, filter type, and encoded event tests
dc347374 refactor(metaengine): enhance reflection utilities and query builder capabilities
fe2eb674 refactor(metaengine): optimize query execution and result handling
c6e43526 feat(metaengine): add ApplyEncoded for JSON event payloads + context threading
84cc141d refactor(metaengine): separate ReadModel from Query to fix write amplification
8cd225c9 fix(metaengine): add concurrency safety with sync.RWMutex
fdebfe97 fix(metaengine): type-safe key extraction and pagination metadata
b67cd17a refactor(metaengine): improve command and query processing engine
eba12345 refactor(metaengine): remove dead code and replace custom joinStrings
6165e308 feat(metaengine): implement event query model with planner, fold, and store components
```

**Module stats:** 11 files, 2,434 lines, 10 tests passing with `-race`, 76.0% coverage.

---

## A) FULLY DONE

| #   | What                                                          | Files                                        | Evidence                                                   |
| --- | ------------------------------------------------------------- | -------------------------------------------- | ---------------------------------------------------------- |
| 1   | Dead code removal (Skip, Remove, describeType, joinStrings)   | types.go, reflect.go, query.go               | All removed, build clean                                   |
| 2   | Filter comparison fix (DeepEqual instead of Sprintf)          | engine.go                                    | TestFilterTypeCorrectness passes                           |
| 3   | Sort comparison fix (type-aware numeric/string/time)          | engine.go                                    | TestNumericSortCorrectness verifies 1<2<10                 |
| 4   | Key extraction via struct tag `metaengine:"key"`              | store.go                                     | FindUser/FriendsOf inputs tagged                           |
| 5   | Pagination metadata (HasMore + Next cursor)                   | store.go                                     | TestPagination_HasMore passes                              |
| 6   | Concurrency safety (sync.RWMutex on MemoryEngine + Store)     | engine.go, store.go                          | TestConcurrentApplyAndExecute -race clean                  |
| 7   | ReadModel/Query separation                                    | readmodel.go, query.go, planner.go, store.go | TestSharedModel_WriteAmplification verifies dedup          |
| 8   | Model() returns error (not panic)                             | readmodel.go                                 | `Model()` returns `(ReadModel, error)`, `MustModel` panics |
| 9   | ApplyEncoded for JSON event payloads (zero-dep)               | encoded.go                                   | TestApplyEncoded passes                                    |
| 10  | Context threading through ExecuteCtx                          | store.go                                     | Checks `ctx.Done()`                                        |
| 11  | Input dispatch collision prevention (package-qualified names) | reflect.go, query.go, store.go               | Uses `PkgPath().Name()`                                    |
| 12  | EventTypeNames enumeration                                    | encoded.go                                   | TestEventTypeNames passes                                  |
| 13  | README updated for new API                                    | README.md                                    | Documents ReadModel/Query, struct tags, adapters           |
| 14  | All tests pass with -race, go vet clean                       | -                                            | `go vet ./... && go test -race -count=1 ./...`             |

---

## B) PARTIALLY DONE

| #   | What                              | Current State                                               | What's Missing                                                                                                                                                                                                                                                           |
| --- | --------------------------------- | ----------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| 1   | event.Event integration           | `ApplyEncoded(type, []byte)` decodes JSON payloads          | No direct `event.Event` import (blocked by pre-existing go.sum checksum issues in workspace — `codec/v4.0.4` tag doesn't exist on remote). Adapter pattern documented in README.                                                                                         |
| 2   | projection.Projection integration | Adapter pattern documented (Name/Handle/EventTypes methods) | Not a direct import — requires consumer to write 10-line adapter. Can't import `projection/` without pulling in event/codec chain.                                                                                                                                       |
| 3   | Concurrency safety                | `sync.RWMutex` on MemoryEngine + Store RLock                | **CRITICAL: read-modify-write in `applyFold` (FoldUpdate) is NOT atomic.** `MapGet → callUpdate → MapSet` can race between concurrent Apply calls for the same key. The engine's mutex serializes individual operations but NOT the compound read-modify-write sequence. |
| 4   | `TypedDelta[K ~string]`           | Defined in types.go                                         | NOT wired into `OnCount` — still accepts untyped `Delta`. Next steps item, never executed.                                                                                                                                                                               |
| 5   | Coverage at 76%                   | Core paths covered                                          | `compareLess` at 18.8%, many String() methods at 0%, `OnSkip` at 0%, `SQLiteEngineProfile` at 0%, `MapDelete` at 0%                                                                                                                                                      |

---

## C) NOT STARTED

| #   | What                                     | Why                                                                                                                                      |
| --- | ---------------------------------------- | ---------------------------------------------------------------------------------------------------------------------------------------- |
| 1   | Real SQLite engine implementation        | `SQLiteEngineProfile()` is still a stub returning a cost profile with no implementation. Would need to wrap `storage/view/SQLViewStore`. |
| 2   | Wire `OnCount` to accept `TypedDelta[K]` | Mentioned in next steps, never started. `OnCount[E]` still takes `func(e E) Delta`.                                                      |
| 3   | Add metaengine to AGENTS.md module list  | Listed as "next step if keeping in monorepo" but not done.                                                                               |
| 4   | Run `nix fmt` on the changes             | Never executed. Formatting may not match `golines` output.                                                                               |
| 5   | Multi-engine plan demo (memory + SQLite) | The planner supports multiple engines but no test demonstrates mixed-engine assignment with degradation diagnostics.                     |
| 6   | `ReadScan` read pattern execution        | Inferred by `classifyADT` but falls through to `default` error in `executeQuery` — unreachable runtime path.                             |
| 7   | Conflict detection for mixed fold kinds  | `classifyADT` silently picks the highest-precedence ADT if both `OnCount` and `OnInsert` are present. No warning emitted.                |
| 8   | Cursor serialization                     | `Cursor{Value any}` has no encode/decode. The `Next` cursor holds the last item as `any` — can't be serialized across HTTP.              |
| 9   | Keyset pagination                        | `After(cursor)` is accepted but `MapScan` ignores it — no offset/seek support. Results always start from beginning.                      |
| 10  | Property-based testing                   | `pgregory.net/rapid` available in repo, not used by metaengine.                                                                          |

---

## D) TOTALLY FUCKED UP

| #   | What                                            | Severity                      | Details                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                           |
| --- | ----------------------------------------------- | ----------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| 1   | **MapScan sorts BEFORE full scan**              | **CRITICAL CORRECTNESS BUG**  | `MapScan` fetches `limit+1` items then breaks. It sorts only those items, NOT the full filtered set. If you have 100 matching items and limit=10, you get the first 11 random-order map entries, sorted, truncated to 10. The "correct" 10 by sort order are lost. This makes paginated sorted queries WRONG for any dataset larger than limit. **Root cause:** early-break optimization before sort. Fix: remove the break, sort the full filtered set, then truncate. Or: use a partial sort (heap) for large datasets.                         |
| 2   | **applyFold FoldUpdate race**                   | **HIGH — DATA CORRUPTION**    | `Store.Apply` uses `RLock`, but `applyFold` for `FoldUpdate` does `MapGet → callUpdate → MapSet`. Two concurrent Apply calls for the same aggregate read the same `prev` value, both compute `updated`, and the second write wins — silently losing the first update. The `MemoryEngine.MapSet` is individually mutex-protected but the compound RMW is not atomic. Fix: use `Lock()` not `RLock()` in `Apply`, or use per-key locking, or make `MapBackend` expose an atomic `MapUpdate(key, func(prev) newVal)` method (like `kv.ViewUpdater`). |
| 3   | **Go map iteration order is random**            | **MEDIUM — NONDETERMINISTIC** | `MapScan` iterates `map[any]any` which is randomized by Go runtime. Combined with the early-break bug (#1), paginated queries return different results on each execution even with identical inputs. Even without the break, the sort should stabilize this — but only after fixing #1.                                                                                                                                                                                                                                                           |
| 4   | **`volume`/`latencyBudgetMs` silently ignored** | LOW — DEAD API                | `QueryConfig` accepts `Volume(n)` and `WithLatencyBudget(ms)` but the planner never uses them for engine selection. This is misleading — users set values expecting behavior change.                                                                                                                                                                                                                                                                                                                                                              |
| 5   | **`OnRemove` has unused V type parameter**      | LOW — API SMELL               | `OnRemove[E, K, V]` requires V but only uses K. Forces users to write `OnRemove[UserDeleted, UserID, FindUserResult]` when V is irrelevant for deletion.                                                                                                                                                                                                                                                                                                                                                                                          |

---

## E) WHAT WE SHOULD IMPROVE

### Architecture

1. **Fix the sort-before-limit bug in MapScan** — sort the FULL filtered set before truncating. For large datasets, consider a partial sort (top-N heap) or a sorted index maintained on write.
2. **Add atomic read-modify-write** — either use `Lock()` in Apply (simpler, less concurrent), or add `MapUpdate(collection, key, func(prev any) any) error` to `MapBackend` (mirrors `kv.ViewUpdater`).
3. **Wire TypedDelta[K] into OnCount** — `OnCount[E, K ~string](sample, func(e E) TypedDelta[K])` variant for compile-time-safe counter keys.
4. **Make Volume/LatencyBudgetMs actually influence planning** — or remove them if they're never going to be used (YAGNI).
5. **Add cursor serialization** — `Cursor` needs `String() string` and `ParseCursor(s) (*Cursor, error)` for HTTP pagination.
6. **Implement keyset pagination** — `After(cursor)` should actually seek past the cursor position in `MapScan`.
7. **Add conflict detection for mixed fold kinds** — if both `OnCount` and `OnInsert` appear in the same ReadModel, emit a warning.

### Code Quality

8. **Remove unused V from OnRemove** — `OnRemove[E, K]` not `OnRemove[E, K, V]`.
9. **Remove ADTLog** — it's defined but unreachable (never inferred by `classifyADT`).
10. **Remove ReadScan** — inferred but unexecutable in `executeQuery` (falls to default error).
11. **Add coverage for `compareLess`** (18.8%) — test all numeric type branches.
12. **Add coverage for String() methods** — they're for human diagnostics but 0% covered.
13. **Run `nix fmt`** — may reformat line lengths.

### Integration

14. **Resolve go.sum checksum issues** so the module can import `event/` directly — the `codec/v4.0.4` tag mismatch blocks real `event.Event` integration workspace-wide.
15. **Add metaengine to AGENTS.md module list**.
16. **Add CI entry for metaengine** — `nix run .#test` should pick it up from go.work.
17. **Implement real SQLite engine** wrapping `storage/view/SQLViewStore` behind `MapBackend`/`ScanBackend`.

### Testing

18. **Property-based tests** — `rapid` is available; generate random event sequences and verify fold invariants.
19. **Multi-engine plan test** — memory + SQLite profile, verify degradation diagnostics fire.
20. **Test OnRemove path** (0% coverage on MapDelete).

---

## F) NEXT 50 THINGS TO DO (Prioritized)

### P0 — Critical Correctness (must fix before any use)

| #   | Task                                                                                       | Effort | Impact   |
| --- | ------------------------------------------------------------------------------------------ | ------ | -------- |
| 1   | Fix MapScan: sort full set before truncating                                               | 10min  | Critical |
| 2   | Fix applyFold FoldUpdate race: Lock not RLock in Apply, or atomic MapUpdate                | 15min  | Critical |
| 3   | Stabilize map iteration order for scan (sorted key list) or accept non-determinism in docs | 10min  | High     |

### P1 — API Hardening

| #   | Task                                                     | Effort | Impact |
| --- | -------------------------------------------------------- | ------ | ------ |
| 4   | Remove unused V from OnRemove signature                  | 5min   | Medium |
| 5   | Remove or implement Volume/LatencyBudgetMs               | 5min   | Medium |
| 6   | Remove unreachable ADTLog constant                       | 2min   | Low    |
| 7   | Remove unreachable ReadScan or implement it              | 5min   | Medium |
| 8   | Add conflict warning for mixed fold kinds in classifyADT | 10min  | Medium |
| 9   | Wire TypedDelta[K] into OnCount variant                  | 10min  | Medium |

### P2 — Pagination

| #   | Task                                                          | Effort | Impact |
| --- | ------------------------------------------------------------- | ------ | ------ |
| 10  | Implement cursor serialization (String/Parse)                 | 15min  | High   |
| 11  | Implement keyset pagination in MapScan (After cursor support) | 20min  | High   |
| 12  | Add offset-based pagination as alternative                    | 10min  | Medium |

### P3 — Event Integration

| #   | Task                                            | Effort | Impact                                  |
| --- | ----------------------------------------------- | ------ | --------------------------------------- |
| 13  | Resolve go.sum checksum issues across workspace | 30min  | Critical (unblocks direct event import) |
| 14  | Add direct event.Event Apply method             | 10min  | High                                    |
| 15  | Add direct projection.Projection implementation | 10min  | High                                    |
| 16  | Add ApplyBatch for multiple events in one call  | 10min  | Medium                                  |

### P4 — Real Engines

| #   | Task                                                       | Effort | Impact |
| --- | ---------------------------------------------------------- | ------ | ------ |
| 17  | Implement SQLite engine wrapping storage/view/SQLViewStore | 2h     | High   |
| 18  | Add pebble engine wrapping storage/pebble                  | 1h     | Medium |
| 19  | Multi-engine plan demo test with degradation diagnostics   | 15min  | Medium |

### P5 — Testing & Coverage

| #   | Task                                                | Effort | Impact |
| --- | --------------------------------------------------- | ------ | ------ |
| 20  | Test compareLess all numeric branches               | 10min  | Medium |
| 21  | Test MapDelete (OnRemove path)                      | 10min  | Medium |
| 22  | Test OnSkip fold kind                               | 5min   | Low    |
| 23  | Test String() methods on all types                  | 10min  | Low    |
| 24  | Add property-based test with rapid                  | 30min  | Medium |
| 25  | Test multi-engine plan with degradation diagnostics | 15min  | Medium |
| 26  | Test concurrent FoldUpdate (read-modify-write race) | 15min  | High   |

### P6 — Planner Improvements

| #   | Task                                                       | Effort | Impact |
| --- | ---------------------------------------------------------- | ------ | ------ |
| 27  | Use Volume/LatencyBudgetMs for engine selection            | 30min  | Medium |
| 28  | Add cost estimation output (estimated QPS, latency)        | 20min  | Low    |
| 29  | Support multi-engine fan-out (query uses multiple engines) | 1h     | Low    |
| 30  | Add health check interface on Engine                       | 10min  | Medium |

### P7 — Code Quality

| #   | Task                                                                        | Effort | Impact |
| --- | --------------------------------------------------------------------------- | ------ | ------ |
| 31  | Run nix fmt on all files                                                    | 2min   | Low    |
| 32  | Add metaengine to AGENTS.md module list                                     | 5min   | Medium |
| 33  | Add CI entry for metaengine                                                 | 5min   | Medium |
| 34  | Add error wrapping with errorfamily classification                          | 15min  | Medium |
| 35  | Add file-level doc comments                                                 | 10min  | Low    |
| 36  | Extract matchFilterFields type comparison to use comparable Type not string | 20min  | Medium |

### P8 — Features

| #   | Task                                                       | Effort | Impact |
| --- | ---------------------------------------------------------- | ------ | ------ |
| 37  | Add ScanBackend for full collection scans (no filter/sort) | 10min  | Low    |
| 38  | Add CounterBackend.GetSingle(key) for point counter reads  | 5min   | Medium |
| 39  | Add GraphBackend.ShortestPath                              | 10min  | Medium |
| 40  | Add batch Apply (multiple events, one lock acquisition)    | 15min  | Medium |
| 41  | Add Reset/Clear method on Store for projection rebuild     | 10min  | Medium |
| 42  | Add Stats/Snapshot method on MemoryEngine for debugging    | 10min  | Low    |

### P9 — Documentation

| #   | Task                                             | Effort | Impact |
| --- | ------------------------------------------------ | ------ | ------ |
| 43  | Add package-level doc.go                         | 5min   | Low    |
| 44  | Document the fold registration API with examples | 10min  | Medium |
| 45  | Document struct tag conventions (key, sort)      | 5min   | Low    |
| 46  | Add migration guide from stack.Materialize       | 20min  | Medium |

### P10 — Future

| #   | Task                                             | Effort  | Impact |
| --- | ------------------------------------------------ | ------- | ------ |
| 47  | Codegen validator (metaengine-check CI tool)     | 4h      | Medium |
| 48  | TypeSpec extension for multi-target generation   | 2 weeks | Low    |
| 49  | Automatic schema generation for SQL engines      | 1 day   | Medium |
| 50  | Benchmark suite (compare with stack.Materialize) | 2h      | Medium |

---

## G) Questions I Cannot Answer

### Q1: Should the read-modify-write race fix use Lock() (simpler, serializes all Apply calls) or an atomic MapUpdate method (mirrors kv.ViewUpdater, allows concurrent applies for different keys)?

Using `Lock()` in `Apply` is the 2-minute fix but serializes ALL event processing globally. Adding `MapUpdate(collection, key, func(prev any) any) error` to `MapBackend` allows concurrent Apply for different keys while keeping per-key atomicity. The latter mirrors the existing `kv.ViewUpdater` interface but changes the `MapBackend` interface (breaking change for any future engine implementors).

### Q2: Should the MapScan sort-before-limit bug fix use full-sort-then-truncate (O(N logN), correct) or maintain a sorted index on write (O(logN) amortized, more complex)?

Full-sort is trivially correct but scans all matching items every time. A sorted index (maintained on MapSet) is O(logN) for scan but adds write overhead and memory. For a prototype/testing engine, full-sort is probably fine. For a "real" engine, you'd use SQL's ORDER BY + LIMIT server-side. Should I optimize the memory engine, or accept it as testing-only and document the limitation?

### Q3: Should metaengine/ stay zero-dependency (stdlib only) or should we resolve the go.sum checksum issues and add the event/ dependency?

Zero-dependency means ApplyEncoded + adapter pattern only — no direct `event.Event`, `projection.Projection`, or `codec.Codec` integration. Adding the event/ dependency enables `ApplyEvent(evt event.Event)`, `AsProjection()` returning a `projection.Projection`, and codec-aware payload decoding. But it requires resolving the pre-existing `codec/v4.0.4` tag mismatch that affects the entire workspace, not just metaengine.

---

## Resolution (2026-07-26)

The critical bugs reported above were **all fixed** in subsequent sessions:

- **MapScan sort-before-scan** → resolved by design: MemoryEngine uses full-sort
  (acceptable for testing), SQLiteEngine delegates to SQL `ORDER BY + LIMIT`
  (ADR-0061).
- **FoldUpdate race (data corruption)** → fixed via tx-atomic `MapUpdate` wrapping
  read-modify-write in a single SQLite transaction (ADR-0067). Multimap seq-seed
  made restart-safe via `sync.Once` + `MAX(seq)` seeding (ADR-0068).
- **`map[string]any` → struct reification** → centralized in `reifyReflect`
  (`metaengine/reify.go`), co-located with generic `reify[R]`. Cross-engine meta-test
  (`cross_engine_meta_test.go`, 150 specs) asserts Memory and SQLite produce
  identical typed results (ADR-0066).

**Open questions resolved:**

- **Q1 (sort strategy):** full-sort for Memory (testing-only), ORDER BY for SQLite.
- **Q2 (FilterOn SQL pushdown):** Phase 1 keeps in-memory closures + `PushdownScan`
  interface seam. Phase 2 declarative `FilterSpec`/`SortSpec` deferred (ADR-0063).
- **Q3 (zero-dependency):** Core `metaengine/v4` stays zero-dep. Event/projection
  integration lives in `metaengine/projectionadapter/` as a separate module
  (ADR-0062). Tagged `metaengine/v4.1.1`.

**Current state:** 174 BDD specs + 150 cross-engine meta specs, 87.7% coverage,
lint clean, SQLite engine + projection adapter + cost calibration shipped.
