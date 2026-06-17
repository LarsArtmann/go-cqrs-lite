# Comprehensive Status Report — 2026-06-17 12:56

**Branch:** `master` · **Commits today:** 23 · **HEAD:** `dc305b76`
**Modules:** 30 (23 library + 1 integration + 3 examples + 2 cmd + 1 sub-package)
**Go:** 1.26.3 · **LOC (non-test):** 32,969 · **Test files:** 394 · **ADRs:** 23

---

## A. Fully Done (This Session)

| # | Commit | What | Impact |
|---|--------|------|--------|
| 1 | `9b4ab792` | Dependency bump: go-error-family v0.4.0, go-branded-id v0.3.1, internal modules v2.4.0 | Security/freshness |
| 2 | `9b4ab792` | AGENTS.md: corrected reactive bus docs (command + query now have CommandBus/QueryBus) | Doc accuracy — was actively misleading |
| 3 | `9b4ab792` | storage: extracted `errStoreClosed` package-level sentinel | Perf — eliminates per-call allocation on hot path |
| 4 | `9b4ab792` | turso: extracted `syncEngine` interface for testable SyncDB | Testability — coverage 54.5% → 83.3% |
| 5 | `9b4ab792` | turso: 14 new sync unit tests (Push/Pull/Checkpoint/Stats/HealthCheck/Close error wrapping) | Coverage + correctness |
| 6 | `6f021475` | turso: CheckpointScheduler.Stop() now blocks until goroutine exits (done channel) | **Critical race fix** — was causing intermittent panics |
| 7 | `6f021475` | turso: added `OpenTemp()` — file-backed test DBs for parallel suites | **Eliminated ~20% flaky test rate** (30/30 stable) |
| 8 | `6f021475` | turso: migrated all tests from `:memory:` to `OpenTemp(t.TempDir())` | Stability |
| 9 | `6f021475` | example/todo: added missing `kv/v2` replace directive | Build fix |
| 10 | `05336146` | go.sum sync across 14 modules after v2.4.0 bump | Build integrity |
| 11 | `dc305b76` | TODO_LIST.md updated — reactive bus docs, PostgreSQL CI marked done | Doc accuracy |

**Pre-session work (already merged in `7de8baa4`):**
- Turso Backend facade, sync config, ConfigurePool
- scanTableRe regex fix (advisor was completely broken for modern SQLite)
- CheckpointScheduler initial race fix (stop channel parameter)
- Pebble KV adapter, reactive buses, pprof endpoints, benchmarks
- Codec fuzz fix, KV contract tests, module READMEs

---

## B. Partially Done

| Item | Current State | What's Left |
|------|---------------|-------------|
| **Turso coverage** | 79.1% (was 54.5%) | `OpenSyncWithConfig` only 16.7% — the `tursoclient.NewTursoSyncDb` call needs network. `SyncClient()` 0% — trivial accessor. Target: 85%+ |
| **Pebble coverage** | 82.9% | Error branches in `adapter.go` (Set/Delete/Batch/Commit at 71-75%), `prefixUpperBound` at 60%. Target: 85%+ |
| **Query coverage** | 79.0% | `errors.go` functions (Classify, IsRetryable, NewConflict, NewTransient, etc.) all 0% — these are thin re-exports of error-family. Need simple tests. |
| **Storage coverage** | 82.1% | SQL error paths, snapshot store edge cases |

---

## C. Not Started (From TODO_LIST.md — 16 open items)

### High Impact (customer-facing capabilities)
1. **Schema registry** — JSON Schema validation middleware for events (ADR-0017)
2. **Distributed checkpointing** — multi-instance projection coordination (ADR-0018)
3. **Prometheus metrics exporter** — replace custom `MetricsRecorder` in `middleware/`
4. **Structured logging middleware** — configurable `slog` levels
5. **Distributed tracing propagation** — span context across module boundaries

### Medium Impact (quality/tooling)
6. **Pebble coverage 85%+** — error branches in helpers.go, serialization.go
7. **Pebble golden test** — deterministic CBOR envelope bytes
8. **MemorySnapshotStore golden test** — baseline for pebble comparison
9. **cqrs-gen v2** — struct tag scanning improvements

