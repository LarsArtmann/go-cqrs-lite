# Roadmap — go-cqrs-lite

> Where we are, where we're going, and what's next.

---

**Release cadence since 2026-08-16:** the 22-tag chain, the full 39-tag v4
wave B1–B7 (2026-08-29 — cut, pushed, verify-ci 76/76 green, zero local
replaces remain), the issue-#20 cqrs-lint proxy-repair tags (09-01:
v4.8.1 + the `/v4`-suffix guard in tag-release.sh), and `cmd/cqrs-lint/v4.9.0`
(09-06: V007 `v5-removed-api-usage` + hardening) — with three broken versions
retracted + repaired same-day on 08-16. **v5 unification in progress**
(ADR-0123). 82 `go.mod` files. The `[Unreleased]` window carries the
2026-09-06 waves (severity/confidence contract, suppression/fix/lintutil
hardening, self-healing formatter guard) and the 2026-09-03 master-CI repair.
See CHANGELOG `[Unreleased]` for the full per-entry detail.

---

## Release History

| Version                      | Date       | Highlights                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                          |
| ---------------------------- | ---------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| [2026-08-16 module releases] | 2026-08-16 | 22 coordinated tags: `id/v4.5.0`, `record/v4.3.0`, `metadata/v4.5.0` (`Metadata[K]` generic), `schema/v4.3.0` (`UpcastSourceTransform`), `event/v4.7.0` (store transforms + actor context), `command/v4.7.1`, `query/v4.6.1` (`AsRecord`), `middleware/v4.5.0` (`CommandActorContext`), `watermill/v4.5.0` (CatchUpSubscriber replay fix), `metaengine/v4.11.0` (layout roles, DemoteEngine, MariaDB dialect, live-cost) + 8 engine tags, `storage/v4.7.0→v4.7.1` (keyset pagination ~285x, packet-safe chunking). **Retracted + repaired:** command/v4.7.0, query/v4.6.0, storage/v4.7.0 (standalone-build breaks)                                                                                                                                                 |
| [Unreleased]                 | —          | • **2026-09-06**: cqrs-lint severity/confidence contract (14 split-brains fixed, S008/S009 now error), `rules --json`, suppression-parser/fix-provider/lintutil hardening, self-healing formatter guard<br/>• **2026-09-05**: V007 `v5-removed-api-usage` (204 rules) + getting-started modernized onto `system.New`<br/>• **2026-09-03**: master-CI repair wave 2 (FlakeHub, module matrix discovery, go.work externals)<br/>• **2026-08-30/31**: D3 planned-table pushdown train (filters/sort/keyset, MapScan/MapUpdate, EXPLAIN proofs, cross-engine parity matrix), `LayoutPlanEvolver`, `BackfillPlannedCollection`, `EffectiveDurability`, `RenewLease` + claim metrics<br/>• **v5 pre-cut**: 73 deprecation markers (stack tiers, view/relational, tombstone API, transport) landed 2026-08-17<br/>• **Durability tiers**: pebble/postgres/bbolt/badger map Strict/Normal/Relaxed (08-17/18)<br/>• **Vector/graph**: binary float32 vector payloads (~31-35x search), depth-1 graph short-circuit, filtered k-NN, GraphRemoveEdge (08-16/17) |
| v4.2.0                       | 2026-07-27 | CBOR→JSON transcoding, 3 new cqrs-lint rules (65 total), coverage-drift checker, CI gates (duplication/layers/api-stability/coverage), wrapClosed consolidation, UP1 test hardening, go-error-family v0.10.0 (6-family)                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                             |
| v4.1.0                       | 2026-07-23 | Deprecated API removal, metaengine, benchkit, Increment/Reset rollups, README overhaul, error taxonomy migration, Aggregate→Stream rename (ADR-0058)                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                |
| v4.0.4                       | 2026-07-23 | COSE signing/encryption, multi-batch event store, OTel storage instrumentation, getting-started guide, architecture docs                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                            |
| v4.0.3                       | 2026-07-22 | SQL dialect abstraction, stack preset centralization, JSON v2 migration, harmful duplication elimination, cqrs-lint scanner overhaul                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                |
| v4.0.2                       | 2026-07-18 | CBOR time encoding fix, timezone-safe types (Instant, WallTime, Date), cqrs-lint loader error surfacing                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                             |
| v4.0.1                       | 2026-07-16 | projectionhost deadlock/leak/sort fix, watermill deadlock fix, storage/view IS NULL+RawWhere+ViewUpdater, cqrs-lint first release (60 rules)                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                        |
| v4.0.0                       | 2026-07-11 | CBOR defaults, API cleanup, BackfillHandler consolidation, HealthCheck, storage split, `/v4` path migration                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                         |

---

## Themes

### 1. Metaengine → Production

The metaengine proves the Event-Query Model works: fold return types infer ADTs,
typed closures avoid strings, pagination is detected from input structs. The
production maturity chain is complete:

The production maturity chain is complete. Highlights (full per-entry detail in
[CHANGELOG.md](CHANGELOG.md) `[Unreleased]`; every item shipped and tested):

- **10 engine backends** — Memory, SQLite, Pebble, bbolt, DuckDB, Postgres,
  MySQL, Badger, Dgraph, Turso (+ Iroh replication prototype). Each in its own
  module (dep isolation); SQLite/Pebble/Bbolt/DuckDB/PG all implement
  StreamLogBackend; DuckDB + PG have AtomicAppender
