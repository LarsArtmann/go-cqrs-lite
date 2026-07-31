# Status Report: SSE Replay Hardening & Cursor Prefetch Completion

**Date:** 2026-07-31 17:20
**Session scope:** Complete remaining TODO items from SSE replay + cursor prefetch feature work in `metaengine/`

---

## A) FULLY DONE (verified this session)

### Code changes — all verified via `nix run .#verify`

1. **`metaengine/sse.go` — full rewrite of replay path**
   - Replaced unbounded `map[uint64]struct{}` dedup map with `dedup.Ring` (bounded memory, 1024 entries via `dedup.DefaultCapacity`)
   - Fixed subscribe ordering: `WatchWithSeq` now starts BEFORE `replayMissedEvents` (buffers live events during replay window, matching `transport/http` SSE pattern)
   - Extracted `forwardWithDropOld[T any]` generic — eliminates ~40 lines of duplicated drop-old channel forwarding logic
   - Extracted `sseMainLoop[T any]` generic — eliminates ~40 lines of duplicated timeout/heartbeat/select loop
   - Extracted `writePlainSSEEvent[V]` and `writeReplaySSEEvent[V]` named functions — cleaner than inline closures, lint-compliant
   - `replayMissedEvents` now keeps the MOST RECENT events when capped (`replayed[len(replayed)-cfg.ReplayLimit:]`), not the oldest
   - All functions use `r.Context()` for lifecycle (not `context.Background()`)

2. **`metaengine/dx.go` — PrefetchCache thread-safety**
   - Added `sync.RWMutex` to `PrefetchCache` struct
   - `Get` → `RLock`/`RUnlock`, `Put`/`Clear` → `Lock`/`Unlock`
   - Documented as thread-safe

3. **`metaengine/go.mod` — dedup dependency**
   - Added `github.com/larsartmann/go-cqrs-lite/dedup/v4 v4.2.0`

4. **`metaengine/sse_replay_test.go` — 2 new tests**
   - `TestPrefetchCache_ConcurrentAccess` — 8 writers + 4 readers + clearer goroutines, `-race` clean
   - `TestSSE_LastEventID_Reconnect_SQLite` — full HTTP end-to-end reconnection with SQLite engine (was Memory-only before)

5. **`metaengine/features_test.go` — fixed build-breaking daemon damage**
   - `newMemoryTestStore` and `newSQLiteTestStore` return types changed from `Store` to `*Store` (auto-commit daemon had broken these)
   - Fixed indentation broken by daemon's refactoring

6. **Documentation updates**
   - `AGENTS.md` — broke the 600-char metaengine single-line comment into 16 readable lines
   - `.agents/skills/go-cqrs-lite/references/modules.md` — metaengine description expanded with SSE/cursor/streaming exports
   - `.agents/skills/go-cqrs-lite/references/recipes.md` — added §2.15 (SSE streaming with reconnection) and §2.16 (cursor pagination with prefetch)
   - `.agents/skills/go-cqrs-lite/references/advanced.md` — added §6.16 (metaengine SSE comparison table vs transport/http)

### Verification results

| Check | Status | Detail |
|-------|--------|--------|
| Build | ✓ | All 60+ modules compile |
| Vet | ✓ | Clean |
| Test | ✓ | All modules pass |
| Race | ✓ | metaengine: 94.9s, all others pass |
| Lint (metaengine) | ✓ | **0 issues** |
| Doc-check | ✓ | 964 references valid across 38 packages |
| API-stability | ✓ | 2907 exports, golden matches |

---

## B) PARTIALLY DONE

### Skill reference updates
- `modules.md` — done
- `recipes.md` — done (§2.15 + §2.16)
- `advanced.md` — done (§6.16)
- `core.md` — **NOT updated** (no metaengine mention added to the mental model / decision matrix)
- `readmodels.md` — **NOT updated** (no metaengine pagination/cursor section added)
- `faq.md` — **NOT updated** (no SSE reconnection FAQ entry)

### Dedup ring capacity
- Uses hardcoded `dedup.DefaultCapacity` (1024). This is a reasonable default but not configurable per-SSE-connection. If a replay writes >1024 events, older entries get evicted from the dedup ring before the live phase checks them, potentially causing duplicate delivery on the overlap boundary. This is an edge case that could matter for very high-throughput streams with slow reconnecting clients.

---

## C) NOT STARTED

