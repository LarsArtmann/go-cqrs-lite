# Review: Drain-to-Live TOCTOU Race — FIXED

**Source feedback:** [`new/2026-08-13_file-renamer_drain-live-toctou-race.md`](../new/2026-08-13_file-renamer_drain-live-toctou-race.md)
**Date reviewed:** 2026-08-13
**Outcome:** Bug fixed. Post-subscribe catch-up drain (Option A from the feedback) implemented.

---

## The Bug

Events published between the projection worker's journal drain (phase 1) and its `SubscribeAll` callback registration (phase 2) were permanently lost when using non-blocking subscribers (e.g., `system/v4`'s `simpleBus`). The projection never saw them, and the read model was silently incomplete.

**Failure rate:** ~40-60% under `-race`, ~20-30% without.

---

## The Fix (Option A: Post-Subscribe Catch-Up Drain)

**Files changed:**
- `projectionhost/worker.go` — added `handleMu sync.Mutex` to serialize event processing between catch-up drain and live handler
- `projectionhost/worker_drain.go` — rewrote `processLive` to subscribe first, then catch-up drain; added `liveHandler` and `drainCatchUp` methods

### How it works

1. `processLive(ctx, afterID)` calls `SubscribeAll(handler)` first — registering the live callback
2. For non-blocking subscribers (simpleBus): `SubscribeAll` returns immediately → `drainCatchUp` reads events from the journal that were published during the drain→subscribe gap → processes them under `handleMu`
3. For blocking subscribers (message brokers): `SubscribeAll` blocks until shutdown → broker retains messages → no gap → `drainCatchUp` is a no-op (context cancelled)
4. `WorkerLive` status is set AFTER the catch-up drain completes (not before, fixing the status-ordering issue)

### Concurrency safety

The `handleMu` mutex serializes event processing between the catch-up drain and the live handler callback. Without this, a non-blocking subscriber could deliver an event via the callback while the catch-up drain processes a different event from the journal, causing concurrent `projection.Handle` calls.

### Dedup

The existing `seenIDs` dedup ring (`wasSeen`/`markSeen`) prevents double-processing at the overlap (events in both journal and live subscription). The ring is now protected by `handleMu` during the catch-up drain and live phases.

---

## Tests Added

**File:** `projectionhost/catchup_drain_test.go`

- `TestHost_CatchUpDrain_PicksUpEventsPublishedDuringSubscribeWindow` — deterministic regression test. Uses an `appendingSub` that appends events to the journal during `SubscribeAll`, simulating the exact race window. Without the catch-up drain, these events would be permanently lost.
- `TestHost_CatchUpDrain_LiveDeliveryWorksAfterCatchUp` — verifies live handler delivery works after the catch-up drain processes missed events.

Both tests pass 30 iterations under `-race`.

---

## Additional Findings (from feedback)

### `processLive` status ordering — ✅ FIXED

`WorkerLive` is now set AFTER `SubscribeAll` and `drainCatchUp`, not before.

### `workerStartStaggerMs` amplifies the race — NOT CHANGED

The 10ms-per-worker stagger remains. It smooths journal-iterator load spikes. With the catch-up drain fix, the stagger no longer affects correctness (events are caught up regardless of when the worker starts).

### No `System.WaitReady(ctx)` API — NOT ADDED

Consumers can poll `host.Status()` for `WorkerLive`/`WorkerStopped`. With the catch-up drain fix, `WorkerLive`/`WorkerStopped` is now a reliable readiness signal (the catch-up drain has completed).

---

## Known Issue: watermill.CatchUpSubscriber

The `watermill/catchup_subscriber.go` has a structurally similar race between `replayPhase` (journal drain) and `livePhase` (live subscription). However:

1. The consumer's actual bug is in the `system/v4` + `projectionhost` path (now fixed)
2. The CatchUpSubscriber uses a different channel-based architecture requiring a separate fix design
3. For blocking message brokers, the broker retains messages, mitigating the gap

This is tracked as a known issue for a future fix.
