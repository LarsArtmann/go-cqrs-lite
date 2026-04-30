# Comprehensive Status Report

**Date:** 2026-04-26 12:09 CEST
**Branch:** master
**Last Commit:** `6815ef3` — refactor(example): use aggregate.EventSourcedRepository instead of hand-rolled repo
**Total Go Code:** ~14,001 lines across 93 files in 5 modules + 2 examples
**Test Packages:** 13 packages, all green with race detection
**Overall Status:** Post-migration cleanup nearly complete. ULID migration done. Lint config valid. Dead code purged. Documentation updated.

---

## Test Coverage Snapshot

| Package                | Coverage  | Trend  |
| ---------------------- | --------- | ------ |
| `catalog/adapters`     | **98.8%** | stable |
| `memory`               | 95.9%     | -3.3%  |
| `catalog/asyncapi`     | 97.6%     | +1.3%  |
| `xtypes`               | 95.7%     | stable |
| `query`                | 91.4%     | -0.1%  |
| `catalog`              | 86.9%     | -4.3%  |
| `catalog/eventcatalog` | 89.7%     | stable |
| `event`                | 89.0%     | stable |
| `middleware`           | 85.6%     | +1.0%  |
| `command`              | 84.4%     | stable |
| `aggregate`            | 86.0%     | +8.7%  |
| `pkg/dispatcher`       | 77.4%     | stable |
| `pkg/id`               | **63.6%** | -21.8% |

**Note:** `pkg/id` coverage dropped from 85.4% to 63.6% after ULID migration — the package now delegates to `go-composable-business-types/id` and is only 28 lines, but test coverage didn't follow the slimming.

---

## Lint Status

`.golangci.yml` passes `golangci-lint config verify` (zero schema errors). Remaining lint issues:

| Module     | Issues | Details                                                                                                                                                                                   |
| ---------- | ------ | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | --------- |
| core       | 14     | depguard (2 — `go-composable-business-types/id`, `oklog/ulid` not in allow-list), err113 (2 — dynamic errors in `id.go`, `repository.go`), wsl_v5 (10 — whitespace style in `id_test.go`) |
| catalog    | 1      | exhaustive switch on `reflect.Kind` missing `Pointer                                                                                                                                      | Ptr` case |
| middleware | 1      | nlreturn in `middleware_test.go:339`                                                                                                                                                      |
| memory     | 0      | Clean                                                                                                                                                                                     |
| xtypes     | 0      | Clean                                                                                                                                                                                     |

---

## A) FULLY DONE

### Multi-Module Migration (Phases 0–4)

| Phase | Description                                                   | Status |
| ----- | ------------------------------------------------------------- | ------ |
| 0     | Fix query handler ctx, delete pkg/errors, replace custom YAML | DONE   |
| 1     | go.work + move into `core/` subdirectory                      | DONE   |
| 2     | Extract `memory/` module                                      | DONE   |
| 3     | Extract `catalog/` module                                     | DONE   |
| 4     | Extract `middleware/` + `xtypes`                              | DONE   |

### Post-Migration Cleanup

| What                                 | Details                                                                                                                    |
| ------------------------------------ | -------------------------------------------------------------------------------------------------------------------------- |
| Remove `query.Result[T]`             | Zero callers — dead type removed                                                                                           |
| Remove unused error sentinels        | `ErrEventNotFound`, `ErrInvalidEventType`, `ErrCommandValidation` — never returned anywhere                                |
| Fix `.golangci.yml` v2 schema errors | Removed stale `wrapcheck.local-prefixes`, empty `formatters`, migrated `issues.exclude-rules` → `linters.exclusions.rules` |
| Remove redundant `//nolint:err113`   | 7 directives removed from test files (now covered by global exclusion)                                                     |
| CONTRIBUTING.md rewritten            | Multi-module workflow, `GOWORK=off`, replace directives, adding new modules                                                |
| CI badges on README.md               | Tests, Lint, Go Reference badges added                                                                                     |
| AGENTS.md updated                    | Coverage table corrected, cleanup section added, known issues updated                                                      |

