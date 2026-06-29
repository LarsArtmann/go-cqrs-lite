# ADR-0043: Dead-Letter Store Unification Options

## Status

**Proposed — 2026-06-29. Awaiting decision.** This ADR presents options but
does NOT pick one. The decision affects every middleware and projectionhost
consumer and requires the user's call.

## Context

Two `DeadLetterEntry` types coexist in the codebase:

| Field            | `middleware.DeadLetterEntry`               | `projectionhost.DeadLetterEntry`         |
| ---------------- | ------------------------------------------ | ---------------------------------------- |
| Scope            | Dispatch-side (command/event/query retry)  | Projection-side (poison event capture)   |
| Store shape      | `DeadLetterHandler` callback (func)        | `DeadLetterStore` interface (Store/List/Delete/Purge) |
| `Kind`           | `"command"` / `"event"` / `"query"`        | absent (always events)                   |
| `AggregateID`    | `id.AggregateID` (typed)                   | `string`                                 |
| `Error`          | `error` (typed)                            | `string`                                 |
| `Event`          | absent                                     | `event.Event` (carries the poison event) |
| `Attempts`       | `int`                                      | absent                                   |
| Replay           | absent                                     | `Host.ReplayDeadLetters` (pure)          |

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

**Lean (A) — a dedicated `dlq/` module** — IF the team values the "reliability
trio" as a single coherent story. The concept earns its own boundary.

**Lean (C) — keep separate + rename** — IF the team accepts that
dispatch-exhaustion and projection-poison are different problems sharing a
name. This is the lowest-risk option and the most honest about the distinction.

The deciding factor: **do consumers ever need ONE dead-letter store that
holds both dispatch-failed commands AND projection-poisoned events?** If yes,
unify (A). If no — and they are typically inspected by different operators —
keep separate (C).

## Decision needed

Which option (A / B / C / D)? This blocks the "reliability trio" integrity
claim and affects module count + the module graph.

## References

- ADR-0042 (pure replay design — the projectionhost side of this story)
- `middleware/deadletter.go`, `projectionhost/dlq.go`
- Plan C3 in `docs/planning/2026-06-29_brutal-self-review-execution-plan.md`
