# Comprehensive Execution Plan — go-cqrs-lite

**Date:** 2026-04-30
**Scope:** ALL remaining work across the entire codebase
**Constraint:** Each task ≤ 12 minutes

---

## Scoring System

| Dimension          | Weight | Description                                 |
| ------------------ | ------ | ------------------------------------------- |
| **Impact**         | 40%    | How much does this improve the system?      |
| **Effort**         | 30%    | How fast is it? (inverse — lower is better) |
| **Customer Value** | 20%    | Does a user care?                           |
| **Risk**           | 10%    | What breaks if we don't do this?            |

**Score = (Impact × 0.4) + ((12 − EffortMin) × 0.3) + (CustomerValue × 0.2) + (Risk × 0.1)**

Higher score = do first.

---

## TIER 1: Quick Wins (Coverage + Dead Code) — Do First

| #    | Task                                                   | File(s)                                | Effort | Impact | Customer Value | Risk | Score | Notes                                                |
| ---- | ------------------------------------------------------ | -------------------------------------- | ------ | ------ | -------------- | ---- | ----- | ---------------------------------------------------- |
| 1.1  | **Remove dead `evtest` package**                       | `core/event/internal/evtest/`          | 5min   | HIGH   | LOW            | LOW  | 8.5   | 0% coverage, not imported anywhere, internal only    |
| 1.2  | **Test `ensureMetadata` when metadata already exists** | `core/event/options.go:9`              | 5min   | MEDIUM | LOW            | LOW  | 7.0   | 50% → 100%, trivial test                             |
| 1.3  | **Test `Metadata()` nil metadata path**                | `core/event/event.go:116`              | 5min   | MEDIUM | LOW            | LOW  | 7.0   | 85.7% → 100%, cover `e.metadata == nil`              |
| 1.4  | **Test `loadEvents` snapshot error branch**            | `core/aggregate/repository.go:138`     | 8min   | MEDIUM | LOW            | LOW  | 6.8   | 85% → 100%, create failing snapshot store            |
| 1.5  | **Test `DefaultRetryConfig.IsRetryable`**              | `middleware/middleware.go:40`          | 5min   | LOW    | LOW            | LOW  | 6.0   | 50% → 100%, assert default func returns false        |
| 1.6  | **Test `collectionSchema` uncovered branches**         | `catalog/schema.go:33`                 | 8min   | MEDIUM | LOW            | LOW  | 6.5   | 66.7% → 100%, test slice/map/array types             |
| 1.7  | **Test `goTypeToJSON` uncovered branches**             | `catalog/schema.go:78`                 | 10min  | MEDIUM | LOW            | LOW  | 6.2   | 64.3% → 100%, test uint, float, interface{}          |
| 1.8  | **Test `SchemaToAny` marshal error path**              | `catalog/asyncapi/helpers.go:10`       | 5min   | LOW    | LOW            | LOW  | 5.5   | 75% → 100%, needs circular ref or unmarshalable type |
| 1.9  | **Test `writeSchema` nil schema path**                 | `catalog/eventcatalog/writer.go:105`   | 5min   | LOW    | LOW            | LOW  | 5.5   | 75% → 100%                                           |
| 1.10 | **Test `writePackageJSON` error path**                 | `catalog/eventcatalog/writer.go:155`   | 8min   | LOW    | LOW            | LOW  | 5.3   | 80% → 100%, use failing writer                       |
| 1.11 | **Test `AddService` merge path fully**                 | `catalog/registry.go:78`               | 8min   | MEDIUM | LOW            | LOW  | 6.5   | 91.7% → 100%, test all three message kinds merge     |
| 1.12 | **Test `addMessageToService` edge case**               | `catalog/adapters/builder.go:77`       | 5min   | LOW    | LOW            | LOW  | 5.5   | 87.5% → 100%                                         |
| 1.13 | **Test `MarshalBinary` error path**                    | `core/pkg/id/id_encoding.go:93`        | 5min   | LOW    | LOW            | LOW  | 5.5   | 83.3% → 100%                                         |
| 1.14 | **Test `UnmarshalBinary` edge case**                   | `core/pkg/id/id_encoding.go:110`       | 5min   | LOW    | LOW            | LOW  | 5.5   | 90.9% → 100%                                         |
| 1.15 | **Test `LoadAtVersion` before version**                | `memory/snapshot.go:75`                | 5min   | LOW    | LOW            | LOW  | 5.5   | 92.3% → 100%                                         |
| 1.16 | **Test `Ack` with nonexistent IDs**                    | `memory/outbox.go:76`                  | 5min   | LOW    | LOW            | LOW  | 5.5   | Already 92.3%, verify no-op on missing IDs           |
| 1.17 | **Test `schemaFromReflect` edge cases**                | `catalog/schema_reflect.go:10`         | 8min   | MEDIUM | LOW            | LOW  | 6.5   | 92.3% → 100%, test anonymous struct, ptr to basic    |
| 1.18 | **Test `writeMessage` error path**                     | `catalog/eventcatalog/exporter.go:117` | 8min   | LOW    | LOW            | LOW  | 5.3   | 91.3% → 100%                                         |
| 1.19 | **Test `Export` error path**                           | `catalog/eventcatalog/exporter.go:24`  | 8min   | LOW    | LOW            | LOW  | 5.3   | 91.7% → 100%                                         |

