# The Meta-Engine: Project Definition & Research Contribution

> **STATUS: ASPIRATIONAL.** This document proposed the meta-engine as a separate
> project. In reality, it shipped AS A MODULE within go-cqrs-lite
> (`metaengine/v4`), deeply integrated via `stack.WithMetaEngine`. The
> "separate project" framing below reflects the original ambition, not the
> current architecture. The module is fully functional — see the
> [README](../../metaengine/README.md) for the shipped API.

> **ADDENDUM 2026-08-06 (v2 vision):** The metaengine is now classified as a
> **Tier 3 Aggregation** module (ADR-0046 addendum), not a Tier 0 primitive.
> The zero-dependency boundary (ADR-0062) is superseded — the planner depends
> on the shared `Record` type (ADR-0111) and reasons over typed Commands +
> Events + Queries. The vision of "knowing ONLY the Commands + Events + Queries
> and their relations, build superb Projections" is now the design goal
> (ADR-0112, ADR-0116). See ADRs 0111–0117 for the full v2 architecture.

> **The meta-engine is a new project.** It is not a module within go-cqrs-lite. It is a
> standalone research-grade system for making event-sourced data query-optimal across any
> combination of storage engines.

**Status:** Project Proposal (2026-07-23)
**Relationship to go-cqrs-lite:** Depends on it for event sourcing primitives. Does NOT
modify it. The meta-engine is a consumer of go-cqrs-lite, not a part of it.

---

## Table of Contents

