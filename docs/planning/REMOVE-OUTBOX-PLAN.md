# Plan: Remove Outbox from go-cqrs-lite

**Date:** 2026-05-29
**Status:** COMPLETE
**Rationale:** See `docs/outbox-explained.html` §12–13. The `SeekableJournal` + `CheckpointStore` already solve crash recovery without an extra table. The outbox is over-scoped for the common case. YAGNI.

---

## Execution Order

Phases are ordered by dependency: leaf modules first, consumers last. Each phase must pass `nix run .#test` before proceeding.

---

### Phase 1: Delete leaf outbox implementations (no consumers depend on these)

**Delete files:**

| File                              | Module      |
| --------------------------------- | ----------- |
| `memory/outbox.go`                | memory      |
| `memory/outbox_test.go`           | memory      |
| `testhelpers/fake_outbox.go`      | testhelpers |
| `testhelpers/fake_outbox_test.go` | testhelpers |

**After:** Run `GOWORK=off go test ./...` in `memory/` and `testhelpers/`.

---

### Phase 2: Delete storage outbox implementation + TransactionalStore

The storage module is the heaviest consumer. Remove in this order:

**2a. Delete outbox files in storage:**

| File                                  | What                                                                                                                |
| ------------------------------------- | ------------------------------------------------------------------------------------------------------------------- |
| `storage/outbox.go`                   | `SQLOutbox`, `NewSQLOutbox`, `NewSQLiteOutbox`, `NewSQLOutboxWithDialect`, `OutboxSchema()`, `SQLiteOutboxSchema()` |
| `storage/outbox_test.go`              | All `TestSQLOutbox_*` tests                                                                                         |
| `storage/outbox_poller.go`            | `OutboxPoller`, `NewOutboxPoller`, `WithPollInterval`, `WithBatchSize`, `WithPollerLogger`                          |
| `storage/outbox_poller_test.go`       | All `TestOutboxPoller_*` tests                                                                                      |
| `storage/outbox_helpers.go`           | `outboxEvent`, `marshalOutboxEvents`, `unmarshalOutboxEvents`, `scanOutboxEntries`                                  |
| `storage/transactional_store.go`      | `SQLTransactionalStore`, `NewSQLTransactionalStore`, `SaveWithOutbox`                                               |
| `storage/transactional_store_test.go` | All `TestSQLTransactionalStore_*` tests                                                                             |

**2b. Update `storage/sql_backend.go`:**

- Remove `outbox *SQLOutbox` and `tx *SQLTransactionalStore` fields from `SQLBackend`
- Remove outbox creation from `newSQLBackendWithDialect()`
- Remove `Outbox()` method
- Remove `TransactionalSink()` method
- `SQLBackend` becomes a simpler wrapper around `SQLEventStore` only

**2c. Update `storage/sql_backend_test.go`:**

- Remove `TestSQLBackend_TransactionalStore` and `TestSQLBackend_SaveWithOutbox`
- Remove outbox-related assertions in other tests

**2d. Update `storage/sql/reconstruction.go`:**

- Remove `SaveWithOutboxTx()` function

**2e. Update `storage/sql/dialect.go`:**

- Remove `OutboxSchema()` from `Dialect` interface
- Remove `OutboxSchema()` implementations from `PostgresDialect` and `SQLiteDialect`

**2f. Update `storage/sql/errors.go`:**

- Remove `OutboxStatus` type, `OutboxStatusPending`, `OutboxStatusAcked`

**2g. Update `storage/sql/tables.go`:**

- Remove `TableOutbox` constant

**2h. Update `storage/sql/helpers.go`:**

- Remove `SharedAckBatch()` function
- Remove `OutboxInsertSQL()` function

**2i. Update `storage/sqlite_helpers.go`:**

- Remove `pg.OutboxSchema()` from `PostgresInitSchema()` DDL list

**2j. Update `storage/doc.go`:**

- Remove `OutboxStatus` type alias, `OutboxStatusPending`, `OutboxStatusAcked` vars

**2k. Update `storage/testhelpers.go`:**

- Remove `ExpectOutboxInsert()`, `ExpectOutboxInsertError()`

**2l. Update `storage/event_store_loadall_test.go`:**

- Remove assertion that `Schema()` contains `CREATE TABLE outbox`

**2m. Update `storage/store_testsuite_test.go`:**

- Remove `testOutbox_Roundtrip()` function (already unused)

**2n. Update `storage/sqlite_integration_helpers_test.go`:**

