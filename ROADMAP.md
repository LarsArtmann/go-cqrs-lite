# Roadmap — go-cqrs-lite

> Where we are, where we're going, and what's next.
> **Last updated:** 2026-06-28

---

## Current State (v3.1.0 released, v3.3.0 in progress)

**v3.1.0 is tagged** (2026-06-25) — all 46 modules are on `/v3` import paths. v3.1.0 adds SQL-backed view stores with queryable columns, multi-database split for all SQL presets, shared metadata utilities, and 12 design documents for future features. v3.0.0 shipped all 11 breaking changes (see [CHANGELOG.md](CHANGELOG.md) and the [v3 Migration Guide](docs/migration/V3_MIGRATION.md)). All core CQRS/ES primitives are shipped.

**v3.3.0 (in progress)** adds three projection tiers (document/KV, relational/SQL, graph/traversal), a Watermill command bridge, dead-letter quarantine, Drainer semantics, and CI isolation gates.

## Short Term (Next 90 Days)

### v3.1.0 — Shipped (2026-06-25)

SQL-backed view stores, multi-database split, shared metadata utilities, 12 design documents. See [CHANGELOG.md](CHANGELOG.md).

### v3.0.0 — Shipped (2026-06-22)

All 11 breaking changes landed and v3.0.0 is tagged. The new shapes were added in v2, so migration is additive. See [`docs/migration/V3_MIGRATION.md`](docs/migration/V3_MIGRATION.md) for the full guide and [CHANGELOG.md](CHANGELOG.md) for the release notes.

### Transport Adapters (ADR-0025)

- [x] `transport/http/` — SSE event delivery (moved from middleware/). Healthcheck/metrics_http/pprof deleted (generic, zero CQRS deps, zero consumers).
- [x] `transport/grpc/` — protobuf command + query dispatch (commit `81d29455`)
- [x] ~~`transport/nats/`~~ — Superseded by Watermill command bridge (ADR-0025 revised). NATS/Redis/Kafka backends plug into `watermill.NewCommandBus` via publisher/subscriber adapters.
- [x] ~~`transport/redis/`~~ — Superseded by Watermill command bridge. See broker plugin recipe in `watermill/doc.go`.

### Read-Model Enhancements

- [x] ~~Secondary indexes / ranged scans for large read-model sets~~ — DONE (SQLViewStore supports IndexSpec, RelationalStore supports cursor pagination)
- [x] ~~Three projection tiers~~ — DONE (Materialize/KV, RelationalProjection/SQL, GraphProjection/graph)

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
- [x] ~~Dead-letter quarantine for retry exhaustion~~ — DONE

### Consumer Experience

- [x] ~~Code generator (cqrs-gen) v3~~ — DONE
- [x] ~~WebAssembly compilation~~ — DONE (7/7 core modules)
- [x] ~~Watermill EventBus~~ — DONE (replaces deprecated MemoryBus)
- [x] ~~Watermill CommandBus~~ — DONE (command pub/sub over any broker)
- [x] ~~gRPC transport~~ — DONE (remote command/query dispatch)
- [x] ~~SSE transport with Last-Event-ID reconnect~~ — DONE

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
- Neo4j/Memgraph graph driver (`graph/neo4j/`) — consumer-pulled sibling module
- Scheduler module (deadlines, timeouts, cron-style triggers)
- Deriver module (stateless saga: events → commands)

---

_Last updated: 2026-06-28 — v3.3.0 in progress. Three projection tiers, Watermill command bridge, dead-letter quarantine._
