# Stream API v4 Execution Plan

> **Date:** 2026-05-28 | **Scope:** Sink/Source decomposition + Tombstones + stream/ read model module
> **Status:** Planning complete, ready for execution

---

## Pareto Summary

| Tier     | % of Tasks | % of Value | What                                                               |
| -------- | ---------- | ---------- | ------------------------------------------------------------------ |
| **1%**   | 4 tasks    | ~51% value | Tombstone types + Sink/Source interfaces + Remove Delete from core |
| **4%**   | 8 tasks    | ~64% value | + Update all store implementations + Interface assertions          |
| **20%**  | 16 tasks   | ~80% value | + Stream module core + Tests + Builder + Middleware                |
| **100%** | 45 tasks   | 100% value | + Full test coverage + Docs + Examples + Cleanup                   |

---

## Phase Breakdown

| Phase | Theme                                        | Tasks | Est. Total |
| ----- | -------------------------------------------- | ----- | ---------- |
| 0     | Git checkpoint + module scaffold             | 1–3   | 20 min     |
| 1     | Foundation: tombstone types                  | 4–7   | 40 min     |
| 2     | Decomposition: Sink/Source + remove Delete   | 8–17  | 100 min    |
| 3     | Stream module: core types + reader + builder | 18–29 | 120 min    |
| 4     | Stream module: middleware + projection + SQL | 30–37 | 80 min     |
| 5     | Tests: all new code                          | 38–42 | 55 min     |
| 6     | Documentation + examples + final verify      | 43–45 | 30 min     |

---

## Complete Task List (sorted by priority score = Impact×Value/Effort)

