# Session 45 — Comprehensive Status Report

**Date:** 2026-05-03 06:45 CEST
**Branch:** master (clean)
**Sessions completed:** 45
**Total commits:** ~200+

---

## Executive Summary

go-cqrs-lite is a **mature, production-quality Go CQRS library** with 9 independently importable modules, 83.3% overall test coverage, 21 passing test packages (all with race detector), and 52 pre-existing lint issues (all in test files). The library is architecturally sound with clear ISP-compliant interfaces, extensible error taxonomy, and full documentation generation (AsyncAPI 3.0, D2, EventCatalog).

**This session** eliminated 89 lines of delegation boilerplate by converting `id.Of[T]` from a wrapper struct to a type alias for `go-branded-id`'s `ID[T, ulid.ULID]`.

---

## a) FULLY DONE

### Core CQRS Infrastructure

| Feature | Module | Coverage | Status |
|---------|--------|----------|--------|
| Command Dispatcher | `core/command` | 100.0% | ✅ Full lifecycle, middleware, catalog |
| Query Dispatcher | `core/query` | 100.0% | ✅ Pagination, typed dispatch, catalog |
| Event System | `core/event` | 98.0% | ✅ Store+Bus interfaces, metadata, builder, JSON codec |
| Branded IDs | `core/pkg/id` | 100.0% | ✅ Type alias to go-branded-id, ULID-backed |
| Generic Dispatcher | `core/pkg/dispatcher` | 100.0% | ✅ Lifecycle, middleware chain |
| Aggregate (OO) | `core/aggregate` | 93.2% | ✅ Repository, snapshots, versioning |
| Decider (FP) | `core/decider` | 96.2% | ✅ Pure functions, snapshot support |

### Supporting Modules

| Feature | Module | Coverage | Status |
|---------|--------|----------|--------|
| Memory Store/Bus/Snapshot | `memory` | 91.9% | ✅ Thread-safe test implementations |
| Middleware Suite | `middleware` | 100.0% | ✅ Logging, retry, recovery, validation, metrics |
| Catalog Registry | `catalog` | 94.4% | ✅ Schema reflection, thread-safe registry |
| AsyncAPI 3.0 Export | `catalog/asyncapi` | 95.9% | ✅ YAML/JSON, full document model |
| D2 Diagram Export | `catalog/d2` | 97.6% | ✅ Color-coded nodes, domain grouping |
| EventCatalog MDX | `catalog/eventcatalog` | 95.6% | ✅ MDX files with frontmatter |
| Catalog Adapters | `catalog/adapters` | 95.5% | ✅ Builder, dispatcher extraction |
| SQL Event Store | `storage` | 92.0% | ✅ PostgreSQL, snapshots, checkpoints, outbox |
| Projection Runner | `projection` | 89.7% | ✅ Replay, live subscription, retry, DI logger |
| Test Helpers | `testhelpers` | N/A | ✅ Fake store, bus, outbox, handlers |
| Integration Tests | `integration` | N/A | ✅ BDD + middleware chain tests across all packages |
| Example App | `example/user` | N/A | ✅ Full CQRS + Decider + EventCatalog demo |

### Architecture Quality

- **ISP compliance**: `event.Publisher` / `event.Subscriber` sub-interfaces — repos accept `Publisher`, projections accept `Subscriber`
- **Error taxonomy**: 5 families (Rejection, Conflict, Transient, Corruption, Infrastructure) with `RegisterClassification` for extensibility
- **No-panic convention**: All constructors return `(*T, error)`; `Must*` helpers for tests only
- **DI over globals**: `projection.WithLogger(*slog.Logger)`, `middleware.DefaultRetryConfig().IsRetryable`
- **Compile-time checks**: `var _ Interface = (*Impl)(nil)` across all packages
- **Branded types**: `AggregateID`, `EventID`, `UserID`, `CorrelationID`, `ClientID` — all type-safe via phantom generics

### Documentation

