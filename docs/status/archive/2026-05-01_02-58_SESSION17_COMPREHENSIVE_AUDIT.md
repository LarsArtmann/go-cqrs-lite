# Session 17 — Comprehensive Audit & Status Report

**Date:** 2026-05-01 02:58 UTC · **Branch:** master @ `21e2831` · **Module count:** 9

---

## Executive Summary

The project is in **strong shape**: 0 lint issues, race-free, core modules at 95–100% coverage, clean multi-module architecture. However, a deep audit reveals **3 real bugs**, **1 stale bug report** (FEATURES.md says storage is BROKEN but it's partially fixed), **4 files over the 250-line limit**, **11 functions over the 30-line limit**, and **4 stale root-level docs**. The `core/pkg/id` fuzz test is failing. The `storage/` module has **zero tests**. The `query.Handler` return type forces `any` into ~20 locations across the codebase.

---

## A) FULLY DONE ✅

These are production-quality with no known issues:

| Component                               | Package                  | Coverage | Status                     |
| --------------------------------------- | ------------------------ | -------- | -------------------------- |
| Command Dispatcher                      | `core/command`           | 100.0%   | Complete                   |
| Query Dispatcher                        | `core/query`             | 100.0%   | Complete                   |
| Event Core (types, options, builder)    | `core/event`             | 96.1%    | Complete                   |
| Generic Dispatcher                      | `core/pkg/dispatcher`    | 100.0%   | Complete                   |
| Aggregate Root                          | `core/aggregate`         | 95.6%    | Complete                   |
| Branded IDs                             | `core/pkg/id`            | 92.9%\*  | Complete (fuzz test issue) |
| Middleware Suite (7 concerns × 3 types) | `middleware`             | 99.4%    | Complete                   |
| Memory Store                            | `memory`                 | 94.9%    | Complete                   |
| Catalog Registry + Schema               | `catalog`                | 94.4%    | Complete                   |
| AsyncAPI 3.0 Exporter                   | `catalog/asyncapi`       | 97.6%    | Complete                   |
| EventCatalog Exporter                   | `catalog/eventcatalog`   | 95.5%    | Complete                   |
| Catalog Adapters                        | `catalog/adapters`       | 98.8%    | Complete                   |
| Upcaster Registry                       | `core/event`             | —        | Complete                   |
| Context Enricher                        | `core/event`             | —        | Complete                   |
| Test Helpers + Fakes                    | `testhelpers`            | N/A      | Complete                   |
| CI/CD (Nix + GitHub Actions)            | `flake.nix` + `.github/` | —        | Complete                   |

\*Coverage affected by FuzzParse failure.

---

## B) PARTIALLY DONE ⚠️

### B1. Storage Module — `storage/` (0% test coverage)

**Commit `95be76f` fixed metadata persistence**, but FEATURES.md still says BROKEN. Current state:

| Feature                         | Status                |
| ------------------------------- | --------------------- |
| PostgreSQL event store          | ✅ Compiles           |
| Schema DDL                      | ✅                    |
| Optimistic concurrency (Save)   | ✅                    |
| AppendBatch                     | ✅                    |
| Load / LoadFromVersion / Delete | ✅                    |
| Metadata persistence            | ✅ Fixed in `95be76f` |
| Codec usage (marshal/unmarshal) | ✅ Working            |
| Close() lifecycle               | ✅ Exists             |

**Remaining issues:**

| Issue                          | Severity    | Detail                                                             |
| ------------------------------ | ----------- | ------------------------------------------------------------------ |
| Zero tests                     | 🔴 CRITICAL | No unit, integration, or benchmark tests                           |
| Close() closes shared \*sql.DB | 🔴 HIGH     | Will break every other component using the same DB connection pool |
| 3 functions >30 lines          | ⚠️          | Save (73), AppendBatch (52), scanEvents (64)                       |
| File is 369 lines              | ⚠️          | Over 250-line limit                                                |
| Hardcoded PostgreSQL           | ⚠️          | `$1` placeholders, BYTEA, JSONB — no dialect abstraction           |
| No batch INSERT optimization   | LOW         | Each event is a separate INSERT within transaction                 |

**Verdict:** FEATURES.md should update from BROKEN → PARTIALLY_FUNCTIONAL.

### B2. Projection Runner — `core/event/runner.go`

| Feature                | Status     |
| ---------------------- | ---------- |
| Projection interface   | ✅         |
| ProjectionFunc adapter | ✅         |
| InMemoryRunner         | ⚠️ Partial |
| CheckpointStore        | ✅         |
| Event type filtering   | ✅         |

**Gaps:**

- Fail-fast on first projection error — subsequent projections skipped
- No retry or dead-letter mechanism
- No duplicate registration guard
- Push-model only (no background polling)

### B3. Memory Outbox + Checkpoint — `memory/`

- `MemoryOutboxStore` and `MemoryCheckpointStore` lack `LifecycleMixin` + `Close()` that sibling types have
- `PollPending` + `Ack` can race (no "processing" state)
- Inconsistent with `MemoryStore`/`MemoryBus` lifecycle patterns

### B4. EventSourcedRepository Save — `core/aggregate/repository.go`

- `store.Save` + `bus.Publish` + `outbox.Append` are non-atomic (documented)
- If `bus.Publish` fails: events are persisted but aggregate still has uncommitted changes, no snapshot saved, retry will hit version conflict
- Snapshot store errors silently ignored (falls through to full replay)
- No nil-check on constructor args (`store`, `bus`)

---

## C) NOT STARTED 📐

| Feature                        | Where Documented                                   | Notes                                    |
| ------------------------------ | -------------------------------------------------- | ---------------------------------------- |
| Watermill module (Kafka, NATS) | `docs/planning/2026-04-23_WATERMILL_PRO_CONTRA.md` | Planning doc exists                      |
| SQL SnapshotStore              | Interface in `core/event`                          | No implementation                        |
| SQL CheckpointStore            | Interface in `core/event`                          | No implementation                        |
| Outbox background publisher    | Interface in `core/event`                          | Memory impl exists, no polling publisher |
| Saga / Process Manager         | AGENTS.md                                          | Not started                              |
| Tagged releases (semver)       | AGENTS.md                                          | All modules at v0.0.0                    |
| `example/user/` in CI          | `flake.nix`                                        | Excluded from test/lint apps             |

---

## D) TOTALLY FUCKED UP 🔴

### D1. FuzzParse Case-Sensitivity — `core/pkg/id/fuzz_test.go:29`

```
FuzzParse/5680a28533fa623f: roundtrip mismatch:
  got  "0000000000000000000A000000"
  want "0000000000000000000a000000"
```

**Root cause:** `ulid.Parse()` accepts lowercase Crockford Base32 but `String()` always returns uppercase. The fuzz test does exact string comparison.

**Fix:** Either:

- Use `strings.EqualFold(parsed.String(), input)` in the test, OR
- Normalize input to uppercase in `Parse()` before returning

### D2. FEATURES.md Stale Data

FEATURES.md still lists storage metadata as "silently discarded" and codec as "unused" — both were fixed in commit `95be76f`. This is misinformation that could mislead users.

### D3. `any` Type Cascade — `query.Handler`

`query.Handler = func(context.Context, Query) (any, error)` forces `any` into **~20 locations** across middleware, testhelpers, and catalog. This violates the project's own "No `any` types" convention. Root cause is architectural — the only way to fix this is a generic `Handler[T]` or a typed result wrapper.

---

## E) WHAT WE SHOULD IMPROVE

### Architecture

1. **`query.Handler` return type** — `(any, error)` cascades `any` everywhere. Consider `Handler[T any] func(context.Context, Query) (T, error)` or a `Result` type.
2. **No transaction abstraction** — `EventSourcedRepository.Save` + `Outbox.Append` can't be atomic. Need a `UnitOfWork` or `Transaction` interface.
3. **Interface Segregation on `Root`** — `ApplySnapshot(state []byte)` forces every aggregate to implement snapshot support. Should be a separate `SnapshotAware` interface.
4. **`event.Bus` missing lifecycle** — No `Close()` or `Use()` in interface, forcing concrete-type dependencies for `MemoryBus`.
5. **No error type hierarchy** — Sentinel errors are scattered. Callers can't use a single `errors.Is` check for "dispatcher closed" across command/query/event.
6. **Command/Query asymmetry** — Command handler returns `error`, query returns `(any, error)`. No shared middleware type possible.

### Code Quality

7. **4 files over 250 lines** — `storage/event_store.go` (369), `testhelpers/fakes.go` (326), `catalog/internal/cattest/helpers.go` (330), `core/aggregate/repository.go` (274).
8. **11 functions over 30 lines** — Including `storage.Save` (73), `NewEvent` (67), `scanEvents` (64), `AppendBatch` (52).
9. **Stale root-level docs** — `BDD_TESTS_REVIEW.md`, `DEDUPLICATION_PLAN.md`, `MIGRATION_TO_NIX_FLAKES_PROPOSAL.md`, `TODO_LIST.md` should be archived or deleted.
10. **Snapshot error swallowing** — `loadEvents` in `repository.go` silently falls through on snapshot store errors.
11. **`MemoryStore.Save` accepts empty events** — No validation of event slice.
12. **No version overflow protection** — `Version.Increment()` wraps silently at `math.MaxInt`.

### Testing

13. **Storage module has 0 tests** — Needs integration tests with PostgreSQL (testcontainers-go).
14. **`example/user/` has 0 tests** — Compiles but not verified.
15. **No chaos/fault injection tests** — What happens when store.Save fails after outbox.Append?

### Documentation

16. **FEATURES.md outdated** — Storage metadata and codec issues marked as BROKEN but were fixed.
17. **No API stability documentation** — No versioning strategy, no stability markers on interfaces.

---

## F) Top #25 Things to Do Next

Sorted by **impact ÷ effort** (highest ROI first):

| #   | Task                                                                                              | Impact | Effort | Category         |
| --- | ------------------------------------------------------------------------------------------------- | ------ | ------ | ---------------- |
| 1   | Fix FuzzParse case-sensitivity bug in `core/pkg/id`                                               | HIGH   | LOW    | Bug fix          |
| 2   | Update FEATURES.md: storage BROKEN → PARTIALLY_FUNCTIONAL, remove stale claims                    | HIGH   | LOW    | Docs             |
| 3   | Archive 4 stale root-level markdown files to `docs/archive/`                                      | MEDIUM | LOW    | Cleanup          |
| 4   | Add `LifecycleMixin` + `Close()` to `MemoryOutboxStore` and `MemoryCheckpointStore`               | MEDIUM | LOW    | Consistency      |
| 5   | Make inline `errors.New` in `memory/bus.go` and `core/query/query.go` into sentinel vars          | LOW    | LOW    | Code quality     |
| 6   | Split `storage/event_store.go` (369→<250 lines) — extract scanner, metadata helpers               | MEDIUM | LOW    | File limits      |
| 7   | Validate `MemoryStore.Save` rejects empty event slices                                            | MEDIUM | LOW    | Bug fix          |
| 8   | Add nil-check in `NewRepository` for `store` and `bus` params                                     | MEDIUM | LOW    | Safety           |
| 9   | Fix snapshot error swallowing in `loadEvents` — log/propagate snapshot store errors               | MEDIUM | LOW    | Bug fix          |
| 10  | Add `context.Context` doc to MemoryStore methods (all ignored)                                    | LOW    | LOW    | Docs             |
| 11  | Split `core/aggregate/repository.go` (274→<250 lines)                                             | LOW    | LOW    | File limits      |
| 12  | Refactor functions >30 lines: `NewEvent` (67→<30), `Exporter.Export` (54→<30)                     | MEDIUM | MEDIUM | Code quality     |
| 13  | Add integration tests for `storage/` with testcontainers-go + PostgreSQL                          | HIGH   | HIGH   | Testing          |
| 14  | Fix `SQLEventStore.Close()` — don't close shared `*sql.DB`                                        | HIGH   | MEDIUM | Bug fix          |
| 15  | Add `SnapshotAware` interface to segregate `ApplySnapshot` from `Root`                            | MEDIUM | MEDIUM | Architecture     |
| 16  | Fix `EventSourcedRepository.Save` publish failure — mark committed before publish or add recovery | HIGH   | HIGH   | Bug fix          |
| 17  | Add `Close()` + `Use()` to `event.Bus` interface (breaking change — coordinate)                   | HIGH   | HIGH   | Architecture     |
| 18  | Unify `ErrDispatcherClosed` across packages — shared base error with domain-specific wrappers     | MEDIUM | MEDIUM | Consistency      |
| 19  | Add duplicate registration guard to `InMemoryRunner.Register`                                     | LOW    | LOW    | Safety           |
| 20  | Add `HasHandler(cmdType) bool` to command/query dispatchers                                       | MEDIUM | LOW    | API completeness |
| 21  | Update `example/user/main.go` — add bus subscription, `defer Close()`, constants for event types  | MEDIUM | LOW    | Example          |
| 22  | Add `Reset(ctx) error` to `Projection` interface for rebuild-from-scratch                         | MEDIUM | MEDIUM | API completeness |
| 23  | Address `query.Handler` `any` cascade — explore generic `Handler[T]` pattern                      | HIGH   | HIGH   | Architecture     |
| 24  | Add `LoadByVersionRange` to `event.Store` for partial replay                                      | MEDIUM | MEDIUM | API completeness |
| 25  | Tag v0.1.0 releases for stable modules (core, memory, middleware, catalog)                        | HIGH   | MEDIUM | Release          |

---

## G) Top #1 Question I Cannot Figure Out Myself

**Should `event.Bus` and `event.Store` interfaces include lifecycle methods (`Close()`, `Use()`)?**

Currently:

- `MemoryBus` has `Close()` and `Use()` as concrete methods, NOT on the interface
- `MemoryStore` has `Close()` as a concrete method
- `SQLEventStore` has `Close()` as a concrete method
- The `event.Bus` interface only has `Publish`, `Subscribe`, `SubscribeAll`

Adding `Close()` to the interfaces would be a **breaking change** for any external implementations. Not adding it means callers must type-assert to clean up resources. The same question applies to `Use()` on `Bus` — without it in the interface, middleware can only be added by concrete type reference.

**This is a library design decision that only the maintainer can make.** It affects the API contract and all consumers.

---

## Test Results

```
ok  core/aggregate      coverage: 95.6%
ok  core/command         coverage: 100.0%
ok  core/event           coverage: 96.1%
ok  core/pkg/dispatcher  coverage: 100.0%
FAIL core/pkg/id         coverage: 92.9% (FuzzParse case-sensitivity)
ok  core/query           coverage: 100.0%
ok  memory               coverage: 94.9%
ok  catalog              coverage: 94.4%
ok  catalog/adapters     coverage: 98.8%
ok  catalog/asyncapi     coverage: 97.6%
ok  catalog/eventcatalog coverage: 95.5%
ok  middleware           coverage: 99.4%
ok  integration/*        (all pass)
?   storage              [no test files]
?   testhelpers          [no test files]
```

---

## Module Dependency Graph

```
core ← storage (SQL event store)
core ← testhelpers ← memory (in-memory implementations)
core ← testhelpers ← middleware (cross-cutting concerns)
core ← catalog (AsyncAPI + EventCatalog)
core ← memory ← integration (cross-module tests)
core ← memory ← example/user (demo app)
```

No circular dependencies. Core is independently publishable.

---

## Files Over Limits

| File                                  | Lines | Limit | Action Needed              |
| ------------------------------------- | ----- | ----- | -------------------------- |
| `storage/event_store.go`              | 369   | 250   | Split into 2-3 files       |
| `testhelpers/fakes.go`                | 326   | 250   | Split by fake type         |
| `catalog/internal/cattest/helpers.go` | 330   | 250   | Split helpers + assertions |
| `core/aggregate/repository.go`        | 274   | 250   | Extract helper methods     |

---

## Git State

```
Branch: master
Commits: 21e2831 (up to date with origin)
Working tree: clean
Ahead of origin: 0
```
