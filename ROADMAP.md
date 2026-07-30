# Roadmap — go-cqrs-lite

> Where we are, where we're going, and what's next.
> **Last updated:** 2026-07-30

---

## Current State (v4.2.0 shipped; 60 modules; 159 cqrs-lint rules; verify gate RED)

**v4.2.0 tagged** (2026-07-27) — 53 modules tagged and pushed. The workspace now
has **60 `go.mod` files** (added `stack/duckdb` and `metaengine/pebbleengine`
since v4.2.0). Verify: `find . -name go.mod -not -path './vendor/*' | wc -l`.

> ⚠️ **Verify gate is currently RED** (2026-07-30) — build error in
> `cmd/cqrs-lint/pkg/rules/correctness/c031.go:83`. The cqrs-lint module does not
> compile. See [TODO_LIST.md](TODO_LIST.md) "Verify Gate".

The library covers the full CQRS/ES lifecycle: event sourcing with branded IDs,
command/query dispatch, pure-function deciders, three projection tiers
(document/KV, relational/SQL, graph), durable deadline scheduling,
event→command derivation, dead-letter quarantine, managed projection hosting,
event signing/encryption, OTel tracing/metrics, auto-documentation generation,
a cost-based storage planner (metaengine), and a domain-aware linter (cqrs-lint,
159 rules).

**Major work since v4.2.0 (unreleased):**

- **cqrs-lint: 65 → 159 rules** across 10 categories — 94 new detectors including
  feature-adoption coaching (F-series), testing quality (T-series), architecture
  validation (E-series), and expanded correctness/API/boilerplate/security. See
  Theme 6.
- **Metaengine Phases 2–5** — SQL pushdown (ADR-0072), layout planning
  (ADR-0073), Pebble engine (ADR-0074), streaming reads, cost calibration.
- **DuckDB analytical backend** — `stack/duckdb` preset with CGo isolation
  (ADR-0071). Columnar OLAP queries alongside the transactional store.
- **Library adoption** — otter TinyLFU cache (decider), failsafe-go circuit
  breaker (middleware), testcontainers-go (Postgres integration tests), go-snaps
  (golden/snapshot testing across 16 modules).
- **Metaengine → taskmanager integration** — Counter ADT query with `/api/stats`
  endpoint. First proof of concept that metaengine works in a real CQRS app.

---

## Release History

| Version    | Date       | Highlights                                                                                                                                                                                                              |
| ---------- | ---------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Unreleased | 2026-07-30 | cqrs-lint 65→159 rules (10 categories), metaengine pushdown/layout/Pebble/streaming, DuckDB backend (ADR-0071), otter/failsafe-go/testcontainers-go/go-snaps adoption, metaengine→taskmanager integration               |
| v4.2.0     | 2026-07-27 | CBOR→JSON transcoding, 3 new cqrs-lint rules (65 total), coverage-drift checker, CI gates (duplication/layers/api-stability/coverage), wrapClosed consolidation, UP1 test hardening, go-error-family v0.10.0 (6-family) |
| v4.1.0     | 2026-07-23 | Deprecated API removal, metaengine, benchkit, Increment/Reset rollups, README overhaul, error taxonomy migration, Aggregate→Stream rename (ADR-0058)                                                                    |
| v4.0.4     | 2026-07-23 | COSE signing/encryption, multi-batch event store, OTel storage instrumentation, getting-started guide, architecture docs                                                                                                |
| v4.0.3     | 2026-07-22 | SQL dialect abstraction, stack preset centralization, JSON v2 migration, harmful duplication elimination, cqrs-lint scanner overhaul                                                                                    |
| v4.0.2     | 2026-07-18 | CBOR time encoding fix, timezone-safe types (Instant, WallTime, Date), cqrs-lint loader error surfacing                                                                                                                 |
| v4.0.1     | 2026-07-16 | projectionhost deadlock/leak/sort fix, watermill deadlock fix, storage/view IS NULL+RawWhere+ViewUpdater, cqrs-lint first release (60 rules)                                                                            |
| v4.0.0     | 2026-07-11 | CBOR defaults, API cleanup, BackfillHandler consolidation, HealthCheck, storage split, `/v4` path migration                                                                                                             |

---

## Themes

### 1. Metaengine → Production

The metaengine prototype proves the Event-Query Model works: fold return types
infer ADTs, typed closures avoid strings, pagination is detected from input
structs. The Pareto execution plan landed the production maturity chain:

- ✅ **Real SQLite engine** — `SQLiteEngine` wrapping `SQLViewStore` (ADR-0061)
- ✅ **Cost model calibration** — `EngineProfile.NsPerOp` with benchmark-driven
  constants (Memory=500ns, SQLite=7000ns), now split into `NsPerRead`/`NsPerWrite`
- ✅ **Projection adapter** — `metaengine/projectionadapter` implements
  `projection.Projection` for `projectionhost.Host` (ADR-0062)
- ✅ **Pebble engine** — `metaengine/pebbleengine` with LSM point reads
  (~7x faster than SQLite on MapGet). Separate module (ADR-0074)
- ✅ **SQL pushdown** — `FilterOnField`/`SortOnField` push WHERE/ORDER BY/LIMIT
  into SQLite via `json_extract()` (ADR-0072)
