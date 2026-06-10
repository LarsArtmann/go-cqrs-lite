# go-cqrs-lite — Comprehensive Status Report

**Date:** 2026-06-02 14:12 CEST
**Session:** 11 (post-release comprehensive audit + project-dependency-graph arrow fix)
**Since Last Report:** `c2b65de2` → HEAD (3 planning/status commits since v2.0.0 tag)
**Status:** 🟡 **v2.0.0 released, but has unfixed production bugs. No code fixes applied yet.**

---

## Executive Summary

**v2.0.0 is shipped but not yet battle-ready.** The release itself is complete — 23 modules tagged, all tests green, zero lint, zero build errors. However, a deep audit found **4 CRITICAL production bugs** (data races, OOM, broken pagination) and **3 HIGH-priority bugs** (dropped context, Postgres incompatibility, data race). None of these have been fixed yet — only planning documents have been committed.

In parallel, the **project-dependency-graph** tool (separate repo) had missing directional arrows in its HTML ECharts output — fixed by adding `edgeSymbol` / `edgeSymbolSize` to the graph series config.

**Bottom line:** The library compiles, tests pass, but real consumers would hit data races in `projection.Runner`, OOM on `HealthCheck`, and broken Postgres queries. These must be fixed before recommending production use.

---

## a) FULLY DONE ✅

### v2.0.0 Release (Complete)

| Item                                                   | Detail                                                                |
| ------------------------------------------------------ | --------------------------------------------------------------------- |
| `/v2` semantic import paths                            | 426 files migrated, 23 modules + 1 root tag pushed                    |
| Generic middleware refactoring                         | 27→9 functions via `middleware/generic.go`                            |
| Circuit breaker fix                                    | Double-wrapping eliminated, bare sentinel pattern                     |
| command.Metadata fix                                   | Split brain resolved via `type Metadata = event.Metadata` alias       |
| watermill metadata fix                                 | Error swallowing eliminated, errors surfaced as Corruption-classified |
| schema.VersionedStore fix                              | Embedded Store hidden from public API                                 |
| Testify → Gomega                                       | 5 test files migrated                                                 |
| Ghost reactive files                                   | Removed from command/ and query/                                      |
| GOWORK=off per-module builds                           | 20/20 verified                                                        |
| go.sum regeneration                                    | All modules have correct checksums                                    |
| Zero lint / Zero build errors / 38 test packages green | Confirmed                                                             |

### Code Quality Baseline

| Metric                     | Value                   |
| -------------------------- | ----------------------- |
| Production LOC             | 23,376                  |
| Test LOC                   | 39,352                  |
| Total Go LOC               | 62,728                  |
| Test packages              | 38/38 GREEN             |
| Lint issues                | 0                       |
| Build errors               | 0                       |
| `//go:generate` directives | 0                       |
| Stale `core/` paths        | 0                       |
| Average coverage           | 89.4%                   |
| `//nolint` directives      | ~120 (mostly justified) |

### Per-Module Coverage (Current)

| Module                      | Coverage | Status             |
| --------------------------- | -------- | ------------------ |
| `dispatcher`                | 100.0%   | ✅                 |
| `codec`                     | 100.0%   | ✅                 |
| `decider`                   | 100.0%   | ✅                 |
| `catalog/internal/caseutil` | 100.0%   | ✅                 |
| `middleware`                | 98.5%    | ✅                 |
| `memory`                    | 99.1%    | ✅                 |
| `catalog`                   | 95.9%    | ✅                 |
| `catalog/openapi`           | 96.2%    | ✅                 |
| `catalog/d2`                | 95.0%    | ✅                 |
| `catalog/asyncapi`          | 93.7%    | ✅                 |
| `catalog/eventcatalog`      | 92.8%    | ✅                 |
| `catalog/schema`            | 86.1%    | ✅                 |
| `catalog/docserver`         | 90.1%    | ✅                 |
| `query`                     | 95.5%    | ✅                 |
| `id`                        | 94.5%    | ✅                 |
| `command`                   | 93.8%    | ✅                 |
| `signing/multisig`          | 94.1%    | ✅                 |
| `signing`                   | 93.9%    | ✅                 |
| `listing`                   | 93.8%    | ✅                 |
| `watermill`                 | 92.5%    | ✅                 |
| `projection`                | 91.3%    | ✅                 |
| `snapshot`                  | 92.3%    | ✅                 |
| `pebble`                    | 88.0%    | ✅                 |
| `event`                     | 89.0%    | ✅                 |
| `cmd/cqrs-gen`              | 89.9%    | ✅                 |
| `schema`                    | 85.5%    | ⚠️ Could improve   |
| `storage`                   | 72.7%    | ⚠️ Needs attention |
| `turso`                     | 28.6%    | 🔴 Critically low  |

