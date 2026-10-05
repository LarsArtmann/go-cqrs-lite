<h1 align="center">go-cqrs-lite</h1>

<p align="center"><strong>CQRS and Event Sourcing for Go — without the framework tax.</strong></p>

<p align="center">
<a href="https://pkg.go.dev/github.com/larsartmann/go-cqrs-lite/event/v4"><img src="https://pkg.go.dev/badge/github.com/larsartmann/go-cqrs-lite/event/v4.svg" alt="Go Reference"></a>
<a href="https://github.com/LarsArtmann/go-cqrs-lite/actions/workflows/ci.yml"><img src="https://github.com/LarsArtmann/go-cqrs-lite/actions/workflows/ci.yml/badge.svg" alt="CI"></a>
<a href="LICENSE"><img src="https://img.shields.io/badge/license-Proprietary-red.svg" alt="License: Proprietary"></a>
<a href="https://github.com/larsartmann/templ-components"><img src="https://img.shields.io/badge/GOTH-stack-8A2BE2?style=flat-square" alt="Part of the GOTH stack"></a>
</p>

<p align="center">
<a href="SKILL.md">Guide (SKILL.md)</a> · <a href="https://pkg.go.dev/github.com/larsartmann/go-cqrs-lite/event/v4">API Reference</a> · <a href="example/getting-started/">Getting Started</a> · <a href="docs/">Docs</a>
</p>

---

A composable library of 90+ independently-versioned modules. Import exactly what you need: nothing is forced on you — no transport, no broker, no database driver. Wire your own stack, or grab a zero-config preset.

> Using this library with an AI assistant? [`SKILL.md`](SKILL.md) is the single-source guide — module decision matrix, copy-paste recipes, and conventions.

> **Part of the GOTH stack** — pair with
> [templ-components](https://github.com/larsartmann/templ-components) (UI components
> for templ + HTMX + Tailwind v4) and
> [cqrs-htmx](https://github.com/larsartmann/cqrs-htmx) (HTTP → CQRS wiring, auth,
> HTMX response building) for a complete server-rendered Go web stack with zero
> framework lock-in.

## Why go-cqrs-lite?

Most Go CQRS libraries are **frameworks** — they own your transport, your broker, your SQL driver, and your project layout. go-cqrs-lite is a **library**. You import only what you need and compose your own stack. Nothing is hidden behind magic.

- **Event Sourcing is first-class** — immutable events, branded IDs, optimistic concurrency, time-travel queries, and schema evolution via upcasters. Not an afterthought bolted onto a CRUD layer.
- **Library, not framework** — no transport, broker, or driver is forced on you. Use standard `net/http`, gRPC, Watermill, NATS — your choice. The `stack/` presets wire sensible defaults when you want zero-config.
- **Pure-Go by default** — SQLite, Pebble, and bbolt engines need no C compiler. CGo is quarantined inside the single DuckDB module; everyone else never notices.
- **Multi-module isolation** — each module has its own `go.mod` with minimal deps. Import `event` alone (10 module deps — only 3 third-party) or the full `stack/sqlite` preset. Your dependency tree stays clean.
- **Production primitives, not stubs** — event signing (HMAC-SHA256, Ed25519, multisig), payload encryption (XChaCha20-Poly1305, AES-256-GCM, key rotation), OTel tracing and metrics, and a Prometheus bridge.
- **Honest error taxonomy** — a 6-family classification (Rejection / Conflict / Transient / Infrastructure / Orchestration / Corruption) with sentinel errors and `%w` wrapping. No panics in production paths.
- **Strong types throughout** — branded IDs make it impossible to mix up an `OrderID` with a `UserID`. The type system catches mistakes the compiler can express.
- **SQL-backed read models** — `SQLViewStore` gives each projection its own table with real, queryable columns: server-side `WHERE`, `ORDER BY`, pagination, indexes, and `COUNT`. Opaque KV-blob read models cannot do this. (Deprecated in v5 — metaengine auto-projection replaces it.)

## Who is this for?

- **Go backend engineers adopting event sourcing** who want proven primitives — stores, buses, upcasters, snapshots — without surrendering their project layout to a framework.
- **DDD practitioners** who model behavior as pure functions (the Decider pattern) and want optimistic concurrency and branded IDs out of the box.
- **Platform engineers embedding CQRS into existing services** — import the `event` module alone (3 third-party deps, the rest are first-party larsartmann modules), or compose upward as needs grow.
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

Modules release on independent version trains. If you pin several, upgrade with the supported sweep pattern (per-module `go get …@latest` + `go mod tidy` + hermetic `GOWORK=off go build`/`go vet`, gated on tag existence) — see the skill FAQ's "How do I upgrade many go-cqrs-lite modules at once?". `cmd/cqrs-upgrade` scans your modules for v5-removed surfaces before you sweep.

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
            return repo.ExecuteRef(ctx, id.NewStreamRef("User", cmd.StreamID()),
                func(s UserState, v event.Version) ([]event.Event, error) {
                    return event.NewEvents(cmd.StreamID(), "User", v,
                        []event.Type{"user.created"}, []any{UserCreated{Name: cmd.Name}})
                })
        })

    basic, _ := command.New("user.create", streamID)
    _ = cmds.Dispatch(ctx, &CreateUser{BasicCommand: basic, Name: "Alice"})

    state, _, _ := repo.LoadRef(ctx, id.NewStreamRef("User", streamID))
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

