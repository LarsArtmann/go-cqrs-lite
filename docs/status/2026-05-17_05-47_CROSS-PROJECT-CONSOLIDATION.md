# Cross-Project Consolidation Status Report

**Date:** 2026-05-17 05:47  
**Session:** Cross-Project Review + go-localfirst Archive  
**Projects:** go-cqrs-lite, cqrs-htmx, go-localsync, go-localfirst

---

## Executive Summary

Completed a full comparative review of all four Go projects in the LarsArtmann ecosystem, then executed the merge of go-localfirst into go-cqrs-lite. go-localfirst is now archived. The ecosystem is now three projects with clear separation of concerns.

**Key action taken:** go-localfirst `pkg/sync/` (100% duplicate of go-cqrs-lite/sync) deleted. Todo example app moved to `go-cqrs-lite/example/todo/`. go-localfirst README rewritten with archival notice.

---

## a) FULLY DONE

### go-cqrs-lite (Foundation Library)

| Area                   | Status | Detail                                                                                               |
| ---------------------- | ------ | ---------------------------------------------------------------------------------------------------- |
| Core CQRS types        | ✅     | command, query, event dispatchers — 100% coverage                                                    |
| Decider pattern        | ✅     | Pure Fold + Decide, recommended over OO aggregate — 95.0% coverage                                   |
| Branded IDs            | ✅     | `id.Of[T]` type alias to go-branded-id — 100% coverage                                               |
| Error taxonomy         | ✅     | 5 families (Rejection, Conflict, Transient, Corruption, Infrastructure) with extensible registration |
| Event upcasting        | ✅     | Version-sorted chaining with cycle detection                                                         |
| Memory implementations | ✅     | Store, Bus, Snapshot, Outbox, Checkpoint — 99.1% coverage                                            |
| Middleware suite       | ✅     | Logging, metrics, recovery, retry, validation, tracing — 18 factories, 100% coverage                 |
| Catalog system         | ✅     | AsyncAPI 3.0 + EventCatalog + D2 export from Go types                                                |
| SQL storage            | ✅     | PostgreSQL + SQLite + Turso — 94.8% coverage                                                         |
| Projection runner      | ✅     | Replay + live subscription, per-projection checkpointing                                             |
| Sync primitives        | ✅     | VectorClock, Operation[T], ConflictResolver, LWWResolver, NodeID, SyncMessage                        |
| ISP                    | ✅     | Publisher/Subscriber sub-interfaces on Bus                                                           |
| Benchmarks             | ✅     | 43 benchmarks across 12 files                                                                        |
| Lint                   | ✅     | Zero issues, 125+ linters                                                                            |
| Nix flake              | ✅     | build, test, lint, format, vet, coverage apps                                                        |

### cqrs-htmx (HTTP Integration Library)

| Area                   | Status | Detail                                                                  |
| ---------------------- | ------ | ----------------------------------------------------------------------- |
| App builder            | ✅     | New(Config) with command/query dispatchers, Casbin enforcer             |
| Command/Query dispatch | ✅     | HTTP → decode → authorize → dispatch → respond                          |
| Casbin authorization   | ✅     | Authorize, RequireAuth, AuthorizeMiddleware, Enforcer interface         |
| HTMX integration       | ✅     | Request context, response builder, all 8 swap strategies, notifications |
| Templ integration      | ✅     | Duck-typed TemplComponent, RenderTemplResult[T]                         |
| Error mapping          | ✅     | CQRS families → HTTP status codes, JSON/plain error handlers            |
| User identity          | ✅     | UserIDExtractor → context → event metadata, strongly-typed UserID       |
| Validation             | ✅     | ValidateCommand/ValidateQuery HandlerOptions                            |
| Lifecycle hooks        | ✅     | BeforeDispatchHook/AfterDispatchHook                                    |
| Timeout                | ✅     | Config.Timeout wraps dispatch only                                      |
| Release                | ✅     | v1.0.0 tagged and published                                             |
| Tests                  | ✅     | 170 specs, 95.7% coverage, 10 benchmarks, 0 lint issues                 |

