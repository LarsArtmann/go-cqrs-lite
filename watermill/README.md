# watermill — Watermill Protocol Adapter (Event Bus + Command Bus + CatchUp)

[![Go Reference](https://pkg.go.dev/badge/github.com/larsartmann/go-cqrs-lite/watermill/v4.svg)](https://pkg.go.dev/github.com/larsartmann/go-cqrs-lite/watermill/v4)

Protocol adapters between go-cqrs-lite event/command interfaces and the [Watermill](https://watermill.io/) message bus library. Provides the default in-process EventBus, CatchUpSubscriber (replay + live), EventPublisher, and CommandBus.

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
