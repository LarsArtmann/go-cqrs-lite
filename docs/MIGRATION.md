# API Migration Guide

## query.Handler: `any` → TypedHandler[T]

### The Problem

The `query.Handler` type returns `any`:

```go
type Handler func(context.Context, Query) (any, error)
```

This is required because the dispatcher manages heterogeneous handlers —
different queries return different types. A single dispatcher cannot be
generic over all return types.

### The Solution: RegisterTyped + DispatchTyped

Use standalone generic functions for type-safe query handling:

```go
// Registration — type-safe at the boundary
err := query.RegisterTyped[*GetUserResult](dispatcher, "GetUser",
    func(ctx context.Context, q query.Query) (*GetUserResult, error) {
        return &GetUserResult{Name: "Alice"}, nil
    },
)

// Dispatch — type-safe result
result, err := query.DispatchTyped[*GetUserResult](ctx, dispatcher, myQuery)
```

### Migration Steps

1. **Identify** all query handlers that return a concrete type
2. **Replace** `dispatcher.Register("QueryName", handler)` with
   `query.RegisterTyped[T](dispatcher, "QueryName", handler)`
3. **Replace** `dispatcher.Dispatch(ctx, query)` with
   `query.DispatchTyped[T](ctx, dispatcher, query)`
4. **Remove** manual type assertions: `result.(*GetUserResult)` → `result`
   is already `*GetUserResult`

### Why Not a Generic Method?

Go does not support generic methods on concrete types. A generic
`DispatchTyped` method on `*query.Dispatcher` is impossible in current
Go. Standalone generic functions are the idiomatic workaround — same
pattern used in `cmp.Ordered`, `slices.Contains`, etc.

### Before / After

**Before:**

```go
dispatcher.Register("GetUser", func(ctx context.Context, q query.Query) (any, error) {
    return &GetUserResult{Name: "Alice"}, nil
})

raw, err := dispatcher.Dispatch(ctx, getUserQuery)
if err != nil {
    return err
}
result := raw.(*GetUserResult) // unchecked type assertion
```

**After:**

```go
query.RegisterTyped[*GetUserResult](dispatcher, "GetUser",
    func(ctx context.Context, q query.Query) (*GetUserResult, error) {
        return &GetUserResult{Name: "Alice"}, nil
    },
)

result, err := query.DispatchTyped[*GetUserResult](ctx, dispatcher, getUserQuery)
// result is already *GetUserResult — no assertion needed
```
