# Comprehensive Execution Plan — go-cqrs-lite Project Groups Improvement

**Date:** 2026-05-18 15:10  
**Scope:** Multi-module monorepo hygiene, type safety, deduplication, documentation  
**Total Tasks:** 86  
**Max Task Duration:** 12 minutes  
**Sorting:** Pareto impact (1%→51% first, then 4%→64%, then 20%→80%)

---

## Pareto Summary

| Tier          | % of Tasks     | % of Value | Theme                                                                |
| ------------- | -------------- | ---------- | -------------------------------------------------------------------- |
| **1% → 51%**  | 28% (24 tasks) | 51%        | Module hygiene, DAG cleanup, file size compliance, type safety fixes |
| **4% → 64%**  | 39% (33 tasks) | 13%        | BDD tests, CatalogMeta consolidation, deduplication, AGENTS.md       |
| **20% → 80%** | 33% (29 tasks) | 16%        | TransactionalStore, Saga, CI/CD, documentation, coverage gaps        |

---

## Tier 1: The 1% → 51% of Value (CRITICAL — Do First)

These 24 tasks deliver 51% of the total value. They fix build integrity, DAG violations, file size compliance, and type safety issues that propagate across the entire codebase.

### 1.1 Module Hygiene — Remove `replace` Directives

| #     | Task                                                                          | Effort | Impact | Customer Value          | Module      | Verification            |
| ----- | ----------------------------------------------------------------------------- | ------ | ------ | ----------------------- | ----------- | ----------------------- |
| 1.1.1 | Remove `replace` block from `memory/go.mod` (core + testhelpers)              | 5min   | High   | Clean published modules | memory      | `go build ./...` passes |
| 1.1.2 | Remove `replace` block from `projection/go.mod` (core + memory + testhelpers) | 5min   | High   | Clean published modules | projection  | `go build ./...` passes |
| 1.1.3 | Remove `replace` block from `middleware/go.mod` (core + testhelpers)          | 5min   | High   | Clean published modules | middleware  | `go build ./...` passes |
| 1.1.4 | Remove `replace` block from `integration/go.mod` (all 6 modules)              | 5min   | High   | Clean published modules | integration | `go build ./...` passes |
| 1.1.5 | Remove `replace` block from `catalog/go.mod` (core)                           | 5min   | High   | Clean published modules | catalog     | `go build ./...` passes |
| 1.1.6 | Standardize `catalog/go.mod` version from `v0.0.0` to `v1.1.0`                | 3min   | High   | Consistent versioning   | catalog     | `go mod tidy` clean     |
| 1.1.7 | Standardize `middleware/go.mod` pseudo-version to `v1.1.0`                    | 3min   | High   | Consistent versioning   | middleware  | `go mod tidy` clean     |
| 1.1.8 | Standardize `integration/go.mod` mixed versions to `v1.1.0`                   | 5min   | High   | Consistent versioning   | integration | `go mod tidy` clean     |

### 1.2 GOWORK=off Verification

| #     | Task                                                        | Effort | Impact   | Customer Value          | Module     | Verification |
| ----- | ----------------------------------------------------------- | ------ | -------- | ----------------------- | ---------- | ------------ |
| 1.2.1 | Verify `GOWORK=off go build ./...` passes for `core/`       | 5min   | Critical | Published module builds | core       | No errors    |
| 1.2.2 | Verify `GOWORK=off go build ./...` passes for `memory/`     | 5min   | Critical | Published module builds | memory     | No errors    |
| 1.2.3 | Verify `GOWORK=off go build ./...` passes for `storage/`    | 5min   | Critical | Published module builds | storage    | No errors    |
| 1.2.4 | Verify `GOWORK=off go build ./...` passes for `catalog/`    | 5min   | Critical | Published module builds | catalog    | No errors    |
| 1.2.5 | Verify `GOWORK=off go build ./...` passes for `middleware/` | 5min   | Critical | Published module builds | middleware | No errors    |
| 1.2.6 | Verify `GOWORK=off go build ./...` passes for `projection/` | 5min   | Critical | Published module builds | projection | No errors    |

### 1.3 File Size Compliance (250-line limit)

