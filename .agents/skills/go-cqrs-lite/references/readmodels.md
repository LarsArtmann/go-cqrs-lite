## Read Models (projection + query) — CQRS read-side recipes

> **Contents:**
>
> - [SQL-backed views](#sql-backed-views-queryable-columns-server-side-filtering)
> - [Canonical projection pattern: CatchUpSubscriber + Materialize](#canonical-projection-pattern-catchupsubscriber--materialize)
> - [Choosing a projection tier: KV vs Relational vs Graph](#choosing-a-projection-tier-kv-vs-relational-vs-graph)

_Extracted from the former recipes §2.3. This is the most-asked-about topic in event-sourced systems — building queryable read models from your event stream._

### 2.3 Read Models (projection + query)

Projections rebuild queryable state from the event stream. There are **two ways to run them** — pick based on your delivery model:

**Option A — `projectionhost` (pull-based, crash-restart, no bus dependency).** Reads directly from an `event.SeekableJournal`. Best for batch replay and systems without a live event bus.

```go
import (
    "github.com/larsartmann/go-cqrs-lite/projection/v4"
    "github.com/larsartmann/go-cqrs-lite/projectionhost/v4"
)

// Define a projection implementing projection.Projection
type TodoProjection struct{ store ReadModel }
func (p *TodoProjection) Name() string { return "todo-read-model" }
func (p *TodoProjection) EventTypes() []event.Type {
    return []event.Type{"todo.created", "todo.updated"} // nil = all types
}
func (p *TodoProjection) Handle(ctx context.Context, evt event.Event) error {
    // mutate your read model
    return nil
}

// journal: any event.SeekableJournal (MemoryStore, SQLEventStore, pebble.EventStore, ...)
host, _ := projectionhost.New(journal, checkpointStore,
    projectionhost.WithBatchSize(100),
    projectionhost.WithDeadLetterStore(projectionhost.NewMemoryDeadLetterStore(), 3),
)
_ = host.Register(&TodoProjection{store: readModel})
go host.Start(ctx)   // one goroutine per projection; crash auto-restart + backoff
defer host.Stop()    // graceful drain (30s timeout)
// host drains the journal from each projection's checkpoint, then idles.
// For live push delivery, pair with watermill/CatchUpSubscriber (Option B).
```

**Option B — `CatchUpSubscriber` (push-based, live tail after replay).** Pairs with `stack.Materialize` for ordered, durable projections. See §2.3 "Canonical projection pattern" below and advanced.md §6.9 for the full `projectionhost` lifecycle.

Query the read model with type-safe dispatch:

```go
qDisp := query.NewDispatcher()
query.RegisterTyped(qDisp, "todo.get",
    func(ctx context.Context, q *GetTodoQuery) (*GetTodoResult, error) {
        return readModel.Get(q.ID)
    })

result, err := query.DispatchTyped[*GetTodoResult](ctx, qDisp, &GetTodoQuery{ID: id})
```

#### SQL-backed views (queryable columns, server-side filtering)

By default, `Materialize` stores views as opaque JSON blobs via `kv.TypedStore`.
For SQL backends (SQLite, Postgres), use `storage.SQLViewStore` to give each
view type its own table with **real, queryable SQL columns** — enabling WHERE,
ORDER BY, and LIMIT/OFFSET at the database level.

```go
import (
    "github.com/larsartmann/go-cqrs-lite/storage/v4"
    "github.com/larsartmann/go-cqrs-lite/stack/v4"
)

// 1. Define how your view maps to SQL columns.
mapper := storage.ViewMapper[TodoView]{
    Table: "todos_view",
    Columns: []storage.ViewColumn[TodoView]{
        {Name: "title", Type: "TEXT", Extract: func(v *TodoView) any { return v.Title }},
        {Name: "completed", Type: "INTEGER", Extract: func(v *TodoView) any { return v.Completed }},
        {Name: "tombstoned", Type: "INTEGER", Extract: func(v *TodoView) any { return v.Tombstoned }},
    },
    ScanRow: func(scan func(dest ...any) error) (*TodoView, error) {
        var v TodoView
        var completed, tombstoned int
        if err := scan(&v.Title, &completed, &tombstoned); err != nil {
            return nil, err
        }
        v.Completed = completed != 0
        v.Tombstoned = tombstoned != 0
        return &v, nil
    },
    TombstoneColumn: "tombstoned", // enables server-side tombstone filtering in List
}

// 2. Create the SQL-backed view store (auto-creates the table).
store, _ := storage.NewSQLiteViewStore[TodoView, id.AggregateID](db, mapper)

// 3. Use it with Materialize — same API as KV-backed.
mat := stack.Materialize[TodoView, id.AggregateID]{
    Store:        store,               // ← kv.ViewStore interface, SQL-backed
    KeyFromEvent: todoKey,
}
configureMaterialize(&mat)  // OnCreate, OnUpdate, OnTombstone — identical

// 4. Query with SQL power (the killer feature):
//    WHERE, ORDER BY, LIMIT/OFFSET — pushed to the database.
results, _ := store.Query(ctx, kv.ViewQuery{
    Where:   "completed = ?",
    Args:    []any{0},
    OrderBy: "title",
    Limit:   20,
})

// 5. List automatically uses server-side tombstone filtering when
//    TombstoneColumn is configured (no full-table load).
active, _ := mat.List(ctx, stack.ExcludeTombstoned)
```

Shortcut: auto-generate the mapper from struct tags:

```go
type TodoView struct {
    Title      string `view:"title"`
    Completed  bool   `view:"completed"`
    Tombstoned bool   `view:"tombstoned"`
}
mapper := storage.AutoMapperWithTombstone[TodoView]("todos_view", "tombstoned")
// Equivalent to the manual ViewMapper above — zero boilerplate.
```

From a Bundle preset (the one-call path):

```go
b, _ := sqlite.New("app.db")
store, _ := sqlite.SQLViewModel[TodoView, TodoID](b, mapper) // ← uses bundle's DB
mat := stack.Materialize[TodoView, TodoID]{Store: store, ...}
```

Advanced capabilities (all optional, checked at runtime):

```go
// Count without loading records (SELECT COUNT(*))
n, _ := store.Count(ctx, kv.ViewQuery{
    Conditions: []kv.Condition{{Column: "completed", Op: kv.OpEq, Value: false}},
})

// Batch upsert for projection replay throughput (chunks to respect SQLite 999-param limit)
_ = store.BatchSet(ctx, items)

// Structured (injection-safe) filtering — no raw SQL
results, _ := store.Query(ctx, kv.ViewQuery{
    Conditions: []kv.Condition{
        {Column: "completed", Op: kv.OpEq, Value: false},
        {Column: "title", Op: kv.OpLike, Value: "%urgent%"},
    },
    OrderBy: "title", Limit: 10})

// Projection reset (DELETE FROM table)
_ = store.DeleteAll(ctx)

// Secondary indexes for fast lookups (set on mapper)
mapper.Indexes = []storage.IndexSpec{
    {Name: "idx_title", Columns: []string{"title"}},
}
```

#### Canonical projection pattern: CatchUpSubscriber + Materialize

For ordered, durable projections, use `CatchUpSubscriber` — NOT the Watermill
Router directly. The Router processes messages in parallel (one goroutine per
message), which breaks ordering for projections that need FIFO guarantees.

The canonical pattern (see `example/taskmanager`):

```go
b, _ := sqlite.New("app.db")
defer b.Close()

// 1. Create the CatchUpSubscriber from the Bundle.
catchUp, _ := b.CatchUpSubscriber()
defer catchUp.Close()

// 2. Create a SQL-backed or KV-backed Materialize.
store, _ := sqlite.SQLViewModel[TodoView, TodoID](b, mapper)
mat := stack.Materialize[TodoView, TodoID]{Store: store, ...}

// 3. Subscribe to a topic — CatchUpSubscriber replays from the journal
//    (Phase 1, ModeReplay), then hands off to live (Phase 2).
msgs, _ := catchUp.Subscribe(ctx, "todo.created")

// 4. Consume from a SINGLE goroutine — FIFO ordering guaranteed.
go func() {
    for msg := range msgs {
        _ = mat.HandlerFunc()(msg)
        msg.Ack()
    }
}()
```

**Why not the Router?** The Router spawns one goroutine per message
(`message/router.go:30`). For ordered projections, consume the
CatchUpSubscriber's output channel from a single goroutine instead. The
EventBus default uses `BlockPublishUntilSubscriberAck=true` for ordered live
delivery and `Persistent=false` to avoid GoChannel's unordered persistent
replay (the CatchUpSubscriber handles replay from the journal instead).

#### Choosing a projection tier: KV vs Relational vs Graph

There are **three projection tiers** — pick by read-access pattern:

| Tier            | Module                               | Writes ONE event to…       | Best for                                          |
| --------------- | ------------------------------------ | -------------------------- | ------------------------------------------------- |
| **Document/KV** | `stack.Materialize` + `kv.ViewStore` | one record in one table    | single-entity lookups, CRUD-style reads           |
| **Relational**  | `storage.RelationalProjection`       | several related SQL tables | multi-table joins, WHERE/ORDER BY, set predicates |
| **Graph**       | `graph.GraphProjection`              | nodes + edges              | variable-depth traversal, path-finding, adjacency |

`SQLViewStore` (above) is the document tier with queryable columns — still one
record per event. When a single event must update several related tables
atomically (a message + its attachments[] + a member_roles junction), use
`RelationalProjection` instead. For N-hop queries, use `GraphProjection` (see advanced.md §6.13).

```go
import "github.com/larsartmann/go-cqrs-lite/storage/v4"

// Relational: one event → many tables, atomic, dialect-agnostic.
schema := storage.RelationalSchema{Tables: []storage.RelationalTable{
    {Name: "messages", PrimaryKey: []string{"id"}, Columns: []storage.RelationalColumn{
        {Name: "id", Type: "TEXT"}, {Name: "channel_id", Type: "TEXT"}, {Name: "content", Type: "TEXT"},
    }},
    {Name: "attachments", PrimaryKey: []string{"id"}, Columns: []storage.RelationalColumn{
        {Name: "id", Type: "TEXT"}, {Name: "message_id", Type: "TEXT"}, {Name: "filename", Type: "TEXT"},
    }},
}}
proj, _ := storage.NewRelationalProjection("messages", schema, db, sqlpkg.SQLiteDialect{},
    func(ctx context.Context, evt event.Event, sink storage.ProjectionSink) error {
        var p MessageCreated
        _ = json.Unmarshal(evt.Payload(), &p)
        sink.Upsert(ctx, "messages", storage.Row{"id": p.ID, "channel_id": p.ChannelID, "content": p.Content})
        for _, a := range p.Attachments {
            sink.Ensure(ctx, "attachments", storage.Row{"id": a.ID, "message_id": p.ID, "filename": a.Name})
        }
        return nil // all writes commit atomically; error → full rollback
    }, []event.Type{"MESSAGE_CREATED"})
// proj implements projection.Projection → register with projectionhost or CatchUpSubscriber.
// SQL-ONLY (SQLite/Postgres). For KV backends use stack.Materialize; for graph see advanced.md §6.13.
```
