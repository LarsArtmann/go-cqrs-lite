# ADR-0098: Metaengine Persistence Enum

Date: 2026-08-04

## Status

Accepted

## Context

The metaengine `EngineProfile` declares cost (NsPerOp, ReadCosts), capability
(Supports, Layouts), topology (Replication, NetworkRTT), and degradation
(DegradedADTs) — but **not durability**. Whether an engine's data survives
process exit is implicit in constructor names and comments, invisible to the
planner.

### Concrete failure mode

A consumer deploys with:

```go
store, _ := metaengine.Plan(
    []metaengine.Engine{metaengine.NewMemoryEngine()},
    userQuery,
)
```

Everything works in dev. In production, the process restarts (deploy, crash,
OOM kill) and **every projection is gone** — silently. No log, no warning, no
diagnostic. The only path to discovering this is data loss in production.

## Decision

Add a `Persistence` type to `EngineProfile` with two levels:

```go
type Persistence string

const (
    PersistenceVolatile   Persistence = ""             // zero value = safe default
    PersistencePersistent Persistence = "persistent"
)
```

### Why volatile is the zero value

Mirrors the `Replication` pattern (`ReplicationNone = ""`). If you forget to
declare persistence, the planner assumes the worst and warns — it does NOT
silently assume durability. Safe default = loud warning, not false security.

### Why two levels (not three or four)

Three alternatives were considered and rejected:

| Alternative                            | Why rejected                                                                                                                                                 |
| -------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| `PersistenceRemote` (3rd level)        | Double-counts NetworkRTT + Replication. "Remote" is already modeled by those orthogonal fields.                                                              |
| `PersistenceDiskRelaxed` (fsync tiers) | `stack.Durability` (Strict/Normal/Relaxed) already models this orthogonal axis.                                                                              |
| `PersistenceCache` / TTL               | Different axis entirely (eviction, not persistence). Proven via 2x2 orthogonality matrix. Redis maps to all 4 quadrants. YAGNI until a cache engine arrives. |

### Why dynamic (not a static profile constant)

Three engines are volatile OR persistent depending on constructor arguments:

| Constructor                      | Persistence                |
| -------------------------------- | -------------------------- |
| `NewMemoryEngine()`              | Volatile (pure RAM)        |
| `NewSQLiteEngine(":memory:")`    | Volatile (RAM-backed SQL)  |
| `NewSQLiteEngine("file:app.db")` | Persistent (disk file)     |
| `NewPebbleEngine("")`            | Volatile (`vfs.NewMem()`)  |
| `NewPebbleEngine("/data")`       | Persistent (LSM on disk)   |
| `duckdb.New("")`                 | Volatile (`:memory:`)      |
| `duckdb.New("analytics.db")`     | Persistent (disk file)     |
| `pgengine.New(dsn)`              | Persistent (remote server) |

The persistence value is set at construction time via a struct field.

## Implementation

Persistence follows the identical 6-step pattern as Replication (ADR-0093):

1. `persistence.go` — type + constants + `IsVolatile()`/`IsPersistent()` helpers
2. `EngineProfile.Persistence` field
3. `durabilityRule` planner rule (`rule_durability.go`)
4. Wired into `defaultRules()` in `rules.go`
5. Exposed in `CollectionInfo` + `Store.Persistence()` accessor + `SerializableQuery`
6. Shown in `Doctor()` (`--- Persistence ---` section) + `ExplainPlan()`

### durabilityRule diagnostics

| Condition                                      | Diagnostic                                           |
| ---------------------------------------------- | ---------------------------------------------------- |
| Volatile engine, no persistent alternative     | **WARN**: "projection will be lost on restart"       |
| Volatile engine, persistent alternative exists | **INFO**: shows alternative engine name + cost delta |
| Persistent engine                              | Silent (happy path)                                  |

## Consequences

- The planner now warns when a query is routed to a volatile engine with no
  persistent alternative, preventing silent data loss on restart.
- All 5 existing engines (Memory, SQLite, Pebble, DuckDB, Postgres) declare
  their persistence honestly.
- Future engines (Iroh, CockroachDB) will set `PersistencePersistent` in their
  profiles.
- The `EvictionPolicy` axis (TTL/LRU for cache engines) is deliberately NOT
  implemented — YAGNI until a cache engine exists.
