# Metaengine Domain Language

> **[← Domain Language](DOMAIN_LANGUAGE.md)** — the shared ubiquitous language for the entire library.

A specialized vocabulary for the **metaengine** — the cost-based storage planner at the strategic core of `go-cqrs-lite`.

> **Mental model:** Declare _what_ you want to query (folds). The metaengine infers _how_ to store it (ADT), decides _where_ to store it (engine), generates the DDL, and routes reads. You never pick an engine or write table schemas — the planner does, backed by real cost estimates.

A **cost-based storage planner** (CBO) for event-sourced projections. Given a set of query declarations and available engines, the planner assigns each query to the cheapest engine and emits cost diagnostics. The core axiom: _every query pattern CAN be served by every storage engine — the question is never "can we?" but "at what cost?"_ The metaengine is the strategic future of this project ([design docs](planning/meta-engine-design.md), ADRs [0061](adr/0061-metaengine-sqlite-engine.md)–[0117](adr/0117-command-lifecycle-as-events.md)).

> **Relationship to Read Models:** The three projection tiers in the [Domain Language](DOMAIN_LANGUAGE.md#read-models) (Document/KV, Relational, Graph) are _manual_ — the consumer hand-writes each projection. The metaengine _automates_ this: it infers the ADT from fold return types, picks the optimal engine, generates DDL, and routes queries — 80% auto-generated, 100% auto-routed ([ADR-0116](adr/0116-layered-auto-projection.md)).

> **When to choose Metaengine vs Manual Read Models:**
>
> | Use Manual Read Models when…                    | Use Metaengine when…                                           |
> | ----------------------------------------------- | -------------------------------------------------------------- |
> | You need full control over SQL/table shape      | You want the planner to pick the engine and generate DDL       |
> | Projections are simple (one type, one table)    | You have many query patterns across multiple collections       |
> | You need graph traversal (Cypher/Gremlin)       | You want cost-based engine selection (CBO)                     |
> | You're comfortable hand-writing projections     | You want auto-projection from Record types (planned, ADR-0116) |
> | You need multi-table atomic writes (relational) | You want filter/sort pushdown without writing SQL              |

> **Implementation status:** ADRs 0061–0098 are implemented (cost-based planner, 11 ADTs, 9 engines, layouts, persistence, replication, materialize-vs-replay, plan diff/manifest). ADRs 0111–0117 describe the v2 vision: ES-native planning from Record types (ADR-0112, partially implemented via `OnRecord`), tombstone-as-domain-event (ADR-0114, planned), and command lifecycle as event streams (ADR-0117, planned). Terms from unimplemented ADRs are marked **(planned)** below.

---

## Table of Contents

- [Core Concepts](#core-concepts)
- [ADTs (Abstract Data Types)](#adts-abstract-data-types)
- [Fold DSL (Event to Projection Mapping)](#fold-dsl-event-to-projection-mapping)
- [Cost Model](#cost-model)
- [Storage Layouts](#storage-layouts)
- [Read Patterns](#read-patterns)
- [Filter, Sort & Pagination](#filter-sort--pagination)
- [PlanRule Pipeline](#planrule-pipeline)
- [Persistence Model](#persistence-model)
- [Replication Model](#replication-model)
- [Materialize vs Replay](#materialize-vs-replay)
- [Temporal Reads (As-Of)](#temporal-reads-as-of)
- [Plan Operations](#plan-operations)
- [Engine Capabilities (Optional Interfaces)](#engine-capabilities-optional-interfaces)
- [Hot Operations](#hot-operations)
- [Key Codec (LSM Backends)](#key-codec-lsm-backends)
- [Engines](#engines)
- [Projection Adapter](#projection-adapter)
- [Testing & Benchmarking](#testing--benchmarking)
- [Shared Terms](#shared-terms-defined-in-domain-language)
- [Interface Hierarchy](#interface-hierarchy--engine)
- [Terms We Avoid](#terms-we-avoid-metaengine-specific)
- [Verification](#verification)

---

### Core Concepts

| Term                  | Definition                                                                                                        | Context                                                                                                         |
| --------------------- | ----------------------------------------------------------------------------------------------------------------- | --------------------------------------------------------------------------------------------------------------- |
| **Engine**            | A storage backend with a cost profile; the unit the optimizer ranks and assigns queries to                        | `metaengine.Engine` interface — `Profile() EngineProfile` + `Closer`; impls: Memory, SQLite, Pebble, DuckDB, PG |
| **EngineProfile**     | Declares what an engine can do (ADTs + Complexity), how fast (calibrated ns/op), layout, persistence, replication | `metaengine.EngineProfile` struct — the engine's "datasheet"                                                    |
| **Store**             | The running, planned runtime: holds engines, registered queries, the plan, and event log                          | `metaengine.Store` — created by `Plan()`, queried via `ExecuteTyped`                                            |
| **Plan / PlanResult** | The optimizer's output: one `QueryAssignment` per query + diagnostics + layout plans                              | `metaengine.PlanResult` — produced by `metaengine.Plan(engines, queries...)`                                    |
| **QueryAssignment**   | The per-query plan decision: engine, ADT, read pattern, complexity, cost, diagnostics                             | `metaengine.QueryAssignment` struct — one per declared query                                                    |
| **Collection**        | A planned query's materialized projection — the named, queryable view an engine serves                            | `metaengine.CollectionInfo` — introspect via `Store.Collections()`                                              |
| **QueryDecl**         | A fully-analyzed query declaration: name, folds, inferred ADT, read pattern, config                               | `metaengine.QueryDecl[Q,R]` — created via `metaengine.Query[Q,R](name, folds...)`                               |
| **QueryConfig**       | Declarative per-query tuning: Volume, LatencyBudget, TTL, filter/sort fields, columnarLayout                      | `metaengine.QueryConfig` — set via `QueryOption` funcs like `metaengine.Volume(n)`                              |

### ADTs (Abstract Data Types)

An **ADT** is the logical data structure the planner infers from a query's fold return types. It determines which engines can serve the query and at what complexity. Inference happens automatically via fold classification.

| ADT            | Operations                         | Key Trait                     | Constant                  |
| -------------- | ---------------------------------- | ----------------------------- | ------------------------- |
| **Map**        | Get/Set/Delete by key              | Keys unique                   | `metaengine.ADTMap`       |
| **Sorted Map** | Map ops + Range/Filter/OrderBy     | Ordered                       | `metaengine.ADTSortedMap` |
| **Multimap**   | Add(key,val)/GetAll(key)           | One key to many values        | `metaengine.ADTMultimap`  |
| **Counter**    | Increment/Get                      | Numeric aggregation           | `metaengine.ADTCounter`   |
| **Set**        | Add/Contains/Members               | Values unique                 | `metaengine.ADTSet`       |
| **Log**        | Append/ReadFrom (ordered)          | Append-only sequence          | `metaengine.ADTLog`       |
| **Stream Log** | Stream-keyed append (ES primitive) | Per-stream ordering           | `metaengine.ADTStreamLog` |
| **Graph**      | AddEdge/Neighbors/Traverse         | Edge traversal                | `metaengine.ADTGraph`     |
| **Vector**     | Insert embedding / k-NN search     | Similarity (cosine/euclidean) | `metaengine.ADTVector`    |
| **Search**     | Insert document / full-text query  | Inverted index (TF-IDF/BM25)  | `metaengine.ADTSearch`    |
| **Spatial**    | Insert point / range proximity     | Geo distance (haversine)      | `metaengine.ADTSpatial`   |

Each ADT maps to an optional **Backend interface** an engine implements (ISP): `MapBackend`, `SetBackend`, `CounterBackend`, `MultimapBackend`, `LogBackend`, `StreamLogBackend`, `VectorBackend`, `SearchBackend`, `SpatialBackend`. An engine that lacks the native backend for an ADT may still serve it via **degraded** brute-force fallback (with a `DEGRADED` diagnostic).

> **Typed fold inputs:** Vector and Search ADTs have dedicated input types: `metaengine.Embedding` (ID + `[]float32` values) for k-NN similarity folds, and `metaengine.IndexedText` (ID + content string) for full-text search folds.

### Fold DSL (Event to Projection Mapping)

A **Fold** is a single typed event-to-projection update rule: "when event E happens, update the projection like this." Folds are the write path. The planner inspects fold return types to infer the ADT.

| Term           | Definition                                                                                   | Context                                                                                |
| -------------- | -------------------------------------------------------------------------------------------- | -------------------------------------------------------------------------------------- |
| **Fold**       | A sealed interface representing one event-to-projection update rule                          | `metaengine.Fold` — created only via `On`/`OnTyped`/`OnRecord`                         |
| **On**         | Constructor binding an event type to a handler closure; classifies the fold kind             | `metaengine.On[E](sample, handler)` — infers ADT from handler return type              |
| **OnTyped**    | Like `On` but binds to an explicit event-type string (for external schemas)                  | `metaengine.OnTyped[E](eventType, sample, handler)`                                    |
| **OnRecord**   | Creates a Record-aware fold: handler receives the full `record.Record` context               | `metaengine.OnRecord[E](sample, handler)` — access StreamID, Version, MetaData         |
| **FoldKind**   | Classifies what a fold does: insert, update, remove, count, edge, set, skip, etc.            | `metaengine.FoldKind` — `FoldInsert`, `FoldUpdate`, `FoldCount`, `FoldEdge`, etc.      |
| **Delta**      | Counter update: `map[string]int64` of key to delta                                           | `metaengine.Delta` — return type for Counter ADT folds                                 |
| **Edge**       | Graph edge (From, To)                                                                        | `metaengine.Edge` struct — return type for Graph ADT folds                             |
| **MultiEntry** | Sentinel return type classifying a fold as a multimap insert (Key + Value)                   | `metaengine.MultiEntry` struct                                                         |
| **Skip**       | Sentinel signaling an event does not apply to this projection (no-op)                        | `metaengine.Skip` struct — return `metaengine.Skip{}` to ignore                        |
| **Remove**     | Constructor for a delete-by-key fold                                                         | `metaengine.Remove[V]()`                                                               |
| **Poison**     | A collection refuses reads after a fold panic; error stored until store recreate             | `Store.IsPoisoned(collection)` — quarantine mechanism                                  |
| **AutoInsert** | Reflection-based fold: inserts a new record from event fields, auto-stamping Record metadata | `metaengine.AutoInsert[E, R](sample, eventType)` — no hand-written handler needed      |
| **AutoUpdate** | Reflection-based fold: updates an existing record's fields from event                        | `metaengine.AutoUpdate[E, R](sample, eventType)` — field-by-field merge via reflection |
| **AutoDelete** | Reflection-based fold: marks a record for deletion by key                                    | `metaengine.AutoDelete[E](sample, eventType)`                                          |
| **AutoCRUD**   | Combines AutoInsert + AutoUpdate + AutoDelete into one declaration                           | `metaengine.AutoCRUD[C, U, D, R](create, update, delete, result)` — full lifecycle     |

### Cost Model

The cost model estimates how expensive serving a query on a given engine is, using Big-O complexity classes and calibrated nanoseconds-per-operation.

> **Core formula:** `estimated_latency = (ops x nsPerOp / 1e6) + NetworkRTT` — ops derived from complexity class x volume, nsPerOp from calibration, NetworkRTT is additive fixed latency (0 for in-process engines). See also [Replication Model](#replication-model) for how NetworkRTT enters the formula.

| Term                   | Definition                                                                           | Context                                                                                                              |
| ---------------------- | ------------------------------------------------------------------------------------ | -------------------------------------------------------------------------------------------------------------------- |
| **Complexity**         | A Big-O complexity class for cost estimation                                         | `metaengine.Complexity` — `ComplexityO1`, `ComplexityOLogN`, `ComplexityON`, `ComplexityONLogN`, `ComplexityODegree` |
| **CostEstimate**       | The estimated cost of serving one query: complexity, volume, ops, latency (ms)       | `metaengine.CostEstimate` — `WithinBudget()` checks against LatencyBudget                                            |
| **NsPerOp**            | Calibrated nanoseconds-per-operation from benchmark calibration                      | `EngineProfile.NsPerOp`, `NsPerRead`, `NsPerWrite` — real measurements, not theoretical                              |
| **ReadCosts**          | Per-read-pattern calibrated costs (point lookup vs scan vs aggregate)                | `metaengine.ReadCosts` — engines span 4000x across operations                                                        |
| **ScaleThreshold**     | The optimal cardinality range for a data structure; planner warns when exceeded      | `metaengine.ScaleThreshold` — e.g. hash map warns past 10M entries                                                   |
| **Volume**             | The expected number of items in a projection, used as N in cost formulas             | `metaengine.Volume(n)` QueryOption                                                                                   |
| **LatencyBudget**      | Target latency ceiling for engine selection; planner flags when unmet                | `metaengine.WithLatencyBudget(ms)` QueryOption                                                                       |
| **WriteAmplification** | The cost of read optimization: each projection an event updates increases write cost | `metaengine.DefaultWriteAmplificationBudget` (3) — planner warns when exceeded                                       |

### Storage Layouts

A **StorageLayout** ([ADR-0073](adr/0073-metaengine-layout-planning.md), [ADR-0092](adr/0092-duckdb-columnar-native-storage.md)) describes the physical storage structure — it lets the planner reason about _why_ one engine beats another for a given access pattern.

| Layout       | Optimal for                          | Used by                       | Constant                    |
| ------------ | ------------------------------------ | ----------------------------- | --------------------------- |
| **Row**      | Point lookups, single-record updates | SQLite (B-Tree), Memory       | `metaengine.LayoutRow`      |
| **Columnar** | Aggregations, field-subset scans     | DuckDB                        | `metaengine.LayoutColumnar` |
| **LSM**      | Write-heavy with point reads         | Pebble, Badger                | `metaengine.LayoutLSM`      |
| **KV**       | Simple point lookups                 | Memory (hash map), generic KV | `metaengine.LayoutKV`       |

**Layout Planning (Level 2 optimization — within-engine):**

| Term                   | Definition                                                                   | Context                                                                             |
| ---------------------- | ---------------------------------------------------------------------------- | ----------------------------------------------------------------------------------- |
| **LayoutPlan**         | A planned table schema: extracts JSON fields into indexed SQL columns        | `metaengine.LayoutPlan` — replaces `json_extract()` scans with indexed column reads |
| **PlannedColumn**      | An extracted column (JSON field name to SQL type)                            | `metaengine.PlannedColumn` struct                                                   |
| **LayoutPlanner**      | Engine capability: can create optimized table layouts for filter/sort fields | `metaengine.LayoutPlanner` interface — implemented by SQLite, DuckDB, Postgres      |
| **LayoutPlanApplier**  | Extension receiving fully-built LayoutPlan with reflection-derived types     | `metaengine.LayoutPlanApplier` — DuckDB columnar-native extraction                  |
| **WithColumnarLayout** | QueryOption requesting full columnar extraction of ALL exported fields       | `metaengine.WithColumnarLayout()` — vectorized GROUP BY/SUM/AVG on DuckDB           |

### Read Patterns

A **ReadPattern** describes how a query reads its projection — distinct from the ADT's data-structure complexity. The cost model adjusts complexity per read pattern (a hash map O(1) for point lookup still scans O(N) for filtered scans).

| Read Pattern         | Meaning                               | Constant                        |
| -------------------- | ------------------------------------- | ------------------------------- |
| **Point Lookup**     | Single key to value                   | `metaengine.ReadPointLookup`    |
| **Membership**       | Key exists in set?                    | `metaengine.ReadMembership`     |
| **Multi-Lookup**     | Batch key to values                   | `metaengine.ReadMultiLookup`    |
| **Filtered Scan**    | WHERE predicate over collection       | `metaengine.ReadFilteredScan`   |
| **Aggregate**        | COUNT/SUM/MIN/MAX/AVG                 | `metaengine.ReadAggregate`      |
| **Traversal**        | Graph neighbor traversal              | `metaengine.ReadTraversal`      |
| **Scan**             | Full collection scan                  | `metaengine.ReadScan`           |
| **Log Tail**         | Read latest N entries from append log | `metaengine.ReadLogTail`        |
| **Vector Search**    | k-NN similarity search                | `metaengine.ReadVectorSearch`   |
| **Full-Text Search** | TF-IDF/BM25 ranked query              | `metaengine.ReadFullTextSearch` |
| **Spatial Range**    | Geo proximity query (haversine)       | `metaengine.ReadSpatialRange`   |

### Filter, Sort & Pagination

| Term              | Definition                                                                            | Context                                                                                |
| ----------------- | ------------------------------------------------------------------------------------- | -------------------------------------------------------------------------------------- |
| **FilterSpec**    | Declarative filter pushdown-able to SQL (Column + Op + Value to `json_extract` WHERE) | `metaengine.FilterSpec` struct                                                         |
| **SortSpec**      | Declarative sort pushdown-able to SQL (Column + Desc to ORDER BY)                     | `metaengine.SortSpec` struct                                                           |
| **FilterOnField** | Declare a filter by declarative field name (SQL pushdown)                             | `metaengine.FilterOnField[R](field, op)` — op: `metaengine.FilterEq`, `FilterLt`, etc. |
| **SortOnField**   | Declare sort by declarative field name (SQL pushdown)                                 | `metaengine.SortOnField[R](field, desc)`                                               |
| **FilterOn**      | Declare a filter via typed closure (Go-side filtering)                                | `metaengine.FilterOn[R,T](accessor)` — for non-SQL engines                             |
| **Cursor**        | Position marker for keyset (offset-free) pagination; encodes to URL-safe base64       | `metaengine.Cursor` struct — `ParseCursor(s)`                                          |
| **TypedReader**   | Typed read access to a collection without constructing a query input struct           | `metaengine.NewReader[V](store, coll)` — Get/Scan/Count/Sum                            |
| **QueryBuilder**  | Fluent, chainable API for building scans (Where/OrderBy/Limit/Cursor/Execute)         | `metaengine.NewQueryBuilder[V](reader)`                                                |

### PlanRule Pipeline

Rules run **after** engine assignment — they enrich the `PlanResult` (diagnostics, layout plans) but never override engine selection. This makes plans debuggable: "why was this query assigned here?"

| Term               | Definition                                                               | Context                                                               |
| ------------------ | ------------------------------------------------------------------------ | --------------------------------------------------------------------- |
| **PlanRule**       | A single composable post-assignment planning decision                    | `metaengine.PlanRule` interface — appends diagnostics or layout plans |
| **RulePipeline**   | Applies a sequence of PlanRules in order; aborts on first error          | `metaengine.RulePipeline` — `NewRulePipeline(rules...)`               |
| **RuleTraceEntry** | Records one rule's decision and reason, making EXPLAIN output debuggable | `metaengine.RuleTraceEntry` struct                                    |
| **Diagnostic**     | A plan-time message at one of four severity levels                       | `metaengine.Diagnostic` struct — Level + Query + Message              |

**Diagnostic levels** (escalating severity):

| Level        | Meaning                                                                        | Constant                       |
| ------------ | ------------------------------------------------------------------------------ | ------------------------------ |
| **SCREAM**   | Configuration that will cause permanent data loss — the store refuses to start | `metaengine.DiagLevelScream`   |
| **DEGRADED** | Engine serves the ADT via brute-force fallback (works but slow)                | `metaengine.DiagLevelDegraded` |
| **WARN**     | Suboptimal but not data-threatening configuration                              | `metaengine.DiagLevelWarn`     |
| **INFO**     | Advisory note (e.g., materialize-vs-replay recommendation)                     | `metaengine.DiagLevelInfo`     |

### Persistence Model

**Persistence** (DDIA Ch1: survivability — [ADR-0098](adr/0098-metaengine-persistence-enum.md)) answers one binary question: _if the process exits, is the data gone?_

| Term           | Definition                                                                  | Constant                           |
| -------------- | --------------------------------------------------------------------------- | ---------------------------------- |
| **Volatile**   | Data lives in process RAM and is lost on exit (the safe zero-value default) | `metaengine.PersistenceVolatile`   |
| **Persistent** | Data survives process exit via disk file or remote server                   | `metaengine.PersistencePersistent` |

The zero value is `PersistenceVolatile` — forgetting to set it causes a WARN (no silent data loss). SQLite/Pebble/DuckDB set it dynamically: volatile for in-memory constructors, persistent for file/DB constructors.

### Replication Model

**Replication** (DDIA Ch5 — [ADR-0093](adr/0093-metaengine-replication-model.md)) declares how an engine's data propagates across process boundaries. All CQRS read models are eventually consistent; the only strongly-consistent operation is the event store's optimistic-concurrency append.

> **Cost impact:** `NetworkRTT` from the table below feeds directly into the [Cost Model](#cost-model) formula as additive fixed latency. `ReplicationLag` is diagnostic-only — it describes freshness, not query latency.

| Term                | Definition                                                                                  | Constant                             |
| ------------------- | ------------------------------------------------------------------------------------------- | ------------------------------------ |
| **None**            | Data stays in this process (zero-value default for all current engines)                     | `metaengine.ReplicationNone`         |
| **Single-Leader**   | Writes to one leader, propagate to followers async (Postgres streaming)                     | `metaengine.ReplicationSingleLeader` |
| **Multi-Leader**    | Multiple nodes accept writes, reconcile via consensus (CockroachDB)                         | `metaengine.ReplicationMultiLeader`  |
| **Leaderless**      | Any node accepts writes, converge via CRDT merge (Iroh, Dynamo)                             | `metaengine.ReplicationLeaderless`   |
| **Replication Lag** | Expected delay between write on one node and visibility on another (freshness, NOT latency) | `EngineProfile.ReplicationLag`       |
| **Network RTT**     | Round-trip time to reach the engine's data (additive fixed latency, 0 for in-process)       | `EngineProfile.NetworkRTT`           |

### Materialize vs Replay

The planner can recommend whether a projection should be **materialized** (persisted, maintained incrementally) or **replayed** on-demand (fold the stream per query). This is the ES-specific killer feature.

| Term                  | Definition                                                                 | Context                                          |
| --------------------- | -------------------------------------------------------------------------- | ------------------------------------------------ |
| **WorkloadStats**     | Observed workload: write rate, read rate, avg stream length                | `metaengine.WorkloadStats` struct                |
| **ReplayCost**        | Cost of replaying a stream per query: `read_rate x stream_len x fold_cost` | `metaengine.ReplayCost(stats)`                   |
| **MaterializeCost**   | Cost of maintaining a materialized projection                              | `metaengine.MaterializeCost(stats)`              |
| **ShouldMaterialize** | Returns true when materialization cost < replay cost                       | `metaengine.ShouldMaterialize(stats)` — advisory |

### Temporal Reads (As-Of)

| Term                 | Definition                                                            | Context                                                 |
| -------------------- | --------------------------------------------------------------------- | ------------------------------------------------------- |
| **VersionedStorage** | Engine capability for temporal point lookups: "value of K at time T?" | `metaengine.VersionedStorage` interface — Memory engine |
| **AsOfSignal**       | Marker type in a query input that triggers temporal routing           | `metaengine.AsOfSignal` struct                          |

### Plan Operations

| Term                 | Definition                                                                         | Context                                                                               |
| -------------------- | ---------------------------------------------------------------------------------- | ------------------------------------------------------------------------------------- |
| **SerializablePlan** | A plan serialized to JSON for drift detection across deploys                       | `metaengine.SerializablePlan` — `Serialize()`, `DeserializePlan(data)`                |
| **PlanDiff**         | Compares two plans: added/removed/changed engines, queries, layouts                | `metaengine.PlanDiff(prev, current)` to `PlanDiffResult`                              |
| **Manifest**         | A plan fingerprint saved to disk for CI drift detection                            | `metaengine.NewManifest(plan)`, `SaveManifest(path)`, `LoadManifest(path)`            |
| **DryRun**           | Plan option that returns PlanResult without modifying engine state                 | `metaengine.WithDryRun()` — no DDL, no pinning                                        |
| **ExplainPlan**      | Human-readable plan explanation: engine per query + diagnostics                    | `Store.ExplainPlan()` — string output for debugging                                   |
| **Doctor**           | Health report: per-collection engine, replication, persistence                     | `Store.Doctor(ctx)` — string output for operations                                    |
| **Collections**      | Introspects all planned collections: engine, ADT, layout, persistence, replication | `Store.Collections()` — returns `[]CollectionInfo`                                    |
| **ReplicationMode**  | Returns the replication topology for a single query                                | `Store.ReplicationMode(queryName)` — `ReplicationNone`, `ReplicationLeaderless`, etc. |
| **Persistence**      | Returns the survivability classification for a single query                        | `Store.Persistence(queryName)` — `PersistenceVolatile` or `PersistencePersistent`     |

### Engine Capabilities (Optional Interfaces)

Engines implement optional capability interfaces (ISP) — the planner checks at runtime which are present:

| Term                       | Definition                                                                     | Context                                                  |
| -------------------------- | ------------------------------------------------------------------------------ | -------------------------------------------------------- |
| **PushdownScan**           | Pushes filter/sort/limit into the engine's query language (SQL WHERE/ORDER BY) | `metaengine.PushdownScan` interface — SQLite, DuckDB, PG |
| **StreamingScan**          | OOM-safe lazy iteration via `iter.Seq2`                                        | `metaengine.StreamingScan` interface                     |
| **RawValueReader**         | Reads raw JSON bytes without decoding to `any` (avoids double-decode tax)      | `metaengine.RawValueReader` interface                    |
| **AtomicAppender**         | Atomic version-check-then-append (optimistic concurrency on streams)           | `metaengine.AtomicAppender` interface                    |
| **SnapshotBackend**        | Engine capability for storing decider snapshots                                | `metaengine.SnapshotBackend` interface                   |
| **MapUpdater**             | Atomic read-modify-write for Map ADT (transactional)                           | `metaengine.MapUpdater` interface                        |
| **AggregateReader**        | SQL-level aggregation avoiding loading all rows into Go                        | `metaengine.AggregateReader` interface                   |
| **GroupedAggregateReader** | SQL-level GROUP BY aggregation (vectorized on columnar engines)                | `metaengine.GroupedAggregateReader` interface            |
| **MultiAggregateReader**   | Multiple aggregate queries in one call (batch optimization)                    | `metaengine.MultiAggregateReader` interface              |

### Hot Operations

| Term               | Definition                                                                  | Context                                                                                        |
| ------------------ | --------------------------------------------------------------------------- | ---------------------------------------------------------------------------------------------- |
| **TieredStore**    | Wraps a primary Store with replicas; writes fan out, reads use primary      | `metaengine.NewTieredStore(primary, replicas...)`                                              |
| **Watcher**        | Subscribe to real-time value updates on a collection (push notifications)   | `metaengine.NewWatcher[V](store, coll)`                                                        |
| **SwapEngine**     | Replaces an engine at runtime, reassigning queries — zero-downtime upgrades | `Store.SwapEngine(name, newEngine)`                                                            |
| **PrefetchCache**  | Caches scan results beyond requested limit for next page                    | `metaengine.PrefetchCache` — eliminates redundant round-trips                                  |
| **MapUpdateTyped** | Typed atomic read-modify-write for a single Map key (transactional)         | `metaengine.MapUpdateTyped[V](store, coll, key, fn)` — fn receives `*V`, returns updated value |

### Key Codec (LSM Backends)

LSM-style engines (Pebble, Badger) encode composite keys as byte slices for ordered range scans. The `keycodec` package provides deterministic key construction with shared prefix/range helpers — the foundation for every prefix iteration in LSM backends.

| Term                 | Definition                                                            | Context                                                                                    |
| -------------------- | --------------------------------------------------------------------- | ------------------------------------------------------------------------------------------ |
| **MapKey**           | Encodes a collection + key into a sortable byte slice                 | `keycodec.MapKey(col, key)` — `\x00`-separated for prefix isolation                        |
| **CollectionPrefix** | The byte prefix for all keys in a collection (range scan start point) | `keycodec.CollectionPrefix(col)` — used by ScanBackend for prefix iteration                |
| **CounterKey**       | Encodes a counter key with a separate prefix namespace                | `keycodec.CounterKey(col, ckey)`                                                           |
| **StreamKey**        | Encodes a stream-scoped key for StreamLog ADT (per-stream ordering)   | `keycodec.StreamKey(col, sid, seq)` — `(collection, streamID, sequence)` composite         |
| **MultimapKey**      | Encodes a multimap entry with monotonic sequence suffix               | `keycodec.MultimapKey(col, key, seq)` — enables ordered retrieval within a multimap bucket |

### Engines

Concrete `Engine` implementations, each in its own subpackage:

| Engine       | Constructor                         | Storage           | Module                           |
| ------------ | ----------------------------------- | ----------------- | -------------------------------- |
| **Memory**   | `NewMemoryEngine()`                 | Hash map          | `metaengine` (core)              |
| **SQLite**   | `sqliteengine.NewSQLiteEngine(...)` | B-Tree            | `metaengine/sqliteengine/`       |
| **Pebble**   | `pebbleengine.NewPebbleEngine(...)` | LSM tree          | `metaengine/pebbleengine/`       |
| **Badger**   | `badgerengine.NewBadgerEngine(...)` | LSM tree          | `metaengine/badgerengine/`       |
| **DuckDB**   | `duckdbengine.New(...)`             | Columnar OLAP     | `metaengine/duckdbengine/` (CGo) |
| **Postgres** | `pgengine.New(...)`                 | Client-server     | `metaengine/pgengine/`           |
| **Dgraph**   | `dgraphengine.New(...)`             | Graph-native      | `metaengine/dgraphengine/`       |
| **Iroh**     | `irohengine.Replicated(local, ...)` | CRDT (leaderless) | `metaengine/irohengine/`         |
| **Graph**    | `graphadapter.New()`                | In-memory graph   | `metaengine/graphadapter/`       |

> **Calibratable:** Engines implementing `metaengine.Calibratable` accept benchmark-calibrated cost measurements (`SetCalibration(CalibrationCosts)`) to replace theoretical Big-O estimates with real ns/op data. Currently: Memory, SQLite, Pebble, Badger, DuckDB.

> **Iroh transport tiers:** The Iroh engine adds CRDT convergence via a pluggable `Transport`. Three tiers form the testing pyramid:
>
> - `irohengine.NewInProcessNetwork()` — goroutine function calls (fastest, no CGo)
> - `irohengine/loopback/` — real TCP connections with length-prefix framing (no CGo)
> - `irohengine/quic/` — real QUIC streams via iroh-go (requires CGo + pre-compiled static lib)
>
> CRDT-safe operations (merge by construction): MapSet (LWW), SetAdd (OR-Set), CounterIncrement (PN-Counter), MultiAdd, LogAppend. Non-CRDT ops (e.g. MapUpdate) stay local-only.

### Projection Adapter

The bridge from a metaengine `Store` to the `projection.Projection` interface, enabling a metaengine-planned collection to participate in the standard [projection lifecycle](DOMAIN_LANGUAGE.md#projection-lifecycle) (`projectionhost.Host`).

| Term               | Definition                                                       | Context                                                                                         |
| ------------------ | ---------------------------------------------------------------- | ----------------------------------------------------------------------------------------------- |
| **Adapter**        | Wraps a metaengine Store collection as a `projection.Projection` | `projectionadapter.New(name, store, decoder)` — implements `Name()`, `Handle()`, `EventTypes()` |
| **EventDecoder**   | Function decoding raw event payloads into typed fold inputs      | `projectionadapter.WithEventDecoder(fn)` — replaces per-event-type switch/case boilerplate      |
| **EventWithID**    | Wrapper bundling stream ID with event payload for keyed folds    | `projectionadapter.EventWithID[P]` — exported type so consumers don't reinvent the wrapper      |
| **NewTypeDecoder** | Builds an EventDecoder from a `catalog.Registry` type registry   | `projectionadapter.NewTypeDecoder(registry)` — replaces 70+ line decoder switch/case            |

### Testing & Benchmarking

| Term                 | Definition                                                       | Context                                                                                                                              |
| -------------------- | ---------------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------ |
| **enginetest**       | Exported engine test harness for cross-engine parity             | `metaengine/enginetest` — `RunTransactionalTest`, `RunStreamLogBackendTest`, `RunAtomicAppenderTest`                                 |
| **adttest**          | Exported ADT test harness: 10-ADT matrix for engine implementors | `metaengine/adttest` — `RunMatrix(t, factories)` — imported by all engine submodules                                                 |
| **metaengine/bench** | Cross-engine benchmark module: columnar vs pushdown vs Memory    | `metaengine/bench` — parity tests + planner/layout/materialize benchmarks. Test-only (no non-test exports). CGo required for DuckDB. |

---

## Shared Terms (defined in [Domain Language](DOMAIN_LANGUAGE.md))

The metaengine is built on concepts shared with the rest of the library. These terms are defined once in the root [Domain Language](DOMAIN_LANGUAGE.md):

| Shared Term     | Where defined                                                                     | Metaengine context                                                                                           |
| --------------- | --------------------------------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------ |
| **Record**      | [Record (Structural Foundation)](DOMAIN_LANGUAGE.md#record-structural-foundation) | The structural base type for events and commands; `OnRecord` folds receive the full Record context           |
| **Stream**      | [Event Sourcing](DOMAIN_LANGUAGE.md#event-sourcing)                               | The consistency boundary; StreamLog ADT provides per-stream ordering                                         |
| **Decider**     | [CQRS](DOMAIN_LANGUAGE.md#cqrs)                                                   | Pure-function command handler; decider state can be served by a metaengine Store                             |
| **Projection**  | [Projection Lifecycle](DOMAIN_LANGUAGE.md#projection-lifecycle)                   | The consumer-side read model; the Projection Adapter bridges metaengine collections to the lifecycle         |
| **Tombstone**   | [Event Sourcing](DOMAIN_LANGUAGE.md#event-sourcing)                               | Soft-delete signal; in the v2 vision (ADR-0114), tombstones become domain events in the metaengine (planned) |
| **Event Store** | [Event Sourcing](DOMAIN_LANGUAGE.md#event-sourcing)                               | The source of truth; the metaengine reads events via the projection adapter to build projections             |

---

## Interface Hierarchy — Engine

```
metaengine.Engine = Profile() EngineProfile + Closer
  Optional ADT backends (ISP — engines implement what they support):
    MapBackend:        MapSet/MapGet/MapDelete
    MapUpdater:        MapUpdate (atomic read-modify-write)
    ScanBackend:       MapScan (filter+sort in Go)
    PushdownScan:      PushdownMapScan (SQL WHERE/ORDER BY pushdown)
    StreamingScan:     StreamScan (iter.Seq2 — OOM-safe)
    SetBackend:        SetAdd/SetContains
    CounterBackend:    CounterIncrement/CounterGet
    MultimapBackend:   MultiAdd/MultiGet
    LogBackend:        LogAppend/LogTail
    StreamLogBackend:  StreamAppend/StreamRead/StreamVersion
    VectorBackend:     VectorInsert/VectorSearch
    SearchBackend:     SearchInsert/SearchQuery
    SpatialBackend:    SpatialInsert/SpatialRange
  Optional capabilities:
    LayoutPlanner:     CreateLayout(collection, plan)
    LayoutPlanApplier: ApplyLayout(collection, plan) — columnar-native
    RawValueReader:    MapGetRaw (skip decode)
    AtomicAppender:    AppendWithVersion (optimistic concurrency)
    SnapshotBackend:   SaveSnapshot/LoadSnapshot
    AggregateReader:   Aggregate(ctx, coll, specs) — SQL-level
    GroupedAggregateReader: GroupedAggregate(ctx, coll, specs) — GROUP BY
    MultiAggregateReader: MultiAggregate(ctx, coll, specs[]) — batch
    VersionedStorage:  GetAsOf(ctx, coll, key, timestamp) — temporal
    Calibratable:      SetCalibration(CalibrationCosts) — replace estimates with real benchmarks
  Lifecycle:
    PlanRule:          Apply(ctx, PlanContext, result) error
    RulePipeline:      Run(ctx, PlanContext, result) error
```

---

## Terms We Avoid (Metaengine-Specific)

| Instead of             | We say                       | Why                                                                                                                                      |
| ---------------------- | ---------------------------- | ---------------------------------------------------------------------------------------------------------------------------------------- |
| "Manual query routing" | "Cost-based planner"         | The metaengine assigns queries to engines by estimated cost — never hand-pick an engine per query                                        |
| "Hand-written DDL"     | "LayoutPlan auto-generation" | The planner generates DDL from declared filter/sort fields — never write table schemas manually when a LayoutPlanner engine is available |
| "God Store"            | "TieredStore" or "Store"     | One store serving all patterns at high cost — the planner splits queries across engines by cost profile                                  |
| "Theoretical cost"     | "Calibrated cost"            | Big-O estimates are replaced by real ns/op measurements via `Calibratable` — never reason about performance without calibration          |

---

## Verification

The code block below is scanned by `cmd/doc-check` to verify every symbol
referenced in this document still exists in the codebase. Do not remove.

```go
package metaengine_domain_language_verification

import (
	// Core planner
	"github.com/larsartmann/go-cqrs-lite/metaengine/v4"
	"github.com/larsartmann/go-cqrs-lite/metaengine/v4/keycodec"

	// Sub-packages (no own go.mod — part of metaengine/v4)
	"github.com/larsartmann/go-cqrs-lite/metaengine/adttest"
	"github.com/larsartmann/go-cqrs-lite/metaengine/enginetest"

	// Engine modules (each own go.mod)
	"github.com/larsartmann/go-cqrs-lite/metaengine/sqliteengine/v4"
	"github.com/larsartmann/go-cqrs-lite/metaengine/pebbleengine/v4"
	"github.com/larsartmann/go-cqrs-lite/metaengine/pgengine/v4"
	"github.com/larsartmann/go-cqrs-lite/metaengine/duckdbengine/v4"
	"github.com/larsartmann/go-cqrs-lite/metaengine/badgerengine/v4"
	"github.com/larsartmann/go-cqrs-lite/metaengine/dgraphengine/v4"
	"github.com/larsartmann/go-cqrs-lite/metaengine/irohengine/v4"
	"github.com/larsartmann/go-cqrs-lite/metaengine/graphadapter/v4"

	// Projection adapter
	"github.com/larsartmann/go-cqrs-lite/metaengine/projectionadapter/v4"
)

var _ = []any{
	// Core
	metaengine.Plan,
	metaengine.NewMemoryEngine,
	metaengine.ExecuteTyped[any, any],
	metaengine.Query[any, any],
	metaengine.On[any],
	metaengine.OnTyped[any],
	metaengine.OnRecord[any],
	metaengine.NewReader[any],
	metaengine.NewQueryBuilder[any],
	metaengine.NewWatcher[any],
	metaengine.NewTieredStore,
	metaengine.ShouldMaterialize,
	metaengine.ReplayCost,
	metaengine.MaterializeCost,
	metaengine.Serialize,
	metaengine.DeserializePlan,
	metaengine.PlanDiff,
	metaengine.NewManifest,
	metaengine.WithDryRun,
	metaengine.WithColumnarLayout,
	metaengine.FilterOnField[any],
	metaengine.SortOnField[any],

	// ADT constants
	metaengine.ADTMap,
	metaengine.ADTCounter,
	metaengine.ADTGraph,
	metaengine.ADTVector,

	// Complexity + Layouts
	metaengine.ComplexityO1,
	metaengine.LayoutRow,
	metaengine.LayoutColumnar,

	// Persistence + Replication
	metaengine.PersistenceVolatile,
	metaengine.PersistencePersistent,
	metaengine.ReplicationNone,
	metaengine.ReplicationLeaderless,

	// Diagnostics
	metaengine.DiagLevelScream,
	metaengine.DiagLevelWarn,
	metaengine.DefaultWriteAmplificationBudget,

	// Types
	metaengine.Delta{},
	metaengine.Edge{},
	metaengine.Skip{},
	metaengine.EngineProfile{},
	metaengine.Store{},

	// Optional interfaces (nil assertions)
	metaengine.VersionedStorage(nil),
	metaengine.GroupedAggregateReader(nil),
	metaengine.MultiAggregateReader(nil),
	metaengine.Calibratable(nil),

	// Read Patterns (complete set)
	metaengine.ReadMembership,
	metaengine.ReadMultiLookup,
	metaengine.ReadLogTail,
	metaengine.ReadVectorSearch,
	metaengine.ReadFullTextSearch,
	metaengine.ReadSpatialRange,

	// Auto folds + typed inputs
	metaengine.AutoInsert[any, any],
	metaengine.AutoUpdate[any, any],
	metaengine.AutoDelete[any],
	metaengine.AutoCRUD[any, any, any, any],
	metaengine.MapUpdateTyped[any],
	metaengine.Embedding{},
	metaengine.IndexedText{},

	// Engines
	sqliteengine.NewSQLiteEngine,
	pebbleengine.NewPebbleEngine,
	pgengine.New,
	duckdbengine.New,
	duckdbengine.NewFromDB,
	badgerengine.NewBadgerEngine,
	badgerengine.NewBadgerEngineFromDB,
	dgraphengine.New,
	dgraphengine.NewFromClient,
	irohengine.Replicated,
	graphadapter.New,
	graphadapter.NewWithDriver,

	// Projection adapter
	projectionadapter.New,
	projectionadapter.NewWithDecoder,
	projectionadapter.WithEventDecoder,
	projectionadapter.NewTypeDecoder,
	projectionadapter.EventWithID[any],

	// Test harnesses
	adttest.RunMatrix,
	enginetest.RunTransactionalTest,
	enginetest.RunStreamLogBackendTest,
	enginetest.RunAtomicAppenderTest,

	// Key codec (symbols documented in Key Codec section)
	keycodec.MapKey,
	keycodec.CollectionPrefix,
	keycodec.CounterKey,
	keycodec.StreamKey,
	keycodec.MultimapKey,
	keycodec.LogPrefix,
	keycodec.LogKey,
	keycodec.EncodeJSON,
	keycodec.DecodeCounterValue,
}
```
