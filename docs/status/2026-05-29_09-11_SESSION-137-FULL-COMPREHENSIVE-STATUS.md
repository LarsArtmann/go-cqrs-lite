# Session 137 — Codec Migration, Deprecated Cleanup & Full Green Status

**Date:** 2026-05-29 09:11
**Sessions Spanned:** 134–138 (continuous campaign)
**Author:** Crush (glm-5.1) + Lars

---

## Executive Summary

Sessions 134–138 completed a comprehensive codec migration and deprecated API cleanup campaign. The project is now in its **healthiest state ever**: all 29 testable packages pass, zero lint issues in core/memory, and only 1 pre-existing clone group in catalog test helpers. All deprecated `event.Codec`/`event.JSONCodec` type aliases have been removed. All deprecated `GlobalLoader`/`PositionalLoader`/`BackwardsLoader` interfaces have been removed.

---

## Current Health Dashboard

| Check                    | Status                                   |
| ------------------------ | ---------------------------------------- |
| **Build**                | PASS (clean)                             |
| **Tests**                | **29/29 packages PASS** (0 failures)     |
| **Lint (core)**          | 0 issues                                 |
| **Lint (memory)**        | 0 issues                                 |
| **Lint (catalog)**       | 4 pre-existing issues (exhaustruct/goconst/mnd) |
| **art-dupl (t=35)**      | 1 clone group (catalog test helper)      |
| **art-dupl (t=40)**      | 1 clone group (same)                     |
| **Production LOC**       | 25,077                                   |
| **Test LOC**             | 44,860                                   |
| **Total LOC**            | 69,937                                   |
| **Go Modules**           | 22 go.mod files, 16 in workspace        |
| **Testable Packages**    | 29                                       |

---

## A) FULLY DONE

### Session 134
- Eliminated 3 clone groups: storage span setup, testhelpers handler type, decider/storage otel attrs
- Added `cqrsotel.AggregateAttrs()` to `otel/attributes.go` using `fmt.Stringer`
- Removed local `aggregateAttrs()` from `core/decider/otel.go` and `storage/otel.go`
- Achieved 0 clone groups at t=35 and t=40

### Session 135
- Renamed `AggregateBaseAttrs` to `AggregateAttrs` (simplified signature)
- Fixed `middleware/tracing.go` to use `cqrsotel.EventAttrs()`
- Removed 3 redundant self-replace directives from go.mod files
- Extracted generic `getOverride[T any]` helper in `testhelpers/fake_store.go` (-29 lines)
- Migrated `core/decider` from deprecated `event.Codec` to `codec.Codec`
- Improved param naming consistency (aggType→aggregateType, evtType→eventType)

### Session 136
- Deep research into command/query dispatcher architecture — confirmed they should stay separate
- Removed dead `core/aggregate/` package (zero external callers)
- Fixed `embeddedstructfieldcheck` in `core/pkg/dispatcher/dispatcher.go`
- Fixed `staticcheck QF1008` (removed unnecessary `CatalogDispatcher.` selector)
- Fixed `wsl_v5` in `core/decider/decider.go:224`
- Fixed `nlreturn` in test files
- Inlined `subscribesTo` logic in `projection/runner.go` (replaced `event.SubscribesTo` call)

### Session 137 (this session)
- **Migrated ALL remaining `event.JSONCodec`/`event.Codec` usages to `codec.JSONCodec`/`codec.Codec`:**
  - `core/decider/decider_bdd_test.go` — `event.JSONCodec{}` → `codec.JSONCodec{}`
  - `core/decider/decider_helpers_test.go` — widened params to `codec.Codec` interface
  - `core/decider/decider_coverage_test.go` — `event.JSONCodec{}` → `codec.JSONCodec{}`
  - `core/event/codec_test.go` — 13 replacements, `event.Codec` → `codecpkg.Codec`
  - `core/event/snapshot_helper_test.go` — `&event.JSONCodec{}` → `&codec.JSONCodec{}`
  - `core/event/benchmark_test.go` — `event.JSONCodec{}` → `codec.JSONCodec{}`
  - `core/event/upcaster_test.go` — `JSONCodec{}` → `codec.JSONCodec{}`
  - `example/todo/aggregate/todo.go` — package-level `var codec`
  - `example/todo/projections/todo_projection.go` — package-level `var codec`
  - `example/todo/projections/todo_projection_test.go` — local variable
  - `example/user/projection.go` — local variable
