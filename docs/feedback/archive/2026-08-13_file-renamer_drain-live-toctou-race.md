# Bug: Drain-to-live TOCTOU race loses events in memory bus projection startup

> **From**: file-and-image-renamer consumer
> **Date**: 2026-08-13
> **Priority**: High (silent data loss — projections permanently miss events published during startup)
> **Versions**: `system/v4 v4.3.0`, `projectionhost/v4 v4.3.0`

---

## Summary

When using the default in-memory event bus (`simpleBus`), events published between the projection worker's journal drain (phase 1) and its `SubscribeAll` callback registration (phase 2) are **permanently lost**. The projection never sees them, and the read model is silently incomplete.

This is a time-of-check-to-time-of-use (TOCTOU) race in `worker.process()` → `worker.processLive()`. It affects both tests and production startup.

---

## Root cause

### The race window

```
sys.Start(ctx) returns
  │
  ├── Worker goroutine spawned (stagger delay: i * 10ms)
  │     │
  │     ├── process(): drain journal (phase 1)
  │     │     └── ReadFrom returns 0 events → break
  │     │
  │     ├── processLive(): phase 2
  │     │     ├── setStatus(WorkerLive)       ← status says "live" but not ready
  │     │     └── subscriber.SubscribeAll()   ← callback registered HERE
  │     │
  │     └── return nil → worker exits (WorkerStopped)
  │
  │   ◄── RACE WINDOW: commands dispatched here publish events
  │       via bus.Publish() → simpleBus.dispatch() iterates allHandlers
  │       → projection callback NOT YET registered → events dropped
  │
  └── caller dispatches commands
```

### Why `simpleBus` drops the events

`simpleBus` is a synchronous in-process bus — `Publish()` calls `dispatch()` which iterates registered handlers **at call time**. There is no queue, no buffer, no replay. If a handler isn't registered when `Publish` is called, the event is gone.

Source: `system/v4@v4.3.0/bus.go:113-151`

```go
func (b *simpleBus) dispatch(ctx context.Context, evt event.Event) error {
    b.mu.RLock()
    handlers := make([]event.Handler, 0)
    if typed, ok := b.handlers[evt.Type()]; ok {
        handlers = append(handlers, typed...)
    }
    handlers = append(handlers, b.allHandlers...) // empty if SubscribeAll not yet called
    // ...
    for _, handler := range handlers {
        // calls each handler synchronously
    }
}
```

`SubscribeAll` just appends to `allHandlers` and returns:

```go
func (b *simpleBus) SubscribeAll(handler event.Handler) error {
    b.mu.Lock()
    defer b.mu.Unlock()
    b.allHandlers = append(b.allHandlers, handler)
    return nil // returns immediately — does NOT block
}
```

### Why `processLive` assumes blocking

The `processLive` doc says: "Blocks until the context is cancelled, the subscriber returns an error, or the worker is stopped."

This is true for **real message brokers** (Watermill, NATS, Postgres LISTEN/NOTIFY) — their `SubscribeAll` blocks, holding a connection open. Events arriving while the subscriber is connected are delivered via callback.

For `simpleBus`, `SubscribeAll` returns immediately. The worker exits, and future events are delivered via `simpleBus.Publish()` → `dispatch()` → callback. This works for events published **after** registration, but events published **before** registration are permanently lost.

### The `WorkerLive` status lies

`processLive` sets `WorkerLive` **before** calling `SubscribeAll`:

```go
func (w *worker) processLive(ctx context.Context) error {
    w.setStatus(WorkerLive)                    // ← status says "live"
    if err := w.opts.subscriber.SubscribeAll(  // ← callback registered AFTER this
```

Consumers polling `host.Status()` for readiness see `WorkerLive` and assume the projection is catching events. It isn't — not until `SubscribeAll` returns.

---

## Reproduction

### Observed in production tests

**Failure rate**: ~40-60% under `-race`, ~20-30% without.

**Test**: `TestHistoryAdapter_StatsAndEntries` in `file-and-image-renamer`

