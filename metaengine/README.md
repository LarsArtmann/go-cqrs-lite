# metaengine

> Cost-based storage planner for event-sourced data.
> Derives projections, indexes, and engine assignments from two primitives:
> **Events** (mutations) and **Queries** (read intent).

## Three Roles

| Role            | Provides                                                            | Receives                         |
| --------------- | ------------------------------------------------------------------- | -------------------------------- |
| **Developer**   | Event types, Query types, Fold functions                            | `Store` to Apply/Execute against |
| **Operator**    | Engines (Memory, SQL, Pebble) with cost profiles                    | `PlanResult` showing assignments |
| **Meta-Engine** | Derives everything from the relationship between events and queries | The bridge                       |

## Quick Example

```go
// 1. Events (pure domain types)
type UserCreated struct { ID UserID; Name string; At time.Time }
type UserDeleted struct { ID UserID }

// 2. Query types (input + result)
type FindUser struct { ID UserID }
type FindUserResult struct { ID UserID; Name string; JoinedAt time.Time }

// 3. Query declaration — folds define how events update this query's projection
findUser := metaengine.Query[FindUser, FindUserResult]("find_user",
    metaengine.On(UserCreated{}, func(e UserCreated) (UserID, FindUserResult) {
        return e.ID, FindUserResult{ID: e.ID, Name: e.Name, JoinedAt: e.At}
    }),
    metaengine.On(UserDeleted{}, metaengine.Remove[FindUserResult]()),
)

// 4. Plan — the optimizer assigns engines to queries
store, _ := metaengine.Plan([]metaengine.Engine{metaengine.NewMemoryEngine()}, findUser)
defer store.Close()

// 5. Apply events + execute queries
store.Apply("UserCreated", UserCreated{ID: "u1", Name: "Alice", At: time.Now()})

result, _ := metaengine.ExecuteTyped[FindUser, FindUserResult](
    context.Background(), store, FindUser{ID: "u1"})
// → FindUserResult{ID: "u1", Name: "Alice", ...}
```

## The Fold Return Type IS the ADT

The developer never declares "I need a Map" or "I need a Counter."
The fold function's return type IS the declaration:

| Handler Signature    | Return Type    | ADT        | Example                |
| -------------------- | -------------- | ---------- | ---------------------- |
| `func(e) (K, V)`     | `(Key, Value)` | Map        | Point lookup by key    |
| `func(e) K`          | `Key`          | Set        | Membership test        |
| `func(e) Delta`      | `Delta`        | Counter    | Aggregate counts       |
| `func(e) Edge`       | `Edge`         | Graph      | Traversal              |
| `func(e) MultiEntry` | `MultiEntry`   | Multimap   | One key, many values   |
| `func(e) Append`     | `Append`       | Log        | Append-only timeline   |
| `func(e, prev V) V`  | `Value`        | Map update | Read-modify-write      |
| `Remove[V]()`        | Sentinel       | Delete     | Remove from projection |
| `func(e) Skip`       | `Skip`         | No-op      | Event doesn't apply    |

## Typed Filter/Sort — No Strings

```go
listByStatus := metaengine.Query[ListByStatus, ListByStatusResult]("list_by_status",
    metaengine.On(UserCreated{}, func(e UserCreated) (UserID, FindUserResult) { ... }),
    metaengine.FilterOn(func(r FindUserResult) string { return r.Status }),
    metaengine.SortOn(func(r FindUserResult) time.Time { return r.JoinedAt }),
)
```

## Pagination + Cursor Serialization

Pagination is detected from the domain input struct. Cursors serialize to
URL-safe base64 strings for HTTP transport:

```go
type ListByStatus struct {
    Status string
    Limit  int
    After  *metaengine.Cursor
}

// Page 1
page1, _ := metaengine.ExecuteTyped[ListByStatus, ListByStatusResult](
    ctx, store, ListByStatus{Status: "active", Limit: 10})

// Serialize cursor for HTTP
cursorStr := page1.Next.String()

// Deserialize on next request
cursor, _ := metaengine.ParseCursor(cursorStr)
page2, _ := metaengine.ExecuteTyped[ListByStatus, ListByStatusResult](
    ctx, store, ListByStatus{Status: "active", Limit: 10, After: cursor})
```

## Cost Model

The planner estimates cost for each query using a formal cost function based on
complexity x volume. Volume hints and latency budgets drive engine selection:

```go
store, _ := metaengine.Plan(engines,
    metaengine.Query[FindUser, FindUserResult]("find_user",
        folds...,
        metaengine.Volume(1_000_000),       // expected items
        metaengine.WithLatencyBudget(5),     // max 5ms
    ),
)

plan := store.Plan()
// plan.Queries[0].Cost.EstimatedLatencyMs -> estimated read latency
// plan.Queries[0].Cost.WithinBudget(5)    -> true/false
```

### Scale Threshold Warnings

When volume exceeds the optimal range for a data structure, the planner warns:

```go
// Volume 50M exceeds hash map optimal range (max 10M)
// -> WARN: "volume 50000000 exceeds optimal range for hash map"
```

