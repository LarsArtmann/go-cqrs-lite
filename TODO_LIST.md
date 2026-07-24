# TODO List

**Updated:** 2026-07-24
**Scope:** Short- and mid-term actionable tasks only. Long-term vision lives in [ROADMAP.md](ROADMAP.md).
Completed work lives in [CHANGELOG.md](CHANGELOG.md).

## Legend

- `[ ]` = Open
- `[BLOCKED]` = Blocked on upstream dependency or user approval
- `🔥` = Pareto high impact (top 20% that delivers 80% of value)
- `⭐` = Top 1% impact (do first)

---

## 20% Tier — Important, Can Queue

### Aggregate→Stream Rename Follow-ups (ADR-0058)

**Done:**

- [x] **Comment + prose cleanup** — ~70 files across event/, command/, decider/,
      listing/, snapshot/, storage/memory/, storage/pebble/, id/ (error message
      prose, BDD narratives, function names, internal helper renames).
- [x] **SKILL.md references** — All 35 mentions cleared across 7 files
      (core.md, advanced.md, recipes.md, modules.md, readmodels.md, faq.md,
      root SKILL.md). doc-check validates 897 references.
- [x] **AGENTS.md cleanup** — All stale references updated (module tree, code
      examples, design principles). Only 1 mention remains (an exported API name).
- [x] **Deprecated-alias compat tests** — `id/compat_aliases_test.go` and
      `event/compat_aliases_test.go` verify all `Aggregate*` aliases still work.
- [x] **Storage root + turso comments** — stale comment lines cleaned in
      `storage/command_store_*.go`, `storage/pg_bus_test.go`,
      `storage/turso/indexing/`.

**Open:**

- [x] **`storage/sql/errors.go` message prose** — Done (Session 3). Error codes kept
      as stable match keys; human-readable messages updated.
- [x] **Stale test diagnostics** — Done (Session 3 + Session 4). All test files
      cleaned (224 files: variable names, comments, assertion messages,
      function names, stream type labels).
- [RESOLVED] **OTel attribute string values** — Decision (Session 3): KEEP
  `cqrs.aggregate.*` string values for dashboard compatibility. The Go
  constants are renamed (`AttrStreamType`, etc.). Documented in source.
- [RESOLVED] **`catalog/d2.AggregateRoot`** — Decision (Session 3): KEEP as-is.
  It is a DDD diagram concept (Aggregate Root label in D2 diagrams), not a
  stream-key naming issue. Would need separate ADR.
- [x] **Run full quality gates** — Done (Session 4). `nix run .#verify`,
      `nix run .#lint`, `nix run .#check-layers` all pass. Race tests pass
      individually (4 flaky under concurrent pressure — pre-existing).

### Metaengine Integration

- [ ] **Projection adapter** — `metaengine` has no `projection.Projection` adapter.
      It is a ghost system with zero consumers inside the repo (expected for a library,
      but integration tests would validate the design).
- [ ] **Real SQLite engine** — Only `MemoryEngine` is implemented. SQLite engine
      wrapping `SQLViewStore` is the first production backend.
- [ ] **Cost model calibration** — `nsPerOp=100` is arbitrary; needs benchmark-driven
      calibration. Scale thresholds warn but don't auto-switch structures.
- [ ] **Resolve `event/` dependency** — `metaengine` is zero-dependency by design.
      If `event.Event` integration is needed, resolve the go.sum checksum issue or
      keep the boundary.

### Book Insights Gaps (from [architecture review](docs/architecture-understanding/2026-07-23_book-insights-vs-codebase.html))

- [ ] **Read-your-writes consistency helper** — `WaitForVersion(ctx, aggID, version)`
      for consumers who need immediate consistency after a write.
- [ ] **Bounded staleness option** — `WithMaxStaleness(duration)` for projections
      that can tolerate lag.
- [ ] **Consistency model document** — `docs/CONSISTENCY_MODEL.md` documenting
      single-process scope, eventual consistency between write and read models.
- [ ] **SQL-backed `idempotency.Store`** — For multi-process Postgres deployments
      (~100 lines: `INSERT ON CONFLICT DO NOTHING`).

### Benchkit Reliability

> First real benchmark run completed 2026-07-24 — see
> [benchmark results](docs/status/2026-07-24_17-54_benchmark-first-real-run.md)
> and [Pareto plan](docs/planning/2026-07-24_17-59_benchkit-hardening-pareto-plan.md).

