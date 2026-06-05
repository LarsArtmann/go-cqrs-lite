# dispatcher — Generic Dispatcher Infrastructure

[![Go Reference](https://pkg.go.dev/badge/github.com/larsartmann/go-cqrs-lite/dispatcher/v2.svg)](https://pkg.go.dev/github.com/larsartmann/go-cqrs-lite/dispatcher/v2)

Shared generic dispatcher with lifecycle management. Used internally by `command` and `query`.

```bash
go get github.com/larsartmann/go-cqrs-lite/dispatcher/v2
```

## Key Types

| Type | Purpose |
|---|---|
| `Dispatcher[H, M]` | Generic handler + middleware dispatcher |
| `LifecycleMixin` | Embedded Close() support — rejects ops after close |
| `CatalogDispatcher[KT, VT]` | Embeddable catalog introspection |
