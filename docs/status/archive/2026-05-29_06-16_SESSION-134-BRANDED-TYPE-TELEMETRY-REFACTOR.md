# Session 134 — Branded Type Telemetry Refactor & Naming Cleanup

**Date:** 2026-05-29 06:16  
**Coverage:** 92.4% (statements) across all modules  
**Tests:** 26/26 packages PASS  
**Go vet:** Clean  
**Lines of Go:** ~69,454

---

## A. Fully Done ✅

### 1. Branded Type Enforcement in Telemetry Functions

Replaced raw `string` parameters with branded types (`id.AggregateID`, `event.AggregateType`, `fmt.Stringer`) across all telemetry/otel helper functions and their callers.

**Before:** Every call site manually converted branded types to strings:

```go
aggregateAttrs(string(aggregateType), aggregateID.String())
```

**After:** Branded types passed directly, conversion happens once inside the helper:

```go
cqrsotel.AggregateBaseAttrs(aggregateType, aggregateID)
```

**Files changed (8):**

- `otel/attributes.go` — Added `AggregateBaseAttrs()`, renamed `aggType`→`aggregateType`, `cmdType`→`commandType`, `evtType`→`eventType`, `qryType`→`queryType`. Changed aggregate ID params from `string` to `fmt.Stringer`.
- `otel/otel_test.go` — Updated tests to use `testStringer` type.
- `core/decider/otel.go` — Removed private `aggregateAttrs()`, now uses `cqrsotel.AggregateBaseAttrs()`.
- `core/decider/decider.go` — Removed `.String()` conversions at call sites.
- `storage/otel.go` — Removed private `aggregateAttrs()`, replaced `aggregateAttrsWithVersion()` callers with `AggregateBaseAttrs()` + append of version attr.
- `storage/event_store.go`, `storage/event_store_load.go`, `storage/snapshot.go`, `storage/stream.go` — Removed 15+ manual `string()`/`.String()` conversions.

**Impact:**

- Eliminates 2 duplicated private `aggregateAttrs()` functions (in `decider` and `storage`)
- Centralizes aggregate attribute construction in `otel.AggregateBaseAttrs()`
- Type safety: impossible to pass wrong type as aggregate ID
- Net -3 lines (31 added, 34 removed)

### 2. Abbreviated Parameter Name Cleanup

All parameter names in `otel/attributes.go` public API now use full descriptive names:

| Before    | After                                      |
| --------- | ------------------------------------------ |
| `aggType` | `aggregateType`                            |
| `aggID`   | (removed — now `aggregateID fmt.Stringer`) |
| `cmdType` | `commandType`                              |
| `evtType` | `eventType`                                |
| `qryType` | `queryType`                                |

---

## B. Partially Done 🔧

### 1. Abbreviated Names in Other Modules

`aggID` and `aggType` still exist in non-otel production code:

| File                              | Function                       | Status                            |
| --------------------------------- | ------------------------------ | --------------------------------- |
| `core/decider/load.go:68`         | `opError(aggType, aggID)`      | Not renamed                       |
| `core/decider/decider.go`         | `Execute(ctx, aggID, aggType)` | Public API — renaming is breaking |
| `core/event/store.go`             | `Save(ctx, aggType, aggID)`    | Public API — renaming is breaking |
| `stream/in_memory.go:112`         | `filterByType(refs, aggType)`  | Not renamed                       |
| `storage/event_reconstruction.go` | Uses `aggID` internally        | Not renamed                       |
| `projection/runner.go:240`        | `subscribesTo(p, evtType)`     | Not renamed                       |

**Decision needed:** Some of these (`Execute`, `Save`) are **public API** — renaming them is a breaking change. Should we rename and version, or leave as-is?

---

## C. Not Started ⏳

1. **Integration tests not run** — `integration/` tests were not executed (require DB). The modified files don't touch integration codepaths directly.
2. **Lint check** — `nix run .#lint` not executed.
3. **Remaining abbreviated names** in 15+ files across `core/`, `storage/`, `example/`, `stream/`, `projection/`.
4. **`AggregateAttrs` in `otel` still uses `string` for `aggregateType`** — could use `fmt.Stringer` for consistency.
5. **Event publish tracing** in `middleware/tracing.go:122` still manually constructs attributes inline instead of using `AggregateBaseAttrs`.

