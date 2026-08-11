# ADR-0125: Developer Priority Is Layout-Only

**Date:** 2026-08-11
**Status:** Accepted
**Related:** ADR-0124 (operator-driven layout planning), ADR-0116 (layered auto-projection)

---

## Context

ADR-0124 established that **layout planning is 100% the operator's call** via a
priority system (`WriteSpeed`, `ReadSpeed`, `StorageSpace`, `Balanced`). The
operator configures priorities globally, per-engine, and per-query through
`DeploymentConfig` YAML.

A subsequent change added `metaengine.WithLayoutPriority(p Priority)` as a
developer-facing `QueryOption`, allowing the developer to set a per-query
priority in code. This created a question: **should the developer's priority
influence engine ranking, or only layout selection within the assigned engine?**

### Two Separate Code Paths

The metaengine has two independent scoring paths that both consume `Priority`:

| Path | Function | What it decides | Inputs |
| --- | --- | --- | --- |
| **Engine ranking** | `planQuery` → `priorityFactor` | Which engine serves the query | Operator `PriorityConfig` only |
| **Layout selection** | `SelectLayout` via `priorityForQuery` | Embed vs. Normalize within the engine | Operator `PriorityConfig` **+** developer `WithLayoutPriority` |

`Store.priorityForQuery` resolves across five levels:

```
per-Query (operator config) → per-Query (developer WithLayoutPriority) → per-Engine → Global → Balanced
```

But `planQuery` (`planner.go:240`) calls `pc.priority.Resolve()` directly — it
never consults `QueryConfig.layoutPriority`. So the developer's
`WithLayoutPriority` affects layout but not engine routing.

This asymmetry was flagged as a "design gap" but is in fact the **correct
design boundary**, for the reasons below.

---

## Decision

**`WithLayoutPriority` is layout-only. It does not influence engine ranking.**

### Ownership Split

| Concern | Owner | API | Code Path |
| --- | --- | --- | --- |
| **Engine selection** ("where data lives") | **Operator** (deployment-time) | `DeploymentConfig.Priority` (YAML/koanf) | `planQuery` → `priorityFactor` |
| **Layout selection** ("physical shape within the engine") | **Developer** (code-time) + operator override | `WithLayoutPriority` (Go API) | `priorityForQuery` → `SelectLayout` |

The developer says: *"When this query's data is materialized, prefer the
read-optimized physical layout."* The operator says: *"This query is served by
PostgreSQL, not Pebble."*

### Why This Is Correct

1. **North star alignment.** The project's guiding principle states: *"where
   data lives is up to operators at DEPLOYMENT time."* Engine ranking IS "where
   data lives." Allowing a developer's Go code to influence engine routing would
   violate deployment-time isolation — a code change would override an operator's
   topology decision.

2. **Different decision granularity.** Engine ranking is a **deployment topology**
   decision (which infrastructure to provision). Layout selection is a **physical
   storage shape** decision (how to structure data within that infrastructure).
   These operate at different abstraction levels and change at different rates.

3. **Operator override preserved.** `priorityForQuery` resolution order ensures
   the operator's per-query config (`PriorityConfig.PerQuery`) still takes
   precedence over the developer's `WithLayoutPriority`. If `WithLayoutPriority`
   also influenced engine ranking, the operator would lose final authority over
   data placement.

4. **Consistent with ADR-0124's rejection of developer-driven layout.** ADR-0124
   Alternative A rejected developer-driven layout because it "puts storage intent
   on the developer." `WithLayoutPriority` is a narrow exception: it expresses a
   **layout preference** (read-optimized vs write-optimized), not a **storage
   intent** (which engine). The developer doesn't choose the engine; they only
   shape data within whatever engine the operator assigns.

### What `WithLayoutPriority` Actually Does

When the operator assigns a query to an engine, `SelectLayout` picks the
physical layout (Embed vs Normalize) based on the resolved priority:

- `ReadSpeed` → favor Embed (no joins on read)
- `WriteSpeed` → favor Normalize (smaller writes on child mutation)
- `StorageSpace` → favor Normalize (less duplication)
- `Balanced` → cost model picks the best tradeoff

The developer's `WithLayoutPriority(ReadSpeed)` means: *"on whatever engine this
query lands, prefer the embed layout."* It does NOT mean: *"route this query to
the fastest-reading engine."*

### Resolution Precedence (Final)

For **layout selection** (`SelectLayout` via `priorityForQuery`):

```
per-Query (operator PriorityConfig.PerQuery)
  → per-Query (developer WithLayoutPriority)
    → per-Engine (PriorityConfig.PerEngine)
      → Global (PriorityConfig.Global)
        → Balanced (default)
```

For **engine ranking** (`planQuery` via `priorityFactor`):

```
per-Query (operator PriorityConfig.PerQuery)
  → per-Engine (PriorityConfig.PerEngine)
    → Global (PriorityConfig.Global)
      → Balanced (default)
```

The developer layer exists only in the layout path.

---

## Alternatives Considered

### A. Wire `WithLayoutPriority` Into Engine Ranking

Allow the developer's priority to influence `priorityFactor` in `planQuery`, so
a developer setting `ReadSpeed` would bias engine selection toward read-optimized
engines.

**Rejected:** Violates the north star ("where data lives is up to operators").
Engine ranking is a deployment concern. A developer who writes
`WithLayoutPriority(ReadSpeed)` in Go code should not be silently overriding the
operator's engine topology. This would make deployments non-portable: the same
code would route to different engines depending on developer hints, breaking the
"operator deploys" invariant.

### B. Remove `WithLayoutPriority` Entirely

If the developer shouldn't influence engine ranking, maybe they shouldn't
influence layout either — make it 100% operator-controlled per ADR-0124.

**Rejected:** The developer has domain knowledge the operator lacks. The
developer knows that a specific query always reads the full aggregate (favor
Embed) or that a child collection mutates independently and frequently (favor
Normalize). Forcing the operator to configure this per-query in YAML for every
query in every service is impractical. `WithLayoutPriority` gives the developer
a **default** that the operator can still override — best of both worlds.

### C. Two Separate Priority Types

Introduce a `LayoutPriority` type distinct from `Priority`, making it
type-impossible to accidentally use one where the other belongs.

**Deferred:** Technically cleaner, but adds API surface for a distinction that
the naming (`WithLayoutPriority`) already communicates. Revisit if misuse is
observed in practice.

---

## Consequences

- **Positive:** Clear ownership: developer shapes data, operator places data. No
  ambiguity about who controls what.
- **Positive:** `WithLayoutPriority` name is honest — it affects layout, not
  engine routing. No false expectations.
- **Positive:** Operator's per-query override (`PriorityConfig.PerQuery`) still
  wins over developer's `WithLayoutPriority` for layout. Full operator authority
  preserved.
- **Positive:** Deployments remain portable. The same Go code runs identically
  whether the operator deploys on SQLite-only or PostgreSQL+Dgraph+Pebble.
- **Negative:** A developer who wants a query on a specific engine must ask the
  operator to configure `PriorityConfig.PerQuery` — there is no code-side API
  for engine pinning. This is intentional.
