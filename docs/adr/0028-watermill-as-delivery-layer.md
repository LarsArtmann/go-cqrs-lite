# ADR-0028: Watermill as the Delivery Layer

| Field   | Value        |
| ------- | ------------ |
| Date    | 2026-06-20   |
| Status  | Accepted     |
| Decider | Lars Artmann |

## Context

The repository currently ships **five parallel bus implementations** for the same
job — moving events/commands/queries from producers to consumers:

1. `event.Bus` / `event.EventBus` (the imperative + reactive seams in `event/`)
2. `memory.MemoryBus` (`memory/bus.go`, ~390 LOC)
3. `memory.MemoryCommandBus` (`memory/command_bus.go`)
4. `storage.PostgresBus` (`storage/pg_bus.go`, LISTEN/NOTIFY, ghost in most presets)
5. Watermill adapter (`watermill/`) — exists but is only used by `stack/`

This is the classic "five buses" anti-pattern: every consumer must learn which bus
fits their deployment, every preset must wire one, and every bus re-implements
middleware (retry, dedup, tracing, poison queue) from scratch.

The deployer-first principle ("consumers should NOT decide on infrastructure;
deployers do") demands **one** delivery abstraction that scales from in-process
to distributed without changing consumer code.

## Decision

**Adopt [ThreeDotsLabs/watermill](https://github.com/ThreeDotsLabs/watermill) as
the single delivery layer for go-cqrs-lite.**

- `watermill/` becomes the canonical pub/sub adapter. It already wraps
  `message.Publisher` / `message.Subscriber` and converts cqrs events to/from
  `message.Message`.
- In-process deployments use Watermill's `gochannel` component — a battle-tested
  in-memory broker with ordered delivery and backpressure. This replaces
  `memory.MemoryBus`.
- Distributed deployments use Watermill's SQL/Kafka/AMQP components. This
  replaces `storage.PostgresBus`.
- Middleware (Retry, PoisonQueue/DLQ, CorrelationID, Throttle, Deduplicator) is
  provided by Watermill and composed on the `message.Router`.

### The one gap: replay / catch-up

Watermill has **no** built-in "replay from journal then hand off to live"
subscriber. Event-sourced systems need this for projections and read models. We
fill the gap with a thin `CatchUpSubscriber` (~300 LOC, ADR-0030) built on
Watermill's own `FanOut` subscriber pattern.

## Alternatives Considered

- **Build a new `bus/` module from scratch.** Rejected — reinvents Router,
  middleware chain, poison queue. Watermill already has these and is
  MIT-licensed, stable, and widely used.
- **Keep `memory.MemoryBus` as the default and add Watermill only for
  distributed.** Rejected — preserves the five-bus problem. The in-process
  `gochannel` is strictly better than our hand-rolled `MemoryBus`.
- **NATS / Kafka as first-class.** Rejected as a _required_ dependency. They
  remain available via Watermill adapters; consumers opt in.

## Consequences

- `memory/bus.go`, `memory/command_bus.go`, and `storage/pg_bus.go` become ghost
  code and are removed at the v3 boundary (T25).
- `event.Bus`, `event.Subscriber`, `event.Middleware`, and the reactive
  `EventBus` are removed at the v3 boundary. `event.Publisher` (one method,
  `Publish(ctx, ...Event) error`) **stays** — it is the Decider's emit seam, not
  a bus.
- `stack/*` presets switch their default bus from `memory.NewMemoryBus()` to
  Watermill's `gochannel.NewGoChannel`.
- Consumers targeting in-process only pay zero new external deps (gochannel is
  part of Watermill core). Distributed consumers add the Watermill backend they
  need.

## Forward references

- ADR-0030 (projection dissolution) depends on this — `CatchUpSubscriber`
  replaces `projection.Runner`.
- ADR-0027 (Postgres LISTEN/NOTIFY bus) is superseded for new code; the
  `PgxListener` lives on as a Watermill SQL-subscriber backend.
- Execution plan T11 (CatchUpSubscriber), T13 (Watermill adapter), T25 (ghost
  code deletion).