**Tier 1 Total: 19 tasks, ~2.5 hrs**

---

## TIER 2: Architecture Foundation (Interfaces + Patterns)

| #   | Task                                              | File(s)                         | Effort | Impact | Customer Value | Risk   | Score | Notes                                                    |
| --- | ------------------------------------------------- | ------------------------------- | ------ | ------ | -------------- | ------ | ----- | -------------------------------------------------------- |
| 2.1 | **Add `Codec` interface to core**                 | `core/event/codec.go`           | 10min  | HIGH   | HIGH           | MEDIUM | 8.2   | JSON, protobuf, msgpack pluggable encoding               |
| 2.2 | **Add `Upcaster` interface to core**              | `core/event/upcaster.go`        | 10min  | HIGH   | HIGH           | MEDIUM | 8.2   | Event versioning: V1 → V2 migration                      |
| 2.3 | **Add `Projection` interface to core**            | `core/projection/` (new)        | 12min  | HIGH   | HIGH           | HIGH   | 8.5   | The missing "Q" in CQRS — read models                    |
| 2.4 | **Add `CheckpointStore` interface**               | `core/projection/checkpoint.go` | 8min   | HIGH   | HIGH           | MEDIUM | 8.0   | Tracks which events each projection processed            |
| 2.5 | **Add `SnapshotStrategy` interface**              | `core/aggregate/` (new)         | 10min  | MEDIUM | MEDIUM         | LOW    | 7.0   | Every N events, time-based, on-demand                    |
| 2.6 | **Extract shared `Close()` pattern**              | `core/pkg/lifecycle/` (new)     | 10min  | MEDIUM | LOW            | LOW    | 6.0   | Currently duplicated across MemoryBus, MemoryStore, etc. |
| 2.7 | **Extract shared `Use()` middleware pattern**     | `core/pkg/middleware/` (new)    | 10min  | MEDIUM | LOW            | LOW    | 6.0   | Currently duplicated in memory/, middleware/             |
| 2.8 | **Add `EventCodec` to `NewEvent`/`EventBuilder`** | `core/event/event.go`           | 10min  | MEDIUM | HIGH           | LOW    | 7.2   | Wire Codec into event creation for automatic encoding    |
| 2.9 | **Add `Codec` default JSON implementation**       | `core/event/codec_json.go`      | 8min   | MEDIUM | HIGH           | LOW    | 7.5   | Default JSON codec using go-json-experiment/json         |

**Tier 2 Total: 9 tasks, ~1.5 hrs**

---

## TIER 3: Storage Module (Phase 5) — Critical Path

