# ADR-0046: Seven-Tier Dependency Model

> Originally titled "Four-Tier Dependency Model." Renamed because the model
> actually describes seven tiers (0–6), not four. The "four" referred only to
> the conceptual grouping (primitives → core → infrastructure → composition),
> but the numbered tier table has always listed seven.

**Status:** Accepted  
**Date:** 2026-07-09

## Context

The project historically used a 7-layer system to describe module dependencies. A detailed
analysis revealed this system was fake:

1. `kv/` claims Layer 0 but depends on `codec/` — not a true leaf
2. `event/` claims Layer 1 but depends on Tier 2-4 modules via test deps that leak into go.mod
3. 38 of 48 modules depend on `codec/` — the true hub was invisible in the old system
   (note: the project now has 55 modules; the proportion is similar)
4. `command/` has a hard compile dependency on `event/` — violates CQRS separation

The 7-layer system provided false confidence that dependencies were well-stratified when
they were not.

## Decision

Replace the 7-layer system with an honest **four-tier model** (plus composition and tooling):

| Tier | Name               | Rule                |
| ---- | ------------------ | ------------------- |
| 0    | Primitives         | No internal deps    |
| 1    | Core Domain        | Depends on Tier 0   |
| 2    | Domain Utilities   | Depends on Tier 0-1 |
| 3    | Aggregation        | Depends on Tier 0-2 |
| 4    | Infrastructure     | Depends on Tier 0-3 |
| 5    | Composition        | Depends on Tier 0-4 |
| 6    | Tooling & Examples | Depends on all      |

See [`FOUR-TIER-MODEL.md`](../architecture-understanding/FOUR-TIER-MODEL.md) for the full
module-to-tier mapping and D2 diagram.

## Consequences

- `dependency budgets` and `check-layers` now validate against the real tiers
- v4 work (id+metadata extraction, kv context.Context, storage split) has a clear target
- The old "Layer N" references in AGENTS.md and docs are superseded
