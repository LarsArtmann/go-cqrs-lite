# Metaengine Cookbook

Practical recipes for common metaengine patterns. Each recipe is copy-paste
ready and includes the query declaration, fold handlers, and reader usage.

## Counter Patterns

### Status Count Dashboard

Track counts by status for a dashboard endpoint. O(1) aggregate reads.

```go
type countInput struct{}

statusCounts := metaengine.Query[countInput, map[string]int64]("status_counts",
    metaengine.On(itemCreated{}, func(e itemCreated) metaengine.Delta {
        return metaengine.Delta{"open": +1}
    }),
    metaengine.On(itemCompleted{}, func(e itemCompleted) metaengine.Delta {
        return metaengine.Delta{"open": -1, "completed": +1}
    }),
    metaengine.On(itemDeleted{}, func(e itemDeleted) metaengine.Delta {
        return metaengine.Delta{"completed": -1}
    }),
)

// Read:
counts, _ := metaengine.ExecuteTyped[countInput, map[string]int64](
    ctx, store, countInput{})
// counts["open"], counts["completed"]
```

### Rate Limiter / Dedup Counter

Track request counts per user for rate limiting.

```go
type rateInput struct{ UserID string }

rateLimit := metaengine.Query[rateInput, map[string]int64]("rate_limit",
    metaengine.On(requestEvent{}, func(e requestEvent) metaengine.Delta {
        return metaengine.Delta{e.UserID: +1}
    }),
)
```

## Map Patterns

### CRUD Projection (Filtered + Sorted Scan)

The most common pattern: a read model with filterable, sortable fields.

```go
type listInput struct{ Status string }
type itemView struct {
    ID       string `json:"id"`
    Title    string `json:"title"`
    Status   string `json:"status"`
    Priority int    `json:"priority"`
}

items := metaengine.Query[listInput, itemView]("items",
    metaengine.On(createdEvt{}, func(e createdEvt) (string, itemView) {
        return e.ID, itemView{ID: e.ID, Title: e.Title, Status: "open"}
    }),
    metaengine.On(updatedEvt{}, func(e updatedEvt, prev itemView) itemView {
        prev.Title = e.Title
        return prev
    }),
    metaengine.On(deletedEvt{}, metaengine.Remove[itemView]()),
    metaengine.FilterOnField[itemView]("status", metaengine.FilterEq),
    metaengine.SortOnField[itemView]("priority", true),
)

// Reader:
reader := metaengine.NewReader[itemView](store, "items")
openItems, _ := reader.Scan(ctx,
    metaengine.WithFilter("status", metaengine.FilterEq, "open"),
    metaengine.WithSort("priority", true),
    metaengine.WithLimit(50))

// Point lookup:
item, found, _ := reader.Get(ctx, "item-123")
```

### EventDecoder with eventWithID Wrapper

For Map queries keyed by entity ID, the fold handler needs the stream ID.
Use EventDecoder to bridge event-sourced stream IDs to Map keys:

```go
// skip-validate
type eventWithID[P any] struct {
    ID      string
    Payload P
}

func myDecoder(evt event.Event) (any, error) {
    id := evt.StreamID().String()
    switch evt.Type() {
    case "created":
        var p createdPayload
        _ = json.Unmarshal(evt.Payload(), &p)
        return eventWithID[createdPayload]{ID: id, Payload: p}, nil
    default:
        return nil, fmt.Errorf("no fold for %q", evt.Type())
    }
}

adapter := projectionadapter.New("items", store, nil,
    projectionadapter.WithEventDecoder(myDecoder))
```

## Multi-Query Patterns

### Fan-Out (Multiple Read Patterns on Same Data)

Declare multiple queries on the same event types. The planner assigns each
to the best engine:

```go
store, _ := metaengine.Plan(
    []metaengine.Engine{metaengine.NewMemoryEngine(), sqliteEng},
    statusCounts,  // Counter → Memory (O(1) reads)
    items,         // Map with FilterOnField → SQLite (pushdown scans)
)
```

### Cursor Pagination

```go
page1, _ := reader.Scan(ctx,
    metaengine.WithFilter("status", metaengine.FilterEq, "open"),
    metaengine.WithSort("priority", true),
    metaengine.WithLimit(20))

// Use ScanPage for cursor-based pagination:
page1, cursor1, _ := reader.ScanPage(ctx,
    metaengine.WithSort("priority", true),
    metaengine.WithLimit(20))

page2, _, _ := reader.ScanPage(ctx,
    metaengine.WithSort("priority", true),
    metaengine.WithLimit(20),
    metaengine.WithCursor(cursor1.Value))
```

## Watcher Patterns

### Reactive Map Updates

Watch a Map query for live updates. This is the read-model push path that backs
`ServeSSE`:

```go
watcher := metaengine.NewWatcher[itemView](store, "items")
ch := watcher.Watch(ctx, nil) // nil = all keys

for v := range ch {
    if v.ID == "" {
        // Delete notification: Remove[itemView]() delivers the zero value.
        log.Printf("item %s deleted", lastKey)
        continue
    }
    log.Printf("item updated: %+v", v)
}
```

### Delete Notifications

A `Remove[V]()` fold emits a watcher value with the zero value of `V`. Do not
interpret "no value" as "deleted" — the zero value is the delete signal. Use
an explicit tombstone field or compare against the zero value to distinguish
updates from deletions:

