# transport/http — Server-Sent Events (SSE) Event Delivery

[![Go Reference](https://pkg.go.dev/badge/github.com/larsartmann/go-cqrs-lite/transport/http/v4.svg)](https://pkg.go.dev/github.com/larsartmann/go-cqrs-lite/transport/http/v4)

Bridges `event.Bus` to HTTP clients via Server-Sent Events (SSE) for real-time, unidirectional event streaming to browsers and API clients.

```bash
go get github.com/larsartmann/go-cqrs-lite/transport/http/v4
```

## Quick Start

### Basic SSE

```go
import cqrshttp "github.com/larsartmann/go-cqrs-lite/transport/http/v4"

broker, err := cqrshttp.NewSSEBroker(bus)
if err != nil { log.Fatal(err) }

mux := http.NewServeMux()
mux.Handle("/events", cqrshttp.SSEHandler(broker))
```

### With Reconnection Support (Last-Event-ID)

```go
broker, err := cqrshttp.NewSSEBroker(bus,
    cqrshttp.WithReconnectJournal(journalStore, cqrshttp.DefaultSSEReplayLimit),
)
// Clients sending "Last-Event-ID" header get missed events replayed
// from the journal before live streaming begins.
```

### Unlimited Replay with Byte Budget

```go
broker, err := cqrshttp.NewSSEBroker(bus,
    cqrshttp.WithReconnectJournal(journalStore, 0), // 0 = unlimited streaming
    cqrshttp.WithReplayByteBudget(8*1024*1024),      // cap at 8 MB
    cqrshttp.WithReplayTimeout(30*time.Second),      // stop replay after 30s
)
```

### CBOR-to-JSON Payload Transcoding

For the common case (CBOR store, JSON browsers), use the ready-made adapter —
it handles decoding, re-encoding, and graceful fallback in one line:

```go
broker, err := cqrshttp.NewSSEBroker(bus,
    cqrshttp.WithPayloadTransform(cqrshttp.CBORToJSONTransform),
)
```

For schema-free transcoding with explicit error handling, wrap
`codec.TranscodeToJSON` directly. For schema-aware JSON (reconstructing field
names from `toarray` structs), use `event.DecodePayloadAuto[T]` in a custom
transform.

### REST Backfill Endpoint

```go
mux.Handle("/events/backfill", cqrshttp.BackfillHandler(broker))
// GET /events/backfill?after=<event-id>&limit=500 -> JSON array of events
```

## API

### SSEBroker

| Symbol                       | Description                                        |
| ---------------------------- | -------------------------------------------------- |
| `NewSSEBroker(bus, opts...)` | Creates a broker subscribed to `bus.SubscribeAll`. |
| `SSEHandler(broker)`         | HTTP handler for SSE streaming.                    |
| `BackfillHandler(broker)`    | HTTP handler for REST-based event backfill.        |
| `SSEClientID`                | Per-client identifier.                             |

### Options

| Option                        | Default | Description                                                   |
| ----------------------------- | ------- | ------------------------------------------------------------- |
| `WithReconnectJournal(j, n)`  | —       | Enables Last-Event-ID replay. `n>0` bounded; `<=0` unlimited. |
| `WithReplayByteBudget(bytes)` | 8 MB    | Caps unlimited replay by total payload bytes. `-1` disables.  |
| `WithReplayTimeout(d)`        | 0 (off) | Max replay duration before switching to live.                 |
| `WithDedupRingCapacity(n)`    | 1024    | Replay-to-live dedup ring size (~90 KB).                      |
| `WithRetryInterval(d)`        | 5s      | SSE `retry:` field sent to browsers.                          |
| `WithEventFilter(fn)`         | —       | Broker-level event-type predicate; `false` = dropped.         |
| `WithPayloadTransform(fn)`    | —       | Transform payload bytes before wire write (CBOR-to-JSON).     |
| `WithReplayMetrics(m)`        | —       | OTel instruments for replay duration/count/incomplete.        |

### Constants

| Constant                   | Value                      | Description                            |
| -------------------------- | -------------------------- | -------------------------------------- |
| `DefaultSSEReplayLimit`    | 1000                       | Suggested bounded replay cap.          |
| `DefaultSSERetryInterval`  | 5s                         | Default SSE retry hint.                |
| `SSEReplayBudgetDisabled`  | -1                         | Disable byte budgeting entirely.       |
| `SSEReplayIncompleteEvent` | `"cqrs.replay.incomplete"` | Advisory event sent on replay timeout. |

## Design

- **Fanout**: Sequential non-blocking channel send under `RLock`. No worker pool — a slow client never blocks the broker or other clients. Targeted at <500 clients; for larger fanout use a message broker (NATS/Redis).
- **Backpressure**: Drop-newest (100-event buffer full = drop incoming). Preserves FIFO order; avoids drain-and-resend race.
- **No WebSocket**: SSE is sufficient for unidirectional server-to-client streaming. For bidirectional needs use [transport/grpc](../grpc/README.md).
- **No compression**: Delegated to the reverse proxy (Nginx/Cloudflare/ALB) to avoid double-compression.
- **Dedup at replay-to-live boundary**: Uses a `dedup.Ring` to prevent duplicate delivery during the handoff from journal replay to live streaming.

## Related Modules

- [**event**](../../event/README.md) — `event.Bus` is the event source
- [**dedup**](../../dedup/README.md) — Ring buffer for replay-to-live dedup
- [**transport/grpc**](../grpc/README.md) — Bidirectional transport alternative
- [**watermill**](../../watermill/README.md) — `CatchUpSubscriber` uses the same replay+live pattern
