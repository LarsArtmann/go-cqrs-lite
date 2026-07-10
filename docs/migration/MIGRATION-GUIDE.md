# Migration Guide: id/ + metadata/ Extraction

> **Audience:** Consumers of `go-cqrs-lite/event/v3` who import `AggregateRef`,
> `AggregateType`, `Tracing`, or `CustomData` from the `event` package.

---

## What Changed

In v3.7+, the types `AggregateRef`, `AggregateType`, `Tracing`, and `CustomData`
were extracted from `event/` into two focused modules:

- **`id/`** — `AggregateRef`, `AggregateType`, `NewAggregateRef`, `ParseAggregateType`
- **`metadata/`** — `Tracing`, `CustomData[K]`, `MergeCustomMaps`

The `event/` package retains **deprecated type aliases** so existing code
continues to compile. These aliases will be **removed in v4**.

---

## Migration Steps

### Step 1: Update imports

```go
// Before
import "github.com/larsartmann/go-cqrs-lite/event/v3"

// After — add the focused modules
import (
    "github.com/larsartmann/go-cqrs-lite/event/v3"
    "github.com/larsartmann/go-cqrs-lite/id/v3"
    "github.com/larsartmann/go-cqrs-lite/metadata/v3"
)
```

### Step 2: Replace type references

| Before (deprecated)        | After (canonical)          |
| -------------------------- | -------------------------- |
| `event.AggregateRef`       | `id.AggregateRef`          |
| `event.NewAggregateRef`    | `id.NewAggregateRef`       |
| `event.AggregateType`      | `id.AggregateType`         |
| `event.ParseAggregateType` | `id.ParseAggregateType`    |
| `event.Tracing`            | `metadata.Tracing`         |
| `event.CustomData[K]`      | `metadata.CustomData[K]`   |
| `event.MergeCustomMaps`    | `metadata.MergeCustomMaps` |

### Step 3: Verify

Run `staticcheck ./...` — SA1019 warnings will disappear once all references
are updated. The deprecated aliases themselves will be removed in v4.

---

## Why?

- **Dependency isolation:** `command/` no longer depends on `event/` at compile
  time. Modules that only need `AggregateRef` can import `id/` without pulling
  in the entire event machinery.
- **Cohesion:** `Tracing` and `CustomData` are metadata types, not event types.
  They belong in `metadata/`, not `event/`.
- **v4 readiness:** The extraction is the groundwork for a cleaner v4 where
  `event/` is focused on events, not on infrastructure types.

---

## Codec Default Changes (v4)

In v4, blind stores (KV, snapshot, command, query) will default to CBOR instead
of JSON. Thanks to the **envelope encoding stamps** (ADR-0044), this migration
is safe:

- **Events:** Already self-describing (`evt.Encoding()` stamped on every event).
  Mixed JSON/CBOR streams decode correctly via `DecodePayloadAuto`.
- **Blind stores:** Now envelope-wrapped. Old JSON data is auto-detected on read
  and decoded with the original codec. New writes use the store's configured
  codec (CBOR by default in v4, JSON in v3).

**Action needed for v4:** To explicitly opt into CBOR before v4:

```go
store := kv.NewTypedStore[UserView, UserID](backend, kv.WithTypedCodec(codec.CBORCodec{}))
snapStore := snapshot.NewTypedStore[State](store, codec.CBORCodec{})
```

Or at the stack level:

```go
bundle, _ := sqlite.New(dsn, stack.WithDefaultCodec(codec.CBORCodec{}))
```

No data migration is required — the envelope handles mixed old/new data
transparently.

---

## Deprecated Alias Removal (v4)

The following aliases will be **deleted** in v4. Update imports now to prepare:

| File                     | Alias                |
| ------------------------ | -------------------- |
| `event/aggregate_ref.go` | `AggregateRef`       |
| `event/aggregate_ref.go` | `NewAggregateRef`    |
| `event/event.go`         | `AggregateType`      |
| `event/event.go`         | `ParseAggregateType` |
| `event/tracing.go`       | `Tracing`            |
| `event/customdata.go`    | `CustomData[K]`      |
| `event/custommap.go`     | `MergeCustomMaps`    |
| `schema/validator.go`    | `WithDecodeFunc`     |
