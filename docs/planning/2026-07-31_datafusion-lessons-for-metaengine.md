# DataFusion Lessons for go-cqrs-lite

**Date:** 2026-07-31
**Subject:** What Apache DataFusion teaches us about `metaengine/` — and how Event Sourcing makes the problem categorically harder
**Related:**
[meta-engine-project-definition](meta-engine-project-definition.md),
[meta-engine-design](meta-engine-design.md),
[meta-engine-assumptions-and-query-planning](meta-engine-assumptions-and-query-planning.md)

---

## TL;DR

metaengine is already ~70% of a mini-DataFusion *by accident*. The DataFusion learnings are real and worth stealing — but they are the **floor, not the ceiling**. Event Sourcing adds temporal, causal, materialization, and idempotency dimensions that DataFusion literally cannot model, making metaengine's planning problem strictly harder than anything a relational query engine has to solve.

The single highest-leverage change: **split the monolithic planner into composable rewrite rules** (DataFusion's core architectural win). Everything else builds on that.

---

## Part 1: What Is Apache DataFusion?

[Apache DataFusion](https://datafusion.apache.org/) is an **embeddable query engine** written in Rust that uses Apache Arrow as its in-memory columnar format. Unlike standalone databases (Spark, Trino, ClickHouse), it is a *library* you build products *on top of* — InfluxDB 3.0, dbt Fusion, Cloudflare R2, Arroyo, and many others embed it as their query core.

It offers SQL and DataFrame APIs that both lower to a single `LogicalPlan`, then optimize and execute via a **pull-based streaming** model.

### Three Design Principles

1. **Work out of the box** — world-class engine with minimal setup
2. **Customize everything via traits** — every layer is replaceable
3. **"Architecturally boring"** — industrial best practices over experimental techniques

### Architecture at a Glance

```
SQL Text ──► [Parser] ──► AST ──► [SqlToRel] ──┐
                                                ├──► LogicalPlan ──► [Analyzer]
DataFrame API ──► [LogicalPlanBuilder] ─────────┘         │
                                                           ▼
                              [Logical Optimizer] ──► [Physical Planner] ──► [Physical Optimizer]
                                    (rule pipeline)         │                      │
                                                           ▼                      ▼
                                                    ExecutionPlan     SendableRecordBatchStream
                                                                   (pull-based, streaming, columnar)
```

**Seven stages:** SQL/DataFrame → `LogicalPlan` (what) → analysis (type coercion) → logical optimization (rule-based rewrites) → `ExecutionPlan` (how, per-backend) → physical optimization (distribution, sorting, pushdown) → streaming execution.

### The Rule Pipeline (the secret sauce)

DataFusion's optimizer is not a monolith. It is a **sequence of independent rewrite passes**, each unit-testable in isolation:

| Rule | What it does |
|---|---|
| `SimplifyExpressions` | Constant folding, boolean simplification, algebraic identities |
| `PushDownFilter` | Moves WHERE predicates as close to data sources as possible |
| `PushDownLimit` | Pushes LIMIT into table scans |
| `OptimizeProjections` | Prunes unused columns, merges consecutive projections |
| `CommonSubexprEliminate` | Extracts repeated expressions so they compute once |
| `EliminateFilter` | Removes always-true/false filters |
| `PropagateEmptyRelation` | Propagates empty results upward (empty join input → empty output) |
| `EliminateCrossJoin` | Converts CROSS JOIN to INNER JOIN when join predicates exist |
| `EliminateOuterJoin` | Converts outer joins to inner joins when nulls can't be produced |
| `DecorrelatePredicateSubquery` | Converts `IN`/`EXISTS` subqueries to SEMI/ANTI joins |

Each rule is a trait implementation. You can add, remove, reorder, or replace any rule. `EXPLAIN` shows the full rewrite chain, rule by rule.

---

## Part 2: Why This Matters — metaengine Is a Query Engine

The direct relevance is the `metaengine/` module — labeled "THE STRATEGIC FUTURE" in AGENTS.md. metaengine is, structurally, a small query engine over CQRS projections. The parallels are already striking:

| DataFusion concept | metaengine equivalent today | Status |
|---|---|---|
| `LogicalPlan` / `ExecutionPlan` split | `Plan` + `LayoutPlan` / backend dispatch | **Gap A** |
| Predicate pushdown to data sources | `FilterOnField` / `SortOnField` → SQLite `json_extract` | Aligned |
| Streaming pull-based execution | `StreamScan(ctx) iter.Seq2` | Aligned |
| `Engine` / `TableProvider` traits | `Engine`, `MapBackend`/`ScanBackend`/... interfaces | Aligned |
| Cost-based planning | `cost.go`, `CostEstimate`, `PlanResult` | Exists (structural) |
| `EXPLAIN` | `explain.go`, `PlanResult.Report()` | Exists |
| Rule-composed optimizer | (monolithic `planner.go`) | **Gap B** |
| Statistics-driven planning | (not present — cost model is structural, not statistical) | **Gap C** |
| Plan serialization / portability | (not present) | **Gap D** |

### The Gaps

These are the concrete gaps between where metaengine is and where a DataFusion-class engine would be. They are the improvement targets.

#### Gap A: Logical/physical split is implicit, not explicit

DataFusion has one `LogicalPlan` (what to compute) that lowers to many possible `ExecutionPlan`s (how, per-backend). metaengine has `LayoutPlan` + `LayoutPlanner` doing access-path selection, but it is ad-hoc per-feature — only SQLite implements `LayoutPlanner`, and the logical→physical lowering is baked into `execute.go` rather than being a separate lowering step.

**The problem:** this makes multi-backend specialization incidental rather than principled. Pebble, a future Postgres engine, or a ClickHouse engine would each need to re-derive the lowering logic.

#### Gap B: The optimizer is monolithic

`planner.go` is a single function that ranks engines and assigns queries. There is no rule pipeline — each optimization (push filter, choose access path, detect write amplification) is inline code, not a composable, testable, replaceable rule.

**The problem:** as pushdown/projection/layout/temporal rules multiply, the monolith becomes untestable and un-extensible. Adding a new optimization means modifying the core planner, not adding a rule to a list.

#### Gap C: No statistical inputs

The cost model (`cost.go`) is *structural*: it maps ADT → Complexity → CostEstimate using calibrated `NsPerOp` constants. It does not ingest runtime statistics (write rate, read rate, stream length, snapshot age). It cannot answer "is this stream hot enough to materialize?"

**The problem:** structural costing picks the obviously-right engine (O(1) memory vs O(N) scan). It cannot make tradeoff decisions (materialize vs. replay, snapshot boundary, index vs. full scan) that depend on actual data shape and access patterns.

#### Gap D: No plan serialization

DataFusion supports [Substrait](https://substrait.io/) — a cross-system query plan interchange format. Plans can be serialized, shipped between systems, and replayed. metaengine has no equivalent; plan decisions live in memory only.

**The problem:** plan decisions cannot be persisted for debugging, compared across runs, or stabilized across version upgrades. A plan that was optimal yesterday may silently change after a code update.

---

## Part 3: Concrete Learnings Worth Stealing

### 1. Compose the optimizer from independent, testable rules

**This is the highest-leverage single change.** DataFusion's optimizer is a sequence of `OptimizerRule` trait implementations, each independently testable. metaengine's `planner.go` is a monolith.

A sketch of what this looks like in Go:

```go
// PlanRewriteRule rewrites a logical plan into a more efficient equivalent.
// Each rule is independently testable — pass a plan, assert the output.
type PlanRewriteRule interface {
    Name() string
    Rewrite(plan *LogicalPlan, ctx PlanContext) *LogicalPlan
}

// PlanContext carries statistics, engine profiles, and config that rules need.
type PlanContext struct {
    Engines []Engine
    Stats   Stats
    Config  PlanConfig
}

// The optimizer is just a list of rules applied in sequence.
type Optimizer struct {
    rules []PlanRewriteRule
}

func (o *Optimizer) Optimize(plan *LogicalPlan, ctx PlanContext) *LogicalPlan {
    for _, rule := range o.rules {
        plan = rule.Rewrite(plan, ctx)
    }
    return plan
}
```

Concrete rules metaengine would benefit from (ES-aware, see Part 5):

| Rule | What it does |
|---|---|
| `PushFilterIntoFold` | Moves filter predicates into the fold function so dead events are skipped early |
| `PushFilterIntoScan` | Pushes `FilterOnField` into the storage backend's scan (already exists inline) |
| `DecideMaterialization` | Chooses compute-on-demand vs. maintain-projection based on write/read ratio |
| `SelectSnapshotBoundary` | Chooses where to snapshot based on stream length + replay cost |
| `PruneTombstones` | Eliminates tombstoned entities from scan targets |
| `SelectTier` | Routes to KV-document vs. relational vs. graph based on query shape |
| `DetectWriteAmplification` | Flags when one event updates too many projections (already exists inline) |

Each rule testable alone. Each replaceable per-backend. `EXPLAIN` shows the rewrite chain. This is not just better engineering — it is the prerequisite for everything else in this document.

### 2. Make the logical/physical split explicit

DataFusion's power comes from separating **what** (logical plan) from **how** (physical plan). metaengine should treat `LayoutPlan` as the *physical plan* and elevate it to a first-class lowered representation that *every* backend can specialize:

```go
// LogicalPlan describes WHAT to compute (backend-agnostic).
type LogicalPlan struct {
    Collection   string
    ADT          ADT
    Filters      []FilterClause
    Sort         *SortClause
    AsOf         *TemporalAnchor  // ES-specific (see Part 5)
}

// PhysicalPlan describes HOW to compute it (backend-specific).
type PhysicalPlan struct {
    Engine      Engine
    Layout      LayoutPlan        // indexes, column extraction, DDL
    Strategy    ExecutionStrategy // scan, point-lookup, snapshot+delta, etc.
}
```

This makes Pebble-vs-SQLite-vs-future-Postgres engine selection **principled rather than incidental** — each backend lowers the same logical plan using its own physical capabilities.

### 3. Add statistical inputs to the cost model

DataFusion ingests row counts and cardinality for join ordering. metaengine's cost model is structural — it knows `ADTMap` is `O(logN)` on SQLite, but not whether a stream is hot enough to justify materialization.

A sketch:

```go
// Stats describes runtime characteristics that the planner needs for
// tradeoff decisions. All fields are optional (zero = unknown).
type Stats struct {
    WriteRatePerSec   float64 // events/sec written to this stream type
    ReadRatePerSec    float64 // queries/sec against this projection
    AvgStreamLength   int     // average events per stream
    SnapshotAge       int     // versions since last snapshot
    TombstoneRatio    float64 // fraction of tombstoned entities (0.0-1.0)
}
```

With statistics, the planner can answer questions it currently cannot:
- "Is this stream hot enough to materialize?" (write/read ratio)
- "Should we snapshot at version 100 or 500?" (replay cost vs. snapshot cost)
- "Is a full scan acceptable or do we need an index?" (volume)

### 4. Make plans serializable

DataFusion supports Substrait for plan interchange. metaengine should persist plan decisions so they can be:
- **Debugged** — "why did the planner choose engine X for this query?"
- **Compared** — "did the plan change after the last deploy?"
- **Stabilized** — "pin this plan until we explicitly re-plan"

```go
// SerializablePlan is a snapshot of a plan decision, persistable for
// debugging, comparison, and plan stability across version upgrades.
type SerializablePlan struct {
    Version   string           // metaengine version that produced this plan
    CreatedAt time.Time
    Logical   LogicalPlan
    Physical  PhysicalPlan
    Stats     Stats
    Cost      CostEstimate
}
```

### 5. The "architecturally boring" north star

DataFusion deliberately avoids novel research techniques in favor of proven industrial patterns. Worth adopting as metaengine's design ethos: be the boringly-correct planner, not the clever one. This aligns with the `cost.go` honesty note already in the codebase: *"a rough first-order model, not a calibrated query optimizer."*

### 6. Extension over fork

DataFusion's strongest cultural lesson: when something is not supported, the answer is "add a trait/extension point," never "fork it." metaengine's backend-interface split (`Engine`, `MapBackend`, `ScanBackend`, `SetBackend`, `CounterBackend`, `GraphBackend`, `MultimapBackend`, `LogBackend`) is already this — protect it. New capabilities should be new interfaces, not modifications to existing ones.

---

## Part 4: What Does NOT Transfer

Not everything DataFusion does is relevant. Being explicit about *why* prevents cargo-culting.

| DataFusion feature | Why it does not transfer |
|---|---|
| **Arrow columnar format** | Go's Arrow ecosystem is immature; the columnar/vectorized execution advantage requires deep SIMD integration that Go's GC model makes impractical. CBOR/JSON self-describing envelopes are the right call for CQRS payloads — events are self-describing, not columnar. |
| **SQL parser** | go-cqrs-lite is programmatic/typed-first (`QueryBuilder`, `FilterOnField`, `TypedReader[V]`). A SQL surface would be scope creep and would fight the typed Go API. DataFusion needs SQL because it's a *database*; metaengine is a *library*. |
| **Distributed execution (Ballista)** | Multi-node execution is out of scope. The single-process embeddable niche is what makes go-cqrs-lite valuable — it is the library you compose *into* a distributed system, not a distributed system itself. |
| **Cost-based join ordering** | Joins across event streams are rare in CQRS (each aggregate is independent). The interesting cost decisions are materialize-vs-replay and snapshot boundaries, not join trees. |
| **Partition-aware execution** | Parallelism in ES is per-stream (independent aggregates), not per-partition within a single query. The `projectionhost` worker model handles this already. |

---

## Part 5: How Event Sourcing Changes Everything

Event Sourcing inverts DataFusion's foundational assumption: **the table is not the truth; the event log is.** The table (projection/read model) is a *derived, disposable, temporally-versioned* artifact. This changes the planning problem in escalating ways.

### The Floor: What Stays the Same

The structural lessons still hold:
- Rule-composed optimizer — still the right architecture; the rules just differ
- Trait-based extensibility — `Engine`/backend split is still correct
- Pull-based streaming — `StreamScan` is still right; the stream is now the event log
- "Architecturally boring" — still the right north star

### The Delta: What Gets Harder

#### A. Materialization is now a planning decision, not a deployment fact

DataFusion assumes the table exists and asks "how do I scan it efficiently." ES asks the prior question: **"should this projection exist at all?"**

A query over events can be answered two ways:

| Strategy | Cost model | When it wins |
|---|---|---|
| **Compute-on-demand** — replay events, fold, filter | O(events in stream) per query | Rare reads, small streams |
| **Materialize** — maintain a projection, query it | O(1) per query + O(events) amortized write | Frequent reads, hot streams |

**The cost formula** that makes this decision:

```
replay_total(q)      = read_rate × avg_stream_length × fold_cost
materialize_total(q) = write_rate × fold_cost + storage_cost + read_rate × query_cost

Materialize when: materialize_total < replay_total

Simplified (ignoring storage_cost):
  write_rate × fold_cost + read_rate × query_cost < read_rate × avg_stream_length × fold_cost
  ⟺  read_rate × (avg_stream_length × fold_cost - query_cost) > write_rate × fold_cost
  ⟺  read_rate / write_rate > fold_cost / (avg_stream_length × fold_cost - query_cost)
  ≈  read_rate / write_rate > 1 / avg_stream_length   (when fold_cost dominates)
```

The **write/read ratio** is the king input DataFusion lacks entirely. It is the difference between a planner and a planner that knows *whether to build the table at all*. This single dimension makes ES planning categorically harder than relational planning.

#### B. The "schema" is the fold function, not a fixed shape

DataFusion's `TableProvider` exposes a static schema. In ES, the read-model shape is *derived* from how you fold events — and it shifts when the fold function changes or when upcasters transform old events (`schema.VersionedStore`, `schema.Upcaster`).

The plan must reason over a **versioned, evolving** schema, not a static one. DataFusion's `LogicalPlan → TableScan` assumes column lists are stable; ES's equivalent must account for:
- "Event v1 means X, v2 means Y" (upcaster chains)
- "This projection was built with fold function F1; we deployed F2" (rebuild decision)
- "This field didn't exist before version 50" (schema evolution)

#### C. Statistics become more valuable AND more complex

ES planning needs statistics DataFusion never considers:

| Statistic | Planning purpose | DataFusion equivalent |
|---|---|---|
| **Write rate** per stream type | Replay cost grows with write rate; materialization cost grows with write rate | None — DataFusion tables don't "grow" |
| **Read rate** per projection | Materialization payoff | None — DataFusion doesn't decide whether to materialize |
| **Stream length distribution** | Long streams = snapshot territory | Row counts (but for static tables) |
| **Snapshot age** | Incremental load cost = `LoadFromVersion(cachedVer)` delta size | None — no concept of incremental state loading |
| **Tombstone ratio** | How much state is dead (prunable from scans) | None — no concept of soft-delete |

### The Ceiling: What Is Entirely New (ES-Only)

These are dimensions DataFusion literally does not model. They are why metaengine could be **more interesting than a mini-DataFusion** — it is solving a strictly harder problem.

#### 1. Temporal queries — "as-of" time travel

Reconstruct state at any point by replaying events to a version/timestamp. `EventSource.LoadToVersion` / `LoadToTimestamp` already exist. A query engine over ES should make `as_of = '2026-07-15'` a **first-class logical operation** with multiple physical realizations:

```go
// TemporalAnchor is the "as-of" specification on the logical plan.
type TemporalAnchor struct {
    Version   *uint64    // as-of event version
    Timestamp *time.Time // as-of wall clock
}
```

| Physical strategy | Cost | Availability |
|---|---|---|
| Snapshot at nearest version + delta replay | O(delta events) | If snapshot exists near anchor |
| Full replay from zero | O(all events) | Always available |
| Cached projection as-of | O(1) | If materialized and up-to-date |

A concrete example — logical plan and physical realizations:

```
LOGICAL PLAN:
  Collection: "users"
  Filter: status = "active"
  AsOf: version 42

PHYSICAL PLAN (option A — snapshot+delta, if snapshot at v35 exists):
  1. LoadSnapshot("users", streamID, v35)
  2. LoadFromVersion(streamID, 35, 42) → fold 7 events
  3. Filter result on status="active"

PHYSICAL PLAN (option B — full replay, no snapshot):
  1. LoadToVersion(streamID, 42) → fold all 42 events
  2. Filter result on status="active"

PHYSICAL PLAN (option C — cached projection):
  1. QueryMaterialized("users_active_asof_42")
  2. (No fold, no replay — O(1))
```

The planner's job: pick the cheapest physical plan given available snapshots, stream length, and whether a materialized view exists. DataFusion has no concept of this — it is a superpower ES gives you for free.

#### 2. Three projection tiers, one planner

"Show me every event caused by command X" — traversing causation chains (`metadata.causation`). This is why `graph/` exists. The planner must recognize the query *shape* and route to the right tier:

| Query shape | Projection tier | Backend | Example |
|---|---|---|---|
| Single-document CRUD, point lookups | **KV-document** (`stack.Materialize`) | Pebble, SQLite KV | `GetUser(id)` |
| Multi-table joins, aggregations, rollups | **Relational** (`storage.RelationalProjection`) | SQLite, Postgres | `CountMessagesByChannelAndDay()` |
| N-hop traversals, adjacency, path-finding | **Graph** (`graph.GraphProjection`) | Memory, Neo4j | `FindReplyChain(msgID, depth=5)` |

DataFusion has **one tier** (relational). metaengine has three. The planner must classify the query and route accordingly:

```go
func classifyTier(plan *LogicalPlan) Tier {
    // Graph: recursive depth, variable-length traversal, adjacency queries
    if hasTraversal(plan) || hasNHopFilter(plan) {
        return TierGraph
    }
    // Relational: joins across tables, GROUP BY, multi-row aggregations
    if hasJoin(plan) || hasAggregation(plan) {
        return TierRelational
    }
    // Default: single-document point lookup
    return TierKV
}
```

This classification is a **rule** — composable, testable, replaceable. It is exactly the kind of optimization that the rule pipeline (Part 3, Learning 1) enables.

#### 3. Replay semantics & processing mode

The same fold function runs in two modes — historical replay and live event processing (`ModeReplay`/`ModeLive`):

| Mode | Parallelism | Ordering | Dedup |
|---|---|---|---|
| **Replay** (`ModeReplay`) | Can parallelize across streams | Not required | Idempotent-safe |
| **Live** (`ModeLive`) | Single goroutine per stream | Required (FIFO) | Required (`dedup.Ring`) |

DataFusion executes once over static data. ES executes **continuously** over a growing log. The planner must know the mode because it changes valid physical plans: replay can use aggressive parallelism; live cannot.

#### 4. Projections are disposable

A projection can be dropped and rebuilt from zero (`projectionhost.Resettable`, `host.Reset(ctx, "users")`). This means planning can be **aggressive** about representation choices — nothing is irreversible because the source of truth is the log.

DataFusion cannot assume this; dropping a Parquet file loses data. This unlocks bolder layout decisions: the planner can experiment with an index layout, measure, and rebuild with a different layout if the access pattern changed. The cost of a bad layout choice is rebuild time, not data loss.

#### 5. Idempotency as a planning constraint

At-least-once delivery means the same event may arrive twice. A materialization strategy must be idempotent (upsert, not append). This constrains which physical plans are *valid*, not just which are *fast*.

This is a **hard correctness constraint**, not an optimization. The planner must reject any physical plan that would produce incorrect results under duplicate delivery — or must inject a dedup mechanism (`dedup.Ring`, idempotency store) into the plan. DataFusion never reasons about correctness under duplicate input.

---

## Part 6: Revised Recommendations (ES-Aware)

| Recommendation | DataFusion view | ES-revised view |
|---|---|---|
| Rule-composed optimizer | Rules: push filter, prune columns, eliminate joins | Rules: **push filter into fold**, decide materialize-vs-replay, snapshot boundary, tombstone pruning, tier selection (KV/relational/graph), detect write amplification |
| Statistics | Row counts, cardinality | **+ write rate, read rate, stream length, snapshot age, tombstone ratio** — write/read ratio is the decisive new input |
| Logical/physical split | `LayoutPlan` = physical plan | `LayoutPlan` is physical, **+ temporal dimension** — logical plan carries `AsOf`; physical plan chooses snapshot+delta vs. full replay vs. cached projection |
| Streaming | `StreamScan` = pull-based scan | Still right, but the stream is **append-only and potentially infinite** (live tailing), not a bounded dataset |
| Plan serialization | Substrait for cross-system exchange | Persist plan decisions for **debugging, comparison, and plan stability** across version upgrades |
| Correctness constraints | (none — static data, execute once) | **Idempotency** (upsert, not append), **ordering** (replay vs. live mode), **disposability** (rebuild-safe aggressive layouts) |

---

## Part 7: Prioritized Action Sequence

The learnings are not independent — there is a dependency chain. Here is the Pareto-optimal execution order.

### Phase 0: Extract the rule pipeline (prerequisite for everything)

**Effort:** Small (refactor, no behavior change)
**Impact:** Unlocks all subsequent phases

Split `planner.go`'s inline optimizations into composable `PlanRewriteRule` implementations. No new behavior — just move existing logic (push filter, detect write amplification, assign engine) into named, testable rules. The existing tests must still pass unchanged.

**Why first:** every subsequent phase adds a new rule. Without the pipeline, each addition modifies the monolith, compounding the problem.

### Phase 1: Enrich the cost model with statistics

**Effort:** Medium
**Impact:** High — enables materialize-vs-replay decisions

Add `Stats` to the planner context. Start with optional fields (zero = unknown, fall back to structural costing). Wire `projectionhost` to collect write/read counts and feed them back into re-planning.

### Phase 2: Explicit logical/physical split

**Effort:** Medium
**Impact:** High — enables principled multi-backend specialization

Elevate `LayoutPlan` to a first-class `PhysicalPlan` that every backend can produce, not just SQLite. Refactor `execute.go`'s lowering into a separate `Lower(logical) → physical` step.

### Phase 3: Temporal queries

**Effort:** Large
**Impact:** High for ES use cases — time travel is a superpower

Add `TemporalAnchor` to the logical plan. Implement the three physical strategies (snapshot+delta, full replay, cached projection). Add a rule that selects the cheapest available strategy.

### Phase 4: Tier classification

**Effort:** Medium
**Impact:** Medium — enables automatic KV/relational/graph routing

Add the `SelectTier` rule that classifies query shape and routes to the appropriate projection tier. This requires the tier interfaces (`kv.ViewStore`, `storage.RelationalProjection`, `graph.GraphProjection`) to share a common planning contract.

### Phase 5: Plan serialization

**Effort:** Small
**Impact:** Medium — debugging, plan stability, cross-version comparison

Add `SerializablePlan` and a `Plan()` variant that returns it. Persist plan decisions alongside the projection checkpoint.

---

## Part 8: Risks and Tradeoffs

Intellectual honesty — these recommendations have costs.

| Risk | Mitigation |
|---|---|
| **Premature abstraction** — splitting into rules before there are enough rules to justify the pipeline adds indirection without benefit | Phase 0 is safe because there are *already* 3+ inline optimizations (push filter, write amplification, engine assignment). The abstraction is justified today. |
| **Statistics collection overhead** — gathering write/read rates adds bookkeeping on every operation | Make stats optional and sampled (e.g., collect every Nth operation). Zero-stats = structural fallback. Never block the hot path for stats. |
| **Temporal queries may be rarely used** — building the full machinery (3 physical strategies, `TemporalAnchor`, snapshot+delta planner) for a feature nobody asks for is waste | Defer Phase 3 until there is a real consumer. The logical plan should *carry* `AsOf` (zero cost) but the physical realizations can wait for demand. |
| **Tier misclassification** — routing a graph query to the KV tier produces wrong results or terrible performance | The classifier must be conservative: when in doubt, use the most general tier (relational). Add a diagnostic warning when classification is uncertain. |
| **Over-engineering the cost model** — a sophisticated statistical cost model that nobody calibrates is worse than a simple structural one | The `cost.go` honesty note is the model: always document assumptions, always provide structural fallback, never trust absolute numbers without calibration. |
| **Plan instability** — statistical plans change as stats fluctuate, causing unexpected performance shifts | Phase 5 (plan serialization) provides the escape hatch: pin a plan when stability matters more than optimality. |

---

## The Meta-Point

DataFusion plans over **state**. metaengine must plan over **derived state with temporal and causal dimensions** — state that:

- Can be reconstructed at any point in its history
- Has causal provenance (command → event chains)
- Evolves its own schema (upcasters, versioned events)
- Whose materialization is itself a cost/benefit decision (not a deployment fact)
- Is disposable and rebuildable from the log
- Must handle idempotent reprocessing

The DataFusion learnings transfer, but they are the **floor**. The ceiling is something DataFusion never had to build: a planner that knows *whether to build the table*, *at what point in time*, and *from what cause*.

That is the strategically interesting frontier — and the argument for why metaengine deserves to be "the strategic future," possibly its own project. The [project-definition](meta-engine-project-definition.md) already frames it as research-grade. DataFusion shows us the *engineering* patterns to steal. Event Sourcing shows us the *problem dimensions* that make it worth solving.
