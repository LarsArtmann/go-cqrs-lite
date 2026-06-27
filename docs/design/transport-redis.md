# Design Spike: Redis Transport Adapter

**Status:** Accepted (ADR-0025). Implementation pending.
**Module:** `transport/redis/`

## Problem

Some consumers already operate Redis and prefer it over introducing NATS. Redis Streams provides consumer groups, persistence, and acknowledgment — sufficient for event distribution across processes.

## Design

### Module Structure

```
transport/redis/
├── go.mod          # github.com/larsartmann/go-cqrs-lite/transport/redis/v3
├── publisher.go    # EventPublisher → Redis Streams
├── subscriber.go   # Redis Streams → event subscription
├── protocol.go     # Event ↔ Redis message conversion
└── publisher_test.go
```

### Interfaces

**Publisher** implements `event.Publisher`:

```go
type Publisher struct {
    client    *redis.Client
    streamKey string
    codec     codec.Codec
}

func NewPublisher(client *redis.Client, streamKey string, opts ...Option) *Publisher
func (p *Publisher) Publish(ctx context.Context, events ...event.Event) error
```

**Subscriber** implements `event.Subscriber`:

```go
type Subscriber struct {
    client    *redis.Client
    streamKey string
    codec     codec.Codec
    group     string
    consumer  string
}

func (s *Subscriber) Subscribe(ctx context.Context, eventType event.Type) (<-chan event.Event, error)
```

### Key Design Decisions

1. **Redis Streams, not Pub/Sub** — Pub/Sub is fire-and-forget (no persistence). Streams provide at-least-once delivery with consumer groups and pending entries lists (PEL).
2. **Consumer groups** — Each subscriber creates a consumer group. Multiple instances of the same projection share a group for load-balanced delivery.
3. **XADD MAXLEN trimming** — Optional stream trimming via `MAXLEN ~N` to cap memory usage. Configurable, off by default.
4. **Event type filtering** — Redis Streams doesn't natively filter by type. Subscriber reads all messages and filters by event type field in the XADD hash. For high-throughput, use separate streams per event type (configurable).
5. **XAUTOCLAIM for stale messages** — Reclaim pending messages from crashed consumers after a configurable timeout.

### Dependencies

- `github.com/redis/go-redis/v9` (production)
- `github.com/larsartmann/go-cqrs-lite/event/v3`
- `github.com/larsartmann/go-cqrs-lite/codec/v3`

### Testing Strategy

- Unit tests with miniredis (`github.com/alicebob/miniredis/v2`)
- Integration test: XADD → XREADGROUP → verify delivery + ack
- Consumer group failover test: simulate crash, verify XAUTOCLAIM reclaims
