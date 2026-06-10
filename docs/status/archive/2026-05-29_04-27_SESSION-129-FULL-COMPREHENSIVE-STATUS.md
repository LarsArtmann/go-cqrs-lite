# Session 128 — Full Comprehensive Status Update

**Date:** 2026-05-29 04:27 | **Branch:** master | **Commit:** 7ddafb7
**Tests:** 29/29 packages pass | **Working tree:** clean

---

## a) FULLY DONE ✅

### OpenTelemetry Instrumentation Core

| Module                     | Spans                                                                               | RecordError  | Tests         | Status      |
| -------------------------- | ----------------------------------------------------------------------------------- | ------------ | ------------- | ----------- |
| `otel/` shared module      | N/A (infrastructure)                                                                | N/A          | 15 test cases | ✅ Complete |
| `middleware/` tracing      | 4 spans (`command.handle`, `event.handle`, `query.handle`, `event.publish`)         | ✅ All paths | 17 files      | ✅ Complete |
| `middleware/` OTel metrics | `CommandOTelMetrics`, `EventOTelMetrics`, `QueryOTelMetrics`, `OTelMetricsRecorder` | N/A          | ✅            | ✅ Complete |

### Storage Module — 17 methods fully instrumented

| Method                          | Span Name                       | RecordError          |
| ------------------------------- | ------------------------------- | -------------------- |
| `SQLEventStore.Save`            | `event.store.save`              | ✅ All 4 error paths |
| `SQLEventStore.AppendBatch`     | `event.store.append_batch`      | ✅ All 3 error paths |
| `SQLEventStore.Load`            | `event.store.load`              | ✅                   |
| `SQLEventStore.LoadFromVersion` | `event.store.load_from_version` | ✅                   |
| `SQLEventStore.LoadToVersion`   | `event.store.load_to_version`   | ✅                   |
| `SQLEventStore.LoadToTimestamp` | `event.store.load_to_timestamp` | ✅                   |
| `SQLEventStore.LoadBackwards`   | `event.store.load_backwards`    | ✅                   |
| `SQLEventStore.ReadAll`         | `event.store.read_all`          | ✅                   |
| `SQLEventStore.ReadFrom`        | `event.store.read_from`         | ✅                   |
| `SQLOutbox.Append`              | `outbox.append`                 | ✅                   |
| `SQLOutbox.PollPending`         | `outbox.poll`                   | ✅                   |
| `SQLOutbox.Ack`                 | `outbox.ack`                    | ✅                   |
| `SQLCheckpointStore.Load`       | `checkpoint.load`               | ✅                   |
| `SQLCheckpointStore.Save`       | `checkpoint.save`               | ✅                   |
| `SQLSnapshotStore.Save`         | `snapshot.save`                 | ✅                   |
| `SQLSnapshotStore.Load`         | `snapshot.load`                 | ✅                   |

### Decider Module — 2 methods instrumented

| Method               | Span Name         | RecordError            |
| -------------------- | ----------------- | ---------------------- |
| `Repository.Execute` | `decider.execute` | ✅ All 5 error paths   |
| `Repository.Load`    | `decider.load`    | ⚠️ Missing (see below) |

### Saga Module — 3 methods instrumented

| Method               | Span Name           | RecordError            |
| -------------------- | ------------------- | ---------------------- |
| `Runner.Start`       | `saga.start`        | ✅ All error paths     |
| `Runner.ExecuteStep` | `saga.step.execute` | ⚠️ Partial (see below) |
| `Runner.compensate`  | `saga.compensate`   | ✅ Dispatch errors     |

### Projection Module — 2 internal spans

| Method                          | Span Name           | RecordError                 |
| ------------------------------- | ------------------- | --------------------------- |
| `replay` (private)              | `projection.replay` | ⚠️ Checkpoint error missing |
| `handleAndCheckpoint` (private) | `projection.handle` | ✅                          |

### Non-OTel Work Fully Done

| Area                                | Status                                                              |
| ----------------------------------- | ------------------------------------------------------------------- |
| Error taxonomy migration (5-family) | ✅ All modules use `go-error-family`                                |
| BDD test suites                     | ✅ command, event, query, saga, signing, memory, middleware, stream |
| Multi-module workspace              | ✅ 15 modules with `go.work`                                        |
| `integration/go.mod`                | ✅ Has otel replace directive                                       |
| `go.sum` propagation                | ✅ All 12 modules tidy                                              |
| Code dedup (middleware)             | ✅ Local helpers removed, using `cqrsotel`                          |

---

## b) PARTIALLY DONE 🔶

### RecordError Gaps on Existing Spans

