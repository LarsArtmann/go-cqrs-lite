# metadata — Shared Tracing and Custom-Data Containers

[![Go Reference](https://pkg.go.dev/badge/github.com/larsartmann/go-cqrs-lite/metadata/v4.svg)](https://pkg.go.dev/github.com/larsartmann/go-cqrs-lite/metadata/v4)

Shared tracing identifiers and typed custom-data containers used by `event`, `command`, and `query` metadata. Extracted from `event/` to break tight coupling between modules.

```bash
go get github.com/larsartmann/go-cqrs-lite/metadata/v4
```

## Why?

Before this package existed, `Tracing` and `CustomData` (the latter deprecated — see below) lived inside `event/`. Every module needing them (command, query) had to import `event/`, creating tight coupling that violated the seven-tier module model (ADR-0046). The `metadata/` module breaks that dependency: `command/` and `query/` embed these types directly without pulling in the full `event/` package.

## Types

### Tracing

Holds cross-cutting tracing identifiers shared by event, command, and query metadata:

```go
type Tracing struct {
    CorrelationID id.CorrelationID `json:"correlationId"`
    CausationID   id.CausationID   `json:"causationId"`
    UserID        id.UserID        `json:"userId"`
    RequestID     id.RequestID     `json:"requestId"`
    ActorID       id.ActorID       `json:"actorId,omitzero"`
}
```

When embedded anonymously in a struct, `encoding/json` promotes these fields to the parent level, preserving the existing JSON shape: `{"correlationId": "...", ...}`.

| Method             | Description                                     |
| ------------------ | ----------------------------------------------- |
| `Tracing.IsZero()` | True when no tracing field has been set.        |
| `Tracing.Merge(o)` | Overlays non-zero fields from `other` onto `t`. |

### Metadata[K]

The canonical reusable base for metadata types that carry tracing identifiers
and a typed custom map. `command.Metadata` and `query.Metadata` are ALIASES of
`Metadata[MetadataKey]` (this module's generic metadata type), not standalone
structs.

```go
type Metadata[K ~string] struct {
    Tracing
    Custom map[K]string `json:"custom,omitempty"`
}
```

The type parameter `K` is a named string type (the module's own `MetadataKey`), so each module's custom keys are type-safe and cannot be accidentally mixed.

| Method                        | Description                                        |
| ----------------------------- | -------------------------------------------------- |
| `Metadata[K].Clone()`         | Returns a copy with a cloned Custom map.           |
| `Metadata[K].Merge(o)`        | Overlays tracing and custom entries from `other`.  |
| `Metadata[K].WithCustom(k,v)` | Returns a copy with `k` set to `v` (non-mutating). |
| `Metadata[K].EnsureCustom()`  | **Deprecated.** Use `WithCustom` instead.          |

> `CustomData[K]` is a **deprecated type alias** of `Metadata[K]` kept for
> backward compatibility — removed at v5 (ADR-0126). Use `Metadata[K]` in
> new code.

### MergeCustomMaps[K]

Utility that merges two custom maps, returning the base unchanged when `other` is empty (no allocation).

## Usage

```go
import "github.com/larsartmann/go-cqrs-lite/metadata/v4"

type MyKey string

// Embed Metadata for the full Clone/Merge/WithCustom API:
type MyMetadata struct {
    metadata.Metadata[MyKey]
    // additional module-specific fields...
}

// Or embed Tracing directly and own your own Custom map
// (the approach command/ and query/ take):
type MyStandaloneMetadata struct {
    metadata.Tracing
    Custom map[MyKey]string `json:"custom,omitempty"`
}
```

## References

- **ADR-0031**: Typed Metadata fields
- **ADR-0046**: Four-tier module model

## Related Modules

- [**event**](../event/README.md) — `event.Metadata` embeds `Tracing` and adds event-specific fields (Source, Tombstone, Causation)
- [**command**](../command/README.md) — `command.Metadata` is an alias of `metadata.Metadata[command.MetadataKey]`
- [**query**](../query/README.md) — `query.Metadata` is an alias of `metadata.Metadata[query.MetadataKey]`
