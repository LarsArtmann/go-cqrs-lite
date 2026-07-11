# Bounded Dedup Ring + Replay Timeout + Lint Cleanup — Session Status

**Date:** 2026-07-05 23:02 CEST
**Scope:** `transport/http` SSE broker, `watermill` CatchUpSubscriber, `scheduling` tests, lint cleanup
**Trigger:** User asked to execute the improvement plan from `2026-07-05_21-37_SSE-REPLAY-DEDUP-FIX.md`

---

## a) FULLY DONE

### 1. Bounded Dedup Ring — SSE (`transport/http/dedup_ring.go` + `dedup_ring_test.go`)

The unbounded `map[string]struct{}` dedup set in the SSE replay path has been replaced with a fixed-capacity ring buffer (`dedupRing`, 1024 entries, ~90KB).

**Why this is provably correct:** Overlapping events (present in both the journal replay and the live stream) are always at the **tail** of the replay sequence — they were published during the replay window (after `AddClient`, before the live loop starts), so they're the newest events in the journal. The live channel buffer is bounded at `sseChannelBufSize = 100`, so at most 100 events can overlap. A ring of 1024 gives a **10x safety margin**.

| Property         | Value                                         |
| ---------------- | --------------------------------------------- |
| Capacity         | 1024 entries                                  |
| Memory           | ~90KB (regardless of journal size)            |
| `Add` complexity | O(1)                                          |
| `Has` complexity | O(1)                                          |
| Nil-safe         | Yes (`Has` on nil `*dedupRing` returns false) |

**Tests:** 5 unit tests — `TestDedupRing_Basic`, `TestDedupRing_Eviction`, `TestDedupRing_DuplicateAdd`, `TestDedupRing_NilSafe`, `TestDedupRing_LargeCapacity` (exercises 3x capacity wraparound).

### 2. SSE Replay Timeout (`transport/http/sse.go`)

Added `WithReplayTimeout(d time.Duration)` — when the timeout fires mid-replay, the broker sends an `SSEReplayIncompleteEvent` advisory event and switches to live delivery. The advisory has no `id` field, so it doesn't advance the client's `Last-Event-ID`. Clients receiving it know they're behind and can reconnect with their latest EventID for incremental catch-up.

The timeout is implemented via `context.WithTimeout` wrapping the replay context. Both bounded (`replayLimit > 0`) and unlimited (`replayLimit <= 0`) replay paths check `ctx.Err()` and mark `timedOut = true` when the deadline fires. The OTel span gets a `cqrs.sse.replay_status = "incomplete"` attribute.

**Test:** `TestSSEHandler_ReplayTimeout_SendsAdvisoryEvent` — uses a `delayedJournal` wrapper that blocks `ReadFrom` for 200ms, with a 10ms replay timeout. Verifies the advisory event appears in the response body.

### 3. Bounded Dedup Ring — CatchUpSubscriber (`watermill/dedup_ring.go` + `catchup_subscriber.go`)

Same ring buffer pattern applied to `watermill.CatchUpSubscriber.replayIDs`. Ring capacity 1024 (4x the 256-deep output channel buffer). The `catchUpSubscription.replayIDs` field changed from `map[string]struct{}` to `*dedupRing`. All existing CatchUp tests pass unchanged.

### 4. File Split — CI 350-Line Compliance (`transport/http/sse_replay.go`)

`sse.go` exceeded the 350-line CI limit after adding the timeout feature. Extracted `replayEvents` + `writeReplayBatch` into `sse_replay.go` (138 lines). `sse.go` is now 283 lines. Removed the now-unused `id` import from `sse.go`.

### 5. Lint Cleanup

| File                              | Fix                                                                                                        | Diagnostics removed       |
| --------------------------------- | ---------------------------------------------------------------------------------------------------------- | ------------------------- |
| `scheduling/scheduler_test.go`    | Removed 5 unnecessary `[string]` type args on `scheduling.New()` — Go infers from `TimerStore[string]` arg | 5 × `gopls infertypeargs` |
| `watermill/trace_context_test.go` | Removed dead `consumeTimeout` var + `consumeTimeoutDuration` func (never referenced anywhere)              | Unused code               |
| `transport/http/sse_test.go`      | `for i := 0; i < numClients; i++` → `for i := range numClients`                                            | 1 × `gopls rangeint`      |
| `transport/http/sse_test.go`      | `[]byte(fmt.Sprintf(...))` → `fmt.Appendf(nil, ...)`                                                       | 1 × `gopls fmtappendf`    |

### Verification

```
transport/http:  go test -count=1 -race ./... → PASS (all 17 tests)
watermill:       go test -count=1 -race ./... → PASS (all tests)
scheduling:      go test -count=1 -race ./... → PASS (all tests)
go vet:          clean across all three modules
LSP diagnostics: 0 errors, 0 warnings, 0 hints
File line counts: all production files ≤ 350 lines (CI limit)
```

