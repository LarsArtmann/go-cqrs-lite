# ADR-0006: Sink/Source Split for Event Store Interface

**Status:** Accepted  
**Date:** 2026-05-29

## Context

The `event.Store` interface had grown to include write operations (Save, AppendBatch), read operations (Load, LoadFromVersion, LoadToVersion, LoadToTimestamp), and a Delete method. This violated the Interface Segregation Principle (ISP):

- **Projections** only need to read events, but were forced to depend on write methods.
- **Writers** (command handlers, sagas) only need to append events, but inherited all read methods.
- **Delete** was available on the interface but should never be used in production — event sourcing treats history as immutable.
- Adding Journal (cross-aggregate reads) and SeekableJournal (position-based reads) further bloated the interface.

## Decision

Split `event.Store` into focused interfaces following ISP:

1. **`EventSink`** — Write side: `Save`, `AppendBatch`, `Close`.
2. **`EventSource`** — Read side: `Load`, `LoadFromVersion`, `LoadToVersion`, `LoadToTimestamp`, `Close`.
3. **`Store`** — Composite: `EventSink + EventSource`. All existing implementations satisfy this.
4. **`Journal`** — Cross-aggregate reads: `ReadAll`.
5. **`SeekableJournal`** — Position-based: `ReadFrom(ctx, afterEventID, limit)`.
6. **`BackwardsSource`** — Reverse version order: `LoadBackwards`.
7. **`TransactionalSink`** — Atomic save + outbox: `SaveWithOutbox`.

**Remove `Delete`** entirely from the Store interface. History is immutable. Soft-delete is handled via tombstone metadata.

### Deprecation path

- `BackwardsLoader = BackwardsSource` (type alias)
- `TransactionalStore` interface kept with deprecation comment
- `GlobalLoader` / `PositionalLoader` deprecated in favor of `Journal` / `SeekableJournal`

### Tombstone soft-delete

Instead of hard-delete, introduce `TombstoneStatus` (Active / Tombstoned / Undetermined) with:

- `DetectTombstone(events)` — O(1) on last event metadata
- `MarkTombstone(evt)` / `MarkRebirth(evt)` — set metadata keys

## Consequences

**Positive:**

- Consumers depend only on what they use (projections → EventSource, commands → EventSink).
- Delete is impossible at the type level — enforced by the compiler.
- Clear migration path: all existing code using `Store` continues to work.
- Journal/SeekableJournal enable projection replay without loading all events.

**Negative:**

- More interfaces to learn (7 vs 1).
- Deprecated aliases must be maintained until v2.0.
- Consumers must type-assert for TransactionalSink (`if ts, ok := sink.(TransactionalSink); ok`).

## Naming collision resolution

`event.Source` already existed as a string type (for event provenance). Using `Source` as an interface name would conflict. Resolved by using `EventSink`/`EventSource` — the `Event` prefix avoids ambiguity and matches the package domain.
