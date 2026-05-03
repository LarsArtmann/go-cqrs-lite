# Architecture Improvements Execution Plan

**Date:** 2026-04-29 22:42  
**Context:** Session 10 identified ghost systems, shallow modules, and untested code. This plan addresses them systematically.

---

## Part 1: High-Level Plan (30–100 min tasks, max 24)

Sorted by **impact ÷ effort** (Pareto principle). Start at the top.

| #   | Task                                        | Module         | Effort  | Impact     | Customer Value       | Why                                                                                                    |
| --- | ------------------------------------------- | -------------- | ------- | ---------- | -------------------- | ------------------------------------------------------------------------------------------------------ |
| 1   | **Snapshot integration tests**              | core/aggregate | 30 min  | **HIGH**   | Reliability          | Fixes 12.7% coverage drop. Tests the ghost system we built.                                            |
| 2   | **Generic ErrorRecovery + ErrorValidation** | middleware     | 45 min  | **HIGH**   | Maintainability      | Eliminates copy-paste (Command/Event share identical recovery/validation). Fix once, fixed everywhere. |
| 3   | **Outbox seam interface + memory adapter**  | core/event     | 60 min  | **HIGH**   | Production readiness | Answers the #1 architectural question: atomic save+publish without coupling core to databases.         |
| 4   | **Query generic result types**              | core/query     | 180 min | **HIGH**   | Type safety          | `Handler[T any]` eliminates runtime type assertions. Compile-time guarantees.                          |
| 5   | **CatalogBuilder wraps Registry**           | catalog        | 40 min  | **MEDIUM** | Maintainability      | Two builders with identical internals. Consolidate for locality.                                       |
| 6   | **MemorySnapshotStore deep copy**           | memory         | 15 min  | **MEDIUM** | Correctness          | `Snapshot.State []byte` is shared on load. Silent data corruption risk.                                |
| 7   | **OpenTelemetry tracing middleware**        | middleware     | 60 min  | **MEDIUM** | Observability        | Production systems need distributed tracing. We have zero.                                             |
| 8   | **Remove core→memory replace directive**    | core/go.mod    | 30 min  | **MEDIUM** | Modularity           | Circular dependency means core cannot be published independently.                                      |
| 9   | **Event upcasting infrastructure design**   | core/event     | 60 min  | **MEDIUM** | Evolution            | Schema evolution is required for any long-lived event store. Design now, implement later.              |
| 10  | **Add event.Builder benchmark**             | core/event     | 15 min  | **LOW**    | Performance          | Baseline for event construction hot path.                                                              |
| 11  | **PostgreSQL event store design**           | new/storage    | 90 min  | **HIGH**   | Production readiness | In-memory is fine for tests; production needs persistence. Design the interface now.                   |
| 12  | **Saga/process manager design**             | new/saga       | 90 min  | **MEDIUM** | Feature expansion    | Long-running workflows are a common CQRS need. Research and design.                                    |

**Pareto rule:** Tasks 1–3 deliver ~51% of the value in ~135 min (the 1%).  
**Tasks 1–6 deliver ~80% of the value in ~370 min (the 20%).**

---

## Part 2: Micro-Task Plan (max 12 min each, max 60 tasks)

Each high-level task broken into ≤12-minute chunks. Sorted by the same impact/effort ratio.

### Task 1: Snapshot Integration Tests (30 min → 6 micro-tasks)

| #   | Micro-Task                                                               | Time  | Verification                                   |
| --- | ------------------------------------------------------------------------ | ----- | ---------------------------------------------- |
| 1.1 | Create `TestRepositoryWithSnapshot_SaveAndLoad` scaffolding              | 5 min | File compiles                                  |
| 1.2 | Test: save events → save snapshot → load more events → load via snapshot | 8 min | Assert state == snapshot + replayed events     |
| 1.3 | Test: snapshot not found → fallback to loading all events                | 5 min | Assert version correct, no snapshot used       |
| 1.4 | Test: `SetVersion` sets aggregate version directly                       | 5 min | Assert Core.Version() matches SetVersion input |
| 1.5 | Test: `NewRepositoryWithSnapshot` construction                           | 3 min | Assert not nil, fields set                     |
| 1.6 | Test: snapshot at version N, replay from N+1                             | 8 min | Assert only events after snapshot are replayed |

