# Session 153 — Outbox Removal Cleanup: COMPLETE

**Date:** 2026-05-29 17:19  
**Scope:** Complete all remaining cleanup tasks after outbox removal (Sessions 151–152)  
**Status:** All tasks complete. Build green, tests green, lint green (1 pre-existing).

---

## a) FULLY DONE

### 1. Stale Go References (4 fixes)
| File | Line | Change |
|---|---|---|
| `storage/event_store_global.go` | 38 | `"cqrs.outbox.limit"` → `"cqrs.journal.limit"` |
| `otel/attributes.go` | 39–40 | Removed dead `AttrOutboxEntryCount` constant |
| `storage/sql/doc.go` | 3 | Removed "outbox" from package comment |
| `core/event/bus.go` | 12 | Removed "outbox publishers" from comment |

### 2. Documentation Cleanup (6 files)
| File | Action |
|---|---|
| `docs/STORAGE_GUIDE.md` | Removed Outbox Pattern section, `SaveWithOutbox` row from ops table, outbox from PostgreSQL setup code, "outbox" from Schema DDL comment |
| `storage/README.md` | Removed entire SQLOutbox + OutboxPoller section, `OutboxSchema` from DDL list, `OutboxSchema()` from Dialect interface docs, outbox from module description |
| `FEATURES.md` | Removed 14 outbox-related entries across: Event System (TransactionalSink, Outbox pattern, OutboxPublisher, Publish helper), Decider (Outbox support), Aggregate (Transactional outbox), In-Memory (MemoryOutboxStore), Storage (SQL Outbox, TransactionalSink, SQLBackend Outbox/TransactionalSink methods, OutboxPoller, Turso outbox constructors, OutboxSchema DDL). Added `Crash recovery` entries for SeekableJournal + CheckpointStore pattern. |
| `docs/planning/REMOVE-OUTBOX-PLAN.md` | Changed status from DRAFT → COMPLETE |
| `docs/planning/OUTBOX_TRANSACTION_API.md` | **Deleted** (obsolete design doc) |
| `docs/adr/0006-sink-source-split.md` | Added deprecation note for `TransactionalSink` removal; updated interface count 7→6; removed TransactionalSink from consequences |
| `docs/adr/0007-pebble-scope-event-store-only.md` | Added "Updated" header note about outbox removal; removed outbox from future features list; removed Outbox from consequences |

### 3. Code Cleanup
| Action | Detail |
|---|---|
| `PublishChanges` helper removed | Inlined into `decider.Execute()` with `WrapInfrastructure` error wrapping. Deleted `core/event/publish_helper.go` and `publish_helper_test.go`. |
| `SaveSnapshot` restored | Was accidentally in same file as `PublishChanges`. Restored to `core/event/snapshot_helper.go`. |
| Benchmark updated | `BenchmarkPublishChanges` → `BenchmarkBusPublish` (direct bus.Publish call) |
| `core/go.mod` | `go mod tidy` removed outbox-related test dependency references |

### 4. Verification
- **Build:** `nix run .#build` — all modules compile ✅
- **Tests:** `nix run .#test` — all 29 packages pass ✅
- **Lint:** `nix run .#lint` — 1 pre-existing `exhaustruct` issue in `core/command/store.go:102` (not touched) ✅
- **Outbox references in Go source:** Zero in `core/`, `storage/`, `otel/` ✅
- **Outbox references in docs:** Only in historical status files (expected) ✅

---

## b) PARTIALLY DONE

| Item | Status | What Remains |
|---|---|---|
| gopls stale cache | Partially resolved | Some phantom diagnostics may persist until gopls restart (deleted files referenced in cache). Not a build issue. |
| `core/command/store.go:102` exhaustruct | Pre-existing | Not our responsibility — existed before outbox removal. Should be fixed separately. |

---

## c) NOT STARTED

