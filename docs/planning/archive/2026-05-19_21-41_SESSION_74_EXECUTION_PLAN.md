# Session 74 — Comprehensive Execution Plan

**Created**: 2026-05-19 21:41
**Source**: Deep research across all modules — architecture, types, libraries, tests, lint

---

## What I Forgot / Could Do Better / Should Improve

### Forgotten

1. `catalog/openapi/exporter.go` at 253 lines — 3 over the 250 limit (from the openapi split, the INSERT SQL template is large)
2. `storage/transactional_store.go` — duplicated outbox INSERT SQL (same query in `outbox.go` and `transactional_store.go`)
3. SQL injection pattern in `outbox.go:83` and `transactional_store.go:97` — `OutboxStatusPending` string-interpolated instead of parameterized
4. `ErrVersionMismatch`, `ErrAggregateTypeMismatch`, `ErrAggregateIDMismatch` classified as `Corruption` — should be `Conflict`
5. Unused `testify` dependency in `catalog/go.mod`
6. `cattest/catalog.go` still uses deprecated types — can be simplified to use zero-cost API

### Could Do Better

1. **Error classification coupling** — `storage`, `middleware`, `projection` all call `event.RegisterClassification()`. Non-event code shouldn't need to import `event`. Should extract to `core/pkg/errors/` or similar.
2. **`WithMetadata` is destructive** — calling it after `WithCorrelationID` wipes the correlation ID. Should merge instead.
3. **Metadata key constants** — `"client.id"`, `"client.occurred_at"` are raw strings. Should be constants.

### Type Model Improvements

1. **`OutboxStatus` type has only one value** — add `OutboxStatusAcked` or remove the type
2. **`TransactionalStore.SaveWithOutbox` accepts an `Outbox` parameter it ignores** — misleading interface
3. **`SQLTransactionalStore` duplicates `db`/`dialect` fields** already on embedded `*SQLEventStore`

### Library Opportunities (REJECTED for now)

1. **SQL query builder (squirrel)** — Would eliminate 20+ `fmt.Sprintf` calls but adds a dependency and is a large refactor. Deferred.
2. **jsonschema (invopop/jsonschema)** — Current reflect-based schema generation works well for its scope. Deferred.
3. **HTML templates for docserver** — `fmt.Sprintf` for HTML is an XSS risk but developer-controlled strings. Deferred.

---

## Execution Plan — Sorted by Impact/Effort

### P0: Unblock + Critical Fixes (Tasks 1-4) — ~45min

| #   | Task                                                                                                                                              | Impact                                   | Effort | Customer Value |
| --- | ------------------------------------------------------------------------------------------------------------------------------------------------- | ---------------------------------------- | ------ | -------------- |
| 1   | **Fix error misclassifications**: `ErrVersionMismatch`→Conflict, `ErrAggregateTypeMismatch`→Conflict, `ErrAggregateIDMismatch`→Conflict           | HIGH — wrong retry behavior              | 5min   | Correctness    |
| 2   | **Parameterize OutboxStatusPending**: replace string interpolation with SQL parameter in outbox.go and transactional_store.go                     | HIGH — security audit failure            | 10min  | Security       |
| 3   | **Remove unused `testify` from catalog/go.mod**                                                                                                   | LOW                                      | 2min   | Hygiene        |
| 4   | **Simplify `cattest/catalog.go`**: rewrite `BuildTestCatalog()` using zero-cost API instead of deprecated `event.CatalogMeta`/`event.CatalogCore` | MEDIUM — removes 2 deprecated references | 10min  | Migration      |

### P1: Type Safety + Interface Cleanup (Tasks 5-9) — ~50min

| #   | Task                                                                                                                              | Impact                  | Effort | Customer Value  |
| --- | --------------------------------------------------------------------------------------------------------------------------------- | ----------------------- | ------ | --------------- |
| 5   | **Fix `TransactionalStore` interface**: remove ignored `outbox` parameter from `SaveWithOutbox`, accept at construction time      | HIGH — misleading API   | 15min  | API honesty     |
| 6   | **Remove duplicated `db`/`dialect` fields** from `SQLTransactionalStore` — use embedded `*SQLEventStore` fields                   | MEDIUM                  | 5min   | Consistency     |
| 7   | **Extract metadata key constants**: `"client.id"` → `MetadataKeyClientID`, `"client.occurred_at"` → `MetadataKeyClientOccurredAt` | MEDIUM — prevents typos | 8min   | Discoverability |
| 8   | **Fix `WithMetadata` to merge** instead of replace                                                                                | MEDIUM — ordering trap  | 10min  | Correctness     |
| 9   | **Add `OutboxStatusAcked` constant** for completeness                                                                             | LOW                     | 2min   | Completeness    |

