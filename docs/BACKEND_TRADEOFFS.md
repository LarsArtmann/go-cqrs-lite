# Backend Tradeoffs

> **Single source of truth for "which backend should I choose?"**

This document unifies backend comparison across every dimension that matters:
durability, speed, resource usage, and operational burden. Every claim here is
backed by benchmark data in [`docs/benchmarks/`](benchmarks/) and
[`docs/performance.md`](performance.md).

## Quick Decision Matrix

| Dimension | memory | sqlite | pebble | postgres | turso | duckdb |
|-----------|--------|--------|--------|----------|-------|--------|
| **Persistent** | No | Yes | Yes | Yes | Yes | Yes |
| **Embedded** | Yes | Yes | Yes | No (server) | Yes | Yes |
| **Distributed** | No | No | No | Optional (LISTEN/NOTIFY) | Via sync | No |
| **Remote Sync** | No | No | No | No | Yes | No |
| **CGo Required** | No | No | No | No | No | Yes |
| **OLAP Optimized** | No | No | No | No | No | Yes |
| **Write P50** (small profile) | 350ns | 212us | 11.6us | 47/s (strict) | ~sqlite | 480us |
| **Read P50** (small profile) | 170ns | 225us | 55.6us | 137us | ~sqlite | 1.01ms |
| **Durability Tiers** | normal, relaxed | strict, normal, relaxed | strict, normal, relaxed | strict, normal, relaxed | strict, normal, relaxed | normal |

