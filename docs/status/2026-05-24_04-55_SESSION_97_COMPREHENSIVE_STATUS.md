# Session 97 — Comprehensive Status Report

**Date:** 2026-05-24 04:55 CEST
**Branch:** master @ `c4462a8`
**Sessions:** 97 (spanning ~3 months of development)

---

## Executive Summary

The go-cqrs-lite library is in **excellent shape**: 27/27 packages pass, zero lint, zero vet, zero TODOs. All Session 97 quality items have been completed. The library is publishable as-is for core modules. The main remaining work is: (1) the `example/todo` app is broken due to a downstream API break in `cqrs-htmx`, (2) 2 files slightly exceed the 250-line limit, (3) several low-coverage modules could be pushed higher, and (4) the `sync` module is new and unproven in production.

---

## A) Fully Done ✅

### Core Library (Production-Ready)

| Module | Coverage | Status | Notes |
|--------|----------|--------|-------|
| `core/command` | 94.7% | ✅ Complete | Dispatcher, handler, middleware, catalog |
| `core/query` | 100.0% | ✅ Complete | Dispatcher, typed handlers, pagination |
| `core/event` | 93.8% | ✅ Complete | Store/Bus/Snapshot interfaces, OutboxPublisher, Projection, Clock |
| `core/aggregate` | 95.9% | ✅ Complete (deprecated) | Full functionality, `// Deprecated: use decider` |
| `core/decider` | 93.6% | ✅ Complete | Pure-function aggregate pattern, recommended |
| `core/pkg/dispatcher` | 100.0% | ✅ Complete | Generic Dispatcher[H, M], LifecycleMixin |
| `core/pkg/id` | 98.1% | ✅ Complete | Branded IDs, ULID, Parse/MustParse |

### Infrastructure Modules

| Module | Coverage | Status | Notes |
|--------|----------|--------|-------|
| `memory` | 99.6% | ✅ Complete | MemoryStore, MemoryBus, MemorySnapshotStore |
| `middleware` | 100.0% | ✅ Complete | Logging, Retry, Recovery, Validation, Metrics |
| `testhelpers` | 94.4% | ✅ Complete | FakeStore, FakeBus, FakeOutbox, handlers, assertions |
| `projection` | 94.4% | ✅ Complete | Runner, HandlerRegistry, Builder with On[T] |

### Documentation & Catalog

| Module | Coverage | Status | Notes |
|--------|----------|--------|-------|
| `catalog` | 96.8% | ✅ Complete | Registry, SchemaFromType, typed IDs |
| `catalog/adapters` | 100.0% | ✅ Complete | CatalogBuilder, FromDispatcher adapters |
| `catalog/asyncapi` | 93.7% | ✅ Complete | AsyncAPI 3.0 YAML/JSON export |
| `catalog/d2` | 95.0% | ✅ Complete | D2 diagram text export |
| `catalog/eventcatalog` | 91.3% | ✅ Complete | EventCatalog MDX generator |
| `catalog/docserver` | 90.1% | ✅ Complete | HTTP doc server |
| `catalog/openapi` | 94.4% | ✅ Complete | OpenAPI 3.0 export |
| `catalog/internal/caseutil` | 100.0% | ✅ Complete | Case conversion utilities |

### Storage

| Module | Coverage | Status | Notes |
|--------|----------|--------|-------|
| `storage` | 89.3% | ✅ Complete | SQL event store (Postgres + SQLite), Pebble store |

### Sync (New Module)

| Module | Coverage | Status | Notes |
|--------|----------|--------|-------|
| `sync` | 97.6% | ✅ Complete | VectorClock, ConflictResolver, LWW, Operation types |

### Session 97 Deliverables (All Complete)

| Commit | Summary | Status |
|--------|---------|--------|
| `7d38273` | FakeStore defensive copies — Load/LoadFromVersion return copies, not references | ✅ |
| `c74129a` | NewOperation validation + MustNewOperation — breaking but safe (no external consumers) | ✅ |
| `0b85fbc` | VectorClock.String() + MergeResult.String() — deterministic, sorted, tested | ✅ |
| `13e0504` | Doc comments — OutboxPublisher, Projection, Builder, On[T], Build | ✅ |
| `65a1116` | testhelpers coverage 79.7% → 94.4% | ✅ |
| `617135a` | Error wrapping — storage Save/AppendBatch, projection handleAndCheckpoint | ✅ |
| `fad7766` | .gitignore SQLite WAL/SHM patterns | ✅ |
| `c4462a8` | Zero lint — extract `"unknown"` to `clockOrderUnknown` constant | ✅ |

