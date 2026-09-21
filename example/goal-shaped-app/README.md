# goal-shaped-app — The Goal in 5 Minutes

The library's north star, runnable:

> Developers declare ONLY Commands + Events + Queries and their relationships.
> Superb projections are derived for them. Where data lives is an **operator**
> decision made at deployment time.

This app is shaped exactly like that sentence:

| File        | Owner     | Contains                                                                                                | Never contains                                       |
| ----------- | --------- | ------------------------------------------------------------------------------------------------------- | ---------------------------------------------------- |
| `domain.go` | developer | plain Go structs (events, view, commands, queries)                                                      | any go-cqrs-lite engine, schema, registration, limit |
| `app.go`    | developer | ONE `Evolution` (naming-convention folds — zero fold closures), two read shapes, typed handlers         | engine names, DSNs, SQL, scan limits                 |
| `main.go`   | nobody    | loads `cqrs.yaml`, composes `system.New(Domain(), deployment)`, runs the story, prints EXPLAIN + Doctor | a hardcoded engine                                   |
| `cqrs.yaml` | operator  | engines (driver + DSN) and role bindings                                                                | anything about the domain                            |

## Run It

```bash
cd example/goal-shaped-app
GOWORK=off go run .
```

`cqrs.yaml` ships with SQLite at `goal.db`. After the five-line story runs
(two tasks created, one completed, one deleted), the binary prints the view,
the plan, and the Doctor report shown below. `go run .` creates `goal.db`
in the module directory (gitignored); reset the demo with
`trash goal.db && GOWORK=off go run .` — never `rm`.

> The Doctor/Explain numbers below are **machine-specific** — ns figures
> come from the calibrated cost model and live probes of the machine that
> ran the demo, so yours will differ. The shape of the report is what the
> story is about, not the exact numbers.

## Swap sqlite → postgres Without Touching the App

Both drivers are compiled in (two blank imports in `main.go`); the swap is
configuration, not code:

```bash
# Option A: edit cqrs.yaml
#   engines.primary: {driver: postgres, dsn: postgres://user:pass@host/goal}

# Option B: leave the file alone, override per environment
CQRS_ENGINES__PRIMARY__DRIVER=postgres \
CQRS_ENGINES__PRIMARY__DSN=postgres://user:pass@localhost:5432/goal \
GOWORK=off go run .
```

Same domain code, same folds, same queries — a different engine serves them.
A typo'd driver fails composition loudly (`TestGoal_UnknownDriverFailsLoud`
pins this); the swap itself is proven without a database in
`TestGoal_OperatorSwapsDriverByConfig`.

## What the Doctor Saw

Real output of `ExplainPlan()` + `Doctor()` after the story (sqlite run):

```
=== Metaengine Plan ===

--- Engines ---
  sqlite (point=3100ns, scan=1240ns/row, push=1080ns/row, agg=530ns/row, write=7000ns/op)

--- Queries ---
  open_tasks: map via sqlite (O(logN)) layout=Embed(Balanced) est=0.011ms read=1080ns
  tasks: map via sqlite (O(logN)) layout=Embed(Balanced) est=0.031ms read=3100ns

--- Diagnostics ---
  [INFO] open_tasks: auto-planned table meta_planned_open_tasks with columns [status]

=== Metaengine Doctor ===

--- Health ---
  all engines healthy

--- Collections ---
  tasks: 1 rows (sqlite)
  open_tasks: 1 rows (sqlite)

--- Poisoned ---
  none

--- Durability ---
  sqlite: engine-default

--- Persistence ---
  all persistent

--- Aggregate Pushdown ---
  open_tasks: pushdown: scalar, grouped, multi, multi-grouped, distinct
  tasks: pushdown: scalar, grouped, multi, multi-grouped, distinct

--- Layout ---
  open_tasks: Embed on sqlite (priority=Balanced, O(logN))
  tasks: Embed on sqlite (priority=Balanced, O(logN))

--- Planned tables ---
  open_tasks: meta_planned_open_tasks (rows=1, columns=[status])

--- Materialized views ---
  none
```

