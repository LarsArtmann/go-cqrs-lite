# Storage Module: First-Principles Analysis

> **Date:** 2026-05-29 | **Scope:** `storage/`, `storage/sql/`, `pebble/`, `turso/`, `core/event/` interfaces
>
> **RESOLUTION STATUS (2026-06-28):** Most findings in this document have been
> addressed. Key updates since this analysis was written:
>
> - **§3.1 Split Brain (`storage/` root vs `storage/sql/`):** RESOLVED. The
>   root package now imports `storage/sql` for all SQL infrastructure. No
>   duplicate dialect/helpers/reconstruction code remains.
> - **§3.2 `AggregateProjection`/`SQLAggregateReader` not dialect-aware:** RESOLVED.
> - **§3.3 No Schema Migration System:** PARTIALLY addressed via embedded DDL
>   in `storage/migrations/` + `RelationalSchema.Migrate()`.
> - **§3.4 Pebble Module Incomplete:** RESOLVED. Pebble now has
>   `SnapshotStore`, `CheckpointStore`, `CommandStore`, `QueryStore`, and
>   `ReadModels` (KV adapter) via `PebbleBackend`.
> - **§3.5 STORAGE_GUIDE.md is Stale:** RESOLVED (rewritten 2026-06-28).
> - **§3.8 No DLQ for Outbox:** Still open. The retry middleware returns
>   `ErrRetryExhausted` and the message is lost — no dead-letter quarantine.

---

## Methodology

Sources studied:

1. `/home/lars/projects/reports/docs/100 Things I hate in modern Software Development.md` — design philosophy and pain points
2. `/home/lars/projects/reports/docs/Perfect Software Architecture for Business Applications.md` — target architecture vision
3. All `.go` files in `storage/`, `storage/sql/`, `pebble/`, `turso/` (~4,500 LOC total)
4. All core event interfaces in `core/event/store.go`, `snapshot.go`, `outbox.go`, `checkpoint.go`, `bus.go`, `stream.go`
5. All 8 ADR documents in `docs/adr/`
6. `docs/STORAGE_GUIDE.md`, `docs/architecture/module-graph.d2`
7. Go CQRS/ES ecosystem survey: `looplab/eventhorizon`, `thefabric-io/eventsourcing`, `j5ik2o/event-store-adapter-go`, `global-soft-ba/go-eventstore`

---

## 1. Philosophy-to-Storage Mapping

What the design philosophy documents demand from storage, and how well the current module delivers:

| #   | Principle (from source docs)                                | Storage Implication                                       | Current Status                                                                                                |
| --- | ----------------------------------------------------------- | --------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------- |
| 1   | **Choose persistence at RUNTIME, not build time** (Hate #4) | Dialect/backend swappable without recompilation           | **GOOD** — `NewXxxWithDialect(db, dialect)` escape hatch; `SQLiteDialect`/`PostgresDialect` runtime selection |
| 2   | **Adaptive I/O: HDD vs SSD vs NVMe** (Hate #21)             | Configurable flush/durability strategies                  | **MISSING** — fixed behavior per backend; Pebble has `WithAsyncWrites()` but no general pattern               |
| 3   | **NEVER LOSE DATA** (Perfect #19)                           | Transactional integrity, outbox, durable writes           | **GOOD** — `SaveWithOutbox` atomic TX, optimistic concurrency, tombstone over delete                          |
| 4   | **Data Store aware but independent** (Perfect R2 #8)        | Clean abstraction boundary; same API, any backend         | **EXCELLENT** — ISP-split interfaces (Sink/Source/Journal/Seekable) + Dialect abstraction                     |
| 5   | **Event Sourcing** (Perfect #3)                             | Immutable append-only store as foundation                 | **EXCELLENT** — no Delete, no mutation, version-ordered streams                                               |
| 6   | **Time-Travel + What-If** (Perfect #41, R4 #1)              | `LoadToVersion`, `LoadToTimestamp`, retroactive scenarios | **GOOD** — `LoadToVersion` + `LoadToTimestamp` cover 95%; no bi-temporal model                                |
| 7   | **Data locality** (Perfect #26)                             | Embedded (Pebble/SQLite) vs remote (PG/Turso) choice      | **GOOD** — 4 backends cover the spectrum; Pebble gaps limit full locality                                     |
| 8   | **Local first** (Perfect #27)                               | SQLite/Pebble as first-class citizens                     | **PARTIAL** — SQLite is first-class; Pebble incomplete (no Snapshot/Checkpoint/Outbox)                        |
| 9   | **Plugin system** (Perfect #2)                              | Storage backends as pluggable modules                     | **GOOD** — `Dialect` interface + `WithDialect` allow custom backends                                          |
| 10  | **Materialized Views / Projections** (Perfect #43)          | Checkpoint store, aggregate reader                        | **GOOD** — `SQLCheckpointStore`, `AggregateProjection`, `SQLAggregateReader` exist                            |
| 11  | **OTEL observability** (Perfect #35)                        | Tracing on every operation                                | **GOOD** — `storage/otel.go` + `storage/sql/otel.go`; spans on Save                                           |
| 12  | **Snapshots** (Perfect #42)                                 | Snapshot store for fast aggregate hydration               | **GOOD** — `SQLSnapshotStore` with Save/Load/LoadAtVersion/Delete                                             |
| 13  | **Dead-Letter Queues** (Perfect #39)                        | Outbox pattern with retry/error tracking                  | **PARTIAL** — Outbox exists but no DLQ; failed events stay pending forever                                    |
| 14  | **Idempotency** (Perfect #40)                               | Optimistic concurrency + idempotent outbox                | **GOOD** — version check on Save, outbox append is idempotent                                                 |
| 15  | **Smart Retry** (Perfect #25)                               | Retry middleware on transient failures                    | **GOOD** — `middleware.Retry` exists; storage itself doesn't retry                                            |
| 16  | **Errors as Values** (Perfect #20)                          | Typed errors, no panics                                   | **EXCELLENT** — 5-family error taxonomy, sentinel errors, `%w` wrapping                                       |
| 17  | **Strong types** (Hate #24)                                 | No `any`, unsigned for non-negative                       | **GOOD** — Version is `uint64`; `dialect.go` uses `any` for SQL interop (unavoidable)                         |
| 18  | **No MEGA files** (Hate #44)                                | <250 lines per file                                       | **GOOD** — most files under 250 lines; `snapshot.go` at 251 is borderline                                     |
| 19  | **Caching on every layer** (Perfect #65)                    | In-memory caching of snapshots/checkpoints                | **MISSING** — no caching layer anywhere in storage                                                            |
| 20  | **Compression** (Perfect R2 #4)                             | Payload compression in storage                            | **MISSING** — no compression; JSON metadata and raw bytes stored as-is                                        |
| 21  | **Automated docs** (Perfect #48)                            | Schema docs, AsyncAPI, auto-generated                     | **GOOD** — `catalog/` module handles this; storage just provides data                                         |

---

## 2. Current Architecture Map

### Module Inventory

| Module         | LOC    | Interfaces Implemented                                                                                                    | Backend            |
| -------------- | ------ | ------------------------------------------------------------------------------------------------------------------------- | ------------------ |
| `storage/`     | ~3,200 | Store, Journal, SeekableJournal, BackwardsSource, StreamLoader, TransactionalSink, Outbox, SnapshotStore, CheckpointStore | PostgreSQL, SQLite |
| `storage/sql/` | ~800   | _(shared infrastructure, no interfaces)_                                                                                  | —                  |
| `pebble/`      | ~600   | Store only                                                                                                                | Pebble KV          |
| `turso/`       | ~200   | _(delegates to storage/)_                                                                                                 | Turso              |
| `memory/`      | ~400   | Store, Bus, SnapshotStore                                                                                                 | In-memory          |

### Interface Coverage Matrix

| Interface                                                             | `storage/` (SQL) | `pebble/` | `memory/` | `turso/` (via SQL) |
| --------------------------------------------------------------------- | ---------------- | --------- | --------- | ------------------ |
| `EventSink` (Save, AppendBatch)                                       | ✅               | ✅        | ✅        | ✅                 |
| `EventSource` (Load, LoadFromVersion, LoadToVersion, LoadToTimestamp) | ✅               | ✅        | ✅        | ✅                 |
| `Store` (Sink + Source)                                               | ✅               | ✅        | ✅        | ✅                 |
| `Journal` (ReadAll)                                                   | ✅               | ❌        | ❌        | ✅                 |
| `SeekableJournal` (ReadFrom)                                          | ✅               | ❌        | ❌        | ✅                 |
| `BackwardsSource` (LoadBackwards)                                     | ✅               | ❌        | ❌        | ✅                 |
| `StreamLoader` (LoadStream)                                           | ✅               | ❌        | ❌        | ✅                 |
| `TransactionalSink` (SaveWithOutbox)                                  | ✅               | ❌        | ❌        | ✅                 |
| `Outbox` (Append, PollPending, Ack)                                   | ✅               | ❌        | ❌        | ✅                 |
| `SnapshotStore` (Save, Load, LoadAtVersion, Delete)                   | ✅               | ❌        | ✅        | ✅                 |
| `CheckpointStore` (Load, Save)                                        | ✅               | ❌        | ❌        | ✅                 |

### Schema (4 tables, dual DDL)

| Table         | Columns                                                                                                           | PG Types                                        | SQLite Types           |
| ------------- | ----------------------------------------------------------------------------------------------------------------- | ----------------------------------------------- | ---------------------- |
| `events`      | id, event_type, aggregate_type, aggregate_id, version, schema_version, payload, metadata, occurred_at, created_at | `BYTEA`, `JSONB`, `TIMESTAMPTZ`, `VARCHAR(255)` | `BLOB`, `TEXT`, `TEXT` |
| `snapshots`   | aggregate_type, aggregate_id, version, state, created_at                                                          | `JSONB` state                                   | `BLOB` state           |
| `checkpoints` | projection_name (PK), event_id, processed_at                                                                      | `TIMESTAMP`                                     | `TEXT`                 |
| `outbox`      | id (PK), status, events (JSON), created_at                                                                        | `JSONB` events                                  | `TEXT` events          |

**Indexes on `events`:** `(aggregate_type, aggregate_id)`, `(event_type)`, `(occurred_at)`, `(aggregate_type, aggregate_id, occurred_at)`

---

## 3. Issues Found

### Critical

#### 3.1. Split Brain: `storage/` root vs `storage/sql/`

The `storage/sql/` sub-package was started as an extraction of shared SQL infrastructure but was never completed. Both locations contain **near-identical** code:

| File                      | `storage/` (root)                     | `storage/sql/`                  |
| ------------------------- | ------------------------------------- | ------------------------------- |
| Dialect interface + impls | `dialect.go` (190 lines)              | `dialect.go` (190 lines)        |
| Base struct               | `sql_base.go` (21 lines)              | `base.go` (25 lines)            |
| Error sentinels           | `errors.go` (53 lines)                | `errors.go` (56 lines)          |
| Table constants           | `tables.go` (9 lines)                 | `tables.go` (13 lines)          |
| SQL helpers               | `sql_helpers.go` (234 lines)          | `helpers.go` (229 lines)        |
| Event reconstruction      | `event_reconstruction.go` (182 lines) | `reconstruction.go` (185 lines) |
| OTEL helpers              | `otel.go` (36 lines)                  | `otel.go` (38 lines)            |
| SQLite timestamp parsing  | `sqlite_helpers.go` (partial)         | `sqlite.go` (35 lines)          |

**Root package uses private versions** (`sqlBase`, `placeholders`, `reconstructEvent`). **`sql/` has exported equivalents** (`Base`, `Placeholders`, `ReconstructEvent`). Neither imports from the other — they are fully independent copies.

**Impact:** Every bug fix must be applied twice. Every feature addition must be implemented twice. A divergence in behavior between the two would be extremely hard to detect.

#### 3.2. `AggregateProjection` and `SQLAggregateReader` Not Dialect-Aware

Both files use hardcoded `?` placeholders in their SQL queries:

- `storage/aggregate_projection.go` — `INSERT OR REPLACE` + `DELETE` with `?`
- `storage/sql_aggregate_reader.go` — `SELECT` with `?`

This means **PostgreSQL is broken** for aggregate listing. These two components are the only ones in the entire module that bypass the Dialect abstraction.

### High

#### 3.3. No Schema Migration System

Only `InitSchema` (CREATE IF NOT EXISTS) exists. When the schema evolves (new columns, index changes, type changes), there is no upgrade path. Consumers must manually alter their tables or drop and recreate.

Real-world scenario: Adding a `correlation_id` column to the `events` table for cross-aggregate tracing requires every consumer to write their own migration.

#### 3.4. Pebble Module Incomplete

Pebble only implements `event.Store`. Missing:

- **SnapshotStore** — trivial to add (single key per aggregate: `cqrs_snapshot:{type}:{id}`)
- **CheckpointStore** — trivial (single key per projection: `cqrs_checkpoint:{name}`)
- **Outbox** — moderate complexity (scan pending keys, atomic ack)

Without these, a consumer using Pebble for event storage must still bring in SQLite/PG for snapshots, checkpoints, and outbox — defeating the "single binary, single process" use case.

#### 3.5. STORAGE_GUIDE.md is Stale

References APIs that don't exist or have changed:

- `storage.SagaSchema()` / `storage.SQLiteSagaSchema()` — no saga schema exists
- `storage.OpenPebbleEventStore()` — Pebble is now a separate module
- `storage.OpenTurso()` — Turso connector is in a separate module
- Constructor signatures don't match current API

### Medium

#### 3.6. No Configurable Durability Strategy

From Hate #21: "Writing to DISK in randomly decided intervals... the dev needs to figure this out at compile time."

Current behavior is fixed per backend:

- PostgreSQL: `synchronous_commit=on` (consumer must tune)
- SQLite: `PRAGMA synchronous=FULL` (set once at init, no runtime control)
- Pebble: `WithAsyncWrites()` option exists but is the only knob

There's no unified `Durability` abstraction across backends.

#### 3.7. Outbox Stores Full Event JSON

The `outbox` table stores a complete JSON serialization of all events in each row. For a system processing 10K events/second, this means events are stored twice: once in `events` table and once in `outbox` table.

A reference-based approach (storing only event IDs in outbox, joining back to `events` on poll) would cut outbox storage by ~60-80%.

#### 3.8. No DLQ for Outbox

From Perfect #39: "Dead-Letter Queues." The `OutboxPoller` retries forever on failure. There is no max-retry count, no DLQ, no way to inspect or retry individual failed entries.

#### 3.9. No Caching Layer

From Perfect #65: "Smart Caches on every layer." No in-memory caching exists for:

- Snapshots (frequently loaded for aggregate hydration)
- Checkpoints (read on every projection tick)
- Aggregate listing queries

---

## 4. Ecosystem Comparison

### How go-cqrs-lite Compares to Go CQRS/ES Libraries

| Feature                          | go-cqrs-lite                                       | looplab/eventhorizon (1.7k★)                          | thefabric-io/eventsourcing  | global-soft-ba/go-eventstore     |
| -------------------------------- | -------------------------------------------------- | ----------------------------------------------------- | --------------------------- | -------------------------------- |
| **Storage backends**             | PG, SQLite, Pebble, Turso                          | Memory, MongoDB, PG (community), DynamoDB (community) | PostgreSQL only             | PG, Memory, Redis, OpenSearch    |
| **Interface segregation**        | ✅ 7 focused interfaces (Sink/Source/Journal/etc.) | ❌ Single `EventStore` interface                      | ❌ Single `Store` interface | ❌ Single `EventStore` interface |
| **Dialect abstraction**          | ✅ `Dialect` interface                             | ❌ Per-backend implementations                        | ❌ PG-only                  | ❌ Per-backend implementations   |
| **Transactional outbox**         | ✅ Built-in, atomic                                | ❌ External middleware                                | ❌ Not built-in             | ❌ Not built-in                  |
| **Snapshots**                    | ✅ Full CRUD                                       | ✅ Yes                                                | ✅ Yes                      | ✅ Yes                           |
| **Projection checkpoints**       | ✅ Built-in                                        | ✅ Yes                                                | ✅ Advanced offset mgmt     | ✅ Yes                           |
| **Time-travel queries**          | ✅ LoadToVersion + LoadToTimestamp                 | ❌ Limited                                            | ❌ Not visible              | ✅ Bi-temporal (AsAt/AsOf)       |
| **Backwards loading**            | ✅ `LoadBackwards`                                 | ❌ No                                                 | ❌ No                       | ❌ No                            |
| **Streaming reads**              | ✅ `EventStream` cursor                            | ❌ No                                                 | ❌ No                       | ❌ No                            |
| **Event signing**                | ✅ HMAC + Ed25519                                  | ❌ No                                                 | ❌ No                       | ❌ No                            |
| **OpenTelemetry**                | ✅ Built-in                                        | ✅ Via middleware                                     | ❌ Not visible              | ❌ Not visible                   |
| **Event upcasting**              | ✅ `VersionedStore` decorator                      | ✅ Yes                                                | ✅ Yes                      | ✅ Yes                           |
| **Branded IDs**                  | ✅ `id.Of[T]`                                      | ❌ `uuid.UUID`                                        | ❌ String IDs               | ❌ String IDs                    |
| **Embedded/local-first**         | ✅ SQLite + Pebble                                 | ❌ No embedded option                                 | ❌ No                       | ❌ No                            |
| **Code generation**              | ✅ `cqrs-gen`                                      | ❌ No                                                 | ❌ No                       | ❌ No                            |
| **Auto-docs (AsyncAPI/OpenAPI)** | ✅ `catalog/` module                               | ❌ No                                                 | ❌ No                       | ❌ No                            |

### Key Differentiators

**go-cqrs-lite leads on:**

1. Interface segregation — the Sink/Source/Journal split is genuinely superior
2. Local-first support — SQLite + Pebble for embedded scenarios
3. Dialect abstraction — one implementation, multiple SQL backends
4. Transactional outbox — built-in, atomic, not an afterthought
5. Branded IDs — type-safe IDs with generics

**go-cqrs-lite trails on:**

1. Number of storage backends (eventhorizon has MongoDB, DynamoDB, Redis, ScyllaDB via community)
2. Bi-temporal queries (go-eventstore has full bi-temporal model)
3. Migration tooling (none vs basic in most libraries)

---

## 5. Proposed Phases

### Phase 1: Eliminate Split Brain [CRITICAL]

**1.1. Consolidate `storage/sql/` into `storage/`**

Options:

- **Option A: Absorb** — Move all `sql/` exports into root package, delete `sql/` sub-package
- **Option B: Canonical `sql/`** — Root files import from `sql/`, re-export or delegate
- **Option C: Keep `sql/` as public API** — Move all shared logic there, root package becomes thin wrapper

Recommendation: **Option A** (absorb). The root package already has all the private implementations. The `sql/` sub-package was a premature extraction that never completed. One canonical location is simpler.

**1.2. Fix dialect-awareness in `AggregateProjection` + `SQLAggregateReader`**

Both must accept and use `Dialect` for placeholder generation. This is a correctness bug for PostgreSQL.

### Phase 2: Schema Lifecycle [HIGH]

**2.1. Versioned schema migrations**

- Add `schema_migrations` table (version INT PRIMARY KEY, applied_at TIMESTAMPTZ)
- Numbered migration functions: `V1_InitSchema`, `V2_AddCorrelationIDIndex`, etc.
- `Migrate(ctx, db, dialect)` function that applies pending migrations in order
- Keep `InitSchema` as shorthand for fresh databases (applies all at once)
- PostgreSQL and SQLite DDL variants per migration

### Phase 3: Complete Pebble [HIGH]

**3.1. Pebble `SnapshotStore`**

- Key: `cqrs_snapshot:{aggregateType}:{aggregateID}`
- Value: JSON `{version, state, created_at}`
- Implements `SnapshotSink`, `SnapshotSource`, `SnapshotStore`

**3.2. Pebble `CheckpointStore`**

- Key: `cqrs_checkpoint:{projectionName}`
- Value: JSON `{event_id, processed_at}`
- Implements `CheckpointSink`, `CheckpointSource`, `CheckpointStore`

**3.3. Pebble `Outbox`**

- Key: `cqrs_outbox:{id}` with separate pending index
- Append: write entry + add to pending set
- PollPending: scan pending set
- Ack: remove from pending set + mark acked
- Implements `event.Outbox`

**Result:** Pebble becomes a complete single-process backend for CLI tools, desktop apps, edge devices.

### Phase 4: Durability Profiles [MEDIUM]

**4.1. Configurable durability strategies**

```go
type Durability int

const (
    DurabilitySync       Durability = iota // fsync per write
    DurabilityBatchedSync                   // fsync every N ms
    DurabilityAsync                         // OS decides
)
```

- PG: `synchronous_commit` tuning
- SQLite: `PRAGMA synchronous` + WAL auto-checkpoint
- Pebble: already has `WithAsyncWrites()` — generalize to `WithDurability()`

### Phase 5: Outbox Optimization [MEDIUM]

**5.1. Reference-based outbox**

- Outbox table stores event IDs instead of full event JSON
- `PollPending` joins back to `events` table to reconstruct full `OutboxEntry`
- ~60-80% storage reduction for high-throughput systems
- Must remain compatible with `event.Outbox` interface

**5.2. Dead-Letter Queue**

- Add `max_retries` and `status` (pending/failed/dead) to outbox entries
- After N failures, move to `dead` status
- `PollDead(ctx) ([]OutboxEntry, error)` for inspection
- `Retry(ctx, id)` to re-queue dead entries

### Phase 6: Documentation [MEDIUM]

**6.1. Fix STORAGE_GUIDE.md** — align all examples with current API
**6.2. Backend comparison matrix** — which interfaces each backend implements
**6.3. Decision flowchart** — "Which backend should I use?"
**6.4. ADR for storage consolidation** — document the decision

---

## 6. Deliberately Excluded (YAGNI)

| Idea                              | Why Excluded                                                         |
| --------------------------------- | -------------------------------------------------------------------- |
| MySQL/Cockroach/ScyllaDB dialects | `WithDialect` escape hatch exists; add when there's real demand      |
| CRDT event merge                  | Fascinating but premature; requires core interface changes           |
| Bi-temporal event model           | `LoadToTimestamp` + `LoadToVersion` covers 95%; bi-temporal is niche |
| General migration framework       | Just our own schema lifecycle, not a framework                       |
| Streaming Journal for Pebble      | ADR-0007 was correct; use SQL for global replay                      |
| Batched async writes              | Consumers handle batching via middleware                             |
| Event compression                 | Premature; add when profiling shows storage is the bottleneck        |
| Read replicas / sharding          | Consumer's infrastructure concern, not library's                     |

---

## 7. Open Design Decisions

These need stakeholder input before execution:

1. **`storage/sql/` consolidation direction** — absorb into root, make `sql/` canonical, or keep as public sub-package?
2. **Migration system scope** — just numbered up-migrations, or full up/down with rollback?
3. **Pebble priority** — is complete Pebble support needed now, or is Phase 3 aspirational?
4. **Reference-based outbox** — worth the JOIN complexity, or keep JSON blob for simplicity?
5. **DLQ scope** — simple retry counter + dead status, or full DLQ with inspection API?

---

## 8. Appendix: Core Event Interface Hierarchy

Reference for what storage must implement:

```
io.Closer
  ├── EventSink        — Save(ctx, AggregateRef, []Event, Version) + AppendBatch
  ├── EventSource      — Load + LoadFromVersion + LoadToVersion + LoadToTimestamp
  ├── Store            — EventSink + EventSource (composite)
  ├── Journal          — ReadAll(ctx) ([]Event, error)
  ├── SeekableJournal  — ReadFrom(ctx, afterEventID, limit) ([]Event, error)
  ├── BackwardsSource  — LoadBackwards(ctx, AggregateRef) ([]Event, error)
  ├── TransactionalSink — SaveWithOutbox(ctx, AggregateRef, []Event, Version) error
  ├── SnapshotSink     — Save(ctx, Snapshot) + Delete(ctx, AggregateRef)
  ├── SnapshotSource   — Load(ctx, AggregateRef) + LoadAtVersion(ctx, AggregateRef, Version)
  ├── SnapshotStore    — SnapshotSink + SnapshotSource
  ├── CheckpointSink   — Save(ctx, name, Checkpoint)
  ├── CheckpointSource — Load(ctx, name) Checkpoint
  ├── CheckpointStore  — CheckpointSink + CheckpointSource
  ├── Outbox           — Append(ctx, []Event) + PollPending(ctx, limit) + Ack(ctx, []OutboxID)
  ├── StreamLoader     — LoadStream(ctx, AggregateRef) EventStream
  └── EventStream      — Next() (Event, bool) + Err() + Close()
```

**Sentinel errors to return:** `ErrVersionConflict`, `ErrAggregateNotFound`, `ErrSnapshotNotFound`, `ErrStoreClosed`, `ErrBusClosed`

**Event construction for DB reads:** `event.NewEvent()` with `WithEventID()`, `WithOccurredAt()`, `WithSchemaVersion()`, `WithEncoding()`, `WithMetadata()`

---

_Research conducted via deep analysis of source code, ADRs, documentation, philosophy documents, and ecosystem survey._
