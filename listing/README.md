# listing — Aggregate Listing Read Model

[![Go Reference](https://pkg.go.dev/badge/github.com/larsartmann/go-cqrs-lite/listing/v3.svg)](https://pkg.go.dev/github.com/larsartmann/go-cqrs-lite/listing/v3)

CQRS read model for aggregate listing and tombstone (soft-delete) management.

```bash
go get github.com/larsartmann/go-cqrs-lite/listing/v3
```

## Overview

The `listing` module provides:

- **Aggregate listing** with cursor-based pagination
- **Tombstone detection** — tri-state status: Active, Tombstoned, Undetermined
- **Rebirth support** — undo soft-deletes via rebirth events
- **Projection-backed SQL reader** for production
- **In-memory reader** for testing
- **Bus middleware** for automatic tombstone/rebirth metadata marking

This module is **read-only**. It never writes events. It queries via `event.Journal` (cross-aggregate) or a projection table.

## Types

| Type              | Purpose                                                                |
| ----------------- | ---------------------------------------------------------------------- |
| `AggregateRef`    | Lightweight identity: ID, Type, Version, EventCount, LastEventAt       |
| `AggregateStatus` | Pairs an `AggregateRef` with its `TombstoneStatus`                     |
| `Page[T]`         | Cursor-based page: `Items []T` + `HasMore bool` (no TotalCount)        |
| `ListOptions`     | Query params: Type (required), After (cursor), Limit, Tombstone policy |
| `TombstonePolicy` | `TombstoneExclude` (default), `TombstoneInclude`, `TombstoneOnly`      |
| `AggregateReader` | Interface: `List` and `ListWithStatus`                                 |

## Setup

### In-memory (testing)

```go
import (
    "github.com/larsartmann/go-cqrs-lite/memory"
    "github.com/larsartmann/go-cqrs-lite/listing"
)

store := memory.NewMemoryStore()
reader := listing.NewInMemoryAggregateReader(store)

page, err := listing.NewListBuilder(reader).
    OfType("User").
    PageSize(20).
    List(ctx)
```

### SQL (production)

```go
import (
    "database/sql"
    "github.com/larsartmann/go-cqrs-lite/listing"
)

// Create projection (creates table if not exists)
proj, err := listing.NewAggregateProjection(db, "cqrs_")

// Register with projection runner
runner.Register(proj)

// Query the projection table
reader := listing.NewSQLAggregateReader(db, "cqrs_")
page, err := listing.NewListBuilder(reader).
    OfType("User").
    PageSize(20).
    List(ctx)
```

### Table Schema

`NewAggregateProjection` auto-creates the table:

```sql
CREATE TABLE IF NOT EXISTS {prefix}listing_aggregates (
    aggregate_type   TEXT NOT NULL,
    aggregate_id     TEXT NOT NULL,
    version          INT  NOT NULL,
    event_count      INT  NOT NULL DEFAULT 0,
    last_event_at    TIMESTAMP NOT NULL,
    tombstone_status INT  NOT NULL DEFAULT 0,
    PRIMARY KEY (aggregate_type, aggregate_id)
);
```

## Tombstone Middleware

Auto-mark tombstone and rebirth events on publish:

```go
bus.UsePublish(listing.StatusMiddleware(
    []event.Type{"user.deleted", "order.cancelled"},    // tombstone types
    []event.Type{"user.reactivated", "order.restored"},  // rebirth types
))
```

Unmatched events pass through unchanged.

## Listing with Status

```go
// Active users only (default)
page, _ := listing.NewListBuilder(reader).
    OfType("User").
    List(ctx)

// Include deleted with status
statusPage, _ := listing.NewListBuilder(reader).
    OfType("User").
    IncludeDeleted().
    ListWithStatus(ctx)

for _, item := range statusPage.Items {
    if item.Status.IsTombstoned() {
        fmt.Printf("Deleted: %s\n", item.Ref.ID)
    }
}

// Only deleted
page, _ := listing.NewListBuilder(reader).
    OfType("User").
    OnlyDeleted().
    List(ctx)
```

## Cursor Pagination

No offset-based pagination — append-only logs make counts stale and expensive. Use cursor-based pagination instead:

```go
page1, _ := listing.NewListBuilder(reader).
    OfType("User").
    PageSize(20).
    List(ctx)

if page1.HasMore {
    page2, _ := listing.NewListBuilder(reader).
        OfType("User").
        PageSize(20).
        After(page1.Items[len(page1.Items)-1].ID).
        List(ctx)
}
```

`PageSize` is clamped to `[1, 100]`. Zero defaults to 20.

## Tombstone Status

The `event.TombstoneStatus` enum has three states:

| Status                  | Value | Meaning                                      |
| ----------------------- | ----- | -------------------------------------------- |
| `TombstoneActive`       | 0     | Aggregate is live                            |
| `TombstoneTombstoned`   | 1     | Aggregate is soft-deleted                    |
| `TombstoneUndetermined` | 2     | No metadata found (no middleware configured) |

Detection uses the **last event** in the listing. Rebirth takes precedence.

```go
// Detect from event listing
status := event.DetectTombstone(events)

// Mark manually (usually done by middleware)
marked, _ := event.MarkTombstone(evt)
marked, _ := event.MarkRebirth(evt)
```

## AggregateReader Interface

```go
type AggregateReader interface {
    List(ctx context.Context, opts ListOptions) (*Page[AggregateRef], error)
    ListWithStatus(ctx context.Context, opts ListOptions) (*Page[AggregateStatus], error)
}
```

Implementations: `InMemoryAggregateReader`, `SQLAggregateReader`.

## Dependencies

| Dependency                       | Purpose                          |
| -------------------------------- | -------------------------------- |
| [event/v2](../event/README.md)   | Event types, tombstone detection |
| [id/v2](../id/README.md)         | AggregateID                      |
| [memory/v2](../memory/README.md) | In-memory reader for testing     |

## Test Coverage

93.2% across all files. BDD test suite covers:

- SQL reader: pagination, cursor, tombstone filtering, empty results, error paths
- Aggregate projection: table creation, event handling, upsert, tombstone detection
- Integration: full Projection → SQL Reader pipeline with pagination
- ListBuilder: PageSize clamping (zero, max), cursor at end, cross-type listing
- In-memory reader: active/tombstoned filtering, pagination, empty journal
- StatusMiddleware: tombstone, rebirth, passthrough

## Related Modules

- [**projection/v2**](../projection/README.md) — Register `AggregateProjection` with the runner to populate the reader
- [**storage/v2**](../storage/README.md) — SQL-backed `AggregateReader` for PostgreSQL/SQLite
- [**event/v2**](../event/README.md) — Tombstone detection and event types
- [**id/v2**](../id/README.md) — `AggregateID` type
- [**memory/v2**](../memory/README.md) — `InMemoryAggregateReader` for tests
