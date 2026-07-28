# Seven-Tier Model — go-cqrs-lite

> Originally titled "Four-Tier Model." The name was a misnomer — the model
> describes seven numbered tiers (0–6). Renamed for accuracy. The filename
> is retained to avoid breaking existing links.

> Replaces the fake "7-layer" system that claimed clean stratification but had
> cross-layer dependencies leaking through go.mod files and test-only deps
> inflating production dep counts.

## The Real Tiers

Each tier may only import from its own tier or lower.

### Tier 0 — Primitives (leaf modules, no internal deps)

| Module        | Purpose                                                  |
| ------------- | -------------------------------------------------------- |
| `id/`         | Branded IDs: `id.Of[T]`, `AggregateID`, etc.             |
| `codec/`      | Payload encoding: JSON, CBOR, Raw                        |
| `kv/`         | Layer-0 KV store: `Store`, `MemStore`, `TypedStore[T,K]` |
| `dedup/`      | Bounded dedup ring buffer                                |
| `dispatcher/` | Generic `Dispatcher[H, M]` with lifecycle                |

### Tier 1 — Core Domain (depends on Tier 0)

| Module        | Purpose                                            |
| ------------- | -------------------------------------------------- |
| `event/`      | EventSink, EventSource, Store, Bus, ImmutableEvent |
| `command/`    | Dispatcher, Handler, Middleware, Bus               |
| `query/`      | Dispatcher, Handler, Pagination                    |
| `scheduling/` | Durable deadline timers                            |

### Tier 2 — Domain Utilities (depends on Tier 0-1)

| Module         | Purpose                             |
| -------------- | ----------------------------------- |
| `schema/`      | Upcaster, VersionedStore, Validator |
| `snapshot/`    | Snapshot types + strategy           |
| `projection/`  | Projection interface                |
| `idempotency/` | Command dedup store                 |
| `deriver/`     | Event→command derivation            |

### Tier 3 — Aggregation (depends on Tier 0-2)

| Module            | Purpose                                |
| ----------------- | -------------------------------------- |
| `decider/`        | `Decider[State]`, `Repository[State]`  |
| `graph/`          | Graph projection + memory driver       |
| `scenario/`       | Fluent BDD test DSL                    |
| `projectionhost/` | Managed projection lifecycle + DLQ     |
| `listing/`        | Aggregate status + tombstone detection |

### Tier 4 — Infrastructure (depends on Tier 0-3)

| Module            | Purpose                                      |
| ----------------- | -------------------------------------------- |
| `storage/memory/` | In-memory test impls                         |
| `signing/`        | Event signing/verification                   |
| `encryption/`     | Payload encryption                           |
| `otel/`           | Shared OTel helpers                          |
| `middleware/`     | Cross-cutting: logging, retry, etc.          |
| `storage/`        | SQL stores (PG/SQLite/Turso/DuckDB) + Pebble |
| `transport/http/` | SSE broker                                   |
| `transport/grpc/` | gRPC transport                               |
| `watermill/`      | Watermill bridges                            |
| `prometheus/`     | OTel→Prometheus bridge                       |

### Tier 5 — Composition (depends on Tier 0-4)

| Module            | Purpose             |
| ----------------- | ------------------- |
| `stack/`          | Bundle + presets    |
| `stack/memory/`   | Memory preset       |
| `stack/sqlite/`   | SQLite preset       |
| `stack/pebble/`   | Pebble preset       |
| `stack/postgres/` | Postgres preset     |
| `stack/turso/`    | Turso preset        |
| `stack/duckdb/`   | DuckDB preset (CGo) |

### Tier 6 — Tooling & Examples (depends on all)

| Module               | Purpose                     |
| -------------------- | --------------------------- |
| `catalog/`           | API documentation generator |
| `integration/`       | Cross-module tests          |
| `cmd/cqrs-gen/`      | Code generator              |
| `cmd/api-stability/` | API surface checker         |
| `cmd/doc-check/`     | Doc link verifier           |
| `example/*`          | Usage demos                 |

## Why the Old 7-Layer System Was Fake

1. **kv/ claims Layer 0 but depends on codec/** — not a true leaf
2. **event/ claims Layer 1 but depends on Tier 2-4 modules** via test deps that leak into go.mod
3. **38 of 48 modules depend on codec/** — the true "Layer 0" hub was invisible in the old system
4. **command/ has a hard compile dependency on event/** — violates CQRS separation at the compile level

## What This Changes

- Honest dependency tracking — no more claiming "Layer 0" while importing codec/
- Clear upgrade path: v4 moves shared types to `id/` + `metadata/`, adds `context.Context` to kv/, splits `storage/`
- The four-tier model makes it obvious which modules are safe to change without downstream impact
