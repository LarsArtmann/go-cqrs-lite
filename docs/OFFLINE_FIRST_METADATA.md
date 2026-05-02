# Offline-First Metadata Convention

> Standard metadata keys for offline-first event attribution and timing.

## Convention-Based Keys

These keys use `event.WithCustom(key, value)` or the convenience wrappers
`event.WithClientID()` and `event.WithClientOccurredAt()`. They are stored
in `event.Metadata.Custom`.

| Key                  | Type                 | Set By        | When                | Purpose                               |
| -------------------- | -------------------- | ------------- | ------------------- | ------------------------------------- |
| `client.id`          | string (ULID)        | Client        | Event creation      | Which device created this event       |
| `client.occurred_at` | string (RFC3339Nano) | Client        | Event creation      | When the event happened on the device |
| `client.timezone`    | string (IANA tz)     | Client        | Event creation      | Device timezone for business grouping |
| `sync.pushed_at`     | string (RFC3339Nano) | Client        | Push attempt        | When push was attempted               |
| `sync.acked_at`      | string (RFC3339Nano) | Server        | Push acknowledgment | When server confirmed receipt         |
| `sync.rebased_at`    | string (RFC3339Nano) | Server/Client | Rebase              | When events were reordered            |

## Usage

```go
// Using convenience wrappers
evt, err := event.NewEvent(
    "order.placed", aggregateID, "Order", 1, payload,
    event.WithClientID(clientID),
    event.WithClientOccurredAt(time.Now()),
)

// Using custom keys directly
evt, err := event.NewEvent(
    "order.placed", aggregateID, "Order", 1, payload,
    event.WithCustom("client.timezone", "Europe/Berlin"),
    event.WithCustom("sync.pushed_at", time.Now().Format(time.RFC3339Nano)),
)
```

## Why Convention, Not Fields?

The `event.Metadata` struct has typed fields for the most common CQRS metadata
(CorrelationID, CausationID, UserID, RequestID). Offline-first metadata is
optional and consumer-specific — adding fields for every possible key would
bloat the struct and break serialization compatibility.

Convention-based keys let consumers add what they need without modifying
go-cqrs-lite, while still being queryable and well-documented.
