# Metaengine: Eventual Consistency Model and Iroh Integration

> **Status:** Design exploration — not yet implemented.
> **Date:** 2026-08-02
> **Related:** [meta-engine-design.md](meta-engine-design.md), [meta-engine-assumptions-and-query-planning.md](meta-engine-assumptions-and-query-planning.md), [ADR-0084](../adr/0084-metaengine-layered-architecture.md)

---

## TL;DR

1. CQRS read models are **already eventually consistent** — the projection host has measurable lag (`LagDuration()`, `LagPerProjection()`). The only strong operation is the event store's optimistic-concurrency append.
2. Modeling "everything is eventual" lets the planner **drop consistency as a dimension** and replace it with **visibility** (local vs global) — a more honest and useful axis.
3. This reframing is the natural entry point for **[Iroh](https://github.com/n0-computer/iroh)** integration — Iroh's `iroh-docs` CRDT key-value store becomes a `VisibilityGlobal` engine whose monotonic fold operations converge without coordination.
4. The change is **backward compatible**: all existing engines default to `VisibilityLocal`, all existing queries default to local visibility, no existing behavior changes.

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

- **Visibility scope** — does this engine see writes from other nodes?
- **Typical lag** — how stale are reads, typically?

---

## Part 2: The Visibility Model

### Replacing consistency with visibility

| Dimension | Old model (proposed then rejected) | Visibility model |
|---|---|---|
| Consistency | Binary: strong vs eventual — planner gates on it | **Eliminated** — all engines are eventual |
| Visibility | Not modeled | **New** — local vs global: does this engine see writes from other nodes? |
| Lag | Not modeled (assumed zero for "strong" engines) | **Engine property** — typical delay between write and read visibility |
| Cost | Latency-based | Unchanged — latency-based |

### EngineProfile changes

```go
type VisibilityModel string

const (
    VisibilityLocal  VisibilityModel = "local"  // Memory, SQLite, Pebble, DuckDB
    VisibilityGlobal VisibilityModel = "global"  // Iroh (iroh-docs), future distributed engines
)

type EngineProfile struct {
    // ...existing fields (Name, Supports, Layouts, NsPerOp, NsPerRead, NsPerWrite)...

    // Visibility declares whether this engine sees writes from other processes.
    // Local:  this engine only sees writes from this process.
    // Global: this engine sees writes from all processes (eventually, via CRDT).
    Visibility VisibilityModel

    // TypicalLag: expected delay between a write and it being readable.
    // Local engines: ~projection processing time (microseconds to ms).
    // Global engines: ~CRDT sync time (ms to seconds).
    // Used for cost estimation, NOT for gating.
    TypicalLag time.Duration
}
```

### Planner routing

The planner gains one new filter and one cost adjustment. No consistency gate.

**Visibility filter:**

```go
func visibilityRule(meta queryMeta, engines []Engine) []Engine {
    if meta.visibility == VisibilityGlobal {
        // Query needs cross-node data — only global engines qualify
        var eligible []Engine
        for _, eng := range engines {
            if eng.Profile().Visibility == VisibilityGlobal {
                eligible = append(eligible, eng)
            }
        }
        return eligible
    }
    // Local queries: all engines eligible (local engines are cheaper)
    return engines
}
```

**Cost estimator (lag added to latency):**

```go
func estimateCost(complexity Complexity, volume int64, nsPerOp float64, lag time.Duration) CostEstimate {
    ops := opsForComplexity(complexity, volume)
    latencyMs := float64(ops)*nsPerOp/1e6 + float64(lag.Milliseconds())
    return CostEstimate{
        Complexity:          complexity,
        Volume:              volume,
        EstimatedOps:        ops,
        EstimatedLatencyMs:  latencyMs,
    }
}
```

A global engine with 5s lag loses to a local engine with ~0ms lag on every single-node query. The planner naturally prefers local. Only queries that explicitly opt into global visibility become candidates for global engine routing.

### Query declaration

```go
// Default: local visibility (backward compatible — all current behavior)
metaengine.Query[Input, Result]("orders", folds...)

// Explicit: needs cross-node data (distributed counter, multi-device read model)
metaengine.Query[Input, Result]("order_counts",
    metaengine.On(OrderCreated{}, func(e OrderCreated) metaengine.Delta { ... }),
    metaengine.WithVisibility(metaengine.VisibilityGlobal),
)
```

Default is `VisibilityLocal`. Every existing query keeps working unchanged.

---

## Part 3: The CALM Theorem Connection

### Why monotonic folds are CRDT-safe

The [CALM theorem](https://link.springer.com/chapter/10.1007/978-3-642-04243-6_27) (Consistency As Logical Monotonicity) states: a distributed program converges without coordination if and only if it is monotonic — its output only grows, never retracts.

The metaengine ADTs are already monotonic by construction:

| ADT | Fold operations | Monotonic? | CRDT equivalent |
|---|---|---|---|
| Map | `FoldInsert`, `FoldUpdate` | Yes (LWW by timestamp) | LWW-Map |
| Set | `FoldSet` | Yes (add-only) | OR-Set (G-Set if no remove) |
| Counter | `FoldCount` (+/-) | Yes (per-author) | PN-Counter |
| Multimap | `FoldMultiInsert` | Yes (add per key) | OR-Set per key |
| Log | `FoldAppend` | Yes (append per author) | Per-author append-only |

### Non-monotonic operations

The non-monotonic operations are the exceptions, and they are already the operations that CRDTs handle specially:

| Operation | Monotonic? | CRDT behavior |
|---|---|---|
| `MapUpdate` (atomic RMW) | No | Local only — not a CRDT operation; does not replicate |
| `MapDelete` | Conditional | Tombstone (safe, replicates) vs physical delete (unsafe, local only) |
| `FoldRemove` (Set) | No | Tombstone-based OR-Set (safe if tombstones are tracked) |

**This means the fold operations are already CRDT-safe for 5 of 7 ADTs.** The all-eventual model is not a redesign — it is an acknowledgment of what the system already is.

### The philosophical payoff

The all-eventual model reveals what the architecture has been quietly saying:

> **The event log is the source of truth. Everything else is a cache.**

The planner's job is not to guarantee consistency — it is to pick the cheapest cache that has the data scope the caller needs. Local caches (Memory, SQLite, Pebble, DuckDB) are fast but process-local. Global caches (Iroh) are slower but cross-node. The caller decides what scope they need; the planner finds the cheapest option.

Since folds are monotonic (CRDT-safe), coordination-free convergence is guaranteed by the CALM theorem. The planner does not need to reason about consistency because monotonicity *proves* eventual consistency. It only needs to reason about cost and visibility.

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

| Backend | iroh-docs implementation | CRDT type |
|---|---|---|
| `MapBackend` | `doc.Set(key, hash)` / `doc.Get(key)` then resolve blob | LWW-Map (last-writer-wins by timestamp) |
| `SetBackend` | `doc.Set(key, empty)` / `doc.Get(key) != nil` | OR-Set (observed-remove) |
| `CounterBackend` | Each author keeps own increment count; `Get` sums across authors | **PN-Counter** (conflict-free) |
| `MultimapBackend` | `doc.Set(prefix+key+seq, value)` / prefix range scan | OR-Set per key |
| `LogBackend` | `doc.Set(author+seq, value)` / reverse range scan | Per-author append-only |

What it **cannot** serve:

| Backend | Why not |
|---|---|
| `PushdownScan` / `ScanBackend` | No server-side filtering — all filtering is client-side after fetch. O(N) always. |
| `GraphBackend` | No graph operations |
| `VectorBackend` / `SearchBackend` / `SpatialBackend` | No similarity/text/geo search |
| `LayoutPlanner` / `LayoutPlanApplier` | No DDL, no indexes |
| `MapUpdater` (atomic RMW) | CRDT does not support atomic read-modify-write — the "read" may be stale |

This is fine — Pebble also does not implement Vector/Search/Spatial. The planner routes by capability.

### The Counter ADT: Iroh's killer feature

All current `CounterBackend` implementations are single-writer:

| Engine | Implementation | Limitation |
|---|---|---|
| SQLite | `ON CONFLICT DO UPDATE SET value = value + excluded.value` | Serialized via SQL transaction |
| Pebble | `Get` + `Merge` (8-byte big-endian) | Atomic via LSM merge, but single-process |
| DuckDB | Columnar aggregation (recompute sum each read) | No incremental writes |
| Memory | `map[key] += delta` under mutex | Single-process |

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
    Visibility: VisibilityGlobal,
    TypicalLag: 5 * time.Second,
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

| Operation | CRDT-safe? | Replication behavior |
|---|---|---|
| `MapSet` | Yes (LWW) | Replicates with timestamp |
| `SetAdd` | Yes (OR-Set) | Replicates |
| `CounterIncrement` | Yes (PN-Counter) | Replicates as per-author increment |
| `MultiAdd` | Yes (OR-Set per key) | Replicates |
| `LogAppend` | Yes (append per author) | Replicates |
| `MapUpdate` (RMW) | **No** | Local only — not a CRDT operation |
| `MapDelete` | **Conditional** | Tombstone (safe) vs physical delete (unsafe) |

---

## Part 5: What This Unlocks

The combination — metaengine cost-based planning + Iroh CRDT convergence — gives the system capabilities no other Go CQRS library has:

| Capability | Current (local engines only) | With Iroh |
|---|---|---|
| Distributed counters | Requires external coordination | PN-Counter: conflict-free, instant |
| Offline read models | Impossible (server must be reachable) | Local copy on every device, converge on reconnect |
| Geo-distributed reads | Single-origin latency | Local read everywhere, CRDT sync in background |
| Multi-writer projections | Single-writer upsert | CRDT merge (LWW-Map, OR-Set) |
| Edge/IoT projections | Requires broker + connectivity | Direct P2P sync, works behind NAT |
| Collaborative read models | Not supported | Multiple devices writing, converging automatically |

---

## Part 6: Integration Challenge — Iroh Is Rust

There is no official Go SDK. Three paths:

| Approach | Effort | Precedent in this repo |
|---|---|---|
| **CGo FFI** over Iroh's C bindings | High | `stack/duckdb` already does this for DuckDB's C++ engine — same isolation pattern (separate module, `//go:build cgo`, static link) |
| **Sidecar process** (Iroh node + local gRPC/Unix socket) | Medium | None directly, but clean separation |
| **Pure-Go reimplementation** of Iroh's QUIC+gossip protocols | Very high | Not recommended — defeats the purpose |

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

Where `sync_cost` depends on peer count, bandwidth, and reconciliation efficiency. This wins when:
- Multiple geographies need low-latency reads (no single-origin bottleneck)
- Devices go offline frequently (edge/mobile/IoT)
- Per-node write volume is low but aggregate read volume is high

Same cost comparison, one more row. No new dimension.

---

## Part 8: Implementation Scope

### Changes required for the visibility model (without Iroh)

| Component | Change | Effort |
|---|---|---|
| `EngineProfile` | Add `Visibility` + `TypicalLag` fields | Small |
| All existing engines | Set `Visibility: VisibilityLocal`, `TypicalLag: ~1ms` | Trivial (4 profile constructors) |
| `Query` declaration | Add `WithVisibility()` option, default `VisibilityLocal` | Small |
| Planner | Add `visibilityRule` to the rule pipeline | Small |
| Cost estimator | Add `lag` to latency calculation | Trivial |
| `adttest.RunMatrix` | Unchanged | None |
| Existing queries/tests | Unchanged — default visibility is local | None |

**Fully backward compatible.** No existing behavior changes. The visibility dimension is opt-in.

### Changes required for Iroh integration (on top of visibility model)

| Component | Change | Effort |
|---|---|---|
| `metaengine/irohengine/` (new module) | CGo FFI over Iroh C bindings, implement MapBackend + SetBackend + CounterBackend + MultimapBackend + LogBackend | High |
| `adttest.RunMatrix` | Add irohengine factory — passes 5 supported ADT scenarios | Small |
| Iroh EngineProfile | `Visibility: VisibilityGlobal`, `TypicalLag: 5s` | Trivial |
| Level 2 replication wrapper | `iroh.Replicated(engine, ...)` — intercepts writes, syncs via iroh-docs | High |

### Recommendation

**Ship the visibility model first** (Part 2), without Iroh. It is a small, backward-compatible improvement that makes the planner more honest and prepares the architecture for any future distributed engine. Then prototype Level 2 (replication wrapper) as a proof-of-concept.

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