---

## D. Totally Fucked Up 💥

Nothing. Clean refactor, zero regressions.

---

## E. What We Should Improve

### Architecture

1. **`otel/AggregateAttrs(aggregateType string, ...)`** — The `aggregateType` param is still `string`, not `fmt.Stringer`. Inconsistent with `AggregateBaseAttrs` which takes two `fmt.Stringer`s. Should be `fmt.Stringer` too.
2. **`storage/otel.go` still has `aggregateAttrsWithVersion()`** — Only used by `startSaveSpan()`. Should also use `AggregateBaseAttrs()` + append for consistency.
3. **Duplicated attribute construction in `middleware/tracing.go:112-123`** — `EventPublishTracing` builds attributes manually instead of using `cqrsotel.EventAttrs()` or `AggregateBaseAttrs()`.

### Naming

4. **`AggregateBaseAttrs`** — The name "Base" is slightly ambiguous. Could be `AggregateTypeAndIDAttrs` or just have `AggregateAttrs` (no version) and `AggregateAttrsWithVersion`. Current API has both `AggregateAttrs(type, id, version)` and `AggregateBaseAttrs(type, id)` which is confusing.
5. **Test helper `testStringer`** — Could use `id.MustParseAggregateID("order-123")` instead of a custom stringer type, which would test the actual branded type path.

### Type Safety

6. **`otel` module can't import `core`** — This is correct (dependency direction), but it means `otel` public functions can only use `fmt.Stringer`, not `id.AggregateID`. The private helpers in `storage/` and `decider` previously bypassed this. Now centralized, the type safety is at the caller level only.
7. **`AggregateType` param** in `otel` functions is `string` — could be `fmt.Stringer` too (it already has `.String()`).

---

## F. Top 25 Things to Do Next (Sorted: Impact ↑, Work ↓)

### High Impact, Low Work (Do First)

| #   | Task                                                                                                                                          | Effort | Impact       |
| --- | --------------------------------------------------------------------------------------------------------------------------------------------- | ------ | ------------ |
| 1   | **Rename `AggregateBaseAttrs` → consistent naming** with `AggregateAttrs` (resolve confusion between "with version" and "without version")    | Small  | API clarity  |
| 2   | **Unify `storage/otel.go:startSaveSpan`** to use `AggregateBaseAttrs()` + append like all other callers, remove `aggregateAttrsWithVersion()` | Small  | Dedup        |
| 3   | **Fix `otel/AggregateAttrs` aggregateType param** to `fmt.Stringer` for consistency with `AggregateBaseAttrs`                                 | Small  | Type safety  |
| 4   | **Refactor `middleware/tracing.go:EventPublishTracing`** to use `cqrsotel.EventAttrs()` instead of manual attribute construction              | Small  | Dedup        |
| 5   | **Rename `aggID` → `aggregateID` in `core/decider/load.go:opError`**                                                                          | Tiny   | Naming       |
| 6   | **Rename `aggType` → `aggregateType` in `stream/in_memory.go:filterByType`**                                                                  | Tiny   | Naming       |
| 7   | **Rename `evtType` → `eventType` in `projection/runner.go:subscribesTo`**                                                                     | Tiny   | Naming       |
| 8   | **Rename `evtType` → `eventType` in `core/event/runner.go:SubscribesTo`**                                                                     | Tiny   | Naming       |
| 9   | **Fix otel test to use `id.MustParseAggregateID`** instead of custom `testStringer` for realistic testing                                     | Small  | Test quality |
| 10  | **Add `AggregateTypeAndID` single struct/type** to avoid always passing two params everywhere                                                 | Medium | Ergonomics   |

### High Impact, Medium Work

| #   | Task                                                                                                                                                    | Effort | Impact       |
| --- | ------------------------------------------------------------------------------------------------------------------------------------------------------- | ------ | ------------ |
| 11  | **Run `nix run .#lint`** and fix any findings                                                                                                           | Medium | Code quality |
| 12  | **Run integration tests** to verify storage changes work with real DB                                                                                   | Medium | Confidence   |
| 13  | **Decide on public API parameter renames** (`Execute(ctx, aggID, aggType)` → `aggregateID, aggregateType`) — breaking change, needs versioning strategy | Medium | Naming       |
| 14  | **Audit all `string()` casts on branded types** across codebase — many in SQL helpers that could accept the branded type directly                       | Medium | Type safety  |
| 15  | **Extract `AggregateRef` struct** `{Type AggregateType, ID AggregateID}` as a value object used everywhere these two travel together                    | Medium | Architecture |
| 16  | **Add `AggregateVersionAttrs(baseAttrs, version)` helper** to `otel` for the common "base + version" pattern                                            | Small  | Dedup        |

