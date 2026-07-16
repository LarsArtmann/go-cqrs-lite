# TODO List

**Updated:** 2026-07-16 (docs-health audit)
**Scope:** Short- and mid-term actionable tasks only. Long-term vision lives in [ROADMAP.md](ROADMAP.md).

## Legend

- `[ ]` = Open
- `[x]` = Done (moved to [CHANGELOG.md](CHANGELOG.md))
- `[BLOCKED]` = Blocked on upstream dependency or user approval
- `🔥` = High impact (top 20% that delivers 80% of value)

## Recently Completed

> **50+ items resolved during July 2026 sessions.** Full list in
> [CHANGELOG.md](CHANGELOG.md) `[Unreleased]` → "Resolved During July 2026 Sessions".
> Key highlights: v4.0.0 shipped, cqrs-lint built (60 rules), all DiscordSync
> feedback gaps closed, ROADMAP/TODO/FEATURES docs audited and fixed.

---

## v4.0.0 — SHIPPED

> v4.0.0 tagged and pushed. All per-module tags exist. See [CHANGELOG.md](CHANGELOG.md) `[4.0.0]` for details.

---

## Priority 1 — Consumer Experience

- [ ] 🔥 **Publish `eventtest` to Go proxy as `v0.1.0`** — The #1 consumer pain point across
      ALL feedback rounds (DiscordSync ×3, SwettySwipper ×2). Tag exists locally but is
      not pushed. Run `git push origin event/v4/eventtest/v0.1.0`, then verify proxy fetch.
      Also delete the wrong `event/v4/eventtest/v4.0.0` tag (violates Go versioning rules).
- [ ] 🔥 **README "sales page" rewrite** — Per docs-health model, README should be the
      end-user entry point: what this does, why it exists, how to get started in 3 steps.
      Currently mixes internal docs with user-facing content. ~90min.
- [ ] **CBOR-stamp tests for gRPC + watermill** — Cross-encoding round-trip tests proving
      CBOR-stamped events survive transport. Only SSE has this coverage today. ~45min.
- [ ] **Pre-commit hooks** — `fmt.Printf` ban in prod packages, api_surface.txt regen check,
      `nix fmt --fail-on-change`. Via flake.nix. ~60min.

---

## Priority 2 — Post-v4.1 (breaking changes, new major version)

- [ ] **Deprecated API removal batch 2** — Remove 9 deprecated items: `middleware.{NewMetrics,
CommandMetrics, EventMetrics, QueryMetrics, MetricsRecorder, Observe}`,
      `catalog.Exporter` (non-generic), `storage/sql.{NewDBHandle, NewDBHandleFromDB}`.
      Breaking → v4.1 cut. ~60min.
- [ ] **Postgres CI coverage matrix** — Add CI Postgres service or label `stack/postgres`
      experimental. ~60min.

---

## Priority 3 — Public Release Readiness (NEEDS USER APPROVAL)

> **These are irreversible. Do NOT execute without explicit user approval.**

- [ ] [BLOCKED] **License swap (PROPRIETARY → Apache-2.0)** — Hard blocker for public adoption.
      **Needs user approval (irreversible).**
- [ ] [BLOCKED] **Git history scrub for internal docs** — AGENTS.md, docs/planning/\* contain
      internal strategy. **Needs user approval (irreversible).**

---

## Future — v4.1+ Parquet Journal + DuckDB

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

## Experimental / Go-stdlib-blocked

- [BLOCKED] **Remove `goexperiment.jsonv2` tag** — JSON v2 is fully adopted (~25 production files
  import `encoding/json/v2`). The build tag remains only because Go 1.26 hasn't graduated json/v2
  from experimental. Remove the tag when Go stabilizes it (expected Go 1.27+).
- [BLOCKED] **Turso MVCC concurrent-write support** — Blocked on upstream experimental MVCC.

---

## Rejected (with reasons)

- **Strengthen envelope magic string (`"cqrs"` → `"cqrs-envelope-v1"`)** — The `"$"` JSON key
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
- **Split event/ module** — 27 importers, real cohesion. Explicitly decided in v4.

---

_All completed items have been moved to [CHANGELOG.md](CHANGELOG.md) under `[4.0.0]` or `[Unreleased]`._