- Remove `TestNewSQLiteOutbox_NilDB`
- Remove `TestNewSQLTransactionalStore_NilStore_SQLite`
- Remove `TestNewSQLTransactionalStore_NilOutbox_SQLite`

**2o. Update `storage/options_test.go`:**

- Remove `TestOutboxStatus_String`

**After:** `GOWORK=off go test ./...` in `storage/` must pass.

---

### Phase 3: Delete core/event outbox abstractions

**3a. Delete files:**

| File                                            | What                                                                         |
| ----------------------------------------------- | ---------------------------------------------------------------------------- |
| `core/event/outbox.go`                          | `Outbox` interface, `OutboxID`, `NewOutboxID`, `OutboxEntry`                 |
| `core/event/outbox_publisher.go`                | `OutboxPublisher`, `NewOutboxPublisher`, `WithPollInterval`, `WithBatchSize` |
| `core/event/outbox_publisher_helpers_test.go`   | Helper tests                                                                 |
| `core/event/outbox_publisher_publish_test.go`   | Publish tests                                                                |
| `core/event/outbox_publisher_lifecycle_test.go` | Lifecycle tests                                                              |

**3b. Update `core/event/errors.go`:**

- Remove `ErrNilOutbox`, `ErrAlreadyStarted`, `ErrPublisherClosed`

**3c. Update `core/event/store.go`:**

- Remove `TransactionalSink` interface (only exists for `SaveWithOutbox`)

**After:** `GOWORK=off go test ./...` in `core/` must pass.

---

### Phase 4: Update decider (core consumer of outbox)

**4a. Update `core/decider/options.go`:**

- Remove `WithOutbox()` option

**4b. Update `core/decider/decider.go`:**

- Remove `outbox event.Outbox` field from `Repository[State]`
- Remove the `TransactionalSink` type assertion branch in `Execute()` — just use `store.Save()` always
- The `Execute` method simplifies to: Load → Fold → Decide → Save → Publish

**4c. Update `core/decider/decider_coverage_test.go`:**

- Remove `fakeTransactionalStore` type
- Remove `TestExecute_TransactionalStore_SaveWithOutboxError`

**4d. Update `core/decider/decider_execute_test.go`:**

- Remove `TestExecute_WithOutbox` and `TestExecute_WithOutboxAndTransactionalStore` tests
- Remove any `WithOutbox` usage

**After:** `GOWORK=off go test ./...` in `core/` must pass.

---

### Phase 5: Update turso module

**5a. Update `turso/connector.go`:**

- Remove `NewTursoTransactionalStore()` function

**After:** `GOWORK=off go test ./...` in `turso/` must pass.

---

### Phase 6: Update documentation

**6a. Delete ADR:**

- Delete `docs/adr/0005-outbox-pattern.md`

**6b. Update `docs/STORAGE_GUIDE.md`:**

- Remove "Outbox Pattern" section
- Remove `SaveWithOutbox` from event store operations table
- Remove outbox from setup code examples

**6c. Update `storage/README.md`:**

- Remove outbox references

**6d. Update `AGENTS.md`:**

- Remove outbox from module descriptions, interface list, key patterns
- Remove `TransactionalSink` from ISP list
- Remove outbox from module graph description
- Add note that crash recovery uses `SeekableJournal` + `CheckpointStore` pattern instead

**6e. Update `docs/outbox-explained.html`:**

- Add a deprecation banner at the top: "The outbox has been removed from go-cqrs-lite. This document is kept for historical reference."

**6f. Update status docs:**

- Any file in `docs/status/` referencing outbox should note removal in next status report

---

### Phase 7: Clean up go.mod files

**7a. Check for unused imports:**
After removing all outbox code, run `go mod tidy` in every module that had outbox-related imports:

- `storage/` — may be able to drop codec import if only used by outbox helpers
- `memory/` — may be able to drop `core/pkg/dispatcher` if only used by `MemoryOutboxStore`

**7b. Verify:**

```bash
nix run .#build
nix run .#test
nix run .#lint
```

---

## Files Changed Summary

### DELETE (entire files)

