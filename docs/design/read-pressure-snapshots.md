# Design Spike: Read-Pressure Snapshot Strategy

**Status:** Proposed
**Module:** `snapshot/`

## Problem

`EveryNEvents` snapshots based on writes (every Nth event saved). But in event sourcing, reads are the expensive path — writes are sequential appends. An aggregate with 5000 events that's read 100 times/sec but rarely written to never gets snapshotted, so every read replays 5000 events.

## Design

```go
type ReadPressureStrategy struct {
    Threshold    int           // snapshot after this many reads since last snapshot
    MinDelta     int           // minimum new events since last snapshot (guard against snapshotting unchanged aggregates)
    MaxInterval  time.Duration // max time between snapshots (even if below Threshold reads)
}

func (s ReadPressureStrategy) ShouldSnapshot(stats SnapshotStats) bool
```

### SnapshotStats

The Repository would track per-aggregate read statistics:

```go
type SnapshotStats struct {
    ReadsSinceSnapshot  int
    EventsSinceSnapshot int
    LastSnapshotAt      time.Time
    CurrentVersion      event.Version
}
```

### Key Design Decisions

1. **MinDelta guard** — Don't snapshot an aggregate that hasn't changed since the last snapshot. `MinDelta=10` means at least 10 new events must exist.
2. **MaxInterval** — Prevents an aggregate that's read slowly (1 read/sec) but never reaches Threshold from never being snapshotted. Default: 5 minutes.
3. **Subsumed by hot-state cache** — If the hot-state cache is enabled, reads don't hit the store at all (after first load). So this strategy has lower payoff when caching is active. Consider only if caching is not viable.
4. **Stats storage** — Read counts can be tracked in-memory (lost on restart) or in a checkpoint store. In-memory is sufficient — cold start just means one uncached read.

### Priority

**Lower than hot-state cache.** The cache eliminates read cost entirely for hot aggregates. This strategy only helps for aggregates that are read frequently but NOT cached (e.g., different aggregate IDs each time, or when caching is disabled).

**Recommendation:** Defer until a consumer reports a read-amplification problem that the hot-state cache doesn't solve.
