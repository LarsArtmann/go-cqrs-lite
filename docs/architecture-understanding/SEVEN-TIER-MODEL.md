# Seven-Tier Model — go-cqrs-lite

> Source of truth: [`scripts/check-module-layers.sh`](../../scripts/check-module-layers.sh)
> enforces these rules at CI time. See [ADR-0046](../adr/0046-seven-tier-model.md)
> for the decision record.

## The Seven Tiers

Each tier may only import from its own tier or lower. Tier assignment is both
**structural** (what does this module depend on?) and **conceptual** (what role
does it play?). A module with zero internal deps can still be Tier 4+ if its
role is infrastructure or tooling — e.g. `otel/` (zero deps, Tier 4),
`catalog/` (zero deps, Tier 6).

> **Enforcement detail:** The check script uses 8 numeric layers (0–7) for
> dependency ordering. Conceptual Tier 4 (Infrastructure) maps to script
> Layers 4–5: Layer 4 modules (signing, encryption, otel, storage/memory) have
> no deps on other infrastructure, while Layer 5 modules (middleware, storage,
> transport, etc.) depend on Layer 4 modules. This split is invisible in the
> conceptual model but enforced by the script. All other tiers map 1:1
> (conceptual tier N = script layer N for tiers 0–3 and 5–6, where Tier 5 =
> Layer 6 and Tier 6 = Layer 7).

| Tier | Name               | Rule                                 | Modules |
| ---- | ------------------ | ------------------------------------ | ------- |
| 0    | Primitives         | No internal deps (or same-tier only) | 9       |
| 1    | Core Domain        | Depends on Tier 0                    | 5       |
| 2    | Domain Utilities   | Depends on Tier 0–1                  | 7       |
| 3    | Aggregation        | Depends on Tier 0–2                  | 5       |
| 4    | Infrastructure     | Depends on Tier 0–3                  | 27      |
| 5    | Composition        | Depends on Tier 0–4                  | 10      |
| 6    | Tooling & Examples | Depends on all                       | 15      |

**Total: 78 modules** across 79 `go.mod` files (78 modules + 1 root workspace
placeholder). Verify: `find . -name go.mod -not -path './vendor/*' | wc -l`

---

## Tier 0 — Primitives (leaf modules, no internal deps or same-tier only)

These are foundational building blocks. Some depend on each other (e.g.
`kv/` → external `go-codec`, `metaengine/` → `dedup/` + `record/`) but never
on higher tiers.

| Module            | Purpose                                                                      |
| ----------------- | ---------------------------------------------------------------------------- |
| `id/`             | Branded IDs: `id.Of[T]`, `AggregateID`, `StreamID`, etc.                     |
| `dispatcher/`     | Generic `Dispatcher[H, M]` with lifecycle mixin                              |
| `kv/`             | KV store: `Store`, `MemStore`, `TypedStore[T,K]`                             |
| `dedup/`          | Bounded dedup ring buffer                                                    |
| `record/`         | Shared Record + CommonMetadata types (structural base for events + commands) |
| `metaengine/`     | Cost-based storage planner (the strategic future of this project)            |

> **`metaengine/` is Tier 3 in the ADR-0046 model, with a Tier-0-style core**
> (ADR-0062 amendment): its PLANNER core depends only on Tier-0 primitives
> (`dedup/`, `record/`) plus `id/`, while the module as a whole also carries
> the embedded default engine (`sqliteengine`) and `go-sse`. The bridge to the
> CQRS event-sourcing world lives in `metaengine/projectionadapter/` (Tier 4).
> `codec/`, `flightrecorder/`, and `retry/` were extracted to external repos
> (ADR-0128) and are no longer in-repo tiers.

> **`record/` is Tier 0** (ADR-0111). Zero deps. Structural base for both
> events and commands. `event.AsRecord()` adapts the ES pipeline.

---

## Tier 1 — Core Domain (depends on Tier 0)

The core CQRS vocabulary: events, commands, queries, scheduling, metadata.

| Module        | Purpose                                            |
| ------------- | -------------------------------------------------- |
| `event/`      | EventSink, EventSource, Store, Bus, ImmutableEvent |
| `command/`    | Dispatcher, Handler, Middleware, Bus (pub/sub)     |
| `query/`      | Dispatcher, Handler, Pagination, TypedHandler      |
| `scheduling/` | Durable deadline timers: TimerStore, Scheduler     |
| `metadata/`   | Tracing, CustomData (shared metadata types)        |

> **CQRS separation is clean at compile level.** `command/` and `query/` have
> zero `event/` imports in production code. Shared types (Tracing, CustomData)
> were extracted to `metadata/` (ADR-0031), breaking the old dependency. The
> only remaining leak: `storage/memory/` as a test dep pulls `event/` into
> go.mod indirect for both modules.

---

## Tier 2 — Domain Utilities (depends on Tier 0–1)

Cross-cutting domain concerns that build on the core vocabulary.