- **SQL pushdown + layout planning** — `FilterOnField`/`SortOnField` pushdown
  per dialect (json_extract / JSONB `->>`); `LayoutPlan` generates indexed-column
  DDL from declared query fields (10x filter+sort); pgengine expression indexes;
  DuckDB columnar planning
- **Streaming + reactive reads** — `StreamScan(ctx) iter.Seq2`, raw-value
  readers (single-pass decode), `ServeSSE` with Last-Event-ID replay,
  `Watcher`/`WatchTyped`, `PrefetchCache`
- **Cost model + rules** — `EngineProfile` with NsPerRead/NsPerWrite, ReadCosts,
  Replication/Persistence/NetworkRTT fields; composable `RulePipeline`;
  materialize-vs-replay advisory; `SerializablePlan`; `Calibratable` +
  `Calibration` for external engines; live RTT via `ProbeEngine`/`LatencyTracker`
- **Correctness rails** — fold sealed interface (no `any` god-struct),
  exhaustiveness guard, property-based cross-engine parity (rapid), Postgres
  testcontainers, `adttest.RunMatrix` over all 10 ADTs,
  `enginetest` shared contract suites (incl. StreamLog positional semantics),
  `AtomicAppender` optimistic concurrency, boundary key validation

**Remaining (short-term):** See [TODO_LIST.md](TODO_LIST.md) — iroh graph
`WriteOp` convergence (edges do not replicate cross-peer yet),
capability-conformance wiring under `#test-integration`, and the pending tag-wave
batches B2-B7 (B1 was cut 2026-08-29). (Layout calibration, seq-carrying journal reads, and the DuckDB float
decode guard have shipped.)

**Metaengine v2 (ADRs 0111-0119) — ES-native architecture shipped:**

- ✅ **`record/` module** — shared `Record` + `CommonMetadata` + `StreamRef`
  (zero deps). The canonical type the ES-native metaengine folds over
  (ADR-0111)
- ✅ **SQLite engine extraction** — `metaengine/sqliteengine/` removes
  `modernc.org/sqlite` from core deps (ADR-0115)
- ✅ **GraphAdapter** — `metaengine/graphadapter/` wraps `graph.MemoryDriver`
  as `metaengine.Engine`, replacing the deleted in-engine GraphBackend
  (ADR-0113)
- ✅ **Badger engine** — `metaengine/badgerengine/` full impl, mirrors
  pebbleengine. Calibrated from benchmarks. ADR-0118
- ✅ **Dgraph engine** — `metaengine/dgraphengine/` distributed graph backend
  via `dgo` gRPC + DQL. ADR-0119
- ✅ **Record-aware folds** — `OnRecord`/`OnRecordTyped` + `ApplyRecord` +
  `RecordAwareFold`. Folds receive StreamID, Version, metadata (ADR-0112)
- ✅ **`event.AsRecord()`** — bridges ES pipeline to `record.Record`.
  `projectionadapter.Handle()` calls `ApplyRecord()`
- ✅ **Auto-projection** (ADR-0116 Layer 1) — `AutoInsert`/`AutoUpdate`/
  `AutoDelete`/`AutoCRUD` reflection-based field-mapping inference. Zero
  per-event reflection cost. `AutoCRUDByConvention` suffix-based naming
- ✅ **Record stamping** — `AutoInsert`/`AutoUpdate` auto-stamp Record metadata
  into matching result fields (`record_stamp.go`)
- ✅ **Tombstone deprecation** — all tombstone API `// Deprecated:`, migration
  guide written. Functional in v4, removal in v5 (ADR-0114)
- ✅ **Bbolt/MySQL/Turso engines** — `metaengine/bboltengine/`,
  `metaengine/mysqlengine/`, `metaengine/tursoengine/` modules with self-registering
  drivers. Bbolt source-of-truth tests, MySQL/Turso driver registration.
- ✅ **Live Cost Measurement** — runtime `NetworkRTT` via `ProbeEngine`,
  `LatencyTracker`, `Store.Replan`, `CheckRouting`, `StartAutoReplan`. PG/Dgraph/
  MySQL/Turso probers wired. `Doctor()`/`ExplainPlan()` live-latency output.
- ✅ **Operator-driven layout planning (ADR-0124)** — `Priority` enum + config
  hierarchy, embed-vs-normalize scoring, `ReplanLayout`, `ConfirmRebuild`,
  `Store.SetPriority`, layout warnings in `Doctor()`/`ExplainPlan()`.
- ✅ **Command lifecycle as event streams (ADR-0117)** — `commandlifecycle/`
  events, `Recorder`, middleware, DLQ/retry/failure projections.

**Remaining (short-term):** See [TODO_LIST.md](TODO_LIST.md) → Metaengine
sections — layout calibration, capability conformance test, DuckDB aggregation
pushdown, seq-carrying journal reads, calibration benchmark regression check.

**Remaining (long-term, ROADMAP):**

- **Vector/Search/Spatial engine backends** — currently Memory-only (brute-force).
  DuckDB VSS extension (vector similarity), Postgres tsvector (full-text search),
  PostGIS (spatial). Each is a separate engine module with its own deps.
  Dgraph also has native vector + geo support (`DgraphVectorBackend`,
  `DgraphSpatialBackend` — separate from the existing Memory brute-force path).
