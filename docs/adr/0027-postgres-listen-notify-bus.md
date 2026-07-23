# ADR-0027: Postgres LISTEN/NOTIFY Event Bus

> ⚠️ **DEPRECATED — DO NOT USE FOR NEW CODE.**
>
> This ADR was implemented and then superseded **one day later** by
> [ADR-0028](0028-watermill-as-delivery-layer.md), which adopts Watermill as
> the single delivery layer. The `storage.PostgresBus` and `PgxListener`
> code still exists for backward compatibility, but new consumers should
> use Watermill's SQL subscriber or broker plugins instead.
>
> The implementation effort (re-fetch logic, lifecycle management, channel
> allow-listing, three CI integration tests) was effectively wasted — the
> feature was built and immediately deprecated.

| Field   | Value                                                                                                               |
| ------- | ------------------------------------------------------------------------------------------------------------------- |
| Date    | 2026-06-19                                                                                                          |
| Status  | **Deprecated** — superseded by [0028](0028-watermill-as-delivery-layer.md) for new code                             |
| Decider | Lars Artmann                                                                                                        |

## Context

Every v2.7.0 Bundle preset (`stack/memory`, `stack/sqlite`, `stack/pebble`,
`stack/postgres`) wires an **in-process** `memory.NewMemoryBus()` for event
publish/subscribe. That is correct and sufficient for single-process
deployments — the common case for a library consumer.

The natural next step for the Postgres preset is a real distributed bus backed
by Postgres `LISTEN`/`NOTIFY`, so multiple processes sharing one database can
propagate events to each other. This was listed as a candidate v2.7.0 item.

## Decision

**Implement the LISTEN/NOTIFY bus and wire it into the Postgres preset via
an opt-in option. Ship the preset with the in-memory bus as default; consumers
opt into distributed pub/sub with `WithDistributedBus(listener)`.**

The design solves:

1. **8 KB NOTIFY payload limit.** Event payloads routinely exceed this, so
   `NOTIFY` carries only a lightweight reference (event ID + type + aggregate
   ref + version) and the listener **re-fetches** the full event from the event
   store. That couples the bus to the store and introduces an ordering/visibility
   question (the `NOTIFY` can arrive before the producing transaction is
   visible to the listener's connection) — handled via configurable retry.
2. **Listener lifecycle.** The `NotificationListener` interface encapsulates
   driver-specific LISTEN. The bus calls `Listen(channel)` itself, starts a
   background receive goroutine, and drains it cleanly on `Close` via
   `sync.WaitGroup` + context cancellation.
3. **Testing.** Three real-Postgres integration tests (`-tags=integration`)
   run in CI's `postgres-integration` job: end-to-end cross-bus delivery,
   channel validation, and full preset wiring.
4. **pgx-based listener.** `PgxListener` in `stack/postgres/` implements
   `NotificationListener` using a dedicated `pgxpool` connection. Channel-name
   allow-listing defends against LISTEN SQL injection (Postgres does not
   parameterize LISTEN).

## Implementation

- `storage.PostgresBus` — `event.Bus` implementation. Publishes via
  `SELECT pg_notify()`, receives via `NotificationListener.Notifications()`.
  Re-fetches events via `LoadByEventID` (O(1) indexed lookup on SQL stores)
  or falls back to `LoadFromVersion` scan (for stores that don't implement
  `EventByIDLoader`). OTel spans on publish + receive.
- `stack/postgres.PgxListener` — `NotificationListener` using `pgxpool`.
  `NewPgxListener(pool)` wraps an existing pool; `NewPgxListenerFromDSN(ctx, dsn)`
  creates an owned single-conn pool.
- `stack/postgres.WithDistributedBus(listener, busOpts...)` — preset option
  that swaps `memory.NewMemoryBus` for `storage.PostgresBus` and registers
  the bus for Close-time cleanup.

## Alternatives Considered

- **Ship a minimal version in v2.7.0.** Rejected initially: the re-fetch +
  lifecycle concerns make "minimal" fragile. Now fully implemented with
  proper lifecycle, testing, and type safety.
- **Use an outbox pattern instead.** Outbox + polling is more robust than
  `LISTEN`/`NOTIFY` but is a larger design still. Better suited to a dedicated
  `transport/`-style module (ADR-0025) for consumers who need exactly-once
  delivery guarantees.

## Consequences

- The Postgres preset's default is still in-memory (`memory.NewMemoryBus`).
  Consumers opt into distributed pub/sub explicitly via `WithDistributedBus`.
  This preserves backwards compatibility and doesn't force a pgxpool dependency
  on single-process deployments.
- Stores that don't implement `EventByIDLoader` (MemoryStore, Pebble) use the
  version-scan fallback for event refetch. This is O(events since version) per
  aggregate — efficient enough for the typical refetch pattern.
- The `notifyPayload` uses branded domain types (`id.EventID`, `event.Type`,
  etc.) for JSON (de)serialization, eliminating manual string parsing on the
  receive side.
- pgx is a direct dependency of `stack/postgres` (for both the driver
  registration and `PgxListener`). The `storage` module remains pgx-free;
  `NotificationListener` is driver-agnostic.

## Forward references

- §4 (M20) for the original kill-switch rationale.
- `ROADMAP.md` "Post-Bundle direction" lists this as the top multi-process theme.
