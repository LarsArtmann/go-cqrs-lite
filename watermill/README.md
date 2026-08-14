# watermill — Watermill Protocol Adapter (Event Bus + Command Bus + CatchUp)

[![Go Reference](https://pkg.go.dev/badge/github.com/larsartmann/go-cqrs-lite/watermill/v4.svg)](https://pkg.go.dev/github.com/larsartmann/go-cqrs-lite/watermill/v4)

Protocol adapters between go-cqrs-lite event/command interfaces and the [Watermill](https://watermill.io/) message bus library. Provides the default in-process EventBus, CatchUpSubscriber (replay + live), EventPublisher, and CommandBus.

**This is the canonical broker delivery path** ([ADR-0127](../docs/adr/0127-deprecate-transport-modules.md)): external fanout goes through this module with any Watermill backend — the `transport/*` modules are deprecated and removed at v5.

```bash
go get github.com/larsartmann/go-cqrs-lite/watermill/v4
```

## Components

### EventBus (default event bus)

GoChannel-backed in-process event bus. Replaces the deprecated `memory.MemoryBus`. Ordered delivery via `BlockPublishUntilSubscriberAck=true`.

```go
bus := watermill.NewEventBus()
defer bus.Close()

bus.Subscribe("user.created", handler)     // typed subscription
bus.SubscribeAll(catchAllHandler)          // catch-all (audit, derivers)
bus.Use(middleware.EventTracing(tracer))   // middleware chain
bus.Publish(ctx, evt1, evt2)               // variadic publish
```

### CatchUpSubscriber (replay + live handoff)

Replays historical events from a journal, then hands off to a live subscriber. Used by projection hosts and SSE brokers.

```go
catchUp, _ := watermill.NewCatchUpSubscriber(journal, liveSub, cpStore, logger)
defer catchUp.Close()

msgs, _ := catchUp.Subscribe(ctx, "user.created")
// Phase 1: replay historical events with ProcessingMode=ModeReplay
// Phase 2: live handoff with EventID-based deduplication
// Checkpoint saved after every forwarded event
```

### EventPublisher

Publishes CQRS events to a Watermill topic. W3C trace context is injected into message metadata.

```go
pub := watermill.NewEventPublisher(wmPublisher, "events")
repo, _ := decider.NewRepository(store, pub, decider)
```

### CommandBus

Command distribution over any broker. GoChannel for single-process; inject NATS/Redis/Kafka for multi-process.

```go
bus := watermill.NewCommandBus()
defer bus.Close()
bus.Subscribe("user.create", handlerFunc)
bus.Publish(ctx, cmd)
```

## External Brokers (canonical delivery path)

Inject any Watermill-compatible backend via `WithBackend` (events) and
`WithCommandBackend` (commands). Example with Redis Streams via the official
[`watermill-redisstream`](https://github.com/ThreeDotsLabs/watermill-redisstream)
plugin:

```go
client := redis.NewClient(&redis.Options{Addr: "localhost:6379"})

pub, _ := redisstream.NewPublisher(redisstream.PublisherConfig{Client: client}, watermill.NopLogger{})
sub, _ := redisstream.NewSubscriber(redisstream.SubscriberConfig{Client: client}, watermill.NopLogger{})

eventBus := watermill.NewEventBus(watermill.WithBackend(pub, sub, client))
cmdBus := watermill.NewCommandBus(watermill.WithCommandBackend(pub, sub, client))
```

The full roundtrip (publish → broker → typed subscribe, events + commands) is
verified against a real Redis broker by `TestRedisStreamRoundtrip` — run it
via the ephemeral broker scripts (no Docker needed):

```bash
bash scripts/ephemeral-redis.sh sh -c 'cd watermill && go test -tags "goexperiment.jsonv2" -run TestRedis -v .'
```

**Kafka/RabbitMQ/etc.**: any Watermill plugin works the same way — construct
the plugin's publisher/subscriber and pass them to `WithBackend`/
`WithCommandBackend`.

**NATS JetStream**: no maintained Watermill plugin exists today
(`watermill-nats` is NATS Streaming — deprecated technology built against a
Watermill release candidate). Revisit once a JetStream subscriber adapter
exists; `scripts/ephemeral-nats.sh` is ready for it.

## Ordering

> **Watermill Router processes messages in parallel** (one goroutine per message). Do NOT route ordered projections through the Router. Instead, consume the CatchUpSubscriber's output channel from a single goroutine (FIFO guarantees ordering).

The EventBus default GoChannel uses `BlockPublishUntilSubscriberAck=true` + `Persistent=false`: the former ensures ordered live delivery, the latter avoids GoChannel's unordered Persistent-mode replay (CatchUpSubscriber handles replay from the journal instead).

## Middleware

```go
// Watermill middleware wrappers
router.AddMiddleware(watermill.CorrelationIDMiddleware())
router.AddMiddleware(watermill.NewRetryMiddleware(watermill.DefaultRetryConfig()))
router.AddMiddleware(watermill.ProcessingModeMiddleware()) // reconstruct replay/live flag
router.AddMiddleware(watermill.TraceContextMiddleware())
```

## Design

- **CatchUpSubscriber**: Always-ordered replay phase. Live phase uses `BlockPublishUntilSubscriberAck=true`.
- **Deduplication**: EventID-based dedup at the replay-to-live boundary prevents duplicate delivery.
- **Checkpoint**: Saved after every forwarded event. Restart resumes from the last checkpoint.
- **Context propagation**: W3C trace context injected on publish, extracted on consume.

## Related Modules

- [**event**](../event/README.md) — Event bus and publisher interfaces
- [**command**](../command/README.md) — Command bus interface
- [**projectionhost**](../projectionhost/README.md) — Reads from SeekableJournal directly (no Watermill dependency)
- [**dedup**](../dedup/README.md) — Ring buffer used by CatchUpSubscriber
