# Session 97B — Comprehensive Status Report

**Date:** 2026-05-24 06:36 CEST
**Branch:** master @ `fc9a0e5`
**Since last report:** 7 commits (since `7f6c28b`)

---

## Executive Summary

The library is in **the best shape it has ever been**. 27/27 packages pass, zero lint, zero vet. The deprecated `IdempotencyKey` is gone. All bare error returns in storage, memory, and projection are wrapped with context. Sentinel errors are consolidated. TursoSyncDB has its own file. The only remaining "fucked up" item is `example/todo` — still broken by the `cqrs-htmx` API break.

---

## A) Fully Done ✅

### This Session's Commits (7 since last report)

| Commit    | Summary                                                                      |
| --------- | ---------------------------------------------------------------------------- |
| `d758dc8` | Wrap bare errors in pebble store (Save, writeEventsToBatch, AppendBatch)     |
| `cb4f927` | Wrap bare errors in event_reconstruction, memory/bus, projection/runner_live |
| `8c24f60` | **Remove deprecated `IdempotencyKey`** from Command interface (breaking)     |
| `ab71fd5` | Consolidate sentinel errors to errors.go (dispatcher + sync)                 |
| `aa4e943` | MustParse panic tests: 4/7 → 7/7 coverage                                    |
| `e023b98` | Extract TursoSyncDB to own file                                              |
| `fc9a0e5` | gofumpt formatting                                                           |

### Quality Gates

| Gate            | Status                                               |
| --------------- | ---------------------------------------------------- |
| Tests           | ✅ 27/27 packages pass                               |
| Lint            | ✅ 0 issues across all 10 modules                    |
| Vet             | ✅ Clean                                             |
| Format          | ✅ Clean (gofumpt)                                   |
| Build           | ❌ example/todo broken (cqrs-htmx)                   |
| TODOs           | ✅ 0 TODO/FIXME/HACK markers                         |
| Deprecated APIs | ✅ Only `core/aggregate` package remains (by design) |

### Coverage (27 packages)

| Package                     | Coverage | Change                                     |
| --------------------------- | -------- | ------------------------------------------ |
| core/query                  | 100.0%   | —                                          |
| core/pkg/dispatcher         | 100.0%   | —                                          |
| core/pkg/id                 | 100.0%   | ↑ was 98.1%                                |
| middleware                  | 100.0%   | —                                          |
| catalog/adapters            | 100.0%   | —                                          |
| catalog/internal/caseutil   | 100.0%   | —                                          |
| memory                      | 99.6%    | —                                          |
| sync                        | 97.6%    | —                                          |
| catalog                     | 96.8%    | —                                          |
| core/aggregate              | 95.9%    | —                                          |
| catalog/d2                  | 95.0%    | —                                          |
| catalog/openapi             | 94.4%    | —                                          |
| testhelpers                 | 94.4%    | ↑ was 79.7% (Session 97)                   |
| projection                  | 94.4%    | —                                          |
| core/command                | 94.6%    | ↓ was 94.7% (removed IdempotencyKey tests) |
| catalog/asyncapi            | 93.7%    | —                                          |
| core/decider                | 93.6%    | —                                          |
| core/event                  | 93.8%    | —                                          |
| catalog/eventcatalog        | 91.3%    | —                                          |
| catalog/docserver           | 90.1%    | —                                          |
| storage                     | 89.3%    | —                                          |
| catalog/internal/schemautil | 84.2%    | —                                          |

### Completed Cleanup

- ✅ **All bare error returns wrapped** in storage (SQL + Pebble), memory/bus, projection runner
- ✅ **IdempotencyKey removed** from Command interface + all 8 callers
- ✅ **Sentinel errors consolidated** to errors.go files (dispatcher, sync)
- ✅ **Doc comments** on all exported types/functions (event, projection)
- ✅ **TursoSyncDB extracted** to its own file
- ✅ **SQLite WAL/SHM** in .gitignore
- ✅ **testhelpers 79.7% → 94.4%**
- ✅ **id 98.1% → 100.0%**
- ✅ **Zero lint** maintained across sessions

---

## B) Partially Done 🔶

### event.go File Size (273 lines, limit 250)

