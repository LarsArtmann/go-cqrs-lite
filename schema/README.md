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

## Named ops (declarative upcasting)

Hand-writing one closure per evolution step is the boilerplate T2 of the
2026-10-09 v5 schema-evolution proposal removes. Declare the steps as ops,
compile once, compose like any other `SourceTransform`:

```go
import (
    "github.com/larsartmann/go-cqrs-lite/event/v4"
    "github.com/larsartmann/go-cqrs-lite/schema/v4"
)

chain, err := schema.Compile(
    schema.RenameType("user.name_changed", "user.renamed"),
    schema.RenameField("user.renamed", 1, "name", "displayName"),
    schema.RemoveField("user.renamed", 2, "legacyToken"),
    schema.Transform("balance.updated", 1, reshapeToMoney), // the only hand-written logic
    schema.Split("checkout.completed", 1,
        schema.Producing("cart.checked_out", cartPayload),
        schema.Producing("payment.requested", paymentPayload)),
    schema.Drop("audit.legacy_ping"),
)
if err != nil {
    return err // duplicates, ambiguous renames, cycles — rejected at build time
}

versioned := event.DecorateStore(store, nil, chain.SourceTransform())
events, _ := versioned.Load(ctx, ref)
```

Semantics (Axon Framework 5 event-transformation model, adopted):

- **Matching is order-independent**: exact `(event type, schema version)`
  first, then type-only ops (`RenameType`, `Drop`). Duplicate matches are
  compile errors, never silently resolved.
- **Versioning**: payload ops advance the schema version by exactly one;
  `RenameType` changes identity but not version; split outputs inherit
  stream identity and position with fresh event IDs.
- **Encoding-aware**: ops decode via the event's own `Encoding()` stamp and
  re-encode with the same codec — mixed JSON/CBOR journals upcast correctly.
- **Identity-preserving**: upcasted events keep their event ID, timestamp,
  metadata, and stream position (the closure style could silently mint new
  IDs — the ops cannot).
- **Decode failures are policy**: `FailOnDecodeError` (default),
  `PassthroughOnDecodeError`, or `DropOnDecodeError` via
  `WithDecodePolicy` on each payload op.
- **Batch ops**: `Split`, `Drop`, and `RenameType` change event count or
  identity, so they need `chain.SourceTransform()` (batch-level). The 1:1
  ops also convert to classic upcasters: `chain.Upcasters()`.

## API

| Symbol                                                                                     | Description                                                                                                     |
| ------------------------------------------------------------------------------------------ | --------------------------------------------------------------------------------------------------------------- |
| `NewUpcaster(eventType, fromVer, fn)`                                                      | Creates an upcaster for a specific event type and source version.                                               |
| `UpcastSourceTransform(upcasters...)`                                                      | `event.SourceTransform` applying upcasters on load — compose via `event.DecorateStore`/`event.DecorateJournal`. |
| `Compile(ops...)`                                                                          | Validates and compiles named ops into an immutable `Chain`.                                                     |
| `Chain.SourceTransform()`                                                                  | Batch-level transform applying every op (incl. `Split`/`Drop`/`RenameType`).                                    |
| `Chain.Upcasters()`                                                                        | Converts the 1:1 ops to classic `Upcaster` values; batch ops are rejected.                                      |
| `RenameType` / `RenameField` / `AddField` / `RemoveField` / `Transform` / `Split` / `Drop` | The named ops. Each compiles into the chain; payload ops advance the schema version by one.                     |
| `WithDecodePolicy(Fail\|Passthrough\|Drop)`                                                | Per-op decode-failure policy (`FailOnDecodeError` is the default).                                              |
| `Validator`                                                                                | Validates event payloads against registered types.                                                              |
| `RegisterType[T]()`                                                                        | Register a Go type for schema validation (ADR-0017).                                                            |
| `NewVersionedStore(store, upcasters...)`                                                   | **Deprecated** (removed v5): pre-transform shell; forwards to `event.DecorateStore`.                            |
| `NewVersionedSeekableJournal(j, upcasters...)`                                             | **Deprecated** (removed v5): pre-transform shell; forwards to `event.DecorateJournal`.                          |

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
