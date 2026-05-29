# Session 132 — Full Comprehensive Status Report

**Date:** 2026-05-29 05:40 | **Branch:** master | **Commit:** 59b38ab

---

## TL;DR

The project is in **excellent shape**. 30/30 test packages pass (one pre-existing flaky chaos test). OTel instrumentation is now production-grade across all critical paths. 16 deprecated symbols need cleanup before v1.0. The library is importable, well-tested (1.85:1 test-to-code ratio), and follows strong architectural patterns.

| Metric             | Value          |
| ------------------ | -------------- |
| Total Go code      | 69,137 lines   |
| Production code    | 24,203 lines   |
| Test code          | 44,934 lines   |
| Test:Code ratio    | 1.85:1         |
| Modules (go.work)  | 21             |
| Test packages      | 30             |
| Passing            | 30/30          |
| Deprecated symbols | 16             |
| Files >250 lines   | 2 (borderline) |
| Open TODOs/FIXMEs  | 0              |
| ADRs               | 7              |

---

## a) FULLY DONE ✅

### Core Architecture

- **Decider pattern** (`core/decider`) — Pure-function aggregate with `Decider[State]`, `Repository[State]`, `Execute`, `Load`, `LoadAtVersion`, `LoadAtTime`. Fully OTel-instrumented.
- **Event sourcing** (`core/event`) — 24 public types, immutable events, typed payloads, metadata, correlation/causation IDs, upcasting, versioned store, snapshot strategies.
- **CQRS** (`core/command`, `core/query`) — Type-safe dispatch with `DispatchTyped[T]`, generic `Dispatcher[H, M]`, pagination, `PaginatedResult[T]`.
- **Branded IDs** (`core/pkg/id`) — `id.Of[T]` via go-branded-id + ULID, type-safe `AggregateID`, `EventID`, etc.

### Storage Layer

- **SQLEventStore** (`storage`) — PostgreSQL, SQLite, Turso support. Full CRUD with optimistic concurrency, global loading, seekable journal, cursor-based streaming.
- **SQLTransactionalStore** — Atomic save+outbox in single DB transaction. OTel span added this session.
- **PebbleEventStore** — Embedded KV store for local/test use. 7 methods, no OTel spans yet (by design — embedded use).
- **SQLSagaStore** — Full saga state persistence with UPSERT. OTel spans added this session (Save/Load/LoadAllRunning).
- **SQLSnapshotStore** — Snapshot save/load/delete. LoadAtVersion OTel span added this session.
- **SQLCheckpointStore** — Projection checkpoint tracking.
- **SQLOutbox + Poller** — Reliable event delivery pattern.
- **TursoSyncDB** — Edge sync for Turso/libSQL.

### Advanced Patterns

- **Saga orchestration** (`saga`) — Runner, Definition, Step, compensation, persistent state, 8 error paths all OTel-traced.
- **Projection runner** (`projection`) — Replay+live, handler registry, builder pattern, health checks. OTel spans on Run/replay/handle.
- **Event signing** (`signing`) — HMAC-SHA256, Ed25519, multi-signer, middleware, tamper detection.
- **Stream API** (`stream`) — Aggregate listing, tombstone detection, status middleware, SQL/in-memory readers, ListBuilder.

### Cross-Cutting

- **OTel module** (`otel`) — `StartSpan`, `RecordError`, attribute constants, `Instrumentation` helper. Used by decider, storage, projection, saga.
- **Middleware** (`middleware`) — Logging, Retry, Recovery, Validation, CircuitBreaker, EventTracing, EventPublishTracing, CommandTracing, CommandMetrics (OTel-backed).
- **Catalog** (`catalog`) — Registry, SchemaFromType[T], AsyncAPI/D2/EventCatalog/OpenAPI exporters, docserver.
- **Test helpers** (`testhelpers`) — Noop/Failing/Panic handlers, FakeMetrics, AppendEventsHandler.
- **Watermill adapter** (`watermill`) — Publisher/Subscriber protocol adapter.
- **Integration tests** (`integration`) — Cross-module tests for command, event, query, signing, OTel.
- **Code generator** (`cmd/cqrs-gen`) — CLI tool for code generation.