How to read it, line by line:

- **Engines** — the operator's `cqrs.yaml` choice, with the planner's cost
  priors. Swap the driver and this line changes; the queries below re-route
  with it.
- **Queries** — every declared read shape, the engine the planner assigned,
  the ADT and complexity, and the estimated latency. `tasks` is a point
  lookup, `open_tasks` a filterable collection — both inherited their folds
  from the ONE `Evolution` in `app.go`.
- **Diagnostics** — the planner noticed `open_tasks` filters on `status` and
  auto-planned a backing table with that column. Nothing was registered by
  hand; the declaration was enough.
- **Health / Poisoned** — engine quarantine state (health-driven
  deactivation) and corrupted collections. Boring is the goal.
- **Persistence** — the durability story the operator signed up for. Bind
  the projections role to a `memory` engine and this section says so loudly.
- **Layout / Planned tables** — where each query physically lives and why
  (the operator's `priority` flows in here).

## The Five-Minute Story in Code

Everything the developer writes beyond the structs in `domain.go`:

```go
// skip-validate
// app.go — folds declared ONCE, by naming convention. No closures.
tasks := system.OnEvolution(
    system.OnEvolution(
        system.Evolve[TaskView]("tasks"),
        "task.created", TaskCreated{},
    ),
    "task.updated", TaskUpdated{},
)
deleted := system.OnEvolution(tasks, "task.deleted", TaskDeleted{})

// two read shapes inherit the folds by result type
system.Lookup[TaskView]("tasks").Done(),
system.QuerySet[TaskView]("open_tasks").Filterable("status").Done(),
```

The app also declares its event universe — `Events: [task.created,
task.updated, task.deleted]` in `Domain()` — arming the coeffect gate
(system v4.8): a projection subscribing to an undeclared type fails
composition loudly (`TestDocs_CoeffectGate_DanglingSubscriptionFailsLoud`
demos the typo case). This fence is compile-gated by
`docs_compile_test.go`; it cannot drift from the API silently.

`Created`/`Updated`/`Deleted` suffixes classify the fold kind — create,
full-row update, and tombstone-remove (ADR-0114: deletion is a domain
event, and the view disappears from every collection). Non-convention
events can attach an explicit fold closure — see recipes.md §2.31 — but
this app doesn't need one.

The `task.deleted` fact makes the deleted task vanish from both read
shapes: `task.get` reports not-found, `task.open` doesn't list it. The
journal keeps every fact.

## Test It

```bash
cd example/goal-shaped-app
GOWORK=off go test ./...
```

- `TestGoal_SqliteEndToEnd` — the full story on a real SQLite engine,
  asserting the view and that EXPLAIN/Doctor name the operator's engine.
- `TestGoal_DeletedTaskStaysDeleted` — tombstone semantics.
- `TestGoal_OperatorSwapsDriverByConfig` — the sqlite→postgres swap via
  `CQRS_*` env overrides, plus all three drivers registered.
- `TestGoal_UnknownDriverFailsLoud` — typo'd drivers never silently boot.
- `TestDocs_ReadmeEvolutionFence` — the README fence above is compile-gated;
  it cannot drift from the API silently.
- `TestDocs_CoeffectGate_DanglingSubscriptionFailsLoud` — the `Events`
  declaration arms the coeffect gate; a typo'd subscription fails loudly.
- `TestDocs_AsyncAPIExport` — the declared commands/events/queries export an
  AsyncAPI 3.0 document (`catalog/asyncapi`): docs generate themselves, there
  is no second source of truth.

## Related

- [**getting-started**](../getting-started/) — the same pipeline with hand-written folds
- [**metaengine-quickstart**](../metaengine-quickstart/) — map, graph, and vector ADTs + `cqrs.yaml` boot
- [**taskmanager**](../taskmanager/) — flagship HTTP service with all modules wired
- Skill references: `core.md` §0 (mental model), `recipes.md` §2.31 (Evolutions)
