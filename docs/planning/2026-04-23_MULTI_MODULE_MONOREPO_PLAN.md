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

## The Four Storage Concerns

CQRS event sourcing has four distinct storage needs. Each has different access patterns,
different backends, and different scaling characteristics. They must be independent modules:

| Concern | Access Pattern | Module | Backend |
|---|---|---|---|
| **Event Store** | Append-only, optimistic concurrency, ordered scans | `storage/` | PostgreSQL, MySQL, SQLite (sqlc) |
| **PubSub** | Fan-out, consumer groups, at-least-once delivery | `watermill/` | Redis Streams, NATS, Kafka, Google PubSub |
| **Projections** | Subscribe to events → write to query-optimized tables | `projection/` | Any SQL database |
| **Snapshots** | Key-value by aggregate ID, infrequent writes | `snapshot/` | Any SQL database, Redis |

Each is independently replaceable. Use PostgreSQL for event store, Redis Streams for pub/sub,
SQLite for projections, Redis for snapshots? Fine. Swap one without touching the others.

## Module Layout

```
go-cqrs-lite/
├── go.work
│
├── core/                           # github.com/larsartmann/go-cqrs-lite/core
│   └── go.mod                      # deps: cockroachdb/errors, oklog/ulid
│   ├── command/                    # command dispatch, types, handler interface
│   ├── query/                      # query dispatch, types (WITH context.Context)
│   ├── event/                      # interfaces: Store, Bus, Streamer, Codec
│   ├── aggregate/                  # Root, Repository interfaces
│   ├── projection/                 # Projection, Handler interfaces
│   ├── snapshot/                   # SnapshotStore interface
│   ├── upcasting/                  # Upcaster interface, version registry
│   └── pkg/
│       └── id/                     # branded IDs (ULID-backed)
│
├── memory/                         # github.com/larsartmann/go-cqrs-lite/memory
│   └── go.mod                      # deps: core
│   ├── store.go                    # MemoryStore
│   ├── bus.go                      # MemoryBus
│   ├── snapshot.go                 # MemorySnapshotStore
│   └── projection.go               # MemoryProjectionStore (for testing)
│
├── storage/                        # github.com/larsartmann/go-cqrs-lite/storage
│   └── go.mod                      # deps: core, pgx/v5, sqlc
│   └── sqlc.yaml                   # multi-engine: postgres, mysql, sqlite
│   └── sql/
│       ├── postgres/               # schema + queries
│       ├── mysql/                  # schema + queries (shared queries, engine-specific DDL)
│       └── sqlite/                 # schema + queries
│   └── internal/db/               # generated code (build-tagged per engine)
│       ├── postgres/
│       ├── mysql/
│       └── sqlite/
│   └── eventstore.go              # implements core/event.Store
│   └── outbox.go                  # transactional outbox pattern
│   └── migration.go               # schema management
│
├── watermill/                      # github.com/larsartmann/go-cqrs-lite/watermill
│   └── go.mod                      # deps: core, watermill
│   └── bus.go                      # implements core/event.Bus via Watermill
│   └── config.go                   # backend configuration helpers
│
├── projection/                     # github.com/larsartmann/go-cqrs-lite/projection
│   └── go.mod                      # deps: core, storage, samber/ro
│   └── runner.go                   # subscribes to events, dispatches to handlers
│   └── handler.go                  # Handler interface, checkpoint tracking
│   └── checkpoint.go               # stores projection position (SQL-backed)
│   └── internal/stream/            # samber/ro operators (encapsulated, not public)
│       ├── pipeline.go             # ro.Pipe wrappers for event stream processing
│       ├── filters.go              # FilterType, FilterAggregate via ro.Filter
│       └── windows.go              # time-windowed aggregation via ro.BufferWhen
│
├── snapshot/                       # github.com/larsartmann/go-cqrs-lite/snapshot
│   └── go.mod                      # deps: core, storage
│   └── store.go                    # implements core/snapshot.SnapshotStore
│   └── strategy.go                 # snapshot strategies (every N events, time-based)
│
├── catalog/                        # github.com/larsartmann/go-cqrs-lite/catalog
│   └── go.mod                      # deps: core, go-faster/yaml
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
├── testutil/                       # github.com/larsartmann/go-cqrs-lite/testutil
│   └── go.mod                      # deps: core, memory
│   ├── aggregate_tester.go         # given events → when command → then events
│   ├── projection_tester.go        # given events → then read model state
│   └── bus_tester.go               # publish → assert handler called
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
    ./storage
    ./watermill
    ./projection
    ./snapshot
    ./catalog
    ./middleware
    ./xtypes
    ./testutil
    ./examples/user
    ./examples/catalog
)
```

