# The Layered Architecture: Separating Data Models from Storage Engines

> **STATUS: DESIGN PROPOSAL** (2026-07-31). This document proposes an architectural
> refinement to `metaengine/`. It does not describe shipped behavior. It is the
> unifying lens that connects the [taxonomy](../research/database-architecture-taxonomy.md),
> the [design](meta-engine-design.md), and the [DataFusion lessons](2026-07-31_datafusion-lessons-for-metaengine.md)
> into one coherent decomposition.

---

## Table of Contents

1. [The Thesis: Two Axes, Not One](#1-the-thesis-two-axes-not-one)
2. [The Universal Cost Matrix](#2-the-universal-cost-matrix)
3. [Temporality as a Storage Capability](#3-temporality-as-a-storage-capability)
4. [The Missing Three: Vector, Search, Spatial](#4-the-missing-three-vector-search-spatial)
5. [Hybrid Engines: One Database, Multiple Layouts](#5-hybrid-engines-one-database-multiple-layouts)
6. [The Event Store Is Also a Storage Choice](#6-the-event-store-is-also-a-storage-choice)
7. [Concrete Before/After: What Changes for the Consumer](#7-concrete-beforeafter-what-changes-for-the-consumer)
8. [Migration Path: Actual Code Diffs](#8-migration-path-actual-code-diffs)
9. [What This Does NOT Change (Brutal Honesty)](#9-what-this-does-not-change-brutal-honesty)
10. [Relationship to Existing Canon](#10-relationship-to-existing-canon)

---

## 1. The Thesis: Two Axes, Not One

### The DDIA Insight

Martin Kleppmann's *Designing Data-Intensive Applications* devotes two foundational chapters to
what are actually **two orthogonal axes**:

- **Chapter 2 (Data Models):** How data is shaped and queried — relational, document, graph,
  time-series, wide-column, vector, search. This is the **interface contract**: what operations
  the consumer sees.
- **Chapter 3 (Storage & Retrieval):** How data physically lives on disk — B-trees, LSM-trees,
  hash indexes, columnar layouts, append-only logs, HNSW graphs, inverted indexes. This is the
  **storage engine**: the physical data structure under the hood.

The central observation: **databases historically bundle these two axes into one product.**
PostgreSQL is Relational × B-Tree. Cassandra is Wide-Column × LSM. Neo4j is Graph ×
adjacency-list. But the axes are independent — the same data model can be backed by many
storage engines, and the same storage engine can power many data models (see the [cross-reference
matrix](../research/database-architecture-taxonomy.md) in the taxonomy doc).

### Today's Problem: The Axes Are Bundled

`metaengine.Engine` bundles both axes into one type:

```go
// TODAY — one type, two concerns conflated
type Engine interface {
    Profile() EngineProfile  // "I support Map ADT" (data model) + "I cost 100ns/op" (storage)
    Close() error
}
// Plus optional interfaces: MapBackend, ScanBackend, CounterBackend, GraphBackend, ...
```

When you call `NewPebbleEngine()`, you get an object that simultaneously declares:

1. "I implement `MapBackend`, `SetBackend`, ..." → **data model** (which ADTs)
2. "I have `NsPerOp: 7ns, NsPerRead: 7ns`" → **storage engine** (LSM characteristics)
3. "I am Pebble" → **driver** (the concrete library)

These three concerns are inseparable in the current type system. The planner can ask "which
engine is cheapest for this Map query?" but it **cannot ask** "this query needs a Map — which
storage engines are available, and is LSM or B-Tree better for this volume?" The storage-engine
characteristics are opaque, buried inside a calibrated nanosecond number.

### Why This Matters Now

The bundling was harmless with three engines (Memory, SQLite, Pebble) that each cover all 7
ADTs. It becomes a **structural blocker** as the catalog grows:

| Scenario | Bundled (today) | Separated (proposed) |
| --- | --- | --- |
| Add Vector search | Write a whole new `VectorEngine` with its own opaque cost profile | Define `VectorBackend` (data model) once; any HNSW-capable storage engine can serve it |
| Add a ClickHouse columnar engine | Duplicate all 7 ADT backend impls; cost profile is opaque | Declare `LayoutColumnar`; the cost matrix already knows columnar makes Counter O(1) |
| Reason about *why* Pebble beats SQLite for a Log | Can only compare `NsPerOp` numbers | The matrix says: LSM makes append O(1); B-Tree makes it O(logN). The *reason* is visible. |
| Combine pgvector (Vector × HNSW) | Must implement as a monolithic engine | pgvector = `VectorBackend` (data model) × `LayoutHNSW` (storage) × pgx (driver). Composable. |

The separation turns "add a new specialized database" from a monolithic engine-writing task
into a **combinatorial** task: pick a data model, pick a storage engine, wire a driver. The
planner does the matching.

### Two Axes Plus One Cross-Cutting Capability

The decomposition is **two orthogonal axes** plus **one cross-cutting capability**:

| Layer | What it controls | Who declares it | Example values |
| --- | --- | --- | --- |
| **Data Model** (Axis 1) | The interface contract (ADT) — what operations exist | Developer (via fold return type) | Map, Set, Counter, Graph, Vector, Search, Spatial |
| **Storage Engine** (Axis 2) | The physical layout — how bytes live on disk | Operator (engine config) | B-Tree, LSM, Hash, Columnar, HNSW, Inverted Index |
| **Temporality** (cross-cutting) | Whether cells are versioned (as-of reads) or latest-only | Operator (engine capability + retention config) | Versioned with retention, or latest-only |

Temporality is not a third axis because it doesn't combine freely with every data model — it
modifies *how* a storage engine stores cells, regardless of which data model sits on top.
Retention (`max_versions`, `max_age`) is a config knob on temporality, not a separate axis: it
only makes sense when versioning is enabled.

---

## 2. The Universal Cost Matrix

### The Single Source of Truth

This is the heart of the document. One matrix. Memorized by the planner, inherited by every
driver for free. Rows = data models (ADTs). Columns = storage engines. Cells = read complexity.

```
                     B+Tree     LSM       Hash      Columnar   HNSW/IVF   Inverted   R-Tree    Append-Log   In-Memory
────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────
Map (point lookup)   O(logN)    O(logN)   O(1)      O(N/k)▾    —          —          —         O(N)         O(1)
Set (membership)     O(logN)    O(logN)   O(1)      O(N/k)▾    —          —          —         O(N)         O(1)
SortedMap (filter+)  O(logN)    O(logN)   —         O(N/k)✓     —          —          —         O(N)         O(N)
Counter (aggregate)  O(N)       O(N)      —         O(1)✓✓     —          —          —         O(N)         O(N)
Graph (traverse d)   O(N)       O(N)      —         O(N)       —          —          —         O(N)         O(d)✓
Multimap (1→many)    O(logN)    O(logN)   O(1)      O(N/k)▾    —          —          —         O(N)         O(1)
Log (append tail)    O(logN)    O(1)✓     —         O(1)✓      —          —          —         O(1)✓✓       O(1)
Vector (k-NN)        —          —         —         —          O(logN)‡   —          —         —            O(N)‡
Search (full-text)   —          —         —         —          —          O(1)§      —         —            —
Spatial (geo range)  —          —         —         —          —          —          O(logN)   —            —

Legend:
  ✓✓  optimal — this is what the storage engine was designed for
  ✓   good fit
  ▾   technically possible but suboptimal (planner warns at scale)
  ‡   approximate — recall vs speed tradeoff (HNSW) or brute-force scan (In-Memory)
  §   term→postings is O(1); ranking adds O(k log k) for BM25/TF-IDF
  —   wrong engine for this ADT (planner rejects)
```

### Why a Static Matrix Works

This matrix is **structural** — it encodes Big-O classes, not calibrated nanosecond numbers.
The properties it captures (hash is O(1) for point lookup, columnar is O(1) for aggregation,
LSM is O(1) for append) are **mathematical facts about data structures**, not benchmarks that
drift with hardware. A driver declares which column it occupies; the matrix tells the planner
the complexity for free. No per-engine calibration needed for structural decisions.

Calibrated `NsPerOp` numbers (from benchmarks like `calibration_bench_test.go`) are still used
for **tie-breaking within the same complexity class** — when two engines are both O(logN), the
one with lower nanoseconds wins. But the structural decision ("is columnar or B-Tree right for
this Counter query?") comes from the matrix, not from benchmarks.

### Scale Thresholds (The Statistical Layer)

Complexity alone doesn't pick the winner — volume matters. The
[assumptions doc](meta-engine-assumptions-and-query-planning.md) already publishes threshold
tables. The matrix is the structural layer; thresholds add the statistical layer:

```
For SortedMap (filtered scan):
  N < 10K       → any engine is fine (In-Memory O(N) is fast enough)
  N < 1M        → B+Tree or LSM (O(logN) index seek + bounded scan)
  N < 100M      → B+Tree or LSM with strong indexes
  N > 100M      → Columnar wins (O(N/k) sequential scan beats O(logN) random I/O at this scale)
                 (at N=100M with k=20 columns, columnar reads 5M values sequentially vs
                  100M random I/O seeks — sequential wins by ~100x)
```

The planner combines both layers: `cost = complexity_matrix[ADT][storage] × volume_factor(N)`.

### Write Costs (The Other Half)

The matrix above shows **read** complexity. Write cost matters too — it determines write
amplification (how many projections an event updates):

```
                     B+Tree              LSM                 Hash               Columnar           HNSW/IVF           Inverted           R-Tree             Append-Log
──────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────
Write (insert)       O(logN)             O(1) amortized      O(1)               O(1) batched       O(logN)             O(doc_len)         O(logN)             O(1)
Write amplification  High (page rewrite) Low (sequential)    Low                Low (column merge) Medium (graph rewire) Medium (postings update) Medium      Minimal
Best for write rate  Low-moderate        High                High               High (batch)       Moderate            Moderate            Moderate            Unlimited
```

The planner uses this to estimate write amplification: if a `UserCreated` event feeds a Map on
Pebble (LSM, O(1) write) and a Counter on ClickHouse (columnar, O(1) batched), total write cost
is low. If it feeds a Map on SQLite (B+Tree, O(logN) page rewrite) and a Graph on Neo4j
(adjacency rewrite), write amplification is higher.

---

## 3. Temporality as a Storage Capability

### The BigTable Insight

Google Cloud BigTable (and its open-source siblings: HBase, Cassandra, ScyllaDB) stores every
cell as a **timestamped version**:

```
(row_key, column_family, column, timestamp) → value
```

Writes don't overwrite — they create a new timestamped version. Reads can specify:

- **Latest** (default) — behaves like a normal mutable KV
- **As-of timestamp T** — returns the value that was current at T, in **O(1)**
- **Range [T1, T2]** — returns the full version history of that cell

This is not a query-engine feature layered on top. It is a **storage-engine capability**: the
LSM stores all versions, and the reader picks which to materialize. Garbage collection policies
(`max_versions=N`, `max_age=7d`) control how much history survives compaction.

### Why This Is Categorically Powerful for Event Sourcing

The event log is already a temporal structure — `LoadToTimestamp` and `LoadToVersion` exist on
the event store. But **projections are currently amnesiac**: `MapSet` overwrites the previous
value, so "what was this user's balance last Tuesday?" requires replaying events and folding —
O(events up to T). A versioned storage engine makes that **O(1)** by keeping the cell history
natively.

This means temporal projection reads have **multiple physical strategies**, and the planner
chooses based on engine capability:

| Query needs as-of? | Engine versions cells? | Planner picks | Cost | Storage cost |
| --- | --- | --- | --- | --- |
| Yes | Yes (BigTable/Cassandra) | **Native as-of read** | **O(1)** | Higher (keeps versions) |
| Yes | No (plain Pebble/SQLite) | **Event-log replay** | O(events to T) | Low (latest only) |
| No | Yes | Read latest, GC the rest | O(1) | Configurable via retention |
| No | No | Normal latest-only read | O(1) | Low |

The top-left cell is the BigTable-enabled future: O(1) time travel with zero replay cost. The
bottom row is today's world. The middle row is the **honest degradation** the planner warns
about.

### Retention: The Cost Knob

Versioned storage is powerful but expensive — keeping every version of every cell costs disk
space proportional to write rate × history depth. The **retention policy** is the knob:

```go
type RetentionPolicy struct {
    MaxVersions int           // 1 = latest only (cheap), 0 = unlimited (expensive)
    MaxAge      time.Duration // 0 = keep forever, else GC versions older than this
}
```

`MaxVersions = 1` collapses a versioned engine into a plain mutable KV. The capability is
present but unused — the engine keeps only the latest version, paying zero history-storage
overhead. This is the operator's "save on data cost" lever: full temporal querying when you need
it, cheap latest-only storage when you don't, on the **same engine**, selected at deploy time.

This maps directly to real database GC mechanisms:

| Database | Retention mechanism | Equivalent config |
| --- | --- | --- |
| **BigTable / HBase** | GC rules (max versions + max age per column family) | `MaxVersions`, `MaxAge` |
| **Cassandra / Scylla** | Cell TTL + `compaction_window` | `MaxAge` (TTL), tombstone compaction |
| **Datomic / XTDB** | Built-in (immutable indexes, manual pruning) | Application-controlled |
| **DuckDB** | `ASOF JOIN` (temporal queries without versioning) | n/a (query-time, not storage-time) |

### The Mechanics: How Versioned Folds Work

This is the part that needs careful design. When a projection handler processes events, how do
timestamps flow?

```
Event arrives:  UserCreated{ID: "u1", Name: "Alice"} at wall-time T1
                → projection handler calls MapSetAt("users", "u1", view, T1)

Event arrives:  UserRenamed{ID: "u1", Name: "Bob"} at wall-time T2
                → projection handler calls MapSetAt("users", "u1", view, T2)

As-of query:    "What was user u1's name at time T1.5?"
                → MapGetAt("users", "u1", T1.5) → returns {Name: "Alice"} in O(1)
```

The cell timestamp comes from the **event's metadata timestamp** (or event version, if using
version-based temporality), not from wall-clock at write time. This ensures correctness during
replay: when the projection host replays historical events, each event's original timestamp is
preserved, so the versioned cells reflect the true temporal order.

### The Hard Interaction: Retention vs Query Correctness

What happens when retention GC's versions that a live query needs?

```
Timeline:
  T1: UserCreated → cell versioned at T1
  T2: UserRenamed → cell versioned at T2
  T3: retention GC (max_age=1h) → version T1 pruned (older than 1 hour)
  T4: Query: "user at T1" → VERSION NOT FOUND
```

This is a **correctness-critical** interaction. The planner must know the retention policy and
warn (or refuse) when:

- A query declares `AsOf` intent
- But the retention policy would GC versions older than the query's temporal range

The mitigation: either (a) widen the retention policy, (b) fall back to event-log replay for
that query, or (c) document that temporal queries beyond the retention window are unsupported.
This is a deployment-time decision, surfaced by the planner.

### Relationship to the DataFusion Doc's TemporalAnchor

The [DataFusion lessons](2026-07-31_datafusion-lessons-for-metaengine.md) doc already proposes
`TemporalAnchor` on the **logical plan** — the *query* declares "I want state as-of version 42."
That doc covers the **logical layer**: which replay strategy to use (snapshot+delta, full
replay, cached projection).

This document adds the **storage layer**: whether the *engine itself* versions cells natively.
The two are complementary:

```
Logical layer (DataFusion doc):    "AsOf: version 42" → which replay strategy?
Storage layer (this doc):          "Is the engine versioned?" → O(1) read or O(N) replay?
```

Together they're complete. The logical plan carries the temporal intent; the storage-engine
capability determines whether it's cheap or expensive.

---

## 4. The Missing Three: Vector, Search, Spatial

These three data models have **no ADT, no backend interface, no fold return type** today. They
are the deepest gap. Adding them requires type-system design (data-model-first), not just driver
work. Each one has a specific design challenge that makes it harder than "just add an
interface."

### Vector (Similarity Search)

```go
// Embedding is the fold return type for vector projections.
type Embedding struct {
    ID     any
    Vector []float32 // e.g., 1536-dim from OpenAI text-embedding-3-small
}

// metaengine.On(DocumentIndexed{}, func(e DocumentIndexed) metaengine.Embedding {
//     return metaengine.Embedding{ID: e.ID, Vector: e.Embedding}
// })
```

**What makes this hard — three design decisions:**

1. **Distance metric is not universal.** Cosine similarity, L2 (Euclidean), and inner-product
   are the three common metrics. Different use cases need different ones. The interface must
   either parameterize the metric at query time (flexible but prevents engine-specific
   optimization) or at declaration time (the `Query[...]` call fixes it for that projection).
  pgvector solves this with operator classes (`<=>` for cosine, `<->` for L2, `<#>` for inner
   product). The metaengine needs an equivalent.

2. **Hybrid search (vector + metadata filter).** Real-world RAG pipelines don't just do pure
   k-NN — they do "find the 10 most similar documents *that are tagged 'finance' and published
   after 2024*." This is vector search + filtered scan combined. The `VectorBackend` interface
   must compose with `FilterSpec` (which already exists for `PushdownScan`). pgvector and
   Qdrant support this natively; the interface must express it.

3. **Approximation parameters.** HNSW has `ef_construction`, `ef_search`, and `M` parameters
   that trade recall for speed. IVF has `nlist` and `nprobe`. The interface must expose these
   as query-time or declaration-time options without leaking engine-specific details.

**Sketch interface (honest about what's unresolved):**

```go
type VectorBackend interface {
    VectorAdd(ctx context.Context, collection string, id any, vec []float32) error
    VectorSearch(ctx context.Context, collection string, query []float32, k int,
        opts ...VectorSearchOption) ([]VectorHit, error)
    VectorDelete(ctx context.Context, collection string, id any) error
}

type VectorHit struct {
    ID    any
    Score float64 // interpretation depends on metric (similarity: higher is better; distance: lower)
}

// VectorSearchOption addresses design decision #2 (hybrid) and #3 (approximation):
type VectorSearchOption func(*vectorSearchConfig)

func WithVectorFilter(filters []FilterSpec) VectorSearchOption  // hybrid: vector + metadata
func WithVectorMetric(m VectorMetric) VectorSearchOption        // cosine, l2, inner_product
func WithEFSearch(ef int) VectorSearchOption                    // HNSW approximation knob
```

**Storage engines:** HNSW (pgvector, Qdrant, Weaviate), IVF (Milvus), or brute-force scan
(In-Memory, for small sets under ~10K vectors where exact search is faster than approximate).

### Search (Full-Text)

```go
// IndexedText is the fold return type for full-text search projections.
type IndexedText struct {
    ID     any
    Fields map[string]string // field name → text to index (title, body, tags)
}
```

**What makes this hard — two design decisions:**

1. **Query DSL complexity.** Full-text search is not "grep with ranking." Real queries use
   boolean operators (`invoice AND NOT draft`), phrase queries (`"exact phrase"`), fuzzy
   matching (`invoice~2` — Levenshtein distance 2), proximity (`"quick fox"~5` — within 5
   words), and field-scoped search (`title:invoice AND body:payment`). The interface must
   either accept a structured query type (complex but type-safe) or a query string (simple but
   engine-dependent in syntax).

2. **Analyzer chain.** Text is processed through tokenization, lowercasing, stemming,
   stop-word removal, and synonym expansion *before* indexing. Different analyzer chains
   produce different indexes for the same text. The projection declaration must specify the
   analyzer config (or accept the engine default), and the same config must apply at query time.

**Sketch interface:**

```go
type SearchBackend interface {
    SearchIndex(ctx context.Context, collection string, docID any, fields map[string]string) error
    SearchQuery(ctx context.Context, collection string, query SearchQuery, limit int) ([]SearchHit, error)
}

type SearchQuery struct {
    Text    string            // the query string (engine-interpreted)
    Fields  []string          // restrict to specific fields (nil = all)
    Filters []FilterSpec      // combine full-text with structured filters (hybrid)
}

type SearchHit struct {
    ID    any
    Score float64 // BM25, TF-IDF, or DFR depending on engine
}
```

**Storage engines:** Inverted index (Elasticsearch, OpenSearch, Meilisearch, Bleve). PostgreSQL
can do this via `tsvector`/`tsquery` (less powerful but embedded). SQLite has FTS5.

### Spatial (Geo Range)

```go
// Geometry is the fold return type for spatial projections.
type Geometry struct {
    ID   any
    Type SpatialType    // Point, Polygon, LineString
    Data []float64      // [lng, lat, ...] flattened coordinates
}
```

**What makes this hard — one design decision:**

1. **Query shape variety.** "Within radius" is the simplest spatial query, but real
   applications need: bounding box (`WITHIN( SW, NE )`), polygon containment
   (`WITHIN_POLYGON( geom )`), nearest-neighbor (`KNN( point, k )`), and intersection
   (`INTERSECTS( geom )`). Each maps to different index operations on the storage engine. R-Tree
   handles bounding-box containment in O(logN) but polygon containment requires a secondary
   filter step.

**Sketch interface:**

```go
type SpatialBackend interface {
    SpatialIndex(ctx context.Context, collection string, id any, geom Geometry) error
    SpatialWithin(ctx context.Context, collection string, q SpatialQuery) ([]SpatialHit, error)
}

type SpatialQuery struct {
    Type    SpatialQueryType // WithinRadius, WithinBox, WithinPolygon, Intersects
    Center  [2]float64       // for WithinRadius
    Radius  float64          // meters, for WithinRadius
    Box     [4]float64       // SW_lng, SW_lat, NE_lng, NE_lat for WithinBox
    Polygon []float64        // flattened [lng, lat, lng, lat, ...] for WithinPolygon/Intersects
    Limit   int
}
```

**Storage engines:** R-Tree (PostGIS, SQLite RTree module), Geohash (Redis GEO, Elasticsearch
geo_point). Google S2 / H3 are cell-based alternatives for spherical geometry.

### Why These Are Type-System Decisions, Not Driver Work

Each of the three requires:

1. **A new fold return type** (`Embedding`, `IndexedText`, `Geometry`) — the developer's
   declaration mechanism
2. **A new backend interface** (`VectorBackend`, `SearchBackend`, `SpatialBackend`) — the
   engine contract
3. **A new ADT constant** (`ADTVector`, `ADTSearch`, `ADTSpatial`) — for the planner's
   classification
4. **New cost matrix entries** — the row in §2's matrix
5. **Honest interface design** — each has a design challenge (distance metrics, query DSL,
   query shapes) that must be resolved *before* any driver can plug in

This is data-model-first work. Until the types exist, no driver can plug in. This is the
strategic priority — it unblocks semantic search, RAG, and geospatial use cases that are
currently impossible to express.

---

## 5. Hybrid Engines: One Database, Multiple Layouts

### The Problem with "One Layout Per Engine"

The model so far assumes each engine has ONE storage layout. Reality is messier:

| Database | Primary layout | Secondary layout | Via |
| --- | --- | --- | --- |
| **PostgreSQL** | B+Tree (tables) | Inverted index (full-text via `tsvector`) | Built-in |
| **PostgreSQL + PostGIS** | B+Tree (tables) | R-Tree (spatial via GiST index) | Extension |
| **PostgreSQL + pgvector** | B+Tree (tables) | HNSW (vector via `vector` extension) | Extension |
| **DuckDB** | Columnar (analytics) | — but has `ASOF JOIN` (temporal-ish at query time) | Built-in |
| **SQLite** | B+Tree (tables) | Inverted index (FTS5 module) | Built-in module |
| **ClickHouse** | Columnar (MergeTree) | Skip indexes (bloom filter, set) | Built-in |

PostgreSQL alone can serve **four different ADTs** from four different storage layouts: Map
(B+Tree), Search (tsvector/inverted), Spatial (PostGIS R-Tree), and Vector (pgvector HNSW) —
all in one database, one connection, one transaction.

### How the Model Handles This

An engine declares **multiple layouts**, one per ADT it serves:

```go
// PROPOSED — EngineProfile carries a layout per ADT, not one global layout
type EngineProfile struct {
    Name     string
    Layouts  map[ADT]StorageLayout  // NEW: which physical layout serves each ADT
    Supports map[ADT]Complexity      // unchanged: complexity per ADT

    NsPerOp    float64               // unchanged: calibrated cost for tie-breaking
    NsPerRead  float64
    NsPerWrite float64
}
```

PostgreSQL's profile would declare:

```go
EngineProfile{
    Name: "postgres",
    Layouts: map[ADT]StorageLayout{
        ADTMap:     LayoutBTree,
        ADTSearch:  LayoutInverted,   // via tsvector
        ADTSpatial: LayoutRTree,      // via PostGIS GiST
        ADTVector:  LayoutHNSW,       // via pgvector
    },
    Supports: map[ADT]Complexity{
        ADTMap:     ComplexityOLogN,
        ADTSearch:  ComplexityO1,
        ADTSpatial: ComplexityOLogN,
        ADTVector:  ComplexityOLogN,
    },
}
```

This is **more honest** than a single `Layout` field. It says: "I serve Map via B+Tree, Search
via inverted index, Vector via HNSW." The planner reads the layout *for the specific ADT* and
gets the right complexity from the matrix.

For engines that use one layout for everything (Pebble = LSM for all ADTs), the map just has the
same value in every cell. No overhead.

### Why This Matters for Adoption

The single-layout model would force pgvector into its own `metaengine/pgvectorengine/` module,
separate from the PostgreSQL engine — even though they share the same connection, transaction,
and process. The multi-layout model lets one PostgreSQL engine serve four ADTs from four
layouts, which is how real consumers use it.

---

## 6. The Event Store Is Also a Storage Choice

### The Blind Spot

This document (and the metaengine generally) focuses on **projections** — the read side. But
the event store itself is a storage-engine choice. The taxonomy lists append-only log as a
storage engine, and the event store IS an append-only log:

| Event store implementation | Data Model | Storage Engine | Why chosen |
| --- | --- | --- | --- |
| `SQLEventStore` (SQLite) | Ordered KV (aggregate+version key) | B+Tree | Transactional, queryable |
| `SQLEventStore` (Postgres) | Ordered KV | B+Tree | Transactional, durable, replicable |
| `PebbleEventStore` | Ordered KV | LSM | Write-optimized (events are append-only) |
| `MemoryStore` | Ordered KV | In-Memory | Speed, testing |

The layered model applies here too: the event store's access pattern is **Ordered KV**
(load-by-aggregate = prefix scan, append = put). The storage engine determines write throughput
(LSM wins) vs. transactional durability (B+Tree wins). The taxonomy doc
([§4: Relevance to go-cqrs-lite](../research/database-architecture-taxonomy.md)) already makes
this exact point.

### What This Means for the Planner

Today, the event store is chosen at deployment time (operator picks SQLite vs Pebble vs
Postgres) and the metaengine has no visibility into it. The planner optimizes *projections*
only.

A more ambitious framing: the planner could reason about the **whole stack** — event store +
projections — and recommend an event store storage engine based on write rate. If the operator
declares "10K events/sec," the planner might recommend LSM (Pebble) for the event store AND
columnar (DuckDB) for the Counter projections. This is out of scope for the current metaengine
(which only plans projections), but the layered model makes it *thinkable*.

---

## 7. Concrete Before/After: What Changes for the Consumer

### Example 1: Vector Search (Impossible Today → Possible After)

**BEFORE (today):** No way to express this. The developer must build a separate pgvector
integration outside the metaengine, manually wire it as a projection, and bypass the planner.

```go
// TODAY — not expressible in metaengine. Must hand-wire:
store, _ := metaengine.Plan(engines, findUser, listByStatus)
// findSimilar is NOT a metaengine query — it's custom code against pgvector directly
```

**AFTER (with ADTVector):** The developer declares it like any other query:

```go
type FindSimilar struct {
    Query []float32
    K     int
}
type SimilarDoc struct {
    ID    string
    Score float64
}

findSimilar := metaengine.Query[FindSimilar, SimilarDoc]("similar_docs",
    metaengine.On(DocumentIndexed{}, func(e DocumentIndexed) metaengine.Embedding {
        return metaengine.Embedding{ID: e.ID, Vector: e.Embedding}
    }),
)

store, _ := metaengine.Plan(
    []metaengine.Engine{sqliteEngine, pgvectorEngine},
    findUser, listByStatus, findSimilar,
)

// The planner sees Embedding return type → ADTVector → checks which engines serve it
// → pgvectorEngine (LayoutHNSW) is the only one → assigns findSimilar to pgvector
// → EXPLAIN says: "similar_docs → pgvector (HNSW, O(logN) approx)"

results, _ := metaengine.ExecuteTyped[FindSimilar, SimilarDoc](
    ctx, store, FindSimilar{Query: qvec, K: 10})
```

### Example 2: Temporal Query (O(N) Replay → O(1) Native)

**BEFORE (today):** No temporal projection support. "Balance as of last Tuesday" requires
custom replay code.

**AFTER (with VersionedStorage):**

```go
type AccountBalance struct {
    ID      AccountID
    AsOf    *time.Time  // nil = latest, non-nil = point-in-time
}

balanceQuery := metaengine.Query[AccountBalance, BalanceResult]("balance",
    metaengine.On(DepositRecorded{}, func(e DepositRecorded) (AccountID, BalanceResult) {
        return e.AccountID, BalanceResult{Amount: e.Amount}
    }),
    metaengine.On(WithdrawalRecorded{}, func(e WithdrawalRecorded, prev BalanceResult) BalanceResult {
        prev.Amount -= e.Amount
        return prev
    }),
)

// Operator provides BigTable engine with versioned cells:
store, _ := metaengine.Plan(
    []metaengine.Engine{bigtableEngine}, // implements VersionedStorage
    balanceQuery,
)
// EXPLAIN: "balance → bigtable (Versioned LSM, as-of O(1))"

// Latest balance — O(1)
latest, _ := metaengine.ExecuteTyped[AccountBalance, BalanceResult](
    ctx, store, AccountBalance{ID: acctID})

// Historical balance — also O(1) on BigTable (vs O(events) replay on Pebble)
historical, _ := metaengine.ExecuteTyped[AccountBalance, BalanceResult](
    ctx, store, AccountBalance{ID: acctID, AsOf: &lastTuesday})
```

### Example 3: Planner EXPLAIN Output Changes When a New Engine Arrives

```
# Deployment A: SQLite only
EXPLAIN:
  findUser    → sqlite (B+Tree, O(logN))     ✓
  listByStatus→ sqlite (B+Tree, O(logN))     ✓
  countUsers  → sqlite (B+Tree, O(N))        ⚠ DEGRADED (Counter on B+Tree = full scan)

# Deployment B: SQLite + ClickHouse added
EXPLAIN:
  findUser    → sqlite (B+Tree, O(logN))     ✓
  listByStatus→ sqlite (B+Tree, O(logN))     ✓
  countUsers  → clickhouse (Columnar, O(1))  ✓✓ OPTIMAL (reassigned: columnar Counter is native)
```

The developer's code is identical between Deployments A and B. The planner reads the cost matrix
and reassigns `countUsers` when a better engine becomes available. This is the "smart
combination" in action.

---

## 8. Migration Path: Actual Code Diffs

The migration is **backward compatible** and **incrementally shippable**. Five steps, each
independently deployable.

### Step 1: Add `Layouts` to `EngineProfile` (refactor, no behavior change)

```diff
 // metaengine/engine.go
 type EngineProfile struct {
     Name     string
+    Layouts  map[ADT]StorageLayout  // which physical layout serves each ADT
     Supports map[ADT]Complexity
     NsPerOp    float64
     NsPerRead  float64
     NsPerWrite float64
 }
+
+type StorageLayout string
+
+const (
+    LayoutBTree     StorageLayout = "btree"
+    LayoutLSM       StorageLayout = "lsm"
+    LayoutHash      StorageLayout = "hash"
+    LayoutColumnar  StorageLayout = "columnar"
+    LayoutAppendLog StorageLayout = "append_log"
+    LayoutHNSW      StorageLayout = "hnsw"
+    LayoutInverted  StorageLayout = "inverted"
+    LayoutRTree     StorageLayout = "rtree"
+    LayoutInMemory  StorageLayout = "in_memory"
+)
```

Then update each engine's `Profile()`:

```diff
 // metaengine/memory_engine.go — Memory uses hash/in-memory for everything
 func (m *memoryEngine) Profile() EngineProfile {
     return EngineProfile{
         Name:    "memory",
+        Layouts: map[ADT]StorageLayout{
+            ADTMap: LayoutInMemory, ADTSet: LayoutInMemory,
+            ADTCounter: LayoutInMemory, ADTGraph: LayoutInMemory,
+            ADTSortedMap: LayoutInMemory, ADTLog: LayoutInMemory,
+            ADTMultimap: LayoutInMemory,
+        },
         // ... Supports unchanged
```

```diff
 // metaengine/pebbleengine/engine.go — Pebble uses LSM for everything
 func (e *pebbleEngine) Profile() metaengine.EngineProfile {
     return metaengine.EngineProfile{
         Name:       "pebble",
+        Layouts: map[metaengine.ADT]metaengine.StorageLayout{
+            metaengine.ADTMap: metaengine.LayoutLSM, /* ... all LSM */
+        },
         // ... Supports unchanged
```

```diff
 // metaengine/sqlite_engine.go — SQLite uses B+Tree for everything
 func (e *sqliteEngine) Profile() EngineProfile {
     return EngineProfile{
         Name: "sqlite",
+        Layouts: map[ADT]StorageLayout{
+            ADTMap: LayoutBTree, /* ... all B+Tree */
+        },
         // ... Supports unchanged
```

**No behavior change.** The planner still uses `Supports` (ADT → complexity) for assignment and
`NsPerOp` for tie-breaking. `Layouts` is metadata, ready for Step 2.

### Step 2: Add the Static Cost Matrix (planner uses it for structural reasoning)

```go
// metaengine/cost_matrix.go
var costMatrix = map[ADT]map[StorageLayout]Complexity{
    ADTMap: {
        LayoutBTree: ComplexityOLogN, LayoutLSM: ComplexityOLogN,
        LayoutHash: ComplexityO1, LayoutColumnar: ComplexityON, LayoutInMemory: ComplexityO1,
    },
    ADTCounter: {
        LayoutBTree: ComplexityON, LayoutLSM: ComplexityON,
        LayoutColumnar: ComplexityO1, LayoutInMemory: ComplexityON,
    },
    // ... full matrix from §2
}
```

The planner consults this for structural classification ("is columnar or B+Tree right for this
Counter?") and falls back to `NsPerOp` for tie-breaking within the same complexity class. Still
no behavior change for existing 3 engines — they produce the same assignments.

### Step 3: Add `VersionedStorage` Interface (additive, no existing engine implements it)

```go
// metaengine/engine.go
type VersionedStorage interface {
    MapSetAt(ctx context.Context, collection string, key any, value any, at time.Time) error
    MapGetAt(ctx context.Context, collection string, key any, at time.Time) (any, bool, error)
    MapHistory(ctx context.Context, collection string, key any, from, to time.Time) ([]VersionedValue, error)
}
```

No existing engine implements it. It's ready for BigTable/Cassandra drivers. The planner checks
for it at runtime (like `PushdownScan` today) and routes `AsOf` queries accordingly.

### Step 4: Add New ADTs (additive — doesn't affect existing queries)

Add `ADTVector`, `ADTSearch`, `ADTSpatial` constants + `VectorBackend`, `SearchBackend`,
`SpatialBackend` interfaces + `Embedding`, `IndexedText`, `Geometry` fold return types. These
are purely additive — existing queries don't reference them. See §4 for the interface sketches
and their design challenges.

### Step 5: Add Temporal Query Signal (additive)

The planner detects `AsOf *time.Time` fields in query input structs (mirroring how `*Cursor`
signals pagination today). If present and no `VersionedStorage` engine is available, emit a
degradation warning. If present and a versioned engine IS available, route to it.

---

## 9. What This Does NOT Change (Brutal Honesty)

### For the Current Three Engines: Zero Behavior Change

With only Memory, SQLite, and Pebble — all covering all 7 ADTs via the same layout families —
the axis separation produces **identical assignments** to today's planner. The separation pays
off only when:

- A second storage-engine family is available (e.g., adding ClickHouse columnar alongside
  SQLite B+Tree)
- Specialized query patterns exist (Vector, Search, Spatial)
- Scale pushes past single-engine comfort (high write rate, large volumes, temporal reads)

For single-engine deployments, the layers are transparent — the planner collapses them into one
assignment. The complexity is paid only when the value is realized.

### The Cost Matrix Must Be Correct

A wrong cell produces wrong planner decisions. The matrix encodes mathematical facts about data
structures (hash is O(1), columnar Counter is O(1)), but the **scale thresholds** and
**calibrated tie-breaking numbers** are empirical. Both must be tested:

- Structural correctness: property-based tests verifying matrix invariants
- Calibration: the `calibration_bench_test.go` pattern should extend to every engine
- Threshold validation: regression tests that verify the planner picks the right engine at
  boundary volumes

### The Temporal Replay Fallback Is Expensive

When no versioned engine is available and a query needs as-of reads, the fallback is
event-log replay — O(events to T). For a stream with 100K events queried for state at event 90K,
that's 90K folds per query. The planner must warn loudly, and the operator must either provide
a versioned engine or accept the cost. Retention policy interacts with correctness: if the
operator sets `max_age=1h`, temporal queries older than 1 hour are impossible without replay.

### The Missing Three Interfaces Are Hard Design Work

The sketches in §4 are **starting points, not final designs**. Each has unresolved design
challenges:

- **Vector:** distance metric parameterization, hybrid search composition, approximation knobs
- **Search:** query DSL (structured vs string), analyzer chain config
- **Spatial:** query shape variety (radius, box, polygon, intersection)

These must be designed before any driver can implement them. Getting the interface wrong means
every driver is constrained by the wrong abstraction. This is the highest-risk part of the
proposal.

---

## 10. Relationship to Existing Canon

This document is the **unifying lens**. It does not replace the existing docs — it connects
them:

| Document | Role | This doc's relationship |
| --- | --- | --- |
| [database-architecture-taxonomy.md](../research/database-architecture-taxonomy.md) | Reference: 13 interface models × 12 storage engines | Provides the raw material. This doc identifies which models map to existing ADTs and which need new ones. |
| [meta-engine-project-definition.md](meta-engine-project-definition.md) | Vision: why cross-engine view selection is novel | Provides the research framing. This doc provides the architectural decomposition that makes it implementable. |
| [meta-engine-design.md](meta-engine-design.md) | Technical design: cost profiles, optimizer, deployments | Provides the cost model and optimizer algorithm. This doc refines `EngineProfile` to separate the two axes the design doc conflates, and adds `Layouts` for hybrid engines. |
| [meta-engine-assumptions-and-query-planning.md](meta-engine-assumptions-and-query-planning.md) | Scale thresholds: N → data structure | Provides the volume thresholds. This doc's cost matrix is the structural layer those thresholds operate on. |
| [DataFusion lessons](2026-07-31_datafusion-lessons-for-metaengine.md) | Engineering patterns: rule pipeline, logical/physical split | Provides the logical/physical plan separation and tier classification. This doc adds the storage-engine axis and the temporal-storage capability that the DataFusion doc treats only at the logical layer. |

### The Single Sentence

**Today's `metaengine.Engine` bundles "what operations I support" (data model) with "how I store
bytes" (storage engine) in one type. Separating them — plus adding temporality as a storage
capability with a retention knob — turns the planner from "pick the cheapest opaque engine"
into "compose the optimal backend from independent layers," and unblocks Vector, Search, and
Spatial as first-class query patterns instead of impossible-to-express gaps.**
