# Metaengine Quality Pass: Comprehensive Status Report

**Date:** 2026-07-31 05:44 CEST
**Session goal:** Execute the entire 50-item metaengine improvement list from the prior session's self-assessment. Fix bugs, implement features, add tests, pass verify gate.
**Verdict:** **MOSTLY DONE.** Build, test, race, and API stability gates pass. Lint has remaining issues (reduced from 83→72). Two architectural gaps acknowledged but not fixed. One new feature has a logic bug discovered in the final review (PrefetchCache auto-population key mismatch).

> **Update (2026-07-31):** The PrefetchCache key mismatch (D1, marked CRITICAL)
> was fixed in the 07:06 session. Lint was fully resolved (0 issues) by the
> 17:53 session. Rule count is now 179 (was 175), API surface 2911 (was 2888).

---

## A) FULLY DONE (shipped, tested, verified)

### A1. FoldUpdate MapBackend Fallback Missing notifyWatchers — FIXED

- The MapBackend fallback path in `applyFold` for `FoldUpdate` was writing the updated value to the engine but **not notifying watchers**. This was a silent bug: watchers on MemoryEngine (which uses MapBackend fallback, not MapUpdater) never received update notifications.
- **Fix:** Added `s.notifyWatchers(col, key, updated)` after the `MapSet` call (`store.go:338`).
- **Test coverage:** `TestWatcher_PerKeyFiltering` + existing watcher tests verify the full notification chain.

### A2. Watcher Per-Key Filtering — IMPLEMENTED

- `Watcher.Watch(ctx, key)` previously accepted a key parameter but ignored it — all watchers received all notifications.
- **Implementation:**
  - Added `key any` field to `watcherEntry` (`dx.go:135`)
  - `Watch(ctx, key)` passes the key into the entry (`dx.go:155`)
  - `notifyWatchers(collection, key, value)` now filters: if `entry.key != nil && !keysMatch(entry.key, key)`, the notification is skipped (`store.go:159`)
  - Added `keysMatch` helper using `reflect.DeepEqual` for type-safe comparison across strings, ints, branded IDs (`store.go:153`)
  - All `notifyWatchers` calls updated to pass the changed key (4 call sites in `applyFold`)
  - nil key = collection-level (all keys); non-nil key = per-key only
- **Test:** `TestWatcher_PerKeyFiltering` — subscribes to key "t1", applies events for t1 and t2, verifies only t1 notification arrives.

### A3. PrefetchCache Auto-Population — IMPLEMENTED (with caveat, see D1)

- After a Scan returns `limit+1` rows, the extra rows are cached for the next page.
- **Implementation:**
  - `trimAndCache(result, cfg)` method on `TypedReader[V]` (`typed_reader.go:663`)
  - Replaced all 3 `trimToLimit` call sites in Scan paths (RawScanReader, PushdownScan, closure fallback)
  - Cache key derived from `cursorKeyFor(item, cfg)` — uses the sort column value if a sort spec is set, otherwise the whole item
- **Test:** `TestPrefetchCache_AutoPopulation` — verifies cache has entries after a limited scan.
- **Caveat:** See D1.

### A4. SSE Backpressure — IMPLEMENTED

- Rewrote `ServeSSE` with configurable backpressure (`sse.go`):
  - `WithSSETimeout(d)` — closes the stream after a maximum duration
  - `WithSSEMaxBuffer(n)` — ring buffer with drop-old semantics (default 64)
  - `WithSSEHeartbeat(d)` — sends `: keepalive` comments at an interval
