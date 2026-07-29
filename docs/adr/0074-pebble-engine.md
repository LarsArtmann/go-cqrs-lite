# ADR-0074: Pebble Metaengine (cost profile & the slices.Backward lesson)

|             |                                                                               |
| ----------- | ----------------------------------------------------------------------------- |
| **Status**  | Accepted                                                                      |
| **Date**    | 2026-07-29                                                                    |
| **Context** | The metaengine needs a third engine with a genuinely different cost profile   |

## Context

The metaengine core shipped with two engines: in-memory (O(1) map) and SQLite
(O(log N) B-tree). They are cost-similar — both fast point reads. To make the
cost-based planner meaningful (and to offer a disk-backed KV that is not SQL), a
third engine with a **different** profile was needed. Pebble (cockroachdb/pebble)
is a production-grade LSM: O(1) point reads via the memtable + bloom filter,
clearly faster than SQLite on point lookups.

Because Pebble is a heavyweight dependency, the engine lives in a **separate
module** (`metaengine/pebbleengine`) so the zero-dependency core stays clean
(ADR-0062).

## Decision

Implement `pebbleEngine` covering all 7 ADTs, calibrated against the memory and
SQLite engines:

- **Map/Set point lookups**: O(1) LSM point read — ~708 ns/op (`PebbleNsPerRead`),
  ~7x faster than SQLite.
- **Writes**: ~1,785 ns/op (`PebbleNsPerWrite`). The single `NsPerOp` was split
  into `NsPerRead`/`NsPerWrite` on the `EngineProfile` (backward-compatible:
  zero falls back to `NsPerOp`) so the planner's read-cost estimate uses the
  read constant.
- **Counter/SortedMap/Graph**: prefix scans + Go sort — degraded O(N), no
  secondary index. Documented in the profile (`ComplexityON`).

Keys use a null-byte separator (`\x00`) and prefix scans compute an exclusive
upper bound via `nextKey(prefix)`.

### The slices.Backward lesson

The first `nextKey` implementation ranged over `slices.Backward(result)`:

```go
for _, v := range slices.Backward(result) { v++; if v != 0 { return result } }
```

`v` is a **copy** of each element, so `v++` mutated the copy and `result` was
never changed. The upper bound then equalled the lower bound and **every prefix
scan silently returned empty**. Worse, the auto-commit daemon reverted the fix to
the broken form mid-session, and a status report claimed GREEN on stale
evidence.

Fixes:
1. Direct index access: `for i := len(result)-1; i >= 0; i-- { result[i]++ ... }`.
2. A pure-function regression test (`nextkey_test.go`) pins the helper.
3. `MapUpdate`'s read-modify-write is guarded by the engine mutex (atomicity
   parity with the SQLite engine, ADR-0067), verified by a 100-goroutine test.
4. `NewPebbleEngine(dir)` now actually uses `dir` (it previously passed `""` to
   `pebble.Open`, silently breaking disk-backed mode); pinned by
   `disk_backed_test.go`.

## Consequences

- Pebble is the disk-backed KV engine of choice for point-read-heavy
  projections; SQLite remains better for filtered/sorted scans (pushdown,
  ADR-0072).
- The `slices.Backward` copy footgun is documented in AGENTS.md alongside the
  other Go pitfalls.
- Disk-backed mode is now real (data survives reopen), enabling production use
  outside `vfs.NewMem()`.

## Alternatives considered

- **Badger**: another Go LSM, but Pebble is the CockroachDB-grade default and
  already used by `storage/pebble`.
- **Hand-rolled LSM**: rejected — no reason to reimplement a production LSM.
