# TODO List

**Scope:** Short- and mid-term actionable work only. Long-term vision lives in
[ROADMAP.md](ROADMAP.md). Completed work lives in [CHANGELOG.md](CHANGELOG.md)
and is **never** duplicated here.

## Legend

- `[ ]` = Open
- `[BLOCKED]` = Blocked on upstream dependency or user approval
- `🔥` = Pareto high impact (top 20% that delivers 80% of value)

---

## Release Hygiene

> Verify gate is 17/17 GREEN. Lint gate is 0 issues. CHANGELOG has entries for
> v4.0.0, v4.1.0, v4.3.0. API stability golden is 3809 exports. All 15 tags
> (`event/v4.4.0` + 14 module tags) are pushed to origin. `event/v4` is bumped
> to v4.4.0 in all 44 dependent go.mod files. Vulncheck: 76/77 modules clean;
> `stack` fails on `storage.SQLiteSetSynchronous` drift (see below).

- [ ] 🔥 **Tag `storage/v4.5.1` (or v4.6.0)** — `SQLiteSetSynchronous` was added
      to `storage/sqlite_helpers.go:124` after `storage/v4.5.0` was tagged.
      `stack/sqlopt/durability.go:40` calls it, but the published tag doesn't
      have it. This blocks `nix run .#vulncheck` for the `stack` module under
      GOWORK=off. Fix: `bash scripts/tag-release.sh storage v4.6.0 "..."` then
      bump `storage/v4` in `stack/go.mod`.
      _(Verified 2026-08-08: `git show storage/v4.5.0:storage/sqlite_helpers.go`
      has 4 SQLite funcs, current HEAD has 5.)_
- [ ] **Write detailed CHANGELOG entries for v4.0.0/v4.1.0/v4.3.0** — current
      entries are vague ("Stack presets gain durability tiers"). Replace with
      specific exported types/functions, file:line refs, ADR references.
      _(Effort: M)_

- [x] ~~Fix `metaengine/explain.go` build break~~ — RESOLVED: daemon completed
      `aggregateCapabilities` in subsequent commits (`2936e8c19`, `4d4da45d5`,
      `797d9ce45`). Workspace build passes.
- [x] ~~Regenerate api-stability golden~~ — VERIFIED: 3809 exports match golden.
      The `DecodeFloat` addition in `docs/api_surface.txt` is correct (daemon's
      commit `a380e1ed1`).
- [x] ~~Run doc-check on system README~~ — PASSES: 47 references valid across
      5 packages.
- [x] ~~Run workspace-wide build~~ — PASSES: `go build -tags "goexperiment.jsonv2" ./...`
      clean.

---

## Metaengine v2 — Test Coverage Gaps

> Metaengine v2 is feature-complete: `record/` module, 9 engines, Record-aware
> folds, auto-projection, tombstone deprecation, GraphBackend cleanup, aggregate
> pushdown. 14 tags created locally.
>
> **Completed across 3 sessions (2026-08-08):** DuckDB `layoutMu` data race
> fixed (Mutex→RWMutex, extracted `lookupPlan` helper for 5 read paths),
> `TestTypedReader_AggregateFallback` split into 3 groups (Scalar/Grouped/Multi),
> `//nolint:tparallel` added to `TestDuckDB_ExplainAggregateQuery`,
> dgraphengine record-stamp test created (graphadapter documented as graph-only),
> race-consolidation tradeoff documented (3 lean modules keep local copies),
> caller-closes-engine doc added to 3 enginetest helpers,
> `TestQuicLogConvergence` timeout increased 15s→30s,
> badgerengine AutoCRUD soak added,
> Engine interface concurrency-safety matrix documented,
> 3 MemoryEngine concurrent-access `-race` tests added.
>
> **Session 3 additions:** DuckDB race regression test
> (`TestDuckDB_RaceRegression_LayoutPlanConcurrentAccess`),
> `lookupPlan` shallow-copy semantics documented,
> DuckDB `t.Parallel()` audit (all 54 tests consistent),
> coverage baselines verified within tolerance,
> QUIC convergence verified under `-parallel 4` (3x pass).
>
> _(Source: `docs/status/2026-08-08_09-27_metaengine-v2-coverage-gaps-and-aggregate-followup.md`)_

