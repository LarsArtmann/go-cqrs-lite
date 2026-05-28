# Journal Naming Proposal

**Date:** 2026-05-28
**Context:** Rename `GlobalLoader` / `PositionalLoader` to better reflect their role in event sourcing

---

## Problem

`GlobalLoader` and `PositionalLoader` are generic, action-oriented names that don't reveal their role in the system.

- "Global" is vague — global to what?
- "Loader" is an action, not a thing — it describes what you do, not what it is
- "Positional" describes the mechanism, not the concept

They also don't connect to standard event sourcing terminology, making the codebase harder to understand for anyone familiar with the pattern.

---

## Proposed Names

### `GlobalLoader` → `Journal`

"Journal" is the standard event sourcing term for the complete, ordered, append-only log of all domain events. It immediately signals:

- Cross-aggregate (not scoped to one stream)
- Time-ordered (by `OccurredAt`)
- Read-only (at this interface level)
- Complete (all events ever)

```go
// Before
type GlobalLoader interface {
    LoadAll(ctx context.Context) ([]Event, error)
}

// After
type Journal interface {
    ReadAll(ctx context.Context) ([]Event, error)
}
```

At the call site:
```go
// Before
events, err := globalLoader.LoadAll(ctx)

// After
events, err := journal.ReadAll(ctx)
```

### `PositionalLoader` → `SeekableJournal`

"Seekable" implies you can jump to a position in the log — exactly what checkpoint-based replay does. It extends `Journal` with the ability to resume from a specific point.

```go
// Before
type PositionalLoader interface {
    GlobalLoader
    LoadAllFromPosition(ctx context.Context, afterEventID id.EventID, limit int) ([]Event, error)
}

// After
type SeekableJournal interface {
    Journal
    ReadFrom(ctx context.Context, afterEventID id.EventID, limit int) ([]Event, error)
}
```

At the call site:
```go
// Before
events, err := positionalLoader.LoadAllFromPosition(ctx, checkpoint, 100)

// After
events, err := journal.ReadFrom(ctx, checkpoint, 100)
```

---

## Method Name Changes

| Current | Proposed | Rationale |
|---|---|---|
| `LoadAll()` | `ReadAll()` | Direct, matches `io.Reader` convention |
| `LoadAllFromPosition()` | `ReadFrom()` | "Read from position X" — natural language, concise |

---

## Why These Names Work

| Current | Proposed | Why Better |
|---|---|---|
| `GlobalLoader` | `Journal` | Standard ES term; implies ordered, append-only, complete |
| `PositionalLoader` | `SeekableJournal` | "Seekable" = can jump to position; extends `Journal` naturally |
| `LoadAll()` | `ReadAll()` | Matches Go's `io.Reader` naming convention |
| `LoadAllFromPosition()` | `ReadFrom()` | Concise; "from" implies the starting position |

---

## The Relationship

```
Store (Sink + Source)    ← per-aggregate, version-ordered
    │
    ├── Save/Load        ← "User" aggregate #123
    │
Journal                  ← cross-aggregate, time-ordered
    │
    ├── ReadAll()        ← all events ever
    └── ReadFrom()       ← events after checkpoint (SeekableJournal)
```

**Store** = the write model and per-aggregate read model.  
**Journal** = the read model for replay, audit, and projections.

This naming also opens the door for future extensions naturally:

| Future Interface | Role |
|---|---|
| `StreamableJournal` | Cursor-based / streaming reads without loading all into memory |
| `FilterableJournal` | `ReadByType(eventType)`, `ReadByAggregate(aggType, aggID)` |
| `ArchivableJournal` | `ArchiveBefore(eventID)` for cold storage |
| `CountableJournal` | `Count()`, `CountFrom(eventID)` for metrics |

---

## Migration Plan

### Phase 1: Add New Interfaces

Introduce `Journal` and `SeekableJournal` alongside existing interfaces. Mark old names as deprecated.

```go
type Journal interface {
    ReadAll(ctx context.Context) ([]Event, error)
}

// Deprecated: use Journal
 type GlobalLoader = Journal
```

### Phase 2: Update Call Sites

Update `projection.Runner`, `memory.MemoryStore`, `storage.SQLEventStore`, tests, and examples to use new names.

### Phase 3: Remove Aliases

After one release cycle, remove the deprecated type aliases.

---

## Files to Change

| File | Change |
|---|---|
| `core/event/store.go` | Rename interfaces, add deprecated aliases |
| `memory/store_load.go` | Rename methods, update comments |
| `storage/event_store.go` | Rename methods, update comments |
| `projection/runner.go` | Update `Runner` field and constructor |
| `projection/runner_test.go` | Update test helpers |
| `testhelpers/fake_store.go` | Add `ReadAll`/`ReadFrom` methods |
| `docs/research/*.md` | Update references |

---

## Open Question

Should `Journal` include `io.Closer`?

Currently `GlobalLoader` does not have lifecycle management. `Store` does (via `io.Closer`). Since `Journal` is a read-only view of the same underlying storage, it may or may not need its own `Close()`. If the journal is backed by the same database connection as the store, closing the store closes the journal. If it's a separate read replica connection, it needs its own lifecycle.

Options:
1. `Journal` without `Closer` — simple, assumes shared lifecycle
2. `Journal` with `io.Closer` — explicit, supports separate connections
3. `Journal` without `Closer`, but implementations may optionally satisfy it

Recommendation: Option 3 (keep `Journal` minimal, let implementations optionally add `Closer` if they have their own resources).
