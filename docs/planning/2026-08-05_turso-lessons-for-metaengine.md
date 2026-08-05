# Turso "Postgres in Rust" Lessons for go-cqrs-lite

**Date:** 2026-08-05
**Source:** [We're building Postgres in Rust. Using the LLVM of databases](https://turso.tech/blog/a-new-modern-version-of-postgres-in-rust)
**Subject:** What Turso's "LLVM of databases" architecture teaches us about `metaengine/` — and the critical trap of taking the analogy too literally
**Related:**
[DataFusion lessons](2026-07-31_datafusion-lessons-for-metaengine.md),
[meta-engine-project-definition](meta-engine-project-definition.md),
[meta-engine-design](meta-engine-design.md),
[meta-engine-assumptions-and-query-planning](meta-engine-assumptions-and-query-planning.md)

---

## TL;DR

Turso's article describes a database architecture where one execution core (a bytecode VM called VDBE) serves multiple query-language frontends (SQLite SQL, Postgres SQL). The headline claim — "the LLVM of databases" — is seductive but **inverts** the metaengine's architecture.

The metaengine is the **mirror image** of Turso: where Turso has *many query languages → one storage engine*, the metaengine has *one event log → many storage engines*. Both use an intermediate representation in the middle, but the direction of multiplicity is opposite.

The single highest-value insight: **Turso's headline feature — live, auto-updating materialized views — is exactly what event sourcing + the metaengine already deliver, but smarter.** The metaengine doesn't just auto-update views; its planner *decides whether to materialize at all* based on cost. This is a strict superset of Turso's capability and should be the metaengine's positioning north star.

The single most dangerous trap: **don't build a bytecode VM.** Turso's VDBE is a massive runtime execution engine. The metaengine is a deployment-time planner that delegates execution to existing SQL engines. Conflating these layers would be a category error.

---

## Part 1: What Is Turso Actually Doing?

Beyond the marketing, Turso's architecture rests on five concrete ideas:

### 1. The VDBE Bytecode VM

SQLite is not "really" a database — it is a Virtual Machine. SQL compiles to a database-specific bytecode language (VDBE) with operations like "find something in a B-tree." Turso inherited this design. The VDBE is not exposed or specified, but it is the execution core.

Turso proved the VDBE is general-purpose enough to run Doom — a C-to-VDBE compiler produces bytecode that the database executes like a `SELECT`. Each frame is a result row.

### 2. "The LLVM of Databases" — Pluggable Frontends

Since Turso goes beyond SQLite, they made the frontend pluggable. Parse Postgres SQL → common AST → compile to Turso VDBE bytecode → execute on the same core. The claim: one modern, reliable core; many database frontends.

```
SQLite SQL ──► [parse] ──┐
                          ├──► common AST ──► [compile] ──► VDBE bytecode ──► [execute]
Postgres SQL ──► [parse] ──┘                                                    (one core)
```

### 3. Live Materialized Views

Postgres materialized views require manual `REFRESH MATERIALIZED VIEW` — users babysat cron jobs or faked live views with triggers for twenty years. Turso's views update themselves, live, automatically. This is their headline differentiator over Postgres.

### 4. Embedded Everywhere

Same engine runs as: a file, embedded in a browser (WASM), on edge, as a server with a wire protocol. One database, no operational weight.

### 5. Pareto Compatibility

Explicitly NOT aiming for 100% Postgres compatibility. "Compatible enough at the core functionality, but not really 100%." Some architectural improvements (live materialized views) make 1:1 compatibility undesirable.

---

## Part 2: The Architecture Comparison

### The Inverse LLVM

| Dimension          | Turso ("LLVM of databases")              | metaengine (what we are)                    |
| ------------------ | ----------------------------------------- | ------------------------------------------- |
| **Direction**      | N query languages → 1 storage engine      | 1 event log → N storage engines             |
| **IR**             | VDBE bytecode (runtime execution)         | ADT classification (deployment-time planning) |
| **Multiplicity**   | Many frontends, one backend               | One frontend, many backends                 |
| **When**           | Runtime (query compilation)               | Deployment-time (plan once, execute forever) |
| **What it owns**   | Storage, execution, B-trees, MVCC         | Planning, cost estimation, engine selection |
| **What it delegates** | Nothing (it IS the storage engine)     | Everything (SQL engines do the actual work) |

This is the crucial distinction. Turso builds the engine. The metaengine picks the engine.

### The IR Parallel (ADTs ↔ VDBE)

Both systems use an intermediate representation between "what the user declares" and "how it executes":

| Turso                          | metaengine                                    |
| ------------------------------ | --------------------------------------------- |
| SQL text (frontend)            | Fold return types (frontend)                  |
| Common AST                     | ADT classification (Map, Set, Counter, ...)   |
| VDBE bytecode (IR)             | ADT × Engine × Layout cost matrix (IR)        |
| VDBE execution (one core)      | Engine backend dispatch (Memory/SQLite/DuckDB/...) |
| Runtime query optimization     | Deployment-time plan + SQL engine's own optimizer |

The metaengine's ADT classification is its "bytecode" — but it's a **compile-time IR** that selects which backend interface to use, not a **runtime IR** that executes operations. The metaengine generates SQL DDL (LayoutPlan) and delegates to the SQL engine's own runtime optimizer for actual query execution.

This is a layered, not competing, design: the metaengine's deployment-time planner sits ABOVE the SQL engine's runtime query planner. They don't conflict — they optimize at different timescales.

---

## Part 3: What We SHOULD Learn

### Lesson 1: Claim "Auto-Updating Materialized Views" as the Value Proposition

**Turso's headline feature is our native capability.**

Turso ships auto-updating materialized views as their killer differentiator over Postgres. The metaengine + event sourcing delivers this natively: projections ARE auto-updating materialized views, driven by the immutable event log.

But the metaengine goes further — the `materialize-vs-replay` planner rule (`materialize.go`) decides WHETHER to materialize based on observed workload stats:

```
replay_cost(q)      = read_rate × avg_stream_length × fold_cost_per_event
materialize_cost(q) = write_rate × fold_cost_per_event + read_rate × query_cost_per_lookup
```

Turso's materialized views ALWAYS update. The metaengine can choose NOT to materialize and replay on demand when that's cheaper. **This is a strict superset of Turso's capability.**

**Action:** Make "cost-based auto-updating materialized views" the metaengine's positioning north star. The design docs already mention materialized views in passing; this should be the headline, not a footnote. The materialize-vs-replay feature should be described as "the intelligent version of what Turso just shipped as their flagship feature."

### Lesson 2: Document ADTs as "The IR" Explicitly

Turso's VDBE is documented as a first-class architectural concept. The metaengine's ADT classification is equally central but hasn't been framed as an "intermediate representation."

The DataFusion lessons doc started this framing (mapping `LogicalPlan` ↔ `Plan + LayoutPlan`). The Turso parallel adds another angle: ADTs are the **planning IR** that makes multi-engine routing possible. Without the ADT abstraction, the metaengine would need N×M adapters (N query patterns × M engines). With it, the problem decomposes to N + M (N pattern→ADT classifications + M ADT→engine implementations).

**Action:** Add an "Architecture: The ADT IR" section to the meta-engine-design.md that explicitly frames:
- Fold return types → ADT classification (the "parse" step)
- ADT → Engine backend dispatch (the "compile" step)
- Cost matrix as the "optimizer"
- LayoutPlan DDL as the "codegen" step

### Lesson 3: Strengthen Cross-Engine Oracle Testing

Turso's reliability story: DST, Antithesis, Oracle testing, fuzzing, formal methods. The metaengine's equivalent is `adttest.RunMatrix` — cross-engine parity testing across 10 ADTs.

The metaengine already has the right instinct: the Memory engine is the reference implementation. Every other engine should produce identical results for the same operations. But `RunMatrix` is scenario-based, not property-based.

**Action:** Add property-based cross-engine testing using `pgregory.net/rapid`:
- For each ADT, generate random operation sequences
- Apply to Memory engine (the Oracle)
- Apply to SQLite/Pebble/DuckDB/Postgres engines
- Assert identical final state

This is the metaengine's version of Turso's Oracle testing. The planner is deterministic (deployment-time), so DST is less relevant — but the engines' runtime behavior must be proven equivalent.

### Lesson 4: "Inside, Databases Are Not That Different" Validates Multi-Engine Routing

Turso's key insight: "Look close enough, and every SQL database is just a fancy collection of B-Trees with a bunch of Indexes. Those are differences in how the bytes are arranged on disk, not in the fundamental operations."

The metaengine formalizes this observation. The ADT × Engine × Layout cost matrix IS the formalization: every ADT CAN be served by every engine, the question is **at what cost**. The "degraded" classification (brute-force fallback when an engine lacks native support) is the metaengine's version of "compatible enough, not 100%."

**Action:** Quote this Turso observation in `meta-engine-assumptions-and-query-planning.md` as external validation of the core thesis. The design doc's central claim — "Every query pattern CAN be served by every engine — the question is at what cost" — is exactly what Turso discovered from the opposite direction.

### Lesson 5: Deployment-Time Planning Is an Advantage, Not a Limitation

Turso compiles SQL at query time. The metaengine plans at deployment time. For event-sourced systems where query patterns are declared upfront (via `Query[Q,R](...)`), deployment-time planning is strictly better:

- No query-time optimization overhead
- No optimizer warmup or cache misses
- Deterministic, reproducible plans (SerializablePlan)
- Plans can be diffed, pinned, and reviewed before deployment

Turso's VDBE compiles at runtime because SQL queries are ad-hoc. The metaengine's queries are declared, not ad-hoc — this is a structural advantage of the event-sourcing model.

**Action:** Frame deployment-time planning as a deliberate design choice in the project definition, not just a current limitation. The SerializablePlan (JSON serialize/diff/pin) is the metaengine's version of Turso's `EXPLAIN` — but it happens before deployment, not at query time.

### Lesson 6: The Pareto Principle for Engine Coverage

Turso explicitly accepts "compatible enough, not 100%" for Postgres. The metaengine should adopt the same stance for engine ADT coverage.

Not every engine needs to implement every ADT optimally. The "degraded" classification with brute-force fallback is the right design — it's the metaengine's version of Pareto compatibility. A Vector search on SQLite (brute-force O(N)) is "compatible enough" when N is small.

**Action:** Document this as an explicit design principle. The current `degradedADTRule` emits a DEGRADED diagnostic — this should be framed as "expected behavior, not a deficiency." Add to the design doc: "The Pareto principle applies to engine coverage: every ADT should be servable by every engine, but optimal service is engine-specific."

---

## Part 4: What We Should NOT Learn

### Anti-Lesson 1: Do NOT Build a Bytecode VM

The VDBE is Turso's core engineering investment — years of work by a funded team. It's a runtime execution engine that interprets database operations.

The metaengine does NOT need this. It delegates execution to SQL engines that already have their own runtime optimizers:
- SQLite has its own VDBE
- DuckDB has vectorized columnar execution
- Postgres has its executor

Adding a bytecode layer to the metaengine would duplicate work that SQL engines already do better. The metaengine's value is in DECIDING which engine to use, not in REPLACING the engine.

**The trap:** The "LLVM of databases" framing is seductive. LLVM has an IR (LLVM IR) and a backend. The metaengine has an IR (ADTs) and backends (engines). The analogy seems perfect — but LLVM's IR is a runtime execution format, while the metaengine's ADTs are a compile-time classification. These are fundamentally different layers.

### Anti-Lesson 2: Do NOT Become a Storage Engine

Turso IS a storage engine: B-trees, pages, MVCC, WAL, compaction. This is the hardest part of database engineering.

The metaengine is a PLANNER that routes to existing storage engines. It should NEVER implement:
- B-trees or page management
- MVCC or transaction isolation
- WAL or crash recovery
- Compaction or storage formats

The Memory engine's brute-force implementations exist for TESTING and as the reference Oracle — not as production storage. The production path always delegates to SQLite/DuckDB/Postgres/Pebble.

### Anti-Lesson 3: Do NOT Build a Wire Protocol or Server

Turso implements the Postgres wire protocol and a server. The metaengine is a Go library/SDK — no transport opinion. This is by design (Design Principle #1: "Library, not framework").

The `transport/http` and `transport/grpc` modules exist for CQRS command/query dispatch, not for database wire protocol. The metaengine has no server story and shouldn't develop one.

### Anti-Lesson 4: Do NOT Chase Browser/WASM Deployment

Turso runs in browsers via WASM. The metaengine is a Go library — it runs wherever Go runs (server, CLI, embedded devices with Go support).

Browser/WASM support would require:
- Compiling Go + a storage engine (modernc.org/sqlite) to WASM
- A completely different deployment story
- A different project entirely

This is not the metaengine's problem to solve. If a consumer wants browser-local event sourcing, they can use the Memory engine in a Go-WASM build — but the metaengine shouldn't optimize for this case.

### Anti-Lesson 5: Do NOT Take the "LLVM of Databases" Branding Literally

The "LLVM of databases" metaphor works for Turso because they have ONE execution engine with MANY frontends. The metaengine has ONE event log with MANY engines. These are inverse architectures.

Calling the metaengine "the LLVM of event-sourced storage" would be misleading. The metaengine is better described as:
- **"The storage optimizer for event-sourced systems"**
- **"The cost-based query router for CQRS read models"**
- **"The planner that decides where each projection lives"**

These descriptions are accurate and differentiated. The LLVM metaphor implies a compilation pipeline; the metaengine is a routing/optimization pipeline.

### Anti-Lesson 6: Do NOT Pursue 100% Feature Parity Across Engines

Turso explicitly abandons 100% Postgres compatibility. The metaengine should not pursue 100% ADT feature parity across engines.

The DuckDB engine will never be as good at point lookups as Pebble. The Pebble engine will never be as good at analytical aggregations as DuckDB. That's the POINT of the cost matrix — it routes each query to its optimal engine. Pursuing parity would flatten the differentiation that makes multi-engine routing valuable.

The "degraded" classification is the honest answer: "this engine CAN serve this ADT, but suboptimally — here's the cost." That's more useful than pretending all engines are equal.

---

## Part 5: Concrete Action Items

| # | Action                                                                                          | Priority | Effort  |
| --- | ----------------------------------------------------------------------------------------------- | -------- | ------- |
| 1   | Rewrite the metaengine positioning to lead with "cost-based auto-updating materialized views"  | High     | Low (docs) |
| 2   | Add "The ADT IR" architecture section to meta-engine-design.md                                  | Medium   | Low (docs) |
| 3   | Add property-based cross-engine Oracle testing (rapid) to adttest                               | Medium   | Medium  |
| 4   | Quote Turso's "inside, databases are not that different" in assumptions-and-query-planning.md   | Low      | Trivial |
| 5   | Frame deployment-time planning as an advantage in project-definition.md                         | Medium   | Low (docs) |
| 6   | Document Pareto engine coverage as an explicit design principle                                 | Low      | Low (docs) |
| 7   | Add an "Anti-Patterns" section to design docs: no bytecode VM, no storage engine, no wire proto | Medium   | Low (docs) |

---

## Part 6: The Deeper Insight — Why Event Sourcing Makes This Better

Turso's materialized views are triggered by writes to tables. The view update mechanism is internal to the database — it's a trigger-like system that re-runs the view query when source data changes.

Event sourcing gives the metaengine a structural advantage Turso cannot match:

1. **The event log is the single source of truth.** Every projection derives from the same immutable log. There's no "source table changed, update the view" race condition — the event IS the change.

2. **Projections are disposable and rebuildable.** If a materialized view is corrupt, you replay the event log. Turso can't do this — their materialized views are mutable state, not derived state.

3. **The materialize-vs-replay decision is per-query, not global.** Turso either materializes a view or doesn't. The metaengine can serve the SAME query from a materialized projection in production AND replay it on-demand in a test — because the event log is always there.

4. **Temporal queries are free.** The event log preserves history. Turso's materialized views show current state only. The metaengine's VersionedStorage / `ExecuteAsOf` can answer "what was this value at time T?" by replaying to a point in time — something no materialized view system can do.

This is why the metaengine is "THE STRATEGIC FUTURE" of this project: it's not just a storage planner, it's a planner that leverages the unique properties of event sourcing to deliver capabilities that traditional databases (even Turso) structurally cannot offer.

---

## Appendix: Turso Feature → Metaengine Mapping

| Turso Feature                    | Metaengine Equivalent                              | Status              |
| -------------------------------- | -------------------------------------------------- | ------------------- |
| VDBE bytecode VM                 | ADT classification + cost matrix                   | Exists (different layer) |
| Pluggable frontends (SQL dialects) | Fold return type → ADT inference                 | Exists (different direction) |
| Live materialized views          | Projections + materialize-vs-replay planner rule   | Exists (superset)   |
| Embedded everywhere              | Memory + SQLite engines (in-process)               | Partial (no browser/WASM) |
| MVCC concurrent writes           | Delegated to SQL engines                           | N/A (not our layer) |
| Rich type system                 | Go generics + branded IDs                          | Exists (different mechanism) |
| DST / Antithesis testing         | adttest.RunMatrix + rapid property tests           | Partial             |
| Oracle testing                   | Memory engine as reference + cross-engine parity   | Partial             |
| Pareto compatibility             | Degraded ADT classification                        | Exists              |
| Wire protocol (Postgres)         | N/A — library, not server                          | N/A                 |
| Extension system (WASM)          | N/A — Go plugin / interface segregation            | N/A                 |