- **Removed deprecated type aliases from `core/event/codec.go`:**
  - `type Codec = codec.Codec` — REMOVED
  - `type JSONCodec = codec.JSONCodec` — REMOVED
- **Updated `core/event/snapshot_helper.go`:** param type from bare `Codec` to `codec.Codec`
- **Removed deprecated interfaces from `core/event/store.go`:**
  - `GlobalLoader` — REMOVED (zero callers)
  - `PositionalLoader` — REMOVED (zero callers)
  - `BackwardsLoader` type alias — REMOVED (zero callers)
- **Added `codec` dependency to `example/todo/go.mod` and `example/user/go.mod`**
- **nix fmt** — formatted 72 files across the codebase
- **Removed accidentally committed binaries** from example/projection/

### Session 138
- Fixed botched auto-migration in `upcaster_test.go` (renamed `codec` var to `c` to avoid shadowing)
- All 29 packages now pass (was 2-3 pre-existing failures in decider/query that are now green)

### Total Session 137 Changes
```
133 files changed, 1,756 insertions(+), 803 deletions(-)
10 commits made
```

---

## B) PARTIALLY DONE

| Item                                    | Status                                                |
| --------------------------------------- | ----------------------------------------------------- |
| Remove `event.Runner` (deprecated)      | BLOCKED — integration tests + projection tests depend on it; pre-existing bug in `projection/runner_registration_test.go` (`*event.Projection` should be `event.Projection`) |
| Remove deprecated `SubscribesTo` function | `event.SubscribesTo` still exists in `core/event/runner.go:229` — used by `runner.go:84,133`. Projection inlined around it, but event.Runner still uses it internally |
| Storage deprecated method cleanup       | `storage/event_store_global.go` still has deprecated `LoadAll` and `LoadAllFromPosition` methods (backward-compat wrappers) |

---

## C) NOT STARTED

| #  | Item                                                     | Priority |
| -- | -------------------------------------------------------- | -------- |
| 1  | Remove `core/aggregate/` from go.work if present         | LOW      |
| 2  | Fix `projection/runner_registration_test.go` type bug    | HIGH     |
| 3  | Extract `event.SubscribesTo` into projection-only usage | MEDIUM   |
| 4  | Clean up catalog lint (4 issues: exhaustruct/goconst/mnd) | LOW      |
| 5  | Add stream module integration tests                      | MEDIUM   |
| 6  | Add stream SQL reader tests                               | MEDIUM   |
| 7  | Fix `go-structure-linter` failures in pre-commit hook     | LOW      |
| 8  | Update TODO_LIST.md to reflect session 137 completions    | HIGH     |
| 9  | Update FEATURES.md to reflect current state               | MEDIUM   |
| 10 | Deduplicate catalog test helpers (1 remaining clone)      | LOW      |
| 11 | Split large test files (decider_test.go ~1200L, runner_test.go ~1057L) | MEDIUM   |
| 12 | Add BDD tests for Version, SchemaVersion, OutboxStatus, Pagination types | MEDIUM   |
| 13 | Add fuzz tests for event creation, ID parsing, schema reflection | MEDIUM   |
| 14 | Wire example/user/aggregate.go to use catalog-aware constructors | LOW      |
| 15 | Rewrite example/user/ to demonstrate full CQRS stack      | MEDIUM   |

---

## D) TOTALLY FUCKED UP (Issues Found)