```go
sys, _ := cqrs.NewMemorySystem(ctx, nil, deps, logger)
sys.Start(ctx)
// ← RACE WINDOW: projection worker may not have called SubscribeAll yet

mustRename(ctx, sys, "hash-ok", "original-ok.png")   // publishes FileReceived + FileRenamed
mustRename(ctx, sys, "hash-fail", "original-fail.png") // publishes FileReceived + RenameFailed

waitForViewInScan(t, ctx, sys, sidFail.String(), cqrs.StatusFailed)
// times out: projection host shows status=stopped, processed=3, errors=0
// (processed hash-ok's 3 events, but missed hash-fail's RenameFailed)
```

### Projection host status on timeout

```
name=projections status=stopped processed=3 errors=0 restarts=0 lastError=
```

3 events processed (hash-ok: received + analyzed + renamed), but hash-fail's `RenameFailed` event was published before `SubscribeAll` registered the callback — permanently lost.

### Minimal reproduction conditions

1. Fresh system with empty journal (drain completes instantly)
2. Command dispatched immediately after `sys.Start()` returns
3. Worker goroutine hasn't been scheduled yet (or is in stagger delay)

The race is amplified by:

- `-race` detector overhead (widens scheduling gaps)
- High parallelism (many goroutines competing for CPU)
- Multiple projections (stagger delay: `i * workerStartStaggerMs` per worker)

---

## Impact

### Tests

Intermittent test failures with 5-15s timeouts. Tests pass in isolation (single test) but fail ~40-60% of the time when the full package runs under `-race`. Diagnosing is difficult because the failure manifests as a projection timeout, not an error.

### Production

Any system using the default memory driver that dispatches commands shortly after `sys.Start()` returns can silently lose projection updates. The events are in the event store (durable), but the projection never applies them — the read model is permanently stale until a system restart (which would re-drain the journal from checkpoint).

For the `file-and-image-renamer` project specifically: the `watch` command starts the CQRS system, then immediately begins watching for file changes. If the first file event arrives in the race window, the rename's projection update is lost.

---

## Consumer workaround

The consumer added a `waitForProjectionReady` test helper that blocks until the projection worker reaches `WorkerStopped` (meaning `SubscribeAll` has completed):

```go
func waitForProjectionReady(t *testing.T, sys *cqrs.System) {
    host := sys.RawSystem().ProjectionHost()
    deadline := time.Now().Add(10 * time.Second)
    for time.Now().Before(deadline) {
        ready := true
        for _, s := range host.Status() {
            if s.Status != "stopped" && s.Status != "failed" {
                ready = false
                break
            }
        }
        if ready { return }
        time.Sleep(5 * time.Millisecond)
    }
}
```

This is called at every `sys.Start()` site in tests. It does **not** protect the production code path.

---

## Proposed fixes (ranked)

### Option A: Post-subscribe catch-up drain (recommended)

After `SubscribeAll` returns, drain the journal again to catch events published during the drain→subscribe gap. This is the standard "drain-catch-up" pattern used in message-driven systems.

```go
// worker_drain.go — processLive or between process() and processLive()

func (w *worker) processLive(ctx context.Context) error {
    // Register the live callback FIRST.
    if err := w.opts.subscriber.SubscribeAll(handler); err != nil {
        return fmt.Errorf("subscribe live events: %w", err)
    }

    // Catch-up drain: process any events published between the initial
    // drain and the subscription registration. The wasSeen() dedup
    // prevents double-processing of events already handled.
    if err := w.drainCatchUp(ctx); err != nil {
        return fmt.Errorf("catch-up drain: %w", err)
    }

    w.setStatus(WorkerLive)
    // For blocking subscribers, SubscribeAll already returned and we exit.
    // For simpleBus, the callback is registered and future events arrive via dispatch.
    return nil
}
```

**Pros**: Fixes the race for ALL subscriber types. No change to `simpleBus`. The existing `wasSeen()` dedup mechanism handles overlap.

**Cons**: Extra journal read on every worker startup. For non-memory subscribers, the catch-up drain is redundant (the broker retains messages), but the dedup makes it harmless.

### Option B: Register subscriber before `Start()` returns

Move the `SubscribeAll` call from the worker goroutine into `Host.Start()`, synchronously, before spawning workers. This guarantees the callback is registered before the system accepts commands.

