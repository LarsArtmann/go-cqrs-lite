# Architecture Decision Records

## ADR-0001: Decider Pattern over Aggregate Root

**Date:** 2026-05-03
**Status:** Accepted
**Supersedes:** Traditional DDD Aggregate Root pattern (see also [ADR-0058](0058-rename-aggregate-to-stream.md) for the identity-type rename)

### Context

The traditional DDD Aggregate Root pattern requires a 9-method interface, mutable state, and tight coupling between infrastructure and domain logic. This makes testing difficult and creates unnecessary abstraction layers.

### Decision

Adopt the **Decider pattern** — a pure-function approach where:

- `Decider[State]` holds `Initial` state and `Fold func(State, Event) (State, error)`
- `DecideFunc` takes a command and state, returns events (pure function)
- `Repository[State].Execute` does load → fold → decide → save → publish
- No mutable state, no 9-method interface, zero-infrastructure testing

### Consequences

- **+** Pure functions are trivially testable without mocks
- **+** Single generic `Repository[State]` replaces per-aggregate repositories
- **+** `ExecuteWithResult` returns typed results alongside events
- **-** Consumers must think in terms of state + fold, not OOP-style objects
- The deprecated `core/aggregate` package was removed in Session 99

---

## ADR-0002: Error Taxonomy via go-error-family

**Date:** 2026-05-03
**Status:** Accepted

### Context

Without classified errors, consumers cannot distinguish transient failures from business rule violations from data corruption. This leads to blanket retry strategies that mask real problems.

### Decision

Adopt a **5-family error taxonomy** via `go-error-family`:

| Family             | Constructor         | Meaning                      | Retry? |
| ------------------ | ------------------- | ---------------------------- | ------ |
| **Rejection**      | `NewRejection`      | Business rule violation      | No     |
| **Conflict**       | `NewConflict`       | Concurrency/version conflict | No     |
| **Transient**      | `NewTransient`      | Temporary failure            | Yes    |
| **Infrastructure** | `NewInfrastructure` | System-level failure         | Maybe  |
| **Corruption**     | `NewCorruption`     | Data integrity violation     | No     |

Each family has `Is*()` predicates and `Wrap*()` wrapping functions. Middleware uses `event.IsRetryable(err)` to decide retry behavior.

### Consequences

