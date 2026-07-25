# ADR-0065: Extract `idempotency/` into Standalone `go-idempotency` Repository

| Field    | Value                                                                                              |
| -------- | -------------------------------------------------------------------------------------------------- |
| Status   | Proposed                                                                                           |
| Date     | 2026-07-25                                                                                         |
| Deciders | Lars Artmann                                                                                       |
| Related  | ADR-0046 (seven-tier model), ADR-0064 (retry extraction), ROADMAP (module extraction)              |
| Supersedes | —                                                                                                |

## Context

The `idempotency/` module provides deduplication for at-least-once message
delivery. It has a clean 3-method `Store` interface (`Seen`, `Record`,
`CheckAndRecord`), an in-memory implementation, and two backend subpackages
(`kvstore/` for KV-backed, `sqlstore/` for SQLite/Postgres). The core module
depends only on `go-error-family` — zero CQRS coupling.

Currently `idempotency/v4` lives inside the `go-cqrs-lite` monorepo. It has 4
production consumers: `middleware`, `example/taskmanager`, plus its own
`kvstore/` and `sqlstore/` subpackages. External consumers who need idempotency
deduplication but not CQRS must import the monorepo to get a 187-line store
interface.

The ROADMAP calls for extracting zero-coupling modules into standalone repos.
This is the companion to ADR-0064 (retry extraction).

## Current State

```
idempotency/
├── doc.go              (31 lines — package doc + quick start)
├── store.go            (156 lines — Store interface, ErrDuplicate, MemoryStore)
├── store_test.go       (225 lines — tests for core)
├── kvstore/
│   ├── store.go        (168 lines — KV-backed Store impl)
│   └── store_test.go   (133 lines — KV tests)
├── sqlstore/
│   ├── store.go        (198 lines — SQL-backed Store impl, SQLite + Postgres)
│   └── store_test.go   (299 lines — SQL tests, concurrent race tests)
└── README.md           (96 lines — docs)
```

Total: 553 source + 657 test = 1,210 lines across 3 Go modules.

### Dependencies

| Module               | Prod Deps                          | Test Deps              |
| -------------------- | ---------------------------------- | ---------------------- |
| `idempotency/v4`     | `go-error-family`                  | ginkgo, gomega         |
| `idempotency/kvstore/v4` | `idempotency/v4`, `kv/v4`, `go-error-family` | ginkgo, gomega |
| `idempotency/sqlstore/v4` | `idempotency/v4`, `go-error-family`, `modernc.org/sqlite` | — |

### Public API (must be preserved)

**Core package:**

| Symbol                                            | Type      |
| ------------------------------------------------- | --------- |
| `Store` interface (`Seen`, `Record`, `CheckAndRecord`) | Type   |
| `ErrDuplicate`                                    | Sentinel  |
| `MemoryStore` struct                              | Type      |
| `NewMemoryStore(sweepInterval) *MemoryStore`      | Function  |

**Subpackage `kvstore`:**

| Symbol                  | Type      |
| ----------------------- | --------- |
| `KVBackend` interface   | Type      |
| `Store` struct          | Type      |
| `New(backend) *Store`   | Function  |

**Subpackage `sqlstore`:**

| Symbol                              | Type      |
| ----------------------------------- | --------- |
| `Dialect` (`DialectSQLite`, `DialectPostgres`) | Type/Consts |
| `Store` struct                      | Type      |
| `NewSQLiteStore(ctx, db) (*Store, error)`  | Function |
| `NewPostgresStore(ctx, db) (*Store, error)` | Function |
| `(*Store).Sweep(ctx) (int64, error)`       | Method   |

### Internal Consumers

| Consumer              | Files                                                    |
| --------------------- | -------------------------------------------------------- |
| `middleware/`         | `middleware/idempotency.go` + 3 test files               |
| `example/taskmanager/`| `setup.go`, `features.go`, `idempotency_test.go`         |
| `integration/`        | `idempotency_test.go`                                    |
| `idempotency/kvstore/`| Depends on parent `idempotency/v4`                       |
| `idempotency/sqlstore/`| Depends on parent `idempotency/v4`                      |

## Decision

**Extract `idempotency/` (core + kvstore + sqlstore) into a standalone
`github.com/larsartmann/go-idempotency` repository and re-export from
`go-cqrs-lite/idempotency/` for backward compatibility.**

All three modules move together because `kvstore` and `sqlstore` depend on the
core `idempotency/v4` interface. The `kvstore` subpackage also depends on
`kv/v4` from `go-cqrs-lite` — this cross-repo dependency is acceptable (it's an
optional backend, not a core requirement).

### Extraction Plan

#### Phase 1: Create the standalone repo

1. Create `github.com/larsartmann/go-idempotency` repository
2. Copy all source files verbatim (core + kvstore + sqlstore)
3. Set module paths:
   - `github.com/larsartmann/go-idempotency` (core)
   - `github.com/larsartmann/go-idempotency/kvstore` (KV backend)
   - `github.com/larsartmann/go-idempotency/sqlstore` (SQL backend)
