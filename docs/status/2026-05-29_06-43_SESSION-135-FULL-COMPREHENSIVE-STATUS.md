# Session 135 — Full Comprehensive Status Report

**Date:** 2026-05-29 06:43 CEST
**Status:** All 31 packages build + test pass. 0 clones at t=35 and t=40.

---

## A) FULLY DONE ✅

### 1. Deduplication Sprint — ZERO Clones Achieved

| Clone | Location | Resolution |
|-------|----------|------------|
| Clone 8 | storage/event_store.go + transactional_store.go | Both now call `startSaveSpan()` helper |
| Clone 10 | testhelpers/handlers.go:151-152 | Extracted `queryHandler` type alias |
| Clone 11 | core/decider/otel.go + storage/otel.go | Shared `cqrsotel.AggregateAttrs()` in otel module |
| Clone 12 | middleware/tracing.go EventPublishTracing | Uses `cqrsotel.EventAttrs()` |
| Clone 13 | FakeStore 9× mutex-read-if-nil pattern | Generic `getOverride[T any]` helper |
| Query dispatcher | command/query registration | Extracted `registerHandler` shared helper |

### 2. OTEL Attribute Consolidation

| Change | Files | Lines |
|--------|-------|-------|
| Added `AggregateAttrs()` to otel/attributes.go | 1 | +8 |
| Removed local `aggregateAttrs()` from decider/otel.go | 1 | -10 |
| Removed local `aggregateAttrs()` from storage/otel.go | 1 | -7 |
| Updated all callers | 6 | net -3 |

### 3. Deprecated Alias Migration (Internal)

| From | To | Files |
|------|-----|-------|
| `event.Codec` | `codec.Codec` | core/decider/decider.go, options.go |

### 4. Self-Replace Directive Cleanup

Removed redundant `replace <module> => ./` from:
- otel/go.mod
- testhelpers/go.mod
- catalog/go.mod

### 5. Naming Consistency

Renamed parameters for clarity:
- `aggType` → `aggregateType`
- `aggID` → `aggregateID`
- `evtType` → `eventType`

---

## B) PARTIALLY DONE / WORK IN PROGRESS ⚠️

### 1. Deprecated Alias Migration (External Consumers)

| Alias | Callers Remaining | Priority |
|-------|-------------------|----------|
| `event.JSONCodec{}` | 17 test files, 3 example files | Low |
| `event.Codec` | 5 test files | Low |

These are type aliases (`type JSONCodec = codec.JSONCodec`) so they work fine — migration is cosmetic hygiene.

### 2. `TransactionalStore` → `TransactionalSink`

| Location | Status |
|----------|--------|
| `storage/transactional_store.go:117` | Backward-compat assertion kept intentionally |
| `storage/sql_backend.go:99` | Return type still `event.TransactionalStore` for API compat |

Cannot remove without breaking consumers. Marked for v2.0.

### 3. `core/aggregate/` Package

Entire package is deprecated in favor of `core/decider`. Tests still reference it for backward compat verification. Safe to keep.

---

## C) NOT STARTED ❌

### 1. Linter Issues (10 total)

| Tool | Count | Example |
|------|-------|---------|
| wrapcheck | 2 | `core/aggregate/aggregate.go:46,59` — errors from decider.NewRepository not wrapped |
| staticcheck | 7 | Deprecated API usages (intentional backward compat) |
| nlreturn | 1 | Missing blank line before return |

### 2. Command/Query Dispatcher Unification

Both dispatchers are ~80% structurally identical. Could be unified into `Dispatcher[Req, Res]` generic. Not started — needs design consideration for typed vs untyped paths.

### 3. Branded ID Boilerplate

7 files (`core/pkg/id/*.go`) with identical `New/Parse/MustParse` pattern per type. Candidate for code generation or generic helper.

### 4. `t.Parallel()` Coverage

~30 test files missing `t.Parallel()`. No functional impact — test runtime optimization.

### 5. `event.Runner` Deprecation

Marked deprecated in favor of `projection.Runner`. No production callers found — safe to remove in v2.0.

### 6. go.work Workspace Build

