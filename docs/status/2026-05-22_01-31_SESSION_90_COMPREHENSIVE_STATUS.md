# Session 90 — Comprehensive Status Report

**Date:** 2026-05-22 01:31
**Branch:** master
**Last commit:** `2f518c1` docs(status): add Session 90 comprehensive status report
**Test Suite:** 27/27 packages PASS, 0 races
**Total Coverage:** 83.7%
**Production Files:** 183 | **Test Files:** 135

---

## A. FULLY DONE ✅

### Session 89 Committed Work (Phases 1–5)

| Phase | Description                                                  | Commit               |
| ----- | ------------------------------------------------------------ | -------------------- |
| 1     | CatalogEntry centralization in `core/pkg/dispatcher`         | `1088fcd`            |
| 2     | CatalogEntry migration in all test files                     | `333a6af`            |
| 3     | `event.New()` typed constructor (struct→[]byte auto-marshal) | `1088fcd`            |
| 4     | `ExecuteWithResult` → typed `Result` in decider              | `4c79338`            |
| 5     | `DeriveAggregateID` + `WithOwnership` for SQLEventStore      | `4c79338`, `e773be7` |

### Session 89 Committed Work (Phases 6, 7, 9)

| Phase | Description                                                   | Commit    |
| ----- | ------------------------------------------------------------- | --------- |
| 6     | `IdempotencyKey()` deprecation comment on `Command` interface | `6ac0a77` |
| 7     | Projection `Builder` + `On[T]()` fluent API                   | `6ac0a77` |
| 9     | `WithReplay`/`IsReplay` context helpers + Runner wiring       | `6ac0a77` |

### Session 90 Completed This Session

| Item                          | Description                                                                                                                               |
| ----------------------------- | ----------------------------------------------------------------------------------------------------------------------------------------- |
| `projection/builder_test.go`  | 6 new tests: typed handler, multi-event, nil handler, empty builder, unregistered type filter, invalid payload                            |
| `core/aggregate/aggregate.go` | Package-level `Deprecated` doc comment pointing to `core/decider`                                                                         |
| `AGENTS.md` update            | Added `event.New`, `ExecuteWithResult`, `DeriveAggregateID`, `IsReplay`, `WithReplay`, `Builder`, `On[T]()` to docs; Session 90 milestone |
| Golden test refresh           | `catalog/asyncapi` + `catalog/eventcatalog` golden files updated                                                                          |
| LSP stale diagnostics         | Confirmed 8 LSP errors are stale (gopls cache), not real — `go build` + `go vet` pass clean                                               |

### Pre-existing Quality Metrics (Maintained)

| Metric                   | Value                                                          |
| ------------------------ | -------------------------------------------------------------- |
| Files > 250 lines        | **0** (was 1, `testhelpers/fake_store.go` split in Session 89) |
| TODO/FIXME/HACK comments | **0**                                                          |
| `go vet` issues          | **0**                                                          |
| Race conditions          | **0**                                                          |
| Build errors             | **0**                                                          |

---

## B. PARTIALLY DONE ⚠️

### 1. Deprecation Cleanup (Phase 6)

**Status:** Comments added, APIs still fully functional.

| Deprecated API                   | Comment Added | Callers Updated                            | Removed |
| -------------------------------- | ------------- | ------------------------------------------ | ------- |
| `core/aggregate` package         | ✅            | ❌ (7 integration test files still import) | ❌      |
| `Command.IdempotencyKey()`       | ✅            | ❌ (5 implementations still exist)         | ❌      |
| `adapters.CatalogBuilder`        | ✅            | ❌ (example/user + 4 test files)           | ❌      |
| `adapters.FromCommandDispatcher` | ✅            | ❌ (2 test files)                          | ❌      |
| `adapters.FromQueryDispatcher`   | ✅            | ❌ (2 test files)                          | ❌      |
| `catalog.MessageIDString()`      | ✅            | ❌ (1 test file)                           | ❌      |

