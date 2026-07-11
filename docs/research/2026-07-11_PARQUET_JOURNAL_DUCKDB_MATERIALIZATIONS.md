# Parquet Journal + DuckDB Materializations

> **Status:** RESEARCH & DESIGN — comprehensive feasibility analysis and implementation plan
> **Date:** 2026-07-11
> **Scope:** Adding Parquet file support for the event journal and DuckDB for analytical materializations

---

## Table of Contents

1. [Executive Summary](#1-executive-summary)
2. [Research Findings](#2-research-findings)
3. [Architecture Fit Analysis](#3-architecture-fit-analysis)
4. [Proposed Design](#4-proposed-design)
5. [Implementation Plan](#5-implementation-plan)
6. [Risks & Mitigations](#6-risks--mitigations)
7. [Alternatives Considered](#7-alternatives-considered)
8. [Appendix: Technology Reference](#8-appendix-technology-reference)

---

## 1. Executive Summary

### The Opportunity

Two complementary capabilities for the analytics-heavy edge of the CQRS stack:

| Capability                  | What It Enables                                                                                       | Complexity  |
| --------------------------- | ----------------------------------------------------------------------------------------------------- | ----------- |
| **Parquet Journal**         | Columnar, compressed, cloud-native archival event storage with schema evolution                       | Medium-High |
| **DuckDB Materializations** | OLAP-grade analytical read models over Parquet/relational data, 30x faster than SQLite for aggregates | Medium      |

### Key Findings

1. **Parquet is an excellent journal/archive format** but CANNOT be the hot-path write store (files are immutable — no per-aggregate `Save` with optimistic concurrency). It fits as a **SeekableJournal** (append-only log) and as a **cold-storage archive tier** alongside a mutable store (Pebble/SQLite).

2. **DuckDB is a natural fit for materializations** — it implements `database/sql`, so adding a `DuckDBDialect` (11 methods) unlocks `SQLViewStore`, `RelationalProjection`, and `RelationalStore` with zero changes to the view/projection tier. DuckDB queries Parquet files natively with predicate pushdown and column pruning.

3. **The killer combination**: Parquet journal segments written by the event pipeline + DuckDB querying them directly via `read_parquet('events/*.parquet')` for analytics — no ETL, no separate warehouse. This is the "lakehouse for events" pattern.

4. **CGO is the elephant in the room**: `parquet-go` is pure Go (no CGO), but `duckdb-go` requires CGO (C++ engine, statically linked, +30-50MB binary). This breaks the current pure-Go build (modernc.org/sqlite is pure Go). This is the primary architectural decision to make.

5. **DuckLake** (released v1.0 April 2026) solves the "multi-writer ACID over Parquet" problem and could provide a full `event.Store` implementation — but adds significant complexity and is overkill for single-process analytics.

### Recommendation

**Phase the work into three independent deliverables:**

| Phase       | Deliverable                                                        | CGO?         | Dependencies            |
| ----------- | ------------------------------------------------------------------ | ------------ | ----------------------- |
| **Phase 1** | `storage/parquet` — Parquet segment journal (SeekableJournal)      | No (pure Go) | `parquet-go/parquet-go` |
| **Phase 2** | `storage/duckdb` — DuckDB connector + DuckDBDialect                | Yes (CGO)    | `duckdb/duckdb-go/v2`   |
| **Phase 3** | `stack/duckdb` — Preset: DuckDB materializations + Parquet journal | Yes (CGO)    | Both above              |

Each phase is independently valuable and independently deployable. Phase 1 gives cloud-native event archiving with zero CGO. Phase 2 gives OLAP materializations. Phase 3 combines them into the lakehouse pattern.

---

## 2. Research Findings

### 2.1 Parquet in Go

#### Library Choice: `github.com/parquet-go/parquet-go`

| Library                  | Status          | CGO              | Notes                                                                                                                                        |
| ------------------------ | --------------- | ---------------- | -------------------------------------------------------------------------------------------------------------------------------------------- |
| **parquet-go (Segment)** | Active, v0.30.1 | **No** (pure Go) | Generics-first (`GenericWriter[T]`, `GenericReader[T]`). Low memory. Designed for high-volume event data at Twilio Segment. **Recommended.** |
| xitongsys/parquet-go     | Maintenance     | No               | Older, known performance/overflow issues. Some files unreadable by Pandas.                                                                   |
| apache/arrow-go/parquet  | Active          | No               | Best if already using Arrow columnar format. Heavier dependency.                                                                             |

#### Write Patterns

```go
// Streaming write to a file — optimized for minimal memory
writer := parquet.NewGenericWriter[EventRecord](output,
    parquet.Compression(parquet.Zstd),
    parquet.MaxRowsPerRowGroup(10000),
)
writer.Write(events)
writer.Close()  // MUST call — writes footer

// In-memory buffer before flushing (for sorting/batching)
buffer := parquet.NewGenericBuffer[EventRecord]()
buffer.Write(events)
// flush to file when threshold reached:
parquet.CopyRows(fileWriter, buffer.Rows())
buffer.Reset()
```

#### Critical Constraint: Parquet Files Are Immutable

Parquet's file format stores column data first, then a **metadata footer** with offsets to every column chunk in every row group. You cannot append bytes — the footer must be rewritten.

**The industry pattern is log-structured segments**: write a new Parquet file per batch/time window, track them in a manifest, compact periodically via `MergeRowGroups`.

```
journal/
├── manifest.json              # ordered segment index (for seeking)
├── segments/
│   ├── seg_000001.parquet     # events 0–9,999
│   ├── seg_000002.parquet     # events 10,000–19,999
│   └── seg_000003.parquet     # events 20,000+
└── checkpoints/
    └── cp_000002.parquet      # compacted manifest state
```

#### Schema Evolution

Parquet supports add/remove column compatibility via `parquet.Convert`. Each file embeds its own schema in the footer. Reading old files with a newer schema returns null/default for new columns.

```go
// Struct tags control encoding — critical for journal performance
type EventRecord struct {
    ID            string `parquet:"id,delta,zstd"`           // ULID, monotonic → delta encoding
    Timestamp     int64  `parquet:"timestamp,timestamp(microsecond),delta,zstd"`
    Type          string `parquet:"type,dict,zstd"`           // low cardinality → dictionary
    AggregateType string `parquet:"aggregate_type,dict,zstd"`
    AggregateID   string `parquet:"aggregate_id,zstd"`
    Version       int64  `parquet:"version,delta,zstd"`
    Payload       []byte `parquet:"payload,zstd"`
    Encoding      string `parquet:"encoding,dict,zstd"`       // "json", "cbor"
    Metadata      []byte `parquet:"metadata,zstd"`            // CBOR/JSON map
}
```

#### Row-Group Seeking

Each `ColumnChunk` exposes `ColumnIndex()` (min/max per page) and `OffsetIndex()` (page byte offsets). Combined with bloom filters, this enables efficient seeking to specific rows within files.

#### Compression

| Codec    | Ratio   | Speed     | Use Case                        |
| -------- | ------- | --------- | ------------------------------- |
| **ZSTD** | High    | Fast      | **Best default** for event logs |
| Snappy   | Medium  | Fastest   | Real-time/low-latency writes    |
| LZ4_RAW  | Medium  | Very fast | High throughput                 |
| Brotli   | Highest | Slow      | Cold archive                    |

Event payloads with repetitive fields (event types, aggregate types) compress 5-10x with dictionary + ZSTD. A 1 GB JSON event log compresses to ~100-200 MB in Parquet.

### 2.2 DuckDB in Go

#### Driver: `github.com/duckdb/duckdb-go/v2`

> **Note:** `marcboeker/go-duckdb` was archived Oct 2025. The project moved to the official DuckDB org at `github.com/duckdb/duckdb-go/v2` (v2.5.0+). Current DuckDB stable: v1.5.4.

```go
import (
    "database/sql"
    _ "github.com/duckdb/duckdb-go/v2"
)

db, err := sql.Open("duckdb", "analytics.db?threads=4")
defer db.Close()
```

Fully conforms to `database/sql`. Supports `db.Exec`, `db.Query`, prepared statements, transactions.

#### CGO Required

DuckDB is a C++ engine. The Go driver statically links pre-built libraries (macOS/Linux/Windows, amd64+arm64). Binary size increases +30-50MB. Cross-compilation requires `CGO_ENABLED=1`.

**This is the fundamental tension with the current codebase.** The library currently uses `modernc.org/sqlite` (pure Go) and `cockroachdb/pebble` (pure Go). Adding DuckDB introduces CGO to the dependency graph for the first time.

#### DuckDB Queries Parquet Natively

```sql
-- Read directly from Parquet files — no import needed
SELECT * FROM read_parquet('events/*.parquet')
WHERE event_type = 'user.created'
  AND timestamp >= '2026-01-01';

-- Glob patterns, multiple files
SELECT event_type, COUNT(*) FROM read_parquet('journal/segments/*.parquet')
GROUP BY event_type;

-- Projection pushdown: only reads requested columns
SELECT id, type FROM read_parquet('events.parquet');

-- Filter pushdown: skips row groups using min/max statistics
SELECT * FROM read_parquet('events.parquet') WHERE version > 100;
```

#### No Materialized Views — Use CTAS

DuckDB does NOT support `CREATE MATERIALIZED VIEW`. Standard views are virtual (re-run on every read). The materialized pattern is `CREATE TABLE AS SELECT`:

```sql
-- Build a materialized aggregate table from Parquet events
CREATE TABLE user_summary AS
    SELECT
        aggregate_id AS user_id,
        COUNT(*) AS event_count,
        MAX(timestamp) AS last_activity
    FROM read_parquet('events/*.parquet')
    WHERE aggregate_type = 'User'
    GROUP BY aggregate_id;

-- Refresh: DROP TABLE + CREATE TABLE AS SELECT
```

#### Concurrency Model

| Mode       | Access                                                                                                        |
| ---------- | ------------------------------------------------------------------------------------------------------------- |
| Read-write | Single process, multiple threads (MVCC). Appends don't conflict; same-row updates use optimistic concurrency. |
| Read-only  | Multiple processes can read the same file simultaneously.                                                     |

#### Performance vs SQLite

| Aspect             | DuckDB                      | SQLite                 |
| ------------------ | --------------------------- | ---------------------- |
| Architecture       | Columnar, vectorized SIMD   | Row-based, per-row     |
| CSV/scan speed     | ~1.2 GB/s                   | ~40 MB/s (~30x slower) |
| Aggregations       | Vectorized batch processing | Scalar per-row         |
| Parquet support    | Native, first-class         | None                   |
| Point lookups      | Moderate                    | Excellent (B-tree)     |
| Single-row inserts | Moderate                    | Excellent              |

DuckDB wins for analytics (OLAP); SQLite wins for point lookups (OLTP). They are complementary.

#### Appender API for Bulk Writes

```go
conn, _ := db.Conn(ctx)
defer conn.Close()
appender, _ := duckdb.NewAppenderFromConn(conn, "", "events")
defer appender.Close()
for _, evt := range batch {
    appender.AppendRow(...)
}
appender.Flush()
```

### 2.3 DuckLake — ACID Transactions Over Parquet

DuckLake (v1.0, April 2026) is DuckDB's lakehouse layer: a SQL catalog (SQLite/Postgres) tracks which Parquet files belong to which table, providing ACID transactions, time travel, and multi-writer support.

#### Architecture

| Component   | Role                                   | Options                                                   |
| ----------- | -------------------------------------- | --------------------------------------------------------- |
| **Storage** | Where Parquet files live               | Local SSD, S3, GCS, Azure Blob                            |
| **Catalog** | SQL DB holding metadata + inlined data | PostgreSQL (multi-writer), SQLite (single-writer), DuckDB |
| **Compute** | Query engine                           | DuckDB (primary), Spark, Trino, DataFusion                |

#### Key Features for Event Sourcing

- **Atomic appends**: Every `INSERT` creates a snapshot. ACID with snapshot isolation.
- **Data inlining**: Small inserts (≤10 rows) go directly into the catalog DB, flushed to Parquet on `CHECKPOINT`. Enables ~100 TPS vs ~1 TPS for incumbent lakehouse formats.
- **Change Data Feed**: `table_changes()` returns every insert/delete/update between snapshots — a built-in event stream.
- **Time travel**: `SELECT * FROM events AT (VERSION => 3)` — query any historical state.
- **Multi-writer**: Optimistic concurrency control. Each writer writes Parquet to storage independently, then registers in a short catalog transaction.

#### DuckLake Could Implement Full event.Store

Because DuckLake provides ACID over Parquet, it could implement the full `event.Store` interface including optimistic concurrency (`Save` with `expectedVersion`). However, this adds significant complexity (catalog management, extension loading) and is overkill for single-process analytics.

### 2.4 Lakehouse Patterns (Delta / Iceberg / Hudi)

All three lakehouse formats solve the same problem: ACID transactions over immutable Parquet files.

| System                | Metadata                              | Write Pattern                          | Compaction                    | Time Travel                |
| --------------------- | ------------------------------------- | -------------------------------------- | ----------------------------- | -------------------------- |
| **Delta Lake**        | JSON transaction log (`_delta_log/`)  | New Parquet file per append            | `OPTIMIZE` merges small files | Replay log from checkpoint |
| **Apache Iceberg**    | Manifest list → manifest → data files | Immutable data files + manifest        | Rewrite manifests             | Snapshot history           |
| **Apache Hudi (MOR)** | Timeline + log files                  | Append to Avro log, compact to Parquet | Background compaction         | Timeline replay            |
| **DuckLake**          | SQL catalog (SQLite/Postgres)         | Parquet file + catalog insert          | CHECKPOINT                    | Snapshot versioning        |

**For an event journal, the simplest approach is segmented Parquet + manifest** — no full lakehouse format needed. The manifest is just a JSON file listing segments in order with min/max timestamps and row counts.

---

## 3. Architecture Fit Analysis

### 3.1 Interface Compatibility Matrix

The library has two distinct storage concerns:

| Concern                     | Interface               | Key Methods                                                          | Parquet Fit                                                             |
| --------------------------- | ----------------------- | -------------------------------------------------------------------- | ----------------------------------------------------------------------- |
| **Per-aggregate store**     | `event.Store`           | `Save(ref, events, expectedVersion)`, `Load(ref)`, `LoadFromVersion` | **Poor** — immutable files, no optimistic concurrency, no point updates |
| **Cross-aggregate journal** | `event.Journal`         | `ReadAll()`                                                          | **Excellent** — sequential scan over segments                           |
| **Seekable journal**        | `event.SeekableJournal` | `ReadFrom(afterEventID, limit)`                                      | **Good** — segment manifest + binary search                             |
| **Backwards source**        | `event.BackwardsSource` | `LoadBackwards(ref)`                                                 | **Poor** — columnar format optimized for forward scan                   |

| Concern                   | Interface                      | DuckDB Fit                                                               |
| ------------------------- | ------------------------------ | ------------------------------------------------------------------------ |
| **View store**            | `kv.ViewStore[V,K]`            | **Excellent** — via `SQLViewStore` + `DuckDBDialect`                     |
| **Queryable views**       | `kv.ViewQuerier[V]`            | **Excellent** — DuckDB excels at filtering/aggregation                   |
| **Relational projection** | `storage.RelationalProjection` | **Excellent** — multi-table ACID projections                             |
| **Relational store**      | `storage.RelationalStore`      | **Excellent** — dialect-agnostic by design                               |
| **KV blob store**         | `kv.Store`                     | **Moderate** — DuckDB can do KV, but SQLite/Pebble better for point gets |

### 3.2 The Event Store Problem

The `event.Store` interface requires `Save(ctx, ref, events, expectedVersion)` — per-aggregate optimistic concurrency:

```
1. Lock aggregate (sharded mutex)
2. Check existing event count == expectedVersion
3. If mismatch → ErrVersionConflict
4. Write events atomically (batch)
5. Unlock
```

This is fundamentally incompatible with immutable Parquet files. Three options:

| Option                         | Approach                                                                                     | Tradeoff                                                                       |
| ------------------------------ | -------------------------------------------------------------------------------------------- | ------------------------------------------------------------------------------ |
| **A: Parquet as journal-only** | Implement `SeekableJournal` + `Journal`. Don't implement `Store`.                            | Can't serve as primary event store. Must pair with Pebble/SQLite for hot path. |
| **B: Parquet as archive tier** | Hot path writes to Pebble/SQLite. Background flusher copies cold events to Parquet segments. | Two stores to manage. Read path may need to merge hot+cold.                    |
| **C: DuckLake full store**     | Use DuckLake's ACID over Parquet to implement full `Store`.                                  | High complexity. Catalog management. Extension loading.                        |

**Recommendation**: **Options A and B are both worth implementing.** A gives a SeekableJournal for analytics. B gives archival. C is a future exploration.

### 3.3 The Materialization Opportunity

The existing `storage/view/store.go` already supports dialect injection:

```go
func NewViewStoreWithDialect[V any, K fmt.Stringer](
    db *sql.DB,
    dialect sqlpkg.Dialect,
    mapper ViewMapper[V],
    opts ...ViewStoreOption,
) (*SQLViewStore[V, K], error)
```

And `storage/relational/projection.go` takes a dialect:

```go
func NewRelationalProjection(name string, schema RelationalSchema, db *sql.DB,
    dialect sqlpkg.Dialect, handler RelationalHandler, types []event.Type,
    opts ...Option) (*RelationalProjection, error)
```

The `Dialect` interface (11 methods) is the only thing needed:

```go
type Dialect interface {
    Placeholder(index int) string
    FormatTime(t time.Time) any
    ScanTimeDest() any
    ParseTime(src any) (time.Time, error)
    EventSchema() string
    CommandSchema() string
    QuerySchema() string
    SnapshotSchema() string
    CheckpointSchema() string
    KVSchema() string
    TimerSchema() string
}
```

**A `DuckDBDialect` implementing these 11 methods unlocks the entire view/projection/relational tier with zero changes to those packages.** This is the lowest-effort, highest-value deliverable.

### 3.4 The Dialect Design Challenge

DuckDB's SQL has differences from SQLite/Postgres:

| Concern           | SQLite                              | Postgres                   | DuckDB                                                                      |
| ----------------- | ----------------------------------- | -------------------------- | --------------------------------------------------------------------------- |
| Placeholder       | `?`                                 | `$N`                       | `$N` or `?`                                                                 |
| Binary type       | `BLOB`                              | `BYTEA`                    | `BLOB`                                                                      |
| JSON type         | `TEXT`                              | `JSONB`                    | `JSON`                                                                      |
| Timestamp         | `TEXT` (RFC3339)                    | `TIMESTAMP WITH TIME ZONE` | `TIMESTAMP`                                                                 |
| Auto-increment    | `INTEGER PRIMARY KEY AUTOINCREMENT` | `SERIAL`                   | `INTEGER PRIMARY KEY` (no AUTOINCREMENT keyword)                            |
| `datetime('now')` | Yes                                 | No                         | No — use `CURRENT_TIMESTAMP`                                                |
| `ON CONFLICT`     | Yes                                 | Yes (via `UPSERT`)         | **No** (use `INSERT INTO ... ON CONFLICT DO NOTHING` — supported in DuckDB) |
| `PRAGMA`          | Yes                                 | No                         | No                                                                          |

DuckDB supports `$N` placeholders (like Postgres) and `?` (like SQLite). For schema DDL, DuckDB types are closest to Postgres but with `BLOB` instead of `BYTEA` and `TIMESTAMP` instead of `TIMESTAMP WITH TIME ZONE`.

---

## 4. Proposed Design

### 4.1 Module Structure

Following the established two-module pattern (`storage/<engine>` + `stack/<engine>`):

```
storage/parquet/          # NEW — Parquet segment journal
├── go.mod               # deps: parquet-go, event, id, codec, go-error-family
├── journal.go           # SeekableJournal + Journal implementation
├── segment.go           # Segment writer (buffer + flush to file)
├── manifest.go          # Segment manifest (JSON index for seeking)
├── reader.go            # Multi-segment reader (binary search + page scan)
├── compaction.go        # MergeRowGroups compaction
├── schema.go            # EventRecord struct + Parquet schema
├── backend.go           # Backend facade (owns segment directory)
└── journal_test.go

storage/duckdb/          # NEW — DuckDB connector + dialect
├── go.mod               # deps: duckdb-go, storage (for Dialect), go-error-family
├── connector.go         # Open, OpenInMemory, ConfigurePool
├── dialect.go           # DuckDBDialect implementing sqlpkg.Dialect
├── backend.go           # Backend facade (owns *sql.DB)
├── analytics.go         # CTAS helpers, read_parquet wrappers
└── dialect_test.go

stack/duckdb/            # NEW — Preset wiring
├── go.mod               # deps: storage/duckdb, storage/parquet, stack, watermill
├── preset.go            # New() → *Bundle with DuckDB materializations + Parquet journal
├── analytics.go         # WithAnalyticsReadModel helpers
└── preset_test.go
```

### 4.2 Parquet Journal Design (`storage/parquet`)

#### Core Concept: Segment-Based Append-Only Journal

The Parquet journal implements **only** `event.SeekableJournal` (which embeds `Journal`). It does NOT implement `event.Store` — no per-aggregate Save/Load. It's a write-once, read-many cross-aggregate log.

#### Event Record Schema

```go
// EventRecord is the Parquet row representation of an event.
type EventRecord struct {
    ID            string `parquet:"id,delta,zstd"`
    Timestamp     int64  `parquet:"timestamp,timestamp(microsecond),delta,zstd"`
    Type          string `parquet:"type,dict,zstd"`
    AggregateType string `parquet:"aggregate_type,dict,zstd"`
    AggregateID   string `parquet:"aggregate_id,zstd"`
    Version       int64  `parquet:"version,delta,zstd"`
    SchemaVersion int32  `parquet:"schema_version,delta,zstd"`
    Payload       []byte `parquet:"payload,zstd"`
    Encoding      string `parquet:"encoding,dict,zstd"`
    Metadata      []byte `parquet:"metadata,zstd"`  // CBOR/JSON serialized
}
```

Design rationale:

- `delta` encoding on ULID-derived ID, timestamp, and version (monotonic → excellent compression)
- `dict` encoding on type/aggregate_type/encoding (low cardinality → dictionary compression)
- `zstd` everywhere (best ratio/speed balance)
- Timestamp as `timestamp(microsecond)` — Parquet native timestamp type, microsecond precision matches ULID's embedded timestamp
- Metadata as blob (preserves the existing `event.Metadata` CBOR serialization)

#### Segment Lifecycle

```
                   ┌──────────────────────────────────────────┐
                   │          SegmentWriter                   │
  events ────────► │  GenericBuffer[EventRecord]              │
                   │  ┌──────────────────────────────────┐    │
                   │  │ buffer.Write(records)             │    │
                   │  │ buffer.Len() >= flushThreshold?   │    │
                   │  └──────────┬───────────────────────┘    │
                   │             │ yes                         │
                   │             ▼                             │
                   │  NewGenericWriter[EventRecord](file)     │
                   │  CopyRows(writer, buffer.Rows())         │
                   │  writer.Close()                          │
                   │  buffer.Reset()                          │
                   │             │                             │
                   │             ▼                             │
                   │  manifest.Append(segmentMeta)            │
                   │  manifest.Save()                         │
                   └──────────────┬───────────────────────────┘
                                  │
                                  ▼
                   ┌──────────────────────────────────────────┐
                   │          Segment Files                   │
                   │  segments/                                │
                   │    seg_000001.parquet  (rows 0-9999)     │
                   │    seg_000002.parquet  (rows 10000-19999)│
                   │    seg_000003.parquet  (rows 20000+)     │
                   └──────────────────────────────────────────┘
```

#### Manifest Format

```go
// Manifest tracks all segments for ordered reading and seeking.
type Manifest struct {
    Version   int             `json:"version"`
    Segments  []SegmentMeta   `json:"segments"`
    UpdatedAt int64           `json:"updated_at"`
}

type SegmentMeta struct {
    Filename     string `json:"filename"`      // "seg_000003.parquet"
    FirstEventID string `json:"first_event_id"` // ULID — for seeking
    LastEventID  string `json:"last_event_id"`  // ULID — for seeking
    FirstTime    int64  `json:"first_time"`     // unix micros
    LastTime     int64  `json:"last_time"`      // unix micros
    RowCount     int64  `json:"row_count"`
    ByteSize     int64  `json:"byte_size"`
}
```

The manifest is a JSON file written atomically (write to temp + rename). It's the index for `ReadFrom(afterEventID, limit)`:

1. Binary search segments by `LastEventID` to find which segment contains `afterEventID`
2. Open that segment, seek past `afterEventID` (via Parquet row seeking or linear scan within the segment)
3. Read `limit` events, continuing into the next segment if needed

#### Backend Facade

```go
// Backend owns the journal directory and manages segments.
type Backend struct {
    dir         string
    manifest    *Manifest
    writer      *SegmentWriter  // active buffer
    mu          sync.Mutex      // serialize segment flushes
    codec       codec.Codec     // for metadata serialization
    flushSize   int             // rows before flush (default 10,000)
    compression parquet.Compression // default ZSTD
    logger      *slog.Logger
}

func Open(dir string, opts ...Option) (*Backend, error)

// Implements event.SeekableJournal
func (b *Backend) ReadAll(ctx context.Context) ([]event.Event, error)
func (b *Backend) ReadFrom(ctx context.Context, after id.EventID, limit int) ([]event.Event, error)

// Write methods (not part of Store — for journal ingestion only)
func (b *Backend) Append(ctx context.Context, events []event.Event) error
func (b *Backend) Flush() error  // force-segment current buffer
func (b *Backend) Compact() error // merge small segments into larger ones
func (b *Backend) Close() error   // flush + close
```

#### How Events Get Into the Parquet Journal

The Parquet journal is an **append-only sink**, not a primary store. Two ingestion patterns:

**Pattern A: Bus subscriber (live tailing)**

```go
// Subscribe to the event bus and write every event to the Parquet journal
bus.SubscribeAll(func(ctx context.Context, evt event.Event) error {
    return parquetJournal.Append(ctx, []event.Event{evt})
})
```

**Pattern B: Periodic archival (cold flush)**

```go
// Periodically copy events from the hot store (Pebble/SQLite) to Parquet
ticker := time.NewTicker(5 * time.Minute)
for range ticker.C {
    events, _ := hotStore.ReadFrom(ctx, lastArchivedID, 10000)
    parquetJournal.Append(ctx, events)
    lastArchivedID = events[len(events)-1].ID()
}
```

#### Record ↔ Event Conversion

```go
func recordToEvent(r EventRecord) (event.Event, error) {
    eventID, _ := id.ParseEventID(r.ID)
    aggID := id.AggregateID(r.AggregateID)
    aggType := id.AggregateType(r.AggregateType)

    var metadata event.Metadata
    _ = cbor.Unmarshal(r.Metadata, &metadata) // or JSON

    return event.NewEvent(
        event.Type(r.Type),
        aggID,
        aggType,
        event.Version(r.Version),
        // Payload is raw bytes — decoded lazily by consumer via DecodePayloadAuto
        event.WithID(eventID),
        event.WithOccurredAt(time.UnixMicro(r.Timestamp)),
        event.WithSchemaVersion(event.SchemaVersion(r.SchemaVersion)),
        event.WithEncoding(codec.Encoding(r.Encoding)),
        event.WithRawPayload(r.Payload),
        event.WithMetadata(metadata),
    )
}
```

### 4.3 DuckDB Materialization Design (`storage/duckdb`)

#### DuckDBDialect

```go
package duckdb

type DuckDBDialect struct{}

func (DuckDBDialect) Placeholder(index int) string {
    return "$" + strconv.Itoa(index) // DuckDB supports $N (like Postgres)
}

func (DuckDBDialect) FormatTime(t time.Time) any {
    return t // DuckDB has native TIMESTAMP support via driver
}

func (DuckDBDialect) ScanTimeDest() any {
    return new(time.Time) // DuckDB driver returns time.Time natively
}

func (DuckDBDialect) ParseTime(src any) (time.Time, error) {
    tp, ok := src.(*time.Time)
    if !ok {
        return time.Time{}, errorfamily.WrapCorruption(
            errors.New("duckdb: unexpected time scan type"),
            "duckdb.time_parse", "timestamp column")
    }
    return *tp, nil
}

func (DuckDBDialect) EventSchema() string {
    return `CREATE TABLE IF NOT EXISTS events (
        id               VARCHAR PRIMARY KEY,
        event_type       VARCHAR NOT NULL,
        aggregate_type   VARCHAR NOT NULL,
        aggregate_id     VARCHAR NOT NULL,
        version          BIGINT NOT NULL,
        schema_version   INTEGER NOT NULL DEFAULT 1,
        payload          BLOB,
        payload_encoding VARCHAR DEFAULT 'json',
        metadata         JSON,
        occurred_at      TIMESTAMP NOT NULL,
        created_at       TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
        UNIQUE(aggregate_type, aggregate_id, version)
    )`
    // + indexes using standard CREATE INDEX
}

// CommandSchema, QuerySchema, SnapshotSchema, CheckpointSchema,
// KVSchema, TimerSchema — analogous, with DuckDB type mappings:
//   BLOB (not BYTEA), JSON (not JSONB), TIMESTAMP (not TIMESTAMPTZ),
//   BIGINT (not INTEGER for large values), VARCHAR (not TEXT)
```

#### Key Dialect Decisions

| Decision       | Choice                          | Rationale                                                                                                                                             |
| -------------- | ------------------------------- | ----------------------------------------------------------------------------------------------------------------------------------------------------- |
| Placeholder    | `$N`                            | DuckDB supports both `$N` and `?`. `$N` matches Postgres pattern, easier for named-arg debugging.                                                     |
| Timestamp      | `TIMESTAMP` (not `TIMESTAMPTZ`) | DuckDB `TIMESTAMP` stores microsecond precision. The Go driver returns `time.Time` natively. `TIMESTAMPTZ` is supported but adds timezone complexity. |
| JSON           | `JSON`                          | DuckDB has native JSON type with `json_extract()`, `json_array_length()`, etc. Superior to `TEXT` for metadata queries.                               |
| Binary         | `BLOB`                          | Standard DuckDB binary type.                                                                                                                          |
| `ON CONFLICT`  | Supported                       | DuckDB supports `INSERT INTO ... ON CONFLICT DO NOTHING/UPDATE` — so `SQLViewStore` upserts work.                                                     |
| Auto-increment | `INTEGER PRIMARY KEY`           | DuckDB auto-increments `INTEGER PRIMARY KEY` columns natively (no `AUTOINCREMENT` keyword needed).                                                    |

#### Backend Facade

```go
package duckdb

// Backend wraps a DuckDB connection for use with the storage layer.
type Backend struct {
    db      *sql.DB
    dialect DuckDBDialect
}

// Open creates a persistent DuckDB database.
func Open(dsn string, opts ...Option) (*Backend, error)

// OpenInMemory creates an in-memory DuckDB (for testing/analytics scratch).
func OpenInMemory() (*Backend, error)

// DB returns the underlying *sql.DB for direct queries.
func (b *Backend) DB() *sql.DB

// Dialect returns the DuckDB dialect for use with SQLViewStore etc.
func (b *Backend) Dialect() sqlpkg.Dialect

// Close closes the database connection.
func (b *Backend) Close() error
```

#### Analytics Helpers (Parquet → Materialized Tables)

```go
// MaterializeFromParquet creates a DuckDB table from Parquet files.
// This is the "lakehouse materialized view" pattern.
func MaterializeFromParquet(
    ctx context.Context,
    db *sql.DB,
    tableName string,
    parquetGlob string,  // e.g., "journal/segments/*.parquet"
    query string,        // SQL transformation (SELECT ... FROM read_parquet(...))
) error {
    // DROP TABLE IF EXISTS + CREATE TABLE AS SELECT
    fullQuery := fmt.Sprintf(
        "CREATE OR REPLACE TABLE %s AS %s",
        tableName, query,
    )
    _, err := db.ExecContext(ctx, fullQuery)
    return err
}

// RefreshMaterialized drops and recreates a materialized table.
func RefreshMaterialized(ctx context.Context, db *sql.DB, tableName string) error
```

#### Connector (following turso pattern)

```go
// Open opens a persistent DuckDB database.
func Open(dsn string) (*sql.DB, error) {
    if dsn == "" {
        dsn = ":memory:"
    }
    db, err := sql.Open("duckdb", dsn)
    if err != nil {
        return nil, fmt.Errorf("duckdb: failed to open %q: %w", dsn, err)
    }
    db.SetMaxOpenConns(1) // DuckDB is single-writer; pool size 1 serializes
    return db, nil
}

// OpenInMemory creates an in-memory DuckDB.
func OpenInMemory() (*sql.DB, error) {
    return Open("")
}

// ConfigurePool sets sane defaults for CQRS workloads.
func ConfigurePool(db *sql.DB) {
    db.SetMaxOpenConns(1)       // single-writer serialization
    db.SetMaxIdleConns(1)
    db.SetConnMaxLifetime(0)    // keep connection alive (temp objects are per-conn)
}
```

### 4.4 Stack Preset Design (`stack/duckdb`)

```go
package duckdb

// Bundle wraps *stack.Bundle with DuckDB-specific analytics capabilities.
type Bundle struct {
    *stack.Bundle
    duckdb  *cqrsduckdb.Backend
    journal *parquet.Backend // optional Parquet journal
}

type Option func(*config)

type config struct {
    duckdbDSN     string
    parquetDir    string // empty = no Parquet journal
    parquetFlush  int    // rows per segment
    threads       int    // DuckDB thread count
    memoryLimit   string // DuckDB memory limit (e.g., "4GB")
}

func New(dsn string, opts ...Option) (*Bundle, error) {
    // 1. Open DuckDB
    duckBackend, _ := cqrsduckdb.Open(dsn)

    // 2. Run migrations (all DuckDB dialect schemas)
    // 3. Create SQLBackend facade with DuckDBDialect
    sqlBackend, _ := storage.NewBackendWithDialect(db, cqrsduckdb.DuckDBDialect{})

    // 4. Optionally open Parquet journal
    var parquetJournal *parquet.Backend
    if cfg.parquetDir != "" {
        parquetJournal, _ = parquet.Open(cfg.parquetDir)
    }

    // 5. Wire stack.Bundle
    bundle, _ := stack.New(
        stack.WithEventStore(sqlBackend.EventStore()),
        stack.WithSeekableJournal(sqlBackend),       // or parquetJournal if configured
        stack.WithCommandStore(sqlBackend.CommandStore()),
        stack.WithQueryStore(sqlBackend.QueryStore()),
        stack.WithSnapshotStore(sqlBackend.SnapshotStore()),
        stack.WithCheckpointStore(sqlBackend.CheckpointStore()),
        stack.WithReadModels(/* DuckDB KV store */),
        stack.WithBus(cqrswatermill.NewEventBus()),
        stack.WithCloser(duckBackend),
    )

    return &Bundle{Bundle: bundle, duckdb: duckBackend, journal: parquetJournal}, nil
}

// WithParquetJournal enables Parquet segment journaling.
func WithParquetJournal(dir string, flushSize int) Option

// WithThreads sets DuckDB's thread count.
func WithThreads(n int) Option

// WithMemoryLimit sets DuckDB's memory limit.
func WithMemoryLimit(limit string) Option

// MaterializeFromParquet builds an analytics table from Parquet segments.
func (b *Bundle) MaterializeFromParquet(ctx context.Context, table, query string) error

// ParquetJournal returns the Parquet journal backend (nil if not configured).
func (b *Bundle) ParquetJournal() *parquet.Backend
```

### 4.5 Usage Recipes

#### Recipe 1: DuckDB as the only materialization engine

```go
// DuckDB replaces SQLite for analytics-heavy read models
bundle, _ := duckdb.New("analytics.db",
    duckdb.WithThreads(4),
    duckdb.WithMemoryLimit("4GB"),
)
defer bundle.Close()

// Standard SQLViewStore works with DuckDBDialect
todoStore, _ := storage.NewViewStoreWithDialect[TodoView, TodoID](
    bundle.Database(), duckdb.DuckDBDialect{}, mapper,
)

// Query with DuckDB's analytical power
results, _ := todoStore.Query(ctx, kv.ViewQuery{
    Conditions: []kv.Condition{{Column: "completed", Op: kv.OpEq, Value: false}},
    OrderBy:    "created_at", Desc: true, Limit: 50,
})
```

#### Recipe 2: Parquet journal + DuckDB analytics (the lakehouse pattern)

```go
// Pebble for hot-path event store (per-aggregate optimistic concurrency)
// Parquet journal for archival + analytics source
// DuckDB for querying Parquet directly

hotStore, _ := pebble.Open("data/pebble")
parquetJournal, _ := parquet.Open("data/parquet",
    parquet.WithFlushSize(10000),
    parquet.WithCompression(parquet.Zstd),
)
analyticsDB, _ := duckdb.OpenInMemory()

// Wire: bus publishes → Pebble stores → Parquet archives
bus.SubscribeAll(func(ctx context.Context, evt event.Event) error {
    return parquetJournal.Append(ctx, []event.Event{evt})
})

// Analytics: DuckDB queries Parquet directly — no ETL
_, _ = analyticsDB.ExecContext(ctx, `
    CREATE TABLE user_signups AS
    SELECT
        date_trunc('day', occurred_at) AS day,
        COUNT(*) AS signups
    FROM read_parquet('data/parquet/segments/*.parquet')
    WHERE event_type = 'user.created'
    GROUP BY day
    ORDER BY day
`)
```

#### Recipe 3: DuckLake for full ACID over Parquet (future)

```go
// DuckLake provides ACID transactions over Parquet — could implement full Store
// This is a future exploration, not Phase 1-3.
//
// ATTACH 'ducklake:postgres://catalog-host/lake' AS lake (DATA_PATH 's3://events/');
// INSERT INTO lake.events VALUES (...);  -- atomic append, snapshot isolation
// SELECT * FROM lake.events AT (VERSION => 42);  -- time travel
```

---

## 5. Implementation Plan

### Phase 1: Parquet Segment Journal (`storage/parquet`)

**Effort: ~3-4 days** | **CGO: No** | **Dependencies: parquet-go**

| Step | Task                                                                   | Deliverable                     |
| ---- | ---------------------------------------------------------------------- | ------------------------------- |
| 1.1  | Create module `storage/parquet/go.mod`                                 | `go mod init`, add to `go.work` |
| 1.2  | Define `EventRecord` Parquet schema struct                             | `schema.go`                     |
| 1.3  | Implement `SegmentWriter` (GenericBuffer + flush to file)              | `segment.go`                    |
| 1.4  | Implement `Manifest` (JSON index, atomic save)                         | `manifest.go`                   |
| 1.5  | Implement `Backend.Open(dir)` + `Append` + `Flush`                     | `backend.go`                    |
| 1.6  | Implement `ReadAll` (iterate all segments)                             | `journal.go`                    |
| 1.7  | Implement `ReadFrom` (binary search manifest + seek within segment)    | `journal.go`                    |
| 1.8  | Implement record ↔ event conversion                                    | `convert.go`                    |
| 1.9  | Implement `Compact` (MergeRowGroups)                                   | `compaction.go`                 |
| 1.10 | Write tests (table-driven: append, read, seek, compact)                | `journal_test.go`               |
| 1.11 | Add dependency budget to `scripts/check-module-layers.sh`              | script update                   |
| 1.12 | Add `DEP_BUDGET["storage/parquet"]=4` and `LAYER["storage/parquet"]=5` | script update                   |

**Acceptance criteria:**

- `storage/parquet.Backend` implements `event.SeekableJournal`
- `Append` buffers events and flushes to Parquet segments at threshold
- `ReadAll` returns all events across segments in OccurredAt order
- `ReadFrom(afterEventID, limit)` binary-searches the manifest and returns the correct page
- `Compact` merges small segments into larger ones
- Tests pass with `go test ./storage/parquet/... -count=1`
- No CGO required

### Phase 2: DuckDB Connector + Dialect (`storage/duckdb`)

**Effort: ~2-3 days** | **CGO: Yes** | **Dependencies: duckdb-go**

| Step | Task                                                                     | Deliverable                     |
| ---- | ------------------------------------------------------------------------ | ------------------------------- |
| 2.1  | Create module `storage/duckdb/go.mod`                                    | `go mod init`, add to `go.work` |
| 2.2  | Implement `DuckDBDialect` (all 11 methods)                               | `dialect.go`                    |
| 2.3  | Verify DuckDB DDL syntax for all schemas                                 | `diaction_test.go`              |
| 2.4  | Implement `Open`, `OpenInMemory`, `ConfigurePool`                        | `connector.go`                  |
| 2.5  | Implement `Backend` facade                                               | `backend.go`                    |
| 2.6  | Write dialect tests (round-trip: create table, insert, query, scan time) | `dialect_test.go`               |
| 2.7  | Verify `SQLViewStore` works with DuckDBDialect                           | integration test                |
| 2.8  | Verify `RelationalProjection` works with DuckDBDialect                   | integration test                |
| 2.9  | Add dependency budget + layer to check-module-layers.sh                  | script update                   |
| 2.10 | Document CGO requirement in module README                                | `README.md`                     |

**Acceptance criteria:**

- `DuckDBDialect` implements all 11 `sqlpkg.Dialect` methods
- `storage.NewViewStoreWithDialect[V,K](db, duckdb.DuckDBDialect{}, mapper)` produces a working view store
- CRUD operations (Set, Get, Delete, Scan, Query) work with DuckDB
- `storage.NewRelationalProjection` works with DuckDBDialect
- Timestamp round-trips correctly (FormatTime → ScanTimeDest → ParseTime)
- `ON CONFLICT DO UPDATE` (upsert) works for view store
- Tests pass with `go test ./storage/duckdb/... -count=1`

**Key risks to validate early:**

- Does DuckDB's `ON CONFLICT DO UPDATE` support the same syntax the view store generates?
- Does DuckDB handle `CREATE INDEX IF NOT EXISTS`?
- Does DuckDB's `database/sql` driver support `lastInsertId` / `RowsAffected` correctly?
- Pool size 1 — does it serialize correctly under concurrent access?

### Phase 3: Stack Preset + Analytics (`stack/duckdb`)

**Effort: ~2 days** | **CGO: Yes** | **Dependencies: both**

| Step | Task                                                             | Deliverable                 |
| ---- | ---------------------------------------------------------------- | --------------------------- |
| 3.1  | Create module `stack/duckdb/go.mod`                              | module init, add to go.work |
| 3.2  | Implement `New(dsn, opts...)`                                    | `preset.go`                 |
| 3.3  | Implement `WithParquetJournal`, `WithThreads`, `WithMemoryLimit` | `preset.go`                 |
| 3.4  | Implement `MaterializeFromParquet` analytics helper              | `analytics.go`              |
| 3.5  | Write contract test (follows stack/sqlite contract pattern)      | `contract_test.go`          |
| 3.6  | Add dependency budget                                            | script update               |
| 3.7  | Write example usage in `example/`                                | optional                    |

**Acceptance criteria:**

- `duckdb.New(dsn)` returns a working `*Bundle` with all stores wired
- View stores use DuckDBDialect
- `MaterializeFromParquet` creates a table from Parquet glob
- `duckdb.New(dsn, WithParquetJournal(dir))` wires Parquet journal as SeekableJournal
- Contract tests pass (same pattern as stack/sqlite)

### Phase 4 (Future): DuckLake Full Store

**Effort: ~5+ days** | **CGO: Yes** | **Complexity: High**

This is a future exploration. DuckLake would allow implementing the full `event.Store` interface over Parquet with ACID transactions. It would be a separate module `storage/ducklake` that depends on both DuckDB and a catalog DB.

Not recommended for initial implementation — the segmented Parquet journal (Phase 1) + DuckDB materializations (Phase 2-3) cover 90% of use cases with much less complexity.

---

## 6. Risks & Mitigations

### 6.1 CGO Introduction

**Risk:** DuckDB requires CGO. The codebase currently has zero CGO dependencies (modernc.org/sqlite is pure Go). This affects:

- Build reproducibility (needs C toolchain in CI)
- Cross-compilation (needs `CGO_ENABLED=1` + target arch toolchain)
- Binary size (+30-50MB from statically linked DuckDB)
- Docker image size (needs libc/glibc)

**Mitigation:**

- Keep `storage/duckdb` and `stack/duckdb` as **opt-in modules** — not imported by any core module
- Document CGO requirement clearly in module READMEs
- CI: add a separate `duckdb` build job that sets `CGO_ENABLED=1`
- The Parquet journal (`storage/parquet`) is pure Go — it works without DuckDB
- Consumers who don't need DuckDB pay zero cost

### 6.2 Parquet Immutability vs. Store Semantics

**Risk:** Users may expect `storage/parquet` to implement `event.Store` (with per-aggregate Save/Load). It only implements `SeekableJournal`.

**Mitigation:**

- Clear naming: `parquet.Backend` exposes `Append`, `ReadAll`, `ReadFrom` — not `Save`/`Load`
- Documentation: "Parquet journal is an append-only cross-aggregate log, NOT a per-aggregate store"
- Stack preset wires it as `SeekableJournal` only, always pairing with a real Store (Pebble/SQLite/DuckDB)
- The type system enforces this: `Backend` does not implement `event.Store`

### 6.3 DuckDB Concurrency

**Risk:** DuckDB is single-writer. Concurrent writes to the same database file from multiple processes fail.

**Mitigation:**

- `ConfigurePool` sets `MaxOpenConns(1)` to serialize within-process access
- For multi-process: use read-only mode for readers, single writer process
- Document clearly: "DuckDB is embedded single-writer. For multi-process, use Postgres or DuckLake."

### 6.4 Schema DDL Differences

**Risk:** DuckDB DDL differs from SQLite/Postgres in subtle ways (no `AUTOINCREMENT`, no `PRAGMA`, `TIMESTAMP` vs `TIMESTAMPTZ`).

**Mitigation:**

- Test every `Dialect.*Schema()` DDL statement against DuckDB
- Integration tests: create table, insert, query, scan — for every schema
- Validate `ON CONFLICT` syntax early (critical for view store upserts)

### 6.5 Dependency Budget

**Risk:** `check-module-layers.sh` enforces per-module dependency budgets. Adding parquet-go (1 dep) and duckdb-go (1 dep) needs budget entries.

**Mitigation:**

- `storage/parquet`: budget 4 (parquet-go + event + codec + go-error-family)
- `storage/duckdb`: budget 4 (duckdb-go + storage + go-error-family + internal)
- `stack/duckdb`: budget 6 (storage/duckdb + storage/parquet + stack + watermill + go-error-family + codec)
- Add these to `scripts/check-module-layers.sh` before first CI run

### 6.6 DuckDB Driver Migration

**Risk:** `marcboeker/go-duckdb` was archived Oct 2025. The official driver is now `github.com/duckdb/duckdb-go/v2`.

**Mitigation:**

- Use `github.com/duckdb/duckdb-go/v2` from the start
- Pin a stable version in go.mod
- Monitor for breaking changes in the v2.x series

---

## 7. Alternatives Considered

### 7.1 Arrow Instead of Parquet

**Considered:** Using Apache Arrow (in-memory columnar) as the journal format instead of Parquet (on-disk columnar).

**Dismissed:** Arrow is an in-memory format — it doesn't solve persistence. Parquet is the on-disk counterpart. Arrow would be relevant only if we needed zero-copy in-memory analytics (e.g., via Arrow Flight), which adds gRPC complexity. Parquet + DuckDB covers the same ground more simply.

### 7.2 Full Lakehouse Format (Delta/Iceberg/Hudi)

**Considered:** Implementing a Delta Lake or Iceberg-compatible transaction log over Parquet segments.

**Dismissed:** These formats are designed for distributed multi-writer clusters (Spark, Trino). For a single-process Go library, a simple JSON manifest + segment files is sufficient. If multi-writer ACID is needed, DuckLake (which wraps this complexity) is the better path. The manifest format in Phase 1 can be upgraded to DuckLake later without changing the `SeekableJournal` interface.

### 7.3 ClickHouse Instead of DuckDB

**Considered:** ClickHouse as the analytical engine (faster for some workloads, columnar-native).

**Dismissed:** ClickHouse is a server process, not an embedded database. It requires deployment, network configuration, and operational overhead. DuckDB is embedded (in-process), zero-config, and follows the library-not-framework principle. ClickHouse is better suited as a sink via `transport/` (future).

### 7.4 Parquet Via Arrow-Go Instead of parquet-go

**Considered:** Using `github.com/apache/arrow-go/parquet` for Parquet support.

**Dismissed:** Arrow-Go is a heavier dependency (full Arrow columnar format). `parquet-go/parquet-go` is purpose-built for Parquet, has a cleaner generics-first API, and was designed for high-volume event data at Twilio Segment. It's pure Go with a smaller dependency footprint.

### 7.5 DuckDB-Only (No Parquet)

**Considered:** Skipping Parquet entirely, using DuckDB's native storage format for both journal and materializations.

**Dismissed:** DuckDB's native format is not cloud-portable (can't be read by other engines). Parquet is the lingua franca of data lakes — it can be read by DuckDB, Spark, Trino, Pandas, and any analytical tool. Writing Parquet segments ensures the event archive is engine-agnostic and cloud-native (S3/GCS). The combination (Parquet storage + DuckDB compute) is more powerful than either alone.

### 7.6 Turso-Style Delegation (DuckDB Delegates to SQLite Code)

**Considered:** Following the `storage/turso` pattern where Turso delegates to SQLite store constructors (because Turso speaks SQLite dialect).

**Dismissed:** DuckDB's SQL dialect is NOT SQLite-compatible (different types, no PRAGMA, different conflict handling). A DuckDBDialect must be a standalone implementation. However, the `storage/turso` pattern of branded DSN types (`DbPath`, `RemoteURL`) is a good pattern to adopt for DuckDB configuration.

---

## 8. Appendix: Technology Reference

### 8.1 Key Libraries

| Library    | Import Path                        | Version | CGO | Purpose                 |
| ---------- | ---------------------------------- | ------- | --- | ----------------------- |
| parquet-go | `github.com/parquet-go/parquet-go` | v0.30.1 | No  | Parquet file read/write |
| duckdb-go  | `github.com/duckdb/duckdb-go/v2`   | v2.5.0+ | Yes | DuckDB Go driver        |

### 8.2 Existing Interface Contracts (Reference)

```go
// event.SeekableJournal — what Parquet journal implements
type SeekableJournal interface {
    Journal  // ReadAll(ctx) ([]Event, error)
    ReadFrom(ctx context.Context, afterEventID id.EventID, limit int) ([]Event, error)
}

// event.CheckpointStore — what projectionhost needs
type CheckpointStore interface {
    Save(ctx context.Context, projectionName string, cp Checkpoint) error
    Load(ctx context.Context, projectionName string) (Checkpoint, error)
}

// sqlpkg.Dialect — what DuckDBDialect implements (11 methods)
type Dialect interface {
    Placeholder(index int) string
    FormatTime(t time.Time) any
    ScanTimeDest() any
    ParseTime(src any) (time.Time, error)
    EventSchema() string
    CommandSchema() string
    QuerySchema() string
    SnapshotSchema() string
    CheckpointSchema() string
    KVSchema() string
    TimerSchema() string
}

// event.ImmutableEvent fields (for Parquet record mapping)
type ImmutableEvent struct {
    id            id.EventID        // ULID
    eventType     Type              // string
    aggregateID   id.AggregateID    // string-backed
    aggregateType id.AggregateType  // string
    version       Version           // uint64
    schemaVersion SchemaVersion     // int
    encoding      codec.Encoding    // "json" | "cbor"
    payload       []byte            // raw encoded payload
    metadata      Metadata          // structured metadata
    occurredAt    time.Time
}
```

### 8.3 Serialization Reference (from Pebble)

The Pebble store uses this CBOR envelope for event serialization — the Parquet journal should mirror the same field set:

```go
type serializableEvent struct {
    ID            id.EventID     `json:"id"`
    Type          string         `json:"type"`
    AggregateID   id.AggregateID `json:"aggregate_id"`
    AggregateType string         `json:"aggregate_type"`
    Version       int            `json:"version"`
    SchemaVersion int            `json:"schema_version,omitempty"`
    Payload       []byte         `json:"payload"`
    OccurredAt    int64          `json:"occurred_at"`  // UnixNano
    Metadata      event.Metadata `json:"metadata"`
    Encoding      string         `json:"encoding,omitempty"`
}
```

The Parquet `EventRecord` mirrors these fields with Parquet-specific encoding tags (`delta`, `dict`, `zstd`).

### 8.4 projectionhost Consumption Pattern

The Parquet journal must satisfy the `projectionhost` drain loop contract:

```
1. cp := cpStore.Load(ctx, projectionName)     // load last checkpoint
2. events := journal.ReadFrom(ctx, cp.EventID, batchSize)  // page of events after checkpoint
3. for each evt: apply to projection
4. cpStore.Save(ctx, projectionName, Checkpoint{lastEventID, now})
5. if len(events) == batchSize: goto 2  // keep paging
```

**Contract**: `ReadFrom(afterEventID, limit)` returns events ordered by OccurredAt, starting after `afterEventID`. Empty/short page = caught up. The Parquet journal must:

- Order events by OccurredAt across all segments (the manifest is sorted by FirstEventID, which is ULID — ULIDs are time-sortable)
- Handle the seek correctly: find the segment containing `afterEventID`, skip to it, read forward
- Return at most `limit` events (may span segment boundaries)

### 8.5 DuckDB Native Parquet Query Patterns

```sql
-- DuckDB reads Parquet with zero import — predicate + projection pushdown
SELECT id, type, occurred_at FROM read_parquet('journal/segments/*.parquet')
WHERE type = 'user.created' AND occurred_at > '2026-07-01'
ORDER BY occurred_at DESC LIMIT 100;

-- Aggregation directly over Parquet (DuckDB's sweet spot)
SELECT
    date_trunc('hour', occurred_at) AS hour,
    type,
    COUNT(*) AS event_count,
    AVG(CAST(LENGTH(payload) AS BIGINT)) AS avg_payload_size
FROM read_parquet('journal/segments/*.parquet')
GROUP BY hour, type
ORDER BY hour DESC;

-- Join events with DuckDB materialized tables
SELECT e.aggregate_id, u.name, e.type, e.occurred_at
FROM read_parquet('events/*.parquet') e
JOIN users u ON e.aggregate_id = u.id
WHERE e.type LIKE 'user.%';

-- Export query results back to Parquet
COPY (
    SELECT type, COUNT(*) as cnt FROM read_parquet('events/*.parquet') GROUP BY type
) TO 'analytics/event_counts.parquet' (FORMAT parquet, COMPRESSION zstd);

-- Schema inspection
DESCRIBE SELECT * FROM read_parquet('journal/segments/seg_000001.parquet');
```

---

## Summary Decision Matrix

| Decision                                 | Choice                  | Confidence | Reversible?                           |
| ---------------------------------------- | ----------------------- | ---------- | ------------------------------------- |
| Parquet library                          | `parquet-go/parquet-go` | High       | Yes (isolated module)                 |
| DuckDB driver                            | `duckdb/duckdb-go/v2`   | High       | Yes (isolated module)                 |
| Parquet as journal-only (not full Store) | Yes                     | High       | Yes (can add DuckLake later)          |
| DuckDBDialect with `$N` placeholders     | Yes                     | Medium     | Yes (also supports `?`)               |
| Segment + manifest (not full lakehouse)  | Yes                     | High       | Yes (manifest → DuckLake upgradeable) |
| Phase into 3 independent deliverables    | Yes                     | High       | N/A                                   |
| CGO opt-in (not in core modules)         | Yes                     | High       | Yes (modules are isolated)            |

**The design follows every library principle**: library-not-framework (opt-in modules), composition over inheritance (dialect interface, stack.Bundle), minimal dependencies (pure Go Parquet, isolated CGO DuckDB), strong types (EventRecord with typed Parquet tags), and the existing patterns (two-module backend+stack, Dialect interface, SeekableJournal).
