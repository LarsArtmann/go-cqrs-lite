# Multi-Module Monorepo Migration Plan

**Date:** 2026-04-23
**Status:** Draft
**Minimum Go Version:** 1.25 (for subdirectory module root support)

## Go 1.25+ Foundations

Go 1.25 introduced two features that make this monorepo structure first-class:

### Subdirectory Module Roots

The `go` command now supports using a **subdirectory of a repository** as the module root.
This is configured via the `go-import` meta tag:

```html
<meta name="go-import" content="github.com/larsartmann/go-cqrs-lite/core mod https://github.com/larsartmann/go-cqrs-lite core">
```

This means `github.com/larsartmann/go-cqrs-lite/core` resolves to the `core/` subdirectory
of the repo — not a separate repo. No more `replace` directives for development,
no more multi-repo confusion.

### The `ignore` Directive

A new `go.mod` `ignore` directive tells the `go` command to skip directories
(e.g., examples, docs, scripts) during package pattern matching like `./...`.
This keeps `go test ./...` fast and focused.

### The `work` Package Pattern

The new `work` pattern matches all packages in workspace modules — replacing
the older `main` pattern terminology. Useful for CI: `go test work./...`.

**Bottom line:** Multi-module monorepos are now a first-class Go pattern, not a workaround.
OpenTelemetry, gRPC, and prometheus already use this. We should too.

## Goal

Restructure go-cqrs-lite as a multi-module monorepo so users only pay for what they import.

## Core Principle

> Each module has its own `go.mod` with **only the dependencies it needs**.
> A root `go.work` ties them together for development.

## Module Layout

```
go-cqrs-lite/
├── go.work
│
├── core/                           # github.com/larsartmann/go-cqrs-lite/core
│   └── go.mod                      # deps: cockroachdb/errors, oklog/ulid
│   ├── command/
│   ├── query/
│   ├── event/                      # interfaces only (Store, Bus — no MemoryStore)
│   ├── aggregate/
│   └── pkg/
│       ├── id/                     # branded IDs
│       └── errors/                 # delete — unused, dead code
│
├── memory/                         # github.com/larsartmann/go-cqrs-lite/memory
│   └── go.mod                      # deps: core
│   ├── store/                      # MemoryStore, MemorySnapshotStore
│   └── bus/                        # MemoryBus
│
├── catalog/                        # github.com/larsartmann/go-cqrs-lite/catalog
│   └── go.mod                      # deps: core, go-faster/yaml (delete custom marshaler)
│   ├── types.go
│   ├── registry.go
│   ├── schema.go
│   ├── asyncapi/                   # uses real YAML lib
│   ├── eventcatalog/
│   └── adapters/
│
├── middleware/                      # github.com/larsartmann/go-cqrs-lite/middleware
│   └── go.mod                      # deps: core
│   ├── logging.go
│   ├── metrics.go
│   ├── recovery.go
│   ├── retry.go
│   └── validation.go
│
├── xtypes/                         # github.com/larsartmann/go-cqrs-lite/xtypes
│   └── go.mod                      # deps: core
│
├── storage/                            # github.com/larsartmann/go-cqrs-lite/storage
│   └── go.mod                      # deps: core, sqlc-dev/sqlc, pgx/v5
│   └── sqlc.yaml                   # multi-engine: postgres, mysql, sqlite
│   └── sql/
│       ├── postgres/               # schema + queries
│       ├── mysql/                  # schema + queries (shared queries, engine-specific DDL)
│       └── sqlite/                 # schema + queries
│   └── internal/db/               # generated code (build-tagged per engine)
│       ├── postgres/
│       ├── mysql/
│       └── sqlite/
│   └── eventstore.go              # adapter: implements core/event.Store
│   └── migration.go               # schema management
├── nats/                           # github.com/larsartmann/go-cqrs-lite/nats
│   └── go.mod                      # deps: core, nats.go
│   └── eventbus.go
├── redis/                          # github.com/larsartmann/go-cqrs-lite/redis
│   └── go.mod                      # deps: core, go-redis
│   └── eventstore.go
│
└── examples/                       # separate modules, CI-tested
    ├── user/
    │   └── go.mod
    └── catalog/
        └── go.mod
```

## go.work

```go
go 1.26

use (
    ./core
    ./memory
    ./catalog
    ./middleware
    ./xtypes
    ./storage
    ./nats
    ./redis
    ./examples/user
    ./examples/catalog
)
```

## Dependency Graph

```
                    core (ulid + errors)
                   /  |  \  \  \  \  \
                  /   |   \  \  \  \  \
             memory  catalog  middleware  xtypes  storage  nats  redis
```

