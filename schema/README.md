# schema — Schema Evolution via Upcasting

[![Go Reference](https://pkg.go.dev/badge/github.com/larsartmann/go-cqrs-lite/schema/v2.svg)](https://pkg.go.dev/github.com/larsartmann/go-cqrs-lite/schema/v3)

Transform old event payloads to the current schema on load, without modifying stored data.

```bash
go get github.com/larsartmann/go-cqrs-lite/schema/v3
```

## Quick Start

```go
upcaster := schema.NewUpcaster("UserCreated", 1, upcastFunc)
versioned, _ := schema.NewVersionedStore(store, upcaster)
events, _ := versioned.Load(ctx, ref)
```

## Related Modules

- [**event/v2**](../event/README.md) — `VersionedStore` wraps an `event.Store`
- [**decider/v2**](../decider/README.md) — Apply upcasters transparently when loading aggregate state
