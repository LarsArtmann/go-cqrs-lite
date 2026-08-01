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
- **[Consistency Model](CONSISTENCY_MODEL.md)** — Read consistency guarantees, eventual consistency windows, and projection lag semantics across the CQRS pipeline
- **[Bundle Presets](PRESETS.md)** — Deployer-facing preset catalog (memory, SQLite, Pebble, Postgres, Turso)
- **[Infrastructure Recommendations](INFRASTRUCTURE_RECOMMENDATIONS.md)** — Which storage engine fits which CQRS concern, and why
- **[Storage Guide](STORAGE_GUIDE.md)** — SQL event store, snapshots, checkpoint stores, backend facade
- **[Signing Architecture](signing-architecture.md)** — Event signing with HMAC-SHA256 and Ed25519
- **[Error Taxonomy](error-taxonomy.md)** — 6-family error classification system (Rejection / Conflict / Transient / Infrastructure / Corruption / Orchestration)
- **[Domain Language](DOMAIN_LANGUAGE.md)** — Glossary of domain terms and ubiquitous language
- **[Turso Indexing Guidance](turso-indexing-guidance.md)** — Index management for Turso

## Modules

The authoritative module index with README links lives in the **[project README](../README.md)** (59 modules). Each module also has a `doc.go` rendered on [pkg.go.dev](https://pkg.go.dev/github.com/larsartmann/go-cqrs-lite/event/v4).

## Examples

| Example        | Demonstrates                                                         | README                                                |
| -------------- | -------------------------------------------------------------------- | ----------------------------------------------------- |
| **todo**       | Full app: HTTP API, decider, projections, queries, Pebble storage    | [example/todo](../example/todo/README.md)             |
| **user**       | Advanced: signing, middleware chains, catalog gen, tombstone/rebirth | [example/user](../example/user/README.md)             |
| **encryption** | Bus-level + store-level encryption, key rotation                     | [example/encryption](../example/encryption/README.md) |

## Architecture Decision Records (ADR)

78 ADRs documenting key architectural decisions (ADRs 0036 and 0041 were never assigned — gaps in numbering). Full text in [`adr/`](adr/); the [ADR index](adr/README.md) contains summaries.

| ADR                                                        | Title                                                   | Status                   |
| ---------------------------------------------------------- | ------------------------------------------------------- | ------------------------ |
| [0001](adr/0001-decider-over-aggregate.md)                 | Decider Pattern Over OO Aggregate                       | Accepted                 |
| [0002](adr/0002-error-taxonomy.md)                         | Error Taxonomy with Six Families                        | Accepted                 |
| [0003](adr/0003-multi-module-monorepo.md)                  | Multi-Module Monorepo Structure                         | Accepted                 |
| [0004](adr/0004-saga-process-manager.md)                   | Saga / Process Manager Module                           | Accepted                 |
| [0005](adr/0005-tombstone-soft-delete.md)                  | Tombstone Soft-Delete Pattern                           | Accepted                 |
| [0006](adr/0006-sink-source-split.md)                      | Sink/Source Split for Event Store Interface             | Accepted                 |
| [0007](adr/0007-gopls-workspace-workaround.md)             | gopls Multi-Module Workspace Workaround                 | Accepted                 |
| [0008](adr/0008-typed-handler-signature.md)                | TypedHandler Dual Type Parameters                       | Accepted                 |
| [0009](adr/0009-pebble-scope-event-store-only.md)          | Pebble Module Scope                                     | Accepted                 |
| [0010](adr/0010-remove-io-closer-from-interfaces.md)       | Remove io.Closer from Core Interfaces                   | Accepted                 |
| [0011](adr/0011-unify-err-dispatcher-closed.md)            | Unify ErrDispatcherClosed                               | Accepted                 |
| [0012](adr/0012-split-catalog-modules.md)                  | Split Catalog into Sub-Modules                          | Accepted                 |
| [0013](adr/0013-zero-copy-payload-for-decode.md)           | Zero-Copy Payload Access                                | Accepted                 |
| [0014](adr/0014-test-only-dependencies-in-go-mod.md)       | Test-Only Dependencies in go.mod                        | Accepted                 |
| [0015](adr/0015-cbor-codec.md)                             | CBOR Codec for Binary Payload Encoding                  | Accepted                 |
| [0016](adr/0016-outbox-pattern.md)                         | Outbox Pattern for Reliable Event Publishing            | Declined — use Watermill |
| [0017](adr/0017-schema-registry.md)                        | Schema Registry for Event Validation                    | Proposed                 |
| [0018](adr/0018-distributed-checkpointing.md)              | Distributed Checkpointing for Projections               | Proposed                 |
| [0019](adr/0019-cbor-envelope-format.md)                   | CBOR Envelope Format for Pebble Stores                  | Accepted                 |
| [0020](adr/0020-performance-optimization-patterns.md)      | Performance Optimization Patterns                       | Accepted                 |
| [0021](adr/0021-store-close-semantics.md)                  | Store Close() Semantics — Shared DB Pattern             | Accepted                 |
| [0022](adr/0022-kv-store-abstraction.md)                   | KV Store Abstraction Module                             | Accepted                 |
| [0023](adr/0023-pebble-kv-adapter.md)                      | Pebble KV Store Adapter                                 | Accepted                 |
| [0024](adr/0024-exported-id-markers.md)                    | Exported ID Marker Types                                | Accepted                 |
| [0025](adr/0025-transport-adapter-strategy.md)             | Transport Adapter Strategy                              | Accepted                 |
| [0026](adr/0026-experimental-features.md)                  | Experimental Features Behind Build Tags                 | Accepted                 |
| [0027](adr/0027-postgres-listen-notify-bus.md)             | Postgres LISTEN/NOTIFY Event Bus                        | Accepted                 |
| [0028](adr/0028-watermill-as-delivery-layer.md)            | Watermill as the Delivery Layer                         | Accepted                 |
| [0029](adr/0029-storage-consolidation.md)                  | Consolidate Storage Backends Under `storage/`           | Accepted                 |
| [0030](adr/0030-dissolve-projection.md)                    | Dissolve `projection/` into CatchUp + Materialize       | Accepted                 |
| [0031](adr/0031-metadata-split.md)                         | Typed Metadata Fields — Embed Tracing                   | Accepted                 |
| [0032](adr/0032-merge-readmodel-into-kv.md)                | Merge `readmodel/` into `kv/`                           | Accepted                 |
| [0033](adr/0033-multi-db-split.md)                         | Multi-Database Split for Concern Isolation              | Accepted                 |
| [0034](adr/0034-session-store-boundary.md)                 | Session Store Boundary                                  | Accepted                 |
| [0035](adr/0035-branded-dsn-types.md)                      | Branded DSN Types (Considered and Rejected)             | Rejected                 |
| [0037](adr/0037-projection-module-extraction.md)           | Projection Interface Extraction from event/             | Accepted                 |
| [0038](adr/0038-graph-projection-tier.md)                  | Graph Projection Tier (Writes Portable, Reads Native)   | Accepted                 |
| [0039](adr/0039-graph-schema.md)                           | Graph Schema — Boundary Typing for Graph Projections    | Accepted                 |
| [0040](adr/0040-deriver-design.md)                         | Deriver Module Design                                   | Accepted                 |
| [0042](adr/0042-pure-replay-dead-letters.md)               | Pure Replay for Dead-Letter Queue                       | Accepted                 |
| [0043](adr/0043-dlq-unification-options.md)                | Dead-Letter Store Unification Options                   | Accepted                 |
| [0044](adr/0044-blind-store-encoding-stamps.md)            | Blind Store Encoding Stamps                             | Accepted                 |
| [0045](adr/0045-eventtest-module-path-fix.md)              | eventtest Module Path / Directory Alignment             | Accepted                 |
| [0046](adr/0046-seven-tier-model.md)                       | Seven-Tier Dependency Model                             | Accepted                 |
| [0047](adr/0047-cose-support.md)                           | COSE Support for Signing, Encryption, and Codec         | Accepted                 |
| [0048](adr/0048-deterministic-json-encoding.md)            | Deterministic JSON Encoding in Security-Critical Paths  | Accepted                 |
| [0049](adr/0049-dispatch-time-middleware.md)               | Dispatch-Time Middleware Application                    | Accepted                 |
| [0050](adr/0050-envelope-json-fallback-keep-forever.md)    | Envelope JSON Fallback — Keep Forever                   | Accepted                 |
| [0051](adr/0051-cbor-as-default-codec.md)                  | CBOR as Default Codec for event.New()                   | Accepted                 |
| [0052](adr/0052-transport-boundary-codec-strategy.md)      | Transport Boundary Codec Strategy                       | Accepted                 |
| [0053](adr/0053-unified-codec-default-flip.md)             | Unified Codec Default Flip (JSON → CBOR)                | Accepted                 |
| [0054](adr/0054-json-v2-case-insensitive-decode.md)        | json/v2 Case-Insensitive Decode                         | Accepted                 |
| [0055](adr/0055-cqrs-lint-loader-error-surfacing.md)       | cqrs-lint Loader Error Surfacing                        | Accepted                 |
| [0056](adr/0056-timezone-safe-time-types.md)               | Timezone-Safe Time Types for Event Payloads             | Accepted                 |
| [0057](adr/0057-catalog-rest-openapi-operation-support.md) | Catalog REST/OpenAPI Operation Support                  | Accepted                 |
| [0058](adr/0058-rename-aggregate-to-stream.md)             | Rename Aggregate* to Stream*                            | Accepted                 |
| [0059](adr/0059-dlq-unification-proposal.md)               | DLQ Unification Proposal                                | Proposed                 |
| [0060](adr/0060-benchkit-design-decisions.md)              | Benchkit Design Decisions                               | Accepted                 |
| [0061](adr/0061-metaengine-sqlite-engine.md)               | Metaengine SQLite Engine                                | Accepted                 |
| [0062](adr/0062-metaengine-dependency-boundary.md)         | Metaengine Dependency Boundary (projectionadapter)      | Accepted                 |
| [0063](adr/0063-metaengine-pushdown.md)                    | FilterOn/SortOn Pushdown Strategy                       | Accepted                 |
| [0064](adr/0064-extract-retry-module.md)                   | Extract retry/ into Standalone go-retry Repository      | Proposed                 |
| [0065](adr/0065-extract-idempotency-module.md)             | Extract idempotency/ into go-idempotency Repository     | Proposed                 |
| [0066](adr/0066-metaengine-reify-fallback.md)              | Metaengine Cross-engine JSON Reification (ExecuteTyped) | Accepted                 |
| [0067](adr/0067-metaengine-tx-mapupdate.md)                | Metaengine Transaction-atomic MapUpdate (SQLite)        | Accepted                 |
| [0068](adr/0068-metaengine-multimap-seq-seed.md)           | Metaengine Multimap seq-seed (sync.Once MAX(seq))       | Accepted                 |
| [0069](adr/0069-error-wrapping-helpers.md)                 | Error-Wrapping Helper Convention                        | Accepted                 |
| [0070](adr/0070-transform-fallback-observability.md)       | Transform Fallback Observability (slog vs OTel)         | Accepted                 |
| [0071](adr/0071-duckdb-cgo-introduction.md)                | DuckDB CGo Introduction                                 | Accepted                 |
| [0072](adr/0072-metaengine-pushdown.md)                    | Metaengine Pushdown (json_extract SQL pushdown)         | Accepted                 |
| [0073](adr/0073-metaengine-layout-planning.md)             | Metaengine Layout Planning (deployment-time DDL)        | Accepted                 |
| [0074](adr/0074-pebble-engine.md)                          | Pebble Metaengine (cost profile & slices.Backward)      | Accepted                 |
| [0075](adr/0075-metaengine-adttest-extraction.md)          | ADT Test Harness Extraction (cross-engine parity)       | Accepted                 |
| [0076](adr/0076-pebble-raw-readers.md)                     | Pebble Raw Value Readers (single-pass JSON decode)      | Accepted                 |
| [0077](adr/0077-metaengine-graph-reconciliation.md)        | Metaengine GraphBackend vs graph/ module reconciliation | Accepted                 |
| [0078](adr/0078-metaengine-kv-coexistence.md)              | Metaengine and kv.ViewStore coexistence                 | Accepted                 |
| [0079](adr/0079-sse-consolidation.md)                      | SSE consolidation — two implementations, two layers     | Accepted                 |
| [0080](adr/0080-dialect-interface-upsert-methods.md)       | Dialect interface expansion for cross-database upsert   | Accepted                 |
| [0081](adr/0081-metaengine-runtime-casts.md)               | Why Metaengine Uses Runtime Casts                       | Accepted                 |
| [0082](adr/0082-metaengine-store-redesign-analysis.md)     | Metaengine Store Redesign Analysis (eliminate casts)    | Analysis                 |
| [0083](adr/0083-metaengine-planner-rule-pipeline.md)       | Metaengine planner rule pipeline (composable PlanRule)  | Accepted                 |
| [0084](adr/0084-metaengine-layered-architecture.md)        | Metaengine layered architecture (StorageLayout, costs)  | Accepted                 |
| [0085](adr/0085-metaengine-new-adts.md)                    | Metaengine new ADTs (Vector, Search, Spatial)           | Accepted                 |
| [0086](adr/0086-metaengine-duckdb-engine.md)               | DuckDB metaengine engine (columnar OLAP, CGo)           | Accepted                 |
| [0087](adr/0087-metaengine-postgres-engine.md)             | Postgres metaengine engine (JSONB + B-tree)             | Accepted                 |
| [0088](adr/0088-block-level-suppression.md)                | Block-level suppression for cqrs-lint                   | Accepted                 |
| [0089](adr/0089-flight-recorder.md)                        | Flight Recorder integration                             | Accepted                 |
| [0090](adr/0090-benchkit-evidence-metrics.md)              | Benchkit evidence-grade metrics                         | Accepted                 |

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
