# V2.0.0 Post-Release Deep Audit — Session 10

**Date:** 2026-06-02 13:56 CEST
**Session:** 10 (post-release deep audit + execution planning)
**Since Last Report:** `af43ecd7` → current (clean tree, no new commits)

---

## Executive Summary

**v2.0.0 is published but has real production bugs.** The release itself was successful — all 23 modules tagged with `/v2` semantic import paths, 38/38 tests pass, 0 lint issues, `GOWORK=off` builds verified. But a deep audit reveals **4 CRITICAL correctness issues**, 3 HIGH-priority context/race bugs, and multiple MEDIUM improvements that should be addressed before consumers rely on this in production.

The audit also identified significant **type system duplication** (3 parallel `Type` string types, duplicated metadata options, near-identical dispatchers) and **130+ `fmt.Errorf` calls** that should use the error taxonomy.

---

## a) FULLY DONE

### v2.0.0 Release (Sessions 8-10)

- [x] Generic middleware refactoring (27→9 functions)
- [x] Circuit breaker double-wrapping fix
- [x] command.Metadata split brain fix
- [x] watermill metadata error swallowing fix
- [x] schema.VersionedStore exposure fix
- [x] Testify → Gomega migration (5 files)
- [x] Ghost reactive files removed
- [x] `/v2` semantic import path migration (426 files)
- [x] 23 module tags + 1 root tag pushed to remote
- [x] `GOWORK=off` per-module builds verified (20/20 pass)
- [x] go.sum files regenerated for per-module CI
- [x] TODO_LIST.md + AGENTS.md updated

### Code Quality Baseline

| Metric                     | Value                                    |
| -------------------------- | ---------------------------------------- |
| Test packages              | 38/38 GREEN                              |
| Lint issues                | 0                                        |
| Build errors               | 0                                        |
| Testify imports            | 0                                        |
| TODO/FIXME/HACK comments   | 0                                        |
| Stale `core/` paths        | 0                                        |
| `//go:generate` directives | 0                                        |
| `slog` consistency         | ✅ All modules use slog                  |
| `slices`/`maps` usage      | ✅ stdlib, not x/exp                     |
| `errors.Join` usage        | ✅ stdlib, not multierror                |
| Total Go LOC               | 62,727 (23,376 production + 39,351 test) |
| Average coverage           | 89.4%                                    |
| `//nolint` directives      | ~120 (mostly justified)                  |

---

## b) PARTIALLY DONE

| Item                        | State                                             | Gap                                  |
| --------------------------- | ------------------------------------------------- | ------------------------------------ |
| **turso module**            | 28.6% coverage                                    | EventStore untested                  |
| **storage module**          | 72.7% coverage                                    | Aggregate reader, error paths        |
| **Replace directives**      | Retained for GOWORK=off                           | Needed for local dev (not removable) |
| **Error taxonomy adoption** | Framework exists, ~130+ `fmt.Errorf` not migrated | Core packages use ad-hoc wrapping    |

---

## c) NOT STARTED

### Type System Unification

- [ ] Shared `TypeName() string` interface across command/event/query (eliminates `MessageAdapter`)
- [ ] Unified `Type` branded string in shared package
- [ ] Unified metadata options (4 duplicated functions)
- [ ] Generic `BrandedInt[T]` for `Version`/`SchemaVersion`
- [ ] `query.Handler` generic `TypedHandler[T]` returning `(T, error)`

### Testing & Quality

- [ ] `errgroup.Group.SetLimit` for `dispatchParallel`
- [ ] `filepath.WalkDir` migration (2 files)
- [ ] Turso meaningful test coverage
- [ ] Storage test coverage improvement
- [ ] Benchmark storage backends

### Documentation

- [ ] ROADMAP.md
- [ ] Documentation site

---

## d) TOTALLY FUCKED UP

### 🔴 CRITICAL — Production Bugs

| #   | Issue                                                                                                                        | File                               | Impact                                                |
| --- | ---------------------------------------------------------------------------------------------------------------------------- | ---------------------------------- | ----------------------------------------------------- |
| 1   | **Data race on `Runner.cancel`** — `Run()` writes, `Close()` reads without synchronization                                   | `projection/runner.go:106,230`     | Panic, corrupted state, cancelled wrong context       |
| 2   | **Data race on `Runner.projections`** — `Register()` appends, `Run()` reads without mutex                                    | `projection/runner.go:85,99`       | Panic, dropped projections, corrupted slice           |
| 3   | **`HealthCheck` loads ALL events** — `ReadAll()` on full event store                                                         | `projection/health.go:20-22`       | **OOM in production** with millions of events         |
| 4   | **`ReadFrom` pagination broken** — `WHERE id > ? ORDER BY occurred_at ASC` is semantically wrong for cursor-based pagination | `storage/event_store_global.go:66` | **Skipped or duplicated events** in projection replay |

