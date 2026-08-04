# ADR-0101: Backend Abstraction — N-Instance Metaengine

## Status

Accepted

## Date

2026-08-04

## Context

Source-of-truth stores and projection engines need different storage guarantees
(persistence vs rebuildability). A single metaengine instance with a mixed
engine pool would let the planner route the event log to Memory (cheapest),
losing all data on restart.

## Decision

The metaengine manages BOTH source-of-truth and projections via N independent
`metaengine.Store` instances. Source-of-truth instances use persistent-only
engine pools (the planner literally cannot route to Memory). Projection instances
use mixed pools (planner routes freely for cost optimization).

## Rationale

- One abstraction for all storage — events, commands, queries, projections.
- Hard invariants enforced structurally via engine pool constraints.
- LiveStore-proven dbEventlog + dbState split, generalized to N instances.

## Consequences

- The planner optimizes per-instance within constrained engine pools.
- Unified introspection: one topology shows all N instances.