### ULID Migration

| What                    | Details                                                           |
| ----------------------- | ----------------------------------------------------------------- |
| `core/pkg/id` rewritten | 222 → 28 lines, delegates to `go-composable-business-types/id`    |
| All 5 modules updated   | core, memory, catalog, middleware, xtypes — all pass with `-race` |
| `oklog/ulid/v2` added   | New dependency for ULID-based IDs                                 |

### Code Quality Improvements (Since Last Report)

| Commit    | What                                                                |
| --------- | ------------------------------------------------------------------- |
| `8f1824d` | Comprehensive dead code removal + lint config fix                   |
| `e7977ac` | ULID migration (UUID v4 → ULID via go-composable-business-types)    |
| `91523c1` | ULID migration status documentation                                 |
| `5ad0356` | Fix dispatcher: use `ErrDispatcherClosed` in `CheckClosed`          |
| `d5ea811` | Fix memory: return defensive copies from `Load`/`LoadFromVersion`   |
| `1862eae` | Remove more dead code                                               |
| `8e5150c` | Unify lifecycle management across `MemoryBus`/`MemorySnapshotStore` |
| `4fdd447` | Add `EventValidation` middleware                                    |
| `c1bc261` | Extract `MessageID` to catalog package                              |
| `699d247` | Extract `Option`/`With*` functions to `event/options.go`            |
| `b23a781` | Remove dead `reflect.Ptr` case in `goTypeToJSON`                    |
| `e84e3a1` | Remove unused handler parameter from `Dispatch`                     |
| `6815ef3` | Use `aggregate.EventSourcedRepository` in example                   |

---

## B) PARTIALLY DONE

### depguard Configuration

The `.golangci.yml` depguard allow-list doesn't include `github.com/larsartmann/go-composable-business-types/id` or `github.com/oklog/ulid/v2`, producing 2 lint errors. Needs update.

### pkg/id Test Coverage

Dropped from 85.4% to 63.6% after ULID migration. The package is only 28 lines now but the test file still has gaps. Needs new tests for the ULID delegation layer.

### example/user Module

Updated to use `aggregate.EventSourcedRepository` but still depends on `go-composable-business-types` which isn't in `go.work`. Cannot build independently with `GOWORK=off go build`.

---

## C) NOT STARTED (Planned from Migration Plan)

| Phase | Description                              | Priority | Dependencies                              |
| ----- | ---------------------------------------- | -------- | ----------------------------------------- |
| 5     | Storage module (sqlc event store)        | HIGH     | Codec interface (done), PostgreSQL schema |
| 6     | Watermill module (pub/sub)               | HIGH     | core/event/Bus interface                  |
| 7     | Projection module (samber/ro internally) | MEDIUM   | core/event/Store, Watermill               |
| 8     | Snapshot module (SQL-backed)             | MEDIUM   | core/event/SnapshotStore interface        |
| 9     | Test utilities module                    | LOW      | Extract testutil/testhelpers from core    |
| 10    | Tag releases (v1.0.0)                    | LOW      | All modules stable                        |

### Not Started — Code Quality Items

