# metadata — Shared Custom-Data Container

[![Go Reference](https://pkg.go.dev/badge/github.com/larsartmann/go-cqrs-lite/metadata/v4.svg)](https://pkg.go.dev/github.com/larsartmann/go-cqrs-lite/metadata/v4)

Typed custom-data container used by `event`, `command`, and `query` metadata. Extracted from `event/` to break tight coupling between modules.

```bash
go get github.com/larsartmann/go-cqrs-lite/metadata/v4
```

## Why?

Before this package existed, `Tracing` and `CustomData` lived inside `event/`. Every module needing them (command, query) had to import `event/`, creating tight coupling that violated the seven-tier module model (ADR-0046). The `metadata/` module breaks that dependency: `command/` and `query/` embed these types directly without pulling in the full `event/` package.

Since ADR-0111 Phase 3, the shared tracing fields (CorrelationID, CausationID, ActorID, RequestID, etc.) live in [`record.CommonMetadata`](../record/) — the single structural base for both events and commands. This package retains `CustomData[K]` for backward compatibility but the preferred pattern is to embed `record.CommonMetadata` directly.

## Types

### CustomData[K]

**Deprecated.** Prefer embedding `record.CommonMetadata` directly (see [Preferred Pattern](#preferred-pattern-adr-0111) below).

A reusable base for metadata types that carry common metadata and a typed custom map. Now embeds `record.CommonMetadata` instead of the deleted `Tracing` type.

```go
type CustomData[K ~string] struct {
    record.CommonMetadata
    Custom map[K]string `json:"custom,omitempty"`
}
```

The type parameter `K` is a named string type (the module's own `MetadataKey`), so each module's custom keys are type-safe and cannot be accidentally mixed.

| Method                          | Description                                        |
| ------------------------------- | -------------------------------------------------- |
| `CustomData[K].Clone()`         | Returns a copy with a cloned Custom map.           |
| `CustomData[K].Merge(o)`        | Overlays common metadata and custom entries from `other`. |
| `CustomData[K].WithCustom(k,v)` | Returns a copy with `k` set to `v` (non-mutating). |
| `CustomData[K].EnsureCustom()`  | **Deprecated.** Use `WithCustom` instead.          |

### MergeCustomMaps[K]

Utility that merges two custom maps, returning the base unchanged when `other` is empty (no allocation).

## Usage

### Preferred Pattern (ADR-0111)

Embed `record.CommonMetadata` directly and own your custom fields:

```go
import "github.com/larsartmann/go-cqrs-lite/record/v4"

type MyKey string

type MyMetadata struct {
    record.CommonMetadata
    Custom map[MyKey]string `json:"custom,omitempty"`
    // additional module-specific fields...
}
```

This is the pattern `command.Metadata` and `query.Metadata` follow.

### Deprecated Pattern (CustomData)

```go
import "github.com/larsartmann/go-cqrs-lite/metadata/v4"

type MyKey string

type MyMetadata struct {
    metadata.CustomData[MyKey]
    // additional module-specific fields...
}
```

## References

- **ADR-0031**: Typed Metadata fields
- **ADR-0046**: Seven-tier module model
- **ADR-0111**: Record type extraction (Phase 3 moved tracing to `record.CommonMetadata`)

## Related Modules

- [**record**](../record/README.md) — `record.CommonMetadata` is the shared structural base for events and commands (tracing fields, timestamps, schema version)
- [**event**](../event/README.md) — `event.Metadata` embeds `record.CommonMetadata` and adds event-specific fields (Source, Tombstone, Causation)
- [**command**](../command/README.md) — `command.Metadata` embeds `record.CommonMetadata` directly
- [**query**](../query/README.md) — `query.Metadata` embeds `record.CommonMetadata` directly
