# DataFusion Lessons for go-cqrs-lite

**Date:** 2026-07-31
**Subject:** What Apache DataFusion teaches us about `metaengine/` — and how Event Sourcing makes the problem categorically harder

---

## What Is Apache DataFusion?

[Apache DataFusion](https://datafusion.apache.org/) is an **embeddable query engine** written in Rust that uses Apache Arrow as its in-memory columnar format. Unlike standalone databases (Spark, Trino, ClickHouse), it is a _library_ you build products _on top of_ — InfluxDB 3.0, dbt Fusion, Cloudflare R2, Arroyo, and many others embed it as their query core.

It offers SQL and DataFrame APIs that both lower to a single `LogicalPlan`, then optimize and execute via a **pull-based streaming** model. Three design principles guide it:

1. **Work out of the box** — world-class engine with minimal setup
2. **Customize everything via traits** — every layer is replaceable
3. **"Architecturally boring"** — industrial best practices over experimental techniques

### Architecture at a glance

```
SQL Text ──► [Parser] ──► AST ──► [SqlToRel] ──┐
                                                ├──► LogicalPlan
DataFrame API ──► [LogicalPlanBuilder] ─────────┘
                                                        │
                                                        ▼
                    [Analyzer] ──► [Logical Optimizer] ──► [Physical Planner] ──► [Physical Optimizer]
                                                                                        │
                                                                                        ▼
                                                                        SendableRecordBatchStream
                                                                    (pull-based, streaming, columnar)
```

**Key stages:** SQL/DataFrame → `LogicalPlan` (what) → logical optimization (rule-based rewrites) → `ExecutionPlan` (how, per-backend) → physical optimization (distribution, sorting, pushdown) → streaming execution.

---

## Why This Matters: metaengine Is a Query Engine

The direct relevance is the `metaengine/` module — labeled "THE STRATEGIC FUTURE" in AGENTS.md. metaengine is, structurally, a small query engine over CQRS projections. The parallels are already striking:

| DataFusion concept                    | metaengine equivalent today                             | Status              |
| ------------------------------------- | ------------------------------------------------------- | ------------------- |
| `LogicalPlan` / `ExecutionPlan` split | `Plan` + `LayoutPlan` / backend dispatch                | Partial — see Gap 1 |
| Predicate pushdown to data sources    | `FilterOnField` / `SortOnField` → SQLite `json_extract` | Aligned             |
| Streaming pull-based execution        | `StreamScan(ctx) iter.Seq2`                             | Aligned             |
| `Engine` / `TableProvider` traits     | `Engine`, `MapBackend`/`ScanBackend`/... interfaces     | Aligned             |
| Cost-based planning                   | `cost.go`, `Plan` result                                | Exists              |
| `EXPLAIN`                             | `explain.go`                                            | Exists              |
| Rule-composed optimizer               | (monolithic `planner.go`)                               | **Gap 2**           |

metaengine is already ~70% of a mini-DataFusion by accident.

---

## Concrete Learnings Worth Stealing

### 1. Make the logical/physical split explicit

DataFusion's power comes from one `LogicalPlan` (what to compute) lowering to many possible `ExecutionPlan`s (how, per-backend). metaengine already has `LayoutPlan` + `LayoutPlanner` doing access-path selection, but it is ad-hoc per-feature.

**The lesson:** treat `LayoutPlan` as the _physical plan_ and elevate it to a first-class lowered representation that _every_ backend can specialize, not just SQLite. That is what makes Pebble-vs-SQLite-vs-future-Postgres engine selection principled rather than incidental.

### 2. Compose the optimizer from independent, testable rules

DataFusion's optimizer is a _sequence of rewrite passes_ (`PushDownFilter`, `OptimizeProjections`, `EliminateFilter`, `EliminateCrossJoin`, `PropagateEmptyRelation`, ...), each unit-testable in isolation. metaengine's `planner.go` is currently monolithic.

**The lesson:** as pushdown/projection/layout rules multiply, split them into a rule pipeline (`[]PlanRewriteRule`). Each rule testable alone, each replaceable per-backend, and you get a clean `EXPLAIN` that shows the rewrite chain. This is the **highest-leverage single change** — it unlocks principled per-backend pushdown, testable rewrites, and a trustworthy `EXPLAIN`.

### 3. Statistics-driven planning

DataFusion ingests row counts and cardinality for join ordering. metaengine's cost model is structural (ADT → backend affinity) but not _statistical_.

**The lesson:** a `Stats{Rows, Card}` input to `Plan()` would let it choose full-scan vs. index vs. materialized rollup based on actual data shape — the difference between a planner and a _good_ planner.

### 4. The "architecturally boring" north star

DataFusion deliberately avoids novel research techniques in favor of proven industrial patterns. Worth adopting as metaengine's design ethos: be the boringly-correct planner, not the clever one.

### 5. Extension over fork

DataFusion's strongest cultural lesson: when something is not supported, the answer is "add a trait/extension point," never "fork it." metaengine's backend-interface split is already this — protect it.

---

## What Does NOT Transfer (from DataFusion)

- **Arrow columnar format** — Go's Arrow story is weak; CBOR/JSON self-describing envelopes are the right call for CQRS payloads
- **SQL parser** — go-cqrs-lite is programmatic/typed-first; a SQL surface would be scope creep
- **Distributed (Ballista)** — multi-node execution is out of scope; single-process embeddable is the niche

---

## How Event Sourcing Changes Everything

Event Sourcing inverts DataFusion's foundational assumption: **the table is not the truth; the event log is.** The table (projection/read model) is a _derived, disposable, temporally-versioned_ artifact. That changes the planning problem in escalating ways.

### What stays the same (the floor)

The structural lessons still hold:

- Rule-composed optimizer — still the right architecture; the rules just differ
- Trait-based extensibility — `Engine`/backend split is still correct
- Pull-based streaming — `StreamScan` is still right; the stream is now the event log
- "Architecturally boring" — still the right north star

DataFusion's lessons are the _baseline_. But ES makes the problem **richer and harder** than anything DataFusion has to plan over.

### What gets harder (the delta)

#### A. Materialization is now a planning decision, not a deployment fact

DataFusion assumes the table exists and asks "how do I scan it efficiently." ES asks the prior question: **"should this projection exist at all?"**

A query over events can be answered two ways:

| Strategy                                            | Cost model                                 | When it wins                |
| --------------------------------------------------- | ------------------------------------------ | --------------------------- |
| **Compute-on-demand** — replay events, fold, filter | O(events in stream) per query              | Rare reads, small streams   |
| **Materialize** — maintain a projection, query it   | O(1) per query + O(events) amortized write | Frequent reads, hot streams |

DataFusion has no equivalent — its tables just _are_. metaengine's cost model (`cost.go`) is currently structural (ADT → backend affinity); the **write/read ratio** is the missing input that decides materialize-vs-replay. This single dimension makes ES planning categorically harder than relational planning.

#### B. The "schema" is the fold function, not a fixed shape

DataFusion's `TableProvider` exposes a static schema. In ES, the read-model shape is _derived_ from how you fold events — and it shifts when the fold function changes or when upcasters transform old events (`schema.VersionedStore`).

The plan must reason over a **versioned, evolving** schema, not a static one. DataFusion's `LogicalPlan → TableScan` assumes column lists are stable; ES's equivalent must account for "event v1 means X, v2 means Y."

#### C. Statistics become more valuable AND more complex

DataFusion ingests row counts for join ordering. ES planning needs:

| Statistic                      | Planning purpose                                                |
| ------------------------------ | --------------------------------------------------------------- |
| **Write rate** per stream type | Replay cost grows with write rate                               |
| **Read rate** per projection   | Materialization payoff                                          |
| **Stream length distribution** | Long streams = snapshot territory                               |
| **Snapshot age**               | Incremental load cost = `LoadFromVersion(cachedVer)` delta size |
| **Tombstone ratio**            | How much state is dead (prunable)                               |

The **write/read ratio** is the king input DataFusion lacks entirely. It is the difference between a planner and a planner that knows _whether to build the table at all_.

### What is entirely new (ES-only, DataFusion cannot teach)

These are dimensions DataFusion literally does not model. They are why metaengine could be **more interesting than a mini-DataFusion** — it is solving a strictly harder problem.

#### 1. Temporal queries — "as-of" time travel

Reconstruct state at any point by replaying events to a version/timestamp. `EventSource.LoadToVersion` / `LoadToTimestamp` already exist. A query engine over ES should make `WHERE as_of = '2026-07-15'` a **first-class logical operation** with multiple physical realizations:

| Physical strategy                          | Cost      | Availability       |
| ------------------------------------------ | --------- | ------------------ |
| Snapshot at nearest version + delta replay | Cheap     | If snapshot exists |
| Full replay from zero                      | Expensive | Always available   |
| Cached projection as-of                    | Cheapest  | If materialized    |

DataFusion has no concept of this. It is a superpower ES gives you for free that a good planner should _expose and optimize_.

#### 2. Causal/temporal graph queries

"Show me every event caused by command X" — traversing causation chains (`metadata.causation`). This is why `graph/` exists. The planner must recognize when a query is:

- **Relational** (use `RelationalProjection`)
- **Graph** (use `GraphProjection`)
- **KV-document** (use `Materialize`)

**Three projection tiers, one planner** — DataFusion has one tier (relational).

#### 3. Replay semantics & processing mode

The same fold function runs in two modes — historical replay and live event processing (`ModeReplay`/`ModeLive`). The planner and projection host must distinguish:

- **Replay** — can be parallelized, idempotent-safe
- **Live** — requires ordering guarantees and dedup

DataFusion executes once over static data; ES executes **continuously** over a growing log.

#### 4. Projections are disposable

A projection can be dropped and rebuilt from zero (`projectionhost.Resettable`). This means planning can be **aggressive** about representation choices — nothing is irreversible because the source of truth is the log. DataFusion cannot assume this; dropping a Parquet file loses data. This unlocks bolder layout decisions.

#### 5. Idempotency as a planning constraint

At-least-once delivery means the same event may arrive twice. A materialization strategy must be idempotent (upsert, not append). This constrains which physical plans are _valid_, not just which are _fast_. DataFusion never reasons about this.

---

## Revised Recommendations (ES-aware)

| Recommendation          | DataFusion view                   | ES-revised view                                                                                                                                             |
| ----------------------- | --------------------------------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Rule-composed optimizer | Rules: push filter, prune columns | Rules: **push filter into fold**, decide materialize-vs-replay, snapshot boundary, tombstone pruning, tier selection (KV/relational/graph)                  |
| Statistics              | Row counts, cardinality           | **+ write rate, read rate, stream length, snapshot age** — write/read ratio is the decisive new input                                                       |
| Logical/physical split  | `LayoutPlan` = physical plan      | `LayoutPlan` is physical, **+ a temporal dimension** — the logical plan must carry `as_of` and the physical plan must choose snapshot+delta vs. full replay |
| Streaming               | `StreamScan` = pull-based scan    | Still right, but the stream is **append-only and potentially infinite** (live tailing), not a bounded dataset                                               |

---

## The Meta-Point

DataFusion plans over **state**. metaengine must plan over **derived state with temporal and causal dimensions** — state that:

- Can be reconstructed at any point in its history
- Has causal provenance (command → event chains)
- Evolves its own schema (upcasters, versioned events)
- Whose materialization is itself a cost/benefit decision (not a deployment fact)
- Is disposable and rebuildable from the log
- Must handle idempotent reprocessing

The DataFusion learnings transfer, but they are the **floor, not the ceiling**. The ceiling is something DataFusion never had to build: a planner that knows _whether to build the table_, _at what point in time_, and _from what cause_.

That is the strategically interesting frontier — and the argument for why metaengine deserves to be "the strategic future," possibly its own project.
