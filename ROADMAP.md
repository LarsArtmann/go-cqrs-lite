# Roadmap — go-cqrs-lite

> Where we are, where we're going, and what's next.
> **Last updated:** 2026-06-20

---

## Current State (v2.x)

The library is at v2.x with 38 modules across a multi-module Go workspace. All core CQRS/ES primitives are shipped: Event Sourcing with branded IDs, CQRS command/query dispatch, tombstone soft-delete, event signing/encryption, OTel observability, Pebble/SQL/Turso storage backends, and a Bundle composition layer with 4 presets.

## Short Term (Next 90 Days)

### v3 Major Release

All breaking changes have been prepared additively in v2 — the v2 types consumers should migrate to already exist. See [`docs/migration/V3_MIGRATION.md`](docs/migration/V3_MIGRATION.md) for the full guide.

| #   | Breaking Change                                   | ADR                                                       | Status               |
| --- | ------------------------------------------------- | --------------------------------------------------------- | -------------------- |
| 1   | Delete ghost bus code (923 LOC)                   | [0028](docs/adr/0028-watermill-as-delivery-layer.md)      | Deprecated in v2     |
| 2   | Move memory/ → storage/memory/                    | [0029](docs/adr/0029-storage-consolidation.md)            | Done                 |
| 3   | Version → uint64                                  | —                                                         | Done                 |
| 4   | Break Metadata alias                              | [0031](docs/adr/0031-metadata-split.md)                   | Planned              |
| 5   | Remove io.Closer from interfaces                  | [0010](docs/adr/0010-remove-io-closer-from-interfaces.md) | Planned              |
| 6   | Delete readmodel/ (merged into kv/)               | [0032](docs/adr/0032-merge-readmodel-into-kv.md)          | Done                 |
| 7   | Delete projection/ (composable stack replaces it) | [0030](docs/adr/0030-dissolve-projection.md)              | In progress          |
| 8   | Move HTTP → transport/                            | [0025](docs/adr/0025-transport-adapter-strategy.md)       | Planned              |
| 9   | query.Handler: any → generic                      | [0008](docs/adr/0008-typed-handler-signature.md)          | TypedHandler shipped |

### Transport Adapters (ADR-0025)

- [ ] `transport/grpc/` — protobuf command dispatch + event pub/sub
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

- [x] ~~Built-in pprof endpoints~~ — DONE
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

_Last updated: 2026-06-20 — projection/ dissolution in progress (ADR-0030). Distributed projection is a ROADMAP item with no deadline._
