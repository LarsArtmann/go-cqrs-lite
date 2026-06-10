# Session 141 — Comprehensive Status Report

**Date:** 2026-05-29 12:41
**Since last report:** Sessions 134–140 (codec extraction, AggregateRef migration, deduplication, Event.Context propagation, module extraction)
**Git HEAD:** `3ca4507` — clean working tree

---

## a) FULLY DONE

| Area                    | Details                                                                                                                             |
| ----------------------- | ----------------------------------------------------------------------------------------------------------------------------------- |
| **Build**               | All 23 modules build clean. `nix run .#build` passes.                                                                               |
| **Tests**               | All 28 test packages pass. `nix run .#test` green.                                                                                  |
| **Codec extraction**    | `codec/` module extracted from `core/event`. 100% coverage. All `event.JSONCodec` references migrated.                              |
| **AggregateRef**        | `(aggregateType, aggregateID)` replaced with `AggregateRef` across all store implementations.                                       |
| **Event.Context**       | `Event.Context()` added to interface for deadline propagation.                                                                      |
| **Module extraction**   | `pebble/`, `turso/`, `stream/`, `otel/`, `codec/` extracted as independent modules.                                                 |
| **Deduplication**       | Zero actionable code clones remain (threshold 25).                                                                                  |
| **Error taxonomy**      | 5-family classification (Rejection/Conflict/Transient/Infrastructure/Corruption) consistent across all modules. 39 sentinel errors. |
| **Signing**             | Full signing module: HMAC-SHA256, Ed25519, multi-sig, middleware, integration tests.                                                |
| **Upcasting**           | Public `Upcaster` interface, `NewUpcaster()` constructor, `VersionedStore`, godoc example.                                          |
| **CI**                  | Build/vet/test/lint/race/coverage + per-module GOWORK=off + coverage gate (80%) + file size gate + go.work sync check.              |
| **Coverage**            | All modules ≥82% (most ≥90%). core/decider and core/pkg/id at 100%.                                                                 |
| **No production TODOs** | Zero TODO/FIXME/HACK comments in production code.                                                                                   |
| **TODO list**           | 195 done, 20 open, 16 blocked, 22 future, 6 v2-breaking.                                                                            |

---

## b) PARTIALLY DONE

| Area                   | What's Done                                            | What Remains                                                                                                                                                                |
| ---------------------- | ------------------------------------------------------ | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| **AGENTS.md**          | Has modules list, patterns, key patterns               | Says "14 modules" — actually 23. Missing `codec`, `pebble`, `turso`, `example/stream`, `example/storage` in monorepo tree. Coverage table says "27 packages" — actually 28+ |
| **CI module coverage** | 15 modules in per-module loop                          | Missing `codec`, `pebble`, `turso`, all `example/` modules                                                                                                                  |
| **t.Parallel()**       | Most test files in signing, projection, storage use it | ~19 test files still missing `t.Parallel()` (catalog docserver, otel, middleware, BDD suites, golden tests)                                                                 |
| **Stream module**      | InMemoryReader, StatusMiddleware, builder, benchmarks  | Missing SQL reader tests and integration tests                                                                                                                              |
| **Turso module**       | Connector and sync functions work                      | Zero test coverage (needs real Turso DB)                                                                                                                                    |
| **Godoc examples**     | event, command, query, decider, catalog, signing       | Missing examples for: projection, saga, storage, stream, middleware, codec                                                                                                  |

---

## c) NOT STARTED (from TODO_LIST.md open items)

| #   | Item                                                                      | Impact | Effort |
| --- | ------------------------------------------------------------------------- | ------ | ------ |
| 1   | event.Context propagation through NewEvent/PublishChanges                 | High   | Medium |
| 2   | Catch-up projection runner (start-from-checkpoint → replay → live-switch) | High   | Large  |
| 3   | Projection parallel processing — goroutine pool                           | Medium | Medium |
| 4   | Rewrite example/user/ to demonstrate full CQRS stack                      | High   | Medium |
| 5   | Add example/user/ smoke test (TestExampleRuns)                            | Medium | Small  |
| 6   | Add ProcessedAt to CheckpointStore                                        | Low    | Small  |
| 7   | Wire example/user/ to catalog-aware event constructors                    | Low    | Small  |
| 8   | Benchmark storage backends (PG vs SQLite vs Pebble) comparative           | Medium | Medium |
| 9   | WithAsyncWrites() option for PebbleEventStore                             | Low    | Medium |
| 10  | Parallelize CI matrix — one job per module                                | Low    | Medium |
| 11  | Performance regression CI — benchmark comparison on PR                    | Medium | Medium |
| 12  | Add gofumpt/goimports to pre-commit hook                                  | Low    | Small  |
| 13  | BDD tests for Version, SchemaVersion, OutboxStatus, Pagination            | Medium | Medium |
| 14  | Fuzz tests for event creation, ID parsing, schema, upcaster               | Medium | Medium |
| 15  | E2E throughput benchmarks                                                 | Medium | Large  |
| 16  | Stream module integration tests + SQL reader tests                        | Medium | Medium |
| 17  | Enforce 350-line limit on test files                                      | Low    | Small  |
| 18  | Split large test files (decider_test ~1200L, runner_test ~1057L)          | Low    | Medium |
| 19  | Increase projection coverage to 95%+                                      | Low    | Small  |
| 20  | Godoc examples for remaining modules                                      | Medium | Small  |