### Task 2: Generic Middleware Core (45 min → 8 micro-tasks)

| #   | Micro-Task                                                      | Time  | Verification                                |
| --- | --------------------------------------------------------------- | ----- | ------------------------------------------- |
| 2.1 | Define `ErrorHandler[M any]` and `ErrorMiddleware[M any]` types | 5 min | Compiles, Command/Event satisfy constraint  |
| 2.2 | Extract `ErrorRecovery[M any]()` generic function               | 8 min | Unit test with panic recovery               |
| 2.3 | Extract `ErrorValidation[M any]()` generic function             | 8 min | Unit test with pass/fail validation         |
| 2.4 | Update `CommandRecovery` to delegate to `ErrorRecovery`         | 5 min | Existing CommandRecovery tests still pass   |
| 2.5 | Update `EventRecovery` to delegate to `ErrorRecovery`           | 5 min | Existing EventRecovery tests still pass     |
| 2.6 | Update `CommandValidation` to delegate to `ErrorValidation`     | 5 min | Existing CommandValidation tests still pass |
| 2.7 | Update `EventValidation` to delegate to `ErrorValidation`       | 5 min | Existing EventValidation tests still pass   |
| 2.8 | Add generic core tests: panic recovery, validation, type safety | 8 min | New tests pass                              |

### Task 3: Outbox Seam (60 min → 6 micro-tasks)

| #   | Micro-Task                                                     | Time   | Verification                                    |
| --- | -------------------------------------------------------------- | ------ | ----------------------------------------------- |
| 3.1 | Research outbox patterns (Go libs, existing code)              | 10 min | Notes in plan file                              |
| 3.2 | Design `OutboxStore` interface: `SaveOutbox`, `PublishPending` | 10 min | Interface documented, no implementation yet     |
| 3.3 | Write `OutboxStore` interface in `core/event/outbox.go`        | 8 min  | Compiles, integrates with existing Event type   |
| 3.4 | Create `MemoryOutboxStore` in memory module                    | 12 min | Implements OutboxStore, uses map[string][]Event |
| 3.5 | Add outbox tests: save, publish, idempotency                   | 15 min | Coverage >80% for new file                      |
| 3.6 | Document outbox usage pattern in AGENTS.md                     | 5 min  | Clear example with Repository + Outbox          |

### Task 4: Query Generic Result Types (180 min → 12 micro-tasks)

| #    | Micro-Task                                                | Time   | Verification                                    |
| ---- | --------------------------------------------------------- | ------ | ----------------------------------------------- |
| 4.1  | Design `Query[T any]` interface                           | 15 min | Documented, backward-compatible path considered |
| 4.2  | Update `query.Handler` to `Handler[T any]`                | 10 min | Core compiles                                   |
| 4.3  | Update `Dispatcher.Register` to generic `Register[T any]` | 15 min | Core compiles                                   |
| 4.4  | Update `Dispatcher.Dispatch` to generic `Dispatch[T any]` | 15 min | Core compiles                                   |
| 4.5  | Update `query_test.go` for generic types                  | 20 min | All query tests pass                            |
| 4.6  | Update `benchmark_test.go` for generic types              | 10 min | Benchmarks pass                                 |
| 4.7  | Update middleware QueryMetrics for generics               | 10 min | Tests pass                                      |
| 4.8  | Update middleware QueryLogging for generics               | 10 min | Tests pass                                      |
| 4.9  | Update middleware QueryRecovery for generics              | 10 min | Tests pass                                      |
| 4.10 | Update middleware QueryRetry for generics                 | 10 min | Tests pass                                      |
| 4.11 | Update middleware QueryValidation for generics            | 10 min | Tests pass                                      |
| 4.12 | Verify ALL tests pass across all modules                  | 20 min | `nix run .#test` green                          |

### Task 5: CatalogBuilder Wraps Registry (40 min → 6 micro-tasks)

