# eventtest — Test Doubles and Helpers for event/v4

[![Go Reference](https://pkg.go.dev/badge/github.com/larsartmann/go-cqrs-lite/event/v4/eventtest.svg)](https://pkg.go.dev/github.com/larsartmann/go-cqrs-lite/event/v4/eventtest)

Fakes, factories, assertions, a golden-file helper, and a reusable store conformance suite for testing code that depends on `event/v4`.

```bash
go get github.com/larsartmann/go-cqrs-lite/event/v4/eventtest
```

## Why?

Testing CQRS code requires realistic event fixtures, controllable fakes, and repeatable assertions. This package provides all three without panicking: every helper uses `tb.Fatalf` on error so test failures produce clean diagnostics instead of stack traces.

## Fakes

### FakeStore

Implements `event.Store` + `event.Journal` + `event.SeekableJournal` fully in memory. Each method has an optional override for testing error paths:

```go
store := eventtest.NewFakeStore()
store.SaveFn(func(ctx context.Context, ref id.StreamRef, events ...event.Event) error {
    return errors.New("simulated save failure")
})
```

| Method            | Description                                              |
| ----------------- | -------------------------------------------------------- |
| `NewFakeStore()`  | Constructor.                                             |
| `SaveFn(fn)`      | Override Save. Returns `*FakeStore` for chaining.        |
| `LoadFn(fn)`      | Override Load.                                           |
| `LoadFromVersionFn(fn)` | Override LoadFromVersion.                          |
| `LoadToVersionFn(fn)`   | Override LoadToVersion.                            |
| `LoadToTimestampFn(fn)` | Override LoadToTimestamp.                          |
| `CloseFn(fn)`     | Override Close.                                          |
| `AppendBatchFn(fn)` | Override AppendBatch.                                  |

### FakeBus

Implements `event.Bus` synchronously in memory. Captures published events and supports handler + publish middleware chains.

```go
bus := eventtest.NewFakeBus()
bus.PublishErr = errors.New("publish failed") // inject a publish error
bus.Published // []event.Event — inspect what was published
```

### FakeSnapshotStore

Implements `snapshot.SnapshotSink` + `SnapshotSource` + `SnapshotStore`. Supports error injection via `SetLoadError` / `SetSaveError`.

## Event Factories

| Function                              | Description                                          |
| ------------------------------------- | ---------------------------------------------------- |
| `NewEvent(t, typ, aggID, aggType, ver, payload)` | Creates an event, fatals on error.         |
| `MakeEvent(...)`                      | Same as `NewEvent` but returns `(event, error)`.     |
| `QuickEvent(...)`                     | Ignores error (panic-free fast path).                |
| `MakeTimelineEvents(tb, aggType, aggID, events)` | Creates events at relative time offsets.   |
| `TamperEvent(original, newPayload)`   | Recreates with same metadata but different payload.  |
| `QuickSnapshot(aggID, aggType, ver, state)` | Creates a snapshot for testing.                |

## Assertions

| Function                           | Description                                    |
| ---------------------------------- | ---------------------------------------------- |
| `AssertEventType(t, events, i, want)` | Assert event type at index.                 |
| `AssertEventVersion(t, events, i, want)` | Assert event version at index.           |
| `AssertCallOrder(t, callOrder, expected)` | Assert middleware call order.             |
| `AssertGolden(t, path, got, update)`   | Golden-file comparison with update mode.    |
| `AssertContains` / `AssertNotContains` | String membership checks.                   |
| `AssertErrorContains(t, err, substr)`  | Error message substring check.             |

## Handler Helpers

| Function                    | Description                                             |
| --------------------------- | ------------------------------------------------------- |
| `AppendEventsHandler(&evts)`| Returns a handler that appends events to a slice.       |
| `NoopEventHandler()`        | No-op handler.                                          |
| `FailingEventHandler(msg)`  | Returns a handler that always errors.                   |
| `PanicEventHandler(msg)`    | Returns a handler that panics (for recovery testing).   |
| `EventMiddleware(&order, name)` | Middleware factory that records call order.         |

## Store Conformance Suite

Reusable test functions for verifying any `event.Store` implementation:

```go
func TestMyStore(t *testing.T) {
    store := NewMyStore()
    cfg := eventtest.NewStoreTestConfig("Order", "order.created", "total", "100")
    eventtest.TestStoreSaveAndLoad(t, store, cfg, aggID)
    eventtest.TestStoreConcurrencyConflict(t, store, cfg, aggID)
    eventtest.TestStoreAppendBatch(t, store, cfg, aggID)
}
```

| Function                          | Description                                            |
| --------------------------------- | ------------------------------------------------------ |
| `StoreTestConfig`                 | Configurable test event factory (type, payload, etc.). |
| `NewStoreTestConfig(...)`         | Factory with JSON payload template.                     |
| `IssueStoreConfig()` / `OrderStoreConfig()` | Prebuilt configs.                             |
| `TestStoreSaveAndLoad(...)`       | Round-trip save and load.                              |
| `TestStoreConcurrencyConflict(...)` | Optimistic concurrency version conflict.             |
| `TestStoreAppendBatch(...)`       | Batch append semantics.                                |
| `TestStoreLoadFromVersion(...)`   | Partial load from version.                             |
| `TestStoreMetadataRoundtrip(...)` | Metadata preservation across save/load.               |

## Design

- **No panics**: All helpers use `tb.Fatalf` on error for clean test output.
- **Override-injection pattern**: `FakeStore` methods check for optional overrides before falling back to default in-memory behavior.
- **Compile-time assertions**: Every fake has `var _ event.Interface = (*Fake)(nil)` to catch interface drift.
- **Prebuilt configs**: `IssueStoreConfig()` and `OrderStoreConfig()` provide ready-made test fixtures.

## Related Modules

- [**event**](../../README.md) — The package under test
- [**id/idtest**](../../../id/README.md) — Branded-ID test helpers used by event factories
