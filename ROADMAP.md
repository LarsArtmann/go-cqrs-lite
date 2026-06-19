# Roadmap — go-cqrs-lite

> Where we are, where we're going, and what's next.

---

## Short Term (Next 90 Days)

### Sprint 1: Trust & Documentation (Week 1)

- [x] `FEATURES.md` updated with all planned features and honest status
- [x] `docs/DOMAIN_LANGUAGE.md` — CQRS & Event Sourcing glossary (≥20 terms)
- [x] `CONTEXT.md` — architecture overview & consumer patterns
- [x] `ROADMAP.md` — this file
- [x] gosec security scanning in devShell + CI with SARIF upload
- [x] `.go-arch-lint.yml` with module dependency layer rules + CI step

### Sprint 2: Operational Readiness (Week 2)

- [x] Health check middleware (`/health`, `/health/live`, `/health/ready`)
- [x] Metrics HTTP handler (request count, error rate, avg response time)
- [x] Graceful shutdown helper (`pkg/gracefulshutdown`)
- [x] Operational endpoints in `example/user/`

### Sprint 3: Testing Rigor — PBT & Snapshots (Week 3)

- [x] `pgregory.net/rapid` property tests on `decider/` — deterministic decide, version monotonicity
- [x] `rapid` property tests on `event/` — event immutability, version correctness
- [x] `rapid` property tests on `id/` — ULID validity, prefix correctness
- [x] `go-snaps` snapshot tests on `integration/` — event JSON serialization, catalog exports
- [x] ~~`go-snaps` snapshot tests on `catalog/` — AsyncAPI, OpenAPI, D2, EventCatalog exports~~ — DONE (golden_test.go in all 4 sub-packages)
- [x] ~~`go-snaps` snapshot tests on `projection/` — state rendering~~ — DONE (golden_test.go with replay-order snapshot)

### Sprint 4: CI & Deployment (Week 4)

- [x] Save `benchmark-baseline.txt` from all benchmarks
- [x] Add CI step: fail if any benchmark >2× slower than baseline
- [x] Dockerfile for `example/user/` (multi-stage: builder → scratch → alpine)
- [x] `docker-compose.yml` for example stack
- [x] Docker build CI step (linux amd64 + arm64)
- [x] PostgreSQL integration tests (`pg_integration_test.go` with `-tags=integration`)

### Sprint 5: Consumer Experience (Week 5–6)