| #   | Micro-Task                                                   | Time   | Verification                  |
| --- | ------------------------------------------------------------ | ------ | ----------------------------- |
| 5.1 | Analyze CatalogBuilder vs Registry duplication               | 5 min  | Document exact overlap        |
| 5.2 | Add `*catalog.Registry` field to `CatalogBuilder`            | 5 min  | Compiles                      |
| 5.3 | Delegate `AddService`, `AddDomain`, `AddCommand` to Registry | 10 min | Existing tests pass           |
| 5.4 | Update `CatalogBuilder.Build()` to use `Registry.Build()`    | 5 min  | Existing tests pass           |
| 5.5 | Remove duplicated fields/methods from CatalogBuilder         | 10 min | No behavior changes           |
| 5.6 | Verify catalog tests pass                                    | 5 min  | `go test ./catalog/...` green |

### Task 6: MemorySnapshotStore Deep Copy (15 min → 4 micro-tasks)

| #   | Micro-Task                                                        | Time  | Verification                        |
| --- | ----------------------------------------------------------------- | ----- | ----------------------------------- |
| 6.1 | Analyze current Load/LoadAtVersion copy behavior                  | 3 min | Confirm State is shared             |
| 6.2 | Add `copy(snapshot.State)` in `Load()`                            | 5 min | Compiles                            |
| 6.3 | Add `copy(snapshot.State)` in `LoadAtVersion()`                   | 3 min | Compiles                            |
| 6.4 | Add test: modify loaded snapshot state, assert original unchanged | 5 min | Test fails before fix, passes after |

### Task 7: OpenTelemetry Tracing Middleware (60 min → 6 micro-tasks)

| #   | Micro-Task                           | Time   | Verification                   |
| --- | ------------------------------------ | ------ | ------------------------------ |
| 7.1 | Research OTel Go middleware patterns | 10 min | Notes on approach              |
| 7.2 | Add `CommandTracing` middleware      | 10 min | Creates span per command       |
| 7.3 | Add `EventTracing` middleware        | 10 min | Creates span per event         |
| 7.4 | Add `QueryTracing` middleware        | 10 min | Creates span per query         |
| 7.5 | Add tests verifying span creation    | 15 min | Mock tracer, assert span count |
| 7.6 | Add tracing example to AGENTS.md     | 5 min  | Clear usage pattern            |

### Task 8: Remove core→memory Replace (30 min → 4 micro-tasks)

| #   | Micro-Task                                                                  | Time   | Verification                    |
| --- | --------------------------------------------------------------------------- | ------ | ------------------------------- |
| 8.1 | Identify which core tests import memory                                     | 5 min  | List of files                   |
| 8.2 | Move memory-dependent tests to `core/aggregate/_integration/` or new module | 15 min | Core tests pass without memory  |
| 8.3 | Remove memory replace from `core/go.mod`                                    | 5 min  | `GOWORK=off go mod tidy` passes |
| 8.4 | Verify isolated build: `cd core && GOWORK=off go build ./...`               | 5 min  | No errors                       |

### Task 9: Event Builder Benchmark (15 min → 4 micro-tasks)

| #   | Micro-Task                                | Time  | Verification        |
| --- | ----------------------------------------- | ----- | ------------------- |
| 9.1 | Create `builder_benchmark_test.go`        | 5 min | File compiles       |
| 9.2 | Add `BenchmarkBuilder_Build`              | 5 min | Runs, outputs ns/op |
| 9.3 | Add `BenchmarkBuilder_Build_WithMetadata` | 3 min | Runs, outputs ns/op |
| 9.4 | Run benchmark and record baseline         | 2 min | Results documented  |

### Task 10: Event Upcasting Design (60 min → 5 micro-tasks)

| #    | Micro-Task                                             | Time   | Verification         |
| ---- | ------------------------------------------------------ | ------ | -------------------- |
| 10.1 | Research upcasting patterns (EventStoreDB, Axon, etc.) | 10 min | Notes in plan        |
| 10.2 | Design `EventUpcaster` interface                       | 10 min | Documented           |
| 10.3 | Design `VersionedEvent` type                           | 10 min | Documented           |
| 10.4 | Write design doc in `docs/planning/EVENT_UPCASTING.md` | 20 min | Reviewable           |
| 10.5 | Create example upcaster (simple field rename)          | 10 min | Demonstrates concept |

### Task 11: PostgreSQL Event Store Design (90 min → 7 micro-tasks)

