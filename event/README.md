# event — Event Sourcing Core

[![Go Reference](https://pkg.go.dev/badge/github.com/larsartmann/go-cqrs-lite/event/v3.svg)](https://pkg.go.dev/github.com/larsartmann/go-cqrs-lite/event/v3)

Immutable events, store/bus interfaces, event sourcing primitives, and the 5-family error taxonomy for CQRS + Event Sourcing. Zero infrastructure dependencies (no HTTP, no database, no message broker).

```bash
go get github.com/larsartmann/go-cqrs-lite/event/v3
```

## Quick Start

```go
import (
    "github.com/larsartmann/go-cqrs-lite/event/v3"
    "github.com/larsartmann/go-cqrs-lite/id/v3"
)

// Create an event
aggID := id.NewAggregateID()
evt, err := event.NewEvent("user.created", aggID, "User", event.Version(1),
    UserCreated{Name: "Alice"},
    event.WithCorrelationID(correlationID),
)

// Batch creation with auto-incrementing versions
events, err := event.NewEvents(aggID, "User", 0,
    []event.Type{"user.created", "user.email.verified"},
    []any{UserCreated{Name: "Alice"}, EmailVerified{At: time.Now()}},
)

// Decode payload
payload, err := event.DecodePayload[UserCreated](evt, codec.JSONCodec{})
```

## Store & Bus Interfaces

```go
// Store = EventSink + EventSource (ISP split)
type EventSink interface {
    Save(ctx, aggRef, events, expectedVersion) error
    AppendBatch(ctx, aggRef, events) error
}

type EventSource interface {
    Load(ctx, aggRef) ([]Event, error)
    LoadFromVersion(ctx, aggRef, fromVersion) ([]Event, error)
    LoadToVersion(ctx, aggRef, maxVersion) ([]Event, error)
    LoadToTimestamp(ctx, aggRef, maxTime) ([]Event, error)
}

type Store interface { EventSink; EventSource }

// Cross-aggregate reads
type Journal interface { ReadAll(ctx) ([]Event, error) }
type SeekableJournal interface { ReadFrom(ctx, afterEventID, limit) ([]Event, error) }

// Event bus
type Bus interface {
    Publish(ctx, ...Event) error
    Subscribe(eventType, handler) error
    SubscribeAll(handler) error
    Use(middleware...)
    UsePublish(middleware...)
}
```

## Key Types

| Type              | Purpose                                                                             |
| ----------------- | ----------------------------------------------------------------------------------- |
| `Event`           | Immutable interface: Type, AggregateID, Version, Payload, Metadata                  |
| `Version`         | Strong-typed event version with Add/Sub/Cmp arithmetic                              |
| `Type`            | Event type string                                                                   |
| `AggregateType`   | Aggregate type string                                                               |
| `SchemaVersion`   | Event schema version for upcasting                                                  |
| `Metadata`        | CorrelationID, CausationID, UserID, RequestID, Source, IPAddress, UserAgent, Custom |
| `Checkpoint`      | EventID + ProcessedAt for projection checkpointing                                  |
| `TombstoneStatus` | Active / Tombstoned / Undetermined for soft-delete                                  |

## 19 Functional Options

`WithEventID`, `WithOccurredAt`, `WithMetadata`, `WithCorrelationID`, `WithCausationID`, `WithUserID`, `WithRequestID`, `WithSource`, `WithIPAddress`, `WithUserAgent`, `WithCustom`, `WithSchemaVersion`, `WithEncoding`, `WithNewCodec`, `WithClock`, `WithClientID`, `WithClientOccurredAt`, `WithDeadline`, `FromContext`

## Error Classification

```go
// Domain rejections (client error, not retryable)
return event.NewRejection("user.create.empty_email", "email is required")

// Conflicts (optimistic concurrency, duplicate)
return event.NewConflict("user.create.duplicate", "user already exists")

// Transient (retryable)
return event.NewTransient("user.create.timeout", "operation timed out")

// Infrastructure (system-level)
return event.NewInfrastructure("store.connection", "connection lost")

// Corruption (data integrity)
return event.NewCorruption("store.invalid_event", "checksum mismatch")
```

## Sub-packages

| Package           | Purpose                                                |
| ----------------- | ------------------------------------------------------ |
| `event/eventtest` | FakeStore, FakeBus, FakeSnapshotStore, test assertions |

## Dependencies

| Dependency        | Purpose                                    |
| ----------------- | ------------------------------------------ |
| `oklog/ulid/v2`   | Binary-sortable, time-ordered identifiers  |
| `go-branded-id`   | Generic branded ID type backing `id.Of[T]` |
| `go-error-family` | Error classification taxonomy (5 families) |
| `samber/ro`       | Reactive event streams (bus subscriptions) |

## Related Modules

- [**command/v2**](../command/README.md) — Command dispatch, typed handlers, middleware
- [**query/v2**](../query/README.md) — Query dispatch, typed results, pagination
- [**decider/v2**](../decider/README.md) — Pure-function aggregate pattern (recommended)
- [**id/v2**](../id/README.md) — Branded IDs: `id.Of[T]` backed by ULID
- [**codec/v2**](../codec/README.md) — Payload encoding (JSON, CBOR, Raw) for `DecodePayload`
- [**schema/v2**](../schema/README.md) — Upcasting and schema evolution
- [**snapshot/v2**](../snapshot/README.md) — Snapshot persistence
- [**signing/v2**](../signing/README.md) — Event signing (HMAC, Ed25519)
- [**encryption/v2**](../encryption/README.md) — Payload encryption (XChaCha20, AES-GCM)
- [**middleware/v2**](../middleware/README.md) — Logging, retry, tracing, metrics
- [**projection/v2**](../projection/README.md) — Read-model projections from event streams