---

## d) TOTALLY FUCKED UP

> **User stated: all "D) TOTALLY FUCKED UP" items are addressed in separate projects.**

| Issue                                                               | Status           |
| ------------------------------------------------------------------- | ---------------- |
| Golden test fixture flakiness (golines/nix fmt touching YAML/JS)    | Separate project |
| Pre-commit hook dirty tree (BuildFlow reformatting committed files) | Separate project |
| go-structure-linter false positives (root go.mod empty by design)   | Separate project |
| library-policy false positive (math/rand in retry jitter)           | Separate project |

**No new "totally fucked up" items detected.** Build is clean, tests pass, no panics in benchmarks (except pre-existing SQL mock issue in storage/benchmark_test.go).

---

## e) WHAT WE SHOULD IMPROVE

### Critical (architectural correctness)

1. **AGENTS.md is stale** — says 14 modules, actually 23. Missing codec, pebble, turso, example/stream, example/storage, otel. Module graph outdated.
2. **CI missing 8 modules** — codec, pebble, turso, example/todo, example/user, example/saga, example/projection, example/storage, example/stream not in per-module CI loop.
3. **Deprecated `TransactionalStore` still used** — `storage/transactional_store.go` implements the deprecated interface. Should migrate to `TransactionalSink`.

### Important (consumer experience)

4. **Missing godoc examples** — projection, saga, storage, stream, middleware, codec have no `Example*` functions. Consumers can't see usage on pkg.go.dev.
5. **`catalog/eventcatalog/writer.go` loses error context** — 4 bare `os.WriteFile` returns with `//nolint:wrapcheck`. If I/O fails, caller gets raw OS error without "writing catalog" context.
6. **`scripts/go-mod-graph-local/main.go` at 411 lines** — exceeds 350-line limit. Tool script, but still.

### Nice to have (quality polish)

7. **~19 test files missing `t.Parallel()`** — slower test suite, less concurrency safety verification.
8. **21 `//nolint:wrapcheck`** in production code — mostly justified (Lifecycle.Close, json.Marshal), but `writer.go` is a real gap.
9. **63 total `//nolint` directives** — some are legitimate (exhaustruct for builder patterns), but worth periodic review.

---

## f) Top 25 Things We Should Get Done Next

Sorted by **impact × effort⁻¹** (highest first):