> **Note on write numbers**: SQLite/Postgres/DuckDB write numbers reflect
> `BatchSize=1` (one fsync per event). See [BatchSize Semantics](#batchsize-semantics).

## Durability Vocabulary

All backends now support a shared [DurabilityTier] vocabulary via
`stack.WithDurability(tier)`:

| Tier | SQLite | Postgres | Pebble | Meaning |
|------|--------|----------|--------|---------|
| `DurabilityStrict` | synchronous=FULL | synchronous_commit=on | WAL on, sync writes | Every commit fsyncs to disk |
| `DurabilityNormal` | synchronous=NORMAL | synchronous_commit=off | WAL on, default | Safe against app crash; small window on kernel crash |
| `DurabilityRelaxed` | synchronous=OFF | synchronous_commit=off | DisableWAL=true | Data may be lost on crash |

**Default**: Every preset ships with `DurabilityNormal`. This matches the
pre-existing behavior of each backend.

### The synchronous_commit Lever (Postgres)

The single biggest performance lever for Postgres:

| Setting | Writes/sec | Relative |
|---------|-----------|----------|
| `synchronous_commit=on` (strict) | 47 | 1x |
| `synchronous_commit=off` (normal) | 18,200 | 387x |

Both settings are safe against application crashes. The difference is only in
the kernel-crash window (a few hundred ms of committed transactions).

### How to Set Durability

```go
// Per-preset option:
b, _ := sqlite.New(dsn, sqlite.WithDurability(stack.DurabilityStrict))

// Via cqrs-bench CLI:
// cqrs-bench run --backend postgres --dsn ... --durability strict
```

## BatchSize Semantics

The `dev` and `small` benchmark profiles use `BatchSize=1` — one event per
`Save()` call. This means one full transaction (BEGIN, INSERT, COMMIT) per
event, with one fsync per commit under strict durability.

**This is intentional for worst-case testing** but misleading for capacity
planning. For fair write comparisons:

- Use `medium` (BatchSize=5) or `large` (BatchSize=10)
- Or set `--durability relaxed` to remove fsync from the equation
- Or use the raw-sink phase (prebuilt events, isolating pure Save cost)

## When to Use Each Backend

### memory — Development, Testing, CI

**Choose when**: You need zero-dependency, instant setup, and data loss on
restart is acceptable.

```go
bundle, _ := memory.New()
```

- **Pros**: Fastest possible (no disk, no serialization overhead for
  in-process objects), zero configuration, zero CGo
- **Cons**: Data lost on process exit, single-process only, no durability
- **Typical write latency**: ~350ns/event
- **Use for**: Unit tests, integration tests, local development, CI pipelines

### sqlite — Single-Process Production, Edge Computing

**Choose when**: You need persistence in a single-process deployment without
external dependencies.

```go
bundle, _ := sqlite.New("app.db",
    sqlite.WithDurability(stack.DurabilityNormal),
    sqlite.WithPragmas(sqlopt.WithOptimizations()),
)
```

- **Pros**: Pure-Go driver (modernc.org/sqlite), zero server administration,
  WAL mode enables concurrent reads during writes, mature and battle-tested
- **Cons**: Single-writer (WAL serializes writes), single-process, no
  distributed pub/sub
- **Typical write latency**: ~200us/event (BatchSize=1, normal durability)
- **Use for**: Desktop apps, IoT/edge, small services, development with
  persistence

### pebble — High-Throughput Embedded Event Sourcing

**Choose when**: You need maximum embedded write throughput for event sourcing
and your read patterns are point-lookups (stream loads by ID, snapshots).

```go
bundle, _ := pebble.New(dir,
    pebble.WithDurability(stack.DurabilityNormal),
)
```

- **Pros**: 10-20x faster writes than SQLite (LSM-optimized for append-only
  workloads), bloom filters for fast point reads, concurrent compaction,
  snapshot-based backups, embedded (no server)
- **Cons**: LSM read amplification for range scans, no SQL query engine, no
  built-in read-model queries (use kv.Store interface)
- **Typical write latency**: ~12us/event (BatchSize=1)
- **Use for**: High-throughput event sourcing where you control the read path

### postgres — Multi-Process, Distributed, Mission-Critical

**Choose when**: You need multi-process deployment, ACID guarantees at scale,
SQL query power for read models, or distributed event propagation.

```go
bundle, _ := postgres.New(dsn,
    postgres.WithDurability(stack.DurabilityStrict),
    postgres.WithPoolSize(20, 5),
    postgres.WithStatementTimeout(30 * time.Second),
)
```

- **Pros**: ACID, SQL engine for complex read-model queries, LISTEN/NOTIFY for
  cross-process event distribution, connection pooling, mature operational
  tooling
- **Cons**: External server (ops burden), network latency, synchronous_commit=on
  is slow (47 writes/sec at strict, 18K at normal)
- **Typical write latency**: 137us (normal) / ~20ms (strict) per event
- **Use for**: Production multi-process services, financial systems, any
  deployment where data durability is non-negotiable

### turso — Edge Sync, Offline-First, Multi-Device

**Choose when**: You need local-first data with optional sync to a remote
server, or you're building offline-capable applications.

```go
bundle, _ := turso.New("app.db") // local-only
// or:
bundle, _ := turso.NewSync(ctx, "app.db", remoteURL, token)
```

- **Pros**: Works offline, syncs when online, SQLite-compatible (libSQL fork),
  embedded, multi-database support
- **Cons**: Single-writer (like SQLite), sync adds complexity, experimental MVCC
- **Typical write latency**: Similar to SQLite
- **Use for**: Mobile apps, edge computing with sync, offline-first
  architectures

### duckdb — Analytics, OLAP, Reporting

**Choose when**: Your read pattern is analytical (aggregations, scans,
GROUP BY) rather than transactional (point reads, single-stream loads).

```go
bundle, _ := duckdb.New("analytics.db",
    duckdb.WithThreads(4),
    duckdb.WithMemoryLimit("1GB"),
)
```

- **Pros**: Columnar storage optimized for analytical queries, in-process (no
  server), excellent scan and aggregation performance
- **Cons**: CGo required (C++ engine, ~30-50MB binary), poor single-row INSERT
  performance (columnar rewrite cost), not designed for OLTP
- **Typical write latency**: ~480us/event (BatchSize=1 — columnar penalty)
- **Use for**: Analytics read models, reporting projections, dashboard backends

## Capabilities API

Every preset declares its capabilities via `Bundle.Capabilities()`:

```go
caps := bundle.Capabilities()
// caps.Persistent, caps.Embedded, caps.Distributed, caps.OLAP,
// caps.CGoRequired, caps.SyncEnabled, caps.DurabilityRange
```

This enables programmatic backend selection (e.g., "find me a persistent,
non-CGo backend that supports strict durability").

## Mixed Workload Benchmarking

The benchmark suite now includes a mixed read-during-write phase that runs
N writers and M readers concurrently against the same store. This answers the
real production question: "can the backend serve reads while writes hammer it?"

```bash
cqrs-bench run --backend sqlite --profile small
# Output includes:
# Mixed Workload (concurrent reads + writes):
#   Write (under read load): P50=... P95=... P99=... Max=...
#   Read (under write load): P50=... P95=... P99=... Max=...
#   Writers=4 Readers=1 | writes=... reads=...
```

Reader count is derived from `Profile.ReadRatio * Config.Concurrency`.

## Related Documents

- [Performance benchmarks](performance.md) — Raw numbers and methodology
- [Backend comparison report](benchmarks/2026-07-31_backend-comparison.md) —
  Dated full-suite results across all 5 backends
- [Storage guide](STORAGE_GUIDE.md) — Access-pattern based backend selection
- [Consistency model](CONSISTENCY_MODEL.md) — Per-backend durability guarantees
- [Stack presets](PRESETS.md) — Bundle composition and option reference
