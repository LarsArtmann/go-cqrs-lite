# ADR-0116: Layered Auto-Projection

**Date:** 2026-08-06
**Status:** Accepted
**Related:** ADR-0112 (ES-native metaengine), ADR-0111 (Record type)

## Context

The project's vision (from "Perfect Software Architecture for Business
Applications") is:

> Knowing ONLY the Commands + Events + Queries and their relations, we should be
> able to build superb Projections (Materialized Views).

Currently, consumers must manually write fold handlers for every projection:
`On(Event{}, func(e any) Result)`. This is boilerplate for the 80% of projections
that are simple CRUD-shaped (event creates/updates/deletes a view row). Only the
20% with complex logic (aggregation, graph traversal, conditional updates)
require hand-written folds.

## Decision

**Layered auto-projection: auto-generate simple projections from type inspection,
auto-route all projections (simple and complex) to optimal engines.**

### Layer 1: Auto-Generate (80% of projections)

The planner inspects event and query type definitions and auto-infers:

1. **ADT classification** — already works via fold return type inspection. Extend
   to infer from event/query struct field shapes:
   - Event `UserCreated{Name, Email}` + Query `GetUser{Name} → UserView` →
     auto-generate a Map fold keyed by user ID, value = `{Name, Email}`.
   - Event `TaskAssigned{TaskID, Assignee}` + Query `TasksByAssignee{Assignee}` →
     auto-generate a Graph edge (Assignee → TaskID).
   - Event `ItemCreated{Status}` + Query `CountByStatus{}` → auto-generate a
     Counter fold.

2. **Materialize-vs-replay** — already exists (ADR-0084). The planner decides
   whether to store the result or replay events on each read.

3. **Tombstone handling** — the planner sees `UserDeleted{}` events and
   auto-generates a removal fold (ADR-0114).

The consumer defines ONLY:

```go
type UserCreated struct { Name string; Email string }
type UserDeleted struct {}
type GetUser struct { ID UserID }
type UserView struct { Name string; Email string }
```

The planner generates the fold: `On(UserCreated{}, MapSet)`, `On(UserDeleted{},
MapDelete)`.

### Layer 2: Explicit Folds (20% of projections)

For complex projections, the consumer writes explicit fold handlers:

```go
On(OrderCompleted{}, func(r Record) Delta {
    // Complex aggregation: count by day, by category, track running total
    return Delta{"orders_completed": +1, "revenue": r.payload.Amount}
})
```

The consumer controls the logic; the planner still auto-routes to the optimal
engine.

### Layer 3: Auto-Route (100% of projections)

Both auto-generated and explicit folds are routed by the cost-based planner to
the best available engine. The consumer never touches infrastructure.

```
Consumer: defines Commands, Events, Queries + relationships
                              |
                    Planner (ES-native, ADR-0112)
                    +----------------------------------+
                    | Layer 1: auto-generate (80%)     |
                    |   inspect types → infer folds    |
                    | Layer 2: explicit folds (20%)    |
                    |   consumer writes handlers       |
                    | Layer 3: auto-route (100%)       |
                    |   cost-based engine selection    |
                    +----------------------------------+
                              |
                    Optimal projections
```

### How Auto-Generation Works

The planner uses Go reflection (or code generation via cqrs-gen) to inspect:

1. **Event struct fields** — which fields become projection keys/values
2. **Query struct fields** — which fields are read patterns (point lookup, scan,
   traversal)
3. **Event naming conventions** — `XCreated` → insert, `XUpdated` → update,
   `XDeleted` → delete (ADR-0114)
4. **Relationships** — which events reference which entities (via StreamType,
   foreign key fields)

When the inference is ambiguous, the planner emits a diagnostic asking the
consumer to provide an explicit fold for that query.

## Alternatives Considered

### A. Auto-route only (current metaengine)

**Rejected.** The consumer still writes every fold handler manually. The vision
is to auto-generate simple projections from types alone.

### B. Full code generation (cqrs-gen only)

**Rejected as sole path.** Code generation is rigid — it can't adapt at runtime
to available engines. The planner's cost-based routing must happen at runtime.
Code generation is complementary (compile-time type checking), not the primary
mechanism.

## Consequences

- **Positive:** Consumer writes ONLY domain types for 80% of projections.
  Massive DX improvement.
- **Positive:** The planner adapts to available engines at deployment time.
  SQLite-only deployment auto-generates SQLite projections. Dgraph deployment
  auto-routes graph queries to Dgraph.
- **Positive:** Explicit folds remain available for complex cases. No loss of
  control.
- **Negative:** Auto-generation requires convention-over-configuration (event
  naming, field matching). Ambiguous cases need explicit folds or configuration.
- **Negative:** Reflection-based inference has a startup cost. Code generation
  (cqrs-gen) can pre-compute folds at compile time for zero startup cost.
