# Session 46 — Comprehensive Status Report

**Date:** 2026-05-03 07:05
**Branch:** `master`
**Test suites:** 21 packages, ALL PASS
**Total LOC:** 31,509 Go (10,067 production + 21,442 test)
**Total coverage:** 90.9%
**Lint:** 46 pre-existing issues (0 new)
**go vet:** clean
**TODOs/FIXMEs:** 0

---

## A) FULLY DONE ✓

### All Modules (10)

| Module                 | Coverage | Test LOC | Status |
| ---------------------- | -------- | -------- | ------ |
| `core/command`         | 100.0%   | —        | ✅     |
| `core/query`           | 100.0%   | —        | ✅     |
| `core/pkg/dispatcher`  | 100.0%   | —        | ✅     |
| `core/pkg/id`          | 100.0%   | —        | ✅     |
| `middleware`           | 100.0%   | —        | ✅     |
| `core/event`           | 98.0%    | —        | ✅     |
| `catalog/d2`           | 97.6%    | —        | ✅     |
| `core/decider`         | 96.2%    | —        | ✅     |
| `catalog/asyncapi`     | 95.9%    | —        | ✅     |
| `catalog/eventcatalog` | 95.6%    | —        | ✅     |
| `catalog/adapters`     | 95.5%    | —        | ✅     |
| `catalog`              | 94.4%    | —        | ✅     |
| `core/aggregate`       | 93.2%    | —        | ✅     |
| `storage`              | 92.0%    | —        | ✅     |
| `memory`               | 91.9%    | —        | ✅     |
| `projection`           | 89.7%    | —        | ✅     |

### BDD Test Suites (34 specs across 3 modules)

| Module         | Specs | Commit    |
| -------------- | ----- | --------- |
| `core/decider` | 13    | `9e65222` |
| `projection`   | 11    | `ba7f52d` |
| `memory`       | 10    | `ab92ed2` |

### Architecture

- Error taxonomy (5 families) + extensible `RegisterClassification`
- `Publisher`/`Subscriber` sub-interfaces on `event.Bus` (ISP)
- Branded IDs (`id.Of[T]`) with 8 branded types
- `Decider[State]` pure-function aggregate pattern
- Snapshot support (aggregate + decider) with codec + strategy
- Outbox pattern (memory + SQL) with panic recovery
- Catalog system (AsyncAPI, D2, EventCatalog exporters)
- Projection runner with replay, live subscription, retry, injected logger
- Multi-module monorepo with 10 modules via `go.work`

---

## B) PARTIALLY DONE ⚠️

### Duplications Found (DEEP AUDIT — Session 46)

| Duplication                                                         | Locations                                                                  | Severity | Fixable?                                                      |
| ------------------------------------------------------------------- | -------------------------------------------------------------------------- | -------- | ------------------------------------------------------------- |
| **`SnapshotStrategy` interface + `EveryNEvents` + `everyN` struct** | `core/aggregate/options.go:10-31` ↔ `core/decider/options.go:13-31`        | HIGH     | ✅ Extract to `core/event/snapshot.go` or `core/pkg/snapshot` |
| **`publishChanges` method**                                         | `core/aggregate/repository.go:104-123` ↔ `core/decider/decider.go:218-237` | MEDIUM   | ⚠️ Signature differs slightly (method receivers)              |
| **`shouldSnapshot` method**                                         | `core/aggregate/repository.go:214-219` ↔ `core/decider/options.go:70-78`   | MEDIUM   | ✅ Same logic, different params                               |
| **`saveSnapshot` method**                                           | `core/aggregate/repository.go:221-245` ↔ `core/decider/options.go:80-104`  | MEDIUM   | ⚠️ aggregate uses `Root` interface; decider uses generics     |
| **`CatalogMeta` struct**                                            | `core/event`, `core/command`, `core/query` (3 packages)                    | LOW      | ⚠️ `event.CatalogMeta` has extra `AggregateType` field        |
| **`CatalogCore` + `Catalogable` pattern**                           | Same 3 packages                                                            | LOW      | ⚠️ Same structure, different embedded types                   |
| **`opError` helper**                                                | `core/aggregate/repository.go:66-68` ↔ `core/decider/decider.go:239-243`   | LOW      | ✅ Slightly different signatures                              |

### Publisher/Subscriber — Unused in Practice