- **Dgraph backend expansion** — `SnapshotBackend` (versioned predicates or
  snapshot namespace), `StreamLogBackend` (stream-keyed log ops). Both needed
  for full `system.Bundle` integration.
- **ADR-0112: Command sourcing** — folding over command history (not just events).
  Requires `CommandAwareFold` interface + command journal replay.
- ✅ **ADR-0113 Phases 3–4: Delete `GraphBackend` interface entirely** — DONE.
  `GraphBackend` removed from `metaengine/engine.go`, all engines route ADTGraph
  through `graphadapter`. `adttest.RunMatrix` updated.
- **ADR-0116 Layers 2–3** — Layer 2 (100% codegen from struct inspection),
  Layer 3 (100% auto-route from declared queries). Currently Layer 1 (reflection)
  is shipped.
- 🧪 **ADR-0117: Command Lifecycle as Event Streams** — `commandlifecycle/`
  module shipped with event types, `Recorder`, middleware, and DLQ/retry/failure
  projections. Remaining: version tracking fix, real retry middleware
  integration, `system.WithCommandLifecycle(eventSink)` one-call setup, skill
  docs. See [TODO_LIST.md](TODO_LIST.md).
- **`metaengine-gen` code generator** — `cmd/metaengine-gen` for typed Store
  methods from query declarations. Go AST parsing + template generation.
- **Structured query expression tree** — `query.Or`/`query.And`/`query.Gt`
  composable tree (currently flat `Conditions` + `RawWhere` escape hatch).
- ✅ **DuckDB columnar-native storage** (ADR-0092) — `WithColumnarLayout()`
  extracts all exported fields into native typed SQL columns (float64→DOUBLE,
  int→INTEGER). Vectorized GROUP BY/SUM/AVG on DuckDB.
- **Postgres GIN containment Indexes** — `@>` operator for JSONB containment
- ✅ **Operator-driven engine selection + layout planning** — `system/` self-registering
  driver registry shipped; `metaengine/` registry moved from `system/`. All 10 drivers
  (memory, sqlite, pebble, bbolt, duckdb, postgres, mysql, badger, dgraph,
  turso) self-register. Layout planning (ADR-0124) shipped with priority system,
  embed-vs-normalize scoring, `ReplanLayout`, `ConfirmRebuild`. Live cost
  measurement shipped with `ProbeEngine`, `LatencyTracker`, `Store.Replan`,
  `CheckRouting`. Remaining: NATS/Redis bus driver registration, role-based sync.

### 2. Benchkit → Evidence-Grade

The benchmarking toolkit is functionally complete and evidence-grade. The full
evidence plan shipped: durability/recovery, production replay,
`benchtest.RunSuite`, analytical profile, Postgres backend, scaling sweeps,
benchstat/manifest output, profiling, and a first real run across
memory/pebble/sqlite (2026-07-24). DuckDB backend added (2026-07-29).
Evidence-grade metrics added (2026-08-01, ADR-0090).

- ✅ **Tagged `benchkit/v4.2.0`** (tagged + pushed 2026-07-27)
- ✅ **DuckDB backend** — benchmarkable via `--backend duckdb` (CGo-isolated)
- ✅ **Evidence-grade metrics** (ADR-0090) — statistical reliability
  (`RepeatCoV`, `RepeatIsReliable`), GC pause (`GCMaxPause`), allocation
  (`AllocsPerOp`, `BytesPerOp`), data integrity (`IntegrityErrors`), write
  amplification (`Disk.WriteAmplification`), cold/warm read distinction,
  tail ratio (P99/P50), environment enrichment (`CPUModel`, `TotalRAMBytes`)
- ✅ **Soak test drift** — `GCMaxPauseDriftPct`, `AllocGrowthPct` for memory
  boundedness verification
- ✅ **Metaengine benchmark** — Memory + SQLite engines, Counter + Map ADTs.
  Counter workload correctness assertions (previous event-type mismatch bug
  silently measured empty stores)
- ✅ **5-backend comparison** — `docs/benchmarks/2026-07-31_backend-comparison.md`
- **Run-to-run variance** — ~20-25% on the memory backend. `--repeat N`
  (median-of-N) mitigates it; the CI gate compares medians for the same reason.
- ✅ **Regression gate** (2026-08-16) — CI `regression` job compares median
  ns/op of the gate set against the previous run's baseline artifact and FAILS
  above 25% (`scripts/benchmark-regression.sh`, artifact self-refreshes). The
  one-bench-system consolidation also removed the redundant integration and
  v2-era harnesses — benchkit (SDK) + cqrs-bench (CLI) + stack/bench (gate
  entry) + slimmed metaengine/bench (planner calibration) remain.

### 3. cqrs-lint → Trustworthy

The linter grew from 65 to **204 rules** across 10 categories. Quality has been
hardened through multiple brutal review passes and 7 consumer feedback rounds.

- ✅ **204 rules shipped** across correctness, API misuse, boilerplate, adoption,
  architecture, consistency, security, performance, testing, version.
  Metaengine-aware detection (F018-F026). Resilience rules (B029-B031).
  Observability rules (F027-F029). Optimistic concurrency rules (C041-C042).
  Deprecated-transport coaching (F030, ADR-0127). v5-removed-API detection
  (V007, 2026-09-05) with a two-directional drift meta-test; severity/
  confidence contract meta-tested (14 split-brains fixed); suppression
  parser, fix provider, and lintutil hardened (2026-09-06).
