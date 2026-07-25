# Parquet Journal Design — Phase 1

> **Date:** 2026-07-25 · **Status:** Design (Tier 4 expansion)
> **Related:** [Research doc](../research/archive/2026-07-11_PARQUET_JOURNAL_DUCKDB_MATERIALIZATIONS.md), [ROADMAP](../../ROADMAP.md) Section 4
> **Scope:** `storage/parquet` — segment-based SeekableJournal (Phase 1 only)

---

## Context

The existing storage backends (SQLite, Pebble, memory) are mutable stores: they
support `event.Store` (read + write with optimistic concurrency). Parquet files
are **immutable after write** — the footer must be rewritten to append rows.
This makes Parquet unsuitable for per-aggregate `Save` with version checks, but
ideal as a **segment-based append-only journal** implementing
`event.SeekableJournal`.

The ROADMAP defines three phases:

1. `storage/parquet` — Parquet segment journal (pure Go, no CGO) ← **this doc**
2. `storage/duckdb` — DuckDB connector + dialect (CGO required)
3. `stack/duckdb` — Preset combining Parquet journal + DuckDB materializations

Phase 1 is independently valuable: it provides cloud-native, compressed event
archival (5-10x compression via dictionary + ZSTD) with seekable replay.

## Architecture

```
┌──────────────────────────────────────────────────────────┐
│  storage/parquet/                                        │
│                                                          │
│  ┌─────────────┐     ┌──────────────┐                   │
│  │  Writer     │────▶│  Segment     │  seg_000001.parquet│
│  │  (batch     │     │  Manager     │  seg_000002.parquet│
│  │   flusher)  │     │  (manifest)  │  seg_000003.parquet│
│  └─────────────┘     └──────┬───────┘                   │
│                             │                            │
│  ┌─────────────┐            │            ┌────────────┐ │
│  │  Reader     │◀───────────┘            │  Manifest  │ │
│  │  (Seekable  │                         │  manifest.json│
│  │   Journal)  │                         └────────────┘ │
│  └─────────────┘                                        │
│                                                          │
│  Implements: event.SeekableJournal                       │
│  Does NOT implement: event.Store, event.EventSink        │
└──────────────────────────────────────────────────────────┘
```

## Segment Layout

```
journal/
├── manifest.json              # segment index: [{id, file, minTime, maxTime, count}]
├── segments/
│   ├── seg_000001.parquet     # rows 0–9,999 (sealed)
│   ├── seg_000002.parquet     # rows 10,000–19,999 (sealed)
│   └── seg_000003.parquet     # rows 20,000+ (active, appended on flush)
└── checkpoints/
    └── cp_projection_x.json   # projection offsets (optional, per-consumer)
```

**Segment sealing:** A segment is "sealed" when it reaches `MaxRowsPerSegment`
(default 10,000). Sealed segments are immutable. The active segment is buffered
in memory until flush.

## Parquet Schema

```go
// EventRecord is the on-disk Parquet row. Maps 1:1 to ImmutableEvent fields.
type EventRecord struct {
    ID            string `parquet:"id,delta,zstd"`
    Timestamp     int64  `parquet:"timestamp,timestamp(microsecond),delta,zstd"`
    Type          string `parquet:"type,dict,zstd"`
    StreamType    string `parquet:"stream_type,dict,zstd"`
    StreamID      string `parquet:"stream_id,zstd"`
    Version       int64  `parquet:"version,delta,zstd"`
    SchemaVersion int64  `parquet:"schema_version,delta,zstd"`
    Payload       []byte `parquet:"payload,zstd"`
    Encoding      string `parquet:"encoding,dict,zstd"`
    Metadata      []byte `parquet:"metadata,zstd"`
}
```

**Encoding rationale:**