- **+** Middleware can make informed retry/recovery decisions
- **+** Error classification is explicit, not heuristic-based
- **+** Hundreds of sentinel errors classified across all modules (count has grown significantly since this ADR's writing)
- **-** Requires `init()` registration for cross-package sentinels (TODO: explicit setup)

---

## ADR-0003: Multi-Module Monorepo with go.work

**Date:** 2026-04-24
**Status:** Accepted

### Context

A single `go.mod` creates tight coupling between all packages. Changes to the catalog module require re-testing storage, even though they're unrelated. External consumers can't import individual packages.

### Decision

Split into independent workspace modules with `go.mod` files tied together by `go.work`. (Originally 9 modules at this ADR's writing; now 55. This ADR is historical — see AGENTS.md for the current module list.)

```
core/          — Zero external deps (ulid, branded-id, error-family)
memory/        — In-memory test implementations
catalog/       — Documentation generation (AsyncAPI, D2, EventCatalog, OpenAPI)
middleware/    — Cross-cutting CQRS middleware
testhelpers/   — Shared test utilities
projection/    — Projection runner with replay
storage/       — SQL + Pebble backends
saga/          — Long-running process orchestration (later removed — ADR-0004)
watermill/     — Watermill protocol adapter
```

> **Historical:** This was the original 9-module layout. The project has since grown to 55 modules. See `AGENTS.md` for the current structure.

### Consequences

- **+** Each module has isolated dependencies
- **+** External consumers import only what they need
- **+** CI can test modules in parallel and in isolation (`GOWORK=off`)
- **-** `replace` directives required until v1.0.0 tags are pushed to remote
- **-** Version management across 55 modules requires discipline
- **-** `golangci-lint` doesn't work well with `go.work` (pre-existing tooling issue)

---

## Index

| ADR                                                    | Title                                                   | Date       | Status                                                                                                                        |
| ------------------------------------------------------ | ------------------------------------------------------- | ---------- | ----------------------------------------------------------------------------------------------------------------------------- |
| [0001](0001-decider-over-aggregate.md)                 | Decider Pattern over Aggregate Root                     | 2026-05-03 | Accepted                                                                                                                      |
| [0002](0002-error-taxonomy.md)                         | Error Taxonomy via go-error-family                      | 2026-05-03 | Accepted                                                                                                                      |
| [0003](0003-multi-module-monorepo.md)                  | Multi-Module Monorepo with go.work                      | 2026-05-03 | Accepted (historical — module count stale)                                                                                    |
| [0004](0004-saga-process-manager.md)                   | Saga/Process Manager Pattern                            | 2026-05-26 | Superseded — removed in Session 146                                                                                           |
| [0005](0005-tombstone-soft-delete.md)                  | Tombstone Soft-Delete Pattern                           | 2026-05-10 | Accepted                                                                                                                      |
| [0006](0006-sink-source-split.md)                      | Sink/Source ISP Split                                   | 2026-05-29 | Accepted                                                                                                                      |
| [0007](0007-gopls-workspace-workaround.md)             | gopls Workspace Workaround                              | —          | Accepted                                                                                                                      |
| [0008](0008-typed-handler-signature.md)                | Typed Handler Signature                                 | —          | Accepted                                                                                                                      |
| [0009](0009-pebble-scope-event-store-only.md)          | Pebble Scope: Event Store Only                          | 2026-05-29 | Superseded by [0019](0019-cbor-envelope-format.md) — pebble now implements Journal                                            |
| [0010](0010-remove-io-closer-from-interfaces.md)       | Remove io.Closer from Core Interfaces                   | —          | Implemented (v3) — io.Closer retained, no Lifecycle type                                                                      |
| [0011](0011-unify-err-dispatcher-closed.md)            | Unify ErrDispatcherClosed                               | —          | Proposed                                                                                                                      |
| [0012](0012-split-catalog-modules.md)                  | Split Catalog into Sub-Modules                          | —          | Proposed                                                                                                                      |
| [0013](0013-zero-copy-payload-for-decode.md)           | Zero-Copy Payload for Decode                            | 2026-06-09 | Accepted                                                                                                                      |
| [0014](0014-test-only-dependencies-in-go-mod.md)       | Test-Only Dependencies in go.mod                        | 2026-06-10 | Superseded by [0045](0045-eventtest-module-path-fix.md)                                                                       |
| [0015](0015-cbor-codec.md)                             | CBOR Codec                                              | 2026-06-11 | Superseded by [0051](0051-cbor-as-default-codec.md)/[0053](0053-unified-codec-default-flip.md) — JSON default flipped to CBOR |
| [0016](0016-outbox-pattern.md)                         | Outbox Pattern                                          | 2026-06-14 | Declined — use Watermill                                                                                                      |
| [0017](0017-schema-registry.md)                        | Schema Registry for Event Validation                    | 2026-06-14 | Accepted                                                                                                                      |
| [0018](0018-distributed-checkpointing.md)              | Distributed Checkpointing                               | 2026-06-14 | Accepted                                                                                                                      |
| [0019](0019-cbor-envelope-format.md)                   | CBOR Envelope Format for Pebble                         | 2026-06-16 | Accepted                                                                                                                      |
| [0020](0020-performance-optimization-patterns.md)      | Performance Optimization Patterns                       | 2026-06-14 | Accepted                                                                                                                      |
| [0021](0021-store-close-semantics.md)                  | Store Close() Semantics                                 | 2026-06-16 | Accepted                                                                                                                      |
| [0022](0022-kv-store-abstraction.md)                   | KV Store Abstraction Module                             | 2026-06-16 | Accepted                                                                                                                      |
| [0023](0023-pebble-kv-adapter.md)                      | Pebble KV Store Adapter                                 | 2026-06-17 | Accepted                                                                                                                      |
| [0024](0024-exported-id-markers.md)                    | Exported ID Marker Types                                | 2026-06-18 | Accepted                                                                                                                      |
| [0025](0025-transport-adapter-strategy.md)             | Transport Adapter Strategy                              | 2026-06-19 | Accepted                                                                                                                      |
| [0026](0026-experimental-features.md)                  | Experimental Features Policy                            | 2026-06-19 | Accepted                                                                                                                      |
| [0027](0027-postgres-listen-notify-bus.md)             | Postgres LISTEN/NOTIFY Bus                              | 2026-06-19 | Deprecated — superseded by [0028](0028-watermill-as-delivery-layer.md) for new code                                           |
| [0028](0028-watermill-as-delivery-layer.md)            | Watermill as the Delivery Layer                         | 2026-06-20 | Accepted (v3 removal of ghost buses not yet executed)                                                                         |
| [0029](0029-storage-consolidation.md)                  | Consolidate Storage Backends                            | 2026-06-20 | Accepted                                                                                                                      |
| [0030](0030-dissolve-projection.md)                    | Dissolve projection/ into CatchUp + Materialize         | 2026-06-20 | Accepted (projection/ name reused by [0037](0037-projection-module-extraction.md))                                            |
| [0031](0031-metadata-split.md)                         | Kill Metadata Aliases, Add Typed Fields                 | 2026-06-20 | Implemented (v3) — aliases still exist, repointed to metadata.CustomData                                                      |
| [0032](0032-merge-readmodel-into-kv.md)                | Merge readmodel/ into kv/                               | 2026-06-20 | Accepted                                                                                                                      |
| [0033](0033-multi-db-split.md)                         | Multi-Database Split for Concern Isolation              | 2026-06-23 | Accepted                                                                                                                      |
| [0034](0034-session-store-boundary.md)                 | Session Store Boundary                                  | 2026-06-23 | Accepted                                                                                                                      |
| [0035](0035-branded-dsn-types.md)                      | Branded DSN Types (Considered and Rejected)             | 2026-06-24 | Rejected                                                                                                                      |
| [0037](0037-projection-module-extraction.md)           | Projection Interface Extraction from event/             | 2026-06-28 | Accepted                                                                                                                      |
| [0038](0038-graph-projection-tier.md)                  | Graph Projection Tier — Writes Portable, Reads Native   | 2026-06-28 | Accepted                                                                                                                      |
| [0039](0039-graph-schema.md)                           | Graph Schema — Boundary Typing for Graph Projections    | 2026-06-28 | Accepted                                                                                                                      |
| [0040](0040-deriver-design.md)                         | Deriver Module Design                                   | 2026-06-28 | Accepted (implemented: `deriver/` module)                                                                                     |
| [0042](0042-pure-replay-dead-letters.md)               | Pure Replay for Dead-Letter Queue                       | 2026-06-29 | Accepted                                                                                                                      |
| [0043](0043-dlq-unification-options.md)                | Dead-Letter Store Unification Options                   | 2026-06-29 | Accepted — see also [0059](0059-dlq-unification-proposal.md) (Proposed)                                                       |
| [0044](0044-blind-store-encoding-stamps.md)            | Blind Store Encoding Stamps                             | 2026-07-01 | Accepted                                                                                                                      |
| [0045](0045-eventtest-module-path-fix.md)              | eventtest Module Path / Directory Alignment             | 2026-07-05 | Accepted                                                                                                                      |
| [0046](0046-seven-tier-model.md)                       | Seven-Tier Dependency Model                             | 2026-07-09 | Accepted                                                                                                                      |
| [0047](0047-cose-support.md)                           | COSE Support for Signing, Encryption, and Codec         | 2026-07-10 | Accepted                                                                                                                      |
| [0048](0048-deterministic-json-encoding.md)            | Deterministic JSON Encoding in Security-Critical Paths  | 2026-07-10 | Accepted                                                                                                                      |
| [0049](0049-dispatch-time-middleware.md)               | Dispatch-Time Middleware Application                    | 2026-07-10 | Accepted                                                                                                                      |
| [0050](0050-envelope-json-fallback-keep-forever.md)    | Envelope JSON Fallback — Keep Forever                   | 2026-07-10 | Accepted                                                                                                                      |
| [0051](0051-cbor-as-default-codec.md)                  | CBOR as Default Codec for event.New()                   | 2026-07-11 | Accepted                                                                                                                      |
| [0052](0052-transport-boundary-codec-strategy.md)      | Transport Boundary Codec Strategy                       | 2026-07-11 | Accepted                                                                                                                      |
| [0053](0053-unified-codec-default-flip.md)             | Unified Codec Default Flip (JSON → CBOR)                | 2026-07-11 | Accepted                                                                                                                      |
| [0054](0054-json-v2-case-insensitive-decode.md)        | json/v2 Case-Insensitive Decode                         | 2026-07-10 | Accepted                                                                                                                      |
| [0055](0055-cqrs-lint-loader-error-surfacing.md)       | cqrs-lint Loader Error Surfacing                        | 2026-07-17 | Accepted                                                                                                                      |
| [0056](0056-timezone-safe-time-types.md)               | Timezone-Safe Time Types for Event Payloads             | 2026-07-18 | Accepted                                                                                                                      |
| [0057](0057-catalog-rest-openapi-operation-support.md) | Catalog REST/OpenAPI Operation Support                  | 2026-07-18 | Accepted                                                                                                                      |
| [0058](0058-rename-aggregate-to-stream.md)             | Rename Aggregate* to Stream*                            | 2026-07-23 | Accepted                                                                                                                      |
| [0059](0059-dlq-unification-proposal.md)               | DLQ Unification Proposal                                | 2026-07-23 | Proposed                                                                                                                      |
| [0060](0060-benchkit-design-decisions.md)              | Benchkit Design Decisions                               | 2026-07-24 | Accepted                                                                                                                      |
| [0061](0061-metaengine-sqlite-engine.md)               | Metaengine SQLite Engine                                | 2026-07-25 | Accepted                                                                                                                      |
| [0062](0062-metaengine-dependency-boundary.md)         | Metaengine Dependency Boundary (projectionadapter)      | 2026-07-25 | Accepted                                                                                                                      |
| [0063](0063-metaengine-pushdown.md)                    | FilterOn/SortOn Pushdown Strategy                       | 2026-07-25 | Accepted                                                                                                                      |
| [0064](0064-extract-retry-module.md)                   | Extract `retry/` into Standalone `go-retry` Repository  | 2026-07-25 | Proposed                                                                                                                      |
| [0065](0065-extract-idempotency-module.md)             | Extract `idempotency/` into `go-idempotency` Repository | 2026-07-25 | Proposed                                                                                                                      |
| [0066](0066-metaengine-reify-fallback.md)              | Metaengine Cross-engine JSON Reification (ExecuteTyped) | 2026-07-25 | Accepted                                                                                                                      |
| [0067](0067-metaengine-tx-mapupdate.md)                | Metaengine Transaction-atomic MapUpdate (SQLite)        | 2026-07-25 | Accepted                                                                                                                      |
| [0068](0068-metaengine-multimap-seq-seed.md)           | Metaengine Multimap seq-seed (sync.Once MAX(seq))       | 2026-07-25 | Accepted                                                                                                                      |
| [0069](0069-error-wrapping-helpers.md)                 | Error-Wrapping Helper Convention                        | 2026-07-26 | Accepted                                                                                                                      |

> **Note:** ADRs 0036 and 0041 were never assigned (gaps in numbering).