| Item                                                                               | Effort | Impact |
| ---------------------------------------------------------------------------------- | ------ | ------ |
| Fix depguard allow-list for ULID/oklog deps                                        | 5 min  | MEDIUM |
| Fix `pkg/id` test coverage to 80%+                                                 | 1h     | MEDIUM |
| Fix `wsl_v5` violations in `id_test.go`                                            | 15 min | LOW    |
| Fix `exhaustive` switch in `catalog/schema.go`                                     | 5 min  | LOW    |
| Fix `nlreturn` in `middleware_test.go:339`                                         | 2 min  | LOW    |
| Fix `err113` dynamic errors in `id.go:53`, `repository.go:92`                      | 15 min | LOW    |
| Add integration example using middleware + xtypes together                         | 2h     | HIGH   |
| Write Go doc `Example*` test functions                                             | 2h     | HIGH   |
| Define `Projection` interface in `core/projection/`                                | 1h     | HIGH   |
| Add `example/ecommerce/` full-stack example                                        | 4h     | HIGH   |
| Define `Upcaster` interface in `core/upcasting/`                                   | 1h     | MEDIUM |
| Write formal CHANGELOG.md entries for v0.3.0                                       | 1h     | MEDIUM |
| Benchmark Codec implementations                                                    | 30 min | LOW    |
| Add fuzz targets for event parsing, ID parsing, schema reflection                  | 2h     | LOW    |
| Publish `go-composable-business-types` as proper Go module                         | 1h     | MEDIUM |
| Update `example/user/` go.mod for independent builds                               | 30 min | MEDIUM |
| Investigate `go 1.26 ignore` directive for examples in go.work                     | 30 min | LOW    |
| Fix `MemoryBus.Subscribe` nil handler check                                        | 15 min | LOW    |
| Fix asyncapi component message key collision                                       | 30 min | MEDIUM |
| Standardize errors: replace `fmt.Errorf` with `cockroachdb/errors` in event/xtypes | 30 min | LOW    |
| Add `Payload()`/`Metadata()` immutability by returning copies                      | 30 min | MEDIUM |
| Fix `time.Time` schema generation to use `{type:"string", format:"date-time"}`     | 15 min | LOW    |

---

## D) TOTALLY FUCKED UP

### go-composable-business-types Dependency Leak

The ULID migration introduced `github.com/larsartmann/go-composable-business-types/id` and `github.com/oklog/ulid/v2` as production dependencies of the `core` module. This has two problems:

1. **depguard violations** — The depguard allow-list doesn't include these imports, so `golangci-lint run` produces errors on `core/pkg/id/id.go`
2. **Unpublished module** — `go-composable-business-types` is a private/unpublished module. Anyone cloning `go-cqrs-lite` cannot `go mod download` without access to this dependency. This breaks `GOWORK=off go mod tidy` for all dependent modules.

The `example/user/` module also directly imports `go-composable-business-types` for domain types, which is fine conceptually but further entrenches the unpublished dependency problem.

**This is the single biggest risk to the project right now.** The code compiles in this workspace because of `go.work` replace directives, but it's not portable.

### pkg/id Coverage Regression

Test coverage dropped 21.8% (85.4% → 63.6%) after ULID migration. The package was rewritten from 222 lines to 28 lines by delegating to `go-composable-business-types/id`, but the test file was not updated to match. Several exported functions are now untested.

### Prior Session Left Broken Tests (Historical Note)

The middleware extraction commit `563f126` had three syntax errors (detached if blocks, duplicate imports, wrong MaxAttempts). This was fixed in commit `569adf7`. The lesson: **always run `go build` + `go test` after writing code**.

---

## E) WHAT WE SHOULD IMPROVE

### Architecture

1. **No real event store** — `MemoryStore` is the only implementation. Phase 5 (storage module) is the highest-impact next feature. The `Codec` interface is ready.

2. **No pub/sub** — Events are published synchronously via `MemoryBus`. Watermill integration (Phase 6) is critical for production.

3. **Unpublished transitive dependency** — `go-composable-business-types` is not accessible outside this workspace. Either publish it or inline the ULID logic back into `core/pkg/id`.

4. **Branded IDs underused** — Only 4 marker types exist. `xtypes` adds `CommandID` but nobody uses it.

5. **No `Projection` interface** — The migration plan references `core/projection/` but no such package exists. Needed before Phase 7.

### Code Quality

6. **depguard config stale** — Allow-list doesn't include ULID/oklog deps.

