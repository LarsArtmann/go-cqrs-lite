# Migration Guide: v3 → v4

> **Audience:** Consumers of `go-cqrs-lite` upgrading from v3 to v4.
> v4 contains two breaking changes: deprecated alias removal and blind store
> codec default flip (JSON → CBOR).

---

## Breaking Change 1: Deprecated Alias Removal

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

### Migration steps

1. Add imports:

```go
import (
    "github.com/larsartmann/go-cqrs-lite/id/v3"
    "github.com/larsartmann/go-cqrs-lite/metadata/v3"
)
```

2. Find-and-replace the table above across your codebase.

3. For `schema.WithDecodeFunc(fn)`, replace with `schema.WithCodec(codec.JSONCodec{})`
   or `schema.WithDecoder(codec.EncodingJSON, fn)`.

### Why?

- **Dependency isolation:** Modules that only need `AggregateRef` can import
  `id/` without pulling in the entire event machinery.
- **Cohesion:** `Tracing` and `CustomData` are metadata types, not event types.
- **Cleaner API surface:** `event/` is focused on events.

---

## Breaking Change 2: Blind Store Codec Default Flip

In v4, blind stores (KV, snapshot, command, query) now default to **CBOR**
instead of JSON. This is safe thanks to the **envelope encoding stamps**
(ADR-0044):

- **Events:** Already self-describing (`evt.Encoding()` stamped on every event).
  No change needed.
- **Blind stores:** Now envelope-wrapped via `codec.WrapEncode`/`UnwrapDecode`.
  The envelope records the codec used. On read, `UnwrapDecode` auto-detects
  the envelope and uses the stamped codec. For pre-envelope data (raw JSON),
  the fallback decoder uses `JSONCodec` — so old data reads correctly.

### What this means for you

- **New data** is written with CBOR inside an envelope (smaller payloads).
- **Old data** (pre-v4, raw JSON without envelope) is auto-detected and
  decoded with JSON. No data migration required.
- **Explicit override** still works if you want JSON:

```go
store := kv.NewTypedStore[UserView, UserID](backend, kv.WithTypedCodec(codec.JSONCodec{}))
snapStore := snapshot.NewTypedStore[State](store, codec.JSONCodec{})
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
