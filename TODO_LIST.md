# TODO List

**Updated:** 2026-08-08 (system lifecycle hardening, aggregate pushdown, GraphBackend cleanup, dedup helpers, metadata deprecation, CBOR bugfix)
**Scope:** Short- and mid-term actionable work only. Long-term vision lives in
[ROADMAP.md](ROADMAP.md). Completed work lives in [CHANGELOG.md](CHANGELOG.md)
and is **never** duplicated here.

## Legend

- `[ ]` = Open
- `[BLOCKED]` = Blocked on upstream dependency or user approval
- `🔥` = Pareto high impact (top 20% that delivers 80% of value)

---

## Metaengine v2 — Release Hygiene

> Metaengine v2 is **feature-complete**: `record/` module, 9 engines, Record-aware
> folds, auto-projection, tombstone deprecation, GraphBackend cleanup, aggregate
> pushdown. 14 tags created and pushed to `origin`. Remaining work is release
> documentation and edge-case coverage.

- [ ] 🔥 **Update CHANGELOG.md for all 14 new tags** — `TestTagContentMatchesChangelog`
      will fail without entries for each version section.
- [ ] 🔥 **Run `nix run .#verify` to completion** — verify gate was killed multiple
      times across sessions without confirming GREEN. Must confirm clean before
      tagging or claiming release readiness.
- [ ] **Run `nix run .#vulncheck`** — verify all tagged modules build under
      GOWORK=off (per-module consumer resolution).
- [ ] **Regen API stability golden** — `RunTransactionalBaselineTest`,
      `RunAutoCRUDSoak`, `DeferClose`, aggregate interfaces, and system lifecycle
      methods are new exports. Run
      `cd cmd/api-stability && GOWORK=off go run main.go -update`.

### Test coverage gaps

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
- [ ] **DuckDB soak CI gating decision** — DuckDB soak takes 82-98s (vs Pebble
      0.27s, Memory 0.03s). Consider `testing.Short()` skip or nightly-only tag
      if it slows per-PR CI.
- [ ] **Add `// Caller owns engine Close.` doc comment to
      `RunTransactionalBaselineTest`** — matching the convention of
      `RunTransactionalTest` and `RunAutoCRUDSoak`.

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
- [ ] **Add `art-dupl:accept` to `duckdbengine/explain.go` and
      `sqliteengine/explain.go`** — cross-module SQL builders accepted as
      intentional.
- [ ] **Add DuckDB planned-path empty-collection test** — currently only
      json_extract path tested for empty collections.
- [ ] **Add cross-engine planned-table parity test** — verify DuckDB + SQLite
      planned-table results match.
- [ ] **Update FEATURES.md** — aggregate pushdown capabilities (DONE this session).
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
- [ ] **Extend `DeferClose` to `storage/pebble/`** (~10 sites) — currently only
      applied to metaengine engines.
- [ ] **Extend `DeferClose` to `storage/bbolt/`** (~8 sites).
- [ ] **Extend `DeferClose` to `storage/eventstore/`** (~5 sites).

---

## Pre-Existing Failures

> CBOR encoding bugfix shipped (2026-08-08): `event.New` WithEncoding fix +
> Watermill CBOR test fixes. 81 modules test GREEN.

- [ ] 🔥 **Fix `cmd/api-stability/main.go:172` — `collectExports` undefined** —
      the api-stability tool itself does not compile. Blocks ALL api-surface
      golden regeneration. A meta-test should ensure `cmd/api-stability`
      compiles (catches this class of breakage).
- [ ] **Regenerate api-stability golden** — after fixing the tool. The golden is
      stale: missing `event.Metadata.WithCustom`, `DeferClose`, aggregate
      interfaces, system lifecycle methods, and likely other symbols.

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
