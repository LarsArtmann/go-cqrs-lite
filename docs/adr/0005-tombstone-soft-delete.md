# ADR-0005: Tombstone Soft-Delete Pattern

**Date:** 2026-05-10
**Status:** Accepted

## Context

Aggregate deletion in event-sourced systems is problematic: events are immutable by design, yet consumers need to know when an aggregate is "deleted." Physical deletion breaks event streams and audit trails.

## Decision

Adopt a **tombstone soft-delete** pattern:

- Aggregates are never physically deleted from the event store
- Tombstone status is detected from event metadata (`TombstoneStatus: Active/Tombstoned/Undetermined`)
- `listing/` module provides `DetectTombstone()` and `StatusMiddleware` for automatic status tracking
- No `Delete` method exists on the Store interface

## Consequences

- **+** Full audit trail preserved — every aggregate lifecycle is traceable
- **+** Tombstone detection works with any event store implementation
- **+** Listing API naturally filters tombstoned aggregates
- **-** Storage grows unbounded for tombstoned aggregates (compaction is a separate concern)
