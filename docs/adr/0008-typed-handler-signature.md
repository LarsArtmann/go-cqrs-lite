# ADR-0008: TypedHandler[T] receives Query, not T

**Status:** Accepted

## Context

`query.TypedHandler[T]` and `command.TypedHandler[T]` are defined as:

```go
type TypedHandler[T any] func(ctx context.Context, q Query) (T, error)
```

The handler receives the generic `Query` interface, not the concrete type `T`.

## Decision

Keep the signature receiving `Query` (not `T`). This is intentional.

## Rationale

1. **Dispatcher compatibility**: The underlying `dispatcher.Dispatcher` is generic over message type. It dispatches `Query` values (interface type). The handler must accept the same type the dispatcher provides.

2. **Registration by type name**: Handlers are registered by type name string (e.g., `"GetUserQuery"`). At dispatch time, only the `Query` interface is available. The handler must downcast internally using `query.ExtractPayload[T]()`.

3. **Type safety at registration**: `RegisterTyped[T]()` ensures the handler's return type `T` matches the registration. Type safety is enforced at registration time, not dispatch time.

4. **Alternative rejected**: `func(ctx context.Context, payload T) (T, error)` would require a separate dispatch mechanism per payload type, defeating the purpose of a unified dispatcher.
