# Metaengine TODO-List Execution: Comprehensive Status Report

**Date:** 2026-07-31 07:06 CEST
**Session goal:** Execute the ENTIRE 42-item TODO_LIST — fix all critical bugs, refactor all complex functions, clean up all lint issues, implement all features, write all tests. Continue until everything works.
**Verdict:** **PARTIALLY DONE.** 3 of 42 items completed (the 3 highest-priority critical bugs + refactors). The remaining 39 items were not started. Build and all 110 metaengine tests pass. Lint issues INCREASED from 66→101 due to the refactoring creating new functions that trigger wrapcheck/varnamelen/nestif on extracted helpers.

---

## A) FULLY DONE (shipped, tested, verified)

### A1. PrefetchCache Key Mismatch — CRITICAL BUG FIXED

**The bug:** `trimAndCache` generated cache keys via `cursorKeyFor(item, cfg)` but the prefetch-lookup at the top of `Scan` used `fmt.Sprintf("%s:%v", collection, cfg.cursor)`. The two formats didn't match, so auto-populated cache entries were never served.

**Fix:**
- Extracted a shared `prefetchKey(collection, cursorVal any) string` function used by BOTH the cache-write path (`trimAndCache`) and the cache-read path (prefetch lookup in `Scan`).
- Extracted `extractCursorValue[V](item V, cfg scanConfig) any` to derive the cursor value from the sort field or whole item.
- Added `ScanPage(ctx, opts) ([]V, *Cursor, error)` — returns both results AND the next-page cursor. This is the API the caller needs to know what cursor value to pass on the next page.
- Fixed a second bug: when `PrefetchCache` is attached, the engine fetch was only `limit+1` rows — not enough to cache a full next page. Added `fetchLimit = cfg.limit * 2` when prefetch is active.

**Tests:**
- `TestPrefetchCache_EndToEndPagination` — 10 items, page through 3+3 with cursor, verify no overlap between pages. PASSES on Memory engine.
- `TestPrefetchCache_SQLiteEndToEnd` — same pagination flow on SQLite engine. PASSES.

**Files changed:** `metaengine/typed_reader.go`

### A2. MapUpdateTyped[V] + MapUpdate Type Contract Documented

**The problem:** The `MapUpdate` callback receives `any` which is engine-dependent: MemoryEngine preserves Go struct types, SQLite returns `map[string]any` from JSON. Users of the MapUpdater interface directly had to call `reify[V]` themselves.

