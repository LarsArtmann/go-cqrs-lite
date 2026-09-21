---
name: watermill
description: Watermill powers, tradeoffs, and limits for go-cqrs-lite work. Use whenever touching the watermill/ module or ANY broker/pub-sub/delivery question — wiring Redis Streams / NATS JetStream / Kafka / RabbitMQ via WithBackend, EventBus/CommandBus, CatchUpSubscriber, at-least-once vs exactly-once semantics, ordering guarantees, retry/redelivery/DLQ, the Forwarder (transactional outbox), or choosing between watermill, go-sse, and metaengine.ServeSSE. Triggers on "watermill", "message.Publisher", "Router", "consumer group", "Ack/Nack", "redelivery", "outbox", "broker", "JetStream", "Redis Streams", "commit events atomically with my database writes", "atomic events + MySQL/Postgres", "browser live updates", "push updates to the UI". Read BEFORE writing or reviewing any code that publishes or subscribes across process boundaries.
user-invocable: true
metadata:
  tags: watermill, pubsub, broker, messaging, at-least-once, outbox, nats, kafka, redis-streams, rabbitmq
---

# watermill — powers, tradeoffs, limits (go-cqrs-lite view)

Watermill (ThreeDotsLabs, MIT) is the message-transport layer under this repo's
external delivery path. In this repo it is a **transport only** — the event
store, versioning, and projections stay in go-cqrs-lite (ADR-0127: no in-repo
transports; `watermill/` + `go-sse` are the sanctioned delivery paths).

Pinned here: watermill **v1.5.3** + watermill-redisstream **v1.4.5**
(`watermill/go.mod`). Upstream is actively maintained (v1.5.3 released
2026-08-25). Facts below verified against watermill.io docs on **2026-09-15**.

**Read [`references/backends.md`](references/backends.md)** for the backend
selection matrix (consumer groups / exactly-once / ordering / persistence per
backend + per-backend gotchas and plugin versions).
**Read [`references/internals.md`](references/internals.md)** for this repo's
`watermill/` module map, the message-metadata protocol contract, CatchUpSubscriber
mechanics, and test infrastructure.

## 1. The contract you are programming against

- **At-least-once delivery, always.** Handlers must be idempotent or dedup.
  Watermill explicitly does NOT ship a universal dedup middleware — the built-in
  `middleware.Deduplicator` is process-local (in-memory map), not distributed.
  In this repo the durable answers are `middleware.{Command,Event,Query}Idempotency`
  and `projectionhost` checkpoint/dedup — not watermill's.
- **`message.Message`**: `UUID` (debug-only, can be empty), `Metadata`
  (HTTP-header-like, marshaled to the broker), `Payload []byte`, and a
  per-message `context.Context`. `Ack()`/`Nack()` are non-blocking, idempotent,
  mutually exclusive (first wins). **Ack/Nack cancels the message's context** —
  never stash that context for post-ack work.
- **Router**: `HandlerFunc` = `func(*Message) ([]*Message, error)` — nil error
  auto-Acks, error auto-Nacks. **Router runs handlers in parallel** (per
  partition / per unacked message). CloseTimeout (default 30s) bounds graceful
  shutdown; `Close()` closes publishers/subscribers and waits for handlers.
- **Publishing multiple messages in one call is NOT atomic** for most backends —
  a mid-batch failure leaves partial publishes. Prefer one event per Publish
  (our `EventPublisher` does exactly that).
- **Subscriber implementations must commit broker offsets only AFTER the app
  Acks** — all official plugins honor this; custom plugins must too.

## 2. Powers — what to reach for

1. **Backend swapping with zero app-code change**: any `message.Publisher` +
   `message.Subscriber` + closer plugs into `watermill.WithBackend` /
   `watermill.WithCommandBackend`. Redis Streams (verified here by
   `TestRedisStreamRoundtrip`), NATS JetStream, Kafka, RabbitMQ, SQL, GCP, AWS…
