# metaengine/graphadapter

Bridge from the legacy `graph.GraphDriver` world (in-memory `graph.MemoryDriver`)
to the `metaengine.Engine` contract — registers as engine name **`graph-memory`**.

## When to use which

| You want | Use |
| --- | --- |
| Graph-native read models from event folds (Edge/EdgeRemoval), planner-routed, any engine | `metaengine.Query` Graph ADT via `system.RawQuery` — see the skill's `recipes.md` §2.44. The adapter is NOT on this path. |
| An in-memory graph engine satisfying `metaengine.Engine` (e.g. to hand to code that wants an Engine) | `graphadapter.New()` / `NewWithDriver(d)` |
| Label-rich direct reads (typed NodeRefs, Traverse/Neighbors/ShortestPath) | `Adapter.Driver()` — escape hatch to the underlying `graph.MemoryDriver` |

## API

```go
adapter := graphadapter.New()                 // engine name "graph-memory"
defer adapter.Close()

_ = adapter.GraphAddEdge(ctx, "follows", metaengine.Edge{From: "alice", To: "bob"})
neighbors, _ := adapter.GraphNeighbors(ctx, "follows", "alice", 2)
_ = adapter.GraphRemoveEdge(ctx, "follows", metaengine.Edge{From: "alice", To: "bob"})

driver := adapter.Driver()                    // *graph.MemoryDriver — direct reads
```

`Profile()` reports the graph capabilities; `Close()` closes the driver.

## Known limitation — flat node identity

`GraphAddEdge` synthesizes NodeRefs with a fixed label (`"entity"`) and key
prop (`"id"`), so all nodes share ONE namespace: distinct node types (User vs
Task) are not distinguishable, and non-string endpoints are stringified. For
label-rich graphs use `Driver()` directly, or prefer the engine-backed Graph
ADT path (sqlite/dgraph/…), which does not route through this adapter.
Richer edge/node labeling is deferred to v5
([ADR-0156](../../docs/adr/0156-graph-edge-labels-at-v5.md)).

Experimental — part of the metaengine family (see FEATURES.md).
