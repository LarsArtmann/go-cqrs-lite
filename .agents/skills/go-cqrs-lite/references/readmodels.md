## Read Models (projection + query) — CQRS read-side recipes

> **Contents:**
>
> - [SQL-backed views](#sql-backed-views-queryable-columns-server-side-filtering)
> - [Canonical projection pattern: CatchUpSubscriber + Materialize](#canonical-projection-pattern-catchupsubscriber--materialize)
> - [Choosing a projection tier: KV vs Relational vs Graph](#choosing-a-projection-tier-kv-vs-relational-vs-graph)

_Extracted from the former recipes §2.3. This is the most-asked-about topic in event-sourced systems — building queryable read models from your event stream._

> **v5 deprecation notice (ADR-0123):** the v1 read-model tiers documented
> here — `stack.Materialize` (+ `storage.SQLViewStore`),
> `storage.RelationalProjection`, and `graph.GraphProjection` — are
> **deprecated and removed in v5**. New code should prefer the `metaengine`
> Store + `projectionadapter` (see `recipes.md` §2.11) and the `system`
> composition root. Everything below remains fully functional through v4.x;
> `projectionhost` (Option A) is the projection runner that survives v5.

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

**Checkpoint tuning (live-phase throughput vs. reprocessing window).** By default
`projectionhost` saves the checkpoint after **every** live event. If your checkpoint
store is slow (remote SQL, per-write fsync), two opt-in knobs batch those saves:

```go
projectionhost.WithCheckpointEvery(100),      // save after every 100 live events
projectionhost.WithCheckpointInterval(500 * time.Millisecond), // or/and: next event after 500ms idle
```

Both default to off (save-per-event). On crash or `Stop()`, pending checkpoints are
flushed; only a hard crash can lose progress, in which case **at most n−1 live events
are reprocessed on restart** — the same at-least-once contract as the replay→live
overlap. Catch-up (drain) phase always saves per batch, independent of these knobs.

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
store, _ := storage.NewSQLiteViewStore[TodoView, id.StreamID](db, mapper)

// 3. Use it with Materialize — same API as KV-backed.
mat := stack.Materialize[TodoView, id.StreamID]{
    Store:        store,               // ← kv.ViewStore interface, SQL-backed
    KeyFromEvent: todoKey,
}
configureMaterialize(&mat)  // OnCreate, OnUpdate, OnTombstone — identical

// 4. Query with SQL power (the killer feature):
//    WHERE, ORDER BY, pagination — pushed to the database.
//    Conditions are VALIDATED fail-closed: columns must be declared mapper
//    columns and operators must be in the supported set (=, !=, <, <=, >, >=,
//    LIKE, IN, IS NULL, IS NOT NULL) — request-derived names can never reach
//    the SQL string.
results, _ := store.Query(ctx, kv.ViewQuery{
    Conditions: []kv.Condition{{Column: "completed", Op: kv.OpEq, Value: 0}},
    OrderBy:    "title",
    Limit:      20,
})

// 5. List automatically uses server-side tombstone filtering when
//    TombstoneColumn is configured (no full-table load).
active, _ := mat.List(ctx, stack.ExcludeTombstoned)
```

#### Validated WHERE, keyset pagination, and the RawWhere escape hatch

`SQLViewStore.Query` (implements `kv.ViewQuerier`) validates every query
fail-closed before SQL is rendered (2026-08-15 SQL-injection guards):

- **Columns** must be declared mapper columns — an unknown column is a
  `Rejection` error, never interpolated SQL. Covers filter conditions,
  `Order`/`OrderBy`, and keyset cursor columns.
- **Operators** must be one of the 10 `kv.Operator` constants
  (`OpEq`, `OpNeq`, `OpLt`, `OpLte`, `OpGt`, `OpGte`, `OpLike`, `OpIn`,
  `OpIsNull`, `OpIsNotNull`).
- **Values are always bound parameters** — never spliced into the string.
- **`RawWhere` + `RawArgs`** is the explicit escape hatch for what
  `Condition` cannot express (OR groups, subqueries, date arithmetic):
  it is AND-joined after the validated conditions, and YOU own
  parameterisation of the raw fragment.

```go
q := kv.ViewQuery{
    Conditions: []kv.Condition{
        {Column: "guild_id", Op: kv.OpEq, Value: gid},
        {Column: "status", Op: kv.OpIn, Values: []any{"active", "pending"}},
    },
    RawWhere: "(deleted_at IS NULL OR deleted_at > ?)",
    RawArgs:  []any{cutoff},
}
```

For deep pagination, prefer keyset (seek) pagination over `Offset` —
performance is constant regardless of depth, and concurrent inserts do not
shift the window. Cursor columns default to the effective `Order` with the
key column appended as tiebreaker:

```go
page1, _ := store.Query(ctx, kv.ViewQuery{
    Order: []kv.OrderClause{{Column: "created_at", Desc: true}},
    Limit: 50,
})
last := page1[len(page1)-1]
page2, _ := store.Query(ctx, kv.ViewQuery{
    Order:  []kv.OrderClause{{Column: "created_at", Desc: true}},
    Keyset: &kv.Keyset{Columns: []string{"created_at", "id"},
        Values: []any{last.CreatedAt, last.ID}},
    Limit: 50,
})
```

Note for hand-rolled SQL: `storage/sql.BuildWhereClause` is deprecated
(interpolates column names/operators); `BuildWhereClauseChecked` is the
validated replacement.

Schema ownership: all view-store constructors auto-create the table and
indexes unless you pass `storage.WithoutViewAutoMigrate()`. Use it when your
migration tooling (goose, golang-migrate, embedded `storage/migrations` DDL)
owns the DDL — the store then assumes the table exists and never issues
`CREATE TABLE`:

```go
store, _ := storage.NewSQLiteViewStore[TodoView, id.StreamID](
    db, mapper, storage.WithoutViewAutoMigrate())
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