**Gap:** Deprecation comments are purely documentation — no `-deprecated` build tags, no compiler warnings. Callers still compile and run fine. This is acceptable for a library (backward compat) but the dead deprecated code should be removed in a future major version.

### 2. `sync` Module — Orphaned

- Has its own `go.mod`, listed in `go.work`
- **Zero consumers** in the entire workspace
- No tests discovered that exercise it from any other module
- Coverage: 92.2% but entirely self-contained
- **Decision needed:** Keep as preview, document as experimental, or remove from workspace?

---

## C. NOT STARTED ❌

### From Session 89 Plan (Skipped/Lower Priority)

| Phase | Description                                           | Priority | Notes                                                          |
| ----- | ----------------------------------------------------- | -------- | -------------------------------------------------------------- |
| 8     | Command interface improvements (remove `any` returns) | Medium   | `query.Handler` returns `any` — violates project "no any" rule |
| —     | Integration test migration (`aggregate` → `decider`)  | Medium   | 7 files still use deprecated `core/aggregate`                  |
| —     | `example/user` catalog modernization                  | High     | Uses deprecated `adapters.CatalogBuilder`                      |
| —     | `testhelpers` coverage improvement (10.5%)            | Low      | Test helpers — low risk                                        |
| —     | `storage` coverage improvement (86.9%)                | Low      | SQL-heavy, needs more mocks                                    |

### Never-Started Features (From Planning Docs)

| Feature                | Doc Reference                             | Status          |
| ---------------------- | ----------------------------------------- | --------------- |
| Saga/Process Manager   | `docs/planning/SAGA_DESIGN.md`            | Design doc only |
| Query handler generics | `docs/planning/QUERY_HANDLER_GENERICS.md` | Design doc only |
| Outbox transaction API | `docs/planning/OUTBOX_TRANSACTION_API.md` | Design doc only |
| Watermill integration  | AGENTS.md "planned"                       | Not started     |
| Nix flake migration    | Skill exists, no plan                     | Not started     |

---

## D. TOTALLY FUCKED UP 💥

### 1. Stale LSP (gopls) — False Errors

**Impact:** Every session starts with 6-8 phantom errors from gopls cache.

Files showing errors that don't actually have issues:

- `catalog/adapters/dispatcher_test.go` — "DuplicateDecl" and "WrongArgCount" (compiles fine)
- `catalog/adapters/export_test.go` — "DuplicateDecl" (compiles fine)
- `catalog/adapters/from_query_dispatcher.go` — "MissingFieldOrMethod CatalogEntries" (compiles fine)

**Root cause:** gopls doesn't understand type aliases well (`command.Dispatcher` = `CatalogDispatcher[string, Handler]` which has `CatalogEntries()`). Only `go build` + `go test` are reliable.

**Mitigation:** Restarting gopls sometimes helps. Always verify with `go build` before trusting LSP.

### 2. File Write Tool Reliability (Session 89)

The `write` tool silently failed multiple times in Session 89 — reported "File successfully written" but files didn't exist on disk. Had to fall back to `bash` `tee` for reliable file creation. This session the tool worked correctly.

### 3. Golden Test Drift

Golden files in `catalog/asyncapi` and `catalog/eventcatalog` drift every time the formatter runs or the catalog types change. They require manual `-update` refresh. Not a bug per se, but a friction point.

---

## E. WHAT WE SHOULD IMPROVE 🔧

### Code Quality

1. **`query.Handler` returns `any`** — Violates "no any" rule. `DispatchTyped[T]` is a workaround, not a fix. The design doc exists at `docs/planning/QUERY_HANDLER_GENERICS.md`.

2. **80 `//nolint` directives** — Mostly justified (`exhaustruct`, `wrapcheck`), but should be reviewed periodically. Some may be suppressable by fixing the underlying issue.

3. **`catalog/internal/cattest` has 0% coverage** — This is an internal test helper package with `[no test files]`. Not a real risk but the 0% skews metrics.