| Column        | Encoding        | Why                                           |
| ------------- | --------------- | --------------------------------------------- |
| `id`          | delta           | ULID, monotonic — delta encodes as small ints |
| `timestamp`   | timestamp+delta | Time-ordered, microsecond precision           |
| `type`        | dict            | Low cardinality (10-100 event types)          |
| `stream_type` | dict            | Low cardinality (5-20 stream types)           |
| `version`     | delta           | Sequential within a stream                    |
| `payload`     | zstd            | Variable-length bytes, high compression       |
| `metadata`    | zstd            | JSON/CBOR map bytes                           |

All columns use ZSTD compression (best ratio for event data, fast decompression).

## Public API

```go
package parquet

// Journal implements event.SeekableJournal over Parquet segment files.
type Journal struct { /* unexported */ }

// Option configures the journal.
type Option func(*config)

func WithMaxRowsPerSegment(n int) Option
func WithCompression(codec parquet.Compression) Option
func WithLogger(logger *slog.Logger) Option
func WithSyncWrites(sync bool) Option  // fsync after segment seal

// Open creates or opens a journal at the given directory path.
func Open(dir string, opts ...Option) (*Journal, error)

// Append writes events to the active segment. Flushes the segment if it
// reaches MaxRowsPerSegment. NOT thread-safe — caller must serialize.
func (j *Journal) Append(ctx context.Context, events []event.Event) error

// ReadAll reads all events across all segments (sealed + active).
// Implements event.Journal.ReadAll.
func (j *Journal) ReadAll(ctx context.Context) ([]event.Event, error)

// ReadFrom reads events ordered by OccurredAt, starting after afterEventID.
// Implements event.SeekableJournal.ReadFrom.
func (j *Journal) ReadFrom(ctx context.Context, afterEventID id.EventID, limit int) ([]event.Event, error)

// Close flushes the active segment and closes the manifest.
func (j *Journal) Close() error

// Compile-time assertions
var _ event.SeekableJournal = (*Journal)(nil)
var _ io.Closer = (*Journal)(nil)
```

**NOT implemented** (Parquet is immutable):

- `event.EventSink.Save` — no per-stream optimistic concurrency
- `event.EventSink.AppendBatch` — events are append-only to the global journal, not per-stream
- `event.EventSource.Load` — no per-stream read (Parquet is a global journal)

Consumers pair the Parquet journal with a mutable store (SQLite/Pebble) for
writes. The Parquet journal serves as: (a) cold archive, (b) analytics input,
(c) replay source for projections.

## SeekableJournal Implementation

### ReadAll

1. Load manifest (list of segments sorted by `minTimestamp`)
2. For each segment (sealed then active):
   a. Open `parquet.GenericReader[EventRecord]`
   b. Read all rows into `[]EventRecord`
   c. Convert each `EventRecord` to `event.Event` via `recordToEvent()`
3. Return concatenated slice

### ReadFrom (seek)

1. Load manifest
2. Binary search segments by `afterEventID` timestamp (ULID embedded timestamp)
3. If `afterEventID` falls in a segment's `[minTime, maxTime]`:
   a. Open that segment's reader
   b. Scan to the matching ID (skip rows until ID found)
   c. Collect remaining rows in that segment
   d. Continue with subsequent segments
4. If `limit > 0`, stop after collecting `limit` events
5. Convert and return

**ULID ordering guarantee:** Event IDs are ULIDs (time-sortable). `ReadFrom`
relies on `OccurredAt ASC, ID ASC` ordering — the same invariant used by the SQL
and Pebble journal readers.

## Segment Manifest

```json
{
	"version": 1,
	"segments": [
		{
			"id": 1,
			"file": "segments/seg_000001.parquet",
			"minTimestamp": "2026-07-11T10:00:00Z",
			"maxTimestamp": "2026-07-11T10:05:00Z",
			"minID": "01JX...",
			"maxID": "01JX...",
			"rowCount": 10000,
			"sealed": true
		},
		{
			"id": 2,
			"file": "segments/seg_000002.parquet",
			"minTimestamp": "2026-07-11T10:05:01Z",
			"maxTimestamp": "2026-07-11T10:10:00Z",
			"minID": "01JX...",
			"maxID": "01JX...",
			"rowCount": 10000,
			"sealed": true
		}
	]
}
```