### Write Amplification Budget

```go
store, _ := metaengine.Plan(engines, q1, q2, q3, q4,
    metaengine.WithWriteAmplificationBudget(5),  // default: 3
)
```

## Per-Query Projections

Each query has its own independent projection. The same event updates each
matching query's projection separately.

```go
store, _ := metaengine.Plan(engines,
    findUser,      // Map<UserID, FindUserResult>
    checkEmail,    // Set<Email>
    countByStatus, // Counter
    friendsOf,     // Graph
    listByStatus,  // SortedMap (filtered scan)
    tasksForUser,  // Multimap<UserID, []TaskID>
    recentTasks,   // Log<TaskCreated>
)
```

## JSON Event Payloads

```go
store.ApplyEncoded(string(evt.Type()), evt.Payload())
```

## Projection Adapter

```go
type projectionAdapter struct{ store *metaengine.Store }

func (p *projectionAdapter) Handle(_ context.Context, evt event.Event) error {
    return p.store.ApplyEncoded(string(evt.Type()), evt.Payload())
}
```

## Engine Interface

Engines implement whichever ADT backends they support:

```go
type Engine interface {
    Profile() EngineProfile  // declares supported ADTs + complexity
    Close() error
}
```

The memory engine (constructed via `NewMemoryEngine()`) implements all backends
for testing and CI deployments.

## SQLite Engine

The SQLite engine persists projections to a `database/sql` database, enabling
restart-safety and multi-process reads. Pass both engines to `Plan` and the
cost model assigns each query to the cheaper one (memory for point lookups,
SQLite for persistence):

```go
import (
    "database/sql"
    "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
    _ "modernc.org/sqlite"
)

db, _ := sql.Open("sqlite", "file:app.db")
defer db.Close()

sqliteEng, _ := metaengine.NewSQLiteEngine(db)
// The caller owns the *sql.DB — sqliteEng.Close() is a no-op.
store, _ := metaengine.Plan(
    []metaengine.Engine{metaengine.NewMemoryEngine(), sqliteEng},
    findUser,
)
defer store.Close()
```

The SQLite engine wraps `MapUpdate` in a transaction (concurrent
read-modify-write never loses updates — see
[ADR-0067](../docs/adr/0067-metaengine-tx-mapupdate.md)) and seeds the multimap
`seq` counter from `MAX(seq)` on first use so it survives restarts
([ADR-0068](../docs/adr/0068-metaengine-multimap-seq-seed.md)). Struct results
read back through `ExecuteTyped` are JSON-reified across the SQL boundary
([ADR-0066](../docs/adr/0066-metaengine-reify-fallback.md)) — use exported
fields on result types.

## Build Tag (Portability)

This module is built with the `goexperiment.jsonv2` build tag (Go 1.26+), which
enables `encoding/json/v2`. Consumers on stock Go 1.26 must build with
`-tags goexperiment.jsonv2`; the tag graduates to default in Go 1.27+. CI and
`nix run .#build` apply the tag automatically.

## ApplyEncoded (Hot Path for Projections)

`Store.ApplyEncoded` is the zero-copy entry point used by `projectionadapter`
when wiring a metaengine Store into a `projectionhost.Host`. It accepts a
JSON-encoded `[]byte` payload directly (no need to unmarshal first), decodes
it via `encoding/json/v2`, and routes it to all matching fold handlers:

```go
// In a projection.Projection.Handle implementation:
func (p *myProjection) Handle(ctx context.Context, evt event.Event) error {
    return p.store.ApplyEncoded(string(evt.Type()), evt.Payload())
}
```

This avoids a double-decode (the adapter would otherwise unmarshal to `any`,
then `Apply` would need to re-inspect the type). The `projectionadapter`
package is the canonical consumer; direct callers should use `Apply` with a
typed value unless they are bridging from a raw-payload source.

## Typed Reads — TypedReader and QueryBuilder

`TypedReader[V]` provides typed point-lookup (`Get`) and filtered scan (`Scan`)
without constructing a query input struct. `QueryBuilder[V]` adds a fluent,
chainable API on top:

```go
reader := metaengine.NewReader[TaskView](store, "task_views")

// Point lookup by key.
view, found, err := reader.Get(ctx, taskID)

// Filtered scan with options.
tasks, err := reader.Scan(ctx,
    metaengine.WithFilter("status", metaengine.FilterEq, "active"),
    metaengine.WithSort("priority", true),
    metaengine.WithLimit(50),
)

// Fluent builder — same result, reads top-to-bottom.
tasks, err := metaengine.NewQueryBuilder(reader).
    Where("status", metaengine.FilterEq, "active").
    SortBy("priority", true).
    Limit(50).
    Execute(ctx)
```

Available scan options: `WithFilter`, `WithRange`, `WithIn`, `WithOr`,
`WithSort`, `WithSortColumns`, `WithLimit`, `WithCursor`.

## Declarative Filter/Sort Pushdown (FilterOnField / SortOnField)

