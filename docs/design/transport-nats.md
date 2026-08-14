# Design Spike: NATS Transport Adapter

> **⚠️ SUPERSEDED — this module will never be built.**
> ADR-0025 was corrected on 2026-06-28: broker transports are served by the
> existing [`watermill/`](../../watermill/) bridge, not native modules. Inject
> any Watermill-compatible backend (NATS JetStream, Redis Streams, Kafka, …)
> via `watermill.NewEventPublisher(publisher, topic)` or the
> `EventBus`/`CommandBus` `WithBackend()` / `WithCommandBackend()` options
> with the official
> [watermill-nats](https://github.com/ThreeDotsLabs/watermill-nats) plugin.
> This spike is retained as a historical reference only — do NOT implement it.
> For the sanctioned JetStream recipe (stream config, subject mapping, durable
> consumers, catch-up integration), see
> [`docs/planning/nats-transport-design.md`](../planning/nats-transport-design.md).

**Status:** Superseded by the `watermill/` bridge (ADR-0025).
**Module:** none — no `transport/nats/` module is planned.

## Problem

Consumers need a message broker-backed transport for distributing CQRS events across process boundaries. NATS JetStream provides at-least-once delivery with persistence, consumer groups, and acknowledgment — a natural fit for event sourcing projections.

## Design

### Module Structure

```
transport/nats/
├── go.mod          # github.com/larsartmann/go-cqrs-lite/transport/nats/v4
├── publisher.go    # EventPublisher → NATS JetStream
├── subscriber.go   # NATS JetStream → event subscription
├── protocol.go     # Event ↔ NATS message conversion
└── publisher_test.go
```

### Interfaces

**Publisher** implements `event.Publisher`:

```go
type Publisher struct {
    js     nats.JetStream
    topic  string
    codec  codec.Codec
}

func NewPublisher(js nats.JetStream, topic string, opts ...Option) *Publisher
func (p *Publisher) Publish(ctx context.Context, events ...event.Event) error
```

**Subscriber** implements `event.Subscriber`:

```go
type Subscriber struct {
    js     nats.JetStream
    codec  codec.Codec
}

func (s *Subscriber) Subscribe(ctx context.Context, eventType event.Type) (<-chan event.Event, error)
func (s *Subscriber) SubscribeAll(ctx context.Context) (<-chan event.Event, error)
```

### Key Design Decisions

1. **JetStream, not Core NATS** — Core NATS is fire-and-forget; JetStream provides persistence, redelivery, and consumer ack semantics needed for event sourcing.
2. **Subject mapping** — Event types map to NATS subjects: `cqrs.events.<eventType>`. SubscribeAll uses wildcard `cqrs.events.>`.
3. **Durable consumers** — Subscribers create durable consumers with ack. This enables catch-up/replay (similar to CatchUpSubscriber).
4. **No dependency on watermill/** — Direct NATS client, same pattern as transport/grpc. Keeps the dependency surface minimal.
5. **Codec reuse** — Uses `codec.Codec` for payload encoding (JSON default, CBOR opt-in).

### Dependencies

- `github.com/nats-io/nats.go` (production)
- `github.com/larsartmann/go-cqrs-lite/event/v4`
- `github.com/larsartmann/go-codec`

### Testing Strategy

- Unit tests with embedded NATS server (`github.com/nats-io/nats-server/test`)
- Integration test: publish → subscribe → verify delivery
- Order preservation test: events arrive in publish order within a single subject