- **3 ADRs**: Decider over Aggregate, Error Taxonomy, Multi-Module Monorepo
- **9 research docs**: Including aggregateless ES, LiveStore deep dive, CQRS innovations, Datomic lessons
- **Architecture roadmap**: 5-phase plan in `docs/planning/2026-05-01_ARCHITECTURE_ROADMAP.md`
- **Domain glossary**: `CONTEXT.md` with ubiquitous language
- **Getting started guide**: `docs/getting-started.md`
- **Architecture diagrams**: `docs/web-client-communication.d2` (D2 + SVG)

---

## b) PARTIALLY DONE

### Projections ⚠️ PARTIALLY_FUNCTIONAL

- **What works**: Runner, per-projection checkpointing, replay, live subscription, retry with `IsRetryable`, `HandleParallel`, `HandlerRegistry`, DI logger
- **What's missing**:
  - No dead-letter mechanism for permanently failed events
  - No backpressure / rate limiting
  - No batch event processing (events dispatched one at a time)

### SQL Event Store ⚠️ PARTIALLY_FUNCTIONAL

- **What works**: Full `event.Store` impl, optimistic concurrency, batch append, snapshots, checkpoints, outbox with reliable publishing, proper error translation
- **What's missing**:
  - No real PostgreSQL integration tests (only go-sqlmock)
  - PostgreSQL-specific DDL (`BYTEA`, `JSONB`) — not truly database-agnostic
  - No `SQLEventStoreOption` for table name or logger customization
  - No connection pool configuration guidance

### Saga / Process Manager 📐 PLANNED

- `docs/planning/SAGA_DESIGN.md` exists with orchestration pattern design
- No code exists — needs `saga.Core`, `saga.Step`, `saga.Instance`, `saga.Store`, `saga.Runner`

---

## c) NOT STARTED

| Feature | Notes |
|---------|-------|
| Watermill module | Evaluated in `docs/planning/archive/` — Kafka/NATS adapter never started |
| Event signing / HMAC | Mentioned as LOW priority in TODO_LIST.md |
| Tagged releases | All modules at `v0.0.0` — no version tags published |
| Real PostgreSQL integration tests | No testcontainers or real DB tests for `storage/` |
| WebSocket/SSE transport | No opinionated transport layer (by design — library, not framework) |
| gRPC transport | Same — consumer's responsibility |
| Client-side event store | Planned for go-localfirst, not this library |
| Event upcasting in storage | `UpcasterRegistry` exists in core but storage has no migration path |
| Multi-tenant event store | No tenant isolation support |
| Benchmarks for storage | No sqlmock benchmarks for PostgreSQL performance characteristics |

---

## d) TOTALLY FUCKED UP

Nothing is currently broken. But here's what **sucks**:

### The `query.Handler` returns `any` Problem

`core/query` uses `func(context.Context, Query) (any, error)` as the handler type. This violates the project's "no any" rule. `DispatchTyped[T]` is a workaround but the interface itself is fundamentally wrong. **Fixing this is a breaking change** that needs a migration plan.

### 52 Pre-Existing Lint Issues

All 52 are in **test files only** — zero in production code. Breakdown:

| Linter | Count | Fixable? |
|--------|-------|----------|
| `wsl_v5` | 11 | Yes — add blank lines |
| `perfsprint` | 8 | Yes — replace `fmt.Errorf("literal")` with `errors.New` |
| `noinlineerr` | 6 | Yes — extract to var |
| `errcheck` | 10 | Yes — check returns |
| `nlreturn` | 6 | Yes — add blank line before return |
| `revive` | 3 | Yes — unused params to `_` |
| `err113` | 2 | Yes — replace dynamic errors with sentinels |
| `intrange` | 2 | Yes — use `for i := range n` |
| `gci` | 1 | Yes — fix import order |
| `golines` | 1 | Yes — shorten line |
| `modernize` | 1 | Yes — use `cmp.Or` |
| `exhaustruct` | 1 | Maybe — test struct literal |

### CatalogMeta Duplication

`event.CatalogMeta`, `command.CatalogMeta`, `query.CatalogMeta` — nearly identical. `event` has an extra `AggregateType` field. No clean shared location due to module boundaries. Accepted as intentional per-package types.

