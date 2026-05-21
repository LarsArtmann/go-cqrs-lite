# go-cqrs-lite — Comprehensive Status Report

**Date:** 2026-05-21 02:27
**Sessions Covered:** 1–86 (May 1 – May 21, 2026)
**Commits Since May 1:** 529 (918 total)
**Report Type:** FULL COMPREHENSIVE — a) Done, b) Partially Done, c) Not Started, d) Totally Fucked Up, e) Improvements, f) Top 25 Next, g) #1 Question

---

## Project Vital Signs

| Metric | Value | Status |
|--------|-------|--------|
| Total LOC | 47,144 (15,682 production + 31,462 test) | ✅ Healthy |
| Production Files | 178 | ✅ |
| Test Files | 127 | ✅ |
| Benchmark Functions | 59 across 13 files | ✅ |
| Go Modules | 12 (root, core, memory, catalog, middleware, testhelpers, projection, storage, sync, integration, example/todo, example/user) | ✅ |
| Sentinel Errors | ~35 classified errors across 7 modules | ✅ |
| Compile-time Interface Checks | 29 `var _` assertions | ✅ |
| All Tests | 32/32 packages PASS (including -race) | ✅ |
| Lint | 5 issues (all in core/pkg/dispatcher — golines, noinlineerr, perfsprint) | ⚠️ Minor |
| Race Detector | 0 races | ✅ |
| File Size Compliance | All production files ≤263 lines (max: testhelpers/fake_store.go at 263, over 250 limit) | ⚠️ 1 violation |

---

## Test Coverage by Module

| Package | Coverage | Trend |
|---------|----------|-------|
| `core/query` | 100.0% | ✅ Stable |
| `core/pkg/dispatcher` | 100.0% | ✅ Stable |
| `middleware` | 100.0% | ✅ Stable |
| `memory` | 99.6% | ✅ Stable |
| `core/pkg/id` | 97.8% | ✅ Stable |
| `catalog/adapters` | 97.1% | ✅ Stable |
| `catalog/asyncapi` | 97.1% | ✅ Stable |
| `catalog/d2` | 97.6% | ✅ Stable |
| `catalog/openapi` | 98.1% | ↑ Was 97.9% |
| `core/aggregate` | 95.9% | ↑ Was 95.5% |
| `catalog/eventcatalog` | 95.8% | ✅ Stable |
| `core/command` | 94.7% | ✅ Stable |
| `projection` | 93.9% | ↑ Was 93.6% |
| `sync` | 92.2% | ✅ Stable |
| `catalog` | 91.2% | ✅ Stable |
| `catalog/docserver` | 91.0% | ✅ Stable |
| `core/decider` | 93.3% | ↓ Was 95.0% (errorfamily migration added paths) |
| `core/event` | 89.1% | ↓ Was 94.4% (errorfamily migration added paths) |
| `storage` | 88.3% | ↓ Was 88.1% |
| `example/todo/domain` | 100.0% | ✅ |
| `example/todo/queries` | 81.8% | ✅ |
| `example/todo/projections` | 78.9% | ✅ |
| `example/todo/commands` | 68.4% | ✅ |
| `example/todo/aggregate` | 88.6% | ✅ |
| `example/todo/cmd/api` | 41.9% | ⚠️ Low |
| `example/todo/storage` | 29.2% | ⚠️ Low |
| `example/user` | 42.7% | ⚠️ Low |
| `testhelpers` | 10.5% | ⚠️ Low (by design — helpers tested by consumers) |

**Weighted average (library modules only, excluding examples/testhelpers):** ~93.5%

---

## a) FULLY DONE ✅

### Architecture & Core Design

- [x] **Multi-module monorepo** — 12 Go modules with clean acyclic dependency DAG. `core` is independently publishable.
- [x] **Branded IDs** — `id.Of[T]` as type alias to `go-branded-id`. ULID-backed, zero delegation boilerplate.
- [x] **Decider pattern** — `decider.Decider[State]` with pure functions. Recommended over OO `aggregate` for new consumers.
- [x] **Error taxonomy** — 5 families (Rejection, Conflict, Transient, Corruption, Infrastructure) with `event.Error` struct, `Classify()`, `IsRetryable()`, `RegisterClassification()` for extensibility.
- [x] **ISP on Bus** — `event.Publisher` + `event.Subscriber` sub-interfaces. Repositories accept `Publisher`, projections accept `Subscriber`.
- [x] **SnapshotStrategy** — Canonical interface in `core/event`, shared by aggregate + decider. `EveryNEvents` validates `n > 0`.
- [x] **Event versioning** — `event.Version` branded type with `Add/Sub/Cmp/IsPositive/Mod/Increment/Decrement`. `event.SchemaVersion` distinct from `Version`.
- [x] **Time-travel queries** — `LoadToVersion`, `LoadToTimestamp`, `LoadAllFromPosition` on all store implementations (Memory, SQL, Pebble, Fake).
- [x] **Decider time-travel** — `decider.Repository.LoadAtVersion()` and `LoadAtTime()` for temporal state reconstruction.
- [x] **Position-based replay** — `projection.Runner` auto-detects `event.PositionalLoader` for efficient catch-up.
- [x] **TransactionalStore** — `event.TransactionalStore` interface + `storage.SQLTransactionalStore` for atomic save+outbox.

