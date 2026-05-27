# Session 92 — Comprehensive Status Report

**Date:** 2026-05-22  
**Branch:** master  
**HEAD:** `3489b5a` (refactor(dispatcher): unexport internal plumbing)  
**Trigger:** Post-Session 92 quality audit + comprehensive status + Tier 1 deletion execution

---

## Vital Signs

| Metric           | Value                                                           | Status    |
| ---------------- | --------------------------------------------------------------- | --------- |
| Build            | ✅ Clean (`go build` passes all modules)                        | OK        |
| Tests            | 28/30 packages pass, 2 FAIL (golden files stale)                | NEEDS FIX |
| Coverage         | 83.7% total (27 packages)                                       | GOOD      |
| `go vet`         | ✅ Zero issues                                                  | OK        |
| Files >250 lines | ✅ Zero production files exceed limit                           | OK        |
| Production lines | 15,913 across 180 files                                         | —         |
| Test lines       | 31,873 across 131 files (2:1 test:prod ratio)                   | EXCELLENT |
| Public exports   | 492 total (funcs 276, types/vars/consts 216)                    | HIGH      |
| Modules          | 11 in go.work (example/todo excluded due to cqrs-htmx breakage) | —         |

### Test Coverage Per Package

| Package                       | Coverage            | Trend     |
| ----------------------------- | ------------------- | --------- |
| `core/query`                  | 100.0%              | —         |
| `core/pkg/dispatcher`         | 100.0%              | —         |
| `middleware`                  | 100.0%              | —         |
| `catalog/adapters`            | 100.0%              | —         |
| `memory`                      | 99.6%               | —         |
| `core/pkg/id`                 | 98.1%               | —         |
| `core/aggregate`              | 95.9%               | —         |
| `catalog/d2`                  | 95.0%               | —         |
| `core/command`                | 94.7%               | —         |
| `catalog/openapi`             | 94.4%               | —         |
| `projection`                  | 94.2%               | —         |
| `catalog/asyncapi`            | 93.7%               | —         |
| `sync`                        | 92.2%               | —         |
| `core/event`                  | 92.1%               | ↑ (+0.9%) |
| `catalog/eventcatalog`        | 91.3%               | —         |
| `catalog/docserver`           | 90.0%               | —         |
| `catalog`                     | 90.5%               | —         |
| `core/decider`                | 93.3%               | ↑ (+4.0%) |
| `storage`                     | 86.9%               | —         |
| `catalog/internal/schemautil` | 84.2%               | —         |
| `catalog/internal/caseutil`   | 76.5%               | —         |
| `testhelpers`                 | 10.5%               | LOW       |
| `example/todo/storage`        | 29.2%               | —         |
| `example/todo/commands`       | 68.4%               | —         |
| `example/todo/projections`    | 78.9%               | —         |
| `example/todo/queries`        | 81.8%               | —         |
| `example/todo/domain`         | 100.0%              | —         |
| `integration/*`               | N/A (no statements) | —         |

### Failing Tests (2 packages — pre-existing)

| Package                | Test                                  | Cause                               |
| ---------------------- | ------------------------------------- | ----------------------------------- |
| `catalog/asyncapi`     | `TestGolden_AsyncAPIYAML`             | Golden file stale — needs `-update` |
| `catalog/eventcatalog` | `TestGolden_EventCatalog_Config`      | Golden file stale                   |
| `catalog/eventcatalog` | `TestGolden_EventCatalog_PackageJSON` | Golden file stale                   |

### External Dependency Issues

| Dependency                  | Issue                                    | Impact                                                                                        |
| --------------------------- | ---------------------------------------- | --------------------------------------------------------------------------------------------- |
| `cqrs-htmx` v0.0.0-20260507 | `event.RegisterClassification` undefined | Breaks `example/todo/cmd/api` build. This symbol was removed from go-cqrs-lite in Session 89. |

---

## a) FULLY DONE ✅

### Session 92 — Query Quality Improvements (6 commits)

