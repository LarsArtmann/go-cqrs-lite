# Session 78 — Execution Plan + Implementation

**Date:** 2026-05-20 03:35 · **Duration:** ~45 min
**Status:** Core plan executed. Consumer API improvements shipped.

---

## What Was Done

### Phase 1: Critical Bug Fixes ✅

| # | Task | Status | Detail |
|---|------|--------|--------|
| 1 | Retry timer leak | ⏭️ Skipped | Already handled — timer.Stop() on both paths (line 109, 114) |
| 2 | Nil codec snapshot | ✅ Fixed | `trySnapshot` now returns early when codec is nil |
| 3 | Pebble concurrency | ⏭️ Skipped | Already implemented at lines 59-75 |
| 4 | Dual %w wrapping | ⏭️ Skipped | Go 1.20+ supports multiple %w — not a bug |
| 5 | LWW nil guard | ⏭️ Skipped | Already exists at conflict.go:42 |
| 6 | SchemaFromType nil | ⏭️ Skipped | Already handled at schema_reflect.go:11 |

**Verdict:** 4 of 6 TODO items were stale (already fixed in earlier sessions). Only 1 real bug found and fixed.

### Phase 2: Consumer API Improvements ✅

| # | Task | Status | Files |
|---|------|--------|-------|
| 7 | `command.TypedHandler[T]` + `RegisterTyped[T]` | ✅ Shipped | `core/command/typed.go` |
| 8 | TypedHandler tests | ✅ Shipped | `core/command/typed_test.go` (4 tests) |
| 9 | `query.TypedHandler[T]` | ⏭️ Already exists | `core/query/query.go:52` |
| 10 | `event.NewEvents` batch helper | ✅ Shipped | `core/event/codec_batch.go` |
| 11 | `event.DecodePayloads` batch helper | ✅ Shipped | `core/event/codec_batch.go` |
| 12 | `event.MustNewEvents` | ✅ Shipped | `core/event/codec_batch.go` |
| 13 | Batch codec tests | ✅ Shipped | `core/event/codec_batch_test.go` (7 tests) |
| 14 | `event.NewTypedProjection[T]` | ✅ Shipped | `core/event/projection.go` |
| 15 | Clock injection | ⏭️ Already exists | `event.WithOccurredAt(time.Time)` option |

### Phase 3: Observability ✅

| # | Task | Status | Detail |
|---|------|--------|--------|
| 16 | Pebble corrupt ID warnings | ✅ Shipped | `storage/pebble_serialization.go` — slog.Warn on parse errors |
| 17 | OutboxPublisher logging | ⏭️ Already exists | `core/event/outbox_publisher.go:224` |
| 18 | Duplicate projection check | ✅ Shipped | `projection/runner.go` + `ErrDuplicateProjection` sentinel |

### Phase 6: Documentation ✅

| # | Task | Status | Detail |
|---|------|--------|--------|
| 19 | Getting-started README | ✅ Shipped | "Your First CQRS App (5 minutes)" section |
| 20 | Updated Commands section | ✅ Shipped | Shows RegisterTyped pattern, no type assertions |

### Phase 5: Architecture Cleanup ✅

| # | Task | Status | Detail |
|---|------|--------|--------|
| 21 | catalog/internal/schemautil | ✅ Shipped | Pre-commit hook extracted shared schema helpers |

---

## Not Done (Deferred)

| Phase | Tasks | Reason |
|-------|-------|--------|
| Phase 4 (Storage) | DDL on Dialect, testhelpers bump, go.mod replaces | Lower impact than consumer API |
| Phase 5 (Architecture) | WalkMessages, unified sentinels | Internal quality, not customer-facing |
| Phase 7 (Release) | Tag all modules | Should happen after consumer API validation with SEC |
| Phase 8 (Examples) | Rewrite example/user | Blocked until Phase 7 tagging |

---

## Key Metrics

| Metric | Before | After |
|--------|--------|-------|
| Test packages | 23/23 pass | 23/23 pass |
| Lint issues | 0 | 0 |
| Consumer API helpers | query.TypedHandler only | + command.TypedHandler, NewEvents, DecodePayloads, NewTypedProjection |
| README getting-started | None | "Your First CQRS App" 5-minute section |
| Stale TODO items found | — | 4 of 6 Phase 1 items, 2 of 5 Phase 3 items |

---

## Commits (9)

```
03c640d refactor(catalog): deduplicate schemaToAny into internal/schemautil
b507809 docs: add "Your First CQRS App" getting-started section
e5df120 feat: add TypedProjection, duplicate projection check, Pebble warnings
7672a1e feat(event): add NewTypedProjection helper + formatting fixes
4b1fe49 feat(event): add NewEvents, MustNewEvents, DecodePayloads batch helpers
44e0395 feat(command): add TypedHandler[T] and RegisterTyped[T]
3399607 fix(aggregate): skip snapshot save when codec is nil
6ee623f docs(planning): add Session 78 execution plan
4b24c30 docs(status): Session 77 status report
```

---

## Impact on SEC (Real Consumer)

The new APIs eliminate SEC's top friction points:

| SEC Friction | New API | LOC Saved |
|---|---|---|
| `assertCmd[T]` workaround (3 sites) | `command.RegisterTyped[T]` | ~15 lines |
| Manual encode loop in `persistAndPublish` (~30 lines) | `event.NewEvents` | ~25 lines |
| Manual decode loop in `foldEvents` (~20 lines) | `event.DecodePayloads` | ~15 lines |
| Type assertion in projection handlers | `event.NewTypedProjection[T]` | ~5 lines/handler |
| **Total estimated reduction** | | **~60 lines** |

---

## Top #1 Question

**Should we tag releases now or wait for SEC to validate the new APIs first?**

The new `TypedHandler[T]` and `NewEvents` are additive (non-breaking). Tagging `core v1.4.0` now would let SEC start using them immediately. But tagging prematurely means we can't change the signatures if SEC finds issues. Recommendation: let SEC try the new APIs first, then tag.