---

## B) Partially Done 🔶

### example/todo — BROKEN BUILD

- **State:** `nix run .#build` fails because `example/todo` depends on `github.com/larsartmann/cqrs-htmx` which references `event.RegisterClassification` — a function that no longer exists in the core module (removed during API surface reduction in Session 89).
- **Impact:** Build failure in `nix run .#build`. All other modules build fine individually.
- **Fix needed:** Either update `cqrs-htmx` to use the current API, or remove the `cqrs-htmx` dependency from `example/todo`.
- **Root cause:** Session 89 removed ~60 exports from the API surface; downstream consumer wasn't updated.

### File Size Limit — 2 Files Exceed 250 Lines

| File | Lines | Over by |
|------|-------|---------|
| `core/event/event.go` | 273 | +23 |
| `catalog/eventcatalog/exporter.go` | 251 | +1 |

- `event.go` has the core event struct, metadata, and constructors. Splitting is possible but tricky (circular deps risk).
- `exporter.go` barely exceeds — minor refactor could fix it.

### Aggregate Package — Deprecated But Not Removed

- `core/aggregate` has `// Deprecated: use decider` on the package.
- 5 integration test files still use it (by design — testing backward compatibility).
- Dead exports remain: some methods on `Root` interface that nobody calls externally.

---

## C) Not Started ⬜

### Coverage Improvements (Ordered by Impact)

| Module | Current | Target | Gap | Effort |
|--------|---------|--------|-----|--------|
| `catalog/internal/schemautil` | 84.2% | 90%+ | 5.8pp | Low — untested edge cases in reflect |
| `storage` | 89.3% | 93%+ | 3.7pp | Medium — Pebble error paths, edge cases |
| `catalog/eventcatalog` | 91.3% | 94%+ | 2.7pp | Low — uncovered branches in writer |
| `catalog/docserver` | 90.1% | 93%+ | 2.9pp | Low — uncovered HTTP handler branches |
| `core/event` | 93.8% | 96%+ | 2.2pp | Medium — batch API edge cases |
| `core/decider` | 93.6% | 96%+ | 2.4pp | Medium — error wrapping paths |

### Architectural Improvements

1. **`sync` module rename** — Currently shadows Go's `sync` package. Breaking change, needs owner decision.
2. **`query.Handler` returns `any`** — Violates project "no any" rule. `DispatchTyped[T]` is the workaround. Design doc exists at `docs/planning/QUERY_HANDLER_GENERICS.md`.
3. **`MemoryBus.Publish` holds RLock during handler execution** — Subscribers block publishers. Acceptable for test utility.
4. **`event.Store` ISP split** — Current interface has Store, GlobalLoader, PositionalLoader, TransactionalStore, etc. Could be split into finer interfaces. Architectural change, not appropriate for quality sweep.

### Documentation & DX

5. **GoDoc completeness** — All exported types now have doc comments (done this session). But many exported functions in `catalog/` still lack examples.
6. **README could show decider pattern** — Current README examples use the older aggregate style.
7. **No CHANGELOG.md** — Version bumps happen but aren't tracked formally.

### Testing Infrastructure

8. **No benchmark suite** — Only `integration/aggregate/benchmark_test.go` exists. No benchmarks for event store, decider, or middleware.
9. **No fuzz tests** — Event parsing, ID generation, and schema reflection could benefit from fuzz testing.
10. **No mutation testing** — Would validate that tests actually catch bugs.

---

## D) Totally Fucked Up 💥

### example/todo Build Failure

**Severity:** HIGH — breaks `nix run .#build`

```
github.com/larsartmann/cqrs-htmx/errors.go:34:9: undefined: event.RegisterClassification
```

The `cqrs-htmx` dependency references `event.RegisterClassification` which was removed in Session 89's API surface reduction (~60 exports removed). The `example/todo` module pulls in `cqrs-htmx` as a dependency, and it hasn't been updated.

**Options:**
1. Update `cqrs-htmx` to use `go-error-family` directly (best long-term fix)
2. Remove `cqrs-htmx` dependency from `example/todo` and inline the needed code
3. Re-add `RegisterClassification` as a compatibility shim (worst option — reverses cleanup)

### example/todo Dependency Chaos

The `example/todo/go.mod` has a massive dependency tree:
- `cockroachdb/pebble` (storage engine)
- `turso.tech/database/tursogo` (Turso DB)
- `casbin/casbin/v3` (authorization)
- `prometheus/client_golang` (metrics)
- `larsartmann/cqrs-htmx` (HTMX framework — broken)