**Done this session:**

- [x] **Run the benchmark and inspect output** — Executed across 3 backends, 7 profiles, CBOR vs JSON. 6 findings documented.
- [x] **SQLite concurrent-write fix** — Added `storage.ConfigureSQLitePool(sqlDB)` to `stack/sqlite/preset.go` (was missing, caused SQLITE_BUSY at 4+ goroutines).
- [x] **Compare-mode disk = 0B** — `compareCmd` now collects per-backend diskPaths instead of discarding them.
- [x] **Fix `--version` drift** — Now uses `runtime/debug.ReadBuildInfo()` instead of hardcoded string.
- [x] **Mixed payload-size support** — `Generator` now holds a size distribution; `NewMixedGenerator(seed, sizes, codec)` picks a size uniformly at random per event. CLI flag `--payload-sizes 64,256,4096` overrides `--payload-size`. Result reports the distribution mean + full distribution. See [scaling report](docs/status/2026-07-24_19-30_event-size-scaling-benchmark.md).

**Done:**

- [x] 🔥 **`--repeat N` flag** — Added `Config.Repeat` + `runRepeated()` logic: runs N iterations, reports median result with min/max throughput spread. CLI `--repeat N` available on `run` and `compare`.
- [x] **Implement `DiskSize()` on `pebble.Bundle`** — 3-layer DiskSizer: `storage/pebble.Backend.DiskUsage()` (computed from Metrics), `stack.WithDiskSize()` option, wired in `stack/pebble` preset.
- [x] **CPU measurement returns n/a** — Replaced `/proc/self/stat` (10ms resolution) with `syscall.Getrusage` (microsecond resolution). Split into `cpu_unix.go` / `cpu_other.go`.
- [x] **Projection benchmark** — Added polling loop (10ms ticker, 30s deadline) in `projectionPhase`. Projection events now reliably > 0.
- [x] **Missing edge-case tests** — 10 tests added: ConcurrencyOverride, SQLite ReadFromTime, Pebble DiskSizer, CLI unknown profile/backend, codec CBOR, warmup, output file, compare disk non-zero, version.
- [x] **Run the benchmark and inspect output** — Executed across 3 backends, 7 profiles, CBOR vs JSON. 6 findings documented.
- [x] **SQLite concurrent-write fix** — Added `storage.ConfigureSQLitePool(sqlDB)` to `stack/sqlite/preset.go` (was missing, caused SQLITE_BUSY at 4+ goroutines).
- [x] **Compare-mode disk = 0B** — `compareCmd` now collects per-backend diskPaths instead of discarding them.
- [x] **Fix `--version` drift** — Now uses `runtime/debug.ReadBuildInfo()` instead of hardcoded string.
- [x] **Mixed payload-size support** — `NewMixedGenerator(seed, sizes, codec)`. CLI `--payload-sizes 64,256,4096`.
- [x] **Phase 2: durability benchmark** — `Config.Recovery` + `recoveryPhase`: close bundle, reopen via factory, reload all streams. `Result.RecoveryTime` + `RecoveredEvents`. CLI `--recovery`.
- [x] **Phase 6: production replay** — `Config.ReplayOnly`: skip writes, discover streams from Journal/SeekableJournal, benchmark reads + projections on existing data. CLI `--replay`.
- [x] **Phase 7: `benchtest.RunSuite`** — `benchkit.RunSuite(b, config, factory)` wraps benchkit into Go `testing.B` with `b.ReportMetric`. Wired into `stack/bench` with 3 backend suites.
- [x] **Analytical benchmark profiles** — `ProfileAnalytical` (10K streams, 90% reads, 5x journal scans) + `Profile.JournalScans` field for multi-pass journal scanning.
- [x] **Postgres benchmark tests** — `postgres` backend added to `cqrs-bench` CLI. Benchkit tests skip without `POSTGRES_TEST_DSN`.
- [x] **Projection benchmark with real kv.Store handler** — Replaced no-op handler with `newKVCountingProjection`: Get+Set per event on `bundle.ReadModels` (kv.Store). Falls back to atomic counter when no kv.Store.

**Open:**

- [ ] **Tag `benchkit/v0.1.0`** when API stabilizes.

---

## Future — v4.2+ Parquet Journal + DuckDB

