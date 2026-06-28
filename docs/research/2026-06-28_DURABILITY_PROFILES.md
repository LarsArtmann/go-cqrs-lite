# Durability Profiles Design

> **Status:** Design — 2026-06-28
> **Scope:** Future enhancement for event store durability control

## Problem

All event stores currently write with maximum durability (synchronous fsync).
This is the safe default but creates unnecessary latency for:

- Batch replay scenarios (projections catching up)
- Non-critical audit logs (command/query stores)
- Development and testing

## Proposed Design

### Durability Option

```go
type Durability int

const (
    // DurabilitySync fsyncs after every write (current default).
    DurabilitySync Durability = iota
    // DurabilityBatchedSync fsyncs every N writes (configurable).
    DurabilityBatchedSync
    // DurabilityAsync never fsyncs (fastest, data loss on crash).
    DurabilityAsync
)
```

### Per-Store Configuration

```go
// Pebble
store, _ := pebble.NewStore(db, logger, pebble.WithDurability(DurabilityBatchedSync))
backend, _ := pebble.Open(dir, pebble.WithBatchSyncInterval(100*time.Millisecond))

// SQL
store, _ := storage.NewSQLEventStore(db, storage.WithDurability(DurabilityBatchedSync))
```

### Current State

- Pebble KV adapter: `WithKVSyncWrites()` exists (toggle Sync vs no-Sync)
- Pebble event/snapshot/checkpoint stores: write with `pebble.Sync` always
- SQL stores: rely on database/sql's default (connection-level setting)
- Memory stores: no durability (by design)

### Implementation Plan

1. Add `Durability` type + `WithDurability()` option to Pebble stores
2. Add `PRAGMA synchronous` mapping for SQLite (FULL/NORMAL/OFF)
3. Add `synchronous_commit` mapping for PostgreSQL (on/off/local)
4. Benchmark the three profiles against the existing default

### Tradeoffs

| Profile     | Crash Safety       | Write Latency | Use Case                      |
| ----------- | ------------------ | ------------- | ----------------------------- |
| Sync        | Zero data loss     | Highest       | Production event store        |
| BatchedSync | ≤ N writes lost    | Medium        | Projection replay, audit logs |
| Async       | All in-flight lost | Lowest        | Dev/test, ephemeral data      |
