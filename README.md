<h1 align="center">go-cqrs-lite</h1>

<p align="center"><strong>CQRS and Event Sourcing for Go — without the framework tax.</strong></p>

<p align="center">
<a href="https://pkg.go.dev/github.com/larsartmann/go-cqrs-lite/event/v4"><img src="https://pkg.go.dev/badge/github.com/larsartmann/go-cqrs-lite/event/v4.svg" alt="Go Reference"></a>
<a href="https://github.com/LarsArtmann/go-cqrs-lite/actions/workflows/ci.yml"><img src="https://github.com/LarsArtmann/go-cqrs-lite/actions/workflows/ci.yml/badge.svg" alt="CI"></a>
<a href="LICENSE"><img src="https://img.shields.io/badge/license-Proprietary-red.svg" alt="License: Proprietary"></a>
</p>

<p align="center">
<a href="SKILL.md">Guide (SKILL.md)</a> · <a href="https://pkg.go.dev/github.com/larsartmann/go-cqrs-lite/event/v4">API Reference</a> · <a href="example/getting-started/">Getting Started</a> · <a href="docs/">Docs</a>
</p>

---

A composable library of 80+ independently-versioned modules. Import exactly what you need: nothing is forced on you — no transport, no broker, no database driver. Wire your own stack, or grab a zero-config preset.

> Using this library with an AI assistant? [`SKILL.md`](SKILL.md) is the single-source guide — module decision matrix, copy-paste recipes, and conventions.

## Why go-cqrs-lite?

Most Go CQRS libraries are **frameworks** — they own your transport, your broker, your SQL driver, and your project layout. go-cqrs-lite is a **library**. You import only what you need and compose your own stack. Nothing is hidden behind magic.

- **Event Sourcing is first-class** — immutable events, branded IDs, optimistic concurrency, time-travel queries, and schema evolution via upcasters. Not an afterthought bolted onto a CRUD layer.
- **Library, not framework** — no transport, broker, or driver is forced on you. Use standard `net/http`, gRPC, Watermill, NATS — your choice. The `stack/` presets wire sensible defaults when you want zero-config.
- **Pure-Go by default** — SQLite, Pebble, and bbolt engines need no C compiler. CGo is quarantined inside the single DuckDB module; everyone else never notices.
- **Multi-module isolation** — each module has its own `go.mod` with minimal deps. Import `event` alone (3 dependencies) or the full `stack/sqlite` preset. Your dependency tree stays clean.
- **Production primitives, not stubs** — event signing (HMAC-SHA256, Ed25519, multisig), payload encryption (XChaCha20-Poly1305, AES-256-GCM, key rotation), OTel tracing and metrics, and a Prometheus bridge.
- **Honest error taxonomy** — a 6-family classification (Rejection / Conflict / Transient / Infrastructure / Orchestration / Corruption) with sentinel errors and `%w` wrapping. No panics in production paths.
- **Strong types throughout** — branded IDs make it impossible to mix up an `OrderID` with a `UserID`. The type system catches mistakes the compiler can express.
- **SQL-backed read models** — `SQLViewStore` gives each projection its own table with real, queryable columns: server-side `WHERE`, `ORDER BY`, pagination, indexes, and `COUNT`. Opaque KV-blob read models cannot do this. (Deprecated in v5 — metaengine auto-projection replaces it.)

## Who is this for?

- **Go backend engineers adopting event sourcing** who want proven primitives — stores, buses, upcasters, snapshots — without surrendering their project layout to a framework.
- **DDD practitioners** who model behavior as pure functions (the Decider pattern) and want optimistic concurrency and branded IDs out of the box.
- **Platform engineers embedding CQRS into existing services** — import the 3-dependency `event` module alone, or compose upward as needs grow.
- **Teams that choose storage at deployment time** — swap SQLite, Postgres, MySQL, Pebble, bbolt, Dgraph, or DuckDB behind the same domain code.
- **AI-assisted development teams** — [`SKILL.md`](SKILL.md) gives coding agents a verified, single-source API guide instead of hallucinated APIs.

## How it compares