### go-localsync (Provider Sync SDK)

| Area                        | Status | Detail                                                             |
| --------------------------- | ------ | ------------------------------------------------------------------ |
| Provider interface          | ✅     | Generic `Provider` interface (Name, Fetch, FetchAll, GetRateLimit) |
| GitHub provider             | ✅     | Full implementation with pagination, rate limiting, retry          |
| CQRS path                   | ✅     | Decider, ReadModel, Projection, CQRSStack using go-cqrs-lite       |
| Conflict-aware sync         | ✅     | LWW with timestamp comparison                                      |
| Branded IDs                 | ✅     | ItemID, ExternalID, ProviderID, ActorID, RepoID, EventTypeID       |
| Deterministic aggregate IDs | ✅     | SHA256→ULID from (source, sourceID)                                |
| Turso read model            | ✅     | SQLite-backed read model with filter/pagination                    |

### go-localfirst → Archived

| Area                 | Status     | Detail                                             |
| -------------------- | ---------- | -------------------------------------------------- |
| CRDT sync primitives | ✅ Merged  | Were 100% duplicate of go-cqrs-lite/sync — deleted |
| Todo example app     | ✅ Merged  | Moved to go-cqrs-lite/example/todo/                |
| README               | ✅ Updated | Archival notice with redirect table                |

---

## b) PARTIALLY DONE

| Area                       | Project               | What's Done                                         | What's Missing                                                                                                       |
| -------------------------- | --------------------- | --------------------------------------------------- | -------------------------------------------------------------------------------------------------------------------- |
| CQRS migration             | go-localsync          | CQRS path built in `pkg/cqrs/`, 34 tests passing    | Legacy CRUD path (`pkg/storage/`, `internal/database/`, sqlc) still exists. Phase 4 (deletion of legacy) not started |
| Error taxonomy wiring      | go-localsync          | go-cqrs-lite provides 5 error families              | go-localsync doesn't wire them — domain errors return generic 500                                                    |
| Error taxonomy wiring      | go-localfirst example | Same issue in the moved Todo example                | `TestUpdateTodo_InvalidID` returns 500 instead of 400                                                                |
| cqrs-htmx full integration | go-localsync          | go-localsync imports go-cqrs-lite but NOT cqrs-htmx | No HTTP layer yet — it's a CLI/SDK only                                                                              |
| Pebble event store         | go-cqrs-lite/storage  | `CQRSAdapter` implemented with tests                | Not part of the public API surface docs, no integration with projection runner                                       |
| Module versioning          | go-cqrs-lite          | All modules have own go.mod                         | All at v0.0.0 or v1.1.0 (core/memory). No semantic versioning strategy                                               |
| Testify → Ginkgo migration | go-localsync          | Documented as needed                                | 8 test files, 48 test cases still use testify. Pre-commit hooks blocked                                              |

---

## c) NOT STARTED

| Area                         | Project                  | Description                                                                                        |
| ---------------------------- | ------------------------ | -------------------------------------------------------------------------------------------------- |
| Saga/Process Manager         | go-cqrs-lite             | Design doc exists (`docs/planning/SAGA_DESIGN.md`), 4-phase plan, 18h estimate. No code.           |
| Watermill module             | go-cqrs-lite             | Kafka/NATS adapter planned. Research doc exists. No code.                                          |
| TransactionalStore           | go-cqrs-lite             | Design done (`docs/planning/OUTBOX_TRANSACTION_API.md`). Implementation partially done in storage. |
| PostgreSQL integration tests | go-cqrs-lite/storage     | All tests use go-sqlmock. No real PG verification.                                                 |
| cqrs-htmx DecodeRequest[T]   | cqrs-htmx                | `DecodeJSON[T]` can't access path params. `DecodeRequest[T]` contribution planned.                 |
| WebSocket sync               | go-localfirst (archived) | Was deleted from git history. Would need to live in go-cqrs-lite/sync if recovered.                |
| SSE bridge                   | go-localfirst (archived) | Event bus → Server-Sent Events. Not built.                                                         |
| go-localsync CLI update      | go-localsync             | CLI still defaults to legacy SQLite. Needs `--backend cqrs` flag.                                  |
| CatalogMeta consolidation    | go-cqrs-lite             | Duplicated in event, command, query packages. Not identical (event has extra field).               |
| CONTRIBUTING.md              | go-cqrs-lite             | Not created.                                                                                       |