```go
// host.go — Start()

func (h *Host) Start(ctx context.Context) error {
    // Register all projection handlers as subscribers BEFORE spawning workers.
    for _, w := range h.workers {
        if w.opts.subscriber != nil && !w.subscribed {
            if err := w.opts.subscriber.SubscribeAll(w.liveHandler()); err != nil {
                return err
            }
            w.subscribed = true
        }
    }
    // ... then spawn worker goroutines for journal drain
}
```

**Pros**: Cleanest semantic — `Start()` returns = system is ready.

**Cons**: Changes the subscriber registration lifecycle. The worker's `processLive` would need to skip the `SubscribeAll` call. Requires refactoring the worker/host boundary.

### Option C: `simpleBus` event retention

Make `simpleBus` retain published events and replay them to new `SubscribeAll` callers.

**Pros**: Transparent fix — no projection host changes.

**Cons**: Changes `simpleBus` memory characteristics. Requires retention policy (TTL? watermark? all-events?). The bus is currently stateless beyond handler registration; this makes it stateful. Also doesn't fix the race for other non-blocking subscriber implementations.

---

## Additional findings

### 1. ~~`processLive` status ordering~~ **NOT-DO as written — the shipped ordering is the OPPOSITE: WorkerLive is set BEFORE SubscribeAll (`8108cad5f`), because blocking subscribers never return from SubscribeAll; the catch-up drain closes the race regardless of status ordering.**

`setStatus(WorkerLive)` is called **before** `SubscribeAll`. Any consumer polling for readiness sees "live" and assumes the projection is catching events. The status should be set to `WorkerLive` **after** `SubscribeAll` returns, or a new `WorkerSubscribing` status should cover the gap.

### 2. `workerStartStaggerMs` amplifies the race

Each worker is started with a `i * 10ms` stagger delay (`host.go:170`). With N projections, the last worker doesn't start draining until `N * 10ms` after `Start()` returns. For systems with many projections, this widens the race window significantly.

### 3. ~~No `System.WaitReady(ctx)` API~~ **Won't implement — declined in the TOCTOU review round (TODO_LIST "Declined / Rejected"); polling Status() is the supported readiness signal.**

There is no public API to wait for projection readiness. Consumers must either:

- Poll `ProjectionHost().Status()` (requires knowing worker state semantics)
- Use `time.Sleep` (unreliable)
- Accept the race (data loss)

A `System.WaitReady(ctx) error` method that blocks until all projections have registered their live subscriptions would give consumers a reliable readiness signal.

---

## Files referenced

| File (version)                             | Lines   | What                                                              |
| ------------------------------------------ | ------- | ----------------------------------------------------------------- |
| `projectionhost/v4@v4.3.0/worker_drain.go` | 18-122  | `process()` — journal drain (phase 1)                             |
| `projectionhost/v4@v4.3.0/worker_drain.go` | 157-211 | `processLive()` — live subscription (phase 2), race site          |
| `projectionhost/v4@v4.3.0/host.go`         | 134-174 | `Start()` — spawns workers with stagger delay                     |
| `system/v4@v4.3.0/bus.go`                  | 94-111  | `simpleBus.Publish()` — synchronous dispatch                      |
| `system/v4@v4.3.0/bus.go`                  | 113-151 | `simpleBus.dispatch()` — iterates handlers at call time           |
| `system/v4@v4.3.0/bus.go`                  | 162-169 | `simpleBus.SubscribeAll()` — appends handler, returns immediately |
| `system/v4@v4.3.0/constructor.go`          | 194-201 | Auto-wires `simpleBus` as projection subscriber                   |

---

## Resolution (2026-08-15, docs-health pass)

Option A (post-subscribe catch-up drain) shipped as `d60d72ed4`, with the
blocking-subscriber status regression fixed at `8108cad5f` and regression
tests in `projectionhost/catchup_drain_test.go`. The same-class watermill
CatchUpSubscriber race closed at `1b4e79b78` (subscribe-first + replayIDs
dedup). Finding 1 was resolved opposite to its recommendation (see inline);
finding 3 declined (TODO_LIST Declined); finding 2 (stagger amplification)
remains an open observation. Reviewed counterpart:
`docs/feedback/reviewed/2026-08-13_file-renamer_drain-live-toctou-race-review.md`.
Archived.