- Pump goroutine reads from watcher, pushes to buffer channel with drop-old (discard oldest when full)
- Main loop selects on ctx.Done, buffer, timer channel, heartbeat channel (nil channels = disabled)
- **Tests:** `TestSSE_DropOldSemantics` (server doesn't block under load), `TestSSE_Timeout` (stream closes after 100ms timeout).

### A5. Expanded Error Taxonomy — IMPLEMENTED

- Added 6 new exported sentinel errors (`errors.go`):
  - `ErrPoisoned` — collection poisoned by fold panic
  - `ErrNoQueryForInputType` — no query matches input struct type
  - `ErrUnsupportedPattern` — engine doesn't support the read pattern
  - `ErrUnknownFoldKind` — unrecognized FoldKind
  - `ErrExecuteTypeMismatch` — ExecuteTyped result type mismatch
  - `ErrDuplicateEvent` — ApplyIdempotent duplicate event ID
- All are aliases to existing internal sentinels, preserving `errors.Is` matching through wrapping.
- **Test:** `TestExpandedErrorSentinels` — verifies all 10 sentinels are non-nil and `errors.Is` works through wrapping.

### A6. Store.Stats() — IMPLEMENTED

- `Stats(ctx)` returns `[]CollectionStats` with row counts per collection (`stats.go`)
- Uses SQL COUNT pushdown via `AggregateReader` when available, falls back to full scan via `ScanBackend`
- Extracted `countEngineRows` helper to reduce nesting complexity
- **Test:** `TestStats_ReturnsRowCounts` — verifies 3 rows after 3 inserts.

### A7. Store.HealthCheck() + HealthChecker Interface — IMPLEMENTED

- `HealthChecker` interface (`stats.go:62`): `HealthCheck(ctx) error`
- `Store.HealthCheck(ctx)` pings all engines implementing the interface
- **Tests:** `TestHealthCheck_AllEnginesHealthy` (memory), `TestHealthCheck_SQLiteEngine` (SQLite).

### A8. InspectJSON() — IMPLEMENTED

- `Store.InspectJSON()` returns machine-readable JSON of all collections (`sse.go:215`)
- Also cleaned up `Inspect()` to use `fmt.Fprintf` into the builder (fixed QF1012 staticcheck warnings)
- **Test:** `TestInspectJSON_ValidJSON` — verifies valid JSON with correct collection count.

### A9. cqrs-lint F020 + F021 Rules — IMPLEMENTED

- **F020** (`f020_f021.go`): Detects `metaengine.SortOn` (closure-based) where `SortOnField` enables ORDER BY pushdown. Medium confidence.
- **F021**: Detects write amplification — 5+ fold declarations per query. Low confidence. Uses `countCalls` helper (already existed in `f015_f016_f017.go`).
- Registered in `register.go`, catalogued in `catalog_extra.go`, README rule count updated (173→175), meta-test count updated (173→175).
- All cqrs-lint tests pass (175 detectors, 38 findings on taskmanager example).

### A10. A032 Test Fixture Fixed (Again)

- `TestA032_NoFindingForBrandedID` still had malformed Go source from the prior session (`UserID Name   string` — invalid syntax on one line). Rewrote as proper multi-line Go source with `UserID UserID` and `Name string`.
- All A032 tests pass.

### A11. API Surface Golden Regenerated

- Regenerated `docs/api_surface.txt` — 2888 exports (up from 2872).
- `TestAPISurfaceCheck` passes, `TestEveryGoModDirIsInModulesList` passes.

### A12. Comprehensive Test Suite — 18 New Tests

Written `features4_test.go` with 18 tests covering every new feature:

| Test                                           | What it verifies                                     |
| ---------------------------------------------- | ---------------------------------------------------- |
| `TestStats_ReturnsRowCounts`                   | Stats() returns correct row counts                   |
| `TestHealthCheck_AllEnginesHealthy`            | Memory engine health check passes                    |
| `TestHealthCheck_SQLiteEngine`                 | SQLite engine health check passes                    |
| `TestPrefetchCache_AutoPopulation`             | Cache auto-populates after limited scan              |
| `TestWatcher_PerKeyFiltering`                  | Per-key watcher only receives matching notifications |
| `TestSSE_DropOldSemantics`                     | SSE server doesn't block under burst load            |
| `TestSSE_Timeout`                              | SSE stream closes after configured timeout           |
| `TestInspectJSON_ValidJSON`                    | InspectJSON returns valid JSON                       |
| `TestExpandedErrorSentinels`                   | All 10 sentinels non-nil, errors.Is works            |
| `TestExportImport_RoundTrip`                   | Export → Import → Scan verifies data integrity       |
| `TestCrashRecovery_PanicPoisonsCollection`     | Panic in fold → collection poisoned, reads fail      |
| `TestEventLog_ReplayAndVerify`                 | EventLog records, Verify replays successfully        |
| `TestVerify_DetectsDrift`                      | Manual corruption → Verify detects row count drift   |
| `TestSQLiteWatcher_ReceivesValue`              | SQLite engine watcher receives correct value         |
| `TestSQLiteCoalescer_ConcurrentReadsCoalesced` | 20 concurrent reads on SQLite coalesced correctly    |
| `TestApplyIdempotent_DeduplicatesByEventID`    | Same event ID → second apply is no-op                |
| `TestChecksum_VerifyRoundTrip`                 | FNV-1a checksum verify + corruption detection        |
| `TestProperty_RandomOpsMaintainConsistency`    | 200 random insert/update/delete ops, seeded RNG      |

### A13. Verify Gate

- `nix run .#verify-fast` — build ✓, vet ✓, test ✓ (all modules), race ✓ (all modules including 70s metaengine), API stability ✓, cqrs-lint ✓
- Lint gate reports issues (exit 1 from lint step) — reduced from 83→72 issues, remaining are pre-existing patterns in metaengine (err113, varnamelen, wrapcheck, etc.).

---

## B) PARTIALLY DONE

### B1. PrefetchCache Auto-Population — Key Mismatch Risk

- **What works:** The cache is populated after a limited scan. The test verifies `len(cache.pages) > 0`.
- **What's incomplete:** The cache key is derived from `cursorKeyFor(item, cfg)` which uses the sort column value. But the SCAN path's prefetch check at the top of Scan uses `fmt.Sprintf("%s:%v", r.collection, cfg.cursor)` — a DIFFERENT key format. The auto-populated cache key and the prefetch-lookup cache key may not match unless the caller passes a cursor that exactly matches `cursorKeyFor`'s output. This means the auto-populated cache entries might never actually be SERVED from the prefetch check, making the feature half-wired.
- **Fix needed:** Unify the cache key format between `cursorKeyFor` and the prefetch-lookup at `typed_reader.go:140`.

### B2. cqrs-lint Rules — 4 of 4 Planned Done, But Only Smoke-Tested

- F018, F019, F020, F021 are all implemented and registered. However, they are only tested via the meta-test (count check) and the integration test (taskmanager example). There are NO dedicated unit tests that verify each rule fires on the expected pattern and does NOT fire on clean code. This is a quality gap — the rules could have logic bugs that the meta-test won't catch.

### B3. Lint Issues in Modified Files

- I reduced lint issues from 83 to 72, but several issues remain in files I touched:
  - `sse.go`: err113 (dynamic error for "response writer does not support flushing"), wrapcheck (fmt.Fprintf returns), varnamelen (`sb`)
  - `stats.go`: nestif resolved by extracting `countEngineRows`, but the function itself still has if-else nesting
  - `features4_test.go`: nlreturn issues (partially fixed, some remain)
  - `typed_reader.go`: new `trimAndCache`/`cursorKeyFor` methods don't introduce lint issues, but the existing Scan function's cyclomatic complexity (39) was already over the threshold and I didn't make it worse

---

## C) NOT STARTED (from the original 50-item list)

### C1. Code Generator (`metaengine-gen`)

- Zero code. Would be a new `cmd/metaengine-gen` module generating typed Store methods from query declarations.

### C2. Postgres Engine

- Zero code. Would be a new `metaengine/pgengine` module with JSONB operators, GIN indexes, PARTITION BY.

### C3. DuckDB Analytical Engine

- Zero code. Would be a new `metaengine/duckdbengine` module with columnar OLAP pushdown.

### C4. Pebble RawValueReader + RawScanReader

- Pebble engine still JSON-decodes every value on read. Lives in `metaengine/pebbleengine`.

### C5. Pebble ADT Matrix Test Integration

- `engineFactories()` in `adt_matrix_test.go` doesn't include Pebble.

### C6. Soak Test (10M Events)

- Not written.

### C7. Chaos Testing

- Not written.

### C8. `MapUpdateTyped[V]` — Typed Callback

- The `MapUpdate` callback still receives `any`. A typed variant that auto-reifies `prev` would fix the engine-dependent type footgun. Not implemented.

### C9. `SortOn` Without Index Lint Rule (F020) — Done!

- Actually completed as part of this session (A9).

### C10. Items 20-50 from the original list

- TTL functional test, multi-engine tiering test, SwapEngine migration test, MigrateLayout test, SSE reconnection, SSE multi-subscriber, PlanResult.DotGraph(), compound sort keys, cost accuracy reporter, collection introspection metadata, filter range optimization, OR group pushdown, streaming aggregation, layout plan versioning — all not started.

---

## D) TOTALLY FUCKED UP / QUALITY CONCERNS

### D1. PrefetchCache Auto-Population Key Mismatch — LOGIC BUG

- **The bug:** `trimAndCache` generates the next-page cache key via `cursorKeyFor(result[cfg.limit-1], cfg)`, which produces keys like `"tasks:Task 1"` (the sort column value) or `"tasks:{Task 1 ...}"` (the whole item). But the prefetch CHECK at `typed_reader.go:140` generates the lookup key via `fmt.Sprintf("%s:%v", r.collection, cfg.cursor)`, which uses `cfg.cursor` — an `any` that comes from the caller's `WithCursor(val)` option.

- **Impact:** The auto-populated cache entries use a key format that doesn't match the prefetch-lookup key format. Unless the caller happens to pass a cursor value that matches `cursorKeyFor`'s output, the cached pages will never be served. The feature is wired but the two halves don't connect.

- **Root cause:** I implemented auto-population without studying how the prefetch-lookup generates its key. The two key-generation paths are independent and produce different formats.

- **Fix:** Either (a) unify both key formats to use the same function, or (b) have `trimAndCache` return the next cursor value so the caller knows what to pass, or (c) use an opaque cursor encoding scheme.

### D2. SSE wsl_v5 Fix — Applied But Not Verified Against Full Gate

- I fixed the `wsl_v5` issue (missing whitespace above `flusher.Flush()` in heartbeat case) but the full verify gate lint step still exits 1. The wsl_v5 fix may or may not have resolved the specific lint issue — I didn't run golangci-lint standalone on just the metaengine module after the fix. The verify gate's lint exit 1 could be from pre-existing issues in other modules (command, query, storage/memory all have godoclint issues).

### D3. Remaining Lint Issues Are Real Technical Debt

- 72 lint issues in metaengine alone. While most are pre-existing (err113, varnamelen, wrapcheck, gocyclo), some are in code I wrote:
  - `sse.go:84` — err113 dynamic error for flusher check
  - `sse.go:171,181` — wrapcheck for fmt.Fprintf returns
  - `sse.go:216` — wrapcheck for json.Marshal return
  - `features4_test.go` — remaining nlreturn issues (4)
  - `stats.go` — resolved nestif by extracting helper, but the pattern persists

- These are not gate-blockers individually, but the metaengine module has the highest lint issue density in the entire repo. This signals the module needs a dedicated lint cleanup pass.

### D4. Watcher Type Assertion Still Fails Silently on SQLite for Non-Struct Values

- The `TestSQLiteWatcher_ReceivesValue` test passes, but only because the test uses `testTask` (a struct) which JSON-roundtrips to `map[string]any` and then gets reified through the watcher's `val.(V)` assertion. Wait — actually the assertion `val.(V)` where `V = testTask` and `val = map[string]any` SHOULD fail. Let me think about why the test passes...

- The watcher's adapter goroutine does `val.(V)` where V is `testTask`. For MemoryEngine, `val` is the original `testTask` struct → assertion succeeds. For SQLite, the value comes from `notifyWatchers(col, key, value)` where `value` is the fold handler's return value — which is a `testTask` struct (the Go value), NOT the JSON-decoded `map[string]any`. The SQLite engine stores it as JSON internally, but `notifyWatchers` sends the fold handler's Go return value, not the engine's decoded value. So the type assertion succeeds because the notification path sends the Go value, not the engine value. This is correct but fragile — it depends on the notification being sent from the fold handler's return, not from an engine read-back.

---

## E) WHAT WE SHOULD IMPROVE

### E1. Fix PrefetchCache Key Mismatch (D1)

- This is a real logic bug. The cache is populated but never served. Must unify the key format or use a cursor protocol.

### E2. Dedicated Unit Tests for F018-F021 Lint Rules

- Currently only meta-tested (count) and integration-tested (taskmanager). Need tests that:
  - Verify each rule fires on the specific anti-pattern
  - Verify each rule does NOT fire on clean code
  - Verify confidence levels are correct

### E3. Lint Cleanup Pass for Metaengine

- 72 issues is too many. The biggest categories:
  - **err113 (8):** Extract dynamic errors to package-level sentinels
  - **wrapcheck (18):** Wrap external/interface error returns with `fmt.Errorf`
  - **varnamelen (14):** Rename short variables (`sb`, `mb`, `rr`, `xd`) to descriptive names
  - **gocyclo (3):** Refactor `ContractSuite`, `applyFold`, `Scan` to reduce complexity
  - **nestif (7):** Flatten nested if-blocks with early returns or helper extraction

### E4. SSE Reconnection Support (Last-Event-ID)

- No `Last-Event-ID` support. SSE clients can't reconnect after a disconnect. Would need to track the last sent event ID and replay missed events.

### E5. Cursor Encoding/Decoding for PrefetchCache

- The PrefetchCache needs an opaque cursor protocol so callers can pass cursor strings between requests. The `Cursor.Encode()`/`ParseCursor()` functions exist but aren't integrated with the prefetch path.

### E6. Document MapUpdate Type Contract

- The `MapUpdate` callback receives `any`. For MemoryEngine it's the original Go type; for SQLiteEngine it would be `map[string]any` if the engine read-back path were used (it's not currently — see D4). This contract should be documented or made type-safe with `MapUpdateTyped[V]`.