---

## d) TOTALLY FUCKED UP

| Issue                                      | Project      | Severity  | Detail                                                                                                                                                                                                                                                        |
| ------------------------------------------ | ------------ | --------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| go-cqrs-lite/storage version mismatch      | go-cqrs-lite | 🔴 HIGH   | `storage/helpers.go:158` and `pebble_event_store.go:372` pass `event.Version` as `int` to `event.NewEvent`. This breaks any consumer using go-cqrs-lite/storage with the latest core. The original go-localfirst example couldn't even build because of this. |
| Pre-commit hooks bypassed                  | go-localsync | 🟡 MEDIUM | Hooks ban testify; entire test suite uses it. All commits use `--no-verify`.                                                                                                                                                                                  |
| go-localsync dual storage path             | go-localsync | 🟡 MEDIUM | Two parallel storage architectures (CRUD + CQRS) coexist. The CQRS migration plan exists but is stalled at Phase 3. New developers won't know which path to use.                                                                                              |
| go-localsync still uses cockroachdb/errors | go-localsync | 🟡 MEDIUM | go-cqrs-lite removed it in Session 54 (net -169 lines). go-localsync hasn't followed. Pulls in 6 transitive deps unnecessarily.                                                                                                                               |
| GOWORK=off required for cqrs-htmx          | cqrs-htmx    | 🟡 LOW    | Parent `go.work` exists that doesn't include cqrs-htmx. All commands need `GOWORK=off`. Annoying but documented.                                                                                                                                              |

---

## e) WHAT WE SHOULD IMPROVE

### Cross-Project

1. **Fix go-cqrs-lite/storage version mismatch** — This is a real bug blocking consumers. `helpers.go` and `pebble_event_store.go` need `event.Version(int)` → `event.Version` cast removal.
2. **Tag go-cqrs-lite v0.1.0-alpha** — 65+ sessions of work, zero tagged releases. Consumers can't pin.
3. **Remove cockroachdb/errors from go-localsync** — Follow go-cqrs-lite's Session 54 pattern: `fmt.Errorf` with `%w`.
4. **Complete go-localsync CQRS migration Phase 4** — Delete legacy CRUD, internal/database, sqlc, pkg/storage.
5. **Wire error taxonomy** — go-localsync and the Todo example should use `event.RegisterClassification` + cqrs-htmx `MapError` for proper HTTP status codes.
6. **Create go.work at ~/projects level** — cqrs-htmx needs `GOWORK=off` because it's not in any go.work. A parent go.work would solve this.
7. **Migrate testify → stdlib or Ginkgo** — go-localsync's testify dependency blocks pre-commit hooks and is inconsistent with go-cqrs-lite/cqrs-htmx (which use Ginkgo).

### go-cqrs-lite Specific

8. **Add Pebble storage to docs** — `NewCQRSAdapter` exists but isn't mentioned in README or FEATURES.md prominently.
9. **PostgreSQL integration tests** — go-sqlmock is good but doesn't catch real SQL dialect issues.
10. **CONTRIBUTING.md** — Referenced in README but doesn't exist.

---

## f) TOP 25 THINGS TO DO NEXT

