# TODO List

**Updated:** 2026-07-12 (post-v4 comprehensive planning session)
**Scope:** Short- and mid-term actionable tasks only. Long-term vision lives in [ROADMAP.md](ROADMAP.md). Raw ideas live in [ROADMAP.md § Raw Ideas](ROADMAP.md#raw-ideas-no-design-yet).
**Full plan:** [`docs/planning/2026-07-12_14-18_POST-V4-COMPREHENSIVE-PLAN.md`](docs/planning/2026-07-12_14-18_POST-V4-COMPREHENSIVE-PLAN.md) — Pareto breakdown, micro-tasks, mermaid graph.

## Legend

- `[ ]` = Open
- `[x]` = Done (moved to [CHANGELOG.md](CHANGELOG.md))
- `[BLOCKED]` = Blocked on upstream dependency or user approval
- `🔥` = High impact (top 20% that delivers 80% of value)

---

## v4.0.0 — SHIPPED

> v4.0.0 tagged and pushed. All per-module tags exist. See [CHANGELOG.md](CHANGELOG.md) `[4.0.0]` for details.

---

## Priority 1 — The 1% that delivers 51%

- [ ] 🔥 **Publish `eventtest` to Go proxy as `v0.1.0`** — The #1 consumer pain point across
      ALL feedback rounds (DiscordSync ×3, SwettySwipper ×2). Every consumer must manually
      `replace` it in go.mod. Tag + push, verify proxy fetch works. ~30min.

---

## Priority 2 — The 4% that delivers 64%

- [ ] 🔥 **Archive 565 session files to `docs/archive/`** — `docs/status/` and `docs/planning/`
      called "overwhelming" and "unnavigable" by both consumers. `git mv` timestamped files,
      keep only current docs in `docs/`. ~45min.
- [ ] 🔥 **Consolidate to one dependency model** — CONTRIBUTING.md has a stale 7-layer model;
      ADR-0046 Four-Tier Model is honest. Update CONTRIBUTING.md to reference it. ~30min.
- [ ] 🔥 **Module relationship diagram in README** — 49 modules, no visual map. Add a mermaid
      dependency graph + "which modules do I need?" decision flow. ~60min.

---

## Priority 3 — The 20% that delivers 80%

### Consumer-facing

- [ ] 🔥 **Middleware ordering guide** — 30+ middlewares, no recommended order. New
      `docs/middleware-ordering.md` with rationale. Both consumers guessed. ~45min.
- [ ] **SQL `TimerStore` for `scheduling`** — Only ships `MemoryTimerStore`. Both consumers
      can't adopt `scheduling` without hand-rolling SQL. ~90min.
- [ ] **SQL `AggregateReader` for `listing`** — Only ships `InMemoryAggregateReader`. Same gap. ~60min.
- [ ] **README "sales page" rewrite** — Per AGENTS.md rule. What this does, why it exists,
      how to get started. ~90min.

### Code quality

- [ ] **Lint-clean `scheduling`** — 19 lint issues (mnd, exhaustruct, gosec, wrapcheck,
      tagliatelle, errname). Mostly mechanical constant extraction + renames. ~45min.
- [ ] **Lint-clean `scenario`** — 7 lint issues (errname, exhaustruct ×3). Mechanical. ~30min.
- [ ] **CBOR-stamp tests for gRPC + watermill** — Cross-encoding round-trip tests proving
      CBOR-stamped events survive transport. Only SSE has this coverage today. ~45min.
- [ ] **Pre-commit hooks** — `fmt.Printf` ban in prod packages, api_surface.txt regen check,
      `nix fmt --fail-on-change`. Via flake.nix. ~60min.

### Documentation

- [ ] **ADR numbering fix** — Two ADRs share number 0047 (COSE + json/v2). Renumber second
      to 0054. Document gaps 0036/0041. ~30min.
- [ ] **CONTRIBUTING.md agent safety rules** — Document concurrent-agent etiquette, debug-print
      discipline, "don't revert changes you didn't author" nuance. ~30min.
- [ ] **`DeadLetterStoreAdmin` documentation** — Document Count, ListPaged, PurgeBefore in
      AGENTS.md + SKILL.md. ~30min.
- [ ] **Per-projection lag in `WorkerState`** — Add `Lag time.Duration` field to `Status()`
      output. Currently only aggregate `LagDuration()` available. ~45min.
- [ ] **`event/batch_test.go` go mod tidy** — Pre-existing per-module testing friction. ~15min.

---

## Priority 4 — Post-v4.1 (breaking changes, new major version)

- [ ] **Deprecated API removal batch 2** — Remove 9 deprecated items: `middleware.{NewMetrics,
    CommandMetrics, EventMetrics, QueryMetrics, MetricsRecorder, Observe}`,
      `catalog.Exporter` (non-generic), `storage/sql.{NewDBHandle, NewDBHandleFromDB}`.
      Breaking → v4.1 cut. ~60min.
- [ ] **Postgres CI coverage matrix** — Add CI Postgres service or label `stack/postgres`
      experimental. ~60min.

---

## Priority 5 — Public Release Readiness (NEEDS USER APPROVAL)

> **These are irreversible. Do NOT execute without explicit user approval.**

- [ ] [BLOCKED] **License swap (PROPRIETARY → Apache-2.0)** — Hard blocker for public adoption.
      **Needs user approval (irreversible).**
- [ ] [BLOCKED] **Git history scrub for internal docs** — AGENTS.md, docs/planning/\* contain
      internal strategy. **Needs user approval (irreversible).**

---

## Future — v4.1+ Parquet Journal + DuckDB

> Design complete at `docs/research/2026-07-11_PARQUET_JOURNAL_DUCKDB_MATERIALIZATIONS.md`.
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

_All completed items have been moved to [CHANGELOG.md](CHANGELOG.md) under `[4.0.0]`._
