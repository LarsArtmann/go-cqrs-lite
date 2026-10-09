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
d.Use(middleware.CommandLogging(slog.Default()))
d.Dispatch(ctx, cmd)

// query.Dispatcher embeds Dispatcher[Handler, Query]
q := query.NewDispatcher()
query.RegisterTyped(q, "user.get", typedHandler)
```

## API

| Type                 | Description                                                                         |
| -------------------- | ----------------------------------------------------------------------------------- |
| `Dispatcher[H, M]`   | Generic handler + middleware dispatcher. `H` = handler type, `M` = middleware type. |
| `Lifecycle`          | Embedded close/lifecycle support. Rejects operations after close with an error.     |
| `handlerEntry[H, M]` | Internal registry entry pairing a handler with its middleware-wrapped form.         |

### Methods (via embedding)

| Method                                | Description                                                                 |
| ------------------------------------- | --------------------------------------------------------------------------- |
| `Register(type, handler, wrap)`       | Bind a handler to a type; `wrap` folds middleware into the wrapped handler. |
| `Use(middleware...)`                  | Append middleware to the chain.                                             |
| `Dispatch(type)`                      | Returns the middleware-wrapped handler for the type (or a rejection).       |
| `Close()` / `IsClosed()`              | Close the dispatcher. Subsequent ops return an error.                       |
| `CheckClosed(err)` / `WrapClose(...)` | Closed-guard helpers embedders call from their own public surface.          |

## Design

- **Type parameters**: `Dispatcher[Handler, CommandMiddleware]` for commands, `Dispatcher[Handler, QueryMiddleware]` for queries. The generic avoids code duplication while keeping type safety.
- **Middleware wrapping**: middleware is folded into the handler at `Register` time via the caller-supplied `wrap` function, so `Use()` may be called in any order relative to `Register()`; `Dispatch(type)` then hands back the wrapped handler.
- **Lifecycle safety**: `Lifecycle.Close()` sets a flag. After close, `Dispatch` and `Register` return an error immediately.

## Related Modules

- [**command**](../command/README.md) — `command.Dispatcher` embeds `Dispatcher[Handler, Command]`
- [**query**](../query/README.md) — `query.Dispatcher` embeds `Dispatcher[Handler, Query]`
- [**catalog**](../catalog/README.md) — Catalog introspection and doc generation
