# ADR-0110: Snapshot Storage — New SnapshotBackend Interface

## Status

Accepted

## Date

2026-08-04

## Context

Snapshots need LoadAtVersion(streamID, version) — find the snapshot at or below
a given version. The Map ADT can only store the latest snapshot per key;
versioned keys require backward scanning that isn't a natural Map operation.

## Decision

New `SnapshotBackend` interface at the ADT level (like StreamLogBackend).
Engines implement it. Provides Save, Load, LoadAtVersion, Delete.

## Rationale

- A dedicated interface gives engines the right operations (versioned lookup).
- Enables decider.LoadAtVersion to use nearest snapshot then replay events.
- Consistent with the ADT-per-concern model.

## Consequences

- Each engine implements SnapshotBackend (small, well-defined surface).
- Time-travel/restore via snapshots becomes a uniform engine capability.
