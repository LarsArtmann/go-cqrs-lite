# metaengine/bigtableengine — Google Cloud BigTable Engine

[![Go Reference](https://pkg.go.dev/badge/github.com/larsartmann/go-cqrs-lite/metaengine/bigtableengine/v4.svg)](https://pkg.go.dev/github.com/larsartmann/go-cqrs-lite/metaengine/bigtableengine/v4)

Google Cloud BigTable-backed [metaengine](../README.md) Engine. Every map
value is a native **versioned cell** — the storage model that inspired the
temporal capability stack ([ADR-0141](../../docs/adr/0141-native-temporal-versioned-cells.md)).
As-of reads, history ranges, and tombstones are BigTable primitives
(`TimestampRangeFilterMicros`, `LatestNFilter`, empty-value cells), not
emulations. Retention is the column-family GC policy, enforced server-side.

```bash
go get github.com/larsartmann/go-cqrs-lite/metaengine/bigtableengine/v4
```

## Quick Start

```go
import (
	"cloud.google.com/go/bigtable"
	"github.com/larsartmann/go-cqrs-lite/metaengine/bigtableengine/v4"
)

// Real instance (Application Default Credentials or BIGTABLE_EMULATOR_HOST):
engine, err := bigtableengine.New(ctx, "my-project", "my-instance", "my-table")

// With retention (applied to both column families at table creation):
engine, err = bigtableengine.New(ctx, "my-project", "my-instance", "my-table",
	bigtableengine.WithGCPolicy(bigtable.UnionPolicy(
		bigtable.MaxVersionsPolicy(10),
		bigtable.MaxAgePolicy(7*24*time.Hour),
	)),
)
```

The engine auto-registers as the `"bigtable"` driver (DSN
`"project/instance/table"`); the emulator is picked up automatically via
`BIGTABLE_EMULATOR_HOST`.

## Capabilities

MapBackend, CounterBackend (native `ReadModifyWrite` increments), plus the
full temporal set, natively:

| Capability (metaengine interface) | BigTable mechanism |
| --------------------------------- | ------------------ |
| `MapBackend` (`MapSet`/`MapGet`/`MapDelete`) | Latest-version cell read/write on row key `collection\x00key`, family `cqrs`, column `v` |
| `VersionedWriter` (`MapSetAt`/`MapDeleteAt`) | `Mutation.Set` with explicit timestamp; deletes are empty-value cells (timestamped tombstones) |
| `VersionedStorage` (`MapGetAsOf`/`MapExistsAsOf`) | `ChainFilters(Family → TimestampRange → LatestN(1))` — range first, then newest |
| `CellHistoryReader` (`MapHistory`) | `TimestampRangeFilterMicros(from, to)`, newest-first |
| `HealthChecker` / `EngineResetter` / `Calibratable` | One-row read / full-table row delete / prior overrides |

`MapUpdate` (read-modify-write) is intentionally NOT implemented: fold
writes go through `MapSetAt`/`MapDeleteAt` under the store's fold locks, so
no client-side RMW is needed (ADR-0141 §"fold path").

## Temporal Semantics (ADR-0141)

- **As-of read**: latest cell with `ts <= T` (`asOfEnd` steps one full
  millisecond past the bound because the SDK truncates range bounds to ms).
- **Tombstone**: empty-value cell — as-of reads at `t >= ts` report absent;
  earlier versions survive.
- **Same-millisecond writes collapse** last-writer-wins (BigTable is
  millisecond-granularity; sub-ms timestamps are meaningless here).
- **Out-of-order stamps are legal**; latest = newest timestamp, not last write.
- **Retention = GC policy** (`WithGCPolicy`), set once at table creation and
  enforced by BigTable server-side. The engine never prunes client-side, and
  BigTable GC never removes the newest version.

## Cost Profile

| ADT     | Complexity | Notes                                          |
| ------- | ---------- | ---------------------------------------------- |
| Map     | O(1)       | Key-addressed LSM point lookup                 |
| Counter | O(1)       | Native `ReadModifyWrite` increment             |

⚠ **Priors are UNCALIBRATED** (`BigtableNsPerOp`, `BigtableNetworkRTT = 3ms`):
compile-time estimates for same-region RTT, replaced by live measurements
once `ProbeEngine` runs (see
[METAENGINE-LIVE-LATENCY-MODEL.md](../../docs/planning/METAENGINE-LIVE-LATENCY-MODEL.md)).

## Design

- **Dep-isolated module**: `cloud.google.com/go/bigtable` + `google.golang.org/api`
  + `google.golang.org/grpc` live only here; consumers that don't import this
  module never pull Google Cloud SDKs.
- **Owns its connections** when built via `New` (Close closes both clients);
  `NewWithClients` injects clients for tests/DI and never closes them.
- **Counters** are 8-byte big-endian int64 cells in family `cqrs_c`
  (BigTable's counter format), incremented server-side.
- **Reset** is a full table wipe (scan + bulk delete — BigTable has no cheap
  truncate); used by `EngineResetter` (ADR-0136 reset is total).

## Testing

The suite runs entirely against the in-process `bttest` fake
(`bigtable/apiv2/bigtabletest`): conformance (`adttest.AssertTemporalConformance`),
ms-collapse, counter round-trips, reset, DSN validation, tombstone→rebirth —
no emulator binary, no GCP credentials, no network.

⚠ **Validation is bttest-only**: never exercised against a real GCP instance
(no credentials in CI). The fake is a rough approximation of production
behavior — especially GC policy timing and request routing. Budget a
real-instance smoke test before production use.

## Related Modules

- [**metaengine**](../README.md) — Core planner, temporal interfaces, `Engine`
- [**metaengine/adttest**](../adttest/) — Cross-engine conformance harness
- [ADR-0141](../../docs/adr/0141-native-temporal-versioned-cells.md) — The temporal cell contract
