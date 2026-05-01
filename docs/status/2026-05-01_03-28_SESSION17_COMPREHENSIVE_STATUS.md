# Session 17 — Comprehensive Status Report

**Date:** 2026-05-01 03:28  
**Branch:** master  
**Working tree:** Clean (all changes committed through session 16)  
**Total Go files:** 136 | **Total lines:** 21,645 | **Total commits:** 399

---

## Session 17 Work

**Task:** Enhance `example/user/` to generate EventCatalog output.

**Result:** The example was already enhanced in session 16 (commit `4324713`). Session 16 added:
- Typed JSON payloads (`UserCreatedPayload`, `UserNameChangedPayload`) with `json`/`description` struct tags
- Catalogable event types (`userCreatedEvent`, `userNameChangedEvent`) embedding `*event.CatalogCore`
- `generateEventCatalog()` function using `catalog/adapters.CatalogBuilder`
- Domain grouping ("Identity" domain)
- Catalog summary printing
- Full EventCatalog output to `eventcatalog-output/` (MDX files, JSON schemas, config, llms.txt)

**What this session actually did:**
1. Re-verified the example builds and runs correctly
2. Re-verified EventCatalog output is correct (MDX frontmatter, JSON schemas, config, llms.txt, domains)
3. Re-verified all 18 test packages pass
4. Cleaned up stale `example/catalog/` binary directory (was already empty of source code)
5. Added `**/eventcatalog-output/` to `.gitignore` (was already added)
6. Updated AGENTS.md with example changes (was already updated)

**Conclusion:** All work was already done. Working tree remains clean.

---

## A) FULLY DONE ✅

### Core Library (Production-Grade)

| Module | Coverage | Status |
|--------|----------|--------|
| `core/command` | 100.0% | Complete — dispatch, middleware, catalog |
| `core/query` | 100.0% | Complete — dispatch, pagination, typed results |
| `core/event` | 86.7% | Complete — store/bus/snapshot interfaces, projections, upcasters, codec |
| `core/aggregate` | 95.6% | Complete — roots, repository, snapshot strategy, codec |
| `core/pkg/dispatcher` | 100.0% | Complete — generic dispatcher with lifecycle |
| `core/pkg/id` | 92.9% | Complete — branded IDs backed by go-branded-id + ULID |
| `memory` | 94.9% | Complete — MemoryStore, MemoryBus, MemorySnapshotStore |
| `middleware` | 99.4% | Complete — logging, metrics, retry, recovery, validation |
| `catalog` | 94.4% | Complete — registry, schema reflection, MessageID |
| `catalog/adapters` | 98.8% | Complete — builder, dispatcher adapters |
| `catalog/asyncapi` | 97.9% | Complete — AsyncAPI 3.0 YAML/JSON export |
| `catalog/eventcatalog` | 95.5% | Complete — EventCatalog MDX generation |
| `testhelpers` | — | Complete — shared fakes, test doubles |
| `integration` | — | Complete — cross-module BDD + integration tests |

### All Tests Green (18 packages)

```
ok  core/aggregate       95.6%
ok  core/command        100.0%
ok  core/event           86.7%
ok  core/pkg/dispatcher 100.0%
ok  core/pkg/id          92.9%
ok  core/query          100.0%
ok  memory               94.9%
ok  catalog              94.4%
ok  catalog/adapters     98.8%
ok  catalog/asyncapi     97.9%
ok  catalog/eventcatalog 95.5%
ok  middleware           99.4%
ok  integration/aggregate/command/event/query — all pass
```

### Infrastructure

- ✅ **Storage module** (`storage/`) — `SQLEventStore` with PostgreSQL, optimistic concurrency (no tests yet)
- ✅ **Example app** (`example/user/`) — full CQRS + Event Sourcing + EventCatalog generation
- ✅ **Nix build system** — `flake.nix` with build/test/lint/fmt/vet/coverage/check
- ✅ **GitHub Actions CI** — single `ci.yml`, Nix-based
- ✅ **Zero lint** across all 6 linted modules
- ✅ **Zero race conditions** in all passing modules
- ✅ **Zero broken tests** — all fuzz tests, golden tests, unit tests pass