7. **`pkg/id` test gaps** — Coverage 63.6% is below the 80% bar.

8. **Style linter violations** — 10 `wsl_v5` violations in `id_test.go`, 1 `nlreturn` in `middleware_test.go`, 1 `exhaustive` in `catalog/schema.go`.

9. **No Go doc examples** — None of our packages have `Example*` test functions for pkg.go.dev.

10. **Benchmark coverage thin for new code** — No benchmarks for `Codec`, ULID-based IDs, or the new `EventValidation` middleware.

### Documentation

11. **CHANGELOG.md outdated** — No entries for ULID migration, dead code removal, or any post-migration work.

12. **TODO_LIST.md stale** — Many items already done (marked with ✅) but not committed; other items reference deleted code (e.g., `catalog/yaml` coverage).

### Tooling

13. **LSP noise from examples** — gopls tries to load example modules not in `go.work`, producing 47+ diagnostic errors. Could add `go.work` entries for examples or add `.golangci.yml` scoping.

14. **`go.work.sum` drift** — Updated checksums not committed. Minor housekeeping.

---

## F) Top 25 Next Actions (Sorted by Impact × Effort)

### HIGH IMPACT, LOW EFFORT (Do These First)

| #   | Action                                                                           | Effort     | Impact | Why                           |
| --- | -------------------------------------------------------------------------------- | ---------- | ------ | ----------------------------- | ----------------- |
| 1   | Fix depguard allow-list: add `go-composable-business-types/id` + `oklog/ulid/v2` | 5 min      | MEDIUM | Lint currently errors on core |
| 2   | Fix `nlreturn` in `middleware_test.go:339`                                       | 2 min      | LOW    | Trivial lint fix              |
| 3   | Fix `exhaustive` switch in `catalog/schema.go` (add `reflect.Pointer             | Ptr` case) | 5 min  | LOW                           | One-line lint fix |
| 4   | Commit `go.work.sum` changes                                                     | 2 min      | LOW    | Housekeeping                  |
| 5   | Update TODO_LIST.md: mark done items, remove stale entries                       | 15 min     | LOW    | Accuracy                      |

### HIGH IMPACT, MEDIUM EFFORT (Next Sprint)

| #   | Action                                                                               | Effort | Impact | Why                            |
| --- | ------------------------------------------------------------------------------------ | ------ | ------ | ------------------------------ |
| 6   | Fix `pkg/id` test coverage to 80%+                                                   | 1h     | MEDIUM | Coverage below bar             |
| 7   | Fix `wsl_v5` violations in `id_test.go`                                              | 15 min | LOW    | Code style                     |
| 8   | Resolve `err113` in `id.go:53` and `repository.go:92`                                | 15 min | LOW    | Use sentinel + wrapping        |
| 9   | Write CHANGELOG.md entries for post-migration work                                   | 1h     | MEDIUM | Release tracking               |
| 10  | Publish `go-composable-business-types` as proper Go module                           | 1h     | HIGH   | Breaks portability             |
| 11  | Or: inline ULID logic back into `core/pkg/id` to remove the dependency               | 2h     | HIGH   | Alternative to #10             |
| 12  | Define `Projection` interface in `core/projection/`                                  | 1h     | HIGH   | Foundation for Phase 7         |
| 13  | Write integration example using middleware + xtypes + core + memory                  | 2h     | HIGH   | First real validation of API   |
| 14  | Add Go doc `Example*` test functions for command, event, query, aggregate            | 2h     | HIGH   | pkg.go.dev discoverability     |
| 15  | Update `example/user/` for independent builds (remove composable-business-types dep) | 30 min | MEDIUM | Example should work standalone |

### HIGH IMPACT, HIGH EFFORT (Major Features)