## Dependency Graph

```
                              core (ulid + errors)
                             / | \  \  \  \  \  \
                            /  |  \  \  \  \  \  \
                       memory  catalog  middleware  xtypes  testutil
                                  |
                          ┌───┬───┴───┬────────┐
                          │   │       │        │
                      storage  watermill  projection  snapshot
                          │              │        │
                          └──────────────┴────────┘
                              (projection + snapshot depend on storage)

projection ──→ samber/ro (internal, encapsulated behind CQRS interfaces)
```

All storage modules depend on `core` interfaces, **not on each other**.
`projection/` and `snapshot/` may optionally depend on `storage/` for SQL-backed persistence.

## What Users Import

| Use Case | Import | Gets These Deps |
|---|---|---|
| CQRS types only | `core/...` | ulid, errors |
| + in-memory testing | `memory/...` | + core |
| + SQL event store (postgres) | `storage` | + core, pgx, sqlc |
| + SQL event store (mysql/sqlite) | `storage` | + core, database/sql |
| + pub/sub (any backend) | `watermill` | + core, watermill |
| + projections | `projection` | + core, storage, samber/ro |
| + snapshots | `snapshot` | + core, storage |
| + API docs | `catalog/...` | + core, go-faster/yaml |
| + test utilities | `testutil` | + core, memory |
| Everything | all modules | all deps |

Nobody who just wants CQRS types pulls in pgx, watermill, or a YAML library.

## What Gets Deleted

| What | Why |
|---|---|
| `catalog/yaml/` (~180 lines) | Replaced by `go-faster/yaml` in catalog's go.mod |
| `pkg/errors/` (~30 lines) | Dead code — never used anywhere |
| `event/memory_store.go` | Moves to `memory/store.go` |
| `event/memory_bus.go` | Moves to `memory/bus.go` |
| `event/memory_snapshot_store.go` | Moves to `memory/snapshot.go` |

## What Gets Fixed (Existing Issues)

| Issue | Fix |
|---|---|
| Query handler missing `context.Context` | Fix in `core/query/` during migration |
| `err113` sentinel errors | Fix in all modules |
| `marshalValue` complexity (14) | Delete the whole file, use go-faster/yaml |
| `catalog/adapters` coverage 66% | Add tests in `catalog/` module |
| Examples not CI-tested | Add to workflow via `go.work` |

## Migration Phases

### Phase 0: Preparation (no breaking changes)

1. Fix query handler signature to include `context.Context`
2. Delete `pkg/errors/` (dead code)
3. Replace custom YAML marshaler with `go-faster/yaml`
4. Fix all `err113` linter warnings
5. Ensure all tests pass clean

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

### Phase 4: Extract middleware + xtypes ✅ DONE

1. Same pattern — own go.mod, depends on core
2. Middleware: inlined testhelpers (cross-module internal packages not importable)
3. Xtypes: clean extraction, no internal deps
4. Core go.mod cleaned: removed go-faster/yaml + transitive deps
5. Empty dirs removed: core/middleware/, core/xtypes/, core/catalog/
6. All tests green across all modules

### Phase 5: Storage module (event store)

1. Create `storage/` with its own `go.mod`
2. Add `sqlc.yaml` (based on template-sqlc pattern)
3. Write engine-specific schemas: `sql/postgres/schema/`, `sql/mysql/schema/`, `sql/sqlite/schema/`
4. Write shared event store queries: Save, Load, LoadFromVersion, Delete, AppendBatch
5. Run `sqlc generate` to produce type-safe Go code in `internal/db/`
6. Write `eventstore.go` adapter implementing `core/event.Store`
7. Implement outbox pattern in `outbox.go` (same-transaction event write + relay)
8. Use build tags per engine (`postgres`, `mysql`, `sqlite`)
9. Add to go.work

### Phase 6: Watermill module (pub/sub)

1. Create `watermill/` with its own `go.mod`
2. Implement `core/event.Bus` via Watermill's Publisher/Subscriber interface
3. Support Redis Streams, NATS, Kafka, Google Cloud PubSub — all free from one module
4. Add to go.work

### Phase 7: Projection module (read models)

1. Create `projection/` with its own `go.mod`
2. Define `Projection` interface in `core/projection/`
3. Implement `Runner` that subscribes to events and dispatches to projection handlers
4. Implement checkpoint tracking (SQL-backed, knows where each projection left off)
5. Add to go.work

### Phase 8: Snapshot module

1. Create `snapshot/` with its own `go.mod`
2. Move `core/snapshot/` interface from core
3. Implement SQL-backed `SnapshotStore` via `storage/`
4. Implement strategies: every N events, time-based, on-demand
5. Add to go.work

### Phase 9: Test utilities module

