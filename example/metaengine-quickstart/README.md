# metaengine-quickstart

The Record-aware metaengine pipeline in one runnable example: declare folds
and queries, `Plan` a store over an engine, apply records, execute typed
queries — then swap the engine without touching domain code.

```bash
go run .
```

## What it demonstrates

1. **Maps** — CRUD folds by `Created`/`Updated`/`Deleted` convention (zero
   boilerplate): `TaskCreated`/`TaskUpdated`/`TaskDeleted` events fold into a
   `TaskView` map collection, queried by filter.
2. **Graph** — a social follow network: one event type (`UserFollowed`) folds
   into `metaengine.Edge` records; a reachability query ("everyone within N
   hops") routes to the engine's graph ADT (native traversal on SQLite/PG,
   BFS fallback elsewhere).
3. **Vector** — document embeddings with k-NN semantic search: `DocEmbedded`
   folds into `metaengine.Embedding` records; the query routes to the
   engine's vector ADT (brute-force on Memory/SQLite, ANN indexes elsewhere).
4. **Operator config** (`runConfigFileDemo`) — the deployment-time story:
   load engine choices from a `cqrs.yaml` (koanf) and boot a `system`
   deployment from it. The developer's domain code stays in Go
   (`DomainConfig`); the operator's engine choices live in the YAML and can
   change per environment without a rebuild.

## The pipeline every demo shares

```go
folds, err := metaengine.AutoCRUDByConvention[TaskView]("ID",
	TaskCreated{}, TaskUpdated{}, TaskDeleted{},
)

query := metaengine.Query[TaskQuery, TaskView]("tasks", foldArgs...)

store, err := metaengine.Plan(
	[]metaengine.Engine{metaengine.NewMemoryEngine()}, // + any backend engine
	query,
)
defer func() { _ = store.Close() }()

result, err := metaengine.ExecuteTyped[TaskQuery, TaskView](ctx, store, TaskQuery{ID: "task-1"})
```

Swap the `Memory` engine for any backend (`sqliteengine`, `pgengine`,
`pebbleengine`, …) — the declarations and queries are engine-agnostic; the
planner routes each query to the engine's cheapest capable ADT and emits an
advisory diagnostic (not an error) when a shape degrades (e.g. graph
traversal on a non-graph engine).

## Files

| File                  | Section                  |
| --------------------- | ------------------------ |
| `main.go`             | Maps demo + entrypoint   |
| `graph_demo.go`       | Graph demo               |
| `vector_demo.go`      | Vector demo              |
| `configfile_demo.go`  | Operator-config demo     |
