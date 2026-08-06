# ADR-0118: Badger Engine (Pure-Go LSM)

**Date:** 2026-08-06
**Status:** Accepted
**Related:** ADR-0062 (dependency boundary), ADR-0061 (SQLite engine)

## Context

The metaengine supports multiple storage backends (Memory, SQLite, Pebble, DuckDB, Postgres).
Consumers who want a pure-Go embedded LSM-tree key-value store currently have only one option:
Pebble (`cockroachdb/pebble`). While Pebble is excellent, having a second pure-Go LSM option
gives consumers a choice and avoids monoculture dependency risk.

Badger (`dgraph-io/badger/v4`) is a widely-used pure-Go LSM-tree key-value store with:

- Single-file deployment (pure Go, no CGo)
- Built-in compression (ZSTD)
- GC-based value log for efficient handling of large values
- Transactional API with optimistic concurrency control

## Decision

Create `metaengine/badgerengine/` as a separate Go module following the pebbleengine pattern.
It implements the same set of backend interfaces:

| Backend          | Implementation Notes                                        |
| ---------------- | ----------------------------------------------------------- |
| MapBackend       | O(1) point read via LSM                                     |
| MapUpdater       | Mutex-guarded read-modify-write (same as Pebble)            |
| ScanBackend      | O(N) prefix scan + Go sort (degraded, no secondary indexes) |
| SetBackend       | O(1) point read                                             |
| CounterBackend   | O(1) increment, O(N) CounterGet (prefix scan)               |
| GraphBackend     | O(N^d) BFS via prefix scan                                  |
| MultimapBackend  | Sequence-keyed entries, prefix scan on read                 |
| LogBackend       | Sequence-keyed entries                                      |
| StreamLogBackend | Per-stream + global journal dual-write                      |
| AtomicAppender   | Mutex-guarded version check + append                        |
| StreamingScan    | Lazy iterator over prefix range                             |

Not implemented (matching Pebble): VectorBackend, SearchBackend, SpatialBackend, PushdownScan,
LayoutPlanner, RawValueReader, RawScanReader.

## Cost Profile

Badger's LSM-tree architecture provides fast point reads but higher write costs
than Pebble due to LSM compaction overhead. Calibrated 2026-08-06 on AMD Ryzen
AI MAX+ 395 (3-run median):

| Operation         | Badger (ns/op) | Pebble (ns/op) | Ratio |
| ----------------- | -------------- | -------------- | ----- |
| MapSet (write)    | ~4300          | ~2500          | 1.7x  |
| MapGet (read)     | ~1200          | ~1300          | 0.9x  |
| CounterIncrement  | ~5800          | ~2000          | 2.9x  |

**Key finding:** Badger reads are slightly faster than Pebble (comparable LSM
point lookups), but writes are significantly more expensive. CounterIncrement
is especially costly due to Badger's transaction overhead on read-modify-write.

Calibrated constants in `engine.go`:
- `BadgerNsPerOp = 4300.0` (measured from MapSet)
- `BadgerNsPerRead = 1200.0` (measured from MapGet)
- `BadgerNsPerWrite = 4300.0` (measured from MapSet)

## Persistence

- `NewBadgerEngine("")` → in-memory (`WithInMemory(true)`) → PersistenceVolatile
- `NewBadgerEngine("/path")` → on-disk → PersistencePersistent

Sequence counters (log, multimap, journal, stream) are seeded from existing data on restart
to prevent key collisions, matching the Pebble engine's restart-safety guarantee.

## Alternatives Considered

### Bbolt engine

Bbolt is a B-tree (not LSM) pure-Go KV store. It offers different tradeoffs (better for
read-heavy workloads, worse for write-heavy). The `storage/bbolt/` module already exists for
event storage. A metaengine Bbolt adapter could be added later if demand exists.

### Embedded Postgres

Rejected — too heavy for an embedded use case. The `pgengine` module already covers Postgres
for server deployments.

## Consequences

- **Positive:** Consumers have a second pure-Go embedded LSM option alongside Pebble.
- **Positive:** Full adttest parity verified (all 8 core ADTs pass cross-engine matrix).
- **Negative:** Additional dependency surface (`dgraph-io/badger/v4` and its transitive deps).
