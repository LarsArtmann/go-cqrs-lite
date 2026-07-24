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

- [ ] **`storage/sql/errors.go` message prose** — Error codes correctly kept as
      stable match keys (`storage.aggregate_type_mismatch`), but the
      human-readable messages still say "aggregate" (lines 24, 35). Same fix
      pattern as the pebble fix.
- [ ] **Stale test diagnostics** — `integration/command/command_test.go:27`,
      `integration/pebble/pebble_test.go` (4 lines),
      `transport/grpc/command_span_test.go:99`.
- [BLOCKED] **OTel attribute string values** — `otel/attributes.go` keeps
      `cqrs.aggregate.*` string values for dashboard compatibility while the Go
      constants are renamed (`AttrStreamType`, etc.). Renaming the strings is a
      **breaking change for consumer dashboards/queries** — needs explicit
      decision. Also blocks `middleware/tracing_test.go` (4 attribute-name
      assertions on lines 57, 119, 123, 127).
- [BLOCKED] **`catalog/d2.AggregateRoot`** — Exported field + "Aggregate Root"
      diagram label not in the ADR-0058 rename map. It is a DDD diagram concept;
      needs decision on whether to rename.
- [ ] **Run full quality gates** — `nix run .#lint` and full test suite not yet
      run after the comment/prose changes (text-only, but gate should pass).

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

**Open:**

- [ ] 🔥 **Run-to-run variance is ~20-25% (memory backend)** — Single-run throughput numbers are unreliable; uniform-1024 measured 92K-190K/s across runs. Cold first-runs can be 2x outliers. Pebble is far more stable (mixed workload ±2%). **Need a `--repeat N` flag** that runs N iterations and reports median + min/max spread. This is the highest-impact benchkit gap — without it, all absolute numbers are suspect. See [scaling report](docs/status/2026-07-24_19-30_event-size-scaling-benchmark.md).
- [ ] **Implement `DiskSize()` on `pebble.Bundle`** — `DiskSizer` interface exists but zero backends implement it. Pebble backend has `Metrics().DiskUsage()` available.
- [ ] **CPU measurement returns n/a** — Fast benchmarks (memory backend, <3ms) complete between polling intervals. Need CPU start+end measurement, not just polling.
- [ ] **Projection benchmark** — Projection phase now runs (10K events on small profile) but only intermittently appears in output (timing race: memory projection catches up within the 10ms poll window sometimes). Stabilize the projection phase reporting.
- [ ] **Phase 2: durability benchmark** — Crash recovery, replay-after-restart.
- [ ] **Phase 6: production replay** — Replay real event streams for benchmarking.
- [ ] **Phase 7: `benchtest.RunSuite`** — Preset integration for `stack/bench`.
- [ ] **Analytical benchmark profiles** — Profiles for read-heavy analytical workloads (OLAP-style queries).
- [ ] **Postgres benchmark tests** — `stack/postgres` tests skip without `POSTGRES_TEST_DSN`.
- [ ] **Missing edge-case tests** — Compare failure isolation, Concurrency override, journal scan metrics, CLI codec/warmup/output flags, unknown profile/backend error paths.
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
  aggregate, SeekableJournal: ReadFrom position-based). YAGNI.
- **VersionedJournal (ReadAll only)** — No consumer needs `ReadAll` with upcasters. YAGNI.
- **Expose `SSEBroker.PayloadTransform()` accessor** — Implemented for BackfillHandler (necessary),
  but no standalone demand from consumers.
- **`WithPayloadTransform` on SSEHandler** — SSEHandler wraps the broker; adding transform there
  duplicates responsibility (SRP violation).
- **Auto-apply CQRS views by default** — Violates "library, not framework." Consumers choose
  their histogram boundaries.
- **VersionedSeekableJournal implementing event.Store** — Different scope (position-based vs
  aggregate-based reads). YAGNI.
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
