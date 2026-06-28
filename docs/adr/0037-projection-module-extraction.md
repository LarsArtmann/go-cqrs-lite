# ADR-0037: Projection Interface Extraction from event/

## Status

Accepted — 2026-06-28

## Context

The `Projection` interface (`Name()`, `Handle()`, `EventTypes()`) lived inside the `event/` module. This was a layering inversion: projections are CONSUMERS of events, not producers. The `event` package's job is to define Event, persist it (Store/Sink/Source/Journal), and deliver it (Bus/Publisher/Subscriber). A consumer-side abstraction inside the producer module violates dependency-direction principles.

The interface had ZERO internal consumers in `event/` — it was a pure export for external packages (`storage`, `graph`, examples). This meant every projection implementation was coupled to `event`'s release cadence for no benefit to `event` itself.

Additionally, `stack.Materialize` — the KV/document projection tier — did NOT implement `event.Projection`. It returned `message.NoPublishHandlerFunc` (Watermill). This created a split brain: the library's own projection tier bypassed its own projection contract.

## Decision

1. **Extract** the `Projection` interface to a new `projection/` module (Layer 2, depends only on `event/`).
2. **Make `stack.Materialize` implement `projection.Projection`** by adding `Name()`, `Handle()`, and `EventTypes()` methods.
3. **Breaking change**: `event.Projection` → `projection.Projection`. Documented in CHANGELOG.

## Consequences

- All three projection tiers (Materialize, RelationalProjection, GraphProjection) now implement the same contract.
- `projection/` can grow a `Runner` (replay → live → checkpoint → dispatch) without bloating `event/`.
- External consumers must update imports from `event/v3` to `projection/v3` for Projection types.
- `event/` is now purely producer-side: Event types, Store interfaces, Bus, metadata, causality.

## Alternatives Considered

- **Keep in event/ with a type alias** — rejected: creates a circular dependency (`event → projection → event`) if alias delegates.
- **Put in a `readmodel/` module** — rejected: the interface is about event consumption, not storage. Projections can write to any backend (KV, SQL, graph).