2. **Forwarder = transactional outbox**: publish INTO your DB transaction
   (SQL/Firestore/Bolt tx publisher wraps the message in an envelope), then a
   background Forwarder relays to the real broker. This is the dual-write
   remedy for consumers whose events must commit with their data.
   (In-repo, `CatchUpSubscriber` plays the same role: the journal IS the outbox.)
3. **SQL pub/sub gives exactly-once + transactional publish** — the only
   mainstream backend here that does (see matrix in `references/backends.md`).
4. **Their CQRS component exists** (`components/cqrs`: EventBus, CommandBus,
   EventProcessor, EventGroupProcessor, generic handlers, requestreply RPC) —
   useful vocabulary when reading watermill docs, but in THIS repo it competes
   with our own decider/journal/projection stack. See §5.
5. **Middleware catalog** (router- or handler-scoped): `Recoverer`,
   `Retry` (backoff/v5 exponential, `ShouldRetry`, `OnRetriesExhausted`),
   `CircuitBreaker` (gobreaker), `Throttle`, `Timeout`, `CorrelationID`,
   `Duplicator`, `InstantAck`, `DelayOnError`, `Poison`, `RandomFail/Panic`
   (testing). Ours wraps the common ones (`watermill.CorrelationIDMiddleware`,
   `watermill.NewRetryMiddleware`, `watermill.TraceContextMiddleware`,
   `watermill.ProcessingModeMiddleware`).
6. **Router context accessors** for observability:
   `HandlerNameFromCtx`, `PublisherNameFromCtx`, `SubscriberNameFromCtx`,
   `SubscribeTopicFromCtx`, `PublishTopicFromCtx`.

## 3. Limits & sharp edges (verified)

