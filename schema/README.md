# schema — Schema Evolution via Upcasting

[![Go Reference](https://pkg.go.dev/badge/github.com/larsartmann/go-cqrs-lite/schema/v2.svg)](https://pkg.go.dev/github.com/larsartmann/go-cqrs-lite/schema/v2)

Transform old event payloads to the current schema on load, without modifying stored data.

```bash
go get github.com/larsartmann/go-cqrs-lite/schema/v2
```

## Quick Start

```go
upcaster := schema.NewUpcaster("UserCreated", 1, upcastFunc)
versioned, _ := schema.NewVersionedStore(store, upcaster)
events, _ := versioned.Load(ctx, ref)
```
