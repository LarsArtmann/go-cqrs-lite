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
- [ ] Docker build CI step (linux amd64 + arm64)

### Sprint 5: Consumer Experience (Week 5–6)

- [x] `example/catalog-server/` — embedded EventCatalog SPA server
- [x] `middleware/sse.go` — SSE broker over event bus + tests
- [x] ~~SSE handler in `example/user/` + JavaScript client~~ — DONE (sse_example.go with SSE broker demo)
- [x] `pkg/config/` module — YAML config loader with env-specific overlays
- [x] ~~Config usage example in `example/user/`~~ — DONE (config_usage_example.go with env overlay demo)
- [x] `integration/simulation/` — event sequence generator + decider stress tests
- [x] Event store throughput simulation benchmark
- [ ] Playwright setup in `example/user/`
- [ ] Playwright E2E test: health endpoint
- [ ] Playwright E2E test: core command→event→query flow
- [ ] Playwright CI step
- [x] ~~Dual store runtime switching in `example/user/` (memory vs SQL)~~ — DONE (dual_store_example.go)

### Sprint 6: Polish & Experiments (Week 7–8)

- [x] ~~Document experimental build tags (`jsonv2`, `arenas`, `simd`, `runtimesecret`)~~ — DONE (docs/EXPERIMENTAL_BUILD_TAGS.md)
- [ ] `go-snaps` across all remaining modules (signing, middleware, storage, listing, watermill, pebble, turso, codec, otel, schema, snapshot, memory)
- [x] ~~`rapid` PBT on `command/` and `query/` modules~~ — DONE (property_test.go in both modules, 9 tests)
- [ ] `jsonv2` codec experiment behind build tag
- [ ] Arena allocation experiment in event module

---

## Long Term Vision (6–12 Months)

### Performance

- [ ] SIMD-accelerated event serialization (Go experiment)
- [ ] Arena allocation for high-throughput event creation
- [ ] Zero-allocation event encoding path (jsonv2)
- [ ] Streaming event reads without materializing full slice

### Reliability

- [ ] Outbox pattern implementation (reliable at-least-once publishing)
- [ ] Saga module (orchestrated multi-step transactions)
- [ ] Event schema registry with validation middleware
- [ ] Distributed checkpointing for projections

### Consumer Experience

- [ ] Code generator (`cqrs-gen`) v2 with struct tag scanning
- [ ] WebAssembly compilation target for decider module
- [ ] gRPC transport adapter
- [ ] NATS / Redis Stream adapter
- [~] GraphQL query adapter for projections → **DECLINED**: framework-level concern, not library scope

### Observability

- [ ] Built-in pprof endpoints
- [ ] Custom metrics exporter (Prometheus format)
- [ ] Structured logging middleware with configurable levels
- [ ] Distributed tracing span propagation across module boundaries

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

_Last updated: 2026-06-08 (verified)_
_See `docs/planning/2026-06-08_00_08-SEC_LESSONS_INTEGRATION_PLAN.md` for detailed execution plan._