### project-dependency-graph Arrow Fix (External)

- Fixed missing directional arrows in HTML/ECharts output
- Added `edgeSymbol: ['none', 'arrow']` and `edgeSymbolSize: [0, 8]` to graph series
- File: `/home/lars/projects/project-dependency-graph/renderer/assets/script.js:462-463`

---

## b) PARTIALLY DONE ⚠️

| Item                         | State                                                   | Gap                                                    |
| ---------------------------- | ------------------------------------------------------- | ------------------------------------------------------ |
| **turso module**             | 28.6% coverage                                          | EventStore CRUD untested, error paths untested         |
| **storage module**           | 72.7% coverage                                          | Aggregate reader, error paths, closed-state behavior   |
| **Replace directives**       | Retained for GOWORK=off                                 | Cannot remove — needed for per-module CI               |
| **Error taxonomy adoption**  | Framework exists (5 families, 13 helpers, 16 sentinels) | ~130 `fmt.Errorf` calls not migrated across 8 packages |
| **project-dependency-graph** | Arrow fix applied                                       | Not committed yet (separate repo)                      |
| **Planning docs**            | 2 execution plans written                               | Zero tasks from either plan have been executed         |

---

## c) NOT STARTED 📐

### From the 73-Task Execution Plan (ZERO tasks executed)

**Phase 1: Critical Production Bugs (6 tasks, ~40 min)** — NOT STARTED

- Fix Runner.cancel data race
- Fix Runner.projections data race
- Fix HealthCheck OOM
- Fix ReadFrom pagination
- Add Runner concurrency tests

**Phase 2: High-Priority Bugs (7 tasks, ~60 min)** — NOT STARTED

- Fix PublisherAdapter drops context
- Fix SQLAggregateReader `?` placeholders
- Fix SubscriberAdapter.handlers data race
- Add Pebble Close() method
- Fix LoadAtVersion snapshot
- Fix createTable context.Background()
- Fix subscribeLive handler leak

**Phases 3-10 (54 tasks)** — NOT STARTED

- Quality fixes, error taxonomy migration, code decomposition, library modernization, test coverage, architecture, CI, docs

### Type System Unification (Architecture)

- [ ] Shared `Message` interface (TypeName() string) across command/event/query
- [ ] Unified metadata options (4 duplicated WithX functions → shared)
- [ ] Generic `BrandedInt[T]` for Version/SchemaVersion
- [ ] `query.TypedHandler[T]` returning `(T, error)` — biggest type safety gap

### Testing & Quality

- [ ] Turso meaningful test coverage (28.6% → 70%+)
- [ ] Storage test coverage (72.7% → 85%+)
- [ ] Benchmark storage backends (PG vs SQLite vs Pebble)
- [ ] Fuzz tests for event creation, ID parsing, schema reflection
- [ ] E2E throughput benchmarks

### Documentation

- [ ] ROADMAP.md
- [ ] Documentation site (Docusaurus/MkDocs/Hugo)
- [ ] pkg.go.dev hosting setup

---

## d) TOTALLY FUCKED UP 🔴

### CRITICAL — Production Bugs (Will Break Real Consumers)

| #   | Issue                                                                                        | File                               | Impact                                             | Fixed? |
| --- | -------------------------------------------------------------------------------------------- | ---------------------------------- | -------------------------------------------------- | ------ |
| 1   | **Data race on `Runner.cancel`** — `Run()` writes, `Close()` reads without sync              | `projection/runner.go:106,230`     | Panic, corrupted state                             | ❌     |
| 2   | **Data race on `Runner.projections`** — `Register()` appends, `Run()` reads without mutex    | `projection/runner.go:85,99`       | Dropped projections, corrupted slice               | ❌     |
| 3   | **`HealthCheck` loads ALL events** — `ReadAll()` on full event store                         | `projection/health.go:20-22`       | **OOM in production**                              | ❌     |
| 4   | **`ReadFrom` pagination broken** — `WHERE id > ? ORDER BY occurred_at` is semantically wrong | `storage/event_store_global.go:66` | **Skipped/duplicated events** in projection replay | ❌     |

### HIGH — Context/Race Bugs (Will Bite Specific Patterns)

