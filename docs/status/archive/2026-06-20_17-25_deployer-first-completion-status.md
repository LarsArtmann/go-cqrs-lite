# Status Report — 2026-06-20 17:25

## Deployer-First Architecture Completion

**Branch:** master
**Plan:** [Deployer-First Completion Plan](../planning/2026-06-20_17-03_DEPLOYER-FIRST-COMPLETION-PLAN.md)
**Goal:** "The Best CQRS/ES SDK in Go" — composable, powerful, deployer-first

---

## A. Fully Done (verified against code)

### Phase 0: Critical Bug Fixes (T01-T03) ✅

| Task | What shipped                                                                                                                                                                                                                                                                                                                                                                                                      | Commit     |
| ---- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ---------- |
| T01  | **Materialize error-misclassification fix.** `stack/materialize.go:132` — any `Store.Get` error was silently classified as "not found" → `OnCreate`, causing silent data corruption on real DB errors. Now uses `errors.Is(err, kv.ErrNotFound)` + propagates other errors. Also fixed tombstone path (line 104) which had the same pattern. Test added: `TestMaterialize_StoreGetErrorPropagates`.               | `72538df7` |
| T02  | **Version.Add(n uint) underflow fix.** `Add(n int)` accepted negative values which silently wrapped to huge uint64 after the int→uint64 change. Changed to `Add(n uint)` — type system prevents negative input at compile time. Fixed all callers (`event/batch.go`, `decider/decider.go`, `decider/load.go`, `event/parser_fuzz_test.go`). Also fixed `storage/pg_bus_listen.go` `np.Version-1` underflow guard. | `72538df7` |
| T03  | **V3_MIGRATION.md fixed.** Broken `memory.NewMemoryBus()` example replaced with working `watermill.NewEventBus()`. Ghost-code lie corrected: `storage/pg_bus.go` is LIVE code (ADR-0027), not ghost. Status table updated: Version→uint64 Done, memory/ move Done, readmodel deletion Done, io.Closer Rejected.                                                                                                   | `72538df7` |

### Phase 1: Kill readmodel↔kv Split Brain (T04-T06) ✅

| Task    | What shipped                                                                                                                                                                                                                                                                                                                                                                                                               | Commit     |
| ------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ---------- |
| T04-T06 | **readmodel/ module deleted entirely.** 482 LOC, 14 files, 2 Go modules eliminated. Types were character-for-character identical to `kv.TypedStore` and `kv.Cache` (6 clone groups, 27% of all duplication). Zero production Go code imported readmodel types — only go.mod replace directives and one bench test. Bench test redirected to `kv.NewTypedStore`. go.work, 7 go.mod files, api-stability golden all updated. | `c4d0ffcd` |

### Phase 2: Eliminate Remaining Duplication (T07-T09) ✅

| Task | What shipped                                                                                                                                                                                                                                                                                                           | Commit     |
| ---- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ---------- |
| T07  | **buildOptions dedup — SKIPPED (intentionally).** The 20-line duplication between `stack/sqlite` and `stack/postgres` is structural isolation. Extracting would require a cross-module dependency (`stack/` → `storage/`) that doesn't exist. Tolerable.                                                               | —          |
| T08  | **CBOR decoder dedup.** Exported `codec.CBORDecMode()` so `storage/pebble` reuses the canonical decoder instead of creating a duplicate. 1 clone group eliminated.                                                                                                                                                     | `dcd350af` |
| T09  | **WithEventDB/WithQueryDB wired.** Were dead options that set config fields `newBundle()` never read. Added `openSecondaryStores()` helper that opens a secondary SQLite DB, creates a backend, and returns `stack.Option` values for event/command/query/snapshot/checkpoint stores. Added `multiCloser` for cleanup. | `dcd350af` |

### Previously Completed (before this session)

