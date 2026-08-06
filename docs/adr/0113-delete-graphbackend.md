# ADR-0113: Delete GraphBackend — graph.GraphDriver Implements Engine

**Date:** 2026-08-06
**Status:** Accepted
**Supersedes:** ADR-0077 (original decision to keep both)
**Related:** ADR-0077 addendum, ADR-0062 addendum (dep boundary removed)

## Context

The codebase has two graph abstractions (detailed in ADR-0077):

1. `metaengine.GraphBackend` — 2 methods (AddEdge, Neighbors), untyped,
   undirected, no properties, no schema. Implemented by Memory, SQLite, Pebble.
2. `graph.GraphDriver` — rich API (MergeNode/MergeEdge/RemoveNode/SetNodeProperty,
   ReadableDriver with Query/Traverse/ShortestPath), typed NodeRef/EdgeRef,
   Schema validation. Implemented by MemoryDriver (future: Dgraph, Neo4j).

ADR-0077 kept both because importing graph/ violated the zero-dep boundary.
That boundary is now removed (ADR-0062 addendum).

## Decision

**Delete `metaengine.GraphBackend`. The `graph/` API is the single graph
abstraction. `graph.GraphDriver` implements `metaengine.Engine`.**

### Changes

1. **Delete `GraphBackend` interface** from metaengine (2 methods: AddEdge,
   Neighbors). Delete its implementations in memory_engine, sqlite_engine,
   pebbleengine.

2. **Keep `ADTGraph`** in the planner's ADT vocabulary. The planner still
   classifies folds that return `Edge` as graph queries.

3. **`graph.GraphDriver` becomes a metaengine Engine.** A new adapter (or direct
   implementation) makes GraphDriver available to the planner. The planner routes
   `ADTGraph` queries to GraphDriver-backed engines.

4. **Simple `Edge{From, To}` folds auto-upgrade** to `MergeEdge` calls
   internally. The planner synthesizes `NodeRef` and `EdgeRef` from the simple
   Edge type.

5. **Schema validation** from graph/ applies to planner-routed graph queries
   when a schema is declared.

6. **New engines implement GraphDriver, not GraphBackend.** Dgraph engine
   implements `graph.GraphDriver` with DQL queries. Neo4j implements it with
   Cypher. MemoryDriver is the default in-process graph engine.

7. **GraphBackend cost profile changes.** Memory goes from O(N) BFS scan to
   O(degree^depth) true BFS via MemoryDriver. SQLite/Pebble lose graph support
   (use MemoryDriver or Dgraph engine instead).

### Migration Path

1. Create `metaengine/graphadapter/` that wraps `graph.GraphDriver` as
   `metaengine.Engine` with `ADTGraph` support.
2. Update `adttest.RunMatrix` to test graph via GraphDriver adapter.
3. Delete `GraphBackend` interface and its 3 implementations.
4. Update planner to route `ADTGraph` to GraphDriver engines.

## Consequences

- **Positive:** One graph abstraction, not two. Rich API (properties, schema,
  traversal, shortest path) is the standard everywhere.
- **Positive:** Dgraph engine gets a real graph DB backend, not a degraded scan.
- **Positive:** MemoryDriver's true BFS (O(degree^depth)) replaces the O(N) scan
  fallback.
- **Negative:** SQLite and Pebble lose native graph support. They must use
  MemoryDriver (in-process) or Dgraph (server) for graph queries.
- **Negative:** The simple `Edge{From, To}` fold handler no longer maps directly
  to a storage operation. It goes through NodeRef/EdgeRef synthesis.
