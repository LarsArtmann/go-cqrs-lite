# Tombstone to Domain Events — Migration Guide

**Related ADR:** [ADR-0114](../adr/0114-tombstone-as-domain-event.md)
**Status:** The old tombstone metadata API (`MarkTombstone`, `DetectTombstone`,
`TombstoneStatus`) is **Deprecated — functional in v4, removal planned for v5**.
It has NOT been removed. This guide shows how to express deletion as a domain
event with the APIs that actually ship today.

> **Honesty note (2026-08-16):** an earlier version of this guide described a
> `DeletePolicy` rename plus `DeleteTypes`/`RebirthTypes`/`WithDeleteTypes`
> fields and options. That API was landed on unreleased master on 2026-08-10
> and reverted on 2026-08-12 before any tag was cut — it never shipped. See the
> correction entry at the top of `CHANGELOG.md`. Everything below compiles
> against the released API.

## Why Migrate?

The old tombstone API (`MarkTombstone`, `DetectTombstone`, `TombstoneStatus`)
marks deletion by mutating/copying event metadata. This cuts against the
principle that event streams are immutable write-ahead logs — a fact that
happened cannot be un-happened.

Domain-event-based deletion keeps streams pure: deletion is just another event
in the stream, carrying its own immutable payload. The event type itself is the
signal — no metadata markers needed.

## Before (Deprecated Tombstone API — still works in v4)

```go
// Command handler: mark deletion via tombstone metadata
func Delete(cmd DeleteCommand) ([]event.Event, error) {
    evt, err := event.New("task.deleted", cmd.ID, "Task", version, TaskDeleted{Reason: cmd.Reason})
    if err != nil {
        return nil, err
    }
    marked, err := event.MarkTombstone(evt) // Deprecated: copies evt, sets metadata mark
    if err != nil {
        return nil, err
    }
    return []event.Event{marked}, nil
}

// Projection: check tombstone status
status := event.DetectTombstone(events) // Deprecated: inspects last event's metadata
if status.IsTombstoned() {
    // skip or remove from projection
}
```

## After (Domain Events)

### 1. Command Handler — Emit a Deletion Event

```go
func Delete(cmd DeleteCommand) ([]event.Event, error) {
    evt, err := event.New("task.deleted", cmd.ID, "Task", version,
        TaskDeleted{Reason: cmd.Reason})
    if err != nil {
        return nil, err
    }
    return []event.Event{evt}, nil // pure, immutable — no metadata mutation
}
```

The event type `"task.deleted"` IS the signal.

### 2. Metaengine Projection — Remove on Delete

The metaengine already implements ADR-0114 natively: fold handlers bind to
event types, and `Remove` hard-removes the entry from the projection.

```go
metaengine.OnRecordTyped(
    "task.deleted",
    projectionadapter.EventWithID[TaskDeletedPayload]{},
    metaengine.Remove[TaskView](),
)
```

See `example/taskmanager/metaengine.go` for a full working example.

### 3. stack.Materialize — Match Event Types in Your Handlers

`Materialize` has **no** `DeleteTypes`/`RebirthTypes` fields (that API never
shipped). Two patterns work today:

**a) Domain-event style (recommended):** route deletion through the regular
`OnUpdate` handler and branch on `evt.Type()` yourself. `Materialize` delivers
every event to your handlers; the event type is visible to you even though the
framework does not switch on it.

```go
mat := stack.Materialize[TaskView, TaskID]{
    Store:        kvStore,
    KeyFromEvent: func(evt event.Event) (TaskID, error) { /* ... */ },
    OnCreate:     func(ctx context.Context, evt event.Event) (*TaskView, error) { /* ... */ },
    OnUpdate: func(ctx context.Context, evt event.Event, existing *TaskView) (*TaskView, error) {
        switch evt.Type() {
        case "task.deleted": // deletion is a domain event
            existing.Deleted = true
            return existing, nil
        case "task.restored":
            existing.Deleted = false
            return existing, nil
        default:
            return applyUpdate(existing, evt)
        }
    },
}
```

