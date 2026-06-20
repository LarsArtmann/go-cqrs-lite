# Session 152 — Outbox Removal: Status & Improvement Plan

**Date:** 2026-05-29 16:39
**Scope:** Remove outbox from go-cqrs-lite, assess remaining cleanup, plan next steps

---

## A) FULLY DONE ✅

| #   | Item                                                                                | Evidence                     |
| --- | ----------------------------------------------------------------------------------- | ---------------------------- |
| 1   | Delete `core/event/outbox.go` (Outbox interface, OutboxID, OutboxEntry)             | File removed                 |
| 2   | Delete `core/event/outbox_publisher.go` + 3 test files                              | Files removed                |
| 3   | Delete `storage/outbox.go`, `outbox_poller.go`, `outbox_helpers.go` + tests         | Files removed                |
| 4   | Delete `storage/transactional_store.go` + tests                                     | File removed                 |
| 5   | Delete `memory/outbox.go` + test                                                    | File removed                 |
| 6   | Delete `testhelpers/fake_outbox.go` + test                                          | File removed                 |
| 7   | Remove `TransactionalSink` interface from `core/event/store.go`                     | Removed                      |
| 8   | Remove `ErrNilOutbox`, `ErrAlreadyStarted`, `ErrPublisherClosed` from errors.go     | Removed                      |
| 9   | Remove `WithOutbox()` from `core/decider/options.go`                                | Removed                      |
| 10  | Simplify `decider.Execute()` — just `Save()` + `PublishChanges()`                   | Done                         |
| 11  | Remove outbox branch in `PublishChanges()` helper                                   | Simplified to direct publish |
| 12  | Update `storage/sql_backend.go` — remove outbox/tx fields                           | Rewritten                    |
| 13  | Update `storage/sql/dialect.go` — remove `OutboxSchema()` from interface + impls    | Removed                      |
| 14  | Update `storage/sql/errors.go` — remove `OutboxStatus` type                         | Removed                      |
| 15  | Update `storage/sql/tables.go` — remove `TableOutbox`                               | Removed                      |
| 16  | Update `storage/sql/helpers.go` — remove `SharedAckBatch`, `OutboxInsertSQL`        | Removed                      |
| 17  | Update `storage/sql/reconstruction.go` — remove `SaveWithOutboxTx`                  | Removed                      |
| 18  | Update `storage/sqlite_helpers.go` — remove OutboxSchema from DDL                   | Removed                      |
| 19  | Update `storage/doc.go` — remove outbox type aliases                                | Updated                      |
| 20  | Update `turso/connector.go` — remove `NewTursoOutbox`, `NewTursoTransactionalStore` | Removed                      |
| 21  | Remove outbox test functions from memory, decider, coverage tests                   | Removed                      |
| 22  | Update `AGENTS.md` — remove outbox from module tree + ISP list                      | Updated                      |
| 23  | Delete `docs/adr/0005-outbox-pattern.md`                                            | Deleted                      |
| 24  | Add deprecation banner to `docs/outbox-explained.html`                              | Added                        |
| 25  | All 29 packages build + pass tests (`nix run .#build`, `nix run .#test`)            | Verified                     |
| 26  | Lint clean (only pre-existing exhaustruct warning)                                  | Verified                     |

---

## B) PARTIALLY DONE ⚠️

| #   | Item                      | What's Left                                                                                    | Impact                   |
| --- | ------------------------- | ---------------------------------------------------------------------------------------------- | ------------------------ |
| 1   | **Stale Go references**   | `storage/event_store_global.go:38` — trace attr `"cqrs.outbox.limit"` (misleading, not broken) | Low — cosmetic           |
| 2   | **Dead Go constant**      | `otel/attributes.go:40` — `AttrOutboxEntryCount` unused after outbox removal                   | Low — dead code          |
| 3   | **Stale Go comment**      | `storage/sql/doc.go:3` — package comment still mentions "outbox"                               | Low — cosmetic           |
| 4   | **Stale Go comment**      | `core/event/bus.go:12` — comment mentions "outbox publishers"                                  | Low — cosmetic           |
| 5   | **STORAGE_GUIDE.md**      | Still contains full outbox section, SaveWithOutbox, NewSQLOutbox examples                      | Medium — misleading docs |
| 6   | **storage/README.md**     | Still contains full outbox section with code examples                                          | Medium — misleading docs |
| 7   | **FEATURES.md**           | 14 outbox-related feature entries still listed                                                 | Medium — misleading docs |
| 8   | **REMOVE-OUTBOX-PLAN.md** | Status still says "DRAFT"                                                                      | Low — update to COMPLETE |
| 9   | **ADR-0006**              | Still lists `TransactionalSink` as interface #7                                                | Low — historical doc     |
| 10  | **ADR-0007**              | Still mentions outbox as future pebble feature                                                 | Low — historical doc     |

