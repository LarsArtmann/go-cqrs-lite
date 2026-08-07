# TODO List

**Updated:** 2026-08-08 (metaengine v2 test coverage + 14 module tags created)
**Scope:** Short- and mid-term actionable work only. Long-term vision lives in
[ROADMAP.md](ROADMAP.md). Completed work lives in [CHANGELOG.md](CHANGELOG.md)
and is **never** duplicated here.

## Legend

- `[ ]` = Open
- `[BLOCKED]` = Blocked on upstream dependency or user approval
- `🔥` = Pareto high impact (top 20% that delivers 80% of value)

---

## Metaengine v2 (Record-aware ES-native architecture)

> Metaengine v2 is **feature-complete and publishable**: `record/` module, 9
> engines (Memory, SQLite, Pebble, DuckDB, Postgres, Badger, Dgraph,
> GraphAdapter, Iroh), `OnRecord`/`ApplyRecord` Record-aware folds,
> `AutoInsert`/`AutoCRUD`/`AutoCRUDByConvention` auto-projection, tombstone
> deprecation. ADRs 0111-0119 written. Tags created: sqliteengine,
> graphadapter, dgraphengine, storage/bbolt, idempotency, pebbleengine,
> metaengine/bench + 11 drifted-module bumps (retry, middleware, benchkit,
> stack/*). Remaining work is release hygiene and edge-case coverage.
>
> **WARNING:** 14 tags created this session are LOCAL ONLY (not pushed). Verify
> gate NOT confirmed GREEN after changes. See
> [status report](docs/status/2026-08-08_01-34_metaengine-v2-publishability-and-test-coverage.md).

### Release hygiene (blocks consumer trust)

- [ ] 🔥 **Push 14 new tags to remote** — `git push origin --tags` or selective.
      Tags: `metaengine/pebbleengine/v4.0.0`, `metaengine/bench/v4.0.0`,
      `retry/v4.3.0`, `middleware/v4.3.0`, `benchkit/v4.3.0`, `stack/v4.3.0`,
      `stack/{memory,sqlite,turso,pebble,postgres}/v4.3.0`,
      `stack/{duckdb,bbolt,mysql}/v4.1.0`.
- [ ] 🔥 **Update CHANGELOG.md for all 14 new tags** — `TestTagContentMatchesChangelog`
      will fail without entries for each version section.
- [ ] 🔥 **Run `nix run .#verify` to completion** — verify gate was killed twice
      this session without confirming GREEN. 10 pre-existing lint findings in
      `system/` (wsl_v5, goconst, err113, prealloc) need fixing.
- [ ] **Run `nix run .#vulncheck`** — verify all tagged modules build under
      GOWORK=off (per-module consumer resolution).

### Test coverage (gaps remaining)

- [ ] **Run new concurrent tests under `-race`** — `RunConcurrentTxTest` was
      tested without the race detector. The whole point of concurrent tests is
      catching data races. Run 3x with `-count=3 -race`.
- [ ] **Record-aware integration test through DuckDB engine** — Pebble + SQLite
      done. DuckDB (CGo path) not yet.
- [ ] **Record-aware integration test through PG engine** — Pebble + SQLite
      done. PG not yet.
- [ ] **`RunTransactionalTest` on Memory engine (baseline)** — no engine module
      calls RunTransactionalTest against the Memory engine for baseline parity.
- [ ] **Soak test with `AutoCRUDByConvention` through Pebble/DuckDB** — current
      soak uses Memory engine only. Verify LSM/OLAP backends under sustained load.

> **Done this session:** Record-aware Pebble test, AutoCRUDByConvention soak
> (45K events, 0.1MB heap growth), RunTransactionalTest on SQLite, concurrent
> RunInTx test (enginetest + SQLite/DuckDB/PG), MultiAdd + LogAppend
> transactional subtests. badgerengine does NOT implement Transactional (no
> RunInTx method), so it is correctly excluded.

### Module health (DONE this session)

- [x] **Add `metaengine/keycodec`, `metaengine/enginetest`,
      `testutil/pgtestcontainer`, `example/metaengine-quickstart` to AGENTS.md** —
      added to Quick Reference table, Test command, and Monorepo Structure tree.
      (api-stability `TestEveryGoModDirIsInModulesList` was already passing —
      keycodec/enginetest are packages within metaengine, not separate go.mod
      modules.)
- [x] **Fix COVERAGE GAPs in `check-module-layers.sh`** — script passes clean.
      No gaps remain (verified by running the script directly).

> Long-term metaengine work (`metaengine-gen` code generator, Vector/Search/
> Spatial engine backends, generic `ScanResult[T]`, operator-driven engine
> selection) lives in [ROADMAP.md](ROADMAP.md).

---

## System Package (EXPERIMENTAL — P0/P1/P2/P3 shipped)

> The `system/` module implements the operator-configured CQRS topology.
> Driver registry wired, SQLite working through `New()`, projections E2E
> proven, MultiBus/SnapshotBackend/scream store wired, introspection real.
> koanf YAML config, DuckDB/PG Transactional, bus driver registry, scream store
> plan-drift detection, CommandAdapter/QueryAdapter serialization, and
> example/taskmanager migration all shipped. Tagged `system/v4.0.0`. **P2
> hardening + P2 test depth + P3 code quality all shipped.**
> HealthCheck, GracefulClose (with Drainer phase), ResetProjection,
> configurable checkpoint store, WithShutdownDependency, errors.Join in Close,
> SQLite engine HealthCheck, README Quick Start fix, doc-check arg validation.

### P2 — Hardening (DONE — see [CHANGELOG.md](CHANGELOG.md))

- [x] **`system/README.md` Quick Start doesn't compile** — replaced with
      complete `package main` program using real API. Verified compiles + runs.
- [x] **Fix `cmd/doc-check` cmdguard arg-parsing** — `cobra.ArbitraryArgs`
      replaced with custom `fileArgs` validator (rejects non-existent files,
      directories, non-`.md` extensions).
- [x] **Add `system.HealthCheck(ctx)` method** — checks stopped state, pings
      engines implementing `metaengine.HealthChecker`, inspects projection host
      for `WorkerFailed` workers.
- [x] **Add `system.GracefulClose(ctx)`** — runs `Close()` in goroutine racing
      against `ctx.Done()`.
- [x] **Add `system.ResetProjection(ctx, name)`** — delegates to
      `projectionhost.Host.Reset()`. Returns `ErrNoProjectionHost` if not
      configured.
- [x] **Wire checkpoint store as configurable** — `DomainConfig.CheckpointStore`
      field; constructor uses it instead of hardcoded `memoryCheckpointStore`.

### P2 — Test depth (DONE)

- [x] **Deepen `TestSystem_CustomCheckpointStore`** — declares a real projection,
      produces events, starts host, waits for processing, asserts `saveCnt > 0`.
- [x] **Add `TestSystem_HealthCheck_FailedProjection`** — registers a projection
      with a failing decoder, `WithMaxRestarts(1)`, waits for `WorkerFailed`,
      asserts `HealthCheck` returns an error.
- [x] **Add `TestSystem_ResetProjection_Positive`** — configures projection,
      produces events, stops host, resets, verifies checkpoint is zero-value.
- [x] **Add `TestSystem_GracefulClose_SlowShutdown`** — creates system with
      projection host, produces events, GracefulClose completes within context.
- [x] **Add `TestSystem_HealthCheck_EngineUnhealthy`** — internal test injects
      a mock engine returning error from HealthCheck, verifies propagation.

### P3 — Code quality (DONE)

- [x] **Join errors in `System.Close()`** — uses `errors.Join` for all close
      errors (projection host + engines), matching `stack.Bundle.Close()`.
- [x] **Remove dead `s.closers` slice** — removed the `closers []func() error`
      field and its loop in Close (never populated in `New()`).
- [x] **Port `WithShutdownDependency`** — `DomainConfig.ShutdownDependencies`
      field + `ShutdownDependency` type + `orderedEngines()` topological sort
      (Kahn's algorithm, cycle fallback to creation order).
- [x] **Add `Drainer` interface** — `GracefulClose` now drains via registered
      `Drainer`s before `Close`. `RegisterDrainer(d)` method on System.
- [x] **Document `DomainConfig.CheckpointStore` in README** — added
      DomainConfig Fields table with CheckpointStore + ShutdownDependencies.
- [x] **SQLite integration test for HealthCheck** — added `HealthCheck` to
      SQLite engine (`db.PingContext`), integration test verifies ping path.

---

## bbolt Storage Backend

> Full storage backend shipped: EventStore, SnapshotStore, CheckpointStore,
> KVAdapter, CommandStore, QueryStore, Backend facade. Streaming iterators
> (`StreamingSource`/`StreamingJournal`), OTel span instrumentation, contract
> test suite (29 tests), durability tiers via `stack/bbolt`.
> `storage/bbolt/v4.0.0` tagged and pushed. CommandStore/QueryStore contract
> tests, same-stream contention test, and contention benchmark all shipped
> (42 total tests, race-clean). See
> [status report](docs/status/2026-08-08_01-17_bbolt-test-and-benchmark-coverage.md).

### DONE this session

- [x] **CommandStore contract tests** — 8 tests: Save/Load, DuplicateDetection,
      AppendBatch, AppendBatchDuplicate, LoadEmptyStream, ReadAll, ReadFrom
      (pagination), LoadFromTimestamp.
- [x] **QueryStore contract tests** — 4 tests: SaveAndLoadQueries, DuplicateDetection,
      ReadAllQueries, ReadQueriesFrom (pagination).
- [x] **Same-stream concurrency contention test** — `TestContract_SameStreamContention`:
      10 goroutines race same stream with `expectedVersion=0`, exactly 1 wins.
- [x] **Add bbolt to `stack/bench/` contention benchmark** — added to
      `BenchmarkContention_Persistent_SameStream` alongside sqlite/pebble.
- [x] **`WithBatchSize` for `AppendBatch`** — dismissed: code already writes all
      events in a single atomic bbolt tx. Splitting into sub-batches would break
      atomicity.

### Remaining

- [ ] **Add `stack/bbolt/contract_test.go`** — every other stack preset (memory,
      sqlite, pebble, postgres, mysql, turso) has one. bbolt is the only one missing.
- [ ] **Add bbolt to `durability_tiers_test.go`** in stack/bench — test all 3
      durability levels via `stack/bbolt`.
- [ ] **Extract `commandtest`/`querytest` shared packages** — pebble and bbolt
      command/query store tests are ~90% identical. Mirror the `eventtest` pattern.
- [ ] **Modernize `for` loops in `contract_test.go`** — 4 instances of
      `for i := 0; i < N; i++` → `for range N` (pre-existing gopls hints).

---

## Irohengine

> Level 2 prototype shipped with CRDT-safe operations. Three transports:
> InProcessNetwork (goroutine, no CGo), loopback (real TCP, no CGo), QUIC
> (`iroh-go` C bindings, CGo required). loopback/v4.0.0 + quic/v4.0.0 tagged.

- [ ] **QUIC transport integration with `adttest.RunMatrix`** — the in-process
      mock + loopback pass the full matrix; verify the QUIC transport also
      passes parity tests (LWW resolution, PN-Counter, MapUpdate-does-not-replicate).
- [ ] **Non-CRDT op rejection on QUIC path** — verify `MapUpdate` operations
      stay local-only and are NOT sent over QUIC (would break CRDT convergence).
- [ ] **Fix `TestQuicSetConvergence` flakiness** — pre-existing, network-dependent.
      Consider `//go:build integration` tag or relaxed timing bounds.
- [ ] **Fix `TestLoad_ConcurrentLoadsCoalescedBySingleflight` flake** —
      pre-existing race-condition test that flakes under parallel load.

---

## cqrs-lint

> 192 rules across 10 categories. Config presets, `--adoption`/`--scorecard`/
> `--group-by` flags, SARIF + Markdown output, self-lint mode, block-level
> suppression, metaengine-aware detection (F018-F026), scorecard metaengine
> section, cross-format consistency tests. v4.4.0 tagged. Self-lint clean
> (0 CRITICAL, 0 ERROR, 0 load errors, 0 stale suppressions).

- [ ] 🔥 **Run cqrs-lint against real consumer projects** — validate
      false-positive rates against Kernovia, Standup-Killer, bank-sync,
      cqrs-htmx, DiscordSync, timesheets, crush-daily, KeyHolderAI. This is the
      single highest-value non-coding task for cqrs-lint trustworthiness.

- [ ] **Fix remaining false positives** — C001 (read-only bbolt transactions),
      D012 (CLI tools should be excluded), C008 (non-monetary floats).

- [ ] **Triage remaining ~199 self-lint WARNING/INFO findings** — D007 (8
      `event.NewEvent` → `event.New`), D014 (15 missing json tags), C034 (8
      `go func()` without ctx), C033 (~15 bare `return err`), C023 (~10
      unchecked `Close()`), P012/P013 (~6 SQLite without WAL/busy_timeout),
      A032 (~8 string/int fields instead of branded ID).

- [ ] **Missing regression tests** — S006 fix (WEAK suppression), A018 fix
      (dispatch activity check), B004 fix (constructor check) — 3 of 7
      KeyHolderAI fixes have no regression tests.

- [ ] **Migrate global detectors to per-module evaluation** —
      `ProfileForFile` infrastructure exists. 13+ detectors migrated. Remaining:
      8 detector files still use `ctx.FeatureProfile` directly (6 in `adoption/`,
      1 in `api/`). F-series rules are intentionally project-level. High
      false-positive risk for multi-module workspaces.

- [ ] **Scorecard SARIF `logicalLocations`** — SARIF output represents adoption
      metrics as `notifications` (not `results`). The `logicalLocations` half
      is still pending.

- [ ] **Deferred P-series rules** — `metaengine.Query` without type parameter,
      `MapUpdate` on replicated engine, Store never Closed, `metaengine.On`
      wrong handler signature. Each needs advanced type inference.

- [ ] **L1.5 domain severity calibration** — `DomainKind` enum +
      `applyDomainBias` shipped; still needs broader testing against
      financial/security projects.

- [ ] **~14 remaining Pareto backlog items** — see the
      [Pareto plan](docs/planning/2026-07-30_21-16_CQRS-LINT-IMPROVEMENT-BACKLOG-PARETO-PLAN.md).
      Highest impact: L1.30–L1.33 deep pattern detection, L1.47–L1.51 new rule
      categories (DOC/OBS/RES/DI).

---

## Code Quality

- [ ] **`metadata.CustomData[K]` immutability gap** — `command.Metadata` and
      `query.Metadata` migrated to `WithCustom()` (functional), but
      `metadata.CustomData[K]` still has pointer-receiver `EnsureCustom()`.
      Decision needed: complete the immutability sweep or accept the exception.

- [ ] **`query.WithCustomMetadata` missing** — `command` has
      `WithCustomMetadata(key, value string) Option` but `query` does not.
      Asymmetry between the two modules.

- [ ] **Stale `metadata/README.md`** — still documents `EnsureCustom` (removed
      from command/query). Needs update to `WithCustom`.

- [ ] **Fix `.golangci.yml` exclusion sprawl** — ~30 blocks, ~50% undocumented.
      Add comments explaining WHY for every exclusion. Consider per-module
      config split.

---

## Dedup

> Clone groups driven to **0 at threshold 3** (was 65). All thresholds (7, 4, 3) reduced to 0 through shared helper extraction. `.art-dupl-baseline.json`
> baseline: 0 groups. `nix run .#check-duplication` gate enforces no-new-clones.

- [ ] **Investigate threshold-2 clone groups** — 92 remaining at t=2. Some are
      intentional (cross-module isolation, table-driven); others may be
      extractable (`capitalizeFirst`, `truncateString`, `isCBORData`,
      `recordErr`, `startStreamSpan` patterns).
- [ ] **Extract `renderTable(b, headers, rows)` helper** — `cmd/cqrs-lint/explain.go`
      still has repeated table-rendering patterns.
- [ ] **`deferClose(closer)` helper** — metaengine engines have repeated
      `if c := eng.Close(); c != nil ...` patterns across 7 engines.

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
- [x] **Fix 16 COVERAGE GAPs** — `check-module-layers.sh` passes clean. All
      newer modules are now in LAYER/DEP_BUDGET maps. (Verified 2026-08-08.)
- [ ] **Expand go-arch-lint to remaining modules** — only 6 modules have
      per-module go-arch-lint configs. The bash script is the enforcement
      mechanism for the rest.
- [ ] **Consider rewriting `check-module-layers.sh` as `cmd/check-layers`** —
      330 lines of bash. A Go program would be more maintainable and testable.

---

> **Deferred debt resolved.** Ghost bus removal (ADR-0028) and metadata
> aliases completion (ADR-0031) are both DONE. `retry/` → `go-retry` and
> `idempotency/` → `go-idempotency` extraction is DONE. `retry/` is now
> DEPRECATED (re-export shim, consumers should import `go-retry` directly).

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
