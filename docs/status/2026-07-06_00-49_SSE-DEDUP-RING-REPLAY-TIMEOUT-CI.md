# SSE Dedup Ring, Replay Timeout & CI Cleanup — Session Status

**Date:** 2026-07-06 00:49 CEST
**Scope:** `transport/http`, `watermill`, `scheduling`, CI compliance, API stability
**Trigger:** User asked for brutal self-review, comprehensive plan, and full execution of all remaining work from the SSE replay/dedup session

---

## a) FULLY DONE

### Bounded Dedup Ring — SSE (`transport/http/dedup_ring.go` + `dedup_ring_test.go`)

Replaced unbounded `map[string]struct{}` with a fixed-capacity ring buffer (`dedupRing`, 1024 entries, ~90KB). Provably correct: overlapping events at the replay→live boundary are always at the tail of the replay stream, bounded by the channel buffer size (100). 10x safety margin.

- `Add` and `Has` are O(1)
- `Has` is nil-safe (returns false on nil `*dedupRing`)
- Constructor falls back to default capacity instead of panicking
- 5 unit tests covering basic ops, eviction, duplicate add, nil safety, large wraparound

### Bounded Dedup Ring — CatchUpSubscriber (`watermill/dedup_ring.go` + `dedup_ring_test.go`)

Same ring buffer applied to `CatchUpSubscriber.replayIDs` (was also unbounded `map[string]struct{}`). Ring capacity 1024 (4x the 256-deep output channel). Both ring implementations now share an identical API surface: configurable capacity, nil-safe `Has`, `Len` method.

- 5 tests added (was zero — identical algorithm had zero test coverage)

### SSE Replay Timeout (`transport/http/sse.go` + `sse_replay.go`)

Added `WithReplayTimeout(d time.Duration)` — when the timeout fires mid-replay, the broker sends an `SSEReplayIncompleteEvent` advisory event and switches to live. The advisory has no `id` field so it doesn't advance the client's `Last-Event-ID`. Implemented via `context.WithTimeout` wrapping the replay context; both bounded and unlimited replay paths check `ctx.Err()`.

- Test: `TestSSEHandler_ReplayTimeout_SendsAdvisoryEvent` uses a `delayedJournal` wrapper

### File Split — CI 350-Line Compliance (`transport/http/sse_replay.go`)

Extracted `replayEvents` + `writeReplayBatch` into `sse_replay.go` (138 lines). `sse.go` is 283 lines (was 412).

### Lint Cleanup

| File                              | Fix                                                                                |
| --------------------------------- | ---------------------------------------------------------------------------------- |
| `scheduling/scheduler_test.go`    | Removed 5 `infertypeargs` warnings (`scheduling.New[string]` → `scheduling.New()`) |
| `watermill/trace_context_test.go` | Removed dead `consumeTimeout` var + `consumeTimeoutDuration` func                  |
| `transport/http/sse_test.go`      | Fixed `rangeint` + `fmtappendf` gopls hints                                        |

### AGENTS.md Updated

Added unlimited replay + `WithReplayTimeout` + `SSEReplayIncompleteEvent` pattern to the SSE Key Patterns section. Verified with `cmd/doc-check` — all 335 references valid.

### API Surface Golden File Updated

Regenerated `docs/api_surface.txt` twice: once for the 9 new SSE exports, once for the watermill `Len` method. Final count: 1724 exports verified.

### CI Compliance — Full Verification

| Check                     | Result                                                                            |
| ------------------------- | --------------------------------------------------------------------------------- |
| `nix run .#build`         | ✅ All 47 modules compile                                                         |
| `nix run .#lint`          | ✅ 0 issues across all modules                                                    |
| `nix run .#test`          | ✅ All modules pass (no `-race` flag in nix; race verified separately per-module) |
| `cmd/api-stability`       | ✅ 1724 exports verified                                                          |
| BuildFlow pre-commit hook | ✅ Passed 30/30 on every commit                                                   |
| File size check           | ✅ All production files ≤ 350 lines                                               |

### Committed and Pushed

6 commits pushed to `origin/master`:

