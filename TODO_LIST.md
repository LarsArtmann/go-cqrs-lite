# TODO List — go-cqrs-lite

**Last Updated:** 2026-04-30  
**Current Branch:** master  
**Build Status:** ✅ All tests passing, zero lint issues, circular dependency fixed

---

## Philosophy

> "Perfect is the enemy of shipped — but sloppy is the enemy of trust."

This list is ruthlessly prioritized by **impact/effort ratio**. Each task includes:

- **Effort** estimate (in minutes or hours)
- **Impact** on the codebase
- **Customer value** (why it matters)
- **Honest blockers** (what might stop us)

---

## Tier 1: Must Do (P0 — Blocks Production Use)

### 1.1 Fix core→memory/testhelpers Circular Dependency

- **Status:** ✅ DONE
- **Effort:** ~2 hours
- **Impact:** CRITICAL — core module now independently publishable
- **Customer value:** Users can `go get github.com/larsartmann/go-cqrs-lite/core` without replace directives
- **Approach:** Created `integration/` module at repo root. Moved 15 cross-module test files from `core/` into it.
- **Files:** `integration/go.mod`, `integration/{aggregate,command,event,query}/`, `core/go.mod`
- **Verification:** `cd core && GOWORK=off go test ./...` passes without any replace directives

---

## Tier 2: Should Do (P1 — High Value, Low Risk)

### 2.1 Add EventRetry Middleware Tests

- **Status:** ✅ DONE (commit `b8c0aa9`)
- **Delivered:** Split retry_test.go into 3 files, added EventRetry non-retryable + context cancellation tests

### 2.2 Add OpenTelemetry Tracing Middleware

- **Status:** ✅ DONE (commits `1e70bfc`, `c7771cb`)
- **Delivered:** CommandTracing, EventTracing, QueryTracing with injectable tracer, span kinds

### 2.3 Add slog Structured Logging Adapter

- **Status:** ✅ DONE (commit `e6bde18`)
- **Delivered:** `middleware.SlogAdapter` wraps `*slog.Logger` for the `Logger` interface

### 2.4 Remove Empty `core/internal/` Directory

- **Status:** ✅ DONE (directory removed during session)

### 2.5 Update Stale Planning Docs (xtypes References)

- **Status:** ✅ DONE (commit `fd51837`)

### 2.6 Fix `go mod tidy` Across All Modules

- **Status:** ✅ DONE (commit `a819c86`)
- **Delivered:** Added replace directives for memory/testhelpers to catalog, middleware, testhelpers go.mod files

---

## Tier 3: Could Do (P2 — Future Architecture)

### 3.1 Design Doc: SQL Event Store Module

- **Status:** Not started
- **Effort:** ~1 hour
- **Impact:** HIGH — next major module
- **Customer value:** Persistent event storage (current memory store loses data on restart)
- **Approach:** Research sqlc + PostgreSQL patterns. Design `storage/` module with `SQLStore`.
- **Files:** `docs/planning/2026-04-29_SQL_STORE_DESIGN.md`
- **Blocker:** Need to evaluate sqlc vs raw SQL vs gorm

### 3.2 Design Doc: Saga / Process Manager

- **Status:** Not started
- **Effort:** ~1 hour
- **Impact:** HIGH — enables complex multi-aggregate workflows
- **Customer value:** Orchestrate long-running business processes across aggregates
- **Approach:** Research saga patterns (choreography vs orchestration). Design `saga/` module.
- **Files:** `docs/planning/2026-04-29_SAGA_DESIGN.md`
- **Blocker:** Need to understand if this library's scope should include sagas

### 3.3 Design Doc: Event Upcasting

- **Status:** Not started
- **Effort:** ~45 minutes
- **Impact:** MEDIUM — event schema evolution
- **Customer value:** Migrate old event payloads to new schema versions without data loss
- **Approach:** Design `event.Upcaster` interface and registry
- **Files:** `docs/planning/2026-04-29_UPCASTING_DESIGN.md`
- **Blocker:** None

### 3.4 Outbox Background Publisher

- **Status:** Partially done — interface exists, no publisher implementation
- **Effort:** ~1 hour
- **Impact:** MEDIUM — completes the outbox pattern
- **Customer value:** Automatic reliable event publishing from outbox
- **Approach:** Design `OutboxPublisher` that polls outbox and publishes to bus
- **Blocker:** None

---

## Tier 4: Explicitly Skipped (with rationale)

### 4.1 Generic Middleware Core (Task 2.1)

- **Skipped because:** Go's defined handler types (`command.Handler`, `event.Handler`) prevent clean generic unification. Adapter boilerplate equals original code length.
- **Verdict:** Complexity > value.

### 4.2 Query Generic Result Types (Task 3.1)

- **Skipped because:** `DispatchTyped[T]` already works. The runtime type assertion is one line. Making `Query` generic would break all handlers and require type erasure at the registry anyway.
- **Verdict:** Marginal benefit for massive breakage.

### 4.3 CatalogBuilder Wraps Registry (Task 2.4)

- **Skipped because:** Behavioral divergence. `Registry.AddService` merges messages; `CatalogBuilder.AddService` overwrites metadata. Unification requires breaking changes or adding adapter methods that increase API surface.
- **Verdict:** Two builders with different semantics is better than one with surprising behavior.

---

## Quality Gates (check before declaring "done")

- [x] All tests pass (`nix run .#test`)
- [x] No lint issues (`nix run .#lint`)
- [x] Build compiles (`nix run .#build`)
- [x] Format clean (`nix fmt` produces no changes)
- [x] Coverage maintained or improved
- [x] AGENTS.md updated with new patterns
- [x] Commit history is clean and descriptive

---

## Execution Strategy

1. **Tier 1 is COMPLETE** — Circular dependency broken via `integration/` module.
2. **Tier 2 is COMPLETE** — All P1 tasks finished.
3. **Tier 3 deferred** — Design docs for SQL store, saga, upcasting, outbox publisher.
4. **Tier 4 remains closed** — Skip decisions stand.

---

## Done (this session)

| Task                                   | Commit    | Date       |
| -------------------------------------- | --------- | ---------- |
| Fix go mod tidy in all modules         | `a819c86` | 2026-04-29 |
| Make tracer injectable (refactor OTel) | `c7771cb` | 2026-04-29 |
| Add slog adapter                       | `e6bde18` | 2026-04-29 |
| Document circular dependency           | `62849df` | 2026-04-29 |
| Remove stale xtypes refs               | `fd51837` | 2026-04-29 |
| OTel tracing middleware                | `1e70bfc` | 2026-04-29 |
| Retry test split + missing coverage    | `b8c0aa9` | 2026-04-29 |
| Outbox seam                            | `2c1de1f` | 2026-04-29 |
| Snapshot integration tests             | `b6aaa4a` | 2026-04-29 |
| Deep copy fix (snapshot)               | `ae0b088` | 2026-04-29 |
| Type-safe validators                   | `d3b27c3` | 2026-04-29 |
| EventBuilder migration                 | `a6755ab` | 2026-04-29 |
| Typed interface (dispatcher)           | `f3532ad` | 2026-04-29 |
| Remove internal/testhelpers shim       | `63b39a5` | 2026-04-29 |
| Delete xtypes module                   | `51b1d95` | 2026-04-29 |
