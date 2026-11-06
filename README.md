<h1 align="center">go-cqrs-lite</h1>

<p align="center"><strong>CQRS and Event Sourcing for Go — without the framework tax.</strong></p>

<p align="center">
<a href="https://pkg.go.dev/github.com/larsartmann/go-cqrs-lite/event/v4"><img src="https://pkg.go.dev/badge/github.com/larsartmann/go-cqrs-lite/event/v4.svg" alt="Go Reference"></a>
<a href="https://github.com/LarsArtmann/go-cqrs-lite/actions/workflows/ci.yml"><img src="https://github.com/LarsArtmann/go-cqrs-lite/actions/workflows/ci.yml/badge.svg" alt="CI"></a>
<a href="LICENSE"><img src="https://img.shields.io/badge/license-Proprietary-red.svg" alt="License: Proprietary"></a>
<a href="https://github.com/LarsArtmann/templ-components"><img src="https://img.shields.io/badge/GOTH-stack-8A2BE2?style=flat-square" alt="Part of the GOTH stack"></a>
</p>

<p align="center">
<a href="SKILL.md">Guide (SKILL.md)</a> · <a href="https://pkg.go.dev/github.com/larsartmann/go-cqrs-lite/event/v4">API Reference</a> · <a href="#quick-start">Quick Start</a> · <a href="example/">Examples</a> · <a href="docs/">Docs</a>
</p>

---

A composable library of 90+ independently-versioned modules. Import exactly what you need: nothing is forced on you — no transport, no broker, no database driver. Declare your domain, pick your engines at deployment time, and let the composition root wire the rest.

> Using this library with an AI assistant? [`SKILL.md`](SKILL.md) is the single-source guide — module decision matrix, copy-paste recipes, and conventions.

> **Part of the GOTH stack** — pair with
> [templ-components](https://github.com/larsartmann/templ-components) (UI components
> for templ + HTMX + Tailwind v4) and
> [cqrs-htmx](https://github.com/larsartmann/cqrs-htmx) (HTTP → CQRS wiring, auth,
> HTMX response building) for a complete server-rendered Go web stack with zero
> framework lock-in.

## The pipeline

One diagram, the whole library:

```text
what you declare           what system.New wires
────────────────           ────────────────────
Commands                   Dispatcher (typed handlers)
Events                ───▶ Decider (load · fold · decide)
State + Apply              Event Journal (immutable, versioned)
One fold per               Projection Host (replay · crash-restart ·
read model                 checkpoints · dead-letter)
                           metaengine (queries routed by measured
                           cost across interchangeable engines)
```

- **You write**: commands, events, state fold (`Apply`), decide logic, and one fold per read model.
- **`system.New` wires**: event store, bus, projection host (crash-restart, checkpoints, dead-letter), snapshots, timers, health, lifecycle — and routes every query to the right engine.
- **Operators decide**: which engines back each role (SQLite in dev, Postgres in prod — same binary, one config line).

## Why go-cqrs-lite?

Most Go CQRS libraries are **frameworks** — they own your transport, your broker, your SQL driver, and your project layout. go-cqrs-lite is a **library**. You import only what you need and compose your own stack. Nothing is hidden behind magic.

