# Seven-Tier Model — go-cqrs-lite

> Replaces the fake "7-layer" system that claimed clean stratification but had
> cross-layer dependencies leaking through go.mod files and test-only deps
> inflating production dep counts. See [ADR-0046](../adr/0046-seven-tier-model.md)
> for the decision record.

> **Filename note:** Originally titled "Four-Tier Model." The name was a
> misnomer — the model describes seven numbered tiers (0–6). Renamed for
> accuracy. The filename is retained to avoid breaking existing links.

## The Seven Tiers

Each tier may only import from its own tier or lower. Tier assignment is both
**structural** (what does this module depend on?) and **conceptual** (what role
does it play?). A module with zero internal deps can still be Tier 4+ if its
role is infrastructure or tooling — e.g. `otel/` (zero deps, Tier 4),
`catalog/` (zero deps, Tier 6).

| Tier | Name               | Rule                                 | Modules |
| ---- | ------------------ | ------------------------------------ | ------- |
| 0    | Primitives         | No internal deps (or same-tier only) | 8       |
| 1    | Core Domain        | Depends on Tier 0                    | 5       |
| 2    | Domain Utilities   | Depends on Tier 0–1                  | 5       |
| 3    | Aggregation        | Depends on Tier 0–2                  | 5       |
| 4    | Infrastructure     | Depends on Tier 0–3                  | 23      |
| 5    | Composition        | Depends on Tier 0–4                  | 9       |
| 6    | Tooling & Examples | Depends on all                       | 13      |

**Total: 68 modules** across 69 `go.mod` files (68 modules + 1 root workspace
placeholder). Verify: `find . -name go.mod -not -path './vendor/*' | wc -l`

---

## Tier 0 — Primitives (leaf modules, no internal deps or same-tier only)

These are foundational building blocks. Some depend on each other (e.g.
`kv/` → `codec/`, `metaengine/` → `dedup/`) but never on higher tiers.

| Module            | Purpose                                                           |
| ----------------- | ----------------------------------------------------------------- |
| `id/`             | Branded IDs: `id.Of[T]`, `AggregateID`, `StreamID`, etc.          |
| `codec/`          | Payload encoding: JSON, CBOR, Raw                                 |
| `kv/`             | KV store: `Store`, `MemStore`, `TypedStore[T,K]`                  |
| `dedup/`          | Bounded dedup ring buffer                                         |
| `dispatcher/`     | Generic `Dispatcher[H, M]` with lifecycle mixin                   |
| `retry/`          | Re-export of go-retry: `Do`, `Config`, `Backoff`                  |
| `flightrecorder/` | Go 1.25 runtime/trace flight recorder wrapper                     |
| `metaengine/`     | Cost-based storage planner (the strategic future of this project) |

> **`metaengine/` is Tier 0 by design** (ADR-0062). The core planner has zero
> internal deps (stdlib + `database/sql` + `dedup/` only). The bridge to the
> CQRS event-sourcing world lives in `metaengine/projectionadapter/` (Tier 4).

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

| Module         | Purpose                                                        |
| -------------- | -------------------------------------------------------------- |
| `schema/`      | Upcaster, VersionedStore, Validator with `RegisterType[T]()`   |
| `snapshot/`    | Snapshot types + strategy (EveryNEvents, ReadPressure)         |
| `projection/`  | Projection interface (consumer-side)                           |
| `idempotency/` | Re-export of go-idempotency: Store, MemoryStore, ErrDuplicate  |
| `deriver/`     | Event-to-command derivation: Deriver, Then, Filter, Idempotent |

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

### Storage Backends

| Module            | Purpose                                                                            |
| ----------------- | ---------------------------------------------------------------------------------- |
| `storage/memory/` | In-memory test impls (MemoryStore, etc.)                                           |
| `storage/`        | SQL facade: EventStore, CommandStore, QueryStore, KV, relational, view, migrations |
| `storage/pebble/` | PebbleDB: EventStore, KVAdapter, Backend facade                                    |
| `storage/turso/`  | Turso embedded database connector                                                  |

### Security

