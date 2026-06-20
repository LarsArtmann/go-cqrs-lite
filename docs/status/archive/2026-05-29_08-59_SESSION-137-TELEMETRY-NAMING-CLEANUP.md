# Session 137 — Full Comprehensive Status Report

**Date:** 2026-05-29 08:59  
**Session Scope:** Tasks 1-10 from telemetry/naming cleanup backlog (Session 134 follow-up)  
**Previous:** Session 136 (deduplication complete, codec migration)

---

## Executive Summary

Tasks 1-8 were **already completed** in prior session commits (08117cd, 313c6b0, d0d7f48).  
Task 9 correctly skipped (circular dependency constraint).  
Task 10 partially done — `event.AggregateRef` type added but not yet integrated into interfaces.

Additionally: 6 files with uncommitted codec migration changes (`event.JSONCodec` → `codec.JSONCodec` in tests and example/todo) need committing.

---

## Build & Test Status

| Metric       | Status                              |
| ------------ | ----------------------------------- |
| **Build**    | ✅ All 14 modules compile           |
| **Tests**    | ✅ 28/28 packages pass (0 failures) |
| **go vet**   | ✅ Clean                            |
| **Coverage** | 82–100% across production packages  |
| **Race**     | Not run this session                |

### Per-Package Coverage

| Package             | Coverage |
| ------------------- | -------- |
| core/command        | 94.2%    |
| core/decider        | 100.0%   |
| core/event          | 90.9%    |
| core/pkg/dispatcher | 92.2%    |
| core/pkg/id         | 100.0%   |
| core/query          | 96.8%    |
| storage             | 90.4%    |
| stream              | 93.9%    |
| otel                | 96.6%    |
| middleware          | 94.0%    |
| projection          | 89.5%    |
| saga                | 94.6%    |
| memory              | 99.6%    |
| testhelpers         | 82.3%    |
| signing             | 93.8%    |
| catalog             | 96.3%    |

### Codebase Size

| Category      | Lines  |
| ------------- | ------ |
| Production Go | 24,996 |
| Test Go       | 44,827 |
| Total         | 69,823 |

---

## Task Status: Session 134/136/137 Backlog

### a) FULLY DONE ✅

| #   | Task                                                     | How                                                                                                                                                                                                     |
| --- | -------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| 1   | Rename `AggregateBaseAttrs` → `AggregateAttrs`           | Merged old 3-arg `AggregateAttrs(type string, id, version)` and `AggregateBaseAttrs(type, id fmt.Stringer)` into single `AggregateAttrs(type, id fmt.Stringer)`. All 11 callers updated across 6 files. |
| 2   | Remove dead `aggregateAttrsWithVersion`                  | Dead function removed from `storage/otel.go`. `startSaveSpan` already used `AggregateAttrs` + append.                                                                                                   |
| 3   | Fix `AggregateAttrs` type consistency                    | `aggregateType` param changed from `string` to `fmt.Stringer` to match `AggregateBaseAttrs` pattern.                                                                                                    |
| 4   | Refactor `EventPublishTracing` → `cqrsotel.EventAttrs()` | `middleware/tracing.go:112-123` replaced 6 manual attribute constructions with single `cqrsotel.EventAttrs()` call.                                                                                     |
| 5   | Rename `aggID` → `aggregateID` in `opError`              | `core/decider/load.go:56` — function params and usage.                                                                                                                                                  |
| 6   | Rename `aggType` → `aggregateType` in `filterByType`     | `stream/in_memory.go:112` — function params and comparison.                                                                                                                                             |
| 7   | Rename `evtType` → `eventType` in `subscribesTo`         | `projection/runner.go:240` — function param and delegation.                                                                                                                                             |
| 8   | Rename `evtType` → `eventType` in `SubscribesTo`         | `core/event/runner.go:229` — exported function param and `slices.Contains`.                                                                                                                             |

### b) PARTIALLY DONE 🔶

| #   | Task                            | Status                           | Detail                                                                                                                                                                                                                                                                                                                                          |
| --- | ------------------------------- | -------------------------------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| 10  | Add `AggregateTypeAndID` struct | **Type defined, not integrated** | `core/event/aggregate_ref.go` defines `AggregateRef{Type AggregateType, ID id.AggregateID}` with `String()`, `StreamKey()`, `NewAggregateRef()`. `StreamKey()` function updated to use it. **NOT YET**: Interface changes (EventSink/EventSource/TransactionalSink still take separate params), internal function refactoring, test assertions. |

### c) NOT STARTED ⬜