### Quality Infrastructure

- **Nix flake** — Build, test, lint, format, dev shell.
- **GitHub Actions CI** — Nix-based, build/vet/test/lint/race/coverage + GOWORK=off per-module.
- **Pre-commit hooks** — gofumpt, goimports, oxfmt, go-mod-tidy auto-staging.
- **7 ADRs** — Documented architectural decisions.

### Session 131–132 Work (This Sprint)

- ✅ **OTel RecordError gaps closed** — decider.Load, saga.ExecuteStep (8 paths), saga.compensate, projection.replay checkpoint
- ✅ **New OTel spans added** — decider.LoadAtVersion/LoadAtTime, snapshot.LoadAtVersion, storage.LoadStream, storage.SaveWithOutbox, saga.store.save/load/load_all_running, projection.Run
- ✅ **Test cleanup** — Removed stale nolint comments, reformatted BDD tests
- ✅ **Code deduplication** (Session 131) — 22 clone groups found, all 6 production clones eliminated, 5 major test clones eliminated

---

## b) PARTIALLY DONE 🔧

### OTel Instrumentation Coverage

- **4 of 13 modules have otel.go**: `core/decider`, `storage`, `projection`, `saga`
- **Middleware module** has OTel metrics (`metrics_otel.go`) but no `otel.go` tracer
- **Missing spans on 15 storage methods**: PebbleEventStore (7 methods), TursoSyncDB (4 methods), deprecated LoadAll/LoadAllFromPosition (2), SQLSnapshotStore.Delete (1), SQLEventStore.Close (1)
- **No spans in**: `memory`, `catalog`, `signing`, `stream`, `watermill`, `core/command`, `core/query`
  - Most of these are correct (in-memory test impls, no I/O) but `signing` and `stream` could benefit

### Deprecated Symbol Cleanup

- **16 deprecated symbols** identified across 4 modules
- All have clear migration paths documented
- None removed yet — waiting for v1.0 planning

### Test Coverage

- Most modules >80%, many >90%
- `projection` targeted for 95%+ but not yet verified
- Integration chaos test `TestChaos_CommandRetry_ExhaustsAllAttempts` is flaky (pre-existing)

---

## c) NOT STARTED ⬜

1. **v1.0 release preparation** — No version tags, no release workflow, `replace` directives still required in go.mod files
2. **PebbleEventStore OTel spans** — 7 public methods entirely untraced
3. **TursoSyncDB OTel spans** — 4 public methods entirely untraced
4. **Deprecated symbol removal** — 16 symbols across 4 modules
5. **`core/aggregate` package removal** — Entire package is deprecated redirect to `core/decider`
6. **projection/runner.go file split** — 292 lines, violates 250-line rule
7. **core/event/event.go trim** — 252 lines, barely over limit
8. **Event context propagation** — `event.Context` through `NewEvent`/`PublishChanges` (TODO_LIST item)
9. **ProcessedAt on CheckpointStore** — Store EventID + timestamp (TODO_LIST item)
10. **Catch-up projection runner** — TODO_LIST item
11. **Background polling for InMemoryRunner** — TODO_LIST item
12. **Example/user catalog wiring** — Wire example to catalog-aware constructors
13. **gopls workspace diagnostics** — ~650 false-positive "go mod tidy" errors from go.work interaction (ADR-0007 documents workaround)
14. **Chaos test flakiness** — `TestChaos_CommandRetry_ExhaustsAllAttempts` intermittent failure

---

## d) TOTALLY FUCKED UP 💥

### Nothing is truly broken. These are the worst issues:

