# TODO List

**Updated:** 2026-08-08 (lint gate GREEN, verify gate 17/17 GREEN, DuckDB race fixed, CHANGELOG tagged)
**Scope:** Short- and mid-term actionable work only. Long-term vision lives in
[ROADMAP.md](ROADMAP.md). Completed work lives in [CHANGELOG.md](CHANGELOG.md)
and is **never** duplicated here.

## Legend

- `[ ]` = Open
- `[BLOCKED]` = Blocked on upstream dependency or user approval
- `🔥` = Pareto high impact (top 20% that delivers 80% of value)

---

## Release Hygiene — BLOCKED on user approval

> Verify gate is 17/17 GREEN. Lint gate is 0 issues. CHANGELOG has entries for
> v4.0.0, v4.1.0, v4.3.0. API stability golden is 3807 exports.
> **Blocker:** `event/v4.4.0` and 14 other tags need to be pushed to origin
> before vulncheck can resolve modules under GOWORK=off.

- [BLOCKED] 🔥 **Push `event/v4.4.0` to origin** — `event/metadata.go` gained
      `WithCustom` after `event/v4.3.0` was tagged. 29 dependent modules require
      v4.4.0 for GOWORK=off resolution. Tagged locally, needs
      `git push origin event/v4.4.0`.
      *(Source: `docs/status/2026-08-08_07-45_lint-cleanup-race-fix-verify-green.md`)*
- [BLOCKED] 🔥 **Push all 14 new module tags to origin** — `stack/mysql/v4.1.0`,
      `stack/postgres/v4.3.0`, `stack/bbolt/v4.1.0`, `stack/duckdb/v4.1.0`,
      `stack/pebble/v4.3.0`, `stack/turso/v4.3.0`, `stack/memory/v4.3.0`,
      `stack/sqlite/v4.3.0`, `stack/v4.3.0`, `benchkit/v4.3.0`,
      `middleware/v4.3.0`, `retry/v4.3.0`, `metaengine/bench/v4.0.0`,
      `metaengine/pebbleengine/v4.0.0`. All created by prior session, none pushed.
- [ ] **Bump `event/v4` from v4.3.0 to v4.4.0 in 29 dependent go.mod files** —
      after pushing `event/v4.4.0`, update all modules that import event/v4.
      `go get github.com/larsartmann/go-cqrs-lite/event/v4@v4.4.0` in each.
- [ ] **Re-run `nix run .#vulncheck` after event/v4.4.0 push** — currently
      blocked: `watermill/protocol.go:277` calls `m.WithCustom()` which doesn't
      exist at `event/v4.3.0`. 76/77 modules scan clean; watermill is the
      holdout.
- [ ] **Write detailed CHANGELOG entries for v4.0.0/v4.1.0/v4.3.0** — current
      entries are vague ("Stack presets gain durability tiers"). Replace with
      specific exported types/functions, file:line refs, ADR references.
      *(Effort: M)*

---

## Metaengine v2 — Test Coverage Gaps

> Metaengine v2 is feature-complete: `record/` module, 9 engines, Record-aware
> folds, auto-projection, tombstone deprecation, GraphBackend cleanup, aggregate
> pushdown. 14 tags created locally. Remaining work is edge-case coverage.

- [ ] 🔥 **Add mutex protection or document single-thread constraint on DuckDB
      engine** — `duckdbengine/engine.go` `layoutPlans` map has no
      synchronization. Discovered when parallel test subtests caused a data race
      between `ExplainAggregateQuery` (read) and `ApplyLayoutPlan` (write).
      Either add `sync.RWMutex` or document the single-thread constraint.
      *(Source: `docs/status/2026-08-08_07-45_lint-cleanup-race-fix-verify-green.md`)*
- [ ] **Split `TestTypedReader_AggregateFallback` into 3 smaller tests** —
      13 subtests give it maintidx=19 (below threshold). Split into Scalar,
      Grouped, Multi subtest groups. Then remove the `maintidx` exclusion from
      `.golangci.yml` test-file rules.
- [ ] **Add `//nolint:tparallel` with justification to DuckDB tests sharing
      engine state** — `TestDuckDB_ExplainAggregateQuery` and similar tests
      share a mutable engine. Document WHY subtests can't be parallel instead
      of fighting the linter.
- [ ] **Add record-stamp test for badgerengine** — completes all-engine parity
      (currently: Memory, SQLite, Pebble, DuckDB, PG have it; Badger, Dgraph,
      GraphAdapter do not).
- [ ] **Add AutoCRUD soak for sqliteengine + pgengine** — currently only
      Memory/Pebble/DuckDB. SQLite and PG are the most-used SQL backends.
- [ ] **Consolidate `race_on.go`/`race_off.go` into `testutil/`** — pattern is
      now duplicated in 5+ locations (benchkit, metaengine, transport/grpc,
      enginetest, metaengine soak tests). Single canonical copy eliminates drift.