`go build ./...` fails with workspace pattern error. `GOWORK=off go build ./...` works (CI approach). Root cause: workspace resolution for multi-module patterns.

---

## D) TOTAL LINES OF CODE

| Category | Lines |
|----------|-------|
| Production code | 24,258 |
| Test code | 45,115 |
| **Total** | **69,373** |
| Test/Production ratio | 1.86:1 |

---

## E) WHAT WE SHOULD IMPROVE (Top 25)

| # | Item | Effort | Impact | Customer Value |
|---|------|--------|--------|----------------|
| 1 | **Generic `Dispatcher[Req, Res]`** — unify command/query | High | High | Less API surface |
| 2 | **Branded ID code generation** — eliminate 7 boilerplate files | Medium | Medium | Maintainability |
| 3 | **Remove `core/aggregate/` package** — fully deprecated | Low | Low | Clean API (breaking) |
| 4 | **Remove `event.Runner`** — fully deprecated | Low | Low | Clean API (breaking) |
| 5 | **Migrate `event.JSONCodec{}` in examples** | Low | Low | Hygiene |
| 6 | **Add `t.Parallel()` to 30 test files** | Low | Low | Faster CI |
| 7 | **Fix `wrapcheck` in `core/aggregate/`** | Low | Low | Lint clean |
| 8 | **Fix `nlreturn` lint issue** | Low | Low | Lint clean |
| 9 | **Remove deprecated type aliases (v2.0)** | Low | Low | Clean API (breaking) |
| 10 | **OTEL builder pattern** — `Attrs().Aggregate(t,id).Version(v)` | Medium | Medium | Ergonomics |
| 11 | **Unify `event.New` and `NewEvent`** — single constructor | Medium | Medium | Simpler API |
| 12 | **Extract `clock`/`newCodec` from `ImmutableEvent`** | Medium | Medium | Cleaner domain model |
| 13 | **Snapshot state caching** — avoid double fold | Medium | Medium | Performance |
| 14 | **Generic `getOverride` pattern** — apply to other fakes | Low | Low | Consistency |
| 15 | **SQLTransactionalStore embedding fix** — hide non-transactional Save | High | High | Safety |
| 16 | **Codec-aware batch operations** | Medium | Medium | Feature parity |
| 17 | **Projection checkpoint batching** | High | High | Performance |
| 18 | **Event upcasting integration tests** | Medium | Medium | Confidence |
| 19 | **Saga timeout handling** | Medium | Medium | Reliability |
| 20 | **Outbox polling backoff strategy** | Medium | Medium | Reliability |
| 21 | **Storage dialect auto-detection** | Medium | Medium | DX |
| 22 | **Memory store snapshot support** | Low | Medium | Test parity |
| 23 | **Catalog live reload** | High | Low | DX |
| 24 | **OpenAPI 3.1 support** | Medium | Low | Standards compliance |
| 25 | **gRPC protocol adapter** | High | Low | New transport |

---

## F) Top #1 Question I Can't Answer

**Should we maintain backward compatibility with deprecated aliases indefinitely, or set a v2.0 timeline?**

The project has 13 deprecated items across 5 files. All are type aliases or interface assertions that cost nothing at runtime but add cognitive load. The `replace` directives in go.mod files mean no consumers can use this as a published library yet (requires v1.0.0 tags). This creates a window where we could clean up deprecated APIs before first stable release.

**Tradeoffs:**
- Keep aliases: Zero migration cost for early adopters, but API clutter forever
- Remove aliases: Clean API, but any existing users must update

Since the project is pre-1.0 and replace-directive-bound, I'd recommend removing deprecated APIs now. But this is a product decision, not a technical one.

---

## G) Metrics

| Metric | Value |
|--------|-------|
| Modules | 21 |
| Packages | 31 testable |
| Test files | 229 |
| Production files | ~280 |
| Coverage | 84–100% (target: 80%+) |
| art-dupl t=35 | 0 clone groups ✅ |
| art-dupl t=40 | 0 clone groups ✅ |
| Build | ✅ Pass |
| Tests | ✅ 31/31 pass |
| Lint | 10 issues (mostly intentional deprecations) |

---

*Generated: 2026-05-29 06:43 CEST*