### Experimental / Long-term
10. **gRPC transport adapter**
11. **NATS/Redis Stream adapter**
12. **Streaming event reads** — `StreamLoader` without materializing
13. **jsonv2 codec experiment**
14. **Arena allocation experiment**
15. **WASM compilation target** — decider module for browser/edge
16. **Documentation site** — Docusaurus/MkDocs/Hugo

---

## D. Totally Fucked Up (Honest Assessment)

### D1. Persistent gopls `kv/v2` false-positive errors (NON-BLOCKING)

gopls reports "kv/v2 is not in your go.mod file" for `integration/`, `pebble/`, and `example/todo/` even though the replace directives ARE present and `go test`/`go build` pass cleanly. This is a **gopls workspace caching bug** — it doesn't resolve `replace` directives for modules not in the workspace `use` list.

**Impact:** Annoying red squiggles in editors. Zero functional impact.
**Fix:** Would need `kv` added to `go.work` (it may already be) or gopls restarted.

### D2. Pre-commit hook (buildflow) is fragile

The `buildflow` pre-commit hook runs `go mod tidy` on ALL modules. If any module has a stale go.sum (common after dependency bumps), the entire commit fails. This happened twice this session.

**Impact:** Commits blocked by unrelated modules' go.sum drift.
**Fix:** Consider a script that runs `go mod tidy` across all modules as part of `nix fmt` or a separate `nix run .#tidy` target.

### D3. The `:memory:` SQLite flakiness was pre-existing for weeks

The turso tests had a ~20% failure rate under `-race` that was apparently accepted as "just flaky." The root cause (LibSQL native engine resource exhaustion from too many simultaneous `:memory:` databases) was never diagnosed until today.

**Lesson:** Flaky tests are never acceptable. They indicate real bugs (in this case, a production race in CheckpointScheduler.Stop()).

---

## E. What We Should Improve (Architectural Reflections)

### E1. **Turso sync.go — OpenSyncWithConfig is still largely untestable**

The `syncEngine` interface extraction was good, but `OpenSyncWithConfig()` itself (which calls `tursoclient.NewTursoSyncDb`) is still untestable without a live server. The pattern of "constructor that needs network" is a testability anti-pattern.

**Suggestion:** Extract a `SyncDBOpener` interface or factory function that can be replaced in tests. This would bring turso package coverage to 90%+.

### E2. **Query module has untested error re-exports**

`query/errors.go` has ~10 functions at 0% coverage. These are thin wrappers around `error-family` constructors. They should either be tested with a simple table test, or removed if they're not adding value over direct `error-family` usage.

**Question:** Are consumers actually using `query.NewConflict()` etc.? If not, these are dead public API surface.

### E3. **Pebble adapter error paths need testing**

The `Set`, `Delete`, `Batch`, `Commit` methods all show 71-75% coverage — the error branches (closed store, write failure) are untested. These are the most important paths to test for a storage adapter.

### E4. **No integration test for the full Turso Backend lifecycle**

`turso/backend_test.go` tests individual stores but doesn't verify the Backend facade end-to-end: EventStore → SnapshotStore → CheckpointStore sharing one DB with the full schema.

### E5. **Missing benchmark: pebble vs SQL event store**

A head-to-head benchmark would help consumers choose between backends and provide a regression baseline. The pebble module already has `benchmark_test.go` but there's no cross-backend comparison.

---

## F. Top 25 Things To Do Next