### P2: Deduplication + File Size (Tasks 10-13) — ~35min

| #   | Task                                                                                                                               | Impact | Effort | Customer Value |
| --- | ---------------------------------------------------------------------------------------------------------------------------------- | ------ | ------ | -------------- |
| 10  | **Extract shared outbox INSERT SQL** to `storage/outbox_queries.go` — deduplicate between `outbox.go` and `transactional_store.go` | MEDIUM | 10min  | DRY            |
| 11  | **Extract `registerOutboxInsert` helper** in `storage/transactional_store.go` to share with `outbox.go`                            | MEDIUM | 5min   | DRY            |
| 12  | **Extract one helper from `catalog/openapi/exporter.go`** (253→<250)                                                               | LOW    | 5min   | Compliance     |
| 13  | **Remove `NewSQLiteTransactionalStore`** alias — no added behavior                                                                 | LOW    | 5min   | API surface    |

### P3: Lint Cleanup (Tasks 14-18) — ~40min

| #   | Task                                                                                    | Impact | Effort | Customer Value |
| --- | --------------------------------------------------------------------------------------- | ------ | ------ | -------------- |
| 14  | **Zero storage lint**: fix 2 err113, 1 gci, 2 staticcheck                               | MEDIUM | 15min  | Lint           |
| 15  | **Zero storage lint**: add nolint for 13 mnd (magic numbers in SQL column counts)       | LOW    | 5min   | Lint           |
| 16  | **Zero middleware lint**: add nolint:staticcheck for 2 deprecated CatalogMeta in tests  | LOW    | 3min   | Lint           |
| 17  | **Zero integration lint**: add nolint:staticcheck for 6 deprecated CatalogMeta in tests | LOW    | 5min   | Lint           |
| 18  | **Fix `catalog/go.mod` go mod tidy** warning for testify                                | LOW    | 2min   | Hygiene        |

### P4: Test Coverage Gaps (Tasks 19-22) — ~40min

| #   | Task                                                                | Impact | Effort | Customer Value  |
| --- | ------------------------------------------------------------------- | ------ | ------ | --------------- |
| 19  | **Add `WithMetadata` merge test** in `core/event/options_test.go`   | MEDIUM | 10min  | Correctness     |
| 20  | **Add metadata key constant tests** in `core/event/options_test.go` | LOW    | 5min   | Discoverability |
| 21  | **Add `TransactionalStore` tests** for the fixed interface          | MEDIUM | 15min  | Coverage        |
| 22  | **Add `OutboxStatus` constant test**                                | LOW    | 3min   | Completeness    |

### P5: Documentation (Tasks 23-24) — ~15min

| #   | Task                                             | Impact | Effort | Customer Value |
| --- | ------------------------------------------------ | ------ | ------ | -------------- |
| 23  | **Update AGENTS.md** with all Session 74 changes | MEDIUM | 10min  | Knowledge      |
| 24  | **Update CHANGELOG.md** with Session 74 entries  | MEDIUM | 5min   | History        |

---

## NOT Doing (With Reasons)

| Item                                                        | Reason                                                                                                              |
| ----------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------- |
| Delete deprecated `CatalogMeta`/`Catalogable`/`CatalogCore` | Blocked — dispatchers embed `CatalogDispatcher[Type, CatalogMeta]`. Requires dispatcher redesign. Separate session. |
| Extract error classification to standalone package          | Large refactor touching 7+ modules. Separate session.                                                               |
| Add squirrel SQL builder                                    | Large dependency addition + refactor. Separate session.                                                             |
| Replace `jsonschema` with `invopop/jsonschema`              | Current reflect schema works. Low ROI.                                                                              |
| Publish `testhelpers@v1.2.0`                                | Requires `git tag` + `git push` — awaiting user decision on release cadence.                                        |
| Fix `query.Handler` returns `any`                           | Breaking change requiring generics migration. Separate session.                                                     |
| Split large test files (24 files >350 lines)                | Low customer value, high effort. Separate session.                                                                  |

---

## Total: 24 tasks, ~185min estimated