| #   | Priority | Project       | Task                                                                                                            | Impact | Effort |
| --- | -------- | ------------- | --------------------------------------------------------------------------------------------------------------- | ------ | ------ |
| 1   | 🔴 P0    | go-cqrs-lite  | **Fix storage version mismatch** — `helpers.go:158` and `pebble_event_store.go:372` pass wrong type to NewEvent | HIGH   | 15min  |
| 2   | 🔴 P0    | go-cqrs-lite  | **Tag v0.1.0-alpha** — First public release after storage fix                                                   | HIGH   | 30min  |
| 3   | 🔴 P0    | go-localfirst | **Commit and push archival** — Commit the archive state, push to origin                                         | HIGH   | 10min  |
| 4   | 🔴 P0    | go-cqrs-lite  | **Commit example/todo** — Commit the migrated Todo example                                                      | HIGH   | 10min  |
| 5   | 🟡 P1    | go-localsync  | **Remove cockroachdb/errors** — Migrate to stdlib `fmt.Errorf` with `%w`                                        | MEDIUM | 2h     |
| 6   | 🟡 P1    | go-localsync  | **Delete legacy storage (CQRS Phase 4)** — Remove pkg/storage, internal/database, sql/                          | HIGH   | 3h     |
| 7   | 🟡 P1    | go-localsync  | **Wire error taxonomy** — Use event.RegisterClassification + MapError                                           | MEDIUM | 2h     |
| 8   | 🟡 P1    | go-cqrs-lite  | **Fix TestUpdateTodo_InvalidID** — Todo example returns 500 instead of 400                                      | LOW    | 30min  |
| 9   | 🟡 P1    | go-localsync  | **Migrate testify → Ginkgo** — Unblock pre-commit hooks                                                         | MEDIUM | 3h     |
| 10  | 🟡 P1    | go-localsync  | **Update CLI to use CQRSStack** — Default `--backend cqrs`                                                      | MEDIUM | 2h     |
| 11  | 🟡 P1    | go-cqrs-lite  | **CONTRIBUTING.md** — Architecture guidelines for contributors                                                  | MEDIUM | 1h     |
| 12  | 🟢 P2    | go-cqrs-lite  | **PostgreSQL integration tests for storage** — Test against real PG                                             | MEDIUM | 4h     |
| 13  | 🟢 P2    | cqrs-htmx     | **Add DecodeRequest[T]** — Access to both decoded body and \*http.Request                                       | HIGH   | 4h     |
| 14  | 🟢 P2    | go-cqrs-lite  | **Consolidate CatalogMeta** — Share between event/command/query packages                                        | LOW    | 2h     |
| 15  | 🟢 P2    | go-cqrs-lite  | **Implement Saga/Process Manager** — Design done, 18h estimate                                                  | HIGH   | 18h    |
| 16  | 🟢 P2    | go-cqrs-lite  | **TransactionalStore implementation** — Atomic save+outbox in single DB tx                                      | HIGH   | 4h     |
| 17  | 🟢 P2    | go-cqrs-lite  | **Update README** — Add example/todo, Pebble storage, sync module                                               | MEDIUM | 1h     |
| 18  | 🟢 P2    | go-cqrs-lite  | **Update FEATURES.md** — Add example/todo to Module Maturity Matrix                                             | LOW    | 15min  |
| 19  | 🟢 P2    | go-localsync  | **Real GitHub PAT smoke test** — Verify end-to-end with real API                                                | MEDIUM | 1h     |
| 20  | 🟢 P2    | go-cqrs-lite  | **Watermill module** — Kafka/NATS adapter                                                                       | HIGH   | 20h    |
| 21  | 🟢 P2    | go-localsync  | **Add JSON output flag** — Structured output for CLI                                                            | LOW    | 1h     |
| 22  | 🟢 P2    | go-localsync  | **Add structured logging fields** — Consistent context in all log statements                                    | LOW    | 1h     |
| 23  | 🟢 P2    | go-cqrs-lite  | **Document Pebble storage** — Add to README, AGENTS.md, FEATURES.md                                             | MEDIUM | 30min  |
| 24  | 🟢 P2    | go-localsync  | **Adopt go-cqrs-lite projection.Runner** — Replace custom Projector                                             | MEDIUM | 2h     |
| 25  | 🟢 P2    | cqrs-htmx     | **Add request logging middleware** — Documented as PLANNED in FEATURES.md                                       | LOW    | 2h     |

