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

## API

| Symbol                                  | Description                                                         |
| --------------------------------------- | ------------------------------------------------------------------- |
| `NewUpcaster(eventType, fromVer, fn)`   | Creates an upcaster for a specific event type and source version.   |
| `NewVersionedStore(store, upcasters...)`| Wraps an `event.Store`. Upcasts on every `Load`.                    |
| `NewVersionedSeekableJournal(j, upcasters...)` | Wraps a `SeekableJournal`. Upcasts on every `ReadFrom`.      |
| `Validator`                             | Validates event payloads against registered types.                  |
| `RegisterType[T]()`                     | Register a Go type for schema validation (ADR-0017).                |

## Design

- **Read-time transformation**: Stored data is never modified. Upcasting happens on every load, so old and new versions coexist seamlessly.
- **Per-event-type**: Each upcaster targets a specific event type and source version. Multiple upcasters chain naturally (v1, then v2, then v3).
- **Immutable events**: Upcasters return new `*ImmutableEvent` instances. The original event is never mutated.
- **Validator**: Optional payload validation via `RegisterType[T]()`. Checks JSON schema conformance at the boundary.

## Related Modules

- [**event**](../event/README.md) — `VersionedStore` wraps an `event.Store`
- [**decider**](../decider/README.md) — Apply upcasters transparently when loading aggregate state
- [**projectionhost**](../projectionhost/README.md) — `VersionedSeekableJournal` feeds upcasted events to projections