---

## C) NOT STARTED ❌

| #   | Item                        | Description                                                                                                                                                                                     | Impact                 |
| --- | --------------------------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ---------------------- |
| 1   | **Journal-based publisher** | No built-in "tail journal → publish" mechanism exists. The old outbox provided this. A `JournalPublisher` that does `ReadFrom(checkpoint) → Publish → SaveCheckpoint` would be the replacement. | High — missing feature |
| 2   | **Example update**          | `example/user/` and other examples may need updating if they referenced outbox                                                                                                                  | Low                    |
| 3   | **Integration test gap**    | Integration tests that tested outbox round-trips may need journal-based replacements                                                                                                            | Medium                 |

---

## D) TOTALLY FUCKED UP 💥

Nothing. The removal was clean. All packages compile and pass. No data loss risk.

**However**, the removal was incomplete in documentation and minor code references (see §B). The plan said to update these but they were missed during execution.

---

## E) WHAT WE SHOULD IMPROVE

### Type Model Improvements

1. **`PublishChanges()` is now trivial** — it's just `publisher.Publish()` with error wrapping. Consider removing it entirely and having decider call `publisher.Publish()` directly. Less indirection, same result.

2. **No crash-recovery publishing path** — The outbox provided a built-in "save events durably, publish later" mechanism. Now the only crash-recovery pattern is the projection's `SeekableJournal + CheckpointStore`. A symmetric `EventPublisher` that tails the journal and publishes to a bus would fill the gap.

3. **`SeekableJournal.ReadFrom()` is underutilized** — This is the core primitive for crash-safe replay. It powers projections but could also power event publishing, command dispatching, and saga orchestration.

4. **Error taxonomy has gaps** — `ErrNilBus` still exists but other `ErrNil*` sentinels were removed. Consider a consistent policy: either validate all constructor params with sentinels, or use `fmt.Errorf` for all.

### Architecture Improvements

5. **`PublishChanges` helper adds no value now** — Before outbox removal, it abstracted "outbox or direct publish". Now it's a thin wrapper. Remove or merge into `decider.Execute()`.

6. **Example quality** — `example/user/` doesn't demonstrate the full stack. A superb example with signing, projections, and journal-based publishing would be the best marketing this library has.

7. **No structured event publishing** — Events are published via `Publisher.Publish()` which is fire-and-forget. There's no built-in concept of "publish with confirmation" or "publish with retry". The outbox provided this implicitly.

### Library Ecosystem

8. **Watermill Forwarder could be the external broker publisher** — For consumers who need broker-backed publishing with crash recovery, Watermill's Forwarder is the right tool. Our `watermill/` adapters already bridge both worlds. We should document this as the recommended path for external broker use cases.

9. **`go-errors-family` underutilized** — The error taxonomy (Rejection/Conflict/Transient/Infrastructure/Corruption) is powerful but inconsistently used. Some paths still use `fmt.Errorf` where classified errors would be better.

10. **OTel attribute cleanup** — Dead constants like `AttrOutboxEntryCount` should be removed. The `cqrs.outbox.limit` attribute in `event_store_global.go` should be renamed.

---

## F) Top 25 Things to Do Next (Sorted by Impact/Effort)

### Tier 1: Quick Wins (High Impact, Low Effort)

| #   | Task                                                             | Effort | Impact                      |
| --- | ---------------------------------------------------------------- | ------ | --------------------------- |
| 1   | Fix 4 stale Go references (comments, trace attrs, dead const)    | 10 min | High — code hygiene         |
| 2   | Update STORAGE_GUIDE.md — remove outbox section, update examples | 15 min | High — consumer-facing docs |
| 3   | Update storage/README.md — remove outbox section                 | 10 min | High — consumer-facing docs |
| 4   | Update FEATURES.md — remove 14 outbox entries                    | 10 min | Medium — feature accuracy   |
| 5   | Update REMOVE-OUTBOX-PLAN.md — mark COMPLETE                     | 2 min  | Low — closure               |
| 6   | Delete `docs/planning/OUTBOX_TRANSACTION_API.md`                 | 1 min  | Low — cleanup               |

### Tier 2: Medium Effort, High Impact

