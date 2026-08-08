# TODO List

**Updated:** 2026-08-08 (metadata immutability sweep, query parity, README fix, lint config cleanup)
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
> **WARNING:** Verify gate NOT confirmed GREEN after the tagging changes. See
> [status report](docs/status/2026-08-08_01-34_metaengine-v2-publishability-and-test-coverage.md).

### Release hygiene (blocks consumer trust)

- [x] 🔥 **Push 14 new tags to remote** — all 14 confirmed on `origin`
      (verified via `git ls-remote --tags origin`): `metaengine/pebbleengine/v4.0.0`,
      `metaengine/bench/v4.0.0`, `retry/v4.3.0`, `middleware/v4.3.0`,
      `benchkit/v4.3.0`, `stack/v4.3.0`,
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
> **Round 2 hardening shipped:** shutdown.go extraction (system.go 350→268),
> `namedEngine` struct (parallel slices eliminated), `ProjectionHostResource`
> removed (was a lie), HealthCheck on all 4 external-state engines (SQLite,
> DuckDB, Postgres, Pebble), 10 new tests (orderedEngines, Close, Drainer,
> ResetProjection restart-and-replay). See
> [status report](docs/status/2026-08-08_01-53_system-hardening-round2.md).

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
      (Kahn's algorithm, cycle fallback to creation order). Projection host
      always closes first (removed `ProjectionHostResource` — it was a lie).
- [x] **Add `Drainer` interface** — `GracefulClose` now drains via registered
      `Drainer`s before `Close`. `RegisterDrainer(d)` method on System.
- [x] **Document `DomainConfig.CheckpointStore` in README** — added
      DomainConfig Fields table with CheckpointStore + ShutdownDependencies.
- [x] **HealthCheck on all external-state engines** — SQLite (`db.PingContext`),
      DuckDB (`db.PingContext`), Postgres (`db.PingContext`), Pebble (lightweight
      point-read). All implement `metaengine.HealthChecker`.
- [x] **Extract shutdown logic to `shutdown.go`** — `Drainer`, `shutdownEdge`,
      `orderedEngines`, `RegisterDrainer` extracted from `system.go` (was at
      350-line CI limit). `namedEngine` struct replaces parallel slices.
- [x] **Tests for shutdown ordering + drainer + close joining** —
      `TestOrderedEngines_NoDeps`, `TestOrderedEngines_BasicOrdering`,
      `TestOrderedEngines_CycleFallback`, `TestOrderedEngines_UnknownNames`,
      `TestSystem_Close_ErrorJoining`, `TestSystem_Close_OrderMatchesOrderedEngines`,
      `TestSystem_RegisterDrainer_CalledBeforeClose`,
      `TestSystem_RegisterDrainer_ErrorPropagation`,
      `TestSystem_ResetProjection_RestartAndReplay` (SQLite persistence).

### Round 2 — Open follow-up items

#### HealthCheck completeness (high priority)

- [ ] 🔥 **Add `HealthCheck` to Badger engine** — `db.View(func(txn) error {
    return nil })` as a lightweight read-only probe.
- [ ] 🔥 **Add `HealthCheck` to Dgraph engine** — gRPC connection check via
      `dgo` client.
- [ ] **Add test for Pebble `HealthCheck`** — in-memory vfs, healthy + closed
      DB (closed DB returns non-`ErrNotFound` error).
- [ ] **Add test for SQLite `HealthCheck` on closed DB** — verify error
      propagation.
- [ ] **Add test for DuckDB `HealthCheck`** — CGo required.
- [ ] **Add test for Postgres `HealthCheck`** — testcontainers or nix
      ephemeral PG.
- [ ] **Add `HealthChecker` to `Store.HealthCheck` test matrix** — verify it
      delegates to all engines, not just the first.

#### System module improvements (medium priority)

- [ ] **Add `System.Drain(ctx)` standalone method** — expose drain without
      Close (for rolling deploys).
- [ ] **Add `System.EngineNames()` introspection** — returns engine names for
      diagnostics (`Explain()` prints count but not names).
- [ ] **Add `System.ShutdownOrder()` introspection** — returns resolved close
      order for debugging shutdown hangs.
- [ ] **Consider structured `HealthCheck` result** — `HealthCheckDetailed()`
      returning `[]EngineHealth{Name, Error}` while `HealthCheck()` stays
      first-error-only (non-breaking).
- [ ] **Add `System.LagPerProjection()`** — expose projection host lag via
      System.
- [ ] **Add `System.LagDuration()`** — same.
- [ ] **Add `System.WorkerStatus()`** — expose projection host worker status.
- [ ] **Add `System.RegisterCloser(name, closer)`** — let consumers register
      external resources for lifecycle management.
- [ ] **Test `GracefulClose` with context expiring during drain phase** —
      slow drainer + short context.
- [ ] **Test `GracefulClose` with context expiring during close phase** —
      slow engine + short context.

#### Shutdown ordering (medium priority)

- [ ] **Test `orderedEngines` with 5+ engines and multiple edges** — complex
      DAG topology.
- [ ] **Test `orderedEngines` with self-loop edge** — `{before: "a", after:
    "a"}` should be silently ignored.
- [ ] **Test `orderedEngines` with duplicate edges** — should not double-count
      inDegree.
- [ ] **Add dedup guard to `orderedEngines` cycle fallback** — currently
      correct but fragile if refactored; explicit guard prevents accidental
      breakage.

#### Documentation (low priority)

- [ ] **Update `AGENTS.md` system module section** — remove
      `ProjectionHostResource` from Modules list, mention all 4 engine
      HealthChecks.
- [ ] **Add `ShutdownDependency` example to README Quick Start**.
- [ ] **Add `Drainer` example to README**.
- [ ] **Update `metaengine` README** — list which engines implement
      `HealthChecker`.

#### Testing polish (low priority)

- [ ] **Split `system_internal_test.go` if it grows past ~250 lines** —
      currently 7 tests + 3 mock types.
- [ ] **Add `TestSystem_Close_NoEngines`** — Close on a system with zero
      engines.
- [ ] **Add `TestSystem_Close_ProjectionHostError`** — projection host Stop
      fails, engine close still runs.
- [ ] **Add `TestSystem_GracefulClose_NoDrainers`** — GracefulClose with zero
      registered drainers (should just Close).
- [ ] **Add `TestSystem_GracefulClose_MultipleDrainers`** — verify all
      drainers called in order.
- [ ] **Add comment to `TestSystem_ResetProjection_RestartAndReplay`** —
      explain SQLite shared-cache DSN pattern.

#### Release (when ready)

- [ ] **Tag `metaengine/sqliteengine/v4.0.1`** (new `HealthCheck`).
- [ ] **Tag `metaengine/duckdbengine/v4.0.1`** (new `HealthCheck`).
- [ ] **Tag `metaengine/pgengine/v4.0.1`** (new `HealthCheck`).
- [ ] **Tag `metaengine/pebbleengine/v4.0.1`** (new `HealthCheck`).
- [ ] **Tag `system/v4.1.0`** (removed `ProjectionHostResource`,
      `namedEngine` refactor, new tests).
- [ ] **Verify all module versions are monotonically increasing before
      tagging** — check `git tag -l '<module>/v4*' | sort -V | tail -1`.

#### Integration (future)

- [ ] **Integration test: SQLite source-of-truth + Memory projections +
      HealthCheck** — end-to-end system with real engines.
- [ ] **Integration test: Pebble source-of-truth + HealthCheck**.
- [ ] **Integration test: GracefulClose with real Watermill router as
      Drainer**.

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

### DONE this session (test suite consolidation)

- [x] **Add `stack/bbolt/contract_test.go`** — mirrors the pebble pattern:
      `contracttest.RunSuite(t, factory)`. All 5 subtests pass.
- [x] **Add bbolt to `durability_tiers_test.go`** in stack/bench —
      `BenchmarkDurabilityTiers_Bbolt` tests all 3 tiers (Strict/Normal/Relaxed).
- [x] **Extract `commandtest`/`querytest` shared packages** —
      `command/commandtest/store_suite.go` (RunStoreSuite + 6 subtests),
      `query/querytest/store_suite.go` (RunStoreSuite + 4 subtests). Pebble +
      bbolt consumer tests refactored to thin wrappers (892 → 136 lines).
- [x] **Modernize `for` loops in `storage/bbolt/contract_test.go`** — 4 instances
      of `for i := 0; i < N; i++` → `for i := range N` (gopls rangeint hints
      cleared).

### Remaining

- [ ] **Refactor `storage/memory/` command/query store tests** — 316+248 lines
      with the same ~90% duplication pattern. Now that `commandtest`/`querytest`
      shared suites exist, memory backend should adopt them too.
- [ ] **Add `command/commandtest` to AGENTS.md module list** — missing from the
      Quick Reference Modules row (lists `query/querytest` but not
      `command/commandtest`).
- [ ] **Add `command/commandtest` to `cmd/api-stability/main.go` modules list** —
      `query/querytest` is tracked; `command/commandtest` is not. The
      `TestEveryGoModDirIsInModulesList` gate will catch this once it's a
      separate package path.
- [ ] **Add `doc.go` to `command/commandtest/`** — `query/querytest` has one;
      `commandtest` puts the package doc in `store_suite.go` header instead.
      Consistency issue.
- [ ] **Add self-test to `commandtest`** — run the suite against
      `storage/memory.MemoryCommandStore` to validate the suite itself (mirrors
      how `eventtest` is tested). Currently `[no test files]`.
- [ ] **Remove unused `time` import in `durability_tiers_test.go`** — line 123
      `var _ = time.Second` suppresses an import that is genuinely unused.
      Pre-existing, not from this session.

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

> **4 items resolved 2026-08-08** — metadata immutability sweep
> (`CustomData.WithCustom` added, `EnsureCustom` deprecated), query parity
> (`query.WithCustomMetadata` added), README updated (EnsureCustom →
> WithCustom, fixed false alias claims), `.golangci.yml` exclusion sprawl
> fixed (100% documented, 12 ireturn blocks → 1, duplicates removed). See
> [status report](docs/status/2026-08-08_02-01_metadata-immutability-query-parity-lint-cleanup.md).

- [x] **`metadata.CustomData[K]` immutability gap** — added value-receiver
      `WithCustom(key K, value string)` matching the command/query pattern.
      `EnsureCustom()` deprecated with `// Deprecated:` doc comment. 5
      immutability tests added.

- [x] **`query.WithCustomMetadata` missing** — added
      `query.WithCustomMetadata(key, value string) Option`, mirroring
      `command.WithCustomMetadata`. 2 tests added (single + accumulate).

- [x] **Stale `metadata/README.md`** — updated: methods table shows
      `WithCustom` + deprecation note, fixed false "command.Metadata IS
      CustomData" claims, added standalone-struct usage example.

- [x] **Fix `.golangci.yml` exclusion sprawl** — every exclusion now has a
      `#` rationale comment (was ~50% undocumented). Consolidated 12
      scattered `ireturn`-only blocks into 1 regex group. Removed 4
      duplicate-path/duplicate-linter entries. ~70 rules → ~45 documented.

### Open follow-up items

- [ ] **Deprecate `event.EnsureCustom()` free function** — the event
      package's mutable `EnsureCustom(&m)` + direct map write pattern
      persists. Needs `event.Metadata.WithCustom` (value-receiver) + caller
      migration (`event.WithCustom` option, `watermill/protocol.go`,
      `event/tombstone.go`). Touches signing/encryption hot paths — defer
      to a dedicated session.
- [ ] **Consider deprecating `metadata.CustomData[K]` entirely** — zero
      internal production consumers (command/query migrated to standalone
      structs; only the v3-compat alias and tests remain). Decision needed:
      keep for external consumers who may embed it, or deprecate and direct
      them to `metadata.Tracing` + own Custom map.
- [ ] **Per-module `.golangci.yml` split** — the monolithic config is now
      fully documented, but golangci-lint v2 `config-dirs` would give each
      module ownership of its own exclusions. Evaluate tradeoff: single
      source of truth vs locality.

---

## Dedup

> Clone groups driven to **0 at threshold 3** (was 65). All thresholds (7, 4, 3) reduced to 0 through shared helper extraction. `.art-dupl-baseline.json`
> baseline: 64 groups. `nix run .#check-duplication` gate enforces no-new-clones.

- [x] **Investigate threshold-2 clone groups** — investigated 2026-08-08. Findings:
  - `capitalizeFirst` / `titleCase`: extracted `benchkit.TitleCase` + `benchkit.Truncate`, eliminated dup in `cmd/cqrs-bench` (2 clone groups gone).
  - `isCBORData`: **accepted** — cross-module (bbolt vs pebble, separate go.mod), 4 lines, already documented.
  - `recordErr`: **accepted** — OTel boilerplate (100 occurrences), cross-module, 1-line calls. bbolt has local helper.
  - `startStreamSpan`: **accepted** — module-local helpers already exist. Call-site boilerplate (`span := ...; defer span.End()`).
  - Remaining t=2 clones are test boilerplate (`t.Parallel()` + `ctx := context.Background()`), cross-module isolation, and DuckDB aggregations internal patterns.
- [x] **Extract `renderTable(b, headers, rows)` helper** — generalized `renderKeyTable` into
      `renderTable` + `writeTableRow` + `writeTableSeparator` + `columnWidths`. Refactored
      `renderRulesConfig` to share the same primitives.
- [x] **`deferClose(closer)` helper** — added `metaengine.DeferClose(c Closer)`. Replaced
      `defer func() { _ = X.Close() }()` with `defer metaengine.DeferClose(X)` across 47
      production sites + 17 test sites in sqlite/pg/duckdb/pebble/badger/dgraph engines.

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