`event.Publisher` and `event.Subscriber` sub-interfaces were added in Session 44 but **no consumer accepts them as standalone types**. Everyone still takes `event.Bus`. The split is architecturally correct (ISP) but adds zero practical decoupling. Either use them in repos/projections or they're dead types.

### Error Classification — Only 3 of 6+ Packages Registered

| Package          | Registered?    | Sentinels Missing              |
| ---------------- | -------------- | ------------------------------ |
| `core/event`     | ✅ (hardcoded) | —                              |
| `core/command`   | ✅ (`init()`)  | —                              |
| `core/query`     | ✅ (`init()`)  | —                              |
| `core/aggregate` | ❌             | `ErrAggregateNotFound`, etc.   |
| `projection`     | ❌             | `ErrDuplicateProjection`, etc. |
| `storage`        | ❌             | SQL-specific errors            |

---

## C) NOT STARTED ○

| Item                                                    | Priority | Effort           |
| ------------------------------------------------------- | -------- | ---------------- |
| Outbox transaction co-participation                     | CRITICAL | LARGE            |
| `query.Handler` returns `any` → generics                | HIGH     | LARGE (breaking) |
| Deduplicate `SnapshotStrategy` + `EveryNEvents`         | HIGH     | SMALL            |
| Deduplicate `publishChanges` / `saveSnapshot`           | MEDIUM   | MEDIUM           |
| Dead `Publisher`/`Subscriber` — use or remove           | MEDIUM   | SMALL            |
| Register aggregate/projection/storage sentinels         | MEDIUM   | SMALL            |
| Root `go.mod` module path mismatch                      | LOW      | SMALL            |
| Remove redundant `replace` directives (go.work handles) | LOW      | SMALL            |

---

## D) TOTALLY FUCKED UP 💥

### 1. `SnapshotStrategy` + `EveryNEvents` Duplicated VERBATIM

`core/aggregate/options.go:10-31` and `core/decider/options.go:13-31` define **identical** types:

```go
type SnapshotStrategy interface { ... }
type everyN struct { ... }
func EveryNEvents(n int) SnapshotStrategy { ... }
func (e everyN) ShouldSnapshot(...) bool { ... }
```

This is a 22-line copy-paste. Every fix to one must be applied to the other. The only difference is the `EveryNEvents` panic message formatting.

### 2. Root `go.mod` Module Path Mismatch

Root `go.mod` declares `module github.com/LarsArtmann/go-cqrs-lite` (uppercase L, A).
All submodules use `github.com/larsartmann/go-cqrs-lite` (lowercase).

These are **two different module paths**. The root module is empty (no Go code), but this is still technically wrong and could confuse tooling.

### 3. `MemoryStore.LoadAll` at 0% Coverage

`event.GlobalLoader` requires `LoadAll()`. `MemoryStore` implements it. But **no test ever calls it**. The projection runner's replay depends on it, but replay tests use pre-saved events through `Save()` not `LoadAll()`.

### 4. `projection.Runner.Close()` at 0% Coverage

`Close()` is a no-op that returns nil. But the `Runner` declares `var _ io.Closer = (*Runner)(nil)`. No test verifies this contract.

### 5. `OutboxSchema` at 0% Coverage

`storage/outbox.go:17` returns a DDL string constant. Never tested. Dead code that should either be tested or removed.

### 6. 46 Pre-existing Lint Issues

All in test files or edge cases. Breakdown: errcheck (11), wsl_v5 (8), perfsprint (8), noinlineerr (6), nlreturn (6), revive (3), err113 (2), exhaustruct (1), golines (1), modernize (1).

---

## E) WHAT WE SHOULD IMPROVE

### Architecture (HIGH impact)

1. **Extract `SnapshotStrategy` to shared package** — The interface, `EveryNEvents`, and `everyN` struct are identical in `aggregate` and `decider`. Move to `core/event/snapshot_strategy.go` (since both packages already import `core/event`). This eliminates 22 lines of duplication and prevents drift.

2. **Use `Publisher` in repositories, `Subscriber` in projections** — Right now `Publisher`/`Subscriber` are dead types. Make `decider.Repository` and `aggregate.EventSourcedRepository` accept `event.Publisher` instead of `event.Bus`. Make `projection.Runner` accept `event.Subscriber`. This gives real ISP value — repos can't accidentally subscribe, projections can't accidentally publish.