| #     | Task                                                                                                            | Effort | Impact | Customer Value  | File    | Verification                        |
| ----- | --------------------------------------------------------------------------------------------------------------- | ------ | ------ | --------------- | ------- | ----------------------------------- |
| 1.3.1 | Split `storage/pebble_event_store.go` (321→<250 lines): extract CQRSAdapter methods to `pebble_adapter.go`      | 12min  | High   | Maintainability | storage | Both files <250 lines, build passes |
| 1.3.2 | Split `catalog/registry.go` (254→<250 lines): extract `ensureService` + helper methods to `registry_helpers.go` | 10min  | High   | Maintainability | catalog | Both files <250 lines, build passes |

### 1.4 Type Safety Fixes (`event.Version` → `int` IncompatibleAssign)

| #     | Task                                                                                          | Effort | Impact | Customer Value | File    | Verification            |
| ----- | --------------------------------------------------------------------------------------------- | ------ | ------ | -------------- | ------- | ----------------------- |
| 1.4.1 | Fix `event.Version` → `int` assignment at `storage/event_store_test.go:101`                   | 5min   | High   | Type safety    | storage | `go build ./...` passes |
| 1.4.2 | Fix `event.Version` → `int` assignments in `core/aggregate/aggregate_test.go` (2 occurrences) | 5min   | High   | Type safety    | core    | `go build ./...` passes |
| 1.4.3 | Fix `event.Version` → `int` assignments in `core/decider/decider_test.go` (2 occurrences)     | 5min   | High   | Type safety    | core    | `go build ./...` passes |
| 1.4.4 | Fix `event.Version` → `int` assignments in `core/decider/decider_bdd_test.go` (2 occurrences) | 5min   | High   | Type safety    | core    | `go build ./...` passes |
| 1.4.5 | Fix `event.Version` → `int` assignments in `core/event/event_test.go` (2 occurrences)         | 5min   | High   | Type safety    | core    | `go build ./...` passes |
| 1.4.6 | Fix `event.Version` → `int` assignment in `core/aggregate/repository_test.go`                 | 5min   | High   | Type safety    | core    | `go build ./...` passes |

### 1.5 Dead Code & Coverage

| #     | Task                                                                                     | Effort | Impact | Customer Value        | File    | Verification                  |
| ----- | ---------------------------------------------------------------------------------------- | ------ | ------ | --------------------- | ------- | ----------------------------- |
| 1.5.1 | Evaluate `catalog/internal/cattest/` — check if any imports exist outside package        | 5min   | High   | Dead code elimination | catalog | grep for imports              |
| 1.5.2 | Remove `catalog/internal/cattest/` package (454 lines, 0% coverage, no external imports) | 5min   | High   | Dead code elimination | catalog | Build + tests pass            |
| 1.5.3 | Verify coverage calculation excludes `example/todo/` (suspected 84.5% skew)              | 5min   | High   | Accurate metrics      | root    | Coverage ≥93% after exclusion |
| 1.5.4 | Add coverage exclusion for `example/todo/` in `flake.nix` or test config                 | 5min   | High   | Accurate metrics      | root    | `nix run .#coverage` ≥93%     |

---

## Tier 2: The 4% → 64% of Value (HIGH — Do Next)

These 33 tasks deliver the next 13% of value. They improve test coverage, consolidate duplicated code, and update documentation.

### 2.1 AGENTS.md Updates

| #     | Task                                                                                   | Effort | Impact | Customer Value       | Verification                     |
| ----- | -------------------------------------------------------------------------------------- | ------ | ------ | -------------------- | -------------------------------- |
| 2.1.1 | Update AGENTS.md: remove `replace` directive recommendation, add `go.work` guidance    | 10min  | Medium | Developer onboarding | AGENTS.md reflects current state |
| 2.1.2 | Update AGENTS.md: document `ErrPublisherClosed`, `RetryConfig.Validate()`, file splits | 8min   | Medium | Developer onboarding | AGENTS.md reflects current state |
| 2.1.3 | Update AGENTS.md: document `storage` no longer depends on `memory` (DAG fix)           | 5min   | Medium | Developer onboarding | AGENTS.md reflects current state |

### 2.2 BDD Tests for Type Safety