- [ ] **Extract `RunRecordStampTest(t, eng)` helper in enginetest** — record-stamp
      test body is copy-pasted across 4 engine modules (pebble, sqlite, duckdb, pg).
      A shared helper eliminates ~100 lines of duplication.
- [ ] **DuckDB soak CI gating decision** — DuckDB soak takes 82-116s (vs Pebble
      0.27s, Memory 0.03s). Consider `testing.Short()` skip or nightly-only tag
      if it slows per-PR CI.
- [ ] **Document concurrency safety on Engine interface** — which engines are
      safe for concurrent use (Memory: yes via RWMutex, Pebble: yes via internal
      locking) vs single-threaded (DuckDB, SQLite — no mutex on layoutPlans).
- [ ] **Add race-detector integration test for MemoryEngine concurrent access** —
      prove RWMutex works under -race with parallel goroutines.

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
- [ ] **Extract shared `DecodeFloat` into metaengine core** — eliminate 3-way
      duplication across DuckDB/SQLite/PG aggregate paths.
- [ ] **Add DuckDB planned-path empty-collection test** — currently only
      json_extract path tested for empty collections.
- [ ] **Add cross-engine planned-table parity test** — verify DuckDB + SQLite
      planned-table results match.
- [ ] **Add aggregate pushdown to `SerializablePlan`** — JSON serialize/diff/pin
      support for aggregate query plans.
- [ ] **Add aggregate diagnostics to `Doctor()`** — show pushdown vs fallback
      per collection.

---

## System Package — Open Items

> P0/P1/P2/P3 + lifecycle hardening all shipped. HealthCheck on all 6 engines,
> Drain/EngineNames/ShutdownOrder/HealthCheckDetailed/LagPerProjection/
> LagDuration/WorkerStatus/RegisterCloser all shipped. Tagged `system/v4.0.0`.

### Lifecycle follow-up

- [ ] **Split `system_lifecycle_test.go`** — 457 lines, CI limit is 350. Split
      into lifecycle_test.go + lifecycle_drain_test.go.
- [ ] **Add `TestSystem_Close_ProjectionHostError`** — projection host Stop fails,
      engine close still runs. Needs `ProjectionHostLifecycle` interface extraction.
- [ ] **Add `TestSystem_HealthCheckDetailed_MultipleEnginesMixed`** — verify
      per-engine health results with mixed healthy/unhealthy engines.
- [ ] **Add `TestSystem_Drain_Error` / `TestSystem_Drain_ContextExpired`** —
      error paths for standalone drain.
- [ ] **Tag `system/v4.1.0`** — lifecycle methods + introspection extensions.
      Verify version monotonically increasing: `git tag -l 'system/v4*' | sort -V | tail -1`.

### Release (when ready)

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

### Documentation

- [ ] **Add `ShutdownDependency` example to README Quick Start**.
- [ ] **Add `Drainer` example to README**.
- [ ] **Add `HealthCheckDetailed` example to README**.
- [ ] **Add "Lifecycle" section to system README** — Close vs GracefulClose vs Drain.

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
      code. Remove unjustified ones. The `maintidx` test-file exclusion
      (added 2026-08-08) should be removed once TestTypedReader_AggregateFallback
      is split.
- [ ] **Extend `DeferClose` to `storage/pebble/`** (~10 sites) — currently only
      applied to metaengine engines.
- [ ] **Extend `DeferClose` to `storage/bbolt/`** (~8 sites).
- [ ] **Extend `DeferClose` to `storage/eventstore/`** (~5 sites).
- [ ] **Fix tag-release script cleanup** — `scripts/tag-release.sh` leaves
      staged deletions of `race_on_test.go`, `race_off_test.go`, and
      modifications to `AGENTS.md` + `soak_10m_test.go`. Script should restore
      ALL working tree changes, not just go.mod.

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
- [ ] **`scripts/test-integration.sh` aggregator** — auto-detect best strategy
      (ephemeral, VM, or testcontainers). (M48)

---

## Layer Enforcement

> `check-module-layers.sh` has a self-enforcing coverage guard. 79 go.mod
> files, 77+ modules in `go.work`. ADR-0046 updated to the seven-tier model.

- [ ] **Rename `FOUR-TIER-MODEL.md` → `SEVEN-TIER-MODEL.md`** — filename lies
      about content (H1 says "seven-tier" but filename says "four-tier").
- [ ] **Remove dead `EXCEPTIONS[storage]="listing"`** — listing moved to
      Layer 3, the exception is no longer needed.
- [ ] **Expand go-arch-lint to remaining modules** — only 6 modules have
      per-module go-arch-lint configs. The bash script is the enforcement
      mechanism for the rest.
- [ ] **Consider rewriting `check-module-layers.sh` as `cmd/check-layers`** —
      330 lines of bash. A Go program would be more maintainable and testable.

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
