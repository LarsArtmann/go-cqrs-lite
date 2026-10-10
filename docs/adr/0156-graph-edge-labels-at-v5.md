# ADR-0156: Graph Edge Labels at v5

- Status: Proposed
- Date: 2026-10-10
- Deciders: owner (graph-native adoption-closure plan V2, Wave 4)
- Related: [ADR-0114](0114-tombstone-as-domain-event.md) (deletion as domain events — `EdgeRemoval`),
  [ADR-0157](0157-engine-fleet-operation-two-level-story.md) (engine fleet operation),
  [`metaengine/types.go`](../../metaengine/types.go) (`Edge`), [`metaengine/graphadapter/doc.go`](../../metaengine/graphadapter/doc.go) (limitation + escape hatch)

## Context

The Graph ADT models edges as `metaengine.Edge{From any, To any}` — a flat, label-less pair.
Edge identity is `(collection, From, To)`; every graph-capable engine implements the same
uniform signature `GraphAddEdge(ctx, collection string, edge Edge) error` (verified across
memory, sqlite, pg, mysql, duckdb, badger, iroh, dgraph, and the graphadapter). Two consequences:

1. **No relationship types on edges.** A social domain with both `follows` and `blocks` cannot
   store both in one collection: adding `alice→bob` twice means the same edge, whatever the
   intent. Multi-relational graphs must split into one collection per relation (a per-relation
   fold), which works today and is the pattern `example/graph-native` and the COOKBOOK chapter
   document.
2. **No parallel edges / no edge properties.** `EdgeRemoval` (ADR-0114) retracts by the same
   flat identity — there is nothing finer to retract.

Native graph databases (Dgraph predicates, property-graph labels) model this natively, which
makes the flat ADT feel like a loss when targeting them.

## Decision

### D1 — v4.x: keep `Edge` label-less; the limitation is documented, not engineered around.

- One **collection per relation** is the canonical modeling rule (documented in the COOKBOOK
  Graph Patterns chapter, `readmodels.md` tier table, and advanced.md §6.13).
- Engines needing native expressiveness use the **`Driver()` escape hatch** on
  `metaengine/graphadapter` (`NewWithDriver`): the adapter still satisfies the Engine contract
  for routing and health while hand-written DQL/Cypher-style access uses the underlying driver
  directly.

### D2 — v5: revisit labeled edges ONLY with a named fleet consumer.

Extending `Edge` with a `Label` field looks additive but is not free anywhere:

- every engine's storage schema gains a label component (key encoding, table column, index) —
  a cross-engine schema migration for all 9 graph-capable engines;
- `EdgeRemoval` identity semantics must decide label-awareness (retract one label vs the whole
  pair) — a semantic choice, not a mechanical one;
- the fold contract (`func(E) Edge`) would grow a label discriminator every fold author must
  think about.

That cost buys nothing the fleet needs today (consumer scope: first-party only, and no
first-party app has asked for multi-relational single-collection graphs). Per the fleet-first
consumer rule, the trigger to reopen this ADR is a **named consumer need** — until then,
collection-per-relation is not a workaround; it IS the model.

## Consequences

**Positive:** zero schema churn in v4.x; the uniform signature stays uniform; the escape hatch
keeps native-graph operators unblocked; the limitation is honestly documented at every surface
that advertises graph-native reads.

**Negative:** domains that conceptually have one node-type graph with many relation types pay a
collection-proliferation cost (one collection + one fold per relation) and cannot answer
"all relations of alice" in one query without a union across collections.