| #     | Task                                                                                   | Effort | Impact | Customer Value | Module  | Verification        |
| ----- | -------------------------------------------------------------------------------------- | ------ | ------ | -------------- | ------- | ------------------- |
| 2.2.1 | BDD tests for `event.Version`: `Int()`, `String()`, zero value, negative guard         | 10min  | Medium | Test coverage  | core    | Ginkgo suite passes |
| 2.2.2 | BDD tests for `event.SchemaVersion`: distinct from `Version`, parsing, invalid input   | 10min  | Medium | Test coverage  | core    | Ginkgo suite passes |
| 2.2.3 | BDD tests for `storage.OutboxStatus`: `Pending` value, string representation           | 10min  | Medium | Test coverage  | storage | Ginkgo suite passes |
| 2.2.4 | BDD tests for `sync.NodeID`: `ParseNodeID`, `MustParseNodeID`, `IsZero()`              | 10min  | Medium | Test coverage  | sync    | Ginkgo suite passes |
| 2.2.5 | BDD tests for `sync.SyncMessageType`: `Request` vs `Response`                          | 10min  | Medium | Test coverage  | sync    | Ginkgo suite passes |
| 2.2.6 | BDD tests for `query.Pagination`: `NewPagination`, `Offset()`, edge cases, uint safety | 10min  | Medium | Test coverage  | core    | Ginkgo suite passes |

### 2.3 CatalogMeta Consolidation

| #     | Task                                                                                           | Effort | Impact | Customer Value | Module | Verification       |
| ----- | ---------------------------------------------------------------------------------------------- | ------ | ------ | -------------- | ------ | ------------------ |
| 2.3.1 | Create `core/catalogmeta/catalogmeta.go` with `Base` struct (Name, Summary, Description)       | 8min   | Medium | DRY principle  | core   | Build passes       |
| 2.3.2 | Update `core/event/catalog.go`: embed `catalogmeta.Base` in `CatalogMeta`, add `AggregateType` | 8min   | Medium | DRY principle  | core   | Build + tests pass |
| 2.3.3 | Update `core/command/catalog.go`: embed `catalogmeta.Base` in `CatalogMeta`                    | 5min   | Medium | DRY principle  | core   | Build + tests pass |
| 2.3.4 | Update `core/query/catalog.go`: embed `catalogmeta.Base` in `CatalogMeta`                      | 5min   | Medium | DRY principle  | core   | Build + tests pass |
| 2.3.5 | Update all test files referencing `CatalogMeta` constructors for new embedded structure        | 10min  | Medium | DRY principle  | core   | All tests pass     |

### 2.4 Deduplication — Production Code

| #     | Task                                                                                                           | Effort | Impact | Customer Value  | Module  | Verification       |
| ----- | -------------------------------------------------------------------------------------------------------------- | ------ | ------ | --------------- | ------- | ------------------ |
| 2.4.1 | Extract table name constants in `storage/`: `tableEvents`, `tableOutbox`, `tableSnapshots`, `tableCheckpoints` | 8min   | Medium | Maintainability | storage | Build passes       |
| 2.4.2 | Replace magic table name strings in `storage/event_store.go` with constants                                    | 8min   | Medium | Maintainability | storage | Build + tests pass |
| 2.4.3 | Replace magic table name strings in `storage/outbox.go` + `sqlite_outbox.go` with constants                    | 8min   | Medium | Maintainability | storage | Build + tests pass |
| 2.4.4 | Replace magic table name strings in `storage/snapshot.go` + `sqlite_snapshot.go` with constants                | 8min   | Medium | Maintainability | storage | Build + tests pass |
| 2.4.5 | Replace magic table name strings in `storage/checkpoint.go` + `sqlite_checkpoint.go` with constants            | 8min   | Medium | Maintainability | storage | Build + tests pass |
| 2.4.6 | Deduplicate `catalog/schema.go` Ptr-unwrapping logic (1 clone group)                                           | 10min  | Medium | Maintainability | catalog | Build + tests pass |
| 2.4.7 | Deduplicate `pkg/id/id.go` validation blocks (1 clone group)                                                   | 10min  | Medium | Maintainability | core    | Build + tests pass |

### 2.5 Deduplication — Test Code