| #    | Micro-Task                                              | Time   | Verification             |
| ---- | ------------------------------------------------------- | ------ | ------------------------ |
| 11.1 | Research sqlc + event sourcing patterns                 | 10 min | Notes on schema          |
| 11.2 | Design SQL schema (events table, outbox table, indexes) | 15 min | Documented               |
| 11.3 | Write `schema.sql` with migrations                      | 10 min | Valid SQL                |
| 11.4 | Generate Go types with sqlc                             | 10 min | Compiles                 |
| 11.5 | Design `SQLStore` adapter interface                     | 15 min | Implements `event.Store` |
| 11.6 | Write design doc in `docs/planning/SQL_EVENT_STORE.md`  | 20 min | Reviewable               |
| 11.7 | Create stub implementation (returns not-implemented)    | 10 min | Compiles                 |

### Task 12: Saga/Process Manager Design (90 min → 6 micro-tasks)

| #    | Micro-Task                                             | Time   | Verification         |
| ---- | ------------------------------------------------------ | ------ | -------------------- |
| 12.1 | Research saga patterns (choreography vs orchestration) | 15 min | Notes in plan        |
| 12.2 | Design `Saga` interface                                | 15 min | Documented           |
| 12.3 | Design `ProcessManager` type                           | 15 min | Documented           |
| 12.4 | Design saga state persistence interface                | 15 min | Documented           |
| 12.5 | Write design doc in `docs/planning/SAGA_DESIGN.md`     | 20 min | Reviewable           |
| 12.6 | Create example saga (order fulfillment)                | 10 min | Demonstrates concept |

---

## Execution Graph

```mermaid
graph TD
    subgraph "Phase 1: Fix Ghost Systems (1%)"
        A[1.1 Snapshot tests scaffolding] --> B[1.2 Snapshot+replay test]
        B --> C[1.3 Snapshot not-found fallback]
        C --> D[1.4 SetVersion test]
        D --> E[1.5 NewRepositoryWithSnapshot test]
        E --> F[1.6 Snapshot version replay test]
    end

    subgraph "Phase 2: Deepen Modules (4%)"
        F --> G[2.1 ErrorHandler types]
        G --> H[2.2 ErrorRecovery generic]
        H --> I[2.3 ErrorValidation generic]
        I --> J[2.4-2.7 Update middleware]
        J --> K[2.8 Generic core tests]
        K --> L[3.1 Research outbox]
        L --> M[3.2 Design OutboxStore]
        M --> N[3.3 Write interface]
        N --> O[3.4 MemoryOutboxStore]
        O --> P[3.5 Outbox tests]
        P --> Q[3.6 Document outbox]
    end

    subgraph "Phase 3: Type Safety (20%)"
        Q --> R[4.1 Design Query[T]]
        R --> S[4.2-4.4 Update core types]
        S --> T[4.5-4.12 Update tests + middleware]
    end

    subgraph "Phase 4: Production Readiness"
        T --> U[6.1-6.4 Snapshot deep copy]
        U --> V[7.1-7.6 OTel tracing]
        V --> W[8.1-8.4 Remove replace directive]
    end

    subgraph "Phase 5: Future Design"
        W --> X[10.1-10.5 Upcasting design]
        X --> Y[11.1-11.7 SQL store design]
        Y --> Z[12.1-12.6 Saga design]
    end
```

---

## Rules for Execution

1. **Run tests after EVERY micro-task.** If tests fail, fix immediately.
2. **Commit after each smallest self-contained change.** Use detailed messages.
3. **Don't break the build.** If a change is too risky, stop and report.
4. **Only fix one concern per commit.**
5. **Use existing libraries.** Prefer stdlib > internal > external.
6. **Check for existing code** before writing from scratch.
7. **Keep functions under 30 lines, files under 250 lines.**
8. **Prefer composition over inheritance.**
9. **Update AGENTS.md** when discovering new patterns.
10. **Push when the session plan is complete.**

---

## Quality Gates

Before declaring done:

- [ ] All tests pass (`nix run .#test`)
- [ ] Zero lint issues (`nix run .#lint`)
- [ ] Coverage documented for changed packages
- [ ] AGENTS.md updated with new patterns
- [ ] Commit history is clean and descriptive
