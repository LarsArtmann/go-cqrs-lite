# Infrastructure Recommendations

> Which storage engine fits which CQRS concern — and why. The deployer's
> decision guide for choosing a preset and splitting databases.

## TL;DR

Every CQRS system has **five distinct storage concerns** with different access
patterns. Matching the engine to the pattern is the single highest-leverage
infrastructure decision. The library encodes these recommendations as
**presets** so the deployer picks one line and gets a well-matched stack.

| Concern                       | Write pattern | Read pattern                 | Best engine         | Why                            |
| ----------------------------- | ------------- | ---------------------------- | ------------------- | ------------------------------ |
| **Event store**               | Append-only   | Range scan by aggregate      | LSM-tree or B+Tree  | Immutable, ordered, unbounded  |
| **Snapshots**                 | Point write   | Point read by key            | KV / B+Tree         | Get/Put by aggregate ID        |
| **Checkpoints**               | Point write   | Point read by key            | KV / B+Tree         | Get/Put by projection name     |
| **Command/query audit**       | Append-only   | Sequential / cursor scan     | B+Tree (SQL)        | Ordered logs, time-range query |
| **Read models** (projections) | Bulk upsert   | Point lookup + filtered list | KV / SQL / columnar | Depends on query shape         |

## The five concerns, deep-dive

### 1. Event store — the append-only log

**Access pattern:** writes are appends (never update, never delete); reads are
range scans over one aggregate's events ordered by version, plus a global
time-ordered scan (`ReadAll`) for projection replay.

**Why LSM-tree (Pebble) fits:** append-only writes are exactly what LSM-trees
are optimized for — sequential writes, no in-place updates. A prefix scan
loads one aggregate's events in version order efficiently.

**Why B+Tree (SQLite, Postgres) fits:** `SELECT * FROM events WHERE aggregate_id = ? ORDER BY version` is a textbook B+Tree range scan. The global
`ReadAll` is `ORDER BY occurred_at` — trivial in SQL, expensive in pure KV
(requires a secondary index, i.e. double-writes — see
[ADR-0009](adr/0009-pebble-scope-event-store-only.md)).

**Never use:** Redis (not durable by default), etcd (8 MB value limit, not a
log), DynamoDB at high throughput (no ordering, expensive scans).

### 2 & 3. Snapshots and checkpoints — simple key-value

**Access pattern:** `Get(key)` and `Put(key, value)`. No scans, no ordering,
no relationships.

**Why this is pure KV:** these are the simplest possible storage needs. Any
engine that supports point lookup works — SQL (`cqrs_snapshots`,
`cqrs_checkpoints` tables), Pebble (prefix-namespaced), or an external KV
store (Redis, etcd).

**Recommendation:** keep these co-located with the event store. They serve
event sourcing and share its lifecycle. The SQLite preset does this by default.

### 4. Command and query audit — the operational log

**Access pattern:** append-only writes, sequential reads with cursor-based
pagination, time-range queries for debugging ("what commands ran between
14:00 and 15:00?").

**Why SQL fits:** this is operational data you'll query ad-hoc — by time, by
aggregate, by command type. SQL's indexing and query flexibility pay off here.
This is a different access pattern from the event store (which is replayed,
not queried ad-hoc), which is why the multi-DB split separates them.

### 5. Read models / materialized views — query-shaped

**Access pattern:** bulk upserts (projection writes), point lookups by key,
and filtered/sorted list queries.

**Engine choice depends on the query shape:**

| Query shape                | Best engine           | Preset option          |
| -------------------------- | --------------------- | ---------------------- |
| Point lookup by ID         | KV (`cqrs_kv` table)  | default / `WithViewDB` |
| Filtered list + pagination | SQL (custom tables)   | SQL preset             |
| Full-text search           | SQLite FTS / Postgres | external               |
| Aggregations / analytics   | Columnar (external)   | external               |
| Relationship traversal     | Graph (external)      | external               |

The library's `kv.TypedStore` and `stack.Materialize` handle the common case
(point lookup + tombstone-aware list). For richer query needs, the deployer
can point `WithViewDB` at a dedicated database and the consumer writes custom
projection handlers that populate SQL tables directly.

## Preset selection guide

| Situation                              | Recommended preset | Why                                    |
| -------------------------------------- | ------------------ | -------------------------------------- |
| **Development / testing**              | `stack/memory`     | Zero setup, ephemeral, fastest         |
| **Single-node, embedded**              | `stack/sqlite`     | One file, zero ops, WAL + busy timeout |
| **Single-node, high write throughput** | `stack/pebble`     | LSM-tree optimized for appends         |
| **Multi-node, managed**                | `stack/postgres`   | LISTEN/NOTIFY for distributed bus      |
| **Edge / offline-first**               | `stack/turso`      | LibSQL sync to remote, works offline   |

All presets implement the same `stack.Bundle` contract (verified by
`contracttest.RunSuite`), so switching is a one-line change with zero consumer
code impact.

## The multi-DB split (SQLite flagship)

The SQLite preset supports splitting concerns across separate database files —
the library's recommendation encoded as deployer options:

```go
bundle, err := sqlite.New("primary.db",
    sqlite.WithEventDB("events.db"),   // events + snapshots + checkpoints
    sqlite.WithQueryDB("queries.db"),  // command + query audit logs
    sqlite.WithViewDB("views.db"),     // materialized views (KV)
)
```

| Database     | Contains                        | Rationale                                                               |
| ------------ | ------------------------------- | ----------------------------------------------------------------------- |
| **Event DB** | events, snapshots, checkpoints  | The event-sourcing write model — isolated from read traffic             |
| **Query DB** | commands, queries               | Operational audit log — ad-hoc query load isolated from the write model |
| **View DB**  | materialized views (`cqrs_kv`)  | Read-model scans don't contend with event appends                       |
| **Primary**  | (unused when all three are set) | Opened for schema; stores are overridden                                |

**When to split:** production deployments with concurrent readers and writers.
A single database is fine for development and low-traffic apps. The split
eliminates reader/writer contention at the file level.

**When NOT to split:** development, tests, low-traffic apps. The default
single-database mode is simpler and sufficient.

## Deployer vs. consumer responsibility

| Who          | Decides                                                     | How                                          |
| ------------ | ----------------------------------------------------------- | -------------------------------------------- |
| **Deployer** | Which engine (memory/SQLite/Pebble/Postgres/Turso)          | Picks a preset constructor                   |
| **Deployer** | Whether to split databases (multi-DB)                       | `WithEventDB` / `WithQueryDB` / `WithViewDB` |
| **Deployer** | Distributed bus (Postgres LISTEN/NOTIFY)                    | `WithDistributedBus`                         |
| **Consumer** | Domain logic: deciders, event types, command/query handlers | Imports `event`, `command`, `decider`        |
| **Consumer** | Projection logic: how events materialize into views         | `stack.Materialize` + `OnCreate` etc.        |
| **Consumer** | Never imports a backend driver                              | No `storage/`, `pebble/`, `turso/` imports   |

The consumer's code is identical regardless of which backend the deployer
chose. This is the core design goal: **infrastructure is a deployer concern,
not a consumer concern.**
