# Deployer-First Architecture Audit
\n> **Status:** RESOLVED — deployer-first architecture shipped

> **Date:** 2026-06-23
> **Question:** How well does the library achieve the goal that consumers
> should NOT decide on infrastructure — the deployer picks where data lives
> and what stores it, with the option to split concerns across multiple
> databases?

---

## Goal Statement

> My Goal is for consumers of this lib should NOT decide on the
> implementation of infrastructure e.g. DB, MessageBuses, ....
> They should have a simple API that allows the person deploying the App,
> to decide where they want to keep there data and what they want to store.
> We should have recommendations though, e.g. some meta data or projections
> aka materialized views may be better places in a SQL/KV/columnar/Graph
> DB - but we should also be able to run fully with just a SQLite + Memory
> setup, maybe even multiple SQLite DBs e.g. 1 for Command + Event
> Sourcing, 1 for Query (logs), and 1 for materialized views.

---

## Two-Path Architecture

The library provides **two paths** for assembling a `stack.Bundle`, each
serving a different deployer need:

### Path 1: Presets (homogeneous — one engine for everything)

```go
bundle, err := sqlite.New("app.db")    // one line, everything in SQLite
bundle, err := memory.New()             // one line, everything in memory
bundle, err := pebble.New("data")       // one line, everything in Pebble
bundle, err := postgres.New(dsn)        // one line, everything in Postgres
bundle, err := turso.New("app.db")      // one line, everything in LibSQL
```

The deployer picks one engine. All six concerns (events, commands, queries,
snapshots, checkpoints, read models) use that engine. Consumer code is
identical regardless of which preset was chosen.

### Path 2: Manual assembly (heterogeneous — mix engines per concern)

```go
bundle, err := stack.New(
    stack.WithEventStore(pebbleBackend.EventStore()),   // Pebble for events
    stack.WithReadModels(sqlKVStore),                    // SQL for views
    stack.WithCheckpointStore(redisCheckpoint),          // Redis for checkpoints
    stack.WithBus(cqrswatermill.NewEventBus()),
)
```

The deployer mixes engines per-concern. Each `With*` option sets one
capability field on the Bundle. Consumer code is still identical.

**This is the key design insight:** presets are a convenience layer over the
`With*` API. The heterogeneous path already exists and works. What's thin is
the middle ground — guided patterns and examples for common heterogeneous
combos.

---

## Preset Coverage Matrix

| Preset     | One-line API           | Multi-DB split                               | Distributed bus                      | Wrapped Bundle extras                  |
| ---------- | ---------------------- | -------------------------------------------- | ------------------------------------ | -------------------------------------- |
| `memory`   | `memory.New()`         | —                                            | —                                    | —                                      |
| `sqlite`   | `sqlite.New("app.db")` | `WithEventDB` / `WithQueryDB` / `WithViewDB` | —                                    | —                                      |
| `turso`    | `turso.New("app.db")`  | Local only (sync mode: no)                   | —                                    | `Sync()` → Push/Pull/Checkpoint        |
| `pebble`   | `pebble.New("data")`   | ❌                                           | —                                    | `Checkpoint()`, `Metrics()`, `Flush()` |
| `postgres` | `postgres.New(dsn)`    | ❌                                           | `WithDistributedBus` (LISTEN/NOTIFY) | —                                      |

### Multi-DB split detail

The SQLite and Turso presets support splitting concerns across separate
database files:

```go
bundle, err := sqlite.New("primary.db",
    sqlite.WithEventDB("events.db"),   // events + snapshots + checkpoints
    sqlite.WithQueryDB("queries.db"),  // command + query audit logs
    sqlite.WithViewDB("views.db"),     // materialized views (cqrs_kv)
)
```

| Database     | Contains                       | Rationale                                              |
| ------------ | ------------------------------ | ------------------------------------------------------ |
| **Event DB** | events, snapshots, checkpoints | Event-sourcing write model, isolated from read traffic |
| **Query DB** | commands, queries              | Operational audit log, isolated from write model       |
| **View DB**  | materialized views (`cqrs_kv`) | Read-model scans don't contend with event appends      |

Postgres and Pebble do **not** support this split.

---

## Storage Engine Inventory

### Event Store implementations

| Engine                    | Implementation                   | Used by presets |
| ------------------------- | -------------------------------- | --------------- |
| In-memory                 | `storage.MemoryStore`            | memory          |
| SQLite (modernc, pure Go) | `storage.SQLEventStore`          | sqlite          |
| Turso/LibSQL              | `storage.SQLEventStore` (shared) | turso           |
| Postgres (pgx)            | `storage.SQLEventStore` (shared) | postgres        |
| Pebble (LSM-tree)         | `pebble.EventStore`              | pebble          |
| Redis                     | —                                | —               |
| DynamoDB                  | —                                | —               |
| MongoDB                   | —                                | —               |

### KV / Read-Model Backends

| Backend               | Implementation       | Used by presets         |
| --------------------- | -------------------- | ----------------------- |
| In-memory             | `kv.MemStore`        | memory                  |
| SQL (`cqrs_kv` table) | `storage.SQLKVStore` | sqlite, turso, postgres |
| Pebble                | `pebble.KVAdapter`   | pebble                  |
| Redis                 | —                    | —                       |
| Columnar (ClickHouse) | —                    | —                       |
| Graph (Neo4j)         | —                    | —                       |
| Document (MongoDB)    | —                    | —                       |

