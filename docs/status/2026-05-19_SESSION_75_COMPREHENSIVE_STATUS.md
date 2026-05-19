# Session 75 — Comprehensive Status Report

**Date:** 2026-05-19 | **Session:** 75 (continuation of Session 74)

---

## Executive Summary

Continued execution of the Session 74 audit findings. Fixed 3 correctness bugs and 1 observability gap across 4 modules. Zero new lint issues introduced. All 23 test packages pass.

---

## Work Completed This Session

### Bugs Fixed

| # | Module | Fix | Severity | Commit |
|---|--------|-----|----------|--------|
| 1 | `storage/pebble` | Added optimistic concurrency check in `Save()` — loads existing events, returns `ErrVersionConflict` if count doesn't match `expectedVersion`. Aligns Pebble with MemoryStore and SQLEventStore. | CRITICAL | `26acfa4` |
| 2 | `middleware/retry` | Added `timer.Stop()` after normal `timer.C` fire path. Previously only context cancellation stopped the timer, leaking timer resources. | HIGH | `ad8cd8b` |
| 3 | `core/decider` | `saveSnapshotAfterEvents` now returns early on fold error instead of continuing to encode partial state. The fold error is logged via `opError`. | HIGH | `b1833e2` |

### Safety Improvements

| # | Module | Fix | Commit |
|---|--------|-----|--------|
| 4 | `sync` | `NewLWWResolver` panics with clear message on nil `TimestampFunc` — prevents nil dereference at Resolve time. | `b1833e2` |
| 5 | `core/event` | `OutboxPublisher.publishPending` logs `slog.Warn` instead of silently swallowing errors. Keeps background loop running but provides observability. | `b1833e2` |
| 6 | `catalog/eventcatalog` | Fixed `wsl_v5` and `golines` lint issues. | `b1833e2` |

### Style

| Commit | Description |
|--------|-------------|
| `4dabeb4` | Auto-format catalog files from `nix fmt` |

---

## False Alarms from Session 74 Audit

The previous session identified several "bugs" that turned out to be already guarded:

| Claimed Bug | Reality |
|-------------|---------|
| Aggregate snapshot with nil state when codec is nil | `event.ShouldSnapshot` checks `codec != nil` — `trySnapshot` never reached with nil codec |
| Decider `saveSnapshotAfterEvents` nil-dereferences codec | Same guard via `shouldSnapshot` → `event.ShouldSnapshot` |
| `catalog.SchemaFromType` panics on interface types | `schemaFromReflect(nil)` returns `{Type: "null"}` — not a panic |
| Decider dual `%w` wrapping makes first error unreachable | Go 1.20+ supports multiple `%w` in `fmt.Errorf` — both errors are findable via `errors.Is/As` |

**Root cause:** The audit traced individual functions in isolation without following the call chain through `shouldSnapshot` → `event.ShouldSnapshot` (which checks codec != nil).

---

## Project Health

### Tests

```
23/23 packages PASS
0 failures
43 benchmarks
958 test functions
```

### Lint

```
2 pre-existing issues (staticcheck SA1019 — deprecated CatalogMeta in middleware test)
0 new issues from this session
```

### Code Metrics

| Metric | Value |
|--------|-------|
| Production LOC | 14,713 |
| Test LOC | 28,576 |
| Total Go files | 277 |
| Total LOC | 43,289 |

### Commits This Session

```
4dabeb4 style(catalog): auto-format from nix fmt
b1833e2 fix(decider,sync,event,catalog): correct bugs and improve observability
ad8cd8b fix(middleware): stop timer after normal fire in retry backoff loop
26acfa4 fix(storage): add optimistic concurrency check to Pebble Save
```

---

## Known Issues (Unchanged)

| Issue | Severity | Status |
|-------|----------|--------|
| `testhelpers v1.1.0` incompatible with current `core` | HIGH | Workspace masks it; needs tag re-release |
| `example/todo` broken build | MEDIUM | Stale storage/core API; needs rewrite or removal |
| `core/go.mod` has test deps in production requires | LOW | `memory` + `testhelpers` only used in `_test.go` |
| 2 pre-existing staticcheck deprecation warnings | LOW | In middleware test code |
| Pre-commit hook golangci-lint workspace error | LOW | "directory prefix . does not contain modules listed in go.work" |

---

## Top #25 Next Steps (Prioritized)

### Critical / High

1. **Re-release `testhelpers` tag** — Published v1.1.0 uses `int` for version; core requires `event.Version`. Blocks isolated module builds.
2. **Fix or remove `example/todo`** — 330 lines, stale APIs, external dep. `example/user` already serves demo purpose.
3. **Clean `core/go.mod`** — Move `memory` + `testhelpers` to test-only dependencies.
4. **Add test for decider fold error during snapshot** — Verify the new early-return behavior.
5. **Add test for `NewLWWResolver` nil panic** — Verify nil guard works.
6. **Add test for Pebble concurrency check** — Verify `ErrVersionConflict` is returned.

### Medium

7. **Merge aggregate/decider repository logic** — ~200 lines duplicated (`persistChanges`, `loadFromSnapshot`, `shouldSnapshot`).
8. **Unify error sentinels across aggregate/decider/projection** — `ErrNilStore`, `ErrNilBus` duplicated across packages.
9. **Add `event.GlobalLoader` to `MemoryStore`** — `LoadAll()` method for projection replay.
10. **Collapse core/event helper files** — 26 files → ~20 by combining small helpers.
11. **Shared catalog exporter skeleton** — `WalkMessages` helper deduplicates asyncapi/eventcatalog/d2 traversal.
12. **Deepen storage by inlining SQL helpers** — `sql_helpers.go` is only used by `event_store.go`.
13. **Add integration tests for Pebble store** — Currently only unit tests with serialization.
14. **Add retry integration test for projection** — Verify `WithRetry` option works end-to-end.
15. **Document `TransactionalStore.SaveWithOutbox` contract** — Atomicity guarantees, partial failure behavior.

### Low / Nice-to-have

16. **Remove deprecated `CatalogMeta`** — Replace with catalog module's auto-derive API.
17. **Add `io.Closer` to `MemoryStore`/`MemoryBus`** — Consistency with other stores.
18. **Fix pre-commit golangci-lint workspace error** — Run per-module instead of workspace root.
19. **Add `example/user` README** — Already has architecture diagram, needs usage instructions.
20. **Audit `sync` module test coverage** — New module, may have gaps.
21. **Add vector clock serialization benchmarks** — Critical for sync performance.
22. **Consider absorbing `projection` into `core/event`** — Eliminates "which runner?" confusion.
23. **Add OpenAPI spec for docserver** — Currently only AsyncAPI + EventCatalog.
24. **Evaluate CRDT conflict resolution** — `sync/` has LWW only; CRDT would enable richer merge strategies.
25. **Document sync protocol wire format** — `SyncMessage`/`SyncRequest`/`SyncResponse` need usage docs.

---

## Open Question

**Should `example/todo` be fixed to match the current library API, or should it be moved to its own repository?**

- 330 lines in `main.go`, has external dep `larsartmann/cqrs-htmx`
- Breaks every time core/storage APIs change
- `example/user` already serves the simple demo purpose
- Moving to its own repo would decouple release cycles