### Session 3 follow-up — direct tests for shared helpers

- [x] **Write direct unit tests for `metaengine.DecodeFloat`** — all 7 type
      branches (nil, float64, float32, int64, int, *big.Int, []byte) + unknown
      type error case + invalid JSON []byte + large *big.Int (2^200).
      `metaengine/scan_test.go`: 11 test functions + 19-subtest table-driven
      variant. _(Effort: S)_
      _(Done 2026-08-08.)_
- [x] **Write direct unit tests for `metaengine.DecodeFloatResults`** — empty
      specs, nil raws, explicit alias keying, default `AliasOr()` keying, mixed
      driver types (int64 + *big.Int + []byte + float64), error propagation
      (errPrefix + alias in message), invalid []byte error.
      `metaengine/scan_test.go`: 9 test functions. _(Effort: S)_
      _(Done 2026-08-08.)_
- [x] **Add Doctor() test asserting aggregate-pushdown section** —
      `metaengine/doctor_aggregate_test.go`: `fakeAggregateEngine` implementing
      all 5 pushdown interfaces asserts
      `pushdown: scalar, grouped, multi, multi-grouped, distinct` line appears.
      Also tests Memory engine → `none`. _(Effort: S)_
      _(Done 2026-08-08.)_
- [x] **Strengthen PG aggregate test assertions** —
      `TestPostgres_ExplainAggregateQuery` now asserts `SUM` keyword + `$1`
      placeholder + first arg is collection name.
      `TestPostgres_DistinctValues` now verifies actual values `"open"` and
      `"closed"` (not just count==2). _(Effort: S)_
      _(Done 2026-08-08.)_

- [x] **Write DuckDB race regression test** —
      `TestDuckDB_RaceRegression_LayoutPlanConcurrentAccess` in
      `metaengine/duckdbengine/race_regression_cgo_test.go`.
      30 goroutines (10 writers + 10 ExplainAggregate readers + 10 MapSet readers)
      × 50 iterations, verified under `-race`. _(Effort: S)_
- [x] **Document `lookupPlan` shallow-copy semantics** — doc comment on
      `lookupPlan` in `metaengine/duckdbengine/engine.go` explains that slice
      fields (Columns, Indexes) share the underlying array. All callers are
      read-only today. _(Effort: S)_
- [x] **Audit all DuckDB tests for `t.Parallel()` consistency** — all 54
      test functions audited. Only `TestDuckDB_ExplainAggregateQuery` needs
      `//nolint:tparallel` (subtests share mutable engine). Race regression
      test deliberately serial (no subtests). No changes needed. _(Effort: S)_
- [x] **Refresh coverage baselines** in `scripts/check-coverage.sh` —
      `nix run .#check-coverage` passes. Metaengine shifted -0.2% (81.0% vs
      80.8% actual), well within ±2.0% tolerance. No baseline updates needed.
      _(Effort: S)_
- [x] **Test QUIC convergence under `-parallel 4`** — 3x pass under
      `-parallel 4` with 30s timeout (0.03s each). Verified in
      `metaengine/irohengine/quic`. _(Effort: S)_

---

## Aggregate Pushdown — Follow-up Items

> 5 aggregate interfaces shipped on DuckDB (all 5), SQLite (4), Postgres (all 5).
> GROUP BY pushdown 4.4x faster, MultiAggregate 2.1x faster at 100K rows.
> TypedReader consumer methods (`GroupedCount`/`Sum`/`Min`/`Max`/`Avg`,
> `MultiAggregate`) shipped.

- [x] 🔥 **Add PG functional tests for all 5 aggregate interfaces** —
      7 test functions in `metaengine/pgengine/aggregations_test.go`: Count,
      Sum/Min/Max/Avg, GroupedAggregate, MultiAggregate, MultiGroupedAggregate,
      DistinctValues, EmptyCollection, ExplainAggregateQuery. All pass via
      testcontainers. _(Effort: M)_
