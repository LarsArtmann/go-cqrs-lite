# One-pager: `AggregateOn(fn, column, group)` — the plan-time aggregate seam

> **Status:** DESIGN MEMO — the SUPERB S28 next-step; awaiting owner ratification.
> **Date:** 2026-09-21 · **Origin:** routing-integration design findings 2026-09-11
> (F110/M20 of the
> [owner-unblock-trust plan](2026-09-20_17-40_SUPERB-owner-unblock-trust-pareto-plan.md))
> **Routing:** folds into the future routing/v5 ADR; adjacency to the G-T01 direction
> ruling — every new declarative surface must answer "is this developer declaration or
> operator config?"

## Problem

The planner prices reads per pattern (`EngineProfile.ReadCosts`,
`metaengine/engine.go:125`; `ReadAggregate` classified in `metaengine/query.go:150`)
but **never sees the aggregate SHAPE** — fn/column/group live in runtime calls
(`TypedReader.Count/Sum/Min/Max/Avg`, `metaengine/typed_reader_aggregates.go`), which
are opaque closures to plan time. Consequences:

- Operator-declared matviews (`MaterializedViewSpec{Collection, Fn, Column, GroupBy}`,
  `metaengine/materialized_view.go:24`) already serve **unfiltered scalar** aggregates
  at execution (`sqliteengine/aggregations.go:26`, `serveScalarMatView`), yet coverage
  cannot influence ROUTING or pricing: the planner cannot prefer the matview-holding
  engine for covered shapes, and cross-engine cost comparison treats them as O(N).
- Doctor cannot say "this query's aggregate is matview-served / uncovered".

## Proposal

Add an optional declarative hint, stamped on `QueryDecl` at construction:

```go
metaengine.Query[TopCustomers, Result]("top_customers",
    folds...,
    metaengine.AggregateOn(metaengine.MatViewSum, "amount", ""), // scalar
)
```

- `QueryDecl.Aggregate *AggregateSpec` (new field — additive, not breaking).
- Construction-time validation: two `AggregateOn` on one query → panic (Query
  convention, `metaengine/query.go:78`); fn must be a valid `AggregateFn`; Column
  required iff fn ≠ COUNT (mirrors `MaterializedViewSpec.Validate`).
- A hint, NOT a requirement: declaring `AggregateOn` with no matview anywhere is
  Doctor INFO, never an error.
- Planner: coverage check against engines' declared specs (new optional capability
  `MatViewSpecReporter interface { MatViewSpecs() []MaterializedViewSpec }`).
  Scalar-covered shapes price O(1) and route to the covering engine; everything else
  unchanged.

## Scope guard (first cut)

**Scalar-covered shapes only.** Grouped shapes (`GroupBy` set) stay O(N) with a Doctor
note: routing them is UNSAFE until upstream turso-go defect A (IVM silently loses
cross-transaction deltas — `docs/turso-go-ivm-fix-flip-runbook.md`) is fixed
upstream and the flip-runbook gate proves it.

## Alternatives rejected

- **Do nothing:** aggregates stay a plan-time blind spot; routing prefers engines on
  wrong costs.
- **Infer aggregates from result-type reflection:** v2-adjacent "Infer" territory;
  that surface was deprecated deliberately (G-T01 direction ruling pending) — do not
  grow it.