- ✅ **Layout planning** — `LayoutPlan` generates indexed-column DDL from
  declared query fields — 10x speedup on filter+sort (ADR-0073)
- ✅ **Streaming reads** — `StreamScan(ctx) iter.Seq2` for OOM-safe iteration
- ✅ **Taskmanager integration** — Counter ADT query with `/api/stats` endpoint

**Remaining:** wire layout planning into `Plan()` (auto-generate), JSON tax
reduction (single-pass decode), generated typed read API (`plan.Users.Get(ctx, id)`),
unified 7-ADT × 3-engine test matrix. See [TODO_LIST.md](TODO_LIST.md).

### 2. Benchkit → Released

The benchmarking toolkit is functionally complete. The full evidence plan shipped:
durability/recovery, production replay, `benchtest.RunSuite`, analytical profile,
Postgres backend, scaling sweeps, benchstat/manifest output, profiling, and a
first real run across memory/pebble/sqlite (2026-07-24). DuckDB backend added
(2026-07-29).

- ✅ **Tagged `benchkit/v4.2.0`** (tagged + pushed 2026-07-27)
- ✅ **DuckDB backend** — benchmarkable via `--backend duckdb` (CGo-isolated)
- **Run-to-run variance** — ~20-25% on the memory backend. `--repeat N`
  (median-of-N) mitigates it; real-world regression tracking is the next step.
- **Real-world validation** — the first run verified plumbing and plausibility;
  a regression baseline + CI integration is the path to trustworthy numbers.

### 3. cqrs-lint → Trustworthy

The linter grew explosively from 65 to 159 rules across 10 categories in a
single day (2026-07-30). The breadth is impressive but the quality needs
hardening before the linter is trustworthy for production use.

- ✅ **159 rules shipped** — correctness (31), API (28), boilerplate (28),
  performance (6), version (6), consistency (12), architecture (15), security (8),
  testing (8), adoption (17)
- ✅ **Feature profile system** — auto-detects consumer module usage and adapts
  context-dependent rules
- ✅ **Self-lint** — 181 suppressions across 83 files for library self-references
- 🔥 **Quality hardening needed** — E010/E011/E013/E014 are architecturally wrong;
  import-alias resolution missing; library self-lint mode would eliminate 35+ FPs.
  See [TODO_LIST.md](TODO_LIST.md) "cqrs-lint Quality".
- **50-item improvement backlog** — triaged in Pareto plan
  (`docs/planning/2026-07-30_21-16_CQRS-LINT-IMPROVEMENT-BACKLOG-PARETO-PLAN.md`)

### 4. Module Extraction

Two modules are zero-CQRS-coupling candidates for standalone repos (see
[extraction analysis](docs/planning/2026-07-23_extraction-analysis.md)):

- ✅ **Extract `retry/` → `go-retry`** — ADR-0064 written (217 LOC, zero CQRS
  coupling). Execution requires creating the standalone repo.
- ✅ **Extract `idempotency/` → `go-idempotency`** — ADR-0065 written (553 LOC
  across 3 modules). Execution requires creating the repo.

### 5. Storage & Transport Expansion (design-doc-backed)

These have design docs and graduated from "Raw Ideas"; concrete phases will move
to [TODO_LIST.md](TODO_LIST.md) when actively worked:

- ✅ **DuckDB analytical backend** — shipped as `stack/duckdb` preset +
  `DuckDBDialect` in `storage/sql/`. CGo isolated (ADR-0071). Columnar OLAP
  queries alongside the transactional store.
- ✅ **NATS transport design** — `docs/planning/nats-transport-design.md`
  documents JetStream stream config, durable consumers, and CatchUpSubscriber
  integration via the existing `watermill/` bridge (no native `transport/nats/`
  module — ADR-0025 decision).

### 6. Consumer Experience

Gaps surfaced by the [book insights vs codebase review](docs/architecture-understanding/2026-07-23_book-insights-vs-codebase.html).
All four consumer experience gaps shipped via the Pareto execution plan:

- ✅ **Consistency model document** — `docs/CONSISTENCY_MODEL.md`
- ✅ **SQL-backed idempotency.Store** — `idempotency/sqlstore`
- ✅ **Read-your-writes WaitForVersion** — `decider.WaitForVersion`
- ✅ **Bounded staleness CheckStaleness** — `projectionhost.CheckStaleness`

---

## Raw Ideas (No Design Yet)

> _Triage 2026-07-30: 10 items reviewed. None stale, none ready to drop._

- Event stream compaction / log truncation strategies
- Multi-tenant event store (schema-per-tenant)
- Distributed projection runner (leader election, multi-node coordination)
- Event archival to S3 / GCS / Azure Blob
- CQRS-lite dashboard (web UI for inspecting streams, events, projections)
- Automatic migration generator for schema evolution
- Property-based integration testing with state machine verification
- Performance regression dashboard (historical benchmark tracking)
- Neo4j/Memgraph graph driver (`graph/neo4j/`) — consumer-pulled sibling module
- SSE fan-out transform memoization — `CBORToJSONTransform` runs once per client
  (208µs for 100 clients, 3400 allocs/op). Memoization keyed by event ID could
  save ~99% of transform cost under high fan-out.

> Items with design docs graduate to a Theme above, then to [TODO_LIST.md](TODO_LIST.md)
> when actively worked.

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