See "Top 25 Next Tasks" below.

### d) TOTALLY FUCKED UP 💥

| #   | Issue                                                  | Detail                                                                                                                                                                                                                                               |
| --- | ------------------------------------------------------ | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| 9   | `otel` test `testStringer` → `id.MustParseAggregateID` | **Correctly skipped** — `otel` module cannot import `core/pkg/id` (circular dep: `core→otel→core/pkg/id`). `testStringer` is the right pattern for an isolated leaf module. The task description was wrong; this should never have been on the list. |

### e) WHAT WE SHOULD IMPROVE

1. **Uncommitted codec migration changes** — 6 files modified (`event.JSONCodec` → `codec.JSONCodec` in tests + example/todo) but never committed. These are from Session 135/136.

2. **Deprecated type aliases still referenced in docs** — `README.md`, `core/README.md`, and research docs still show `event.JSONCodec{}`. Should be updated to `codec.JSONCodec{}`.

3. **`AggregateRef` integration incomplete** — Type exists but interfaces still take `(aggregateType, aggregateID)` as separate params. This is the #1 ergonomics gap.

4. **Remaining abbreviated names** — Production code still uses `aggID`/`aggType` in:
   - `core/decider/decider.go` (Execute, Load public methods — these are PUBLIC API params, should be `aggregateID`/`aggregateType`)
   - `stream/sql_reader.go` (scan variables — internal, acceptable)
   - `stream/in_memory.go` (streamKey struct fields — internal, acceptable)
   - `signing/signer.go` (local variables — internal, acceptable)
   - `core/event/runner.go:127` (`matchingProjections` uses `evtType` — missed in task 8)

5. **Example modules not in go.work** — `example/` modules can't be tested via workspace root. Need `cd example/todo && go test`.

6. **No race condition testing this session** — Should run `-race` at least once.

7. **Deprecated API surface** — 8 deprecated types/interfaces still exist:
   - `event.Codec` → `codec.Codec`
   - `event.JSONCodec` → `codec.JSONCodec`
   - `event.GlobalLoader` → `event.Journal`
   - `event.PositionalLoader` → `event.SeekableJournal`
   - `event.BackwardsLoader` → `event.BackwardsSource`
   - `event.TransactionalStore` → `event.TransactionalSink`
   - `storage.LoadAll` → `ReadAll`
   - `storage.LoadAllFromPosition` → `ReadFrom`

---

## f) Top 25 Next Tasks

### High Impact, Low Work (Do First)

| #   | Task                                                                                                | Effort | Impact                |
| --- | --------------------------------------------------------------------------------------------------- | ------ | --------------------- |
| 1   | **Commit uncommitted codec migration changes** (6 files)                                            | Tiny   | Clean working tree    |
| 2   | **Rename `aggID`/`aggType` → `aggregateID`/`aggregateType` in `decider.Execute`/`Load` public API** | Small  | Public API naming     |
| 3   | **Rename `evtType` → `eventType` in `runner.matchingProjections`**                                  | Tiny   | Naming consistency    |
| 4   | **Update README.md + core/README.md to use `codec.JSONCodec` instead of `event.JSONCodec`**         | Small  | Doc accuracy          |
| 5   | **Run full test suite with `-race` flag**                                                           | Tiny   | Safety verification   |
| 6   | **Add `AggregateRef` to `EventSink`/`EventSource` interfaces** (method params)                      | Medium | Ergonomics revolution |
| 7   | **Update `ImmutableEvent` to embed or expose `AggregateRef`**                                       | Small  | Type unification      |
| 8   | **Refactor `decider.Repository` public methods to accept `AggregateRef`**                           | Medium | API ergonomics        |
| 9   | **Add `event.AggregateRef` to `opError` signature**                                                 | Small  | Internal consistency  |
| 10  | **Migrate `storage/` internal functions to use `AggregateRef`**                                     | Medium | Dedup params          |

### Medium Impact, Medium Work

