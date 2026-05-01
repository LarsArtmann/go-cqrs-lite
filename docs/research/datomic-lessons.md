# What Datomic Teaches Us

> Analysis of Datomic's architecture and its implications for go-cqrs-lite

**Source:** https://www.datomic.com/

## Core Insights

### 1. Facts are Immutable, Never Updated In-Place

Datomic stores **datoms** — immutable atomic facts `[Entity Attribute Value Transaction]`. There's no UPDATE or DELETE — only assertions and retractions (which are themselves new facts). This is event sourcing at the database level.

**go-cqrs-lite already aligns here.** Events are append-only, never mutated. But Datomic formalizes _retractions_ as a first-class concept — a semantic "this is no longer true" without destroying the original fact. We don't have an equivalent. Today, "undo" is just another event type, but there's no explicit retraction pattern in the event model.

### 2. Database as a Value

Datomic's most radical idea: the entire database is an **immutable value**. `d/as-of` and `d/since` return a database _value_ at a point in time — not a copy, not a snapshot, but a view of the same indelible log. You can pass it around, query it, compare it — with zero coordination.

**Implication for us:** Our `event.Store` returns event slices, but there's no concept of an "event store as of version X" as a first-class value. A `Snapshot` is closer, but it's a materialized state, not a view of the log. We could add:

- `Store.AsOf(aggregateID, version)` — return all events up to a version
- `StoreView` — an immutable handle to the store at a point in time, safe for concurrent reads

### 3. Time as a First-Class Dimension

Every datom carries a transaction ID (`t`) — a monotonic global counter. This enables:

- **Time-travel queries**: "What was the state on 2024-03-15?"
- **Temporal joins**: "What was user X's role when event Y fired?"
- **Audit trails for free**: No extra tables, no event replay needed

**go-cqrs-lite has per-aggregate versioning but no global transaction ordering.** The event has `Version()` (aggregate-level) but no global `TransactionID`. Adding a monotonic `t` to events (or to the store's transaction log) would unlock temporal queries across aggregates.

### 4. Separation of Read and Write is Architectural, Not Just Pattern

Datomic's Transactor (single writer) vs. Peers (unlimited readers) is _infrastructure-level_ CQRS, not just code-level. Writes go through one serialized path; reads happen anywhere with a local cache of the immutable log.

**We already separate command/query dispatch, but the store layer could reflect this more explicitly:**

- Write path: `Transactor` — single-writer, ACID, ordered
- Read path: `Projection` / `Store.AsOf()` — read from any cached/snapshot view, no coordination

### 5. The Universal Relation: One Model, Many Access Patterns

Datomic's `[E A V T]` model is a single universal relation that supports **row-oriented, column-oriented, graph, and document** access patterns without schema impedance. The key insight: if your data model is generic enough, you don't need different stores for different queries.

**Our event model `[AggregateID Type Payload Version]` is already a universal relation for the write side.** But on the read side, we're projecting into typed structs. What if projections could be queried relationally without rebuilding Go structs? A `ProjectionStore` with attribute-level indexing (like Datomic's EAVT/AVET indexes) could serve multiple read patterns from one source of truth.

### 6. Retraction > Deletion

Datomic has `:db/retract` — a first-class operation meaning "this fact is no longer true" without destroying history. This is semantically richer than both:

- Hard delete (loses history)
- Soft delete (requires filtering on every read)

**We could add a `Retract` concept to the event model** — not a new event type per domain, but a structural operation: "retract event X" is recorded in the log, and projections can interpret it without domain-specific handling.

### 7. Schema as Data, Not as DDL

Datomic's schema is just more datoms — attributes about attributes. Schema evolves by adding new facts, not by ALTER TABLE. Any entity can acquire any attribute at any time.

**Our `catalog` module's reflection-based schema (`SchemaFromType[T]`) is close in spirit**, but it's read-only — it describes existing types. A Datomic-inspired evolution: allow the catalog to _define_ attributes independently of Go structs, enabling schema evolution without code changes.

## Concrete Opportunities for go-cqrs-lite

| Datomic Concept     | go-cqrs-lite Equivalent                    | Gap                              | Opportunity                                            |
| ------------------- | ------------------------------------------ | -------------------------------- | ------------------------------------------------------ |
| Datom `[E A V T]`   | Event `[AggregateID Type Payload Version]` | No global `T`                    | Add monotonic transaction ID to store                  |
| `d/as-of`           | `Store.Load()`                             | No time-travel                   | Add `Store.AsOf(id, version)`                          |
| Database as value   | N/A                                        | Store returns slices             | `StoreView` — immutable handle to log at point-in-time |
| Retraction          | N/A                                        | Only domain events               | First-class `Retract` in event model                   |
| Peer caching        | N/A                                        | No read caching                  | Projection caching with invalidation on new events     |
| Schema as data      | `catalog.SchemaFromType[T]`                | Schema describes, doesn't define | Schema-first attribute definitions                     |
| Functional `d/with` | N/A                                        | No dry-run transactions          | `Store.Apply(events) → StoreView` without persisting   |

## The Deepest Lesson

**If you never destroy information, every downstream capability — auditing, time-travel, reprocessing, debugging — becomes trivial rather than engineered.** go-cqrs-lite already believes this on the write side. Datomic shows what happens when you take it seriously everywhere.