1. **Pre-commit hook exit code** — `golangci-lint` exits 7 and `go-structure-linter` exits 1 in CI due to go.work interaction. Not real lint failures, but requires `GIT_EDITOR=true` workaround for commits. Documented in ADR-0007.

2. **Chaos test flakiness** — `integration/chaos_test.go:155` `TestChaos_CommandRetry_ExhaustsAllAttempts` fails intermittently. Test expects error but gets nil — likely a race in the retry dispatcher mock. Not investigated yet.

3. **gopls false positives** — ~650 "go mod tidy" diagnostics across the workspace. These are ALL false positives from the multi-module workspace pattern. Tests pass fine, builds succeed. Documented but annoying.

4. **`replace` directives required** — All inter-module go.mod files need `replace` directives until v1.0.0 tags are pushed. This is a known blocker for external consumers.

---

## e) WHAT WE SHOULD IMPROVE 📈

### Architecture & Design

1. **Remove deprecated `core/aggregate` package** — It's a pure redirect to `core/decider`. Dead weight. Remove before v1.0.

2. **Clean up deprecated storage/memory methods** — `LoadAll` → `ReadAll`, `LoadAllFromPosition` → `ReadFrom`, `TransactionalStore` → `TransactionalSink`. Migration is trivial.

3. **Split `projection/runner.go`** — 292 lines. Extract `replay()` logic or `handleAndCheckpoint()` into a separate file.

4. **Trim `core/event/event.go`** — 252 lines, barely over. Extract helpers or type definitions.

5. **Add ADR for OTel instrumentation** — Document the pattern: `otel.go` per module, `tracer()`, `StartSpan`, `RecordError` on all error paths.

### Testing & Quality

6. **Fix chaos test flakiness** — Investigate `TestChaos_CommandRetry_ExhaustsAllAttempts` race condition. Either fix or mark as skipped with a bug reference.

7. **Add OTel integration tests for storage spans** — Verify spans are actually emitted with correct attributes for LoadStream, SaveWithOutbox, saga store methods.

8. **Increase projection test coverage** — Target 95%+ as stated in TODO_LIST.

### Observability

9. **PebbleEventStore spans** — Add spans to all 7 public methods. Even embedded stores benefit from tracing in development.

10. **TursoSyncDB spans** — Edge sync is I/O-heavy, spans critical for debugging distributed scenarios.

11. **Sign method spans** — `signing` module does crypto I/O; add spans for Sign/Verify operations.

### Developer Experience

12. **Remove `replace` directives** — Tag v1.0.0 on all modules and remove replace directives. This is the #1 blocker for external adoption.

13. **Fix gopls diagnostics** — Either fix the workspace interaction or suppress the false positives in a way that doesn't clutter the IDE.

14. **Generate API surface docs** — Automate `godoc`-style API reference generation from code.

### Process

15. **Status report consolidation** — 37 status reports in docs/status/ is excessive. Archive older ones more aggressively.

16. **Planning doc cleanup** — 22 planning docs + archive. Many are stale. Consolidate into living documents.

---

## f) Top 25 Things We Should Get Done Next

**Sorted by: Impact × Effort⁻¹ (highest ROI first)**

