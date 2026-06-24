# Design Spike: Hot-State Cache for Decider

**Status:** Proposed
**Module:** `decider/`

## Problem

In event sourcing, loading an aggregate requires replaying its entire event stream (or loading from a snapshot + delta events). For hot aggregates (commanded 100+ times/sec), this replay cost dominates. The existing `singleflight.Group` coalesces concurrent loads, but sequential loads still pay the full replay cost.

## Design

```go
type cacheConfig[State any] struct {
    enabled   bool
    capacity  int           // max entries (LRU eviction)
    ttl       time.Duration // optional max age
}

func WithHotStateCache[State any](capacity int) RepositoryOption[State]
```

### Implementation

The cache sits above `loadFromSnapshot`/`loadFromStore` in `Repository[State]`:

```
Execute(ctx, aggID, decider, cmd)
  → cache.get(aggID)  ──hit──→  use cached state + version
  │                                    ↓
  │ miss                               ↓
  → singleflight.Do(aggID)             ↓
    → loadFromSnapshot + delta events  ↓
    → fold to state                     ↓
    → cache.set(aggID, state, version) ←┘
  → decide(cmd, state) → events
  → store.Save(events, expectedVersion=version)
  → cache.update(aggID, newEvents)  // write-through: fold events onto cached state
```

### Key Design Decisions

1. **Write-through** — On successful `Save`, fold the new events onto the cached state. No read-after-write penalty.
2. **Version-keyed** — Cache entry stores `(state, version)`. If version mismatch on next load (stale cache), evict and reload.
3. **Optimistic concurrency safe** — `Save` uses `expectedVersion` for optimistic locking. A stale cache entry can cause one failed Save, which is caught and triggers a cache invalidation + retry.
4. **Above singleflight** — singleflight handles concurrent bursts (N goroutines calling Load simultaneously → 1 load). The cache handles sequential bursts (same goroutine calling Load N times → 1 load after first). They compose: first load fills cache + singleflight, subsequent sequential loads hit cache directly.
5. **Capacity bound (LRU)** — Uses `maypok86/otter` (already a dependency via `kv.Cache`) for a lock-free LRU. Configurable capacity, no TTL by default (version-keyed eviction is sufficient).

### When NOT to enable

- Cold aggregates (loaded rarely) — pays memory cost with no benefit
- Aggregates with external mutation (not through this Repository) — cache would be stale
- Memory-constrained environments

### Profile Before Building

Snapshot + page-cache-resident events already make sequential loads cheap (~microseconds for <1000 events). This cache only pays off for:
- Aggregates with very long event streams (10K+ events)
- Aggregates commanded at high frequency (100+ ops/sec)
- Scenarios where snapshot frequency is low (EveryNEvents with large N)

**Recommendation:** Implement behind `WithHotStateCache` option, default off. Consumers profile and enable per-aggregate.