**b) Metadata-triggered (deprecated, still wired):** `OnTombstone`/`OnRebirth`
fire only when an event carries `event.TombstoneMark` metadata (set by the
deprecated `event.MarkTombstone`/`MarkRebirth`, or by
`listing.StatusMiddleware` below). This is the pre-ADR-0114 path; use it only
while migrating.

### 4. listing — Stream Status via StatusMiddleware

`listing` has no `WithDeleteTypes` reader option (never shipped). The event-type
→ status bridge that DOES ship is `listing.StatusMiddleware`: install it on the
publish bus and it marks tombstone/rebirth metadata on events **whose type
matches your configured lists**. `ListWithStatus` then reports the status.

```go
reader := listing.NewInMemoryStreamReader(journal)

// Type-driven detection: one place to declare which event types mean delete/restore.
bus.UsePublish(listing.StatusMiddleware(
    []event.Type{"task.deleted", "task.archived"},  // delete types
    []event.Type{"task.restored"},                  // rebirth types
))

// Streams whose last event is a delete type report Status = Tombstoned.
page, err := reader.ListWithStatus(ctx, listing.ListOptions{
    Type:     "Task",
    Tombstone: listing.TombstoneExclude, // hide deleted (default)
    // Tombstone: listing.TombstoneInclude, // show all
    // Tombstone: listing.TombstoneOnly,    // only deleted
})
```

## API Mapping (old → what to do instead)

| Old (Deprecated, still functional in v4)      | Migration path                                                    |
| --------------------------------------------- | ----------------------------------------------------------------- |
| `event.MarkTombstone(evt)`                    | Emit `"entity.deleted"` event directly                            |
| `event.MarkRebirth(evt)`                      | Emit `"entity.restored"` event directly                           |
| `event.DetectTombstone(events)`               | Check last event type: `events[len-1].Type() == "entity.deleted"` |
| `event.TombstoneStatus` + `IsTombstoned()`    | Your own logic over event types (or `listing.StatusMiddleware`)   |
| `listing.StatusMiddleware(deletes, rebirths)` | Keep — it IS the type-driven bridge today                         |
| `kv.TombstoneQuerier` / `QueryByTombstone`    | Unchanged — server-side SQL filtering                             |

**There is no `DeletePolicy` rename.** The `listing.TombstonePolicy`
(`TombstoneExclude`/`TombstoneInclude`/`TombstoneOnly`) and
`stack.TombstonePolicy` (`IncludeTombstoned`/`ExcludeTombstoned`/`OnlyTombstoned`)
constants are the shipped names. See the CHANGELOG correction (2026-08-16).

## Filtering Deleted Records

### stack.Materialize.List

```go
// ExcludeTombstoned is the default — hides records implementing IsTombstoned() == true.
results, _ := mat.List(ctx, stack.ExcludeTombstoned)

// OnlyTombstoned returns only soft-deleted records.
deleted, _ := mat.List(ctx, stack.OnlyTombstoned)

// IncludeTombstoned returns everything.
all, _ := mat.List(ctx, stack.IncludeTombstoned)
```

`FilterTombstoned(results, policy)` applies the same policy to an in-memory
slice. For server-side filtering (SQL stores with a tombstone column), implement
`kv.TombstoneQuerier` on your store. `List` pushes the filter to SQL
automatically when the store supports it.

### listing StreamReader

```go
// Use the fluent builder:
page, _ := listing.NewListBuilder(reader).
    OfType("Task").
    IncludeDeleted(). // or OnlyDeleted()
    List(ctx)
```

## What Is Still Missing (ADR-0114 backlog)

ADR-0114 is accepted as the direction, but its framework-level machinery —
`Materialize.DeleteTypes`/`RebirthTypes`, `listing.WithDeleteTypes`,
type-triggered `OnTombstone` — is not implemented. Until it lands, deletion
detection is either yours (pattern a above) or metadata-backed via
`StatusMiddleware` (deprecated path). Track progress in `TODO_LIST.md`.