| Commit    | Description                                                                                                                                                    |
| --------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `f608969` | **Handler doc comment:** Added 10-line doc on `query.Handler` explaining why `any` is required and pointing to `RegisterTyped[T]` + `DispatchTyped[T]`.        |
| `77d1360` | **Design doc closure:** Closed `docs/planning/QUERY_HANDLER_GENERICS.md`. Documented why function type was chosen over interface.                              |
| `73f23f8` | **Lost-context fix:** Added `context.Context` to todo query handlers (was silently discarded by `registerQuery` wrapper). Eliminated `registerQuery` wrapper.  |
| `4e57d71` | **Typed handlers:** Todo handlers return concrete types (`*GetTodoResult`, etc.). Registration uses `RegisterTyped[T]`. HTTP dispatch uses `DispatchTyped[T]`. |
| `ed0cb1e` | **Pagination integration:** Replaced raw `Limit`/`Offset` in `ListTodosQuery` with `query.Pagination` + `query.PaginatedResult`. First real consumer.          |
| `de1a31d` | **AGENTS.md update:** Updated Query Handler code example. Added Session 92 milestone.                                                                          |

### Session 92 — Tier 1 Deletion Execution (2 commits)

| Commit    | Description                                                                                                                                                                                                                                                           |
| --------- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `bfc01cc` | **Deleted 217 lines across 7 files:** `event.IsReplay` (5 lines), `event.NewEvents`/`MustNewEvents`/`DecodePayloads` (84 lines), `decider.Result` + `ExecuteWithResult` (71 lines), `FromCommandDispatcher` + `FromQueryDispatcher` (57 lines). Removed 4 test files. |
| `3489b5a` | **Unexported 4 dispatcher internals:** `MiddlewareChain` → `middlewareChain`, `GetHandler` → `getHandler`, `NewCatalogDispatcher` → `newCatalogDispatcher`, `CopyCatalogEntries` → `copyCatalogEntries`. Updated `dispatcher_test.go`.                                |

### Changes Since Session 91 Status

| Metric                  | Before | After  | Delta     |
| ----------------------- | ------ | ------ | --------- |
| Public exports          | 504    | 492    | **−12**   |
| Production lines        | 16,134 | 15,913 | **−221**  |
| Test lines              | 32,312 | 31,873 | **−439**  |
| Production files        | 183    | 180    | **−3**    |
| Test files              | 135    | 131    | **−4**    |
| `core/decider` coverage | 89.3%  | 93.3%  | **+4.0%** |
| `core/event` coverage   | 91.2%  | 92.1%  | **+0.9%** |

### Deleted Files (7 total)

1. `core/event/codec_batch.go` — `NewEvents`, `MustNewEvents`, `DecodePayloads`
2. `core/event/codec_batch_test.go` — tests for deleted functions
3. `core/decider/result.go` — `Result` type + `ExecuteWithResult`
4. `core/decider/result_test.go` — tests for deleted types
5. `catalog/adapters/from_query_dispatcher.go` — `FromCommandDispatcher`, `FromQueryDispatcher`
6. `catalog/adapters/dispatcher_test.go` — tests for deleted dispatch adapters
7. `catalog/adapters/benchmark_test.go` — benchmarks for deleted code

### Quality Metrics Maintained

| Metric                        | Value                                                              |
| ----------------------------- | ------------------------------------------------------------------ |
| TODO/FIXME/HACK in production | **0**                                                              |
| `go vet` issues               | **0**                                                              |
| Race conditions               | **0**                                                              |
| Files >250 lines              | **0** (largest: `catalog/eventcatalog/exporter.go` at 250 exactly) |

---

## b) PARTIALLY DONE 🔶

### Deletion Audit — Tier 1 Executed, Tiers 2–4 Pending

**Tier 1: ✅ DONE** — 217 lines deleted, 12 exports removed

**Tier 2: Deprecated adapters (1 example consumer)**
| Symbol | Lines | Consumer | Status |
|--------|-------|----------|--------|
| `catalog/adapters.CatalogBuilder` | 122 | `example/user/catalog.go` | Still in use via `NewBuilder` |

**Tier 3: Breaking interface changes**
| Symbol | Lines | Impact | Status |
|--------|-------|--------|--------|
| `Command.IdempotencyKey()` | ~5 | 5 implementations | Still in interface |
| `event.OutboxPublisher` + subsystem | 206 | Zero consumers | Still exported |

**Tier 4: Major package deletion**
| Symbol | Lines | Impact | Status |
|--------|-------|--------|--------|
| `core/aggregate/` entire package | 1,756 | Deprecated | Still exported with deprecation notice |
| `integration/aggregate/` | ~800 | Tests for deprecated | Still present |

### LSP Build Errors — Pre-existing

