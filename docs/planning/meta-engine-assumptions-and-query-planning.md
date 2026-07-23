# Meta-Engine: Assumptions, Data Model & Query Planning

> **The question:** How do we design the type system and query planner so that event-sourced
> data is stored as efficiently as possible — on whatever combination of engines the operator
> chooses — WITHOUT doing stupid things like denormalizing every filter combination when a SQL
> index would suffice?

**Status:** Design (2026-07-23)
**Prerequisite reading:** [meta-engine-design.md](meta-engine-design.md) (the vision)

---

## Table of Contents

1. [Required Assumptions](#1-required-assumptions)
2. [The Two-Level Optimization Model](#2-the-two-level-optimization-model)
3. [The Data Model: How Developers Declare Intent](#3-the-data-model-how-developers-declare-intent)
4. [Scale-Dependent Data Structure Selection](#4-scale-dependent-data-structure-selection)
5. [The Complete Query Pattern Catalog](#5-the-complete-query-pattern-catalog)
6. [The "Don't Be Stupid" Rules](#6-the-dont-be-stupid-rules)
7. [How the Planner Actually Works](#7-how-the-planner-actually-works)
8. [What Already Exists (Don't Rebuild)](#8-what-already-exists-dont-rebuild)
9. [Open Problems](#9-open-problems)

---

## 1. Required Assumptions

The meta-engine cannot function without these assumptions. Each one must be either guaranteed
by the architecture or declared by the developer/operator.

### Assumption 1: Events Are Immutable and Ordered

**Status: Already guaranteed by event sourcing.**

Events are append-only, numbered by version, never mutated. This means:

- Projections can be rebuilt from scratch by replaying events
- Multiple projections can read the same event stream independently
- No distributed transactions needed across engines (each projection is independent)
- The event log is the single source of truth — projections are disposable caches

### Assumption 2: Query Patterns Are Declared, Not Discovered

**Status: Must be enforced by the API.**

The developer declares which query operations they need at declaration time:

```go
projection.Declare[UserView, UserID]("users",
    handlers...,
    projection.PointLookup(),
    projection.EqualityFilter("status"),
    projection.OrderBy("joined_at"),
)
```

The planner uses these declarations to plan the physical layout. It does NOT analyze queries at
runtime. This is a **compile-time/startup-time planner**, not a runtime query optimizer.

**Why this assumption is necessary:** If query patterns were discovered at runtime, the planner
would need to retroactively add indexes/projections — which means blocking queries while
rebuilding structures. Declared patterns let us plan upfront.

### Assumption 3: Cardinality Is Declared or Estimable

**Status: Must be provided by developer. Default: "unknown" → conservative.**

The optimal data structure depends on N. Without cardinality, the planner cannot distinguish
between "sorted slice is optimal" (N < 10K) and "B-tree is necessary" (N > 100K).

```go
projection.Declare[UserView, UserID]("users",
    handlers...,
    projection.PointLookup(),
    projection.EqualityFilter("status"),
    projection.Volume(1_000_000),  // expect ~1M users
)
```

**When cardinality is not declared:** The planner assumes "grows unbounded" and picks the
structure that scales (B-tree over sorted slice, Bloom filter over hash set for membership at
scale). This is conservative — it may over-engineer for small datasets, but it won't break at
scale.

**Alternative: adaptive re-planning.** The planner could measure actual cardinality at runtime
and re-plan on restart. This is a future enhancement, not a day-one requirement.

### Assumption 4: Latency Budget Is Declared per Query Pattern

**Status: Optional but recommended.**

```go
projection.PointLookup(projection.WithLatencyBudget(1 * time.Millisecond)),
projection.EqualityFilter("status", projection.WithLatencyBudget(50 * time.Millisecond)),
```

When declared, the planner can reject plans that can't meet the budget:

- "Filter on 'status' with 1ms budget requires a B-tree index. Only KV (memory) available.
  O(N) scan exceeds budget. Add SQLite or reduce budget."

When not declared, the planner optimizes for lowest cost without a hard constraint.

### Assumption 5: Engines Declare Their Cost Profiles

**Status: Must be implemented per engine plugin.**

Each engine plugin declares what structures it provides and at what complexity:

```go
func (sqliteEngine) Profile() metaengine.EngineProfile {
    return metaengine.EngineProfile{
        ADTs: map[metaengine.ADT]metaengine.ADTOps{
            metaengine.ADTMap: {
                Lookup:     metaengine.CLogN,    // B-tree PK index
                NativeIndex: true,                // has indexes — no need for separate projection
                PersistMode: metaengine.PersistDisk,
            },
            metaengine.ADTSortedMap: {
                Lookup:      metaengine.CLogN,   // B-tree index
                RangeFilter: metaengine.CLogN,   // index range scan
                EqualityFilter: metaengine.CLogN, // index seek
                Sort:         metaengine.CLogN,  // index order
                NativeIndex:  true,               // one table + N indexes serves all patterns
                PersistMode:  metaengine.PersistDisk,
            },
            metaengine.ADTCounter: {
                Count:    metaengine.CN,         // COUNT(*) — full scan without rollup
                Rollup:   metaengine.C1,         // pre-materialized counter via Increment
                PersistMode: metaengine.PersistDisk,
            },
            metaengine.ADTGraph: {
                Traverse: metaengine.CN,         // WITH RECURSIVE CTE — slow
                PersistMode: metaengine.PersistDisk,
            },
            metaengine.ADTLog: {
                Append: metaengine.C1,
                Read:   metaengine.CLogN,
                PersistMode: metaengine.PersistDisk,
            },
        },
    }
}
```

The critical field is `NativeIndex: true`. This tells the planner: **do NOT create separate
projections for different query patterns within this ADT. One table + N indexes suffices.**

### Assumption 6: One Projection Instance Per Engine Per ADT

**Status: Architecture decision.**

For each declared projection, the planner assigns each ADT to at most ONE engine. If the
projection needs both Map (point lookup) and SortedMap (filter), and the deployer provides
both SQLite and Pebble:

- The planner picks the best engine for EACH ADT independently
- It does NOT replicate the same ADT across multiple engines (that's write amplification
  for no read benefit)

**Exception:** auto-denormalization for cross-engine query avoidance (see section 2).

---

## 2. The Two-Level Optimization Model

This is the answer to "don't denormalize everything when SQL indexes suffice."

The optimizer operates at two distinct levels. Conflating them is the source of the
over-optimization problem.

### Level 1: Engine Assignment (Across Engines)

**Question:** Which engine serves which ADT for this projection?

```
Projection: users
  PointLookup()        → Map ADT       → Pebble (O(1) hash)
  EqualityFilter(status) → SortedMap ADT → SQLite (O(log N) B-tree)
  OrderBy(joined_at)    → SortedMap ADT → SQLite (same table, different index)
  Count(status)         → Counter ADT    → SQLite rollup table
```

This level decides: Pebble gets the hash index, SQLite gets the table + indexes + rollup.
Each ADT goes to the engine that serves it cheapest. **This is where multi-engine distribution
happens.**

### Level 2: Within-Engine Layout (Inside One Engine)

**Question:** Within the assigned engine, what physical structures serve the query patterns?

For SQLite (NativeIndex = true):

```
One table: users (id, email, status, country, joined_at, ...)
Indexes:  idx_status (status), idx_country (country), idx_joined (joined_at)
Rollup:   user_count_by_status (status, count)
```

The optimizer creates ONE table and adds indexes. It does NOT create separate tables for each
filter column. It does NOT create separate projections for each sort order. **SQL indexes ARE
the optimization.**

For Pebble (NativeIndex = false for SortedMap):

```
Keyspace 1: users_by_id (key=UserID → value=UserView JSON)     [Map ADT]
Keyspace 2: users_by_email (key=email → value=UserID)           [Map ADT, secondary]
```

Pebble has no secondary indexes within a keyspace. So if Pebble is assigned SortedMap, the
optimizer must either:

- Create a separate keyspace per filter column (write amplification)
- OR recognize that Pebble's LSM IS sorted and can do range scans on the primary key
- OR recommend SQLite for the SortedMap ADT instead

### The Anti-Pattern (What NOT To Do)

```
❌ STUPID: Creating a separate projection/table per (filter, sort) combination

users_by_status_active_ordered_by_joined    → materialized view
users_by_status_active_ordered_by_country   → materialized view
users_by_status_suspended_ordered_by_joined → materialized view
users_by_country_DK_ordered_by_joined       → materialized view
... (15 more combinations)

This is what a naive optimizer would do. It's catastrophic for writes.
```

```
✅ SMART: One table, multiple indexes — let the engine's query planner handle it

users table:
  - PK index on id
  - Index on status
  - Index on country
  - Index on joined_at

The SQL engine's own query planner picks which index to use per query.
WHERE status='active' AND country='DK' uses idx_status (or idx_country,
whichever is more selective — SQL's planner decides, not us).
```

**The rule:** Our optimizer plans the PHYSICAL LAYOUT (tables, indexes, keyspaces). The
engine's INTERNAL query planner handles per-query execution. We do not re-implement query
planning — we set up the structures and let the engine optimize within them.

---

## 3. The Data Model: How Developers Declare Intent

The developer's view type is pure data. Query patterns are declared separately. This
separation is critical: the data shape and the access pattern are independent dimensions.

### View Types: Pure Data, No Storage Metadata

```go
// The view is a plain Go struct. No struct tags for indexes.
// No "column" annotations. No storage concerns.
type UserView struct {
    ID       UserID
    Email    string
    Name     string
    Status   string
    Country  string
    JoinedAt time.Time
    Bio      string  // stored but never queried by index
}
```

### Query Declarations: Explicit Access Patterns

```go
users := projection.Declare[UserView, UserID]("users",
    // Event → view handlers (how events fold into the projection)
    projection.On(UserCreated{}, func(e UserCreated) (UserID, UserView) {
        return e.ID, UserView{...}
    }),
    projection.On(UserSuspended{}, func(e UserSuspended, v UserView) UserView {
        v.Status = "suspended"
        return v
    }),
    projection.On(UserDeleted{}, projection.Tombstone[UserView]()),

    // Query pattern declarations (what the optimizer plans for)
    projection.PointLookup(),                    // Get(userID) → *UserView
    projection.UniqueLookup("email"),            // GetByEmail(email) → *UserView
    projection.EqualityFilter("status"),         // Where(status="active") → []*UserView
    projection.EqualityFilter("country"),        // Where(country="DK") → []*UserView
    projection.RangeFilter("joined_at"),         // Where(joinedAt > x) → []*UserView
    projection.OrderBy("joined_at"),             // Recent(10) → []*UserView
    projection.Count("status"),                  // CountByStatus() → map[string]int64

    // Optional: cardinality hint for scale-aware planning
    projection.Volume(1_000_000),               // expect ~1M users
)
```

### Why Separation Matters

If query patterns were embedded in struct tags (like `filter:"status"`), then:

- The view type would change every time a query pattern is added/removed
- Different consumers of the same view type might need different query patterns
- The type would carry storage concerns (violating domain purity)

By separating declaration from data:

- The view type is stable, pure data
- Query patterns are composable and independent
- The optimizer sees ALL patterns at once and can plan holistically (e.g., "status and country
  both need indexes on the same table — one table, two indexes, not two projections")

---

## 4. Scale-Dependent Data Structure Selection

**This is the core insight the user pushed on.** The optimal data structure is a function of
N (cardinality). The same query pattern has a different optimal structure at different scales.

### The Scale Thresholds

For each query pattern, there are scale thresholds where the optimal structure changes:

```
POINT LOOKUP (Get by key):
  N < 100      → sorted slice + binary search     [O(log N), cache-optimal, zero overhead]
  N < 10K      → hash map                          [O(1), in-process]
  N < 100M     → B-tree index on disk              [O(log N), persistent]
  N > 100M     → distributed hash partition        [O(1), network hop]

MEMBERSHIP (Does X exist?):
  N < 100K     → hash set                          [O(1), exact]
  N < 10M      → Bloom filter                      [O(k), probabilistic, 10x less memory]
  N > 10M      → partitioned Bloom filter          [O(k), disk-backed]

EQUALITY FILTER (WHERE col = val):
  N < 1K       → full scan                         [O(N), faster than index for tiny N]
  N < 1M       → B-tree index on column            [O(log N + results)]
  N < 100M     → bitmap index (if low cardinality) [O(1) per value]
                 OR B-tree (if high cardinality)    [O(log N + results)]
  N > 100M     → columnar scan                     [O(N/k), k=columns]

RANGE FILTER (WHERE col > val):
  N < 1K       → full scan                         [O(N)]
  N < 100M     → B-tree range scan                 [O(log N + results)]
  N > 100M     → LSM range scan (Pebble)           [O(log N + results)]

SORT (ORDER BY col):
  N < 10K      → in-memory sort on demand          [O(N log N)]
  N < 100M     → B-tree ordered scan               [O(log N + results), index-ordered]
  N > 100M     → LSM ordered scan or columnar      [O(log N + results)]

COUNT (COUNT WHERE col = val):
  N < 100K     → scan + count                      [O(N)]
  N < 10M      → index scan + count                [O(results)]
  N < 100M     → pre-materialized counter          [O(1), maintained via Increment]
  N > 100M     → approximate counter (HyperLogLog) [O(1), ~2% error]

GRAPH TRAVERSAL (N-hop neighbors):
  N < 10K      → in-memory adjacency list          [O(degree^depth)]
  N < 1M       → SQL recursive CTE                 [O(N), works but slow]
  N > 1M       → native graph DB                   [O(degree^depth)]

FULL-TEXT SEARCH (WHERE col LIKE '%term%'):
  N < 1K       → substring scan                    [O(N)]
  N < 100K     → trigram index                     [O(|trigrams|)]
  N > 100K     → inverted index                    [O(|term_docs|)]

PREFIX MATCH (WHERE col LIKE 'alice%'):
  N < 10K      → sorted scan                       [O(log N + results)]
  N > 10K      → trie / B-tree range scan          [O(|prefix| + results)]
```

### How the Planner Uses Scale

The planner combines the declared `Volume(N)` hint with the available engines:

```
Developer declares: PointLookup() + Volume(1_000_000)

Planner evaluates candidate engines:
  Pebble (hash index):   O(1)  at N=1M  → cost: 0.001ms  → OPTIMAL
  SQLite (PK B-tree):    O(logN) at N=1M → cost: 0.01ms   → acceptable
  Memory (hash map):     O(1)  at N=1M  → cost: 0.001ms   → optimal BUT needs ~500MB RAM

  Decision: Pebble (O(1) + persistent + no RAM constraint)
```

```
Developer declares: PointLookup() + Volume(500)

Planner evaluates:
  Pebble: O(1) at N=500  → cost: 0.001ms  → overkill (Pebble startup cost > query savings)
  SQLite: O(logN) at N=500 → cost: 0.01ms → fine, and already used for other patterns
  Memory: O(1) at N=500  → cost: 0.0001ms → optimal AND cheap

  Decision: Memory (sorted slice would also be fine at this scale)
  Note: Don't add Pebble just for 500 records — it's not worth the operational overhead
```

### Without Cardinality: Conservative Defaults

When `Volume()` is not declared, the planner assumes unbounded growth and picks the
structure that scales indefinitely:

| Pattern        | Conservative default (no volume hint)  |
| -------------- | -------------------------------------- |
| PointLookup    | B-tree or hash index (persistent)      |
| EqualityFilter | B-tree index                           |
| RangeFilter    | B-tree index                           |
| OrderBy        | B-tree index                           |
| Count          | Pre-materialized counter               |
| Membership     | Bloom filter                           |
| Traverse       | Graph engine (or error if unavailable) |

This means small datasets may be over-engineered. The warning:

```
ℹ PointLookup on 'users': no Volume hint declared. Using B-tree (conservative).
  If N < 10K, a hash map would be simpler and faster. Add projection.Volume(N) to optimize.
```

---

## 5. The Complete Query Pattern Catalog

Every query pattern maps to an ADT, which maps to data structures, which are optimal at
specific scales. This is the planner's lookup table.

### Membership Queries ("Does X exist?")

```go
projection.Exists("email")  // Does a user with this email exist?
```

| Scale    | Optimal structure        | Engine                | Complexity         |
| -------- | ------------------------ | --------------------- | ------------------ |
| N < 100K | Hash set                 | Memory, Pebble, Redis | O(1) exact         |
| N < 10M  | Bloom filter             | Memory, Pebble        | O(k) probabilistic |
| N > 10M  | Partitioned Bloom filter | Scylla, custom        | O(k) probabilistic |

**Bloom filter tradeoffs:**

- 10x less memory than a hash set for the same data
- False positives (typically 1%): "might exist" vs "definitely not"
- No false negatives: if it says "no," it's definitely no
- Cannot delete (use Cuckoo filter if deletion needed)

**When the planner picks Bloom filter:** N > 100K AND the query is "does NOT exist" (the
no-false-negative property is valuable) AND approximate is acceptable.

**When the planner picks hash set:** N < 100K OR exact answers required OR deletions needed.

### Point Lookup ("Get entity by key")

```go
projection.PointLookup()  // Get(userID) → *UserView
```

| Scale    | Optimal structure | Engine           | Complexity             |
| -------- | ----------------- | ---------------- | ---------------------- |
| N < 100  | Sorted slice      | Memory           | O(log N) cache-optimal |
| N < 10K  | Hash map          | Memory           | O(1)                   |
| N < 100M | B-tree PK index   | SQLite, Postgres | O(log N) persistent    |
| N < 1B   | Hash index        | Pebble, Redis    | O(1) persistent        |
| N > 1B   | Distributed hash  | Scylla, DynamoDB | O(1) network           |

### Unique Secondary Lookup ("Get by email")

```go
projection.UniqueLookup("email")  // GetByEmail("a@b.com") → *UserView
```

This requires a separate index mapping email → UserID, then a point lookup on UserID.

| Scale    | Optimal structure                  | Notes                            |
| -------- | ---------------------------------- | -------------------------------- |
| N < 10K  | Inverted hash map (email → UserID) | Memory, 2 lookups: email→ID→User |
| N < 100M | B-tree unique index                | SQLite: `CREATE UNIQUE INDEX`    |
| N > 100M | Hash index (separate keyspace)     | Pebble: key=email, value=UserID  |

### Equality Filter ("WHERE status = 'active'")

```go
projection.EqualityFilter("status")  // Where(status="active") → []*UserView
```

| Scale    | Cardinality of column      | Optimal structure | Complexity                          |
| -------- | -------------------------- | ----------------- | ----------------------------------- |
| N < 1K   | any                        | Full scan         | O(N) — faster than index for tiny N |
| N < 1M   | low (< 20 distinct values) | Bitmap index      | O(1) per value, popcount            |
| N < 1M   | high                       | B-tree index      | O(log N + results)                  |
| N < 100M | low                        | Bitmap index      | O(1) per value                      |
| N < 100M | high                       | B-tree index      | O(log N + results)                  |
| N > 100M | any                        | Columnar scan     | O(N/k), k = columns                 |

**Bitmap vs B-tree for low-cardinality columns:**

- Status has ~5 values (active, suspended, deleted, pending, banned)
- A bitmap index stores 1 bit per record per value: 5 bitmaps of N bits each
- Query "status=active" → popcount the active bitmap → O(N/64) CPU (SIMD)
- B-tree index → O(log N + results) disk I/O
- At N=1M: bitmap uses 625KB (5 × 1M bits), B-tree uses ~40MB. Bitmap is 64x smaller.
- **The planner should detect low-cardinality columns and prefer bitmap when available.**

### Range Filter ("WHERE created_at > X")

```go
projection.RangeFilter("joined_at")  // Where(joinedAt > X) → []*UserView
```

| Scale    | Optimal structure       | Complexity         |
| -------- | ----------------------- | ------------------ |
| N < 1K   | Full scan               | O(N)               |
| N < 100M | B-tree range scan       | O(log N + results) |
| N > 100M | LSM range scan (Pebble) | O(log N + results) |

### Sort ("ORDER BY joined_at DESC")

```go
projection.OrderBy("joined_at")  // Recent(10) → []*UserView ordered by joined_at DESC
```

| Scale    | Optimal structure            | Complexity                               |
| -------- | ---------------------------- | ---------------------------------------- |
| N < 10K  | In-memory sort on demand     | O(N log N) per query                     |
| N < 100M | B-tree ordered scan          | O(log N + results), index already sorted |
| N > 100M | LSM ordered scan or columnar | O(log N + results)                       |

**Key insight:** If OrderBy and RangeFilter use the same column, they share the same index.
The planner should detect this and avoid creating redundant indexes.

### Count ("COUNT WHERE status = 'active'")

```go
projection.Count("status")  // CountByStatus() → map[string]int64
```

| Scale    | Optimal structure         | Complexity      | Tradeoff                             |
| -------- | ------------------------- | --------------- | ------------------------------------ |
| N < 100K | Scan + count              | O(N)            | No extra storage, accurate           |
| N < 10M  | Index scan + count        | O(results)      | Uses existing index                  |
| N < 100M | Pre-materialized counter  | O(1)            | Extra write per event (+1 Increment) |
| N > 100M | HyperLogLog (approximate) | O(1), ~2% error | Tiny memory, approximate             |

**When the planner picks pre-materialized counter:** N > 100K AND Count is declared as a
query pattern (not just a one-off). The counter is maintained via `sink.Increment()` on each
relevant event.

**When the planner picks scan:** N < 100K OR count is rarely queried. A SQL `COUNT(*)` with
an index is fast enough. Don't pay write amplification for a rarely-used counter.

### Graph Traversal ("Friends of friends")

```go
projection.Traverse(2)  // Network(userID, depth=2) → []*UserView
```

| Scale   | Optimal structure        | Complexity                               |
| ------- | ------------------------ | ---------------------------------------- |
| N < 10K | In-memory adjacency list | O(degree^depth)                          |
| N < 1M  | SQL recursive CTE        | O(N) — works but slow for deep traversal |
| N > 1M  | Native graph DB (Neo4j)  | O(degree^depth) — native                 |

**No fallback for large scale:** If N > 1M and no graph engine is available, the planner
errors: "Traversal requires a graph engine at this scale. SQL recursive CTE is O(N) which
exceeds latency budget. Add Neo4j or reduce Volume."

### Full-Text Search

```go
projection.Search("bio")  // Search("golang enthusiast") → []*UserView
```

| Scale    | Optimal structure | Complexity       |
| -------- | ----------------- | ---------------- |
| N < 1K   | Substring scan    | O(N × \|bio\|)   |
| N < 100K | Trigram index     | O(\|trigrams\|)  |
| N > 100K | Inverted index    | O(\|term_docs\|) |

### Prefix Match

```go
projection.Prefix("email")  // Prefix("alice") → all users with email starting with "alice"
```

| Scale   | Optimal structure         | Complexity              |
| ------- | ------------------------- | ----------------------- |
| N < 10K | Sorted scan               | O(log N + results)      |
| N > 10K | Trie or B-tree range scan | O(\|prefix\| + results) |

### Approximate Cardinality ("How many distinct countries?")

```go
projection.DistinctCount("country")  // ~How many distinct countries?
```

| Scale    | Optimal structure  | Complexity | Accuracy  |
| -------- | ------------------ | ---------- | --------- |
| N < 100K | Full distinct scan | O(N)       | Exact     |
| N > 100K | HyperLogLog        | O(1)       | ~2% error |

### Top-N Frequency ("Most common statuses")

```go
projection.TopN("status", 5)  // Top 5 most common statuses
```

| Scale    | Optimal structure | Complexity                |
| -------- | ----------------- | ------------------------- |
| N < 100K | Full scan + sort  | O(N log N)                |
| N > 100K | Count-min sketch  | O(1) lookup, overestimate |

---

## 6. The "Don't Be Stupid" Rules

These rules prevent the optimizer from over-optimizing. Each rule has a concrete cost.

### Rule 1: Don't Create Separate Projections When One Table + Indexes Suffices

**Applies when:** An engine with `NativeIndex: true` (SQL) is assigned the SortedMap ADT.

```
✅ CORRECT (SQLite assigned both Filter(status) and OrderBy(joined_at)):
  One table: users
  Indexes: idx_status, idx_joined
  → SQL's planner picks which index to use per query

❌ STUPID:
  Table 1: users_by_status (materialized, sorted by status)
  Table 2: users_by_joined (materialized, sorted by joined_at)
  → Double the storage, double the writes, zero query benefit
```

### Rule 2: Don't Materialize What Can Be Computed Cheaply

**Applies when:** A query pattern can be served by an existing index without additional storage.

```
Developer declares: Count("status") + Volume(50_000)

✅ CORRECT:
  SQLite COUNT(*) with idx_status → O(results) ≈ O(N/5) ≈ 10K rows
  At 50K total users with 5 statuses, average 10K rows per status.
  SQLite counts 10K indexed rows in <1ms. No rollup needed.

❌ STUPID:
  Pre-materialized rollup table: user_count_by_status
  → Extra write on every UserCreated/UserSuspended/UserDeleted event
  → Extra table to migrate, sync, and maintain
  → Saves <1ms per query at the cost of 3x write amplification
```

**Threshold:** Only materialize counters when N > 100K AND the column has been declared with
`Count()`. Below 100K, `COUNT(*)` with an index is fast enough.

### Rule 3: Don't Index a Column That's Never Filtered

**Applies when:** The optimizer is creating indexes.

Every index costs: +1 disk write per insert/update, +storage, +maintenance on schema change.

The optimizer creates indexes ONLY for columns that appear in declared query patterns:

```
UserView fields: ID, Email, Name, Status, Country, JoinedAt, Bio

Declared patterns: PointLookup(), EqualityFilter("status"), OrderBy("joined_at")

✅ Indexes created:
  PK on ID (required for point lookup)
  idx_status (for EqualityFilter)
  idx_joined (for OrderBy)

❌ NOT created:
  No index on Email (not declared as a query pattern — it's just stored data)
  No index on Name (not queried)
  No index on Country (not declared — even though it COULD be filtered)
  No index on Bio (not queried)
```

### Rule 4: Don't Split Across Engines When One Engine Suffices

**Applies when:** Two ADTs could be served by the same engine at acceptable cost.

```
Available engines: SQLite only

Declared patterns: PointLookup() + EqualityFilter("status")

✅ CORRECT:
  Both in SQLite: PK index for lookup, secondary index for filter
  One table, two indexes. Simple.

❌ STUPID:
  PointLookup → Memory hash map (O(1) but volatile)
  EqualityFilter → SQLite index (O(log N))
  → Two engines for what one handles fine. Memory map must be rebuilt on restart.
  → Cross-engine coordination on writes (update both atomically — hard).
```

**The rule:** Only split to a second engine when the first engine's cost for that ADT is
significantly worse AND the volume justifies it.

### Rule 5: Don't Use a Complex Structure When a Simple One Works

**Applies when:** Cardinality is low.

```
PointLookup() + Volume(500)

✅ CORRECT:
  Sorted slice in memory (or even just a SQLite table with PK)
  Binary search: O(log 500) ≈ 9 comparisons. Microseconds.

❌ STUPID:
  Pebble hash index for 500 records
  → Pebble startup (open DB, init LSM) takes 50ms+
  → Each lookup crosses a CGo boundary or LSM seek
  → For 500 records, a map[string]UserView is faster end-to-end
```

### Rule 6: Don't Approximate When Exact Is Affordable

**Applies when:** Approximate structures (Bloom filter, HyperLogLog, Count-min sketch) are
considered.

```
Membership("email") + Volume(10_000)

✅ CORRECT:
  Hash set: 10K emails × ~50 bytes = 500KB. Exact. O(1).
  500KB is nothing. Use the exact structure.

❌ STUPID:
  Bloom filter for 10K emails
  → False positives on email existence checks → user confusion
  → Only saves ~450KB of RAM (irrelevant at this scale)
  → Bloom filter is justified at N > 1M where the hash set would be >500MB
```

### Rule 7: Don't Denormalize for Queries That Don't Cross Engines

**Applies when:** Auto-denormalization is being considered.

```
Declared projections:
  users:    Filter("status")     → SQLite
  friendships: Traverse(2)       → Neo4j

Query: "active users"  → only touches SQLite → no denormalization needed

Query: "friends of user X" → only touches Neo4j → no denormalization needed

Query: "active users with >5 friends" → touches BOTH → denormalize friend_count onto SQLite
```

Denormalization has a cost: extra writes per event. Only denormalize when a declared query
pattern actually needs data from two different engines.

---

## 7. How the Planner Actually Works

### Step-by-Step Planning Algorithm

```
INPUT:
  declarations: []ReadModelSpec    (from developer)
  engines: []Engine                (from operator, each with EngineProfile)

ALGORITHM:

Phase 1: Parse Declarations
  For each ReadModelSpec:
    1. Extract query patterns → group by ADT
    2. Extract volume hint (or default: unbounded)
    3. Extract latency budget (or default: none)

Phase 2: Assign ADTs to Engines
  For each (ReadModelSpec, ADT) pair:
    1. Find all engines that provide this ADT
    2. For each candidate engine, compute cost:
       cost = f(complexity_at_volume, persistence, memory_usage, write_amplification)
    3. Pick the cheapest engine
    4. If cheapest is suboptimal (e.g., O(N) where O(logN) is possible), mark DEGRADED

Phase 3: Within-Engine Layout Planning
  For each (ReadModelSpec, engine) assignment:
    If engine.NativeIndex == true (SQL):
      → Plan ONE table + N indexes (one per filter/sort column)
      → Plan rollup counters ONLY if volume > threshold AND count is declared
      → Deduplicate indexes (same column used by filter + range = one index)
    If engine.NativeIndex == false (KV):
      → Plan one keyspace per unique lookup key
      → Cannot serve SortedMap natively → mark as degraded or assign to SQL instead

Phase 4: Cross-Engine Denormalization
  For each pair of projections (A, B) assigned to different engines:
    If a declared query needs data from both A and B:
      1. Identify the minimal data to copy (e.g., friend_count from B onto A's engine)
      2. Add a denormalization projection: B's events also update A's engine
      3. Record the write amplification cost

Phase 5: Validation & Warnings
  1. Every declared query pattern MUST have an assigned engine (else: hard error)
  2. Check degradation: warn for each suboptimal assignment
  3. Check write amplification: warn if >3 projections per event
  4. Check memory constraints: warn if memory engine assigned to high-volume projection
  5. Check persistence: warn if volatile engine is the only one serving a pattern

OUTPUT:
  - Projection assignments (which engine, which table/keyspace, which indexes)
  - Auto-generated projection handlers (event → engine writes)
  - Denormalization plan (which data is copied where)
  - Startup warnings (degradation, write amplification, memory)
  - Typed read API (wired function fields per query pattern)
```

### Index Deduplication Logic

When the planner creates indexes within a SQL engine, it must deduplicate:

```
Declared patterns on 'users':
  EqualityFilter("status")     → needs index on status
  EqualityFilter("country")    → needs index on country
  RangeFilter("joined_at")     → needs index on joined_at
  OrderBy("joined_at")         → needs index on joined_at (ALREADY HAVE IT from RangeFilter)
  UniqueLookup("email")        → needs UNIQUE index on email

Deduplicated index plan:
  idx_status  (status)             ← from EqualityFilter
  idx_country (country)            ← from EqualityFilter
  idx_joined  (joined_at)          ← shared by RangeFilter + OrderBy
  idx_email   (email) UNIQUE       ← from UniqueLookup

4 indexes, not 5. No redundant idx_joined_created_twice.
```

### Composite Index Detection

When multiple filter columns are frequently queried together:

```
Declared patterns:
  EqualityFilter("status")
  EqualityFilter("country")

If the developer also declares:
  projection.CompositeFilter("status", "country")  // I query WHERE status=? AND country=?

Planner creates:
  idx_status_country (status, country)  ← composite index serves both individual AND combined

Instead of:
  idx_status (status)
  idx_country (country)
  idx_status_country (status, country)  ← redundant with the individual indexes? NO —
                                          composite serves AND queries, individual serves
                                          single-column queries. SQL planner picks.
```

The planner does NOT auto-create composite indexes unless the developer declares
`CompositeFilter`. Individual filter patterns get individual indexes. This is correct: the
SQL engine's planner handles index intersection for AND queries.

---

## 8. What Already Exists (Don't Rebuild)

The codebase already has significant query infrastructure. The meta-engine must BUILD ON
this, not replace it.

| Capability                        | Status    | Location                                      | Meta-engine use                        |
| --------------------------------- | --------- | --------------------------------------------- | -------------------------------------- |
| Structured query descriptor       | **Built** | `kv.ViewQuery`, `kv.Condition`, `kv.Operator` | Use as-is for all queries              |
| WHERE clause builder              | **Built** | `storage/sql/where.go`                        | Use as-is                              |
| ORDER BY (single + multi-col)     | **Built** | `storage/view/query.go`                       | Use as-is                              |
| Keyset cursor pagination          | **Built** | `storage/view/query.go`                       | Use as-is                              |
| COUNT without row loading         | **Built** | `storage/view/count.go`                       | Use as-is                              |
| Struct-tag column inference       | **Built** | `storage/view/auto.go`                        | Extend for index inference             |
| Manual index declaration          | **Built** | `storage/view/store.go` `IndexSpec`           | Replace with auto-planned indexes      |
| Transactional projection sink     | **Built** | `storage/relational/sink.go`                  | Use as-is for generated handlers       |
| Atomic counter increment          | **Built** | `storage/relational/sink.go`                  | Use for rollup counters                |
| Tombstone pushdown                | **Built** | `storage/view/query.go`                       | Use as-is                              |
| ViewStore + capability interfaces | **Built** | `kv/view_store.go`                            | Use as-is — engine impls satisfy these |

**What the meta-engine ADDS (does not replace):**

1. **Query pattern declarations** — `projection.Filter()`, `.Sort()`, `.Count()`, etc.
2. **Cost profiles** — `EngineProfile` per engine plugin
3. **The planner** — assigns ADTs to engines, plans indexes, detects denormalization
4. **Auto-generated projection handlers** — event → engine writes (for the 80% simple case)
5. **Auto-generated typed read API** — `store.Users.Get(id)`, `store.Users.ByStatus("active")`
6. **Degradation detection + warnings** — startup diagnostics
7. **Index auto-creation + deduplication** — from declared query patterns

---

## 9. Open Problems

### Problem 1: How Does the Planner Detect Composite Query Needs?

The developer declares individual patterns: `Filter("status")` + `Filter("country")`. But
the app might query `WHERE status='active' AND country='DK'`.

**Options:**
A. Developer explicitly declares `CompositeFilter("status", "country")` — simple, explicit
B. Planner infers from the existing `ViewQuery` struct (which supports multiple Conditions) —
but the planner runs at startup, before any queries execute
C. Adaptive: planner starts with individual indexes, detects AND queries at runtime, adds
composite index on restart — complex, deferred optimization

**Recommendation:** Option A. The developer knows their query patterns. If they need composite,
they declare it. The planner doesn't guess.

### Problem 2: How Does Auto-Generated Handler Logic Work for Complex Projections?

For a simple Map projection:

```
UserCreated event → Set(userID, UserView{...})     ← trivial to auto-generate
UserSuspended event → Update(userID, {status: "suspended"})  ← also trivial
```

For a relational projection with 5 tables:

```
MessageCreated event →
  Ensure(channels, {id, name})
  Ensure(users, {id, name})
  Upsert(messages, {id, channel_id, author_id, content, created_at})
  Ensure(attachments, {id, message_id, filename})  for each attachment
  Increment(channel_counts, {channel_id}, message_count, +1)
```

This cannot be auto-generated from a simple type declaration. It requires custom handler logic.

**Recommendation:** Two tiers:

- **Tier 1 (auto-generated):** Single-document projections (one event → one record). The
  optimizer generates the handler from `OnCreate/OnUpdate/OnTombstone` declarations.
- **Tier 2 (custom handler):** Multi-table/complex projections. The developer writes the
  handler function, receiving a typed sink. The optimizer still decides WHICH engine and
  WHICH indexes, but the developer controls HOW events map to writes.

### Problem 3: How Does the Planner Handle Schema Changes?

When `UserView` gains a `PhoneNumber` field:

- SQLite: `ALTER TABLE users ADD COLUMN phone TEXT`
- Pebble: schemaless (just start writing the field in new events)
- Neo4j: schemaless
- ClickHouse: `ALTER TABLE` (can be slow on large tables)

**Recommendation:** The planner tracks a schema version per projection per engine. On startup:

1. Compare declared schema to stored schema
2. Auto-migrate where safe (ADD COLUMN for SQL)
3. Warn where manual migration needed (column type change, column removal)

### Problem 4: How Does Volume Estimation Work Without Developer Hints?

If the developer doesn't declare `Volume(N)`, the planner is conservative. But it could also
measure:

- Count events processed for this projection on previous run
- Read the event log size
- Check existing projection row counts (if engine supports `Count`)

**Recommendation:** Start conservative (assume unbounded). Add runtime measurement in a
future iteration. Log: "Based on 847K events processed, actual volume for 'users' is ~847K.
Consider declaring projection.Volume(1_000_000) for more accurate planning."

### Problem 5: Bloom Filter / HyperLogLog Are Not in Any Existing Engine

The planner might decide a Bloom filter is optimal for membership queries at scale. But no
existing engine in the codebase provides a Bloom filter primitive.

**Options:**
A. Build a `bloom` module (Tier 0 primitive, like `kv/` and `dedup/`)
B. Use SQL-based Bloom filter (via a `bloom` extension or manual bitset table)
C. Use an in-memory Bloom filter (rebuilt on startup from the projection)

**Recommendation:** Option A. A `bloom/` module is a leaf primitive (depends on nothing but
stdlib). It implements the membership ADT. The planner assigns it like any other engine. This
is consistent with the existing architecture (leaf primitives like `kv/`, `dedup/`).

Similarly, `hll/` for HyperLogLog and `cms/` for Count-min sketch.

### Problem 6: How Does the Planner Know an Engine Supports Bitmap Indexes?

SQLite doesn't natively support bitmap indexes. Postgres doesn't either (though it has
`bitmap AND/OR` scan strategies internally). ClickHouse does. A custom Memory engine could.

The engine profile must declare not just ADT support but INDEX TYPE support:

```go
metaengine.ADTSortedMap: {
    EqualityFilter: metaengine.CLogN,
    IndexTypes: []metaengine.IndexType{
        metaengine.IndexBTree,      // standard B-tree index
        // metaengine.IndexBitmap,  // NOT supported by this engine
    },
}
```

The planner then picks the best available index type for the column's cardinality:

- Low cardinality + bitmap available → bitmap
- Low cardinality + no bitmap → B-tree (still fine, just larger)
- High cardinality → B-tree (bitmap would be sparse and wasteful)