| #   | Task                                                                                                   | Effort | Impact                   |
| --- | ------------------------------------------------------------------------------------------------------ | ------ | ------------------------ |
| 11  | **Remove deprecated type aliases** (`event.Codec`, `event.JSONCodec`, `event.GlobalLoader`, etc.)      | Medium | API surface reduction    |
| 12  | **Add `example/` modules to `go.work`** or document standalone testing                                 | Small  | DX improvement           |
| 13  | **Add `AggregateRef` integration tests**                                                               | Small  | Confidence in new type   |
| 14  | **Refactor `testhelpers` fake store function types to use `AggregateRef`**                             | Medium | Dedup test signatures    |
| 15  | **Add `otel.AggregateRefAttrs(ref AggregateRef)` for telemetry**                                       | Small  | DX for otel consumers    |
| 16  | **Add `StreamKey()` method on `ImmutableEvent`** (derived from AggregateType+AggregateID)              | Tiny   | Convenience              |
| 17  | **Consolidate `stream.AggregateRef` fields** — consider using `event.AggregateRef` for ID/Type         | Small  | Cross-module consistency |
| 18  | **Add code-generated Go doc examples** for `AggregateRef` usage                                        | Small  | Documentation            |
| 19  | **Review `storage/pebble_*.go` naming** — 3 files with `pebble_` prefix, could be `pebble/` subpackage | Medium | File organization        |
| 20  | **Add `cqrs-gen` support for generating `AggregateRef` constants**                                     | Medium | Code gen                 |

### Longer Term / Strategic

| #   | Task                                                                        | Effort | Impact                      |
| --- | --------------------------------------------------------------------------- | ------ | --------------------------- |
| 21  | **Remove `replace` directives in all `go.mod` files** (requires v1.0.0 tag) | Large  | Publish readiness           |
| 22  | **Add Go workspace `go.work` example for consumer projects**                | Small  | Adoption                    |
| 23  | **Create `CHANGELOG.md` with all session work**                             | Medium | Release preparation         |
| 24  | **Performance benchmarks for `AggregateRef` vs separate params**            | Small  | Evidence-based API          |
| 25  | **Audit all exported types for `fmt.Stringer` compliance**                  | Medium | Debugging/telemetry quality |

---

## g) Top #1 Question I Cannot Figure Out Myself

**Should `AggregateRef` replace `(aggregateType, aggregateID)` in the core interfaces (`EventSink.Save`, `EventSource.Load`, etc.) NOW, or wait for v1.0.0?**

This is a **breaking API change** for every consumer of the library. The interfaces `EventSink`, `EventSource`, `Store`, `TransactionalSink`, `BackwardsSource` all take `aggregateType AggregateType, aggregateID id.AggregateID` as separate params. Changing them to `ref AggregateRef` would:

- Break every implementation (`memory.MemoryStore`, `storage.SQLEventStore`, `testhelpers.FakeStore`, all consumer implementations)
- Break every caller (`decider.Repository`, `event.VersionedStore`, examples, tests — ~400+ call sites)
- Significantly improve ergonomics and reduce param-count bugs

**The question is**: Do we YOLO this as a breaking change in a pre-1.0 library, or do we add parallel methods (`LoadRef(ctx, AggregateRef)` alongside `Load(ctx, AggregateType, AggregateID)`) and deprecate the old signatures?

---

## Uncommitted Changes (6 files)

These need to be committed — they are codec migration changes from Session 135/136:

| File                                          | Change                                                                                          |
| --------------------------------------------- | ----------------------------------------------------------------------------------------------- |
| `core/event/benchmark_test.go`                | `event.JSONCodec{}` → `codec.JSONCodec{}`                                                       |
| `core/event/codec_test.go`                    | All 13 instances `event.JSONCodec{}` → `codecpkg.JSONCodec{}`, `event.Codec` → `codecpkg.Codec` |
| `core/event/snapshot_helper_test.go`          | 3 instances `event.JSONCodec{}` → `codec.JSONCodec{}`                                           |
| `example/todo/aggregate/todo.go`              | `event.JSONCodec{}` → `codecpkg.JSONCodec{}`                                                    |
| `example/todo/go.mod`                         | Added `codec` dependency + replace directive                                                    |
| `example/todo/projections/todo_projection.go` | `event.JSONCodec{}` → `codecpkg.JSONCodec{}`                                                    |

---

## Git Log (Last 10 Commits)

```
08117cd style: nix fmt applied formatting across codebase
313c6b0 style(example): remove unnecessary explicit type args from projection.On calls
d0d7f48 fix(decider,query,projection): fix all 3 broken test modules + codec migration
221ffca fix(projection): remove pointer-to-interface in testProjection helper
7a3a970 chore: formatting consistency and Go receiver call style cleanup
8012d9f docs(status): add Session 136 full comprehensive status report
b671d7a refactor: deduplicate code at threshold 25 — eliminate all actionable clones
677fa4a refactor(saga): extract nilActionStep helper to improve test readability
48f5d3d refactor: fix formatting inconsistencies and extract failingFold helper
390007d refactor(catalog): make Option generic for type-safe exporter configuration
```