- **Event Sourcing is first-class** — immutable events, branded IDs, optimistic concurrency, time-travel queries, and schema evolution via upcasters. Not an afterthought bolted onto a CRUD layer.
- **Declare the domain, deploy the engines** — `system.New(ctx, DomainConfig, DeploymentConfig)` separates what the developer writes (commands, events, folds) from what the operator picks (engines, durability, topology) in two compiler-enforced config types. Swap SQLite → Postgres → Pebble by changing one line; the domain code never notices.
- **Cost-based query planner** — `metaengine` routes each query to the cheapest engine that supports it (11 interchangeable drivers), with live latency measurement, health-driven rerouting, and an `EXPLAIN`-style doctor.
- **Library, not framework** — no transport, broker, or driver is forced on you. Use standard `net/http`, gRPC, Watermill, NATS — your choice.
- **Pure-Go by default** — SQLite, Pebble, bbolt, Postgres, and MySQL engines need no C compiler. CGo is quarantined inside the single DuckDB module; everyone else never notices.
- **Multi-module isolation** — each module has its own `go.mod` with minimal deps. Import `event` alone: 10 first-party modules plus CBOR — the only third-party production dependency. Your dependency tree stays clean.
- **Production primitives, not stubs** — event signing (HMAC-SHA256, Ed25519, multisig), payload encryption (XChaCha20-Poly1305, AES-256-GCM, key rotation), OTel tracing and metrics, and a Prometheus bridge.
- **Honest error taxonomy** — a 6-family classification (Rejection / Conflict / Transient / Infrastructure / Orchestration / Corruption) with sentinel errors and `%w` wrapping. No panics in production paths.
- **Strong types throughout** — branded IDs make it impossible to mix up an `OrderID` with a `UserID`. The type system catches mistakes the compiler can express.

## Who is this for?