| Module        | Purpose                                             |
| ------------- | --------------------------------------------------- |
| `signing/`    | Event signing: HMAC-SHA256, Ed25519, multisig       |
| `encryption/` | Payload encryption: XChaCha20-Poly1305, AES-256-GCM |

### Observability

| Module        | Purpose                                                                   |
| ------------- | ------------------------------------------------------------------------- |
| `otel/`       | Shared OTel helpers (zero internal deps, but conceptually infrastructure) |
| `prometheus/` | OTel-to-Prometheus metrics bridge                                         |

### Cross-Cutting

| Module        | Purpose                                                                           |
| ------------- | --------------------------------------------------------------------------------- |
| `middleware/` | Logging, Retry, Recovery, Validation, Idempotency, Metrics, OTel, Circuit Breaker |
| `testutil/`   | Shared test helpers (NewCmd, etc.)                                                |

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

| Module                            | Purpose                                             |
| --------------------------------- | --------------------------------------------------- |
| `metaengine/projectionadapter/`   | Wraps metaengine Store as projection.Projection     |
| `metaengine/pebbleengine/`        | Pebble-backed engine (LSM point reads)              |
| `metaengine/duckdbengine/`        | DuckDB-backed engine (columnar OLAP, CGo)           |
| `metaengine/pgengine/`            | Postgres-backed engine (JSONB + B-tree)             |
| `metaengine/irohengine/`          | Iroh Level 2 replication wrapper (CRDT convergence) |
| `metaengine/irohengine/loopback/` | Loopback transport (real TCP, no CGo)               |
| `metaengine/irohengine/quic/`     | QUIC transport (real Iroh, CGo)                     |

### Sub-Store Implementations

| Module                  | Purpose                                        |
| ----------------------- | ---------------------------------------------- |
| `idempotency/sqlstore/` | SQL-backed idempotency (SQLite/Postgres)       |
| `idempotency/kvstore/`  | KV-backed idempotency                          |
| `scheduling/sqlstore/`  | SQL-backed timer store (SQLite/Postgres/MySQL) |

---

## Tier 5 — Composition (depends on Tier 0–4)

One-call wiring that composes infrastructure into deployable bundles.

| Module            | Purpose                                                            |
| ----------------- | ------------------------------------------------------------------ |
| `stack/`          | Bundle abstraction + shared options (durability, codec, health)    |
| `stack/memory/`   | Memory preset                                                      |
| `stack/sqlite/`   | SQLite preset                                                      |
| `stack/duckdb/`   | DuckDB preset (CGo, columnar OLAP)                                 |
| `stack/pebble/`   | Pebble preset (LSM)                                                |
| `stack/postgres/` | Postgres preset                                                    |
| `stack/mysql/`    | MySQL/MariaDB preset                                               |
| `stack/turso/`    | Turso preset (embedded sync)                                       |
| `system/`         | Deployer-driven composition root (DomainConfig + DeploymentConfig) |

---

## Tier 6 — Tooling & Examples (depends on all)

Developer tools, code generators, linters, test harnesses, and usage demos.

| Module                       | Purpose                                             |
| ---------------------------- | --------------------------------------------------- |
| `catalog/`                   | API documentation generator (AsyncAPI, D2, OpenAPI) |
| `integration/`               | Cross-module integration tests                      |
| `benchkit/`                  | Factory-driven benchmarking suite                   |
| `stack/bench/`               | Stack-level benchmark presets                       |
| `cmd/cqrs-gen/`              | Code generator: typed handler registration          |
| `cmd/cqrs-lint/`             | Domain-aware linter (186 rules, 10 categories)      |
| `cmd/cqrs-bench/`            | CLI: benchmark any backend with workload profiles   |
| `cmd/api-stability/`         | API surface checker (golden file comparison)        |
| `cmd/doc-check/`             | Doc link verifier (Go import paths in markdown)     |
| `example/taskmanager/`       | Flagship full HTTP service example                  |
| `example/getting-started/`   | Minimal 80-line example                             |
| `example/readme-quickstart/` | README quickstart example                           |
| `event/v4/eventtest/`        | Test helpers: FakeStore, FakeBus, golden assertions |

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
3. **44 of 68 modules depend on `codec/`** — the true hub was invisible in the
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
