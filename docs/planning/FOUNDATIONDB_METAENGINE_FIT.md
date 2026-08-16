# FoundationDB × Metaengine — Fit Analysis

> A design research report: could (and should) Apple's FoundationDB become a
> metaengine storage backend?
>
> **Status:** Research / design proposal — no code written
> **Date:** 2026-08-10
> **Primary sources:** https://apple.github.io/foundationdb/ (v7.3.79 docs),
> `pkg.go.dev/github.com/apple/foundationdb/bindings/go`, upstream repo README.

---

## Table of Contents

1. [TL;DR](#1-tldr)
2. [What Is FoundationDB](#2-what-is-foundationdb)
3. [How the Metaengine Engine Contract Works](#3-how-the-metaengine-engine-contract-works)
4. [The Mapping: FDB Capability → Metaengine ADT](#4-the-mapping-fdb-capability--metaengine-adt)
5. [Fit Analysis: Where FDB Wins](#5-fit-analysis-where-fdb-wins)
6. [Fit Analysis: Where FDB Loses](#6-fit-analysis-where-fdb-loses)
7. [Cost Model: What a FoundationDB EngineProfile Would Look Like](#7-cost-model-what-a-foundationdb-engineprofile-would-look-like)
8. [The Replication Story: FDB vs the Current Replication Model](#8-the-replication-story-fdb-vs-the-current-replication-model)
9. [The StreamLog / Journal Problem](#9-the-streamlog--journal-problem)
10. [Alternative Roles: Not Just an Engine](#10-alternative-roles-not-just-an-engine)
11. [Module and Packaging Strategy](#11-module-and-packaging-strategy)
12. [Effort and Risk Assessment](#12-effort-and-risk-assessment)
13. [Recommendation](#13-recommendation)
14. [Verified External Claims](#14-verified-external-claims)

---

## 1. TL;DR

**FoundationDB is a legitimately viable — though operationally heavy — 9th
metaengine backend.** It is a perfect _point-lookup and atomic-counter_ engine,
the only engine that natively replicates (MultiLeader topology, zero write
loss), and the only engine with a native push change-notification mechanism
(watches) that maps directly onto metaengine's watchers. Its weaknesses are
exactly where the current engine roster is already strong (SQL pushdown, big
values, OLAP scans), so it adds a new axis — **shared durable state across
processes** — rather than competing head-on.

**The recommendation is: build it, but with a specific scope.**

| Dimension                   | Verdict                                                                          |
| --------------------------- | -------------------------------------------------------------------------------- |
| Map / Set / Counter         | ✅ Native and excellent (atomic ops, watches, O(1)-ish point reads)              |
| SortedMap (secondary index) | ✅ Via the documented simple-index key pattern, ACID-consistent                  |
| Multimap / Log              | ✅ Via sequence-keyed, ordered composite keys                                    |
| StreamLog / Journal         | ⚠️ Workable per-stream, **unsafe globally** (10 MB txn cap)                       |
| Replicated writes           | ✅ Only engine that is honest about replication                                  |
| Watchers                    | ✅ Native push notification (no polling)                                         |
| Degraded ADTs               | 🔸 Vector = the _FDB vector recipe_ (array, not ANN); Search/Spatial = O(N) scan |
| OLAP / analytics            | ❌ Weakest spot; row B-tree / Redwood, no columnar engine                        |
| Operations                  | ❌ Heavyweight — separate fdbserver processes (NixOS module, **not** embedded)   |

**Why not "just use it as the whole system"?** FDB hard-limits transactions
(10 MB affected data, 5 s lifetime) and deliberately has no SQL/query language.
It needs constant sharding-away of large scans. Postgres/Turso-plus-layers, or
Dgraph, already cover the "one rich engine" sales pitch better — FDB's pitch is
**shared, duplicated, transactional state across processes**, which nothing in
the current roster offers.

The full scoring table (every ADT × metaengine relevance):

| Metaengine ADT               | FDB Support                          | FDB Complexity                      | vs Best-In-Roster                     | Verdict               |
| ---------------------------- | ------------------------------------ | ----------------------------------- | ------------------------------------- | --------------------- |
| Map                          | Native KV                            | O(logN) range tree, ~O(1) practical | Pebble O(1), PG O(logN)               | ✅ Competitive        |
| Set                          | Native KV presence                   | same                                | Pebble O(1)                           | ✅ Competitive        |
| Counter                      | Atomic Add (O(1))                    | O(1) read, ~O(1) write              | PG O(1)                               | ✅ Best-in-class      |
| SortedMap                    | Simple-index pattern                 | O(logN + k)                         | PG O(logN)                            | ✅ Native parity      |
| Multimap                     | Key-per-entry, ordered               | O(logN + k)                         | Pebble/PG O(logN)                     | ✅ Native parity      |
| Log                          | Seq-keyed append                     | O(logN) tail                        | Pebble O(logN)                        | ✅ Native parity      |
| StreamLog                    | Per-stream append OK                 | O(logN)                             | Pebble O(logN)                        | ⚠️ Works per-stream    |
| **StreamLog global journal** | ❌ 10 MB txn cap                     | n/a                                 | Pebble/SQLite                         | ❌ **Does not scale** |
| Graph                        | Adjacency-key pattern                | O(N^d)                              | Dgraph native                         | ❌ Avoid              |
| Vector (ANN)                 | ❌ (vector recipe = array, not HNSW) | O(N) scan                           | DuckDB O(1)-ish / PG pgvector O(logN) | ❌ Avoid              |
| Search (FTS)                 | ❌ no FTS core                       | O(N) substring scan                 | Dgraph @index(term)                   | ❌ Avoid              |
| Spatial                      | ❌ no spatial core                   | O(N) scan                           | (best: PG PostGIS-ish via SQL)        | ❌ Avoid              |

_Importantly: FDB's **excluded** ADTs (Graph/Search/Spatial/Vector) are the
same ones the current planner already marks degraded on simple KV engines —
so the engine can honestly declare them degraded and let the planner route
away. The planner already has the machinery for exactly this._

---

## 2. What Is FoundationDB

FoundationDB (FDB) is a **distributed, ordered key-value store with ACID
transactions**, originally developed by FoundationDB Inc., open-sourced in 2018,
now maintained by Apple. Docs: <https://apple.github.io/foundationdb/>.

Core facts, verified from primary sources:

- **Data model:** single ordered byte-key/byte-value space. No query language.
  All richer models (indexes, documents, tables, queues, graphs) are built as
  _layers_ on top, using ACID transactions to keep multiple keys consistent.
- **Transactions:** fully ACID, serializable (strongest isolation), multi-key
  across the cluster; durable before commit returns. Read-your-writes inside a
  transaction. Interactive (client can do many reads/writes per txn).
- **MVCC:** optimistic concurrency; conflicts detected at commit; no locks, no
  deadlocks. Retry loops are the normal pattern (integrated into the Go
  binding's `Transact` helper).
- **Ordered keys:** lexicographic scan of any range is efficient — one
  range-read retrieves all keys in a prefix. This is the basis of every recipe
  (index, queue, multimap, time series).
- **Atomic operations:** `Add`, `Min`, `Max`, `And`, `Or`, `Xor`, `BitAnd`,
  etc., applied without read-modify-write round-trips, **inside transactions**.
- **Watches:** push notifications when a key changes (after commit). No polling.
  Bounded per cluster (`max_watches` option, default typically thousands).
- **Scalability:** linear scale-out, shared-nothing, automatic partitioning and
  load balancing. Datacenter-aware replication, multi-DC mode with automatic
  failover. Tenants (experimental) for transaction-domain isolation.
- **Storage engines:** B-tree (SQLite-derived), in-memory, and Redwood
  (b-tree variant, the default since 7.0).
- **Throughput/latency (docs):** ~8.2M ops/sec on a 384-process commodity
  cluster (90/10 read/write); single-core ~55K reads/sec and ~20K writes/sec
  (SSD engine); commit latency 1.5–2.5 ms, reads 0.1–1 ms at <75% load.
- **Known limitations (hard):** single txn ≤ 10,000,000 bytes affected data;
  key ≤ 10,000 bytes; value ≤ 100,000 bytes; txn lifetime ≤ 5 seconds;
  no user-level access control (network boundary must be protected);
  large-offset key selectors are O(offset).
- **Bindings:** official C, Go, Java, Python, Ruby. The **Go binding is
  CGo-based** — it links `libfdb_c` (see §12).

---

## 3. How the Metaengine Engine Contract Works

An engine is anything implementing `metaengine.Engine`:

```go
type Engine interface {
    Profile() EngineProfile  // supported ADTs + complexity
    Close() error
}
```

Plus **per-ADT capability interfaces** (ISP — implement what you support):

| Capability                                                        | Used by                                       |
| ----------------------------------------------------------------- | --------------------------------------------- |
| `MapBackend`                                                      | MapGet / MapSet / MapDelete                   |
| `MapUpdater`                                                      | atomic read-modify-write                      |
| `PushdownScan`                                                    | SQL-level filter/sort/limit                   |
| `ScanBackend`                                                     | Go-side filter/sort fallback                  |
| `StreamingScan`                                                   | iteration without full materialization        |
| `SetBackend`                                                      | membership                                    |
| `CounterBackend`                                                  | increments + read                             |
| `MultimapBackend`                                                 | key → many values                             |
| `LogBackend`                                                      | append-only ordered log                       |
| `StreamLogBackend`                                                | stream-keyed append-only log + global journal |
| `LayoutPlanner`                                                   | extracted-column tables + secondary indexes   |
| `RawValueReader`/`RawScanReader`                                  | raw JSON bytes fast paths                     |
| `GraphBackend` (deprecated)                                       | graph edges/traversal (graphadapter now)      |
| `HealthChecker`, `Calibratable`, `Transactional`, `WatcherSource` | operational extras                            |

`EngineProfile` carries calibrated **nanoseconds-per-operation** costs and a
`Supports: map[ADT]Complexity` table, plus `Persistence`, `Replication`,
`ReplicationLag`, `NetworkRTT`, `DegradedADTs`, and per-read-pattern `ReadCosts`
(point lookup vs filtered scan vs aggregate vs full scan). **The planner uses
these numbers to pick the cheapest engine per query, and emits WARN/DEGRADED
diagnostics when the only candidate is a poor fit.**

The precedent that matters: **`pgengine`** — a remote-server engine with a
`New(dsn)` constructor, `NetworkRTT`, calibrated per-pattern read costs, and a
durability claim (`PersistencePersistent`, "remote server — always survives").
A FoundationDB engine would follow this exact template.

---

## 4. The Mapping: FDB Capability → Metaengine ADT

### 4.1 Map — ✅ Native

FDB's core is a KV store; a metaengine Map is just `collection\0key → value`.

- `MapSet`: `tr.Set(mapKey)`, `MapGet`: `tr.Get`, `MapDelete`: `tr.Clear`.
- **Atomic read-modify-write (`MapUpdater`)**: FDB _transactions_ make
  `MapUpdate` safe across processes — read + write in one serializable
  transaction. This beats every local engine (which rely on single-process
  mutexes) and restores the semantics the SQLite engine gets from `BEGIN`.
- **RawValueReader**: FDB returns raw bytes natively; no JSON double-decode.

### 4.2 Set — ✅ Native

Presence-encoded (`membership key = ""`). `SetAdd` = `tr.Set`, `SetContains` =
`tr.Get != nil`. The FDB "simple indexes" recipe literally uses this.

### 4.3 Counter — ✅ Best-in-class

`CounterBackend.CounterIncrement` maps to `tr.Add(key, int64LE)` — one atomic
mutation, no read-modify-write, no contention, exactness under concurrency.
`CounterGet` is a range scan of `c\0<col>\0*`. This is the strongest ADT fit:
FDB is _the_ canonical "atomic counter at scale" system.

### 4.4 SortedMap — ✅ Native via the simple-index pattern

A `SortOnField`/`FilterOnField` secondary index is exactly the documented
simple-index recipe (doc: _Simple Indexes_):

```
(main, col, key)  → value
(indx, col, fval, key) → ""
```

A range read on the index subspace returns matching keys in sorted order —
**one range read, not a full scan**. The index is kept consistent with the
data _in the same ACID transaction_ — no index-drift bug class at all (which is
the entire point of FDB, per its layer manifesto). **This is strictly stronger
than the Pebble layout planner**, which must implement the secondary index as a
separate write path with deletion/GC subtleties.

### 4.5 Multimap — ✅ Native

`mm\0<col>\0<key>\0<seq:%020d> → value`. Appends are `tr.Set` with a
client-maintained sequence per (collection, key), monotonic within a
transaction using `GetVersionstamp`/`SetVersionstampedKey` if cross-process
ordering matters. `MultiGet` = one prefix range read.

### 4.6 Log — ✅ Native

`l\0<col>\0<seq:%020d>`. `LogAppend` = bump seq (versionstamp or a
transactional counter key), `Set`; `LogTail` = `GetRange` on the last N keys
(reverse range). Truncation (if ever needed) is `ClearRange`, which FDB
documents as fast.

### 4.7 StreamLog — ⚠️ Per-stream yes, global journal no

Per-stream `StreamAppend`/`StreamRead`/`StreamVersion` map to the same
sequence-keyed pattern under `sl\0<col>\0<sid>\0<seq>`.

**The global journal is the problem.** Metaengine's `JournalReadAll` /
`JournalReadFrom` need a _single global monotonic sequence across all streams_
(used by projectionhost replay + catch-up subscribers). The FDB-idiomatic way
is `GetVersionstamp`/`SetVersionstampedKey` (the FDB _queues_ recipe) — but
versionstamps are only assigned at **commit time and only per-txn**, and FDB
caps each txn at 10 MB. If a projection batch must write N events AND the
journal, and events flow at high multi-node write rates, the journal of a large
collection can easily exceed 10 MB in a single replay batch → txn aborts.

Workarounds exist (fixed-size journal shards per time window, per-shard
versionstamped counters, or a _separate_ local journal as today with FDB only
for projections), but they are genuine design work. **The honest answer: FDB
should serve the projection space (Maps/Sets/Counters/SortedMaps) and NOT the
journal space** — the source-of-truth event log stays in a local engine
(SQLite/Pebble), exactly like the current setups.

### 4.8 Graph — ❌ Not a fit

Graph storage would be adjacency lists as prefix scans; traversal is
`O(degree^depth)` and each hop is a separate transaction/range read. Dgraph
(the dedicated graph engine) already beats this by orders of magnitude, and
metaengine already treats Graph as degraded on non-graph engines. FDB should
declare `ADTGraph` degraded/unsupported and the planner routes away (the
`degradedADTRule` exists for exactly this).

### 4.9 Vector / Search / Spatial — ❌ Not a fit (with a naming caveat)

- **FDB "Vector"** is _not_ ANN vector search. The docs' _Vector_ recipe is a
  **growable array/vector data structure** (element per key, tuple index), with
  efficient append/scan/truncate and no similarity search. FDB has **no ANN
  index** in core. Metaengine's `ADTVector` means cosine-similarity K-NN, which
  FDB simply cannot do natively.
- **Full-text search**: no inverted-index core — substring scans only. Dgraph
  and (partially) pgengine already beat this.
- **Spatial**: no spatial index core — the docs' _Spatial Indexing_ recipe
  builds geohash-range keys as a layer, which is doable but a lot of sharp
  work for an engine whose niche is elsewhere.

So FDB should mark `ADTVector/Search/Spatial` degraded and rely on the
existing `DegradedADTs` planner machinery. **Note:** the docs' "vector recipe"
is a good _Log_-like building block (the growable-array pattern is a Log), so
the naming collision is a source of confusion worth flagging, not a feature.

### 4.10 Watchers — ✅ Strategic (the sleeper feature)

FDB **watches** are transactional push notifications: register on a key,
commit, and the client is notified when the key next changes. The metaengine
has an in-process `Watcher`/`subscriberHub`, but no backend interface exists
for pushing change events from _remote_ engines — watchers today are local
publish/subscribe inside a single process.

An FDB engine could implement a new `WatcherSource`-style interface (or be
wired into the existing watcher layer) to deliver **cross-process,
cross-machine change notifications** with zero polling. Nothing in the current
roster (Memory/SQLite/Pebble/DuckDB/PG/Dgraph/Iroh) can do this. This is the
most compelling _new capability_ FDB brings to metaengine, and it is cheap to
implement on top of a backend that already has raw `Watch`.

---

## 5. Fit Analysis: Where FDB Wins

1. **The only honest multi-node engine.** Every engine today is single-node
   by design; Iroh replicates via CRDTs with explicit CALM constraints and
   _fire-and-forget_ writes. FDB gives synchronous, serializable,
   fully-ACID multi-process shared state. If metaengine is ever used by N
   app replicas sharing one projection space, every alternative today is
   "run PG" or "run Dgraph" — FDB becomes a first-class option.
2. **Transactional exactness.** `MapUpdate` (read-modify-write) is atomic and
   serializable across processes — today only SQLite (single-node, via
   `BEGIN`) matches, and only within one process.
3. **Atomic counters at scale.** `tr.Add` is the canonical implementation:
   contention-free, exact under concurrent writers, no lost updates.
4. **Consistent secondary indexes.** The simple-index pattern updates data +
   index in one ACID txn. No drift, no GC of stale index entries, no
   read-your-writes surprises. This directly addresses the `LayoutPlanner`
   gap in Pebble (which must hand-maintain index entries with explicit
   delete-on-update logic).
5. **Push-based change notification** (watches) — see §4.10.
6. **Operational resilience.** Automatic sharding/rebalancing, no manual
   partitioning, datacenter failover, elastic scale-out. For an operator
   choosing "deploy metaengine across a fleet," FDB removes the
   shard-management burden that a bare PG cluster or Dgraph cluster imposes.
7. **Raw-byte fast paths.** `RawValueReader`/`RawScanReader` come for free
   (bytes in, JSON decode once).

---

## 6. Fit Analysis: Where FDB Loses

1. **No SQL / no pushdown.** `PushdownScan` (filter/sort pushed into
   WHERE/ORDER BY) is impossible — there is no query language. All filtered
   scans must be Go-side (`ScanBackend` fallback), i.e. O(N) decode + filter
   in the client. PG/SQLite/DuckDB beat FDB here.
2. **10 MB transaction cap.** Any batch write that touches >10 MB of keys
   aborts. Projection replays (the projectionhost path — often replaying big
   event batches) must be chunked manually. This is the single most
   constraining limit for the metaengine _write_ path.
3. **Value size cap (100 KB).** Metaengine values are JSON-encoded projection
   rows; a large `FindUserResult` (nested slices, blobs) can exceed 100 KB.
   The engine must either reject, chunk (the FDB "managing large values"
   recipe), or route big projections elsewhere. Pebble/SQLite have no such
   cap.
4. **5-second transaction lifetime.** Long-running interactive transactions
   (which the metaengine `MapUpdater` callback pattern encourages!) are
   rejected. The retry loop + 5s limit means fold callbacks must be fast, and
   any slow user code inside `MapUpdate` risks `transaction_too_old`.
5. **No analytical engine.** Scans/aggregates through FDB are range reads to
   the client; no columnar/vectorized aggregation, no GROUP BY pushdown.
   DuckDB and PG remain the aggregate engines. `ReadAggregate` on FDB is
   effectively O(N) client-side.
6. **Heavyweight deployment.** FDB is a _server_ (multiple processes:
   coordinators, logs, resolvers, storage servers), configured via NixOS
   module or manually. It is not an embedded store like SQLite/Pebble/bbolt.
   There is no "FDB in a file." A metaengine consumer's `go get` experience
   cannot spin one up — this is a fundamental mismatch with the "embedded
   first" positioning of the current engine roster.
7. **CGo requirement.** The official Go binding links `libfdb_c` (C client
   library). This adds a CGO constraint like duckdbengine — manageable only
   via a dedicated module (see §11).
8. **No access control.** Anyone who can reach the cluster can read/write
   everything (docs: "not a security boundary"). In a multi-tenant deployment
   this forces network-level isolation or experimental tenants.
9. **10,000-byte key cap.** Keys are small (metaengine keys are
   collection + key strings; fine in practice, but a pathological 8 KB ID
   would fail).
10. **Degraded OLAP and query-language absence** make it a poor _sole_
    engine — the exact scenario the metaengine planner warns about
    (`DEGRADED` diagnostics). It is only useful _alongside_ local engines.

---

## 7. Cost Model: What a FoundationDB EngineProfile Would Look Like

Following the `pgengine` remote-server template (same-datacenter assumptions):

```go
profile := metaengine.EngineProfile{
    Name:        "foundationdb",
    Persistence: metaengine.PersistencePersistent, // remote cluster survives process exit
    Replication: metaengine.ReplicationSingleLeader,
    ReplicationLag: 0,     // synchronous durability at commit
    NetworkRTT:   1 * time.Millisecond, // same-DC conservative; FDB docs: commit 1.5-2.5ms, read 0.1-1ms
    NsPerOp:     10_000,   // budget placeholder; must be calibrated (see below)
    NsPerRead:    2_000,
    NsPerWrite:  12_000,   // durable commit to log + replication
    Supports: map[metaengine.ADT]metaengine.Complexity{
        ADTMap:       ComplexityO1,    // point read, practical ~O(1)
        ADTSet:       ComplexityO1,
        ADTCounter:   ComplexityO1,    // atomic add + point read
        ADTSortedMap: ComplexityOLogN, // index range read
        ADTMultimap:  ComplexityOLogN,
        ADTLog:       ComplexityOLogN,
        ADTStreamLog: ComplexityOLogN, // per-stream
        // Graph/Search/Spatial/Vector deliberately ABSENT from Supports
    },
    DegradedADTs: map[metaengine.ADT]bool{
        ADTGraph:   true, // adjacency scans
        ADTVector:  true, // no ANN core
        ADTSearch:  true, // no FTS core
        ADTSpatial: true, // no spatial core
    },
    ReadCosts: metaengine.ReadCosts{
        NsPerPointLookup: 2_000, // FDB read 0.1-1ms per doc; network + client decode
        NsPerFilteredScan: 1_500, // per-row: range read to client + Go filter (no pushdown!)
        NsPerScan:         1_200, // per-row: range read to client, JSON decode
        // NsPerAggregate: NOT set → falls back to NsPerScan (client-side aggregation)
    },
}
```

Notes:

- **No `NsPerAggregate`** — deliberately. FDB has no aggregation pushdown, so
  the planner must treat aggregates as client-side scans. This is the honest
  multi-engine signal: "read this query from DuckDB/PG, not FDB."
- **No `FilterOnField` pushdown** — FDB implements `ScanBackend` only, not
  `PushdownScan`. Planner inference handles this: queries with declarative
  filters route to SQL engines.
- **`NetworkRTT`** — the planner already adds RTT as a fixed latency term, so
  a same-DC estimate (~1 ms) keeps FDB competitive with PG for point lookups
  and loses decisively for tiny per-row scans (per-row RTT is the killer —
  exactly the number that makes PG beat FDB for scan-heavy queries).
- **Calibration** — real numbers must come from a
  `calibration_bench_test.go` (`BenchmarkCalibration_FDB_*`) using
  `fdb.MustOpenDefault()` against a CI cluster, exactly as
  `pgengine/calibration_bench_test.go` does for PG via testcontainers.

### What the planner would decide with FDB in the mix

| Query shape                                         | Winner today              | Winner with FDB                                            |
| --------------------------------------------------- | ------------------------- | ---------------------------------------------------------- |
| `FindUser` point lookup, low volume                 | Memory (~ns)              | Memory (RTT kills FDB)                                     |
| `FindUser` point lookup, high volume, multi-process | PG                        | **FDB** (same RTT, better point-read scaling + redundancy) |
| Counter increments, multi-writer                    | PG                        | **FDB** (atomic `Add`, no lost updates, linear scale-out)  |
| Filtered scan                                       | PG/SQLite/DuckDB pushdown | PG/SQLite/DuckDB (FDB = O(N) client-side)                  |
| Aggregate                                           | DuckDB                    | DuckDB (FDB has no vectorized sum)                         |
| Any high-volume replay batch                        | Pebble/SQLite             | **Pebble/SQLite** (10 MB txn cap)                          |

The planner already has every rule needed to express this: `ReadCosts` for
per-pattern costs, `NetworkRTT` for remote engines, `DegradedADTs` for
fallbacks, `durabilityRule` for volatile-vs-persistent routing.

---

## 8. The Replication Story: FDB vs the Current Replication Model

Metaengine models replication after DDIA Ch5 (`Replication` enum:
None/SingleLeader/MultiLeader/Leaderless, `ReplicationLag`,
`NetworkRTT`, `Persistence`). Today:

- **Every engine** is `ReplicationNone` — metaengine replication exists only in
  the planner's _declarative_ model; the lone consumer is `irohengine`, which
  wraps a local engine with CRDT-based async replication (CALM-safe writes only,
  `MapUpdate` stays local, eventual consistency, fire-and-forget).

FDB changes this meaningfully:

| Property                 | irohengine (today)                 | FoundationDB (proposed)                        |
| ------------------------ | ---------------------------------- | ---------------------------------------------- |
| Coordination             | Peer-to-peer CRDT merge            | Centralized cluster, serializable commits      |
| Write visibility         | Eventually consistent (async)      | **Synchronous before commit returns**          |
| Failure safety           | CRDT merge on reconnect            | Redundant tx log + storage, auto-recovery      |
| `MapUpdate` across nodes | ❌ stays local                     | ✅ atomic, serializable                        |
| Conflict semantics       | LWW / OR-Set (add-wins)            | Strict serializability (txn abort on conflict) |
| Latency uniformity       | Async, can lag arbitrarily         | Bounded (commit 1.5-2.5 ms)                    |
| Topology                 | Ad hoc node graph                  | Managed cluster, DC-aware, elastic             |
| What it replaces         | "eventually-consistent multi-node" | **"shared source of truth across nodes"**      |

**FDB is the only manifestly-replicated engine that can honestly declare
`Replication: SingleLeader`** in its profile (with `ReplicationLag: 0`,
because durability is synchronous at commit). This is a _new slot_ in the
planner's model — the first engine that actually exercises the DDIA
dimensions the planner was built to reason about. The planner's `explain`/
`serializable` outputs already print replication, lag, and RTT (e.g.
`explain.go:141`, `serializable.go:94-95`), so an FDB engine would light up a
diagnostic path that has been dormant since the replication work.

**Caveats:** FDB is _single-leader_ for its commit protocol (the write path is
a master-elected transaction system), so MultiLeader (multi-DC) is available
but has its own semantics (datacenter affinity + still single transaction
system per DC group; actual cross-DC writes go through the transactional
system). For metaengine purposes: declare SingleLeader with lag 0 and treat
multi-DC as an ops concern, not a planner concern — this matches how FDB
itself documents it (multi-DC failover, not multi-leader writes).

---

## 9. The StreamLog / Journal Problem

This is the deepest design tension in the whole fit, worth its own section.

**What metaengine needs:** `StreamLogBackend` exposes `StreamAppend`,
`StreamRead`, `StreamVersion`, `JournalReadAll`, `JournalReadFrom`. The
projectionhost and CatchUpSubscriber rely on the global journal for
position-based replay (`JournalReadFrom(afterSeq)`). The journal is a _global,
monotonic, cross-stream sequence_.

**How FDB would provide it:**

- Per-stream sequence: fine (sequence keys + versionstamp).
- **Global sequence:** FDB gives `GetVersionstamp()` — a unique, monotonically
  increasing version assigned at commit, usable in keys via
  `SetVersionstampedKey`. This is the canonical FDB queue/log technique.

**The conflict:** FDB's 10 MB per-txn cap means a journal whose keyspace is
written by many concurrent processes (each appending its own rows) must
organize itself so that _no single transaction writes more than 10 MB of
journal keys_. That's achievable with sharded time-bucket journals
(`journal/2026-08/dd/...` with per-bucket commit counters) — the standard FDB
pattern — but it is real, subtle engineering:

- Reading "all events since position X" becomes a multi-range read across
  buckets + an ordering merge.
- Truncation/compaction of old buckets must not race with in-flight
  `JournalReadFrom` cursors.
- Event payloads are JSON-encoded; a 10 MB bucket fills fast at high rates.

**Recommendation (strong):** Do NOT put the source-of-truth event journal on
FDB. Keep the event log on the local engine (Pebble/SQLite — as every current
setup does, with FDB only as a _projection_ engine). For the _projection_
space, `StreamLog`-typed queries (e.g. a `recentTasks` Log) are fine on FDB at
per-stream scale. The global-journal path simply stays local. This preserves
the current architecture's clear separation ("event log = replay source,
engines = projections") and avoids the one place where FDB's guarantees
actively fight the workload.

---

## 10. Alternative Roles: Not Just an Engine

Beyond `Engine`, FDB could plug in as several _other_ metaengine-adjacent
components. Worth listing so the design isn't blindered:

1. **FDB as the operator-facing "system of record" for plans/snapshots.**
   The planner produces `PlanResult`, `Explain`, Doctor reports; `serializable`
   already has JSON forms. A tiny FDB subspace could store the _active plan_,
   _calibration results_, and _schema/collection registry_ — shared across app
   replicas. This is a "control plane" use, not a data-plane use, and plays to
   FDB's strength (small multi-key transactional state shared across
   processes). No CGo needed for consumers (the registry reads happen in a
   separate ops binary).
2. **FDB as a durable outbox/queue for cross-instance event fan-out.**
   The FDB _queues_ recipe is a mature pattern (locked dequeue via
   transactions). Metaengine's bus/streaming layer
   (`sse.go`, `subscribers.go`) currently fan out in-process; FDB could serve
   as the durable multi-instance broadcast channel — watch for changes, push
   to local subscribers. Again, this is the watch/sse story from §4.10.
3. **FDB as the multi-instance `projectionhost` coordinator.**
   Checkpoint state (`projectionhosthost`), DLQ state, and per-projection
   progress are exactly the kind of small shared transactional state FDB was
   built for. A production projection fleet running N replicas could store
   checkpoints on FDB instead of racing on a local DB file.

Each of these is _less work_ than a full engine backend and delivers the same
"shared durable state across processes" win with none of the ADT-cap
troubles. **If the goal is "FDB in the metaengine ecosystem" with minimal
risk, the control-plane/coordinator role is the better first slice.** If the
goal is "FDB as a projection engine," §4-§7 is the design.

---

## 11. Module and Packaging Strategy

If built, it must be a **separate module**, following the established pattern:

- `metaengine/fdbengine/` (own `go.mod`, like `pgengine/`, `dgraphengine/`,
  `duckdbengine/`). Deps: `github.com/apple/foundationdb/bindings/go` (CGo).
- **CGo isolation** — the binding requires `libfdb_c` (see §14). This is the
  same isolation story as `duckdbengine`, which ships with
  `//go:build cgo`-gated files and a `doc.go` warning. Consumers who don't
  import the module never need CGo. FDB's Go module has **no tagged releases**
  (pseudo-versions only, e.g. `v0.0.0-2026...`), and the binding must match
  the cluster major version (e.g. all 7.3 bindings work against 7.3 clusters)
  — both worth documenting in the module README.
- **Minimal-engine philosophy** — implement only:
  `MapBackend`, `MapUpdater`, `SetBackend`, `CounterBackend`,
  `MultimapBackend`, `LogBackend` (+ per-stream `StreamLogBackend` subset),
  `ScanBackend` (Go-side), `RawValueReader`/`RawScanReader`, `HealthChecker`.
  Explicitly do NOT implement `PushdownScan`, `LayoutPlanner`, `StreamingScan`
  (initially), `Graph`/`Search`/`Spatial`/`Vector`.
- **Testing:** `fdb.MustOpenDefault()` against a local `fdbserver`
  (single-node cluster via a Nix flake devShell or a CI ephemeral service —
  the repo already uses "ephemeral PG/Redis/NATS/Dgraph via nixpkgs services"
  for integration tests, and nixpkgs provides `foundationdb` 7.3.68 with a
  NixOS module). Tests marked with the external-service build tag/intent like
  `pgengine`'s testcontainers tests.
- **Genesis files:** add the module path to `go.work`, `testModules` in
  `flake.nix`, `cmd/api-stability/main.go` `modules` slice, then regenerate
  the api-stability golden (`cd cmd/api-stability && GOWORK=off go run main.go
  -update`). Per AGENTS.md "Add a New Module" procedure.
- **Docs:** module README + a row in `.agents/skills/go-cqrs-lite/references/modules.md`.

---

## 12. Effort and Risk Assessment

| Item                                                               | Effort | Risk     | Notes                                                                                                                             |
| ------------------------------------------------------------------ | ------ | -------- | --------------------------------------------------------------------------------------------------------------------------------- |
| Core backend (Map/Set/Counter/Multimap/Log + transaction handling) | M      | Low      | Straightforward key shapes; the _only_ tricky part is the retry loop for `commit_unknown_result`/conflicts                        |
| `MapUpdater` with retry loop + 5s txn cap                          | M      | Medium   | User fold callbacks inside a txn can exceed 5s; need "short txn" discipline + fallback to a read-modify-write retry pattern       |
| Secondary index (SortedMap)                                        | M      | Medium   | Key shapes + read-through-index; FDB does the consistency for us                                                                  |
| Scans (Go-side) + cursors + pagination                             | M      | Medium   | Range iterations + cursor positions map to FDB key selectors; watch the O(offset) selector pitfall — prefer limit-based continues |
| StreamLog subset + versionstamp journal                            | L      | High     | Global journal is the risky part (10 MB cap, §9)                                                                                  |
| Watchers integration                                               | S-M    | Low      | Trivial at backend level (raw `Watch`), needs a new interface or wiring                                                           |
| Calibration benchmarks + profile                                   | M      | Low      | Pattern exists (pgengine testcontainers)                                                                                          |
| Nix devShell + CI cluster provisioning                             | M      | Medium   | nixpkgs has `foundationdb` 7.3.68 + NixOS module; ephemeral service conventions exist                                             |
| CGo + module isolation + golden regen                              | S      | Low      | Established procedure                                                                                                             |
| Operations burden on consumers                                     | —      | **High** | This is the real adoption barrier: consumers must run an fdbserver cluster                                                        |

**Overall:** A _minimal viable FDB projection engine_ (Map/Set/Counter/
Multimap/Log/SortedMap, with 10 MB-aware batching) is bounded work (~2-4
weeks incl. calibration + CI). The global-journal path is the one genuinely
hard design decision and should be explicitly out of scope v1.

---

## 13. Recommendation

**Build `metaengine/fdbengine` as a projection engine — with a narrow,
honest scope.** Ranked by value-per-effort:

1. **P0 — Projection engine (v1 scope):** Map, Set, Counter, Multimap, Log,
   SortedMap secondary indexes, Go-side scans, raw-byte fast paths,
   `MapUpdater` with transactional retries. Profile uses
   `ReplicationSingleLeader`, lag 0, `NetworkRTT ~1ms`, per-pattern
   `ReadCosts`, `DegradedADTs` for Graph/Search/Spatial/Vector. Calibrate
   against a real cluster. Deliver the "multi-instance, crash-proof,
   serializable shared projection store" story.
2. **P1 — Watchers:** expose FDB watches as cross-process change
   notifications (new `WatcherSource`-style interface or subscriber wiring).
   This is the single most novel capability FDB grants metaengine and the
   cheapest to ship after the backend.
3. **P1 — Multi-instance control plane:** plan/checkpoint/DLQ/registry
   storage on FDB via the existing local engines' serialization, for
   deployment-time configuration that must be shared across replicas.
4. **P2 — Explicitly NOT v1:**
   - ❌ Global journal on FDB (10 MB cap; keep the event log local).
   - ❌ Graph/Search/Spatial/Vector ADTs on FDB (declare degraded, let the
     planner route away — the machinery already exists).
   - ❌ SQL-style pushdown (impossible; FDB is a KV core by design).

**Why not "FDB as the whole metaengine"?** The vision "give metaengine
SQLite-only and it handles everything" does not extend to FDB-only. FDB's
10 MB txn cap, 100 KB value cap, 5 s txn lifetime, lack of SQL/aggregation
pushdown, and heavy server deployment combine to make it an unsuitable
_sole_ engine for event-sourced projection workloads with large values.
Metaengine's "graceful degradation, never failure" invariant still holds —
FDB would emit DEGRADED diagnostics for the ADTs it can't serve natively —
but the right architectural shape is **FDB alongside local engines**, exactly
as PG sits alongside them today.

If the goal is a _multi-node engine_ rather than a _specific database_, the
brief alternative worth a half-day spike before committing: compare
`fdbengine` P0 against "pgengine with `MultiLeader`/HA tooling (Citus,
Patroni, read replicas)" to confirm FDB's point-lookup/counter/watch win is
worth its operational tax in the target deployment. The report's assumption:
for shared-binary deployment (multiple app replicas sharing one projection
space, with strong consistency and atomic counters), FDB wins; for
single-instance SQL-ish workloads, PG stays ahead.

---

## 14. Verified External Claims

All claims were verified against primary sources on 2026-08-10.

| Claim                                                                                                                                             | Status      | Source                                                                                                                                           |
| ------------------------------------------------------------------------------------------------------------------------------------------------- | ----------- | ------------------------------------------------------------------------------------------------------------------------------------------------ |
| FDB is an ordered KV store with ACID multi-key transactions, serializable                                                                         | ✅ Verified | [features.html](https://apple.github.io/foundationdb/features.html), [architecture.html](https://apple.github.io/foundationdb/architecture.html) |
| Commit latency 1.5–2.5 ms, reads 0.1–1 ms (<75% load)                                                                                             | ✅ Verified | [performance.html](https://apple.github.io/foundationdb/performance.html)                                                                        |
| 8.2M ops/s on 384-process commodity cluster (90/10 R/W)                                                                                           | ✅ Verified | [performance.html](https://apple.github.io/foundationdb/performance.html)                                                                        |
| Single-process per-core ~55K reads/s / ~20K writes/s (SSD engine)                                                                                 | ✅ Verified | [performance.html](https://apple.github.io/foundationdb/performance.html)                                                                        |
| Txn cap: 10,000,000 bytes affected data; key ≤ 10,000 B; value ≤ 100,000 B; txn ≤ 5 s                                                             | ✅ Verified | [known-limitations.html](https://apple.github.io/foundationdb/known-limitations.html)                                                            |
| No SQL, no query language in core; layers provide data models                                                                                     | ✅ Verified | [anti-features.html](https://apple.github.io/foundationdb/anti-features.html)                                                                    |
| Atomic ops in core: Add/Min/Max/And/Or/Xor/BitXor etc.                                                                                            | ✅ Verified | [features.html](https://apple.github.io/foundationdb/features.html), pkg.go.dev Go API (`Transaction.Add`)                                       |
| Watches: transactional push change notifications (Go API `Watch`)                                                                                 | ✅ Verified | [features.html](https://apple.github.io/foundationdb/features.html), pkg.go.dev Go API                                                           |
| Simple-index recipe (index stored as keys, data+index updated in same txn)                                                                        | ✅ Verified | [simple-indexes.html](https://apple.github.io/foundationdb/simple-indexes.html)                                                                  |
| Vector doc = growable array recipe, NOT ANN vector search                                                                                         | ✅ Verified | [vector.html](https://apple.github.io/foundationdb/vector.html)                                                                                  |
| No user-level access control ("not a security boundary")                                                                                          | ✅ Verified | [known-limitations.html](https://apple.github.io/foundationdb/known-limitations.html)                                                            |
| Tenants currently experimental                                                                                                                    | ✅ Verified | [tenants.html](https://apple.github.io/foundationdb/tenants.html)                                                                                |
| Automatic idempotency experimental                                                                                                                | ✅ Verified | [automatic-idempotency.html](https://apple.github.io/foundationdb/automatic-idempotency.html)                                                    |
| Official Go binding exists, `github.com/apple/foundationdb/bindings/go`, importable, Apache-2.0, no tagged stable releases (pseudo-versions only) | ✅ Verified | pkg.go.dev (version v0.0.0-2026080818...), repo README                                                                                           |
| Go binding requires CGo + FDB client library (libfdb_c)                                                                                           | ✅ Verified | upstream `bindings/go/README.md`: "Go 1.22+ with CGO enabled; FoundationDB client package"                                                       |
| Binding supports API versions 200-800 (7.3 cluster ↔ 7.3 bindings)                                                                                | ✅ Verified | upstream `bindings/go/README.md`                                                                                                                 |
| FDB server available in nixpkgs (foundationdb 7.3.68) + NixOS module                                                                              | ✅ Verified | `nix search nixpkgs foundationdb` (local), `/nix/store` module doc                                                                               |
| Multi-DC failover via three-DC replication, elastic scale-out                                                                                     | ✅ Verified | [features.html](https://apple.github.io/foundationdb/features.html)                                                                              |
| Storage engines: B-tree (SQLite-derived), memory, Redwood                                                                                         | ✅ Verified | [architecture.html](https://apple.github.io/foundationdb/architecture.html)                                                                      |
| Key selectors with large offsets resolve in O(offset)                                                                                             | ✅ Verified | [known-limitations.html](https://apple.github.io/foundationdb/known-limitations.html)                                                            |

_Unverified / out of scope:_ exact FDB Go binding ergonomics under Go 1.26 +
`goexperiment.jsonv2`; live calibration numbers (require a running cluster);
behavior of the future multi-version client in this workload.

---

_Sources:_ [apple.github.io/foundationdb](https://apple.github.io/foundationdb/) (v7.3.79 docs),
[pkg.go.dev fdb package](https://pkg.go.dev/github.com/apple/foundationdb/bindings/go/src/fdb),
[upstream bindings/go README](https://raw.githubusercontent.com/apple/foundationdb/main/bindings/go/README.md),
[metaengine README](../../metaengine/README.md) and engine contract
(`metaengine/engine.go`, `metaengine/types.go`, `metaengine/pgengine/engine.go`,
`metaengine/pebbleengine/engine.go`, `metaengine/irohengine/README.md`,
`metaengine/duckdbengine/doc.go`).
