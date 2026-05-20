# Session 47 — Comprehensive Status Report + Execution Plan

**Date:** 2026-05-03 07:06
**Branch:** `master`
**Test suites:** 21 packages, ALL PASS (including `-race`)
**Total LOC:** 31,509 Go (10,067 production + 21,442 test), 84 test files
**Example LOC:** 553 (example/user)
**Documentation:** 127 markdown files
**Commits since May 1:** 154
**Lint:** 50 pre-existing issues (0 new)
**go vet:** clean
**TODOs/FIXMEs:** 0

---

## A) FULLY DONE ✓

### All Modules (22 test packages)

| Module                 | Coverage | Files | Status       |
| ---------------------- | -------- | ----- | ------------ |
| `core/command`         | 100.0%   | 10    | ✅ Stable    |
| `core/query`           | 100.0%   | 7     | ✅ Stable    |
| `core/pkg/dispatcher`  | 100.0%   | 3     | ✅ Stable    |
| `core/pkg/id`          | 100.0%   | 4     | ✅ Stable    |
| `middleware`           | 100.0%   | 10    | ✅ Stable    |
| `core/event`           | 98.0%    | 16    | ✅ Excellent |
| `catalog/d2`           | 97.6%    | 4     | ✅ Excellent |
| `core/decider`         | 96.2%    | 6     | ✅ Excellent |
| `catalog/asyncapi`     | 95.9%    | 5     | ✅ Excellent |
| `catalog/eventcatalog` | 95.6%    | 4     | ✅ Excellent |
| `catalog/adapters`     | 95.5%    | 3     | ✅ Excellent |
| `catalog`              | 94.4%    | 6     | ✅ Good      |
| `core/aggregate`       | 93.2%    | 5     | ✅ Good      |
| `storage`              | 92.0%    | 6     | ✅ Good      |
| `memory`               | 91.9%    | 7     | ✅ Good      |
| `projection`           | 89.7%    | 5     | ⚠️ Below 95% |

### BDD Test Suites (34 specs)

| Module         | Specs | Commit    |
| -------------- | ----- | --------- |
| `core/decider` | 13    | `9e65222` |
| `projection`   | 11    | `ba7f52d` |
| `memory`       | 10    | `ab92ed2` |

### Architecture Achievements

- Error taxonomy (5 families: Rejection, Conflict, Transient, Corruption, Infrastructure)
- Extensible `RegisterClassification` for cross-package sentinel mapping
- `Publisher`/`Subscriber` sub-interfaces on `event.Bus` (ISP)
- Branded IDs (`id.Of[T]` = type alias to `go-branded-id`) with 8 branded types
- `Decider[State]` pure-function aggregate pattern (recommended over OO aggregate)
- Snapshot support (aggregate + decider) with codec + strategy options
- Outbox pattern (memory + SQL) with panic recovery
- Catalog system (AsyncAPI 3.0, D2 diagrams, EventCatalog MDX exporters)
- Projection runner with replay, live subscription, retry, injected logger
- Multi-module monorepo (10 modules) via `go.work`
- 3 ADRs (decider pattern, error taxonomy, multi-module monorepo)
- `CONTEXT.md` domain glossary (20 terms)

---

## B) PARTIALLY DONE ⚠️

### Duplication Heatmap (VERBATIM COPIES)

| Duplication                                                     | Location A                                 | Location B                        | Lines          | Fixable?                                        |
| --------------------------------------------------------------- | ------------------------------------------ | --------------------------------- | -------------- | ----------------------------------------------- |
| `SnapshotStrategy` interface + `EveryNEvents` + `everyN` struct | `core/aggregate/options.go:10-31`          | `core/decider/options.go:13-31`   | 22 lines       | ✅ Extract to `core/event/snapshot_strategy.go` |
| `publishChanges` method                                         | `core/aggregate/repository.go:104-123`     | `core/decider/decider.go:218-237` | ~20 lines      | ⚠️ Different receiver signatures                |
| `shouldSnapshot` method                                         | `core/aggregate/repository.go:214-219`     | `core/decider/options.go:70-78`   | ~8 lines       | ✅ Same logic                                   |
| `saveSnapshot` method                                           | `core/aggregate/repository.go:221-245`     | `core/decider/options.go:80-104`  | ~25 lines      | ⚠️ aggregate uses `Root`, decider uses generics |
| `CatalogMeta` struct                                            | `core/event`, `core/command`, `core/query` | (3 packages)                      | ~15 lines each | ⚠️ event has extra `AggregateType`              |
| `opError` helper                                                | `core/aggregate/repository.go:66-68`       | `core/decider/decider.go:239-243` | 3 lines        | ✅ Slightly different signatures                |