1. Create `testutil/` with its own `go.mod`
2. Implement `AggregateTester` — given events, when command, then events
3. Implement `ProjectionTester` — given events, then read model state
4. Implement `BusTester` — publish, assert handler called
5. Add to go.work

### Phase 10: Tag releases

1. Tag `core/v1.0.0`
2. Tag `memory/v1.0.0`
3. Tag `storage/v1.0.0`
4. Tag `watermill/v1.0.0`
5. Tag `projection/v1.0.0`
6. Tag `snapshot/v1.0.0`
7. Tag `catalog/v1.0.0`
8. Tag other modules as ready

## Module Path Convention

Before Go 1.25, multi-module repos required either:
- Separate repos per module (fragmented)
- `replace` directives in every go.mod (fragile)
- Accepting that subdirectory modules couldn't resolve from their import path

Now, `go-import` meta tags map import paths to subdirectories natively.
Every module lives as a top-level directory with its own `go.mod`:

```
github.com/larsartmann/go-cqrs-lite/core
github.com/larsartmann/go-cqrs-lite/memory
github.com/larsartmann/go-cqrs-lite/storage
github.com/larsartmann/go-cqrs-lite/watermill
github.com/larsartmann/go-cqrs-lite/projection
github.com/larsartmann/go-cqrs-lite/snapshot
github.com/larsartmann/go-cqrs-lite/catalog
github.com/larsartmann/go-cqrs-lite/middleware
github.com/larsartmann/go-cqrs-lite/xtypes
github.com/larsartmann/go-cqrs-lite/testutil
```

Same repo, one source of truth, subdirectory resolution via Go 1.25.
This is the pattern used by OpenTelemetry and gRPC.

### Import Example

```go
import (
    "github.com/larsartmann/go-cqrs-lite/core/command"
    "github.com/larsartmann/go-cqrs-lite/core/event"
    "github.com/larsartmann/go-cqrs-lite/memory"               // testing
    "github.com/larsartmann/go-cqrs-lite/storage"              // SQL event store
    "github.com/larsartmann/go-cqrs-lite/watermill"            // pub/sub
    "github.com/larsartmann/go-cqrs-lite/projection"           // read models
)
```

### go-import Configuration

For each module, add a `go-import` meta tag to the GitHub HTML (via `godoc.org`
or a custom landing page):

```html
<meta name="go-import" content="github.com/larsartmann/go-cqrs-lite/core mod https://github.com/larsartmann/go-cqrs-lite core">
<meta name="go-import" content="github.com/larsartmann/go-cqrs-lite/memory mod https://github.com/larsartmann/go-cqrs-lite memory">
<meta name="go-import" content="github.com/larsartmann/go-cqrs-lite/storage mod https://github.com/larsartmann/go-cqrs-lite storage">
<meta name="go-import" content="github.com/larsartmann/go-cqrs-lite/watermill mod https://github.com/larsartmann/go-cqrs-lite watermill">
<meta name="go-import" content="github.com/larsartmann/go-cqrs-lite/projection mod https://github.com/larsartmann/go-cqrs-lite projection">
<meta name="go-import" content="github.com/larsartmann/go-cqrs-lite/snapshot mod https://github.com/larsartmann/go-cqrs-lite snapshot">
<meta name="go-import" content="github.com/larsartmann/go-cqrs-lite/catalog mod https://github.com/larsartmann/go-cqrs-lite catalog">
<meta name="go-import" content="github.com/larsartmann/go-cqrs-lite/middleware mod https://github.com/larsartmann/go-cqrs-lite middleware">
<meta name="go-import" content="github.com/larsartmann/go-cqrs-lite/xtypes mod https://github.com/larsartmann/go-cqrs-lite xtypes">
<meta name="go-import" content="github.com/larsartmann/go-cqrs-lite/testutil mod https://github.com/larsartmann/go-cqrs-lite testutil">
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
      - storage
      - watermill
      - projection
      - snapshot
      - catalog
      - middleware
      - xtypes
      - testutil

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
3. **Module priorities?** `storage/` (postgres primary) is highest-value. `watermill/` second. `projection/` third. Rest follow.
4. **go-import hosting strategy?** Need to decide how to serve the meta tags — GitHub Pages, godoc.org, or a custom domain. GitHub Pages via a static `index.html` in the repo is simplest.
5. **sqlc query sharing?** Event store queries (Save, Load, LoadFromVersion, Delete) are identical across SQL engines. Consider sharing a single `queries/` dir with engine-specific overrides only for DDL differences.
6. **Event Codec?** Payload is `[]byte`. Need a pluggable `Codec` interface in core (JSON, protobuf, msgpack). Should this be in `core/event/` or its own module?
7. **Event Upcasting?** Events evolve (`UserCreatedV1` → `V2`). Need an upcasting mechanism. Interface in `core/upcasting/`, implementation in storage module?
8. **Schema migration tool?** `storage/migration.go` needs a concrete strategy. golang-migrate? goose? Raw SQL?

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
storage/
├── sqlc.yaml                       # multi-engine config
├── sql/
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
├── outbox.go                       # transactional outbox pattern
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

### Outbox Schema

```sql
CREATE TABLE outbox (
    id             TEXT NOT NULL PRIMARY KEY,  -- ULID
    event_id       TEXT NOT NULL,
    aggregate_type TEXT NOT NULL,
    aggregate_id   TEXT NOT NULL,
    event_type     TEXT NOT NULL,
    payload        BYTEA,
    metadata       JSONB,
    occurred_at    TIMESTAMPTZ NOT NULL,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    published      BOOLEAN NOT NULL DEFAULT FALSE
);

