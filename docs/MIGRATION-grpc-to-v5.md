# Migrating off transport/grpc before v5

> **Audience:** consumers of `github.com/larsartmann/go-cqrs-lite/transport/grpc/v4`.
> The module is deprecated (ADR-0127) and DELETED at the v5 cut. This guide
> maps every exported capability to its replacement so mesh consumers are
> unbroken by the cut. The general v5 horizon is ADR-0123; the sanctioned
> external-delivery decision is ADR-0127.

## What the module actually gave you

Four thin adapters over grpc-go — no domain logic, no delivery guarantees
of its own:

| Export                          | What it did                                                       |
| ------------------------------- | ----------------------------------------------------------------- |
| `RegisterCommandService`        | served `Dispatch` RPCs by calling your local `command.Dispatcher` |
| `NewCommandClient().Dispatch`   | sent a command to a remote dispatcher (payload + metadata)        |
| `NewQueryClient().Ask`          | sent a query, decoded the JSON reply into `out`                   |
| `NewEventClient` / event server | streamed events over gRPC streams                                 |
| `error_mapping.go`              | preserved the errorfamily taxonomy (family/code) across the wire  |

## The mapping

| transport/grpc feature                   | v5 replacement                                                                                                                                                                                            |
| ---------------------------------------- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Remote COMMAND dispatch (RPC semantics)  | `watermill.NewCommandPublisher(publisher, topic)` over your broker — commands are messages; at-least-once + idempotency middleware dedupes                                                                |
| Remote QUERY dispatch (request/response) | Expose queries as HTTP on your own stack (the payload codec is JSON; `query.Dispatcher.Ask` stays local) — or serve the read model via `metaengine.ServeSSE` when consumers can subscribe instead of poll |
| Event streaming to remote consumers      | `watermill.CatchUpSubscriber` (durable checkpoint, crash-restart) for services; `metaengine.ServeSSE[V]` (recent-window replay, `Last-Event-ID`) for browsers                                             |
| Error taxonomy over the wire             | handler errors do NOT return over async messaging; outcomes arrive as command-lifecycle EVENTS — `command.rejected` carries the errorfamily family + code (`commandlifecycle`, ADR-0117)                  |
| Bidirectional gRPC specifically          | Nothing in go-cqrs-lite replaces raw gRPC — bridge your dispatcher over grpc-go yourself if you must keep the protocol; the adapter was ~100 lines, deliberately reproducible                             |

## Decision help: which replacement fits

- **You called `Dispatch` and waited for the handler's error** (command RPC
  semantics): the closest replacement is command-as-message over a broker —
  you lose the synchronous error return and gain retries, but the
  command-lifecycle stream (`commandlifecycle`, ADR-0117) gives you the
  outcome as events: `command.accepted` / `command.rejected` (with the
  errorfamily code stamped on rejections).
- **You polled `Ask` for state**: prefer flipping to a subscription —
  `ServeSSE` for browser dashboards, `CatchUpSubscriber` for services; both
  are cheaper than repeated remote queries and survive v5.
- **You streamed events over gRPC streams**: `CatchUpSubscriber` is the
  durable twin (checkpoint store vs. lost-on-reconnect gRPC streams).

## Worked shape: command dispatch over NATS

```go
// skip-validate: illustrative — see watermill/README.md for the full runnable version
pub, err := watermill.NewCommandPublisher(natsPublisher, "commands.orders")
err = pub.Publish(ctx, placeOrderCmd)
// consumer side: watermill router handler dispatches into the LOCAL
// command.Dispatcher; idempotency middleware dedupes redeliveries
```

The full broker matrix (NATS JetStream / Redis Streams / Kafka / RabbitMQ),
ordering and redelivery semantics, and the transactional-outbox pattern
(journal-as-outbox, ADR-0016/0146) live in
[watermill/README.md](../watermill/README.md) and the
[watermill skill](../.agents/skills/watermill/SKILL.md).

## Timeline

- Through v4.x: the module compiles and works; cqrs-lint flags new usage.
- v5 cut (ADR-0123 Phase 8): module deleted; imports break HERE if you have
  not migrated. The FAQ's "Will the v5 cut break my imports?" entry carries
  the canonical removal list.
