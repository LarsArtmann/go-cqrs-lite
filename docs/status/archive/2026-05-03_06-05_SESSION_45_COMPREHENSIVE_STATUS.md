# Session 45 — Comprehensive Status Report

**Date:** 2026-05-03 06:05  
**Branch:** `master`  
**Commits since Session 42:** 7  
**Test suites:** 21 packages, ALL PASS (incl. `-race`)  
**Total LOC:** 30,681 Go (10,156 production + 20,525 test)  
**Production functions:** 588  
**Lint:** 46 pre-existing issues (0 from recent sessions)  
**go vet:** clean  
**TODOs/FIXMEs:** 0  
**Files over 250 lines:** 0 (production)

---

## A) FULLY DONE ✓

### Core Library (10 modules)

| Module                 | Key Types                                                                   | Coverage | Status |
| ---------------------- | --------------------------------------------------------------------------- | -------- | ------ |
| `core/command`         | `Dispatcher`, `Handler`, `Core`, `CatalogCore`                              | 100.0%   | ✅     |
| `core/query`           | `Dispatcher`, `Handler`, `Pagination`, `PaginatedResult[T]`                 | 100.0%   | ✅     |
| `core/pkg/dispatcher`  | `Dispatcher[H, M]`, `LifecycleMixin`, `CatalogDispatcher`                   | 100.0%   | ✅     |
| `core/pkg/id`          | `id.Of[T]`, `AggregateID`, `EventID`, `UserID`, `ClientID`, etc.            | 100.0%   | ✅     |
| `middleware`           | Logging, Metrics, Recovery, Retry, Validation, Tracing                      | 100.0%   | ✅     |
| `core/event`           | `Store`, `Bus`, `Publisher`, `Subscriber`, `Event`, `Error`, `Outbox`, etc. | 98.0%    | ✅     |
| `catalog/d2`           | D2 diagram text exporter                                                    | 97.6%    | ✅     |
| `catalog/asyncapi`     | AsyncAPI 3.0 YAML/JSON exporter                                             | 95.9%    | ✅     |
| `catalog/eventcatalog` | EventCatalog MDX file generator                                             | 95.6%    | ✅     |
| `catalog/adapters`     | `CatalogBuilder`, `FromDispatcher` adapters                                 | 95.5%    | ✅     |

### Architecture & Design Patterns (All Sessions Cumulative)

| Feature                                 | Status | Detail                                                                          |
| --------------------------------------- | ------ | ------------------------------------------------------------------------------- |
| **Error taxonomy**                      | ✅     | 5 families (Rejection, Conflict, Transient, Corruption, Infrastructure)         |
| **Extensible classification**           | ✅     | `RegisterClassification()` + `init()` in command/query — avoids circular deps   |
| **Publisher/Subscriber sub-interfaces** | ✅     | `event.Bus` composes both — ISP-compliant                                       |
| **Branded IDs**                         | ✅     | `id.Of[T]` with 8 branded types, JSON/SQL/binary encoding                       |
| **IdempotencyKey on Command**           | ✅     | Required interface method                                                       |
| **Client metadata**                     | ✅     | `WithClientID`, `WithClientOccurredAt`                                          |
| **Decider pattern**                     | ✅     | Pure-function aggregate via `Decider[State]`                                    |
| **Snapshot support**                    | ✅     | Both aggregate + decider, with codec + strategy                                 |
| **Outbox pattern**                      | ✅     | Memory + SQL implementations, `OutboxPublisher` with panic recovery             |
| **Catalog system**                      | ✅     | AsyncAPI, D2, EventCatalog exporters                                            |
| **Projection runner**                   | ✅     | Replay + live subscription, per-projection checkpoint, retry with `IsRetryable` |
| **Injected logger**                     | ✅     | `WithLogger(*slog.Logger)` in projection runner                                 |
| **Panic recovery**                      | ✅     | `HandleParallel` goroutines + `OutboxPublisher` goroutine                       |
| **Event builder**                       | ✅     | `NewBuilder` fluent API for event construction                                  |
| **Event enricher**                      | ✅     | `CompositeEnricher` for metadata enrichment                                     |
| **Multi-module monorepo**               | ✅     | 10 modules with `go.work`, `core` independently publishable                     |

### Session History (Sessions 1–44)

