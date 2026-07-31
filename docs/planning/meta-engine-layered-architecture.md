# The Layered Architecture: Separating Data Models from Storage Engines

> **STATUS: DESIGN PROPOSAL** (2026-07-31). This document proposes an architectural
> refinement to `metaengine/`. It does not describe shipped behavior. It is the
> unifying lens that connects the [taxonomy](../research/database-architecture-taxonomy.md),
> the [design](meta-engine-design.md), and the [DataFusion lessons](2026-07-31_datafusion-lessons-for-metaengine.md)
> into one coherent decomposition.

---

## Table of Contents

1. [The Thesis: Two Axes, Not One](#1-the-thesis-two-axes-not-one)
2. [The Four Decision Dimensions](#2-the-four-decision-dimensions)
3. [Axis 1 — Data Models (The Interface Layer)](#3-axis-1--data-models-the-interface-layer)
4. [Axis 2 — Storage Engines (The Physical Layer)](#4-axis-2--storage-engines-the-physical-layer)
5. [Axis 3 — Temporality (The Time Dimension)](#5-axis-3--temporality-the-time-dimension)
6. [Axis 4 — Retention (The Cost Knob)](#6-axis-4--retention-the-cost-knob)
7. [Backends = Smart Combinations](#7-backends--smart-combinations)
8. [The Cost Matrix (The Heart)](#8-the-cost-matrix-the-heart)
9. [The Planner's Decision Logic](#9-the-planners-decision-logic)
10. [The Missing Three: Vector, Search, Spatial](#10-the-missing-three-vector-search-spatial)
11. [Migration Path: From Bundled Engine to Separated Layers](#11-migration-path-from-bundled-engine-to-separated-layers)
12. [What Is Hard (Honest Assessment)](#12-what-is-hard-honest-assessment)
13. [Relationship to Existing Canon](#13-relationship-to-existing-canon)

---

## 1. The Thesis: Two Axes, Not One

### The DDIA Insight

Martin Kleppmann's *Designing Data-Intensive Applications* devotes two foundational chapters to
what are actually **two orthogonal axes**:

- **Chapter 2 (Data Models):** How data is shaped and queried — relational, document, graph,
  time-series, wide-column. This is the **interface contract**: what operations the consumer
  sees.
- **Chapter 3 (Storage & Retrieval):** How data physically lives on disk — B-trees, LSM-trees,
  hash indexes, columnar layouts, append-only logs. This is the **storage engine**: the
  physical data structure under the hood.

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

---

## 2. The Four Decision Dimensions

The axis separation reveals that backend selection is actually a **four-dimensional** decision,
not the one-dimensional "pick an engine" it appears today:

| Dimension | What it controls | Who declares it | Example values |
| --- | --- | --- | --- |
| **Data Model** | The interface contract (ADT) — what operations exist | Developer (via fold return type) | Map, Set, Counter, Graph, Vector, Search, Spatial |
| **Storage Engine** | The physical layout — how bytes live on disk | Operator (engine config) | B-Tree, LSM, Hash, Columnar, HNSW, Inverted Index |
| **Temporality** | Whether cells are versioned (as-of reads) or latest-only | Operator (engine capability) | Versioned (BigTable), Latest-only (Pebble) |
| **Retention** | How much history to keep — the storage-cost knob | Operator (GC policy) | `max_versions=1` (cheap), `max_versions=0` (full history) |

Dimensions 1–2 are the DDIA axes. Dimension 3 (temporality) is the BigTable insight — some
storage engines natively timestamp every cell, making point-in-time reads O(1). Dimension 4
(retention) is the cost lever that lets the operator trade history depth for storage savings.

**A concrete backend is a point in this four-dimensional space:**

```
BigTable   = Wide-Column(Map) × LSM × Versioned × GC(max_versions=N)
DuckDB     = Relational × Columnar × Latest-only × n/a
pgvector   = Vector × B-Tree+HNSW × Latest-only × n/a
Cassandra  = Wide-Column(Map) × LSM × Versioned × TTL(per-cell)
Pebble     = KV(Map) × LSM × Latest-only × n/a
SQLite     = Relational × B-Tree × Latest-only × n/a
```

The planner's job: given a query pattern + volume + latency budget + temporal requirement,
find the cheapest point in this space from the engines the operator provided.

---

## 3. Axis 1 — Data Models (The Interface Layer)

### What Exists Today

The metaengine derives data-model requirements from **fold return types** — the developer never
declares "I need a Map." The return type IS the declaration:

| Fold signature | Return type | ADT | Read pattern |
| --- | --- | --- | --- |
| `func(e) (K, V)` | `(Key, Value)` | `Map` | Point lookup |
| `func(e) K` | `Key` | `Set` | Membership test |
| `func(e) Delta` | `Delta` | `Counter` | Aggregate |
| `func(e) Edge` | `Edge` | `Graph` | Traversal |
| `func(e) MultiEntry` | `MultiEntry` | `Multimap` | One-to-many lookup |
| `func(e) Append` | `Append` | `Log` | Append-only tail |
| `func(e, prev V) V` | `Value` | `Map` (update) | Read-modify-write |
| `Remove[V]()` | Sentinel | Delete | Remove from projection |

Seven ADTs. This is elegant and correct — but it covers only **5 of the 13 interface models**
in the taxonomy. Five entire families of databases have no representation:

### What Is Missing

| Interface model | Taxonomy section | Required ADT | Required fold signature | Status |
| --- | --- | --- | --- | --- |
| **Vector / Similarity** | Taxonomy §12 | `Vector` (k-NN) | `func(e) Embedding` | **No ADT exists** |
| **Search / Full-text** | Taxonomy §6 | `Search` (inverted index) | `func(e) IndexedText` | **No ADT exists** |
| **Spatial / Geo** | (implied by §6 geo) | `Spatial` (R-tree / geohash) | `func(e) Geometry` | **No ADT exists** |
| **Wide-Column** | Taxonomy §7 | `Map` (partition-key) | Already served by `Map` ADT | Driver-only gap |
| **Time-Series** | Taxonomy §5 | `Counter` + `Log` | Already served by existing ADTs | Driver-only gap |
| **Triple Store / RDF** | Taxonomy §9 | `Graph` | Already served by `Graph` ADT | Driver-only gap |
| **Datalog** | Taxonomy §10 | (esoteric, skip) | — | Out of scope |
| **Streaming** | Taxonomy §11 | (transport, not storage) | — | Out of scope |
| **Object Storage** | Taxonomy §8 | (blob, not queryable) | — | Out of scope |
| **Multi-Model** | Taxonomy §13 | (union of above) | — | Emerges from composition |

The first three — **Vector, Search, Spatial** — are the deep gap. They have no fold return type,
no backend interface, no read pattern. You cannot express "find the 10 nearest neighbors to this
embedding" or "full-text search for 'invoice' ranked by relevance" or "all users within 5km of
this point" as a metaengine projection today. These require new ADTs (see
[§10](#10-the-missing-three-vector-search-spatial)).

The next three — **Wide-Column, Time-Series, Triple Store** — are NOT blocked at the type level.
Their access patterns already map to existing ADTs (Map, Counter/Log, Graph). Adding Cassandra
or InfluxDB is "just" writing a driver that implements `MapBackend` or `CounterBackend` with the
right cost profile. The architecture is ready; the drivers don't exist yet.

---

## 4. Axis 2 — Storage Engines (The Physical Layer)

### What a Storage Engine Is

A storage engine is the physical data structure that holds bytes on disk (or in RAM). DDIA
Chapter 3 catalogs the major families. Each has a characteristic cost profile for the basic
operations (point lookup, range scan, insert, ordered iteration):

| Storage engine | Point lookup | Range scan | Insert | Write amplification | Best at |
| --- | --- | --- | --- | --- | --- |
| **B+Tree** | O(logN) | O(logN + k) | O(logN) | High (rewrites pages) | Read-heavy OLTP |
| **LSM-Tree** | O(logN) | O(logN + k) | O(1) amortized | Low (sequential) | Write-heavy, append-only |
| **Hash** | O(1) | ❌ (no range) | O(1) | Low | Point-only KV |
| **Columnar** | O(N/k) | O(N/k) | O(1) batched | Low (column compression) | Analytics, aggregation |
| **Append-only log** | O(N) | O(N) | O(1) | Minimal | Event sourcing (the log itself) |
| **CoW B-Tree** | O(logN) | O(logN + k) | O(logN) | High (path copy) | Lock-free snapshot reads |
| **HNSW / IVF** | O(logN) approx | ❌ | O(logN) | Medium | Vector similarity (k-NN) |
| **Inverted index** | O(1) term→postings | O(k) | O(doc_len) | Medium | Full-text search |
| **R-Tree / Geohash** | O(logN) | O(logN + k) | O(logN) | Medium | Spatial range queries |
| **In-memory (any)** | O(1)–O(logN) | O(logN + k) | O(1) | n/a (volatile) | Speed, not persistence |

### Why This Is Separable From the Data Model

The same data model performs differently on different storage engines. The **Map ADT** (point
lookup by key) is the clearest example:

```
Map ADT point lookup cost by storage engine:
  Hash:         O(1)     ← fastest (Redis, Memcached)
  LSM-Tree:     O(logN)  ← fast, persistent (Pebble, RocksDB)
  B+Tree:       O(logN)  ← fast, persistent (Postgres, SQLite)
  Columnar:     O(N/k)   ← slow (must scan a column stripe) — WRONG engine for this ADT
  Inverted idx: ❌       ← not designed for key-value (WRONG engine entirely)
```

The planner should know this. Today, it only sees "Engine A costs 7ns, Engine B costs 100ns" —
opaque numbers. With separation, it sees "this Map query needs O(1) point lookup → Hash or LSM
storage → which drivers are available?" The **reason** for the cost difference is visible and
composable.

### The Cost Matrix Becomes Two-Dimensional

Today's `EngineProfile.Supports map[ADT]Complexity` is a 1D mapping (ADT → complexity) per
engine. With separation, the cost matrix becomes **ADT × StorageEngine** — a 2D lookup that any
new driver inherits for free:

```
                    B+Tree      LSM         Hash        Columnar    HNSW        Inverted    R-Tree
Map (point)         O(logN)     O(logN)     O(1)        O(N/k)*     ❌          ❌          ❌
Set (membership)    O(logN)     O(logN)     O(1)        O(N/k)*     ❌          ❌          ❌
Counter             O(N)        O(N)        ❌          O(1)†       ❌          ❌          ❌
Graph (traversal)   O(N)        O(N)        ❌          O(N)        ❌          ❌          ❌
Multimap            O(logN)     O(logN)     O(1)        O(N/k)      ❌          ❌          ❌
Log (append)        O(logN)     O(1)        ❌          O(1)        ❌          ❌          ❌
Vector (k-NN)       ❌          ❌          ❌          ❌          O(logN)‡    ❌          ❌
Search (full-text)  ❌          ❌          ❌          ❌          ❌          O(1)§       ❌
Spatial (range)     ❌          ❌          ❌          ❌          ❌          ❌          O(logN)

*  Columnar point lookup requires scanning a column stripe — technically possible but the wrong engine
†  Columnar SUM/COUNT is native O(1) — this is why ClickHouse makes Counter optimal
‡  HNSW gives approximate O(logN) with recall/speed tradeoff
§  Inverted index maps term → postings list in O(1), then ranks
```

This matrix is the **planner's lookup table**. A driver declares which row(s) and column(s) it
occupies. The planner reads the cell at (required ADT, available storage engine) and gets the
complexity. No per-engine cost calibration needed for structural decisions — the matrix is
universal.

---

## 5. Axis 3 — Temporality (The Time Dimension)

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
| No | Yes | Read latest, GC the rest | O(1) | Configurable |
| No | No | Normal latest-only read | O(1) | Low |

The top-left cell is the BigTable-enabled future: O(1) time travel with zero replay cost. The
bottom row is today's world. The middle row is the **honest degradation** the planner warns
about.

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

### The `VersionedStorage` Capability

```go
// VersionedStorage is a cross-cutting storage-engine capability.
// Engines that natively version cells (BigTable, Cassandra, HBase) implement
// this for O(1) as-of reads. Engines that don't (Pebble, SQLite, Memory) omit
// it, and the planner falls back to event-log replay.
type VersionedStorage interface {
    MapSetAt(ctx context.Context, collection string, key any, value any, at time.Time) error
    MapGetAt(ctx context.Context, collection string, key any, at time.Time) (any, bool, error)
    MapHistory(ctx context.Context, collection string, key any, from, to time.Time) ([]VersionedValue, error)
}

type VersionedValue struct {
    Value any
    At    time.Time
}
```

The planner checks for this interface at runtime, exactly as it checks for `PushdownScan` or
`RawValueReader` today. If present and the query has an `AsOf` field, it uses the native path.
If absent, it falls back to replay with a degradation warning.

---

## 6. Axis 4 — Retention (The Cost Knob)

### The Tradeoff

Versioned storage is powerful but expensive — keeping every version of every cell costs disk
space proportional to write rate × history depth. The **retention policy** is the knob that lets
the operator control this tradeoff:

```go
// RetentionPolicy controls storage cost vs history depth on versioned engines.
type RetentionPolicy struct {
    MaxVersions int           // 1 = latest only (cheap), 0 = unlimited (expensive)
    MaxAge      time.Duration // 0 = keep forever, else GC versions older than this
}
```

This maps directly to real database GC mechanisms:

| Database | Retention mechanism | Equivalent config |
| --- | --- | --- |
| **BigTable / HBase** | GC rules (max versions + max age per column family) | `MaxVersions`, `MaxAge` |
| **Cassandra / Scylla** | Cell TTL + `compaction_window` | `MaxAge` (TTL), tombstone compaction |
| **Datomic / XTDB** | Built-in (immutable indexes, manual pruning) | Application-controlled |
| **DuckDB** | Manual (history table, app-managed) | Application-controlled |
| **Postgres** | SQL:2011 temporal tables / `tstzrange` | Application-managed |

### The "Make It Null" Knob

The operator's key decision: `MaxVersions = 1` collapses a versioned engine into a plain mutable
KV. The capability is present but unused — the engine keeps only the latest version, paying zero
history-storage overhead. This is the "save on data cost" lever: full temporal querying when you
need it, cheap latest-only storage when you don't, on the **same engine**, selected at deploy
time.

```go
// Example: BigTable configured for latest-only (cheap, no time travel)
btEngine, _ := bigtable.New(cluster, bigtable.WithRetention(
    metaengine.RetentionPolicy{MaxVersions: 1},
))

// Example: same engine, full history (expensive, O(1) time travel)
btEngine, _ := bigtable.New(cluster, bigtable.WithRetention(
    metaengine.RetentionPolicy{MaxVersions: 0}, // unlimited
))
```

The planner sees the retention policy and knows whether as-of reads will be O(1) (versions
retained) or will require replay (versions GC'd to 1).

---

## 7. Backends = Smart Combinations

A concrete backend is a **combination** of all four dimensions plus a driver. The planner's job
is to find the optimal combination from what the operator provides.

### The Combination Table

| Backend | Data Model | Storage Engine | Temporal | Retention | Driver |
| --- | --- | --- | --- | --- | --- |
| **PostgreSQL** | Relational (Map, SortedMap) | B+Tree | Latest-only | n/a | pgx |
| **SQLite** | Relational (Map, SortedMap) | B+Tree | Latest-only | n/a | modernc.org/sqlite |
| **MySQL** | Relational (Map, SortedMap) | B+Tree | Latest-only | n/a | go-sql-driver/mysql |
| **Pebble** | KV (Map) | LSM | Latest-only | n/a | cockroachdb/pebble |
| **DuckDB** | Relational (Map, SortedMap, Counter) | Columnar | Latest-only (ASOF JOIN) | Manual | duckdb-go |
| **BigTable / HBase** | Wide-Column (Map) | LSM | **Versioned** | **GC (versions/age)** | googleapis |
| **Cassandra / Scylla** | Wide-Column (Map) | LSM | **Versioned** | **TTL + compaction** | gocql |
| **ClickHouse** | Relational (SortedMap, Counter) | Columnar | Latest-only | TTL per table | clickhouse-go |
| **Neo4j / Memgraph** | Graph (Graph) | Adjacency list | Latest-only | n/a | neo4j-go-driver |
| **InfluxDB** | Time-Series (Counter, Log) | TSM (LSM variant) | Latest-only | retention policy | influxdb-client |
| **pgvector** | Vector (k-NN) | HNSW over B+Tree | Latest-only | n/a | pgx + extension |
| **Milvus / Qdrant** | Vector (k-NN) | HNSW / IVF | Latest-only | n/a | milvus-sdk-go |
| **Elasticsearch** | Search (full-text) | Inverted index | Latest-only | ILM policies | olivere/elastic |
| **Redis** | KV (Map, Set, Counter) | Hash / append-only | Latest-only | eviction policies | go-redis |

### How the Planner Combines

The planner does NOT pick a backend by name. It picks by **capability match**:

```
INPUT:
  Query pattern:  Filter("status") + Sort("joined_at") + Count, Volume=50M
  Temporal need:  AsOf = nil (latest only)
  Available engines: [SQLite(B+Tree), ClickHouse(Columnar), Pebble(LSM)]

STEP 1 — Classify the ADT:
  Filter + Sort → SortedMap
  Count → Counter

STEP 2 — Look up the cost matrix:
  SortedMap × B+Tree     = O(logN)   ← SQLite
  SortedMap × Columnar   = O(N/k)    ← ClickHouse (fast at scale: N/20 columns)
  SortedMap × LSM        = O(logN)   ← Pebble
  Counter × Columnar     = O(1)      ← ClickHouse (native SUM) ✓ OPTIMAL
  Counter × B+Tree       = O(N)      ← SQLite (full scan)
  Counter × LSM          = O(N)      ← Pebble (full scan)

STEP 3 — Check temporal requirement:
  AsOf = nil → no versioned storage needed → any engine qualifies

STEP 4 — Assign by volume threshold:
  Volume 50M exceeds B+Tree/LSM optimal range for scan-heavy queries
  → ClickHouse (Columnar) wins for both SortedMap and Counter at this scale

OUTPUT:
  SortedMap → ClickHouse (columnar scan, O(N/k))
  Counter   → ClickHouse (native columnar aggregation, O(1))
  Write amplification: 2 projections (both on ClickHouse, same engine)
```

The developer's code is identical whether the result is SQLite or ClickHouse. The planner reads
the matrix and picks. This is the "smart combination."

---

## 8. The Cost Matrix (The Heart)

The full decision matrix the planner consults. Rows = data models (ADTs). Columns = storage
engines. Cells = read complexity. Empty cells = wrong engine for that ADT (planner rejects or
warns).

```
Data Model          B+Tree    LSM       Hash      Columnar   HNSW/IVF   Inverted   R-Tree    In-Memory
─────────────────────────────────────────────────────────────────────────────────────────────────────────
Map (point)         O(logN)   O(logN)   O(1)      O(N/k)▾    —          —          —         O(1)
Set (membership)    O(logN)   O(logN)   O(1)      O(N/k)▾    —          —          —         O(1)
SortedMap (filter)  O(logN)   O(logN)   —         O(N/k)✓     —          —          —         O(N)
Counter             O(N)      O(N)      —         O(1)✓✓     —          —          —         O(N)
Graph (traverse)    O(N)      O(N)      —         O(N)       —          —          —         O(d)✓
Multimap            O(logN)   O(logN)   O(1)      O(N/k)▾    —          —          —         O(1)
Log (append)        O(logN)   O(1)✓     —         O(1)✓      —          —          —         O(1)
Vector (k-NN)       —         —         —         —          O(logN)‡   —          —         O(logN)‡
Search (full-text)  —         —         —         —          —          O(1)§      —         —
Spatial (range)     —         —         —         —          —          —          O(logN)   —

Legend:
  ✓✓  optimal (this is what the engine was designed for)
  ✓   good fit
  ▾   technically possible but suboptimal (planner warns)
  ‡   approximate (recall vs speed tradeoff)
  §   term→postings is O(1); ranking adds O(k log k)
  —   wrong engine / not applicable (planner rejects)
```

### Scale Thresholds

Complexity alone isn't enough — the planner also considers volume. The
[assumptions doc](meta-engine-assumptions-and-query-planning.md) already publishes threshold
tables. The matrix above is the structural layer; thresholds add the statistical layer:

```
For SortedMap (filtered scan):
  N < 10K       → any engine is fine (In-Memory O(N) is fast enough)
  N < 1M        → B+Tree or LSM (O(logN) index seek)
  N < 100M      → B+Tree or LSM with strong indexes
  N > 100M      → Columnar (O(N/k) — scan is faster than random I/O at this scale)
```

The planner combines: `cost = complexity_matrix[ADT][storage] × volume_threshold(N)`.

---

## 9. The Planner's Decision Logic

### The Algorithm

```
INPUT:
  - Declared query patterns (from developer: fold return types → ADTs, filters, sorts)
  - Available engines (from operator: each declares DataModel + StorageEngine + Temporal + Retention)
  - Volume hints and latency budgets (from query declarations)
  - Temporal requirements (AsOf fields in query input structs)

ALGORITHM:
  1. CLASSIFY each query pattern → required ADT(s)
     - func(e) (K,V)       → Map
     - func(e) Embedding   → Vector
     - AsOf field present  → temporal read required

  2. For each (query, ADT) pair:
     a. Look up cost_matrix[ADT][storage] for every available engine
     b. Filter out engines where the cell is "—" (wrong engine) or "▾" (below volume threshold)
     c. If temporal required: filter to engines with VersionedStorage, OR mark replay fallback
     d. Pick the cheapest surviving engine (lowest complexity × volume factor)

  3. VALIDATE:
     a. Every query has an assigned engine (or a documented fallback)
     b. Write amplification within budget (sum of projections per event)
     c. No query requires cross-REMOTE-engine read (denormalize if needed)

  4. WARN:
     a. Degraded patterns (O(N) where O(logN) is possible with a better engine)
     b. Temporal queries falling back to replay (O(events) instead of O(1))
     c. High write amplification (>3 projections per event)
```

### Temporal-Aware Planning

When a query input struct has an `AsOf *time.Time` field (mirroring how `*Cursor` signals
pagination today), the planner knows temporal reads are needed:

```go
type AccountBalanceQuery struct {
    ID   AccountID
    AsOf *time.Time  // nil = latest, non-nil = point-in-time read
}
```

The planner then:

1. Checks if any available engine implements `VersionedStorage`
2. If yes → as-of reads are O(1), assign normally
3. If no → as-of reads require replay (O(events to T)), emit degradation warning:
   `"query AccountBalanceQuery needs as-of reads but no versioned engine is available;
   falling back to event-log replay (cost: O(events to T))"`

---

## 10. The Missing Three: Vector, Search, Spatial

These three data models have **no ADT, no backend interface, no fold return type** today. They
are the deepest gap. Adding them requires type-system design (data-model-first), not just driver
work.

### Vector (Similarity Search)

```go
// Embedding is the fold return type for vector projections.
type Embedding struct {
    ID     any       // the entity ID
    Vector []float32 // the embedding (e.g., 1536-dim from OpenAI)
}

// metaengine.On(DocumentIndexed{}, func(e DocumentIndexed) metaengine.Embedding {
//     return metaengine.Embedding{ID: e.ID, Vector: e.Embedding}
// })

// VectorBackend is the data-model interface for k-NN search.
type VectorBackend interface {
    VectorAdd(ctx context.Context, collection string, id any, vec []float32) error
    VectorSearch(ctx context.Context, collection string, query []float32, k int) ([]VectorHit, error)
    VectorDelete(ctx context.Context, collection string, id any) error
}

type VectorHit struct {
    ID    any
    Score float64 // cosine similarity or L2 distance
}

// Query declaration:
// findSimilar := metaengine.Query[FindSimilar, VectorHit]("similar_docs",
//     metaengine.On(DocumentIndexed{}, func(e DocumentIndexed) metaengine.Embedding { ... }),
// )
// // The planner sees Embedding return type → Vector ADT → needs HNSW/IVF storage engine
```

**Storage engines:** HNSW (pgvector, Qdrant, Weaviate), IVF (Milvus), or brute-force scan
(In-Memory, for small sets). The cost matrix cell `Vector × HNSW = O(logN) approximate`.

### Search (Full-Text)

```go
// IndexedText is the fold return type for full-text search projections.
type IndexedText struct {
    ID     any
    Fields map[string]string // field name → text to index (title, body, etc.)
}

// SearchBackend is the data-model interface for inverted-index search.
type SearchBackend interface {
    SearchIndex(ctx context.Context, collection string, docID any, fields map[string]string) error
    SearchQuery(ctx context.Context, collection string, query string, limit int) ([]SearchHit, error)
}

type SearchHit struct {
    ID    any
    Score float64 // BM25 or TF-IDF relevance
}
```

**Storage engines:** Inverted index (Elasticsearch, OpenSearch, Meilisearch, Bleve). The cost
matrix cell `Search × Inverted = O(1)` for term lookup, plus O(k log k) for ranking.

### Spatial (Geo Range)

```go
// Geometry is the fold return type for spatial projections.
type Geometry struct {
    ID   any
        Type  SpatialType // Point, Polygon, etc.
    Data []float64     // [lng, lat, lng, lat, ...] or GeoJSON
}

// SpatialBackend is the data-model interface for geo range queries.
type SpatialBackend interface {
    SpatialIndex(ctx context.Context, collection string, id any, geom Geometry) error
    SpatialWithin(ctx context.Context, collection string, center [2]float64, radiusMeters float64) ([]SpatialHit, error)
}
```

**Storage engines:** R-Tree (PostGIS, SQLite RTree module), Geohash (Redis GEO, Elasticsearch
geo). The cost matrix cell `Spatial × R-Tree = O(logN)`.

### Why These Are Type-System Decisions

Each of the three requires:

1. **A new fold return type** (`Embedding`, `IndexedText`, `Geometry`) — the developer's
   declaration mechanism
2. **A new backend interface** (`VectorBackend`, `SearchBackend`, `SpatialBackend`) — the
   engine contract
3. **A new ADT constant** (`ADTVector`, `ADTSearch`, `ADTSpatial`) — for the planner's
   classification
4. **New cost matrix entries** — the (ADT × StorageEngine) cells

This is data-model-first work. Until the types exist, no driver can plug in. This is the
strategic priority — it unblocks an entire category of use cases (semantic search, RAG,
geospatial) that are currently impossible to express.

---

## 11. Migration Path: From Bundled Engine to Separated Layers

### Today's Interface (Keep the ISP Split)

The current per-ADT interface split (`MapBackend`, `ScanBackend`, `SetBackend`,
`CounterBackend`, `GraphBackend`, `MultimapBackend`, `LogBackend`) is already correct — it is
the data-model layer. **Do not change it.** The optional capability interfaces
(`PushdownScan`, `RawValueReader`, `StreamingScan`, `LayoutPlanner`) are also correct — they are
storage-engine capabilities.

### What Changes: `EngineProfile` Gains Storage-Engine Metadata

The only structural change is enriching `EngineProfile` to name the storage engine explicitly,
rather than encoding it opaquely in `NsPerOp`:

```go
// PROPOSED — EngineProfile names the storage engine, not just the cost
type EngineProfile struct {
    Name     string
    Layout   StorageLayout       // NEW: B-Tree, LSM, Hash, Columnar, HNSW, Inverted, R-Tree
    Supports map[ADT]Complexity   // unchanged: which data models this engine serves

    NsPerOp    float64            // unchanged: calibrated cost (still useful for tie-breaking)
    NsPerRead  float64            // unchanged
    NsPerWrite float64            // unchanged
}

type StorageLayout string

const (
    LayoutBTree       StorageLayout = "btree"
    LayoutLSM         StorageLayout = "lsm"
    LayoutHash        StorageLayout = "hash"
    LayoutColumnar    StorageLayout = "columnar"
    LayoutAppendLog   StorageLayout = "append_log"
    LayoutHNSW        StorageLayout = "hnsw"        // for Vector engines
    LayoutInverted    StorageLayout = "inverted"     // for Search engines
    LayoutRTree       StorageLayout = "rtree"        // for Spatial engines
    LayoutInMemory    StorageLayout = "in_memory"
)
```

This is **backward compatible**: existing engines set `Layout` to the appropriate constant and
keep everything else unchanged. The planner gains the ability to reason about storage engines
without breaking any existing code.

### The Planner Reads the Matrix

```go
// The planner consults a static cost matrix keyed by (ADT, StorageLayout).
// This replaces opaque per-engine NsPerOp comparisons for structural decisions.
var costMatrix = map[ADT]map[StorageLayout]Complexity{
    ADTMap: {
        LayoutBTree:    ComplexityOLogN,
        LayoutLSM:      ComplexityOLogN,
        LayoutHash:     ComplexityO1,
        LayoutColumnar: ComplexityON, // suboptimal — planner warns
        LayoutInMemory: ComplexityO1,
    },
    ADTCounter: {
        LayoutBTree:    ComplexityON,
        LayoutLSM:      ComplexityON,
        LayoutColumnar: ComplexityO1, // optimal — columnar SUM is native
        LayoutInMemory: ComplexityON,
    },
    ADTVector: {
        LayoutHNSW:     ComplexityOLogN, // approximate k-NN
        LayoutInMemory: ComplexityON,    // brute force
    },
    // ... full matrix from §8
}

// The planner uses the matrix to pick engines:
func pickEngine(adt ADT, engines []Engine) (Engine, Complexity) {
    best := engines[0]
    bestCost := costMatrix[adt][best.Profile().Layout]
    for _, e := range engines[1:] {
        c := costMatrix[adt][e.Profile().Layout]
        if rankComplexity(c) < rankComplexity(bestCost) {
            best, bestCost = e, c
        }
    }
    return best, bestCost
}
```

### Migration Steps

1. **Add `StorageLayout` to `EngineProfile`** — set `Layout` on Memory, SQLite, Pebble engines.
   No behavior change; the planner still uses `NsPerOp` for tie-breaking.
2. **Add the static cost matrix** — the (ADT × StorageLayout) table from §8. The planner uses
   it for structural classification (which engine family fits), then `NsPerOp` for fine-grained
   tie-breaking within the same family.
3. **Add `VersionedStorage` interface** — optional capability, checked at runtime. No existing
   engine implements it initially; it's ready for BigTable/Cassandra drivers.
4. **Add new ADTs** (`ADTVector`, `ADTSearch`, `ADTSpatial`) + backend interfaces + fold return
   types. These are additive — they don't affect existing queries.
5. **Add the temporal query signal** (`AsOf *time.Time` in query input structs) — the planner
   checks for it and routes to `VersionedStorage` or replay fallback.

Each step is independently shippable. Steps 1–2 are a refactor with no behavior change. Steps
3–5 are additive features.

---

## 12. What Is Hard (Honest Assessment)

### The Combination Explosion

With 9 data models × 8 storage engines × 2 temporal modes × 3 retention policies, the
theoretical space is 432 combinations. In practice:

- Most engines only support 1–3 data models (Neo4j doesn't do Vector)
- Most deployments have 1–5 engines
- The matrix has many empty cells (wrong engine for that ADT)

The real decision space is small (tens of valid combinations), but the **matrix must be
correct** — a wrong cell produces wrong planner decisions. Calibration and testing matter.

### Calibration Drift

The static cost matrix is structural (Big-O classes). The `NsPerOp` numbers are calibrated
(benchmarks). Calibrated numbers drift with hardware, data shape, and engine versions. The
matrix is stable; the calibration is not. Mitigation: the `calibration_bench_test.go` pattern
already in `metaengine/pebbleengine/` should extend to every engine.

### When Separation Hurts

For the common case (single SQLite engine, small scale), the axis separation adds cognitive
overhead without benefit. The planner picks SQLite for everything because it's the only engine.
The separation pays off only when:

- Multiple engines are available (2+)
- Specialized query patterns exist (Vector, Search, Spatial)
- Scale pushes past single-engine comfort (high write rate, large volumes)

For single-engine deployments, the layers are transparent — the planner collapses them into one
assignment. The complexity is paid only when the value is realized.

### The Temporal Replay Fallback Is Expensive

When no versioned engine is available and a query needs as-of reads, the fallback is
event-log replay — O(events to T). For a stream with 100K events queried for state at event 90K,
that's 90K folds per query. The planner must warn loudly, and the operator must either provide
a versioned engine or accept the cost.

---

## 13. Relationship to Existing Canon

This document is the **unifying lens**. It does not replace the existing docs — it connects
them:

| Document | Role | This doc's relationship |
| --- | --- | --- |
| [database-architecture-taxonomy.md](../research/database-architecture-taxonomy.md) | Reference: 13 interface models × 12 storage engines | Provides the raw material. This doc identifies which models map to existing ADTs and which need new ones. |
| [meta-engine-project-definition.md](meta-engine-project-definition.md) | Vision: why cross-engine view selection is novel | Provides the research framing. This doc provides the architectural decomposition that makes it implementable. |
| [meta-engine-design.md](meta-engine-design.md) | Technical design: cost profiles, optimizer, deployments | Provides the cost model and optimizer algorithm. This doc refines `EngineProfile` to separate the two axes the design doc conflates. |
| [meta-engine-assumptions-and-query-planning.md](meta-engine-assumptions-and-query-planning.md) | Scale thresholds: N → data structure | Provides the volume thresholds. This doc's cost matrix is the structural layer those thresholds operate on. |
| [DataFusion lessons](2026-07-31_datafusion-lessons-for-metaengine.md) | Engineering patterns: rule pipeline, logical/physical split | Provides the logical/physical plan separation and tier classification. This doc adds the storage-engine axis and the temporal-storage capability that the DataFusion doc treats only at the logical layer. |

### The Single Sentence

**Today's `metaengine.Engine` bundles "what operations I support" (data model) with "how I store
bytes" (storage engine) in one type. Separating them — plus adding temporality as a storage
capability with a retention knob — turns the planner from "pick the cheapest opaque engine"
into "compose the optimal backend from independent layers," and unblocks Vector, Search, and
Spatial as first-class query patterns instead of impossible-to-express gaps.**