- ✅ **Feature profile system** — auto-detects consumer module usage and adapts
  context-dependent rules. TLS-aware server detection, ServerLocal heuristic.
  Per-module detection infrastructure (`ProfileForFile`) — C017, S002, S003,
  S006, S007, C036 migrated; ~20 detectors still on primary profile
- ✅ **Self-lint mode** — `IsLibrarySelfLint()` auto-detects and skips 29
  consumer-coaching rules when linting the go-cqrs-lite source itself
- ✅ **Quality hardening done** — E010/E011/E013/E014 rewritten with type-aware
  matching; import-alias resolution built; C030/S006 reviewed; suppression
  parser fixed; block-level suppression (ADR-0088); C008 struct-level ignore
  config; C038/C039/C040 event-type mismatch + dead-fold-case detection;
  E006 fold-aware orphaned event detection
- ✅ **Scorecard** — `--scorecard` / `scorecard` subcommand. Module adoption
  scorecard with used/missing/irrelevant partitioning, coverage %, top-3
  recommendations. Text + JSON + Markdown output. `--scorecard-threshold N` CI gate
- ✅ **Group-by aggregate** — `--group-by aggregate` infers aggregate from
  event-type prefixes + decider state types, groups findings by aggregate
- ✅ **Config UX** — JSONC config loader (comment support), `explain`
  interactive documentation explorer, `doctor` overhaul with preset + override
  display, config presets unified as single source of truth
- ✅ **Consumer feedback received** — 7 consumer reviews (crush-daily,
  Standup-Killer, bank-sync, cqrs-htmx, browser-history, timesheets, DiscordSync).
  Drove signal-to-noise from ~25% to ~90%+.
- ✅ **v4.6.0 tagged + pushed** — 202 rules, resilience/observability/correctness categories,
  detection improvements, KeyHolderAI feedback fixes.
- ✅ **F021 rewrite** — per-query fold analysis (3+ folds per query, not global)
- ✅ **Scorecard metaengine section** — `ScorecardMetaengine` in text/JSON/
  markdown/SARIF. Cross-format consistency tests. E2E metaengine tests.
- ✅ **Self-lint cleanup** — 15 stale suppressions removed, C005 CBOR decode
  bug fix, 0 CRITICAL/0 ERROR/0 load errors
- ✅ **Lint gate 58→0** — all 65 modules clean
- **Run against real consumers** — validate false-positive rates against live
  projects. See [TODO_LIST.md](TODO_LIST.md).

### 4. Module Extraction

Two modules extracted to standalone repos (see
[extraction analysis](docs/planning/archived/2026-07-23_extraction-analysis.md)):

- ✅ **Extract `retry/` → `go-retry`** — ADR-0064. Repo created, pushed, tagged
  (v0.1.0). go-cqrs-lite `retry/go.mod` uses real versioned require directives.
- ✅ **Extract `idempotency/` → `go-idempotency`** — ADR-0065. Repo created,
  pushed, tagged (v0.1.0 + v0.1.1). Sub-modules (`kvstore`, `sqlstore`) resolved
  against tagged deps.
- ✅ **Extract `flightrecorder/` → standalone** — zero-dep module (stdlib only).
  Natural standalone candidate once API stabilizes.

### 5. Storage & Transport Expansion (design-doc-backed)

These have design docs and graduated from "Raw Ideas"; concrete phases will move
to [TODO_LIST.md](TODO_LIST.md) when actively worked:

- ✅ **DuckDB analytical backend** — shipped as `stack/duckdb` preset +
  `DuckDBDialect` in `storage/sql/`. CGo isolated (ADR-0071). Columnar OLAP
  queries alongside the transactional store.
- ✅ **MySQL/MariaDB backend** — shipped as `stack/mysql` preset +
  `MySQLDialect` in `storage/sql/`. `Dialect` interface expanded with 4 upsert/
  quoting methods (ADR-0080). Pure-Go driver (`go-sql-driver/mysql`), no CGo.
  Full `idempotency/sqlstore` support with MySQL `IF()` conditional TTL.
- ✅ **Backend tradeoff framework** — `DurabilityTier` (Strict/Normal/Relaxed)
  translated to per-backend pragmas. `Capabilities` machine-checkable matrix.
  Mixed workload benchmark phase. `BACKEND_TRADEOFFS.md` guide.
  5-backend comparison benchmark.
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

### 7. Observability — Flight Recorder

Go 1.25 `runtime/trace` capture on slow/error/always triggers (ADR-0089):

- ✅ **Zero-dep `flightrecorder/` module** — `Recorder`, `TriggerFunc` system,
  configurable options
- ✅ **CQRS middleware** — `CommandFlightRecorder`, `EventFlightRecorder`,
  `QueryFlightRecorder`
- ✅ **Decider/projectionhost/stack integration** — `WithFlightRecorder` options
  across all lifecycle-managed components
- **Deeper integrations (not started)** — `scheduling.Scheduler` hook,
  `transport/http` SSE hook, `metaengine` Store hook, example/taskmanager demo,
  trace file validity integration test.

### 8. Benchmark Trust — evidence-based cost constants

The ADR review session flagged this as the "highest-leverage next move."
29 of 43 metaengine benchmarks discard results (no correctness assertions).
DuckDB and Postgres engine cost constants are hand-picked with zero empirical
backing (0 benchmarks exist for these engines).