| #   | Action                                                               | Effort   | Impact   | Why                                 |
| --- | -------------------------------------------------------------------- | -------- | -------- | ----------------------------------- |
| 16  | **Phase 5: Storage module** (sqlc PostgreSQL event store)            | 2-3 days | CRITICAL | First real persistence layer        |
| 17  | **Phase 6: Watermill module** (pub/sub integrations)                 | 2-3 days | CRITICAL | Production-grade event distribution |
| 18  | **Phase 7: Projection module** (event handlers → SQL tables)         | 2-3 days | HIGH     | Read-model generation               |
| 19  | **Phase 8: Snapshot module** (SQL-backed snapshots)                  | 1-2 days | MEDIUM   | Aggregate load performance          |
| 20  | Write comprehensive integration test suite (core + memory + storage) | 1 day    | HIGH     | Confidence in module interactions   |

### MEDIUM IMPACT, VARIOUS EFFORT

| #   | Action                                                            | Effort | Impact | Why                              |
| --- | ----------------------------------------------------------------- | ------ | ------ | -------------------------------- |
| 21  | Add benchmarks for Codec, ULID IDs, EventValidation middleware    | 2h     | MEDIUM | Performance regression detection |
| 22  | Define `Upcaster` interface in `core/upcasting/`                  | 1h     | MEDIUM | Event schema evolution           |
| 23  | Add fuzz targets for event parsing, ID parsing, schema reflection | 2h     | LOW    | Edge case coverage               |
| 24  | Add `example/ecommerce/` full-stack example (all modules)         | 4h     | HIGH   | "Kitchen sink" demo              |
| 25  | Investigate `go 1.26 ignore` directive for examples/ in go.work   | 30 min | LOW    | Clean `go test ./...`            |

---

## G) Top Question I Cannot Figure Out Myself

**Should we publish `go-composable-business-types` as a standalone Go module, or inline the ULID ID logic back into `core/pkg/id`?**

The ULID migration replaced 222 lines of hand-rolled UUID-based ID code with 28 lines delegating to `go-composable-business-types/id`. This is elegant but creates a critical portability problem: no one can build this project without access to the unpublished `go-composable-business-types` module.

**Option A: Publish `go-composable-business-types`**

- Pros: Clean separation of concerns, reusable across projects
- Cons: Yet another module to maintain, version, and document. It's a very thin wrapper over `oklog/ulid`.

**Option B: Inline the ULID logic into `core/pkg/id`**

- Pros: Zero external deps beyond `oklog/ulid`, portable, self-contained
- Cons: Adds ~50-80 lines back to `core/pkg/id`, slightly less DRY

**Option C: Remove ULID, go back to UUID**

- Pros: `google/uuid` was already a dep, zero new deps, fully portable
- Cons: Loses sortability, time-ordering, and other ULID benefits

**My recommendation:** Option A or B. If `go-composable-business-types` is intended to be a real library with more than just IDs, publish it. If it's just a thin ULID wrapper, inline it.

---

## Module Dependency Graph

```
core/          (cockroachdb/errors, google/uuid, go-json-experiment/json,
                go-composable-business-types/id, oklog/ulid/v2; test-dep on memory)
  ↑
memory/        (core, cockroachdb/errors)
  ↑ (transitive only)
catalog/       (core, go-faster/yaml, go-json-experiment/json)
middleware/    (core only — cockroachdb/errors is indirect)
xtypes/       (core)
```

**Note:** `memory` is a transitive dep of catalog, middleware, and xtypes only because core/event tests import memory. In production code, only `core` is a direct dep.

---

## Session History

This session continues 4+ prior sessions:

- **Session 1:** Multi-module migration Phases 0–4 (20+ commits)
- **Session 2:** Post-migration cleanup (docs, CI, examples, Makefile)
- **Session 3:** Code quality improvements (7 commits)
- **Session 4:** ULID migration + further cleanup (14+ commits)
- **Session 5 (this one):** Verification, status report

Total commits across all sessions: ~30+
Total lines changed: ~2,000+