### 🟠 HIGH — Context/Race Bugs

| #   | Issue                                                                                     | File                                    | Impact                              |
| --- | ----------------------------------------------------------------------------------------- | --------------------------------------- | ----------------------------------- |
| 5   | **`PublisherAdapter` drops context** — `context.Background()` breaks tracing/cancellation | `watermill/publisher.go:24`             | Broken OTel spans, lost deadlines   |
| 6   | **`SQLAggregateReader` hardcodes `?`** — SQLite placeholder breaks PostgreSQL             | `storage/sql_aggregate_reader.go:63-86` | Runtime error on Postgres           |
| 7   | **`SubscriberAdapter.handlers` data race** — no mutex on map                              | `watermill/subscriber.go:58`            | Panic on concurrent subscribe+close |

### 🟡 MEDIUM — Design Issues

| #   | Issue                                                                                   | File                                  | Impact                                  |
| --- | --------------------------------------------------------------------------------------- | ------------------------------------- | --------------------------------------- |
| 8   | **Pebble `EventStore` has no `Close()`** — DB never closed, locks grow forever          | `pebble/store.go`                     | Resource leak in long-running processes |
| 9   | **`subscribeLive` never unsubscribes handler** — stale handler on bus after stop        | `projection/runner_live.go:19`        | Handler fires on stopped runner         |
| 10  | **`SQLEventStore` no closed state** — operations after Close hit cryptic errors         | `storage/event_store.go:55-61`        | Confusing error messages                |
| 11  | **`LoadAtVersion` snapshot doesn't filter in SQL** — loads latest, checks version in Go | `storage/snapshot.go:109`             | Misleading name, wasted DB round-trips  |
| 12  | **`createTable()` uses `context.Background()`** — DDL can't be cancelled                | `storage/aggregate_projection.go:100` | Unstoppable table creation              |
| 13  | **Pervasive `time.Sleep` in tests** — 15+ occurrences                                   | Multiple test files                   | Flaky CI under load                     |

---

## e) WHAT WE SHOULD IMPROVE

### Architecture Improvements

1. **Shared `Message` interface** — `command.Type`, `event.Type`, `query.Type` are all `type X string` with identical methods. A shared interface `TypeName() string` would eliminate the `MessageAdapter` bridge in middleware/generic.go and enable compile-time safety.

2. **Unified metadata options** — `WithCorrelationID`, `WithCausationID`, `WithUserID`, `WithRequestID` are duplicated across event/ and command/ with identical implementations targeting different structs. A `MetadataCarrier` interface would collapse 8 functions to 4.

3. **Generic `BrandedInt[T]`** — `event.Version` and `event.SchemaVersion` share ~80% of their methods (Int(), String(), IsZero(), IsPositive(), Increment(), Decrement(), Cmp()). A generic base type would eliminate ~60 lines of duplication.

4. **`query.Handler` generic return** — Currently returns `(any, error)`. Should be `TypedHandler[T any] func(ctx, Query) (T, error)` for type safety. This is the single biggest type safety gap.

### Code Quality

5. **Error taxonomy adoption** — 130+ `fmt.Errorf` calls across production code should use `event.Wrap*` / `event.NewRejection` etc. Biggest offenders: `decider/load.go`, `storage/`, `watermill/protocol.go`, `schema/versioned_source.go`.

6. **`errgroup` for parallel dispatch** — `projection/runner_live.go:54-77` uses manual semaphore + WaitGroup. `errgroup.Group.SetLimit(n)` would reduce this to ~5 lines.

7. **`filepath.WalkDir` migration** — 2 files still use deprecated `filepath.Walk`.

### Library Choices

8. **Circuit breaker** — Custom impl is adequate and domain-integrated. `gobreaker` would add dependency for marginal benefit. **Keep custom.**
9. **`slices`/`maps`** — Already on stdlib versions. ✅
10. **`errors.Join`** — Already used. ✅
11. **`log/slog`** — Consistent across all modules. ✅

---

## f) Top #25 Things to Get Done Next

Sorted by Impact × Effort (highest first):