### Error Handling

- [x] **All sentinels classified** — 35+ sentinels across 7 modules registered via `RegisterClassification()`.
- [x] **No-panic convention** — All `New*` functions return `(*T, error)`. `MustNew*` helpers for panic behavior.
- [x] **Error family migration** — Bare `errors.New` sentinels converted to `errorfamily` structured errors with `Wrap*` helpers.
- [x] **Middleware sentinels** — `ErrValidationFailed`, `ErrRetryExhausted`, `ErrRetryCanceled`, `ErrPanicRecovered` all wrap-able.
- [x] **WrapFrom helper** — `event.WrapFrom()` preserves error family classification.

### Testing & Quality

- [x] **32/32 test packages PASS** — including race detector.
- [x] **59 benchmarks** across 13 files.
- [x] **29 compile-time interface checks** — `var _ Interface = (*Impl)(nil)`.
- [x] **Godoc on all critical exports** — 57+ symbols documented across 8 files.
- [x] **Package doc.go** — Added to memory, catalog, testhelpers, storage modules.
- [x] **No dead code** — Zero TODO/FIXME in production code.
- [x] **Panic recovery** — `HandleParallel` and `OutboxPublisher.run()` have panic recovery.

### Dependencies

- [x] **CockroachDB/errors removed** — Replaced with `fmt.Errorf` + `%w`. 6 transitive deps eliminated.
- [x] **go-json-experiment/json removed** — Replaced with `encoding/json`.
- [x] **No banned libraries** — All deps pass `depguard` checks.
- [x] **Storage isolates memory dep** — `PebbleBackendMemory` returns `ErrPebbleProviderRequired` instead of importing memory.

### Documentation Generation

- [x] **AsyncAPI 3.0 exporter** — YAML/JSON from catalog registry.
- [x] **EventCatalog MDX generator** — Service/schemas structure.
- [x] **D2 diagram exporter** — Color-coded nodes (command=blue, event=red, query=purple).
- [x] **OpenAPI exporter** — REST API schema generation.
- [x] **DocServer** — HTTP server combining all formats.

### Examples

- [x] **example/user** — Full CQRS stack using Decider pattern + middleware + EventCatalog.
- [x] **example/todo** — Complete todo app with Pebble + SQLite storage.

### Cleanup & Refactoring (Recent Sessions)

- [x] **Golden test fixtures refreshed** — AsyncAPI, EventCatalog, package.json all current.
- [x] **example/todo migrated** — testify → stdlib testing. Stale go.mod dep removed.
- [x] **SQLite schema godoc** — `PostgresInitSchema`, `SQLiteEnableWAL`, `OpenSQLite` documented.
- [x] **sync unusedwrite fixed** — `TestLWWResolver_ImplementsInterface` lint resolved.
- [x] **File splits** — All production files ≤263 lines (1 over 250 limit — see below).
- [x] **publishChanges/saveSnapshot** — Shared helpers in `core/event/`, eliminating aggregate↔decider duplication.
- [x] **Classify() unified** — Single registered map, eliminated 30-line hardcoded switch.

---

## b) PARTIALLY DONE ⚠️

### Lint Cleanup (5 remaining issues)

- [x] 46→5 issues across 8 modules (Sessions 20, 44, 48)
- [ ] **5 remaining in `core/pkg/dispatcher`**: 1 golines (long line in runner.go:148), 2 noinlineerr, 2 perfsprint
- **Status:** 97% complete. Easy to finish.

### Coverage Gaps (core/event dropped)

- [x] Error taxonomy and classification fully tested
- [ ] **core/event coverage dropped from 94.4% → 89.1%** after errorfamily migration added new code paths
- **Status:** Need to add tests for new `Wrap*` helpers and `WrapFrom` paths.