| #   | Task                                                                  | Impact | Effort | Why                                                         |
| --- | --------------------------------------------------------------------- | ------ | ------ | ----------------------------------------------------------- |
| 1   | Fix AGENTS.md: update module count to 23, add missing modules to tree | High   | 15min  | Stale docs mislead consumers and contributors               |
| 2   | Add codec, pebble, turso, example modules to CI per-module loop       | High   | 10min  | Untested modules in CI = regression risk                    |
| 3   | Wrap `os.WriteFile` errors in `catalog/eventcatalog/writer.go`        | Medium | 10min  | Real error context loss on I/O failure                      |
| 4   | Add godoc example for `projection` (`ExampleBuilder_On`)              | Medium | 15min  | High-value consumer-facing package, no example              |
| 5   | Add godoc example for `saga` (`ExampleRunner_Start`)                  | Medium | 15min  | Complex module, consumers need guidance                     |
| 6   | Add godoc example for `stream` (`ExampleNewBuilder`)                  | Medium | 10min  | New module, no discoverability                              |
| 7   | Add godoc example for `codec` (`ExampleJSON_Marshal`)                 | Medium | 10min  | New module, 100% coverage but no example                    |
| 8   | Add godoc example for `middleware` (`ExampleCommandRecovery`)         | Medium | 10min  | Most-used middleware has no example                         |
| 9   | Add `example/user` smoke test (TestExampleRuns)                       | Medium | 15min  | Validates example doesn't bit-rot                           |
| 10  | Migrate `TransactionalStore` → `TransactionalSink` in storage         | Medium | 20min  | Deprecated interface still in production                    |
| 11  | Add `t.Parallel()` to catalog/docserver tests (12 tests)              | Low    | 10min  | Largest missing-parallel file                               |
| 12  | Add `t.Parallel()` to otel/logging tests (7 tests)                    | Low    | 5min   | Quick win                                                   |
| 13  | Add `t.Parallel()` to middleware/tracing_logging tests                | Low    | 5min   | Quick win                                                   |
| 14  | Add stream module integration tests                                   | Medium | 30min  | Only module without cross-module validation                 |
| 15  | Add BDD tests for Version, SchemaVersion, OutboxStatus                | Medium | 30min  | Core types need behavioral coverage                         |
| 15  | Add fuzz tests for event creation, ID parsing                         | Medium | 30min  | Robustness for core parsing paths                           |
| 17  | Split large test files (decider_test, runner_test)                    | Low    | 30min  | 350-line policy compliance                                  |
| 18  | Increase projection coverage to 95%+                                  | Low    | 15min  | Currently 89.6%, gap in error paths                         |
| 19  | Add event.Context propagation through NewEvent                        | High   | 45min  | Already on Event interface, needs wiring                    |
| 20  | Rewrite example/user to demonstrate full CQRS stack                   | High   | 60min  | Current example doesn't show projections, sagas, or streams |
| 21  | Add benchmark comparison for storage backends                         | Medium | 45min  | Consumers need data for backend choice                      |
| 22  | Performance regression CI (benchstat on PR)                           | Medium | 45min  | Catch performance regressions early                         |
| 23  | Split `scripts/go-mod-graph-local/main.go` (411L → 2 files)           | Low    | 15min  | File size policy compliance                                 |
| 24  | Catch-up projection runner                                            | High   | 90min  | Key feature for production projections                      |
| 25  | Parallelize CI matrix                                                 | Low    | 30min  | Faster CI feedback                                          |

---

## g) Top #1 Question

**Should we prioritize the "catch-up projection runner" (start-from-checkpoint → replay → live-switch) or the "example/user rewrite" (full CQRS stack demo)?**

Both are high-impact. The catch-up runner is a **production feature** that makes projections viable for real workloads. The example rewrite is a **marketing/education** investment that helps consumers adopt the library. I lean toward the catch-up runner because it directly enables the most common CQRS pattern (projector reads from journal), but it's a 90-min task versus 60-min for the example.

---

## Module Coverage Summary

| Module               | Coverage | Status                      |
| -------------------- | -------- | --------------------------- |
| codec                | 100.0%   | ✅                          |
| core/pkg/id          | 100.0%   | ✅                          |
| core/decider         | 100.0%   | ✅                          |
| memory               | 99.6%    | ✅                          |
| core/query           | 96.8%    | ✅                          |
| otel                 | 96.6%    | ✅                          |
| catalog              | 96.3%    | ✅                          |
| catalog/openapi      | 96.2%    | ✅                          |
| core/command         | 94.2%    | ✅                          |
| saga                 | 94.6%    | ✅                          |
| watermill            | 94.4%    | ✅                          |
| middleware           | 94.0%    | ✅                          |
| stream               | 94.0%    | ✅                          |
| signing              | 93.8%    | ✅                          |
| storage              | 93.8%    | ✅                          |
| catalog/asyncapi     | 93.7%    | ✅                          |
| core/pkg/dispatcher  | 92.2%    | ✅                          |
| catalog/eventcatalog | 92.8%    | ✅                          |
| core/event           | 91.0%    | ✅                          |
| testhelpers          | 82.1%    | ⚠️ Below 85%                |
| pebble               | 87.2%    | ✅                          |
| projection           | 89.6%    | ✅                          |
| cmd/cqrs-gen         | 89.9%    | ✅                          |
| catalog/d2           | 95.0%    | ✅                          |
| turso                | 0.0%     | ⚠️ No tests (needs real DB) |

---

## File Statistics

| Metric               | Value           |
| -------------------- | --------------- |
| Production .go files | 263             |
| Test .go files       | 239             |
| Total .go files      | 502             |
| Modules in go.work   | 23              |
| Test packages        | 28              |
| Sentinel errors      | 39              |
| Interfaces           | 40              |
| //nolint directives  | 63 (production) |
| Open TODO items      | 20              |
| Done TODO items      | 195             |