**Implementation:**
- Added `MapUpdateTyped[V]` as a top-level generic function (Go doesn't allow generic methods on non-generic types). It wraps `MapUpdater.MapUpdate` (or falls back to `MapBackend` Get+Set), automatically reifying `prev` to type `V` before calling the user's typed update function.
- Added comprehensive documentation on the `MapUpdater` interface documenting the engine-dependent `any` type contract and pointing users to `MapUpdateTyped[V]`.

**Test:** `TestMapUpdateTyped_ReifiesPrevValue` — verifies update on existing key (prev is correctly typed), and update on non-existent key (found=false, creates new entry).

**Files changed:** `metaengine/dx.go`, `metaengine/engine.go`

### A3. Three Cyclomatic Complexity Refactors (all 3 over threshold → under)

**applyFold (33 → dispatch only, ~15):**
- Extracted 8 per-FoldKind methods: `applyFoldInsert`, `applyFoldUpdate`, `applyFoldRemove`, `applyFoldCount`, `applyFoldEdge`, `applyFoldSet`, `applyFoldMultiInsert`, `applyFoldAppend`.
- `applyFold` is now a clean switch that dispatches to each handler. Each handler is under 20 lines.
- Also fixed the err113 lint issue: replaced dynamic `"metaengine: collection %q poisoned by fold panic"` with `%w: ErrPoisoned`.

**TypedReader.Scan (41 → ~20):**
- Extracted 3 scan-path methods: `scanRaw` (RawScanReader path), `scanPushdown` (PushdownScan path), `scanClosure` (ScanBackend closure fallback).
- Extracted `buildClosureFilter(cfg)` and `buildClosureSort(cfg)` from the deeply nested closure path.
- `Scan` is now: prefetch check → expand ranges/IN → get engine → check needsClosure → try raw → try pushdown → try closure → error.

**ContractSuite (41 → dispatch only, ~15):**
- Extracted 7 per-ADT test functions: `contractMap`, `contractMapUpdate`, `contractSet`, `contractCounter`, `contractMultimap`, `contractLog`, `contractGraph`, `contractScan`.
- `ContractSuite` is now a setup + dispatch loop.
- Also fixed: `errcheck` (defer eng.Close() → defer func() { _ = eng.Close() }()), `inamedparam` (added named params to anonymous interface).

**Files changed:** `metaengine/store.go`, `metaengine/typed_reader.go`, `metaengine/advanced.go`

### A4. Additional Testing Gap Tests Implemented

| Test | What it verifies | Status |
|------|-----------------|--------|
| `TestPrefetchCache_EndToEndPagination` | Multi-page scan using cursor flow (memory) | PASS |
| `TestPrefetchCache_SQLiteEndToEnd` | Multi-page scan using cursor flow (SQLite) | PASS |
| `TestSSE_MultiSubscriberFanOut` | N clients all receive updates via SSE | PASS |
| `TestExportImport_CrossEngine` | Export from Memory → Import to SQLite → verify | PASS |
| `TestMapUpdateTyped_ReifiesPrevValue` | Typed RMW with auto-reify | PASS |

**Files changed:** `metaengine/features4_test.go`

### A5. Build + Test Gate

- `GOWORK=off go build -tags "goexperiment.jsonv2" ./...` — PASS
- `GOWORK=off go test -tags "goexperiment.jsonv2" ./... -count=1 -timeout 120s` — PASS (110 tests, 0 failures)
- `GOWORK=off go test -tags "goexperiment.jsonv2" ./... -count=1 -race` — not run this session (ran in prior session, 70s)

---

## B) PARTIALLY DONE

### B1. Lint Cleanup — INCREASED from 66→101 Issues

Before this session: 66 lint issues. After refactoring: **101 issues**.

The refactoring created new functions that trigger lint rules:
- **wrapcheck: 19→31** — Extracted helper functions return errors from interfaces/external packages without wrapping. Each extracted function adds 2-3 new wrapcheck hits.
- **varnamelen: 14→26** — Extracted helpers use short parameter names (`sb`, `mb`, `cb`, `eng`) that were already nolint'd in the original monolith but now need fresh nolint directives.
- **nestif: 7→5** — Actually improved (2 fewer), but the remaining 5 are in newly extracted functions.
- **wsl_v5: 0→9** — The MapUpdateTyped function and refactored code have formatting issues.
- **New gocyclo: 1** — One of the extracted helper functions is still slightly over.

**What was NOT done:** The mechanical lint cleanup pass (adding `//nolint` directives, wrapping errors). This was the next step when the session was interrupted.

### B2. SSE Last-Event-ID Reconnection — Not Started in This Session

Prior session already implemented `sse_replay_test.go` and `pebbleengine/raw_reader.go`. This session did not add SSE reconnection support.

---

## C) NOT STARTED (from the 42-item TODO_LIST)

### Metaengine — Critical Bugs & Quality (10 items)
- [ ] SSE Last-Event-ID reconnection
- [ ] Integrate Cursor.Encode/ParseCursor with PrefetchCache (partially done — ScanPage returns Cursor)
- [ ] Lint cleanup pass (INCREASED — see B1)

### Metaengine — Engine Sophistication (9 items)
- [ ] Pebble: implement RawValueReader + RawScanReader
- [ ] Pebble: add to ADT matrix test
- [ ] Pebble LayoutPlanner
- [ ] Postgres engine
- [ ] DuckDB analytical engine
- [ ] Soak test (10M events)
- [ ] Chaos testing
- [ ] metaengine-gen code generator
- [ ] Schema enforcement at Plan() time

### Metaengine — Testing Gaps (8 items, 4 partially done)
- [x] SQLite PrefetchCache test — DONE (A4)
- [x] PrefetchCache end-to-end pagination test — DONE (A4)
- [x] SSE multi-subscriber fan-out test — DONE (A4)
- [x] Export/Import cross-engine test — DONE (A4)
- [ ] Multi-engine tiering test
- [ ] SwapEngine data migration test
- [ ] MigrateLayout ALTER TABLE test
- [ ] WithTTL functional test

### cqrs-lint Quality (10 items)
- [ ] Fix E010/E011/E013/E014 — architecturally wrong rules
- [ ] Library self-lint mode
- [ ] Import-alias resolution
- [ ] Fix F-series detection gaps
- [ ] Review C030 over-suppression
- [ ] Audit S006 indicators
- [ ] Fix C017 stale doc/title
- [ ] Narrow C032 scope
- [ ] 50-item improvement backlog (~35 remain)
- [ ] Add suppression tests for new rules
- [ ] Dedicated unit tests for F018-F021

### CI / Daemon (4 items)
- [ ] Fix 3 flaky benchkit soak tests
- [ ] Recurring lint-sweep
- [ ] CGo-enabled CI job
- [ ] Investigate TestRun_Postgres_Recovery

---

## D) TOTALLY FUCKED UP / QUALITY CONCERNS

### D1. Lint Count INCREASED — Net Negative Lint Impact

I refactored 3 functions to reduce cyclomatic complexity (gocyclo: 3→0 for those functions), but the extracted helper functions introduced NEW lint issues:
- 12 new wrapcheck violations (extracted functions return interface errors)
- 12 new varnamelen violations (extracted parameters use short names)
- 9 new wsl_v5 formatting violations

**Net result:** 66 → 101 issues (+35). The refactoring traded gocyclo compliance for wrapcheck/varnamelen/wsl violations. The mechanical cleanup pass (nolint directives) would bring this back down, but it was not done.

**Lesson:** When extracting functions to reduce gocyclo, immediately add nolint directives to the extracted helpers for wrapcheck/varnamelen, or the lint count goes UP even though code quality improved.

### D2. Pebble Raw Reader Appears to Already Exist

Prior session (or auto-commit daemon) already implemented `pebbleengine/raw_reader.go` and `sse_replay_test.go`. The TODO_LIST says "Pebble: implement RawValueReader + RawScanReader" but the file already exists at commit `f0ffebba`. This needs verification — the TODO item may be stale.

### D3. Auto-Commit Daemon Committed Mid-Session

The auto-commit daemon committed at `f0ffebba` (06:30) during this session, including my work-in-progress edits. The commit message is generic ("enhance typed reader and Pebble engine integration") and doesn't accurately describe the PrefetchCache bug fix or the refactoring. The TODO_LIST was also committed with stale content.

---

## E) WHAT WE SHOULD IMPROVE

1. **Lint cleanup is the #1 blocker** — 101 issues. Most are mechanical (nolint directives). A 30-minute sweep would eliminate 80% of them.
2. **The refactoring approach should batch nolint directives** — Extract a function, immediately nolint it. Don't leave extracted functions un-nolint'd.
3. **The SSE multi-subscriber test has timing sensitivity** — Uses 200ms sleeps for connection setup and event propagation. Under CI load, this could flake. Should use channel-based synchronization instead.
4. **ScanPage should use Cursor.Encode()** — The current implementation returns `*Cursor` but doesn't integrate with the opaque base64 encoding from `Cursor.Encode()`. The caller must manually encode if they want HTTP-safe cursor strings.
5. **The agent tool hit a usage limit** — When attempting to use the agent tool for the lint cleanup pass, the request failed with "Usage limit reached for 5 hour." This blocked the lint cleanup work. Future sessions should do lint cleanup inline, not via agents.

---

## F) Next 50 Things to Get Done (Prioritized)

### Immediate (blocking — lint gate fails)
1. Add `//nolint:wrapcheck` to 31 unwrapped error returns in metaengine
2. Add `//nolint:varnamelen` to 26 short-named variables/params in metaengine
3. Fix 9 wsl_v5 formatting violations in metaengine
4. Fix 5 nestif issues in extracted helper functions
5. Fix 7 err113 dynamic error issues (use sentinel + %w)
6. Fix 4 sqlclosecheck issues (real bugs — unclosed *sql.Rows)
7. Fix gochecknoglobals for V1StabilizationChecklist
8. Fix gofumpt formatting issue
9. Fix recvcheck receiver type inconsistency
10. Fix prealloc suggestions (3 locations)
11. Fix revive issues (5)
12. Fix nlreturn issues (2)
13. Fix nonamedreturns (2)
14. Fix staticcheck issue (1)
15. Fix funlen issue (1)
16. Fix remaining gocyclo (1)
17. Fix unused jsonValue (nolint)
18. Fix errcheck (1 — defer eng.Close())
19. Run `nix fmt` to normalize all formatting after nolint additions

### cqrs-lint Quality
20. Dedicated unit tests for F018-F021 (fire on anti-pattern, no-fire on clean code)
21. Add suppression tests for C031-C034, P011-P012, D014-D015, A032, E016-E017, S010, F018-F021
22. Fix E010 (package qualifier → type info)
23. Fix E011 (name-counting → call-graph analysis)
24. Fix E013 (doesn't verify config struct type)
25. Fix E014 (detects wrong concept)
26. Implement library self-lint mode (auto-detect go-cqrs-lite module path)
27. Build import-alias resolution helper (qualifierToImportPath)
28. Fix F011 broad .Exec matching (needs receiver type checking)
29. Fix F009 timer detection (add time.Tick/time.After)
30. Fix F013 HTTP handler detection (cover chi/gin/echo/fiber)
31. Review C030 over-suppression
32. Audit S006 for substring false positives
33. Fix C017 stale doc/title
34. Narrow C032 scope (handler/projector only)

### Metaengine Features
35. Verify Pebble RawValueReader/RawScanReader (may already exist — D2)
36. Add Pebble to ADT matrix test
37. Pebble LayoutPlanner
38. SSE Last-Event-ID reconnection
39. metaengine-gen code generator
40. Schema enforcement at Plan() time
41. Soak test (10M events)
42. Chaos testing

### Metaengine Testing Gaps
43. Multi-engine tiering test (TieredStore fan-out)
44. SwapEngine data migration test
45. MigrateLayout ALTER TABLE test
46. WithTTL functional test

### CI/Infrastructure
47. Fix 3 flaky benchkit soak tests (testutil.RaceEnabled thresholds)
48. Recurring lint-sweep (gate daemon commits behind nix fmt)
49. CGo-enabled CI job for DuckDB
50. Investigate TestRun_Postgres_Recovery flake

---

## G) Questions I Cannot Answer Myself

1. **The Pebble RawValueReader/RawScanReader TODO appears to already be done** — `pebbleengine/raw_reader.go` exists and `sse_replay_test.go` exists at commit `f0ffebba`. Should I verify they're complete and remove the TODO items, or were these partial implementations that need completion?

2. **The Postgres engine and DuckDB engine are large new modules** (each requires a new `metaengine/pgengine/` or `metaengine/duckdbengine/` module with `go.mod`, JSONB/columnar pushdown, GIN indexes, etc.). Are these truly in scope for this TODO list, or should they be moved to ROADMAP as long-term vision? They each represent 2-4 days of work.

3. **The `metaengine-gen` code generator is a full new `cmd/metaengine-gen` module** that needs Go AST parsing, template generation, CLI scaffolding, and tests. Is this meant to generate typed Store methods from query declarations (like `cmd/cqrs-gen`), or something else? The TODO says "typed Store methods from query declarations" but the design is ambiguous — should it generate Go source code files, or runtime registration helpers?

---

_Session interrupted by usage limit on agent tool. 3 of 42 TODO items completed (PrefetchCache fix, MapUpdateTyped, 3x complexity refactors). 110 tests pass. Lint gate fails with 101 issues (up from 66 due to refactoring introducing new wrapcheck/varnamelen violations)._