### Architecture Seams (Session 14)

- ✅ **SnapshotStrategy** — `EveryNEvents` convenience, wired into `EventSourcedRepository`
- ✅ **Codec** — wired into `EventSourcedRepository` for snapshot serialization
- ✅ **JSONCodec** — default JSON implementation using `go-json-experiment/json`
- ✅ **DecodePayload[T]** — type-safe event payload deserialization
- ✅ **ContextEnricher** + **CompositeEnricher** — metadata extraction from context
- ✅ **Projection system** — `Projection` interface, `InMemoryRunner`, checkpoint tracking
- ✅ **CheckpointStore** — interface + `MemoryCheckpointStore`
- ✅ **Upcaster system** — `Upcaster`, `UpcasterFunc`, `UpcasterRegistry` with sorted chains

### Sessions 15–16 Fixes

- ✅ **go-branded-id delegation** — all 8 serialization methods forward to upstream
- ✅ **Ptr/FromPtr/fmt.Formatter** — forwarded from go-branded-id
- ✅ **FuzzParse fix** — case-insensitive ULID comparison in fuzz test
- ✅ **time.Time schema fix** — returns `{type:"string"}` instead of `{type:"string", Properties:{}}`
- ✅ **AsyncAPI key collision** — prefixed with message kind (command.X vs event.X)
- ✅ **toDotAddress** — fixed acronym and number handling
- ✅ **crypto/rand for retry jitter** — replaced `math/rand/v2`
- ✅ **gofumpt formatting** — project-wide
- ✅ **Golden files regenerated** — asyncapi + eventcatalog
- ✅ **Projection test split** — moved memory-dependent tests to integration/
- ✅ **InMemoryRunner nil guards** — nil projection registration returns sentinel error
- ✅ **InMemoryRunner duplicate detection** — duplicate registration returns sentinel error

---

## B) PARTIALLY DONE ⚠️

### Storage Module — Code Without Tests

- ✅ `SQLEventStore` implements `event.Store` with optimistic concurrency (346 lines, 10 functions)
- ✅ Transactional `AppendBatch` implementation
- ✅ Branded IDs via `driver.Valuer` for SQL params
- ✅ Metadata persistence and restoration
- ⚠️ **Zero tests** — no unit tests, no integration tests
- ⚠️ No testcontainers setup
- ⚠️ Coverage: N/A (no test files)

### Event Coverage Drop

- `core/event` dropped from 97.9% → 86.7% after projection test split (moved tests to `integration/event/`)
- The split was correct (removes memory dependency from core), but core/event lost coverage on projection runner paths
- Integration tests cover these paths but don't contribute to core/event's coverage number

### Projection Runner — In-Memory Only

- ✅ `Projection` interface, `InMemoryRunner`, `CheckpointStore` all implemented
- ⚠️ No persistent runner (SQL-backed checkpoints)
- ⚠️ No partitioning or parallel processing

---

## C) NOT STARTED 📋

| # | Item | Priority | Effort |
|---|------|----------|--------|
| 1 | Storage module tests (PostgreSQL integration) | HIGH | 1 day |
| 2 | Watermill module (`watermill/`) — pub/sub integration | HIGH | 2-3 days |
| 3 | Tag v0.3.0 release | HIGH | 1 hour |
| 4 | SQL Snapshot Store — persistent snapshots | MEDIUM | 1 day |
| 5 | Persistent projection runner (SQL checkpoints) | MEDIUM | 1 day |
| 6 | Circuit breaker middleware | MEDIUM | 4 hrs |
| 7 | Dead letter queue mechanism | MEDIUM | 4 hrs |
| 8 | Event bus partitioning by aggregate ID | MEDIUM | 4 hrs |
| 9 | HTTP handler examples (chi/echo) | LOW | 2 hrs |
| 10 | API documentation (godoc polish) | LOW | 1 day |
| 11 | Performance benchmarks for storage | LOW | 0.5 day |
| 12 | Migration CLI tool (schema versioning) | LOW | 2 days |
| 13 | Graceful shutdown patterns | LOW | 0.5 day |
| 14 | OpenTelemetry integration (tracing middleware) | LOW | 1 day |
| 15 | Documentation site (Docusaurus) | LOW | 2 days |