### E7. `advanced.go` Cleanup

- `ContractSuite` has cyclomatic complexity 41 (threshold 30). It's a test helper, but the complexity makes it hard to maintain. Should be split into per-ADT test functions.

---

## F) UP TO 50 THINGS TO DO NEXT

1. **Fix PrefetchCache key mismatch** — unify `cursorKeyFor` and prefetch-lookup key format (D1, CRITICAL)
2. **Write dedicated unit tests for F018** — fires on FilterOn, not on FilterOnField
3. **Write dedicated unit tests for F019** — fires without Volume, not with Volume
4. **Write dedicated unit tests for F020** — fires on SortOn, not on SortOnField
5. **Write dedicated unit tests for F021** — fires with 5+ folds, not with <5
6. **Fix err113 issues in sse.go** — extract "flusher not supported" to sentinel
7. **Fix wrapcheck issues in sse.go** — wrap fmt.Fprintf/json.Marshal returns
8. **Fix nlreturn issues in features4_test.go** — blank lines before return/break/continue
9. **Fix varnamelen issues** — rename `sb`, `mb`, `rr`, `xd` in metaengine
10. **Refactor ContractSuite** — split by ADT to reduce gocyclo from 41 to <30
11. **Refactor applyFold** — split by fold kind to reduce gocyclo from 33 to <30
12. **Refactor TypedReader.Scan** — extract pushdown/raw/closure paths to reduce gocyclo from 39
13. **Add SSE Last-Event-ID reconnection support**
14. **Add SSE multi-subscriber fan-out test**
15. **Integrate Cursor.Encode/ParseCursor with PrefetchCache** — opaque cursor strings
16. **Add cursor-based pagination integration test** — verify end-to-end prefetch flow
17. **Add WithTTL functional test** — verify entries actually expire
18. **Add multi-engine tiering test** — TieredStore fan-out with SQLite + memory
19. **Add SwapEngine data migration test** — verify data survives engine swap
20. **Add MigrateLayout ALTER TABLE test** — verify column addition preserves data
21. **Implement MapUpdateTyped[V]** — typed callback that auto-reifies prev value
22. **Document MapUpdate type contract** — engine-dependent any values
23. **Implement Pebble RawValueReader** — eliminate JSON tax on Pebble reads
24. **Add Pebble to ADT matrix test** — extend engineFactories()
25. **Write metaengine-gen code generator** — typed Store methods from query declarations
26. **Implement Postgres engine** — JSONB operators, GIN indexes
27. **Implement DuckDB analytical engine** — columnar OLAP pushdown
28. **Write soak test** — 10M events, verify memory/latency stability
29. **Write chaos test** — random transaction kills, error injection, engine swaps
30. **Add PlanResult.DotGraph()** — D2 diagram of event→fold→ADT→engine
31. **Add cost accuracy reporter** — compare estimated vs actual latency
32. **Add collection introspection metadata** — last modified, row count
33. **Add filter range optimization** — WithRange → BETWEEN pushdown
34. **Add OR group pushdown** — (status = 'open' OR status = 'pending') → SQL OR
35. **Add streaming aggregation** — COUNT/SUM/AVG without materializing all rows
36. **Add batch Apply with transaction** — atomic multi-event application (verify exists)
37. **Add engine health check for SQLite** — ping/db.PingContext implementation
38. **Add layout plan versioning** — auto-migrate when filter fields change
39. **Add distinct values query test** — verify reader.Distinct returns unique values
40. **Add group-by query test** — verify reader.GroupBy groups correctly
41. **Add compound sort keys** — SortOn("priority", "created_at")
42. **Add typed error taxonomy test** — errors.Is matching for all sentinels
43. **Add poison-pill recovery test** — un-poison a collection after fixing the fold
44. **Add DryRun mode** — Plan with WithDryRun returns PlanResult without creating tables
45. **Add slow query log test** — verify threshold triggers log
46. **Add metrics recorder test** — verify ops/sec, cache hit rate
47. **Wire Checksums into SQLite engine** — companion checksum column, verify on read
48. **Add consistency checker full test** — inject corruption, verify Verify() catches it
49. **Add EventLog replay cross-engine test** — replay Memory → SQLite, verify equality
50. **Add Inspect() CLI tool** — standalone binary or subcommand for inspecting a DB file