| Commit     | Description                                                    |
| ---------- | -------------------------------------------------------------- |
| `6dad501e` | API surface golden file update (9 new exports)                 |
| `24ae7c58` | Remove panic from dedupRing constructor                        |
| `f469fd37` | Unify watermill dedupRing with SSE version + add tests         |
| `f957c937` | Document WithReplayTimeout in AGENTS.md                        |
| `6d404977` | Main feature: bounded dedup ring, replay timeout, lint cleanup |
| `e9dc935f` | Final API surface update for `Len()` method                    |

---

## b) PARTIALLY DONE

### CatchUpSubscriber journal read still unbounded

The dedup map is now bounded (ring), but `replayPhase` in `watermill/catchup_subscriber.go:183` still calls `s.journal.ReadFrom(ctx, after, 0)` which materializes the entire journal tail into one `[]event.Event` slice. The SSE broker solves this with batched streaming (`sseReplayBatchSize = 500`). Applying the same pattern to CatchUpSubscriber would change the replay→live handoff sequencing.

### SSE replay metrics are span attributes, not OTel instruments

The replay span records `cqrs.sse.last_event_id`, `cqrs.sse.replay_status`, and `cqrs.event.count`. Richer metrics (replay_duration histogram, dedup_hits counter, replay_incomplete_total) would need `Int64Counter`/`Float64Histogram` instruments for dashboard use.

---

## c) NOT STARTED

- **No SQL/Redis `TimerStore`** — `scheduling/` only ships `MemoryTimerStore`
- **No client push for truly-offline mobile** — SSE requires live connection, no APNs/FCM
- **No conflict resolution for offline writes** — application logic
- **No per-event-type SSE filtering** — clients receive all events
- **No SSE authentication middleware example**

---

## d) TOTALLY FUCKED UP!

### The original dedup test was a lie (historical — now fixed)

The old `TestSSEHandler_ReplayDedup_NoDuplicates` sent `Last-Event-ID: ""` so replay never ran. The dedup code path was never exercised. **Fixed** — the rewritten test sends a real Last-Event-ID and verifies actual replay→live dedup.

### Dead code in `watermill/trace_context_test.go` shipped undetected

`consumeTimeout` var + `consumeTimeoutDuration` func were never referenced by anything. Pure dead weight that passed lint for who knows how long. **Removed.**

### I almost shipped a panic in library code

The first version of `dedupRing.newDedupRing(capacity)` panicked if `capacity <= 0`. This violates the project's "errors as values, no panics in library code" principle. Caught during self-review, replaced with a fallback to default capacity.

### I forgot to update the API surface golden file — twice

First: added 9 new exports (`DefaultSSEReplayLimit`, `SSEReplayIncompleteEvent`, `WithReplayTimeout`, dedupRing methods) without updating `docs/api_surface.txt`. CI would have failed. Second: after unifying the watermill ring with a `Len()` method, missed that export too. Both caught by running the api-stability checker manually.

---

## e) WHAT WE SHOULD IMPROVE!

1. **Batched streaming in `CatchUpSubscriber.replayPhase`** — the dedup ring is bounded but the journal read materializes everything. Port the SSE batch streaming pattern.
2. **SSE replay metrics as OTel instruments** — promote span attributes to real counters/histograms for dashboards.
3. **`delayedJournal` test helper should be promoted** — from `sse_test.go` to `eventtest` or `testutil` for reuse.
4. **`handleEvent` fanout holds `RLock` during channel sends** — a slow client that blocks on `ch <- evt` holds the lock. Worker pool or per-client goroutine would decouple.
5. **`sseReplayBatchSize = 500` is hardcoded** — for very large payloads (1MB+), 500 events per batch = 500MB in memory. A byte-budget alternative would be safer.
6. **The dedupRing is duplicated across two packages** — `transport/http` and `watermill` have identical implementations. Extracting to a shared package would mean a new module + dependency wiring in a repo with 47 `go.mod` files.

---

## f) Up to 25 things we should get done next