### ISP Split — Dead Types

`event.Publisher` and `event.Subscriber` sub-interfaces exist but **no consumer uses them standalone**. All code still takes `event.Bus`. The split is architecturally correct but adds zero practical decoupling today.

### Error Classification — 3 of 6+ Packages Registered

| Package          | Registered?    | Missing Sentinels        |
| ---------------- | -------------- | ------------------------ |
| `core/event`     | ✅ (hardcoded) | —                        |
| `core/command`   | ✅ (`init()`)  | —                        |
| `core/query`     | ✅ (`init()`)  | —                        |
| `core/aggregate` | ❌             | `ErrAggregateNotFound`   |
| `projection`     | ❌             | `ErrDuplicateProjection` |
| `storage`        | ❌             | SQL-specific errors      |

---

## C) NOT STARTED ○

| Item                                      | Priority | Effort           | Customer Value              |
| ----------------------------------------- | -------- | ---------------- | --------------------------- |
| Outbox transaction co-participation       | CRITICAL | LARGE            | Production readiness        |
| `query.Handler` returns `any` → generics  | HIGH     | LARGE (breaking) | Type safety                 |
| Deduplicate `SnapshotStrategy`            | HIGH     | SMALL            | Maintainability             |
| Use `Publisher`/`Subscriber` in consumers | MEDIUM   | SMALL            | ISP value                   |
| Register remaining error sentinels        | MEDIUM   | SMALL            | Error taxonomy completeness |
| Fix root `go.mod` module path             | LOW      | SMALL            | Correctness                 |
| Remove redundant `replace` directives     | LOW      | SMALL            | Cleanup                     |
| `CHANGELOG.md`                            | MEDIUM   | SMALL            | Adopter trust               |
| Tag `v0.1.0-alpha`                        | MEDIUM   | SMALL            | Adopter trust               |
| Benchmarks                                | LOW      | MEDIUM           | Performance claims          |
| Saga / Process Manager                    | LOW      | LARGE            | Feature richness            |
| Watermill module                          | LOW      | LARGE            | Integration breadth         |

---

## D) TOTALLY FUCKED UP 💥

### 1. `SnapshotStrategy` + `EveryNEvents` — VERBATIM 22-line Copy-Paste

`core/aggregate/options.go:10-31` and `core/decider/options.go:13-31` define **identical** types. Every fix to one MUST be applied to the other. This WILL drift and cause bugs.

### 2. Root `go.mod` Module Path Mismatch

Root declares `github.com/LarsArtmann/go-cqrs-lite` (uppercase L, A).
All submodules use `github.com/larsartmann/go-cqrs-lite` (lowercase).
Two different module paths. Currently harmless but confusing.

### 3. `MemoryStore.LoadAll` at 0% Coverage

`event.GlobalLoader` requires `LoadAll()`. `MemoryStore` implements it. Zero tests call it. The projection runner's replay depends on this interface.

### 4. `projection.Runner.Close()` at 0% Coverage

`Close()` returns nil. `var _ io.Closer` compile check exists but no test verifies the contract.

### 5. `OutboxSchema` at 0% Coverage

`storage/outbox.go:17` returns a DDL constant. Never tested. Dead code or untested API.

### 6. 50 Pre-existing Lint Issues

Breakdown: errcheck (10), wsl_v5 (11), perfsprint (8), noinlineerr (6), nlreturn (6), revive (3), err113 (2), intrange (2), exhaustruct (1), modernize (1).

### 7. Files Near 250-Line Limit

| File                           | Lines | Status             |
| ------------------------------ | ----- | ------------------ |
| `core/aggregate/repository.go` | 245   | ⚠️ 5 lines margin  |
| `catalog/asyncapi/exporter.go` | 244   | ⚠️ 6 lines margin  |
| `core/decider/decider.go`      | 243   | ⚠️ 7 lines margin  |
| `core/event/event.go`          | 241   | ⚠️ 9 lines margin  |
| `core/event/errors.go`         | 240   | ⚠️ 10 lines margin |
| `projection/runner.go`         | 237   | ✅ OK              |
| `storage/outbox.go`            | 235   | ✅ OK              |

---

## E) WHAT WE SHOULD IMPROVE

### Architecture (HIGH impact)

