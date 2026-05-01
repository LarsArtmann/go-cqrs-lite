# Metadata & Timing Delay Analysis

> Comprehensive audit of what metadata each CQRS message type carries and how timing delays are handled.

**Date:** 2026-05-01

---

## 1. Metadata Inventory

### 1.1 Event — richest metadata

Events carry both core fields and a dedicated `Metadata` struct.

**Core fields** (`event.Core` — `core/event/event.go:64`):

| Field          | Type              | Source                                   | Notes                                      |
| -------------- | ----------------- | ---------------------------------------- | ------------------------------------------ |
| `ID`           | `id.EventID`      | Auto-generated via `id.NewEventID()`     | ULID-backed, encodes timestamp, sortable   |
| `Type`         | `Type`            | Required param                           | e.g. `"user.created"`                      |
| `AggregateID`  | `id.AggregateID`  | Required param                           |                                            |
| `AggregateType`| `AggregateType`   | Required param                           | e.g. `"User"`                              |
| `Version`      | `Version` (≥0)    | Required param, validated via `ParseVersion` | Stream position within aggregate       |
| `SchemaVersion`| `int`             | Defaults to 1, `WithSchemaVersion`       | Drives upcasting chain                     |
| `Payload`      | `[]byte`          | Required param                           | Defensive copy on read                     |
| `OccurredAt`   | `time.Time`       | `time.Now()` at creation, `WithOccurredAt` | When the event *happened*              |

**Metadata struct** (`event.Metadata` — `core/event/event.go:52`):

| Field           | Type                    | Option              | Notes                         |
| --------------- | ----------------------- | ------------------- | ----------------------------- |
| `CorrelationID` | `id.CorrelationID`      | `WithCorrelationID` | Distributed tracing           |
| `CausationID`   | `id.CausationID`        | `WithCausationID`   | What triggered this event     |
| `UserID`        | `id.UserID`            | `WithUserID`        | Who triggered the event       |
| `RequestID`     | `id.RequestID`          | `WithRequestID`     | Per-request debugging        |
| `Source`        | `Source`                | `WithSource`        | e.g. "api", "scheduler", "cli" |
| `IPAddress`     | `IPAddress`             | `WithIPAddress`     | Validated v4/v6 via `netip`  |
| `UserAgent`     | `UserAgent`             | `WithUserAgent`     | HTTP client identification    |
| `Custom`        | `map[MetadataKey]string`| `WithCustom(key,val)`| Extensibility escape hatch  |

**Additional APIs:**

- **Builder** (`core/event/builder.go`) — Fluent API: `NewBuilder().WithPayload().WithCorrelationID().Build()`
- **ContextEnricher** (`core/event/enricher.go`) — `func(ctx context.Context) []Option`; extracts options from context. `CompositeEnricher` chains multiple. Zero internal consumers — library API for consumers to wire.
- **EnrichEvent** — Applies a `ContextEnricher` to an existing `*Core` event.

### 1.2 Command — minimal

| Field         | Type             | Notes                  |
| ------------- | ---------------- | ---------------------- |
| `Type`        | `Type` (string)  | e.g. `"create_user"`   |
| `AggregateID` | `id.AggregateID` | Target aggregate       |

No metadata struct. No correlation/causation/user/request IDs. No timestamp.

### 1.3 Query — minimal

| Field  | Type            | Notes               |
| ------ | --------------- | ------------------- |
| `Type` | `Type` (string) | e.g. `"get_user"`   |

No metadata. No aggregate ID (queries are read-side, not aggregate-scoped). No timestamp.

### 1.4 Aggregate Root

| Field   | Type             | Notes                          |
| ------- | ---------------- | ------------------------------ |
| `ID`    | `id.AggregateID` | Identity                       |
| `Type`  | `AggregateType`  | e.g. `"User"`                  |
| `Version` | `Version`      | Incremented on `RecordEvent`  |

No metadata. Aggregate roots are state machines, not messages.

---

## 2. What Middleware Adds

Middleware enriches observability but does **not** attach metadata to the messages themselves:

| Middleware    | What it adds                                         | Where it lives             |
| ------------- | --------------------------------------------------- | -------------------------- |
| **Logging**   | Logs type, aggregateID, duration (start/end/error)  | `middleware/logging.go`    |
| **Metrics**   | Records duration + success/error with type label     | `middleware/metrics.go`    |
| **Tracing**   | OpenTelemetry spans with `cqrs.command.type` attrs  | `middleware/tracing.go`    |
| **Retry**     | Exponential backoff + jitter, context-aware cancel  | `middleware/retry.go`      |
| **Recovery**  | Panic → error with stack trace                      | `middleware/recovery.go`  |
| **Validation**| Pre-handler validation predicate                    | `middleware/validation.go` |