- **Go backend engineers adopting event sourcing** who want proven primitives — stores, buses, upcasters, snapshots — without surrendering their project layout to a framework.
- **DDD practitioners** who model behavior as pure functions (the Decider pattern) and want optimistic concurrency and branded IDs out of the box.
- **Platform engineers embedding CQRS into existing services** — import the modules you need (the `event` module's only third-party production dependency is CBOR), or compose upward as needs grow.
- **Teams that choose storage at deployment time** — declare folds once; let operators bind memory, SQLite, Turso, Pebble, bbolt, Badger, DuckDB, Postgres, MySQL, Dgraph, or Bigtable behind the same domain code.
- **AI-assisted development teams** — [`SKILL.md`](SKILL.md) gives coding agents a verified, single-source API guide instead of hallucinated APIs.

## How it compares

| Capability                                  | go-cqrs-lite | Hand-rolled (stdlib) | [looplab/eventhorizon](https://github.com/looplab/eventhorizon) | [ThreeDotsLabs/watermill](https://github.com/ThreeDotsLabs/watermill) |
| ------------------------------------------- | :----------: | :------------------: | :-------------------------------------------------------------: | :-------------------------------------------------------------------: |
| **Library (not framework)**                 |      ✓       |          ✓           |                             Partial                             |                                Partial                                |
| **Event sourcing**                          |      ✓       |                      |                                ✓                                |                              Via plugins                              |
| **Deployment-time engine choice**           |      ✓       |                      |                                ✗                                |                                   ✗                                   |
| **Cost-based query planner**                |      ✓       |                      |                                ✗                                |                                   ✗                                   |
| **Per-module go.mod**                       |      ✓       |                      |                                ✗                                |                                   ✗                                   |
| **Branded, mix-up-proof IDs**               |      ✓       |                      |                                ✗                                |                                   ✗                                   |
| **Event signing** (HMAC, Ed25519, multisig) |      ✓       |                      |                                ✗                                |                                   ✗                                   |
| **Payload encryption** (XChaCha20, AES-GCM) |      ✓       |                      |                                ✗                                |                                   ✗                                   |
| **Schema evolution** (upcasters)            |      ✓       |                      |                                ✗                                |                                   ✗                                   |
| **Auto-docs** (AsyncAPI, OpenAPI, D2)       |      ✓       |                      |                                ✗                                |                                   ✗                                   |
| **Managed projection host** (crash-restart) |      ✓       |                      |                             Partial                             |                                   ✗                                   |

An empty cell means "you build it yourself." Claims verified against each project's repository (August 2026) — the links are there so you can check the cells.

## When NOT to use this

Skip this library if:

- **Your app is plain CRUD without domain events** — `database/sql`, sqlc, GORM, or ent is simpler. Event sourcing adds ceremony that CRUD does not need.
- **You want a framework that owns transport and project layout** — application frameworks in the go-zero/Kratos style do that; go-cqrs-lite deliberately owns neither.
- **You need an ops-heavy event store** — server-side projections, persistent subscriptions, a query UI, multi-node clustering. [KurrentDB](https://github.com/EventStore/EventStore) (EventStoreDB) is a purpose-built server; this is an embedded library.
- **Your problem is messaging, not domain aggregates** — plain [Watermill](https://github.com/ThreeDotsLabs/watermill) is lighter. You can adopt our `watermill/` adapter later if domain events join the picture.

## Install

New applications build on the composition root ([ADR-0123](docs/adr/0123-v5-unification-single-composition-root.md)):

```bash
go get github.com/larsartmann/go-cqrs-lite/system/v4
go get github.com/larsartmann/go-cqrs-lite/metaengine/v4
# plus one blank import per engine you deploy, e.g. SQLite:
go get github.com/larsartmann/go-cqrs-lite/metaengine/sqliteengine/v4
```

Every module also releases on its own version train — import layers individually when you need them (`event`, `command`, `decider`, `id`, …) and your dependency tree stays lean. If you pin several, upgrade with the supported sweep pattern (per-module `go get …@latest` + `go mod tidy` + hermetic `GOWORK=off go build`/`go vet`, gated on tag existence) — see the skill FAQ's "How do I upgrade many go-cqrs-lite modules at once?". `cmd/cqrs-upgrade` scans your modules for v5-removed surfaces before you sweep.

## Quick Start

The recommended path for new code: declare a domain, declare read models as folds, boot one composition root. (Condensed from [`example/getting-started/`](example/getting-started/) — a complete, tested, runnable version of exactly this.)

### 1. Declare the domain — events, state, decide logic

```go
type Incremented struct {
	Amount int `json:"amount"`
}

type CounterState struct{ Value int }

func applyCounter(s CounterState, evt event.Event) (CounterState, error) {
	if evt.Type() != "counter.incremented" {
		return s, nil
	}
	p, err := event.DecodePayloadAuto[Incremented](evt)
	if err != nil {
		return s, err
	}
	s.Value += p.Amount
	return s, nil
}

type IncrementCmd struct {
	*command.BasicCommand

	Amount int
}
```

### 2. Declare the read model — one fold per event type

```go
type CounterView struct {
	Value int `json:"value"`
}

type counterLookup struct{} // query-input type; reads key by fold-returned stream ID

counterViews := metaengine.Query[counterLookup, CounterView]("counter_views",
	// first event creates the view …
	metaengine.OnRecordTyped("counter.incremented",
		projectionadapter.EventWithID[Incremented]{},
		func(_ record.Record, e projectionadapter.EventWithID[Incremented]) (string, CounterView) {
			return e.ID, CounterView{Value: e.Payload.Amount}
		}),
	// … later events update it
	metaengine.OnRecordTyped("counter.incremented",
		projectionadapter.EventWithID[Incremented]{},
		func(_ record.Record, e projectionadapter.EventWithID[Incremented], prev CounterView) CounterView {
			prev.Value += e.Payload.Amount
			return prev
		}),
)
```

### 3. Boot the system — and never touch infrastructure again

```go
sys, err := system.New(ctx,
	system.DomainConfig{ // what the developer writes
		Commands: func(sys *system.System) {
			_ = system.RegisterDecider(sys, "Counter", decider.Decider[CounterState]{
				Initial: CounterState{},
				Apply:   applyCounter,
			})
			_ = system.RegisterCommand[IncrementCmd, CounterState](sys, "counter.increment",
				func(cmdCtx context.Context, cmd IncrementCmd) system.Op[CounterState] {
					return system.Execute(cmdCtx, cmd.StreamID(), "Counter",
						func(_ CounterState, v event.Version) ([]event.Event, error) {
							evt, err := event.New("counter.incremented", cmd.StreamID(), "Counter",
								v.Increment(), Incremented{Amount: cmd.Amount})
							if err != nil {
								return nil, err
							}
							return []event.Event{evt}, nil
						})
				})
		},
		Projections: []system.ProjectionDeclaration{system.RawQuery(counterViews)},
	},
	system.DeploymentConfig{ // what the operator picks
		Engines: map[string]system.EngineConfig{
			"primary": {Driver: "memory"}, // dev
			// "primary": {Driver: "sqlite", DSN: "app.db"}, // production — one line
		},
		Instances: []system.InstanceConfig{
			{Role: system.RoleSourceOfTruth, Engine: "primary"},
			{Role: system.RoleProjections, Engine: "primary"},
		},
	},
)
```

Dispatch a command, read the projected view:

```go
basic, _ := command.New("counter.increment", counterID)
_ = sys.CommandDispatcher().Dispatch(ctx,
	IncrementCmd{BasicCommand: basic, Amount: 5})

reader := metaengine.NewReader[CounterView](sys.MetaEngine(), "counter_views")
view, found, _ := reader.Get(ctx, counterID.Get()) // projection catches up async
```

**That is the whole mental model.** Store, bus, projection host (replay + live tail + crash-restart + dead-letter), engine registration, health, and shutdown are wired by `system.New`. Run the full example:

```bash
git clone https://github.com/LarsArtmann/go-cqrs-lite && cd go-cqrs-lite/example/getting-started
GOWORK=off go run main.go   # → Counter <ulid>: value=10 (expected 10)
```

> **Heads-up (v5):** the older `stack/*` presets and v1 read-model tiers
> (`stack.Materialize`, `SQLViewStore`, `RelationalProjection`, `GraphProjection`)
> are deprecated and will be **removed in v5**
> ([ADR-0123](docs/adr/0123-v5-unification-single-composition-root.md)) —
> they work unchanged through v4.x, but new code should start on
> `system/` + `metaengine` as shown above. Canonical v5-removal list:
> [FAQ — "Will the v5 cut break my imports?"](.agents/skills/go-cqrs-lite/references/faq.md#will-the-v5-cut-break-my-imports-what-is-going-away).

## Engines: storage is a deployment decision

All durable I/O — journal, read models, snapshots, checkpoints, timers, dedup — rides one engine layer ([ADR-0142](docs/adr/0142-universal-storage-substrate.md)). Drivers self-register `database/sql`-style via blank import; `DeploymentConfig` binds them to roles; the planner routes each query to the cheapest engine that supports it.

| Driver     | Module                      | Build    | Strengths                                         |
| ---------- | --------------------------- | -------- | ------------------------------------------------- |
| `memory`   | in-core                     | pure Go  | tests, defaults, zero setup                       |
| `sqlite`   | `metaengine/sqliteengine`   | pure Go  | embedded single-file persistence                  |
| `turso`    | `metaengine/tursoengine`    | pure Go  | embedded Turso Database, optional page encryption |
| `pebble`   | `metaengine/pebbleengine`   | pure Go  | high-throughput embedded LSM                      |
| `bbolt`    | `metaengine/bboltengine`    | pure Go  | single-file B+tree                                |
| `badger`   | `metaengine/badgerengine`   | pure Go  | embedded KV with prefix scans                     |
| `postgres` | `metaengine/pgengine`       | pure Go  | distributed, pooling, `WITH RECURSIVE` graph CTEs |
| `mysql`    | `metaengine/mysqlengine`    | pure Go  | MySQL/MariaDB (pure-Go driver)                    |
| `dgraph`   | `metaengine/dgraphengine`   | pure Go  | native graph traversal                            |
| `bigtable` | `metaengine/bigtableengine` | pure Go  | GCP Bigtable                                      |
| `duckdb`   | `metaengine/duckdbengine`   | CGo      | embedded columnar OLAP, vector search             |
| —          | `metaengine/irohengine`     | pure Go  | CRDT replication across nodes (programmatic, p2p) |

Read models are ADTs, not opaque blobs: Map (CRUD), Graph (edges, traversals), Vector (k-NN), Log, Set, Counter — the planner picks SQL pushdown where the engine supports it and scans otherwise. Full per-capability matrix (generated from source): [`docs/engine-capabilities.md`](docs/engine-capabilities.md). Deployment-from-YAML demo: [`example/metaengine-quickstart/`](example/metaengine-quickstart/).

### Optional engine interfaces: token-resumable journal reads

Engines expose optional capabilities beyond the base `Engine` profile; callers
type-assert them. The newest is `metaengine.SeqSeekableStreamLog` — journal
reads that resume by token instead of position:

```go
if seekable, ok := eng.(metaengine.SeqSeekableStreamLog); ok {
	entries, _ := seekable.JournalReadAllWithSeq(ctx, "orders")
	last := entries[len(entries)-1].Seq // opaque, strictly increasing token
	// ... later, exactly after that entry — an index range seek, not OFFSET:
	next, _ := seekable.JournalReadFromSeq(ctx, "orders", last, 500)
	_ = next
}
```

`StreamLogBackend.JournalReadFrom` takes a POSITION (forcing OFFSET-style skips
on engines with shared sequence counters); `JournalReadFromSeq` takes a TOKEN
previously returned by the engine, so resuming is a pure
`WHERE collection = ? AND seq > ?` index seek — O(log n) per page and
gap-tolerant by construction. Tokens are engine-instance-local: never persist a
Seq across engine migrations or restarts; long-lived checkpoints store the
domain cursor (e.g. event ID), not the raw Seq.

Implemented by: memory, sqlite (incl. turso), pebble, bbolt, badger, postgres,
mysql.

## Prefer explicit wiring? (v4 composition)

The layers under the composition root are independently importable — some v4 surfaces predate `system/` and remain fully supported through v4.x:

- **Manual tutorial**: [`docs/getting-started.md`](docs/getting-started.md) — decider, repository, dispatcher wired by hand (compile-verified in CI).
- **Minimal example**: [`example/readme-quickstart/`](example/readme-quickstart/) — command + decider, no system layer.
- **`stack/*` presets** (deprecated, removed in v5): one-call bundles that wire store + bus + read models per backend — `stack/memory`, `stack/sqlite`, `stack/pebble`, `stack/bbolt`, `stack/duckdb`, `stack/postgres`, `stack/mysql`, `stack/turso`. Existing v4 code keeps working; `cmd/cqrs-upgrade` lists what to migrate before v5.

## Examples

| Example                                                   | Shows                                                                                          |
| --------------------------------------------------------- | ---------------------------------------------------------------------------------------------- |
| [`getting-started`](example/getting-started/)             | single-file `system.New` tour; a test proves the one-line engine swap                          |
| [`taskmanager`](example/taskmanager/)                     | flagship HTTP service: ES, projections, middleware, signing, SSE, snapshots, OTel              |
| [`metaengine-quickstart`](example/metaengine-quickstart/) | deployment-time story: folds + queries, engine picks from a `cqrs.yaml`, map/graph/vector demos |
| [`graph-native`](example/graph-native/)                   | follow-network graph reads, deletion-as-domain-event, BDD-verified                             |
| [`mesh-demo`](example/mesh-demo/)                         | two bounded contexts wired only through replicated event contracts (data mesh)                 |
| [`goal-shaped-app`](example/goal-shaped-app/)             | the library's north star, runnable: declare commands/events/queries, operators place the data  |
| [`readme-quickstart`](example/readme-quickstart/)         | minimal command + decider without the system layer                                             |
| [`scheduler-otel-status`](example/scheduler-otel-status/) | durable deadline timers + OTel/Prometheus claim observability                                  |

## Key modules

Every module is independently importable and has its own `go.mod`. Here are the most important ones — see [AGENTS.md](AGENTS.md) for the full module catalog and [FEATURES.md](FEATURES.md) for the feature inventory.

| Module                  | Purpose                                                                       |
| ----------------------- | ----------------------------------------------------------------------------- |
| **system**              | Composition root: `system.New(ctx, DomainConfig, DeploymentConfig)` wires everything |
| **metaengine**          | Cost-based storage planner; Map/Graph/Vector/Log/Set/Counter read-model ADTs  |
| **metaengine/*engine**  | 11 self-registering drivers: sqlite, postgres, mysql, pebble, bbolt, …        |
| **event**               | Immutable events, store/bus interfaces, event sourcing                        |
| **command**             | Typed command dispatch, middleware, audit journal, pub/sub bus                |
| **query**               | Typed query dispatch, pagination, audit journal                               |
| **decider**             | Pure-function event-sourcing pattern: load, apply, decide, save               |
| **id**                  | Branded IDs backed by ULID                                                    |
| **projectionhost**      | Managed projection lifecycle: crash-restart, checkpoints, dead-letter         |
| **scheduling**          | Durable deadline timers (cancellations, retries, manual clock for tests)      |
| **watermill**           | Broker pub/sub bridge (NATS, Kafka, Redis Streams) + catch-up subscriber      |
| **middleware**          | Logging, retry, validation, recovery, circuit breaker, OTel                   |
| **kv**                  | Layer-0 KV abstraction: `kv.Store`, `TypedStore[T,K]`, `Cache`                |
| **catalog**             | Auto-generate AsyncAPI 3.0, EventCatalog, OpenAPI, D2 from Go types           |
| **systemscenario**      | BDD Given/When/Then over a real `system.New` boot, deterministic manual clock |

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

90+ modules on `/v4` import paths (103 go.mod files incl. root — gate-derived via `scripts/check-canonical-facts.sh`). Core modules carry roughly 86–95% test coverage (event 89%, decider 95%, id 86%, dispatcher 87%). The library covers the full CQRS/ES lifecycle: event sourcing with branded IDs, command/query dispatch, pure-function deciders, cost-based query planning across interchangeable engines, durable deadline scheduling, dead-letter quarantine, managed projection hosting, event signing and encryption, OTel tracing and metrics, auto-documentation generation, and a domain-aware linter (cqrs-lint).

**Migrating from v3?** Read the **[Migration Guide](docs/migration/MIGRATION-GUIDE.md)** — covers the v4 breaking changes (codec defaults, API cleanup, path migration). For v2-to-v3 changes, see the **[v3 Migration Guide](docs/migration/V3_MIGRATION.md)**.

For the full feature inventory see [FEATURES.md](FEATURES.md), for direction see [ROADMAP.md](ROADMAP.md), and for architecture decisions, benchmarks, and storage guides see [docs/](docs/).

## Published versions

Modules release on independent per-module version trains (`<module>/vX.Y.Z` git tags). The manifest below is machine-readable as [`versions.json`](versions.json) (key `.` = the repo-root module train); both are regenerated by `bash scripts/check-versions-manifest.sh --update` and kept fresh by a nightly gate.

<details>
<summary>Module → latest published tag (generated — do not edit by hand)</summary>

<!-- versions-manifest:begin -->
| Module | Latest published tag |
| --- | --- |
| . | `v4.0.0` *(untagged per ADR-0152)* |
| benchkit | `benchkit/v4.7.0` *(untagged per ADR-0152)* |
| catalog | `catalog/v4.13.0` |
| claiming | `claiming/v4.0.2` |
| cmd/api-stability | `cmd/api-stability/v4.4.2` *(untagged per ADR-0152)* |
| cmd/cqrs-bench | `cmd/cqrs-bench/v4.3.4` *(untagged per ADR-0152)* |
| cmd/cqrs-gen | `cmd/cqrs-gen/v4.3.3` *(untagged per ADR-0152)* |
| cmd/cqrs-lint | `cmd/cqrs-lint/v4.21.0` |
| cmd/cqrs-upgrade | `cmd/cqrs-upgrade/v4.1.2` *(untagged per ADR-0152)* |
| cmd/doc-check | `cmd/doc-check/v4.3.3` *(untagged per ADR-0152)* |
| codec | `codec/v4.4.0` |
| command | `command/v4.14.1` |
| commandlifecycle | `commandlifecycle/v4.3.1` |
| commandlifecycle/projections | `commandlifecycle/projections/v4.2.3` |
| core | `core/v1.6.0` |
| cqrs-lite | `cqrs-lite/v0.1.1` |
| decider | `decider/v4.8.0` |
| dedup | `dedup/v4.3.0` |
| deriver | `deriver/v4.5.0` |
| dispatcher | `dispatcher/v4.5.2` |
| encryption | `encryption/v4.4.6` |
| event | `event/v4.15.2` |
| event/v3/eventtest | `event/v3/eventtest/v3.7.4` |
| event/v4/eventtest | `event/v4/eventtest/v0.4.0` |
| example/getting-started | `example/getting-started/v4.1.0` *(untagged per ADR-0152)* |
| example/goal-shaped-app | `example/goal-shaped-app/v0.1.2` *(untagged per ADR-0152)* |
| example/metaengine-quickstart | `example/metaengine-quickstart/v0.1.3` *(untagged per ADR-0152)* |
| example/readme-quickstart | `example/readme-quickstart/v0.2.3` *(untagged per ADR-0152)* |
| example/scheduler-otel-status | `example/scheduler-otel-status/v0.1.2` *(untagged per ADR-0152)* |
| example/taskmanager | `example/taskmanager/v3.7.1` *(untagged per ADR-0152)* |
| flightrecorder | `flightrecorder/v4.0.0` |
| graph | `graph/v4.3.4` *(untagged per ADR-0152)* |
| id | `id/v4.7.5` |
| idempotency | `idempotency/v4.4.0` |
| idempotency/kvstore | `idempotency/kvstore/v4.3.3` *(untagged per ADR-0152)* |
| idempotency/sqlstore | `idempotency/sqlstore/v4.4.3` *(untagged per ADR-0152)* |
| integration | `integration/v4.2.3` *(untagged per ADR-0152)* |
| kv | `kv/v4.4.0` |
| listing | `listing/v4.4.4` |
| memory | `memory/v2.6.0` |
| metadata | `metadata/v4.7.3` |
| metaengine | `metaengine/v4.21.0` |
| metaengine/badgerengine | `metaengine/badgerengine/v4.3.3` *(untagged per ADR-0152)* |
| metaengine/bboltengine | `metaengine/bboltengine/v4.3.3` *(untagged per ADR-0152)* |
| metaengine/bench | `metaengine/bench/v4.1.2` *(untagged per ADR-0152)* |
| metaengine/bigtableengine | `metaengine/bigtableengine/v4.0.2` *(untagged per ADR-0152)* |
| metaengine/dgraphengine | `metaengine/dgraphengine/v4.3.3` *(untagged per ADR-0152)* |
| metaengine/duckdbengine | `metaengine/duckdbengine/v4.3.3` *(untagged per ADR-0152)* |
| metaengine/graphadapter | `metaengine/graphadapter/v4.1.3` *(untagged per ADR-0152)* |
| metaengine/irohengine | `metaengine/irohengine/v4.3.2` *(untagged per ADR-0152)* |
| metaengine/irohengine/loopback | `metaengine/irohengine/loopback/v4.0.5` *(untagged per ADR-0152)* |
| metaengine/irohengine/quic | `metaengine/irohengine/quic/v4.2.3` *(untagged per ADR-0152)* |
| metaengine/mysqlengine | `metaengine/mysqlengine/v4.3.3` *(untagged per ADR-0152)* |
| metaengine/otelobserver | `metaengine/otelobserver/v4.0.2` *(untagged per ADR-0152)* |
| metaengine/pebbleengine | `metaengine/pebbleengine/v4.4.4` |
| metaengine/pgengine | `metaengine/pgengine/v4.4.3` *(untagged per ADR-0152)* |
| metaengine/projectionadapter | `metaengine/projectionadapter/v4.6.2` |
| metaengine/sqliteengine | `metaengine/sqliteengine/v4.7.0` |
| metaengine/tursoengine | `metaengine/tursoengine/v4.2.3` *(untagged per ADR-0152)* |
| middleware | `middleware/v4.7.2` |
| otel | `otel/v4.5.3` |
| otel/otlp | `otel/otlp/v4.0.2` *(untagged per ADR-0152)* |
| pebble | `pebble/v2.6.0` |
| projection | `projection/v4.4.2` |
| projectionhost | `projectionhost/v4.7.1` |
| prometheus | `prometheus/v4.3.4` |
| query | `query/v4.10.1` |
| queue | `queue/v4.0.3` |
| queue/mysql | `queue/mysql/v4.0.3` *(untagged per ADR-0152)* |
| queue/postgres | `queue/postgres/v4.0.3` |
| queue/sqlite | `queue/sqlite/v4.0.3` |
| record | `record/v4.6.2` |
| retry | `retry/v4.3.0` |
| saga | `saga/v1.0.0` |
| scenario | `scenario/v4.4.3` |
| scheduling | `scheduling/v4.7.0` |
| scheduling/engine | `scheduling/engine/v4.0.2` |
| scheduling/sqlstore | `scheduling/sqlstore/v4.1.4` |
| schema | `schema/v4.7.1` |
| signing | `signing/v4.4.2` |
| snapshot | `snapshot/v4.7.0` |
| stack | `stack/v4.5.1` |
| stack/bbolt | `stack/bbolt/v4.2.3` *(untagged per ADR-0152)* |
| stack/bench | `stack/bench/v4.3.2` *(untagged per ADR-0152)* |
| stack/duckdb | `stack/duckdb/v4.2.3` *(untagged per ADR-0152)* |
| stack/memory | `stack/memory/v4.4.3` *(untagged per ADR-0152)* |
| stack/metaengine | `stack/metaengine/v4.0.0` *(untagged per ADR-0152)* |
| stack/mysql | `stack/mysql/v4.2.3` *(untagged per ADR-0152)* |
| stack/pebble | `stack/pebble/v4.4.3` *(untagged per ADR-0152)* |
| stack/postgres | `stack/postgres/v4.4.3` *(untagged per ADR-0152)* |
| stack/sqlite | `stack/sqlite/v4.3.5` |
| stack/turso | `stack/turso/v4.4.3` *(untagged per ADR-0152)* |
| storage | `storage/v4.11.1` |
| storage/backuptest | `storage/backuptest/v4.2.3` *(untagged per ADR-0152)* |
| storage/bbolt | `storage/bbolt/v4.2.3` |
| storage/memory | `storage/memory/v4.6.2` |
| storage/pebble | `storage/pebble/v4.4.4` *(untagged per ADR-0152)* |
| storage/turso | `storage/turso/v4.3.5` |
| sync | `sync/v0.2.0` |
| system | `system/v4.16.0` |
| system/integration | `system/integration/v4.0.2` *(untagged per ADR-0152)* |
| systemcatalog | `systemcatalog/v4.0.0` |
| systemscenario | `systemscenario/v4.3.0` |
| systemtest | `systemtest/v4.0.0` *(untagged per ADR-0152)* |
| testhelpers | `testhelpers/v1.7.1` |
| testing | `testing/v3.3.0` |
| testutil | `testutil/v4.3.4` |
| testutil/mysqltestcontainer | `testutil/mysqltestcontainer/v4.0.0` *(untagged per ADR-0152)* |
| testutil/pgtestcontainer | `testutil/pgtestcontainer/v4.2.3` |
| transport/grpc | `transport/grpc/v4.3.3` *(untagged per ADR-0152)* |
| transport/http | `transport/http/v4.3.5` *(untagged per ADR-0152)* |
| turso | `turso/v2.6.0` |
| watermill | `watermill/v4.7.0` |
<!-- versions-manifest:end -->

</details>

## License

PROPRIETARY — see [LICENSE](LICENSE).

Because the license is not OSS-approved, pkg.go.dev hides module documentation for every module by design (verified 2026-09-28; per-module LICENSE copies do not change this). Browse the API locally with `go doc <module>`, or read [SKILL.md](SKILL.md).