4. Update `kvstore/go.mod` to import `go-idempotency` instead of the
   `go-cqrs-lite/idempotency/v4` path. The `kv/v4` dependency remains
   `go-cqrs-lite/kv/v4` (cross-repo dependency on go-cqrs-lite's KV interface).
5. Tag `v1.0.0` for all three modules
6. Verify `go list -m` fetches each

#### Phase 2: Re-export alias in go-cqrs-lite

Replace the core source files with type aliases:

```go
// Package idempotency re-exports github.com/larsartmann/go-idempotency.
// New consumers should import go-idempotency directly.
package idempotency

import (
    goidempotency "github.com/larsartmann/go-idempotency"
)

type Store = goidempotency.Store
type MemoryStore = goidempotency.MemoryStore

var ErrDuplicate = goidempotency.ErrDuplicate

func NewMemoryStore(sweepInterval time.Duration) *MemoryStore {
    return goidempotency.NewMemoryStore(sweepInterval)
}
```

The `idempotency/go.mod` changes to require `go-idempotency v1.0.0`.

The `kvstore/` and `sqlstore/` subpackages in go-cqrs-lite similarly become
re-export aliases pointing at `go-idempotency/kvstore` and
`go-idempotency/sqlstore`.

#### Phase 3: Update internal consumers

No consumer code changes needed — the re-export aliases preserve all import
paths and types. The `middleware` package continues to use
`idempotency/v4.Store` transparently.

### Tagging

| Repo                                | Tag      | Notes                                |
| ----------------------------------- | -------- | ------------------------------------ |
| `go-idempotency` (core)             | `v1.0.0` | Stable, 3-method Store interface     |
| `go-idempotency/kvstore`            | `v1.0.0` | KV-backed backend                    |
| `go-idempotency/sqlstore`           | `v1.0.0` | SQL-backed backend (SQLite + PG)     |
| `go-cqrs-lite/idempotency/v4`       | `v4.2.0` | Minor bump (re-export)               |

## Alternatives Considered

### A. Keep idempotency/ in the monorepo forever

**Rejected.** Same reasoning as ADR-0064. The core module has zero CQRS coupling.
The `sqlstore` backend is broadly useful (any at-least-once delivery system
needs SQL dedup), not just for CQRS. Keeping it in the monorepo inflates the
module count and forces non-CQRS consumers to import the full graph.

### B. Extract only the core, keep kvstore/sqlstore in go-cqrs-lite

**Rejected.** The subpackages are tightly coupled to the core `Store` interface.
Splitting them across repos creates a fragile import dependency: the standalone
`go-idempotency` core would need to be tagged before `kvstore` or `sqlstore`
could build in go-cqrs-lite. Moving all three together is simpler and mirrors
the existing module structure.

### C. Hard-replace (delete idempotency/, point consumers to go-idempotency)

**Rejected.** Same reasoning as ADR-0064 alternative B. Four production consumers
would need import path updates. The re-export alias provides zero-friction
migration.

### D. Merge kvstore/sqlstore into the core go-idempotency module

**Considered.** Having three separate go.mod files for idempotency is heavy. An
alternative is to make `go-idempotency` a single module with optional backends
as internal packages (e.g., `go-idempotency/kvstore` as a non-separate-module
subpackage). This reduces the number of go.mod files from 3 to 1.

**Deferred.** This is a packaging decision that doesn't affect the extraction
ADR. The standalone repo can start with 3 modules (matching current structure)
and consolidate later if desired.

## Consequences

**Positive:**
- `go-idempotency` becomes independently consumable by any at-least-once delivery system
- Clear ownership boundary: idempotency dedup is its own project
- The `sqlstore` backend (SQL dedup) gets its own release cadence
- Monorepo shrinks by 3 modules (core + 2 backends become thin aliases)

**Negative:**
- Three repos to maintain instead of one (mitigated by re-export aliases)
- `kvstore` cross-depends on `go-cqrs-lite/kv/v4` — a cross-repo dependency that
  couples the two repos at this one point. If `kv/v4` breaks, `go-idempotency/kvstore`
  breaks too.
- `go-cqrs-lite/idempotency/` becomes a thin wrapper (~30 LOC of aliases)

**Neutral:**
- All 4 internal consumers continue to work unchanged
- CI continues testing `idempotency/v4` + subpackages (they're in testModules)

## Cross-Repository Dependencies After Extraction

```
go-idempotency (standalone)
├── core:       depends on go-error-family only
├── kvstore:    depends on go-idempotency/core + go-cqrs-lite/kv/v4
└── sqlstore:   depends on go-idempotency/core + modernc.org/sqlite

go-cqrs-lite (monorepo)
├── idempotency/:       re-export alias → go-idempotency
├── idempotency/kvstore/: re-export alias → go-idempotency/kvstore
├── idempotency/sqlstore/: re-export alias → go-idempotency/sqlstore
├── middleware/:        imports idempotency/v4 (transparent alias)
├── example/taskmanager/: imports idempotency/v4 (transparent alias)
└── integration/:       imports idempotency/v4 (transparent alias)
```

## Open Questions

1. **Should `go-idempotency` keep the `kv/v4` dependency for `kvstore`?**
   The KV backend requires `kv.Store` (conditional writes). This couples
   `go-idempotency/kvstore` to `go-cqrs-lite/kv/v4`. Alternative: define a
   local `KVBackend` interface in `go-idempotency` that `kv.Store` happens to
   satisfy structurally. **Recommendation:** keep the explicit dep for now —
   the `KVBackend` interface is already local, but the types it references
   (`kv.Reader`, `kv.Writer`, `kv.ConditionalWriter`) come from `kv/v4`.

2. **Should `sqlstore` stay in `go-cqrs-lite` since it's already a separate module?**
   No — it depends on `idempotency/v4` which is being extracted. It must move
   with the core. The three modules form a cohesive unit.

3. **When to execute?** This ADR is the design step (M15). Execution requires
   creating the new repo, which is a manual step outside this codebase.
