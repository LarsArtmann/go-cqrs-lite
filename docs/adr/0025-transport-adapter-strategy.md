# ADR-0025: Transport Adapter Strategy

| Field   | Value        |
| ------- | ------------ |
| Date    | 2026-06-19   |
| Status  | Implemented  |
| Decider | Lars Artmann |

## Context

Consumers need to dispatch commands and queries across process boundaries
(gRPC, NATS, Redis Streams, HTTP). The library currently has no transport
module — all dispatch is in-process.

## Decision

**Transport adapters are separate modules, not part of core.** Each transport
gets its own module with its own go.mod, importing only `command/`, `query/`,
and `event/`.

### Module Layout (future)

```
transport/
├── grpc/       # gRPC command/query dispatch
├── nats/       # NATS JetStream adapter
├── redis/      # Redis Streams adapter
└── http/       # REST/JSON dispatch (moved from middleware/ in v3)
```

### Why Separate Modules?

1. **Heavy dependencies** — gRPC pulls protobuf + grpc-go (~20MB). NATS pulls
   nats.go. Redis pulls go-redis. Each consumer imports only what they use.
2. **Independent versioning** — transport protocols evolve independently of
   the core CQRS library.
3. **Build tag isolation** — transports can use build tags for features like
   TLS, tracing, compression without polluting core.

### Contract

Each transport adapter implements these existing interfaces:

```go
// Command transport — wraps command.Dispatcher
type CommandTransport interface {
    Send(ctx context.Context, cmd command.Command) error
    Receive(ctx context.Context) (<-chan command.Command, error)
}

// Query transport — wraps query.Dispatcher
type QueryTransport interface {
    Ask(ctx context.Context, q query.Query) (any, error)
    Serve(ctx context.Context, handler query.Handler) error
}
```

These are consumer-side interfaces, not in core. Each transport module
defines its own wire format.

## Consequences

- **+** Core stays dependency-free
- **+** Consumers pick exactly one transport
- **+** New transports can be added without touching core
- **-** No built-in cross-service dispatch until modules are created
- **-** Wire format is transport-specific (no universal serialization)

## Status

- **gRPC**: **Implemented.** `transport/grpc/` module with CommandService and QueryService. Proto at `transport/grpc/proto/cqrs.proto`. Server adapters wrap `command.Dispatcher` and `query.Dispatcher`. Client adapters provide remote dispatch.
- **NATS**: Not yet implemented. Planned as `transport/nats/` module.
- **Redis**: Not yet implemented. Planned as `transport/redis/` module.
- **HTTP**: **Implemented.** SSE event delivery moved from `middleware/` to `transport/http/`. Generic HTTP utilities (healthcheck, metrics, pprof) were deleted — they had no CQRS dependencies and zero consumers.

## References

- `watermill/` — existing message broker adapter (Watermill protocol)
- ADR-0003 (multi-module monorepo)
