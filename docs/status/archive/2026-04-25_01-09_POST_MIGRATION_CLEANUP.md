# Comprehensive Status Report

**Date:** 2026-04-25 01:09 CEST
**Branch:** master
**Last Commit:** `67dac28` — feat(core): add Codec interface for event payload serialization
**Total Go Code:** ~14,102 lines across 5 modules + 2 examples
**Test Packages:** 13 packages, all green with race detection
**Overall Status:** Multi-module migration (Phases 0–4) complete. Post-migration cleanup in progress.

---

## Test Coverage Snapshot

| Package                | Coverage  | Trend  |
| ---------------------- | --------- | ------ |
| `catalog/adapters`     | **98.8%** | +32.8% |
| `memory`               | 99.2%     | stable |
| `catalog/asyncapi`     | 96.3%     | stable |
| `xtypes`               | 95.7%     | stable |
| `core/event`           | **89.0%** | +3.6%  |
| `catalog`              | 91.2%     | stable |
| `catalog/eventcatalog` | 89.7%     | stable |
| `core/pkg/id`          | 85.4%     | stable |
| `middleware`           | 84.6%     | stable |
| `core/command`         | 84.4%     | stable |
| `core/query`           | 91.5%     | stable |
| `core/pkg/dispatcher`  | 77.4%     | stable |
| `core/aggregate`       | 77.3%     | stable |

---

## A) FULLY DONE

### Multi-Module Migration (Phases 0–4)

| Phase | Description                                                   | Commits                                    | Status |
| ----- | ------------------------------------------------------------- | ------------------------------------------ | ------ |
| 0     | Fix query handler ctx, delete pkg/errors, replace custom YAML | multiple                                   | DONE   |
| 1     | go.work + move into `core/` subdirectory                      | multiple                                   | DONE   |
| 2     | Extract `memory/` module                                      | multiple                                   | DONE   |
| 3     | Extract `catalog/` module                                     | multiple                                   | DONE   |
| 4     | Extract `middleware/` + `xtypes`                              | `563f126`, `569adf7`, `4f4b0c7`, `d91990a` | DONE   |

### Post-Migration Cleanup (This + Prior Session)

| What                                                    | Commit    | Details                                                    |
| ------------------------------------------------------- | --------- | ---------------------------------------------------------- |
| Fix middleware test bugs from prior session             | `569adf7` | Detached `if` blocks, duplicate imports, wrong MaxAttempts |
| Remove stale core deps (go-faster/yaml + 5 transitives) | `d91990a` | `go mod tidy` after catalog/xtypes/middleware extraction   |
| Update AGENTS.md for Phase 4                            | `f12d71b` | Module table, architecture, test commands                  |
| Update Makefile for 5-module workspace                  | `f35c131` | Per-module test targets, explicit module paths             |
| Update CI workflows (test.yml + lint.yml)               | `58805de` | Matrix strategy for all 5 modules                          |
| Fix example import paths + go.mod                       | `5883c6a` | user + catalog examples updated                            |
| Fix .gitignore merged line                              | `1f36eb8` | Line 54 had two entries fused into one                     |
| Remove accidentally committed binaries                  | `1f36eb8` | example/user/user, example/catalog/catalog                 |
| Remove vestigial store_config.go                        | `889e7c3` | Zero callers, returned errors pointing to memory module    |
| Drop cockroachdb/errors from middleware                 | `5299ea9` | Replaced `errors.Wrapf` with `fmt.Errorf("%w")`            |
| Update README.md for 5-module structure                 | `380b99d` | Import paths, deps table, module structure, usage examples |
| Add go.work.example                                     | `d63c872` | Developer onboarding                                       |
| Fix flaky concurrency BDD test                          | `d057bf1` | `successes + conflicts == goroutines`                      |
| Add tests for 5 untested adapters functions             | `4815279` | `catalog/adapters` coverage 66% → 98.8%                    |
| Remove unused Streamer interface                        | `f0a38f0` | Streamer, StreamOptions, BatchSize — zero implementations  |
| Add Codec interface + JSONCodec                         | `67dac28` | Foundation for planned storage module                      |