- ✅ Add correctness assertions to benchmarks — 50+ benchmarks across 18 files
  now assert results. Found 3 real bugs (empty-stream Save, wrong cursor,
  JSON map decode failure).
- ✅ Create DuckDB + Postgres engine benchmarks — 4 calibration benchmarks per
  engine (batch insert, pushdown scan, vectorized aggregation, full scan).
- ✅ ReadCosts — per-read-pattern cost model added to `EngineProfile`. Exposes
  the 4000× gap between DuckDB point lookups and aggregations.
- ✅ **Export `Calibratable` for external engines** — `Calibration` struct +
  `CalibrationCosts` (includes `ReadCosts`). All external engines embed
  `Calibration` and implement `Calibratable`.
- ✅ **Consumer DX helpers** — `NewSQLiteEngineFromDSN`, `PlanFromSQLite`,
  `Store.LogPlan`, typed projection decoders (`EventWithID[P]`, `Register`,
  `NewTypeDecoder`, `NewWithDecoder`). ~130 lines of boilerplate eliminated.
- ✅ **Metaengine benchmark module** — `metaengine/bench/` with cross-engine
  benchmarks. M4.2 DuckDB columnar extraction benchmark (3-way comparison).
- ✅ **Full-pipeline benchmarks** — 7 new files in `stack/bench/` + 6 cross-module
  benchmark files (projectionhost, transport, decider, scheduling, middleware).
- **Regression baseline + CI integration** — calibration benchmarks should run
  in CI and fail if constants drift >3×. See [TODO_LIST.md](TODO_LIST.md).

### 9. Deferred Debt (ADR-committed) — RESOLVED

Both items explicitly committed to in the 2026-08-03 ADR review are now DONE:

- ✅ **Ghost bus removal** (ADR-0028) — all three ghost bus files deleted
- ✅ **Metadata aliases completion** (ADR-0031) — both `command.Metadata` and
  `query.Metadata` are standalone structs

### 10. Iroh Distributed Engine (ADR-0096)

Evaluating Iroh (Rust CRDT) as a distributed metaengine backend. The
replication model (ADR-0093) established the foundation; Iroh would be the
first `ReplicationLeaderless` engine.

- ✅ **ADR-0096 written** — evaluates CGo FFI vs sidecar bridge approaches.
  Documents maturity assessment: `iroh-docs` NOT in C FFI, blocks direct
  integration. PN-Counter via Iroh identified as the killer feature.
- ✅ **Level 2 prototype shipped** — `metaengine/irohengine/` implements
  `Replicated(localEngine, ...)` with a pluggable `Transport` interface. In-process
  `Network` mock simulates P2P convergence. Passes `adttest.RunMatrix` parity,
  LWW resolution, PN-Counter, and MapUpdate-does-not-replicate tests.
- ✅ **Real QUIC FFI transport shipped** — `metaengine/irohengine/quic/` implements
  `Transport` over **real Iroh QUIC streams** via the `iroh-go` C bindings (CGo
  required). Every `Publish` opens a QUIC BiStream, serializes the `WriteOp`, and
  sends it to all connected peers. RTT measured from QUIC's own ACK timing via
  `conn.Rtt()`. Demo executable with real latency measurements.
- ✅ **Loopback transport shipped** — `metaengine/irohengine/loopback/` implements
  `Transport` over **real TCP connections** with length-prefix framing. NO CGo
  required. Middle tier of the transport testing pyramid: catches
  serialization/framing bugs that InProcessNetwork cannot. 9 convergence tests.
- ✅ **Latency measurement** — `LatencyCollector` with rolling 512-sample window
  (P50/P95/P99/max). `Profile()` returns measured values (P99 for lag, 2×P50
  for RTT).
- ✅ **CBOR encoding** — both transports switched from JSON to `fxamacker/cbor`.
  Fixed `time.Time` truncation and `map[any]any` decode issues.
- **Evaluate `iroh-go` C binding stability** (third-party binding for Iroh Rust)
- ✅ **Tag loopback + quic modules** — `loopback/v4.0.0` + `quic/v4.0.0` tagged
- ✅ **WriteOp.ID dedup ring** — both transports now have bounded dedup sets (10K entries)
- ✅ **Fix `TestQuicSetConvergence` flakiness** — fixed 2026-08-08 (unified `Eventually` blocks)
- ✅ **QUIC ADT matrix** — `TestQuicADTMatrix` runs full 10-ADT matrix against QUIC transport

### 11. Metaengine Persistence + System Redesign

Two interconnected design efforts documented in late 2026-08 planning sessions.

**Persistence enum** (ADR-0098, `docs/planning/2026-08-04_07-15_SUPERB-METAENGINE-PERSISTENCE-ENUM.md`):
Declares whether an engine's data survives process exit (DDIA Ch1 reliability
axis). ✅ Shipped — `Persistence` field on `EngineProfile`, per-engine
`Profile()` updates (Memory=volatile, SQLite/Pebble/DuckDB/PG=persistent),
`durabilityRule` WARN/INFO diagnostics, `CollectionInfo`/`SerializablePlan`
fields, `Doctor()`/`ExplainPlan()` output.

