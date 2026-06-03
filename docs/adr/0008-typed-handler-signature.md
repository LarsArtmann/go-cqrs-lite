# ADR-0008: TypedHandler[Q Query, R any] — dual type parameters

**Status:** Accepted

## Context

`query.TypedHandler` is defined with two type parameters:

```go
type TypedHandler[Q Query, R any] func(ctx context.Context, q Q) (R, error)
```

- `Q` — constrained to `Query` (the concrete query type, e.g., `*GetUserQuery`)
- `R` — constrained to `any` (the result type, e.g., `*GetUserResult`)

The handler receives the **concrete type `Q`**, not the generic `Query` interface.

## Decision

Use dual type parameters `[Q Query, R any]` so the handler receives a typed query and returns a typed result.

## Rationale

1. **Typed query parameter**: Unlike the previous `TypedHandler[T any]` design (which received the generic `Query` interface and required `ExtractPayload[T]`), the handler now receives the concrete query type `Q` directly — no downcasting needed inside the handler.

2. **Typed result**: `R` provides compile-time safety for the return value. Callers use `DispatchTyped[R]()` to get a typed result back without `any` assertion.

3. **Registration bridges the gap**: `RegisterTyped[Q, R]` adapts the typed handler to the untyped `Handler = func(ctx, Query) (any, error)` via a type assertion at the boundary:

```go
func RegisterTyped[Q Query, R any](d *Dispatcher, queryType Type, handler TypedHandler[Q, R]) error {
    return d.Register(queryType, func(ctx context.Context, q Query) (any, error) {
        typed, ok := q.(Q)
        if !ok {
            return nil, ErrTypeAssertion
        }
        return handler(ctx, typed)
    })
}
```

4. **Dispatcher compatibility**: The underlying `dispatcher.Dispatcher` remains generic over a single message type. `RegisterTyped` bridges the typed handler to the untyped core, keeping the dispatcher simple.

5. **Alternative rejected (single-param)**: `TypedHandler[T any] func(ctx, Query) (T, error)` required the handler to call `ExtractPayload[T]()` internally — shifting boilerplate to every handler. The dual-param design eliminates this.

6. **Alternative rejected (Query-only)**: `TypedHandler[Q Query] func(ctx, Q) (any, error)` would still require callers to assert the result type. The dual-param design provides end-to-end type safety.