`core/event/event.go` is 23 lines over the 250-line convention. Contains the core `Core` event struct, `NewEvent` (51 lines), `validateEventParams` (48 lines), and constructors. Splitting is risky — tight coupling between struct definition, constructors, and validation.

**Options:**

1. Extract `NewEvent` + `validateEventParams` to `event_constructors.go`
2. Extract `MustParseType` + `MustParseAggregateType` to `event_parse.go`
3. Accept it — it's the most central file in the library

### Remaining Bare Error Returns (intentional)

These are in middleware and core internal paths where the error already has sufficient context from the caller:

- `middleware/retry.go` (4 locations) — retry loop internals, error already wrapped by caller
- `middleware/metrics.go` (2) — thin wrapper passing through
- `middleware/logging.go` (1) — thin wrapper
- `middleware/tracing.go` (2) — thin wrapper
- `core/decider/decider.go` (2) — already wrapped via `opError`
- `core/aggregate/` (2) — deprecated package
- `core/pkg/dispatcher/dispatcher.go` (1) — internal `CheckClosed` path
- `projection/builder.go` (2) — already wrapped by caller

These are acceptable. Middleware is intentionally transparent — it should not add context to errors passing through.

---

## C) Not Started ⬜

### Coverage Improvements

| Module                      | Current      | Lowest Functions                                    | Effort                   |
| --------------------------- | ------------ | --------------------------------------------------- | ------------------------ |
| storage/turso_sync.go       | 0% (7 funcs) | All TursoSyncDB methods                             | Needs Turso embedded lib |
| storage                     | 89.3%        | parseBrandedID 50%, scan funcs 75-78%               | Medium                   |
| catalog/internal/schemautil | 84.2%        | Reflect edge cases                                  | Low                      |
| core/event                  | 93.8%        | mergeFrom 61%, WithMetadata 60%, collectResults 73% | Medium                   |
| testhelpers assertions      | 60-67%       | Failure branches (t.Errorf paths)                   | Low but low value        |

### Structural

1. **Split `core/event/event.go`** — 273 lines → two files
2. **`sync` module rename** — Shadows stdlib `sync`. Needs owner decision.
3. **`query.Handler` returns `any`** — Design doc exists, architectural change
4. **No CHANGELOG.md** — Version history not tracked formally
5. **No benchmark suite** — Only integration/aggregate has benchmarks

### Documentation

6. **Update README** — Shows aggregate pattern, should show decider
7. **Add Go ExampleXxx functions** — Testable examples in godoc
8. **EventCatalog examples** — Usage docs for catalog builders

---

## D) Totally Fucked Up 💥

### example/todo — STILL BROKEN

```
github.com/larsartmann/cqrs-htmx/errors.go:34:9: undefined: event.RegisterClassification
```

**Status:** Unchanged since last report. `nix run .#build` still fails.
**Root cause:** `cqrs-htmx` references `event.RegisterClassification` removed in Session 89.
**Dependency chaos:** 39 indirect deps (Pebble, Turso, Casbin, Prometheus, cqrs-htmx).
**My recommendation:** Move `example/todo` to its own repo. It's the only thing breaking the build, and it pulls in more dependencies than the library itself.

---

## E) What We Should Improve

### Session 97 Remaining Bare Returns — Acceptable or Fix?

The 16 remaining bare `return err` fall into three categories:

1. **Middleware (9)** — Intentionally transparent. Wrapping would add noise. **Accept.**
2. **Core decider/aggregate/dispatcher (5)** — Already wrapped by callers or deprecated. **Accept.**
3. **Projection builder (2)** — `On` returns `ErrNilHandler` from registry, `Build` path. **Accept.**

### Top Issues by Priority

1. **example/todo build break** — Only build failure, only external coupling
2. **event.go size** — Only file over 250 lines
3. **TursoSyncDB test coverage** — 7 functions at 0%, need embedded Turso
4. **event metadata mergeFrom** — 61% coverage, untested merge paths

---

## F) Top 25 Things to Do Next

