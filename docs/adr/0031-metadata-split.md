# ADR-0031: Kill Metadata Aliases — Embed `Tracing`, Add Typed Fields

| Field   | Value            |
| ------- | ---------------- |
| Date    | 2026-06-20       |
| Status  | Implemented (v3) |
| Decider | Lars Artmann     |

## Context

`event.Metadata` is a `map[string]string` with ~12 domain concepts stuffed into
its `Custom` sub-map as strings:

- `CorrelationID`, `CausationID`, `UserID`, `RequestID` (tracing)
- `TombstoneStatus` (lifecycle, ADR-0006)
- `CommandType`, `CommandID` (causation)
- plus codec/schema/version bookkeeping

Worse, `command.Metadata = event.Metadata` and `query.Metadata = event.Metadata`
**alias** the event's map type — forcing the event's shape onto commands and
queries that have nothing to do with tombstones or event versions.

This causes:

1. **Stringly-typed domain.** `metadata["tombstone"] == "true"` is a bug waiting
   to happen (typo, no compile-time check).
2. **Wrong-home coupling.** A query that wants a correlation ID must import the
   event's tombstone-aware metadata type.
3. **Silent metadata loss.** The SQL stores had a bug where `scanCommand` /
   `scanQuery` forgot to unmarshal metadata — a direct consequence of the alias
   hiding the real shape.

## Decision

**Three changes, all additive (old API kept for v2, removed at v3):**

### 1. Extract `Tracing` struct

```go
// event/tracing.go
type Tracing struct {
    CorrelationID string
    CausationID   string
    UserID        string
    RequestID     string
}
```

`event.Metadata` embeds `Tracing`. `command.Metadata` and `query.Metadata`
become their **own** structs that also embed `Tracing` — they no longer alias
`event.Metadata`.

### 2. Add typed `TombstoneMark` and `Causation`

```go
type TombstoneStatus int // already exists as iota: Active, Tombstoned, Undetermined

type TombstoneMark struct {
    Status TombstoneStatus
    Reason string // optional
}

type Causation struct {
    CommandType string
    CommandID   string
}
```

`event.Metadata` gains `Tombstone *TombstoneMark` and `Causation *Causation`
typed fields. The old string-based `Custom["tombstone.status"]` entries are
written **in parallel** during the transition so old readers keep working.

### 3. Each module owns its own `Metadata`

- `event.Metadata` — embeds `Tracing`, has `Tombstone` + `Causation` (event-only
  concerns).
- `command.Metadata` — embeds `Tracing` only. Own struct, own file.
- `query.Metadata` — embeds `Tracing` only. Own struct, own file.

The `command.Metadata = event.Metadata` and `query.Metadata = event.Metadata`
aliases are **deleted** at the v3 boundary.

## Alternatives Considered

- **Keep the alias, add typed accessors.** Rejected — the alias is the root
  cause. Accessors paper over it; the wrong-home coupling remains.
- **Make `Metadata` a full struct with all fields.** Rejected — loses the
  `Custom` escape hatch that consumers legitimately use for tenant IDs,
  audit data, etc. Keep `Custom map[string]string` for the long tail.
- **Use a generic `Metadata[T]`.** Rejected — over-engineered. Three small
  structs sharing a `Tracing` embed is simpler and more readable.

## Consequences

- `event.WithCorrelationID` etc. now set the typed `Tracing.CorrelationID`
  field (and still mirror to `Custom` for v2 back-compat).
- `MarkTombstone` / `MarkRebirth` set the typed `Tombstone` field.
- `WithCommandCausality` sets the typed `Causation` field.
- SQL stores' `scanCommand`/`scanQuery` unmarshal into the new
  `command.Metadata` / `query.Metadata` structs directly — no more alias-induced
  silent loss.
- Signing and encryption are unaffected: they hash/encrypt the payload bytes,
  not the metadata shape.

## Forward references

- Execution plan T05 (Tracing), T06 (TombstoneMark), T07 (Causation).
- ADR-0006 (tombstone soft-delete) — `TombstoneMark` is its typed realization.
- ADR-0017 (schema registry) — typed fields are schema-stable across renames.
