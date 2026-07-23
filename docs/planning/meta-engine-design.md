# The Meta-Engine: A Cost-Based Storage Optimizer for Event-Sourced Data

> **The vision:** Given event-sourced data + declared query patterns, automatically distribute
> projections across whatever combination of engines the operator provides — optimizing each
> query pattern to its ideal physical data structure within real hardware constraints.
>
> Run on a single SQLite. Run on SQLite + Pebble. Run on ScyllaDB + ClickHouse + Neo4j. The
> developer's code is identical. The operator's config decides. The optimizer figures out the
> best physical layout.

**Status:** Vision / Design (2026-07-23)
**Author:** Lars + Crush brainstorming session
**Related:** [keep-apps-off-db-layer.md](keep-apps-off-db-layer.md),
[storage-domain-separation.md](storage-domain-separation.md)

---

## Table of Contents

1. [The Core Insight](#1-the-core-insight)
2. [First Principles](#2-first-principles)
3. [The Cost Model](#3-the-cost-model)
4. [How the Optimizer Works](#4-how-the-optimizer-works)
5. [The Developer API](#5-the-developer-api)
6. [The Operator API](#6-the-operator-api)
7. [Concrete Walkthrough](#7-concrete-walkthrough)
8. [The Hard Problems (Honest)](#8-the-hard-problems-honest)
9. [What Makes This Novel](#9-what-makes-this-novel)

---

## 1. The Core Insight

### The Fundamental Axiom

**Every query pattern CAN be served by every storage engine.**

A KV store can do:

- **Point lookup** — O(1) hash lookup. Native.
- **Filtered scan** — O(n) scan all values, decode each, filter in memory. Slow but correct.
- **Sorted range** — O(n log n) scan + sort in memory. Works.
- **Graph traversal** — O(degree^depth) iterative adjacency lookups. Exponential but finite.
- **Count** — O(n) full scan. Correct.
- **Full-text search** — O(n) scan with substring match. Brute but works.

**The question is never "can we?" It's "at what cost, given:**

```
RAM        — can't load 1TB into 50GB server (stream/buffer/paginate)
Disk       — sequential scan vs index lookup (orders of magnitude difference)
Time       — every query has a latency budget (1ms? 100ms? 10s?)
CPU        — in-memory filtering vs index-assisted (100x difference)
Network    — ONLY relevant for remote engines (ClickHouse cluster, Neo4j server).
             Local engines (SQLite file, Pebble dir) = zero network cost.
             Cross-engine reads between local engines = just different syscalls.
```

### What the Meta-Engine Is

The meta-engine is a **cost-based optimizer (CBO)** for storage layout. It's what database
query planners do internally (pick index vs scan vs merge), but operating at a higher level:
**across multiple engines, at deployment time, based on declared query patterns.**

```
Traditional query planner:
  "Given this SQL query + table statistics, pick the best access path
   (index scan vs full scan vs hash join) within ONE database."

Meta-engine:
  "Given these event-sourced projections + declared query patterns + available engines,
   pick the best physical layout (which engine, which data structure, which indexes)
   across ALL engines, within real hardware constraints."
```

### What the Meta-Engine Is NOT

- **NOT an ORM.** ORMs map objects to tables in one DB with direct writes. The meta-engine
  projects events into optimized layouts across N engines. Source of truth is always the
  event log.
- **NOT a federated query engine.** Federated engines (Calcite, Presto) route queries across
  databases at query time. The meta-engine optimizes the physical layout at deployment time
  so queries DON'T need to cross engines.
- **NOT a universal DB interface.** We're not hiding the engines behind one API. We're
  distributing data across engines optimally and exposing engine-specific read paths.
- **NOT a magical silver bullet.** It makes intelligent tradeoffs. Sometimes the optimal
  layout is "everything in SQLite." Sometimes it's "spread across 3 engines." The optimizer
  figures out which.

---

## 2. First Principles

### From Data Structure Theory (Wikipedia: Data Structure)

> "A data structure is the physical implementation of a data type, including specifications
> of the data organization and storage format, as well functions or operations for working
> with this data."

> "There may be multiple concrete data structures for the same ADT, for example a linked list
> or a resizable array for the list ADT."

> "Rob Pike has stated that the choice of data structure almost always has a greater impact
> on efficiency than the choice of algorithm."

The seven ADTs relevant to persistent storage:

| ADT            | Operations                     | Ordered? | Uniqueness?  |
| -------------- | ------------------------------ | -------- | ------------ |
| **Map**        | Get/Set/Delete by key          | no       | keys only    |
| **Sorted Map** | Map ops + Range/Filter/OrderBy | **yes**  | keys only    |
| **Multimap**   | Add(key,val)/GetAll(key)       | no       | no           |
| **Counter**    | Increment/Get                  | no       | n/a          |
| **Set**        | Add/Contains/Members           | no       | **yes**      |
| **Log**        | Append/ReadFrom                | **yes**  | no           |
| **Graph**      | AddEdge/Neighbors/Traverse     | no       | edges unique |

### From Designing Data-Intensive Applications

DDIA's core lessons that apply:

1. **"No one-size-fits-all storage engine."** Different workloads need different data structures.
   OLTP needs B-trees. Analytics needs columnar. Graph needs adjacency. KV needs hash tables.

2. **"The same data can be stored in multiple shapes."** This is the foundation of CQRS and
   materialized views. The event log is the source of truth; projections are derived copies
   optimized for specific read patterns.

3. **"Denormalization is often necessary."** Normalized data requires joins. In a multi-engine
   world, joins across REMOTE engines incur network cost. So we denormalize: project the
   same data into multiple shapes, each optimized for a query pattern.

4. **"Indexes are a form of materialized view."** A B-tree index is a sorted copy of data
   optimized for range queries. A hash index is a copy optimized for point lookup. We're
   extending this concept ACROSS engines, not just within one.

5. **"Write amplification is the cost of read optimization."** Each additional projection
   increases write cost (more places to update per event). The optimizer must balance read
   savings against write overhead.

### The Hardware Constraint Layer

The optimizer respects these physical limits:

| Constraint   | Implication for the optimizer                                                                                                                                                                                                                   |
| ------------ | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| **RAM**      | Large datasets can't fit in memory. The optimizer must prefer streaming/paginated access paths for large projections.                                                                                                                           |
| **Disk I/O** | Sequential scan (O(n) disk) vs index seek (O(log n) disk) is 100x+ difference at scale. Prefer indexed paths.                                                                                                                                   |
| **CPU**      | In-memory decode + filter is CPU-bound. Columnar engines (ClickHouse) avoid this by only reading relevant columns.                                                                                                                              |
| **Time**     | Every query has an implicit latency budget. The optimizer picks the path most likely to meet it.                                                                                                                                                |
| **Network**  | Only relevant when an engine is REMOTE (ClickHouse cluster, Neo4j server, ScyllaDB). Local engines (SQLite file, Pebble dir) have zero network cost — cross-engine reads between local engines are just different syscalls on the same machine. |

---

## 3. The Cost Model

The optimizer needs a way to estimate the cost of serving each query pattern on each candidate
engine. This is the cost model.

### Cost Dimensions

For each (query pattern, engine, data structure) triple:

```
Cost = f(
    time_complexity,      // O(1), O(log n), O(n), O(n log n)
    memory_footprint,     // bytes of RAM needed
    disk_io,              // bytes read from disk
    cpu_work,             // decode + filter + sort cycles
    network_hops,         // 0 for local, 1+ for cross-engine
    write_amplification   // how much extra work per event append
)
```

### Engine Cost Profiles

Each engine plugin declares its cost profile — how well it serves each ADT:

```go
// EngineProfile declares what an engine can do and how well it does it.
type EngineProfile struct {
    Name    string
    Provides map[ADT]Performance
}

type Performance struct {
    Complexity     Complexity  // O(1), O(logN), O(N), O(NLogN)
    Persistent     bool        // survives restart?
    StreamsResults bool        // can stream/paginate large results?
    Columnar       bool        // only reads relevant fields? (ClickHouse)
    MaxDataSize    int64       // practical data size limit (0 = unlimited)
}

type Complexity int
const (
    O1     Complexity = iota  // hash lookup
    OLogN                      // B-tree / LSM seek
    ON                         // full scan
    ONLogN                     // scan + sort
    OExponential               // graph traversal at depth
)
```

### Example Engine Profiles

```
SQLite:
  Map:        O(logN) persistent, streams  ✓
  SortedMap:  O(logN) persistent, streams  ✓
  Counter:    O(N)    persistent (COUNT(*)) — or O(1) via rollup table
  Graph:      O(N)    persistent (WITH RECURSIVE CTE) — slow for deep traversal
  Log:        O(1)    persistent (append)  ✓

Pebble (KV/LSM):
  Map:        O(1)    persistent            ✓
  SortedMap:  O(logN) persistent (LSM is sorted!)
  Counter:    O(N)    persistent (full scan)
  Graph:      O(N^d)  persistent (iterative lookup)
  Log:        O(1)    persistent            ✓

Memory:
  Map:        O(1)    volatile, RAM-bounded
  SortedMap:  O(N)    volatile (scan + sort)
  Counter:    O(N)    volatile (scan count)
  Graph:      O(d)    volatile (adjacency list — fast!)
  Log:        O(1)    volatile

ClickHouse (columnar):
  Map:        O(logN) persistent (sparse index)
  SortedMap:  O(logN) persistent, columnar (only reads needed columns!)
  Counter:    O(1)    persistent (columnar SUM is native)
  Graph:      O(N)    persistent (not designed for this)
  Log:        O(1)    persistent (append, native)

Neo4j (graph):
  Map:        O(1)    persistent (node lookup)
  SortedMap:  O(N)    persistent (not designed for this)
  Counter:    O(N)    persistent (traverse + count)
  Graph:      O(d)    persistent (native adjacency — optimal!)
  Log:        O(1)    persistent (append)
```

### The Optimization Function

The optimizer minimizes total cost:

```
Minimize: Σ (query_pattern_cost × estimated_query_frequency)
          + Σ (write_amplification × estimated_write_frequency)

Subject to:
  - Every declared query pattern MUST be servable
  - Total projection storage ≤ disk budget
  - Hot projections must fit in RAM (for memory engines)
  - No cross-REMOTE-engine queries at read time (denormalize only when it avoids a network hop)
```

In practice, we don't need precise frequency estimates. We use declared query patterns as
binary signals ("this pattern WILL be used") and the cost profiles to rank engines. The
optimizer picks the cheapest engine for each pattern.

---

## 4. How the Optimizer Works

### The Planning Algorithm

```
INPUT:
  - Declared projections (from developer: events → view + query patterns)
  - Available engines (from operator: engine plugins + their cost profiles)

OUTPUT:
  - Projection assignments: which engine serves which query pattern
  - Auto-generated projection handlers: events → engine-specific writes
  - Denormalization plan: which data is projected where (only when remote engines involved)
  - Startup warnings: where degradation occurs

ALGORITHM:
  1. For each declared projection:
     a. Parse query patterns → identify required ADTs
     b. For each ADT, find the cheapest available engine (from cost profiles)
     c. If the cheapest is not the native one, mark as DEGRADED with warning

  2. For cross-pattern dependencies:
     a. If query pattern A needs data that lives in engine X (for pattern B),
        auto-denormalize: project the needed data into engine X too
     b. This increases write amplification but eliminates remote-engine reads

  3. Generate projection handlers:
     a. For Map ADT → generate Set/Get/Delete handler
     b. For SortedMap ADT → generate columnar/table handler with indexes
     c. For Counter ADT → generate increment/decrement handler
     d. For Graph ADT → generate node/edge merge handler

  4. Validate:
     a. Every query pattern has an assigned engine
     b. No query requires cross-REMOTE-engine read (denormalized away in step 2, or served
        via cheap local cross-engine fan-out)
     c. Hardware constraints respected (memory engines don't hold > declared max)

  5. Emit warnings for:
     a. Degraded patterns (O(n) where O(log n) is possible with a better engine)
     b. High write amplification (>3 projections per event)
     c. Volatile engines holding data that should be persistent
```

### Degradation Examples

```
Pattern: Filter(status="active") on 1M users

Available engines: SQLite + Memory
  → SQLite provides SortedMap at O(logN)
  → Assigned to SQLite with index on status
  → Cost: ~1ms per query
  → No degradation

Available engines: Pebble only (KV, no SQL)
  → Pebble provides SortedMap at O(logN) via LSM (it IS sorted!)
  → But has no secondary index — must scan key range + decode + filter
  → Assigned to Pebble, DEGRADED: O(N) scan per query
  → ⚠ WARNING: "Filter on 'status': Pebble has no secondary index.
     Using full scan — O(N) per query, ~500ms for 1M records.
     Add SQLite or Postgres for O(logN) indexed filtering."

Available engines: Memory only
  → Memory provides SortedMap at O(N) (scan + sort)
  → Assigned to Memory, DEGRADED: O(N) per query
  → ⚠ WARNING: "Filter on 'status': only in-memory engine available.
     O(N) scan per query. Suitable for <100K records.
     Add a persistent sorted engine for production."
```

### Auto-Denormalization Example

```
Declared projections:
  users:    Filter("status"), Sort("joined_at")
  friendships: Traverse(depth=2)

Query needed by the app:
  "Active users with >5 friends"
  → needs: users.status (from SQLite) + friend count (from Graph engine)

Optimizer detects cross-REMOTE-engine dependency (if both engines are local, this is
  optional — the cross-engine read is just two syscalls):
  → If both LOCAL (SQLite + Pebble): no denormalization needed. Fan-out read is cheap.
  → If either is REMOTE (SQLite + Neo4j server): auto-denormalize friend_count onto SQLite
  → Generated projection handler for friendships:
     1. MergeEdge into Graph engine (for traversal queries)
     2. Increment friend_count on the user's row in SQLite (for the combined query)
  → Now "active users with >5 friends" is a single SQLite query:
     WHERE status='active' AND friend_count > 5
  → Now "active users with >5 friends" is a single SQLite query:
     WHERE status='active' AND friend_count > 5
  → No cross-REMOTE-engine read at query time. Write amplification: +1 update per friendship event.
```

---

## 5. The Developer API

The developer declares **projections and query patterns.** Never engines. Never data
structures. Never SQL.

### Projections (Read Models)

```go
// Events are declared as pure Go types (already the case today)
type UserCreated   struct { ID UserID; Email, Name, Country string; At time.Time }
type UserSuspended struct { ID UserID; Reason string; At time.Time }
type UserDeleted   struct { ID UserID; At time.Time }
type Friendship    struct { From, To UserID; At time.Time }

// A projection declares: how events fold into a view + what query operations are needed
users := projection.Declare[UserView, UserID]("users",
    projection.On(UserCreated{}, func(e UserCreated) (UserID, UserView) {
        return e.ID, UserView{
            ID: e.ID, Email: e.Email, Name: e.Name,
            Status: "active", Country: e.Country, JoinedAt: e.At,
        }
    }),
    projection.On(UserSuspended{}, func(e UserSuspended, v UserView) UserView {
        v.Status = "suspended"
        return v
    }),
    projection.On(UserDeleted{}, projection.Tombstone[UserView]()),

    // Query patterns — the optimizer maps these to engines
    projection.Lookup(),                    // Get by UserID
    projection.Filter("status", "country"), // WHERE status=? OR country=?
    projection.Sort("joined_at"),           // ORDER BY joined_at
    projection.Count("status", "country"),  // GROUP BY status / country
)

friendships := projection.Graph[Friendship]("friendships",
    projection.On(Friendship{}, func(e Friendship) (UserID, UserID) {
        return e.From, e.To
    }),
    projection.Traverse(2),                 // Friends-of-friends
)
```

### What the Developer Does NOT Write

- No `storage.ViewMapper` with SQL column types
- No `storage.RelationalSchema` with table DDL
- No `graph.GraphDriver` selection
- No `kv.ViewStore` implementation choice
- No `*sql.DB` handling
- No `database/sql` import
- No engine-specific constructor calls

The developer declares **what operations they need on the projection.** The optimizer
figures out the rest.

### The Read API (Generated at Runtime)

The optimizer generates typed read methods based on declared patterns:

```go
store := metaengine.Plan(engines, users, friendships)

// These methods are wired at startup based on the optimization plan:
user, err      := store.Users.Get(ctx, userID)            // → cheapest Map engine
alice, err     := store.Users.GetByEmail(ctx, "a@b.com")  // → unique index
activeUsers    := store.Users.ByStatus(ctx, "active")      // → SortedMap with index
recentUsers    := store.Users.Recent(ctx, 10)              // → SortedMap ORDER BY joined_at
statusCounts   := store.Users.CountByStatus(ctx)           // → Counter rollup
countryCounts  := store.Users.CountByCountry(ctx)          // → Counter rollup
friends        := store.Friendships.Network(ctx, userID, 2) // → Graph traversal
```

Each method dispatches to the engine the optimizer chose. The developer doesn't know or care
which engine serves which query.

---

## 6. The Operator API

The operator provides engines via config. The optimizer inspects them and plans.

### Engine Registration (Plugin Pattern)

Each engine registers itself with its cost profile:

```go
// In each engine plugin's init():
func init() {
    metaengine.Register(sqliteEngine{})
}

type sqliteEngine struct{}

func (sqliteEngine) Name() string { return "sqlite" }

func (sqliteEngine) Profile() metaengine.EngineProfile {
    return metaengine.EngineProfile{
        Provides: map[metaengine.ADT]metaengine.Performance{
            metaengine.ADTMap:       {Complexity: metaengine.OLogN, Persistent: true, StreamsResults: true},
            metaengine.ADTSortedMap: {Complexity: metaengine.OLogN, Persistent: true, StreamsResults: true},
            metaengine.ADTCounter:   {Complexity: metaengine.ON, Persistent: true}, // COUNT(*) — or O(1) via rollup
            metaengine.ADTGraph:     {Complexity: metaengine.ON, Persistent: true}, // WITH RECURSIVE
            metaengine.ADTLog:       {Complexity: metaengine.O1, Persistent: true},
        },
    }
}

func (e sqliteEngine) Open(cfg metaengine.EngineConfig) (metaengine.Engine, error) {
    db, err := sql.Open("sqlite3", cfg.DSN)
    if err != nil {
        return nil, err
    }
    return &sqliteImpl{db: db}, nil
}
```

### Deployment Config (Operator's YAML)

```yaml
# Development: single SQLite, everything in one DB
engines:
  sqlite:
    driver: sqlite
    dsn: /data/app.db

# Production: multi-engine optimization
engines:
  sqlite:
    driver: sqlite
    dsn: /data/events.db
  pebble:
    driver: pebble
    dsn: /data/pebble
  clickhouse:
    driver: clickhouse
    dsn: clickhouse://analytics:9000

# "Crazy" setup: everything optimized for extreme scale
engines:
  scylla:
    driver: scylla
    dsn: scylla://cluster:9042
  clickhouse:
    driver: clickhouse
    dsn: clickhouse://analytics:9000
  neo4j:
    driver: neo4j
    dsn: bolt://graph:7687
```

### What the Operator Controls

| Decision                               | Who           | How                                    |
| -------------------------------------- | ------------- | -------------------------------------- |
| Which engines to provide               | Operator      | `deploy.yaml`                          |
| Connection strings                     | Operator      | `deploy.yaml` DSN per engine           |
| Connection tuning                      | Operator      | `deploy.yaml` options per engine       |
| Which patterns to optimize             | Developer     | `projection.Filter()`, `.Sort()`, etc. |
| Which engine serves which pattern      | **Optimizer** | Cost-based assignment at startup       |
| Physical layout (tables, indexes, CFs) | **Optimizer** | Auto-generated from declarations       |

The operator provides engines. The developer declares query patterns. The optimizer matches
them. Neither talks to the other directly.

---

## 7. Concrete Walkthrough

### Setup

Developer declares two projections:

```go
users := projection.Declare[UserView, UserID]("users", ...,
    projection.Lookup(),                    // Map ADT
    projection.Filter("status", "country"), // SortedMap ADT
    projection.Sort("joined_at"),           // SortedMap ADT
    projection.Count("status"),             // Counter ADT
)

friendships := projection.Graph[Friendship]("friendships", ...,
    projection.Traverse(2),                 // Graph ADT
)
```

### Deployment A: Single SQLite (Dev)

```yaml
engines:
  sqlite:
    driver: sqlite
    dsn: /data/app.db
```

**Optimizer plan:**

```
Projection: users
  Lookup()           → Map       → SQLite table users, PK index         O(logN) ✓
  Filter("status")   → SortedMap → SQLite table users, idx_status        O(logN) ✓
  Filter("country")  → SortedMap → SQLite table users, idx_country       O(logN) ✓
  Sort("joined_at")  → SortedMap → SQLite table users, idx_joined_at     O(logN) ✓
  Count("status")    → Counter   → SQLite rollup: user_count_by_status   O(1)   ✓

Projection: friendships
  Traverse(2)        → Graph     → SQLite WITH RECURSIVE CTE             O(N)   ⚠ DEGRADED

⚠ WARNING: friendships.Traverse(2): no graph engine available.
  Using SQL recursive CTE — O(N) per traversal, fine for <10K relationships.
  Add Neo4j or a Graph engine for deep traversal at scale.

Generated projections: 1 (SQLite only)
  - users table + 3 indexes + 1 rollup table
  - friendships stored as edges in a junction table
```

**Everything works. One engine. Correct but graph traversal is slow at scale.**

### Deployment B: SQLite + Pebble (Balanced)

```yaml
engines:
  sqlite:
    driver: sqlite
    dsn: /data/indexes.db
  pebble:
    driver: pebble
    dsn: /data/kv
```

**Optimizer plan:**

```
Projection: users
  Lookup()           → Map       → Pebble (users_by_id)          O(1)    ✓ OPTIMAL
  Filter("status")   → SortedMap → SQLite (idx_status)            O(logN) ✓
  Filter("country")  → SortedMap → SQLite (idx_country)           O(logN) ✓
  Sort("joined_at")  → SortedMap → SQLite (idx_joined_at)         O(logN) ✓
  Count("status")    → Counter   → SQLite rollup table            O(1)    ✓

Projection: friendships
  Traverse(2)        → Graph     → Pebble adjacency lists          O(d^2)  ⚠ DEGRADED (but better than SQL CTE)

Generated projections: 2
  - Pebble: users_by_id (hash), friendships adjacency (key per edge)
  - SQLite: users table + 3 indexes + 1 rollup table
```

**Point lookups are now O(1) on Pebble. Everything else on SQLite. Graph traversal improved
but still not native.**

### Deployment C: SQLite + Pebble + Neo4j (Full Optimization)

```yaml
engines:
  sqlite:
    driver: sqlite
    dsn: /data/indexes.db
  pebble:
    driver: pebble
    dsn: /data/kv
  neo4j:
    driver: neo4j
    dsn: bolt://graph:7687
```

**Optimizer plan:**

```
Projection: users
  Lookup()           → Map       → Pebble                  O(1)    ✓ OPTIMAL
  Filter("status")   → SortedMap → SQLite idx_status        O(logN) ✓ OPTIMAL
  Filter("country")  → SortedMap → SQLite idx_country       O(logN) ✓ OPTIMAL
  Sort("joined_at")  → SortedMap → SQLite idx_joined_at     O(logN) ✓ OPTIMAL
  Count("status")    → Counter   → SQLite rollup            O(1)    ✓ OPTIMAL

Projection: friendships
  Traverse(2)        → Graph     → Neo4j native             O(d)    ✓ OPTIMAL

Generated projections: 3
  - Pebble: users_by_id (hash lookup)
  - SQLite: users table + 3 indexes + 1 rollup table (filtered/sorted/counted queries)
  - Neo4j: User nodes + FRIENDS_WITH edges (traversal queries)

ALL query patterns at OPTIMAL complexity. Zero degradation.
```

### Deployment D: "Crazy" — ScyllaDB + ClickHouse + Neo4j

```yaml
engines:
  scylla:
    driver: scylla
    dsn: scylla://cluster:9042
  clickhouse:
    driver: clickhouse
    dsn: clickhouse://analytics:9000
  neo4j:
    driver: neo4j
    dsn: bolt://graph:7687
```

**Optimizer plan:**

```
Projection: users
  Lookup()           → Map       → Scylla (partition key)         O(1)    ✓
  Filter("status")   → SortedMap → ClickHouse (columnar scan)      O(N/k)  ✓ (k=columns, columnar is fast)
  Filter("country")  → SortedMap → ClickHouse (columnar scan)      O(N/k)  ✓
  Sort("joined_at")  → SortedMap → ClickHouse (ordered by col)     O(logN) ✓
  Count("status")    → Counter   → ClickHouse (native columnar)    O(1)    ✓ OPTIMAL (columnar aggregation)

Projection: friendships
  Traverse(2)        → Graph     → Neo4j                          O(d)    ✓ OPTIMAL

⚠ NOTE: ClickHouse Filter is O(N/k) not O(logN) because ClickHouse is a scan engine,
  not an index engine. But at billion-row scale with 20 columns, O(N/20) columnar scan
  is FASTER than O(logN) B-tree random I/O. The optimizer knows this.

Generated projections: 3
  - Scylla: users_by_id (wide-column partition lookup)
  - ClickHouse: users table (columnar, optimized for analytics + counts)
  - Neo4j: friendship graph

This deployment serves BILLIONS of users with optimal query performance.
The developer's code is IDENTICAL to Deployment A (single SQLite).
```

---

## 8. The Hard Problems (Honest)

### Problem 1: Projection Handler Generation (Medium-Hard)

The optimizer must generate event handlers for each engine. For simple cases this is
mechanical:

```
UserCreated event + Pebble Map projection
  → handler: pebble.Set(user_id, encode(UserView{...}))

UserSuspended event + SQLite SortedMap projection
  → handler: sql.Exec("UPDATE users SET status=? WHERE id=?", "suspended", id)

Friendship event + Neo4j Graph projection
  → handler: neo4j.MergeEdge(:User{id:from})-[:FRIENDS_WITH]->(:User{id:to})
```

**Where it gets hard:** conditional logic, computed fields, multi-step derivations.

**Solution:** Auto-generate for the declarative 80% (simple OnEvent → view mapping). For the
20% that needs custom logic, let the developer provide a custom handler function that receives
a typed sink. The optimizer still decides WHICH engine, but the developer controls HOW events
map to writes.

```go
// Custom handler — developer controls the write logic, optimizer picks the engine
users := projection.Declare[UserView, UserID]("users",
    projection.On(UserCreated{}, func(e UserCreated) (UserID, UserView) {
        return e.ID, UserView{...}
    }),
    // ... same query pattern declarations
    projection.Lookup(),
    projection.Filter("status"),
)

// The optimizer sees Lookup + Filter and assigns:
//   Lookup → Pebble (if available) or SQLite
//   Filter → SQLite (if available) or Pebble (degraded)
// It generates the projection handlers automatically.
```

### Problem 2: Schema Migration (Medium)

When the UserView type gains a field, projections need updating:

- SQLite: `ALTER TABLE users ADD COLUMN new_field TEXT`
- Pebble: schemaless (just start writing the new field)
- Neo4j: schemaless
- ClickHouse: schema migration

**Solution:** Track schema version per projection. On startup, compare declared schema to
stored schema. Auto-migrate where possible (ALTER TABLE). Warn where manual migration needed.

### Problem 3: Write Amplification (Manageable)

More engines = more writes per event. A `UserCreated` event that feeds 3 projections (Pebble +
SQLite + Neo4j) costs 3x the write of feeding 1.

**Solution:** The optimizer reports write amplification at startup:

```
⚠ Write amplification report:
  UserCreated      → 3 projections (Pebble + SQLite + Neo4j)
  UserSuspended    → 2 projections (Pebble + SQLite)
  Friendship       → 2 projections (Neo4j + SQLite denormalized friend_count)

  Average: 2.3 writes per event. Peak: 3 (UserCreated).
  This is acceptable for <10K events/sec. For higher throughput,
  consider reducing query patterns or engines.
```

### Problem 4: Consistency (Solved by Design)

Each projection is independently eventually-consistent (standard CQRS). The event log is the
source of truth. If projections diverge, drop and replay. This is already how `projectionhost`
works today.

### Problem 5: The Read API (Hard — but solvable)

Generating `store.Users.Get(id)` and `store.Users.ByStatus("active")` at runtime requires
either reflection or code generation. The user chose runtime (no codegen step).

**Solution:** Use Go generics + interface type assertions at startup:

```go
// The optimizer builds a typed store at Plan() time
type UserStore struct {
    getByID     func(ctx context.Context, id UserID) (*UserView, error)  // wired to Pebble
    byStatus    func(ctx context.Context, status string) ([]*UserView, error) // wired to SQLite
    recent      func(ctx context.Context, limit int) ([]*UserView, error)     // wired to SQLite
    countStatus func(ctx context.Context) (map[string]int64, error)           // wired to SQLite rollup
}

store := metaengine.Plan(engines, users, friendships)
user, _ := store.Users.Get(ctx, userID)
active, _ := store.Users.ByStatus(ctx, "active")
```

The `UserStore` struct is assembled at runtime by the planner, with each function field
wired to the appropriate engine. Type-safe, no reflection on the hot path.

### Problem 6: Memory Constraints (Acknowledged)

The user correctly pointed out: "if you want to query a full 1TB DB but the server only has
50GB RAM, we need to stream/buffer/paginate."

**Solution:** The optimizer respects `MaxDataSize` in engine profiles:

- Memory engine declares `MaxDataSize: 1GB` (operator-configurable)
- If a projection would exceed this, the optimizer refuses to assign it to memory
- All read methods support `Limit(offset, count)` for pagination
- Large result sets stream via Go channels or iterators, not `[]V` slices

```go
// Paginated read
page, _ := store.Users.ByStatusPaginated(ctx, "active", offset=100, limit=50)

// Streaming read (for bulk export/processing)
err := store.Users.Stream(ctx, func(u *UserView) error {
    return processUser(u)
})
```

---

## 9. What Makes This Novel

No existing system does this in an embedded Go library:

| System                         | What it does                                | What the meta-engine does differently         |
| ------------------------------ | ------------------------------------------- | --------------------------------------------- |
| **ORMs (GORM, ent)**           | Map objects → SQL tables, single engine     | Projects events → N engines, multi-shape      |
| **CQRS frameworks (Axon)**     | Separate read/write, manual projection      | Auto-generated projections, cost-optimized    |
| **Apache Calcite**             | Federated query planning (at query time)    | Physical layout optimization (at deploy time) |
| **Materialize (streaming DB)** | Real-time views from streams, single engine | Multi-engine, embedded, event-sourced         |
| **dbt**                        | Batch transforms in warehouses              | Real-time, multi-engine, typed, library       |
| **Kafka Connect**              | Move data between systems                   | Typed read API, cost-based optimization       |
| **DuckDB (embedded OLAP)**     | Single embedded analytics engine            | Multi-engine coordinator                      |

### The Novel Contribution

**A deployment-time, cost-based storage optimizer that:**

1. Takes **event-sourced data** + **declared query patterns** as input
2. Takes **whatever engines the operator provides** as available hardware
3. **Automatically distributes projections** across engines based on cost
4. **Auto-denormalizes** to eliminate cross-REMOTE-engine reads (when engines are local, cross-engine reads are cheap syscalls)
5. **Generates typed read APIs** wired to the optimal engine per query
6. **Degrades gracefully** when optimal engines aren't available (warns, never fails silently)
7. **Respects hardware constraints** (RAM, disk, network, CPU, time)
8. Is **embedded in a Go library** (no external service, no query language to learn)

The closest analogy: **it's what a database query planner does, but one level up.** Database
planners optimize WITHIN one engine. The meta-engine optimizes ACROSS engines.

### Why Event Sourcing Makes This Possible

This ONLY works because of event sourcing. Here's why:

1. **The event log is the single source of truth.** Every projection is derived. We can have
   as many projections as we want, in as many engines as we want, without consistency risk.

2. **Projections are disposable.** If we change the physical layout (add an engine, change an
   index), we drop projections and replay from the event log. No data migration.

3. **The write side is already abstracted.** Events are appended via `event.Store`. The
   optimizer only touches the read side (projections). No write-path changes needed.

4. **Events are the natural unit of denormalization.** Each event carries the delta. The
   optimizer can route that delta to multiple engines atomically per projection (within the
   projection handler's transaction).

Without event sourcing, this would require distributed transactions across engines (hard,
slow, fragile). With event sourcing, each projection is independent and eventually consistent.
The event log is the transaction boundary.

---

## Summary: The Vision in One Paragraph

The developer writes domain logic (events, decider, fold functions) and declares what query
patterns they need on their projections (point lookup, filter, sort, count, graph traversal).
The operator provides whatever engines they want (SQLite alone, or SQLite + Pebble +
ClickHouse + Neo4j, or anything else). At startup, the meta-engine inspects the available
engines and the declared query patterns, and **automatically plans the optimal physical
layout**: which projections go in which engines, which indexes to create, what to
denormalize, and how to wire the typed read API. The developer's code is identical whether
running on a single SQLite for development or ScyllaDB + ClickHouse + Neo4j at planetary
scale. **The storage layer is not abstracted away — it's optimized away.**
