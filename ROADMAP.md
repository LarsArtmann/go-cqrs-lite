# Roadmap — go-cqrs-lite

> Where we are, where we're going, and what's next.
> **Last updated:** 2026-07-24

---

## Current State (v4.1.0 shipped)

**v4.1.0 tagged** (2026-07-23) — 49 modules on `/v4` import paths (verify:
`git tag --list '*/v4.1.0' | wc -l`). The deprecated API removal batch shipped
(`middleware.NewMetrics`, `catalog.ErrorExporter`, `storage/sql.NewOwnedDBHandle`,
etc. — see [CHANGELOG.md](CHANGELOG.md) `[v4.1.0]` equivalent under `[Unreleased]`).

The library covers the full CQRS/ES lifecycle: event sourcing with branded IDs,
command/query dispatch, pure-function deciders, three projection tiers (document/KV,
relational/SQL, graph), durable deadline scheduling, event→command derivation,
dead-letter quarantine, managed projection hosting, event signing/encryption,
OTel tracing/metrics, auto-documentation generation, and a domain-aware linter
(cqrs-lint, 60 rules).

**New since v4.0.0:**

- **Metaengine** (`metaengine/v4`) — cost-based storage planner. Derives projections
  and engine assignments from two primitives (Events + Queries). 7 ADTs inferred
  from fold return types. MemoryEngine only; no real SQL/Pebble engine yet.
- **Benchkit** (`benchkit/v4` + `cmd/cqrs-bench`) — benchmarking toolkit with
  7 named workload profiles, 8-phase runner, structured reports.
- **Incremental rollups** — `ProjectionSink.Increment` + `RelationalProjection.Reset`
  for atomic counter maintenance in relational projections.
- **Aggregate→Stream rename** (ADR-0058) — type aliases + deprecated wrappers.
  Structurally complete; comment cleanup and 2 error var pairs remaining.
- **Comprehensive README coverage** — all 56 modules with READMEs, 248 Go symbol
  references verified by `doc-check`.
- **Error taxonomy migration** — 13 sentinels migrated to `errorfamily` constructors.

56 `go.mod` files total (verify: `find . -name go.mod -not -path './vendor/*' | wc -l`).

---

## Release History

| Version | Date       | Highlights                                                                                                                                          |
| ------- | ---------- | --------------------------------------------------------------------------------------------------------------------------------------------------- |
| v4.1.0  | 2026-07-23 | Deprecated API removal, metaengine, benchkit, Increment/Reset rollups, README overhaul, error taxonomy migration, Aggregate→Stream rename (ADR-0058) |
| v4.0.4  | 2026-07-23 | COSE signing/encryption, multi-batch event store, OTel storage instrumentation, getting-started guide, architecture docs                            |
| v4.0.3  | 2026-07-22 | SQL dialect abstraction, stack preset centralization, JSON v2 migration, harmful duplication elimination, cqrs-lint scanner overhaul                 |
| v4.0.2  | 2026-07-18 | CBOR time encoding fix, timezone-safe types (Instant, WallTime, Date), cqrs-lint loader error surfacing                                              |
| v4.0.1  | 2026-07-16 | projectionhost deadlock/leak/sort fix, watermill deadlock fix, storage/view IS NULL+RawWhere+ViewUpdater, cqrs-lint first release (60 rules)        |
| v4.0.0  | 2026-07-11 | CBOR defaults, API cleanup, BackfillHandler consolidation, HealthCheck, storage split, `/v4` path migration                                         |
| v3.6.0  | 2026-07-05 | Error taxonomy, deriver module, DOMAIN_LANGUAGE rebuild                                                                                             |
| v3.5.0  | 2026-06-29 | Idempotency, dispatch middleware, scenario DSL, scheduling, projectionhost                                                                          |
| v3.3.0  | 2026-06-28 | Three projection tiers, Watermill command bridge                                                                                                    |
| v3.1.0  | 2026-06-25 | SQL-backed view stores, multi-database split                                                                                                        |
| v3.0.0  | 2026-06-22 | 11 breaking changes — see [V3 Migration Guide](docs/migration/V3_MIGRATION.md)                                                                      |