| # | Limit | Consequence here |
| --- | --- | --- |
| 1 | At-least-once only (except SQL; NATS JetStream can do exactly-once with `TrackMsgID` + sync acks) | Handlers/projections must be idempotent; use repo idempotency middleware |
| 2 | No atomic multi-message publish | Publish one event per call; batch events via the journal, not the bus |
| 3 | **Router processes messages in parallel** — ordering is per partition/queue/stream, never global | NEVER route ordered projections through a Router; consume `CatchUpSubscriber`'s channel from ONE goroutine (README Ordering section) |
| 4 | `Retry` exhaustion → Nack → broker redelivery (later, not never) — no built-in durable DLQ | Durable poison handling is `projectionhost.WithDeadLetterStore`, not watermill's `Poison` middleware |
| 5 | Retried handlers can see a canceled context (their issue #467) → `Retry.ResetContextOnRetry` exists; default off | If handlers honor ctx deadlines, set it or expect spurious `context.Canceled` |
| 6 | GoChannel: no persistence, **no global state** (same instance must pub+sub), Persistent-mode replay breaks ordering | Our EventBus default: `BlockPublishUntilSubscriberAck=true`, `Persistent=false`; replay is CatchUpSubscriber's job |
| 7 | Redis Streams: no exactly-once, no ordering guarantee; pending-claim semantics (`MaxIdleTime`, `ClaimInterval`, `ShouldClaimPendingMessage`); go-redis indefinite initial block leaks a reader goroutine → `DisableIndefiniteInitialBlock` | Configure `Maxlens` trimming; set the flag on long-lived subscribers |
| 8 | Kafka: exactly-once impossible (no Go-client txn support); ordering only per partition key; empty/missing partition key funnels everything to one partition | Use the partitioning marshaler; derive keys from stream/entity ID |
| 9 | NATS JetStream: order NOT guaranteed under redelivery; consumer groups need `QueueGroupPrefix`/`DurablePrefix`; Core NATS mode acks are no-ops | Prefer durable + queue-group; treat Core NATS as best-effort |
| 10 | SQL/PostgreSQL: offsets rely on tx-id bookkeeping (issue #311); long-running transactions delay delivery; `AckDeadline=0` can block reads (snapshot xmin) | Keep business txs short or switch to the Queue schema (no consumer groups, but `WHERE` filtering + `DeleteOnAck`) |
| 11 | AMQP topic→exchange/routing/queue mapping is config-dependent (nomenclature mismatch) | Debug-log the generated topology; use `NewDurablePubSubConfig`/`NewDurableQueueConfig` |
| 12 | Close semantics differ per backend (async publishers flush on Close) | Always `defer` closes; our buses own the closer passed to `WithBackend` |

## 4. Backend selection (summary — full matrix + citations in `references/backends.md`)

| Backend | Consumer groups | Exactly-once | Ordering | Persistent | First choice when |
| --- | --- | --- | --- | --- | --- |
| GoChannel (in-proc) | no | yes (in-proc) | yes (non-persistent) | no | default EventBus/CommandBus, tests |
| Redis Streams | yes | no | no | yes | verified path here; lightweight broker |
| NATS JetStream (v2 plugin) | yes | yes (`TrackMsgID`) | no (redelivery) | yes | edge/geo, subject wildcards |
| Kafka (v3 plugin) | yes | no | yes (partition key) | yes | existing Kafka estates, ordered streams |
| RabbitMQ/AMQP (v3 plugin) | pseudo (queue suffix) | no | yes (per queue) | yes (durable configs) | legacy routing topologies |
| SQL PG/MySQL (v4 plugin) | yes (not queue schema) | yes | yes | yes | transactional outbox / no broker allowed |

Not deep-dived here: SQLite, Bolt, Firestore, GCP Pub/Sub, AWS SNS/SQS, HTTP,
io — see `references/backends.md` §"not verified here".

## 5. Their CQRS vs this repo (do not double-stack)

Watermill's `components/cqrs` (EventBus/CommandBus/Processors over struct
marshalers) and go-cqrs-lite solve the same problem at different depths:

| Concern | watermill `components/cqrs` | go-cqrs-lite |
| --- | --- | --- |
| Events | structs + JSON/protobuf marshaler | typed `event.Event`, branded IDs, versions, causality, tombstones |
| Event sourcing | none (transport-level) | decider + Store + Journal + snapshots |
| Ordered handling | `EventGroupProcessor` (shared subscriber) | `CatchUpSubscriber` + single-goroutine consumption + `projectionhost` |
| Retry/DLQ | middleware (process-local) | `middleware` + `projectionhost` durable DLQ |
| Sagas | none (examples only) | `deriver` |

Rule: **inside this repo, watermill is the pipe under OUR buses** — never build
app logic on `components/cqrs` when the repo stack exists. `EventGroupProcessor`'s
shared-subscriber ordering trick is the one pattern worth stealing conceptually
(one subscription per ordered group, not per handler).

## 6. Canonical wiring (repo API, verified)

```go
import (
	"github.com/larsartmann/go-cqrs-lite/watermill/v4"
)

// Single-process default: GoChannel-backed, ordered live delivery.
bus := watermill.NewEventBus()
defer bus.Close()

// Multi-process: construct ANY watermill-compatible plugin's publisher +
// subscriber (redisstream, nats v2 JetStream, kafka, amqp, ...) and inject:
eventBus := watermill.NewEventBus(watermill.WithBackend(pub, sub, client))
cmdBus := watermill.NewCommandBus(watermill.WithCommandBackend(pub, sub, client))
```

```go
import (
	"github.com/larsartmann/go-cqrs-lite/watermill/v4"
)

// Durable replay + live handoff for projections and SSE feeds:
// journal (replay, ProcessingMode=ModeReplay) -> checkpointed live sub,
// EventID-deduped at the seam. Consume from ONE goroutine for ordering.
catchUp, err := watermill.NewCatchUpSubscriber(journal, liveSub, cpStore, logger)
```

Broker-bridge and SSE decision matrix, plus when NOT to use watermill at all
(browser push → `go-sse` / `metaengine.ServeSSE`; queries stay in-process):
see the root [`SKILL.md`](../../../SKILL.md) routing tables and
[`references/internals.md`](references/internals.md).