1. **Extract `SnapshotStrategy` to `core/event/snapshot_strategy.go`** — Eliminate 22-line verbatim duplication. Both packages already import `core/event`.
2. **Use `Publisher` in repositories, `Subscriber` in projections** — Make ISP split real.
3. **Register aggregate/projection/storage sentinels** — Complete error taxonomy coverage.

### Code Quality (MEDIUM impact)

4. **Fix all 50 lint issues** — Mechanical but necessary for `v0.1.0`.
5. **Remove or test `OutboxSchema`** — Dead code or tested.
6. **Remove redundant `replace` directives** — `go.work` handles resolution.
7. **Fix root `go.mod` module path** — Uppercase → lowercase.

### Test Quality (MEDIUM impact)

8. **Test `LoadAll`** — Interface method at 0% coverage.
9. **Test `Runner.Close()`** — Verify `io.Closer` contract.
10. **Test `WithLogger` option** — 0% coverage.

### Type Safety (LOW impact, HIGH effort)

11. **`query.Handler` returns `any`** — Only remaining `any` in production code. Breaking change.
12. **`CatalogMeta` consolidation** — 3 nearly-identical structs. Package split makes it awkward.

---

## F) COMPREHENSIVE EXECUTION PLAN — ALL TASKS (max 12 min each)

Tasks sorted by: **Importance → Impact → Customer Value → Effort (ascending)**

### Phase 1: Duplication Elimination (HIGH impact, SMALL effort)

| #   | Task                                                                                       | Est. | Impact | Detail                                  |
| --- | ------------------------------------------------------------------------------------------ | ---- | ------ | --------------------------------------- |
| 1   | Create `core/event/snapshot_strategy.go` with `SnapshotStrategy`, `EveryNEvents`, `everyN` | 8min | HIGH   | Extract from aggregate; decider follows |
| 2   | Update `core/aggregate/options.go` to import from `event`                                  | 5min | HIGH   | Delete 22 lines, add import alias       |
| 3   | Update `core/decider/options.go` to import from `event`                                    | 5min | HIGH   | Delete 22 lines, add import alias       |
| 4   | Run tests to verify SnapshotStrategy extraction                                            | 3min | HIGH   | Full test suite                         |
| 5   | Extract `shouldSnapshot` to shared helper in `core/event`                                  | 8min | MEDIUM | ~8 lines shared logic                   |
| 6   | Update aggregate + decider to use shared `shouldSnapshot`                                  | 5min | MEDIUM | Delete duplicated methods               |
| 7   | Run tests to verify shouldSnapshot extraction                                              | 3min | MEDIUM | Full test suite                         |

### Phase 2: ISP Activation (HIGH impact, SMALL effort)

| #   | Task                                                                           | Est.  | Impact | Detail                                     |
| --- | ------------------------------------------------------------------------------ | ----- | ------ | ------------------------------------------ |
| 8   | Update `decider.Repository` to accept `event.Publisher` instead of `event.Bus` | 8min  | HIGH   | Change constructor + field type            |
| 9   | Update `aggregate.EventSourcedRepository` to accept `event.Publisher`          | 8min  | HIGH   | Change constructor + field type            |
| 10  | Update `projection.Runner` to accept `event.Subscriber` instead of `event.Bus` | 8min  | HIGH   | Change constructor + field type            |
| 11  | Update `OutboxPublisher` to accept `event.Publisher`                           | 5min  | MEDIUM | Only needs Publish, not Subscribe          |
| 12  | Update `event.Runner` to accept `event.Subscriber`                             | 5min  | MEDIUM | Only needs Subscribe                       |
| 13  | Update all callers (tests, example/user, integration)                          | 10min | HIGH   | Find all `NewRepository`/`NewRunner` calls |
| 14  | Run tests to verify ISP activation                                             | 3min  | HIGH   | Full test suite                            |

### Phase 3: Error Classification Completion (MEDIUM impact, SMALL effort)

| #   | Task                                                             | Est. | Impact | Detail                                     |
| --- | ---------------------------------------------------------------- | ---- | ------ | ------------------------------------------ |
| 15  | Add `init()` in `core/aggregate/errors.go` registering sentinels | 5min | MEDIUM | ErrAggregateNotFound → Rejection           |
| 16  | Add `init()` in `projection/errors.go` registering sentinels     | 5min | MEDIUM | ErrDuplicateProjection → Conflict          |
| 17  | Add `init()` in `storage/errors.go` registering sentinels        | 8min | MEDIUM | SQL-specific errors → appropriate families |
| 18  | Add tests for `Classify()` with all newly registered sentinels   | 8min | MEDIUM | `errors.Is` checks for each                |
| 19  | Run tests to verify error classification                         | 3min | MEDIUM | Full test suite                            |