Important: Tracing propagates via `context.Context`, not via message metadata. The `ContextEnricher` bridge exists but is not wired internally.

---

## 3. Timing Delay Mechanisms

### 3.1 What exists today

| Mechanism                    | Location                          | How it works                                                                                             |
| ---------------------------- | --------------------------------- | -------------------------------------------------------------------------------------------------------- |
| **Transactional Outbox**     | `core/event/outbox.go:30`         | `Outbox` interface: `Append` (in-tx), `PollPending` (oldest first), `Ack`. Prevents write-then-publish gap. |
| **OutboxPublisher**          | `core/event/outbox_publisher.go`  | Background goroutine polls outbox (default 1s), publishes to bus, acks on success. `PublishNow` for sync. |
| **Retry with backoff**       | `middleware/retry.go:57`          | Exponential backoff with crypto/rand jitter. Configurable `MaxAttempts`, `InitialDelay`, `MaxDelay`, `Multiplier`. |
| **ULID EventIDs**            | `core/pkg/id/id.go`               | Encode timestamp at creation. Even if publishing is delayed, `ID` and `OccurredAt` reflect when event *happened*. |
| **`WithOccurredAt`**        | `core/event/options.go:34`        | Override timestamp when reconstructing from storage. Preserves original timing.                          |
| **`ContextEnricher`**        | `core/event/enricher.go`          | Extracts `Option`s from `context.Context`. Consumer-facing API for injecting correlation/trace IDs.     |

### 3.2 The outbox flow (timing diagram)

```
Command Handler
      │
      ▼
Aggregate.RecordEvent()     ← event.OccurredAt = time.Now()
      │
      ▼
Repository.Save()
      │
      ├─► Store.Save(events, expectedVersion)   ← durable write
      │
      ├─► [outbox configured?]
      │     YES → Outbox.Append(events)         ← same TX
      │     NO  → Bus.Publish(events...)         ← immediate, risk of loss
      │
      ▼
      ... (time passes: up to poll interval) ...
      │
      ▼
OutboxPublisher.publishPending()
      │
      ├─► Outbox.PollPending(limit)
      ├─► Bus.Publish(entry.Events...)
      └─► Outbox.Ack(entry.IDs)
```

### 3.3 Retry flow (timing diagram)

```
Handler call
      │
      ▼
  err != nil && IsRetryable(err)?
      │
      YES → delay = backoff(config, attempt)
      │        = InitialDelay × Multiplier^(attempt-1)
      │        + jitter(rand(0, delay/2))
      │        capped at MaxDelay
      │
      ▼
  time.Sleep(delay) — or ctx.Done() cancellation
      │
      ▼
  retry (up to MaxAttempts)
```

---

## 4. Gaps & Risks

### 4.1 Critical gaps

| Gap                                           | Severity | Risk                                                                                                                                                  |
| --------------------------------------------- | -------- | ----------------------------------------------------------------------------------------------------------------------------------------------------- |
| **No `PublishedAt` / `DeliveredAt` timestamp** | HIGH    | There's `OccurredAt` but no record of *when the event was actually published/delivered*. Consumers can't measure outbox lag or detect stuck entries. |
| **No idempotency key on Command**             | MEDIUM  | Commands carry no deduplication metadata. If a caller retries a command (beyond middleware retry), it could execute twice. `RequestID` exists on events but not commands. |
| **No clock skew handling**                    | MEDIUM  | `time.Now()` in `NewEvent` uses the process clock. In distributed deployments, different nodes may have skewed clocks. ULID mitigates within-process (monotonic) but doesn't solve cross-node skew. |

### 4.2 Observability gaps

| Gap                                           | Severity | Risk                                                                                                                                                  |
| --------------------------------------------- | -------- | ----------------------------------------------------------------------------------------------------------------------------------------------------- |
| **No outbox age / staleness tracking**        | MEDIUM  | `PollPending` returns entries but no metadata about how long they've been pending. No alerting on stuck entries.                                       |
| **No `ProcessedAt` on projection checkpoints**| LOW     | `CheckpointStore` stores last `EventID` only, not *when* it was processed. Can't measure projection lag.                                              |
| **`OutboxPublisher` silently swallows poll errors** | LOW | `outbox_publisher.go:137` — `PollPending` errors are silently ignored (just `return`). No logging, no metrics, no retry beyond the next tick.       |