| Capability                                  | go-cqrs-lite | Hand-rolled (stdlib) | [looplab/eventhorizon](https://github.com/looplab/eventhorizon) | [ThreeDotsLabs/watermill](https://github.com/ThreeDotsLabs/watermill) |
| ------------------------------------------- | :----------: | :------------------: | :-------------------------------------------------------------: | :-------------------------------------------------------------------: |
| **Library (not framework)**                 |      ✓       |          ✓           |                             Partial                             |                                Partial                                |
| **Event sourcing**                          |      ✓       |                      |                                ✓                                |                              Via plugins                              |
| **Per-module go.mod**                       |      ✓       |                      |                                ✗                                |                                   ✗                                   |
| **Branded, mix-up-proof IDs**               |      ✓       |                      |                                ✗                                |                                   ✗                                   |
| **Event signing** (HMAC, Ed25519, multisig) |      ✓       |                      |                                ✗                                |                                   ✗                                   |
| **Payload encryption** (XChaCha20, AES-GCM) |      ✓       |                      |                                ✗                                |                                   ✗                                   |
| **Schema evolution** (upcasters)            |      ✓       |                      |                                ✗                                |                                   ✗                                   |
| **Auto-docs** (AsyncAPI, OpenAPI, D2)       |      ✓       |                      |                                ✗                                |                                   ✗                                   |
| **Managed projection host** (crash-restart) |      ✓       |                      |                             Partial                             |                                   ✗                                   |
| **One-call storage presets**                |      ✓       |                      |                                ✗                                |                                   ✗                                   |

An empty cell means "you build it yourself." Claims verified against each project's repository (August 2026) — the links are there so you can check the cells.

## When NOT to use this

Skip this library if:

- **Your app is plain CRUD without domain events** — `database/sql`, sqlc, GORM, or ent is simpler. Event sourcing adds ceremony that CRUD does not need.
- **You want a framework that owns transport and project layout** — application frameworks in the go-zero/Kratos style do that; go-cqrs-lite deliberately owns neither.
- **You need an ops-heavy event store** — server-side projections, persistent subscriptions, a query UI, multi-node clustering. [KurrentDB](https://github.com/EventStore/EventStore) (EventStoreDB) is a purpose-built server; this is an embedded library.
- **Your problem is messaging, not domain aggregates** — plain [Watermill](https://github.com/ThreeDotsLabs/watermill) is lighter. You can adopt our `watermill/` adapter later if domain events join the picture.

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
    streamID := id.NewStreamID()
    _ = command.RegisterTyped(cmds, "user.create",
        func(ctx context.Context, cmd *CreateUser) error {
            return repo.Execute(ctx, cmd.StreamID(), "User", func(s UserState, v event.Version) ([]event.Event, error) {
                return event.NewEvents(cmd.StreamID(), "User", v,
                    []event.Type{"user.created"}, []any{UserCreated{Name: cmd.Name}})
            })
        })

    basic, _ := command.New("user.create", streamID)
    _ = cmds.Dispatch(ctx, &CreateUser{BasicCommand: basic, Name: "Alice"})

    state, _, _ := repo.Load(ctx, streamID, "User")
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

Seven presets cover every deployment shape (all deprecated in v5 — `system.System` becomes the one composition root, see the heads-up below):

| Preset           | When to use                                       |
| ---------------- | ------------------------------------------------- |
| `stack/memory`   | Tests and local dev — everything in RAM           |
| `stack/sqlite`   | Embedded single-file persistence                  |
| `stack/pebble`   | High-throughput embedded KV (PebbleDB + CBOR)     |
| `stack/duckdb`   | Embedded columnar OLAP (analytical workloads)     |
| `stack/postgres` | Distributed, with connection pooling + timeouts   |
| `stack/mysql`    | MySQL/MariaDB with pure-Go driver                 |
| `stack/turso`    | Embedded Turso Database with optional remote sync |

> **Heads-up (v5):** the `stack/*` presets and the v1 read-model tiers
> (`stack.Materialize`, `SQLViewStore`, `RelationalProjection`,
> `GraphProjection`) are deprecated and will be **removed in v5**
> ([ADR-0123](docs/adr/0123-v5-unification-single-composition-root.md)) —
> `system.System` becomes the single composition root and metaengine
> auto-projection serves read models. Everything above works unchanged
> through v4.x; new projects can already adopt `system/` + `metaengine`.

See [`example/getting-started/`](example/getting-started/) for a single-file tour of the core loop (event sourcing + projection + materialized view), and [`example/taskmanager/`](example/taskmanager/) for a complete HTTP service (CQRS/ES, projections, signing, SSE, snapshots).

## Key modules

Every module is independently importable and has its own `go.mod`. Here are the most important ones — see [AGENTS.md](AGENTS.md) for the full module catalog and [FEATURES.md](FEATURES.md) for the feature inventory.

| Module             | Purpose                                                               |
| ------------------ | --------------------------------------------------------------------- |
| **event**          | Immutable events, store/bus interfaces, event sourcing                |
| **command**        | Typed command dispatch, middleware, audit journal, pub/sub bus        |
| **query**          | Typed query dispatch, pagination, audit journal                       |
| **decider**        | Pure-function event-sourcing pattern: load, apply, decide, save       |
| **id**             | Branded IDs backed by ULID                                            |
| **storage**        | SQL event/snapshot/checkpoint stores (PostgreSQL, SQLite)             |
| **storage/pebble** | Embedded KV: event/snapshot/checkpoint stores (PebbleDB + CBOR)       |
| **projectionhost** | Managed projection lifecycle: crash-restart, checkpoints, dead-letter |
| **middleware**     | Logging, retry, validation, recovery, circuit breaker, OTel           |
| **kv**             | Layer-0 KV abstraction: `Store`, `TypedStore[T,K]`, `Cache`           |
| **catalog**        | Auto-generate AsyncAPI 3.0, EventCatalog, OpenAPI, D2 from Go types   |
| **stack/sqlite**   | One-call preset: SQLite + event bus + read models + projections       |

## Key dependencies

Each module declares its own `go.mod`; this is the greatest-hits across the library:

| Dependency                                                                      | Where                 | Purpose                               |
| ------------------------------------------------------------------------------- | --------------------- | ------------------------------------- |
| [`ThreeDotsLabs/watermill`](https://github.com/ThreeDotsLabs/watermill)         | `watermill/`, presets | In-process and broker pub/sub         |
| [`failsafe-go/failsafe-go`](https://github.com/failsafe-go/failsafe-go)         | `middleware/`         | Circuit breaker                       |
| [`maypok86/otter/v2`](https://github.com/maypok86/otter)                        | `decider/`            | TinyLFU state cache                   |
| `golang.org/x/crypto`                                                           | `encryption/`         | XChaCha20-Poly1305 payload encryption |
| `modernc.org/sqlite`                                                            | SQLite engines        | CGo-free SQLite driver                |
| [`larsartmann/go-error-family`](https://github.com/larsartmann/go-error-family) | all core modules      | 6-family error taxonomy               |

## Maturity

80+ modules on `/v4` import paths. Core modules carry 82–97% test coverage (event 88%, decider 97%, id 86%, dispatcher 82%). The library covers the full CQRS/ES lifecycle: event sourcing with branded IDs, command/query dispatch, pure-function deciders, three projection tiers (document/KV, relational/SQL, graph), durable deadline scheduling, dead-letter quarantine, managed projection hosting, event signing and encryption, OTel tracing and metrics, auto-documentation generation, and a domain-aware linter (cqrs-lint).

**Migrating from v3?** Read the **[Migration Guide](docs/migration/MIGRATION-GUIDE.md)** — covers the v4 breaking changes (codec defaults, API cleanup, path migration). For v2-to-v3 changes, see the **[v3 Migration Guide](docs/migration/V3_MIGRATION.md)**.

For the full feature inventory see [FEATURES.md](FEATURES.md), for direction see [ROADMAP.md](ROADMAP.md), and for architecture decisions, benchmarks, and storage guides see [docs/](docs/).

## License

PROPRIETARY — see [LICENSE](LICENSE).