| #   | File                                            | Lines |
| --- | ----------------------------------------------- | ----- |
| 1   | `core/event/outbox.go`                          | ~50   |
| 2   | `core/event/outbox_publisher.go`                | ~210  |
| 3   | `core/event/outbox_publisher_helpers_test.go`   | ~?    |
| 4   | `core/event/outbox_publisher_publish_test.go`   | ~?    |
| 5   | `core/event/outbox_publisher_lifecycle_test.go` | ~?    |
| 6   | `storage/outbox.go`                             | ~180  |
| 7   | `storage/outbox_test.go`                        | ~460  |
| 8   | `storage/outbox_poller.go`                      | ~160  |
| 9   | `storage/outbox_poller_test.go`                 | ~?    |
| 10  | `storage/outbox_helpers.go`                     | ~130  |
| 11  | `storage/transactional_store.go`                | ~75   |
| 12  | `storage/transactional_store_test.go`           | ~230  |
| 13  | `memory/outbox.go`                              | ~110  |
| 14  | `memory/outbox_test.go`                         | ~?    |
| 15  | `testhelpers/fake_outbox.go`                    | ~85   |
| 16  | `testhelpers/fake_outbox_test.go`               | ~?    |
| 17  | `docs/adr/0005-outbox-pattern.md`               | ~53   |

### EDIT (surgical changes)

| #   | File                                         | Change                                         |
| --- | -------------------------------------------- | ---------------------------------------------- |
| 1   | `core/event/errors.go`                       | Remove 3 error sentinels                       |
| 2   | `core/event/store.go`                        | Remove `TransactionalSink` interface           |
| 3   | `core/decider/decider.go`                    | Remove `outbox` field, simplify `Execute()`    |
| 4   | `core/decider/options.go`                    | Remove `WithOutbox()`                          |
| 5   | `core/decider/decider_coverage_test.go`      | Remove `fakeTransactionalStore`, 1 test        |
| 6   | `core/decider/decider_execute_test.go`       | Remove 2 tests with `WithOutbox`               |
| 7   | `storage/sql_backend.go`                     | Remove outbox/tx fields and methods            |
| 8   | `storage/sql_backend_test.go`                | Remove outbox-related tests                    |
| 9   | `storage/sql/reconstruction.go`              | Remove `SaveWithOutboxTx()`                    |
| 10  | `storage/sql/dialect.go`                     | Remove `OutboxSchema()` from interface + impls |
| 11  | `storage/sql/errors.go`                      | Remove `OutboxStatus` type + consts            |
| 12  | `storage/sql/tables.go`                      | Remove `TableOutbox`                           |
| 13  | `storage/sql/helpers.go`                     | Remove `SharedAckBatch()`, `OutboxInsertSQL()` |
| 14  | `storage/sqlite_helpers.go`                  | Remove `OutboxSchema()` from init DDL          |
| 15  | `storage/doc.go`                             | Remove outbox type/const aliases               |
| 16  | `storage/testhelpers.go`                     | Remove `ExpectOutboxInsert*`                   |
| 17  | `storage/event_store_loadall_test.go`        | Remove outbox DDL assertion                    |
| 18  | `storage/store_testsuite_test.go`            | Remove `testOutbox_Roundtrip()`                |
| 19  | `storage/sqlite_integration_helpers_test.go` | Remove 3 outbox tests                          |
| 20  | `storage/options_test.go`                    | Remove `TestOutboxStatus_String`               |
| 21  | `turso/connector.go`                         | Remove `NewTursoTransactionalStore()`          |
| 22  | `AGENTS.md`                                  | Remove outbox references, add checkpoint note  |
| 23  | `docs/STORAGE_GUIDE.md`                      | Remove outbox section                          |
| 24  | `docs/outbox-explained.html`                 | Add deprecation banner                         |

---

## Risk Assessment

- **Breaking change:** Yes — `event.Outbox`, `event.TransactionalSink`, `WithOutbox()`, `SaveWithOutbox()`, `SQLOutbox`, `MemoryOutboxStore`, `FakeOutbox`, `OutboxPublisher`, `OutboxPoller` all removed from public API.
- **Migration path:** Consumers using outbox should switch to `SeekableJournal.ReadFrom()` + `CheckpointStore` for crash recovery (same pattern as `projection.Runner`).
- **No data loss risk:** Removing the outbox code doesn't affect existing databases — the `outbox` table simply becomes unused and can be dropped manually.

---

## Estimated Time

| Phase                   | Time        | Risk                                |
| ----------------------- | ----------- | ----------------------------------- |
| Phase 1: Leaf modules   | 5 min       | None                                |
| Phase 2: Storage        | 30 min      | Medium (most files, schema changes) |
| Phase 3: Core/event     | 10 min      | Low                                 |
| Phase 4: Decider        | 10 min      | Low                                 |
| Phase 5: Turso          | 2 min       | None                                |
| Phase 6: Docs           | 15 min      | None                                |
| Phase 7: go.mod cleanup | 5 min       | Low                                 |
| **Total**               | **~75 min** |                                     |