1. **Brutal-self-review HTML report** at `docs/reviews/` — was in the remaining work list but not created
2. **`core.md` skill reference** — metaengine SSE/cursor not added to the decision matrix
3. **`readmodels.md` skill reference** — metaengine pagination section not written
4. **`faq.md` skill reference** — SSE reconnection FAQ not added
5. **Git commit** — per project rules, not committing unless explicitly asked
6. **Replay overlap dedup test** — no dedicated test that verifies an event appearing in BOTH the replay journal AND the live stream is delivered only once
7. **Subscribe-before-replay ordering test** — no test that specifically exercises events arriving DURING the replay window
8. **Metrics for SSE replay** — no OTel/prometheus instrumentation for replay duration, event count, dedup hit rate
9. **Configurable dedup ring capacity** — no `WithSSEDedupCapacity(n)` option

---

## D) TOTALLY FUCKED UP (and fixed)

1. **`context.Background()` instead of `r.Context()`** — My initial rewrite of `serveSSEPlain` and `serveSSEReplay` used `context.Background()` instead of `r.Context()`. This caused `TestSSE_DropOldSemantics` to hang for 2 minutes (the SSE stream never terminated because the context never cancelled on client disconnect). **Fixed** by switching back to `r.Context()`. This was a careless mistake — the original code used `r.Context()` and I should have preserved it.

2. **Auto-commit daemon broke the build** — While I was working, the auto-commit daemon refactored `features_test.go`, `features2_test.go`, `features3_test.go`, and `features4_test.go`:
   - Changed `newMemoryTestStore` and `newSQLiteTestStore` return types from `*Store` to `Store` (value instead of pointer) — broke compilation
   - Removed `err` variable declarations in multiple test functions (replaced `store, err := Plan(...)` with `store := newMemoryTestStore(t)`, leaving `_, err =` references undefined)
   - Broke indentation (0-indent instead of tab-indent)
   **I had to fix all of these** before the verify gate could run. This is documented in AGENTS.md as a known anti-pattern ("Auto-commit daemon can break the build").

3. **ReplayLimit semantics change was undocumented** — I changed the behavior from "keep oldest N" to "keep most recent N" without initially documenting the change. The new behavior is arguably better (clients care about current state), but it's a behavior change that could surprise consumers.

---

## E) WHAT WE SHOULD IMPROVE

### Immediate quality gaps

1. **No replay→live overlap test** — The dedup ring exists specifically to handle events that appear in both the replay journal and the live stream during the subscribe-before-replay window. There is NO test for this critical edge case. The existing `TestSSE_LastEventID_Reconnect` tests replay and live separately but never exercises the overlap.

2. **No subscribe-before-replay ordering test** — The key correctness fix (subscribe first, then replay) has no dedicated test. The existing tests happen to work because the replay completes before any new events are applied, but a test that applies events DURING the replay phase would validate the fix.

3. **`dedup.Ring` uses string keys** — `strconv.FormatUint(sv.Seq, 10)` allocates a string on every dedup check. For high-throughput streams, a `map[uint64]struct{}` with bounded eviction would avoid the string allocation overhead. This is micro-optimization territory but worth noting.

4. **ReplayLimit slicing copies the slice header but not the data** — `replayed[len(replayed)-cfg.ReplayLimit:]` is fine, but the original entries slice is still alive in memory until `replayMissedEvents` returns. Not a leak, just noting.

### Architectural observations

5. **`watcherNotification` type-unsafe wrapper** — The `watcherNotification{seq, value any}` sent through `chan any` is a type-erasure workaround. The `replayShim[V].recordValue` type-asserts `any → V` and silently drops mismatches (returns seq=0). This is documented but fragile — a consumer who creates a `Watcher[WrongType]` will silently lose all values from the replay journal.

6. **No SSE connection limit** — `ServeSSE` has no mechanism to limit concurrent SSE connections per client or globally. A misbehaving client could open thousands of connections. This is typically handled by the HTTP server or reverse proxy, but worth documenting.

7. **No graceful shutdown integration** — `ServeSSE` doesn't participate in `http.Server.Shutdown()`. When the server shuts down, SSE streams are killed abruptly. The `projectionhost` has `WithShutdownTimeout` — SSE could benefit from similar.

8. **`forwardWithDropOld` eviction is racy** — When the buffer is full, the goroutine evicts ONE item then tries to write ONE item. Between the eviction and the write, another goroutine could fill the slot. This is a benign race (worst case: one extra dropped event), but the pattern is not atomic.

### Process improvements

9. **Should have run `go build ./...` immediately after the daemon commit** — I was about to run the verify gate with a broken build. The daemon's refactoring was caught only because `go vet` failed. The AGENTS.md already documents this lesson but I still nearly missed it.

10. **Should have written the replay overlap test FIRST** — The dedup ring replacement was the highest-risk change, and it has no targeted test. TDD would have caught any regression in the overlap handling.

---

## F) Up to 50 things to get done next