---

## B) PARTIALLY DONE

### go.mod Stale Replace Directives

Four modules have `replace github.com/larsartmann/go-cqrs-lite/memory => ../memory` but never import the memory package:

- `catalog/go.mod` — needs `replace memory` removed
- `middleware/go.mod` — needs `replace memory` removed
- `xtypes/go.mod` — needs `replace memory` removed
- `example/catalog/go.mod` — needs `replace memory` removed

These were added during extraction as a safety measure (transitive test deps), but `go mod tidy` shows catalog actually needs the memory replace (core/event tests import memory). The others (`middleware`, `xtypes`) likely don't need it.

### LSP Diagnostics Noise

The `example/user` and `example/catalog` modules show persistent gopls errors because they're not in `go.work`. This is intentional (examples use `GOWORK=off` + replace directives), but the LSP noise makes real errors harder to spot. No real bugs — just tooling friction.

---

## C) NOT STARTED (Planned from Migration Plan)

| Phase | Description                              | Priority | Dependencies                                  |
| ----- | ---------------------------------------- | -------- | --------------------------------------------- |
| 5     | Storage module (sqlc event store)        | HIGH     | Codec interface (now done), PostgreSQL schema |
| 6     | Watermill module (pub/sub)               | HIGH     | core/event/Bus interface                      |
| 7     | Projection module (samber/ro internally) | MEDIUM   | core/event/Store, Watermill                   |
| 8     | Snapshot module (SQL-backed)             | MEDIUM   | core/event/SnapshotStore interface            |
| 9     | Test utilities module                    | LOW      | Extract testutil/testhelpers from core        |
| 10    | Tag releases (v1.0.0)                    | LOW      | All modules stable                            |

### Not Started — Code Quality Items

| Item                                                                                              | Effort | Impact |
| ------------------------------------------------------------------------------------------------- | ------ | ------ |
| Remove stale `replace memory` from middleware/xtypes go.mod                                       | LOW    | LOW    |
| Remove unused error sentinels (`ErrInvalidEventType`, `ErrEventNotFound`, `ErrCommandValidation`) | LOW    | LOW    |
| Remove or use `query.Result[T]` (never referenced)                                                | LOW    | LOW    |
| Add integration example using middleware + xtypes together                                        | MEDIUM | HIGH   |
| Write CONTRIBUTING.md for multi-module structure                                                  | MEDIUM | MEDIUM |
| Add Go doc examples (`Example*` test functions) for key APIs                                      | MEDIUM | HIGH   |
| Benchmark Codec implementations                                                                   | LOW    | LOW    |

---

## D) TOTALLY FUCKED UP

### Prior Session Left Broken Tests (Commit `563f126`)

The previous assistant's middleware extraction commit had **three syntax errors**:

1. Two detached `if` blocks (premature `}`)
2. Duplicate `errors` import (`"errors"` AND `"github.com/cockroachdb/errors"`)
3. Wrong `MaxAttempts=1` in retry test expecting 2 retries

**Lesson:** Always run `go build` + `go test` after writing code. The tests literally could not compile.

### README.md Edit Tool Failures (This Session)

The `multiedit` tool could not match UTF-8 emoji characters (✅) in the README tables. 3 of 8 edits failed silently. The tool's string comparison is byte-level and the ✅ checkmark character didn't match.

**Lesson:** For files with non-ASCII content, use `cat -A` to see exact bytes, or rewrite the entire section instead of surgical edits.

### No Downstream Consumers

The `xtypes` and `middleware` packages have **zero production consumers** — no file outside their own package imports them. They're library code waiting for users. This is by design (we're building the library), but means we have no validation that the API ergonomics actually work in practice.

---

## E) WHAT WE SHOULD IMPROVE

### Architecture

1. **No real event store** — The only event store implementation is `MemoryStore`. The planned `storage/` module (Phase 5) is the most impactful next step. The `Codec` interface just added is the foundation.

2. **No pub/sub** — Events are published synchronously via `MemoryBus`. No real message broker integration. The `Watermill` integration (Phase 6) is critical for production use.

