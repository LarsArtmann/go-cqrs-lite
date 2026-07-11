# dispatcher — Generic Dispatcher Infrastructure

[![Go Reference](https://pkg.go.dev/badge/github.com/larsartmann/go-cqrs-lite/dispatcher/v4.svg)](https://pkg.go.dev/github.com/larsartmann/go-cqrs-lite/dispatcher/v4)

Shared generic dispatcher with lifecycle management. Used internally by `command` and `query`.

```bash
go get github.com/larsartmann/go-cqrs-lite/dispatcher/v4
```

## Key Types

| Type                        | Purpose                                            |
| --------------------------- | -------------------------------------------------- |
| `Dispatcher[H, M]`          | Generic handler + middleware dispatcher            |
| `LifecycleMixin`            | Embedded Close() support — rejects ops after close |
| `CatalogDispatcher[KT, VT]` | Embeddable catalog introspection                   |

## Related Modules

- [**command/v2**](../command/README.md) — `command.Dispatcher` embeds `Dispatcher[Handler, Command]`
- [**query/v2**](../query/README.md) — `query.Dispatcher` embeds `Dispatcher[Handler, Query]`
- [**catalog/v2**](../catalog/README.md) — Uses `CatalogDispatcher` for introspection