---

## D) TOTALLY FUCKED UP 💥

### Nothing is currently broken.

All 18 test packages pass. All golden files are up to date. Fuzz tests pass. Zero lint. Zero races.

### Annoying but Not Broken: LSP Noise (31 diagnostics)

All 31 LSP errors are false positives from `gopls` not understanding `go.work` workspace correctly. Pattern: "X is not in your go.mod file" for transitive workspace dependencies. These do not affect builds or tests.

### Technical Debt (Not Broken, But Risk)

1. **`storage/event_store.go` is 346 lines** — exceeds the 250-line file size limit. Should be split.
2. **`storage/` has zero tests** — 346 lines of untested PostgreSQL code. A time bomb.
3. **`example/user/` has no test file** — the example compiles and runs but isn't tested programmatically.
4. **31 LSP false positives** — noisy but not fixable without gopls changes.
5. **`core/event` coverage at 86.7%** — dropped after projection test split. Should add focused unit tests to bring it back above 95%.

---

## E) WHAT WE SHOULD IMPROVE 🔧

### Critical

1. **Storage module tests** — This is the #1 risk. 346 lines of PostgreSQL code with zero tests. Any bug in the SQL queries, optimistic concurrency logic, or metadata handling is undetectable.
2. **Event coverage recovery** — `core/event` dropped to 86.7%. Add focused unit tests for `InMemoryRunner` and `UpcasterRegistry` that don't need `memory` module.
3. **File size limit violation** — `storage/event_store.go` at 346 lines exceeds the 250-line convention.

### Process

4. **No example test** — `example/user/main.go` is a `package main` with no test file. Should have at least a smoke test that verifies the catalog output is generated correctly.
5. **Status report consolidation** — 17 status reports in 4 days. Consider switching to weekly summaries.
6. **Go workspace + gopls** — 31 false positive diagnostics. Consider adding a `.golangci.yml` or gopls setting to suppress workspace-related noise.

### Architecture

7. **No circuit breaker** — Retry middleware retries blindly. Should integrate with a circuit breaker for production use.
8. **Event bus has no ordering guarantees** — `MemoryBus` delivers events as they come. No partitioning by aggregate ID.
9. **No dead letter queue** — Failed events are just logged. Need a DLQ mechanism.
10. **Projection runner is in-memory only** — Need persistent runner (SQL-backed checkpoints).
11. **`query.Handler` returns `(any, error)`** — Loses type safety at the interface level. `DispatchTyped[T]` exists but the underlying interface is untyped.
12. **No structured logging interface** — Middleware logging uses `slog` directly. Should accept a logger interface.
13. **Storage depends on pgx directly** — Should abstract SQL driver for database portability.

---

## F) Top 25 Things We Should Get Done Next

### Immediate — Fix Coverage Gap (2-3 hours)

| # | Task | Effort |
|---|------|--------|
| 1 | Add `InMemoryRunner` unit tests to `core/event/` (without memory dependency) | 1 hr |
| 2 | Add `UpcasterRegistry` unit tests to `core/event/` | 30 min |
| 3 | Split `storage/event_store.go` under 250 lines | 30 min |
| 4 | Verify `core/event` coverage recovers to 95%+ | 15 min |

### Short-Term — Ship Storage Tests (1-2 days)

| # | Task | Effort |
|---|------|--------|
| 5 | Create `storage/event_store_test.go` with pgx mock or testcontainers | 4 hrs |
| 6 | Test optimistic concurrency conflicts | 1 hr |
| 7 | Test metadata persistence roundtrip | 1 hr |
| 8 | Test branded ID SQL parameter binding | 30 min |
| 9 | Test `AppendBatch` transactionality | 1 hr |
| 10 | Run storage tests in CI (PostgreSQL service container) | 2 hrs |

