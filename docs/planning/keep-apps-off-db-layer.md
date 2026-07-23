# Architecture: Keep Apps Off The DB Layer

> **The problem in one sentence:** Application code must never import a storage backend.
> When it does, the app has "gone down on the DB layer" — it cannot survive an engine
> swap, a driver update, or a backend removal without editing domain code.

**Status:** Proposed (2026-07-23)
**Supersedes:** This document consolidates and refines the conclusions from
[storage-domain-separation.md](storage-domain-separation.md) and
[storage-plugin-system.md](storage-plugin-system.md). Those documents contain
the full design analysis; this one contains the actionable plan.

---

## Table of Contents

1. [The Journey: How We Got Here](#1-the-journey-how-we-got-here)
2. [The Principle: Dependency Direction](#2-the-principle-dependency-direction)
3. [What "Going Down On The DB Layer" Means Today](#3-what-going-down-on-the-db-layer-means-today)
4. [The Four Leaks](#4-the-four-leaks)
5. [The Fix: Close The Leaks (Path A)](#5-the-fix-close-the-leaks-path-a)
6. [The Plugin System Is A Secondary Layer](#6-the-plugin-system-is-a-secondary-layer)
7. [Lessons From go-plugin-mvp](#7-lessons-from-go-plugin-mvp)
8. [Why WASM / .so Plugins Don't Work For Storage](#8-why-wasm--so-plugins-dont-work-for-storage)
9. [The Read Side: Shape vs. Mechanism](#9-the-read-side-shape-vs-mechanism)
10. [Action Plan](#10-action-plan)

---

## 1. The Journey: How We Got Here

This investigation went through three phases before arriving at the core insight:

1. **Phase 1 — "Extract the storage layer":** Initial analysis of the domain/storage boundary.
   Found that the write side is clean but the read side has leaky abstractions. Proposed three
   paths: surgical leak fixes, deployment manifest, or a "Declare + Bind" API.

2. **Phase 2 — "Make storage actual plugins":** Explored Go's plugin mechanisms (database/sql
   registration, Go `plugin` package, hashicorp/go-plugin, WASM). Designed a full plugin
   system with `Plugin` interface, registry, and `Deploy(config)`.

3. **Phase 3 — The key insight:** **The problem isn't plugin loading. The problem is
   dependency direction.** App code imports storage packages. That's the entire disease.
   The fix is to close 4 specific leak points. Everything else (plugin system, capability
   negotiation, Declare/Bind) is secondary polish on a solid foundation.

---

## 2. The Principle: Dependency Direction

```
APP CODE                        INFRASTRUCTURE (deployer's problem)
────────                        ──────────────────────────────────
imports ONLY:                   provides:
  event/  (interfaces)            storage/sqlite/
  command/ (interfaces)           storage/pebble/
  query/   (interfaces)           watermill/
  decider/                        bus/nats/
  kv/     (interfaces)
  projection/ (interfaces)
  stack/  (assembly helpers)

NEVER imports:                  registered via:
  database/sql                    init() in each preset
  storage/*                       activated by deployer config
  *sql.DB
  watermill/
```

If the app code never imports a storage package, then:

- Swap SQLite -> Postgres? **App doesn't change.**
- Pebble has a breaking API change? **App doesn't change.**
- Remove a backend entirely? **App doesn't compile-fail.**
- DB driver has a bug? **App returns errors through clean interfaces, doesn't crash.**

**The write side already does this perfectly.** `decider.Repository` depends on `event.Store`
(interface). The app never touches `*sql.DB`. The read side must match.

### Dependency Direction Summary

| Layer                        | Depends on                                 | Role                                                              |
| ---------------------------- | ------------------------------------------ | ----------------------------------------------------------------- |
| `stack.Bundle`               | event, kv, command, query, snapshot, codec | Top: assembly point, wires deployer-provided implementations      |
| `decider.Repository`         | event, snapshot, codec, id, otel           | Mid: domain logic — load, apply, decide, save, publish            |
| `event.*` interfaces         | id only                                    | Bottom: port definitions (EventSink, EventSource, Store, Journal) |
| `kv.Store` / `kv.TypedStore` | codec only                                 | Bottom: port definitions for read models                          |

Concrete storage adapters (SQL, memory, pebble) implement these interfaces and are injected at
the deployer level via `stack.Bundle`. Classic dependency inversion / ports-and-adapters.

---

## 3. What "Going Down On The DB Layer" Means Today

The taskmanager example demonstrates every leak:

```go
// example/taskmanager/setup.go — THIS is the app going down on the DB layer:

// LEAK: app imports database/sql and type-asserts *sql.DB
if db, ok := bundle.Database().(*sql.DB); ok {
    db.SetMaxOpenConns(1)
}

// LEAK: app imports storage/sqlite and writes SQL DDL
store, _ := sqlite.SQLViewModel[TaskView, TaskID](bundle,
    storage.ViewMapper[V]{
        Columns: []storage.ViewColumn[V]{
            {Name: "title", Type: "TEXT", Extract: ...},
            {Name: "completed", Type: "INTEGER", Extract: ...},
        },
        ScanRow: func(scan func(dest ...any) error) (*TodoView, error) { ... },
    },
)
```

The app **imports `database/sql`.** The app **writes SQL column types.** The app **calls
SQLite-specific constructors.** If you swap the DB, all of this breaks. That's going down on
the DB layer.

### What Does NOT Leak (the clean paths)

- **KV/document read models:** `stack.NewMaterialize[V,K](bundle, codec, keyFunc)` — consumer
  gets a `kv.ViewStore` interface from the Bundle, never knows if it's memory, Pebble, or a
  SQL blob table.
- **Decider/Repository:** `stack.Repository(bundle, decider)` — no storage types leak.
- **ProjectionHost:** `projectionhost.New(bundle.SeekableJournal, bundle.CheckpointStore)` —
  pure interfaces.

---

## 4. The Four Leaks

Traced from consumer code to storage internals. These are the exact, complete set of places
where app code is forced to depend on a storage backend.

### Leak 1: `bundle.Database()` returns `any` (Medium severity)

**Location:** `stack/bundle.go:206` — `func (b *Bundle) Database() any`

**Consumer code:** `example/taskmanager/setup.go:77-79`

```go
if db, ok := bundle.Database().(*sql.DB); ok {
    db.SetMaxOpenConns(1)
}
```

The consumer must import `database/sql` and type-assert on an erased `any` to tune connections.

### Leak 2: `storage.ViewMapper[V]` exposes SQL types (High severity)

**Location:** `storage/view/store.go:50` — `Type: "TEXT"`, `Type: "INTEGER"`

The consumer writes raw SQL DDL (column names, SQL type strings, scan callbacks) when defining
a view model:

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

### Leak 3: `sqlite.SQLViewModel` returns a concrete SQL type (High severity)

**Location:** `stack/sqlite/view_models.go:45`

```go
func SQLViewModel[V any, K fmt.Stringer](b *stack.Bundle, mapper storage.ViewMapper[V]) (*storage.SQLViewStore[V, K], error)
```

The consumer holds a `*storage.SQLViewStore` — a concrete infrastructure type, not an interface.

### Leak 4: `RelationalProjection` takes `*sql.DB` directly (High severity)

**Location:** `storage/relational/projection.go`

```go
func NewRelationalProjection(name string, schema RelationalSchema, db *sql.DB, dialect sqlpkg.Dialect, handler RelationalHandler, types []event.Type) (*RelationalProjection, error)
```

The consumer passes `*sql.DB` and a `Dialect` directly into a projection constructor. No
`stack.RelationalProjection(...)` helper exists.

### Leak Summary Table

| #   | Leak                                   | Severity | Consumer must...                        | Fix                                  |
| --- | -------------------------------------- | -------- | --------------------------------------- | ------------------------------------ |
| 1   | `bundle.Database()` returns `any`      | Medium   | Import `database/sql`, assert `*sql.DB` | Move tuning to preset options        |
| 2   | `ViewMapper.Type` uses SQL strings     | High     | Write SQL DDL (column types)            | Neutral `ColumnType` enum            |
| 3   | `SQLViewModel` returns concrete type   | High     | Hold `*storage.SQLViewStore`            | Return `kv.ViewStore[V,K]` interface |
| 4   | `RelationalProjection` takes `*sql.DB` | High     | Pass `*sql.DB` + `Dialect`              | Bundle method provides DB internally |

---

## 5. The Fix: Close The Leaks (Path A)

**This is the primary recommendation.** Close the 4 leaks. No new abstractions, no plugin
system, no Declare/Bind pattern. Just fix the dependency direction.

### Fix 1: Move connection tuning into presets

```go
// BEFORE (app imports database/sql):
bundle, _ := sqlite.New("events.db")
if db, ok := bundle.Database().(*sql.DB); ok {
    db.SetMaxOpenConns(1)
}

// AFTER (app never touches database/sql):
bundle, _ := sqlite.New("events.db",
    sqlite.WithMaxOpenConns(1),
    sqlite.WithConnMaxLifetime(5 * time.Minute),
)
// bundle.Database() deprecated or removed from public API.
```

### Fix 2: Neutralize ViewMapper types

```go
// BEFORE (app writes SQL DDL):
Columns: []storage.ViewColumn[TodoView]{
    {Name: "title", Type: "TEXT", Extract: ...},
    {Name: "completed", Type: "INTEGER", Extract: ...},
}

// AFTER (app uses neutral type, SQL adapter translates):
Columns: []stack.ViewColumn[TodoView]{
    {Name: "title", Type: stack.TypeString, Extract: ...},
    {Name: "completed", Type: TypeBool, Extract: ...},
}
// stack.ColumnType enum: TypeString, TypeInt, TypeBool, TypeReal, TypeBytes, TypeTimestamp
```

### Fix 3: Return interfaces from constructors

```go
// BEFORE (app holds concrete SQL type):
store, _ := sqlite.SQLViewModel[V,K](bundle, mapper)
// store is *storage.SQLViewStore[V,K] — concrete, SQL-coupled

// AFTER (app holds interface):
store, _ := stack.ViewModel[V,K](bundle, mapper)
// store is kv.ViewStore[V,K] — interface, engine-agnostic
// Optional capabilities (ViewQuerier, ViewCounter) available via type assertion
```

### Fix 4: Bundle method for relational projections

```go
// BEFORE (app passes *sql.DB + Dialect):
proj, _ := storage.NewRelationalProjection("messages", schema, db, sqlpkg.SQLiteDialect{}, handler, types)

// AFTER (Bundle provides DB internally):
proj, _ := stack.RelationalProjection(bundle, "messages", schema, handler, types)
// If no SQL backend in Bundle, returns clear error.
```

### What The App Looks Like After All 4 Fixes

```go
// App code — ZERO storage imports
package main

import (
    "github.com/larsartmann/go-cqrs-lite/decider/v4"
    "github.com/larsartmann/go-cqrs-lite/stack/v4"
    // NO database/sql, NO storage/, NO *sql.DB
)

func main() {
    bundle, _ := stack.Deploy(loadConfig("deploy.yaml"))

    repo, _ := stack.Repository(bundle, TaskDecider,
        decider.WithSnapshotStore[TaskState](bundle.SnapshotStore),
    )

    mat, _ := stack.NewMaterialize[TaskView, TaskID](bundle, bundle.DefaultCodec(), taskKey)

    host, _ := stack.ProjectionHost(bundle)
    host.Register(mat)

    go host.Start(ctx)
    defer host.Stop()
}
```

No `database/sql`. No `storage/`. No `*sql.DB`. No SQL types. The app cannot go down on the
DB layer because **it doesn't know a DB layer exists.**

---

## 6. The Plugin System Is A Secondary Layer

Closing the 4 leaks is the foundation. Once app code only imports interfaces, a plugin system
becomes an **optional convenience** on top — not a requirement for correctness.

### The database/sql Pattern

Go's most successful plugin architecture is `database/sql`. Every SQL driver (pq, sqlite3,
mysql) is a "plugin" loaded via blank import and activated by name string:

```go
import (
    _ "github.com/mattn/go-sqlite3"  // registers "sqlite3" driver
    _ "github.com/lib/pq"            // registers "postgres" driver
)

db, _ := sql.Open("sqlite3", "test.db")  // activate by name
```

This is how Go does plugins. Not runtime `.so` loading. Not out-of-process binaries. Blank
import + name-based activation.

### Applied to go-cqrs-lite

Each preset registers itself via `init()`:

```go
// stack/sqlite/plugin.go
func init() {
    stack.Register(sqlitePlugin{})
}
```

The deployer activates via config:

```yaml
# deploy.yaml — the deployer owns this file, not the developer
events:
  driver: sqlite
  dsn: /data/events.db
  options:
    wal: true
    maxOpenConns: 1
views:
  driver: pebble
  dsn: /data/views
bus:
  driver: memory
```

```go
// main.go
import (
    _ "go-cqrs-lite/stack/sqlite/v4"  // registers "sqlite"
    _ "go-cqrs-lite/stack/pebble/v4"  // registers "pebble"
    _ "go-cqrs-lite/stack/memory/v4"  // registers "memory"
)

func main() {
    cfg := stack.LoadConfig("deploy.yaml")
    bundle, _ := stack.Deploy(cfg)  // deployer's YAML decides
}
```

Changing the entire storage topology is a config edit, not a code change. The app's Go code
is identical regardless of backend.

### The Plugin Interface

```go
// stack/plugin.go
type Plugin interface {
    Name() string
    Capabilities() CapabilitySet
    Build(cfg PluginConfig) (*Partial, error)
}

type CapabilitySet struct {
    EventStore  bool
    ViewStore   bool
    Snapshot    bool
    Checkpoint  bool
    CommandLog  bool
    QueryLog    bool
    Bus         bool
}
```

### Why This Is Secondary

The plugin system adds deployment convenience (config-driven selection, multi-backend
mixing) but it does NOT fix the dependency direction problem. Even without the plugin system,
if the 4 leaks are closed, the app code is engine-agnostic. The plugin system just makes the
deployer's life easier.

**Build order:** Fix the 4 leaks first. Then add the plugin system. Not the other way around.

---

## 7. Lessons From go-plugin-mvp

The `/home/lars/projects/go-plugin-mvp/` project (a WASM plugin marketplace built on
go-cqrs-lite) already hit this exact wall and reveals both what to copy and what to avoid.

### What go-plugin-mvp Does Right

| Pattern                                          | Where                            | Lesson                                                                        |
| ------------------------------------------------ | -------------------------------- | ----------------------------------------------------------------------------- |
| Interface-driven stores with concrete ReadModels | `marketplace/store.go:68-74`     | Reader interfaces as compile-time conformance checks, not runtime indirection |
| Three-tier override pattern                      | `container/container.go:208-242` | Explicit override -> backend enum -> default memory. Pragmatic.               |
| Lazy DI construction via samber/do               | `container/container.go`         | Services build on first use, not at startup                                   |
| Lifecycle wrappers                               | `aclLifecycle`                   | DI container manages cleanup without store knowing about lifecycle            |
| Shared `*sql.DB` handle                          | `sqlite.Store.DB()`              | EventStore and AuditStore share one connection pool                           |

### What go-plugin-mvp Does Wrong (The Anti-Pattern)

```go
// marketplace/container/container.go — THE CLOSED-ENUM ANTI-PATTERN
func providePersistence(inj do.Injector, opts Options) {
    do.Provide(inj, func(i do.Injector) (marketplace.ACLStore, error) {
        var store marketplace.ACLStore
        if opts.ACL != nil {                              // 1. Explicit override wins
            store = opts.ACL
        } else if opts.Backend == BackendSQLite {         // 2. Closed enum -> sqlite
            store = do.MustInvoke[*sqlite.Store](i)
        } else {                                           // 3. Default -> memory
            store = memory.NewStore()
        }
        return &aclLifecycle{ACLStore: store}, nil
    })
}
```

The `Backend` type is a closed enum with two values. Adding PostgreSQL, CockroachDB, DynamoDB,
or any third backend requires **forking `container.go`** and adding another `else if`.

The PostgreSQL proposal (`docs/proposals/024-postgresql-event-store.md`) explicitly acknowledges
this gap: it says "switch `Config.EventStoreBackend` from sqlite to postgres" — but the
architecture doesn't support it without modifying `container.go`.

**go-cqrs-lite must avoid this.** The registry pattern (`Register("sqlite", factory)`) makes
the set of backends open for extension. Third-party packages `init()`-register themselves.
No enum to modify, no container to fork.

### The WASM Plugin System (For Compute, Not Storage)

go-plugin-mvp uses Extism + wazero (pure-Go WASM) for **compute plugins** — stateless or
state-via-host-functions functions that transform data (greet, count, transform, chart). This
is excellent and correct:

- Compute plugins are coarse-grained (one call per user action)
- They are stateless or use host KV
- They are latency-tolerant
- They run in a sandbox (no direct filesystem/network)

Storage adapters are the opposite of all of these. See section 8.

---

## 8. Why WASM / .so Plugins Don't Work For Storage

Go's compilation model is closed at build time. "Actual plugins loaded at deployment time"
for **storage** means one of these, and none are viable:

| Approach                        | True runtime loading?             | Performance                                    | Viability for storage                                                                        |
| ------------------------------- | --------------------------------- | ---------------------------------------------- | -------------------------------------------------------------------------------------------- |
| **Registration (database/sql)** | No — compiled in, config-selected | Direct calls (zero overhead)                   | **The only correct answer**                                                                  |
| **Go `plugin` package (.so)**   | Yes                               | Direct calls                                   | **Trap** — exact toolchain+dep version coupling, Linux/macOS only, Go team doesn't recommend |
| **hashicorp/go-plugin**         | Yes — separate binaries           | gRPC serialization on EVERY call               | Application-level, not library-level                                                         |
| **WASM (Extism/wazero)**        | Yes — .wasm files                 | Serialization + sandbox boundary on every call | **Absurd for storage** — you sandbox the adapter then punch holes for every I/O operation    |

### Why WASM Is Absurd For Storage

Storage adapters are:

- **Fine-grained** — called on every event append, every projection update
- **Stateful** — connection pools, transactions, buffers
- **Latency-critical** — microseconds matter on the write path

Running `store.Save()` through a WASM boundary means: serialize events to bytes -> cross
sandbox boundary -> host function writes to disk -> serialize response back -> cross boundary
again. On every single event. That is 100x+ overhead for zero benefit.

WASM is excellent for **compute plugins** (as go-plugin-mvp proves). It is the wrong tool for
storage adapters.

### The Key Insight: The Library Defines The Contract, Not The Loading Mechanism

The library defines `Plugin` as the contract. **How plugins arrive in the process is an
application concern:**

1. **Most apps:** Blank imports compile plugins in. Config picks which to activate. Zero
   overhead. This is `database/sql` and it works.
2. **Apps needing runtime loading (rare):** Wrap `go-plugin` (out-of-process) or Extism (WASM)
   around the `Plugin` interface. The plugin binary implements `Plugin` via gRPC proxy. The
   library doesn't know or care.
3. **Nobody should use Go's `plugin` package.** It's a trap.

The library doesn't impose serialization overhead on everyone. It provides the `Plugin`
contract and the `Deploy(config)` assembly. Loading mechanism is the application's choice.

---

## 9. The Read Side: Shape vs. Mechanism

The write side has one shape: append-only event log -> one port (`event.Store`) -> all
backends implement it.

The read side has three fundamentally different shapes, each optimal for different query
patterns:

| Shape                                     | Query Pattern                                       | Sink API                        | Backends             | Current projection type        |
| ----------------------------------------- | --------------------------------------------------- | ------------------------------- | -------------------- | ------------------------------ |
| **Document** (1 entity = 1 blob)          | Point lookup, scan all, prefix                      | `Get/Set/Delete`                | KV, SQL blob, Memory | `stack.Materialize[V,K]`       |
| **Relational** (1 entity = N joined rows) | Filtered lists, counts, pagination, junction tables | `Upsert/Update/Increment/Query` | SQL only             | `storage.RelationalProjection` |
| **Graph** (nodes + edges)                 | Variable-depth traversal, path-finding, adjacency   | `MergeNode/MergeEdge/Traverse`  | Graph DB only        | `graph.GraphProjection`        |

These cannot collapse into one universal interface:

- **Leaky collapse:** A "universal" interface that exposes SQL concepts (WHERE, ORDER BY) to KV
  backends
- **Anemic collapse:** A lowest-common-denominator interface (just `Get/Set`) that throws away
  SQL/graph power

The three-tier projection design is **architecturally correct.** The problem is not that there
are three paradigms — it's that **the consumer currently makes the storage choice (mechanism)
when the deployer should.**

### The Conflation

When a consumer writes:

```go
store, _ := sqlite.SQLViewModel[TaskView, TaskID](bundle, mapper)
```

...they make two decisions at once:

1. **Shape** (domain concern): "I have one view per task, and I want to filter/sort"
2. **Mechanism** (infrastructure concern): "Use SQLite columns"

Decision 1 belongs to the **consumer** (developer). Decision 2 belongs to the **deployer**
(operator). The current API forces the consumer to make both.

### Future Direction (After Path A): Declare + Bind

Once the 4 leaks are closed, the read-side API can be further refined so the consumer declares
shape + query intent and the deployer's Bundle determines mechanism:

```go
// CONSUMER declares shape + intent (engine-agnostic)
tasksRM := stack.Declare[TaskView, TaskID]("tasks").
    Key(taskKeyFromEvent).
    OnCreate(taskOnCreate).
    FilterBy("status", "assignee").
    SortBy("created_at").
    Done()

// DEPLOYER picks infrastructure
bundle, _ := stack.Deploy(loadConfig("deploy.yaml"))

// LIBRARY matches declaration -> best available store
mat, _ := stack.Bind(bundle, tasksRM)
```

This is **Path C** from the full design analysis. It is valuable but secondary to Path A.
**Build order:** Path A (close leaks) first. Path C (Declare/Bind) later if needed.

### The Bus Abstraction Has The Same Disease

The event bus has the same leak pattern as the read-side storage:

| Leak                                                            | Location                             | Problem                                         |
| --------------------------------------------------------------- | ------------------------------------ | ----------------------------------------------- |
| Signing middleware requires `*watermill.EventBus`               | `example/taskmanager/features.go:75` | Consumer imports the concrete Watermill type    |
| `bundle.CatchUpSubscriber()` hard-asserts `*watermill.EventBus` | `stack/bundle.go:182`                | Watermill is mandatory for catch-up projections |
| `EventBus.MessageSubscriber()` returns raw `message.Subscriber` | `watermill/event_bus.go:52`          | Watermill types escape the abstraction          |

The `event.Publisher`/`event.Subscriber` interfaces are clean. But the moment a consumer wants
**middleware** (signing, tracing, encryption) or **catch-up replay**, they must downcast to the
concrete `*watermill.EventBus`. Same pattern as the `*sql.DB` leak.

**Fix direction:** `bundle.Use(middleware)` should work without downcasting. The Bundle should
expose middleware installation on the interface, not require the concrete bus type. This is a
fifth leak to close alongside the four storage leaks.

---

## 10. Action Plan

### Phase 1: Close The 4 Leaks (THE critical path)

**Goal:** App code imports zero storage packages. This is the fix for "going down on the DB
layer."

| Task                                     | Files affected                                      | Description                                                                               | Effort |
| ---------------------------------------- | --------------------------------------------------- | ----------------------------------------------------------------------------------------- | ------ |
| Move connection tuning to presets        | `stack/sqlite/`, `stack/pebble/`, `stack/postgres/` | Add `WithMaxOpenConns`, `WithConnMaxLifetime`, etc. Remove/deprecate `bundle.Database()`  | 2h     |
| Neutralize ViewMapper types              | `storage/view/`, `stack/`                           | Replace `string` SQL types with `stack.ColumnType` enum. SQL adapter translates           | 3h     |
| Return interfaces from constructors      | `stack/sqlite/view_models.go`, `stack/postgres/`    | `SQLViewModel` returns `kv.ViewStore[V,K]`, not `*storage.SQLViewStore`                   | 2h     |
| Bundle method for relational projections | `stack/bundle.go` or `stack/sqlite/`                | `bundle.RelationalProjection(name, schema, handler, types)` provides `*sql.DB` internally | 2h     |

**Total: ~1 day. Risk: Low (additive changes and type widening).**

### Phase 2: Close The Bus Leak (5th leak)

| Task                                               | Files affected                  | Description                                             | Effort |
| -------------------------------------------------- | ------------------------------- | ------------------------------------------------------- | ------ |
| `bundle.Use(middleware)` without downcast          | `stack/bundle.go`, `watermill/` | Bundle exposes middleware installation on the interface | 3h     |
| `bundle.CatchUpSubscriber()` without concrete type | `stack/bundle.go`               | Return interface, not `*watermill.CatchUpSubscriber`    | 2h     |

**Total: ~5h.**

### Phase 3: Plugin System (optional convenience layer)

| Task                                    | Description                                                                              | Effort |
| --------------------------------------- | ---------------------------------------------------------------------------------------- | ------ |
| `Plugin` interface + registry           | `stack/plugin.go` with `Plugin`, `CapabilitySet`, `Partial`, `Register()`, `Available()` | 2h     |
| `Deploy()` + `Open()`                   | `stack/deploy.go` with config parsing + multi-plugin assembly                            | 2h     |
| `Bundle.ToPartial()` + `mergePartial()` | Extract Bundle fields into Partial; merge Partials into Bundle                           | 1h     |
| Register each preset                    | `stack/sqlite/plugin.go`, `stack/pebble/plugin.go`, `stack/memory/plugin.go`             | 2h     |
| Config loading helper                   | `stack.LoadConfig(path)` YAML decoder                                                    | 30m    |

**Total: ~1 day. Only after Phase 1 is complete.**

### Phase 4: Declare + Bind (optional read-side refinement)

| Task                                | Description                                                                     | Effort |
| ----------------------------------- | ------------------------------------------------------------------------------- | ------ |
| `ReadModelSpec[V,K]` type           | Struct with handler funcs + QueryHints                                          | 2h     |
| `stack.Declare[V,K]` fluent builder | Produces ReadModelSpec via `.Key()`, `.OnCreate()`, `.FilterBy()`, `.SortBy()`  | 3h     |
| `stack.Bind(bundle, spec)`          | Inspects Bundle capabilities, selects best store, returns projection.Projection | 4h     |
| Capability mismatch diagnostics     | BindWarning on degradation, typed errors on impossible matches                  | 2h     |

**Total: ~2 days. Only if Phase 1-3 prove insufficient.**

### Total Effort

| Phase                   | Effort  | Priority                |
| ----------------------- | ------- | ----------------------- |
| Phase 1: Close 4 leaks  | ~1 day  | **Critical — do first** |
| Phase 2: Close bus leak | ~5h     | High                    |
| Phase 3: Plugin system  | ~1 day  | Medium (convenience)    |
| Phase 4: Declare + Bind | ~2 days | Low (refinement)        |

### Verification

After Phase 1, verify with:

```bash
# The taskmanager example must compile with ZERO imports from:
#   database/sql, storage/, watermill/
grep -r "database/sql" example/taskmanager/ && echo "FAIL: still importing database/sql"
grep -r "storage/" example/taskmanager/ && echo "FAIL: still importing storage/"
```

The app code should only import: `decider/`, `event/`, `stack/`, `projectionhost/`, and
domain types.