`*` core keeps cockroachdb/errors + oklog/ulid.
`storage` uses sqlc for type-safe, multi-engine code generation (postgres, mysql, sqlite).

## What Users Import

| Use Case | Import | Gets These Deps |
|---|---|---|
| CQRS types only | `core/...` | ulid, errors |
| + in-memory testing | `memory/...` | + core |
| + API docs | `catalog/...` | + core, go-faster/yaml |
|| + SQL store (postgres) | `storage` | + core, pgx, sqlc |
|| + SQL store (mysql) | `storage` | + core, database/sql |
|| + SQL store (sqlite) | `storage` | + core, database/sql |
|| + NATS bus | `nats` | + core, nats.go |
| Everything | all modules | all deps |

Nobody who just wants CQRS types pulls in pgx, nats, or a YAML library.

## What Gets Deleted

| What | Why |
|---|---|
| `catalog/yaml/` (~180 lines) | Replaced by `go-faster/yaml` in catalog's go.mod |
| `pkg/errors/` (~30 lines) | Dead code — never used anywhere |
| `event/memory_store.go` | Moves to `memory/store/` |
| `event/memory_bus.go` | Moves to `memory/bus/` |
| `event/memory_snapshot_store.go` | Moves to `memory/store/` |

## What Gets Fixed (Existing Issues)

| Issue | Fix |
|---|---|
| Query handler missing `context.Context` | Fix in `core/query/` during migration |
| `err113` sentinel errors | Fix in all modules |
| `marshalValue` complexity (14) | Delete the whole file, use go-faster/yaml |
| `catalog/adapters` coverage 66% | Add tests in `catalog/` module |
| Examples not CI-tested | Add to workflow via `go.work` |

## Migration Phases

### Phase 0: Preparation (no breaking changes) ✅

1. ~~Fix query handler signature to include `context.Context`~~ ✅
2. ~~Delete `pkg/errors/` (dead code)~~ ✅
3. ~~Replace custom YAML marshaler with `go-faster/yaml`~~ ✅ (done in prior commit 3c09f0b)
4. ~~Fix all `err113` linter warnings~~ ✅
5. ~~Ensure all tests pass clean~~ ✅

### Phase 1: Create go.work + move into subdirectories

1. Create `go.work` at root pointing at current single module
2. Verify everything still works (this is the safe checkpoint)
3. Create `core/` directory, move `command/`, `query/`, `event/`, `aggregate/`, `pkg/` into it
4. Update `core/go.mod` — module path becomes `github.com/larsartmann/go-cqrs-lite/core`,
   `go 1.25` minimum (required for subdirectory module root resolution)
5. Update all internal import paths
6. Add `go-import` meta tags for the new module paths
7. Run tests, fix until green

### Phase 2: Extract memory implementations

1. Create `memory/` with its own `go.mod`
2. Move MemoryStore, MemoryBus, MemorySnapshotStore from `core/event/` to `memory/`
3. `memory/go.mod` depends on `core`
4. Update `go.work`
5. Run tests, fix until green

### Phase 3: Extract catalog

1. Create `catalog/` with its own `go.mod`
2. Move all catalog code
3. Add `go-faster/yaml` dependency
4. Delete `catalog/yaml/` custom marshaler
5. Update `go.work`
6. Run tests, fix until green

### Phase 4: Extract middleware + xtypes

1. Same pattern — own go.mod, depends on core
2. Run tests, fix until green

### Phase 5: SQL store module (new)

1. Create `storage/` with its own `go.mod`
2. Add `sqlc.yaml` (based on template-sqlc pattern)
3. Write engine-specific schemas: `sql/postgres/schema/`, `sql/mysql/schema/`, `sql/sqlite/schema/`
4. Write shared event store queries: `sql/*/queries/` (Save, Load, LoadFromVersion, Delete, AppendBatch)
5. Run `sqlc generate` to produce type-safe Go code in `internal/db/`
6. Write `eventstore.go` adapter implementing `core/event.Store` using generated queries
7. Use build tags per engine (`postgres`, `mysql`, `sqlite`) so users only compile what they need
8. Add to go.work

### Phase 6: Message bus modules (new)

1. Create `nats/` — implement `event.Bus` backed by nats.go
2. Create `redis/` — implement `event.Store` backed by go-redis (optional, low priority)
3. Each has its own go.mod with only its backend dependency
4. Add to go.work

### Phase 6: Tag releases

1. Tag `core/v1.0.0`
2. Tag `memory/v1.0.0`
3. Tag `storage/v1.0.0`
4. Tag `catalog/v1.0.0`
5. Tag other modules as ready

## Module Path Convention