---

## Themes

### 1. Metaengine → Production

The metaengine prototype proves the Event-Query Model works: fold return types
infer ADTs, typed closures avoid strings, pagination is detected from input
structs. The gap between prototype and production:

- **Real SQLite engine** — wrap `SQLViewStore` as a metaengine backend.
  The first production engine validates the interface design.
- **Cost model calibration** — `nsPerOp=100` is arbitrary. Needs benchmark-driven
  calibration with real engine profiles.
- **Integration** — `projection.Projection` adapter, `kv.Store` bridge,
  `graph.GraphSink` bridge. The metaengine must connect to existing infrastructure.
- **FilterOn/SortOn → SQL pushdown** — Go closures cannot be inspected. Design
  decision needed: DSL, codegen, or keep in-memory filtering.

### 2. Benchkit → Reliable

The benchmarking toolkit is functional but has known gaps:

- **Warmup store pollution** — warmup writes to the main store, inflating
  journal metrics for subsequent phases.
- **Pebble backend tests** — Pebble works via CLI but has no test coverage.
- **Production replay** (Phase 6) — replay real event streams for benchmarking.
- **benchtest.RunSuite** (Phase 7) — preset integration for `stack/bench`.

### 3. Codebase Health

- **Stale API golden file** — `docs/api_surface.txt` still contains 9 removed
  APIs. CI `api-stability` job will fail until regenerated.
- **Aggregate→Stream completion** — 2 exported error var pairs missed, ~70 files
  with stale comments, AGENTS.md/SKILL.md references.
- **Module extraction** — `retry/` and `idempotency/` are zero-CQRS-coupling
  candidates for standalone repos (see [extraction analysis](docs/planning/2026-07-23_extraction-analysis.md)).

### 4. Consumer Experience

- **Read-your-writes helper** — `WaitForVersion(ctx, aggID, version)` for
  consumers who need immediate consistency after a write (book insights gap).
- **Bounded staleness** — `WithMaxStaleness(duration)` for projections that
  can tolerate lag (book insights gap).
- **Consistency model document** — `docs/CONSISTENCY_MODEL.md` documenting
  single-process scope and eventual consistency guarantees.
- **SQL-backed `idempotency.Store`** — for multi-process Postgres deployments
  (~100 lines: `INSERT ON CONFLICT DO NOTHING`).

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
- Parquet journal (`storage/parquet`) — columnar, compressed, cloud-native
- DuckDB connector (`storage/duckdb`) — OLAP-grade analytical materializations
- NATS/ValKey stream adapter — ADR-0025 accepted
- Distributed event bus — multi-process backend for event distribution

---

## Non-Goals

- **Framework opinions** — the library will never mandate a transport, message
  broker, or SQL driver. Consumers compose their own stack.
- **Splitting the `event/` module** — 27 importers, real cohesion. Explicitly
  decided in v4. Do not split.
- **ORM features** — no query builder, no ORM-style relations, no lazy loading.
  `RawWhere` escape hatch covers the 5% case. Principle: "Library, not framework."
- **RollupSpec / RollupProjection** — premature abstraction. `sink.Increment`
  is the composable primitive; consumers compose it directly.
- **Redis adapter** — the author is not a fan of Redis. ValKey (the LF-backed
  fork) is the recommended alternative. If starting fresh, pick ValKey, NATS,
  or Kafka instead.

---

## Experimental / Go-stdlib-blocked

- **Remove `goexperiment.jsonv2` tag** — JSON v2 is fully adopted (~25 production
  files). The tag remains only because Go 1.26 hasn't graduated json/v2 from
  experimental. Remove when Go stabilizes it (expected Go 1.27+).
- **Turso MVCC concurrent-write support** — blocked on upstream experimental MVCC.
