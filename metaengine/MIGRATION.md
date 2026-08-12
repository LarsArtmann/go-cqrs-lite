# Migration Guide: kv.ViewStore/Materialize to Metaengine

This guide walks you through migrating a read model from `stack.Materialize` +
`kv.ViewStore` to the metaengine cost-based query planner.

## When to Migrate

Migrate when you hit **any** of these signals:

| Signal                              | Materialize Limitation                 | Metaengine Solution                                                             |
| ----------------------------------- | -------------------------------------- | ------------------------------------------------------------------------------- |
| Filtered scans are slow at scale    | O(N) Go-side filter on every List call | `FilterOnField` → SQLite `json_extract` WHERE pushdown (50x faster at 10K rows) |
| You need sorted results             | Sort in Go after loading all records   | `SortOnField` → SQLite ORDER BY pushdown                                        |
| You need O(1) aggregates            | Count by scanning all records          | Counter ADT with Delta folds                                                    |
| Multiple read patterns on same data | One projection per pattern             | One Map query, multiple fold handlers                                           |
| Point lookups dominate              | kv.Get is already fast                 | Memory engine for O(1) or SQLite for persistence                                |

**Don't migrate** if your read model is small (< 1K records) or you only do
simple key-value lookups. Materialize + kv.ViewStore is simpler and sufficient.

## Step-by-Step Migration

### Step 1: Declare Queries

Define your query types and fold handlers. Each query is a `metaengine.Query[I, V]`
declaration:

```go
type listInput struct {
    Status string
}

type itemView struct {
    ID       string
    Title    string
    Status   string
    Priority int
}

query := metaengine.Query[listInput, itemView]("items",
    // Insert fold: creates a new entry keyed by ID
    metaengine.On(itemCreated{}, func(e itemCreated) (string, itemView) {
        return e.ID, itemView{ID: e.ID, Title: e.Title, Status: "open"}
    }),
    // Update fold: modifies existing entry
    metaengine.On(itemCompleted{}, func(e itemCompleted, prev itemView) itemView {
        prev.Status = "completed"
        return prev
    }),
    // Remove fold: deletes entry
    metaengine.On(itemDeleted{}, metaengine.Remove[itemView]()),
    // Declarative filter for SQLite pushdown
    metaengine.FilterOnField[itemView]("status", metaengine.FilterEq),
    // Declarative sort for SQLite pushdown
    metaengine.SortOnField[itemView]("priority", true), // DESC
)
```

### Step 2: Write an EventDecoder

The EventDecoder bridges CQRS events to metaengine fold values. For Map queries
that key on the stream ID, wrap the payload with the ID:

```go
type eventWithID[P any] struct {
    ID      string
    Payload P
}

func myEventDecoder(evt event.Event) (any, error) {
    id := evt.StreamID().String()
    switch evt.Type() {
    case "item.created":
        var p itemCreated
        if err := json.Unmarshal(evt.Payload(), &p); err != nil {
            return nil, err
        }
        return eventWithID[itemCreated]{ID: id, Payload: p}, nil
    // ... other event types
    default:
        return nil, fmt.Errorf("no fold for %q", evt.Type())
    }
}
```

### Step 3: Plan and Wire

Create a Store with engines, plan the queries, and create the adapter:

```go
meDB, _ := sql.Open("sqlite", dsn)
meDB.SetMaxOpenConns(1)
meDB.Exec("PRAGMA journal_mode=WAL")
meDB.Exec("PRAGMA busy_timeout=5000")

sqliteEng, _ := metaengine.NewSQLiteEngine(meDB)

store, _ := metaengine.Plan(
    []metaengine.Engine{metaengine.NewMemoryEngine(), sqliteEng},
    query,
)

adapter := projectionadapter.New("items", store, nil,
    projectionadapter.WithEventDecoder(myEventDecoder))

reader := metaengine.NewReader[itemView](store, "items")
```

### Step 4: Register with Projection Host

Replace the Materialize projection registration:

```go
// OLD: host.Register(mat)
// NEW:
host.Register(adapter)
```

### Step 5: Switch Handlers

Replace kv.ViewStore reads with TypedReader:

```go
// OLD:
// items, _ := mat.List(ctx)
// filtered := filterByStatus(items, "open")

// NEW:
items, _ := reader.Scan(ctx,
    metaengine.WithFilter("status", metaengine.FilterEq, "open"),
    metaengine.WithSort("priority", true),
    metaengine.WithLimit(50),
)
```

### Step 6: Parallel Run (Optional)

During migration, run both projections in parallel to verify correctness:

```go
host.Register(mat)      // old projection (for verification)
host.Register(adapter)  // new metaengine projection
```

Compare results from both. Once verified, remove the old projection.

### Step 7: Cutover

Remove the Materialize projection and kv.ViewStore. Update handlers to use
the TypedReader exclusively.

## Common Patterns

### Counter Query (O(1) Aggregates)

```go
counterQ := metaengine.Query[countInput, map[string]int64]("counts",
    metaengine.On(itemCreated{}, func(e itemCreated) metaengine.Delta {
        return metaengine.Delta{e.Status: +1}
    }),
    metaengine.On(itemCompleted{}, func(e itemCompleted) metaengine.Delta {
        return metaengine.Delta{"open": -1, "completed": +1}
    }),
)
```

### Multi-Engine Distribution

The planner assigns each query to the cheapest engine that supports its
operations. Counters go to Memory (O(1) reads), filtered Maps go to SQLite
(pushdown scans):

```go
store, _ := metaengine.Plan(
    []metaengine.Engine{metaengine.NewMemoryEngine(), sqliteEng},
    counterQ, mapQ,  // planner picks the best engine for each
)
```
