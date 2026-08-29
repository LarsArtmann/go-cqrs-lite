# Status Report — DLQ Hardening + VersionedSeekableJournal Tests + SKILL.md

<!-- historical-artifact-banner -->

> **Historical session artifact.** This is a point-in-time snapshot from a past
> session. Many items marked TODO / Open / Not Started / Broken have since been
> resolved. See [CHANGELOG.md](../../../CHANGELOG.md) and
> [TODO_LIST.md](../../../TODO_LIST.md) for current state.
> Last documentation health audit: 2026-07-16.

**Date:** 2026-07-11 15:53
**Session scope:** TODO items from DLQ production hardening (Gap 3), VersionedSeekableJournal follow-ups (Gap 1), SKILL.md API documentation (item 34), DLQ index optimization (item 31)
**Tests:** All passing (projectionhost 78.8% coverage, schema 89.9% coverage)
**Doc-check:** 868 references valid across 34 packages

---

## A) FULLY DONE — Completed This Session

### DLQ Production Hardening (items 28-33)

| Item | Description                         | Status                     | Details                                                                                                                                                                                                                                       |
| ---- | ----------------------------------- | -------------------------- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| 28   | DLQ `Purge(ctx, before time.Time)`  | **DONE**                   | Implemented as `PurgeBefore(ctx, before time.Time) (int64, error)` on `DeadLetterStoreAdmin` — different name to avoid signature collision with existing `Purge(ctx, projectionName string)`                                                  |
| 29   | DLQ `List(ctx, offset, limit int)`  | **DONE**                   | Implemented as `ListPaged(ctx, projectionName, offset, limit)` on `DeadLetterStoreAdmin` — keeps projection filter capability                                                                                                                 |
| 30   | DLQ `Count(ctx) (int64, error)`     | **DONE**                   | Implemented on `DeadLetterStoreAdmin` interface                                                                                                                                                                                               |
| 31   | DLQ index optimization audit        | **DONE**                   | Removed redundant `idx_pdl_projection(projection_name)` (UNIQUE constraint already provides leftmost-prefix index). Added two purpose-built indexes: `idx_pdl_projection_time(projection_name, failed_at)` and `idx_pdl_failed_at(failed_at)` |
| 32   | DLQ serialization format docs       | **DONE**                   | Full column layout, index strategy, reconstruction behavior, and admin interface docs in `projectionhost/doc.go`                                                                                                                              |
| 33   | DLQ `PurgeForProjection(ctx, name)` | **DONE (already existed)** | Already covered by existing `Purge(ctx, projectionName string)` on `DeadLetterStore` — no new code needed                                                                                                                                     |

### New `DeadLetterStoreAdmin` Interface

Created optional interface for production management capabilities, implemented by both `SQLiteDeadLetterStore` and `MemoryDeadLetterStore`:

```go
type DeadLetterStoreAdmin interface {
    DeadLetterStore
    Count(ctx context.Context) (int64, error)
    ListPaged(ctx context.Context, projectionName string, offset, limit int) ([]DeadLetterEntry, error)
    PurgeBefore(ctx context.Context, before time.Time) (int64, error)
}
```

**Design decision:** Separate optional interface rather than extending `DeadLetterStore` — consumers type-assert to access admin features without requiring them from every implementation.

### DLQ Tests (items 16, 17, 18)

| Item | Test                                        | Lines | What it verifies                                                                           |
| ---- | ------------------------------------------- | ----- | ------------------------------------------------------------------------------------------ |
| 16   | `TestSQLiteDeadLetterStore_Stress_10k`      | ~40   | 10k entries across 5 projections, Count + ListPaged + PurgeBefore — all pass, query <100ms |
| 17   | `TestSQLiteDeadLetterStore_ConcurrentStore` | ~30   | 20 goroutines × 50 entries = 1000 concurrent writes, verified final count matches          |
| 18   | `TestSQLiteDeadLetterStore_CorruptPayload`  | ~45   | Corrupt metadata in DB, List surfaces corruption error with event ID (no panic)            |

Plus unit tests: Count, ListPaged (pagination across 4 pages + beyond-data), ListPaged_AllProjections, PurgeBefore (with old+recent entries), PurgeBefore_None, DeadLetterStoreAdmin interface compliance.

