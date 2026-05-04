# Outbox Transaction Co-Participation API Design

**Status:** Design | **Date:** 2026-05-04

## Problem

Event store `Save` and outbox `Append` run in **separate transactions**. If the process crashes between them, events are persisted but never published — lost events with no recovery path.

```
repository.Execute()
  ├── store.Save(events)       ← tx #1 (owns transaction)
  ├── outbox.Append(events)    ← tx #2 (separate transaction)
  └── (crash here → events saved, never published)
```

## Design Goals

1. **Atomicity**: Events + outbox entry in the same database transaction
2. **Backward compatibility**: Existing code without outbox works unchanged
3. **Database agnostic interface**: Core doesn't know about SQL
4. **Opt-in**: Consumers who don't need outbox see zero API change

## Current Architecture

```
event.Store.Save(ctx, aggType, aggID, events, expectedVersion) error
event.Outbox.Append(ctx, events) error
```

Both open their own transactions. No way to share a transaction between them.

## Proposed Solution: TransactionalStore Interface

### Option A: Transaction-Aware Store (Recommended)

Add a new optional interface that stores can implement. When present, repositories use it instead of the two-step approach.

```go
// TransactionalStore extends Store with atomic save+outbox append.
//
// Implementations MUST guarantee that SaveWithOutbox persists events
// AND appends to the outbox within a single database transaction.
// If either operation fails, the entire transaction rolls back.
type TransactionalStore interface {
    Store

    // SaveWithOutbox atomically persists events and appends them to the outbox.
    // The outbox parameter is provided by the caller — the implementation
    // extracts the underlying SQL outbox (or equivalent) and participates
    // in the same transaction.
    SaveWithOutbox(
        ctx context.Context,
        aggType AggregateType,
        aggID id.AggregateID,
        events []Event,
        expectedVersion Version,
        outbox Outbox,
    ) error
}
```

### Repository Integration

```go
// In aggregate/repository.go and decider/decider.go:

func (r *Repository) save(ctx context.Context, ...) error {
    if r.outbox != nil {
        // Check if store supports atomic save+outbox
        if ts, ok := r.store.(TransactionalStore); ok {
            return ts.SaveWithOutbox(ctx, aggType, aggID, events, expectedVersion, r.outbox)
        }
        // Fallback: separate transactions (existing behavior)
        err := r.store.Save(ctx, aggType, aggID, events, expectedVersion)
        if err != nil {
            return err
        }
        return r.outbox.Append(ctx, events)
    }
    return r.store.Save(ctx, aggType, aggID, events, expectedVersion)
}
```

### SQL Implementation (storage module)

```go
// In storage/transactional_store.go:

type TransactionalEventStore struct {
    db     *sql.DB
    store  *SQLEventStore
    outbox *SQLOutbox
}

func (s *TransactionalEventStore) SaveWithOutbox(
    ctx context.Context,
    aggType event.AggregateType,
    aggID id.AggregateID,
    events []event.Event,
    expectedVersion event.Version,
    outbox event.Outbox,
) error {
    tx, err := s.db.BeginTx(ctx, nil)
    if err != nil {
        return err
    }

    defer tx.Rollback() //nolint:errcheck // no-op after commit

    // 1. Insert events (re-use existing query, but on tx)
    err = s.saveEvents(tx, aggType, aggID, events, expectedVersion)
    if err != nil {
        return err
    }

    // 2. Append to outbox (re-use existing query, but on tx)
    err = s.appendOutbox(tx, events)
    if err != nil {
        return err
    }

    return tx.Commit()
}
```

### Option B: Transaction Factory (Alternative)

Expose a transaction factory from the store, letting callers wrap multiple operations.

```go
type TransactionProvider interface {
    // BeginTx starts a database transaction.
    // Returns a context that carries the transaction.
    BeginTx(ctx context.Context) (context.Context, error)

    // CommitTx commits the transaction stored in the context.
    CommitTx(ctx context.Context) error

    // RollbackTx rolls back the transaction stored in the context.
    RollbackTx(ctx context.Context) error
}
```

**Rejected because**: Leaks database transaction semantics into the core library. Couples core to SQL concepts. Harder to test and mock.

### Option C: Composite Interface (Alternative)

```go
type StoreWithOutbox interface {
    Store
    Outbox
}
```

**Rejected because**: Merges two distinct concerns (event persistence + outbox relay) into one interface. Violates ISP. Doesn't work when outbox and store use different databases.

## Why Option A

| Criterion           | Option A (TransactionalStore) | Option B (Tx Factory) | Option C (Composite) |
| ------------------- | ----------------------------- | --------------------- | -------------------- |
| Database-agnostic   | ✅ Interface lives in core    | ❌ Leaks SQL to core  | ✅                   |
| ISP compliant       | ✅ Optional interface         | ✅                    | ❌ Merges concerns   |
| Backward compatible | ✅ Existing code unchanged    | ⚠️ New pattern        | ⚠️ Breaking          |
| Testable            | ✅ Easy to mock               | ❌ Complex mock       | ✅                   |
| Multiple DBs        | ✅ Outbox param is separate   | ❌ Single tx scope    | ❌ Same interface    |

## Migration Path

### Phase 1: Interface (Non-breaking)

1. Add `TransactionalStore` interface to `core/event/`
2. Add interface check in `storage/`: `var _ event.TransactionalStore = (*TransactionalEventStore)(nil)`
3. No changes to aggregate/decider repositories yet

### Phase 2: Repository Integration (Non-breaking)

1. Add `save()` helper to aggregate/decider repositories
2. Use type assertion to detect `TransactionalStore`
3. Fallback to existing two-step approach
4. All existing tests continue to pass

### Phase 3: SQL Implementation

1. Create `storage/transactional_store.go`
2. Accept shared `*sql.DB` in constructor
3. Extract `saveEvents(tx)` and `appendOutbox(tx)` helpers from existing code
4. Add integration tests with real PostgreSQL

### Phase 4: Documentation

1. Update FEATURES.md with `TransactionalStore` row
2. Add example in `example/user/`
3. Update CHANGELOG

## Open Questions

- Should `SaveWithOutbox` skip the bus publish entirely (since the outbox guarantees eventual delivery)?
- How does `AppendBatch` fit? Should it also have a `AppendBatchWithOutbox`?
- Should we add `SnapshotSaveWithOutbox` for snapshot + outbox atomicity?
- What about consumers using non-SQL stores (e.g., DynamoDB)? They'd implement `TransactionalStore` with their own atomic primitives.

## API Surface

```
core/event/interfaces.go     — TransactionalStore interface (6 new lines)
storage/transactional_store.go — SQL implementation (~80 lines)
aggregate/repository.go      — save() helper with type assertion (~10 lines)
decider/decider.go           — save() helper with type assertion (~10 lines)
```

**Estimated effort:** 4 hours (interface + SQL impl + tests).
