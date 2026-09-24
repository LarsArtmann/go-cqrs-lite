# ADR-0146: No Federated Query Engine — Replication Is the Cross-Domain Mechanism

**Date:** 2026-09-24
**Status:** Accepted
**Related:** ADR-0111 (Record type), ADR-0142 (universal storage substrate), ADR-0143 (journal survives resets), [meta-engine design](../planning/meta-engine-design.md)

## Context

The meta-engine design line has always carried one explicit negative:

> **NOT a federated query engine.** Federated engines (Calcite, Presto)
> route queries across databases at query time. The meta-engine optimizes
> the physical layout at deployment time so queries DON'T need to cross
> engines.

The data-mesh conformance assessment (2026-09-23) surfaced the question
again, because "federated computational governance" and the hub federation
story make "why not federated queries too?" a natural ask. An undecided
scope is worse than a recorded boundary, so this ADR promotes the design
line to a decision.

The cost-based planner already routes a query to the engine that holds the
data (live latency calibration, health-driven deactivation, catch-up
rebuilds). That is placement optimization INSIDE one deployment's boundary.
Federated query execution — joining across independently owned domains at
query time — is a different product with different failure modes:

- Cross-domain joins execute against OTHER domains' availability and schema
  drift; a domain owner cannot reason about their own latency or cost.
- Query-time federation hides data duplication that data-mesh governance
  exists to make EXPLICIT (contracts, owners, versions).
- The event journal already IS the sanctioned cross-domain data path:
  append-only, replayable, contract-checked at the boundary.

## Decision

**go-cqrs-lite will not grow a federated query engine.** Cross-domain data
flows are built from REPLICATION over each domain's journal:

1. A domain that needs another domain's facts subscribes to the producing
   domain's journal (outbox pattern; `watermill.CatchUpSubscriber` with a
   checkpoint store is the reference consumer — durable, crash-restart,
   broker-routable).
2. The replicated facts fold into the consuming domain's OWN projections,
   on engines the consuming domain's operator picked — the query then runs
   entirely inside one boundary, at one engine, with normal planner
   treatment.
3. The consuming dependency is declared as a coeffect (event subscription)
   and documented as a data-product INPUT (`catalog.DataProduct.Inputs`) —
   governance sees it, the exporter renders it, the linter checks it.

## Consequences

- Cross-domain "joins" are eventual: consumers see the producer's facts as
  of their replication checkpoint. This is the same trade every
  outbox-based mesh makes; synchronous cross-domain reads are an
  application-level concern (HTTP/gRPC against the owning domain's serving
  ports), not a planner feature.
- The planner's cross-engine story stays within ONE `system` deployment
  (one operator's engine fleet). Anything spanning deployments is
  replication.
- `metaengine` remains free to keep optimizing single-deployment placement
  without inheriting distributed-join semantics, partial-failure
  protocols, or cross-owner cost accounting.
- The data-mesh mapping doc records this as the standing answer to
  "federated queries?" — do not re-litigate without new OWNER direction.
