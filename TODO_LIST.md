# TODO List

**Updated:** 2026-08-08 (system lifecycle hardening shipped: interface extraction,
test split, 4 new tests, README lifecycle docs; metaengine Doctor aggregate
diagnostics broken by daemon incomplete commit)
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
> v4.0.0, v4.1.0, v4.3.0. API stability golden is 3807 exports. All 15 tags
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

- [ ] 🔥 **Fix `metaengine/explain.go` build break** — `aggregateCapabilities`
      is referenced at line 274 (commit `b6fc8413b`) but its definition is in
      an uncommitted working-tree change the daemon didn't finish. Blocks
      `go build ./...` workspace-wide and `nix run .#verify`.
      _(Source: `docs/status/2026-08-08_08-57_system-lifecycle-hardening.md`)_
- [ ] 🔥 **Regenerate api-stability golden** — after structural changes to
      `system/system.go` (interface extraction, field type change). Run:
      `cd cmd/api-stability && GOWORK=off go run main.go -update`.
- [ ] **Run doc-check on system README** — README edited with Go-qualified
      symbols. Run: `cd cmd/doc-check && GOWORK=off go run . ../../system/README.md`.
- [ ] **Run workspace-wide `go build -tags "goexperiment.jsonv2" ./...`** —
      after all changes, not just per-module builds.

---

## Metaengine v2 — Test Coverage Gaps

> Metaengine v2 is feature-complete: `record/` module, 9 engines, Record-aware
> folds, auto-projection, tombstone deprecation, GraphBackend cleanup, aggregate
> pushdown. 14 tags created locally.
>
> **Completed across 2 sessions (2026-08-08):** DuckDB `layoutMu` data race
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
> _(Source: `docs/status/2026-08-08_08-34_metaengine-v2-coverage-gaps-duckdb-race-fix.md`)_

- [ ] **Write DuckDB race regression test** — dedicated test spawning parallel
      `ApplyLayoutPlan` + `ExplainAggregateQuery` goroutines under `-race`. The
      fix is verified by existing tests, but a targeted regression test is
      stronger proof. _(Effort: S)_
- [ ] **Document `lookupPlan` shallow-copy semantics** — returns a struct copy
      but slice fields (column names) share the underlying array. All callers
      are read-only today; document the constraint or add deep-copy. _(Effort: S)_
- [ ] **Audit all DuckDB tests for `t.Parallel()` consistency** — only
      `TestDuckDB_ExplainAggregateQuery` has `//nolint:tparallel`. Other tests
      sharing a mutable engine instance may need it too. _(Effort: S)_
- [ ] **Refresh coverage baselines** in `scripts/check-coverage.sh` — 3 new
      concurrent tests + aggregate test split + badger soak added; baselines
      unchanged but within tolerance. _(Effort: S)_
- [ ] **Test QUIC convergence under `-parallel 4`** — 30s timeout verified 3x
      in isolation (0.03s each) but not under real CI parallel pressure.
      _(Effort: S)_

---

## Aggregate Pushdown — Follow-up Items

> 5 aggregate interfaces shipped on DuckDB (all 5), SQLite (4), Postgres (all 5).
> GROUP BY pushdown 4.4x faster, MultiAggregate 2.1x faster at 100K rows.
> TypedReader consumer methods (`GroupedCount`/`Sum`/`Min`/`Max`/`Avg`,
> `MultiAggregate`) shipped.

- [ ] 🔥 **Add PG functional tests for all 5 aggregate interfaces** — testcontainers,
      zero tests currently. DuckDB + SQLite have full coverage; PG has compile-time
      assertions only.
- [ ] **Write ADR for aggregate pushdown architecture** — documents the 5-interface
      design, cross-engine parity strategy, and why aggregation goes to the engine
      instead of Go-side accumulation.
- [x] **Extract shared `DecodeFloat` into metaengine core** — done in commit
      `a380e1ed1` (promoted to package-level `DecodeFloat`).
- [ ] **Add DuckDB planned-path empty-collection test** — currently only
      json_extract path tested for empty collections.
