# Roadmap — go-cqrs-lite

> Where we are, where we're going, and what's next.

---

**metaengine v4.5.0 tagged** (2026-08-06) — 69 modules in `go.work`.
SerializableReadCosts in plan JSON (ADR-0100), consumer DX helpers,
6 file splits under CI limit, F015 store-aware fix, go-output daemon break
fixed. Module releases: cmd/cqrs-lint/v4.4.0, metaengine/v4.5.0, system/v4.0.0,
stack/mysql/v4.0.0, loopback/v4.0.0, quic/v4.0.0.
See CHANGELOG `[Unreleased]`.

---

## Release History

| Version      | Date       | Highlights                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                            |
| ------------ | ---------- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| [Unreleased] | —          | System/ Pareto P0/P1 fixes (driver registry wired, SQLite working, MultiBus/snapshot/scream wired, introspection real, decoder wiring, 5-engine StreamLogBackend, snapshot E2E), loopback transport (real TCP, no CGo), latency measurement overhaul, CBOR encoding, SSE race fix, consumer DX helpers (NewSQLiteEngineFromDSN/PlanFromSQLite/typed decoders), CalibrateEngine export, DuckDB LayoutPlanner follow-ups, dedup passes (68→66 clone groups, 46 lint fixes), layer enforcement (check-module-layers.sh self-enforcing, seven-tier model), KeyHolderAI feedback fixes (7 rules), scorecard SARIF output, go-humanize adoption, code quality (encryption double-clone, metadata immutability, kvstore TTL fix), M43/M44/M14 integration tests, Flight recorder (ADR-0089), metaengine Tier 4, replication model (ADR-0093), Universal ADT Phase 3 (ADR-0094), persistence enum (ADR-0098), ReadCosts, MySQL/MariaDB (ADR-0080), Pebble sort index, cqrs-lint 179→186 rules + scorecard + group-by + C038-C040 + per-module + JSONC + explain, go-sse consumption (ADR-0097), Nix integration tests (ADR-0095), Iroh bridge (ADR-0096) + QUIC FFI transport |
| v4.2.0       | 2026-07-27 | CBOR→JSON transcoding, 3 new cqrs-lint rules (65 total), coverage-drift checker, CI gates (duplication/layers/api-stability/coverage), wrapClosed consolidation, UP1 test hardening, go-error-family v0.10.0 (6-family)                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                               |
| v4.1.0       | 2026-07-23 | Deprecated API removal, metaengine, benchkit, Increment/Reset rollups, README overhaul, error taxonomy migration, Aggregate→Stream rename (ADR-0058)                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                  |
| v4.0.4       | 2026-07-23 | COSE signing/encryption, multi-batch event store, OTel storage instrumentation, getting-started guide, architecture docs                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                              |
| v4.0.3       | 2026-07-22 | SQL dialect abstraction, stack preset centralization, JSON v2 migration, harmful duplication elimination, cqrs-lint scanner overhaul                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                  |
| v4.0.2       | 2026-07-18 | CBOR time encoding fix, timezone-safe types (Instant, WallTime, Date), cqrs-lint loader error surfacing                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                               |
| v4.0.1       | 2026-07-16 | projectionhost deadlock/leak/sort fix, watermill deadlock fix, storage/view IS NULL+RawWhere+ViewUpdater, cqrs-lint first release (60 rules)                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                          |
| v4.0.0       | 2026-07-11 | CBOR defaults, API cleanup, BackfillHandler consolidation, HealthCheck, storage split, `/v4` path migration                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                           |

---

## Themes

### 1. Metaengine → Production

The metaengine proves the Event-Query Model works: fold return types infer ADTs,
typed closures avoid strings, pagination is detected from input structs. The
production maturity chain is complete:

- ✅ **Real SQLite engine** — `SQLiteEngine` wrapping `SQLViewStore` (ADR-0061)
- ✅ **Cost model calibration** — `EngineProfile.NsPerOp` with benchmark-driven
  constants (Memory=500ns, SQLite=7000ns), now split into `NsPerRead`/`NsPerWrite`
