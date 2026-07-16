# go-cqrs-lite

[![CI](https://github.com/LarsArtmann/go-cqrs-lite/actions/workflows/ci.yml/badge.svg)](https://github.com/LarsArtmann/go-cqrs-lite/actions/workflows/ci.yml)
[![Go Reference](https://pkg.go.dev/badge/github.com/larsartmann/go-cqrs-lite/event/v4.svg)](https://pkg.go.dev/github.com/larsartmann/go-cqrs-lite/event/v4)

**CQRS and Event Sourcing for Go — without the framework tax.**

A composable library of 52 independent modules. Import exactly what you need: nothing is forced on you — no transport, no broker, no database driver. Wire your own stack, or grab a zero-config preset. Every module ships with its own `go.mod`, so your dependency tree stays as lean as you want it.

> Using this library with an AI assistant? [`SKILL.md`](SKILL.md) is the single-source guide — module decision matrix, copy-paste recipes, and conventions.

## Quick Start

```bash
go get github.com/larsartmann/go-cqrs-lite/event/v4
```

```go
package main

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

type UserState struct{ Name string }
type UserCreated struct{ Name string }

// CreateUser implements command.Command via embedded *BasicCommand.
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

## Production in one call

When you are ready for persistent storage, a wired event bus, read models, and projections, take a **stack preset** — one import, one function call:

```go
import "github.com/larsartmann/go-cqrs-lite/stack/sqlite/v4"

// Events, commands, queries, snapshots, checkpoints, read models — all persisted.
// Event bus wired (watermill GoChannel for in-process pub/sub).
bundle, err := sqlite.New("app.db")
// Use bundle.EventStore, bundle.Bus, bundle.ReadModels, bundle.CommandDispatcher...
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

See [`example/taskmanager/`](example/taskmanager/) for a complete end-to-end HTTP service (CQRS/ES, projections, signing, SSE, snapshots) and [`example/getting-started/`](example/getting-started/) for a minimal 80-line tour of the core loop.

## Why go-cqrs-lite?

Most Go CQRS libraries are **frameworks** — they own your transport, your broker, your SQL driver, and your project layout. go-cqrs-lite is a **library**. You import only what you need and compose your own stack. Nothing is hidden behind magic.

- **Event Sourcing is first-class** — immutable events, branded IDs, optimistic concurrency, tombstone soft-delete, time-travel queries, and schema evolution via upcasters. Not an afterthought bolted onto a CRUD layer.
- **Library, not framework** — no transport, broker, or driver is forced on you. Use standard `net/http`, gRPC, Watermill, NATS — your choice. The `stack/` presets wire sensible defaults when you want zero-config.
- **SQL-backed read models** — `SQLViewStore` gives each projection its own table with real, queryable columns: server-side `WHERE`, `ORDER BY`, pagination, indexes, and `COUNT`. Opaque KV-blob read models cannot do this.
- **Multi-module isolation** — each module has its own `go.mod` with minimal deps. Import `event` alone (3 dependencies) or the full `stack/sqlite` preset. Your dependency tree stays clean.
- **Production primitives, not stubs** — event signing (HMAC-SHA256, Ed25519, multisig), payload encryption (XChaCha20-Poly1305, AES-256-GCM, key rotation), OTel tracing and metrics, and a Prometheus bridge.
- **Honest error taxonomy** — a 5-family classification (Rejection / Conflict / Transient / Infrastructure / Corruption) with sentinel errors and `%w` wrapping. No panics in production paths.
- **Strong types throughout** — branded IDs make it impossible to mix up an `OrderID` with a `UserID`. The type system catches mistakes the compiler can express.

## Module catalog

Every module is independently importable and has its own `go.mod`. Modules with detailed docs link to a README.

### Core domain

| Module       | Purpose                                                                | Docs                        |
| ------------ | ---------------------------------------------------------------------- | --------------------------- |
| **event**    | Immutable events, store/bus interfaces, event sourcing, error taxonomy | [README](event/README.md)   |
| **command**  | Typed command dispatch, middleware, audit journal, pub/sub bus         | [README](command/README.md) |
| **query**    | Typed query dispatch, pagination, audit journal                        | [README](query/README.md)   |
| **decider**  | Pure-function aggregate pattern: load → apply → decide → save          | [README](decider/README.md) |
| **deriver**  | Event-to-command derivation pipelines (sagas, reactions)               |                             |
| **id**       | Branded IDs backed by ULID — impossible to mix up ID types             | [README](id/README.md)      |
| **metadata** | Shared typed metadata (tracing, causation, tombstone) for all messages |                             |

### Persistence

| Module             | Purpose                                                         | Docs                              |
| ------------------ | --------------------------------------------------------------- | --------------------------------- |
| **storage/memory** | In-memory store/snapshot/checkpoint — tests and dev             |                                   |
| **storage**        | SQL event/snapshot/checkpoint/command/query stores (PG, SQLite) | [README](storage/README.md)       |
| **storage/pebble** | Embedded KV: event/snapshot/checkpoint stores (PebbleDB + CBOR) |                                   |
| **storage/turso**  | Turso connector with offline-first sync and indexing advisor    | [README](storage/turso/README.md) |
| **kv**             | Layer-0 KV abstraction: `Store`, `TypedStore[T,K]`, `Cache`     | [README](kv/README.md)            |
| **snapshot**       | Snapshot types, strategies, and store interfaces                | [README](snapshot/README.md)      |
| **schema**         | Schema evolution via upcasters and `VersionedStore`             | [README](schema/README.md)        |

### Projections and read models

| Module             | Purpose                                                                  | Docs                               |
| ------------------ | ------------------------------------------------------------------------ | ---------------------------------- |
| **projection**     | The consumer-side `Projection` contract (`Name`, `Handle`, `EventTypes`) |                                    |
| **projectionhost** | Managed host: crash-restart lifecycle, checkpoints, dead-letter queue    | [README](projectionhost/README.md) |
| **graph**          | Graph projection tier: nodes + edges for traversal-heavy read models     | [README](graph/README.md)          |
| **listing**        | Aggregate listing read model with tombstone-aware status                 |                                    |

### Infrastructure

| Module             | Purpose                                                           | Docs                            |
| ------------------ | ----------------------------------------------------------------- | ------------------------------- |
| **middleware**     | Logging, retry, validation, recovery, circuit breaker, OTel       |                                 |
| **transport/http** | SSE event delivery (Server-Sent Events over HTTP)                 |                                 |
| **transport/grpc** | gRPC command and query dispatch                                   |                                 |
| **signing**        | Event signing: HMAC-SHA256, Ed25519, multisig                     | [README](signing/README.md)     |
| **encryption**     | Payload encryption: XChaCha20-Poly1305, AES-256-GCM, key rotation | [README](encryption/README.md)  |
| **watermill**      | EventBus adapter, `CatchUpSubscriber`, `EventPublisher`           | [README](watermill/README.md)   |
| **otel**           | Shared OpenTelemetry helpers (tracer, meter, spans)               | [README](otel/README.md)        |
| **prometheus**     | OTel-to-Prometheus metrics bridge with `/metrics` handler         | [README](prometheus/README.md)  |
| **idempotency**    | Dedup store for at-least-once delivery retries                    | [README](idempotency/README.md) |
| **dedup**          | Bounded dedup ring buffer for stream boundaries                   |                                 |
| **scheduling**     | Durable deadline timers ("cancel order after 30 min unpaid")      |                                 |
| **codec**          | Payload encoding: JSON, deterministic CBOR, Raw passthrough       | [README](codec/README.md)       |
| **dispatcher**     | Generic `Dispatcher[H, M]` with lifecycle management              | [README](dispatcher/README.md)  |

### Composition

| Module             | Purpose                                               |
| ------------------ | ----------------------------------------------------- |
| **stack**          | Bundle composition + `Materialize` projection builder |
| **stack/memory**   | All-in-memory preset (tests and dev)                  |
| **stack/sqlite**   | Embedded SQLite preset (single-file persistence)      |
| **stack/pebble**   | Embedded PebbleDB preset (high-throughput KV)         |
| **stack/postgres** | PostgreSQL preset (distributed, `LISTEN/NOTIFY`)      |
| **stack/turso**    | Turso preset (offline-first with sync)                |

### Tooling and testing

| Module                | Purpose                                                             | Docs                                  |
| --------------------- | ------------------------------------------------------------------- | ------------------------------------- |
| **catalog**           | Auto-generate AsyncAPI 3.0, EventCatalog, OpenAPI, D2 from Go types | [README](catalog/README.md)           |
| **scenario**          | Fluent BDD DSL: `Given`/`When`/`Then` for deciders and projections  |                                       |
| **testutil**          | Shared test helpers: `NewCmd`, `NoopCommandHandler`                 | [README](testutil/README.md)          |
| **cmd/cqrs-gen**      | Code generator: typed handler registration from `//cqrs:` markers   | [README](cmd/cqrs-gen/README.md)      |
| **cmd/api-stability** | API surface checker: compare exports against a golden file          | [README](cmd/api-stability/README.md) |
| **cmd/doc-check**     | Doc cross-reference verifier for markdown                           |                                       |
| **integration**       | Cross-module integration tests                                      | [README](integration/README.md)       |

## Module dependency graph

Modules are organized in tiers (ADR-0046). Each tier may only import from its own tier or lower.

```mermaid
graph TD
    subgraph T0["Tier 0 — Primitives"]
        id[id]
        codec[codec]
        kv[kv]
        dedup[dedup]
        dispatcher[dispatcher]
    end

    subgraph T1["Tier 1 — Core Domain"]
        event[event]
        command[command]
        query[query]
        scheduling[scheduling]
        metadata[metadata]
    end

    subgraph T2["Tier 2 — Utilities"]
        schema[schema]
        snapshot[snapshot]
        projection[projection]
        idempotency[idempotency]
        deriver[deriver]
    end

    subgraph T3["Tier 3 — Aggregation"]
        decider[decider]
        projectionhost[projectionhost]
        listing[listing]
        graphmod["graph"] %% node ID can't be 'graph' — mermaid keyword
        scenario[scenario]
    end

    subgraph T4["Tier 4 — Infrastructure"]
        storage[storage]
        pebble[storage/pebble]
        memory[storage/memory]
        middleware[middleware]
        signing[signing]
        encryption[encryption]
        otel[otel]
        watermill[watermill]
        http[transport/http]
        grpc[transport/grpc]
        prometheus[prometheus]
    end

    subgraph T5["Tier 5 — Composition"]
        stack[stack]
        sqlite[stack/sqlite]
        postgres[stack/postgres]
        turso[stack/turso]
    end

    event --> id & codec & metadata
    command --> event
    query --> event

    schema --> event
    snapshot --> event
    deriver --> event & command

    decider --> event & snapshot & schema
    projectionhost --> projection & event & dedup

    storage --> event & kv & snapshot
    middleware --> event & command & query
    signing --> event
    watermill --> event & command

    stack --> storage & decider & middleware
    sqlite --> stack
```

**Quick start:** Need events only? → `event` + `id`. Need aggregates? → add `decider`. Need projections? → add `projectionhost` + a storage module. Need everything wired? → use `stack/sqlite`.

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

52 modules on `/v4` import paths. Core modules carry 84–98% test coverage (event 91%, decider 98%, id 97%, dispatcher 98%). The library covers the full CQRS/ES lifecycle: event sourcing with branded IDs, command/query dispatch, pure-function deciders, three projection tiers (document/KV, relational/SQL, graph), durable deadline scheduling, dead-letter quarantine, managed projection hosting, event signing and encryption, OTel tracing and metrics, auto-documentation generation, and a domain-aware linter (cqrs-lint).

**Migrating from v3?** Read the **[Migration Guide](docs/migration/MIGRATION-GUIDE.md)** — covers the v4 breaking changes (codec defaults, API cleanup, path migration). For v2→v3 changes, see the **[v3 Migration Guide](docs/migration/V3_MIGRATION.md)**.

For the full feature inventory see [FEATURES.md](FEATURES.md), for direction see [ROADMAP.md](ROADMAP.md), and for architecture decisions, benchmarks, and storage guides see [docs/](docs/).

## License

PROPRIETARY — see [LICENSE](LICENSE). Public release under Apache-2.0 is planned (see [ROADMAP.md](ROADMAP.md)).
