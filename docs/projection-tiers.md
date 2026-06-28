# Projection Tier Decision Guide

Three tiers, one contract (`projection.Projection`). Pick by your read pattern.

## Quick Decision Table

| Your dominant read pattern                          | Use                                               | Backend reach            | Schema strength |
| --------------------------------------------------- | ------------------------------------------------- | ------------------------ | --------------- |
| Get by ID, list all, prefix scan                    | `stack.Materialize` + `kv.ViewStore[V,K]`         | memory, pebble, SQL-blob | Implicit (Go type) |
| Filtered/ordered/paginated lists, counts, stats     | `storage.RelationalProjection` + `ProjectionSink` | SQLite, PostgreSQL       | `RelationalSchema` (columns, types, nullable) |
| N-hop traversal, recursive relationships, adjacency | `graph.GraphProjection` + `GraphSink`             | memory, Neo4j (future)   | `graph.Schema` (node/edge types, props) — **opt-in** |

## Why three tiers, not one

Each tier serves a fundamentally different **read pattern**. Choosing the wrong tier means either hand-writing raw SQL/Cypher (defeating the library's purpose) or accepting queries that are orders of magnitude slower than they need to be.

### Document/KV (`Materialize`)

One document per key. The Go generic type IS the schema. Best for:
- Point lookups by ID
- List/scan all documents
- Tombstone-aware soft-delete

**Cannot do:** Server-side WHERE, ORDER BY, JOIN, variable-depth traversal.

### Relational/SQL (`RelationalProjection`)

Multiple related tables per event. Atomic cross-table writes. Best for:
- Filtered, ordered, paginated queries (WHERE, ORDER BY, LIMIT)
- Junction tables (many-to-many)
- Append-only history tables
- Count/stats queries

**Cannot do:** Variable-depth traversal (needs recursive CTEs — slow and complex). JOINs are not supported — denormalize FK columns in the projection handler instead (documented decision).

### Graph/traversal (`GraphProjection`)

Nodes and edges via MERGE semantics. Best for:
- Variable-depth traversal (reply chains, social graphs, causation DAGs)
- Path-finding (shortest path, connected components)
- Adjacency queries (1-hop neighbors)
- Where relationships between entities are as important as the entities themselves

**Cannot do (by design):** N-ary relations with typed roles (breaks MERGE portability — use relational junction tables instead). Reads from external graph databases are NOT abstracted (run native Cypher/Gremlin).

## Schema: closed-world vs open-world

| Tier | Default | With Schema |
| --- | --- | --- |
| Events | `schema.Validator` (closed-world) | — |
| Relational | `RelationalSchema` + column-name validation (closed-world) | — |
| Graph | Open-world (any label, any props — like Neo4j) | `graph.Schema` → closed-world (rejects unknown labels/props at sink) |

**Closed-world** means the sink rejects writes with unknown labels, properties, or endpoint types before they hit storage. This catches the most common projection bug: a typo in a label or property name that silently creates a phantom node no query will ever find.

**Open-world** (graph default, nil Schema) means any write is accepted. Useful for rapid prototyping or when the graph shape is not fully known. Set a Schema when the projection is stable.

## Graph Schema — what we adopted from TypeDB, what we rejected

**Adopted:** Boundary-typing at the sink. TypeDB's core insight is that graph-shaped read models don't require abandoning strong types. We validate node labels, edge types, property names, and edge endpoint constraints — the same pattern the relational tier uses for column-name validation.

**Rejected (and why):**
- **N-ary relations / typed roles** — breaks openCypher MERGE portability (the graph tier's central thesis). Use relational junction tables for multi-way relationships.
- **Inheritance polymorphism** — querying a supertype matching all subtypes. YAGNI for read models.
- **Full constraint grammar** (`@regex`, `@values`, cardinality ranges) — database-engine concerns, not sink-validation concerns. Keep the schema minimal: types, keys, properties, endpoint constraints.
- **Reasoning/inference engine** — TypeDB generates derived facts at query time. In event sourcing, a projection IS materialized derived data (the opposite philosophy). The planned `Deriver` module (ADR-0040) handles event→event derivation at the application layer, not inside the database.

## Can I use multiple tiers together?

Yes. Each projection processes the same events independently. A single application can materialize a KV cache for fast lookups, a SQL table for filtered queries, and a graph for social traversal — all from the same event stream.

```go
err := bundle.RunProjections(ctx, kvProjection, sqlProjection, graphProjection)
```

## Running projections

All three tiers implement `projection.Projection`. Use `bundle.RunProjections`:

```go
bundle, _ := sqlite.New("app.db")
defer bundle.Close()

mat := &stack.Materialize[UserView, UserID]{...}
relProj, _ := storage.NewRelationalProjection(...)
graphProj, _ := graph.NewGraphProjection("my-graph", driver, handler, types,
    graph.WithSchema(mySchema))

err := bundle.RunProjections(ctx, mat, relProj, graphProj)
```

## Graph read API (MemoryDriver only)

The `MemoryDriver` implements `ReadableDriver` — a typed Go-native read surface:

```go
// Query: filter by label + Go predicate function
nodes := driver.Query(graph.Pattern{
    Label: "User",
    Where: func(props map[string]any) bool { return props["active"] == true },
})

// Traverse: BFS following edge type
ancestors := driver.Traverse(msgRef, "REPLY_TO", -1)

// ShortestPath: BFS with parent tracking
path, err := driver.ShortestPath(userA, userB)

// Neighbors: 1-hop adjacency
nodes, edges := driver.Neighbors(centerNode)
```

This is deliberately NOT a query language. The Go compiler type-checks predicate functions; a query parser cannot. For external graph databases (Neo4j, Memgraph), run native Cypher/Gremlin queries via their driver — per ADR-0038, reads are engine-native.
