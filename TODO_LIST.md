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

### Aggregate→Stream Rename Follow-ups

- [x] **Comment cleanup** — 41 files cleaned across decider/, listing/,
      storage/pebble/, storage/memory/, event/, snapshot/, command/.
      Remaining: `storage/` root (3 lines), `storage/turso/` (3 lines).
- [x] **SKILL.md references** — All 35 mentions cleared across 7 files
      (core.md, advanced.md, recipes.md, modules.md, readmodels.md, faq.md,
      root SKILL.md). doc-check validates 897 references.
- [ ] **AGENTS.md cleanup** — 16 stale "aggregate" references in code
      examples, module tree comments, and prose (lines 52, 78, 138, 156-170,
      231, 241, 401-422, 473-474, 789).
- [ ] **Storage root + turso comments** — 6 stale comment lines in
      `storage/command_store_journal.go`, `storage/command_store_load.go`,
      `storage/pg_bus_test.go`, `storage/turso/indexing/advisor_test.go`,
      `storage/turso/indexing/doc.go`.
- [ ] **Run full quality gates** — `nix run .#lint` and full test suite
      not yet run after comment changes (comment-only, but gate should pass).

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

- [ ] **Run the benchmark and inspect output** — 55 tests verify plumbing (event
      counts, sample counts, error classification) but no test validates that
      throughput/latency numbers are physically plausible. The benchmark has never
      been actually executed and inspected.
- [ ] **Fix `--version` drift** — Hardcoded `v4.1.0` in `cmd/cqrs-bench/main.go`;
      should use `runtime/debug.ReadBuildInfo()` for automatic versioning.
- [ ] **Implement `DiskSize()` on `pebble.Bundle`** — `DiskSizer` interface exists
      but zero backends implement it. All disk measurement falls back to filesystem walk.
- [ ] **Phase 6: Production replay** — Replay real event streams for benchmarking.
- [ ] **Phase 7: benchtest.RunSuite** — Preset integration for `stack/bench`.
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
