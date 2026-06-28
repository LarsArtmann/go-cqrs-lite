# Projection Tier Decision Guide

Three tiers, one contract (`projection.Projection`). Pick by your read pattern.

## Quick Decision Table

| Your dominant read pattern | Use | Backend reach |
|---|---|---|
| Get by ID, list all, prefix scan | `stack.Materialize` + `kv.ViewStore[V,K]` | memory, pebble, SQL-blob |
| Filtered/ordered/paginated lists, counts, stats | `storage.RelationalProjection` + `ProjectionSink` | SQLite, PostgreSQL |
| N-hop traversal, recursive relationships, adjacency | `graph.GraphProjection` + `GraphSink` | memory, Neo4j (future) |

## When to use Materialize (KV/Document)
- Your read model is a single document per key (a user profile, a todo item).
- You need Get by ID and Scan (list all), not WHERE/ORDER BY/LIMIT.
- You want the widest backend portability (memory, Pebble, SQL blob).
- Your backend might change at deployment time.

## When to use RelationalProjection (SQL)
- One event updates multiple related tables (messages + attachments + embeds).
- You need server-side WHERE, ORDER BY, LIMIT/OFFSET pagination.
- You need junction tables (many-to-many), append-only history tables.
- Your reads are structured queries, not document lookups.

## When to use GraphProjection (Graph)
- Your dominant reads are variable-depth traversal (reply chains, social graphs).
- You need path-finding or connected-component queries.
- Relationships between entities are as important as the entities themselves.
- Relational recursive CTEs are too slow or too complex.

## Running projections

All three tiers implement `projection.Projection`. Use `bundle.RunProjections`:

```go
bundle, _ := sqlite.New("app.db")
defer bundle.Close()

// Build projections (any tier, mixed):
mat := &stack.Materialize[UserView, UserID]{...}
relProj, _ := storage.NewRelationalProjection(...)
graphProj, _ := graph.NewGraphProjection(...)

// One call replays journal + subscribes to live + dispatches to all:
err := bundle.RunProjections(ctx, mat, relProj, graphProj)
```

## Can I use multiple tiers together?
Yes. Each projection processes the same events independently. A single application can materialize a KV cache for fast lookups, a SQL table for filtered queries, and a graph for social traversal — all from the same event stream.
