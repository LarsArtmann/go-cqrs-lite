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

## Verify Gate — confirm GREEN end-to-end

> The file-size gate is GREEN (all 11 oversized files split under 350 lines).
> The otel flakiness is fixed (`-race -count=10` clean). However the **full**
> `nix run .#verify` has not been run in one pass since the splits — lint,
> doc-check, and module-coverage sub-checks are unconfirmed on the 16 new files.

- [ ] ⭐ **Run `nix run .#verify` end-to-end** and fix anything red (lint nits
      on split files, doc-check symbol drift, module-coverage gaps).
- [ ] **gofmt all 16 new split files** in one pass (`gofmt -w`), confirm zero diff.
- [ ] **Document `otel.WithoutGlobalRegistration()`** in `AGENTS.md` OTel section
      + Crush skill `references/core.md` — new public API added during the otel
      flakiness fix, currently undocumented for consumers.

---

## Module Tagging

> 2 of 3 untagged modules tagged locally. `benchkit/v4.1.0` was pushed to origin.
> 56 of 58 modules now have tags (55 pushed + 1 local-only).

- [ ] 🔥 **Push `metaengine/v4.0.0` and `idempotency/sqlstore/v4.0.0`** to origin
      (tags exist locally, annotated, release-clean go.mod). Requires user
      push authorization.
- [BLOCKED] **Tag `metaengine/projectionadapter/v4.0.0`** — its `go.mod` has a
      local `metaengine/v4 => ../` replace; cannot resolve metaengine from the
      Go proxy until `metaengine/v4.0.0` is pushed. After the push above, run
      `./scripts/tag-release.sh metaengine/projectionadapter v4.0.0 "..."`.

---

## Release Tooling

- [ ] **Audit `scripts/tag-release.sh`** for other `pipefail` traps like the one
      fixed this session (grep `-P` no-match on non-cqrs replace directives
      aborted the whole release under `set -euo pipefail`). Consider `--dry-run`
      mode and single-module tagging (currently touches all 58 go.mod files).

---

## Documentation Health

- [ ] **Update `docs/README.md` ADR index** — the table stops at ADR-0035, then
      jumps to ADR-0046. ADRs 0036–0065 are missing from the index (files exist
      in `docs/adr/`).
- [ ] **Add the 3 newly-tagged modules** (`metaengine/v4`, `idempotency/sqlstore/v4`,
      `metaengine/projectionadapter/v4` once tagged) to `FEATURES.md` status table.

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
