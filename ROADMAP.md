# Roadmap — go-cqrs-lite

> Where we are, where we're going, and what's next.
> **Last updated:** 2026-07-05

---

## Current State (v3.6.0 released)

**v3.6.0 is tagged** (2026-07-05) — all 47 modules are on `/v4` import paths. The library covers the full CQRS/ES lifecycle: event sourcing with branded IDs, command/query dispatch, pure-function deciders, three projection tiers (document/KV, relational/SQL, graph), durable deadline scheduling, event→command derivation, dead-letter quarantine, managed projection hosting, event signing/encryption, OTel tracing/metrics, and auto-documentation generation (AsyncAPI, OpenAPI, D2, EventCatalog).

v3.0.0 shipped all 11 breaking changes (see [CHANGELOG.md](CHANGELOG.md) and the [v3 Migration Guide](docs/migration/V3_MIGRATION.md)).

## Short Term

### v3.6.0 — Shipped (2026-07-05)

Error taxonomy sweep (5-family classification), deriver module (event→command derivation, ADR-0040), flagship example consolidation, DOMAIN_LANGUAGE.md rebuild.

### v3.5.0 — Shipped

Idempotency module, dispatch middleware, dead-letter queue (dispatch-tier), SSE promoted to branded types, scenario-testing DSL (Given/When/Then for deciders + projections), scheduling module (durable deadline timers), managed projection host (crash-restart lifecycle).

### v3.3.0 — Shipped (2026-06-28)

Three projection tiers (document/KV, relational/SQL, graph/traversal), Watermill command bridge, dead-letter quarantine, Drainer semantics, CI isolation gates.

### v3.1.0 — Shipped (2026-06-25)

SQL-backed view stores with queryable columns, multi-database split for all SQL presets, shared metadata utilities.

### v3.0.0 — Shipped (2026-06-22)

All 11 breaking changes landed. See [`docs/migration/V3_MIGRATION.md`](docs/migration/V3_MIGRATION.md) for the full guide and [CHANGELOG.md](CHANGELOG.md) for release notes.

### Transport Adapters (ADR-0025)

- [x] `transport/http/` — SSE event delivery (moved from middleware/). Healthcheck/metrics_http/pprof deleted (generic, zero CQRS deps, zero consumers).
- [x] `transport/grpc/` — protobuf command + query dispatch (commit `81d29455`)
- [x] ~~`transport/nats/`~~ — Superseded by Watermill command bridge (ADR-0025 revised). NATS/Redis/Kafka backends plug into `watermill.NewCommandBus` via publisher/subscriber adapters.
- [x] ~~`transport/redis/`~~ — Superseded by Watermill command bridge. See broker plugin recipe in `watermill/doc.go`.

> **Editorial note on Redis:** The author is not a fan of Redis. A native
> adapter may ship one day for consumers who already operate it — but even
> then, [ValKey](https://valkey.io) (the LF-backed Redis fork) is the
> recommended alternative. If you are starting fresh, pick ValKey, NATS, or
> Kafka instead.

### Read-Model Enhancements

- [x] ~~Secondary indexes / ranged scans for large read-model sets~~ — DONE (SQLViewStore supports IndexSpec, RelationalStore supports cursor pagination)
- [x] ~~Three projection tiers~~ — DONE (Materialize/KV, RelationalProjection/SQL, GraphProjection/graph)

### Operability

- [x] ~~Surface Pebble Checkpoint (backup) from stack presets~~ — DONE (`stack/pebble.Bundle.Checkpoint(dir)`)
- [x] ~~Surface graceful shutdown from stack presets~~ — DONE (`stack/pebble.Bundle.GracefulClose(ctx)`, `pebble.Backend.GracefulClose(ctx)`)

---

## Long Term Vision (6–12 Months)

### Performance

- [ ] Zero-allocation event encoding path (jsonv2 — blocked on Go stdlib)
- [ ] Arena allocation for high-throughput event creation (blocked on Go stdlib)
- [x] ~~Hot-State cache (decider)~~ — DONE (`decider/cache.go`: `StateCache[State]`, `WithStateCache`, LRU-bounded, 7.4x faster Load)
- [x] ~~Read-pressure snapshot strategy~~ — DONE (`snapshot/read_pressure.go`: `ReadPressure`, `AggregateAwareStrategy`, `ReadTracker`)
- [x] ~~Streaming event reads without materializing full slice~~ — DONE

### Reliability

- [x] ~~Event schema registry with validation middleware~~ — DONE
- [x] ~~Distributed checkpointing for projections~~ — DONE
- [x] ~~Projection replay→live dedup gap~~ — FIXED
- [x] ~~Dead-letter quarantine for retry exhaustion~~ — DONE
- [x] ~~Managed projection host (crash-restart lifecycle)~~ — DONE (`projectionhost/`)
- [x] ~~Durable deadline timers~~ — DONE (`scheduling/`)
- [x] ~~Event→command derivation (stateless saga)~~ — DONE (`deriver/`)

### Consumer Experience

- [x] ~~Code generator (cqrs-gen) v3~~ — DONE
- [x] ~~WebAssembly compilation~~ — DONE (7/7 core modules)
- [x] ~~Watermill EventBus~~ — DONE (replaces deprecated MemoryBus)
- [x] ~~Watermill CommandBus~~ — DONE (command pub/sub over any broker)
- [x] ~~gRPC transport~~ — DONE (remote command/query dispatch)
- [x] ~~SSE transport with Last-Event-ID reconnect~~ — DONE
- [x] ~~Scenario-testing DSL~~ — DONE (`scenario/` — Given/When/Then for deciders + projections)

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

---

_Last updated: 2026-07-11 — v3.6.0 tagged. All framework gaps (A1–A6) shipped: projectionhost, scenario, scheduling, deriver, idempotency. Performance features shipped: hot-state cache, read-pressure snapshots. Remaining: operability surfacing from stack presets, Go-stdlib-blocked jsonv2 build tag removal (Go 1.27+). Arena experiment removed — zero consumers, zero value._