- [x] **Write ADR for aggregate pushdown architecture** —
      [ADR-0120](docs/adr/0120-aggregate-pushdown-architecture.md): documents the
      5-interface design, cross-engine parity strategy, DecodeFloat extraction,
      and why aggregation goes to the engine.
- [x] **Extract shared `DecodeFloat` into metaengine core** — done in commit
      `a380e1ed1` (promoted to package-level `DecodeFloat`).
- [x] **Add DuckDB planned-path empty-collection test** —
      `TestDuckDB_Aggregate_EmptyPlannedCollection` tests all 5 interfaces on
      an empty planned table (COUNT, SUM, GroupedAggregate, MultiAggregate,
      MultiGroupedAggregate, DistinctValues).
- [x] **Add cross-engine planned-table parity test** —
      `TestAggregateParity_PlannedTable_DuckDB_vs_SQLite` in
      `metaengine/bench/aggregate_parity_cgo_test.go` verifies DuckDB + SQLite
      produce identical results on planned tables (Count, Sum, Min, Max, Avg,
      GroupedCount, GroupedSum).
- [x] **Add aggregate pushdown to `SerializablePlan`** — `ReadPattern` field
      added to `SerializableQuery`; `QueryChange` now detects read-pattern
      changes in `PlanDiff`. `ReadPattern` populated from `QueryAssignment`
      during serialization.
- [x] **Add aggregate diagnostics to `Doctor()`** — new
      `--- Aggregate Pushdown ---` section in `Doctor()` output shows
      per-collection pushdown capabilities (scalar/grouped/multi/distinct) via
      `aggregateCapabilities` helper.

---

## System Package — Open Items

> ✅ P0/P1/P2/P3 + lifecycle hardening ALL SHIPPED. HealthCheck on all 6
> engines, Drain/EngineNames/ShutdownOrder/HealthCheckDetailed/LagPerProjection/
> LagDuration/WorkerStatus/RegisterCloser all shipped. Tagged `system/v4.0.0`.
>
> **Lifecycle hardening completed (2026-08-08):** `projectionHostLifecycle`
> interface extracted (enables mock projection host in tests). Test file split:
> `system_lifecycle_test.go` (461 lines) → `lifecycle_test.go` (420) +
> `lifecycle_drain_test.go` (219). 4 new tests: `Close_ProjectionHostError`,
> `HealthCheckDetailed_MultipleEnginesMixed`, `Drain_Error`, `Drain_ContextExpired`.
> README: Lifecycle section (Close vs GracefulClose vs Drain table) +
> HealthCheckDetailed example. ShutdownDependency + Drainer examples already
> existed in README.
>
> _(Source: `docs/status/2026-08-08_08-57_system-lifecycle-hardening.md`)_

### Lifecycle follow-up ✅

- [x] **Split `system_lifecycle_test.go`** — done: split into
      `lifecycle_test.go` (420 lines) + `lifecycle_drain_test.go` (219 lines).
- [x] **Add `TestSystem_Close_ProjectionHostError`** — done: projection host
      Stop fails, engine close still runs. `projectionHostLifecycle` interface
      extracted.
- [x] **Add `TestSystem_HealthCheckDetailed_MultipleEnginesMixed`** — done:
      verifies per-engine results with healthy + unhealthy + non-HC engines.
- [x] **Add `TestSystem_Drain_Error` / `TestSystem_Drain_ContextExpired`** —
      done: both error paths tested.

### Release (when ready)

- [ ] 🔥 **Tag `system/v4.1.0`** — lifecycle methods + introspection extensions.
      Verify version monotonically increasing: `git tag -l 'system/v4*' | sort -V | tail -1`.
      **Depends on:** engine tags below being tagged first (consumers resolving
      `system/v4.1.0` will pull engine modules that must be at compatible versions).