**System topology redesign** (`docs/planning/metaengine-redesign.md`):
A comprehensive design for a `system.System` type that replaces the current
`Bundle` with an operator-configured, driver-registered, `samber/do`-scoped
composition root. Features: driver registry (database/sql model), operator
YAML+env config, N-instance composition (source-of-truth + projection layers),
scream store (tiered deployment enforcement), cache tier, HTTP admin.

🧪 **P0/P1/P2/P3 + lifecycle hardening shipped** (`system/v4` module).
DomainConfig/DeploymentConfig separation, Op[State] routing, driver registry
(wired — SQLite works through `New()`), EventAdapter/CommandAdapter/QueryAdapter,
simpleBus + MultiBus (both wired into `New()`), CachedEventStore, SnapshotBackend
(wired with codec + strategy), scream store (wired into `New()` + plan-drift
detection), introspection API (real health checks + HealthCheckDetailed +
EngineNames + ShutdownOrder), koanf YAML config, System.Verify/Plan/Explain,
projection decoder wiring (`ProjectionTypeDecoder`/`ProjectionEventDecoder`).
All 5 engines implement StreamLogBackend (Memory, SQLite, Pebble, DuckDB,
Postgres). DuckDB + Postgres have AtomicAppender. HealthCheck on all 6
metaengine engines. Drain/EngineNames/ShutdownOrder/HealthCheckDetailed/
LagPerProjection/LagDuration/WorkerStatus/RegisterCloser shipped. Drainer
interface + RegisterDrainer shipped. orderedEngines topological sort shipped.
example/taskmanager migrated to `system.New()`.

**Remaining:** NATS/Redis bus driver registration, command lifecycle
integration (see [TODO_LIST.md](TODO_LIST.md) → WithActor Hardening and
ADR-0117 follow-ups). Dgraph real-instance testing and the layout-planning
verify gate are DONE (24/24 live Dgraph 2026-08-15; regression matrix
2026-08-11).

### 12. FoundationDB Backend (design-doc-backed)

> Fit analysis: `docs/planning/FOUNDATIONDB_METAENGINE_FIT.md` (2026-08-10) —
> all external claims verified against primary sources. Ordered tasks when
> actively worked: [TODO_LIST.md](TODO_LIST.md).

FoundationDB (Apple, Apache-2.0) as a metaengine engine. **Verdict: a viable
9th projection backend with a narrow scope — "shared durable projection state
across processes."** It is the only native multi-node, serializable, fully-ACID
engine in the roster; atomic counters (`tr.Add`), consistent secondary indexes
(simple-index pattern, one ACID txn), and push-based watch notifications are
its killer features. It loses on SQL pushdown, large values (10 MB txn cap,
100 KB value cap), OLAP aggregates, and operational weight (separate
`fdbserver` processes; Go binding requires CGo + `libfdb_c` → dedicated module
like `duckdbengine`).

- ✅ **Fit analysis** — full ADT × engine scoring, proposed `EngineProfile`
  (SingleLeader, lag 0, live-RTT via the Live Cost Measurement work), planner
  routing matrix, module strategy (`metaengine/fdbengine/`, register.go,
  NixOS `foundationdb` 7.3.68 ephemeral service for CI)
- **Goal (P0): projection engine** — Map/Set/Counter/Multimap/Log/SortedMap
  secondary indexes, Go-side scans, `MapUpdater` with transactional retries,
  10 MB-aware batching. Explicitly NOT v1: global journal (10 MB cap — keep
  event log local), Graph/Search/Spatial/Vector ADTs (declare degraded;
  FDB's "vector" recipe is a growable array, not ANN search)
- **Goal (P1): cross-process watchers** — FDB watches → change notifications
  across app replicas (new `WatcherSource`-style wiring); multi-instance
  control plane (plan/checkpoint/DLQ registry on FDB)
- **Alternative role** (cheaper first slice): FDB as multi-instance
  coordinator for plan/checkpoint/DLQ state, using existing local engines
- **Dependency:** live latency measurement (Theme 8 → Live Cost Measurement)
  must land first so FDB's profile uses measured RTT, not a hardcoded constant
- **Prereq:** half-day spike comparing `fdbengine` P0 vs pgengine + HA tooling
  for the target deployment (FDB wins on counters/watches/elasticity, pays
  ops tax)

---

## v5 Unification — Declare Types, Not Storage

> Decision: [ADR-0123](docs/adr/0123-v5-unification-single-composition-root.md).
> Ordered tasks: [TODO_LIST.md](TODO_LIST.md) → v5 Unification.

**The vision:** a developer declares only Commands, Events, and Query types
(their relationships to each other). The system infers projections, storage
layout, indexes, and engine routing automatically. Where data lives is an
operator decision at deployment time. No developer ever thinks about the
storage layer.

**The problem it solves:** go-cqrs-lite currently has two unreconciled
generations — `stack.Bundle` (8 backends, watermill, v1 projection tiers) and
`system.System` (2 drivers, simpleBus, metaengine) — plus three manual
read-model tiers (Materialize, RelationalProjection, GraphProjection) that
overlap with metaengine. Consumers face two valid, overlapping stacks with no
single blessed path. See
`docs/architecture-understanding/2026-08-09_self-integration-review.md`.

### Design Principles

1. **Developer sees only domain types.** Events, Commands, Query inputs,
   Query results. Nothing else. No ADT selection, no tier selection, no
   storage decisions.
2. **Auto-projection is the only API.** The planner inspects struct shapes and
   synthesizes folds. Explicit folds are an override, not a parallel system.
3. **Engines are universal.** Every engine implements every ADT. If an engine
   can't do something natively, it degrades (recursive CTE for graph on SQLite,
   brute-force for vectors on Memory). The planner warns honestly about cost.
