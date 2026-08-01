# Migration Guide: kv.ViewStore → metaengine

> **When to use this guide:** You have a projection built on `stack.Materialize[V,K]` + `kv.ViewStore[V,K]` and want to migrate to the metaengine planner for multi-engine routing, filtered scans, and cost-based optimization.

## TL;DR Decision Matrix

| If your projection needs...                      | Use                                                        |
| ------------------------------------------------ | ---------------------------------------------------------- |
| Single key CRUD (get/set/delete one document)    | **Keep `kv.ViewStore`** — simpler API, fewer deps          |
| Filtered queries (WHERE, ORDER BY, LIMIT)        | **metaengine** with `ScanBackend` or `FilterOnField`       |
| Counter aggregations (count by status)           | **metaengine** with `ADTCounter`                           |
| Multi-ADT (docs + counters + graph in one store) | **metaengine** — one engine, multiple ADTs                 |
| Vector/search/spatial queries                    | **metaengine** — the only option                           |
| SQL-backed views with WHERE/ORDER BY             | **`storage.SQLViewStore`** (existing, no migration needed) |

## Why Migrate?

The metaengine planner offers capabilities that `kv.ViewStore` cannot:

1. **Cost-based engine routing** — Memory for dev, SQLite/Postgres for prod, automatically
2. **Filtered scans** — `FilterOnField` + `SortOnField` push to SQL WHERE/ORDER BY
3. **Multi-ADT** — counters, maps, graphs, vectors, search, spatial in one store
4. **Temporal queries** — `MapGetAsOf(t)` for point-in-time reads (Memory engine)
5. **Typed reads** — `TypedReader[V]` and `QueryBuilder[V]` fluent API

## Migration Steps

### Step 1: Define your event payloads

```go
// Before (kv.ViewStore): your projection handled events manually
type TodoView struct {
    ID        string
    Title     string
    Completed bool
}

// After (metaengine): same struct, but now registered as a query result type
```

### Step 2: Replace Materialize with metaengine.Plan

```go
// BEFORE: kv.ViewStore pattern
store, _ := sqlite.NewViewModel[TodoView, TodoID](bundle, mapper)
mat := stack.Materialize[TodoView, TodoID]{
    Store:        store,
    KeyFromEvent: keyFunc,
    OnCreate:     onCreate,
    OnUpdate:     onUpdate,
    OnTombstone:  onTombstone,
}

// AFTER: metaengine pattern
store, _ := metaengine.Plan(
    []metaengine.Engine{metaengine.NewMemoryEngine()},
    metaengine.Query[ListInput, []TodoView]("todos",
        metaengine.On(TodoCreated{}, func(e TodoCreated) (string, TodoView) {
            return e.ID, TodoView{ID: e.ID, Title: e.Title, Completed: false}
        }),
        metaengine.On(TodoCompleted{}, func(e TodoCompleted) (string, TodoView) {
            return e.ID, TodoView{ID: e.ID, Completed: true}
        }),
    ),
)
```

### Step 3: Replace projection handler with Apply

```go
// BEFORE: router handler
router.AddNoPublisherHandler("todos", topic, sub, mat.HandlerFunc())

// AFTER: store.Apply in your projection loop
for evt := range events {
    store.Apply(ctx, evt.Type(), evt.Payload())
}
```

### Step 4: Replace kv queries with metaengine reads

```go
// BEFORE: kv.ViewStore
result, _ := store.Query(ctx, kv.ViewQuery{
    Conditions: []kv.Condition{{Column: "completed", Op: kv.OpEq, Value: false}},
})

// AFTER: metaengine TypedReader
reader := metaengine.NewReader[TodoView](store, "todos")
results, _ := reader.Scan(ctx,
    metaengine.WithFilter("completed", metaengine.FilterEq, false),
    metaengine.WithSort("title", false),
    metaengine.WithLimit(50),
)
```

### Step 5: Wire to projectionhost (optional)

```go
// For managed lifecycle (checkpoint, DLQ, restart):
adapter := projectionadapter.New("todos", store, nil,
    projectionadapter.WithEventDecoder(decodeEvent))
host.Register(adapter)
```

## Common Patterns

### Counter Aggregation

```go
// Count todos by status — no separate projection needed
store, _ := metaengine.Plan(
    []metaengine.Engine{metaengine.NewMemoryEngine()},
    metaengine.Query[CountInput, map[string]int64]("todo_counts",
        metaengine.On(TodoCreated{}, func(e TodoCreated) metaengine.Delta {
            return metaengine.Delta{"open": 1}
        }),
        metaengine.On(TodoCompleted{}, func(e TodoCompleted) metaengine.Delta {
            return metaengine.Delta{"open": -1, "done": 1}
        }),
    ),
)
counts, _ := metaengine.ExecuteTyped[CountInput, map[string]int64](ctx, store, CountInput{})
// counts["open"] = 5, counts["done"] = 3
```

### Multi-Engine (Memory + SQLite)

```go
// The planner routes each query to the cheapest engine that supports it.
store, _ := metaengine.Plan(
    []metaengine.Engine{
        metaengine.NewMemoryEngine(),  // fast for vector/search (brute-force)
        sqliteEngine,                   // fast for filtered scans (indexed)
    },
    // ... queries ...
)
```

## What You Lose

- **Simpler API** — metaengine has more concepts (ADTs, planners, fold classification)
- **Backward compat** — existing `kv.ViewStore` consumers keep working; no forced migration
- **SQL power queries** — `SQLViewStore` supports full SQL (JOINs, subqueries); metaengine's `ScanBackend` is simpler

## When NOT to Migrate

- Your projection is a simple key-value lookup with no filtering
- You need JOINs across multiple tables (use `RelationalProjection` instead)
- You need the `stack.Materialize` tombstone lifecycle (metaengine doesn't have `OnTombstone`)
- Your team is unfamiliar with the planner concept

## See Also

- [Metaengine design docs](../planning/meta-engine-design.md)
- [ADR-0061: SQLite metaengine engine](../adr/0061-metaengine-sqlite-engine.md)
- [ADR-0085: New ADTs (Vector/Search/Spatial)](../adr/0085-metaengine-new-adts.md)
- [SKILL.md recipes](../../.agents/skills/go-cqrs-lite/references/recipes.md)