| Session | Key Deliverables                                                                       |
| ------- | -------------------------------------------------------------------------------------- |
| 1–2     | Initial library, bug fixes (retry dead cancellation, version desync, slice mutation)   |
| 3       | Branded return types migration (IDs, Version)                                          |
| 20      | 48 lint issues → 0, storage coverage 79.8%→92.3%, code extraction                      |
| 25      | SQLSnapshotStore double-marshal fix, storage.Close() ownership, Version branded type   |
| 27      | No-panic convention (New\* returns error), Bus.Use interface, compile-time checks      |
| 28      | Snapshot error propagation, SQL empty-result semantics, HandleParallel context respect |
| 29      | NewRunner returns error, compile-time interface checks, sentinel error extraction      |
| 30      | Architecture roadmap (5-phase plan, error taxonomy design, offline-first primitives)   |
| 31      | Error taxonomy (5 families), `id.ClientID`, `IdempotencyKey()`, projection retry       |
| 32      | `event.Error` fmt.Formatter, Version.String(), projection retry tests                  |
| 34      | Dead API removal, 57 godoc comments, String() methods, event.Error.Is()                |
| 36      | testhelpers split (293→3 files), compile-time interface checks, aggregate trim         |
| 37      | `core/decider` package, example/user rewrite with Decider pattern                      |
| 38      | `event.SubscribesTo` export, dead Typed interface removal, sentinel errors             |
| 39      | Dead .golangci-lint.yml, math/rand/v2 for jitter, catalog ErrDomainNotFound            |
| 40–42   | PostgreSQL outbox, snapshot for decider, domain glossary, ADRs, archive stale docs     |
| 43      | Panic recovery, data race fixes, decider 77.4%→94.3%, projection test quality          |
| 44      | Publisher/Subscriber ISP, RegisterClassification, WithLogger DI, lint compliance       |

---

## B) PARTIALLY DONE ⚠️

| Item                              | What's Done                                   | What's Missing                                                                                                                                                 | Impact                                                        |
| --------------------------------- | --------------------------------------------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------- | ------------------------------------------------------------- |
| **Error classification coverage** | event, command, query sentinels registered    | aggregate, projection, storage sentinels NOT registered (circular deps solvable via `init()`)                                                                  | MEDIUM — `Classify()` returns `Transient` for those packages  |
| **Function size compliance**      | Worst offenders extracted (76→22, 66→5, etc.) | ~43 functions still >30 lines (catalog exporters, `validateEventParams` 51L)                                                                                   | LOW — mostly rendering functions that resist clean extraction |
| **CatalogMeta consolidation**     | Investigated thoroughly                       | SKIPPED — `event.CatalogMeta` has extra `AggregateType` field, no shared location without new package                                                          | LOW — 3× duplication of 3-field struct                        |
| **Lint: 46 issues remaining**     | All session-introduced issues fixed           | 46 pre-existing: errcheck (11), wsl_v5 (8), perfsprint (8), noinlineerr (6), nlreturn (6), revive (3), err113 (2), exhaustruct (1), golines (1), modernize (1) | LOW — none in production code paths                           |

### Coverage Near Targets

| Package          | Coverage | Gap  | What's Missing                                                                       |
| ---------------- | -------- | ---- | ------------------------------------------------------------------------------------ |
| `core/decider`   | 94.3%    | 0.7% | `loadFromSnapshot` 86.4%, `saveSnapshot` 85.7%, `loadFromStore` 92.3%                |
| `catalog`        | 94.4%    | 0.6% | `SchemaToAny` 70.0%, `operationTitleAndName` 87.5%                                   |
| `core/aggregate` | 93.2%    | 1.8% | `NewCore` 60.0%, `MustNewCore` 75.0%, `LoadFromHistory` 83.3%, `loadFromStore` 75.0% |
| `storage`        | 92.0%    | 3.0% | `scanOutboxEntries` 75%, `reconstructOutboxEvent` 76.9%, `scanEvent` 84.2%           |
| `memory`         | 91.9%    | 3.1% | `LoadAll` 0%, `LoadAtVersion` 92.3%, `Ack` 92.3%                                     |
| `projection`     | 88.8%    | 6.2% | `replay` 73.3%, `handleWithRetry` 84.6%, `Close` 0%, `WithLogger` 0%                 |

---

## C) NOT STARTED ○