### Medium-Term — Ship v0.3.0 (1 week)

| # | Task | Effort |
|---|------|--------|
| 11 | Polish godoc on all exported types | 2 hrs |
| 12 | Add `example/user/main_test.go` smoke test | 1 hr |
| 13 | Update CHANGELOG for v0.3.0 | 30 min |
| 14 | Tag v0.3.0 release | 15 min |
| 15 | Watermill module — pub/sub with Kafka/NATS | 2-3 days |
| 16 | SQL Snapshot Store implementation | 1 day |
| 17 | Persistent projection runner (SQL checkpoints) | 1 day |

### Long-Term — Production Ecosystem (2-4 weeks)

| # | Task | Effort |
|---|------|--------|
| 18 | Circuit breaker middleware | 4 hrs |
| 19 | Dead letter queue mechanism | 4 hrs |
| 20 | Event bus partitioning by aggregate ID | 4 hrs |
| 21 | HTTP handler examples (chi/echo integration) | 2 hrs |
| 22 | OpenTelemetry tracing middleware | 1 day |
| 23 | Migration CLI tool (schema versioning) | 2 days |
| 24 | Documentation site (Docusaurus) | 2 days |
| 25 | Example microservice (multi-service demo with EventCatalog) | 1 day |

---

## G) Top #1 Question I Cannot Answer Myself

**What is the release strategy for v0.3.0?**

The library has 8 modules, 136 Go files, 21K+ lines, and all tests green. But:

- **Storage module has zero tests.** Should v0.3.0 ship without storage tests, or block on them?
- **Should we include the `storage/` module in v0.3.0 at all?** It's the only module with pgx dependency. Removing it would make the v0.3.0 tag purely about the core library.
- **Watermill module is not started.** Is pub/sub a v0.3.0 blocker, or can it ship in v0.4.0?
- **Module versioning:** All modules use `v0.0.0` with replace directives. Do we version them independently (semver per module) or as a monorepo release?

This determines whether items 5-17 are "this sprint" or "next sprint."

---

## Module Coverage Matrix (Current)

| Module | Coverage | Tests | Lint | Race |
|--------|----------|-------|------|------|
| `core/command` | 100.0% | ✅ | ✅ | ✅ |
| `core/query` | 100.0% | ✅ | ✅ | ✅ |
| `core/event` | 86.7% | ✅ | ✅ | ✅ |
| `core/aggregate` | 95.6% | ✅ | ✅ | ✅ |
| `core/pkg/dispatcher` | 100.0% | ✅ | ✅ | ✅ |
| `core/pkg/id` | 92.9% | ✅ | ✅ | ✅ |
| `memory` | 94.9% | ✅ | ✅ | ✅ |
| `catalog` | 94.4% | ✅ | ✅ | ✅ |
| `catalog/adapters` | 98.8% | ✅ | ✅ | ✅ |
| `catalog/asyncapi` | 97.9% | ✅ | ✅ | ✅ |
| `catalog/eventcatalog` | 95.5% | ✅ | ✅ | ✅ |
| `middleware` | 99.4% | ✅ | ✅ | ✅ |
| `integration` | — | ✅ | ✅ | ✅ |
| `storage` | — | ⚠️ no tests | — | — |
| `testhelpers` | — | — | ✅ | — |

## Module Dependency Graph

```
testhelpers → core
memory      → core + testhelpers
middleware  → core + testhelpers
catalog     → core (via cattest internal helpers)
integration → core + memory + testhelpers
storage     → core (pgx dependency)
example/user → core + memory + catalog
core        → (no internal deps — independently publishable)
```

## Monorepo Module Count: 8

```
core/           — Core library (independently publishable)
memory/         — In-memory test implementations
catalog/        — Registry, AsyncAPI, EventCatalog exporters
middleware/     — Cross-cutting CQRS middleware
testhelpers/    — Shared test doubles
integration/    — Cross-module BDD + integration tests
storage/        — PostgreSQL SQLEventStore (untested)
example/user/   — Demo app (CQRS + EventCatalog generation)
```