| #    | Task                                                 | File(s)                                      | Effort | Impact | Customer Value | Risk   | Score | Notes                                 |
| ---- | ---------------------------------------------------- | -------------------------------------------- | ------ | ------ | -------------- | ------ | ----- | ------------------------------------- |
| 3.1  | **Create `storage/` directory + `go.mod`**           | `storage/go.mod`                             | 5min   | HIGH   | HIGH           | HIGH   | 8.5   | Depends on core only                  |
| 3.2  | **Add `storage/` to `go.work`**                      | `go.work`                                    | 2min   | HIGH   | HIGH           | HIGH   | 9.0   | One-line change                       |
| 3.3  | **Create PostgreSQL event store schema**             | `storage/sql/postgres/schema/001_events.sql` | 10min  | HIGH   | HIGH           | HIGH   | 8.5   | Events table + outbox table + indexes |
| 3.4  | **Write sqlc queries (Save, Load, LoadFromVersion)** | `storage/sql/postgres/queries/events.sql`    | 12min  | HIGH   | HIGH           | HIGH   | 8.2   | Core event store operations           |
| 3.5  | **Write sqlc queries (Delete, AppendBatch)**         | `storage/sql/postgres/queries/events.sql`    | 8min   | HIGH   | HIGH           | HIGH   | 8.0   | Remaining Store interface methods     |
| 3.6  | **Create `sqlc.yaml` config**                        | `storage/sqlc.yaml`                          | 10min  | HIGH   | HIGH           | HIGH   | 8.2   | PostgreSQL engine, pgx/v5             |
| 3.7  | **Run `sqlc generate`**                              | `storage/internal/db/postgres/`              | 5min   | HIGH   | HIGH           | HIGH   | 8.5   | Generates type-safe Go                |
| 3.8  | **Implement `event.Store` adapter**                  | `storage/eventstore.go`                      | 12min  | HIGH   | HIGH           | HIGH   | 8.2   | Wraps sqlc-generated queries          |
| 3.9  | **Implement transactional outbox**                   | `storage/outbox.go`                          | 12min  | HIGH   | HIGH           | HIGH   | 8.2   | Same-tx event write + outbox append   |
| 3.10 | **Add schema migration helpers**                     | `storage/migration.go`                       | 10min  | MEDIUM | HIGH           | MEDIUM | 7.5   | golang-migrate or raw SQL runner      |
| 3.11 | **Write integration tests with testcontainers**      | `storage/eventstore_test.go`                 | 12min  | HIGH   | HIGH           | HIGH   | 8.2   | Real PostgreSQL in Docker             |
| 3.12 | **Write outbox integration tests**                   | `storage/outbox_test.go`                     | 10min  | HIGH   | HIGH           | HIGH   | 8.0   | Verify tx atomicity                   |
| 3.13 | **Add SQLite support (schema + queries)**            | `storage/sql/sqlite/`                        | 12min  | MEDIUM | MEDIUM         | MEDIUM | 7.0   | Alternative to PostgreSQL             |
| 3.14 | **Add MySQL support (schema + queries)**             | `storage/sql/mysql/`                         | 12min  | MEDIUM | MEDIUM         | MEDIUM | 7.0   | Alternative engine                    |

**Tier 3 Total: 14 tasks, ~2.5 hrs**

---

## TIER 4: Watermill Module (Phase 6) — Pub/Sub

| #   | Task                                              | File(s)                 | Effort | Impact | Customer Value | Risk   | Score | Notes                               |
| --- | ------------------------------------------------- | ----------------------- | ------ | ------ | -------------- | ------ | ----- | ----------------------------------- |
| 4.1 | **Create `watermill/` directory + `go.mod`**      | `watermill/go.mod`      | 5min   | HIGH   | HIGH           | MEDIUM | 8.0   | Depends on core                     |
| 4.2 | **Add `watermill/` to `go.work`**                 | `go.work`               | 2min   | HIGH   | HIGH           | MEDIUM | 8.5   | One-line change                     |
| 4.3 | **Implement `event.Bus` via Watermill Publisher** | `watermill/bus.go`      | 12min  | HIGH   | HIGH           | MEDIUM | 8.0   | Redis, NATS, Kafka, etc.            |
| 4.4 | **Add backend config helpers**                    | `watermill/config.go`   | 10min  | MEDIUM | HIGH           | LOW    | 7.2   | Redis Streams, NATS JetStream, etc. |
| 4.5 | **Write unit tests with in-memory bus**           | `watermill/bus_test.go` | 10min  | MEDIUM | HIGH           | LOW    | 7.2   | Watermill's Go channel pub/sub      |

**Tier 4 Total: 5 tasks, ~0.7 hrs**

---

## TIER 5: Projection Module (Phase 7) — Read Models