| Task                            | What shipped                                                                                                     | Commit                  |
| ------------------------------- | ---------------------------------------------------------------------------------------------------------------- | ----------------------- |
| Watermill EventBus              | Default bus in all 4 presets (memory/sqlite/pebble/postgres). GoChannel-backed, multi-process via `WithBackend`. | `85d5eb70`              |
| PostgresBus (ADR-0027)          | LISTEN/NOTIFY with auto-reconnect, channel validation, real-Postgres integration tests.                          | `79d1f6a6`              |
| Version uint64                  | Negative versions impossible at type level.                                                                      | `c7df1c79`              |
| Storage consolidation           | `memory/`, `pebble/`, `turso/` all under `storage/`.                                                             | `22ecccd0`              |
| Materialize + CatchUpSubscriber | New projection API types built.                                                                                  | `98ebd0b3`              |
| Typed metadata fields           | `Tracing`, `TombstoneMark`, `Causation` structs added with dual-write.                                           | `02f7eaa8`              |
| Ghost bus deletion              | `memory/bus.go`, `memory/command_bus.go` deleted.                                                                | `8b6684ef`              |
| Docs wipe + restore             | 564 docs wiped then restored.                                                                                    | `f276873a` + `9c544d6f` |

---

## B. Partially Done

| Task                              | Status                             | What remains                                                                  |
| --------------------------------- | ---------------------------------- | ----------------------------------------------------------------------------- |
| **Materialize wired into stack/** | Code exists but no accessor        | T10: Add `stack.Materialize[V,K]()` accessor to make it reachable from Bundle |
| **CatchUpSubscriber**             | 297 LOC exists but no consumer     | T11: Add stack option to wire it                                              |
| **TypedDecider**                  | Exists but not surfaced from stack | T12: Add `stack.TypedRepository()` accessor                                   |
| **AuditMiddleware**               | Exists but not preset-wired        | T13: Wire into sqlite preset                                                  |
| **Metadata alias break**          | Typed fields added; aliases intact | T22-T24: Break `command.Metadata` and `query.Metadata` aliases                |
| **Projection dissolution**        | Replacement types exist            | T17-T21: Migrate examples, delete projection/                                 |
| **Deployer-first example**        | Not started                        | T14-T16: Build example validating the architecture                            |

---

## C. Not Started

| Task    | Description                                                 |
| ------- | ----------------------------------------------------------- |
| T14-T16 | Deployer-first example (validates architecture end-to-end)  |
| T17-T21 | Dissolve projection/ (~1343 LOC, biggest architectural lie) |
| T22-T24 | Kill metadata aliases (command/query own structs)           |
| T25-T27 | ADR status updates, TODO_LIST fixes, FEATURES/ROADMAP       |

---

## D. What We Fucked Up (and Fixed)

1. **Docs wipe (f276873a)** — Deleted 564 docs by directory pattern without verifying staleness. Restored in `9c544d6f`. Lesson: always grep before deleting.
2. **Ghost-code lie** — V3_MIGRATION labeled `storage/pg_bus.go` as ghost. It's live code wired at `stack/postgres/preset.go:161`. Fixed in T03.
3. **Materialize data corruption** — Any `Store.Get` error silently triggered `OnCreate`, overwriting real data. Fixed in T01.
4. **Version.Add underflow** — `Add(n int)` accepted negatives after uint64 change. Fixed in T02.
5. **Dead WithEventDB/WithQueryDB** — Options that silently did nothing. Wired in T09.
6. **io.Closer removal attempt** — Would have destroyed type safety (verschlimmbessern). Correctly stopped and reverted.
7. **TransactionID ghost type** — `id/transaction_id.go` exists, zero consumers, TODO marks it DONE. Still unfixed (T26).

---

## E. What We Should Improve

1. **Stop half-applying ADRs.** Three ADRs (0030, 0031, 0032) had destinations built but sources not deleted. ADR-0032 now completed (readmodel deleted). 0030 and 0031 remain half-applied.
2. **Wire before deleting.** The projection dissolution requires the replacement (Materialize + CatchUpSubscriber) to be reachable from stack/ first.
3. **Validate with examples.** The deployer-first example (T14-T16) is the safety net for all subsequent dissolution.
4. **Test error paths.** Materialize's bug existed because the fake store only returned ErrNotFound. Inject real errors; assert they propagate.
5. **Keep the FakeBus simple.** 130 LOC of middleware-chain logic in a test helper is a parallel implementation. Consider extracting or shrinking.

---

## F. Top 25 Things to Get Done Next

| #   | Task                                            | Impact   | Effort   |
| --- | ----------------------------------------------- | -------- | -------- |
| 1   | T10: Add `stack.Materialize()` accessor         | High     | 20min    |
| 2   | T11: Add `stack.CatchUpSubscriber()` option     | High     | 20min    |
| 3   | T12: Add `stack.TypedRepository()` accessor     | High     | 20min    |
| 4   | T13: Wire AuditMiddleware into sqlite preset    | Medium   | 15min    |
| 5   | T14: Build example/deployer-first consumer code | Critical | 40min    |
| 6   | T15: Build deployer configs (SQLite multi-DB)   | High     | 30min    |
| 7   | T16: Write integration test for all configs     | High     | 20min    |
| 8   | T17: Migrate example/todo to Materialize        | Critical | 30min    |
| 9   | T18: Migrate example/user to Materialize        | High     | 25min    |
| 10  | T19: Update cqrs-gen for Materialize codegen    | Medium   | 20min    |
| 11  | T20: Remove ProjectionRunner from stack/        | High     | 15min    |
| 12  | T21: Delete projection/ module                  | Critical | 20min    |
| 13  | T22: command.Metadata own struct                | High     | 25min    |
| 14  | T23: query.Metadata own struct                  | High     | 25min    |
| 15  | T24: Update storage scan helpers                | High     | 15min    |
| 16  | T25: Update ADR statuses                        | Medium   | 10min    |
| 17  | T26: Delete TransactionID ghost type            | Medium   | 15min    |
| 18  | T27: Update FEATURES.md + ROADMAP.md            | Low      | 15min    |
| 19  | Fix CBOR fuzz test (duplicate map key -17)      | Medium   | 30min-2h |
| 20  | Add rapid property tests for Version            | Medium   | 20min    |
| 21  | Fix PgxListener race condition                  | Medium   | 45min    |
| 22  | Consider go-cache/otter for dedup               | Low      | 1h       |
| 23  | Extract FakeBus to syncbus module               | Low      | 1h       |
| 24  | Move indexing advisor to storage/sql/           | Low      | 30min    |
| 25  | encoding/json/v2 migration                      | Low      | 2h       |

---

## G. The One Big Question

**Should we dissolve projection/ now, or wait for v3?**

The projection/ module is 1343 LOC with 20 production references across examples, stack, integration tests, and cqrs-gen. Dissolving it requires:

1. Migrating example/todo and example/user to Materialize + Watermill Router
2. Updating cqrs-gen to emit Materialize code
3. Removing ProjectionRunner from stack/
4. Updating all integration tests

The risk: if the Materialize + CatchUpSubscriber replacement has a bug we haven't caught, dissolution breaks every example. The mitigation: build the deployer-first example first (T14-T16), which validates the new architecture end-to-end before any deletion.

**Answer: Build the example first. Then dissolve. The example is the safety net.**

---

## Progress Summary

| Phase                   | Tasks  | Status                | Effort invested    |
| ----------------------- | ------ | --------------------- | ------------------ |
| 0 — Bug Fixes           | 3      | ✅ Done               | ~45min             |
| 1 — Kill readmodel      | 3      | ✅ Done               | ~35min             |
| 2 — Dedup               | 3      | ✅ Done (T07 skipped) | ~50min             |
| 3 — Wire Islands        | 4      | 🔄 In progress        | 0min               |
| 4 — Example             | 3      | ⏳ Not started        | 0min               |
| 5 — Dissolve projection | 5      | ⏳ Not started        | 0min               |
| 6 — Kill aliases        | 3      | ⏳ Not started        | 0min               |
| 7 — Docs                | 3      | ⏳ Not started        | 0min               |
| **Total**               | **27** | **9/27 done (33%)**   | **~2.5h of ~8.5h** |