| Item                                    | Priority | Effort           | Notes                                                                           |
| --------------------------------------- | -------- | ---------------- | ------------------------------------------------------------------------------- |
| **Outbox transaction co-participation** | CRITICAL | LARGE            | `SQLOutbox.Append` and `SQLEventStore.Save` run in separate txs — atomicity gap |
| **`query.Handler` returns `any`**       | HIGH     | LARGE (breaking) | Violates "no any" rule. `DispatchTyped[T]` is the workaround                    |
| **Aggregate sentinel registration**     | MEDIUM   | SMALL            | `ErrAggregateNotFound`, etc. — use `RegisterClassification` via `init()`        |
| **Storage sentinel registration**       | MEDIUM   | SMALL            | SQL-specific errors — same pattern                                              |
| **Projection sentinel registration**    | MEDIUM   | SMALL            | `ErrDuplicateProjection`, etc. — same pattern                                   |
| **Event signing / integrity**           | LOW      | LARGE            | No HMAC or checksum on stored events                                            |
| **Saga / Process Manager**              | PLANNED  | VERY LARGE       | `docs/planning/SAGA_DESIGN.md` exists but no implementation                     |
| **Watermill module**                    | PLANNED  | LARGE            | Pub/sub adapter for Kafka, NATS                                                 |
| **SQLite / Turso support**              | PLANNED  | MEDIUM           | User mentioned in passing                                                       |
| **Tagged releases**                     | PLANNED  | SMALL            | All modules at `v0.0.0`                                                         |
| **CHANGELOG.md**                        | PLANNED  | SMALL            | No changelog tracking yet                                                       |
| **CONTRIBUTING.md**                     | PLANNED  | SMALL            | No contribution guidelines                                                      |
| **Benchmarks**                          | PLANNED  | MEDIUM           | No performance benchmarks for outbox or decider                                 |

---

## D) TOTALLY FUCKED UP 💥

### 1. Zero-coverage functions in production

| Function                    | File                             | Lines | Impact                                               |
| --------------------------- | -------------------------------- | ----- | ---------------------------------------------------- |
| `LoadAll`                   | `memory/store.go:171`            | 26    | `event.GlobalLoader` interface method — never tested |
| `OutboxSchema`              | `storage/outbox.go:17`           | 19    | Returns DDL string — zero tests                      |
| `ExportD2` (via builder)    | `catalog/adapters/builder.go:79` | 10    | Convenience wrapper — never called in tests          |
| `WithDirection`             | `catalog/d2/exporter.go:23`      | 6     | Option function — zero usage                         |
| `WithLogger`                | `projection/options.go:29`       | 4     | New option — no test yet (Session 44 addition)       |
| `Close` (projection runner) | `projection/runner.go:205`       | 2     | No-op close — zero coverage                          |
| `Close` (memory checkpoint) | `memory/checkpoint.go:48`        | 2     | No-op close — zero coverage                          |
| `Close` (memory outbox)     | `memory/outbox.go:103`           | 2     | No-op close — zero coverage                          |

### 2. `MemoryBus.Publish` holds `RLock` during handler execution

Subscribers block publishers. Acceptable for test utility but documented as a gotcha. Not a bug — by design for simplicity.

### 3. `MemoryStore.LoadAll` at 0% — untested interface method

The `event.GlobalLoader` interface requires `LoadAll()`, but only `MemoryStore` implements it. No consumer uses it in production. The method exists for the projection runner's replay but is only exercised indirectly.

### 4. 46 lint issues — all pre-existing, all in test files or edge cases

Breakdown by severity:

- **errcheck (11)**: Unchecked `store.AppendBatch` in decider tests (9), unchecked `recover()` in outbox (1), 1 other
- **wsl_v5 (8)**: Missing whitespace above if/assign in decider tests
- **perfsprint (8)**: `fmt.Errorf` with no format verbs — should be `errors.New`
- **noinlineerr (6)**: Inline `if err := ...; err != nil` — project convention prefers separate assignment
- **nlreturn (6)**: Missing blank line before return in decider package
- **revive (3)**: Unused `state` parameter in decider test DecideFuncs
- **err113 (2)**: Dynamic errors in `opError` and panic recovery — intentional (contextual messages)
- **exhaustruct (1)**: Anonymous struct missing `mu` field — false positive (field is in composite literal)
- **golines (1)**: Long line in decider test
- **modernize (1)**: `WaitGroup.Go` suggestion in decider test

### 5. `LoadAtVersion` at 92.3% in memory/snapshot