- [ ] **Add cross-engine planned-table parity test** — verify DuckDB + SQLite
      planned-table results match.
- [ ] **Add aggregate pushdown to `SerializablePlan`** — JSON serialize/diff/pin
      support for aggregate query plans.
- [ ] **Add aggregate diagnostics to `Doctor()`** — show pushdown vs fallback
      per collection. ⚠️ **Daemon started this but left the build broken** —
      `aggregateCapabilities` in `metaengine/explain.go` is uncommitted. Either
      finish the implementation or revert the reference.

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

### Integration (future)

- [ ] **Integration test: SQLite source-of-truth + Memory projections + HealthCheck**
      — end-to-end system with real engines.
- [ ] **Integration test: Pebble source-of-truth + HealthCheck**.
- [ ] **Integration test: GracefulClose with real Watermill router as Drainer**.

---

## cqrs-lint

> 192 rules across 10 categories. v4.4.0 tagged. Self-lint clean (0 CRITICAL,
> 0 ERROR). C001/D012/C008 false-positive fixes shipped. SARIF logicalLocations,
> A034 per-module migration, cross-format consistency tests all shipped.

- [ ] 🔥 **Run cqrs-lint against real consumer projects** — validate
      false-positive rates against Kernovia, Standup-Killer, bank-sync,
      cqrs-htmx, DiscordSync, timesheets, crush-daily, KeyHolderAI. This is the
      single highest-value non-coding task for cqrs-lint trustworthiness.
- [ ] 🔥 **C023 false positive on void-return `Close()`** — `dgo` client's
      `Close()` returns void but C023 flags it. Needs type-awareness: check call
      expression returns an error before flagging. Requires `TypesInfo`.
- [ ] **C008 word-boundary matching** — `TotalDays` matches `total`; add
      word-boundary regex to prevent substring false positives.
- [ ] **D007 auto-fix test** — `--fix` path (replaces `event.NewEvent` →
      `event.New`) is untested.
- [ ] **Generalize C001 `Begin(false)` check** — currently bbolt-specific; other
      DBs may use different read-only patterns.
- [ ] **Dedicated SARIF logicalLocations test** — verify array is populated,
      index mapping is correct, `kind` is `"module"`.
- [ ] **~80 C033 bare `return err` findings** — across `metaengine/*engine/`
      and `benchkit/`. All INFO-level. Needs bulk-fix vs suppress decision.
- [ ] **~15 D014 missing json tags** findings.
- [ ] **~8 C034 `go func()` without context** findings.
- [ ] **~6 P012/P013 SQLite without WAL/busy_timeout** findings.
- [ ] **~8 A032 string/int fields instead of branded ID** findings.
- [ ] **Deferred P-series rules** — `metaengine.Query` without type parameter,
      `MapUpdate` on replicated engine, Store never Closed, `metaengine.On`
      wrong handler signature. Each needs advanced type inference.
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
- [ ] **Cache ephemeral PG data dir** — skip `initdb` on repeated runs. (M35)
- [ ] **Performance profiling: ephemeral PG vs testcontainers** — measure
      speedup and document. (M36)
- [ ] **Explore `nixos-container` as lighter-weight VM alternative** (M37)
- [ ] **DuckDB CGo VM test** — hermetic DuckDB testing with GCC in VM. (M38)
- [ ] **SQLite WAL concurrency VM test** — concurrent access patterns. (M39)
- [ ] **Turso sync VM test** — real libSQL server. (M40)
- [ ] **Pebble backup/restore lifecycle VM test** (M42)
- [ ] **Contract test suite across ALL backends in VMs** — SQLite, PG, MySQL,
      DuckDB simultaneously. (M46)
- [ ] **Ephemeral Redis/NATS for future integration tests** — Watermill adapter
      testing with real brokers. (M47)
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
      `metaengine/bench/`, `example/metaengine-quickstart/`. Moved `idempotency/kvstore/`
      + `idempotency/sqlstore/` to Tier 2. All subgraph labels corrected.
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
