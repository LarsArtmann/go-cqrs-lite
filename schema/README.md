# schema — Schema Evolution via Upcasting

[![Go Reference](https://pkg.go.dev/badge/github.com/larsartmann/go-cqrs-lite/schema/v4.svg)](https://pkg.go.dev/github.com/larsartmann/go-cqrs-lite/schema/v4)

Transform old event payloads to the current schema on load, without modifying stored data.

```bash
go get github.com/larsartmann/go-cqrs-lite/schema/v4
```

## Quick Start

Compose upcasters as a `SourceTransform` with `event.DecorateStore` — the
decorated store keeps every capability of the inner store (Journal,
SeekableJournal, MultiSink, `io.Closer`) with upcasting applied on reads:

```go
import (
    "github.com/larsartmann/go-cqrs-lite/event/v4"
    "github.com/larsartmann/go-cqrs-lite/schema/v4"
)

upcaster := schema.NewUpcaster("UserCreated", 1, upcastFunc)

versioned := event.DecorateStore(eventStore, nil, schema.UpcastSourceTransform(upcaster))
events, _ := versioned.Load(ctx, ref)
```

## Upcasters on the projection read path

For `projectionhost.New()`, which reads through an `event.SeekableJournal`
(position-based `ReadFrom` across all aggregates), decorate the journal
instead of the store:

```go
import (
    "github.com/larsartmann/go-cqrs-lite/event/v4"
    "github.com/larsartmann/go-cqrs-lite/schema/v4"
)

vjournal := event.DecorateJournal(journal, schema.UpcastSourceTransform(upcaster))
host, _ := projectionhost.New(vjournal, checkpointStore)
```

Upcasters run transparently on every `ReadFrom` call. The projection handler
always sees the latest schema version, regardless of what version was stored.

## API

| Symbol                                         | Description                                                                                                     |
| ---------------------------------------------- | --------------------------------------------------------------------------------------------------------------- |
| `NewUpcaster(eventType, fromVer, fn)`          | Creates an upcaster for a specific event type and source version.                                               |
| `UpcastSourceTransform(upcasters...)`          | `event.SourceTransform` applying upcasters on load — compose via `event.DecorateStore`/`event.DecorateJournal`. |
| `Validator`                                    | Validates event payloads against registered types.                                                              |
| `RegisterType[T]()`                            | Register a Go type for schema validation (ADR-0017).                                                            |
| `NewVersionedStore(store, upcasters...)`       | **Deprecated** (removed v5): pre-transform shell; forwards to `event.DecorateStore`.                            |
| `NewVersionedSeekableJournal(j, upcasters...)` | **Deprecated** (removed v5): pre-transform shell; forwards to `event.DecorateJournal`.                          |

## Design

- **Read-time transformation**: Stored data is never modified. Upcasting happens on every load, so old and new versions coexist seamlessly.
- **Per-event-type**: Each upcaster targets a specific event type and source version. Multiple upcasters chain naturally (v1, then v2, then v3).
- **Capability-preserving**: `UpcastSourceTransform` composes with `event.DecorateStore`, so Journal, SeekableJournal, BackwardsSource, MultiSink, and `io.Closer` all keep working (ADR-0126).
- **Immutable events**: Upcasters return new `*ImmutableEvent` instances. Returning nil or the input event is rejected (ErrInvalidUpcastResult) — the original event is never mutated. An upcaster that stamps a higher schema version (e.g. a v1→v3 jump) keeps it; otherwise source+1 is stamped. Duplicate (type, source version) registrations are ignored — first wins.
- **Validator**: Optional payload validation via `RegisterType[T]()`. Checks JSON schema conformance at the boundary.

## Related Modules

- [**event**](../event/README.md) — `DecorateStore`/`DecorateJournal` apply the transform
- [**decider**](../decider/README.md) — Apply upcasters transparently when loading aggregate state
- [**projectionhost**](../projectionhost/README.md) — Decorated journals feed upcasted events to projections