> Design complete at `docs/research/archive/2026-07-11_PARQUET_JOURNAL_DUCKDB_MATERIALIZATIONS.md`.
> Three independent phases, all additive (no breaking changes).

- [ ] **Phase 1: `storage/parquet`** — Parquet segment journal (`SeekableJournal`). Pure Go
      (`parquet-go/parquet-go`), no CGO. Segment-based append-only log with manifest index.
- [ ] **Phase 2: `storage/duckdb`** — DuckDB connector + `DuckDBDialect` (11 methods). Unlocks
      `SQLViewStore` + `RelationalProjection` for OLAP-grade materializations. Requires CGO.
- [ ] **Phase 3: `stack/duckdb`** — Preset wiring: DuckDB materializations + optional Parquet
      journal. The "lakehouse for events" pattern.

---

## Future — Transport Expansion

- [ ] **NATS/ValKey Stream adapter** — ADR-0025 accepted. Separate `transport/nats/` and
      `transport/redis/` modules.
- [ ] **Distributed event bus** — No multi-process backend for event distribution.

---

## Future — Module Extraction

> Analysis at `docs/planning/2026-07-23_extraction-analysis.md`.

- [ ] **Extract `retry/` → `go-retry`** — Best candidate: 217 LOC, zero CQRS coupling,
      1 dependency. Standalone repo.
- [ ] **Extract `idempotency/` → `go-idempotency`** — Strong candidate: 355 LOC, zero CQRS
      coupling, 3-method `Store` interface. Standalone repo.

---

## Experimental / Go-stdlib-blocked

- [BLOCKED] **Remove `goexperiment.jsonv2` tag** — JSON v2 is fully adopted (~25 production files
  import `encoding/json/v2`). The build tag remains only because Go 1.26 hasn't graduated json/v2
  from experimental. Remove the tag when Go stabilizes it (expected Go 1.27+).
- [BLOCKED] **Turso MVCC concurrent-write support** — Blocked on upstream experimental MVCC.

---

## Rejected (with reasons)

- **Strengthen envelope magic string (`"cqrs" → "cqrs-envelope-v1"`)** — The `"$"` JSON key
  provides 99% of collision avoidance. Extra bytes per record for near-zero benefit.
- **Composite keys in `SQLViewStore`** — Breaks `K fmt.Stringer` type parameter. Composite keys
  are relational territory — use `RelationalProjection` (supports junction tables, multi-table
  atomic writes). See ADR-0033.
- **OR conditions / query builder in ViewStore** — `RawWhere` escape hatch covers the 5% case.
  Building `OrClause`/`NotClause`/nested groups is ORM creep. Principle #1: "Library, not framework."
- **Unify VersionedStore + VersionedSeekableJournal** — Different interfaces (Store: Load/Save per
  stream, SeekableJournal: ReadFrom position-based). YAGNI.
- **VersionedJournal (ReadAll only)** — No consumer needs `ReadAll` with upcasters. YAGNI.
- **Expose `SSEBroker.PayloadTransform()` accessor** — Implemented for BackfillHandler (necessary),
  but no standalone demand from consumers.
- **`WithPayloadTransform` on SSEHandler** — SSEHandler wraps the broker; adding transform there
  duplicates responsibility (SRP violation).
- **Auto-apply CQRS views by default** — Violates "library, not framework." Consumers choose
  their histogram boundaries.
- **VersionedSeekableJournal implementing event.Store** — Different scope (position-based vs
  stream-based reads). YAGNI.
- **Integration test in `integration/` module** — Redundant with
  `projectionhost/versioned_journal_integration_test.go`.
- **`storage/auditstore/` package** — Lying name. Renamed to "dispatch log" and kept in `storage/`.
- **Split `event/` module** — 27 importers, real cohesion. Explicitly decided in v4.
- **RollupSpec / RollupProjection** — Premature abstraction. `sink.Increment` is the composable
  primitive; consumers compose it directly. See [analytics rollup review](docs/feedback/reviewed/2026-07-23_analytics-rollup-support-review.md).
- **IncrementWhere on ProjectionSink** — Footgun: can silently update multiple rows. Use `RelationalProjection`
  handler with explicit `Upsert`.

---

_All completed items are recorded in [CHANGELOG.md](CHANGELOG.md) under `[4.1.0]` or earlier versions._