---

## b) PARTIALLY DONE

### SSE replay metrics (OTel span attributes)

The replay span already records `cqrs.sse.last_event_id`, `cqrs.sse.replay_status` (on timeout), and `cqrs.event.count` (total replayed events). But richer metrics — `replay_duration` histogram, `dedup_hits` counter, `replay_incomplete_total` — would require adding Int64Counter/Histogram instruments. The span attributes are useful for ad-hoc tracing but not for dashboards/alerts. **Not blocking — the tracing foundation is there.**

### CatchUpSubscriber `ReadFrom(ctx, after, 0)` still loads all events at once

The dedup map is now bounded, but `replayPhase` in `watermill/catchup_subscriber.go:183` still calls `s.journal.ReadFrom(ctx, after, 0)` which loads the **entire** journal tail into a single `[]event.Event` slice. For a million-event journal this materializes ~1M event pointers. The SSE broker solves this with batched streaming (`sseReplayBatchSize = 500`), but applying the same pattern to CatchUpSubscriber would change the replay→live handoff sequencing and needs careful testing. **The dedup is bounded; the read is not.**

---

## c) NOT STARTED

These are adjacent improvements noticed during the session but explicitly out of scope:

- **No SQL/Redis `TimerStore`** — `scheduling/` only ships `MemoryTimerStore`. Timers are lost on restart.
- **No client push for truly-offline mobile** — SSE requires a live connection. No APNs/FCM integration.
- **No conflict resolution for offline writes** — if an offline client made local writes, reconciling is application logic.
- **No SSE `retry:` field auto-tuning** — the field is supported in the wire format but not dynamically adjusted.
- **No per-event-type SSE filtering** — clients receive all events, can't subscribe to `user.*` only.
- **No SSE authentication middleware example** — security docs mention it but no example exists.

---

## d) TOTALLY FUCKED UP!

### Dead code in `watermill/trace_context_test.go` shipped for who knows how long

The `consumeTimeout` variable and `consumeTimeoutDuration` function were **never referenced by anything** in the entire watermill module. Not by a single test, not by production code, not by other test files. They were just sitting there — a `<-chan struct{}` that was immediately closed and never read from. Pure dead weight masquerading as test infrastructure. **Removed.**

### The `watermill/go.mod` has a pre-existing `go mod tidy` issue

Running `go mod tidy` in the watermill module fails because `event/v4`'s test suite imports `schema/v4` which has the `000000000000` zero-revision placeholder. This is a **pre-existing workspace issue** (the `000000000000` revision is Go's placeholder for replace-directive modules that haven't been published). `go mod tidy -e` suppresses it but also silently "fixes" the `command/v4` dependency version from `00010101000000-000000000000` to `v3.5.0` — an unintended change. I caught this and reverted the `go.mod` change. **Tests pass via `go.work` (workspace mode) regardless.**

---

## e) WHAT WE SHOULD IMPROVE!

### Immediate follow-ups (directly related to this session's work)

1. **Batched streaming in `CatchUpSubscriber.replayPhase`** — the dedup ring is bounded but the journal read still materializes everything. Port the SSE batch streaming pattern (500 events/batch with cursor advancement) to CatchUpSubscriber.
2. **SSE replay metrics as OTel instruments** — promote span attributes to real `Int64Counter` / `Float64Histogram` instruments for dashboard-ready observability.
3. **`delayedJournal` test helper should be promoted** — the `delayedJournal` wrapper in `sse_test.go` is useful for any context-cancellation test. Consider promoting to `eventtest` or `testutil`.

### Broader observations (noticed but not researched)

4. **`handleEvent` fanout holds `RLock` during channel sends** — a slow client that blocks on `ch <- evt` (when the 100-buffer is full) currently gets silently dropped via `default:`. But the lock is held during the entire iteration. A worker pool or per-client goroutine would decouple fanout from the lock.
5. **SSE `RemoveClient` doesn't close the channel** — documented as intentional (avoids send-on-closed races), but means the channel lingers until GC. For high client churn this could accumulate.
6. **The `sseReplayBatchSize = 500` constant is hardcoded** — for very large payloads (e.g. 1MB events), 500 events × 1MB = 500MB per batch in memory before writing. A byte-budget alternative would be safer.

---

## f) Up to 25 things we should get done next