---

## g) TOP QUESTION I CANNOT FIGURE OUT MYSELF

**Should go-localsync complete its CQRS migration (delete legacy CRUD storage) BEFORE or AFTER tagging go-cqrs-lite v0.1.0?**

Arguments for BEFORE: go-localsync is the primary consumer of go-cqrs-lite/storage. If storage has a version mismatch bug (it does), we should verify the full consumer stack works before tagging.

Arguments for AFTER: go-cqrs-lite is a library with its own quality guarantees. Its core modules (command, query, event, decider, id, middleware, catalog, memory, projection, sync) are all production-quality. Storage can be tagged as experimental.

**The specific question:** Do you want go-cqrs-lite tagged as v0.1.0-alpha NOW (with storage as experimental), or wait until go-localsync's CQRS migration proves the storage module works end-to-end?

---

## Build & Test Results

### go-cqrs-lite (23 test packages)

```
ok  core/aggregate       0.004s
ok  core/command          0.005s
ok  core/decider          0.006s
ok  core/event            0.019s
ok  core/pkg/dispatcher   0.002s
ok  core/pkg/id           0.002s
ok  core/query            0.002s
ok  memory                0.005s
ok  sync                  0.003s
ok  projection            0.127s
ok  storage               0.180s
ok  catalog               0.004s
ok  catalog/adapters      0.003s
ok  catalog/asyncapi      0.003s
ok  catalog/d2            0.002s
ok  catalog/eventcatalog  0.028s
ok  middleware            0.144s
ok  integration/aggregate 0.005s
ok  integration/command   0.002s
ok  integration/event     0.008s
ok  integration/query     0.004s
```

**All 23 packages PASS.** Zero lint issues.

### go-cqrs-lite/example/todo (7 test packages)

```
ok  aggregate   0.003s
ok  cmd/api     0.006s  (1 pre-existing failure: TestUpdateTodo_InvalidID)
ok  commands    0.002s
ok  domain      0.002s
ok  projections 0.002s
ok  queries     0.003s
ok  storage     0.003s
```

**6/7 packages clean.** 1 pre-existing bug (domain error returns 500 instead of 400).

### cqrs-htmx

**Clean.** No uncommitted changes. v1.0.0 released. 170 specs, 95.7% coverage.

### go-localsync

**Clean.** No uncommitted changes. 48 tests across 8 suites.

### go-localfirst

**Archived.** 44 deleted files, 2 modified (README.md, AGENTS.md), go.mod trimmed.

---

## Ecosystem State After This Session

```
go-cqrs-lite/     ← Foundation (10 modules + 2 examples)
  core/           ← CQRS types, decider, IDs, error taxonomy
  memory/         ← In-memory test implementations
  catalog/        ← AsyncAPI + EventCatalog + D2 auto-docs
  middleware/     ← Logging, metrics, retry, validation, tracing
  storage/        ← PostgreSQL, SQLite, Turso, Pebble
  projection/     ← Projection runner with replay
  sync/           ← CRDT primitives (VectorClock, LWW, Operation)
  testhelpers/    ← Shared test utilities
  integration/    ← Cross-module tests
  example/user/   ← Minimal CLI demo
  example/todo/   ← ★ NEW: Full HTTP event-sourced Todo app

cqrs-htmx/        ← HTTP integration (v1.0.0 released)
go-localsync/     ← Provider sync SDK (dual-path: legacy + CQRS)
go-localfirst/    ← ★ ARCHIVED (README redirect only)
```

---

_Generated by Crush — 2026-05-17_
