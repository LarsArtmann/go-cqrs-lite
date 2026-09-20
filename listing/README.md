# listing — Aggregate Listing Read Model

[![Go Reference](https://pkg.go.dev/badge/github.com/larsartmann/go-cqrs-lite/listing/v4.svg)](https://pkg.go.dev/github.com/larsartmann/go-cqrs-lite/listing/v4)

CQRS read model for aggregate listing and tombstone (soft-delete) management.

```bash
go get github.com/larsartmann/go-cqrs-lite/listing/v4
```

## Overview

The `listing` module provides:

- **Aggregate listing** with cursor-based pagination
- **Status classification** — tri-state: Active, Tombstoned, Undetermined
- **Domain-event-driven soft deletes** — deletion/restoration are domain events (ADR-0114), classified from the last event's type
- **Projection-backed SQL reader** for production
- **In-memory reader** for testing
- **`StatusClassifier`** — derive per-stream status from delete/rebirth event types, no middleware, no metadata mutation

This module is **read-only**. It never writes events. It queries via `event.Journal` (cross-aggregate) or a projection table.

## Types

| Type              | Purpose                                                                |
| ----------------- | ---------------------------------------------------------------------- |
| `StreamListing`   | Lightweight identity: ID, Type, Version, EventCount, LastEventAt       |
| `StreamStatus`    | Pairs a `StreamListing` with its `Status`                              |
| `Page[T]`         | Cursor-based page: `Items []T` + `HasMore bool` (no TotalCount)        |
| `ListOptions`     | Query params: Type (required), After (cursor), Limit, Tombstone policy |
| `TombstonePolicy` | `TombstoneExclude` (default), `TombstoneInclude`, `TombstoneOnly`      |
| `StreamReader`    | Interface: `List` and `ListWithStatus`                                 |

> The legacy `Aggregate*` spellings (`AggregateListing`, `AggregateStatus`,
> `AggregateReader`) are deprecated aliases of the `Stream*` names — removed
> in v5.

## Setup

### In-memory (testing)

```go
import (
    "github.com/larsartmann/go-cqrs-lite/storage/memory/v4"
    "github.com/larsartmann/go-cqrs-lite/listing/v4"
)

store := memory.NewMemoryStore()

// Classify status from domain event types (ADR-0114) — recommended.
reader := listing.NewInMemoryAggregateReader(store,
    listing.WithStatusClassifier(listing.NewStatusClassifier(
        []event.Type{"user.deleted"},       // deletion events
        []event.Type{"user.reactivated"},   // restoration events
    )))

page, err := listing.NewListBuilder(reader).
    OfType("User").
    PageSize(20).
    List(ctx)
```

## Status Classification (ADR-0114)

Deletion and restoration are domain events. A stream's status follows from
its LAST event's type — a delete type means Tombstoned, a rebirth type means
Active again. No metadata is mutated and no middleware is needed:

```go
classifier := listing.NewStatusClassifier(
    []event.Type{"user.deleted", "order.cancelled"},     // deletion events
    []event.Type{"user.reactivated", "order.restored"},  // restoration events
)

// In-memory reader:
reader := listing.NewInMemoryAggregateReader(store,
    listing.WithStatusClassifier(classifier))

// Classify a single stream's last event directly:
status := classifier.ClassifyLast(lastEvent) // StatusActive / StatusTombstoned
```

> `listing.StatusMiddleware` (which stamped tombstone/rebirth METADATA on
> publish via `event.MarkTombstone`) is **deprecated** — metadata marks
> violate stream immutability and are removed in v5. Derive status from
> event types as shown above.

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

## Status

`listing.Status` is a tri-state enum (numeric values match the deprecated
legacy `event.TombstoneStatus` wire values):

| Status               | Value | Meaning                                             |
| -------------------- | ----- | --------------------------------------------------- |
| `StatusActive`       | 0     | Stream is live                                      |
| `StatusTombstoned`   | 1     | Last event is a deletion event                      |
| `StatusUndetermined` | 2     | No classifier configured (status cannot be derived) |

Classification uses the **last event** in the stream. Restoration takes
precedence (newest event wins).

## StreamReader Interface

```go
type StreamReader interface {
    List(ctx context.Context, opts ListOptions) (*Page[StreamListing], error)
    ListWithStatus(ctx context.Context, opts ListOptions) (*Page[StreamStatus], error)
}
```

Implementations: `InMemoryStreamReader`, `SQLStreamReader`.

## Dependencies

| Dependency                            | Purpose                                    |
| ------------------------------------- | ------------------------------------------ |
| [event](../event/README.md)           | Event types (`event.Type`) for classifiers |
| [id](../id/README.md)                 | AggregateID                                |
| [memory](../storage/memory/README.md) | In-memory reader for testing               |

## Test Coverage

93.2% across all files. BDD test suite covers:

- SQL reader: pagination, cursor, tombstone filtering, empty results, error paths
- Aggregate projection: table creation, event handling, upsert, tombstone detection
- Integration: full Projection → SQL Reader pipeline with pagination
- ListBuilder: PageSize clamping (zero, max), cursor at end, cross-type listing
- In-memory reader: active/tombstoned filtering, pagination, empty journal
- StatusMiddleware: tombstone, rebirth, passthrough

## Related Modules

- [**projection**](../projection/README.md) — Register `StreamProjection` with the runner to populate the reader
- [**storage**](../storage/README.md) — SQL-backed `StreamReader` for PostgreSQL/SQLite
- [**event**](../event/README.md) — Event types consumed by `StatusClassifier`
- [**id**](../id/README.md) — `StreamID` type
- [**memory**](../storage/memory/README.md) — `InMemoryStreamReader` for tests
