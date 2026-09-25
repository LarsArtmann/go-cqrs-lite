# Watermill backends — matrix, gotchas, plugin map

> Verified against watermill.io pub/sub pages on **2026-09-15**. Core repo:
> ThreeDotsLabs/watermill (v1.5.3, 2026-08-25, active — not archived, 9.9k stars).
> Plugin majors are independent of core: check each repo's releases before bumping.
>
> **Plugin latests re-verified against proxy.golang.org on 2026-09-25** (all
> cited majors/versions current): watermill v1.5.3 · redisstream v1.4.5 ·
> nats/v2 v2.2.0 (2026-05-15) · kafka/v3 v3.1.4 (2026-07-29) · amqp/v3
> v3.1.0 (2026-05-14) · sql/v4 v4.1.5 (2026-05-14).

## 1. Characteristics matrix (from each plugin's official docs page)

| Backend (module) | Consumer groups | Exactly-once | Guaranteed order | Persistent | Notable config |
| --- | --- | --- | --- | --- | --- |
| **GoChannel** (core `pubsub/gochannel`) | no | yes (in-process) | yes — **non-persistent mode only** | no | `BlockPublishUntilSubscriberAck`, `Persistent` |
| **NATS JetStream** (`watermill-nats/v2`, v2.2.0) | yes — `QueueGroupPrefix` / `DurableCalculator` | **yes** — `TrackMsgID: true` + `AckAsync` false (default) | **no** — redelivery breaks order | yes | `AutoProvision`, `NakDelay`, durable prefix |
| **Kafka** (`watermill-kafka/v3`, Sarama) | yes | **no** — Kafka txns unsupported by any Go client | yes — per partition key | yes | `NackResendSleep` (100ms default), `OverwriteSaramaConfig`, `Tracer` |
| **RabbitMQ/AMQP** (`watermill-amqp/v3`) | pseudo — queue-name suffix + fanout exchange | no | yes — per-queue semantics | yes — `NewDurablePubSubConfig`/`NewDurableQueueConfig` | `TopologyBuilder`, `Qos.PrefetchCount` |
| **Redis Streams** (`watermill-redisstream`; repo pins v1.4.5) | yes | no | no | yes | `ConsumerGroup` vs fan-out XREAD, `Maxlens` trimming |
| **SQL PG/MySQL** (`watermill-sql/v4`) | yes — **not** in queue schema | **yes** | yes | yes | `AckDeadline`, `PollInterval`, tx publishers |
| SQLite / Bolt / Firestore / GCP Pub/Sub / AWS SNS-SQS / HTTP / io | — | — | — | — | **not verified here** — see each docs page before relying on any cell |

## 2. Per-backend gotchas (all from official docs, 2026-09-15)

### GoChannel
- No global state: **the same instance must be used for Publish and Subscribe.**
- Publish is background (non-blocking) unless `BlockPublishUntilSubscriberAck`.
- `Persistent=true` (replay to late subscribers) **sacrifices ordering** — that
  is why this repo's EventBus uses `Persistent=false` and lets
  `watermill.NewCatchUpSubscriber` own replay from the journal.

### NATS JetStream (`pkg/jetstream` stable since plugin v2.1.0, 2024-08)
- `pkg/nats` with `JetStream` enabled is the recommended production surface;
  limited **Core NATS** support also exists (`Disabled=true`) where most acks
  are no-ops — treat as best-effort.
- Durable subscriptions: set `DurablePrefix` (or a `DurableCalculator`) and
  `QueueGroupPrefix`; the server then tracks last-acked per client+durable.
- Exactly-once recipe (official): `TrackMsgID=true` (msg UUID → NATS MsgId
  dedup) and keep `AckAsync=false`.
- Marshaler: `NATSMarshaler` carries watermill metadata in NATS headers;
  reserved header `_watermill_message_uuid`.

### Kafka
- Exactly-once: NOT achievable — official docs: Kafka transactions have no Go
  client support. Plan for at-least-once + consumer dedup.
- Ordering requires partitioning: `NewWithPartitioningMarshaler(func(topic, msg)…)`
  — read the key from metadata to avoid unmarshaling the payload.
  **Empty key ("") funnels all such messages into ONE partition** (hotspot).
- `Tracer` field supersedes the deprecated `OTELEnabled` bool (otelsarama).
- Subscriber spawns one goroutine per partition — the Router then runs
  handlers concurrently; ordering holds only within a partition.

