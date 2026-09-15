# watermill/ module — repo internals map

> Verified against the repo tree and sources on **2026-09-15** (module
> `github.com/larsartmann/go-cqrs-lite/watermill/v4`, watermill core v1.5.3,
> watermill-redisstream v1.4.5). ADR-0028 created the bridge; ADR-0127 made it
> THE canonical external delivery path (transport/* deprecated, removed v5).

## 1. File map

| File | Role |
| --- | --- |
| `event_bus.go` / `event_bus_options.go` / `event_bus_internals.go` | `EventBus` — default GoChannel bus; `Subscribe(eventType, handler)` exact-type, `SubscribeAll` catch-all; `WithBackend(pub, sub, closer)` swaps the transport |
| `command_bus.go` / `command_bus_options.go` | `CommandBus` + `WithCommandBackend(...)`; command protocol carries actor attribution both directions |
| `catchup_subscriber.go` / `catchup_replay.go` | `CatchUpSubscriber` — journal replay → checkpointed live handoff |
| `protocol.go` | Event ↔ `message.Message` mapping incl. the metadata contract (§2) |
| `publisher.go` / `event_publisher.go` / `command_publisher.go` | `NewEventPublisher(wmPublisher, topic)` and command twin — app events → watermill topics |
| `subscriber.go` / `command_subscriber_adapter.go` | `SubscriberAdapter` — repo `event.Bus` as a watermill `message.Subscriber` |
| `middleware.go` | Repo wrappers: `CorrelationIDMiddleware`, `NewRetryMiddleware` + `DefaultRetryConfig` |
| `trace_context.go` | W3C tracecontext inject/extract over message metadata (`TraceContextMiddleware`, `ExtractContext`) |
| `otel.go` | Tracer plumbing via the repo `otel/` re-export module |
| `errors.go` | Sentinel errors |

## 2. Wire protocol metadata contract (`protocol.go`)

Every event crossing a broker is flattened into watermill metadata — preserve
these keys in any custom plugin marshaler (case must match):

`event_id`, `event_type`, `stream_id`, `stream_type`, `version`,
`schema_version`, `occurred_at`, `correlation_id`, `causation_id`, `user_id`,
`request_id`, `source`, `ip_address`, `user_agent`, `payload_encoding`,
`custom.*` (free-form prefix), plus command-side `causation_command_type` /
`causation_command_id` and tombstone `tombstone_status` / `tombstone_reason`.
Legacy aliases `aggregate_id` / `aggregate_type` are read for old messages.
Actor attribution travels as `actor_id` = `"kind:raw"` (`id.ActorID`
`PrefixedString()`), both directions on commands.

## 3. EventBus defaults (why replay lives in CatchUpSubscriber)

Default backend is GoChannel configured `BlockPublishUntilSubscriberAck=true` +
`Persistent=false`: blocking-until-ack gives ordered live delivery;
non-persistent avoids GoChannel's unordered Persistent-mode replay. Replay is
instead served from the durable journal. Internal comments note the
deadlock-avoidance reasoning around `BlockPublishUntilSubscriberAck`
(`event_bus_internals.go`).

## 4. CatchUpSubscriber mechanics

Phase 1 replays journal history (`ProcessingMode = ModeReplay`), phase 2 hands
off to the live subscription with EventID-based dedup at the seam; the
checkpoint store saves after every forwarded event (at-least-once on crash —
downstream must be idempotent, which projections are by design).
`ProcessingModeMiddleware()` reconstructs the replay/live flag on the router
side. **Ordering rule: consume the output channel from ONE goroutine** — never
fan these into watermill's Router, which processes messages in parallel.

## 5. Test infrastructure (verified commands)

- `TestRedisStreamRoundtrip` — full publish→broker→typed-subscribe for events
  AND commands against a real Redis:
  `bash scripts/ephemeral-redis.sh sh -c 'cd watermill && go test -tags "goexperiment.jsonv2" -run TestRedis -v .'`
- `nix run .#integration-redis` — broker suite: roundtrip, Nack redelivery,
  consumer-group exactly-once semantics, 2 MiB payloads.
- `scripts/ephemeral-nats.sh` — ephemeral NATS; ready to host a JetStream
  roundtrip leg once `watermill-nats/v2` is wired (README corrected 2026-09-15;
  the old "no maintained plugin" claim was wrong).

## 6. Wiring an example backend (repo API + plugin shape)

```go
import (
	"github.com/larsartmann/go-cqrs-lite/watermill/v4"
)

// Plugin side (external module, e.g. watermill-redisstream): construct
// Publisher/Subscriber with the broker client, then:
bus := watermill.NewEventBus(watermill.WithBackend(pub, sub, client))
defer bus.Close()
```

The closer passed to `WithBackend` (usually the broker client) is closed with
the bus — do not double-close it in plugin teardown.

## 7. Decision boundaries (from the root skill + ADR-0127)

| Need | Use |
| --- | --- |
| Events to server-side workers/projections across processes | `watermill.CatchUpSubscriber` + `projectionhost.Host` |
| Raw events to browsers | `go-sse` (or deprecated `transport/http.SSEBroker` until v5) |
| Materialized read-model state to browsers | `metaengine.ServeSSE[V]` |
| Command distribution across processes | `watermill.NewCommandBus(watermill.WithCommandBackend(...))` |
| Queries | stay in-process (`query.Dispatcher`); watermill's requestreply is RPC-over-messages, not a query bus |