### CatalogMeta Consolidation

- [x] Identified as split brain (3 nearly-identical types across command/query/event)
- [ ] **Not consolidated** — `event.CatalogMeta` has extra `AggregateType` field preventing clean merge
- **Status:** Blocked on design decision. `event.CatalogMeta` has an extra field; the other two are identical.

### Example/todo Pattern Quality

- [x] Migrated from testify to stdlib testing
- [x] Removed stale go.mod dep
- [ ] **Still teaches wrong patterns** — uses raw store access, own `loggingMiddleware` instead of library's `middleware.CommandLogging`, no error classification
- **Status:** 40% done. Need to wire library middleware and fix patterns.

### MetricsRecorder → OTel Bridge

- [x] `MetricsRecorder` interface exists in middleware
- [ ] **No OTel adapter** — inconsistent with tracing (which has OTel bridge)
- **Status:** Interface designed, bridge not built.

---

## c) NOT STARTED 🆕

1. **`govulncheck` in CI** — No vulnerability scanning in GitHub Actions. Security gap.
2. **Release automation** — No goreleaser, no tag-based versioning, no changelog generation.
3. **Saga/Process Manager** — Design doc exists (`docs/planning/SAGA_DESIGN.md`) with 18h estimate. Zero implementation.
4. **Typed query handler migration** — Design doc exists (`docs/planning/QUERY_HANDLER_GENERICS.md`). `TypedHandler[T]` added but old `Handler = func(...)(any, error)` still exists.
5. **`io.Closer` removal from interfaces** — Evaluated in Session 55-56, deferred as breaking change.
6. **OTel metrics adapter** — For `MetricsRecorder` interface.
7. **Client-side store** — Design concept for offline-first, not started.
8. **Event signing** — Integrity verification, not started.
9. **WASM/mobile SDK** — Explicitly non-goal per Session 30.
10. **Offline-first sync protocol** — `sync/` module exists but no actual protocol implementation.

---

## d) TOTALLY FUCKED UP 💥

### 1. `sync/` Module — FULL GHOST

**Severity:** HIGH (dead weight, misrepresents library capabilities)

- Zero external consumers. No package outside `sync/` imports it.
- Well-tested (92.2% coverage), well-documented, completely unused.
- Contains: `NodeID`, `OperationID`, `VectorClock`, `ConflictResolver`, `LWWResolver`, `Operation`, `SyncMessageType`.
- **Why it's fucked:** It exists in a library/SDK where "public API surface IS the product." A ghost module with zero consumers is either premature abstraction or misdirected effort. It signals "we have distributed sync" when we don't.
- **Options:** (a) Delete it, (b) Integrate into core as `event.VectorClock` metadata, (c) Keep with explicit `EXPERIMENTAL` doc, (d) Find a real consumer.

### 2. `catalog/docserver/` — GHOST PACKAGE

**Severity:** MEDIUM (dead code, maintenance burden)

- Zero imports from outside the package. No external consumer.
- 215-line HTTP server combining all catalog export formats.
- Decent coverage (91.0%) but nobody uses it.
- **Why it's fucked:** 215 lines of production code + tests for something with zero consumers in a library that values "every module must be trustworthy on its own."

### 3. `example/todo/handler/` — EMPTY DIRECTORY

**Severity:** TRIVIAL (but sloppy)

- Exists as an empty directory. Leftover from a refactoring.
- **Why it's fucked:** It's a minor embarrassment. Empty directories shouldn't exist in a clean repo.

### 4. `goexperiment` Flags — CARGO CULT

**Severity:** MEDIUM (misleading, potential for breakage)

- `goexperiment.arenas`, `goexperiment.simd`, `goexperiment.goroutineleakprofile`, `goexperiment.runtimesecret` in `flake.nix`.
- **Zero usage** of arenas or SIMD in the entire codebase.
- `goroutineleakprofile` and `runtimesecret` are reasonable for dev/profiling but `arenas` and `simd` are cargo cult.
- **Why it's fucked:** Enables unstable runtime features that no code uses. Adds noise. Could cause subtle bugs with future Go versions.

### 5. `testhelpers/fake_store.go` at 263 Lines — FILE SIZE VIOLATION

**Severity:** LOW (project convention violation)

- Project limit: 250 lines per file. This is 263.
- Contains `FakeStore` with all time-travel methods.
- **Why it's fucked:** The project has a hard rule. This is the only violation.

