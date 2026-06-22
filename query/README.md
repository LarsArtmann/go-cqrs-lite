# query — CQRS Query Dispatch

[![Go Reference](https://pkg.go.dev/badge/github.com/larsartmann/go-cqrs-lite/query/v2.svg)](https://pkg.go.dev/github.com/larsartmann/go-cqrs-lite/query/v3)

Typed query dispatch with pagination and middleware chains.

```bash
go get github.com/larsartmann/go-cqrs-lite/query/v3
```

## Quick Start

```go
queries := query.NewDispatcher()
queries.Register("user.get", handler)
result, err := queries.Dispatch(ctx, q)

// Type-safe result
user, err := query.DispatchTyped[*GetUserResult](ctx, queries, q)
```

## Key Types

| Type                   | Purpose                                                   |
| ---------------------- | --------------------------------------------------------- |
| `Dispatcher`           | Query dispatcher with handler registry + middleware chain |
| `Query`                | Interface: Type()                                         |
| `BasicQuery`           | Embed for interface satisfaction                          |
| `TypedHandler[Q, R]`   | Type-safe handler returning (R, error)                    |
| `Pagination`           | Page/PageSize with Offset(), Validate()                   |
| `PaginatedResult[T]`   | Data + TotalCount + HasNext()/HasPrev()                   |
| `PersistedQuery`       | Stored query record for audit/replay                      |
| `QueryStore`           | `QuerySink + QuerySource` — persist queries               |
| `QueryJournal`         | `ReadAllQueries(ctx)` — full query audit trail            |
| `SeekableQueryJournal` | `ReadQueriesFrom(ctx, afterID, limit)` — batched replay   |

## Query Persistence & Audit

Queries can be persisted for audit ("who queried what data and when?") — the query-side equivalent of event sourcing:

```go
// Create a persisted query record
pq, err := query.NewPersistedQuery("user.get", payload,
    query.WithQueryCorrelationID(corrID))

// Persist via a QueryStore (Sink + Source)
store := memory.NewMemoryQueryStore()
store.SaveQuery(ctx, pq)

// Read back (after a given timestamp)
queries, _ := store.LoadQueries(ctx, someTime)

// Cross-aggregate audit trail (Journal)
allQueries, _ := store.ReadAllQueries(ctx)

// Position-based replay (SeekableJournal)
batch, _ := store.ReadQueriesFrom(ctx, lastRequestID, 100)
```

## Related Modules

- [event/v2](../event/README.md) — Event store/bus with Journal/SeekableJournal pattern
- [command/v2](../command/README.md) — Command dispatch with parallel PersistedCommand/CommandStore
- [decider/v2](../decider/README.md) — Aggregate pattern producing the events queries read
- [memory/v2](../memory/README.md) — `MemoryQueryStore` in-memory implementation
- [middleware/v2](../middleware/README.md) — Logging, retry, tracing for queries
- [id/v2](../id/README.md) — Branded `RequestID`