`FilterOn` and `SortOn` use typed closures (in-Go evaluation). For SQL-aware
engines (SQLite, Pebble), use `FilterOnField` and `SortOnField` instead — they
produce declarative specs that the engine pushes down to `json_extract()`
WHERE/ORDER BY clauses, achieving O(logN) instead of O(N):

```go
listTasks := metaengine.Query[ListTasks, TaskView]("task_views",
    metaengine.On(TaskCreated{}, func(e TaskCreated) (TaskID, TaskView) { ... }),
    metaengine.On(TaskCompleted{}, func(e TaskCompleted, prev TaskView) TaskView { ... }),
    metaengine.On(TaskDeleted{}, metaengine.Remove[TaskView]()),
    metaengine.FilterOnField[TaskView]("status", metaengine.FilterEq),
    metaengine.SortOnField[TaskView]("priority", true),
)
```

The planner sees the FilterOnField spec and assigns the query to the SQLite
engine (O(logN) pushdown) over Memory (O(N) closure scan).

## Watcher — Reactive Reads

`Watcher[V]` provides push notifications when a collection's values change.
Subscribe to all keys or a specific key:

```go
watcher := metaengine.NewWatcher[TaskView](store, "task_views")
ch := watcher.Watch(ctx, taskID)  // key-specific
// ch := watcher.Watch(ctx, nil)  // all keys

for v := range ch {
    fmt.Printf("task changed: %+v\n", v)
}
```

## ServeSSE — HTTP Streaming of Query Results

`ServeSSE` streams collection mutations to HTTP clients via Server-Sent Events.
This is the read-model push layer — clients see materialized query changes,
not raw domain events:

```go
watcher := metaengine.NewWatcher[TaskView](store, "task_views")
http.HandleFunc("/tasks/stream", metaengine.ServeSSE(watcher))
```

For raw domain event streaming (bus-to-client), use
`transport/http.SSEBroker` instead. See
[ADR-0079](../docs/adr/0079-sse-consolidation.md) for the rationale.

## Optional Engine Interfaces

Engines implement whichever ADT backends they support (`MapBackend`,
`SetBackend`, `CounterBackend`, `GraphBackend`, `MultimapBackend`,
`LogBackend`). Additionally, engines can implement these optional capability
interfaces for optimized read paths:

| Interface        | Method            | Benefit                               |
| ---------------- | ----------------- | ------------------------------------- |
| `PushdownScan`   | `PushdownMapScan` | SQL WHERE/ORDER BY/LIMIT pushdown     |
| `RawValueReader` | `GetRawValue`     | Single-pass JSON decode on Get        |
| `RawScanReader`  | `ScanRawValues`   | Single-pass JSON decode per scan row  |
| `MapUpdater`     | `MapUpdate`       | Atomic read-modify-write              |
| `Transactional`  | `RunInTx`         | Cross-collection transactional writes |

The SQLite engine implements all of these. The Memory engine implements
`MapUpdater` but not the pushdown/raw interfaces (it returns decoded Go
values directly).

## Projection Adapter with EventDecoder

The `projectionadapter` package wraps a Store as a `projection.Projection`
for registration with `projectionhost.Host`. For Map ADT queries that need
the entity ID (stream ID) as the projection key, use `WithEventDecoder`:

```go
adapter := projectionadapter.New("tasks", store, nil,
    projectionadapter.WithEventDecoder(func(evt event.Event) (any, error) {
        return eventWithID{
            ID:      evt.StreamID().String(),
            Payload: decodePayload(evt),
        }, nil
    }))
host.Register(adapter)
```

## TieredStore (Read Replicas)

`TieredStore` wraps a primary Store with optional replica Stores. Writes
(`Apply`/`ApplyBatch`) fan out to all stores; reads use the primary exclusively.

Use for read-scale-out or warm-standby scenarios:

```go
primary, _ := metaengine.Plan([]metaengine.Engine{sqliteEng}, query)
replica, _ := metaengine.Plan([]metaengine.Engine{memEng}, query)
tiered := metaengine.NewTieredStore(primary, replica)

// Writes go to both stores:
_ = tiered.Apply(ctx, "task.created", payload)

// Reads use the primary:
reader := metaengine.NewReader[TaskView](primary, "tasks")
```

## SwapEngine (Live Engine Migration)

`SwapEngine` replaces an engine at runtime. All queries assigned to the old
engine are reassigned to the new one. The old engine is NOT closed (caller
manages lifecycle).

Use for zero-downtime engine upgrades (e.g., swap Memory for SQLite after
warmup):

```go
// After warming up a SQLite engine with historical data:
err := store.SwapEngine("memory", "sqlite", sqliteEng)
// All queries previously on "memory" now route to "sqlite"
```

## QueryBuilder (Fluent API)

`QueryBuilder` provides a fluent builder on top of `TypedReader`:

```go
reader := metaengine.NewReader[TaskView](store, "tasks")
qb := metaengine.NewQueryBuilder[TaskView](reader)

results, _ := qb.
    Where("status", metaengine.FilterEq, "active").
    OrderBy("priority", true).
    Limit(50).
    Execute(ctx)
```