3. **Register aggregate/projection/storage sentinels** — `RegisterClassification` via `init()` works. Add 3 more registrations to cover the full error taxonomy.

### Code Quality (MEDIUM impact)

4. **Fix all 46 lint issues** — Mechanical, ~1 hour. 11 unchecked errors, 8 perfsprint, 8 wsl_v5, 6 nlreturn, etc.

5. **Remove dead `OutboxSchema`** or test it — It's a constant that nobody uses externally.

6. **Remove redundant `replace` directives** — `go.work` resolves local modules. The `replace` blocks in each `go.mod` are unnecessary when using the workspace.

7. **Fix root `go.mod` module path** — Change to lowercase `github.com/larsartmann/go-cqrs-lite` to match submodules.

### Test Quality (MEDIUM impact)

8. **Test `LoadAll`** — `MemoryStore.LoadAll` is an interface method at 0% coverage.

9. **Test `Runner.Close()`** — Verify `io.Closer` contract.

10. **Test `WithLogger` option** — 0% coverage on the option function.

### Type Safety (LOW impact)

11. **`query.Handler` returns `any`** — The only remaining `any` in production code (besides codec boundaries). Breaking change to generics.

12. **`CatalogMeta` consolidation** — 3 nearly-identical structs. `event.CatalogMeta` has extra `AggregateType`. Could use embedding but the package split makes it awkward.

---

## F) TOP 25 THINGS TO DO NEXT

Ranked by impact × effort:

### HIGH IMPACT, SMALL EFFORT (Do first)

| #   | Task                                                                 | Effort | Impact | Detail                                                                       |
| --- | -------------------------------------------------------------------- | ------ | ------ | ---------------------------------------------------------------------------- |
| 1   | **Extract `SnapshotStrategy` to `core/event/snapshot_strategy.go`**  | SMALL  | HIGH   | Eliminate 22-line verbatim duplication between aggregate and decider         |
| 2   | **Fix all 46 lint issues**                                           | SMALL  | HIGH   | Mechanical: errcheck, perfsprint, wsl_v5, nlreturn, noinlineerr, revive      |
| 3   | **Use `event.Publisher` in repos, `event.Subscriber` in projection** | SMALL  | HIGH   | Make ISP split real — repos can only publish, projections can only subscribe |
| 4   | **Register aggregate sentinels**                                     | SMALL  | MEDIUM | `init()` in `core/aggregate/errors.go`                                       |
| 5   | **Register projection sentinels**                                    | SMALL  | MEDIUM | `init()` in `projection/errors.go`                                           |
| 6   | **Register storage sentinels**                                       | SMALL  | MEDIUM | `init()` in `storage/errors.go`                                              |
| 7   | **Test `MemoryStore.LoadAll`**                                       | SMALL  | MEDIUM | 0% → tested                                                                  |
| 8   | **Test `projection.Runner.Close()` and `WithLogger`**                | SMALL  | MEDIUM | 0% → tested                                                                  |
| 9   | **Fix root `go.mod` module path**                                    | SMALL  | LOW    | Uppercase → lowercase                                                        |
| 10  | **Remove or test `OutboxSchema`**                                    | SMALL  | LOW    | Dead code or tested                                                          |

### HIGH IMPACT, MEDIUM EFFORT

| #   | Task                                       | Effort | Impact | Detail                                                      |
| --- | ------------------------------------------ | ------ | ------ | ----------------------------------------------------------- |
| 11  | **Increase `projection` coverage to 95%+** | MEDIUM | HIGH   | `replay` at 73.3%, `collectResults` at 73.3%                |
| 12  | **Increase `storage` coverage to 95%+**    | MEDIUM | HIGH   | `scanOutboxEntries` 75%, `reconstructOutboxEvent` 76.9%     |
| 13  | **Increase `aggregate` coverage to 95%+**  | MEDIUM | HIGH   | `NewCore` 60%, `loadFromStore` 75%                          |
| 14  | **Deduplicate `publishChanges`**           | MEDIUM | MEDIUM | Extract to shared helper in `core/event`                    |
| 15  | **Deduplicate `saveSnapshot`**             | MEDIUM | MEDIUM | aggregate uses `Root`, decider uses generics — needs design |

### MEDIUM IMPACT, MEDIUM EFFORT

