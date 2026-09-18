# ADR-0141: Native Temporal Versioned Cells (BigTable-Aligned)

**Status:** Accepted
**Date:** 2026-09-18
**Related:** [ADR-0112](0112-es-native-planner.md), [ADR-0136](0136-temporal-composability-contract.md), [ADR-0140](0140-vector-distance-semantics-contract.md), [`docs/planning/meta-engine-layered-architecture.md` §3](../planning/meta-engine-layered-architecture.md)

## Context

The shipped temporal surface (`metaengine.VersionedStorage`, `Store.ExecuteAsOf`) is a
Tier-4 spike with three gaps:

1. **Memory-only.** No persistent engine versions cells, so as-of reads are impossible
   after restart and unusable in production topologies.
2. **Wall-clock stamps.** The memory engine stamps `time.Now()` at write time. During
   replay the projection host re-folds historical events; their original event time is
   discarded, so versioned cells reflect *replay* order, not *event* order — the exact
   hazard the layered-architecture doc §3 calls out.
3. **Dead signal.** `AsOfSignal` documents planner routing that does not exist; the only
   as-of entry point is the manual `Store.ExecuteAsOf` call.

Meanwhile Google Cloud BigTable (and HBase, Cassandra, ScyllaDB) store every cell as a
timestamped version natively — `(row, family, column, timestamp) → value` — with GC
policies (`max_versions`, `max_age`) as the retention knob. An engine that maps onto
that model gets O(1) time travel for free; engines that do not can emulate the model
(history tables, version chains).

## Decision

### 1. The temporal cell model (the contract)

A versioned cell is `(collection, key, timestamp) → value` with:

- **As-of read** = the latest version with `ts <= T`; `ErrNotFound` if none exists or
  the resolved version is a tombstone.
- **Tombstone** = a version whose value is nil/empty. Deletion is a timestamped write,
  never a hard erase (matches ADR-0114 deletion-as-event philosophy).
- **Same-timestamp writes** = last-writer-wins. This mirrors BigTable cell dedup
  (identical `(row, column, ts)` collapses) and is pinned identically across engines.
- **Out-of-order stamps are legal.** Replays and clock skew deliver old events late;
  chains/history must resolve by timestamp, not append order.
- **Retention** may prune versions (never the latest) along two axes: `MaxVersions`
  (keep newest N) and `MaxAge` (drop older than cutoff). Retention GC'd versions are
  simply gone — as-of reads older than the window return `ErrNotFound`.

### 2. Capability interfaces (additive-only in v4.x)

- `VersionedStorage` (existing) — as-of point reads: `MapGetAsOf`, `MapExistsAsOf`.
- `VersionedWriter` (new) — `MapSetAt`, `MapDeleteAt`: timestamped cell writes, the
  BigTable `Mutation.Set(family, column, ts, value)` analog.
- `CellHistoryReader` (new) — `MapHistory(col, key, from, to)`: BigTable range reads,
  the full version history of one cell.
- `RetentionPolicy` (new) — `MaxVersions`, `MaxAge`, the column-family GC analog.

Existing interfaces are NOT grown (growing core interfaces is breaking; v5 per
ADR-0123 discipline). Engines adopt temporal capabilities by implementing the
optional interfaces; consumers check via type assertion or `Store`.

### 3. Stamp derivation: event time, not write time

`CellTimestamp(rec record.Record)` derives a cell timestamp as
`Stored → Received → Created → time.Now()` (first non-zero `record.Stamp` wins).
The fold write path (`applyFoldInsert/Update/Remove`) prefers `VersionedWriter`
when the engine implements it, passing the stamp. Replay therefore rebuilds the
true temporal order, closing gap 2.

### 4. AsOf intent travels in the query input

A query input struct may declare `AsOf time.Time` (the same meta-field convention as
`Limit`/`After`/`Depth` — excluded from filter inference). Precomputed on the
`QueryDecl` at construction; at execution a **non-zero** `AsOf` routes the point
lookup through `VersionedStorage` (nil result when the key did not exist then);
zero means "latest", the normal path. This makes `AsOfSignal`'s documented behavior
real at the execute layer. `Store.ExecuteAsOf` remains the explicit named form.

### 5. Planner honesty

A plan rule warns when an AsOf-declaring query is assigned to an engine that does not
implement `VersionedStorage` — the "honest degradation" row of the layered-architecture
§3 matrix. Execution on such an engine fails loudly (`ErrUnsupportedADT` family),
never silently returning latest-only.

### 6. Engine coverage

| Engine | Mechanism | Notes |
| --- | --- | --- |
| memory | version chains (sorted insert), retention trim on write | wall-clock fallback when no stamp |
| sqlite | `meta_cell_versions(collection, key, ts, value)` history table; PK collision = REPLACE | latest still served from the hot table |
| bigtable (new module) | **native**: cell timestamps ARE the mechanism; GC via column-family `GCPolicy` | BigTable is millisecond-granularity — same-ms same-key writes collapse (documented, tested) |

### 7. One conformance contract, every engine

`adttest.TemporalConformance` pins the §1 semantics (as-of resolution, tombstones,
same-ts last-write-wins, out-of-order stamps, retention, history ranges) for every
engine claiming temporal capabilities — the same pattern ADR-0140 used for vector
distance. Engine-specific notes (ms truncation) are asserted in-engine.

## Consequences

- Versioned storage costs disk/memory proportional to write rate × retained depth;
  `RetentionPolicy` is the operator's knob, and `MaxVersions=1` collapses a versioned
  engine to latest-only at near-zero overhead.
- The memory engine's `NewMemoryEngineWithVersioning` becomes variadic
  (`opts ...VersioningOption`) — source-compatible, golden regenerated.
- The new `metaengine/bigtableengine` module is dep-isolated (only engine importing
  `cloud.google.com/go/bigtable`); tests run against the in-process `bttest` fake,
  no emulator binary needed.
- Health-driven rerouting (ADR-0137) may land an as-of read on a non-versioned engine;
  that fails loud rather than lying — acceptable and documented.
