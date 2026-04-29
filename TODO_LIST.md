# TODO List — go-cqrs-lite

**Last Updated:** 2026-04-29 23:07 UTC  
**Current Branch:** master (13 commits ahead of origin)  
**Build Status:** ✅ All tests passing, zero lint issues

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
- **Status:** Not started
- **Effort:** ~2 hours
- **Impact:** CRITICAL — blocks publishing core module independently
- **Customer value:** Users can't `go get github.com/larsartmann/go-cqrs-lite/core` without replace directives
- **Approach:** Create `integration/` module at repo root. Move memory-dependent core tests into it.
- **Files:** `core/go.mod`, all `core/*/*_test.go` that import `memory` or `testhelpers`
- **Blocker:** Need to decide — is independent publishability required, or are coordinated releases acceptable?

---

## Tier 2: Should Do (P1 — High Value, Low Risk)

### 2.1 Add EventRetry Middleware Tests
- **Status:** Not started
- **Effort:** ~20 minutes
- **Impact:** MEDIUM — closes last coverage gap in retry logic
- **Customer value:** Confidence that event retry behaves identically to command retry
- **Approach:** Extract shared retry test harness from `middleware/retry_test.go`, add EventRetry test cases
- **Files:** `middleware/retry_test.go` (extend), or `middleware/retry_event_test.go` (new)
- **Blocker:** None

### 2.2 Add OpenTelemetry Tracing Middleware
- **Status:** Not started
- **Effort:** ~1 hour
- **Impact:** MEDIUM — production observability
- **Customer value:** Distributed tracing across command dispatch, event publish, query handling
- **Approach:** Add `middleware/tracing.go` with `CommandTracing`, `EventTracing`, `QueryTracing`
- **Files:** `middleware/tracing.go`, `middleware/tracing_test.go`
- **Blocker:** None (Go OTel SDK is stable)

### 2.3 Remove Empty `core/internal/` Directory
- **Status:** Not started
- **Effort:** ~1 minute
- **Impact:** LOW — cosmetic cleanup
- **Customer value:** Cleaner repo structure
- **Approach:** `rmdir core/internal/`
- **Blocker:** None

### 2.4 Update Stale Planning Docs (xtypes References)
- **Status:** Not started
- **Effort:** ~15 minutes
- **Impact:** LOW — prevents contributor confusion
- **Customer value:** Accurate documentation
- **Files:** `docs/planning/go-composable-business-types-usage.md`, `CHANGELOG.md`
- **Blocker:** None

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

- [ ] All tests pass (`nix run .#test`)
- [ ] No lint issues (`nix run .#lint`)
- [ ] Build compiles (`nix run .#build`)
- [ ] Format clean (`nix fmt` produces no changes)
- [ ] Coverage maintained or improved
- [ ] AGENTS.md updated with new patterns
- [ ] Commit history is clean and descriptive

---

## Execution Strategy

1. **Start with Tier 1** — The circular dependency is the only thing blocking real-world adoption.
2. **Batch Tier 2** — EventRetry tests + OTel tracing + cleanup can be done in a single focused session.
3. **Defer Tier 3** until after Tier 1 and Tier 2 are complete. Design docs are valuable but not urgent.
4. **Never revisit Tier 4** without new evidence. The skip decisions were made with thorough analysis.

---

## Done (for reference)

See `docs/status/archive/` for complete history.

| Task | Commit | Date |
|------|--------|------|
| Type-safe validators | `d3b27c3` | 2026-04-29 |
| EventBuilder migration | `a6755ab` | 2026-04-29 |
| Typed interface (dispatcher) | `f3532ad` | 2026-04-29 |
| Remove internal/testhelpers shim | `63b39a5` | 2026-04-29 |
| Delete xtypes module | `51b1d95` | 2026-04-29 |
| Snapshot integration tests | `b6aaa4a` | 2026-04-29 |
| Deep copy fix (snapshot) | `ae0b088` | 2026-04-29 |
| Outbox seam | `2c1de1f` | 2026-04-29 |