| # | Task | Impact | Effort | Category |
|---|------|--------|--------|----------|
| 1 | Query `errors.go` table test (10 funcs at 0%) | Medium | 15min | Coverage |
| 2 | Pebble adapter error-branch tests (closed store) | High | 30min | Coverage |
| 3 | Turso `OpenSyncWithConfig` factory extraction | High | 45min | Testability |
| 4 | Prometheus metrics exporter | High | 2h | Feature |
| 5 | Structured logging middleware (slog) | High | 2h | Feature |
| 6 | Pebble vs SQL event store benchmark | Medium | 1h | Benchmark |
| 7 | Pebble golden test (CBOR envelope) | Medium | 1h | Regression |
| 8 | Turso Backend facade integration test | Medium | 30min | Coverage |
| 9 | Schema registry middleware | High | 3h | Feature |
| 10 | Distributed checkpointing (ADR-0018) | High | 4h | Feature |
| 11 | Streaming event reads (`StreamLoader`) | Medium | 2h | Feature |
| 12 | cqrs-gen v2 improvements | Medium | 3h | Tooling |
| 13 | `nix run .#tidy` target for all-module go mod tidy | Low | 20min | DX |
| 14 | MemorySnapshotStore golden test | Low | 30min | Regression |
| 15 | Distributed tracing propagation | Medium | 3h | Feature |
| 16 | gRPC transport adapter | High | 4h | Feature |
| 17 | Documentation site (Hugo/MkDocs) | Medium | 4h | Docs |
| 18 | Add `go.work` entry for kv module if missing | Low | 5min | Fix |
| 19 | Pebble `prefixUpperBound` edge-case test | Low | 15min | Coverage |
| 20 | Query `Dispatcher.Close()` error path test | Low | 10min | Coverage |
| 21 | Storage snapshot store edge-case tests | Medium | 1h | Coverage |
| 22 | NATS/Redis Stream adapter | High | 4h | Feature |
| 23 | jsonv2 codec experiment | Low | 2h | Experiment |
| 24 | Arena allocation experiment | Low | 3h | Experiment |
| 25 | WASM target for decider module | Medium | 4h | Experiment |

---

## G. Top Question I Cannot Figure Out Myself

**#1: Should the `query/errors.go` re-export functions (`NewConflict`, `NewTransient`, `NewCorruption`, `Wrap`, `WrapRejection`, etc.) be kept or removed?**

These are thin wrappers around `go-error-family` that exist in the `query` package. They're at 0% coverage, meaning no test uses them. But this is a **library** — consumers outside this repo may depend on them. The AGENTS.md explicitly says "Public API surface IS the product" and "Zero internal consumers is the EXPECTED state."

**The dilemma:** Testing them is trivial but adds maintenance. Removing them is a breaking API change. Keeping them untested leaves a coverage gap.

**My recommendation:** Keep them (they're public API), add a single table test that calls each function and verifies the error family classification. 15 minutes of work, brings query coverage from 79% to ~90%.

---

## Module Health Dashboard

| Module | Coverage | Status | Notes |
|--------|----------|--------|-------|
| event | 93.0% | ✅ Green | Core, stable |
| command | 96.9% | ✅ Green | |
| query | 79.0% | ⚠️ Yellow | errors.go at 0% |
| decider | 99.4% | ✅ Excellent | |
| id | 97.5% | ✅ Green | |
| dispatcher | 98.0% | ✅ Green | |
| schema | 91.4% | ✅ Green | |
| snapshot | 88.9% | ✅ Green | |
| memory | 98.5% | ✅ Excellent | |
| catalog | 84.5% | ✅ Green | |
| middleware | 93.9% | ✅ Green | |
| integration | 92.3% | ✅ Green | |
| storage | 82.1% | ⚠️ Yellow | SQL error paths |
| projection | 90.8% | ✅ Green | |
| signing | 94.5% | ✅ Green | |
| encryption | 86.9% | ✅ Green | |
| otel | 97.3% | ✅ Green | |
| watermill | 94.3% | ✅ Green | |
| pebble | 82.9% | ⚠️ Yellow | Adapter error paths |
| codec | 88.9% | ✅ Green | |
| kv | 94.9% | ✅ Green | |
| turso | 79.1% | ⚠️ Yellow | OpenSyncWithConfig needs network |
| listing | 94.9% | ✅ Green | |
| turso/indexing | 86.7% | ✅ Green | |

**All 23 modules: build ✅, test ✅, lint ✅, race ✅**
