# dispatcher — Generic Dispatcher Infrastructure

[![Go Reference](https://pkg.go.dev/badge/github.com/larsartmann/go-cqrs-lite/dispatcher/v4.svg)](https://pkg.go.dev/github.com/larsartmann/go-cqrs-lite/dispatcher/v4)

Shared generic dispatcher with handler registration, middleware chains, and lifecycle management. The foundation for `command.Dispatcher` and `query.Dispatcher`.

```bash
go get github.com/larsartmann/go-cqrs-lite/dispatcher/v4
```

## Why?

Both command and query dispatching need the same machinery: register handlers by type, chain middleware, and manage lifecycle (close/reject-after-close). Rather than duplicate this logic, the `dispatcher` package provides a generic `Dispatcher[H, M]` that both modules embed.

## Quick Start

```go
// command.Dispatcher embeds Dispatcher[Handler, Command]
d := command.NewDispatcher()
d.Register("user.create", handler)
d.Use(middleware.Logging())
d.Dispatch(ctx, cmd)

// query.Dispatcher embeds Dispatcher[Handler, Query]
q := query.NewDispatcher()
query.RegisterTyped(q, "user.get", typedHandler)
```

## API

| Type                        | Description                                                                      |
| --------------------------- | -------------------------------------------------------------------------------- |
| `Dispatcher[H, M]`          | Generic handler + middleware dispatcher. `H` = handler type, `M` = message type. |
| `LifecycleMixin`            | Embedded `Close()` support. Rejects operations after close with an error.        |
| `CatalogDispatcher[KT, VT]` | Embeddable catalog introspection for documentation generation.                   |

### Methods (via embedding)

| Method                    | Description                                           |
| ------------------------- | ----------------------------------------------------- |
| `Register(type, handler)` | Register a handler for a message type.                |
| `Use(middleware...)`      | Append middleware to the chain.                       |
| `Dispatch(ctx, msg)`      | Dispatch a message through the middleware chain.      |
| `Close()`                 | Close the dispatcher. Subsequent ops return an error. |
| `Handlers()`              | Returns registered handler types (for catalog/docs).  |

## Design

- **Type parameters**: `Dispatcher[Handler, Command]` for commands, `Dispatcher[Handler, Query]` for queries. The generic avoids code duplication while keeping type safety.
- **Middleware chain**: Pre-computed on `Use()`. Each `Dispatch` call walks the chain once — no per-call allocation for the middleware list.
- **Lifecycle safety**: `LifecycleMixin.Close()` sets a flag. After close, `Dispatch` and `Register` return an error immediately.

## Related Modules

- [**command**](../command/README.md) — `command.Dispatcher` embeds `Dispatcher[Handler, Command]`
- [**query**](../query/README.md) — `query.Dispatcher` embeds `Dispatcher[Handler, Query]`
- [**catalog**](../catalog/README.md) — Uses `CatalogDispatcher` for introspection and doc generation
