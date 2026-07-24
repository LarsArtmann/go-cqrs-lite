# TODO List

**Updated:** 2026-07-24
**Scope:** Short- and mid-term actionable tasks only. Long-term vision lives in
[ROADMAP.md](ROADMAP.md). Completed work lives in [CHANGELOG.md](CHANGELOG.md)
and is **never** duplicated here — when a task is finished it is removed from
this list and recorded in CHANGELOG.

## Legend

- `[ ]` = Open
- `[BLOCKED]` = Blocked on upstream dependency or user approval
- `🔥` = Pareto high impact (top 20% that delivers 80% of value)
- `⭐` = Top 1% impact (do first)

---

## Benchkit

- [ ] 🔥 **Tag `benchkit/v0.1.0`** when the API stabilizes. Currently unreleased
      (no git tag); `metaengine`, `cmd/cqrs-bench`, and `example/readme-quickstart`
      are in the same state and tag together when ready.

> The full benchmark suite shipped unreleased — durability/recovery, production
> replay, `benchtest.RunSuite`, analytical profile, Postgres backend, scaling
> sweeps, benchstat, manifest, and profiling. See [CHANGELOG.md](CHANGELOG.md)
> `[Unreleased]` and [FEATURES.md](FEATURES.md#benchmarking-toolkit-) for what
> exists. The first real benchmark run completed 2026-07-24 across
> memory/pebble/sqlite — see
> [benchmark results](docs/status/2026-07-24_17-54_benchmark-first-real-run.md).

---

## Metaengine → Production

> `metaengine/v4` is experimental (🧪): MemoryEngine only, 87.7% coverage, 174
> BDD specs, zero production deps. The path from prototype to production:

- [ ] ⭐ **Real SQLite engine** — wrap `SQLViewStore` as a metaengine backend.
      The first production engine validates the interface design.
- [ ] **Projection adapter** — `metaengine` has no `projection.Projection`
      adapter. Integration tests would validate the design against the existing
      projection host.
- [ ] **Cost model calibration** — `nsPerOp=100` is arbitrary; needs
      benchmark-driven calibration. Scale thresholds warn but don't auto-switch
      structures.
- [ ] **FilterOn/SortOn → SQL pushdown** — Go closures cannot be inspected.
      Design decision needed: DSL, codegen, or keep in-memory filtering.
- [ ] **Resolve `event/` dependency** — `metaengine` is zero-dependency by
      design. If `event.Event` integration is needed, resolve the go.sum checksum
      issue or keep the boundary.

---

## Consumer Experience

> Gaps surfaced by the [book insights vs codebase review](docs/architecture-understanding/2026-07-23_book-insights-vs-codebase.html).

- [ ] **Read-your-writes helper** — `WaitForVersion(ctx, streamID, version)` for
      consumers who need immediate consistency after a write.
- [ ] **Bounded staleness** — `WithMaxStaleness(duration)` for projections that
      can tolerate lag.
- [ ] **Consistency model document** — `docs/CONSISTENCY_MODEL.md` documenting
      single-process scope and eventual consistency between write/read models.
- [ ] **SQL-backed `idempotency.Store`** — for multi-process Postgres deployments
      (~100 lines: `INSERT ON CONFLICT DO NOTHING`).

---

## Rejected (with reasons)

> Kept here so decisions are not re-litigated. Full rationale in the linked
> ADRs/reviews.

- **Strengthen envelope magic string (`"cqrs" → "cqrs-envelope-v1"`)** — the `"$"`
  JSON key provides 99% of collision avoidance. Extra bytes per record for
  near-zero benefit.
- **Composite keys in `SQLViewStore`** — breaks `K fmt.Stringer`. Composite keys
  are relational territory — use `RelationalProjection` (junction tables,
  multi-table atomic writes). See ADR-0033.
- **OR conditions / query builder in ViewStore** — `RawWhere` covers the 5% case.
  `OrClause`/`NotClause`/nested groups is ORM creep. Principle #1: "Library, not
  framework."
- **Unify VersionedStore + VersionedSeekableJournal** — different interfaces
  (Store: Load/Save per stream, SeekableJournal: ReadFrom position-based). YAGNI.
- **VersionedJournal (ReadAll only)** — no consumer needs `ReadAll` with
  upcasters. YAGNI.
- **Expose `SSEBroker.PayloadTransform()` accessor** — implemented for
  BackfillHandler (necessary), but no standalone demand.
- **`WithPayloadTransform` on SSEHandler** — duplicates responsibility (SRP
  violation); SSEHandler wraps the broker.
- **Auto-apply CQRS views by default** — violates "library, not framework."
- **VersionedSeekableJournal implementing event.Store** — different scope
  (position-based vs stream-based). YAGNI.
- **Integration test in `integration/` module** — redundant with
  `projectionhost/versioned_journal_integration_test.go`.
- **`storage/auditstore/` package** — lying name. Renamed to "dispatch log" and
  kept in `storage/`.
- **Split `event/` module** — 27 importers, real cohesion. Explicitly decided in
  v4.
- **RollupSpec / RollupProjection** — premature abstraction. `sink.Increment` is
  the composable primitive. See [analytics rollup review](docs/feedback/reviewed/2026-07-23_analytics-rollup-support-review.md).
- **IncrementWhere on ProjectionSink** — footgun: silently updates multiple rows.
  Use `RelationalProjection` with explicit `Upsert`.
- **Redis adapter** — see ROADMAP Non-Goals (ValKey/NATS/Kafka preferred).

---

_Long-term direction (Parquet journal + DuckDB, transport expansion, module
extraction, goexperiment.jsonv2 / Turso MVCC blockers) lives in
[ROADMAP.md](ROADMAP.md). Completed work is in [CHANGELOG.md](CHANGELOG.md)._
