# go-cqrs-lite

[![CI](https://github.com/LarsArtmann/go-cqrs-lite/actions/workflows/ci.yml/badge.svg)](https://github.com/LarsArtmann/go-cqrs-lite/actions/workflows/ci.yml)
[![Go Reference](https://pkg.go.dev/badge/github.com/larsartmann/go-cqrs-lite/event/v2.svg)](https://pkg.go.dev/github.com/larsartmann/go-cqrs-lite/event/v2)

A lightweight CQRS + Event Sourcing **library** for Go. Import only what you need — each module has its own `go.mod` with minimal dependencies. Not a framework: no opinionated transport, broker, or SQL driver.

> **Using this library with an AI assistant?** Read [`SKILL.md`](SKILL.md) — the single-source guide with a module decision matrix, copy-paste composition recipes, conventions, and anti-patterns.

## Quick Start

```bash
go get github.com/larsartmann/go-cqrs-lite/event/v2
go get github.com/larsartmann/go-cqrs-lite/command/v2
go get github.com/larsartmann/go-cqrs-lite/decider/v2
go get github.com/larsartmann/go-cqrs-lite/memory/v2
```

```go
package main

import (
    "context"
    "fmt"

    "github.com/larsartmann/go-cqrs-lite/codec/v2"
    "github.com/larsartmann/go-cqrs-lite/command/v2"
    "github.com/larsartmann/go-cqrs-lite/decider/v2"
    "github.com/larsartmann/go-cqrs-lite/event/v2"
    "github.com/larsartmann/go-cqrs-lite/id/v2"
    "github.com/larsartmann/go-cqrs-lite/memory/v2"
)

type UserState struct{ Name string }
type CreateUser struct{ Name string }
type UserCreated struct{ Name string }

func main() {
    ctx := context.Background()
    store := memory.NewMemoryStore()
    bus := memory.NewMemoryBus()

    d := decider.Decider[UserState]{
        Initial: UserState{},
        Fold: func(s UserState, e event.Event) (UserState, error) {
            p, _ := event.DecodePayload[UserCreated](e, codec.JSONCodec{})
            s.Name = p.Name
            return s, nil
        },
    }
    repo, _ := decider.NewRepository(store, bus, d)

    cmds := command.NewDispatcher()
    aggID := id.NewAggregateID()
    command.RegisterTyped(cmds, "user.create",
        func(ctx context.Context, cmd *CreateUser) error {
            return repo.Execute(ctx, aggID, "User", func(s UserState, v event.Version) ([]event.Event, error) {
                return event.NewEvents(aggID, "User", v,
                    []event.Type{"user.created"}, []any{UserCreated{Name: cmd.Name}})
            })
        })

    _ = cmds.Dispatch(ctx, &CreateUser{Name: "Alice"})

    state, _, _ := repo.Load(ctx, aggID, "User")
    fmt.Printf("User: %s\n", state.Name) // User: Alice
}
```

See [`example/todo/`](example/todo/) for a full application with HTTP API, projections, and queries.

## Modules

Every module has its own README with detailed usage, types, and examples.

### Core

| Module         | Purpose                                                                         | README                         |
| -------------- | ------------------------------------------------------------------------------- | ------------------------------ |
| **event**      | Immutable events, store/bus interfaces, event sourcing, 5-family error taxonomy | [README](event/README.md)      |
| **command**    | Typed command dispatch, middleware, audit journal, pub/sub bus                  | [README](command/README.md)    |
| **query**      | Typed query dispatch, pagination, audit journal                                 | [README](query/README.md)      |
| **decider**    | Pure-function aggregate pattern (load → fold → decide → save)                   | [README](decider/README.md)    |
| **id**         | Branded IDs backed by ULID — impossible to mix up ID types                      | [README](id/README.md)         |
| **dispatcher** | Generic `Dispatcher[H, M]` with lifecycle management                            | [README](dispatcher/README.md) |
| **codec**      | Payload encoding: JSON, deterministic CBOR, Raw passthrough                     | [README](codec/README.md)      |

### Persistence

| Module       | Purpose                                                               | README                       |
| ------------ | --------------------------------------------------------------------- | ---------------------------- |
| **memory**   | In-memory store/bus/snapshot/checkpoint/command-bus — for tests & dev | [README](memory/README.md)   |
| **storage**  | SQL event/snapshot/checkpoint/command stores (PostgreSQL, SQLite)     | [README](storage/README.md)  |
| **pebble**   | Embedded KV: event/snapshot/checkpoint stores (PebbleDB + CBOR)       | [README](pebble/README.md)   |
| **kv**       | Layer-0 KV store abstraction: Store, MemStore, Iterator, Batch        | [README](kv/README.md)       |
| **turso**    | Turso/LibSQL connector with offline-first sync + indexing advisor     | [README](turso/README.md)    |
| **snapshot** | Snapshot types, strategies, store interfaces                          | [README](snapshot/README.md) |
| **schema**   | Schema evolution via upcasters and VersionedStore                     | [README](schema/README.md)   |

### Infrastructure

| Module         | Purpose                                                                  | README                         |
| -------------- | ------------------------------------------------------------------------ | ------------------------------ |
| **middleware** | Logging, retry, validation, recovery, circuit breaker, OTel, SSE, health | [README](middleware/README.md) |
| **projection** | Runner with replay + live subscription, handler registry                 | [README](projection/README.md) |
| **signing**    | Event signing: HMAC-SHA256, Ed25519, multi-sig                           | [README](signing/README.md)    |
| **encryption** | Payload encryption: XChaCha20-Poly1305, AES-256-GCM, key rotation        | [README](encryption/README.md) |
| **listing**    | Aggregate listing read model with tombstone-aware status                 | [README](listing/README.md)    |
| **otel**       | Shared OpenTelemetry helpers (tracer, meter, spans)                      | [README](otel/README.md)       |
| **watermill**  | Adapter for the Watermill message router ecosystem                       | [README](watermill/README.md)  |

### Tooling & Docs

| Module                | Purpose                                                               | README                                |
| --------------------- | --------------------------------------------------------------------- | ------------------------------------- |
| **catalog**           | Auto-generate AsyncAPI 3.0, EventCatalog, OpenAPI, D2 from Go types   | [README](catalog/README.md)           |
| **testutil**          | Shared test helpers: `MustNewCmd`, `ParseAggID`, `NoopCommandHandler` | [README](testutil/README.md)          |
| **cmd/cqrs-gen**      | Code generator: typed handler registration from `//cqrs:` markers     | [README](cmd/cqrs-gen/README.md)      |
| **cmd/api-stability** | API surface checker: compare exports against golden file              | [README](cmd/api-stability/README.md) |
| **integration**       | Cross-module integration tests                                        | [README](integration/README.md)       |

### Examples

| Module                 | Purpose                                             | README                                 |
| ---------------------- | --------------------------------------------------- | -------------------------------------- |
| **example/todo**       | Full app: HTTP API, decider, projections, queries   | [README](example/todo/README.md)       |
| **example/user**       | Advanced patterns: signing, middleware, catalog     | [README](example/user/README.md)       |
| **example/encryption** | Event encryption patterns: bus, store, key rotation | [README](example/encryption/README.md) |

## Design Principles

1. **Library, not framework** — Import what you need. No opinionated transport, broker, or driver.
2. **Interface-first** — All core types are interfaces (`Store = EventSink + EventSource`).
3. **Composition over inheritance** — Per Go best practices.
4. **Strong types** — Branded IDs, no `any` in public API (except DB interop).
5. **Errors as values** — No panics in production paths; 5-family error taxonomy.
6. **Multi-module isolation** — Each module has its own `go.mod` with only needed deps.
7. **Tombstone over delete** — Soft-delete via metadata; no `Delete` on Store.

## Comparison

| Feature              | go-cqrs-lite | go-cqrs | cqrs-go |
| -------------------- | :----------: | :-----: | :-----: |
| Minimal deps         |      ✅      |   ❌    |   ❌    |
| Event Sourcing       |      ✅      |   ✅    |   ✅    |
| Strong IDs           |      ✅      |   ❌    |   ❌    |
| Middleware           |      ✅      |   ❌    |   ❌    |
| Event Signing        |      ✅      |   ❌    |   ❌    |
| Event Encryption     |      ✅      |   ❌    |   ❌    |
| Schema Evolution     |      ✅      |   ❌    |   ❌    |
| Auto-docs (AsyncAPI) |      ✅      |   ❌    |   ❌    |
| Stream Loading       |      ✅      |   ❌    |   ❌    |

## Status

**v2.6.0** — 30 modules, 84–100% test coverage, 0 lint issues. Active development.

See [FEATURES.md](FEATURES.md) for the full feature inventory and [docs/](docs/) for architecture decisions (ADRs), benchmarks, and storage guides.

## License

MIT