---

## e) WHAT WE SHOULD IMPROVE 📈

### Architecture Improvements

1. **Consolidate CatalogMeta** — 3 near-identical types across command/query/event. Extract shared base or use composition.
2. **Dispatcher deduplication** — `command.Dispatcher` and `query.Dispatcher` share significant structure. Generic `dispatcher.Dispatcher[H, M]` exists but the public APIs don't leverage it fully.
3. **core/event god-package** — 23 files, ~90 exports, 8+ logical clusters. Go convention favors flat packages, but this is at the upper bound. Consider splitting `event.Store`/`event.Bus` interfaces into `event/store.go` sub-package? No — flat package is idiomatic Go.
4. **InMemoryRunner vs projection.Runner overlap** — `core/event/runner.go` (InMemoryRunner) and `projection/runner.go` (Runner) have overlapping responsibilities. InMemoryRunner is a simple bus+store wrapper; projection.Runner adds replay+checkpoint. Should clarify relationship.

### Error Handling Improvements

5. **Wrap all bare `errors.New` with errorfamily** — Some sentinels in storage, aggregate, projection still use `errors.New` instead of structured error family constructors. The `WrapFrom` helper now makes this possible.
6. **Cross-module sentinel classification** — Currently `Classify()` has registered sentinels from aggregate/projection/storage, but some edge cases fall through to `Transient` default.

### Testing Improvements

7. **core/event coverage gap** — Dropped from 94.4% to 89.1% after errorfamily migration. New `Wrap*` paths need test coverage.
8. **example/todo coverage** — `cmd/api` at 41.9%, `storage` at 29.2%. Not library code, but still embarassing for an "example."
9. **storage coverage** — At 88.3%, lowest among library modules. Time-travel SQL paths need more tests.

### Operational Improvements

10. **govulncheck in CI** — No dependency vulnerability scanning. Industry standard.
11. **Release automation** — Manual releases are error-prone. goreleaser or similar.
12. **Benchmark regression tracking** — 59 benchmarks exist but no CI tracking of regressions.

### Documentation Improvements

13. **doc.go for remaining modules** — `core`, `projection`, `sync` still lack `doc.go` package documentation.
14. **ADR quality** — Some ADRs are thin. Could benefit from context/alternatives/consequences sections.
15. **CHANGELOG.md** — Exists but `Unreleased` section has merged duplicates. Needs cleanup.

---

## f) TOP 25 THINGS TO DO NEXT (Ranked by Impact × Feasibility)

| # | Task | Impact | Effort | Category |
|---|------|--------|--------|----------|
| 1 | **Fix 5 remaining lint issues** in `core/pkg/dispatcher` | HIGH | LOW (30min) | Quality |
| 2 | **Decide fate of `sync/` module** — delete, integrate, or mark EXPERIMENTAL | HIGH | MED (2h) | Architecture |
| 3 | **Add `core/event` tests for Wrap*/WrapFrom paths** — restore 94%+ coverage | HIGH | MED (2h) | Testing |
| 4 | **Remove cargo cult `goexperiment.arenas` and `goexperiment.simd`** from flake.nix | MED | LOW (5min) | Cleanup |
| 5 | **Delete empty `example/todo/handler/` directory** | LOW | LOW (1min) | Cleanup |
| 6 | **Wire library middleware into `example/todo`** — replace custom loggingMiddleware | MED | MED (3h) | Examples |
| 7 | **Add `govulncheck` to CI pipeline** | HIGH | MED (2h) | Security |
| 8 | **Consolidate 3x `CatalogMeta`** — extract shared base type | MED | MED (3h) | Dedup |
| 9 | **Split `testhelpers/fake_store.go`** (263→<250 lines) | LOW | LOW (15min) | Quality |
| 10 | **Decide fate of `catalog/docserver/`** — delete or find consumer | MED | MED (1h) | Architecture |
| 11 | **Add `doc.go` to core, projection, sync** | MED | LOW (30min) | Documentation |
| 12 | **Build OTel metrics adapter** for `MetricsRecorder` | MED | MED (4h) | Observability |
| 13 | **Add storage time-travel SQL test coverage** | MED | MED (2h) | Testing |
| 14 | **Add benchmark regression tracking in CI** | MED | MED (3h) | CI/CD |
| 15 | **Set up release automation** (goreleaser + tag-based versioning) | MED | HIGH (6h) | DevOps |
| 16 | **Clean up CHANGELOG.md** — merge duplicate sections, update Unreleased | LOW | LOW (30min) | Documentation |
| 17 | **Migrate query.Handler to typed** — complete TypedHandler migration plan | MED | HIGH (8h) | API |
| 18 | **Implement Saga/Process Manager** — design doc exists, 18h estimate | HIGH | HIGH (18h) | Feature |
| 19 | **Wrap remaining bare `errors.New` sentinels** with errorfamily constructors | LOW | MED (3h) | Error Handling |
| 20 | **Add example/todo coverage** for cmd/api and storage | LOW | MED (2h) | Testing |
| 21 | **Remove `io.Closer` from store interfaces** — breaking but correct | MED | HIGH (4h) | API |
| 22 | **Add `RegisterClassification` for remaining edge-case sentinels** | LOW | LOW (1h) | Error Handling |
| 23 | **Build client-side event store** for offline-first | HIGH | HIGH (40h) | Feature |
| 24 | **Add event signing/verification** | MED | HIGH (20h) | Security |
| 25 | **Clarify InMemoryRunner vs projection.Runner relationship** in docs | LOW | LOW (1h) | Documentation |

