# Consistency Model

> **go-cqrs-lite** is a CQRS + Event Sourcing library. This document defines
> the consistency guarantees consumers can rely on, the patterns for handling
> eventual consistency, and the explicit out-of-scope areas.

---

## Scope: Single-Process

go-cqrs-lite is a **single-process library**, not a distributed database.
All consistency guarantees apply within a single process:

- **Event Store** writes are synchronous within `Repository.Execute`.
- **Projections** run in-process via `projectionhost.Host`.
- **No distributed consensus** — no Raft, no Paxos, no CRDTs.

Distributed consistency (multi-process write coordination, cross-process
projection synchronization) is **explicitly out of scope**. Consumers building
distributed systems must layer their own coordination (e.g., a shared SQL
database as the event store, NATS for cross-process messaging).

---

## Write Path: Strong Consistency

The write path (command → event store) is **strongly consistent** within a
single process:

```
Client → Command Handler → decider.Repository.Execute()
                                ↓
                         1. Load events from store (synchronous)
                         2. Decide (pure function: state + command → events)
                         3. Save events to store (synchronous, atomic)
                         4. Publish events to bus (async if bus is async)
```

**Guarantees:**

- After `Execute` returns successfully, events are **persisted** in the event store.
- Version numbers are sequential and gap-free per stream.
- Optimistic concurrency: `Execute` loads the current version, and the store
  rejects concurrent writes that conflict (AppendBatch with version check).

**No guarantee:**

- Events may not yet be **published** to subscribers when `Execute` returns
  (depends on the bus implementation — `MemoryBus` is synchronous; external
  buses like Watermill may be async).

---

## Read Path: Eventual Consistency

The read path (projection → read model) is **eventually consistent**:

```
Event Store → projectionhost.Host → Projection.Handle()
                                        ↓
                                 Read Model (KV, SQL View, Graph)
                                        ↓
                                 Query Handler → Client
```

**Projection lag** is the time between an event being appended to the store
and the projection processing it. Lag depends on:

1. **Worker batch size** — larger batches amortize per-event overhead but
   increase latency.
2. **Event volume** — under sustained write load, the projection may fall behind.
3. **Handler speed** — slow projection handlers (e.g., HTTP calls) create lag.

**Monitoring lag:**

```go
// Total lag across all projections (max)
gauge.Set(float64(host.LagDuration().Milliseconds()))

// Per-projection lag
for name, lag := range host.LagPerProjection() {
    gauge.WithLabelValues(name).Set(float64(lag.Milliseconds()))
}
```

### SSE Delivery: Encoding Projection

Browser SSE clients receive JSON even when events are stored as CBOR. Wire
`transport/http.CBORToJSONTransform` once via `WithPayloadTransform` — the
transform converts CBOR payloads to JSON per-client-per-event on the wire.
This is a schema-free encoding projection, not a consistency concern: the
event data is identical, only the wire format changes.

---

## Read-After-Write: WaitForVersion

For request/response flows where the client writes a command and immediately
reads the result, projection lag can cause stale reads. The
`decider.Repository.WaitForVersion` helper solves this:

```go
// Execute a command.
result, _ := repo.Execute(ctx, streamID, decider, cmd)

// Wait until the event store shows the new version.
// In single-process setups, this returns immediately (the store is synchronous).
events, _ := repo.WaitForVersion(ctx, streamID, "User", result.NewVersion,
    decider.WithWaitTimeout(2*time.Second))
```

**How it works:**

1. Polls `store.LoadFromVersion(ctx, ref, targetVersion-1)` every 10ms.
2. Returns when events at or after the target version are visible.
3. Times out after 2s (configurable via `WithWaitTimeout`).
4. Respects the caller's context deadline (shortest deadline wins).

**Important:** `WaitForVersion` waits for the **event store**, not the
projection. After `WaitForVersion` returns, the projection may still be
lagging. For read-your-writes through a projection, combine with
`CheckStaleness` (below) or poll the read model directly.

**When to use it:**

- Distributed setups where the event store has read replicas (the write
  goes to the primary, reads go to a replica with replication lag).
- Cross-process scenarios where another process wrote the event.

**When you DON'T need it:**

- Single-process with MemoryStore, SQLite, or Pebble — the write is
  immediately visible after `Execute` returns.

---

## Bounded Staleness: CheckStaleness

For read paths that go through projections, `projectionhost.Host.CheckStaleness`
rejects reads when projection lag exceeds a threshold:

```go
// Before serving a read from a projection-backed read model:
if err := host.CheckStaleness(5 * time.Second); err != nil {
    // Projection is > 5s behind — return 503 or serve stale data with a warning.
    http.Error(w, "service temporarily unavailable", http.StatusServiceUnavailable)
    return
}

// Projection is fresh enough — serve the read.
```

**Per-projection variant:**

```go
if err := host.CheckProjectionStaleness("users", 5*time.Second); err != nil {
    // Only the "users" projection is stale.
}
```

**Semantics:**

- `maxStaleness <= 0` disables the check (always returns nil).
- Lag == 0 (no events processed yet) is treated as fresh — the projection
  hasn't had a chance to fall behind.
- Returns `ErrProjectionStale` (Transient) when lag exceeds the threshold.

---

## Idempotency: Exactly-Once Command Processing

For at-least-once delivery (e.g., message queues that redeliver), the
idempotency middleware prevents duplicate command execution:

```go
store := idempotency.NewMemoryStore(5 * time.Minute)
defer store.Close()
cmdDisp.Use(middleware.CommandIdempotency(store, 10*time.Minute, nil))
```

**Single-process:** `MemoryStore` — in-memory map with mutex-based atomicity.

**Multi-process:** `idempotency/sqlstore` — SQL-backed store with
`INSERT ... ON CONFLICT DO UPDATE WHERE` for atomic check-and-record:

```go
store, _ := sqlstore.NewSQLiteStore(ctx, db)
// or: store, _ := sqlstore.NewPostgresStore(ctx, db)
cmdDisp.Use(middleware.CommandIdempotency(store, 10*time.Minute, nil))
```

The SQL store guarantees exactly-one-winner semantics even across multiple
processes sharing the same database, because the UPSERT statement is atomic
at the row level.

---

## Summary Table

| Path                        | Consistency       | Mechanism                               | Helper           |
| --------------------------- | ----------------- | --------------------------------------- | ---------------- |
| Command → Event Store       | Strong (sync)     | `Repository.Execute`                    | —                |
| Event Store → Bus           | Depends on bus    | `MemoryBus` (sync) or Watermill (async) | —                |
| Event Store → Projection    | Eventual          | `projectionhost.Host` batch drain       | `LagDuration()`  |
| Write → Read (same process) | Immediate         | Store is synchronous                    | `WaitForVersion` |
| Write → Read (distributed)  | Eventual          | Read replica replication lag            | `WaitForVersion` |
| Read model freshness        | Bounded staleness | Projection lag check                    | `CheckStaleness` |
| Command dedup               | Exactly-once      | Idempotency store atomic claim          | `CheckAndRecord` |

---

## What This Library Does NOT Provide

- **Distributed transactions** across multiple aggregates/services.
- **Cross-process ordering** guarantees (events within a stream are ordered;
  across streams, order is undefined).
- **Linearizability** or sequential consistency across processes.
- **Saga/orchestration** — multi-step workflows emerge from bus subscriptions
  - command dispatch (see `example/taskmanager`).
- **Read replica management** — the library works with whatever database
  the consumer configures; replication lag is the database's responsibility.
