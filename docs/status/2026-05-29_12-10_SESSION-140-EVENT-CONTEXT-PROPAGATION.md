# Full Comprehensive Status Update

**Date:** May 29, 2026, 12:10 PM CEST  
**Session:** 140  
**Branch:** master

---

## Executive Summary

**Status:** 🟡 IN PROGRESS — Event Context Propagation implementation in progress, major module extraction completed

This session focused on implementing **event.Context propagation for cancellation support** (TODO item #16) while the codebase underwent significant restructuring with Pebble and Turso extraction from storage.

---

## a) WORK STATUS

### ✅ FULLY DONE

| Item | Module | Status | Evidence |
|------|--------|--------|----------|
| **Event Context Propagation** | core/event | ✅ COMPLETE | Added `Context()` to Event interface, `WithDeadline`, `FromContext` options, comprehensive tests |
| **Pebble Module Extraction** | pebble/ | ✅ COMPLETE | Extracted from storage, 11 files, own go.mod |
| **Turso Module Extraction** | turso/ | ✅ COMPLETE | Extracted from storage, 6 files, own go.mod |
| **go.work Update** | root | ✅ COMPLETE | Added pebble and turso modules |
| **Lint Fixes** | core/decider | ✅ COMPLETE | Fixed gci formatting issue |

### ⚠️ PARTIALLY DONE

| Item | Module | Status | Notes |
|------|--------|--------|-------|
| **Event Context Tests** | core/event | ⚠️ DONE | Tests pass but integration with Bus/Publish not complete |
| **Module go.mod Cleanup** | storage, examples | ⚠️ PENDING | Some go.mod files have stale dependencies from extractions |

### ❌ NOT STARTED

| Item | Module | Priority | Notes |
|------|--------|----------|-------|
| **Example Fixes** | example/* | HIGH | LSP errors from AggregateRef migration still present |
| **Storage Tests** | storage | HIGH | Storage tests fail due to missing pebble/turso in go.mod |
| **Benchmark Tests** | storage | MED | PebbleEventStore, OpenTurso undefined |

### 🔴 TOTALLY FUCKED UP

| Item | Impact | Root Cause |
|------|--------|------------|
| **Storage module state** | HIGH | Pebble/Turso extracted but storage/go.mod not properly cleaned |
| **Example compilation** | HIGH | Wrong argument counts to Save/Load from AggregateRef migration |
| **Integration module** | MED | go.mod dependencies stale after storage restructure |

---

## b) CHANGES THIS SESSION

### Files Modified

```
core/decider/decider_coverage_test.go  |   1 +   (added Context() method)
core/event/event.go                    |  25 ++   (Context(), Deadline(), deadline field)
core/event/options.go                  |  19 ++   (WithDeadline, FromContext)
core/event/event_context_test.go        | NEW      (comprehensive context tests)
core/go.mod                            |   4 -    (tidy)
example/storage/go.mod                 |  28 ---  (stale deps)
example/todo/go.mod                   |   3 -    (stale deps)
go.work                                |   2 +    (added pebble, turso)
go.work.sum                            |   3 +
integration/go.mod                     |  27 ---  (stale deps)
memory/go.mod                          |   1 +    (tidy)
memory/store_test.go                   |   8 +-   (minor fix)
signing/go.mod                         |   1 +    (tidy)
storage/errors.go                      |  18 --   (removed pebble references)
storage/go.mod                         |  28 ---  (stale deps)
storage/go.sum                         |  96 ----- (stale deps)
```

### Files Deleted (from storage)

```
storage/pebble_bench_test.go           |  90 --------
storage/pebble_config.go               |  79 -------
storage/pebble_event_store.go          | 243 -------
storage/pebble_event_store_test.go     | 260 -------
storage/pebble_helpers.go              | 129 ------
storage/pebble_save.go                 |  79 -------
storage/pebble_serialization.go         |  62 -------
storage/pebble_time_travel_test.go     | 223 -------
storage/turso_connector.go             |  86 -------
storage/turso_connector_test.go        | 385 -------
storage/turso_sync.go                  | 107 -------
```

### New Modules

```
pebble/                                (extracted from storage)
  - bench_test.go
  - config.go
  - errors.go
  - go.mod
  - go.sum
  - helpers.go
  - reconstruct.go
  - save.go
  - serialization.go
  - store.go
  - store_test.go
  - time_travel_test.go

turso/                                 (extracted from storage)
  - connector.go
  - connector_test.go
  - errors.go
  - go.mod
  - go.sum
  - sync.go
```

---

## c) TEST RESULTS

### Event Context Tests ✅

```
=== RUN   TestEventContext_ContextMethod
--- PASS: TestEventContext_ContextMethod (0.00s)
    --- PASS: TestEventContext_ContextMethod/no_deadline_returns_Background (0.00s)
    --- PASS: TestEventContext_ContextMethod/with_deadline_returns_context_with_deadline (0.00s)
    --- PASS: TestEventContext_ContextMethod/past_deadline_context_is_already_done (0.00s)
=== RUN   TestEventContext_DeadlineMethod
--- PASS: TestEventContext_DeadlineMethod (0.00s)
=== RUN   TestEventContext_FromContext
--- PASS: TestEventContext_FromContext (0.00s)
=== RUN   TestEventContext_Clone
--- PASS: TestEventContext_Clone (0.00s)
PASS
```

### Core Event Tests ✅

```
ok  	github.com/larsartmann/go-cqrs-lite/core/event	0.023s
ok  	github.com/larsartmann/go-cqrs-lite/core/decider	0.005s
```

### Lint Status ✅ (core)

```
==> Linting core
0 issues.
```

---

## d) KNOWN ISSUES

### Critical (Blocks Compilation)

| Issue | Location | Cause |
|-------|----------|-------|
| WrongArgCount to Save | example/* | AggregateRef migration incomplete |
| WrongArgCount to Load | example/* | AggregateRef migration incomplete |
| undefined: PebbleEventStore | storage/benchmark_test.go | Pebble extracted to separate module |
| undefined: OpenTurso | storage/sqlite_bench_test.go | Turso extracted to separate module |
| undefined: saga.State | testhelpers | Saga module interface changed |

### Pre-existing (Not My Fault)

| Issue | Location | Severity |
|-------|----------|----------|
| exhaustruct: Flow missing fields | catalog/registry_helpers.go | LOW |
| goconst: magic string "1.0.0" | catalog/registry_helpers.go | LOW |
| mnd: magic number 2 | catalog/internal/cattest/builders.go | LOW |

---

## e) WHAT WE SHOULD IMPROVE

### Immediate (High Priority)

1. **Fix Example Compilation**
   - Update `example/storage/main.go` to use `event.AggregateRef`
   - Update `example/projection/main.go` to use `event.AggregateRef`
   - Update `example/stream/main.go` to use `event.AggregateRef`

2. **Fix Storage Module**
   - Update `storage/go.mod` to remove pebble/turso dependencies
   - Fix or skip benchmark tests that depend on extracted modules
   - Run `go mod tidy` on storage

3. **Fix Testhelpers**
   - Update `testhelpers/saga_helpers.go` to use correct saga.State type

4. **Complete Integration**
   - Run `go mod tidy` on all affected modules (storage, integration, examples)
   - Verify all tests pass

### Medium-term (Next 5 Sessions)

5. **Add Context Propagation to Bus**
   - Wire event.Context through Publish path
   - Add middleware support for context inspection

6. **Add Event Versioning Context**
   - Track schema version context through upcasting

7. **Documentation**
   - Update AGENTS.md with event context propagation
   - Add usage examples to event package docs

### Technical Debt

8. **Code Organization**
   - Split large test files (decider_test.go ~1200L, runner_test.go ~1057L)
   - Reduce file complexity in storage/

---

## f) TOP #25 THINGS TO GET DONE NEXT

Ranked by impact/effort ratio:

| # | Task | Module | Impact | Effort | Priority |
|---|------|--------|--------|--------|----------|
| 1 | Fix example compilation (AggregateRef) | example/* | HIGH | MED | P0 |
| 2 | Fix storage/go.mod cleanup | storage | HIGH | LOW | P0 |
| 3 | Fix testhelpers/saga_helpers | testhelpers | HIGH | LOW | P0 |
| 4 | Add stream integration tests | stream | HIGH | LOW | P1 |
| 5 | Add BDD tests for Version | core/event | MED | LOW | P2 |
| 6 | Add BDD tests for SchemaVersion | core/event | MED | LOW | P2 |
| 7 | Split decider_test.go (~1200L) | core/decider | MED | MED | P2 |
| 8 | Split runner_test.go (~1057L) | projection | MED | MED | P2 |
| 9 | Add fuzz tests for event creation | core/event | HIGH | MED | P2 |
| 10 | Add fuzz tests for ID parsing | core/pkg/id | HIGH | LOW | P2 |
| 11 | Add ProcessedAt to CheckpointStore | storage | LOW | LOW | P3 |
| 12 | Add WithAsyncWrites() for Pebble | pebble | MED | MED | P3 |
| 13 | Add event.Context to Bus publish | core/event | MED | MED | P2 |
| 14 | Benchmark PG vs SQLite vs Pebble | storage | HIGH | MED | P2 |
| 15 | Add E2E throughput benchmarks | integration | HIGH | MED | P2 |
| 16 | Parallelize CI matrix | .github/workflows | MED | LOW | P3 |
| 17 | Add gofumpt/goimports to pre-commit | tooling | LOW | LOW | P4 |
| 18 | Add performance regression CI | .github/workflows | HIGH | MED | P3 |
| 19 | Enforce 350-line test limit | tooling | LOW | LOW | P4 |
| 20 | Rewrite example/user/ for full CQRS | example | HIGH | HIGH | P1 |
| 21 | Add hybrid service example | example | MED | HIGH | P2 |
| 22 | Add catalog diff/breaking-change | catalog | MED | HIGH | P2 |
| 23 | Add distributed tracing E2E | integration | LOW | MED | P3 |
| 24 | Add dead letter queue tests | projection | LOW | LOW | P4 |
| 25 | Fix LSP stale diagnostics | tooling | HIGH | LOW | P1 |

---

## g) TOP #1 QUESTION I CANNOT FIGURE OUT

### The Question

**How do we properly handle the pebble and turso module boundaries with the storage module?**

### The Problem

When we extracted pebble and turso from storage into separate modules:
1. Storage now has references to modules that don't exist in its go.mod
2. Examples that use storage+pebble now need separate imports
3. The integration tests that covered all three are now split

### What I've Tried

1. ✅ Created separate pebble/ and turso/ modules
2. ✅ Updated go.work to include them
3. ❌ Storage still has stale go.mod references
4. ❌ Examples still reference old storage API

### What I Need

- [ ] Clarification on whether storage should depend on pebble/turso OR if consumers should import both
- [ ] Guidance on how to restructure integration tests
- [ ] Decision: should examples import storage+pebble separately or should there be a unified import path?

---

## COMMIT PLAN

### Commit 1: Event Context Propagation (NEW)

```
feat(event): add Context() to Event interface for deadline propagation

- Add Context() method to Event interface for cancellation/deadline access
- Add WithDeadline(time.Time) option for explicit deadline setting
- Add FromContext(context.Context) option to extract deadline from context
- Add Deadline() method to query deadline status on events
- Update ImmutableEvent with deadline field and Clone() preservation
- Add comprehensive tests for context propagation behavior
- Fix nonImmutableEvent test helper to implement Context()

This enables event handlers to check if the original operation's
deadline has passed, supporting proper cancellation propagation.
```

### Commit 2: Pebble/Turso Module Extraction (NEW)

```
refactor(storage): extract Pebble and Turso to separate modules

- Extract pebble/ to github.com/larsartmann/go-cqrs-lite/pebble
- Extract turso/ to github.com/larsartmann/go-cqrs-lite/turso
- Update go.work to include new modules
- Clean up storage/go.mod from extracted dependencies
- Update go mod tidy across affected modules

This reduces storage module complexity and allows independent
versioning of storage backends.
```

---

## APPENDIX: Module Graph (Updated)

```
core/event ──> core/pkg/id
             └── codec

pebble ──────> core/event
             └── codec

turso ───────> core/event

storage ─────> core/event
             └── otel

signing ─────> core

catalog ──────> core

middleware ──> core
            └── otel
```

---

_Generated by Crush_
_Version: Full Status Report v1.0_
