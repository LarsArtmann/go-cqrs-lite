# go-cqrs-lite

[![CI](https://github.com/LarsArtmann/go-cqrs-lite/actions/workflows/ci.yml/badge.svg)](https://github.com/LarsArtmann/go-cqrs-lite/actions/workflows/ci.yml)
|[![Go Reference](https://pkg.go.dev/badge/github.com/larsartmann/go-cqrs-lite/event/v4.svg)](https://pkg.go.dev/github.com/larsartmann/go-cqrs-lite/event/v4)

**CQRS and Event Sourcing for Go — without the framework tax.**

A composable library of 52 independent modules. Import exactly what you need: nothing is forced on you — no transport, no broker, no database driver. Wire your own stack, or grab a zero-config preset.

> Using this library with an AI assistant? [`SKILL.md`](SKILL.md) is the single-source guide — module decision matrix, copy-paste recipes, and conventions.

---

## Install

```bash
go get github.com/larsartmann/go-cqrs-lite/event/v4
go get github.com/larsartmann/go-cqrs-lite/decider/v4
go get github.com/larsartmann/go-cqrs-lite/command/v4
go get github.com/larsartmann/go-cqrs-lite/id/v4
```

Each module has its own `go.mod` — import only what you need and your dependency tree stays lean.

## Quick Start (3 steps)

### 1. Define your domain

```go
type UserState struct{ Name string }
type UserCreated struct{ Name string }
```

### 2. Event-source with a Decider

The Decider pattern uses pure functions: load state, apply events, decide new events, save, publish.

```go
import (
    "context"
    "fmt"

    "github.com/larsartmann/go-cqrs-lite/command/v4"
    "github.com/larsartmann/go-cqrs-lite/decider/v4"
    "github.com/larsartmann/go-cqrs-lite/event/v4"
    "github.com/larsartmann/go-cqrs-lite/id/v4"
    "github.com/larsartmann/go-cqrs-lite/storage/memory/v4"
    cqrswatermill "github.com/larsartmann/go-cqrs-lite/watermill/v4"
)

type CreateUser struct {
    *command.BasicCommand
    Name string
}

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
    _ = command.RegisterTyped(cmds, "user.create",
        func(ctx context.Context, cmd *CreateUser) error {
            return repo.Execute(ctx, cmd.AggregateID(), "User", func(s UserState, v event.Version) ([]event.Event, error) {
                return event.NewEvents(cmd.AggregateID(), "User", v,
                    []event.Type{"user.created"}, []any{UserCreated{Name: cmd.Name}})
            })
        })

    basic, _ := command.New("user.create", aggID)
    _ = cmds.Dispatch(ctx, &CreateUser{BasicCommand: basic, Name: "Alice"})

    state, _, _ := repo.Load(ctx, aggID, "User")
    fmt.Printf("User: %s\n", state.Name) // User: Alice
}
```

### 3. Go to production with one call

Swap in-memory for persistent storage — change **one line**, keep the domain code identical:

```go
import "github.com/larsartmann/go-cqrs-lite/stack/sqlite/v4"

bundle, err := sqlite.New("app.db")
// Events, commands, queries, snapshots, checkpoints, read models — all persisted.
// Event bus wired (watermill GoChannel for in-process pub/sub).
defer bundle.Close()
```

Five presets cover every deployment shape:

| Preset           | When to use                                       |
| ---------------- | ------------------------------------------------- |
| `stack/memory`   | Tests and local dev — everything in RAM           |
| `stack/sqlite`   | Embedded single-file persistence                  |
| `stack/pebble`   | High-throughput embedded KV (PebbleDB + CBOR)     |
| `stack/postgres` | Distributed, with `LISTEN/NOTIFY` event bus       |
| `stack/turso`    | Embedded Turso Database with optional remote sync |

See [`example/getting-started/`](example/getting-started/) for a minimal 80-line tour of the core loop (event sourcing + projection + materialized view), and [`example/taskmanager/`](example/taskmanager/) for a complete HTTP service (CQRS/ES, projections, signing, SSE, snapshots).

---

## Why go-cqrs-lite?

Most Go CQRS libraries are **frameworks** — they own your transport, your broker, your SQL driver, and your project layout. go-cqrs-lite is a **library**. You import only what you need and compose your own stack. Nothing is hidden behind magic.

- **Event Sourcing is first-class** — immutable events, branded IDs, optimistic concurrency, tombstone soft-delete, time-travel queries, and schema evolution via upcasters. Not an afterthought bolted onto a CRUD layer.
- **Library, not framework** — no transport, broker, or driver is forced on you. Use standard `net/http`, gRPC, Watermill, NATS — your choice. The `stack/` presets wire sensible defaults when you want zero-config.
- **SQL-backed read models** — `SQLViewStore` gives each projection its own table with real, queryable columns: server-side `WHERE`, `ORDER BY`, pagination, indexes, and `COUNT`. Opaque KV-blob read models cannot do this.
- **Multi-module isolation** — each module has its own `go.mod` with minimal deps. Import `event` alone (3 dependencies) or the full `stack/sqlite` preset. Your dependency tree stays clean.
- **Production primitives, not stubs** — event signing (HMAC-SHA256, Ed25519, multisig), payload encryption (XChaCha20-Poly1305, AES-256-GCM, key rotation), OTel tracing and metrics, and a Prometheus bridge.
- **Honest error taxonomy** — a 5-family classification (Rejection / Conflict / Transient / Infrastructure / Corruption) with sentinel errors and `%w` wrapping. No panics in production paths.
- **Strong types throughout** — branded IDs make it impossible to mix up an `OrderID` with a `UserID`. The type system catches mistakes the compiler can express.

## Key modules

Every module is independently importable and has its own `go.mod`. Here are the most important ones — see [AGENTS.md](AGENTS.md) for the full 52-module catalog.

| Module             | Purpose                                                                |
| ------------------ | --------------------------------------------------------------------- |
| **event**          | Immutable events, store/bus interfaces, event sourcing                |
| **command**        | Typed command dispatch, middleware, audit journal, pub/sub bus        |
| **query**          | Typed query dispatch, pagination, audit journal                       |
| **decider**        | Pure-function aggregate pattern: load, apply, decide, save            |
| **id**             | Branded IDs backed by ULID                                            |
| **storage**        | SQL event/snapshot/checkpoint stores (PostgreSQL, SQLite)             |
| **storage/pebble** | Embedded KV: event/snapshot/checkpoint stores (PebbleDB + CBOR)       |
| **projectionhost** | Managed projection lifecycle: crash-restart, checkpoints, dead-letter |
| **middleware**     | Logging, retry, validation, recovery, circuit breaker, OTel           |
| **kv**             | Layer-0 KV abstraction: `Store`, `TypedStore[T,K]`, `Cache`           |
| **catalog**        | Auto-generate AsyncAPI 3.0, EventCatalog, OpenAPI, D2 from Go types   |
| **stack/sqlite**   | One-call preset: SQLite + event bus + read models + projections       |

## How it compares

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

## Maturity

52 modules on `/v4` import paths. Core modules carry 84-98% test coverage (event 91%, decider 98%, id 97%, dispatcher 98%). The library covers the full CQRS/ES lifecycle: event sourcing with branded IDs, command/query dispatch, pure-function deciders, three projection tiers (document/KV, relational/SQL, graph), durable deadline scheduling, dead-letter quarantine, managed projection hosting, event signing and encryption, OTel tracing and metrics, auto-documentation generation, and a domain-aware linter (cqrs-lint).

**Migrating from v3?** Read the **[Migration Guide](docs/migration/MIGRATION-GUIDE.md)** — covers the v4 breaking changes (codec defaults, API cleanup, path migration). For v2-to-v3 changes, see the **[v3 Migration Guide](docs/migration/V3_MIGRATION.md)**.

For the full feature inventory see [FEATURES.md](FEATURES.md), for direction see [ROADMAP.md](ROADMAP.md), and for architecture decisions, benchmarks, and storage guides see [docs/](docs/).

## License

PROPRIETARY — see [LICENSE](LICENSE). Public release under Apache-2.0 is planned (see [ROADMAP.md](ROADMAP.md)).