| #   | Task                                                                    | Effort | Impact                           |
| --- | ----------------------------------------------------------------------- | ------ | -------------------------------- |
| 7   | Remove `PublishChanges` helper — inline into decider                    | 15 min | Medium — simplify                |
| 8   | Add deprecation note to ADR-0006 and ADR-0007 about outbox removal      | 5 min  | Medium — historical accuracy     |
| 9   | Document "JournalPublisher" pattern in STORAGE_GUIDE or new guide       | 30 min | High — shows crash-recovery path |
| 10  | Add example of `SeekableJournal + CheckpointStore` for event publishing | 20 min | High — practical guidance        |

### Tier 3: Feature Work (Higher Effort, Strategic Value)

| #   | Task                                                                             | Effort | Impact                                      |
| --- | -------------------------------------------------------------------------------- | ------ | ------------------------------------------- |
| 11  | Implement `JournalPublisher` — tails journal, publishes to bus, saves checkpoint | 2h     | High — replaces outbox for broker consumers |
| 12  | Update `example/user/` to demonstrate full stack                                 | 2h     | High — marketing                            |
| 13  | Add Watermill Forwarder integration docs for external broker use                 | 1h     | High — production guidance                  |
| 14  | Consistent error classification across all modules                               | 3h     | Medium — debuggability                      |
| 15  | Dead code audit — find all unused exports post-removal                           | 1h     | Medium — code hygiene                       |

### Tier 4: Strategic Improvements (Larger Effort)

| #   | Task                                                       | Effort | Impact                   |
| --- | ---------------------------------------------------------- | ------ | ------------------------ |
| 16  | `core/` dissolution (per PROPOSAL-dissolve-core-v2)        | 8h     | High — modularity        |
| 17  | BDD tests for Version, SchemaVersion, Pagination types     | 1h     | Medium — coverage        |
| 18  | Code duplication audit (dedup)                             | 2h     | Medium — maintainability |
| 19  | Pebble store: add CheckpointStore + SnapshotStore          | 4h     | Medium — completeness    |
| 20  | Full code review of every file                             | 4h     | Medium — quality         |
| 21  | Naming review across all modules                           | 2h     | Medium — clarity         |
| 22  | Add saga pattern example (projection + command dispatch)   | 2h     | Medium — documentation   |
| 23  | Catalog: AsyncAPI 3.0 support                              | 4h     | Low — spec compliance    |
| 24  | OTel: add metrics for journal read, publish latency        | 2h     | Low — observability      |
| 25  | Performance benchmarks for hot paths (Save, Load, Publish) | 2h     | Low — perf visibility    |

---

## G) Top #1 Question I Cannot Figure Out Myself

**Should we build `JournalPublisher` (a general-purpose "tail journal → publish to bus" component), or is the Watermill Forwarder the intended replacement?**

Arguments for JournalPublisher:

- Self-contained, no external dependency
- Works with in-process `MemoryBus` (Watermill Forwarder needs a SQL Pub/Sub)
- Symmetric to `projection.Runner` (same pattern: journal + checkpoint + handler)
- Library consumers get a "just works" solution

Arguments for Watermill Forwarder:

- Battle-tested, handles dedup/retry/poison queue
- Works with any broker (NATS, Kafka, RabbitMQ)
- Our `watermill/` adapters already bridge both worlds
- Avoids building Yet Another Poller™

The question is whether this is a library concern (provide the primitive) or a consumer concern (compose the primitives yourself). Given go-cqrs-lite's "library, not framework" philosophy, the answer should be "provide the primitive, let consumers compose." But `projection.Runner` already violates this by being a composed solution...

**My recommendation:** Build a minimal `JournalPublisher` that does `ReadFrom(checkpoint) → Publish → SaveCheckpoint` with configurable interval and batch size. Same pattern as `projection.Runner` but publishing instead of handling. Document Watermill Forwarder as the alternative for external broker use cases.

---

## Build Status

```
✅ nix run .#build — all 29 modules
✅ nix run .#test  — all 29 packages green
✅ nix run .#lint  — clean (pre-existing exhaustruct only)
```

## Files Changed This Session

- **Deleted:** 17 files (outbox interfaces, implementations, tests, ADR)
- **Edited:** 24 files (storage/sql, sql_backend, decider, turso, doc.go, test files, AGENTS.md)
- **Created:** 1 file (docs/planning/REMOVE-OUTBOX-PLAN.md)
- **Remaining cleanup:** 4 Go files, 3 doc files, 1 plan file