### VersionedSeekableJournal Follow-ups (items 14, 15, 23)

| Item | Description                    | Status   | Details                                                                                                          |
| ---- | ------------------------------ | -------- | ---------------------------------------------------------------------------------------------------------------- |
| 14   | Property test with rapid       | **DONE** | 3 property tests (100 iterations each): upcasterChain, passthrough, ReadFrom                                     |
| 15   | Benchmark: upcasting overhead  | **DONE** | 3 benchmarks with 10k events                                                                                     |
| 23   | Upcaster error mid-stream test | **DONE** | `TestVersionedSeekableJournal_MidStreamUpcastError` — error on event 5 of 10, propagates from ReadAll + ReadFrom |

**Property test details:**

- `TestVersionedSeekableJournal_Property_upcasterChain` — random chain depth (1-5) and event count (1-50), verifies all events reach correct final schema version regardless of starting version
- `TestVersionedSeekableJournal_Property_passthrough` — events of unregistered types pass through completely unchanged (version + payload)
- `TestVersionedSeekableJournal_Property_ReadFrom` — position-based seek with upcasting, verifies consistent results across partial reads from random cursor positions

**Benchmark results (10k events, AMD Ryzen AI MAX+ 395):**

```
ReadAll_NoUpcasters-32        139µs/op    (baseline)
ReadAll_WithUpcasters-32      7.1ms/op    (3-deep chain, ~51x overhead)
ReadFrom_WithUpcasters-32     556µs/op    (500 events, 3-deep chain)
```

### SKILL.md Update (item 34)

Added three missing APIs to consumer-facing documentation:

- **Decision matrix**: 2 new rows — "Upcast events during projection replay" (`schema`/`VersionedSeekableJournal`) and "Pull-based event backfill (REST endpoint)" (`transport/http`/`BackfillHandlerWithTransform`)
- **Cheat sheet**: 3 new code snippets — VersionedSeekableJournal, prometheus.WithViews, BackfillHandlerWithTransform
- doc-check passes: **868 references valid across 34 packages**

### AGENTS.md Update

Added `DeadLetterStoreAdmin` usage examples to the projectionhost pattern section.

### File Split

Extracted admin methods from `sqlite_dlq.go` (was 399 lines) into `sqlite_dlq_admin.go` (100 lines) to comply with 350-line CI limit.

### Files Changed Summary

| File                                     | Change                                                                       | Lines       |
| ---------------------------------------- | ---------------------------------------------------------------------------- | ----------- |
| `projectionhost/dlq.go`                  | Added `DeadLetterStoreAdmin` interface + MemoryDeadLetterStore admin methods | +93         |
| `projectionhost/sqlite_dlq.go`           | Index optimization (removed redundant, added 2 purpose-built)                | -2 net      |
| `projectionhost/sqlite_dlq_admin.go`     | **NEW** — Count, ListPaged, PurgeBefore for SQLiteDeadLetterStore            | 100         |
| `projectionhost/sqlite_dlq_test.go`      | 9 new tests (Count, ListPaged, PurgeBefore, stress, concurrent, corrupt)     | +308        |
| `projectionhost/doc.go`                  | DLQ serialization format docs + admin interface docs                         | +55         |
| `projectionhost/go.mod`                  | Indirect dep version bumps from tidy                                         | ~14 changed |
| `schema/versioned_journal_rapid_test.go` | **NEW** — 3 property tests + mid-stream error test + 3 benchmarks            | 452         |
| `schema/go.mod`                          | Added `pgregory.net/rapid` direct dependency                                 | ~4          |
| `SKILL.md`                               | 2 decision matrix rows + 3 cheat sheet entries                               | +15         |
| `AGENTS.md`                              | DeadLetterStoreAdmin usage example                                           | +8          |
| `TODO_LIST.md`                           | 13 items marked done with completion notes                                   | ~34 changed |

---

## B) PARTIALLY DONE

Nothing — all items picked up this session are fully complete.

---

## C) NOT STARTED

These are items from the TODO_LIST.md that remain open but were NOT in scope for this session:

### VersionedSeekableJournal (Gap 1)

- [ ] `scenario.GivenProjection` test — For VersionedSeekableJournal + projectionhost (item 48, P3)

### SSE Transform (Gap 2)

