# Session 99 — Comprehensive Status Report

**Date:** 2026-05-25 19:47
**Branch:** master (clean, up to date with origin)
**Since last report:** Sessions 95–96 (naming overhaul, bug fixes, dependency cleanup)

---

## a) FULLY DONE ✅

### Session 95 (Naming Overhaul)
| Change | Detail |
|--------|--------|
| `event.Core` → `event.ImmutableEvent` | Concrete Event struct renamed across all modules |
| `command.Core` → `command.BasicCommand` | Concrete Command struct renamed |
| `query.Core` → `query.BasicQuery` | Concrete Query struct renamed |
| `CatalogEntry` → `HandlerMeta` | In `core/pkg/dispatcher/` |
| `RegisterCatalogEntry` → `RegisterHandlerMeta` | Consistent naming |
| `NewCheckpointStore` → `NewMemoryCheckpointStore` | Constructor naming consistency |
| `NewWithDialect` public constructors | All 4 storage types (EventStore, SnapshotStore, Outbox, CheckpointStore) |
| Go version alignment | 1.26.2 → 1.26.3 across all 9 go.mod files |
| Broken file copies removed | `memory/outbox_publisher.go`, `memory/outbox_publisher_test.go`, `projection/inmemory_runner.go` |
| `validateEventParams` extracted | `event.go` 273→222 lines (under 250 limit) |
| Command/Query decoupled from event | Import `go-error-family` directly instead of `event.NewRejection` |
| `InMemoryRunner` deprecated | Stays in `event/` with `// Deprecated:` notice |
| Example struct literals fixed | `Core:` → `BasicCommand:` / `BasicQuery:` |
| Golden test snapshots refreshed | All catalog tests |
| Custom `Logger` interface removed | `middleware.Logger` + `SlogAdapter` → `*slog.Logger` |

### Session 96 (Bug Fixes & Quality)
| Change | Detail |
|--------|--------|
| **Dispatch() closed-state bug fixed** | `command.Dispatch()` and `query.Dispatch()` now pre-check closed state, returning domain sentinels (`command.ErrDispatcherClosed` / `query.ErrDispatcherClosed`) consistently via `errors.Is()` |
| **cqrs-htmx removed** | Broken external dependency (called removed `event.RegisterClassification`), replaced with inline `chainMiddleware` |
| `loadFromSnapshot` moved | From `options.go` → `load.go` (file organization) |
| `TestMetrics` → `FakeMetrics` | Consistent with FakeBus, FakeStore, FakeOutbox, FakeSnapshotStore |
| `codec_typed.go` → `event_new.go` | File contains `event.New()` constructor, not codec logic |
| FakeStore override setters complete | Added `AppendBatchFn`, `LoadToVersionFn`, `LoadToTimestampFn` |

---

## b) PARTIALLY DONE 🔶

| Item | Status | Remaining |
|------|--------|-----------|
| **testhelpers event constructors** | 4 overlapping constructors exist (`NewEvent`, `MakeEvent`, `QuickEvent`, `QuickEventOpts`) | Could simplify to 2–3 |
| **FakeBus/FakeOutbox thread safety** | `Published`/`Entries` exported without lock protection | Need accessor methods or documentation |
| **`catalog/adapters/builder.go` deprecated** | 4 deprecated symbols still exist (`CatalogBuilder`, `AddCommand`, `AddEvent`, `AddQuery`) | No consumers; safe to delete |

---

## c) NOT STARTED ⬜

| # | Item | Impact | Effort |
|---|------|--------|--------|
| 1 | Delete deprecated `core/aggregate` package entirely | MED | LOW |
| 2 | Delete deprecated `catalog/adapters/builder.go` | LOW | LOW |
| 3 | Add test coverage for new Dispatch() closed-state path | MED | LOW |
| 4 | Unify `query.Handler` type alias → named type | MED | MED |
| 5 | Extract `requireType[T]` helper from example/todo | LOW | LOW |
| 6 | Extract `countingHandler` into testhelpers | LOW | LOW |
| 7 | Split large test files (>250 lines: schema_test.go 605, store_test.go 512, outbox_test.go 415) | LOW | MED |
| 8 | `go-error-family` replacement with stdlib | HIGH | HIGH |
| 9 | `go-branded-id` replacement with stdlib | HIGH | HIGH |
| 10 | Module split: storage into sub-packages (SQL, Pebble, Turso) | MED | HIGH |
| 11 | Add `event.Result` type to decider `Execute` for publish failure visibility | MED | MED |
| 12 | Add `NotFound` error family to `go-error-family` | LOW | MED |
| 13 | Document `opError` wrapping pattern for `errors.Is` traceability | LOW | LOW |
| 14 | Add `LoadToVersion`/`LoadToTimestamp` override to FakeBus subscriber tests | LOW | LOW |
| 15 | Rename `slog_test.go` → `logging_slog_test.go` (test is for logging, not a separate slog adapter) | LOW | LOW |

