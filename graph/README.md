# graph — Projection tier for graph-structured read models

[![Go Reference](https://pkg.go.dev/badge/github.com/larsartmann/go-cqrs-lite/graph/v3.svg)](https://pkg.go.dev/github.com/larsartmann/go-cqrs-lite/graph/v3)

The third fundamental projection tier, alongside `kv.ViewStore` (document/KV)
and `storage.RelationalProjection` (relational/SQL). Use it when the dominant
read patterns are **variable-depth traversal, adjacency, path-finding, or
connected-component queries** — reply chains, social graphs, causation DAGs,
role memberships, reaction networks.

```bash
go get github.com/larsartmann/go-cqrs-lite/graph/v3
```

## When to use which tier

| Read pattern                                                | Tier                                        |
| ----------------------------------------------------------- | ------------------------------------------- |
| Get/Lookup by ID, list all                                  | `kv.ViewStore[V,K]` + `stack.Materialize`   |
| Filtered/ordered/paginated lists, counts, stats             | `storage.RelationalProjection` + `Row` sink |
| **N-hop traversal, recursive relationships, graph queries** | **`graph.GraphProjection`**                 |

## Scope: writes portable, reads native

Every graph database supports MERGE/upsert semantics on nodes and edges
(openCypher `MERGE` is spoken by Neo4j, Memgraph, Apache Age, RedisGraph), so
`GraphSink` with `MergeNode`/`MergeEdge` is genuinely cross-backend — exactly
as the SQL `Dialect` tier makes relational writes portable.

Graph reads are **not** abstracted. Cypher, Gremlin, and GQL differ enough that
abstracting them is a research problem rather than an engineering one. A
`GraphDriver` exposes its underlying query mechanism directly; consumers run
native queries against it. This asymmetry is documented, not hidden.

## Quick start

```go
driver := graph.NewMemoryDriver() // or graph/neo4j.NewDriver(uri, auth)

proj, _ := graph.NewGraphProjection("social-graph", driver,
    func(ctx context.Context, evt event.Event, sink graph.GraphSink) error {
        var p MessageCreated
        _ = json.Unmarshal(evt.Payload(), &p)

        msgRef := graph.NodeRef{Label: "Message", KeyProp: "id", KeyValue: p.ID}
        _ = sink.MergeNode(msgRef, map[string]any{"created_at": p.CreatedAt})

        // Auto-creates endpoint nodes — handlers need not pre-merge.
        _ = sink.MergeEdge(graph.EdgeRef{
            Type: "AUTHORED_BY", From: msgRef,
            To: graph.NodeRef{Label: "User", KeyProp: "id", KeyValue: p.AuthorID},
        }, nil)

        // The recursive edge — relational tier needs WITH RECURSIVE CTE.
        if p.ReplyToMessageID != "" {
            _ = sink.MergeEdge(graph.EdgeRef{
                Type: "REPLY_TO", From: msgRef,
                To: graph.NodeRef{Label: "Message", KeyProp: "id", KeyValue: p.ReplyToMessageID},
            }, map[string]any{"at": p.CreatedAt})
        }

        return nil // atomic: all merges commit or all roll back
    },
    []event.Type{"MESSAGE_CREATED"},
)

// proj implements event.Projection → register with any projection runner.
_ = proj.Handle(ctx, evt)

// Reads (in-memory): driver.Snapshot() returns the graph for traversal.
// Reads (Neo4j): run native Cypher via the driver module.
```

## Atomicity

`GraphProjection.Handle` runs the handler inside a single driver transaction.
All sink writes commit atomically when the handler returns nil and roll back
when it returns an error. A real graph database's transaction (Neo4j, Memgraph)
provides the same guarantee as the in-memory `MemoryDriver`, which snapshots
the graph on Begin and swaps it back only on commit.

## Backends

| Driver                           | Module                                          | Use case                                             |
| -------------------------------- | ----------------------------------------------- | ---------------------------------------------------- |
| `MemoryDriver`                   | `graph/v3` (this module)                        | Tests, single-process local-first apps. Zero deps.   |
| Neo4j / openCypher               | consumer-pulled sibling module (`graph/neo4j/`) | Production graph queries. Brings its own driver dep. |
| Memgraph, Apache Age, RedisGraph | future                                          | All speak openCypher MERGE — same sink interface.    |

The Neo4j driver is deliberately **not** in this module — same convention as
`storage/pebble` and `storage/turso` being separate from `storage/`. Consumers
bring their backend; the library provides the portable sink abstraction.

## Related Modules

- [**event/v3**](../event/README.md) — `event.Projection` interface this tier implements
- [**storage/v3**](../storage/README.md) — `RelationalProjection` (the relational peer tier)
- [**stack/v3**](../stack/README.md) — `Materialize` (the document/KV peer tier)
- [**kv/v3**](../kv/README.md) — `ViewStore[V,K]` (the document interface)
