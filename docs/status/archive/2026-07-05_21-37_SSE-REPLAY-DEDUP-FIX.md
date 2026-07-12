# SSE Replay & Dedup Fix — Session Status

**Date:** 2026-07-05 21:37 CEST (updated 22:15)
**Scope:** `transport/http` SSE broker replay/dedup overhaul + `watermill` CatchUpSubscriber
**Trigger:** User asked about offline client handling, identified the 1000-event replay cap as broken

---

## a) FULLY DONE

### SSE Replay Overhaul (`transport/http/sse.go` + `sse_replay.go` + `sse_test.go`)

Three bugs fixed, all in the SSE reconnection subsystem:

| #   | Bug                                                                                                                                                    | Fix                                                                                                                                                       | File                      |
| --- | ------------------------------------------------------------------------------------------------------------------------------------------------------ | --------------------------------------------------------------------------------------------------------------------------------------------------------- | ------------------------- |
| 1   | **Silent 1000-event cap** — `replayLimit <= 0` was silently coerced to 1000, permanently dropping events for long-offline clients                      | `replayLimit <= 0` now means **unlimited** with batch streaming (500 events/batch). Exported `DefaultSSEReplayLimit = 1000` for callers who want bounded. | `sse.go:37-47`            |
| 2   | **Dedup wired wrong** — `replayEvents` returned a dedup set but `SSEHandler` discarded it. Live events that were already replayed got delivered twice. | Dedup ring now captured in `replayed` variable and checked in the live loop with `continue` on match.                                                     | `sse.go:222-227, 263-266` |
| 3   | **Client registered AFTER replay** — race window where concurrent live events published during replay were lost (client not yet in the fanout map).    | `AddClient` now runs **before** `replayEvents`. Live events buffer in the channel (size 100) during replay, then drain through the dedup-aware live loop. | `sse.go:217-219`          |

### Bounded Dedup Ring — Memory Safety (`transport/http/dedup_ring.go`)

**Resolves section g) — the bounded-vs-unbounded dedup question.**

The unbounded `map[string]struct{}` dedup set is replaced with a fixed-capacity ring buffer (`dedupRing`, 1024 entries, ~90KB). This is **provably sufficient** because:

- Overlapping events (in both replay and live) are always at the **tail** of the replay stream — they were published during the replay window, so they're the newest events in the journal.
- The live channel buffer is bounded (`sseChannelBufSize = 100`), so at most 100 events can overlap.
- A ring of 1024 gives a **10x safety margin** while bounding memory regardless of journal size.

Both `Add` and `Has` are O(1). The `Has` method is nil-safe (returns false on nil receiver).

**Files:** `transport/http/dedup_ring.go` (68 lines), `transport/http/dedup_ring_test.go` (5 tests)

### SSE Replay Timeout — Browser Safety (`transport/http/sse.go`)

Added `WithReplayTimeout(d time.Duration)` — when the timeout fires mid-replay, the broker sends an `SSEReplayIncompleteEvent` advisory event and switches to live. The advisory has no `id` field, so it doesn't advance the client's Last-Event-ID. Clients receiving it know they're behind and should reconnect with their latest EventID for incremental catch-up.

**Test:** `TestSSEHandler_ReplayTimeout_SendsAdvisoryEvent` — uses a delayed journal wrapper that blocks ReadFrom past the timeout.

### CatchUpSubscriber Bounded Dedup (`watermill/dedup_ring.go`)

Same ring buffer pattern applied to `watermill.CatchUpSubscriber.replayIDs`. Ring capacity 1024 (4x the 256-deep output channel buffer). All existing CatchUp tests pass.

### File Split — CI 350-Line Compliance

`sse.go` exceeded the 350-line CI limit after the timeout feature. Extracted `replayEvents` + `writeReplayBatch` into `sse_replay.go` (138 lines). `sse.go` is now 283 lines.

### Lint Cleanup

| File                              | Fix                                                                                         |
| --------------------------------- | ------------------------------------------------------------------------------------------- |
| `scheduling/scheduler_test.go`    | Removed 5 `infertypeargs` warnings (unnecessary `[string]` type args on `scheduling.New()`) |
| `watermill/trace_context_test.go` | Removed dead `consumeTimeout` var + `consumeTimeoutDuration` func                           |
| `transport/http/sse_test.go`      | Fixed `rangeint` hint (`for i := range numClients`) and `fmtappendf` hint (`fmt.Appendf`)   |

**Verification:** `go test ./... -count=1 -race` passes clean across transport/http, watermill, scheduling. `go vet` clean. All LSP diagnostics clean (0 errors, 0 warnings, 0 hints).

---

## b) FULLY DONE (previously PARTIALLY DONE)

### SSE unlimited replay — time-bounded

Previously: "no total time limit for unlimited replay." Now resolved via `WithReplayTimeout`. Default remains unlimited (correct for server-to-server); browser-facing SSE should set a timeout.

### Dedup set memory — bounded

Previously: "unbounded map for unlimited replay (~26MB for 1M events)." Now resolved via `dedupRing` (1024 entries, ~90KB, regardless of journal size).

---

## c) NOT STARTED (out of scope — separate sessions)

