# ADR-0024: Exported ID Marker Types

**Date:** 2026-06-18  
**Status:** Accepted  
**Deciders:** Lars Artmann, Crush

## Context

The `id/` module uses phantom types (`struct{}`) as type parameters for branded IDs via `id.Of[T]`. This is the standard Go phantom-type pattern for type-safe identifiers.

Prior to this ADR, only `AggregateMarker` was exported. When cqrs-htmx requested `UserMarker`, `CorrelationMarker`, and `RequestMarker` for `go-branded-id`'s `BrandNamer` integration, three more were exported — leaving `CausationMarker`, `ClientMarker`, `CommandMarker`, and `EventMarker` still unexported.

This created a **split brain in the type model**: half the branded IDs could be used with type-parameterized downstream tooling (BrandNamer, JSON formatters, custom marshalers), and half could not, with no principled distinction between the two groups.

## Decision

**Export all 8 phantom marker types.** The full set:

| Marker              | Branded Type    |
| ------------------- | --------------- |
| `AggregateMarker`   | `AggregateID`   |
| `UserMarker`        | `UserID`        |
| `CorrelationMarker` | `CorrelationID` |
| `RequestMarker`     | `RequestID`     |
| `CausationMarker`   | `CausationID`   |
| `ClientMarker`      | `ClientID`      |
| `CommandMarker`     | `CommandID`     |
| `EventMarker`       | `EventID`       |

### Rationale

1. **Consistency over arbitrariness** — `AggregateMarker` was already exported, establishing the precedent that markers are public API. Having a mix is worse than either extreme.

2. **Zero behavior change** — These are empty structs used only as type parameters. Exporting them changes only visibility, not runtime semantics.

3. **Enables the full tooling ecosystem** — `go-branded-id`'s `BrandNamer`, custom `json.Marshaler`/`TextMarshaler` implementations, logging formatters, and any generic type-parameterized tooling now works uniformly across ALL branded IDs.

4. **Additive and reversible** — No existing API is removed or renamed. If this proves wrong, the markers can be re-unexported in a major version bump.

## Alternatives Considered

- **Export only the 3 requested by cqrs-htmx** — Rejected because it leaves the API self-inconsistent: `BrandNamer[id.CommandMarker]()` compiles for `AggregateID` but not `CommandID`, which is confusing and invites identical feedback from future consumers.

- **Re-unexport `AggregateMarker`** — Rejected as a breaking change with no upside; `AggregateMarker` has been public since early versions.

- **Marker interface instead of phantom structs** — Rejected. Phantom types derive their power from being zero-cost type parameters, not from interface satisfaction. Adding an interface would complicate the type system without benefit.

## Consequences

- Downstream packages can reference any marker as a type parameter without forking or wrapping.
- The `id` package's public surface grows by 4 exported types (8 total markers).
- API stability golden file updated (1324 → 1328 exports).
