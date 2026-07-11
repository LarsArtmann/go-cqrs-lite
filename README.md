# go-cqrs-lite

[![CI](https://github.com/LarsArtmann/go-cqrs-lite/actions/workflows/ci.yml/badge.svg)](https://github.com/LarsArtmann/go-cqrs-lite/actions/workflows/ci.yml)
[![Go Reference](https://pkg.go.dev/badge/github.com/larsartmann/go-cqrs-lite/event/v3.svg)](https://pkg.go.dev/github.com/larsartmann/go-cqrs-lite/event/v3)

A lightweight CQRS + Event Sourcing **library** for Go. Import only what you need — each module has its own `go.mod` with minimal dependencies. Not a framework: no opinionated transport, broker, or SQL driver.

> **Using this library with an AI assistant?** Read [`SKILL.md`](SKILL.md) — the single-source guide with a module decision matrix, copy-paste composition recipes, conventions, and anti-patterns.

## Quick Start

```bash
go get github.com/larsartmann/go-cqrs-lite/event/v3
```

```go
package main

import (
    "context"
    "fmt"

    "github.com/larsartmann/go-cqrs-lite/codec/v3"
    "github.com/larsartmann/go-cqrs-lite/command/v3"
    "github.com/larsartmann/go-cqrs-lite/decider/v3"
    "github.com/larsartmann/go-cqrs-lite/event/v3"
    "github.com/larsartmann/go-cqrs-lite/id/v3"
    "github.com/larsartmann/go-cqrs-lite/storage/memory/v3"
    cqrswatermill "github.com/larsartmann/go-cqrs-lite/watermill/v3"
)

type UserState struct{ Name string }
type UserCreated struct{ Name string }

// CreateUser implements command.Command (Type + AggregateID required).
type CreateUser struct {
    aggID id.AggregateID
    Name  string
}

func (c *CreateUser) Type() command.Type          { return "user.create" }
func (c *CreateUser) AggregateID() id.AggregateID { return c.aggID }

func main() {
    ctx := context.Background()
    store := memory.NewMemoryStore()
    bus := cqrswatermill.NewEventBus()

    d := decider.Decider[UserState]{
        Initial: UserState{},
        Apply: func(s UserState, e event.Event) (UserState, error) {
            p, _ := event.DecodePayloadAuto[UserCreated](e)
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

    _ = cmds.Dispatch(ctx, &CreateUser{aggID: aggID, Name: "Alice"})

    state, _, _ := repo.Load(ctx, aggID, "User")
    fmt.Printf("User: %s\n", state.Name) // User: Alice
}
```

### Deployer-First Pattern (recommended for production)

For a fully-wired stack with persistent storage, event bus, read models, and
projections — use a **stack preset**. One import, one function call:

```go
import "github.com/larsartmann/go-cqrs-lite/stack/sqlite/v3"

// Events, commands, queries, snapshots, checkpoints, read models — all persisted.
// Event bus wired (watermill GoChannel for in-process pub/sub).
bundle, err := sqlite.New("app.db")
// Use bundle.EventStore, bundle.Bus, bundle.ReadModels, bundle.CommandDispatcher...
defer bundle.Close()
```

Five presets available: `stack/memory` (tests), `stack/sqlite` (embedded),
`stack/pebble` (high-throughput embedded KV), `stack/postgres` (distributed),
`stack/turso` (embedded Turso Database with optional remote sync).

See [`example/taskmanager/`](example/taskmanager/) for a complete
end-to-end HTTP service with CQRS/ES, projections, signing, SSE, and more.
See [`example/getting-started/`](example/getting-started/) for a minimal
80-line demo of the core loop.

## Modules

Every module has its own README with detailed usage, types, and examples.

### Core

| Module          | Purpose                                                                         | README                          |
| --------------- | ------------------------------------------------------------------------------- | ------------------------------- |
| **event**       | Immutable events, store/bus interfaces, event sourcing, 5-family error taxonomy | [README](event/README.md)       |
| **command**     | Typed command dispatch, middleware, audit journal, pub/sub bus                  | [README](command/README.md)     |
| **query**       | Typed query dispatch, pagination, audit journal                                 | [README](query/README.md)       |
| **decider**     | Pure-function aggregate pattern (load → fold → decide → save)                   | [README](decider/README.md)     |
| **id**          | Branded IDs backed by ULID — impossible to mix up ID types                      | [README](id/README.md)          |
| **idempotency** | Command idempotency store — dedup at-least-once delivery retries                | [README](idempotency/README.md) |
| **dispatcher**  | Generic `Dispatcher[H, M]` with lifecycle management                            | [README](dispatcher/README.md)  |
| **codec**       | Payload encoding: JSON, deterministic CBOR, Raw passthrough                     | [README](codec/README.md)       |

### Persistence

| Module             | Purpose                                                           | README                            |
| ------------------ | ----------------------------------------------------------------- | --------------------------------- |
| **storage/memory** | In-memory store/snapshot/checkpoint — for tests & dev             |                                   |
| **storage**        | SQL event/snapshot/checkpoint/command stores (PostgreSQL, SQLite) |                                   |
| **storage/pebble** | Embedded KV: event/snapshot/checkpoint stores (PebbleDB + CBOR)   |                                   |
| **storage/turso**  | Turso connector with offline-first sync + indexing advisor        | [README](storage/turso/README.md) |
| **kv**             | Layer-0 KV store abstraction: Store, MemStore, TypedStore, Cache  |                                   |
| **snapshot**       | Snapshot types, strategies, store interfaces                      |                                   |
| **schema**         | Schema evolution via upcasters and VersionedStore                 |                                   |

### Infrastructure

| Module             | Purpose                                                             | README                         |
| ------------------ | ------------------------------------------------------------------- | ------------------------------ |
| **middleware**     | Logging, retry, validation, recovery, circuit breaker, OTel tracing |                                |
| **transport/http** | SSE event delivery (Server-Sent Events over HTTP)                   |                                |
| **signing**        | Event signing: HMAC-SHA256, Ed25519, multi-sig                      |                                |
| **encryption**     | Payload encryption: XChaCha20-Poly1305, AES-256-GCM, key rotation   | [README](encryption/README.md) |
| **listing**        | Aggregate listing read model with tombstone-aware status            |                                |
| **otel**           | Shared OpenTelemetry helpers (tracer, meter, spans)                 |                                |
| **watermill**      | EventBus adapter, CatchUpSubscriber, EventPublisher                 |                                |
| **prometheus**     | OTel→Prometheus metrics bridge with /metrics handler                |                                |

### Tooling & Docs

| Module                | Purpose                                                             | README                                |
| --------------------- | ------------------------------------------------------------------- | ------------------------------------- |
| **catalog**           | Auto-generate AsyncAPI 3.0, EventCatalog, OpenAPI, D2 from Go types | [README](catalog/README.md)           |
| **testutil**          | Shared test helpers: `NewCmd`, `NoopCommandHandler`                 | [README](testutil/README.md)          |
| **cmd/cqrs-gen**      | Code generator: typed handler registration from `//cqrs:` markers   | [README](cmd/cqrs-gen/README.md)      |
| **cmd/api-stability** | API surface checker: compare exports against golden file            | [README](cmd/api-stability/README.md) |
| **cmd/doc-check**     | Doc cross-reference verifier: validates Go symbols in markdown docs |                                       |
| **integration**       | Cross-module integration tests                                      | [README](integration/README.md)       |

### Stack Presets

| Module             | Purpose                                             |
| ------------------ | --------------------------------------------------- |
| **stack**          | Bundle composition + Materialize projection builder |
| **stack/memory**   | All-in-memory preset (tests & dev)                  |
| **stack/sqlite**   | Embedded SQLite preset (single-file persistence)    |
| **stack/pebble**   | Embedded PebbleDB preset (high-throughput KV)       |
| **stack/postgres** | PostgreSQL preset (distributed, LISTEN/NOTIFY bus)  |
| **stack/turso**    | Turso preset (offline-first with sync)              |

### Examples

| Module                      | Purpose                                                               |
| --------------------------- | --------------------------------------------------------------------- |
| **example/taskmanager**     | Flagship HTTP service: full CQRS/ES pipeline, signing, SSE, snapshots |
| **example/getting-started** | Minimal demo: bundle → repository → commands → projection → query     |

## Design Principles

1. **Library, not framework** — Import what you need. No opinionated transport, broker, or driver.
2. **Concrete types where it matters** — `event.Event = *ImmutableEvent` (no interface indirection on hot paths).
3. **Composition over inheritance** — Per Go best practices.
4. **Strong types** — Branded IDs, no `any` in public API (except DB interop).
5. **Errors as values** — No panics in production paths; 5-family error taxonomy.
6. **Multi-module isolation** — Each module has its own `go.mod` with only needed deps.
7. **Tombstone over delete** — Soft-delete via metadata; no `Delete` on Store.

## Why this library?

Most Go CQRS libraries are **frameworks** — they own your transport, your broker, your SQL driver, your project structure. go-cqrs-lite is a **library**: 42+ independent modules with their own `go.mod` files. You import only what you need and compose your own stack. Nothing is hidden behind magic.

**What makes it different:**

- **Event Sourcing first-class** — Immutable events, branded IDs, optimistic concurrency, tombstone soft-delete, time-travel queries, schema evolution via upcasters. Not an afterthought bolted onto a CRUD library.
- **Library, not framework** — No transport, broker, or SQL driver is forced on you. Use Watermill, standard `net/http`, gRPC, NATS — your choice. The `stack/` presets wire sensible defaults if you want zero-config.
- **SQL-backed read models** — `SQLViewStore` gives each projection its own table with real, queryable columns. Server-side WHERE, ORDER BY, pagination, indexes, and COUNT — impossible with opaque KV-blob read models.
- **Multi-module isolation** — Each module has its own `go.mod` with minimal deps. Import `event` alone (3 deps) or the full `stack/sqlite` preset. Your dependency tree stays clean.
- **Production primitives** — Event signing (HMAC, Ed25519, multisig), payload encryption (XChaCha20-Poly1305, AES-256-GCM), OTel tracing+metrics, Prometheus bridge. Not stubs.
- **Honest error taxonomy** — 5-family classification (Rejection / Conflict / Transient / Infrastructure / Corruption) with sentinel errors and `%w` wrapping. No panics in production paths.

## Security & Operations

### Event Encryption

Encrypt event payloads at rest with AEAD ciphers. Key rotation via `keyID` stamps.

```go
enc, _ := encryption.NewXChaCha20Poly1305(key)
bus.UsePublish(encryption.EncryptMiddleware(enc, encryption.WithMiddlewareKeyID("v1")))
bus.Use(encryption.DecryptMiddleware(enc))
```

Supports XChaCha20-Poly1305 (recommended), AES-256-GCM, HKDF key derivation for multi-tenant setups, and a composable codec wrapper. See [`encryption/README.md`](encryption/README.md).

### Turso: Embedded SQLite with Remote Sync

`storage/turso` provides an embedded Turso Database (SQLite-compatible) with optional remote replication. The indexing advisor analyzes your query patterns and recommends optimal indexes.

```go
store, _ := turso.NewStore(db, turso.WithRemoteSync("libsql://your-db.turso.io", token))
advisor := turso.NewIndexingAdvisor(db)
recommendations, _ := advisor.Analyze(ctx, slowQueries)
```

See [`storage/turso/README.md`](storage/turso/README.md) for the full API.

### Test Utilities

`testutil` provides shared helpers for writing concise test code across all CQRS modules:

```go
cmd := testutil.NewCmd(t, "user.create", aggID)
handler := testutil.NoopCommandHandler()
```

See [`testutil/README.md`](testutil/README.md) for the full list of helpers.

## Comparison

| Capability                          | go-cqrs-lite | go-cqrs | Watermill  | cqrs-go |
| ----------------------------------- | :----------: | :-----: | :--------: | :-----: |
| **Library (not framework)**         |      ✅      |   ❌    |  Partial   |   ❌    |
| **Event Sourcing**                  |      ✅      |   ✅    | Via plugin |   ✅    |
| **Per-module go.mod**               |      ✅      |   ❌    |     ❌     |   ❌    |
| **Branded IDs**                     |      ✅      |   ❌    |     ❌     |   ❌    |
| **Event signing**                   |      ✅      |   ❌    |     ❌     |   ❌    |
| **Event encryption**                |      ✅      |   ❌    |     ❌     |   ❌    |
| **Schema evolution**                |      ✅      |   ❌    |     ❌     |   ❌    |
| **Auto-docs (AsyncAPI/OpenAPI/D2)** |      ✅      |   ❌    |     ❌     |   ❌    |
| **SQL-backed read models**          |      ✅      |   ❌    |     ❌     |   ❌    |
| **Tombstone soft-delete**           |      ✅      |   ❌    |     ❌     |   ❌    |
| **Bundle presets**                  |      ✅      |   ❌    |     ❌     |   ❌    |

## Status

**v3.0.0 released** — 42+ modules on `/v3` import paths, 84–100% test coverage on core modules. SQL-backed read models, gRPC transport, SSE delivery, event signing/encryption, and 5 stack presets shipped.

**Migrating from v2?** Read the **[v3 Migration Guide](docs/migration/V3_MIGRATION.md)** — all changes are additive, and import paths move from `…/v2` to `…/v3`.

See [FEATURES.md](FEATURES.md) for the full feature inventory, [ROADMAP.md](ROADMAP.md) for direction, and [docs/](docs/) for architecture decisions (ADRs), benchmarks, and storage guides.

## License

MIT
