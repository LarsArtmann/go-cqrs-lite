# go-cqrs-lite Documentation

> A lightweight CQRS + Event Sourcing **library** for Go with branded IDs, tombstone soft-delete, and auto-documentation generation.

## Getting Started

| Resource                                        | Audience  | Description                                                            |
| ----------------------------------------------- | --------- | ---------------------------------------------------------------------- |
| **[Getting Started Guide](getting-started.md)** | New users | Step-by-step tutorial: events → commands → decider → branded IDs       |
| **[Project README](../README.md)**              | All       | Quick start, module index, feature comparison                          |
| **[SKILL.md](../SKILL.md)**                     | AI agents | Single-source AI consumer guide: decision matrix, recipes, conventions |
| **[Migration Guide v1](MIGRATION_v1.md)**       | Upgraders | Upgrade guide for v1 → v2 breaking changes                             |
| **[API Migration](MIGRATION.md)**               | Upgraders | `query.Handler: any → TypedHandler[T]` migration                       |

## Guides

- **[Migration Guide](MIGRATION_TO_STACK.md)** — How to replace hand-wired infrastructure with stack presets (200+ lines → 5 lines)
- **[Architecture Patterns](ARCHITECTURE_PATTERNS.md)** — CQRS, Event Sourcing, and Decider patterns explained
- **[Bundle Presets](PRESETS.md)** — Deployer-facing preset catalog (memory, SQLite, Pebble, Postgres, Turso)
- **[Infrastructure Recommendations](INFRASTRUCTURE_RECOMMENDATIONS.md)** — Which storage engine fits which CQRS concern, and why
- **[Storage Guide](STORAGE_GUIDE.md)** — SQL event store, snapshots, checkpoint stores, backend facade
- **[Signing Architecture](signing-architecture.md)** — Event signing with HMAC-SHA256 and Ed25519
- **[Error Taxonomy](error-taxonomy.md)** — 5-family error classification system (Rejection / Conflict / Transient / Infrastructure / Corruption)
- **[Domain Language](DOMAIN_LANGUAGE.md)** — Glossary of domain terms and ubiquitous language
- **[Turso Indexing Guidance](turso-indexing-guidance.md)** — Index management for Turso

## Modules