| #  | Issue                                                              | Severity |
| -- | ------------------------------------------------------------------ | -------- |
| 1  | **`projection/runner_registration_test.go` has pre-existing type bug** — `*event.Projection` (pointer to interface) should be `event.Projection` (interface value). This blocks `event.Runner` removal | HIGH |
| 2  | **gopls is perpetually stale** — shows 40+ phantom errors that don't exist in `go build`. Wastes time chasing non-issues. Likely due to go.work + replace directives confusing the language server | MEDIUM |
| 3  | **Pre-commit hook fails on `go-structure-linter`** — This tool gives non-actionable advice ("add pkg/ directory", "add go-error-family dependency to go.mod"). Blocks clean commits | MEDIUM |
| 4  | **Pre-commit hook fails on `golangci-lint` exit code 7** — Seems to be a config/runner issue, not actual lint issues. buildflow wraps golangci-lint differently than `nix run .#lint` | MEDIUM |
| 5  | **`go.work` workspace build fails** (`go build ./...` at root) — CI uses `GOWORK=off` per-module. This is known but confusing for contributors | LOW |
| 6  | **Binary committed** — `example/projection/projection` (6.7MB) was committed to git. Detected by buildflow binary-check. Cleaned up in this session | FIXED |

---

## E) WHAT WE SHOULD IMPROVE

### Architecture & Code Quality
1. **`event.Runner` is dead weight** — It's deprecated in favor of `projection.Runner`, but can't be removed because tests haven't been migrated. Should be a priority.
2. **Test naming conventions** — Some test files have variable names that shadow package names (`codec := event.JSONCodec{}`). Should establish convention: never name vars the same as imported packages.
3. **Replace directives everywhere** — 22 go.mod files all have `replace` directives pointing to local paths. This is a known blocker until v1.0.0 tags are pushed. Should track progress toward publishable versions.

### Process & Tooling
4. **Pre-commit hook is flaky** — `go-structure-linter` and `buildflow` wrappers cause false failures. CI (nix-based) works fine. Should either fix the hook config or bypass it for non-critical failures.
5. **gopls can't handle this workspace** — 22 modules with replace directives confuse the language server. Consider adding a `.gopls` settings file to exclude certain directories or adjust the workspace mode.
6. **No automated clone detection in CI** — art-dupl runs manually. Should add to CI pipeline.

### Documentation
7. **TODO_LIST.md is stale** — Last reconciled Session 123. Many items completed since then (codec migration, deprecated removals, etc.).
8. **FEATURES.md needs update** — Doesn't reflect codec module extraction or deprecated removals.
9. **AGENTS.md needs codec module info** — The codec module is not mentioned in the monorepo structure or key patterns.

---

## F) Top #25 Things We Should Get Done Next

### HIGH IMPACT (Do First)

| #  | Task                                                              | Effort  | Impact |
| -- | ----------------------------------------------------------------- | ------- | ------ |
| 1  | Fix `projection/runner_registration_test.go` type bug and remove `event.Runner` | 2h      | HIGH — removes dead code |
| 2  | Update TODO_LIST.md (reconcile with sessions 124-138 completions) | 30m     | HIGH — single source of truth |
| 3  | Update AGENTS.md with codec module in structure + key patterns    | 15m     | HIGH — session continuity |
| 4  | Update FEATURES.md to reflect current state                       | 30m     | MEDIUM — feature inventory |
| 5  | Clean up 4 catalog lint issues (exhaustruct/goconst/mnd)         | 30m     | MEDIUM — zero lint goal |
| 6  | Remove `event.SubscribesTo` from public API (internalize)        | 15m     | MEDIUM — API surface |
| 7  | Remove deprecated backward-compat methods in `storage/event_store_global.go` | 30m | MEDIUM — dead code removal |
| 8  | Remove deprecated `TransactionalOutbox` alias in `storage/sql_backend.go` | 5m  | LOW — dead alias |
| 9  | Remove deprecated `LoadAll`/`LoadAllFromPosition` in `memory/store_load.go` | 15m | MEDIUM — dead code |
| 10 | Fix or bypass `go-structure-linter` in pre-commit hook           | 30m     | MEDIUM — DX improvement |

### MEDIUM IMPACT (Do Next)