| #   | Priority | Task                                                             | Impact                                |
| --- | -------- | ---------------------------------------------------------------- | ------------------------------------- |
| 1   | **P1**   | Port batched streaming to `CatchUpSubscriber.replayPhase`        | Bounds memory for large journals      |
| 2   | **P2**   | Implement `SQLTimerStore` in `scheduling/`                       | Durable deadlines, production-ready   |
| 3   | **P2**   | Promote `delayedJournal` test helper to `eventtest`/`testutil`   | Reusable context-cancellation testing |
| 4   | **P2**   | Add SSE replay integration test with real `MemoryStore`          | Catches store-specific replay bugs    |
| 5   | **P2**   | Fix `watermill/go.mod` tidy issue (`schema/v3` zero-revision)    | Clean `go mod tidy` without `-e`      |
| 6   | **P3**   | Add SSE replay metrics as OTel instruments                       | Dashboard-ready observability         |
| 7   | **P3**   | Document SSE vs CatchUpSubscriber decision matrix in SKILL.md    | Consumer guidance                     |
| 8   | **P3**   | Add `example/` showing SSE + offline client reconnection         | Usage demo                            |
| 9   | **P3**   | Byte-budget replay batching (replace fixed `sseReplayBatchSize`) | Memory safety for large payloads      |
| 10  | **P3**   | Extract dedupRing to shared package (resolve duplication)        | DRY — currently in 2 modules          |
| 11  | **P4**   | Investigate `handleEvent` fanout via worker pool                 | Performance at scale                  |
| 12  | **P4**   | Add backpressure: when client channel full, drop oldest          | Prevents slow-client memory bloat     |
| 13  | **P4**   | Add `SSEBroker.Stats()` per-client lag                           | Debugging                             |
| 14  | **P4**   | Consider WebSocket transport alongside SSE                       | Bidirectional needs                   |
| 15  | **P4**   | SSE `RemoveClient` channel lifecycle (close with guard)          | Reduces channel accumulation          |
| 16  | **P5**   | SSE compression (gzip + `Content-Encoding: gzip`)                | Bandwidth                             |
| 17  | **P5**   | Per-event-type SSE filtering (`user.*` subscription)             | Bandwidth                             |
| 18  | **P5**   | SSE authentication middleware example                            | Security                              |
| 19  | **P5**   | Connection draining on `broker.Close()` with grace period        | Clean shutdown                        |
| 20  | **P5**   | SSE `retry:` field auto-tuning                                   | UX                                    |
| 21  | **P5**   | `dedupRing` benchmark (Add/Has under contention)                 | Performance characterization          |
| 22  | **P5**   | Configurable `sseDedupRingCapacity` via option                   | Tunability                            |
| 23  | **P5**   | Add `dedupRing.Len()` to OTel span                               | Replay diagnostics                    |
| 24  | **P5**   | Property-based test for dedupRing (rapid)                        | Edge case coverage                    |
| 25  | **P5**   | SSE replay backfill endpoint (REST complement to SSE)            | Alternative catch-up path             |

---

## g) Top #1 question I can NOT figure out myself

**Should the `dedupRing` be extracted into a shared package, or is duplication acceptable?**

Both `transport/http/dedup_ring.go` and `watermill/dedup_ring.go` are now identical in algorithm and API. The only question is whether ~70 lines of identical code warrants a new shared module.

**The tradeoff:**

- **Extract:** Clean DRY, single tested implementation. But this repo has 47 `go.mod` files with strict per-module dependency budgets enforced by `scripts/check-module-layers.sh`. Adding a 48th module means wiring it into `go.work`, setting deps for both consumers, updating the layer checker, and updating `cmd/api-stability/main.go`'s module list.
- **Keep duplicated:** Each module is self-contained, no new dependency, no layer wiring. The algorithm is stable (ring buffer won't change). But if we find a bug in one, we must fix both.
- **Put in existing `kv/` module:** `kv/` is Layer 0 (leaf), both consumers are Layer 5. Direction works. But `kv/`'s identity is "KV store abstraction" — a ring buffer doesn't belong there conceptually.

This is a project philosophy question: is ~70 lines of identical code worth the dependency wiring overhead in a repo that already has 47 modules? The answer depends on how much the maintainers value zero duplication vs zero wiring overhead — and I can't make that call.