```go
if v == (itemView{}) {
    // handle delete
}
```

### Cross-Engine Watcher Semantics

The watcher pipeline reifies engine-specific value representations back to the
declared type `V`:

- **Memory engine:** returns typed Go values directly (fast path, no alloc).
- **SQLite/Postgres/DuckDB:** stored JSON decodes to `map[string]any` (or to
  raw `jsonValue` for pushdown paths). The watcher JSON round-trips to `V`
  transparently.

The same watcher consumer works unchanged across engines.

## Engine Selection Patterns

### When to Use Each Engine

| Engine   | Best For                                       | Persistence        | Filter Pushdown                 |
| -------- | ---------------------------------------------- | ------------------ | ------------------------------- |
| Memory   | Counters, small datasets, testing              | Volatile           | O(N) Go-side                    |
| SQLite   | Filtered scans, point lookups with persistence | Persistent         | json_extract WHERE/ORDER BY     |
| Pebble   | Ultra-fast point lookups (LSM)                 | Dynamic (dir/mem)  | Raw value scan + closure filter |
| DuckDB   | Analytical queries, GROUP BY aggregations      | Dynamic (file/mem) | json_extract WHERE/ORDER BY     |
| Postgres | Remote persistent storage, JSONB + B-tree      | Persistent         | JSONB operator WHERE/ORDER BY   |

> **Persistence** is now a first-class type on `EngineProfile` (ADR-0098).
> The planner emits WARN when a query routes to a volatile engine with no
> persistent alternative, and INFO when a persistent alternative exists.

### TieredStore (Read Replicas)

Fan out writes to multiple stores, read from primary:

```go
primary, _ := metaengine.Plan([]metaengine.Engine{sqliteEng}, query)
replica, _ := metaengine.Plan([]metaengine.Engine{memEng}, query)
tiered := metaengine.NewTieredStore(primary, replica)

// Writes go to both:
tiered.Apply(ctx, "created", payload)
// Reads use primary:
reader := metaengine.NewReader[itemView](primary, "items")
```

### SwapEngine (Zero-Downtime Upgrade)

Replace an engine at runtime without stopping the service:

```go
// Warm up a new SQLite engine with data, then swap:
err := store.SwapEngine("memory", "sqlite", sqliteEng)
// All queries previously on "memory" now use "sqlite"
```

### QueryBuilder (Fluent API)

```go
qb := metaengine.NewQueryBuilder[itemView](reader)
results, _ := qb.
    Where("status", metaengine.FilterEq, "open").
    OrderBy("priority", true).
    Limit(50).
    Execute(ctx)
```

## Graph Patterns

Relations as events, traversals as queries. Fold follow/unfollow facts into
`metaengine.Edge` records; the planner classifies the query as the Graph ADT
and routes depth-limited reachability to the deployed engine. Full
composition-root wiring (system.New, decider guards, BDD suite):
[`example/graph-native`](../example/graph-native) and the skill's
[recipes §2.44](../.agents/skills/go-cqrs-lite/references/recipes.md).

### Follow Network with Retraction (Edge + EdgeRemoval)

```go
type Followed struct{ Follower, Followee string }
type Unfollowed struct{ Follower, Followee string }

// Node field = start node; Depth defaults to 1 when zero; Undirected is
// optional engine capability (HasUndirectedGraphSupport).
type Reachability struct {
    Node  string
    Depth int
}

followGraph := metaengine.Query[Reachability, []string]("follow_graph",
    metaengine.On(Followed{}, func(e Followed) metaengine.Edge {
        return metaengine.Edge{From: e.Follower, To: e.Followee}
    }),
    // ADR-0114 deletion-as-event: the retraction fold removes exactly the
    // edge the retracted fact added (engines without GraphRemoveEdge fail
    // loudly at apply time — there is no degraded fallback).
    metaengine.On(Unfollowed{}, func(e Unfollowed) metaengine.EdgeRemoval {
        return metaengine.EdgeRemoval{From: e.Follower, To: e.Followee}
    }),
)

// Apply facts, read reachability (typed; raw ExecuteCtx returns []any):
reach, _ := metaengine.ExecuteTyped[Reachability, []string](
    ctx, store, Reachability{Node: "alice", Depth: 2})
```

### Which Engine Serves the Traversal

| Engine | Traversal path | Undirected | Edge removal |
| ------ | ------------- | ---------- | ------------ |
| sqlite / turso | recursive CTE on `meta_graph_edges` (iterative BFS fallback on old servers) | yes | yes |
| postgres / duckdb | single WITH RECURSIVE statement | yes | yes |
| mysql | WITH RECURSIVE (8.0+), iterative fallback via probe | yes | yes |
| dgraph | native `n(depth:)` recurse | yes | yes |
| badger | prefix-scan BFS over adjacency keys | yes | yes |
| memory / iroh | in-process BFS / replicated passthrough | yes | yes |
| pebble / bbolt | not supported — the planner routes elsewhere or Plan fails | – | – |

> Node identity is payload-derived by convention: `Edge.From`/`To` are
> untyped endpoints, and stream-ID display strings are brand-prefixed
> (`"StreamMarker:alice"`) — carry clean node names in the event payload.
> The in-memory `metaengine/graphadapter` bridge additionally flattens all
> nodes to one label; engine-backed graphs are unaffected (its package doc
> details the `Driver()` escape hatch).