| #   | Issue                                                         | File                                    | Impact                              | Fixed? |
| --- | ------------------------------------------------------------- | --------------------------------------- | ----------------------------------- | ------ |
| 5   | **`PublisherAdapter` drops context** — `context.Background()` | `watermill/publisher.go:24`             | Broken OTel spans, lost deadlines   | ❌     |
| 6   | **`SQLAggregateReader` hardcodes `?`** — breaks PostgreSQL    | `storage/sql_aggregate_reader.go:63-86` | Runtime error on Postgres           | ❌     |
| 7   | **`SubscriberAdapter.handlers` data race** — no mutex         | `watermill/subscriber.go:58`            | Panic on concurrent subscribe+close | ❌     |

### MEDIUM — Design Issues

| #   | Issue                                                                | File                                  | Impact                          |
| --- | -------------------------------------------------------------------- | ------------------------------------- | ------------------------------- |
| 8   | Pebble has no `Close()` — DB never closed                            | `pebble/store.go`                     | Resource leak                   |
| 9   | `subscribeLive` handler leak — stale handler on bus after stop       | `projection/runner_live.go:19`        | Handler fires on stopped runner |
| 10  | `SQLEventStore` no closed state — ops after Close get cryptic errors | `storage/event_store.go:55-61`        | Confusing errors                |
| 11  | `LoadAtVersion` snapshot doesn't filter in SQL                       | `storage/snapshot.go:109`             | Wasted DB round-trips           |
| 12  | `createTable()` uses `context.Background()`                          | `storage/aggregate_projection.go:100` | Unstoppable DDL                 |
| 13  | 15+ `time.Sleep` in tests                                            | Multiple test files                   | Flaky CI                        |

### The Problem: Plans Written, Zero Execution

The biggest "fuck up" is that we spent 3 sessions writing planning documents (deep audit, 73-task plan, Pareto plan) but **have not fixed a single bug**. The 4 CRITICAL bugs have been known since Session 10 and remain unfixed. Every consumer who imports this library in production will hit these.

---

## e) WHAT WE SHOULD IMPROVE

### Architecture

1. **Shared `Message` interface** — Eliminate `MessageAdapter` bridge in middleware/generic.go. `command.Type`, `event.Type`, `query.Type` are all `type X string` with identical methods.

2. **Unified metadata options** — `WithCorrelationID` duplicated across event/ and command/ with identical implementations. A `MetadataCarrier` interface would halve the code.

3. **Generic `BrandedInt[T]`** — `Version` and `SchemaVersion` share ~80% of methods. A generic base eliminates ~60 lines of duplication.

4. **`query.TypedHandler[T]` generic return** — Currently returns `(any, error)`. Should be `(T, error)`. Single biggest type safety gap.

### Code Quality

5. **Error taxonomy adoption** — 130+ `fmt.Errorf` across production code should use `event.Wrap*` families. Offenders: decider, storage, watermill, schema, codec, listing, projection, id, command, query.

6. **`errgroup` for parallel dispatch** — Manual semaphore + WaitGroup in `projection/runner_live.go:54-77`. `errgroup.Group.SetLimit(n)` = ~5 lines.

7. **`filepath.WalkDir`** — 2 files use deprecated `filepath.Walk`.

8. **Test flakiness** — 15+ `time.Sleep` calls in tests. Use channels or sync primitives.

### Process

9. **Stop planning, start executing** — 3 sessions of planning with zero bug fixes is unacceptable. The Pareto plan says 30 minutes fixes 51% of value. DO IT.

10. **Test with `-race` systematically** — The Runner data races existed before v2.0.0 release. Tests pass because they don't exercise concurrent Close()+Run().

11. **Tag verification** — The first v2.0.0 tag push had wrong module paths. Test `GOWORK=off go build` before tagging.

---

## f) Top #25 Things to Get Done Next

Sorted by Impact × Effort (highest first):

