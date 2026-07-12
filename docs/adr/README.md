# Architecture Decision Records

## ADR-0001: Decider Pattern over Aggregate Root

**Date:** 2026-04-29
**Status:** Accepted
**Supersedes:** Traditional DDD Aggregate Root pattern

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
- **+** 38 sentinel errors classified across all modules
- **-** Requires `init()` registration for cross-package sentinels (TODO: explicit setup)

---

## ADR-0003: Multi-Module Monorepo with go.work

**Date:** 2026-04-24
**Status:** Accepted

### Context

A single `go.mod` creates tight coupling between all packages. Changes to the catalog module require re-testing storage, even though they're unrelated. External consumers can't import individual packages.

### Decision

Split into **16 workspace modules** with independent `go.mod` files, tied together by `go.work`:

```
core/          — Zero external deps (ulid, branded-id, error-family)
memory/        — In-memory test implementations
catalog/       — Documentation generation (AsyncAPI, D2, EventCatalog, OpenAPI)
middleware/    — Cross-cutting CQRS middleware
testhelpers/   — Shared test utilities
projection/    — Projection runner with replay
storage/       — SQL + Pebble backends
saga/          — Long-running process orchestration
watermill/     — Watermill protocol adapter
```

### Consequences

- **+** Each module has isolated dependencies
- **+** External consumers import only what they need
- **+** CI can test modules in parallel and in isolation (`GOWORK=off`)
- **-** `replace` directives required until v1.0.0 tags are pushed to remote
- **-** Version management across 16 modules requires discipline
- **-** `golangci-lint` doesn't work well with `go.work` (pre-existing tooling issue)

---

## Index

