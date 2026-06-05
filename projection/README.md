# projection — Replay+Live Projection Runner

[![Go Reference](https://pkg.go.dev/badge/github.com/larsartmann/go-cqrs-lite/projection/v2.svg)](https://pkg.go.dev/github.com/larsartmann/go-cqrs-lite/projection/v2)

Build read models from event streams with automatic checkpoint management.

```bash
go get github.com/larsartmann/go-cqrs-lite/projection/v2
```

## Quick Start

```go
b := projection.NewBuilder("user-projection")
b.On("user.created", handler)
runner := b.Runner(store, bus)
go runner.Run(ctx)
```