| #   | Task                                                                            | Impact      | Effort | Category      |
| --- | ------------------------------------------------------------------------------- | ----------- | ------ | ------------- |
| 1   | **Fix HealthCheck OOM** — replace ReadAll with checkpoint-only ping             | 🔴 CRITICAL | 10 min | Bug           |
| 2   | **Fix SQLAggregateReader `?` → Dialect.Placeholder**                            | 🔴 CRITICAL | 15 min | Bug           |
| 3   | **Fix SubscriberAdapter map race** — add sync.Mutex                             | 🔴 CRITICAL | 10 min | Bug           |
| 4   | **Fix Runner.cancel data race** — mutex or atomic                               | 🔴 CRITICAL | 10 min | Bug           |
| 5   | **Fix Runner.projections data race** — mutex around Register+Run                | 🔴 CRITICAL | 10 min | Bug           |
| 6   | **Fix ReadFrom pagination** — cursor-based with proper ordering                 | 🔴 CRITICAL | 30 min | Bug           |
| 7   | **Fix PublisherAdapter drops context**                                          | 🟠 HIGH     | 5 min  | Bug           |
| 8   | **Add Pebble Close() method**                                                   | 🟠 HIGH     | 10 min | Bug           |
| 9   | **Fix subscribeLive handler leak**                                              | 🟠 HIGH     | 20 min | Bug           |
| 10  | **Add Runner concurrency tests** (Register+Run, Run+Close)                      | 🟠 HIGH     | 15 min | Test          |
| 11  | **Fix SQLEventStore closed state tracking**                                     | 🟡 MED      | 15 min | Quality       |
| 12  | **Fix createTable context.Background()**                                        | 🟡 MED      | 5 min  | Bug           |
| 13  | **Fix LoadAtVersion snapshot SQL filter**                                       | 🟡 MED      | 10 min | Bug           |
| 14  | **Error taxonomy: decider + schema + codec + listing + projection**             | 🟡 MED      | 30 min | Quality       |
| 15  | **Error taxonomy: storage + watermill + id + command + query**                  | 🟡 MED      | 30 min | Quality       |
| 16  | **Fix 6 quality bugs** (Version.Sub, codec raw, GetID, ToAny, HasSignature)     | 🟡 MED      | 20 min | Quality       |
| 17  | **Remove dead code** (ErrUnknownBackend, return nil, TombstoneInclude, aliases) | 🟡 MED      | 10 min | Cleanup       |
| 18  | **Replace dispatchParallel → errgroup.SetLimit**                                | 🟢 LOW      | 10 min | Modernization |
| 19  | **Decompose watermill messageToEvent** (81L → 3 funcs)                          | 🟢 LOW      | 20 min | Quality       |
| 20  | **Decompose storage ListWithStatus** (112L → 3 funcs)                           | 🟢 LOW      | 30 min | Quality       |
| 21  | **Turso test coverage** 28.6% → 70%+                                            | 🟢 LOW      | 45 min | Test          |
| 22  | **Storage test coverage** 72.7% → 85%+                                          | 🟢 LOW      | 45 min | Test          |
| 23  | **Create shared `message/` module** with TypeName interface                     | 🟡 MED      | 45 min | Architecture  |
| 24  | **Unified metadata options** (4 dupes → shared)                                 | 🟡 MED      | 30 min | Architecture  |
| 25  | **ROADMAP.md creation**                                                         | 🟢 LOW      | 15 min | Docs          |

---

## g) Top #1 Question I Cannot Figure Out Myself

**Should the shared `Message` interface live in a new `message/` module, inside `event/`, or inside `dispatcher/`?**

Options:

- **(A) New `message/` module** — Clean domain vocabulary, leaf module (no deps), importable by consumers. Adds 31st module.
- **(B) In `event/`** — Already foundational, command/ and query/ already depend on it. Semantically weird for "query" to import from "event".
- **(C) In `dispatcher/`** — Already imported by command/ and query/. But dispatcher is about dispatch mechanics, not message typing.

**My recommendation:** Option (A) — a `message/` module with just `TypeName() string`, the `Type` branded string, and potentially shared metadata carrier interface. Clean, minimal, and correct.

**Why I can't decide:** This is a domain modeling decision that affects every consumer's import experience. The owner should weigh in on whether adding another module is worth the cleaner domain boundaries.

---

## What We Should Stop Doing

1. **Stop writing planning docs and start fixing bugs.** We have 2 execution plans (73 tasks + 83 tasks) and 0 fixes applied. The Pareto analysis says items 1-3 take 30 minutes and deliver 51% of the value. Execute first, plan second.

2. **Stop adding features until the bugs are fixed.** The type system unification, shared Message interface, and architecture improvements are important but NOT urgent. Production bugs are urgent.

---

## git Status

```
On branch master
Changes not staged for commit:
  (use "git add/rm <file>..." to update what will be committed)
  - docs/status/2026-06-02_14-12_COMPREHENSIVE-STATUS.md (new)

Untracked files:
  (none)
```

---

_Waiting for instructions._