| #      | Task                                                                          | Effort | Impact | Customer Value  | Module  | Verification |
| ------ | ----------------------------------------------------------------------------- | ------ | ------ | --------------- | ------- | ------------ |
| 2.5.1  | Refactor `catalog/registry_test.go` to table-driven (4 clone groups)          | 12min  | Medium | Maintainability | catalog | Tests pass   |
| 2.5.2  | Refactor `catalog/benchmark_test.go` to table-driven (1 clone group)          | 10min  | Low    | Maintainability | catalog | Tests pass   |
| 2.5.3  | Refactor `catalog/asyncapi/exporter_test.go` to table-driven (5 clone groups) | 12min  | Medium | Maintainability | catalog | Tests pass   |
| 2.5.4  | Refactor `catalog/schema_test.go` to table-driven (4 clone groups)            | 12min  | Medium | Maintainability | catalog | Tests pass   |
| 2.5.5  | Refactor `event/memory_bus_test.go` to table-driven (4 clone groups)          | 12min  | Medium | Maintainability | memory  | Tests pass   |
| 2.5.6  | Refactor `event/event_test.go` to table-driven (1 clone group)                | 10min  | Low    | Maintainability | core    | Tests pass   |
| 2.5.7  | Refactor `event/types_test.go` to table-driven (2 clone groups)               | 12min  | Medium | Maintainability | core    | Tests pass   |
| 2.5.8  | Refactor `aggregate/aggregate_test.go` to table-driven (3 clone groups)       | 12min  | Medium | Maintainability | core    | Tests pass   |
| 2.5.9  | Refactor `command/command_test.go` to table-driven (2 clone groups)           | 10min  | Low    | Maintainability | core    | Tests pass   |
| 2.5.10 | Refactor `query/query_test.go` to table-driven (3 clone groups)               | 12min  | Medium | Maintainability | core    | Tests pass   |
| 2.5.11 | Refactor `pkg/id/id_test.go` to table-driven (2 clone groups)                 | 12min  | Medium | Maintainability | core    | Tests pass   |

---

## Tier 3: The 20% → 80% of Value (MEDIUM — Do After)

These 29 tasks deliver the remaining 16% of value. They add major features, improve CI, and enhance documentation.

### 3.1 TransactionalStore Implementation

| #     | Task                                                                                 | Effort | Impact | Customer Value       | Module      | Verification       |
| ----- | ------------------------------------------------------------------------------------ | ------ | ------ | -------------------- | ----------- | ------------------ |
| 3.1.1 | Review `docs/planning/OUTBOX_TRANSACTION_API.md` design doc                          | 10min  | High   | Feature completeness | core        | Design validated   |
| 3.1.2 | Define `TransactionalStore` interface in `core/event/transactional.go`               | 8min   | High   | Feature completeness | core        | Build passes       |
| 3.1.3 | Implement `TransactionalStore` for SQLite (`storage/sqlite_transactional_store.go`)  | 12min  | High   | Feature completeness | storage     | Build passes       |
| 3.1.4 | Implement `TransactionalStore` for PostgreSQL (`storage/transactional_store.go`)     | 12min  | High   | Feature completeness | storage     | Build passes       |
| 3.1.5 | Add `TransactionalStore` unit tests for SQLite backend                               | 12min  | High   | Feature completeness | storage     | Tests pass         |
| 3.1.6 | Add `TransactionalStore` unit tests for PostgreSQL backend                           | 12min  | High   | Feature completeness | storage     | Tests pass         |
| 3.1.7 | Update `core/aggregate/repository.go` to use `TransactionalStore` via type assertion | 10min  | High   | Feature completeness | core        | Build + tests pass |
| 3.1.8 | Update `core/decider/repository.go` to use `TransactionalStore` via type assertion   | 10min  | High   | Feature completeness | core        | Build + tests pass |
| 3.1.9 | Add integration tests for atomic save+outbox in `integration/event/`                 | 12min  | High   | Feature completeness | integration | Tests pass         |

### 3.2 Saga / Process Manager

