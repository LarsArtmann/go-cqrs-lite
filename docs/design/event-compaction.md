# Design Spike: Event Stream Compaction / Log Truncation

**Status:** Raw Idea — No Design Yet
**Module:** `event/`, `storage/`

## Problem

Event stores grow unboundedly. An aggregate with 100K events consumes significant storage even though only the latest state matters. Consumers need strategies to compact or truncate old events without losing correctness.

## Approaches

### 1. Snapshot-Based Truncation

After a snapshot is taken, events before the snapshot version are theoretically redundant (they've been folded into the snapshot state). The store could truncate them.

**Risk:** Projections that haven't caught up to the snapshot version will miss events. Replay from scratch becomes impossible.

**Mitigation:** Only truncate events that are older than ALL projection checkpoints. This requires coordination between the event store and checkpoint store.

### 2. Tombstone-Based Compaction

When an aggregate is tombstoned and stays tombstoned for a configurable retention period (e.g., 30 days), all its events can be deleted (or archived).

**Risk:** Audit trail loss. Some domains require permanent event retention.

### 3. Tiered Storage

- Hot tier: last N events per aggregate (fast storage: SSD, memory)
- Cold tier: archived events (slow storage: S3, GCS)

Load hot tier first; fetch cold tier only for deep replay.

**Complexity:** High. Requires a tier-aware store implementation.

### 4. Event-Level Compaction (like Kafka log compaction)

Keep only the latest event per key for certain event types (e.g., "user.profile.updated" — only the latest matters). State-snapshot events can be compacted; delta events cannot.

**Risk:** Requires event-type metadata declaring whether an event type is "compactable." Breaks pure event sourcing semantics.

## Recommendation

**Defer.** No consumer has reported storage growth as a problem. When they do, snapshot-based truncation with checkpoint coordination is the safest first step.