---

## g) MY #1 QUESTION I CANNOT FIGURE OUT MYSELF 🤔

**What is the intended future of the `sync/` module?**

It has zero consumers. It's well-built. It sits in a library that says "every module must be trustworthy on its own" and "public API surface IS the product." But nobody imports it.

Three possible futures:
1. **Delete it** — It's premature. If someone needs vector clocks, they'll tell us. YAGNI.
2. **Integrate it** — Make `event.VectorClock` a metadata field on events. Make sync part of the core event sourcing story. This would be the "offline-first primitives" path from the architecture roadmap.
3. **Mark it `// Deprecated: EXPERIMENTAL`** — Keep it but be honest about its status.

The architecture roadmap (Session 30) says "Offline-First Primitives" is a future initiative. The execution plan (Session 31) lists it. But it's the *library's* roadmap — I don't know if **you** actually want to build offline-first sync into go-cqrs-lite, or if `sync/` was an experiment that didn't pan out.

**This decision changes the top-10 priority list significantly.** If sync stays and gets integrated, it's a major feature. If it goes, we save maintenance burden and can focus on catalog/storage/middleware.

---

## Session-by-Session Summary (Recent Sessions 80–86)

| Session | Focus | Key Outcome |
|---------|-------|-------------|
| 80 | Time-travel tests + Decider API + File splits | 30 new tests, `LoadAtVersion`/`LoadAtTime` on decider |
| 81 | Position-based replay + SQL optimization | PositionalLoader auto-detect, composite index, flaky test fix |
| 83 | File splits + Type API + Deprecated API removal | `Version.Add/Sub/Cmp`, removed `VectorClock.Compare()`, `OutboxID.String()` |
| 84 | Error family migration | Bare sentinels → structured errorfamily, `Wrap*` helpers |
| 85 | Brutal self-review + Execution plan | Ghost systems audit, 46-task plan |
| 86 | Storage cleanup (Turso/SQLite) + Golden fixtures | Dedup Turso tests, SQLite helpers, refreshed fixtures |

---

## Module Dependency DAG (Current State)

```
core (leaf — zero internal deps)
  ↑
  ├── memory (test impls)
  ├── testhelpers (shared test utils)
  ├── middleware (cross-cutting)
  ├── catalog (documentation gen)
  ├── projection (replay + subscription)
  ├── storage (PostgreSQL/SQLite/Pebble)
  ├── sync (ghost — zero consumers)
  ├── integration (cross-module tests)
  ├── example/todo (demo app)
  └── example/user (demo app)
```

**DAG is clean.** No cycles. `core` is independently publishable. `storage` no longer depends on `memory`.

---

## Key Metrics Over Time

| Metric | Session 48 | Session 73 | Session 86 (Now) |
|--------|-----------|-----------|-------------------|
| Test Packages | 22 | 22 | 32 |
| Lint Issues | 0 → 0 | 0 | 5 (core/dispatcher) |
| Total LOC | ~28,000 | ~36,000 | 47,144 |
| Production LOC | ~9,500 | ~12,000 | 15,682 |
| Benchmarks | ~30 | 43 | 59 |
| Sentinel Errors | ~25 | 35 | 35+ |
| Interface Checks | ~15 | 25 | 29 |
| Go Modules | 9 | 10 | 12 |
| Race Detector | PASS | PASS | PASS |

---

*Generated by Crush — Session 86 continuation, 2026-05-21 02:27*
