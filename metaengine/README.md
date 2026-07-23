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