4. **`testhelpers` at 10.5% coverage** — Test utilities that are exercised indirectly through integration tests. Direct coverage is low but real usage is high.

5. **`storage` at 86.9%** — SQL-heavy code that's hard to test without real databases. `go-sqlmock` helps but doesn't cover edge cases well.

### Architecture

6. **`CatalogMeta` duplicated across 2 packages** — `command.CatalogMeta` and `query.CatalogMeta` are nearly identical. Should be consolidated to `dispatcher.CatalogEntry` (which already exists).

7. **`Root.LoadEvents` vs `Core.LoadFromHistory` mismatch** — Every aggregate must implement `LoadEvents` and delegate to `LoadFromHistory`. This is fragile and error-prone for consumers.

8. **`MemoryBus.Publish` holds RLock during handler execution** — Subscribers block publishers. Acceptable for test utility but documented as a known limitation.

### Developer Experience

9. **No `go generate` or code generation** — Schema generation is reflection-based at runtime. Compile-time safety via generics exists (`SchemaFromType[T]`) but the builder pattern could be more discoverable.

10. **Example app (`example/user`) uses deprecated APIs** — The example should showcase best practices, not deprecated patterns.

---

## F. TOP 25 THINGS TO DO NEXT 🎯

Sorted by impact (Pareto principle):

### Tier 1: Ship Blockers (Do First)

| #   | Task                                                                           | Impact                                              | Effort |
| --- | ------------------------------------------------------------------------------ | --------------------------------------------------- | ------ |
| 1   | **Fix `example/user` to use `event.New()` + new catalog API**                  | High — example is the first thing consumers see     | Low    |
| 2   | **Migrate `example/user/catalog.go` off deprecated `adapters.CatalogBuilder`** | High — example teaches wrong patterns               | Low    |
| 3   | **Remove `IdempotencyKey()` from `Command` interface**                         | Medium — deprecated but still in interface contract | Medium |
| 4   | **Consolidate `CatalogMeta` → `dispatcher.CatalogEntry`**                      | Medium — eliminates split-brain type duplication    | Medium |

### Tier 2: Quality Improvements (High Value)

| #   | Task                                                       | Impact                              | Effort                 |
| --- | ---------------------------------------------------------- | ----------------------------------- | ---------------------- |
| 5   | **Add `event.New()` examples to README**                   | High — discoverability              | Low                    |
| 6   | **Write `projection.Builder` usage example**               | High — new API needs docs           | Low                    |
| 7   | **Add `IsReplay` usage example**                           | Medium — new API needs docs         | Low                    |
| 8   | **Migrate integration/aggregate tests to decider pattern** | Medium — tests use deprecated API   | Medium                 |
| 9   | **Improve `storage` coverage to >90%**                     | Medium — lowest production coverage | Medium                 |
| 10  | **Fix `query.Handler` `any` return type**                  | Medium — violates project rule      | High (breaking change) |

### Tier 3: New Features (Consumer Value)

| #   | Task                                                       | Impact                                      | Effort |
| --- | ---------------------------------------------------------- | ------------------------------------------- | ------ |
| 11  | **Projection `OnError` callback option**                   | Medium — consumers need error observability | Low    |
| 12  | **`event.New()` with schema validation**                   | Medium — type-safe payload construction     | Medium |
| 13  | **`decider.ExecuteWithResult` example**                    | Medium — new API needs real-world demo      | Low    |
| 14  | **Builder pattern for `event.Projection` with middleware** | Medium — composable projection handlers     | Medium |
| 15  | **`DeriveAggregateID` documentation + examples**           | Low — new API, needs discoverability        | Low    |

### Tier 4: Cleanup & Hygiene