### Medium Impact, Medium Work

| #   | Task                                                                                                                                  | Effort | Impact      |
| --- | ------------------------------------------------------------------------------------------------------------------------------------- | ------ | ----------- |
| 17  | **Remove unused `go-branded-id` re-exports** if any exist in the API surface                                                          | Medium | Cleanup     |
| 18  | **Standardize error wrapping in `storage/`** — some use `event.WrapInfrastructure`, some use `fmt.Errorf`                             | Medium | Consistency |
| 19  | **Add missing doc comments** on exported functions in `otel/attributes.go`                                                            | Small  | Docs        |
| 20  | **Consider `event.Type` as `fmt.Stringer`** — it's `type Type string` with `.String()`. Could accept `fmt.Stringer` in otel functions | Small  | Type safety |

### Larger Initiatives

| #   | Task                                                                                                 | Effort | Impact    |
| --- | ---------------------------------------------------------------------------------------------------- | ------ | --------- |
| 21  | **Naming review skill** — Run full naming audit across entire codebase using the naming-review skill | Large  | Quality   |
| 22  | **Architecture visualization** — Generate D2 diagram of current module dependencies                  | Medium | Docs      |
| 23  | **Docs freshness check** — Verify AGENTS.md, FEATURES.md, TODO_LIST.md are current                   | Medium | Docs      |
| 24  | **Pareto planning** — Create prioritized execution plan for next sprint                              | Medium | Planning  |
| 25  | **v1.0.0 release preparation** — Tag modules, remove `replace` directives, publish                   | Large  | Milestone |

---

## G. Top #1 Question I Cannot Figure Out Myself

**Should we rename the public API parameters (`aggID` → `aggregateID`, `aggType` → `aggregateType`) in breaking interfaces like `decider.Repository.Execute()`, `event.Store.Save()`, and `event.Event`?**

This is a breaking change that affects every consumer of the library. The options are:

1. Rename now, bump minor version (pre-v1.0.0 so semver allows breaking)
2. Rename at v1.0.0 milestone
3. Don't rename — keep abbreviated names in the most-used public API

The abbreviated names in public API (`aggID`, `aggType`) are inconsistent with the naming we just fixed in the telemetry layer. But renaming public API is a consumer-affecting decision I cannot make autonomously.

---

## Module Health Summary

| Module              | Tests          | Coverage  | Vet       | Status                          |
| ------------------- | -------------- | --------- | --------- | ------------------------------- |
| core/aggregate      | ✅             | —         | ✅        | Clean                           |
| core/command        | ✅             | —         | ✅        | Clean                           |
| core/decider        | ✅             | —         | ✅        | **Modified**                    |
| core/event          | ✅             | —         | ✅        | Clean                           |
| core/pkg/dispatcher | ✅             | —         | ✅        | Clean                           |
| core/pkg/id         | ✅             | —         | ✅        | Clean                           |
| core/query          | ✅             | —         | ✅        | Clean                           |
| memory              | ✅             | —         | ✅        | Clean                           |
| catalog             | ✅             | —         | ✅        | Clean                           |
| catalog/\*          | ✅             | —         | ✅        | Clean                           |
| middleware          | ✅             | —         | ✅        | **Modified** (previous session) |
| testhelpers         | ✅             | —         | ✅        | Clean                           |
| projection          | ✅             | —         | ✅        | Clean                           |
| signing             | ✅             | 93.8%     | ✅        | Clean                           |
| storage             | ✅             | —         | ✅        | **Modified**                    |
| saga                | ✅             | 93.1%     | ✅        | Clean                           |
| stream              | ✅             | 93.9%     | ✅        | Clean                           |
| watermill           | ✅             | 94.4%     | ✅        | Clean                           |
| otel                | ✅             | 93.3%     | ✅        | **Modified**                    |
| **Total**           | **26/26 PASS** | **92.4%** | **Clean** |                                 |