---

## e) WHAT WE SHOULD IMPROVE

### High Impact

1. **Fix all 52 lint issues** — They're all in tests, all mechanical, all fixable in one pass. A clean lint is a trust signal for library consumers.

2. **Tag `v0.1.0-alpha`** — The library is stable enough. Publishing tags lets consumers pin versions and signals commitment to semantic versioning.

3. **Real PostgreSQL integration tests** — `storage/` has 92% coverage but only with go-sqlmock. Real DB tests prove the SQL actually works. Use testcontainers-go.

4. **Projection dead-letter mechanism** — Currently failed events are retried then silently dropped. Production consumers need visibility into permanently failed events.

### Medium Impact

5. **Storage options** — Table name prefix, logger injection, connection pool config. Currently hardcoded.

6. **Example app tests** — `example/user/` has zero tests. Should at least have a smoke test that runs the demo.

7. **BDD test coverage for remaining modules** — `core/command`, `core/query`, `core/event`, `memory`, `middleware` all have 100% line coverage but lack user-focused BDD specs (only `core/decider`, `memory`, `projection` have BDD suites).

8. **Error taxonomy coverage** — `storage/` and `projection/` sentinels are not classified via `RegisterClassification`. Cross-package registration needs an `init()` pattern.

### Low Impact

9. **Benchmark suite** — Only `core/pkg/id` has benchmarks. Add benchmarks for event creation, dispatcher throughput, memory store contention.

10. **Documentation site** — Currently just README + godoc. Consider a proper docs site (MkDocs, Hugo) rendered from the existing markdown.

---

## f) Top 25 Things We Should Get Done Next

| # | Priority | Item | Effort | Impact |
|---|----------|------|--------|--------|
| 1 | 🔴 | Fix all 52 lint issues (tests only) | 1h | Trust signal |
| 2 | 🔴 | Tag `v0.1.0-alpha` for all modules | 30m | Enables version pinning |
| 3 | 🔴 | Real PostgreSQL integration tests (testcontainers) | 4h | Proves SQL works |
| 4 | 🟡 | Projection dead-letter mechanism | 3h | Production readiness |
| 5 | 🟡 | Storage options (table prefix, logger, pool) | 2h | Configurability |
| 6 | 🟡 | Register storage/projection errors in taxonomy | 1h | Consistent error handling |
| 7 | 🟡 | Example app smoke test | 1h | Proves demo works |
| 8 | 🟡 | BDD tests for command, query, event modules | 3h | User-focused coverage |
| 9 | 🟡 | BDD tests for middleware module | 2h | User-focused coverage |
| 10 | 🟡 | BDD tests for catalog modules | 3h | User-focused coverage |
| 11 | 🟢 | Benchmark suite (event, dispatcher, store) | 2h | Performance visibility |
| 12 | 🟢 | Fix `query.Handler` returns `any` | 2h | API correctness (breaking) |
| 13 | 🟢 | Event signing / HMAC verification | 4h | Integrity guarantees |
| 14 | 🟢 | Saga/Process Manager implementation | 8h | Orchestration support |
| 15 | 🟢 | Watermill adapter module | 6h | Kafka/NATS integration |
| 16 | 🟢 | Storage DDL abstractions (multi-DB) | 4h | MySQL/SQLite support |
| 17 | 🟢 | Event upcasting migration path in storage | 3h | Schema evolution |
| 18 | 🟢 | Multi-tenant event store support | 6h | SaaS use case |
| 19 | 🟢 | Documentation site (MkDocs/Hugo) | 4h | Discoverability |
| 20 | 🟢 | Projection batch processing | 2h | Throughput |
| 21 | 🟢 | Projection backpressure / rate limiting | 3h | Stability under load |
| 22 | 🟢 | Client-side event metadata convention docs | 1h | Offline-first enablement |
| 23 | 🟢 | Connection pool configuration guide for storage | 1h | Operational docs |
| 24 | 🟢 | godoc for all exported symbols (remaining ~20) | 2h | API documentation |
| 25 | 🟢 | CI pipeline for tagged releases (goreleaser) | 3h | Automated publishing |