| Module           | Method            | Issue                                                                                                                           |
| ---------------- | ----------------- | ------------------------------------------------------------------------------------------------------------------------------- |
| **core/decider** | `Repository.Load` | Returns `loadState` error without `cqrsotel.RecordError` on the span                                                            |
| **saga**         | `ExecuteStep`     | 5+ error paths missing RecordError: `hydrate` error, not-running rejection, `store.Save` errors (×3), `dispatchWithRetry` error |
| **saga**         | `compensate`      | `store.Save` error at end not recorded                                                                                          |
| **projection**   | `replay`          | `checkpoint.Load` error calls `span.End()` but skips `RecordError`                                                              |

### Missing Spans on Public I/O Methods

| Module           | Method                                  | Priority                           |
| ---------------- | --------------------------------------- | ---------------------------------- |
| **storage**      | `SQLSnapshotStore.LoadAtVersion`        | High — public time-travel API      |
| **storage**      | `SQLSagaStore.Save/Load/LoadAllRunning` | High — saga state persistence      |
| **storage**      | `SQLTransactionalStore.SaveWithOutbox`  | High — critical write path         |
| **storage**      | `SQLEventStore.LoadStream`              | Medium — stream reader             |
| **core/decider** | `LoadAtVersion`                         | Medium — time-travel query         |
| **core/decider** | `LoadAtTime`                            | Medium — temporal query            |
| **projection**   | `Run`                                   | Medium — orchestration entry point |

---

## c) NOT STARTED ⬜

| #   | Area                                    | Methods                                                      | Effort |
| --- | --------------------------------------- | ------------------------------------------------------------ | ------ |
| 1   | `stream/` module                        | `List`, `ListWithStatus`, `AggregateProjection.Handle`       | 1 hr   |
| 2   | `signing/` module                       | `Sign`, `Verify`, `VerifyActor` (HMAC, Ed25519, Multi)       | 2 hr   |
| 3   | `memory/` module                        | 20 methods across MemoryStore/Bus/Outbox/Snapshot/Checkpoint | 3 hr   |
| 4   | `watermill/` module                     | `Publish`, `Subscribe`, `Close`                              | 1 hr   |
| 5   | `storage/PebbleEventStore`              | All methods (Save, Load, variants)                           | 2 hr   |
| 6   | OTel metrics for storage                | Latency histograms for store/query operations                | 3 hr   |
| 7   | OTel semconv adoption                   | `db.operation`, `messaging.system` standard attributes       | 4 hr   |
| 8   | Integration test with trace propagation | command → decider → store → bus → projection                 | 3 hr   |
| 9   | W3C trace context propagation           | Through event bus for distributed tracing                    | 4 hr   |
| 10  | Example code for OTel provider wiring   | Consumer-facing example                                      | 1 hr   |
| 11  | `testhelpers/otel.go` shared test utils | Extract `testTracerWithRecorder()`                           | 30 min |

---

## d) TOTALLY FUCKED UP 💥

| Issue                    | Severity | Details                                                                                                                                   |
| ------------------------ | -------- | ----------------------------------------------------------------------------------------------------------------------------------------- |
| Nothing critical broken  | —        | All 29 test packages pass. Clean working tree. Zero compilation errors.                                                                   |
| `gopls` diagnostics spam | Low      | ~560 "go mod tidy" errors from gopls not understanding go.work. Known issue, documented in ADR-0007. Tests pass fine with workspace mode. |
| Pre-commit hook exit-1   | Low      | golangci-lint exit-7 from go.work incompatibility + go-structure-linter medium-severity suggestions. Not real issues.                     |

---

## e) WHAT WE SHOULD IMPROVE

### Architecture

1. **RecordError discipline**: Every span MUST record errors. Currently 4 methods have spans but silently swallow errors. This defeats the purpose of observability — a failed span appears green in dashboards.

2. **Span coverage completeness**: 28 spans exist but ~47 public I/O methods remain uninstrumented. The storage module is ~60% covered. Critical gaps: `SaveWithOutbox` (the atomic write path), `LoadAtVersion` (time-travel), saga store.

3. **Semconv alignment**: Custom `cqrs.*` attributes work but don't integrate with OTel ecosystem tooling (Grafana dashboards, Datadog APM, Jaeger). Should layer `db.operation`/`messaging.system` underneath.

4. **Test helper consolidation**: `testTracerWithRecorder()` duplicated in `otel/otel_test.go` and `middleware/tracing_test.go`. Should live in `testhelpers/`.

5. **Option pattern for span creation**: Every instrumented method repeats the same 6-line pattern (`StartSpan`, `defer span.End()`, `RecordError`). Could use `cqrsotel.Span(ctx, "name", opts...)` with deferred error recording.

### Type Model

6. **SpanKind type safety**: Passing raw `trace.SpanKindClient` everywhere. Could add semantic constants: `SpanKindStore`, `SpanKindProjector`, etc.

7. **Attribute builder pattern**: Instead of `aggregateAttrs()` + `aggregateAttrsWithVersion()` in every module, use a builder: `cqrsotel.Attrs().Aggregate(type, id).Version(v).EventCount(n)`.

---

