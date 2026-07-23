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

| Handler Signature   | Return Type    | ADT        | Example                |
| ------------------- | -------------- | ---------- | ---------------------- |
| `func(e) (K, V)`    | `(Key, Value)` | Map        | Point lookup by key    |
| `func(e) K`         | `Key`          | Set        | Membership test        |
| `func(e) Delta`     | `Delta`        | Counter    | Aggregate counts       |
| `func(e) Edge`      | `Edge`         | Graph      | Traversal              |
| `func(e, prev V) V` | `Value`        | Map update | Read-modify-write      |
| `Remove[V]()`       | Sentinel       | Delete     | Remove from projection |
| `func(e) Skip`      | `Skip`         | No-op      | Event doesn't apply    |

## Typed Filter/Sort — No Strings

```go
listByStatus := metaengine.Query[ListByStatus, ListByStatusResult]("list_by_status",
    metaengine.On(UserCreated{}, func(e UserCreated) (UserID, FindUserResult) { ... }),
    metaengine.FilterOn(func(r FindUserResult) string { return r.Status }),
    metaengine.SortOn(func(r FindUserResult) time.Time { return r.JoinedAt }),
)
```

Pagination is detected from the domain input struct:

```go
type ListByStatus struct {
    Status string             // domain filter — matched by type to FilterOn closure
    Limit  int                // pagination — detected by field name + type
    After  *metaengine.Cursor // pagination — detected by type
}

type ListByStatusResult struct {
    Users []FindUserResult    // collection — detected by []T field shape
    Next  *metaengine.Cursor  // continuation cursor
}
```

## Per-Query Projections

Each query has its own independent projection. The same event updates each
matching query's projection separately. Write amplification is reported as
a diagnostic warning — it is the operator's choice, not the engine's.

```go
store, _ := metaengine.Plan(engines,
    findUser,      // Map<UserID, FindUserResult>
    checkEmail,    // Set<Email>
    countByStatus, // Counter
    friendsOf,     // Graph
    listByStatus,  // SortedMap (filtered scan)
)
// UserCreated event updates all 5 projections independently.
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

`MemoryEngine` implements all backends for testing/CI deployments.
