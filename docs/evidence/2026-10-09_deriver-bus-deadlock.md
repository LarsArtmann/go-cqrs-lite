# Evidence Pack: Synchronous-Deriver-on-Bus Deadlock

> **Date:** 2026-10-09 · **Status:** FINAL (input to [ADR-0154](../adr/0154-deriver-async-dispatch-and-journal-tailed-host.md))
> **Source:** adoption-wave plan T01 ([`docs/planning/2026-10-09_14-49_SUPERB-bdd-harness-adoption-wave.md`](../planning/2026-10-09_14-49_SUPERB-bdd-harness-adoption-wave.md)).
> **Finding origin:** systemscenario harness saga test, 2026-10-09 (ADR-0153 execution session); re-reproduced and stack-captured 2026-10-09 for this pack.
> **Raw stacks:** [`docs/status/deriver-deadlock-stacks-2026-10-09.txt`](../status/deriver-deadlock-stacks-2026-10-09.txt) (captured live, not reconstructed).

## 1. Reproduction (F01.1)

### Minimal wiring

A `deriver.Deriver` subscribed on `sys.Bus()` whose derived command is dispatched
**synchronously** (the default `AsHandler` path), where the derived command's handler
emits an event:

```go
archiver := deriver.Deriver(
	func(ctx context.Context, evt event.Event) ([]command.Command, error) {
		return []command.Command{newTaskCmd("task.archive", evt.StreamID())}, nil // sync: no goroutine
	},
)
_ = sys.Bus().Subscribe("task.updated", archiver.AsHandler(sys.CommandDispatcher()))
```

Any dispatch that emits `task.updated` then hangs forever inside `Dispatch`.
Repro executed as a temporary test (deleted after capture): boot `system.New` with the
above domain + memory deployment, dispatch `task.create` then `task.complete`, poll 3s,
dump all goroutine stacks.

### Observed stack cycle (watermill v1.5.3, GoChannel)

Four goroutines form the cycle (raw dump in the linked file; frames abridged):

```
goroutine 53 [select]   — the CALLER's dispatch goroutine
  GoChannel.waitForAckFromSubscribers      pubsub.go:144   ← waits for subscriber ack
  GoChannel.Publish                       pubsub.go:133   ← HOLDS per-topic subLock (deferred unlock)
  EventBus.Publish                        event_bus.go:126
  command.Dispatcher.Dispatch

goroutine 51 [sync.Mutex.Lock]  — the single event-loop goroutine
  GoChannel.Publish                       pubsub.go:108   ← BLOCKED acquiring the same per-topic subLock
  EventBus.Publish                        event_bus.go:126   (the NESTED publish, from the handler)
  command.Dispatcher.Dispatch             (derived task.archive)
  EventBus.rebuildHandlerChain.func1      event_bus_internals.go:54  (deriver handler)
  EventBus.dispatchLocal                  event_bus_internals.go:70
  EventBus.runEventLoop                   event_bus_internals.go:98

goroutine 57 [select]  — delivery goroutine
  subscriber.sendMessageToSubscriber      pubsub.go:408   ← blocked sending to the busy event loop

goroutine 56 [sync.WaitGroup.Wait] — delivery supervisor waiting on 57
```

### Mechanism, precisely

1. Outer `Publish(task.updated)` acquires the GoChannel **per-topic subscriber mutex**
   (`subscribersByTopicLock`, watermill `pubsub.go:106-113`; held across the whole
   publish via `defer Unlock`) and blocks in `waitForAckFromSubscribers` because
   `watermill.NewEventBus` configures `BlockPublishUntilSubscriberAck: true`
   ([`watermill/event_bus.go:92`](../../watermill/event_bus.go)).
2. The subscriber is our **single event-loop goroutine**
   ([`watermill/event_bus_internals.go:77`](../../watermill/event_bus_internals.go)),
   which processes one message at a time; it is inside the deriver handler.
3. The deriver handler dispatches the derived command; its handler emits
   `task.archived`; the journal-bus path calls the nested `Publish`, which blocks
   acquiring the **same per-topic mutex** the outer publish still holds.
4. Cycle closed: outer publish holds the lock and waits for an ack that requires the
   event loop to finish; the event loop waits on the lock held by the outer publish.
   `Close()` does not reliably unwind it: the nested publish's mutex acquisition has
   no closing-select escape (plain `Lock` at `pubsub.go:108`).

