# ADR-0030: Dissolve `projection/` into `CatchUpSubscriber` + `Materialize`

| Field   | Value        |
| ------- | ------------ |
| Date    | 2026-06-20   |
| Status  | Implemented  |
| Decider | Lars Artmann |

> **Cross-reference:** This ADR dissolved the old `projection/` module
> (which was a bus masquerading as a projection runner). The `projection/`
> name was subsequently **reused** by [ADR-0037](0037-projection-module-extraction.md),
> which extracted the `Projection` _interface_ into a new minimal `projection/`
> module (Layer 2, interface-only). The two `projection/` packages are
> different things: the old one was removed; the new one is a 1-file interface.

## Context

The `projection/` module is **named a lie**. Its central type,
`projection.Runner`, is a message bus — it requires an `event.Subscriber` for its
live tail, manages a worker pool, deduplicates, retries, routes to a dead-letter
queue, and dispatches to registered handlers. That is the job of a delivery
layer, not a projection.

Meanwhile the actual _projection_ concern — "turn a stream of events into a
materialized view" — is buried inside `projection/handler.go` (97 LOC) and
`projection/builder.go`'s `On[T]()` registry (~100 LOC).

This conflates two orthogonal responsibilities:

1. **Delivery**: replay-from-journal + live-handoff + dedup + retry. → belongs
   in the delivery layer (Watermill, per ADR-0028).
2. **Materialization**: `OnCreate`/`OnUpdate`/`OnTombstone`/`OnRebirth` handlers
   that write to a `kv.TypedStore`. → belongs in `stack/` (deployer concern).

## Decision

**Split `projection/` along its true seam.**

### 1. `watermill.CatchUpSubscriber` (delivery)

A `message.Subscriber` implementation (~300 LOC) that:

1. Loads the last checkpoint from a `CheckpointStore`.
2. **Replays** events from a `SeekableJournal.ReadFrom(checkpoint)` into a
   Watermill `GoChannel`, marking each message with
   `ProcessingMode = ModeReplay`.
3. **Hands off** to a live `message.Subscriber`, deduplicating the overlap
   (events already seen during replay).
4. After each `Ack`, persists the new event ID to the checkpoint.

This is a faithful port of the battle-tested replay loop in
`projection/runner.go` + `projection/runner_live.go` (~500 LOC combined), minus
the handler-dispatch logic that does not belong here.

### 2. `stack.Materialize[V, K]` (materialization)

A typed facade that turns an event stream into a materialized view in a
`kv.TypedStore[V, K]`:

```go
mat := stack.Materialize[UserView, UserID]{
    Store:    kvStore,
    Decide:   func(evt event.Event, existing *V) (op MatOp, err error),
    OnCreate: func(ctx, evt) (*V, error),
    OnUpdate: func(ctx, evt, existing *V) (*V, error),
    OnTombstone: func(ctx, evt, existing *V) error,
    OnRebirth:   func(ctx, evt, existing *V) (*V, error),
}
router.AddHandler("users", topic, catchUpSub, "users_view", gochannelPub, mat.HandlerFunc())
```

`Materialize` is **tombstone-aware** (ADR-0006): there are no hard deletes. The
query policy (`IncludeTombstoned` / `ExcludeTombstoned` / `OnlyTombstoned`)
controls visibility.

## Alternatives Considered

- **Keep `projection/` and rename it `delivery/`.** Rejected — the
  materialization API is the valuable part and it should live next to the read
  model (`kv/`), not in a generic delivery package.
- **Put `Materialize` in `projection/` and fix the bus part.** Rejected —
  preserves the split-brain. Two packages with one job each is cleaner.
- **Make `Materialize` untyped (`any` payload).** Rejected — violates the
  strong-types principle. Generics give us `Materialize[UserView, UserID]` with
  zero runtime casts.

## Consequences

- `projection/` is removed at the v3 boundary. Existing consumers migrate:
  - `projection.Runner` → `watermill.CatchUpSubscriber` + Watermill `Router`.
  - `projection.Builder.On[T]()` → `stack.Materialize` handler fields.
  - `projection.HandlerRegistry` → Watermill router handler registration.
- The leader-election, health, and distributed-runner pieces in `projection/`
  stay relevant but are rehomed (leader election is a deployer concern, likely
  `stack/` or a dedicated `coordination/` module in a later ADR).
- Read-model rebuild becomes "drop the checkpoint table and restart the
  subscriber" — identical operational model, far less code.

## Forward references

- Depends on ADR-0028 (Watermill) and ADR-0031 (Materialize uses typed Metadata).
- Execution plan T11 (CatchUpSubscriber), T12 (Materialize).
- Supersedes the "Saga via projection" note in AGENTS.md — sagas now compose
  from CatchUpSubscriber + command dispatch on the same Watermill router.
