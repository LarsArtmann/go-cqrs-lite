# go-cqrs-lite Documentation

> A lightweight CQRS + Event Sourcing library for Go with branded IDs, tombstone soft-delete, and auto-documentation generation.

## Getting Started

- **[Getting Started Guide](getting-started.md)** — Quick start tutorial
- **[README](../README.md)** — Project overview and quick reference
- **[Migration Guide v1](MIGRATION_v1.md)** — Upgrade guide for breaking changes

## Architecture

- **[Architecture Patterns](ARCHITECTURE_PATTERNS.md)** — CQRS, Event Sourcing, and Decider patterns
- **[Storage Guide](STORAGE_GUIDE.md)** — SQL event store, snapshots, outbox, checkpoint stores
- **[Signing Architecture](signing-architecture.md)** — Event signing with HMAC-SHA256 and Ed25519
- **[Error Taxonomy](error-taxonomy.md)** — 5-family error classification system
- **[Domain Language](DOMAIN_LANGUAGE.md)** — Glossary of domain terms

## Architecture Decision Records (ADR)

| ADR                                            | Title                      | Status   |
| ---------------------------------------------- | -------------------------- | -------- |
| [0001](adr/0001-decider-over-aggregate.md)     | Decider over Aggregate     | Accepted |
| [0002](adr/0002-error-taxonomy.md)             | Error Taxonomy             | Accepted |
| [0003](adr/0003-multi-module-monorepo.md)      | Multi-Module Monorepo      | Accepted |
| [0004](adr/0004-saga-process-manager.md)       | Saga Process Manager       | Accepted |
| [0005](adr/0005-outbox-pattern.md)             | Outbox Pattern             | Accepted |
| [0006](adr/0006-sink-source-split.md)          | Sink/Source ISP Split      | Accepted |
| [0007](adr/0007-gopls-workspace-workaround.md) | gopls Workspace Workaround | Accepted |

## Modules

| Module        | Purpose                                                              |
| ------------- | -------------------------------------------------------------------- |
| `core`        | Command, Event, Query, Decider, Branded IDs                          |
| `memory`      | In-memory Store, Bus, SnapshotStore                                  |
| `storage`     | SQL Store (PG/SQLite/Turso), Outbox, Checkpoint                      |
| `middleware`  | Logging, Retry, Recovery, Metrics, OTel, Circuit Breaker, Validation |
| `projection`  | Replay + Live runner, HandlerRegistry                                |
| `saga`        | Saga Runner, Steps, Compensation, Memory Store                       |
| `stream`      | Aggregate listing, Tombstone filtering, Cursor pagination            |
| `catalog`     | Registry, AsyncAPI/D2/OpenAPI exporters                              |
| `signing`     | Event signing/verification (HMAC, Ed25519)                           |
| `otel`        | Shared OpenTelemetry helpers (Tracer, Meter, Spans, Attributes)      |
| `watermill`   | Watermill protocol adapter                                           |
| `testhelpers` | Noop/Failing/Panic handlers, FakeStore, FakeBus                      |

## Examples

| Example              | Demonstrates                                              |
| -------------------- | --------------------------------------------------------- |
| `example/user`       | Full CQRS + Event Sourcing with Decider pattern           |
| `example/stream`     | Aggregate listing, tombstone filtering, cursor pagination |
| `example/storage`    | SQL-backed event store                                    |
| `example/projection` | Projection replay and live subscription                   |
| `example/saga`       | Saga orchestration with compensation                      |

## API Surface

- **[API Surface Snapshot](api_surface.txt)** — Machine-readable list of all 1058+ exported symbols
- Run `go run ./cmd/api-stability/ -update` to regenerate
- Run `go run ./cmd/api-stability/` to verify no breaking changes

## Quality & Status

- **[Status Reports](status/)** — Comprehensive status snapshots
- **[Planning](planning/)** — Execution plans and roadmaps
- **[Research](research/)** — Technology evaluations and deep dives

## Diagrams

- [Module Architecture](architecture-understanding/) — D2 diagrams of current and target architecture
- [Perfect World Modules](perfect-world-modules.svg) — Ideal module layout
- [Web Client Communication](web-client-communication.svg) — Client→Server event flow
