# listing — Aggregate Listing Read Model

[![Go Reference](https://pkg.go.dev/badge/github.com/larsartmann/go-cqrs-lite/listing/v4.svg)](https://pkg.go.dev/github.com/larsartmann/go-cqrs-lite/listing/v4)

CQRS read model for aggregate listing and tombstone (soft-delete) management.

```bash
go get github.com/larsartmann/go-cqrs-lite/listing/v4
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
| `StreamListing`   | Lightweight identity: ID, Type, Version, EventCount, LastEventAt       |
| `StreamStatus`    | Pairs a `StreamListing` with its `Status` (Active/Deleted)             |
| `Page[T]`         | Cursor-based page: `Items []T` + `HasMore bool` (no TotalCount)        |
| `ListOptions`     | Query params: Type (required), After (cursor), Limit, DeletePolicy     |
| `DeletePolicy`    | `DeleteExclude` (default), `DeleteInclude`, `DeleteOnly`               |
| `StreamReader`    | Interface: `List` and `ListWithStatus`                                 |

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

## Delete Type Configuration

Configure which event types signal stream deletion (ADR-0114):

```go
reader := listing.NewInMemoryStreamReader(store,
    listing.WithDeleteTypes("user.deleted", "order.cancelled"),
)
```

Streams whose last event matches a delete type show `StatusDeleted`. Without
configuration, all streams are `StatusActive`.

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
    if item.Status.IsDeleted() {
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

## Stream Status

The `listing.Status` enum has two states:

| Status          | Value | Meaning                                      |
| --------------- | ----- | -------------------------------------------- |
| `StatusActive`  | 0     | Stream is live                               |
| `StatusDeleted` | 1     | Stream's last event is a configured delete type |

Detection uses the **last event** in the stream. Configure delete types via
`WithDeleteTypes(...)` on the reader.

## StreamReader Interface

```go
type StreamReader interface {
    List(ctx context.Context, opts ListOptions) (*Page[StreamListing], error)
    ListWithStatus(ctx context.Context, opts ListOptions) (*Page[StreamStatus], error)
}
```

Implementations: `InMemoryStreamReader`, `SQLStreamReader`.

## Dependencies

| Dependency                            | Purpose                          |
| ------------------------------------- | -------------------------------- |
| [event](../event/README.md)           | Event types, journal                  |
| [id](../id/README.md)                 | AggregateID                      |
| [memory](../storage/memory/README.md) | In-memory reader for testing     |

## Test Coverage

93.2% across all files. BDD test suite covers:

- SQL reader: pagination, cursor, tombstone filtering, empty results, error paths
- Aggregate projection: table creation, event handling, upsert, tombstone detection
- Integration: full Projection → SQL Reader pipeline with pagination
- ListBuilder: PageSize clamping (zero, max), cursor at end, cross-type listing
- In-memory reader: active/tombstoned filtering, pagination, empty journal
- StatusMiddleware: tombstone, rebirth, passthrough

## Related Modules

- [**projection**](../projection/README.md) — Register `AggregateProjection` with the runner to populate the reader
- [**storage**](../storage/README.md) — SQL-backed `AggregateReader` for PostgreSQL/SQLite
- [**event**](../event/README.md) — Tombstone detection and event types
- [**id**](../id/README.md) — `AggregateID` type
- [**memory**](../storage/memory/README.md) — `InMemoryAggregateReader` for tests