| #   | Phase | Task                                                                                                                                          | Files                                                         | Est (min) | Impact | Value | Effort | Score   |
| --- | ----- | --------------------------------------------------------------------------------------------------------------------------------------------- | ------------------------------------------------------------- | --------- | ------ | ----- | ------ | ------- |
| 1   | 1     | **Create `core/event/tombstone.go`** — TombstoneStatus enum (Active/Tombstoned/Undetermined), String(), IsActive(), IsTombstoned(), IsKnown() | `core/event/tombstone.go`                                     | 8         | 3      | 3     | 1      | **9.0** |
| 2   | 1     | **Add metadata keys + MarkTombstone/MarkRebirth helpers** — MetadataKeyTombstone, MetadataKeyRebirth, MarkTombstone(evt), MarkRebirth(evt)    | `core/event/tombstone.go`                                     | 8         | 3      | 3     | 1      | **9.0** |
| 3   | 1     | **Add DetectTombstone(events) function** — Inspect last event's metadata, return tri-state status                                             | `core/event/tombstone.go`                                     | 6         | 3      | 3     | 1      | **9.0** |
| 4   | 2     | **Define Sink interface** — io.Closer + Save() + AppendBatch(), no Delete                                                                     | `core/event/store.go`                                         | 5         | 3      | 3     | 1      | **9.0** |
| 5   | 2     | **Define Source interface** — io.Closer + Load() + LoadFromVersion() + LoadToVersion() + LoadToTimestamp()                                    | `core/event/store.go`                                         | 5         | 3      | 3     | 1      | **9.0** |
| 6   | 2     | **Update Store = Sink + Source composite** — Remove Delete, add deprecated alias for old Store                                                | `core/event/store.go`                                         | 5         | 3      | 3     | 1      | **9.0** |
| 7   | 2     | **Rename BackwardsLoader → BackwardsSource** — extends Source, add deprecated alias                                                           | `core/event/store.go`                                         | 4         | 3      | 3     | 1      | **9.0** |
| 8   | 2     | **Rename TransactionalStore → TransactionalSink** — extends Sink, add deprecated alias                                                        | `core/event/store.go`                                         | 4         | 3      | 3     | 1      | **9.0** |
| 9   | 2     | **Remove Delete from MemoryStore** — Delete method + tests, add Sink/Source assertions                                                        | `memory/store.go`, `memory/*_test.go`                         | 8         | 3      | 3     | 2      | **4.5** |
| 10  | 2     | **Remove Delete from SQLEventStore** — Delete method + tests, add Sink/Source assertions                                                      | `storage/event_store.go`, `storage/*_test.go`                 | 8         | 3      | 3     | 2      | **4.5** |
| 11  | 2     | **Remove Delete from FakeStore** — Delete method + tests, add Sink/Source assertions                                                          | `testhelpers/fake_store.go`                                   | 6         | 3      | 3     | 2      | **4.5** |
| 12  | 2     | **Remove Delete from PebbleEventStore** — Delete method + tests, add Sink/Source assertions                                                   | `storage/pebble_event_store.go`                               | 6         | 3      | 3     | 2      | **4.5** |
| 13  | 2     | **Add var \_ assertions for Journal/SeekableJournal** — MemoryStore, SQLEventStore, FakeStore, PebbleEventStore                               | `memory/store_load.go`, `storage/event_store_global.go`, etc. | 5         | 2      | 2     | 1      | **4.0** |
| 14  | 3     | **Create `stream/` module scaffold** — go.mod, doc.go, module init                                                                            | `stream/go.mod`, `stream/doc.go`                              | 5         | 3      | 3     | 1      | **9.0** |
| 15  | 3     | **Create `stream/types.go`** — AggregateRef, AggregateStatus, Page[T], TombstonePolicy, ListOptions                                           | `stream/types.go`                                             | 8         | 3      | 3     | 1      | **9.0** |
| 16  | 3     | **Create `stream/aggregate_reader.go`** — AggregateReader interface with List + ListWithStatus                                                | `stream/aggregate_reader.go`                                  | 5         | 3      | 3     | 1      | **9.0** |
| 17  | 3     | **Create `stream/builder.go`** — ListBuilder, NewListBuilder(reader), OfType, After, PageSize, IncludeDeleted, OnlyDeleted, List              | `stream/builder.go`                                           | 10        | 3      | 3     | 2      | **4.5** |
| 18  | 3     | **Create `stream/in_memory.go`** — InMemoryAggregateReader, NewInMemoryAggregateReader(journal), List, ListWithStatus                         | `stream/in_memory.go`                                         | 10        | 3      | 3     | 2      | **4.5** |
| 19  | 3     | **Create `stream/middleware.go`** — StatusMiddleware(deleteTypes, rebirthTypes) event.PublishMiddleware                                       | `stream/middleware.go`                                        | 10        | 3      | 3     | 2      | **4.5** |
| 20  | 4     | **Create `stream/projection.go`** — AggregateProjection, NewAggregateProjection(db, prefix), Handle, createTable                              | `stream/projection.go`                                        | 12        | 3      | 3     | 2      | **4.5** |
| 21  | 4     | **Create `stream/sql_reader.go`** — SQLAggregateReader, NewSQLAggregateReader(db, prefix), List, ListWithStatus                               | `stream/sql_reader.go`                                        | 10        | 3      | 3     | 2      | **4.5** |
| 22  | 5     | **Test tombstone types** — TombstoneStatus String/IsActive/IsTombstoned/IsKnown, DetectTombstone                                              | `core/event/tombstone_test.go`                                | 10        | 3      | 2     | 2      | **3.0** |
| 23  | 5     | **Test MarkTombstone + MarkRebirth** — Verify metadata keys set, original event unmodified                                                    | `core/event/tombstone_test.go`                                | 8         | 3      | 2     | 2      | **3.0** |
| 24  | 5     | **Test stream builder** — NewListBuilder, OfType, After, PageSize, IncludeDeleted, OnlyDeleted                                                | `stream/builder_test.go`                                      | 8         | 3      | 2     | 2      | **3.0** |
| 25  | 5     | **Test InMemoryAggregateReader** — List, ListWithStatus, pagination, TombstonePolicy filters                                                  | `stream/in_memory_test.go`                                    | 10        | 3      | 2     | 2      | **3.0** |
| 26  | 5     | **Test StatusMiddleware** — Auto-mark tombstone on delete types, auto-mark rebirth on rebirth types                                           | `stream/middleware_test.go`                                   | 10        | 3      | 2     | 2      | **3.0** |
| 27  | 2     | **Update projection/Runner for Sink/Source** — Ensure runner uses Journal (already done), check for Store type assertions                     | `projection/runner.go`                                        | 5         | 2      | 2     | 1      | **4.0** |
| 28  | 2     | **Update decider/Repository for Sink** — Type-assert to Sink instead of Store if needed                                                       | `core/decider/decider.go`                                     | 5         | 2      | 2     | 1      | **4.0** |
| 29  | 2     | **Check saga/ for Delete usage** — Remove if present, add Sink assertion                                                                      | `saga/*.go`                                                   | 5         | 2      | 2     | 1      | **4.0** |
| 30  | 3     | **Update go.work** — Add stream/ module to workspace                                                                                          | `go.work`                                                     | 3         | 2      | 2     | 1      | **4.0** |
| 31  | 6     | **Update example/user** — Remove Delete usage, show tombstone middleware pattern                                                              | `example/user/*.go`                                           | 10        | 2      | 3     | 2      | **3.0** |
| 32  | 6     | **Update AGENTS.md** — Document Sink/Source, tombstone pattern, stream/ module                                                                | `AGENTS.md`                                                   | 8         | 2      | 2     | 1      | **4.0** |
| 33  | 6     | **Update v4 proposal** — Mark as implemented, add any deviations                                                                              | `docs/research/2026-05-28_STREAM_API_V4_PROPOSAL.md`          | 5         | 1      | 1     | 1      | **1.0** |
| 34  | 0     | **Git checkpoint** — Commit any uncommitted changes before starting                                                                           | —                                                             | 3         | 3      | 3     | 1      | **9.0** |
| 35  | 5     | **Test memory store without Delete** — Update tests that relied on Delete                                                                     | `memory/*_test.go`                                            | 8         | 2      | 2     | 2      | **2.0** |
| 36  | 5     | **Test storage store without Delete** — Update tests that relied on Delete                                                                    | `storage/*_test.go`                                           | 8         | 2      | 2     | 2      | **2.0** |
| 37  | 5     | **Test fake store without Delete** — Update tests that relied on Delete                                                                       | `testhelpers/*_test.go`                                       | 6         | 2      | 2     | 2      | **2.0** |
| 38  | 5     | **Run full test suite** — `nix run .#test` or per-module tests                                                                                | —                                                             | 10        | 3      | 3     | 2      | **4.5** |
| 39  | 5     | **Run linter** — `nix run .#lint`, fix any issues                                                                                             | —                                                             | 8         | 2      | 2     | 2      | **2.0** |
| 40  | 6     | **Write stream/ README.md** — Usage examples, setup instructions                                                                              | `stream/README.md`                                            | 8         | 2      | 3     | 2      | **3.0** |
| 41  | 6     | **Add stream/ to FEATURES.md** — Update feature inventory                                                                                     | `FEATURES.md`                                                 | 5         | 1      | 1     | 1      | **1.0** |
| 42  | 6     | **Verify module graph** — Check no cycles, all deps correct                                                                                   | —                                                             | 5         | 2      | 2     | 1      | **4.0** |
| 43  | 4     | **SQL schema for AggregateProjection** — CREATE TABLE statement, migration notes                                                              | `stream/projection.go` (docs)                                 | 5         | 2      | 2     | 1      | **4.0** |
| 44  | 5     | **Test AggregateProjection** — Handle upsert, tombstone status update                                                                         | `stream/projection_test.go`                                   | 10        | 3      | 2     | 2      | **3.0** |
| 45  | 6     | **Final git commit + push** — Comprehensive commit message                                                                                    | —                                                             | 5         | 2      | 2     | 1      | **4.0** |

