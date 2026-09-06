# Task Manager — go-cqrs-lite Flagship Example

A production-grade task management service that demonstrates go-cqrs-lite **to the max**: event sourcing, CQRS, projections, middleware, observability, signing, and a real HTTP API.

## What This Example Shows

| Feature              | How                                                                                       |
| -------------------- | ----------------------------------------------------------------------------------------- |
| **Event Sourcing**   | Per-event payloads (not fat payloads), `system.Execute` Op, optimistic concurrency        |
| **CQRS**             | Command dispatcher with typed handlers; metaengine Map read model                         |
| **Projections**      | `metaengine` Map ADT folds + `projectionhost` (crash-restart, DLQ, checkpoints)           |
| **Ordered Delivery** | `projectionhost` replays the journal, then follows live events into the metaengine store  |
| **Persistence**      | SQLite via `system.EngineConfig{Driver: "sqlite"}` (swap by changing one `Driver` line)   |
| **Middleware**       | Recovery, Logging, Retry on the command dispatcher                                        |
| **Observability**    | OpenTelemetry tracing + metrics via `otel.Setup` + `middleware.NewOTelBundle`             |
| **Signing**          | HMAC-SHA256 event signing (tamper-evident streams)                                        |
| **Tombstone**        | Soft-delete as a `task.deleted` domain event (ADR-0114) — no hard deletes, data preserved |
| **Deriver sagas**    | `deriver` derives follow-up commands from events (assignment cascade)                     |
| **Testing**          | Scenario DSL (`Given/When/Then`), integration tests, HTTP API tests                       |
| **Error Taxonomy**   | 6-family error classification mapped to HTTP status codes                                 |
| **Branded IDs**      | `TaskID = id.StreamID` for type-safe identifiers                                          |

## Architecture

```
HTTP API ──▶ Command Dispatcher ──▶ system.Execute Op ──▶ Event Store (SQLite)
                 (middleware:         (decider:              │
                  recovery,            load → fold →         ▼
                  logging,             decide → save)   EventBus
                  retry, OTel)                               │
                                                             ▼
                                                    projectionhost
                                                    (replay + live, DLQ)
                                                             │
                                                             ▼
                                                    metaengine Map ADT
                                                    (TaskView read model)
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
├── domain.go           # Value types, branded IDs, validation, error helpers
├── events.go           # Per-event payloads (11 event types)
├── decider.go          # Pure fold + decide functions (11 commands)
├── decider_test.go     # Scenario tests (Given/When/Then BDD)
├── projection.go       # TaskView read-model type
├── metaengine.go       # metaengine Map ADT folds + projection adapter registration
├── handlers.go         # Command/query handler registration (typed dispatch)
├── deriver.go          # deriver: events → follow-up commands
├── http.go             # HTTP routes, handlers, error taxonomy → status mapping
├── setup.go            # Composition root: system.New with DomainConfig + DeploymentConfig
├── features.go         # Middleware, OTel, signing
├── codec_init.go       # Codec registration
├── main.go             # Entry point
├── integration_test.go # End-to-end: command pipeline + HTTP API
├── idempotency_test.go # Idempotent command dispatch
├── sse_test.go         # SSE stream tests
└── metaengine_test.go  # Read-model convergence tests
```

## Key Design Decisions

### Per-Event Payloads (not fat payloads)

Each event carries ONLY the data that changed. `TaskTitleUpdated` has just `{title}`, not the entire task state. This is the correct event sourcing pattern.

### system.Execute Op (not raw store access)

Commands go through `system.RegisterCommand` + the `system.Execute` Op, which
drives a decider underneath: load → fold → decide → save → publish happens
automatically. No manual event store access in handlers.

### Deployer-First

The deployer chooses infrastructure in the `DeploymentConfig` (one `Driver`
line in `setup.go`). The consumer code is identical whether you use SQLite,
Pebble, Postgres, or in-memory.

### Error Taxonomy → HTTP Status

The 6-family error taxonomy maps directly to HTTP status codes:

- Rejection → 400 (bad request)
- Conflict → 409 (state conflict)
- Transient → 503 (retry later)
- Infrastructure → 500 (server error)
- Corruption → 500 (data corruption)
- Orchestration → 500 (workflow coordination failure)

## Swapping Databases

Change ONE `Driver` line in the `Engines` map in `setup.go`:

```go
// SQLite (default)
Engines: map[string]system.EngineConfig{
    primaryEngine: {Driver: "sqlite", DSN: cfg.DatabasePath},
}

// Pebble (embedded KV) — Driver: "pebble", DSN: "./data"
// In-memory (testing) — Driver: "memory"
```

Each backend the deployment might use needs one blank import so its driver
self-registers (see `setup.go`). The domain, events, decider, projection, and
handler code doesn't change.