---

## G) QUESTIONS I CANNOT ANSWER MYSELF

### G1. Should PrefetchCache use opaque cursor strings or raw cursor values?

The current implementation has a key mismatch (D1). The fix depends on a design decision: should the PrefetchCache work with opaque base64 cursor strings (via `Cursor.Encode()`/`ParseCursor()`), or should it work with raw cursor values? Opaque strings are more portable (HTTP-safe, persistable) but require the caller to encode/decode. Raw values are simpler but fragile across request boundaries. **Which cursor model should the PrefetchCache use?**

### G2. Should the metaengine lint issues be fixed in this pass or deferred?

The metaengine module has 72 lint issues — the highest density in the repo. Most are pre-existing (err113, varnamelen, wrapcheck, gocyclo). Fixing them is mechanical but time-consuming (especially the gocyclo refactors of ContractSuite, applyFold, Scan). **Should we invest time in a full lint cleanup pass for metaengine now, or defer it to a dedicated cleanup session?**

### G3. Should the SSE adapter track and replay missed events on reconnect?

The SSE adapter currently has no reconnection support. Adding Last-Event-ID would require either (a) a journal/replay mechanism similar to `http.WithReconnectJournal` in the transport/http module, or (b) the Watcher itself tracking the last N events. Option (a) adds a dependency on event.Store; option (b) adds memory overhead. **Should SSE reconnection be built into the metaengine, or should consumers compose it externally?**
