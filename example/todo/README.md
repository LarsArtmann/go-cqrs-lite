# Todo Example — Event Sourcing with go-cqrs-lite

A complete todo application demonstrating how to build on [go-cqrs-lite](../../) using **pure event sourcing** — no CRUD anywhere.

## Architecture

```
HTTP Request
    │
    ▼
┌─────────────────────────────────────────────┐
│  Handler (cmd/api/main.go)                   │
│  Deserializes HTTP → Command/Query           │
└──────────┬──────────────────┬────────────────┘
           │                  │
     Command│                 │Query
           ▼                  ▼
┌──────────────────┐  ┌───────────────────┐
│ CommandDispatcher│  │  QueryDispatcher  │
│ (go-cqrs-lite)   │  │  (go-cqrs-lite)   │
└────────┬─────────┘  └────────┬──────────┘
         │                      │
         ▼                      ▼
┌──────────────────┐  ┌───────────────────┐
│ Command Handler  │  │  Query Handler    │
│ (no repo deps!)  │  │  (reads from      │
│                  │  │   projection)     │
└────────┬─────────┘  └────────┬──────────┘
         │                      │
         ▼                      │
┌──────────────────┐            │
│ Aggregate (Todo) │            │
│ Business logic,  │            │
│ produces events  │            │
└────────┬─────────┘            │
         │                      │
    Events│                     │
         ▼                      │
┌──────────────────┐            │
│   Event Store    │            │
│   (Pebble DB)    │            │
└────────┬─────────┘            │
         │                      │
    Publish│                     │
         ▼                      │
┌──────────────────┐            │
│   Event Bus      │            │
│   (in-memory)    │            │
└────────┬─────────┘            │
         │                      │
    Subscribe                     │
         ▼                      │
┌──────────────────┐     ┌──────┴──────────┐
│   Projection     │     │  Read Model     │
│   (TodoProj.)    │────▶│  Store          │
│   events → state │     │  (Pebble/Mem)   │
└──────────────────┘     └─────────────────┘
```

## Key Principles

### 1. Commands write ONLY to the Event Store

Command handlers load events → rebuild aggregate → apply business logic → save new events. They **never** write directly to a read model.

### 2. Projections build the read model asynchronously

The `TodoProjection` subscribes to the event bus and maintains a denormalized read model. Query handlers read exclusively from this projection store.

### 3. No CRUD Repository

There is no `TodoRepository` interface with `Create/Update/Delete`. The read model uses `Put` (upsert) and `Delete` — operations driven entirely by events.

## Project Structure

```
example/todo/
├── cmd/api/main.go          # Entry point, wires everything together
├── domain/
│   ├── todo.go              # Todo model, statuses, events
│   └── ids.go               # Strongly-typed IDs (TodoID, EventID, etc.)
├── aggregate/
│   └── decider.go           # Event-sourced aggregate (pure Fold + Decide)
├── commands/
│   ├── mixin.go             # Shared command handler infrastructure
│   ├── create_todo.go       # CreateTodo command
│   ├── update_todo.go       # UpdateTodo command
│   ├── delete_todo.go       # DeleteTodo command
│   └── change_status.go     # ChangeStatus command
├── queries/
│   ├── get_todo.go          # Get single todo
│   ├── list_todos.go        # List with filters
│   └── count_todos.go       # Count matching todos
├── projections/
│   └── todo_projection.go   # Events → read model (the key piece!)
├── storage/
│   ├── pebble_store.go      # Pebble-backed read model store
│   ├── memory_store.go      # In-memory read model store (for testing)
│   └── iterator_helpers.go  # Pebble iterator utilities
└── go.mod                   # Own module with go-cqrs-lite deps
```

## Running

```bash
cd example/todo
GOPRIVATE='github.com/larsartmann/*' GONOSUMCHECK='*' go run ./cmd/api

# API at http://localhost:8080
# Health: GET /health
# Todos:  GET/POST /api/v1/todos
#         GET/PUT/DELETE /api/v1/todos/:id
#         PATCH /api/v1/todos/:id/status
```

## Dependencies

| Package                            | Purpose                                                |
| ---------------------------------- | ------------------------------------------------------ |
| `go-cqrs-lite/core`                | CQRS framework (decider, commands, queries, events)    |
| `go-cqrs-lite/memory`              | In-memory event bus                                    |
| `go-cqrs-lite/storage`             | Pebble event store adapter                             |
| `cqrs-htmx`                        | HTTP middleware composition                            |
| `cockroachdb/pebble`               | Embedded key-value storage                             |

## Related

- [go-cqrs-lite/example/user](../user/) — Minimal CLI demo of the aggregate lifecycle
- [cqrs-htmx](../../../cqrs-htmx/) — HTMX-aware HTTP integration for go-cqrs-lite
