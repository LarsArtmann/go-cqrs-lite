# ADR-0077: Metaengine GraphBackend vs graph/ module reconciliation

|             |                                                                                                   |
| ----------- | ------------------------------------------------------------------------------------------------- |
| **Status**  | **Amended** (see addendum below)                               |
| **Date**    | 2026-07-31 (original), 2026-08-06 (amendment)                 |
| **Context** | The metaengine ships a `GraphBackend` (in-memory BFS) that overlaps the dedicated `graph/` module |

## Context

The codebase has two independent graph implementations:

1. **`graph/` module** (Tier 3): A full graph projection tier with
   `GraphDriver`, `GraphProjection`, `MemoryDriver`, `Schema` (closed-world
   validation), `NodeRef`, `EdgeRef`, `GraphSink`, and a read API
   (`Query`, `Traverse`, `Neighbors`, `ShortestPath`). Designed as the third
   projection tier alongside `stack.Materialize` (document/KV) and
   `storage.RelationalProjection` (multi-table SQL). Documented in ADR-0033
   and ADR-0039.

2. **`metaengine/` GraphBackend** (Tier 0): An in-memory BFS graph stored in
   the `meta_graph_edges` table (SQLite) or `map[string]map[string][]string`
   (Memory). Exposed as one of the 7 ADTs the planner can assign queries to.
   The fold handler returns `metaengine.Edge{From, To}`, and the planner
   infers `ADTGraph`.

These serve **different conceptual layers**:

- `graph/` is a **projection tier** — it writes graph data (nodes + edges)
  from domain events and provides a rich read API for traversal queries.
  It is consumed via `projectionhost.Register(graphProjection)`.

- `metaengine/` GraphBackend is a **query ADT** — it stores edges as one
  possible projection shape the cost-based planner can choose. The planner
  assigns a `FoldEdge` query to the cheapest engine that supports graph
  operations. It is consumed via `store.Apply` + `ExecuteTyped`.

The overlap is real but the integration points differ. `graph/` is a
standalone projection with its own lifecycle. `metaengine/` GraphBackend
is a query result shape managed by the planner.

## Decision

**Keep both implementations. Do NOT delete GraphBackend.**

Rationale:

1. **GraphBackend is a planner ADT, not a projection tier.** Deleting it
   would leave the planner unable to assign graph-shaped queries. The 7-ADT
   model (Map, Set, Counter, Graph, Multimap, Log, SortedMap) is the
   planner's vocabulary; removing Graph would break the model.

2. **graph/ is the recommended path for graph projections.** Consumers who
   need graph read models (traversal, path-finding, adjacency) should use
   `graph.NewGraphProjection` — it has Schema validation, a rich read API,
   and is designed for the projection lifecycle.

3. **GraphBackend is for planner-managed graph queries.** Consumers who
   want the cost-based planner to assign graph queries across engines
   (Memory, SQLite, Pebble) should use `metaengine.Query` with `FoldEdge`.

4. **No consumer currently uses GraphBackend for real reads.** The
   `tasksByAssigneeQuery` fixture uses `FoldEdge` in tests, but no
   production code or example reads graph data via metaengine. This is
   acceptable — GraphBackend exists for planner completeness.

## Future Direction

If a consumer needs graph reads through the metaengine planner (e.g.,
hybrid queries that traverse both map and graph data), GraphBackend should
**delegate to a `graph.GraphDriver`** instead of reimplementing BFS. This
would mean:

- GraphBackend wraps a `graph.Driver` (MemoryDriver in-process, Neo4j
  driver remotely)
- `store.Apply` routes FoldEdge events to the driver via `MergeNode`/`MergeEdge`
- Reads go through the driver's native API

This unification is **not urgent** because no consumer needs it. It should
be done when the first real graph-query-through-planner use case appears.

## Consequences

- GraphBackend stays in the codebase as a planner ADT.
- graph/ remains the recommended projection tier for graph read models.
- No code changes required — this ADR documents the status quo and the
  intended direction.
- Consumers choosing between the two: use `graph/` for projections, use
  `metaengine/` only if you want the planner to manage graph queries.

---

## Addendum 2026-08-06: graph/ API Wins — Delete GraphBackend

**The original decision is superseded. graph/ becomes the single graph
abstraction. GraphBackend is deleted.**

### What Was Wrong

The original decision kept both systems separate based on two arguments:

1. "GraphBackend is a planner ADT" — but a graph ADT doesn't need its own
   storage backend interface. The planner can route graph-shaped folds to any
   engine that implements `graph.GraphDriver`. The ADT classification stays;
   the storage interface changes.

2. "Importing graph/ violates the zero-dependency boundary" — this was the real
   blocker, and it is now superseded (ADR-0062 addendum). The zero-dep boundary
   was an artificial constraint that kneecapped the planner.

### New Decision: graph/ API Wins

`graph.GraphDriver` implements `metaengine.Engine`. The planner routes
`ADTGraph` queries to GraphDriver-backed engines. The simple `Edge{From, To}`
fold return type auto-upgrades to `MergeEdge` calls internally.

- **Delete** `metaengine.GraphBackend` interface (2 methods: AddEdge, Neighbors)
- **Delete** graph implementations in memory_engine, sqlite_engine, pebbleengine
- **Keep** `ADTGraph` in the planner's ADT vocabulary
- **graph.GraphDriver** becomes a metaengine Engine (via adapter or directly)
- **graph.MemoryDriver** becomes the default in-memory graph engine
- **New engines** (Dgraph) implement GraphDriver, not GraphBackend
- **Schema validation** from graph/ applies to all planner-routed graph queries

The richer API (MergeNode/MergeEdge with properties, typed NodeRef/EdgeRef,
Schema validation, ReadableDriver with Query/Traverse/ShortestPath) becomes the
standard for all graph operations in the system.

See ADR-0113 for the full implementation plan.