- **No SQL/Redis `TimerStore`** — `scheduling/` only ships `MemoryTimerStore`. Timers are lost on restart.
- **No client push for truly-offline mobile** — SSE requires a live connection. No APNs/FCM integration.
- **No conflict resolution for offline writes** — if an offline client made local writes, reconciling is application logic.
- **CatchUpSubscriber `ReadFrom(ctx, after, 0)` loads all events in one call** — the dedup map is now bounded, but the journal read itself still materializes all events into a single slice. Batched streaming (like SSE) would further reduce memory.

---

## d) TOTALLY FUCKED UP! (historical — now fixed)

### The original dedup test was a lie

`TestSSEHandler_ReplayDedup_NoDuplicates` (the OLD version) sent `Last-Event-ID: ""` (empty string), which meant the dedup code path was **never exercised by any test**. The "dedup" was dead code dressed up as a tested feature. **Fixed in this session** — the rewritten test sends a real Last-Event-ID and verifies actual replay→live deduplication.

---

## e) Improvements — status update

### Immediate (this session's scope)

| #   | Task                                                | Status                                                                                                                                                                                          |
| --- | --------------------------------------------------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| 1   | `WithReplayTimeout(maxDuration)` for browser safety | **DONE** — `WithReplayTimeout(d time.Duration)`                                                                                                                                                 |
| 2   | Bounded dedup set (ring buffer)                     | **DONE** — `dedupRing` in both SSE and CatchUpSubscriber                                                                                                                                        |
| 3   | Simplify `id` field branding round-trip             | **SKIPPED** — the SSEEventID brand provides newline/CR injection validation at the wire boundary. Removing it would lose a security check for no meaningful simplification. Not worth the risk. |

### Broader (noticed during investigation)

| #   | Task                                  | Status                                |
| --- | ------------------------------------- | ------------------------------------- |
| 4   | `SQLTimerStore` in `scheduling/`      | Not started — separate session        |
| 5   | CatchUpSubscriber replayIDs unbounded | **DONE** — bounded dedup ring applied |

---

## f) Remaining work (priority-ordered)

| #   | Priority | Task                                                                                    | Impact                                   |
| --- | -------- | --------------------------------------------------------------------------------------- | ---------------------------------------- |
| 1   | **P0**   | Commit the SSE replay fix + improvements                                                | Ships everything above                   |
| 2   | **P2**   | Implement `SQLTimerStore` in `scheduling/`                                              | Makes durable deadlines production-ready |
| 3   | **P2**   | Add SSE replay integration test with real `MemoryStore` (not just `FakeStore`)          | Catches SQL-specific replay bugs         |
| 4   | **P2**   | Batched streaming in `CatchUpSubscriber.replayPhase`                                    | Reduces peak memory for large journals   |
| 5   | **P3**   | Add SSE replay metrics (replay_count, replay_duration, dedup_hits) to OTel span         | Observability                            |
| 6   | **P3**   | Document SSE vs CatchUpSubscriber decision matrix in SKILL.md                           | Consumer guidance                        |
| 7   | **P3**   | Add `example/` showing SSE + offline client reconnection                                | Usage demo                               |
| 8   | **P3**   | Consider SSE `retry:` field auto-tuning based on client reconnect frequency             | UX                                       |
| 9   | **P4**   | Investigate whether `handleEvent` fanout should use worker pool instead of lock+iterate | Performance at scale                     |
| 10  | **P4**   | Add backpressure: when client channel is full, slow down or drop oldest                 | Prevents slow-client memory bloat        |
| 11  | **P4**   | Add `SSEBroker.Stats()` returning per-client lag (events buffered, events dropped)      | Debugging                                |
| 12  | **P4**   | Consider WebSocket transport alongside SSE for bidirectional needs                      | Feature completeness                     |
| 13  | **P5**   | Add compression support to SSE (gzip + `Content-Encoding: gzip`)                        | Bandwidth                                |
| 14  | **P5**   | Add per-event-type SSE filtering (client subscribes to `user.*` not all events)         | Bandwidth                                |
| 15  | **P5**   | Add SSE authentication middleware example                                               | Security                                 |
| 16  | **P5**   | Add connection draining on `broker.Close()` with grace period                           | Clean shutdown                           |

---

## g) Resolved: Bounded vs Unbounded Dedup

**Decision: bounded ring buffer (1024 entries).**

The dedup set must cover the entire replay→live overlap window to prevent double-delivery. Analysis showed this window is inherently bounded:

1. `AddClient` runs before `replayEvents` — the live channel starts buffering from that point.
2. Replay reads from the journal — events that existed at journal-read time.
3. Overlapping events are those published during the replay window (after AddClient, before the live loop starts).
4. These events are at the **tail** of the replay stream (newest in the journal).
5. The live channel buffer is bounded (100 for SSE, 256 for CatchUpSubscriber).

Therefore: at most `channelBufferSize` events can overlap, and they're always the last entries in the replay. A ring of 1024 (10x SSE margin, 4x CatchUp margin) is provably sufficient with generous safety margin.

**Memory:** ~90KB regardless of journal size (vs ~90MB for 1M events with unbounded map).

**Consistency:** Same pattern applied to both SSE broker and CatchUpSubscriber.