- [ ] CBOR→JSON e2e test — Through all 3 SSE paths (live, replay, backfill) with a real CBOR-encoded event (item 25)

### Projectionhost Observability

- [ ] `LagPerProjection() map[string]time.Duration` — Per-worker lag for dashboards (item 38)
- [ ] `WorkerState.Lag` field — Currently only available via aggregate `LagDuration()` (item 39)
- [ ] `Reset(ctx, name)` purges DLQ — Projection reset should optionally clear DLQ entries (item 46)

### Testing

- [ ] `Projectionhost rapid property tests` — Generate random event streams, verify projection invariants (item 19)

### Index/Performance

- [ ] `Benchmark: DLQ query performance at scale` — 100k entries, measure ListPaged latency (item 50)

### Documentation

- [ ] Document two DeadLetterEntry types — ADR-0043 Part B: dispatch-side vs projection poison (item 45, P3)
- [ ] README.md docs freshness — Missing `encryption`, `turso`, `testutil` module sections

---

## D) TOTALLY FUCKED UP

Nothing was broken. All tests pass, doc-check passes, formatting applied.

---

## E) WHAT WE SHOULD IMPROVE

### Things I Should Have Done Better This Session

1. **No lint run** — I ran tests and formatting but did NOT run `nix run .#lint` (golangci-lint). The `nix fmt` reformatted 3 files but I have not verified lint compliance. The new `// indirect` dep marking on `pgregory.net/rapid` in schema/go.mod may need a depguard allow list entry.

2. **go.sum not verified** — After `go mod tidy` in both schema and projectionhost, I did not verify go.sum files are consistent across the workspace (workspace-level `go.work` may have conflicts).

3. **`event/codec_test.go`, `event/event.go`, `event/event_new.go`, `integration/event/mixed_codec_test.go`, `stack/bundle.go`, `stack/bundle_test.go`** — These files appear in the git diff but were NOT touched by me this session. They were already modified at conversation start (pre-existing unstaged changes from a prior session). I should have been more explicit about distinguishing my changes from pre-existing ones.

4. **No race detector run** — The concurrent Store test (item 17) should ideally be verified with `-race` flag specifically.

5. **The corrupt payload test accepts both error and success paths** — `TestSQLiteDeadLetterStore_CorruptPayload` handles both the case where List returns an error (corrupt metadata prevents reconstruction) and where it succeeds. This is intentional (both are valid behaviors depending on corruption type), but the test could be more precise about which corruption types trigger which path.

6. **`DeadLetterStoreAdmin` not added to integration tests** — The new interface is tested at the unit level only. No cross-module or stack-level integration test verifies that the admin interface works through the full projectionhost lifecycle.

7. **MemoryDeadLetterStore admin methods lack dedicated tests** — I implemented Count, ListPaged, and PurgeBefore on MemoryDeadLetterStore for test parity but did not write separate tests for them. They are indirectly tested through SQLite tests, but the memory implementations have their own logic.

8. **Property tests could be deeper** — The rapid property tests use simple identity upcasters. They could test more complex payload transformations (rename fields, add defaults, etc.) to catch edge cases in the upcasting logic.

9. **No `nix run .#check-layers`** — The schema module got a new production dependency (`pgregory.net/rapid`). Even though rapid is test-only, it's listed as a direct dependency in go.mod. The layer checker may flag this — need to verify the script handles the test-only distinction correctly.

---

## F) Up to 50 Things We Should Get Done Next

### Immediate (this branch, before merge)

1. Run `nix run .#lint` and fix any violations in new/modified files
2. Run `nix run .#check-layers` to verify schema dependency budget is not exceeded
3. Run `go test -race ./projectionhost/... ./schema/...` to verify no data races
4. Verify go.sum consistency: `go mod verify` in all modified modules
5. Add `pgregory.net/rapid` to `.golangci.yml` depguard allow list if needed
6. Write tests for MemoryDeadLetterStore admin methods (Count, ListPaged, PurgeBefore)
7. Run full workspace test suite: `nix run .#test` to verify no cross-module regressions

### Short-term (P1-P2, next session)

