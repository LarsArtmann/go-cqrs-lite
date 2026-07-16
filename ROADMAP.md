# Roadmap — go-cqrs-lite

> Where we are, where we're going, and what's next.
> **Last updated:** 2026-07-16

---

## Current State (v4.0.0 shipped)

**v4.0.0 is tagged** (2026-07-11) — all 52 `go.mod` files on `/v4` import paths
(verify: `find . -name go.mod -not -path './vendor/*' | wc -l`). The library
covers the full CQRS/ES lifecycle: event sourcing with branded IDs, command/query
dispatch, pure-function deciders, three projection tiers (document/KV,
relational/SQL, graph), durable deadline scheduling, event→command derivation,
dead-letter quarantine, managed projection hosting, event signing/encryption,
OTel tracing/metrics, auto-documentation generation, and a domain-aware linter
(cqrs-lint, 60 rules).

See [CHANGELOG.md](CHANGELOG.md) `[4.0.0]` for the full v4 release notes and
[docs/migration/MIGRATION-GUIDE.md](docs/migration/MIGRATION-GUIDE.md) for
migration steps.

---

## Release History

| Version | Date       | Highlights                                                                                                                                          |
| ------- | ---------- | --------------------------------------------------------------------------------------------------------------------------------------------------- |
| v4.0.1  | 2026-07-16 | Patch: projectionhost deadlock/leak/sort fix, watermill deadlock fix, storage/view IS NULL+RawWhere+ViewUpdater, cqrs-lint first release (60 rules) |
| v4.0.0  | 2026-07-11 | CBOR defaults, API cleanup, BackfillHandler consolidation, HealthCheck, storage split, `/v4` path migration                                         |
| v3.6.0  | 2026-07-05 | Error taxonomy, deriver module, DOMAIN_LANGUAGE rebuild                                                                                             |
| v3.5.0  | 2026-06-29 | Idempotency, dispatch middleware, scenario DSL, scheduling, projectionhost                                                                          |
| v3.3.0  | 2026-06-28 | Three projection tiers, Watermill command bridge                                                                                                    |
| v3.1.0  | 2026-06-25 | SQL-backed view stores, multi-database split                                                                                                        |
| v3.0.0  | 2026-06-22 | 11 breaking changes — see [V3 Migration Guide](docs/migration/V3_MIGRATION.md)                                                                      |

---

## Themes

### 1. Consumer Experience

Making it trivially easy for new consumers to adopt the library.

- **Publish `eventtest` to the Go proxy** — the #1 consumer pain point across
  all feedback rounds. Tag exists locally (`v0.1.0`) but is not pushed.
- **README "sales page" rewrite** — the README has grown into internal
  documentation. Per the docs-health model, it should be the end-user entry
  point: what this does, why it exists, how to get started in 3 steps.
- **Pre-commit hooks** — `fmt.Printf` ban in production packages,
  `api_surface.txt` regeneration check, `nix fmt --fail-on-change`.
- **CBOR-stamp cross-encoding tests** — gRPC and watermill transports lack
  round-trip tests proving CBOR-stamped events survive transport.

### 2. Public Release Readiness

Preparing the library for broader adoption.

- **License swap** (PROPRIETARY → Apache-2.0) — hard blocker for public
  adoption. Irreversible; needs explicit user approval.
- **Git history scrub** — internal strategy docs in git history. Irreversible;
  needs explicit user approval.
- **Postgres CI coverage** — `stack/postgres` shows 0% coverage locally
  (tests skip without `POSTGRES_TEST_DSN`). Either add a CI Postgres service
  or label the module experimental.

### 3. Ecosystem Expansion

New modules and capabilities that extend the library's reach.

- **Parquet journal** (`storage/parquet`) — columnar, compressed, cloud-native
  event archival. Pure Go (no CGO). Design complete, three additive phases.
- **DuckDB connector** (`storage/duckdb`) — OLAP-grade analytical materializations
  over Parquet or relational data. Requires CGO. Unlocks `SQLViewStore` +
  `RelationalProjection` for analytics.
- **Lakehouse preset** (`stack/duckdb`) — DuckDB materializations + Parquet
  journal. The "lakehouse for events" pattern.
- **NATS/ValKey stream adapter** — ADR-0025 accepted. Broker plugins for
  `watermill.NewCommandBus` via publisher/subscriber adapters.
- **Distributed event bus** — multi-process backend for event distribution.

### 4. Codebase Health

Keeping the library clean and trustworthy.

- **Deprecated API removal batch 2** — 9 deprecated items in middleware,
  catalog, and storage. Breaking change → v4.1 cut.
- **cqrs-lint hardening** — extend source snippets to all detectors, add
  property-based tests, improve scanner accuracy for edge cases.

---

## Raw Ideas (No Design Yet)

- Event stream compaction / log truncation strategies
- Multi-tenant event store (schema-per-tenant)
- Distributed projection runner (leader election, multi-node coordination)
- Event archival to S3 / GCS / Azure Blob
- CQRS-lite dashboard (web UI for inspecting aggregates, events, projections)
- Automatic migration generator for schema evolution
- Property-based integration testing with state machine verification
- Performance regression dashboard (historical benchmark tracking)
- Neo4j/Memgraph graph driver (`graph/neo4j/`) — consumer-pulled sibling module
- Hierarchical error taxonomy adoption (buildflow finding — needs evaluation)

---

## Non-Goals

- **Framework opinions** — the library will never mandate a transport, message
  broker, or SQL driver. Consumers compose their own stack.
- **Splitting the `event/` module** — 27 importers, real cohesion. Explicitly
  decided in v4. Do not split.
- **ORM features** — no query builder, no ORM-style relations, no lazy loading.
  `RawWhere` escape hatch covers the 5% case. Principle: "Library, not framework."
- **Redis adapter** — the author is not a fan of Redis. ValKey (the LF-backed
  fork) is the recommended alternative. If starting fresh, pick ValKey, NATS,
  or Kafka instead.

---

## Experimental / Go-stdlib-blocked

- **Remove `goexperiment.jsonv2` tag** — JSON v2 is fully adopted (~25 production
  files). The tag remains only because Go 1.26 hasn't graduated json/v2 from
  experimental. Remove when Go stabilizes it (expected Go 1.27+).
- **Turso MVCC concurrent-write support** — blocked on upstream experimental MVCC.