| #   | Priority | Task                                                                                   | Impact                                                               |
| --- | -------- | -------------------------------------------------------------------------------------- | -------------------------------------------------------------------- |
| 1   | **P0**   | Commit all session work (5 modified, 5 new files)                                      | Ships the bounded dedup + timeout + lint fixes                       |
| 2   | **P1**   | Port batched streaming to `CatchUpSubscriber.replayPhase`                              | Bounds memory for large journals (currently materializes all events) |
| 3   | **P2**   | Implement `SQLTimerStore` in `scheduling/`                                             | Makes durable deadlines production-ready                             |
| 4   | **P2**   | Add SSE replay integration test with real `MemoryStore` (not just `FakeStore`)         | Catches store-specific replay bugs                                   |
| 5   | **P2**   | Promote `delayedJournal` test helper to `eventtest`/`testutil`                         | Reusable context-cancellation testing                                |
| 6   | **P2**   | Fix `watermill/go.mod` tidy issue (`schema/v4` zero-revision)                          | Clean `go mod tidy` without `-e`                                     |
| 7   | **P3**   | Add SSE replay metrics as OTel instruments (replay_count, replay_duration, dedup_hits) | Dashboard-ready observability                                        |
| 8   | **P3**   | Document SSE vs CatchUpSubscriber decision matrix in SKILL.md                          | Consumer guidance for choosing transport                             |
| 9   | **P3**   | Add `example/` showing SSE + offline client reconnection with `WithReplayTimeout`      | Usage demo for the new feature                                       |
| 10  | **P3**   | Byte-budget replay batching (replace fixed `sseReplayBatchSize`)                       | Memory safety for large-payload events                               |
| 11  | **P3**   | Document `SSEReplayIncompleteEvent` in AGENTS.md key patterns                          | Discoverability of the advisory event                                |
| 12  | **P4**   | Investigate `handleEvent` fanout via worker pool instead of lock+iterate               | Performance at scale                                                 |
| 13  | **P4**   | Add backpressure: when client channel is full, slow down or drop oldest                | Prevents slow-client memory bloat                                    |
| 14  | **P4**   | Add `SSEBroker.Stats()` returning per-client lag (events buffered, events dropped)     | Debugging                                                            |
| 15  | **P4**   | Consider WebSocket transport alongside SSE for bidirectional needs                     | Feature completeness                                                 |
| 16  | **P4**   | SSE `RemoveClient` channel lifecycle — consider closing with guard                     | Reduces channel accumulation under high churn                        |
| 17  | **P5**   | Add compression support to SSE (gzip + `Content-Encoding: gzip`)                       | Bandwidth                                                            |
| 18  | **P5**   | Add per-event-type SSE filtering (client subscribes to `user.*`)                       | Bandwidth                                                            |
| 19  | **P5**   | Add SSE authentication middleware example                                              | Security                                                             |
| 20  | **P5**   | Add connection draining on `broker.Close()` with grace period                          | Clean shutdown                                                       |
| 21  | **P5**   | Consider SSE `retry:` field auto-tuning based on client reconnect frequency            | UX                                                                   |
| 22  | **P5**   | Add `dedupRing` benchmark (Add/Has under contention)                                   | Performance characterization                                         |
| 23  | **P5**   | Consider configurable `sseDedupRingCapacity` via option                                | Tunability for non-default channel sizes                             |
| 24  | **P5**   | Add `dedupRing.Len()` to OTel span for replay diagnostics                              | Observability                                                        |
| 25  | **P5**   | Consider shared `dedupRing` package (extract from both transport/http and watermill)   | DRY — currently duplicated                                           |

---

## g) Top #1 question I can NOT figure out myself

**Should the `dedupRing` be extracted into a shared package, or is duplication acceptable?**

Both `transport/http/dedup_ring.go` and `watermill/dedup_ring.go` implement the **exact same** ring buffer — same algorithm, same O(1) operations, same eviction logic. The only differences are:

- `transport/http` version takes capacity as a constructor arg and has `Len()`, nil-safe `Has`, and is tested.
- `watermill` version hardcodes capacity (1024) and has no `Len()`, no nil-safety, no tests.

Options:

- **A) Extract to a shared package** (e.g. `internal/dedup` or `kv` or a new `collections` module). Clean DRY, single tested implementation. **But:** this repo is a multi-module workspace where each module has its own `go.mod` with strict dependency budgets. Adding a new shared module means wiring it into `go.work`, setting `go.mod` deps for both `transport/http` and `watermill`, and updating `check-module-layers.sh`. For ~50 lines of code.
- **B) Keep duplicated.** Each module is self-contained, no new dependency, no layer wiring. The duplication is small and stable (the algorithm won't change). **But:** if we find a bug in one, we have to remember to fix the other.
- **C) Put it in an existing shared leaf module** like `kv/` (which already has generic data structures). **But:** `kv/` is Layer 0 and `transport/http` is Layer 5 — this would be fine directionally, but `kv/`'s `go.mod` would need `transport/http` as... no, the dependency goes the right way (transport/http → kv). However `watermill` (Layer 5) → `kv` (Layer 0) is also fine.

I can't decide this because it's a **project philosophy question**: is ~50 lines of identical code worth a new module + dependency wiring in a repo that enforces strict per-module dependency budgets? The answer depends on how much the maintainers value "zero duplication" vs "zero dependency wiring overhead" — and this repo already has 47 `go.mod` files, so the cost of adding a 48th is real.