| #   | Task                                          | File(s)                     | Effort | Impact | Customer Value | Risk   | Score | Notes                                       |
| --- | --------------------------------------------- | --------------------------- | ------ | ------ | -------------- | ------ | ----- | ------------------------------------------- |
| 5.1 | **Create `projection/` directory + `go.mod`** | `projection/go.mod`         | 5min   | HIGH   | HIGH           | MEDIUM | 8.0   | Depends on core                             |
| 5.2 | **Add `projection/` to `go.work`**            | `go.work`                   | 2min   | HIGH   | HIGH           | MEDIUM | 8.5   | One-line change                             |
| 5.3 | **Implement `Runner` (subscribe + dispatch)** | `projection/runner.go`      | 12min  | HIGH   | HIGH           | MEDIUM | 8.0   | Subscribes to Bus, dispatches to handlers   |
| 5.4 | **Implement `Checkpoint` SQL store**          | `projection/checkpoint.go`  | 10min  | HIGH   | HIGH           | MEDIUM | 7.8   | Tracks per-projection position              |
| 5.5 | **Add `Projector` builder API**               | `projection/projector.go`   | 10min  | MEDIUM | HIGH           | LOW    | 7.5   | `projector.On("user.created", handler)`     |
| 5.6 | **Write integration tests**                   | `projection/runner_test.go` | 12min  | HIGH   | HIGH           | MEDIUM | 8.0   | Full flow: events → projection → read model |

**Tier 5 Total: 6 tasks, ~0.9 hrs**

---

## TIER 6: Snapshot Module (Phase 8)

| #   | Task                                        | File(s)                        | Effort | Impact | Customer Value | Risk | Score | Notes                                       |
| --- | ------------------------------------------- | ------------------------------ | ------ | ------ | -------------- | ---- | ----- | ------------------------------------------- |
| 6.1 | **Create `snapshot/` directory + `go.mod`** | `snapshot/go.mod`              | 5min   | MEDIUM | MEDIUM         | LOW  | 6.5   | Depends on core                             |
| 6.2 | **Add `snapshot/` to `go.work`**            | `go.work`                      | 2min   | MEDIUM | MEDIUM         | LOW  | 7.0   | One-line change                             |
| 6.3 | **Implement SQL-backed `SnapshotStore`**    | `snapshot/store.go`            | 12min  | MEDIUM | MEDIUM         | LOW  | 6.8   | Uses storage/ or raw SQL                    |
| 6.4 | **Implement snapshot strategies**           | `snapshot/strategy.go`         | 10min  | MEDIUM | MEDIUM         | LOW  | 6.5   | Every N events, time-based                  |
| 6.5 | **Wire snapshot strategy into repository**  | `core/aggregate/repository.go` | 10min  | MEDIUM | HIGH           | LOW  | 7.0   | `Save` checks strategy, calls SnapshotStore |
| 6.6 | **Write integration tests**                 | `snapshot/store_test.go`       | 10min  | MEDIUM | MEDIUM         | LOW  | 6.5   | Test save/load roundtrip                    |

**Tier 6 Total: 6 tasks, ~0.8 hrs**

---

## TIER 7: Known Issues + Small Fixes

| #   | Task                                               | File(s)                           | Effort | Impact | Customer Value | Risk | Score | Notes                                                |
| --- | -------------------------------------------------- | --------------------------------- | ------ | ------ | -------------- | ---- | ----- | ---------------------------------------------------- |
| 7.1 | **Fix `MemorySnapshotStore` deep copy of `State`** | `memory/snapshot.go`              | 8min   | LOW    | LOW            | LOW  | 5.5   | `Load` returns shallow copy; defensive copy needed   |
| 7.2 | **Fix `toDotAddress` number handling**             | `catalog/asyncapi/helpers.go:27`  | 8min   | LOW    | LOW            | LOW  | 5.3   | "Get3DView" → "get.3.d.view" should be "get.3d.view" |
| 7.3 | **Document `MemoryBus.Publish` RLock behavior**    | `memory/bus.go`                   | 5min   | LOW    | LOW            | LOW  | 5.5   | Acceptable for test utility, but needs godoc         |
| 7.4 | **Add `EventRetry` direct tests**                  | `middleware/retry_test.go`        | 10min  | MEDIUM | LOW            | LOW  | 6.0   | Currently only tested indirectly via `CommandRetry`  |
| 7.5 | **Add BDD test for aggregate error paths**         | `core/aggregate/cqrs_bdd_test.go` | 12min  | MEDIUM | LOW            | LOW  | 6.0   | More Ginkgo scenarios for failing stores             |

**Tier 7 Total: 5 tasks, ~0.7 hrs**

---

## TIER 8: Documentation + Release