### Phase 4: Lint Cleanup — Mechanical Fixes (MEDIUM impact, SMALL effort)

| #   | Task                                                                           | Est.  | Impact | Detail                                     |
| --- | ------------------------------------------------------------------------------ | ----- | ------ | ------------------------------------------ |
| 20  | Fix `errcheck` (10 issues) — add error checks for `AppendBatch`, `store.Save`  | 10min | MEDIUM | `_ = store.AppendBatch(...)` → error check |
| 21  | Fix `perfsprint` (8 issues) — `fmt.Errorf` → `errors.New` where no format args | 8min  | LOW    | Mechanical replacement                     |
| 22  | Fix `wsl_v5` (11 issues) — add whitespace above if/assign statements           | 10min | LOW    | Formatting only                            |
| 23  | Fix `noinlineerr` (6 issues) — split inline error handling                     | 8min  | LOW    | `if err := ...; err != nil` → 2 lines      |
| 24  | Fix `nlreturn` (6 issues) — add blank line before returns                      | 5min  | LOW    | Formatting only                            |
| 25  | Fix `revive` unused params (3 issues) — rename to `_`                          | 3min  | LOW    | Parameter cleanup                          |
| 26  | Fix `err113` (2 issues) — replace dynamic errors with wrapped static           | 5min  | LOW    | Wrap sentinel errors                       |
| 27  | Fix `intrange` (2 issues) — use integer range in BDD tests                     | 3min  | LOW    | Go 1.22+ syntax                            |
| 28  | Fix `exhaustruct` (1 issue) — initialize `mu` field in anonymous struct        | 2min  | LOW    | Add mutex init                             |
| 29  | Fix `modernize` (1 issue) — use `WaitGroup.Go`                                 | 2min  | LOW    | Go 1.26+ API                               |
| 30  | Run lint to verify 0 issues                                                    | 3min  | HIGH   | Confirm clean                              |

### Phase 5: Test Coverage Gaps (MEDIUM impact, SMALL-MEDIUM effort)

| #   | Task                                                           | Est.  | Impact | Detail                  |
| --- | -------------------------------------------------------------- | ----- | ------ | ----------------------- |
| 31  | Test `MemoryStore.LoadAll` — verify returns all events sorted  | 8min  | MEDIUM | 0% → tested             |
| 32  | Test `projection.Runner.Close()` — verify returns nil          | 5min  | MEDIUM | 0% → tested             |
| 33  | Test `projection.WithLogger` option — verify logger is set     | 5min  | MEDIUM | 0% → tested             |
| 34  | Test `OutboxSchema` — verify DDL string is valid SQL           | 5min  | LOW    | 0% → tested or delete   |
| 35  | Increase `projection` coverage: replay edge cases              | 10min | HIGH   | 89.7% → 92%+            |
| 36  | Increase `projection` coverage: collectResults error paths     | 10min | HIGH   | 73.3% function coverage |
| 37  | Increase `storage` coverage: scanOutboxEntries error paths     | 8min  | MEDIUM | 75% → 90%+              |
| 38  | Increase `storage` coverage: reconstructOutboxEvent edge cases | 8min  | MEDIUM | 76.9% → 90%+            |
| 39  | Increase `aggregate` coverage: NewCore validation paths        | 8min  | MEDIUM | 60% → 90%+              |
| 40  | Increase `aggregate` coverage: loadFromStore error paths       | 8min  | MEDIUM | 75% → 90%+              |
| 41  | Increase `memory` coverage: Bus/Store error paths              | 8min  | MEDIUM | 91.9% → 95%+            |
| 42  | Run full test suite + coverage report                          | 3min  | HIGH   | Verify all improvements |

### Phase 6: Code Quality Cleanup (LOW-MEDIUM impact, SMALL effort)

| #   | Task                                                          | Est. | Impact | Detail                                    |
| --- | ------------------------------------------------------------- | ---- | ------ | ----------------------------------------- |
| 43  | Fix root `go.mod` module path: `LarsArtmann` → `larsartmann`  | 3min | LOW    | Case-only change                          |
| 44  | Remove redundant `replace` directives from all `go.mod` files | 8min | LOW    | `go.work` handles resolution              |
| 45  | Remove or test `OutboxSchema` constant                        | 5min | LOW    | Decision: test or delete                  |
| 46  | Trim `core/aggregate/repository.go` (245 → under 240)         | 8min | LOW    | Extract `publishChanges` to event helper  |
| 47  | Trim `catalog/asyncapi/exporter.go` (244 → under 240)         | 8min | LOW    | Extract helper function                   |
| 48  | Trim `core/decider/decider.go` (243 → under 240)              | 5min | LOW    | Already uses shared helpers after Phase 1 |

