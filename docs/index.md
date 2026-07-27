# go-cqrs-lite

A lightweight CQRS **library** for Go with Event Sourcing, branded IDs, and
auto-documentation generation.

Consumers import what they need and compose their own stack. Not a framework —
no opinionated transport, message broker, or SQL driver.

## Why go-cqrs-lite?

| Feature                    | Description                                                                                                               |
| -------------------------- | ------------------------------------------------------------------------------------------------------------------------- |
| **Library, not framework** | Import only what you need. No magic, no lock-in.                                                                          |
| **Event Sourcing**         | Immutable event streams with optimistic concurrency, snapshots, and time travel.                                          |
| **Bundle presets**         | One call wires every store + bus + read-model backend (`stack/memory`, `stack/sqlite`, `stack/pebble`, `stack/postgres`). |
| **Multi-module isolation** | 30+ modules, each with its own `go.mod`. Minimal dependencies.                                                            |
| **Strong types**           | Branded IDs (`id.Of[T]`), typed command/query handlers, generic stores.                                                   |
| **Trustworthy**            | 0 lint violations, 84–100% test coverage on core modules, API stability checked.                                          |
| **Production-ready**       | OTel tracing/metrics, event signing, encryption, schema evolution, Pebble backups.                                        |

## Quick Start

### Bundle preset (recommended)

```go
import "github.com/larsartmann/go-cqrs-lite/stack/memory/v4"

// One call wires the event store, bus, snapshot store, checkpoint store,
// command store, query store, and read-model backend.
bundle, cleanup, err := stackmemory.New()
if err != nil { panic(err) }
defer cleanup()

// Typed read models persist across restarts (for SQL/Pebble presets)
store := readmodel.NewStore[UserView, UserID](bundle.ReadModels, codec.JSONCodec{})
```

### Individual modules

```go
import (
    "github.com/larsartmann/go-cqrs-lite/event/v4"
    "github.com/larsartmann/go-cqrs-lite/decider/v4"
    "github.com/larsartmann/go-cqrs-lite/memory/v4"
)

store := memory.NewMemoryStore()
defer store.Close()

// Pure-function decider
d := decider.Decider[UserState]{
    Initial: UserState{},
    Fold:    foldUserEvents,
}

// Execute a command → produce events → persist → publish
repo := decider.NewRepository(store, bus)
result, err := decider.Execute(ctx, repo, aggregateID, d, command)
```

## Module Overview

| Layer              | Modules                                                     | Purpose                            |
| ------------------ | ----------------------------------------------------------- | ---------------------------------- |
| **Core**           | `event`, `command`, `query`, `decider`, `id`, `dispatcher`  | Domain primitives                  |
| **Persistence**    | `memory`, `storage` (SQL), `pebble`, `turso`                | Store implementations              |
| **Infrastructure** | `middleware`, `signing`, `encryption`, `otel`, `prometheus` | Cross-cutting concerns             |
| **Composition**    | `stack`, `readmodel`, `readmodel/cache`                     | Bundle presets                     |
| **Tooling**        | `cqrs-gen`, `api-stability`, `catalog`                      | Code generation, docs, validation  |
| **Schema**         | `schema`, `snapshot`, `codec`, `kv`                         | Evolution, snapshots, encoding, KV |

## Presets Comparison

| Preset           | Event Store         | Read Models         | Multi-Process       | Best For                 |
| ---------------- | ------------------- | ------------------- | ------------------- | ------------------------ |
| `stack/memory`   | In-memory           | In-memory           | No                  | Testing, prototyping     |
| `stack/sqlite`   | SQLite (persistent) | SQL KV (persistent) | No                  | Single-process apps      |
| `stack/pebble`   | Pebble (embedded)   | Pebble KV           | No                  | High-throughput embedded |
| `stack/postgres` | PostgreSQL          | SQL KV (persistent) | Yes (LISTEN/NOTIFY) | Multi-process production |

## Next Steps

- [Getting Started](getting-started.md) — installation and first event
- [Storage Guide](STORAGE_GUIDE.md) — SQL, Pebble, Turso backends
- [Architecture Patterns](ARCHITECTURE_PATTERNS.md) — CQRS/ES patterns
- [Domain Language](DOMAIN_LANGUAGE.md) — ubiquitous language glossary
- [Error Taxonomy](error-taxonomy.md) — 6-family error classification
- [ADRs](adr/README.md) — architecture decision records
