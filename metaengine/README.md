# metaengine

> **Prototype.** Cost-based storage optimizer for event-sourced data.
> Derives projections, indexes, and engine assignments from two primitives:
> **Events** (mutations) and **Queries** (read intent).

## What This Proves

The developer writes **only** event types, query types, and fold functions.
The planner automatically:

- Infers the ADT (Map, Set, Counter, Graph, SortedMap) from fold return types
- Infers filter/sort criteria from query input fields matched to result type fields
- Detects pagination from `Page[T]` result types
- Assigns each query to the cheapest engine that supports its ADT
- Emits degradation warnings (e.g. "O(N) scan — add SQLite for O(logN)")

## Quick Example

```go
// 1. Events (pure domain types)
type UserCreated struct { ID UserID; Name string; At time.Time }
type UserDeleted struct { ID UserID }

// 2. Query types (input + result — zero storage knowledge)
type FindUser struct { ID UserID }
type FindUserResult struct {
    ID   UserID
    Name string
    JoinedAt time.Time `metaengine:"sort"`
}

// 3. Query declaration (fold functions only)
findUser := metaengine.Query[FindUser, FindUserResult]("find_user", []metaengine.Fold{
    metaengine.OnInsert(UserCreated{}, func(e UserCreated) (UserID, FindUserResult) {
        return e.ID, FindUserResult{ID: e.ID, Name: e.Name, JoinedAt: e.At}
    }),
    metaengine.OnRemove[UserDeleted, UserID, FindUserResult](UserDeleted{},
        func(e UserDeleted) UserID { return e.ID }),
})

// 4. Plan — the optimizer assigns engines
store, _ := metaengine.Plan([]metaengine.Engine{metaengine.NewMemoryEngine()}, findUser)
defer store.Close()

// 5. Apply events + execute queries
store.Apply("UserCreated", UserCreated{ID: "u1", Name: "Alice", At: time.Now()})

result, _ := metaengine.ExecuteTyped[FindUser, FindUserResult](
    context.Background(), store, FindUser{ID: "u1"})
// → FindUserResult{ID: "u1", Name: "Alice", ...}
```

## The 5 ADTs

| Fold Function                 | Return Type | ADT       | Example Query                 |
| ----------------------------- | ----------- | --------- | ----------------------------- |
| `OnInsert`                    | `(K, V)`    | Map       | Point lookup by key           |
| `OnSet`                       | `K`         | Set       | Membership test               |
| `OnCount`                     | `Delta`     | Counter   | Aggregate counts              |
| `OnEdge`                      | `Edge`      | Graph     | Traversal                     |
| `OnInsert` + `Page[T]` result | `(K, V)`    | SortedMap | Filtered scan with pagination |

## Key Design Decisions

1. **Pure type inference** — no codegen, no DSL, no FilterOn/SortOn declarations.
   The planner reflects on Go types at startup.
2. **`Page[T]` result** — pagination is a mechanical concern, not domain intent.
   Query inputs carry only domain fields.
3. **ISP-split backends** — `MapBackend`, `SetBackend`, `CounterBackend`,
   `GraphBackend`, `ScanBackend`. Engines implement only what they support.
4. **Typed key extractors** — `OnUpdate` and `OnRemove` take explicit key functions,
   not first-field convention.
5. **`TypedDelta[K ~string]`** — counter keys can be typed string aliases.