### Module Path Convention

Before Go 1.25, multi-module repos required either:
- Separate repos per module (fragmented)
- `replace` directives in every go.mod (fragile)
- Accepting that subdirectory modules couldn't resolve from their import path

Now, `go-import` meta tags map import paths to subdirectories natively.
Every module lives as a top-level directory with its own `go.mod`:

```
github.com/larsartmann/go-cqrs-lite/core
github.com/larsartmann/go-cqrs-lite/memory
github.com/larsartmann/go-cqrs-lite/catalog
github.com/larsartmann/go-cqrs-lite/middleware
github.com/larsartmann/go-cqrs-lite/xtypes
github.com/larsartmann/go-cqrs-lite/storage
github.com/larsartmann/go-cqrs-lite/nats
github.com/larsartmann/go-cqrs-lite/redis
```

Same repo, one source of truth, subdirectory resolution via Go 1.25.
This is the pattern used by OpenTelemetry and gRPC.

### Import Example

```go
import (
    "github.com/larsartmann/go-cqrs-lite/core/command"
    "github.com/larsartmann/go-cqrs-lite/core/event"
    "github.com/larsartmann/go-cqrs-lite/memory/store"
    "github.com/larsartmann/go-cqrs-lite/storage"          // SQL-backed event store
)
```

### go-import Configuration

For each module, add a `go-import` meta tag to the GitHub HTML (via `godoc.org`
or a custom landing page):

```html
<meta name="go-import" content="github.com/larsartmann/go-cqrs-lite/core mod https://github.com/larsartmann/go-cqrs-lite core">
<meta name="go-import" content="github.com/larsartmann/go-cqrs-lite/memory mod https://github.com/larsartmann/go-cqrs-lite memory">
<meta name="go-import" content="github.com/larsartmann/go-cqrs-lite/catalog mod https://github.com/larsartmann/go-cqrs-lite catalog">
<meta name="go-import" content="github.com/larsartmann/go-cqrs-lite/middleware mod https://github.com/larsartmann/go-cqrs-lite middleware">
<meta name="go-import" content="github.com/larsartmann/go-cqrs-lite/xtypes mod https://github.com/larsartmann/go-cqrs-lite xtypes">
<meta name="go-import" content="github.com/larsartmann/go-cqrs-lite/storage mod https://github.com/larsartmann/go-cqrs-lite storage">
<meta name="go-import" content="github.com/larsartmann/go-cqrs-lite/nats mod https://github.com/larsartmann/go-cqrs-lite nats">
<meta name="go-import" content="github.com/larsartmann/go-cqrs-lite/redis mod https://github.com/larsartmann/go-cqrs-lite redis">
```

This tells `go get` that `go-cqrs-lite/storage` lives in the `storage/` subdirectory of
the `go-cqrs-lite` repo — no separate repo needed.

## CI Changes

```yaml
# .github/workflows/test.yml
strategy:
  matrix:
    module:
      - core
      - memory
      - catalog
      - middleware
      - xtypes
      - storage
      - nats
      - redis

steps:
  - name: Test ${{ matrix.module }}
    run: cd ${{ matrix.module }} && go test ./... -race -count=1
```

Each module tests independently. CI catches cross-module breakage via go.work.

## Risks and Mitigations

| Risk | Mitigation |
|---|---|
| Import path churn for existing users | Provide a compatibility shim in root module that re-exports core types |
| Cross-module refactoring pain | go.work makes IDE navigation seamless — same experience as today |
| Version skew between modules | Use replace directives in go.work during dev; tag together initially |
| Lost git history from moves | Use `git mv` — GitHub tracks history across renames |
| Examples break | They're in go.work, tested in CI |

## ID Strategy: ULID

All IDs (EventID, AggregateID, CorrelationID, etc.) use **ULID** (`github.com/oklog/ulid`)
instead of UUID v4 (`github.com/google/uuid`).

**Why ULID for event sourcing:**

| Property | UUID v4 | ULID |
|---|---|---|
| Sortable | Random — no sort guarantee | Lexicographic (millisecond precision) |
| DB index inserts | Random page splits | Append-right (fast) |
| Timestamp extraction | No | Yes — from the ID itself |
| String length | 36 chars | 26 chars |
| Encoding | Hex with dashes | Crockford Base32 (no 0/O, 1/I/l) |
| Cross-language spec | RFC 4122 | Formal spec, 40+ implementations |
| Collision risk | Negligible | Negligible (80 random bits per ms) |

The killer property: time-sortable IDs mean event store indexes are append-only.
No page fragmentation from random UUID inserts.

## Open Questions

