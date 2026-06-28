# ADR-0039: Graph Schema — Boundary Typing for Graph Projections

## Status

Accepted — 2026-06-28

## Context

The graph projection tier (ADR-0038) shipped with open-world semantics: any node label, any property name, any edge type was accepted at the sink. This is the weakest typing of the three projection tiers:

| Tier | Schema |
| --- | --- |
| Events | `catalog/` (JSON Schema, AsyncAPI) + `schema.Validator` |
| Relational | `RelationalSchema` + column-name validation |
| Document/KV | Implicit via generics (`TypedStore[T,K]`) |
| **Graph** | **None — strings and `any`** |

The most common graph-projection bug: a typo in a label or property name silently creates a phantom node that no query will ever find. The relational tier solved this exact problem with Row column-name validation.

TypeDB (typedb.com) is the existence proof that graph-shaped read models don't require abandoning strong types. Its PERA model provides typed entity types, relation types with roles, and attribute types with constraints — all enforced at write and query time.

## Decision

1. **Add `graph.Schema`** — declares node types (label, key prop, properties), edge types (type, endpoint labels, properties), and property types. Mirrors the shape of `storage.RelationalSchema`.

2. **Validate at the sink boundary** — `schemaSink` wraps `GraphSink` and validates every `MergeNode`/`MergeEdge`/`SetNodeProperty` against the schema before forwarding to the underlying sink. This is the same pattern as relational Row column-name validation.

3. **Schema is OPT-IN** — `WithSchema` on `NewGraphProjection`, `WithDriverSchema` on `NewMemoryDriver`. Nil schema = open-world (backward-compatible default). Setting a schema = closed-world at the projection boundary.

4. **Minimal constraint grammar** — only structural validation: label exists, key prop matches, property names declared, edge endpoints match. No `@regex`, `@values`, cardinality ranges, or inheritance.

5. **Schema contract in graphtest** — drivers that support schema validation pass 3 additional contract tests (rejects unknown label, rejects unknown prop, accepts valid write).

## What we adopted from TypeDB

- **Boundary-typing** — validate at the sink, not inside the database engine. TypeDB type-checks at write AND query time because it owns the engine. We validate at the sink because we're a library running against other databases.
- **Node types with key properties** — the concept that a node's identity is (label, key prop, key value), and that the key prop should be declared and enforced.
- **Edge endpoint constraints** — TypeDB's "intentional casting" ensures relations only connect entities of the right type. We enforce `FromLabel` and `ToLabel` on edge types.

## What we rejected from TypeDB

| Feature | Why rejected |
| --- | --- |
| N-ary relations / typed roles | Breaks openCypher MERGE portability (graph tier's thesis). Use relational junction tables. |
| Inheritance polymorphism | YAGNI for read models. No query-time subtype matching. |
| Full constraint grammar (`@regex`, `@values`, `@card`) | Database-engine concerns, not sink-validation concerns. |
| Reasoning/inference engine | A projection IS materialized derived data — opposite philosophy. Deriver module (ADR-0040) handles derivation at the application layer. |
| First-class/global shared attributes | Storage-engine optimization, not a projection concern. |

## Consequences

- Graph projection handlers can opt into type safety at the sink boundary.
- Typos in labels/props are caught before they create phantom nodes.
- The graph tier now has the same boundary-typing guarantee as the relational tier.
- Backward compatibility preserved: nil schema = open-world.
- The schema is intentionally NOT a type system — it's a validation guard. Future extensions should resist cargo-culting TypeDB's full constraint grammar.

### Property `Required` constraint — removed

An initial `PropertyType.Required bool` field was removed post-design. MERGE
semantics allow partial node updates (`MergeNode(ref, {"name": "new"})` to
update one property while preserving others). Enforcing `Required` at the
sink would break this valid use case — the sink only sees the incoming
partial props, not the merged result. This conflicts with the "minimal
constraint grammar" decision (point 4 above), so the field was removed as
YAGNI rather than shipped as a lie.

## Alternatives Considered

- **Make schema mandatory** — rejected: breaks backward compatibility, and open-world mode is useful for prototyping.
- **Full TypeDB-style schema** — rejected: DB-engine concerns, not library concerns. Scope creep.
- **Schema on the driver only** — rejected: the projection wraps the sink from any driver, so schema validation at the projection level works regardless of driver. Both `WithSchema` (projection) and `WithDriverSchema` (standalone driver) are supported.
