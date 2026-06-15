# ADR-0020: Performance Optimization Patterns

**Date:** 2026-06-14  
**Status:** Accepted

## Context

During the v2.3.0 → v2.4.0 performance optimization sprint, we identified and applied several patterns for eliminating allocations on hot paths. One optimization (T3/T4) was **dead code** — a type assertion that never matched the constructor users actually call. This ADR documents the patterns that work and the anti-patterns to avoid.

## Decision

### Pattern 1: Cache at the Integration Boundary

**Problem:** A public interface method (`EventTypes() []Type`) is called on every event in a hot loop. The method defensively clones its return value. The clone is correct for external callers but wasteful when the caller is an internal integration point that doesn't mutate the slice.

**Anti-pattern (dead code):**
```go
// WRONG: type assertion to a specific concrete type
func subscribesTo(p Projection, eventType Type) bool {
    if bp, ok := p.(*builtProjection); ok { // dead if user used event.NewProjection
        return slices.Contains(bp.eventTypes, eventType)
    }
    return slices.Contains(p.EventTypes(), eventType) // allocates
}
```

**Solution:** Cache the result at registration time — the integration boundary:
```go
type projectionEntry struct {
    projection Projection
    eventTypes []Type // cached once at Register() time
}

func (r *Runner) Register(p Projection) error {
    r.projections = append(r.projections, projectionEntry{
        projection: p,
        eventTypes: p.EventTypes(), // single clone at registration
    })
    return nil
}

func subscribesTo(types []Type, eventType Type) bool {
    return len(types) == 0 || slices.Contains(types, eventType)
}
```

**Why:** This works for ALL projection implementations, not just one concrete type. The cost is one `EventTypes()` call per projection at registration, not per event.

### Pattern 2: Pre-compute Middleware Chains

**Problem:** Rebuilding a middleware chain on every `Publish()` call allocates N closures.

**Solution:** Rebuild only when middleware changes:
```go
func (b *MemoryBus) Use(mw EventMiddleware) {
    b.mu.Lock()
    b.publishMiddleware = append(b.publishMiddleware, mw)
    b.rebuildPublisherChain()
    b.mu.Unlock()
}

// Hot path — zero allocations
func (b *MemoryBus) Publish(ctx context.Context, evt Event) error {
    b.mu.RLock()
    handler := b.cachedPublisher
    b.mu.RUnlock()
    return handler(ctx, evt)
}
```

### Pattern 3: Lazy Initialization of Rarely-Used Fields

**Problem:** A struct always allocates a map even when the map is never used.

**Solution:** Return zero-value, call `EnsureCustom()` before first write:
```go
func NewMetadata() Metadata { return Metadata{} } // no map

func EnsureCustom(m *Metadata) {
    if m.Custom == nil {
        m.Custom = make(map[MetadataKey]string)
    }
}
```

### Pattern 4: Lock-Free Happy Paths

**Problem:** Mutex contention serializes all traffic through a component.

**Solution:** Use atomics for the happy path, keep mutex for state transitions:
```go
type CircuitBreaker struct {
    state       atomic.Int32 // lock-free happy path
    failures    atomic.Int32
    successes   atomic.Int32
    mu          sync.Mutex   // only for state transitions
    lastFailure time.Time
}

func (cb *CircuitBreaker) allow() bool {
    return cb.state.Load() == circuitClosed // single atomic load
}
```

## Consequences

- Public API contracts (defensive cloning) are preserved for external callers
- Internal hot paths use cached/pre-computed values to avoid per-event allocations
- The `Projection` interface still requires `EventTypes()` to clone — a future v3 could add `SubscribesTo(Type) bool` to eliminate this entirely
- Lesson learned: type assertions for fast paths are **dead code** if users create types via different constructors. Always cache at the integration boundary.

## Benchmark Impact

| Optimization | Pattern | Allocs Eliminated |
|---|---|---|
| T15: Runner event type caching | Pattern 1 | 10.5M per 100K events |
| T12: MemoryBus pre-computation | Pattern 2 | 2 per publish |
| T2: Lazy metadata map | Pattern 3 | 1 per event |
| T11: CircuitBreaker atomic | Pattern 4 | N/A (lock-free, 9.4 ns/op) |