| #   | Task                                                                | Impact                                 | Effort |
| --- | ------------------------------------------------------------------- | -------------------------------------- | ------ |
| 16  | **Remove deprecated `FromCommandDispatcher`/`FromQueryDispatcher`** | Low — dead code path                   | Low    |
| 17  | **Remove deprecated `catalog.MessageIDString()`**                   | Low — dead code                        | Low    |
| 18  | **Add `//nolint` expiry comments (revisit by date)**                | Low — maintenance discipline           | Low    |
| 19  | **Add `catalog/internal/cattest` basic tests**                      | Low — metric improvement               | Low    |
| 20  | **Improve `testhelpers` coverage >50%**                             | Low — indirect coverage already exists | Medium |

### Tier 5: Strategic (Longer Term)

| #   | Task                                                    | Impact                             | Effort |
| --- | ------------------------------------------------------- | ---------------------------------- | ------ |
| 21  | **Decide fate of `sync` module** (keep/remove/document) | Low — zero consumers               | Low    |
| 22  | **Golden test auto-refresh in CI**                      | Low — removes manual step          | Medium |
| 23  | **Versioned module tags (git tag per module)**          | High — consumers need version pins | High   |
| 24  | **Watermill integration module**                        | High — real message broker support | High   |
| 25  | **Nix flake migration** (from justfile remnants)        | Medium — build reproducibility     | Medium |

---

## G. TOP #1 QUESTION 🤔

**Should the `sync` module stay in `go.work` or be removed/isolated?**

The `sync` module has:

- Its own `go.mod` ✅
- Zero consumers in the workspace ❌
- 92.2% self-coverage ✅
- No documentation explaining its purpose or intended consumers ❌

Options:

1. **Keep as-is** — It's an experimental/preview module. Consumers can import independently.
2. **Remove from `go.work`** — Keeps it in repo but removes from workspace builds. Consumers must `GOWORK=off` to use.
3. **Document as experimental** — Add README to `sync/` explaining its purpose and stability level.
4. **Remove entirely** — YAGNI. No consumers, no plan for consumers.

---

## Coverage by Package

| Package                       | Coverage  | Change vs Session 89 |
| ----------------------------- | --------- | -------------------- |
| `core/query`                  | 100.0%    | —                    |
| `core/pkg/dispatcher`         | 100.0%    | —                    |
| `middleware`                  | 100.0%    | —                    |
| `catalog/adapters`            | 100.0%    | —                    |
| `memory`                      | 99.6%     | —                    |
| `core/pkg/id`                 | 98.1%     | +0.3%                |
| `core/aggregate`              | 95.9%     | —                    |
| `catalog/d2`                  | 95.0%     | —                    |
| `core/command`                | 94.7%     | —                    |
| `projection`                  | 94.2%     | +0.3%                |
| `catalog/asyncapi`            | 93.7%     | —                    |
| `core/decider`                | 89.3%     | —                    |
| `catalog/eventcatalog`        | 91.3%     | —                    |
| `catalog/docserver`           | 90.0%     | -1.0%                |
| `catalog`                     | 90.5%     | —                    |
| `core/event`                  | 91.2%     | -0.9%                |
| `catalog/openapi`             | 94.4%     | —                    |
| `sync`                        | 92.2%     | —                    |
| `storage`                     | 86.9%     | -1.2%                |
| `catalog/internal/schemautil` | 84.2%     | —                    |
| `catalog/internal/caseutil`   | 76.5%     | —                    |
| `testhelpers`                 | 10.5%     | —                    |
| **Total**                     | **83.7%** | —                    |

---

## File Inventory

| Metric                    | Count                                                                  |
| ------------------------- | ---------------------------------------------------------------------- |
| Production Go files       | 183                                                                    |
| Test Go files             | 135                                                                    |
| Total Go files            | 318                                                                    |
| Largest production file   | `catalog/eventcatalog/exporter.go` (250 lines, exactly at limit)       |
| Files > 250 lines         | **0** ✅                                                               |
| Modules in go.work        | 12                                                                     |
| Dependencies (production) | 4 (`oklog/ulid`, `go-branded-id`, `go-error-family`, `go-faster/yaml`) |