1. **Should core be truly zero-dep?** Currently depends on cockroachdb/errors + oklog/ulid. Could vendor ID generation and use stdlib errors. Tradeoff: lose branded error types and ULID's time-sortability + collision resistance.
2. **Should middleware/ be split further?** e.g., `middleware/tracing` with OTel dep vs `middleware/retry` with no deps. Low priority.
3. **Backend module priorities?** `storage/` (postgres primary, mysql/sqlite follow) is highest-value. NATS bus second. Redis optional.** Need to decide how to serve the meta tags — GitHub Pages, godoc.org, or a custom domain. GitHub Pages via a static `index.html` in the repo is simplest.
5. **sqlc query sharing?** Event store queries (Save, Load, LoadFromVersion, Delete) are identical across SQL engines. Consider sharing a single `queries/` dir with engine-specific overrides only for DDL differences.

## storage/ Module Design

The `storage/` module uses **sqlc** to generate type-safe Go code for PostgreSQL, MySQL, and SQLite
from a single `sqlc.yaml` config (based on the template-sqlc pattern).

### Why sqlc

| Approach | Problem |
|---|---|
| Hand-written SQL | Runtime errors, no type safety, string concatenation |
| ORM (GORM, Ent) | Abstraction leak, event store needs raw SQL control |
| **sqlc** | Write SQL, get type-safe Go. No runtime reflection. |

### Structure

```
sql/
├── sqlc.yaml                       # multi-engine config
├── storage/
│   ├── postgres/
│   │   ├── schema/
│   │   │   └── 001_events.sql      # CREATE TABLE events (...)
│   │   └── queries/
│   │       └── events.sql          # -- name: Save :exec
│   ├── mysql/
│   │   ├── schema/
│   │   │   └── 001_events.sql
│   │   └── queries/
│   │       └── events.sql
│   └── sqlite/
│       ├── schema/
│       │   └── 001_events.sql
│       └── queries/
│           └── events.sql
├── internal/db/                    # generated (committed, build-tagged)
│   ├── postgres/
│   │   ├── db.go
│   │   ├── models.go
│   │   └── querier.go
│   ├── mysql/
│   │   ├── db.go
│   │   ├── models.go
│   │   └── querier.go
│   └── sqlite/
│       ├── db.go
│       ├── models.go
│       └── querier.go
├── eventstore.go                   # implements core/event.Store
├── migration.go                    # schema creation/management
└── go.mod
```

### sqlc.yaml Key Settings

```yaml
sql:
  - name: "postgres"
    engine: "postgresql"
    queries: ["sql/postgres/queries"]
    schema: ["sql/postgres/schema"]
    gen:
      go:
        package: "postgres"
        out: "internal/db/postgres"
        sql_package: "pgx/v5"
        emit_interface: true
        emit_result_struct_pointers: true
        emit_empty_slices: true
        emit_sql_as_comment: true

  - name: "mysql"
    engine: "mysql"
    queries: ["sql/mysql/queries"]
    schema: ["sql/mysql/schema"]
    gen:
      go:
        package: "mysql"
        out: "internal/db/mysql"
        sql_package: "database/sql"
        emit_interface: true
        emit_result_struct_pointers: true
        emit_empty_slices: true

  - name: "sqlite"
    engine: "sqlite"
    queries: ["sql/sqlite/queries"]
    schema: ["sql/sqlite/schema"]
    gen:
      go:
        package: "sqlite"
        out: "internal/db/sqlite"
        sql_package: "database/sql"
        emit_interface: true
        emit_result_struct_pointers: true
        emit_empty_slices: true
```

### Event Store Schema (PostgreSQL)

```sql
CREATE TABLE events (
    event_id       TEXT NOT NULL PRIMARY KEY,  -- ULID, time-sortable
    event_type     TEXT NOT NULL,
    aggregate_id   TEXT NOT NULL,
    aggregate_type TEXT NOT NULL,
    version        INTEGER NOT NULL,
    payload        BYTEA,
    metadata       JSONB,
    occurred_at    TIMESTAMPTZ NOT NULL,
    UNIQUE (aggregate_type, aggregate_id, version)
);

CREATE INDEX idx_events_aggregate ON events (aggregate_type, aggregate_id, version);
```

### Build Tags

Users compile only the engine they need:

```bash
# PostgreSQL (default)
go build -tags postgres ./...

# MySQL
go build -tags mysql ./...

# SQLite
go build -tags sqlite ./...
```

The `eventstore.go` adapter uses build-tagged files:
```
eventstore_postgres.go   //go:build postgres
eventstore_mysql.go      //go:build mysql
eventstore_sqlite.go     //go:build sqlite
```