**Atomic update:** Manifest is written to `manifest.json.tmp` then renamed.
On open, if `manifest.json` is missing but segments exist, a recovery scan
rebuilds the manifest from segment file metadata.

## Segment Compaction (Future)

Sealed segments can be compacted via `parquet.MergeRowGroups` to reduce file
count. This is an offline operation — not needed for Phase 1 but the segment
layout supports it without API changes.

## Dependencies

```
storage/parquet/go.mod:
  module github.com/larsartmann/go-cqrs-lite/storage/parquet/v4

  require (
      github.com/larsartmann/go-cqrs-lite/event/v4 v4.x.x
      github.com/larsartmann/go-cqrs-lite/codec/v4 v4.x.x
      github.com/larsartmann/go-error-family v0.9.0
      github.com/parquet-go/parquet-go v0.23.x
  )
```

**No CGO.** `parquet-go` is pure Go. Binary size increase is negligible (~2MB
for the library). This preserves the pure-Go build that uses `modernc.org/sqlite`
and `cockroachdb/pebble`.

## Usage Example

```go
// Pair with a mutable store for writes
sqlStore, _ := sqlite.NewEventStore(db)     // writes (Save, AppendBatch)

// Parquet journal for archival + analytics + replay
parquetJournal, _ := parquet.Open("/var/lib/myapp/journal",
    parquet.WithMaxRowsPerSegment(10000),
)
defer parquetJournal.Close()

// Archive: batch-copy events from SQL to Parquet periodically
events, _ := sqlStore.ReadAll(ctx)
parquetJournal.Append(ctx, events)

// Analytics: DuckDB reads Parquet directly (Phase 2)
//   SELECT event_type, COUNT(*) FROM read_parquet('journal/segments/*.parquet')
//   GROUP BY event_type;

// Replay: projectionhost reads from Parquet journal
host, _ := projectionhost.New(parquetJournal, cpStore)
host.Register(&UserProjection{})
go host.Start(ctx)
```

## Alternatives Considered

### A. Use Parquet as a full Store (Save + Load per stream)

**Rejected.** Parquet files are immutable. Supporting `Save(streamID, events,
expectedVersion)` would require rewriting the entire file on each write, or
maintaining an unflushed in-memory buffer with crash recovery. The segment
approach (global append-only journal) is simpler and matches Parquet's strengths.

### B. Use Arrow IPC instead of Parquet

**Rejected.** Arrow IPC supports appending, but the format is row-oriented
(streams), not columnar. Parquet's columnar storage gives 5-10x compression via
dictionary + ZSTD, which is the primary value for event archival.

### C. Use flat files with JSON Lines (JSONL)

**Rejected.** JSONL is uncompressed and has no column pruning. A 1GB JSONL log
compresses to ~100-200MB in Parquet with column pruning and filter pushdown.
JSONL is also slower to scan (no row-group skipping).

## Risks

| Risk                           | Mitigation                                          |
| ------------------------------ | --------------------------------------------------- |
| Manifest corruption            | Atomic rename + recovery scan from segment metadata |
| Active segment data loss       | `WithSyncWrites(true)` fsyncs after each flush      |
| Large replay reads             | `limit` parameter bounds memory; stream in batches  |
| Schema evolution (new columns) | Parquet schema evolution via `parquet.Convert`      |
| `parquet-go` API breakage      | Pin to v0.23.x; vendor if needed                    |

## Phase 2/3 Preview

| Phase | Module           | What It Adds                                      | CGO? |
| ----- | ---------------- | ------------------------------------------------- | ---- |
| 2     | `storage/duckdb` | DuckDB dialect: `SQLViewStore` over Parquet       | Yes  |
| 3     | `stack/duckdb`   | Preset: Parquet journal + DuckDB materializations | Yes  |

Phase 2 requires CGO (`duckdb-go` statically links the C++ engine). This is the
primary architectural decision for Phase 2 — whether to introduce CGO to the
build. The research doc recommends keeping DuckDB as an optional module that
consumers opt into, NOT a core dependency.