---

## d) TOTALLY FUCKED UP 💥

| Item | Detail |
|------|--------|
| **None currently** | All tests pass. Both examples build. `go vet` clean. No production files over 250 lines. Working tree clean. |

### Known Pre-existing Issues (not from this session)

| Issue | Severity | Detail |
|-------|----------|--------|
| `MemoryBus.Publish` holds RLock during handler execution | LOW | Acceptable for test utility |
| `query.Handler` returns `any` | LOW | `DispatchTyped[T]` is the documented workaround |
| `Root.LoadEvents` vs `Core.LoadFromHistory` mismatch | LOW | Only affects deprecated `aggregate` package |
| gopls stale cache | LOW | Shows phantom errors for deleted `slog.go` and old `Core` field names — files don't exist on disk |

---

## e) WHAT WE SHOULD IMPROVE

### Architecture
1. **Error family taxonomy gap**: No `NotFound` family in `go-error-family`. `ErrAggregateNotFound` is a `Rejection`, which is semantically misleading. A `NotFound` family would improve clarity.
2. **Dual serialization paths**: `event.New()` (in `event_new.go`) hardcodes `json.Marshal` while the `Codec` interface is used for snapshots. If a consumer provides a custom codec (e.g., protobuf), payloads are still JSON-encoded.
3. **Silent snapshot degradation**: Setting `WithSnapshotStore` without `WithCodec` silently disables snapshots — no error, no warning.
4. **`decider.Execute` double-fold**: `saveSnapshotAfterEvents` re-folds events that were already folded during `Execute`. For large event batches, this doubles fold work.

### Code Quality
5. **`query.Handler` is a type alias (`=`)**, not a named type — loses godoc documentation and compiler distinction.
6. **Exported fields without mutex protection**: `FakeBus.Published`, `FakeOutbox.Entries`, `FakeMetrics.Records` are all exported but accessed without holding the lock.
7. **testhelpers inconsistency**: FakeBus uses direct field (`PublishErr`), FakeSnapshotStore uses `Set*` methods, FakeStore uses fluent `*Fn` setters. Should pick one pattern.

### Dependencies
8. **`go-error-family` and `go-branded-id` are personal packages** with no community validation. Could be replaced with stdlib patterns (Go 1.13+ error wrapping, typed strings for IDs).
9. **`storage/` pulls in 32+ transitive deps** from Pebble — consumers who only want SQLite still pay for Pebble.

---

## f) TOP #25 THINGS TO DO NEXT (sorted by impact × ease)

| # | What | Impact | Effort | ROI |
|---|------|--------|--------|-----|
| 1 | Delete deprecated `core/aggregate` package | MED | LOW | ⭐⭐⭐⭐ |
| 2 | Delete deprecated `catalog/adapters/builder.go` functions | LOW | LOW | ⭐⭐⭐ |
| 3 | Add tests for new Dispatch() closed-state path | MED | LOW | ⭐⭐⭐⭐ |
| 4 | Rename `slog_test.go` → `logging_slog_test.go` | LOW | LOW | ⭐⭐⭐ |
| 5 | Extract `requireType[T]` helper from example/todo | LOW | LOW | ⭐⭐⭐ |
| 6 | Document `opError` wrapping pattern | LOW | LOW | ⭐⭐⭐ |
| 7 | Unify `query.Handler` type alias → named type | MED | MED | ⭐⭐⭐ |
| 8 | Add `NotFound` error family to go-error-family | LOW | MED | ⭐⭐ |
| 9 | Validate snapshot+codec pair in NewRepository | MED | MED | ⭐⭐⭐ |
| 10 | Simplify testhelpers event constructors (4→2) | LOW | MED | ⭐⭐ |
| 11 | Add accessor methods to FakeBus/FakeOutbox | LOW | MED | ⭐⭐ |
| 12 | Standardize testhelpers override pattern (pick one) | LOW | MED | ⭐⭐ |
| 13 | Extract `countingHandler` into testhelpers | LOW | MED | ⭐⭐ |
| 14 | Split `schema_test.go` (605 lines) into focused files | LOW | MED | ⭐⭐ |
| 15 | Split `store_test.go` (512 lines) into focused files | LOW | MED | ⭐⭐ |
| 16 | Split `outbox_test.go` (415 lines) into focused files | LOW | MED | ⭐⭐ |
| 17 | Add `event.Result` type to decider Execute for publish failure | MED | MED | ⭐⭐⭐ |
| 18 | Unify `event.New()` serialization with Codec interface | MED | HIGH | ⭐⭐ |
| 19 | Replace `go-error-family` with stdlib patterns | HIGH | HIGH | ⭐⭐ |
| 20 | Replace `go-branded-id` with stdlib patterns | HIGH | HIGH | ⭐⭐ |
| 21 | Split `storage/` into sub-packages (SQL, Pebble, Turso) | MED | HIGH | ⭐ |
| 22 | Add `LoadFn`/`LoadToVersionFn` override consistency to FakeStore (DONE in S7) | — | — | ✅ |
| 23 | Eliminate double-fold in `saveSnapshotAfterEvents` | MED | MED | ⭐⭐ |
| 24 | Add `IsAggregateNew()` boolean to decider `loadByEvents` return | MED | LOW | ⭐⭐⭐ |
| 25 | CI: add `go vet` and race detector to GitHub Actions | HIGH | LOW | ⭐⭐⭐⭐ |