### 4.3 Ordering & concurrency gaps

| Gap                                           | Severity | Risk                                                                                                                                                  |
| --------------------------------------------- | -------- | ----------------------------------------------------------------------------------------------------------------------------------------------------- |
| **No event ordering guarantee across aggregates** | MEDIUM | `OutboxPublisher.publishPending` processes entries sequentially within a poll cycle, but between poll cycles or across outbox instances, no global ordering. Only per-aggregate ordering via `Version`. |
| **No backpressure / flow control**             | LOW     | `publishPending` processes all entries in one batch with no concurrency limit. Under load, a single `Publish` call could block for a long time.      |

### 4.4 Structural asymmetry

| Observation | Detail                                                                                           |
| ----------- | ------------------------------------------------------------------------------------------------ |
| **Command metadata gap** | Commands have `Type` + `AggregateID` only. No `CorrelationID`, `CausationID`, `UserID`, `RequestID`, `Source`, timestamp, or `Custom` metadata. Events are rich; commands are bare. |
| **Query metadata gap**   | Queries have `Type` only. No tracing metadata, no request context.                                                      |
| **ContextEnricher is unwired** | Exists as a library API but no internal code uses it. Consumers must wire it themselves. The bridge between HTTP middleware (extracting trace IDs from headers) and event metadata is not provided. |

---

## 5. Recommendations

### 5.1 Short-term (low effort, high impact)

1. **Add `Command.Metadata`** — Mirror the `event.Metadata` struct (or a subset: `CorrelationID`, `CausationID`, `UserID`, `RequestID`). This enables distributed tracing through the full command → event lifecycle.
2. **Add `PublishedAt` to `OutboxEntry`** — Record when each outbox entry was successfully published. Enables outbox lag measurement.
3. **Log `OutboxPublisher` poll errors** — Change silent `return` to at minimum a log entry. Ideally, increment a metric counter.

### 5.2 Medium-term

4. **Add `CreatedAt` to Command** — When the command was received. Enables measuring command-to-event latency.
5. **Add `ProcessedAt` to `CheckpointStore`** — Store `(EventID, time.Time)` instead of just `EventID`. Enables projection lag dashboards.
6. **Wire `ContextEnricher` into `EventSourcedRepository.Save`** — When the repository creates/publishes events, automatically enrich from the request context. Bridge the gap between HTTP middleware and event metadata.
7. **Add `OutboxEntry.CreatedAt`** — Timestamp when the entry was appended. Enables staleness detection and alerting on stuck entries.

### 5.3 Long-term

8. **Clock skew mitigation** — Consider HLC (Hybrid Logical Clock) or vector clock for multi-node deployments. The ULID approach works for single-writer-per-aggregate but breaks down with cross-aggregate ordering requirements.
9. **Command idempotency** — Add `idempotency_key` to Command metadata. Store deduplication table in the same TX as command handling.
10. **Outbox backpressure** — Rate-limit or batch-publish with configurable concurrency. Prevent thundering herd on bus recovery after outage.

---

## 6. Summary Table

| Concern                  | Event | Command | Query | Aggregate |
| ------------------------ | ----- | ------- | ----- | --------- |
| CorrelationID            | ✅    | ❌      | ❌    | ❌        |
| CausationID              | ✅    | ❌      | ❌    | ❌        |
| UserID                   | ✅    | ❌      | ❌    | ❌        |
| RequestID                | ✅    | ❌      | ❌    | ❌        |
| Source                   | ✅    | ❌      | ❌    | ❌        |
| IPAddress                | ✅    | ❌      | ❌    | ❌        |
| UserAgent                | ✅    | ❌      | ❌    | ❌        |
| Custom metadata          | ✅    | ❌      | ❌    | ❌        |
| Timestamp (OccurredAt)   | ✅    | ❌      | ❌    | ❌        |
| Timestamp (PublishedAt)  | ❌    | N/A     | N/A   | N/A       |
| Timestamp (ProcessedAt)  | ❌    | N/A     | N/A   | ❌ (ckpt) |
| Idempotency key          | ❌    | ❌      | ❌    | N/A       |
| Clock skew handling      | ❌    | ❌      | ❌    | ❌        |
| Outbox lag measurement   | ❌    | N/A     | N/A   | N/A       |
| Retry with backoff        | ✅ (mw)| ✅ (mw) | ✅ (mw)| N/A       |
| Transactional outbox      | ✅    | N/A     | N/A   | ✅ (repo) |
