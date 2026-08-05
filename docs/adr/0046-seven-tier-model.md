# ADR-0046: Seven-Tier Dependency Model

> Originally titled "Four-Tier Dependency Model." Renamed because the model
> actually describes seven tiers (0–6), not four. The "four" referred only to
> the conceptual grouping (primitives → core → infrastructure → composition),
> but the numbered tier table has always listed seven.

**Status:** Accepted
**Date:** 2026-07-09
**Updated:** 2026-08-05 (module count, tier assignments, structural-vs-conceptual note)

## Context

The project historically used a 7-layer system to describe module dependencies. A detailed
analysis revealed this system was fake:

1. `kv/` claims Layer 0 but depends on `codec/` — not a true leaf
2. `event/` claims Layer 1 but depends on Tier 2–4 modules via test deps that leak into go.mod
3. 44 of 68 modules depend on `codec/` — the true hub was invisible in the old system
4. `command/` and `query/` each pull `event/` into their go.mod as `// indirect`
   via `storage/memory/` (test-only dep). Production code has zero `event/`
   imports — the `metadata/` extraction (ADR-0031) broke the real compile
   dependency. The indirect leak remains: `storage/memory/` transitively
   requires `event/`, `snapshot/`, and `query/` for its own purposes.

The 7-layer system provided false confidence that dependencies were well-stratified when
they were not.

## Decision

Replace the fake 7-layer system with an honest **seven-tier model** (0–6):

| Tier | Name               | Rule                  | Modules |
| ---- | ------------------ | --------------------- | ------- |
| 0    | Primitives         | No internal deps (or same-tier only) | 8       |
| 1    | Core Domain        | Depends on Tier 0     | 5       |
| 2    | Domain Utilities   | Depends on Tier 0–1   | 5       |
| 3    | Aggregation        | Depends on Tier 0–2   | 5       |
| 4    | Infrastructure     | Depends on Tier 0–3   | 23      |
| 5    | Composition        | Depends on Tier 0–4   | 9       |
| 6    | Tooling & Examples | Depends on all        | 13      |

**Total: 68 modules** across 69 `go.mod` files (68 modules + 1 root workspace
placeholder).

### Tier Assignment: Structural + Conceptual

Tier assignment uses two inputs:

1. **Structural** (sets the minimum tier): a module cannot be lower than its
   dependencies allow. `snapshot/` depends on `event/` (Tier 1), so it is at
   least Tier 2.
2. **Conceptual** (can raise the tier): a module's role can place it higher
   than its dependency floor. `otel/` has zero internal deps but is Tier 4
   (infrastructure). `catalog/` has zero deps but is Tier 6 (tooling).
   `idempotency/` has zero deps but is Tier 2 (domain utility).

Same-tier dependencies are allowed (e.g. `kv/` → `codec/`, both Tier 0;
`command/` → `event/`, both Tier 1) as long as there is no cycle.

See [`FOUR-TIER-MODEL.md`](../architecture-understanding/FOUR-TIER-MODEL.md) for
the complete module-to-tier mapping with every module listed.

## Consequences

- `nix run .#check-layers` validates against the real tiers
- `nix run .#check-layers` enforces per-module production dependency budgets
  (test-only packages like gomega, ginkgo, rapid are excluded from the count)
- v4 extraction work is complete: `id/`, `metadata/`, `retry/`, `idempotency/`
  are standalone modules; `kv/` has `context.Context`; `storage/` is split into
  focused sub-packages (`eventstore/`, `readmodel/`, `sql/`, `relational/`,
  `view/`, `migrations/`)
- The old "Layer N" references in docs are superseded by the seven-tier model

## Notable Tier-0 Exceptions

- **`metaengine/` is Tier 0 by design** (ADR-0062). The core planner has zero
  internal deps (stdlib + `database/sql` + `dedup/` only). The bridge to the
  CQRS event-sourcing world lives in `metaengine/projectionadapter/` (Tier 4).
  Conceptually it aggregates events into projections, but tiering is
  dependency-based.
- **`idempotency/` is Tier 2 conceptually** despite zero internal deps. It
  re-exports `github.com/larsartmann/go-idempotency` — the types (Store,
  MemoryStore, ErrDuplicate) are domain utilities, not primitives.
- **`catalog/` is Tier 6 conceptually** despite zero internal deps. It is a
  documentation generator — tooling, not a library primitive.

## Alternatives Considered

### Keep the old 7-layer system

Rejected: The old system was inaccurate. It claimed clean stratification that
did not exist. Modules were assigned layers by aspiration, not by actual
dependency structure.

### Merge tiers 1–3 into a single "Domain" tier

Rejected: The distinction between Core Domain (event/command/query), Domain
Utilities (schema/snapshot/projection), and Aggregation (decider/projectionhost)
is meaningful. Each tier has different stability guarantees and change rates.
Collapsing them would hide the real dependency distances.

### Pure structural tiering (deps only, no conceptual role)

Rejected: Would place `catalog/` and `otel/` in Tier 0 alongside `id/` and
`codec/`, which is misleading. A documentation generator is not a primitive,
even if it has zero deps. Conceptual role must be able to raise the tier.