- ✅ **Projection adapter** — `metaengine/projectionadapter` implements
  `projection.Projection` for `projectionhost.Host` (ADR-0062)
- ✅ **Pebble engine** — `metaengine/pebbleengine` with LSM point reads
  (~7x faster than SQLite on MapGet). Separate module (ADR-0074)
- ✅ **Pebble LayoutPlanner** — secondary index with O(matches) prefix scan
  (108x speedup over full scan). Range filters via index bounds.
  Sort index (1,233x speedup via `'o'` prefix key structure)
- ✅ **Raw value readers** — `RawValueReader`/`RawScanReader` skip JSON decode
  for filter/sort/cursor paths (single-pass decode)
- ✅ **SQL pushdown** — `FilterOnField`/`SortOnField` push WHERE/ORDER BY/LIMIT
  into SQLite via `json_extract()` (ADR-0072). pgengine via JSONB `->>`,
  duckdbengine via `json_extract()`
- ✅ **Layout planning** — `LayoutPlan` generates indexed-column DDL from
  declared query fields — 10x speedup on filter+sort (ADR-0073). pgengine
  expression indexes on JSONB paths
- ✅ **Streaming reads** — `StreamScan(ctx) iter.Seq2` for OOM-safe iteration
  (Memory + SQLite + Pebble)
- ✅ **SSE event delivery** — `ServeSSE` with Last-Event-ID reconnection,
  backpressure, dedup ring, byte-budgeted replay
- ✅ **PrefetchCache** — cursor-encoded auto-population, thread-safe
- ✅ **Watcher** — reactive notifications with per-key filtering
- ✅ **Transaction API** — fully threaded `*sql.Tx` through engine ops
- ✅ **ADT test harness** — `adttest.RunMatrix` cross-engine parity tests
  for all 10 ADTs (Map, Set, Counter, Multimap, Log, Graph, SortedMap, Scan,
  Vector, Search, Spatial)
- ✅ **Taskmanager integration** — Counter ADT query with `/api/stats` endpoint
- ✅ **DuckDB engine** — `metaengine/duckdbengine` with MapBackend,
  CounterBackend, PushdownScan. Separate module (ADR-0086)
- ✅ **Postgres engine** — `metaengine/pgengine` with MapBackend, CounterBackend,
  ScanBackend, PushdownScan (JSONB operators), LayoutPlanner (expression
  indexes). Pure Go (pgx). Separate module (ADR-0087)
- ✅ **Vector/Search/Spatial ADTs** (ADR-0085) — k-NN similarity (cosine/
  euclidean/dot), full-text search (TF-IDF inverted index), geo range queries
  (haversine). Memory-only (brute-force); future backends below
- ✅ **Rule pipeline** — `PlanRule` interface + `RulePipeline`. Composable rules
  extracted from monolithic `planner.go` (279→226 lines). 4 rules: schemaRule,
  layoutRule, writeAmpRule (ADR-0083)
- ✅ **Materialize-vs-replay** — `ReplayCost`/`MaterializeCost`/
  `ShouldMaterialize`. Advisory INFO/WARN diagnostics when materialization is
  cheaper than replay. The ES-specific killer feature
- ✅ **StorageLayout + cost matrix** — `Layout{Row, Columnar, LSM, KV}`,
  `(ADT × Layout) → Complexity` mapping, `EngineProfile.Layouts`, `RuleTrace`
- ✅ **SerializablePlan** — JSON-serializable `PlanResult` for diff/pin/round-trip
- ✅ **VersionedStorage** — temporal queries (`ExecuteAsOf`) on Memory engine
- ✅ **Fold sealed interface** — 12 concrete unexported fold types replace the
  11-field `any` god-struct. Zero nil-panic risk
- ✅ **ScanResult explicit HasMore** — across all 5 engines and 3 scan interfaces
- ✅ **Property-based cross-engine parity** — `pgregory.net/rapid` generates
  random operation sequences, verifies Memory and SQLite agree
