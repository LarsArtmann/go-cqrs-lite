# Roadmap — go-cqrs-lite

> Where we are, where we're going, and what's next.
> **Last updated:** 2026-07-27
>
> ✅ **`nix run .#verify` is GENUINELY GREEN end-to-end** (re-verified 2026-07-27
> after fixing a hidden cqrs-lint build break: the auto-daemon bumped go-output
> to v0.33.0 but `go-output/table` has no v0.33.0 release — downgraded to v0.32.0.
> The prior "GREEN" claim was stale for 3+ sessions). CI now also runs
> `#check-duplication`, `#check-api-stability`, `#check-layers`, `#check-coverage`.
> Race-aware thresholds, DSN-level SQLite `busy_timeout`, `soakTestScale`
> consolidation all in place.

---

## Current State (v4.1.0 shipped; file-size gate GREEN)

**v4.1.0 tagged** (2026-07-23) — initial module batch tagged on `/v4` import paths
(verify: `git tag --list '*/v4.1.0' | wc -l`). The workspace has 58 `go.mod`
files; 57 of 58 have tags reachable from HEAD. One module
(`metaengine/projectionadapter`) is tagged locally and on origin, but its
`v4.0.0` tag points to a commit **not reachable from HEAD** (orphaned) — it
needs re-tagging on the correct commit before consumers can resolve it
reliably. `metaengine/v4.1.1` was tagged 2026-07-26 (fixes a panicking
`MapUpdate` from v4.1.0). The deprecated-API removal batch shipped (see
[CHANGELOG.md](CHANGELOG.md) `[Unreleased]` → Removed).

The library covers the full CQRS/ES lifecycle: event sourcing with branded IDs,
command/query dispatch, pure-function deciders, three projection tiers
(document/KV, relational/SQL, graph), durable deadline scheduling,
event→command derivation, dead-letter quarantine, managed projection hosting,
event signing/encryption, OTel tracing/metrics, auto-documentation generation,
and a domain-aware linter (cqrs-lint, 60 rules).

**New since v4.0.0:**

- **Metaengine** (`metaengine/v4`, 🧪 experimental, tagged v4.0.0 + v4.1.0 +
  v4.1.1) — cost-based storage planner. Derives projections and engine
  assignments from two primitives (Events + Queries). 7 ADTs inferred from fold
  return types. 174 BDD specs, 86.2% coverage. SQLiteEngine shipped (ADR-0061);
  cost model calibrated; projection adapter integrated (ADR-0062); pushdown ADR
  written (ADR-0063); cross-engine meta-test guards parity. Phase 2 declarative
  pushdown deferred — see Theme 1.
- **Benchkit** (`benchkit/v4` + `cmd/cqrs-bench`, 🧪 experimental) — benchmarking
  toolkit with 7 named workload profiles + an analytical profile, 9-phase runner,
  scaling sweeps, benchstat/manifest output, and a first real benchmark run
  completed 2026-07-24. 88 benchkit + 12 CLI test functions. See Theme 2.
- **Incremental rollups** — `ProjectionSink.Increment` + `RelationalProjection.Reset`
  for atomic counter maintenance in relational projections.
- **Aggregate→Stream rename** (ADR-0058) — complete across code, tests, and docs.
  Deprecated aliases + wire-format identifiers preserved for compatibility.
- **Comprehensive README coverage** — all 58 modules with READMEs, 248 Go symbol
  references verified by `doc-check`.
- **Error taxonomy migration** — 13 sentinels migrated to `errorfamily` constructors.

58 `go.mod` files total (verify: `find . -name go.mod -not -path './vendor/*' | wc -l`).

---

## Release History

| Version | Date       | Highlights                                                                                                                                           |
| ------- | ---------- | ---------------------------------------------------------------------------------------------------------------------------------------------------- |
| v4.1.0  | 2026-07-23 | Deprecated API removal, metaengine, benchkit, Increment/Reset rollups, README overhaul, error taxonomy migration, Aggregate→Stream rename (ADR-0058) |
| v4.0.4  | 2026-07-23 | COSE signing/encryption, multi-batch event store, OTel storage instrumentation, getting-started guide, architecture docs                             |
| v4.0.3  | 2026-07-22 | SQL dialect abstraction, stack preset centralization, JSON v2 migration, harmful duplication elimination, cqrs-lint scanner overhaul                 |
| v4.0.2  | 2026-07-18 | CBOR time encoding fix, timezone-safe types (Instant, WallTime, Date), cqrs-lint loader error surfacing                                              |
| v4.0.1  | 2026-07-16 | projectionhost deadlock/leak/sort fix, watermill deadlock fix, storage/view IS NULL+RawWhere+ViewUpdater, cqrs-lint first release (60 rules)         |
| v4.0.0  | 2026-07-11 | CBOR defaults, API cleanup, BackfillHandler consolidation, HealthCheck, storage split, `/v4` path migration                                          |
| v3.6.0  | 2026-07-05 | Error taxonomy, deriver module, DOMAIN_LANGUAGE rebuild                                                                                              |
| v3.5.0  | 2026-06-29 | Idempotency, dispatch middleware, scenario DSL, scheduling, projectionhost                                                                           |
| v3.3.0  | 2026-06-28 | Three projection tiers, Watermill command bridge                                                                                                     |
| v3.1.0  | 2026-06-25 | SQL-backed view stores, multi-database split                                                                                                         |
| v3.0.0  | 2026-06-22 | 11 breaking changes — see [V3 Migration Guide](docs/migration/V3_MIGRATION.md)                                                                       |

