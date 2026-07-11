# schema — Schema Evolution via Upcasting

[![Go Reference](https://pkg.go.dev/badge/github.com/larsartmann/go-cqrs-lite/schema/v4.svg)](https://pkg.go.dev/github.com/larsartmann/go-cqrs-lite/schema/v4)

Transform old event payloads to the current schema on load, without modifying stored data.

```bash
go get github.com/larsartmann/go-cqrs-lite/schema/v4
```

## Quick Start

```go
upcaster := schema.NewUpcaster("UserCreated", 1, upcastFunc)
versioned, _ := schema.NewVersionedStore(store, upcaster)
events, _ := versioned.Load(ctx, ref)
```

## VersionedSeekableJournal — upcasters for projectionhost

`VersionedStore` wraps an `event.Store` (provides `Load`/`Save` per aggregate).
`VersionedSeekableJournal` wraps an `event.SeekableJournal` (provides
`ReadFrom(position)` across all aggregates) — the interface `projectionhost.New()`
requires. Use it when you need schema evolution on the projection read path:

```go
vjournal, _ := schema.NewVersionedSeekableJournal(journal, upcaster)
host, _ := projectionhost.New(vjournal, checkpointStore)
```

Upcasters run transparently on every `ReadFrom` call. The projection handler
always sees the latest schema version, regardless of what version was stored.

**Scope:** `VersionedSeekableJournal` wraps `SeekableJournal` (position-based
`ReadFrom`), not the full `event.Store` interface. For aggregate-scoped loads
(`Load`/`Save`), use `VersionedStore` instead.

## Related Modules

- [**event/v4**](../event/README.md) — `VersionedStore` wraps an `event.Store`
- [**decider/v4**](../decider/README.md) — Apply upcasters transparently when loading aggregate state
- [**projectionhost/v4**](../projectionhost/README.md) — `VersionedSeekableJournal` feeds upcasted events to projections
