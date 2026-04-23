# Comprehensive Status Report

**Date:** 2026-04-23 05:54
**Branch:** master
**Last Commit:** 4d5a210 — docs(planning): complete multi-module monorepo redesign

## Executive Summary

go-cqrs-lite is a ~14,600 LOC CQRS library in Go. All tests pass (race-clean, vet-clean).
Today's session produced a deep code review, a comprehensive multi-module monorepo migration plan,
and several architectural decisions. No production code was changed — all work was planning and documentation.

---

## A) FULLY DONE

### Planning & Documentation (This Session)

| Item | Status | File |
|---|---|---|
| In-depth project review | ✅ Complete | `docs/planning/2026-04-23_PROJECT_REVIEW.md` |
| Multi-module monorepo plan | ✅ Complete | `docs/planning/2026-04-23_MULTI_MODULE_MONOREPO_PLAN.md` |
| ULID vs UUID decision | ✅ Decided: ULID | Documented in plan |
| sqlc for SQL storage | ✅ Decided: sqlc | Based on template-sqlc |
| Watermill for pub/sub | ✅ Decided: Watermill | Replaces hand-rolled nats/redis |
| Storage concern separation | ✅ Decided: 4 independent modules | storage, watermill, projection, snapshot |
| Custom YAML marshaler | ✅ Deleted, replaced with go-faster/yaml | Committed in 3c09f0b |
| Go 1.25 subdirectory module roots | ✅ Researched | Documented in plan |
| Watermill pro/contra analysis | ✅ Exists | `docs/planning/2026-04-23_WATERMILL_PRO_CONTRA.md` |

### Codebase (Pre-existing)

| Item | Status |
|---|---|
| All tests pass | ✅ 18 test packages, 0 failures |
| Race detector clean | ✅ No races |
| go vet clean | ✅ No issues |
| Core CQRS types | ✅ command, query, event, aggregate |
| Generic dispatcher | ✅ `internal/dispatcher/` with generics |
| Branded IDs | ✅ `pkg/id/` with `id.Of[T]` |
| Catalog system | ✅ AsyncAPI 3.0 + EventCatalog export |
| Memory implementations | ✅ MemoryStore, MemoryBus, MemorySnapshotStore |
| Event sourcing | ✅ Store, Bus, Snapshot, Repository |
| BDD tests | ✅ Ginkgo/Gomega suites for event + aggregate |
| Benchmarks | ✅ All packages benchmarked |
| CI workflows | ✅ test.yml + lint.yml |

---

## B) PARTIALLY DONE

| Item | Current State | What Remains |
|---|---|---|
| **Phase 0 of migration plan** | Items 1-5 partially done prior (see commit history), but not verified against current plan checklist | Verify each item, confirm go-faster/yaml migration complete |
| **Test coverage** | Most packages >80% | `catalog/adapters` at 66%, `aggregate` at 77.3%, `internal/dispatcher` at 77.4% |
| **Module boundaries** | Planned but not started | Phase 1-10 of migration plan |
| **Event sourcing BDD** | Ginkgo suites exist but some scenarios incomplete | More edge cases needed |
| **Middleware** | 5 middleware implemented | No tracing (OTel), no idempotency middleware |

---

## C) NOT STARTED

| Item | Priority | Module |
|---|---|---|
| Multi-module migration (go.work) | HIGH | Phase 1-10 |
| Query handler context.Context fix | HIGH | core/query |
| ULID migration (replace google/uuid) | MEDIUM | core/pkg/id |
| Storage module (sqlc event store) | HIGH | storage/ |
| Watermill module (pub/sub) | HIGH | watermill/ |
| Projection module (read models) | HIGH | projection/ |
| Snapshot module (SQL-backed) | MEDIUM | snapshot/ |
| Testutil module (AggregateTester, etc.) | MEDIUM | testutil/ |
| Event Codec interface | MEDIUM | core/event |
| Event Upcasting | LOW | core/upcasting |
| Outbox pattern implementation | HIGH | storage/ |
| Projection runner + checkpoint | HIGH | projection/ |
| err113 sentinel error fixes | LOW | all modules |
| Compatibility shim (old → new import paths) | MEDIUM | root module |
| go-import meta tag hosting | LOW | GitHub Pages |
| Schema migration tool decision | LOW | storage/ |
| Examples CI integration | LOW | examples/ |
| pkg/errors deletion | LOW | pkg/ (already dead code) |

---

## D) TOTALLY FUCKED UP

| Item | Problem | Severity |
|---|---|---|
| **Query handler missing context.Context** | `query.Dispatcher.Dispatch` ignores ctx (`_ = ctx`). Handler signature is `func(Query) (any, error)` — no context. This breaks tracing, cancellation, timeouts, and middleware. | 🔴 CRITICAL — fundamental API design flaw |
| **Duplicate Phase 6 in old plan** | Was fixed in the rewrite, but indicates the plan grew organically without structure review | 🟡 Minor |
| **Broken example modules** | `example/catalog/go.mod` is stale, all imports broken. LSP shows 60+ errors. Not in CI. | 🔴 HIGH — examples silently broken |
| **pkg/errors is dead code** | Defined but never imported anywhere. Confuses users about error handling strategy. | 🟡 Moderate |
| **No production event store** | Only MemoryStore exists. The entire storage/ module is a plan, not code. Users can't use this in production today. | 🔴 CRITICAL — blocks real-world usage |

---

## E) WHAT WE SHOULD IMPROVE

### Architecture

