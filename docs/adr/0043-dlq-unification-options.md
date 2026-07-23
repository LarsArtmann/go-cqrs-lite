# ADR-0043: Dead-Letter Store Unification Options

## Status

**Accepted — 2026-06-29 (Option C: keep separate, document why).** The two
types serve genuinely different lifecycles (dispatch-retry-exhaustion vs
projection-poison). Forcing them together would add coupling without benefit.
This ADR now documents the decision rather than just presenting options.

## Context

Two `DeadLetterEntry` types coexist in the codebase:

| Field         | `middleware.DeadLetterEntry`              | `projectionhost.DeadLetterEntry`                      |
| ------------- | ----------------------------------------- | ----------------------------------------------------- |
| Scope         | Dispatch-side (command/event/query retry) | Projection-side (poison event capture)                |
| Store shape   | `DeadLetterHandler` callback (func)       | `DeadLetterStore` interface (Store/List/Delete/Purge) |
| `Kind`        | `"command"` / `"event"` / `"query"`       | absent (always events)                                |
| `AggregateID` | `id.AggregateID` (typed)                  | `string`                                              |
| `Error`       | `error` (typed)                           | `string`                                              |
| `Event`       | absent                                    | `event.Event` (carries the poison event)              |
| `Attempts`    | `int`                                     | absent                                                |
| Replay        | absent                                    | `Host.ReplayDeadLetters` (pure)                       |

They serve **different lifecycles**:

- **middleware**: a message exhausted its retry budget in the dispatch
  pipeline. The handler is invoked once; the caller decides what to do. There
  is no Store interface — only `DeadLetterHandler func(ctx, entry)`. The
  `MemoryDeadLetterStore` is one impl of that handler.
- **projectionhost**: a projection's `Handle` returned an error past the DLQ
  threshold. The entry carries the original event so it can be replayed after a
  fix. The Store is a first-class interface.

## The real question

Are these the **same concept at different layers** (unify), or **two different
concepts that happen to share a name** (keep separate)?

## Options

### (A) New top-level `dlq/` module — UNIFY

Both `middleware` and `projectionhost` import a shared `dlq.Entry[P]` and
`dlq.Store[P]` (generic over the message type). middleware gets replay
capability for free; projectionhost's type becomes an alias.

- **Pro:** one type, one store interface, replay everywhere. The "reliability
  trio" claim becomes honest.
