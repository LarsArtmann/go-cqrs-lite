# Migration Guide: v3 → v4

> **Audience:** Consumers of `go-cqrs-lite` upgrading from v3 to v4.
> v4 contains four breaking changes: module path migration, deprecated alias
> removal, codec default flip (JSON → CBOR), and BackfillHandler consolidation.

---

## Breaking Change 1: Module Path Migration `/v4` → `/v4`

All module paths change from `/v4` to `/v4`. This is the largest mechanical
change but requires no logic changes.

### What to change

Every `go.mod` require directive and every import path:

```go
// Before (v3):
import (
    "github.com/larsartmann/go-cqrs-lite/event/v4"
    "github.com/larsartmann/go-cqrs-lite/decider/v4"
)

// After (v4):
import (
    "github.com/larsartmann/go-cqrs-lite/event/v4"
    "github.com/larsartmann/go-cqrs-lite/decider/v4"
)
```

### Migration steps

1. Update all `go.mod` require directives: `/v4` → `/v4`
2. Find-and-replace import paths in all `.go` files
3. Run `go mod tidy`

---

## Breaking Change 2: Deprecated Alias Removal

The following aliases were extracted from `event/` and `schema/` into focused
modules in v3.7+ and have now been **removed** in v4.

### What to change

| Before (v3, deprecated)    | After (v4, canonical)                                  |
| -------------------------- | ------------------------------------------------------ |
| `event.AggregateRef`       | `id.AggregateRef`                                      |
| `event.NewAggregateRef`    | `id.NewAggregateRef`                                   |
| `event.AggregateType`      | `id.AggregateType`                                     |
| `event.ParseAggregateType` | `id.ParseAggregateType`                                |
| `event.Tracing`            | `metadata.Tracing`                                     |
| `event.CustomData[K]`      | `metadata.CustomData[K]`                               |
| `event.MergeCustomMaps`    | `metadata.MergeCustomMaps`                             |
| `schema.WithDecodeFunc`    | `schema.WithCodec(c)` or `schema.WithDecoder(enc, fn)` |
| `event.WithNewCodec`       | `event.WithCodec(c)`                                   |
| `event.WithReplay`         | `event.WithProcessingMode(ctx, event.ModeReplay)`      |

### Migration steps

1. Add imports:

```go
import (
    "github.com/larsartmann/go-cqrs-lite/id/v4"
    "github.com/larsartmann/go-cqrs-lite/metadata/v4"
)
```

2. Find-and-replace the table above across your codebase.

3. For `schema.WithDecodeFunc(fn)`, replace with `schema.WithCodec(codec.JSONCodec{})`
   or `schema.WithDecoder(codec.EncodingJSON, fn)`.

---

## Breaking Change 3: Codec Default Flip (JSON → CBOR)

In v4, all codec defaults flip from JSON to CBOR. This applies to:

- `event.DefaultCodec` (used by `event.New()`)
- `kv.NewTypedStore` default codec
- `snapshot.NewTypedStore` default codec
- `command.NewTypedStore` default codec
- `query.NewTypedStore` default codec
- `stack.ReadModel` / `stack.Materialize` default codec

This is safe thanks to two mechanisms (see ADR-0053):

- **Events:** Already self-describing (`evt.Encoding()` stamped on every event).
  Mixed JSON+CBOR streams decode correctly via `DecodePayloadAuto`.
- **Blind stores:** Envelope-wrapped via `codec.WrapEncode`/`UnwrapDecode`
  (ADR-0044). The envelope records the codec used. On read, `UnwrapDecode`
  auto-detects the envelope and uses the stamped codec. For pre-envelope data
  (raw JSON), the fallback decoder uses `JSONCodec` — so old data reads correctly.

### What this means for you

- **No data migration required.** Old data reads correctly.
- **New data** is written with CBOR inside an envelope (smaller payloads).
- **Explicit override** still works if you want JSON:

```go
store := kv.NewTypedStore[UserView, UserID](backend, kv.WithTypedCodec(codec.JSONCodec{}))
snapStore := snapshot.NewTypedStore[State](store, codec.JSONCodec{})
```

- **Process-wide revert** for events:

```go
event.DefaultCodec = codec.JSONCodec{}
```

### Stack-level codec control

```go
// CBOR for everything (events + read models):
bundle, _ := sqlite.New(dsn,
    stack.WithEventCodec(codec.CBORCodec{}),
    stack.WithDefaultCodec(codec.CBORCodec{}),
)

// Then in decide functions:
event.WithCodec(bundle.EventCodec())
```

---

## Breaking Change 4: BackfillHandler Signature Change

`BackfillHandler` now takes `*SSEBroker` instead of `event.SeekableJournal`.
This consolidates SSE and REST backfill under a single codec configuration —
the broker's `WithPayloadTransform` applies to both paths automatically.

`BackfillHandlerWithTransform` is removed (consolidated into `BackfillHandler`).

### What to change

```go
// Before (v3):
handler := http.BackfillHandler(journal)
// or:
handler := http.BackfillHandlerWithTransform(journal, transformFn)

// After (v4):
broker, _ := http.NewSSEBroker(bus,
    http.WithReconnectJournal(journal, 1000),
    http.WithPayloadTransform(transformFn), // if you had a transform
)
handler := http.BackfillHandler(broker)
```

The broker must have `WithReconnectJournal` configured; otherwise the handler
returns 503 Service Unavailable.

---

## Verification

After migrating, verify your code compiles and tests pass:

```bash
go build ./...
go test ./...
```

The `cmd/doc-check` tool verifies that all Go references in documentation
are valid:

```bash
cd cmd/doc-check && GOWORK=off go run . ../../SKILL.md
```