1. **Fix query handler signature** — add `context.Context` before migration. This is a breaking change; do it now while the module is still single-path.
2. **Add Projection concept to core** — currently the "Q" in CQRS is completely missing. Users have no way to build read models.
3. **Add Event Codec interface** — `[]byte` payload with no serialization strategy is a footgun. Define `Codec` in core with JSON default.
4. **Add Event Upcasting** — events evolve. Without an upcasting mechanism, old events become unreadable after schema changes.

### Developer Experience

5. **Add testutil module** — every user will build `AggregateTester` and `ProjectionTester` themselves. Ship it.
6. **Fix examples** — broken examples are worse than no examples. Put them in go.work, test in CI.
7. **Write a getting-started guide** — pkg.go.dev isn't enough for a framework. Need architecture overview + migration guide.

### Code Quality

8. **Raise `catalog/adapters` coverage from 66% to >80%** — lowest tested package, critical for doc generation.
9. **Fix all err113 warnings** — 9 instances of dynamic errors that should be sentinels. Linter is correctly flagging them.
10. **Delete pkg/errors** — dead code adds confusion.

### Operational

11. **Add outbox pattern to storage module** — without it, events are silently lost on bus publish failure after SQL commit.
12. **Add OpenTelemetry tracing** — middleware for distributed tracing is essential for debugging production CQRS flows.
13. **Add schema migration strategy** — storage/ needs a supported path (golang-migrate, goose, or raw SQL).

---

## F) TOP 25 THINGS TO DO NEXT

### Tier 1: Unblock Production Use (Do First)

| # | Task | Module | Effort |
|---|---|---|---|
| 1 | Fix query handler to include `context.Context` | core/query | S |
| 2 | Create `go.work` + move into `core/` subdirectory | root | M |
| 3 | Extract memory implementations to `memory/` module | memory | M |
| 4 | Implement `storage/` module with sqlc (PostgreSQL first) | storage | L |
| 5 | Implement outbox pattern in `storage/` | storage | M |
| 6 | Implement `watermill/` module (Redis Streams first) | watermill | M |
| 7 | Implement `projection/` module with checkpoint tracking | projection | L |

### Tier 2: Complete the SDK (Do Second)

| # | Task | Module | Effort |
|---|---|---|---|
| 8 | Implement `snapshot/` module (SQL-backed) | snapshot | M |
| 9 | Implement `testutil/` module (AggregateTester, ProjectionTester) | testutil | M |
| 10 | Add Event Codec interface to core (JSON default) | core/event | S |
| 11 | Migrate from google/uuid to oklog/ulid | core/pkg/id | M |
| 12 | Extract catalog to own module with go-faster/yaml | catalog | M |
| 13 | Extract middleware to own module | middleware | S |
| 14 | Extract xtypes to own module | xtypes | S |

### Tier 3: Polish & Production Hardening (Do Third)

| # | Task | Module | Effort |
|---|---|---|---|
| 15 | Add MySQL + SQLite schemas to storage/ | storage | M |
| 16 | Add Event Upcasting interface + implementation | core/upcasting | M |
| 17 | Fix all err113 linter warnings (sentinel errors) | all | S |
| 18 | Raise catalog/adapters coverage to >80% | catalog | S |
| 19 | Delete pkg/errors (dead code) | pkg | XS |
| 20 | Fix broken example modules, add to CI | examples | S |
| 21 | Add compatibility shim for old import paths | root | S |
| 22 | Set up go-import meta tag hosting (GitHub Pages) | docs | S |
| 23 | Add OpenTelemetry tracing middleware | middleware | M |
| 24 | Write getting-started guide + architecture overview | docs | M |
| 25 | Tag v1.0.0 releases for all modules | all | S |

---

## G) TOP #1 QUESTION I CANNOT FIGURE OUT MYSELF

**Should `projection/` and `snapshot/` depend on `storage/`?**

The current plan says they "may optionally depend on storage/ for SQL-backed persistence." But this creates a tight coupling:

- Option A: `projection/` and `snapshot/` import `storage/` directly → simpler, but forces the same SQL engine for all concerns
- Option B: `projection/` and `snapshot/` depend only on `core/` interfaces (`Store`, `Bus`) and users wire them to `storage/` themselves → more flexible, but more boilerplate
- Option C: `projection/` and `snapshot/` accept a `*sql.DB` or `sqlc` interface directly → engine-specific but no module dependency

This affects the dependency graph, the user experience, and how independent these modules truly are. I need your call on this architectural decision.

---

## Test Coverage Summary

| Package | Coverage | Status |
|---|---|---|
| catalog/asyncapi | 96.3% | ✅ |
| xtypes | 95.7% | ✅ |
| event | 95.4% | ✅ |
| query | 91.5% | ✅ |
| catalog | 91.2% | ✅ |
| catalog/eventcatalog | 89.7% | ✅ |
| pkg/id | 85.4% | ✅ |
| command | 84.4% | ✅ |
| middleware | 84.6% | ✅ |
| catalog/yaml | 84.4% | ✅ |
| internal/dispatcher | 77.4% | ⚠️ |
| aggregate | 77.3% | ⚠️ |
| catalog/adapters | 66.0% | 🔴 |
| pkg/errors | 0.0% | 💀 dead code |

## Benchmark Summary

| Operation | ns/op | allocs |
|---|---|---|
| Command dispatch | 59.75 | 0 |
| Command dispatch + middleware | 82.65 | 2 |
| Event creation | 239.8 | 5 |
| MemoryBus publish | 29.28 | 1 |
| MemoryStore save | 2,200 | 11 |
| MemoryStore load | 46.54 | 1 |
| ID generation (ULID) | 118.3 | 2 |
| ID parse | 1.326 | 0 |
| SchemaFromType (reflect) | 518.8 | 15 |
| AsyncAPI export | 1,882 | 46 |
| AsyncAPI YAML marshal | 16,392 | 323 |
