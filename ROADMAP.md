# Roadmap — go-cqrs-lite

> Where we are, where we're going, and what's next.
> **Last updated:** 2026-06-22

---

## Current State (v3.0.0 released)

**v3.0.0 is tagged** (2026-06-22) — all 38 modules are on `/v3` import paths. The 11 v3 breaking changes shipped (see [CHANGELOG.md](CHANGELOG.md) and the [v3 Migration Guide](docs/migration/V3_MIGRATION.md)). All core CQRS/ES primitives are shipped: Event Sourcing with branded IDs, CQRS command/query dispatch, tombstone soft-delete, event signing/encryption, OTel observability, Pebble/SQL/Turso storage backends, and a Bundle composition layer with 4 presets.

## Short Term (Next 90 Days)

### v3.0.0 — Shipped

All 11 breaking changes landed and v3.0.0 is tagged. The new shapes were added in v2, so migration is additive. See [`docs/migration/V3_MIGRATION.md`](docs/migration/V3_MIGRATION.md) for the full guide and [CHANGELOG.md](CHANGELOG.md) for the release notes.

### Transport Adapters (ADR-0025)

- [x] `transport/http/` — SSE event delivery (moved from middleware/). Healthcheck/metrics_http/pprof deleted (generic, zero CQRS deps, zero consumers).
- [x] `transport/grpc/` — protobuf command + query dispatch (commit `81d29455`)
- [ ] `transport/nats/` — JetStream publisher/subscriber
- [ ] `transport/redis/` — Redis Streams publisher/subscriber

### Read-Model Enhancements

- [ ] Secondary indexes / ranged scans for large read-model sets (today Scan filters in memory)

### Operability

- [ ] Surface Pebble Checkpoint (backup) from stack presets
- [ ] Surface graceful shutdown from stack presets

---

## Long Term Vision (6–12 Months)

### Performance

- [ ] Zero-allocation event encoding path (jsonv2 — blocked on Go stdlib)
- [ ] Arena allocation for high-throughput event creation (blocked on Go stdlib)
- [x] ~~Streaming event reads without materializing full slice~~ — DONE

### Reliability

- [x] ~~Event schema registry with validation middleware~~ — DONE
- [x] ~~Distributed checkpointing for projections~~ — DONE
- [x] ~~Projection replay→live dedup gap~~ — FIXED

### Consumer Experience

- [x] ~~Code generator (cqrs-gen) v3~~ — DONE
- [x] ~~WebAssembly compilation~~ — DONE (7/7 core modules)
- [ ] gRPC / NATS / Redis transport adapters
- [x] ~~Watermill EventBus~~ — DONE (replaces deprecated MemoryBus)

### Observability

- [x] ~~Built-in pprof endpoints~~ — DELETED (trivial net/http/pprof re-export, zero consumers)
- [x] ~~Prometheus metrics exporter~~ — DONE
- [x] ~~Structured logging middleware~~ — DONE
- [x] ~~Distributed tracing span propagation~~ — DONE

---

## Raw Ideas (No Design Yet)

- Event stream compaction / log truncation strategies
- Multi-tenant event store (schema-per-tenant)
- Distributed projection runner (leader election, multi-node coordination) — single-machine is sufficient for now; ROADMAP item with no deadline
- Event archival to S3 / GCS / Azure Blob
- CQRS-lite dashboard (web UI for inspecting aggregates, events, projections)
- Automatic migration generator for schema evolution
- Property-based integration testing with state machine verification
- Performance regression dashboard (historical benchmark tracking)

---

_Last updated: 2026-06-22 — v3.0.0 tagged. All 11 breaking changes shipped (Metadata split ADR-0031, projection dissolution ADR-0030)._
