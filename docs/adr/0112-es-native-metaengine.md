# ADR-0112: ES-Native Metaengine

**Date:** 2026-08-06
**Status:** Accepted
**Related:** ADR-0062 addendum (dep boundary), ADR-0111 (Record type), ADR-0116 (auto-projection)

## Context

The metaengine was originally designed as a generic storage planner with zero
dependencies. It sees events as `any` blobs and cannot reason about event types,
causality, stream relationships, or command-to-event derivation. This kneecapped
its core purpose: **the metaengine IS the Event Sourcing projection planner**.

The project's vision (from "Perfect Software Architecture for Business
Applications") is:

> Knowing ONLY the Commands + Events + Queries and their relations, we should be
> able to build superb Projections (Materialized Views).

A planner that receives `any` cannot do this. It needs to understand what it's
projecting.

## Decision

**The metaengine becomes ES-native. It depends on the `Record` type (ADR-0111)
and understands typed records.**

### What ES-Native Means

1. **Fold handlers receive `Record`, not `any`.** The planner can inspect
   `record.Type`, `record.StreamType`, `record.Version`, and `record.MetaData`
   to make intelligent projection decisions.

2. **The planner reasons about event relationships.** It can see that two events
   share a `CausationID` (same command produced them), detect tombstone events
   (ADR-0114), and understand stream boundaries.

3. **Auto-projection is layered (ADR-0116).** For simple cases (80%), the planner
   inspects event and query types and auto-generates projections. For complex
   cases (20%), the consumer writes explicit fold handlers. Both are auto-routed
   to optimal engines.

4. **Command Sourcing is supported.** The planner can fold over command history
   (Commands are Records too). This enables:
   - Full command+event replay ("what-if" time-travel)
   - Command audit projections
   - Dead-letter queues as projections over lifecycle event streams (ADR-0117)

5. **Materialize-vs-replay is a first-class decision.** The planner decides
   whether to materialize a projection (store the result) or replay events on
   each read, based on read/write rates and cost estimates.

### What ES-Native Does NOT Mean

- **Not coupled to a specific event store.** The planner works with any engine
  that implements the backend interfaces. The ES awareness is in the type system
  (Record), not in the storage layer.
- **Not opinionated about transport.** Events arrive as Records; the planner
  doesn't care if they came from a local store, a message bus, or gRPC.
- **Not a framework.** Consumers still compose their own stack. The planner is a
  library that takes Records and queries and produces optimized projections.

### Architecture

```
Consumer defines:  Commands, Events, Queries (typed structs + relationships)
                               |
                    Planner (ES-native, depends on Record)
                    +-----------------------------------+
                    | Inspects record types             |
                    | Infers ADTs from fold returns     |
                    | Decides materialize vs replay     |
                    | Auto-generates simple projections |
                    | Auto-routes to optimal engines    |
                    +-----------------------------------+
                               |
           +---------+---------+---------+---------+---------+
           v         v         v         v         v         v
        SQLite    Pebble    Dgraph    Postgres   Memory    Badger
        (KV/log)  (KV/log)  (graph)   (SQL/KV)   (all)     (KV/log)
```

### Dependency Rule

The metaengine core depends on the `Record` type (ADR-0111). This is the
**minimum** dependency needed to be ES-native. Additional dependencies are
justified when they make the planner better:

- `graph/` types: when the planner needs to route graph queries (ADR-0113)
- `command/` types: when command sourcing is used (future)

Modules are split by deployment concern (CGo isolation, heavy deps), not by
purity. See ADR-0062 addendum.

## Alternatives Considered

### A. ES-aware layer (keep core generic)

**Rejected.** A separate ES-aware module on top of a generic core duplicates the
adapter pattern that already exists (projectionadapter/). The translation loss
between generic and ES-aware is the problem, not the solution.

### B. Dual-mode (generic + ES entry points)

**Rejected.** Two entry points create a split brain: consumers must choose
between generic Plan() and ES-optimized PlanFromEvents(). The ES-native path is
strictly more capable; there is no value in a generic path that sees `any`.

## Consequences

- **Positive:** The planner can auto-generate projections from event/query type
  inspection (ADR-0116). This is the project's killer feature.
- **Positive:** Command sourcing, dead-letter queues, and audit trails emerge
  naturally from folding over command streams.
- **Positive:** Tombstone handling becomes a domain event concern (ADR-0114),
  not a fragile metadata mutation.
- **Negative:** The metaengine is no longer usable as a standalone generic
  planner. This is intentional — it was never the right abstraction.
- **Negative:** Phased migration required. The `any`-typed fold handlers must be
  updated to `Record`-typed handlers across all consumers and tests.