- **Con:** +1 module (we're at 53). Forces middleware to carry `event.Event`
  even when the dead-lettered message is a command or query (the `Kind` field
  exists precisely because the payload type varies).
- **Generic tradeoff:** `Entry[P]` where P is the message type — clean, but
  middleware's P is `command.Command`/`event.Event`/`query.Query` (varies),
  while projectionhost's is always `event.Event`.

### (B) Move into `event/` — PARTIAL UNIFY

`event/` already defines `Checkpoint`, `Tombstone`, and other cross-cutting
concepts. Add `event.DeadLetterEntry` + `event.DeadLetterStore`.

- **Pro:** no new module. Both middleware (Layer 5) and projectionhost (Layer 3)
  already import `event/`.
- **Con:** `event/` keeps absorbing cross-cutting concerns. Commands and
  queries can also be dead-lettered — putting the type in `event/` is
  conceptually narrow.

### (C) Keep separate — DOCUMENT THE SPLIT

The two types are genuinely different: dispatch-retry-exhaustion vs
projection-poison. They have different fields, different store shapes, and
different replay semantics. Rename one to avoid the collision
(e.g. `middleware.RetryExhaustionEntry`, `projectionhost.PoisonEntry`) and
document that they are intentionally separate.

- **Pro:** no coupling. Each layer evolves its DLQ to fit its lifecycle. Zero
  module-graph change.
- **Con:** the "split brain" framing persists in casual reading. Consumers who
  want both must learn two APIs.

### (D) Bridge via middleware replay capability — NARROW UNIFY

Keep both types, but add `Event event.Event` to `middleware.DeadLetterEntry`
and a `Replay(handler)` method to `middleware.MemoryDeadLetterStore`, so
middleware gains replay without merging types.

- **Pro:** minimal change. Each layer keeps its field shape. Replay parity.
- **Con:** two types still coexist. The duplication of Store/List/Delete/Purge
  patterns remains.

## Recommendation

**Decision: Option C — keep separate, document why.**

The two types serve genuinely different lifecycles:

- **middleware.DeadLetterEntry** captures a message that exhausted retries in the
  _dispatch pipeline_. Its `Kind` field exists because the dead-lettered
  message can be a command, event, or query. There is no Store interface — only
  a `DeadLetterHandler` callback. Replay is not needed because dispatch-side
  failures are typically permanent (bad payload, schema mismatch).

- **projectionhost.DeadLetterEntry** captures an event that poisoned a
  _projection handler_. It carries the original `event.Event` so it can be
  replayed after a fix. Replay IS needed because projection bugs are fixable —
  deploy a patch, re-run the handler, clear the entry.

Unifying them would force middleware to carry `event.Event` even when the
dead-lettered message is a command or query, and would couple the dispatch
pipeline to projection semantics. The field divergence (`Error error` vs
`Error string`, `AggregateID id.AggregateID` vs `AggregateID string`) reflects
these different storage and replay needs, not carelessness.

**This is not a split brain. It is two different patterns that share a name.**
The name "dead letter" is accurate for both — it's the postal term for
undeliverable mail. The implementations diverge because the problems diverge.

## Consumer Burden — The Real Cost of Two Types

Consumers who use **both** retry middleware (dispatch-side DLQ) AND
projectionhost (projection-side DLQ) must learn two parallel APIs with
divergent field shapes:

```go
// Dispatch-side dead letter (middleware)
entry := middleware.DeadLetterEntry{
    Kind:        "command",           // string enum
    AggregateID: someID,              // id.AggregateID (typed)
    Error:       someErr,             // error (typed)
    // no Event field — handler already has the message
}

// Projection-side dead letter (projectionhost)
entry := projectionhost.DeadLetterEntry{
    EventID:     "01J...",            // string
    EventType:   "user.created",      // string
    Error:       someErr.Error(),     // string (serialized)
    Event:       evt,                 // event.Event (carries payload for replay)
    // no Kind field — always events
}
```

The concrete pain points:

1. **Error handling differs** — dispatch DLQ gives you `error` (inspect
   immediately); projection DLQ gives you `string` (must parse or re-derive).
2. **Aggregate ID typing differs** — `id.AggregateID` vs `string` (no
   compile-time safety on the projection side).
3. **No shared tooling** — a dead-letter dashboard must handle both shapes
   separately. A shared store, alerting hook, or audit log adapter cannot
   treat them uniformly.
4. **Two mental models** — "retry exhausted" (dispatch) vs "handler
   poisoned" (projection). Conceptually adjacent, structurally disjoint.

**This is the cost of Option C.** It was accepted because the alternative
(unification) would force middleware to carry `event.Event` even for
command/query dead-letters, and would couple the dispatch pipeline to
projection replay semantics. See [ADR-0059](0059-dlq-unification-proposal.md)
for a Proposed path toward reducing this burden in a future major version.

## Part B: Consumer Operational Guide

The decision above is the _why_. This section is the _what to do about it_ when
you encounter dead letters in production.

### Which DeadLetterEntry am I looking at?

| Symptom                                                       | Type                             | Where                      |
| ------------------------------------------------------------- | -------------------------------- | -------------------------- |
| A command/event/query failed in the middleware retry pipeline | `middleware.DeadLetterEntry`     | `middleware/deadletter.go` |
| A projection handler returned an error past the retry limit   | `projectionhost.DeadLetterEntry` | `projectionhost/dlq.go`    |

### Dispatch-side dead letters (`middleware.DeadLetterEntry`)

These fire when the **retry middleware** exhausts its attempt budget on a
command, event, or query handler. The message is handed to a
`DeadLetterHandler` callback (one-shot, no Store interface). Common causes:
malformed payload, schema mismatch, nil-pointer in the handler.

```go
// Wire a dead-letter handler into the retry middleware:
cmds.Use(middleware.RetryWithDeadLetter(retryCfg, func(ctx context.Context, entry middleware.DeadLetterEntry) {
    alerting.Notify(ctx, fmt.Sprintf("dispatch dead-letter: %s/%s: %s",
        entry.Kind, entry.Type, entry.Error))
}))
```

**Replay:** Not built-in. Dispatch-side failures are typically permanent. If the
handler was fixed and the original message can be reconstructed, re-dispatch
manually.

### Projection-side dead letters (`projectionhost.DeadLetterEntry`)

These fire when a **projection's `Handle` method** returns an error past the DLQ
threshold (default: 3 retries with backoff). The entry is stored in a
`DeadLetterStore` (interface with Store/List/Delete/Purge/Count). Common causes:
handler bug, transient data corruption, downstream service unavailable.

```go
// Inspect dead-letter entries:
entries, _ := dlqStore.List(ctx, "users")
for _, e := range entries {
    log.Printf("poison event %s (%s): %s [%s/%s]", e.EventID, e.EventType, e.Error, e.ErrorFamily, e.ErrorCode)
}

// Replay after fixing the handler bug:
result, _ := host.ReplayDeadLetters(ctx, "users")
for _, e := range result.StillFailing {
    log.Printf("still failing: %s", e.Err)
}
// Purge successfully replayed entries:
for _, e := range result.Replayed {
    dlqStore.Delete(ctx, e.ProjectionName, e.EventID)
}

// Or purge everything during a full reset:
host.Reset(ctx, "users", projectionhost.WithPurgeDeadLetters())
```

**Replay:** Built-in via `Host.ReplayDeadLetters` (pure — does NOT mutate the
store). The caller decides whether to Delete/Purge after a successful replay.

### Why they can't share a type

The field divergence is structural, not cosmetic:

| Need                          | middleware.DLQ                           | projectionhost.DLQ                       |
| ----------------------------- | ---------------------------------------- | ---------------------------------------- |
| Message type varies?          | Yes (command/event/query → `Kind` field) | No (always `event.Event`)                |
| Need the original message?    | No (handler has it)                      | Yes (replay requires the event payload)  |
| Error as `error` or `string`? | `error` (caller inspects immediately)    | `string` (serialized for persistence)    |
| Store vs callback?            | Callback (fire-and-forget)               | Store interface (query, paginate, purge) |
| Replay needed?                | Rarely (permanent failures)              | Always (fixable bugs)                    |

Forcing them into one type would make middleware carry `event.Event` even for
command/query dead-letters, and would couple the dispatch pipeline to projection
replay semantics.

## References

- ADR-0042 (pure replay design — the projectionhost side of this story)
- `middleware/deadletter.go`, `projectionhost/dlq.go`
- Plan C3 in `docs/planning/2026-06-29_brutal-self-review-execution-plan.md`
