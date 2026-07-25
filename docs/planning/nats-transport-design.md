# NATS JetStream Transport Design

> **Date:** 2026-07-25 · **Status:** Design (Tier 4 expansion)
> **Related:** [ADR-0025](../adr/0025-transport-adapter-strategy.md), [watermill/](../../watermill/)

---

## Context

ADR-0025 accepted the transport adapter strategy: transport-specific heavy deps
(nats.go, go-redis) stay out of core via the `watermill/` bridge. The
`watermill/` module implements `event.Bus` and `command.Bus` by bridging to any
Watermill-compatible `message.Publisher`/`message.Subscriber`. NATS JetStream
is a first-class backend candidate.

**No `transport/nats/` module is planned.** The sanctioned path is injecting
JetStream components into the existing `watermill.EventBus` and
`watermill.CommandBus` via `WithBackend()` / `WithCommandBackend()`.

This document specifies the JetStream recipe: stream configuration, subject
mapping, durable consumers, and integration with the catch-up/replay pipeline.

## Architecture

```
┌─────────────────────────────────────────────────────────┐
│  Consumer Application                                    │
│  (imports watermill/)                                    │
│                                                         │
│  watermill.EventBus    watermill.CommandBus              │
│       ↓ Publish            ↓ Publish                     │
│  EventPublisher        CommandPublisher                   │
│       ↓                    ↓                              │
│  message.Publisher     message.Publisher                  │
│       ↓                    ↓                              │
│  ┌─────────────────────────────────────────┐             │
│  │  watermill-nats JetStream Publisher      │             │
│  │  (subject: cqrs.events / cqrs.commands)  │             │
│  └─────────────┬───────────────────────────┘             │
│                ↓                                         │
│  ┌─────────────────────────────────────────┐             │
│  │  NATS JetStream                          │             │
│  │  Stream: EVENTS (subject: cqrs.events)   │             │
│  │  Stream: COMMANDS (subj: cqrs.commands)  │             │
│  └─────────────┬───────────────────────────┘             │
│                ↓                                         │
│  ┌─────────────────────────────────────────┐             │
│  │  watermill-nats JetStream Subscriber     │             │
│  │  (durable consumer per service instance) │             │
│  └─────────────┬───────────────────────────┘             │
│                ↓                                         │
│  CatchUpSubscriber  ←  journal replay + live handoff     │
│       ↓                                                  │
│  projectionhost / command handlers / event handlers       │
└─────────────────────────────────────────────────────────┘
```

## Stream Configuration

### Event Stream

```hocon
# JetStream stream for CQRS events
Stream {
  Name: "EVENTS"
  Subjects: ["cqrs.events"]
  Retention: Limits
  MaxAge: 720h   # 30 days retention for replay
  Storage: File
  Replicas: 1    # R1 for single-node, R3 for HA
}
```

**Subject:** Flat single-subject per bus. The `event_type` metadata field (set by
`watermill.EventToMessage`) carries the type for routing — consumers filter by
metadata, not by subject wildcard. This matches the existing `watermill/` wire
protocol where `MessageToEvent` reads `event_type` from metadata.

**Retention:** `Limits` mode with time-based expiry. Events are immutable; the
stream serves as a temporary delivery buffer + short-term replay source. The
authoritative event store is the database (SQLite/Postgres/Pebble). If the stream
expires, the `CatchUpSubscriber` falls back to journal-based replay from the DB.

### Command Stream

```hocon
Stream {
  Name: "COMMANDS"
  Subjects: ["cqrs.commands"]
  Retention: WorkQueue
  MaxAge: 24h
  Storage: File
}
```

**Retention:** `WorkQueue` mode — each command is delivered to exactly one
consumer (load-balanced). This matches CQRS command semantics: a command is
handled by exactly one handler instance.

## Durable Consumers

### Event Consumer (Per Service)

```
Consumer {
  Stream: "EVENTS"
  Name: "projection-host-{{.ServiceName}}"
  FilterSubject: "cqrs.events"
  DeliverPolicy: DeliverAll
  AckPolicy: Explicit
  MaxDeliver: -1    # unlimited redelivery for at-least-once
  AckWait: 30s
}
```

**Naming:** `{service}-{projection}` so each projection host gets its own
consumer offset. Multiple projections in the same process share one consumer
(the projection host fans out internally).

**Delivery:** `DeliverAll` on first start, then relies on the consumer offset.
Combined with `CatchUpSubscriber`, this gives:

1. **Phase 1 (replay):** `CatchUpSubscriber` reads from `event.SeekableJournal`
   starting at the last checkpoint. JetStream consumer offset stays at zero.
2. **Phase 2 (live handoff):** JetStream delivers from the current position.
   `CatchUpSubscriber` deduplicates events seen during replay (bounded ring buffer
   keyed by `event_id`).

### Command Consumer (Per Service)

```
Consumer {
  Stream: "COMMANDS"
  Name: "command-handler-{{.ServiceName}}"
  FilterSubject: "cqrs.commands"
  DeliverPolicy: DeliverAll
  AckPolicy: Explicit
  MaxDeliver: 5     # bounded retry for poison commands
  AckWait: 60s
}
```

**MaxDeliver:** Bounded retry (default 5). After max deliveries, the message
goes to JetStream's `Advisory` system. The application should listen for
`com.nats.jetstream.advisory.max_deliveries` advisories to route poison
commands to a dead-letter mechanism.

## Wiring Recipe

