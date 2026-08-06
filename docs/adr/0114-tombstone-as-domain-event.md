# ADR-0114: Tombstones Are Domain Events, Not Mutable Metadata

**Date:** 2026-08-06
**Status:** Accepted
**Related:** ADR-0111 (Record type — removes Tombstone from CommonMetadata)

## Context

The current system uses mutable tombstone metadata on events to signal soft
deletion. `event.DetectTombstone(events)` inspects metadata;
`event.MarkTombstone(evt)` mutates event metadata to set a tombstone marker.

This violates the fundamental principle that **WAL logs (event streams) must
contain only immutable data**. An event is a fact — it happened, it cannot be
un-happened. Mutating metadata after the fact to signal "this thing was deleted"
corrupts the integrity of the event stream.

## Decision

**Tombstones are domain events. The event stream carries only immutable facts.
No tombstone metadata exists on records.**

### How Deletion Works

Deletion is expressed as a domain event in the same stream:

```
Stream: User/01J...
  Record 1: {Type: "user.created", ...}     // fact: user was created
  Record 2: {Type: "user.deleted", ...}     // fact: user was deleted
  Record 3: {Type: "user.restored", ...}    // fact: user was restored (optional)
```

The projection fold handler decides what "deleted" means:

```go
On(UserDeleted{}, func(r Record) Skip {
    // Remove from projection, or mark as inactive
    return Skip{}
})
```

### What This Replaces

| Current                                                          | New                                                           |
| ---------------------------------------------------------------- | ------------------------------------------------------------- |
| `event.DetectTombstone(events)` → Active/Tombstoned/Undetermined | Fold handler checks event types (UserDeleted, OrderCancelled) |
| `event.MarkTombstone(evt)` → mutates metadata                    | Not needed — deletion events are immutable records            |
| `event.TombstoneStatus` metadata field                           | Does not exist                                                |
| Tombstone filtering in store reads                               | Projection concern — fold handlers handle it                  |

### Design Principles

1. **The event stream is pure.** Every record is an immutable fact. No metadata
   mutation, no tombstone markers, no status fields.

2. **Deletion semantics are domain-specific.** What "deleted" means depends on
   the domain: a deleted user is different from a cancelled order. The projection
   handler encodes this knowledge.

3. **The planner understands deletion events.** When the ES-native planner
   (ADR-0112) sees a fold handler for UserDeleted that returns Skip, it knows
   this projection should remove the entry. This is enough information for
   auto-projection (ADR-0116).

4. **Schema version is the only exception to immutability.** It travels with the
   record for upcasting purposes but is set once at creation time and never
   changed.

## Alternatives Considered

### A. Separate tombstone stream

**Rejected.** A separate stream that marks other streams as deleted adds
indirection without value. The deletion event IS the immutable record — it
belongs in the same stream as the entity it deletes.

### B. Tombstone as a special event type flag

**Rejected.** This is just metadata with a different name. The event type
("user.deleted") already signals the intent; no additional flag is needed.

## Consequences

- **Positive:** Event streams are 100% immutable. No metadata mutation. The WAL
  is pure truth.
- **Positive:** Deletion semantics are explicit in domain code, not hidden in
  generic metadata machinery.
- **Positive:** The planner can reason about deletion by inspecting event types,
  not by parsing tombstone metadata.
- **Negative:** Breaking change — removes DetectTombstone, MarkTombstone,
  TombstoneStatus from the event API. Consumers must update projection handlers.
- **Negative:** The `listing/` module (tombstone detection, StatusMiddleware)
  must be refactored to use event-type-based detection.
