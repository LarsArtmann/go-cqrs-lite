# Session 32 Status: Test Coverage, Type Quality & Stale Issue Cleanup

> **Date:** 2026-05-02
> **Commits:** 3 (cc9d6c3, c9c4c6d, 7bf7525)
> **Tests:** All 20 packages pass

---

## Summary

Continuation session focused on closing gaps from Session 31's execution plan. Fixed bugs, added missing tests, improved type quality, and cleaned up stale documentation.

## Commits

### 1. `cc9d6c3` — Test coverage for taxonomy, ClientID, offline metadata, projection retry

- `Classify(nil)` now returns `Rejection` (was `Transient`) — `IsRetryable(nil)` returns `false`
- `ErrDuplicateProjection` classified as `Conflict` in `Classify()`
- `TestWithClientID` and `TestWithClientOccurredAt` in `core/event`
- `TestClientID` branded ID roundtrip in `core/pkg/id`
- `TestRunner_RetryOnTransientError` — verifies exponential backoff retry
- `TestRunner_NoRetryOnNonRetryableError` — verifies conflict errors not retried

### 2. `c9c4c6d` — Type quality improvements

- `event.Error` now implements `fmt.Formatter` — `%+v` shows `family:code: message` with cause chain
- `event.Version` now has `String()` method — consistent with project conventions
- `MustParseClientID` added to panic test table
- `RetryConfig.IsRetryable` field documented

### 3. `7bf7525` — Remove stale Known Issue

- Removed "FakeStore/MemoryStore key separator mismatch" — both now use `":"`, was already fixed

## Remaining Known Issues

| Issue                                                                | Severity | Notes                                                                                                                                      |
| -------------------------------------------------------------------- | -------- | ------------------------------------------------------------------------------------------------------------------------------------------ |
| Cross-package sentinels not in `Classify()`                          | MEDIUM   | Circular dependency prevents mapping aggregate/projection/storage errors. Documented. Consumers use `errors.Is` or wrap with typed errors. |
| `WithBatchSize`/`WithBatchWindow`/`WithConcurrency` unused in runner | MEDIUM   | Options exist but runner never reads the fields. Dead API surface.                                                                         |
| `MemoryBus.Publish` holds RLock during handler execution             | LOW      | Acceptable for test utility                                                                                                                |
| `query.Handler` returns `any`                                        | LOW      | `DispatchTyped[T]` workaround exists                                                                                                       |
| `CatalogMeta` duplicated across 3 packages                           | LOW      |                                                                                                                                            |
| `Root.LoadEvents` vs `Core.LoadFromHistory` mismatch                 | LOW      |                                                                                                                                            |

## Test Coverage

All 20 test packages pass. Key coverage improvements:

- `core/event/errors_taxonomy_test.go` — Full coverage of `Error`, `Family`, `Classify`, `IsRetryable`, `Format`
- `core/event/event_test.go` — `WithClientID`, `WithClientOccurredAt`
- `core/pkg/id/id_test.go` — `ClientID` roundtrip, `MustParseClientID` panic
- `projection/runner_test.go` — Retry transient, no-retry non-retryable, `newTestRunnerWithOpts` helper
- `core/event/types_test.go` — `Version.String()`