- [ ] **Tag `metaengine/sqliteengine/v4.0.1`** (new `HealthCheck`).
- [ ] **Tag `metaengine/duckdbengine/v4.0.1`** (new `HealthCheck` + aggregates).
- [ ] **Tag `metaengine/pgengine/v4.0.1`** (new `HealthCheck` + aggregates).
- [ ] **Tag `metaengine/pebbleengine/v4.0.1`** (new `HealthCheck`).
- [ ] **Tag `metaengine/badgerengine/v4.0.1`** (new `HealthCheck`).
- [ ] **Tag `metaengine/dgraphengine/v4.0.1`** (new `HealthCheck`).
- [ ] **Tag `command/v4.4.0`** — includes `commandtest` subpackage (blocks
      GOWORK=off tests).
- [ ] **Tag `storage/memory/v4.3.0`** — includes `limit=0` fix + duplicate
      detection fix.

### Documentation ✅

- [x] **Add `ShutdownDependency` example to README** — already existed at
      `system/README.md:202-212`.
- [x] **Add `Drainer` example to README** — already existed at
      `system/README.md:214-233`.
- [x] **Add `HealthCheckDetailed` example to README** — added in the Lifecycle
      section with code example.
- [x] **Add "Lifecycle" section to system README** — done: Close vs GracefulClose
      vs Drain comparison table + shutdown order documentation.

### Integration ✅

- [x] **Integration test: SQLite source-of-truth + Memory projections + HealthCheck**
      — `TestIntegration_SQLiteSource_MemoryProjection_HealthCheck`: two-engine
      deployment, full CQRS roundtrip, projection catch-up, HealthCheck +
      HealthCheckDetailed + EngineNames + GracefulClose.
- [x] **Integration test: Pebble source-of-truth + HealthCheck** —
      `TestIntegration_PebbleSource_HealthCheck`: Pebble driver registered via
      `init()`, command dispatch, event persistence verification, HealthCheck +
      HealthCheckDetailed + Close.
- [x] **Integration test: GracefulClose with real Watermill router as Drainer** —
      `TestIntegration_GracefulClose_WatermillDrainer`: real Watermill EventBus
      wrapped as `system.Drainer`, event pub/sub verified, GracefulClose drains
      before closing, post-close state verified.
      _(File: `system/integration_lifecycle_test.go`, 366 lines, 3 tests, all pass
      with `-race`.)_

---

## cqrs-lint

- [ ] **~14 remaining Pareto backlog items** — see the
      [Pareto plan](docs/planning/2026-07-30_21-16_CQRS-LINT-IMPROVEMENT-BACKLOG-PARETO-PLAN.md).
- [ ] **Tag cqrs-lint v4.5.0** — with all false-positive fixes + regression
      tests. Or wait for C023 fix + consumer validation.

---

## Code Quality

- [ ] **Add `// Deprecated:` to `event.CustomData` v3-compat alias** —
      `event/v3_compat_aliases.go:31` re-exports `metadata.CustomData[K]`
      but the alias does not carry the deprecation notice.
- [ ] **Migrate remaining test callers off deprecated `EnsureCustom`** —
      `event/customdata_test.go:177,190` and `metadata/metadata_test.go:252,267`
      still call `CustomData[K].EnsureCustom()`. Migrate to `WithCustom` or
      keep as backward-compat coverage.
- [ ] **Per-module `.golangci.yml` split** — the monolithic config is now
      fully documented, but golangci-lint v2 `config-dirs` would give each
      module ownership of its own exclusions.
- [ ] **Review and tighten `.golangci.yml` exclusion blocks** — 30+ blocks
      exist. Each should have a comment explaining why it can't be fixed in
      code. Remove unjustified ones. The `maintidx` test-file exclusion is now
      safe to remove — `TestTypedReader_AggregateFallback` was split into 3
      smaller groups (Scalar/Grouped/Multi) on 2026-08-08. Verify with
      `nix run .#lint` after removal.
- [ ] **Extend `DeferClose` to `storage/pebble/`** (~10 sites) — currently only
      applied to metaengine engines.