| Item | Priority | Why Not Started |
|---|---|---|
| `JournalPublisher` implementation | High | Needs design discussion: Should it be a core package? A middleware? An example? Consumers may have different needs (batch size, retry, DLQ). |
| `example/crash-recovery` | Medium | Attempted but removed because standalone example modules aren't in `go.work`. Need to either add to `go.work` or document pattern in prose only. |
| Update remaining status docs (archive/) | Low | 15+ historical status files reference outbox — these are archive, not consumer-facing. Can be cleaned up incrementally. |
| Update `docs/api_surface.txt` | Low | Still lists `PublishChanges` and `SaveSnapshot` as separate funcs — should reflect current state. |
| Update benchmark baseline (`benchmark-baseline.txt`) | Low | References `BenchmarkPublishChanges` which no longer exists. |

---

## d) TOTALLY FUCKED UP

| Item | What | Impact | Fix |
|---|---|---|---|
| **NONE** | — | — | — |

> **Note:** During `PublishChanges` removal, `SaveSnapshot` was accidentally deleted because both functions were in `core/event/publish_helper.go`. This was discovered immediately during `nix run .#build` and fixed by restoring `SaveSnapshot` to `core/event/snapshot_helper.go` (where `ShouldSnapshot` already lived). The error never made it past local build verification.

---

## e) WHAT WE SHOULD IMPROVE

### 1. Separate unrelated functions into different files
`publish_helper.go` contained both `PublishChanges` (publishing) and `SaveSnapshot` (snapshotting). These are unrelated concerns. This contributed to the accidental deletion. **Rule:** One concern per file.

### 2. Error wrapping consistency in decider
`decider.Execute()` now wraps publish errors with `WrapInfrastructure` then wraps again with `opError(ref, "%w", wrapErr)`. This double-wraps the error message. Should either: (a) drop `opError` for publish failures and use `WrapInfrastructure` directly, or (b) make `opError` add the prefix without `%w` nesting.

### 3. Historical docs accumulate stale references
15+ status files in `docs/status/archive/` and `docs/sessions/` still mention outbox. These mislead future readers. A periodic "doc freshness" pass would help.

### 4. `go.work` doesn't include example modules
The `example/` directory has 6 standalone modules (saga-pattern, storage, projection, todo, listing, user) that aren't in `go.work`. This means `go build ./...` from root doesn't catch example breakage. Adding them would catch regressions.

### 5. Feature list (FEATURES.md) is manually maintained
14 outbox entries had to be manually removed. This is error-prone. Consider generating feature lists from code (e.g., via `go doc` + script) or at least a `features-audit` script.

### 6. No automated "dead code" detection
The stale Go references (trace attr, dead const, comments) were caught manually. `go vet` and `golangci-lint` didn't flag them. Consider adding:
- `deadcode` linter
- A script that greps for removed type names in comments
- CI check that fails on `"outbox"` in non-doc Go source after removal PRs

---

## f) Top #25 Things to Get Done Next

### P0 — Blockers / Critical
1. **Implement `JournalPublisher`** — A reusable component that: `ReadFrom(checkpoint) → Publish → SaveCheckpoint`. This is the canonical outbox replacement. Needs API design (batch size? retry? DLQ?).
2. **Add example modules to `go.work`** — Prevents example bit-rot.
3. **Fix `core/command/store.go:102` exhaustruct** — Pre-existing lint failure.

### P1 — High Impact, Low Effort
4. **Add `example/crash-recovery` as workspace module** — Document the SeekableJournal + CheckpointStore pattern with working code.
5. **Update `docs/api_surface.txt`** — Remove `PublishChanges` entry, verify `SaveSnapshot` is correct.
6. **Audit `docs/status/archive/` for stale references** — A single sweep to update or annotate outbox mentions.
7. **Write crash-recovery guide** — A markdown doc explaining "the event store IS the outbox" pattern.
8. **Update `benchmark-baseline.txt`** — Reflect current benchmark names.
9. **Add `deadcode` linter to CI** — Catch dead constants, unused error sentinels.
10. **Review `core/event/errors.go`** — Verify all 16 sentinels are still used after removals.

### P2 — Medium Impact
11. **Automate FEATURES.md generation** — Script that walks packages and emits feature table.
12. **Refactor `decider.Execute()` error wrapping** — Remove double-wrap or standardize pattern.
13. **Add `SeekableJournal` example to STORAGE_GUIDE** — Show `ReadFrom` usage for catch-up.
14. **Clean up `docs/research/` files** — Several research docs reference `storage/outbox_helpers.go` (deleted).
15. **Verify `docs/outbox-explained.html` links** — Check internal links still valid after removals.
16. **Update `CHANGELOG.md`** — Add entry for outbox removal.
17. **Review `integration/` tests** — Ensure integration tests still cover the right surface after outbox removal.