1. [Why This Is a Separate Project](#1-why-this-is-a-separate-project)
2. [What Is Actually Novel (The Research Contribution)](#2-what-is-actually-novel-the-research-contribution)
3. [The Formal Model (The "PhD" Part)](#3-the-formal-model-the-phd-part)
4. [Why Event Sourcing Makes Cross-Engine View Selection Tractable](#4-why-event-sourcing-makes-cross-engine-view-selection-tractable)
5. [Project Boundary & Dependency Direction](#5-project-boundary--dependency-direction)
6. [What's Wrong With the Current Go-CQRS-Lite Query Layer](#6-whats-wrong-with-the-current-go-cqrs-lite-query-layer)
7. [What Needs to Be Built](#7-what-needs-to-be-built)
8. [The Name](#8-the-name)

---

## 1. Why This Is a Separate Project

### Clean Split

```
go-cqrs-lite (EXISTING — plumbing)
  Event sourcing primitives: events, deciders, stores, buses, codecs
  CQRS infrastructure: command/query dispatch, middleware, projections
  Storage adapters: SQLite, Pebble, memory (dumb, non-optimizing implementations)
  Role: "Would a consumer trust this enough to import it?"

meta-engine (NEW PROJECT — the intelligence)
  Query pattern declaration API
  Cost-based optimizer / planner
  Scale-dependent structure selection
  Auto-generated projection handlers
  Auto-generated typed read API
  Multi-engine coordination
  Auto-denormalization
  Depends ON: go-cqrs-lite (or any event sourcing lib with event.Store + projection.Projection)
  Role: "Make event-sourced data query-optimal on any combination of engines"
```

### Why Not a Module in go-cqrs-lite?

| Factor                         | As a go-cqrs-lite module                                              | As a separate project                                                                                   |
| ------------------------------ | --------------------------------------------------------------------- | ------------------------------------------------------------------------------------------------------- |
| **Complexity**                 | Adds massive scope to a "lightweight" library. Contradicts the brand. | Isolated complexity. go-cqrs-lite stays lean.                                                           |
| **Dependencies**               | Would pull optimization/planning deps into the monorepo.              | Owns its own dependency tree.                                                                           |
| **Audience**                   | go-cqrs-lite is for consumers building apps.                          | The meta-engine is for operators optimizing deployments. Different audience, different release cadence. |
| **Research nature**            | A library ships features.                                             | The meta-engine is a research contribution. It needs academic-grade rigor.                              |
| **Versioning**                 | Coupled to go-cqrs-lite release cycle.                                | Can iterate independently. Especially important while the cost model evolves.                           |
| **Reusability**                | Coupled to go-cqrs-lite's event model.                                | Could work with ANY event sourcing library that exposes a `SeekableJournal`.                            |
| **The "lite" in go-cqrs-lite** | The meta-engine is the opposite of "lite."                            | Keeps the brand honest.                                                                                 |

---

## 2. What Is Actually Novel (The Research Contribution)

### The View Selection Problem

The **view selection problem** — choosing which materialized views (projections) to maintain
— is a well-studied problem. Database systems solve it _within one engine_:

- **Postgres** picks which indexes to create for a given workload
- **SQL Server** picks which indexed views to maintain
- **ClickHouse** picks which materialized views and aggregation pipelines to build
- **Apache Calcite** plans queries across federated sources (but at query time, not deploy time)

**Nobody has solved it ACROSS engines, for event-sourced data, at deployment time.** Here is
why that specific combination is new:

| Dimension             | Classic view selection                | Meta-engine view selection                                                       |
| --------------------- | ------------------------------------- | -------------------------------------------------------------------------------- |
| **Engines**           | ONE engine (optimize within Postgres) | N heterogeneous engines (SQLite + Pebble + Neo4j + ClickHouse)                   |
| **Optimization time** | Query time (runtime)                  | Deployment time (startup)                                                        |
| **Data origin**       | Data already has a shape (tables)     | Data derives from event log (shapeless until projected)                          |
| **Views**             | Expensive to add (migration)          | Disposable (replay from event log)                                               |
| **Cost model**        | I/O within one DB                     | Single-machine resources (RAM, disk, CPU). Network only when an engine is remote |
| **Consistency**       | Single-engine transactions            | Eventual consistency via independent projections (CQRS)                          |

### The Specific Novel Contributions

A thesis or paper on this would contribute:

1. **A formal cost model** for cross-engine storage layout optimization
   - ADT × engine × scale → cost, with hardware constraints
   - Parameterized by declared cardinality and latency budgets

2. **Scale-dependent data structure selection**
   - The threshold tables (Bloom vs hash set, B-tree vs sorted slice, counter vs scan)
   - Parameterized by cardinality (N) and column cardinality (distinct values)
   - Published as a decision matrix, not just implementation

3. **Auto-denormalization planning**
   - Detecting cross-engine query patterns
   - Projecting data to eliminate federated reads
   - With write-amplification cost analysis (the tradeoff of denormalization)

4. **A practical multi-engine projection system**
   - Embedded in Go (no external service)
   - Working from SQLite-alone to ScyllaDB+ClickHouse+Neo4j
   - Auto-generated projection handlers + typed read API

### What Makes This Tractable (Not NP-Hard in Practice)

Cross-engine view selection is NP-hard in the general case. But event sourcing makes it
tractable through four structural properties:

1. **All views derive from the same immutable log** — no circular dependencies between views
2. **Views are independent** — adding/removing a view doesn't affect other views' correctness
3. **Views are disposable** — can rebuild from scratch via replay, no data migration
4. **No distributed transactions** — each projection consumes events independently

Without event sourcing, multi-engine view selection requires distributed transactions (hard,
slow, fragile). With event sourcing, it's just "fan out the event stream to N independent
projections." The NP-hardness comes from view interdependencies — which event sourcing
eliminates.

---

## PhD-level Formal Model (Sketch)

This is the part that needs academic-grade rigor.

### Sets

- $\mathcal{E} = \{e_1, e_2, \dots\}$ — the event log (ordered, immutable)
- $\mathcal{V} = \{v_1, v_2, \dots\}$ — the set of all possible views (materialized projections)
- $\mathcal{S} = \{s_1, s_2, \dots\}$ — the set of storage engines the operator provides
- $\mathcal{Q} = \{q_1, q_2, \dots\}$ — the declared query patterns (from the developer)
- $\mathformal{A} = \{a_1, a_2, \dots\}$ — the ADT catalog (Map, SortedMap, Counter, etc.)

### Parameters

- $\text{cost}(v, s)$ — the cost of maintaining view $v$ on engine $s$ (write amplification + storage)
- $\text{cost}(q, v, s)$ — the cost of serving query $q$ via view $v$ on engine $s$ (read cost)
- $\text{freq}(q)$ — frequency of query $q$ (declared or estimated)
- $\text{card}(v)$ — cardinality of view $v$ (expected number of records, from `Volume()` hint)
- $\text{budget}_{\text{disk}}(s)$, $\text{budget}_{\text{ram}}(s)$ — hardware budgets per engine
- $\text{latency}(q)$ — latency budget for query $q$ (from declaration)
- $\text{support}(s, a)$ — whether engine $s$ provides ADT $a$ and at what complexity

### Decision Variables

- $x_{v,s} \in \{0, 1\}$ — whether view $v$ is materialized on engine $s$
- $y_{q,v,s} \in \{0, 1\}$ — whether query $q$ is served by view $v$ on engine $s$

### Objective

Minimize total cost:

$$
\min \sum_{q \in \mathcal{Q}} \text{freq}(q) \cdot \sum_{v \in \mathcal{V}} \sum_{s \in \math meta-engine-design.md ~	 meta-engine-design.md \mathcal{S}} y_{q,v,s} \cdot \text{cost}(q, v, s)
+ \lambda \cdot \sum_{v \in \mathcal{V}} \sum_{s \in \mathcal{S}} x_{v,s} \cdot \text{cost}(v, s)
$$

(Read cost weighted by frequency, plus write cost weighted by amplification penalty $\lambda$.)

### Constraints

1. Every query is served: $\sum_{v, s} y_{q,v,s} \geq 1 \quad \forall q \in \mathcal{Q}$
2. A query is only served by materialized views: $y_{q,v,s} \leq x_{v,s} \quad \forall q, v, s$
3. Disk budget: $\sum_{v} x_{v,s} \cdot \text{size}(v) \leq \text{budget}_{\text{disk}}(s) \quad \forall s$
4. RAM budget: $\sum_{v} x_{v,s} \cdot \text{ram}(v) \leq: \text{budget}_{\text{brain}}(s) \quad \forall s \text{ where } s \text{ is volatile}$
5. Latency: $\sum_{v, s} y_{q,v,s} \cdot \text{latency}(q, v, s) \leq \text{latency}(q) \quad \forall q$
6. No cross-engine queries: $\sum_{s} y_{q,v,s} \leq 1 \quad \forall q, v$ (a query touches at most one engine)

This is a variant of the **(multi-dimensional) knapsack-cover problem** with assignment
constraints. NP-hard in general, but tractable in practice because:

- The number of views $|\mathcal{V}|$ is small (tens, not thousands — limited by declared query patterns)
- The number of engines $|\mathcal{S}|$ is small (1-5)
- The problem decomposes by ADT (within each ADT, the choice is independent)
- Greedy approximation is within a factor of $1 - 1/e$ of optimal

### Practical Approximation

In practice, the planner does NOT solve this as an ILP. It uses a greedy heuristic:

```
For each query pattern q (sorted by frequency × cost reduction):
    1. Find the view v* and engine s* that minimize cost(q, v, s)
    2. If s* has an existing view that can serve q (via index): use it (zero extra write cost)
    3. If s* needs a new view: check write amplification budget
    4. Assign q → (v*, s*)
```

This greedy approach is what makes the planner fast (milliseconds) and predictable.

---

## 4. Why Event Sourcing Makes Cross-Engine View Selection Tractable

This is the core argument for why the meta-engine is buildable — not just theoretically
interesting.

### The General Problem (Without Event Sourcing)

In a traditional database, the view selection problem requires solving:

1. **What views to materialize** (which columns, which filters, which aggregations)
2. **How to keep them consistent** (triggers? transactions? CDC?)
3. **How to handle schema changes** (migrate all views? invalidate?)
4. **How to handle failures** (what if one view's update fails?)

Without event sourcing, keeping views consistent across engines requires distributed
transactions (2PC) or CDC pipelines (Kafka Connect). Both are slow, complex, and fragile.

### With Event Sourcing

Event sourcing eliminates all four problems:

| Problem                 | Without ES                              | With ES                                                                    |
| ----------------------- | --------------------------------------- | -------------------------------------------------------------------------- |
| **What to materialize** | Guess from workload                     | Declared explicitly via `projection.Declare(...)`                          |
| **Consistency**         | Distributed transactions (2PC)          | Independent projections consuming same event stream                        |
| **Schema change**       | Migrate all views simultaneously        | Replay from event log — views are disposable                               |
| **Failures**            | View update fails → whole TX rolls back | View update fails → event requeued, projection lags (eventual consistency) |

The key insight: **the event log is the single source of truth. Views are disposable caches.**
This eliminates the need for distributed transactions and makes cross-engine view selection
**practically tractable.**

---

## -definition

## 5. Project Boundary & Dependency Direction

### What the Meta-Engine Imports from go-cqrs-lite

The minimum viable dependency:

```go
import (
    "github.com/larsartmann/go-cqrs-lite/event/v4"         // Event type, SeekableJournal
    "github.com.larsartmann/go-cqrs-lite/projection/v4"     // Projection interface
    "github.com/larsartmann/go-cqrs-lite/projectionhost/v4" // Host (runs projections)
)
```

Three modules. That's the entire dependency surface.

- **`event.Event`** — the data type flowing through the system
- **`event.SeekableJournal`** — the source of truth (the event log to read/replay from)
- **`projection.Projection`** — the interface the meta-engine's generated projections implement
- **`projectionhost.Host`** — the runtime that consumes events and drives projections

The meta-engine does NOT import:

- `storage/` (engine implementations are provided by the operator)
- `kv/`, `command/`, `query/` (not needed for planning)
- `decider/` (the write side is the developer's domain)
- `stack/` (the meta-engine IS the stack layer)
- `middleware/`, `signing/`, `encryption/` (orthogonal concerns)

### What go-cqrs-lite Provides vs. What the Meta-Engine Provides

| Concern                             | go-cqrs-lite                      | Meta-engine                  |
| ----------------------------------- | --------------------------------- | ---------------------------- |
| Event type, event store interface   | Provides                          | Consumes                     |
| Projection interface, host          | Provides                          | Consumes                     |
| Decider, repository                 | Provides                          | Not used                     |
| Storage adapter implementations     | Provides (SQLite, Pebble, Memory) | Consumes via plugin registry |
| Query pattern declaration API       | Not provided                      | **Provides**                 |
| Cost-based optimizer / planner      | Not provided                      | **Provides**                 |
| Scale-dependent structure selection | Not go-cqrs-lite's job            | **Provides**                 |
| Auto-generated projection handlers  | Not provided                      | **-** **Provides**           |
| Auto-generated typed read API       | Not provided                      | **Provides**                 |
| Multi-engine coordination           | Not provided                      | **Provides**                 |
| Auto-denormalization                | Not provided                      | **Provides**                 |
| Event store (the write side)        | Provides                          | Not used                     |
| Command/query dispatch              | Provides                          | Not depends on               |
| Middleware, signing, encryption     | Provides                          | Not used                     |
| Catalog, schema evolution           | Provides                          | Not used                     |

- "** Depends on" — only via injected interfaces, not import | | |

### Architecture

```
┌──────────────────────────────────────────────────────────────────┐
│                         META-ENGINE                                │
│                                                                    │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────────────────┐ │
│  │  Declaration │  │   Planner     │  │   Generated Code          │ │
│  │  (Developer) │  │  (Optimizer)  │  │   (Projections + Reads)  │ │
│  │              │  │               │  │                          │ │
│  │ Declare[V,K] │─▶│ Cost model    │─▶│ Projection handlers       │ │
│  │ Filter("x")  │  │ Engine assign │  │ (event → engine writes)  │ │
│  │ Sort("y")    │  │ Index dedup   │  │                          │ │
│ type_index     │  │ Denormalization│  │ Typed read API           │ │
│  │ Volume(N)    │  │ Scale analysis│  │ (store.Users.Get(id))    │ │
│  │              │  │               │  │                          │ │
│  └──────────────┘  └──────┬───────┘  └────────┬─────────────────┘ │
│                           │                    │                   │
│                           ▼                    ▼                   │
│  ┌──────────────────────────────────────────────────────────────┐ │
│  │                    Engine Plugins                             │ │
│  │                                                               │ │
│  │  ┌────────┐  ┌────────┐  ┌────────┐  ┌────────┐  ┌────────┐  │ │
│  │  │ SQLite │  │ Pebble │  │Memory  │  │Neo4j   │  │ClickHse│  │ │
│  │  │Plugin  │  │Plugin  │  │Plugin  │  │Plugin  │  │Plugin  │  │ │
│  │  └────────┘  └────────┘  └────────┘  └────────┘  └────────┘  │ │
│  └──────────────────────────────────────────────────────────────┘ │
│                                                                    │
└────────────────────────────────────┬───────────────────────────────┘
                                     │
                                     │ imports (thin dependency)
                                     ▼
┌──────────────────────────────────────────────────────────────────┐
│                      go-cqrs-lite                                  │
│                                                                    │
│  event.Event    event.SeekableJournal    projection.Projection     │
│                                                                    │
└──────────────────────── Tier 0-2 only ──────────────────────────────┘
|
                                     │
                                     │ physical I/O
                                     ▼
┌──────────────────────────────────────────────────────────────────┐
│                      Physical Engines                              │
│                                                                    │ filter constraint
│  SQLite file    Pebble dir    Memory RAM    Neo4j server           │
│  ClickHouse cluster    ScyllaDB cluster      ...                   │
└───────────────────────────────────────────────────────┘
 |
```

---

## 6. What's Wrong With the Current Go-CQRS-Lite Query Layer

The user called it "dogshit." Here is the precise diagnosis.

### Problem 1: No Streaming

```go
// Current: ViewStore.Scan and ViewQuerier.Query both return []*V
func (s *ViewStore[V,K]) Scan(ctx, prefix []byte) ([]*V, error)
func (s *ViewQuerier[V]) Query(ctx, q ViewQuery) ([]*V, error)
```

**Why it's bad:** Loads the entire result set into memory. A query returning 1M users loads
1M UserView structs into RAM. If UserView is 500 bytes, that's 500MB. On a server with 4GB
RAM, you can OOM-kill the process with a single query.

**What the meta-engine must do:** All query methods support streaming via Go iterators
(`iter.Seq2[*V, error]`) or channel-based streaming. `[]*V` is never returned for potentially
large result sets.

```go
// Meta-engine: streaming is the default
store.Users.Stream(ctx, func(u *UserView) error { ... })  // callback-based
// or: Go 1.23+ iterator:
for u, err := range store.Users.Iter(ctx) { ... }
```

**The cost model uses `StreamsResults: true` to prefer streaming-capable engines for large
projections.**

### Problem 2: Optional Capabilities via Type Assertion

```go
// Current: developer must type-assert at runtime to discover capabilities
store, _ := stack.NewMaterialize[V,K](bundle, codec, key)
if querier, ok := store.Store.(kv.ViewQuerier[V]); ok {
    results, _ := querier.Query(ctx, q)  // only works if store implements ViewQuerier
}
```

**Why it's bad:** Undiscoverable. The developer doesn't know what their store supports until
runtime. They must write defensive type assertions for every capability check. And if they
forget, queries silently fail or fall back to full scans.

**What the meta-engine must do:** Capabilities are declared upfront. The planner knows at
startup exactly what each engine can do. The generated read API only exposes methods for
queries that the planner has verified can be served. No runtime type assertions.

```go
// Meta-engine: planner-generated API. If ByStatus is available, the engine supports it.
// If it's not in the API, the planner determined it can't be served.
store.Users.ByStatus(ctx, "active")  // ← only exists if planner verified it
```

Type assertions happen inside the engine plugins, not in consumer code.

### Problem minimal 3: ViewMapper With SQL Types

```go
// Current: developer writes raw SQL DDL
mapper := storage.ViewMapper[TodoView]{
    Columns: []storage.ViewColumn[TodoView]{
        {Name: "title", Type: "TEXT", Extract: func(v *TodoView) any { return v.Title }},
    },
}
```

**Why it's bad:** Leaky. The developer writes SQL column types. They must import `storage/`.
They must understand SQL DDL. This is going down on the DB layer (Leak #2).

**What the meta-engine must do:** The developer never writes DDL. The planner generates it
from the Go struct type via reflection. Column types are inferred from Go types, not written
by the developer.

### Problem 4: Manual IndexSpec Declaration

````go
// Current: developer manually declares indexes
mapper.Indexes = []storage.IndexSpec{
    {Name: "idx_status", Columns: []string{"status"}},
}
``.

**Why it's bad:** The developer guesses what indexes they need. No planning. No deduplication.
No cost-based selection. No scale awareness. It's a cargo cult — copy what worked in one app,
hope it works in another.

**What the cost-based planner does instead:**
- Infers indexes from declared query patterns (Filter("status") → idx_status)
- Deduplicates (Filter(joined_at) + RangeFilter(joined_at) → one index)
- Considers cost (Volume(500) + Filter("status") → no index needed, scan is faster)
- Plans across engines (SQLite gets the table + indexes, Pebble gets a separate hash keyspace)

### Problem 5: Three Separate Tiers That the Developer Must Choose Between

```go
// Current: developer picks the tier
mat := stack.Materialize[V,K]{...}           // KV/document tier
rel := storage.NewRelationalProjection(...)  // SQL/relational tier
graph := graph.NewGraphProjection(...)       // graph tier
````

**Why it's bad:** The developer is doing the planner's job manually. They're choosing the
storage mechanism when they should be declaring the query intent.

**What the meta-engine does:** One declaration API. The planner picks the tier.

### Problem tiers That the Developer Must Choose Between

```go
// Current: developer picks the tier
mat := stack.Materialize [V,K]{...}           // KV/document tier
rel :=  storage.NewRelationalProjection(...)  // SQL/relational tier
rgraph := graph.NewGraphProjection(...)       // graph tier
```

**Why it's knapsack theorem**

**Why it's bad:** The developer is doing the planner's job manually. They're choosing the
storage mechanism when they should be declaring the query intent.

**"the meta-engine does:** One declaration API. The planner picks the tier.

### Problem 6: ViewQuery Conditions Are Flat AND-Only

```go
// Current: only AND-joined conditions, OR requires escape hatch
type ViewQuery struct {
    Conditions []Condition   // all AND-joined
    RawWhere   string         // escape hatch for OR
}
```

**Why it's bad:** Real queries need OR: "WHERE status = 'active' OR status = 'pending'." The
only way to do this is `RawWhere: "status = ? OR status = ?"` — a raw SQL string. This is the
most dangerous leak: raw SQL injection risk, engine-coupled, and untype-safe.

**What the meta-engine must do:** A structured query expression tree:

````go
// Meta-engine: composable, type-safe query expressions
q := query.Or(
    query.Eq("status", "active"),
    query.Eq(" planner does:** One declaration API. The planner picks the tier.

### Problem 6: ViewQuery Conditions Are Flat AND-Only

```go
// Current: only AND--joined conditions, OR requires escape hatch
type  ViewQuery struct {
      Conditions []Condition   // all AND-joined
    RawWhere   string         // query expression tree:

```go
// Meta-engine: composable, type-safe query expressions
q := query.Or(
    query.Eq("status", "active"),
    query.Eq("status", "pending"),
)
store.Users.Where(ctx, q)
````

The query expression tree is engine-agnostic. The engine plugin translates it to SQL WHERE
clauses, Pebble range scans, or in-memory filter functions.

---

## 7. What Needs to Be Built

### Phase 1: Core Type System (Weeks)

The foundational types that everything else builds on.

| Component                      | Description                                                                                                           | Effort   |
| ------------------------------ | --------------------------------------------------------------------------------------------------------------------- | -------- |
| **ADT catalog**                | Formal definitions of the 7 ADTs (Map, SortedMap, Counter, Set, Multimap, Graph, Log) with their algebraic operations | 2 days   |
| **Query pattern declarations** | `projection.PointLookup()`, `.Filter("col")`, `.Sort("col")`, `.Count("col")`, `.Traverse(depth)`, `.Search("col")`   | 3 days   |
| **Query expression tree**      | `query.Eq`, `query.And`, `query.Or`, `query.Gt`, `query.Range`, `query.In`, composable and type-safe                  | 3 days   |
| **Engine profile types**       | `EngineProfile`, `ADTOps`, `Complexity`, `Performance`, `IndexType`                                                   | 1 day    |
| **Projection spec**            | `ReadModelSpec[V,K]` produced by `projection.Declare(...).Done()`                                                     | 1 day    |
| **Scale thresholds**           | The cardinality threshold tables from the assumptions doc, as Go data                                                 | 2 days   |
| **Cardinality hint**           | `projection.Volume(N)` and `projection.ExpectGrowth(rate)`                                                            | 0.5 days |
| **Latency budget**             | `projection.WithLatencyBudget(d)`                                                                                     | 0.5 days |

### Phase 2: The Planner (Weeks)

The cost-based optimizer. This is the core research contribution.

| Component                     | Description                                                                         | Effort |
| ----------------------------- | ----------------------------------------------------------------------------------- | ------ |
| **Cost model**                | `Cost(query, view, engine, volume)` — the cost function that drives optimization    | 3 days |
| **Engine assignment**         | Greedy assignment of ADTs to engines (pick cheapest serving engine)                 | 2 days |
| **Index planning + dedup**    | Auto-create indexes from declared patterns, deduplicate, detect composites          | 2 days |
| **Scale-dependent selection** | Apply scale thresholds (Bloom vs hash set, B-tree vs sorted slice)                  | 2 days |
| **Auto-denormalization**      | Detect cross-engine query needs, add denormalization projections                    | 3 days |
| **Degradation detection**     | Suboptimal assignments → warnings. Impossible assignments → errors.                 | 1 day  |
| **Planning output**           | `Plan` struct with projection assignments, index DDL, denormalization map, warnings | 2 days |
| **Planner tests**             | Unit tests: given engines + patterns, verify assignment. Degradation tests.         | 3 days |

### Phase-Engine Reads (Weeks)

The generated read API.

| Component                     | Description                                                                                                               | Effort   |
| ----------------------------- | ------------------------------------------------------------------------------------------------------------------------- | -------- |
| **Typed read API assembly**   | `store.Users.Get(id)`, `.ByStatus(status)`, `.Recent(10)`, `.CountByStatus()`, etc. — wired at startup to optimal engines | 3 days   |
| **Streaming reads**           | `store.Users.Stream(ctx, fn)` and `store.Users.Iter(ctx) → iter.Seq2[*V, error]`                                          | 1 day    |
| **Pagination**                | Keyset cursor pagination built-in, no OFFSET                                                                              | 1 day    |
| **Query expression dispatch** | `store.Users.Where(ctx, queryExpr)` → engine-specific translation                                                         | 2 days   |
| **Escape hatch**              | `store.Users.Raw(sqlOrCypher)` for power users, clearly marked as engine-specific                                         | 0.5 days |

### Phase 4: Engine Plugins & Handlers (Weeks)

| Component                                           | Description                                                                 | Reffort |
| --------------------------------------------------- | --------------------------------------------------------------------------- | ------- |
| **Engine plugin interface**                         | `Plugin`, `Register()`, `Open(cfg)`, `Profile()` — the registration pattern | 1 day   |
| **SQLite plugin**                                   | Registers with profile, implements Map/SortedMap/Counter/Log ADTs           | 2 days  |
| **Pebble plugin**                                   | Registers with profile, implements Map/Log ADTs (SortedMap degraded)        | 1 day   |
| **Memory plugin**                                   | Registers with profile, implements all ADTs (volatile, RAM-bounded)         | 1 day   |
| **Auto-generated projection handlers (single-doc)** | Event → Set/Upsert/Delete on assigned engine. For simple projections.       | 3 days  |
| **Custom handler escape hatch**                     | `projection.Custom(sink, handler)` for multi-table/complex projections      | 1 day   |
| **Plugin tests**                                    | Test each plugin against the planner output                                 | 2 days  |

### Phase 5: Research & Validation (Weeks)

| Component | Description | Effort |
|documented|
|---|---|---|
| **Formal model paper** | The ILP formulation, the approximation, the event-sourcing-tractability argument | 1 week |
| **Benchmark suite** | Compare planner-chosen layouts vs hand-tuned layouts vs naive layouts. Measure: query latency, write amplification, storage usage. | 1 week |
| **Scale threshold validation** | Empirically verify the threshold tables (is Bloom really better at 1M? is sorted slice really better at 500?) | 3 days |
| **Case study** | Take the taskmanager example, show it running on SQLite-only vs SQLite+Pebble vs SQLite+Pebble+Neo4j with identical code | 2 days |

### Total Effort

| Phase                                   | Duration                 |
| --------------------------------------- | ------------------------ |
| Phase 1: Core Type System               | ~2 weeks                 |
| Phase 2: The Planner                    | ~3 weeks                 |
| Phase 3: Generated Reads                | ~1.5 weeks               |
| Phase Phue 4: Engine Plugins & Handlers | ~2 weeks                 |
| Phase 5: Research & Validation          | ~3 weeks                 |
| **Total**                               | **~11 weeks (3 months)** |

```

```