| Module                  | Purpose                                                        |
| ----------------------- | -------------------------------------------------------------- |
| `schema/`               | Upcaster, VersionedStore, Validator with `RegisterType[T]()`   |
| `snapshot/`             | Snapshot types + strategy (EveryNEvents, ReadPressure)         |
| `projection/`           | Projection interface (consumer-side)                           |
| `idempotency/`          | Re-export of go-idempotency: Store, MemoryStore, ErrDuplicate  |
| `deriver/`              | Event-to-command derivation: Deriver, Then, Filter, Idempotent |
| `idempotency/kvstore/`  | KV-backed idempotency                                          |
| `idempotency/sqlstore/` | SQL-backed idempotency (SQLite/Postgres)                       |

---

## Tier 3 — Aggregation (depends on Tier 0–2)

Higher-level domain patterns that combine multiple Tier 0–2 modules.

| Module            | Purpose                                              |
| ----------------- | ---------------------------------------------------- |
| `decider/`        | `Decider[State]`, `Repository[State]`, cache         |
| `graph/`          | Graph projection: nodes, edges, traversal            |
| `scenario/`       | Fluent BDD test DSL: Given/When/Then                 |
| `projectionhost/` | Managed projection lifecycle + DLQ + checkpoint      |
| `listing/`        | Stream status, tombstone detection, StatusMiddleware |

---

## Tier 4 — Infrastructure (depends on Tier 0–3)

Concrete implementations of the abstractions: storage backends, transport,
middleware, observability, and security. The largest tier.

> **Inner Infrastructure (script Layer 4)** — no deps on other infrastructure:

| Module            | Purpose                                                                   |
| ----------------- | ------------------------------------------------------------------------- |
| `signing/`        | Event signing: HMAC-SHA256, Ed25519, multisig                             |
| `encryption/`     | Payload encryption: XChaCha20-Poly1305, AES-256-GCM                       |
| `otel/`           | Shared OTel helpers (zero internal deps, but conceptually infrastructure) |
| `storage/memory/` | In-memory test impls (MemoryStore, etc.)                                  |

> **Outer Infrastructure (script Layer 5)** — depends on inner infrastructure:

### Storage Backends

| Module                | Purpose                                                                            |
| --------------------- | ---------------------------------------------------------------------------------- |
| `storage/`            | SQL facade: EventStore, CommandStore, QueryStore, KV, relational, view, migrations |
| `storage/pebble/`     | PebbleDB: EventStore, KVAdapter, Backend facade                                    |
| `storage/bbolt/`      | bbolt: EventStore, KVAdapter, Backend facade (B+tree, pure Go)                     |
| `storage/backuptest/` | Shared backup lifecycle test suite (Backend interface, Factory, RunFullLifecycle)  |
| `storage/turso/`      | Turso embedded database connector                                                  |

### Observability

| Module        | Purpose                           |
| ------------- | --------------------------------- |
| `prometheus/` | OTel-to-Prometheus metrics bridge |

### Cross-Cutting

| Module                      | Purpose                                                                           |
| --------------------------- | --------------------------------------------------------------------------------- |
| `middleware/`               | Logging, Retry, Recovery, Validation, Idempotency, Metrics, OTel, Circuit Breaker |
| `testutil/`                 | Shared test helpers (NewCmd, etc.)                                                |
| `testutil/pgtestcontainer/` | Postgres testcontainers helper for integration tests                              |

### Transport

| Module            | Purpose                                |
| ----------------- | -------------------------------------- |
| `transport/http/` | SSE broker (event.Bus to HTTP clients) |
| `transport/grpc/` | gRPC transport (remote command/query)  |

### Messaging

| Module       | Purpose                                                    |
| ------------ | ---------------------------------------------------------- |
| `watermill/` | Watermill bridges: EventBus, CommandBus, CatchUpSubscriber |

### Metaengine Infrastructure

| Module                            | Purpose                                              |
| --------------------------------- | ---------------------------------------------------- |
| `metaengine/projectionadapter/`   | Wraps metaengine Store as projection.Projection      |
| `metaengine/pebbleengine/`        | Pebble-backed engine (LSM point reads)               |
| `metaengine/duckdbengine/`        | DuckDB-backed engine (columnar OLAP, CGo)            |
| `metaengine/pgengine/`            | Postgres-backed engine (JSONB + B-tree)              |
| `metaengine/sqliteengine/`        | SQLite-backed engine (extracted from core, ADR-0115) |
| `metaengine/badgerengine/`        | Badger-backed engine (LSM point reads)               |
| `metaengine/dgraphengine/`        | Dgraph-backed engine (distributed graph DB)          |
| `metaengine/graphadapter/`        | Wraps graph.MemoryDriver as metaengine Engine        |
| `metaengine/irohengine/`          | Iroh Level 2 replication wrapper (CRDT convergence)  |
| `metaengine/irohengine/loopback/` | Loopback transport (real TCP, no CGo)                |
| `metaengine/irohengine/quic/`     | QUIC transport (real Iroh, CGo)                      |

### Sub-Store Implementations

| Module                 | Purpose                                        |
| ---------------------- | ---------------------------------------------- |
| `scheduling/sqlstore/` | SQL-backed timer store (SQLite/Postgres/MySQL) |

