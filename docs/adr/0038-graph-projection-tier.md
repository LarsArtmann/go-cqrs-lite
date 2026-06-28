# ADR-0038: Graph Projection Tier — Writes Portable, Reads Native

## Status

Accepted — 2026-06-28

## Context

The library had two projection tiers: document/KV (`Materialize` + `ViewStore`) and relational/SQL (`RelationalProjection`). Graph-structured read models — variable-depth traversal, adjacency, path-finding, causation DAGs — were served poorly by both (recursive CTEs in SQL, impossible in KV).

Research (`docs/research/graph-db-event-sourcing.html`) concluded that graph databases are the wrong shape for event STORES (append-only logs ≠ graph traversal) but the RIGHT shape for event PROJECTIONS — the "correct production answer."

## Decision

1. **Build a graph projection tier** (`graph/` module) that merges events into nodes and edges via a transactional `GraphSink`.
2. **Writes ARE portable**: `MergeNode`/`MergeEdge` map to openCypher `MERGE`, spoken by Neo4j, Memgraph, Apache Age, and RedisGraph. A handler written against `GraphSink` runs unchanged across all of them.
3. **Reads are NOT abstracted**: Cypher, Gremlin, and GQL differ too fundamentally. The `GraphDriver` exposes its underlying query mechanism directly. This asymmetry is documented at the interface level.
4. **Reference `MemoryDriver`** ships with zero production deps. Concrete graph DB drivers are consumer-pulled sibling modules (`graph/neo4j/`).

## Consequences

- All three data-model paradigms (document, relational, graph) now have projection tiers.
- Graph projection handlers are backend-portable for writes but backend-specific for reads.
- The library does NOT ship a Neo4j/Memgraph driver — consumers bring their backend.
- `graphtest/contract.go` provides a shared behavioral test suite for all future drivers.

## Alternatives Considered

- **Build graph reads too (Cypher abstraction)** — rejected: research problem, not engineering one. Cypher/Gremlin/GQL are too different.
- **Use a graph as event store** — rejected: research proved this is an anti-pattern (append-only ≠ graph traversal).