---

## Execution Order (by priority, dependencies respected)

### Phase 0: Preparation (3 tasks, 8 min)

- **#34** Git checkpoint
- **#14** Create stream/ module scaffold
- **#30** Update go.work

### Phase 1: Foundation (4 tasks, 28 min)

- **#1** Create core/event/tombstone.go
- **#2** Add metadata keys + MarkTombstone/MarkRebirth
- **#3** Add DetectTombstone function
- **#22** Test tombstone types

### Phase 2: Decomposition (11 tasks, 85 min)

- **#4** Define Sink interface
- **#5** Define Source interface
- **#6** Update Store = Sink + Source
- **#7** Rename BackwardsLoader → BackwardsSource
- **#8** Rename TransactionalStore → TransactionalSink
- **#13** Add Journal/SeekableJournal assertions
- **#9** Remove Delete from MemoryStore
- **#10** Remove Delete from SQLEventStore
- **#11** Remove Delete from FakeStore
- **#12** Remove Delete from PebbleEventStore
- **#35–37** Update tests for removed Delete

### Phase 3: Stream Core (6 tasks, 53 min)

- **#15** Create stream/types.go
- **#16** Create stream/aggregate_reader.go
- **#17** Create stream/builder.go
- **#18** Create stream/in_memory.go
- **#24** Test stream builder
- **#25** Test InMemoryAggregateReader