3. **`query.Result[T]` is dead** — Defined in `core/query/query.go:27` but never referenced anywhere. Either use it or remove it.

4. **Branded IDs underused** — `id.Of[T]` is powerful but only 4 marker types exist (`AggregateID`, `EventID`, `UserID`, `CorrelationID`). The `xtypes` module adds `CommandID` but nobody uses it yet.

5. **No `Projection` interface** — The migration plan references `core/projection/` but no such package exists. This is needed before Phase 7.

### Code Quality

6. **Unused error sentinels** — `event.ErrInvalidEventType`, `event.ErrEventNotFound`, `command.ErrCommandValidation` are defined but never used. They should either be used in validation paths or removed.

7. **Stale go.mod replace directives** — `middleware` and `xtypes` have `replace memory => ../memory` but never import memory. Should be cleaned up.

8. **`example/user` is the only real integration test** — It uses core + memory + catalog together, but doesn't exercise middleware or xtypes. We need a more complete example.

9. **No Go doc examples** — The `Example*` test function convention gives interactive documentation on pkg.go.dev. None of our packages have these.

10. **Benchmark coverage is thin** — Only `catalog/adapters` and `core/event` have benchmarks. No benchmarks for command dispatch, query dispatch, aggregate repository, or middleware chains.

### Documentation

11. **README has no badges** — No CI status, coverage, Go Reference, or Go Report Card badges.

12. **No CHANGELOG** — 20+ status reports but no formal changelog tracking releases.

13. **No CONTRIBUTING.md** — Multi-module monorepos have non-obvious workflows (GOWORK=off, per-module tidy, replace directives). This should be documented.

### Tooling

14. **LSP noise from examples** — gopls tries to load example modules not in go.work, producing 47 diagnostic errors. Could add `go.work` entries for examples or add a `.golangci.yml` that scopes linting.

15. **No `golangci.yml`** — The project uses `golangci-lint run ./...` but has no config file. Default linters miss things (like the unused `query.Result[T]`).

---

## F) Top 25 Next Actions (Sorted by Impact × Effort)

### HIGH IMPACT, LOW EFFORT (Do These First)

| #   | Action                                                             | Effort | Impact | Why                           |
| --- | ------------------------------------------------------------------ | ------ | ------ | ----------------------------- |
| 1   | Remove stale `replace memory` from middleware/xtypes go.mod        | 5 min  | LOW    | Clean deps                    |
| 2   | Remove `query.Result[T]` (dead code)                               | 5 min  | LOW    | No callers                    |
| 3   | Remove unused error sentinels or wire them into validation         | 15 min | MEDIUM | Dead code confuses readers    |
| 4   | Add `.golangci.yml` with `unused`, `deadcode` linters              | 15 min | MEDIUM | Catch dead code automatically |
| 5   | Clean up stale `docs/status/2026-04-25_STATUS.md` (orphaned draft) | 2 min  | LOW    | Housekeeping                  |

### HIGH IMPACT, MEDIUM EFFORT (Next Sprint)

| #   | Action                                                                    | Effort | Impact | Why                                     |
| --- | ------------------------------------------------------------------------- | ------ | ------ | --------------------------------------- |
| 6   | Write integration example using middleware + xtypes + core + memory       | 2h     | HIGH   | First real validation of API ergonomics |
| 7   | Add Go doc `Example*` test functions for command, event, query, aggregate | 2h     | HIGH   | pkg.go.dev discoverability              |
| 8   | Define `Projection` interface in `core/projection/`                       | 1h     | HIGH   | Foundation for Phase 7                  |
| 9   | Write `CONTRIBUTING.md` for multi-module workflow                         | 1h     | MEDIUM | Onboarding                              |
| 10  | Add CI badges + Go Reference badge to README                              | 30 min | MEDIUM | Professional appearance                 |
| 11  | Remove or reduce `//nolint:err113` directives (use sentinel errors)       | 30 min | LOW    | Code quality                            |
| 12  | Add `golangci-lint` to CI with proper config                              | 30 min | MEDIUM | Automated quality gate                  |

### HIGH IMPACT, HIGH EFFORT (Major Features)