| #     | Task                                                                | Effort | Impact | Customer Value       | Module | Verification    |
| ----- | ------------------------------------------------------------------- | ------ | ------ | -------------------- | ------ | --------------- |
| 3.2.1 | Review `docs/planning/SAGA_DESIGN.md` and finalize API design       | 10min  | Medium | Feature completeness | —      | Design approved |
| 3.2.2 | Create `saga/` module directory with `go.mod`                       | 5min   | Medium | Feature completeness | saga   | `go mod init`   |
| 3.2.3 | Define `saga.Core` struct with state machine fields                 | 10min  | Medium | Feature completeness | saga   | Build passes    |
| 3.2.4 | Define `saga.Step` type with action, compensation, and retry config | 10min  | Medium | Feature completeness | saga   | Build passes    |
| 3.2.5 | Implement `saga.Coordinator` with step-by-step execution            | 12min  | Medium | Feature completeness | saga   | Build passes    |
| 3.2.6 | Implement compensation logic (rollback on failure)                  | 12min  | Medium | Feature completeness | saga   | Build passes    |
| 3.2.7 | Add saga unit tests (happy path + compensation)                     | 12min  | Medium | Feature completeness | saga   | Tests pass      |
| 3.2.8 | Add saga BDD tests with Ginkgo                                      | 12min  | Medium | Feature completeness | saga   | Tests pass      |

### 3.3 CI/CD & Tooling

| #     | Task                                                                                   | Effort | Impact | Customer Value     | Module | Verification            |
| ----- | -------------------------------------------------------------------------------------- | ------ | ------ | ------------------ | ------ | ----------------------- |
| 3.3.1 | Add `GOWORK=off` CI verification job to `.github/workflows/ci.yml`                     | 10min  | High   | Build integrity    | root   | CI passes               |
| 3.3.2 | Parallelize CI matrix: one job per module (core, memory, storage, etc.)                | 12min  | Medium | Faster CI          | root   | All jobs pass           |
| 3.3.3 | Optimize `integration/` CI: skip on unrelated changes via path filters                 | 10min  | Medium | Faster CI          | root   | CI skips correctly      |
| 3.3.4 | Fix `gomodguard` deprecation warning in `.golangci.yml` → `gomodguard_v2`              | 8min   | Low    | Lint compliance    | root   | `nix run .#lint` passes |
| 3.3.5 | Fix `MemoryBus.Publish` RLock during handler execution (documented low-severity issue) | 10min  | Low    | Concurrency safety | memory | Tests pass, no deadlock |

### 3.4 Documentation

| #     | Task                                                                                   | Effort | Impact | Customer Value       | Module  | Verification |
| ----- | -------------------------------------------------------------------------------------- | ------ | ------ | -------------------- | ------- | ------------ |
| 3.4.1 | Create `CONTRIBUTING.md` with architecture guidelines for contributors                 | 12min  | Medium | Community            | root    | Reviewed     |
| 3.4.2 | Add `core/README.md` explaining public API (event, command, query, aggregate, decider) | 12min  | Medium | Developer experience | core    | Accurate     |
| 3.4.3 | Add `storage/README.md` explaining store implementations (SQLite, PostgreSQL, Pebble)  | 10min  | Medium | Developer experience | storage | Accurate     |
| 3.4.4 | Add `catalog/README.md` explaining doc generation (AsyncAPI, EventCatalog, D2)         | 10min  | Medium | Developer experience | catalog | Accurate     |
| 3.4.5 | Update `FEATURES.md` with `TransactionalStore` status (📐 PLANNED → implementation)    | 5min   | Low    | Documentation        | root    | Accurate     |
| 3.4.6 | Update `CHANGELOG.md` with all changes from this session                               | 8min   | Low    | Documentation        | root    | Complete     |

### 3.5 Example & Misc

| #     | Task                                                                            | Effort | Impact | Customer Value  | Module  | Verification       |
| ----- | ------------------------------------------------------------------------------- | ------ | ------ | --------------- | ------- | ------------------ |
| 3.5.1 | Split `example/todo/cmd/api/main.go` (329 lines): extract routes to `routes.go` | 12min  | Low    | Demo quality    | example | Build passes       |
| 3.5.2 | Split `example/todo/cmd/api/main.go`: extract handlers to `handlers.go`         | 12min  | Low    | Demo quality    | example | Build passes       |
| 3.5.3 | Evaluate `io.Closer` on all interfaces (Bus, Store, SnapshotStore, Outbox)      | 10min  | Low    | API consistency | core    | Document decision  |
| 3.5.4 | Add error path coverage tests for `storage/` (target: 93% from 85.1%)           | 12min  | Medium | Test coverage   | storage | Coverage ≥93%      |
| 3.5.5 | Tag `v0.1.0-alpha` across all modules with shared versioning                    | 10min  | High   | Release         | root    | All modules tagged |