| #    | Task                                                  | File(s)                                      | Effort | Impact | Customer Value | Risk   | Score | Notes                                                |
| ---- | ----------------------------------------------------- | -------------------------------------------- | ------ | ------ | -------------- | ------ | ----- | ---------------------------------------------------- |
| 8.1  | **Update AGENTS.md with session 11+ changes**         | `AGENTS.md`                                  | 10min  | MEDIUM | MEDIUM         | LOW    | 6.5   | ApplySnapshot, functional options, cached middleware |
| 8.2  | **Update README with new module structure**           | `README.md`                                  | 12min  | HIGH   | HIGH           | LOW    | 7.8   | Storage, watermill, projection modules               |
| 8.3  | **Write storage module design doc**                   | `docs/planning/2026-04-30_storage_design.md` | 10min  | MEDIUM | MEDIUM         | LOW    | 6.0   | Schema, sqlc config, build tags                      |
| 8.4  | **Tag `core/v1.0.0`**                                 | Git tags                                     | 3min   | HIGH   | HIGH           | HIGH   | 8.5   | First stable release of core                         |
| 8.5  | **Tag `memory/v1.0.0`**                               | Git tags                                     | 3min   | HIGH   | HIGH           | HIGH   | 8.5   | First stable release of memory                       |
| 8.6  | **Tag `catalog/v1.0.0`**                              | Git tags                                     | 3min   | HIGH   | HIGH           | MEDIUM | 8.0   | First stable release of catalog                      |
| 8.7  | **Tag `middleware/v1.0.0`**                           | Git tags                                     | 3min   | HIGH   | HIGH           | MEDIUM | 8.0   | First stable release of middleware                   |
| 8.8  | **Add GitHub Pages `index.html` with go-import tags** | `docs/index.html`                            | 10min  | MEDIUM | HIGH           | MEDIUM | 7.5   | Required for Go 1.25 subdirectory module resolution  |
| 8.9  | **Update CI matrix for new modules**                  | `.github/workflows/ci.yml`                   | 10min  | MEDIUM | MEDIUM         | LOW    | 6.5   | Test each module independently                       |
| 8.10 | **Write CONTRIBUTING.md update for new modules**      | `CONTRIBUTING.md`                            | 8min   | LOW    | LOW            | LOW    | 5.0   | Multi-module workflow instructions                   |
| 8.11 | **Add example modules to go.work + CI**               | `examples/*/go.mod`                          | 10min  | MEDIUM | MEDIUM         | LOW    | 6.0   | Currently broken/removed in session 9                |
| 8.12 | **Archive old planning docs**                         | `docs/status/archive/`                       | 5min   | LOW    | LOW            | LOW    | 4.5   | Keep workspace clean                                 |

**Tier 8 Total: 12 tasks, ~1.5 hrs**

---

## Summary

| Tier | Name                              | Tasks  | Est. Time   | Priority     |
| ---- | --------------------------------- | ------ | ----------- | ------------ |
| 1    | Quick Wins (Coverage + Dead Code) | 19     | ~2.5 hrs    | **CRITICAL** |
| 2    | Architecture Foundation           | 9      | ~1.5 hrs    | **HIGH**     |
| 3    | Storage Module                    | 14     | ~2.5 hrs    | **CRITICAL** |
| 4    | Watermill Module                  | 5      | ~0.7 hrs    | **HIGH**     |
| 5    | Projection Module                 | 6      | ~0.9 hrs    | **HIGH**     |
| 6    | Snapshot Module                   | 6      | ~0.8 hrs    | **MEDIUM**   |
| 7    | Known Issues + Small Fixes        | 5      | ~0.7 hrs    | **MEDIUM**   |
| 8    | Documentation + Release           | 12     | ~1.5 hrs    | **MEDIUM**   |
|      | **TOTAL**                         | **76** | **~11 hrs** |              |

---

## Recommended Execution Order

### Week 1 (Sessions 11–12): Foundation

1. Tier 1 (all 19 tasks) — coverage to 100% everywhere possible
2. Tier 7 (tasks 7.1–7.3) — fix known issues
3. Tier 2 (tasks 2.1–2.4) — core interfaces for Codec, Upcaster, Projection

### Week 2 (Sessions 13–14): Storage

4. Tier 3 (tasks 3.1–3.12) — PostgreSQL storage module + tests

### Week 3 (Sessions 15–16): Pub/Sub + Read Models

5. Tier 4 (all 5 tasks) — Watermill module
6. Tier 5 (all 6 tasks) — Projection module

### Week 4 (Session 17): Polish

7. Tier 6 (all 6 tasks) — Snapshot module
8. Tier 8 (all 12 tasks) — docs, tags, CI
9. Tier 2 (tasks 2.5–2.9) — remaining architecture