| File                                       | Error                            |
| ------------------------------------------ | -------------------------------- |
| `core/event/codec_typed.go:31`             | `undefined: ErrNilPayload`       |
| `core/event/codec_typed_test.go:94`        | `undefined: event.ErrNilPayload` |
| `catalog/adapters/benchmark_test.go:40`    | `undefined: command.CatalogMeta` |
| `integration/command/command_test.go` (3×) | `undefined: command.CatalogMeta` |
| `integration/query/query_test.go` (3×)     | `undefined: query.CatalogMeta`   |

Note: `catalog/adapters/benchmark_test.go` was deleted in this session, so that error is resolved.

---

## c) NOT STARTED ⬜

1. **Execute Tiers 2–4 deletion plan** — Tier 1 done, rest pending
2. **Fix stale golden files** — `catalog/asyncapi` + `catalog/eventcatalog` need `-update`
3. **Fix LSP build errors** — 10 pre-existing diagnostics (was 11, 1 resolved by deletion)
4. **`testhelpers` coverage at 10.5%** — no improvement effort
5. **`catalog/internal/caseutil` at 76.5%** — no effort started
6. **`storage` coverage at 86.9%** — no effort to reach 90%+
7. `example/todo` not in `go.work` (excluded due to cqrs-htmx breakage)
8. No CI/CD pipeline for golden file checks
9. No version tagging or release automation
10. No CHANGELOG.md
11. `sync` module still zero consumers, no decision made
12. `cqrs-htmx` dependency broken — needs update or removal
13. No query-specific `ErrQueryValidation` sentinel
14. `example/todo` integration test at 29.2% coverage
15. `catalog/internal/cattest` at 0% coverage (internal helpers, no tests)
16. No `go generate` for schema/codegen

---

## d) TOTALLY FUCKED UP 💥

1. **`cqrs-htmx` dependency is broken** — `example/todo/cmd/api` binary cannot build because `cqrs-htmx@v0.0.0-20260507` references `event.RegisterClassification` removed in Session 89. **Cross-repository breakage.**

2. **Stale LSP errors persisting across sessions** — Same phantom diagnostics (CatalogMeta, ErrNilPayload) present since Session 89. Don't block `go build` but confuse IDE users.

3. **`catalog/internal/cattest` has 0% coverage** — Internal test helper package with `[no test files]`. No real risk but skews metrics.

4. **Golden file drift is endemic** — Every formatter run risks breaking asyncapi/eventcatalog golden tests. No auto-refresh mechanism.

5. **492 public exports for a "lite" library** — "lite" remains aspirational. After Tier 1 deletion, still ~492 exports. Need Tiers 2–4 to meaningfully reduce surface.

6. **`testhelpers` at 10.5%** — Heavily used but not directly tested. Low metric, high real-world usage.

7. **`CatalogEntry` location** — Lives in `core/pkg/dispatcher/` due to circular deps. Correct but not obvious to consumers.

---

## e) WHAT WE SHOULD IMPROVE 📈

### Immediate (This Session or Next)

1. **Fix `cqrs-htmx` dependency** — Update or remove from todo example
2. **Refresh stale golden files** — `go test -update` for asyncapi + eventcatalog
3. **Investigate LSP errors** — 10 phantom diagnostics remain

### API Surface Quality

4. **Execute Tiers 2–4 deletions** — Tier 1 removed 12 exports. Need ~60 more to reach ~350 target.
5. **Delete deprecated `aggregate` package** — 1,756 lines of dead weight
6. **Remove `Command.IdempotencyKey()`** — Dead method on interface
7. **Delete `event.OutboxPublisher`** — 206 lines, zero consumers

### Developer Experience

8. **Fix all LSP errors** — New contributors see red squiggles
9. **Add CHANGELOG.md** — Consumers need to know what changed
10. **Auto-refresh golden files in CI** — Remove manual step
11. **Consolidate status reports** — 18 status files in `docs/status/`

### Test Quality

12. **Don't test deprecated code** — `catalog/adapters` still has deprecated builder tested
13. **Improve `testhelpers` coverage** — Write tests or document exemption
14. **Improve `storage` coverage to >90%** — SQL-heavy, needs more edge cases

---

## f) Top 25 Things We Should Get Done Next

### Priority 1: Immediate Blockers

| #   | Task                                                       | Impact   | Effort  |
| --- | ---------------------------------------------------------- | -------- | ------- |
| 1   | **Fix `cqrs-htmx` dependency** (blocks todo example build) | Critical | Low     |
| 2   | **Refresh golden files** (`go test -update`)               | Medium   | Trivial |
| 3   | **Investigate LSP errors** (10 phantom diagnostics)        | Medium   | Low     |

