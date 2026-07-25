# TODO List

**Updated:** 2026-07-25
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

## CI Quality Gate (🔴 blocks `nix run .#verify`)

> The verify gate is currently RED. These items must be resolved before the next
> release tag. Verify with `nix run .#verify` (build + vet + test + race + lint).

- [ ] ⭐ **Split 13 oversized production files** (>350-line CI limit, excluding
      tests/generated). Worst offenders:
  - `benchkit/phases.go` (610), `benchkit/runner.go` (498)
  - `cmd/cqrs-bench/main.go` (602), `cmd/cqrs-lint/main.go` (452)
  - `cmd/cqrs-lint/pkg/analyzer/scanner_calls.go` (412), `scanner.go` (387)
  - `projectionhost/host.go` (403), `storage/relational/sink.go` (378)
  - `codec/cose.go` (376), `graph/schema.go` (368), `benchkit/benchkit.go` (368)
- [ ] 🔥 **Fix otel test flakiness** — global provider state leaks across test
      packages. Needs per-test provider isolation or a reset guard.
- [ ] **Restore truncated sentinel error messages** — several sentinels were
      shortened to satisfy test regex assertions during the 06-30 lint cleanup;
      they should be complete sentences again (update the assertions instead).

---

## Module Tagging

- [ ] 🔥 **Tag `metaengine/v4`**, `metaengine/projectionadapter/v4`, and
      `idempotency/sqlstore/v4` when their APIs stabilize and the file-size gate
      passes. These are the 3 untagged modules (58 `go.mod` files total; 55 tagged).
- [BLOCKED] **Push `benchkit/v4.1.0` to origin** — tag points to grab-bag commit
  `c3286bc8` (BuildFlow auto-commit shoved 16 unrelated files in). Decide:
  keep the tag as-is or recreate at a cleaner commit. Push requires user
  approval. See
  [tagging session status](docs/status/2026-07-25_04-54_benchkit-v4.1.0-tagging-session.md).

---

## Documentation Cross-links

> Minor gaps from the Pareto execution; the features shipped but the docs don't
> all point at each other.

- [ ] **Cross-link `docs/CONSISTENCY_MODEL.md`** from `docs/README.md` index.
- [ ] **Add ADR index links** in `AGENTS.md` for ADR-0061 through ADR-0065
      (metaengine pushdown, projection adapter, cost calibration, retry/idempotency
      extraction).
- [ ] **Reference NATS + Parquet design docs** in the Crush skill
      (`.agents/skills/go-cqrs-lite/references/recipes.md`) so consumers discover them.

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

_Long-term direction (module extraction execution, NATS/Parquet implementation,
benchkit journey benchmarks, metaengine Phase 2 pushdown, goexperiment.jsonv2 /
Turso MVCC blockers) lives in [ROADMAP.md](ROADMAP.md). Completed work is in
[CHANGELOG.md](CHANGELOG.md)._