The authoritative module index with README links lives in the **[project README](../README.md)** (52 modules). Each module also has a `doc.go` rendered on [pkg.go.dev](https://pkg.go.dev/github.com/larsartmann/go-cqrs-lite/event/v4).

## Examples

| Example        | Demonstrates                                                         | README                                                |
| -------------- | -------------------------------------------------------------------- | ----------------------------------------------------- |
| **todo**       | Full app: HTTP API, decider, projections, queries, Pebble storage    | [example/todo](../example/todo/README.md)             |
| **user**       | Advanced: signing, middleware chains, catalog gen, tombstone/rebirth | [example/user](../example/user/README.md)             |
| **encryption** | Bus-level + store-level encryption, key rotation                     | [example/encryption](../example/encryption/README.md) |

## Architecture Decision Records (ADR)

34 ADRs documenting key architectural decisions. Full text in [`adr/`](adr/); the [ADR index](adr/README.md) contains summaries.

| ADR                                                   | Title                                             | Status                   |
| ----------------------------------------------------- | ------------------------------------------------- | ------------------------ |
| [0001](adr/0001-decider-over-aggregate.md)            | Decider Pattern Over OO Aggregate                 | Accepted                 |
| [0002](adr/0002-error-taxonomy.md)                    | Error Taxonomy with Five Families                 | Accepted                 |
| [0003](adr/0003-multi-module-monorepo.md)             | Multi-Module Monorepo Structure                   | Accepted                 |
| [0004](adr/0004-saga-process-manager.md)              | Saga / Process Manager Module                     | Accepted                 |
| [0005](adr/0005-tombstone-soft-delete.md)             | Tombstone Soft-Delete Pattern                     | Accepted                 |
| [0006](adr/0006-sink-source-split.md)                 | Sink/Source Split for Event Store Interface       | Accepted                 |
| [0007](adr/0007-gopls-workspace-workaround.md)        | gopls Multi-Module Workspace Workaround           | Accepted                 |
| [0008](adr/0008-typed-handler-signature.md)           | TypedHandler Dual Type Parameters                 | Accepted                 |
| [0009](adr/0009-pebble-scope-event-store-only.md)     | Pebble Module Scope                               | Accepted                 |
| [0010](adr/0010-remove-io-closer-from-interfaces.md)  | Remove io.Closer from Core Interfaces             | Accepted                 |
| [0011](adr/0011-unify-err-dispatcher-closed.md)       | Unify ErrDispatcherClosed                         | Accepted                 |
| [0012](adr/0012-split-catalog-modules.md)             | Split Catalog into Sub-Modules                    | Accepted                 |
| [0013](adr/0013-zero-copy-payload-for-decode.md)      | Zero-Copy Payload Access                          | Accepted                 |
| [0014](adr/0014-test-only-dependencies-in-go-mod.md)  | Test-Only Dependencies in go.mod                  | Accepted                 |
| [0015](adr/0015-cbor-codec.md)                        | CBOR Codec for Binary Payload Encoding            | Accepted                 |
| [0016](adr/0016-outbox-pattern.md)                    | Outbox Pattern for Reliable Event Publishing      | Declined — use Watermill |
| [0017](adr/0017-schema-registry.md)                   | Schema Registry for Event Validation              | Proposed                 |
| [0018](adr/0018-distributed-checkpointing.md)         | Distributed Checkpointing for Projections         | Proposed                 |
| [0019](adr/0019-cbor-envelope-format.md)              | CBOR Envelope Format for Pebble Stores            | Accepted                 |
| [0020](adr/0020-performance-optimization-patterns.md) | Performance Optimization Patterns                 | Accepted                 |
| [0021](adr/0021-store-close-semantics.md)             | Store Close() Semantics — Shared DB Pattern       | Accepted                 |
| [0022](adr/0022-kv-store-abstraction.md)              | KV Store Abstraction Module                       | Accepted                 |
| [0023](adr/0023-pebble-kv-adapter.md)                 | Pebble KV Store Adapter                           | Accepted                 |
| [0024](adr/0024-exported-id-markers.md)               | Exported ID Marker Types                          | Accepted                 |
| [0025](adr/0025-transport-adapter-strategy.md)        | Transport Adapter Strategy                        | Accepted                 |
| [0026](adr/0026-experimental-features.md)             | Experimental Features Behind Build Tags           | Accepted                 |
| [0027](adr/0027-postgres-listen-notify-bus.md)        | Postgres LISTEN/NOTIFY Event Bus                  | Accepted                 |
| [0028](adr/0028-watermill-as-delivery-layer.md)       | Watermill as the Delivery Layer                   | Accepted                 |
| [0029](adr/0029-storage-consolidation.md)             | Consolidate Storage Backends Under `storage/`     | Accepted                 |
| [0030](adr/0030-dissolve-projection.md)               | Dissolve `projection/` into CatchUp + Materialize | Accepted                 |
| [0031](adr/0031-metadata-split.md)                    | Typed Metadata Fields — Embed Tracing             | Accepted                 |
| [0032](adr/0032-merge-readmodel-into-kv.md)           | Merge `readmodel/` into `kv/`                     | Accepted                 |
| [0033](adr/0033-multi-db-split.md)                    | Multi-Database Split for Concern Isolation        | Accepted                 |
| [0034](adr/0034-session-store-boundary.md)            | Session Store Boundary                            | Accepted                 |
| [0035](adr/0035-branded-dsn-types.md)                 | Branded DSN Types (Considered and Rejected)       | Rejected                 |
| [0046](adr/0046-seven-tier-model.md)                  | Seven-Tier Dependency Model                       | Accepted                 |

## API Reference

- **[API Surface Snapshot](api_surface.txt)** — Machine-readable list of all exported symbols
- **[pkg.go.dev](https://pkg.go.dev/github.com/larsartmann/go-cqrs-lite/event/v4)** — Godoc for every module
- Run `go run ./cmd/api-stability/ -update` to regenerate the surface snapshot
- Run `go run ./cmd/api-stability/` to verify no breaking changes against the golden file

## Benchmarks

- **[Benchmarks](benchmarks/README.md)** — Performance baselines and regression detection

## Diagrams

- [Module Architecture](architecture-understanding/) — D2 diagrams of current and target architecture
- [Perfect World Modules](perfect-world-modules.svg) — Ideal module layout
- [Web Client Communication](web-client-communication.svg) — Client → Server event flow

---

## Internal Development Artifacts

> **Not user documentation.** These are historical development records — status snapshots, execution plans, research notes, and code reviews. Useful for understanding the project's evolution, not for learning the library.

| Directory                            | Contents                                                     |
| ------------------------------------ | ------------------------------------------------------------ |
| [`status/`](status/)                 | Comprehensive status snapshots from each development session |
| [`planning/`](planning/)             | Execution plans, Pareto analyses, and roadmaps               |
| [`research/`](research/)             | Technology evaluations, deep dives, and design proposals     |
| [`quality/`](quality/)               | Code quality scans, architecture reviews, naming audits      |
| [`sessions/`](sessions/)             | Session history and milestones                               |
| [`modularization/`](modularization/) | Module boundary analysis and restructuring history           |
| [`brainstorming/`](brainstorming/)   | Design explorations and concept drafts                       |
| [`feedback/`](feedback/)             | External feedback and comparison reports                     |