- ✅ **Postgres testcontainer tests** — first real-DB tests for pgengine
- ✅ **Dead code wiring** — branded unit types (`NsPerRead`, `NsPerWrite`,
  `ByteSize`) have `Valid()` called in `planQuery()`. `ApplyError` wraps fold
  failures in `applyFold()`
- ✅ **Exhaustiveness guard** — `TestApplyFoldExhaustiveness` count check +
  mirror switch catches unhandled fold types at test time
- ✅ **DuckDB LayoutPlanner** — columnar layout planning with type coercion
  and aggregation via `LayoutPlanApplier`
- ✅ **Reification failure tracking** — `IncReificationFailure()` /
  `ReificationFailures()` surfaces type mismatches between planned and stored
  types
- ✅ **Replication model** (ADR-0093) — `EngineProfile.Replication`/
  `ReplicationLag`/`NetworkRTT` (DDIA Ch5). `replicationRule` +
  `mapUpdateReplicationRule` diagnostics. `WithReplication`/`WithNetworkRTT`
  plan options. `CollectionInfo` exposure. `SerializablePlan` replication fields.
  `ExplainPlan`/`Doctor` replication output. Foundation for distributed engines
- ✅ **Universal ADT Phase 3** (ADR-0094) — `DegradedADTs`: all 5 engines
  declare 10/10 ADTs. Non-native ADTs run in O(N) degraded mode. Eliminates
  `ErrUnsupportedADT`. `degradedADTRule` SCREAM diagnostics
- ✅ **WatchTyped** — `WatchTyped[V]`/`WatchTypedWithSeq[V]` typed watcher
  convenience functions
- ✅ **Boundary key validation** — `ErrKeyTypeMismatch` at `Store.Execute`/
  `ExecuteTyped` boundary
- ✅ **CalibrateEngine** — `calibratable` interface; copy-discard bug fixed
- ✅ **AtomicAppender** — `StreamAppendExpected(ctx, collection, streamID,
expectedVersion, entries)` — atomic optimistic concurrency under a single
  lock. Memory + SQLite. `ErrVersionConflict` sentinel
- ✅ **StreamLogBackend** — 5-method interface for stream-keyed event journals.
  Memory + SQLite implementations. Foundation for the `system/` package

**Remaining (short-term, see [TODO_LIST.md](TODO_LIST.md)):** Postgres GIN
indexes, serialize ReadCosts into SerializablePlan, tag metaengine v4.5.0,
fix DuckDB/PG go.mod version drift, sse.go file-size split.

**Remaining (long-term, ROADMAP):**

- **Vector/Search/Spatial engine backends** — currently Memory-only (brute-force).
  DuckDB VSS extension (vector similarity), Postgres tsvector (full-text search),
  PostGIS (spatial). Each is a separate engine module with its own deps.
- **DuckDB columnar-native storage** — DuckDB stores JSON as VARCHAR; columnar
  scans not leveraged. Vectorized GROUP BY for CounterGet would use DuckDB's
  native columnar engine.
- **Postgres GIN containment indexes** — `@>` operator for JSONB path queries.
  Currently only expression indexes (B-tree on JSONB paths).
- **`metaengine-gen` code generator** — `cmd/metaengine-gen` for typed Store
  methods from query declarations. Go AST parsing + template generation.
- **Operator-driven engine selection (close the "deployer decides" gap)** —
  today the _consumer_ opens the `*sql.DB`, sets pragmas, constructs
  `metaengine.NewSQLiteEngine`, and hardcodes the engine list passed to `Plan`
  (see `example/taskmanager/setup.go` + `metaengine.go`: ~140-line
  `taskEventDecoder`, per-event `eventWithID[P]` wrappers, `onTyped` helper).
  The design vision (`meta-engine-design.md` §6) is the opposite: the _operator_
  provides engines (config/registry), the consumer declares query patterns only.
  Target: a `stack.WithMetaEngineFromBundle` (or engine-registry) path where the
  bundle supplies available engines and the consumer never touches a `*sql.DB`.
  Pairs with the deferred auto event-decoder (fold handlers keyed by CQRS event
  type string, removing the parallel decoder switch).

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
  (median-of-N) mitigates it; real-world regression tracking is the next step.