---

## Execution Order

```
Phase 1 (Tasks 1.1.1–1.5.4):  24 tasks, ~180min → Module hygiene + type safety
Phase 2 (Tasks 2.1.1–2.5.11): 33 tasks, ~330min → Tests + deduplication + docs
Phase 3 (Tasks 3.1.1–3.5.5):  29 tasks, ~290min → Features + CI + documentation
─────────────────────────────────────────────────────────────────────────
TOTAL:                         86 tasks, ~800min (~13.3 hours)
```

### Daily Capacity Assumptions

- **Focused execution:** 6–8 tasks/day (2 hours of focused work)
- **Total calendar time:** ~11–14 working days

---

## Dependency Graph (D2)

```d2
direction: down

ModuleHygiene: {
  label: |md
    **Module Hygiene**
    - Remove replace directives
    - Standardize versions
    - GOWORK=off verification
  |
  style.fill: "#ffcccc"
}

FileSize: {
  label: |md
    **File Size Compliance**
    - Split pebble_event_store.go
    - Split catalog/registry.go
  |
  style.fill: "#ffcccc"
}

TypeSafety: {
  label: |md
    **Type Safety Fixes**
    - Version IncompatibleAssign
    - All test files fixed
  |
  style.fill: "#ffcccc"
}

DeadCode: {
  label: |md
    **Dead Code Removal**
    - cattest evaluation
    - Coverage exclusion
  |
  style.fill: "#ffcccc"
}

AGENTS: {
  label: |md
    **AGENTS.md Updates**
    - Replace directive guidance
    - Document fixes
  |
  style.fill: "#ccffcc"
}

BDDTests: {
  label: |md
    **BDD Tests**
    - Version, SchemaVersion
    - OutboxStatus, NodeID
    - SyncMessageType, Pagination
  |
  style.fill: "#ccffcc"
}

CatalogMeta: {
  label: |md
    **CatalogMeta**
    - Extract Base type
    - Embed in 3 packages
  |
  style.fill: "#ccffcc"
}

Dedup: {
  label: |md
    **Deduplication**
    - Table constants
    - Schema Ptr-unwrapping
    - ID validation blocks
    - Test table-driven refactors
  |
  style.fill: "#ccffcc"
}

TransactionalStore: {
  label: |md
    **TransactionalStore**
    - Interface design
    - SQLite + PG impl
    - Repository integration
  |
  style.fill: "#ccccff"
}

Saga: {
  label: |md
    **Saga/Process Manager**
    - Core + Step types
    - Coordinator
    - Compensation
  |
  style.fill: "#ccccff"
}

CI: {
  label: |md
    **CI/CD**
    - GOWORK=off job
    - Per-module parallel
    - Path filters
  |
  style.fill: "#ccccff"
}

Docs: {
  label: |md
    **Documentation**
    - CONTRIBUTING.md
    - Module READMEs
    - CHANGELOG.md
  |
  style.fill: "#ccccff"
}

ModuleHygiene -> FileSize -> TypeSafety -> DeadCode -> AGENTS -> BDDTests -> CatalogMeta -> Dedup -> TransactionalStore -> Saga -> CI -> Docs
```

---

## Quality Gates Per Phase

### Phase 1 Gate

- [ ] All 6 modules build with `GOWORK=off`
- [ ] `nix run .#test` passes (all 20 packages)
- [ ] `nix run .#lint` passes (zero issues)
- [ ] No production files >250 lines
- [ ] All `IncompatibleAssign` issues resolved

### Phase 2 Gate

- [ ] All BDD test suites pass
- [ ] CatalogMeta consolidation: 3 structs → 1 base + embeddings
- [ ] All deduplication table-driven refactors pass
- [ ] AGENTS.md reflects current state
- [ ] `cattest` removed or tested

### Phase 3 Gate

- [ ] TransactionalStore integration tests pass
- [ ] Saga BDD tests pass
- [ ] CI parallel matrix passes
- [ ] CONTRIBUTING.md + module READMEs written
- [ ] `v0.1.0-alpha` tag applied
- [ ] Coverage ≥93% across all modules

---

_Plan generated 2026-05-18. Pareto-sorted by impact/effort/customer-value. Each task ≤12 minutes._
