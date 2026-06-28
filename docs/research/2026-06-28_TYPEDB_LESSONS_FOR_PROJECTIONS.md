# What TypeDB Teaches Us About Projection Tiers

> **Status:** PROPOSAL — research synthesis, awaiting direction on 3 decisions
> **Date:** 2026-06-28
> **Sources:** [typedb.com](https://typedb.com/), [github.com/typedb/typedb](https://github.com/typedb/typedb), `docs/projection-tiers.md`, `docs/adr/0038-graph-projection-tier.md`, `graph/`, `storage/relational_schema.go`

---

## TL;DR

TypeDB is a strongly-typed, conceptual database built on type theory (the **PERA** model: Polymorphic Entity-Relation-Attribute). It looks like a graph but is *not* one — it has **n-ary relations with typed roles**, **inheritance**, a **schema that type-checks and enforces constraints**, and a **symbolic reasoning/inference engine**. It won the ACM PODS 2024 "Best Newcomer" award for its query language TypeQL. 3.x is a Rust rewrite (RocksDB storage, RAFT replication), MPL-2.0 licensed.

The lessons for **go-cqrs-lite's projections** are real but uneven. The single highest-leverage insight: **the graph tier is the least-typed of the three projection tiers, and TypeDB shows exactly what strong projection typing looks like.** Several other TypeDB ideas are deliberately *inapplicable* to a library (vs a database), and saying *why* sharpens our own design. Three concrete decisions fall out for the maintainer.

---

## 1. What TypeDB actually is

| Concept | TypeDB | go-cqrs-lite equivalent today |
| --- | --- | --- |
| Data model | **PERA**: entity types, **relation types with roles**, attribute types | Graph tier = property graph (binary edges, untyped props) |
| Schema | First-class, **closed-world**, queryable, type-checks at write + query time | Graph tier = none; relational = `RelationalColumn{Name,Type,Nullable}` |
| Relations | **N-ary** (`employment: employer + employee + witness`); roles are typed | `EdgeRef{Type, From, To}` — strictly binary |
| Inheritance | Single-inheritance subtyping; query "user" matches "admin" | None |
| Attributes | First-class, global, shared, subtypable | Per-projection copies |
| Querying | **TypeQL** — declarative pattern match, portable (runs anywhere TypeDB runs) | Graph reads = "native, not abstracted" |
| Derived data | **Reasoning engine**: `when {...} then {...}` rules + recursive functions, evaluated at query time | `Deriver` module designed, **zero code** (TODO C11) |
| Constraints | `@key`, `@card(1..n)`, `@values(...)`, `@regex(...)` | Relational: NOT NULL / PK only |

TypeDB achieves full read/write portability **because it owns the engine.** That is the crux for a library, addressed in §3.

---

## 2. Why TypeDB is *not* just a graph DB (and why that matters here)

ADR-0038 positions our graph tier as "writes portable (openCypher MERGE), reads native." TypeDB's core argument against property graphs is that binary edges + untyped labels + no schema lose three things: **expressive schema, abstraction/reuse, and data integrity by design.** Our graph tier currently exhibits all three losses:

- `NodeRef{Label string, KeyProp string, KeyValue any}` — labels are strings, keys are `any`.
- `GraphSink.MergeNode(ref, props map[string]any)` — properties are an untyped bag.
- No schema declaration; typos in a label or prop name create silent phantom nodes/edges (the *exact* bug class that `Row` column-name validation just fixed for the relational tier — see status `2026-06-28_10-04` A3).

In other words: **the graph tier today has weaker typing than our relational tier had before this morning's fix.** TypeDB is the existence proof that graph-shaped read models don't require abandoning strong types.

---

## 3. Reflection: ADOPT / ADAPT / REJECT

### ADOPT (cheap, library-compatible, high leverage)

**A1. A schema for the graph tier** (`graph.Schema`).
Model `NodeType{Label, KeyProp, KeyType, Properties[]PropertyType}`, `EdgeType{Type, FromLabel, ToLabel, Properties[]}`. Validate every `MergeNode`/`MergeEdge`/`SetNodeProperty` against it at the sink boundary — exactly as `ProjectionSink` now validates column names against `RelationalSchema`. Catches label/prop typos before they create phantom nodes. Aligns with project principle #4 ("Strong types over runtime checks") and mirrors the just-shipped relational validation. The `MemoryDriver` enforces it at runtime; real drivers translate it to their own DDL. **This is the #1 takeaway.**

**A2. Replace `map[string]any` props with a typed option where Go generics allow.**
`MergeNode[NodeOf[T]]` style is overreach, but a `graph.PropertySet` with a typed registry (prop name → value kind) plus a `Validate()` is straightforward and removes the worst of the `any` sprawl without breaking the openCypher MERGE portability (props still serialize to driver-native values).

### ADAPT (TypeDB's idea, but reshaped to fit a *library*)

**D1. Reasoning → use as the design reference for the unbuilt `Deriver` module.**
TypeDB's `when/then` rule engine is, conceptually, the same primitive as our planned `Deriver` (events → derived events/commands; TODO C11). The key TypeDB lessons to port: **rules are part of the schema**, **results are deterministic and idempotent**, and **chaining/recursion terminate because each derived instance is produced at most once.** We should NOT build an in-DB inference engine (a projection is already materialized derived data — the opposite philosophy), but the `Deriver` API should steal TypeDB's declarative `when/then` shape.

**D2. Portable reads — but only for the `MemoryDriver` we own.**
TypeDB's read portability comes from owning the engine. We can't impose a query language on Neo4j/Memgraph — ADR-0038's "reads native" is *correct* for cross-backend portability and must stand. **But the library *does* own the `MemoryDriver` engine**, and right now it exposes only `Snapshot()` (raw `*graphData`) — i.e., the in-memory graph has **no usable read API at all.** A typed pattern-match/traversal API for `MemoryDriver` only would (a) make the v3.x ship target actually queryable, (b) give handlers a portable read path in tests and single-process apps, (c) directly unblock the "graph tier has zero real consumers" problem (status `2026-06-28_02-45` E6). Asymmetry preserved, but the owned-engine half gets a real read surface.

### REJECT (and the reason matters)

**R1. N-ary relations / typed roles** — would break openCypher MERGE portability (the central thesis of the graph tier). Note as a documented limitation; model n-ary needs via the relational tier's junction tables (already supported). TypeDB gets n-ary by being its own engine; we don't.

**R2. A portable query language across real graph backends** — a TypeDB-scale research project, explicitly out of scope for a library. ADR-0038 stands.

**R3. Inheritance polymorphism in projections** — deep type theory, low ROI for a CQRS read-model library. YAGNI.

**R4. First-class/global shared attributes** — a storage-engine optimization, not a projection-tier concern. Lives inside the DB.

**R5. A `graph/typedb/` driver** — **blocked:** TypeDB has no official Go driver (Rust core; drivers for Rust/Java/Python/TS/C#/C++/C only). Note as future consumer-pull if a Go driver emerges.

---

## 4. The cross-tier insight TypeDB surfaces

The projection schema story is currently **split-brain**:

| Tier | Schema strength |
| --- | --- |
| Events | Rich — `catalog/` (JSON Schema, AsyncAPI, OpenAPI), `schema/Validator` |
| Relational | Medium — `RelationalSchema` + `RelationalColumn{Name,Type,Nullable}` + column validation |
| Document/KV | Implicit — `TypedStore[T,K]` (the Go type *is* the schema) |
| **Graph** | **None** — strings and `any` |

TypeDB's holistic "schema is a first-class, queryable, type-checking artifact" thesis suggests the graph tier is the outlier dragging the average down. A graph schema (A1) brings it to relational-tier parity; unifying *all* projection schemas into one metamodel would risk a god-object and is explicitly deferred.

---

## 5. Action plan (Pareto order)

### Tier 0 — do now (low risk, high value, no design sign-off needed)

1. **`graph.Schema` + sink validation (A1)** — mirrors the relational-tier pattern that shipped today. ~1–2h. Catches phantom-node bugs.
2. **`MemoryDriver` read API (D2)** — typed `Query(pattern)` / `Traverse(from, edgeType, depth)` / `Neighbors(...)`. Unblocks real consumers. ~2–3h.

### Tier 1 — design proposals (bring to maintainer)

3. **TypeDB as reference for `Deriver` module (D1)** — draft `when/then` API for the already-planned event→event/command derivation (C11). ~1 day to spec + prototype.
4. **Document the n-ary/roles limitation (R1)** in `docs/projection-tiers.md` + `graph/README.md`, with the "use relational junction tables" guidance. ~15 min.

### Tier 2 — explicitly note, don't build

5. Record R2–R5 as decided non-goals in this doc + `ROADMAP.md` so future agents/sessions don't re-propose them.

---

## 6. Decisions needed from maintainer

1. **Graph schema (A1):** adopt now, or defer until a real graph consumer materializes? *(Recommend: adopt — it's the same pattern you just shipped for relational, and removes the worst type-safety hole in the newest tier.)*
2. **MemoryDriver read API (D2):** ship as part of v3.x, or keep graph tier write-only until a Neo4j driver exists? *(Recommend: ship — otherwise the v3.x "graph tier is done" claim is hollow: nothing can read it.)*
3. **Deriver module (D1):** is now the time to start it, using TypeDB's rule model as the design north star?