---

## g) TOP #1 QUESTION

**Should we keep `go-error-family` and `go-branded-id` as dependencies, or invest in replacing them with stdlib patterns?**

Arguments for keeping:
- They provide structured error classification (`Rejection`, `Conflict`, `Transient`, `Corruption`, `Infrastructure`) — not easy to replicate with stdlib
- `go-branded-id` provides type-safe IDs with zero boilerplate — Go doesn't have this natively

Arguments for replacing:
- Both are personal packages with no community validation or adoption
- `go-error-family` could be replaced with custom error types + `errors.As()` — ~50 lines of code
- `go-branded-id` could be replaced with `type ID[T any] string` + constructors — ~30 lines of code
- Removing them eliminates 2 external dependencies that consumers must trust

This is a **strategic decision** that affects the library's positioning: minimal-dependency SDK vs. opinionated ecosystem.

---

## Test Coverage (Current)

| Package | Coverage | Trend |
|---------|----------|-------|
| `core/pkg/dispatcher` | 100.0% | — |
| `core/pkg/id` | 100.0% | — |
| `middleware` | 100.0% | — |
| `catalog/adapters` | 100.0% | — |
| `catalog/internal/caseutil` | 100.0% | — |
| `memory` | 99.6% | — |
| `core/query` | 98.4% | ↓ from 100% (new Dispatch() code path) |
| `core/aggregate` | 95.9% | — |
| `catalog` | 96.8% | — |
| `catalog/d2` | 95.0% | — |
| `catalog/openapi` | 94.4% | — |
| `projection` | 94.4% | — |
| `core/event` | 93.8% | — |
| `catalog/asyncapi` | 93.7% | — |
| `core/decider` | 93.6% | — |
| `core/command` | 92.3% | ↓ from 94.6% (new Dispatch() code path) |
| `testhelpers` | 91.3% | ↑ from 80.3% (new setters) |
| `catalog/eventcatalog` | 91.3% | — |
| `catalog/docserver` | 90.1% | — |
| `storage` | 88.7% | — |
| `catalog/internal/schemautil` | 84.2% | — |

---

## Commit History (Sessions 95–96)

```
0db44f9 docs: update AGENTS.md with Session 96 changes and coverage
26e4c79 testhelpers: add missing FakeStore override setters
0ca5ca2 core/event: rename codec_typed.go → event_new.go
fc6a8a5 testhelpers: rename TestMetrics → FakeMetrics
5785cdb core/decider: move loadFromSnapshot and shouldSnapshot to load.go
31788b5 example/todo: remove broken cqrs-htmx dependency
28b4f2f fix: pre-check closed state in command/query Dispatch()
398064e docs: add Logger interface removal to session 95 history
639d7f4 middleware: replace custom Logger interface with *slog.Logger
dd3b66f example: fix struct literal field names after Core rename
fac52a3 docs: update AGENTS.md with Session 95 naming overhaul changes
35f94c4 core/event: deprecate InMemoryRunner in favor of projection.Runner
5b607dd core/command, core/query: decouple error constructors from event package
132f405 core/event: extract validateEventParams to reduce event.go below 250 lines
0f8137a fix: remove broken file copies and refresh golden tests
```

---

## Build & CI Status

- ✅ `go vet` — clean (0 issues)
- ✅ `go test` — all 26 packages pass
- ✅ `example/todo` — builds independently
- ✅ `example/user` — builds independently
- ✅ No production files over 250 lines
- ✅ Working tree clean, pushed to origin/master
- ⚠️ gopls shows 5 phantom errors (stale cache for deleted/renamed files — not real)