---

## Themes

### 1. Metaengine → Production

The metaengine prototype proves the Event-Query Model works: fold return types
infer ADTs, typed closures avoid strings, pagination is detected from input
structs. The Pareto execution plan landed the production maturity chain:

- ✅ **Real SQLite engine** — `SQLiteEngine` wrapping `SQLViewStore` (ADR-0061)
- ✅ **Cost model calibration** — `EngineProfile.NsPerOp` with benchmark-driven
  constants (Memory=500ns, SQLite=7000ns)
- ✅ **Projection adapter** — `metaengine/projectionadapter` implements
  `projection.Projection` for `projectionhost.Host` (ADR-0062)
- ✅ **Pushdown ADR** — Phase 1: in-memory closures + `PushdownScan` interface
  seam. Phase 2 deferred (ADR-0063)
- ✅ **Dependency boundary** — Core `metaengine/v4` stays zero-dep; adapter is
  a separate module (ADR-0062)

**Remaining:** re-tag `metaengine/projectionadapter/v4.0.0` (current tag is
orphaned — points to a commit not in HEAD); implement Phase 2 declarative
pushdown when a production consumer needs SQL filter/sort pushdown. Metaengine
lint is clean (143 → 0); cost calibration shipped (Memory=500ns, SQLite=7000ns);
fold-classify logic and cross-engine meta-test guard correctness.

### 2. Benchkit → Released

The benchmarking toolkit is functionally complete — the full evidence plan
shipped: durability/recovery, production replay, `benchtest.RunSuite`,
analytical profile, Postgres backend, scaling sweeps, benchstat/manifest output,
profiling, and a first real run across memory/pebble/sqlite (2026-07-24). The
remaining work is maturity, not features:

- **Tagged `benchkit/v4.1.0`** (tagged + pushed 2026-07-25; points to a grab-bag
  commit mixing unrelated files — functionally correct but commit history is
  noisy). Also covers `cmd/cqrs-bench/v0.1.0` and `example/readme-quickstart/v0.1.0`.
- **Race-aware timing thresholds** — transport/grpc and benchkit soak tests now
  use build-tag race scaling (`race_on.go`/`race_off.go`). DSN-level SQLite
  `busy_timeout` eliminates SQLITE_BUSY under parallel test load. Verify gate GREEN.
- **Run-to-run variance** — ~20-25% on the memory backend. `--repeat N`
  (median-of-N) mitigates it; real-world regression tracking is the next step.
- **Real-world validation** — the first run verified plumbing and plausibility;
  a regression baseline + CI integration is the path to trustworthy numbers.

### 3. Module Extraction

Two modules are zero-CQRS-coupling candidates for standalone repos (see
[extraction analysis](docs/planning/2026-07-23_extraction-analysis.md)):

- ✅ **Extract `retry/` → `go-retry`** — ADR-0064 written (217 LOC, zero CQRS
  coupling, re-export alias plan). Execution requires creating the standalone repo.
- ✅ **Extract `idempotency/` → `go-idempotency`** — ADR-0065 written (553 LOC
  across 3 modules, re-export alias plan). Execution requires creating the repo.

### 4. Storage & Transport Expansion (design-doc-backed)

These have design docs and graduated from "Raw Ideas"; concrete phases will move
to [TODO_LIST.md](TODO_LIST.md) when actively worked:

- ✅ **Parquet journal design** — `docs/planning/parquet-journal-design.md`
  specifies Phase 1 (`storage/parquet` segment-based SeekableJournal, pure Go).
  Phase 2 (`storage/duckdb`, CGO) and Phase 3 (`stack/duckdb`) deferred.
  Original research at `docs/research/archive/...PARQUET_JOURNAL...md`.
- ✅ **NATS transport design** — `docs/planning/nats-transport-design.md`
  documents JetStream stream config, durable consumers, and CatchUpSubscriber
  integration via the existing `watermill/` bridge (no native `transport/nats/`
  module — ADR-0025 decision).

### 5. Consumer Experience

Gaps surfaced by the [book insights vs codebase review](docs/architecture-understanding/2026-07-23_book-insights-vs-codebase.html).
All four consumer experience gaps shipped via the Pareto execution plan:

- ✅ **Consistency model document** — `docs/CONSISTENCY_MODEL.md`
- ✅ **SQL-backed idempotency.Store** — `idempotency/sqlstore`
- ✅ **Read-your-writes WaitForVersion** — `decider.WaitForVersion`
- ✅ **Bounded staleness CheckStaleness** — `projectionhost.CheckStaleness`

---

## Raw Ideas (No Design Yet)

> _Triage 2026-07-27: 10 items reviewed. None stale, none ready to drop. Closest
> to graduation: SSE fan-out memoization (benchmarked, specific numbers) and
> Neo4j graph driver (consumer-pulled, design exists). The rest need design docs
> before becoming actionable._

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
  save ~99% of transform cost under high fan-out. Optimization, not a bug;
  benchmarked 2026-07-27.

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
