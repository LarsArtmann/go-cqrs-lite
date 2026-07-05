# Task Manager — go-cqrs-lite Flagship Example

A production-grade task management service that demonstrates go-cqrs-lite **to the max**: event sourcing, CQRS, projections, middleware, observability, signing, and a real HTTP API.

## What This Example Shows

| Feature              | How                                                                                         |
| -------------------- | ------------------------------------------------------------------------------------------- |
| **Event Sourcing**   | Per-event payloads (not fat payloads), `decider.Repository.Execute`, optimistic concurrency |
| **CQRS**             | Command dispatcher with typed handlers; KV-backed read model via Materialize                |
| **Projections**      | `stack.Materialize` with tombstone-aware `OnCreate`/`OnUpdate`/`OnTombstone` callbacks      |
| **Ordered Delivery** | `CatchUpSubscriber` (journal replay + live handoff) consumed from a single goroutine        |
| **Persistence**      | SQLite via `stack/sqlite` preset (swap to Pebble/Postgres by changing one line)             |
| **Middleware**       | Recovery, Logging, Retry on the command dispatcher                                          |
| **Observability**    | OpenTelemetry tracing + metrics via `otel.Setup` + `middleware.NewOTelBundle`               |
| **Signing**          | HMAC-SHA256 event signing (tamper-evident streams)                                          |
| **Tombstone**        | Soft-delete via metadata — no hard deletes, data preserved                                  |
| **Testing**          | Scenario DSL (`Given/When/Then`), integration tests, HTTP API tests                         |
| **Error Taxonomy**   | 5-family error classification mapped to HTTP status codes                                   |
| **Branded IDs**      | `id.AggregateID` for type-safe identifiers                                                  |

## Architecture

```
HTTP API ──▶ Command Dispatcher ──▶ Decider Repository ──▶ Event Store (SQLite)
                 (middleware:         (load → fold →          │
                  recovery,            decide → save)         ▼
                  logging,                               EventBus (Watermill)
                  retry, OTel)                               │
                                                             ▼
                                                    CatchUpSubscriber
                                                    (ordered replay + live)
                                                             │
                                                             ▼
                                                    Materialize
                                                    (KV-backed TaskView)
                                                             │
                                                             ▼
                                                    Read Model Queries
```

## Quick Start

```bash
# Run the server (in-memory database)
go run .

# Or with persistent storage
DATABASE_PATH=./tasks.db go run .

# Create a task
curl -X POST http://localhost:8080/api/tasks \
  -H 'Content-Type: application/json' \
  -d '{"title":"Learn go-cqrs-lite","priority":"high"}'

# List tasks
curl http://localhost:8080/api/tasks

# Start a task
curl -X POST http://localhost:8080/api/tasks/<id>/start

# Complete a task
curl -X POST http://localhost:8080/api/tasks/<id>/complete

# Get a single task
curl http://localhost:8080/api/tasks/<id>

# Delete (soft-delete via tombstone)
curl -X DELETE http://localhost:8080/api/tasks/<id>
```

## API Endpoints

| Method   | Path                       | Description                             |
| -------- | -------------------------- | --------------------------------------- |
| `GET`    | `/health`                  | Health check                            |
| `POST`   | `/api/tasks`               | Create a task                           |
| `GET`    | `/api/tasks`               | List tasks (optional `?status=` filter) |
| `GET`    | `/api/tasks/{id}`          | Get a task                              |
| `PATCH`  | `/api/tasks/{id}`          | Update title, priority, or due date     |
| `DELETE` | `/api/tasks/{id}`          | Soft-delete (tombstone)                 |
| `POST`   | `/api/tasks/{id}/assign`   | Assign to a user                        |
| `POST`   | `/api/tasks/{id}/start`    | Start the task                          |
| `POST`   | `/api/tasks/{id}/complete` | Complete the task                       |
| `POST`   | `/api/tasks/{id}/archive`  | Archive the task                        |
| `POST`   | `/api/tasks/{id}/blockers` | Add a blocking dependency               |

## File Structure

```
taskmanager/
├── domain.go          # Value types, branded IDs, validation, error helpers
├── events.go          # Per-event payloads (11 event types)
├── decider.go         # Pure fold + decide functions (10 commands)
├── decider_test.go    # Scenario tests (Given/When/Then BDD)
├── projection.go      # KV Materialize: TaskView read model
├── handlers.go        # Command/query handler registration (typed dispatch)
├── http.go            # HTTP routes, handlers, error taxonomy → status mapping
├── setup.go           # Composition root: stack.Bundle, Repository, CatchUp
├── features.go        # Middleware, OTel, signing
├── main.go            # Entry point
└── integration_test.go # End-to-end: command pipeline + HTTP API
```

## Key Design Decisions

### Per-Event Payloads (not fat payloads)

Each event carries ONLY the data that changed. `TaskTitleUpdated` has just `{title}`, not the entire task state. This is the correct event sourcing pattern.

### Decider.Repository.Execute (not raw store access)

Commands go through `decider.Repository.Execute`, which handles load → fold → decide → save → publish automatically. No manual event store access in handlers.

### Deployer-First

The deployer chooses infrastructure (one line: `sqlite.New(...)`). The consumer code is identical whether you use SQLite, Pebble, Postgres, or in-memory.

### Error Taxonomy → HTTP Status

The 5-family error taxonomy maps directly to HTTP status codes:

- Rejection → 400 (bad request)
- Conflict → 409 (state conflict)
- Transient → 503 (retry later)
- Infrastructure → 500 (server error)
- Corruption → 500 (data corruption)

## Swapping Databases

Change ONE line in `setup.go`:

```go
// SQLite (default)
bundle, err := sqlite.New("tasks.db", sqlite.WithOptimizations())

// Pebble (embedded KV)
bundle, err := pebble.New("./data")

// Postgres
bundle, err := postgres.New(dsn)

// In-memory (testing)
bundle, err := memory.New()
```

The domain, events, decider, projection, and handler code doesn't change.
