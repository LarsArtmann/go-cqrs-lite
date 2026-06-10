# Domain Language

A **Unified Language** for `go-cqrs-lite` — shared across library consumers, contributors, and AI.
Inspired by Domain-Driven Design (DDD) Ubiquitous Language.

Every term below should mean the **same thing** to everyone who reads it.
If a word means something different to a consumer than to an implementer, it is defined here.

---

## Core Concepts

### Event Sourcing

| Term               | Definition                                                            | Context                                                                    |
| ------------------ | --------------------------------------------------------------------- | -------------------------------------------------------------------------- |
| **Event**          | Immutable record of something that happened in the domain             | The fundamental unit of state change                                       |
| **Aggregate**      | Cluster of domain objects treated as a single unit of consistency     | Has a unique identity (AggregateRef) and an event stream                   |
| **AggregateRef**   | `{Type, ID}` — canonical identity of an aggregate instance            | Passed to Store methods as a single value                                  |
| **Stream**         | Ordered sequence of events for a single aggregate, ordered by Version | `Load()` returns the full stream; `LoadFromVersion()` returns a suffix     |
| **Version**        | Position of an event within its aggregate's stream (1-indexed)        | Used for optimistic concurrency                                            |
| **Event Store**    | Persistence layer for append-only event streams                       | Implements `event.Store` (Sink + Source + Journal)                         |
| **Snapshot**       | Point-in-time capture of aggregate state at a specific version        | Avoids replaying the entire stream on every load                           |
| **Snapshot Store** | Persistence layer for aggregate snapshots                             | Implements `event.SnapshotStore`                                           |
| **Projection**     | Read model built by consuming events from the event store             | Has a name, processes specific event types, tracks position via Checkpoint |
| **Checkpoint**     | Last processed event ID for a specific projection                     | Stored in a `CheckpointStore`; enables resume after restart                |
| **Tombstone**      | Soft-delete marker (metadata on an event)                             | `DetectTombstone()` reads status; `MarkTombstone()` sets it                |

### CQRS

| Term           | Definition                                                         | Context                                                                    |
| -------------- | ------------------------------------------------------------------ | -------------------------------------------------------------------------- |
| **Command**    | Action that mutates state                                          | Dispatched to exactly one handler; may produce events                      |
| **Query**      | Request for read-only data                                         | Dispatched to exactly one handler; returns a result                        |
| **Dispatcher** | Routes commands/queries to registered handlers                     | `command.Dispatcher`, `query.Dispatcher`                                   |
| **Handler**    | Function that processes a command or query                         | `command.Handler`, `query.Handler`                                         |
| **Decider**    | Pure-function aggregate: fold state from events, decide new events | `decider.Decider[State]` — no side effects, fully testable                 |
| **Repository** | Loads aggregate state, executes decider, saves events              | `decider.Repository[State]` — orchestrates Store + Bus + optional Snapshot |
| **Bus**        | Message bus for publishing/subscribing to events                   | `event.Bus` — Publisher + Subscriber                                       |

### Storage

| Term             | Definition                                                   | Context                                           |
| ---------------- | ------------------------------------------------------------ | ------------------------------------------------- |
| **Dialect**      | SQL dialect abstraction (`PostgresDialect`, `SQLiteDialect`) | Handles placeholder style, timestamp format, DDL  |
| **Store** (SQL)  | `SQLEventStore` — SQL-backed implementation of `event.Store` | `NewSQLiteEventStore(db)`, `NewSQLEventStore(db)` |
| **Pebble Store** | Embedded KV event store (no SQL dependency)                  | `pebble.NewPebbleStore(db, logger)`               |

### Saga

| Term             | Definition                                                      | Context                                         |
| ---------------- | --------------------------------------------------------------- | ----------------------------------------------- |
| **Saga**         | Long-running business process that may span multiple aggregates | Compensating transactions on failure            |
| **Saga Store**   | Persistence for saga state                                      | `saga.Store` — `Save`, `Load`, `LoadAllRunning` |
| **Compensation** | Reversing action taken by a saga step on failure                | Saga guarantees at-least-once execution         |

### Identity

| Term              | Definition                                                    | Context                                                |
| ----------------- | ------------------------------------------------------------- | ------------------------------------------------------ |
| **Branded ID**    | Strongly-typed ULID-backed identifier — `id.Of[T]`            | Prevents mixing IDs of different types at compile time |
| **AggregateID**   | String-based ID for aggregates (accepts any non-empty string) | `id.NewAggregateID()`, `id.ParseAggregateID()`         |
| **EventID**       | ULID-branded ID for events                                    | `id.NewEventID()` — time-sortable                      |
| **CorrelationID** | ULID tracing across distributed operations                    | `id.NewCorrelationID()` — links related events         |
| **CausationID**   | ULID tracking which event caused this event                   | `id.NewCausationID()` — causal chain                   |

### Error Taxonomy

All errors are classified into a 5-family taxonomy:

| Family             | Meaning                                                | Example                                        |
| ------------------ | ------------------------------------------------------ | ---------------------------------------------- |
| **Rejection**      | Business rule violation (4xx equivalent)               | `event.NewRejection(...)`                      |
| **Conflict**       | Optimistic concurrency or duplicate (409 equivalent)   | `event.NewConflict(...)`, `ErrVersionConflict` |
| **Transient**      | Retryable infrastructure failure (503 equivalent)      | `event.NewTransient(...)`                      |
| **Infrastructure** | Non-retryable infrastructure failure (500 equivalent)  | `event.NewInfrastructure(...)`                 |
| **Corruption**     | Data integrity violation — human intervention required | `event.NewCorruption(...)`                     |

---

## Interface Hierarchy

```
event.Store = EventSink + EventSource
  EventSink: Save(ctx, ref, events, expectedVersion), AppendBatch(ctx, ref, events)
  EventSource: Load(ctx, ref), LoadFromVersion(ctx, ref, version), LoadToVersion(ctx, ref, maxVersion), LoadToTimestamp(ctx, ref, maxTime)
  Journal: ReadAll(ctx)
  SeekableJournal: ReadFrom(ctx, afterEventID, limit)
  BackwardsSource: LoadBackwards(ctx, ref)
```

---

## Anti-Patterns (Terms We Avoid)

| Instead of              | We say                         | Why                                                                       |
| ----------------------- | ------------------------------ | ------------------------------------------------------------------------- |
| "Database"              | "Store" or "Event Store"       | CQRS separates write/read; "database" implies a single thing              |
| "Entity"                | "Aggregate"                    | DDD aggregate is the consistency boundary; entity is too vague            |
| "CRUD"                  | "Command + Event + Projection" | No updates or deletes — only append                                       |
| "Backend" (as a struct) | Individual store constructors  | `SQLBackend` couples unrelated concerns; prefer `NewSQLiteEventStore(db)` |

---

> **How to use this file:**
>
> - Keep terms concise — one clear sentence per definition
> - Update when new domain concepts emerge
> - Use these terms consistently in code, docs, and conversations
> - When in doubt about a word's meaning, check here first