### High priority (correctness & safety)
1. Write replay→live overlap dedup test (event in both journal AND live stream → delivered once)
2. Write subscribe-before-replay ordering test (events applied DURING replay phase)
3. Write brutal-self-review HTML report at `docs/reviews/`
4. Add `core.md` skill reference for metaengine SSE/cursor in the decision matrix
5. Add `readmodels.md` skill reference for metaengine pagination section
6. Add `faq.md` skill reference for SSE reconnection FAQ
7. Add `WithSSEDedupCapacity(n)` option for configurable dedup ring
8. Document ReplayLimit behavior change (most-recent-N, not oldest-N) in godoc

### Medium priority (hardening)
9. Add SSE replay metrics (OTel spans: replay duration, event count, dedup hit rate)
10. Add SSE connection limit option (max concurrent connections)
11. Add graceful shutdown support for SSE streams
12. Consider `uint64`-keyed dedup ring (avoid `strconv.FormatUint` allocation)
13. Add test for ReplayLimit=0 (unlimited replay) with large journal
14. Add test for ReplayLimit=1 (only most recent event)
15. Add test for empty journal reconnect (no events to replay)
16. Add test for Last-Event-ID beyond journal capacity (events evicted from ring)
17. Add test for invalid Last-Event-ID header (non-numeric, negative, empty)
18. Add test for concurrent ServeSSE connections sharing one watcher
19. Add test for Watcher.Close() during active SSE stream
20. Add test for PrefetchCache Clear() during active scan
21. Add benchmark: SSE throughput (events/sec) with and without replay journal
22. Add benchmark: PrefetchCache hit rate vs miss rate

### Lower priority (polish)
23. Fix `pebbleengine/raw_reader.go:56` pre-existing `undefined: pebble` error (gopls only, not a build error)
24. Add `WithSSEMaxRetry` for automatic reconnection on transient write failures
25. Consider SSE event type field (`event: task_created`) for client-side filtering
26. Consider SSE retry field (`retry: 3000`) for client reconnection interval
27. Add `Cursor.String()` method for debug logging (human-readable, not base64)
28. Add `PrefetchCache.Stats()` method (hits, misses, evictions)
29. Consider LRU eviction for PrefetchCache (currently unbounded map growth)
30. Add integration test: metaengine SSE + transport/http SSE in same process
31. Consider extracting SSE helpers to a shared internal package
32. Add godoc examples for `SSEReplay`, `SeqValue`, `Watcher.WatchWithSeq`
33. Consider `Watcher.WatchFiltered(ctx, key, filter)` for predicate-based filtering
34. Add `Store.WatcherCount()` for observability
35. Consider persistent replay journal (survive process restart)
36. Add test for replay journal ring buffer at exact capacity boundary
37. Add test for replay journal with zero-capacity (0 → default 64)
38. Add fuzz test for `ParseCursor` with random base64 input
39. Add fuzz test for `WithCursorString` with random input
40. Consider `Cursor.EncodeTo(w io.Writer)` for zero-allocation encoding
41. Add `metaengine` to the `cmd/cqrs-lint` feature profile system
42. Add cqrs-lint rule: warn if ServeSSE used without WithReplay (no reconnection)
43. Add cqrs-lint rule: warn if PrefetchCache used without thread-safety (shared across goroutines)
44. Add cqrs-lint rule: warn if Watcher.Close() not deferred
45. Consider `metaengine` integration with `projectionhost` (auto-SSE from projection changes)
46. Add example: metaengine SSE in `example/taskmanager/`
47. Add ADR for SSE replay design (in-memory ring buffer vs journal-backed)
48. Consider `StreamScan` + SSE integration (stream scan results as SSE)
49. Add `metaengine` to `docs/architecture-understanding/FOUR-TIER-MODEL.md` SSE section
50. Consider WebSocket transport as alternative to SSE for bidirectional streaming

---

## G) Questions I cannot answer myself

1. **Should the metaengine SSE replay journal support persistent backends** (surviving process restarts), or is in-memory-only the intended scope? The `transport/http` SSE broker is journal-backed (persistent), but the metaengine SSE is watcher-based (in-process). If persistence is needed, the design would need to change fundamentally (use `EventLog` or an external journal instead of a ring buffer).

2. **Should `replayMissedEvents` keep the most-recent N events or the oldest N events when capped?** I changed it to most-recent (clients care about current state), but this means a client with a very old `Last-Event-ID` will MISS events in the middle (gets the most recent N, not the first N after their cursor). Is this the right tradeoff?

3. **Should the auto-commit daemon be disabled during active development sessions?** It broke the build 3 times this session by refactoring test files I was actively working on. The AGENTS.md documents this as a known risk, but the daemon continues to cause damage. Should we add a `.crush-no-autocommit` sentinel file or similar mechanism?