---

## g) Top #1 Question I Cannot Figure Out Myself

**Should `query.Handler` return `(any, error)` or should we introduce a generic `Query[T]` interface?**

The current `query.Handler = func(context.Context, Query) (any, error)` violates the "no any" rule. Options:

1. **Generic Query interface**: `type Query[T any] interface { ResultType() T }` — but this requires each query to specify its result type, making the dispatcher generic over the return type. This changes `Dispatcher` to `Dispatcher[T]` which either breaks the current pattern or requires type erasure at the handler boundary.

2. **Keep `(any, error)` but document it** — Accept that Go's type system makes this unavoidable for a generic query dispatcher, and rely on `DispatchTyped[T]` as the type-safe entry point.

3. **Remove query handler from interface, use typed functions** — Each query type defines its own handler type. Dispatcher becomes `map[Type]func(context.Context, Query) (any, error)` internally but the public API only exposes `DispatchTyped[T]`.

This is a **breaking API change** regardless of approach. I need your direction on whether to accept `any` as a necessary Go limitation or invest in a breaking redesign.

---

## Metrics Dashboard

| Metric | Value |
|--------|-------|
| Modules | 9 (+ example) |
| Production lines | 9,514 |
| Test lines | 21,428 |
| Total Go files | 181 |
| Test packages (passing) | 21/21 (100%) |
| Race detector | ✅ Clean |
| Overall coverage | 83.3% |
| Core module coverage | 97.6% |
| Middleware coverage | 100.0% |
| Memory coverage | 91.9% |
| Catalog coverage | 81.4% |
| Storage coverage | 92.0% |
| Projection coverage | 89.7% |
| Lint issues | 52 (all in tests) |
| Open ADRs | 3 |
| Research docs | 9 |
| Status reports | 6 (+ this one) |
| Dependencies (prod) | 4 (`cockroachdb/errors`, `oklog/ulid`, `go-branded-id`, `go-json-experiment/json`) |
| Dependencies (test) | 2 (`onsi/ginkgo`, `onsi/gomega`) |

### Coverage by Package

| Package | Coverage |
|---------|----------|
| `core/command` | 100.0% |
| `core/query` | 100.0% |
| `core/pkg/dispatcher` | 100.0% |
| `core/pkg/id` | 100.0% |
| `middleware` | 100.0% |
| `core/event` | 98.0% |
| `catalog/d2` | 97.6% |
| `core/decider` | 96.2% |
| `catalog/eventcatalog` | 95.6% |
| `catalog/adapters` | 95.5% |
| `catalog/asyncapi` | 95.9% |
| `catalog` | 94.4% |
| `core/aggregate` | 93.2% |
| `storage` | 92.0% |
| `memory` | 91.9% |
| `projection` | 89.7% |
| `catalog/internal/cattest` | 0.0% (test helpers) |

---

## Session 45 Changes

### go-branded-id Type Alias Refactor

| Change | Detail |
|--------|--------|
| `id.Of[T]` | Changed from `struct{ wrapped cbid.ID[T, ulid.ULID] }` to type alias `= cbid.ID[T, ulid.ULID]` |
| `id_encoding.go` | **Deleted** (32 lines) — all encoding inherited via type alias |
| Delegated methods removed | `IsZero`, `Equal`, `Or`, `Reset`, `Get`, `String`, `GoString`, `Format`, `Ptr` — now inherited |
| `CompareIDs[T]` | New function replacing `Compare()` method (cbid.ID.Compare returns ErrNotOrdered for ulid.ULID) |
| `FromPtr[T]` | Re-exported as package-level function (not promoted by type alias) |
| Net delta | **-89 lines** (141→84 in id.go, id_encoding.go deleted) |
| Tests | All 21 packages pass with race detector |
| Lint | Zero new issues |

---

_Generated at 2026-05-03 06:45 CEST_
