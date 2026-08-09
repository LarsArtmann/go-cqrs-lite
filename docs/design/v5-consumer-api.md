# v5 Consumer API: Declare Domain, Deploy Infrastructure

> **STATUS: DESIGN.** This document specifies the target API for the v5 major
> release. The current `system/` package implements an earlier iteration
> (`DomainConfig` with `Projections []any`, imperative `Commands func(*System)`
> callbacks, and separate `Aggregate`/`View`/`Decider` concepts). This document
> supersedes those. Implementation is in progress.

**Date:** 2026-08-09
**Status:** Design
**Related:** [ADR-0123](../adr/0123-v5-unification-single-composition-root.md), [ADR-0116](../adr/0116-layered-auto-projection.md), [ADR-0111](../adr/0111-record-type-extraction.md), [ADR-0112](../adr/0112-es-native-metaengine.md), [Domain Language](../DOMAIN_LANGUAGE.md), [Metaengine Domain Language](../METAENGINE_DOMAIN_LANGUAGE.md)

---

## Table of Contents

1. [The Vision](#1-the-vision)
2. [What's Wrong With the Current API](#2-whats-wrong-with-the-current-api)
3. [Core Design Principles](#3-core-design-principles)
4. [The Domain: What the Developer Declares](#4-the-domain-what-the-developer-declares)
5. [Queries: Access Patterns Drive Storage](#5-queries-access-patterns-drive-storage)
6. [QuerySet: Declare Once, Query Flexibly](#6-queryset-declare-once-query-flexibly)
7. [Commands: Logic Without Infrastructure](#7-commands-logic-without-infrastructure)
8. [How Commands Find Their State](#8-how-commands-find-their-state)
9. [The Deployment: Where Data Lives](#9-the-deployment-where-data-lives)
10. [The Sealed Type System](#10-the-sealed-type-system)
11. [Runtime API: Typed Reads](#11-runtime-api-typed-reads)
12. [Progressive Disclosure](#12-progressive-discipline)
13. [What Disappears](#13-what-disappears)
14. [The Taskmanager Transformation](#14-the-taskmanager-transformation)
15. [Open Questions](#15-open-questions)
16. [Alternatives Considered](#16-alternatives-considered)

---

## 1. The Vision

> "Developers declare ONLY Commands + Events + Queries and their relationships.
> We should be able to build superb projections (materialized views) and
> developers never need to worry about anything else, while where data lives is
> up to operators at DEPLOYMENT time."
>
> "If I only give you SQLite, metaengine should deal with all query projections
> via SQLite. If there are graph queries, it should warn about them being slow.
> At the same time I should be able to ONLY provide a GraphDB, even for event
> logs."
>
> "metaengine should be SMART enough to handle EVERYTHING so developers REALLY
> NEVER need to think about the storage layer!"

The developer writes pure Go types and business logic. The operator picks
databases. The system bridges the two: it reads the domain declarations, reads
the deployment engines, and auto-generates every fold, decoder, adapter, index,
and storage plan needed to make it work.

**The developer never imports a driver, names an engine, writes a decoder,
builds a projection adapter, or registers a decider.**

---

## 2. What's Wrong With the Current API

The current `system.DomainConfig` is called "domain" but 6 of its 10 fields are
infrastructure concerns:

```go
type DomainConfig struct {
    Commands              func(*System)                   // imperative callback, not declarative
    Queries               func(*System)                   // imperative callback, not declarative
    Projections           []any                           // type-erased: "oops a string" compiles
    ProjectionDecoder     func(...) (any, error)          // INFRASTRUCTURE — developer shouldn't know
    ProjectionTypeDecoder *TypeDecoder                   // INFRASTRUCTURE
    ProjectionEventDecoder EventDecoder                  // INFRASTRUCTURE
    Middleware            []command.Middleware            // OK — domain concern
    ProjectionHostOptions []HostOption                   // INFRASTRUCTURE — operator concern
    CheckpointStore       event.CheckpointStore           // INFRASTRUCTURE — operator concern
    ShutdownDependencies  []ShutdownDependency            // INFRASTRUCTURE — operator concern
}
```

Three fundamental problems:

### Problem 1: `Projections []any` is a type safety hole

Go generics are invariant: `QueryDecl[UserInput, UserResult]` is not assignable
to `QueryDecl[any, any]`. Since the slice holds heterogeneous query declarations
with different type parameters, the only escape valve was `[]any` + runtime
type-assertion. This means a stray string, a nil, or a typo compiles fine and
blows up at runtime.

### Problem 2: "Aggregate" and "View" are unnecessary concepts

The current API forces the developer to understand:

- **Aggregates** — a DDD concept that adds a registration step (`RegisterDecider`)
- **Views** — database jargon for "materialized projection"
- **Deciders** — pure-function folds that the developer writes by hand
- **Folds** — event-to-projection mapping functions
- **ADTs** — the data structure classification the planner uses
- **Decoders** — the payload-to-struct mapping for projection adapters

None of these are things the developer should care about. They are
implementation details of the system, not domain concepts.

### Problem 3: Infrastructure leaks into the domain config

`ProjectionHostOptions`, `CheckpointStore`, `ShutdownDependencies`, three
different decoder fields — these are all operator concerns sitting on the
"domain" struct. The developer has to understand them to configure the system.

---

## 3. Core Design Principles

| Principle                         | Meaning                                                                                                                         |
| --------------------------------- | ------------------------------------------------------------------------------------------------------------------------------- |
| **Declare, don't wire**           | The developer declares types and relationships. The system wires everything.                                                    |
| **Type safety, no `any`**         | Sealed interfaces compile-time-reject invalid declarations. Go generics carry types end-to-end.                                 |
| **Domain != Deployment**          | Two config structs, zero overlap. Developer never touches infrastructure. Operator never writes logic.                          |
| **Access pattern drives storage** | The developer declares HOW they read (point lookup vs filtered scan vs aggregate). The engine builds the optimal storage shape. |
| **Convention over configuration** | Field matching by name + type auto-generates folds for 80% of cases. Explicit folds for the rest.                               |
| **Everything works everywhere**   | Any engine serves any query. The planner warns, never blocks.                                                                   |

---

## 4. The Domain: What the Developer Declares

The developer declares exactly **two things**:

1. **Commands** — business logic: `func(ctx, cmd, state) → events`
2. **Queries** — read patterns: which events build this result, and how it's accessed

Plus optional **Middleware** (validation, authorization).

```go
type Domain struct {
    Commands   []CommandSpec    // sealed — handlers + stream routing
    Queries    []QuerySpec      // sealed — access patterns + event relationships
    Middleware []command.Middleware
}
```

No deciders. No aggregates. No projections. No decoders. No host options.

---

## 5. Queries: Access Patterns Drive Storage

A Query is not a "view." A Query is a declaration of **how the developer reads
data**, which tells the engine **what storage shape to build**.

Three access patterns, three constructors:

| Constructor           | Access Pattern                  | Engine Storage  | Read Cost        | Use When                       |
| --------------------- | ------------------------------- | --------------- | ---------------- | ------------------------------ |
| `Lookup[R]("name")`   | Single row by key               | Hash map / KV   | O(1)             | "Get user X"                   |
| `QuerySet[R]("name")` | Multi-row with flexible filters | Table + indexes | O(log N) indexed | "Find tasks WHERE status=open" |
| `Count("name")`       | Numeric aggregate               | Counter         | O(1)             | "How many tasks per status?"   |

### Why three constructors instead of one?

Because the engine needs to know the access pattern to pick the right data
structure:

- `Lookup` tells the engine: "build a hash map keyed by ID." O(1) reads.
- `QuerySet` tells the engine: "build a table with indexes on these fields."
  O(log N) filtered/sorted reads.
- `Count` tells the engine: "maintain a counter." O(1) aggregate reads.

If the developer declared a single "View" type, the engine wouldn't know
whether to optimize for point lookup or filtered scan. The access pattern IS
the information the engine needs.

### "Only name and email" = different result type

The developer doesn't ask for field projection at query time. They declare a
different query with a smaller result type:

```go
// Full struct — all fields stored
system.Lookup[TaskSummary]("get-task")...

// Only what you need — engine stores a leaner row
system.Lookup[TaskContact]("get-task-contact")
    .On("task.created", TaskCreated{},
        func(e TaskCreated, v *TaskContact) {
            v.ID = e.ID
            v.Title = e.Title
        }).
    Done()
```

Each query maintains its own independent projection. The engine stores exactly
the fields the result type needs. No wasted columns.

---

## 6. QuerySet: Declare Once, Query Flexibly

The problem with separate query declarations per filter combination:

```go
// WRONG — same data, same events, N declarations:
system.Lookup[TaskSummary]("get-task")...
system.Scan[TaskSummary]("tasks-by-status")...       // WHERE status=?
system.Scan[TaskSummary]("tasks-by-assignee")...     // WHERE assignee=?
system.Scan[TaskSummary]("tasks-by-status-priority") // WHERE status=? AND priority=?
```

The developer shouldn't enumerate every WHERE combination. `QuerySet` declares
one collection with flexible runtime access:

### Declaration

```go
system.QuerySet[TaskSummary]("tasks").
    On("task.created",   TaskCreated{}).
    On("task.completed", TaskCompleted{},
        func(e TaskCompleted, v *TaskSummary) { v.Status = "done" }).
    On("task.deleted",   TaskDeleted{}).
    // Declare which fields the engine should index.
    // The planner builds the right storage shape based on these.
    Filterable("status", "priority", "assignee").
    Sortable("priority", "created_at").
    Done(),
```

### Runtime: any combination of declared filters

```go
// Get one by ID (implicit primary key)
task, err := system.Get[TaskSummary](ctx, sys, "tasks", taskID)

// All tasks (paginated)
page1, err := system.Find[TaskSummary](ctx, sys, "tasks", system.Limit(50))
page2, err := system.Find[TaskSummary](ctx, sys, "tasks",
    system.Limit(50), system.After(page1.Cursor))

// Filter by status only
active, err := system.Find[TaskSummary](ctx, sys, "tasks",
    system.Where("status", "active"))

// Filter by status AND priority (optional filters, any combination)
hot, err := system.Find[TaskSummary](ctx, sys, "tasks",
    system.Where("status", "active"),
    system.Where("priority", "high"))

// No filters at all — give me everything
all, err := system.Find[TaskSummary](ctx, sys, "tasks")

// Filter + sort + paginate
mine, err := system.Find[TaskSummary](ctx, sys, "tasks",
    system.Where("assignee", "user-123"),
    system.OrderBy("priority", system.Desc),
    system.Limit(10))
```

### Why `Filterable(...)` is a build-time declaration

```go
.Filterable("status", "priority", "assignee")
```

This is not documentation. At `system.New()` time, the system:

1. **Validates** each field exists on `TaskSummary` (reflection — fail fast on typo)
2. **Generates indexes** on SQL engines (`CREATE INDEX idx_tasks_status ON ...`)
3. **Generates layout plans** for columnar engines (extract these columns natively)
4. **Enables pushdown** — runtime `Where("status", ...)` goes to SQL `WHERE`,
   not in-Go scan

If a runtime query uses a field NOT in `Filterable(...)`, the system falls back
to in-Go filtering (with a WARN diagnostic: "unindexed field 'foo' filtered in
Go").

### What the engine does

```
Developer declares:                     Engine builds:
  QuerySet[TaskSummary]("tasks")         → table/collection "tasks"
    .Filterable("status", "priority",      → indexes on status, priority, assignee
                "assignee")                 → (SQLite: B-tree indexes, DuckDB: zones)
    .Sortable("priority", "created_at")    → sort optimization hints

Runtime query:                          Engine executes:
  Where("status", "active")             → SELECT ... WHERE status = 'active' (index seek)
  Where("priority", "high")             →   AND priority = 'high' (composite index)
  OrderBy("priority", Desc)             → ORDER BY priority DESC
  Limit(10)                             → LIMIT 10
```

On SQL engines (SQLite, Postgres, DuckDB): WHERE/ORDER BY/LIMIT pushed down to
SQL. On Memory engine: in-Go filter + sort. The planner picks the strategy
based on engine capabilities.

---

## 7. Commands: Logic Without Infrastructure

Commands are pure business logic. Each handler receives:

- `context.Context`
- The typed command struct (embeds `*command.BasicCommand` for `StreamID()`)
- The current state (loaded by replaying events through the matching query fold)

Returns events to emit.

```go
system.On("task.complete",
    func(ctx context.Context, cmd CompleteTaskCmd, state TaskSummary) ([]system.Emission, error) {
        if state.ID == "" {
            return nil, errors.New("task not found")
        }
        if state.Status == "done" {
            return nil, errors.New("already completed")
        }
        return system.Emit(
            system.Event("task.completed", TaskCompleted{ID: cmd.StreamID().String()}),
        ), nil
    }),
```

### The State type is inferred from the handler signature

The compiler infers `State = TaskSummary` from the third parameter. The system
finds the query whose result type matches `TaskSummary` and uses its fold for
state loading. No explicit wiring.

### No aggregates, no deciders, no repositories

The developer never writes:

- `decider.Decider[State]{Initial: ..., Fold: ...}`
- `RegisterDecider[State](sys, "Task", decider)`
- `decider.NewRepository(store, bus, decider)`

The system auto-builds all of these from the query fold + command handler
signatures.

---

## 8. How Commands Find Their State

The key insight: **the fold declared on a Query IS the state fold**. The same
function that builds the read model builds the command state.

```
Command handler declares:    func(..., state TaskSummary) → State = TaskSummary
Query declares:              QuerySet[TaskSummary]("tasks") → Result = TaskSummary
                                            ↑ same type ↑

System auto-connects:
  1. Command arrives → system loads stream events from event store
  2. Events replayed through the TaskSummary fold (from the "tasks" query)
  3. State passed to command handler
  4. Handler returns events → system appends to event store
  5. Events published to projection host → query projections updated
```

### When State != Public Query (Internal State)

If a command needs state that doesn't belong in a public query (e.g., "was this
task ever assigned?"), the developer declares an internal query:

```go
system.Lookup[TaskInternalState]("task-internal", system.Internal()).
    On("task.created", TaskCreated{},
        func(e TaskCreated, v *TaskInternalState) { v.Exists = true }).
    On("task.assigned", TaskAssigned{},
        func(e TaskAssigned, v *TaskInternalState) { v.EverAssigned = true }).
    Done(),
```

`system.Internal()` marks the query as state-only — not queryable via `Get` or
`Find`, but available for command state loading. Same fold mechanism, just not
exposed as a read endpoint.

### When multiple queries share the same result type

If both `Lookup[TaskSummary]("get-task")` and `QuerySet[TaskSummary]("tasks")`
exist, the system prefers the `Lookup` fold for state loading (cheapest for
single-stream replay).

---

## 9. The Deployment: Where Data Lives

The operator picks databases. The developer never touches this.

### Level 1: Single-engine shortcuts (90% of cases)

```go
sys, err := system.New(ctx, domain, system.SQLite("app.db"))
sys, err := system.New(ctx, domain, system.Memory())
sys, err := system.New(ctx, domain, system.Pebble("/data/db"))
sys, err := system.New(ctx, domain, system.Postgres(dsn))
sys, err := system.New(ctx, domain, system.DuckDB("analytics.db"))
```

Everything in one file. The system auto-wires event store, command log, query
log, projections, snapshots, and bus.

### Level 2: Multi-engine topology

```go
system.Deployment{
    Engines: system.Engines(
        system.Named("source",    system.SQLite("events.db")),
        system.Named("views",     system.SQLite("views.db")),
        system.Named("analytics", system.DuckDB("analytics.db")),
    ),
    Topology: system.Topology{
        SourceOfTruth: "source",
        Projections:   []string{"views", "analytics"},
    },
}
```

The planner routes each query to its engine and warns if the engine doesn't
natively support the query's access pattern (e.g., Counter on a Graph engine).

### Level 3: Full operational config

```go
system.Deployment{
    Engines:    ...,
    Topology:   ...,
    Bus:        system.Bus{Driver: "nats", URL: natsURL},
    Durability: system.DurabilityStrict,
    ProjectionHost: system.ProjectionHost{
        BatchSize:     500,
        DeadLetterMax: 3,
    },
    ManifestPath: "/var/lib/myapp/plan.json",
}
```

`ProjectionHost`, `CheckpointStore`, `ShutdownDependencies` all live here, not
on `Domain`.

---

## 10. The Sealed Type System

No `[]any` anywhere. Every slice is a sealed interface:

```go
// Sealed interfaces — only constructors in the system package can satisfy these.
type CommandSpec interface { isCommandSpec() }
type QuerySpec    interface { isQuerySpec() }

// Internal implementations (unexported):
type commandSpec struct{ ... }; func (commandSpec) isCommandSpec() {}
type lookupSpec   struct{ ... }; func (lookupSpec)   isQuerySpec() {}
type querySetSpec struct{ ... }; func (querySetSpec) isQuerySpec() {}
type countSpec    struct{ ... }; func (countSpec)    isQuerySpec() {}
```

A stray string, int, or nil in `Domain.Queries` is a **compile error**, not a
runtime panic:

```go
system.Domain{
    Queries: []system.QuerySpec{
        system.Lookup[TaskSummary]("get-task")...,
        "oops a string", // COMPILE ERROR: string does not implement QuerySpec
    },
}
```

### Generic methods for type inference

Go generics allow methods with additional type parameters. This enables
per-event type inference on the query builder:

```go
// viewBuilder[R] carries the result type. On[E] introduces the event type.
type querySetBuilder[R any] struct { ... }

func QuerySet[R any](name string) *querySetBuilder[R]

// E is inferred from the sample argument. R is already known from the builder.
// The fold func type func(E, *R) is checked at compile time.
func (b *querySetBuilder[R]) On[E any](
    eventType string,
    sample E,
    fold ...func(E, *R),  // optional: 0 = auto-classify by convention, 1 = explicit
) *querySetBuilder[R]
```

**Compile-time inference proof:**

```go
system.QuerySet[TaskSummary]("tasks").
    On("task.created", TaskCreated{})
//  ↑ R=TaskSummary (fixed)   ↑ E=TaskCreated (inferred from sample)

    .On("task.started", TaskStarted{}, func(e TaskStarted, v *TaskSummary) {
        v.Status = "active"
    })
//  ↑ E=TaskStarted (inferred)  ↑ checked: func(TaskStarted, *TaskSummary) matches func(E, *R) ✓
```

---

## 11. Runtime API: Typed Reads

All reads are top-level generic functions, not methods on `*System` (Go can't
infer type params from method arguments alone):

```go
// Point lookup (from Lookup query)
task, err := system.Get[TaskSummary](ctx, sys, "get-task", taskID)

// Filtered scan (from QuerySet query)
board, err := system.Find[TaskSummary](ctx, sys, "tasks",
    system.Where("status", "open"),
    system.OrderBy("priority", system.Desc),
    system.Limit(20),
)

// Counter (from Count query)
counts, err := system.Get[map[string]int64](ctx, sys, "task-counts")
// counts = {"pending": 42, "active": 17, "done": 103}

// Pagination
page2, err := system.Find[TaskSummary](ctx, sys, "tasks",
    system.After(page1.Cursor),
    system.Limit(50),
)
```

---

## 12. Progressive Disclosure

Three levels of fold declaration, each adding power. All return the same sealed
`QuerySpec`.

### Level 1: Convention (80% of cases)

Field matching by name + type. Event struct suffix (`Created`/`Updated`/
`Deleted`) classifies the fold kind. No hand-written fold function.

```go
system.Lookup[TaskSummary]("get-task").
    On("task.created", TaskCreated{}).    // Created suffix → insert
    On("task.deleted", TaskDeleted{}).    // Deleted suffix → remove
    Done()
// System auto-generates: insertFold (Title, Priority from TaskCreated fields)
```

### Level 2: Explicit fold (15% of cases)

For computed/derived fields that don't exist on the event struct:

```go
system.Lookup[TaskSummary]("get-task").
    On("task.created", TaskCreated{}).
    On("task.started", TaskStarted{},
        func(e TaskStarted, v *TaskSummary) { v.Status = "active" }).
    Done()
// System uses explicit fold for task.started, convention for the rest
```

### Level 3: Custom query (5% of cases)

For non-CRUD shapes (search, graph, vector) or fully manual folds:

```go
system.Custom[SearchInput, SearchResult]("task-search",
    metaengine.OnTyped("task.created", TaskCreated{},
        func(e TaskCreated) (string, SearchResult) {
            return e.ID, SearchResult{ID: e.ID, Title: e.Title, Score: relevance(e)}
        }),
    metaengine.Volume(10_000),
)
```

Returns the same sealed `QuerySpec`. No `[]any` leak.

---

## 13. What Disappears

| Concept                  | Current API                                               | v5 API                                                                         |
| ------------------------ | --------------------------------------------------------- | ------------------------------------------------------------------------------ |
| **Aggregate**            | `RegisterDecider[State](sys, "Task", decider)`            | **Deleted.** System auto-builds from Query fold + Command handler.             |
| **State type**           | Separate `TaskState` struct                               | **Unified.** Query result type IS the state type.                              |
| **View**                 | `system.View[R,K]("name").From(events...)`                | **Renamed to Query.** `Lookup`, `QuerySet`, `Count`.                           |
| **Decider**              | `decider.Decider[State]{Initial, Fold}`                   | **Auto-generated.** System builds from Query fold.                             |
| **Fold**                 | `metaengine.On(...)` / `OnTyped(...)`                     | **Auto-generated** for L1/L2. Explicit only for L3.                            |
| **ADT**                  | Manual classification                                     | **Inferred** from constructor (Lookup→Map, QuerySet→SortedMap, Count→Counter). |
| **Decoder**              | `ProjectionTypeDecoder`, `EventDecoder`, `PayloadDecoder` | **Auto-generated** from event struct types.                                    |
| **Projection adapter**   | `projectionadapter.New(name, store, decoder)`             | **Auto-wired.**                                                                |
| **Engine selection**     | Manual `metaengine.Query` + `Plan`                        | **Auto-planned** by metaengine.                                                |
| **Projection host**      | `projectionhost.New(journal, cpStore, opts...)`           | **Auto-created** from Deployment config.                                       |
| **Bus**                  | `watermill.NewEventBus()`                                 | **Auto-wired** from Deployment.Bus.                                            |
| `Projections []any`      | Type-erased slice                                         | `Queries []QuerySpec` — sealed interface.                                      |
| `Commands func(*System)` | Imperative callback                                       | `Commands []CommandSpec` — declarative.                                        |

---

## 14. The Taskmanager Transformation

### Before: 199 LOC (metaengine.go) + 60 LOC (handlers.go) + decoder setup

```go
// 11 typed folds with manual EventWithID wrapping + 11-line decoder
func buildProjections() ([]any, *projectionadapter.TypeDecoder) {
    taskCounts := metaengine.Query[taskCountsInput, map[string]int64](
        "task_counts_by_status",
        metaengine.OnTyped(string(evtTaskCreated),
            projectionadapter.EventWithID[TaskCreatedPayload]{},
            func(_ projectionadapter.EventWithID[TaskCreatedPayload]) metaengine.Delta {
                return metaengine.Delta{string(StatusPending): 1}
            }),
        // ... 10 more folds ...
    )
    taskViews := metaengine.Query[listTasksInput, TaskView](
        "task_views",
        metaengine.OnTyped(string(evtTaskCreated), ...),
        // ... 10 more folds ...
        metaengine.FilterOnField[TaskView]("status", metaengine.FilterEq),
        metaengine.SortOnField[TaskView]("priority", true),
    )
    decoder := projectionadapter.NewTypeDecoder(
        projectionadapter.Register(evtTaskCreated, TaskCreatedPayload{}),
        // ... 10 more registrations ...
    )
    return []any{taskCounts, taskViews}, decoder
}
```

### After: ~60 LOC total

```go
sys, err := system.New(ctx,
    system.Domain{
        Commands: system.Commands(
            system.On("task.create",
                func(ctx context.Context, cmd CreateTaskCmd, _ TaskSummary) ([]system.Emission, error) {
                    return system.Emit(system.Event("task.created", TaskCreated{
                        ID: cmd.StreamID().String(), Title: cmd.Title, Priority: cmd.Priority,
                    })), nil
                }),
            system.On("task.complete",
                func(ctx context.Context, cmd CompleteTaskCmd, state TaskSummary) ([]system.Emission, error) {
                    if state.Status != "active" {
                        return nil, errors.New("not active")
                    }
                    return system.Emit(system.Event("task.completed", TaskCompleted{
                        ID: cmd.StreamID().String(),
                    })), nil
                }),
            system.On("task.delete",
                func(ctx context.Context, cmd DeleteTaskCmd, _ TaskSummary) ([]system.Emission, error) {
                    return system.Emit(system.Event("task.deleted", TaskDeleted{
                        ID: cmd.StreamID().String(),
                    })), nil
                }),
        ),

        Queries: system.Queries(
            system.QuerySet[TaskSummary]("tasks").
                On("task.created",   TaskCreated{}).
                On("task.started",   TaskStarted{},
                    func(e TaskStarted, v *TaskSummary) { v.Status = "active" }).
                On("task.completed", TaskCompleted{},
                    func(e TaskCompleted, v *TaskSummary) { v.Status = "done" }).
                On("task.deleted",   TaskDeleted{}).
                Filterable("status", "priority", "assignee").
                Sortable("priority", "created_at").
                Done(),

            system.Count("task-counts").
                On("task.created",   +1, "pending").
                On("task.started",   +1, "active").
                On("task.started",   -1, "pending").
                On("task.completed", +1, "done").
                On("task.completed", -1, "active").
                On("task.deleted",   -1, "done").
                Done(),
        ),
    },
    system.SQLite("app.db"),
)

sys.Start(ctx)
defer sys.Close()
```

No folds. No decoder. No `EventWithID` wrapper. No `[]any`. No aggregate. No
decider. No projection adapter.

---

## 15. Open Questions

### Q1: Fold classification beyond Created/Updated/Deleted

Events like `TaskStarted`, `TaskAssigned`, `TaskArchived` don't match the
naming convention. Three options:

| Option                           | Mechanism                                                                          | Pro                           | Con                                 |
| -------------------------------- | ---------------------------------------------------------------------------------- | ----------------------------- | ----------------------------------- |
| **A. Field-matching**            | If event has matching fields → update (patch); if only key → delete; else → insert | Zero hints needed             | Magical, harder to debug edge cases |
| **B. `.As(system.Update)` hint** | `On("task.started", TaskStarted{}, system.As(system.Update))`                      | Explicit, minimal API surface | Developer must know fold kinds      |
| **C. Always explicit fold**      | Every event beyond Created/Deleted requires a fold func                            | Fully explicit                | Verbose for simple cases            |

**Current recommendation:** Field-matching (A) as default, with `.As(...)`
override for ambiguous cases. The existing `matchFields` in `auto_naming.go`
already classifies by field overlap. If a field overlaps → update (patch). If
only the key field → remove. Otherwise → insert.

### Q2: Stream type derivation

Commands need a stream type string (e.g., `"Task"`) for the event store. Three
options:

| Option                       | Mechanism                                               | Pro                       | Con                       |
| ---------------------------- | ------------------------------------------------------- | ------------------------- | ------------------------- |
| **A. From state type name**  | `TaskSummary` → strip suffix → `"Task"`                 | Zero config               | Fragile naming convention |
| **B. Explicit on On()**      | `system.On("Task", "task.create", handler)`             | Explicit, no surprise     | Slightly more verbose     |
| **C. System.Stream wrapper** | `system.Stream("Task", system.On(...), system.On(...))` | Groups commands by stream | Adds nesting              |

**Current recommendation:** Option B — explicit stream type on the command
handler or a `Stream` grouping function. Naming conventions for stream
derivation (A) are too fragile for a library.

### Q3: Multiple result types from the same events

When two queries fold the same events into different result types (e.g.,
`TaskSummary` and `TaskContact`), each maintains its own independent projection.
This is already how the metaengine works ("each query has its own independent
projection"). The only cost is write amplification — each event updates N
projections. The planner already warns when this exceeds the budget
(`DefaultWriteAmplificationBudget = 3`).

### Q4: Command state loading performance

Commands load state by replaying events through the query fold. For hot streams
with long histories, this can be expensive. The existing `decider.Repository`
already solves this via:

- **Snapshots** — point-in-time state capture (`snapshot.EveryNEvents`)
- **Hot-state cache** — LRU-bounded in-memory cache (`decider.WithStateCache`)
- **Load coalescing** — `singleflight.Group` for concurrent loads

These continue to work. The system auto-wires snapshots when the engine
implements `SnapshotBackend`. The developer doesn't configure these — they're
deployment-time concerns.

---

## 16. Alternatives Considered

### A. Keep `View` terminology

**Rejected.** "View" is database jargon. The developer thinks in terms of
queries: "get me this task" or "find tasks by status." The API should match the
mental model. Additionally, the metaengine already uses `QueryDecl[Q,R]` and
`ExecuteTyped[Q,R]` — the internal vocabulary is already "query."

### B. Single `Query` constructor with access-pattern options

```go
// Rejected: too much flexibility, engine can't optimize upfront
system.Query[TaskSummary]("tasks").
    AccessPattern(system.PointLookup).  // or FilteredScan, or Aggregate
    ...
```

**Rejected.** The access pattern determines the ADT, storage shape, and index
strategy. Making it a builder option hides the one thing the engine needs to
know at plan time. Three named constructors (`Lookup`, `QuerySet`, `Count`)
are more honest — they make the access pattern a first-class choice.

### C. Declarative struct tags instead of builder API

```go
// Rejected: too magical, poor IDE support
type TaskSummary struct {
    ID     string `cqrs:"key"`
    Status string `cqrs:"filterable"`
    Priority int  `cqrs:"sortable"`
}
```

**Rejected.** Struct tags don't carry event relationships (which events build
this struct?) and don't support explicit fold functions for derived fields.
The builder API is more powerful and has better IDE autocomplete.

### D. Code generation (cqrs-gen) as the primary path

**Rejected as sole path.** Code generation can't adapt at runtime to available
engines. The planner's cost-based routing must happen at startup when engines
are known. Code generation is complementary (compile-time type checking,
pre-computed folds), not the primary mechanism.

### E. Keep `func(*System)` callback for commands

**Rejected.** The callback is imperative: the developer calls
`RegisterDecider`, `RegisterCommand`, `RegisterQuery` inside it. This means
errors surface at callback time, not construction time. A declarative
`[]CommandSpec` slice enables upfront validation and better error messages.