### RabbitMQ (AMQP)
- Watermill's "topic" is NOT the AMQP topic exchange — it maps to
  exchange name / routing key / queue name depending on config. When lost,
  enable debug logging to see the generated topology.
- "Consumer groups" are emulated: `GenerateQueueNameTopicNameWithSuffix` with a
  fanout exchange gives each suffix-group an independent queue (all messages,
  separately). Queue semantics (load balancing) come from
  `NewDurableQueueConfig` instead — choose deliberately.
- Persistence only with durable configs (`NewDurablePubSubConfig`,
  `NewDurableQueueConfig`) + persistent delivery mode (default marshaler).

### Redis Streams
- Delivery: consumer-group mode (`ConsumerGroup` + `Consumer`) or fan-out via
  XREAD (no group). Fan-out default starts at `$` (latest);
  `OldestId "0"` starts groups from the beginning.
- Stuck consumers: pending entries are re-claimed after `MaxIdleTime`
  (checked every `ClaimInterval`, `ClaimBatchSize` per sweep);
  `ShouldClaimPendingMessage` callback gates claims when processing times vary.
- **go-redis quirk**: the initial XRead blocks indefinitely and ignores ctx
  cancellation — reader-goroutine leak (go-redis#2556). Set
  `DisableIndefiniteInitialBlock=true` on long-lived subscribers.
- `NackResendSleep` (default: no sleep — immediate) controls redelay after Nack.
- Trim streams via publisher `Maxlens`/`DefaultMaxlen` (XADD MAXLEN).
- Default marshaler is MessagePack (`DefaultMarshallerUnmarshaller`).

### SQL (PostgreSQL, MySQL)
- Exactly-once + persistent + ordered — polling `SELECT` with per-consumer
  offsets via `OffsetsAdapter` (`DefaultPostgreSQLOffsetsAdapter` /
  `DefaultMySQLOffsetsAdapter`).
- **Transactional publish**: build the publisher on a `*sql.Tx`
  (`sql.TxFromStdSQL(tx)`) — inserts commit with your business tx. Combine with
  the **Forwarder** (outbox): wrap the tx publisher in
  `forwarder.NewPublisher(pub, forwarder.PublisherConfig{ForwarderTopic: …})`,
  run `forwarder.NewForwarder(sqlSubscriber, brokerPublisher, …)` in the
  background. This is the canonical dual-write fix.
- PostgreSQL caveat (watermill#311): `SERIAL` assigns offsets at insert time,
  not commit time; offsets also track the last tx-id. **Long-running
  transactions delay delivery** (snapshot xmin) — keep txs short or use the
  **Queue schema** (`PostgreSQLQueueSchema`): no consumer groups, but custom
  `WHERE` filtering, optional `DeleteOnAck`, `SubscribeBatchSize` (default
  100; higher = more redelivery risk on crash, 1 = safest).

## 3. Repo wiring (verified surface)

```go
import (
	"github.com/larsartmann/go-cqrs-lite/watermill/v4"
)

// Any plugin's pub/sub + a Closer (usually the broker client):
// redisstream, nats (v2 JetStream), kafka, amqp, sql, ...
eventBus := watermill.NewEventBus(watermill.WithBackend(pub, sub, client))
```

Verified broker path here: Redis Streams roundtrip
(`TestRedisStreamRoundtrip` via `scripts/ephemeral-redis.sh`);
`scripts/ephemeral-nats.sh` exists for a JetStream leg.
NATS note: this repo's README claimed "no maintained JetStream plugin" until
corrected 2026-09-15 — watermill-nats/v2 IS maintained (v2.2.0, 2026-05).

## 4. Choosing (operator cheat sheet)

- **No broker / single process** → GoChannel default (`watermill.NewEventBus()`).
- **Want a broker, minimal ops** → Redis Streams (verified here) with
  `Maxlens` + claim tuning; accept at-least-once + no global order.
- **Need exactly-once at the broker** → NATS JetStream (`TrackMsgID`) or SQL
  pub/sub; Kafka cannot in Go.
- **Need per-key ordering at scale** → Kafka with partitioning marshaler
  (key = stream/entity ID) — or sidestep: journal-first, CatchUpSubscriber
  single-goroutine consumption (ordering from the journal, not the broker).
- **Events must commit with DB writes** → SQL tx publisher + Forwarder outbox.