| #   | Task                                                    | Impact | Effort | Type         |
| --- | ------------------------------------------------------- | ------ | ------ | ------------ |
| 1   | Fix or remove example/todo (cqrs-htmx break)            | HIGH   | S      | Fix/Remove   |
| 2   | Split core/event/event.go under 250 lines               | MEDIUM | S      | Convention   |
| 3   | Add TursoSyncDB tests (0% → 70%+)                       | MEDIUM | M      | Testing      |
| 4   | Add event metadata mergeFrom tests (61% → 90%+)         | LOW    | S      | Testing      |
| 5   | Add storage parseBrandedID tests (50% → 90%+)           | LOW    | S      | Testing      |
| 6   | Add storage scanEvent/scanOutbox tests (75-78% → 90%+)  | MEDIUM | M      | Testing      |
| 7   | Add schemautil coverage (84.2% → 90%+)                  | MEDIUM | M      | Testing      |
| 8   | Update README to show decider pattern                   | MEDIUM | S      | Docs         |
| 9   | Create CHANGELOG.md                                     | LOW    | S      | Docs         |
| 10  | Add benchmark suite for core modules                    | MEDIUM | M      | Testing      |
| 11  | Discuss `sync` module rename with owner                 | LOW    | XS     | Decision     |
| 12  | Evaluate query.Handler generic redesign                 | MEDIUM | L      | Architecture |
| 13  | Add Go ExampleXxx functions for godoc                   | LOW    | M      | Docs         |
| 14  | Add collectResults coverage (73% → 90%+)                | LOW    | S      | Testing      |
| 15  | Add WithMetadata/WithCustom coverage (60-67% → 90%+)    | LOW    | S      | Testing      |
| 16  | Add publishPending coverage (67% → 90%+)                | LOW    | S      | Testing      |
| 17  | Add filterEvents coverage (73% → 90%+)                  | LOW    | S      | Testing      |
| 18  | Extract example/user and example/todo to separate repos | MEDIUM | M      | Structure    |
| 19  | Consider event.Store ISP split                          | HIGH   | L      | Architecture |
| 20  | Add fuzz tests for ID parsing and event marshaling      | MEDIUM | M      | Testing      |
| 21  | Review nolint directives (35 in production code)        | LOW    | M      | Cleanup      |
| 22  | Add NegativeCounterError.String() method                | LOW    | XS     | API          |
| 23  | Add Operation.String() test (0% coverage)               | LOW    | XS     | Testing      |
| 24  | Version bump audit across all modules                   | MEDIUM | S      | Release      |
| 25  | Add catalog usage examples                              | LOW    | S      | Docs         |

---

## G) My Top #1 Question

**Same as last time, still unresolved: Should `example/todo` move to its own repo?**

The build is broken. It has been broken since Session 89 (when we removed ~60 exports). The fix requires either:

1. Updating `cqrs-htmx` (external project, not ours)
2. Removing the `cqrs-htmx` dependency from `example/todo`
3. Moving `example/todo` out of this repo entirely

Option 3 is cleanest — it removes the only external coupling that breaks our build. `example/user` can stay since it has no external dependencies beyond the library itself.

---

## Codebase Stats

| Metric                  | Value                                              |
| ----------------------- | -------------------------------------------------- |
| Production Go code      | ~14,000 lines                                      |
| Test Go code            | ~31,300 lines                                      |
| Test:Production ratio   | 2.24:1                                             |
| Packages                | 27 (all passing)                                   |
| Modules                 | 12                                                 |
| Files over 250 lines    | 1 (`core/event/event.go` at 273)                   |
| Functions over 30 lines | 20                                                 |
| Bare error returns      | 16 (all acceptable)                                |
| nolint directives       | 35 in production                                   |
| Deprecated items        | 1 (`core/aggregate` package)                       |
| `any` usage             | 93 (mostly generics, codec, query — all justified) |

---

## Session History (Sessions 95–97)

| Session | Key Achievement                                                                                                                      |
| ------- | ------------------------------------------------------------------------------------------------------------------------------------ |
| 95      | Code deduplication sweep, 19+ test helper extractions                                                                                |
| 96      | VectorClock nil map bug, gopls hints, golden test refresh                                                                            |
| 97A     | Doc comments, testhelpers 80→94%, error wrapping, .gitignore, zero lint                                                              |
| 97B     | Pebble/memory/projection error wrapping, IdempotencyKey removal, sentinel consolidation, TursoSyncDB split, MustParse tests, id→100% |

---

_This report covers the state as of commit `fc9a0e5` on 2026-05-24._