CREATE INDEX idx_outbox_unpublished ON outbox (created_at) WHERE NOT published;
```

Events are written to both `events` and `outbox` in the same transaction.
A background relay publishes from outbox to the message bus, then marks published.

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

## watermill/ Module Design

A single module that provides `event.Bus` via Watermill, giving users
access to all pub/sub backends without maintaining N separate modules.

### Supported Backends (all free from Watermill)

- Redis Streams
- NATS / NATS JetStream
- Apache Kafka
- Google Cloud PubSub
- AMQP (RabbitMQ)
- SQL (fallback)
- HTTP (testing)
- Go channel (in-process — but `memory/` is better for this)

### Why Watermill

| Approach | Problem |
|---|---|
| Hand-rolled per backend | N modules to maintain, N sets of bugs |
| **Watermill** | Battle-tested, all backends, maintained by community |
| NATS-only | Locks users into one transport |

### Tradeoff

Watermill is a heavier dependency than hand-rolling one backend.
But one dependency for all backends > N dependencies for N backends.
Users who don't want Watermill can use `memory/` for testing or implement `event.Bus` themselves.

## projection/ Module Design

Builds read models by subscribing to the event stream and writing to
query-optimized tables. This is the "Q" in CQRS — currently missing entirely.

Uses **samber/ro** internally as the stream processing engine. Users never see
Observable types — they just call `projector.On()`. See `docs/planning/2026-04-24_SAMBER_RO_PRO_CONTRA.md`
for the rationale behind encapsulating ro as an implementation detail.

### Core Concepts

- **Projection** — a function that handles events and updates a read model
- **Checkpoint** — tracks which events each projection has processed
- **Runner** — subscribes to events, dispatches to projections, updates checkpoints

### Flow

```
Event Store → Runner → Projection Handler → Read Model Table
                    └→ Checkpoint Store (position tracking)
```

### Example

```go
// User defines a projection:
projector.On("user.created", func(ctx context.Context, evt event.Event) error {
    return updateUserReadModel(ctx, evt)
})

projector.On("user.email_changed", func(ctx context.Context, evt event.Event) error {
    return updateUserReadModel(ctx, evt)
})

// Runner handles subscription, checkpointing, and error recovery
runner := projection.NewRunner(store, bus, projector, checkpointStore)
go runner.Run(ctx)
```

## snapshot/ Module Design

Materializes aggregate state to avoid replaying thousands of events on every load.

### Core Concepts

- **SnapshotStore** — saves/loads serialized aggregate state by ID + version
- **Strategy** — decides when to snapshot (every N events, time-based, on-demand)

### Flow

```
Load aggregate:
  1. Check SnapshotStore for latest snapshot
  2. Load events after snapshot version
  3. Apply only the delta events

Save aggregate:
  1. Persist events to EventStore (as usual)
  2. If strategy says "snapshot now", save current aggregate state
```

### Schema

```sql
CREATE TABLE snapshots (
    aggregate_type TEXT NOT NULL,
    aggregate_id   TEXT NOT NULL,
    version        INTEGER NOT NULL,
    state          BYTEA NOT NULL,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (aggregate_type, aggregate_id)
);
```

## testutil/ Module Design

Testing utilities that every user of the SDK would otherwise build themselves.

### AggregateTester

```go
tester := testutil.NewAggregateTester(store, bus)
tester.Given(events...)                    // seed history
result := tester.When(createUserCmd)       // execute command
result.ThenShouldEmit("user.created")      // assert events
result.ThenShouldSucceed()                 // assert no error
```

### ProjectionTester

```go
tester := testutil.NewProjectionTester(handler)
tester.GivenEvents(events...)              // feed events
tester.ThenReadModel(userID, &expected)    // assert read model state
```