| #   | Task                                                             | Impact   | Effort | Category     |
| --- | ---------------------------------------------------------------- | -------- | ------ | ------------ |
| 1   | **Fix Runner.cancel data race** (add mutex or atomic)            | CRITICAL | LOW    | Bug          |
| 2   | **Fix Runner.projections data race** (mutex around Register+Run) | CRITICAL | LOW    | Bug          |
| 3   | **Fix HealthCheck ReadAll → use checkpoint-only check**          | CRITICAL | LOW    | Bug          |
| 4   | **Fix ReadFrom pagination** (cursor-based with proper ordering)  | CRITICAL | MED    | Bug          |
| 5   | **Fix PublisherAdapter drops context**                           | HIGH     | LOW    | Bug          |
| 6   | **Fix SQLAggregateReader `?` → use Dialect.Placeholder**         | HIGH     | LOW    | Bug          |
| 7   | **Fix SubscriberAdapter.handlers data race** (add mutex)         | HIGH     | LOW    | Bug          |
| 8   | **Add Pebble Close() method** (close DB, cleanup locks)          | MED      | LOW    | Bug          |
| 9   | **Add shared `Message` interface** across command/event/query    | HIGH     | MED    | Architecture |
| 10  | **Replace dispatchParallel manual semaphore → errgroup**         | MED      | LOW    | Quality      |
| 11  | **Fix LoadAtVersion snapshot to filter in SQL**                  | MED      | LOW    | Bug          |
| 12  | **Fix createTable context.Background() → use caller ctx**        | MED      | LOW    | Bug          |
| 13  | **Migrate error taxonomy** (decider, storage, watermill, schema) | MED      | MED    | Quality      |
| 14  | **Fix subscribeLive handler leak** (unsubscribe on stop)         | MED      | MED    | Bug          |
| 15  | **Add SQLEventStore closed state tracking**                      | MED      | LOW    | Quality      |
| 16  | **Unified metadata options** (4 dupes → shared)                  | MED      | MED    | Architecture |
| 17  | **`filepath.Walk` → `filepath.WalkDir`** (2 files)               | LOW      | LOW    | Quality      |
| 18  | **Generic BrandedInt[T]** for Version/SchemaVersion              | MED      | MED    | Architecture |
| 19  | **`query.TypedHandler[T]`** generic return type                  | HIGH     | HIGH   | Architecture |
| 20  | **Fix flaky time.Sleep tests** (15+ occurrences)                 | MED      | MED    | Quality      |
| 21  | **Turso test coverage** 28.6% → 70%+                             | MED      | MED    | Quality      |
| 22  | **Storage test coverage** 72.7% → 85%+                           | MED      | MED    | Quality      |
| 23  | **Benchmark storage backends** (PG vs SQLite vs Pebble)          | MED      | MED    | Quality      |
| 24  | **LSP diagnostic cleanup** (12 hints/infos)                      | LOW      | LOW    | Quality      |
| 25  | **ROADMAP.md creation**                                          | LOW      | LOW    | Docs         |

---

## g) Top #1 Question I Cannot Figure Out Myself

**Should the `Message` interface live in a new shared package or in `event/`?**

Options:

- **(A) New `internal/message` package** — Clean separation, but adds a package layer. Not importable by consumers.
- **(B) New top-level `message/` module** — Consumers can use it. But adds another module to the workspace (31 modules).
- **(C) In `event/`** — Already the foundational package. command/ and query/ already depend on event/. But it's semantically weird for "query" to import an interface from "event".
- **(D) In `dispatcher/`** — Already imported by command/ and query/. But dispatcher/ is about dispatch mechanics, not message typing.

**Recommendation:** Option (B) — a lightweight `message/` module with just the `Message` interface, `Type` branded string, and potentially shared metadata types. It's a leaf module (no internal deps), provides clean domain vocabulary, and consumers can import just what they need.

---

## What I Forgot / Could Have Done Better

1. **Testing the tags before declaring done** — The first v2.0.0 tag push had wrong module paths (no `/v2`). I had to delete tags, fix imports, and re-push. Should have tested `GOWORK=off go build` before tagging.

2. **The replace block formatting** — The shell script to add replace directives produced malformed go.mod files (missing closing `)`, double `)`). Required 3 fix-up passes. Should have used a proper Go-based tool or `go mod edit`.

3. **Not catching the production bugs earlier** — The Runner data races, HealthCheck OOM, and ReadFrom pagination bug existed before the v2.0.0 release. They passed tests because tests don't exercise concurrent Close()+Run() or large event stores. Should have run `-race` flag more systematically.

4. **The `/v2` migration scope** — 426 files changed in one commit is risky. Should have done it in smaller batches (e.g., leaf modules first, then dependents) with tests after each batch.

---

_Waiting for instructions._
