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
