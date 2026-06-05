# pebble — Embedded Key-Value Event Store

[![Go Reference](https://pkg.go.dev/badge/github.com/larsartmann/go-cqrs-lite/pebble/v2.svg)](https://pkg.go.dev/github.com/larsartmann/go-cqrs-lite/pebble/v2)

Event store backed by CockroachDB Pebble. Implements `event.Store` with optimistic concurrency.

```bash
go get github.com/larsartmann/go-cqrs-lite/pebble/v2
```

## Quick Start

```go
db, _ := pebble.Open("data", &pebble.Options{})
store, _ := pebble.NewStore(db, slog.Default())
store.Save(ctx, ref, events, 0)
```
