# Design: Storage-Domain Separation

> **Goal:** Consumers of this library should NOT decide on infrastructure implementation
> (DB, message buses, etc.). They declare domain shapes + query intent. The **deployer**
> decides where data lives and what engines serve it. The library matches them.

**Status:** Proposed (2026-07-23)
**Related:** [FOUR-TIER-MODEL.md](architecture-understanding/FOUR-TIER-MODEL.md),
[STORAGE_GUIDE.md](STORAGE_GUIDE.md), [INFRASTRUCTURE_RECOMMENDATIONS.md](INFRASTRUCTURE_RECOMMENDATIONS.md),
[projection-tiers.md](projection-tiers.md), [ADR-0033](adr/0033-multi-db-split.md),
[ADR-0046](adr/0046-four-tier-model.md)

---

## Table of Contents

1. [The Goal](#1-the-goal)
2. [Current State Assessment](#2-current-state-assessment)
3. [The Core Insight: Shape vs. Mechanism](#3-the-core-insight-shape-vs-mechanism)
4. [The Four Leak Points](#4-the-four-leak-points)
5. [Why The Read Side Is Harder](#5-why-the-read-side-is-harder)
6. [Three Paths To Fix It](#6-three-paths-to-fix-it)
7. [Recommended Path: "Declare + Bind"](#7-recommended-path-declare--bind)
8. [Implementation Plan](#8-implementation-plan)
9. [Open Design Question](#9-open-design-question)

---

## 1. The Goal

```
DEVELOPER (consumer)                    DEPLOYER (operator)
───────────────────                     ────────────────────
Writes domain logic                     Picks infrastructure:
Declares read model shape               - sqlite.New("events.db")
Declares query intent                   - pebble.New(dir)
NEVER imports a backend driver          - memory.New()
NEVER touches *sql.DB                   - multi-DB: WithEventDB + WithViewDB
```

The consumer says **what** they need. The deployer provides **what they have**. The library
negotiates the match.

Recommendations are fine (SQL columns are better for filtered queries than KV blobs) — but the
library must also be able to run fully with just SQLite + Memory, or multiple SQLite DBs
(e.g. 1 for Command + Event Sourcing, 1 for Query logs, 1 for materialized views).

### Design Constraints

| Constraint | Implication |
|---|---|
| Consumer code is 100% engine-agnostic | No `database/sql`, no `*sql.DB`, no SQL DDL in consumer code |
| Deployer controls infrastructure | Connection tuning, engine choice, multi-DB topology all in deployer land |
| Graceful degradation | SQL query hint + only KV available → correct (slower) result, not an error |
| Fail fast on impossible matches | Graph traversal hint + no graph driver → error at bind time |
| No leaky abstractions | No SQL concepts (WHERE, ORDER BY) leaking into KV-only interfaces |
| No anemic abstractions | Don't collapse SQL/graph power into lowest-common-denominator `Get/Set` |

---

## 2. Current State Assessment

### What Works: The Write Side

The write-side abstraction is **clean and correct**. This is the model the read side should follow.

```
Consumer writes:                     Deployer picks:
─────────────────                    ──────────────
decider.Decider[State]{...}          sqlite.New("events.db")
stack.Repository(bundle, decider)    memory.New()
                                     pebble.New(dir)
                                     postgres.New(dsn)
```

The consumer never imports `database/sql`, never touches `*sql.DB`, never knows if events live
in SQLite or Pebble. The `event.Store` interface (split into `EventSink`/`EventSource` via ISP)
is the port; presets are the adapter. The four-tier model enforces this structurally via Go
module boundaries.

**Dependency direction (write side):**

| Layer | Depends on | Role |
|-------|-----------|------|
| `stack.Bundle` | event, kv, command, query, snapshot, codec | Top: assembly point, wires deployer-provided implementations |
| `decider.Repository` | event, snapshot, codec, id, otel | Mid: domain logic — load, apply, decide, save, publish |
| `event.*` interfaces | id only | Bottom: port definitions (EventSink, EventSource, Store, Journal) |

Concrete storage adapters (SQL, memory, pebble) implement these interfaces and are injected at
the deployer level via `stack.Bundle`. Classic dependency inversion / ports-and-adapters.

### What Struggles: The Read Side

The read side has **three working paradigms** (Document/KV, Relational/SQL, Graph) but the
consumer is forced to make the storage choice that the deployer should make. The abstractions
exist (`kv.ViewStore`, `projection.Projection`), but the wiring leaks.

---

## 3. The Core Insight: Shape vs. Mechanism

The root cause of every leak: **the consumer conflates two independent decisions.**

| Decision | Who decides | What it means |
|---|---|---|
| **Shape** (domain concern) | Consumer (developer) | "I have one view per task, and I want to filter by status" |
| **Mechanism** (infrastructure concern) | Deployer (operator) | "Use SQLite columns for the view store" |

Currently, calling `sqlite.SQLViewModel[TaskView, TaskID](bundle, mapper)` makes **both**
decisions in consumer code. The consumer picks the shape (document, one row per task, queryable
columns) AND the mechanism (SQLite) in a single call.

**The fix:** separate these two decisions. The consumer declares shape + query intent. The
deployer's Bundle determines mechanism. The library matches them.

```
CONSUMER (shape)                    LIBRARY (negotiation)              DEPLOYER (mechanism)
────────────────                    ─────────────────────              ────────────────────
ReadModelSpec{                      capability check:                  Bundle{
  Type: TaskView                      - needs filter? yes              .ReadModels: kv.Store (SQL blob)
  Key: taskKey                         - store has ViewQuerier? yes    .ViewColumns: *sql.DB (real cols)
  QueryHints:                          - → SQLViewStore + indexes      .GraphDriver: nil
    Filter: ["status"]               - needs traversal? no           }
    Sort: ["created_at"]             - → skip GraphProjection
}                                   downgrade? check warnings
```

---

## 4. The Four Leak Points

Traced from consumer code to storage internals. These are the exact places where the
abstraction breaks today.

### Leak 1: `bundle.Database()` returns `any` (Medium severity)

**Location:** `stack/bundle.go:206` — exposed as `func (b *Bundle) Database() any`

**Consumer impact:** `example/taskmanager/setup.go:77-79`:

```go
if db, ok := bundle.Database().(*sql.DB); ok {
    db.SetMaxOpenConns(1)
}
```

The consumer must import `database/sql` and type-assert on an erased `any` to tune connections.

**Fix:** Move connection tuning into presets. `sqlite.New(dsn, sqlite.WithMaxOpenConns(1))`.
Remove `Database()` from the public API (or keep it as an explicit escape hatch, clearly marked).

### Leak 2: `storage.ViewMapper[V]` exposes SQL types (High severity)

**Location:** `storage/view/store.go:50` — `Type: "TEXT"`, `Type: "INTEGER"`

**Consumer impact:** The consumer writes raw SQL DDL when defining a view model:

```go
mapper := storage.ViewMapper[TodoView]{
    Table: "todos_view",
    Columns: []storage.ViewColumn[TodoView]{
        {Name: "title", Type: "TEXT", Extract: func(v *TodoView) any { return v.Title }},
        {Name: "completed", Type: "INTEGER", Extract: ...},
    },
    ScanRow: func(scan func(dest ...any) error) (*TodoView, error) { ... },
}
```

Column names, SQL type strings, and a manual `sql.Rows.Scan` callback are all in consumer code.

**Fix:** Replace SQL type strings with a neutral `stack.ColumnType` enum. The SQL adapter
translates `ColumnType.String` → `"TEXT"`, `ColumnType.Int` → `"INTEGER"`, etc. AutoMapper
(from struct tags) already reduces this but doesn't eliminate it.

### Leak 3: `sqlite.SQLViewModel` returns a concrete SQL type (High severity)

**Location:** `stack/sqlite/view_models.go:45`

```go
func SQLViewModel[V any, K fmt.Stringer](b *stack.Bundle, mapper storage.ViewMapper[V]) (*storage.SQLViewStore[V, K], error)
```

**Consumer impact:** The consumer holds a `*storage.SQLViewStore` — a concrete infrastructure
type, not an interface. They're coupled to the SQL implementation.

**Fix:** Return `kv.ViewStore[V,K]` (which `SQLViewStore` already implements). The consumer
holds the interface; the optional capability interfaces (`ViewQuerier`, `ViewCounter`,
`TombstoneQuerier`) are available via type assertion if needed.

### Leak 4: `RelationalProjection` takes `*sql.DB` directly (High severity)

**Location:** `storage/relational/projection.go`

```go
func NewRelationalProjection(name string, schema RelationalSchema, db *sql.DB, dialect sqlpkg.Dialect, handler RelationalHandler, types []event.Type) (*RelationalProjection, error)
```

**Consumer impact:** The consumer passes `*sql.DB` and a `Dialect` directly into a projection
constructor. This means the consumer imports `database/sql` and the `storage/sql` package.

**Fix:** Add `bundle.RelationalProjection(name, schema, handler, types)` — the Bundle provides
`*sql.DB` internally from whatever preset the deployer chose. If no SQL backend is available,
return a clear error.

### Leak Summary Table

| # | Leak | Severity | Consumer must... | Proposed fix |
|---|------|----------|-----------------|--------------|
| 1 | `bundle.Database()` returns `any` | Medium | Import `database/sql`, assert `*sql.DB` | Move tuning to preset options |
| 2 | `ViewMapper.Type` uses SQL strings | High | Write SQL DDL (column types) | Neutral `ColumnType` enum |
| 3 | `SQLViewModel` returns concrete type | High | Hold `*storage.SQLViewStore` | Return `kv.ViewStore[V,K]` interface |
| 4 | `RelationalProjection` takes `*sql.DB` | High | Pass `*sql.DB` + `Dialect` | Bundle method provides DB internally |

### What Does NOT Leak (the clean paths)

- **KV/document read models:** `stack.NewMaterialize[V,K](bundle, codec, keyFunc)` — consumer
  gets a `kv.ViewStore` interface from the Bundle, never knows if it's memory, Pebble, or a
  SQL blob table.
- **Decider/Repository:** `stack.Repository(bundle, decider)` — no storage types leak.
- **ProjectionHost:** `projectionhost.New(bundle.SeekableJournal, bundle.CheckpointStore)` —
  pure interfaces.

---

## 5. Why The Read Side Is Harder

The write side has **one shape**:

```
Append-only event log → one port (event.Store) → all backends implement it
```

The read side has **three fundamentally different shapes**, each optimal for different query
patterns:

| Shape | Query Pattern | Sink API | Backends | Current projection type |
|-------|--------------|----------|----------|------------------------|
| **Document** (1 entity = 1 blob) | Point lookup, scan all, prefix | `Get/Set/Delete` | KV, SQL blob, Memory | `stack.Materialize[V,K]` |
| **Relational** (1 entity = N joined rows) | Filtered lists, counts, pagination, junction tables | `Upsert/Update/Increment/Query` | SQL only | `storage.RelationalProjection` |
| **Graph** (nodes + edges) | Variable-depth traversal, path-finding, adjacency | `MergeNode/MergeEdge/Traverse` | Graph DB only | `graph.GraphProjection` |

**Why these cannot collapse into one universal interface:**

- **Leaky collapse:** A "universal" interface that exposes SQL concepts (WHERE, ORDER BY) to KV
  backends — leaks relational semantics where they don't belong.
- **Anemic collapse:** A lowest-common-denominator interface (just `Get/Set`) that throws away
  SQL/graph power — the whole point of having multiple tiers is lost.

The three-tier projection design (`Materialize` / `RelationalProjection` / `GraphProjection`)
is **architecturally correct**. The problem is not that there are three paradigms — it's that
**the consumer currently makes the storage choice, when the deployer should.**

A hybrid exists too: `SQLViewStore` implements `kv.ViewStore` (so it drops into `Materialize`)
but also implements `ViewQuerier`, `ViewCounter`, `TombstoneQuerier`, `ViewResetter`,
`ViewBatchSetter`, and `ViewUpdater`. This is the migration path from "blob per key" to
"queryable columns" without changing projection code.

---

## 6. Three Paths To Fix It

### Path A: Surgical Leak Fixes (Minimal Disruption)

Fix the four leaks without adding new abstractions.

**Changes:**

1. Kill `bundle.Database()` → add connection tuning to presets (`WithMaxOpenConns`, etc.)
2. Neutralize `ViewMapper` types → replace `"TEXT"`/`"INTEGER"` with `stack.ColumnType` enum
3. Return interfaces, not concrete types → `SQLViewModel` returns `kv.ViewStore[V,K]`
4. `bundle.RelationalProjection(name, schema, handler, types)` → Bundle provides `*sql.DB` internally

**Pros:**

- Small, safe, API-compatible-ish
- Fixes every identified leak
- No new concepts for consumers to learn

**Cons:**

- Consumer still calls `sqlite.SQLViewModel` (chooses mechanism explicitly)
- Does not achieve full shape/mechanism separation
- It is a band-aid, not a cure — the fundamental conflation remains

**Effort:** ~1 day

### Path B: Deployment Manifest (Full Separation)

Add a declarative deployment layer where the deployer writes a manifest and the consumer never
touches the Bundle for read models.

```go
// CONSUMER: pure declaration
tasks := stack.ReadModel[TaskView, TaskID]{
    Name: "tasks", Key: taskKey, OnCreate: ..., OnUpdate: ...,
    Query: stack.QueryHints{Filter: []string{"status"}, Sort: []string{"created_at"}},
}

// DEPLOYER: manifest (no Go types, just config)
deployment := deploy.Manifest{
    Events: deploy.SQLite("events.db"),
    Views:  deploy.SQLiteColumns("views.db"),
}
app := deploy.Wire(deployment, TaskDecider, tasks, users, ...)
```

**Pros:**

- Cleanest separation
- Deployer has total control via config
- Consumer code is 100% infrastructure-free

**Cons:**

- Large redesign
- Type-erasure challenges (manifest cannot know `V` and `K` types at config time)
- Risk of over-abstraction and "magic"
- Breaks existing API significantly

**Effort:** ~1-2 weeks

### Path C: "Declare + Bind" (Recommended) ⭐

Mirror the pattern that already works for the write side (`stack.Repository(bundle, decider)`)
and extend it to read models. Two phases: the consumer declares intent, the Bundle binds mechanism.

**Effort:** ~4-5 days (see [Implementation Plan](#8-implementation-plan))

---

## 7. Recommended Path: "Declare + Bind"

### The Consumer Experience

```go
// ════════════ CONSUMER CODE (engine-agnostic, never imports storage) ════════════

// 1. Domain model — same as today, pure functions
var TaskDecider = decider.Decider[TaskState]{
    Initial: TaskState{},
    Apply:   foldTask,
}

// 2. Read model DECLARATION — storage-agnostic specification
tasksRM := stack.Declare[TaskView, TaskID]("tasks").
    Key(taskKeyFromEvent).
    OnCreate(taskOnCreate).
    OnUpdate(taskOnUpdate).
    OnTombstone(taskOnDelete).
    // Query hints: domain intent, NOT storage mechanism
    FilterBy("status", "assignee").    // "I need WHERE status = ?"
    SortBy("created_at").              // "I need ORDER BY created_at"
    Done()

// ════════════ DEPLOYER CODE (operator decides infrastructure) ════════════

// 3. Deployer picks infrastructure (same as today)
bundle, _ := sqlite.New("events.db",
    sqlite.WithViewDB("views.db"),      // separate DB for read models
    sqlite.WithMaxOpenConns(1),         // tuning stays in deployer land
)

// 4. BIND — the library matches declaration → best available store
//    SQL available + filter hints → SQLViewStore with real columns + indexes
//    Only memory available        → kv.TypedStore (blob) + in-memory filter
//    (correct but slower — capability downgrade, never silent failure)
mat, _ := stack.Bind(bundle, tasksRM)

// 5. Run (same as today)
repo, _ := stack.Repository(bundle, TaskDecider)
host, _ := stack.ProjectionHost(bundle)
host.Register(mat)
```

### Capability Negotiation at Bind Time

This is the core mechanism. `stack.Bind` inspects what the Bundle provides and matches it
against what the `ReadModelSpec` requests.

| Declaration says | Bundle has | Result | Why |
|---|---|---|---|
| `FilterBy("status")` | SQL columns available | `SQLViewStore` + index on `status` | Native WHERE, O(log n) |
| `FilterBy("status")` | Memory/KV only | `kv.TypedStore` + in-memory scan | Correct, O(n), logs warning |
| `SortBy("created_at")` | SQL columns available | `SQLViewStore` + index on `created_at` | Native ORDER BY |
| `SortBy("created_at")` | Memory/KV only | `kv.TypedStore` + in-memory sort | Correct, slower |
| No query hints | Anything | `kv.TypedStore` (blob) | Simplest, point-lookup optimal |
| Graph traversal hints | `GraphDriver` in Bundle | `GraphProjection` | Native traversal |
| Graph traversal hints | No `GraphDriver` | **Error at Bind time** | Fail fast, don't silently degrade |

### How It Reuses Existing Machinery

The optional capability interfaces (`ViewQuerier`, `ViewCounter`, `TombstoneQuerier`) that are
already built become the **runtime detection mechanism**. `Bind` checks for them on the
candidate store and wires accordingly. The capability detection is lifted from the store level
(where it is now — consumers must type-assert) to the declaration level (where the consumer
lives — `FilterBy`/`SortBy` hints).

| Existing component | Role in Declare+Bind |
|---|---|
| `kv.ViewStore[V,K]` | Interface that `Bind` returns as the projection's store |
| `kv.ViewQuerier[V]` | Runtime check: does this store support filtering? |
| `kv.ViewCounter[V]` | Runtime check: does this store support counting? |
| `storage.SQLViewStore` | One implementation of all capability interfaces (SQL columns) |
| `kv.TypedStore` | Basic implementation (blob store, no capabilities) |
| `stack.Materialize` | The projection wrapper that `Bind` produces internally |
| `projection.Projection` | The common interface all three tiers implement |

### Pros

- Consumer code is **100% infrastructure-free** — no `database/sql`, no `*sql.DB`, no SQL types
- Deployer controls everything via preset choice (as today)
- Graceful degradation: SQLite dev → Postgres prod with zero consumer code changes
- Fails fast when capabilities are genuinely impossible (graph without graph DB)
- **Reuses existing machinery**: `kv.ViewStore` + capability interfaces are already built
- Same pattern as write side (`stack.Repository(bundle, decider)`) — consistency
- Existing `Materialize`/`RelationalProjection`/`GraphProjection` remain as low-level escape
  hatches for power users who want explicit control

### Cons

- New `Declare`/`Bind` API surface (but `Materialize` remains as low-level escape hatch)
- `FilterBy`/`SortBy` hints are a new concept consumers must learn
- Capability downgrade (in-memory scan) could be silently slow if not logged clearly
- `Declare` builder produces a type-erased spec that `Bind` must handle generically

---

## 8. Implementation Plan

Incremental, no breaking changes to existing APIs.

### Step 1: Close the leaks (Path A fixes)

**Goal:** Eliminate all four leak points from consumer-facing code.

| Task | Files affected | Description |
|---|---|---|
| Remove `bundle.Database()` escape hatch | `stack/bundle.go`, presets | Add `WithMaxOpenConns`, `WithConnMaxLifetime` etc. to `sqlite`/`postgres` presets. Keep `Database()` as deprecated/escape hatch. |
| Neutralize `ViewMapper.Type` | `storage/view/store.go`, `stack/` | Replace `string` SQL types with `stack.ColumnType` enum (`String`, `Int`, `Bool`, `Real`, `Bytes`, `Timestamp`). SQL adapter translates to DDL strings. |
| Return interfaces from constructors | `stack/sqlite/view_models.go`, `stack/postgres/` | `SQLViewModel` returns `kv.ViewStore[V,K]`, not `*storage.SQLViewStore`. Capability interfaces available via assertion. |
| Bundle method for relational projections | `stack/bundle.go` or `stack/sqlite/` | `bundle.RelationalProjection(name, schema, handler, types)` provides `*sql.DB` internally. Error if no SQL backend. |

**Effort:** ~1 day
**Risk:** Low — additive changes and type widening (concrete → interface)

### Step 2: Add `stack.Declare[V,K]` + `stack.Bind(bundle, rm)`

**Goal:** The consumer-facing Declare+Bind API.

| Task | Description |
|---|---|
| `ReadModelSpec[V,K]` type | Struct with handler funcs + `QueryHints` (Filter []string, Sort []string, Index []string) + graph hints (Traversal bool) |
| `stack.Declare[V,K](name)` fluent builder | Produces `ReadModelSpec[V,K]` via `.Key()`, `.OnCreate()`, `.OnUpdate()`, `.OnTombstone()`, `.FilterBy()`, `.SortBy()`, `.Done()` |
| `stack.Bind(bundle, spec)` function | Inspects Bundle capabilities, selects best store implementation, returns `projection.Projection` |
| Capability detection logic | Check Bundle for: SQL columns (`*sql.DB` available), KV store, graph driver. Match against `QueryHints`. |
| Index auto-creation | When SQL columns selected, auto-create indexes for `FilterBy`/`SortBy` columns |

Under the hood, `Bind` does exactly what the consumer does by hand today: picks `TypedStore` or
`SQLViewStore` or `GraphProjection` based on what is available.

**Effort:** ~2-3 days
**Risk:** Medium — new API surface, but existing APIs remain untouched

### Step 3: Add capability-mismatch diagnostics

**Goal:** Make degradation visible and impossible-matches fail fast.

| Task | Description |
|---|---|
| `BindWarning` type | Returned alongside projection when degradation occurs: `"filter hint for 'status' but only KV blob store available — using in-memory scan"` |
| Clear logging | `slog.Warn` on degradation with column name and fallback strategy |
| Typed errors for impossible matches | `ErrCapabilityUnavailable`: `"graph traversal requested but no GraphDriver in Bundle — add a graph backend or remove Traversal hint"` |
| `BindResult` struct | `{ Projection projection.Projection; Warnings []BindWarning }` so consumers can surface diagnostics |

**Effort:** ~1 day
**Risk:** Low — diagnostics layer only

### Total effort: ~4-5 days

---

## 9. Open Design Question

The one design tension that needs resolution before implementation:

**How should consumers express query intent?**

### Option A: Declarative hints (simpler)

```go
stack.Declare[TaskView, TaskID]("tasks").
    FilterBy("status", "assignee").
    SortBy("created_at").
    Done()
```

- Consumer provides metadata about query patterns
- Library wires to the right store + creates indexes
- Consumer still writes actual queries separately (via `ViewQuerier` at read time)
- **Simpler, less expressive, easier to optimize across backends**

### Option B: Expressive query functions (more powerful)

```go
stack.Declare[TaskView, TaskID]("tasks").
    Query(func(q stack.Query[TaskView]) stack.QueryBuilder[TaskView] {
        return q.Where("status", q.Eq).OrderBy("created_at").Limit(50)
    }).
    Done()
```

- Consumer provides actual query logic
- Library compiles to the right store's query mechanism
- **More expressive, but harder to optimize across backends, risk of re-creating an ORM**

### Recommendation

Start with **Option A** (declarative hints). It is sufficient for 90% of read-model use cases,
keeps the API simple, and avoids the ORM trap. The actual query execution at read time already
works via `ViewQuerier` / `RelationalStore.Query` — the hints are only about which store to
build and what indexes to create, not about how to execute queries.
