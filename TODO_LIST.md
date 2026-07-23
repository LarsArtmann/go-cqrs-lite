# TODO List

**Updated:** 2026-07-23 (docs compile tests, README rewrite, deprecated API removal batch 2)
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
- [x] ⭐ **Fix README "sales page" friction** — ✅ `example/readme-quickstart/` created
      with compile-verified `main.go` + `main_test.go`, registered in `go.work` and `flake.nix`.
- [x] ⭐ **Add pre-commit hooks** — ✅ `scripts/pre-commit.sh` + flake apps `check-printf`,
      `pre-commit`, `install-hooks`.
- [x] ⭐ **Add CBOR-stamp round-trip tests for gRPC + watermill** — ✅ Tests in
      `transport/grpc/event_test.go` and `watermill/event_publisher_test.go`. gRPC CBOR
      encoding bug fixed (`payload_encoding` metadata preservation).
- [x] ⭐ **Update SKILL.md eventtest FAQ** — ✅ Updated with correct `go get` command.
- [x] ⭐ **Achieve zero-lint across all 44 modules** — ✅ Down from 76 → 0 issues.
      All `ireturn`, `exhaustruct`, `wrapcheck`, `gocritic`, `gci` findings resolved.

---

## 🔥 4% Tier — High Impact, Slightly Larger

- [x] **Fix lint findings (was 76 issues)** — ✅ Zero-lint achieved across all 44 modules.
- [x] **`scheduling.SQLTimerStore`** — ✅ Already implemented at `storage/timer_store.go`.
- [x] **`listing.SQLAggregateReader`** — ✅ Already implemented at `storage/sql_aggregate_reader.go`.
- [x] **`projectionhost.Host.LagDuration()`** — ✅ Already implemented at `projectionhost/host.go:263`
      with `LagPerProjection()` at line 286. Tests at `host_test.go`.
- [x] **Compile-verify `docs/getting-started.md` examples** — ✅ `docs_compile_test.go`
      in `example/getting-started/` tests every API pattern from the docs. Fixed
      missing `fmt` and `context` imports in the docs.

---

## 20% Tier — Important, Can Queue

- [x] **README full "sales page" rewrite** — ✅ Restructured as 3-step Quick Start
      (define domain, event-source with decider, go to production). Added Install section,
      trimmed module catalog to 12 key modules (links to AGENTS.md for full 52).
- [x] **Deprecated API removal batch 2** — ✅ Removed: `middleware.{NewMetrics,
CommandMetrics, EventMetrics, QueryMetrics, MetricsRecorder, Observe}` (entire
      `metrics.go` deleted), `catalog.ErrorExporter`, `storage/sql.{NewOwnedDBHandle,
SetOwnership}`. Tests migrated to typed metrics API. Breaking → v4.1 cut.
- [x] **Postgres CI coverage matrix** — ✅ Already implemented at `ci.yml:380-418`.
      Postgres 16 service container, `POSTGRES_TEST_DSN` env var, `-tags=integration` tests.
- [x] **Add `docs/*/archive/README.md` files** — ✅ All 8 archive directories already
      have README.md files explaining what the archived artifacts are.

---

## Priority — Public Release Readiness

> **These are irreversible. Do NOT execute without explicit user approval.**

- [x] [DECLINED] **License swap (PROPRIETARY to Apache-2.0)** — User declined (2026-07-23).
      Keeping PROPRIETARY for now.
- [x] [DECLINED] **Git history scrub for internal docs** — User declined (2026-07-23).
      History stays as-is.
- [x] **Delete wrong remote tag `event/v4/eventtest/v4.0.0`** — Deleted from remote (2026-07-23).

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
