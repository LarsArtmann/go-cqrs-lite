# stream

CQRS read model for aggregate listing and tombstone (soft-delete) management.

## Overview

The `stream` module provides:

- **Aggregate listing** with cursor-based pagination
- **Tombstone detection** — tri-state status: Active, Tombstoned, Undetermined
- **Rebirth support** — undo soft-deletes via rebirth events
- **Projection-backed SQL reader** for production
- **In-memory reader** for testing
- **Bus middleware** for automatic tombstone/rebirth metadata marking

This module is **read-only**. It never writes events. It queries via `event.Journal` (cross-aggregate) or a projection table.

## Setup

### In-memory (testing)

```go
import (
    "github.com/larsartmann/go-cqrs-lite/memory"
    "github.com/larsartmann/go-cqrs-lite/stream"
)

store := memory.NewMemoryStore()
reader := stream.NewInMemoryAggregateReader(store)

page, err := stream.NewListBuilder(reader).
    OfType("User").
    PageSize(20).
    List(ctx)
```

### SQL (production)

```go
import (
    "database/sql"
    "github.com/larsartmann/go-cqrs-lite/stream"
)

// Create projection (creates table if not exists)
proj, err := stream.NewAggregateProjection(db, "cqrs_")

// Register with projection runner
runner.Register(proj)

// Query the projection table
reader := stream.NewSQLAggregateReader(db, "cqrs_")
page, err := stream.NewListBuilder(reader).
    OfType("User").
    PageSize(20).
    List(ctx)
```

## Tombstone Middleware

Auto-mark tombstone and rebirth events on publish:

```go
bus.UsePublish(stream.StatusMiddleware(
    []event.Type{"user.deleted", "order.cancelled"},    // tombstone types
    []event.Type{"user.reactivated", "order.restored"},  // rebirth types
))
```

## Listing with Status

```go
// Active users only (default)
page, _ := stream.NewListBuilder(reader).
    OfType("User").
    List(ctx)

// Include deleted with status
statusPage, _ := stream.NewListBuilder(reader).
    OfType("User").
    IncludeDeleted().
    ListWithStatus(ctx)

for _, item := range statusPage.Items {
    if item.Status.IsTombstoned() {
        fmt.Printf("Deleted: %s\n", item.Ref.ID)
    }
}

// Only deleted
page, _ := stream.NewListBuilder(reader).
    OfType("User").
    OnlyDeleted().
    List(ctx)
```

## Cursor Pagination

```go
page1, _ := stream.NewListBuilder(reader).
    OfType("User").
    PageSize(20).
    List(ctx)

if page1.HasMore {
    page2, _ := stream.NewListBuilder(reader).
        OfType("User").
        PageSize(20).
        After(page1.Items[len(page1.Items)-1].ID).
        List(ctx)
}
```

## Tombstone Status

The `event.TombstoneStatus` enum has three states:

| Status                  | Meaning                                   |
| ----------------------- | ----------------------------------------- |
| `TombstoneActive`       | Aggregate is live                         |
| `TombstoneTombstoned`   | Aggregate is soft-deleted                 |
| `TombstoneUndetermined` | Status cannot be determined (no metadata) |

```go
// Mark manually (usually done by middleware)
marked, _ := event.MarkTombstone(evt)
marked, _ := event.MarkRebirth(evt)

// Detect from event stream
status := event.DetectTombstone(events)
```

## Dependencies

- `github.com/larsartmann/go-cqrs-lite/core` (event types)
