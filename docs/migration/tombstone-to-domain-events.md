# Tombstone to Domain Events — Migration Guide

**Related ADR:** [ADR-0114](../adr/0114-tombstone-as-domain-event.md)
**Status:** The old tombstone metadata API (`MarkTombstone`, `DetectTombstone`, `TombstoneStatus`) has been **removed**. This guide shows how to migrate to event-type-based deletion.

## Why Migrate?

The old tombstone API (`MarkTombstone`, `DetectTombstone`, `TombstoneStatus`) mutated event metadata after creation. This violated the principle that event streams are immutable write-ahead logs — a fact that happened cannot be un-happened.

Domain-event-based deletion keeps streams pure: deletion is just another event in the stream, carrying its own immutable payload. The event type itself is the signal — no metadata markers needed.

## Before (Removed Tombstone API)

```go
// Command handler: mark deletion via tombstone metadata
func Delete(cmd DeleteCommand) []event.Event {
    evt, _ := event.NewEvent("task.deleted", streamID, "Task", version, payload)
    marked, _ := event.MarkTombstone(evt)        // REMOVED: mutated metadata
    return []event.Event{marked}
}

// Projection: check tombstone status
status := event.DetectTombstone(events)          // REMOVED
if status.IsTombstoned() {
    // skip or remove from projection
}
```

## After (Domain Events)

### 1. Command Handler — Emit a Deletion Event

```go
func Delete(cmd DeleteCommand) []event.Event {
    evt, _ := event.NewEvent("task.deleted", streamID, "Task", version,
        TaskDeleted{Reason: cmd.Reason})
    return []event.Event{evt}                    // pure, immutable
}
```

No metadata mutation. The event type `"task.deleted"` IS the signal.

### 2. Metaengine Projection — Remove on Delete

```go
metaengine.OnRecordTyped(
    string(evtTaskDeleted),
    projectionadapter.EventWithID[TaskDeletedPayload]{},
    metaengine.Remove[TaskView](),               // hard-remove from the Map ADT
)
```

See `example/taskmanager/metaengine.go` for a full working example.

### 3. stack.Materialize — Event-Type-Based Detection

```go
mat := stack.Materialize[TaskView, TaskID]{
    Store:        kvStore,
    KeyFromEvent: func(evt event.Event) (TaskID, error) { ... },
    OnCreate:     func(ctx, evt) (*TaskView, error) { ... },
    OnUpdate:     func(ctx, evt, prev *TaskView) (*TaskView, error) { ... },

    // ADR-0114: deletion events trigger OnTombstone via type matching
    DeleteTypes:  []event.Type{"task.deleted"},
    OnTombstone: func(ctx context.Context, evt event.Event, existing *TaskView) (*TaskView, error) {
        existing.Tombstoned = true
        return existing, nil
    },

    // Optional: rebirth (undo deletion)
    RebirthTypes: []event.Type{"task.restored"},
    OnRebirth: func(ctx context.Context, evt event.Event, existing *TaskView) (*TaskView, error) {
        existing.Tombstoned = false
        return existing, nil
    },
}
```

`DeleteTypes` and `RebirthTypes` are event-type slices. When an event's type
matches, the corresponding callback fires. No metadata inspection.

### 4. listing — Stream Status via Delete Types

```go
reader := listing.NewInMemoryStreamReader(journal,
    listing.WithDeleteTypes("task.deleted", "task.archived"),
)

// Streams whose last event is a delete type show StatusDeleted.
// Use DeletePolicy to control visibility:
page, _ := reader.ListWithStatus(ctx, listing.ListOptions{
    Type:         "Task",
    DeletePolicy: listing.DeleteExclude,           // hide deleted (default)
    // DeletePolicy: listing.DeleteInclude,         // show all
    // DeletePolicy: listing.DeleteOnly,            // only deleted
})
```

## API Mapping

| Old (Removed)                                    | New (Domain Events)                                               |
| ------------------------------------------------ | ----------------------------------------------------------------- |
| `event.MarkTombstone(evt)`                       | Emit `"entity.deleted"` event directly                            |
| `event.MarkRebirth(evt)`                         | Emit `"entity.restored"` event directly                           |
| `event.DetectTombstone(events)`                  | Check last event type: `events[len-1].Type() == "entity.deleted"` |
| `event.TombstoneStatus` + `IsTombstoned()`       | Custom logic based on event types                                 |
| `listing.TombstonePolicy` (Exclude/Include/Only) | `listing.DeletePolicy` (DeleteExclude/DeleteInclude/DeleteOnly)   |
| `stack.TombstonePolicy` (Include/Exclude/Only)   | `stack.DeletePolicy` (IncludeDeleted/ExcludeDeleted/OnlyDeleted)  |
| `stack.FilterTombstoned(results, policy)`        | `stack.FilterDeleted(results, policy)`                            |
| `stack.Materialize` triggered by metadata        | `stack.Materialize` triggered by `DeleteTypes`/`RebirthTypes`     |
| `kv.TombstoneQuerier` / `QueryByTombstone`       | Unchanged — still works for server-side SQL filtering             |

## listing/ Module

`listing.StatusMiddleware` (auto-marks events as tombstones from type lists) is
no longer needed — the event type itself communicates the intent. Instead of
configuring middleware, use `listing.WithDeleteTypes(...)` on the reader.

## Filtering Deleted Records

### stack.Materialize.List

```go
// ExcludeDeleted is the default — hides records where IsTombstoned() returns true.
results, _ := mat.List(ctx, stack.ExcludeDeleted)

// OnlyDeleted returns only soft-deleted records.
deleted, _ := mat.List(ctx, stack.OnlyDeleted)

// IncludeDeleted returns everything.
all, _ := mat.List(ctx, stack.IncludeDeleted)
```

For server-side filtering (SQL stores with a tombstone column), implement
`kv.TombstoneQuerier` on your store. The `List` method automatically pushes
the filter to SQL when available.

### listing StreamReader

```go
// Use the fluent builder:
page, _ := listing.NewListBuilder(reader).
    OfType("Task").
    IncludeDeleted().    // or OnlyDeleted()
    List(ctx)
```
