# Query Handler Generics Migration

**Status:** Implemented (differently) | **Date:** 2026-05-04 | **Closed:** 2026-05-22

## Resolution

Implemented as the "typed bookend" pattern using **function types** (not interfaces):

- `TypedHandler[T any] func(ctx context.Context, q Query) (T, error)` — typed handler function
- `RegisterTyped[T]` — wraps typed handler into legacy `Handler` at registration
- `DispatchTyped[T]` — runtime-asserts result back to `T` after dispatch

**Why function type instead of interface (Option A):** Function types are more Go-idiomatic for
handler patterns (see `http.HandlerFunc`, `command.Handler`). Consumers pass closures directly —
no struct boilerplate needed. The interface approach was considered but added ceremony without benefit.

**Why `any` at the boundary is correct:** Heterogeneous dispatch (one dispatcher, many result types)
requires type erasure at the interface level in Go's type system. This is the same pattern as
`database/sql.Scan(any)`, `json.Unmarshal([]byte, any)`, and `http.Handler.ServeHTTP`. The typed
bookend pattern pushes the `any` ↔ T conversion to framework boundaries, giving consumers
compile-time safety in their handler and caller code.

## Original Problem

`query.Handler` returns `(any, error)`, violating the project's "no `any` types" convention. `DispatchTyped[T]` is a runtime cast workaround that can panic on type mismatch.

```go
// Current — loses type safety at the boundary
type Handler = func(context.Context, Query) (any, error)

// Consumer must use DispatchTyped[T] for type safety
result, err := query.DispatchTyped[User](ctx, dispatcher, q)
```

## Design Goals

1. **Type-safe handlers**: Handler result type known at compile time
2. **Backward compatible**: Existing `Handler` (returns `any`) must still work
3. **No runtime casts**: Eliminate `result.(T)` assertions
4. **Consistent with command pattern**: Command handlers return `error` only; query handlers should be equally clean

## Current Architecture

```
Query interface { Type() Type }
Handler = func(ctx, Query) (any, error)
Dispatcher.Register(Type, Handler)
Dispatcher.Dispatch(ctx, Query) → (any, error)
DispatchTyped[T](ctx, *Dispatcher, Query) → (T, error)  // runtime cast
```

## Option A: Typed Handler Interface (Recommended)

Add a typed handler interface alongside the existing `Handler` function type.

```go
// TypedHandler processes a query and returns a typed result.
type TypedHandler[T any] interface {
    HandleQuery(ctx context.Context, query Query) (T, error)
}
```

### Dispatcher Changes

```go
// RegisterTyped binds a typed handler to a query type.
func RegisterTyped[T any](d *Dispatcher, queryType Type, handler TypedHandler[T]) error {
    // Wrap typed handler into legacy Handler for internal storage
    wrapper := func(ctx context.Context, q Query) (any, error) {
        return handler.HandleQuery(ctx, q)
    }
    return d.Register(queryType, wrapper)
}

// DispatchTyped sends a query and returns a typed result.
// No runtime cast needed — the handler already returns T.
func DispatchTyped[T any](ctx context.Context, d *Dispatcher, query Query) (T, error) {
    result, err := d.Dispatch(ctx, query)
    if err != nil {
        var zero T
        return zero, err
    }
    // Still need cast because internal storage is (any, error)
    // But the cast is guaranteed safe if registered via RegisterTyped
    typed, ok := result.(T)
    if !ok {
        var zero T
        return zero, fmt.Errorf("type mismatch for query %q", query.Type())
    }
    return typed, nil
}
```

**Pros**: Non-breaking, opt-in, typed handlers for new consumers.
**Cons**: Internal storage still uses `any`; the cast still exists but is guaranteed safe.

### Option B: Generic Dispatcher (Breaking)

Replace the entire dispatcher with generics.

```go
type TypedDispatcher[Q Query, R any] struct {
    handlers map[Type]func(context.Context, Q) (R, error)
}

func (d *TypedDispatcher[Q, R]) Register(queryType Type, handler func(context.Context, Q) (R, error)) error
func (d *TypedDispatcher[Q, R]) Dispatch(ctx context.Context, query Q) (R, error)
```

**Rejected because**:

- Each `(QueryType, ResultType)` pair needs a separate dispatcher instance
- Can't have queries returning different types in one dispatcher
- Breaks all existing consumers
- Middleware becomes much harder (can't chain handlers with different return types)

### Option C: Query-Specific Result Type (Breaking)

Add a result type to the Query interface.

```go
type Query interface {
    Type() Type
    ResultType() reflect.Type  // or a generic constraint
}
```

**Rejected because**: Runtime type inspection, not compile-time safe. Adds reflection overhead.

## Why Option A

| Criterion           | Option A (Typed Interface) | Option B (Generic Dispatcher) | Option C (Result Type) |
| ------------------- | -------------------------- | ----------------------------- | ---------------------- |
| Backward compatible | ✅ Non-breaking            | ❌ Breaking                   | ❌ Breaking            |
| Compile-time safe   | ✅ (at registration site)  | ✅                            | ❌ Runtime             |
| Single dispatcher   | ✅                         | ❌ Per type pair              | ✅                     |
| Middleware support  | ✅ Existing works          | ❌ Complex                    | ✅                     |
| Effort              | Low (interface + wrapper)  | High (full rewrite)           | Medium                 |

## Migration Path

### Phase 1: Add Interface (Non-breaking)

1. Add `TypedHandler[T]` interface to `core/query/query.go`
2. Add `RegisterTyped[T]` function to `core/query/dispatcher.go`
3. Update `DispatchTyped[T]` to document the guaranteed-safe cast
4. Add tests for typed handler registration and dispatch

### Phase 2: Consumer Migration (Gradual)

1. Consumers adopt `TypedHandler[T]` for new query handlers
2. Existing `Handler` func continues to work
3. No rush — both patterns coexist

### Phase 3: Deprecation (Future)

1. Add `// Deprecated: Use TypedHandler[T] instead` to `Handler` type alias
2. Eventually remove in next major version

## API Surface

```
core/query/query.go       — TypedHandler[T] interface (3 lines)
core/query/dispatcher.go  — RegisterTyped[T] function (15 lines)
core/query/dispatcher.go  — Update DispatchTyped doc comments
```

**Estimated effort:** 2 hours (interface + wrapper + tests).

## Example Usage

```go
// Before (current)
func handleGetUser(ctx context.Context, q query.Query) (any, error) {
    // ... type-unsafe return
    return User{ID: "123", Name: "Alice"}, nil
}
dispatcher.Register("GetUser", handleGetUser)

result, err := query.DispatchTyped[User](ctx, dispatcher, getUserQuery)

// After (typed)
type GetUserHandler struct{}

func (h *GetUserHandler) HandleQuery(ctx context.Context, q query.Query) (User, error) {
    return User{ID: "123", Name: "Alice"}, nil
}

query.RegisterTyped[User](dispatcher, "GetUser", &GetUserHandler{})

result, err := query.DispatchTyped[User](ctx, dispatcher, getUserQuery)
```

## Open Questions

- Should `TypedHandler[T]` be a struct or interface? Interface is more flexible (mocks, multiple implementations).
- Should `RegisterTyped` validate that T matches the handler's return type at compile time? Yes — generics enforce this.
- Can middleware inspect the return type? Only with `reflect` — but middleware typically doesn't need to.
