# ADR-0122: WithClock — Injectable Time for CRDT Testing

## Status

Accepted

## Context

The Iroh replicated engine (`metaengine/irohengine`) uses last-writer-wins (LWW)
semantics for `MapSet` operations. The LWW timestamp determines which write
"wins" when two nodes concurrently set different values for the same key. In
production, timestamps come from `time.Now()`.

Testing CRDT convergence requires **deterministic timestamps**. If two test
nodes both call `time.Now()`, the nanosecond ordering is non-deterministic —
the test's outcome depends on OS scheduling, GC pauses, and CPU cache behavior.
This makes convergence tests flaky and impossible to reason about.

The same problem applies to any time-based conflict resolution: PN-Counter
epochs, OR-Set add timestamps, and log sequencing all need controllable time
in tests.

## Decision

Introduce a `Clock` interface and inject it via `WithClock`:

```go
// Clock provides the current time. Used for LWW timestamps in CRDT convergence.
type Clock interface {
    Now() time.Time
}
```

The default implementation is `realClock{}` which delegates to `time.Now()`.
Tests inject a `manualClock` that starts at a fixed epoch and advances
explicitly via `Advance(d time.Duration)`.

```go
// In tests:
clock := newManualClock(time.Unix(0, 0))
cluster := newTwoNodeClusterWithClock(t, clock, ...)

// Node A sets value at t=0
cluster.NodeA.MapSet(ctx, "users", "alice", `{"v":1}`)

// Advance time, then node B overwrites
clock.Advance(1 * time.Second)
cluster.NodeB.MapSet(ctx, "users", "alice", `{"v":2}`)

// Converge: node A sees v=2 (later timestamp wins)
```

This pattern is already implemented in `metaengine/irohengine/options.go` and
used throughout the convergence test suite.

## Consequences

- **Deterministic CRDT tests**: convergence outcomes are reproducible across
  runs, CI environments, and `-race` detector overhead.
- **No production behavior change**: `realClock` is the zero-value default.
  Consumers who never call `WithClock` get wall-clock time.
- **Extensible**: future time-based features (TTL expiry, deadline timers) can
  accept the same `Clock` interface for testability.
- **Scope limit**: `WithClock` only affects CRDT LWW resolution. It does NOT
  affect event timestamps, command timestamps, or projection processing — those
  use their own `time.Now()` calls and are tested via different strategies
  (event ordering by version, not by wall-clock time).