## f) TOP 25 THINGS TO DO NEXT

### Tier 1: Fix bugs (high impact, minutes each)

| #   | Item                                                          | Effort |
| --- | ------------------------------------------------------------- | ------ |
| 1   | Add `RecordError` to `decider.Load` return path               | 5 min  |
| 2   | Add `RecordError` to all 5+ error paths in `saga.ExecuteStep` | 15 min |
| 3   | Add `RecordError` to `saga.compensate` store.Save error       | 5 min  |
| 4   | Add `RecordError` to `projection.replay` checkpoint error     | 5 min  |
| 5   | Extract `testTracerWithRecorder()` to `testhelpers/otel.go`   | 30 min |

### Tier 2: Complete coverage (high impact, 1-2 hr each)

| #   | Item                                                           | Effort |
| --- | -------------------------------------------------------------- | ------ |
| 6   | Add span to `storage/SQLSnapshotStore.LoadAtVersion`           | 15 min |
| 7   | Add span to `storage/SQLTransactionalStore.SaveWithOutbox`     | 30 min |
| 8   | Add spans to `storage/SQLSagaStore` (Save/Load/LoadAllRunning) | 30 min |
| 9   | Add span to `storage/SQLEventStore.LoadStream`                 | 15 min |
| 10  | Add spans to `core/decider/LoadAtVersion` and `LoadAtTime`     | 30 min |
| 11  | Add span to `projection/Run`                                   | 30 min |
| 12  | Add spans to `stream/` module (List, ListWithStatus)           | 1 hr   |

### Tier 3: New instrumentation (medium impact, 1-2 hr each)

| #   | Item                                               | Effort |
| --- | -------------------------------------------------- | ------ |
| 13  | Add spans to `signing/` module                     | 2 hr   |
| 14  | Add spans to `storage/PebbleEventStore`            | 2 hr   |
| 15  | Add spans to `memory/` module (test observability) | 3 hr   |
| 16  | Add spans to `watermill/` module                   | 1 hr   |

### Tier 4: Ecosystem integration (high impact, higher effort)

| #   | Item                                                            | Effort |
| --- | --------------------------------------------------------------- | ------ |
| 17  | Adopt OTel semconv `db.operation`/`messaging.system` attributes | 4 hr   |
| 18  | OTel metrics for storage operations (latency histograms)        | 3 hr   |
| 19  | Integration test with full trace propagation                    | 3 hr   |
| 20  | W3C trace context propagation through event bus                 | 4 hr   |

### Tier 5: Polish and documentation

| #   | Item                                                        | Effort |
| --- | ----------------------------------------------------------- | ------ |
| 21  | Option pattern API for span creation (`cqrsotel.Span(...)`) | 3 hr   |
| 22  | Example code for OTel provider wiring                       | 1 hr   |
| 23  | Performance benchmark: span overhead (noop vs real)         | 2 hr   |
| 24  | ADR for OTel instrumentation decisions                      | 1 hr   |
| 25  | CI addition: OTel integration test in GitHub Actions        | 2 hr   |

---

## g) TOP #1 QUESTION

**Should `memory/` (in-memory test doubles) get OTel spans?**

- **Pro**: Integration tests become observable — you can see the full trace from command handler through in-memory store to bus. Makes debugging test failures much easier.
- **Con**: MemoryStore/MemoryBus are test utilities, not production code. Adding spans increases their complexity and changes their performance profile. Production consumers never use them.
- **Alternative**: Only instrument the `integration/` module's cross-module tests, not the memory implementations themselves.

**My recommendation**: Skip memory/ spans. Instead, instrument the `integration/` test suite to validate trace propagation. This gives the observability benefit without polluting test utilities.

---

## Module Scorecard

| Module          | otel.go                 | Spans  | Missing   | RecordError Gaps | Test Files |
| --------------- | ----------------------- | ------ | --------- | ---------------- | ---------- |
| `otel/`         | — (IS module)           | —      | 0         | 0                | 1          |
| `middleware/`   | No (uses passed tracer) | 4      | 0         | 0                | 17         |
| `storage/`      | ✅                      | 17     | 5 methods | 0                | 29         |
| `core/decider/` | ✅                      | 2      | 2 methods | 1                | 9          |
| `projection/`   | ✅                      | 2      | 1 method  | 1                | 12         |
| `saga/`         | ✅                      | 3      | 0         | 6+ paths         | 10         |
| `stream/`       | ❌                      | 0      | 3         | —                | 9          |
| `signing/`      | ❌                      | 0      | 7         | —                | 18         |
| `memory/`       | ❌                      | 0      | 20        | —                | 11         |
| `watermill/`    | ❌                      | 0      | 2         | —                | 3          |
| **Total**       | **4/10**                | **28** | **~40**   | **~8**           | **119**    |

---

## Test Results

```
29 packages tested — 29 PASS, 0 FAIL
Total duration: ~2.5s
```