### Phase 7: Deduplicate publishChanges + saveSnapshot (MEDIUM impact, MEDIUM effort)

| #   | Task                                                             | Est.  | Impact | Detail                              |
| --- | ---------------------------------------------------------------- | ----- | ------ | ----------------------------------- |
| 49  | Extract `publishChanges` to `core/event/publish_helper.go`       | 10min | MEDIUM | Generic version that works for both |
| 50  | Update `aggregate/repository.go` to use shared `publishChanges`  | 8min  | MEDIUM | Delete local method                 |
| 51  | Update `decider/decider.go` to use shared `publishChanges`       | 8min  | MEDIUM | Delete local method                 |
| 52  | Design shared `saveSnapshot` — needs adapter for Root vs generic | 10min | MEDIUM | Interface-based or generic          |
| 53  | Implement shared `saveSnapshot` in `core/event`                  | 8min  | MEDIUM | With snapshot store + codec         |
| 54  | Update aggregate + decider to use shared `saveSnapshot`          | 10min | MEDIUM | Delete local methods                |
| 55  | Run tests to verify publishChanges + saveSnapshot extraction     | 3min  | MEDIUM | Full test suite                     |

### Phase 8: Documentation + Release Prep (MEDIUM impact, SMALL effort)

| #   | Task                                                      | Est.  | Impact | Detail                     |
| --- | --------------------------------------------------------- | ----- | ------ | -------------------------- |
| 56  | Create `CHANGELOG.md` — 46 sessions of changes            | 12min | MEDIUM | Structured by session      |
| 57  | Update `TODO_LIST.md` to current state                    | 8min  | MEDIUM | Reflect all Phase 1-7 work |
| 58  | Update `FEATURES.md` to reflect ISP + snapshot extraction | 8min  | MEDIUM | Add new public types       |
| 59  | Update `AGENTS.md` with Session 47 changes                | 5min  | MEDIUM | Memory update              |
| 60  | Verify all documentation freshness                        | 5min  | LOW    | Cross-check with code      |

### Phase 9: Future-Looking (LOW impact, LARGE effort)

| #   | Task                                                  | Est.  | Impact   | Detail                |
| --- | ----------------------------------------------------- | ----- | -------- | --------------------- |
| 61  | Tag `v0.1.0-alpha` after all phases complete          | 5min  | MEDIUM   | First public release  |
| 62  | Add performance benchmarks for core dispatch path     | 10min | LOW      | Zero benchmarks exist |
| 63  | Add performance benchmarks for event store operations | 10min | LOW      | Storage benchmark     |
| 64  | Design outbox transaction co-participation API        | 12min | CRITICAL | Interface design only |
| 65  | Design `query.Handler` generics migration             | 10min | HIGH     | Breaking change plan  |
| 66  | Review `docs/planning/SAGA_DESIGN.md` for next steps  | 8min  | LOW      | Refresh if needed     |

---

## G) TOP #1 QUESTION

**Should `SnapshotStrategy` live in `core/event/` or a new `core/pkg/snapshot/`?**

Both `aggregate` and `decider` import `core/event`. So `core/event/snapshot_strategy.go` is simplest — zero new packages, zero new imports. It's a ~30-line file. The alternative (`core/pkg/snapshot/`) keeps `event` focused but adds a package.

**Recommendation:** `core/event/snapshot_strategy.go` — YAGNI on the new package.

---

## Summary Statistics

| Metric                 | Value                                |
| ---------------------- | ------------------------------------ |
| Production LOC         | 10,067                               |
| Test LOC               | 21,442                               |
| Test/Code Ratio        | 2.13:1                               |
| Total Coverage         | 84.1% (packages), 90.9% (statements) |
| Packages ≥95%          | 11 of 16                             |
| Lint Issues            | 50 (all pre-existing)                |
| Race Issues            | 0                                    |
| Open TODOs             | 66 tasks across 9 phases             |
| Estimated Total Effort | ~8.5 hours                           |

_Generated: 2026-05-03 07:06 — Session 47_