### Wrappers (composable layers over any store)

| Wrapper                     | Purpose                                |
| --------------------------- | -------------------------------------- |
| `schema.VersionedStore`     | Schema evolution via upcasters on load |
| `encryption.encryptedStore` | Transparent payload encryption at rest |

---

## Consumer-Facing API

These are the functions app developers call. None of them reference a storage
engine — they all operate on the abstract `Bundle`.

| Function                                             | Purpose                               |
| ---------------------------------------------------- | ------------------------------------- |
| `stack.Repository[State](bundle, decider)`           | Typed aggregate repository            |
| `stack.TypedRepository[State, Cmd](bundle, decider)` | Compile-time command-type-bound repo  |
| `stack.ReadModel[T, K](bundle, codec)`               | Typed KV store for views              |
| `stack.NewMaterialize[V, K](bundle, codec, keyFunc)` | Tombstone-aware projection builder    |
| `bundle.CatchUpSubscriber()`                         | Replay + live + checkpoint subscriber |
| `stack.QueryAuditMiddleware(bundle, level)`          | Query audit logging middleware        |

---

## Goal Checklist

| Goal dimension                     | What we have                                                                   | What's missing                                                                        | Grade |
| ---------------------------------- | ------------------------------------------------------------------------------ | ------------------------------------------------------------------------------------- | ----- |
| **Consumers don't import storage** | Presets + `stack.New(With*)`; consumer code is engine-agnostic                 | `example/deployer-first` uses manual wiring (comment added, not refactored to preset) | B+    |
| **Simple deployer API**            | 5 presets, all one-line constructors                                           | Postgres + Pebble missing multi-DB split                                              | B     |
| **Recommendations per concern**    | `INFRASTRUCTURE_RECOMMENDATIONS.md` maps 5 concerns to engines                 | No recommendation for bus choice (only Postgres has distributed)                      | A-    |
| **SQLite + Memory only**           | Both fully working standalone                                                  | —                                                                                     | A     |
| **Multiple SQLite DBs**            | `WithEventDB`/`WithQueryDB`/`WithViewDB`                                       | Same split unavailable on Postgres and Pebble                                         | B+    |
| **Heterogeneous engine mixing**    | `stack.New(WithEventStore(pebble), WithReadModels(sql))` works                 | No preset/builder for common combos; no example demonstrating it                      | B-    |
| **Specialized read-model engines** | `kv.Store` interface is extensible; `WithReadModels(myStore)` accepts anything | Zero implementations for columnar/graph/document; no docs on wiring external engines  | D     |
| **Distributed bus**                | Postgres LISTEN/NOTIFY via `WithDistributedBus`                                | No Kafka, NATS, Redis Pub/Sub adapters                                                | C+    |

---

## Honest Gaps

| #   | Gap                                           | Impact                                                                   | Effort                                      |
| --- | --------------------------------------------- | ------------------------------------------------------------------------ | ------------------------------------------- |
| 1   | **Postgres has no multi-DB split**            | Production users can't isolate concerns on the most common production DB | Medium — pattern exists in sqlite, port it  |
| 2   | **Pebble has no multi-DB split**              | Can't isolate Pebble stores by column family                             | Low-Medium                                  |
| 3   | **No distributed bus except Postgres**        | Kafka/NATS users must hand-wire via Watermill router                     | High per-adapter                            |
| 4   | **No external KV/read-model adapters**        | Columnar/graph/document are "bring your own" with zero guidance          | High                                        |
| 5   | **No heterogeneous example**                  | The killer feature (mix engines per-concern) has zero demonstration      | Low — just an example                       |
| 6   | **deployer-first example uses manual wiring** | Flagship example doesn't showcase the preset one-liner path              | Low                                         |
| 7   | **Turso multi-DB + sync incompatible**        | `WithEventDB` etc. silently ignored in sync mode                         | Low — documented but could error explicitly |

---

## What Works Well

1. **The preset abstraction is real.** Switching engines is a one-line change.
   The `contracttest.RunSuite` proves all presets satisfy the same behavioral
   contract.

2. **The `With*` API is the heterogeneous escape hatch.** A deployer who wants
   Pebble events + SQL views can do it today without any new code. The Bundle
   doesn't care that the stores came from different engines.

3. **The consumer API is genuinely storage-agnostic.** `Repository[State]`,
   `ReadModel[T,K]`, `NewMaterialize[V,K]`, `CatchUpSubscriber()` — none of
   these reference a storage engine. A consumer's code compiles unchanged
   whether the deployer chose memory, SQLite, or a custom heterogeneous mix.

4. **Multi-DB split on SQLite/Turso is exactly the pattern described in the
   goal.** Events + snapshots + checkpoints in one DB, commands + queries in
   another, materialized views in a third.

5. **Documentation maps concerns to engines.** `INFRASTRUCTURE_RECOMMENDATIONS.md`
   explains WHY each engine fits each access pattern, with a preset selection
   guide and multi-DB rationale.