---

## Tier 5 — Composition (depends on Tier 0–4)

One-call wiring that composes infrastructure into deployable bundles.

| Module            | Purpose                                                            |
| ----------------- | ------------------------------------------------------------------ |
| `stack/`          | Bundle abstraction + shared options (durability, codec, health)    |
| `stack/memory/`   | Memory preset                                                      |
| `stack/sqlite/`   | SQLite preset                                                      |
| `stack/pebble/`   | Pebble preset (LSM)                                                |
| `stack/bbolt/`    | bbolt preset (B+tree, pure Go)                                     |
| `stack/postgres/` | Postgres preset                                                    |
| `stack/duckdb/`   | DuckDB preset (CGo, columnar OLAP)                                 |
| `stack/mysql/`    | MySQL/MariaDB preset                                               |
| `stack/turso/`    | Turso preset (embedded sync)                                       |
| `system/`         | Deployer-driven composition root (DomainConfig + DeploymentConfig) |

---

## Tier 6 — Tooling & Examples (depends on all)

Developer tools, code generators, linters, test harnesses, and usage demos.

| Module                           | Purpose                                             |
| -------------------------------- | --------------------------------------------------- |
| `catalog/`                       | API documentation generator (AsyncAPI, D2, OpenAPI) |
| `integration/`                   | Cross-module integration tests                      |
| `benchkit/`                      | Factory-driven benchmarking suite                   |
| `stack/bench/`                   | Stack-level benchmark presets                       |
| `metaengine/bench/`              | Cross-engine benchmark module                       |
| `cmd/cqrs-gen/`                  | Code generator: typed handler registration          |
| `cmd/cqrs-lint/`                 | Domain-aware linter (192 rules, 10 categories)      |
| `cmd/cqrs-bench/`                | CLI: benchmark any backend with workload profiles   |
| `cmd/api-stability/`             | API surface checker (golden file comparison)        |
| `cmd/doc-check/`                 | Doc link verifier (Go import paths in markdown)     |
| `example/taskmanager/`           | Flagship full HTTP service example                  |
| `example/getting-started/`       | Minimal 80-line example                             |
| `example/readme-quickstart/`     | README quickstart example                           |
| `example/metaengine-quickstart/` | Metaengine quickstart example                       |
| `event/v4/eventtest/`            | Test helpers: FakeStore, FakeBus, golden assertions |

---

## Sub-Packages (not separate modules)

These directories have no `go.mod` — they are sub-packages that inherit their
parent module's tier:

| Sub-package           | Parent module | Purpose                                |
| --------------------- | ------------- | -------------------------------------- |
| `id/idtest/`          | `id/`         | Parse*(tb, s) test helpers             |
| `query/querytest/`    | `query/`      | New(tb, queryType) test helper         |
| `kv/viewstoretest/`   | `kv/`         | ViewStore contract test helpers        |
| `metaengine/adttest/` | `metaengine/` | ADT test harness (RunMatrix, 10 ADTs)  |
| `storage/eventstore/` | `storage/`    | SQLEventStore, SQLSnapshotStore        |
| `storage/readmodel/`  | `storage/`    | SQLKVStore                             |
| `storage/sql/`        | `storage/`    | Dialect, DBHandle, RunInTx, ScanSlice  |
| `storage/relational/` | `storage/`    | RelationalSchema, RelationalProjection |
| `storage/view/`       | `storage/`    | SQLViewStore, ViewMapper, AutoMapper   |
| `storage/migrations/` | `storage/`    | Embedded .sql DDL files                |
| `catalog/schema/`     | `catalog/`    | JSON Schema types, reflection engine   |

---

## Why the Old 7-Layer System Was Fake

The original layer system claimed clean stratification but had four structural
lies:

1. **`kv/` claimed Layer 0 but depended on `codec/`** — not a true leaf. The
   new model acknowledges same-tier deps are fine (kv/ and codec/ are both
   Tier 0).
2. **`event/` claimed Layer 1 but pulled Tier 2–4 modules into its go.mod** via
   test deps (`eventtest`, `schema`, `snapshot`, `storage/memory`) that leaked
   into production dep counts. The `eventtest` extraction to a nested module
   (ADR-0045) reduced this, but Go modules still list test deps in `require`.
3. **48 of 78 modules depend on `codec/`** — the true hub was invisible in the
   old system. `codec/` is now correctly placed in Tier 0.
4. **`command/` and `query/` pull `event/` as `// indirect` in go.mod** —
   via `storage/memory/` (test-only dep). Production code has zero `event/`
   imports — the `metadata/` extraction (ADR-0031) broke the compile dependency.
   The indirect leak is cosmetic but still inflates perceived coupling.

## What This Changes

- Honest dependency tracking — no more claiming "Layer 0" while importing codec/
- `nix run .#check-layers` validates against real tiers
- `nix run .#check-layers` enforces per-module production dependency budgets
- v4 extraction work is complete: `id/`, `metadata/`, `retry/`, `idempotency/`
  are standalone modules; `kv/` has `context.Context`; `storage/` is split into
  focused sub-packages