| #   | Task                                      | Effort | Impact | Detail                                                         |
| --- | ----------------------------------------- | ------ | ------ | -------------------------------------------------------------- |
| 16  | **Add outbox integration test**           | MEDIUM | MEDIUM | Full cycle: Append → PollPending → Publish → Ack               |
| 17  | **Remove redundant `replace` directives** | SMALL  | LOW    | `go.work` handles resolution                                   |
| 18  | **Tag `v0.1.0-alpha`**                    | SMALL  | MEDIUM | All modules stable enough for early adopters                   |
| 19  | **Add `CHANGELOG.md`**                    | SMALL  | MEDIUM | 46 sessions of changes                                         |
| 20  | **Refactor long functions**               | MEDIUM | LOW    | `validateEventParams` 50L, `collectResults` 41L, `Execute` 45L |

### LARGE EFFORT (Plan carefully)

| #   | Task                                    | Effort | Impact   | Detail                            |
| --- | --------------------------------------- | ------ | -------- | --------------------------------- |
| 21  | **Outbox transaction co-participation** | LARGE  | CRITICAL | Needs interface decision first    |
| 22  | **`query.Handler` → generics**          | LARGE  | HIGH     | Breaking API change               |
| 23  | **Benchmarks**                          | MEDIUM | LOW      | Zero performance benchmarks exist |
| 24  | **Saga / Process Manager**              | LARGE  | LOW      | Complex feature                   |
| 25  | **Watermill module**                    | LARGE  | LOW      | Pub/sub adapter                   |

---

## G) TOP #1 QUESTION

**Should `SnapshotStrategy` live in `core/event/` or a new `core/pkg/snapshot/` package?**

Both `core/aggregate` and `core/decider` import `core/event`. So `core/event/snapshot_strategy.go` is the simplest location — zero new packages, zero new imports. But it adds snapshot-related types to the `event` package which already has 241 lines in `event.go` + 240 lines in `errors.go`.

Alternatively, a new `core/pkg/snapshot/` package keeps `event` focused on event concerns. But it adds a new package to the module.

The right answer is probably `core/event/snapshot_strategy.go` — it's a 30-line file that defines `SnapshotStrategy`, `EveryNEvents`, and `everyN`. Both `aggregate/options.go` and `decider/options.go` would import from `event` (which they already do). Zero circular deps. Zero new packages.

---

## Metrics Dashboard

### Test Coverage by Package

| Package                | Coverage  | Target | Status |
| ---------------------- | --------- | ------ | ------ |
| `core/command`         | 100.0%    | >95%   | ✅     |
| `core/query`           | 100.0%    | >95%   | ✅     |
| `core/pkg/dispatcher`  | 100.0%    | >95%   | ✅     |
| `core/pkg/id`          | 100.0%    | >95%   | ✅     |
| `middleware`           | 100.0%    | >95%   | ✅     |
| `core/event`           | 98.0%     | >95%   | ✅     |
| `catalog/d2`           | 97.6%     | >95%   | ✅     |
| `core/decider`         | 96.2%     | >95%   | ✅     |
| `catalog/asyncapi`     | 95.9%     | >95%   | ✅     |
| `catalog/eventcatalog` | 95.6%     | >95%   | ✅     |
| `catalog/adapters`     | 95.5%     | >95%   | ✅     |
| `catalog`              | 94.4%     | >95%   | ⚠️     |
| `core/aggregate`       | 93.2%     | >95%   | ⚠️     |
| `storage`              | 92.0%     | >95%   | ⚠️     |
| `memory`               | 91.9%     | >95%   | ⚠️     |
| `projection`           | 89.7%     | >95%   | ⚠️     |
| **Total**              | **90.9%** | >95%   | ⚠️     |

### Duplication Heatmap

```
core/event/          ← owns: SnapshotStrategy (SHOULD), Bus, Store, Event, Error taxonomy
core/aggregate/      ← DUPLICATES: SnapshotStrategy, EveryNEvents, shouldSnapshot, saveSnapshot, publishChanges
core/decider/        ← DUPLICATES: SnapshotStrategy, EveryNEvents, shouldSnapshot, saveSnapshot, publishChanges
core/command/        ← CatalogMeta (3×), CatalogCore (3×) — LOW priority
core/query/          ← CatalogMeta (3×), CatalogCore (3×) — LOW priority
```

---

_Generated: 2026-05-03 07:05 — Session 46_