- [ ] **Extend `DeferClose` to `storage/bbolt/`** (~8 sites).
- [ ] **Extend `DeferClose` to `storage/eventstore/`** (~5 sites).
- [ ] **Fix tag-release script cleanup** — `scripts/tag-release.sh` leaves
      staged deletions of `race_on_test.go`, `race_off_test.go`, and
      modifications to `AGENTS.md` + `soak_10m_test.go`. Script should restore
      ALL working tree changes, not just go.mod. The auto-commit daemon can
      also silently drop uncommitted edits during multi-step refactors —
      commit immediately after each logical step to avoid this.
      _(Source: `docs/status/2026-08-08_07-45_metaengine-v2-release-hygiene.md`)_

---

## CI / Release / Infrastructure

- [BLOCKED] **Publish go-finding + go-must as tagged modules** — the go.mod
  replace directives are needed for dev; consumers resolving the published
  modules depend on the real tagged versions (go-finding v1.4.1, go-must v0.1.2).

- [ ] **Pin GitHub Actions to commit SHAs** — 72+ unpinned actions
      (supply-chain risk).
- [ ] **Add self-lint to CI** — `cqrs-lint --self-lint` works but no GitHub
      Actions step gates it.
- [ ] **Add `--fail-on-stale-suppressions` CI gate** — prevents stale
      `//cqrs-lint:ignore` directives from accumulating.
- [ ] **Add CI check for API-version drift** — verify every exported symbol
      in a tagged module exists at that tag. Catches the `WithCustom`/
      `event/v4.3.0` class of drift before vulncheck fails.

---

## Integration Test Infrastructure

> Nix-based integration test infrastructure shipped: ephemeral PG, NixOS VM
> tests (PG+MySQL), nspawn MySQL (~15s), projectionhost PG crash-restart,
> scheduling/sqlstore durable timers. All in ephemeral-pg.sh.

- [ ] **macOS verification of ephemeral PG** — script claims cross-platform but
      never tested on Darwin. (M34)
- [x] **Cache ephemeral PG data dir** — skip `initdb` on repeated runs. (M35)
      `PGDATA_CACHE` env var in `ephemeral-pg.sh`.
- [x] **Performance profiling: ephemeral PG vs testcontainers** — measured
      12x speedup (1.2s vs 14.5s). Script: `scripts/profile-pg-strategies.sh`. (M36)
- [x] **Explore `nixos-container` as lighter-weight VM alternative** (M37)
      Conclusion: project already uses `runNixOSTest` with `containers.machine` (nspawn),
      which is the lighter-weight VM alternative. `nixos-container` CLI is NixOS-host-only
      and not applicable to ephemeral test infrastructure.
- [x] **DuckDB CGo VM test** — NixOS VM test verifies columnar ops, JSON extraction,
      aggregation. `nix build .#checks.x86_64-linux.duckdb-vm`. (M38)
- [x] **SQLite WAL concurrency VM test** — `storage/sqlite_wal_concurrency_test.go`:
      concurrent read/write, snapshot isolation, busy_timeout retry. (M39)
- [x] **Turso sync VM test** — NixOS VM test with `sqld` (libSQL server), CRUD via
      v2/pipeline API. `nix build .#checks.x86_64-linux.turso-vm`. (M40)
- [x] **Pebble backup/restore lifecycle VM test** — `storage/pebble/backup_lifecycle_test.go`:
      full lifecycle (events+snapshots+checkpoints), incremental backups. (M42)
- [x] **Contract test suite across ALL backends** — `scripts/test-all-backends.sh`:
      SQLite, Pebble, bbolt, DuckDB, PG, MySQL in one command.
      Flake app: `nix run .#test-all-backends`. (M46)
- [x] **Ephemeral Redis/NATS for integration tests** — `scripts/ephemeral-redis.sh`,
      `scripts/ephemeral-nats.sh`. Flake apps: `nix run .#ephemeral-redis`,
      `nix run .#ephemeral-nats`. (M47)