```go
import (
    "github.com/nats-io/nats.go"
    "github.com/nats-io/nats.go/jetstream"
    natswatermill "github.com/ThreeDotsLabs/watermill-nats/v2/pkg/nats"

    "github.com/larsartmann/go-cqrs-lite/watermill/v4"
)

// 1. Connect to NATS
nc, _ := nats.Connect(natsURL)
js, _ := jetstream.New(nc)
defer nc.Close()

// 2. Create JetStream publisher + subscriber for events
eventPublisher, _ := natswatermill.NewPublisher(
    natswatermill.PublisherConfig{
        JetStream: js,
        Marshaler: natswatermill.NATSMarshaler{},
    },
    logger,
)

eventSubscriber, _ := natswatermill.NewSubscriber(
    natswatermill.SubscriberConfig{
        JetStream:         js,
        ConsumerName:      "projection-host-users",
        DurablePrefix:     "users",
        ManualAck:         true,
    },
    logger,
)

// 3. Create EventBus with JetStream backend
eventBus := watermill.NewEventBus(
    watermill.WithBackend(eventPublisher, eventSubscriber, eventPublisher),
)
defer eventBus.Close()

// 4. Create CommandBus with JetStream backend
cmdPublisher, _ := natswatermill.NewPublisher(...)
cmdSubscriber, _ := natswatermill.NewSubscriber(...)
commandBus := watermill.NewCommandBus(
    watermill.WithCommandBackend(cmdPublisher, cmdSubscriber, cmdPublisher),
)
defer commandBus.Close()

// 5. Wire catch-up subscriber for projections
journal := eventStore // must implement event.SeekableJournal
cpStore := checkpointStore
catchUp, _ := watermill.NewCatchUpSubscriber(journal, eventSubscriber, cpStore, logger)
defer catchUp.Close()

// 6. Register projections with the host
host, _ := projectionhost.New(catchUp, cpStore,
    projectionhost.WithBatchSize(100),
)
host.Register(&UserProjection{})
go host.Start(ctx)
defer host.Stop()
```

## Topic Mapping Summary

| CQRS Concept      | JetStream Mapping                          |
| ----------------- | ------------------------------------------ |
| `event.Type`      | `event_type` metadata field on message     |
| `command.Type`    | `command_type` metadata field on message   |
| `event.EventID`   | `message.UUID` (NATS MsgId for dedup)      |
| Event bus topic   | Subject `cqrs.events`, Stream `EVENTS`     |
| Command bus topic | Subject `cqrs.commands`, Stream `COMMANDS` |
| Tracing           | W3C `traceparent` in message metadata      |
| Codec (CBOR/JSON) | `payload_encoding` metadata field          |
| Tombstone         | `tombstone_status` metadata field          |

## JetStream-to-CatchUpSubscriber Interaction

The `CatchUpSubscriber` is designed to work with ANY `message.Subscriber`,
including JetStream. The interaction:

```
Time ──────────────────────────────────────────────────►

[DB event store]    [JetStream EVENTS stream]
      │                        │
      │ Phase 1: Replay        │ Phase 2: Live
      │ ReadFrom(checkpoint)   │ Subscribe (durable consumer)
      │         │              │         │
      │    ┌────┴────┐         │    ┌────┴────┐
      │    │ dedup   │         │    │ dedup   │
      │    │ ring    │◄────────┼────┤ ring    │
      │    └────┬────┘         │    └────┬────┘
      │         │              │         │
      │    replay channel      │    live channel
      │         │              │         │
      └─────────┴──────────────┴─────────┘
                           │
                    output channel (256 buffer)
                           │
                    projectionhost.Worker
```

**Key:** JetStream's durable consumer provides native at-least-once + offset
persistence. The `CatchUpSubscriber` adds:

- Cross-source dedup (events in both DB replay and JetStream live)
- Checkpoint persistence (projection-name keyed, survives restarts)
- Processing mode tagging (replay vs live in context)

## Error Handling

| Scenario                | Behavior                                              |
| ----------------------- | ----------------------------------------------------- |
| NATS connection lost    | `natswatermill.Subscriber` auto-reconnects (built-in) |
| JetStream stream full   | Publish returns error; bus middleware can retry       |
| Consumer max deliveries | JetStream advisory; app routes to DLQ                 |
| Poison message          | `projectionhost.WithDeadLetterStore` catches it       |
| Replay checkpoint gap   | `CatchUpSubscriber` replays from last checkpoint      |

## When to Use Native JetStream Replay vs CatchUpSubscriber

| Scenario                                | Recommendation                                    |
| --------------------------------------- | ------------------------------------------------- |
| Event store IS JetStream (KV or stream) | Native `DeliverAll` + durable consumer            |
| Event store is SQL/Pebble DB            | `CatchUpSubscriber` (DB is authoritative)         |
| Multi-projection, same events           | One durable consumer + `CatchUpSubscriber`        |
| Exactly-once required                   | Neither — use idempotency keys + `CheckAndRecord` |

## Dependencies

```go
// go.mod additions for consumers using NATS
require (
    github.com/nats-io/nats.go v1.x
    github.com/ThreeDotsLabs/watermill-nats/v2 v2.x
)
```

No new dependency is added to `go-cqrs-lite`. The `watermill/` module already
accepts `message.Publisher`/`message.Subscriber` interfaces — the NATS adapter
plugs in without core changes.

## Future Considerations

1. **JetStream KV as checkpoint store** — `event.CheckpointStore` backed by
   JetStream KV bucket for multi-process checkpoint sharing. ~50 LOC adapter.

2. **Multi-region JetStream** — mirror streams across regions for geo-replication.
   Requires `Subjects: ["cqrs.events.>"]` with region prefix.

3. **NATS request/reply for queries** — the current design uses gRPC for
   query dispatch. NATS core supports request/reply natively, but the
   request/reply semantics differ from JetStream's at-least-once model.
   Deferred until a use case requires it.