| #   | Action                                                               | Effort   | Impact   | Why                                 |
| --- | -------------------------------------------------------------------- | -------- | -------- | ----------------------------------- |
| 13  | **Phase 5: Storage module** (sqlc PostgreSQL event store)            | 2-3 days | CRITICAL | First real persistence layer        |
| 14  | **Phase 6: Watermill module** (pub/sub integrations)                 | 2-3 days | CRITICAL | Production-grade event distribution |
| 15  | **Phase 7: Projection module** (event handlers → SQL tables)         | 2-3 days | HIGH     | Read-model generation               |
| 16  | **Phase 8: Snapshot module** (SQL-backed snapshots)                  | 1-2 days | MEDIUM   | Aggregate load performance          |
| 17  | Write comprehensive integration test suite (core + memory + storage) | 1 day    | HIGH     | Confidence in module interactions   |

### MEDIUM IMPACT, VARIOUS EFFORT

| #   | Action                                                                 | Effort | Impact | Why                              |
| --- | ---------------------------------------------------------------------- | ------ | ------ | -------------------------------- |
| 18  | Add benchmarks for command/query dispatch, middleware chains           | 2h     | MEDIUM | Performance regression detection |
| 19  | Define `Upcaster` interface in `core/upcasting/` (from migration plan) | 1h     | MEDIUM | Event schema evolution           |
| 20  | Extract `testutil` into standalone module (Phase 9)                    | 2h     | LOW    | Reduce core test dep surface     |
| 21  | Write formal CHANGELOG.md                                              | 1h     | MEDIUM | Release tracking                 |
| 22  | Add `example/ecommerce/` full-stack example (all modules)              | 4h     | HIGH   | "Kitchen sink" demo              |
| 23  | Investigate `go 1.26 ignore` directive for examples/ in go.work        | 30 min | LOW    | Clean `go test ./...`            |

### LOW PRIORITY

| #   | Action                                                            | Effort | Impact | Why                 |
| --- | ----------------------------------------------------------------- | ------ | ------ | ------------------- |
| 24  | Phase 10: Tag v0.1.0 releases for each module                     | 1h     | MEDIUM | Semantic versioning |
| 25  | Add fuzz targets for event parsing, ID parsing, schema reflection | 2h     | LOW    | Edge case coverage  |

---

## G) Top Question I Cannot Figure Out Myself

**Should we keep `xtypes` and `middleware` as top-level modules, or merge them into `core`?**

Both packages have **zero external consumers**. They exist as separate modules because the migration plan said "extract everything," but:

- `xtypes` depends only on `core` and adds typed wrappers. It could be `core/xtypes/` with zero overhead.
- `middleware` depends only on `core` and adds cross-cutting concerns. It could be `core/middleware/` with zero overhead.

Keeping them separate means:

- Users can `go get` only what they need (the stated goal)
- But nobody `go get`s them because they're not useful without `core`
- Two extra `go.mod` files to maintain with stale replace directives

Merging them back means:

- Simpler module graph (3 modules instead of 5: core, memory, catalog)
- Fewer go.mod files to maintain
- But users who only want `core/command` would also pull in middleware types (though Go's module system doesn't load all packages, so this is mostly theoretical)

The answer depends on whether we expect users to import `middleware` without `core` (unlikely) or `xtypes` without `core` (impossible — it re-exports core types).

---

## Module Dependency Graph

```
core/          (errors, uuid, json; test-dep on memory)
  ↑
memory/        (core, errors)
  ↑ (transitive only)
catalog/       (core, yaml, json)
middleware/     (core only — errors is indirect)
xtypes/        (core)
```

**Note:** `memory` is a transitive dependency of `catalog`, `middleware`, and `xtypes` only because `core/event` tests import `memory`. In production code, only `core` is a direct dep.

---

## Session History

This session is a continuation of two prior sessions:

- Session 1: Multi-module migration Phases 0–4 (20+ commits)
- Session 2: Post-migration cleanup (docs, CI, examples, Makefile)
- Session 3 (this one): Code quality improvements (7 commits)

Total commits across all sessions: ~27
Total lines changed: ~1,500+