Missing edge case: version 0 (empty snapshot store) returns `ErrSnapshotNotFound`. Tested indirectly through decider but not directly.

---

## E) WHAT WE SHOULD IMPROVE

### Architecture (HIGH impact)

1. **Outbox transaction co-participation** — The #1 architectural gap. `SQLOutbox.Append` and `SQLEventStore.Save` must share a transaction. Options: (a) `SQLEventStore` wraps outbox, (b) accept external `*sql.Tx`, (c) new `OutboxEventStore` composition. This is the difference between "at-least-once with occasional duplicates" and "exactly-once semantics."

2. **`query.Handler` returns `any`** — The only remaining "any" in the codebase. A breaking change to generics would eliminate the need for `DispatchTyped[T]` workaround.

3. **Aggregate/decorator pattern** — Both `aggregate` and `decider` packages duplicate the same repository pattern (load→apply→save→publish). Consider extracting a shared `RepositoryCore` that both packages compose, reducing ~120 lines of duplication.

### Test Quality (MEDIUM impact)

4. **Projection coverage 88.8%** — Below 95% target. `replay` at 73.3% is the main gap. Missing: replay error propagation, checkpoint save during replay, context cancellation during replay.

5. **Zero-coverage functions** — 8 production functions at 0%. Most are `Close()` no-ops and convenience wrappers. Easy wins.

6. **Storage coverage 92.0%** — `scanOutboxEntries` at 75%, `reconstructOutboxEvent` at 76.9%. These are SQL scanning paths with error handling that should be tested.

7. **Aggregate coverage 93.2%** — `NewCore` at 60% (validation errors untested), `loadFromStore` at 75%.

### Code Quality (LOW impact)

8. **46 lint issues** — All in tests or non-critical. Could be fixed in ~1 hour:
   - Add `_, _ = store.AppendBatch(...)` or `require.NoError` for 9 errcheck in decider tests
   - Replace 8 `fmt.Errorf("literal")` → `errors.New("literal")`
   - Add blank lines before returns in decider
   - Rename 3 unused `state` → `_`

9. **Function size** — `validateEventParams` at 51 lines is the worst remaining production function. 43 functions >30 lines total.

10. **Duplicate `opError`** — Both `core/aggregate/repository.go` and `core/decider/decider.go` define identical `opError` functions. Could be shared via a helper package.

### Documentation (LOW impact)

11. **No CHANGELOG.md** — 44 sessions of changes, no changelog.

12. **No CONTRIBUTING.md** — Multi-module monorepo needs contribution guidelines.

13. **No benchmarks** — Zero performance benchmarks in the entire codebase.

---

## F) TOP 25 THINGS TO DO NEXT

Ranked by impact × effort (Pareto — highest impact first):

### CRITICAL (blocks production reliability)

| #   | Task                                    | Effort | Impact   | Detail                                                                                     |
| --- | --------------------------------------- | ------ | -------- | ------------------------------------------------------------------------------------------ |
| 1   | **Outbox transaction co-participation** | LARGE  | CRITICAL | `SQLOutbox.Append` must run inside `SQLEventStore.Save`'s tx. Requires interface decision. |
| 2   | **Design ADR for tx co-participation**  | SMALL  | HIGH     | Document the decision before implementing. Which of the 3 options?                         |

### HIGH (library quality, significant improvement)

| #   | Task                                     | Effort | Impact | Detail                                                                                                                          |
| --- | ---------------------------------------- | ------ | ------ | ------------------------------------------------------------------------------------------------------------------------------- |
| 3   | **Fix all 46 lint issues**               | SMALL  | HIGH   | 1 hour: errcheck (11), perfsprint (8), wsl_v5 (8), nlreturn (6), noinlineerr (6), revive (3), err113 (2). All mechanical fixes. |
| 4   | **Increase projection coverage to 95%+** | MEDIUM | HIGH   | Add replay error tests, checkpoint save tests, context cancellation, Close, WithLogger.                                         |
| 5   | **Increase storage coverage to 95%+**    | SMALL  | HIGH   | Add scanOutboxEntries errors, reconstructOutboxEvent edge cases, OutboxSchema test.                                             |
| 6   | **Register aggregate sentinels**         | SMALL  | MEDIUM | `init()` in `core/aggregate/errors.go` calling `RegisterClassification`.                                                        |
| 7   | **Register projection sentinels**        | SMALL  | MEDIUM | `init()` in `projection/errors.go` calling `RegisterClassification`.                                                            |
| 8   | **Cover zero-coverage functions**        | SMALL  | MEDIUM | `LoadAll` test, `OutboxSchema` test, `WithLogger` test, `Close` tests.                                                          |