- [x] ~~`example/catalog-server/`~~ — consolidated into `example/user/` catalog.go
- [x] `middleware/sse.go` — SSE broker over event bus + tests
- [x] ~~SSE handler in `example/user/` + JavaScript client~~ — available in `middleware/` package
- [x] `pkg/config/` module — YAML config loader with env-specific overlays
- [x] ~~Config usage example in `example/user/`~~ — DONE (config_usage_example.go with env overlay demo)
- [x] `integration/simulation/` — event sequence generator + decider stress tests
- [x] Event store throughput simulation benchmark
- [~] ~~Playwright setup in `example/user/`~~ — **NOT APPLICABLE**: `example/user` is a CLI demo with no HTTP server. `example/todo` has comprehensive Go integration tests.`
- [~] ~~Playwright E2E test: health endpoint~~ — **NOT APPLICABLE** (see above)
- [~] ~~Playwright E2E test: core command→event→query flow~~ — **NOT APPLICABLE** (see above)
- [~] ~~Playwright CI step~~ — **NOT APPLICABLE** (see above)
- [x] ~~Dual store runtime switching in `example/user/` (memory vs SQL)~~ — DONE (dual_store_example.go)

### Sprint 6: Polish & Experiments (Week 7–8)

- [x] ~~Document experimental build tags (`jsonv2`, `arenas`, `simd`, `runtimesecret`)~~ — DONE (docs/EXPERIMENTAL_BUILD_TAGS.md)
- [~] ~~`go-snaps` across all remaining modules~~ — **DESCOPED**: Using `eventtest.AssertGolden` pattern instead (already integrated)
- [x] ~~`rapid` PBT on `command/` and `query/` modules~~ — DONE (property_test.go in both modules, 9 tests)
- [~] ~~`jsonv2` codec experiment behind build tag~~ — **EXPERIMENTAL**: Not yet available in stable Go. Tracked in `docs/EXPERIMENTAL_BUILD_TAGS.md`
- [~] ~~Arena allocation experiment in event module~~ — **EXPERIMENTAL**: Not yet available in stable Go. Tracked in `docs/EXPERIMENTAL_BUILD_TAGS.md`

### Sprint 7: CQRS Audit Trail (Post v2.3.0 — ✅ COMPLETE)

Symmetric persistence across all three CQRS message types (events, commands, queries) for complete auditability.

- [x] Command journal interfaces (`CommandJournal`, `SeekableCommandJournal`)
- [x] `MemoryCommandStore.ReadAll()` + `ReadFrom()` implementation
- [x] Query store interfaces (`PersistedQuery`, `QuerySink/Source/Store`, `QueryJournal`, `SeekableQueryJournal`)
- [x] `MemoryQueryStore` implementation
- [x] ~~Tests for command journal + query store (memory implementations)~~ — DONE (`0c0cd5b3`)
- [x] ~~Query module store-specific sentinel errors~~ — DONE (`query/errors.go`: `ErrStoreClosed`, `ErrQueryNotFound`)
- [x] ~~`SQLCommandStore` → add journal support~~ — DONE (`bf7b3ed8`)
- [x] ~~`SQLQueryStore` — SQL backend~~ — DONE (`bf7b3ed8`)
- [x] ~~`SQLBackend.QueryStore()` facade method~~ — DONE (`bf7b3ed8`)

### Sprint 8: Bundle Composition Layer (v2.7.0 — ✅ COMPLETE)

The overarching goal: **consumers stop deciding on infrastructure.** A deployer
picks a backend via one preset call; the application imports only `readmodel`
and `stack`, never a storage driver. Eight new modules shipped this release.

- [x] `readmodel/` — typed `Store[T,K]` over `kv.Store` with codec + key prefixing
- [x] `readmodel/cache/` — Otter-backed `CachedStore[T,K]` decorator (write-through)
- [x] `stack/` — `Bundle` composition root (ISP-honest fields, pointer-dedup Close)
- [x] Presets: `stack/memory`, `stack/sqlite`, `stack/pebble`, `stack/postgres`
- [x] `stack/bench/` — zero-overhead benchmarks
- [x] Typed stores: `TypedSnapshot[State]`, `TypedCommandStore[P]`, `TypedQueryStore[P]`
- [x] Pebble gaps: CommandStore, QueryStore, `ReadModels()` on Backend
- [x] **Persistent read models for SQL presets** — `storage.SQLKVStore` (kv.Store over `cqrs_kv`); SQLite + Postgres presets now keep read models across restarts
- [x] Shared contract test suite (`stack/contracttest`) — 5 tests × 4 presets
- [x] Postgres preset tests run in CI (env-var mismatch fixed)

### Post-Bundle direction (next themes)

- [x] **Multi-process pub/sub** — Postgres `LISTEN/NOTIFY` event bus (`storage.PostgresBus`) with lightweight reference payloads, driver-agnostic `NotificationListener` interface (with `Listen(channel)`), and re-fetch with visibility-gap retry. Wired into `stack/postgres` via `WithDistributedBus(PgxListener)` — PgxListener uses pgxpool with a dedicated conn. Real-Postgres integration tests in CI.
- **Operability helpers** — health-check, graceful-shutdown, and backup/restore exposed from `stack/` presets (Pebble `Checkpoint` is available but not surfaced).
- **Transports** — `transport/grpc`, `transport/nats`, `transport/redis` per ADR-0025, composing over the Bundle.
- **Read-model query ergonomics** — secondary indexes / ranged scans for large read-model sets (today `Scan` filters in memory).

---

## Long Term Vision (6–12 Months)

### Performance

- [ ] SIMD-accelerated event serialization (Go experiment)
- [ ] Arena allocation for high-throughput event creation
- [ ] Zero-allocation event encoding path (jsonv2)
- [x] ~~Streaming event reads without materializing full slice~~ — DONE: `StreamingSource`/`StreamingJournal` implemented on SQL, Pebble, and Memory stores

### Reliability

- [~] ~~Outbox pattern~~ — **REMOVED**. Use [Watermill](https://github.com/ThreeDotsLabs/watermill) for reliable at-least-once publishing. The `watermill/` adapter already exists in this repo.
- [~] ~~Saga module~~ — **HARD NO**. Vertical scaling (bigger server) is sufficient for this library's scope. Multi-instance orchestration is the consumer's concern.
- [x] ~~Event schema registry with validation middleware~~ — **DONE**: `schema.Validator` with `RegisterType[T]()`, `RegisterTypeWithValidator[T]()`, strict/lenient modes, custom codec. ADR-0017 accepted.
- [x] ~~Distributed checkpointing for projections~~ — DONE: `DistributedRunner` with `LeaderElection` gating
- [x] ~~**Projection replay→live dedup gap**~~ — **FIXED** (commit `8d4ea2cc`). The Runner's `subscribeLive` now builds a reactive pipeline: `live → DistinctByEventIDWith(replayIDs) → handler`. Event IDs from journal replay seed the dedup set, so overlap-window duplicates are silently suppressed. New exports: `event.SubscriberToObservable`, `event.DistinctByEventIDWith`.

### Consumer Experience

- [x] ~~Code generator (`cqrs-gen`) v2 with struct tag scanning~~ — DONE: v3 adds event handler generation (`-type=event`)
- [x] ~~WebAssembly compilation target for decider module~~ — DONE: 7/7 core modules compile to WASM
- [ ] gRPC transport adapter
- [ ] NATS / Redis Stream adapter
- [~] GraphQL query adapter for projections → **HARD NO**: framework-level concern, not library scope

### Observability

- [x] ~~Built-in pprof endpoints~~ — **DONE**: `middleware.ProfilingHandler()` and `middleware.RegisterProfiling()` expose all pprof endpoints via `net/http`
- [x] ~~Custom metrics exporter (Prometheus format)~~ — **DONE**: `prometheus/` module with `Setup()`, `WithRegistry()`, `MustSetup()`. OTel→Prometheus bridge.
- [x] ~~Structured logging middleware with configurable levels~~ — **DONE**: `middleware.NewLogging[M]()` with `*slog.Logger` (levels configurable via `slog.HandlerOptions`). `EventLogging`, `CommandLogging`, `QueryLogging` constructors.
- [x] ~~Distributed tracing span propagation across module boundaries~~ — **DONE**: OTel spans on event/command/query middleware (`EventTracing`, `EventPublishTracing`, `CommandTracing`). Baggage propagation via `cqrsotel.WithCorrelationID`, `cqrsotel.CorrelationIDFromContext`, `cqrsotel.NewTextMapPropagator` (W3C trace + baggage).

---

## Raw Ideas (No Design Yet)

- Event stream compaction / log truncation strategies
- Multi-tenant event store (schema-per-tenant)
- Event archival to S3 / GCS / Azure Blob
- CQRS-lite dashboard (web UI for inspecting aggregates, events, projections)
- Automatic migration generator for schema evolution
- Property-based integration testing with state machine verification
- Chaos engineering integration (random network partitions, disk failures)
- Performance regression dashboard (historical benchmark tracking)

---

_Last updated: 2026-06-19 (Sprint 9: streaming reads, DistributedRunner, PostgresBus wired+tested+reconnect, cqrs-gen events, WASM 7/7, pgx CVE fix. ROADMAP freshness audit: 4 stale `[ ]` items marked done.)_
