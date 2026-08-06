# ADR-0111: Extract Record Type as Shared Base for Commands and Events

**Date:** 2026-08-06
**Status:** Accepted
**Supersedes:** Partially supersedes the type hierarchy implied by ADR-0031 (metadata extraction)
**Related:** ADR-0062 addendum (dep boundary), ADR-0112 (ES-native metaengine), ADR-0117 (command lifecycle)

## Context

Commands and Events are both append-only records in streams. They share the same
structural shape: a type, a payload, a stream identity, a version, and metadata.
Currently they have parallel type hierarchies with duplicated concepts:

- `event.Event` and `command.Command` both carry type, payload, stream refs
- `event.Metadata` and `command.Metadata` both carry correlation IDs, timestamps
- The metaengine planner sees both as `any` — it cannot reason about either

The duplication causes inconsistent metadata handling and prevents the planner
from understanding what it's projecting.

## Decision

**Extract a shared `Record` type as the base for both Commands and Events.**

### Record Type

```go
type Record struct {
    Type       string         // "user.created" / "create_user"
    Payload    []byte         // encoded payload (codec-stamped)
    StreamID   StreamRef      // "User/01J..."
    StreamType string         // "User"
    Version    int64          // position in stream (1-indexed)
    MetaData   CommonMetadata // shared metadata (see below)
}
```

### CommonMetadata

```go
type CommonMetadata struct {
    CorrelationID    string       // links a chain of related records
    CausationID      string       // what caused this record (command ID, request ID)
    ActorID          string       // who or what produced this (user ID, service name)
    ClientCreatedAt  time.Time    // client clock (may lie; for offline-first)
    ServerReceivedAt time.Time    // server clock (before store.Save)
    ServerStoredAt   time.Time    // DB acknowledgment (not a lie: what the DB told us)
    SchemaVersion    int          // payload schema version (enables upcasting)
}
```

### Design Principles

1. **Events and Commands have identical Record shape.** The only difference is
   conceptual: Events are facts (post-decision, immutable truth), Commands are
   intents (pre-decision, may be rejected). The type system does not need to
   distinguish them — `StreamType` ("Command" vs "Event" vs "User") is sufficient.

2. **Commands add nothing to Record.** No IntentStatus, no RetryCount, no
   lifecycle fields. A Command is a single immutable record expressing intent.
   Its lifecycle (accepted, rejected, retried, dead-lettered) is tracked by
   Events in a separate lifecycle stream (ADR-0117).

3. **All timestamps are timezone-aware (time.Time).** Three honest timestamps:
   - `ClientCreatedAt` — when the user/action initiated this (client clock, may lie)
   - `ServerReceivedAt` — when the server got it (server clock, trustworthy)
   - `ServerStoredAt` — when the DB acknowledged the write (not "committed" —
     we report what the DB told us, not what it actually did internally)

4. **MetaData field name, CommonMetadata type name.** The struct field is named
   `MetaData` (read naturally as "this record's metadata"). The type is named
   `CommonMetadata` (emphasizing it is shared across all record types).

5. **SchemaVersion is the only non-immutable-friendly field.** It travels with
   the record because upcasting needs it at decode time. It is set once at record
   creation and never changed — the record is immutable, but different schema
   versions of the same logical event type can coexist in the stream.

### What This Replaces

| Current | New |
|---------|-----|
| `event.Event` (= `*ImmutableEvent`) | `Record` with `StreamType: "Event"` |
| `event.Metadata` (Tracing + CustomData + Tombstone) | `CommonMetadata` (no Tombstone — see ADR-0114) |
| `command.BasicCommand` | `Record` with `StreamType: "Command"` |
| `command.Metadata` | `CommonMetadata` |
| `metadata.Tracing` (CorrelationID, CausationID) | Folded into `CommonMetadata` |
| `metadata.CustomData[K]` | Stays as-is (generic side-channel metadata, orthogonal) |

### Migration Path

This is a foundational type change. The migration is phased:

1. **Phase 1:** Define `Record` and `CommonMetadata` in a new `record/` module
   (Tier 0 primitive). Both `event/` and `command/` depend on it.
2. **Phase 2:** metaengine depends on this module. Fold handlers receive `Record`
   instead of `any`.
3. **Phase 3:** Remove duplicate metadata types from `event/` and `command/`.
4. **Phase 4:** Remove `Tombstone` from event metadata (ADR-0114).

## Alternatives Considered

### A. Keep event.Event as the base

**Rejected.** "event/ IS the base" was explicitly vetoed. Events are facts;
Commands are intents. Making one embed the other conflates domain concepts. A
neutral `Record` base avoids implying one is more fundamental than the other.

### B. Unified Message type with no distinction

**Rejected.** While the Record shape is identical, the domain semantics matter:
a decider processes Commands and emits Events. The `StreamType` field preserves
this distinction without requiring separate types.

### C. Metadata interface pattern

**Rejected.** An interface with a `Common()` method adds indirection without
value. The concrete `CommonMetadata` struct is simpler, faster (no virtual
dispatch), and sufficient for both record types.

## Consequences

- **Positive:** One metadata type, one Record shape, one mental model. No more
  parallel hierarchies with subtle differences.
- **Positive:** The planner receives typed Records and can inspect Type,
  StreamType, Version, and MetaData to make smarter projection decisions.
- **Positive:** Multi-timestamp support enables offline-first, clock skew handling,
  and sync conflict resolution without schema changes later.
- **Negative:** Foundational refactor — touches event/, command/, metadata/, and
  every consumer. Phased migration mitigates this.
- **Negative:** `time.Time` is larger than the current string-based timestamps.
  Acceptable: 3 x 24 bytes = 72 bytes per record, negligible vs payload size.