4. **Operator picks infrastructure.** Backend choice is a deployment concern.
   The planner routes queries to the best available engine. Everything works
   everywhere — the planner tells you when it's suboptimal.
5. **One composition root.** `system.New()` with `DomainConfig` (closures) +
   `DeploymentConfig` (YAML). No dual paths.

### What Gets Deleted in v5

| Deleted                          | Replaced by                                               |
| -------------------------------- | --------------------------------------------------------- |
| `stack.Bundle` + all 8 presets   | `system.System` with self-registering drivers             |
| `simpleBus` + `BusDriverFactory` | `watermill.EventBus` (already abstracts NATS/Redis/Kafka) |
| `stack.Materialize`              | Auto-projection                                           |
| `storage.RelationalProjection`   | Multi-collection batch atomicity (engine internals)       |
| `storage.SQLViewStore`           | sqliteengine/pgengine with layout planning                |
| `graph.GraphProjection`          | Auto-projection + graphadapter                            |
| `stack.RunProjections`           | `projectionhost.Host` (the only runner)                   |
| `metaengine.GraphBackend`        | `graphadapter` (ADR-0113)                                 |
| payload-only `On` fold           | `OnRecord` (Record-typed default)                         |
| Duplicate metadata types         | `record.CommonMetadata` (ADR-0111 P3-4)                   |

### Phased Delivery

```
Phase 1: Record consolidation (type foundation)
Phase 2: Dead code removal (GraphBackend, simpleBus → watermill)
Phase 3: Self-registration infrastructure (registry → metaengine/)
Phase 4: All 8 backends self-register
Phase 5: Record-typed default folds
Phase 6: Auto-projection (Infer / OnRecord default)
Phase 6b: Operator-driven layout planning
Phase 7: Universal engine coverage + batch atomicity

Phase 8: Delete v1 tiers + stack.Bundle → cut v5.0.0
```

**v4.x bridge:** auto-projection ships alongside v1 tiers before v5. Consumers
can try it while v1 paths still work. v5 is the clean cut.

---

## Raw Ideas (No Design Yet)

> _Triage 2026-08-04: 14 items reviewed. `cqrs-lint init` SHOWSTOPPER removed (fixed). None stale._

- Event stream compaction / log truncation strategies
- Transactional outbox (ADR-0016 designed, zero code — the biggest ES-library
  gap per the 2026-08-14 review)
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
- Auto-denormalization in metaengine — planner detects that two queries share
  a common prefix and recommends a denormalized projection to avoid fan-out
  joins at read time. **→ Subsumed by operator-driven layout planning
  ([ADR-0124](docs/adr/0124-operator-driven-layout-planning.md)).**
- Metaengine plugin registry — third-party engine backends registered at
  runtime without recompiling (operator YAML config for engine selection).
  **→ Subsumed by v5 self-registration (ADR-0123 Phase 3).**
- CALM theorem ADR for metaengine — document why monotonic folds are CRDT-safe
  for replicated engines (supports Iroh integration, ADR-0096).
- `CalibrateScanEngine` — runtime calibration for scan/aggregation costs (not
  just point lookups; `CalibrateEngine` only measures `MapGet`).
- Per-module `.golangci.yml` split — golangci-lint v2 `config-dirs` would give
  each module ownership of its own exclusions. L effort, deferred until
  monolithic config becomes a maintenance problem.
- Rewrite `check-module-layers.sh` as `cmd/check-layers` — 348 lines of bash
  → Go program with testability. Deferred: script is stable, only runs in CI.
- Run cqrs-lint against real consumer repos (Kernovia, Standup-Killer,
  bank-sync, cqrs-htmx, DiscordSync, timesheets, crush-daily, KeyHolderAI) —
  validate false-positive rates. L effort, needs private repo access.
- Test against CockroachDB — it speaks the PostgreSQL wire protocol, so the
  existing `stack/postgres` + `storage/` SQL layer (pgx driver, `INSERT ON
CONFLICT`, JSONB) should work with near-zero changes. Point the DSN at port
  26257 and run the Postgres test suite. Note: CockroachDB is source-available
  (not OSS) — single-node/dev use only without a commercial license. Users who
  bring their own license can run it; go-cqrs-lite just speaks Postgres wire.
