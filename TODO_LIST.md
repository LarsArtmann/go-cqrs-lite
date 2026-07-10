# TODO List

**Updated:** 2026-07-10
**Scope:** Short- and mid-term actionable tasks only. Long-term vision lives in [ROADMAP.md](ROADMAP.md). Raw ideas live in [ROADMAP.md § Raw Ideas](ROADMAP.md#raw-ideas-no-design-yet).

## Legend

- `[ ]` = Open
- `[x]` = Done
- `[v4]` = Breaking change, deferred to v4
- `[BLOCKED]` = Blocked on upstream dependency

---

## P0 — Critical (correctness, CI green, trust)

- [ ] **Register stdlib error classifications** — Call `errorfamily.RegisterStdlibDefaults()` in an init() or documented startup path. Without this, `context.DeadlineExceeded` defaults to Transient (wrong: should be Transient but `sql.ErrNoRows` defaults to Transient when it should be Rejection, and `context.Canceled` should be Rejection). Storage layer encounters these constantly. See `docs/status/2026-07-06_03-39_ERROR-TAXONOMY-MIGRATION-STATUS.md` item C1.
- [ ] **Register database driver classifiers** — SQLite BUSY/LOCKED → Transient, CONSTRAINT → Conflict; Postgres error codes via `*pgconn.PgError`. `go-error-family` ships `RegisterClassifier` for exactly this. The storage layer would benefit enormously. See `docs/status/2026-07-06_03-39` item C2.
- [ ] **Fix `WithReplayByteBudget(0)` semantics** — The SSE auto-default of 8MB for unlimited replay silently broke the old "0 = disabled" contract. Either revert the auto-default (keep 0=disabled) or add a sentinel value (-1 = explicitly disabled). See `docs/status/2026-07-06_04-52_SSE-CONSTANT-WIRING-AND-SPAN-TEST.md` §d.

## P1 — High Value (architecture, consumer experience)

- [ ] **metadata/ package tests + doc.go** — The extracted `metadata/` module has no dedicated tests (Tracing.Merge, CustomData.Clone, MergeCustomMaps edge cases) and no package-level doc.go. Types are tested only transitively through event/command/query tests. See `docs/status/2026-07-09_08-46_DISPATCHER-JSONV2-FIXES.md` item b.
- [ ] **Consumer migration guide for id/ + metadata/ extraction** — Write a guide: "import id/ for AggregateRef, import metadata/ for Tracing/CustomData, stop importing event/ for these types." Three status reports flag this as not started.
- [ ] **Deprecated alias verification test** — Add a test verifying the `// Deprecated:` comments in event/ are correct (staticcheck SA1019 compliance). See `docs/status/2026-07-09_08-46` item c.
- [ ] **stack/v3 health checks** — `Bundle` lacks a `HealthCheck(ctx)` interface. Real services need liveness/readiness probes that verify event store connectivity. Cross-consumer feedback rates this HIGH severity. See `docs/feedback/2026-07-05_cross-consumer-integration-gaps.md` Gap 1.
- [ ] **stack/v3 topological shutdown ordering** — No way to express shutdown dependencies (close projections before event store). Either add `WithDependency()` or document the consumer pattern. See `docs/feedback/2026-07-05_cross-consumer-integration-gaps.md` Gap 2.
- [ ] **Update `scripts/check-module-layers.sh`** — Still enforces the old 7-layer system from pre-ADR-0046. The `metadata` module is not in the script at all. ADR-0046 defines the four-tier model. See `docs/status/2026-07-09_08-46` item b.

## P2 — Medium Value (test parity, observability, quality)

- [ ] **BDD tests for EventIdempotency middleware** — Only command+query idempotency have BDD coverage. The middleware package has a Ginkgo suite but the new event idempotency middleware has no BDD test. See `docs/status/2026-07-09_07-41_PARETO-EXECUTION-COMPLETE.md` #11.
- [ ] **CI check: go.work ↔ flake.nix testModules sync** — No automated check ensures every module in `go.work` is also in `flake.nix` testModules. The `idempotency/` module was missing from CI for weeks before discovery. See `docs/status/2026-07-08_19-40` item e5.
- [ ] **CI check: go.work ↔ api-stability tracking sync** — Same gap: no check ensures every module in `go.work` is tracked by `cmd/api-stability/main.go`. See `docs/status/2026-07-08_19-40` item e5.
- [ ] **Fix file-size violations** — Multiple production files exceed the 350-line CI limit: `projectionhost/worker.go` (473), `projectionhost/host.go` (433), `storage/relational/sink.go` (413), `catalog/eventcatalog/frontmatter_types.go` (375), `middleware/deadletter_sql.go` (368). Split by concern.
- [ ] **SSE large-payload test (>8MB)** — Add a test with events whose total payload exceeds the default byte budget to verify budget boundary behavior. The current test uses 9KB of data and passes by luck. See `docs/status/2026-07-06_04-52` §e.
- [ ] **Adopt `errorfamily.HTTPStatus()` in example/taskmanager** — The taskmanager HTTP handlers hand-roll error→status mapping. `go-error-family` ships `HTTPStatus(err)` that maps the 5-family taxonomy to HTTP status codes. See `docs/status/2026-07-06_03-39` item C4.
- [ ] **Add SECURITY.md** — Expected for public libraries handling encryption/signing. Document the security reporting policy and the scope of the signing/encryption modules.
- [ ] **Projection parallelism** — `WithParallelProjections()` — each projection in its own goroutine with own checkpoint. Consumer request from DiscordSync. See `docs/feedback/2026-07-05_DiscordSync.md`.

## P3 — Polish & Cleanup

- [ ] **Document dispatcher middleware-at-dispatch-time behavior** — The fix shipped (middleware can now be added in any order relative to Register), but AGENTS.md/SKILL.md may still have stale "must Use before Register" language. Verify and update docs. See `docs/status/2026-07-09_08-46` items e2, P1 #12-13.
- [ ] **Remove `codec/jsonv2_experiment.go`** — `JSONCodecV2` is redundant now that `JSONCodec` itself uses json/v2. See `docs/status/2026-07-09_08-46` P3 #25.
- [ ] **Add `metadata/` to AGENTS.md** — Module list and Quick Reference table don't include the `metadata` module. See `docs/status/2026-07-09_07-41` P3 #34-35.
- [ ] **README.md docs freshness** — Missing `encryption`, `turso`, `testutil` module sections. See `docs/quality/2026-06-13_06-43_DOCS_FRESHNESS.md`.
- [ ] **Review all `json.Marshal` calls for missing `Deterministic(true)`** — Every `json.Marshal` with map fields should default to `Deterministic(true)` in library code where output might be compared or cached. See `docs/status/2026-07-09_08-46` P2 #21.
- [ ] **Review all `json.Unmarshal` calls for missing `MatchCaseInsensitiveNames(true)`** — json/v2 defaults to case-sensitive field matching. Any untagged struct decode path silently produces zero values. See `docs/status/2026-07-09_08-46` P2 #22.
- [ ] **Add ADRs for json/v2 decisions** — Case-insensitive decode, deterministic encoding in catalog exports, dispatch-time middleware. See `docs/status/2026-07-09_08-46` P3 #30-32.

### Experimental / Go-stdlib-blocked

- [BLOCKED] **jsonv2 codec experiment** — `codec/jsonv2_experiment.go` exists behind `goexperiment.jsonv2` build tag (ADR-0026). Pending Go stdlib stabilization (expected Go 1.27+). Note: `JSONCodec` already uses json/v2 experimentally; this file is redundant (see P3 cleanup).
- [BLOCKED] **Arena allocation experiment** — `event/arena_experiment.go` exists behind `goexperiment.arenas` build tag (ADR-0026). Pending Go arena API stabilization.
- [BLOCKED] **Turso MVCC concurrent-write support** — The Turso Database engine has experimental MVCC mode. Once stable, would allow raising `MaxOpenConns` above 1. Currently blocked: MVCC is experimental, `RunInTx` uses standard `BEGIN`, and needs conflict-retry logic. See `docs/design/blocked/BLOCKED-ITEMS.md`.

### Performance

- [ ] **Hot-State cache (decider)** — Optional `RepositoryOption[State]` that caches folded aggregate state keyed by `(aggID, version)`. Profile before building — snapshot + page-cache-resident events already make sequential loads cheap.
- [ ] **Read-pressure snapshot strategy** — `EveryNEvents` snapshots based on writes, but reads are the expensive path. Add a `ReadPressureStrategy`. Consider after hot-state cache.

### Transport

- [ ] **NATS/ValKey Stream adapter** — ADR-0025 accepted. Separate `transport/nats/` and `transport/redis/` modules. _(Author is not a fan of Redis; [ValKey](https://valkey.io) is the recommended alternative.)_
- [ ] **Distributed event bus** — `MemoryBus` is synchronous; Watermill GoChannel is single-process. No Redis/NATS backend for multi-process event distribution.

### Public Release Readiness

- [ ] **License swap (PROPRIETARY → Apache-2.0)** — Hard blocker for public adoption.
- [ ] **Git history scrub for internal docs** — AGENTS.md, docs/planning/*, docs/ActaFlow-* contain internal strategy/AI-workflow detail. Going public exposes ALL git history.
- [ ] **Postgres CI coverage matrix** — `stack/postgres` shows 0% coverage locally (tests skip without `POSTGRES_TEST_DSN`). Either add CI Postgres service or label experimental.
- [ ] **README polish to "sales page" standard** — Per project's own AGENTS.md rule, README should be a sales page for end-users.

---

## v4 Breaking Changes (deferred)

- [v4] **Flip codec defaults** — Events default to JSON, blind stores (KV/snapshot/command/query) default to JSON. v4 flips both to CBOR. Migration guide needed. See `docs/adr/0044-blind-store-encoding-stamps.md`.
- [v4] **Remove deprecated APIs** — `query.Handler` (replaced by `TypedHandler`), `memory.MemoryBus` (replaced by `watermill.EventBus`), deprecated event/ aliases for AggregateRef/Tracing/CustomData.
- [v4] **Storage/ split execution** — Proposal at `docs/planning/2026-07-09_STORAGE-SPLIT-PROPOSAL.md`. 109 files → 4 focused packages. Awaits approval.
- [v4] **Event/ god module decomposition** — event/ still owns 9 concerns (tombstone detection, command causality, replay mode, codec utilities, event construction, store interfaces, bus interfaces, checkpoint tracking, metadata). The deprecated aliases lay groundwork for extraction. See `docs/planning/2026-07-06_ARCHITECTURE_LAYERS_RECONSIDERED.md`.

---

## Recently Completed

- [x] **Dispatcher middleware-at-dispatch-time fix** — Middleware can now be added in any order relative to Register (was applied at registration time, silently bypassing all middleware if ordered wrong). See `docs/status/2026-07-09_08-46`.
- [x] **json/v2 case-insensitive decode fix** — `encoding/json/v2` defaults to case-sensitive; all decode paths now use `MatchCaseInsensitiveNames(true)`. See `docs/status/2026-07-09_08-46`.
- [x] **Catalog golden determinism fix** — Added `json.Deterministic(true)` to all catalog JSON marshaling (map iteration order was non-deterministic). See `docs/status/2026-07-09_08-46`.
- [x] **kv/ context.Context propagation** — All 11 kv I/O methods now accept `context.Context` across all implementations (MemStore, Pebble KVAdapter, SQLKVStore, TypedStore). See `docs/status/2026-07-09_07-41`.
- [x] **id/ + metadata/ extraction** — AggregateRef moved to id/, Tracing/CustomData moved to metadata/. command/ no longer depends on event/ at compile time. See `docs/status/2026-07-09_07-41`.
- [x] **Idempotency merge** — Generic `NewIdempotency[M]` factory + 3 wrappers in middleware/. Module slimmed to kv/ + go-error-family only. See `docs/status/2026-07-08_19-40`.
- [x] **Projectionhost production hardening** — M1-M13: checkpoint error fix, bounded dedup ring, WorkerDraining, WithShutdownTimeout, OTel tracing, OnFailed callback, Reset, jitter, integration tests. See `docs/status/2026-07-06_03-07`.
- [x] **Error taxonomy migration** — All `event.*` facade calls → `errorfamily.*` direct imports across 210+ files. See `docs/status/2026-07-06_03-39`.
- [x] **genproto conflict resolved** — Workspace-level replace directive in go.work pins genproto to a version where googleapis/rpc packages are split. transport/grpc is now a first-class workspace member.
- [x] **DOMAIN_LANGUAGE.md rebuild** — Complete rewrite, 303 lines, verified against source.
- [x] **Error taxonomy sweep** — All `fmt.Errorf` calls classified into 5-family taxonomy.
- [x] **Deriver module** — Event→command derivation (ADR-0040).
- [x] **Scenario-testing DSL** — Fluent Given/When/Then for deciders + projections.
- [x] **Scheduling module** — Durable deadline timers: `TimerStore`, `Scheduler`.
- [x] **Managed projection host** — `projectionhost.Host` with crash-restart, DLQ.
- [x] **Three projection tiers** — Materialize/KV, RelationalProjection/SQL, GraphProjection/graph.
- [x] **CBOR first-class codec** — Dual-codec (JSON + CBOR), mixed-stream decode.
- [x] **Bundle composition layer** — 5 presets (memory, sqlite, pebble, postgres, turso).
- [x] **KV store abstraction** — `kv.Store`, Pebble `KVAdapter`, `TypedStore[T,K]`, `Cache[T,K]`.

---

_Files read for this update: all `2026-07-0*` status/planning/feedback files, ROADMAP.md, v4-WISHLIST.md, BLOCKED-ITEMS.md, all docs/feedback/*, docs/quality/* freshness report, TODO_LIST.md (prior), and code verification of metadata/, catalog/, dispatcher/, projectionhost/, storage/, transport/http/, pebble/. Historical archive docs (docs/status/archive/*, docs/planning/archive/*) excluded as point-in-time snapshots._
