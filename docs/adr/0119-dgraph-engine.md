# ADR-0119: Dgraph Engine (Distributed Graph Database)

**Date:** 2026-08-06
**Status:** Proposed (design complete, implementation deferred)
**Related:** ADR-0113 (graphadapter), ADR-0096 (Iroh distributed engine bridge evaluation)

## Context

The metaengine's GraphBackend enables graph-based projections (traversal, adjacency,
path-finding). Currently three implementations exist:

1. **Memory engine** — in-process BFS, O(N^d), no persistence
2. **Pebble/SQLite/Badger engines** — prefix-scan BFS, O(N^d), persistent
3. **graphadapter** — wraps `graph.MemoryDriver` as a metaengine Engine

All current graph backends are single-node. For large-scale graph workloads (social networks,
recommendation engines, fraud detection), a distributed graph database is needed.

Dgraph is an open-source distributed graph database with:

- GraphQL/DQL query language
- Horizontal scalability (sharding by predicate)
- gRPC + HTTP APIs
- Built-in full-text search and geo queries

## Decision

Design a `metaengine/dgraphengine/` module that implements `metaengine.Engine` +
`metaengine.GraphBackend` via Dgraph's gRPC API (`dgraph-io/dgo/v240`).

### Architecture

```
consumer → metaengine.Store → dgraphengine.Engine → dgo client → Dgraph cluster
                                         ↓
                                    GraphBackend:
                                    - GraphAddEdge → DQL mutation (add edge + nodes)
                                    - GraphNeighbors → DQL query (recursive traversal)
```

### Key Design Points

1. **Graph-only:** The engine declares only `ADTGraph` in its EngineProfile.
   Map/Set/Counter/Log ADTs are not supported (ErrUnsupportedADT).

2. **gRPC transport:** Uses `dgo.NewDgraphClient()` with configurable endpoints.
   Supports TLS and ACL authentication.

3. **DQL mapping:**
   - `GraphAddEdge(col, edge)` → `mutation { set { <from> <col_edge> <to> . } }`
   - `GraphNeighbors(col, node, depth)` → `query { node(func: uid(node)) { ~col_edge @cascade { uid } } }`

4. **Persistence:** Always `PersistencePersistent` (Dgraph is a persistent database).

5. **Replication:** `ReplicationSingleLeader` (Dgraph uses Raft consensus per group).

### Implementation Plan (Deferred)

The implementation requires a running Dgraph cluster for integration testing. The module
will be created when:

- A consumer requests it, OR
- CI infrastructure with Dgraph is available

The module structure would follow the `pgengine` pattern:

```
metaengine/dgraphengine/
├── go.mod          # dgraph-io/dgo/v240 dep
├── engine.go       # Engine + GraphBackend impl
└── adt_matrix_test.go  # Skip when DGRAPH_ADDR not set
```

## Alternatives Considered

### Neo4j

Neo4j is the market leader for graph databases but has no official Go driver. Community drivers
exist but lack long-term support. A Neo4j engine could be added later via the Bolt protocol.

### Apache AGE (Postgres extension)

Apache AGE adds Cypher support to Postgres. The existing `pgengine` could be extended to use
AGE for graph queries, avoiding a separate module. This is the most pragmatic long-term option.

### RedisGraph

In-memory graph module for Redis. Good for latency-sensitive workloads but data does not
persist by default.

## Consequences

- **Positive:** Design is documented for future implementation.
- **Positive:** Consumers requiring distributed graph can use `pgengine` with Apache AGE as
  an interim solution.
- **Negative:** No immediate implementation — deferred until demand or infrastructure exists.