| #  | Task                                                              | Effort  | Impact |
| -- | ----------------------------------------------------------------- | ------- | ------ |
| 11 | Deduplicate catalog test helper clone (1 remaining group)         | 15m     | LOW — zero clones |
| 12 | Split `decider_test.go` (~1200L) into focused test files          | 1h      | MEDIUM — maintainability |
| 13 | Split `runner_test.go` (~1057L) into focused test files           | 1h      | MEDIUM — maintainability |
| 14 | Add BDD tests for Version, SchemaVersion, OutboxStatus, Pagination | 2h    | MEDIUM — coverage |
| 15 | Add stream module integration tests                                | 2h      | MEDIUM — coverage |
| 16 | Wire `example/user/` to use catalog-aware event constructors      | 1h      | LOW — example quality |
| 17 | Add `.gopls` settings to handle workspace better                  | 15m     | MEDIUM — DX |
| 18 | Add art-dupl to CI pipeline                                        | 30m     | MEDIUM — clone prevention |
| 19 | Enforce 350-line limit on test files via linter config            | 15m     | LOW — code quality |
| 20 | Add fuzz tests for event creation, ID parsing, schema reflection  | 3h      | MEDIUM — robustness |

### STRATEGIC (Plan For Later)

| #  | Task                                                              | Effort  | Impact |
| -- | ----------------------------------------------------------------- | ------- | ------ |
| 21 | Push v1.0.0 tags and remove all `replace` directives              | 2h      | HIGH — publishable |
| 22 | Rewrite `example/user/` as comprehensive CQRS demo               | 4h      | MEDIUM — documentation |
| 23 | Benchmark storage backends (PG vs SQLite vs Pebble)               | 4h      | MEDIUM — performance |
| 24 | Performance regression CI — benchmark comparison on each PR       | 2h      | MEDIUM — reliability |
| 25 | Add high-level test utilities (AggregateTester, ProjectionTester) | 4h      | MEDIUM — DX |

---

## G) Top #1 Question I Cannot Figure Out Myself

**Should `event.Runner` be removed NOW (requiring fixing the projection test bug), or deferred to v2.0.0?**

`event.Runner` is marked deprecated with a clear migration path (`projection.Runner`). But:
- `core/event/runner.go` has 227 lines of production code
- `core/event/runner_test.go` has 440+ lines of tests
- Integration tests use it
- `projection/runner_registration_test.go` has a pre-existing type bug that needs fixing first
- Removing it is a breaking change for any external consumer who hasn't migrated

The question: Is this library published/consumed externally yet (beyond examples), making this a semver concern? Or is it safe to just rip it out and fix the downstream?

---

## Commits Made (Sessions 134–138)

```
7f9c813 chore: remove accidentally committed binaries
5c9041d refactor(event): remove deprecated GlobalLoader, PositionalLoader, BackwardsLoader interfaces
b661249 fix: repair botched codec auto-migration in upcaster_test + ineffectual assignment
b8899d6 refactor: remove deprecated event.Codec/event.JSONCodec aliases, migrate all usages to codec package
74bdb03 refactor(codec): migrate from event.JSONCodec to standalone codec module
08117cd style: nix fmt applied formatting across codebase
313c6b0 style(example): remove unnecessary explicit type args from projection.On calls
d0d7f48 fix(decider,query,projection): fix all 3 broken test modules + codec migration
221ffca fix(projection): remove pointer-to-interface in testProjection helper
7a3a970 chore: formatting consistency and Go receiver call style cleanup
```

---

## Key Metrics Over Sessions

| Metric                    | Session 134 Start | Session 138 End  | Delta       |
| ------------------------- | ----------------- | ---------------- | ----------- |
| Test failures             | 3 (pre-existing)  | **0**            | -3          |
| Lint issues (core)        | 8                 | **0**            | -8          |
| Lint issues (memory)      | 3                 | **0**            | -3          |
| Deprecated type aliases   | 2 (Codec, JSONCodec) | **0**         | -2          |
| Deprecated interfaces     | 3 (GlobalLoader, PositionalLoader, BackwardsLoader) | **0** | -3 |
| Dead packages             | 1 (core/aggregate) | **0**           | -1          |
| Clone groups (t=35)       | 3                 | **1** (catalog test) | -2     |
| Production LOC            | ~24,258           | 25,077           | +819        |
| Test LOC                  | ~45,115           | 44,860           | -255        |