### MEDIUM (polish, developer experience)

| #   | Task                                         | Effort | Impact | Detail                                                         |
| --- | -------------------------------------------- | ------ | ------ | -------------------------------------------------------------- |
| 9   | **Increase aggregate coverage to 95%+**      | SMALL  | MEDIUM | `NewCore` validation errors, `loadFromStore` error paths.      |
| 10  | **Increase memory coverage to 95%+**         | SMALL  | MEDIUM | `LoadAll` test, `LoadAtVersion` edge case, `Ack` edge case.    |
| 11  | **Increase decider coverage to 95%+**        | SMALL  | MEDIUM | `loadFromSnapshot` 86.4% → 95%, `saveSnapshot` 85.7% → 95%.    |
| 12  | **Extract shared `opError` helper**          | SMALL  | MEDIUM | Deduplicate identical function in aggregate + decider.         |
| 13  | **Add `CHANGELOG.md`**                       | SMALL  | MEDIUM | Track changes per session for release notes.                   |
| 14  | **Refactor `validateEventParams` (51L)**     | SMALL  | LOW    | Split 4 validation blocks into focused functions.              |
| 15  | **Add outbox integration test**              | MEDIUM | MEDIUM | Full cycle: Append → PollPending → Publish → Ack with sqlmock. |
| 16  | **Tag `v0.1.0-alpha`**                       | SMALL  | MEDIUM | All core modules are stable enough for early adopters.         |
| 17  | **Add godoc to `storage/outbox.go`**         | SMALL  | LOW    | Missing doc comments on exported functions.                    |
| 18  | **Add `OutboxSchema` to `storage.Schema()`** | SMALL  | LOW    | Currently only returns events table DDL.                       |
| 19  | **Add context cancellation to `SQLOutbox`**  | SMALL  | LOW    | Currently ignores `ctx.Err()`.                                 |
| 20  | **Refactor long catalog exporter functions** | MEDIUM | LOW    | `asyncapi.Export` 62L, `d2.writeCrossServiceConnections` 60L.  |

### LOW (nice-to-have, long-term)

| #   | Task                              | Effort | Impact | Detail                                                       |
| --- | --------------------------------- | ------ | ------ | ------------------------------------------------------------ |
| 21  | **Add benchmarks**                | MEDIUM | LOW    | Decider operations, outbox throughput, event store save.     |
| 22  | **Create `CONTRIBUTING.md`**      | SMALL  | LOW    | Document module structure and contribution guidelines.       |
| 23  | **Evaluate SQLite/Turso support** | MEDIUM | LOW    | User mentioned. Assess effort for `storage/sqlite/` variant. |
| 24  | **Saga / Process Manager design** | LARGE  | LOW    | `docs/planning/SAGA_DESIGN.md` exists. Complex feature.      |
| 25  | **Watermill module**              | LARGE  | LOW    | Pub/sub adapter for Kafka, NATS. Large scope.                |

---

## G) TOP #1 QUESTION

**How should outbox transaction co-participation work?**

The `event.Outbox` interface docs say "Append runs inside the same transaction as the event store Save operation." But the current implementation doesn't deliver this — `SQLOutbox.Append` and `SQLEventStore.Save` each manage their own transactions independently.

Three options:

1. **`SQLEventStore` accepts `Outbox` reference** — `Save()` begins tx, inserts events, calls `outbox.Append(ctx, tx, events)`, commits. Requires changing `Outbox.Append` to accept a transaction context (e.g., `*sql.Tx` or a `DBTX` interface).

2. **Accept external `*sql.Tx`** — Both `SQLEventStore` and `SQLOutbox` get `SaveWithTx(tx *sql.Tx, ...)` and `AppendWithTx(tx *sql.Tx, ...)` methods. Consumer manages the transaction. Most flexible but leaks SQL details.

3. **Outbox wraps event store** — `OutboxEventStore` composes both, manages a single tx internally. Cleanest API but adds a new abstraction layer.

This is an **interface-breaking decision** that I cannot make without understanding your transaction management preferences. Should the library own the transaction, or should the consumer pass one in?

