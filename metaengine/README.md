# metaengine

> Cost-based storage optimizer for event-sourced data.
> Derives projections, indexes, and engine assignments from two primitives:
> **Events** (mutations) and **Queries** (read intent).

## What This Does

The developer writes **only** ReadModels (event-to-projection folds) and Queries
(input/result types). The planner automatically:

- Infers the ADT (Map, Set, Counter, Graph, SortedMap) from fold return types
- Infers filter/sort criteria from query input fields matched to result type fields
- Detects pagination from `Page[T]` result types
- Assigns each ReadModel to the cheapest engine that supports its ADT
- Deduplicates models shared across queries (no write amplification)
- Emits degradation warnings (e.g. "O(N) scan — add SQLite for O(logN)")

## Quick Example

```go
// 1. Events (pure domain types)
type UserCreated struct { ID UserID; Name string; At time.Time }
type UserDeleted struct { ID UserID }

// 2. Query types (input + result — zero storage knowledge)
type FindUser struct { ID UserID `metaengine:"key"` }
type FindUserResult struct {
    ID   UserID
    Name string
    JoinedAt time.Time `metaengine:"sort"`
}

// 3. ReadModel (write side: how events update the projection)
users := metaengine.MustModel("users",
    metaengine.OnInsert(UserCreated{}, func(e UserCreated) (UserID, FindUserResult) {
        return e.ID, FindUserResult{ID: e.ID, Name: e.Name, JoinedAt: e.At}
    }),
    metaengine.OnRemove(UserDeleted{},
        func(e UserDeleted) UserID { return e.ID }),
)

// 4. Queries (read side: how to read from the model)
findUser := metaengine.Query[FindUser, FindUserResult]("find_user", users)
listUsers := metaengine.Query[ListUsers, metaengine.Page[FindUserResult]]("list_users", users)

// 5. Plan — the optimizer assigns engines to models
store, _ := metaengine.Plan([]metaengine.Engine{metaengine.NewMemoryEngine()}, findUser, listUsers)
defer store.Close()

// 6. Apply events + execute queries
store.Apply("UserCreated", UserCreated{ID: "u1", Name: "Alice", At: time.Now()})

result, _ := metaengine.ExecuteTyped[FindUser, FindUserResult](
    context.Background(), store, FindUser{ID: "u1"})
// → FindUserResult{ID: "u1", Name: "Alice", ...}
```

## ReadModel + Query Separation

The core architectural insight: **the write side (how events update a projection)
is separate from the read side (how queries read it).**

- **ReadModel** owns the folds and ADT. Multiple queries can read from the same model.
- **Query** declares input/result types and references a ReadModel.
- One event updates each model exactly once, regardless of how many queries read it.

```go
// One model, two queries — no fold duplication, no write amplification.
users := metaengine.MustModel("users", folds...)
findUser   := metaengine.Query[FindUser, FindUserResult]("find_user", users)
listActive := metaengine.Query[ListActive, metaengine.Page[FindUserResult]]("list_active", users)
```

## The 5 ADTs

| Fold Function                 | Return Type | ADT       | Example Query                 |
| ----------------------------- | ----------- | --------- | ----------------------------- |
| `OnInsert`                    | `(K, V)`    | Map       | Point lookup by key           |
| `OnSet`                       | `K`         | Set       | Membership test               |
| `OnCount`                     | `Delta`     | Counter   | Aggregate counts              |
| `OnEdge`                      | `Edge`      | Graph     | Traversal                     |
| `OnInsert` + `Page[T]` result | `(K, V)`    | SortedMap | Filtered scan with pagination |

## Struct Tags

| Tag                 | Purpose                                  |
| ------------------- | ---------------------------------------- |
| `metaengine:"key"`  | Marks the key field in a query input     |
| `metaengine:"sort"` | Marks the sort field in a result element |

## Event Integration

For JSON-encoded events (including `event.Event` payloads):

```go
// Decode JSON payload into the fold's expected type automatically.
store.ApplyEncoded(string(evt.Type()), evt.Payload())
```

For non-JSON encodings (CBOR, raw), decode manually and use `store.Apply`.

### projection.Projection Adapter

Create a thin adapter to integrate with `projectionhost.Host`:

```go
type projectionAdapter struct{ store *metaengine.Store }

func (p *projectionAdapter) Name() string { return "metaengine" }
func (p *projectionAdapter) Handle(_ context.Context, evt event.Event) error {
    return p.store.ApplyEncoded(string(evt.Type()), evt.Payload())
}
func (p *projectionAdapter) EventTypes() []event.Type {
    names := p.store.EventTypeNames()
    types := make([]event.Type, len(names))
    for i, n := range names { types[i] = event.Type(n) }
    return types
}

// Register with projectionhost:
host.Register(&projectionAdapter{store: store})
```

## Key Design Decisions

1. **Pure type inference** — no codegen, no DSL, no FilterOn/SortOn declarations.
   The planner reflects on Go types at startup.
2. **ReadModel/Query separation** — write side (folds) is decoupled from read side
   (queries). Eliminates fold duplication and write amplification.
3. **`Page[T]` result** — pagination is a mechanical concern, not domain intent.
   Query inputs carry only domain fields.
4. **ISP-split backends** — `MapBackend`, `SetBackend`, `CounterBackend`,
   `GraphBackend`, `ScanBackend`. Engines implement only what they support.
5. **Typed key extractors** — `OnUpdate` and `OnRemove` take explicit key functions,
   not first-field convention. Query inputs can use `metaengine:"key"` struct tag.
6. **Type-aware comparisons** — filters use `reflect.DeepEqual`, sorting uses
   type-specific numeric/string/time comparison (not string formatting).
7. **Zero dependencies** — the module uses only the Go standard library.
8. **Concurrency-safe** — `MemoryEngine` and `Store` use `sync.RWMutex`.