| ADR                                                 | Title                                                  | Date       | Status                   |
| --------------------------------------------------- | ------------------------------------------------------ | ---------- | ------------------------ |
| [0001](0001-decider-over-aggregate.md)              | Decider Pattern over Aggregate Root                    | 2026-04-29 | Accepted                 |
| [0002](0002-error-taxonomy.md)                      | Error Taxonomy via go-error-family                     | 2026-05-03 | Accepted                 |
| [0003](0003-multi-module-monorepo.md)               | Multi-Module Monorepo with go.work                     | 2026-04-24 | Accepted                 |
| [0004](0004-saga-process-manager.md)                | Saga/Process Manager Pattern                           | —          | Accepted                 |
| [0005](0005-tombstone-soft-delete.md)               | Tombstone Soft-Delete Pattern                          | 2026-05-10 | Accepted                 |
| [0006](0006-sink-source-split.md)                   | Sink/Source ISP Split                                  | —          | Accepted                 |
| [0007](0007-gopls-workspace-workaround.md)          | gopls Workspace Workaround                             | —          | Accepted                 |
| [0008](0008-typed-handler-signature.md)             | Typed Handler Signature                                | —          | Accepted                 |
| [0009](0009-pebble-scope-event-store-only.md)       | Pebble Scope: Event Store Only                         | —          | Accepted                 |
| [0010](0010-remove-io-closer-from-interfaces.md)    | Remove io.Closer from Core Interfaces                  | —          | Accepted                 |
| [0011](0011-unify-err-dispatcher-closed.md)         | Unify ErrDispatcherClosed                              | —          | Accepted                 |
| [0012](0012-split-catalog-modules.md)               | Split Catalog into Sub-Modules                         | —          | Accepted                 |
| [0013](0013-zero-copy-payload-for-decode.md)        | Zero-Copy Payload for Decode                           | —          | Accepted                 |
| [0014](0014-test-only-dependencies-in-go-mod.md)    | Test-Only Dependencies in go.mod                       | —          | Accepted                 |
| [0015](0015-cbor-codec.md)                          | CBOR Codec                                             | —          | Accepted                 |
| [0016](0016-outbox-pattern.md)                      | Outbox Pattern                                         | 2026-06-14 | Declined — use Watermill |
| [0017](0017-schema-registry.md)                     | Schema Registry for Event Validation                   | 2026-06-14 | Proposed                 |
| [0018](0018-distributed-checkpointing.md)           | Distributed Checkpointing                              | 2026-06-14 | Proposed                 |
| [0019](0019-cbor-envelope-format.md)                | CBOR Envelope Format for Pebble                        | 2026-06-16 | Accepted                 |
| [0020](0020-performance-optimization-patterns.md)   | Performance Optimization Patterns                      | 2026-06-14 | Accepted                 |
| [0021](0021-store-close-semantics.md)               | Store Close() Semantics                                | 2026-06-16 | Accepted                 |
| [0022](0022-kv-store-abstraction.md)                | KV Store Abstraction Module                            | 2026-06-16 | Accepted                 |
| [0023](0023-pebble-kv-adapter.md)                   | Pebble KV Store Adapter                                | 2026-06-17 | Accepted                 |
| [0024](0024-exported-id-markers.md)                 | Exported ID Marker Types                               | 2026-06-18 | Accepted                 |
| [0025](0025-transport-adapter-strategy.md)          | Transport Adapter Strategy                             | 2026-06-19 | Accepted                 |
| [0026](0026-experimental-features.md)               | Experimental Features Policy                           | 2026-06-19 | Accepted                 |
| [0027](0027-postgres-listen-notify-bus.md)          | Postgres LISTEN/NOTIFY Bus                             | 2026-06-19 | Implemented              |
| [0028](0028-watermill-as-delivery-layer.md)         | Watermill as the Delivery Layer                        | 2026-06-20 | Accepted                 |
| [0029](0029-storage-consolidation.md)               | Consolidate Storage Backends                           | 2026-06-20 | Accepted                 |
| [0030](0030-dissolve-projection.md)                 | Dissolve projection/ into CatchUp + Materialize        | 2026-06-20 | Accepted                 |
| [0031](0031-metadata-split.md)                      | Kill Metadata Aliases, Add Typed Fields                | 2026-06-20 | Accepted                 |
| [0032](0032-merge-readmodel-into-kv.md)             | Merge readmodel/ into kv/                              | 2026-06-20 | Accepted                 |
| [0033](0033-multi-db-split.md)                      | Multi-Database Split for Concern Isolation             | 2026-06-23 | Accepted                 |
| [0034](0034-session-store-boundary.md)              | Session Store Boundary                                 | 2026-06-23 | Accepted                 |
| [0035](0035-branded-dsn-types.md)                   | Branded DSN Types (Considered and Rejected)            | 2026-06-24 | Rejected                 |
| [0037](0037-projection-module-extraction.md)        | Projection Interface Extraction from event/            | —          | Accepted                 |
| [0038](0038-graph-projection-tier.md)               | Graph Projection Tier — Writes Portable, Reads Native  | —          | Accepted                 |
| [0039](0039-graph-schema.md)                        | Graph Schema — Boundary Typing for Graph Projections   | —          | Accepted                 |
| [0040](0040-deriver-design.md)                      | Deriver Module Design (TypeDB Rule Model Reference)    | —          | Accepted                 |
| [0042](0042-pure-replay-dead-letters.md)            | Pure Replay for Dead-Letter Queue                      | —          | Accepted                 |
| [0043](0043-dlq-unification-options.md)             | Dead-Letter Store Unification Options                  | —          | Accepted                 |
| [0044](0044-blind-store-encoding-stamps.md)         | Blind Store Encoding Stamps                            | 2026-07-01 | Accepted                 |
| [0045](0045-eventtest-module-path-fix.md)           | eventtest Module Path / Directory Alignment            | 2026-07-05 | Accepted                 |
| [0046](0046-four-tier-model.md)                     | Four-Tier Dependency Model                             | 2026-07-09 | Accepted                 |
| [0047](0047-cose-support.md)                        | COSE Support for Signing, Encryption, and Codec        | 2026-07-10 | Accepted                 |
| [0047](0047-json-v2-case-insensitive-decode.md)     | json/v2 Case-Insensitive Decode                        | 2026-07-10 | Accepted                 |
| [0048](0048-deterministic-json-encoding.md)         | Deterministic JSON Encoding in Security-Critical Paths | 2026-07-10 | Accepted                 |
| [0049](0049-dispatch-time-middleware.md)            | Dispatch-Time Middleware Application                   | 2026-07-10 | Accepted                 |
| [0050](0050-envelope-json-fallback-keep-forever.md) | Envelope JSON Fallback — Keep Forever                  | 2026-07-10 | Accepted                 |
| [0051](0051-cbor-as-default-codec.md)               | CBOR as Default Codec for event.New()                  | 2026-07-11 | Accepted                 |
| [0052](0052-transport-boundary-codec-strategy.md)   | Transport Boundary Codec Strategy                      | —          | Accepted                 |
| [0053](0053-unified-codec-default-flip.md)          | Unified Codec Default Flip (JSON → CBOR)               | 2026-07-11 | Accepted                 |

> **Note:** ADRs 0036 and 0041 were never assigned (gaps in numbering). Two
> ADRs share number 0047 (COSE Support and json/v2 Case-Insensitive Decode) —
> this is a pre-existing numbering conflict.
