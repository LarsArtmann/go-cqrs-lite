# Domain Glossary

Core concepts in go-cqrs-lite, defined precisely.

## Aggregate

A cluster of domain objects treated as a single unit. Each aggregate has a type, a branded ID (`id.AggregateID`), and a linear version history. Events are the source of truth — the aggregate's current state is derived by folding its events.

## Command

An intent to change state. Commands are dispatched to a single handler, which validates business rules and produces events. Commands carry an `IdempotencyKey()` for deduplication. Unlike events, commands represent "what you want to happen," not "what happened."

## Decider

A pure-function pattern for implementing aggregates. `Decider[State]` holds:

- `Initial State` — the zero value
- `Fold func(State, Event) (State, error)` — applies events to produce new state

The `DecideFunc` takes a command and current state, returns events. No mutable state, no 9-method interface. The `Repository[State].Execute` method handles load → fold → decide → save → publish.

## Event

An immutable record of something that happened. Events carry: type, aggregate ID, version, payload, schema version, occurred-at timestamp, and optional metadata (correlation ID, causation ID, client ID). Events are the source of truth in event sourcing.

## Fold

The process of applying a sequence of events to an initial state to derive the current state. Also called "rehydration" or "replay." The fold function must be pure — same events + same initial state always produces the same result.

## Projection

A read-model derived from events. Projections subscribe to event types and maintain a denormalized view optimized for queries. The `projection.Runner` handles replay (catch-up from checkpoint) and live subscription. Projections can be rebuilt from scratch at any time — state is disposable.

## Query

A request for information. Queries are dispatched to handlers that return typed results. Supports pagination via `PaginatedResult[T]`. Queries never modify state.

## Saga

A long-running process that orchestrates multiple aggregates or services. Sagas consist of ordered steps; each step dispatches a command. On failure, the saga compensates by running `Compensate` functions in reverse order. State is persisted in a `saga.Store` for crash recovery.

## Event Store

The append-only log of all events. Supports loading by aggregate, loading from a position (for projections), and time-travel queries (`LoadToVersion`, `LoadToTimestamp`). Implementations: `MemoryStore` (testing), `SQLEventStore` (PostgreSQL/SQLite), `PebbleEventStore` (embedded).

## Outbox

A reliable publishing mechanism. Events are first appended to the outbox (in the same transaction as the event store), then a background poller publishes them to the event bus. Guarantees at-least-once delivery.

## Branded ID

A type-safe identifier: `id.Of[T]` = `cbid.ID[T, ulid.ULID]`. Each domain concept has its own branded type (`AggregateID`, `EventID`, `UserID`, etc.) preventing accidental mixing of IDs at compile time.

## Error Taxonomy

Five error families for classified error handling:

- **Rejection** — business rule violation (no retry)
- **Conflict** — concurrency/version conflict (no retry)
- **Transient** — temporary failure (retry)
- **Infrastructure** — system-level failure (maybe retry)
- **Corruption** — data integrity violation (no retry)

## Middleware

Cross-cutting concerns applied to command/query/event handlers. Middleware wraps handlers in a chain (last added runs first). Built-in: logging, retry, recovery, validation, metrics, tracing.