**Hang condition (generalized):** any handler invoked by `watermill.EventBus` delivery
that (transitively) publishes to the same bus topic from the delivering goroutine
deadlocks. Derivers are simply the most likely first contact because `AsHandler`
dispatches synchronously by design.

## 2. Option-design memo (F01.2)

|                        | (a) Async bus delivery                                                                              | (b) `deriver.WithAsyncDispatch` option                                                             | (c) Journal-tailed deriver host                                |
| ---------------------- | --------------------------------------------------------------------------------------------------- | -------------------------------------------------------------------------------------------------- | -------------------------------------------------------------- |
| Change surface         | `watermill` EventBus delivery semantics (every subscriber, every consumer)                          | `deriver` `AsHandler` options                                                                      | New infrastructure (host process tailing the journal)          |
| Ordering               | Breaks ordered live delivery for ALL consumers (the reason `BlockPublishUntilSubscriberAck` is set) | Derived commands run after the handler returns; per-source-event order preserved by dispatch queue | Totally ordered by journal position; survives restarts         |
| Risk to v4.x consumers | HIGH: silent semantic shift of a shipped bus                                                        | LOW: opt-in, additive, default unchanged                                                           | New module: none until adopted                                 |
| Error surfacing        | Unchanged                                                                                           | Requires explicit design (see §3)                                                                  | Host owns retries/DLQ semantics (design at v5)                 |
| Deadlock cured         | Yes (nested publish no longer blocks the loop)                                                      | Yes for the deriver class (dispatch leaves the handler goroutine)                                  | Yes (derivers never touch the bus)                             |
| Cost                   | Rework of delivery + all tests that rely on sync publish→ack                                        | ~90 min, additive option + tests                                                                   | v5-scale project (cursoring, at-least-once, idempotency story) |

**Ruling (accepted by owner 2026-10-09, G1):** (b) NOW, (c) as the v5 direction.
(a) is rejected for v4.x — it changes global ordering semantics for every consumer to
fix a deriver-specific wiring hazard.

## 3. Async error-surfacing design (F01.3)

The workaround being promoted (`go func() { _ = Dispatch(...) }()`) **swallows errors**.
Decision (b) must not. Design:

- **Callback over logger:** `WithAsyncDispatch` takes an error callback
  (`func(evt event.Event, cmd command.Command, err error)`). Rationale: a logger-only
  design repeats the swallow (logs are not assertable); a callback lets hosts, tests,
  and the systemscenario harness turn failures into visible state. No default logger
  fallback: if no callback is set, dispatch errors are still reported via the handler's
  returned error path ONLY where the goroutine can reach it — it cannot — so the
  contract is: **no callback = errors dropped with a one-line doc warning** (matching
  Go's `http.Server` error-nil convention is NOT acceptable here; we document loudly
  instead of panicking in a library).
- **Ordering caveat wording** (docs + option godoc): derived commands are dispatched on
  a per-handler goroutine; commands derived from the SAME event dispatch in order, but
  commands derived from DIFFERENT events may interleave. Tests that need cross-event
  ordering must assert via `Await()`-style polling or the journal, not dispatch order.
- **Context:** dispatches run with the handler's context detached from cancellation
  deadlines that die with the publish (`context.WithoutCancel`), so async work is not
  killed by the outer request scope ending.

## 4. Mismatch vs current workaround (F01.4)

- The harness fixture (`systemscenario/fixtures_test.go` `sagaDomain`) spawns a raw
  goroutine per event: fire-and-forget, zero error visibility, unbounded goroutine
  spawn. `WithAsyncDispatch` replaces it (T09) — same async semantics, first-class
  error surface.
- `deriver.AsHandler` doc claims "Each command is dispatched sequentially" — still true
  per event under (b); the async mode adds the interleaving caveat above.
- `WithMaxDepth` guards synchronous cycles only; async dispatch escapes the depth
  counter (context does not cross goroutines). Docs must state that async derivation
  cycles are bounded by idempotency (`Deriver.Idempotent`), not depth. This is the
  exact class §3 of the deriver docs already warns about — wording gets tightened in T08.
