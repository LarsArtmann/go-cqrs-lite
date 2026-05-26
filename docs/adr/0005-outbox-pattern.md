# ADR-0005: Outbox Pattern for Reliable Event Publishing

**Status:** Accepted  
**Date:** 2026-05-26

## Context

In CQRS + Event Sourcing, events must be reliably delivered to subscribers. The naive approach — saving events to the store, then publishing to the bus — has a fatal flaw: if the application crashes between `Save` and `Publish`, the event is persisted but never broadcast. Subscribers miss it permanently.

Common workarounds have their own problems:

- **In-process bus inside the save transaction** — couples store to bus, prevents async consumers
- **Two-phase commit (2PC)** — complex, slow, not supported by all databases
- **Change data capture (CDC)** — external infrastructure (Debezium), operational complexity

## Decision

Implement the Outbox pattern with three components:

1. **`event.Outbox` interface** — `Append(ctx, events)`, `PollPending(ctx, limit)`, `Ack(ctx, ids)`
2. **`SQLOutbox` + `OutboxSchema`** — stores event batches as JSON in an `outbox` table with `status` column (`pending` / `acknowledged`)
3. **`OutboxPoller`** — background worker that polls pending entries, publishes each event via `event.Publisher`, and acknowledges successful batches

The outbox table uses the same transaction as event storage via `TransactionalStore.SaveWithOutbox`, ensuring atomicity: either both the events and outbox entry commit, or neither does.

The `OutboxPoller` is intentionally separate from the store:

- `Start(ctx)` begins polling on a configurable interval
- `Stop()` signals graceful shutdown
- Failed entries are skipped (not acknowledged) and retried on next poll
- Batch size and interval are configurable via functional options

## Consequences

**Positive:**

- Exactly-once delivery semantics — events are either stored+outboxed or neither
- Decoupled publishing — poller runs independently, consumers can be async
- Resilient to crashes — pending outbox entries are replayed on restart
- Observable — poll metrics, pending entry counts, processed rates
- Minimal infrastructure — single SQL table, no external CDC needed

**Negative:**

- Slight latency — events are delivered after poll interval, not instantaneously
- Additional storage — outbox entries exist until acknowledged (can be archived/deleted)
- Requires `event.Outbox` + `event.Publisher` wiring in consumer applications

**Neutral:**

- The outbox schema stores events as JSON batches (not individual rows) — simpler schema, but harder to query individual events
- `OutboxPoller` acks at the batch level — partial batch failure retries the entire batch
