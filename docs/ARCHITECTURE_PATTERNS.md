# Architecture Patterns

## Time-Travel API

The event store supports querying historical state:

| Method                                           | Description                               |
| ------------------------------------------------ | ----------------------------------------- |
| `LoadToVersion(ctx, aggType, aggID, maxVersion)` | Load events up to and including a version |
| `LoadToTimestamp(ctx, aggType, aggID, maxTime)`  | Load events up to a point in time         |
| `LoadFromVersion(ctx, aggType, aggID, version)`  | Load events starting from a version       |
| `ReadFrom(ctx, afterEventID, limit)`              | Cursor-based global event loading (SeekableJournal) |

The decider module provides convenience methods:

```go
decider.LoadAtVersion(ctx, repo, decider, aggID, version)
decider.LoadAtTime(ctx, repo, decider, aggID, time)
```

## State Is Disposable

A key insight from event sourcing: **derived state can always be rebuilt from events**. This means:

- Projections can be deleted and rebuilt from scratch
- Aggregate state is derived by folding events
- Snapshots are an optimization, not a requirement
- If in doubt, delete and replay

## Determinism Rule

Inside projections and fold functions:

- **No `time.Now()`** — use `event.OccurredAt()` from the event
- **No `uuid.New()`** — use `event.ID()` from the event
- **No random values** — fold functions must be pure
- **No external API calls** — projections should be deterministic

Breaking this rule makes replay produce different results, defeating the purpose of event sourcing.

## Versioned Event Names

Use namespace prefixes for breaking event schema changes:

```go
// v1 — original
event.Type("user.created")

// v2 — breaking schema change
event.Type("v2.user.created")
```

Upcasters bridge v1 → v2 during replay:

```go
registry.Register(upcaster.Chain{
    From: "user.created",
    To:   "v2.user.created",
    Transform: func(payload []byte) ([]byte, error) { ... },
})
```

## Soft Deletes Over Hard Deletes

Never hard-delete events. Instead:

1. Append a `UserDeleted` event
2. The fold function sets a `Deleted: true` flag on state
3. Projections filter out deleted entities in their queries
4. Hard delete is no longer available on the Store interface — use tombstone metadata (`event.MarkTombstone`) for soft-delete, or drop data at the storage layer for GDPR

## Offline-First Metadata Conventions

Events carry metadata for distributed tracing and offline sync:

| Key              | Type               | Purpose                             |
| ---------------- | ------------------ | ----------------------------------- |
| `correlation_id` | `id.CorrelationID` | Links events from the same command  |
| `causation_id`   | `id.EventID`       | Links events caused by other events |
| `client_id`      | `id.ClientID`      | Identifies the originating client   |
| `user_id`        | `id.UserID`        | The user who triggered the command  |

Use the option functions:

```go
event.WithCorrelationID(cid)
event.WithCausationID(eid)
event.WithClientID(clientID)
event.WithUserID(userID)
```
