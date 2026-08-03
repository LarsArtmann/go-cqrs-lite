# Metaengine: Eventual Consistency Model and Iroh Integration

> **Status:** Replication model shipped (Phase 2 complete — `EngineProfile` fields, cost estimator, planner rule, `CollectionInfo` exposure, `ExplainPlan`/`Doctor` output). Iroh integration (Phase 4) not yet started.
> **Date:** 2026-08-02 (updated 2026-08-03: replaced Visibility model with DDIA-canonical Replication + NetworkRTT + ReplicationLag)
> **Related:** [meta-engine-design.md](meta-engine-design.md), [meta-engine-assumptions-and-query-planning.md](meta-engine-assumptions-and-query-planning.md), [ADR-0084](../adr/0084-metaengine-layered-architecture.md)

---

## TL;DR

1. CQRS read models are **already eventually consistent** — the projection host has measurable lag (`LagDuration()`, `LagPerProjection()`). The only strong operation is the event store's optimistic-concurrency append.
2. Modeling "everything is eventual" lets the planner **drop consistency as a dimension** and replace it with three DDIA-canonical engine properties: **Replication** (Ch5: how data propagates), **ReplicationLag** (Ch5: how stale), and **NetworkRTT** (Ch1: how far away). These are **engine properties only** — queries declare what to compute, not where data lives.
3. This reframing is the natural entry point for **[Iroh](https://github.com/n0-computer/iroh)** integration — Iroh's `iroh-docs` CRDT key-value store becomes a `ReplicationLeaderless` engine whose monotonic fold operations converge without coordination.
4. The change is **backward compatible**: all existing engines default to `ReplicationNone` (zero value), zero lag, zero RTT. No existing behavior changes.

---

## Part 1: The Core Insight — Read Models Are Already Eventual

### The false dichotomy

A naive model of the metaengine would label SQLite/Pebble/DuckDB/Memory as "strong consistency" and any distributed engine as "eventual consistency." This is wrong.

Every metaengine collection is populated by a projection — either via `projectionadapter` or direct `Apply`/`ApplyBatch` calls. The projection processes events asynchronously. There is always a gap between an event being appended to the store and the projection folding it into the collection. The `projectionhost.Host` exposes this directly:

```go
for _, s := range host.Status() {
    fmt.Printf("%s: lag=%s\n", s.Name, s.Lag)
}
```

This lag is **not zero**. It ranges from microseconds (in-process, hot projection) to seconds (under load, after restart, during replay). That gap is the definition of eventual consistency.

### The only strong operation

The single strongly-consistent operation in the entire CQRS architecture is the event store append with optimistic concurrency control (`AppendBatch` checking expected version). Everything downstream — projections, read models, metaengine collections — is a **stale cache** of the event log.

### Implication for the planner

The planner should not model consistency as a binary dimension (`strong | eventual`). It should model what actually differs between engines:

- **Replication** — does this engine replicate data across nodes? (DDIA Ch5)
- **Replication lag** — how stale might reads be? (DDIA Ch5)
- **Network RTT** — how far away is the data? (DDIA Ch1)

---

## Part 2: The Replication Model

### Replacing consistency with three DDIA-canonical properties

The initial design proposed a `Visibility` dimension (`VisibilityLocal` / `VisibilityGlobal`). Through analysis, this was proven wrong — "visibility" conflates three orthogonal concerns into one name. The correct model uses three separate properties, each grounded in DDIA:

| Property         | DDIA concept           | What it captures                 | Type            | Scales with volume?     | Planner use                  |
| ---------------- | ---------------------- | -------------------------------- | --------------- | ----------------------- | ---------------------------- |
| `Replication`    | Ch5: Replication modes | How data propagates across nodes | Enum            | N/A                     | Diagnostics / future routing |
| `ReplicationLag` | Ch5: Replication lag   | How stale local data may be      | `time.Duration` | No                      | Diagnostics / freshness      |
| `NetworkRTT`     | Ch1: RTT               | How far a read must travel       | `time.Duration` | **No (additive fixed)** | Cost estimation              |

**Key principle:** Replication is an **engine property only**. Queries declare _what_ to compute (folds, ADTs, read patterns). Engines declare _how_ data is stored (layout, cost, replication). The query API (`QueryConfig`) has **zero** new fields.

### EngineProfile changes

```go
type Replication string

const (
    ReplicationNone         Replication = ""               // zero value = single-node
    ReplicationSingleLeader Replication = "single-leader"  // Postgres streaming
    ReplicationMultiLeader  Replication = "multi-leader"   // CockroachDB, Spanner
    ReplicationLeaderless   Replication = "leaderless"     // Iroh CRDT, Dynamo
)

type EngineProfile struct {
    // ...existing fields (Name, Supports, Layouts, NsPerOp, NsPerRead, NsPerWrite)...

    // Replication declares how this engine's data propagates across processes (DDIA Ch5).
    // ReplicationNone (zero value) means single-node. All current engines are ReplicationNone.
    Replication Replication

    // ReplicationLag is expected delay between a write on one node and it being
    // visible on another (DDIA Ch5). Zero for local/primary engines.
    ReplicationLag time.Duration

    // NetworkRTT is the round-trip time to reach this engine's data (DDIA Ch1).
    // Zero for in-process engines. Additive in cost estimation — does NOT scale with volume.
    NetworkRTT time.Duration
}
```

### Planner routing

The planner routes by cost, naturally preferring local engines. `NetworkRTT` is an additive fixed cost — it doesn't multiply with query volume:

```
total_latency = (ops × nsPerRead / 1e6) + NetworkRTT
```

A remote engine with 50ms RTT loses to a local engine with ~0ms RTT on every query, because RTT is constant regardless of whether the query touches 1 row or 10,000. This is why RTT must be separate from `NsPerRead` — if RTT were baked into the per-op cost, scan estimates would be wildly inflated (10,000 × 50ms = 500s instead of 30ms + 50ms).

`ReplicationLag` is NOT part of latency estimation. Staleness is a freshness property, not a performance cost. It surfaces in diagnostics when non-zero.

### Query declaration (unchanged)

```go
// Queries declare WHAT to compute, not WHERE data lives.
// No replication/visibility options on queries — that's an engine concern.
metaengine.Query[Input, Result]("orders", folds...)
```

If a consumer needs freshness guarantees, they check at read time — same pattern as `host.LagDuration()` today.

---

## Part 3: The CALM Theorem Connection

### Why monotonic folds are CRDT-safe

The [CALM theorem](https://link.springer.com/chapter/10.1007/978-3-642-04243-6_27) (Consistency As Logical Monotonicity) states: a distributed program converges without coordination if and only if it is monotonic — its output only grows, never retracts.

The metaengine ADTs are already monotonic by construction:

| ADT      | Fold operations            | Monotonic?              | CRDT equivalent             |
| -------- | -------------------------- | ----------------------- | --------------------------- |
| Map      | `FoldInsert`, `FoldUpdate` | Yes (LWW by timestamp)  | LWW-Map                     |
| Set      | `FoldSet`                  | Yes (add-only)          | OR-Set (G-Set if no remove) |
| Counter  | `FoldCount` (+/-)          | Yes (per-author)        | PN-Counter                  |
| Multimap | `FoldMultiInsert`          | Yes (add per key)       | OR-Set per key              |
| Log      | `FoldAppend`               | Yes (append per author) | Per-author append-only      |

### Non-monotonic operations

The non-monotonic operations are the exceptions, and they are already the operations that CRDTs handle specially:

| Operation                | Monotonic?  | CRDT behavior                                                        |
| ------------------------ | ----------- | -------------------------------------------------------------------- |
| `MapUpdate` (atomic RMW) | No          | Local only — not a CRDT operation; does not replicate                |
| `MapDelete`              | Conditional | Tombstone (safe, replicates) vs physical delete (unsafe, local only) |
| `FoldRemove` (Set)       | No          | Tombstone-based OR-Set (safe if tombstones are tracked)              |

**This means the fold operations are already CRDT-safe for 5 of 7 ADTs.** The all-eventual model is not a redesign — it is an acknowledgment of what the system already is.

### The philosophical payoff

The all-eventual model reveals what the architecture has been quietly saying:

> **The event log is the source of truth. Everything else is a cache.**

The planner's job is not to guarantee consistency — it is to pick the cheapest cache that has the data scope the caller needs. Local caches (Memory, SQLite, Pebble, DuckDB) are fast but process-local. Global caches (Iroh) are slower but cross-node. The caller decides what scope they need; the planner finds the cheapest option.

Since folds are monotonic (CRDT-safe), coordination-free convergence is guaranteed by the CALM theorem. The planner does not need to reason about consistency because monotonicity _proves_ eventual consistency. It only needs to reason about cost and visibility.

---

## Part 4: Iroh Integration

### What Iroh provides

[Iroh](https://github.com/n0-computer/iroh) is a peer-to-peer networking stack ("dial keys, not IPs") with NAT traversal, built on QUIC + TLS 1.3. The relevant protocol for the metaengine is `iroh-docs`:

- **Distributed CRDT key-value store** identified by a `NamespaceId`
- Entries keyed by `(namespace, author, key)` — each author signs its own entries
- **Range-based set reconciliation** for efficient sync (fully-in-sync peers exchange a single fingerprint)
- `iroh-gossip` carries live sync notifications
- Content values are BLAKE3 hashes into `iroh-blobs` (content-addressed, verified streaming)

### Backend interface mapping

`iroh-docs` maps onto 5 of the 10 metaengine ADT backends:

| Backend           | iroh-docs implementation                                         | CRDT type                               |
| ----------------- | ---------------------------------------------------------------- | --------------------------------------- |
| `MapBackend`      | `doc.Set(key, hash)` / `doc.Get(key)` then resolve blob          | LWW-Map (last-writer-wins by timestamp) |
| `SetBackend`      | `doc.Set(key, empty)` / `doc.Get(key) != nil`                    | OR-Set (observed-remove)                |
| `CounterBackend`  | Each author keeps own increment count; `Get` sums across authors | **PN-Counter** (conflict-free)          |
| `MultimapBackend` | `doc.Set(prefix+key+seq, value)` / prefix range scan             | OR-Set per key                          |
| `LogBackend`      | `doc.Set(author+seq, value)` / reverse range scan                | Per-author append-only                  |

What it **cannot** serve:

| Backend                                              | Why not                                                                           |
| ---------------------------------------------------- | --------------------------------------------------------------------------------- |
| `PushdownScan` / `ScanBackend`                       | No server-side filtering — all filtering is client-side after fetch. O(N) always. |
| `GraphBackend`                                       | No graph operations                                                               |
| `VectorBackend` / `SearchBackend` / `SpatialBackend` | No similarity/text/geo search                                                     |
| `LayoutPlanner` / `LayoutPlanApplier`                | No DDL, no indexes                                                                |
| `MapUpdater` (atomic RMW)                            | CRDT does not support atomic read-modify-write — the "read" may be stale          |

This is fine — Pebble also does not implement Vector/Search/Spatial. The planner routes by capability.

### The Counter ADT: Iroh's killer feature

All current `CounterBackend` implementations are single-writer:

| Engine | Implementation                                             | Limitation                               |
| ------ | ---------------------------------------------------------- | ---------------------------------------- |
| SQLite | `ON CONFLICT DO UPDATE SET value = value + excluded.value` | Serialized via SQL transaction           |
| Pebble | `Get` + `Merge` (8-byte big-endian)                        | Atomic via LSM merge, but single-process |
| DuckDB | Columnar aggregation (recompute sum each read)             | No incremental writes                    |
| Memory | `map[key] += delta` under mutex                            | Single-process                           |

All assume **one process increments the counter**. Concurrent multi-process increments require external coordination (distributed lock, consensus round).

Iroh gives a **PN-Counter**: each author (device/node) maintains its own increment count. `CounterGet` sums across all authors. `CounterIncrement` adds to the local author's count only. This is **conflict-free by construction** — no locks, no coordination, guaranteed convergence.

For distributed CQRS (multiple aggregate instances across devices writing to the same counter), this is categorically superior.

### Two integration levels

#### Level 1: Iroh as a partial Engine (tactical)

Iroh as another engine in `[]Engine`. Implements MapBackend, SetBackend, CounterBackend, MultimapBackend, LogBackend. The planner routes to it for global-visibility queries.

```go
EngineProfile{
    Name:       "iroh",
    NsPerRead:  50_000,   // pessimistic: remote RTT (~50ms)
    NsPerWrite: 5_000,    // local write + sync overhead
    Replication:    ReplicationLeaderless,
    ReplicationLag: 5 * time.Second,
    NetworkRTT:     50 * time.Millisecond,   // P2P relay RTT
    Supports: map[ADT]Complexity{
        ADTMap:      ComplexityO1,
        ADTSet:      ComplexityO1,
        ADTCounter:  ComplexityO1,     // PN-Counter: O(authors), authors is small
        ADTMultimap: ComplexityOLogN,  // prefix range scan
        ADTLog:      ComplexityON,     // tail = range scan
    },
    Layouts: map[ADT]StorageLayout{
        ADTMap: LayoutKV,  // all KV — iroh-docs is a KV store
    },
}
```

Passes `adttest.RunMatrix` for the 5 supported ADTs.

**Limitation:** no pushdown, no scan, no indexes. Every read is either a point lookup or a full scan. The planner rarely picks it over SQLite/DuckDB/Pebble for anything but distributed counters.

#### Level 2: Iroh as a replication layer (strategic, recommended)

Instead of Iroh implementing backends from scratch, `iroh-docs` **wraps** any existing engine:

```go
// The replication layer — wraps a local engine, adds CRDT convergence
distributed := iroh.Replicated(
    metaengine.NewMemoryEngine(),  // or sqliteEngine, pebbleEngine, etc.
    iroh.WithNamespace(docNamespace),
    iroh.WithAuthor(authorKeypair),
)
```

The local engine handles all reads/writes with full performance — SQLite pushdown, DuckDB vectorization, Pebble point reads. Iroh sits underneath and handles cross-peer replication:

```
Write path:  Engine.MapSet() → local write → iroh-doc.Set(key, blob_hash)
Sync path:   iroh-doc entry arrives → apply to local engine
Read path:   Engine.MapGet() → local read (always fast, always local)
```

**Why this is architecturally superior:**

1. **Planner does not change** — it picks the local engine as usual. Replication is transparent.
2. **Full query power retained** — SQLite pushdown, DuckDB columnar, Pebble point reads. Iroh only handles transport/convergence.
3. **CRDT-safe operations replicate automatically** — MapSet (LWW), SetAdd (OR-Set), CounterIncrement (PN-Counter), MultiAdd, LogAppend.
4. **Non-CRDT operations stay local** — MapUpdate (atomic RMW) executes locally but does not replicate. The replication layer warns or rejects.

### Operation CRDT safety matrix

| Operation          | CRDT-safe?              | Replication behavior                         |
| ------------------ | ----------------------- | -------------------------------------------- |
| `MapSet`           | Yes (LWW)               | Replicates with timestamp                    |
| `SetAdd`           | Yes (OR-Set)            | Replicates                                   |
| `CounterIncrement` | Yes (PN-Counter)        | Replicates as per-author increment           |
| `MultiAdd`         | Yes (OR-Set per key)    | Replicates                                   |
| `LogAppend`        | Yes (append per author) | Replicates                                   |
| `MapUpdate` (RMW)  | **No**                  | Local only — not a CRDT operation            |
| `MapDelete`        | **Conditional**         | Tombstone (safe) vs physical delete (unsafe) |

> **Footgun guard:** `MapUpdate` (atomic read-modify-write) cannot replicate — a CRDT cannot guarantee atomicity across replicas. On a distributed engine, `MapUpdate` executes locally but the result never syncs. **Recommended:** the planner should emit a **WARN diagnostic** when a query's fold includes `MapUpdate` and routes it to a `ReplicationLeaderless`/`MultiLeader` engine: *"non-CRDT operation MapUpdate will not replicate — use MapSet with LWW timestamp instead"*. This makes the silent-local-execution failure mode visible at plan time, not at runtime.

---

## Part 5: What This Unlocks

The combination — metaengine cost-based planning + Iroh CRDT convergence — gives the system capabilities no other Go CQRS library has:

| Capability                | Current (local engines only)          | With Iroh                                          |
| ------------------------- | ------------------------------------- | -------------------------------------------------- |
| Distributed counters      | Requires external coordination        | PN-Counter: conflict-free, instant                 |
| Offline read models       | Impossible (server must be reachable) | Local copy on every device, converge on reconnect  |
| Geo-distributed reads     | Single-origin latency                 | Local read everywhere, CRDT sync in background     |
| Multi-writer projections  | Single-writer upsert                  | CRDT merge (LWW-Map, OR-Set)                       |
| Edge/IoT projections      | Requires broker + connectivity        | Direct P2P sync, works behind NAT                  |
| Collaborative read models | Not supported                         | Multiple devices writing, converging automatically |

---

## Part 6: Integration Challenge — Iroh Is Rust

There is no official Go SDK. Three paths:

| Approach                                                     | Effort    | Precedent in this repo                                                                                                             |
| ------------------------------------------------------------ | --------- | ---------------------------------------------------------------------------------------------------------------------------------- |
| **CGo FFI** over Iroh's C bindings                           | High      | `stack/duckdb` already does this for DuckDB's C++ engine — same isolation pattern (separate module, `//go:build cgo`, static link) |
| **Sidecar process** (Iroh node + local gRPC/Unix socket)     | Medium    | None directly, but clean separation                                                                                                |
| **Pure-Go reimplementation** of Iroh's QUIC+gossip protocols | Very high | Not recommended — defeats the purpose                                                                                              |

The CGo path is the most consistent with existing architecture: isolate in `transport/iroh/` (or `metaengine/irohengine/`) with its own `go.mod` so consumers who do not import it never need CGo — exactly the `stack/duckdb` precedent.

---

## Part 7: Materialize-vs-Replay Cost Model Update

The cost model currently has two options:

```
replay_cost(q)      = read_rate × avg_stream_length × fold_cost
materialize_cost(q) = write_rate × fold_cost + read_rate × query_cost
```

The all-eventual model adds a third row — distributed materialization:

```
distributed_materialize_cost(q) = write_rate × fold_cost + sync_cost + read_rate × local_query_cost
```

Where `sync_cost` is the amortized per-write replication overhead:

```
sync_cost(q) = write_rate × (peer_count × value_size / bandwidth + reconciliation_overhead)
```

Range-based set reconciliation makes `reconciliation_overhead` near-constant (a single fingerprint exchange for in-sync peers), so for steady-state workloads `sync_cost` collapses to `write_rate × peer_count × value_size / bandwidth` — dominated by bandwidth, not coordination. This is a **fixed write-time cost** that does not affect read latency (reads are always local). Distributed materialization wins when:

- Multiple geographies need low-latency reads (no single-origin bottleneck)
- Devices go offline frequently (edge/mobile/IoT)
- Per-node write volume is low but aggregate read volume is high

Same cost comparison, one more row. No new dimension.

---

## Part 8: Implementation Scope

### Changes for the replication model (DONE — without Iroh)

| Component            | Change                                                                           | Status |
| -------------------- | -------------------------------------------------------------------------------- | ------ |
| `EngineProfile`      | Add `Replication` + `ReplicationLag` + `NetworkRTT` fields                       | Done   |
| All existing engines | Zero-value defaults (`ReplicationNone`, lag=0, RTT=0) — no code changes          | Done   |
| `QueryConfig`        | **No changes** — replication is engine-only                                      | N/A    |
| Cost estimator       | `NetworkRTT` added as additive latency                                           | Done   |
| Planner              | Pass `profile.NetworkRTT` to cost estimator                                      | Done   |
| Tests                | 6 tests pinning the model (zero-value defaults, RTT additive, no volume scaling) | Done   |
| `adttest.RunMatrix`  | Unchanged                                                                        | N/A    |

**Fully backward compatible.** No existing behavior changes. All new fields default to zero.

### Changes required for Iroh integration (future work)

| Component                             | Change                                                                                                          | Effort  |
| ------------------------------------- | --------------------------------------------------------------------------------------------------------------- | ------- |
| `metaengine/irohengine/` (new module) | CGo FFI over Iroh C bindings, implement MapBackend + SetBackend + CounterBackend + MultimapBackend + LogBackend | High    |
| `adttest.RunMatrix`                   | Add irohengine factory — passes 5 supported ADT scenarios                                                       | Small   |
| Iroh EngineProfile                    | `Replication: ReplicationLeaderless`, `ReplicationLag: 5s`, `NetworkRTT: ~50ms`                                 | Trivial |
| Level 2 replication wrapper           | `iroh.Replicated(engine, ...)` — intercepts writes, syncs via iroh-docs                                         | High    |

### Recommendation

**Replication model is shipped.** Next step: prototype Level 2 (replication wrapper) as a proof-of-concept.

The hybrid approach is likely the sweet spot: `iroh.Replicated(pebbleEngine)` for Map/Set/Multimap/Log (local engine performance + CRDT convergence), but `irohengine` directly for Counter (where the PN-Counter semantic IS the implementation).

---

## References

- [Iroh GitHub](https://github.com/n0-computer/iroh) — "Dial keys, not IPs"
- [Iroh Docs — Documents Protocol](https://docs.iroh.computer/protocols/documents) — CRDT key-value store
- [Iroh Docs — Blobs Protocol](https://docs.iroh.computer/protocols/blobs) — content-addressed storage
- [CALM Theorem](https://link.springer.com/chapter/10.1007/978-3-642-04243-6_27) — Consistency As Logical Monotonicity
- [Range-based Set Reconciliation](https://arxiv.org/abs/2212.13567) — Meyer 2022, the sync algorithm behind iroh-docs
- [meta-engine-design.md](meta-engine-design.md) — canonical design doc
- [meta-engine-assumptions-and-query-planning.md](meta-engine-assumptions-and-query-planning.md) — cost model + planning assumptions
- [ADR-0084](../adr/0084-metaengine-layered-architecture.md) — layered architecture
- [ADR-0085](../adr/0085-metaengine-new-adts.md) — Vector, Search, Spatial ADTs