- [x] **`scripts/test-integration.sh` aggregator** — auto-detect best strategy
      (ephemeral, VM, or testcontainers). (M48)

---

## Layer Enforcement

> `check-module-layers.sh` has a self-enforcing coverage guard. 79 go.mod
> files, 78 modules in `go.work`. Model doc renamed to `SEVEN-TIER-MODEL.md`
> with accurate counts (78 modules). Dead `EXCEPTIONS[storage]="listing"`
> removed (storage L5 → listing L3 is a downward dep, no violation).
> ✅ **Split-brain resolved: metaengine is Tier 3** — all references reconciled (2026-08-08).
>
> _(Source: `docs/status/2026-08-08_08-23_layer-enforcement-cleanup-status.md`)_

### ✅ Split-brain fixes (resolved 2026-08-08)

- [x] **Metaengine tier: Tier 3 (confirmed).** Script updated `LAYER[metaengine]=3`.
      All references reconciled: AGENTS.md module graph + metaengine description,
      ADR-0046 body (tier table, mermaid, key insights, amendment). `record/` is a
      direct dep (verified in `metaengine/go.mod`), so Tier 3→0 downward dep passes.
- [x] **"44 of 78" → "48 of 78"** fixed in `SEVEN-TIER-MODEL.md` + `AGENTS.md`.
- [x] **ADR-0046 stale counts fixed** — all module counts updated: 68→78, 69→79,
      44 of 68→48 of 78. Tier table: T0=8, T1=5, T2=7, T3=6, T4=27, T5=10, T6=15.
- [x] **ADR-0046 mermaid diagram updated** — added `record/`, `storage/bbolt/`,
      `metaengine/sqliteengine/`, `metaengine/badgerengine/`, `metaengine/dgraphengine/`,
      `metaengine/graphadapter/`, `testutil/pgtestcontainer/`, `stack/bbolt/`,
      `metaengine/bench/`, `example/metaengine-quickstart/`. Moved `idempotency/kvstore/` + `idempotency/sqlstore/` to Tier 2. All subgraph labels corrected.
- [x] **`nix fmt`** run — 0 files changed (already formatted).
- [x] **Full `check-arch.sh`** — both layers pass (Layer 1: 78 modules, Layer 2: 6 modules).

### Backlog

- [ ] **Audit remaining 10 EXCEPTIONS entries for dead rules** — only
      `EXCEPTIONS[storage]` was checked and removed. The other 10 entries
      (event, schema, snapshot, decider, query, command, listing,
      projectionhost, transport/http, metaengine) were not verified.
- [ ] **Add go-arch-lint config for `cmd/cqrs-lint`** — it has 16 production
      sub-packages (pkg/analyzer, pkg/rules, etc.) but no intra-module
      architecture config. The other 72 modules are single-package or have
      only test sub-packages — the bash script already covers cross-module
      rules for all 78 modules. Only `storage/` and `catalog/` have
      meaningful multi-package structure with existing configs.
- [ ] **Consider rewriting `check-module-layers.sh` as `cmd/check-layers`** —
      348 lines of bash with self-enforcing coverage guard. A Go program
      would add testability but the script is stable (layer assignments
      change rarely) and only runs in CI. Defer until the script grows
      significantly more complex.

---

## Declined / Rejected (do not re-litigate)

> Kept here so decisions are not re-litigated. Full rationale in the linked
> ADRs/reviews.

- **Wire `#verify-parallel` into CI** — declined 2026-07-29. CI already has a
  per-module matrix strategy that provides better isolation.
- **Add `#verify-fast` as a pre-merge CI gate** — done (already wired as
  `verify-fast-gate` at ci.yml:128).
- **Composite keys in `SQLViewStore`** — breaks `K fmt.Stringer`. Use
  `RelationalProjection` (junction tables). See ADR-0033.
- **OR conditions / query builder in ViewStore** — `RawWhere` covers the 5% case.
- **Redis adapter** — the author is not a fan of Redis. See ROADMAP Non-Goals.