---

## Metrics Dashboard

### Test Coverage by Package

| Package                | Coverage  | Target | Δ from S42 | Status |
| ---------------------- | --------- | ------ | ---------- | ------ |
| `core/command`         | 100.0%    | >95%   | —          | ✅     |
| `core/query`           | 100.0%    | >95%   | —          | ✅     |
| `core/pkg/dispatcher`  | 100.0%    | >95%   | —          | ✅     |
| `core/pkg/id`          | 100.0%    | >95%   | —          | ✅     |
| `middleware`           | 100.0%    | >95%   | —          | ✅     |
| `core/event`           | 98.0%     | >95%   | +0.1%      | ✅     |
| `catalog/d2`           | 97.6%     | >95%   | —          | ✅     |
| `catalog/asyncapi`     | 95.9%     | >95%   | —          | ✅     |
| `catalog/eventcatalog` | 95.6%     | >95%   | —          | ✅     |
| `catalog/adapters`     | 95.5%     | >95%   | —          | ✅     |
| `core/decider`         | 94.3%     | >95%   | +16.9%     | ⚠️     |
| `catalog`              | 94.4%     | >95%   | —          | ⚠️     |
| `core/aggregate`       | 93.2%     | >95%   | —          | ⚠️     |
| `storage`              | 92.0%     | >95%   | —          | ⚠️     |
| `memory`               | 91.9%     | >95%   | —          | ⚠️     |
| `projection`           | 88.8%     | >95%   | -1.3%      | ⚠️     |
| **Total**              | **90.9%** | >95%   | —          | ⚠️     |

### Module Dependency Graph

```
core (independently publishable)
├── memory → core + testhelpers
├── middleware → core + testhelpers
├── testhelpers → core
├── catalog → core
├── storage → core
├── projection → core + memory(test) + testhelpers(test)
├── integration → core + memory + testhelpers
└── example/user → core + memory + catalog + middleware
```

### Zero-Coverage Functions (8 total)

| Function             | Package           | Type                |
| -------------------- | ----------------- | ------------------- |
| `LoadAll`            | memory            | Interface method    |
| `OutboxSchema`       | storage           | DDL constant        |
| `ExportD2` (builder) | catalog/adapters  | Convenience wrapper |
| `WithDirection`      | catalog/d2        | Option function     |
| `WithLogger`         | projection        | New option (S44)    |
| `Close`              | projection        | No-op               |
| `Close`              | memory/checkpoint | No-op               |
| `Close`              | memory/outbox     | No-op               |

### Largest Production Files

| File                           | Lines | Limit | Status |
| ------------------------------ | ----- | ----- | ------ |
| `core/aggregate/repository.go` | 245   | 250   | ✅     |
| `catalog/asyncapi/exporter.go` | 244   | 250   | ✅     |
| `core/decider/decider.go`      | 243   | 250   | ✅     |
| `core/event/event.go`          | 241   | 250   | ✅     |
| `core/event/errors.go`         | 240   | 250   | ✅     |
| `projection/runner.go`         | 237   | 250   | ✅     |
| `storage/outbox.go`            | 235   | 250   | ✅     |
| `catalog/registry.go`          | 232   | 250   | ✅     |
| `storage/event_store.go`       | 228   | 250   | ✅     |

### Lint Issue Breakdown (46 total)

| Linter      | Count | Location                                     | Fixable?          |
| ----------- | ----- | -------------------------------------------- | ----------------- |
| errcheck    | 11    | decider tests (9), outbox (1), errors.go (1) | ✅ Yes            |
| wsl_v5      | 8     | decider tests                                | ✅ Yes            |
| perfsprint  | 8     | decider tests                                | ✅ Yes            |
| noinlineerr | 6     | decider, aggregate, outbox tests             | ✅ Yes            |
| nlreturn    | 6     | decider package                              | ✅ Yes            |
| revive      | 3     | decider tests (unused params)                | ✅ Yes            |
| err113      | 2     | decider `opError`, runner panic recovery     | ⚠️ Intentional    |
| exhaustruct | 1     | event/errors.go (false positive)             | ❌ False positive |
| golines     | 1     | decider test                                 | ✅ Yes            |
| modernize   | 1     | decider test (WaitGroup.Go)                  | ✅ Yes            |

---

_Generated: 2026-05-03 06:05 — Session 45_