Eight presets cover every deployment shape (all deprecated in v5 — `system.System` becomes the one composition root, see the heads-up below):

| Preset           | When to use                                       |
| ---------------- | ------------------------------------------------- |
| `stack/memory`   | Tests and local dev — everything in RAM           |
| `stack/sqlite`   | Embedded single-file persistence                  |
| `stack/pebble`   | High-throughput embedded KV (PebbleDB + CBOR)     |
| `stack/bbolt`    | Embedded single-file B+tree KV (pure Go)          |
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
> Canonical v5-removal list:
> [FAQ — "Will the v5 cut break my imports?"](.agents/skills/go-cqrs-lite/references/faq.md#will-the-v5-cut-break-my-imports-what-is-going-away).

See [`example/getting-started/`](example/getting-started/) for a single-file tour of the v5 composition path (`system.New` + metaengine folds, with a test that proves the one-line engine swap), [`example/metaengine-quickstart/`](example/metaengine-quickstart/) for the deployment-time story (declare folds and queries, then let the operator pick the engines in a `cqrs.yaml` — maps, graph, and vector demos included), and [`example/taskmanager/`](example/taskmanager/) for a complete HTTP service (CQRS/ES, projections, signing, SSE, snapshots).

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

90+ modules on `/v4` import paths (98 `go.mod` files incl. root — count gate-derived via `scripts/check-canonical-facts.sh`). Core modules carry 86–96% test coverage (event 90%, decider 96%, id 86%, dispatcher 87%). The library covers the full CQRS/ES lifecycle: event sourcing with branded IDs, command/query dispatch, pure-function deciders, three projection tiers (document/KV, relational/SQL, graph), durable deadline scheduling, dead-letter quarantine, managed projection hosting, event signing and encryption, OTel tracing and metrics, auto-documentation generation, and a domain-aware linter (cqrs-lint).

**Migrating from v3?** Read the **[Migration Guide](docs/migration/MIGRATION-GUIDE.md)** — covers the v4 breaking changes (codec defaults, API cleanup, path migration). For v2-to-v3 changes, see the **[v3 Migration Guide](docs/migration/V3_MIGRATION.md)**.

For the full feature inventory see [FEATURES.md](FEATURES.md), for direction see [ROADMAP.md](ROADMAP.md), and for architecture decisions, benchmarks, and storage guides see [docs/](docs/).

## Published versions

Modules release on independent per-module version trains (`<module>/vX.Y.Z` git tags). The manifest below is machine-readable as [`versions.json`](versions.json) (key `.` = the repo-root module train); both are regenerated by `bash scripts/check-versions-manifest.sh --update` and kept fresh by a nightly gate.

<details>
<summary>Module → latest published tag (generated — do not edit by hand)</summary>

<!-- versions-manifest:begin -->
| Module | Latest published tag |
| --- | --- |
| . | `v4.0.0` |
| benchkit | `benchkit/v4.6.2` |
| catalog | `catalog/v4.7.0` |
| claiming | `claiming/v4.0.2` |
| cmd/api-stability | `cmd/api-stability/v4.4.2` |
| cmd/cqrs-bench | `cmd/cqrs-bench/v4.3.3` |
| cmd/cqrs-gen | `cmd/cqrs-gen/v4.3.3` |
| cmd/cqrs-lint | `cmd/cqrs-lint/v4.14.0` |
| cmd/cqrs-upgrade | `cmd/cqrs-upgrade/v4.1.2` |
| cmd/doc-check | `cmd/doc-check/v4.3.3` |
| codec | `codec/v4.4.0` |
| command | `command/v4.13.1` |
| commandlifecycle | `commandlifecycle/v4.2.2` |
| commandlifecycle/projections | `commandlifecycle/projections/v4.2.2` |
| core | `core/v1.6.0` |
| cqrs-lite | `cqrs-lite/v0.1.1` |
| decider | `decider/v4.7.2` |
| dedup | `dedup/v4.2.4` |
| deriver | `deriver/v4.3.3` |
| dispatcher | `dispatcher/v4.5.2` |
| encryption | `encryption/v4.4.3` |
| event | `event/v4.13.1` |
| event/v3/eventtest | `event/v3/eventtest/v3.7.4` |
| event/v4/eventtest | `event/v4/eventtest/v0.4.0` |
| example/getting-started | `example/getting-started/v4.1.0` |
| example/goal-shaped-app | `example/goal-shaped-app/v0.1.2` |
| example/metaengine-quickstart | `example/metaengine-quickstart/v0.1.3` |
| example/readme-quickstart | `example/readme-quickstart/v0.2.3` |
| example/scheduler-otel-status | `example/scheduler-otel-status/v0.1.2` |
| example/taskmanager | `example/taskmanager/v3.7.1` |
| flightrecorder | `flightrecorder/v4.0.0` |
| graph | `graph/v4.3.3` |
| id | `id/v4.7.0` |
| idempotency | `idempotency/v4.4.0` |
| idempotency/kvstore | `idempotency/kvstore/v4.3.2` |
| idempotency/sqlstore | `idempotency/sqlstore/v4.4.2` |
| integration | `integration/v4.2.3` |
| kv | `kv/v4.3.3` |
| listing | `listing/v4.4.3` |
| memory | `memory/v2.6.0` |
| metadata | `metadata/v4.7.3` |
| metaengine | `metaengine/v4.16.1` |
| metaengine/badgerengine | `metaengine/badgerengine/v4.3.2` |
| metaengine/bboltengine | `metaengine/bboltengine/v4.3.2` |
| metaengine/bench | `metaengine/bench/v4.1.2` |
| metaengine/bigtableengine | `metaengine/bigtableengine/v4.0.2` |
| metaengine/dgraphengine | `metaengine/dgraphengine/v4.3.2` |
| metaengine/duckdbengine | `metaengine/duckdbengine/v4.3.2` |
| metaengine/graphadapter | `metaengine/graphadapter/v4.1.3` |
| metaengine/irohengine | `metaengine/irohengine/v4.3.2` |
| metaengine/irohengine/loopback | `metaengine/irohengine/loopback/v4.0.5` |
| metaengine/irohengine/quic | `metaengine/irohengine/quic/v4.2.3` |
| metaengine/mysqlengine | `metaengine/mysqlengine/v4.3.2` |
| metaengine/otelobserver | `metaengine/otelobserver/v4.0.2` |
| metaengine/pebbleengine | `metaengine/pebbleengine/v4.4.2` |
| metaengine/pgengine | `metaengine/pgengine/v4.4.2` |
| metaengine/projectionadapter | `metaengine/projectionadapter/v4.5.2` |
| metaengine/sqliteengine | `metaengine/sqliteengine/v4.5.1` |
| metaengine/tursoengine | `metaengine/tursoengine/v4.2.3` |
| middleware | `middleware/v4.7.2` |
| otel | `otel/v4.5.2` |
| otel/otlp | `otel/otlp/v4.0.2` |
| pebble | `pebble/v2.6.0` |
| projection | `projection/v4.4.2` |
| projectionhost | `projectionhost/v4.5.3` |
| prometheus | `prometheus/v4.3.3` |
| query | `query/v4.10.1` |
| queue | `queue/v4.0.2` |
| queue/mysql | `queue/mysql/v4.0.2` |
| queue/postgres | `queue/postgres/v4.0.2` |
| queue/sqlite | `queue/sqlite/v4.0.2` |
| record | `record/v4.6.2` |
| retry | `retry/v4.3.0` |
| saga | `saga/v1.0.0` |
| scenario | `scenario/v4.4.2` |
| scheduling | `scheduling/v4.6.0` |
| scheduling/engine | `scheduling/engine/v4.0.2` |
| scheduling/sqlstore | `scheduling/sqlstore/v4.1.3` |
| schema | `schema/v4.5.1` |
| signing | `signing/v4.3.4` |
| snapshot | `snapshot/v4.6.1` |
| stack | `stack/v4.4.3` |
| stack/bbolt | `stack/bbolt/v4.2.3` |
| stack/bench | `stack/bench/v4.3.2` |
| stack/duckdb | `stack/duckdb/v4.2.3` |
| stack/memory | `stack/memory/v4.4.3` |
| stack/mysql | `stack/mysql/v4.2.3` |
| stack/pebble | `stack/pebble/v4.4.3` |
| stack/postgres | `stack/postgres/v4.4.3` |
| stack/sqlite | `stack/sqlite/v4.3.4` |
| stack/turso | `stack/turso/v4.4.3` |
| storage | `storage/v4.10.4` |
| storage/backuptest | `storage/backuptest/v4.2.3` |
| storage/bbolt | `storage/bbolt/v4.2.3` |
| storage/memory | `storage/memory/v4.6.1` |
| storage/pebble | `storage/pebble/v4.4.3` |
| storage/turso | `storage/turso/v4.3.4` |
| sync | `sync/v0.2.0` |
| system | `system/v4.10.2` |
| system/integration | `system/integration/v4.0.2` |
| testhelpers | `testhelpers/v1.7.1` |
| testing | `testing/v3.3.0` |
| testutil | `testutil/v4.3.3` |
| testutil/pgtestcontainer | `testutil/pgtestcontainer/v4.2.3` |
| transport/grpc | `transport/grpc/v4.3.3` |
| transport/http | `transport/http/v4.3.4` |
| turso | `turso/v2.6.0` |
| watermill | `watermill/v4.6.4` |
<!-- versions-manifest:end -->

</details>

## License

PROPRIETARY — see [LICENSE](LICENSE).

Because the license is not OSS-approved, pkg.go.dev hides module documentation for every module by design (verified 2026-09-28; per-module LICENSE copies do not change this). Browse the API locally with `go doc <module>`, or read [SKILL.md](SKILL.md).