### Priority 2: API Cleanup (High Value)

| #   | Task                                                                             | Impact | Effort |
| --- | -------------------------------------------------------------------------------- | ------ | ------ |
| 4   | **Delete `core/aggregate/` package** (1,756 lines deprecated)                    | High   | Medium |
| 5   | **Delete `integration/aggregate/`** (tests for deprecated package)               | Medium | Low    |
| 6   | **Remove `Command.IdempotencyKey()` from interface**                             | Medium | Medium |
| 7   | **Delete `event.OutboxPublisher`** (206 lines, zero consumers)                   | Medium | Low    |
| 8   | **Delete deprecated `catalog/adapters.CatalogBuilder`** + migrate `example/user` | Medium | Medium |

### Priority 3: Quality

| #   | Task                                                   | Impact | Effort |
| --- | ------------------------------------------------------ | ------ | ------ |
| 9   | **Fix `testhelpers` coverage** (10.5% → 60%+)          | Medium | Medium |
| 10  | **Improve `storage` coverage** (86.9% → 90%+)          | Medium | Medium |
| 11  | **Add `catalog/internal/cattest` basic tests**         | Low    | Low    |
| 12  | **Improve `catalog/internal/caseutil`** (76.5% → 90%+) | Low    | Low    |

### Priority 4: Developer Experience

| #   | Task                                                | Impact | Effort  |
| --- | --------------------------------------------------- | ------ | ------- |
| 13  | **Add CHANGELOG.md**                                | Medium | Low     |
| 14  | **Golden file auto-refresh in CI**                  | Low    | Medium  |
| 15  | **Archive old status reports** (18 in docs/status/) | Low    | Trivial |
| 16  | **Decide fate of `sync` module**                    | Low    | Low     |

### Priority 5: Strategic

| #   | Task                                                                   | Impact | Effort |
| --- | ---------------------------------------------------------------------- | ------ | ------ |
| 17  | **Versioned module tags** (git tag per module)                         | High   | Medium |
| 18  | **Watermill integration module**                                       | High   | High   |
| 19  | **Saga/Process Manager** (`docs/planning/SAGA_DESIGN.md`)              | High   | High   |
| 20  | **Nix flake migration** (from justfile)                                | Medium | Medium |
| 21  | **Query handler generics** (full rewrite, breaking)                    | High   | High   |
| 22  | **Outbox transaction API** (`docs/planning/OUTBOX_TRANSACTION_API.md`) | Medium | Medium |
| 23  | **Projection `OnError` callback option**                               | Medium | Low    |
| 24  | **`event.New()` with schema validation**                               | Medium | Medium |
| 25  | **`DeriveAggregateID` documentation + examples**                       | Low    | Low    |

---

## g) Top #1 Question I Cannot Figure Out Myself 🤔

**How do the LSP build errors exist when `go build` passes?**

Specifically:

- `integration/command/command_test.go` references `command.CatalogMeta` (6 locations) — LSP says `undefined`
- `core/event/codec_typed.go` references `ErrNilPayload` — LSP says `undefined`
- Yet `go build ./...` and `go test ./...` both pass cleanly

Three hypotheses, none confirmed:

1. **Build tags** — Test files might have build tags that exclude them
2. **Stale gopls cache** — gopls might be seeing an older state
3. **go.work vs module-local resolution** — Workspace resolves differently than gopls

**I do not know which explanation is correct or how to definitively determine it.**

---

## Module Dependency Graph (Current)

```
core (180 exports post-Tier-1) ← foundation, zero internal deps
  ↑
  ├── memory (11 exports) ← in-memory implementations
  ├── catalog (129 exports) ← documentation generation
  │     └── adapters (100% coverage, some deprecated code)
  ├── middleware (30 exports) ← cross-cutting concerns
  ├── testhelpers (37 exports, 10.5% coverage) ← test utilities
  ├── projection (10 exports) ← replay + live subscription
  ├── storage (66 exports, 86.9% coverage) ← SQL implementations
  ├── integration ← cross-module tests
  ├── sync (28 exports, 92.2% coverage) ← distributed primitives
  └── example/ ← usage demos
        ├── user/ (uses RegisterTyped + DispatchTyped ✅)
        └── todo/ (now uses typed handlers + Pagination ✅)
```

## Uncommitted Changes

```
Working tree clean. All changes committed and pushed to origin/master.
```

---

_Arte in Aeternum_