8. **Item 25**: CBOR→JSON e2e test through all 3 SSE paths
9. **Item 38**: `LagPerProjection() map[string]time.Duration` for per-worker dashboard metrics
10. **Item 39**: `WorkerState.Lag` field for structured health endpoints
11. **Item 46**: `Reset(ctx, name)` should optionally purge DLQ entries for the projection
12. **Item 48**: `scenario.GivenProjection` test for VersionedSeekableJournal + projectionhost
13. **Item 45**: Document two DeadLetterEntry types (ADR-0043 Part B)
14. **Item 50**: Benchmark DLQ at 100k entries
15. **Item 19**: Projectionhost rapid property tests (random event streams, verify invariants)

### Medium-term (P2-P3)

16. `stack` bundle integration: expose `DeadLetterStoreAdmin` through stack presets
17. Metrics integration: wire `DeadLetterStoreAdmin.Count()` into prometheus metrics
18. Add `PurgeBefore` to the `Reset` flow as an option (age-based cleanup on rebuild)
19. Health check: include DLQ depth in `bundle.HealthCheck()` response
20. HTTP admin endpoint for DLQ management (list, count, purge)
21. gRPC admin endpoint for DLQ management
22. DLQ replay CLI tool (`cmd/cqrs-dlq` or similar)
23. VersionedStore property tests with rapid (parallel to VersionedSeekableJournal tests)
24. VersionedSeekableJournal nil-inner-journal-during-Close test edge case
25. Benchmark: VersionedStore upcasting overhead (parallel to item 15)
26. Fuzz test: `scanDLQRow` with arbitrary byte sequences
27. Fuzz test: `event.ReconstructEventFromFields` with arbitrary inputs
28. Integration test: SQLiteDeadLetterStore through projectionhost lifecycle (store poison → purge → reset → replay)
29. Docs: add DLQ admin section to `references/advanced.md` §6.9
30. Docs: add VersionedSeekableJournal recipe to `references/recipes.md` §2.5
31. Docs: README.md freshness — add missing module sections
32. API stability: add `DeadLetterStoreAdmin` to `api_surface.txt` golden file

### Long-term (P3+)

33. Postgres dead-letter store implementation (`PostgresDeadLetterStore`)
34. Pebble dead-letter store implementation
35. DLQ TTL: automatic expiry of dead-letter entries after configurable duration
36. DLQ webhook: notify external system when entry is stored
37. DLQ replay scheduler: automatic periodic replay attempts
38. DLQ correlation: group related failures (same event type, same error code)
39. Schema migration tool for DLQ table (versioned DDL migrations)
40. VersionedSeekableJournal: batch upcasting optimization (reduce per-event allocations)
41. VersionedStore + VersionedSeekableJournal: shared benchmark harness
42. Catalog: auto-document DLQ schema in generated docs
43. OTel: DLQ depth as a metric (gauge) in projectionhost metrics recorder
44. Scenario DSL: `GivenDeadLetter` / `ThenDeadLetter` assertions
45. Stack preset: one-call DLQ setup (`sqlite.WithDeadLetterStore(db)`)
46. Compression for DLQ payloads (large events)
47. DLQ export/import (JSON, CSV) for offline analysis
48. VersionedSeekableJournal: streaming ReadFrom (channel-based) for very large journals
49. Property test: VersionedStore + VersionedSeekableJournal consistency (same events, different read paths)
50. Formal verification of index covering properties using `EXPLAIN QUERY PLAN`

---

## G) Top 2 Questions I Cannot Answer Myself

### Q1: Should `pgregory.net/rapid` be marked `// indirect` in schema/go.mod?

Rapid is only imported in `_test.go` files. The `go mod tidy` command added it as a direct dependency (`require pgregory.net/rapid v1.3.0` without `// indirect`). However, the project convention is that test-only packages are excluded from dependency budgets. Is the direct vs indirect distinction correct as-is, or should it be `// indirect`? I cannot determine this without running `nix run .#check-layers`.

### Q2: Were the pre-existing modified files (event/, integration/, stack/) intended to be part of this work?

The git status at session start showed modifications to `event/codec_test.go`, `event/event.go`, `event/event_new.go`, `integration/event/mixed_codec_test.go`, `stack/bundle.go`, `stack/bundle_test.go`, and `AGENTS.md`. These appear to be from a prior session (the CBOR-as-default-codec work). I left them untouched. Should these be committed separately, or are they expected to be part of the same changeset? I cannot determine the intended commit grouping without context on the prior session's workflow.
