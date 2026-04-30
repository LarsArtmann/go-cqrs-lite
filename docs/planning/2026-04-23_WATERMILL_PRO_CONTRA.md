# Watermill Integration — PRO / CONTRA Analysis

**Date:** 2026-04-23
**Status:** Decision pending
**Related:** `event/store.go`, `event/bus.go`, `event/store_config.go`

## Current State

go-cqrs-lite currently has **only in-memory implementations**:

| Component    | Interface      | Implementation  | File                    |
| ------------ | -------------- | --------------- | ----------------------- |
| Event Store  | `event.Store`  | `MemoryStore`   | `event/memory_store.go` |
| Event Bus    | `event.Bus`    | `MemoryBus`     | `event/memory_bus.go`   |
| Store Config | `Backend` type | `"memory"` only | `event/store_config.go` |

Persistent backends (PostgreSQL event store, NATS/JetStream bus) are listed as unchecked TODOs in `TODO_LIST.md` and various status docs.

No Watermill or ThreeDotsLabs references exist anywhere in the codebase.

## PRO — Adding Watermill

1. **Avoid reinventing the wheel** — Battle-tested pub/sub implementations for Kafka, NATS, RabbitMQ, Redis Streams, SQL, Google Cloud Pub/Sub, and more. Building from scratch would take months.
2. **Fits existing interfaces** — `event.Store` (Save/Load/AppendBatch) and `event.Bus` (Publish/Subscribe) map naturally to Watermill's `Publisher`/`Subscriber` and its CQRS event store.
3. **Built-in CQRS/ES package** — `watermill/components/cqrs` has event store, event bus, command bus, and saga support — strong conceptual overlap.
4. **Observability** — Metrics (Prometheus), tracing (OpenTelemetry), and structured logging out of the box.
5. **Production-ready** — Used at scale, well-documented, Apache 2.0 license.

## CONTRA — Adding Watermill

1. **Breaks zero-dependency principle** — Core selling point is _"Zero external dependencies — Only stdlib + google/uuid + cockroachdb/errors"_. Watermill is heavy.
2. **Overlapping abstractions** — Watermill has its own CQRS types that may conflict with or duplicate go-cqrs-lite's `command.Dispatcher`, `query.Dispatcher`, `event.Bus`, `aggregate.Aggregate`. Users may ask: "why not just use Watermill directly?"
3. **Module bloat** — Watermill brings transitive dependencies per broker. Even one backend adds significant module weight.
4. **Scope creep** — go-cqrs-lite is a _library_ for structuring apps. Watermill is a _framework_ for message routing. Adding it shifts toward being a Watermill wrapper.
5. **Alternative path exists** — Thin backends (PostgreSQL store, NATS bus) could be implemented directly with minimal code, maintaining the zero-dep philosophy. The `Backend` config system in `store_config.go` already anticipates this.

## Recommendation

Keep Watermill out of the core library. Instead, create a **separate Go module** (e.g., `go-cqrs-lite/watermill` or a companion repo) that provides `event.Store` and `event.Bus` adapters backed by Watermill. This preserves the zero-dep core while offering opt-in integration for users who want production broker support.

### Suggested Approach

```
go-cqrs-lite/              ← core (zero deps, stays as-is)
go-cqrs-lite/watermill/    ← opt-in module (go.mod with watermill dep)
go-cqrs-lite/postgres/     ← opt-in module (thin pgx/sqlx store)
go-cqrs-lite/nats/         ← opt-in module (thin nats.go bus)
```

Each adapter module would implement the existing `event.Store` and `event.Bus` interfaces, registered via the `Backend` config system.
