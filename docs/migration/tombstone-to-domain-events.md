# Tombstone to Domain Events — Migration Guide

**Related ADR:** [ADR-0114](../adr/0114-tombstone-as-domain-event.md)
**Status:** Tombstone API is deprecated as of v4. Removal planned for v5.

## Why Migrate?

The tombstone API (`MarkTombstone`, `DetectTombstone`, `TombstoneStatus`) mutates
event metadata after creation. This violates the principle that event streams are
immutable write-ahead logs — a fact that happened cannot be un-happened.

Domain-event-based deletion keeps streams pure: deletion is just another event in
the stream, carrying its own immutable payload.

## Before (Tombstone Metadata)

```go
// Command handler: mark deletion via tombstone metadata
func Delete(cmd DeleteCommand) []event.Event {
    evt, _ := event.NewEvent("task.deleted", streamID, "Task", version, payload)
    marked, _ := event.MarkTombstone(evt)        // mutates metadata
    return []event.Event{marked}
}

// Projection: check tombstone status
status := event.DetectTombstone(events)
if status.IsTombstoned() {
    // skip or remove from projection
}
```

## After (Domain Events)

```go
// Command handler: emit a deletion event — no metadata mutation
func Delete(cmd DeleteCommand) []event.Event {
    evt, _ := event.NewEvent("task.deleted", streamID, "Task", version,
        TaskDeleted{Reason: cmd.Reason})
    return []event.Event{evt}                    // pure, immutable
}

// Projection fold: handle the deletion event type directly
metaengine.On(TaskDeleted{}, func(e TaskDeleted) metaengine.Skip {
    return metaengine.Skip{}                     // remove from projection
})

// Or in a Materialize builder:
mat := stack.Materialize[TaskView, TaskID]{
    OnCreate: func(ctx, evt) (*TaskView, error) { ... },
    OnUpdate: func(ctx, evt, prev *TaskView) (*TaskView, error) { ... },
    // Handle deletion as a regular event — set Tombstoned or return nil
}
```

## API Mapping

| Old (Tombstone)                                  | New (Domain Events)                                               |
| ------------------------------------------------ | ----------------------------------------------------------------- |
| `event.MarkTombstone(evt)`                       | Emit `"entity.deleted"` event directly                            |
| `event.MarkRebirth(evt)`                         | Emit `"entity.restored"` event directly                           |
| `event.DetectTombstone(events)`                  | Check last event type: `events[len-1].Type() == "entity.deleted"` |
| `event.TombstoneStatus` + `IsTombstoned()`       | Custom logic based on event types                                 |
| `listing.TombstonePolicy` (Exclude/Include/Only) | Filter in your projection handler                                 |
| `stack.Materialize.OnTombstone` / `OnRebirth`    | Handle deletion events in `OnUpdate` or a custom fold             |
| `kv.TombstoneQuerier` / `QueryByTombstone`       | Add a `Deleted bool` column to your view struct                   |

## listing/ Module

The `listing.StatusMiddleware` auto-marks events as tombstones based on event
type lists. With domain events, this is unnecessary — the event type itself
communicates the intent. Instead of configuring `StatusMiddleware`, simply
emit the appropriate event types.

## stack.Materialize

`Materialize.OnTombstone` and `OnRebirth` callbacks are triggered when
`md.Tombstone` metadata is present. To migrate:

1. Add a `Tombstoned bool` field to your view struct
2. Handle deletion events in `OnUpdate` (or a dedicated fold)
3. Set `Tombstoned = true` in the handler
4. Filter using `kv.ViewQuery` conditions on the `Tombstoned` column

## Timeline

- **v4 (current):** Tombstone API deprecated with `// Deprecated:` directives.
  All existing code continues to work. IDE warnings guide consumers to migrate.
- **v5 (future):** Tombstone API removed. `Metadata.Tombstone` field removed.
  `DetectTombstone`, `MarkTombstone`, `MarkRebirth` removed. `TombstoneStatus`,
  `TombstoneMark` types removed. Consumers must migrate before upgrading to v5.