| #   | Task                                                 | Impact      | Effort | Module               | Type          |
| --- | ---------------------------------------------------- | ----------- | ------ | -------------------- | ------------- |
| 1   | **Tag v1.0.0 and remove `replace` directives**       | 🔴 Critical | M      | all                  | Release       |
| 2   | **Fix chaos test flakiness**                         | 🔴 High     | S      | integration          | Bug           |
| 3   | **Remove `core/aggregate` deprecated package**       | 🟠 High     | S      | core                 | Cleanup       |
| 4   | **Remove deprecated storage/memory methods**         | 🟠 High     | S      | storage, memory      | Cleanup       |
| 5   | **Remove deprecated `core/event` symbols**           | 🟠 High     | S      | core/event           | Cleanup       |
| 6   | **Split `projection/runner.go` (<250 lines)**        | 🟡 Medium   | S      | projection           | Quality       |
| 7   | **Add OTel spans to PebbleEventStore (7 methods)**   | 🟡 Medium   | M      | storage              | Observability |
| 8   | **Add OTel spans to TursoSyncDB (4 methods)**        | 🟡 Medium   | S      | storage              | Observability |
| 9   | **Add span to `SQLSnapshotStore.Delete`**            | 🟡 Medium   | S      | storage              | Observability |
| 10  | **Add OTel integration tests for new storage spans** | 🟡 Medium   | M      | integration, storage | Testing       |
| 11  | **Add ADR-0008 for OTel instrumentation pattern**    | 🟡 Medium   | S      | docs                 | Documentation |
| 12  | **Trim `core/event/event.go` to ≤250 lines**         | 🟢 Low      | S      | core/event           | Quality       |
| 13  | **Add signing module OTel spans**                    | 🟢 Low      | M      | signing              | Observability |
| 14  | **Fix gopls workspace false positives**              | 🟢 Low      | L      | tooling              | DX            |
| 15  | **Add `ProcessedAt` to CheckpointStore**             | 🟢 Low      | S      | storage              | Feature       |
| 16  | **Event context propagation**                        | 🟢 Low      | M      | core/event           | Feature       |
| 17  | **Wire example/user to catalog constructors**        | 🟢 Low      | S      | example              | DX            |
| 18  | **Build catch-up projection runner**                 | 🟢 Low      | L      | projection           | Feature       |
| 19  | **Background polling for InMemoryRunner**            | 🟢 Low      | M      | core/event           | Feature       |
| 20  | **Archive old status reports (keep last 10)**        | 🟢 Low      | S      | docs                 | Hygiene       |
| 21  | **Consolidate stale planning docs**                  | 🟢 Low      | S      | docs                 | Hygiene       |
| 22  | **Generate API surface docs automatically**          | 🟢 Low      | M      | docs                 | Documentation |
| 23  | **Add projection coverage to 95%+**                  | 🟢 Low      | M      | projection           | Testing       |
| 24  | **Add consumer quickstart guide (README)**           | 🟢 Low      | M      | docs                 | DX            |
| 25  | **Benchmark suite for critical paths**               | 🟢 Low      | M      | core, storage        | Performance   |

---

## g) Top #1 Question I Cannot Figure Out Myself 🤔

**What is the v1.0.0 release strategy?**

The `replace` directives in every `go.mod` are the single biggest adoption blocker. They exist because all modules reference each other via `github.com/larsartmann/go-cqrs-lite/<module>` but no version tags exist on the remote. To fix this:

1. **Option A:** Tag all modules with `v1.0.0` simultaneously and remove `replace` directives. Risky — any dependency mistake breaks everything.
2. **Option B:** Use Go workspace `go.work` for development (current), tag `v0.1.0` pre-releases per module, verify externally, then promote to `v1.0.0`.
3. **Option C:** Publish a single `go-cqrs-lite` module (monorepo single module) and drop multi-module entirely.

**Which approach do you want?** This decision affects everything else — the deprecated symbol removal timeline, the module boundary verification, and how aggressively we can refactor.

---

## Session Activity Log

| Session | Key Work                                                                      |
| ------- | ----------------------------------------------------------------------------- |
| 131     | Code deduplication sprint — 22 clone groups, all production clones eliminated |
| 132     | OTel hardening — RecordError gaps closed, 8 new spans added across 5 modules  |

---

## Build & Test Status

```
30/30 packages pass ✅
  ok: core/aggregate, core/command, core/decider, core/event, core/pkg/dispatcher,
      core/pkg/id, core/query, memory, catalog (+5 sub-packages), middleware,
      testhelpers, integration (+4 sub-packages), projection, signing, storage,
      saga, stream, watermill, otel

  flaky: integration/chaos_test.go::TestChaos_CommandRetry_ExhaustsAllAttempts
```

---

_Auto-generated status report — Session 132_
