# Domain Glossary — go-cqrs-lite

Core concepts and terminology used throughout the library.

## Aggregate

A consistency boundary that groups related events. Each aggregate has a unique ID, a type, and a version (optimistic concurrency). Aggregates are loaded by replaying their event history and can produce new events via commands.

See: `core/aggregate/`, `core/decider/`

## Decider

A pure-function alternative to the traditional OO aggregate. A `Decider[State]` holds an `Initial` state and a `Fold` function — no mutable state, no interface with 9 methods. The `Repository[State]` handles load → fold → decide → save → publish.

See: `core/decider/`

## Fold

A pure function `(state, event) → (newState, error)` that applies a single event to the current state. Folding all events for an aggregate reconstructs its current state. Must be side-effect-free.

## Decide

A function `(state, version) → ([]events, error)` that takes the current state and returns new events to persist. Returning an error rejects the command (no events saved). Returning an empty slice is a no-op.

## Event

An immutable record of something that happened. Each event carries a type, an aggregate ID/reference, a version number, a payload, metadata, and an occurrence timestamp. Events are the source of truth — they are never modified after creation.

See: `core/event/`

## Event Sourcing

A persistence pattern where state is derived entirely from a sequence of events, rather than storing the current state directly. Loading an aggregate means replaying its event history through a fold function.

## Command

An intent to change state. Commands are dispatched to a handler, which validates business rules and produces events. Commands carry an `IdempotencyKey()` for deduplication and an `AggregateID()` for routing.

See: `core/command/`

## Query

A request for information. Queries are dispatched to a handler and return results, optionally paginated. Queries must not mutate state.

See: `core/query/`

## Projection

A read-model built by consuming events. Projections subscribe to specific event types (or all events) and update a denormalized view. The `Runner` supports replay (from the event store) and live subscription (from the event bus) with per-projection checkpoints.

See: `projection/`

## Outbox

A reliability mechanism for event publishing. Instead of publishing events directly to the bus after saving, events are staged in an outbox table within the same database transaction. A background publisher polls the outbox and publishes to the bus, guaranteeing at-least-once delivery even if the bus is temporarily unavailable.

See: `core/event/outbox.go`, `storage/outbox.go`

## Snapshot

A point-in-time serialization of an aggregate's state at a specific version. Snapshots accelerate loading by allowing the system to skip replaying events before the snapshot version. Snapshot strategy (e.g., every N events) is configurable.

See: `core/event/snapshot.go`, `storage/snapshot.go`

## Checkpoint

A recorded position in the event stream, used by projections to track which events have been processed. Each projection maintains its own checkpoint, enabling independent replay and recovery.

See: `storage/checkpoint.go`

## Branded ID

A type-safe identifier using Go generics (`id.Of[T]`). Each domain concept gets its own ID type (AggregateID, EventID, UserID, etc.) that is incompatible with other ID types at compile time. Prevents accidental mixing of IDs.

See: `core/pkg/id/`

## Catalog

A registry of all commands, events, and queries in a service, with associated metadata and JSON schemas. Used to generate documentation (AsyncAPI 3.0, EventCatalog MDX, D2 diagrams) from Go type definitions.

See: `catalog/`

## Codec

An encoder/decoder for serializing aggregate state to/from bytes. Used by the snapshot mechanism. The library provides a `JSONCodec` implementation.

See: `core/event/codec.go`

## Bus

A publish-subscribe mechanism for events. Producers publish events; consumers subscribe by event type. The library provides an in-memory `MemoryBus` for testing.

See: `core/event/`, `memory/bus.go`

## Store

A persistence interface for events. Supports saving with optimistic concurrency, loading by aggregate, loading from a specific version, and deletion. The library provides `SQLEventStore` (PostgreSQL) and `MemoryStore` (testing).

See: `core/event/store.go`, `storage/event_store.go`

## Middleware

Cross-cutting concerns applied to command, query, and event pipelines. Includes logging, metrics, retry with exponential backoff, recovery from panics, and validation.

See: `middleware/`

## Error Taxonomy

A classification system for errors into five families:
- **Rejection** — business rule violation (not retryable)
- **Conflict** — optimistic concurrency or duplicate (not retryable)
- **Transient** — temporary infrastructure failure (retryable)
- **Corruption** — data integrity violation (not retryable)
- **Infrastructure** — non-transient system error (not retryable)

See: `core/event/errors.go`
