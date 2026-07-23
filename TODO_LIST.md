# TODO List

**Updated:** 2026-07-23 (docs-health + Pareto audit)
**Scope:** Short- and mid-term actionable tasks only. Long-term vision lives in [ROADMAP.md](ROADMAP.md).
Completed work lives in [CHANGELOG.md](CHANGELOG.md).

## Legend

- `[ ]` = Open
- `[x]` = Done (moved to [CHANGELOG.md](CHANGELOG.md))
- `[BLOCKED]` = Blocked on upstream dependency or user approval
- `🔥` = Pareto high impact (top 20% that delivers 80% of value)
- `⭐` = Top 1% impact (do first)

---

## ⭐ 1% Tier — Do First (Highest Impact)

These items deliver the majority of perceived consumer value.

- [x] **Publish `eventtest` to Go proxy as `v0.1.0`** — ✅ Verified available via
      `GOPROXY=proxy.golang.org go list -m ...@v0.1.0`. v0.2.0 is also published.
      Remaining: delete the wrong `event/v4/eventtest/v4.0.0` tag from **remote**
      (`git push --delete origin event/v4/eventtest/v4.0.0`). Local tag deleted.
- [ ] ⭐ **Fix README "sales page" friction** — The Quick Start example has a missing
      trailing comma (line 80) and is not compile-verified. Fix the bug, verify the
      snippet compiles, and tighten the top-of-page pitch. ~20min.
- [ ] ⭐ **Add pre-commit hooks** — `fmt.Printf` ban in production packages,
      `api_surface.txt` regeneration check, `nix fmt --fail-on-change`. Prevents
      avoidable CI failures and debug prints leaking to main. ~60min.
- [ ] ⭐ **Add CBOR-stamp round-trip tests for gRPC + watermill** — Cross-encoding
      tests proving CBOR-stamped events survive transport. Only SSE has this today. ~45min.
- [ ] ⭐ **Update SKILL.md eventtest FAQ** — The FAQ still says eventtest has "no
      published tag"; it is now published. Replace with the `go get` command. ~10min.

---

## 🔥 4% Tier — High Impact, Slightly Larger

- [ ] **Fix lint findings (76 issues)** — Mostly `ireturn` in `cmd/cqrs-lint` detector
      constructors, plus a few `tagliatelle`/`modernize`/`revive`. Clean or explicitly
      suppress with justification. ~60min.
- [ ] **Add `scheduling.SQLTimerStore`** — Persistent deadline timers backed by `*sql.DB`.
      Both major consumer feedback rounds asked for scheduling adoption; SQLite-only
      memory store blocks production use. ~90min.
- [ ] **Add `listing.SQLAggregateReader`** — SQL-backed aggregate listing reader so
      listing works on real databases, not just in-memory. ~90min.
- [ ] **Add `projectionhost.Host.LagDuration()`** — Built-in lag metric requested by
      DiscordSync. Enables a single Prometheus gauge. ~20min.
- [ ] **Compile-verify `docs/getting-started.md` examples** — Ensure every snippet
      builds; add a CI step or test file to catch drift. ~30min.

---

## 20% Tier — Important, Can Queue

- [ ] **README full "sales page" rewrite** — Per docs-health model, README should be
      the end-user entry point: what this does, why it exists, how to get started in 3 steps.
      ~90min.
- [ ] **Deprecated API removal batch 2** — Remove 9 deprecated items: `middleware.{NewMetrics,
CommandMetrics, EventMetrics, QueryMetrics, MetricsRecorder, Observe}`,
      `catalog.Exporter` (non-generic), `storage/sql.{NewDBHandle, NewDBHandleFromDB}`.
      Breaking → v4.1 cut. ~60min.
- [ ] **Postgres CI coverage matrix** — Add CI Postgres service or label `stack/postgres`
      experimental. ~60min.
- [ ] **Add `docs/*/archive/README.md` files** — Explain what archived historical
      artifacts are. Currently empty directories without guidance. ~20min.

---

## Priority — Public Release Readiness (NEEDS USER APPROVAL)

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

---

_All completed items have been moved to [CHANGELOG.md](CHANGELOG.md) under `[4.0.0]` or `[Unreleased]`._
