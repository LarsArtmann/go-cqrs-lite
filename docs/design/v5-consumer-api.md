# v5 Consumer API: Declare Domain, Deploy Infrastructure

> **STATUS: DESIGN.** This document specifies the target API for the v5 major
> release. The current `system/` package implements an earlier iteration
> (`DomainConfig` with `Projections []any`, imperative `Commands func(*System)`
> callbacks, and separate `Aggregate`/`View`/`Decider` concepts). This document
> supersedes those. Implementation is in progress.

**Date:** 2026-08-09
**Status:** Design
**Related:** [ADR-0001](../adr/0001-decider-over-aggregate.md), [ADR-0058](../adr/0058-rename-aggregate-to-stream.md), [ADR-0116](../adr/0116-layered-auto-projection.md), [ADR-0123](../adr/0123-v5-unification-single-composition-root.md), [Aggregate Concept Analysis](../architecture-understanding/AGGREGATE-CONCEPT-ANALYSIS.md), [Domain Language](../DOMAIN_LANGUAGE.md), [Metaengine Domain Language](../METAENGINE_DOMAIN_LANGUAGE.md)

---

## Table of Contents

1. [The Vision](#1-the-vision)
2. [What's Wrong With the Current API](#2-whats-wrong-with-the-current-api)
3. [Aggregateless by Design](#3-aggregateless-by-design)
4. [Core Design Principles](#4-core-design-principles)
5. [The Three-Way Split: Commands + Evolutions + Queries](#5-the-three-way-split-commands--evolutions--queries)
6. [Evolutions: How State Emerges From Events](#6-evolutions-how-state-emerges-from-events)
7. [Queries: Access Patterns Drive Storage](#7-queries-access-patterns-drive-storage)
8. [QuerySet: Declare Once, Query Flexibly](#8-queryset-declare-once-query-flexibly)
9. [Graph-Native Queries: Typed, No map\[string\]any](#9-graph-native-queries-typed-no-mapstringany)
10. [Commands: Logic Without Infrastructure](#10-commands-logic-without-infrastructure)
11. [How Commands Find Their State](#11-how-commands-find-their-state)
12. [The Deployment: Where Data Lives](#12-the-deployment-where-data-lives)
13. [The Sealed Type System](#13-the-sealed-type-system)
14. [Runtime API: Typed Reads](#14-runtime-api-typed-reads)
15. [Progressive Disclosure](#15-progressive-disclosure)
16. [What Disappears](#16-what-disappears)
17. [The Taskmanager Transformation](#17-the-taskmanager-transformation)
18. [What to Learn from Akka](#18-what-to-learn-from-akka)
19. [Open Questions](#19-open-questions)
20. [Alternatives Considered](#20-alternatives-considered)

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
type-assertion. A stray string, a nil, or a typo compiles fine and blows up at
runtime.

### Problem 2: "Aggregate" and "View" are unnecessary concepts

The current API forces the developer to understand aggregates, deciders,
folds, ADTs, and decoders. None of these are domain concepts — they are
implementation details of the storage system.

### Problem 3: Infrastructure leaks into the domain config

`ProjectionHostOptions`, `CheckpointStore`, `ShutdownDependencies`, three
different decoder fields — these are all operator concerns sitting on the
"domain" struct.

---

## 3. Aggregateless by Design

This API is the logical conclusion of work that started with
[ADR-0001](../adr/0001-decider-over-aggregate.md) (killing the OO aggregate),
continued with [ADR-0058](../adr/0058-rename-aggregate-to-stream.md) (renaming
all Aggregate\* types to Stream\*), and is documented extensively in
[AGGREGATE-CONCEPT-ANALYSIS.md](../architecture-understanding/AGGREGATE-CONCEPT-ANALYSIS.md).

### What the research found

The project researched aggregateless event sourcing (Rico Fritzsche 2025, Sara
Pellegrini 2023, Ralf Westphal) across three documents in `docs/research/`.
The conclusion: **the library is already 80% aggregateless.** ADR-0001 killed
the OO aggregate, replaced it with pure functions, and never stores state.

The only thing retained from traditional DDD is the **stream partition key**
(`StreamRef`) — kept for operational reasons: optimistic concurrency, loading,
snapshots. The aggregateless critique's core insight — **feature slices** — is
exactly what this API adopts:

| Aspect               | Aggregate Hydration       | Feature Slice (this API)                |
| -------------------- | ------------------------- | --------------------------------------- |
| State shape          | Fixed (one per aggregate) | Flexible (one per result type)          |
| Events loaded        | All events in stream      | Only relevant event types               |
| Unused fields        | Hydrated but ignored      | Not loaded at all                       |
| Multiple consumers   | All see same state        | Each sees tailored state                |
| Consistency boundary | The aggregate             | The stream (via optimistic concurrency) |

### What this API rejects from aggregateless ES

| Aggregateless idea                    | Why rejected                                       |
| ------------------------------------- | -------------------------------------------------- |
| Single flat events table (no streams) | Lose optimistic concurrency, versioning, snapshots |
| Raw JSONB (no typed events)           | Lose compile-time safety                           |
| No snapshotting                       | Operational regression                             |

### Evolutions are NOT aggregates

`Evolve[TaskSummary]` is a **feature slice** — "how does this specific read
model evolve from events?" It is:

- **Not a consistency boundary** — multiple Evolutions can fold the same events
- **Not identity** — the stream ID is on the command, not the Evolution
- **Per-result-type** — `TaskSummary` and `TaskContact` have separate Evolutions
- **Disposable** — drop the Evolution, replay from events, get a new shape

The word "Evolve" avoids the DDD baggage of "Aggregate" while describing exactly
what's happening: events evolve a materialized state.

---

## 4. Core Design Principles

| Principle                          | Meaning                                                                                                                             |
| ---------------------------------- | ----------------------------------------------------------------------------------------------------------------------------------- |
| **Declare, don't wire**            | The developer declares types and relationships. The system wires everything.                                                        |
| **Type safety, no `any`**          | Sealed interfaces compile-time-reject invalid declarations. Go generics carry types end-to-end. No `map[string]any` in domain code. |
| **Domain != Deployment**           | Two config structs, zero overlap. Developer never touches infrastructure. Operator never writes logic.                              |
| **Access pattern drives storage**  | The developer declares HOW they read (point lookup vs filtered scan vs traversal). The engine builds the optimal storage shape.     |
| **Convention over configuration**  | Field matching by name + type auto-generates folds for 80% of cases. Explicit folds for the rest.                                   |
| **Everything works everywhere**    | Any engine serves any query. The planner warns, never blocks.                                                                       |
| **Feature slices, not aggregates** | Each result type is an independent fold over events. No god-object state.                                                           |

---

## 5. The Three-Way Split: Commands + Evolutions + Queries

The developer declares **three things** and nothing else:

1. **Commands** — business logic: `func(ctx, cmd, state) → events`
2. **Evolutions** — materialization rules: "when event E happens, update state S like this"
3. **Queries** — access patterns: "I want to read TaskSummary, filtered by status"

Plus optional **Middleware** (validation, authorization).

```go
type Domain struct {
    Commands   []CommandSpec    // sealed — handlers + stream routing
    Evolutions []EvolutionSpec  // sealed — event-to-state folds
    Queries    []QuerySpec      // sealed — access patterns
    Middleware []command.Middleware
}
```

No deciders. No aggregates. No projections. No decoders. No host options.

### Why three instead of two?

The fold (materialization rule) was originally placed on the Query. But the
fold IS domain logic — "completed tasks have status 'done'" is a business rule.
Putting it on "how you read data" mixes concerns.

The separation:

```
Evolution:  "When TaskCompleted happens, TaskSummary.Status = 'done'"
              ↑ materialization rule (domain logic, declared once per result type)

Query:      "I want to read TaskSummary, filtered by status"
              ↑ access pattern (read configuration, can be shared across multiple queries)

Command:    "Can this task be completed? Yes → emit TaskCompleted"
              ↑ business decision (validation + event emission)
```

The Evolution fold is declared ONCE per result type. All queries and commands
with that result type reuse it. The developer writes the rule once, and the
system uses it for both state loading (command side) and projection building
(query side).

### The connection

```
Evolve[TaskSummary](...)              ← the fold (materialization rule)
        ↑
        ├── QuerySet[TaskSummary]("tasks")    ← access pattern (read config)
        ├── Lookup[TaskSummary]("get-task")   ← access pattern (read config)
        └── On("task.complete", ..., TaskSummary)  ← command (write logic)
```

The system matches by type parameter. `Evolve[TaskSummary]` provides the fold.
All queries and commands with `TaskSummary` reuse it. One declaration, three
consumers.

---

## 6. Evolutions: How State Emerges From Events

An Evolution declares how a specific result type is built from events. It IS
the fold — the materialization rule.

### Level 1: Convention (80% of cases)

Field matching by name + type. Event struct suffix (`Created`/`Updated`/
`Deleted`) classifies the fold kind:

```go
system.Evolve[TaskSummary]("tasks").
    On("task.created", TaskCreated{}).      // Created suffix → insert
    On("task.deleted", TaskDeleted{}).      // Deleted suffix → remove
    Done()
// System auto-generates: insertFold (Title, Priority from TaskCreated fields)
```

### Level 2: Explicit fold (15% of cases)

For computed/derived fields:

```go
system.Evolve[TaskSummary]("tasks").
    On("task.created", TaskCreated{}).
    On("task.started", TaskStarted{},
        func(e TaskStarted, v *TaskSummary) { v.Status = "active" }).
    On("task.completed", TaskCompleted{},
        func(e TaskCompleted, v *TaskSummary) { v.Status = "done" }).
    On("task.deleted", TaskDeleted{}).
    Done()
```

The fold function `func(E, *R)` is type-checked at compile time. `E` is inferred
from the sample struct, `R` is fixed by the Evolution's type parameter.

### Counter query folds

Counter folds declare numeric deltas per event:

```go
system.Evolve[map[string]int64]("task-counts").
    On("task.created",   +1, "pending").
    On("task.started",   +1, "active").
    On("task.started",   -1, "pending").
    On("task.completed", +1, "done").
    On("task.completed", -1, "active").
    On("task.deleted",   -1, "done").
    Done()
```

The system sees the `map[string]int64` result type and classifies this as a
Counter ADT — not a Map.

### Multiple result types from the same events

When two Evolutions fold the same events into different result types (e.g.,
`TaskSummary` and `TaskContact`), each maintains its own independent projection.
Each result type stores only the fields it needs. No wasted columns.

---

## 7. Queries: Access Patterns Drive Storage

A Query declares **how the developer reads data**. It tells the engine **what
storage shape to build**. It does NOT contain domain logic — that lives on
the Evolution.

Three access patterns, three constructors:

| Constructor            | Access Pattern                  | Engine Storage  | Read Cost        | Use When                       |
| ---------------------- | ------------------------------- | --------------- | ---------------- | ------------------------------ |
| `Lookup[R]("name")`    | Single row by key               | Hash map / KV   | O(1)             | "Get user X"                   |
| `QuerySet[R]("name")`  | Multi-row with flexible filters | Table + indexes | O(log N) indexed | "Find tasks WHERE status=open" |
| `Count("name")`        | Numeric aggregate               | Counter         | O(1)             | "How many tasks per status?"   |
| `Traversal[R]("name")` | Graph neighbor traversal        | Graph adjacency | O(degree^depth)  | "Who follows this user?"       |

### Why access-pattern constructors?

Because the engine needs to know the access pattern to pick the right data
structure:

- `Lookup` tells the engine: "build a hash map keyed by ID." O(1) reads.
- `QuerySet` tells the engine: "build a table with indexes on these fields."
  O(log N) filtered/sorted reads.
- `Count` tells the engine: "maintain a counter." O(1) aggregate reads.
- `Traversal` tells the engine: "build a graph adjacency list." O(degree^depth) traversal.

If the developer declared a single "View" type, the engine wouldn't know whether
to optimize for point lookup or filtered scan. The access pattern IS the
information the engine needs.

### "Only name and email" = different result type

The developer declares a different Evolution + Query pair with a smaller result
type. The engine stores exactly the fields that result type needs:

```go
// Full struct — all fields stored
system.Evolve[TaskSummary]("tasks")...
system.Lookup[TaskSummary]("get-task").Done()

// Only what you need — engine stores a leaner row
system.Evolve[TaskContact]("task-contacts")
    .On("task.created", TaskCreated{},
        func(e TaskCreated, v *TaskContact) {
            v.ID = e.ID
            v.Title = e.Title
        }).
    Done()
system.Lookup[TaskContact]("get-task-contact").Done()
```

---

## 8. QuerySet: Declare Once, Query Flexibly

The problem with separate query declarations per filter combination:

```go
// WRONG — same data, same events, N declarations:
system.Lookup[TaskSummary]("get-task")...
system.Scan[TaskSummary]("tasks-by-status")...       // WHERE status=?
system.Scan[TaskSummary]("tasks-by-assignee")...     // WHERE assignee=?
system.Scan[TaskSummary]("tasks-by-status-priority") // WHERE status=? AND priority=?
```

The developer shouldn't enumerate every WHERE combination. `QuerySet` declares
one collection with flexible runtime access.

### Declaration

```go
system.QuerySet[TaskSummary]("tasks").
    Filterable("status", "priority", "assignee").
    Sortable("priority", "created_at").
    Done(),
```

Note: no `.On(...)` calls on the Query. The fold comes from the matching
`Evolve[TaskSummary]`. The Query is pure access-pattern declaration.

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
to in-Go filtering (with a WARN diagnostic).

### What the engine does

```
Developer declares:                     Engine builds:
  Evolve[TaskSummary]("tasks")           → fold events into TaskSummary rows
  QuerySet[TaskSummary]("tasks")         → table/collection "tasks"
    .Filterable("status", "priority",      → indexes on status, priority, assignee
                "assignee")
    .Sortable("priority", "created_at")    → sort optimization hints

Runtime query:                          Engine executes:
  Where("status", "active")             → SELECT ... WHERE status = 'active' (index seek)
  Where("priority", "high")             →   AND priority = 'high' (composite index)
  OrderBy("priority", Desc)             → ORDER BY priority DESC
  Limit(10)                             → LIMIT 10
```

On SQL engines (SQLite, Postgres, DuckDB): WHERE/ORDER BY/LIMIT pushed down to
SQL. On Memory engine: in-Go filter + sort.

---

## 9. Graph-Native Queries: Typed, No map\[string\]any

Graph queries use the **exact same typed struct pattern** as CRUD. No string
labels, no `map[string]any`, no `graph.Graph` type the developer manipulates.
The developer writes pure Go structs and the system detects the graph shape.

### The developer's types

```go
// ── Pure Go structs — the developer's domain types ──

type User struct {
    ID     string
    Name   string
    Joined time.Time
}

type Follow struct {
    From   string // references User.ID
    To     string // references User.ID
    Since  time.Time
}
```

`Follow` has `From`/`To` string fields. The system inspects the struct shape and
classifies it as an **edge type**. `User` has no `From`/`To`, so it's a **node
type**.

### Evolutions: same API as CRUD

```go
system.Evolutions(

    // Node type — same as any CRUD entity
    system.Evolve[User]("users").
        On("user.registered", UserRegistered{},
            func(e UserRegistered, u *User) {
                u.ID = e.UserID
                u.Name = e.Name
                u.Joined = e.At
            }).
        Done(),

    // Edge type — system detects From/To fields → Graph ADT
    system.Evolve[Follow]("follows").
        On("user.followed", UserFollowed{},
            func(e UserFollowed, f *Follow) {
                f.From = e.FollowerID
                f.To = e.FolloweeID
                f.Since = e.At
            }).
        On("user.blocked", UserBlocked{},
            func(e UserBlocked, f *Follow) {
                // Remove signal — system removes the edge matching From/To
            }).
        Done(),
),
```

If the developer wants to be explicit about which fields are endpoints (instead
of relying on `From`/`To` convention):

```go
system.Evolve[Follow]("follows", system.Edge("From", "To"))
```

### Queries: graph-native access patterns

```go
system.Queries(

    // "Who does this user follow?" — 1-hop traversal
    system.Traversal[User]("following").
        From("users").            // node collection
        Via("follows").           // edge collection
        Depth(1).
        Done(),

    // "Influence network" — unlimited-depth BFS from a user
    system.Traversal[User]("influence-network").
        From("users").
        Via("follows").
        Depth(-1).                // unlimited
        Done(),

    // "Shortest connection between two users"
    system.Path[User]("shortest-path").
        From("users").
        Via("follows").
        Done(),
),
```

### Runtime: graph queries

```go
// 1-hop: who does Alice follow?
following, err := system.Traverse[User](ctx, sys, "following", aliceID)

// Unlimited BFS: Alice's entire influence network
network, err := system.Traverse[User](ctx, sys, "influence-network", aliceID)

// Shortest path: how are Alice and Dave connected?
path, err := system.FindPath[User](ctx, sys, "shortest-path", aliceID, daveID)
```

### How the engine handles graph queries

```
Evolve[User]("users")                  → nodes materialized as typed rows
Evolve[Follow]("follows")              → edges materialized as adjacency list

Traversal[User]("following")           → GraphNeighbors(coll, node, depth=1)
Traversal[User]("influence-network")   → GraphNeighbors(coll, node, depth=-1)
Path[User]("shortest-path")            → ShortestPath(from, to)

On Memory engine                      → graphadapter wraps MemoryDriver, native traversal
On Dgraph engine                      → native DQL traversal, 0.3ms/query
On SQLite engine (no graph backend)   → WARN: graph traversal via recursive CTE,
                                        O(depth x degree). Estimated cost: 12ms/query.
```

The key insight: the developer experience is **identical** for CRUD and graph.
The system detects the edge type from the struct shape and classifies the ADT
as Graph instead of Map. The engine builds a graph internally or warns when it
can't.

---

## 10. Commands: Logic Without Infrastructure

Commands are pure business logic. Each handler receives:

- `context.Context`
- The typed command struct (embeds `*command.BasicCommand` for `StreamID()`)
- The current state (loaded by replaying events through the matching Evolution fold)

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

### No aggregates, no deciders, no repositories

The developer never writes `decider.Decider[State]`, `RegisterDecider`, or
`decider.NewRepository`. The system auto-builds all of these from the Evolution
fold + command handler signatures.

### Emissions

`system.Emit(...)` wraps events for return. `system.Emission` is the sealed
return type — it carries typed events and optional metadata (causation,
correlation):

```go
return system.Emit(
    system.Event("task.completed", TaskCompleted{ID: id}),
), nil

// Multiple events from one command:
return system.Emit(
    system.Event("task.completed", TaskCompleted{ID: id}),
    system.Event("notification.sent", NotificationSent{TaskID: id}),
), nil
```

---

## 11. How Commands Find Their State

The type parameter does the matching — no explicit wiring.

```
Command handler declares:    func(..., state TaskSummary) → State = TaskSummary
Evolution declares:          Evolve[TaskSummary](...)      → Result = TaskSummary
                                            ↑ same type ↑

System auto-connects:
  1. Command arrives → system loads stream events from event store
  2. Events replayed through the TaskSummary Evolution fold
  3. State passed to command handler
  4. Handler returns events → system appends to event store
  5. Events published to projection host → query projections updated
```

### When State != Public Query (Internal State)

If a command needs state that doesn't belong in a public query (e.g., "was this
task ever assigned?"), the developer declares an internal Evolution:

```go
system.Evolve[TaskInternalState]("task-internal", system.Internal()).
    On("task.created", TaskCreated{},
        func(e TaskCreated, v *TaskInternalState) { v.Exists = true }).
    On("task.assigned", TaskAssigned{},
        func(e TaskAssigned, v *TaskInternalState) { v.EverAssigned = true }).
    Done(),
```

`system.Internal()` marks the Evolution as state-only — not queryable via `Get`
or `Find`, but available for command state loading. Same fold mechanism, just
not exposed as a read endpoint.

### When multiple Evolutions share the same result type

If both `Lookup[TaskSummary]("get-task")` and `QuerySet[TaskSummary]("tasks")`
exist, the system uses the same `Evolve[TaskSummary]` fold for both projections.
The Evolution is declared once. Each query maintains its own independent
projection, but the fold logic is shared.

---

## 12. The Deployment: Where Data Lives

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
natively support the query's access pattern.

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

## 13. The Sealed Type System

No `[]any` anywhere. Every slice is a sealed interface:

```go
// Sealed interfaces — only constructors in the system package can satisfy these.
type CommandSpec   interface { isCommandSpec() }
type EvolutionSpec interface { isEvolutionSpec() }
type QuerySpec     interface { isQuerySpec() }

// Internal implementations (unexported):
type commandSpec   struct{ ... }; func (commandSpec)   isCommandSpec() {}
type evolutionSpec struct{ ... }; func (evolutionSpec) isEvolutionSpec() {}
type lookupSpec    struct{ ... }; func (lookupSpec)    isQuerySpec() {}
type querySetSpec  struct{ ... }; func (querySetSpec)  isQuerySpec() {}
type countSpec     struct{ ... }; func (countSpec)     isQuerySpec() {}
type traversalSpec struct{ ... }; func (traversalSpec) isQuerySpec() {}
```

A stray string, int, or nil in `Domain.Queries` is a **compile error**, not a
runtime panic.

### Generic methods for type inference

Go generics allow methods with additional type parameters. This enables
per-event type inference on the Evolution builder:

```go
// evolutionBuilder[R] carries the result type. On[E] introduces the event type.
type evolutionBuilder[R any] struct { ... }

func Evolve[R any](name string, opts ...EvolveOption) *evolutionBuilder[R]

// E is inferred from the sample argument. R is already known from the builder.
// The fold func type func(E, *R) is checked at compile time.
func (b *evolutionBuilder[R]) On[E any](
    eventType string,
    sample E,
    fold ...func(E, *R),  // optional: 0 = auto-classify by convention, 1 = explicit
) *evolutionBuilder[R]
```

**Compile-time inference proof:**

```go
system.Evolve[TaskSummary]("tasks").
    On("task.created", TaskCreated{})
//  ↑ R=TaskSummary (fixed)   ↑ E=TaskCreated (inferred from sample)

    .On("task.completed", TaskCompleted{}, func(e TaskCompleted, v *TaskSummary) {
        v.Status = "done"
    })
//  ↑ E=TaskCompleted (inferred)  ↑ checked: func(TaskCompleted, *TaskSummary) matches func(E, *R) ✓
```

---

## 14. Runtime API: Typed Reads

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

// Graph traversal (from Traversal query)
following, err := system.Traverse[User](ctx, sys, "following", aliceID)

// Graph shortest path (from Path query)
path, err := system.FindPath[User](ctx, sys, "shortest-path", aliceID, daveID)

// Pagination
page2, err := system.Find[TaskSummary](ctx, sys, "tasks",
    system.After(page1.Cursor),
    system.Limit(50),
)
```

---

## 15. Progressive Disclosure

Three levels of fold declaration, each adding power. All return the same sealed
`EvolutionSpec`.

### Level 1: Convention (80% of cases)

Field matching by name + type. Event struct suffix classifies the fold kind:

```go
system.Evolve[TaskSummary]("tasks").
    On("task.created", TaskCreated{}).    // Created suffix → insert
    On("task.deleted", TaskDeleted{}).    // Deleted suffix → remove
    Done()
```

### Level 2: Explicit fold (15% of cases)

For computed/derived fields:

```go
system.Evolve[TaskSummary]("tasks").
    On("task.created", TaskCreated{}).
    On("task.started", TaskStarted{},
        func(e TaskStarted, v *TaskSummary) { v.Status = "active" }).
    Done()
```

### Level 3: Custom (5% of cases)

For non-CRUD shapes (search, vector) or fully manual folds:

```go
system.Evolve[SearchIndex]("task-search",
    system.OnFold("task.created", TaskCreated{},
        func(e TaskCreated, idx *SearchIndex) {
            idx.Insert(e.ID, e.Title)
        }),
    system.OnFold("task.deleted", TaskDeleted{},
        func(e TaskDeleted, idx *SearchIndex) {
            idx.Remove(e.ID)
        }),
)
```

Returns the same sealed `EvolutionSpec`. No `[]any` leak.

---

## 16. What Disappears

| Concept                         | Current API                                               | v5 API                                                                                                    |
| ------------------------------- | --------------------------------------------------------- | --------------------------------------------------------------------------------------------------------- |
| **Aggregate**                   | `RegisterDecider[State](sys, "Task", decider)`            | **Deleted.** System auto-builds from Evolution + Command handler.                                         |
| **State type**                  | Separate `TaskState` struct                               | **Unified.** Evolution result type IS the state type.                                                     |
| **View**                        | `system.View[R,K]("name")`                                | **Deleted.** Queries (`Lookup`, `QuerySet`, `Count`, `Traversal`) replace it.                             |
| **Decider**                     | `decider.Decider[State]{Initial, Fold}`                   | **Auto-generated.** System builds from Evolution fold.                                                    |
| **Fold**                        | `metaengine.OnRecord(...)` / `OnRecordTyped(...)`          | **On Evolution.** Auto-generated for L1/L2.                                                               |
| **ADT**                         | Manual classification                                     | **Inferred** from result type shape (struct → Map, From/To fields → Graph, map\[string\]int64 → Counter). |
| **Decoder**                     | `ProjectionTypeDecoder`, `EventDecoder`, `PayloadDecoder` | **Auto-generated** from event struct types.                                                               |
| **Projection adapter**          | `projectionadapter.New(...)`                              | **Auto-wired.**                                                                                           |
| **Engine selection**            | Manual `metaengine.Query` + `Plan`                        | **Auto-planned** by metaengine.                                                                           |
| **Projection host**             | `projectionhost.New(...)`                                 | **Auto-created** from Deployment config.                                                                  |
| `Projections []any`             | Type-erased slice                                         | `Queries []QuerySpec` — sealed interface.                                                                 |
| `Commands func(*System)`        | Imperative callback                                       | `Commands []CommandSpec` — declarative.                                                                   |
| `map[string]any` in domain code | `graph.MergeNode(ref, map[string]any{...})`               | **Typed structs.** No `any` in domain logic.                                                              |

---

## 17. The Taskmanager Transformation

### Before: 199 LOC (metaengine.go) + 60 LOC (handlers.go) + decoder setup

```go
// 11 typed folds with manual EventWithID wrapping + 11-line decoder
func buildProjections() ([]any, *projectionadapter.TypeDecoder) {
    taskCounts := metaengine.Query[taskCountsInput, map[string]int64](
        "task_counts_by_status",
        metaengine.OnRecordTyped(string(evtTaskCreated),
            projectionadapter.EventWithID[TaskCreatedPayload]{},
            func(_ record.Record, _ projectionadapter.EventWithID[TaskCreatedPayload]) metaengine.Delta {
                return metaengine.Delta{string(StatusPending): 1}
            }),
        // ... 10 more folds ...
    )
    taskViews := metaengine.Query[listTasksInput, TaskView](
        "task_views",
        metaengine.OnRecordTyped(string(evtTaskCreated), ...),
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

### After: ~70 LOC total

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

        Evolutions: system.Evolutions(
            system.Evolve[TaskSummary]("tasks").
                On("task.created",   TaskCreated{}).
                On("task.started",   TaskStarted{},
                    func(e TaskStarted, v *TaskSummary) { v.Status = "active" }).
                On("task.completed", TaskCompleted{},
                    func(e TaskCompleted, v *TaskSummary) { v.Status = "done" }).
                On("task.deleted",   TaskDeleted{}).
                Done(),

            system.Evolve[map[string]int64]("task-counts").
                On("task.created",   +1, "pending").
                On("task.started",   +1, "active").
                On("task.started",   -1, "pending").
                On("task.completed", +1, "done").
                On("task.completed", -1, "active").
                On("task.deleted",   -1, "done").
                Done(),
        ),

        Queries: system.Queries(
            system.QuerySet[TaskSummary]("tasks").
                Filterable("status", "priority", "assignee").
                Sortable("priority", "created_at").
                Done(),

            system.Lookup[TaskSummary]("get-task").Done(),
        ),
    },
    system.SQLite("app.db"),
)

sys.Start(ctx)
defer sys.Close()
```

No folds on queries. No decoder. No `EventWithID` wrapper. No `[]any`. No
aggregate. No decider. No projection adapter. No `map[string]any`.

---

## 18. What to Learn from Akka

The project surveyed Akka Persistence and Kalix (formerly Akka Serverless) as
part of the CQRS/ES innovations research
(`docs/research/archive/2026-05-01_CQRS_EVENT_SOURCING_INNOVATIONS.md`).

### What Akka/Kalix does the same way

| Kalix concept        | What it does                     | This API's equivalent        |
| -------------------- | -------------------------------- | ---------------------------- |
| Event Sourced Entity | Classic event-sourced write side | Commands + Streams           |
| Event Handler        | Folds events into state          | Evolutions                   |
| View                 | Read-side projection from events | Queries                      |
| Topic                | Event stream for pub/sub         | Event Bus (auto-wired)       |
| Snapshot             | Periodic state capture           | Auto-wired (SnapshotBackend) |

The architecture is **identical**. Kalix calls them "Entities" and "Views."
This API calls them "Commands," "Evolutions," and "Queries." Same separation,
different vocabulary.

### What to take from Akka

| Take it                       | How                                                                     |
| ----------------------------- | ----------------------------------------------------------------------- |
| Views as first-class          | Already in this design — Queries are declared separately from Commands  |
| Event Sourced Entity = Stream | Already done via ADR-0001, ADR-0058                                     |
| Replicated Entity (CRDT)      | Future: maps to metaengine's replication model for multi-leader engines |
| Topic-based distribution      | Already have `event.Bus` + watermill                                    |

### What to leave

| Leave it                     | Why                                                                                                                          |
| ---------------------------- | ---------------------------------------------------------------------------------------------------------------------------- |
| Actor model / virtual actors | Requires a distributed runtime (cluster, sharding, location service). Contradicts "lite." Optimistic concurrency is simpler. |
| Kalix service mesh           | Deployment platform concern, not library concern.                                                                            |
| Kalix protobuf codegen       | Go generics provide compile-time safety without codegen.                                                                     |

### The single-writer comparison

|                         | Virtual Actors (Akka)  | Optimistic Concurrency (this lib)         |
| ----------------------- | ---------------------- | ----------------------------------------- |
| Single writer guarantee | Yes (runtime enforced) | No (conflicts possible, detected on Save) |
| Conflict handling       | None (serialized)      | Retry on version mismatch                 |
| Distribution            | Built-in               | External (Postgres, coordinator)          |
| Complexity              | Actor runtime required | Zero runtime — just version numbers       |

**Decision:** Do NOT adopt the actor model. It requires a distributed runtime
that contradicts "lite." Optimistic concurrency is simpler, works with any
backend, and the retry-on-conflict pattern is well-understood.

---

## 19. Open Questions

### Q1: Fold classification beyond Created/Updated/Deleted

Events like `TaskStarted`, `TaskAssigned`, `TaskArchived` don't match the naming
convention. Three options:

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

### Q3: Edge type detection

For graph queries, the system needs to know which fields on a struct are
endpoints. Two options:

| Option            | Mechanism                                      | Pro                   | Con                   |
| ----------------- | ---------------------------------------------- | --------------------- | --------------------- |
| **A. Convention** | `From`/`To` or `Source`/`Target` fields → edge | Zero config           | Fragile naming        |
| **B. Explicit**   | `system.Edge("From", "To")` option on Evolve   | Explicit, no surprise | Slightly more verbose |

**Current recommendation:** Convention (A) as default, with explicit `Edge()`
override. Convention + override = zero config for common case, escape hatch for
unconventional field names.

### Q4: Command state loading performance

Commands load state by replaying events through the Evolution fold. For hot
streams with long histories, this can be expensive. The existing
`decider.Repository` already solves this via snapshots, hot-state cache, and
load coalescing. These continue to work. The system auto-wires snapshots when
the engine implements `SnapshotBackend`. The developer doesn't configure these —
they're deployment-time concerns.

### Q5: Multiple result types from the same events

When two Evolutions fold the same events into different result types, each
maintains its own independent projection. The only cost is write amplification —
each event updates N projections. The planner already warns when this exceeds
the budget (`DefaultWriteAmplificationBudget = 3`).

---

## 20. Alternatives Considered

### A. Keep fold on the Query (two-way split)

**Rejected.** The fold IS domain logic — "completed tasks have status 'done'" is
a business rule, not access-pattern configuration. Mixing them on the Query
couples materialization logic to read configuration. The three-way split
(Evolutions + Queries + Commands) separates concerns cleanly.

### B. Keep `View` terminology

**Rejected.** "View" is database jargon. The developer thinks in terms of
queries: "get me this task" or "find tasks by status." The API should match the
mental model.

### C. Single `Query` constructor with access-pattern options

```go
// Rejected: too much flexibility, engine can't optimize upfront
system.Query[TaskSummary]("tasks").
    AccessPattern(system.PointLookup).
    ...
```

**Rejected.** The access pattern determines the ADT, storage shape, and index
strategy. Making it a builder option hides the one thing the engine needs to
know at plan time. Named constructors (`Lookup`, `QuerySet`, `Count`,
`Traversal`) make the access pattern a first-class choice.

### D. Declarative struct tags instead of builder API

```go
type TaskSummary struct {
    Status string `cqrs:"filterable"`
}
```

**Rejected.** Struct tags don't carry event relationships and don't support
explicit fold functions. The builder API is more powerful and has better IDE
autocomplete.

### E. Code generation (cqrs-gen) as the primary path

**Rejected as sole path.** Code generation can't adapt at runtime to available
engines. The planner's cost-based routing must happen at startup when engines
are known. Code generation is complementary, not primary.

### F. `map[string]any` for graph properties

**Rejected.** Violates the core principle: "No `any` as a value type in
domain/business logic." Graph nodes and edges use the same typed struct pattern
as CRUD entities. The system detects `From`/`To` fields and classifies the ADT
as Graph.

### G. Adopt the Akka actor model

**Rejected.** Requires a distributed runtime (cluster, sharding, location
service) that contradicts "lite." Optimistic concurrency is simpler, works with
any backend, and the retry-on-conflict pattern is well-understood. See
[Section 18](#18-what-to-learn-from-akka).

### H. Single flat events table (aggregateless ES)

**Rejected.** Loses optimistic concurrency, versioning, snapshots, and
stream-based loading. The stream partition key stays for operational reasons.
See [Section 3](#3-aggregateless-by-design).