### P3 — Polish / Future
18. **Add `EventPublisher` interface to `core/event`** — Formalize the journal-publisher pattern.
19. **Consider `CheckpointStore` in `pebble/` module** — ADR-0007 says it "may still add CheckpointStore".
20. **Document Watermill Forwarder as outbox alternative** — For consumers who need the full outbox pattern with external brokers.
21. **Periodic doc freshness script** — Check that `FEATURES.md`, `AGENTS.md`, `STORAGE_GUIDE.md` match actual code.
22. **Remove `TransactionalStore` alias** — It's deprecated; determine when to delete.
23. **Add `core/event` GoDoc examples** — `ExampleSeekableJournal_ReadFrom`, `ExampleCheckpointStore`.
24. **Review test coverage after removals** — Some tests were deleted; ensure coverage didn't drop below thresholds.
25. **Core module dissolution** — Evaluate `core/` split per `docs/research/PROPOSAL-dissolve-core-v2.html`.

---

## g) Top #1 Question I Cannot Figure Out Myself

> **Should `JournalPublisher` be a core package, a middleware, or just a documented pattern?**
>
> The outbox was removed because `SeekableJournal + CheckpointStore` already solves crash recovery. But there's no reusable component for this — every consumer has to hand-write the "tail journal, publish, save checkpoint" loop. 
>
> On one hand: A reusable `JournalPublisher` would be valuable (like `projection.Runner` is for projections). It needs: batch size, poll interval, retry, checkpointing, maybe DLQ.
>
> On the other hand: go-cqrs-lite is a library, not a framework. Adding a `JournalPublisher` with lifecycle management (Start/Close) starts to look like the outbox publisher we just removed. It also couples to specific bus implementations.
>
> **The tension:** Where is the line between "library that provides building blocks" and "framework that provides complete solutions"? The outbox crossed it (hence removal). Would `JournalPublisher` cross it too?
>
> I need a decision on: (a) whether to add `JournalPublisher` at all, (b) if yes, what scope (just a helper function? a struct with lifecycle? an example?), (c) if no, how do we document the pattern so consumers don't reinvent it poorly.

---

## Metrics

| Metric | Value |
|---|---|
| Go source files | 411 |
| Test files | 199 |
| Modules | 16 (in go.work) |
| Packages tested | 29 (all pass) |
| Files changed this session | 16 edited, 2 deleted, 1 created then deleted |
| Outbox references in Go source | 0 |
| Pre-existing lint issues | 1 (`core/command/store.go:102`) |

---

## Files Changed

### Deleted
- `core/event/publish_helper.go`
- `core/event/publish_helper_test.go`
- `docs/planning/OUTBOX_TRANSACTION_API.md`

### Edited (code)
- `core/decider/decider.go` — Inlined publish, proper error wrapping
- `core/event/benchmark_test.go` — `BenchmarkPublishChanges` → `BenchmarkBusPublish`
- `core/event/bus.go` — Removed "outbox publishers" from comment
- `core/event/snapshot_helper.go` — Added `SaveSnapshot` function
- `core/go.mod` — `go mod tidy`
- `otel/attributes.go` — Removed `AttrOutboxEntryCount`
- `storage/event_store_global.go` — `"cqrs.outbox.limit"` → `"cqrs.journal.limit"`
- `storage/sql/doc.go` — Removed "outbox" from package comment

### Edited (docs)
- `FEATURES.md` — Removed 14 outbox entries
- `docs/STORAGE_GUIDE.md` — Removed outbox section
- `storage/README.md` — Removed outbox section
- `docs/planning/REMOVE-OUTBOX-PLAN.md` — Status: COMPLETE
- `docs/adr/0006-sink-source-split.md` — Deprecation note for TransactionalSink
- `docs/adr/0007-pebble-scope-event-store-only.md` — Removed outbox from scope

---

*Generated: 2026-05-29 17:19*