- **Real-world validation** — the first run verified plumbing and plausibility;
  a regression baseline + CI integration is the path to trustworthy numbers.

### 3. cqrs-lint → Trustworthy

The linter grew from 65 to **186 rules** across 10 categories. Quality has been
hardened through multiple brutal review passes and 7 consumer feedback rounds.

- ✅ **186 rules shipped** across correctness (40), API (31), boilerplate (28),
  adoption (21), architecture (17), consistency (16), security (10), performance
  (9), testing (8), version (6)
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
- **Publish v4.4.0** — all fixes in source but version constant still "4.3.0".
  See [TODO_LIST.md](TODO_LIST.md).
- **Run against real consumers** — validate false-positive rates against live
  projects. See [TODO_LIST.md](TODO_LIST.md).

### 4. Module Extraction

Two modules extracted to standalone repos (see
[extraction analysis](docs/planning/2026-07-23_extraction-analysis.md)):

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
- [ ] **Regression baseline + CI integration** — calibration benchmarks should run
      in CI and fail if constants drift >3×.

### 9. Deferred Debt (ADR-committed)

Two items explicitly committed to in the 2026-08-03 ADR review as "the next
real roadmap." Each has a clear ADR with rationale.

- [ ] **Ghost bus removal** (ADR-0028) — delete `memory/bus.go`,
      `memory/command_bus.go`, `storage/pg_bus.go`. Largest blast radius — audit
      ALL consumer repos first.
- [ ] **Metadata aliases completion** (ADR-0031) — `command.Metadata` /
      `query.Metadata` → standalone structs (currently repointed aliases with
      functional `WithCustom`, but not yet fully standalone types).

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
- [ ] Evaluate `iroh-go` C binding stability (third-party binding for Iroh Rust)
- [ ] Tag loopback + quic modules
- [ ] WriteOp.ID dedup ring on loopback path (quic has it)

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

🧪 **First pass + Pareto P0/P1 fixes shipped** (`system/v4` module).
DomainConfig/DeploymentConfig separation, Op[State] routing, driver registry
(wired — SQLite works through `New()`), EventAdapter/CommandAdapter/QueryAdapter,
simpleBus + MultiBus (both wired into `New()`), CachedEventStore, SnapshotBackend
(wired with codec + strategy), scream store (wired into `New()`), introspection
API (real health checks), YAML config parsing, System.Verify/Plan/Explain,
projection decoder wiring (`ProjectionTypeDecoder`/`ProjectionEventDecoder`).
All 5 engines implement StreamLogBackend (Memory, SQLite, Pebble, DuckDB,
Postgres). DuckDB + Postgres have AtomicAppender.

**⚠️ Remaining gaps:** Three files exceed the 350-line CI limit (constructor.go
382, system.go 364, adapter_event.go 357). Scream store has 2 of ~12 rules.
CommandAdapter/QueryAdapter serialization for SQL engines unstarted. Example
migration not done. See [TODO_LIST.md](TODO_LIST.md) → System Package.

---

## Raw Ideas (No Design Yet)

> _Triage 2026-08-04: 14 items reviewed. `cqrs-lint init` SHOWSTOPPER removed (fixed). None stale._

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
- Auto-denormalization in metaengine — planner detects that two queries share
  a common prefix and recommends a denormalized projection to avoid fan-out
  joins at read time.
- Metaengine plugin registry — third-party engine backends registered at
  runtime without recompiling (operator YAML config for engine selection).
- CALM theorem ADR for metaengine — document why monotonic folds are CRDT-safe
  for replicated engines (supports Iroh integration, ADR-0096).
- `CalibrateScanEngine` — runtime calibration for scan/aggregation costs (not
  just point lookups; `CalibrateEngine` only measures `MapGet`).

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