This is a 39-line indirect dependency chain for what should be a simple example app. It pulls in far more than the library itself.

---

## E) What We Should Improve

### High Priority (Next Session)

1. **Fix example/todo build** — Either update cqrs-htmx or decouple. This is the only build failure.
2. **Trim example/todo dependencies** — The example should demonstrate go-cqrs-lite, not pull in every infrastructure library.
3. **Split core/event/event.go** — At 273 lines it exceeds the 250-line convention. Extract metadata types or constructors to a separate file.
4. **Split catalog/eventcatalog/exporter.go** — Barely over at 251 lines; trivial fix.

### Medium Priority

5. **Version bump audit** — Several modules have stale version tags (e.g., `storage v0.2.0`, `middleware v1.0.0`). After API surface changes, versions should be bumped.
6. **Schemautil coverage** — At 84.2% this is the lowest non-test package. The reflect-based schema generation deserves better coverage.
7. **Storage coverage** — At 89.3% with Pebble error paths uncovered. Real-world users will hit these.
8. **Remove dead `Command.IdempotencyKey` deprecation** — Deprecated since Session 31 but still present. Either remove or document timeline.
9. **Consolidate nolint directives** — 32 files have nolint comments. Review if any are stale.

### Low Priority / Nice to Have

10. **Add benchmarks** — For event creation, store operations, decider execute, middleware chain.
11. **Add fuzz tests** — For ID parsing, event marshaling, schema reflection.
12. **`sync` module rename** — Discuss with owner whether to rename to `crdt`, `distsync`, or keep as-is.
13. **CHANGELOG.md** — Track version bumps formally.
14. **Update README** — Show decider pattern instead of deprecated aggregate.
15. **Consider `query.Handler` generic redesign** — Document at `docs/planning/QUERY_HANDLER_GENERICS.md`.

---

## F) Top 25 Things to Do Next

**Sorted by impact × effort (highest first):**

| # | Task | Impact | Effort | Type |
|---|------|--------|--------|------|
| 1 | Fix example/todo build (cqrs-htmx API break) | HIGH | S | Fix |
| 2 | Trim example/todo dependency tree | HIGH | M | Cleanup |
| 3 | Split core/event/event.go under 250 lines | MEDIUM | S | Convention |
| 4 | Fix catalog/eventcatalog/exporter.go under 250 lines | LOW | XS | Convention |
| 5 | Bump module versions after API surface changes | MEDIUM | S | Release |
| 6 | Add schemautil coverage (84.2% → 90%+) | MEDIUM | M | Testing |
| 7 | Add storage Pebble error path tests (89.3% → 93%+) | MEDIUM | M | Testing |
| 8 | Add eventcatalog coverage (91.3% → 94%+) | LOW | S | Testing |
| 9 | Add docserver coverage (90.1% → 93%+) | LOW | S | Testing |
| 10 | Review and clean up nolint directives (32 files) | LOW | M | Cleanup |
| 11 | Remove deprecated IdempotencyKey or set removal timeline | LOW | XS | API |
| 12 | Update README to show decider pattern | MEDIUM | S | Docs |
| 13 | Add decider coverage (93.6% → 96%+) | LOW | M | Testing |
| 14 | Add event coverage (93.8% → 96%+) | LOW | M | Testing |
| 15 | Create CHANGELOG.md | LOW | S | Docs |
| 16 | Add benchmark suite for core modules | MEDIUM | M | Testing |
| 17 | Discuss `sync` module rename with owner | LOW | XS | Decision |
| 18 | Evaluate query.Handler generic redesign | MEDIUM | L | Architecture |
| 19 | Add fuzz tests for ID parsing and event marshaling | MEDIUM | M | Testing |
| 20 | Consider event.Store ISP split into finer interfaces | HIGH | L | Architecture |
| 21 | Extract example/user and example/todo to separate repos | MEDIUM | M | Structure |
| 22 | Add Go examples (testable ExampleXxx functions) | LOW | M | Docs |
| 23 | Review catalog dead exports (MakeEvent, AssertEqual, etc.) | LOW | S | Cleanup |
| 24 | Add integration test for OutboxPublisher with real timing | LOW | M | Testing |
| 25 | Evaluate MemoryBus RLock-during-handler fix | LOW | M | Design |

---

## G) My Top #1 Question

**Should `example/todo` stay in this repo, or should it move to its own repository?**

The `example/todo` app is the **only consumer** of the broken `cqrs-htmx` dependency, and it pulls in 39 indirect dependencies (Pebble, Turso, Casbin, Prometheus, etc.) — far more than the library itself. This creates three problems:

1. **Build breakage** — `nix run .#build` fails because of a downstream consumer, not the library.
2. **False coupling** — The library's CI is gated on an example app's external dependencies.
3. **Bloat** — The example has more production dependencies than the library.

Moving `example/todo` (and possibly `example/user`) to their own repos would:
- Keep the library build clean and self-contained
- Allow examples to version-lock independently
- Remove the `cqrs-htmx` dependency from this repo entirely
- Let `nix run .#build` pass again without fixing downstream code

The alternative is to fix `cqrs-htmx` and keep examples here, but that creates an ongoing maintenance burden — every library API change must also update the examples.

---

## Quality Metrics Dashboard

### Coverage Summary (27 packages)

```
100.0% ████████████████████  core/query, core/pkg/dispatcher, middleware, catalog/adapters, catalog/internal/caseutil
 99.6% ████████████████████  memory
 98.1% ████████████████████  core/pkg/id
 97.6% ████████████████████  sync
 96.8% ████████████████████  catalog
 95.9% ████████████████████  core/aggregate
 95.0% ███████████████████░  catalog/d2
 94.7% ███████████████████░  core/command
 94.4% ███████████████████░  testhelpers, projection, catalog/openapi
 94.3% ███████████████████░  core/decider (93.6%), core/event (93.8%)
 93.7% ███████████████████░  catalog/asyncapi
 91.3% ███████████████████░  catalog/eventcatalog
 90.1% ███████████████████░  catalog/docserver
 89.3% ███████████████████░  storage
 84.2% █████████████████░░░  catalog/internal/schemautil
```

### Build & Quality Gates

| Gate | Status |
|------|--------|
| Tests | ✅ 27/27 packages pass |
| Lint | ✅ 0 issues across all 10 modules |
| Vet | ✅ Clean |
| Format | ✅ Clean (gofumpt) |
| Build | ❌ example/todo broken (cqrs-htmx API break) |
| File sizes | ⚠️ 2 files over 250 lines |
| TODOs | ✅ 0 TODO/FIXME/HACK markers |
| Deprecated | 2 items (aggregate package, IdempotencyKey) |

### Codebase Stats

| Metric | Value |
|--------|-------|
| Production Go code | 13,993 lines (16,266 with examples) |
| Test Go code | 31,263 lines |
| Test:Production ratio | 2.23:1 |
| Total functions | 926 |
| Structs | 64 |
| Interfaces | 23 |
| Modules | 12 (core, memory, catalog, middleware, testhelpers, projection, storage, integration, sync, example/user, example/todo, root) |
| nolint directives | 32 files |

### Session History (Sessions 89–97)

| Session | Key Achievement |
|---------|-----------------|
| 89 | API surface reduction: ~60 exports removed, 89.3→92.1% coverage |
| 90 | Projection builder On[T](), IsReplay, event.New, ExecuteWithResult, DeriveAggregateID |
| 92 | Query typed bookend docs, example/todo typed handlers + Pagination |
| 93 | Zero lint across 10 modules, decider dual-wrap fix, registry deterministic Build |
| 94 | gci v2 fix, orphaned go.mod replace, testhelpers 64.6→80.3%, caseutil 76.5→100% |
| 95 | Code deduplication sweep — 19+ test helper extractions |
| 96 | VectorClock nil map bug fix, gopls hints, golden test refresh |
| 97 | FakeStore defensive copies, NewOperation validation, String() methods, doc comments, error wrapping, testhelpers 79.7→94.4%, zero lint |

---

## Module Dependency Graph

```
                    ┌──────────┐
                    │  core    │  ← no internal deps, independently publishable
                    └────┬─────┘
                         │
          ┌──────────────┼──────────────┐
          ▼              ▼              ▼
    ┌──────────┐  ┌───────────┐  ┌──────────┐
    │ memory   │  │testhelpers│  │ catalog  │  ← catalog has NO core dep (standalone)
    └────┬─────┘  └─────┬─────┘  └──────────┘
         │              │
    ┌────┴─────┐  ┌─────┴──────┐
    │projection│  │ middleware  │
    └──────────┘  └────────────┘
         │              │
         ▼              ▼
    ┌────────────────────────┐
    │     integration        │
    └────────────────────────┘
         │
         ▼
    ┌──────────┐
    │ storage  │  ← core only, no testhelpers
    └──────────┘

    ┌──────────┐
    │   sync   │  ← zero internal deps, only go-error-family
    └──────────┘
```

---

_This report covers the state as of commit `c4462a8` on 2026-05-24._