- **FoundationDB as a metaengine backend** — distributed ordered KV with
  ACID transactions (Apple, Apache-2.0). Atomic counters, consistent
  secondary indexes, push watches. Requires CGo binding + separate
  `fdbserver` deployment. **→ Graduated to [Theme 12](#12-foundationdb-backend-design-doc-backed)**
  (fit analysis: `docs/planning/FOUNDATIONDB_METAENGINE_FIT.md`).
- Add ScyllaDB metaengine backend + performance benchmarks — ScyllaDB is a
  NoSQL wide-column store (CQL, Cassandra-compatible, shard-per-core C++/Seastar,
  ultra-low single-digit-ms latency). NOT SQL — integration would be a new
  `metaengine/scyllaengine/` module via `gocql` (scylladb fork), not a SQL layer
  adaptation. Natural fit for write-heavy ADTs (Counter via distributed counters,
  Map/Set via wide-column rows, Log via clustering-key append). Compare against
  Pebble/SQLite on write throughput and point-read latency. Note: ScyllaDB is
  source-available (not OSS, switched from AGPL v3 in 2025) — free tier caps at
  10 TB / 50 vCPUs; "Never Customer" clause.

> Items with design docs graduate to a Theme above, then to [TODO_LIST.md](TODO_LIST.md)
> when actively worked.

---

## Open Questions (user decisions pending)

> Standing questions from recent sessions that block or shape work. Answers
> should be folded into TODO_LIST once decided.

1. **Next tag wave + severity policy** (updated 2026-09-06): the 39-tag v4
   wave B1–B7 was cut+pushed 2026-08-29 (plan archived at
   `docs/planning/archived/2026-08-27_17-30_PENDING-TAG-WAVE-PLAN.md`).
   OPEN decisions: (a) authorize the next patch/minor wave (metaengine
   session-7 surface is ≥1 minor untagged; see TODO_LIST Release section);
   (b) is severity-tightening (S008/S009 now `error` in v4.9.0) acceptable
   in a minor release, or gated behind a "Changed" + dedicated minor?
2. **SA1019 exclusion permanence**: keep the scoped
   `(middleware|idempotency)/.*_test\.go$` exclusion permanently, or migrate
   kvstore test matrices onto the go-idempotency contract suite before v5?
3. **Stale-pin sweep policy**: may future sessions bump ALL sibling-module pins
   repo-wide to latest tags mechanically (gate-verified), or only on breakage?
   A yes also greenlights the pin-drift meta-test failing on staleness.
4. **Vulncheck placement**: check-coverage / check-duplication / check-arch /
   check-depguard already run inside `#verify` (wired since `6f7c88388`,
   2026-08-03 — the Aug-14 "gates are unwired" premise was wrong; check-coverage
   rotted while WIRED because the script was broken). Only `#vulncheck` sits
   outside `#verify`. Fold it in (+time per run), keep it a manual pre-tag
   step (current TODO_LIST pre-tag checklist), or wire it into CI?
5. **Tracing JSON `omitempty` standardization** (from the WithActor review):
   only `ActorID` omits zero; `CorrelationID`/`CausationID`/`UserID`/`RequestID`
   serialize as empty strings. Making them all omit-zero is cleaner but a
   breaking JSON change for consumers parsing the raw shape. Standardize
   (needs ADR) or leave asymmetric?
6. **MySQL-8 nix backend** (2026-08-15): the nix integration envs run MariaDB
   (`pkgs.mariadb`), but MySQL 8 has meaningfully different JSON behavior
   (functional indexes, native JSON type). Add a real MySQL-8 nix VM check
   (`mysql8-vm`, ~130s in CI), or stay MariaDB-only and treat MySQL via
   docker probes as today?
7. **`mysqlengine.Dialect()` export** (2026-08-15): mysqlengine exports
   `Dialect() string` ("mysql"/"mariadb"). Keep as stable public API, or
   demote to internal and expose via `Profile()` metadata (avoids a
   stringly-typed API surface before the v5 freeze)?

---

## Non-Goals

- **Framework opinions** — the library will never mandate a transport, message
  broker, or SQL driver. Consumers compose their own stack.
- **Splitting the `event/` module** — 27 importers, real cohesion. Explicitly
  decided in v4. Do not split.
- **ORM features** — no query builder, no ORM-style relations, no lazy loading.
  Auto-projection (v5, ADR-0123) infers everything from struct shapes. If the
  auto-projection gets it wrong, override with an explicit `OnRecord` fold.
  There is no raw-SQL escape hatch — the developer never touches SQL.
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

---

## Metaengine Integration — Deferred Items

The following metaengine integration items were intentionally deferred. Each
has clear rationale.

### Catalog Bridge for Metaengine Queries (DEFERRED — YAGNI)

A `metaengine/catalogbridge/` module that feeds metaengine query collection
schemas into the catalog for OpenAPI/AsyncAPI documentation. No consumer needs
this yet — the catalog is consumer-side registration. Build it when a consumer
asks for it. Risk of doing it now: creates an unused module that rots.

### Stack HTTP/gRPC Server Convenience (DEFERRED — Coupling)

One-call options like `stack.WithHTTPServer()` or `stack.WithGRPCServer()` that
wire an SSEBroker or gRPC server to the Bundle. The manual wiring is ~5 lines.
Adding options couples stack/ to `net/http` and `google.golang.org/grpc` — the
stack package currently has zero transport deps. `sqlite.WithStack()` covers
the passthrough use case without coupling.

### SSE Code Consolidation (DEFERRED — Different Semantics)

Extracting a shared `sse` helper package from `metaengine/sse.go` and
`transport/http/sse.go`. The two implementations serve different layers:
`metaengine/sse.go` watches a Store collection for mutations (collection-watch
→ replay), while `transport/http/sse.go` bridges an `event.Bus` to HTTP clients
(bus-to-client). Merging risks a leaky abstraction. Cross-reference comments
were added to both files instead.

> **Update 2026-08-03:** The ADR review session discovered `go-sse` exists as
> a standalone library; both SSE implementations now consume it internally for
> wire-format serialization (ADR-0097, shipped — see CHANGELOG [Unreleased]).
> ADR-0091's rationale was revisited: the two implementations stay separate
> (different layers), but no longer reimplement the wire format.