### Phase 4: Stream Advanced (5 tasks, 52 min)

- **#19** Create stream/middleware.go
- **#20** Create stream/projection.go
- **#21** Create stream/sql_reader.go
- **#26** Test StatusMiddleware
- **#44** Test AggregateProjection

### Phase 5: Verification (5 tasks, 42 min)

- **#23** Test MarkTombstone + MarkRebirth
- **#38** Run full test suite
- **#39** Run linter
- **#27–29** Update projection/decider/saga for Sink
- **#42** Verify module graph

### Phase 6: Documentation (5 tasks, 33 min)

- **#31** Update example/user
- **#32** Update AGENTS.md
- **#40** Write stream/README.md
- **#33** Update v4 proposal
- **#41** Update FEATURES.md
- **#45** Final git commit + push

---

## Total Estimates

| Metric                   | Value               |
| ------------------------ | ------------------- |
| **Total tasks**          | 45                  |
| **Total estimated time** | ~301 min (~5 hours) |
| **Max task duration**    | 12 min              |
| **Phases**               | 7                   |
| **New files**            | ~15                 |
| **Modified files**       | ~25                 |
| **Tests added**          | ~10 files           |

---

## Risk Mitigation

| Risk                                     | Mitigation                                            |
| ---------------------------------------- | ----------------------------------------------------- |
| Delete removal breaks many tests         | Phase 2 includes dedicated test-update tasks (#35–37) |
| Sink/Source split breaks type assertions | Add var \_ assertions (#13) to catch at compile time  |
| stream/ module has circular deps         | Verify module graph (#42) before declaring done       |
| SQL projection table schema wrong        | Test AggregateProjection (#44) with real SQLite       |
| Backward compat broken                   | Deprecated aliases in core/event/store.go             |

---

## Success Criteria

1. `nix run .#build` passes
2. `nix run .#test` passes (all modules)
3. `nix run .#lint` passes
4. `go test ./stream/...` passes with >80% coverage
5. `event.Store` = `Sink + Source` (verified by interface assertions)
6. No `Delete` method on any Store implementation
7. `stream/` module is independently importable
8. Tombstone middleware auto-marks delete/rebirth events
9. SQLAggregateReader queries projection table efficiently
10. Documentation updated (AGENTS.md, stream/README.md)
